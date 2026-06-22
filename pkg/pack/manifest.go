package pack

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/bmanson/backstop-core/pkg/pack/engine"
	"gopkg.in/yaml.v3"
)

// Manifest is the top-level pack manifest.
type Manifest struct {
	Name           string                `yaml:"name"`
	NormalizedName string                `yaml:"-"`
	Version        string                `yaml:"version"`
	Language       string                `yaml:"language"`
	Archetype      string                `yaml:"archetype"`
	Description    string                `yaml:"description"`
	Content        Content               `yaml:"content"`
	ToolConfig     []ToolConfigEntry     `yaml:"tool_config"`
	// Engines holds the pack-declared engine bindings parsed from the top-level
	// `engines:` block (SPEC-035 REQ-001/CLM-001). Each EngineSpec carries the
	// yaml-tagged binding fields and is converted to an engine.EngineBinding at
	// load (EngineSpec.Binding). resolveEngineRegistry merges these over the
	// fallback registry so a rule's declared engine resolves to the pack-declared
	// binding when present.
	Engines map[string]EngineSpec `yaml:"engines"`
}

// EngineSpec is one entry in a pack manifest's top-level `engines:` block: the
// yaml-tagged surface a pack declares an execution engine with (SPEC-035
// REQ-001/CLM-001). The string-valued scope_kind / category / input_mode /
// gate_type spellings are parsed into the engine package's enum types at load
// (parseEngineSpec); the resolved engine.EngineBinding is stored on Binding so
// every consumer reads ONE converted binding, never the raw spec.
type EngineSpec struct {
	Command       string                `yaml:"command"`
	InputMode     string                `yaml:"input_mode"`
	InputFlag     string                `yaml:"input_flag"`
	Convert       string                `yaml:"convert"`
	ScopeKind     string                `yaml:"scope_kind"`
	Category      string                `yaml:"category"`
	GateType      string                `yaml:"gate_type"`
	StrictSarif   bool                  `yaml:"strict_sarif"`
	PackageScoped bool                  `yaml:"package_scoped"`
	ProjectTarget string                `yaml:"project_target"`
	Provision     *engine.Provision     `yaml:"provision"`
	FieldContract *engine.FieldContract `yaml:"field_contract"`
	// Binding is the engine.EngineBinding the spec converts to at load. It is
	// populated by parseEngineSpec during ParseManifest, not parsed directly from
	// yaml, so the string-enum spellings resolve through the fail-loud parsers.
	Binding engine.EngineBinding `yaml:"-"`
}

// Content contains rulesets, scaffolds, and SDK metadata.
type Content struct {
	Ruleset   Ruleset    `yaml:"ruleset"`
	Scaffolds []Scaffold `yaml:"scaffolds"`
	SDK       *SDK       `yaml:"sdk"`
}

// Ruleset contains rules and optional version.
type Ruleset struct {
	Version string `yaml:"version"`
	Rules   []Rule `yaml:"rules"`
}

// Rule defines a single rule entry. The execution engine is declared via the
// first-class Engine field (SPEC-031 REQ-001); the retired Layer field and its
// yaml key are gone (REQ-002). A rule whose engine is empty is a blocking
// ConfigError at the migrated reader — there is no layer:2 -> engine:semgrep
// aliasing (REQ-002/REQ-015).
type Rule struct {
	ID            string    `yaml:"id"`
	NamespacedID  string    `yaml:"-"`
	Engine        string    `yaml:"engine"`
	Standard      string    `yaml:"standard"`
	RulePath      string    `yaml:"rule_path"`
	// Pattern is the inline rule pattern a pattern-arg engine passes as a command
	// argument instead of resolving a rule file on disk (SPEC-035 REQ-004). Empty
	// for non-pattern-arg engines; an empty Pattern under a pattern-arg engine is
	// a blocking broken-pack config error at gather time (CLM-017).
	Pattern       string    `yaml:"pattern"`
	RiskClass     string    `yaml:"risk_class"`
	Claims        []Claim   `yaml:"claims"`
	Category      string    `yaml:"category"`
	Justification string    `yaml:"justification"`
	Validator     string    `yaml:"validator"`
	InputScope    string    `yaml:"input_scope"`
	PairsWith     PairsWith `yaml:"pairs_with"`
}

// Claim defines a rule claim and fixture mappings.
type Claim struct {
	ID       string   `yaml:"id"`
	Text     string   `yaml:"text"`
	Fixtures Fixtures `yaml:"fixtures"`
}

// Fixtures maps claim fixtures.
type Fixtures struct {
	Positive []FixtureEntry `yaml:"positive"`
	Negative []FixtureEntry `yaml:"negative"`
}

// FixtureEntry is either a plain string path or object with metadata.
type FixtureEntry struct {
	Path          string `yaml:"path"`
	BypassAttempt bool   `yaml:"bypass_attempt"`
}

// UnmarshalYAML handles fixture polymorphism.
func (f *FixtureEntry) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var asString string
	if err := unmarshal(&asString); err == nil {
		f.Path = asString
		f.BypassAttempt = false
		return nil
	}

	var asObject struct {
		Path          string `yaml:"path"`
		BypassAttempt bool   `yaml:"bypass_attempt"`
	}
	if err := unmarshal(&asObject); err != nil {
		return fmt.Errorf("fixture entry must be string or object: %w", err)
	}
	if asObject.Path == "" {
		return fmt.Errorf("fixture entry path is required")
	}
	f.Path = asObject.Path
	f.BypassAttempt = asObject.BypassAttempt
	return nil
}

// Scaffold describes a scaffolded asset.
type Scaffold struct {
	ID           string            `yaml:"id"`
	Version      string            `yaml:"version"`
	Tier         string            `yaml:"tier"`
	Path         string            `yaml:"path"`
	TestCommand  string            `yaml:"test_command"`
	Description  string            `yaml:"description"`
	UseWhen      []string          `yaml:"use_when"`
	Assumes      []string          `yaml:"assumes"`
	PairsWith    PairsWith         `yaml:"pairs_with"`
	SampleConfig map[string]any    `yaml:"sample_config"`
}

// SDK describes an SDK dependency.
type SDK struct {
	Module   string   `yaml:"module"`
	Version  string   `yaml:"version"`
	Provides []string `yaml:"provides"`
}

// ToolConfigEntry describes a tool config item.
type ToolConfigEntry struct {
	ID         string         `yaml:"id"`
	Tool       string         `yaml:"tool"`
	File       string         `yaml:"file"`
	Settings   map[string]any `yaml:"settings"`
	RiskClass  string         `yaml:"risk_class"`
	Claims     []Claim        `yaml:"claims"`
	RequiredBy string         `yaml:"required_by"`
}

// PairsWith groups relationships to other elements.
type PairsWith struct {
	Rules     []string `yaml:"rules"`
	Scaffolds []string `yaml:"scaffolds"`
	SDK       string   `yaml:"sdk"`
}

// Coordinate identifies a versioned pack item reference.
type Coordinate struct {
	PackName    string
	PackVersion string
	ItemName    string
	ItemVersion string
}

// ParseManifest parses and validates pack.yml bytes.
func ParseManifest(data []byte) (*Manifest, error) {
	var manifest Manifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("parse manifest yaml: %w", err)
	}

	if manifest.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if manifest.Version == "" {
		return nil, fmt.Errorf("version is required")
	}
	if manifest.Language == "" {
		return nil, fmt.Errorf("language is required")
	}
	if manifest.Archetype == "" {
		return nil, fmt.Errorf("archetype is required")
	}
	if manifest.Description == "" {
		return nil, fmt.Errorf("description is required")
	}

	if len(manifest.Content.Ruleset.Rules) == 0 &&
		len(manifest.Content.Scaffolds) == 0 &&
		manifest.Content.SDK == nil {
		return nil, fmt.Errorf("content is required")
	}

	if err := validateName(manifest.Name); err != nil {
		return nil, err
	}
	if err := validateLanguageField(manifest.Language); err != nil {
		return nil, err
	}
	if err := validateArchetype(manifest.Archetype); err != nil {
		return nil, err
	}
	if err := validateSemver(manifest.Version); err != nil {
		return nil, fmt.Errorf("invalid version: %w", err)
	}

	// Convert each pack-declared engines: spec into a resolved engine.EngineBinding,
	// fail-louding on any unknown string-enum spelling (input_mode / scope_kind /
	// category / gate_type). This is the REQ-001/CLM-001 parse and the CLM-021
	// fail-loud on an unrecognized gate_type.
	for name := range manifest.Engines {
		spec := manifest.Engines[name]
		binding, err := parseEngineSpec(spec)
		if err != nil {
			return nil, fmt.Errorf("engine %q: %w", name, err)
		}
		spec.Binding = binding
		manifest.Engines[name] = spec
	}

	if manifest.Content.Ruleset.Version == "" && manifest.Archetype == "enforcement" {
		manifest.Content.Ruleset.Version = manifest.Version
	}
	if manifest.Content.Ruleset.Version != "" {
		if err := validateSemver(manifest.Content.Ruleset.Version); err != nil {
			return nil, fmt.Errorf("invalid ruleset version: %w", err)
		}
	}

	declaredBindings := declaredEngineBindings(&manifest)
	for i := range manifest.Content.Ruleset.Rules {
		rule := &manifest.Content.Ruleset.Rules[i]
		if err := ValidateRuleID(rule.ID); err != nil {
			return nil, fmt.Errorf("invalid rule id %q: %w", rule.ID, err)
		}
		if err := validateRiskClass(rule.RiskClass); err != nil {
			return nil, err
		}
		if err := validateEngine(rule.Engine, declaredBindings); err != nil {
			return nil, fmt.Errorf("rule %q: %w", rule.ID, err)
		}
		for j := range rule.Claims {
			for k := range rule.Claims[j].Fixtures.Positive {
				rule.Claims[j].Fixtures.Positive[k].BypassAttempt = false
			}
			if err := validateFixtures(rule.Claims[j].Fixtures); err != nil {
				return nil, err
			}
		}
	}

	for i := range manifest.Content.Scaffolds {
		scaffold := manifest.Content.Scaffolds[i]
		if scaffold.Version != "" {
			if err := validateSemver(scaffold.Version); err != nil {
				return nil, fmt.Errorf("invalid scaffold version: %w", err)
			}
		}
		if err := validateScaffold(scaffold); err != nil {
			return nil, err
		}
	}
	if manifest.Content.SDK != nil {
		if err := validateSDK(manifest.Content.SDK); err != nil {
			return nil, err
		}
		if err := validateSemver(manifest.Content.SDK.Version); err != nil {
			return nil, fmt.Errorf("invalid sdk version: %w", err)
		}
	}
	for _, tc := range manifest.ToolConfig {
		if err := validateToolConfig(tc); err != nil {
			return nil, err
		}
		for i := range tc.Claims {
			for j := range tc.Claims[i].Fixtures.Positive {
				tc.Claims[i].Fixtures.Positive[j].BypassAttempt = false
			}
			if err := validateFixtures(tc.Claims[i].Fixtures); err != nil {
				return nil, err
			}
		}
	}

	manifest.NormalizedName = strings.ToLower(manifest.Name)
	seenClaimIDs := make(map[string]struct{})
	for i := range manifest.Content.Ruleset.Rules {
		rule := &manifest.Content.Ruleset.Rules[i]
		for _, claim := range rule.Claims {
			if _, exists := seenClaimIDs[claim.ID]; exists {
				return nil, fmt.Errorf("duplicate claim id: %s", claim.ID)
			}
			seenClaimIDs[claim.ID] = struct{}{}
		}
	}
	for _, tc := range manifest.ToolConfig {
		for _, claim := range tc.Claims {
			if _, exists := seenClaimIDs[claim.ID]; exists {
				return nil, fmt.Errorf("duplicate claim id: %s", claim.ID)
			}
			seenClaimIDs[claim.ID] = struct{}{}
		}
	}
	for i := range manifest.Content.Ruleset.Rules {
		manifest.Content.Ruleset.Rules[i].NamespacedID = NamespacedRuleID(manifest.NormalizedName, manifest.Content.Ruleset.Rules[i].ID)
	}

	return &manifest, nil
}

// ParseManifestFile reads and parses a manifest file.
func ParseManifestFile(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read manifest file: %w", err)
	}
	return ParseManifest(data)
}

var namePartPattern = regexp.MustCompile(`^[A-Za-z0-9-]+$`)
var semverPattern = regexp.MustCompile(`^\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$`)

func validateName(name string) error {
	parts := strings.Split(name, "/")
	if len(parts) != 2 {
		return fmt.Errorf("name must contain exactly one slash")
	}
	if parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("name parts must be non-empty")
	}
	for _, part := range parts {
		if !namePartPattern.MatchString(part) {
			return fmt.Errorf("name part %q contains invalid characters", part)
		}
	}
	return nil
}

func validateLanguageField(lang string) error {
	return ValidateLanguage(lang)
}

func validateArchetype(a string) error {
	switch a {
	case "enforcement", "code":
		return nil
	default:
		return fmt.Errorf("archetype must be enforcement or code")
	}
}

func validateRiskClass(rc string) error {
	switch rc {
	case "security", "correctness", "style", "perf":
		return nil
	default:
		return fmt.Errorf("risk_class must be one of security, correctness, style, perf")
	}
}

// declaredEngineBindings collects the pack-declared engines: block into a
// {name -> resolved EngineBinding} map for validateEngine's union lookup. The
// bindings are already converted (EngineSpec.Binding) by the engines-parse loop
// in ParseManifest, so this just projects them.
func declaredEngineBindings(m *Manifest) map[string]engine.EngineBinding {
	declared := make(map[string]engine.EngineBinding, len(m.Engines))
	for name, spec := range m.Engines {
		declared[name] = spec.Binding
	}
	return declared
}

// validateEngine fail-louds on a rule whose engine is empty (a layer-only rule
// under the migrated reader) or unknown to BOTH the pack-declared engines: block
// (declared) and the fallback registry (SPEC-035 REQ-003/CLM-010..012). When the
// engine IS known, its tool must additionally pass the trusted-tool allowlist:
// the validation-time half of the trust gate (Sharp Edge 1). A pack-declared
// binding overrides a same-named built-in (the deterministic merge, CLM-004), so
// the declared map is consulted FIRST. There is no layer:2 -> engine:semgrep
// aliasing: a layer-only rule is a blocking config error (REQ-002/REQ-015).
//
// The allowlist trust gate only applies to a binding with a non-nil Provision —
// the backstop-introduced tools that ride the backstop.lock pin (semgrep,
// ast-grep, or any pack-declared provisioned tool). A nil-Provision binding is an
// assume-present Layer-0 toolchain engine (go build/test, golangci) or the
// sandbox/config-file shape: it carries no provisioned tool, so it is governed by
// the assume-present-on-PATH provisioning model, not the allowlist+lock-pin
// (CLM-011 still accepts it as a known engine). The lockedVersion passed to
// CheckToolAllowed is the binding's Provision.Version (the lock-resolved pin),
// NOT a second literal (CLM-029).
func validateEngine(name string, declared map[string]engine.EngineBinding) error {
	if name == "" {
		return fmt.Errorf("engine is required (the retired layer field is no longer read; declare engine explicitly)")
	}

	binding, ok := declared[name]
	if !ok {
		var err error
		binding, err = engine.DefaultRegistry().Lookup(name)
		if err != nil {
			return err
		}
	}

	if binding.Provision != nil {
		if err := engine.CheckToolAllowed(
			engine.TrustedToolAllowlist(),
			binding.Provision.Tool,
			binding.Provision.Version,
		); err != nil {
			return err
		}
	}
	return nil
}

// parseEngineSpec converts a pack-declared EngineSpec (the yaml-tagged engines:
// block surface) into a resolved engine.EngineBinding, resolving every
// string-enum spelling through its fail-loud parser (SPEC-035 REQ-001/CLM-001).
// An unrecognized input_mode, scope_kind, category, or gate_type is a blocking
// config error — no silent default (CLM-021 for gate_type).
func parseEngineSpec(spec EngineSpec) (engine.EngineBinding, error) {
	binding := engine.EngineBinding{
		Command:       spec.Command,
		InputFlag:     spec.InputFlag,
		Convert:       spec.Convert,
		StrictSarif:   spec.StrictSarif,
		PackageScoped: spec.PackageScoped,
		ProjectTarget: spec.ProjectTarget,
		Provision:     spec.Provision,
	}

	inputMode, err := engine.ParseInputMode(spec.InputMode)
	if err != nil {
		return engine.EngineBinding{}, err
	}
	binding.InputMode = inputMode

	scopeKind, err := parseScopeKind(spec.ScopeKind)
	if err != nil {
		return engine.EngineBinding{}, err
	}
	binding.ScopeKind = scopeKind

	category, err := parseEngineCategory(spec.Category)
	if err != nil {
		return engine.EngineBinding{}, err
	}
	binding.Category = category

	gateType, err := engine.ParseGateType(spec.GateType)
	if err != nil {
		return engine.EngineBinding{}, err
	}
	binding.GateType = gateType

	if spec.FieldContract != nil {
		binding.FieldContract = *spec.FieldContract
	}

	return binding, nil
}

// parseScopeKind resolves the pack-declared scope_kind spelling (file-args |
// project-wide) into engine.ScopeKind, fail-louding on an unrecognized value. An
// empty value defaults to file-args (the scoped-file-args attachment), matching
// the binding's zero value.
func parseScopeKind(s string) (engine.ScopeKind, error) {
	switch s {
	case "", "file-args":
		return engine.ScopeKindFileArgs, nil
	case "project-wide":
		return engine.ScopeKindProjectWide, nil
	default:
		return 0, fmt.Errorf("unknown scope_kind %q: must be one of file-args, project-wide", s)
	}
}

// parseEngineCategory resolves the pack-declared category spelling (opinion |
// mechanism) into engine.EngineCategory, fail-louding on an unrecognized value.
// An empty value defaults to EngineCategoryUnset (neither mechanism nor opinion,
// invisible to the pack-separation boundary check).
func parseEngineCategory(s string) (engine.EngineCategory, error) {
	switch s {
	case "":
		return engine.EngineCategoryUnset, nil
	case "mechanism":
		return engine.EngineCategoryMechanism, nil
	case "opinion":
		return engine.EngineCategoryOpinion, nil
	default:
		return 0, fmt.Errorf("unknown category %q: must be one of mechanism, opinion", s)
	}
}

func validateSemver(v string) error {
	if !semverPattern.MatchString(v) {
		return fmt.Errorf("version must be semver")
	}
	return nil
}

func validateFixtures(f Fixtures) error {
	if len(f.Positive) == 0 {
		return fmt.Errorf("fixtures.positive must contain at least one entry")
	}
	if len(f.Negative) == 0 {
		return fmt.Errorf("fixtures.negative must contain at least one entry")
	}
	return nil
}

func validateScaffold(s Scaffold) error {
	if err := validateScaffoldTier(s.Tier); err != nil {
		return err
	}
	if s.TestCommand == "" {
		return fmt.Errorf("scaffold test_command is required")
	}
	if len(s.UseWhen) == 0 {
		return fmt.Errorf("scaffold use_when is required")
	}
	if len(s.Assumes) == 0 {
		return fmt.Errorf("scaffold assumes is required")
	}
	if len(s.PairsWith.Rules) == 0 && len(s.PairsWith.Scaffolds) == 0 && s.PairsWith.SDK == "" {
		return fmt.Errorf("scaffold pairs_with is required")
	}
	for key, value := range s.SampleConfig {
		if _, ok := value.(string); !ok {
			return fmt.Errorf("scaffold sample_config %q must be string", key)
		}
	}
	return nil
}

func validateScaffoldTier(tier string) error {
	switch tier {
	case "complete", "skeleton":
		return nil
	default:
		return fmt.Errorf("scaffold tier must be complete or skeleton")
	}
}

func validateSDK(sdk *SDK) error {
	if sdk == nil {
		return nil
	}
	if sdk.Module == "" {
		return fmt.Errorf("sdk module is required")
	}
	if sdk.Version == "" {
		return fmt.Errorf("sdk version is required")
	}
	if len(sdk.Provides) == 0 {
		return fmt.Errorf("sdk provides is required")
	}
	return nil
}

func validateToolConfig(tc ToolConfigEntry) error {
	if tc.Tool == "" {
		return fmt.Errorf("tool_config tool is required")
	}
	if tc.File == "" {
		return fmt.Errorf("tool_config file is required")
	}

	hasID := tc.ID != ""
	hasRequiredBy := tc.RequiredBy != ""
	if hasID == hasRequiredBy {
		return fmt.Errorf("tool_config must have exactly one of id or required_by")
	}

	if hasID {
		if tc.RiskClass == "" {
			return fmt.Errorf("standalone tool_config risk_class is required")
		}
		if err := validateRiskClass(tc.RiskClass); err != nil {
			return err
		}
		if len(tc.Claims) == 0 {
			return fmt.Errorf("standalone tool_config claims are required")
		}
	} else {
		if tc.RiskClass != "" || len(tc.Claims) > 0 {
			return fmt.Errorf("supporting tool_config must not include risk_class or claims")
		}
	}

	return nil
}
