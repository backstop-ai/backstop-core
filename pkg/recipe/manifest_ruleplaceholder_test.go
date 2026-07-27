package recipe

import (
	"strings"
	"testing"
)

// The OTHER half of the ISSUE-079 invariant. Op.Rule is deliberately NOT
// substituted at apply time — it is validated HERE by exact string equality
// against the recipe's declared transform_rules, and it selects which rewrite
// file an allowlisted engine executes IN PLACE over the consumer's tree, so a
// consumer-supplied param must not carry that authority.
//
// Because substitution skips it, a templated rule would otherwise send a literal
// `{{` down to the engine. So the excluded field is closed at the EARLIEST point
// instead: parse.
//
// The check is `rule:` ONLY, and the ACCEPT case below is what holds that line.
//
// `fragment:` is no longer part of what this guard holds: ISSUE-081's Decision
// (2026-07-27) pinned it as path-only, and it is now separately refused when it
// carries a placeholder — see manifest_fragmentform_test.go. `payload:` is the
// residual UNDECIDED facet, and the accept case below is scoped to it alone.
//
// The asymmetry is evidence-driven, not timidity. Fragment had a live THREE-WAY
// contradiction — the applier read it as a path, the spec said nothing, and a
// committed fixture declared it inline — which is what forced a decision. Payload
// has none: renderPayload resolves it under the recipe dir, every committed
// fixture declares it as a path, and no artifact disagrees. So a future
// uniformity refactor that widens the placeholder check to `payload:` would
// settle that question from the wrong place on no evidence — and it FAILS HERE
// when it tries, which is exactly what this accept case is for.

// templatedRuleManifest declares the placeholder in `rule:` AND lists that exact
// literal string in transform_rules, so the allowlist cross-check would PASS. The
// only thing that can reject it is the placeholder check, which is the point of
// shaping the case this way.
const templatedRuleManifest = `
kind: implementing
version: 1.0.0
params:
  - name: variant
    default: rename-key
transform_rules:
  - "rules/{{ variant }}.yml"
ops:
  - id: seed-config
    kind: create
    target: config/app.settings
    payload: payload/app.settings
  - id: rewrite-entry
    kind: transform
    target: config/app.settings
    rule: "rules/{{ variant }}.yml"
    manual: Apply the rewrite by hand.
`

// templatedPayloadManifest is templatedRuleManifest with the rule made LITERAL
// and the placeholder moved to `payload:` instead. It is the SCOPE GUARD: it
// fails the moment someone widens the rule-only check to payload, which is
// exactly what it is for. It declares NO merge op — fragment's form is decided
// and guarded in manifest_fragmentform_test.go, so including one here would make
// this file fail for a reason it is not about.
const templatedPayloadManifest = `
kind: implementing
version: 1.0.0
params:
  - name: variant
    default: rename-key
transform_rules:
  - rules/rename-key.yml
ops:
  - id: seed-config
    kind: create
    target: config/app.settings
    payload: "payload/{{ variant }}.settings"
  - id: rewrite-entry
    kind: transform
    target: config/app.settings
    rule: rules/rename-key.yml
    manual: Apply the rewrite by hand.
`

// literalManifest carries no placeholder anywhere, so a check that rejected every
// rule path outright could not look green.
const literalManifest = `
kind: implementing
version: 1.0.0
transform_rules:
  - rules/rename-key.yml
ops:
  - id: rewrite-entry
    kind: transform
    target: config/app.settings
    rule: rules/rename-key.yml
    manual: Apply the rewrite by hand.
`

// TestParseRecipeManifest_TemplatedTransformRuleFailsLoud proves a placeholder in
// `rule:` — and ONLY in `rule:` — is refused at manifest validation (CLM-010),
// naming the op index, the op id and the offending FIELD. A generic "invalid op"
// tells a pack author nothing about which declaration to change.
func TestParseRecipeManifest_TemplatedTransformRuleFailsLoud(t *testing.T) {
	t.Run("a templated rule is refused at parse", func(t *testing.T) {
		_, err := ParseRecipeManifest([]byte(templatedRuleManifest))
		if err == nil {
			t.Fatal("a transform op whose rule carries a placeholder must be a validation error, got nil")
		}
		msg := err.Error()
		for _, want := range []string{"ops[1]", "rewrite-entry", "rule"} {
			if !strings.Contains(msg, want) {
				t.Errorf("error must NAME %q so the pack author can locate the declaration, got: %v", want, err)
			}
		}
		// The failure must be the PLACEHOLDER, not the allowlist: the manifest
		// declares that exact literal in transform_rules, so a message about an
		// undeclared rule would mean the checks are ordered wrong.
		if strings.Contains(msg, "not among the recipe's declared transform_rules") {
			t.Errorf("the templated rule was reported as undeclared rather than as templated; the placeholder check must run FIRST, got: %v", err)
		}
	})

	t.Run("a templated payload still parses clean", func(t *testing.T) {
		manifest, err := ParseRecipeManifest([]byte(templatedPayloadManifest))
		if err != nil {
			t.Fatalf("payload templatability is ISSUE-081's RESIDUAL open question and this check must not decide it; got error: %v", err)
		}
		byID := make(map[string]Op, len(manifest.Ops))
		for _, op := range manifest.Ops {
			byID[op.ID] = op
		}
		if !strings.Contains(byID["seed-config"].Payload, placeholderOpen) {
			t.Errorf("the parsed payload %q carries no placeholder; the scope guard would be vacuous", byID["seed-config"].Payload)
		}
	})

	t.Run("a fully literal manifest still parses clean", func(t *testing.T) {
		manifest, err := ParseRecipeManifest([]byte(literalManifest))
		if err != nil {
			t.Fatalf("a literal rule path must parse clean; got error: %v", err)
		}
		if len(manifest.Ops) != 1 || manifest.Ops[0].Rule != "rules/rename-key.yml" {
			t.Errorf("Ops = %+v, want the single transform op with its declared literal rule", manifest.Ops)
		}
	})
}
