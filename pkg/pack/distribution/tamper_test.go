package distribution_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/pack/distribution"
)

func TestPackUpdate_TamperDetectsFixtureRemoval(t *testing.T) {
	oldDir := setupTamperOldPack(t)
	newDir := filepath.Join("testdata", "tamper-pack-fixture-removed")

	result, err := distribution.DetectTamper(oldDir, newDir)
	if err != nil {
		t.Fatalf("DetectTamper: %v", err)
	}

	if !result.HasTamper {
		t.Fatal("expected tamper detection for fixture removal")
	}

	found := false
	for _, c := range result.Changes {
		if c.Kind == "fixture_removal" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected fixture_removal tamper change")
	}
}

func TestPackUpdate_TamperDetectsSeverityDowngrade(t *testing.T) {
	oldDir := filepath.Join("testdata", "valid-pack")
	newDir := filepath.Join("testdata", "tamper-pack-severity-downgrade")

	result, err := distribution.DetectTamper(oldDir, newDir)
	if err != nil {
		t.Fatalf("DetectTamper: %v", err)
	}

	if !result.HasTamper {
		t.Fatal("expected tamper detection for severity downgrade")
	}

	found := false
	for _, c := range result.Changes {
		if c.Kind == "severity_downgrade" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected severity_downgrade tamper change")
	}
}

func TestPackUpdate_TamperDetectsRiskClassChange(t *testing.T) {
	oldDir := filepath.Join("testdata", "valid-pack")
	newDir := filepath.Join("testdata", "tamper-pack-risk-class-change")

	result, err := distribution.DetectTamper(oldDir, newDir)
	if err != nil {
		t.Fatalf("DetectTamper: %v", err)
	}

	if !result.HasTamper {
		t.Fatal("expected tamper detection for risk_class change")
	}

	found := false
	for _, c := range result.Changes {
		if c.Kind == "risk_class_change" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected risk_class_change tamper change")
	}
}

func TestPackUpdate_TamperDetectsRuleRemoval(t *testing.T) {
	oldDir := filepath.Join("testdata", "valid-pack")
	newDir := filepath.Join("testdata", "tamper-pack-rule-removed")

	result, err := distribution.DetectTamper(oldDir, newDir)
	if err != nil {
		t.Fatalf("DetectTamper: %v", err)
	}

	if !result.HasTamper {
		t.Fatal("expected tamper detection for rule removal")
	}

	found := false
	for _, c := range result.Changes {
		if c.Kind == "rule_removal" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected rule_removal tamper change")
	}
}

func TestDetectTamper_NoTamper(t *testing.T) {
	// Compare pack to itself — no tamper.
	dir := filepath.Join("testdata", "valid-pack")

	result, err := distribution.DetectTamper(dir, dir)
	if err != nil {
		t.Fatalf("DetectTamper: %v", err)
	}

	if result.HasTamper {
		t.Errorf("expected no tamper, got %d changes", len(result.Changes))
	}
}

func TestDetectTamper_NonTamperChangesAccepted(t *testing.T) {
	// valid-pack-v2 adds a new rule and updates descriptions — non-tamper changes.
	oldDir := filepath.Join("testdata", "valid-pack")
	newDir := filepath.Join("testdata", "valid-pack-v2")

	result, err := distribution.DetectTamper(oldDir, newDir)
	if err != nil {
		t.Fatalf("DetectTamper: %v", err)
	}

	if result.HasTamper {
		t.Errorf("expected no tamper for non-tamper changes, got %d changes:", len(result.Changes))
		for _, c := range result.Changes {
			t.Logf("  %s: %s", c.Kind, c.Description)
		}
	}
}

// setupTamperOldPack creates a pack directory with a test fixture file for tamper comparison.
func setupTamperOldPack(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Copy valid-pack content plus a testdata fixture.
	src := filepath.Join("testdata", "valid-pack", "pack.yml")
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("reading source pack.yml: %v", err)
	}
	writeFile(t, filepath.Join(dir, "pack.yml"), string(data))

	// Add a test fixture that will be "removed" in the tamper fixture.
	fixtureDir := filepath.Join(dir, "testdata")
	if err := os.MkdirAll(fixtureDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(fixtureDir, "fixture.json"), `{"input": "test", "expected": "pass"}`)

	return dir
}

func TestDetectTamper_OldManifestMissing(t *testing.T) {
	oldDir := t.TempDir()
	// No pack.yml in oldDir.
	newDir := filepath.Join("testdata", "valid-pack")

	_, err := distribution.DetectTamper(oldDir, newDir)
	if err == nil {
		t.Fatal("expected error when old manifest is missing")
	}

	if !strings.Contains(err.Error(), "reading old manifest") {
		t.Errorf("error should mention reading old manifest, got: %v", err)
	}
}

func TestDetectTamper_NewManifestMissing(t *testing.T) {
	oldDir := filepath.Join("testdata", "valid-pack")
	newDir := t.TempDir()
	// No pack.yml in newDir.

	_, err := distribution.DetectTamper(oldDir, newDir)
	if err == nil {
		t.Fatal("expected error when new manifest is missing")
	}

	if !strings.Contains(err.Error(), "reading new manifest") {
		t.Errorf("error should mention reading new manifest, got: %v", err)
	}
}

func TestDetectTamper_InvalidYAML(t *testing.T) {
	oldDir := t.TempDir()
	writeFile(t, filepath.Join(oldDir, "pack.yml"), "not: [valid: yaml: {{{")

	newDir := filepath.Join("testdata", "valid-pack")

	_, err := distribution.DetectTamper(oldDir, newDir)
	if err == nil {
		t.Fatal("expected error for invalid YAML in manifest")
	}
}

func TestDetectTamper_OldTestdataDoesNotExist(t *testing.T) {
	oldDir := t.TempDir()
	writeFile(t, filepath.Join(oldDir, "pack.yml"), "name: test\nrules: []\n")
	// No testdata directory.

	newDir := filepath.Join("testdata", "valid-pack")

	result, err := distribution.DetectTamper(oldDir, newDir)
	if err != nil {
		t.Fatalf("DetectTamper: %v", err)
	}

	// No fixture removal should be detected when old pack has no testdata.
	for _, c := range result.Changes {
		if c.Kind == "fixture_removal" {
			t.Error("should not detect fixture_removal when old pack has no testdata dir")
		}
	}
}

func TestDetectTamper_SeverityUpgradeNotTamper(t *testing.T) {
	// Old pack with info severity, new pack with error severity (upgrade).
	oldDir := t.TempDir()
	writeFile(t, filepath.Join(oldDir, "pack.yml"),
		"name: test\nrules:\n  - id: RULE-1\n    severity: info\n    risk_class: low\n")

	newDir := t.TempDir()
	writeFile(t, filepath.Join(newDir, "pack.yml"),
		"name: test\nrules:\n  - id: RULE-1\n    severity: error\n    risk_class: low\n")

	result, err := distribution.DetectTamper(oldDir, newDir)
	if err != nil {
		t.Fatalf("DetectTamper: %v", err)
	}

	for _, c := range result.Changes {
		if c.Kind == "severity_downgrade" {
			t.Error("severity upgrade should not be detected as tamper")
		}
	}

	if result.HasTamper {
		t.Error("severity upgrade should not count as tamper")
	}
}

func TestDetectTamper_EmptyRules(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "pack.yml"), "name: test\nrules: []\n")

	result, err := distribution.DetectTamper(dir, dir)
	if err != nil {
		t.Fatalf("DetectTamper: %v", err)
	}

	if result.HasTamper {
		t.Error("expected no tamper for empty rules")
	}
}

func TestDetectTamper_NewRuleAddedNotTamper(t *testing.T) {
	oldDir := t.TempDir()
	writeFile(t, filepath.Join(oldDir, "pack.yml"),
		"name: test\nrules:\n  - id: RULE-1\n    severity: error\n    risk_class: high\n")

	newDir := t.TempDir()
	writeFile(t, filepath.Join(newDir, "pack.yml"),
		"name: test\nrules:\n  - id: RULE-1\n    severity: error\n    risk_class: high\n  - id: RULE-2\n    severity: warning\n    risk_class: medium\n")

	result, err := distribution.DetectTamper(oldDir, newDir)
	if err != nil {
		t.Fatalf("DetectTamper: %v", err)
	}

	if result.HasTamper {
		t.Error("adding a new rule should not be tamper")
	}
}

func TestDetectTamper_MultipleChanges(t *testing.T) {
	oldDir := t.TempDir()
	writeFile(t, filepath.Join(oldDir, "pack.yml"),
		"name: test\nrules:\n  - id: RULE-1\n    severity: error\n    risk_class: high\n  - id: RULE-2\n    severity: warning\n    risk_class: medium\n")

	// New pack: RULE-1 severity downgraded + risk_class changed, RULE-2 removed.
	newDir := t.TempDir()
	writeFile(t, filepath.Join(newDir, "pack.yml"),
		"name: test\nrules:\n  - id: RULE-1\n    severity: info\n    risk_class: low\n")

	result, err := distribution.DetectTamper(oldDir, newDir)
	if err != nil {
		t.Fatalf("DetectTamper: %v", err)
	}

	if !result.HasTamper {
		t.Fatal("expected tamper for multiple changes")
	}

	kinds := make(map[string]bool)
	for _, c := range result.Changes {
		kinds[c.Kind] = true
	}

	if !kinds["severity_downgrade"] {
		t.Error("expected severity_downgrade")
	}
	if !kinds["risk_class_change"] {
		t.Error("expected risk_class_change")
	}
	if !kinds["rule_removal"] {
		t.Error("expected rule_removal")
	}
}
