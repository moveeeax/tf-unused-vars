// Package analyzer walks a set of Terraform (.tf) files, resolves every
// var./local. reference, and reports declarations that are never used.
package analyzer

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

// Kind is the sort of declaration a finding refers to.
type Kind string

const (
	KindVariable Kind = "variable"
	KindLocal    Kind = "local"
	KindOutput   Kind = "output"
)

// Finding is a single declared-but-unused declaration.
type Finding struct {
	Kind Kind   `json:"kind"`
	Name string `json:"name"`
	File string `json:"file"`
	Line int    `json:"line"`
}

// Options tunes what the analyzer reports.
type Options struct {
	// StrictOutputs additionally flags outputs that nothing in the module
	// references. It is opt-in because a leaf module's outputs are normally
	// consumed by a caller, so they look unused from inside the module.
	StrictOutputs bool
}

// ignoreMarker waives a declaration from being reported. Place it as a trailing
// or preceding comment on the declaration line.
const ignoreMarker = "tf-unused-vars:ignore"

type decl struct {
	kind Kind
	name string
	file string
	line int
}

func (d decl) finding() Finding {
	return Finding{Kind: d.kind, Name: d.name, File: d.file, Line: d.line}
}

// Analyze parses every .tf file reachable from paths (files are read directly,
// directories are walked recursively) and returns the unused declarations,
// sorted by file then line. A parse error for any file is returned as-is.
func Analyze(paths []string, opts Options) ([]Finding, error) {
	files, err := collectFiles(paths)
	if err != nil {
		return nil, err
	}

	var decls []decl
	usedVar := map[string]bool{}
	usedLocal := map[string]bool{}
	usedOutput := map[string]bool{}
	ignored := map[string]bool{} // key: kind|name

	for _, f := range files {
		src, rerr := os.ReadFile(f)
		if rerr != nil {
			return nil, rerr
		}
		file, diags := hclsyntax.ParseConfig(src, f, hcl.Pos{Line: 1, Column: 1})
		if diags.HasErrors() {
			return nil, diags
		}
		body, ok := file.Body.(*hclsyntax.Body)
		if !ok {
			continue
		}
		lines := splitLines(src)
		walk(body, f, lines, &decls, usedVar, usedLocal, usedOutput, ignored)
	}

	var findings []Finding
	for _, d := range decls {
		if ignored[string(d.kind)+"|"+d.name] {
			continue
		}
		switch d.kind {
		case KindVariable:
			if !usedVar[d.name] {
				findings = append(findings, d.finding())
			}
		case KindLocal:
			if !usedLocal[d.name] {
				findings = append(findings, d.finding())
			}
		case KindOutput:
			if opts.StrictOutputs && !usedOutput[d.name] {
				findings = append(findings, d.finding())
			}
		}
	}

	sort.Slice(findings, func(i, j int) bool {
		if findings[i].File != findings[j].File {
			return findings[i].File < findings[j].File
		}
		if findings[i].Line != findings[j].Line {
			return findings[i].Line < findings[j].Line
		}
		return findings[i].Name < findings[j].Name
	})
	return findings, nil
}

// walk descends a body, recording declarations and collecting references from
// every attribute expression it encounters.
func walk(body *hclsyntax.Body, file string, lines []string, decls *[]decl,
	usedVar, usedLocal, usedOutput, ignored map[string]bool) {

	for _, attr := range body.Attributes {
		collectRefs(attr.Expr, usedVar, usedLocal, usedOutput)
	}

	for _, block := range body.Blocks {
		switch block.Type {
		case "variable":
			if len(block.Labels) == 1 {
				line := block.DefRange().Start.Line
				*decls = append(*decls, decl{KindVariable, block.Labels[0], file, line})
				markIgnored(KindVariable, block.Labels[0], line, lines, ignored)
			}
		case "output":
			if len(block.Labels) == 1 {
				line := block.DefRange().Start.Line
				*decls = append(*decls, decl{KindOutput, block.Labels[0], file, line})
				markIgnored(KindOutput, block.Labels[0], line, lines, ignored)
			}
		case "locals":
			for name, attr := range block.Body.Attributes {
				line := attr.SrcRange.Start.Line
				*decls = append(*decls, decl{KindLocal, name, file, line})
				markIgnored(KindLocal, name, line, lines, ignored)
			}
		}
		walk(block.Body, file, lines, decls, usedVar, usedLocal, usedOutput, ignored)
	}
}

// collectRefs records every var.<name> and local.<name> traversal in expr.
func collectRefs(expr hclsyntax.Expression, usedVar, usedLocal, usedOutput map[string]bool) {
	for _, tr := range expr.Variables() {
		if len(tr) < 2 {
			continue
		}
		attr, ok := tr[1].(hcl.TraverseAttr)
		if !ok {
			continue
		}
		switch tr.RootName() {
		case "var":
			usedVar[attr.Name] = true
		case "local":
			usedLocal[attr.Name] = true
		case "output":
			usedOutput[attr.Name] = true
		}
	}
}

// markIgnored waives a declaration when its line, or the line just above it,
// carries the ignore marker in a comment.
func markIgnored(kind Kind, name string, line int, lines []string, ignored map[string]bool) {
	if lineHasMarker(lines, line) || lineHasMarker(lines, line-1) {
		ignored[string(kind)+"|"+name] = true
	}
}

func lineHasMarker(lines []string, line int) bool {
	if line < 1 || line > len(lines) {
		return false
	}
	return strings.Contains(lines[line-1], ignoreMarker)
}

func splitLines(src []byte) []string {
	var out []string
	sc := bufio.NewScanner(strings.NewReader(string(src)))
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		out = append(out, sc.Text())
	}
	return out
}

// collectFiles expands paths into a deduplicated, sorted list of .tf files.
func collectFiles(paths []string) ([]string, error) {
	seen := map[string]bool{}
	var out []string
	add := func(p string) {
		if !seen[p] {
			seen[p] = true
			out = append(out, p)
		}
	}
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			return nil, err
		}
		if info.IsDir() {
			werr := filepath.WalkDir(p, func(path string, d os.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if d.IsDir() {
					if d.Name() == ".terraform" {
						return filepath.SkipDir
					}
					return nil
				}
				if strings.HasSuffix(path, ".tf") {
					add(path)
				}
				return nil
			})
			if werr != nil {
				return nil, werr
			}
			continue
		}
		if strings.HasSuffix(p, ".tf") {
			add(p)
		}
	}
	sort.Strings(out)
	return out, nil
}
