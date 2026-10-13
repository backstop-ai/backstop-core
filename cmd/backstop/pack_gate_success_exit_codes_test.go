package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/check"
	"github.com/backstop-ai/backstop-core/pkg/gate"
	"github.com/backstop-ai/backstop-core/pkg/pack"
)

// pack_gate_success_exit_codes_test.go (PLAN-ISSUE-207 TASK-002). Gate-side
// findings-engine exit/payload semantics, including the issue's reproduced
// trigger. Every manifest is YAML TEXT parsed by pack.ParseManifestFile, so an
// unwired success_exit_codes key is silently ignored before the fix — that silence
// is the behavioural red. Engines are real POSIX sh scripts run through the gate's
// own runner shape (check.ExecCommandRunner{Dir: projectRoot}); every engine script
// except the repro's is referenced by ABSOLUTE path, so cwd plays no role elsewhere.

// secSarifOneResult is a one-result SARIF log naming bad.txt.
func secSarifOneResult() string {
	return `{"version":"2.1.0","runs":[{"results":[{"ruleId":"marker","level":"error","message":{"text":"MARKER found"},"locations":[{"physicalLocation":{"artifactLocation":{"uri":"bad.txt"},"region":{"startLine":1}}}]}]}]}`
}

// secSarifEmpty is an empty-results SARIF envelope.
func secSarifEmpty() string {
	return `{"version":"2.1.0","runs":[{"results":[]}]}`
}

// secFixture is one temp pack + project pair.
type secFixture struct {
	packsDir    string
	packRoot    string
	projectRoot string
}

// secNewFixture builds <packsDir>/acme/<name>/ and an empty project root.
func secNewFixture(t *testing.T, name string) secFixture {
	t.Helper()
	packsDir := t.TempDir()
	packRoot := filepath.Join(packsDir, "acme", name)
	mkDirAll(t, packRoot)
	projectRoot := t.TempDir()
	return secFixture{packsDir: packsDir, packRoot: packRoot, projectRoot: projectRoot}
}

// secScript writes an executable POSIX sh script under dir and returns its path.
func secScript(t *testing.T, dir, name, body string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	mkDirAll(t, filepath.Dir(path))
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body), 0o755); err != nil {
		t.Fatalf("write script %s: %v", path, err)
	}
	return path
}

// secManifest writes the pack.yml for engine `tool` (the given engine body lines,
// each already indented under the engine key) and parses it with the production
// reader.
func secManifest(t *testing.T, fx secFixture, name, engineLines string) *pack.Manifest {
	t.Helper()
	body := fmt.Sprintf(`name: acme/%s
version: 1.0.0
language: generic
archetype: enforcement
description: ISSUE-207 success_exit_codes fixture
engines:
  tool:
%s
content:
  ruleset:
    version: 1.0.0
    rules:
      - id: marker
        engine: tool
        risk_class: correctness
`, name, engineLines)
	path := filepath.Join(fx.packRoot, "pack.yml")
	writeFileStr(t, path, body)
	m, err := pack.ParseManifestFile(path)
	if err != nil {
		t.Fatalf("fixture manifest must parse: %v\n%s", err, body)
	}
	return m
}

// secDispatch runs the manifest through the real dispatcher with the gate's runner
// shape and the direct-convert sandbox double.
func secDispatch(t *testing.T, fx secFixture, m *pack.Manifest) ([]gate.Violation, error) {
	t.Helper()
	runner := &check.ExecCommandRunner{Dir: fx.projectRoot}
	result, err := dispatchPackEnginesWithEvidence([]*pack.Manifest{m}, fx.packsDir, fx.projectRoot, nil, runner, directConvertSandboxRunner(nil))
	return result.Violations, err
}

// requireSecD2 asserts err is the D-2 engine failure: a *check.ConfigError whose
// text names every wanted fragment.
func requireSecD2(t *testing.T, err error, wants ...string) {
	t.Helper()
	if err == nil {
		t.Fatal("expected the empty-payload engine failure, got a nil error (a vacuous clean pass)")
	}
	var cfgErr *check.ConfigError
	if !errors.As(err, &cfgErr) {
		t.Errorf("the engine failure must be a *check.ConfigError (exit 2), got %T: %v", err, err)
	}
	for _, want := range wants {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error must contain %q, got: %v", want, err)
		}
	}
}

// requireSecClean asserts a nil error and zero findings.
func requireSecClean(t *testing.T, vs []gate.Violation, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("expected a clean result, got error: %v", err)
	}
	if len(vs) != 0 {
		t.Fatalf("expected zero violations, got %#v", vs)
	}
}

// requireSecOneFinding asserts a nil error and exactly the one bad.txt finding.
func requireSecOneFinding(t *testing.T, vs []gate.Violation, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("a non-empty payload must be parsed as findings, got error: %v", err)
	}
	if len(vs) != 1 {
		t.Fatalf("expected exactly one finding, got %#v", vs)
	}
	if vs[0].File != "bad.txt" || !strings.Contains(vs[0].Rule, "marker") || !strings.Contains(vs[0].Message, "MARKER found") {
		t.Errorf("finding must carry the SARIF result's file/rule/message, got %#v", vs[0])
	}
}

// TestFindingsEngine_PackRelativeScriptInCommandFailsLoud (CLM-001, CLM-002): the
// issue's trigger. command: "sh tool.sh" with tool.sh only in the pack root runs
// with cwd = projectRoot, so sh cannot find it, exits non-zero and prints nothing.
func TestFindingsEngine_PackRelativeScriptInCommandFailsLoud(t *testing.T) {
	fx := secNewFixture(t, "repro")
	secScript(t, fx.packRoot, "tool.sh", "cat <<'SARIF'\n"+secSarifOneResult()+"\nSARIF\nexit 1\n")
	writeFileStr(t, filepath.Join(fx.projectRoot, "bad.txt"), "MARKER\n")
	m := secManifest(t, fx, "repro", "    command: sh tool.sh\n    input_mode: none\n    gate_type: findings")

	// The expected status is whatever this machine's sh reports for a missing
	// script when run from the project root — measured, not assumed.
	probe := exec.Command("sh", "tool.sh")
	probe.Dir = fx.projectRoot
	probeErr := probe.Run()
	var exitErr *exec.ExitError
	if !errors.As(probeErr, &exitErr) || exitErr.ExitCode() <= 0 {
		t.Fatalf("fixture invariant: `sh tool.sh` from the project root must exit non-zero, got %v", probeErr)
	}
	status := exitErr.ExitCode()

	vs, err := secDispatch(t, fx, m)
	if err == nil && len(vs) == 0 {
		t.Fatal("THE DEFECT: a pack-relative script that never ran read as a clean pass (nil error, zero violations)")
	}
	requireSecD2(t, err, "acme/repro", "sh tool.sh", fmt.Sprintf("exit status %d", status), "success_exit_codes")
}

// TestFindingsEngine_EmptyPayloadNonZeroExitUndeclaredFails (CLM-002, CLM-005).
func TestFindingsEngine_EmptyPayloadNonZeroExitUndeclaredFails(t *testing.T) {
	fx := secNewFixture(t, "empty-exit1")
	script := secScript(t, fx.packRoot, "engine.sh", "exit 1\n")
	m := secManifest(t, fx, "empty-exit1", "    command: "+script+"\n    input_mode: none\n    gate_type: findings")

	_, err := secDispatch(t, fx, m)
	requireSecD2(t, err, "acme/empty-exit1", script, "exit status 1", "success_exit_codes: [0, 1]")
}

// TestFindingsEngine_DeclaredSuccessExitCodeEmptyPayloadIsClean (CLM-003; control).
func TestFindingsEngine_DeclaredSuccessExitCodeEmptyPayloadIsClean(t *testing.T) {
	fx := secNewFixture(t, "declared-clean")
	script := secScript(t, fx.packRoot, "engine.sh", "exit 1\n")
	m := secManifest(t, fx, "declared-clean", "    command: "+script+"\n    input_mode: none\n    gate_type: findings\n    success_exit_codes: [0, 1]")

	vs, err := secDispatch(t, fx, m)
	requireSecClean(t, vs, err)
}

// TestFindingsEngine_DeclaredSetDoesNotAcceptOtherStatuses (CLM-003).
func TestFindingsEngine_DeclaredSetDoesNotAcceptOtherStatuses(t *testing.T) {
	fx := secNewFixture(t, "declared-other")
	script := secScript(t, fx.packRoot, "engine.sh", "exit 2\n")
	m := secManifest(t, fx, "declared-other", "    command: "+script+"\n    input_mode: none\n    gate_type: findings\n    success_exit_codes: [0, 1]")

	_, err := secDispatch(t, fx, m)
	requireSecD2(t, err, "acme/declared-other", "exit status 2", "success_exit_codes")
}

// TestFindingsEngine_NonEmptyPayloadNonZeroExitStillParsesFindings (CLM-004; control).
func TestFindingsEngine_NonEmptyPayloadNonZeroExitStillParsesFindings(t *testing.T) {
	fx := secNewFixture(t, "nonempty-exit3")
	script := secScript(t, fx.packRoot, "engine.sh", "cat <<'SARIF'\n"+secSarifOneResult()+"\nSARIF\nexit 3\n")
	m := secManifest(t, fx, "nonempty-exit3", "    command: "+script+"\n    input_mode: none\n    gate_type: findings")

	vs, err := secDispatch(t, fx, m)
	requireSecOneFinding(t, vs, err)
}

// TestFindingsEngine_StdoutArtifactEmptyFailsDespiteNoisyStdout (CLM-006): the raw
// output is the ARTIFACT, so noisy stdout does not make an empty artifact non-empty.
func TestFindingsEngine_StdoutArtifactEmptyFailsDespiteNoisyStdout(t *testing.T) {
	fx := secNewFixture(t, "artifact-empty")
	script := secScript(t, fx.packRoot, "engine.sh", "echo 'ran 12 checks'\n: > out.sarif\nexit 1\n")
	m := secManifest(t, fx, "artifact-empty", "    command: "+script+"\n    input_mode: none\n    gate_type: findings\n    stdout_artifact: out.sarif")

	_, err := secDispatch(t, fx, m)
	requireSecD2(t, err, "acme/artifact-empty", "exit status 1", "success_exit_codes")
}

// TestFindingsEngine_StdoutArtifactNonEmptyNonZeroExitParses (CLM-006, CLM-004; control).
func TestFindingsEngine_StdoutArtifactNonEmptyNonZeroExitParses(t *testing.T) {
	fx := secNewFixture(t, "artifact-nonempty")
	script := secScript(t, fx.packRoot, "engine.sh", "echo 'ran 12 checks'\ncat > out.sarif <<'SARIF'\n"+secSarifOneResult()+"\nSARIF\nexit 1\n")
	m := secManifest(t, fx, "artifact-nonempty", "    command: "+script+"\n    input_mode: none\n    gate_type: findings\n    stdout_artifact: out.sarif")

	vs, err := secDispatch(t, fx, m)
	requireSecOneFinding(t, vs, err)
}

// secConvertScript writes a pack-relative convert that records that it ran (an
// absolute marker path baked in) and emits the given SARIF.
func secConvertScript(t *testing.T, fx secFixture, sarif string) string {
	t.Helper()
	marker := filepath.Join(fx.packRoot, "convert-ran.txt")
	secScript(t, fx.packRoot, "convert/to-sarif.sh", "cat > /dev/null\n: > '"+marker+"'\ncat <<'SARIF'\n"+sarif+"\nSARIF\n")
	return marker
}

// TestFindingsEngine_ConvertEngineEmptyRawOutputFailsBeforeConvert (CLM-006,
// CLM-002): grep-shaped crash. The convert would wrap the empty input in an empty
// SARIF envelope; the check precedes it, so the convert never runs.
func TestFindingsEngine_ConvertEngineEmptyRawOutputFailsBeforeConvert(t *testing.T) {
	fx := secNewFixture(t, "convert-crash")
	script := secScript(t, fx.packRoot, "engine.sh", "exit 2\n")
	marker := secConvertScript(t, fx, secSarifEmpty())
	m := secManifest(t, fx, "convert-crash", "    command: "+script+"\n    input_mode: none\n    gate_type: findings\n    convert: convert/to-sarif.sh")

	_, err := secDispatch(t, fx, m)
	requireSecD2(t, err, "acme/convert-crash", "exit status 2", "success_exit_codes")
	if _, statErr := os.Stat(marker); statErr == nil {
		t.Error("the convert must NOT run over an empty raw output with an unaccepted exit status")
	}
}

// TestFindingsEngine_ConvertEngineDeclaredNoMatchStatusIsClean (CLM-003, CLM-006):
// grep no-match shape. Declared [0, 1] is clean (the convert runs and its envelope
// parses to zero findings); the same manifest without the key fails.
func TestFindingsEngine_ConvertEngineDeclaredNoMatchStatusIsClean(t *testing.T) {
	t.Run("declared", func(t *testing.T) {
		fx := secNewFixture(t, "convert-nomatch")
		script := secScript(t, fx.packRoot, "engine.sh", "exit 1\n")
		marker := secConvertScript(t, fx, secSarifEmpty())
		m := secManifest(t, fx, "convert-nomatch", "    command: "+script+"\n    input_mode: none\n    gate_type: findings\n    convert: convert/to-sarif.sh\n    success_exit_codes: [0, 1]")

		vs, err := secDispatch(t, fx, m)
		requireSecClean(t, vs, err)
		if _, statErr := os.Stat(marker); statErr != nil {
			t.Errorf("with the status declared the convert must run: %v", statErr)
		}
	})
	t.Run("undeclared", func(t *testing.T) {
		fx := secNewFixture(t, "convert-nomatch-undeclared")
		script := secScript(t, fx.packRoot, "engine.sh", "exit 1\n")
		secConvertScript(t, fx, secSarifEmpty())
		m := secManifest(t, fx, "convert-nomatch-undeclared", "    command: "+script+"\n    input_mode: none\n    gate_type: findings\n    convert: convert/to-sarif.sh")

		_, err := secDispatch(t, fx, m)
		requireSecD2(t, err, "acme/convert-nomatch-undeclared", "exit status 1", "success_exit_codes: [0, 1]")
	})
}

// TestFindingsEngine_ConvertEngineNonEmptyRawOutputIsParsedAfterConvert (CLM-004,
// CLM-006; control).
func TestFindingsEngine_ConvertEngineNonEmptyRawOutputIsParsedAfterConvert(t *testing.T) {
	fx := secNewFixture(t, "convert-nonempty")
	script := secScript(t, fx.packRoot, "engine.sh", "echo 'bad.txt:1:MARKER'\nexit 1\n")
	secConvertScript(t, fx, secSarifOneResult())
	m := secManifest(t, fx, "convert-nonempty", "    command: "+script+"\n    input_mode: none\n    gate_type: findings\n    convert: convert/to-sarif.sh")

	vs, err := secDispatch(t, fx, m)
	requireSecOneFinding(t, vs, err)
}

// TestFindingsEngine_CrashGuardHonorsDeclaredSuccessExitCodes (CLM-007).
func TestFindingsEngine_CrashGuardHonorsDeclaredSuccessExitCodes(t *testing.T) {
	fx := secNewFixture(t, "crashguard-declared")
	script := secScript(t, fx.packRoot, "engine.sh", "cat <<'SARIF'\n"+secSarifEmpty()+"\nSARIF\nexit 1\n")
	m := secManifest(t, fx, "crashguard-declared", "    command: "+script+"\n    input_mode: none\n    gate_type: findings\n    crash_guard: true\n    success_exit_codes: [0, 1]")

	vs, err := secDispatch(t, fx, m)
	requireSecClean(t, vs, err)
}

// TestFindingsEngine_CrashGuardUndeclaredKeepsCrashWording (CLM-007; control).
func TestFindingsEngine_CrashGuardUndeclaredKeepsCrashWording(t *testing.T) {
	fx := secNewFixture(t, "crashguard-undeclared")
	script := secScript(t, fx.packRoot, "engine.sh", "exit 1\n")
	m := secManifest(t, fx, "crashguard-undeclared", "    command: "+script+"\n    input_mode: none\n    gate_type: findings\n    crash_guard: true")

	_, err := secDispatch(t, fx, m)
	if err == nil {
		t.Fatal("an undeclared CrashGuard engine exiting 1 with nothing parseable must still crash loud")
	}
	if !strings.Contains(err.Error(), "crashed: non-zero exit with no parseable findings") {
		t.Errorf("the CrashGuard wording must stay authoritative, got: %v", err)
	}
	if strings.Contains(err.Error(), "success_exit_codes") {
		t.Errorf("a CrashGuard engine must not get the empty-payload wording, got: %v", err)
	}
}

// TestFindingsEngine_RunErrorWithoutExitStatusIsNotAccepted (CLM-005): an error
// carrying no exit status is never accepted, so it cannot hide behind the set.
func TestFindingsEngine_RunErrorWithoutExitStatusIsNotAccepted(t *testing.T) {
	fx := secNewFixture(t, "no-status")
	m := secManifest(t, fx, "no-status", "    command: tool\n    input_mode: none\n    gate_type: findings")
	runner := &fixtureRunner{
		byCmd:    map[string][]byte{"tool": nil},
		byCmdErr: map[string]error{"tool": errors.New("boom")},
	}

	_, err := runFindingsEngine(m, fx.packRoot, fx.projectRoot, nil, m.Engines["tool"].Binding, m.Content.Ruleset.Rules, runner, findingsSandboxExecution{runner: directConvertSandboxRunner(nil)})
	if err == nil {
		t.Fatal("a run error with no exit status and an empty payload must not read as a clean pass")
	}
	for _, want := range []string{"no exit status", "boom"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error must contain %q, got: %v", want, err)
		}
	}
}
