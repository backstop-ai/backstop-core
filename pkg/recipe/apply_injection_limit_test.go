package recipe

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The INJECTION LIMIT (REQ-011): where generation cannot reach, apply fails loud and
// hands the consumer the recipe's OWN instruction. Core never composes that text —
// an actionable "wire it in by hand like THIS" is inherently language- and
// framework-specific, so the recipe supplies it as DATA and the applier emits it
// VERBATIM. The declared strings below are deliberately distinctive and
// multi-clause: a paraphrase, a re-wrap, or a synthesized instruction cannot contain
// them, so the verbatim property is falsifiable rather than decorative.

const declaredTransformManual = "HAND-STEP ALPHA-7: open the declared target yourself, re-point every superseded call site, and re-run the check -- the recipe cannot reach this site."

const declaredInsertManual = "HAND-STEP BRAVO-7: paste the declared snippet where the anchor would have been, keeping the surrounding block intact -- the recipe cannot find the anchor."

// unreachableTransformRecipe targets a path under a directory the project does not
// have, so both the file and its parent are absent.
const unreachableTransformRecipe = `
kind: implementing
version: 1.0.0
transform_rules:
  - rules/rewrite-rule.yml
ops:
  - id: op-unreachable-transform
    kind: transform
    target: unbuilt/wiring/site.conf
    rule: rules/rewrite-rule.yml
    manual: "HAND-STEP ALPHA-7: open the declared target yourself, re-point every superseded call site, and re-run the check -- the recipe cannot reach this site."
`

// presentTargetTransformRecipe targets a file that DOES exist, so the unreachability
// can only come from the engine reporting that the declared rule matched nothing.
const presentTargetTransformRecipe = `
kind: implementing
version: 1.0.0
transform_rules:
  - rules/rewrite-rule.yml
ops:
  - id: op-unmatched-transform
    kind: transform
    target: existing/site.conf
    rule: rules/rewrite-rule.yml
    manual: "HAND-STEP ALPHA-7: open the declared target yourself, re-point every superseded call site, and re-run the check -- the recipe cannot reach this site."
`

// missingAnchorInsertRecipe declares an anchor the seeded target does not contain —
// the insert analog of an unreachable transform.
const missingAnchorInsertRecipe = `
kind: scaffolding
version: 1.0.0
ops:
  - id: op-missing-anchor
    kind: insert
    target: existing/site.conf
    anchor: "# THE ANCHOR THE TARGET DOES NOT CARRY"
    snippet: "\nspliced"
    manual: "HAND-STEP BRAVO-7: paste the declared snippet where the anchor would have been, keeping the surrounding block intact -- the recipe cannot find the anchor."
`

// unreachableSiteContent is the seeded target's body for the cases that need the file
// to exist. It carries no anchor and no match for the declared rule.
const unreachableSiteContent = "an existing consumer file with neither the declared anchor nor a match\n"

// decoyContent seeds files the applier must leave alone, so "nothing was written"
// is distinguishable from "the project root was empty anyway".
const decoyContent = "a neighbouring consumer file the applier has no business touching\n"

// refusingDispatch reports the engine finding nothing to rewrite. It records whether
// it ran, so a case that should never reach the engine can assert that it did not.
func refusingDispatch(ran *bool) TransformDispatch {
	return func(rule string, target string) error {
		*ran = true
		return errors.New("the declared rule matched nothing in the target")
	}
}

// TestApply_TransformUnreachable_MessageEqualsDeclaredManualVerbatim proves the
// fail-loud message carries the op's DECLARED manual text verbatim plus a locator
// (CLM-050), in both ways a transform goes unreachable: the declared target is not
// there at all, and the engine reports the declared rule matched nothing.
//
// The expected text is read off the PARSED manifest rather than re-typed, so the
// assertion is against what the recipe actually declared.
func TestApply_TransformUnreachable_MessageEqualsDeclaredManualVerbatim(t *testing.T) {
	cases := []struct {
		name         string
		recipeYAML   string
		seedTarget   bool
		wantDispatch bool
	}{
		{name: "declared_target_absent", recipeYAML: unreachableTransformRecipe, seedTarget: false, wantDispatch: false},
		{name: "declared_rule_matched_nothing", recipeYAML: presentTargetTransformRecipe, seedTarget: true, wantDispatch: true},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			projectRoot := t.TempDir()
			resolved := resolvedFromManifest(t, t.TempDir(), testCase.recipeYAML)
			transformOp := resolved.Manifest.Ops[0]
			if testCase.seedTarget {
				writeUnder(t, projectRoot, transformOp.Target, unreachableSiteContent)
			}

			var dispatched bool
			_, err := Apply(resolved, ApplyOptions{
				Mode:        ModeDirect,
				ProjectRoot: projectRoot,
				Dispatch:    refusingDispatch(&dispatched),
			})
			if err == nil {
				t.Fatalf("Apply over an unreachable transform: expected a fail-loud error, got nil")
			}
			if dispatched != testCase.wantDispatch {
				t.Errorf("dispatch ran = %v, want %v", dispatched, testCase.wantDispatch)
			}

			if transformOp.Manual != declaredTransformManual {
				t.Fatalf("the parsed manual %q is not the declared text; the fixture no longer proves the verbatim property", transformOp.Manual)
			}
			if !strings.Contains(err.Error(), transformOp.Manual) {
				t.Errorf("error\n%q\ndoes not carry the DECLARED manual instruction VERBATIM:\n%q", err, transformOp.Manual)
			}
			if !strings.Contains(err.Error(), transformOp.ID) {
				t.Errorf("error %q does not locate the offending op %q", err, transformOp.ID)
			}
			if !strings.Contains(err.Error(), transformOp.Target) {
				t.Errorf("error %q does not name the intended target %q", err, transformOp.Target)
			}
		})
	}
}

// TestApply_TransformUnreachable_NeverSilentSkip proves an unreachable transform is a
// FAILED apply, not a logged inconvenience (CLM-051): a non-nil error AND the zero
// verdict. An implementation that reported the problem and carried on would return a
// populated result beside a nil error and fail both halves.
func TestApply_TransformUnreachable_NeverSilentSkip(t *testing.T) {
	// The recipe creates a file BEFORE the unreachable transform, so an applier that
	// continued past the failure would have a genuine success to report.
	const mixedRecipe = `
kind: implementing
version: 1.0.0
transform_rules:
  - rules/rewrite-rule.yml
ops:
  - id: op-precedes
    kind: create
    target: generated/first.conf
    payload: body.txt
  - id: op-unreachable-transform
    kind: transform
    target: unbuilt/wiring/site.conf
    rule: rules/rewrite-rule.yml
    manual: "HAND-STEP ALPHA-7: open the declared target yourself, re-point every superseded call site, and re-run the check -- the recipe cannot reach this site."
`

	recipeDir := t.TempDir()
	projectRoot := t.TempDir()

	resolved := resolvedFromManifest(t, recipeDir, mixedRecipe)
	writeUnder(t, recipeDir, resolved.Manifest.Ops[0].Payload, "body\n")

	var dispatched bool
	result, err := Apply(resolved, ApplyOptions{
		Mode:        ModeDirect,
		ProjectRoot: projectRoot,
		Dispatch:    refusingDispatch(&dispatched),
	})
	if err == nil {
		t.Fatalf("Apply over an unreachable transform: expected a fail-loud error, got nil (result %+v)", result)
	}
	if len(result.Written) != 0 || len(result.Preserved) != 0 {
		t.Errorf("result = %+v, want the ZERO verdict; an unreachable transform must not report a successful apply", result)
	}
}

// TestApply_TransformUnreachable_NeverGuessesSite proves the applier contributes NO
// fallback location (CLM-052): the whole project root is walked before and after, and
// every path and byte must be unchanged — no nearest-match write, no created parent
// directory for the absent target, no marker file.
func TestApply_TransformUnreachable_NeverGuessesSite(t *testing.T) {
	projectRoot := t.TempDir()
	resolved := resolvedFromManifest(t, t.TempDir(), unreachableTransformRecipe)
	transformOp := resolved.Manifest.Ops[0]

	// Decoys a guessing implementation might plausibly land on: a file with the
	// declared target's base name in another directory, and an unrelated neighbour.
	writeUnder(t, projectRoot, filepath.Join("elsewhere", filepath.Base(transformOp.Target)), decoyContent)
	writeUnder(t, projectRoot, "neighbour.conf", decoyContent)
	before := snapshotTree(t, projectRoot)

	var dispatched bool
	if _, err := Apply(resolved, ApplyOptions{
		Mode:        ModeDirect,
		ProjectRoot: projectRoot,
		Dispatch:    refusingDispatch(&dispatched),
	}); err == nil {
		t.Fatalf("Apply over an unreachable transform: expected a fail-loud error, got nil")
	}

	after := snapshotTree(t, projectRoot)
	if len(after) != len(before) {
		t.Fatalf("project file set changed: before %v, after %v", pathSet(before), pathSet(after))
	}
	for path, content := range before {
		if after[path] != content {
			t.Errorf("file %q changed on the unreachable op's account: before %q, after %q", path, content, after[path])
		}
	}

	parent := filepath.Join(projectRoot, filepath.Dir(transformOp.Target))
	if _, err := os.Stat(parent); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the absent target's parent directory %q exists (stat error %v); an unreachable transform creates nothing", parent, err)
	}
}

// TestApply_InsertMissingAnchor_MessageEqualsDeclaredManualVerbatim proves the insert
// analog of the injection limit (CLM-053): a declared anchor the target does not
// carry fails with the op's DECLARED manual text verbatim plus a locator, and the
// target is left byte-identical — never appended to at EOF as a consolation write.
func TestApply_InsertMissingAnchor_MessageEqualsDeclaredManualVerbatim(t *testing.T) {
	projectRoot := t.TempDir()
	resolved := resolvedFromManifest(t, t.TempDir(), missingAnchorInsertRecipe)
	insertOp := resolved.Manifest.Ops[0]
	writeUnder(t, projectRoot, insertOp.Target, unreachableSiteContent)

	result, err := Apply(resolved, ApplyOptions{Mode: ModeDirect, ProjectRoot: projectRoot})
	if err == nil {
		t.Fatalf("Apply over an insert whose anchor is absent: expected a fail-loud error, got nil (result %+v)", result)
	}

	if insertOp.Manual != declaredInsertManual {
		t.Fatalf("the parsed manual %q is not the declared text; the fixture no longer proves the verbatim property", insertOp.Manual)
	}
	if !strings.Contains(err.Error(), insertOp.Manual) {
		t.Errorf("error\n%q\ndoes not carry the DECLARED manual instruction VERBATIM:\n%q", err, insertOp.Manual)
	}
	if !strings.Contains(err.Error(), insertOp.ID) {
		t.Errorf("error %q does not locate the offending op %q", err, insertOp.ID)
	}
	if !strings.Contains(err.Error(), insertOp.Target) {
		t.Errorf("error %q does not name the intended target %q", err, insertOp.Target)
	}

	if got := snapshotTree(t, projectRoot)[insertOp.Target]; got != unreachableSiteContent {
		t.Errorf("target content = %q, want the untouched seed %q", got, unreachableSiteContent)
	}
	if len(result.Written) != 0 {
		t.Errorf("result.Written = %v, want empty alongside the error", result.Written)
	}
}
