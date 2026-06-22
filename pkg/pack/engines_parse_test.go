package pack_test

import (
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/pack"
	"github.com/bmanson/backstop-core/pkg/pack/engine"
)

// enginesPackYAML returns a minimal enforcement pack.yml carrying the given
// top-level `engines:` block plus a single content rule, so the engines-block
// parse tests can drive the full ParseManifest path (which requires content).
// The rule binds the `config-file` built-in so the rule's own engine field is
// valid regardless of what the engines: block under test declares.
func enginesPackYAML(enginesBlock string) string {
	return `name: acme/engines-demo
version: 1.0.0
language: go
archetype: enforcement
description: engines block parse fixture
` + enginesBlock + `
content:
  ruleset:
    version: 1.0.0
    rules:
      - id: demo-rule
        engine: config-file
        risk_class: security
        claims:
          - id: claim-1
            text: must enforce behavior
            fixtures:
              positive:
                - fixtures/rules/demo-rule/positive.go
              negative:
                - path: fixtures/rules/demo-rule/negative.go
                  bypass_attempt: true
`
}

// TestManifest_EnginesBlockParsesBindingFields asserts a manifest carrying a
// top-level engines: block parses each binding spec into an engine.EngineBinding
// via its yaml tags — Command, InputMode, InputFlag, Convert, Provision,
// ScopeKind, Category — readable off the resolved binding. CLM-001.
func TestManifest_EnginesBlockParsesBindingFields(t *testing.T) {
	body := enginesPackYAML(`engines:
  acme-findings:
    command: acme-scan --sarif --quiet
    input_mode: rule-flags
    input_flag: --config
    scope_kind: file-args
    category: opinion
    gate_type: findings
    provision:
      tool: acme-scan
      version: 2.3.1
    convert: acme-findings/to-sarif.sh`)

	m, err := pack.ParseManifestFile(writePackYAML(t, body))
	if err != nil {
		t.Fatalf("parse engines block: %v", err)
	}

	spec, ok := m.Engines["acme-findings"]
	if !ok {
		t.Fatalf("engine acme-findings not parsed into Engines map; got %#v", m.Engines)
	}
	b := spec.Binding

	if b.Command != "acme-scan --sarif --quiet" {
		t.Errorf("Command = %q, want %q", b.Command, "acme-scan --sarif --quiet")
	}
	if b.InputMode != engine.InputModeRuleFlags {
		t.Errorf("InputMode = %q, want rule-flags", b.InputMode)
	}
	if b.InputFlag != "--config" {
		t.Errorf("InputFlag = %q, want --config", b.InputFlag)
	}
	if b.Convert != "acme-findings/to-sarif.sh" {
		t.Errorf("Convert = %q, want acme-findings/to-sarif.sh", b.Convert)
	}
	if b.ScopeKind != engine.ScopeKindFileArgs {
		t.Errorf("ScopeKind = %v, want ScopeKindFileArgs", b.ScopeKind)
	}
	if b.Category != engine.EngineCategoryOpinion {
		t.Errorf("Category = %v, want EngineCategoryOpinion", b.Category)
	}
	if b.Provision == nil {
		t.Fatalf("Provision = nil, want non-nil")
	}
	if b.Provision.Tool != "acme-scan" || b.Provision.Version != "2.3.1" {
		t.Errorf("Provision = %+v, want {acme-scan 2.3.1}", *b.Provision)
	}
}

// TestManifest_GateTypeParsesAllSevenValues asserts a pack engines: block parses
// a declared gate_type into the binding's GateType for each of the seven valid
// values lint/build/test/findings/coverage/substantiveness/contracts. CLM-020.
func TestManifest_GateTypeParsesAllSevenValues(t *testing.T) {
	cases := map[string]engine.GateType{
		"lint":            engine.GateTypeLint,
		"build":           engine.GateTypeBuild,
		"test":            engine.GateTypeTest,
		"findings":        engine.GateTypeFindings,
		"coverage":        engine.GateTypeCoverage,
		"substantiveness": engine.GateTypeSubstantiveness,
		"contracts":       engine.GateTypeContracts,
	}
	for spelling, want := range cases {
		body := enginesPackYAML(`engines:
  acme-engine:
    command: acme-scan
    input_mode: none
    scope_kind: project-wide
    category: mechanism
    gate_type: ` + spelling)

		m, err := pack.ParseManifestFile(writePackYAML(t, body))
		if err != nil {
			t.Fatalf("gate_type %q: parse: %v", spelling, err)
		}
		got := m.Engines["acme-engine"].Binding.GateType
		if got != want {
			t.Errorf("gate_type %q parsed to GateType %v, want %v", spelling, got, want)
		}
	}
}

// TestManifest_UnknownGateTypeFailsLoud asserts an unrecognized gate_type value
// in a pack engines: block is a blocking config error at parse time — fail-loud,
// no silent default. CLM-021. (Distinct from the engine-package ParseGateType
// unit test of the same name: this drives the value through ParseManifest.)
func TestManifest_UnknownGateTypeFailsLoud(t *testing.T) {
	body := enginesPackYAML(`engines:
  acme-bogus:
    command: acme-scan
    input_mode: rule-flags
    input_flag: --config
    scope_kind: file-args
    category: opinion
    gate_type: teleport`)

	_, err := pack.ParseManifestFile(writePackYAML(t, body))
	if err == nil {
		t.Fatal("unknown gate_type must fail loud at the manifest parser, got nil error")
	}
	if !strings.Contains(err.Error(), "teleport") {
		t.Errorf("error must name the offending gate_type value, got: %v", err)
	}
}

// TestManifest_RulePatternParsed asserts the Rule struct exposes a first-class
// Pattern string parsed from pack.yml for pattern-arg engines. CLM-001 (Rule
// field surface).
func TestManifest_RulePatternParsed(t *testing.T) {
	// The Pattern field is a plain Rule-struct field parsed independently of the
	// rule's engine, so this test binds the built-in config-file engine to isolate
	// the Pattern parse from the validateEngine widening (TASK-016).
	body := `name: acme/pattern-demo
version: 1.0.0
language: go
archetype: enforcement
description: rule pattern parse fixture
content:
  ruleset:
    version: 1.0.0
    rules:
      - id: demo-rule
        engine: config-file
        risk_class: security
        pattern: "comment.contains('TODO')"
        claims:
          - id: claim-1
            text: must enforce behavior
            fixtures:
              positive:
                - fixtures/rules/demo-rule/positive.go
              negative:
                - path: fixtures/rules/demo-rule/negative.go
                  bypass_attempt: true
`
	m, err := pack.ParseManifestFile(writePackYAML(t, body))
	if err != nil {
		t.Fatalf("parse rule pattern: %v", err)
	}
	if got := m.Content.Ruleset.Rules[0].Pattern; got != "comment.contains('TODO')" {
		t.Errorf("Rule.Pattern = %q, want comment.contains('TODO')", got)
	}
}
