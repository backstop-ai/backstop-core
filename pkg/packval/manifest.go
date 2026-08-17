package packval

import (
	"fmt"
	"os"
	"sort"

	"github.com/backstop-ai/backstop-core/pkg/baseengines"
	"github.com/backstop-ai/backstop-core/pkg/pack/engine"
	"gopkg.in/yaml.v3"
)

type PackManifest struct {
	Name       string            `json:"name" yaml:"name"`
	Version    string            `json:"version" yaml:"version"`
	Language   string            `json:"language" yaml:"language"`
	Archetype  string            `json:"archetype" yaml:"archetype"`
	Content    Content           `json:"content" yaml:"content"`
	ToolConfig []ToolConfigEntry `json:"tool_config,omitempty" yaml:"tool_config,omitempty"`
	// Engines is the pack's optional engine block: name -> binding, DATA the pack
	// declares to add or OVERRIDE an execution engine (ISSUE-019). resolveEngine
	// merges these OVER baseengines.Registry(), a pack-declared engine winning over a
	// same-named base binding. The binding carries the command as DATA, so the harness
	// bakes no tool name. The block decodes STRAIGHT into engine.EngineBinding: its
	// int-enum fields (ScopeKind/EngineCategory/GateType) resolve their STRING
	// spellings through the enum UnmarshalYAML added in the leaf engine package
	// (ISSUE-032 B0/CLM-012), so a real engine pack's `scope_kind: project-wide`,
	// `category: mechanism`, `gate_type: build` parse to the SAME resolved bindings the
	// consumer's EngineSpec/parseEngineSpec yields — packval no longer dies at phase1
	// with the int-enum unmarshal errors that kept go-toolchain from reaching phase2/5.
	Engines map[string]engine.EngineBinding `json:"engines,omitempty" yaml:"engines,omitempty"`
	// Recipes is the pack's OPTIONAL top-level `recipes:` index (ISSUE-085): a stable
	// recipe id mapped to the pack-relative directory holding that recipe's recipe.yml
	// and payload. It mirrors the runtime model's own top-level index and, like it, is a
	// DISTINCT top-level pack.yml key from Content.Scaffolds (a rule's paired test
	// scaffold) — declaring both is valid and the two NEVER populate each other.
	// Zero-value (nil) when absent.
	//
	// Parsing it here is what lets phase4 enforce the `recipes` archetype's teeth at the
	// RECIPE grain. It is deliberately NOT a Content field: `recipes:` is top level in
	// pack.yml, so this is the honest parse rather than a back door into the content
	// model. Consequence, stated rather than hidden: a recipes-ONLY pack still fails
	// phase1's content-is-required check, which SPEC-054's sharp edge "A RECIPES-ONLY
	// pack does not validate yet" names as a tracked three-site follow-up.
	Recipes map[string]string `json:"recipes,omitempty" yaml:"recipes,omitempty"`

	// declaredEngineKeys records, per engine name, the set of manifest keys the pack
	// author ACTUALLY WROTE in that engine's block. It is the tri-state a plain decode
	// destroys: an omitted `exempt_from_scope_filter` and an explicit
	// `exempt_from_scope_filter: false` resolve to the SAME Go bool, so only key
	// presence can tell an unmade decision from a recorded one.
	//
	// It is unexported — a validator concern, not part of the pack's JSON/YAML surface
	// — and it is populated ONLY by ParseManifest, which ALWAYS assigns a non-nil map,
	// including for a manifest with no `engines:` block. That keeps "never parsed"
	// (nil) distinguishable from "parsed, declares no keys"; a PackManifest built as a
	// Go literal therefore has nil here by design.
	declaredEngineKeys map[string]map[string]bool
}

// engineKeyProbe is a deliberately loose second view of the same manifest bytes: it
// reads WHICH keys each engine block contains and nothing about what they mean.
// yaml.Node defers interpretation entirely, so this is a presence probe, not a second
// parser competing with the Engines decode.
type engineKeyProbe struct {
	Engines map[string]map[string]yaml.Node `yaml:"engines"`
}

// exemptFromScopeFilterField is the manifest key whose PRESENCE records the decision.
// It matches the yaml tag on engine.EngineBinding.ExemptFromScopeFilter.
const exemptFromScopeFilterField = "exempt_from_scope_filter"

// ExemptDecisionPending returns the sorted names of engines the manifest declares
// `scope_kind: project-wide` whose block OMITS `exempt_from_scope_filter` — the
// engines whose scope-filter decision has never been recorded anywhere.
//
// It consults scope_kind and key presence ONLY. It reads no gate_type, no engine name
// and no tool name, and it infers no default: whether a project-wide engine's
// violations survive diff-scope filtering is the pack author's call to record, never
// this function's to guess (SPEC-041 REQ-004/CLM-017).
//
// A manifest that was never parsed (nil record) yields nil rather than every
// project-wide engine, so a Go-literal manifest cannot manufacture a false prompt.
// This is the SINGLE authority for "a decision is owed here": the coherence advisory
// and the audit census both call it rather than re-deriving the rule.
func ExemptDecisionPending(m *PackManifest) []string {
	if m == nil || m.declaredEngineKeys == nil {
		return nil
	}
	pending := make([]string, 0, len(m.Engines))
	for name, binding := range m.Engines {
		if binding.ScopeKind != engine.ScopeKindProjectWide {
			continue
		}
		if m.declaredEngineKeys[name][exemptFromScopeFilterField] {
			continue
		}
		pending = append(pending, name)
	}
	sort.Strings(pending)
	return pending
}

// resolveEngine resolves an engine name against the base engine registry merged with
// the pack's declared engines: block, a pack-declared engine WINNING over a same-named
// base binding (ISSUE-019). An unknown engine — present in neither — fails loud via
// engine.Registry.Lookup; the harness never silently skips or defaults.
func resolveEngine(pack *PackManifest, name string) (engine.EngineBinding, error) {
	reg := baseengines.Registry()
	if pack != nil {
		for n, binding := range pack.Engines {
			reg[n] = binding
		}
	}
	binding, err := reg.Lookup(name)
	if err != nil {
		return engine.EngineBinding{}, fmt.Errorf("resolving rule engine %q: %w", name, err)
	}
	return binding, nil
}

type Content struct {
	Ruleset   Ruleset    `json:"ruleset" yaml:"ruleset"`
	Scaffolds []Scaffold `json:"scaffolds,omitempty" yaml:"scaffolds,omitempty"`
	SDK       *SDK       `json:"sdk,omitempty" yaml:"sdk,omitempty"`
}

type Ruleset struct {
	Rules []Rule `json:"rules" yaml:"rules"`
}

type Rule struct {
	ID     string `json:"id" yaml:"id"`
	Engine string `json:"engine,omitempty" yaml:"engine,omitempty"`
	// RulePath is the CANONICAL key naming a rule's source file. It is what the
	// gate-runtime model (pkg/pack.Rule) reads and the only one real pack.yml files
	// write (ISSUE-092).
	RulePath string `json:"rule_path,omitempty" yaml:"rule_path,omitempty"`
	// File is a BACK-COMPAT ALIAS for RulePath, kept because older testdata packs
	// declare it. Never read it directly — call RuleSourcePath(), which is the single
	// place the precedence between the two is decided.
	File          string    `json:"file,omitempty" yaml:"file,omitempty"`
	Tool          string    `json:"tool,omitempty" yaml:"tool,omitempty"`
	RiskClass     string    `json:"risk_class,omitempty" yaml:"risk_class,omitempty"`
	Layer         int       `json:"layer,omitempty" yaml:"layer,omitempty"`
	Category      string    `json:"category,omitempty" yaml:"category,omitempty"`
	Justification string    `json:"justification,omitempty" yaml:"justification,omitempty"`
	InputScope    string    `json:"input_scope,omitempty" yaml:"input_scope,omitempty"`
	Validator     string    `json:"validator,omitempty" yaml:"validator,omitempty"`
	PairsWith     PairsWith `json:"pairs_with,omitempty" yaml:"pairs_with,omitempty"`
	Claims        []Claim   `json:"claims,omitempty" yaml:"claims,omitempty"`
}

// RuleSourcePath resolves the rule's source file from the canonical `rule_path` key,
// falling back to the `file` alias. This is the ONLY place the precedence is decided;
// every consumer calls it and none re-implements it, so a future caller cannot
// re-derive the choice differently (CLM-001).
func (r Rule) RuleSourcePath() string {
	if r.RulePath != "" {
		return r.RulePath
	}
	return r.File
}

// RuleSourceManifestKey names the yaml key RuleSourcePath actually read, so a
// structural error can point at the key the pack author wrote rather than a guess.
func (r Rule) RuleSourceManifestKey() string {
	if r.RulePath != "" {
		return "rule_path"
	}
	return "file"
}

type ToolConfigEntry struct {
	ID         string  `json:"id,omitempty" yaml:"id,omitempty"`
	RequiredBy string  `json:"required_by,omitempty" yaml:"required_by,omitempty"`
	Engine     string  `json:"engine,omitempty" yaml:"engine,omitempty"`
	Tool       string  `json:"tool" yaml:"tool"`
	File       string  `json:"file" yaml:"file"`
	RiskClass  string  `json:"risk_class,omitempty" yaml:"risk_class,omitempty"`
	Claims     []Claim `json:"claims,omitempty" yaml:"claims,omitempty"`
}

type Claim struct {
	ID       string   `json:"id" yaml:"id"`
	Fixtures Fixtures `json:"fixtures" yaml:"fixtures"`
}

type Fixtures struct {
	Positive []FixtureRef `json:"positive" yaml:"positive"`
	Negative []FixtureRef `json:"negative" yaml:"negative"`
}

type FixtureRef struct {
	Path          string `json:"path" yaml:"path"`
	BypassAttempt bool   `json:"bypass_attempt,omitempty" yaml:"bypass_attempt,omitempty"`
}

func (f *FixtureRef) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var asPath string
	if err := unmarshal(&asPath); err == nil {
		f.Path = asPath
		return nil
	}
	var obj struct {
		Path          string `yaml:"path"`
		BypassAttempt bool   `yaml:"bypass_attempt"`
	}
	if err := unmarshal(&obj); err != nil {
		return fmt.Errorf("unmarshal fixture ref: %w", err)
	}
	f.Path = obj.Path
	f.BypassAttempt = obj.BypassAttempt
	return nil
}

type PairsWith struct {
	Rules     []string `json:"rules,omitempty" yaml:"rules,omitempty"`
	Scaffolds []string `json:"scaffolds,omitempty" yaml:"scaffolds,omitempty"`
	SDK       string   `json:"sdk,omitempty" yaml:"sdk,omitempty"`
}

type Scaffold struct {
	ID   string `json:"id" yaml:"id"`
	Path string `json:"path" yaml:"path"`
	Tier string `json:"tier,omitempty" yaml:"tier,omitempty"`
	// TestIndicator is the pack-DECLARED substring a skeleton scaffold's files must
	// contain to count as carrying test structure (e.g. "func Test", "describe(",
	// "@Test") — ISSUE-019. The skeleton check reads this from the manifest instead of
	// hardwiring a Go "_test.go" / "func Test" scan, so the harness bakes no language
	// convention. Empty => only the language-neutral "has structure" check applies.
	TestIndicator string            `json:"test_indicator,omitempty" yaml:"test_indicator,omitempty"`
	TestCommand   string            `json:"test_command,omitempty" yaml:"test_command,omitempty"`
	SampleConfig  map[string]string `json:"sample_config,omitempty" yaml:"sample_config,omitempty"`
	PairsWith     PairsWith         `json:"pairs_with,omitempty" yaml:"pairs_with,omitempty"`
}

type SDK struct {
	Provides []string `json:"provides,omitempty" yaml:"provides,omitempty"`
}

func ParseManifest(path string) (*PackManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading manifest %s: %w", path, err)
	}
	var out PackManifest
	if err := yaml.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}
	var probe engineKeyProbe
	if err := yaml.Unmarshal(data, &probe); err != nil {
		return nil, fmt.Errorf("probe manifest engine keys: %w", err)
	}
	out.declaredEngineKeys = make(map[string]map[string]bool, len(probe.Engines))
	for name, block := range probe.Engines {
		keys := make(map[string]bool, len(block))
		for key := range block {
			keys[key] = true
		}
		out.declaredEngineKeys[name] = keys
	}
	return &out, nil
}

func AllRules(m *PackManifest) []Rule {
	if m == nil {
		return nil
	}
	out := make([]Rule, 0, len(m.Content.Ruleset.Rules)+len(m.ToolConfig))
	out = append(out, m.Content.Ruleset.Rules...)
	for _, tc := range m.ToolConfig {
		if tc.ID == "" {
			continue
		}
		out = append(out, Rule{
			ID:        tc.ID,
			File:      tc.File,
			Tool:      tc.Tool,
			RiskClass: tc.RiskClass,
			Claims:    tc.Claims,
		})
	}
	return out
}

func AllRuleIDs(m *PackManifest) []string {
	rules := AllRules(m)
	out := make([]string, 0, len(rules))
	for _, r := range rules {
		out = append(out, r.ID)
	}
	return out
}
