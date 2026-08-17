package packval_test

import (
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/packval"
)

// TestPackVal_P1_RulePathFileExistenceIsChecked (CLM-006): phase 1's "the rule file
// you declared exists" structural check must be live for a rule that names its source
// with `rule_path:`. Before ISSUE-092 the branch was guarded on `rule.File != ""`, so
// for every real pack — none of which declares `file:` — phase 1 silently checked
// nothing at all.
func TestPackVal_P1_RulePathFileExistenceIsChecked(t *testing.T) {
	const missing = "rules/definitely-not-here.yml"
	m := &packval.PackManifest{
		Name: "acme/p1", Version: "1.0.0", Language: "go", Archetype: "enforcement",
		Content: packval.Content{Ruleset: packval.Ruleset{Rules: []packval.Rule{{
			ID: "R1", Engine: "semgrep", RulePath: missing, RiskClass: "correctness",
		}}}},
	}

	res := packval.RunStructural(m, t.TempDir())

	found := false
	for _, e := range res.Errors {
		if e.Check == "file-exists" && e.Rule == "R1" {
			found = true
			if !strings.Contains(e.ManifestPath, "rule_path") {
				t.Errorf("the structural error must point at the manifest key actually used, got ManifestPath %q", e.ManifestPath)
			}
		}
	}
	if !found {
		t.Fatalf("expected a file-exists error naming the missing rule_path %q; got %+v", missing, res.Errors)
	}
	if res.Status != "fail" {
		t.Fatalf("a declared-but-absent rule source file must fail phase 1, got status %q", res.Status)
	}
}

// TestPackVal_P1_RulePathPresentFilePasses is the discriminating half: the same
// manifest whose rule_path DOES exist must not raise a file-exists error. Without it
// the test above would also pass against an implementation that failed everything.
func TestPackVal_P1_RulePathPresentFilePasses(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "rules/present.yml", "rules:\n  - id: R1\n")
	m := &packval.PackManifest{
		Name: "acme/p1", Version: "1.0.0", Language: "go", Archetype: "enforcement",
		Content: packval.Content{Ruleset: packval.Ruleset{Rules: []packval.Rule{{
			ID: "R1", Engine: "semgrep", RulePath: "rules/present.yml", RiskClass: "correctness",
		}}}},
	}

	res := packval.RunStructural(m, dir)

	for _, e := range res.Errors {
		if e.Check == "file-exists" {
			t.Fatalf("a present rule_path must not raise file-exists; got %+v", e)
		}
	}
}
