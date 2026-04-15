package pack

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Manifest is the top-level pack manifest.
type Manifest struct {
	Name           string            `yaml:"name"`
	NormalizedName string            `yaml:"-"`
	Version        string            `yaml:"version"`
	Language       string            `yaml:"language"`
	Archetype      string            `yaml:"archetype"`
	Description    string            `yaml:"description"`
	Content        Content           `yaml:"content"`
	ToolConfig     []ToolConfigEntry `yaml:"tool_config"`
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

// Rule defines a single rule entry.
type Rule struct {
	ID            string    `yaml:"id"`
	NamespacedID  string    `yaml:"-"`
	Standard      string    `yaml:"standard"`
	RulePath      string    `yaml:"rule_path"`
	RiskClass     string    `yaml:"risk_class"`
	Layer         int       `yaml:"layer"`
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

	if manifest.Content.Ruleset.Version == "" && manifest.Archetype == "enforcement" {
		manifest.Content.Ruleset.Version = manifest.Version
	}
	if manifest.Content.Ruleset.Version != "" {
		if err := validateSemver(manifest.Content.Ruleset.Version); err != nil {
			return nil, fmt.Errorf("invalid ruleset version: %w", err)
		}
	}

	for i := range manifest.Content.Ruleset.Rules {
		rule := &manifest.Content.Ruleset.Rules[i]
		if err := ValidateRuleID(rule.ID); err != nil {
			return nil, fmt.Errorf("invalid rule id %q: %w", rule.ID, err)
		}
		if err := validateRiskClass(rule.RiskClass); err != nil {
			return nil, err
		}
		if err := validateLayer(rule.Layer); err != nil {
			return nil, err
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
	if manifest.Content.SDK != nil && manifest.Content.SDK.Version != "" {
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

func validateLayer(l int) error {
	switch l {
	case 1, 2, 3:
		return nil
	default:
		return fmt.Errorf("layer must be one of 1,2,3")
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
