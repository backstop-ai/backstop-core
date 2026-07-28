package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/pack"
	"github.com/backstop-ai/backstop-core/pkg/pack/engine"
)

// gate_substantiveness_helpers_test.go covers the cmd/backstop-side substantiveness
// helper edge cases the re-wired step + capability re-key depend on: the installed-pack
// presence signal (substantivenessPackInstalled) and the resolution filter
// (resolveSubstantivenessPacks empty case). Real behavior assertions — each pins a
// disposition the gate verdict turns on.
//
// SPEC-045 REQ-003: TestGoFilePackageMatchesTarget_CmdEdgeCases was DELETED with the
// Go `package`-clause reader goFilePackageMatchesTarget it subjected. The same-unit
// derivation is now the language-neutral directory-leaf testFileColocatedWithTarget,
// covered by TestColocated_* (gate_substantiveness_coloc_test.go, CLM-019..022). Its
// old package-clause fixtures are NOT portable 1:1 — directory-leaf is a deliberate
// behavior change (Sharp Edge), so the obsolete assertions are removed, not ported.

// TestSubstantivenessPackInstalled_ReadsPacksMap — MIGRATED FOR ISSUE-063: the
// substantiveness capability signal keys on a DECLARED gate_type engine, not the pack
// name. Empty/nil pack set → false; a pack declaring only some other gate_type → false; a
// pack (any name/org) declaring a substantiveness engine → true.
func TestSubstantivenessPackInstalled_ReadsPacksMap(t *testing.T) {
	if substantivenessPackInstalled(nil) {
		t.Errorf("nil pack set must report the capability as NOT present")
	}
	if substantivenessPackInstalled([]*pack.Manifest{}) {
		t.Errorf("an empty pack set must report NOT present")
	}
	otherOnly := []*pack.Manifest{packDeclaringGateType("other/pack", engine.GateTypeLint)}
	if substantivenessPackInstalled(otherOnly) {
		t.Errorf("a pack declaring no substantiveness engine must report NOT present")
	}
	declaring := []*pack.Manifest{packDeclaringGateType("any/name", engine.GateTypeSubstantiveness)}
	if !substantivenessPackInstalled(declaring) {
		t.Errorf("a pack declaring a substantiveness engine must report present regardless of name")
	}
}

// TestResolveSubstantivenessPacks_FiltersInstalled — with no installed substantiveness
// pack in a temp project, the resolver returns an empty set (the step's no-op path).
func TestResolveSubstantivenessPacks_FiltersInstalled(t *testing.T) {
	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, "backstop.yml"), []byte("project: p\nlanguage: go\npacks: {}\n"), 0o644); err != nil {
		t.Fatalf("writing backstop.yml: %v", err)
	}
	packs, err := resolveSubstantivenessPacks(tmp)
	if err != nil {
		t.Fatalf("resolveSubstantivenessPacks: %v", err)
	}
	if len(packs) != 0 {
		t.Errorf("with no substantiveness pack installed, the resolver must return an empty set; got %d", len(packs))
	}
}
