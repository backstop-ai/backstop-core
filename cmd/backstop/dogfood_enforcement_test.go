package main

import (
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/check"
	"github.com/bmanson/backstop-core/pkg/pack"
	"github.com/bmanson/backstop-core/pkg/pack/engine"
)

// goStandardsRuleConfigs resolves the installed backstop/go-standards pack's
// engine:semgrep rule --config paths via the production loadInstalledPacks +
// the same per-input_mode gathering dispatchPackEngines uses. This is exactly
// the rule set the gate dispatches to the semgrep engine, so the test proves
// the CONSUMED pack enforces.
func goStandardsRuleConfigs(t *testing.T, root string) []string {
	t.Helper()
	packs, err := loadInstalledPacks(root)
	if err != nil {
		t.Fatalf("loadInstalledPacks: %v", err)
	}
	var manifest *pack.Manifest
	for _, m := range packs {
		if m.NormalizedName == dogfoodPackName {
			manifest = m
		}
	}
	if manifest == nil {
		t.Fatalf("installed packs do not include %q; got %d packs", dogfoodPackName, len(packs))
	}
	packRoot := filepath.Join(root, ".backstop", "packs", filepath.FromSlash(manifest.NormalizedName))
	seen := map[string]struct{}{}
	var configs []string
	for _, rule := range manifest.Content.Ruleset.Rules {
		if rule.Engine != "semgrep" {
			continue
		}
		abs, _ := filepath.Abs(filepath.Join(packRoot, filepath.FromSlash(rule.RulePath)))
		if _, dup := seen[abs]; dup {
			continue
		}
		seen[abs] = struct{}{}
		configs = append(configs, abs)
	}
	sort.Strings(configs)
	if len(configs) == 0 {
		t.Fatalf("no engine:semgrep rule paths for %q; the pack rule tree is missing or unmigrated", dogfoodPackName)
	}
	// Guard: the registry must classify these as the semgrep rule-flags engine,
	// matching the dispatch path's gathering.
	if b, lookErr := engine.DefaultRegistry().Lookup("semgrep"); lookErr != nil || b.InputMode != engine.InputModeRuleFlags {
		t.Fatalf("semgrep engine binding missing or not rule-flags: %v", lookErr)
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

	// runFixture runs a REAL semgrep pass directly over a single fixture with the
	// consumed pack's engine:semgrep rule --config set — the same paths the gate
	// dispatches to the semgrep engine — and parses the JSON findings via the
	// production parseSemgrepJSON path through check.ParsePackFindings is not
	// applicable here (semgrep emits its own JSON, not SARIF), so we run semgrep
	// --json and parse with the production semgrep executor by feeding the
	// fixture as the sole scoped file. We invoke semgrep directly to avoid the
	// retired ExtraSemgrepConfigs option.
	runFixture := func(fixture string) []check.Violation {
		args := []string{"--json", "--quiet"}
		for _, cfg := range ruleConfigs {
			args = append(args, "--config", cfg)
		}
		args = append(args, fixture)
		out, _ := exec.Command(semgrepBin, args...).Output()
		violations, parseErr := check.ParseSemgrepJSONForTest(out)
		if parseErr != nil {
			t.Fatalf("parse semgrep json over %s: %v", fixture, parseErr)
		}
		return violations
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
