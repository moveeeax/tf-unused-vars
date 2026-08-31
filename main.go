// Command tf-unused-vars finds declared-but-unused Terraform variables,
// locals, and outputs in a module and exits non-zero so CI can gate on it.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/moveeeax/tf-unused-vars/internal/analyzer"
)

const usage = `tf-unused-vars — find unused Terraform variables, locals, and outputs

Usage:
  tf-unused-vars scan [paths...] [flags]

Arguments:
  paths            files or directories to scan (default ".")

Flags:
  --json           emit findings as a JSON array
  --strict-outputs also flag outputs that nothing in the module references

Exit status:
  0  no unused declarations
  1  at least one unused declaration found
  2  usage or parse error
`

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" || args[0] == "help" {
		fmt.Fprint(os.Stdout, usage)
		if len(args) == 0 {
			return 2
		}
		return 0
	}
	if args[0] != "scan" {
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n%s", args[0], usage)
		return 2
	}

	fs := flag.NewFlagSet("scan", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	asJSON := fs.Bool("json", false, "emit findings as JSON")
	strict := fs.Bool("strict-outputs", false, "flag unreferenced outputs")
	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}

	paths := fs.Args()
	if len(paths) == 0 {
		paths = []string{"."}
	}

	findings, err := analyzer.Analyze(paths, analyzer.Options{StrictOutputs: *strict})
	if err != nil {
		fmt.Fprintf(os.Stderr, "tf-unused-vars: %v\n", err)
		return 2
	}

	if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if findings == nil {
			findings = []analyzer.Finding{}
		}
		if err := enc.Encode(findings); err != nil {
			fmt.Fprintf(os.Stderr, "tf-unused-vars: %v\n", err)
			return 2
		}
	} else {
		printTable(findings)
	}

	if len(findings) > 0 {
		return 1
	}
	return 0
}

func printTable(findings []analyzer.Finding) {
	if len(findings) == 0 {
		fmt.Println("No unused declarations found.")
		return
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintln(w, "KIND\tNAME\tLOCATION")
	for _, f := range findings {
		fmt.Fprintf(w, "%s\t%s\t%s:%d\n", f.Kind, f.Name, f.File, f.Line)
	}
	w.Flush()
	fmt.Printf("\n%d unused declaration(s) found.\n", len(findings))
}
