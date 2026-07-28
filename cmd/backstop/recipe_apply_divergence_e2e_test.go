package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/recipe"
)

// ISSUE-080, driven through the SHIPPED binary.
//
// The library tests next door in pkg/recipe prove the applier's decision. They
// cannot prove what the OPERATOR experiences, and the operator's experience was
// the defect: `backstop recipe apply` regenerated over a hand-edited file marked
// with a malformed @waiver, exited 0, and printed one cheerful line. Only a run of
// the real command can falsify that.
//
// Streams are captured SEPARATELY (runBackstopStreams, never a CombinedOutput
// helper). Against one merged buffer "the diagnostic is on stderr" is
// unfalsifiable, and "stderr is empty" — the contrast that makes the
// covered-plus-malformed warning meaningful — cannot be expressed at all.
const (
	recipeDivergenceProject = "recipe-divergence-e2e"
	recipeDivergencePack    = "demo-org/divergence-pack"
	recipeDivergenceID      = "divergence"
)

// divergenceReasonCode is the illegal reason code the operator's token carries. It
// is outside pkg/waiver's closed ReasonCode enum, so the token does not parse.
const divergenceReasonCode = "intentional-fork"

// stageRecipeDivergenceProject copies the committed divergence fixture project —
// its installed pack under .backstop/packs included — into a fresh temp root, so a
// run mutates the copy and never the tracked fixture. It is separate from
// stageRecipeE2EProject because it stages a DIFFERENT project; sharing one helper
// would mean parameterizing a fixture name that reads better as a constant.
func stageRecipeDivergenceProject(t *testing.T) string {
	t.Helper()
	src, err := filepath.Abs(filepath.Join("testdata", recipeDivergenceProject))
	if err != nil {
		t.Fatalf("resolve fixture project %q: %v", recipeDivergenceProject, err)
	}
	dst := t.TempDir()
	copyTree(t, src, dst)
	return dst
}

// divergenceFixture stages a fresh copy of the project and reads back everything a
// case needs from the fixture's OWN parsed manifests: the pinned ref, the declared
// target, the declared enforcement rule and the payload bytes. Nothing here is
// retyped as a literal, so a fixture edit cannot silently desynchronize the tests.
type divergenceFixture struct {
	root    string
	ref     string
	target  string
	rule    string
	version string
	payload string
}

func stageDivergenceFixture(t *testing.T) divergenceFixture {
	t.Helper()

	root := stageRecipeDivergenceProject(t)
	ref, parsed, recipeDir := stagedRecipe(t, root, recipeDivergencePack, recipeDivergenceID)

	if parsed.Enforcement == nil || len(parsed.Enforcement.Rules) == 0 {
		t.Fatalf("fixture recipe %q declares no enforcement rules; a divergence has no rule id to be adjudicated against", recipeDivergenceID)
	}
	op := parsed.Ops[0]
	payload, err := os.ReadFile(filepath.Join(recipeDir, filepath.FromSlash(op.Payload)))
	if err != nil {
		t.Fatalf("read declared payload %q: %v", op.Payload, err)
	}

	return divergenceFixture{
		root:    root,
		ref:     ref,
		target:  op.Target,
		rule:    parsed.Enforcement.Rules[0],
		version: parsed.Version,
		payload: string(payload),
	}
}

// targetPath is the absolute path of the recipe's declared target inside the
// staged project.
func (f divergenceFixture) targetPath() string {
	return filepath.Join(f.root, filepath.FromSlash(f.target))
}

// readTarget reads the declared target back off disk.
func readTarget(t *testing.T, f divergenceFixture) string {
	t.Helper()
	body, err := os.ReadFile(f.targetPath())
	if err != nil {
		t.Fatalf("read declared target %q: %v", f.target, err)
	}
	return string(body)
}

// writeDivergence overwrites the recipe-owned target with a consumer edit whose
// tokenLine sits on line 4 — neither the first line nor the second, per the line
// contract. A token on line 1 would be found by an applier that only scanned the
// first association window, so it would prove nothing.
func writeDivergence(t *testing.T, f divergenceFixture, tokenLine string) string {
	t.Helper()
	diverged := "# recipe-owned output\nalpha\nCONSUMER EDIT the recipe did not write\n" + tokenLine + "\ncharlie\n"
	if err := os.WriteFile(f.targetPath(), []byte(diverged), 0o644); err != nil {
		t.Fatalf("write consumer divergence at %q: %v", f.target, err)
	}
	return diverged
}

// writeDivergenceWithTwoTokens places first on line 3 and second on line 4, for the
// covered-AND-malformed case.
func writeDivergenceWithTwoTokens(t *testing.T, f divergenceFixture, first string, second string) string {
	t.Helper()
	diverged := "# recipe-owned output\nalpha\n" + first + "\n" + second + "\ncharlie\n"
	if err := os.WriteFile(f.targetPath(), []byte(diverged), 0o644); err != nil {
		t.Fatalf("write consumer divergence at %q: %v", f.target, err)
	}
	return diverged
}

// malformedTokenLineFor and validTokenLineFor build the operator's two tokens
// against the fixture's OWN declared rule id.
func malformedTokenLineFor(rule string) string {
	return "# @waiver:" + rule + ":" + divergenceReasonCode + ":2099-01-01 hand-edited on purpose"
}

func validTokenLineFor(rule string) string {
	return "# @waiver:" + rule + ":accepted-risk:2099-01-01 divergence accepted by the consumer"
}

// applyOnce runs the shipped `recipe apply` and returns its separated streams and
// exit code.
func applyOnce(t *testing.T, bin string, f divergenceFixture) (string, string, int) {
	t.Helper()
	return runBackstopStreams(t, bin, f.root, "recipe", "apply", f.ref)
}

// adoptThroughTheBinary performs the FIRST apply, which is what makes the target
// recipe-owned: it writes the adoption record. The adopted state is PRODUCED by the
// real command rather than staged by hand, so no test depends on a hand-written
// record being shaped correctly.
func adoptThroughTheBinary(t *testing.T, bin string, f divergenceFixture) (string, string) {
	t.Helper()

	stdout, stderr, code := applyOnce(t, bin, f)
	if code != 0 {
		t.Fatalf("first apply exited %d, want 0\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	if got := readTarget(t, f); got != f.payload {
		t.Fatalf("first apply produced %q at %q, want the recipe's declared payload %q", got, f.target, f.payload)
	}
	return stdout, stderr
}

// TestRecipeApply_CLI_MalformedWaiverTokenSurfacesDiagnosticOnStderr is ISSUE-080's
// live field repro, end to end (CLM-009).
//
// The byte-for-byte survival assertion is the one that matters most: the PRE-FIX
// binary passes every stream assertion here trivially by exiting 0 with an empty
// stderr, and fails only that one — because it had already destroyed the
// operator's edit.
func TestRecipeApply_CLI_MalformedWaiverTokenSurfacesDiagnosticOnStderr(t *testing.T) {
	bin := buildBackstopBinary(t)
	fixture := stageDivergenceFixture(t)

	adoptThroughTheBinary(t, bin, fixture)

	// The precondition, read back through the applier's own reader: the first run
	// recorded the recipe as ADOPTED, which is the only thing that makes the target
	// recipe-owned rather than the consumer's outright. Without it the re-apply
	// below would take the user-owned branch and this test would pass for entirely
	// the wrong reason.
	adopted, err := recipe.ReadAdoptions(filepath.Join(fixture.root, recipe.AdoptionRecordName))
	if err != nil {
		t.Fatalf("read the adoption record the first apply wrote: %v", err)
	}
	if len(adopted.Recipes) != 1 {
		t.Fatalf("adoption record holds %d entries (%+v), want exactly the one the first apply recorded", len(adopted.Recipes), adopted.Recipes)
	}
	for _, entry := range adopted.Recipes {
		if entry.Version != fixture.version {
			t.Errorf("adoption entry pins version %q, want the recipe's declared %q", entry.Version, fixture.version)
		}
	}

	diverged := writeDivergence(t, fixture, malformedTokenLineFor(fixture.rule))

	stdout, stderr, code := applyOnce(t, bin, fixture)

	if code != ExitViolations {
		t.Errorf("re-apply over a malformed token exited %d, want ExitViolations (%d)\nstdout: %s\nstderr: %s", code, ExitViolations, stdout, stderr)
	}
	if strings.TrimSpace(stderr) == "" {
		t.Errorf("stderr is empty; the operator is told nothing about why the apply refused")
	}
	if !strings.Contains(stderr, fixture.target) {
		t.Errorf("stderr %q does not name the target %q", stderr, fixture.target)
	}
	if !strings.Contains(stderr, divergenceReasonCode) {
		t.Errorf("stderr %q does not name the illegal reason code %q", stderr, divergenceReasonCode)
	}
	if strings.Contains(stdout, "applied recipe") {
		t.Errorf("stdout %q claims the recipe was applied; this run applied nothing and destroyed nothing", stdout)
	}

	if got := readTarget(t, fixture); got != diverged {
		t.Errorf("the operator's edit at %q is %q, want it untouched %q — a refused apply rewrites nothing", fixture.target, got, diverged)
	}
	if got := readTarget(t, fixture); got == fixture.payload {
		t.Errorf("the recipe's payload is back at %q; the operator's edit was destroyed by the apply that refused the token", fixture.target)
	}
}

// TestRecipeApply_CLI_SuccessOutputNamesWrittenPreservedAndRegenerated closes the
// compounding gap (CLM-007/CLM-010/CLM-012): before this, every successful apply
// printed the same single line, so a CLOBBER and a clean re-apply were
// indistinguishable to the operator.
//
// Four arms over four FRESH staged copies. The arms are compared against each
// other, not just against substrings: a claim that a clobber is now visible is not
// satisfied by four identical strings.
func TestRecipeApply_CLI_SuccessOutputNamesWrittenPreservedAndRegenerated(t *testing.T) {
	bin := buildBackstopBinary(t)

	var cleanStdout, preservedStdout, regeneratedStdout string

	t.Run("clean_first_apply_names_the_written_target", func(t *testing.T) {
		fixture := stageDivergenceFixture(t)

		stdout, stderr, code := applyOnce(t, bin, fixture)
		if code != 0 {
			t.Fatalf("clean apply exited %d, want 0\nstdout: %s\nstderr: %s", code, stdout, stderr)
		}
		if !strings.Contains(stdout, fixture.target) {
			t.Errorf("stdout %q does not name the written target %q; a bare headline tells the operator nothing about what landed", stdout, fixture.target)
		}
		if strings.Contains(strings.ToLower(stdout), "regenerated") {
			t.Errorf("stdout %q marks a regeneration; this apply wrote over nothing", stdout)
		}

		// The run that WROTE something must also have adopted it — read back
		// through the applier's own reader, so the assertion is about the record
		// the shipped command actually produced.
		adopted, err := recipe.ReadAdoptions(filepath.Join(fixture.root, recipe.AdoptionRecordName))
		if err != nil {
			t.Fatalf("read the adoption record the clean apply wrote: %v", err)
		}
		if len(adopted.Recipes) != 1 {
			t.Fatalf("adoption record holds %d entries (%+v), want exactly one", len(adopted.Recipes), adopted.Recipes)
		}

		cleanStdout = stdout
	})

	t.Run("valid_token_preserves_quotes_the_token_and_warns_about_nothing", func(t *testing.T) {
		fixture := stageDivergenceFixture(t)
		adoptThroughTheBinary(t, bin, fixture)

		token := validTokenLineFor(fixture.rule)
		diverged := writeDivergence(t, fixture, token)

		stdout, stderr, code := applyOnce(t, bin, fixture)
		if code != 0 {
			t.Fatalf("re-apply over a covered divergence exited %d, want 0\nstdout: %s\nstderr: %s", code, stdout, stderr)
		}
		if got := readTarget(t, fixture); got != diverged {
			t.Errorf("covered divergence at %q = %q, want the consumer's bytes preserved %q", fixture.target, got, diverged)
		}
		if !strings.Contains(stdout, "preserved") || !strings.Contains(stdout, fixture.target) {
			t.Errorf("stdout %q does not mark %q as preserved", stdout, fixture.target)
		}
		if !strings.Contains(stdout, "@waiver:"+fixture.rule+":") {
			t.Errorf("stdout %q does not quote the token that accounted for the divergence (%q)", stdout, token)
		}
		// The contrast that makes the covered-plus-malformed arm falsifiable: a
		// preserve-and-drop implementation leaves stderr empty in BOTH.
		if strings.TrimSpace(stderr) != "" {
			t.Errorf("stderr %q is non-empty; there is no token hygiene problem in this file to warn about", stderr)
		}
		preservedStdout = stdout
	})

	t.Run("uncovered_divergence_is_marked_regenerated_over", func(t *testing.T) {
		fixture := stageDivergenceFixture(t)
		adoptThroughTheBinary(t, bin, fixture)

		writeDivergence(t, fixture, "# an ordinary comment carrying no token at all")

		stdout, stderr, code := applyOnce(t, bin, fixture)
		if code != 0 {
			t.Fatalf("re-apply over an uncovered divergence exited %d, want 0\nstdout: %s\nstderr: %s", code, stdout, stderr)
		}
		if got := readTarget(t, fixture); got != fixture.payload {
			t.Errorf("uncovered divergence at %q = %q, want the recipe's payload back %q", fixture.target, got, fixture.payload)
		}
		if !strings.Contains(strings.ToLower(stdout), "regenerated") {
			t.Errorf("stdout %q does not mark the clobber; an operator whose edit was overwritten must be able to see it", stdout)
		}
		if !strings.Contains(stdout, fixture.target) {
			t.Errorf("stdout %q does not name the clobbered target %q", stdout, fixture.target)
		}
		regeneratedStdout = stdout
	})

	t.Run("covered_plus_malformed_preserves_and_warns_on_stderr", func(t *testing.T) {
		fixture := stageDivergenceFixture(t)
		adoptThroughTheBinary(t, bin, fixture)

		diverged := writeDivergenceWithTwoTokens(t, fixture, validTokenLineFor(fixture.rule), malformedTokenLineFor(fixture.rule))

		stdout, stderr, code := applyOnce(t, bin, fixture)
		if code != 0 {
			t.Fatalf("re-apply over a covered divergence carrying a separate malformed token exited %d, want 0 — an unrelated typo must not revoke an accountable divergence\nstdout: %s\nstderr: %s", code, stdout, stderr)
		}
		if got := readTarget(t, fixture); got != diverged {
			t.Errorf("covered divergence at %q = %q, want the consumer's bytes preserved %q", fixture.target, got, diverged)
		}
		if !strings.Contains(stdout, "preserved") || !strings.Contains(stdout, fixture.target) {
			t.Errorf("stdout %q does not mark %q as preserved", stdout, fixture.target)
		}
		if !strings.Contains(stderr, divergenceReasonCode) {
			t.Errorf("stderr %q does not warn about the malformed token's reason code %q", stderr, divergenceReasonCode)
		}
		if !strings.Contains(stderr, "4") {
			t.Errorf("stderr %q does not name the malformed token's line", stderr)
		}
	})

	// The whole point of CLM-007 is that these three outcomes are now
	// DISTINGUISHABLE. Asserting the substrings above without this would be
	// satisfied by three identical reports.
	if cleanStdout == regeneratedStdout {
		t.Errorf("a clean apply and a clobber print identically (%q); the operator cannot tell them apart", cleanStdout)
	}
	if preservedStdout == regeneratedStdout {
		t.Errorf("a preserve and a clobber print identically (%q)", preservedStdout)
	}
	if cleanStdout == preservedStdout {
		t.Errorf("a clean apply and a preserve print identically (%q)", cleanStdout)
	}
}
