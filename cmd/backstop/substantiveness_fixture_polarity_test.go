package main

// ISSUE-148 — fixture polarity for `packs/substantiveness`, pinned against the
// property packval ACTUALLY measures.
//
// THE NON-OBVIOUS PART, stated here because a future reader will otherwise "tidy"
// these fixtures straight back into the defect: both of this pack's rules ship in
// ONE `ast-grep/sgconfig.yml` (ISSUE-028), and phase3 dispatches that whole config
// per fixture. So packval's per-fixture verdict is "did ANY rule fire", not "did
// THIS rule fire". A declared POSITIVE (clean) fixture must therefore trigger
// NEITHER rule — which is why a clean fixture may not contain `t.Fatalf`:
// `t.Fatalf` is a selector_expression inside a `^Test`-named function, exactly what
// `referenced-symbol-go` matches. A clean fixture's assertion must be an UNQUALIFIED
// call whose identifier hits the hollow rule's assertion vocabulary (e.g.
// `mustEqual`), and helper functions — not being `^Test`-named — may hold `t.Fatalf`
// freely because `inside: {matches: is-test}` never reaches into them.

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/packval"
	"gopkg.in/yaml.v3"
)

// astGrepTool is the engine these fixtures are validated against. It matches the
// version `packs/substantiveness`'s own engine binding provisions.
const astGrepTool = "ast-grep"

// astGrepFinding is the slice of ast-grep's `--json` result shape these tests read:
// the firing rule id, and the extracted symbol metavariable used by the `t` discard.
type astGrepFinding struct {
	RuleID        string `json:"ruleId"`
	MetaVariables struct {
		Single map[string]struct {
			Text string `json:"text"`
		} `json:"single"`
	} `json:"metaVariables"`
}

func (f astGrepFinding) pkgText() string {
	return f.MetaVariables.Single["PKG"].Text
}

func (f astGrepFinding) fnText() string {
	return f.MetaVariables.Single["FN"].Text
}

// scanWithCombinedRuleset runs the pack's own combined ast-grep config over one file
// and returns every finding. It invokes the real binary on purpose: this is about the
// ENGINE's verdict, which is a different question from whether packval's wrapper
// reports pass — TestSubstantivenessFixtures_RealPackTestPassesPhase3 covers that, and
// the two must not collapse into one.
func scanWithCombinedRuleset(t *testing.T, sgconfig, target string) []astGrepFinding {
	t.Helper()
	// execCommand is the package's parametric dispatch (root_test.go): the tool name
	// travels as data rather than as a literal handed straight to exec.Command, which
	// is what the backstop-self no-baked-tool-exec rule requires.
	cmd := execCommand(astGrepTool, "scan", "--config", sgconfig, "--json", target)
	var stderr []byte
	out, err := cmd.Output()
	if ee, ok := err.(*exec.ExitError); ok {
		stderr = ee.Stderr
	}
	var findings []astGrepFinding
	if jsonErr := json.Unmarshal(out, &findings); jsonErr != nil {
		t.Fatalf("ast-grep scan --config %s %s: unparseable json (%v); run err=%v stderr=%s stdout=%s",
			sgconfig, target, jsonErr, err, stderr, out)
	}
	return findings
}

func describeFindings(findings []astGrepFinding) string {
	if len(findings) == 0 {
		return "(none)"
	}
	out, _ := json.Marshal(findings)
	return string(out)
}

// TestSubstantivenessFixtures_RealPackTestPassesPhase3 (CLM-001) drives the SAME
// packval pipeline `pack test` drives, against an ABSOLUTE pack directory (a relative
// packDir hits a separate, already-filed sandbox defect — ISSUE-147).
func TestSubstantivenessFixtures_RealPackTestPassesPhase3(t *testing.T) {
	requireAstGrepE2E(t)

	absPackDir := substantivenessSourceDir(repoRoot(t))
	result := packval.NewPipeline(absPackDir, packval.PipelineOptions{Mode: "test"}).Run()

	for _, e := range result.Errors {
		t.Logf("packval error: phase=%q check=%q rule=%q claim=%q message=%q",
			e.Phase, e.Check, e.Rule, e.Claim, e.Message)
	}
	if result.Status != "pass" {
		t.Errorf("pack test on %s: status = %q, want %q (%d errors, logged above)",
			absPackDir, result.Status, "pass", len(result.Errors))
	}

	// phase3-fixtures is asserted SPECIFICALLY. An overall pass that came from phase3
	// being skipped is exactly the vacuous shape this cluster of work exists to prevent.
	var phase3 *packval.PhaseResult
	for i := range result.Phases {
		if result.Phases[i].Phase == "phase3-fixtures" {
			phase3 = &result.Phases[i]
		}
	}
	if phase3 == nil {
		t.Fatalf("pack test on %s produced no phase3-fixtures phase result at all; phases = %+v",
			absPackDir, result.Phases)
	}
	if phase3.Status != "pass" {
		t.Errorf("phase3-fixtures status = %q, want %q (reason=%q)", phase3.Status, "pass", phase3.Reason)
	}
}

// TestSubstantivenessFixtures_PolarityHoldsAgainstTheCombinedRuleset (CLM-002) is
// DATA-DRIVEN from the manifest, so it keeps covering every declared fixture even if
// somebody adds one.
func TestSubstantivenessFixtures_PolarityHoldsAgainstTheCombinedRuleset(t *testing.T) {
	requireAstGrepE2E(t)

	absPackDir := substantivenessSourceDir(repoRoot(t))
	m, err := packval.ParseManifest(filepath.Join(absPackDir, "pack.yml"))
	if err != nil {
		t.Fatalf("parse manifest at %s: %v", absPackDir, err)
	}

	scanned := 0
	for _, rule := range m.Content.Ruleset.Rules {
		sgconfig := filepath.Join(absPackDir, rule.RuleSourcePath())
		for _, claim := range rule.Claims {
			for _, fx := range claim.Fixtures.Positive {
				scanned++
				t.Run("positive/"+fx.Path, func(t *testing.T) {
					findings := scanWithCombinedRuleset(t, sgconfig, filepath.Join(absPackDir, fx.Path))
					// A DECLARED POSITIVE is the CLEAN case and must trigger ZERO rules of
					// the shared sgconfig — not merely zero hits of its own rule.
					if len(findings) != 0 {
						t.Errorf("positive fixture %s triggered %d finding(s) of the combined ruleset, want 0: %s",
							fx.Path, len(findings), describeFindings(findings))
					}
				})
			}
			for _, fx := range claim.Fixtures.Negative {
				scanned++
				t.Run("negative/"+fx.Path, func(t *testing.T) {
					findings := scanWithCombinedRuleset(t, sgconfig, filepath.Join(absPackDir, fx.Path))
					// RULE-ID EQUALITY ALONE IS NOT ENOUGH, and the `t` discard is the whole
					// point. `t.Fatalf` is a selector_expression whose operand is the
					// *testing.T receiver `t`, so it fires referenced-symbol-go from inside
					// ANY substantive Go test regardless of the property the fixture was
					// written to demonstrate. Without the discard, referenced-symbol-go's
					// negative fixture satisfies bare rule-id equality on that boilerplate hit
					// alone and this subtest is accidentally satisfied rather than
					// discriminating. Do not weaken it back to bare equality.
					//
					// A rule that binds no PKG — hollow-test-go does not — has nothing
					// discarded, so own-rule equality alone carries it.
					var surviving []astGrepFinding
					for _, f := range findings {
						if f.RuleID != rule.ID {
							continue
						}
						if f.pkgText() == "t" {
							continue
						}
						surviving = append(surviving, f)
					}
					if len(surviving) == 0 {
						t.Errorf("negative fixture %s yielded no own-rule (%q) finding surviving the `t`-receiver discard; all findings: %s",
							fx.Path, rule.ID, describeFindings(findings))
					}
				})
			}
		}
	}
	if scanned == 0 {
		t.Fatalf("manifest at %s declared no fixtures to scan — the data-driven walk covered nothing", absPackDir)
	}
}

// TestSubstantivenessFixtures_TestMainExemptionKeepsThePositiveClean (CLM-003) is the
// falsification test. The hollow-test POSITIVE fixture holds an EMPTY-bodied
// `TestMain`, which is clean ONLY because hollow-test-go exempts `^TestMain$` by name
// (ISSUE-035 CLM-001). Strip that exemption from a COPY of the pack and the fixture
// must immediately fire — which is what proves the empty-bodied TestMain is a genuine
// discriminating pin on the exemption and not inert filler. Without this test, "move
// TestMain into the clean fixture" would be indistinguishable from deleting the
// exemption's coverage. The in-repo pack is NEVER mutated.
func TestSubstantivenessFixtures_TestMainExemptionKeepsThePositiveClean(t *testing.T) {
	requireAstGrepE2E(t)

	absPackDir := substantivenessSourceDir(repoRoot(t))
	m, err := packval.ParseManifest(filepath.Join(absPackDir, "pack.yml"))
	if err != nil {
		t.Fatalf("parse manifest at %s: %v", absPackDir, err)
	}

	const hollowRuleID = "hollow-test-go"
	var hollowRule *packval.Rule
	for i := range m.Content.Ruleset.Rules {
		if m.Content.Ruleset.Rules[i].ID == hollowRuleID {
			hollowRule = &m.Content.Ruleset.Rules[i]
		}
	}
	if hollowRule == nil {
		t.Fatalf("manifest at %s declares no rule %q", absPackDir, hollowRuleID)
	}
	var positive string
	for _, claim := range hollowRule.Claims {
		for _, fx := range claim.Fixtures.Positive {
			positive = fx.Path
		}
	}
	if positive == "" {
		t.Fatalf("rule %q declares no positive fixture", hollowRuleID)
	}

	copyDir := t.TempDir()
	copyPackTree(t, absPackDir, copyDir)

	pristine := scanWithCombinedRuleset(t, filepath.Join(copyDir, hollowRule.RuleSourcePath()), filepath.Join(copyDir, positive))
	if len(pristine) != 0 {
		t.Fatalf("control arm: positive fixture %s is not clean against the pristine rules (%s) — the counterfactual below would be meaningless",
			positive, describeFindings(pristine))
	}

	stripTestMainExemption(t, filepath.Join(copyDir, "ast-grep", "rules", hollowRuleID+".yml"))

	stripped := scanWithCombinedRuleset(t, filepath.Join(copyDir, hollowRule.RuleSourcePath()), filepath.Join(copyDir, positive))
	found := false
	for _, f := range stripped {
		if f.RuleID == hollowRuleID && f.fnText() == "TestMain" {
			found = true
		}
	}
	if !found {
		t.Errorf("with the ^TestMain$ exemption stripped, positive fixture %s did not yield a %s finding naming TestMain — the fixture is not pinning the exemption; findings: %s",
			positive, hollowRuleID, describeFindings(stripped))
	}
}

// copyPackTree mirrors a pack directory into dst. It is deliberately local rather than
// reusing an e2e workspace builder: this test needs the pack's raw tree, unmodified,
// with nothing installed around it.
func copyPackTree(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.Walk(src, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, relErr := filepath.Rel(src, path)
		if relErr != nil {
			return relErr
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		return os.WriteFile(target, data, info.Mode().Perm())
	})
	if err != nil {
		t.Fatalf("copy pack tree %s -> %s: %v", src, dst, err)
	}
}

// stripTestMainExemption removes the `not: has: {field: name, regex: "^TestMain$"}`
// conjunct from a COPY of the hollow rule. It edits the parsed YAML rather than the
// text so it does not depend on the rule file's formatting.
func stripTestMainExemption(t *testing.T, rulePath string) {
	t.Helper()
	data, err := os.ReadFile(rulePath)
	if err != nil {
		t.Fatalf("read rule %s: %v", rulePath, err)
	}
	var doc map[string]any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		t.Fatalf("parse rule %s: %v", rulePath, err)
	}
	ruleBlock, ok := doc["rule"].(map[string]any)
	if !ok {
		t.Fatalf("rule %s has no `rule:` mapping", rulePath)
	}
	all, ok := ruleBlock["all"].([]any)
	if !ok {
		t.Fatalf("rule %s has no `rule.all:` sequence to strip the exemption from", rulePath)
	}
	kept := make([]any, 0, len(all))
	removed := 0
	for _, conjunct := range all {
		if yamlMentions(conjunct, "^TestMain$") {
			removed++
			continue
		}
		kept = append(kept, conjunct)
	}
	if removed != 1 {
		t.Fatalf("rule %s: expected exactly 1 conjunct carrying the ^TestMain$ exemption, removed %d", rulePath, removed)
	}
	ruleBlock["all"] = kept
	out, err := yaml.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal stripped rule: %v", err)
	}
	if err := os.WriteFile(rulePath, out, 0o644); err != nil {
		t.Fatalf("write stripped rule %s: %v", rulePath, err)
	}
}

func yamlMentions(node any, want string) bool {
	switch v := node.(type) {
	case string:
		return v == want
	case map[string]any:
		for _, child := range v {
			if yamlMentions(child, want) {
				return true
			}
		}
	case []any:
		for _, child := range v {
			if yamlMentions(child, want) {
				return true
			}
		}
	}
	return false
}
