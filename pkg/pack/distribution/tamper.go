package distribution

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// TamperResult holds the result of a tamper detection comparison.
type TamperResult struct {
	Changes   []TamperChange `json:"changes"`
	HasTamper bool           `json:"has_tamper"`
}

// TamperChange describes a single tamper-detected change.
type TamperChange struct {
	Kind        string `json:"kind"`
	Description string `json:"description"`
}

// tamperRule represents a rule entry for tamper comparison.
type tamperRule struct {
	ID        string `yaml:"id"`
	Severity  string `yaml:"severity"`
	RiskClass string `yaml:"risk_class"`
}

// tamperManifest is a minimal pack.yml for tamper detection.
type tamperManifest struct {
	Rules []tamperRule `yaml:"rules"`
}

// severityRank maps severity levels to numeric ranks for downgrade detection.
var severityRank = map[string]int{
	"error":   3,
	"warning": 2,
	"info":    1,
}

// DetectTamper compares an old and new pack directory for the four tamper categories:
// fixture_removal, severity_downgrade, risk_class_change, rule_removal.
func DetectTamper(oldPackDir string, newPackDir string) (*TamperResult, error) {
	result := &TamperResult{
		Changes: []TamperChange{},
	}

	// Check for fixture removal (testdata files).
	if err := detectFixtureRemoval(oldPackDir, newPackDir, result); err != nil {
		return nil, fmt.Errorf("detecting fixture removal: %w", err)
	}

	// Read pack manifests for rule comparison.
	oldManifest, err := readTamperManifest(oldPackDir)
	if err != nil {
		return nil, fmt.Errorf("reading old manifest: %w", err)
	}

	newManifest, err := readTamperManifest(newPackDir)
	if err != nil {
		return nil, fmt.Errorf("reading new manifest: %w", err)
	}

	// Build rule maps for comparison.
	oldRules := make(map[string]tamperRule)
	for _, r := range oldManifest.Rules {
		oldRules[r.ID] = r
	}

	newRules := make(map[string]tamperRule)
	for _, r := range newManifest.Rules {
		newRules[r.ID] = r
	}

	// Check for rule removal.
	for id := range oldRules {
		if _, exists := newRules[id]; !exists {
			result.Changes = append(result.Changes, TamperChange{
				Kind:        "rule_removal",
				Description: fmt.Sprintf("rule %s present in old version but absent in new version", id),
			})
		}
	}

	// Check for severity downgrade and risk_class change.
	for id, oldRule := range oldRules {
		newRule, exists := newRules[id]
		if !exists {
			continue
		}

		// Severity downgrade.
		oldRank := severityRank[oldRule.Severity]
		newRank := severityRank[newRule.Severity]
		if newRank < oldRank {
			result.Changes = append(result.Changes, TamperChange{
				Kind:        "severity_downgrade",
				Description: fmt.Sprintf("rule %s severity downgraded from %s to %s", id, oldRule.Severity, newRule.Severity),
			})
		}

		// Risk class change.
		if oldRule.RiskClass != newRule.RiskClass {
			result.Changes = append(result.Changes, TamperChange{
				Kind:        "risk_class_change",
				Description: fmt.Sprintf("rule %s risk_class changed from %s to %s", id, oldRule.RiskClass, newRule.RiskClass),
			})
		}
	}

	result.HasTamper = len(result.Changes) > 0
	return result, nil
}

// detectFixtureRemoval checks if testdata files were removed between versions.
func detectFixtureRemoval(oldDir, newDir string, result *TamperResult) error {
	testdataDir := filepath.Join(oldDir, "testdata")
	if _, err := os.Stat(testdataDir); os.IsNotExist(err) {
		return nil
	}

	return filepath.Walk(testdataDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		rel, relErr := filepath.Rel(oldDir, path)
		if relErr != nil {
			return relErr
		}

		newPath := filepath.Join(newDir, rel)
		if _, statErr := os.Stat(newPath); os.IsNotExist(statErr) {
			result.Changes = append(result.Changes, TamperChange{
				Kind:        "fixture_removal",
				Description: fmt.Sprintf("test fixture %s removed in new version", rel),
			})
		}

		return nil
	})
}

// readTamperManifest reads pack.yml for tamper comparison.
func readTamperManifest(dir string) (*tamperManifest, error) {
	data, err := os.ReadFile(filepath.Join(dir, "pack.yml"))
	if err != nil {
		return nil, err
	}

	var manifest tamperManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}

	return &manifest, nil
}
