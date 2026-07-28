package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/gate"
)

// The founder's file-granularity ruling, locked at the EXIT-CODE level.
//
// The unit-level ratchet is already covered by pkg/gate/baseline_ratchet_test.go.
// What that does not cover is the whole path a CI run actually takes: real base
// resolution, real scope computation, a real baseline artifact on disk, and the
// exit code a workflow blocks on. These three tests walk that path.
//
// THEY ARE EXPECTED TO PASS ON FIRST RUN. They are REGRESSION LOCKS over behaviour
// the strict file-level ratchet (ISSUE-050, pkg/gate/baseline.go scopeTouches)
// already implements — the founder asked for a falsifier that catches the day
// someone loosens it, and a test that is green on arrival is still doing that job.
// There is no manufactured red phase here, and claiming one would be a lie about
// what happened.

// ratchetProjectManifest is the same minimal rule-fed pack the diff-scope tests
// use, so pack_engines is a CAPABLE dimension and its step actually runs. The
// findings themselves come from the dispatch seam, not from a live engine.
const ratchetProjectManifest = `name: org/pack
version: 1.0.0
language: go
archetype: enforcement
description: Pack with a rule-fed semgrep engine
content:
  ruleset:
    version: 1.0.0
    rules:
      - id: no-foo
        standard: standards/go/no-foo.standard.md
        rule_path: semgrep/rule.yml
        risk_class: security
        engine: semgrep
        claims:
          - id: c-no-foo
            text: No foo.
            fixtures:
              positive:
                - fixtures/positive.go
              negative:
                - fixtures/negative.go
`

// newBaselineRatchetProject builds a git-backed project with a pack, a file the
// second commit TOUCHES, and a file it leaves alone. It returns the project root
// and the base SHA to gate against.
//
// The tree is left fully committed, which is the CI condition: `--base <sha>`
// resolves a non-empty scope where bare diff mode would resolve none.
func newBaselineRatchetProject(t *testing.T) (root, baseSHA, touched, untouched string) {
	t.Helper()

	root = t.TempDir()
	writeProjectFile(t, root, "backstop.yml", "project: p\nlanguage: go\npacks:\n  org/pack: \"1.0.0\"\n")

	packRoot := filepath.Join(root, ".backstop", "packs", "org", "pack")
	if err := os.MkdirAll(filepath.Join(packRoot, "semgrep"), 0o755); err != nil {
		t.Fatalf("create pack dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(packRoot, "semgrep", "rule.yml"), []byte("rules: []\n"), 0o644); err != nil {
		t.Fatalf("write rule file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(packRoot, "pack.yml"), []byte(ratchetProjectManifest), 0o644); err != nil {
		t.Fatalf("write pack manifest: %v", err)
	}

	// A lock file is mandatory once backstop.yml declares a pack: pack_lock_verification
	// is the FIRST gate step and a missing lock is an exit-2 config error, which would
	// make every assertion below read a config failure as a violation. The entry is
	// source_type "local" because VerifyLock skips local packs rather than hashing
	// them — this fixture is about the ratchet, not about distribution integrity.
	writeProjectFile(t, root, "backstop.lock", "packs:\n"+
		"  org/pack:\n"+
		"    name: org/pack\n"+
		"    version: \"1.0.0\"\n"+
		"    git_ref: null\n"+
		"    content_hash: \"\"\n"+
		"    source_type: local\n"+
		"    install_date: \"2026-07-28T00:00:00Z\"\n"+
		"    local_path: .backstop/packs/org/pack\n")

	// test_verification reads the artifact corpus and treats an ABSENT specs
	// directory as a hard error, so the fixture needs one. Empty is the correct
	// content: these tests are about the ratchet, and a spec here would drag
	// mandated-test extraction into an experiment that has nothing to do with it.
	if err := os.MkdirAll(filepath.Join(root, "specs"), 0o755); err != nil {
		t.Fatalf("create specs dir: %v", err)
	}
	writeProjectFile(t, root, "specs/.gitkeep", "")

	touched, untouched = "touched.go", "untouched.go"
	writeProjectFile(t, root, touched, "package p\n")
	writeProjectFile(t, root, untouched, "package p\n")

	gitInProject(t, root, "init")
	gitInProject(t, root, "config", "user.email", "test@example.com")
	gitInProject(t, root, "config", "user.name", "Test User")
	gitInProject(t, root, "checkout", "-b", "main")
	gitInProject(t, root, "add", ".")
	gitInProject(t, root, "commit", "-m", "commit A")
	baseSHA = gitInProject(t, root, "rev-parse", "HEAD")

	// Commit B touches exactly one of the two files. That asymmetry is the whole
	// experiment: one file enters the diff scope, the other does not.
	writeProjectFile(t, root, touched, "package p\n\nfunc touchedByThisChange() {}\n")
	gitInProject(t, root, "add", ".")
	gitInProject(t, root, "commit", "-m", "commit B")

	return root, baseSHA, touched, untouched
}

// writeRatchetBaseline plants a baseline artifact recording violations as
// pre-existing, grandfathered debt.
func writeRatchetBaseline(t *testing.T, root string, violations ...gate.Violation) {
	t.Helper()

	artifact := gate.BaselineArtifact{SchemaVersion: "1", Violations: violations}
	data, err := json.MarshalIndent(artifact, "", "  ")
	if err != nil {
		t.Fatalf("marshal baseline: %v", err)
	}
	path := filepath.Join(root, ".backstop", "baseline.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create baseline dir: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write baseline: %v", err)
	}
}

// TestGate_ExplicitBase_PreExistingFindingInTouchedFileReds is the exact defense
// against "that was there before, my changes are clean".
//
// The finding IS in the baseline, so under a net-new-only ratchet it would be
// grandfathered and the run would go green. The diff touches its file, which
// REVOKES the grandfather — every pre-existing finding in a touched file has to be
// resolved, fixed or waived, not merely kept net-new-clean. (CLM-024)
func TestGate_ExplicitBase_PreExistingFindingInTouchedFileReds(t *testing.T) {
	root, baseSHA, touched, _ := newBaselineRatchetProject(t)

	preExisting := gate.Violation{
		Rule:       "org/pack/no-foo",
		File:       touched,
		Message:    "foo predates this change",
		Severity:   "error",
		SourcePack: "org/pack",
	}
	setDispatchSeam(t, []gate.Violation{preExisting})
	writeRatchetBaseline(t, root, preExisting)

	stdout, stderr, exitCode := runGateCommand(t, root, "--base", baseSHA)
	if exitCode == ExitPass {
		t.Fatalf("a baselined finding in a TOUCHED file exited %d (pass). Its grandfather must be revoked the "+
			"moment the file enters the diff — this is the ratchet loosening the founder asked to be caught.\n"+
			"stdout:\n%s\nstderr:\n%s", exitCode, stdout, stderr)
	}
	if !strings.Contains(stdout+stderr, touched) {
		t.Errorf("the run redded but never named %q, so it is not clear the PRE-EXISTING finding is what "+
			"failed it.\nstdout:\n%s\nstderr:\n%s", touched, stdout, stderr)
	}
}

// TestGate_ExplicitBase_UntouchedFileFindingStaysGrandfathered is the mirror, and
// without it the test above is satisfied by any gate that ignores the baseline
// entirely and reds on every finding.
//
// THE FINDING IS IDENTICAL to the one in the test above except for its FILE. That
// is what makes the pair a falsifier rather than two unrelated assertions: the only
// variable between "reds" and "grandfathered" is whether the diff touched the file.
// A gate that ignored the baseline and redded on every finding fails here; a gate
// that grandfathered unconditionally fails above.
//
// The violation is deliberately NOT ProjectWide. Exempt/ProjectWide findings —
// go-build breaks — are RETAINED on untouched files by design (CLM-005), because an
// unchanged-file build break must still fail the gate. Using one here would assert
// the opposite of an existing, intended guarantee. (CLM-024)
func TestGate_ExplicitBase_UntouchedFileFindingStaysGrandfathered(t *testing.T) {
	root, baseSHA, touched, untouched := newBaselineRatchetProject(t)

	debt := gate.Violation{
		Rule:       "org/pack/no-foo",
		File:       untouched,
		Message:    "foo predates this change",
		Severity:   "error",
		SourcePack: "org/pack",
	}
	setDispatchSeam(t, []gate.Violation{debt})
	writeRatchetBaseline(t, root, debt)

	stdout, stderr, exitCode := runGateCommand(t, root, "--base", baseSHA)
	if exitCode != ExitPass {
		t.Fatalf("a baselined finding on an UNTOUCHED file exited %d, want pass — the grandfather holds while "+
			"nobody touches its file, or the ratchet is unusable on any repo with existing debt.\n"+
			"stdout:\n%s\nstderr:\n%s", exitCode, stdout, stderr)
	}
	if strings.Contains(stdout, untouched) && !strings.Contains(stdout, touched) {
		t.Errorf("the run passed but reported the untouched-file finding as live; it should be grandfathered "+
			"out:\n%s", stdout)
	}
}

// TestGate_ExplicitBase_CleanDiffIsGreenDespiteRepoDebt — "green if the diff is
// green" (CLM-025).
//
// This is the property that makes the file-granularity ruling livable. Without it,
// the strict ratchet would mean nobody can land anything until the whole repo is
// clean, and the pressure would go straight into loosening the rule the test above
// exists to protect.
func TestGate_ExplicitBase_CleanDiffIsGreenDespiteRepoDebt(t *testing.T) {
	root, baseSHA, _, untouched := newBaselineRatchetProject(t)

	// Debt elsewhere in the repo, on a file this diff does not touch. The touched
	// file produces NO finding at all.
	debt := gate.Violation{
		Rule:       "org/pack/no-foo",
		File:       untouched,
		Message:    "repo debt, unrelated to this change",
		Severity:   "error",
		SourcePack: "org/pack",
	}
	setDispatchSeam(t, []gate.Violation{debt})
	writeRatchetBaseline(t, root, debt)

	stdout, stderr, exitCode := runGateCommand(t, root, "--base", baseSHA)
	if exitCode != ExitPass {
		t.Fatalf("a diff touching only clean files exited %d while the repo carried baselined debt elsewhere; "+
			"want pass.\nstdout:\n%s\nstderr:\n%s", exitCode, stdout, stderr)
	}
}
