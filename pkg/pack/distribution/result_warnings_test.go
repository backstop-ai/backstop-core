package distribution_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/pack/distribution"
)

// The REQ-011 carrier suite: a warning this spec produces must have somewhere to RIDE.
//
// WHY A FIELD ASSERTION WOULD BE WORTHLESS HERE, and why none of these is written that
// way. `result.Warnings = []string{"x"}; if result.Warnings[0] != "x"` tests the Go
// compiler, not the requirement — and it would keep passing after someone deleted the
// diagnostic that populates the field. The contracts gate cannot see a struct field
// either (it checks symbol existence, not shape), so these claims ARE the enforcement.
// Each test therefore drives the COMMAND and requires a warning the command PRODUCED to
// arrive on the result.
//
// THREE OF THE FOUR ARE SKIPPED ON ARRIVAL, AND THAT IS THE DESIGN. The carriers land in
// this phase deliberately EARLIER than the paths that populate them: a divergence
// diagnostic computed before its carrier exists is computed and dropped, and the code
// looks correct while doing it (spec REQ-011). Each skip names the task that removes it.
// The alternative — weakening these into field assertions so the phase goes green — is
// exactly the vacuous form described above.

// TestAddResult_CarriesWarnings requires a divergent-name add to surface its divergence
// diagnostic on the result (CLM-105).
//
// The divergence comes from the REF, not from a new fixture: testdata/valid-pack's
// manifest declares acme/valid-pack, so requesting acme/renamed-pack makes the manifest
// name and the requested coordinate differ byte-exactly.
func TestAddResult_CarriesWarnings(t *testing.T) {
	projectDir := setupAddProject(t)
	add := newTestAddCommand(t, defaultTestPackCloner(), &mockValidator{})

	result, err := add.Run("acme/renamed-pack@1.0.0", distribution.AddOptions{
		ProjectDir: projectDir,
		Version:    "1.0.0",
	})
	if err != nil {
		t.Fatalf("divergence must never refuse the add: %v", err)
	}
	if len(result.Warnings) == 0 {
		t.Fatal("a divergent-name add must carry its divergence diagnostic on AddResult.Warnings; a warning the result cannot hold is a warning nobody sees")
	}
	if !strings.Contains(strings.Join(result.Warnings, "\n"), "acme/valid-pack") {
		t.Errorf("the divergence diagnostic must name the manifest name, got: %v", result.Warnings)
	}
}

// TestUpdateResult_CarriesWarnings requires update to carry a diagnostic out (CLM-106).
// UpdateResult declares no warning field at all today (update.go:28-34) and pack update
// renders nothing of the kind, so a warning computed inside update is currently dropped.
func TestUpdateResult_CarriesWarnings(t *testing.T) {
	t.Skip("update's coordinate-fallback and divergence warnings are wired in TASK-035 (phase 10); the carrier lands here first")

	projectDir := setupUpdateProject(t)
	update := newTestUpdateCommand(t,
		&mockGitCloner{cloneDir: filepath.Join("testdata", "valid-pack-v2")},
		&mockValidator{},
		&mockVersionResolver{latestMinor: "1.1.0"},
	)

	result, err := update.Run("acme/valid-pack", distribution.UpdateOptions{ProjectDir: projectDir})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	// setupUpdateProject writes a git entry with NO source_coordinate — the shape of
	// every entry written before this spec — so the fallback warning must fire.
	if len(result.Warnings) == 0 {
		t.Fatal("an update resolving a coordinate by fallback must carry the fallback warning on UpdateResult.Warnings")
	}
}

// TestUpgradeResult_CarriesWarnings requires upgrade to carry a diagnostic out (CLM-107).
// UpgradeResult declares no warning field today either (upgrade.go:22-29).
func TestUpgradeResult_CarriesWarnings(t *testing.T) {
	t.Skip("upgrade's coordinate-fallback and divergence warnings are wired in TASK-035 (phase 10); the carrier lands here first")

	projectDir := setupUpgradeProject(t)
	upgrade := newTestUpgradeCommand(t,
		&mockGitCloner{cloneDir: filepath.Join("testdata", "valid-pack-v3")},
		&mockValidator{},
		&mockScanner{},
		&mockRemediationGenerator{},
	)

	result, err := upgrade.Run("acme/valid-pack@2.0.0", distribution.UpgradeOptions{ProjectDir: projectDir})
	if err != nil {
		t.Fatalf("Upgrade: %v", err)
	}
	if len(result.Warnings) == 0 {
		t.Fatal("an upgrade resolving a coordinate by fallback must carry the fallback warning on UpgradeResult.Warnings")
	}
}

// TestInstallResult_CarriesReconciliationAndFallbackWarningsTogether requires BOTH KINDS
// AT ONCE (CLM-108), and that pairing is the entire point.
//
// Install already produces reconciliation warnings (stale lock entries). This spec adds a
// second kind, the coordinate fallback. An implementation that ASSIGNS Warnings rather
// than APPENDING to it passes a test for either kind alone and fails only when both must
// survive the same invocation — which is the bug that would silently delete whichever
// warning was computed first.
func TestInstallResult_CarriesReconciliationAndFallbackWarningsTogether(t *testing.T) {
	t.Skip("install's coordinate-fallback warning is wired in TASK-031 (phase 9); the reconciliation half already works and the carrier is shared")

	projectDir := t.TempDir()

	// A local pack that installs cleanly, so the run succeeds.
	declared := "backstop/go-standards"
	srcRel := "gostd-src"
	writeFile(t, filepath.Join(projectDir, srcRel, "pack.yml"), "name: "+declared+"\nversion: \"1.0.0\"\n")
	hash := mustContentHash(t, filepath.Join(projectDir, srcRel))

	ref := "v1.0.0"
	lf := &distribution.Lockfile{
		Packs: map[string]distribution.LockEntry{
			declared: {
				Name: declared, ContentHash: hash, SourceType: "local",
				InstallDate: "2026-01-01T00:00:00Z", LocalPath: srcRel,
			},
			// Kind 1: a stale entry the manifest does not declare.
			"slotly/go-standards": {
				Name: "slotly/go-standards", ContentHash: "sha256:stale", SourceType: "local",
				InstallDate: "2026-01-01T00:00:00Z", LocalPath: "renamed-away",
			},
			// Kind 2: a git entry carrying NO source_coordinate.
			"acme/valid-pack": {
				Name: "acme/valid-pack", Version: "1.0.0", GitRef: &ref,
				ContentHash: "sha256:whatever", SourceType: "git",
				InstallDate: "2026-01-01T00:00:00Z",
			},
		},
	}
	if err := distribution.WriteLockfile(filepath.Join(projectDir, "backstop.lock"), lf); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(projectDir, "backstop.yml"), "packs:\n  "+declared+": local\n")

	install := newTestInstallCommand(t, defaultTestPackCloner())
	// The git entry's hash will not verify, so the run reports an error. The WARNINGS
	// are what this claim is about, and they must survive alongside it.
	result, runErr := install.Run(distribution.InstallOptions{ProjectDir: projectDir})
	if result == nil {
		t.Fatalf("install returned no result to carry warnings on (err: %v)", runErr)
	}

	joined := strings.Join(result.Warnings, "\n")
	if !strings.Contains(joined, "slotly/go-standards") {
		t.Errorf("the reconciliation warning for the stale entry is missing; got: %v", result.Warnings)
	}
	if !strings.Contains(joined, "acme/valid-pack") {
		t.Errorf("the coordinate-fallback warning is missing; got: %v", result.Warnings)
	}
	if len(result.Warnings) < 2 {
		t.Fatalf("both warning kinds must survive one invocation — an implementation that assigns rather than appends drops one; got %d: %v",
			len(result.Warnings), result.Warnings)
	}
}
