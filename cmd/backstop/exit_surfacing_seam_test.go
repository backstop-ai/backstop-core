package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/scaffold"
)

// SPEC-055 REQ-011, the six commands whose disposition is LOUD and whose diagnostic is
// asserted through the reportError seam (CLM-069..073, CLM-076), plus the invariant the
// whole opt-out direction rests on (CLM-082).
//
// Every test here asserts the DIAGNOSTIC TEXT, never the exit code alone. Each of these
// commands already exited 1 before this spec; what it did not do was say why, and an
// exit-code-only assertion cannot tell those two states apart. That is precisely how the
// defect survived until someone dogfooded `pack relock` (ISSUE-074) and `recipe apply`
// (ISSUE-080).

// reportedThroughSeam drives the ASSEMBLED root command from inside dir — the real
// RunE, not a hand-built error — and funnels the error it returns through the
// production reportError seam.
//
// Obtaining the error from the command rather than constructing an *ExitCodeError here
// is what makes these tests falsifiable twice over: a hand-constructed error would prove
// reportError prints, but not that the COMMAND still populates Message, so a command
// that silently dropped its diagnostic would keep passing.
//
// It returns the seam's diagnostic, the seam's exit code, and separately whatever the
// command itself printed, because "did the command already explain itself" is the exact
// question the Explained opt-out turns on.
func reportedThroughSeam(t *testing.T, dir string, args ...string) (string, int, string) {
	t.Helper()
	t.Chdir(dir)

	printed := new(bytes.Buffer)
	root := NewRootCommand()
	root.SetOut(printed)
	root.SetErr(printed)
	root.SetArgs(args)

	err := root.Execute()
	if err == nil {
		t.Fatalf("backstop %s succeeded; this claim needs its failing path\noutput:\n%s", strings.Join(args, " "), printed.String())
	}

	reported := new(bytes.Buffer)
	code := reportError(reported, err)
	return reported.String(), code, printed.String()
}

// requireLoudViolation asserts the seam reported a violation the operator can act on:
// exit 1 AND a diagnostic naming each wanted fragment. The wanted fragments come from
// the command's own message, so a command that swapped its message for an empty string
// fails here even though its exit code never moved.
func requireLoudViolation(t *testing.T, diagnostic string, code int, want ...string) {
	t.Helper()

	if code != ExitViolations {
		t.Errorf("reportError returned exit %d, want ExitViolations (%d)\ndiagnostic: %q", code, ExitViolations, diagnostic)
	}
	if strings.TrimSpace(diagnostic) == "" {
		t.Fatalf("the seam wrote NOTHING for a failing command: the silent exit-1 defect is still open")
	}
	for _, fragment := range want {
		if !strings.Contains(diagnostic, fragment) {
			t.Errorf("diagnostic %q does not name %q", diagnostic, fragment)
		}
	}
}

// requireNothingPrintedFirst asserts the command wrote nothing of its own before
// failing. It is the PRECONDITION check for the loud default: these six sites cannot
// claim Explained, because there is no report already in front of the operator for the
// seam's line to duplicate.
func requireNothingPrintedFirst(t *testing.T, commandOutput string) {
	t.Helper()
	if strings.TrimSpace(commandOutput) != "" {
		t.Errorf("the command printed a report of its own before failing:\n%s\nif that is now the shipped behavior this site's disposition must be re-decided, not silently kept loud", commandOutput)
	}
}

// consumerWithConfig returns a temp project carrying a backstop.yml the pack lifecycle
// commands can read, so a failure is the command's own and not a missing-config error.
func consumerWithConfig(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "backstop.yml"), []byte("project: exit-surfacing\npacks: {}\n"), 0o644); err != nil {
		t.Fatalf("writing consumer backstop.yml: %v", err)
	}
	return dir
}

// uninstalledPack is a pack name no fixture installs, so every lifecycle command below
// fails on the same honest ground: it was asked for something that is not there.
const uninstalledPack = "demo-org/absent-pack"

// TestExitSurfacing_PackInstall_PrintsDiagnostic — CLM-069. A `pack install` with no
// lockfile has nothing to restore and must say so.
func TestExitSurfacing_PackInstall_PrintsDiagnostic(t *testing.T) {
	diagnostic, code, printed := reportedThroughSeam(t, consumerWithConfig(t), "pack", "install")

	requireLoudViolation(t, diagnostic, code, "backstop.lock")
	requireNothingPrintedFirst(t, printed)
}

// TestExitSurfacing_PackUpdate_PrintsDiagnostic — CLM-070.
func TestExitSurfacing_PackUpdate_PrintsDiagnostic(t *testing.T) {
	diagnostic, code, printed := reportedThroughSeam(t, consumerWithConfig(t), "pack", "update", uninstalledPack)

	requireLoudViolation(t, diagnostic, code, uninstalledPack, "backstop.yml")
	requireNothingPrintedFirst(t, printed)
}

// TestExitSurfacing_PackUpgrade_PrintsDiagnostic — CLM-071. The major-version target is
// explicit because `pack upgrade` requires one; the pack is still absent.
func TestExitSurfacing_PackUpgrade_PrintsDiagnostic(t *testing.T) {
	diagnostic, code, printed := reportedThroughSeam(t, consumerWithConfig(t), "pack", "upgrade", uninstalledPack+"@2.0.0")

	requireLoudViolation(t, diagnostic, code, uninstalledPack, "backstop.yml")
	requireNothingPrintedFirst(t, printed)
}

// TestExitSurfacing_PackRemove_PrintsDiagnostic — CLM-072.
func TestExitSurfacing_PackRemove_PrintsDiagnostic(t *testing.T) {
	diagnostic, code, printed := reportedThroughSeam(t, consumerWithConfig(t), "pack", "remove", uninstalledPack)

	requireLoudViolation(t, diagnostic, code, uninstalledPack, "not installed")
	requireNothingPrintedFirst(t, printed)
}

// TestExitSurfacing_PackList_PrintsDiagnostic — CLM-073. `pack list` reads the manifest
// directly, so a project with no backstop.yml is its failing path.
func TestExitSurfacing_PackList_PrintsDiagnostic(t *testing.T) {
	diagnostic, code, printed := reportedThroughSeam(t, t.TempDir(), "pack", "list")

	requireLoudViolation(t, diagnostic, code, "backstop.yml")
	requireNothingPrintedFirst(t, printed)
}

// fixedTagExecutor reserves IDs from a FIXED tag list and accepts every write, so
// ResolveID returns the SAME id on repeated calls. That determinism is what lets the
// second `artifact new` below compute the path the first one wrote and hit the
// refusal-to-overwrite branch; a resolver that incremented would quietly scaffold a
// second artifact and the claim would never be exercised.
type fixedTagExecutor struct{}

func (*fixedTagExecutor) ListTags(string) ([]string, error)       { return nil, nil }
func (*fixedTagExecutor) CreateAnnotatedTag(string, string) error { return nil }
func (*fixedTagExecutor) PushTag(string) error                    { return nil }
func (*fixedTagExecutor) FetchTags() error                        { return nil }
func (*fixedTagExecutor) IsGitRepo() bool                         { return true }
func (*fixedTagExecutor) IsGitAvailable() bool                    { return true }

// TestExitSurfacing_ArtifactNew_PrintsDiagnostic — CLM-076, the refusal-to-overwrite
// path (artifact_new.go:110). This one is driven through the deps-injected constructor
// rather than the root command because the shipped constructor reaches for the real git
// binary; the RunE under test is the same one either way.
func TestExitSurfacing_ArtifactNew_PrintsDiagnostic(t *testing.T) {
	projectDir := t.TempDir()
	const slug = "exit-surfacing"

	newArtifact := func() (string, error) {
		cmd := newArtifactNewCommandWithDeps(scaffold.ArtifactNewDeps{
			Executor:    &fixedTagExecutor{},
			ProjectRoot: projectDir,
			DateFunc:    func() string { return "2026-07-26" },
		})
		printed := new(bytes.Buffer)
		cmd.SetOut(printed)
		cmd.SetErr(printed)
		cmd.SetArgs([]string{"spec", "--slug", slug})
		return printed.String(), cmd.Execute()
	}

	if _, err := newArtifact(); err != nil {
		t.Fatalf("scaffolding the first artifact failed: %v", err)
	}

	printed, err := newArtifact()
	if err == nil {
		t.Fatalf("the second artifact new overwrote the first; it must refuse\noutput:\n%s", printed)
	}

	reported := new(bytes.Buffer)
	code := reportError(reported, err)

	requireLoudViolation(t, reported.String(), code, "already exists", slug)
	requireNothingPrintedFirst(t, printed)
}

// TestExitSurfacing_DefaultIsLoud — CLM-082. THE invariant the opt-out direction rests
// on: an *ExitCodeError carrying a message with Explained unset prints, whatever its
// code. If a future edit ever re-couples suppression to a code, this fails first.
func TestExitSurfacing_DefaultIsLoud(t *testing.T) {
	const message = "the operator must be able to read this"

	// ExitPass is in the table deliberately. Explained is the ONLY suppressor; a code
	// the reporter happens to consider benign is not one.
	for _, code := range []int{ExitPass, ExitViolations, ExitConfigError, 7} {
		reported := new(bytes.Buffer)
		got := reportError(reported, &ExitCodeError{Code: code, Message: message})

		if got != code {
			t.Errorf("reportError returned %d for an ExitCodeError with code %d, want the error's own code", got, code)
		}
		if !strings.Contains(reported.String(), message) {
			t.Errorf("reportError wrote %q for an unexplained ExitCodeError with code %d, want it to carry the message %q", reported.String(), code, message)
		}
	}

	// The falsifying twin: the SAME error, explained, writes nothing. Without it
	// "always print something" would pass the loop above, and the opt-out that the four
	// sanctioned sites depend on would be unproven in this direction.
	explained := new(bytes.Buffer)
	if got := reportError(explained, &ExitCodeError{Code: ExitViolations, Message: message, Explained: true}); got != ExitViolations {
		t.Errorf("reportError returned %d for an explained error, want ExitViolations (%d)", got, ExitViolations)
	}
	if explained.String() != "" {
		t.Errorf("reportError wrote %q for an EXPLAINED error, want nothing", explained.String())
	}
}
