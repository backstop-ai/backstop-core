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

// The name-keyed baked built-in field-contract map that once lived here is DELETED
// (ISSUE-027). Each engine's FieldContract now travels INLINE on its binding: the
// four generic engines carry it in the embedded base-engines pack
// (packs/base-engines/pack.yml, via engines:.<name>.field_contract), and
// pack-declared engines carry it in their own engines: block. The validator reads
// binding.FieldContract directly — there is no baked name-keyed fallback map.
