//go:build linux

package packval

import (
	"os"
	"strings"
	"testing"
)

// Error-path coverage for the PARENT-SIDE half of the Linux sandbox.
//
// ⚠ THESE RUN ONLY ON LINUX, AND THEY DO NOT PROVE THE SANDBOX INSTALLS. They
// exercise branches that are reachable without a working Landlock ruleset; the
// mechanism itself is settled by sandbox_linux_exec_test.go on a real host.
//
// The exec-side functions are NOT here and cannot be: they run in a process that
// ends in unix.Exec, so their counters never flush (evidence:
// testdata/sandbox-linux-coverage-profile.txt). They live in sandbox_linux_helper.go
// and are covered by the pack-declared exclusion in .backstop/coverage-exclusions.

// TestMaybeRunSandboxHelper_DispatchesWhenTheEnvVarIsPresent covers the dispatch
// line, which looks exec-erased and is not.
//
// ⚠ THE INVALID JSON IS LOAD-BEARING, NOT LAZINESS. With a VALID request this test
// would apply Landlock and seccomp to the TEST BINARY and then execve — confining
// the suite irrevocably and replacing the process, which is unrecoverable. A
// deliberately malformed spec makes runSandboxHelper fail at the DECODE step, before
// any restriction is installed and before any exec, so the call returns normally and
// its counters flush like any other in-process call.
//
// What it proves: MaybeRunSandboxHelper does not swallow helper mode. A version that
// returned nil when the env var was present would report "not a helper" to a process
// that IS one — the silent pass-through this mechanism exists to prevent.
func TestMaybeRunSandboxHelper_DispatchesWhenTheEnvVarIsPresent(t *testing.T) {
	t.Setenv(sandboxHelperEnvVar, "{ this is not valid json")

	err := MaybeRunSandboxHelper()
	if err == nil {
		t.Fatal("MaybeRunSandboxHelper returned nil while the helper env var was SET; the process would " +
			"fall through into the CLI still in helper mode, or run pack code unsandboxed")
	}
	if !strings.Contains(err.Error(), "decode") {
		t.Errorf("expected the decode failure to surface, got: %v", err)
	}
}

// TestMaybeRunSandboxHelper_ReturnsNilWhenNotAHelper covers the overwhelmingly common
// path — every ordinary backstop invocation takes it.
//
// It is the half of the gate that must stay CHEAP and SILENT: a non-nil return here
// makes cmd/backstop exit 126 before parsing a single argument.
func TestMaybeRunSandboxHelper_ReturnsNilWhenNotAHelper(t *testing.T) {
	if _, present := os.LookupEnv(sandboxHelperEnvVar); present {
		t.Skip("this process IS a sandbox helper; the not-a-helper path is not observable here")
	}
	if err := MaybeRunSandboxHelper(); err != nil {
		t.Fatalf("a non-helper process must return nil, got: %v", err)
	}
}

// TestFilterHelperEnv_StripsTheHelperVariable covers the skip branch.
//
// The stripping is what stops an infinite trampoline: the exec'd command may itself
// be backstop, and inheriting the spec would put the child straight back into helper
// mode. A filter that dropped nothing would loop; one that dropped everything would
// hand the command an empty environment.
func TestFilterHelperEnv_StripsTheHelperVariable(t *testing.T) {
	in := []string{"PATH=/usr/bin", sandboxHelperEnvVar + "={}", "HOME=/root"}

	got := filterHelperEnv(in)

	for _, entry := range got {
		if strings.HasPrefix(entry, sandboxHelperEnvVar+"=") {
			t.Errorf("the helper variable survived the filter (%q); the exec'd command would re-enter "+
				"helper mode and trampoline forever", entry)
		}
	}
	if len(got) != len(in)-1 {
		t.Errorf("filterHelperEnv kept %d of %d entries; it must strip EXACTLY the helper variable, and a "+
			"command handed a stripped environment fails in ways that look nothing like a sandbox bug",
			len(got), len(in))
	}
}

// TestKernelRelease_ReportsSomething covers the success path of the diagnostic used
// in CLM-015's refusal message.
//
// A blank release would strip the kernel version out of the "Landlock unavailable"
// error, which is one of the three tokens that error exists to carry.
func TestKernelRelease_ReportsSomething(t *testing.T) {
	got := kernelRelease()
	if strings.TrimSpace(got) == "" {
		t.Error("kernelRelease returned empty; the Landlock refusal would name no kernel, losing one of " +
			"the three diagnostic tokens CLM-039 requires")
	}
	if got == "unknown" {
		t.Errorf("kernelRelease fell back to %q on a real Linux host — Uname failed, which should not "+
			"happen here and would silently degrade every sandbox diagnostic", got)
	}
}
