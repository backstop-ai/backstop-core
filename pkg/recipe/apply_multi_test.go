package recipe

import (
	"sort"
	"strings"
	"testing"
)

// Multi-recipe apply (REQ-013): strictly sequential, in the DECLARED order the
// consumer supplied, never reordered, never interleaved, never parallelized.
//
// Every recipe below is declared as recipe.yml DATA parsed through the real
// ParseRecipeManifest, and every expectation is rebuilt from the snippets read back
// off those parsed manifests, so nothing here can agree with the applier by
// coincidence.

// multiSharedTarget is the ONE file the order-revealing recipes co-write. Each
// insert splices immediately after the shared anchor, so the LAST op to run ends up
// NEAREST the anchor and the file records the execution order in reverse.
const multiSharedTarget = "shared/registry.txt"

const multiRecipeOne = `
kind: scaffolding
version: 1.0.0
ops:
  - id: op-one
    kind: insert
    target: shared/registry.txt
    anchor: "# SLOTS"
    snippet: "\nfrom-recipe-one"
    manual: "Add the line beneath the SLOTS marker by hand."
`

const multiRecipeTwo = `
kind: scaffolding
version: 1.0.0
ops:
  - id: op-two
    kind: insert
    target: shared/registry.txt
    anchor: "# SLOTS"
    snippet: "\nfrom-recipe-two"
    manual: "Add the line beneath the SLOTS marker by hand."
`

const multiRecipeThree = `
kind: scaffolding
version: 1.0.0
ops:
  - id: op-three
    kind: insert
    target: shared/registry.txt
    anchor: "# SLOTS"
    snippet: "\nfrom-recipe-three"
    manual: "Add the line beneath the SLOTS marker by hand."
`

// multiRecipeBroken declares an anchor that is ABSENT from the shared target, so
// it is the recipe that fails mid-sequence.
const multiRecipeBroken = `
kind: scaffolding
version: 1.0.0
ops:
  - id: op-broken
    kind: insert
    target: shared/registry.txt
    anchor: "# NOT PRESENT"
    snippet: "\nfrom-the-broken-recipe"
    manual: "Add the line beneath the marker you choose, by hand."
`

// resolvedNamed parses a recipe manifest and binds it to a DISTINCT pack + recipe
// identity, so a multi-recipe run spans more than one pack and each recipe owns its
// own adoption key.
func resolvedNamed(t *testing.T, dir string, packName string, recipeName string, recipeYAML string) *ResolvedRecipe {
	t.Helper()

	manifest, err := ParseRecipeManifest([]byte(recipeYAML))
	if err != nil {
		t.Fatalf("parse test recipe manifest: %v", err)
	}

	return &ResolvedRecipe{
		Ref:      RecipeRef{Pack: packName, Recipe: recipeName, Version: manifest.Version},
		Dir:      dir,
		Manifest: manifest,
	}
}

// applyMultiInto seeds a FRESH project root with the shared target and applies the
// supplied recipes through ApplyAll, returning the resulting tree.
func applyMultiInto(t *testing.T, recipes []*ResolvedRecipe) (map[string]string, []ApplyResult, error) {
	t.Helper()

	projectRoot := t.TempDir()
	writeUnder(t, projectRoot, multiSharedTarget, seedPayload)
	results, err := ApplyAll(recipes, ApplyOptions{Mode: ModeDirect, ProjectRoot: projectRoot})

	return snapshotTree(t, projectRoot), results, err
}

// composedInOrder rebuilds the bytes the shared target must carry after the given
// EXECUTION order. Each splice lands immediately after the anchor, so the observable
// sequence is the execution order REVERSED, and the expectation is assembled from
// the snippets read off the parsed manifests rather than re-typed.
func composedInOrder(executed []*ResolvedRecipe) string {
	composed := "# SLOTS"
	for index := len(executed) - 1; index >= 0; index-- {
		ops := executed[index].Manifest.Ops
		for opIndex := len(ops) - 1; opIndex >= 0; opIndex-- {
			composed += ops[opIndex].Snippet
		}
	}

	return composed + "\n# END\n"
}

// TestApplyMulti_AppliesInDeclaredOrder proves several recipes, spanning more than
// one pack, apply strictly in the order the consumer declared (CLM-057).
//
// The falsifier is the reversed input slice: the observable order must reverse WITH
// it. An applier that sorted by pack, recipe id, or version would emit identical
// bytes for both orders. The third sub-case pins the first-failure behavior: the run
// STOPS at the failing recipe and reports the results accumulated so far, so a
// caller can see exactly how far the sequence got.
func TestApplyMulti_AppliesInDeclaredOrder(t *testing.T) {
	one := resolvedNamed(t, t.TempDir(), "first-org/first-pack", "one", multiRecipeOne)
	two := resolvedNamed(t, t.TempDir(), "second-org/second-pack", "two", multiRecipeTwo)
	three := resolvedNamed(t, t.TempDir(), "second-org/second-pack", "three", multiRecipeThree)

	forwardOrder := []*ResolvedRecipe{one, two, three}
	forwardTree, forwardResults, err := applyMultiInto(t, forwardOrder)
	if err != nil {
		t.Fatalf("ApplyAll in the declared order: unexpected error: %v", err)
	}
	if len(forwardResults) != len(forwardOrder) {
		t.Fatalf("ApplyAll returned %d results, want one per recipe", len(forwardResults))
	}
	for index, result := range forwardResults {
		if len(result.Written) != 1 || result.Written[0] != multiSharedTarget {
			t.Errorf("results[%d].Written = %v, want exactly [%q]", index, result.Written, multiSharedTarget)
		}
	}

	wantForward := composedInOrder(forwardOrder)
	if got := forwardTree[multiSharedTarget]; got != wantForward {
		t.Errorf("declared-order composition =\n%q\nwant\n%q", got, wantForward)
	}

	reversedOrder := []*ResolvedRecipe{three, two, one}
	reversedTree, reversedResults, err := applyMultiInto(t, reversedOrder)
	if err != nil {
		t.Fatalf("ApplyAll in the reversed order: unexpected error: %v", err)
	}
	if len(reversedResults) != len(reversedOrder) {
		t.Fatalf("ApplyAll returned %d results for the reversed order, want one per recipe", len(reversedResults))
	}
	wantReversed := composedInOrder(reversedOrder)
	if wantForward == wantReversed {
		t.Fatal("the two orders expect the same bytes; the comparison would be a tautology")
	}
	if got := reversedTree[multiSharedTarget]; got != wantReversed {
		t.Errorf("reversed-order composition =\n%q\nwant\n%q", got, wantReversed)
	}

	// First failure: the run stops there and reports what it had already applied.
	broken := resolvedNamed(t, t.TempDir(), "second-org/second-pack", "broken", multiRecipeBroken)
	stoppedOrder := []*ResolvedRecipe{one, broken, three}
	stoppedTree, stoppedResults, stoppedErr := applyMultiInto(t, stoppedOrder)
	if stoppedErr == nil {
		t.Fatalf("ApplyAll past a failing recipe: expected a fail-loud error, got nil (%+v)", stoppedResults)
	}
	if len(stoppedResults) != 1 {
		t.Errorf("ApplyAll returned %d results after failing at recipe 2, want the 1 accumulated before it", len(stoppedResults))
	}
	stopped := stoppedTree[multiSharedTarget]
	if !strings.Contains(stopped, one.Manifest.Ops[0].Snippet) {
		t.Errorf("the shared target %q does not carry the first recipe contribution; the sequence never started", stopped)
	}
	if strings.Contains(stopped, three.Manifest.Ops[0].Snippet) {
		t.Errorf("the shared target %q carries the third recipe contribution; the run continued past the failure", stopped)
	}
}

// The co-write pair: two recipes merging DIFFERENT fragments into the SAME
// captured structured target. Both declare the target the consumer already has on
// disk; neither declares anything about the other.
const multiMergeTarget = "app/config.json"

const multiMergeRecipeFirst = `
kind: scaffolding
version: 1.0.0
ops:
  - id: op-merge-first
    kind: merge
    target: app/config.json
    fragment: fragment.json
`

const multiMergeRecipeSecond = `
kind: scaffolding
version: 1.0.0
ops:
  - id: op-merge-second
    kind: merge
    target: app/config.json
    fragment: fragment.json
`

// multiSharedScalar is the name BOTH fragments assign, so the declared order
// decides which assignment survives. multiFirstMark / multiSecondMark are the names
// only one fragment declares, so BOTH contributions are observable.
const (
	multiSharedScalar = "shared_scalar"
	multiFirstMark    = "added_by_the_first_recipe"
	multiSecondMark   = "added_by_the_second_recipe"
)

// nestedTableName returns the name of the first nested table in the CAPTURED
// target, chosen by sorted name so the pick is deterministic. Deriving it keeps the
// fragments free of any noun borrowed from the captured document domain.
func nestedTableName(t *testing.T, captured map[string]any) string {
	t.Helper()

	names := make([]string, 0, len(captured))
	for name, value := range captured {
		if _, isTable := value.(map[string]any); isTable {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		t.Fatal("the captured merge target carries no nested table; the deep-merge half of this test would be vacuous")
	}
	sort.Strings(names)

	return names[0]
}

// mergeFragmentFor authors one recipe fragment: a mark only this recipe declares, a
// shared scalar both declare, and a contribution INTO the captured nested table.
func mergeFragmentFor(mark string, scalar string, nested string, nestedMark string) string {
	return "{\n" +
		"  \"" + mark + "\": true,\n" +
		"  \"" + multiSharedScalar + "\": \"" + scalar + "\",\n" +
		"  \"" + nested + "\": { \"" + nestedMark + "\": \"contributed\" }\n" +
		"}\n"
}

// TestApplyMulti_SameFileCoWritesComposeViaMerge proves two recipes merging into
// the SAME captured structured file COMPOSE (CLM-058): the final document carries
// BOTH contributions, and on a direct scalar conflict the LATER declared recipe
// wins. This is normal composition, not a conflict to arbitrate: no error, no
// prompt, no preserved divergence.
//
// The falsifier is the reversed order, which must flip the winner while both
// contributions still land.
func TestApplyMulti_SameFileCoWritesComposeViaMerge(t *testing.T) {
	captured := readMergeFixture(t, "target.json")
	capturedTree, err := decodeJSONTree(captured)
	if err != nil {
		t.Fatalf("decode the captured merge target: %v", err)
	}
	nested := nestedTableName(t, capturedTree)

	compose := func(t *testing.T, reversedOrder bool) map[string]any {
		t.Helper()

		firstDir, secondDir := t.TempDir(), t.TempDir()
		first := resolvedNamed(t, firstDir, "first-org/first-pack", "merge-first", multiMergeRecipeFirst)
		second := resolvedNamed(t, secondDir, "second-org/second-pack", "merge-second", multiMergeRecipeSecond)
		writeUnder(t, firstDir, first.Manifest.Ops[0].Fragment,
			mergeFragmentFor(multiFirstMark, "from-the-first-recipe", nested, "first_contribution"))
		writeUnder(t, secondDir, second.Manifest.Ops[0].Fragment,
			mergeFragmentFor(multiSecondMark, "from-the-second-recipe", nested, "second_contribution"))

		recipes := []*ResolvedRecipe{first, second}
		if reversedOrder {
			recipes = []*ResolvedRecipe{second, first}
		}

		projectRoot := t.TempDir()
		writeUnder(t, projectRoot, multiMergeTarget, string(captured))
		results, applyErr := ApplyAll(recipes, ApplyOptions{Mode: ModeDirect, ProjectRoot: projectRoot})
		if applyErr != nil {
			t.Fatalf("ApplyAll co-writing one file: unexpected error: %v", applyErr)
		}
		if len(results) != len(recipes) {
			t.Fatalf("ApplyAll returned %d results, want one per recipe", len(results))
		}
		for index, result := range results {
			if len(result.Preserved) != 0 {
				t.Errorf("results[%d].Preserved = %v; a co-write composes rather than being arbitrated", index, result.Preserved)
			}
		}

		merged, decodeErr := decodeJSONTree([]byte(snapshotTree(t, projectRoot)[multiMergeTarget]))
		if decodeErr != nil {
			t.Fatalf("decode the co-written target: %v", decodeErr)
		}

		return merged
	}

	forward := compose(t, false)
	if forward[multiFirstMark] != true || forward[multiSecondMark] != true {
		t.Errorf("the composed document = %v, want BOTH contributions", forward)
	}
	if forward[multiSharedScalar] != "from-the-second-recipe" {
		t.Errorf("%s = %v, want the LATER declared recipe to win the direct conflict", multiSharedScalar, forward[multiSharedScalar])
	}
	for name := range capturedTree {
		if _, survived := forward[name]; !survived {
			t.Errorf("the captured name %q was dropped by the co-write", name)
		}
	}

	capturedNested, isTable := capturedTree[nested].(map[string]any)
	if !isTable {
		t.Fatalf("the captured %q is not a table; the deep-merge assertion would be vacuous", nested)
	}
	mergedNested, isTable := forward[nested].(map[string]any)
	if !isTable {
		t.Fatalf("the composed %q is not a table: %v", nested, forward[nested])
	}
	for name := range capturedNested {
		if _, survived := mergedNested[name]; !survived {
			t.Errorf("the captured nested name %q was dropped by the co-write", name)
		}
	}
	if mergedNested["first_contribution"] != "contributed" || mergedNested["second_contribution"] != "contributed" {
		t.Errorf("the composed %q = %v, want BOTH nested contributions", nested, mergedNested)
	}

	reversed := compose(t, true)
	if reversed[multiSharedScalar] != "from-the-first-recipe" {
		t.Errorf("reversing the declared order left %s = %v; the declared order decides the winner", multiSharedScalar, reversed[multiSharedScalar])
	}
	if reversed[multiFirstMark] != true || reversed[multiSecondMark] != true {
		t.Errorf("the reversed composition = %v, want BOTH contributions regardless of order", reversed)
	}
}

// The two-op recipes for the interleaving falsifier. Their pack names and versions
// are deliberately NOT in the given order: zulu (v2.0.0) is declared FIRST and alpha
// (v1.0.0) SECOND, so a sort by pack name, recipe id, or version would reorder them.
const multiRecipeZulu = `
kind: scaffolding
version: 2.0.0
ops:
  - id: op-zulu-first
    kind: insert
    target: shared/registry.txt
    anchor: "# SLOTS"
    snippet: "\nzulu-first"
    manual: "Add the line beneath the SLOTS marker by hand."
  - id: op-zulu-second
    kind: insert
    target: shared/registry.txt
    anchor: "# SLOTS"
    snippet: "\nzulu-second"
    manual: "Add the line beneath the SLOTS marker by hand."
`

const multiRecipeAlpha = `
kind: scaffolding
version: 1.0.0
ops:
  - id: op-alpha-first
    kind: insert
    target: shared/registry.txt
    anchor: "# SLOTS"
    snippet: "\nalpha-first"
    manual: "Add the line beneath the SLOTS marker by hand."
  - id: op-alpha-second
    kind: insert
    target: shared/registry.txt
    anchor: "# SLOTS"
    snippet: "\nalpha-second"
    manual: "Add the line beneath the SLOTS marker by hand."
`

// TestApplyMulti_NeverReordersOrInterleaves proves the full observable sequence is
// recipe-1 ops in order, THEN recipe-2 ops in order (CLM-059). The input slice is
// deliberately NOT in lexical or version order, so a sorting implementation is
// caught, and the two wrong-but-plausible sequences (sorted, interleaved) are
// computed and asserted DIFFERENT from the expectation, so the comparison cannot
// pass by accident.
func TestApplyMulti_NeverReordersOrInterleaves(t *testing.T) {
	zulu := resolvedNamed(t, t.TempDir(), "zulu-org/zulu-pack", "zulu", multiRecipeZulu)
	alpha := resolvedNamed(t, t.TempDir(), "alpha-org/alpha-pack", "alpha", multiRecipeAlpha)

	givenOrder := []*ResolvedRecipe{zulu, alpha}
	tree, results, err := applyMultiInto(t, givenOrder)
	if err != nil {
		t.Fatalf("ApplyAll: unexpected error: %v", err)
	}
	if len(results) != len(givenOrder) {
		t.Fatalf("ApplyAll returned %d results, want one per recipe", len(results))
	}

	want := composedInOrder(givenOrder)
	got := tree[multiSharedTarget]
	if got != want {
		t.Errorf("observable sequence =\n%q\nwant recipe-1 ops in order then recipe-2 ops in order\n%q", got, want)
	}

	// A sort by pack/recipe/version would run alpha first.
	sorted := composedInOrder([]*ResolvedRecipe{alpha, zulu})
	if sorted == want {
		t.Fatal("the sorted sequence expects the same bytes as the given order; the falsifier is inert")
	}
	if got == sorted {
		t.Errorf("the applier produced the SORTED sequence %q; the given order must survive", got)
	}

	// An interleave would alternate the two recipes op by op. Reading the file
	// top-down yields the reverse of the execution order, so the interleaved
	// expectation is built by reversing z1, a1, z2, a2.
	interleaved := "# SLOTS" +
		alpha.Manifest.Ops[1].Snippet +
		zulu.Manifest.Ops[1].Snippet +
		alpha.Manifest.Ops[0].Snippet +
		zulu.Manifest.Ops[0].Snippet +
		"\n# END\n"
	if interleaved == want {
		t.Fatal("the interleaved sequence expects the same bytes as the given order; the falsifier is inert")
	}
	if got == interleaved {
		t.Errorf("the applier INTERLEAVED the recipes: %q", got)
	}
}
