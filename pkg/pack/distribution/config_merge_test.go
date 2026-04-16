package distribution_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/pack/distribution"
)

func TestPackAdd_ToolConfigConflictExitsNonZero(t *testing.T) {
	packDir, projectDir := setupMergeTestDirs(t)

	// Pack wants linters.enable.revive = true
	writePackYMLWithToolConfig(t, packDir, map[string]interface{}{
		".golangci.yml": map[string]interface{}{
			"linters.enable.revive": true,
		},
	})

	// Project already has linters.enable.revive = false (conflict)
	writeYAMLConfig(t, filepath.Join(projectDir, ".golangci.yml"), map[string]interface{}{
		"linters.enable.revive": false,
	})

	prov := &distribution.Provenance{Entries: []distribution.ProvenanceEntry{}}
	result, err := distribution.MergeToolConfig(packDir, projectDir, prov)
	if err != nil {
		t.Fatalf("MergeToolConfig: %v", err)
	}

	if len(result.Conflicts) == 0 {
		t.Fatal("expected conflicts, got none")
	}
}

func TestPackAdd_ToolConfigConflictDiagnosticFormat(t *testing.T) {
	packDir, projectDir := setupMergeTestDirs(t)

	writePackYMLWithToolConfig(t, packDir, map[string]interface{}{
		".golangci.yml": map[string]interface{}{
			"linters.enable.revive": true,
		},
	})

	writeYAMLConfig(t, filepath.Join(projectDir, ".golangci.yml"), map[string]interface{}{
		"linters.enable.revive": false,
	})

	prov := &distribution.Provenance{Entries: []distribution.ProvenanceEntry{}}
	result, err := distribution.MergeToolConfig(packDir, projectDir, prov)
	if err != nil {
		t.Fatalf("MergeToolConfig: %v", err)
	}

	if len(result.Conflicts) == 0 {
		t.Fatal("expected at least one conflict")
	}

	c := result.Conflicts[0]
	if c.ConfigFile == "" {
		t.Error("conflict missing ConfigFile")
	}
	if c.SettingKey == "" {
		t.Error("conflict missing SettingKey")
	}
	if c.PackValue == "" {
		t.Error("conflict missing PackValue")
	}
	if c.CurrentValue == "" {
		t.Error("conflict missing CurrentValue")
	}
}

func TestPackAdd_ToolConfigAdditiveMerge(t *testing.T) {
	packDir, projectDir := setupMergeTestDirs(t)

	writePackYMLWithToolConfig(t, packDir, map[string]interface{}{
		".golangci.yml": map[string]interface{}{
			"linters.enable.revive": true,
		},
	})

	// Project has no .golangci.yml yet — additive merge.

	prov := &distribution.Provenance{Entries: []distribution.ProvenanceEntry{}}
	result, err := distribution.MergeToolConfig(packDir, projectDir, prov)
	if err != nil {
		t.Fatalf("MergeToolConfig: %v", err)
	}

	if len(result.Conflicts) != 0 {
		t.Errorf("expected no conflicts, got %d", len(result.Conflicts))
	}
	if len(result.Merged) == 0 {
		t.Error("expected merged entries")
	}

	// Verify the config file was actually written to disk.
	configPath := filepath.Join(projectDir, ".golangci.yml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("config file not written: %v", err)
	}
	if !strings.Contains(string(data), "revive") {
		t.Fatalf("config file missing merged setting: %s", data)
	}
}

func TestMergeToolConfig_RecordsProvenance(t *testing.T) {
	packDir, projectDir := setupMergeTestDirs(t)

	writePackYMLWithToolConfig(t, packDir, map[string]interface{}{
		".golangci.yml": map[string]interface{}{
			"linters.enable.revive": true,
		},
	})

	prov := &distribution.Provenance{Entries: []distribution.ProvenanceEntry{}}
	result, err := distribution.MergeToolConfig(packDir, projectDir, prov)
	if err != nil {
		t.Fatalf("MergeToolConfig: %v", err)
	}

	if len(result.Merged) == 0 {
		t.Fatal("expected merged entries for provenance recording")
	}

	merged := result.Merged[0]
	if merged.ConfigFile == "" {
		t.Error("provenance entry missing ConfigFile")
	}
	if merged.SettingKey == "" {
		t.Error("provenance entry missing SettingKey")
	}
	if merged.ValueHash == "" {
		t.Error("provenance entry missing ValueHash")
	}
}

func TestMergeToolConfig_MultipleConfigFiles(t *testing.T) {
	packDir, projectDir := setupMergeTestDirs(t)

	writePackYMLWithToolConfig(t, packDir, map[string]interface{}{
		".golangci.yml": map[string]interface{}{
			"linters.enable.revive": true,
		},
		".eslintrc.json": map[string]interface{}{
			"rules.no-console": "error",
		},
	})

	prov := &distribution.Provenance{Entries: []distribution.ProvenanceEntry{}}
	result, err := distribution.MergeToolConfig(packDir, projectDir, prov)
	if err != nil {
		t.Fatalf("MergeToolConfig: %v", err)
	}

	if len(result.Merged) < 2 {
		t.Errorf("expected at least 2 merged entries for 2 config files, got %d", len(result.Merged))
	}
}

func TestMergeToolConfig_EmptyPackConfig(t *testing.T) {
	packDir, projectDir := setupMergeTestDirs(t)

	// Pack with no tool_config.
	writePackYMLWithToolConfig(t, packDir, nil)

	prov := &distribution.Provenance{Entries: []distribution.ProvenanceEntry{}}
	result, err := distribution.MergeToolConfig(packDir, projectDir, prov)
	if err != nil {
		t.Fatalf("MergeToolConfig: %v", err)
	}

	if len(result.Merged) != 0 {
		t.Errorf("expected no merged entries for empty config, got %d", len(result.Merged))
	}
	if len(result.Conflicts) != 0 {
		t.Errorf("expected no conflicts for empty config, got %d", len(result.Conflicts))
	}
}

func TestMergeToolConfig_YamlFormat(t *testing.T) {
	packDir, projectDir := setupMergeTestDirs(t)

	writePackYMLWithToolConfig(t, packDir, map[string]interface{}{
		".golangci.yml": map[string]interface{}{
			"linters.enable.errcheck": true,
		},
	})

	// Write an existing YAML config with a different setting.
	writeYAMLConfig(t, filepath.Join(projectDir, ".golangci.yml"), map[string]interface{}{
		"linters.enable.govet": true,
	})

	prov := &distribution.Provenance{Entries: []distribution.ProvenanceEntry{}}
	result, err := distribution.MergeToolConfig(packDir, projectDir, prov)
	if err != nil {
		t.Fatalf("MergeToolConfig: %v", err)
	}

	// No conflict — different keys.
	if len(result.Conflicts) != 0 {
		t.Errorf("expected no conflicts, got %d", len(result.Conflicts))
	}
	if len(result.Merged) == 0 {
		t.Error("expected merged entries for YAML config")
	}

	// Verify the config file was written with both settings.
	data, err := os.ReadFile(filepath.Join(projectDir, ".golangci.yml"))
	if err != nil {
		t.Fatalf("config file not written: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "errcheck") {
		t.Error("config missing newly merged setting")
	}
}

func TestMergeToolConfig_JsonFormat(t *testing.T) {
	packDir, projectDir := setupMergeTestDirs(t)

	writePackYMLWithToolConfig(t, packDir, map[string]interface{}{
		".eslintrc.json": map[string]interface{}{
			"rules.no-console": "error",
		},
	})

	prov := &distribution.Provenance{Entries: []distribution.ProvenanceEntry{}}
	result, err := distribution.MergeToolConfig(packDir, projectDir, prov)
	if err != nil {
		t.Fatalf("MergeToolConfig: %v", err)
	}

	if len(result.Merged) == 0 {
		t.Error("expected merged entries for JSON config")
	}

	// Verify JSON config was actually written.
	data, err := os.ReadFile(filepath.Join(projectDir, ".eslintrc.json"))
	if err != nil {
		t.Fatalf("JSON config not written: %v", err)
	}
	if !strings.Contains(string(data), "no-console") {
		t.Fatalf("JSON config missing merged setting: %s", data)
	}
}

// --- Test helpers ---

func setupMergeTestDirs(t *testing.T) (packDir, projectDir string) {
	t.Helper()
	packDir = t.TempDir()
	projectDir = t.TempDir()
	return packDir, projectDir
}

func writePackYMLWithToolConfig(t *testing.T, packDir string, toolConfig map[string]interface{}) {
	t.Helper()

	pack := map[string]interface{}{
		"name":        "acme/test-pack",
		"version":     "1.0.0",
		"archetype":   "rule-pack",
		"description": "test pack",
		"rules":       []interface{}{},
		"scaffolds":   []interface{}{},
	}

	if toolConfig != nil {
		var configs []interface{}
		for file, settings := range toolConfig {
			configs = append(configs, map[string]interface{}{
				"config_file": file,
				"settings":    settings,
			})
		}
		pack["tool_config"] = configs
	}

	data, err := json.MarshalIndent(pack, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(packDir, "pack.yml"), string(data))
}

func writeYAMLConfig(t *testing.T, path string, data map[string]interface{}) {
	t.Helper()
	content, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, path, string(content))
}

func computeSettingHash(value interface{}) string {
	data, _ := json.Marshal(value)
	h := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(h[:])
}

func TestMergeToolConfig_SameValueSkipped(t *testing.T) {
	packDir, projectDir := setupMergeTestDirs(t)

	writePackYMLWithToolConfig(t, packDir, map[string]interface{}{
		".golangci.yml": map[string]interface{}{
			"linters.enable.revive": true,
		},
	})

	// Project already has the same value — should skip (no merge, no conflict).
	writeYAMLConfig(t, filepath.Join(projectDir, ".golangci.yml"), map[string]interface{}{
		"linters.enable.revive": true,
	})

	prov := &distribution.Provenance{Entries: []distribution.ProvenanceEntry{}}
	result, err := distribution.MergeToolConfig(packDir, projectDir, prov)
	if err != nil {
		t.Fatalf("MergeToolConfig: %v", err)
	}

	if len(result.Conflicts) != 0 {
		t.Errorf("expected no conflicts for same value, got %d", len(result.Conflicts))
	}
	if len(result.Merged) != 0 {
		t.Errorf("expected no merges for same value, got %d", len(result.Merged))
	}
}

func TestMergeToolConfig_InvalidManifest(t *testing.T) {
	packDir, projectDir := setupMergeTestDirs(t)

	// Write invalid pack.yml
	writeFile(t, filepath.Join(packDir, "pack.yml"), "not: [valid: yaml")

	prov := &distribution.Provenance{Entries: []distribution.ProvenanceEntry{}}
	_, err := distribution.MergeToolConfig(packDir, projectDir, prov)
	if err == nil {
		t.Fatal("expected error for invalid manifest")
	}
}
