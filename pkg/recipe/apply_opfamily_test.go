package recipe

import (
	"strings"
	"testing"
)

// TestApply_InsertOp_AtAnchor proves an insert op places its declared snippet at the
// declared ANCHOR (CLM-011). The seed carries a trailing marker after the anchor, so
// the assertion pins the snippet BETWEEN them: an applier that appended at EOF — the
// most likely wrong-but-passing implementation — lands after the marker and fails
// both the byte comparison and the position check.
func TestApply_InsertOp_AtAnchor(t *testing.T) {
	const insertRecipe = `
kind: scaffolding
version: 1.0.0
ops:
  - id: op-seed
    kind: create
    target: generated/anchored.txt
    payload: seed.txt
  - id: op-insert
    kind: insert
    target: generated/anchored.txt
    anchor: "# SLOTS"
    snippet: "\nspliced-at-anchor"
    manual: "Add the spliced line directly beneath the SLOTS marker."
`

	recipeDir := t.TempDir()
	projectRoot := t.TempDir()

	resolved := resolvedFromManifest(t, recipeDir, insertRecipe)
	createOp := resolved.Manifest.Ops[0]
	insertOp := resolved.Manifest.Ops[1]
	writeUnder(t, recipeDir, createOp.Payload, seedPayload)

	if _, err := Apply(resolved, ApplyOptions{Mode: ModeDirect, ProjectRoot: projectRoot}); err != nil {
		t.Fatalf("Apply: unexpected error: %v", err)
	}

	got := snapshotTree(t, projectRoot)[insertOp.Target]
	want := "# SLOTS\nspliced-at-anchor\n# END\n"
	if got != want {
		t.Errorf("inserted content =\n%q\nwant\n%q", got, want)
	}

	anchorAt := strings.Index(got, insertOp.Anchor)
	snippetAt := strings.Index(got, strings.TrimSpace(insertOp.Snippet))
	endAt := strings.Index(got, "# END")
	if anchorAt < 0 || snippetAt < 0 || endAt < 0 {
		t.Fatalf("anchor/snippet/trailing marker not all present in %q", got)
	}
	if snippetAt < anchorAt {
		t.Errorf("snippet at %d precedes the anchor at %d; the snippet must land AT the declared anchor", snippetAt, anchorAt)
	}
	if snippetAt > endAt {
		t.Errorf("snippet at %d follows the trailing marker at %d; the snippet was appended at EOF instead of at the anchor", snippetAt, endAt)
	}
}

// TestApply_StepOp_RecognizedAndSequenced proves a `step` op is a RECOGNIZED fifth
// family that holds its sequence position (CLM-028). The recipe is
// [create, step, insert]: the step must neither fail as an unknown kind nor abort
// the run, and the file ops on BOTH sides of it must still execute in declared
// order — the insert can only succeed if the create ran first, and the content
// proves both landed. The step itself contributes no entry to Written.
func TestApply_StepOp_RecognizedAndSequenced(t *testing.T) {
	const sequencedRecipe = `
kind: scaffolding
version: 1.0.0
ops:
  - id: op-create
    kind: create
    target: generated/sequenced.txt
    payload: seed.txt
  - id: op-step
    kind: step
  - id: op-insert
    kind: insert
    target: generated/sequenced.txt
    anchor: "# SLOTS"
    snippet: "\nafter-the-step"
    manual: "Add the line directly beneath the SLOTS marker."
`

	recipeDir := t.TempDir()
	projectRoot := t.TempDir()

	resolved := resolvedFromManifest(t, recipeDir, sequencedRecipe)
	createOp := resolved.Manifest.Ops[0]
	writeUnder(t, recipeDir, createOp.Payload, seedPayload)

	result, err := Apply(resolved, ApplyOptions{Mode: ModeDirect, ProjectRoot: projectRoot})
	if err != nil {
		t.Fatalf("Apply over [create, step, insert]: unexpected error: %v", err)
	}

	got := snapshotTree(t, projectRoot)[createOp.Target]
	want := "# SLOTS\nafter-the-step\n# END\n"
	if got != want {
		t.Errorf("sequenced content =\n%q\nwant\n%q — the ops on both sides of the step must run in declared order", got, want)
	}
	if len(result.Written) != 1 || result.Written[0] != createOp.Target {
		t.Errorf("result.Written = %v, want exactly [%q] — the step occupies its sequence position without writing", result.Written, createOp.Target)
	}
}

// TestApply_StepOp_NotExecutedReservedSeam proves the applier does NOT execute a
// step op (CLM-029) — the executor is BUNDLE-019's. The step declares a would-be
// side effect (a target AND a payload present in the recipe directory), so an
// applier that ran it, or that quietly treated it as a create, WOULD produce the
// file. The whole project root is walked: the step's declared target must not exist
// anywhere, and the result must account for the run as the two file ops only.
func TestApply_StepOp_NotExecutedReservedSeam(t *testing.T) {
	const reservedStepRecipe = `
kind: scaffolding
version: 1.0.0
ops:
  - id: op-before
    kind: create
    target: generated/before.txt
    payload: before.txt
  - id: op-step
    kind: step
    target: generated/step-side-effect.txt
    payload: step-side-effect.txt
  - id: op-after
    kind: create
    target: generated/after.txt
    payload: after.txt
`

	recipeDir := t.TempDir()
	projectRoot := t.TempDir()

	resolved := resolvedFromManifest(t, recipeDir, reservedStepRecipe)
	beforeOp := resolved.Manifest.Ops[0]
	stepOp := resolved.Manifest.Ops[1]
	afterOp := resolved.Manifest.Ops[2]
	writeUnder(t, recipeDir, beforeOp.Payload, "before body\n")
	writeUnder(t, recipeDir, stepOp.Payload, "the side effect a step executor would produce\n")
	writeUnder(t, recipeDir, afterOp.Payload, "after body\n")

	result, err := Apply(resolved, ApplyOptions{Mode: ModeDirect, ProjectRoot: projectRoot})
	if err != nil {
		t.Fatalf("Apply: unexpected error: %v", err)
	}

	tree := snapshotTree(t, projectRoot)
	if _, executed := tree[stepOp.Target]; executed {
		t.Errorf("the step op's declared side effect %q exists; the step must be reserved, not executed", stepOp.Target)
	}
	if len(tree) != 2 {
		t.Errorf("project root holds %v, want only the two file ops' targets", pathSet(tree))
	}

	want := []string{beforeOp.Target, afterOp.Target}
	if len(result.Written) != len(want) {
		t.Fatalf("result.Written = %v, want %v — the step contributes nothing", result.Written, want)
	}
	for i, target := range want {
		if result.Written[i] != target {
			t.Errorf("result.Written[%d] = %q, want %q", i, result.Written[i], target)
		}
	}
}

// TestApply_UnknownOpKindFailsLoud proves the op-family allowlist is CLOSED
// (CLM-030): a kind outside {create, merge, transform, insert, step} is a fail-loud
// error naming the kind and the op id, never a silent skip. The result is asserted
// too — an implementation that logged the unknown kind and reported the run as
// successful would pass an error-only assertion.
func TestApply_UnknownOpKindFailsLoud(t *testing.T) {
	const unknownKindRecipe = `
kind: scaffolding
version: 1.0.0
ops:
  - id: op-known
    kind: create
    target: generated/known.txt
    payload: known.txt
  - id: op-bogus
    kind: conjure
    target: generated/bogus.txt
`

	recipeDir := t.TempDir()
	projectRoot := t.TempDir()

	resolved := resolvedFromManifest(t, recipeDir, unknownKindRecipe)
	knownOp := resolved.Manifest.Ops[0]
	bogusOp := resolved.Manifest.Ops[1]
	writeUnder(t, recipeDir, knownOp.Payload, "known body\n")

	result, err := Apply(resolved, ApplyOptions{Mode: ModeDirect, ProjectRoot: projectRoot})
	if err == nil {
		t.Fatalf("Apply over an op of kind %q: expected a fail-loud error, got nil", bogusOp.Kind)
	}
	if !strings.Contains(err.Error(), bogusOp.Kind) {
		t.Errorf("error %q does not name the offending kind %q", err, bogusOp.Kind)
	}
	if !strings.Contains(err.Error(), bogusOp.ID) {
		t.Errorf("error %q does not name the offending op id %q", err, bogusOp.ID)
	}
	if len(result.Written) != 0 || len(result.Preserved) != 0 {
		t.Errorf("result = %+v, want no partial-success verdict alongside the error", result)
	}
}
