package main

// SPEC-067 — the twelve GitHub Actions claims, every one asserted against a REAL
// apply of the REAL installed pack into a scratch consumer.

import (
	"strings"
	"testing"
)

const (
	ciGitHubTarget = ".github/workflows/backstop-gate.yml"
	// The ONE org/repo literal REQ-010 permits beyond the backstop release
	// coordinate. The permission is CLOSED to this action: `fetch-depth: 0` has
	// no other expression on GitHub Actions, and a second `actions/*` is scope
	// creep into toolchain territory.
	ciGitHubPermittedAction = "actions/checkout"
)

// TestCIRecipes_GitHubActions_RecipeDeclarationShape proves CLM-008.
func TestCIRecipes_GitHubActions_RecipeDeclarationShape(t *testing.T) {
	if problems := ciRecipeShapeProblems(t, ciRecipeGitHubActions, ciGitHubTarget); len(problems) != 0 {
		t.Errorf("the github-actions-gate recipe declaration is not the shape REQ-003 fixes:\n  %s", strings.Join(problems, "\n  "))
	}
}

// TestCIRecipes_GitHubActions_ApplyWritesOnlyItsDeclaredTarget proves CLM-012.
func TestCIRecipes_GitHubActions_ApplyWritesOnlyItsDeclaredTarget(t *testing.T) {
	added, want := ciAppliedFileDelta(t, ciRecipeGitHubActions)
	if strings.Join(added, "\n") != strings.Join(want, "\n") {
		t.Errorf("applying github-actions-gate added %v, want exactly %v", added, want)
	}
}

// TestCIRecipes_GitHubActions_RenderedDeclaresFullHistory proves CLM-016: the
// checkout step declares `fetch-depth: 0`.
//
// Asserted through a YAML WALK rather than a substring, so an indented comment
// mentioning fetch-depth cannot satisfy it.
func TestCIRecipes_GitHubActions_RenderedDeclaresFullHistory(t *testing.T) {
	_, _, rendered := ciApplyProbe(t, ciRecipeGitHubActions)

	doc := ciDecodeYAML(t, rendered)
	if !ciWalkYAMLValue(doc, "fetch-depth", ciIsZero) {
		t.Errorf("the rendered workflow declares no checkout step with `fetch-depth: 0`; a shallow clone makes the diff base unreachable\nrendered:\n%s", rendered)
	}
}

// TestCIRecipes_GitHubActions_RenderedInstallsPinnedBackstop proves CLM-020.
func TestCIRecipes_GitHubActions_RenderedInstallsPinnedBackstop(t *testing.T) {
	_, _, rendered := ciApplyProbe(t, ciRecipeGitHubActions)
	if problems := ciPinnedInstallProblems(rendered); len(problems) != 0 {
		t.Errorf("the rendered GitHub Actions workflow does not install a pinned backstop:\n  %s", strings.Join(problems, "\n  "))
	}
}

// TestCIRecipes_GitHubActions_PackInstallPrecedesGate proves CLM-024: the byte
// offset of `backstop pack install` is strictly less than that of the gate
// invocation, and BOTH appear exactly once.
//
// A gate run against an empty `.backstop/packs/` reports capability_absent on
// every dimension and passes having checked nothing — the exact vacuous green
// ISSUE-020 was filed about. On GitHub Actions the step list is the execution
// order, so byte order and execution order agree by construction.
func TestCIRecipes_GitHubActions_PackInstallPrecedesGate(t *testing.T) {
	_, _, rendered := ciApplyProbe(t, ciRecipeGitHubActions)

	install := ciByteOffset(t, rendered, "backstop pack install")
	gate := ciByteOffset(t, rendered, "backstop gate")
	if install >= gate {
		t.Errorf("`backstop pack install` is at byte %d and `backstop gate` at byte %d; the install must precede the gate\nrendered:\n%s", install, gate, rendered)
	}
}

// TestCIRecipes_GitHubActions_GateVerdictNotSwallowed proves CLM-028 (absence).
//
// The denylist is CLM-028's, term for term: the universal three plus
// `continue-on-error`. `|| echo` is DELIBERATELY not in it, because it marks a
// step that REPORTS rather than gates.
//
// The example this comment used to cite — backstop-core's own diagnostic
// `gate --json` capture, which ended in `|| echo` so it could not gate and left
// the blocking run to the very next step — retired under ISSUE-099, when ci.yml
// collapsed to ONE `--json-out` invocation. The EXEMPTION's reasoning is
// unaffected: `|| echo` is still live in that same ci.yml, on the
// `Confirm the self-healing baseline pull landed a file` step's failure branch
// (`./bin/backstop baseline pull || echo "bare baseline pull exited $?"`), for
// exactly the same reporting-not-gating purpose. Only the cited example moved.
//
// Widening this list is a spec amendment through the spec-author agent, touching
// CLM-028..031 and the four rule files together, never a unilateral edit here.
func TestCIRecipes_GitHubActions_GateVerdictNotSwallowed(t *testing.T) {
	_, _, rendered := ciApplyProbe(t, ciRecipeGitHubActions)
	if problems := ciSwallowFormsFound(rendered, []string{"|| true", "|| exit 0", "continue-on-error"}); len(problems) != 0 {
		t.Errorf("the rendered GitHub Actions workflow swallows the gate verdict:\n  %s", strings.Join(problems, "\n  "))
	}
}

// TestCIRecipes_GitHubActions_ResolvesDiffBaseAndFailsWhenUnresolvable proves
// CLM-032.
func TestCIRecipes_GitHubActions_ResolvesDiffBaseAndFailsWhenUnresolvable(t *testing.T) {
	_, _, rendered := ciApplyProbe(t, ciRecipeGitHubActions)
	if problems := ciBaseResolutionProblems(t, rendered, []string{"GITHUB_BASE_REF", "GITHUB_EVENT_NAME"}, "main"); len(problems) != 0 {
		t.Errorf("the rendered GitHub Actions workflow does not resolve a diff base and fail loud:\n  %s", strings.Join(problems, "\n  "))
	}
}

// TestCIRecipes_GitHubActions_RenderedParsesAsWorkflowYAML proves CLM-036.
//
// The document is decoded through yaml.Node and its top-level KEYS are read AS
// WRITTEN: YAML 1.1 resolves an unquoted `on` to the boolean true, so decoding
// into a struct field named On would silently paper over an absent key.
func TestCIRecipes_GitHubActions_RenderedParsesAsWorkflowYAML(t *testing.T) {
	_, _, rendered := ciApplyProbe(t, ciRecipeGitHubActions)

	doc := ciDecodeYAML(t, rendered)
	for _, key := range []string{"on", "jobs"} {
		if _, present := doc[key]; !present {
			t.Fatalf("the rendered workflow declares no top-level %q key (declares %v)\nrendered:\n%s", key, ciKeys(doc), rendered)
		}
	}

	jobs, wellShaped := doc["jobs"].(map[string]any)
	if !wellShaped || len(jobs) == 0 {
		t.Fatalf("the rendered workflow's `jobs` is not a non-empty mapping\nrendered:\n%s", rendered)
	}

	complete := false
	for name, raw := range jobs {
		job, isMapping := raw.(map[string]any)
		if !isMapping {
			t.Errorf("job %q is not a mapping", name)
			continue
		}
		if _, hasRunsOn := job["runs-on"]; !hasRunsOn {
			continue
		}
		steps, hasSteps := job["steps"].([]any)
		if hasSteps && len(steps) > 0 {
			complete = true
		}
	}
	if !complete {
		t.Errorf("no job declares both `runs-on` and a non-empty `steps` list\nrendered:\n%s", rendered)
	}
}

// ciKeys lists a decoded document's top-level keys for a failure message.
func ciKeys(doc map[string]any) []string {
	keys := []string{}
	for key := range doc {
		keys = append(keys, key)
	}
	return keys
}

// TestCIRecipes_GitHubActions_RenderedCarriesNoResidualPlaceholder proves
// CLM-040 (absence).
func TestCIRecipes_GitHubActions_RenderedCarriesNoResidualPlaceholder(t *testing.T) {
	_, _, rendered := ciApplyProbe(t, ciRecipeGitHubActions)
	if problems := ciResidualPlaceholders(rendered); len(problems) != 0 {
		t.Errorf("the rendered GitHub Actions workflow carries a residual placeholder:\n  %s", strings.Join(problems, "\n  "))
	}
}

// TestCIRecipes_GitHubActions_ReapplyIsByteIdentical proves CLM-045.
func TestCIRecipes_GitHubActions_ReapplyIsByteIdentical(t *testing.T) {
	if problems := ciReapplyProblems(t, ciRecipeGitHubActions); len(problems) != 0 {
		t.Errorf("re-applying github-actions-gate is not byte-identical and write-free:\n  %s", strings.Join(problems, "\n  "))
	}
}

// TestCIRecipes_GitHubActions_RulesScopedByTargetBasenameGlob proves CLM-053.
func TestCIRecipes_GitHubActions_RulesScopedByTargetBasenameGlob(t *testing.T) {
	if problems := ciGlobScopingProblems(t, ciRecipeGitHubActions, []string{"backstop-gate*.yml"}); len(problems) != 0 {
		t.Errorf("the github-actions rules are not scoped by their target basename glob:\n  %s", strings.Join(problems, "\n  "))
	}
}

// TestCIRecipes_GitHubActions_RenderedCarriesNoToolchainOrConsumerLiteral proves
// CLM-059 (absence). `actions/checkout` is the ONE permitted action literal and
// the permission is CLOSED to it — a second `actions/*` fails.
func TestCIRecipes_GitHubActions_RenderedCarriesNoToolchainOrConsumerLiteral(t *testing.T) {
	_, _, rendered := ciApplyProbe(t, ciRecipeGitHubActions)
	if problems := ciTravelProblems(rendered, []string{ciGitHubPermittedAction}); len(problems) != 0 {
		t.Errorf("the rendered GitHub Actions workflow does not travel:\n  %s", strings.Join(problems, "\n  "))
	}

	// THE CLOSED PERMISSION, ASSERTED DIRECTLY: the workflow USES exactly one
	// action, and it is checkout.
	//
	// Counted over `uses:` steps rather than over occurrences of the substring
	// `actions/`. A workflow that NAMES the permitted action in a comment —
	// which this payload does, explaining why the reference is there at all —
	// has not referenced a second action, and a substring count cannot tell the
	// two apart. What REQ-010 forbids is a second action being USED.
	text := string(rendered)
	used := []string{}
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		trimmed = strings.TrimPrefix(trimmed, "- ")
		if strings.HasPrefix(trimmed, "uses:") {
			used = append(used, strings.TrimSpace(strings.TrimPrefix(trimmed, "uses:")))
		}
	}
	if len(used) != 1 {
		t.Errorf("the rendered workflow uses %d action(s) (%v), want exactly 1 (%s)\nrendered:\n%s",
			len(used), used, ciGitHubPermittedAction, text)
	}
	if len(used) == 1 && !strings.HasPrefix(used[0], ciGitHubPermittedAction) {
		t.Errorf("the rendered workflow uses %q, want %s — the only expression of `fetch-depth: 0`\nrendered:\n%s", used[0], ciGitHubPermittedAction, text)
	}
}
