package validate_test

import (
	"path/filepath"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/artifact"
	"github.com/backstop-ai/backstop-core/pkg/validate"
)

// deliveredByRoot returns the absolute path to the shared delivered_by fixture
// root (…/pkg/validate/testdata/delivered_by). Resolution is anchored on the
// issue's own SourcePath, so tests pass ABSOLUTE fixture paths and never rely on
// the process working directory (ISSUE-043 CLM-012).
func deliveredByRoot(t *testing.T) string {
	t.Helper()
	return filepath.Join(terminalRepoRoot(t), "pkg", "validate", "testdata", "delivered_by")
}

// parseDeliveredByFixture parses an issue fixture by its path relative to the
// delivered_by root, using its absolute path so SourcePath anchoring works
// regardless of CWD.
func parseDeliveredByFixture(t *testing.T, rel string) *artifact.ParsedArtifact {
	t.Helper()
	path := filepath.Join(deliveredByRoot(t), rel)
	art, err := artifact.ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile(%s): %v", rel, err)
	}
	return art
}

// TestIssueDeliveredBy_ValidTraceClosesWithoutOwnClaims (CLM-001): a closed
// issue with a valid delivered_by pointer + a Resolution section validates with
// ZERO violations, WITHOUT its own requirements/claims/verification/
// implementation/contracts arrays.
func TestIssueDeliveredBy_ValidTraceClosesWithoutOwnClaims(t *testing.T) {
	sch := loadTerminalSchema(t, "issue", "v1")
	art := parseDeliveredByFixture(t, "root/issues/ISSUE-901-thin-close.issue.md")
	res := validate.Issue(art, sch)

	if len(res.Violations) != 0 {
		t.Fatalf("thin delivered_by close expected clean, got: %v", res.Violations)
	}
	// Specifically pin that the own-REQ/CLM chain was NOT demanded.
	assertNoViolationRule(t, res, "issue/requirements-required")
	assertNoViolationRule(t, res, "issue/claims-required")
	assertNoViolationRule(t, res, "issue/verification-required")
	assertNoViolationRule(t, res, "issue/contracts-required")
}

// TestIssueDeliveredBy_ResolvesFromArtifactPath_NotCWD (CLM-012, integration):
// validate ISSUE-901 while the process working directory is an UNRELATED temp
// dir — deliberately NOT the fixture root/plans parent. The trace must STILL
// resolve the backing plan via the artifact's own SourcePath and validate clean.
// Guards against a fixture whose CWD coincidentally equals the plans parent.
func TestIssueDeliveredBy_ResolvesFromArtifactPath_NotCWD(t *testing.T) {
	sch := loadTerminalSchema(t, "issue", "v1")
	// Resolve the absolute fixture path BEFORE chdir so terminalRepoRoot (which
	// walks up from CWD) is not perturbed by the chdir below.
	path := filepath.Join(deliveredByRoot(t), "root/issues/ISSUE-901-thin-close.issue.md")

	t.Chdir(t.TempDir()) // unrelated dir — NOT the fixture root/plans parent

	art, err := artifact.ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	res := validate.Issue(art, sch)
	if len(res.Violations) != 0 {
		t.Fatalf("delivered_by trace must resolve via SourcePath from an unrelated CWD, got: %v", res.Violations)
	}
	assertNoViolationRule(t, res, "issue/delivered-by-plans-unresolvable")
	assertNoViolationRule(t, res, "issue/delivered-by-plan-not-found")
}

// TestIssueDeliveredBy_MalformedPointer_Errors (CLM-002): a delivered_by value
// that does not match PLAN-ISSUE-NNN is a fail-loud error, before any plan
// lookup.
func TestIssueDeliveredBy_MalformedPointer_Errors(t *testing.T) {
	sch := loadTerminalSchema(t, "issue", "v1")
	art := parseDeliveredByFixture(t, "root/issues/ISSUE-902-malformed-pointer.issue.md")
	res := validate.Issue(art, sch)
	assertHasViolation(t, res, "issue/delivered-by-malformed")
	// Malformed pointer must not be mistaken for a plans-dir problem.
	assertNoViolationRule(t, res, "issue/delivered-by-plan-not-found")
}

// TestIssueDeliveredBy_MissingPlanFile_Errors (CLM-003): a delivered_by naming a
// plan file that does not exist under the resolved plans/ dir is an error.
func TestIssueDeliveredBy_MissingPlanFile_Errors(t *testing.T) {
	sch := loadTerminalSchema(t, "issue", "v1")
	art := parseDeliveredByFixture(t, "root/issues/ISSUE-903-missing-plan.issue.md")
	res := validate.Issue(art, sch)
	assertHasViolation(t, res, "issue/delivered-by-plan-not-found")
	// The plans/ dir DOES resolve here — only the file is missing.
	assertNoViolationRule(t, res, "issue/delivered-by-plans-unresolvable")
}

// TestIssueDeliveredBy_PlansDirUnresolvable_Errors (CLM-011): a delivered_by on
// a closed issue whose plans/ dir is unresolvable is a fail-loud error, NEVER a
// silent trace-satisfied pass. Covers (a) a real fixture with no sibling plans/
// dir and (b) a directory-less (base-only) SourcePath.
func TestIssueDeliveredBy_PlansDirUnresolvable_Errors(t *testing.T) {
	sch := loadTerminalSchema(t, "issue", "v1")

	// (a) fixture under no-plans-dir/ — no sibling plans/ dir.
	art := parseDeliveredByFixture(t, "no-plans-dir/issues/ISSUE-911-orphan-close.issue.md")
	res := validate.Issue(art, sch)
	assertHasViolation(t, res, "issue/delivered-by-plans-unresolvable")
	// Must not silently satisfy the close by falling through to a clean pass.
	assertNoViolationRule(t, res, "issue/delivered-by-plan-not-found")

	// (b) directory-less SourcePath (Parse with a base-only synthetic filename).
	content := "---\n" +
		"title: \"Base-only synthetic close\"\n" +
		"schema_version: issue/v1\n\n" +
		"issue:\n" +
		"  id: ISSUE-901\n" +
		"  title: \"Base-only synthetic close\"\n" +
		"  type: enhancement\n" +
		"  status: closed\n" +
		"  created: \"2026-07-08\"\n" +
		"  closed: \"2026-07-08\"\n\n" +
		"delivered_by: PLAN-ISSUE-901\n" +
		"---\n\n" +
		"# Base-only synthetic close\n\n" +
		"## Problem\n\nBase-only path.\n\n" +
		"## Resolution\n\nBase-only path, so plans/ cannot be resolved.\n"
	baseArt, err := artifact.Parse(content, "ISSUE-901-base-only.issue.md")
	if err != nil {
		t.Fatal(err)
	}
	baseRes := validate.Issue(baseArt, sch)
	assertHasViolation(t, baseRes, "issue/delivered-by-plans-unresolvable")
}

// TestIssueDeliveredBy_PlanNotCompleted_Errors (CLM-004): a delivered_by plan
// whose status is not `completed` is an error. Table-driven over draft, replaced
// (retired), and canceled (retired) — BOTH retired terminals and a live
// non-terminal must be rejected.
func TestIssueDeliveredBy_PlanNotCompleted_Errors(t *testing.T) {
	sch := loadTerminalSchema(t, "issue", "v1")
	cases := []struct {
		name    string
		fixture string
	}{
		{"draft", "root/issues/ISSUE-904-draft-plan.issue.md"},
		{"replaced", "root/issues/ISSUE-910-replaced-plan.issue.md"},
		{"canceled", "root/issues/ISSUE-912-canceled-plan.issue.md"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			art := parseDeliveredByFixture(t, tc.fixture)
			res := validate.Issue(art, sch)
			assertHasViolation(t, res, "issue/delivered-by-plan-not-completed")
			// A non-completed but otherwise-valid plan must not report plan-invalid.
			assertNoViolationRule(t, res, "issue/delivered-by-plan-invalid")
		})
	}
}

// TestIssueDeliveredBy_InvalidPlan_Errors (CLM-005): a delivered_by naming a
// plan that itself fails validate.Plan is an error (the trace reuses
// validate.Plan, not a reimplementation).
func TestIssueDeliveredBy_InvalidPlan_Errors(t *testing.T) {
	sch := loadTerminalSchema(t, "issue", "v1")
	art := parseDeliveredByFixture(t, "root/issues/ISSUE-905-invalid-plan.issue.md")
	res := validate.Issue(art, sch)
	assertHasViolation(t, res, "issue/delivered-by-plan-invalid")
}

// TestIssueDeliveredBy_PlanBacksDifferentIssue_Errors (CLM-006): a delivered_by
// plan whose spec_id != the closing issue's id (backs a different issue) is an
// error.
func TestIssueDeliveredBy_PlanBacksDifferentIssue_Errors(t *testing.T) {
	sch := loadTerminalSchema(t, "issue", "v1")
	art := parseDeliveredByFixture(t, "root/issues/ISSUE-906-wrong-backing.issue.md")
	res := validate.Issue(art, sch)
	assertHasViolation(t, res, "issue/delivered-by-spec-mismatch")
}

// TestIssueDeliveredBy_MissingResolutionSection_Errors (CLM-008): a
// delivered_by-closed issue lacking a Resolution section is an error (minimum
// standalone content), even when the backing plan is a clean completed trace.
func TestIssueDeliveredBy_MissingResolutionSection_Errors(t *testing.T) {
	sch := loadTerminalSchema(t, "issue", "v1")
	art := parseDeliveredByFixture(t, "root/issues/ISSUE-907-no-resolution.issue.md")
	res := validate.Issue(art, sch)
	assertHasViolation(t, res, "issue/delivered-by-resolution-required")
	// The backing plan itself is a valid completed trace — no plan-level defect.
	assertNoViolationRule(t, res, "issue/delivered-by-plan-not-completed")
	assertNoViolationRule(t, res, "issue/delivered-by-plan-invalid")
}

// TestIssueClosed_NoDeliveredBy_StillRequiresTraceability (CLM-007): a bare
// close (no delivered_by) still reports the full requirements/claims-required
// violations, while a full close (no delivered_by, full chain) validates clean.
// Proves the relaxation is conditional, not a general loosening.
func TestIssueClosed_NoDeliveredBy_StillRequiresTraceability(t *testing.T) {
	sch := loadTerminalSchema(t, "issue", "v1")

	// Negative half: bare close must still demand the full chain.
	bare := parseDeliveredByFixture(t, "root/issues/ISSUE-908-bare-close.issue.md")
	bareRes := validate.Issue(bare, sch)
	assertHasViolation(t, bareRes, "issue/requirements-required")
	assertHasViolation(t, bareRes, "issue/claims-required")

	// Positive half: full close with no delivered_by validates clean.
	full := parseDeliveredByFixture(t, "root/issues/ISSUE-909-full-close.issue.md")
	fullRes := validate.Issue(full, sch)
	if len(fullRes.Violations) != 0 {
		t.Fatalf("full non-plan-backed close expected clean, got: %v", fullRes.Violations)
	}
}
