package schema_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// schemaJSON loads a schema file as a generic JSON map for enum inspection.
func schemaJSON(t *testing.T, rel ...string) map[string]any {
	t.Helper()
	root := repoRoot(t)
	parts := append([]string{root}, rel...)
	data, err := os.ReadFile(filepath.Join(parts...))
	if err != nil {
		t.Fatalf("read schema %v: %v", rel, err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal schema %v: %v", rel, err)
	}
	return m
}

// enumAt walks a dotted path through the JSON map and returns the enum string
// slice found there. Fails the test if the path or enum is missing.
func enumAt(t *testing.T, m map[string]any, path ...string) []string {
	t.Helper()
	cur := any(m)
	for _, key := range path {
		obj, ok := cur.(map[string]any)
		if !ok {
			t.Fatalf("path %v: %q is not an object", path, key)
		}
		cur, ok = obj[key]
		if !ok {
			t.Fatalf("path %v: missing key %q", path, key)
		}
	}
	raw, ok := cur.([]any)
	if !ok {
		t.Fatalf("path %v: not an array", path)
	}
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		s, ok := v.(string)
		if !ok {
			t.Fatalf("path %v: enum value %v is not a string", path, v)
		}
		out = append(out, s)
	}
	return out
}

func enumContains(enum []string, want string) bool {
	for _, e := range enum {
		if e == want {
			return true
		}
	}
	return false
}

func assertHas(t *testing.T, enum []string, label string, wants ...string) {
	t.Helper()
	for _, w := range wants {
		if !enumContains(enum, w) {
			t.Errorf("%s enum missing %q; got %v", label, w, enum)
		}
	}
}

func assertLacks(t *testing.T, enum []string, label string, unwants ...string) {
	t.Helper()
	for _, u := range unwants {
		if enumContains(enum, u) {
			t.Errorf("%s enum should NOT contain %q; got %v", label, u, enum)
		}
	}
}

func TestSchema_SpecV1_HasTerminalStates(t *testing.T) {
	m := schemaJSON(t, "artifacts", "spec", "v1", "schema.json")
	enum := enumAt(t, m, "metadata", "properties", "status", "enum")
	assertHas(t, enum, "spec/v1 status", "replaced", "canceled", "deprecated")
	// Originals must remain.
	assertHas(t, enum, "spec/v1 status", "draft", "ready-for-implementation", "implemented")
}

func TestSchema_BundleV1V2_HasDeliveredAndRetirementStates(t *testing.T) {
	for _, ver := range []string{"v1", "v2"} {
		m := schemaJSON(t, "artifacts", "bundle", ver, "schema.json")
		enum := enumAt(t, m, "nested_blocks", "status", "properties", "maturity", "enum")
		label := "bundle/" + ver + " maturity"
		assertHas(t, enum, label, "delivered", "replaced", "canceled", "deprecated")
		// Originals must remain.
		assertHas(t, enum, label, "idea", "exploring", "defined", "ready")
	}
}

func TestSchema_PlanV1_HasRetirementStatesNoDeprecated(t *testing.T) {
	m := schemaJSON(t, "artifacts", "plan", "v1", "schema.json")
	enum := enumAt(t, m, "metadata", "properties", "status", "enum")
	assertHas(t, enum, "plan/v1 status", "replaced", "canceled")
	assertHas(t, enum, "plan/v1 status", "draft", "ready", "implementing", "completed")
	assertLacks(t, enum, "plan/v1 status", "deprecated")
}

func TestSchema_DirectiveV1_HasRetirementStatesNoDeprecated(t *testing.T) {
	m := schemaJSON(t, "artifacts", "directive", "v1", "schema.json")
	enum := enumAt(t, m, "nested_blocks", "directive", "properties", "status", "enum")
	assertHas(t, enum, "directive/v1 status", "replaced", "canceled")
	assertHas(t, enum, "directive/v1 status", "queued", "active", "specced", "done")
	assertLacks(t, enum, "directive/v1 status", "deprecated")
}

func TestSchema_IssueV1_HasRetirementStatesKeepsClosed(t *testing.T) {
	m := schemaJSON(t, "artifacts", "issue", "v1", "schema.json")
	enum := enumAt(t, m, "nested_blocks", "issue", "properties", "status", "enum")
	assertHas(t, enum, "issue/v1 status", "replaced", "canceled")
	assertHas(t, enum, "issue/v1 status", "open", "ready", "in-progress", "blocked", "closed")
	assertLacks(t, enum, "issue/v1 status", "deprecated")
}

// TestSchema_TerminalStates_IncludeObsoleted (CLM-010): the `obsoleted` retirement
// terminal is documented in the issue/v1, spec/v1, and plan/v1 status enums. Pins the
// schema-documentation claim directly rather than only transitively via the validator.
func TestSchema_TerminalStates_IncludeObsoleted(t *testing.T) {
	iss := schemaJSON(t, "artifacts", "issue", "v1", "schema.json")
	assertHas(t, enumAt(t, iss, "nested_blocks", "issue", "properties", "status", "enum"),
		"issue/v1 status", "obsoleted")

	spec := schemaJSON(t, "artifacts", "spec", "v1", "schema.json")
	assertHas(t, enumAt(t, spec, "metadata", "properties", "status", "enum"),
		"spec/v1 status", "obsoleted")

	plan := schemaJSON(t, "artifacts", "plan", "v1", "schema.json")
	assertHas(t, enumAt(t, plan, "metadata", "properties", "status", "enum"),
		"plan/v1 status", "obsoleted")
}

func TestSchema_AdrStandardCapability_Unchanged(t *testing.T) {
	// adr/v2 status enum is unchanged.
	adr := schemaJSON(t, "artifacts", "adr", "v2", "schema.json")
	adrEnum := enumAt(t, adr, "metadata", "properties", "status", "enum")
	wantADR := []string{"Proposed", "Accepted", "Deprecated", "Superseded"}
	if len(adrEnum) != len(wantADR) {
		t.Fatalf("adr/v2 status enum = %v, want %v", adrEnum, wantADR)
	}
	for i, s := range wantADR {
		if adrEnum[i] != s {
			t.Errorf("adr/v2 status enum[%d] = %q, want %q", i, adrEnum[i], s)
		}
	}

	// standard/v1 status enum is unchanged.
	std := schemaJSON(t, "artifacts", "standard", "v1", "schema.json")
	stdEnum := enumAt(t, std, "metadata", "properties", "status", "enum")
	wantSTD := []string{"draft", "active", "deprecated"}
	if len(stdEnum) != len(wantSTD) {
		t.Fatalf("standard/v1 status enum = %v, want %v", stdEnum, wantSTD)
	}
	for i, s := range wantSTD {
		if stdEnum[i] != s {
			t.Errorf("standard/v1 status enum[%d] = %q, want %q", i, stdEnum[i], s)
		}
	}

	// capability/v1 status enum is unchanged (no terminal-state additions).
	cap := schemaJSON(t, "artifacts", "capability", "v1", "schema.json")
	capEnum := enumAt(t, cap, "metadata", "properties", "capability", "properties", "status", "enum")
	wantCAP := []string{"draft", "defined", "ready", "in-progress", "verified", "broken", "deprecated"}
	if len(capEnum) != len(wantCAP) {
		t.Fatalf("capability/v1 status enum = %v, want %v", capEnum, wantCAP)
	}
	for i, s := range wantCAP {
		if capEnum[i] != s {
			t.Errorf("capability/v1 status enum[%d] = %q, want %q", i, capEnum[i], s)
		}
	}
}
