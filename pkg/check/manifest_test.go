package check

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// copyFixture copies a testdata fixture file into dir under the given name.
// Used to stage real/embedded compiled-manifest fixtures into a tmpdir so
// LoadManifest reads them like production manifests.
func copyFixture(t *testing.T, dir, fixtureName, destName string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", fixtureName))
	if err != nil {
		t.Fatalf("read fixture %s: %v", fixtureName, err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, destName), data, 0o644); err != nil {
		t.Fatalf("write fixture %s: %v", destName, err)
	}
}

// writeRawManifest writes raw JSON bytes as a manifest file into dir. Used for
// inline compiled-schema literals that the typed manifestFile helper cannot
// express (compiled schema carries standard/language/enforcement fields).
func writeRawManifest(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("write raw manifest %s: %v", name, err)
	}
}

// containsCheckType reports whether checks contains target.
func containsCheckType(checks []CheckType, target CheckType) bool {
	for _, ct := range checks {
		if ct == target {
			return true
		}
	}
	return false
}

// TestCodeCheck_LoadManifest_DerivesRoutingFromCompiledManifest verifies that
// LoadManifest recognizes a compiled standards manifest (standard + language +
// rules[].enforcement) and derives language-based routing from it rather than
// treating its rules as zero routing rules. A legacy routing-schema manifest in
// the same dir must continue to coexist and load. (CLM-001)
func TestCodeCheck_LoadManifest_DerivesRoutingFromCompiledManifest(t *testing.T) {
	// The compiled manifest ALONE must derive non-empty go routing — NOT the
	// empty degenerate state that silently skips every pass, and NOT the
	// defaults fallback. Tested in isolation so the derivation, not a legacy
	// rule, is what produces the routing.
	t.Run("compiled_alone_derives_go_routing", func(t *testing.T) {
		dir := t.TempDir()
		copyFixture(t, dir, "compiled-std-go-001.manifest.json", "STD-GO-001.manifest.json")

		m, err := LoadManifest(dir)
		if err != nil {
			t.Fatalf("LoadManifest: %v", err)
		}
		if m == nil {
			t.Fatal("LoadManifest returned nil manifest")
		}
		if m.isDefaults {
			t.Error("manifest fell back to defaults; want routing derived from the compiled manifest")
		}

		checks := m.RouteFile("pkg/foo/bar.go")
		if len(checks) == 0 {
			t.Fatalf("compiled manifest derived empty routing for .go file; want derived go routing")
		}
		if !containsCheckType(checks, CheckTypeLint) ||
			!containsCheckType(checks, CheckTypeBuild) ||
			!containsCheckType(checks, CheckTypeTest) {
			t.Errorf("derived go routing = %v, want at least lint/build/test", checks)
		}
	})

	// A compiled manifest and a legacy routing-schema manifest must coexist in
	// the same dir: both formats load, and the dir routes .go files (proving the
	// legacy format still works alongside the new compiled-derivation path).
	t.Run("compiled_and_legacy_coexist", func(t *testing.T) {
		dir := t.TempDir()
		copyFixture(t, dir, "compiled-std-go-001.manifest.json", "STD-GO-001.manifest.json")
		copyFixture(t, dir, "routing-schema.manifest.json", "legacy.manifest.json")

		m, err := LoadManifest(dir)
		if err != nil {
			t.Fatalf("LoadManifest: %v", err)
		}
		if m.isDefaults {
			t.Error("manifest fell back to defaults with files present; want populated routing")
		}
		checks := m.RouteFile("pkg/foo/bar.go")
		if len(checks) == 0 {
			t.Fatalf("coexisting compiled+legacy manifests routed nothing for .go file")
		}
	})
}

// TestCodeCheck_LoadManifest_CompiledManifestRoutesGoFiles verifies RouteFile
// routes .go files under a compiled go-language manifest. A manifest carrying
// semgrep-enforced rules (or semgrep_config) routes .go files to
// lint/build/test/semgrep; a go manifest with no semgrep signal routes to
// lint/build/test WITHOUT semgrep. Unknown-language manifests with a semgrep
// signal route any file to semgrep-only via "**"; with no semgrep signal they
// contribute no routing. (CLM-002)
func TestCodeCheck_LoadManifest_CompiledManifestRoutesGoFiles(t *testing.T) {
	// Compiled go manifest carrying semgrep rules + semgrep_config -> all four.
	t.Run("go_with_semgrep_signal", func(t *testing.T) {
		dir := t.TempDir()
		copyFixture(t, dir, "compiled-std-go-001.manifest.json", "STD-GO-001.manifest.json")

		m, err := LoadManifest(dir)
		if err != nil {
			t.Fatalf("LoadManifest: %v", err)
		}
		checks := m.RouteFile("pkg/foo/bar.go")
		want := []CheckType{CheckTypeLint, CheckTypeBuild, CheckTypeTest, CheckTypeSemgrep}
		if len(checks) != len(want) {
			t.Fatalf("got %d check types, want %d: %v", len(checks), len(want), checks)
		}
		for i, ct := range want {
			if checks[i] != ct {
				t.Errorf("checks[%d] = %v, want %v (full set %v)", i, checks[i], ct, checks)
			}
		}
	})

	// Compiled go manifest with NO semgrep signal -> lint/build/test, no semgrep.
	t.Run("go_without_semgrep_signal", func(t *testing.T) {
		dir := t.TempDir()
		copyFixture(t, dir, "compiled-go-no-semgrep.manifest.json", "STD-GO-NATIVE.manifest.json")

		m, err := LoadManifest(dir)
		if err != nil {
			t.Fatalf("LoadManifest: %v", err)
		}
		checks := m.RouteFile("pkg/foo/bar.go")
		want := []CheckType{CheckTypeLint, CheckTypeBuild, CheckTypeTest}
		if len(checks) != len(want) {
			t.Fatalf("got %d check types, want %d (no semgrep): %v", len(checks), len(want), checks)
		}
		for i, ct := range want {
			if checks[i] != ct {
				t.Errorf("checks[%d] = %v, want %v (full set %v)", i, checks[i], ct, checks)
			}
		}
		if containsCheckType(checks, CheckTypeSemgrep) {
			t.Errorf("go manifest with no semgrep signal routed semgrep: %v", checks)
		}
	})

	// Unknown language WITH a semgrep signal -> semgrep-only for any file via **.
	t.Run("unknown_language_with_semgrep", func(t *testing.T) {
		dir := t.TempDir()
		writeRawManifest(t, dir, "STD-PY-001.manifest.json", `{
  "standard": "STD-PY-001",
  "language": "python",
  "semgrep_config": "STD-PY-001.semgrep.yml",
  "rules": [
    {"id": "PY-001", "name": "no-eval", "enforcement": "semgrep"}
  ]
}`)

		m, err := LoadManifest(dir)
		if err != nil {
			t.Fatalf("LoadManifest: %v", err)
		}
		checks := m.RouteFile("scripts/app.py")
		if len(checks) != 1 || checks[0] != CheckTypeSemgrep {
			t.Errorf("unknown-language semgrep manifest routed %v, want [semgrep] only", checks)
		}
		// Files of any other extension also route to semgrep-only via **.
		other := m.RouteFile("docs/notes.rst")
		if len(other) != 1 || other[0] != CheckTypeSemgrep {
			t.Errorf("unknown-language semgrep manifest routed %v for .rst, want [semgrep] only", other)
		}
	})

	// Unknown language with NO semgrep signal contributes no routing. Pair it
	// with a go manifest so the dir is overall routable (the all-empty case is
	// the zero-routable config error covered in CLM-003); assert the unknown
	// manifest adds nothing while the go manifest still routes .go files.
	t.Run("unknown_language_without_semgrep_contributes_nothing", func(t *testing.T) {
		dir := t.TempDir()
		copyFixture(t, dir, "compiled-go-no-semgrep.manifest.json", "STD-GO-NATIVE.manifest.json")
		writeRawManifest(t, dir, "STD-PY-002.manifest.json", `{
  "standard": "STD-PY-002",
  "language": "python",
  "rules": [
    {"id": "PY-002", "name": "max-file-length", "enforcement": "native"}
  ]
}`)

		m, err := LoadManifest(dir)
		if err != nil {
			t.Fatalf("LoadManifest: %v", err)
		}
		// Unknown-language native-only manifest derives no routing for its files.
		pyChecks := m.RouteFile("scripts/app.py")
		if len(pyChecks) != 0 {
			t.Errorf("unknown-language no-signal manifest routed %v for .py, want none", pyChecks)
		}
		// The go manifest still routes .go files to lint/build/test.
		goChecks := m.RouteFile("pkg/foo/bar.go")
		if len(goChecks) != 3 {
			t.Errorf("go routing = %v, want lint/build/test (3)", goChecks)
		}
	})
}

// TestCodeCheck_LoadManifest_NoRoutableRulesIsConfigError verifies that
// manifest files which parse but yield zero routable rules surface a config
// error (a *check.ConfigError), never a silently empty route table that skips
// every pass as a green result. The no-manifests case (empty dir) must still
// return the defaults manifest with a nil error — the fail-loud rule only fires
// when files exist but nothing routes. (CLM-003)
func TestCodeCheck_LoadManifest_NoRoutableRulesIsConfigError(t *testing.T) {
	t.Run("zero_routable_is_config_error", func(t *testing.T) {
		dir := t.TempDir()
		copyFixture(t, dir, "zero-routable.manifest.json", "empty.manifest.json")

		m, err := LoadManifest(dir)
		if err == nil {
			t.Fatalf("LoadManifest returned nil error for zero-routable manifest; want config error (got manifest %+v)", m)
		}
		if m != nil {
			t.Errorf("LoadManifest returned non-nil manifest %+v alongside error; want nil manifest", m)
		}
		var cfgErr *ConfigError
		if !errors.As(err, &cfgErr) {
			t.Fatalf("error %T (%v) is not a *check.ConfigError", err, err)
		}
	})

	// An unknown-language compiled manifest with no semgrep signal — derivable
	// to nothing — is also zero-routable and must be a config error, not a
	// silent skip.
	t.Run("unknown_language_no_signal_alone_is_config_error", func(t *testing.T) {
		dir := t.TempDir()
		writeRawManifest(t, dir, "STD-PY-003.manifest.json", `{
  "standard": "STD-PY-003",
  "language": "python",
  "rules": [
    {"id": "PY-003", "name": "max-file-length", "enforcement": "native"}
  ]
}`)

		_, err := LoadManifest(dir)
		var cfgErr *ConfigError
		if !errors.As(err, &cfgErr) {
			t.Fatalf("unknown-language no-signal manifest: error %T (%v) is not a *check.ConfigError", err, err)
		}
	})

	// The no-manifests case (empty dir) must STILL return defaults with nil error.
	t.Run("empty_dir_still_returns_defaults", func(t *testing.T) {
		dir := t.TempDir()
		m, err := LoadManifest(dir)
		if err != nil {
			t.Fatalf("LoadManifest on empty dir returned error %v; want defaults, nil", err)
		}
		checks := m.RouteFile("main.go")
		if len(checks) != 4 {
			t.Errorf("empty dir defaults: got %d checks for .go, want 4: %v", len(checks), checks)
		}
	})
}

// TestCodeCheck_LoadManifest_CompilerOutputContractRoundTrip pins the
// producer→consumer boundary: the real compiler-emitted manifest schema
// (embedded VERBATIM from .backstop/rules/STD-GO-001.manifest.json) must
// round-trip through LoadManifest into non-empty routing for the standard's
// language. A .go path must route to a non-empty check set including semgrep
// (the fixture carries semgrep-enforced rules) plus lint/build/test. This is
// the contract that would have caught the original schema drift. (CLM-004)
func TestCodeCheck_LoadManifest_CompilerOutputContractRoundTrip(t *testing.T) {
	dir := t.TempDir()
	copyFixture(t, dir, "compiled-std-go-001.manifest.json", "STD-GO-001.manifest.json")

	m, err := LoadManifest(dir)
	if err != nil {
		t.Fatalf("LoadManifest on real compiler output: %v", err)
	}
	if m == nil {
		t.Fatal("LoadManifest returned nil manifest for real compiler output")
	}

	checks := m.RouteFile("pkg/check/manifest.go")
	if len(checks) == 0 {
		t.Fatal("real compiler manifest round-tripped to EMPTY routing for .go file")
	}
	if !containsCheckType(checks, CheckTypeSemgrep) {
		t.Errorf("routing %v missing semgrep; the fixture carries semgrep-enforced rules", checks)
	}
	if !containsCheckType(checks, CheckTypeLint) ||
		!containsCheckType(checks, CheckTypeBuild) ||
		!containsCheckType(checks, CheckTypeTest) {
		t.Errorf("routing %v missing one of lint/build/test", checks)
	}
}

// writeManifest is a test helper that writes a manifest JSON file to dir.
func writeManifest(t *testing.T, dir string, name string, m manifestFile) {
	t.Helper()
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
}

// TestCodeCheck_Routing_GoFileAllPasses verifies that .go files route to all
// four passes per manifest. (CLM-014)
func TestCodeCheck_Routing_GoFileAllPasses(t *testing.T) {
	dir := t.TempDir()
	writeManifest(t, dir, "go.manifest.json", manifestFile{
		Rules: []ManifestRule{
			{
				Extensions: []string{".go"},
				CheckTypes: []string{"lint", "build", "test", "semgrep"},
			},
		},
	})

	m, err := LoadManifest(dir)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}

	checks := m.RouteFile("pkg/handler.go")
	if len(checks) != 4 {
		t.Fatalf("got %d check types, want 4: %v", len(checks), checks)
	}
	want := []CheckType{CheckTypeLint, CheckTypeBuild, CheckTypeTest, CheckTypeSemgrep}
	for i, ct := range want {
		if checks[i] != ct {
			t.Errorf("checks[%d] = %v, want %v", i, checks[i], ct)
		}
	}
}

// TestCodeCheck_Routing_UnmatchedFileNoChecks verifies that files matching
// no manifest entry receive no checks. (CLM-015)
func TestCodeCheck_Routing_UnmatchedFileNoChecks(t *testing.T) {
	dir := t.TempDir()
	writeManifest(t, dir, "go.manifest.json", manifestFile{
		Rules: []ManifestRule{
			{
				Extensions: []string{".go"},
				CheckTypes: []string{"lint", "build", "test", "semgrep"},
			},
		},
	})

	m, err := LoadManifest(dir)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}

	checks := m.RouteFile("README.md")
	if len(checks) != 0 {
		t.Errorf("got %d check types for README.md, want 0: %v", len(checks), checks)
	}
}

// TestCodeCheck_Routing_DefaultsWhenNoManifest verifies built-in defaults
// apply when no manifest files exist. (CLM-016)
func TestCodeCheck_Routing_DefaultsWhenNoManifest(t *testing.T) {
	dir := t.TempDir() // empty dir, no manifests

	m, err := LoadManifest(dir)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}

	// Go files should get all 4 passes
	goChecks := m.RouteFile("main.go")
	if len(goChecks) != 4 {
		t.Errorf("Go file: got %d checks, want 4: %v", len(goChecks), goChecks)
	}

	// Non-Go files should get semgrep only
	pyChecks := m.RouteFile("script.py")
	if len(pyChecks) != 1 || pyChecks[0] != CheckTypeSemgrep {
		t.Errorf("Python file: got %v, want [semgrep]", pyChecks)
	}
}

// TestCodeCheck_Routing_PathPatternMatching verifies manifest path patterns
// route files to correct check types. (CLM-017)
func TestCodeCheck_Routing_PathPatternMatching(t *testing.T) {
	dir := t.TempDir()
	writeManifest(t, dir, "patterns.manifest.json", manifestFile{
		Rules: []ManifestRule{
			{
				PathPatterns: []string{"pkg/**/*.go"},
				CheckTypes:   []string{"lint", "build", "test", "semgrep"},
			},
			{
				PathPatterns: []string{"scripts/*.sh"},
				CheckTypes:   []string{"semgrep"},
			},
		},
	})

	m, err := LoadManifest(dir)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}

	// File matching pkg/**/*.go pattern
	goChecks := m.RouteFile("pkg/handler/serve.go")
	if len(goChecks) != 4 {
		t.Errorf("pkg Go file: got %d checks, want 4: %v", len(goChecks), goChecks)
	}

	// File matching scripts/*.sh pattern
	shChecks := m.RouteFile("scripts/deploy.sh")
	if len(shChecks) != 1 || shChecks[0] != CheckTypeSemgrep {
		t.Errorf("shell file: got %v, want [semgrep]", shChecks)
	}

	// File matching nothing
	noChecks := m.RouteFile("docs/readme.md")
	if len(noChecks) != 0 {
		t.Errorf("unmatched file: got %v, want []", noChecks)
	}
}

// TestCodeCheck_Routing_MultipleExtensions verifies files with multiple
// extensions (e.g., foo.test.go) route correctly.
func TestCodeCheck_Routing_MultipleExtensions(t *testing.T) {
	dir := t.TempDir()
	writeManifest(t, dir, "go.manifest.json", manifestFile{
		Rules: []ManifestRule{
			{
				Extensions: []string{".go"},
				CheckTypes: []string{"lint", "build", "test", "semgrep"},
			},
		},
	})

	m, err := LoadManifest(dir)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}

	checks := m.RouteFile("pkg/handler_test.go")
	if len(checks) != 4 {
		t.Errorf("foo_test.go: got %d checks, want 4", len(checks))
	}
}

// TestCodeCheck_Routing_LoadFromNonExistentDir verifies defaults when dir
// does not exist.
func TestCodeCheck_Routing_LoadFromNonExistentDir(t *testing.T) {
	m, err := LoadManifest("/nonexistent/dir/that/does/not/exist")
	if err != nil {
		t.Fatalf("LoadManifest should return defaults for missing dir: %v", err)
	}
	// Should use defaults
	goChecks := m.RouteFile("main.go")
	if len(goChecks) != 4 {
		t.Errorf("Go file: got %d checks, want 4 (defaults)", len(goChecks))
	}
}

// TestCodeCheck_Routing_CheckTypeString verifies CheckType.String.
func TestCodeCheck_Routing_CheckTypeString(t *testing.T) {
	tests := []struct {
		ct   CheckType
		want string
	}{
		{CheckTypeLint, "lint"},
		{CheckTypeBuild, "build"},
		{CheckTypeTest, "test"},
		{CheckTypeSemgrep, "semgrep"},
		{CheckType(99), "unknown(99)"},
	}
	for _, tc := range tests {
		if got := tc.ct.String(); got != tc.want {
			t.Errorf("CheckType(%d).String() = %q, want %q", tc.ct, got, tc.want)
		}
	}
}

// TestCodeCheck_Routing_ParseCheckType verifies parseCheckType.
func TestCodeCheck_Routing_ParseCheckType(t *testing.T) {
	tests := []struct {
		input string
		want  CheckType
		ok    bool
	}{
		{"lint", CheckTypeLint, true},
		{"BUILD", CheckTypeBuild, true},
		{"Test", CheckTypeTest, true},
		{"semgrep", CheckTypeSemgrep, true},
		{"unknown", 0, false},
	}
	for _, tc := range tests {
		got, ok := parseCheckType(tc.input)
		if ok != tc.ok || (ok && got != tc.want) {
			t.Errorf("parseCheckType(%q) = %v,%v, want %v,%v", tc.input, got, ok, tc.want, tc.ok)
		}
	}
}

// TestCodeCheck_Routing_InvalidJSON verifies LoadManifest returns error
// for malformed JSON manifest files.
func TestCodeCheck_Routing_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "bad.manifest.json"), []byte("{not json!}"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadManifest(dir)
	if err == nil {
		t.Fatal("expected error for invalid JSON manifest, got nil")
	}
}

// TestCodeCheck_Routing_EmptyDir verifies LoadManifest returns defaults
// when the directory exists but has no manifest files.
func TestCodeCheck_Routing_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	m, err := LoadManifest(dir)
	if err != nil {
		t.Fatalf("LoadManifest on empty dir: %v", err)
	}
	// Should use defaults — Go files get all 4
	checks := m.RouteFile("main.go")
	if len(checks) != 4 {
		t.Errorf("expected 4 checks for .go with defaults, got %d", len(checks))
	}
}

// TestCodeCheck_Routing_MatchGlobPattern_EdgeCases verifies matchGlobPattern
// handles various edge cases correctly.
func TestCodeCheck_Routing_MatchGlobPattern_EdgeCases(t *testing.T) {
	tests := []struct {
		path    string
		pattern string
		want    bool
	}{
		// Simple wildcard
		{"main.go", "*.go", true},
		{"main.py", "*.go", false},
		// No wildcard — exact match
		{"Makefile", "Makefile", true},
		{"makefile", "Makefile", false},
		// Invalid pattern — should return false, not panic
		{"main.go", "[invalid", false},
		// Double star patterns
		{"pkg/foo/bar.go", "pkg/**/*.go", true},
		{"pkg/bar.go", "pkg/**/*.go", true},
		// Empty pattern
		{"main.go", "", false},
		// Empty path with pattern
		{"", "*.go", false},
	}
	for _, tc := range tests {
		got := matchGlobPattern(tc.path, tc.pattern)
		if got != tc.want {
			t.Errorf("matchGlobPattern(%q, %q) = %v, want %v", tc.path, tc.pattern, got, tc.want)
		}
	}
}

// TestCodeCheck_Routing_MatchDoubleStarPattern_EdgeCases verifies
// matchDoubleStarPattern handles various edge cases correctly.
func TestCodeCheck_Routing_MatchDoubleStarPattern_EdgeCases(t *testing.T) {
	tests := []struct {
		path    string
		pattern string
		want    bool
	}{
		// Basic double star
		{"pkg/handler/serve.go", "pkg/**/*.go", true},
		// Double star at end — matches everything under prefix
		{"pkg/anything/here", "pkg/**", true},
		// Double star at start — matches anything ending with suffix
		{"deeply/nested/file.go", "**/*.go", true},
		// No match
		{"pkg/handler/serve.py", "pkg/**/*.go", false},
		// Single segment after prefix
		{"pkg/serve.go", "pkg/**/*.go", true},
		// Empty suffix after **
		{"anything", "**", true},
		// Prefix with no match
		{"src/handler.go", "pkg/**/*.go", false},
		// Multiple segments deep
		{"a/b/c/d/e.go", "a/**/*.go", true},
	}
	for _, tc := range tests {
		got := matchDoubleStarPattern(tc.path, tc.pattern)
		if got != tc.want {
			t.Errorf("matchDoubleStarPattern(%q, %q) = %v, want %v", tc.path, tc.pattern, got, tc.want)
		}
	}
}

// TestCodeCheck_Routing_ReadError verifies LoadManifest returns error when
// a manifest file exists but cannot be read.
func TestCodeCheck_Routing_ReadError(t *testing.T) {
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "bad.manifest.json")
	// Create a manifest file with no read permission
	if err := os.WriteFile(manifestPath, []byte(`{"rules":[]}`), 0o000); err != nil {
		t.Fatal(err)
	}
	// Ensure cleanup restores permissions so t.TempDir can remove it
	t.Cleanup(func() { os.Chmod(manifestPath, 0o644) })

	_, err := LoadManifest(dir)
	if err == nil {
		t.Skip("test requires OS-level permission enforcement (may not work as root)")
	}
}

// TestCodeCheck_Routing_NoExtension verifies files with no extension
// fall through to path pattern matching or defaults.
func TestCodeCheck_Routing_NoExtension(t *testing.T) {
	dir := t.TempDir() // empty, uses defaults

	m, err := LoadManifest(dir)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}

	// No extension, non-Go: should get semgrep via defaults
	checks := m.RouteFile("Makefile")
	if len(checks) != 1 || checks[0] != CheckTypeSemgrep {
		t.Errorf("Makefile: got %v, want [semgrep]", checks)
	}
}
