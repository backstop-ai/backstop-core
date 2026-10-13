package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/config"
	"github.com/backstop-ai/backstop-core/pkg/gate"
	"github.com/backstop-ai/backstop-core/pkg/packval"
)

// ISSUE-217 wiring: the configured enforcement.coverage_floor must reach the REAL
// step assembly (buildGateStepsWithSandbox, shared by gate, baseline, waiver and the
// init seams) from the project's own backstop.yml — not a floor the test hands to
// buildCoverageStep.
//
// THE FIXTURE. A temp project declares one local pack with a gate_type: coverage
// engine. Its producer is a no-op script run for real by the production runner; its
// convert is answered by a recording sandbox runner with ONE measured record:
// app/legacy.ts at 60/100. No spec exists, so the file has no qualifying spec
// threshold and the floor is the bar. 60 sits between the two configured floors
// used below (50 passes, 70 fails), and both are below the default 90.

const floorWiringFile = "app/legacy.ts"

const floorWiringPack = `name: fixture/cov-floor
version: 1.0.0
language: typescript
archetype: code
description: ISSUE-217 coverage-floor wiring fixture with one coverage engine.
engines:
  floor-coverage:
    command: floor-coverage
    producer: scripts/produce.sh
    input_mode: none
    convert: scripts/convert.sh
    scope_kind: project-wide
    category: mechanism
    gate_type: coverage
classification:
  source:
    - "**/*.ts"
  test:
    - "**/*.test.ts"
content:
  ruleset:
    rules:
      - id: floor-coverage
        engine: floor-coverage
        risk_class: correctness
        category: toolchain
        justification: Fixture coverage pass producing one canned per-file record.
`

// floorWiringRecords is the convert's output: the single measured record.
const floorWiringRecords = `[{"path":"app/legacy.ts","covered":60,"total":100,"measured":true,"excluded":false,"metric":"line"}]`

// writeFloorWiringProject lays out the temp project with the given enforcement
// block appended to backstop.yml ("" means no enforcement block at all).
func writeFloorWiringProject(t *testing.T, enforcement string) string {
	t.Helper()
	root := t.TempDir()
	packRoot := filepath.Join(root, ".backstop", "packs", "fixture", "cov-floor")
	files := map[string]string{
		"backstop.yml":                      "project: coverage-floor-wiring\npacks:\n  fixture/cov-floor: local\n" + enforcement,
		floorWiringFile:                     "export const legacy = 1;\n",
		filepath.Join("specs", ".gitkeep"):  "",
		filepath.Join(packRoot, "pack.yml"): floorWiringPack,
	}
	for rel, body := range files {
		path := rel
		if !filepath.IsAbs(path) {
			path = filepath.Join(root, rel)
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir for %s: %v", rel, err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
	if err := os.MkdirAll(filepath.Join(packRoot, "scripts"), 0o755); err != nil {
		t.Fatalf("mkdir pack scripts: %v", err)
	}
	for name, body := range map[string]string{
		"produce.sh": "#!/bin/sh\nexit 0\n",
		"convert.sh": "#!/bin/sh\ncat >/dev/null\n",
	} {
		if err := os.WriteFile(filepath.Join(packRoot, "scripts", name), []byte(body), 0o755); err != nil {
			t.Fatalf("write script %s: %v", name, err)
		}
	}
	return root
}

// floorWiringSandbox answers the pack's convert with the canned record and every
// other sandboxed call with empty output.
func floorWiringSandbox() *recordingSandboxRunner {
	return &recordingSandboxRunner{
		mode: packval.SandboxModeNative,
		stdoutFn: func(cmd string, _ []string, _ string, _ []byte) (packval.SandboxRunResult, error) {
			if strings.HasSuffix(filepath.ToSlash(cmd), "cov-floor/scripts/convert.sh") {
				return recordedSandboxResult([]byte(floorWiringRecords), true), nil
			}
			return recordedSandboxResult(nil, true), nil
		},
	}
}

// assembledFloorResults runs the real assembled steps over the project in file
// mode for the changed file, returning every step result in order.
func assembledFloorResults(t *testing.T, projectRoot string) []gate.StepResult {
	t.Helper()
	scope, err := gate.ComputeGateScope(projectRoot, gate.GateScopeModeFile, []string{floorWiringFile})
	if err != nil {
		t.Fatalf("ComputeGateScope: %v", err)
	}
	steps := buildGateStepsWithSandbox(projectRoot, rootAtDir(t, projectRoot), floorWiringSandbox(), scope)
	results := make([]gate.StepResult, 0, len(steps))
	for _, step := range steps {
		results = append(results, step(context.Background()))
	}
	return results
}

func coverageResultFrom(t *testing.T, results []gate.StepResult) gate.StepResult {
	t.Helper()
	for _, res := range results {
		if res.StepName == gate.StepCoverageThreshold {
			return res
		}
	}
	t.Fatalf("coverage step (%s) not present in the assembled gate steps: %#v", gate.StepCoverageThreshold, results)
	return gate.StepResult{}
}

func belowBarFor(res gate.StepResult, file string) []gate.Violation {
	var out []gate.Violation
	for _, v := range res.Violations {
		if v.Rule == "coverage_threshold" && v.File == file {
			out = append(out, v)
		}
	}
	return out
}

// TestCoverageFloorGateWiringUsesConfiguredValue (CLM-001, CLM-002, CLM-004).
func TestCoverageFloorGateWiringUsesConfiguredValue(t *testing.T) {
	if config.DefaultCoverageFloor != 90 {
		t.Fatalf("config.DefaultCoverageFloor = %d, want 90", config.DefaultCoverageFloor)
	}

	t.Run("unset fails below the default 90", func(t *testing.T) {
		res := coverageResultFrom(t, assembledFloorResults(t, writeFloorWiringProject(t, "")))
		got := belowBarFor(res, floorWiringFile)
		if res.Status != "fail" || len(got) != 1 {
			t.Fatalf("with no coverage_floor the 60%% file must fail below 90, got %s: %#v", res.Status, res.Violations)
		}
		if !strings.Contains(got[0].Message, "below threshold 90%") {
			t.Errorf("the bar must be the default 90, got %q", got[0].Message)
		}
	})

	t.Run("floor 50 passes", func(t *testing.T) {
		res := coverageResultFrom(t, assembledFloorResults(t, writeFloorWiringProject(t, "enforcement:\n  coverage_floor: 50\n")))
		if res.Status != "pass" {
			t.Fatalf("with coverage_floor: 50 the 60%% file must pass through the real assembly, got %s: %#v", res.Status, res.Violations)
		}
		if got := belowBarFor(res, floorWiringFile); len(got) != 0 {
			t.Errorf("no below-bar violation expected at floor 50, got %#v", got)
		}
	})

	t.Run("floor 70 fails below 70", func(t *testing.T) {
		res := coverageResultFrom(t, assembledFloorResults(t, writeFloorWiringProject(t, "enforcement:\n  coverage_floor: 70\n")))
		got := belowBarFor(res, floorWiringFile)
		if res.Status != "fail" || len(got) != 1 {
			t.Fatalf("with coverage_floor: 70 the 60%% file must fail, got %s: %#v", res.Status, res.Violations)
		}
		if !strings.Contains(got[0].Message, "below threshold 70%") {
			t.Errorf("the bar must be the configured 70, got %q", got[0].Message)
		}
	})

	// An invalid floor makes backstop.yml fail to load. The assembly loads the same
	// file for the pack set first, so the gate halts on a ConfigErr naming the key
	// before any coverage verdict exists — never a coverage step silently at 90.
	t.Run("floor 150 is a config error naming the key", func(t *testing.T) {
		results := assembledFloorResults(t, writeFloorWiringProject(t, "enforcement:\n  coverage_floor: 150\n"))
		named := false
		for _, res := range results {
			if res.StepName == gate.StepCoverageThreshold && !res.ConfigErr {
				t.Fatalf("an invalid coverage_floor must never yield a coverage verdict, got %s: %#v", res.Status, res.Violations)
			}
			if !res.ConfigErr || res.Status != "fail" {
				continue
			}
			for _, v := range res.Violations {
				if strings.Contains(v.Message, "enforcement.coverage_floor") {
					named = true
				}
			}
		}
		if !named {
			t.Fatalf("an invalid coverage_floor must surface as a ConfigErr naming enforcement.coverage_floor, got %#v", results)
		}
		if summary := gate.NewGateResult(results); summary.Pass {
			t.Errorf("an invalid coverage_floor must fail the gate, got pass")
		}
	})

	// The coverage step's own load-error branch: handed a floor load error, it is a
	// ConfigErr naming the error and never runs the threshold check at a default.
	t.Run("coverage step with a floor load error is a ConfigErr", func(t *testing.T) {
		root := writeFloorWiringProject(t, "")
		loadErr := errors.New("backstop.yml schema validation: invalid enforcement.coverage_floor 150")
		called := false
		records := func(*gate.GateScope) (coverageRecordsResult, error) {
			called = true
			return coverageRecordsResult{}, nil
		}
		step := buildCoverageStep(filepath.Join(root, "specs"), root, nil, gate.NewSourceClassifier([]string{"**/*.ts"}, nil), records, config.DefaultCoverageFloor, loadErr)
		res := step(context.Background())
		if res.StepName != gate.StepCoverageThreshold || res.Status != "fail" || !res.ConfigErr {
			t.Fatalf("a floor load error must make the coverage step a failing ConfigErr, got %#v", res)
		}
		if len(res.Violations) != 1 || res.Violations[0].Rule != "coverage_threshold" || !strings.Contains(res.Violations[0].Message, "enforcement.coverage_floor") {
			t.Errorf("the violation must carry the load error, got %#v", res.Violations)
		}
		if called {
			t.Error("with a floor load error the step must not produce records or judge coverage")
		}
	})

	t.Run("missing backstop.yml loads the default floor", func(t *testing.T) {
		floor, err := coverageFloorForProject(t.TempDir())
		if err != nil || floor != config.DefaultCoverageFloor {
			t.Errorf("coverageFloorForProject with no backstop.yml = (%d, %v), want (%d, nil)", floor, err, config.DefaultCoverageFloor)
		}
		root := writeFloorWiringProject(t, "enforcement:\n  coverage_floor: 0\n")
		if floor, err := coverageFloorForProject(root); err != nil || floor != 0 {
			t.Errorf("coverageFloorForProject with coverage_floor: 0 = (%d, %v), want (0, nil)", floor, err)
		}
		bad := writeFloorWiringProject(t, "enforcement:\n  coverage_floor: 150\n")
		if _, err := coverageFloorForProject(bad); err == nil || !strings.Contains(err.Error(), "enforcement.coverage_floor") {
			t.Errorf("coverageFloorForProject must return the load error as is, got %v", err)
		}
	})
}
