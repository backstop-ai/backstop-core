package engine

import (
	"slices"
	"testing"
)

// hasField reports whether name appears in the contract list.
func hasField(list []string, name string) bool {
	return slices.Contains(list, name)
}

// TestFieldContract_DeclaredOnBindingNotNameKeyed asserts an EngineBinding
// carries its OWN FieldContract (Requires/Forbids Rule-field lists) as declared
// data read FROM the binding, not from a map keyed by engine name (REQ-003/
// CLM-036). A pack-declared engine therefore supplies its field-contract inline
// in the engines: block; the validator reads binding.FieldContract directly,
// never a name-keyed lookup. The disposition of the DEFAULT name-keyed contracts
// (CLM-037) is verified in Phase 6 — this pins only that the binding is the
// authoritative carrier of a declared contract.
func TestFieldContract_DeclaredOnBindingNotNameKeyed(t *testing.T) {
	b := EngineBinding{
		Command:   "acme-scan --sarif",
		InputMode: InputModeRuleFlags,
		InputFlag: "--config",
		FieldContract: FieldContract{
			Requires: []string{FieldRulePath, FieldStandard},
			Forbids:  []string{FieldCategory, FieldInputScope, FieldValidator},
		},
	}

	// The contract is read straight off the binding — no engine-name argument is
	// involved, proving it is a declared property of the binding itself.
	if !hasField(b.FieldContract.Requires, FieldRulePath) {
		t.Errorf("binding contract must require %q, got requires=%v", FieldRulePath, b.FieldContract.Requires)
	}
	if !hasField(b.FieldContract.Requires, FieldStandard) {
		t.Errorf("binding contract must require %q, got requires=%v", FieldStandard, b.FieldContract.Requires)
	}
	for _, forb := range []string{FieldCategory, FieldInputScope, FieldValidator} {
		if !hasField(b.FieldContract.Forbids, forb) {
			t.Errorf("binding contract must forbid %q, got forbids=%v", forb, b.FieldContract.Forbids)
		}
	}

	// A binding with no declared contract carries the zero-value FieldContract
	// (empty lists), not a panic or a name-keyed fallback — the binding is the
	// sole source of truth.
	bare := EngineBinding{Command: "noop"}
	if len(bare.FieldContract.Requires) != 0 || len(bare.FieldContract.Forbids) != 0 {
		t.Errorf("a binding with no declared contract must have an empty FieldContract, got %+v", bare.FieldContract)
	}
}

// TestDefaultFieldContracts_PerEngineRequiresAndForbids asserts the built-in
// engine field-contracts encode the re-keyed per-layer presence rules (REQ-003):
// each engine requires exactly the fields its retired layer required and forbids
// the rest. This is the verify-not-guide contract the validator dispatches
// through, so a drift here silently weakens rule validation.
func TestDefaultFieldContracts_PerEngineRequiresAndForbids(t *testing.T) {
	contracts := DefaultFieldContracts()

	// Every built-in findings engine must carry a contract.
	for _, name := range []string{"semgrep", "ast-grep", "sandbox", "config-file"} {
		if _, ok := contracts[name]; !ok {
			t.Fatalf("DefaultFieldContracts must declare a contract for %q", name)
		}
	}

	// semgrep: requires rule_path+standard; forbids category/input_scope/validator.
	semgrep := contracts["semgrep"]
	for _, req := range []string{FieldRulePath, FieldStandard} {
		if !hasField(semgrep.Requires, req) {
			t.Errorf("semgrep contract must require %q, got requires=%v", req, semgrep.Requires)
		}
	}
	for _, forb := range []string{FieldCategory, FieldInputScope, FieldValidator} {
		if !hasField(semgrep.Forbids, forb) {
			t.Errorf("semgrep contract must forbid %q, got forbids=%v", forb, semgrep.Forbids)
		}
	}
	// semgrep must NOT require the sandbox-only validator field.
	if hasField(semgrep.Requires, FieldValidator) {
		t.Errorf("semgrep (rule-fed) must not require %q", FieldValidator)
	}

	// ast-grep: rule-fed like semgrep but WITHOUT the standard requirement.
	astgrep := contracts["ast-grep"]
	if !hasField(astgrep.Requires, FieldRulePath) {
		t.Errorf("ast-grep must require %q, got requires=%v", FieldRulePath, astgrep.Requires)
	}
	if hasField(astgrep.Requires, FieldStandard) {
		t.Errorf("ast-grep must NOT require %q (it drops semgrep's standard requirement)", FieldStandard)
	}

	// sandbox: requires validator/input_scope/category; forbids rule_path.
	sandbox := contracts["sandbox"]
	for _, req := range []string{FieldValidator, FieldInputScope, FieldCategory} {
		if !hasField(sandbox.Requires, req) {
			t.Errorf("sandbox contract must require %q, got requires=%v", req, sandbox.Requires)
		}
	}
	if !hasField(sandbox.Forbids, FieldRulePath) {
		t.Errorf("sandbox contract must forbid %q (it is not rule-fed), got forbids=%v", FieldRulePath, sandbox.Forbids)
	}

	// config-file: requires nothing; forbids rule_path/category/input_scope/validator.
	configFile := contracts["config-file"]
	if len(configFile.Requires) != 0 {
		t.Errorf("config-file contract must require nothing, got requires=%v", configFile.Requires)
	}
	for _, forb := range []string{FieldRulePath, FieldCategory, FieldInputScope, FieldValidator} {
		if !hasField(configFile.Forbids, forb) {
			t.Errorf("config-file contract must forbid %q, got forbids=%v", forb, configFile.Forbids)
		}
	}
}
