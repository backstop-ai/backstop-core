package distribution_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/pack/distribution"
)

// validation_gate_test.go proves the validator is WIRED, not merely present.
//
// A test that only asserts "a valid pack installs" passes against the broken
// code, because a nil validator made an INVALID pack install just as cleanly.
// So these tests assert the two things a nil-skip cannot fake: that check and
// test actually RUN, in that order, before anything is copied into the consumer
// project, and that a rejection leaves the project completely untouched.

// recordingValidator records the order of the validation calls AND what the
// consumer project looked like at the moment of each call.
//
// The order matters because "runs check and test" is satisfied by running them
// backwards, and the observation matters because "validates before install" is
// satisfied by validating after the copy. Both are correct today by inspection
// and would regress silently without an assertion.
type recordingValidator struct {
	calls []string

	// observe is called with the phase name at the start of each method, so a
	// test can capture consumer state at exactly that instant.
	observe func(phase string)

	checkFail bool
	testFail  bool
}

func (v *recordingValidator) RunPackCheck(_ string) error {
	v.calls = append(v.calls, "check")
	if v.observe != nil {
		v.observe("check")
	}
	if v.checkFail {
		return &distribution.ValidationError{Message: "pack check failed"}
	}
	return nil
}

func (v *recordingValidator) RunPackTest(_ string) error {
	v.calls = append(v.calls, "test")
	if v.observe != nil {
		v.observe("test")
	}
	if v.testFail {
		return &distribution.ValidationError{Message: "pack test failed"}
	}
	return nil
}

// pathExists reports whether a path is present, treating any stat error other
// than not-exist as a test failure rather than as an absence — an unreadable
// path silently read as "absent" would make a mutation assertion pass vacuously.
func pathExists(t *testing.T, path string) bool {
	t.Helper()

	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	t.Fatalf("stat %s: %v", path, err)
	return false
}

// TestAddCommand_RunsCheckThenTestBeforeInstall asserts pack check runs, then
// pack test runs, and BOTH complete before any content reaches the consumer
// project (CLM-050).
func TestAddCommand_RunsCheckThenTestBeforeInstall(t *testing.T) {
	projectDir := setupAddProject(t)
	installedPath := filepath.Join(projectDir, ".backstop", "packs", "acme", "valid-pack")

	installedAt := map[string]bool{}
	validator := &recordingValidator{
		observe: func(phase string) {
			installedAt[phase] = pathExists(t, installedPath)
		},
	}

	add := newTestAddCommand(t, defaultTestPackCloner(), validator)

	if _, err := add.Run("acme/valid-pack@1.0.0", newTestAddOptions(projectDir)); err != nil {
		t.Fatalf("add of a pack that passes validation: %v", err)
	}

	if got := strings.Join(validator.calls, ","); got != "check,test" {
		t.Errorf("validation calls = [%s], want [check,test]; add must run pack check and THEN pack test", got)
	}

	for _, phase := range []string{"check", "test"} {
		if installedAt[phase] {
			t.Errorf("the pack was already copied to %s when pack %s ran; validation must complete before anything is installed", installedPath, phase)
		}
	}

	if !pathExists(t, installedPath) {
		t.Errorf("pack was not installed at %s after validation passed", installedPath)
	}
}

// TestAddCommand_CheckFailureAbortsWithoutMutation asserts a pack check failure
// aborts with the validation diagnostic and leaves NOTHING behind (CLM-051).
//
// All four absences are asserted: a partial rollback that removes the content
// but keeps the manifest entry passes a content-only check.
func TestAddCommand_CheckFailureAbortsWithoutMutation(t *testing.T) {
	projectDir := setupAddProject(t)

	add := newTestAddCommand(t, defaultTestPackCloner(), &recordingValidator{checkFail: true})

	_, err := add.Run("acme/valid-pack@1.0.0", newTestAddOptions(projectDir))
	if err == nil {
		t.Fatal("add must fail when pack check rejects the pack")
	}
	if !strings.Contains(err.Error(), "pack check failed") {
		t.Errorf("error = %v, want it to carry the validation diagnostic %q", err, "pack check failed")
	}

	assertAddLeftNothingBehind(t, projectDir)
}

// TestAddCommand_TestFailureAbortsWithoutMutation asserts a pack test failure
// aborts with the validation diagnostic and leaves NOTHING behind (CLM-052).
func TestAddCommand_TestFailureAbortsWithoutMutation(t *testing.T) {
	projectDir := setupAddProject(t)

	add := newTestAddCommand(t, defaultTestPackCloner(), &recordingValidator{testFail: true})

	_, err := add.Run("acme/valid-pack@1.0.0", newTestAddOptions(projectDir))
	if err == nil {
		t.Fatal("add must fail when pack test rejects the pack")
	}
	if !strings.Contains(err.Error(), "pack test failed") {
		t.Errorf("error = %v, want it to carry the validation diagnostic %q", err, "pack test failed")
	}

	assertAddLeftNothingBehind(t, projectDir)
}

// assertAddLeftNothingBehind checks all four consumer artifacts a rejected add
// must not have touched: installed content, the backstop.yml manifest entry, the
// backstop.lock entry, and the provenance record.
func assertAddLeftNothingBehind(t *testing.T, projectDir string) {
	t.Helper()

	const packName = "acme/valid-pack"

	installedPath := filepath.Join(projectDir, ".backstop", "packs", "acme", "valid-pack")
	if pathExists(t, installedPath) {
		t.Errorf("installed content survives at %s after a validation failure", installedPath)
	}

	manifest := string(mustReadFile(t, filepath.Join(projectDir, "backstop.yml")))
	if strings.Contains(manifest, packName) {
		t.Errorf("backstop.yml declares %s after a validation failure:\n%s", packName, manifest)
	}

	lockPath := filepath.Join(projectDir, "backstop.lock")
	if pathExists(t, lockPath) {
		if _, locked := mustReadLock(t, lockPath).Packs[packName]; locked {
			t.Errorf("backstop.lock records %s after a validation failure", packName)
		}
	}

	provPath := filepath.Join(projectDir, ".backstop", "pack-config-provenance.json")
	if pathExists(t, provPath) {
		if entries := mustReadProvenance(t, provPath).Entries; len(entries) != 0 {
			t.Errorf("provenance holds %d entries after a validation failure, want 0", len(entries))
		}
	}
}

// TestUpgradeCommand_ValidatesBeforeInstall asserts upgrade runs pack check and
// pack test on the NEW version before it replaces the installed one (CLM-053).
//
// The observation is the installed pack's own content: at validation time it
// must still be the OLD version. That is what fails if validation moves after
// the copy, which a mere "check and test were called" assertion would not catch.
func TestUpgradeCommand_ValidatesBeforeInstall(t *testing.T) {
	projectDir := setupUpgradeProject(t)
	installedManifest := filepath.Join(projectDir, ".backstop", "packs", "acme", "valid-pack", "pack.yml")

	contentAt := map[string]string{}
	validator := &recordingValidator{
		observe: func(phase string) {
			contentAt[phase] = string(mustReadFile(t, installedManifest))
		},
	}

	upgrade := newTestUpgradeCommand(t,
		&mockGitCloner{cloneDir: filepath.Join("testdata", "valid-pack-v3")},
		validator,
		&mockScanner{},
		&mockRemediationGenerator{},
	)

	if _, err := upgrade.Run("acme/valid-pack@2.0.0", distribution.UpgradeOptions{ProjectDir: projectDir}); err != nil {
		t.Fatalf("upgrade of a pack that passes validation: %v", err)
	}

	if got := strings.Join(validator.calls, ","); got != "check,test" {
		t.Errorf("validation calls = [%s], want [check,test]; upgrade must validate the new version", got)
	}

	for _, phase := range []string{"check", "test"} {
		if strings.Contains(contentAt[phase], "test-pattern-1-v2") {
			t.Errorf("the new version was already installed when pack %s ran; upgrade must validate before it replaces the installed pack", phase)
		}
	}
}

// TestInstallCommand_DoesNotValidate asserts install RESTORES a pack whose
// content would fail validation (CLM-054).
//
// The fixture is verified to genuinely fail validation by running the production
// validator against it first — restoring a pack that would have passed anyway
// proves nothing about whether install validates.
func TestInstallCommand_DoesNotValidate(t *testing.T) {
	invalidPackDir := filepath.Join("testdata", "invalid-pack")

	if err := distribution.NewPackvalValidator().RunPackCheck(invalidPackDir); err == nil {
		t.Fatalf("the %s fixture passes pack check; this test proves nothing unless its content genuinely fails validation", invalidPackDir)
	}

	projectDir := t.TempDir()
	ref := "v1.0.0"
	lf := &distribution.Lockfile{
		Packs: map[string]distribution.LockEntry{
			"acme/invalid-pack": {
				Name:        "acme/invalid-pack",
				Version:     "1.0.0",
				GitRef:      &ref,
				ContentHash: mustContentHash(t, invalidPackDir),
				SourceType:  "git",
				InstallDate: "2026-01-01T00:00:00Z",
			},
		},
	}
	if err := distribution.WriteLockfile(filepath.Join(projectDir, "backstop.lock"), lf); err != nil {
		t.Fatalf("writing the lock the restore reads: %v", err)
	}
	writeManifestForLock(t, projectDir, lf)

	install := newTestInstallCommand(t, &mockGitCloner{cloneDir: invalidPackDir})

	result, err := install.Run(distribution.InstallOptions{ProjectDir: projectDir})
	if err != nil {
		t.Fatalf("install must restore what the lock records without re-validating it: %v", err)
	}

	if len(result.InstalledPacks) != 1 || result.InstalledPacks[0] != "acme/invalid-pack" {
		t.Fatalf("InstalledPacks = %v, want [acme/invalid-pack]", result.InstalledPacks)
	}

	restored := string(mustReadFile(t, filepath.Join(projectDir, ".backstop", "packs", "acme", "invalid-pack", "pack.yml")))
	if !strings.Contains(restored, "not-semver") {
		t.Errorf("restored pack.yml does not carry the invalid content install was asked to restore:\n%s", restored)
	}
}
