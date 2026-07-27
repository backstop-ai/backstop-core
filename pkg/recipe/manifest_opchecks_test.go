package recipe

import (
	"strings"
	"testing"
)

// The op-level cross-checks ParseRecipeManifest runs at parse time are each
// driven as a MATCHED PAIR — the negative errors AND the positive validates
// clean — so a validator that simply rejects everything cannot pass this file.
//
// WHY they matter: op `id` is the key ApplyOptions.InjectionSites routes the
// SDLC-mediated WHERE by (REQ-003). A duplicate id misroutes an injection and an
// empty id makes routing ambiguous, so both are parse-time errors. `manual:` is
// the human-actionable fallback emitted VERBATIM when the injection limit is hit
// (REQ-011) — core cannot synthesize it, so the injection-limit op families must
// declare it.

func TestRecipeManifest_TransformInsertMissingManualErrors(t *testing.T) {
	cases := []struct {
		name string
		opID string
		src  string
	}{
		{
			name: "transform op with no manual",
			opID: "rename-key",
			src: `
kind: scaffolding
version: 1.0.0
transform_rules:
  - rules/rename-key.yml
ops:
  - id: rename-key
    kind: transform
    target: config/app.settings
    rule: rules/rename-key.yml
`,
		},
		{
			name: "transform op with empty manual",
			opID: "rename-key",
			src: `
kind: scaffolding
version: 1.0.0
transform_rules:
  - rules/rename-key.yml
ops:
  - id: rename-key
    kind: transform
    target: config/app.settings
    rule: rules/rename-key.yml
    manual: "   "
`,
		},
		{
			name: "insert op with no manual",
			opID: "register-app",
			src: `
kind: scaffolding
version: 1.0.0
ops:
  - id: register-app
    kind: insert
    target: config/registry.json
    anchor: '"registrations": ['
    snippet: '"app"'
`,
		},
		{
			name: "insert op with empty manual",
			opID: "register-app",
			src: `
kind: scaffolding
version: 1.0.0
ops:
  - id: register-app
    kind: insert
    target: config/registry.json
    anchor: '"registrations": ['
    snippet: '"app"'
    manual: ""
`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseRecipeManifest([]byte(tc.src))
			if err == nil {
				t.Fatal("an injection-limit op with no non-empty manual must be a validation error, got nil")
			}
			msg := err.Error()
			if !strings.Contains(msg, tc.opID) {
				t.Errorf("error must NAME the offending op id %q, got: %v", tc.opID, err)
			}
			if !strings.Contains(msg, "manual") {
				t.Errorf("error must NAME the missing field 'manual', got: %v", err)
			}
		})
	}
}

func TestRecipeManifest_CreateMergeManualOptional(t *testing.T) {
	src := `
kind: scaffolding
version: 1.0.0
ops:
  - id: create-config
    kind: create
    target: config/app.settings
    payload: payload/app.settings
  - id: merge-registry
    kind: merge
    target: config/registry.json
    format: json
    fragment: payload/registry.fragment.json
`
	m, err := ParseRecipeManifest([]byte(src))
	if err != nil {
		t.Fatalf("create/merge ops carry no manual and must parse clean — manual is required only on the injection-limit families; got error: %v", err)
	}
	if len(m.Ops) != 2 {
		t.Fatalf("len(Ops) = %d, want 2", len(m.Ops))
	}
	for _, op := range m.Ops {
		if op.Manual != "" {
			t.Errorf("op %q Manual = %q, want empty (the fixture declares none)", op.ID, op.Manual)
		}
	}
}

func TestRecipeManifest_TransformOpUndeclaredRuleErrors(t *testing.T) {
	src := `
kind: scaffolding
version: 1.0.0
transform_rules:
  - rules/rename-key.yml
ops:
  - id: rewrite-entry
    kind: transform
    target: config/app.settings
    rule: rules/undeclared-rewrite.yml
    manual: Apply the rewrite by hand.
`
	_, err := ParseRecipeManifest([]byte(src))
	if err == nil {
		t.Fatal("a transform op citing a rule absent from the declared transform rules must be a validation error, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "rewrite-entry") {
		t.Errorf("error must NAME the offending op id 'rewrite-entry', got: %v", err)
	}
	if !strings.Contains(msg, "rules/undeclared-rewrite.yml") {
		t.Errorf("error must NAME the undeclared rule 'rules/undeclared-rewrite.yml', got: %v", err)
	}
}

func TestRecipeManifest_TransformOpDeclaredRuleValid(t *testing.T) {
	src := `
kind: scaffolding
version: 1.0.0
transform_rules:
  - rules/rename-key.yml
  - rules/undeclared-rewrite.yml
ops:
  - id: rewrite-entry
    kind: transform
    target: config/app.settings
    rule: rules/undeclared-rewrite.yml
    manual: Apply the rewrite by hand.
`
	m, err := ParseRecipeManifest([]byte(src))
	if err != nil {
		t.Fatalf("the SAME op whose rule IS among the declared transform rules must parse clean, got error: %v", err)
	}
	if len(m.Ops) != 1 || m.Ops[0].Rule != "rules/undeclared-rewrite.yml" {
		t.Errorf("Ops = %+v, want the single transform op with its declared rule", m.Ops)
	}
}

func TestRecipeManifest_DuplicateOrEmptyOpIdErrors(t *testing.T) {
	t.Run("two ops sharing an id", func(t *testing.T) {
		src := `
kind: scaffolding
version: 1.0.0
ops:
  - id: create-config
    kind: create
    target: config/app.settings
    payload: payload/app.settings
  - id: create-config
    kind: merge
    target: config/registry.json
    format: json
    fragment: payload/registry.fragment.json
`
		_, err := ParseRecipeManifest([]byte(src))
		if err == nil {
			t.Fatal("two ops sharing an id must be a validation error, got nil")
		}
		msg := err.Error()
		if !strings.Contains(msg, "recipe") || !strings.Contains(msg, "1.0.0") {
			t.Errorf("error must NAME the recipe (its declared kind+version), got: %v", err)
		}
		if !strings.Contains(msg, "create-config") {
			t.Errorf("error must NAME the duplicate id 'create-config', got: %v", err)
		}
	})

	t.Run("injection-accepting op with an empty id", func(t *testing.T) {
		src := `
kind: scaffolding
version: 1.0.0
transform_rules:
  - rules/rename-key.yml
ops:
  - id: create-config
    kind: create
    target: config/app.settings
    payload: payload/app.settings
  - id: ""
    kind: transform
    target: config/app.settings
    rule: rules/rename-key.yml
    manual: Apply the rewrite by hand.
`
		_, err := ParseRecipeManifest([]byte(src))
		if err == nil {
			t.Fatal("a transform op with an empty id must be a validation error (the InjectionSites routing key), got nil")
		}
		msg := err.Error()
		if !strings.Contains(msg, "recipe") || !strings.Contains(msg, "1.0.0") {
			t.Errorf("error must NAME the recipe (its declared kind+version), got: %v", err)
		}
		if !strings.Contains(msg, "ops[1]") {
			t.Errorf("error must LOCATE the offending op (ops[1]), got: %v", err)
		}
	})
}

func TestRecipeManifest_UniqueOpIdsValid(t *testing.T) {
	src := `
kind: scaffolding
version: 1.0.0
transform_rules:
  - rules/rename-key.yml
ops:
  - id: create-config
    kind: create
    target: config/app.settings
    payload: payload/app.settings
  - id: rename-key
    kind: transform
    target: config/app.settings
    rule: rules/rename-key.yml
    manual: Rename the entry by hand.
  - id: register-app
    kind: insert
    target: config/registry.json
    anchor: '"registrations": ['
    snippet: '"app"'
    manual: Add the app to the registrations list by hand.
`
	m, err := ParseRecipeManifest([]byte(src))
	if err != nil {
		t.Fatalf("unique, non-empty op ids on every injection-accepting op must parse clean, got error: %v", err)
	}
	wantIDs := []string{"create-config", "rename-key", "register-app"}
	gotIDs := opIDs(m.Ops)
	if len(gotIDs) != len(wantIDs) {
		t.Fatalf("op ids = %v, want %v", gotIDs, wantIDs)
	}
	for i := range wantIDs {
		if gotIDs[i] != wantIDs[i] {
			t.Fatalf("op ids = %v, want %v", gotIDs, wantIDs)
		}
	}
}
