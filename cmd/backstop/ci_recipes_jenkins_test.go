package main

// SPEC-067 — the twelve Jenkins claims.

import (
	"strings"
	"testing"
)

const ciJenkinsTarget = "Jenkinsfile"

// TestCIRecipes_Jenkins_RecipeDeclarationShape proves CLM-011.
func TestCIRecipes_Jenkins_RecipeDeclarationShape(t *testing.T) {
	if problems := ciRecipeShapeProblems(t, ciRecipeJenkins, ciJenkinsTarget); len(problems) != 0 {
		t.Errorf("the jenkins-gate recipe declaration is not the shape REQ-003 fixes:\n  %s", strings.Join(problems, "\n  "))
	}
}

// TestCIRecipes_Jenkins_ApplyWritesOnlyItsDeclaredTarget proves CLM-015.
func TestCIRecipes_Jenkins_ApplyWritesOnlyItsDeclaredTarget(t *testing.T) {
	added, want := ciAppliedFileDelta(t, ciRecipeJenkins)
	if strings.Join(added, "\n") != strings.Join(want, "\n") {
		t.Errorf("applying jenkins-gate added %v, want exactly %v", added, want)
	}
}

// TestCIRecipes_Jenkins_RenderedDeclaresFullHistory proves CLM-019: the checkout
// step carries a `CloneOption` whose `shallow` setting is false.
//
// A TEXT assertion, deliberately: a Jenkinsfile is Groovy and there is no parser
// to walk. The check requires both the CloneOption class reference and the
// `shallow: false` setting, and rejects `shallow: true` outright.
func TestCIRecipes_Jenkins_RenderedDeclaresFullHistory(t *testing.T) {
	_, _, rendered := ciApplyProbe(t, ciRecipeJenkins)

	text := string(rendered)
	if !strings.Contains(text, "CloneOption") {
		t.Fatalf("the rendered Jenkinsfile declares no `CloneOption` checkout extension\nrendered:\n%s", text)
	}
	if !strings.Contains(text, "shallow: false") {
		t.Errorf("the rendered Jenkinsfile's CloneOption does not declare `shallow: false`; a shallow clone makes the diff base unreachable\nrendered:\n%s", text)
	}
	if strings.Contains(text, "shallow: true") {
		t.Errorf("the rendered Jenkinsfile declares `shallow: true`\nrendered:\n%s", text)
	}
}

// TestCIRecipes_Jenkins_RenderedInstallsPinnedBackstop proves CLM-023.
func TestCIRecipes_Jenkins_RenderedInstallsPinnedBackstop(t *testing.T) {
	_, _, rendered := ciApplyProbe(t, ciRecipeJenkins)
	if problems := ciPinnedInstallProblems(rendered); len(problems) != 0 {
		t.Errorf("the rendered Jenkinsfile does not install a pinned backstop:\n  %s", strings.Join(problems, "\n  "))
	}
}

// TestCIRecipes_Jenkins_PackInstallPrecedesGate proves CLM-027. A declarative
// pipeline's stages execute in declaration order, so byte order and execution
// order agree — asserted here by checking that the gate stage's `sh` block runs
// the install before the gate within one shell script.
func TestCIRecipes_Jenkins_PackInstallPrecedesGate(t *testing.T) {
	_, _, rendered := ciApplyProbe(t, ciRecipeJenkins)

	install := ciByteOffset(t, rendered, "backstop pack install")
	gate := ciByteOffset(t, rendered, "backstop gate")
	if install >= gate {
		t.Errorf("`backstop pack install` is at byte %d and `backstop gate` at byte %d; the install must precede the gate\nrendered:\n%s", install, gate, rendered)
	}

	// Stage ORDER is execution order. The stage containing the install must be at
	// or before the stage containing the gate.
	stages := ciJenkinsStageOffsets(string(rendered))
	if len(stages) == 0 {
		t.Fatalf("the rendered Jenkinsfile declares no `stage(` at all\nrendered:\n%s", rendered)
	}
	installStage := ciEnclosingStage(stages, install)
	gateStage := ciEnclosingStage(stages, gate)
	if installStage > gateStage {
		t.Errorf("`backstop pack install` runs in stage index %d while `backstop gate` runs in stage index %d; execution order contradicts byte order", installStage, gateStage)
	}
}

// ciJenkinsStageOffsets returns the byte offset of every `stage(` declaration,
// in declaration order.
func ciJenkinsStageOffsets(text string) []int {
	offsets := []int{}
	for index := 0; ; {
		found := strings.Index(text[index:], "stage(")
		if found < 0 {
			return offsets
		}
		offsets = append(offsets, index+found)
		index += found + len("stage(")
	}
}

// ciEnclosingStage returns the index of the last stage declared at or before
// offset, or -1 when the offset precedes every stage.
func ciEnclosingStage(stages []int, offset int) int {
	enclosing := -1
	for index, start := range stages {
		if start <= offset {
			enclosing = index
		}
	}
	return enclosing
}

// TestCIRecipes_Jenkins_GateVerdictNotSwallowed proves CLM-031 (absence): the
// universal three plus `returnStatus: true` and `catchError`, which are the two
// Jenkins forms that turn a non-zero `sh` into a passing build.
func TestCIRecipes_Jenkins_GateVerdictNotSwallowed(t *testing.T) {
	_, _, rendered := ciApplyProbe(t, ciRecipeJenkins)
	if problems := ciSwallowFormsFound(rendered,
		[]string{"|| true", "|| exit 0", "returnStatus: true", "catchError"}); len(problems) != 0 {
		t.Errorf("the rendered Jenkinsfile swallows the gate verdict:\n  %s", strings.Join(problems, "\n  "))
	}
}

// TestCIRecipes_Jenkins_ResolvesDiffBaseAndFailsWhenUnresolvable proves CLM-035.
//
// Groovy's `${...}` is SINGLE-brace and therefore invisible to the `{{ }}`
// substituter, which is why the environment references can be written directly.
func TestCIRecipes_Jenkins_ResolvesDiffBaseAndFailsWhenUnresolvable(t *testing.T) {
	_, _, rendered := ciApplyProbe(t, ciRecipeJenkins)
	// One platform variable, because one is what Jenkins honestly supplies for
	// this purpose: CHANGE_TARGET carries the target branch on a change
	// request and is unset otherwise, where the declared default_branch IS the
	// right base. BRANCH_NAME would add a read the resolution has no use for,
	// and a payload that reads a variable it does not act on is decoration.
	if problems := ciBaseResolutionProblems(t, rendered, []string{"CHANGE_TARGET"}, "main"); len(problems) != 0 {
		t.Errorf("the rendered Jenkinsfile does not resolve a diff base and fail loud:\n  %s", strings.Join(problems, "\n  "))
	}
}

// TestCIRecipes_Jenkins_RenderedIsStructurallyWellFormedDeclarativePipeline
// proves CLM-039 — a STRUCTURAL check, deliberately NOT a Groovy parse.
//
// WHY IT IS WEAKER THAN THE OTHER THREE, recorded rather than hidden (Sharp Edge
// 4): no Groovy parser is reachable from a Go test without a JVM dependency this
// repository does not carry. A pass here means "the Jenkinsfile is not obviously
// malformed", NOT "the Jenkinsfile is valid". It cannot catch an unbalanced
// quote, an invalid step argument, or anything a real parser would.
//
// Braces are counted OUTSIDE string literals and comments — a `{` inside a
// double- or single-quoted Groovy string, or in a `//` comment, is not a block
// delimiter, and counting it would make this check fail on correct input. That
// scan is itself approximate (it does not model Groovy's triple-quoted strings
// beyond treating them as ordinary quotes), which is the honest limit of a
// parser-free check.
func TestCIRecipes_Jenkins_RenderedIsStructurallyWellFormedDeclarativePipeline(t *testing.T) {
	_, _, rendered := ciApplyProbe(t, ciRecipeJenkins)
	text := string(rendered)

	depth, balanced := ciBraceBalance(text)
	if !balanced {
		t.Errorf("the rendered Jenkinsfile's braces do not balance (ends at depth %d), counting only braces outside strings and comments\nrendered:\n%s", depth, text)
	}

	for _, block := range []string{"pipeline", "agent", "stages", "stage(", "steps"} {
		if !strings.Contains(text, block) {
			t.Errorf("the rendered Jenkinsfile declares no %q block\nrendered:\n%s", block, text)
		}
	}
}

// ciBraceBalance counts brace depth outside string literals and line comments.
// It returns the final depth and whether the file balanced without ever going
// negative.
func ciBraceBalance(text string) (int, bool) {
	depth := 0
	inSingle, inDouble, inComment := false, false, false

	for index := 0; index < len(text); index++ {
		char := text[index]

		if inComment {
			if char == '\n' {
				inComment = false
			}
			continue
		}
		if inSingle {
			switch char {
			case '\\':
				index++
			case '\'':
				inSingle = false
			}
			continue
		}
		if inDouble {
			switch char {
			case '\\':
				index++
			case '"':
				inDouble = false
			}
			continue
		}
		if char == '/' && index+1 < len(text) && text[index+1] == '/' {
			inComment = true
			index++
			continue
		}

		switch char {
		// A `#` opens a SHELL comment inside an `sh` block. Treated as a line
		// comment for the same reason `//` is: its braces are not Groovy blocks.
		case '#':
			inComment = true
		case '\'':
			inSingle = true
		case '"':
			inDouble = true
		case '{':
			depth++
		case '}':
			depth--
			if depth < 0 {
				return depth, false
			}
		}
	}
	return depth, depth == 0
}

// TestCIRecipes_Jenkins_RenderedCarriesNoResidualPlaceholder proves CLM-043.
func TestCIRecipes_Jenkins_RenderedCarriesNoResidualPlaceholder(t *testing.T) {
	_, _, rendered := ciApplyProbe(t, ciRecipeJenkins)
	if problems := ciResidualPlaceholders(rendered); len(problems) != 0 {
		t.Errorf("the rendered Jenkinsfile carries a residual placeholder:\n  %s", strings.Join(problems, "\n  "))
	}
}

// TestCIRecipes_Jenkins_ReapplyIsByteIdentical proves CLM-048.
func TestCIRecipes_Jenkins_ReapplyIsByteIdentical(t *testing.T) {
	if problems := ciReapplyProblems(t, ciRecipeJenkins); len(problems) != 0 {
		t.Errorf("re-applying jenkins-gate is not byte-identical and write-free:\n  %s", strings.Join(problems, "\n  "))
	}
}

// TestCIRecipes_Jenkins_RulesScopedByTargetBasenameGlob proves CLM-056. The glob
// has NO extension, which is exactly what lets it match both the deployed
// `Jenkinsfile` and the `Jenkinsfile-*` fixtures.
func TestCIRecipes_Jenkins_RulesScopedByTargetBasenameGlob(t *testing.T) {
	if problems := ciGlobScopingProblems(t, ciRecipeJenkins, []string{"Jenkinsfile*"}); len(problems) != 0 {
		t.Errorf("the jenkins rules are not scoped by their target basename glob:\n  %s", strings.Join(problems, "\n  "))
	}
}

// TestCIRecipes_Jenkins_RenderedCarriesNoToolchainOrConsumerLiteral proves
// CLM-062 (absence). No Jenkins plugin coordinate is referenced, so the permitted
// org/repo literal set is the release coordinate alone — in particular the
// checkout step must carry no `userRemoteConfigs` URL, which would be a consumer
// repository literal. It derives everything from `scm` instead.
func TestCIRecipes_Jenkins_RenderedCarriesNoToolchainOrConsumerLiteral(t *testing.T) {
	_, _, rendered := ciApplyProbe(t, ciRecipeJenkins)
	if problems := ciTravelProblems(rendered, nil); len(problems) != 0 {
		t.Errorf("the rendered Jenkinsfile does not travel:\n  %s", strings.Join(problems, "\n  "))
	}

	text := string(rendered)
	if !strings.Contains(text, "scm.userRemoteConfigs") {
		t.Errorf("the rendered Jenkinsfile does not derive userRemoteConfigs from `scm`; a literal remote URL would be a consumer-repository literal\nrendered:\n%s", text)
	}
}
