package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bmanson/backstop-core/pkg/baseengines"
	"github.com/bmanson/backstop-core/pkg/pack"
	"github.com/bmanson/backstop-core/pkg/pack/engine"
)

// astGrepConfigBinding returns the production ast-grep EngineBinding from the
// default registry, so these unit tests assert gatherEngineInputs over the REAL
// (flipped) binding rather than an ad-hoc shape.
func astGrepConfigBinding(t *testing.T) engine.EngineBinding {
	t.Helper()
	b, err := baseengines.Registry().Lookup("ast-grep")
	if err != nil {
		t.Fatalf("ast-grep must be a built-in engine: %v", err)
	}
	return b
}

// writeMultiRuleAstGrepPack lays down a minimal pack root carrying a
// pack-shipped ast-grep/sgconfig.yml (the config-file input the dispatch
// resolves) and returns the pack root. No backstop generation: the sgconfig is
// pack DATA written here as a fixture.
func writeMultiRuleAstGrepPack(t *testing.T) string {
	t.Helper()
	packRoot := t.TempDir()
	agDir := filepath.Join(packRoot, "ast-grep")
	if err := os.MkdirAll(filepath.Join(agDir, "rules"), 0o755); err != nil {
		t.Fatalf("mkdir ast-grep/rules: %v", err)
	}
	sg := "ruleDirs:\n  - rules\n"
	if err := os.WriteFile(filepath.Join(agDir, "sgconfig.yml"), []byte(sg), 0o644); err != nil {
		t.Fatalf("write sgconfig.yml: %v", err)
	}
	return packRoot
}

// TestGatherEngineInputs_AstGrepMultiRuleEmitsSingleConfig proves the corrected
// ast-grep dispatch shape (ISSUE-028 / CLM-002): given a manifest with TWO
// ast-grep rules that BOTH declare rule_path ast-grep/sgconfig.yml,
// gatherEngineInputs returns exactly one ["--config", <abs sgconfig.yml>] pair —
// NOT two per-dir "--rule" emissions, and NOT a duplicate "--config". One
// invocation, one config, all rules.
func TestGatherEngineInputs_AstGrepMultiRuleEmitsSingleConfig(t *testing.T) {
	packRoot := writeMultiRuleAstGrepPack(t)
	binding := astGrepConfigBinding(t)

	manifest := &pack.Manifest{NormalizedName: "test-org/multirule-pack"}
	rules := []pack.Rule{
		{ID: "rule-one", Engine: "ast-grep", RulePath: "ast-grep/sgconfig.yml"},
		{ID: "rule-two", Engine: "ast-grep", RulePath: "ast-grep/sgconfig.yml"},
	}

	args, err := gatherEngineInputs(manifest, packRoot, binding, rules)
	if err != nil {
		t.Fatalf("gatherEngineInputs (ast-grep multi-rule): %v", err)
	}

	wantPath, _ := filepath.Abs(filepath.Join(packRoot, "ast-grep", "sgconfig.yml"))
	if len(args) != 2 {
		t.Fatalf("ast-grep multi-rule must emit exactly one --config pair, got %d args: %#v", len(args), args)
	}
	if args[0] != "--config" {
		t.Errorf("ast-grep input flag must be --config, got %q (no --rule, no duplicate flag)", args[0])
	}
	if args[1] != wantPath {
		t.Errorf("ast-grep --config must point at the pack-shipped sgconfig.yml, got %q want %q", args[1], wantPath)
	}
	// Defensively prove the broken rule-dir shape is gone: no "--rule" flag and
	// no second config emission for the second rule.
	for _, a := range args {
		if a == "--rule" {
			t.Errorf("ast-grep must NOT emit --rule after the config-file flip, got args %#v", args)
		}
	}
}

// TestGatherEngineInputs_AstGrepConfigPathResolvedFromPack proves the emitted
// config path is the PACK-SHIPPED sgconfig.yml resolved under the pack root —
// backstop passes pack DATA, it never generates a temp config (ISSUE-028 /
// CLM-003). Asserts the resolved path lives under packRoot and that no sibling
// temp/generated config file was created.
func TestGatherEngineInputs_AstGrepConfigPathResolvedFromPack(t *testing.T) {
	packRoot := writeMultiRuleAstGrepPack(t)
	binding := astGrepConfigBinding(t)

	manifest := &pack.Manifest{NormalizedName: "test-org/multirule-pack"}
	rules := []pack.Rule{
		{ID: "rule-one", Engine: "ast-grep", RulePath: "ast-grep/sgconfig.yml"},
		{ID: "rule-two", Engine: "ast-grep", RulePath: "ast-grep/sgconfig.yml"},
	}

	// Snapshot the ast-grep dir before dispatch input-gathering so we can prove no
	// generated config file is written by backstop.
	agDir := filepath.Join(packRoot, "ast-grep")
	before := dirEntryNames(t, agDir)

	args, err := gatherEngineInputs(manifest, packRoot, binding, rules)
	if err != nil {
		t.Fatalf("gatherEngineInputs: %v", err)
	}
	if len(args) != 2 || args[0] != "--config" {
		t.Fatalf("expected one --config pair, got %#v", args)
	}

	resolved := args[1]
	absPackRoot, _ := filepath.Abs(packRoot)
	rel, relErr := filepath.Rel(absPackRoot, resolved)
	if relErr != nil || rel == ".." || filepath.IsAbs(rel) || len(rel) >= 2 && rel[0:2] == ".." {
		t.Errorf("config path must resolve UNDER the pack root (pack data, not a temp file), got %q (rel %q)", resolved, rel)
	}
	if filepath.Base(resolved) != "sgconfig.yml" {
		t.Errorf("config path must be the pack-shipped sgconfig.yml, got %q", resolved)
	}

	// Backstop must not have GENERATED a config: the ast-grep dir contents are
	// unchanged by input-gathering (no temp/generated config dropped).
	after := dirEntryNames(t, agDir)
	if len(after) != len(before) {
		t.Errorf("backstop must not generate or write config files; ast-grep dir changed from %v to %v", before, after)
	}
}

// dirEntryNames lists the base names of entries in dir, for the no-generated-file
// assertion.
func dirEntryNames(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir %s: %v", dir, err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}
