package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/pack/distribution"
	"github.com/spf13/cobra"
)

// The REQ-011 renderer suite: every warning renders to STDERR, and rendering one never
// changes an exit code.
//
// NONE OF THESE MAY USE executeCommand. root_test.go:17-23 points cobra's SetOut AND
// SetErr at the SAME bytes.Buffer, so against it "renders to stderr" passes for a command
// that printed to stdout — every claim here would be vacuous. Each test below builds the
// cobra command directly and gives it TWO buffers.
//
// THE NEGATIVE HALF IS THE LOAD-BEARING HALF. A renderer that writes to BOTH streams
// satisfies "the warning appears in stderr" perfectly well; only "and NOT in stdout"
// catches it.
//
// LEAVE THE TWO MERGED-BUFFER INSTALL TESTS ALONE.
// TestPackInstallCommand_PrintsStaleLockWarning (pack_install_test.go:26) and
// TestPackInstallE2E_StaleLockWarnsInstallsDeclared (pack_install_e2e_test.go:83) both
// read a merged buffer and both stay green through the stdout→stderr move. That is
// convenient, and it is also the trap: their green says nothing about streams.

// runWithSeparateStreams executes a cobra command with genuinely independent stdout and
// stderr buffers, returning both plus the command's error.
func runWithSeparateStreams(t *testing.T, cmd *cobra.Command, args ...string) (string, string, error) {
	t.Helper()

	var out, errBuf bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errBuf)
	cmd.SetArgs(args)

	err := cmd.Execute()
	return out.String(), errBuf.String(), err
}

// TestPackInstallCommand_RendersWarningsToStderr pins the MOVE (CLM-110): the existing
// warning loop at pack_install.go:32 goes from cmd.Printf (stdout) to cmd.ErrOrStderr().
//
// It is pinned in BOTH directions on purpose — the warning must land on stderr AND the
// "Installed N packs" summary must stay on stdout. A change that moved everything to
// stderr would satisfy the first assertion alone, and would break every consumer piping
// this command's output.
func TestPackInstallCommand_RendersWarningsToStderr(t *testing.T) {
	projectDir := t.TempDir()

	declared := "backstop/go-standards"
	stale := "slotly/go-standards"

	srcRel := "gostd-src"
	writeFileForTest(t, projectDir, filepath.Join(srcRel, "pack.yml"),
		"name: "+declared+"\nversion: \"1.0.0\"")
	writeFileForTest(t, projectDir, filepath.Join(srcRel, "rules", "r.yml"), "rules: []")
	hash, hashErr := distribution.ComputeContentHash(filepath.Join(projectDir, srcRel))
	if hashErr != nil {
		t.Fatal(hashErr)
	}

	lf := &distribution.Lockfile{
		Packs: map[string]distribution.LockEntry{
			declared: {
				Name: declared, ContentHash: hash, SourceType: "local",
				InstallDate: "2026-01-01T00:00:00Z", LocalPath: srcRel,
			},
			stale: {
				Name: stale, ContentHash: "sha256:stale", SourceType: "local",
				InstallDate: "2026-01-01T00:00:00Z", LocalPath: "renamed-away",
			},
		},
	}
	if err := distribution.WriteLockfile(filepath.Join(projectDir, "backstop.lock"), lf); err != nil {
		t.Fatal(err)
	}
	writeFileForTest(t, projectDir, "backstop.yml", "packs:\n  "+declared+": local")

	restore := chdirForTest(t, projectDir)
	defer restore()

	jsonFlag := false
	stdout, stderr, err := runWithSeparateStreams(t, newPackInstallCommand(&jsonFlag))
	if err != nil {
		t.Fatalf("pack install must succeed while warning: %v (stdout: %s stderr: %s)", err, stdout, stderr)
	}

	if !strings.Contains(stderr, stale) {
		t.Errorf("the reconciliation warning must render to STDERR and name %q; stderr was:\n%s", stale, stderr)
	}
	if strings.Contains(stdout, stale) {
		t.Errorf("the warning must NOT appear on stdout — a renderer that writes to both streams passes the positive assertion alone; stdout was:\n%s", stdout)
	}
	// The other direction: ordinary output stays where consumers expect it.
	if !strings.Contains(stdout, "Installed") {
		t.Errorf("the installed summary must stay on STDOUT; stdout was:\n%s", stdout)
	}
}

// TestPackAddCommand_RendersWarningsToStderr (CLM-109).
//
// It drives a HERMETIC remote rather than a mock, because newPackAddCommand assembles its
// own production dependencies inside RunE with no injection seam — the only way to hand
// the real command a divergent pack is to publish one. divergent-name-pack's manifest
// declares hermetic/renamed-pack while its directory, and therefore the coordinate
// remoteE2ESetup builds, is hermetic/divergent-name-pack.
func TestPackAddCommand_RendersWarningsToStderr(t *testing.T) {
	packName, projectDir := remoteE2ESetup(t, "divergent-name-pack", "v1.0.0")

	restore := chdirForTest(t, projectDir)
	defer restore()

	jsonFlag := false
	stdout, stderr, err := runWithSeparateStreams(t, newPackAddCommand(&jsonFlag), packName+"@1.0.0")
	if err != nil {
		t.Fatalf("a divergent add must SUCCEED while warning — divergence is loud, never fatal: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}

	if !strings.Contains(strings.ToLower(stderr), "warning") {
		t.Errorf("the divergence diagnostic must render to STDERR; stderr was:\n%s", stderr)
	}
	if !strings.Contains(stderr, "hermetic/renamed-pack") {
		t.Errorf("the diagnostic must name the manifest name; stderr was:\n%s", stderr)
	}
	if strings.Contains(strings.ToLower(stdout), "warning") {
		t.Errorf("the diagnostic must NOT appear on stdout — a renderer writing to both streams passes the positive assertion alone; stdout was:\n%s", stdout)
	}
}

// TestPackUpdateCommand_RendersWarningsToStderr (CLM-111).
//
// pack update renders NOTHING of the kind today: UpdateResult has no warning field at all
// (update.go:28-34) and the command has no loop. Both halves arrive in TASK-021; the
// producing path arrives in TASK-035.
func TestPackUpdateCommand_RendersWarningsToStderr(t *testing.T) {
	t.Skip("pack update produces no warning until the coordinate fallback is wired in TASK-035 (phase 10); the rendering loop lands in TASK-021")

	projectDir := t.TempDir()
	restore := chdirForTest(t, projectDir)
	defer restore()

	jsonFlag := false
	stdout, stderr, _ := runWithSeparateStreams(t, newPackUpdateCommand(&jsonFlag), "acme/valid-pack")

	if !strings.Contains(strings.ToLower(stderr), "warning") {
		t.Errorf("the fallback warning must render to STDERR; stderr was:\n%s", stderr)
	}
	if strings.Contains(strings.ToLower(stdout), "warning") {
		t.Errorf("the warning must NOT appear on stdout; stdout was:\n%s", stdout)
	}
}

// TestPackUpgradeCommand_RendersWarningsToStderr (CLM-112).
//
// Same shape as update: UpgradeResult has no warning field today (upgrade.go:22-29) and
// pack upgrade has no loop. The loop goes INSIDE newPackUpgradeCommand's RunE — the
// file's line-1 coverage waiver must not move.
func TestPackUpgradeCommand_RendersWarningsToStderr(t *testing.T) {
	t.Skip("pack upgrade produces no warning until the coordinate fallback is wired in TASK-035 (phase 10); the rendering loop lands in TASK-021")

	projectDir := t.TempDir()
	restore := chdirForTest(t, projectDir)
	defer restore()

	jsonFlag := false
	stdout, stderr, _ := runWithSeparateStreams(t, newPackUpgradeCommand(&jsonFlag), "acme/valid-pack@2.0.0")

	if !strings.Contains(strings.ToLower(stderr), "warning") {
		t.Errorf("the fallback warning must render to STDERR; stderr was:\n%s", stderr)
	}
	if strings.Contains(strings.ToLower(stdout), "warning") {
		t.Errorf("the warning must NOT appear on stdout; stdout was:\n%s", stdout)
	}
}
