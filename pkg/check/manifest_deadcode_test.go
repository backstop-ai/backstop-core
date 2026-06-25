package check

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// SPEC-039 REQ-010 deletion-assertion + surviving-API tests for the dead
// .manifest.json reader and the rule-matching path it strands. The
// deletion-assertion tests scan non-test .go sources under pkg/check and assert
// the named identifiers/branches are ABSENT — they are RED while the symbols
// exist and go GREEN only after the TASK-004 deletion, preventing silent
// reintroduction (model: pkg/validate/deletion_assertion_test.go).

// deadcodeNonTestGoSources returns the contents of every non-test .go file in
// the current package directory (tests run with CWD == the package dir), keyed
// by file name, so the deletion-assertions scan production source only.
func deadcodeNonTestGoSources(t *testing.T) map[string]string {
	t.Helper()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("reading package dir: %v", err)
	}
	out := map[string]string{}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		b, rerr := os.ReadFile(name)
		if rerr != nil {
			t.Fatalf("reading %s: %v", name, rerr)
		}
		out[name] = string(b)
	}
	return out
}

// TestCompiledManifestReader_Removed proves the compiledManifestFile type is
// gone from every non-test source under pkg/check. CLM-010.
func TestCompiledManifestReader_Removed(t *testing.T) {
	for name, src := range deadcodeNonTestGoSources(t) {
		if strings.Contains(src, "type compiledManifestFile") {
			t.Errorf("%s still declares `type compiledManifestFile`; the dead compiled-manifest reader must be deleted", name)
		}
	}
}

// TestCompiledManifestReader_MethodsRemoved proves the compiled-reader methods
// are gone from every non-test source under pkg/check. CLM-011.
func TestCompiledManifestReader_MethodsRemoved(t *testing.T) {
	methods := []string{"deriveRules", "isCompiled", "hasSemgrepSignal", "legacyRules", "routableExtensions"}
	for name, src := range deadcodeNonTestGoSources(t) {
		for _, m := range methods {
			if strings.Contains(src, m+"(") {
				t.Errorf("%s still references compiled-reader method %q; it must be deleted", name, m)
			}
		}
	}
}

// TestCompiledManifestReader_CombinedRuleAndLanguageExtensionsRemoved proves the
// combinedRule decode type and the languageExtensions map are gone from non-test
// source under pkg/check. CLM-012.
func TestCompiledManifestReader_CombinedRuleAndLanguageExtensionsRemoved(t *testing.T) {
	for name, src := range deadcodeNonTestGoSources(t) {
		if strings.Contains(src, "type combinedRule") {
			t.Errorf("%s still declares `type combinedRule`; it must be deleted", name)
		}
		if strings.Contains(src, "languageExtensions") {
			t.Errorf("%s still references `languageExtensions`; the map must be deleted", name)
		}
	}
}

// TestLoadManifest_NoCompiledManifestBranch proves LoadManifest's body does not
// decode a compiledManifestFile or branch on isCompiled(). CLM-013.
func TestLoadManifest_NoCompiledManifestBranch(t *testing.T) {
	for name, src := range deadcodeNonTestGoSources(t) {
		if strings.Contains(src, "compiledManifestFile") {
			t.Errorf("%s still references compiledManifestFile; LoadManifest must not decode it", name)
		}
		if strings.Contains(src, "isCompiled(") {
			t.Errorf("%s still branches on isCompiled(); the compiled branch must be deleted", name)
		}
	}
}

// TestLoadManifest_NoManifestJSONRead proves LoadManifest's body contains no
// `.manifest.json` suffix match / decode of a manifestFile. CLM-013.
func TestLoadManifest_NoManifestJSONRead(t *testing.T) {
	for name, src := range deadcodeNonTestGoSources(t) {
		if strings.Contains(src, `".manifest.json"`) {
			t.Errorf("%s still references the \".manifest.json\" suffix; LoadManifest must no longer read manifest files", name)
		}
		if strings.Contains(src, "manifestFile") {
			t.Errorf("%s still references the manifestFile decode struct; it must be deleted", name)
		}
	}
}

// TestLoadManifest_LegacyArmHelpersRemoved proves manifestFile, parseCheckTypes
// (PLURAL), and hasRoutableRule are absent from non-test source under pkg/check.
// CLM-019. parseCheckType (SINGULAR) must remain — it is asserted present by
// TestManifest_SurvivingAPIIntact.
func TestLoadManifest_LegacyArmHelpersRemoved(t *testing.T) {
	for name, src := range deadcodeNonTestGoSources(t) {
		if strings.Contains(src, "func parseCheckTypes(") {
			t.Errorf("%s still declares parseCheckTypes (plural); the legacy-arm helper must be deleted", name)
		}
		if strings.Contains(src, "func hasRoutableRule(") {
			t.Errorf("%s still declares hasRoutableRule; the legacy-arm helper must be deleted", name)
		}
	}
}

// TestRuleMatchingPath_Removed proves matchesRule, matchGlobPattern,
// matchDoubleStarPattern, and `type ManifestRule` are absent, and the Manifest
// struct declares no `rules` or `isDefaults` field. CLM-021.
func TestRuleMatchingPath_Removed(t *testing.T) {
	for name, src := range deadcodeNonTestGoSources(t) {
		for _, fn := range []string{"func matchesRule(", "func matchGlobPattern(", "func matchDoubleStarPattern("} {
			if strings.Contains(src, fn) {
				t.Errorf("%s still declares %q; the stranded rule-matching path must be deleted", name, fn)
			}
		}
		if strings.Contains(src, "type ManifestRule") {
			t.Errorf("%s still declares `type ManifestRule`; it must be deleted", name)
		}
		if strings.Contains(src, "[]ManifestRule") {
			t.Errorf("%s still references []ManifestRule; the Manifest.rules field and its type must be removed", name)
		}
		if strings.Contains(src, "isDefaults") {
			t.Errorf("%s still references isDefaults; the field must be removed and Manifest become empty", name)
		}
	}
}

// TestLoadManifest_ManifestJSONIgnoredReturnsDefaults proves a .manifest.json
// placed in a rules dir is IGNORED post-deletion: LoadManifest returns
// defaultManifest() routing (the whole arm is gone, no production producer).
// CLM-019. Written as a POST-deletion assertion grouped with the deletion suite:
// pre-deletion the reader would honor the file, so this is GREEN only after
// TASK-004.
func TestLoadManifest_ManifestJSONIgnoredReturnsDefaults(t *testing.T) {
	dir := t.TempDir()
	// A routing-schema manifest that, IF read, would route .custom → lint-only —
	// a route the default manifest never gives. Post-deletion it is ignored.
	planted := `{"rules":[{"extensions":[".custom"],"check_types":["lint"]}]}`
	if err := os.WriteFile(filepath.Join(dir, "routing.manifest.json"), []byte(planted), 0o644); err != nil {
		t.Fatalf("write planted manifest: %v", err)
	}

	m, err := LoadManifest(dir)
	if err != nil {
		t.Fatalf("LoadManifest with a planted .manifest.json: %v", err)
	}
	// The planted route must NOT take effect: .custom routes to nothing under the
	// default manifest (it is not .go/.ts/.tsx).
	if got := m.RouteFile("thing.custom"); len(got) != 0 {
		t.Errorf(".custom routed to %v; the planted .manifest.json must be IGNORED (default manifest in force)", got)
	}
	// And .go still routes to all four passes via the built-in defaults.
	if got := m.RouteFile("main.go"); len(got) != 4 {
		t.Errorf(".go routed to %v, want 4 default passes; LoadManifest must return defaultManifest()", got)
	}
}

// TestLoadManifest_RepoRulesDirStillRoutesGo proves LoadManifest over a path
// with no rules dir / no .manifest.json (this repo's shape) returns defaults and
// routes .go → {lint,build,test,findings}. CLM-014.
func TestLoadManifest_RepoRulesDirStillRoutesGo(t *testing.T) {
	dir := t.TempDir()
	rulesDir := filepath.Join(dir, ".backstop", "rules")
	// Deliberately do NOT create the rules dir — mirror the repo (no .backstop/rules).
	m, err := LoadManifest(rulesDir)
	if err != nil {
		t.Fatalf("LoadManifest over absent rules dir: %v", err)
	}
	checks := m.RouteFile("pkg/server/handler.go")
	want := []CheckType{CheckTypeLint, CheckTypeBuild, CheckTypeTest, CheckTypeFindings}
	if len(checks) != len(want) {
		t.Fatalf(".go routed to %v, want %v", checks, want)
	}
	for i, ct := range want {
		if checks[i] != ct {
			t.Errorf("checks[%d] = %v, want %v", i, checks[i], ct)
		}
	}
}

// TestLoadManifest_MissingDirReturnsDefaults proves LoadManifest over an absent
// / unreadable dir returns the default manifest with a nil error. CLM-015.
func TestLoadManifest_MissingDirReturnsDefaults(t *testing.T) {
	m, err := LoadManifest("/nonexistent/dir/that/does/not/exist")
	if err != nil {
		t.Fatalf("LoadManifest over a missing dir returned error %v; want defaults, nil", err)
	}
	if got := m.RouteFile("main.go"); len(got) != 4 {
		t.Errorf(".go routed to %v, want 4 default passes", got)
	}
}

// TestManifest_SurvivingAPIIntact proves the surviving manifest API is present
// and routes a .go file to all four passes: Manifest, RouteFile,
// routeFileDefaults, parseCheckType (SINGULAR), defaultManifest, the CheckType
// enum, and the ConfigError TYPE all compile and behave. CLM-016.
func TestManifest_SurvivingAPIIntact(t *testing.T) {
	// defaultManifest + RouteFile route a .go file to all four passes.
	m := defaultManifest()
	checks := m.RouteFile("main.go")
	want := []CheckType{CheckTypeLint, CheckTypeBuild, CheckTypeTest, CheckTypeFindings}
	if len(checks) != len(want) {
		t.Fatalf(".go routed to %v, want %v", checks, want)
	}
	for i, ct := range want {
		if checks[i] != ct {
			t.Errorf("checks[%d] = %v, want %v", i, checks[i], ct)
		}
	}
	// routeFileDefaults is reachable directly and agrees.
	if got := m.routeFileDefaults("main.go"); len(got) != 4 {
		t.Errorf("routeFileDefaults(.go) = %v, want 4 passes", got)
	}
	// parseCheckType (SINGULAR) survives.
	if ct, ok := parseCheckType("findings"); !ok || ct != CheckTypeFindings {
		t.Errorf("parseCheckType(\"findings\") = %v,%v, want CheckTypeFindings,true", ct, ok)
	}
	// The ConfigError TYPE survives and is constructible.
	ce := error(&ConfigError{Message: "x"})
	if ce.Error() == "" {
		t.Error("ConfigError.Error() returned empty; the ConfigError TYPE must survive")
	}
}
