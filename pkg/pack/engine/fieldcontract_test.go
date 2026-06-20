package engine

import (
	"slices"
	"testing"
)

// hasField reports whether name appears in the contract list.
func hasField(list []string, name string) bool {
	return slices.Contains(list, name)
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
