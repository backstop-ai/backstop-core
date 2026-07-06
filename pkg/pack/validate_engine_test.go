package pack_test

import (
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/pack"
)

// validateEngine is exercised through ParseManifest (the migrated reader calls
// it per rule), so these tests drive a full pack.yml and assert the parse
// accepts or rejects based on the rule's engine being known in the pack-declared
// engines: block UNION the fallback registry, AND its tool being on the
// trusted-tool allowlist (SPEC-035 REQ-003/CLM-010..013).

// packWithEngineRule returns a pack.yml that declares the given engines: block
// and a single rule bound to ruleEngine, carrying the fields a rule-fed findings
// engine needs (rule_path + standard) so the field-contract validation does not
// mask the engine-known/allowlist outcome under test.
func packWithEngineRule(enginesBlock, ruleEngine string) string {
	block := ""
	if enginesBlock != "" {
		block = enginesBlock + "\n"
	}
	return `name: acme/validate-engine-demo
version: 1.0.0
language: go
archetype: enforcement
description: validateEngine widening fixture
` + block + `content:
  ruleset:
    version: 1.0.0
    rules:
      - id: demo-rule
        engine: ` + ruleEngine + `
        standard: No eval
        rule_path: rules/demo.yml
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

// TestValidateEngine_PackDeclaredEngineKnown asserts validateEngine accepts a
// rule whose engine is declared ONLY in the manifest's own engines: block (not
// the built-in registry) when that engine's tool is on the trusted-tool
// allowlist + lock-pinned. CLM-010.
func TestValidateEngine_PackDeclaredEngineKnown(t *testing.T) {
	// acme-findings is pack-declared (not a built-in) and rides semgrep — an
	// allowlisted, lock-pinned tool — so the validation-time trust gate passes.
	enginesBlock := `engines:
  acme-findings:
    command: acme-scan --sarif --quiet
    input_mode: rule-flags
    input_flag: --config
    scope_kind: file-args
    category: opinion
    gate_type: findings
    provision:
      tool: semgrep
      version: 1.96.0`
	body := packWithEngineRule(enginesBlock, "acme-findings")

	if _, err := pack.ParseManifestFile(writePackYAML(t, body)); err != nil {
		t.Fatalf("pack-declared engine with allowlisted tool must validate, got: %v", err)
	}
}

// TestValidateEngine_BuiltinEngineKnown asserts validateEngine accepts a rule on
// a built-in/fallback engine whose tool is allowlisted (semgrep is a built-in
// with an allowlisted, lock-pinned Provision). CLM-011.
func TestValidateEngine_BuiltinEngineKnown(t *testing.T) {
	body := packWithEngineRule("", "semgrep")

	if _, err := pack.ParseManifestFile(writePackYAML(t, body)); err != nil {
		t.Fatalf("built-in engine with allowlisted tool must validate, got: %v", err)
	}
}

// TestValidateEngine_UnknownEngineRejected asserts a rule whose engine is unknown to
// BOTH the pack-declared engines: block and the injected base registry is rejected as
// a blocking validation error (CLM-012). After ISSUE-027 deleted the baked fallback,
// parse-time validateEngine no longer resolves built-ins (it has no base): it accepts
// an undeclared name and DEFERS the unknown-engine fail-loud to the validation/gate
// layer where the base pack is merged. ValidateManifest — given the injected base —
// reports the unknown-engine violation; the gate's dispatch Lookup fail-louds
// identically at runtime.
func TestValidateEngine_UnknownEngineRejected(t *testing.T) {
	body := packWithEngineRule("", "totally-unknown-engine")

	m, err := pack.ParseManifestFile(writePackYAML(t, body))
	if err != nil {
		t.Fatalf("parse defers unknown-engine detection to the gate; ParseManifest must accept the undeclared name, got: %v", err)
	}

	errs := pack.ValidateManifest(m, baseTestRegistry())
	found := false
	for _, e := range errs {
		if e.Rule == "CLM-020-unknown-engine" && strings.Contains(e.Message, "totally-unknown-engine") {
			found = true
		}
	}
	if !found {
		t.Fatalf("an engine unknown to both pack-declared and the injected base must be rejected at validation, got %#v", errs)
	}
}

// TestValidateEngine_KnownEngineUnallowlistedToolRejected asserts validateEngine
// rejects (a blocking config error) a rule whose engine is KNOWN (declared in the
// engines: block) but whose tool is NOT on the trusted-tool allowlist. CLM-013.
func TestValidateEngine_KnownEngineUnallowlistedToolRejected(t *testing.T) {
	// acme-rogue is a well-formed pack-declared engine, but it rides a provisioned
	// tool ("rogue-tool") that is absent from TrustedToolAllowlist — so even though
	// the engine is KNOWN, the validation-time trust gate rejects it.
	enginesBlock := `engines:
  acme-rogue:
    command: rogue-tool --sarif
    input_mode: rule-flags
    input_flag: --config
    scope_kind: file-args
    category: opinion
    gate_type: findings
    provision:
      tool: rogue-tool
      version: 9.9.9`
	body := packWithEngineRule(enginesBlock, "acme-rogue")

	_, err := pack.ParseManifestFile(writePackYAML(t, body))
	if err == nil {
		t.Fatal("a known engine whose tool is not allowlisted must be rejected, got nil error")
	}
	if !strings.Contains(err.Error(), "rogue-tool") {
		t.Errorf("error must name the un-allowlisted tool, got: %v", err)
	}
}
