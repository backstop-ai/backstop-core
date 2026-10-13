package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// gate_pack_engines_empty_payload_test.go (PLAN-ISSUE-207 TASK-003). The
// pack_engines STEP verdict for the issue's reproduced trigger, and the unchanged
// gate cwd. Harness: a real projectRoot with backstop.yml and the pack installed
// under .backstop/packs/acme/<name> (the gate_scope_test.go shape), with the step
// obtained via runPackEnginesStep — buildGateSteps loads the installed pack
// hermetically, so no dispatcher seam is used and the real runner construction
// (check.ExecCommandRunner{Dir: projectRoot}) is what runs.

// sepStepInstallPack writes backstop.yml declaring acme/<name> and creates the
// installed pack root. It returns the project root and the pack root.
func sepStepInstallPack(t *testing.T, name string) (string, string) {
	t.Helper()
	projectRoot := t.TempDir()
	writeFileStr(t, filepath.Join(projectRoot, "backstop.yml"),
		"project: p\nlanguage: generic\npacks:\n  acme/"+name+": \"1.0.0\"\n")
	packRoot := filepath.Join(projectRoot, ".backstop", "packs", "acme", name)
	mkDirAll(t, packRoot)
	return projectRoot, packRoot
}

// sepStepWritePack writes the installed pack.yml: a single engine `tool` (the
// given indented engine lines) bound by one rule.
func sepStepWritePack(t *testing.T, packRoot, name, engineLines string) {
	t.Helper()
	writeFileStr(t, filepath.Join(packRoot, "pack.yml"), `name: acme/`+name+`
version: 1.0.0
language: generic
archetype: enforcement
description: ISSUE-207 pack_engines step fixture
engines:
  tool:
`+engineLines+`
content:
  ruleset:
    version: 1.0.0
    rules:
      - id: marker
        engine: tool
        risk_class: correctness
`)
}

// sepStepScript writes an executable POSIX sh script and returns its path.
func sepStepScript(t *testing.T, path, body string) string {
	t.Helper()
	mkDirAll(t, filepath.Dir(path))
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body), 0o755); err != nil {
		t.Fatalf("write script %s: %v", path, err)
	}
	return path
}

// TestPackEnginesStep_PackRelativeScriptIsNotAPass (CLM-008, CLM-001): the repro
// pack's engine never finds its pack-relative script from the project root, so the
// step must fail as a config error naming the failure — never report pass.
func TestPackEnginesStep_PackRelativeScriptIsNotAPass(t *testing.T) {
	projectRoot, packRoot := sepStepInstallPack(t, "repro")
	sepStepWritePack(t, packRoot, "repro", "    command: sh tool.sh\n    input_mode: none\n    gate_type: findings")
	sepStepScript(t, filepath.Join(packRoot, "tool.sh"),
		"cat <<'SARIF'\n"+`{"version":"2.1.0","runs":[{"results":[{"ruleId":"marker","level":"error","message":{"text":"MARKER found"},"locations":[{"physicalLocation":{"artifactLocation":{"uri":"bad.txt"},"region":{"startLine":1}}}]}]}]}`+"\nSARIF\nexit 1\n")
	writeFileStr(t, filepath.Join(projectRoot, "bad.txt"), "MARKER\n")

	res := runPackEnginesStep(t, projectRoot, nil)
	if res.Status == "pass" {
		t.Fatalf("THE DEFECT: the pack_engines step reported pass over an engine that never ran its script; violations=%#v", res.Violations)
	}
	if res.Status != "fail" {
		t.Errorf("want status fail, got %q", res.Status)
	}
	if !res.ConfigErr {
		t.Error("the engine failure must be a config error (exit 2)")
	}
	found := false
	for _, v := range res.Violations {
		msg := v.Message
		if strings.Contains(msg, "acme/repro") && strings.Contains(msg, "sh tool.sh") &&
			strings.Contains(msg, "exit status") && strings.Contains(msg, "success_exit_codes") {
			found = true
		}
	}
	if !found {
		t.Errorf("a violation must name the pack, the command, the exit status and success_exit_codes; got %#v", res.Violations)
	}
}

// TestPackEnginesStep_RunnerCwdStaysProjectRoot (CLM-010; control): the gate runs
// an engine with the PROJECT root as its working directory, never the pack root.
func TestPackEnginesStep_RunnerCwdStaysProjectRoot(t *testing.T) {
	projectRoot, packRoot := sepStepInstallPack(t, "cwd-probe")
	script := sepStepScript(t, filepath.Join(packRoot, "engine.sh"),
		": > cwd-probe.txt\ncat <<'SARIF'\n"+`{"version":"2.1.0","runs":[{"results":[]}]}`+"\nSARIF\nexit 0\n")
	sepStepWritePack(t, packRoot, "cwd-probe", "    command: "+script+"\n    input_mode: none\n    gate_type: findings")

	res := runPackEnginesStep(t, projectRoot, nil)
	if res.Status != "pass" {
		t.Fatalf("the probe engine exits 0 with empty-results SARIF and must pass; got status=%q violations=%#v", res.Status, res.Violations)
	}
	if _, err := os.Stat(filepath.Join(projectRoot, "cwd-probe.txt")); err != nil {
		t.Errorf("the engine must run with cwd = projectRoot (probe file absent there): %v", err)
	}
	if _, err := os.Stat(filepath.Join(packRoot, "cwd-probe.txt")); err == nil {
		t.Error("the engine must NOT run with cwd = pack root (probe file found there)")
	}
}
