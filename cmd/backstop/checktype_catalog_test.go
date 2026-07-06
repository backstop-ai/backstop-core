package main

import (
	"strings"
	"testing"
)

// TestCatalog_EnumeratesGateSemanticConsumersExcludesDisplaySites proves
// CheckTypeConsumerCatalog() enumerates the gate-semantic consumers (scope-filter,
// engine dispatch, violation verdict — C-1..C-8) and EXCLUDES cosmetic
// .Pass.String() display/serialization sites (pkg/check/output.go's render sites),
// so the catalog does not red on every added log line (SPEC-041 CLM-020).
func TestCatalog_EnumeratesGateSemanticConsumersExcludesDisplaySites(t *testing.T) {
	catalog := CheckTypeConsumerCatalog()
	if len(catalog) == 0 {
		t.Fatal("catalog must enumerate the gate-semantic CheckType consumers")
	}

	// The NEW C-1 declared engine-property bridge must be present (the sole live
	// ProjectWide locus).
	if !catalogHasSite(catalog, "cmd/backstop/pack_gate.go:runFindingsEngine") {
		t.Error("catalog must include the C-1 engine-path exempt bridge (CLM-020)")
	}

	// No cosmetic .Pass.String() display site (output.go) may appear in the catalog.
	for _, entry := range catalog {
		if strings.Contains(entry.Site, "output.go") {
			t.Errorf("catalog must EXCLUDE cosmetic display site %q — it carries no gate-semantic decision (CLM-020)", entry.Site)
		}
		if strings.Contains(entry.Site, ".Pass.String") {
			t.Errorf("catalog must EXCLUDE .Pass.String() display sites, found %q (CLM-020)", entry.Site)
		}
	}
}

// TestCatalog_SurvivingSitesNotMistaggedDeleted proves the surviving CheckType
// site (the findings stamping inside the LIVE ParsePackFindings — C-5) is tagged
// with its real post-cutover role, and that the sites DELETED by ISSUE-018 with
// the `backstop code check` command + in-process check engine (C-4
// code_check.go, C-6 check.go:passOrder, C-7 registry.go:Entries, C-8
// manifest.go:parseCheckType) — plus the earlier-deleted C-2/C-3 — are ABSENT
// from the live catalog (SPEC-041 CLM-021).
func TestCatalog_SurvivingSitesNotMistaggedDeleted(t *testing.T) {
	catalog := CheckTypeConsumerCatalog()

	survivors := []string{
		"pkg/check/parsers.go:CheckTypeFindings",
	}
	for _, site := range survivors {
		entry, ok := catalogEntry(catalog, site)
		if !ok {
			t.Errorf("surviving site %q must be cataloged (CLM-021)", site)
			continue
		}
		if entry.PostCutoverSource != SourceFindingsPackEngine {
			t.Errorf("surviving site %q mis-tagged %q — must be a surviving post-cutover role, NOT deleted (CLM-021)", site, entry.PostCutoverSource)
		}
	}

	// The sites deleted with the in-process check engine (ISSUE-018) must be ABSENT
	// from the live catalog, alongside the earlier-deleted C-2/C-3 — DELETED rows
	// are removed, not present-and-tagged-DELETED, so the stale-entry guard stays
	// green.
	deletedSites := []string{
		"cmd/backstop/code_check.go:CheckTypeFindings",
		"pkg/check/check.go:passOrder",
		"pkg/check/registry.go:Entries",
		"pkg/check/manifest.go:parseCheckType",
	}
	for _, site := range deletedSites {
		if catalogHasSite(catalog, site) {
			t.Errorf("DELETED site %q must be ABSENT from the live catalog after ISSUE-018 (CLM-021)", site)
		}
	}
	for _, entry := range catalog {
		if strings.Contains(entry.Site, "checkViolationsToGate") || strings.Contains(entry.Site, "sharedTestRunner") || strings.Contains(entry.Site, "newSharedTestRunner") {
			t.Errorf("DELETED site %q must be ABSENT from the live catalog (CLM-021)", entry.Site)
		}
		if entry.PostCutoverSource == "deleted" {
			t.Errorf("no live catalog entry may be tagged deleted (%q) — DELETED rows are removed, not retained (CLM-021)", entry.Site)
		}
	}
}

func catalogHasSite(catalog []CheckTypeConsumer, site string) bool {
	_, ok := catalogEntry(catalog, site)
	return ok
}

func catalogEntry(catalog []CheckTypeConsumer, site string) (CheckTypeConsumer, bool) {
	for _, e := range catalog {
		if e.Site == site {
			return e, true
		}
	}
	return CheckTypeConsumer{}, false
}
