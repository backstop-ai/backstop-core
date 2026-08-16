package main

import (
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/gate"
	"github.com/backstop-ai/backstop-core/pkg/pack"
	"github.com/backstop-ai/backstop-core/pkg/pack/engine"
)

// gateTypeStampManifest builds an in-memory manifest whose single engine declares
// the given gate_type over an allowlisted, input-less, project-wide command. The
// binding carries NO convert, so the payload the runner hands back reaches
// check.ParsePackFindings unchanged and the produced gate.Violations are exactly
// what the stamp site emitted.
func gateTypeStampManifest(name, engineName string, gt engine.GateType) *pack.Manifest {
	return &pack.Manifest{
		NormalizedName: name,
		Engines: map[string]pack.EngineSpec{
			engineName: {Binding: engine.EngineBinding{
				Command:   "grep",
				InputMode: engine.InputModeNone,
				ScopeKind: engine.ScopeKindProjectWide,
				GateType:  gt,
			}},
		},
		Content: pack.Content{Ruleset: pack.Ruleset{Rules: []pack.Rule{
			{ID: engineName + "-rule", Engine: engineName, Standard: "x"},
		}}},
	}
}

// realGoTestSarif runs the REAL committed go-toolchain converter over the REAL
// committed `go test` capture and returns its SARIF bytes. Three real hops
// (capture -> converter -> caller), nothing hand-built.
func realGoTestSarif(t *testing.T) []byte {
	t.Helper()
	out, err := runConvertScriptDirect(convertScript(t, "test-to-sarif.sh"), readFixture(t, "go-test-failures.txt"))
	if err != nil {
		t.Fatalf("running committed test-to-sarif.sh over committed capture: %v", err)
	}
	return out
}

// runStampedEngine drives the REAL runFindingsEngine for a manifest declaring gt,
// feeding it real converter output, and returns the produced violations.
func runStampedEngine(t *testing.T, gt engine.GateType, engineName string) []gate.Violation {
	t.Helper()
	m := gateTypeStampManifest("acme/stamp-probe", engineName, gt)
	binding := m.Engines[engineName].Binding
	runner := &fixtureRunner{byCmd: map[string][]byte{"grep": realGoTestSarif(t)}}
	violations, err := runFindingsEngine(m, t.TempDir(), t.TempDir(), nil, binding, m.Content.Ruleset.Rules, runner)
	if err != nil {
		t.Fatalf("runFindingsEngine (gate_type %s): %v", gt.String(), err)
	}
	if len(violations) == 0 {
		t.Fatalf("expected the real converter payload to yield findings, got none")
	}
	return violations
}

// TestPackDispatch_StampsDeclaredGateTypeOnBridgedViolations (CLM-001): every
// violation the dispatch produces carries ITS producing binding's DECLARED
// gate_type. The second binding declares a DIFFERENT gate_type over the SAME
// payload, which is what proves the value is READ from the declaration rather
// than being a constant the stamp site happens to agree with.
func TestPackDispatch_StampsDeclaredGateTypeOnBridgedViolations(t *testing.T) {
	testViolations := runStampedEngine(t, engine.GateTypeTest, "verdict")
	for i, v := range testViolations {
		if v.GateType != engine.GateTypeTest.String() {
			t.Fatalf("violation %d: GateType = %q, want %q", i, v.GateType, engine.GateTypeTest.String())
		}
		if v.ProjectWide {
			t.Fatalf("violation %d: ProjectWide = true, want false (binding declares no ExemptFromScopeFilter)", i)
		}
	}

	lintViolations := runStampedEngine(t, engine.GateTypeLint, "linter")
	for i, v := range lintViolations {
		if v.GateType != engine.GateTypeLint.String() {
			t.Fatalf("violation %d: GateType = %q, want %q — the stamp must READ the binding, not assume a type", i, v.GateType, engine.GateTypeLint.String())
		}
		if v.ProjectWide {
			t.Fatalf("violation %d: ProjectWide = true, want false", i)
		}
	}

	if len(testViolations) != len(lintViolations) {
		t.Fatalf("same payload yielded %d vs %d violations; the gate_type must not change parsing", len(testViolations), len(lintViolations))
	}
}

// TestPackDispatch_GateTypeIsAbsentFromBaselineIdentity (CLM-001, CLM-008): two
// violations differing ONLY in GateType hash to the SAME baseline identity. This
// is the anti-regression pin for sharp edge 7 — folding GateType into identity
// would silently stop every existing baseline entry from matching.
func TestPackDispatch_GateTypeIsAbsentFromBaselineIdentity(t *testing.T) {
	base := gate.Violation{
		Rule:       "acme/probe.go-test",
		File:       "pkg/widget/widget_test.go",
		Message:    "TestWidgetFrobnicate: expected 5, got 7",
		Severity:   "error",
		SourcePack: "acme/probe",
	}
	withTest := base
	withTest.GateType = engine.GateTypeTest.String()
	withLint := base
	withLint.GateType = engine.GateTypeLint.String()

	a := gate.EnrichViolationIdentity(withTest)
	b := gate.EnrichViolationIdentity(withLint)

	if a.Identity != b.Identity {
		t.Fatalf("GateType leaked into baseline identity: %q != %q", a.Identity, b.Identity)
	}
	if a.IdentityHash != b.IdentityHash {
		t.Fatalf("GateType leaked into the identity hash: %q != %q", a.IdentityHash, b.IdentityHash)
	}
	if a.RegionHash != b.RegionHash {
		t.Fatalf("GateType leaked into the region hash: %q != %q", a.RegionHash, b.RegionHash)
	}
}
