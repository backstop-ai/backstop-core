package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/gate"
	"github.com/backstop-ai/backstop-core/pkg/pack/engine"
)

// ISSUE-112 CLM-008 — END TO END: an installed pack whose declared engine binary
// does not exist must make the gate REFUSE, loudly, naming the binary.
//
// This is the acceptance fixture the issue asks for, expressed in this repo's
// terms. The issue's literal repro ("any repo + typescript-substantiveness on a
// PATH without ast-grep") is not reproducible here — there is no TypeScript pack
// installed, and removing a tool from the host is not something a test may do — so
// this builds the EQUIVALENT: a locally-installed fixture pack whose engine
// command's argv[0] is a binary that exists nowhere. Absence is STRUCTURAL, never
// a PATH mutation and never a skip.
//
// ── WHY THE FIXTURE'S Provision MUST BE NON-NIL AND ALLOWLIST-PINNED ─────────
// A nil-Provision binding would make this whole test VACUOUS, in a way that is
// easy to miss because it still goes green: it would have been ALREADY RED AT
// HEAD. At HEAD a nil-Provision binding's argv[0] was collected into the required
// set and a LookPath miss already returned a ConfigError naming it — so the silent
// pass this test exists to capture would have been unobtainable, and the test
// would have "passed" without a defect underneath it. The pinned branch is the one
// HEAD exempted, and therefore the only one that can demonstrate the defect.
//
// The pinned tool + version come from the PRODUCTION trusted-tool allowlist, read
// at run time rather than hardcoded, so a future pin bump cannot silently convert
// this refusal into a version-divergence refusal wearing the same fail+ConfigErr
// shape. No withTestAllowlist seam is installed: CLM-008 is the end-to-end claim,
// and this is the one test in the lane that should carry no seam at all.
//
// ── WHY THE ASSERTION IS ON THE STEP RESULT, NOT A WHOLE-GATE VERDICT ────────
// The workspace declares packs: in backstop.yml and deliberately ships NO
// backstop.lock, so a whole-gate run fails at pack_lock_verification and the kill
// chain halts before pack_engines ever runs. A test asserting "non-zero,
// config-error" on the whole gate would therefore pass for the WRONG reason — a
// lock refusal, not an absent-tool refusal — and its anti-vacuous twin would be
// impossible to write at all, since a lock-less workspace can never produce a
// passing whole-gate run. Driving buildGateSteps and inspecting the pack_engines
// StepResult is both simpler and strictly more precise about WHICH refusal fired.

// absentEngineToolName is a binary that exists nowhere on any PATH and is
// deliberately DIVERGENT from the pinned Provision.Tool below. That divergence is
// the fixture, not an accident: a pack may legitimately pin `ast-grep` and invoke
// `sg`, so the refusal must name the PROBED argv[0] — the binary a user has to
// actually install — and not only the pin it rode in on.
const absentEngineToolName = "backstop-absent-engine-112"

// pinnedAllowlistTool returns a tool + version straight from the PRODUCTION
// trusted-tool allowlist, so the fixture's trust gate passes for real reasons. It
// fatals rather than guessing if the allowlist ever stops carrying the tool.
func pinnedAllowlistTool(t *testing.T, tool string) string {
	t.Helper()
	version, ok := engine.TrustedToolAllowlist()[tool]
	if !ok {
		t.Fatalf("fixture invariant: %q must be on the production trusted-tool allowlist so the trust gate passes and the PRESENCE check is what fires", tool)
	}
	if version == "" {
		t.Fatalf("fixture invariant: %q must carry a pinned version", tool)
	}
	return version
}

// absentEngineWorkspace writes a temp project with ONE locally-installed fixture
// pack whose single engine's command is `command`.
//
// It writes the pack directly into .backstop/packs/ rather than installing it via
// `pack add`, for two reasons. First, `pack add` runs the packval pipeline, whose
// phase 3 EXECUTES a tier: complete scaffold's claim fixtures and refuses an
// unstartable run at INSTALL time — for a fixture pack whose whole point is an
// absent binary, that refusal would fire before the gate ever ran and the test
// would assert the wrong mechanism. Second, gate_substantiveness_e2e.go's workspace
// builders are off limits to this lane (they belong to a sibling plan this cycle),
// so the fixture stays local to this file. The pack declares NO claim fixtures.
//
// The engine declares NO convert and NO stdout_artifact. Both omissions are
// load-bearing: a declared-but-absent convert hard-errors "missing convert script",
// and a declared stdout_artifact is ALWAYS missing after a binary that never ran,
// hard-erroring "not produced". Either would replace the silent pass this test must
// observe at HEAD with a loud error for an unrelated reason, and would keep firing
// after the fix — so the green would prove the wrong thing too.
func absentEngineWorkspace(t *testing.T, command string) string {
	t.Helper()
	projectRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectRoot, "backstop.yml"), []byte(
		"project: p\nlanguage: go\npacks:\n  org/absent: \"1.0.0\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	packRoot := filepath.Join(projectRoot, ".backstop", "packs", "org", "absent")
	if err := os.MkdirAll(filepath.Join(packRoot, "rules"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(packRoot, "rules", "r.yml"), []byte("rules: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	const pinnedTool = "ast-grep"
	pinnedVersion := pinnedAllowlistTool(t, pinnedTool)
	manifest := fmt.Sprintf(`name: org/absent
version: 1.0.0
language: go
archetype: enforcement
description: Fixture pack whose declared engine binary does not exist
engines:
  absent-engine:
    command: %s
    input_mode: rule-flags
    input_flag: --config
    scope_kind: file-args
    category: opinion
    gate_type: findings
    provision:
      tool: %s
      version: %q
content:
  ruleset:
    version: 1.0.0
    rules:
      - id: r1
        standard: standards/go/r1.standard.md
        rule_path: rules/r.yml
        risk_class: correctness
        engine: absent-engine
        claims:
          - id: c-r1
            text: Rule one.
            fixtures:
              positive:
                - fixtures/positive.go
              negative:
                - fixtures/negative.go
`, command, pinnedTool, pinnedVersion)
	if err := os.WriteFile(filepath.Join(packRoot, "pack.yml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	return projectRoot
}

// packEnginesStepResult drives the REAL gate wiring over projectRoot and returns
// the pack_engines StepResult. Running the steps and selecting by StepName is the
// shipped precedent (gate_buildsteps_test.go).
func packEnginesStepResult(t *testing.T, projectRoot string) gate.StepResult {
	t.Helper()
	steps := buildGateSteps(projectRoot, rootAtDir(t, projectRoot))
	for _, step := range steps {
		res := step(context.Background())
		if res.StepName == "pack_engines" {
			return res
		}
	}
	t.Fatal("expected a pack_engines step in the gate step list")
	return gate.StepResult{}
}

// TestE2E_AbsentEngineTool_GateRefusesLoudlyNamingTool is CLM-008: driving the real
// gate wiring over a workspace whose installed pack declares an engine binary that
// does not exist yields a pack_engines refusal — fail + ConfigErr (the gate's exit-2
// class) — with the absent binary NAMED, and never a silent pass.
//
// It asserts the OUTCOME, not WHICH mechanism caught it. After this lane both the
// provisioning presence check and the dispatch never-started refusal can catch this
// shape, and belt-and-braces is deliberate: pinning the mechanism would make the
// test brittle against a future rewiring while proving nothing extra.
func TestE2E_AbsentEngineTool_GateRefusesLoudlyNamingTool(t *testing.T) {
	projectRoot := absentEngineWorkspace(t, absentEngineToolName+" --sarif")
	res := packEnginesStepResult(t, projectRoot)

	if res.Status != "fail" || !res.ConfigErr {
		t.Fatalf("an installed pack whose engine binary does not exist must make the gate REFUSE: got status=%q configErr=%v violations=%#v — a findings engine that never ran was reported as a clean scan",
			res.Status, res.ConfigErr, res.Violations)
	}
	if len(res.Violations) == 0 {
		t.Fatal("the refusal must carry a violation a human can read; got none")
	}
	msg := res.Violations[0].Message
	// The PROBED argv[0] is what the user must actually put on PATH, so it is the
	// name the message has to carry — not merely the pinned Provision.Tool.
	if !strings.Contains(msg, absentEngineToolName) {
		t.Errorf("the refusal must NAME the absent binary %q so a human learns WHICH tool is missing, got: %s", absentEngineToolName, msg)
	}
	// A refusal must not also emit fabricated downstream findings — the failure mode
	// where an engine that could not run still produces violations about the code.
	for _, v := range res.Violations {
		if v.Rule != "pack_engines" {
			t.Errorf("a refused run must emit no downstream findings; got a %q violation: %s", v.Rule, v.Message)
		}
	}
}

// TestE2E_PresentEngineTool_GateStillRuns is the anti-vacuous twin: the SAME
// workspace shape whose engine command DOES start is NOT refused. Without it,
// "the gate refuses an absent tool" is satisfiable by a gate that refuses
// everything.
//
// This twin is only writable BECAUSE the assertion is per-step — the lock-less
// workspace can never produce a passing whole-gate run (see the file header).
func TestE2E_PresentEngineTool_GateStillRuns(t *testing.T) {
	// A real, executable engine that starts and emits an empty-but-valid SARIF run.
	// An absolute path is a legitimate binding shape and resolves on PATH lookup.
	scriptDir := t.TempDir()
	scriptPath := filepath.Join(scriptDir, "present-engine.sh")
	if err := os.WriteFile(scriptPath,
		[]byte("#!/bin/sh\necho '{\"version\":\"2.1.0\",\"runs\":[{\"tool\":{\"driver\":{\"name\":\"present\"}},\"results\":[]}]}'\n"),
		0o755); err != nil { // #nosec G306 — the exec bit IS the fixture: this engine must genuinely start
		t.Fatalf("write present engine script: %v", err)
	}

	projectRoot := absentEngineWorkspace(t, scriptPath+" --sarif")
	res := packEnginesStepResult(t, projectRoot)

	if res.ConfigErr {
		t.Fatalf("an engine whose binary DOES exist must not be refused as a config error: status=%q violations=%#v", res.Status, res.Violations)
	}
	if res.Status != "pass" {
		t.Errorf("an engine that starts and reports no findings must yield a passing pack_engines step, got status=%q violations=%#v", res.Status, res.Violations)
	}
	for _, v := range res.Violations {
		if strings.Contains(v.Message, "not found on PATH") || strings.Contains(v.Message, "never started") {
			t.Errorf("a present engine must not be reported as missing or unstartable, got: %s", v.Message)
		}
	}
}
