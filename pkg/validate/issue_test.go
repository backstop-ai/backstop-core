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
			"verification": map[string]interface{}{
				"level":              "unit",
				"coverage_threshold": 90,
				"test_command":       "go test ./...",
			},
			"implementation": map[string]interface{}{
				"summary": "Fix nil/empty input handling in parser",
				"package": "pkg/artifact",
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
			"contracts": []interface{}{
				map[string]interface{}{
					"file": "pkg/artifact/parse.go",
					"provides": []interface{}{
						map[string]interface{}{
							"name":      "ParseFile",
							"kind":      "function",
							"signature": "func ParseFile(path string) (*ParsedArtifact, error)",
						},
					},
					"consumes": []interface{}{
						map[string]interface{}{
							"source": "pkg/artifact/artifact.go",
							"name":   "ParsedArtifact",
							"kind":   "type",
						},
					},
				},
			},
		},
		Sections: []string{"Problem", "Solution", "Resolution"},
	}
}

// --- Core validation tests ---

func TestIssue_Valid_Open(t *testing.T) {
	result := validate.Issue(validIssueArtifact(), issueSchema())
	if !result.Pass() {
		t.Errorf("expected valid open issue to pass, got %d violations:", len(result.Violations))
		for _, v := range result.Violations {
			t.Errorf("  [%s] %s", v.Rule, v.Message)
		}
	}
}

func TestIssue_Valid_Closed(t *testing.T) {
	result := validate.Issue(validClosedIssueArtifact(), issueSchema())
	if !result.Pass() {
		t.Errorf("expected valid closed issue to pass, got %d violations:", len(result.Violations))
		for _, v := range result.Violations {
			t.Errorf("  [%s] %s", v.Rule, v.Message)
		}
	}
}

// --- Filename tests ---

func TestIssue_BadFilename(t *testing.T) {
	art := validIssueArtifact()
	art.Filename = "issue-042-bad.md"
	result := validate.Issue(art, issueSchema())
	assertHasViolation(t, result, "issue/filename-pattern")
}

// --- Schema version mismatch ---

func TestIssue_SchemaVersionMismatch(t *testing.T) {
	art := validIssueArtifact()
	art.Metadata["schema_version"] = "spec/v1"
	result := validate.Issue(art, issueSchema())
	assertHasViolation(t, result, "issue/schema-version-mismatch")
}

// --- Issue block tests ---

func TestIssue_MissingIssueBlock(t *testing.T) {
	art := validIssueArtifact()
	delete(art.Frontmatter, "issue")
	result := validate.Issue(art, issueSchema())
	assertHasViolation(t, result, "issue/block-required")
}

func TestIssue_IssueBlockNotMap(t *testing.T) {
	art := validIssueArtifact()
	art.Frontmatter["issue"] = "not-a-map"
	result := validate.Issue(art, issueSchema())
	assertHasViolation(t, result, "issue/block-required")
}

func TestIssue_MissingID(t *testing.T) {
	art := validIssueArtifact()
	delete(art.Frontmatter["issue"].(map[string]interface{}), "id")
	result := validate.Issue(art, issueSchema())
	assertHasViolation(t, result, "issue/id-required")
}

func TestIssue_BadIDPattern(t *testing.T) {
	art := validIssueArtifact()
	art.Frontmatter["issue"].(map[string]interface{})["id"] = "ISS-42"
	result := validate.Issue(art, issueSchema())
	assertHasViolation(t, result, "issue/id-pattern")
}

func TestIssue_MissingTitle(t *testing.T) {
	art := validIssueArtifact()
	delete(art.Frontmatter["issue"].(map[string]interface{}), "title")
	result := validate.Issue(art, issueSchema())
	assertHasViolation(t, result, "issue/title-required")
}

func TestIssue_MissingType(t *testing.T) {
	art := validIssueArtifact()
	delete(art.Frontmatter["issue"].(map[string]interface{}), "type")
	result := validate.Issue(art, issueSchema())
	assertHasViolation(t, result, "issue/type-required")
}

func TestIssue_BadType(t *testing.T) {
	art := validIssueArtifact()
	art.Frontmatter["issue"].(map[string]interface{})["type"] = "feature"
	result := validate.Issue(art, issueSchema())
	assertHasViolation(t, result, "issue/type-enum")
}

func TestIssue_MissingStatus(t *testing.T) {
	art := validIssueArtifact()
	delete(art.Frontmatter["issue"].(map[string]interface{}), "status")
	result := validate.Issue(art, issueSchema())
	assertHasViolation(t, result, "issue/status-required")
}

func TestIssue_BadStatus(t *testing.T) {
	art := validIssueArtifact()
	art.Frontmatter["issue"].(map[string]interface{})["status"] = "resolved"
	result := validate.Issue(art, issueSchema())
	assertHasViolation(t, result, "issue/status-enum")
}

func TestIssue_MissingCreated(t *testing.T) {
	art := validIssueArtifact()
	delete(art.Frontmatter["issue"].(map[string]interface{}), "created")
	result := validate.Issue(art, issueSchema())
	assertHasViolation(t, result, "issue/created-required")
}

func TestIssue_BadCreatedPattern(t *testing.T) {
	art := validIssueArtifact()
	art.Frontmatter["issue"].(map[string]interface{})["created"] = "March 19"
	result := validate.Issue(art, issueSchema())
	assertHasViolation(t, result, "issue/created-pattern")
}

// --- ID/filename consistency ---

func TestIssue_IDFilenameMismatch(t *testing.T) {
	art := validIssueArtifact()
	art.Frontmatter["issue"].(map[string]interface{})["id"] = "ISSUE-099"
	result := validate.Issue(art, issueSchema())
	assertHasViolation(t, result, "issue/id-filename-mismatch")
}

func TestIssue_IDFilenameConsistent(t *testing.T) {
	art := validIssueArtifact()
	result := validate.Issue(art, issueSchema())
	assertNoViolationRule(t, result, "issue/id-filename-mismatch")
}

// --- Status-gated rules ---

func TestIssue_Blocked_MissingBlockedBy(t *testing.T) {
	art := validIssueArtifact()
	art.Frontmatter["issue"].(map[string]interface{})["status"] = "blocked"
	result := validate.Issue(art, issueSchema())
	assertHasViolation(t, result, "issue/blocked-requires-blocked-by")
}

func TestIssue_Blocked_WithBlockedBy(t *testing.T) {
	art := validIssueArtifact()
	art.Frontmatter["issue"].(map[string]interface{})["status"] = "blocked"
	art.Frontmatter["context"] = map[string]interface{}{
		"blocked_by": []interface{}{"ISSUE-041"},
	}
	result := validate.Issue(art, issueSchema())
	assertNoViolationRule(t, result, "issue/blocked-requires-blocked-by")
}

func TestIssue_Blocked_EmptyBlockedBy(t *testing.T) {
	art := validIssueArtifact()
	art.Frontmatter["issue"].(map[string]interface{})["status"] = "blocked"
	art.Frontmatter["context"] = map[string]interface{}{
		"blocked_by": []interface{}{},
	}
	result := validate.Issue(art, issueSchema())
	assertHasViolation(t, result, "issue/blocked-requires-blocked-by")
}

func TestIssue_Closed_MissingClosedDate(t *testing.T) {
	art := validClosedIssueArtifact()
	delete(art.Frontmatter["issue"].(map[string]interface{}), "closed")
	result := validate.Issue(art, issueSchema())
	assertHasViolation(t, result, "issue/closed-requires-date")
}

func TestIssue_Closed_WithClosedDate(t *testing.T) {
	art := validClosedIssueArtifact()
	result := validate.Issue(art, issueSchema())
	assertNoViolationRule(t, result, "issue/closed-requires-date")
}

// --- Complexity tests ---

func TestIssue_Complexity_Valid(t *testing.T) {
	art := validIssueArtifact()
	art.Frontmatter["complexity"] = map[string]interface{}{
		"scope":       "contained",
		"uncertainty": "known",
		"risk":        "safe",
	}
	result := validate.Issue(art, issueSchema())
	assertNoViolationRule(t, result, "issue/complexity-scope-enum")
	assertNoViolationRule(t, result, "issue/complexity-uncertainty-enum")
	assertNoViolationRule(t, result, "issue/complexity-risk-enum")
}

func TestIssue_Complexity_BadScope(t *testing.T) {
	art := validIssueArtifact()
	art.Frontmatter["complexity"] = map[string]interface{}{
		"scope": "huge",
	}
	result := validate.Issue(art, issueSchema())
	assertHasViolation(t, result, "issue/complexity-scope-enum")
}

func TestIssue_Complexity_BadUncertainty(t *testing.T) {
	art := validIssueArtifact()
	art.Frontmatter["complexity"] = map[string]interface{}{
		"uncertainty": "unknown",
	}
	result := validate.Issue(art, issueSchema())
	assertHasViolation(t, result, "issue/complexity-uncertainty-enum")
}

func TestIssue_Complexity_BadRisk(t *testing.T) {
	art := validIssueArtifact()
	art.Frontmatter["complexity"] = map[string]interface{}{
		"risk": "extreme",
	}
	result := validate.Issue(art, issueSchema())
	assertHasViolation(t, result, "issue/complexity-risk-enum")
}

func TestIssue_Complexity_NotMap(t *testing.T) {
	art := validIssueArtifact()
	art.Frontmatter["complexity"] = "simple"
	result := validate.Issue(art, issueSchema())
	assertHasViolation(t, result, "issue/complexity-format")
}

func TestIssue_NoComplexity_OK(t *testing.T) {
	art := validIssueArtifact()
	result := validate.Issue(art, issueSchema())
	assertNoViolationRule(t, result, "issue/complexity-format")
	assertNoViolationRule(t, result, "issue/complexity-scope-enum")
}

// --- Requirements tests (on close) ---

func TestIssue_Open_NoRequirements_OK(t *testing.T) {
	art := validIssueArtifact()
	result := validate.Issue(art, issueSchema())
	assertNoViolationRule(t, result, "issue/requirements-required")
}

func TestIssue_Closed_NoRequirements(t *testing.T) {
	art := validClosedIssueArtifact()
	delete(art.Frontmatter, "requirements")
	result := validate.Issue(art, issueSchema())
	assertHasViolation(t, result, "issue/requirements-required")
}

func TestIssue_Closed_EmptyRequirements(t *testing.T) {
	art := validClosedIssueArtifact()
	art.Frontmatter["requirements"] = []interface{}{}
	result := validate.Issue(art, issueSchema())
	assertHasViolation(t, result, "issue/requirements-required")
}

func TestIssue_Closed_RequirementsNotArray(t *testing.T) {
	art := validClosedIssueArtifact()
	art.Frontmatter["requirements"] = "not-array"
	result := validate.Issue(art, issueSchema())
	assertHasViolation(t, result, "issue/requirements-format")
}

func TestIssue_Closed_RequirementNotMap(t *testing.T) {
	art := validClosedIssueArtifact()
	art.Frontmatter["requirements"] = []interface{}{"not-a-map"}
	// Claims still reference REQ-001/REQ-002 which won't exist
	result := validate.Issue(art, issueSchema())
	assertHasViolation(t, result, "issue/requirement-format")
}

func TestIssue_Closed_RequirementMissingID(t *testing.T) {
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
	result := validate.Issue(art, issueSchema())
	assertHasViolation(t, result, "issue/requirement-id-required")
}

func TestIssue_Closed_RequirementBadIDFormat(t *testing.T) {
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
	result := validate.Issue(art, issueSchema())
	assertHasViolation(t, result, "issue/requirement-id-format")
}

func TestIssue_Closed_DuplicateRequirementID(t *testing.T) {
	art := validClosedIssueArtifact()
	art.Frontmatter["requirements"] = []interface{}{
		map[string]interface{}{"id": "REQ-001", "text": "first"},
		map[string]interface{}{"id": "REQ-001", "text": "duplicate"},
	}
	result := validate.Issue(art, issueSchema())
	assertHasViolation(t, result, "issue/requirement-id-duplicate")
}

func TestIssue_Closed_RequirementMissingText(t *testing.T) {
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
	result := validate.Issue(art, issueSchema())
	assertHasViolation(t, result, "issue/requirement-text-required")
}

func TestIssue_Closed_RequirementEmptyText(t *testing.T) {
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
	result := validate.Issue(art, issueSchema())
	assertHasViolation(t, result, "issue/requirement-text-required")
}

// --- Supports field tests ---

func TestIssue_Closed_ValidSupports(t *testing.T) {
	art := validClosedIssueArtifact()
	// REQ-002 already has supports: "my-feature:REQ-003" in the fixture
	result := validate.Issue(art, issueSchema())
	assertNoViolationRule(t, result, "issue/requirement-supports-format")
}

func TestIssue_Closed_BadSupportsFormat(t *testing.T) {
	art := validClosedIssueArtifact()
	reqs := art.Frontmatter["requirements"].([]interface{})
	reqs[0].(map[string]interface{})["supports"] = "bad format"
	result := validate.Issue(art, issueSchema())
	assertHasViolation(t, result, "issue/requirement-supports-format")
}

func TestIssue_Closed_EmptySupports(t *testing.T) {
	art := validClosedIssueArtifact()
	reqs := art.Frontmatter["requirements"].([]interface{})
	reqs[0].(map[string]interface{})["supports"] = ""
	result := validate.Issue(art, issueSchema())
	assertHasViolation(t, result, "issue/requirement-supports-format")
}

// --- Claims tests (full spec parity on close) ---

func TestIssue_Open_NoClaims_OK(t *testing.T) {
	art := validIssueArtifact()
	result := validate.Issue(art, issueSchema())
	assertNoViolationRule(t, result, "issue/claims-required")
}

func TestIssue_Closed_NoClaims(t *testing.T) {
	art := validClosedIssueArtifact()
	delete(art.Frontmatter, "claims")
	result := validate.Issue(art, issueSchema())
	assertHasViolation(t, result, "issue/claims-required")
}

func TestIssue_Closed_EmptyClaims(t *testing.T) {
	art := validClosedIssueArtifact()
	art.Frontmatter["claims"] = []interface{}{}
	result := validate.Issue(art, issueSchema())
	assertHasViolation(t, result, "issue/claims-required")
}

func TestIssue_Closed_ClaimsNotArray(t *testing.T) {
	art := validClosedIssueArtifact()
	art.Frontmatter["claims"] = "not-an-array"
	result := validate.Issue(art, issueSchema())
	assertHasViolation(t, result, "issue/claims-format")
}

func TestIssue_Closed_ClaimNotMap(t *testing.T) {
	art := validClosedIssueArtifact()
	art.Frontmatter["claims"] = []interface{}{"not-a-map"}
	result := validate.Issue(art, issueSchema())
	assertHasViolation(t, result, "issue/claim-format")
}

func TestIssue_Closed_ClaimMissingID(t *testing.T) {
	art := validClosedIssueArtifact()
	art.Frontmatter["claims"] = []interface{}{
		map[string]interface{}{
			"requirement": "REQ-001", "text": "some claim",
			"tests": []interface{}{map[string]interface{}{"test_name": "TestX"}},
		},
	}
	result := validate.Issue(art, issueSchema())
	assertHasViolation(t, result, "issue/claim-id-required")
}

func TestIssue_Closed_ClaimBadIDPattern(t *testing.T) {
	art := validClosedIssueArtifact()
	art.Frontmatter["claims"] = []interface{}{
		map[string]interface{}{
			"id": "AC-001.1", "requirement": "REQ-001",
			"text":  "wrong pattern",
			"tests": []interface{}{map[string]interface{}{"test_name": "TestX"}},
		},
	}
	result := validate.Issue(art, issueSchema())
	assertHasViolation(t, result, "issue/claim-id-format")
}

func TestIssue_Closed_DuplicateClaimID(t *testing.T) {
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
	result := validate.Issue(art, issueSchema())
	assertHasViolation(t, result, "issue/claim-id-duplicate")
}

func TestIssue_Closed_ClaimMissingText(t *testing.T) {
	art := validClosedIssueArtifact()
	art.Frontmatter["claims"] = []interface{}{
		map[string]interface{}{
			"id": "CLM-001", "requirement": "REQ-001",
			"tests": []interface{}{map[string]interface{}{"test_name": "TestX"}},
		},
	}
	result := validate.Issue(art, issueSchema())
	assertHasViolation(t, result, "issue/claim-text-required")
}

func TestIssue_Closed_ClaimMissingRequirement(t *testing.T) {
	art := validClosedIssueArtifact()
	art.Frontmatter["claims"] = []interface{}{
		map[string]interface{}{
			"id": "CLM-001", "text": "claim without req",
			"tests": []interface{}{map[string]interface{}{"test_name": "TestX"}},
		},
	}
	result := validate.Issue(art, issueSchema())
	assertHasViolation(t, result, "issue/claim-requirement-required")
}

func TestIssue_Closed_ClaimInvalidRequirement(t *testing.T) {
	art := validClosedIssueArtifact()
	art.Frontmatter["claims"] = []interface{}{
		map[string]interface{}{
			"id": "CLM-001", "requirement": "REQ-999",
			"text":  "refs nonexistent req",
			"tests": []interface{}{map[string]interface{}{"test_name": "TestX"}},
		},
		map[string]interface{}{
			"id": "CLM-002", "requirement": "REQ-001",
			"text": "valid", "tests": []interface{}{map[string]interface{}{"test_name": "TestY"}},
		},
	}
	result := validate.Issue(art, issueSchema())
	assertHasViolation(t, result, "issue/claim-requirement-invalid")
}

func TestIssue_Closed_ClaimMissingTests(t *testing.T) {
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
	result := validate.Issue(art, issueSchema())
	assertHasViolation(t, result, "issue/claim-tests-required")
}

func TestIssue_Closed_ClaimEmptyTests(t *testing.T) {
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
	result := validate.Issue(art, issueSchema())
	assertHasViolation(t, result, "issue/claim-tests-empty")
}

// --- Requirement coverage ---

func TestIssue_Closed_UncoveredRequirement(t *testing.T) {
	art := validClosedIssueArtifact()
	// Only CLM-001 covers REQ-001, nothing covers REQ-002
	art.Frontmatter["claims"] = []interface{}{
		map[string]interface{}{
			"id": "CLM-001", "requirement": "REQ-001",
			"text":  "covers REQ-001 only",
			"tests": []interface{}{map[string]interface{}{"test_name": "TestX"}},
		},
	}
	result := validate.Issue(art, issueSchema())
	assertHasViolationContaining(t, result, "issue/requirement-uncovered", "REQ-002")
}

func TestIssue_Closed_AllRequirementsCovered(t *testing.T) {
	art := validClosedIssueArtifact()
	result := validate.Issue(art, issueSchema())
	assertNoViolationRule(t, result, "issue/requirement-uncovered")
}

// --- Open issues skip traceability validation ---

func TestIssue_Open_BadClaimsNotValidated(t *testing.T) {
	art := validIssueArtifact()
	art.Frontmatter["claims"] = []interface{}{
		map[string]interface{}{"id": "bad-id"},
	}
	result := validate.Issue(art, issueSchema())
	assertNoViolationRule(t, result, "issue/claim-id-format")
	assertNoViolationRule(t, result, "issue/claim-text-required")
}

func TestIssue_InProgress_EnforcesTraceability(t *testing.T) {
	art := validIssueArtifact()
	art.Frontmatter["issue"].(map[string]interface{})["status"] = "in-progress"
	result := validate.Issue(art, issueSchema())
	assertHasViolation(t, result, "issue/requirements-required")
	assertHasViolation(t, result, "issue/claims-required")
}

func TestIssue_Ready_EnforcesTraceability(t *testing.T) {
	art := validIssueArtifact()
	art.Frontmatter["issue"].(map[string]interface{})["status"] = "ready"
	result := validate.Issue(art, issueSchema())
	assertHasViolation(t, result, "issue/requirements-required")
	assertHasViolation(t, result, "issue/claims-required")
}

func TestIssue_Ready_WithFullTraceability_Passes(t *testing.T) {
	art := validClosedIssueArtifact()
	art.Frontmatter["issue"].(map[string]interface{})["status"] = "ready"
	delete(art.Frontmatter["issue"].(map[string]interface{}), "closed")
	result := validate.Issue(art, issueSchema())
	assertNoViolationRule(t, result, "issue/requirements-required")
	assertNoViolationRule(t, result, "issue/claims-required")
	assertNoViolationRule(t, result, "issue/requirement-uncovered")
}

// --- Composition test ---

func TestIssue_ComposesBaseAndIssue(t *testing.T) {
	art := validIssueArtifact()
	art.Title = ""
	art.Sections = []string{}
	delete(art.Frontmatter["issue"].(map[string]interface{}), "id")
	result := validate.Issue(art, issueSchema())
	assertHasViolation(t, result, "base/title-required")
	assertHasViolation(t, result, "base/section-required")
	assertHasViolation(t, result, "issue/id-required")
}

// --- All issue types valid ---

func TestIssue_AllTypesValid(t *testing.T) {
	types := []string{"bug", "technical-debt", "enhancement", "question", "policy-violation"}
	for _, typ := range types {
		art := validIssueArtifact()
		art.Frontmatter["issue"].(map[string]interface{})["type"] = typ
		result := validate.Issue(art, issueSchema())
		assertNoViolationRule(t, result, "issue/type-enum")
	}
}

func TestIssue_EmptyFrontmatter(t *testing.T) {
	art := &artifact.ParsedArtifact{
		Filename:    "ISSUE-001-empty.md",
		Title:       "Empty",
		Metadata:    map[string]string{"schema_version": "issue/v1"},
		Frontmatter: map[string]interface{}{"schema_version": "issue/v1"},
		Sections:    []string{"Problem"},
	}
	result := validate.Issue(art, issueSchema())
	assertHasViolation(t, result, "issue/block-required")
}

// --- Verification block on issues ---

func TestIssue_Ready_MissingVerification(t *testing.T) {
	art := validClosedIssueArtifact()
	art.Frontmatter["issue"].(map[string]interface{})["status"] = "ready"
	delete(art.Frontmatter["issue"].(map[string]interface{}), "closed")
	delete(art.Frontmatter, "verification")
	result := validate.Issue(art, issueSchema())
	assertHasViolation(t, result, "issue/verification-required")
}

func TestIssue_Ready_VerificationNotMap(t *testing.T) {
	art := validClosedIssueArtifact()
	art.Frontmatter["issue"].(map[string]interface{})["status"] = "ready"
	delete(art.Frontmatter["issue"].(map[string]interface{}), "closed")
	art.Frontmatter["verification"] = "not-a-map"
	result := validate.Issue(art, issueSchema())
	assertHasViolation(t, result, "issue/verification-required")
}

func TestIssue_Ready_VerificationMissingLevel(t *testing.T) {
	art := validClosedIssueArtifact()
	art.Frontmatter["issue"].(map[string]interface{})["status"] = "ready"
	delete(art.Frontmatter["issue"].(map[string]interface{}), "closed")
	art.Frontmatter["verification"] = map[string]interface{}{
		"test_command": "go test ./...",
	}
	result := validate.Issue(art, issueSchema())
	assertHasViolation(t, result, "issue/verification-level-required")
}

func TestIssue_Ready_VerificationBadLevel(t *testing.T) {
	art := validClosedIssueArtifact()
	art.Frontmatter["issue"].(map[string]interface{})["status"] = "ready"
	delete(art.Frontmatter["issue"].(map[string]interface{}), "closed")
	art.Frontmatter["verification"] = map[string]interface{}{
		"level":        "e2e",
		"test_command": "go test ./...",
	}
	result := validate.Issue(art, issueSchema())
	assertHasViolation(t, result, "issue/verification-level-invalid")
}

func TestIssue_Ready_ThresholdMismatch(t *testing.T) {
	art := validClosedIssueArtifact()
	art.Frontmatter["issue"].(map[string]interface{})["status"] = "ready"
	delete(art.Frontmatter["issue"].(map[string]interface{}), "closed")
	art.Frontmatter["verification"] = map[string]interface{}{
		"level":              "unit",
		"coverage_threshold": 80,
		"test_command":       "go test ./...",
	}
	result := validate.Issue(art, issueSchema())
	assertHasViolation(t, result, "issue/threshold-value")
}

func TestIssue_Ready_ThresholdNotAllowed(t *testing.T) {
	art := validClosedIssueArtifact()
	art.Frontmatter["issue"].(map[string]interface{})["status"] = "ready"
	delete(art.Frontmatter["issue"].(map[string]interface{}), "closed")
	art.Frontmatter["verification"] = map[string]interface{}{
		"level":              "static",
		"coverage_threshold": 90,
		"test_command":       "go test ./...",
	}
	result := validate.Issue(art, issueSchema())
	assertHasViolation(t, result, "issue/threshold-not-allowed")
}

func TestIssue_Ready_MissingTestCommand(t *testing.T) {
	art := validClosedIssueArtifact()
	art.Frontmatter["issue"].(map[string]interface{})["status"] = "ready"
	delete(art.Frontmatter["issue"].(map[string]interface{}), "closed")
	art.Frontmatter["verification"] = map[string]interface{}{
		"level":              "unit",
		"coverage_threshold": 90,
	}
	result := validate.Issue(art, issueSchema())
	assertHasViolation(t, result, "issue/test-command-required")
}

func TestIssue_Open_VerificationNotRequired(t *testing.T) {
	art := validIssueArtifact()
	result := validate.Issue(art, issueSchema())
	assertNoViolationRule(t, result, "issue/verification-required")
}

// --- Implementation block on issues ---

func TestIssue_Ready_MissingImplementation(t *testing.T) {
	art := validClosedIssueArtifact()
	art.Frontmatter["issue"].(map[string]interface{})["status"] = "ready"
	delete(art.Frontmatter["issue"].(map[string]interface{}), "closed")
	delete(art.Frontmatter, "implementation")
	result := validate.Issue(art, issueSchema())
	assertHasViolation(t, result, "issue/implementation-required")
}

func TestIssue_Ready_ImplementationNotMap(t *testing.T) {
	art := validClosedIssueArtifact()
	art.Frontmatter["issue"].(map[string]interface{})["status"] = "ready"
	delete(art.Frontmatter["issue"].(map[string]interface{}), "closed")
	art.Frontmatter["implementation"] = "not-a-map"
	result := validate.Issue(art, issueSchema())
	assertHasViolation(t, result, "issue/implementation-required")
}

func TestIssue_Ready_MissingSummary(t *testing.T) {
	art := validClosedIssueArtifact()
	art.Frontmatter["issue"].(map[string]interface{})["status"] = "ready"
	delete(art.Frontmatter["issue"].(map[string]interface{}), "closed")
	art.Frontmatter["implementation"] = map[string]interface{}{
		"package": "pkg/artifact",
	}
	result := validate.Issue(art, issueSchema())
	assertHasViolation(t, result, "issue/implementation-summary-required")
}

func TestIssue_Ready_MissingPackage(t *testing.T) {
	art := validClosedIssueArtifact()
	art.Frontmatter["issue"].(map[string]interface{})["status"] = "ready"
	delete(art.Frontmatter["issue"].(map[string]interface{}), "closed")
	art.Frontmatter["implementation"] = map[string]interface{}{
		"summary": "Fix the thing",
	}
	result := validate.Issue(art, issueSchema())
	assertHasViolation(t, result, "issue/implementation-package-required")
}

func TestIssue_Open_ImplementationNotRequired(t *testing.T) {
	art := validIssueArtifact()
	result := validate.Issue(art, issueSchema())
	assertNoViolationRule(t, result, "issue/implementation-required")
}
