package recipe

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The transform half of the applier, driven against the TASK-004 capture corpus.
//
// What is deliberately NOT here: the engine TRUST GATE. pkg/recipe does not import
// pkg/pack/engine and holds no tool name or locked version to check — opts.Dispatch
// is the whole seam, and the concrete engine.CheckToolAllowed gate is wired at
// cmd/backstop, the layer that can see the pack's engines: block and backstop.lock.

// capturedTransformDirName / capturedCaptureSourceName locate the capture corpus.
// Every operand below is read back out of those files, so nothing about the engine
// invocation is re-typed here.
const (
	capturedTransformDirName  = "transform"
	capturedCaptureSourceName = "CAPTURE-SOURCE-transform.md"
	capturedBeforeName        = "before.go"
	capturedAfterName         = "after.go"
	capturedTransformRuleName = "rewrite-rule.yml"
)

// capturedRecipeSubdir is where the per-recipe directory sits INSIDE the pack, the
// same shape ResolveRecipe produces (pack root + the indexed recipe directory). The
// two bases have to be distinguishable for the rule-resolution assertions to mean
// anything: a rule staged under the recipe directory would prove only that the applier
// agrees with the test.
const capturedRecipeSubdir = "recipes/rewrite"

// capturedShellPrompt / capturedRewriteFlag identify the ONE documented shell line in
// the capture-source note that produced after.go. The flag is an engine option, not a
// tool name, so keeping it here bakes nothing: the tool itself is read from the note.
const (
	capturedShellPrompt = "$ "
	capturedRewriteFlag = "--update-all"
)

// capturedTransformRecipe declares the transform op the captured rule rewrites. The
// rule is declared PACK-relative, as the recipe contract requires and as the committed
// pack fixtures declare theirs; its base name is the captured rule file, so the fixture
// reaches the pack by copy rather than by a re-typed rule body.
const capturedTransformRecipe = `
kind: implementing
version: 1.0.0
transform_rules:
  - recipes/rewrite/rules/rewrite-rule.yml
ops:
  - id: op-captured-transform
    kind: transform
    target: source/greeting.go
    rule: recipes/rewrite/rules/rewrite-rule.yml
    manual: "Re-point the superseded call sites in the declared target by hand."
`

// capturedTransformPath joins a name onto the captured transform fixture directory.
func capturedTransformPath(name string) string {
	return filepath.Join("testdata", "captured", capturedTransformDirName, name)
}

// packRootFor lays a recipe out the way resolution does — the per-recipe directory
// nested under the PACK ROOT, with both bases carried on the resolved recipe — and
// returns the resolved recipe alongside that pack root.
func packRootFor(t *testing.T, recipeYAML string) (*ResolvedRecipe, string) {
	t.Helper()

	packDir := t.TempDir()
	recipeDir := filepath.Join(packDir, filepath.FromSlash(capturedRecipeSubdir))
	if err := os.MkdirAll(recipeDir, 0o755); err != nil {
		t.Fatalf("lay out the recipe directory under the pack root: %v", err)
	}

	resolved := resolvedFromManifest(t, recipeDir, recipeYAML)
	resolved.PackDir = packDir

	return resolved, packDir
}

// documentedCaptureArgv returns the argv of the command the capture note records as
// having produced after.go, read out of the note as DATA.
//
// The tool NAME is never a literal in this file: a literal tool name handed to
// exec.Command is exactly what backstop/self Family A forbids (globally, test source
// included), and the point of the transform seam is that the engine comes from
// declared data. Reading the documented command back also makes this test run the
// SAME invocation the fixture was captured with — if the note and the test ever
// diverge, the captured bytes stop matching.
func documentedCaptureArgv(t *testing.T) []string {
	t.Helper()

	note, err := os.ReadFile(filepath.Join("testdata", "captured", capturedCaptureSourceName))
	if err != nil {
		t.Fatalf("read the capture-source note: %v", err)
	}

	for _, line := range strings.Split(string(note), "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, capturedShellPrompt) || !strings.Contains(trimmed, capturedRewriteFlag) {
			continue
		}
		argv := strings.Fields(strings.TrimPrefix(trimmed, capturedShellPrompt))
		if len(argv) < 2 {
			t.Fatalf("the documented capture command %q carries no operands", trimmed)
		}
		return argv
	}

	t.Fatalf("the capture-source note records no %q command; the transform fixture cannot be reproduced", capturedRewriteFlag)
	return nil
}

// realEngineDispatch builds a TransformDispatch that actually RUNS the documented
// capture command, with its two operands re-pointed at the rule and target the
// applier resolved. A dispatch that did nothing would leave the target byte-identical
// and fail the captured comparison — which is the whole point of driving a real run.
//
// The engine not being runnable, or the resolved rule path not being readable, is a
// FAILURE, never a skip: a skipped transform test is indistinguishable from a passing
// no-op, and a mis-resolved rule is exactly the defect this file has to catch.
func realEngineDispatch(ctx context.Context, argv []string) TransformDispatch {
	return func(rule string, target string) error {
		operands := argv[1:]
		args := make([]string, 0, len(operands))
		for index, token := range operands {
			switch {
			case filepath.Base(token) == filepath.Base(rule):
				args = append(args, rule)
			case index == len(operands)-1:
				// The documented command's final operand is the file it rewrites.
				args = append(args, target)
			default:
				args = append(args, token)
			}
		}

		output, err := exec.CommandContext(ctx, argv[0], args...).CombinedOutput()
		if err != nil {
			return fmt.Errorf("run the documented capture command %v: %w (output: %s)", args, err, output)
		}

		return nil
	}
}

// TestApply_TransformOp_DispatchesToAllowlistedEngine_CapturedFixture proves a
// transform op runs a REAL engine over the DECLARED rule and rewrites the target
// (CLM-010): a copy of the CAPTURED before-fixture, applied through the dispatch,
// must come out byte-equal to the CAPTURED after-fixture.
//
// The rule is staged at its declared PACK-relative path under the pack root — where a
// real installed pack carries it — so an applier that resolved the rule against the
// recipe directory instead would hand the engine a doubled, non-existent path and fail
// here rather than in a consumer's project.
//
// The before/after inequality is asserted first, so a fixture pair that had gone
// vacuous (a neutral extension the engine's parser ignores, say) fails loudly instead
// of letting a no-op dispatch pass.
func TestApply_TransformOp_DispatchesToAllowlistedEngine_CapturedFixture(t *testing.T) {
	projectRoot := t.TempDir()
	resolved, packDir := packRootFor(t, capturedTransformRecipe)
	transformOp := resolved.Manifest.Ops[0]

	before, err := os.ReadFile(capturedTransformPath(capturedBeforeName))
	if err != nil {
		t.Fatalf("read the captured before-fixture: %v", err)
	}
	after, err := os.ReadFile(capturedTransformPath(capturedAfterName))
	if err != nil {
		t.Fatalf("read the captured after-fixture: %v", err)
	}
	if string(before) == string(after) {
		t.Fatalf("the captured before/after fixtures are identical; the transform corpus proves nothing")
	}

	copyUnder(t, packDir, transformOp.Rule, capturedTransformPath(filepath.Base(transformOp.Rule)))
	writeUnder(t, projectRoot, transformOp.Target, string(before))

	result, err := Apply(resolved, ApplyOptions{
		Mode:        ModeDirect,
		ProjectRoot: projectRoot,
		Dispatch:    realEngineDispatch(t.Context(), documentedCaptureArgv(t)),
	})
	if err != nil {
		t.Fatalf("Apply through a real engine dispatch: unexpected error: %v", err)
	}

	got := snapshotTree(t, projectRoot)[transformOp.Target]
	if got != string(after) {
		t.Errorf("transformed target =\n%q\nwant the captured after-state\n%q", got, string(after))
	}

	var recorded bool
	for _, written := range result.Written {
		if written == transformOp.Target {
			recorded = true
		}
	}
	if !recorded {
		t.Errorf("result.Written = %v, want the rewritten declared target %q", result.Written, transformOp.Target)
	}
}

// TestApply_TransformOp_RunsDeclaredRuleNotBakedLogic proves the applier carries NO
// rewrite of its own (CLM-026): the rule and target handed to the dispatch are the
// recipe's DECLARED paths resolved under the bases they are declared against — the
// rule under the PACK ROOT, the target under the project root — and swapping the
// declared rule for a second one swaps what the dispatch receives.
//
// The nil-dispatch sub-case is the guard that keeps the seam honest — without it,
// every assertion about the transform would be satisfiable by doing nothing.
func TestApply_TransformOp_RunsDeclaredRuleNotBakedLogic(t *testing.T) {
	const secondRuleRecipe = `
kind: implementing
version: 1.0.0
transform_rules:
  - recipes/rewrite/rules/rewrite-rule.yml
  - recipes/rewrite/rules/second-rule.yml
ops:
  - id: op-captured-transform
    kind: transform
    target: source/greeting.go
    rule: recipes/rewrite/rules/second-rule.yml
    manual: "Re-point the superseded call sites in the declared target by hand."
`

	type recorded struct {
		rule   string
		target string
		calls  int
	}

	applyWithRecorder := func(t *testing.T, recipeYAML string) (*ResolvedRecipe, string, string, recorded) {
		t.Helper()

		projectRoot := t.TempDir()
		resolved, packDir := packRootFor(t, recipeYAML)
		transformOp := resolved.Manifest.Ops[0]
		copyUnder(t, packDir, transformOp.Rule, capturedTransformPath(capturedTransformRuleName))
		copyUnder(t, projectRoot, transformOp.Target, capturedTransformPath(capturedBeforeName))

		var seen recorded
		if _, err := Apply(resolved, ApplyOptions{
			Mode:        ModeDirect,
			ProjectRoot: projectRoot,
			Dispatch: func(rule string, target string) error {
				seen.rule = rule
				seen.target = target
				seen.calls++
				return nil
			},
		}); err != nil {
			t.Fatalf("Apply with a recording dispatch: unexpected error: %v", err)
		}

		return resolved, packDir, projectRoot, seen
	}

	firstResolved, firstPackDir, firstProjectRoot, first := applyWithRecorder(t, capturedTransformRecipe)
	firstOp := firstResolved.Manifest.Ops[0]

	if first.calls != 1 {
		t.Fatalf("the dispatch was called %d times, want exactly once per declared transform op", first.calls)
	}
	if want := filepath.Join(firstPackDir, firstOp.Rule); first.rule != want {
		t.Errorf("dispatched rule = %q, want the DECLARED pack-relative rule resolved under the pack root %q", first.rule, want)
	}
	if _, err := os.Stat(first.rule); err != nil {
		t.Errorf("the dispatched rule path is not readable (%v); the engine would be handed a path that does not exist", err)
	}
	if want := filepath.Join(firstProjectRoot, firstOp.Target); first.target != want {
		t.Errorf("dispatched target = %q, want the DECLARED target resolved under the project root %q", first.target, want)
	}

	secondResolved, secondPackDir, _, second := applyWithRecorder(t, secondRuleRecipe)
	secondOp := secondResolved.Manifest.Ops[0]

	if secondOp.Rule == firstOp.Rule {
		t.Fatalf("both recipes declare rule %q; the swap proves nothing", secondOp.Rule)
	}
	if want := filepath.Join(secondPackDir, secondOp.Rule); second.rule != want {
		t.Errorf("dispatched rule after the swap = %q, want the newly DECLARED rule %q; the applier is not resolving declared data", second.rule, want)
	}
	if filepath.Base(second.rule) == filepath.Base(first.rule) {
		t.Errorf("the dispatched rule did not change with the declared rule (both %q); the rewrite is not coming from the recipe", filepath.Base(second.rule))
	}

	t.Run("nil_dispatch_fails_loud", func(t *testing.T) {
		projectRoot := t.TempDir()
		resolved, packDir := packRootFor(t, capturedTransformRecipe)
		transformOp := resolved.Manifest.Ops[0]
		copyUnder(t, packDir, transformOp.Rule, capturedTransformPath(capturedTransformRuleName))
		copyUnder(t, projectRoot, transformOp.Target, capturedTransformPath(capturedBeforeName))
		before := snapshotTree(t, projectRoot)

		result, err := Apply(resolved, ApplyOptions{Mode: ModeDirect, ProjectRoot: projectRoot})
		if err == nil {
			t.Fatalf("Apply with a nil transform dispatch: expected a fail-loud error, got nil (result %+v)", result)
		}
		if !strings.Contains(err.Error(), transformOp.ID) {
			t.Errorf("error %q does not name the offending op %q", err, transformOp.ID)
		}
		if len(result.Written) != 0 {
			t.Errorf("result.Written = %v, want empty alongside the error", result.Written)
		}

		after := snapshotTree(t, projectRoot)
		for path, content := range before {
			if after[path] != content {
				t.Errorf("file %q changed under a nil dispatch: before %q, after %q", path, content, after[path])
			}
		}
	})
}
