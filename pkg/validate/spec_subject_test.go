package validate_test

import (
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/validate"
)

// spec_subject_test.go drives the non-breaking neutralization of the
// implementation target key (ISSUE-047 CLM-001/CLM-002): `package` → the
// language-neutral `subject`, with the legacy `package` key accepted as a
// DEPRECATED ALIAS so the ~40 existing specs keep validating. The enforcement
// locus is the hand-rolled validateImplementation (spec.go), exercised here via
// the real validate.Spec entry point; schema.json required_keys are documentary
// only (pkg/schema/load.go does not parse them).

// hasImplementationViolation reports whether any violation names an
// implementation-* rule under the given prefix ("spec"). Used to assert a block
// validated with NO implementation target violation.
func hasImplementationViolation(vs []validate.Violation) (validate.Violation, bool) {
	for _, v := range vs {
		if strings.HasPrefix(v.Rule, "spec/implementation-") {
			return v, true
		}
	}
	return validate.Violation{}, false
}

// TestValidateImplementation_SubjectAccepted (CLM-001) — GENUINELY RED today: an
// implementation block declaring the neutral `subject` (no `package`) currently
// raises implementation-package-required; after the subject-or-package guard it
// validates with NO implementation-* violation.
func TestValidateImplementation_SubjectAccepted(t *testing.T) {
	art := validSpecArtifact()
	art.Frontmatter["implementation"] = map[string]interface{}{
		"summary": "Neutralized target key",
		"subject": "pkg/gate",
	}

	result := validate.Spec(art, specSchema())
	if v, found := hasImplementationViolation(result.Violations); found {
		t.Errorf("subject-only implementation block must validate with no implementation-* violation, got: [%s] %s", v.Rule, v.Message)
	}
}

// TestValidateImplementation_MissingSubjectAndPackageRejected (CLM-001) —
// GENUINELY RED today: a block with `summary` but NEITHER `subject` NOR
// `package` must raise implementation-subject-required. The guard is non-vacuous:
// at least one of the two target keys must be present and non-empty.
func TestValidateImplementation_MissingSubjectAndPackageRejected(t *testing.T) {
	art := validSpecArtifact()
	art.Frontmatter["implementation"] = map[string]interface{}{
		"summary": "No target declared",
	}

	result := validate.Spec(art, specSchema())
	found := false
	for _, v := range result.Violations {
		if v.Rule == "spec/implementation-subject-required" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'spec/implementation-subject-required' when both subject and package are absent, got: %v", result.Violations)
	}
}

// TestValidateImplementation_LegacyPackageAliasAccepted (CLM-001) — REGRESSION
// GUARD (passes today AND after): a block declaring ONLY the legacy
// `package` key still validates with no implementation-* violation, pinning the
// deprecated alias so every unmigrated spec stays green (non-breaking).
func TestValidateImplementation_LegacyPackageAliasAccepted(t *testing.T) {
	art := validSpecArtifact()
	art.Frontmatter["implementation"] = map[string]interface{}{
		"summary": "Legacy target key",
		"package": "pkg/check",
	}

	result := validate.Spec(art, specSchema())
	if v, found := hasImplementationViolation(result.Violations); found {
		t.Errorf("legacy package-only implementation block must still validate (non-breaking alias), got: [%s] %s", v.Rule, v.Message)
	}
}

// TestValidateClaims_PerClaimSubjectOptionalAccepted (CLM-002) — REGRESSION
// GUARD (passes today AND after): a claim carrying an optional per-claim
// `subject` validates, and a claim WITHOUT a subject validates, pinning that
// subject is OPTIONAL on claims and never required.
func TestValidateClaims_PerClaimSubjectOptionalAccepted(t *testing.T) {
	// A claim carrying an optional per-claim subject must not raise a claim-* violation.
	withSubject := validSpecArtifact()
	withSubject.Frontmatter["claims"] = []interface{}{
		map[string]interface{}{
			"id":          "CLM-001",
			"requirement": "REQ-001",
			"text":        "ParseFile extracts H1 title",
			"subject":     "pkg/gate",
			"tests": []interface{}{
				map[string]interface{}{"test_name": "TestParseFile_ValidADR"},
			},
		},
		map[string]interface{}{
			"id":          "CLM-002",
			"requirement": "REQ-002",
			"text":        "Parse extracts metadata from YAML frontmatter",
			"tests": []interface{}{
				map[string]interface{}{"test_name": "TestParse_ExtractsMetadata"},
			},
		},
	}
	result := validate.Spec(withSubject, specSchema())
	for _, v := range result.Violations {
		if strings.HasPrefix(v.Rule, "spec/claim") {
			t.Errorf("a claim with an optional per-claim subject must validate, got claim violation: [%s] %s", v.Rule, v.Message)
		}
	}

	// The baseline valid spec (claims WITHOUT any subject) still validates — subject
	// is never required on a claim.
	plain := validate.Spec(validSpecArtifact(), specSchema())
	for _, v := range plain.Violations {
		if strings.HasPrefix(v.Rule, "spec/claim") {
			t.Errorf("a claim WITHOUT a subject must validate (subject is optional), got claim violation: [%s] %s", v.Rule, v.Message)
		}
	}
}
