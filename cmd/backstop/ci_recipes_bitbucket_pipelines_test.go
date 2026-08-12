package main

// SPEC-067 — the twelve Bitbucket Pipelines claims.

import (
	"strings"
	"testing"
)

const ciBitbucketTarget = "bitbucket-pipelines.yml"

// TestCIRecipes_BitbucketPipelines_RecipeDeclarationShape proves CLM-010.
func TestCIRecipes_BitbucketPipelines_RecipeDeclarationShape(t *testing.T) {
	if problems := ciRecipeShapeProblems(t, ciRecipeBitbucketPipelines, ciBitbucketTarget); len(problems) != 0 {
		t.Errorf("the bitbucket-pipelines-gate recipe declaration is not the shape REQ-003 fixes:\n  %s", strings.Join(problems, "\n  "))
	}
}

// TestCIRecipes_BitbucketPipelines_ApplyWritesOnlyItsDeclaredTarget proves
// CLM-014.
func TestCIRecipes_BitbucketPipelines_ApplyWritesOnlyItsDeclaredTarget(t *testing.T) {
	added, want := ciAppliedFileDelta(t, ciRecipeBitbucketPipelines)
	if strings.Join(added, "\n") != strings.Join(want, "\n") {
		t.Errorf("applying bitbucket-pipelines-gate added %v, want exactly %v", added, want)
	}
}

// TestCIRecipes_BitbucketPipelines_RenderedDeclaresFullHistory proves CLM-018:
// a `clone` block whose `depth` is `full`, asserted through a YAML walk.
func TestCIRecipes_BitbucketPipelines_RenderedDeclaresFullHistory(t *testing.T) {
	_, _, rendered := ciApplyProbe(t, ciRecipeBitbucketPipelines)

	doc := ciDecodeYAML(t, rendered)
	clone, declared := doc["clone"].(map[string]any)
	if !declared {
		t.Fatalf("the rendered pipeline declares no top-level `clone` block\nrendered:\n%s", rendered)
	}
	depth, hasDepth := clone["depth"]
	if !hasDepth {
		t.Fatalf("the rendered pipeline's `clone` block declares no `depth`\nrendered:\n%s", rendered)
	}
	if depth != "full" {
		t.Errorf("the rendered pipeline declares `clone.depth: %v`, want `full`; a shallow clone makes the diff base unreachable\nrendered:\n%s", depth, rendered)
	}
}

// TestCIRecipes_BitbucketPipelines_RenderedInstallsPinnedBackstop proves CLM-022.
func TestCIRecipes_BitbucketPipelines_RenderedInstallsPinnedBackstop(t *testing.T) {
	_, _, rendered := ciApplyProbe(t, ciRecipeBitbucketPipelines)
	if problems := ciPinnedInstallProblems(rendered); len(problems) != 0 {
		t.Errorf("the rendered Bitbucket Pipelines config does not install a pinned backstop:\n  %s", strings.Join(problems, "\n  "))
	}
}

// TestCIRecipes_BitbucketPipelines_PackInstallPrecedesGate proves CLM-026.
//
// ★ THE ANCHOR/ALIAS IS WHY THE COUNTS ARE ONE. The payload defines the gate step
// ONCE and aliases it into both the pull-request and branch pipelines, which is
// what keeps `backstop pack install` and `backstop gate` at exactly one literal
// occurrence each. The counts and offsets are asserted on the RAW BYTES because
// the claim is about byte offsets — and then, separately, the PARSED document is
// checked to confirm the aliased step really is the step BOTH pipelines run.
// An anchor is exactly where byte order and EXECUTION order could diverge.
func TestCIRecipes_BitbucketPipelines_PackInstallPrecedesGate(t *testing.T) {
	_, _, rendered := ciApplyProbe(t, ciRecipeBitbucketPipelines)

	install := ciByteOffset(t, rendered, "backstop pack install")
	gate := ciByteOffset(t, rendered, "backstop gate")
	if install >= gate {
		t.Errorf("`backstop pack install` is at byte %d and `backstop gate` at byte %d; the install must precede the gate\nrendered:\n%s", install, gate, rendered)
	}

	// EXECUTION order, through the parsed document: every pipeline's step runs a
	// script list in which the install precedes the gate.
	doc := ciDecodeYAML(t, rendered)
	pipelines, declared := doc["pipelines"].(map[string]any)
	if !declared {
		t.Fatalf("the rendered pipeline declares no `pipelines` block\nrendered:\n%s", rendered)
	}

	checked := 0
	ciWalkPipelineScripts(pipelines, func(script []any) {
		checked++
		installAt, gateAt := -1, -1
		for index, raw := range script {
			line, isString := raw.(string)
			if !isString {
				continue
			}
			if installAt < 0 && strings.Contains(line, "backstop pack install") {
				installAt = index
			}
			if gateAt < 0 && strings.Contains(line, "backstop gate") {
				gateAt = index
			}
		}
		if installAt < 0 || gateAt < 0 {
			t.Errorf("a pipeline's script list runs install=%d gate=%d; both must appear in the executed step", installAt, gateAt)
			return
		}
		if installAt >= gateAt {
			t.Errorf("in the EXECUTED script list `backstop pack install` is at index %d and `backstop gate` at %d; byte order and execution order disagree", installAt, gateAt)
		}
	})
	if checked == 0 {
		t.Errorf("no pipeline step with a script list was found, so execution order was never checked\nrendered:\n%s", rendered)
	}
}

// ciWalkPipelineScripts visits every `script:` list reachable under a decoded
// `pipelines:` block, whatever nesting of branches / pull-requests / steps the
// author chose.
func ciWalkPipelineScripts(node any, visit func([]any)) {
	switch typed := node.(type) {
	case map[string]any:
		for key, value := range typed {
			if key == "script" {
				if script, isList := value.([]any); isList {
					visit(script)
					continue
				}
			}
			ciWalkPipelineScripts(value, visit)
		}
	case []any:
		for _, item := range typed {
			ciWalkPipelineScripts(item, visit)
		}
	}
}

// TestCIRecipes_BitbucketPipelines_GateVerdictNotSwallowed proves CLM-030
// (absence): the universal three, with no platform-specific term added.
func TestCIRecipes_BitbucketPipelines_GateVerdictNotSwallowed(t *testing.T) {
	_, _, rendered := ciApplyProbe(t, ciRecipeBitbucketPipelines)
	if problems := ciSwallowFormsFound(rendered, []string{"|| true", "|| exit 0"}); len(problems) != 0 {
		t.Errorf("the rendered Bitbucket Pipelines config swallows the gate verdict:\n  %s", strings.Join(problems, "\n  "))
	}
}

// TestCIRecipes_BitbucketPipelines_ResolvesDiffBaseAndFailsWhenUnresolvable
// proves CLM-034.
func TestCIRecipes_BitbucketPipelines_ResolvesDiffBaseAndFailsWhenUnresolvable(t *testing.T) {
	_, _, rendered := ciApplyProbe(t, ciRecipeBitbucketPipelines)
	// One platform variable, because one is what Bitbucket honestly supplies:
	// BITBUCKET_PR_DESTINATION_BRANCH carries the target branch on a
	// pull-request pipeline and is unset on a branch pipeline, where the
	// declared default_branch IS the right base. Demanding a second variable
	// here would force the payload to read one it has no use for.
	if problems := ciBaseResolutionProblems(t, rendered, []string{"BITBUCKET_PR_DESTINATION_BRANCH"}, "main"); len(problems) != 0 {
		t.Errorf("the rendered Bitbucket Pipelines config does not resolve a diff base and fail loud:\n  %s", strings.Join(problems, "\n  "))
	}
}

// TestCIRecipes_BitbucketPipelines_RenderedParsesAsPipelinesYAML proves CLM-038.
func TestCIRecipes_BitbucketPipelines_RenderedParsesAsPipelinesYAML(t *testing.T) {
	_, _, rendered := ciApplyProbe(t, ciRecipeBitbucketPipelines)

	doc := ciDecodeYAML(t, rendered)
	pipelines, declared := doc["pipelines"].(map[string]any)
	if !declared || len(pipelines) == 0 {
		t.Fatalf("the rendered file declares no non-empty `pipelines` block\nrendered:\n%s", rendered)
	}

	found := false
	for name, value := range pipelines {
		if name != "branches" && name != "pull-requests" && name != "default" {
			continue
		}
		ciWalkPipelineScripts(value, func(script []any) {
			if len(script) > 0 {
				found = true
			}
		})
	}
	if !found {
		t.Errorf("no branch or pull-request pipeline declares a step with a non-empty `script` list\nrendered:\n%s", rendered)
	}
}

// TestCIRecipes_BitbucketPipelines_RenderedCarriesNoResidualPlaceholder proves
// CLM-042.
func TestCIRecipes_BitbucketPipelines_RenderedCarriesNoResidualPlaceholder(t *testing.T) {
	_, _, rendered := ciApplyProbe(t, ciRecipeBitbucketPipelines)
	if problems := ciResidualPlaceholders(rendered); len(problems) != 0 {
		t.Errorf("the rendered Bitbucket Pipelines config carries a residual placeholder:\n  %s", strings.Join(problems, "\n  "))
	}
}

// TestCIRecipes_BitbucketPipelines_ReapplyIsByteIdentical proves CLM-047.
func TestCIRecipes_BitbucketPipelines_ReapplyIsByteIdentical(t *testing.T) {
	if problems := ciReapplyProblems(t, ciRecipeBitbucketPipelines); len(problems) != 0 {
		t.Errorf("re-applying bitbucket-pipelines-gate is not byte-identical and write-free:\n  %s", strings.Join(problems, "\n  "))
	}
}

// TestCIRecipes_BitbucketPipelines_RulesScopedByTargetBasenameGlob proves
// CLM-055.
func TestCIRecipes_BitbucketPipelines_RulesScopedByTargetBasenameGlob(t *testing.T) {
	if problems := ciGlobScopingProblems(t, ciRecipeBitbucketPipelines,
		[]string{"bitbucket-pipelines*.yml"}); len(problems) != 0 {
		t.Errorf("the bitbucket-pipelines rules are not scoped by their target basename glob:\n  %s", strings.Join(problems, "\n  "))
	}
}

// TestCIRecipes_BitbucketPipelines_RenderedCarriesNoToolchainOrConsumerLiteral
// proves CLM-061 (absence). No Bitbucket pipe is referenced, so the permitted
// org/repo literal set is the release coordinate alone.
func TestCIRecipes_BitbucketPipelines_RenderedCarriesNoToolchainOrConsumerLiteral(t *testing.T) {
	_, _, rendered := ciApplyProbe(t, ciRecipeBitbucketPipelines)
	if problems := ciTravelProblems(rendered, nil); len(problems) != 0 {
		t.Errorf("the rendered Bitbucket Pipelines config does not travel:\n  %s", strings.Join(problems, "\n  "))
	}
}
