package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/backstop-ai/backstop-core/pkg/config"
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
	// SPEC-046: the `language` field is retired; the full-backstop.yml fixture may
	// still carry a `language:` key but it is now an inert, ignored legacy key.
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

// TestConfig_RequiredFields_ProjectAndLanguage verifies YAML missing the project
// field fails. (CLM-035) SPEC-046: `language` is RETIRED and no longer required —
// a config missing only `language` parses cleanly, so only project-missing cases
// remain failing (the `language: go` line in the first case is now an inert key).
func TestConfig_RequiredFields_ProjectAndLanguage(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{"missing project", "language: go\n"},
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

func TestConfig_BaselineTTLDuration(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *config.Config
		want    time.Duration
		wantErr bool
	}{
		{name: "nil config", cfg: nil, want: 15 * time.Minute},
		{name: "empty default", cfg: &config.Config{}, want: 15 * time.Minute},
		{name: "configured", cfg: &config.Config{Enforcement: config.Enforcement{BaselineTTL: "1h30m"}}, want: 90 * time.Minute},
		{name: "invalid", cfg: &config.Config{Enforcement: config.Enforcement{BaselineTTL: "nope"}}, wantErr: true},
		{name: "nonpositive", cfg: &config.Config{Enforcement: config.Enforcement{BaselineTTL: "0s"}}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.cfg.BaselineTTLDuration()
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("duration = %s, want %s", got, tt.want)
			}
		})
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

// writeTempConfig writes content to a temp backstop.yml and returns its path.
func writeTempConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "backstop.yml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp backstop.yml: %v", err)
	}
	return path
}

// TestConfig_Enforcement_ToolchainAndTestCommand verifies enforcement.toolchain
// (per-pass {command, format, extensions, test_dependency_command}) and
// enforcement.test_command decode into the Enforcement struct: the strict
// KnownFields decoder accepts the new keys and the JSON schema allows them under
// additionalProperties:false. (ISSUE-003 schema extension; CLM-004)
func TestConfig_Enforcement_ToolchainAndTestCommand(t *testing.T) {
	content := `project: ts-example
language: typescript
enforcement:
  test_command: "vitest run"
  toolchain:
    lint:
      command: "eslint --format json"
      format: eslint-json
      extensions: [".ts", ".tsx"]
    build:
      command: "tsc --noEmit"
      format: tsc
    test:
      command: "vitest run"
      format: regex-lines
      test_dependency_command: "vitest related"
`
	path := writeTempConfig(t, content)
	cfg, err := config.LoadConfigFromPath(path)
	if err != nil {
		t.Fatalf("LoadConfigFromPath: %v", err)
	}
	if cfg.Enforcement.TestCommand != "vitest run" {
		t.Errorf("TestCommand = %q, want 'vitest run'", cfg.Enforcement.TestCommand)
	}
	if len(cfg.Enforcement.Toolchain) != 3 {
		t.Fatalf("Toolchain has %d passes, want 3", len(cfg.Enforcement.Toolchain))
	}
	lint := cfg.Enforcement.Toolchain["lint"]
	if lint.Command != "eslint --format json" || lint.Format != "eslint-json" {
		t.Errorf("lint pass = %+v, want eslint command/format", lint)
	}
	if len(lint.Extensions) != 2 || lint.Extensions[0] != ".ts" {
		t.Errorf("lint extensions = %v, want [.ts .tsx]", lint.Extensions)
	}
	test := cfg.Enforcement.Toolchain["test"]
	if test.TestDependencyCommand != "vitest related" {
		t.Errorf("test test_dependency_command = %q, want 'vitest related'", test.TestDependencyCommand)
	}
}

// TestConfig_Enforcement_ToolchainRejectsUnknownNestedKey guards the
// additionalProperties:false invariant for the new toolchain-pass objects: an
// unknown key under a toolchain entry must be rejected by the strict
// KnownFields decoder, preserving the TestConfig_Struct_RejectsUnknownKeys
// invariant for the new nested objects.
func TestConfig_Enforcement_ToolchainRejectsUnknownNestedKey(t *testing.T) {
	content := `project: ts-example
language: typescript
enforcement:
  test_command: "vitest run"
  toolchain:
    lint:
      command: "eslint --format json"
      format: eslint-json
      bogus_key: "should be rejected"
`
	path := writeTempConfig(t, content)
	_, err := config.LoadConfigFromPath(path)
	if err == nil {
		t.Fatal("expected error for unknown nested toolchain key, got nil")
	}
}

// TestConfig_LoadConfig_DiscoversFromCWD verifies the zero-arg LoadConfig path:
// it discovers backstop.yml relative to the process working directory and
// returns a populated config.
func TestConfig_LoadConfig_DiscoversFromCWD(t *testing.T) {
	dir := t.TempDir()
	content := "project: cwd-loaded\nlanguage: go\n"
	if err := os.WriteFile(filepath.Join(dir, "backstop.yml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	origWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origWD) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Project != "cwd-loaded" {
		t.Errorf("project = %q, want %q", cfg.Project, "cwd-loaded")
	}
}

// TestConfig_LoadConfig_PropagatesDiscoveryError verifies that LoadConfig
// surfaces the discovery error when no backstop.yml exists anywhere up the tree
// from the working directory.
func TestConfig_LoadConfig_PropagatesDiscoveryError(t *testing.T) {
	dir := t.TempDir()
	origWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origWD) })
	// BACKSTOP_CONFIG must be unset so discovery walks up and fails at root.
	t.Setenv("BACKSTOP_CONFIG", "")
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	if _, err := config.LoadConfig(); err == nil {
		t.Fatal("expected discovery error from LoadConfig, got nil")
	}
}

// TestConfig_LoadConfigFromDir_PropagatesDiscoveryError verifies that
// LoadConfigFromDir returns the discovery error (rather than a load error) when
// the starting directory has no backstop.yml in its ancestry.
func TestConfig_LoadConfigFromDir_PropagatesDiscoveryError(t *testing.T) {
	t.Setenv("BACKSTOP_CONFIG", "")
	if _, err := config.LoadConfigFromDir(t.TempDir()); err == nil {
		t.Fatal("expected discovery error from LoadConfigFromDir, got nil")
	}
}

// TestConfig_LoadConfigFromPath_ReadError verifies that a path pointing at a
// nonexistent file produces a read error mentioning the path.
func TestConfig_LoadConfigFromPath_ReadError(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "absent.yml")
	_, err := config.LoadConfigFromPath(missing)
	if err == nil {
		t.Fatal("expected read error for missing config file, got nil")
	}
	if !strings.Contains(err.Error(), "reading config") {
		t.Errorf("expected error to mention reading config, got %q", err.Error())
	}
}

// TestConfig_LoadConfigFromPath_ParseError verifies that malformed YAML
// produces a parse error from the strict decoder.
func TestConfig_LoadConfigFromPath_ParseError(t *testing.T) {
	path := writeTempConfig(t, "project: [this is: not valid scalar\n")
	_, err := config.LoadConfigFromPath(path)
	if err == nil {
		t.Fatal("expected parse error for malformed YAML, got nil")
	}
	if !strings.Contains(err.Error(), "parsing backstop.yml") {
		t.Errorf("expected error to mention parsing backstop.yml, got %q", err.Error())
	}
}

// TestConfig_Validate_NonMappingDocumentRejected verifies that a top-level YAML
// document that is a sequence rather than a mapping is rejected by schema
// validation. The strict struct decoder rejects it first; in both cases a
// non-mapping document must not load.
func TestConfig_Validate_NonMappingDocumentRejected(t *testing.T) {
	path := writeTempConfig(t, "- project: svc\n- language: go\n")
	if _, err := config.LoadConfigFromPath(path); err == nil {
		t.Fatal("expected error for non-mapping top-level document, got nil")
	}
}

// TestConfig_Validate_RejectsUnknownTopLevelField verifies the
// additionalProperties:false schema branch: an unknown top-level key must be
// rejected with a message naming the offending field.
func TestConfig_Validate_RejectsUnknownTopLevelField(t *testing.T) {
	// Use a key the strict struct decoder would also reject; assert the schema
	// validator's message specifically by validating through LoadConfigFromPath.
	path := writeTempConfig(t, "project: svc\nlanguage: go\nmystery_field: nope\n")
	_, err := config.LoadConfigFromPath(path)
	if err == nil {
		t.Fatal("expected error for unknown top-level field, got nil")
	}
	if !strings.Contains(err.Error(), "mystery_field") {
		t.Errorf("expected error to name the unknown field, got %q", err.Error())
	}
}

// TestConfig_Validate_AcceptsNestedMapsAndSequences verifies that a config with
// nested mappings and sequences (packs map, registries map) validates cleanly,
// exercising the recursive map/slice conversion of YAML values into
// JSON-compatible structures during schema validation.
func TestConfig_Validate_AcceptsNestedMapsAndSequences(t *testing.T) {
	content := `project: nested
language: go
runtimes:
  - go
  - node
packs:
  go-core: "1.2.3"
  ts-core: "local"
registries:
  "@acme": "https://packs.acme.example/registry"
`
	path := writeTempConfig(t, content)
	cfg, err := config.LoadConfigFromPath(path)
	if err != nil {
		t.Fatalf("expected nested maps and sequences to validate, got %v", err)
	}
	if cfg.Packs["go-core"] != "1.2.3" {
		t.Errorf("packs[go-core] = %q, want %q", cfg.Packs["go-core"], "1.2.3")
	}
	if len(cfg.Runtimes) != 2 {
		t.Errorf("expected 2 runtimes, got %d", len(cfg.Runtimes))
	}
}
