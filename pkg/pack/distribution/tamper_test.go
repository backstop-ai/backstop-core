package distribution_test

import (
	"os"
	"path/filepath"
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
