package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// SPEC-055 REQ-011, the claims that can only be settled on a SEPARATED stream: the two
// commands whose silence was filed as a defect (CLM-074 pack relock / ISSUE-074,
// CLM-075 recipe apply / ISSUE-080), pack add (CLM-068), and the four sites that opt
// OUT of the loud default because they already printed a report (CLM-077..081).
//
// Every test in this file drives the BUILT BINARY through runBackstopStreams. Not one
// may use the merged helper: against a single CombinedOutput buffer "the diagnostic is
// on stderr" and "no Error line was added to stderr" are both unfalsifiable, because
// the report the command wrote to stdout is in the same buffer (spec Review Question 7).

// errorLinePrefix is the exact token reportError writes. The four no-duplicate claims
// assert its ABSENCE from stderr, so it is named once here rather than retyped.
const errorLinePrefix = "Error:"

// requireReportThenNoErrorLine is the assertion shape shared by the four explained
// sites. BOTH halves are load-bearing: the absence half alone would pass against a
// command that printed nothing at all — which is the very failure this spec exists to
// close — so the presence of the command's own report is asserted first.
func requireReportThenNoErrorLine(t *testing.T, stdout, stderr string, code int, wantReport ...string) {
	t.Helper()

	if code != ExitViolations {
		t.Errorf("exit %d, want ExitViolations (%d)\nstdout: %s\nstderr: %s", code, ExitViolations, stdout, stderr)
	}
	for _, fragment := range wantReport {
		if !strings.Contains(stdout, fragment) {
			t.Errorf("stdout does not carry the command's report fragment %q; without a report there is nothing for the opt-out to be justified by\nstdout: %s", fragment, stdout)
		}
	}
	if strings.Contains(stderr, errorLinePrefix) {
		t.Errorf("stderr carries an added %q line duplicating the report already on stdout\nstderr: %s", errorLinePrefix, stderr)
	}
}

// requireLoudOnStderr asserts a failing run put a readable diagnostic on the SEPARATED
// stderr and exited 1 — the shape every non-explained site must have.
func requireLoudOnStderr(t *testing.T, stdout, stderr string, code int, want ...string) {
	t.Helper()

	if code != ExitViolations {
		t.Errorf("exit %d, want ExitViolations (%d)\nstdout: %s\nstderr: %s", code, ExitViolations, stdout, stderr)
	}
	if strings.TrimSpace(stderr) == "" {
		t.Fatalf("stderr is EMPTY for a failing run: the silent exit-1 defect is still open\nstdout: %s", stdout)
	}
	for _, fragment := range want {
		if !strings.Contains(stderr, fragment) {
			t.Errorf("stderr does not name %q\nstderr: %s", fragment, stderr)
		}
	}
}

// TestExitSurfacing_PackAdd_PrintsDiagnostic — CLM-068. A local ref that names no pack
// is the hermetic failing path: it never reaches a remote, so this asserts the
// surfacing and nothing about the network.
func TestExitSurfacing_PackAdd_PrintsDiagnostic(t *testing.T) {
	bin := buildBackstopBinary(t)
	proj := newConsumerProject(t)

	stdout, stderr, code := runBackstopStreams(t, bin, proj, "pack", "add", "./absent-pack")

	requireLoudOnStderr(t, stdout, stderr, code, "absent-pack", "pack.yml")
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("stdout = %q for a failing pack add, want the diagnostic on stderr alone", stdout)
	}
}

// TestExitSurfacing_PackRelock_PrintsDiagnostic — CLM-074, closing the ISSUE-074
// SILENCE. `pack relock <name>` on an installed local pack exited 1 with an entirely
// empty stderr; the operator saw a bare failure and nothing else.
//
// This claim fixes the silence ONLY. Relock's underlying path-vs-name argument defect
// (it reads <arg>/pack.yml while remove/update/upgrade take a NAME) is ISSUE-074's other
// half and is NOT fixed here, so the run is asserted to fail LOUDLY, never to succeed.
func TestExitSurfacing_PackRelock_PrintsDiagnostic(t *testing.T) {
	bin := buildBackstopBinary(t)
	proj := stageRecipeE2EProject(t)

	stdout, stderr, code := runBackstopStreams(t, bin, proj, "pack", "relock", recipeE2EPassPack)

	requireLoudOnStderr(t, stdout, stderr, code, recipeE2EPassPack)
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("stdout = %q for a failing pack relock, want the diagnostic on stderr alone", stdout)
	}
}

// TestExitSurfacing_RecipeApply_PrintsDiagnostic — CLM-075, closing the ISSUE-080
// silence. The declared target is deliberately NOT staged, so the applier fails at the
// op with the error that carries the op's DECLARED manual instruction verbatim — the
// text that exists precisely so an unreachable site still leaves the operator with
// something to do, and which the old blanket suppression threw away.
//
// The expected fragment is read out of the fixture's own recipe manifest rather than
// retyped, so a fixture whose manual text changes cannot leave this asserting a string
// nothing produces.
func TestExitSurfacing_RecipeApply_PrintsDiagnostic(t *testing.T) {
	bin := buildBackstopBinary(t)
	proj := stageRecipeE2EProject(t)
	ref, manifest, _ := stagedRecipe(t, proj, recipeE2EPassPack, recipeE2EPassID)

	declaredManual := strings.TrimSpace(manifest.Ops[0].Manual)
	if declaredManual == "" {
		t.Fatal("the fixture recipe declares no manual instruction; this claim has nothing to prove survived")
	}
	// The manual is a folded YAML scalar, so compare on its first line — the whole
	// string carries line breaks the CLI reflows.
	manualFragment := strings.TrimSpace(strings.SplitN(declaredManual, "\n", 2)[0])

	stdout, stderr, code := runBackstopStreams(t, bin, proj, "recipe", "apply", ref)

	requireLoudOnStderr(t, stdout, stderr, code, manifest.Ops[0].Target, manualFragment)
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("stdout = %q for a failing recipe apply, want the diagnostic on stderr alone", stdout)
	}
}

// TestExitSurfacing_GateViolations_NoDuplicateErrorLine — CLM-077. The gate renders its
// full verdict to stdout before returning, so its ExitViolations error is EXPLAINED and
// must add nothing to stderr. This is also the test that keeps `gate --json` parseable:
// a trailing human line on the wrong stream is what broke the provenance suite.
func TestExitSurfacing_GateViolations_NoDuplicateErrorLine(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	bin := buildBackstopBinary(t)
	proj := gitInitRepoWithBackstop(t)

	stdout, stderr, code := runBackstopStreams(t, bin, proj, "gate")

	requireReportThenNoErrorLine(t, stdout, stderr, code, "Total violations:", "Steps:")
}

// TestExitSurfacing_GateConfigError_StillPrints — CLM-078, the falsifier for the
// opt-out's CONDITION. The gate's own exit-2 constructions must keep printing, so the
// opt-out is set only on the ExitViolations verdict. Gating it on "the command is gate"
// instead of "the code is ExitViolations" makes every gate misconfiguration silent, and
// this is the single easiest way to get that wrong.
//
// The mutually-exclusive-flags refusal is chosen deliberately: it is constructed by
// runGate ITSELF (gate.go:82), so it exercises the same construction site as the
// explained verdict rather than a pre-run guard that never reaches it.
func TestExitSurfacing_GateConfigError_StillPrints(t *testing.T) {
	bin := buildBackstopBinary(t)
	proj := gitInitRepoWithBackstop(t)

	stdout, stderr, code := runBackstopStreams(t, bin, proj, "gate", "--all", "--file", "does-not-matter.go")

	if code != ExitConfigError {
		t.Errorf("exit %d, want ExitConfigError (%d)\nstdout: %s\nstderr: %s", code, ExitConfigError, stdout, stderr)
	}
	if !strings.Contains(stderr, "mutually exclusive") {
		t.Errorf("stderr does not carry the gate's configuration refusal; the explained opt-out swallowed an exit-2 message\nstderr: %s", stderr)
	}
}

// stageFailingPack writes a pack directory whose manifest omits required fields, so
// pack check and pack test both reach a phase-1 FAIL and render a report before
// returning their violation. It is deliberately minimal: the point is the surfacing,
// not which fields are missing.
func stageFailingPack(t *testing.T) (string, string) {
	t.Helper()
	proj := t.TempDir()
	const packDir = "failing-pack"
	full := filepath.Join(proj, packDir)
	if err := os.MkdirAll(full, 0o755); err != nil {
		t.Fatalf("creating pack dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(full, "pack.yml"), []byte("name: demo-org/failing-pack\nversion: 1.0.0\n"), 0o644); err != nil {
		t.Fatalf("writing failing pack manifest: %v", err)
	}
	return proj, "./" + packDir
}

// TestExitSurfacing_PackCheck_NoDuplicateErrorLine — CLM-079.
func TestExitSurfacing_PackCheck_NoDuplicateErrorLine(t *testing.T) {
	bin := buildBackstopBinary(t)
	proj, packDir := stageFailingPack(t)

	stdout, stderr, code := runBackstopStreams(t, bin, proj, "pack", "check", packDir)

	requireReportThenNoErrorLine(t, stdout, stderr, code, "status: fail", "phase1-structural")
}

// TestExitSurfacing_PackTest_NoDuplicateErrorLine — CLM-080.
func TestExitSurfacing_PackTest_NoDuplicateErrorLine(t *testing.T) {
	bin := buildBackstopBinary(t)
	proj, packDir := stageFailingPack(t)

	stdout, stderr, code := runBackstopStreams(t, bin, proj, "pack", "test", packDir)

	requireReportThenNoErrorLine(t, stdout, stderr, code, "status: fail", "phase1-structural")
}

// invalidSpecFixture is a spec that parses but violates the schema, so artifact
// validate reaches its VIOLATIONS return (artifact_validate.go:365) rather than the
// ExitConfigError above it — the two have opposite dispositions and only the first is
// explained.
const invalidSpecFixture = `---
title: "SPEC-002: Invalid"
number: SPEC-002
created: "2026-04-01"
status: draft
schema_version: spec/v1
spec_version: 1.0.0

implementation:
  summary: Missing sections.
  package: pkg/test

verification:
  level: unit
  test_command: go test ./...
  coverage_threshold: 90
---

# SPEC-002: Invalid

## Overview

Missing sections.
`

// TestExitSurfacing_ArtifactValidate_NoDuplicateErrorLine — CLM-081.
func TestExitSurfacing_ArtifactValidate_NoDuplicateErrorLine(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	bin := buildBackstopBinary(t)
	proj := setupArtifactTestDir(t, artifactTestBackstopYML, map[string]string{
		"specs/SPEC-002-invalid.spec.md": invalidSpecFixture,
	})

	stdout, stderr, code := runBackstopStreams(t, bin, proj, "artifact", "validate", "--all")

	requireReportThenNoErrorLine(t, stdout, stderr, code, "SPEC-002")
}
