package main

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/check"
)

// systemSemgrepEnsurer resolves the semgrep binary already on PATH, so the
// enforcement-transfer test runs a REAL semgrep pass without downloading a
// pinned binary. If semgrep is not installed the test skips — the load-bearing
// assertion needs a real engine.
type systemSemgrepEnsurer struct {
	path string
}

func (e systemSemgrepEnsurer) EnsureSemgrep(_, _ string) (string, error) {
	return e.path, nil
}

// goStandardsRuleConfigs resolves the installed backstop/go-standards pack's
// layer-2 rule --config paths via the production loadInstalledPacks +
// mergePackRules path. This is exactly the rule set the gate/code-check semgrep
// pass receives, so the test proves the CONSUMED pack enforces.
func goStandardsRuleConfigs(t *testing.T, root string) []string {
	t.Helper()
	packs, err := loadInstalledPacks(root)
	if err != nil {
		t.Fatalf("loadInstalledPacks: %v", err)
	}
	var target int
	found := false
	for i, m := range packs {
		if m.NormalizedName == dogfoodPackName {
			target, found = i, true
		}
	}
	if !found {
		t.Fatalf("installed packs do not include %q; got %d packs", dogfoodPackName, len(packs))
	}
	configs, mergeErr := mergePackRules(packs[target:target+1], filepath.Join(root, ".backstop", "packs"))
	if mergeErr != nil {
		t.Fatalf("mergePackRules: %v", mergeErr)
	}
	if len(configs) == 0 {
		t.Fatalf("mergePackRules returned no --config paths for %q; the pack rule tree is missing", dogfoodPackName)
	}
	return configs
}

// TestDogfoodPack_FlagsKnownGoViolation is the load-bearing CLM-025
// enforcement-transfer proof: with backstop/go-standards consumed as a pack
// (declared, installed, locked), a code-check / gate semgrep pass over a
// self-contained known-bad Go fixture that violates the pack's GO-060
// hardcoded-credentials rule produces at least one semgrep Violation referencing
// that rule. The negative control — the clean fixture loading its credential
// from the environment — produces NO violation of that rule, so a mis-wired
// config that flags everything fails as surely as a dropped config that flags
// nothing.
func TestDogfoodPack_FlagsKnownGoViolation(t *testing.T) {
	semgrepBin, lookErr := exec.LookPath("semgrep")
	if lookErr != nil {
		t.Skip("semgrep not installed on PATH; enforcement-transfer proof needs a real engine")
	}

	root := repoRoot(t)
	ruleConfigs := goStandardsRuleConfigs(t, root)

	const go060RuleSuffix = "go.security.no-hardcoded-credentials"
	fixtureDir := filepath.Join(root, "cmd", "backstop", "testdata", "dogfood_enforcement")

	// runFixture drives the production semgrep executor through check.RunWith
	// over a single fixture with the consumed pack's rule --config set, returning
	// the semgrep-pass violations. Inlined as a closure so the enforcement-claim
	// assertions call pkg/check directly within this test body.
	runFixture := func(fixture string) []check.Violation {
		result, runErr := check.RunWith(context.Background(), check.RunOptions{
			Options: check.Options{
				Mode:                check.ScopeModeFile,
				FilePath:            fixture,
				BackstopDir:         filepath.Dir(filepath.Dir(fixture)),
				ProjectDir:          filepath.Dir(fixture),
				ExtraSemgrepConfigs: ruleConfigs,
			},
			SemgrepEnsurer: systemSemgrepEnsurer{path: semgrepBin},
		})
		if runErr != nil {
			t.Fatalf("check.RunWith over %s: %v", fixture, runErr)
		}
		var semgrep []check.Violation
		for _, v := range result.AllViolations() {
			if v.Pass == check.CheckTypeSemgrep {
				semgrep = append(semgrep, v)
			}
		}
		return semgrep
	}

	// POSITIVE: the known-bad fixture must be flagged by the GO-060 pack rule.
	badViolations := runFixture(filepath.Join(fixtureDir, "known_bad.go"))
	flaggedByGo060 := false
	for _, v := range badViolations {
		if strings.Contains(v.Rule, go060RuleSuffix) {
			flaggedByGo060 = true
		}
	}
	if !flaggedByGo060 {
		t.Fatalf("known-bad fixture was NOT flagged by the consumed pack's %s rule; enforcement did not transfer. semgrep violations: %+v", go060RuleSuffix, badViolations)
	}

	// NEGATIVE CONTROL: the clean fixture must NOT be flagged by the GO-060 rule.
	cleanViolations := runFixture(filepath.Join(fixtureDir, "clean.go"))
	for _, v := range cleanViolations {
		if strings.Contains(v.Rule, go060RuleSuffix) {
			t.Errorf("clean control fixture was flagged by %s; the semgrep config is mis-wired (flags everything): %+v", go060RuleSuffix, v)
		}
	}
}
