package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bmanson/backstop-core/pkg/config"
	"github.com/bmanson/backstop-core/pkg/gate"
	"github.com/bmanson/backstop-core/pkg/pack"
)

func TestGateViolationsToCheck(t *testing.T) {
	violations := []gate.Violation{
		{Rule: "pack/rule-1", File: "main.go", Message: "bad", Severity: "error", SourcePack: "acme/go"},
		{Rule: "pack/rule-2", File: "util.go", Message: "also bad", Severity: "warning"},
	}
	result := gateViolationsToCheck(violations)
	if len(result) != 2 {
		t.Fatalf("expected 2 violations, got %d", len(result))
	}
	if result[0].File != "main.go" || result[0].Message != "bad" {
		t.Errorf("first violation mismatch: %+v", result[0])
	}
	if result[1].File != "util.go" {
		t.Errorf("second violation mismatch: %+v", result[1])
	}
}

func TestGateViolationsToCheck_Empty(t *testing.T) {
	result := gateViolationsToCheck(nil)
	if len(result) != 0 {
		t.Errorf("expected 0 violations, got %d", len(result))
	}
}

func TestPackNamesFromManifests(t *testing.T) {
	packs := []*pack.Manifest{
		{NormalizedName: "acme/go-standards"},
		{NormalizedName: "internal/security"},
	}
	names := packNamesFromManifests(packs)
	if len(names) != 2 {
		t.Fatalf("expected 2 names, got %d", len(names))
	}
	if names[0] != "acme/go-standards" {
		t.Errorf("first name: %q", names[0])
	}
	if names[1] != "internal/security" {
		t.Errorf("second name: %q", names[1])
	}
}

func TestPackNamesFromManifests_Empty(t *testing.T) {
	names := packNamesFromManifests(nil)
	if len(names) != 0 {
		t.Errorf("expected 0 names, got %d", len(names))
	}
}

func TestDeclaredPackNames_NilConfig(t *testing.T) {
	names := declaredPackNames(nil)
	if len(names) != 0 {
		t.Errorf("expected 0 names for nil config, got %d", len(names))
	}
}

func TestDeclaredPackNames_EmptyConfig(t *testing.T) {
	cfg := &config.Config{}
	names := declaredPackNames(cfg)
	if len(names) != 0 {
		t.Errorf("expected 0 names for empty config, got %d", len(names))
	}
}

func TestLoadInstalledPacks_NoPacks(t *testing.T) {
	dir := t.TempDir()
	// No backstop.yml — returns nil packs
	packs, err := loadInstalledPacks(dir)
	if err != nil {
		// Expected — no backstop.yml
		return
	}
	if len(packs) != 0 {
		t.Errorf("expected 0 packs, got %d", len(packs))
	}
}

func TestVerifyPackLock_NoPacks(t *testing.T) {
	dir := t.TempDir()
	// No packs — verify should pass or return nil error
	err := verifyPackLock(dir, nil)
	if err != nil {
		t.Errorf("expected no error for nil packs, got: %v", err)
	}
}

func TestMergePackRules_NoPacks(t *testing.T) {
	rules, err := mergePackRules(nil, t.TempDir())
	if err != nil {
		t.Fatalf("mergePackRules: %v", err)
	}
	if len(rules) != 0 {
		t.Errorf("expected 0 rules, got %d", len(rules))
	}
}

func TestRunPackValidators_NoPacks(t *testing.T) {
	violations, err := runPackValidators(nil, t.TempDir(), t.TempDir())
	if err != nil {
		t.Fatalf("runPackValidators: %v", err)
	}
	if len(violations) != 0 {
		t.Errorf("expected 0 violations, got %d", len(violations))
	}
}

func TestRunPackValidators_SkipsNonLayer3(t *testing.T) {
	manifests := []*pack.Manifest{{
		NormalizedName: "test/pack",
		Content: pack.Content{
			Ruleset: pack.Ruleset{
				Rules: []pack.Rule{
					{ID: "r1", Layer: 1},
					{ID: "r2", Layer: 2},
				},
			},
		},
	}}
	violations, err := runPackValidators(manifests, t.TempDir(), t.TempDir())
	if err != nil {
		t.Fatalf("runPackValidators: %v", err)
	}
	if len(violations) != 0 {
		t.Errorf("expected 0 violations for non-layer-3 rules, got %d", len(violations))
	}
}

func TestMergePackRules_CollectsLayer2Paths(t *testing.T) {
	packDir := t.TempDir()
	packRoot := filepath.Join(packDir, "test/pack")
	os.MkdirAll(filepath.Join(packRoot, "rules"), 0o755)
	os.WriteFile(filepath.Join(packRoot, "rules", "r1.yml"), []byte("rules: []"), 0o644)
	os.WriteFile(filepath.Join(packRoot, "rules", "r2.yml"), []byte("rules: []"), 0o644)

	manifests := []*pack.Manifest{{
		NormalizedName: "test/pack",
		Content: pack.Content{
			Ruleset: pack.Ruleset{
				Rules: []pack.Rule{
					{ID: "r1", Layer: 2, RulePath: "rules/r1.yml"},
					{ID: "r2", Layer: 2, RulePath: "rules/r2.yml"},
					{ID: "r3", Layer: 3, Validator: "v.sh"}, // should be skipped
				},
			},
		},
	}}
	paths, err := mergePackRules(manifests, packDir)
	if err != nil {
		t.Fatalf("mergePackRules: %v", err)
	}
	if len(paths) != 2 {
		t.Errorf("expected 2 rule paths, got %d: %v", len(paths), paths)
	}
}

func TestVerifyPackLock_EmptyPacks(t *testing.T) {
	err := verifyPackLock(t.TempDir(), []string{})
	if err != nil {
		t.Errorf("expected no error for empty packs, got: %v", err)
	}
}
