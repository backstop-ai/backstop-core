package distribution_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/pack/distribution"
)

// The REQ-008 seam suite: validation runs against a SCRATCH COPY, and a validation
// failure is reported against the ORIGINAL source rather than the scratch directory.
//
// Why the seam takes a LABEL and not only a directory: runPackvalPipeline renders
// "pack validation (%s) of %s failed in %s" using the directory it was handed
// (validator.go:69-71). Once validation moves onto a temp copy, an unmodified pipeline
// shows the operator a /var/folders path that will not exist by the time they look at
// it. A function that received only a directory could not report against anything else,
// which is why the signature is (validator, packDir, sourceLabel).

// scratchSeamValidator captures the directory it was handed and can be told to fail.
type scratchSeamValidator struct {
	checkedDir string
	testedDir  string
	failCheck  bool
	failTest   bool
}

// failureFor reproduces runPackvalPipeline's real diagnostic shape (validator.go:69-71),
// which EMBEDS the directory it was handed inside its own text.
//
// That embedding is the whole reason CLM-088/089 exist, so the double must reproduce it.
// A double returning a fixed string with an unrelated path would make those tests pass no
// matter what the seam did with the message — the assertion would be checking a constant.
func failureFor(mode, packDir string) error {
	return &distribution.ValidationError{Message: fmt.Sprintf(
		"pack validation (%s) of %s failed in phase1-structural: 2 validation error(s)", mode, packDir)}
}

func (v *scratchSeamValidator) RunPackCheck(packDir string) error {
	v.checkedDir = packDir
	if v.failCheck {
		return failureFor("check", packDir)
	}
	return nil
}

func (v *scratchSeamValidator) RunPackTest(packDir string) error {
	v.testedDir = packDir
	if v.failTest {
		return failureFor("test", packDir)
	}
	return nil
}

// writeScratchSourcePack materializes a small pack tree to validate.
func writeScratchSourcePack(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "pack.yml"), "name: acme/valid-pack\nversion: \"1.0.0\"\n")
	writeFile(t, filepath.Join(dir, "rules", "r.yml"), "rules: []\n")
	return dir
}

// TestRunValidationOnScratchCopy_RemovesScratchOnSuccess proves the validator ran against
// a COPY and that the copy is gone when the call returns (CLM-085).
//
// Both halves matter. If the validator were handed the original directory the seam would
// be decorative; if the copy survived, every add would leak a temp tree.
func TestRunValidationOnScratchCopy_RemovesScratchOnSuccess(t *testing.T) {
	source := writeScratchSourcePack(t)
	validator := &scratchSeamValidator{}

	if err := distribution.RunValidationOnScratchCopy(validator, source, "acme/pack at tag v1.0.0"); err != nil {
		t.Fatalf("validation of a good pack must succeed: %v", err)
	}

	if validator.checkedDir == "" {
		t.Fatal("the validator was never invoked; the seam must still run pack check")
	}
	if validator.checkedDir == source {
		t.Errorf("the validator was handed the ORIGINAL directory %q; it must receive a scratch copy, or validation can still contaminate what is installed and hashed", source)
	}
	if validator.testedDir != validator.checkedDir {
		t.Errorf("pack check ran against %q but pack test ran against %q; both must see the same scratch copy", validator.checkedDir, validator.testedDir)
	}

	if _, err := os.Stat(validator.checkedDir); !os.IsNotExist(err) {
		t.Errorf("the scratch copy %q still exists after a successful run (stat error: %v); it must be removed on BOTH paths", validator.checkedDir, err)
	}

	// The source itself is untouched — the seam copies, it does not move.
	if _, err := os.Stat(filepath.Join(source, "pack.yml")); err != nil {
		t.Errorf("the source tree was disturbed: %v", err)
	}
}

// TestRunValidationOnScratchCopy_RemoteFailureNamesCoordinateNotScratchPath (CLM-088).
//
// ASSERTING THE ABSENCE OF THE SCRATCH PATH IS THE LOAD-BEARING HALF. A wrapper that
// PREPENDS the label while leaving the inner pipeline message intact satisfies a
// presence-only assertion and still hands the operator a dead /var/folders path.
func TestRunValidationOnScratchCopy_RemoteFailureNamesCoordinateNotScratchPath(t *testing.T) {
	source := writeScratchSourcePack(t)
	const label = "acme/valid-pack at tag v1.0.0"

	validator := &scratchSeamValidator{failCheck: true}

	err := distribution.RunValidationOnScratchCopy(validator, source, label)
	if err == nil {
		t.Fatal("a failing validator must surface an error")
	}

	msg := err.Error()
	if !strings.Contains(msg, label) {
		t.Errorf("the diagnostic must name the source %q, got: %s", label, msg)
	}
	assertNoScratchPathLeak(t, msg, validator.checkedDir)
}

// TestRunValidationOnScratchCopy_LocalFailureNamesSourcePathNotScratchPath (CLM-089).
// The same property with the OPERATOR'S OWN path as the label — the local-path add is not
// an exception to the seam (spec Review Question 4).
func TestRunValidationOnScratchCopy_LocalFailureNamesSourcePathNotScratchPath(t *testing.T) {
	source := writeScratchSourcePack(t)

	validator := &scratchSeamValidator{failTest: true}

	err := distribution.RunValidationOnScratchCopy(validator, source, source)
	if err == nil {
		t.Fatal("a failing validator must surface an error")
	}

	msg := err.Error()
	if !strings.Contains(msg, source) {
		t.Errorf("the diagnostic must name the operator's own source path %q, got: %s", source, msg)
	}
	assertNoScratchPathLeak(t, msg, validator.testedDir)
}

// assertNoScratchPathLeak requires the rendered diagnostic to mention neither the scratch
// directory the validator was handed nor the OS temp-dir prefix generally. An operator
// cannot inspect either — the directory is gone by the time they read the message.
func assertNoScratchPathLeak(t *testing.T, message, scratchDir string) {
	t.Helper()

	if scratchDir != "" && strings.Contains(message, scratchDir) {
		t.Errorf("the diagnostic quotes the scratch directory %q; that path no longer exists when the operator reads it.\nGot: %s", scratchDir, message)
	}
	// The scratch base name is the durable tell: even a truncated or re-rooted scratch
	// path carries it, so this catches a leak the exact-path check would miss.
	if strings.Contains(message, scratchValidationDirPrefix) {
		t.Errorf("the diagnostic carries the scratch directory prefix %q, so some scratch path leaked into it.\nGot: %s", scratchValidationDirPrefix, message)
	}
}

// scratchValidationDirPrefix is the name pattern the seam gives its temp directories.
// The test asserts on it rather than on os.TempDir() because a source pack legitimately
// lives under t.TempDir() during these tests — asserting "no temp prefix at all" would
// fire on the SOURCE label and make the claim unfalsifiable in the wrong direction.
const scratchValidationDirPrefix = "backstop-pack-validate-"
