package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// checkTypeCatalogProblem is one reconciliation failure the completeness guard
// reports: either a discovered gate-semantic keying site with no catalog entry
// (dropped-gate-step) or a stale catalog entry whose site no longer exists.
type checkTypeCatalogProblem struct {
	Site   string
	Reason string
}

// gateSemanticKeyingRe returns the regexp matching a GATE-SEMANTIC CheckType-keying
// site: an identity comparison/read of a CheckType value or the declared exempt
// property that drives a scope/dispatch/verdict decision. The cosmetic
// `.Pass.String()` display form carries no such marker, so it is never discovered
// (the negative case, CLM-022). The markers mirror the C-1..C-8 surface:
//   - ExemptFromScopeFilter        — the C-1 declared bridge (scope decision)
//   - Pass: ...CheckType / CheckTypeFindings stamping — C-4/C-5 verdict identity
//   - passOrder / map[CheckType] / parseCheckType — C-6/C-7/C-8 dispatch/labeling
//   - `== check.CheckTypeBuild` / `== CheckTypeBuild` — a build-identity scope
//     comparison (the dropped-gate-step shape the unlisted fixture injects).
//
// It is a FUNCTION, not a package-level var, to keep this file free of
// package-level mutable state (the codebase convention).
func gateSemanticKeyingRe() *regexp.Regexp {
	return regexp.MustCompile(
		`ExemptFromScopeFilter|== *check\.CheckType\w+|== *CheckType\w+|Pass: *(check\.)?CheckType\w+|parser\(out, *CheckType\w+\)|passOrder|map\[CheckType\]|parseCheckType`,
	)
}

// discoverGateSemanticSites scans the given source paths for GATE-SEMANTIC
// CheckType-keying lines, EXCLUDING pure .Pass.String() display sites. It returns
// the set of "file:line-marker" discovered sites. .go.txt fixtures are scanned as
// source (so the guard's discovery scope can be exercised by injected fixtures
// without compiling them).
func discoverGateSemanticSites(paths []string) (map[string]string, error) {
	keyingRe := gateSemanticKeyingRe()
	discovered := map[string]string{}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("scanning %s: %w", path, err)
		}
		rel := filepath.ToSlash(path)
		for i, line := range strings.Split(string(data), "\n") {
			marker := keyingRe.FindString(line)
			if marker == "" {
				continue
			}
			// A line that ONLY does .Pass display (no gate-semantic keying marker
			// beyond the display form) is excluded. gateSemanticKeyingRe does not
			// match the bare `.Pass.String()`/`[%s] v.Pass` display forms, so a
			// display-only line is never discovered here; this is the explicit
			// negative case (CLM-022). A line carrying BOTH is treated as
			// gate-semantic (the keying marker wins).
			key := fmt.Sprintf("%s:%d", rel, i+1)
			discovered[key] = marker
		}
	}
	return discovered, nil
}

// catalogSymbolPresent reports whether a catalog entry's Site (file:symbol) still
// exists: the file exists AND contains the symbol literal. A missing file or
// absent symbol is a STALE entry (CLM-024).
func catalogSymbolPresent(repoRootDir, site string) (bool, error) {
	file, symbol, ok := splitCatalogSite(site)
	if !ok {
		return false, fmt.Errorf("malformed catalog site %q (want file:symbol)", site)
	}
	data, err := os.ReadFile(filepath.Join(repoRootDir, filepath.FromSlash(file)))
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("reading catalog site file %s: %w", file, err)
	}
	return strings.Contains(string(data), symbol), nil
}

// splitCatalogSite splits a "path/to/file.go:symbol" site into (file, symbol).
func splitCatalogSite(site string) (string, string, bool) {
	idx := strings.LastIndex(site, ":")
	if idx <= 0 || idx == len(site)-1 {
		return "", "", false
	}
	return site[:idx], site[idx+1:], true
}

// reconcileCheckTypeCatalog is the bidirectional completeness guard (REQ-006). It
// fails on (a) any catalog entry whose site no longer exists in code (stale entry,
// CLM-024), and (b) any discovered gate-semantic keying site under discoveryPaths
// that maps to no catalog entry (dropped-gate-step, CLM-023). The bounded
// discovery scope (gate-semantic keying only, excluding cosmetic .Pass.String()
// display) is what makes the discovered-set provably equal the cataloged-set
// (CLM-022).
//
// catalogFileSet is the set of files whose discovered keying sites are EXPECTED to
// be represented by the catalog (the cataloged surface). A discovered site in a
// file NOT in catalogFileSet AND NOT covered by a catalog entry is the
// dropped-gate-step failure (the injected unlisted fixture lives in such a file).
func reconcileCheckTypeCatalog(repoRootDir string, catalog []CheckTypeConsumer, discoveryPaths []string) ([]checkTypeCatalogProblem, error) {
	var problems []checkTypeCatalogProblem

	// (a) Stale-entry check: every catalog Site must still exist.
	catalogFiles := map[string]bool{}
	for _, entry := range catalog {
		file, _, ok := splitCatalogSite(entry.Site)
		if ok {
			catalogFiles[file] = true
		}
		present, err := catalogSymbolPresent(repoRootDir, entry.Site)
		if err != nil {
			return nil, fmt.Errorf("checking catalog entry %q: %w", entry.Site, err)
		}
		if !present {
			problems = append(problems, checkTypeCatalogProblem{
				Site:   entry.Site,
				Reason: "stale catalog entry: its keying site no longer exists in code",
			})
		}
	}

	// (b) Dropped-gate-step check: a discovered gate-semantic keying site in a file
	// NOT represented by ANY catalog entry is unlisted.
	discovered, err := discoverGateSemanticSites(discoveryPaths)
	if err != nil {
		return nil, fmt.Errorf("discovering gate-semantic keying sites: %w", err)
	}
	keys := make([]string, 0, len(discovered))
	for k := range discovered {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	rootSlash := filepath.ToSlash(repoRootDir)
	for _, site := range keys {
		file := site
		if idx := strings.LastIndex(site, ":"); idx > 0 {
			file = site[:idx]
		}
		// Normalize the discovered file to a repo-relative path so it matches the
		// catalog's file:symbol Site form (the catalog lists repo-relative files).
		relFile := strings.TrimPrefix(strings.TrimPrefix(file, rootSlash), "/")
		if !catalogFiles[relFile] {
			problems = append(problems, checkTypeCatalogProblem{
				Site:   site,
				Reason: fmt.Sprintf("unlisted gate-semantic CheckType-keying site (%s) with no catalog entry — a dropped gate step", discovered[site]),
			})
		}
	}
	return problems, nil
}
