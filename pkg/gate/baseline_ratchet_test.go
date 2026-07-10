package gate

import (
	"context"
	"testing"
)

// ISSUE-050 — the strict file-level ratchet, welded into CompareBaseline (the
// single grandfathering chokepoint) so all three baseline consumers inherit it.
// A baseline-present finding whose file is explicitly TOUCHED (diff/file scope
// Contains it) has its grandfather REVOKED and reclassifies as NEW. Untouched
// files, project-wide findings on unchanged files, --all mode, and nil scope all
// keep grandfathering unchanged. These tests exercise CompareBaseline directly
// (and, for CLM-007, the aggregate computeBaselineResult step through Run) with
// hand-built Violation / BaselineArtifact / GateScope fixtures — no engine run.

// TestPolicy_TouchedFileRevokesGrandfathering (CLM-001, CLM-002): a
// baseline-present finding on a file the author touched is reclassified NEW where
// the old half-ratchet grandfathered it. The sanity companion pins that the TOUCH
// is the cause: the identical baseline-present finding whose file is NOT in the
// scope's touched set stays grandfathered.
func TestPolicy_TouchedFileRevokesGrandfathering(t *testing.T) {
	finding := vioSrc("code_check/no-eval", "pkg/x/x.go", "r1", "test/pack")
	baseline := &BaselineArtifact{SchemaVersion: BaselineSchemaV1, Violations: []Violation{finding}}

	touchedScope := newGateScope("", GateScopeModeDiff, []string{"pkg/x/x.go"}, nil)
	cmp := CompareBaseline([]Violation{finding}, baseline, BaselineCompareOptions{Scope: touchedScope})
	if len(cmp.NewViolations) != 1 {
		t.Fatalf("touched baseline-present finding must be revoked as NEW, got %d: %#v", len(cmp.NewViolations), cmp.NewViolations)
	}
	if cmp.NewViolations[0].File != "pkg/x/x.go" {
		t.Fatalf("expected revoked new violation on pkg/x/x.go, got %#v", cmp.NewViolations[0])
	}

	// Sanity companion: SAME baseline-present finding, but the scope touches a
	// DIFFERENT file. ProjectWide keeps it in scope (so it isn't merely filtered
	// out) yet, untouched, it must stay grandfathered — proving touch is the cause.
	projectWide := finding
	projectWide.ProjectWide = true
	otherScope := newGateScope("", GateScopeModeDiff, []string{"pkg/y/y.go"}, nil)
	cmp2 := CompareBaseline([]Violation{projectWide}, baseline, BaselineCompareOptions{Scope: otherScope})
	if len(cmp2.NewViolations) != 0 {
		t.Fatalf("untouched baseline-present finding must stay grandfathered, got %d: %#v", len(cmp2.NewViolations), cmp2.NewViolations)
	}
}

// TestPolicy_UntouchedFileUnderProjectWideDimensionStaysGrandfathered (CLM-003):
// a project-wide-scanning dimension reports a baseline-present finding on a file
// the author did NOT touch (ProjectWide keeps it past filterViolations). It stays
// grandfathered — revocation is keyed strictly on the touched file, never the
// whole dimension or the whole repo's debt in one commit.
func TestPolicy_UntouchedFileUnderProjectWideDimensionStaysGrandfathered(t *testing.T) {
	finding := vioSrc("go-toolchain/lint", "pkg/other/other.go", "r1", "test/toolchain")
	finding.ProjectWide = true
	baseline := &BaselineArtifact{SchemaVersion: BaselineSchemaV1, Violations: []Violation{finding}}

	// Scope touches a DIFFERENT file than the finding's.
	scope := newGateScope("", GateScopeModeDiff, []string{"pkg/changed/changed.go"}, nil)
	cmp := CompareBaseline([]Violation{finding}, baseline, BaselineCompareOptions{Scope: scope})
	if len(cmp.NewViolations) != 0 {
		t.Fatalf("project-wide finding on an untouched file must stay grandfathered, got %d: %#v", len(cmp.NewViolations), cmp.NewViolations)
	}
}

// TestPolicy_AllModeUnaffectedByStrictRevocation (CLM-004): the identical
// baseline-present finding under GateScopeModeAll stays grandfathered — the
// Mode∈{Diff,File} guard excludes All (where scope.Contains is universally true),
// so gate --all / baseline generate / seeding behavior is unchanged.
func TestPolicy_AllModeUnaffectedByStrictRevocation(t *testing.T) {
	finding := vioSrc("code_check/no-eval", "pkg/x/x.go", "r1", "test/pack")
	baseline := &BaselineArtifact{SchemaVersion: BaselineSchemaV1, Violations: []Violation{finding}}

	scope := newGateScope("", GateScopeModeAll, nil, nil)
	cmp := CompareBaseline([]Violation{finding}, baseline, BaselineCompareOptions{Scope: scope})
	if len(cmp.NewViolations) != 0 {
		t.Fatalf("--all mode must leave grandfathering intact, got %d: %#v", len(cmp.NewViolations), cmp.NewViolations)
	}
}

// TestPolicy_RevokedFindingClearsWhenFixedOrSuppressed (CLM-005): a revoked
// (touched-file, baseline-present) finding that is ABSENT from the current slice —
// because it was fixed, or waived/engine-suppressed and subtracted upstream by
// computeWaiverResult before comparison — is no longer NEW. Absent-from-current is
// the shared mechanism for both fix and waive, so the step returns to pass.
func TestPolicy_RevokedFindingClearsWhenFixedOrSuppressed(t *testing.T) {
	finding := vioSrc("code_check/no-eval", "pkg/x/x.go", "r1", "test/pack")
	baseline := &BaselineArtifact{SchemaVersion: BaselineSchemaV1, Violations: []Violation{finding}}

	scope := newGateScope("", GateScopeModeDiff, []string{"pkg/x/x.go"}, nil)
	// current omits the finding (fixed or suppressed/waived upstream).
	cmp := CompareBaseline([]Violation{}, baseline, BaselineCompareOptions{Scope: scope})
	if len(cmp.NewViolations) != 0 {
		t.Fatalf("a cleared (absent-from-current) revoked finding must not be NEW, got %d: %#v", len(cmp.NewViolations), cmp.NewViolations)
	}
	// On a touched file it registers as FIXED (paid down), which is the clean-up
	// signal — the point is it is no longer blocking.
	if len(cmp.FixedViolations) != 1 {
		t.Fatalf("expected the cleared finding on the touched file to be reported fixed, got %d: %#v", len(cmp.FixedViolations), cmp.FixedViolations)
	}
}

// TestPolicy_TouchedRevocation_MatchesNormalizedPath (CLM-006, S1): the marquee
// anti-vacuous-green pin. The scope is built via the real newGateScope path with a
// non-empty projectRoot; the baseline-present Violation carries a NON-canonical
// (projectRoot-joined absolute) File form. EnrichViolationIdentity normalizes with
// projectRoot=="" while scope.Contains normalizes with the REAL ProjectRoot, so
// this exercises the path-format seam directly. S1: FIRST assert the finding IS
// baseline-present (so the "already net-new" short-circuit is NOT what makes it
// count and the scope-touch path is genuinely exercised), THEN assert
// scope.Contains(v.File) is true and CompareBaseline reports it NEW.
func TestPolicy_TouchedRevocation_MatchesNormalizedPath(t *testing.T) {
	projectRoot := "/repo/root"
	// Non-canonical form of the touched file "pkg/x/x.go": the projectRoot-joined
	// absolute path an engine may emit under a full-scope walk.
	noncanon := vioSrc("code_check/no-eval", "/repo/root/pkg/x/x.go", "r1", "test/pack")
	baseline := &BaselineArtifact{SchemaVersion: BaselineSchemaV1, Violations: []Violation{noncanon}}
	scope := newGateScope(projectRoot, GateScopeModeDiff, []string{"pkg/x/x.go"}, nil)

	// S1a: the finding is baseline-present (identity is IN the baseline set), so it
	// is NOT independently net-new — the scope-touch path is what must reclassify it.
	enriched := EnrichViolationIdentity(noncanon)
	present := false
	for _, b := range baseline.Violations {
		if EnrichViolationIdentity(b).IdentityHash == enriched.IdentityHash {
			present = true
			break
		}
	}
	if !present {
		t.Fatalf("fixture invalid: finding must be baseline-present so the test is non-vacuous")
	}

	// S1b: scope must recognize the non-canonical path via its real-ProjectRoot
	// normalization — otherwise the touch-check silently no-ops (vacuous green).
	if !scope.Contains(noncanon.File) {
		t.Fatalf("scope.Contains must match the non-canonical path form %q under projectRoot %q", noncanon.File, projectRoot)
	}

	cmp := CompareBaseline([]Violation{noncanon}, baseline, BaselineCompareOptions{Scope: scope})
	if len(cmp.NewViolations) != 1 {
		t.Fatalf("baseline-present finding on the touched file (non-canonical path) must be revoked as NEW, got %d: %#v", len(cmp.NewViolations), cmp.NewViolations)
	}
}

// TestPolicy_NoPolicyBaselineComparisonStepFiresRatchet (CLM-007): the
// consolidation payoff. With a baseline present but NO enforcement.policy, the
// aggregate baseline_comparison step (computeBaselineResult) is the SOLE
// grandfathering path — ApplyPolicy early-returns on empty policy. Drive the full
// Run and assert the baseline_comparison step FAILS on a touched-file baselined
// finding, proving the ratchet is not left dead by the empty-policy early-return.
func TestPolicy_NoPolicyBaselineComparisonStepFiresRatchet(t *testing.T) {
	finding := vioSrc("code_check/no-eval", "pkg/x/x.go", "r1", "test/pack")

	result, exitCode := New(
		WithSteps([]StepFunc{
			func(_ context.Context) StepResult {
				return StepResult{StepName: StepCodeCheck, Status: "pass", Violations: []Violation{finding}}
			},
			StepBaselineComparisonFunc(),
			StepWaiverResolutionFunc(),
			StepLedgerIntegrityFunc(),
		}),
		WithScope(newGateScope("", GateScopeModeDiff, []string{"pkg/x/x.go"}, nil)),
		WithBaseline(&BaselineArtifact{SchemaVersion: BaselineSchemaV1, Violations: []Violation{finding}}),
		// NO WithPolicy — empty policy map, so the aggregate step is the only ratchet.
	).Run(context.Background())

	var baselineStep *StepResult
	for i := range result.Steps {
		if result.Steps[i].StepName == StepBaselineComparison {
			baselineStep = &result.Steps[i]
		}
	}
	if baselineStep == nil {
		t.Fatalf("expected a baseline_comparison step in results, got %#v", result.Steps)
	}
	if baselineStep.Status != "fail" {
		t.Fatalf("no-policy aggregate baseline_comparison must FIRE the ratchet (fail) on a touched-file baselined finding, got %q", baselineStep.Status)
	}
	if len(baselineStep.NewViolations) != 1 {
		t.Fatalf("expected exactly the revoked finding as new, got %d: %#v", len(baselineStep.NewViolations), baselineStep.NewViolations)
	}
	if exitCode != 1 {
		t.Fatalf("expected gate exit 1 when the ratchet fires, got %d", exitCode)
	}
}

// TestPolicy_NilScopeKeepsGrandfathering (CLM-007, S2): a baseline-present finding
// compared with a nil scope stays grandfathered (0 new) and does not panic — the
// positive nil-scope guard on scopeTouches.
func TestPolicy_NilScopeKeepsGrandfathering(t *testing.T) {
	finding := vioSrc("code_check/no-eval", "pkg/x/x.go", "r1", "test/pack")
	baseline := &BaselineArtifact{SchemaVersion: BaselineSchemaV1, Violations: []Violation{finding}}

	cmp := CompareBaseline([]Violation{finding}, baseline, BaselineCompareOptions{Scope: nil})
	if len(cmp.NewViolations) != 0 {
		t.Fatalf("nil scope must keep grandfathering (no revocation, no panic), got %d: %#v", len(cmp.NewViolations), cmp.NewViolations)
	}
}
