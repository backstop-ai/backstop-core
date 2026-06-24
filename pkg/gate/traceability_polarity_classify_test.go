package gate

import (
	"path/filepath"
	"testing"

	"github.com/bmanson/backstop-core/pkg/config"
)

// loadPolarityFixture loads a backstop.yml fixture from testdata/polarity.
func loadPolarityFixture(t *testing.T, name string) *config.Config {
	t.Helper()
	path := filepath.Join("testdata", "polarity", name)
	cfg, err := config.LoadConfigFromPath(path)
	if err != nil {
		t.Fatalf("loading fixture %s: %v", name, err)
	}
	return cfg
}

// TestClassify_SubstantivenessDeclared_WhenToolchainHasGateType (CLM-001): a
// backstop.yml whose enforcement.toolchain declares a substantiveness gate_type
// classifies the substantiveness dimension as DECLARED. With a missing
// capability that surfaces as class 3 (declared-intent-unmet), proving the
// dimension was recognized as DECLARED (an UNDECLARED dimension with the same
// absent capability would be class 2).
func TestClassify_SubstantivenessDeclared_WhenToolchainHasGateType(t *testing.T) {
	cfg := loadPolarityFixture(t, "declared-substantiveness.yml")
	if !declaredDimension(cfg, DimensionSubstantiveness) {
		t.Fatal("substantiveness must be DECLARED when a toolchain entry names gate_type: substantiveness")
	}
	// Declared + capability absent -> class 3, not class 2.
	got := ClassifyDimension(cfg, DimensionSubstantiveness, CapabilityState{Present: false})
	if got != ClassDeclaredIntentUnmet {
		t.Errorf("declared substantiveness with absent capability: got %v, want ClassDeclaredIntentUnmet", got)
	}
}

// TestClassify_CoverageUndeclared_WhenNoToolchainGateType (CLM-002): a
// backstop.yml with no enforcement.toolchain entry for coverage classifies the
// coverage dimension as UNDECLARED; with an absent capability that is class 2.
func TestClassify_CoverageUndeclared_WhenNoToolchainGateType(t *testing.T) {
	cfg := loadPolarityFixture(t, "undeclared-coverage.yml")
	if declaredDimension(cfg, DimensionCoverage) {
		t.Fatal("coverage must be UNDECLARED when no toolchain entry names gate_type: coverage")
	}
	got := ClassifyDimension(cfg, DimensionCoverage, CapabilityState{Present: false})
	if got != ClassCapabilityAbsent {
		t.Errorf("undeclared coverage with absent capability: got %v, want ClassCapabilityAbsent", got)
	}
}

// TestClassify_DeclaredAndWorking_IsNotAFailLoudClass (CLM-003): a dimension
// that is declared AND has a working capability is classified as NEITHER of the
// three fail-loud classes and routes to its normal step.
func TestClassify_DeclaredAndWorking_IsNotAFailLoudClass(t *testing.T) {
	cfg := loadPolarityFixture(t, "declared-working.yml")
	got := ClassifyDimension(cfg, DimensionSubstantiveness, CapabilityState{Present: true, Working: true})
	if got != ClassNone {
		t.Errorf("declared-and-working: got %v, want ClassNone (proceed to normal step)", got)
	}
}

// TestClassify_MutuallyExclusive_ExactlyOneClassPerDimension (CLM-004): each
// dimension resolves to exactly one of the three classes (or none/proceed); the
// classifier never returns an out-of-range value and the allowlist is
// exhaustive across the full input grid.
func TestClassify_MutuallyExclusive_ExactlyOneClassPerDimension(t *testing.T) {
	dims := []TraceabilityDimension{DimensionSubstantiveness, DimensionCoverage, DimensionContracts}
	caps := []CapabilityState{
		{Present: false},
		{Present: true, Working: false},
		{Present: true, Working: true},
	}
	declaredCfg := loadPolarityFixture(t, "declared-working.yml")      // declares substantiveness
	undeclaredCfg := loadPolarityFixture(t, "undeclared-coverage.yml") // declares no dimension
	valid := map[PolarityClass]bool{
		ClassNone:                true,
		ClassBrokenDeclared:      true,
		ClassCapabilityAbsent:    true,
		ClassDeclaredIntentUnmet: true,
	}
	for _, cfg := range []*config.Config{declaredCfg, undeclaredCfg} {
		for _, dim := range dims {
			for _, cap := range caps {
				got := ClassifyDimension(cfg, dim, cap)
				if !valid[got] {
					t.Errorf("ClassifyDimension(%v, %+v) = %v, not a recognized class", dim, cap, got)
				}
			}
		}
	}
}

// TestClassify_LanguageAgnostic_SameVerdictAcrossStacks (CLM-005): the
// classifier reaches its verdict from the declaration surface and capability
// availability alone, with no language-specific branch — the same undeclared
// input yields class-2 for a Go project and a TypeScript project alike.
func TestClassify_LanguageAgnostic_SameVerdictAcrossStacks(t *testing.T) {
	goCfg := loadPolarityFixture(t, "undeclared-coverage.yml") // language: go, coverage undeclared
	tsCfg := loadPolarityFixture(t, "typescript-project.yml")  // language: typescript, coverage undeclared
	absent := CapabilityState{Present: false}
	goClass := ClassifyDimension(goCfg, DimensionCoverage, absent)
	tsClass := ClassifyDimension(tsCfg, DimensionCoverage, absent)
	if goClass != ClassCapabilityAbsent || tsClass != ClassCapabilityAbsent {
		t.Errorf("undeclared+absent must be class 2 regardless of stack: go=%v ts=%v", goClass, tsClass)
	}
	if goClass != tsClass {
		t.Errorf("language-agnostic classifier returned different classes across stacks: go=%v ts=%v", goClass, tsClass)
	}
}

// TestBrokenDeclared_CommandErrors_BlocksExit2 (CLM-006): a declared dimension
// whose command exits non-zero (capability Present but not Working) is
// classified BROKEN-DECLARED.
func TestBrokenDeclared_CommandErrors_BlocksExit2(t *testing.T) {
	cfg := loadPolarityFixture(t, "declared-substantiveness.yml")
	got := ClassifyDimension(cfg, DimensionSubstantiveness, CapabilityState{Present: true, Working: false, Detail: "exit status 1"})
	if got != ClassBrokenDeclared {
		t.Errorf("declared + command-errors: got %v, want ClassBrokenDeclared", got)
	}
}

// TestBrokenDeclared_UnparseableOutput_BlocksExit2 (CLM-007): a declared
// dimension whose command output cannot be parsed by its declared format
// (Present but not Working) is classified BROKEN-DECLARED.
func TestBrokenDeclared_UnparseableOutput_BlocksExit2(t *testing.T) {
	cfg := loadPolarityFixture(t, "declared-substantiveness.yml")
	got := ClassifyDimension(cfg, DimensionSubstantiveness, CapabilityState{Present: true, Working: false, Detail: "unparseable output"})
	if got != ClassBrokenDeclared {
		t.Errorf("declared + unparseable-output: got %v, want ClassBrokenDeclared", got)
	}
}

// TestBrokenDeclared_UnknownToolchainKey_BlocksExit2 (CLM-008, Sharp Edge 2): a
// declared dimension whose enforcement.toolchain entry names an unknown
// toolchain key (a gate_type that matches no recognized dimension) is
// classified BROKEN-DECLARED, NOT silently treated as undeclared (class 2).
func TestBrokenDeclared_UnknownToolchainKey_BlocksExit2(t *testing.T) {
	cfg := loadPolarityFixture(t, "declared-unknown-key.yml") // gate_type: bogus-dimension
	// The malformed declaration must NOT silently read as undeclared/class-2.
	// Classifying any real dimension on this config yields BROKEN-DECLARED
	// because the config carries an unrecognized gate_type.
	got := ClassifyDimension(cfg, DimensionCoverage, CapabilityState{Present: false})
	if got != ClassBrokenDeclared {
		t.Errorf("unknown toolchain gate_type must be class 1 (broken-declared), got %v", got)
	}
}

// TestCapabilityAbsent_Undeclared (CLM-010 verdict input): an undeclared,
// capability-absent dimension classifies as CAPABILITY-ABSENT.
func TestCapabilityAbsent_Undeclared(t *testing.T) {
	cfg := loadPolarityFixture(t, "undeclared-coverage.yml")
	got := ClassifyDimension(cfg, DimensionContracts, CapabilityState{Present: false})
	if got != ClassCapabilityAbsent {
		t.Errorf("undeclared + absent: got %v, want ClassCapabilityAbsent", got)
	}
	// Undeclared but present -> none/proceed (existing baked capability covers it).
	gotPresent := ClassifyDimension(cfg, DimensionContracts, CapabilityState{Present: true, Working: true})
	if gotPresent != ClassNone {
		t.Errorf("undeclared + present: got %v, want ClassNone", gotPresent)
	}
}

// TestDeclaredIntentUnmet_MissingCapability (CLM-013): a dimension that IS
// declared but whose required capability is missing classifies as
// DECLARED-INTENT-UNMET.
func TestDeclaredIntentUnmet_MissingCapability(t *testing.T) {
	cfg := loadPolarityFixture(t, "declared-substantiveness.yml")
	got := ClassifyDimension(cfg, DimensionSubstantiveness, CapabilityState{Present: false})
	if got != ClassDeclaredIntentUnmet {
		t.Errorf("declared + missing capability: got %v, want ClassDeclaredIntentUnmet", got)
	}
}
