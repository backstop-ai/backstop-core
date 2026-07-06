package pack_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/pack"
)

// enginePackYAML returns a minimal valid enforcement pack.yml using the given
// rule fields block, so parse-level tests can exercise the migrated reader.
func enginePackYAML(ruleFields string) string {
	return `name: acme/demo
version: 1.0.0
language: go
archetype: enforcement
description: engine field parse fixture
content:
  ruleset:
    version: 1.0.0
    rules:
      - id: demo-rule
        risk_class: security
` + ruleFields + `
        claims:
          - id: claim-1
            text: must enforce behavior
            fixtures:
              positive:
                - path: fixtures/rules/demo-rule/positive.go
                  bypass_attempt: true
              negative:
                - fixtures/rules/demo-rule/negative.go
`
}

func writePackYAML(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "pack.yml")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write pack.yml: %v", err)
	}
	return path
}

// TestManifest_EngineFieldParsed asserts the Rule struct exposes a first-class
// engine string parsed from pack.yml. CLM-001.
func TestManifest_EngineFieldParsed(t *testing.T) {
	body := enginePackYAML("        engine: semgrep\n        standard: No eval\n        rule_path: semgrep/demo.yml")
	m, err := pack.ParseManifestFile(writePackYAML(t, body))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got := m.Content.Ruleset.Rules[0].Engine; got != "semgrep" {
		t.Errorf("Engine = %q, want semgrep", got)
	}
}

// TestManifest_EngineSetNoLayerPasses asserts a rule with engine set and no
// layer parses and validates cleanly. CLM-004.
func TestManifest_EngineSetNoLayerPasses(t *testing.T) {
	body := enginePackYAML("        engine: semgrep\n        standard: No eval\n        rule_path: semgrep/demo.yml")
	m, err := pack.ParseManifestFile(writePackYAML(t, body))
	if err != nil {
		t.Fatalf("engine-set/no-layer rule must parse cleanly, got: %v", err)
	}
	errs := pack.ValidateManifest(m, baseTestRegistry())
	for _, e := range errs {
		if strings.Contains(e.Message, "semgrep") {
			t.Fatalf("engine-set rule must validate cleanly, got %#v", errs)
		}
	}
}

// TestManifest_LayerWithoutEngineFailsLoud asserts a rule with layer but no
// engine is a blocking ConfigError after cutover — no silent default. CLM-005.
func TestManifest_LayerWithoutEngineFailsLoud(t *testing.T) {
	body := enginePackYAML("        layer: 2\n        standard: No eval\n        rule_path: semgrep/demo.yml")
	_, err := pack.ParseManifestFile(writePackYAML(t, body))
	if err == nil {
		t.Fatal("a layer-only rule (no engine) must fail loud at the migrated reader, got nil error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "engine") {
		t.Errorf("error must mention the missing engine, got: %v", err)
	}
}

// TestManifest_LayerKeyNotReadAsSelector asserts the Rule struct no longer reads
// a layer YAML key into an execution selector: a rule with engine set parses
// even when a stray layer key is present (the layer key is inert), and the
// parsed rule carries the engine, not a layer-derived selector. CLM-006.
func TestManifest_LayerKeyNotReadAsSelector(t *testing.T) {
	// engine present + a stray layer key: parses, engine is the selector, layer
	// is ignored (not surfaced as a field on Rule).
	body := enginePackYAML("        engine: semgrep\n        layer: 99\n        standard: No eval\n        rule_path: semgrep/demo.yml")
	m, err := pack.ParseManifestFile(writePackYAML(t, body))
	if err != nil {
		t.Fatalf("stray layer key with engine present must parse (layer inert), got: %v", err)
	}
	if m.Content.Ruleset.Rules[0].Engine != "semgrep" {
		t.Errorf("engine must be the selector, got %q", m.Content.Ruleset.Rules[0].Engine)
	}
}

// TestManifest_NoLayerKeyedValidationRemains asserts validateLayer/
// validateLayerFields are gone — replaced by engine-keyed validation — by
// grepping the pkg/pack source for the retired identifiers. CLM-016.
func TestManifest_NoLayerKeyedValidationRemains(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		data, readErr := os.ReadFile(f)
		if readErr != nil {
			t.Fatalf("read %s: %v", f, readErr)
		}
		src := string(data)
		for _, retired := range []string{"validateLayerFields", "func validateLayer("} {
			if strings.Contains(src, retired) {
				t.Errorf("%s still references retired %q; layer-keyed validation must be replaced by engine-keyed validation", f, retired)
			}
		}
	}
}
