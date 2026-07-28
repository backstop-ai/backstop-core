package distribution_test

// Behavior suite for the production Validator (SPEC-055 REQ-003).
//
// Every test here drives the hermetic pack SOURCES authored in phase 1. The
// valid/invalid PAIRING is the falsifier: the passing tests prove the validator
// does not reject everything, and the failing tests prove it does not accept
// everything. Neither half alone establishes that distribution is wired to the
// real pack check / pack test pipeline.
//
// The three fixtures differ in verdict by construction: valid-pack passes both
// modes, invalid-pack fails phase1-structural in BOTH modes, and
// fixture-fail-pack passes check and fails phase3-fixtures — which is the one
// shape that makes a fixture failure distinguishable from a check failure.

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/pack/distribution"
)

// hermeticPackFixture resolves one of the phase-1 hermetic pack SOURCES, relative
// to this package's directory.
//
// The fixtures live under cmd/backstop/testdata because the CLI parity test
// (CLM-019) drives the SAME directories through the built commands; keeping one
// copy is what makes "the validator agrees with the commands" a statement about
// one pack rather than about two hand-synchronized ones.
//
// It fails loudly when the directory or its manifest is missing, so a moved
// fixture surfaces here instead of as a mysterious validation failure that would
// read exactly like the defect these tests exist to catch.
func hermeticPackFixture(t *testing.T, name string) string {
	t.Helper()

	dir := filepath.Join("..", "..", "..", "cmd", "backstop", "testdata", "hermetic-remote", name)
	if _, err := os.Stat(filepath.Join(dir, "pack.yml")); err != nil {
		t.Fatalf("the hermetic pack fixture %s has no readable pack.yml at %s: %v", name, dir, err)
	}

	return dir
}

// assertValidationError asserts err is a *distribution.ValidationError whose
// message names the failing phase and the pack directory.
//
// Asserting the PHASE is what keeps the test honest: a validator that rejected
// every pack, or one that reported a generic "validation failed", would satisfy
// an errors.As check alone. The phase name is the datum that says WHICH pipeline
// stage ran and reached a verdict.
func assertValidationError(t *testing.T, err error, packDir, wantPhase string) *distribution.ValidationError {
	t.Helper()

	if err == nil {
		t.Fatalf("validating %s returned no error; the fixture fails %s, so a nil verdict means the pipeline never ran", packDir, wantPhase)
	}

	var validationErr *distribution.ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("validating %s returned %T (%v), want a *distribution.ValidationError", packDir, err, err)
	}

	message := validationErr.Error()
	if !strings.Contains(message, wantPhase) {
		t.Errorf("the validation diagnostic for %s does not name the failing phase %q: %s", packDir, wantPhase, message)
	}
	if !strings.Contains(message, packDir) {
		t.Errorf("the validation diagnostic does not name the pack directory %q: %s", packDir, message)
	}

	return validationErr
}

// TestPackvalValidator_RunPackCheck_PassesValidPack proves RunPackCheck accepts a
// pack the pack check pipeline accepts (CLM-015). Without this half, a validator
// that rejected every pack would satisfy the failure tests.
func TestPackvalValidator_RunPackCheck_PassesValidPack(t *testing.T) {
	packDir := hermeticPackFixture(t, "valid-pack")

	if err := distribution.NewPackvalValidator().RunPackCheck(packDir); err != nil {
		t.Fatalf("RunPackCheck rejected the hermetic valid pack at %s: %v", packDir, err)
	}
}

// TestPackvalValidator_RunPackCheck_InvalidPackReturnsValidationError proves
// RunPackCheck returns a *ValidationError naming the failing phase for a
// structurally invalid pack (CLM-016).
//
// invalid-pack parses far enough to REACH validation and then fails
// phase1-structural on a rule file that does not exist, so the rejection is
// attributable to the pipeline rather than to an unparseable manifest that would
// have aborted before validation ran.
func TestPackvalValidator_RunPackCheck_InvalidPackReturnsValidationError(t *testing.T) {
	packDir := hermeticPackFixture(t, "invalid-pack")

	err := distribution.NewPackvalValidator().RunPackCheck(packDir)
	validationErr := assertValidationError(t, err, packDir, "phase1-structural")

	// The pack declares exactly one absent rule file, so the pipeline reports one
	// error. Rendering the COUNT is REQ-003's third named datum: it is what tells
	// an operator whether one thing or twenty are wrong before they read the log.
	if !strings.Contains(validationErr.Error(), "1 validation error") {
		t.Errorf("the validation diagnostic does not render the validation error count: %s", validationErr.Error())
	}
}

// TestPackvalValidator_RunPackTest_PassesValidPack proves RunPackTest accepts a
// pack the pack test pipeline accepts (CLM-017) — the fixture phase included,
// which check mode never runs.
func TestPackvalValidator_RunPackTest_PassesValidPack(t *testing.T) {
	packDir := hermeticPackFixture(t, "valid-pack")

	if err := distribution.NewPackvalValidator().RunPackTest(packDir); err != nil {
		t.Fatalf("RunPackTest rejected the hermetic valid pack at %s: %v", packDir, err)
	}
}

// TestPackvalValidator_RunPackTest_FixtureFailureReturnsValidationError proves
// RunPackTest returns a *ValidationError for a pack whose FIXTURE phase fails
// (CLM-018).
//
// The test asserts both halves of the distinguishing property: the same pack
// PASSES RunPackCheck and FAILS RunPackTest in phase3-fixtures. Against a
// validator that ran check-mode in both methods, the second half fails; against
// one that ran test-mode in both, the first half fails. Reusing the
// check-failing invalid-pack here would prove neither.
func TestPackvalValidator_RunPackTest_FixtureFailureReturnsValidationError(t *testing.T) {
	packDir := hermeticPackFixture(t, "fixture-fail-pack")
	validator := distribution.NewPackvalValidator()

	if err := validator.RunPackCheck(packDir); err != nil {
		t.Fatalf("RunPackCheck rejected %s, so a RunPackTest failure would not be attributable to the fixture phase: %v", packDir, err)
	}

	err := validator.RunPackTest(packDir)
	validationErr := assertValidationError(t, err, packDir, "phase3-fixtures")

	// The diagnostic must not read as a check failure. A validator that ran the
	// wrong mode, or one that reported the first phase it looked at rather than
	// the one that failed, would send an author to the manifest for a rule-file
	// mismatch the manifest cannot explain.
	if strings.Contains(validationErr.Error(), "phase1-structural") {
		t.Errorf("the fixture-phase diagnostic names the structural phase, which passed for this pack: %s", validationErr.Error())
	}
}
