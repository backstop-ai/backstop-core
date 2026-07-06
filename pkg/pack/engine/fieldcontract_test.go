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

// The per-engine built-in field-contract assertions that lived here
// (TestDefaultFieldContracts_PerEngineRequiresAndForbids) are DELETED with the baked
// name-keyed map (ISSUE-027). The four generic engines' inline field_contract values
// are now DATA in the embedded base-engines pack and asserted against the parsed pack
// in pkg/baseengines/base_engines_test.go (TestBaseRegistry_InlineFieldContractsPresent).
