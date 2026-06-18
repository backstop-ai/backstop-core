package engine

// FieldContract is an engine's declared field-contract: the Rule field names it
// requires and the ones it forbids (REQ-003). Validation verifies a rule's
// populated fields satisfy its declared engine's contract — verify, not guide
// (REQ-004). The lists are re-keyed faithfully from the retired per-layer
// requirements: semgrep <- layer-2, ast-grep adds (rule-fed, no standard),
// sandbox <- layer-3, config-file <- layer-1. No forbid present in the old
// per-layer checks is dropped.
type FieldContract struct {
	Requires []string
	Forbids  []string
}

// Canonical Rule field names referenced by the field-contracts. These are the
// pack.yml authoring field names (not Go struct field names) so validation
// messages name the field the author wrote.
const (
	FieldRulePath   = "rule_path"
	FieldStandard   = "standard"
	FieldCategory   = "category"
	FieldInputScope = "input_scope"
	FieldValidator  = "validator"
)

// DefaultFieldContracts returns the built-in engine field-contracts keyed by
// engine name. Re-keyed from the live per-layer requirements:
//   - semgrep (ex-layer-2): requires rule_path+standard; forbids
//     category/input_scope/validator.
//   - ast-grep: requires rule_path; forbids category/input_scope/validator
//     (rule-fed like semgrep, minus the standard requirement).
//   - sandbox (ex-layer-3): requires validator/input_scope/category; forbids
//     rule_path. (Value-enum + justification checks are applied in addition to
//     this presence contract by the validator.)
//   - config-file (ex-layer-1 native linter): requires nothing; forbids
//     rule_path/category/input_scope/validator.
func DefaultFieldContracts() map[string]FieldContract {
	return map[string]FieldContract{
		"semgrep": {
			Requires: []string{FieldRulePath, FieldStandard},
			Forbids:  []string{FieldCategory, FieldInputScope, FieldValidator},
		},
		"ast-grep": {
			Requires: []string{FieldRulePath},
			Forbids:  []string{FieldCategory, FieldInputScope, FieldValidator},
		},
		"sandbox": {
			Requires: []string{FieldValidator, FieldInputScope, FieldCategory},
			Forbids:  []string{FieldRulePath},
		},
		"config-file": {
			Requires: nil,
			Forbids:  []string{FieldRulePath, FieldCategory, FieldInputScope, FieldValidator},
		},
	}
}
