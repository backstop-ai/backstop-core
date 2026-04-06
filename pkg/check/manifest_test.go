package check

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

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
