package distribution_test

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/pack/distribution"
)

// The REQ-008 contamination suite: what a validator writes must reach neither the
// installed tree nor the recorded content hash, in ALL THREE validating commands.
//
// WHY ALL THREE AND NOT JUST ADD. AddCommand.Run (command.go:154), UpdateCommand.Run
// (:539) and UpgradeCommand.Run (:654) carry the IDENTICAL defect — validate a tree in
// place, then copy and hash that same tree. A fix landing in add alone leaves two thirds
// of the requirement standing, and nothing about add's own tests would notice (spec
// Review Question 3). These three tests are what catch a partial fix.

// contaminatingValidator writes a marker file into whatever directory it validates,
// reproducing what pkg/packval genuinely does: phase 3 renders every tier:complete
// scaffold's sample_config into <packDir>/<scaffold.path>/ before running its test
// command. TestRunFixtures_CompleteScaffoldWritesSampleConfigIntoPackDir (CLM-080)
// characterizes the real thing; this double makes it cheap to provoke on demand.
type contaminatingValidator struct {
	markerRel string
	// calls counts invocations so a caller can still prove validation RAN — the
	// not-mutated assertion alone would also pass against a command that skipped
	// validation entirely.
	calls int
}

const contaminationMarkerRel = "rendered-by-validator.yml"

func newContaminatingValidator() *contaminatingValidator {
	return &contaminatingValidator{markerRel: contaminationMarkerRel}
}

func (v *contaminatingValidator) write(packDir string) error {
	v.calls++
	return os.WriteFile(filepath.Join(packDir, v.markerRel), []byte("marker: rendered\n"), 0o644)
}

func (v *contaminatingValidator) RunPackCheck(packDir string) error { return v.write(packDir) }
func (v *contaminatingValidator) RunPackTest(packDir string) error  { return v.write(packDir) }

// installedPackDir is where a command materializes a pack inside a consumer project.
func installedPackDir(projectDir, packName string) string {
	return filepath.Join(projectDir, ".backstop", "packs", packName)
}

// assertMarkerAbsent requires the validator's write to have missed the installed tree.
func assertMarkerAbsent(t *testing.T, installedPath string) {
	t.Helper()
	marker := filepath.Join(installedPath, contaminationMarkerRel)
	if _, err := os.Stat(marker); err == nil {
		t.Errorf("the validator's write reached the INSTALLED tree at %s; validation ran against the directory that was then copied", marker)
	} else if !os.IsNotExist(err) {
		t.Errorf("stat %s: %v", marker, err)
	}
}

func TestAddCommand_Run_ValidatorWritesDoNotReachInstalledContent(t *testing.T) {
	projectDir := setupAddProject(t)
	add := newTestAddCommand(t, defaultTestPackCloner(), newContaminatingValidator())

	if _, err := add.Run("acme/valid-pack@1.0.0", newTestAddOptions(projectDir)); err != nil {
		t.Fatalf("Add: %v", err)
	}

	assertMarkerAbsent(t, installedPackDir(projectDir, "acme/valid-pack"))
}

// TestAddCommand_Run_ContentHashExcludesValidatorWrites compares the recorded hash to an
// INDEPENDENTLY computed hash of the pristine source (CLM-082).
//
// Comparing two adds against each other would prove nothing: two adds contaminated
// IDENTICALLY agree with each other perfectly. The pristine fixture is the only
// reference that can actually fail.
func TestAddCommand_Run_ContentHashExcludesValidatorWrites(t *testing.T) {
	projectDir := setupAddProject(t)

	pristine, hashErr := distribution.ComputeContentHash(filepath.Join("testdata", "valid-pack"))
	if hashErr != nil {
		t.Fatalf("computing the pristine reference hash: %v", hashErr)
	}

	add := newTestAddCommand(t, defaultTestPackCloner(), newContaminatingValidator())
	result, err := add.Run("acme/valid-pack@1.0.0", newTestAddOptions(projectDir))
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	if result.ContentHash != pristine {
		t.Errorf("recorded hash %q != the pristine tree's %q — the hash was computed over a tree the validator had written to, so a fresh clone of the same tag can never reproduce it",
			result.ContentHash, pristine)
	}
}

// TestUpdateCommand_Run_ValidatorWritesReachNeitherContentNorHash (CLM-083).
//
// IT ALSO CARRIES THE TAMPER-ADJACENT REGRESSION, which is the second-order failure the
// scratch copy prevents. Update runs DetectTamper(currentPackDir, tmpDir) at
// command.go:549. If validation contaminated tmpDir, the rendered files appear as ADDED
// and DetectTamper reports tamper on a pack nobody touched — the operator is told their
// installed pack was modified when in fact the tool modified the comparison input.
func TestUpdateCommand_Run_ValidatorWritesReachNeitherContentNorHash(t *testing.T) {
	projectDir := setupUpdateProject(t)

	pristine, hashErr := distribution.ComputeContentHash(filepath.Join("testdata", "valid-pack-v2"))
	if hashErr != nil {
		t.Fatalf("computing the pristine reference hash: %v", hashErr)
	}

	update := newTestUpdateCommand(t,
		&mockGitCloner{cloneDir: filepath.Join("testdata", "valid-pack-v2")},
		newContaminatingValidator(),
		&mockVersionResolver{latestMinor: "1.1.0"},
	)

	result, err := update.Run("acme/valid-pack", distribution.UpdateOptions{ProjectDir: projectDir})
	if err != nil {
		t.Fatalf("a contaminating validator must not fail the update — and must not trip tamper detection on a pack nobody touched: %v", err)
	}

	assertMarkerAbsent(t, installedPackDir(projectDir, "acme/valid-pack"))
	if result.ContentHash != pristine {
		t.Errorf("recorded hash %q != the pristine tree's %q", result.ContentHash, pristine)
	}
}

// TestUpgradeCommand_Run_ValidatorWritesReachNeitherContentNorHash (CLM-084).
func TestUpgradeCommand_Run_ValidatorWritesReachNeitherContentNorHash(t *testing.T) {
	projectDir := setupUpgradeProject(t)

	pristine, hashErr := distribution.ComputeContentHash(filepath.Join("testdata", "valid-pack-v3"))
	if hashErr != nil {
		t.Fatalf("computing the pristine reference hash: %v", hashErr)
	}

	upgrade := newTestUpgradeCommand(t,
		&mockGitCloner{cloneDir: filepath.Join("testdata", "valid-pack-v3")},
		newContaminatingValidator(),
		&mockScanner{},
		&mockRemediationGenerator{},
	)

	result, err := upgrade.Run("acme/valid-pack@2.0.0", distribution.UpgradeOptions{ProjectDir: projectDir})
	if err != nil {
		t.Fatalf("Upgrade: %v", err)
	}

	assertMarkerAbsent(t, installedPackDir(projectDir, "acme/valid-pack"))
	if result.ContentHash != pristine {
		t.Errorf("recorded hash %q != the pristine tree's %q", result.ContentHash, pristine)
	}
}

// failingContaminatingValidator writes its marker AND then fails, so a test can prove the
// scratch copy is removed on the FAILURE path too.
type failingContaminatingValidator struct {
	contaminatingValidator
	handedDir string
}

// handedDir records the directory the seam handed over, so a test can assert THAT
// specific scratch tree is gone rather than scanning the shared OS temp area.
func (v *failingContaminatingValidator) RunPackCheck(packDir string) error {
	v.handedDir = packDir
	_ = v.write(packDir)
	return errors.New("pack validation (check) failed in phase1-structural: 1 validation error(s)")
}

// TestAddCommand_Run_ScratchCopyRemovedOnValidationFailure asserts THREE properties
// (CLM-086), because a scratch cleanup that ran while the install was left half-written
// is not the property REQ-008 wants.
func TestAddCommand_Run_ScratchCopyRemovedOnValidationFailure(t *testing.T) {
	projectDir := setupAddProject(t)

	validator := &failingContaminatingValidator{contaminatingValidator: contaminatingValidator{markerRel: contaminationMarkerRel}}
	add := newTestAddCommand(t, defaultTestPackCloner(), validator)

	if _, err := add.Run("acme/valid-pack@1.0.0", newTestAddOptions(projectDir)); err == nil {
		t.Fatal("a failing validator must fail the add")
	}

	// 1. No installed content.
	if _, err := os.Stat(installedPackDir(projectDir, "acme/valid-pack")); !os.IsNotExist(err) {
		t.Errorf("a validation failure left installed content behind (stat error: %v)", err)
	}
	// 2. No lock entry — and no lock file at all, since this project had none.
	if _, err := os.Stat(filepath.Join(projectDir, "backstop.lock")); !os.IsNotExist(err) {
		t.Errorf("a validation failure wrote backstop.lock (stat error: %v)", err)
	}
	// 3. The scratch copy itself is gone.
	//
	// It asserts on the SPECIFIC directory this run was handed, not on a scan of the
	// shared OS temp area: a leftover from any other process — or from a previously
	// crashed run — would make a directory scan fail for reasons unrelated to the code
	// under test, which is a flaky assertion rather than a strong one.
	if validator.handedDir == "" {
		t.Fatal("the validator was never handed a directory; the seam did not run")
	}
	if _, err := os.Stat(validator.handedDir); !os.IsNotExist(err) {
		t.Errorf("the scratch copy %q survived a validation FAILURE (stat error: %v); it must be removed on both paths", validator.handedDir, err)
	}
}

// TestAddCommand_Run_LocalSourceDirectoryNotMutatedByValidation is the local half of
// CLM-087, and it is the case with the sharpest consequence: for a local-path add the
// directory packval mutates is the OPERATOR'S OWN WORKING TREE (spec Review Question 4).
// A seam applied to the remote branch alone would leave a tool that writes files into a
// developer's checkout every time they add a local pack.
func TestAddCommand_Run_LocalSourceDirectoryNotMutatedByValidation(t *testing.T) {
	projectDir := setupAddProject(t)

	// The operator's own directory, a copy of the local-pack fixture.
	operatorSource := filepath.Join(t.TempDir(), "my-pack")
	if err := os.MkdirAll(operatorSource, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := copyDir(filepath.Join("testdata", "local-pack"), operatorSource); err != nil {
		t.Fatalf("seeding the operator source: %v", err)
	}

	before := snapshotTree(t, operatorSource)

	add := newTestAddCommand(t, defaultTestPackCloner(), newContaminatingValidator())
	if _, err := add.Run(operatorSource, distribution.AddOptions{ProjectDir: projectDir}); err != nil {
		t.Fatalf("a local-path add must succeed: %v", err)
	}

	after := snapshotTree(t, operatorSource)
	if len(before) != len(after) {
		t.Fatalf("the operator's source directory gained or lost files during validation: %d before, %d after\nbefore: %v\nafter:  %v",
			len(before), len(after), before, after)
	}
	for i := range before {
		if before[i] != after[i] {
			t.Errorf("the operator's own working tree was modified by validation; entry %d differs", i)
		}
	}
}

// snapshotTree records every file path and its contents under root, sorted.
func snapshotTree(t *testing.T, root string) []string {
	t.Helper()
	var out []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		out = append(out, rel+"\x00"+string(data))
		return nil
	})
	if err != nil {
		t.Fatalf("snapshotting %s: %v", root, err)
	}
	sort.Strings(out)
	return out
}
