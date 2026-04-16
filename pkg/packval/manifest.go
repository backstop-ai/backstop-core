package packval

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type PackManifest struct {
	Name       string            `json:"name" yaml:"name"`
	Version    string            `json:"version" yaml:"version"`
	Language   string            `json:"language" yaml:"language"`
	Archetype  string            `json:"archetype" yaml:"archetype"`
	Content    Content           `json:"content" yaml:"content"`
	ToolConfig []ToolConfigEntry `json:"tool_config,omitempty" yaml:"tool_config,omitempty"`
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
	ID            string    `json:"id" yaml:"id"`
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

type ToolConfigEntry struct {
	ID         string  `json:"id,omitempty" yaml:"id,omitempty"`
	RequiredBy string  `json:"required_by,omitempty" yaml:"required_by,omitempty"`
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
		return err
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
	ID           string            `json:"id" yaml:"id"`
	Path         string            `json:"path" yaml:"path"`
	Tier         string            `json:"tier,omitempty" yaml:"tier,omitempty"`
	TestCommand  string            `json:"test_command,omitempty" yaml:"test_command,omitempty"`
	SampleConfig map[string]string `json:"sample_config,omitempty" yaml:"sample_config,omitempty"`
	PairsWith    PairsWith         `json:"pairs_with,omitempty" yaml:"pairs_with,omitempty"`
}

type SDK struct {
	Provides []string `json:"provides,omitempty" yaml:"provides,omitempty"`
}

func ParseManifest(path string) (*PackManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var out PackManifest
	if err := yaml.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
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
