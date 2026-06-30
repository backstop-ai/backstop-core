package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// deletedBridgeSymbols returns the three language-derived bridge identifiers
// SPEC-046 deletes (REQ-001). They are assembled by string concatenation so the
// contiguous identifier never appears as a literal anywhere in this _test.go —
// TestGate_NoTestReferencesDeletedBridgeSymbols scans the whole cmd/backstop test
// tree for these symbols, and a contiguous literal in the guard file would
// self-trip it.
func deletedBridgeSymbols() []string {
	return []string{
		"loadBridged" + "ToolchainPacks",
		"gate" + "Language",
		"toolchainPack" + "Name",
	}
}

// TestGate_BridgeLoaderSymbolDeleted (CLM-001): the language-derived bridge loader
// is DELETED from gate.go — a source guard over the file asserts the function
// definition is absent, so a reintroduced language-derived toolchain loader is
// caught as a regression.
func TestGate_BridgeLoaderSymbolDeleted(t *testing.T) {
	src := readFileStr(t, "gate.go")
	if strings.Contains(src, "func "+"loadBridged"+"ToolchainPacks") {
		t.Error("loadBridgedToolchainPacks must be DELETED from gate.go — toolchain packs load ONLY via the declared-pack path, never auto-loaded from a language field (CLM-001)")
	}
}

// TestGate_ToolchainPackNameDeriverDeleted (CLM-002): the
// `"backstop/"+language+"-toolchain"` name deriver is DELETED — no pack name is
// ever computed from a language field.
func TestGate_ToolchainPackNameDeriverDeleted(t *testing.T) {
	src := readFileStr(t, "gate.go")
	if strings.Contains(src, "func "+"toolchainPack"+"Name") {
		t.Error("toolchainPackName must be DELETED — no toolchain pack name may be derived from a language field (CLM-002)")
	}
}

// TestGate_GateLanguageReaderDeleted (CLM-003): the gateLanguage reader is DELETED
// from gate.go.
func TestGate_GateLanguageReaderDeleted(t *testing.T) {
	src := readFileStr(t, "gate.go")
	if strings.Contains(src, "func "+"gate"+"Language") {
		t.Error("gateLanguage must be DELETED from gate.go — the bridge read it to derive a toolchain pack; that path is retired (CLM-003)")
	}
}

// TestGate_BridgedInputRemovedFromGateWiring (CLM-006): the `bridged` manifest list
// is removed as an input across the gate wiring — coverageRecordsProducer,
// toolchainEnforcementStatus, the dispatch set, and the zero-pack early return no
// longer reference a `bridged` list (they key on the declared `packs` alone). The
// guard asserts no `bridged` identifier survives anywhere in gate.go.
func TestGate_BridgedInputRemovedFromGateWiring(t *testing.T) {
	src := readFileStr(t, "gate.go")
	if strings.Contains(src, "bridged") {
		t.Error("the `bridged` manifest list must be removed from the gate wiring — coverageRecordsProducer / toolchainEnforcementStatus / the dispatch set / the zero-pack early return must key on the declared `packs` alone (CLM-006)")
	}
}

// TestGate_ClassifierPlumbingSurvivesBridgeDeletion (CLM-022): the bridge deletion
// does NOT orphan SPEC-043's classifier plumbing — mergeSourceClassifier /
// gate.SourceClassifier SURVIVE, are still called with the DECLARED-pack manifest
// set, and the declared-pack set (loadInstalledPacks over backstop.yml packs:)
// remains in scope at the classifier call site after `bridged` is removed.
func TestGate_ClassifierPlumbingSurvivesBridgeDeletion(t *testing.T) {
	src := readFileStr(t, "gate.go")
	if !strings.Contains(src, "func mergeSourceClassifier") {
		t.Error("mergeSourceClassifier (a SPEC-043 symbol) must SURVIVE the bridge deletion — it is fenced, not deleted (CLM-022)")
	}
	if !strings.Contains(src, "mergeSourceClassifier(packs)") {
		t.Error("mergeSourceClassifier must still be CALLED with the declared-pack manifest set (`packs`), so the classifier sources from the declared set after `bridged` is removed (CLM-022)")
	}
	if !strings.Contains(src, "gate.SourceClassifier") {
		t.Error("the gate.SourceClassifier plumbing must survive the bridge deletion (CLM-022)")
	}
	if !strings.Contains(src, "loadInstalledPacks(projectRoot)") {
		t.Error("the declared-pack manifest set (loadInstalledPacks over backstop.yml packs:) must remain in scope at the classifier call site after `bridged` is removed (CLM-022)")
	}
}

// TestGate_NoTestReferencesDeletedBridgeSymbols (CLM-023): NO _test.go file in
// cmd/backstop references loadBridgedToolchainPacks, gateLanguage, or
// toolchainPackName — a surviving reference means the bridge was shimmed rather
// than deleted (the green-keeping shim the spec forbids).
func TestGate_NoTestReferencesDeletedBridgeSymbols(t *testing.T) {
	cmdDir := filepath.Join(repoRoot(t), "cmd", "backstop")
	entries, err := os.ReadDir(cmdDir)
	if err != nil {
		t.Fatalf("reading %s: %v", cmdDir, err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		// Skip this guard's OWN source file: it legitimately names the forbidden
		// symbols in its assertion messages/comments to prove their absence — those
		// are prose, not a live code reference, and a textual scan cannot tell them
		// apart. Every OTHER test file is held to "no reference."
		if e.Name() == "gate_bridge_deletion_test.go" {
			continue
		}
		b, err := os.ReadFile(filepath.Join(cmdDir, e.Name()))
		if err != nil {
			t.Fatalf("reading %s: %v", e.Name(), err)
		}
		src := string(b)
		for _, sym := range deletedBridgeSymbols() {
			if strings.Contains(src, sym) {
				t.Errorf("%s references deleted bridge symbol %q — the bridge must be DELETED, not kept alive as a green-keeping shim (CLM-023)", e.Name(), sym)
			}
		}
	}
}
