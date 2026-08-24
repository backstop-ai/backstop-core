//go:build linux

package packval

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/unix"
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

func TestRunSandboxHelperWith_ReturningApplyPathsAreLoud(t *testing.T) {
	encoded, err := json.Marshal(sandboxHelperRequest{Command: "child"})
	if err != nil {
		t.Fatal(err)
	}
	applyFailure := errors.New("apply sentinel")
	err = runSandboxHelperWith(string(encoded), func(request sandboxHelperRequest) error {
		if request.Command != "child" {
			t.Fatalf("decoded command=%q, want child", request.Command)
		}
		return applyFailure
	})
	if !errors.Is(err, applyFailure) || !strings.Contains(err.Error(), "install the sandbox restrictions") {
		t.Fatalf("apply error=%v, want wrapped sentinel", err)
	}
	err = runSandboxHelperWith(string(encoded), func(sandboxHelperRequest) error { return nil })
	if err == nil || !strings.Contains(err.Error(), "returned without exec'ing") {
		t.Fatalf("returning apply path error=%v, want fail-closed no-exec diagnostic", err)
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

func TestKernelAndLandlockProbe_InjectedSyscallOutcomes(t *testing.T) {
	if got := kernelReleaseWith(func(*unix.Utsname) error { return unix.EIO }); got != "unknown" {
		t.Fatalf("failed uname release=%q, want unknown", got)
	}
	successSyscall := func(trap, arg1, arg2, arg3 uintptr) (uintptr, uintptr, unix.Errno) {
		if trap != unix.SYS_LANDLOCK_CREATE_RULESET || arg1 != 0 || arg2 != 0 || arg3 != uintptr(unix.LANDLOCK_CREATE_RULESET_VERSION) {
			t.Fatalf("Landlock syscall args=%d/%d/%d/%d", trap, arg1, arg2, arg3)
		}
		return 7, 0, 0
	}
	abi, release, err := probeLandlockABIWith(func() string { return "test-kernel" }, successSyscall)
	if err != nil || abi != 7 || release != "test-kernel" {
		t.Fatalf("successful probe=%d/%q/%v, want 7/test-kernel/nil", abi, release, err)
	}
	failureSyscall := func(uintptr, uintptr, uintptr, uintptr) (uintptr, uintptr, unix.Errno) {
		return 0, 0, unix.ENOSYS
	}
	abi, release, err = probeLandlockABIWith(func() string { return "missing-kernel" }, failureSyscall)
	if abi != 0 || release != "missing-kernel" || !errors.Is(err, unix.ENOSYS) {
		t.Fatalf("failed probe=%d/%q/%v, want 0/missing-kernel/ENOSYS", abi, release, err)
	}
}

// TestNewSandboxHelperInvocation_RefusesWhenTheMechanismIsUnavailable drives the
// refusal branch through the injected prober.
//
// This is the statement that gates two more: platformSandboxedRun and
// platformSandboxedRunStdout both wrap this failure, and neither wrap can execute
// unless this one does. On a healthy runner the probe always succeeds, so without
// the seam all three are permanently unreachable.
//
// The behaviour it locks is CLM-015's: an unavailable mechanism REFUSES rather than
// falling through to an unsandboxed exec. A version that logged and continued would
// run pack-supplied code with no confinement — the defect ISSUE-020 exists to fix.
func TestNewSandboxHelperInvocation_RefusesWhenTheMechanismIsUnavailable(t *testing.T) {
	unavailable := func() (int, string, error) {
		return 0, "6.17.0-test", unix.ENOSYS
	}

	invocation, err := newSandboxHelperInvocation("/bin/echo", []string{"hi"}, t.TempDir(), unavailable)

	if err == nil {
		t.Fatalf("an unavailable Landlock mechanism must REFUSE; got a runnable invocation (%v) and no error. "+
			"Falling through here runs pack-supplied code unsandboxed", invocation)
	}
	if invocation != nil {
		t.Errorf("a refusal must return no invocation, got %v — a caller that ignored the error would exec it", invocation)
	}
	if !strings.Contains(err.Error(), "negotiate the Landlock mechanism") {
		t.Errorf("the refusal must name what failed; got: %v", err)
	}
}

func TestNewSandboxHelperInvocation_AcceptedProbeBuildsClosableControlPipe(t *testing.T) {
	probe := func() (int, string, error) { return 6, "test-kernel", nil }
	invocation, err := newSandboxHelperInvocation("/bin/true", nil, t.TempDir(), probe)
	if err != nil {
		t.Fatal(err)
	}
	if err := invocation.ackRead.Close(); err != nil {
		t.Errorf("close acknowledgement reader: %v", err)
	}
	if err := invocation.command.ExtraFiles[0].Close(); err != nil {
		t.Errorf("close acknowledgement writer: %v", err)
	}
}

func TestLinuxSandboxedProductionEntryPoints_FailClosedForMissingTarget(t *testing.T) {
	packDir := t.TempDir()
	const missing = "/definitely/missing/issue185-target"
	if output, err := platformSandboxedRun(missing, nil, packDir); err == nil {
		t.Fatalf("combined-output production path ran missing target: %q", output)
	}
	if output, err := platformSandboxedRunStdout(missing, nil, packDir, []byte("input")); err == nil {
		t.Fatalf("stdout production path ran missing target: %q", output)
	}
	result, err := platformSandboxedExecute(missing, nil, packDir, nil, false)
	if err == nil {
		t.Fatalf("typed production path ran missing target: %#v", result)
	}
}

func TestNewSandboxHelperInvocationForCapability_WiresAcknowledgementPipe(t *testing.T) {
	invocation, err := newSandboxHelperInvocationForCapability("/bin/echo", []string{"ok"}, t.TempDir(), SandboxCapability{Network: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(invocation.command.ExtraFiles) != 1 {
		_ = invocation.ackRead.Close()
		t.Fatalf("helper inherited %d files, want one acknowledgement writer", len(invocation.command.ExtraFiles))
	}
	defer func() {
		if err := invocation.ackRead.Close(); err != nil {
			t.Errorf("close acknowledgement reader: %v", err)
		}
		if err := invocation.command.ExtraFiles[0].Close(); err != nil {
			t.Errorf("close acknowledgement writer: %v", err)
		}
	}()
	prefix := sandboxHelperEnvVar + "="
	var encoded string
	for _, entry := range invocation.command.Env {
		if strings.HasPrefix(entry, prefix) {
			encoded = strings.TrimPrefix(entry, prefix)
			break
		}
	}
	if encoded == "" {
		t.Fatal("helper environment carries no encoded request")
	}
	var request sandboxHelperRequest
	if err := json.Unmarshal([]byte(encoded), &request); err != nil {
		t.Fatalf("decode helper request: %v", err)
	}
	if request.AckFD != sandboxAckFD || !request.Capability.Network {
		t.Fatalf("request AckFD/network=%d/%v, want %d/true", request.AckFD, request.Capability.Network, sandboxAckFD)
	}
}

func TestNewSandboxHelperInvocationForCapability_DependencyFailuresAreWrapped(t *testing.T) {
	marshalFailure := errors.New("marshal sentinel")
	executableFailure := errors.New("executable sentinel")
	pipeFailure := errors.New("pipe sentinel")
	validMarshal := func(value any) ([]byte, error) { return json.Marshal(value) }
	validExecutable := func() (string, error) { return "/proc/self/exe", nil }
	validPipe := func() (*os.File, *os.File, error) { return os.Pipe() }

	tests := []struct {
		name         string
		dependencies sandboxHelperInvocationDependencies
		want         error
		context      string
	}{
		{name: "marshal", dependencies: sandboxHelperInvocationDependencies{
			marshal: func(any) ([]byte, error) { return nil, marshalFailure }, executable: validExecutable, pipe: validPipe,
		}, want: marshalFailure, context: "encode sandbox helper request"},
		{name: "executable", dependencies: sandboxHelperInvocationDependencies{
			marshal: validMarshal, executable: func() (string, error) { return "", executableFailure }, pipe: validPipe,
		}, want: executableFailure, context: "resolve this executable"},
		{name: "pipe", dependencies: sandboxHelperInvocationDependencies{
			marshal: validMarshal, executable: validExecutable, pipe: func() (*os.File, *os.File, error) { return nil, nil, pipeFailure },
		}, want: pipeFailure, context: "create sandbox acknowledgement pipe"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			invocation, err := newSandboxHelperInvocationForCapabilityWithDependencies("child", nil, t.TempDir(), SandboxCapability{}, test.dependencies)
			if invocation != nil || !errors.Is(err, test.want) || !strings.Contains(err.Error(), test.context) {
				t.Fatalf("invocation=%v error=%v, want nil wrapped %v with %q", invocation, err, test.want, test.context)
			}
		})
	}
}

func testLinuxSandboxInvocation(t *testing.T, script string) *sandboxHelperInvocation {
	t.Helper()
	ackRead, ackWrite, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	command := exec.Command("/bin/sh", "-c", script)
	command.ExtraFiles = []*os.File{ackWrite}
	return &sandboxHelperInvocation{command: command, ackRead: ackRead}
}

func runTestLinuxSandboxedStdoutInvocation(invocation *sandboxHelperInvocation, stdin []byte) ([]byte, error) {
	if stdin != nil {
		invocation.command.Stdin = bytes.NewReader(stdin)
	}
	var stdout, stderr bytes.Buffer
	invocation.command.Stdout = &stdout
	invocation.command.Stderr = &stderr
	return runLinuxSandboxedStdoutInvocation(invocation, &stdout, &stderr)
}

func TestRunLinuxSandboxedStdoutInvocation_ReturningBranches(t *testing.T) {
	t.Run("acknowledged success preserves stdin and stdout", func(t *testing.T) {
		invocation := testLinuxSandboxInvocation(t, `printf '\245' >&3; exec cat`)
		out, err := runTestLinuxSandboxedStdoutInvocation(invocation, []byte("payload"))
		if err != nil || string(out) != "payload" {
			t.Fatalf("output=%q error=%v, want payload/nil", out, err)
		}
	})
	t.Run("child failure preserves stdout and stderr diagnostic", func(t *testing.T) {
		invocation := testLinuxSandboxInvocation(t, `printf '\245' >&3; printf partial; printf diagnostic >&2; exit 7`)
		out, err := runTestLinuxSandboxedStdoutInvocation(invocation, nil)
		if string(out) != "partial" || err == nil || !strings.Contains(err.Error(), "diagnostic") {
			t.Fatalf("output=%q error=%v, want partial and diagnostic failure", out, err)
		}
	})
	t.Run("start failure closes the control pipe", func(t *testing.T) {
		ackRead, ackWrite, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		command := exec.Command(filepath.Join(t.TempDir(), "missing-command"))
		command.ExtraFiles = []*os.File{ackWrite}
		_, err = runTestLinuxSandboxedStdoutInvocation(&sandboxHelperInvocation{command: command, ackRead: ackRead}, nil)
		if err == nil || !strings.Contains(err.Error(), "sandboxed run (stdout) failed") {
			t.Fatalf("start error=%v", err)
		}
	})
	t.Run("acknowledgement read failure still waits for child", func(t *testing.T) {
		ackRead, err := os.Open(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		ackWrite, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
		if err != nil {
			t.Fatal(err)
		}
		command := exec.Command("/bin/sh", "-c", "exit 0")
		command.ExtraFiles = []*os.File{ackWrite}
		_, err = runTestLinuxSandboxedStdoutInvocation(&sandboxHelperInvocation{command: command, ackRead: ackRead}, nil)
		if err == nil || !strings.Contains(err.Error(), "read sandbox acknowledgement") || command.ProcessState == nil {
			t.Fatalf("read error=%v processState=%v, want read failure after Wait", err, command.ProcessState)
		}
	})
}

func TestRunLinuxSandboxedInvocation_EvidenceAndProcessBranches(t *testing.T) {
	t.Run("combined output acknowledged success", func(t *testing.T) {
		invocation := testLinuxSandboxInvocation(t, `printf '\245' >&3; printf stdout; printf stderr >&2`)
		result, err := runLinuxSandboxedInvocation(invocation, nil, false)
		if err != nil || string(result.Output) != "stdoutstderr" || !result.NativeSandboxApplied {
			t.Fatalf("result=%#v error=%v", result, err)
		}
	})
	t.Run("stdout-only child failure is folded", func(t *testing.T) {
		invocation := testLinuxSandboxInvocation(t, `printf '\245' >&3; printf stdout; printf stderr >&2; exit 9`)
		result, err := runLinuxSandboxedInvocation(invocation, []byte("ignored"), true)
		if string(result.Output) != "stdout" || err == nil || !strings.Contains(err.Error(), "stderr") || !result.NativeSandboxApplied {
			t.Fatalf("result=%#v error=%v", result, err)
		}
	})
	t.Run("combined child failure retains evidence", func(t *testing.T) {
		invocation := testLinuxSandboxInvocation(t, `printf '\245' >&3; printf failed; exit 11`)
		result, err := runLinuxSandboxedInvocation(invocation, nil, false)
		if string(result.Output) != "failed" || err == nil || !result.NativeSandboxApplied {
			t.Fatalf("result=%#v error=%v", result, err)
		}
	})
	t.Run("missing acknowledgement is not native evidence", func(t *testing.T) {
		invocation := testLinuxSandboxInvocation(t, `printf ran`)
		result, err := runLinuxSandboxedInvocation(invocation, nil, false)
		if err != nil || string(result.Output) != "ran" || result.NativeSandboxApplied {
			t.Fatalf("result=%#v error=%v", result, err)
		}
	})
	t.Run("start failure returns no evidence", func(t *testing.T) {
		ackRead, ackWrite, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		command := exec.Command(filepath.Join(t.TempDir(), "missing-command"))
		command.ExtraFiles = []*os.File{ackWrite}
		result, err := runLinuxSandboxedInvocation(&sandboxHelperInvocation{command: command, ackRead: ackRead}, nil, false)
		if err == nil || result.NativeSandboxApplied || !strings.Contains(err.Error(), "sandboxed run failed") {
			t.Fatalf("result=%#v error=%v", result, err)
		}
	})
}

// TestLinuxSandboxedRunWith_WrapsThePrepareFailure drives platformSandboxedRun's
// refusal wrap, which is unreachable through the production entry point.
//
// The wrap only fires when newSandboxHelperInvocation fails from INSIDE this function,
// and the production call site passes the real prober — as
// TestSandboxLinux_ProductionPathUsesTheRealABIProbe enforces, from the untagged
// sandbox_wiring_guard_test.go where it now lives so it runs on every platform — so
// on a healthy runner it never executes. Injecting the failure here is
// the only honest way to reach it, and what it locks is that the refusal is WRAPPED
// with context rather than returned bare: a caller seeing "prepare the linux sandbox"
// knows the sandbox never came up, not that the command itself failed.
func TestLinuxSandboxedRunWith_WrapsThePrepareFailure(t *testing.T) {
	probeCalls := 0
	unavailable := func() (int, string, error) {
		probeCalls++
		return 0, "6.17.0-test", unix.ENOSYS
	}

	out, err := linuxSandboxedRunWith("/bin/echo", []string{"hi"}, t.TempDir(), unavailable)

	if err == nil {
		t.Fatalf("an unavailable mechanism must refuse; got output %q and no error", string(out))
	}
	if out != nil {
		t.Errorf("a refusal must return no output, got %q — nothing ran", string(out))
	}
	if probeCalls != 1 || !strings.Contains(err.Error(), "prepare the linux sandbox") || !strings.Contains(err.Error(), unix.ENOSYS.Error()) {
		t.Errorf("the failure must be wrapped so the caller can tell setup from execution; got: %v", err)
	}
}

// TestLinuxSandboxedRunStdoutWith_WrapsThePrepareFailure is the stdout arm's twin.
//
// It matters independently: this arm returns "stdout captured so far" on a normal
// command failure, so a setup failure must be distinguishable from a converter that
// ran and produced partial output. Returning nil bytes here is what keeps the gate
// from parsing a sandbox-setup failure as empty SARIF.
func TestLinuxSandboxedRunStdoutWith_WrapsThePrepareFailure(t *testing.T) {
	probeCalls := 0
	unavailable := func() (int, string, error) {
		probeCalls++
		return 0, "6.17.0-test", unix.ENOSYS
	}

	out, err := linuxSandboxedRunStdoutWith("/bin/echo", nil, t.TempDir(), nil, unavailable)

	if err == nil {
		t.Fatalf("an unavailable mechanism must refuse; got output %q and no error", string(out))
	}
	if out != nil {
		t.Errorf("a setup refusal must return NO bytes, got %q; the gate would parse them as the "+
			"converter's SARIF output", string(out))
	}
	if probeCalls != 1 || !strings.Contains(err.Error(), "prepare the linux sandbox") || !strings.Contains(err.Error(), unix.ENOSYS.Error()) {
		t.Errorf("expected the prepare-failure wrap, got: %v", err)
	}
}

func TestLinuxSandboxedExecuteWith_UnavailableLandlockReturnsExactZeroEvidence(t *testing.T) {
	probeCalls := 0
	unavailable := func() (int, string, error) {
		probeCalls++
		return 0, "6.17.0-test", unix.ENOSYS
	}
	result, err := linuxSandboxedExecuteWith("/bin/true", nil, t.TempDir(), []byte("must-not-run"), false, unavailable)
	if probeCalls != 1 || !strings.Contains(err.Error(), unix.ENOSYS.Error()) {
		t.Fatalf("probe calls/error=%d/%v, want one call and ENOSYS diagnostic", probeCalls, err)
	}
	if len(result.Output) != 0 || result.NativeSandboxApplied {
		t.Fatalf("unavailable setup returned evidence %#v, want exact zero result", result)
	}
	for _, token := range []string{"prepare the linux sandbox", "6.17.0-test", "refuses to run pack-supplied code unsandboxed"} {
		if !strings.Contains(err.Error(), token) {
			t.Errorf("unavailable error %q lacks %q", err, token)
		}
	}
}

func TestWriteSandboxAcknowledgement_WritesOneByteAndCloses(t *testing.T) {
	readEnd, writeEnd, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := readEnd.Close(); err != nil {
			t.Errorf("close acknowledgement reader: %v", err)
		}
	}()
	if err := writeSandboxAcknowledgement(int(writeEnd.Fd())); err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(readEnd)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != sandboxAckByte {
		t.Fatalf("acknowledgement=%x, want exactly %x", got, sandboxAckByte)
	}
	if _, err := writeEnd.Write([]byte("still-open")); err == nil {
		t.Fatal("acknowledgement writer remained open")
	}
	if err := writeSandboxAcknowledgement(-1); err == nil {
		t.Fatal("invalid acknowledgement descriptor returned success")
	}
	readOnly, peer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := writeSandboxAcknowledgement(int(readOnly.Fd())); err == nil || !strings.Contains(err.Error(), "write sandbox acknowledgement") {
		t.Fatalf("read-only acknowledgement descriptor error=%v, want wrapped write failure", err)
	}
	if err := peer.Close(); err != nil {
		t.Errorf("close acknowledgement peer: %v", err)
	}
}

func TestSeccompPolicyHelpers_ResolveConcretePolicyBeforeInstall(t *testing.T) {
	for _, test := range []struct {
		goarch  string
		want    uint32
		wantErr bool
	}{
		{goarch: "amd64", want: uint32(unix.AUDIT_ARCH_X86_64)},
		{goarch: "arm64", want: uint32(unix.AUDIT_ARCH_AARCH64)},
		{goarch: "unsupported", wantErr: true},
	} {
		got, err := seccompAuditArch(test.goarch)
		if test.wantErr != (err != nil) || got != test.want {
			t.Errorf("seccompAuditArch(%q)=%d,%v; want %d,error=%v", test.goarch, got, err, test.want, test.wantErr)
		}
	}

	known := []string{"socket", "socketpair", "connect", "bind", "sendto", "sendmsg", "sendmmsg", "recvfrom", "recvmsg", "recvmmsg", "io_uring_setup", "io_uring_enter"}
	for _, name := range known {
		numbers, err := seccompSyscallNumbers([]string{name})
		if err != nil || len(numbers) != 1 || numbers[0] == 0 {
			t.Errorf("seccompSyscallNumbers(%q)=%v,%v", name, numbers, err)
		}
	}
	if _, err := seccompSyscallNumbers([]string{"not-a-real-syscall"}); err == nil {
		t.Fatal("unknown denied syscall was silently omitted")
	}

	installerCalls := 0
	installerErr := errors.New("installer sentinel")
	installer := func(got SandboxRestrictionSpec) error {
		installerCalls++
		return installerErr
	}
	if err := applySeccompPolicy(SandboxRestrictionSpec{}, installer); err != nil || installerCalls != 0 {
		t.Fatalf("empty policy error/calls=%v/%d, want nil/0", err, installerCalls)
	}
	policy := DeriveSandboxRestrictions(ConvertValidatorCapability(t.TempDir(), 6))
	if err := applySeccompPolicy(policy, installer); !errors.Is(err, installerErr) || installerCalls != 1 {
		t.Fatalf("non-empty policy error/calls=%v/%d, want sentinel/1", err, installerCalls)
	}
}
