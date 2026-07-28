package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/config"
)

// TestPolicy_PackScopedEntryParsesAndUnscopedEntryUnchanged proves the enforcement
// policy gains per-PACK/per-rule-SOURCE granularity (SPEC-047 REQ-007/CLM-036): a
// pack-scoped policy entry (pack_engines scoped to source backstop/self with level
// block + applies-to all-code) parses onto the EXTENDED DimensionPolicy structure, AND
// an unscoped (dimension-only) entry keeps its current behavior — backward compatible.
func TestPolicy_PackScopedEntryParsesAndUnscopedEntryUnchanged(t *testing.T) {
	dir := t.TempDir()
	yml := `project: perpack-test
packs:
    backstop/self: local
enforcement:
    policy:
        pack_engines:
            level: block
            applies-to: new-code
            sources:
                backstop/self:
                    level: block
                    applies-to: all-code
        coverage_threshold:
            level: block
            applies-to: new-code
`
	path := filepath.Join(dir, "backstop.yml")
	if err := os.WriteFile(path, []byte(yml), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.LoadConfigFromPath(path)
	if err != nil {
		t.Fatalf("a per-pack-scoped policy entry must parse (backward compatible strict load): %v", err)
	}

	pe, ok := cfg.Enforcement.Policy["pack_engines"]
	if !ok {
		t.Fatal("pack_engines policy entry must parse")
	}
	// The dimension-level default is unchanged.
	if pe.Level != "block" || pe.AppliesTo != "new-code" {
		t.Errorf("pack_engines dimension default must be level block + applies-to new-code, got level=%q applies-to=%q", pe.Level, pe.AppliesTo)
	}
	// The per-SOURCE override for backstop/self parses onto the new scoping.
	self, ok := pe.Sources["backstop/self"]
	if !ok {
		t.Fatalf("pack_engines must carry a per-source override for backstop/self, got sources=%v", pe.Sources)
	}
	if self.Level != "block" || self.AppliesTo != "all-code" {
		t.Errorf("the backstop/self-scoped override must be level block + applies-to all-code (the flip), got level=%q applies-to=%q", self.Level, self.AppliesTo)
	}

	// The UNSCOPED entry keeps its current behavior — no sources map, parses exactly
	// as today (backward compatible).
	ct, ok := cfg.Enforcement.Policy["coverage_threshold"]
	if !ok {
		t.Fatal("coverage_threshold policy entry must parse")
	}
	if len(ct.Sources) != 0 {
		t.Errorf("an unscoped (dimension-only) entry must carry NO per-source overrides, got %v", ct.Sources)
	}
	if ct.Level != "block" || ct.AppliesTo != "new-code" {
		t.Errorf("the unscoped coverage_threshold entry must behave as today (block, applies-to new-code), got level=%q applies-to=%q", ct.Level, ct.AppliesTo)
	}
}

// TestDimensionPolicy_AbsentAppliesTo_DefaultsAllCode proves the SAFETY INVARIANT
// (CLM-002): a policy entry with NO applies-to key parses to a DimensionPolicy whose
// AppliesTo is empty, and the consumer treats an empty AppliesTo as the all-code
// (block-on-total) strict default — an absent key is NEVER silently promoted to the
// grandfathering new-code mode. This preserves today's key-absent = block-on-total
// behavior across the rename.
func TestDimensionPolicy_AbsentAppliesTo_DefaultsAllCode(t *testing.T) {
	dir := t.TempDir()
	yml := `project: absent-appliesto-test
enforcement:
    policy:
        artifact_validation:
            level: block
`
	path := filepath.Join(dir, "backstop.yml")
	if err := os.WriteFile(path, []byte(yml), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.LoadConfigFromPath(path)
	if err != nil {
		t.Fatalf("a bare (applies-to-absent) policy entry must parse and validate: %v", err)
	}

	av, ok := cfg.Enforcement.Policy["artifact_validation"]
	if !ok {
		t.Fatal("artifact_validation policy entry must parse")
	}
	// The absent key parses to the empty string — NOT "new-code". Empty is the
	// all-code default (block on total), matching today's strict key-absent behavior.
	if av.AppliesTo != "" {
		t.Errorf("an absent applies-to key must parse to the empty string (the all-code default), not %q — the strict default must not silently become new-code", av.AppliesTo)
	}
	if av.AppliesTo == "new-code" {
		t.Error("an absent applies-to key must NEVER resolve to new-code (grandfathering) — that would silently weaken the strict default")
	}
}

// TestDimensionPolicy_NestedSourcesOverride_Parses proves the rename covers the
// self-referential Sources map, not just the top-level field (CLM-001): a dimension
// carrying sources: { backstop/self: { applies-to: all-code, level: block } } parses so
// the nested entry's AppliesTo == "all-code" and Level == "block".
func TestDimensionPolicy_NestedSourcesOverride_Parses(t *testing.T) {
	dir := t.TempDir()
	yml := `project: nested-sources-test
packs:
    backstop/self: local
enforcement:
    policy:
        pack_engines:
            level: block
            applies-to: new-code
            sources:
                backstop/self:
                    level: block
                    applies-to: all-code
`
	path := filepath.Join(dir, "backstop.yml")
	if err := os.WriteFile(path, []byte(yml), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.LoadConfigFromPath(path)
	if err != nil {
		t.Fatalf("a nested-sources override must parse: %v", err)
	}

	pe, ok := cfg.Enforcement.Policy["pack_engines"]
	if !ok {
		t.Fatal("pack_engines policy entry must parse")
	}
	self, ok := pe.Sources["backstop/self"]
	if !ok {
		t.Fatalf("the nested sources override for backstop/self must parse, got sources=%v", pe.Sources)
	}
	if self.AppliesTo != "all-code" {
		t.Errorf("the nested Sources entry's AppliesTo must parse to all-code (the rename covers the self-referential map), got %q", self.AppliesTo)
	}
	if self.Level != "block" {
		t.Errorf("the nested Sources entry's Level must parse to block, got %q", self.Level)
	}
}
