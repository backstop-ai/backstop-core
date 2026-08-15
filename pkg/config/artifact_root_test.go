package config

// This file is deliberately an INTERNAL test (package config, not config_test) so
// TestLoadConfig_ArtifactRootAcceptedByJSONSchema can drive validateAgainstSchema
// DIRECTLY. backstop.yml passes through TWO independent strictness gates — the
// KnownFields YAML decoder and the additionalProperties:false JSON-schema pass — and a
// test that only went through LoadConfigFromPath could be made green by adding the Go
// struct field alone, which is exactly the half-fix CLM-041 exists to forbid.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeConfig(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "backstop.yml")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("writing backstop.yml: %v", err)
	}
	return path
}

// TestLoadConfig_ArtifactRootParses pins CLM-040: artifact_root survives the STRICT
// typed loader. The decoder runs with KnownFields(true), so an unlisted key errors
// here rather than being silently dropped.
func TestLoadConfig_ArtifactRootParses(t *testing.T) {
	path := writeConfig(t, "project: artifact-root-fixture\nartifact_root: .backstop\n")

	cfg, err := LoadConfigFromPath(path)
	if err != nil {
		t.Fatalf("LoadConfigFromPath returned error %v, want nil", err)
	}
	if cfg.ArtifactRoot != ".backstop" {
		t.Errorf("cfg.ArtifactRoot = %q, want \".backstop\"", cfg.ArtifactRoot)
	}
}

// TestLoadConfig_ArtifactRootAcceptedByJSONSchema pins CLM-041 — the SECOND,
// independent strictness pass. artifacts/backstop-yml/v1/schema.json is
// additionalProperties:false at the top level, so adding only the Go struct field
// passes the yaml decoder and FAILS here.
func TestLoadConfig_ArtifactRootAcceptedByJSONSchema(t *testing.T) {
	if err := validateAgainstSchema([]byte("project: artifact-root-fixture\nartifact_root: .backstop\n")); err != nil {
		t.Fatalf("the JSON-schema pass rejected artifact_root: %v", err)
	}

	// The same pass must still REJECT a genuinely unknown key. Without this the test
	// above would also pass against a schema whose additionalProperties guard had been
	// loosened rather than a schema that declares artifact_root.
	err := validateAgainstSchema([]byte("project: artifact-root-fixture\nnot_a_real_key: value\n"))
	if err == nil {
		t.Fatal("the JSON-schema pass accepted an unknown top-level key; additionalProperties:false is not being enforced")
	}
	if !strings.Contains(err.Error(), "not_a_real_key") {
		t.Errorf("rejection message %q does not name the offending key", err.Error())
	}
}

// TestLoadConfig_AbsentArtifactRootLeavesFieldEmpty pins CLM-042. The default lives in
// artifact.ResolveRoot's absent-declaration branch; baking one here would give the
// system two answers.
func TestLoadConfig_AbsentArtifactRootLeavesFieldEmpty(t *testing.T) {
	path := writeConfig(t, "project: artifact-root-fixture\n")

	cfg, err := LoadConfigFromPath(path)
	if err != nil {
		t.Fatalf("LoadConfigFromPath returned error %v, want nil", err)
	}
	if cfg.ArtifactRoot != "" {
		t.Errorf("cfg.ArtifactRoot = %q, want empty — no default root may be baked into the loader", cfg.ArtifactRoot)
	}
}
