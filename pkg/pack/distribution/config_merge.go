package distribution

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// MergeResult holds the result of a tool_config merge operation.
type MergeResult struct {
	Merged    []ProvenanceEntry `json:"merged"`
	Conflicts []ConfigConflict  `json:"conflicts"`
}

// ConfigConflict describes a conflict between a pack's desired config value
// and the consumer's current value.
type ConfigConflict struct {
	ConfigFile   string `json:"config_file"`
	SettingKey   string `json:"setting_key"`
	PackValue    string `json:"pack_value"`
	CurrentValue string `json:"current_value"`
}

// toolConfigEntry represents a single tool_config entry from pack.yml.
type toolConfigEntry struct {
	ConfigFile string                 `json:"config_file" yaml:"config_file"`
	Settings   map[string]interface{} `json:"settings"    yaml:"settings"`
}

// packManifest is a minimal pack.yml structure for tool_config extraction.
type packManifest struct {
	ToolConfig []toolConfigEntry `json:"tool_config" yaml:"tool_config"`
}

// MergeToolConfig reads tool_config from the pack manifest, compares to the
// consumer's current config, detects conflicts, and merges additively.
func MergeToolConfig(packDir string, projectDir string, prov *Provenance) (*MergeResult, error) {
	result := &MergeResult{
		Merged:    []ProvenanceEntry{},
		Conflicts: []ConfigConflict{},
	}

	manifest, err := readPackManifest(packDir)
	if err != nil {
		return nil, fmt.Errorf("reading pack manifest: %w", err)
	}

	if len(manifest.ToolConfig) == 0 {
		return result, nil
	}

	for _, tc := range manifest.ToolConfig {
		currentConfig, _ := readConfigFile(filepath.Join(projectDir, tc.ConfigFile))

		for key, packValue := range tc.Settings {
			packValueStr := fmt.Sprintf("%v", packValue)

			if currentConfig != nil {
				if currentValue, exists := currentConfig[key]; exists {
					currentValueStr := fmt.Sprintf("%v", currentValue)
					if currentValueStr != packValueStr {
						result.Conflicts = append(result.Conflicts, ConfigConflict{
							ConfigFile:   tc.ConfigFile,
							SettingKey:   key,
							PackValue:    packValueStr,
							CurrentValue: currentValueStr,
						})
						continue
					}
					// Same value — already set, skip.
					continue
				}
			}

			// New setting — merge additively.
			valueHash := computeValueHash(packValue)
			result.Merged = append(result.Merged, ProvenanceEntry{
				ConfigFile: tc.ConfigFile,
				SettingKey: key,
				ValueHash:  valueHash,
			})
		}
	}

	return result, nil
}

// readPackManifest reads the pack.yml manifest from a pack directory.
func readPackManifest(packDir string) (*packManifest, error) {
	path := filepath.Join(packDir, "pack.yml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	var manifest packManifest

	// Try YAML first, then JSON.
	if yamlErr := yaml.Unmarshal(data, &manifest); yamlErr != nil {
		if jsonErr := json.Unmarshal(data, &manifest); jsonErr != nil {
			return nil, fmt.Errorf("parsing %s: yaml: %w, json: %v", path, yamlErr, jsonErr)
		}
	}

	return &manifest, nil
}

// readConfigFile reads a configuration file (YAML or JSON) and returns
// a flat key-value map. Returns nil map and no error if file does not exist.
func readConfigFile(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	var result map[string]interface{}

	if strings.HasSuffix(path, ".json") {
		if err := json.Unmarshal(data, &result); err != nil {
			return nil, err
		}
	} else {
		if err := yaml.Unmarshal(data, &result); err != nil {
			return nil, err
		}
	}

	return result, nil
}

// computeValueHash computes a SHA-256 hash of a serialized setting value.
func computeValueHash(value interface{}) string {
	data, _ := json.Marshal(value)
	h := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(h[:])
}
