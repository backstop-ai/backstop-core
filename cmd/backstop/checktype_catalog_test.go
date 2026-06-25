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

// TestCatalog_SurvivingSitesNotMistaggedDeleted proves the surviving pkg/check
// CheckType sites (passOrder, Violation.Pass/PassResult.Pass, Executors/dispatch,
// registry Entries, manifest enum+routing, parsers findings stamping — C-4..C-8)
// are tagged SURVIVING with their real post-cutover role, NOT mis-tagged DELETED;
// the genuinely-DELETED sites (C-2 orphaned gate.go:1173 exemption, C-3
// shared-runner feeds) are ABSENT from the live catalog (SPEC-041 CLM-021).
func TestCatalog_SurvivingSitesNotMistaggedDeleted(t *testing.T) {
	catalog := CheckTypeConsumerCatalog()

	survivors := []string{
		"pkg/check/check.go:passOrder",
		"pkg/check/registry.go:Entries",
		"pkg/check/manifest.go:parseCheckType",
		"pkg/check/parsers.go:CheckTypeFindings",
		"cmd/backstop/code_check.go:CheckTypeFindings",
	}
	for _, site := range survivors {
		entry, ok := catalogEntry(catalog, site)
		if !ok {
			t.Errorf("surviving site %q must be cataloged (CLM-021)", site)
			continue
		}
		switch entry.PostCutoverSource {
		case SourceSurvivingCheckTypeDispatch, SourceFindingsPackEngine:
			// correct surviving role
		default:
			t.Errorf("surviving site %q mis-tagged %q — must be a surviving post-cutover role, NOT deleted (CLM-021)", site, entry.PostCutoverSource)
		}
	}

	// No live catalog entry may be tagged with the deleted-site sources: the
	// genuinely-deleted C-2/C-3 rows are ABSENT, not present-and-tagged-DELETED, so
	// the stale-entry guard stays green.
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
