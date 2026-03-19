package validate_test

import (
	"testing"

	"github.com/bmanson/backstop-core/pkg/artifact"
	"github.com/bmanson/backstop-core/pkg/schema"
	"github.com/bmanson/backstop-core/pkg/validate"
)

func issueSchema() *schema.Schema {
	return &schema.Schema{
		ArtifactType:     "issue",
		FilenamePattern:  `^ISSUE-[0-9]{3}-[a-z][a-z0-9]*(-[a-z0-9]+)*\.md$`,
		SlugPattern:      `^[a-z][a-z0-9]*(-[a-z0-9]+)*$`,
		SlugMinLength:    2,
		SlugMaxLength:    64,
		RequiredMetadata: []string{"title", "schema_version"},
		RequiredSections: []string{"Problem"},
	}
}

func validIssueArtifact() *artifact.ParsedArtifact {
	return &artifact.ParsedArtifact{
		Filename: "ISSUE-042-fix-parser-crash.md",
		Title:    "ISSUE-042: Fix Parser Crash",
		Metadata: map[string]string{
			"schema_version": "issue/v1",
		},
		Frontmatter: map[string]interface{}{
			"schema_version": "issue/v1",
			"issue": map[string]interface{}{
				"id":      "ISSUE-042",
				"title":   "Fix parser crash on empty input",
				"type":    "bug",
				"status":  "open",
				"created": "2026-03-19",
			},
		},
		Sections: []string{"Problem"},
	}
}

func validClosedIssueArtifact() *artifact.ParsedArtifact {
	return &artifact.ParsedArtifact{
		Filename: "ISSUE-042-fix-parser-crash.md",
		Title:    "ISSUE-042: Fix Parser Crash",
		Metadata: map[string]string{
			"schema_version": "issue/v1",
		},
		Frontmatter: map[string]interface{}{
			"schema_version": "issue/v1",
			"issue": map[string]interface{}{
				"id":      "ISSUE-042",
				"title":   "Fix parser crash on empty input",
				"type":    "bug",
				"status":  "closed",
				"created": "2026-03-15",
				"closed":  "2026-03-19",
			},
			"requirements": []interface{}{
				map[string]interface{}{
					"id":   "REQ-001",
					"text": "Parser must return error on nil input",
				},
				map[string]interface{}{
					"id":       "REQ-002",
					"text":     "Parser must return error on empty string",
					"supports": "my-feature:REQ-003",
				},
			},
			"claims": []interface{}{
				map[string]interface{}{
					"id":          "CLM-001",
					"requirement": "REQ-001",
					"text":        "ParseFile returns ErrNilInput when given nil",
					"tests": []interface{}{
						map[string]interface{}{"test_name": "TestParseFile_NilInput"},
					},
				},
				map[string]interface{}{
					"id":          "CLM-002",
					"requirement": "REQ-002",
					"text":        "ParseFile returns ErrEmptyInput when given empty string",
					"tests": []interface{}{
						map[string]interface{}{"test_name": "TestParseFile_EmptyInput"},
					},
				},
			},
		},
		Sections: []string{"Problem", "Solution", "Resolution"},
	}
}

// --- Core validation tests ---

func TestValidateIssue_Valid_Open(t *testing.T) {
	result := validate.ValidateIssue(validIssueArtifact(), issueSchema())
	if !result.Pass() {
		t.Errorf("expected valid open issue to pass, got %d violations:", len(result.Violations))
		for _, v := range result.Violations {
			t.Errorf("  [%s] %s", v.Rule, v.Message)
		}
	}
}

func TestValidateIssue_Valid_Closed(t *testing.T) {
	result := validate.ValidateIssue(validClosedIssueArtifact(), issueSchema())
	if !result.Pass() {
		t.Errorf("expected valid closed issue to pass, got %d violations:", len(result.Violations))
		for _, v := range result.Violations {
			t.Errorf("  [%s] %s", v.Rule, v.Message)
		}
	}
}

// --- Filename tests ---

func TestValidateIssue_BadFilename(t *testing.T) {
	art := validIssueArtifact()
	art.Filename = "issue-042-bad.md"
	result := validate.ValidateIssue(art, issueSchema())
	assertHasViolation(t, result, "issue/filename-pattern")
}

// --- Schema version mismatch ---

func TestValidateIssue_SchemaVersionMismatch(t *testing.T) {
	art := validIssueArtifact()
	art.Metadata["schema_version"] = "spec/v1"
	result := validate.ValidateIssue(art, issueSchema())
	assertHasViolation(t, result, "issue/schema-version-mismatch")
}

// --- Issue block tests ---

func TestValidateIssue_MissingIssueBlock(t *testing.T) {
	art := validIssueArtifact()
	delete(art.Frontmatter, "issue")
	result := validate.ValidateIssue(art, issueSchema())
	assertHasViolation(t, result, "issue/block-required")
}

func TestValidateIssue_IssueBlockNotMap(t *testing.T) {
	art := validIssueArtifact()
	art.Frontmatter["issue"] = "not-a-map"
	result := validate.ValidateIssue(art, issueSchema())
	assertHasViolation(t, result, "issue/block-required")
}

func TestValidateIssue_MissingID(t *testing.T) {
	art := validIssueArtifact()
	delete(art.Frontmatter["issue"].(map[string]interface{}), "id")
	result := validate.ValidateIssue(art, issueSchema())
	assertHasViolation(t, result, "issue/id-required")
}

func TestValidateIssue_BadIDPattern(t *testing.T) {
	art := validIssueArtifact()
	art.Frontmatter["issue"].(map[string]interface{})["id"] = "ISS-42"
	result := validate.ValidateIssue(art, issueSchema())
	assertHasViolation(t, result, "issue/id-pattern")
}

func TestValidateIssue_MissingTitle(t *testing.T) {
	art := validIssueArtifact()
	delete(art.Frontmatter["issue"].(map[string]interface{}), "title")
	result := validate.ValidateIssue(art, issueSchema())
	assertHasViolation(t, result, "issue/title-required")
}

func TestValidateIssue_MissingType(t *testing.T) {
	art := validIssueArtifact()
	delete(art.Frontmatter["issue"].(map[string]interface{}), "type")
	result := validate.ValidateIssue(art, issueSchema())
	assertHasViolation(t, result, "issue/type-required")
}

func TestValidateIssue_BadType(t *testing.T) {
	art := validIssueArtifact()
	art.Frontmatter["issue"].(map[string]interface{})["type"] = "feature"
	result := validate.ValidateIssue(art, issueSchema())
	assertHasViolation(t, result, "issue/type-enum")
}

func TestValidateIssue_MissingStatus(t *testing.T) {
	art := validIssueArtifact()
	delete(art.Frontmatter["issue"].(map[string]interface{}), "status")
	result := validate.ValidateIssue(art, issueSchema())
	assertHasViolation(t, result, "issue/status-required")
}

func TestValidateIssue_BadStatus(t *testing.T) {
	art := validIssueArtifact()
	art.Frontmatter["issue"].(map[string]interface{})["status"] = "resolved"
	result := validate.ValidateIssue(art, issueSchema())
	assertHasViolation(t, result, "issue/status-enum")
}

func TestValidateIssue_MissingCreated(t *testing.T) {
	art := validIssueArtifact()
	delete(art.Frontmatter["issue"].(map[string]interface{}), "created")
	result := validate.ValidateIssue(art, issueSchema())
	assertHasViolation(t, result, "issue/created-required")
}

func TestValidateIssue_BadCreatedPattern(t *testing.T) {
	art := validIssueArtifact()
	art.Frontmatter["issue"].(map[string]interface{})["created"] = "March 19"
	result := validate.ValidateIssue(art, issueSchema())
	assertHasViolation(t, result, "issue/created-pattern")
}

// --- ID/filename consistency ---

func TestValidateIssue_IDFilenameMismatch(t *testing.T) {
	art := validIssueArtifact()
	art.Frontmatter["issue"].(map[string]interface{})["id"] = "ISSUE-099"
	result := validate.ValidateIssue(art, issueSchema())
	assertHasViolation(t, result, "issue/id-filename-mismatch")
}

func TestValidateIssue_IDFilenameConsistent(t *testing.T) {
	art := validIssueArtifact()
	result := validate.ValidateIssue(art, issueSchema())
	assertNoViolationRule(t, result, "issue/id-filename-mismatch")
}

// --- Status-gated rules ---

func TestValidateIssue_Blocked_MissingBlockedBy(t *testing.T) {
	art := validIssueArtifact()
	art.Frontmatter["issue"].(map[string]interface{})["status"] = "blocked"
	result := validate.ValidateIssue(art, issueSchema())
	assertHasViolation(t, result, "issue/blocked-requires-blocked-by")
}

func TestValidateIssue_Blocked_WithBlockedBy(t *testing.T) {
	art := validIssueArtifact()
	art.Frontmatter["issue"].(map[string]interface{})["status"] = "blocked"
	art.Frontmatter["context"] = map[string]interface{}{
		"blocked_by": []interface{}{"ISSUE-041"},
	}
	result := validate.ValidateIssue(art, issueSchema())
	assertNoViolationRule(t, result, "issue/blocked-requires-blocked-by")
}

func TestValidateIssue_Blocked_EmptyBlockedBy(t *testing.T) {
	art := validIssueArtifact()
	art.Frontmatter["issue"].(map[string]interface{})["status"] = "blocked"
	art.Frontmatter["context"] = map[string]interface{}{
		"blocked_by": []interface{}{},
	}
	result := validate.ValidateIssue(art, issueSchema())
	assertHasViolation(t, result, "issue/blocked-requires-blocked-by")
}

func TestValidateIssue_Closed_MissingClosedDate(t *testing.T) {
	art := validClosedIssueArtifact()
	delete(art.Frontmatter["issue"].(map[string]interface{}), "closed")
	result := validate.ValidateIssue(art, issueSchema())
	assertHasViolation(t, result, "issue/closed-requires-date")
}

func TestValidateIssue_Closed_WithClosedDate(t *testing.T) {
	art := validClosedIssueArtifact()
	result := validate.ValidateIssue(art, issueSchema())
	assertNoViolationRule(t, result, "issue/closed-requires-date")
}

// --- Complexity tests ---

func TestValidateIssue_Complexity_Valid(t *testing.T) {
	art := validIssueArtifact()
	art.Frontmatter["complexity"] = map[string]interface{}{
		"scope":       "contained",
		"uncertainty": "known",
		"risk":        "safe",
	}
	result := validate.ValidateIssue(art, issueSchema())
	assertNoViolationRule(t, result, "issue/complexity-scope-enum")
	assertNoViolationRule(t, result, "issue/complexity-uncertainty-enum")
	assertNoViolationRule(t, result, "issue/complexity-risk-enum")
}

func TestValidateIssue_Complexity_BadScope(t *testing.T) {
	art := validIssueArtifact()
	art.Frontmatter["complexity"] = map[string]interface{}{
		"scope": "huge",
	}
	result := validate.ValidateIssue(art, issueSchema())
	assertHasViolation(t, result, "issue/complexity-scope-enum")
}

func TestValidateIssue_Complexity_BadUncertainty(t *testing.T) {
	art := validIssueArtifact()
	art.Frontmatter["complexity"] = map[string]interface{}{
		"uncertainty": "unknown",
	}
	result := validate.ValidateIssue(art, issueSchema())
	assertHasViolation(t, result, "issue/complexity-uncertainty-enum")
}

func TestValidateIssue_Complexity_BadRisk(t *testing.T) {
	art := validIssueArtifact()
	art.Frontmatter["complexity"] = map[string]interface{}{
		"risk": "extreme",
	}
	result := validate.ValidateIssue(art, issueSchema())
	assertHasViolation(t, result, "issue/complexity-risk-enum")
}

func TestValidateIssue_Complexity_NotMap(t *testing.T) {
	art := validIssueArtifact()
	art.Frontmatter["complexity"] = "simple"
	result := validate.ValidateIssue(art, issueSchema())
	assertHasViolation(t, result, "issue/complexity-format")
}

func TestValidateIssue_NoComplexity_OK(t *testing.T) {
	art := validIssueArtifact()
	result := validate.ValidateIssue(art, issueSchema())
	assertNoViolationRule(t, result, "issue/complexity-format")
	assertNoViolationRule(t, result, "issue/complexity-scope-enum")
}

// --- Requirements tests (on close) ---

func TestValidateIssue_Open_NoRequirements_OK(t *testing.T) {
	art := validIssueArtifact()
	result := validate.ValidateIssue(art, issueSchema())
	assertNoViolationRule(t, result, "issue/requirements-required")
}

func TestValidateIssue_Closed_NoRequirements(t *testing.T) {
	art := validClosedIssueArtifact()
	delete(art.Frontmatter, "requirements")
	result := validate.ValidateIssue(art, issueSchema())
	assertHasViolation(t, result, "issue/requirements-required")
}

func TestValidateIssue_Closed_EmptyRequirements(t *testing.T) {
	art := validClosedIssueArtifact()
	art.Frontmatter["requirements"] = []interface{}{}
	result := validate.ValidateIssue(art, issueSchema())
	assertHasViolation(t, result, "issue/requirements-required")
}

func TestValidateIssue_Closed_RequirementsNotArray(t *testing.T) {
	art := validClosedIssueArtifact()
	art.Frontmatter["requirements"] = "not-array"
	result := validate.ValidateIssue(art, issueSchema())
	assertHasViolation(t, result, "issue/requirements-format")
}

func TestValidateIssue_Closed_RequirementNotMap(t *testing.T) {
	art := validClosedIssueArtifact()
	art.Frontmatter["requirements"] = []interface{}{"not-a-map"}
	// Claims still reference REQ-001/REQ-002 which won't exist
	result := validate.ValidateIssue(art, issueSchema())
	assertHasViolation(t, result, "issue/requirement-format")
}

func TestValidateIssue_Closed_RequirementMissingID(t *testing.T) {
	art := validClosedIssueArtifact()
	art.Frontmatter["requirements"] = []interface{}{
		map[string]interface{}{"text": "some requirement"},
	}
	art.Frontmatter["claims"] = []interface{}{
		map[string]interface{}{
			"id": "CLM-001", "requirement": "REQ-001",
			"text": "claim", "tests": []interface{}{map[string]interface{}{"test_name": "TestX"}},
		},
	}
	result := validate.ValidateIssue(art, issueSchema())
	assertHasViolation(t, result, "issue/requirement-id-required")
}

func TestValidateIssue_Closed_RequirementBadIDFormat(t *testing.T) {
	art := validClosedIssueArtifact()
	art.Frontmatter["requirements"] = []interface{}{
		map[string]interface{}{"id": "BAD-001", "text": "requirement"},
	}
	art.Frontmatter["claims"] = []interface{}{
		map[string]interface{}{
			"id": "CLM-001", "requirement": "BAD-001",
			"text": "claim", "tests": []interface{}{map[string]interface{}{"test_name": "TestX"}},
		},
	}
	result := validate.ValidateIssue(art, issueSchema())
	assertHasViolation(t, result, "issue/requirement-id-format")
}

func TestValidateIssue_Closed_DuplicateRequirementID(t *testing.T) {
	art := validClosedIssueArtifact()
	art.Frontmatter["requirements"] = []interface{}{
		map[string]interface{}{"id": "REQ-001", "text": "first"},
		map[string]interface{}{"id": "REQ-001", "text": "duplicate"},
	}
	result := validate.ValidateIssue(art, issueSchema())
	assertHasViolation(t, result, "issue/requirement-id-duplicate")
}

func TestValidateIssue_Closed_RequirementMissingText(t *testing.T) {
	art := validClosedIssueArtifact()
	art.Frontmatter["requirements"] = []interface{}{
		map[string]interface{}{"id": "REQ-001"},
	}
	art.Frontmatter["claims"] = []interface{}{
		map[string]interface{}{
			"id": "CLM-001", "requirement": "REQ-001",
			"text": "claim", "tests": []interface{}{map[string]interface{}{"test_name": "TestX"}},
		},
	}
	result := validate.ValidateIssue(art, issueSchema())
	assertHasViolation(t, result, "issue/requirement-text-required")
}

func TestValidateIssue_Closed_RequirementEmptyText(t *testing.T) {
	art := validClosedIssueArtifact()
	art.Frontmatter["requirements"] = []interface{}{
		map[string]interface{}{"id": "REQ-001", "text": "  "},
	}
	art.Frontmatter["claims"] = []interface{}{
		map[string]interface{}{
			"id": "CLM-001", "requirement": "REQ-001",
			"text": "claim", "tests": []interface{}{map[string]interface{}{"test_name": "TestX"}},
		},
	}
	result := validate.ValidateIssue(art, issueSchema())
	assertHasViolation(t, result, "issue/requirement-text-required")
}

// --- Supports field tests ---

func TestValidateIssue_Closed_ValidSupports(t *testing.T) {
	art := validClosedIssueArtifact()
	// REQ-002 already has supports: "my-feature:REQ-003" in the fixture
	result := validate.ValidateIssue(art, issueSchema())
	assertNoViolationRule(t, result, "issue/requirement-supports-format")
}

func TestValidateIssue_Closed_BadSupportsFormat(t *testing.T) {
	art := validClosedIssueArtifact()
	reqs := art.Frontmatter["requirements"].([]interface{})
	reqs[0].(map[string]interface{})["supports"] = "bad format"
	result := validate.ValidateIssue(art, issueSchema())
	assertHasViolation(t, result, "issue/requirement-supports-format")
}

func TestValidateIssue_Closed_EmptySupports(t *testing.T) {
	art := validClosedIssueArtifact()
	reqs := art.Frontmatter["requirements"].([]interface{})
	reqs[0].(map[string]interface{})["supports"] = ""
	result := validate.ValidateIssue(art, issueSchema())
	assertHasViolation(t, result, "issue/requirement-supports-format")
}

// --- Claims tests (full spec parity on close) ---

func TestValidateIssue_Open_NoClaims_OK(t *testing.T) {
	art := validIssueArtifact()
	result := validate.ValidateIssue(art, issueSchema())
	assertNoViolationRule(t, result, "issue/claims-required")
}

func TestValidateIssue_Closed_NoClaims(t *testing.T) {
	art := validClosedIssueArtifact()
	delete(art.Frontmatter, "claims")
	result := validate.ValidateIssue(art, issueSchema())
	assertHasViolation(t, result, "issue/claims-required")
}

func TestValidateIssue_Closed_EmptyClaims(t *testing.T) {
	art := validClosedIssueArtifact()
	art.Frontmatter["claims"] = []interface{}{}
	result := validate.ValidateIssue(art, issueSchema())
	assertHasViolation(t, result, "issue/claims-required")
}

func TestValidateIssue_Closed_ClaimsNotArray(t *testing.T) {
	art := validClosedIssueArtifact()
	art.Frontmatter["claims"] = "not-an-array"
	result := validate.ValidateIssue(art, issueSchema())
	assertHasViolation(t, result, "issue/claims-format")
}

func TestValidateIssue_Closed_ClaimNotMap(t *testing.T) {
	art := validClosedIssueArtifact()
	art.Frontmatter["claims"] = []interface{}{"not-a-map"}
	result := validate.ValidateIssue(art, issueSchema())
	assertHasViolation(t, result, "issue/claim-format")
}

func TestValidateIssue_Closed_ClaimMissingID(t *testing.T) {
	art := validClosedIssueArtifact()
	art.Frontmatter["claims"] = []interface{}{
		map[string]interface{}{
			"requirement": "REQ-001", "text": "some claim",
			"tests": []interface{}{map[string]interface{}{"test_name": "TestX"}},
		},
	}
	result := validate.ValidateIssue(art, issueSchema())
	assertHasViolation(t, result, "issue/claim-id-required")
}

func TestValidateIssue_Closed_ClaimBadIDPattern(t *testing.T) {
	art := validClosedIssueArtifact()
	art.Frontmatter["claims"] = []interface{}{
		map[string]interface{}{
			"id": "AC-001.1", "requirement": "REQ-001",
			"text": "wrong pattern",
			"tests": []interface{}{map[string]interface{}{"test_name": "TestX"}},
		},
	}
	result := validate.ValidateIssue(art, issueSchema())
	assertHasViolation(t, result, "issue/claim-id-format")
}

func TestValidateIssue_Closed_DuplicateClaimID(t *testing.T) {
	art := validClosedIssueArtifact()
	art.Frontmatter["claims"] = []interface{}{
		map[string]interface{}{
			"id": "CLM-001", "requirement": "REQ-001",
			"text": "first", "tests": []interface{}{map[string]interface{}{"test_name": "TestX"}},
		},
		map[string]interface{}{
			"id": "CLM-001", "requirement": "REQ-002",
			"text": "duplicate", "tests": []interface{}{map[string]interface{}{"test_name": "TestY"}},
		},
	}
	result := validate.ValidateIssue(art, issueSchema())
	assertHasViolation(t, result, "issue/claim-id-duplicate")
}

func TestValidateIssue_Closed_ClaimMissingText(t *testing.T) {
	art := validClosedIssueArtifact()
	art.Frontmatter["claims"] = []interface{}{
		map[string]interface{}{
			"id": "CLM-001", "requirement": "REQ-001",
			"tests": []interface{}{map[string]interface{}{"test_name": "TestX"}},
		},
	}
	result := validate.ValidateIssue(art, issueSchema())
	assertHasViolation(t, result, "issue/claim-text-required")
}

func TestValidateIssue_Closed_ClaimMissingRequirement(t *testing.T) {
	art := validClosedIssueArtifact()
	art.Frontmatter["claims"] = []interface{}{
		map[string]interface{}{
			"id": "CLM-001", "text": "claim without req",
			"tests": []interface{}{map[string]interface{}{"test_name": "TestX"}},
		},
	}
	result := validate.ValidateIssue(art, issueSchema())
	assertHasViolation(t, result, "issue/claim-requirement-required")
}

func TestValidateIssue_Closed_ClaimInvalidRequirement(t *testing.T) {
	art := validClosedIssueArtifact()
	art.Frontmatter["claims"] = []interface{}{
		map[string]interface{}{
			"id": "CLM-001", "requirement": "REQ-999",
			"text": "refs nonexistent req",
			"tests": []interface{}{map[string]interface{}{"test_name": "TestX"}},
		},
		map[string]interface{}{
			"id": "CLM-002", "requirement": "REQ-001",
			"text": "valid", "tests": []interface{}{map[string]interface{}{"test_name": "TestY"}},
		},
	}
	result := validate.ValidateIssue(art, issueSchema())
	assertHasViolation(t, result, "issue/claim-requirement-invalid")
}

func TestValidateIssue_Closed_ClaimMissingTests(t *testing.T) {
	art := validClosedIssueArtifact()
	art.Frontmatter["claims"] = []interface{}{
		map[string]interface{}{
			"id": "CLM-001", "requirement": "REQ-001", "text": "no tests",
		},
		map[string]interface{}{
			"id": "CLM-002", "requirement": "REQ-002", "text": "valid",
			"tests": []interface{}{map[string]interface{}{"test_name": "TestX"}},
		},
	}
	result := validate.ValidateIssue(art, issueSchema())
	assertHasViolation(t, result, "issue/claim-tests-required")
}

func TestValidateIssue_Closed_ClaimEmptyTests(t *testing.T) {
	art := validClosedIssueArtifact()
	art.Frontmatter["claims"] = []interface{}{
		map[string]interface{}{
			"id": "CLM-001", "requirement": "REQ-001", "text": "empty tests",
			"tests": []interface{}{},
		},
		map[string]interface{}{
			"id": "CLM-002", "requirement": "REQ-002", "text": "valid",
			"tests": []interface{}{map[string]interface{}{"test_name": "TestX"}},
		},
	}
	result := validate.ValidateIssue(art, issueSchema())
	assertHasViolation(t, result, "issue/claim-tests-empty")
}

// --- Requirement coverage ---

func TestValidateIssue_Closed_UncoveredRequirement(t *testing.T) {
	art := validClosedIssueArtifact()
	// Only CLM-001 covers REQ-001, nothing covers REQ-002
	art.Frontmatter["claims"] = []interface{}{
		map[string]interface{}{
			"id": "CLM-001", "requirement": "REQ-001",
			"text": "covers REQ-001 only",
			"tests": []interface{}{map[string]interface{}{"test_name": "TestX"}},
		},
	}
	result := validate.ValidateIssue(art, issueSchema())
	assertHasViolationContaining(t, result, "issue/requirement-uncovered", "REQ-002")
}

func TestValidateIssue_Closed_AllRequirementsCovered(t *testing.T) {
	art := validClosedIssueArtifact()
	result := validate.ValidateIssue(art, issueSchema())
	assertNoViolationRule(t, result, "issue/requirement-uncovered")
}

// --- Open issues skip traceability validation ---

func TestValidateIssue_Open_BadClaimsNotValidated(t *testing.T) {
	art := validIssueArtifact()
	art.Frontmatter["claims"] = []interface{}{
		map[string]interface{}{"id": "bad-id"},
	}
	result := validate.ValidateIssue(art, issueSchema())
	assertNoViolationRule(t, result, "issue/claim-id-format")
	assertNoViolationRule(t, result, "issue/claim-text-required")
}

func TestValidateIssue_InProgress_EnforcesTraceability(t *testing.T) {
	art := validIssueArtifact()
	art.Frontmatter["issue"].(map[string]interface{})["status"] = "in-progress"
	result := validate.ValidateIssue(art, issueSchema())
	assertHasViolation(t, result, "issue/requirements-required")
	assertHasViolation(t, result, "issue/claims-required")
}

func TestValidateIssue_Ready_EnforcesTraceability(t *testing.T) {
	art := validIssueArtifact()
	art.Frontmatter["issue"].(map[string]interface{})["status"] = "ready"
	result := validate.ValidateIssue(art, issueSchema())
	assertHasViolation(t, result, "issue/requirements-required")
	assertHasViolation(t, result, "issue/claims-required")
}

func TestValidateIssue_Ready_WithFullTraceability_Passes(t *testing.T) {
	art := validClosedIssueArtifact()
	art.Frontmatter["issue"].(map[string]interface{})["status"] = "ready"
	delete(art.Frontmatter["issue"].(map[string]interface{}), "closed")
	result := validate.ValidateIssue(art, issueSchema())
	assertNoViolationRule(t, result, "issue/requirements-required")
	assertNoViolationRule(t, result, "issue/claims-required")
	assertNoViolationRule(t, result, "issue/requirement-uncovered")
}

// --- Composition test ---

func TestValidateIssue_ComposesBaseAndIssue(t *testing.T) {
	art := validIssueArtifact()
	art.Title = ""
	art.Sections = []string{}
	delete(art.Frontmatter["issue"].(map[string]interface{}), "id")
	result := validate.ValidateIssue(art, issueSchema())
	assertHasViolation(t, result, "base/title-required")
	assertHasViolation(t, result, "base/section-required")
	assertHasViolation(t, result, "issue/id-required")
}

// --- All issue types valid ---

func TestValidateIssue_AllTypesValid(t *testing.T) {
	types := []string{"bug", "technical-debt", "enhancement", "question", "policy-violation"}
	for _, typ := range types {
		art := validIssueArtifact()
		art.Frontmatter["issue"].(map[string]interface{})["type"] = typ
		result := validate.ValidateIssue(art, issueSchema())
		assertNoViolationRule(t, result, "issue/type-enum")
	}
}

func TestValidateIssue_EmptyFrontmatter(t *testing.T) {
	art := &artifact.ParsedArtifact{
		Filename:    "ISSUE-001-empty.md",
		Title:       "Empty",
		Metadata:    map[string]string{"schema_version": "issue/v1"},
		Frontmatter: map[string]interface{}{"schema_version": "issue/v1"},
		Sections:    []string{"Problem"},
	}
	result := validate.ValidateIssue(art, issueSchema())
	assertHasViolation(t, result, "issue/block-required")
}
