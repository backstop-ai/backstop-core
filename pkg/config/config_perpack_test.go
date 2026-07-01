package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bmanson/backstop-core/pkg/config"
)

// TestPolicy_PackScopedEntryParsesAndUnscopedEntryUnchanged proves the enforcement
// policy gains per-PACK/per-rule-SOURCE granularity (SPEC-047 REQ-007/CLM-036): a
// pack-scoped policy entry (pack_engines scoped to source backstop/self with level
// block + baseline false) parses onto the EXTENDED DimensionPolicy structure, AND an
// unscoped (dimension-only) entry keeps its current behavior — backward compatible.
func TestPolicy_PackScopedEntryParsesAndUnscopedEntryUnchanged(t *testing.T) {
	dir := t.TempDir()
	yml := `project: perpack-test
packs:
    backstop/self: local
enforcement:
    policy:
        pack_engines:
            level: block
            baseline: true
            sources:
                backstop/self:
                    level: block
                    baseline: false
        coverage_threshold:
            level: block
            baseline: true
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
	if pe.Level != "block" || pe.Baseline != true {
		t.Errorf("pack_engines dimension default must be level block + baseline true, got level=%q baseline=%v", pe.Level, pe.Baseline)
	}
	// The per-SOURCE override for backstop/self parses onto the new scoping.
	self, ok := pe.Sources["backstop/self"]
	if !ok {
		t.Fatalf("pack_engines must carry a per-source override for backstop/self, got sources=%v", pe.Sources)
	}
	if self.Level != "block" || self.Baseline != false {
		t.Errorf("the backstop/self-scoped override must be level block + baseline false (the flip), got level=%q baseline=%v", self.Level, self.Baseline)
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
	if ct.Level != "block" || ct.Baseline != true {
		t.Errorf("the unscoped coverage_threshold entry must behave as today (block, baseline true), got level=%q baseline=%v", ct.Level, ct.Baseline)
	}
}
