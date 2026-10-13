package packval

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// phase1_success_exit_codes_test.go (PLAN-ISSUE-207 TASK-004, CLM-009). packval
// decodes engines straight into engine.EngineBinding, so success_exit_codes is
// validated by phase1-structural (the shared engine.ValidateSuccessExitCodes
// authority) and a non-integer value is a parse-manifest error.

// secP1PackDir writes a minimal enforcement pack whose engine `tool` carries the
// given success_exit_codes line (empty = absent) and returns its dir.
func secP1PackDir(t *testing.T, successLine string) string {
	t.Helper()
	dir := t.TempDir()
	engineBlock := "  tool:\n    command: grep -rn -H -I\n    input_mode: none\n    gate_type: findings\n"
	if successLine != "" {
		engineBlock += "    " + successLine + "\n"
	}
	body := `name: acme/success-exit
version: 1.0.0
language: generic
archetype: enforcement
description: ISSUE-207 phase1 fixture
engines:
` + engineBlock + `content:
  ruleset:
    version: 1.0.0
    rules:
      - id: marker
        engine: tool
        risk_class: correctness
`
	if err := os.WriteFile(filepath.Join(dir, "pack.yml"), []byte(body), 0o644); err != nil {
		t.Fatalf("write pack.yml: %v", err)
	}
	return dir
}

// secP1Phase returns the named phase result from a pipeline run.
func secP1Phase(t *testing.T, res *Result, name string) PhaseResult {
	t.Helper()
	for _, p := range res.Phases {
		if p.Phase == name {
			return p
		}
	}
	t.Fatalf("pipeline result has no %s phase: %+v", name, res.Phases)
	return PhaseResult{}
}

// TestRunStructural_SuccessExitCodesValidated (CLM-009).
func TestRunStructural_SuccessExitCodesValidated(t *testing.T) {
	for _, line := range []string{
		"success_exit_codes: []",
		"success_exit_codes: [1]",
		"success_exit_codes: [0, 256]",
		"success_exit_codes: [0, 1, 1]",
	} {
		t.Run(line, func(t *testing.T) {
			res := NewPipeline(secP1PackDir(t, line), PipelineOptions{Mode: "check"}).Run()
			p1 := secP1Phase(t, res, "phase1-structural")
			if p1.Status != "fail" {
				t.Fatalf("%q must fail phase1-structural, got %q (errors %+v)", line, p1.Status, p1.Errors)
			}
			named := false
			for _, e := range p1.Errors {
				if e.Check == "engine-success-exit-codes" &&
					e.ManifestPath == "engines.tool.success_exit_codes" &&
					strings.Contains(e.Message, "tool") && strings.Contains(e.Message, "success_exit_codes") {
					named = true
				}
			}
			if !named {
				t.Errorf("%q: want an engine-success-exit-codes error naming engine tool and success_exit_codes, got %+v", line, p1.Errors)
			}
		})
	}

	t.Run("non-integer element", func(t *testing.T) {
		res := NewPipeline(secP1PackDir(t, `success_exit_codes: [0, "x"]`), PipelineOptions{Mode: "check"}).Run()
		p1 := secP1Phase(t, res, "phase1-structural")
		if p1.Status != "fail" || len(p1.Errors) == 0 || p1.Errors[0].Check != "parse-manifest" {
			t.Errorf("a non-integer element must be a parse-manifest error, got %+v", p1)
		}
	})

	t.Run("valid set is clean", func(t *testing.T) {
		res := NewPipeline(secP1PackDir(t, "success_exit_codes: [0, 1]"), PipelineOptions{Mode: "check"}).Run()
		p1 := secP1Phase(t, res, "phase1-structural")
		if p1.Status != "pass" {
			t.Errorf("success_exit_codes: [0, 1] must pass phase1, got %q (errors %+v)", p1.Status, p1.Errors)
		}
	})
}
