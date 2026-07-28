package packval

import (
	"bytes"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"runtime"
	"strings"
	"testing"
	"unicode/utf8"
)

// THESE TESTS RUN ON DARWIN, AND THAT IS THE POINT.
//
// The defect they lock was invisible to every existing test because the only
// thing exercising the stdout arm was a Linux runner: the helper wrote a correct
// CLM-015 diagnostic naming Landlock, the kernel and ISSUE-020, and the parent
// threw it away by never setting helper.Stderr. Run
// https://github.com/backstop-ai/backstop-core/actions/runs/30381252600 failed
// with nothing but "exit status 126" to show for it.
//
// A regression lock that also needed a Linux runner would restore exactly that
// blind spot. So the fold is a PURE FUNCTION over (stdout, stderr, run error) and
// these drive it directly — the same discipline as the capability-derivation
// tests.

// helperDiagnostic is a realistic sandbox-helper diagnostic: the shape
// probeLandlockABI produces, wrapped the way cmd/backstop's writeDiagnostic emits
// it before the helper exits 126.
const helperDiagnostic = "Error: kernel 6.17.0-1020-azure reports Landlock ABI 0, so no filesystem " +
	"confinement is available; backstop refuses to run pack-supplied code unsandboxed — see ISSUE-020"

const helperKernelRelease = "6.17.0-1020-azure"

// TestFoldHelperStderrIntoError_CarriesDiagnosticTokens is the claim itself
// (CLM-039): a failed helper's reason must reach the caller.
//
// It asserts TOKENS rather than the exact sentence so the diagnostic can be
// reworded without a test edit — the contract is "the reader can tell it was
// Landlock, on which kernel, and where to read about it", not any particular
// phrasing. "andlock" is a substring on purpose: it matches Landlock and landlock
// alike, matching the convention the loud-error test already uses.
func TestFoldHelperStderrIntoError_CarriesDiagnosticTokens(t *testing.T) {
	stdout := []byte(`{"runs":[]}`)
	runErr := errors.New("exit status 126")

	_, err := foldHelperStderrIntoError(stdout, []byte(helperDiagnostic), runErr)
	if err == nil {
		t.Fatal("a non-zero helper exit must return an error")
	}

	for _, token := range []string{"andlock", helperKernelRelease, "ISSUE-020"} {
		if !strings.Contains(err.Error(), token) {
			t.Errorf("the folded error is missing token %q; without it a runner failure is undiagnosable, "+
				"which is the exact defect this fold exists to fix. got: %v", token, err)
		}
	}

	// The underlying run error must survive too — the exit code is how a reader
	// tells "the sandbox refused" (126) from "the converter failed" (anything else).
	if !strings.Contains(err.Error(), "exit status 126") {
		t.Errorf("the folded error dropped the underlying run error; got: %v", err)
	}
	if !errors.Is(err, runErr) {
		t.Errorf("the run error must stay unwrappable via errors.Is so callers can classify it; got: %v", err)
	}
}

// TestFoldHelperStderrIntoError_ReturnedStdoutStaysByteClean is the guarantee
// that makes the fold safe to ship.
//
// This arm exists at all because a converter's stderr banner must never reach the
// SARIF the gate parses. Folding stderr into the ERROR must not undo that: the
// returned bytes have to be the stdout bytes EXACTLY — nothing appended,
// prepended, or interleaved.
func TestFoldHelperStderrIntoError_ReturnedStdoutStaysByteClean(t *testing.T) {
	sarif := []byte(`{"version":"2.1.0","runs":[{"results":[]}]}`)
	banner := []byte("WARNING: jq: falling back to slow path\n" + helperDiagnostic)

	out, err := foldHelperStderrIntoError(sarif, banner, errors.New("exit status 1"))
	if err == nil {
		t.Fatal("expected an error on a non-zero exit")
	}

	if !bytes.Equal(out, sarif) {
		t.Fatalf("the returned stdout is no longer byte-identical to what the command wrote.\n got: %q\nwant: %q",
			out, sarif)
	}
	if bytes.Contains(out, []byte("WARNING")) || bytes.Contains(out, []byte("ISSUE-020")) {
		t.Fatalf("stderr bytes leaked into the returned stdout; the gate would fail to parse this as SARIF. got: %q", out)
	}
}

// TestFoldHelperStderrIntoError_SuccessPathIgnoresStderr guards the common case.
//
// Plenty of healthy converters write a banner or a deprecation notice to stderr
// and exit 0. Turning a non-empty stderr into a failure would break every one of
// them, so the run error — not the presence of stderr — is what decides.
func TestFoldHelperStderrIntoError_SuccessPathIgnoresStderr(t *testing.T) {
	sarif := []byte(`{"runs":[]}`)
	out, err := foldHelperStderrIntoError(sarif, []byte("WARNING: harmless banner\n"), nil)

	if err != nil {
		t.Fatalf("a successful run with a noisy stderr must not become an error; got: %v", err)
	}
	if !bytes.Equal(out, sarif) {
		t.Fatalf("stdout was altered on the success path: got %q want %q", out, sarif)
	}
}

// TestFoldHelperStderrIntoError_EmptyStderrStillErrors keeps absence of a
// diagnostic from becoming absence of a failure.
//
// A helper can die before writing anything — killed by the kernel, or exec'ing
// something that never starts. That is still a failed sandboxed run, and it must
// still be an error that names the exit rather than a silent success.
func TestFoldHelperStderrIntoError_EmptyStderrStillErrors(t *testing.T) {
	runErr := errors.New("signal: killed")

	_, err := foldHelperStderrIntoError([]byte(`{"runs":[]}`), nil, runErr)
	if err == nil {
		t.Fatal("a non-zero exit with a silent stderr must still return an error; a silent failure here " +
			"reads to the gate as a convert step that produced no findings")
	}
	if !strings.Contains(err.Error(), "signal: killed") {
		t.Errorf("the error must name the underlying failure even with no diagnostic to add; got: %v", err)
	}
}

// TestFoldHelperStderrIntoError_TruncationKeepsTheTail asserts the direction of
// the bound, which is the whole reason the bound is safe to have.
//
// A runaway converter must not produce an unbounded error string, but truncating
// the WRONG END would defeat the fold entirely: the helper writes its diagnostic
// LAST, after whatever the interpreter spewed on the way down, so keeping the head
// would preserve the noise and discard the reason. The noise here is multi-byte on
// purpose — the cut lands mid-rune, and the returned message must still be valid
// UTF-8 rather than a broken sequence pasted into a CI log.
func TestFoldHelperStderrIntoError_TruncationKeepsTheTail(t *testing.T) {
	noise := strings.Repeat("— converter noise ", 500) // multi-byte, well over the bound
	stderr := noise + "\n" + helperDiagnostic
	if len(stderr) <= maxHelperDiagnosticBytes {
		t.Fatalf("test fixture is too small to exercise truncation: %d bytes vs bound %d",
			len(stderr), maxHelperDiagnosticBytes)
	}

	_, err := foldHelperStderrIntoError([]byte(`{"runs":[]}`), []byte(stderr), errors.New("exit status 126"))
	if err == nil {
		t.Fatal("expected an error on a non-zero exit")
	}

	for _, token := range []string{"andlock", helperKernelRelease, "ISSUE-020"} {
		if !strings.Contains(err.Error(), token) {
			t.Errorf("truncation dropped token %q — the bound cut the TAIL instead of the head, which "+
				"discards exactly the diagnostic this fold exists to preserve. got: %v", token, err)
		}
	}
	if !utf8.ValidString(err.Error()) {
		t.Error("truncation produced invalid UTF-8; the cut must be trimmed forward to a rune boundary")
	}
	if strings.Count(err.Error(), "converter noise") > 300 {
		t.Error("the error string is unbounded — the whole point of the bound is that a runaway converter " +
			"cannot produce one")
	}

	// Whether the bound above lands mid-rune depends on the fixture's exact length,
	// so pin the boundary case deterministically: "—abc" is six bytes, and a limit
	// of four cuts inside the leading three-byte rune. The trim must walk FORWARD to
	// the next boundary. (An earlier draft tested each byte with utf8.ValidString,
	// which rejects a valid multi-byte lead byte and discarded a good boundary.)
	got := tailWithinLimit("—abc", 4)
	if !utf8.ValidString(got) {
		t.Errorf("tailWithinLimit emitted invalid UTF-8 %q when the cut landed inside a rune", got)
	}
	if !strings.HasSuffix(got, "abc") {
		t.Errorf("tailWithinLimit(%q, 4) = %q; the tail after the boundary must survive", "—abc", got)
	}
}

// TestSandboxLinuxStdoutArm_CapturesStderrIntoDistinctBuffer closes the hole the
// fold tests above CANNOT reach.
//
// Every assertion in this file so far exercises foldHelperStderrIntoError, and a
// perfect fold is worthless if nothing hands it any stderr — which is precisely
// what shipped: helper.Stdin and helper.Stdout were set, helper.Stderr was left
// nil, and os/exec routed the diagnostic to /dev/null. That wiring lives in a
// `//go:build linux` file, so no darwin test can execute it, and requiring a Linux
// runner to lock it would rebuild the blind spot that hid the bug for a whole CI
// run.
//
// So assert it STRUCTURALLY, the same way the dispatch test asserts the absence of
// a runtime.GOOS switch: go/parser reads a file as text regardless of its build
// constraints. Two failures are caught here and nowhere else — Stderr not assigned
// at all, and Stderr ALIASED to the stdout buffer, which would pass every test in
// this file while corrupting the SARIF the gate parses.
func TestSandboxLinuxStdoutArm_CapturesStderrIntoDistinctBuffer(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "sandbox_linux.go", nil, 0)
	if err != nil {
		t.Fatalf("parsing sandbox_linux.go: %v", err)
	}

	// linuxSandboxedRunStdoutWith, not platformSandboxedRunStdout: the buffer setup
	// moved into the inner function when the ABI-prober seam was threaded through.
	// This assertion follows the BODY rather than the entry point, and it failed
	// loudly at the move rather than passing against a function that no longer owns
	// the streams — which is what the presence check below is for.
	const fn = "linuxSandboxedRunStdoutWith"
	var target *ast.FuncDecl
	for _, decl := range file.Decls {
		if fd, ok := decl.(*ast.FuncDecl); ok && fd.Name.Name == fn {
			target = fd
		}
	}
	if target == nil {
		t.Fatalf("sandbox_linux.go does not declare %s; this test asserts a property OF that function, so "+
			"it is meaningless if the function moved or was renamed", fn)
	}

	// Collect `helper.<Field> = &<ident>` targets.
	streams := map[string]string{}
	ast.Inspect(target, func(n ast.Node) bool {
		assign, ok := n.(*ast.AssignStmt)
		if !ok || len(assign.Lhs) != 1 || len(assign.Rhs) != 1 {
			return true
		}
		sel, ok := assign.Lhs[0].(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "Stdout" && sel.Sel.Name != "Stderr" {
			return true
		}
		unary, ok := assign.Rhs[0].(*ast.UnaryExpr)
		if !ok {
			return true
		}
		if ident, ok := unary.X.(*ast.Ident); ok {
			streams[sel.Sel.Name] = ident.Name
		}
		return true
	})

	if streams["Stderr"] == "" {
		t.Fatalf("%s never assigns helper.Stderr. os/exec sends a nil Stderr to /dev/null, so the sandbox "+
			"helper's diagnostic naming Landlock, the kernel and ISSUE-020 is DISCARDED — the exact defect "+
			"CI run 30381252600 could not explain", fn)
	}
	if streams["Stdout"] == "" {
		t.Fatalf("%s never assigns helper.Stdout; the captured-stdout contract is broken", fn)
	}
	if streams["Stdout"] == streams["Stderr"] {
		t.Fatalf("%s aliases helper.Stdout and helper.Stderr to the same buffer (%q). Every test in this "+
			"file would still pass while a converter's stderr banner corrupted the SARIF the gate parses",
			fn, streams["Stdout"])
	}
}

// TestSandboxedRun_CombinedOutputArmSurfacesDiagnostic locks the SIBLING arm.
//
// platformSandboxedRun satisfies the token contract BY CONSTRUCTION — CombinedOutput
// merges the helper's stderr into the bytes it returns — and by-construction
// guarantees are precisely the ones that get refactored away without anyone
// noticing. The DRIFT between the two arms is the whole defect: one surfaced the
// diagnostic and the other discarded it, and nothing asserted the difference.
//
// Guarded to darwin because it drives the real sandbox; the Linux arm is covered
// by sandbox_linux_exec_test.go on a host that has one.
func TestSandboxedRun_CombinedOutputArmSurfacesDiagnostic(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("sandbox-exec is macOS-only; the linux arm is covered by sandbox_linux_exec_test.go")
	}
	packDir := mustEvalSymlinks(t, t.TempDir())

	// Stand in for a helper that fails after writing its diagnostic to stderr.
	out, err := SandboxedRun("/bin/sh",
		[]string{"-c", "echo '" + helperDiagnostic + "' 1>&2; exit 126"}, packDir)

	if err == nil {
		t.Fatalf("a command exiting 126 must surface as an error; got output %q", string(out))
	}
	for _, token := range []string{"andlock", helperKernelRelease, "ISSUE-020"} {
		if !strings.Contains(string(out), token) {
			t.Errorf("the CombinedOutput arm dropped token %q. If this fails, the two arms have drifted "+
				"apart again and a Linux sandbox failure is once more undiagnosable. got: %q",
				token, string(out))
		}
	}
}
