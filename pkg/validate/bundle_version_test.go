package validate_test

import (
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/artifact"
	"github.com/backstop-ai/backstop-core/pkg/validate"
)

// bundleReqFixture builds an otherwise-valid `ready` bundle carrying the given
// requirements[] array, so REQ-004 version-log tests can vary only the
// requirement shape and read the version violation in isolation.
func bundleReqFixture(reqs []interface{}) *artifact.ParsedArtifact {
	return &artifact.ParsedArtifact{
		Filename: "my-feature.bundle.md",
		Title:    "My Feature",
		Metadata: map[string]string{"schema_version": "bundle/v1"},
		Frontmatter: map[string]interface{}{
			"schema_version": "bundle/v1",
			"bundle": map[string]interface{}{
				"name":     "my-feature",
				"version":  "1.0.0",
				"created":  "2026-03-01",
				"updated":  "2026-03-19",
				"category": "feature",
			},
			"status": map[string]interface{}{"maturity": "ready"},
			"problem": map[string]interface{}{
				"summary":          "A real problem summary",
				"user_story":       "As a user, I want X",
				"success_criteria": []interface{}{"criterion-1"},
			},
			"solution": map[string]interface{}{
				"approach":    "Build it this way",
				"assumptions": []interface{}{"assumption-1"},
			},
			"requirements": reqs,
		},
		Sections: []string{
			"Current Thinking", "Draft Requirements",
			"Draft Design Decisions", "Spec Seeds", "Version History",
		},
	}
}

// bundleAtMaturity builds a bundle at the given maturity for the REQ-005
// maturity-matrix tests, optionally without a requirements[] array.
func bundleAtMaturity(maturity string, withReqs bool) *artifact.ParsedArtifact {
	art := &artifact.ParsedArtifact{
		Filename: "my-feature.bundle.md",
		Title:    "My Feature",
		Metadata: map[string]string{"schema_version": "bundle/v1"},
		Frontmatter: map[string]interface{}{
			"schema_version": "bundle/v1",
			"bundle": map[string]interface{}{
				"name":     "my-feature",
				"version":  "1.0.0",
				"created":  "2026-03-01",
				"updated":  "2026-03-19",
				"category": "feature",
			},
			"status": map[string]interface{}{"maturity": maturity},
		},
		Sections: []string{},
	}
	if withReqs {
		art.Frontmatter["requirements"] = []interface{}{
			map[string]interface{}{"id": "REQ-001", "text": "System must support X", "version": "1.0.0"},
		}
	}
	return art
}

// --- REQ-004: per-REQ version + version-log well-formedness ---

func TestBundleReqVersion_ImplicitSingleEntryValid(t *testing.T) {
	// A REQ with a well-formed version: and NO explicit versions: list — the
	// implicit single-entry log — is valid (backward-compat hinge, CLM-015).
	art := bundleReqFixture([]interface{}{
		map[string]interface{}{"id": "REQ-001", "text": "System must support X", "version": "1.0.0"},
	})
	result := validate.Bundle(art, bundleSchema())
	assertNoViolationRule(t, result, "bundle/requirement-version-required")
	assertNoViolationRule(t, result, "bundle/requirement-version-format")
	assertNoViolationRule(t, result, "bundle/requirement-versions-empty")
	if !result.Pass() {
		t.Errorf("expected implicit single-entry log to be valid, got %d violations:", len(result.Violations))
		for _, v := range result.Violations {
			t.Errorf("  [%s] %s", v.Rule, v.Message)
		}
	}
}

func TestBundleReqVersion_MissingVersionErrors(t *testing.T) {
	art := bundleReqFixture([]interface{}{
		map[string]interface{}{"id": "REQ-001", "text": "System must support X"},
	})
	result := validate.Bundle(art, bundleSchema())
	assertHasViolation(t, result, "bundle/requirement-version-required")
}

func TestBundleReqVersion_MalformedVersionErrors(t *testing.T) {
	for _, bad := range []string{"1.0", "v1.0.0", "1.0.0-rc1"} {
		art := bundleReqFixture([]interface{}{
			map[string]interface{}{"id": "REQ-001", "text": "System must support X", "version": bad},
		})
		result := validate.Bundle(art, bundleSchema())
		assertHasViolation(t, result, "bundle/requirement-version-format")
	}
}

func TestBundleReqVersionLog_WellFormedValid(t *testing.T) {
	// Explicit ascending, unique versions:, top-level version/text == newest.
	art := bundleReqFixture([]interface{}{
		map[string]interface{}{
			"id":      "REQ-001",
			"text":    "Second understanding",
			"version": "2.0.0",
			"versions": []interface{}{
				map[string]interface{}{"version": "1.0.0", "text": "First understanding"},
				map[string]interface{}{"version": "2.0.0", "text": "Second understanding"},
			},
		},
	})
	result := validate.Bundle(art, bundleSchema())
	if !result.Pass() {
		t.Errorf("expected well-formed version log to be valid, got %d violations:", len(result.Violations))
		for _, v := range result.Violations {
			t.Errorf("  [%s] %s", v.Rule, v.Message)
		}
	}
}

func TestBundleReqVersionLog_NonSemverEntryErrors(t *testing.T) {
	art := bundleReqFixture([]interface{}{
		map[string]interface{}{
			"id":      "REQ-001",
			"text":    "text",
			"version": "2.0.0",
			"versions": []interface{}{
				map[string]interface{}{"version": "1.0", "text": "First"},
				map[string]interface{}{"version": "2.0.0", "text": "text"},
			},
		},
	})
	result := validate.Bundle(art, bundleSchema())
	assertHasViolation(t, result, "bundle/requirement-versions-entry-format")
}

func TestBundleReqVersionLog_NonMonotonicErrors(t *testing.T) {
	// Genuinely descending list is an error.
	descending := bundleReqFixture([]interface{}{
		map[string]interface{}{
			"id":      "REQ-001",
			"text":    "First",
			"version": "1.0.0",
			"versions": []interface{}{
				map[string]interface{}{"version": "2.0.0", "text": "Second"},
				map[string]interface{}{"version": "1.0.0", "text": "First"},
			},
		},
	})
	assertHasViolation(t, validate.Bundle(descending, bundleSchema()), "bundle/requirement-versions-nonmonotonic")

	// Numeric-semver ascending [1.9.0, 1.10.0] must be accepted (string order
	// would wrongly flag it). Top-level version/text == newest (1.10.0).
	numericAscending := bundleReqFixture([]interface{}{
		map[string]interface{}{
			"id":      "REQ-001",
			"text":    "Tenth",
			"version": "1.10.0",
			"versions": []interface{}{
				map[string]interface{}{"version": "1.9.0", "text": "Ninth"},
				map[string]interface{}{"version": "1.10.0", "text": "Tenth"},
			},
		},
	})
	assertNoViolationRule(t, validate.Bundle(numericAscending, bundleSchema()), "bundle/requirement-versions-nonmonotonic")
}

func TestBundleReqVersionLog_DuplicateVersionErrors(t *testing.T) {
	art := bundleReqFixture([]interface{}{
		map[string]interface{}{
			"id":      "REQ-001",
			"text":    "text",
			"version": "1.0.0",
			"versions": []interface{}{
				map[string]interface{}{"version": "1.0.0", "text": "First"},
				map[string]interface{}{"version": "1.0.0", "text": "text"},
			},
		},
	})
	result := validate.Bundle(art, bundleSchema())
	assertHasViolation(t, result, "bundle/requirement-versions-duplicate")
}

func TestBundleReqVersionLog_CurrentNotNewestErrors(t *testing.T) {
	// Top-level version: (1.0.0) is not the newest (2.0.0) log entry.
	art := bundleReqFixture([]interface{}{
		map[string]interface{}{
			"id":      "REQ-001",
			"text":    "Second understanding",
			"version": "1.0.0",
			"versions": []interface{}{
				map[string]interface{}{"version": "1.0.0", "text": "First understanding"},
				map[string]interface{}{"version": "2.0.0", "text": "Second understanding"},
			},
		},
	})
	result := validate.Bundle(art, bundleSchema())
	assertHasViolation(t, result, "bundle/requirement-version-not-newest")
}

func TestBundleReqVersionLog_EmptyEntryTextErrors(t *testing.T) {
	art := bundleReqFixture([]interface{}{
		map[string]interface{}{
			"id":      "REQ-001",
			"text":    "text",
			"version": "2.0.0",
			"versions": []interface{}{
				map[string]interface{}{"version": "1.0.0", "text": ""},
				map[string]interface{}{"version": "2.0.0", "text": "text"},
			},
		},
	})
	result := validate.Bundle(art, bundleSchema())
	assertHasViolation(t, result, "bundle/requirement-versions-entry-text")
}

func TestBundleReqVersionLog_EmptyVersionsListErrors(t *testing.T) {
	// An EXPLICIT but empty versions: [] is an error (asymmetry with absent).
	art := bundleReqFixture([]interface{}{
		map[string]interface{}{
			"id":       "REQ-001",
			"text":     "text",
			"version":  "1.0.0",
			"versions": []interface{}{},
		},
	})
	result := validate.Bundle(art, bundleSchema())
	assertHasViolation(t, result, "bundle/requirement-versions-empty")
}

func TestBundleReqVersionLog_CurrentTextNotNewestErrors(t *testing.T) {
	// Top-level text: differs from the newest entry's text.
	art := bundleReqFixture([]interface{}{
		map[string]interface{}{
			"id":      "REQ-001",
			"text":    "Stale top-level text",
			"version": "2.0.0",
			"versions": []interface{}{
				map[string]interface{}{"version": "1.0.0", "text": "First understanding"},
				map[string]interface{}{"version": "2.0.0", "text": "Second understanding"},
			},
		},
	})
	result := validate.Bundle(art, bundleSchema())
	assertHasViolation(t, result, "bundle/requirement-text-not-newest")
}

func TestBundleReqVersionLog_TextNotNewestWhitespaceNormalized(t *testing.T) {
	// ParseFile clips the trailing newline of a folded `>` scalar when it is the
	// last line before the closing `---`, while mid-document scalars keep theirs —
	// so byte-equal-authored texts can differ by trailing whitespace depending on
	// the requirement's position in frontmatter. The text-not-newest check
	// normalizes surrounding whitespace, so a trailing-whitespace-only difference
	// is NOT a violation (CLM-033 stays meaning-equality, not byte-equality).
	art := bundleReqFixture([]interface{}{
		map[string]interface{}{
			"id":      "REQ-001",
			"text":    "Second understanding\n", // parser artifact: trailing newline
			"version": "2.0.0",
			"versions": []interface{}{
				map[string]interface{}{"version": "1.0.0", "text": "First understanding"},
				map[string]interface{}{"version": "2.0.0", "text": "Second understanding"},
			},
		},
	})
	result := validate.Bundle(art, bundleSchema())
	assertNoViolationRule(t, result, "bundle/requirement-text-not-newest")
}

// --- REQ-005: requirements[] required at delivered; terminal stays exempt ---

func TestBundleRequirements_DeliveredMissingErrors(t *testing.T) {
	art := bundleAtMaturity("delivered", false)
	result := validate.Bundle(art, bundleSchema())
	assertHasViolation(t, result, "bundle/requirements-required")
}

func TestBundleRequirements_DeliveredPresentValid(t *testing.T) {
	art := bundleAtMaturity("delivered", true)
	result := validate.Bundle(art, bundleSchema())
	assertNoViolationRule(t, result, "bundle/requirements-required")
}

func TestBundleRequirements_DefinedMissingStillErrors(t *testing.T) {
	art := bundleAtMaturity("defined", false)
	result := validate.Bundle(art, bundleSchema())
	assertHasViolation(t, result, "bundle/requirements-required")
}

func TestBundleRequirements_ReadyMissingStillErrors(t *testing.T) {
	art := bundleAtMaturity("ready", false)
	result := validate.Bundle(art, bundleSchema())
	assertHasViolation(t, result, "bundle/requirements-required")
}

func TestBundleRequirements_ReplacedStaysExempt(t *testing.T) {
	art := bundleAtMaturity("replaced", false)
	result := validate.Bundle(art, bundleSchema())
	assertNoViolationRule(t, result, "bundle/requirements-required")
}

func TestBundleRequirements_CanceledStaysExempt(t *testing.T) {
	art := bundleAtMaturity("canceled", false)
	result := validate.Bundle(art, bundleSchema())
	assertNoViolationRule(t, result, "bundle/requirements-required")
}

func TestBundleRequirements_DeprecatedStaysExempt(t *testing.T) {
	art := bundleAtMaturity("deprecated", false)
	result := validate.Bundle(art, bundleSchema())
	assertNoViolationRule(t, result, "bundle/requirements-required")
}
