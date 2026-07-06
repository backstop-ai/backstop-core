package pack_test

import (
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/pack"
)

// ISSUE-027: the validation path resolves the four generic built-in engines from
// an INJECTED base registry (option (a) — param injection into ValidateManifest),
// NOT the deleted engine.DefaultRegistry() baked table. These tests prove a rule
// that uses a built-in engine WITHOUT the pack declaring it is still checked against
// that engine's inline FieldContract, sourced from the injected base.

// manifestWithSemgrepRule builds a minimal enforcement manifest whose single rule
// binds the built-in "semgrep" engine (which the pack does NOT declare in its own
// engines: block) with the given rule_path — so an empty rule_path trips semgrep's
// inline requires-contract.
func manifestWithSemgrepRule(rulePath string) *pack.Manifest {
	return &pack.Manifest{
		Name:      "test/consumer",
		Version:   "1.0.0",
		Language:  "go",
		Archetype: "enforcement",
		Content: pack.Content{
			Ruleset: pack.Ruleset{
				Rules: []pack.Rule{
					{
						ID:       "sem-rule",
						Engine:   "semgrep",
						Standard: "some-standard",
						RulePath: rulePath,
						PairsWith: pack.PairsWith{
							Scaffolds: []string{"noop"},
						},
					},
				},
			},
		},
	}
}

// TestValidateManifest_SemgrepContractFromInjectedBase proves a rule using the
// undeclared built-in "semgrep" engine is validated against semgrep's inline
// FieldContract resolved from the INJECTED base registry: a missing rule_path
// produces an engine-field ValidationError on the rule_path field (CLM-004).
func TestValidateManifest_SemgrepContractFromInjectedBase(t *testing.T) {
	base := baseTestRegistry()

	// With rule_path present, semgrep's requires-contract is satisfied — no
	// rule_path engine-field violation.
	okErrs := pack.ValidateManifest(manifestWithSemgrepRule("rules/x.yml"), base)
	if fieldContractError(okErrs, "rule_path") != nil {
		t.Errorf("rule_path present should satisfy semgrep contract, got %+v", okErrs)
	}

	// With rule_path missing, the injected base's semgrep contract fires.
	badErrs := pack.ValidateManifest(manifestWithSemgrepRule(""), base)
	ve := fieldContractError(badErrs, "rule_path")
	if ve == nil {
		t.Fatalf("missing rule_path must trip semgrep's inline requires-contract from the injected base; got %+v", badErrs)
	}
	if !strings.Contains(ve.Message, "semgrep") {
		t.Errorf("engine-field error should name the semgrep engine, got %q", ve.Message)
	}
}

// TestValidateManifest_GenericFieldContractClaimCode proves a field-contract
// violation reports the single GENERIC claim code, not a per-field baked code (the
// name-keyed claim map + lookup are deleted) — CLM-006.
func TestValidateManifest_GenericFieldContractClaimCode(t *testing.T) {
	base := baseTestRegistry()

	errs := pack.ValidateManifest(manifestWithSemgrepRule(""), base)
	ve := fieldContractError(errs, "rule_path")
	if ve == nil {
		t.Fatalf("expected a rule_path engine-field violation, got %+v", errs)
	}
	if ve.Rule != "CLM-020-engine-field-contract" {
		t.Errorf("field-contract violation Rule = %q, want the single generic code %q (no per-field baked code)",
			ve.Rule, "CLM-020-engine-field-contract")
	}
	// The retired per-field baked codes must not appear on an engine-field violation.
	for _, baked := range []string{"CLM-007", "CLM-008"} {
		if ve.Rule == baked {
			t.Errorf("field-contract violation Rule = %q; the per-field baked claim codes must be gone", ve.Rule)
		}
	}
}

// fieldContractError returns the first ValidationError whose Field targets the named
// rule field (e.g. "content.ruleset.rules[0].rule_path"), or nil.
func fieldContractError(errs []pack.ValidationError, field string) *pack.ValidationError {
	for i := range errs {
		if strings.HasSuffix(errs[i].Field, "."+field) {
			return &errs[i]
		}
	}
	return nil
}
