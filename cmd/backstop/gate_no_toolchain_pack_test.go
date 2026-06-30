package main

import (
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/gate"
	"github.com/bmanson/backstop-core/pkg/pack"
)

// toolchainPackFixture loads a real toolchain pack manifest for the "packs ran"
// comparison cases.
func toolchainPackFixture(t *testing.T) *pack.Manifest {
	t.Helper()
	return goToolchainManifest(t)
}

// noToolchainResult returns the no-toolchain-pack StepResult for an empty declared
// toolchain set (0 declared toolchain packs). SPEC-046: toolchainEnforcementStatus
// takes a SINGLE declared-pack argument — a toolchain is an ordinary declared pack,
// so there is no `bridged` input.
func noToolchainResult(t *testing.T) gate.StepResult {
	t.Helper()
	res, emitted := toolchainEnforcementStatus(nil)
	if !emitted {
		t.Fatal("0 toolchain packs must emit a no-toolchain-pack StepResult, not nothing")
	}
	return res
}

// TestNoToolchainPack_EnforcementStatusRewrittenToDeclaredOnlyArg (CLM-024): the
// no-toolchain-pack WARN state keys on the SINGLE declared-pack argument after the
// bridge deletion — toolchainEnforcementStatus(declared) takes NO `bridged`
// parameter. Zero declared toolchain packs emit the warn; a declared toolchain pack
// suppresses it. (The 1-argument signature is itself compile-enforced here, so no
// test can pin the removed `bridged` parameter.)
func TestNoToolchainPack_EnforcementStatusRewrittenToDeclaredOnlyArg(t *testing.T) {
	// Zero declared toolchain packs -> warn emitted (declared-only arg).
	res, emitted := toolchainEnforcementStatus(nil)
	if !emitted {
		t.Fatal("zero DECLARED toolchain packs must emit the no-toolchain-pack warn state via the 1-arg signature")
	}
	if res.Status != "warning" {
		t.Fatalf("the no-toolchain-pack state must be a non-failing warning, got %q", res.Status)
	}
	// A declared toolchain pack suppresses the warn — proving the helper keys on the
	// declared set alone.
	if _, emitted := toolchainEnforcementStatus([]*pack.Manifest{toolchainPackFixture(t)}); emitted {
		t.Fatal("a DECLARED toolchain pack must suppress the no-toolchain-pack warn state (declared-only keying)")
	}
}

// TestNoToolchainPack_DoesNotBlockGatePasses proves 0 toolchain packs does NOT
// block — folded into a GateResult, Pass stays true and the exit code stays 0
// (CLM-016).
func TestNoToolchainPack_DoesNotBlockGatePasses(t *testing.T) {
	res := noToolchainResult(t)
	result := gate.NewGateResult([]gate.StepResult{res})
	if !result.Pass {
		t.Fatal("0 toolchain packs must NOT flip gate.Pass — enforcement is warn-only opt-in")
	}
	if result.StepsFailed != 0 {
		t.Fatalf("0 toolchain packs must not count as a failed step, got %d failed", result.StepsFailed)
	}
}

// TestNoToolchainPack_UsesWarningStatusNotFail proves the state uses the
// non-failing "warning" step status (counted in StepsWarned), not "fail"
// (CLM-017).
func TestNoToolchainPack_UsesWarningStatusNotFail(t *testing.T) {
	res := noToolchainResult(t)
	if res.Status != "warning" {
		t.Fatalf("no-toolchain-pack status must be the non-failing \"warning\", got %q", res.Status)
	}
	result := gate.NewGateResult([]gate.StepResult{res})
	if result.StepsWarned != 1 {
		t.Fatalf("the no-toolchain-pack warning must be counted in StepsWarned, got %d", result.StepsWarned)
	}
}

// TestNoToolchainPack_NotSilentPass proves the state is NOT a silent normal
// "pass" — it is distinct from a toolchain-packs-ran-and-passed run (CLM-018).
func TestNoToolchainPack_NotSilentPass(t *testing.T) {
	res := noToolchainResult(t)
	if res.Status == "pass" {
		t.Fatal("no-toolchain-pack state must NOT render as a normal green pass")
	}
	// A toolchain-packs-present run does NOT emit the warn state.
	_, emitted := toolchainEnforcementStatus([]*pack.Manifest{toolchainPackFixture(t)})
	if emitted {
		t.Fatal("a run WITH a toolchain pack must NOT emit the no-toolchain-pack warning")
	}
}

// TestNoEnforcement_LoudMessageOnHumanReport proves a 0-pack run renders a
// stable, recognizable "enforcement not configured (0 toolchain packs)" message
// on the human report surface (CLM-019).
func TestNoEnforcement_LoudMessageOnHumanReport(t *testing.T) {
	res := noToolchainResult(t)
	result := gate.NewGateResult([]gate.StepResult{res})
	human := gate.FormatHuman(result, true)
	if !strings.Contains(human, "enforcement not configured (0 toolchain packs)") {
		t.Fatalf("human report must carry the stable loud message; got:\n%s", human)
	}
}

// TestNoEnforcement_ReflectedInMachineSummary proves the state is reflected in
// the machine-readable summary (StepsWarned), distinct from steps-passed, so a
// CI consumer detects it despite exit 0 (CLM-020).
func TestNoEnforcement_ReflectedInMachineSummary(t *testing.T) {
	res := noToolchainResult(t)
	result := gate.NewGateResult([]gate.StepResult{res})
	if result.StepsWarned != 1 {
		t.Fatalf("machine summary must reflect the warn state in StepsWarned, got %d", result.StepsWarned)
	}
	if result.StepsPassed != 0 {
		t.Fatalf("the no-toolchain-pack state must NOT be counted as a passed step, got %d passed", result.StepsPassed)
	}
}

// TestNoEnforcement_DistinctFromToolchainPacksPassedRun proves a 0-pack run is
// visibly distinguishable in report output from a toolchain-packs-ran-and-passed
// run — NOT identical green (CLM-021).
func TestNoEnforcement_DistinctFromToolchainPacksPassedRun(t *testing.T) {
	warn := noToolchainResult(t)
	warnReport := gate.FormatHuman(gate.NewGateResult([]gate.StepResult{warn}), true)

	// A toolchain-packs-ran-and-passed run: a normal green pass step on the same
	// step name, no warning.
	passed := gate.StepResult{StepName: warn.StepName, Status: "pass", Violations: []gate.Violation{}}
	passReport := gate.FormatHuman(gate.NewGateResult([]gate.StepResult{passed}), true)

	if warnReport == passReport {
		t.Fatal("a 0-toolchain-pack run must be visibly distinguishable from a packs-passed run — they produced identical report output")
	}
	if strings.Contains(passReport, "enforcement not configured") {
		t.Fatal("a packs-passed run must NOT carry the no-enforcement message")
	}
}

// TestNoEnforcement_NeverVacuousGreen proves the loud state never collapses into
// a normal green — 0 packs can never produce output indistinguishable from a
// fully-enforced green pass (CLM-022).
func TestNoEnforcement_NeverVacuousGreen(t *testing.T) {
	warn := noToolchainResult(t)
	if warn.Status == "pass" {
		t.Fatal("the no-enforcement state must never be a normal green pass status")
	}
	report := gate.FormatHuman(gate.NewGateResult([]gate.StepResult{warn}), true)
	// The summary line must show a warned count, and the loud message must be
	// present — both impossible to confuse with a fully-enforced green pass.
	if !strings.Contains(report, "1 warned") {
		t.Fatalf("warned count must surface in the summary line; got:\n%s", report)
	}
	if !strings.Contains(report, "enforcement not configured (0 toolchain packs)") {
		t.Fatalf("the loud message must be present so 0-packs never reads as fully-enforced green; got:\n%s", report)
	}
}
