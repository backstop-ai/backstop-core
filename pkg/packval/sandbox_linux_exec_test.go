//go:build linux

package packval

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The Linux sandbox's EXECUTION tests. Everything here drives the real
// trampoline: a helper process that installs Landlock and seccomp on itself and
// then execs the command. None of it runs on the development machine.
//
// ─── DO NOT ADD t.Skip TO THIS FILE ─────────────────────────────────────────────
// When Landlock is unavailable these tests must FAIL. That is a deliberate
// posture, not an oversight. A skip is how a Linux sandbox stays unproven for
// another six months: the suite goes green on a host where the mechanism does not
// exist, and nobody learns anything. requireLandlock below turns an unavailable
// mechanism into a loud, attributed failure, and every test calls it.
//
// The darwin tests guard on runtime.GOOS because sandbox-exec is genuinely
// macOS-only — that is a platform N/A. "This kernel has no Landlock" is NOT a
// platform N/A; it is the mechanism being missing on a platform that is supposed
// to have it.
//
// ─── WHY EVERY DENIAL TEST CARRIES A POSITIVE MARKER ────────────────────────────
// A bare non-zero exit cannot distinguish "the sandbox denied correctly" from
// "the sandbox never started". pkg/packval/testdata/ubuntu-runner-probe.txt is the
// recorded instance of exactly that confusion: under bubblewrap, ITEM3_1 (unbound
// read) and ITEM3_2 (denied network) both came back non-zero, which the naive
// reading calls a pass — while ITEM3_0, the POSITIVE control that must succeed
// under a working sandbox, was non-zero too, and so was the ITEM3_3 control leg
// that must exit 0. Read together the triple proved the mechanism was unavailable.
// It proved nothing about denial.
//
// So the probes here print a completion marker from INSIDE the sandboxed process,
// and the denials are paired with controls that must succeed:
// TestLinuxSandbox_ReadsPackDirContents for the filesystem and
// TestLinuxSandbox_NetworkAllowedControlLegSucceeds for the network.

// requireLandlock fails loudly when the mechanism is unavailable, so no test in
// this file can pass vacuously on a kernel without Landlock. It returns the ABI
// the running kernel actually reports.
func requireLandlock(t *testing.T) int {
	t.Helper()
	abi, err := resolveLandlockMechanism(probeLandlockABI)
	if err != nil {
		t.Fatalf("Landlock is unavailable on this host, so the Linux sandbox cannot be proven here: %v\n"+
			"This is a FAILURE and not a skip on purpose — see the header of this file. "+
			"kernel %s, LSMs in /sys/kernel/security/lsm", err, kernelRelease())
	}
	return abi
}

// requireExecutable resolves a helper binary in the PARENT, so a missing
// coreutils is reported as a missing binary rather than surfacing later as an
// ambiguous non-zero exit from inside the sandbox.
func requireExecutable(t *testing.T, name string) string {
	t.Helper()
	resolved, err := exec.LookPath(name)
	if err != nil {
		t.Fatalf("the Linux sandbox tests need %q on PATH and it was not found (%v); "+
			"on a Debian/Ubuntu host it comes from coreutils or bash", name, err)
	}
	return resolved
}

// writeProbeScript drops a probe into packDir. It has to live INSIDE packDir
// because packDir is the only project path the sandbox may read — a script
// anywhere else is unreadable by construction, which would fail every test here
// for the wrong reason.
func writeProbeScript(t *testing.T, packDir, name, body string) string {
	t.Helper()
	path := filepath.Join(packDir, name)
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatalf("write probe script %q: %v", path, err)
	}
	return path
}

// sandboxPackDir makes a symlink-resolved packDir. Resolution is load-bearing,
// not hygiene: a Landlock path_beneath rule matches the KERNEL-resolved path, so
// an unresolved rule silently fails to match and denies legitimate reads.
func sandboxPackDir(t *testing.T) string {
	t.Helper()
	return mustEvalSymlinks(t, t.TempDir())
}

// loopbackPorts binds a TCP listener and a UDP socket on 127.0.0.1 and returns
// their ports.
//
// Loopback rather than a real host is what makes the network legs deterministic:
// the control leg must SUCCEED, and hanging that on a DNS lookup or an outbound
// connection would make CI flakiness indistinguishable from a sandbox that works.
func loopbackPorts(t *testing.T) (tcpPort, udpPort int) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("bind loopback TCP listener: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	go func() {
		for {
			conn, acceptErr := listener.Accept()
			if acceptErr != nil {
				// The listener was closed by t.Cleanup. Leaving the loop is the
				// whole shutdown path; `break` rather than `return` because a bare
				// return inside a closure reads as a naked return of the enclosing
				// function's named results.
				break
			}
			_ = conn.Close()
		}
	}()

	packet, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("bind loopback UDP socket: %v", err)
	}
	t.Cleanup(func() { _ = packet.Close() })

	return listener.Addr().(*net.TCPAddr).Port, packet.LocalAddr().(*net.UDPAddr).Port
}

// networkProbeBody is a bash probe that attempts BOTH a TCP and a UDP socket and
// reports which succeeded, then prints a completion marker.
//
// UDP is not decoration. Landlock's network rights are TCP-ONLY
// (LANDLOCK_ACCESS_NET_BIND_TCP / _CONNECT_TCP is the entire surface), so a
// TCP-only assertion would still pass if the seccomp filter — which is what the
// network denial actually rests on — were missing entirely. The UDP leg is the one
// that can only be explained by seccomp.
//
// Each attempt runs in a SUBSHELL: a failed redirection on `exec` terminates a
// non-interactive shell, so an unparenthesised attempt would abort the probe
// before the second leg and before the marker.
func networkProbeBody(tcpPort, udpPort int) string {
	return fmt.Sprintf(`#!/bin/bash
# Network probe for the Linux sandbox tests. Both attempts are loopback, so a
# success is a real socket and not a routable-internet question.
if (exec 3<>/dev/tcp/127.0.0.1/%d) 2>/dev/null; then echo TCP_OPEN; else echo TCP_BLOCKED; fi
if (exec 4<>/dev/udp/127.0.0.1/%d) 2>/dev/null; then echo UDP_OPEN; else echo UDP_BLOCKED; fi
echo PROBE_COMPLETED
`, tcpPort, udpPort)
}

// runUnderCapability spawns the trampoline with an EXPLICIT capability.
//
// The production entry points (SandboxedRun / SandboxedRunStdout) always build
// ConvertValidatorCapability, which denies the network — correctly, since that is
// the darwin-parity default. The control leg needs the SAME machinery with
// Network: true, and nothing else different, so the only variable between it and
// the denial test is the capability itself.
//
// It reproduces the parent side of newSandboxHelperCommand rather than calling it
// because that function hardcodes the capability, and its file belongs to another
// task. The duplication is four lines and it is confined to the control leg;
// every DENIAL assertion in this file goes through the production path.
func runUnderCapability(t *testing.T, capability SandboxCapability, command string, args []string, packDir string) ([]byte, error) {
	t.Helper()

	encoded, err := json.Marshal(sandboxHelperRequest{
		Capability: capability,
		Command:    command,
		Args:       args,
		Dir:        packDir,
	})
	if err != nil {
		t.Fatalf("encode helper request: %v", err)
	}
	self, err := os.Executable()
	if err != nil {
		t.Fatalf("resolve the test binary for the trampoline: %v", err)
	}

	helper := exec.Command(self)
	helper.Dir = packDir
	helper.Env = append(filterHelperEnv(os.Environ()), sandboxHelperEnvVar+"="+string(encoded))
	out, runErr := helper.CombinedOutput()
	return out, runErr
}

// TestLinuxSandbox_DeniesReadOutsideReadableSet asserts a path outside the
// capability's readable set cannot be read.
//
// The secret lives in a DIFFERENT temp directory from packDir, so it is outside
// the readable set without being outside /tmp — which is the realistic shape of
// the threat: a pack script reading the project it is being run against.
func TestLinuxSandbox_DeniesReadOutsideReadableSet(t *testing.T) {
	requireLandlock(t)
	packDir := sandboxPackDir(t)
	cat := requireExecutable(t, "cat")

	// The marker is named for what it is — a byte string whose APPEARANCE in the
	// output proves a read got through. Calling it a secret would trip the secrets
	// pack's hardcoded-credential rule, which keys on the identifier rather than on
	// the value.
	outside := filepath.Join(mustEvalSymlinks(t, t.TempDir()), "outside-marker.txt")
	const outsideMarker = "UNREADABLE-OUTSIDE-PACKDIR"
	if err := os.WriteFile(outside, []byte(outsideMarker), 0o600); err != nil {
		t.Fatalf("seed the outside-the-readable-set file: %v", err)
	}

	out, err := SandboxedRun(cat, []string{outside}, packDir)
	if err == nil {
		t.Fatalf("reading %q from outside the readable set was ALLOWED (Landlock confines nothing): output %q",
			outside, string(out))
	}
	if strings.Contains(string(out), outsideMarker) {
		t.Fatalf("the file's contents leaked despite the non-zero exit: %q", string(out))
	}
}

// TestLinuxSandbox_ReadsPackDirContents is a POSITIVE CONTROL, and the probe's
// ITEM3_0 is why it is mandatory: that was the bound-path read which MUST succeed
// under a working sandbox, and its failure is what exposed the assertion triple as
// vacuous. Without this test, "everything is denied because the sandbox never ran"
// reads identically to "the sandbox denies the right things".
func TestLinuxSandbox_ReadsPackDirContents(t *testing.T) {
	requireLandlock(t)
	packDir := sandboxPackDir(t)
	cat := requireExecutable(t, "cat")

	const payload = "READABLE-INSIDE-PACKDIR"
	inside := filepath.Join(packDir, "inside.txt")
	if err := os.WriteFile(inside, []byte(payload), 0o600); err != nil {
		t.Fatalf("seed packDir file: %v", err)
	}

	out, err := SandboxedRun(cat, []string{inside}, packDir)
	if err != nil {
		t.Fatalf("reading a file INSIDE packDir failed, so the sandbox denies the pack its own files "+
			"(this is the ISSUE-029 failure mode on Linux): %v\noutput: %q", err, string(out))
	}
	if !strings.Contains(string(out), payload) {
		t.Fatalf("packDir read produced %q, want it to contain %q", string(out), payload)
	}
}

// TestLinuxSandbox_WriteInsidePackDirIsDenied asserts writes are denied even where
// reads are allowed. That combination is what proves the handled-access mask
// covers the WRITE family: a mask carrying only the read rights would leave every
// write right unhandled, and Landlock permits outright whatever the mask does not
// handle.
func TestLinuxSandbox_WriteInsidePackDirIsDenied(t *testing.T) {
	requireLandlock(t)
	packDir := sandboxPackDir(t)
	touch := requireExecutable(t, "touch")

	target := filepath.Join(packDir, "should-not-exist.txt")
	out, err := SandboxedRun(touch, []string{target}, packDir)
	if err == nil {
		t.Fatalf("creating a file inside packDir was ALLOWED; the handled-access mask is missing the "+
			"write family, which Landlock then permits outright: output %q", string(out))
	}
	if _, statErr := os.Stat(target); statErr == nil {
		t.Fatalf("%q exists despite the write denial", target)
	}
}

// TestLinuxSandbox_DeniesNetwork asserts the default capability denies both a TCP
// and a UDP socket.
//
// It asserts on the probe's MARKERS rather than on the exit code, because the
// probe deliberately exits 0 in both directions — an exit-code assertion here
// could be satisfied by the script failing to start at all. PROBE_COMPLETED is
// what rules that out.
func TestLinuxSandbox_DeniesNetwork(t *testing.T) {
	requireLandlock(t)
	packDir := sandboxPackDir(t)
	bash := requireExecutable(t, "bash")
	tcpPort, udpPort := loopbackPorts(t)

	script := writeProbeScript(t, packDir, "network-probe.sh", networkProbeBody(tcpPort, udpPort))
	out, err := SandboxedRun(bash, []string{script}, packDir)
	if err != nil {
		t.Fatalf("the network probe did not run to completion: %v\noutput: %q", err, string(out))
	}
	report := string(out)
	if !strings.Contains(report, "PROBE_COMPLETED") {
		t.Fatalf("the probe never reached its completion marker, so nothing below can be read as a "+
			"denial: %q", report)
	}
	for marker, meaning := range map[string]string{
		"TCP_BLOCKED": "a TCP socket",
		"UDP_BLOCKED": "a UDP socket — this is the leg seccomp alone can explain, since Landlock's " +
			"network rights are TCP-only",
	} {
		if !strings.Contains(report, marker) {
			t.Errorf("the sandbox permitted %s (expected %s in the probe report): %q", meaning, marker, report)
		}
	}
}

// TestLinuxSandbox_NetworkAllowedControlLegSucceeds is THE CONTROL LEG: the same
// probe, the same trampoline, a capability that permits the network — and both
// sockets must open.
//
// Without it, TestLinuxSandbox_DeniesNetwork above proves nothing. TCP_BLOCKED and
// UDP_BLOCKED would also be printed by a probe running on a host with no loopback,
// by a bash built without /dev/tcp support, or by a sandbox so broken that no
// syscall works. This test fails in every one of those cases.
func TestLinuxSandbox_NetworkAllowedControlLegSucceeds(t *testing.T) {
	abi := requireLandlock(t)
	packDir := sandboxPackDir(t)
	bash := requireExecutable(t, "bash")
	tcpPort, udpPort := loopbackPorts(t)

	script := writeProbeScript(t, packDir, "network-probe.sh", networkProbeBody(tcpPort, udpPort))

	permitted := ConvertValidatorCapability(packDir, abi)
	permitted.Network = true

	out, err := runUnderCapability(t, permitted, bash, []string{script}, packDir)
	if err != nil {
		t.Fatalf("the control-leg probe did not run to completion: %v\noutput: %q", err, string(out))
	}
	report := string(out)
	if !strings.Contains(report, "PROBE_COMPLETED") {
		t.Fatalf("the control-leg probe never reached its completion marker: %q", report)
	}
	for marker, meaning := range map[string]string{
		"TCP_OPEN": "TCP",
		"UDP_OPEN": "UDP",
	} {
		if !strings.Contains(report, marker) {
			t.Errorf("%s was blocked under a capability that PERMITS the network (%s absent): %q\n"+
				"Read this as invalidating TestLinuxSandbox_DeniesNetwork rather than as a failure here — "+
				"if the permitted case also fails, the denial it asserts is not attributable to the filter.",
				meaning, marker, report)
		}
	}
}

// TestLinuxSandbox_ConfinementCarriesIntoTheExecdChild is CLM-032, and it is the
// one test that catches the thread-migration hole.
//
// Landlock and seccomp attach to the CALLING THREAD. Go's scheduler migrates
// goroutines across OS threads at will, so a helper that restricts itself on one
// thread and execs from another produces an UNCONFINED CHILD while every syscall
// in the helper returns 0 and the suite stays green. runtime.LockOSThread plus an
// exec from that same thread is what closes it.
//
// The assertion therefore has to come from INSIDE the exec'd process. The script
// attempts the denied operations ITSELF and reports what happened, and CHILD_ALIVE
// proves the report came from a process that actually ran — checking the helper's
// syscall return codes would pass in exactly the broken case this exists to catch.
func TestLinuxSandbox_ConfinementCarriesIntoTheExecdChild(t *testing.T) {
	requireLandlock(t)
	packDir := sandboxPackDir(t)
	bash := requireExecutable(t, "bash")
	tcpPort, udpPort := loopbackPorts(t)

	outside := filepath.Join(mustEvalSymlinks(t, t.TempDir()), "outside-marker.txt")
	if err := os.WriteFile(outside, []byte("UNREADABLE-OUTSIDE-PACKDIR"), 0o600); err != nil {
		t.Fatalf("seed the outside-the-readable-set file: %v", err)
	}

	body := fmt.Sprintf(`#!/bin/bash
# Every line below executes in the EXEC'D CHILD. It reports its own confinement.
if cat %q >/dev/null 2>&1; then echo CHILD_READ_ALLOWED; else echo CHILD_READ_DENIED; fi
if (exec 3<>/dev/tcp/127.0.0.1/%d) 2>/dev/null; then echo CHILD_TCP_ALLOWED; else echo CHILD_TCP_DENIED; fi
if (exec 4<>/dev/udp/127.0.0.1/%d) 2>/dev/null; then echo CHILD_UDP_ALLOWED; else echo CHILD_UDP_DENIED; fi
echo CHILD_ALIVE
`, outside, tcpPort, udpPort)

	script := writeProbeScript(t, packDir, "child-confinement-probe.sh", body)
	out, err := SandboxedRun(bash, []string{script}, packDir)
	if err != nil {
		t.Fatalf("the child probe did not run to completion: %v\noutput: %q", err, string(out))
	}
	report := string(out)

	if !strings.Contains(report, "CHILD_ALIVE") {
		t.Fatalf("the exec'd child never reported, so its confinement is unobserved: %q", report)
	}
	for _, want := range []string{"CHILD_READ_DENIED", "CHILD_TCP_DENIED", "CHILD_UDP_DENIED"} {
		if !strings.Contains(report, want) {
			t.Errorf("the exec'd child was NOT confined (%s absent from %q). The restrictions were "+
				"installed on a thread other than the one that exec'd — LockOSThread and unix.Exec must "+
				"stay on the same thread in applyRestrictionsAndExec.", want, report)
		}
	}
}

// TestLinuxSandbox_ConvertScriptProducesCleanStdout exercises the ACTUAL gate
// path. SandboxedRunStdout captures stdout through an explicit buffer precisely so
// a converter's stderr banner cannot interleave into the SARIF bytes the gate
// parses, and the trampoline has to be transparent to that.
func TestLinuxSandbox_ConvertScriptProducesCleanStdout(t *testing.T) {
	requireLandlock(t)
	packDir := sandboxPackDir(t)
	sh := requireExecutable(t, "sh")

	const banner = "WARNING-THIS-MUST-NOT-REACH-STDOUT"
	const payload = `{"version":"2.1.0"}`
	script := writeProbeScript(t, packDir, "convert-shaped.sh", fmt.Sprintf(`#!/bin/sh
echo %q 1>&2
printf '%%s' %q
`, banner, payload))

	out, err := SandboxedRunStdout(sh, []string{script}, packDir, nil)
	if err != nil {
		t.Fatalf("the convert-shaped script failed under the sandbox: %v\nstdout: %q", err, string(out))
	}
	if got := string(out); got != payload {
		t.Fatalf("stdout was %q, want exactly %q", got, payload)
	}
	if strings.Contains(string(out), banner) {
		t.Fatalf("the stderr banner interleaved into stdout, which would corrupt the SARIF the gate "+
			"parses: %q", string(out))
	}
}

// TestLinuxSandbox_RealInterpreterRunsUnderTheFilter is the Linux analogue of
// darwin's TestSandboxConvertWithRealInterpreter, and it is a MEASUREMENT rather
// than an assumption.
//
// Blocking socket(2) may break an interpreter that opens a socket during startup —
// glibc's NSS/nscd paths and node's platform init are the known candidates, and
// nobody had measured any of them when this was written. jq is expected to be
// fine. If an interpreter a real pack needs cannot start under the filter, that is
// a capability question for BUNDLE-021 OQ-2 and belongs in an issue; it is NOT a
// licence to widen the denied set until the test goes green.
//
// It also re-proves the ISSUE-029 lesson on Linux: jq is dynamically linked, so it
// only starts if the readable set covers the loader and the shared libraries. A
// failure here that mentions a shared object is a READ-RIGHTS finding, not a
// seccomp one.
func TestLinuxSandbox_RealInterpreterRunsUnderTheFilter(t *testing.T) {
	requireLandlock(t)
	packDir := sandboxPackDir(t)
	sh := requireExecutable(t, "sh")

	if locateJQ() == "" {
		t.Fatalf("jq is required for the real-interpreter sandbox test and was not found on PATH; " +
			"install it with `sudo apt-get install -y jq`. This is a hard failure and not a skip: a " +
			"skip-on-missing-interpreter is the vacuous green ISSUE-029 exists to kill.")
	}

	script := filepath.Join(packDir, "convert-jq.sh")
	copyFixtureInto(t, "convert-jq.sh", script)

	out, err := SandboxedRunStdout(sh, []string{script}, packDir, []byte(`{"a":41}`))
	if err != nil {
		t.Fatalf("a real interpreter could not run a real convert script under the sandbox: %v\nstdout: %q\n"+
			"If the error names a shared object, the readable set is missing a system library path "+
			"(linuxSystemReadPaths). If it names a socket, an interpreter opens one at startup and that "+
			"is a BUNDLE-021 OQ-2 capability finding — report it, do not widen the filter.",
			err, string(out))
	}
	if got := strings.TrimSpace(string(out)); got != "42" {
		t.Fatalf("the convert produced %q, want %q", got, "42")
	}
}

// devNullProbeBody is the ISSUE-168 probe: a positive leg for the /dev/null
// redirect, a negative control for an ordinary packDir write, and a completion
// marker.
//
// EACH LEG RUNS IN A SUBSHELL, for the same reason networkProbeBody parenthesises
// its legs: a failed redirection must not be able to abort the probe before the
// remaining legs and the marker run. A probe that dies early produces exactly the
// ambiguous non-zero exit this file's header exists to rule out.
//
// LEG 1 IS THE VERDICT. `echo` cannot fail on its own, so the subshell's exit
// status is precisely "did the redirect open /dev/null for write". LEG 2 runs the
// EXACT reported idiom for shape, and deliberately asserts nothing on its exit
// status — that status reports the missing binary, not the redirect. LEG 3, the
// negative control, deliberately does NOT redirect its own stderr to /dev/null,
// because a control whose plumbing depends on the mechanism under test cannot
// falsify it.
//
// ── A NOTE ON networkProbeBody, WHICH USES `2>/dev/null` ON ITS `if` CONDITIONS ──
// A shell performs a command's redirections BEFORE running it, so when /dev/null
// cannot be opened for write the subshell never attempts the socket at all: it
// exits non-zero and the ELSE branch prints ..._BLOCKED. That means
// TestLinuxSandbox_NetworkAllowedControlLegSucceeds' TCP_BLOCKED/UDP_BLOCKED
// markers may have been an ARTEFACT of this very defect rather than evidence of a
// network-permission bug. This lane changes nothing about that test; the reading is
// adjudicated against a real CI run.
func devNullProbeBody(packDirTarget string) string {
	return fmt.Sprintf(`#!/bin/bash
# ISSUE-168 probe. /dev/null is a write-only sink, so a write there is safe; an
# ordinary file inside packDir must still be refused.
if ( echo probe >/dev/null 2>&1 ); then echo DEVNULL_WRITE_OK; else echo DEVNULL_WRITE_DENIED; fi
command -v definitely-not-a-real-binary >/dev/null 2>&1 || true
echo DEVNULL_IDIOM_RAN
if ( : > %q ); then echo PACKDIR_WRITE_LEAKED; else echo PACKDIR_WRITE_BLOCKED; fi
echo DEVNULL_PROBE_COMPLETED
`, packDirTarget)
}

// TestLinuxSandbox_DevNullWriteIsPermittedAndOtherWritesAreNot is the direct Linux
// assertion for ISSUE-168: the universal `>/dev/null` idiom works inside the
// sandbox, and nothing else became writable.
//
// ★ ITS PRE-FIX STATE IS AN ASSERTION ABOUT CI, NOT AN OBSERVATION. Nothing in this
// file compiles on the darwin development machine — sandbox_linux.go and
// sandbox_linux_helper.go are //go:build linux, no Landlock ruleset is ever created
// there, and no trampoline re-exec happens. A Docker container does not lift that
// ceiling either: the Docker-Desktop LinuxKit kernel reports Landlock ABI 0 and is
// REFUSED by resolveLandlockMechanism, by design. So the expectation recorded here —
// that DEVNULL_WRITE_OK was ABSENT before the fix, because Landlock enforced the
// profile's stated "no writes" literally and the redirect was refused with
// `cannot create /dev/null: Permission denied`, exit 127 — is derived from the
// reported CI evidence, not from a local run.
//
// THE TWO LEGS ARE ONE TEST ON PURPOSE. A green /dev/null leg beside a red packDir
// leg would mean the carve-out WIDENED the write surface rather than scoping it, and
// splitting them across two tests makes that combination easy to miss in a CI log.
func TestLinuxSandbox_DevNullWriteIsPermittedAndOtherWritesAreNot(t *testing.T) {
	requireLandlock(t)
	packDir := sandboxPackDir(t)
	bash := requireExecutable(t, "bash")

	target := filepath.Join(packDir, "should-not-exist.txt")
	script := writeProbeScript(t, packDir, "devnull-probe.sh", devNullProbeBody(target))

	out, err := SandboxedRun(bash, []string{script}, packDir)
	if err != nil {
		t.Fatalf("the ISSUE-168 probe did not run to completion: %v\nreport:\n%s", err, string(out))
	}
	report := string(out)

	if !strings.Contains(report, "DEVNULL_PROBE_COMPLETED") {
		t.Fatalf("the probe never ran to its completion marker, so nothing below is readable as a "+
			"permission or a denial — the sandbox may have failed to start at all (ISSUE-168 probe).\n"+
			"expected marker %q\nreport:\n%s", "DEVNULL_PROBE_COMPLETED", report)
	}

	if !strings.Contains(report, "DEVNULL_WRITE_OK") {
		t.Errorf("ISSUE-168 IS UNFIXED ON THIS KERNEL: `echo probe >/dev/null 2>&1` was REFUSED inside "+
			"the sandbox, so the expected marker %q is absent (the probe printed DEVNULL_WRITE_DENIED "+
			"instead). `command -v foo >/dev/null 2>&1` is a universal shell idiom and a pack-supplied "+
			"convert or validator script must be able to use it; when the redirect is refused the shell "+
			"reports `cannot create /dev/null: Permission denied` and exits 127, which reads as a broken "+
			"converter rather than as a sandbox decision. Check that DeriveSandboxRestrictions still "+
			"emits the /dev/null rule and that landlock_add_rule accepted it.\nreport:\n%s",
			"DEVNULL_WRITE_OK", report)
	}

	// The anti-widening control: the carve-out must be ONE inode, not a relaxed
	// write policy.
	if strings.Contains(report, "PACKDIR_WRITE_LEAKED") {
		t.Errorf("a file was created INSIDE packDir; the ISSUE-168 carve-out widened the write surface "+
			"instead of scoping it to the /dev/null sink. Expected %q, got %q in the report.\nreport:\n%s",
			"PACKDIR_WRITE_BLOCKED", "PACKDIR_WRITE_LEAKED", report)
	}
	if !strings.Contains(report, "PACKDIR_WRITE_BLOCKED") {
		t.Errorf("the negative control marker %q is absent, so the packDir write leg cannot be read as "+
			"a denial — the probe may have been cut short between the /dev/null leg and the completion "+
			"marker.\nreport:\n%s", "PACKDIR_WRITE_BLOCKED", report)
	}
	if _, statErr := os.Stat(target); statErr == nil {
		t.Errorf("%q EXISTS on disk despite the write denial; the marker-based reading above disagrees "+
			"with the filesystem, and the filesystem is authoritative.\nreport:\n%s", target, report)
	}
}
