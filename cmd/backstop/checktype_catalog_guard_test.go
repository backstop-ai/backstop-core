package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// catalogRealFiles returns the absolute paths of the real gate-semantic source
// files the catalog covers (the cataloged surface), so the guard can reconcile the
// real discovered-set against the real catalog and prove it does NOT red on arrival.
func catalogRealFiles(t *testing.T) []string {
	t.Helper()
	root := repoRoot(t)
	// After ISSUE-018 deleted the `backstop code check` command and its in-process
	// check engine, cmd/backstop/code_check.go and pkg/check/registry.go no longer
	// exist; the surviving gate-semantic CheckType surface is pack_gate.go (C-1),
	// parsers.go (C-5), and the CheckType enum vocabulary in check.go/manifest.go.
	rel := []string{
		"cmd/backstop/pack_gate.go",
		"pkg/check/parsers.go",
		"pkg/check/check.go",
		"pkg/check/manifest.go",
	}
	out := make([]string, 0, len(rel))
	for _, r := range rel {
		out = append(out, filepath.Join(root, filepath.FromSlash(r)))
	}
	return out
}

// TestCatalog_GuardScansGateSemanticSurfaceOnly proves the guard restricts
// discovery to GATE-SEMANTIC CheckType keying (identity comparisons +
// exemption/dispatch decisions), EXCLUDING pure .Pass.String() display sites, so
// the discovered-set provably equals the cataloged-set — and the display-site
// fixture is NOT discovered (SPEC-041 CLM-022).
func TestCatalog_GuardScansGateSemanticSurfaceOnly(t *testing.T) {
	// Reconcile the REAL catalog against the REAL cataloged surface: zero problems
	// (does not red on arrival; discovered == cataloged).
	problems, err := reconcileCheckTypeCatalog(repoRoot(t), CheckTypeConsumerCatalog(), catalogRealFiles(t))
	if err != nil {
		t.Fatalf("reconcile real surface: %v", err)
	}
	if len(problems) != 0 {
		t.Fatalf("the real catalog must reconcile clean against the real surface (no red on arrival), got %d problems: %#v", len(problems), problems)
	}

	// The cosmetic .Pass.String() display-site fixture must NOT be discovered as a
	// gate-semantic keying site (the negative case for the bounded scope).
	displayFixture := filepath.Join(repoRoot(t), "cmd", "backstop", "testdata", "catalog-display-site.go.txt")
	discovered, err := discoverGateSemanticSites([]string{displayFixture})
	if err != nil {
		t.Fatalf("discover over display fixture: %v", err)
	}
	if len(discovered) != 0 {
		t.Fatalf("a cosmetic .Pass.String() display site must NOT be discovered (CLM-022), got %#v", discovered)
	}
}

// TestCatalog_GuardFailsOnUnlistedConsumer proves the guard FAILS when a
// gate-semantic CheckType-keyed source site exists with no catalog entry — driven
// by the injected unlisted-keying-site fixture (SPEC-041 CLM-023). This proves the
// guard is not a tautology.
func TestCatalog_GuardFailsOnUnlistedConsumer(t *testing.T) {
	unlisted := filepath.Join(repoRoot(t), "cmd", "backstop", "testdata", "catalog-unlisted-keying-site.go.txt")
	// Discovery scope includes the unlisted fixture (a gate-semantic keying site in
	// a file NOT represented by any catalog entry).
	discovery := append(catalogRealFiles(t), unlisted)
	problems, err := reconcileCheckTypeCatalog(repoRoot(t), CheckTypeConsumerCatalog(), discovery)
	if err != nil {
		t.Fatalf("reconcile with unlisted fixture: %v", err)
	}
	if !hasProblemForFile(problems, "catalog-unlisted-keying-site.go.txt") {
		t.Fatalf("the guard must FAIL on the injected unlisted gate-semantic keying site (CLM-023), got %#v", problems)
	}
}

// TestCatalog_GuardFailsOnStaleEntry proves the guard FAILS on a stale catalog
// entry whose keying site no longer exists in code — driven by the stale-entry
// fixture (SPEC-041 CLM-024).
func TestCatalog_GuardFailsOnStaleEntry(t *testing.T) {
	staleFixture := filepath.Join(repoRoot(t), "cmd", "backstop", "testdata", "catalog-stale-entry.json")
	data, err := os.ReadFile(staleFixture)
	if err != nil {
		t.Fatalf("read stale fixture: %v", err)
	}
	var stale []CheckTypeConsumer
	if err := json.Unmarshal(data, &stale); err != nil {
		t.Fatalf("parse stale fixture: %v", err)
	}
	if len(stale) == 0 {
		t.Fatal("stale fixture must carry at least one entry")
	}

	// Reconcile a catalog containing the stale entry; the guard must flag it because
	// its site (cmd/backstop/ghost.go:phantomKeying) does not exist in code.
	problems, err := reconcileCheckTypeCatalog(repoRoot(t), stale, nil)
	if err != nil {
		t.Fatalf("reconcile stale catalog: %v", err)
	}
	if !hasProblemForSite(problems, stale[0].Site) {
		t.Fatalf("the guard must FAIL on a stale catalog entry whose site no longer exists (CLM-024), got %#v", problems)
	}
}

func hasProblemForFile(problems []checkTypeCatalogProblem, fileSubstr string) bool {
	for _, p := range problems {
		if strings.Contains(p.Site, fileSubstr) {
			return true
		}
	}
	return false
}

func hasProblemForSite(problems []checkTypeCatalogProblem, site string) bool {
	for _, p := range problems {
		if p.Site == site {
			return true
		}
	}
	return false
}
