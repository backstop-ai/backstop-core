package packval

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestPackSandbox_NativeUnavailableFailsClosedWithoutFallback(t *testing.T) {
	nativeCalls, externalCalls := 0, 0
	unavailable := func(string, []string, string, []byte, bool) (SandboxRunResult, error) {
		nativeCalls++
		return SandboxRunResult{}, errors.New("native confinement unavailable")
	}
	fallback := func(string, []string, string, []byte, bool) (SandboxRunResult, error) {
		externalCalls++
		return SandboxRunResult{Output: []byte("unsandboxed")}, nil
	}
	runner, err := newSandboxRunnerWithExecution(SandboxModeNative, unavailable, fallback)
	if err != nil {
		t.Fatal(err)
	}
	result, runErr := runner.Run("pack-child", nil, t.TempDir())
	if runErr == nil || !strings.Contains(runErr.Error(), "native confinement unavailable") {
		t.Fatalf("native unavailability did not fail closed: result=%#v err=%v", result, runErr)
	}
	if nativeCalls != 1 || externalCalls != 0 {
		t.Fatalf("native/external calls = %d/%d, want 1/0; native must never fall back", nativeCalls, externalCalls)
	}
}

func TestExternalSandbox_UsesExistingConvertAndValidatorPathsWithoutNativeLayer(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture is POSIX-only")
	}
	t.Setenv(PackSandboxEnvVar, "external")
	packDir := t.TempDir()
	runner, err := NewSandboxRunner(SandboxModeExternal)
	if err != nil {
		t.Fatal(err)
	}

	combined, err := runner.Run("/bin/sh", []string{"-c", `printf '%s|%s' "$PWD" "${BACKSTOP_PACK_SANDBOX-unset}"; printf '|stderr' >&2`}, packDir)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	resolved, _ := filepath.EvalSymlinks(packDir)
	if got, want := string(combined.Output), resolved+"|unset|stderr"; got != want {
		t.Fatalf("combined output = %q, want %q", got, want)
	}
	if combined.NativeSandboxApplied {
		t.Fatal("external combined run reported native application")
	}

	stdout, err := runner.RunStdout("/bin/sh", []string{"-c", `read value; printf '%s' "$value"; printf ignored >&2`}, packDir, []byte("payload\n"))
	if err != nil {
		t.Fatalf("RunStdout: %v", err)
	}
	if string(stdout.Output) != "payload" || stdout.NativeSandboxApplied {
		t.Fatalf("stdout result = %#v", stdout)
	}

	partial, err := runner.RunStdout("/bin/sh", []string{"-c", `printf partial; exit 7`}, packDir, nil)
	if err == nil || !strings.Contains(string(partial.Output), "partial") {
		t.Fatalf("non-zero result = %#v, err=%v", partial, err)
	}
	if got := os.Getenv(PackSandboxEnvVar); got != "external" {
		t.Fatalf("parent environment mutated: %q", got)
	}
}

func TestExternalSandbox_DoesNotRedefineSandboxedSurfaces(t *testing.T) {
	source, err := os.ReadFile("sandbox.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"producer", "recipe", "engine"} {
		if strings.Contains(strings.ToLower(string(source)), "sandbox"+forbidden) {
			t.Fatalf("sandbox boundary added %s surface", forbidden)
		}
	}
}
