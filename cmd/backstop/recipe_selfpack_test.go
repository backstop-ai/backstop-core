package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/check"
	"github.com/backstop-ai/backstop-core/pkg/gate"
	"github.com/backstop-ai/backstop-core/pkg/pack"
)

// The INSTALLED dogfood pack whose rules are the mechanical form of the
// zero-baked-knowledge invariant, and the two source surfaces this spec added.
const (
	selfPackName         = "backstop/self"
	recipeLibrarySubject = "pkg/recipe"
	recipeCLISubject     = "cmd/backstop/recipe_apply.go"
)

// repoRootFromTest resolves the repository root from the package's test working
// directory, since the pack corpus and the subject files are both addressed
// relative to it.
func repoRootFromTest(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	return root
}

// recipeSubjectFiles returns the project-relative files this test dispatches the
// self-pack rules over: every file of the recipe library (its tests included) plus
// the CLI wiring. Paths are collected by WALKING the directory rather than by a
// language-shaped glob, so the file set is discovered, not declared here.
func recipeSubjectFiles(t *testing.T, repoRoot string) []string {
	t.Helper()

	files := []string{}
	libraryRoot := filepath.Join(repoRoot, filepath.FromSlash(recipeLibrarySubject))
	walkErr := filepath.WalkDir(libraryRoot, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(repoRoot, path)
		if relErr != nil {
			return relErr
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	if walkErr != nil {
		t.Fatalf("walk %s: %v", recipeLibrarySubject, walkErr)
	}

	if _, err := os.Stat(filepath.Join(repoRoot, filepath.FromSlash(recipeCLISubject))); err != nil {
		t.Fatalf("subject %s is missing: %v", recipeCLISubject, err)
	}
	files = append(files, recipeCLISubject)

	return files
}

// TestSelfPack_GreenOverRecipeApplier proves CLM-027 (kind: absence): the recipe
// applier and its CLI wiring introduce ZERO baked language/platform/CI knowledge,
// as judged by the INSTALLED backstop/self pack's own rules rather than by reading.
//
// WHAT THIS ACTUALLY GUARANTEES — stated plainly, because the test's name promises
// more than it delivers and pretending otherwise would be its own vacuous green.
// Only the GLOBAL rule families reach these paths: A (no-baked-tool-exec), B1
// (no-baked-tool-command) and B2 (no-baked-language-token). Families B3
// (no-language-literal-on-neutral-spine — the one that catches ".go"/"_test.go"/
// "./..."), B4, B5 and B6 are PATH-SCOPED to the neutral gate spine (pkg/gate/*.go,
// pkg/check/manifest.go, pkg/check/parsers.go, pkg/validate/plan.go,
// cmd/backstop/gate.go, cmd/backstop/pack_gate*.go, pkg/pack/engine/binding.go) and
// additionally exclude *_test.go — none of them covers pkg/recipe or the recipe CLI.
// So what is proven here is: no literal tool name in an exec call, no baked engine
// command string, and no foreign-language/manifest token. Extending B3's include
// list to these paths would give the applier spine-grade coverage; that is a change
// to the backstop/self PACK repo (packs are external by design) and is recorded as
// a follow-up, not claimed here.
//
// CAPABILITY-ABSENT IS A SKIP, NOT A RED. `.backstop/packs/` is gitignored and packs
// install like node_modules, so an absent pack is un-adopted capability, never a
// defect — it must not fail a clean checkout. The skip is LOUD (it names the pack and
// the command that restores coverage) precisely so it can never read as a silent
// pass, and `backstop gate` runs these same rules for real over the whole repo.
func TestSelfPack_GreenOverRecipeApplier(t *testing.T) {
	repoRoot := repoRootFromTest(t)
	packsDir := filepath.Join(repoRoot, ".backstop", "packs")
	packRoot := filepath.Join(packsDir, filepath.FromSlash(selfPackName))

	if info, err := os.Stat(packRoot); err != nil || !info.IsDir() {
		t.Skipf("SELF-PACK COVERAGE ABSENT: the %s pack is not installed at %s, so its rules did NOT run over %s or %s. This is un-adopted capability, not a defect (packs are external and .backstop/packs is gitignored). Restore coverage with: backstop pack add %s", selfPackName, packRoot, recipeLibrarySubject, recipeCLISubject, selfPackName)
	}

	manifest, err := pack.ParseManifestFile(filepath.Join(packRoot, "pack.yml"))
	if err != nil {
		t.Fatalf("parse installed pack %s: %v", selfPackName, err)
	}
	if len(manifest.Content.Ruleset.Rules) == 0 {
		t.Fatalf("installed pack %s declares no rules; a zero-rule dispatch is a vacuous green", selfPackName)
	}

	subjects := recipeSubjectFiles(t, repoRoot)
	scope := &gate.GateScope{Mode: gate.GateScopeModeDiff, Files: subjects}
	runner := &check.ExecCommandRunner{Dir: repoRoot}

	violations, dispatchErr := dispatchPackEngines([]*pack.Manifest{manifest}, packsDir, repoRoot, scope, runner)
	if dispatchErr != nil {
		t.Fatalf("dispatch %s over %v: %v", selfPackName, subjects, dispatchErr)
	}
	if len(violations) == 0 {
		return
	}

	reported := make([]string, 0, len(violations))
	for _, violation := range violations {
		reported = append(reported, violation.Rule+" @ "+violation.File+": "+violation.Message)
	}
	t.Errorf("%s reported %d finding(s) over the recipe applier; make the offending value declared DATA rather than waiving it:\n%s", selfPackName, len(violations), strings.Join(reported, "\n"))
}
