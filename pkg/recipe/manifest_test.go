package recipe

import (
	"strings"
	"testing"
)

// wellFormedRecipeYAML is the reference recipe.yml: every field REQ-009 declares
// structurally valid — a param schema, declared transform rules, a paired
// enforcement declaration, target paths, a semver version, and one op from each
// family in the closed allowlist. Target paths are deliberately neutral: the
// manifest reader carries ZERO language knowledge, so nothing here names a
// language, toolchain, or project-manifest file.
const wellFormedRecipeYAML = `
kind: scaffolding
version: 1.2.0
params:
  - name: app_name
    required: true
  - name: config_dir
    required: false
    default: config
transform_rules:
  - rules/rename-key.yml
enforcement:
  rules:
    - recipe.output-present
ops:
  - id: create-config
    kind: create
    target: "{{ config_dir }}/app.settings"
    payload: payload/app.settings
  - id: merge-registry
    kind: merge
    target: "{{ config_dir }}/registry.json"
    format: json
    fragment: payload/registry.fragment.json
  - id: rename-key
    kind: transform
    target: "{{ config_dir }}/app.settings"
    rule: rules/rename-key.yml
    manual: Open the generated settings and rename the legacy_name entry to name by hand.
  - id: register-app
    kind: insert
    target: "{{ config_dir }}/registry.json"
    anchor: '"registrations": ['
    snippet: '"{{ app_name }}"'
    manual: Add the app to the registrations list by hand.
  - id: confirm-adoption
    kind: step
`

// opIDs reduces a parsed op list to its ids in slice order, so a test can assert
// DECLARED ORDER is preserved (never sorted, deduped-by-reorder, or map-ified).
func opIDs(ops []Op) []string {
	ids := make([]string, 0, len(ops))
	for _, op := range ops {
		ids = append(ids, op.ID)
	}
	return ids
}

func TestRecipeManifest_WellFormedValid(t *testing.T) {
	m, err := ParseRecipeManifest([]byte(wellFormedRecipeYAML))
	if err != nil {
		t.Fatalf("well-formed recipe.yml must parse clean, got error: %v", err)
	}

	if m.Kind != KindScaffolding {
		t.Errorf("Kind = %q, want %q", m.Kind, KindScaffolding)
	}
	if m.Version != "1.2.0" {
		t.Errorf("Version = %q, want %q", m.Version, "1.2.0")
	}

	// Param schema round-trips, including the optional default.
	if len(m.Params) != 2 {
		t.Fatalf("len(Params) = %d, want 2", len(m.Params))
	}
	if m.Params[0].Name != "app_name" || !m.Params[0].Required {
		t.Errorf("Params[0] = %+v, want name=app_name required=true", m.Params[0])
	}
	if m.Params[1].Name != "config_dir" || m.Params[1].Required || m.Params[1].Default != "config" {
		t.Errorf("Params[1] = %+v, want name=config_dir required=false default=config", m.Params[1])
	}

	// Declared transform rules and the paired enforcement declaration round-trip.
	if len(m.TransformRules) != 1 || m.TransformRules[0] != "rules/rename-key.yml" {
		t.Errorf("TransformRules = %v, want [rules/rename-key.yml]", m.TransformRules)
	}
	if m.Enforcement == nil {
		t.Fatal("Enforcement declaration must round-trip, got nil")
	}
	if len(m.Enforcement.Rules) != 1 || m.Enforcement.Rules[0] != "recipe.output-present" {
		t.Errorf("Enforcement.Rules = %v, want [recipe.output-present]", m.Enforcement.Rules)
	}

	// Ops keep their DECLARED ORDER — determinism downstream depends on it.
	wantIDs := []string{"create-config", "merge-registry", "rename-key", "register-app", "confirm-adoption"}
	gotIDs := opIDs(m.Ops)
	if len(gotIDs) != len(wantIDs) {
		t.Fatalf("op ids = %v, want %v", gotIDs, wantIDs)
	}
	for i := range wantIDs {
		if gotIDs[i] != wantIDs[i] {
			t.Fatalf("op ids = %v, want %v (declared order must be preserved)", gotIDs, wantIDs)
		}
	}

	// Per-family payload fields round-trip onto the right op.
	create := m.Ops[0]
	if create.Kind != OpCreate || create.Target != "{{ config_dir }}/app.settings" || create.Payload != "payload/app.settings" {
		t.Errorf("create op = %+v, want kind=create with declared target+payload", create)
	}
	// Fragment is a recipe-directory-relative PATH (ISSUE-081's pinned canon), so
	// the assertion is on the declared path itself — the content string it used to
	// carry inline no longer lives in the manifest at all.
	merge := m.Ops[1]
	if merge.Kind != OpMerge || merge.Format != "json" || merge.Fragment != "payload/registry.fragment.json" {
		t.Errorf("merge op = %+v, want kind=merge with declared format + the fragment PATH payload/registry.fragment.json", merge)
	}
	transform := m.Ops[2]
	if transform.Kind != OpTransform || transform.Rule != "rules/rename-key.yml" || transform.Manual == "" {
		t.Errorf("transform op = %+v, want kind=transform with declared rule+manual", transform)
	}
	insert := m.Ops[3]
	if insert.Kind != OpInsert || insert.Anchor != `"registrations": [` || insert.Snippet != `"{{ app_name }}"` || insert.Manual == "" {
		t.Errorf("insert op = %+v, want kind=insert with declared anchor+snippet+manual", insert)
	}
	if m.Ops[4].Kind != OpStep {
		t.Errorf("step op kind = %q, want %q", m.Ops[4].Kind, OpStep)
	}
}

func TestRecipeManifest_MissingVersionErrors(t *testing.T) {
	src := `
kind: scaffolding
ops:
  - id: create-config
    kind: create
    target: config/app.settings
    payload: payload/app.settings
`
	_, err := ParseRecipeManifest([]byte(src))
	if err == nil {
		t.Fatal("a recipe.yml with no version must be a validation error, got nil")
	}
	if !strings.Contains(err.Error(), "version") {
		t.Errorf("error must NAME the offending field 'version', got: %v", err)
	}
}

func TestRecipeManifest_MissingOpsErrors(t *testing.T) {
	src := `
kind: scaffolding
version: 1.0.0
params:
  - name: app_name
    required: true
`
	_, err := ParseRecipeManifest([]byte(src))
	if err == nil {
		t.Fatal("a recipe.yml with no ops must be a validation error, got nil")
	}
	if !strings.Contains(err.Error(), "ops") {
		t.Errorf("error must NAME the offending field 'ops', got: %v", err)
	}
}

func TestRecipeManifest_MalformedVersionErrors(t *testing.T) {
	cases := []struct {
		name    string
		version string
	}{
		{name: "two components", version: "1.2"},
		{name: "v prefix", version: "v1.2.3"},
		{name: "non-numeric", version: "latest"},
		{name: "four components", version: "1.2.3.4"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			src := `
kind: scaffolding
version: "` + tc.version + `"
ops:
  - id: create-config
    kind: create
    target: config/app.settings
    payload: payload/app.settings
`
			_, err := ParseRecipeManifest([]byte(src))
			if err == nil {
				t.Fatalf("version %q is not semver MAJOR.MINOR.PATCH and must error, got nil", tc.version)
			}
			msg := err.Error()
			if !strings.Contains(msg, "version") {
				t.Errorf("error must NAME the offending field 'version', got: %v", err)
			}
			if !strings.Contains(msg, tc.version) {
				t.Errorf("error must quote the offending value %q, got: %v", tc.version, err)
			}
		})
	}
}

// kindOnlyRecipeYAML builds a minimal-but-valid recipe.yml declaring the given
// kind, so the three kind tests differ ONLY in the kind under test.
func kindOnlyRecipeYAML(kind string) string {
	return `
kind: ` + kind + `
version: 1.0.0
ops:
  - id: create-config
    kind: create
    target: config/app.settings
    payload: payload/app.settings
`
}

func TestRecipeManifest_KindScaffoldingValid(t *testing.T) {
	m, err := ParseRecipeManifest([]byte(kindOnlyRecipeYAML(KindScaffolding)))
	if err != nil {
		t.Fatalf("kind %q must parse clean, got error: %v", KindScaffolding, err)
	}
	if m.Kind != KindScaffolding {
		t.Errorf("Kind = %q, want the DECLARED kind %q (a silently coerced kind is a defect)", m.Kind, KindScaffolding)
	}
}

func TestRecipeManifest_KindImplementingValid(t *testing.T) {
	m, err := ParseRecipeManifest([]byte(kindOnlyRecipeYAML(KindImplementing)))
	if err != nil {
		t.Fatalf("kind %q must parse clean, got error: %v", KindImplementing, err)
	}
	if m.Kind != KindImplementing {
		t.Errorf("Kind = %q, want the DECLARED kind %q (a silently coerced kind is a defect)", m.Kind, KindImplementing)
	}
}

func TestRecipeManifest_KindTemplatingValid(t *testing.T) {
	m, err := ParseRecipeManifest([]byte(kindOnlyRecipeYAML(KindTemplating)))
	if err != nil {
		t.Fatalf("kind %q must parse clean, got error: %v", KindTemplating, err)
	}
	if m.Kind != KindTemplating {
		t.Errorf("Kind = %q, want the DECLARED kind %q (a silently coerced kind is a defect)", m.Kind, KindTemplating)
	}
}

func TestRecipeManifest_InvalidKindErrors(t *testing.T) {
	for _, kind := range []string{"generating", "SCAFFOLDING", ""} {
		t.Run("kind_"+kind, func(t *testing.T) {
			src := `
kind: "` + kind + `"
version: 1.0.0
ops:
  - id: create-config
    kind: create
    target: config/app.settings
    payload: payload/app.settings
`
			_, err := ParseRecipeManifest([]byte(src))
			if err == nil {
				t.Fatalf("kind %q is outside the three valid kinds and must error, got nil", kind)
			}
			if !strings.Contains(err.Error(), "kind") {
				t.Errorf("error must NAME the offending field 'kind', got: %v", err)
			}
		})
	}
}

func TestRecipeManifest_OptionalCompatVariantsValidateStructurally(t *testing.T) {
	src := `
kind: implementing
version: 3.0.0
compat:
  - file: dependencies.lock
    path: dependencies.payments.version
    range: ">=18 <19"
variants:
  - compat:
      - file: dependencies.lock
        path: dependencies.payments.version
        range: ">=19 <20"
    ops:
      - id: variant-only-op
        kind: create
        target: config/variant.settings
        payload: payload/variant.settings
ops:
  - id: base-op
    kind: create
    target: config/app.settings
    payload: payload/app.settings
`
	m, err := ParseRecipeManifest([]byte(src))
	if err != nil {
		t.Fatalf("an optional compat matrix + version-keyed variants must validate structurally, got error: %v", err)
	}

	if len(m.Compat) != 1 {
		t.Fatalf("len(Compat) = %d, want 1", len(m.Compat))
	}
	got := m.Compat[0]
	if got.File != "dependencies.lock" || got.Path != "dependencies.payments.version" || got.Range != ">=18 <19" {
		t.Errorf("Compat[0] = %+v, want the declared {file, path, range} selector", got)
	}

	if len(m.Variants) != 1 {
		t.Fatalf("len(Variants) = %d, want 1", len(m.Variants))
	}
	variant := m.Variants[0]
	if len(variant.Compat) != 1 || variant.Compat[0].Range != ">=19 <20" {
		t.Errorf("Variants[0].Compat = %+v, want the declared variant range", variant.Compat)
	}
	if len(variant.Ops) != 1 || variant.Ops[0].ID != "variant-only-op" {
		t.Errorf("Variants[0].Ops = %v, want the variant's own declared ops", opIDs(variant.Ops))
	}

	// NO apply-time resolution: the variant is not selected, resolved, or merged
	// into the recipe's own ops (REQ-015/016 own that, and are out of scope here).
	baseIDs := opIDs(m.Ops)
	if len(baseIDs) != 1 || baseIDs[0] != "base-op" {
		t.Fatalf("Ops = %v, want exactly [base-op] — a resolved/merged variant means apply-time behavior leaked in", baseIDs)
	}
}
