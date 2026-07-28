package main

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/pack"
)

// TestPackLayout_EngineDirResolvesInputs proves engine inputs resolve relative to
// the per-engine pack directory matching the input_mode (CLM-049 / REQ-021):
// rule-flags (semgrep) resolves the rule FILE under semgrep/, config-file (ast-grep)
// resolves the pack-shipped sgconfig under ast-grep/, and none (sandbox) gathers no inputs.
// Substantive: drives the real gatherEngineInputs against the real engine-pack
// fixture and asserts the gathered paths point into the correct per-engine dir.
func TestPackLayout_EngineDirResolvesInputs(t *testing.T) {
	packRoot := filepath.Join(engineDispatchPacksDir(t), "test-org", "engine-pack")
	manifest := &pack.Manifest{NormalizedName: "test-org/engine-pack"}

	reg := resolveEngineRegistry(nil)

	// rule-flags engine (semgrep): inputs are `--config <packRoot>/semgrep/no-eval.yml`.
	semgrepInputs, err := gatherEngineInputs(manifest, packRoot, reg["semgrep"], []pack.Rule{
		{ID: "semgrep-no-eval", Engine: "semgrep", RulePath: "semgrep/no-eval.yml"},
	})
	if err != nil {
		t.Fatalf("gatherEngineInputs(semgrep): %v", err)
	}
	wantSemgrep := filepath.Join(packRoot, "semgrep", "no-eval.yml")
	if !argsContainFlagValue(semgrepInputs, "--config", wantSemgrep) {
		t.Errorf("rule-flags engine must resolve its rule file under semgrep/, got %v want --config %s", semgrepInputs, wantSemgrep)
	}

	// config-file engine (ast-grep, ISSUE-028): inputs are `--config
	// <packRoot>/ast-grep/sgconfig.yml` — the pack-shipped sgconfig (ruleDirs)
	// runs every ast-grep rule in one invocation; backstop never emits `--rule`.
	astGrepInputs, err := gatherEngineInputs(manifest, packRoot, reg["ast-grep"], []pack.Rule{
		{ID: "ast-grep-proof", Engine: "ast-grep", RulePath: "ast-grep/sgconfig.yml"},
	})
	if err != nil {
		t.Fatalf("gatherEngineInputs(ast-grep): %v", err)
	}
	wantAstGrepConfig := filepath.Join(packRoot, "ast-grep", "sgconfig.yml")
	if !argsContainFlagValue(astGrepInputs, "--config", wantAstGrepConfig) {
		t.Errorf("config-file engine must resolve the pack-shipped sgconfig under ast-grep/, got %v want --config %s", astGrepInputs, wantAstGrepConfig)
	}

	// none engine (sandbox): gathers NO inputs (the executable is the logic).
	noneInputs, err := gatherEngineInputs(manifest, packRoot, reg["sandbox"], []pack.Rule{
		{ID: "sandbox-presence", Engine: "sandbox", Validator: "scripts/check-presence.sh"},
	})
	if err != nil {
		t.Fatalf("gatherEngineInputs(sandbox): %v", err)
	}
	if len(noneInputs) != 0 {
		t.Errorf("input_mode none must gather no inputs, got %v", noneInputs)
	}
}

// TestPackLayout_MissingInputPathFailsLoud proves a rule whose declared input
// path is absent on disk is a blocking broken-pack error naming the pack and the
// missing path (CLM-050 / REQ-021): gathering inputs for a rule pointing at a
// non-existent rule file fails loud, never a silent skip. Substantive: drives the
// real gatherEngineInputs against a missing path and asserts the error names both
// the pack and the absent file.
func TestPackLayout_MissingInputPathFailsLoud(t *testing.T) {
	packRoot := filepath.Join(engineDispatchPacksDir(t), "test-org", "engine-pack")
	manifest := &pack.Manifest{NormalizedName: "test-org/engine-pack"}

	_, err := gatherEngineInputs(manifest, packRoot, resolveEngineRegistry(nil)["semgrep"], []pack.Rule{
		{ID: "ghost", Engine: "semgrep", RulePath: "semgrep/does-not-exist.yml"},
	})
	if err == nil {
		t.Fatal("a rule whose declared input path is absent must fail loud, got nil — that is a silent skip")
	}
	if !strings.Contains(err.Error(), "test-org/engine-pack") {
		t.Errorf("broken-pack error must name the pack, got: %v", err)
	}
	if !strings.Contains(err.Error(), "does-not-exist.yml") {
		t.Errorf("broken-pack error must name the missing input path, got: %v", err)
	}
}

// argsContainFlagValue reports whether args contains the pair `flag value`
// consecutively.
func argsContainFlagValue(args []string, flag, value string) bool {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == flag && args[i+1] == value {
			return true
		}
	}
	return false
}
