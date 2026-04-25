package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/config"
)

// TestConfig_Struct_AllFields loads full-backstop.yml, verifies all struct
// fields populated (project, language, runtimes, enforcement, packs,
// registries). (CLM-032)
func TestConfig_Struct_AllFields(t *testing.T) {
	path := filepath.Join("..", "..", "cmd", "backstop", "testdata", "full-backstop.yml")
	cfg, err := config.LoadConfigFromPath(path)
	if err != nil {
		t.Fatalf("LoadConfigFromPath(%s) error: %v", path, err)
	}

	if cfg.Project != "my-service" {
		t.Errorf("Project = %q, want %q", cfg.Project, "my-service")
	}
	if cfg.Language != "go" {
		t.Errorf("Language = %q, want %q", cfg.Language, "go")
	}
	if len(cfg.Runtimes) != 2 {
		t.Errorf("Runtimes length = %d, want 2", len(cfg.Runtimes))
	}
	if cfg.Enforcement.Security.Tier != "standard" {
		t.Errorf("Enforcement.Security.Tier = %q, want %q", cfg.Enforcement.Security.Tier, "standard")
	}
	if cfg.Enforcement.WaiverWarningDays != 14 {
		t.Errorf("WaiverWarningDays = %d, want 14", cfg.Enforcement.WaiverWarningDays)
	}
	if len(cfg.Packs) == 0 {
		t.Error("Packs is empty, expected entries")
	}
	if cfg.Packs["owasp-go"] != "1.0.0" {
		t.Errorf("Packs[owasp-go] = %q, want %q", cfg.Packs["owasp-go"], "1.0.0")
	}
	if cfg.Packs["go-standards"] != "1.0.0" {
		t.Errorf("Packs[go-standards] = %q, want %q", cfg.Packs["go-standards"], "1.0.0")
	}
	if len(cfg.Registries) == 0 {
		t.Error("Registries is empty, expected entries")
	}
	if cfg.Registries["@company"] != "https://registry.company.com/backstop" {
		t.Errorf("Registries[@company] = %q, want %q", cfg.Registries["@company"], "https://registry.company.com/backstop")
	}
}

// TestConfig_Struct_RejectsUnknownKeys loads YAML with unknown top-level key,
// expects error. (CLM-033)
func TestConfig_Struct_RejectsUnknownKeys(t *testing.T) {
	path := filepath.Join("..", "..", "cmd", "backstop", "testdata", "unknown-keys-backstop.yml")
	_, err := config.LoadConfigFromPath(path)
	if err == nil {
		t.Fatal("expected error for unknown top-level keys, got nil")
	}
}

// TestConfig_LoaderValidatesAgainstSchema loads invalid YAML, expects schema
// validation error. (CLM-034)
func TestConfig_LoaderValidatesAgainstSchema(t *testing.T) {
	path := filepath.Join("..", "..", "cmd", "backstop", "testdata", "invalid-backstop.yml")
	_, err := config.LoadConfigFromPath(path)
	if err == nil {
		t.Fatal("expected schema validation error, got nil")
	}
}

// TestConfig_RequiredFields_ProjectAndLanguage verifies YAML missing project
// or language field fails. (CLM-035)
func TestConfig_RequiredFields_ProjectAndLanguage(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{"missing project", "language: go\n"},
		{"missing language", "project: my-svc\n"},
		{"missing both", "runtimes:\n  - go1.25\n"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "backstop.yml")
			if err := os.WriteFile(path, []byte(tc.content), 0o644); err != nil {
				t.Fatal(err)
			}
			_, err := config.LoadConfigFromPath(path)
			if err == nil {
				t.Fatalf("expected error for %s, got nil", tc.name)
			}
		})
	}
}

// TestConfig_Enforcement_ValidTier verifies enforcement block with tier
// "baseline", "standard", "compliance" accepted. (CLM-036)
func TestConfig_Enforcement_ValidTier(t *testing.T) {
	for _, tier := range []string{"baseline", "standard", "compliance"} {
		t.Run(tier, func(t *testing.T) {
			dir := t.TempDir()
			content := "project: svc\nlanguage: go\nenforcement:\n  security:\n    tier: " + tier + "\n"
			path := filepath.Join(dir, "backstop.yml")
			if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
			cfg, err := config.LoadConfigFromPath(path)
			if err != nil {
				t.Fatalf("valid tier %q rejected: %v", tier, err)
			}
			if cfg.Enforcement.Security.Tier != tier {
				t.Errorf("Tier = %q, want %q", cfg.Enforcement.Security.Tier, tier)
			}
		})
	}
}

// TestConfig_Enforcement_InvalidTier verifies enforcement block with tier
// "unknown" rejected. (CLM-037)
func TestConfig_Enforcement_InvalidTier(t *testing.T) {
	dir := t.TempDir()
	content := "project: svc\nlanguage: go\nenforcement:\n  security:\n    tier: unknown\n"
	path := filepath.Join(dir, "backstop.yml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := config.LoadConfigFromPath(path)
	if err == nil {
		t.Fatal("expected error for invalid tier, got nil")
	}
}

// TestConfig_Packs_ValidVersions verifies packs block with version strings
// accepted. (CLM-038)
func TestConfig_Packs_ValidVersions(t *testing.T) {
	dir := t.TempDir()
	content := `project: svc
language: go
packs:
  owasp-go: "1.0.0"
  go-standards: "2.0.0"
`
	path := filepath.Join(dir, "backstop.yml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.LoadConfigFromPath(path)
	if err != nil {
		t.Fatalf("valid packs rejected: %v", err)
	}
	if cfg.Packs["owasp-go"] != "1.0.0" {
		t.Errorf("Packs[owasp-go] = %q, want %q", cfg.Packs["owasp-go"], "1.0.0")
	}
	if cfg.Packs["go-standards"] != "2.0.0" {
		t.Errorf("Packs[go-standards] = %q, want %q", cfg.Packs["go-standards"], "2.0.0")
	}
}

// TestConfig_Registries_ScopeResolution verifies registries block with
// scope-to-URL mapping accepted. (CLM-039)
func TestConfig_Registries_ScopeResolution(t *testing.T) {
	dir := t.TempDir()
	content := `project: svc
language: go
registries:
  "@company": "https://registry.company.com/backstop"
`
	path := filepath.Join(dir, "backstop.yml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.LoadConfigFromPath(path)
	if err != nil {
		t.Fatalf("valid registries rejected: %v", err)
	}
	if cfg.Registries["@company"] != "https://registry.company.com/backstop" {
		t.Errorf("Registries[@company] = %q, want URL", cfg.Registries["@company"])
	}
}

// TestConfig_WaiverWarningDays_Default verifies omitted waiver_warning_days
// defaults to 30. (CLM-040)
func TestConfig_WaiverWarningDays_Default(t *testing.T) {
	dir := t.TempDir()
	content := "project: svc\nlanguage: go\n"
	path := filepath.Join(dir, "backstop.yml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.LoadConfigFromPath(path)
	if err != nil {
		t.Fatalf("valid config rejected: %v", err)
	}
	if cfg.Enforcement.WaiverWarningDays != 30 {
		t.Errorf("WaiverWarningDays = %d, want 30 (default)", cfg.Enforcement.WaiverWarningDays)
	}
}

// TestCLI_ConfigLoader_WalkUpDiscovery creates a temp dir tree, places
// backstop.yml in parent, runs discovery from child. (CLM-009)
func TestCLI_ConfigLoader_WalkUpDiscovery(t *testing.T) {
	parent := t.TempDir()
	child := filepath.Join(parent, "sub", "deep")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "project: walk-up-test\nlanguage: go\n"
	if err := os.WriteFile(filepath.Join(parent, "backstop.yml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	path, err := config.DiscoverConfigPathFrom(child)
	if err != nil {
		t.Fatalf("DiscoverConfigPath from child: %v", err)
	}
	actual, _ := filepath.EvalSymlinks(path)
	expected, _ := filepath.EvalSymlinks(filepath.Join(parent, "backstop.yml"))
	if actual != expected {
		t.Errorf("discovered path = %q (resolved %q), want %q", path, actual, expected)
	}
}

// TestCLI_ConfigLoader_NotFound_Exit2 verifies empty temp dir discovery
// returns error. (CLM-010)
func TestCLI_ConfigLoader_NotFound_Exit2(t *testing.T) {
	dir := t.TempDir()
	_, err := config.DiscoverConfigPathFrom(dir)
	if err == nil {
		t.Fatal("expected error when backstop.yml not found, got nil")
	}
}

// TestCLI_ConfigLoader_InvalidSchema_Exit2 verifies invalid backstop.yml
// loader returns validation error. (CLM-011)
func TestCLI_ConfigLoader_InvalidSchema_Exit2(t *testing.T) {
	dir := t.TempDir()
	content := "project: svc\nlanguage: go\nenforcement:\n  security:\n    tier: bad-tier\n"
	if err := os.WriteFile(filepath.Join(dir, "backstop.yml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := config.LoadConfigFromDir(dir)
	if err == nil {
		t.Fatal("expected validation error for invalid schema, got nil")
	}
}

// TestConfig_BackstopConfig_OverridesWalkUp sets env var to valid file,
// discovery uses that path instead of walk-up. (CLM-044)
func TestConfig_BackstopConfig_OverridesWalkUp(t *testing.T) {
	dir := t.TempDir()
	overridePath := filepath.Join(dir, "custom-backstop.yml")
	content := "project: override-test\nlanguage: go\n"
	if err := os.WriteFile(overridePath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("BACKSTOP_CONFIG", overridePath)
	// Use a different directory to prove walk-up is bypassed
	emptyDir := t.TempDir()
	path, err := config.DiscoverConfigPathFrom(emptyDir)
	if err != nil {
		t.Fatalf("DiscoverConfigPath with BACKSTOP_CONFIG: %v", err)
	}
	if path != overridePath {
		t.Errorf("discovered path = %q, want override %q", path, overridePath)
	}
}

// TestConfig_BackstopConfig_NonexistentFile_Exit2 sets env var to nonexistent
// path, expect error. (CLM-045)
func TestConfig_BackstopConfig_NonexistentFile_Exit2(t *testing.T) {
	t.Setenv("BACKSTOP_CONFIG", "/nonexistent/path/backstop.yml")
	_, err := config.DiscoverConfigPathFrom("")
	if err == nil {
		t.Fatal("expected error for nonexistent BACKSTOP_CONFIG path, got nil")
	}
}

// TestConfig_BackstopConfig_EmptyString_FallsBackToWalkUp sets env var to "",
// discovery falls back to walk-up. (CLM-046)
func TestConfig_BackstopConfig_EmptyString_FallsBackToWalkUp(t *testing.T) {
	parent := t.TempDir()
	child := filepath.Join(parent, "child")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "project: fallback-test\nlanguage: go\n"
	if err := os.WriteFile(filepath.Join(parent, "backstop.yml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("BACKSTOP_CONFIG", "")
	path, err := config.DiscoverConfigPathFrom(child)
	if err != nil {
		t.Fatalf("DiscoverConfigPath with empty BACKSTOP_CONFIG: %v", err)
	}
	actual, _ := filepath.EvalSymlinks(path)
	expected, _ := filepath.EvalSymlinks(filepath.Join(parent, "backstop.yml"))
	if actual != expected {
		t.Errorf("discovered path = %q (resolved %q), want %q", path, actual, expected)
	}
}

func TestConfig_LoadConfig_UsesCurrentDirectory(t *testing.T) {
	dir := t.TempDir()
	content := "project: cwd-test\nlanguage: go\n"
	if err := os.WriteFile(filepath.Join(dir, "backstop.yml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	orig, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(orig) }()

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}
	if cfg.Project != "cwd-test" {
		t.Fatalf("unexpected project: %s", cfg.Project)
	}
}

func TestConfig_DiscoverConfigPath_BackstopConfigDirectoryError(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("BACKSTOP_CONFIG", dir)
	_, err := config.DiscoverConfigPathFrom(t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "is a directory") {
		t.Fatalf("expected directory error, got: %v", err)
	}
}

func TestConfig_BackstopConfig_RelativePath_ResolvesFromCWD(t *testing.T) {
	base := t.TempDir()
	cfg := "project: rel-test\nlanguage: go\n"
	if err := os.WriteFile(filepath.Join(base, "backstop.yml"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}

	orig, _ := os.Getwd()
	if err := os.Chdir(base); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(orig) }()

	t.Setenv("BACKSTOP_CONFIG", "backstop.yml")
	path, err := config.DiscoverConfigPathFrom(t.TempDir())
	if err != nil {
		t.Fatalf("DiscoverConfigPath error: %v", err)
	}
	if filepath.Base(path) != "backstop.yml" {
		t.Fatalf("resolved file mismatch: %s", path)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("resolved path does not exist: %s (%v)", path, err)
	}
}
