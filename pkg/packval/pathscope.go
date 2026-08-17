package packval

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/backstop-ai/backstop-core/pkg/pack/engine"
	"gopkg.in/yaml.v3"
)

// PathPatternInertUnderFileDispatch answers the one question this whole check is built
// on: is a semgrep `paths.include` / `paths.exclude` pattern INERT when the gate hands
// the engine an explicit list of files rather than a directory to walk?
//
// It decides on the presence of a `/` in the pattern and on NOTHING else. That is not a
// simplification of a richer rule — it is the measured contract. Against real semgrep
// 1.156.0, with the same two explicit file targets, four spellings of one intent were
// compared:
//
//	include: "cmd/backstop/pack_gate*.go"      -> 0 findings
//	include: "**/cmd/backstop/pack_gate*.go"   -> 0 findings
//	include: "/cmd/backstop/pack_gate*.go"     -> 0 findings
//	include: "pack_gate*.go"                   -> 2 findings
//
// All four fire under a DIRECTORY target; only the slash-free one survives explicit-file
// dispatch. `backstop/pack_gate*.go` (tail-only) and `*/*/pack_gate*.go` behave the same
// as the rest — dark.
//
// ⚠ DO NOT "FIX" THIS TO BLESS THE `**/` OR `/` SPELLINGS. semgrep itself prints a
// deprecation warning on every one of these patterns recommending exactly those two
// rewrites. Both were measured and both remain dark, so an implementer who trusts the
// tool's own remedy ships a change that does nothing. The measurement above is the
// authority here, not the warning text.
//
// This is the SINGLE statement of the contract in the tree. cmd/backstop's CI-recipe
// harness consumes it rather than re-deriving it, so the two cannot drift.
func PathPatternInertUnderFileDispatch(pattern string) bool {
	return strings.Contains(pattern, "/")
}

const (
	pathScopeDispatchCheckName = "path-scope-dispatch"
	pathScopeMaskCheckName     = "path-scope-fixture-mask"
)

// pathScopeFixHint names the ONLY remedy that was measured to work, and says plainly
// that the tool's own suggestion does not — because that suggestion is what the pack
// author is being shown by semgrep at the same moment they read this.
const pathScopeFixHint = "restate the scope with a SLASH-FREE (single-segment) pattern such as \"handler*.go\" — that is the only spelling honored under BOTH directory and explicit-file dispatch. Do NOT apply semgrep's own deprecation remedy: `**/`-prefixing and `/`-anchoring were both measured dark under explicit-file dispatch and change nothing. Where no slash-free spelling preserves the intended directory scope, the real choice is a wider blast radius or no scoping at all — that is a pack-authoring judgement call, which is why this is a warning and not an error."

// pathScopeFinding is one inert path pattern, attributed to the SEMGREP rule that
// declared it rather than to the pack manifest rule that named the file (SE-4): a single
// rule file routinely holds many semgrep rules, and pointing at the manifest id would
// point the author at the wrong one.
type pathScopeFinding struct {
	RuleSource string // pack-relative path of the rule file
	SemgrepID  string
	Key        string // "include" or "exclude"
	Pattern    string
}

// pathScopeMaskFinding is one rule whose live scope is dark while its declared fixtures
// are still matched — the shape that keeps fixture EXECUTION green and is therefore the
// reason a broken pack looks healthy.
type pathScopeMaskFinding struct {
	RuleSource     string
	SemgrepID      string
	ManifestRuleID string
}

// semgrepPathRule is the narrow view of a semgrep rule file this check needs: the rule id
// and its path scoping. Deliberately a loose anonymous-shaped struct in the same style as
// phase3's semgrepFileContainsRuleID — a full semgrep schema here would make an
// unfamiliar-but-valid rule file unreadable, and this check is advisory.
type semgrepPathRule struct {
	ID      string   `yaml:"id"`
	Include []string `yaml:"-"`
	Exclude []string `yaml:"-"`
}

func parseSemgrepPathRules(data []byte) []semgrepPathRule {
	var doc struct {
		Rules []struct {
			ID    string `yaml:"id"`
			Paths struct {
				Include []string `yaml:"include"`
				Exclude []string `yaml:"exclude"`
			} `yaml:"paths"`
		} `yaml:"rules"`
	}
	// A rule source that does not parse, or that declares no paths: block anywhere, is
	// SILENT (SE-6). Absence of path scoping is not a defect and file existence is
	// phase 1's job; adding an error class for either here would be a new failure mode
	// rather than a diagnostic.
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil
	}
	out := make([]semgrepPathRule, 0, len(doc.Rules))
	for _, r := range doc.Rules {
		out = append(out, semgrepPathRule{ID: r.ID, Include: r.Paths.Include, Exclude: r.Paths.Exclude})
	}
	return out
}

// ruleFlagsSourcePaths collapses a pack's manifest rules to their DISTINCT rule-source
// paths, keeping declaration order, and keeps a path only when at least one manifest rule
// naming it resolves to a rule-flags binding.
//
// COLLAPSING FIRST IS THE POINT, NOT AN OPTIMIZATION (CLM-009). N manifest rules sharing
// one rule file is the common shape, not an oddity — backstop-self declares SEVEN
// manifest rules all naming rules/no-baked.yml, which holds 26 inert patterns. A walk
// that iterates patterns inside the manifest-rule loop emits 7x26=182 warnings where the
// correct answer is 26, and caching the PARSE does not help: a cached parse walked once
// per referencing manifest rule still emits N copies.
//
// The InputMode gate (CLM-007) is evaluated per manifest rule, exactly as phase3's
// semgrep-rule-id check does it, and never against an engine-name literal.
func ruleFlagsSourcePaths(pack *PackManifest) []string {
	var ordered []string
	seen := map[string]bool{}
	for _, rule := range pack.Content.Ruleset.Rules {
		src := rule.RuleSourcePath()
		if src == "" || seen[src] {
			continue
		}
		if !ruleFlagsBinding(pack, rule) {
			continue
		}
		seen[src] = true
		ordered = append(ordered, src)
	}
	return ordered
}

// ruleFlagsBinding reports whether a manifest rule's declared engine RESOLVES and
// declares input_mode rule-flags — the mode under which the declared file is a LIST OF
// RULES each carrying an id and a paths block. Under config-file the declared file is a
// project config naming rule directories, so a rules[].paths scan of it is categorically
// inapplicable rather than merely empty.
func ruleFlagsBinding(pack *PackManifest, rule Rule) bool {
	if rule.Engine == "" {
		return false
	}
	binding, err := resolveEngine(pack, rule.Engine)
	if err != nil {
		return false
	}
	return binding.InputMode == engine.InputModeRuleFlags
}

func readRuleSource(packDir, ruleSource string) []byte {
	data, err := os.ReadFile(filepath.Join(packDir, ruleSource))
	if err != nil {
		return nil
	}
	return data
}

// pathScopeDispatchFindings yields one finding per inert pattern, deduplicated on
// (rule-source path, semgrep rule id, key, pattern). The manifest rule id is
// DELIBERATELY ABSENT from that key: including it would restore the N-copy duplication
// under a different spelling.
func pathScopeDispatchFindings(pack *PackManifest, packDir string) []pathScopeFinding {
	var out []pathScopeFinding
	emitted := map[pathScopeFinding]bool{}
	for _, src := range ruleFlagsSourcePaths(pack) {
		data := readRuleSource(packDir, src)
		if len(data) == 0 {
			continue
		}
		for _, sr := range parseSemgrepPathRules(data) {
			for _, keyed := range []struct {
				key      string
				patterns []string
			}{
				{"include", sr.Include},
				{"exclude", sr.Exclude},
			} {
				for _, pattern := range keyed.patterns {
					if !PathPatternInertUnderFileDispatch(pattern) {
						continue
					}
					f := pathScopeFinding{RuleSource: src, SemgrepID: sr.ID, Key: keyed.key, Pattern: pattern}
					if emitted[f] {
						continue
					}
					emitted[f] = true
					out = append(out, f)
				}
			}
		}
	}
	return out
}

// pathScopeDispatchMessage names the failure DIRECTION, because the two directions do
// different harm and reporting one generic string leaves half of it invisible. An inert
// include is a silent no-op; an inert exclude fails OPEN.
func pathScopeDispatchMessage(f pathScopeFinding) string {
	consequence := "this rule scans NOTHING when the gate hands its engine an explicit file list — a silent no-op whose green is vacuous"
	if f.Key == "exclude" {
		consequence = "this exclusion FAILS OPEN — the rule fires on files the pack explicitly meant to exempt, which reads as a false RED the author believes they already suppressed"
	}
	return fmt.Sprintf(
		"semgrep rule %q declares a paths.%s pattern containing a %q: %q. A slash-bearing path pattern is unsatisfied under the gate's explicit-file dispatch in EVERY spelling, so %s.",
		f.SemgrepID, f.Key, "/", f.Pattern, consequence,
	)
}

// pathScopeMaskFindings detects the fixture-hook masking shape, evaluated per
// (MANIFEST rule x SEMGREP rule):
//
//   - the FIXTURE side comes from the MANIFEST rule's OWN claims[].fixtures (SE-11).
//     phase2's referencedFixtures map is PACK-WIDE — accumulated across every rule and
//     every tool_config entry — so feeding it here would let one rule's fixtures satisfy
//     another rule's slash-free patterns, which is exactly the copy-pasted-hook shape
//     real multi-rule packs have.
//   - the PATTERN side comes from the SEMGREP rule's OWN slash-free includes, never the
//     file's union across siblings (SE-15). no-baked.yml holds 7 semgrep rules of which
//     2 declare no include at all while carrying an inert exclude; unioning masks those
//     2 off a sibling's hook.
//
// QUANTIFIERS, and they are settled by measurement rather than taste (SE-16). Per
// FIXTURE the quantifier is ANY: the real no-structural-name-split-on-spine declares two
// fixtures and two hooks, each hook matching exactly one fixture, so an ALL-patterns
// reading would see the primary real-world case as unmasked. Per FIXTURE SET the
// quantifier is EVERY: partial coverage means fixture execution is ALREADY failing on the
// uncovered fixture, so the pack does not look healthy and the mask's premise is absent.
// The dispatch advisory still fires there, so nothing goes unreported.
func pathScopeMaskFindings(pack *PackManifest, packDir string) []pathScopeMaskFinding {
	var out []pathScopeMaskFinding
	emitted := map[pathScopeMaskFinding]bool{}
	for _, rule := range pack.Content.Ruleset.Rules {
		src := rule.RuleSourcePath()
		if src == "" || !ruleFlagsBinding(pack, rule) {
			continue
		}
		fixtures := declaredFixtureBasenames(rule)
		// A rule with no declared fixtures can never be masked — it has nothing to be
		// masked BY.
		if len(fixtures) == 0 {
			continue
		}
		data := readRuleSource(packDir, src)
		if len(data) == 0 {
			continue
		}
		for _, sr := range parseSemgrepPathRules(data) {
			if !semgrepRuleHasInertPattern(sr) {
				continue
			}
			hooks := slashFreePatterns(sr.Include)
			if len(hooks) == 0 {
				continue
			}
			if !everyFixtureMatchedByAnyHook(fixtures, hooks) {
				continue
			}
			m := pathScopeMaskFinding{RuleSource: src, SemgrepID: sr.ID, ManifestRuleID: rule.ID}
			if emitted[m] {
				continue
			}
			emitted[m] = true
			out = append(out, m)
		}
	}
	return out
}

func declaredFixtureBasenames(rule Rule) []string {
	var out []string
	for _, claim := range rule.Claims {
		for _, f := range append(append([]FixtureRef{}, claim.Fixtures.Positive...), claim.Fixtures.Negative...) {
			if f.Path == "" {
				continue
			}
			out = append(out, filepath.Base(filepath.FromSlash(f.Path)))
		}
	}
	return out
}

func semgrepRuleHasInertPattern(sr semgrepPathRule) bool {
	for _, p := range append(append([]string{}, sr.Include...), sr.Exclude...) {
		if PathPatternInertUnderFileDispatch(p) {
			return true
		}
	}
	return false
}

func slashFreePatterns(patterns []string) []string {
	var out []string
	for _, p := range patterns {
		if !PathPatternInertUnderFileDispatch(p) {
			out = append(out, p)
		}
	}
	return out
}

// everyFixtureMatchedByAnyHook matches each pattern against the fixture's BASENAME, which
// is what a slash-free pattern is: the single-segment case.
func everyFixtureMatchedByAnyHook(fixtureBasenames, hooks []string) bool {
	for _, base := range fixtureBasenames {
		matched := false
		for _, hook := range hooks {
			if ok, err := filepath.Match(hook, base); err == nil && ok {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	return true
}

func pathScopeMaskMessage(m pathScopeMaskFinding) string {
	return fmt.Sprintf(
		"semgrep rule %q (declared by manifest rule %q) carries at least one inert slash-bearing path pattern, and EVERY fixture that manifest rule declares is matched only by the rule's slash-free patterns. Those slash-free patterns are fixture hooks: the fixture is the one file the rule can still match, so fixture execution stays GREEN while the rule's intended live scope is dark. A passing `pack test` is therefore NOT evidence that this rule scans anything in a consuming repo.",
		m.SemgrepID, m.ManifestRuleID,
	)
}
