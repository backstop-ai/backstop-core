package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/config"
)

// writePolicyFixture writes an inline backstop.yml fixture to a temp dir and returns its
// path — the same load/schema-validate path the live gate uses (LoadConfigFromPath runs
// schema validation before the strict unmarshal).
func writePolicyFixture(t *testing.T, yml string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "backstop.yml")
	if err := os.WriteFile(path, []byte(yml), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestSchema_AppliesToForm_Accepted proves the backstop-yml/v1 schema now DESCRIBES and
// ACCEPTS the enforcement.policy block in its applies-to form (CLM-005): a policy using
// applies-to: new-code / all-code + level: block, including a nested
// sources.backstop/self.applies-to: all-code override, validates with NO schema error.
func TestSchema_AppliesToForm_Accepted(t *testing.T) {
	path := writePolicyFixture(t, `project: appliesto-accept
packs:
    backstop/self: local
enforcement:
    policy:
        artifact_validation:
            level: block
        pack_engines:
            level: block
            applies-to: new-code
            sources:
                backstop/self:
                    level: block
                    applies-to: all-code
        coverage_threshold:
            level: block
            applies-to: new-code
`)
	if _, err := config.LoadConfigFromPath(path); err != nil {
		t.Fatalf("the applies-to policy form (with a nested sources override) must validate against the schema, got: %v", err)
	}
}

// TestSchema_LegacyBaselineKey_Rejected proves the clean cutover leaves NO schema-legal
// path back to the retired key (CLM-005): a policy entry still using the old
// `baseline: true` key is REJECTED by schema validation (additionalProperties:false on
// the dimension object forbids the retired key).
func TestSchema_LegacyBaselineKey_Rejected(t *testing.T) {
	path := writePolicyFixture(t, `project: legacy-baseline-reject
enforcement:
    policy:
        pack_engines:
            level: block
            baseline: true
`)
	_, err := config.LoadConfigFromPath(path)
	if err == nil {
		t.Fatal("the retired `baseline:` policy key must be REJECTED by the schema (additionalProperties:false) — the clean cutover leaves no path back")
	}
	if !strings.Contains(err.Error(), "schema") && !strings.Contains(err.Error(), "baseline") {
		t.Errorf("rejection must be a schema-validation error mentioning the offending key, got: %v", err)
	}
}

// TestSchema_UnknownPolicyKey_Rejected proves the policy dimension object is now a real
// fence, not an open object (CLM-005): an invented key (an underscore typo of the new
// key) is rejected.
func TestSchema_UnknownPolicyKey_Rejected(t *testing.T) {
	path := writePolicyFixture(t, `project: unknown-key-reject
enforcement:
    policy:
        pack_engines:
            level: block
            applies_to: new-code
`)
	if _, err := config.LoadConfigFromPath(path); err == nil {
		t.Fatal("an unknown policy dimension key (applies_to underscore typo) must be REJECTED — the schema is a real fence")
	}
}

// TestSchema_AppliesToEnum_Rejected proves the applies-to enum is constrained (CLM-005):
// a value outside new-code|all-code is rejected.
func TestSchema_AppliesToEnum_Rejected(t *testing.T) {
	path := writePolicyFixture(t, `project: appliesto-enum-reject
enforcement:
    policy:
        pack_engines:
            level: block
            applies-to: sometimes
`)
	if _, err := config.LoadConfigFromPath(path); err == nil {
		t.Fatal("an applies-to value outside the new-code|all-code enum (sometimes) must be REJECTED by the schema")
	}
}

// TestSchema_LevelEnum_Rejected proves the level enum is constrained (CLM-005): a
// syntactically-valid string that is not a member of off|warn|block is rejected (the
// strict decode accepts any string for a known field, so the enum is the fence).
func TestSchema_LevelEnum_Rejected(t *testing.T) {
	path := writePolicyFixture(t, `project: level-enum-reject
enforcement:
    policy:
        pack_engines:
            level: loud
            applies-to: new-code
`)
	if _, err := config.LoadConfigFromPath(path); err == nil {
		t.Fatal("a level value outside the off|warn|block enum (loud) must be REJECTED by the schema")
	}
}

// TestSchema_SourceOverrideEnum_Rejected proves the enum constraint descends into the
// nested per-source override (CLM-005): an invalid applies-to on
// sources.backstop/self is rejected just like the top-level dimension.
func TestSchema_SourceOverrideEnum_Rejected(t *testing.T) {
	path := writePolicyFixture(t, `project: source-enum-reject
packs:
    backstop/self: local
enforcement:
    policy:
        pack_engines:
            level: block
            applies-to: new-code
            sources:
                backstop/self:
                    level: block
                    applies-to: whenever
`)
	if _, err := config.LoadConfigFromPath(path); err == nil {
		t.Fatal("an invalid applies-to on a per-source override must be REJECTED — the enum fence descends into sources")
	}
}
