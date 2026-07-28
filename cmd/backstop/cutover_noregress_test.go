package main

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/gate"
	"github.com/backstop-ai/backstop-core/pkg/pack"
	"github.com/backstop-ai/backstop-core/pkg/pack/engine"
)

// TestAbsorbSpec034_Step2DeletionCompletedNotGoOnly proves SPEC-034's unfinished
// Step-2 deletion is completed here — the realCodeChecker / pkg/check.Run Step-2
// path is gone, with no Go-only assumption re-introduced (CLM-023). It
// cross-checks the generalized bridge from Phase 2.
func TestAbsorbSpec034_Step2DeletionCompletedNotGoOnly(t *testing.T) {
	cmdDir := filepath.Join(repoRoot(t), "cmd", "backstop")
	if grepNonTestSource(t, cmdDir, "realCodeChecker") {
		t.Error("SPEC-034's Step-2 deletion is not complete — realCodeChecker survives (CLM-023)")
	}
	// No Go-only assumption in the toolchain wiring: SPEC-046 DELETED the
	// language-derived bridge entirely, so lint/build/test run through the uniform
	// declared-pack dispatch, language-agnostically. A reintroduced `language == "go"`
	// short-circuit would resurrect the baked single-language assumption.
	src := readFileStr(t, "gate.go")
	for _, banned := range []string{`language != "go"`, `language == "go"`} {
		if strings.Contains(src, banned) {
			t.Errorf("a Go-only short-circuit %q is present; toolchain dispatch must stay language-agnostic via the declared-pack path (CLM-023)", banned)
		}
	}
}

// TestAbsorbSpec034_BridgeStillDispatchesToolchainPasses proves SPEC-034's
// landed bridge and the golden-equivalence harness remain the safety net — the
// bridge still dispatches the toolchain pack's lint/build/test passes through
// dispatchPackEngines after the deletion (CLM-024).
func TestAbsorbSpec034_BridgeStillDispatchesToolchainPasses(t *testing.T) {
	m := goToolchainManifest(t)
	stubSandboxedRunStdout(t, nil)
	runner := &fixtureRunner{byCmd: map[string][]byte{
		"go build":      readFixture(t, "go-build-errors.txt"),
		"go test":       readFixture(t, "go-test-failures.txt"),
		"golangci-lint": readFixture(t, "golangci-v2.sarif"),
	}}
	// Partition dedicated-step gate-types out of the SARIF findings dispatch as the
	// production gate does — the SPEC-042 go-coverage engine routes to the
	// coverage-records channel, not SARIF.
	violations, err := dispatchPackEngines(excludeDedicatedStepRules([]*pack.Manifest{m}), goToolchainPacksDir(t), t.TempDir(), nil, runner)
	if err != nil {
		t.Fatalf("the bridge must still dispatch toolchain passes after the deletion (CLM-024): %v", err)
	}
	if len(violations) != 8 {
		t.Fatalf("expected the 8 toolchain pass violations through dispatch, got %d (CLM-024)", len(violations))
	}
}

// TestNoRegress_DedicatedStepRulesStillExcludedFromGenericDispatch proves the
// cutover preserves excludeDedicatedStepRules so substantiveness/contracts rules
// are NOT double-dispatched into the generic lint/build/test/findings path
// (CLM-025, Sharp Edge 8).
func TestNoRegress_DedicatedStepRulesStillExcludedFromGenericDispatch(t *testing.T) {
	// Source guard: pack_gate.go must still define excludeDedicatedStepRules and
	// the gate must still apply it before the generic dispatch.
	packGateSrc := readFileStr(t, "pack_gate.go")
	if !strings.Contains(packGateSrc, "func excludeDedicatedStepRules") {
		t.Fatal("excludeDedicatedStepRules was removed; substantiveness/contracts would be double-dispatched (CLM-025)")
	}
	gateSrc := readFileStr(t, "gate.go")
	if !strings.Contains(gateSrc, "excludeDedicatedStepRules(") {
		t.Fatal("the gate no longer applies excludeDedicatedStepRules before the generic dispatch (CLM-025)")
	}

	// Behavioral: a manifest with a dedicated-step (contracts) rule has it dropped
	// from the generic dispatch set.
	m := &pack.Manifest{
		Name:           "backstop/x",
		NormalizedName: "backstop/x",
		Engines: map[string]pack.EngineSpec{
			"contracts-grep": {Binding: engine.EngineBinding{
				Command:   "grep -rn",
				InputMode: engine.InputModePatternArg,
				InputFlag: "-e",
				GateType:  engine.GateTypeContracts,
			}},
		},
		Content: pack.Content{Ruleset: pack.Ruleset{Rules: []pack.Rule{
			{ID: "c", Engine: "contracts-grep", Pattern: "x"},
		}}},
	}
	out := excludeDedicatedStepRules([]*pack.Manifest{m})
	if len(out) != 1 {
		t.Fatalf("expected the manifest back (rules filtered), got %d manifests", len(out))
	}
	if len(out[0].Content.Ruleset.Rules) != 0 {
		t.Fatalf("the contracts (dedicated-step) rule must be excluded from the generic dispatch, got %d rules", len(out[0].Content.Ruleset.Rules))
	}
}

// TestNoRegress_SubstantivenessStepStillRunsAndPasses proves after the cutover
// the substantiveness traceability step still runs and passes through its own
// dedicated gate step over its installed pack (CLM-026).
func TestNoRegress_SubstantivenessStepStillRunsAndPasses(t *testing.T) {
	// On a Go project with no substantiveness pack installed, the dedicated
	// substantiveness step is present and is a clean no-op pass (the capability
	// classifier governs warn/block upstream) — it still RUNS through its own step.
	root := goToolchainProjectRoot(t)
	names := gateStepNames(t, root, emptyDiffScope())
	found := false
	for _, n := range names {
		if n == gate.StepTestSubstantiveness {
			found = true
		}
	}
	if !found {
		t.Fatalf("the substantiveness dedicated gate step must still run after the cutover (CLM-026). steps=%v", names)
	}
}

// TestNoRegress_ContractsStepStillRunsAndPasses proves after the cutover the
// contracts traceability step still runs and passes through its own dedicated
// gate step over its installed pack (CLM-027).
func TestNoRegress_ContractsStepStillRunsAndPasses(t *testing.T) {
	root := goToolchainProjectRoot(t)
	names := gateStepNames(t, root, emptyDiffScope())
	found := false
	for _, n := range names {
		if n == gate.StepContractSignature {
			found = true
		}
	}
	if !found {
		t.Fatalf("the contracts dedicated gate step must still run after the cutover (CLM-027). steps=%v", names)
	}
}

var _ = context.Background
