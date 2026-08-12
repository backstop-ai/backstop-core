package main

// SPEC-067 — the twelve GitLab CI claims.

import (
	"strings"
	"testing"
)

const ciGitLabTarget = ".gitlab-ci.yml"

// TestCIRecipes_GitLabCI_RecipeDeclarationShape proves CLM-009.
func TestCIRecipes_GitLabCI_RecipeDeclarationShape(t *testing.T) {
	if problems := ciRecipeShapeProblems(t, ciRecipeGitLabCI, ciGitLabTarget); len(problems) != 0 {
		t.Errorf("the gitlab-ci-gate recipe declaration is not the shape REQ-003 fixes:\n  %s", strings.Join(problems, "\n  "))
	}
}

// TestCIRecipes_GitLabCI_ApplyWritesOnlyItsDeclaredTarget proves CLM-013.
func TestCIRecipes_GitLabCI_ApplyWritesOnlyItsDeclaredTarget(t *testing.T) {
	added, want := ciAppliedFileDelta(t, ciRecipeGitLabCI)
	if strings.Join(added, "\n") != strings.Join(want, "\n") {
		t.Errorf("applying gitlab-ci-gate added %v, want exactly %v", added, want)
	}
}

// TestCIRecipes_GitLabCI_RenderedDeclaresFullHistory proves CLM-017: the
// `GIT_DEPTH` variable is declared with the value 0.
//
// Both spellings are accepted — GitLab wants the string "0" and a YAML-typed 0 is
// equally valid. Absence and any non-zero value are rejected. Asserted through a
// YAML walk, not a substring.
func TestCIRecipes_GitLabCI_RenderedDeclaresFullHistory(t *testing.T) {
	_, _, rendered := ciApplyProbe(t, ciRecipeGitLabCI)

	doc := ciDecodeYAML(t, rendered)
	if !ciWalkYAMLValue(doc, "GIT_DEPTH", ciIsZero) {
		t.Errorf("the rendered pipeline declares no `GIT_DEPTH` with value 0; a shallow clone makes the diff base unreachable\nrendered:\n%s", rendered)
	}
}

// TestCIRecipes_GitLabCI_RenderedInstallsPinnedBackstop proves CLM-021.
func TestCIRecipes_GitLabCI_RenderedInstallsPinnedBackstop(t *testing.T) {
	_, _, rendered := ciApplyProbe(t, ciRecipeGitLabCI)
	if problems := ciPinnedInstallProblems(rendered); len(problems) != 0 {
		t.Errorf("the rendered GitLab CI config does not install a pinned backstop:\n  %s", strings.Join(problems, "\n  "))
	}
}

// TestCIRecipes_GitLabCI_PackInstallPrecedesGate proves CLM-025. A GitLab
// `script:` list executes in list order, so byte order and execution order agree
// by construction.
func TestCIRecipes_GitLabCI_PackInstallPrecedesGate(t *testing.T) {
	_, _, rendered := ciApplyProbe(t, ciRecipeGitLabCI)

	install := ciByteOffset(t, rendered, "backstop pack install")
	gate := ciByteOffset(t, rendered, "backstop gate")
	if install >= gate {
		t.Errorf("`backstop pack install` is at byte %d and `backstop gate` at byte %d; the install must precede the gate\nrendered:\n%s", install, gate, rendered)
	}
}

// TestCIRecipes_GitLabCI_GateVerdictNotSwallowed proves CLM-029 (absence): the
// universal three plus `allow_failure: true`, which is GitLab's spelling of
// continue-on-error. Exactly those four; in particular not `|| echo`.
func TestCIRecipes_GitLabCI_GateVerdictNotSwallowed(t *testing.T) {
	_, _, rendered := ciApplyProbe(t, ciRecipeGitLabCI)
	if problems := ciSwallowFormsFound(rendered, []string{"|| true", "|| exit 0", "allow_failure: true"}); len(problems) != 0 {
		t.Errorf("the rendered GitLab CI config swallows the gate verdict:\n  %s", strings.Join(problems, "\n  "))
	}
}

// TestCIRecipes_GitLabCI_ResolvesDiffBaseAndFailsWhenUnresolvable proves CLM-033.
func TestCIRecipes_GitLabCI_ResolvesDiffBaseAndFailsWhenUnresolvable(t *testing.T) {
	_, _, rendered := ciApplyProbe(t, ciRecipeGitLabCI)
	if problems := ciBaseResolutionProblems(t, rendered,
		[]string{"CI_MERGE_REQUEST_TARGET_BRANCH_NAME", "CI_DEFAULT_BRANCH"}, "main"); len(problems) != 0 {
		t.Errorf("the rendered GitLab CI config does not resolve a diff base and fail loud:\n  %s", strings.Join(problems, "\n  "))
	}
}

// TestCIRecipes_GitLabCI_RenderedParsesAsPipelineYAML proves CLM-037: the file
// parses as YAML declaring at least one job mapping carrying a non-empty
// `script` list.
func TestCIRecipes_GitLabCI_RenderedParsesAsPipelineYAML(t *testing.T) {
	_, _, rendered := ciApplyProbe(t, ciRecipeGitLabCI)

	doc := ciDecodeYAML(t, rendered)
	jobs := 0
	for name, raw := range doc {
		job, isMapping := raw.(map[string]any)
		if !isMapping {
			continue
		}
		script, hasScript := job["script"].([]any)
		if hasScript && len(script) > 0 {
			jobs++
			if name == "" {
				t.Errorf("a job carrying a script list has no name")
			}
		}
	}
	if jobs == 0 {
		t.Errorf("the rendered pipeline declares no job mapping with a non-empty `script` list\nrendered:\n%s", rendered)
	}
}

// TestCIRecipes_GitLabCI_RenderedCarriesNoResidualPlaceholder proves CLM-041.
func TestCIRecipes_GitLabCI_RenderedCarriesNoResidualPlaceholder(t *testing.T) {
	_, _, rendered := ciApplyProbe(t, ciRecipeGitLabCI)
	if problems := ciResidualPlaceholders(rendered); len(problems) != 0 {
		t.Errorf("the rendered GitLab CI config carries a residual placeholder:\n  %s", strings.Join(problems, "\n  "))
	}
}

// TestCIRecipes_GitLabCI_ReapplyIsByteIdentical proves CLM-046.
func TestCIRecipes_GitLabCI_ReapplyIsByteIdentical(t *testing.T) {
	if problems := ciReapplyProblems(t, ciRecipeGitLabCI); len(problems) != 0 {
		t.Errorf("re-applying gitlab-ci-gate is not byte-identical and write-free:\n  %s", strings.Join(problems, "\n  "))
	}
}

// TestCIRecipes_GitLabCI_RulesScopedByTargetBasenameGlob proves CLM-054.
//
// TWO patterns, and exactly two: the first matches the deployed dot-prefixed
// target, the second the undotted fixture names REQ-009 mandates. One glob cannot
// cover both spellings.
func TestCIRecipes_GitLabCI_RulesScopedByTargetBasenameGlob(t *testing.T) {
	if problems := ciGlobScopingProblems(t, ciRecipeGitLabCI,
		[]string{".gitlab-ci*.yml", "gitlab-ci*.yml"}); len(problems) != 0 {
		t.Errorf("the gitlab-ci rules are not scoped by their target basename globs:\n  %s", strings.Join(problems, "\n  "))
	}
}

// TestCIRecipes_GitLabCI_RenderedCarriesNoToolchainOrConsumerLiteral proves
// CLM-060 (absence). GitLab references no action or pipe at all, so the permitted
// org/repo literal set is the backstop release coordinate ALONE — in particular
// `actions/checkout` must not appear here.
func TestCIRecipes_GitLabCI_RenderedCarriesNoToolchainOrConsumerLiteral(t *testing.T) {
	_, _, rendered := ciApplyProbe(t, ciRecipeGitLabCI)
	if problems := ciTravelProblems(rendered, nil); len(problems) != 0 {
		t.Errorf("the rendered GitLab CI config does not travel:\n  %s", strings.Join(problems, "\n  "))
	}
}
