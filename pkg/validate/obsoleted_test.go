package validate_test

import (
	"path/filepath"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/artifact"
	"github.com/backstop-ai/backstop-core/pkg/validate"
)

// terminalVocabRoot is the absolute path to the obsoleted terminal-vocab fixture
// tree. Anchored on the repo root (not CWD) so parsing is deterministic.
func terminalVocabRoot(t *testing.T) string {
	t.Helper()
	return filepath.Join(terminalRepoRoot(t), "pkg", "validate", "testdata", "terminal_vocab")
}

// validateVocabFixture parses a terminal_vocab fixture by its path relative to the
// vocab root and dispatches to the type-appropriate validator. Uses the absolute
// path so SourcePath anchoring works regardless of the process working directory.
func validateVocabFixture(t *testing.T, kind, rel string) validate.ValidationResult {
	t.Helper()
	path := filepath.Join(terminalVocabRoot(t), rel)
	art, err := artifact.ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile(%s): %v", rel, err)
	}
	switch kind {
	case "issue":
		return validate.Issue(art, loadTerminalSchema(t, "issue", "v1"))
	case "spec":
		return validate.Spec(art, loadTerminalSchema(t, "spec", "v1"))
	case "plan":
		return validate.Plan(art, nil)
	default:
		t.Fatalf("unknown kind %q", kind)
		return validate.ValidationResult{}
	}
}

// assertErrorViolation asserts a violation with the given rule AND error severity
// is present — pinning the intended fail-loud rule so the test cannot pass on a
// warning or a differently-named violation.
func assertErrorViolation(t *testing.T, res validate.ValidationResult, rule string) {
	t.Helper()
	for _, v := range res.Violations {
		if v.Rule == rule {
			if v.Severity != "error" {
				t.Errorf("violation %q has severity %q, want error", rule, v.Severity)
			}
			return
		}
	}
	t.Errorf("expected error-severity violation %q, got none. Violations:", rule)
	for _, v := range res.Violations {
		t.Errorf("  [%s/%s] %s", v.Severity, v.Rule, v.Message)
	}
}

// TestObsoleted_RequiresObsoletedBy (CLM-002): status `obsoleted` REQUIRES a
// non-empty, TYPED obsoleted-by across issue/spec/plan. Absent -> fail-loud
// <type>/obsoleted-by-required; malformed/non-typed -> <type>/obsoleted-by-malformed.
func TestObsoleted_RequiresObsoletedBy(t *testing.T) {
	// Absent obsoleted-by across all three artifact types.
	absent := []struct {
		name string
		kind string
		rel  string
		rule string
	}{
		{"issue", "issue", "issues/ISSUE-821-obsoleted-no-by.issue.md", "issue/obsoleted-by-required"},
		{"spec", "spec", "specs/SPEC-821-obsoleted-no-by.spec.md", "spec/obsoleted-by-required"},
		{"plan", "plan", "plans/PLAN-ISSUE-821-obsoleted-no-by.plan.yml", "plan/obsoleted-by-required"},
	}
	for _, tc := range absent {
		t.Run(tc.name+"-absent", func(t *testing.T) {
			res := validateVocabFixture(t, tc.kind, tc.rel)
			assertErrorViolation(t, res, tc.rule)
		})
	}

	// Malformed (non-typed) obsoleted-by on the issue fixture.
	t.Run("issue-malformed", func(t *testing.T) {
		res := validateVocabFixture(t, "issue", "issues/ISSUE-822-obsoleted-malformed-by.issue.md")
		assertErrorViolation(t, res, "issue/obsoleted-by-malformed")
		// A malformed ref is not the same failure as an absent one.
		assertNoViolationRule(t, res, "issue/obsoleted-by-required")
	})
}

// TestObsoleted_ValidTerminalValidatesClean (CLM-001, CLM-003): an obsoleted
// artifact with a typed obsoleted-by validates with ZERO violations across
// issue/spec/plan — proving obsoleted is both an ACCEPTED status (not
// invalid-status) AND a terminal exempt from the live-work REQ->CLM->tests rigor.
func TestObsoleted_ValidTerminalValidatesClean(t *testing.T) {
	cases := []struct {
		name string
		kind string
		rel  string
	}{
		{"issue", "issue", "issues/ISSUE-820-obsoleted-valid.issue.md"},
		{"spec", "spec", "specs/SPEC-820-obsoleted-valid.spec.md"},
		{"plan", "plan", "plans/PLAN-ISSUE-820-obsoleted.plan.yml"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := validateVocabFixture(t, tc.kind, tc.rel)
			if len(res.Violations) != 0 {
				t.Fatalf("valid obsoleted %s expected clean, got: %v", tc.name, res.Violations)
			}
			// Terminal exemption + accepted status: none of these fire.
			assertNoViolationRule(t, res, tc.kind+"/requirements-required")
			assertNoViolationRule(t, res, tc.kind+"/claims-required")
			assertNoViolationRule(t, res, tc.kind+"/invalid-status")
			assertNoViolationRule(t, res, tc.kind+"/status-enum")
		})
	}
}
