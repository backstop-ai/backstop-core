package main

// Parity suite for the production Validator (SPEC-055 REQ-003 / CLM-019).
//
// REQ-003 forbids a SECOND implementation of pack validation: distribution must
// apply exactly the authority `backstop pack check` and `backstop pack test`
// apply. The only way to state that as a test is to run both and compare
// verdicts on the same directory — which is why this claim lives in cmd/backstop
// rather than in the distribution package, where the commands are unreachable.
//
// It is also the PROOF OF THE FIXTURES (spec Review Question 5). A hermetic pack
// hand-written to satisfy whatever the validator happened to accept falsifies
// nothing; running the real commands against it is what establishes that
// valid-pack genuinely passes and invalid-pack genuinely fails. The passing half
// is therefore not "obviously green" — dropping it would discard that proof.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/pack/distribution"
)

// parityFixture resolves a hermetic pack SOURCE relative to this package.
func parityFixture(t *testing.T, name string) string {
	t.Helper()

	dir := filepath.Join("testdata", "hermetic-remote", name)
	if _, err := os.Stat(filepath.Join(dir, "pack.yml")); err != nil {
		t.Fatalf("the hermetic pack fixture %s has no readable pack.yml at %s: %v", name, dir, err)
	}

	return dir
}

// TestPackvalValidator_MatchesPackCheckAndPackTestCommandVerdicts proves the
// validator's pass/fail verdict equals the CLI commands' verdict in every cell of
// the fixture × mode matrix (CLM-019).
//
// A disagreement in ANY cell means distribution is not running the same pipeline
// and REQ-003 is unmet. Both a passing and a failing fixture are driven, because
// two implementations that both always-pass, or both always-fail, would agree
// vacuously across a single-verdict matrix.
func TestPackvalValidator_MatchesPackCheckAndPackTestCommandVerdicts(t *testing.T) {
	validator := distribution.NewPackvalValidator()

	cases := []struct {
		fixture  string
		wantFail bool
	}{
		{fixture: "valid-pack", wantFail: false},
		{fixture: "invalid-pack", wantFail: true},
	}

	for _, testCase := range cases {
		packDir := parityFixture(t, testCase.fixture)

		checkCommandFailed := packCommandFails(t, "check", packDir)
		testCommandFailed := packCommandFails(t, "test", packDir)
		checkValidatorFailed := validator.RunPackCheck(packDir) != nil
		testValidatorFailed := validator.RunPackTest(packDir) != nil

		// The expectation is pinned per fixture as well as compared, so a build in
		// which BOTH sides silently flipped verdicts still fails here.
		if checkCommandFailed != testCase.wantFail {
			t.Errorf("`pack check %s` failed=%v, want failed=%v — the fixture no longer has the verdict this test was built on", packDir, checkCommandFailed, testCase.wantFail)
		}

		if checkValidatorFailed != checkCommandFailed {
			t.Errorf("RunPackCheck(%s) failed=%v but `pack check` failed=%v; distribution is not running the pack check pipeline", packDir, checkValidatorFailed, checkCommandFailed)
		}
		if testValidatorFailed != testCommandFailed {
			t.Errorf("RunPackTest(%s) failed=%v but `pack test` failed=%v; distribution is not running the pack test pipeline", packDir, testValidatorFailed, testCommandFailed)
		}
	}
}

// packCommandFails runs `pack <mode> <packDir>` through a freshly built root
// command and reports whether it exited non-zero.
//
// A fresh root per invocation matches how the binary runs one command per
// process; reusing one would let flag state from an earlier subcommand leak into
// the next and quietly change a verdict.
func packCommandFails(t *testing.T, mode, packDir string) bool {
	t.Helper()

	out, err := executeCommand(NewRootCommand(), "pack", mode, packDir)
	if out == "" {
		t.Fatalf("`pack %s %s` printed nothing; the command did not run its pipeline", mode, packDir)
	}

	return err != nil
}
