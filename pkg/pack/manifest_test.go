package pack

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func fixtureBytes(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return data
}

func validManifestYAML(name string) string {
	return `name: ` + name + `
version: 1.2.3
language: go
archetype: enforcement
description: sample
content:
  ruleset:
    rules:
      - id: sample-rule
        standard: standards/go/sample.standard.md
        risk_class: security
        engine: config-file
        claims:
          - id: c-1
            text: sample claim
            fixtures:
              positive:
                - fixtures/positive/sample.go
              negative:
                - path: fixtures/negative/sample.go
`
}

func TestParseManifest_ValidEnforcementPack(t *testing.T) {
	m, err := ParseManifest(fixtureBytes(t, "valid-enforcement-pack.yml"))
	if err != nil {
		t.Fatalf("ParseManifest returned error: %v", err)
	}
	if m.Name != "acme/security-pack" {
		t.Fatalf("unexpected name: %q", m.Name)
	}
	if m.Archetype != "enforcement" {
		t.Fatalf("unexpected archetype: %q", m.Archetype)
	}
	if len(m.Content.Ruleset.Rules) == 0 {
		t.Fatal("expected at least one rule")
	}
}

func TestParseManifest_ValidCodePack(t *testing.T) {
	m, err := ParseManifest(fixtureBytes(t, "valid-code-pack.yml"))
	if err != nil {
		t.Fatalf("ParseManifest returned error: %v", err)
	}
	if m.Name != "acme/tooling-pack" {
		t.Fatalf("unexpected name: %q", m.Name)
	}
	if m.Archetype != "code" {
		t.Fatalf("unexpected archetype: %q", m.Archetype)
	}
}

func TestParseManifest_MissingName(t *testing.T) {
	if _, err := ParseManifest(fixtureBytes(t, "missing-name.yml")); err == nil {
		t.Fatal("expected missing name error")
	}
}

func TestParseManifest_MissingVersion(t *testing.T) {
	if _, err := ParseManifest(fixtureBytes(t, "missing-version.yml")); err == nil {
		t.Fatal("expected missing version error")
	}
}

func TestParseManifest_MissingLanguage(t *testing.T) {
	if _, err := ParseManifest(fixtureBytes(t, "missing-language.yml")); err == nil {
		t.Fatal("expected missing language error")
	}
}

func TestParseManifest_MissingArchetype(t *testing.T) {
	if _, err := ParseManifest(fixtureBytes(t, "missing-archetype.yml")); err == nil {
		t.Fatal("expected missing archetype error")
	}
}

func TestParseManifest_MissingDescription(t *testing.T) {
	if _, err := ParseManifest(fixtureBytes(t, "missing-description.yml")); err == nil {
		t.Fatal("expected missing description error")
	}
}

func TestParseManifest_MissingContent(t *testing.T) {
	if _, err := ParseManifest(fixtureBytes(t, "missing-content.yml")); err == nil {
		t.Fatal("expected missing content error")
	}
}

func TestValidateName_ValidTwoPart(t *testing.T) {
	if _, err := ParseManifest([]byte(validManifestYAML("acme/sample-pack"))); err != nil {
		t.Fatalf("expected valid name, got error: %v", err)
	}
}

func TestValidateName_NoSlash(t *testing.T) {
	if _, err := ParseManifest([]byte(validManifestYAML("acme"))); err == nil {
		t.Fatal("expected name without slash to fail")
	}
}

func TestValidateName_MultipleSlashes(t *testing.T) {
	if _, err := ParseManifest([]byte(validManifestYAML("acme/platform/sample"))); err == nil {
		t.Fatal("expected name with multiple slashes to fail")
	}
}

func TestValidateName_InvalidChars(t *testing.T) {
	if _, err := ParseManifest([]byte(validManifestYAML("acme!/sample_pack"))); err == nil {
		t.Fatal("expected invalid name characters to fail")
	}
}

func TestValidateName_NormalizedToLowercase(t *testing.T) {
	m, err := ParseManifest([]byte(validManifestYAML("AcMe/Sample-Pack")))
	if err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	if m.NormalizedName != "acme/sample-pack" {
		t.Fatalf("expected normalized lowercase name, got %q", m.NormalizedName)
	}
	if m.Name != "AcMe/Sample-Pack" {
		t.Fatalf("expected original name preserved, got %q", m.Name)
	}
}

func TestValidateLanguage_Valid(t *testing.T) {
	if err := validateLanguageField("go"); err != nil {
		t.Fatalf("expected valid language, got error: %v", err)
	}
}

func TestValidateLanguage_Empty(t *testing.T) {
	if err := validateLanguageField(""); err == nil {
		t.Fatal("expected empty language to fail")
	}
}

func TestParseStandard_Filepath(t *testing.T) {
	m, err := ParseManifest(fixtureBytes(t, "valid-enforcement-pack.yml"))
	if err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	if got := m.Content.Ruleset.Rules[0].Standard; got != "standards/go/transport.standard.md" {
		t.Fatalf("unexpected standard filepath: %q", got)
	}
}

func TestParseStandard_InlineString(t *testing.T) {
	m, err := ParseManifest(fixtureBytes(t, "valid-code-pack.yml"))
	if err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	if got := m.Content.Ruleset.Rules[0].Standard; !strings.Contains(got, "inline standard") {
		t.Fatalf("expected inline standard string, got %q", got)
	}
}

func TestParseFixture_StringPath(t *testing.T) {
	m, err := ParseManifest(fixtureBytes(t, "valid-code-pack.yml"))
	if err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	fixture := m.Content.Ruleset.Rules[0].Claims[0].Fixtures.Positive[0]
	if fixture.Path != "fixtures/positive/no-debug.go" {
		t.Fatalf("unexpected fixture path: %q", fixture.Path)
	}
	if fixture.BypassAttempt {
		t.Fatal("positive fixture should not set bypass_attempt")
	}
}

func TestParseFixture_ObjectWithBypass(t *testing.T) {
	m, err := ParseManifest(fixtureBytes(t, "valid-enforcement-pack.yml"))
	if err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	fixture := m.Content.Ruleset.Rules[0].Claims[0].Fixtures.Negative[0]
	if fixture.Path != "fixtures/negative/insecure.go" {
		t.Fatalf("unexpected fixture path: %q", fixture.Path)
	}
	if !fixture.BypassAttempt {
		t.Fatal("expected bypass_attempt=true")
	}
}

func TestParseFixture_ObjectWithoutBypass(t *testing.T) {
	yaml := `
name: acme/sample-pack
version: 1.2.3
language: go
archetype: enforcement
description: sample
content:
  ruleset:
    rules:
      - id: sample-rule
        standard: standards/go/sample.standard.md
        risk_class: security
        engine: config-file
        claims:
          - id: c-1
            text: sample
            fixtures:
              positive:
                - fixtures/positive/sample.go
              negative:
                - path: fixtures/negative/sample.go
`
	m, err := ParseManifest([]byte(yaml))
	if err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	fixture := m.Content.Ruleset.Rules[0].Claims[0].Fixtures.Negative[0]
	if fixture.Path != "fixtures/negative/sample.go" {
		t.Fatalf("unexpected fixture path: %q", fixture.Path)
	}
	if fixture.BypassAttempt {
		t.Fatal("expected bypass_attempt default false")
	}
}

func TestScaffoldType_HasPathNoUpdate(t *testing.T) {
	typ := reflect.TypeOf(Scaffold{})
	if _, ok := typ.FieldByName("Path"); !ok {
		t.Fatal("expected Scaffold.Path field")
	}
	if _, ok := typ.FieldByName("Update"); ok {
		t.Fatal("did not expect Scaffold.Update field")
	}
}

func TestSDKType_ModuleVersionProvides(t *testing.T) {
	typ := reflect.TypeOf(SDK{})
	for _, field := range []string{"Module", "Version", "Provides"} {
		// field ranges over a fixed in-test literal slice, not external input.
		// nosemgrep: go.lang.security.audit.unsafe-reflect-by-name.unsafe-reflect-by-name
		if _, ok := typ.FieldByName(field); !ok {
			t.Fatalf("expected SDK.%s field", field)
		}
	}
	if _, ok := typ.FieldByName("Distribution"); ok {
		t.Fatal("did not expect SDK.Distribution field")
	}
}

func TestValidateArchetype_Enforcement(t *testing.T) {
	if err := validateArchetype("enforcement"); err != nil {
		t.Fatalf("expected valid archetype, got: %v", err)
	}
}

func TestValidateArchetype_Code(t *testing.T) {
	if err := validateArchetype("code"); err != nil {
		t.Fatalf("expected valid archetype, got: %v", err)
	}
}

func TestValidateArchetype_Invalid(t *testing.T) {
	if err := validateArchetype("library"); err == nil {
		t.Fatal("expected invalid archetype to fail")
	}
}

func TestValidateArchetype_Missing(t *testing.T) {
	if _, err := ParseManifest([]byte(strings.ReplaceAll(validManifestYAML("acme/sample-pack"), "archetype: enforcement\n", ""))); err == nil {
		t.Fatal("expected missing archetype to fail")
	}
}

func TestValidateRiskClass_Security(t *testing.T) {
	if err := validateRiskClass("security"); err != nil {
		t.Fatalf("expected valid risk_class, got: %v", err)
	}
}

func TestValidateRiskClass_Correctness(t *testing.T) {
	if err := validateRiskClass("correctness"); err != nil {
		t.Fatalf("expected valid risk_class, got: %v", err)
	}
}

func TestValidateRiskClass_Style(t *testing.T) {
	if err := validateRiskClass("style"); err != nil {
		t.Fatalf("expected valid risk_class, got: %v", err)
	}
}

func TestValidateRiskClass_Perf(t *testing.T) {
	if err := validateRiskClass("perf"); err != nil {
		t.Fatalf("expected valid risk_class, got: %v", err)
	}
}

func TestValidateRiskClass_Invalid(t *testing.T) {
	if err := validateRiskClass("availability"); err == nil {
		t.Fatal("expected invalid risk_class to fail")
	}
}

func TestValidateRiskClass_Missing(t *testing.T) {
	if err := validateRiskClass(""); err == nil {
		t.Fatal("expected missing risk_class to fail")
	}
}

func TestValidateVersion_ValidSemver(t *testing.T) {
	if err := validateSemver("1.2.3"); err != nil {
		t.Fatalf("expected valid semver, got: %v", err)
	}
}

func TestValidateVersion_InvalidFormat(t *testing.T) {
	if err := validateSemver("v1"); err == nil {
		t.Fatal("expected invalid semver to fail")
	}
}

func TestValidateVersion_RulesetDefaultsToPackVersion(t *testing.T) {
	m, err := ParseManifest(fixtureBytes(t, "valid-enforcement-pack.yml"))
	if err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	if m.Content.Ruleset.Version != m.Version {
		t.Fatalf("expected ruleset version defaulted to %s, got %s", m.Version, m.Content.Ruleset.Version)
	}
}

func TestValidateVersion_ExplicitRulesetVersion(t *testing.T) {
	m, err := ParseManifest(fixtureBytes(t, "valid-code-pack.yml"))
	if err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	if m.Content.Ruleset.Version != "2.1.0" {
		t.Fatalf("expected explicit ruleset version, got %q", m.Content.Ruleset.Version)
	}
}

func TestValidateVersion_CodePackNoRulesetDefault(t *testing.T) {
	yaml := strings.ReplaceAll(string(fixtureBytes(t, "valid-code-pack.yml")), "    version: 2.1.0\n", "")
	m, err := ParseManifest([]byte(yaml))
	if err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	if m.Content.Ruleset.Version != "" {
		t.Fatalf("expected code pack ruleset version to remain empty, got %q", m.Content.Ruleset.Version)
	}
}

func TestValidateFixtures_BothPresent(t *testing.T) {
	if err := validateFixtures(Fixtures{
		Positive: []FixtureEntry{{Path: "a"}},
		Negative: []FixtureEntry{{Path: "b"}},
	}); err != nil {
		t.Fatalf("expected fixtures valid, got: %v", err)
	}
}

func TestValidateFixtures_NoPositive(t *testing.T) {
	if err := validateFixtures(Fixtures{
		Positive: nil,
		Negative: []FixtureEntry{{Path: "b"}},
	}); err == nil {
		t.Fatal("expected missing positive fixtures to fail")
	}
}

func TestValidateFixtures_NoNegative(t *testing.T) {
	if err := validateFixtures(Fixtures{
		Positive: []FixtureEntry{{Path: "a"}},
		Negative: nil,
	}); err == nil {
		t.Fatal("expected missing negative fixtures to fail")
	}
}

func TestParseFixture_BypassAttemptOnPositiveIgnored(t *testing.T) {
	yaml := `
name: acme/sample-pack
version: 1.2.3
language: go
archetype: enforcement
description: sample
content:
  ruleset:
    rules:
      - id: sample-rule
        standard: standards/go/sample.standard.md
        risk_class: security
        engine: config-file
        claims:
          - id: c-1
            text: sample
            fixtures:
              positive:
                - path: fixtures/positive/sample.go
                  bypass_attempt: true
              negative:
                - path: fixtures/negative/sample.go
`
	m, err := ParseManifest([]byte(yaml))
	if err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	if m.Content.Ruleset.Rules[0].Claims[0].Fixtures.Positive[0].BypassAttempt {
		t.Fatal("expected bypass_attempt on positive fixture to be ignored")
	}
}

func TestParseManifest_EmptyContent(t *testing.T) {
	yaml := `
name: acme/sample-pack
version: 1.2.3
language: go
archetype: enforcement
description: sample
content: {}
`
	if _, err := ParseManifest([]byte(yaml)); err == nil {
		t.Fatal("expected empty content to fail")
	}
}

func TestValidateScaffoldTier_Complete(t *testing.T) {
	if err := validateScaffoldTier("complete"); err != nil {
		t.Fatalf("expected valid tier, got: %v", err)
	}
}

func TestValidateScaffoldTier_Skeleton(t *testing.T) {
	if err := validateScaffoldTier("skeleton"); err != nil {
		t.Fatalf("expected valid tier, got: %v", err)
	}
}

func TestValidateScaffoldTier_Invalid(t *testing.T) {
	if err := validateScaffoldTier("full"); err == nil {
		t.Fatal("expected invalid tier to fail")
	}
}

func TestValidateScaffoldTier_Missing(t *testing.T) {
	if err := validateScaffoldTier(""); err == nil {
		t.Fatal("expected missing tier to fail")
	}
}

func TestValidateScaffold_WithTestCommand(t *testing.T) {
	s := Scaffold{
		ID:          "s-1",
		Version:     "1.0.0",
		Tier:        "complete",
		Path:        "scaffolds/s-1",
		TestCommand: "go test ./...",
		UseWhen:     []string{"apps"},
		Assumes:     []string{"go-mod"},
		PairsWith:   PairsWith{Rules: []string{"r-1"}},
	}
	if err := validateScaffold(s); err != nil {
		t.Fatalf("expected valid scaffold, got: %v", err)
	}
}

func TestValidateScaffold_MissingTestCommand(t *testing.T) {
	s := Scaffold{
		ID:        "s-1",
		Version:   "1.0.0",
		Tier:      "complete",
		Path:      "scaffolds/s-1",
		UseWhen:   []string{"apps"},
		Assumes:   []string{"go-mod"},
		PairsWith: PairsWith{Rules: []string{"r-1"}},
	}
	if err := validateScaffold(s); err == nil {
		t.Fatal("expected missing test_command to fail")
	}
}

func TestValidateScaffold_AllFieldsPresent(t *testing.T) {
	s := Scaffold{
		ID:           "s-1",
		Version:      "1.0.0",
		Tier:         "complete",
		Path:         "scaffolds/s-1",
		TestCommand:  "go test ./...",
		Description:  "desc",
		UseWhen:      []string{"apps"},
		Assumes:      []string{"go-mod"},
		PairsWith:    PairsWith{Rules: []string{"r-1"}, Scaffolds: []string{"s-2"}, SDK: "sdk/ref"},
		SampleConfig: map[string]any{"mode": "safe"},
	}
	if err := validateScaffold(s); err != nil {
		t.Fatalf("expected valid scaffold, got: %v", err)
	}
}

func TestValidateScaffold_MissingUseWhen(t *testing.T) {
	s := Scaffold{
		ID:          "s-1",
		Version:     "1.0.0",
		Tier:        "complete",
		Path:        "scaffolds/s-1",
		TestCommand: "go test ./...",
		Assumes:     []string{"go-mod"},
		PairsWith:   PairsWith{Rules: []string{"r-1"}},
	}
	if err := validateScaffold(s); err == nil {
		t.Fatal("expected missing use_when to fail")
	}
}

func TestValidateScaffold_MissingAssumes(t *testing.T) {
	s := Scaffold{
		ID:          "s-1",
		Version:     "1.0.0",
		Tier:        "complete",
		Path:        "scaffolds/s-1",
		TestCommand: "go test ./...",
		UseWhen:     []string{"apps"},
		PairsWith:   PairsWith{Rules: []string{"r-1"}},
	}
	if err := validateScaffold(s); err == nil {
		t.Fatal("expected missing assumes to fail")
	}
}

func TestValidateScaffold_MissingPairsWith(t *testing.T) {
	s := Scaffold{
		ID:          "s-1",
		Version:     "1.0.0",
		Tier:        "complete",
		Path:        "scaffolds/s-1",
		TestCommand: "go test ./...",
		UseWhen:     []string{"apps"},
		Assumes:     []string{"go-mod"},
	}
	if err := validateScaffold(s); err == nil {
		t.Fatal("expected missing pairs_with to fail")
	}
}

func TestValidateScaffold_EmptyUseWhen(t *testing.T) {
	s := Scaffold{
		ID:          "s-1",
		Version:     "1.0.0",
		Tier:        "complete",
		Path:        "scaffolds/s-1",
		TestCommand: "go test ./...",
		UseWhen:     []string{},
		Assumes:     []string{"go-mod"},
		PairsWith:   PairsWith{Rules: []string{"r-1"}},
	}
	if err := validateScaffold(s); err == nil {
		t.Fatal("expected empty use_when to fail")
	}
}

func TestValidateScaffold_SampleConfigNonString(t *testing.T) {
	s := Scaffold{
		ID:           "s-1",
		Version:      "1.0.0",
		Tier:         "complete",
		Path:         "scaffolds/s-1",
		TestCommand:  "go test ./...",
		UseWhen:      []string{"apps"},
		Assumes:      []string{"go-mod"},
		PairsWith:    PairsWith{Rules: []string{"r-1"}},
		SampleConfig: map[string]any{"retries": 3},
	}
	if err := validateScaffold(s); err == nil {
		t.Fatal("expected non-string sample_config value to fail")
	}
}

func TestParseScaffold_PairsWithAllKeys(t *testing.T) {
	m, err := ParseManifest(fixtureBytes(t, "valid-enforcement-pack.yml"))
	if err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	pairs := m.Content.Scaffolds[0].PairsWith
	if len(pairs.Rules) == 0 || len(pairs.Scaffolds) == 0 || pairs.SDK == "" {
		t.Fatalf("expected all pairs_with keys parsed, got %+v", pairs)
	}
}

func TestParsePairsWith_AllKeys(t *testing.T) {
	m, err := ParseManifest(fixtureBytes(t, "valid-enforcement-pack.yml"))
	if err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	pairs := m.Content.Ruleset.Rules[0].PairsWith
	if len(pairs.Rules) == 0 || len(pairs.Scaffolds) == 0 || pairs.SDK == "" {
		t.Fatalf("expected all pairs_with keys parsed, got %+v", pairs)
	}
}

func TestParsePairsWith_RulesOnly(t *testing.T) {
	yaml := `
name: acme/sample-pack
version: 1.2.3
language: go
archetype: enforcement
description: sample
content:
  ruleset:
    rules:
      - id: sample-rule
        standard: standards/go/sample.standard.md
        risk_class: security
        engine: config-file
        claims:
          - id: c-1
            text: sample
            fixtures:
              positive: [fixtures/positive/sample.go]
              negative: [fixtures/negative/sample.go]
        pairs_with:
          rules: [r-2]
`
	m, err := ParseManifest([]byte(yaml))
	if err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	pairs := m.Content.Ruleset.Rules[0].PairsWith
	if len(pairs.Rules) != 1 || pairs.Rules[0] != "r-2" {
		t.Fatalf("expected rules key parsed, got %+v", pairs)
	}
	if len(pairs.Scaffolds) != 0 || pairs.SDK != "" {
		t.Fatalf("expected missing keys to remain empty, got %+v", pairs)
	}
}

func TestValidateSDK_Valid(t *testing.T) {
	err := validateSDK(&SDK{
		Module:   "github.com/acme/sdk",
		Version:  "1.0.0",
		Provides: []string{"one"},
	})
	if err != nil {
		t.Fatalf("expected valid sdk, got: %v", err)
	}
}

func TestValidateSDK_MissingModule(t *testing.T) {
	err := validateSDK(&SDK{
		Version:  "1.0.0",
		Provides: []string{"one"},
	})
	if err == nil {
		t.Fatal("expected missing module to fail")
	}
}

func TestValidateSDK_MissingVersion(t *testing.T) {
	err := validateSDK(&SDK{
		Module:   "github.com/acme/sdk",
		Provides: []string{"one"},
	})
	if err == nil {
		t.Fatal("expected missing version to fail")
	}
}

func TestValidateSDK_EmptyProvides(t *testing.T) {
	err := validateSDK(&SDK{
		Module:   "github.com/acme/sdk",
		Version:  "1.0.0",
		Provides: []string{},
	})
	if err == nil {
		t.Fatal("expected empty provides to fail")
	}
}

func TestValidateSDK_MissingProvides(t *testing.T) {
	err := validateSDK(&SDK{
		Module:  "github.com/acme/sdk",
		Version: "1.0.0",
	})
	if err == nil {
		t.Fatal("expected missing provides to fail")
	}
}

func TestValidateToolConfig_Standalone(t *testing.T) {
	err := validateToolConfig(ToolConfigEntry{
		ID:        "tc-1",
		Tool:      "semgrep",
		File:      "semgrep.yml",
		RiskClass: "security",
		Claims: []Claim{{
			ID: "c-1",
		}},
	})
	if err != nil {
		t.Fatalf("expected valid standalone tool_config, got: %v", err)
	}
}

func TestValidateToolConfig_Supporting(t *testing.T) {
	err := validateToolConfig(ToolConfigEntry{
		RequiredBy: "tc-1",
		Tool:       "semgrep",
		File:       "semgrep-extra.yml",
	})
	if err != nil {
		t.Fatalf("expected valid supporting tool_config, got: %v", err)
	}
}

func TestValidateToolConfig_BothIdAndRequiredBy(t *testing.T) {
	err := validateToolConfig(ToolConfigEntry{
		ID:         "tc-1",
		RequiredBy: "tc-0",
		Tool:       "semgrep",
		File:       "semgrep.yml",
	})
	if err == nil {
		t.Fatal("expected both id and required_by to fail")
	}
}

func TestValidateToolConfig_NeitherIdNorRequiredBy(t *testing.T) {
	err := validateToolConfig(ToolConfigEntry{
		Tool: "semgrep",
		File: "semgrep.yml",
	})
	if err == nil {
		t.Fatal("expected neither id nor required_by to fail")
	}
}

func TestValidateToolConfig_StandaloneMissingRiskClass(t *testing.T) {
	err := validateToolConfig(ToolConfigEntry{
		ID:    "tc-1",
		Tool:  "semgrep",
		File:  "semgrep.yml",
		Claims: []Claim{{ID: "c-1"}},
	})
	if err == nil {
		t.Fatal("expected standalone missing risk_class to fail")
	}
}

func TestValidateToolConfig_StandaloneMissingClaims(t *testing.T) {
	err := validateToolConfig(ToolConfigEntry{
		ID:        "tc-1",
		Tool:      "semgrep",
		File:      "semgrep.yml",
		RiskClass: "security",
	})
	if err == nil {
		t.Fatal("expected standalone missing claims to fail")
	}
}

func TestValidateToolConfig_MissingTool(t *testing.T) {
	err := validateToolConfig(ToolConfigEntry{
		ID:        "tc-1",
		File:      "semgrep.yml",
		RiskClass: "security",
		Claims:    []Claim{{ID: "c-1"}},
	})
	if err == nil {
		t.Fatal("expected missing tool to fail")
	}
}

func TestValidateToolConfig_MissingFile(t *testing.T) {
	err := validateToolConfig(ToolConfigEntry{
		ID:        "tc-1",
		Tool:      "semgrep",
		RiskClass: "security",
		Claims:    []Claim{{ID: "c-1"}},
	})
	if err == nil {
		t.Fatal("expected missing file to fail")
	}
}

func TestValidateClaimIDs_Unique(t *testing.T) {
	m, err := ParseManifest(fixtureBytes(t, "valid-enforcement-pack.yml"))
	if err != nil {
		t.Fatalf("expected unique claim IDs to pass: %v", err)
	}
	for _, rule := range m.Content.Ruleset.Rules {
		if rule.NamespacedID != NamespacedRuleID(m.NormalizedName, rule.ID) {
			t.Fatalf("expected namespaced rule ID on parse, got %q", rule.NamespacedID)
		}
	}
}

func TestValidateClaimIDs_Duplicate(t *testing.T) {
	yaml := `
name: acme/sample-pack
version: 1.2.3
language: go
archetype: enforcement
description: sample
content:
  ruleset:
    rules:
      - id: rule-one
        standard: standards/go/sample.standard.md
        risk_class: security
        engine: config-file
        claims:
          - id: c-dup
            text: first
            fixtures:
              positive: [fixtures/positive/a.go]
              negative: [fixtures/negative/a.go]
      - id: rule-two
        standard: standards/go/sample.standard.md
        risk_class: security
        engine: config-file
        claims:
          - id: c-dup
            text: second
            fixtures:
              positive: [fixtures/positive/b.go]
              negative: [fixtures/negative/b.go]
`
	if _, err := ParseManifest([]byte(yaml)); err == nil {
		t.Fatal("expected duplicate claim IDs to fail")
	}
}

func TestParseManifestFile_Success(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "pack.yml")
	if err := os.WriteFile(path, fixtureBytes(t, "valid-enforcement-pack.yml"), 0o644); err != nil {
		t.Fatalf("write fixture file: %v", err)
	}
	if _, err := ParseManifestFile(path); err != nil {
		t.Fatalf("expected ParseManifestFile success, got: %v", err)
	}
}

func TestParseManifestFile_ReadError(t *testing.T) {
	if _, err := ParseManifestFile("/no/such/pack.yml"); err == nil {
		t.Fatal("expected ParseManifestFile read error")
	}
}

func TestParseManifest_InvalidYAML(t *testing.T) {
	if _, err := ParseManifest([]byte("name: [")); err == nil {
		t.Fatal("expected invalid yaml to fail")
	}
}

func TestParseManifest_InvalidRuleID(t *testing.T) {
	yaml := strings.ReplaceAll(validManifestYAML("acme/sample-pack"), "id: sample-rule", "id: Sample_Rule")
	if _, err := ParseManifest([]byte(yaml)); err == nil {
		t.Fatal("expected invalid rule id to fail")
	}
}

func TestParseManifest_InvalidArchetype(t *testing.T) {
	yaml := strings.ReplaceAll(validManifestYAML("acme/sample-pack"), "archetype: enforcement", "archetype: library")
	if _, err := ParseManifest([]byte(yaml)); err == nil {
		t.Fatal("expected invalid archetype to fail")
	}
}

func TestParseManifest_InvalidPackVersion(t *testing.T) {
	yaml := strings.ReplaceAll(validManifestYAML("acme/sample-pack"), "version: 1.2.3", "version: v1")
	if _, err := ParseManifest([]byte(yaml)); err == nil {
		t.Fatal("expected invalid pack version to fail")
	}
}

func TestParseManifest_InvalidRulesetVersion(t *testing.T) {
	yaml := strings.ReplaceAll(validManifestYAML("acme/sample-pack"), "ruleset:\n    rules:", "ruleset:\n    version: bad\n    rules:")
	if _, err := ParseManifest([]byte(yaml)); err == nil {
		t.Fatal("expected invalid ruleset version to fail")
	}
}

func TestParseManifest_InvalidScaffoldVersion(t *testing.T) {
	yaml := `
name: acme/sample-pack
version: 1.2.3
language: go
archetype: enforcement
description: sample
content:
  ruleset:
    rules:
      - id: sample-rule
        standard: standards/go/sample.standard.md
        risk_class: security
        engine: config-file
        claims:
          - id: c-1
            text: sample
            fixtures:
              positive: [fixtures/positive/sample.go]
              negative: [fixtures/negative/sample.go]
  scaffolds:
    - id: s1
      version: bad
      tier: complete
      path: scaffolds/s1
      test_command: go test ./...
      use_when: [apps]
      assumes: [go-mod]
      pairs_with:
        rules: [sample-rule]
`
	if _, err := ParseManifest([]byte(yaml)); err == nil {
		t.Fatal("expected invalid scaffold version to fail")
	}
}

func TestParseManifest_InvalidSDKVersion(t *testing.T) {
	yaml := strings.ReplaceAll(string(fixtureBytes(t, "valid-code-pack.yml")), "version: 0.9.0", "version: bad")
	if _, err := ParseManifest([]byte(yaml)); err == nil {
		t.Fatal("expected invalid sdk version to fail")
	}
}

func TestParseFixture_InvalidType(t *testing.T) {
	yaml := strings.ReplaceAll(validManifestYAML("acme/sample-pack"), "- fixtures/positive/sample.go", "- [bad, fixture]")
	if _, err := ParseManifest([]byte(yaml)); err == nil {
		t.Fatal("expected invalid fixture type to fail")
	}
}

func TestParseFixture_ObjectMissingPath(t *testing.T) {
	yaml := strings.ReplaceAll(validManifestYAML("acme/sample-pack"), "- path: fixtures/negative/sample.go", "- bypass_attempt: true")
	if _, err := ParseManifest([]byte(yaml)); err == nil {
		t.Fatal("expected fixture object missing path to fail")
	}
}

func TestValidateToolConfig_SupportingWithRiskClassRejected(t *testing.T) {
	err := validateToolConfig(ToolConfigEntry{
		RequiredBy: "tc-1",
		Tool:       "semgrep",
		File:       "semgrep-extra.yml",
		RiskClass:  "security",
	})
	if err == nil {
		t.Fatal("expected supporting tool config with risk_class to fail")
	}
}
