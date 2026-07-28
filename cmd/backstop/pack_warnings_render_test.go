package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/pack/distribution"
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
// Driven against a HERMETIC remote for the same reason CLM-109 is: newPackUpdateCommand
// assembles its own production dependencies inside RunE, so the only way to make the real
// command emit a coordinate-fallback warning is to give it a real repository and a lock
// entry that genuinely lacks a coordinate.
//
// The lock entry is stripped of its source_coordinate AFTER the add, which reproduces the
// shape of every entry written before SPEC-056. The fallback resolves to the pack name,
// which for this fixture is also the redirected coordinate, so the update completes
// hermetically while still taking the fallback path.
func TestPackUpdateCommand_RendersWarningsToStderr(t *testing.T) {
	packName, projectDir := remoteE2ESetup(t, "valid-pack", "v1.0.0", "v1.1.0")

	restore := chdirForTest(t, projectDir)
	defer restore()

	jsonFlag := false
	if _, stderr, code := runWithSeparateStreams(t, newPackAddCommand(&jsonFlag), packName+"@1.0.0"); code != nil {
		t.Fatalf("seeding the project with an add failed: %v\nstderr: %s", code, stderr)
	}

	stripRecordedCoordinate(t, projectDir, packName)

	stdout, stderr, err := runWithSeparateStreams(t, newPackUpdateCommand(&jsonFlag), packName)
	if err != nil {
		t.Fatalf("update must succeed while warning: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}

	if !strings.Contains(strings.ToLower(stderr), "warning") {
		t.Errorf("the coordinate-fallback warning must render to STDERR; stderr was:\n%s", stderr)
	}
	if strings.Contains(strings.ToLower(stdout), "warning") {
		t.Errorf("the warning must NOT appear on stdout; stdout was:\n%s", stdout)
	}
}

// stripRecordedCoordinate removes a lock entry's source_coordinate, reproducing the shape
// of every entry written before SPEC-056.
func stripRecordedCoordinate(t *testing.T, projectDir, packName string) {
	t.Helper()
	lockPath := filepath.Join(projectDir, "backstop.lock")
	lf, err := distribution.ReadLockfile(lockPath)
	if err != nil {
		t.Fatalf("reading the lock to strip its coordinate: %v", err)
	}
	entry, ok := lf.Packs[packName]
	if !ok {
		t.Fatalf("no lock entry for %q to strip", packName)
	}
	entry.SourceCoordinate = ""
	lf.Packs[packName] = entry
	if writeErr := distribution.WriteLockfile(lockPath, lf); writeErr != nil {
		t.Fatalf("writing the stripped lock: %v", writeErr)
	}
}

// TestPackUpgradeCommand_RendersWarningsToStderr (CLM-112).
//
// SKIPPED FOR A STRUCTURAL REASON, NOT A MISSING PRODUCING PATH. Upgrade's coordinate
// fallback IS wired (TASK-035) and its carrier IS populated — TestUpgradeResult_CarriesWarnings
// proves that at the distribution level and passes. What cannot happen is the RENDERING.
//
// `pack upgrade` cannot succeed under production wiring: newProductionUpgradeCommand
// assembles unavailableScanner{} and unavailableRemediationGenerator{} by design (SPEC-055
// REQ-009), so every invocation returns "the pack upgrade violation scan is declared but
// not yet available; it is tracked by BUNDLE-006 REQ-014" — verified by running it. The
// rendering loop sits after the error check, and the plan forbids running a warning loop
// on the failure path ("a command that already failed reports its failure through
// packLifecycleFailure"), so there is no honest way to reach the renderer here.
//
// This is the SAME structural fact pack_upgrade.go's line-1 coverage waiver already
// records: "upgrade success path unreachable until the scanner/remediation capability
// seed (BUNDLE-006 REQ-014/018) lands". This skip retires with that seed, alongside the
// waiver.
func TestPackUpgradeCommand_RendersWarningsToStderr(t *testing.T) {
	t.Skip("pack upgrade cannot succeed under production wiring (unavailableScanner, SPEC-055 REQ-009), so the renderer is unreachable; retires with BUNDLE-006 REQ-014/018 alongside pack_upgrade.go's line-1 coverage waiver")

	packName, projectDir := remoteE2ESetup(t, "valid-pack", "v1.0.0", "v2.0.0")

	restore := chdirForTest(t, projectDir)
	defer restore()

	jsonFlag := false
	if _, stderr, code := runWithSeparateStreams(t, newPackAddCommand(&jsonFlag), packName+"@1.0.0"); code != nil {
		t.Fatalf("seeding add failed: %v\nstderr: %s", code, stderr)
	}
	stripRecordedCoordinate(t, projectDir, packName)

	stdout, stderr, err := runWithSeparateStreams(t, newPackUpgradeCommand(&jsonFlag), packName+"@2.0.0")
	if err != nil {
		t.Fatalf("upgrade must succeed while warning: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}
	if !strings.Contains(strings.ToLower(stderr), "warning") {
		t.Errorf("the fallback warning must render to STDERR; stderr was:\n%s", stderr)
	}
	if strings.Contains(strings.ToLower(stdout), "warning") {
		t.Errorf("the warning must NOT appear on stdout; stdout was:\n%s", stdout)
	}
}
