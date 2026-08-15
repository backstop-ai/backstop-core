package initialize

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/recipe"
)

// TestInit_ScaffoldingKindEmptyPairPreserveIsClassifiedUserOwned (SPEC-069 CLM-123).
//
// An EMPTY Rule/CoveringWaiver pair at declared kind `scaffolding` is UNAMBIGUOUSLY
// producer (a) — the consumer's own file, which no apply of this recipe ever produced
// — because preserveOrRegenerate's producer (b) branch is unreachable for that kind.
// That makes it the REQ-035 brownfield gap.
func TestInit_ScaffoldingKindEmptyPairPreserveIsClassifiedUserOwned(t *testing.T) {
	divergence := recipe.PreservedDivergence{Path: "ci/pipeline.yml"}

	got := classifyPreserve(divergence, recipe.KindScaffolding)
	if got != PreserveUserOwned {
		t.Fatalf("classifyPreserve(empty pair, kind=%s) = %v, want PreserveUserOwned: branch (b) is unreachable for a scaffolding recipe, so the empty pair can only be the consumer's own file",
			recipe.KindScaffolding, got)
	}

	// The falsifier: the SAME path with a POPULATED pair must not land here, or the
	// classifier is ignoring the discriminator entirely.
	covered := recipe.PreservedDivergence{Path: "ci/pipeline.yml", Rule: "pack/rule", CoveringWaiver: coveringWaiverFixture("pack/rule", "accepted", "2027-01-01")}
	if classifyPreserve(covered, recipe.KindScaffolding) == PreserveUserOwned {
		t.Fatal("a POPULATED Rule/CoveringWaiver pair classified user-owned; the pair outranks the kind and marks an accountable customization")
	}
}

// TestInit_ImplementingKindEmptyPairPreserveIsClassifiedUserOwned (SPEC-069 CLM-124).
//
// The same reasoning as CLM-123 for the second kind whose branch (b) is unreachable.
// It is a separate claim rather than a table row because "the two kinds where (b) is
// unreachable" asserted in aggregate hides a missing member.
func TestInit_ImplementingKindEmptyPairPreserveIsClassifiedUserOwned(t *testing.T) {
	divergence := recipe.PreservedDivergence{Path: "lint.config.json"}

	got := classifyPreserve(divergence, recipe.KindImplementing)
	if got != PreserveUserOwned {
		t.Fatalf("classifyPreserve(empty pair, kind=%s) = %v, want PreserveUserOwned", recipe.KindImplementing, got)
	}

	// And the third kind is genuinely DIFFERENT: a templating recipe's empty pair is
	// the case init cannot resolve, so a classifier that returned user-owned for
	// every empty pair would pass the two claims above while being wrong.
	if classifyPreserve(recipe.PreservedDivergence{Path: "lint.config.json"}, recipe.KindTemplating) == PreserveUserOwned {
		t.Fatalf("classifyPreserve(empty pair, kind=%s) returned PreserveUserOwned; producer (a) and producer (b) are indistinguishable at that kind, so it must be PreserveIndeterminate",
			recipe.KindTemplating)
	}
}

// TestInit_PreserveClassifierCoversEveryObservableCombination is the exhaustive
// table behind the two mandated claims above. It is additive: the mandated names
// carry the claims, and this one makes sure no cell of the two-observable matrix
// (pair populated/empty x three declared kinds) is unasserted.
func TestInit_PreserveClassifierCoversEveryObservableCombination(t *testing.T) {
	populated := recipe.PreservedDivergence{Path: "p", Rule: "pack/rule", CoveringWaiver: coveringWaiverFixture("pack/rule", "accepted", "2027-01-01")}
	empty := recipe.PreservedDivergence{Path: "p"}

	cases := []struct {
		kind       string
		divergence recipe.PreservedDivergence
		want       PreserveClass
		why        string
	}{
		{recipe.KindScaffolding, populated, PreserveWaiverCovered, "a populated pair is producer (c) at any kind"},
		{recipe.KindImplementing, populated, PreserveWaiverCovered, "a populated pair is producer (c) at any kind"},
		{recipe.KindTemplating, populated, PreserveWaiverCovered, "the pair OUTRANKS the kind, templating included"},
		{recipe.KindScaffolding, empty, PreserveUserOwned, "branch (b) is unreachable for scaffolding"},
		{recipe.KindImplementing, empty, PreserveUserOwned, "branch (b) is unreachable for implementing"},
		{recipe.KindTemplating, empty, PreserveIndeterminate, "producers (a) and (b) are byte-identical at templating"},
	}

	for _, tc := range cases {
		t.Run(tc.kind+"/"+preserveCasePairName(tc.divergence), func(t *testing.T) {
			if got := classifyPreserve(tc.divergence, tc.kind); got != tc.want {
				t.Fatalf("classifyPreserve(pair=%s, kind=%s) = %v, want %v: %s",
					preserveCasePairName(tc.divergence), tc.kind, got, tc.want, tc.why)
			}
		})
	}
}

// preserveCasePairName labels a table row by the state of its discriminator pair.
func preserveCasePairName(d recipe.PreservedDivergence) string {
	if d.Rule == "" && d.CoveringWaiver == "" {
		return "empty-pair"
	}
	return "populated-pair"
}

// TestInit_HoldsExactlyOnePreserveClassifierSharedByBothRecipeSteps is STRUCTURAL
// (SPEC-069 CLM-144).
//
// REQ-035 requires the init source set to contain EXACTLY ONE implementation of the
// classification, because two classifiers drifting apart would let one recipe step
// report a class the other would not — the "one authority, not a copy" hazard this
// spec refuses everywhere else.
//
// THE DISCRIMINATOR THE TEST KEYS ON is the declared recipe KIND. The kind is
// consulted for exactly one purpose in the whole of init: splitting the empty-pair
// case. So "how many functions in the init source set read a recipe-kind constant"
// IS the count of classifiers, and a second classifier cannot avoid bumping it
// without ceasing to classify.
//
// THE SCAN BOUNDARY IS PART OF THE CLAIM (Sharp Edge 10): the init source set is
// enumerated from a GLOB at test time, never a hardcoded file list, and the
// enumeration is asserted NON-EMPTY first so an empty-set scan fails loudly rather
// than passing trivially.
func TestInit_HoldsExactlyOnePreserveClassifierSharedByBothRecipeSteps(t *testing.T) {
	files := initSourceSet(t)

	fset := token.NewFileSet()
	classifiers := []string{}
	callers := map[string][]string{}

	for _, path := range files {
		parsed, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			t.Fatalf("parsing init source %s: %v", path, err)
		}
		ast.Inspect(parsed, func(node ast.Node) bool {
			fn, ok := node.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				return true
			}
			if functionReadsARecipeKind(fn) {
				classifiers = append(classifiers, filepath.Base(path)+":"+fn.Name.Name)
			}
			for _, called := range calledFunctionNames(fn) {
				if called == preserveClassifierName {
					callers[filepath.Base(path)] = append(callers[filepath.Base(path)], fn.Name.Name)
				}
			}
			return true
		})
	}

	if len(classifiers) != 1 {
		sort.Strings(classifiers)
		t.Fatalf("the init source set holds %d functions reading a recipe-kind constant (%v), want exactly ONE.\nThe declared kind is consulted for exactly one purpose — splitting the empty Rule/CoveringWaiver pair — so more than one is a second preserve classifier, and two classifiers drifting would let one recipe step report a class the other would not.",
			len(classifiers), classifiers)
	}

	// ★ THE COUNT ABOVE READS CONSTANTS, AND A SECOND CLASSIFIER NEED NOT USE THEM.
	//
	// functionReadsARecipeKind matches the recipe package's Kind* selectors. A rival
	// classifier written as `if kind == "scaffolding"` reads the SAME observable while
	// touching no constant at all — the count stays at one, this test stays green, and
	// two live classifiers drift apart exactly as CLM-144 forbids. That evasion is not
	// hypothetical; it is one keystroke from the honest spelling.
	//
	// So the raw kind SPELLINGS are banned as string literals anywhere in the init
	// source set outside preserve.go, which forces any code that wants to reason about a
	// kind through the constants — and therefore through the count.
	for _, path := range files {
		if filepath.Base(path) == "preserve.go" {
			continue
		}
		for _, literal := range nonImportStringLiterals(t, path) {
			for _, kind := range []string{`"scaffolding"`, `"implementing"`, `"templating"`} {
				if literal == kind {
					t.Fatalf("%s carries the raw recipe-kind literal %s. Reasoning about a kind through its SPELLING rather than its constant evades the one-classifier count above entirely — the count would stay at 1 while two classifiers were live. Use recipe.Kind* so any second classifier is countable.",
						filepath.Base(path), kind)
				}
			}
		}
	}
	if !strings.HasPrefix(classifiers[0], "preserve.go:") {
		t.Fatalf("the one classifier lives in %s, want preserve.go; CLM-144 pins its home so a reader can find it", classifiers[0])
	}
	if !strings.HasSuffix(classifiers[0], ":"+preserveClassifierName) {
		t.Fatalf("the one classifier is named %s, want %s", classifiers[0], preserveClassifierName)
	}

	// ═══ THE CALLER HALF, LOCKED HOP BY HOP ═══
	//
	// "Both recipe steps run their preserves through the one classifier" is a CHAIN,
	// and asserting only its endpoints would let a step reach the classifier by some
	// other route. So each hop is pinned separately:
	//
	//	hop 1: exactly ONE function calls classifyPreserve, and it is the shared loop;
	//	hop 2: BOTH recipe steps call that shared loop, and nothing else does.
	//
	// A second classification would have to break one of the two, because it would
	// either call the classifier (breaking hop 1) or read the kind itself (breaking the
	// count above).
	if len(callers) != 1 {
		t.Fatalf("%d file(s) call %s (%v), want exactly ONE — the shared loop. More than one call site means each recipe step reaches the classifier by its own route, which is the drift a single classifier was supposed to make impossible.",
			len(callers), preserveClassifierName, callers)
	}
	if _, viaPreserve := callers["preserve.go"]; !viaPreserve {
		t.Fatalf("the single call site of %s is not in preserve.go: %v", preserveClassifierName, callers)
	}
	if !containsName(callers["preserve.go"], preserveLoopName) {
		t.Fatalf("preserve.go calls %s from %v, want it called from %s", preserveClassifierName, callers["preserve.go"], preserveLoopName)
	}

	// Hop 2. It acquires teeth the moment the two recipe-step files land, and is
	// asserted here so the whole invariant is stated once, in one place.
	recipeSteps := []string{"step_ci.go", "step_scaffold.go"}
	present := []string{}
	loopCallers := map[string][]string{}
	for _, path := range files {
		base := filepath.Base(path)
		if containsName(recipeSteps, base) {
			present = append(present, base)
		}
		parsed, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			t.Fatalf("re-parsing init source %s: %v", path, err)
		}
		ast.Inspect(parsed, func(node ast.Node) bool {
			fn, ok := node.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				return true
			}
			for _, called := range calledFunctionNames(fn) {
				if called == preserveLoopName {
					loopCallers[base] = append(loopCallers[base], fn.Name.Name)
				}
			}
			return true
		})
	}

	for _, step := range present {
		if len(loopCallers[step]) == 0 {
			t.Fatalf("%s does not call %s; both recipe steps must run their preserves through the ONE shared classifier", step, preserveLoopName)
		}
	}
	for file := range loopCallers {
		if !containsName(recipeSteps, file) {
			t.Fatalf("%s calls %s, but only the two recipe steps classify preserves; a third caller means something other than a recipe apply is producing preserve classes",
				file, preserveLoopName)
		}
	}
	if len(present) == len(recipeSteps) && len(loopCallers) != len(recipeSteps) {
		t.Fatalf("both recipe steps exist but %d file(s) reach the classifier (%v); want exactly the two", len(loopCallers), loopCallers)
	}
}

// preserveLoopName is the ONE hop between a recipe step and the classifier.
const preserveLoopName = "classifyApplyPreserves"

// preserveClassifierName is the single classifier's name, asserted rather than
// assumed so a rename has to be a deliberate edit to this test too.
const preserveClassifierName = "classifyPreserve"

// initSourceSet enumerates the init source set from a GLOB — `pkg/initialize/**`
// plus `cmd/backstop/init*.go`, excluding `_test.go` — and fails when the
// enumeration is empty.
//
// The non-empty guard is the point: a future implementer who moves init logic
// outside this boundary would otherwise silently empty every structural claim that
// scans it, without failing one.
func initSourceSet(t *testing.T) []string {
	t.Helper()

	repoRoot := repositoryRoot(t)
	var files []string

	packageDir := filepath.Join(repoRoot, "pkg", "initialize")
	walkErr := filepath.Walk(packageDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if walkErr != nil {
		t.Fatalf("enumerating pkg/initialize: %v", walkErr)
	}

	cmdMatches, globErr := filepath.Glob(filepath.Join(repoRoot, "cmd", "backstop", "init*.go"))
	if globErr != nil {
		t.Fatalf("enumerating cmd/backstop/init*.go: %v", globErr)
	}
	cmdSide := []string{}
	for _, path := range cmdMatches {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		cmdSide = append(cmdSide, path)
		files = append(files, path)
	}

	if len(files) == 0 {
		t.Fatal("the init source set enumerated ZERO files; the scan boundary resolved to nothing, so every claim scanning it would pass trivially")
	}

	// ★ THE UNION IS NOT A SUFFICIENT GUARD, AND THAT ASYMMETRY WAS A REAL HOLE.
	//
	// pkg/initialize/** always yields files, so the emptiness check above can NEVER
	// fire even if the cmd-side glob silently returned nothing — after a rename, say,
	// or a move out of `cmd/backstop/init*.go`. Every pkg-side denylist claim would go
	// on passing over a source set that had quietly lost half its territory, which is
	// Sharp Edge 10 landing on the half nobody was watching. The cmd-side scan already
	// asserts one of its files BY NAME; this is the symmetric guard.
	for _, required := range []string{"init.go", "init_seams.go", "init_toolchain.go"} {
		if !containsBaseName(cmdSide, required) {
			t.Fatalf("the cmd/backstop half of the init source set is missing %s (it enumerated %v). Half the scan boundary has moved or been renamed, and every claim scanning it would keep passing over what is left.",
				required, cmdSide)
		}
	}
	sort.Strings(files)
	return files
}

// repositoryRoot walks up from the test's working directory to the module root.
func repositoryRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("resolving working directory: %v", err)
	}
	for {
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("walked to the filesystem root without finding go.mod")
		}
		dir = parent
	}
}

// functionReadsARecipeKind reports whether fn's body mentions any recipe-kind
// constant. The kind is the classifier's second observable and is read nowhere else
// in init, so this is what makes "how many classifiers are there" a countable
// question.
func functionReadsARecipeKind(fn *ast.FuncDecl) bool {
	kinds := map[string]bool{"KindScaffolding": true, "KindImplementing": true, "KindTemplating": true}
	found := false
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		selector, ok := node.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if kinds[selector.Sel.Name] {
			found = true
		}
		return true
	})
	return found
}

// calledFunctionNames returns the unqualified names of every function fn calls.
func calledFunctionNames(fn *ast.FuncDecl) []string {
	names := []string{}
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		switch callee := call.Fun.(type) {
		case *ast.Ident:
			names = append(names, callee.Name)
		case *ast.SelectorExpr:
			names = append(names, callee.Sel.Name)
		}
		return true
	})
	return names
}

// containsName reports whether needle is in haystack.
func containsName(haystack []string, needle string) bool {
	for _, candidate := range haystack {
		if candidate == needle {
			return true
		}
	}
	return false
}

// containsBaseName reports whether any path in paths has the given base name.
func containsBaseName(paths []string, base string) bool {
	for _, path := range paths {
		if filepath.Base(path) == base {
			return true
		}
	}
	return false
}
