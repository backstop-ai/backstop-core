package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/check"
	"github.com/backstop-ai/backstop-core/pkg/gate"
	"github.com/backstop-ai/backstop-core/pkg/pack"
	"github.com/backstop-ai/backstop-core/pkg/pack/engine"
	"github.com/backstop-ai/backstop-core/pkg/packval"
)

// ISSUE-209 FAIL-WHOLE GUARDS. The dispatcher now returns the findings it collected
// alongside an engine error. Two callers must keep treating any error as a whole
// failure and must NEVER read that partial stream: the substantiveness step (a
// partial stream would manufacture hollow and noTarget verdicts) and the
// contract-entry probe (an engine-error violation in the stream would read as a
// MATCH). These are controls: they pass before and after the dispatcher change.

// failWholeTestEngineError is the D-1 engine-error violation shape.
func failWholeTestEngineError(msg string) gate.Violation {
	return gate.Violation{Rule: gate.StepPackEngines, Severity: "error", File: "", ProjectWide: true, Message: msg}
}

// TestSubstantivenessStep_DispatchErrorIgnoresPartialStream (CLM-006): with the seam
// returning a hollow-role finding on the mandated test file (and NO extraction
// findings) plus an engine-error violation and a non-nil error, the step emits
// exactly one config-error violation. Had it read the stream it would have emitted
// a hollow test_substantiveness violation and a noTarget violation for the spec's
// mandated test.
func TestSubstantivenessStep_DispatchErrorIgnoresPartialStream(t *testing.T) {
	specDir := t.TempDir()
	codeDir := t.TempDir()
	writeSubstantivenessSpec(t, specDir)
	injectSubstantivenessManifest(t)
	// The mandated test file EXISTS, so a step that read the stream would make it
	// join-eligible and raise a noTarget violation (nothing populates the
	// referenced set) on top of the hollow conversion.
	testPath := writeMandatedTestFile(t, codeDir, "subject_test.go", "TestSubjectHollow", hollowBody)

	engineMsg := `dispatching findings engine "ast-grep" for pack backstop/substantiveness: BROKE`
	orig := dispatchPackEnginesFn
	t.Cleanup(func() { dispatchPackEnginesFn = orig })
	dispatchPackEnginesFn = func([]*pack.Manifest, string, string, *gate.GateScope, check.CommandRunner) ([]gate.Violation, error) {
		return []gate.Violation{
			warningSeverityHollowSeamFinding(testPath, "error"),
			failWholeTestEngineError(engineMsg),
		}, errors.New(engineMsg)
	}

	classifier, matcher := goSubstDiscovery(t)
	res := buildTestSubstantivenessStep(specDir, codeDir, codeDir, nil, classifier, matcher)(context.Background())

	if len(res.Violations) != 1 {
		t.Fatalf("a dispatch error must yield exactly ONE config-error violation (no hollow, no noTarget), got %d: %#v", len(res.Violations), res.Violations)
	}
	v := res.Violations[0]
	if v.Rule != gate.StepTestSubstantiveness {
		t.Errorf("violation rule = %q, want %q", v.Rule, gate.StepTestSubstantiveness)
	}
	if !strings.HasPrefix(v.Message, "dispatching substantiveness pack: ") {
		t.Errorf("violation message must be prefixed %q, got %q", "dispatching substantiveness pack: ", v.Message)
	}
	if v.Severity != "error" {
		t.Errorf("violation severity = %q, want error", v.Severity)
	}
	if res.Status != "fail" || !res.ConfigErr {
		t.Errorf("the step must be fail + ConfigErr, got status=%q configErr=%v", res.Status, res.ConfigErr)
	}
}

// TestDispatchContractEntry_EngineErrorFailsWhole (CLM-007): a contract-entry probe
// whose engine fails inside the dispatcher (here at the trusted-tool allowlist, via
// a pack-declared "grep" binding provisioning a tool not on the allowlist) returns a
// non-nil error and the ZERO ContractEngineResult — the engine-error violation the
// dispatcher now returns is never read as a match.
func TestDispatchContractEntry_EngineErrorFailsWhole(t *testing.T) {
	projectRoot := t.TempDir()
	target := filepath.Join(projectRoot, "subject.go")
	if err := os.WriteFile(target, []byte("package subject\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest := &pack.Manifest{
		Name:           "acme/contracts",
		NormalizedName: "acme/contracts",
		Engines: map[string]pack.EngineSpec{
			"grep": {Binding: engine.EngineBinding{
				Command:   "grep -rn -H -I",
				InputMode: engine.InputModePatternArg,
				InputFlag: "-e",
				ScopeKind: engine.ScopeKindFileArgs,
				Category:  engine.EngineCategoryOpinion,
				GateType:  engine.GateTypeContracts,
				Provision: &engine.Provision{Tool: "backstop-untrusted-tool-209", Version: "1.0.0"},
			}},
		},
	}
	entry := gate.ContractEntry{File: target, Name: "ForbiddenSymbol", Absent: true}

	got, err := dispatchContractEntryWithSandbox(projectRoot, manifest, entry, &recordingSandboxRunner{mode: packval.SandboxModeNative})
	if err == nil {
		t.Fatalf("an engine error must fail the contract entry whole, got nil with %#v", got)
	}
	if !strings.Contains(err.Error(), "dispatching contract engine for") {
		t.Errorf("the error must come from the dispatcher branch, got: %v", err)
	}
	if !strings.Contains(err.Error(), "trust gate") {
		t.Errorf("the fixture must fail at the in-dispatcher trust gate, got: %v", err)
	}
	if got.Matched || got.Scanned || got.Entry != (gate.ContractEntry{}) || got.Locations != nil {
		t.Errorf("a failed probe must return the ZERO ContractEngineResult (never a match), got %#v", got)
	}
}
