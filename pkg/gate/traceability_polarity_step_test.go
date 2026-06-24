package gate

import (
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/config"
)

// goCfgDeclaring builds a config (language go) that declares dim with the given
// waived flag, for PolarityStepResult message/waive tests.
func cfgDeclaring(language string, dim TraceabilityDimension, waived bool) *config.Config {
	return &config.Config{
		Project:  "p",
		Language: language,
		Enforcement: config.Enforcement{
			Toolchain: map[string]config.ToolchainPass{
				"test": {Command: "cmd", Format: "fmt", GateType: string(dim), Waived: waived},
			},
		},
	}
}

// TestBrokenDeclared_NeverDowngradedToWarn (CLM-006/007/008 mapping + CLM-009):
// a class-1 broken-declared result sets ConfigErr=true with status "fail" — it
// blocks at exit 2 and is NEVER downgraded to a warning / exit 0, for every
// class-1 trigger (command error, unparseable output, unknown key).
func TestBrokenDeclared_NeverDowngradedToWarn(t *testing.T) {
	cfg := cfgDeclaring("go", DimensionSubstantiveness, false)
	cap := CapabilityState{Present: true, Working: false, PackOrCommand: "go test ./...", Detail: "exit status 1"}
	res := PolarityStepResult(StepTestSubstantiveness, DimensionSubstantiveness, ClassBrokenDeclared, cfg, cap)

	if res.Status != "fail" {
		t.Errorf("class 1 status = %q, want fail", res.Status)
	}
	if !res.ConfigErr {
		t.Error("class 1 must set ConfigErr (block exit 2)")
	}
	if res.Status == "warning" {
		t.Error("class 1 must never be a warning")
	}
}

// TestCapabilityAbsent_StatusNotFail_NoExitContribution (CLM-010/011): a class-2
// capability-absent result emits a "warning" status with ConfigErr=false — it
// passes (exit 0) and contributes nothing to a non-zero exit code, and its
// violation is severity-tagged "warning".
func TestCapabilityAbsent_StatusNotFail_NoExitContribution(t *testing.T) {
	cfg := cfgDeclaring("typescript", DimensionContracts, false) // declares contracts, but we classify coverage (undeclared)
	cap := CapabilityState{Present: false, PackOrCommand: "a typescript coverage pack"}
	res := PolarityStepResult(StepCoverageThreshold, DimensionCoverage, ClassCapabilityAbsent, cfg, cap)

	if res.Status != "warning" {
		t.Errorf("class 2 status = %q, want warning", res.Status)
	}
	if res.Status == "fail" {
		t.Error("class 2 status must not be fail")
	}
	if res.ConfigErr {
		t.Error("class 2 must NOT set ConfigErr (no exit contribution)")
	}
	if len(res.Violations) == 0 || res.Violations[0].Severity != "warning" {
		t.Errorf("class 2 advisory must be tagged severity warning, got %#v", res.Violations)
	}
}

// TestCapabilityAbsent_NeverAutoPromotesToBlocking (CLM-012): repeated
// classification + mapping of the same undeclared dimension stays class-2
// warn-and-pass — there is no escalation knob that flips it to blocking.
func TestCapabilityAbsent_NeverAutoPromotesToBlocking(t *testing.T) {
	cfg := loadPolarityFixture(t, "undeclared-coverage.yml")
	cap := CapabilityState{Present: false}
	for i := 0; i < 5; i++ {
		class := ClassifyDimension(cfg, DimensionCoverage, cap)
		if class != ClassCapabilityAbsent {
			t.Fatalf("run %d: class = %v, want ClassCapabilityAbsent (no auto-promotion)", i, class)
		}
		res := PolarityStepResult(StepCoverageThreshold, DimensionCoverage, class, cfg, cap)
		if res.Status != "warning" || res.ConfigErr {
			t.Fatalf("run %d: result must stay warn-and-pass, got status=%q configErr=%v", i, res.Status, res.ConfigErr)
		}
	}
}

// TestDeclaredIntentUnmet_NotDowngradedToWarnAndPass (CLM-014): a class-3
// declared-intent-unmet result sets ConfigErr=true (status fail) — it blocks at
// exit 2 and is NOT downgraded to a class-2 warn-and-pass.
func TestDeclaredIntentUnmet_NotDowngradedToWarnAndPass(t *testing.T) {
	cfg := cfgDeclaring("go", DimensionCoverage, false)
	cap := CapabilityState{Present: false, PackOrCommand: "the go coverage analyzer"}
	res := PolarityStepResult(StepCoverageThreshold, DimensionCoverage, ClassDeclaredIntentUnmet, cfg, cap)

	if res.Status != "fail" {
		t.Errorf("class 3 status = %q, want fail", res.Status)
	}
	if !res.ConfigErr {
		t.Error("class 3 must set ConfigErr (block exit 2)")
	}
	if res.Status == "warning" {
		t.Error("class 3 must not be downgraded to a warning")
	}
}

// TestCapabilityAbsent_Undeclared_WarnsAndPassesExit0 (CLM-010): an undeclared,
// capability-absent dimension classifies CAPABILITY-ABSENT, emits a report-surface
// advisory, and the wrapping step passes (exit 0, ConfigErr false) — the loudness
// lives on the report surface, never the exit code.
func TestCapabilityAbsent_Undeclared_WarnsAndPassesExit0(t *testing.T) {
	cfg := loadPolarityFixture(t, "undeclared-coverage.yml")
	cap := CapabilityState{Present: false, PackOrCommand: "a coverage pack the project hasn't pulled"}

	class := ClassifyDimension(cfg, DimensionCoverage, cap)
	if class != ClassCapabilityAbsent {
		t.Fatalf("undeclared + absent: class = %v, want ClassCapabilityAbsent", class)
	}
	res := PolarityStepResult(StepCoverageThreshold, DimensionCoverage, class, cfg, cap)
	if res.Status != "warning" {
		t.Errorf("class 2 status = %q, want warning (advisory on the report surface)", res.Status)
	}
	if res.ConfigErr {
		t.Error("class 2 must pass exit 0 — ConfigErr must be false (undeclared is not yet a broken promise)")
	}
	if len(res.Violations) == 0 {
		t.Error("class 2 must emit a conspicuous report-surface advisory, not a silent pass")
	}
}

// TestDeclaredIntentUnmet_MissingCapability_BlocksExit2 (CLM-013): a dimension
// that IS declared but whose required capability is missing classifies
// DECLARED-INTENT-UNMET and blocks at exit 2 (ConfigErr true) — a broken promise,
// not a warn-and-pass.
func TestDeclaredIntentUnmet_MissingCapability_BlocksExit2(t *testing.T) {
	cfg := cfgDeclaring("go", DimensionCoverage, false)
	cap := CapabilityState{Present: false, PackOrCommand: "the declared coverage capability"}

	class := ClassifyDimension(cfg, DimensionCoverage, cap)
	if class != ClassDeclaredIntentUnmet {
		t.Fatalf("declared + missing capability: class = %v, want ClassDeclaredIntentUnmet", class)
	}
	res := PolarityStepResult(StepCoverageThreshold, DimensionCoverage, class, cfg, cap)
	if res.Status != "fail" {
		t.Errorf("class 3 status = %q, want fail", res.Status)
	}
	if !res.ConfigErr {
		t.Error("class 3 must block exit 2 — ConfigErr must be true (declared-but-unmet is a broken promise)")
	}
}

// TestWaive_SuppressesClass2Advisory_StillPasses (CLM-022): a dimension marked
// waived has its class-2 advisory suppressed to a plain pass and the gate still
// passes (exit 0).
func TestWaive_SuppressesClass2Advisory_StillPasses(t *testing.T) {
	cfg := cfgDeclaring("typescript", DimensionCoverage, true) // coverage waived
	cap := CapabilityState{Present: false, PackOrCommand: "a typescript coverage pack"}
	res := PolarityStepResult(StepCoverageThreshold, DimensionCoverage, ClassCapabilityAbsent, cfg, cap)

	if res.Status != "pass" {
		t.Errorf("waived class 2 status = %q, want pass (advisory suppressed)", res.Status)
	}
	if res.ConfigErr {
		t.Error("waived class 2 must not set ConfigErr")
	}
	// Suppressed: no warning-severity advisory surfaced.
	for _, v := range res.Violations {
		if v.Severity == "warning" {
			t.Errorf("waived class 2 must suppress the warning advisory, got %#v", res.Violations)
		}
	}
}

// TestWaive_DoesNotSilenceClass1BrokenDeclared (CLM-023): a waive on a dimension
// does NOT silence a class-1 broken-declared failure — it still blocks at exit 2.
func TestWaive_DoesNotSilenceClass1BrokenDeclared(t *testing.T) {
	cfg := cfgDeclaring("go", DimensionSubstantiveness, true) // waived, but broken
	cap := CapabilityState{Present: true, Working: false, Detail: "exit status 1"}
	res := PolarityStepResult(StepTestSubstantiveness, DimensionSubstantiveness, ClassBrokenDeclared, cfg, cap)

	if res.Status != "fail" || !res.ConfigErr {
		t.Errorf("a waive must NOT silence class 1; got status=%q configErr=%v", res.Status, res.ConfigErr)
	}
}

// TestWaive_DoesNotSilenceClass3DeclaredIntentUnmet (CLM-024): a waive on a
// dimension does NOT silence a class-3 declared-intent-unmet failure — it still
// blocks at exit 2.
func TestWaive_DoesNotSilenceClass3DeclaredIntentUnmet(t *testing.T) {
	cfg := cfgDeclaring("go", DimensionCoverage, true) // waived, but declared+missing
	cap := CapabilityState{Present: false}
	res := PolarityStepResult(StepCoverageThreshold, DimensionCoverage, ClassDeclaredIntentUnmet, cfg, cap)

	if res.Status != "fail" || !res.ConfigErr {
		t.Errorf("a waive must NOT silence class 3; got status=%q configErr=%v", res.Status, res.ConfigErr)
	}
}

// TestMessage_Class2_NamesDimensionStackPackAndDeclareOrWaive (CLM-018): a
// class-2 advisory message names the dimension, the project's stack/language,
// the exact pack/command to adopt, and the declare-or-waive next step.
func TestMessage_Class2_NamesDimensionStackPackAndDeclareOrWaive(t *testing.T) {
	cfg := cfgDeclaring("typescript", DimensionContracts, false)
	cap := CapabilityState{Present: false, PackOrCommand: "a typescript contracts pack"}
	res := PolarityStepResult(StepContractSignature, DimensionContracts, ClassCapabilityAbsent, cfg, cap)
	if len(res.Violations) == 0 {
		t.Fatal("class 2 must carry an advisory violation")
	}
	msg := res.Violations[0].Message
	for _, want := range []string{"contracts", "typescript", "a typescript contracts pack", "declare", "waive"} {
		if !strings.Contains(msg, want) {
			t.Errorf("class 2 message missing %q; got: %s", want, msg)
		}
	}
}

// TestMessage_Class1_CarriesExpectedVsGot (CLM-019): a class-1 broken-declared
// command/format error message carries expected-vs-got detail (the declared
// command/format and the observed failure).
func TestMessage_Class1_CarriesExpectedVsGot(t *testing.T) {
	cfg := cfgDeclaring("go", DimensionSubstantiveness, false)
	cap := CapabilityState{Present: true, Working: false, PackOrCommand: "go test ./...", Detail: "exit status 1: panic"}
	res := PolarityStepResult(StepTestSubstantiveness, DimensionSubstantiveness, ClassBrokenDeclared, cfg, cap)
	if len(res.Violations) == 0 {
		t.Fatal("class 1 must carry a violation")
	}
	msg := res.Violations[0].Message
	for _, want := range []string{"substantiveness", "go test ./...", "exit status 1: panic"} {
		if !strings.Contains(msg, want) {
			t.Errorf("class 1 message missing expected-vs-got token %q; got: %s", want, msg)
		}
	}
}

// TestMessage_Class3_NamesMissingCapabilityAndFix (CLM-020): a class-3
// declared-intent-unmet message names the declared dimension and the specific
// missing capability plus the fix.
func TestMessage_Class3_NamesMissingCapabilityAndFix(t *testing.T) {
	cfg := cfgDeclaring("go", DimensionCoverage, false)
	cap := CapabilityState{Present: false, PackOrCommand: "the go coverage analyzer"}
	res := PolarityStepResult(StepCoverageThreshold, DimensionCoverage, ClassDeclaredIntentUnmet, cfg, cap)
	if len(res.Violations) == 0 {
		t.Fatal("class 3 must carry a violation")
	}
	msg := res.Violations[0].Message
	for _, want := range []string{"coverage", "the go coverage analyzer"} {
		if !strings.Contains(msg, want) {
			t.Errorf("class 3 message missing %q; got: %s", want, msg)
		}
	}
	// Names a fix: install/declare the capability or fix the command.
	if !strings.Contains(msg, "install") && !strings.Contains(msg, "declare") && !strings.Contains(msg, "fix") {
		t.Errorf("class 3 message must name a fix (install/declare/fix); got: %s", msg)
	}
}

// TestMessage_NoBareExitCodeOrUnannotatedFailure (CLM-021): no traceability
// fail-loud path produces a bare exit code or an unannotated "failed" with no
// cause and no fix — every class carries a non-trivial, annotated message.
func TestMessage_NoBareExitCodeOrUnannotatedFailure(t *testing.T) {
	cases := []struct {
		name  string
		dim   TraceabilityDimension
		class PolarityClass
		cfg   *config.Config
		cap   CapabilityState
	}{
		{"class1", DimensionSubstantiveness, ClassBrokenDeclared, cfgDeclaring("go", DimensionSubstantiveness, false), CapabilityState{Present: true, Working: false, PackOrCommand: "go test ./...", Detail: "boom"}},
		{"class2", DimensionCoverage, ClassCapabilityAbsent, cfgDeclaring("typescript", DimensionContracts, false), CapabilityState{Present: false, PackOrCommand: "a ts coverage pack"}},
		{"class3", DimensionContracts, ClassDeclaredIntentUnmet, cfgDeclaring("go", DimensionContracts, false), CapabilityState{Present: false, PackOrCommand: "the go contract analyzer"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := PolarityStepResult(StepContractSignature, tc.dim, tc.class, tc.cfg, tc.cap)
			if len(res.Violations) == 0 {
				t.Fatalf("%s must carry an annotated violation, got none", tc.name)
			}
			msg := res.Violations[0].Message
			if strings.TrimSpace(msg) == "" {
				t.Errorf("%s message is empty (unannotated failure)", tc.name)
			}
			if msg == "failed" || msg == "exit 2" || msg == "exit code 2" {
				t.Errorf("%s message is a bare exit code / unannotated failure: %q", tc.name, msg)
			}
			// Must name the dimension at minimum.
			if !strings.Contains(msg, string(tc.dim)) {
				t.Errorf("%s message must name the dimension %q; got: %s", tc.name, tc.dim, msg)
			}
		})
	}
}
