package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/baseengines"
)

// ISSUE-027 deletion-assertion tests. The absence assertions pin that the baked
// engine tables (DefaultRegistry, DefaultFieldContracts) and the baked claim-code
// indirection (engineFieldClaim map + claimFor) are GONE from production source, so
// the binary holds ZERO baked engine knowledge — every engine fact now resolves
// from pack DATA (the embedded base-engines pack + the external go-toolchain pack).
// They are RED while the symbols still exist and go green only after the Phase-4
// rewire + deletion lands.

func readRepoFile(t *testing.T, rel string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), rel))
	if err != nil {
		t.Fatalf("reading %s: %v", rel, err)
	}
	return string(raw)
}

// TestDeletion_DefaultRegistryGone proves the leaf engine package no longer defines
// the baked DefaultRegistry table (CLM-007). binding.go must contain only the types,
// ParseInputMode, and Registry.Lookup.
func TestDeletion_DefaultRegistryGone(t *testing.T) {
	src := readRepoFile(t, filepath.Join("pkg", "pack", "engine", "binding.go"))
	if strings.Contains(src, "DefaultRegistry") {
		t.Error("pkg/pack/engine/binding.go still references DefaultRegistry; the baked engine table must be deleted (built-ins resolve from the embedded base pack)")
	}
}

// TestDeletion_DefaultFieldContractsGone proves the leaf engine package no longer
// defines the baked DefaultFieldContracts map (CLM-008). fieldcontract.go must
// contain only the FieldContract type + field-name consts.
func TestDeletion_DefaultFieldContractsGone(t *testing.T) {
	src := readRepoFile(t, filepath.Join("pkg", "pack", "engine", "fieldcontract.go"))
	if strings.Contains(src, "DefaultFieldContracts") {
		t.Error("pkg/pack/engine/fieldcontract.go still references DefaultFieldContracts; the baked contract map must be deleted (contracts travel inline on the base pack bindings)")
	}
}

// TestDeletion_EngineFieldClaimGone proves the validator no longer carries the baked
// engineFieldClaim map or the claimFor indirection (CLM-006). Field-contract
// violations now report a single generic claim code.
func TestDeletion_EngineFieldClaimGone(t *testing.T) {
	src := readRepoFile(t, filepath.Join("pkg", "pack", "validate_manifest.go"))
	for _, sym := range []string{"engineFieldClaim", "claimFor"} {
		if strings.Contains(src, sym) {
			t.Errorf("pkg/pack/validate_manifest.go still references baked claim symbol %q; it must be deleted for a single generic engine-field-contract code", sym)
		}
	}
}

// TestResolveEngineRegistry_BuiltinsFromBasePack proves the GATE path resolves the
// four generic built-ins from the embedded base pack, not a baked table (CLM-003):
// with the engineRegistry seam nil, resolveEngineRegistry(nil) returns semgrep /
// ast-grep / sandbox / config-file sourced from baseengines.Registry().
func TestResolveEngineRegistry_BuiltinsFromBasePack(t *testing.T) {
	orig := engineRegistry
	engineRegistry = nil
	t.Cleanup(func() { engineRegistry = orig })

	reg := resolveEngineRegistry(nil)

	for _, name := range []string{"semgrep", "ast-grep", "sandbox", "config-file"} {
		if _, ok := reg[name]; !ok {
			t.Errorf("resolveEngineRegistry(nil) missing built-in %q sourced from the base pack", name)
		}
	}

	base := baseengines.Registry()
	if reg["semgrep"].Command != base["semgrep"].Command {
		t.Errorf("semgrep Command = %q, want base-pack value %q — built-in not sourced from embedded base pack",
			reg["semgrep"].Command, base["semgrep"].Command)
	}
	if reg["semgrep"].Command != "semgrep --sarif --quiet" {
		t.Errorf("semgrep Command = %q, want %q (base pack data)", reg["semgrep"].Command, "semgrep --sarif --quiet")
	}
}
