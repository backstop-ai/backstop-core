package config

// ISSUE-217: enforcement.coverage_floor is the project-level fallback per-file
// coverage threshold. These tests load real backstop.yml files through
// LoadConfigFromPath, so they exercise the strict decoder and the load-time
// range/type check together.

import (
	"strings"
	"testing"
)

// TestCoverageFloorUnsetDefaultsTo90 (CLM-001): an absent key resolves to the
// default 90, and an explicit value (including 0, meaning "no floor") is kept
// rather than being collapsed into the default.
func TestCoverageFloorUnsetDefaultsTo90(t *testing.T) {
	if DefaultCoverageFloor != 90 {
		t.Fatalf("DefaultCoverageFloor = %d, want 90 (no behavior change for existing consumers)", DefaultCoverageFloor)
	}

	t.Run("unset", func(t *testing.T) {
		cfg, err := LoadConfigFromPath(writeConfig(t, "project: floor-unset\n"))
		if err != nil {
			t.Fatalf("loading a config without coverage_floor: %v", err)
		}
		if cfg.Enforcement.CoverageFloor != nil {
			t.Errorf("an absent coverage_floor must stay nil (unset), got %d", *cfg.Enforcement.CoverageFloor)
		}
		if got := cfg.Enforcement.EffectiveCoverageFloor(); got != 90 {
			t.Errorf("EffectiveCoverageFloor() with no key = %d, want 90 (== DefaultCoverageFloor)", got)
		}
	})

	t.Run("unset with other enforcement keys", func(t *testing.T) {
		cfg, err := LoadConfigFromPath(writeConfig(t, "project: floor-unset-enf\nenforcement:\n  waiver_warning_days: 10\n"))
		if err != nil {
			t.Fatalf("loading: %v", err)
		}
		if got := cfg.Enforcement.EffectiveCoverageFloor(); got != 90 {
			t.Errorf("EffectiveCoverageFloor() = %d, want 90", got)
		}
	})

	t.Run("explicit zero is kept", func(t *testing.T) {
		cfg, err := LoadConfigFromPath(writeConfig(t, "project: floor-zero\nenforcement:\n  coverage_floor: 0\n"))
		if err != nil {
			t.Fatalf("loading coverage_floor: 0: %v", err)
		}
		if cfg.Enforcement.CoverageFloor == nil {
			t.Fatal("an explicit coverage_floor: 0 must be kept as a present value, not read as unset")
		}
		if got := cfg.Enforcement.EffectiveCoverageFloor(); got != 0 {
			t.Errorf("EffectiveCoverageFloor() with coverage_floor: 0 = %d, want 0 (no floor)", got)
		}
	})

	t.Run("explicit 75", func(t *testing.T) {
		cfg, err := LoadConfigFromPath(writeConfig(t, "project: floor-75\nenforcement:\n  coverage_floor: 75\n"))
		if err != nil {
			t.Fatalf("loading coverage_floor: 75: %v", err)
		}
		if got := cfg.Enforcement.EffectiveCoverageFloor(); got != 75 {
			t.Errorf("EffectiveCoverageFloor() = %d, want 75", got)
		}
	})

	t.Run("zero value Enforcement", func(t *testing.T) {
		var enf Enforcement
		if got := enf.EffectiveCoverageFloor(); got != DefaultCoverageFloor {
			t.Errorf("zero-value Enforcement.EffectiveCoverageFloor() = %d, want %d", got, DefaultCoverageFloor)
		}
	})
}

// TestCoverageFloorRejectsInvalidValues (CLM-004): out-of-range values,
// non-integers and an explicit null are refused at load, and the error names the
// key so the operator can find it. The range edges 0 and 100 load fine.
func TestCoverageFloorRejectsInvalidValues(t *testing.T) {
	invalid := []struct {
		name  string
		value string
	}{
		{name: "negative", value: "-1"},
		{name: "above 100", value: "101"},
		{name: "float", value: "85.5"},
		{name: "quoted string", value: `"90"`},
		{name: "boolean", value: "true"},
		{name: "explicit null", value: "null"},
		{name: "bare key", value: ""},
		{name: "sequence", value: "[90]"},
	}
	for _, tc := range invalid {
		t.Run(tc.name, func(t *testing.T) {
			body := "project: floor-invalid\nenforcement:\n  coverage_floor: " + tc.value + "\n"
			cfg, err := LoadConfigFromPath(writeConfig(t, body))
			if err == nil {
				t.Fatalf("coverage_floor: %q must be rejected at load, got config %+v", tc.value, cfg)
			}
			if !strings.Contains(err.Error(), "enforcement.coverage_floor") {
				t.Errorf("the error for coverage_floor: %q must name enforcement.coverage_floor, got: %v", tc.value, err)
			}
		})
	}

	for _, edge := range []struct {
		value string
		want  int
	}{{"0", 0}, {"100", 100}} {
		t.Run("edge "+edge.value, func(t *testing.T) {
			cfg, err := LoadConfigFromPath(writeConfig(t, "project: floor-edge\nenforcement:\n  coverage_floor: "+edge.value+"\n"))
			if err != nil {
				t.Fatalf("coverage_floor: %s is in range and must load, got: %v", edge.value, err)
			}
			if got := cfg.Enforcement.EffectiveCoverageFloor(); got != edge.want {
				t.Errorf("EffectiveCoverageFloor() = %d, want %d", got, edge.want)
			}
		})
	}
}
