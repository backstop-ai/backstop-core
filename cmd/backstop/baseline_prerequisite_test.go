package main

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

// ratchetTestFile is the file whose baseline read these assertions govern. It is
// named relative to this package's directory, which is the working directory of
// a `go test ./cmd/backstop/` run.
const ratchetTestFile = "bun_ratchet_flip_test.go"

// TestCommittedBaselineAbsence_FailsWithActionableRemedy pins the MESSAGE, which
// is the whole deliverable of ISSUE-176's second fix. (CLM-005)
//
// The three ratchet tests read `.backstop/baseline.json` — a GENERATED,
// gitignored artifact CI publishes, never committed source. On any machine that
// has not fetched one they failed with a bare
//
//	read committed baseline: open .../.backstop/baseline.json: no such file or directory
//
// which is loud but names no remedy. The failure stays a FATAL — the ratchet
// property is unchanged — and only the message improves.
func TestCommittedBaselineAbsence_FailsWithActionableRemedy(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "no-such-baseline.json")

	raw, err := readGeneratedBaseline(missing)
	if err == nil {
		t.Fatalf("readGeneratedBaseline(%q) returned %d bytes and no error, but nothing was ever written there",
			missing, len(raw))
	}

	text := err.Error()
	for _, want := range []string{
		missing,                  // self-locating: name the file it could not read
		"backstop baseline pull", // the CLI remedy
		"make baseline",          // the opt-in make target that wraps it
	} {
		if !strings.Contains(text, want) {
			t.Errorf("the absent-baseline error does not name %q. It is the only surface a developer running "+
				"`make test` sees, so it has to carry the remedy:\n%s", want, text)
		}
	}

	// NEVER a skip. Skipping would quietly retire the ratchet property these
	// three tests exist to guard, which is strictly worse than a loud failure —
	// so the word must not appear even as a suggestion.
	if strings.Contains(strings.ToLower(text), "skip") {
		t.Errorf("the absent-baseline error mentions skipping. The ratchet must FAIL on a missing baseline, "+
			"never be skipped past:\n%s", text)
	}
}

// TestRatchetBaselineRead_NeverSkipsAndNeverPullsImplicitly is a source sweep
// over the ratchet test file. (CLM-005, CLM-006)
//
// Two properties, neither of which any single test can observe at runtime: the
// ratchet never skips, and the test path never acquires a network dependency.
// The second is what makes the OPT-IN direction real — `go test ./...` must not
// silently shell a `backstop baseline pull` on a fresh clone.
func TestRatchetBaselineRead_NeverSkipsAndNeverPullsImplicitly(t *testing.T) {
	source := readFileStr(t, ratchetTestFile)
	lines := strings.Split(source, "\n")

	// LEG 1: no skipping, whole-file. Neither spelling belongs in this file at
	// all, so the coarse scope is the correct one here.
	for _, forbidden := range []string{"t.Skip", "t.Skipf"} {
		for i, line := range lines {
			if strings.Contains(line, forbidden) {
				t.Errorf("%s:%d calls %s. A missing baseline must FAIL the ratchet, not retire it silently:\n%s",
					ratchetTestFile, i+1, forbidden, strings.TrimSpace(line))
			}
		}
	}

	// LEG 2: the baseline read goes through the remedy-naming helper.
	//
	// ★ THE PREDICATE IS LINE-SCOPED, DELIBERATELY. A whole-file
	// `!contains("os.ReadFile")` assertion would be PERMANENTLY unsatisfiable:
	// this file legitimately contains os.ReadFile twice — inside the helper
	// itself, and at the pre-existing pillarASitesClean read that walks SOURCE
	// files and has nothing to do with the baseline. What is worth forbidding is
	// re-inlining the BARE baseline read, and that is exactly one line: a read
	// call and the baseline path together. It is satisfiable precisely because
	// the helper takes the path as a PARAMETER. Do not loosen this — narrow it
	// if it ever fires for the wrong reason.
	for i, line := range lines {
		if strings.Contains(line, "os.ReadFile") && strings.Contains(line, "baseline.json") {
			t.Errorf("%s:%d reads the baseline path directly:\n%s\nRoute it through readGeneratedBaseline, "+
				"which is the one place that names the remedy for an absent, CI-generated artifact",
				ratchetTestFile, i+1, strings.TrimSpace(line))
		}
	}

	// LEG 3: the test path never shells out and never opens a socket.
	//
	// ★ THE MECHANISM IS FORBIDDEN, NOT THE LITERAL. A leg reading "the file
	// contains no `baseline pull`" would be inverted: TestCommittedBaselineAbsence
	// REQUIRES the helper's error text to name `backstop baseline pull` verbatim,
	// so such a leg would be green today and red the moment the fix lands. The
	// remedy text is a STRING LITERAL and must stay one; imports and call sites
	// are what distinguish EXECUTING a command from NAMING it.
	fileSet := token.NewFileSet()
	parsed, err := parser.ParseFile(fileSet, ratchetTestFile, source, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("parse %s: %v", ratchetTestFile, err)
	}
	if len(parsed.Imports) == 0 {
		t.Fatalf("%s parsed with zero imports — the import assertion below would be vacuous", ratchetTestFile)
	}
	for _, imported := range parsed.Imports {
		path := strings.Trim(imported.Path.Value, `"`)
		for _, forbidden := range []string{"os/exec", "net/http"} {
			if path == forbidden {
				t.Errorf("%s imports %q. The ratchet's test path must never hit the network or shell a command: "+
					"the remedy is OPT-IN (`make baseline`), so that a developer's `go test ./...` cannot silently "+
					"acquire a network dependency (ISSUE-176)", ratchetTestFile, forbidden)
			}
		}
	}
	for i, line := range lines {
		for _, forbidden := range []string{"exec.Command", "exec.CommandContext"} {
			if strings.Contains(line, forbidden) {
				t.Errorf("%s:%d calls %s:\n%s\nNaming `backstop baseline pull` in an error message is the fix; "+
					"RUNNING it from the test path is the thing ISSUE-176 rules out",
					ratchetTestFile, i+1, forbidden, strings.TrimSpace(line))
			}
		}
	}
}
