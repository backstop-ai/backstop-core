package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/gate"
)

// The baseline job's FIRST EVER run (main, 30398137055) failed with exactly this and
// nothing more:
//
//	baseline generate: gate reported a configuration error (exit 2); refusing to
//	write a baseline from a gate that produced no steps
//
// The cause was recoverable — the internal gate had run a step whose ConfigErr was
// set and whose violation named the missing Layer-0 tool — and the wrapper discarded
// it, justified by a comment asserting "result.Steps is empty" that was true for one
// config-error class and assumed for all. Third instance in this lane of the same
// defect family: the information existed and the surfacing did not.

// TestConfigErrorDiagnostic_SurfacesTheStepsOwnReason is the regression lock.
func TestConfigErrorDiagnostic_SurfacesTheStepsOwnReason(t *testing.T) {
	steps := []gate.StepResult{
		{StepName: "pack_engines", Status: "fail", ConfigErr: true,
			Violations: []gate.Violation{{Rule: "pack_engines", Severity: "error",
				Message: "engine tool \"golangci-lint\" not found on PATH"}}},
	}

	got := configErrorDiagnostic(steps)

	if got == "" {
		t.Fatal("a ConfigErr-carrying step produced NO diagnostic; the operator sees only 'exit 2' and " +
			"cannot tell a missing tool from a broken config — the failure this test exists to prevent")
	}
	for _, want := range []string{"pack_engines", "golangci-lint"} {
		if !strings.Contains(got, want) {
			t.Errorf("the diagnostic omits %q; got: %s", want, got)
		}
	}
}

// TestConfigErrorDiagnostic_EmptyWhenGenuinelyStepless keeps the original wording
// honest for the case the old comment actually described.
//
// A gate that returned exit 2 BEFORE building any step has nothing to add, and
// inventing a diagnostic there would be worse than saying less.
func TestConfigErrorDiagnostic_EmptyWhenGenuinelyStepless(t *testing.T) {
	if got := configErrorDiagnostic(nil); got != "" {
		t.Errorf("a step-less gate must add nothing, got: %s", got)
	}
	ok := []gate.StepResult{{StepName: "artifact_validation", Status: "pass"}}
	if got := configErrorDiagnostic(ok); got != "" {
		t.Errorf("no step carried ConfigErr, so there is nothing to surface; got: %s", got)
	}
}

// TestConfigErrorDiagnostic_NamesTheStepEvenWhenItReportedNoDetail covers the
// last fallback, and it is the one case where "no detail reported" is the honest answer.
//
// A step can set ConfigErr and carry neither a Reason nor a violation message. Rendering
// that as an empty string would collapse it into the stepless case above and lose the
// only fact still available — WHICH step config-errored. That name is what tells an
// operator where to look, so the fallback keeps it and admits the rest is missing rather
// than inventing a cause.
func TestConfigErrorDiagnostic_NamesTheStepEvenWhenItReportedNoDetail(t *testing.T) {
	silent := []gate.StepResult{{StepName: "pack_loading", Status: "fail", ConfigErr: true}}

	got := configErrorDiagnostic(silent)

	if !strings.Contains(got, "pack_loading") {
		t.Errorf("the diagnostic dropped the step name %q, which is the only fact a detail-less "+
			"ConfigErr step still carries; got: %q", "pack_loading", got)
	}
	if !strings.Contains(got, "no detail reported") {
		t.Errorf("a detail-less step must SAY it reported no detail — silence here reads as a "+
			"diagnostic that was never rendered rather than one that had nothing to render; got: %q", got)
	}
}

// TestRunBaselineGenerate_SurfacesTheGatesOwnConfigErrorAndRefusesToWrite drives the
// whole wrapper, not the helper.
//
// The tests above prove configErrorDiagnostic RENDERS a step's own report. They cannot
// prove runBaselineGenerate CALLS it — and a correct-but-uncalled renderer is exactly
// how the original defect shipped: the information existed, the surfacing did not. This
// test runs the command in-process against a project declaring a pack that is not
// installed, which is the cheapest real exit-2 (the gate refuses at pack_loading before
// any step executes), and asserts the pack's name reaches the returned error.
//
// The second assertion is the one that protects the artifact consumer: a config-erroring
// gate never really ran, so a baseline built from it would ratchet the project against
// nothing. Nothing may be written.
func TestRunBaselineGenerate_SurfacesTheGatesOwnConfigErrorAndRefusesToWrite(t *testing.T) {
	const missingPack = "acme/nonexistent-pack"

	project := t.TempDir()
	config := "project: probe\npacks:\n    " + missingPack + ": 1.0.0\n"
	if err := os.WriteFile(filepath.Join(project, "backstop.yml"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(project)

	err := runBaselineGenerate(nil, nil)

	if err == nil {
		t.Fatal("baseline generate accepted a gate that config-errored. The published artifact would " +
			"describe a gate that never ran, and every later comparison would ratchet against it")
	}
	for _, want := range []string{"exit 2", missingPack, "refusing to write a baseline"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the failure omits %q — the operator gets a bare exit code and no way to tell a "+
				"missing pack from a broken config; got: %v", want, err)
		}
	}
	if _, statErr := os.Stat(filepath.Join(project, ".backstop", "baseline.json")); statErr == nil {
		t.Error("a baseline was WRITTEN from a gate that reported a configuration error; it describes " +
			"no real run and would silently become the ratchet every later gate compares against")
	}
}

// TestWriteBaselineAtomically_RemovesTheTempFileWhenTheRenameFails locks the half of
// "atomically" that only matters when things go wrong.
//
// The write is a two-step dance — write <path>.tmp, rename it over <path> — and the
// rename is what makes it atomic. When the rename fails the temp file is already on
// disk holding a complete baseline under a name nothing will ever read. Leaving it
// there litters .backstop with a file that looks like a baseline to a human and is
// invisible to the gate, so the failure path must clean up after itself.
func TestWriteBaselineAtomically_RemovesTheTempFileWhenTheRenameFails(t *testing.T) {
	dir := t.TempDir()
	// A NON-EMPTY directory standing where the baseline file belongs: rename refuses to
	// replace it, which is the failure this test needs and cannot get from a valid path.
	target := filepath.Join(dir, "baseline.json")
	if err := os.MkdirAll(filepath.Join(target, "occupied"), 0o755); err != nil {
		t.Fatal(err)
	}

	err := writeBaselineAtomically(target, []byte(`{"schema_version":"baseline/v1"}`))

	if err == nil {
		t.Fatal("writeBaselineAtomically reported success while the rename could not have happened; a " +
			"caller would believe the baseline is durable when nothing was published")
	}
	if _, statErr := os.Stat(target + ".tmp"); statErr == nil {
		t.Error("the temp file survived the failed rename. It holds a complete baseline under a name " +
			"the gate never reads, so .backstop keeps a file that looks published and is not")
	}
}

// TestWriteBaselineAtomically_FailsWhenTheDirectoryCannotBeCreated covers the first
// guard, which runs before any bytes exist.
//
// It is a distinct failure from the rename case and must stay distinct: nothing has been
// written yet, so there is nothing to clean up and the only requirement is that the
// caller is told. A silent success here would leave the CI job believing it published a
// baseline into a path that cannot hold one.
func TestWriteBaselineAtomically_FailsWhenTheDirectoryCannotBeCreated(t *testing.T) {
	dir := t.TempDir()
	// A REGULAR FILE where a parent directory would have to be — MkdirAll cannot descend
	// through it.
	blocker := filepath.Join(dir, "not-a-directory")
	if err := os.WriteFile(blocker, []byte("occupied"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := writeBaselineAtomically(filepath.Join(blocker, ".backstop", "baseline.json"), []byte("{}"))

	if err == nil {
		t.Fatal("writeBaselineAtomically reported success for a path whose parent directory cannot " +
			"exist; the caller would report a published baseline that was never written")
	}
}
