package main

import (
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/pack"
)

// migratedGoPackYAML is a go-pack-shaped manifest as it looks AFTER the flag-day
// migration: every rule declares engine: semgrep (the retired `layer` field is
// gone) and carries a non-empty `standard`. It is authored in-Go so the fixture
// is self-contained and is parsed by the REAL migrated reader (pack.ParseManifest
// -> validateEngine), not a stub. Returned from a func (not a package var/const)
// to keep no global mutable state.
func migratedGoPackYAML() []byte {
	return []byte(`name: backstop/go-pack
version: 1.0.0
language: go
archetype: enforcement
description: Migrated go-pack — semgrep-engine rules each carrying a non-empty standard
content:
  ruleset:
    version: 1.0.0
    rules:
      - id: no-eval
        engine: semgrep
        standard: "No eval — direct dynamic execution is forbidden"
        rule_path: semgrep/no-eval.yml
        risk_class: security
        claims:
          - id: gp-no-eval
            text: Migrated semgrep rule fires under the engine model.
            fixtures:
              positive:
                - fixtures/rules/no-eval/positive.go
              negative:
                - fixtures/rules/no-eval/negative.go
      - id: no-panic
        engine: semgrep
        standard: "No panic in library code — return errors instead"
        rule_path: semgrep/no-panic.yml
        risk_class: correctness
        claims:
          - id: gp-no-panic
            text: Second migrated semgrep rule, also standard-bearing.
            fixtures:
              positive:
                - fixtures/rules/no-panic/positive.go
              negative:
                - fixtures/rules/no-panic/negative.go
`)
}

// legacyLayerOnlyYAML is a pre-migration rule that declares only `layer: 2` and
// NO engine. Under the migrated reader the retired layer field is not read, so the
// rule's engine is empty — which must fail loud (no layer:2 -> engine:semgrep
// aliasing). Authored in-Go for a self-contained fixture. Returned from a func to
// keep no global mutable state.
func legacyLayerOnlyYAML() string {
	return `name: test-org/legacy-pack
version: 1.0.0
language: go
archetype: enforcement
description: Legacy layer-2-only rule with no engine for the no-grandfather proof
content:
  ruleset:
    version: 1.0.0
    rules:
      - id: legacy-no-eval
        layer: 2
        standard: "Legacy layer-2 rule with no engine field"
        rule_path: rules/no-eval.yml
        risk_class: security
        claims:
          - id: lg-no-eval
            text: Legacy rule reaching migrated reader must fail loud.
            fixtures:
              positive:
                - fixtures/rules/legacy-no-eval/positive.go
              negative:
                - fixtures/rules/legacy-no-eval/negative.go
`
}

// TestMigration_GoPackEngineSemgrep proves a migrated semgrep rule (engine:
// semgrep, the retired layer field gone) parses and VALIDATES cleanly under the
// real migrated reader (CLM-037 / REQ-015): pack.ParseManifest accepts the
// go-pack-shaped manifest and each rule's engine resolves to the registered
// semgrep binding. Substantive: parses real go-pack-shaped YAML through the real
// reader and asserts the rules survive with engine == "semgrep".
func TestMigration_GoPackEngineSemgrep(t *testing.T) {
	m, err := pack.ParseManifest(migratedGoPackYAML())
	if err != nil {
		t.Fatalf("migrated go-pack (engine: semgrep) must parse+validate under the migrated reader, got: %v", err)
	}
	if len(m.Content.Ruleset.Rules) != 2 {
		t.Fatalf("expected 2 migrated rules, got %d", len(m.Content.Ruleset.Rules))
	}
	for _, r := range m.Content.Ruleset.Rules {
		if r.Engine != "semgrep" {
			t.Errorf("migrated rule %q must declare engine: semgrep, got %q", r.ID, r.Engine)
		}
		// ParseManifest invokes validateEngine internally; a clean parse with the
		// rule's engine intact confirms the migrated reader resolved it (a known
		// engine validates, an unknown one would have failed the parse above).
		if r.NamespacedID == "" {
			t.Errorf("migrated rule %q must be namespaced by the reader, got empty NamespacedID", r.ID)
		}
	}
}

// TestMigration_GoPackRulesRetainStandard proves the migrated semgrep rules each
// carry a non-empty `standard`, satisfying the re-keyed semgrep field-contract
// (CLM-037 / REQ-015). Substantive: asserts every migrated rule preserves its
// standard string through the real reader (the migration did not drop it).
func TestMigration_GoPackRulesRetainStandard(t *testing.T) {
	m, err := pack.ParseManifest(migratedGoPackYAML())
	if err != nil {
		t.Fatalf("migrated go-pack must parse: %v", err)
	}
	for _, r := range m.Content.Ruleset.Rules {
		if strings.TrimSpace(r.Standard) == "" {
			t.Errorf("migrated semgrep rule %q must retain a non-empty standard (re-keyed field-contract), got empty", r.ID)
		}
	}
	// And the two standards are the distinct, real strings (not a single defaulted
	// value), proving the field genuinely survived per-rule.
	got := map[string]bool{}
	for _, r := range m.Content.Ruleset.Rules {
		got[r.Standard] = true
	}
	if len(got) != 2 {
		t.Errorf("each migrated rule must keep its OWN standard; expected 2 distinct standards, got %d: %v", len(got), got)
	}
}

// TestMigration_NoSilentGrandfather proves a legacy layer-2-only rule (no engine)
// reaching the migrated reader fails loud with NO layer:2 -> engine:semgrep alias
// (CLM-038 / REQ-002/REQ-015): the retired layer field is not read, so the empty
// engine is a blocking config error. Substantive: parses a real legacy layer-only
// manifest through the real reader and asserts it errors (never silently
// grandfathered to semgrep) and that the error names the missing engine.
func TestMigration_NoSilentGrandfather(t *testing.T) {
	_, err := pack.ParseManifest([]byte(legacyLayerOnlyYAML()))
	if err == nil {
		t.Fatal("a legacy layer-2-only rule (no engine) must fail loud under the migrated reader — got nil, that is a silent grandfather")
	}
	if !strings.Contains(err.Error(), "engine") {
		t.Errorf("the no-grandfather error must name the missing engine, got: %v", err)
	}
	// There must be NO layer:2 -> semgrep aliasing: the SAME manifest with the
	// layer field removed entirely (an explicitly engine-less rule) must fail the
	// reader identically — proving the rejection is driven by the empty engine, not
	// by the presence of the legacy `layer` key, and that nothing aliases empty ->
	// semgrep.
	engineless := strings.Replace(legacyLayerOnlyYAML(), "        layer: 2\n", "", 1)
	if engineless == legacyLayerOnlyYAML() {
		t.Fatal("test setup: expected to strip the legacy layer key")
	}
	if _, err := pack.ParseManifest([]byte(engineless)); err == nil {
		t.Error("an engine-less rule must be a blocking config error (no layer:2->engine:semgrep alias), got nil")
	}
}
