package analyzer

import (
	"testing"
)

func names(findings []Finding, kind Kind) map[string]bool {
	out := map[string]bool{}
	for _, f := range findings {
		if f.Kind == kind {
			out[f.Name] = true
		}
	}
	return out
}

func TestUnusedModule(t *testing.T) {
	findings, err := Analyze([]string{"../../testdata/unused"}, Options{})
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}

	vars := names(findings, KindVariable)
	if !vars["instance_type"] {
		t.Errorf("expected instance_type flagged as unused variable, got %v", vars)
	}
	if vars["region"] || vars["instance_count"] {
		t.Errorf("used variables were wrongly flagged: %v", vars)
	}

	locals := names(findings, KindLocal)
	if !locals["legacy_tags"] {
		t.Errorf("expected legacy_tags flagged as unused local, got %v", locals)
	}
	if locals["name_prefix"] {
		t.Errorf("used local name_prefix was wrongly flagged")
	}

	// Cross-file resolution: region is declared in variables.tf and used in main.tf.
	for _, f := range findings {
		if f.Name == "region" {
			t.Errorf("region should resolve across files, but was flagged: %+v", f)
		}
	}
}

func TestCleanModuleExitsZero(t *testing.T) {
	findings, err := Analyze([]string{"../../testdata/clean"}, Options{})
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("expected no findings for a clean module, got %v", findings)
	}
}

func TestStrictOutputs(t *testing.T) {
	// Without strict, the leaf-module output is not flagged.
	findings, err := Analyze([]string{"../../testdata/outputs"}, Options{})
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}
	if len(names(findings, KindOutput)) != 0 {
		t.Errorf("outputs should not be flagged without --strict-outputs: %v", findings)
	}

	strictFindings, err := Analyze([]string{"../../testdata/outputs"}, Options{StrictOutputs: true})
	if err != nil {
		t.Fatalf("analyze strict: %v", err)
	}
	if !names(strictFindings, KindOutput)["bucket_name"] {
		t.Errorf("expected bucket_name flagged under --strict-outputs, got %v", strictFindings)
	}
}

func TestIgnoreComment(t *testing.T) {
	findings, err := Analyze([]string{"../../testdata/ignore"}, Options{})
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}
	vars := names(findings, KindVariable)
	if vars["kept_on_purpose"] {
		t.Errorf("kept_on_purpose carries an ignore marker and must be waived")
	}
	if !vars["really_unused"] {
		t.Errorf("really_unused should still be flagged, got %v", vars)
	}
}

func TestReportsFileAndLine(t *testing.T) {
	findings, err := Analyze([]string{"../../testdata/unused"}, Options{})
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}
	for _, f := range findings {
		if f.File == "" || f.Line <= 0 {
			t.Errorf("finding is missing location: %+v", f)
		}
	}
}
