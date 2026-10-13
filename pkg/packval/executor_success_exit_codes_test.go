package packval

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/pack/engine"
)

// executor_success_exit_codes_test.go (PLAN-ISSUE-207 TASK-004, CLM-011). The
// pack-test dispatch (DefaultExecutor.RunEngine) applies the same declared
// success-exit-code rule as the gate. The binding is decoded from YAML TEXT by
// packval.ParseManifest, so before the fix an unwired success_exit_codes key is
// silently dropped — that silence is the behavioural red. Engine scripts are real
// sh scripts referenced by ABSOLUTE path (writeConvertFixtureScript), cwd packDir.

// secPVBinding writes pack.yml (engine `tool` with the given indented engine lines)
// into packDir and returns the binding packval.ParseManifest resolved.
func secPVBinding(t *testing.T, packDir, engineLines string) engine.EngineBinding {
	t.Helper()
	body := "name: acme/success-exit\nversion: 1.0.0\nlanguage: generic\narchetype: enforcement\nengines:\n  tool:\n" + engineLines + "\n"
	path := filepath.Join(packDir, "pack.yml")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write pack.yml: %v", err)
	}
	m, err := ParseManifest(path)
	if err != nil {
		t.Fatalf("fixture manifest must parse: %v\n%s", err, body)
	}
	binding, ok := m.Engines["tool"]
	if !ok {
		t.Fatalf("fixture manifest must declare engine tool, got %+v", m.Engines)
	}
	return binding
}

// secPVRequireEmptyPayloadErr asserts the D-2 failure naming the command, the
// status and the success_exit_codes hint.
func secPVRequireEmptyPayloadErr(t *testing.T, res ExecutionResult, err error, command, status string) {
	t.Helper()
	if err == nil {
		t.Fatalf("THE DEFECT: an empty payload with an unaccepted exit returned a nil error (result %+v) — a negative fixture's pass over a crashed run", res)
	}
	if res.Passed {
		t.Errorf("expected a non-passing result alongside the error, got %+v", res)
	}
	for _, want := range []string{command, "exit status " + status, "success_exit_codes"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error must contain %q, got: %v", want, err)
		}
	}
}

// TestRunEngine_EmptyPayloadNonZeroExitUndeclaredFails (CLM-011).
func TestRunEngine_EmptyPayloadNonZeroExitUndeclaredFails(t *testing.T) {
	packDir := convertFixturePackDir(t)
	script := writeConvertFixtureScript(t, packDir, "engine.sh", "#!/bin/sh\nexit 1\n")
	binding := secPVBinding(t, packDir, "    command: "+script+"\n    input_mode: none")

	res, err := (&DefaultExecutor{}).RunEngine(packDir, binding, []string{"fixture.txt"})
	secPVRequireEmptyPayloadErr(t, res, err, script, "1")
}

// TestRunEngine_DeclaredSuccessExitCodeEmptyPayloadIsClean (CLM-011, CLM-003; control).
func TestRunEngine_DeclaredSuccessExitCodeEmptyPayloadIsClean(t *testing.T) {
	packDir := convertFixturePackDir(t)
	script := writeConvertFixtureScript(t, packDir, "engine.sh", "#!/bin/sh\nexit 1\n")
	binding := secPVBinding(t, packDir, "    command: "+script+"\n    input_mode: none\n    success_exit_codes: [0, 1]")

	res, err := (&DefaultExecutor{}).RunEngine(packDir, binding, []string{"fixture.txt"})
	if err != nil {
		t.Fatalf("a declared no-findings status with an empty payload is clean, got: %v", err)
	}
	if res.Passed {
		t.Errorf("a clean run reports not-fired, got %+v", res)
	}
}

// TestRunEngine_ConvertEngineEmptyRawOutputFailsBeforeConvert (CLM-011, CLM-006).
// "envelope_convert" is the convert-family gap: the convert would wrap the empty
// input in an empty-results SARIF envelope, which parses clean. The convert runs
// SANDBOXED with every file write denied (a marker write would abort the script,
// not record that it ran), so the "ordering" sub-case is what proves the convert
// never ran: its convert FAILS loud if it is ever invoked.
func TestRunEngine_ConvertEngineEmptyRawOutputFailsBeforeConvert(t *testing.T) {
	requireSandboxPlatform(t)

	t.Run("envelope_convert", func(t *testing.T) {
		packDir := convertFixturePackDir(t)
		script := writeConvertFixtureScript(t, packDir, "engine.sh", "#!/bin/sh\nexit 2\n")
		writeConvertFixtureScript(t, packDir, "to-sarif.sh",
			"#!/bin/sh\ncat > /dev/null\nprintf '%s\\n' '"+stdoutArtifactSarif(0)+"'\n")
		binding := secPVBinding(t, packDir, "    command: "+script+"\n    input_mode: none\n    convert: to-sarif.sh")

		res, err := (&DefaultExecutor{}).RunEngine(packDir, binding, []string{"fixture.txt"})
		secPVRequireEmptyPayloadErr(t, res, err, script, "2")
	})

	t.Run("ordering", func(t *testing.T) {
		packDir := convertFixturePackDir(t)
		script := writeConvertFixtureScript(t, packDir, "engine.sh", "#!/bin/sh\nexit 2\n")
		writeConvertFixtureScript(t, packDir, "to-sarif.sh", "#!/bin/sh\ncat > /dev/null\nexit 7\n")
		binding := secPVBinding(t, packDir, "    command: "+script+"\n    input_mode: none\n    convert: to-sarif.sh")

		res, err := (&DefaultExecutor{}).RunEngine(packDir, binding, []string{"fixture.txt"})
		secPVRequireEmptyPayloadErr(t, res, err, script, "2")
		if err != nil && strings.Contains(err.Error(), "convert step") {
			t.Errorf("the empty-payload check must precede the convert, but the convert ran: %v", err)
		}
	})
}

// TestRunEngine_CrashGuardHonorsDeclaredSuccessExitCodes (CLM-011, CLM-007).
func TestRunEngine_CrashGuardHonorsDeclaredSuccessExitCodes(t *testing.T) {
	packDir := convertFixturePackDir(t)
	script := writeConvertFixtureScript(t, packDir, "engine.sh",
		"#!/bin/sh\nprintf '%s\\n' '"+stdoutArtifactSarif(0)+"'\nexit 1\n")
	binding := secPVBinding(t, packDir, "    command: "+script+"\n    input_mode: none\n    crash_guard: true\n    success_exit_codes: [0, 1]")

	res, err := (&DefaultExecutor{}).RunEngine(packDir, binding, []string{"fixture.txt"})
	if err != nil {
		t.Fatalf("a CrashGuard engine exiting with a DECLARED status and zero findings is clean, got: %v", err)
	}
	if res.Passed {
		t.Errorf("zero findings reports not-fired, got %+v", res)
	}
}
