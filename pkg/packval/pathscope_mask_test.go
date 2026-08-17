package packval

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// pathScopeMaskCheck is the Check string the fixture-mask advisory carries. It is
// DISTINCT from pathScopeDispatchCheck on purpose: the dispatch advisory says a pattern
// is dark, the mask advisory says the pack nonetheless LOOKS healthy, and conflating them
// loses the second diagnostic entirely.
const pathScopeMaskCheck = "path-scope-fixture-mask"

// TestPathScope_FixtureMaskAdvisoryFires (CLM-006) drives the masking shape: an inert
// live scope alongside slash-free "fixture hooks" that match the rule's own declared
// fixtures. That combination is why execution-based validation cannot see this defect —
// the fixture is the one file the rule can still match.
func TestPathScope_FixtureMaskAdvisoryFires(t *testing.T) {
	// PIN THE CORPUS SHAPE BEFORE ASSERTING ON THE WARNING. fixture-masked's power comes
	// entirely from carrying TWO slash-free hooks that match DISJOINT fixtures, mirroring
	// the real no-structural-name-split-on-spine. If someone collapses them back to one
	// shared hook, every assertion below stays green while the only corpus-wide signal
	// separating the per-fixture ANY reading from the ALL reading is destroyed — and an
	// ALL-patterns implementation would then pass this whole suite and still fail to flag
	// the real flagship rule. This is a structural guard on the TESTDATA, not on the
	// implementation.
	assertDisjointHookShape(t, "fixture-masked", "sg-masked",
		[]string{"masked-alpha-positive.go", "masked-beta-negative.go"})

	hits, _ := pathScopeWarnings(t, "fixture-masked", pathScopeMaskCheck)
	if len(hits) != 1 {
		t.Fatalf("fixture-masked: got %d %s warnings, want exactly 1: %+v", len(hits), pathScopeMaskCheck, hits)
	}
	msg := hits[0].Message
	if !strings.Contains(msg, "sg-masked") {
		t.Errorf("mask warning does not name the semgrep rule it applies to: %q", msg)
	}
	// The whole diagnostic value of this check is the CONSEQUENCE, not the shape: a green
	// fixture run is not evidence the rule scans anything in a consuming repo.
	if !strings.Contains(msg, "pack test") {
		t.Errorf("mask warning does not explain that a passing fixture run is not evidence the rule scans anything: %q", msg)
	}
	if !containsString(hits[0].Files, "rules/masked.yml") {
		t.Errorf("mask warning does not carry the rule source in Files: %v", hits[0].Files)
	}

	// The mask is a REFINEMENT of the dispatch finding, never a replacement: the rule is
	// dark AND it looks healthy, and an author who saw only the mask would not know which
	// pattern to fix.
	dispatch, _ := pathScopeWarnings(t, "fixture-masked", pathScopeDispatchCheck)
	if len(dispatch) == 0 {
		t.Errorf("a mask warning fired with no accompanying %s warning, so the author is never told which pattern is inert", pathScopeDispatchCheck)
	}
}

// TestPathScope_FixtureMaskSilentWithoutTheHook (SE-7) is the falsifier that makes CLM-006
// non-vacuous. multiseg-no-mask is the SAME shape with the slash-free hooks removed: it
// carries an inert pattern and fixtures matched by nothing. An implementation that raises
// the mask whenever an inert pattern exists passes the test above and fails here.
func TestPathScope_FixtureMaskSilentWithoutTheHook(t *testing.T) {
	dispatch, _ := pathScopeWarnings(t, "multiseg-no-mask", pathScopeDispatchCheck)
	if len(dispatch) == 0 {
		t.Fatalf("multiseg-no-mask declares an inert pattern, so the dispatch advisory must still fire; got none")
	}
	mask, _ := pathScopeWarnings(t, "multiseg-no-mask", pathScopeMaskCheck)
	if len(mask) != 0 {
		t.Fatalf("multiseg-no-mask has NO slash-free hook, so its fixtures are matched by nothing and the pack does not look healthy; want 0 %s warnings, got %d: %+v", pathScopeMaskCheck, len(mask), mask)
	}
}

// TestPathScope_FixtureMaskScopedPerRule (SE-11) pins the FIXTURE side to the manifest
// rule's OWN claims[].fixtures. phase2's referencedFixtures map is PACK-WIDE — it
// accumulates every fixture across every rule and every tool_config entry — so an
// implementation reusing it lets probe-b's copy-pasted hook match probe-a's fixtures and
// raises a FALSE mask on probe-b. Both directions are asserted because such an
// implementation passes the presence half.
func TestPathScope_FixtureMaskScopedPerRule(t *testing.T) {
	hits, _ := pathScopeWarnings(t, "two-rule-cross-contamination", pathScopeMaskCheck)
	var sawA, sawB bool
	for _, w := range hits {
		if strings.Contains(w.Message, "sg-probe-a") {
			sawA = true
		}
		if strings.Contains(w.Message, "sg-probe-b") {
			sawB = true
		}
	}
	if !sawA {
		t.Errorf("probe-a's own fixtures are covered by its own slash-free hook, so the mask must fire for sg-probe-a; got %+v", hits)
	}
	if sawB {
		t.Errorf("probe-b's hook was copy-pasted from probe-a and matches NEITHER of probe-b's own fixtures, so no mask may name sg-probe-b — this is the pack-wide referencedFixtures contamination; got %+v", hits)
	}
}

// TestPathScope_FixtureMaskAttributedPerSemgrepRule (CLM-010, SE-15) pins the PATTERN side
// to the semgrep rule's OWN slash-free includes rather than the file's union across
// siblings. sg-unhooked carries an inert pattern and no slash-free pattern of its own, so
// the only way it can be masked is by borrowing sg-hooked's hook — which is exactly the
// shape 2 of the 7 real no-baked.yml rules have.
func TestPathScope_FixtureMaskAttributedPerSemgrepRule(t *testing.T) {
	hits, _ := pathScopeWarnings(t, "multi-semgrep-rule-file", pathScopeMaskCheck)
	if len(hits) != 1 {
		t.Fatalf("multi-semgrep-rule-file: want exactly 1 %s warning (naming sg-hooked), got %d: %+v", pathScopeMaskCheck, len(hits), hits)
	}
	if !strings.Contains(hits[0].Message, "sg-hooked") {
		t.Errorf("the single mask warning does not name sg-hooked: %q", hits[0].Message)
	}
	if strings.Contains(hits[0].Message, "sg-unhooked") {
		t.Errorf("sg-unhooked owns no slash-free pattern, so masking it means the file's patterns were unioned across sibling semgrep rules: %q", hits[0].Message)
	}
}

// TestPathScope_FixtureMaskRequiresEveryFixtureCovered (CLM-010, SE-16) pins the
// per-FIXTURE-SET quantifier as EVERY. partial-fixture-coverage declares two fixtures of
// which exactly one is matched, so fixture execution is ALREADY failing on the uncovered
// one — the pack does not look healthy and the mask's premise is absent. The dispatch
// advisory still fires, so this under-reports a diagnostic rather than hiding a defect.
//
// PAIRING, AND NEITHER PACK SUBSTITUTES FOR THE OTHER. This pack settles the
// per-fixture-SET half only; both its patterns sets are single-hook, so ANY and ALL are
// indistinguishable here at the per-FIXTURE level. That complementary half is pinned by
// fixture-masked, whose one semgrep rule carries TWO disjointly-matching hooks
// (`*masked-alpha*.go` matching only masked-alpha-positive.go, `*masked-beta*.go` matching
// only masked-beta-negative.go). An implementation requiring ALL of a rule's patterns to
// match each fixture declines the mask there and fails
// TestPathScope_FixtureMaskAdvisoryFires. Do not "simplify" either pack away.
func TestPathScope_FixtureMaskRequiresEveryFixtureCovered(t *testing.T) {
	dispatch, _ := pathScopeWarnings(t, "partial-fixture-coverage", pathScopeDispatchCheck)
	if len(dispatch) == 0 {
		t.Fatalf("partial-fixture-coverage declares an inert pattern, so the dispatch advisory must fire; got none")
	}
	mask, _ := pathScopeWarnings(t, "partial-fixture-coverage", pathScopeMaskCheck)
	if len(mask) != 0 {
		t.Fatalf("only ONE of the manifest rule's two declared fixtures is matched by its slash-free hook, so coverage is partial and the mask's premise is absent; want 0 %s warnings, got %d: %+v", pathScopeMaskCheck, len(mask), mask)
	}
}

// TestPathScope_FixtureMaskNeverBlocks (CLM-005) applies the non-blocking contract to the
// second check across every pack that can raise it.
func TestPathScope_FixtureMaskNeverBlocks(t *testing.T) {
	for _, name := range []string{
		"fixture-masked",
		"multiseg-no-mask",
		"two-rule-cross-contamination",
		"multi-semgrep-rule-file",
		"partial-fixture-coverage",
	} {
		t.Run(name, func(t *testing.T) {
			pack, dir := parsePathScopeFixture(t, name)
			res := RunCoherence(pack, dir)
			for _, e := range res.Errors {
				if strings.HasPrefix(e.Check, "path-scope-") {
					t.Fatalf("%s: the path-scope advisory populated Errors, which would fail pack check/test/add/install: %+v", name, e)
				}
			}
			if len(res.Errors) != 0 {
				t.Fatalf("%s: phase 2 reported %d errors, want 0: %+v", name, len(res.Errors), res.Errors)
			}
			if res.Status != "pass" {
				t.Fatalf("%s: PhaseResult.Status = %q, want \"pass\"", name, res.Status)
			}
			full := &Result{Phases: []PhaseResult{*res}, Errors: res.Errors, Warnings: res.Warnings}
			full.FinalizeStatus()
			if full.Status != "pass" {
				t.Fatalf("%s: finalized Result.Status = %q, want \"pass\"", name, full.Status)
			}
		})
	}
}

// TestPathScope_ChecksIndependentlyAttributable pins that the two advisories are separate
// signals: a pack can raise the dispatch advisory ALONE (multiseg-no-mask does), and every
// mask warning is accompanied by at least one dispatch warning on the same pack.
func TestPathScope_ChecksIndependentlyAttributable(t *testing.T) {
	dispatchOnly, _ := pathScopeWarnings(t, "multiseg-no-mask", pathScopeDispatchCheck)
	maskOnSame, _ := pathScopeWarnings(t, "multiseg-no-mask", pathScopeMaskCheck)
	if len(dispatchOnly) == 0 || len(maskOnSame) != 0 {
		t.Fatalf("expected multiseg-no-mask to raise the dispatch advisory alone; dispatch=%d mask=%d", len(dispatchOnly), len(maskOnSame))
	}
	for _, name := range []string{"fixture-masked", "two-rule-cross-contamination", "multi-semgrep-rule-file"} {
		mask, _ := pathScopeWarnings(t, name, pathScopeMaskCheck)
		if len(mask) == 0 {
			continue
		}
		dispatch, _ := pathScopeWarnings(t, name, pathScopeDispatchCheck)
		if len(dispatch) == 0 {
			t.Errorf("%s: raised %d mask warnings with no %s warning, leaving the author without the offending pattern", name, len(mask), pathScopeDispatchCheck)
		}
	}
}

// assertDisjointHookShape reads a fixture pack's rule file and asserts the named semgrep
// rule declares at least two slash-free include patterns, no one of which matches ALL the
// listed fixture basenames. That disjointness is the only thing in the corpus that makes
// CLM-010's per-fixture ANY quantifier observable.
func assertDisjointHookShape(t *testing.T, packName, semgrepID string, fixtureBasenames []string) {
	t.Helper()

	pack, dir := parsePathScopeFixture(t, packName)
	var ruleSource string
	for _, rule := range pack.Content.Ruleset.Rules {
		if rule.RuleSourcePath() != "" {
			ruleSource = rule.RuleSourcePath()
			break
		}
	}
	if ruleSource == "" {
		t.Fatalf("%s declares no rule source", packName)
	}
	data, err := os.ReadFile(filepath.Join(dir, ruleSource))
	if err != nil {
		t.Fatalf("reading %s/%s: %v", dir, ruleSource, err)
	}
	var doc struct {
		Rules []struct {
			ID    string `yaml:"id"`
			Paths struct {
				Include []string `yaml:"include"`
			} `yaml:"paths"`
		} `yaml:"rules"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		t.Fatalf("parsing %s: %v", ruleSource, err)
	}

	var hooks []string
	found := false
	for _, r := range doc.Rules {
		if r.ID != semgrepID {
			continue
		}
		found = true
		for _, p := range r.Paths.Include {
			if !strings.Contains(p, "/") {
				hooks = append(hooks, p)
			}
		}
	}
	if !found {
		t.Fatalf("%s: rule file %s declares no semgrep rule %q", packName, ruleSource, semgrepID)
	}
	if len(hooks) < 2 {
		t.Fatalf("CORPUS SHAPE BROKEN: %s rule %q declares %d slash-free include pattern(s), want at least 2. "+
			"With at most one hook per semgrep rule anywhere in the corpus, the per-fixture ANY reading and the "+
			"per-fixture ALL reading return identical results on every pack, so an ALL-patterns implementation "+
			"would pass this entire suite and still fail to flag the real no-structural-name-split-on-spine. "+
			"Restore the two disjoint hooks; do not collapse them into one shared pattern.", packName, semgrepID, len(hooks))
	}
	for _, hook := range hooks {
		matchesAll := true
		for _, base := range fixtureBasenames {
			ok, matchErr := filepath.Match(hook, base)
			if matchErr != nil || !ok {
				matchesAll = false
				break
			}
		}
		if matchesAll {
			t.Fatalf("CORPUS SHAPE BROKEN: %s rule %q hook %q matches ALL declared fixtures %v. "+
				"The hooks must be DISJOINT — each matching exactly one fixture, mirroring the real "+
				"no-structural-name-split-on-spine — or an ALL-patterns implementation is indistinguishable "+
				"from the correct ANY-patterns one.", packName, semgrepID, hook, fixtureBasenames)
		}
	}
}
