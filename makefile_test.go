package backstopcore_test

import (
	"regexp"
	"strings"
	"testing"
)

// This is a REPO-STRUCTURAL assertion, so it lives at the root beside
// workflows_test.go, module_path_test.go, pack_fleet_test.go and
// release_config_test.go rather than inside a package. The Makefile is repo
// structure, and nothing asserted on it before this.

const makefilePath = "Makefile"

// makeRule is one Makefile rule: its prerequisites and its recipe lines.
type makeRule struct {
	prerequisites []string
	recipe        []string
}

// makeRulePattern matches a rule line — a target list, a colon, and the
// prerequisites. Variable assignments (`NAME := value`) also carry a colon, so
// the caller rejects any match whose remainder begins with `=`.
var makeRulePattern = regexp.MustCompile(`^([A-Za-z0-9_.\-/ ]+):(.*)$`)

// parseMakefile returns every rule keyed by target name. Recipe lines are the
// tab-indented lines following a rule.
func parseMakefile(t *testing.T, source string) map[string]makeRule {
	t.Helper()

	rules := map[string]makeRule{}
	current := ""
	for _, line := range strings.Split(source, "\n") {
		if strings.HasPrefix(line, "\t") {
			if current != "" {
				rule := rules[current]
				rule.recipe = append(rule.recipe, strings.TrimPrefix(line, "\t"))
				rules[current] = rule
			}
			continue
		}
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		match := makeRulePattern.FindStringSubmatch(trimmed)
		if match == nil || strings.HasPrefix(match[2], "=") {
			// A variable assignment or a continuation — not a rule.
			current = ""
			continue
		}
		targets := strings.Fields(match[1])
		if len(targets) != 1 {
			current = ""
			continue
		}
		current = targets[0]
		rules[current] = makeRule{prerequisites: strings.Fields(match[2])}
	}
	if len(rules) == 0 {
		t.Fatalf("%s: parsed zero rules — every assertion below would be vacuous", makefilePath)
	}
	return rules
}

// TestMakefile_DeclaresOptInBaselineTargetOutsideTestAndCI pins the remedy half
// of ISSUE-176's second fix: the fetch is RUNNABLE, and it is OPT-IN.
// (CLM-006)
//
// The "documented local prerequisite" direction only works if the prerequisite
// can actually be run. `make baseline` is that; `test:` and `ci:` deliberately
// do not depend on it, because no local `go test` run may hit the network by
// default.
func TestMakefile_DeclaresOptInBaselineTargetOutsideTestAndCI(t *testing.T) {
	source := readWorkflowSource(t, makefilePath)
	rules := parseMakefile(t, source)

	baseline, declared := rules["baseline"]
	if !declared {
		t.Fatalf("%s declares no `baseline:` target. The absent-baseline error names `make baseline` as the "+
			"remedy, so the target has to exist or the message points at nothing (ISSUE-176)", makefilePath)
	}

	recipe := strings.Join(baseline.recipe, "\n")
	if !strings.Contains(recipe, "baseline pull") {
		t.Errorf("%s: the `baseline:` recipe does not invoke `baseline pull`:\n%s", makefilePath, recipe)
	}

	// `build` IS REQUIRED, not decoration: the recipe runs ./bin/backstop, and
	// bin/ is untracked and removed by `make clean`. On a clean tree a bare
	// target fails with "no such file or directory" — the exact class of
	// unhelpful error this lane exists to remove.
	if !containsString(baseline.prerequisites, "build") {
		t.Errorf("%s: `baseline:` lists prerequisites %v, which must include `build`. The recipe runs "+
			"./bin/backstop, and bin/ is untracked and removed by `make clean`, so on a clean tree the target "+
			"fails with `no such file or directory` — a remedy that cannot run is not a remedy",
			makefilePath, baseline.prerequisites)
	}

	phony, declaredPhony := rules[".PHONY"]
	if !declaredPhony {
		t.Fatalf("%s declares no .PHONY rule", makefilePath)
	}
	if !containsString(phony.prerequisites, "baseline") {
		t.Errorf("%s: .PHONY lists %v, which must include `baseline` — the target produces no file of that name",
			makefilePath, phony.prerequisites)
	}

	// ★ THE OPT-IN FENCE. This is the assertion the whole direction rests on,
	// and it is the change most likely to look like a helpful follow-up later.
	for _, target := range []string{"test", "ci"} {
		rule, present := rules[target]
		if !present {
			t.Fatalf("%s declares no `%s:` target — this assertion would be vacuous", makefilePath, target)
		}
		if containsString(rule.prerequisites, "baseline") {
			t.Errorf("%s: `%s:` lists `baseline` among its prerequisites %v. The remedy is deliberately OPT-IN: "+
				"wiring it in makes every `make %s` shell `backstop baseline pull` and hit the network, which is "+
				"the hard constraint ISSUE-176 names", makefilePath, target, rule.prerequisites, target)
		}
	}
}
