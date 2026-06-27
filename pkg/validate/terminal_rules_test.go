package validate_test

import (
	"testing"

	"github.com/bmanson/backstop-core/pkg/validate"
)

// TestValidate_ReplacedRequiresReplacedBy: a replaced artifact with NO
// replaced-by raises a fail-loud <type>/replaced-by-required violation.
func TestValidate_ReplacedRequiresReplacedBy(t *testing.T) {
	// spec
	specSch := loadTerminalSchema(t, "spec", "v1")
	spec := parseTerminalFixture(t, "spec-replaced-no-replacedby.spec.md")
	res := validate.Spec(spec, specSch)
	if !hasViolationRule(res, "spec/replaced-by-required") {
		t.Errorf("replaced spec without replaced-by: expected spec/replaced-by-required, got %v", res.Violations)
	}

	// plan
	planSch := loadTerminalSchema(t, "plan", "v1")
	plan := parseTerminalFixture(t, "plan-replaced.plan.yml")
	delete(plan.Frontmatter, "replaced-by")
	res = validate.Plan(plan, planSch)
	if !hasViolationRule(res, "plan/replaced-by-required") {
		t.Errorf("replaced plan without replaced-by: expected plan/replaced-by-required, got %v", res.Violations)
	}

	// bundle
	bundleSch := loadTerminalSchema(t, "bundle", "v1")
	bundle := parseTerminalFixture(t, "bundle-delivered.bundle.md")
	bundle.Frontmatter["status"].(map[string]interface{})["maturity"] = "replaced"
	res = validate.Bundle(bundle, bundleSch)
	if !hasViolationRule(res, "bundle/replaced-by-required") {
		t.Errorf("replaced bundle without replaced-by: expected bundle/replaced-by-required, got %v", res.Violations)
	}

	// directive
	dirSch := loadTerminalSchema(t, "directive", "v1")
	dir := parseTerminalFixture(t, "directive-canceled.directive.md")
	dir.Frontmatter["directive"].(map[string]interface{})["status"] = "replaced"
	res = validate.Directive(dir, dirSch)
	if !hasViolationRule(res, "directive/replaced-by-required") {
		t.Errorf("replaced directive without replaced-by: expected directive/replaced-by-required, got %v", res.Violations)
	}

	// issue
	issueSch := loadTerminalSchema(t, "issue", "v1")
	issue := parseTerminalFixture(t, "issue-replaced.issue.md")
	delete(issue.Frontmatter, "replaced-by")
	res = validate.Issue(issue, issueSch)
	if !hasViolationRule(res, "issue/replaced-by-required") {
		t.Errorf("replaced issue without replaced-by: expected issue/replaced-by-required, got %v", res.Violations)
	}
}

// TestValidate_ReplacedByMustBeTypedRef: a malformed replaced-by (not matching
// the typed-ref pattern) raises <type>/replaced-by-malformed; a well-formed
// single ref validates clean (no malformed/required violation).
func TestValidate_ReplacedByMustBeTypedRef(t *testing.T) {
	specSch := loadTerminalSchema(t, "spec", "v1")

	// malformed
	bad := parseTerminalFixture(t, "spec-replaced-malformed.spec.md")
	res := validate.Spec(bad, specSch)
	if !hasViolationRule(res, "spec/replaced-by-malformed") {
		t.Errorf("malformed replaced-by: expected spec/replaced-by-malformed, got %v", res.Violations)
	}

	// well-formed single ref
	good := parseTerminalFixture(t, "spec-replaced-malformed.spec.md")
	good.Frontmatter["replaced-by"] = "BUNDLE-011"
	res = validate.Spec(good, specSch)
	if hasViolationRule(res, "spec/replaced-by-malformed") {
		t.Errorf("well-formed replaced-by raised malformed: %v", res.Violations)
	}
	if hasViolationRule(res, "spec/replaced-by-required") {
		t.Errorf("well-formed replaced-by raised required: %v", res.Violations)
	}
}

// TestValidate_ReplacedByAcceptsArray: an array of well-formed refs validates
// clean (multi-absorber); an array containing a malformed entry raises malformed.
func TestValidate_ReplacedByAcceptsArray(t *testing.T) {
	specSch := loadTerminalSchema(t, "spec", "v1")

	// array of well-formed refs
	good := parseTerminalFixture(t, "spec-replaced-malformed.spec.md")
	good.Frontmatter["replaced-by"] = []interface{}{"BUNDLE-011", "SPEC-902"}
	res := validate.Spec(good, specSch)
	if hasViolationRule(res, "spec/replaced-by-malformed") {
		t.Errorf("array of well-formed refs raised malformed: %v", res.Violations)
	}
	if hasViolationRule(res, "spec/replaced-by-required") {
		t.Errorf("array of well-formed refs raised required: %v", res.Violations)
	}

	// array with one malformed entry
	mixed := parseTerminalFixture(t, "spec-replaced-malformed.spec.md")
	mixed.Frontmatter["replaced-by"] = []interface{}{"BUNDLE-011", "oops"}
	res = validate.Spec(mixed, specSch)
	if !hasViolationRule(res, "spec/replaced-by-malformed") {
		t.Errorf("array with malformed entry: expected spec/replaced-by-malformed, got %v", res.Violations)
	}
}

// TestValidate_CanceledDeprecatedReasonOptional: a canceled/deprecated artifact
// validates clean whether or not a free-text reason is present.
func TestValidate_CanceledDeprecatedReasonOptional(t *testing.T) {
	specSch := loadTerminalSchema(t, "spec", "v1")

	// canceled WITHOUT reason — no reason-required violation
	noReason := parseTerminalFixture(t, "spec-deprecated.spec.md")
	noReason.Metadata["status"] = "canceled"
	noReason.Frontmatter["status"] = "canceled"
	res := validate.Spec(noReason, specSch)
	if hasViolationRule(res, "spec/reason-required") {
		t.Errorf("canceled spec without reason raised spec/reason-required: %v", res.Violations)
	}

	// canceled WITH reason — also clean
	withReason := parseTerminalFixture(t, "spec-deprecated.spec.md")
	withReason.Metadata["status"] = "canceled"
	withReason.Frontmatter["status"] = "canceled"
	withReason.Frontmatter["reason"] = "abandoned in favor of packs-only"
	res = validate.Spec(withReason, specSch)
	if hasViolationRule(res, "spec/reason-required") {
		t.Errorf("canceled spec with reason raised spec/reason-required: %v", res.Violations)
	}

	// deprecated WITHOUT reason — clean (spec)
	dep := parseTerminalFixture(t, "spec-deprecated.spec.md")
	res = validate.Spec(dep, specSch)
	if hasViolationRule(res, "spec/reason-required") {
		t.Errorf("deprecated spec without reason raised spec/reason-required: %v", res.Violations)
	}

	// deprecated WITHOUT reason — clean (bundle)
	bundleSch := loadTerminalSchema(t, "bundle", "v1")
	depBundle := parseTerminalFixture(t, "bundle-deprecated-mingate.bundle.md")
	res = validate.Bundle(depBundle, bundleSch)
	if hasViolationRule(res, "bundle/reason-required") {
		t.Errorf("deprecated bundle without reason raised bundle/reason-required: %v", res.Violations)
	}
}

// TestValidateSpec_TerminalExemptFromClaimCompleteness: a replaced spec with a
// valid replaced-by but no claims/requirements/verification does NOT raise the
// live-work completeness violations a draft/ready spec would.
func TestValidateSpec_TerminalExemptFromClaimCompleteness(t *testing.T) {
	specSch := loadTerminalSchema(t, "spec", "v1")
	spec := parseTerminalFixture(t, "spec-replaced-no-replacedby.spec.md")
	spec.Frontmatter["replaced-by"] = "SPEC-902"

	res := validate.Spec(spec, specSch)

	liveWorkRules := []string{
		"spec/claims-required", "spec/claims-empty", "spec/claim-tests-empty",
		"spec/requirements-required", "spec/requirements-empty",
		"spec/verification-required", "spec/implementation-required",
	}
	for _, rule := range liveWorkRules {
		if hasViolationRule(res, rule) {
			t.Errorf("terminal spec must be exempt from live-work rule %q, got: %v", rule, res.Violations)
		}
	}
	// The structural replaced-by check must still be satisfied (clean here).
	if hasViolationRule(res, "spec/replaced-by-required") || hasViolationRule(res, "spec/replaced-by-malformed") {
		t.Errorf("terminal spec with valid replaced-by should pass structural check: %v", res.Violations)
	}
}

// TestValidateBundle_TerminalExemptFromMaturityGates: a deprecated bundle is not
// held to the defined/ready required-section + requirements[] gates.
func TestValidateBundle_TerminalExemptFromMaturityGates(t *testing.T) {
	bundleSch := loadTerminalSchema(t, "bundle", "v1")
	bundle := parseTerminalFixture(t, "bundle-deprecated-mingate.bundle.md")

	res := validate.Bundle(bundle, bundleSch)

	gateRules := []string{
		"bundle/maturity-gate", "bundle/maturity-section", "bundle/requirements-required",
	}
	for _, rule := range gateRules {
		if hasViolationRule(res, rule) {
			t.Errorf("terminal bundle must be exempt from maturity-gate rule %q, got: %v", rule, res.Violations)
		}
	}
}

// TestValidateIssue_TerminalExemptFromTraceability: a replaced issue is not held
// to the REQ→CLM→tests traceability requirement.
func TestValidateIssue_TerminalExemptFromTraceability(t *testing.T) {
	issueSch := loadTerminalSchema(t, "issue", "v1")
	issue := parseTerminalFixture(t, "issue-replaced-no-traceability.issue.md")

	res := validate.Issue(issue, issueSch)

	traceabilityRules := []string{
		"issue/requirements-required", "issue/claims-required",
		"issue/verification-required", "issue/implementation-required",
	}
	for _, rule := range traceabilityRules {
		if hasViolationRule(res, rule) {
			t.Errorf("terminal issue must be exempt from traceability rule %q, got: %v", rule, res.Violations)
		}
	}
}
