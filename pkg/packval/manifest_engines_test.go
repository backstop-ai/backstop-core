package packval

import (
	"reflect"
	"testing"

	"github.com/bmanson/backstop-core/pkg/pack"
	"github.com/bmanson/backstop-core/pkg/pack/engine"
)

// TestParseManifest_EnginesBlockStringEnums (CLM-012 / CLM-006, B0): a pack
// manifest whose engines: block carries STRING enum values — go-toolchain's real
// shape (scope_kind: project-wide, category: mechanism, gate_type: build/coverage/
// lint, input_mode: none) — parses at phase1 WITHOUT error. Before the fix this
// dies with the int-enum unmarshal errors (engine.ScopeKind/EngineCategory/GateType
// have no UnmarshalYAML), so go-toolchain never reaches phase2/5.
func TestParseManifest_EnginesBlockStringEnums(t *testing.T) {
	m, err := ParseManifest("testdata/engine-pack/pack.yml")
	if err != nil {
		t.Fatalf("expected engines-block string enums to parse, got error: %v", err)
	}
	if len(m.Engines) == 0 {
		t.Fatal("expected engines block to be populated")
	}

	// The string spellings must resolve to the exact enum values, not zero-values.
	build, ok := m.Engines["mybuild"]
	if !ok {
		t.Fatal("engine mybuild missing from parsed manifest")
	}
	if build.ScopeKind != engine.ScopeKindProjectWide {
		t.Errorf("mybuild scope_kind: got %v, want ScopeKindProjectWide", build.ScopeKind)
	}
	if build.Category != engine.EngineCategoryMechanism {
		t.Errorf("mybuild category: got %v, want EngineCategoryMechanism", build.Category)
	}
	if build.GateType != engine.GateTypeBuild {
		t.Errorf("mybuild gate_type: got %v, want GateTypeBuild", build.GateType)
	}
	if build.InputMode != engine.InputModeNone {
		t.Errorf("mybuild input_mode: got %v, want InputModeNone", build.InputMode)
	}
	cov, ok := m.Engines["mycoverage"]
	if !ok {
		t.Fatal("engine mycoverage missing from parsed manifest")
	}
	if cov.GateType != engine.GateTypeCoverage {
		t.Errorf("mycoverage gate_type: got %v, want GateTypeCoverage", cov.GateType)
	}
}

// TestParseManifest_EnginesRoundTripMatchesConsumer (CLM-012): packval parses the
// engines: block to the SAME resolved bindings the real consumer (pkg/pack, via
// EngineSpec/parseEngineSpec) yields for the identical file. This pins packval to
// the consumer so the two parsers cannot drift.
func TestParseManifest_EnginesRoundTripMatchesConsumer(t *testing.T) {
	pvM, err := ParseManifest("testdata/engine-pack/pack.yml")
	if err != nil {
		t.Fatalf("packval ParseManifest: %v", err)
	}
	consumerM, err := pack.ParseManifestFile("testdata/engine-pack/pack.yml")
	if err != nil {
		t.Fatalf("consumer ParseManifestFile: %v", err)
	}
	for name, pvBinding := range pvM.Engines {
		spec, ok := consumerM.Engines[name]
		if !ok {
			t.Fatalf("engine %q present in packval parse but absent from consumer parse", name)
		}
		if !reflect.DeepEqual(pvBinding, spec.Binding) {
			t.Errorf("engine %q binding mismatch:\n packval  = %+v\n consumer = %+v", name, pvBinding, spec.Binding)
		}
	}
}
