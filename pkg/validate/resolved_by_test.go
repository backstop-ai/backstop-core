package validate_test

import (
	"path/filepath"
	"testing"

	"github.com/bmanson/backstop-core/pkg/artifact"
	"github.com/bmanson/backstop-core/pkg/validate"
)

// resolvedByRoot is the absolute path to the resolved-by fixture tree. The typed-ref
// existence check resolves the referenced artifact relative to the issue's OWN path,
// so tests pass ABSOLUTE fixture paths and never rely on the process working dir.
func resolvedByRoot(t *testing.T) string {
	t.Helper()
	return filepath.Join(terminalRepoRoot(t), "pkg", "validate", "testdata", "resolved_by")
}

// parseResolvedByFixture parses a resolved-by issue fixture by its path relative to
// the resolved_by root, using its absolute path so SourcePath anchoring works.
func parseResolvedByFixture(t *testing.T, rel string) *artifact.ParsedArtifact {
	t.Helper()
	path := filepath.Join(resolvedByRoot(t), rel)
	art, err := artifact.ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile(%s): %v", rel, err)
	}
	return art
}

// TestResolvedBy_ValidCloseWithoutOwnClaimsOrPlan (CLM-004): a closed issue with a
// valid resolved-by (commit-SHA ref OR typed ref to an existing artifact) plus a
// Resolution section validates with ZERO violations, WITHOUT its own requirements/
// claims/verification/implementation/contracts and WITHOUT a backing plan or a
// mandated test. Proves a directly-fixed issue with NO test can honestly close.
func TestResolvedBy_ValidCloseWithoutOwnClaimsOrPlan(t *testing.T) {
	sch := loadTerminalSchema(t, "issue", "v1")
	cases := []struct {
		name string
		rel  string
	}{
		{"commit-ref", "issues/ISSUE-830-resolved-commit.issue.md"},
		{"typed-ref-exists", "issues/ISSUE-831-resolved-typedref.issue.md"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			art := parseResolvedByFixture(t, tc.rel)
			res := validate.Issue(art, sch)
			if len(res.Violations) != 0 {
				t.Fatalf("valid resolved-by close (%s) expected clean, got: %v", tc.name, res.Violations)
			}
			// The own-REQ/CLM chain was NOT demanded.
			assertNoViolationRule(t, res, "issue/requirements-required")
			assertNoViolationRule(t, res, "issue/claims-required")
			assertNoViolationRule(t, res, "issue/verification-required")
			assertNoViolationRule(t, res, "issue/contracts-required")
		})
	}
}

// TestResolvedBy_MalformedShape_Errors (CLM-005): a resolved-by that is neither a
// typed artifact ref nor a commit/PR ref (arbitrary prose, or empty) is a fail-loud
// issue/resolved-by-malformed error — the vacuous-close hatch is shut.
func TestResolvedBy_MalformedShape_Errors(t *testing.T) {
	sch := loadTerminalSchema(t, "issue", "v1")

	// (a) arbitrary prose fixture.
	t.Run("prose", func(t *testing.T) {
		art := parseResolvedByFixture(t, "issues/ISSUE-832-resolved-prose.issue.md")
		res := validate.Issue(art, sch)
		assertErrorViolation(t, res, "issue/resolved-by-malformed")
		// Free text is rejected on SHAPE, before any existence lookup.
		assertNoViolationRule(t, res, "issue/resolved-by-artifact-not-found")
	})

	// (b) empty-string value — present but blank is still malformed, not a silent
	//     fall-through to the full REQ/CLM chain.
	t.Run("empty-string", func(t *testing.T) {
		content := "---\n" +
			"title: \"Close with an empty resolved-by\"\n" +
			"schema_version: issue/v1\n\n" +
			"issue:\n" +
			"  id: ISSUE-837\n" +
			"  title: \"Close with an empty resolved-by\"\n" +
			"  type: bug\n" +
			"  status: closed\n" +
			"  created: \"2026-07-08\"\n" +
			"  closed: \"2026-07-08\"\n\n" +
			"resolved-by: \"\"\n" +
			"---\n\n" +
			"# Close with an empty resolved-by\n\n" +
			"## Problem\n\nEmpty resolved-by.\n\n" +
			"## Resolution\n\nEmpty pointer must be malformed.\n"
		art, err := artifact.Parse(content, "ISSUE-837-empty.issue.md")
		if err != nil {
			t.Fatal(err)
		}
		res := validate.Issue(art, sch)
		assertErrorViolation(t, res, "issue/resolved-by-malformed")
	})
}

// TestResolvedBy_TypedRefMustExist (CLM-005): a well-shaped TYPED resolved-by ref
// that resolves to no artifact file is issue/resolved-by-artifact-not-found; a
// commit-SHA ref is shape-only and does NOT trigger the existence check.
func TestResolvedBy_TypedRefMustExist(t *testing.T) {
	sch := loadTerminalSchema(t, "issue", "v1")

	// Typed ref to a nonexistent artifact -> existence failure.
	missing := parseResolvedByFixture(t, "issues/ISSUE-833-resolved-missing-ref.issue.md")
	missingRes := validate.Issue(missing, sch)
	assertErrorViolation(t, missingRes, "issue/resolved-by-artifact-not-found")
	// A missing artifact is not a shape problem.
	assertNoViolationRule(t, missingRes, "issue/resolved-by-malformed")

	// Commit-SHA ref: existence is enforced for typed refs ONLY — the SHA close is
	// clean and never triggers the existence check.
	commit := parseResolvedByFixture(t, "issues/ISSUE-830-resolved-commit.issue.md")
	commitRes := validate.Issue(commit, sch)
	assertNoViolationRule(t, commitRes, "issue/resolved-by-artifact-not-found")
	if len(commitRes.Violations) != 0 {
		t.Fatalf("commit-SHA resolved-by close expected clean, got: %v", commitRes.Violations)
	}
}

// TestResolvedBy_MissingResolutionSection_Errors (CLM-006): a resolved-by close
// lacking a Resolution section is issue/resolved-by-resolution-required (minimum
// standalone content), even with an otherwise-valid resolved-by ref.
func TestResolvedBy_MissingResolutionSection_Errors(t *testing.T) {
	sch := loadTerminalSchema(t, "issue", "v1")
	art := parseResolvedByFixture(t, "issues/ISSUE-834-resolved-no-resolution.issue.md")
	res := validate.Issue(art, sch)
	assertErrorViolation(t, res, "issue/resolved-by-resolution-required")
	// The ref itself is a valid commit SHA — no malformed/existence defect.
	assertNoViolationRule(t, res, "issue/resolved-by-malformed")
	assertNoViolationRule(t, res, "issue/resolved-by-artifact-not-found")
}

// TestResolvedBy_ConditionalNoRegression (CLM-007): a bare close (neither
// resolved-by nor delivered_by) still reports the full requirements/claims-required
// violations — the relaxation is conditional, not a general loosening.
func TestResolvedBy_ConditionalNoRegression(t *testing.T) {
	sch := loadTerminalSchema(t, "issue", "v1")
	bare := parseResolvedByFixture(t, "issues/ISSUE-835-bare-close.issue.md")
	res := validate.Issue(bare, sch)
	assertHasViolation(t, res, "issue/requirements-required")
	assertHasViolation(t, res, "issue/claims-required")
	// No close pointer present -> no pointer-specific violation.
	assertNoViolationRule(t, res, "issue/resolved-by-malformed")
	assertNoViolationRule(t, res, "issue/close-pointer-conflict")
}

// TestResolvedBy_BothPointersConflict (CLM-007): a closed issue carrying BOTH
// delivered_by AND resolved-by is a fail-loud issue/close-pointer-conflict error —
// at most one close pointer, no silent precedence.
func TestResolvedBy_BothPointersConflict(t *testing.T) {
	sch := loadTerminalSchema(t, "issue", "v1")
	art := parseResolvedByFixture(t, "issues/ISSUE-836-both-pointers.issue.md")
	res := validate.Issue(art, sch)
	assertErrorViolation(t, res, "issue/close-pointer-conflict")
}
