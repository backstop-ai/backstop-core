package validate_test

import (
	"testing"

	"github.com/bmanson/backstop-core/pkg/artifact"
	"github.com/bmanson/backstop-core/pkg/schema"
	"github.com/bmanson/backstop-core/pkg/validate"
)

func bundleSchema() *schema.Schema {
	return &schema.Schema{
		ArtifactType:     "bundle",
		FilenamePattern:  `^[a-z0-9-]+(\.epic)?\.bundle\.md$`,
		RequiredMetadata: []string{"title", "schema_version"},
		RequiredSections: []string{},
		StatusEnum:       []string{},
	}
}

func validBundleArtifact() *artifact.ParsedArtifact {
	return &artifact.ParsedArtifact{
		Filename: "my-feature.bundle.md",
		Title:    "My Feature",
		Metadata: map[string]string{
			"schema_version": "bundle/v1",
		},
		Frontmatter: map[string]interface{}{
			"schema_version": "bundle/v1",
			"bundle": map[string]interface{}{
				"name":     "my-feature",
				"version":  "0.1.0",
				"created":  "2026-03-19",
				"category": "feature",
			},
			"status": map[string]interface{}{
				"maturity": "idea",
			},
		},
		Sections: []string{},
	}
}

func validReadyBundleArtifact() *artifact.ParsedArtifact {
	return &artifact.ParsedArtifact{
		Filename: "my-feature.bundle.md",
		Title:    "My Feature",
		Metadata: map[string]string{
			"schema_version": "bundle/v1",
		},
		Frontmatter: map[string]interface{}{
			"schema_version": "bundle/v1",
			"bundle": map[string]interface{}{
				"name":     "my-feature",
				"version":  "1.0.0",
				"created":  "2026-03-01",
				"updated":  "2026-03-19",
				"category": "feature",
			},
			"status": map[string]interface{}{
				"maturity": "ready",
			},
			"problem": map[string]interface{}{
				"summary":          "A real problem summary",
				"user_story":       "As a user, I want X",
				"success_criteria": []interface{}{"criterion-1"},
			},
			"solution": map[string]interface{}{
				"approach":    "Build it this way",
				"assumptions": []interface{}{"assumption-1"},
			},
			"requirements": []interface{}{
				map[string]interface{}{
					"id":   "REQ-001",
					"text": "System must support feature X",
				},
				map[string]interface{}{
					"id":   "REQ-002",
					"text": "Feature X must handle edge case Y",
				},
			},
		},
		Sections: []string{
			"Current Thinking", "Draft Requirements",
			"Draft Design Decisions", "Spec Seeds", "Version History",
		},
	}
}

func validEpicBundleArtifact() *artifact.ParsedArtifact {
	return &artifact.ParsedArtifact{
		Filename: "ci-enforcement.epic.bundle.md",
		Title:    "CI Enforcement Epic",
		Metadata: map[string]string{
			"schema_version": "bundle/v1",
		},
		Frontmatter: map[string]interface{}{
			"schema_version": "bundle/v1",
			"bundle": map[string]interface{}{
				"name":     "ci-enforcement",
				"version":  "0.1.0",
				"created":  "2026-03-10",
				"category": "epic",
			},
			"status": map[string]interface{}{
				"maturity": "exploring",
			},
			"epic": map[string]interface{}{
				"id":             "EPIC-CI-ENFORCEMENT",
				"goal":           "Automated CI enforcement for all repos",
				"success_metric": "All PRs validated in < 60s",
				"children":       []interface{}{"bundle-a", "bundle-b"},
			},
		},
		Sections: []string{},
	}
}

// --- Core validation tests ---

func TestValidateBundle_Valid_Idea(t *testing.T) {
	result := validate.ValidateBundle(validBundleArtifact(), bundleSchema())
	if !result.Pass() {
		t.Errorf("expected valid idea bundle to pass, got %d violations:", len(result.Violations))
		for _, v := range result.Violations {
			t.Errorf("  [%s] %s", v.Rule, v.Message)
		}
	}
}

func TestValidateBundle_Valid_Ready(t *testing.T) {
	result := validate.ValidateBundle(validReadyBundleArtifact(), bundleSchema())
	if !result.Pass() {
		t.Errorf("expected valid ready bundle to pass, got %d violations:", len(result.Violations))
		for _, v := range result.Violations {
			t.Errorf("  [%s] %s", v.Rule, v.Message)
		}
	}
}

func TestValidateBundle_Valid_Epic(t *testing.T) {
	result := validate.ValidateBundle(validEpicBundleArtifact(), bundleSchema())
	if !result.Pass() {
		t.Errorf("expected valid epic bundle to pass, got %d violations:", len(result.Violations))
		for _, v := range result.Violations {
			t.Errorf("  [%s] %s", v.Rule, v.Message)
		}
	}
}

// --- Filename tests ---

func TestValidateBundle_BadFilename(t *testing.T) {
	art := validBundleArtifact()
	art.Filename = "My Feature.bundle.md"
	result := validate.ValidateBundle(art, bundleSchema())
	assertHasViolation(t, result, "bundle/filename-pattern")
}

func TestValidateBundle_EpicFilename(t *testing.T) {
	art := validBundleArtifact()
	art.Filename = "my-feature.epic.bundle.md"
	fm := art.Frontmatter["bundle"].(map[string]interface{})
	fm["category"] = "epic"
	art.Frontmatter["epic"] = map[string]interface{}{
		"id":             "EPIC-MY-FEATURE",
		"goal":           "Goal",
		"success_metric": "Metric",
		"children":       []interface{}{"child-a"},
	}
	result := validate.ValidateBundle(art, bundleSchema())
	assertNoViolationRule(t, result, "bundle/filename-pattern")
}

// --- Schema version mismatch ---

func TestValidateBundle_SchemaVersionMismatch(t *testing.T) {
	art := validBundleArtifact()
	art.Metadata["schema_version"] = "spec/v1"
	result := validate.ValidateBundle(art, bundleSchema())
	assertHasViolation(t, result, "bundle/schema-version-mismatch")
}

// --- Bundle block tests ---

func TestValidateBundle_MissingBundleBlock(t *testing.T) {
	art := validBundleArtifact()
	delete(art.Frontmatter, "bundle")
	result := validate.ValidateBundle(art, bundleSchema())
	assertHasViolation(t, result, "bundle/block-required")
}

func TestValidateBundle_BundleBlockNotMap(t *testing.T) {
	art := validBundleArtifact()
	art.Frontmatter["bundle"] = "not-a-map"
	result := validate.ValidateBundle(art, bundleSchema())
	assertHasViolation(t, result, "bundle/block-required")
}

func TestValidateBundle_MissingName(t *testing.T) {
	art := validBundleArtifact()
	delete(art.Frontmatter["bundle"].(map[string]interface{}), "name")
	result := validate.ValidateBundle(art, bundleSchema())
	assertHasViolation(t, result, "bundle/name-required")
}

func TestValidateBundle_BadNamePattern(t *testing.T) {
	art := validBundleArtifact()
	art.Frontmatter["bundle"].(map[string]interface{})["name"] = "My Feature"
	result := validate.ValidateBundle(art, bundleSchema())
	assertHasViolation(t, result, "bundle/name-pattern")
}

func TestValidateBundle_MissingVersion(t *testing.T) {
	art := validBundleArtifact()
	delete(art.Frontmatter["bundle"].(map[string]interface{}), "version")
	result := validate.ValidateBundle(art, bundleSchema())
	assertHasViolation(t, result, "bundle/version-required")
}

func TestValidateBundle_BadVersionPattern(t *testing.T) {
	art := validBundleArtifact()
	art.Frontmatter["bundle"].(map[string]interface{})["version"] = "v1.0"
	result := validate.ValidateBundle(art, bundleSchema())
	assertHasViolation(t, result, "bundle/version-pattern")
}

func TestValidateBundle_MissingCreated(t *testing.T) {
	art := validBundleArtifact()
	delete(art.Frontmatter["bundle"].(map[string]interface{}), "created")
	result := validate.ValidateBundle(art, bundleSchema())
	assertHasViolation(t, result, "bundle/created-required")
}

func TestValidateBundle_BadCreatedPattern(t *testing.T) {
	art := validBundleArtifact()
	art.Frontmatter["bundle"].(map[string]interface{})["created"] = "March 19"
	result := validate.ValidateBundle(art, bundleSchema())
	assertHasViolation(t, result, "bundle/created-pattern")
}

func TestValidateBundle_MissingCategory(t *testing.T) {
	art := validBundleArtifact()
	delete(art.Frontmatter["bundle"].(map[string]interface{}), "category")
	result := validate.ValidateBundle(art, bundleSchema())
	assertHasViolation(t, result, "bundle/category-required")
}

func TestValidateBundle_BadCategory(t *testing.T) {
	art := validBundleArtifact()
	art.Frontmatter["bundle"].(map[string]interface{})["category"] = "widget"
	result := validate.ValidateBundle(art, bundleSchema())
	assertHasViolation(t, result, "bundle/category-enum")
}

// --- Status block tests ---

func TestValidateBundle_MissingStatusBlock(t *testing.T) {
	art := validBundleArtifact()
	delete(art.Frontmatter, "status")
	result := validate.ValidateBundle(art, bundleSchema())
	assertHasViolation(t, result, "bundle/status-required")
}

func TestValidateBundle_StatusBlockNotMap(t *testing.T) {
	art := validBundleArtifact()
	art.Frontmatter["status"] = "ready"
	result := validate.ValidateBundle(art, bundleSchema())
	assertHasViolation(t, result, "bundle/status-required")
}

func TestValidateBundle_MissingMaturity(t *testing.T) {
	art := validBundleArtifact()
	art.Frontmatter["status"] = map[string]interface{}{}
	result := validate.ValidateBundle(art, bundleSchema())
	assertHasViolation(t, result, "bundle/maturity-required")
}

func TestValidateBundle_BadMaturity(t *testing.T) {
	art := validBundleArtifact()
	art.Frontmatter["status"] = map[string]interface{}{"maturity": "done"}
	result := validate.ValidateBundle(art, bundleSchema())
	assertHasViolation(t, result, "bundle/maturity-enum")
}

// --- Name/filename consistency ---

func TestValidateBundle_NameFilenameMismatch(t *testing.T) {
	art := validBundleArtifact()
	art.Frontmatter["bundle"].(map[string]interface{})["name"] = "other-name"
	result := validate.ValidateBundle(art, bundleSchema())
	assertHasViolation(t, result, "bundle/name-filename-mismatch")
}

func TestValidateBundle_EpicNameFilenameConsistency(t *testing.T) {
	art := validEpicBundleArtifact()
	// Filename is ci-enforcement.epic.bundle.md, name is ci-enforcement — should match
	result := validate.ValidateBundle(art, bundleSchema())
	assertNoViolationRule(t, result, "bundle/name-filename-mismatch")
}

// --- Version-gated updated ---

func TestValidateBundle_UpdatedNotRequired_AtInitial(t *testing.T) {
	art := validBundleArtifact()
	// version 0.1.0 — updated not required
	result := validate.ValidateBundle(art, bundleSchema())
	assertNoViolationRule(t, result, "bundle/updated-required")
}

func TestValidateBundle_UpdatedRequired_BeyondInitial(t *testing.T) {
	art := validBundleArtifact()
	art.Frontmatter["bundle"].(map[string]interface{})["version"] = "0.2.0"
	result := validate.ValidateBundle(art, bundleSchema())
	assertHasViolation(t, result, "bundle/updated-required")
}

func TestValidateBundle_UpdatedPresent_BeyondInitial(t *testing.T) {
	art := validBundleArtifact()
	fm := art.Frontmatter["bundle"].(map[string]interface{})
	fm["version"] = "0.2.0"
	fm["updated"] = "2026-03-19"
	result := validate.ValidateBundle(art, bundleSchema())
	assertNoViolationRule(t, result, "bundle/updated-required")
}

// --- Maturity-gated tests ---

func TestValidateBundle_Defined_MissingProblemSummary(t *testing.T) {
	art := validReadyBundleArtifact()
	art.Frontmatter["status"] = map[string]interface{}{"maturity": "defined"}
	delete(art.Frontmatter["problem"].(map[string]interface{}), "summary")
	result := validate.ValidateBundle(art, bundleSchema())
	assertHasViolationContaining(t, result, "bundle/maturity-gate", "problem.summary")
}

func TestValidateBundle_Defined_MissingSolutionApproach(t *testing.T) {
	art := validReadyBundleArtifact()
	art.Frontmatter["status"] = map[string]interface{}{"maturity": "defined"}
	delete(art.Frontmatter["solution"].(map[string]interface{}), "approach")
	result := validate.ValidateBundle(art, bundleSchema())
	assertHasViolationContaining(t, result, "bundle/maturity-gate", "solution.approach")
}

func TestValidateBundle_Ready_MissingSuccessCriteria(t *testing.T) {
	art := validReadyBundleArtifact()
	delete(art.Frontmatter["problem"].(map[string]interface{}), "success_criteria")
	result := validate.ValidateBundle(art, bundleSchema())
	assertHasViolationContaining(t, result, "bundle/maturity-gate", "problem.success_criteria")
}

func TestValidateBundle_Ready_MissingAssumptions(t *testing.T) {
	art := validReadyBundleArtifact()
	delete(art.Frontmatter["solution"].(map[string]interface{}), "assumptions")
	result := validate.ValidateBundle(art, bundleSchema())
	assertHasViolationContaining(t, result, "bundle/maturity-gate", "solution.assumptions")
}

func TestValidateBundle_Ready_MissingSections(t *testing.T) {
	art := validReadyBundleArtifact()
	art.Sections = []string{"Current Thinking"}
	result := validate.ValidateBundle(art, bundleSchema())
	assertHasViolationContaining(t, result, "bundle/maturity-section", "Draft Requirements")
}

func TestValidateBundle_Idea_NoMaturityGates(t *testing.T) {
	art := validBundleArtifact()
	// idea maturity — no gated requirements
	result := validate.ValidateBundle(art, bundleSchema())
	assertNoViolationRule(t, result, "bundle/maturity-gate")
	assertNoViolationRule(t, result, "bundle/maturity-section")
}

func TestValidateBundle_Exploring_NoMaturityGates(t *testing.T) {
	art := validBundleArtifact()
	art.Frontmatter["status"] = map[string]interface{}{"maturity": "exploring"}
	result := validate.ValidateBundle(art, bundleSchema())
	assertNoViolationRule(t, result, "bundle/maturity-gate")
	assertNoViolationRule(t, result, "bundle/maturity-section")
}

// --- Epic validation tests ---

func TestValidateBundle_EpicCategory_MissingEpicBlock(t *testing.T) {
	art := validBundleArtifact()
	art.Frontmatter["bundle"].(map[string]interface{})["category"] = "epic"
	result := validate.ValidateBundle(art, bundleSchema())
	assertHasViolation(t, result, "bundle/epic-required")
}

func TestValidateBundle_EpicBlockNotMap(t *testing.T) {
	art := validBundleArtifact()
	art.Frontmatter["bundle"].(map[string]interface{})["category"] = "epic"
	art.Frontmatter["epic"] = "not-a-map"
	result := validate.ValidateBundle(art, bundleSchema())
	assertHasViolation(t, result, "bundle/epic-required")
}

func TestValidateBundle_EpicMissingID(t *testing.T) {
	art := validEpicBundleArtifact()
	delete(art.Frontmatter["epic"].(map[string]interface{}), "id")
	result := validate.ValidateBundle(art, bundleSchema())
	assertHasViolation(t, result, "bundle/epic-id-required")
}

func TestValidateBundle_EpicBadIDPattern(t *testing.T) {
	art := validEpicBundleArtifact()
	art.Frontmatter["epic"].(map[string]interface{})["id"] = "epic-lowercase"
	result := validate.ValidateBundle(art, bundleSchema())
	assertHasViolation(t, result, "bundle/epic-id-pattern")
}

func TestValidateBundle_EpicMissingGoal(t *testing.T) {
	art := validEpicBundleArtifact()
	delete(art.Frontmatter["epic"].(map[string]interface{}), "goal")
	result := validate.ValidateBundle(art, bundleSchema())
	assertHasViolation(t, result, "bundle/epic-goal-required")
}

func TestValidateBundle_EpicMissingSuccessMetric(t *testing.T) {
	art := validEpicBundleArtifact()
	delete(art.Frontmatter["epic"].(map[string]interface{}), "success_metric")
	result := validate.ValidateBundle(art, bundleSchema())
	assertHasViolation(t, result, "bundle/epic-success-metric-required")
}

func TestValidateBundle_EpicMissingChildren(t *testing.T) {
	art := validEpicBundleArtifact()
	delete(art.Frontmatter["epic"].(map[string]interface{}), "children")
	result := validate.ValidateBundle(art, bundleSchema())
	assertHasViolation(t, result, "bundle/epic-children-required")
}

func TestValidateBundle_EpicEmptyChildren(t *testing.T) {
	art := validEpicBundleArtifact()
	art.Frontmatter["epic"].(map[string]interface{})["children"] = []interface{}{}
	result := validate.ValidateBundle(art, bundleSchema())
	assertHasViolation(t, result, "bundle/epic-children-empty")
}

func TestValidateBundle_NonEpicCategory_NoEpicValidation(t *testing.T) {
	art := validBundleArtifact()
	// category is "feature" — no epic validation
	result := validate.ValidateBundle(art, bundleSchema())
	assertNoViolationRule(t, result, "bundle/epic-required")
}

// --- Placeholder ban tests ---

func TestValidateBundle_PlaceholderInSummary_Ready(t *testing.T) {
	art := validReadyBundleArtifact()
	art.Frontmatter["problem"].(map[string]interface{})["summary"] = "TBD — need to define this"
	result := validate.ValidateBundle(art, bundleSchema())
	assertHasViolation(t, result, "bundle/placeholder-ban")
}

func TestValidateBundle_PlaceholderTODO(t *testing.T) {
	art := validReadyBundleArtifact()
	art.Frontmatter["problem"].(map[string]interface{})["summary"] = "TODO figure this out"
	result := validate.ValidateBundle(art, bundleSchema())
	assertHasViolation(t, result, "bundle/placeholder-ban")
}

func TestValidateBundle_PlaceholderQuestionMarks(t *testing.T) {
	art := validReadyBundleArtifact()
	art.Frontmatter["problem"].(map[string]interface{})["summary"] = "What is the problem???"
	result := validate.ValidateBundle(art, bundleSchema())
	assertHasViolation(t, result, "bundle/placeholder-ban")
}

func TestValidateBundle_PlaceholderNotChecked_AtIdea(t *testing.T) {
	art := validBundleArtifact()
	art.Frontmatter["problem"] = map[string]interface{}{
		"summary": "TBD placeholder is fine at idea stage",
	}
	result := validate.ValidateBundle(art, bundleSchema())
	assertNoViolationRule(t, result, "bundle/placeholder-ban")
}

func TestValidateBundle_PlaceholderFIXME(t *testing.T) {
	art := validReadyBundleArtifact()
	art.Frontmatter["problem"].(map[string]interface{})["summary"] = "FIXME this later"
	result := validate.ValidateBundle(art, bundleSchema())
	assertHasViolation(t, result, "bundle/placeholder-ban")
}

func TestValidateBundle_PlaceholderXXX(t *testing.T) {
	art := validReadyBundleArtifact()
	art.Frontmatter["problem"].(map[string]interface{})["summary"] = "XXX needs work"
	result := validate.ValidateBundle(art, bundleSchema())
	assertHasViolation(t, result, "bundle/placeholder-ban")
}

// --- Composition test ---

func TestValidateBundle_ComposesBaseAndBundle(t *testing.T) {
	art := validBundleArtifact()
	art.Title = "" // base violation
	delete(art.Frontmatter["bundle"].(map[string]interface{}), "name") // bundle violation
	result := validate.ValidateBundle(art, bundleSchema())
	assertHasViolation(t, result, "base/title-required")
	assertHasViolation(t, result, "bundle/name-required")
}

// --- Maturity-gated: problem block missing entirely ---

func TestValidateBundle_Defined_MissingProblemBlock(t *testing.T) {
	art := validReadyBundleArtifact()
	art.Frontmatter["status"] = map[string]interface{}{"maturity": "defined"}
	delete(art.Frontmatter, "problem")
	result := validate.ValidateBundle(art, bundleSchema())
	assertHasViolationContaining(t, result, "bundle/maturity-gate", "problem.summary")
	assertHasViolationContaining(t, result, "bundle/maturity-gate", "problem.user_story")
}

func TestValidateBundle_Defined_MissingSolutionBlock(t *testing.T) {
	art := validReadyBundleArtifact()
	art.Frontmatter["status"] = map[string]interface{}{"maturity": "defined"}
	delete(art.Frontmatter, "solution")
	result := validate.ValidateBundle(art, bundleSchema())
	assertHasViolationContaining(t, result, "bundle/maturity-gate", "solution.approach")
}

// --- Edge cases ---

func TestValidateBundle_EmptyFrontmatter(t *testing.T) {
	art := &artifact.ParsedArtifact{
		Filename:    "empty.bundle.md",
		Title:       "Empty",
		Metadata:    map[string]string{"schema_version": "bundle/v1"},
		Frontmatter: map[string]interface{}{"schema_version": "bundle/v1"},
		Sections:    []string{},
	}
	result := validate.ValidateBundle(art, bundleSchema())
	assertHasViolation(t, result, "bundle/block-required")
	assertHasViolation(t, result, "bundle/status-required")
}

func TestValidateBundle_DefinedWithAllSections(t *testing.T) {
	art := validReadyBundleArtifact()
	art.Frontmatter["status"] = map[string]interface{}{"maturity": "defined"}
	// Remove ready-only requirements
	delete(art.Frontmatter["problem"].(map[string]interface{}), "success_criteria")
	delete(art.Frontmatter["solution"].(map[string]interface{}), "assumptions")
	result := validate.ValidateBundle(art, bundleSchema())
	assertNoViolationRule(t, result, "bundle/maturity-gate")
	assertNoViolationRule(t, result, "bundle/maturity-section")
}

// --- Helpers ---

func assertHasViolation(t *testing.T, result validate.ValidationResult, rule string) {
	t.Helper()
	for _, v := range result.Violations {
		if v.Rule == rule {
			return
		}
	}
	t.Errorf("expected violation with rule '%s', got none. Violations:", rule)
	for _, v := range result.Violations {
		t.Errorf("  [%s] %s", v.Rule, v.Message)
	}
}

func assertNoViolationRule(t *testing.T, result validate.ValidationResult, rule string) {
	t.Helper()
	for _, v := range result.Violations {
		if v.Rule == rule {
			t.Errorf("expected no violation with rule '%s', but found: %s", rule, v.Message)
			return
		}
	}
}

func assertHasViolationContaining(t *testing.T, result validate.ValidationResult, rule string, substring string) {
	t.Helper()
	for _, v := range result.Violations {
		if v.Rule == rule && contains(v.Message, substring) {
			return
		}
	}
	t.Errorf("expected violation with rule '%s' containing '%s', got none. Violations:", rule, substring)
	for _, v := range result.Violations {
		t.Errorf("  [%s] %s", v.Rule, v.Message)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && searchString(s, sub)
}

func searchString(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// --- Bundle requirements tests ---

func TestValidateBundle_Defined_RequirementsRequired(t *testing.T) {
art := validReadyBundleArtifact()
art.Frontmatter["status"].(map[string]interface{})["maturity"] = "defined"
delete(art.Frontmatter, "requirements")
result := validate.ValidateBundle(art, bundleSchema())
assertHasViolation(t, result, "bundle/requirements-required")
}

func TestValidateBundle_Ready_RequirementsRequired(t *testing.T) {
art := validReadyBundleArtifact()
delete(art.Frontmatter, "requirements")
result := validate.ValidateBundle(art, bundleSchema())
assertHasViolation(t, result, "bundle/requirements-required")
}

func TestValidateBundle_Idea_RequirementsOptional(t *testing.T) {
art := validBundleArtifact()
delete(art.Frontmatter, "requirements")
result := validate.ValidateBundle(art, bundleSchema())
assertNoViolationRule(t, result, "bundle/requirements-required")
}

func TestValidateBundle_Ready_EmptyRequirements(t *testing.T) {
art := validReadyBundleArtifact()
art.Frontmatter["requirements"] = []interface{}{}
result := validate.ValidateBundle(art, bundleSchema())
assertHasViolation(t, result, "bundle/requirements-required")
}

func TestValidateBundle_RequirementsNotArray(t *testing.T) {
art := validReadyBundleArtifact()
art.Frontmatter["requirements"] = "not-array"
result := validate.ValidateBundle(art, bundleSchema())
assertHasViolation(t, result, "bundle/requirements-format")
}

func TestValidateBundle_RequirementNotMap(t *testing.T) {
art := validReadyBundleArtifact()
art.Frontmatter["requirements"] = []interface{}{"not-a-map"}
result := validate.ValidateBundle(art, bundleSchema())
assertHasViolation(t, result, "bundle/requirement-format")
}

func TestValidateBundle_RequirementMissingID(t *testing.T) {
art := validReadyBundleArtifact()
art.Frontmatter["requirements"] = []interface{}{
map[string]interface{}{"text": "Some requirement"},
}
result := validate.ValidateBundle(art, bundleSchema())
assertHasViolation(t, result, "bundle/requirement-id-required")
}

func TestValidateBundle_RequirementBadIDPattern(t *testing.T) {
art := validReadyBundleArtifact()
art.Frontmatter["requirements"] = []interface{}{
map[string]interface{}{"id": "R-001", "text": "Some requirement"},
}
result := validate.ValidateBundle(art, bundleSchema())
assertHasViolation(t, result, "bundle/requirement-id-pattern")
}

func TestValidateBundle_RequirementDuplicateID(t *testing.T) {
art := validReadyBundleArtifact()
art.Frontmatter["requirements"] = []interface{}{
map[string]interface{}{"id": "REQ-001", "text": "First"},
map[string]interface{}{"id": "REQ-001", "text": "Second"},
}
result := validate.ValidateBundle(art, bundleSchema())
assertHasViolation(t, result, "bundle/requirement-id-duplicate")
}

func TestValidateBundle_RequirementMissingText(t *testing.T) {
art := validReadyBundleArtifact()
art.Frontmatter["requirements"] = []interface{}{
map[string]interface{}{"id": "REQ-001"},
}
result := validate.ValidateBundle(art, bundleSchema())
assertHasViolation(t, result, "bundle/requirement-text-required")
}

func TestValidateBundle_Exploring_RequirementsOptional(t *testing.T) {
art := validBundleArtifact()
art.Frontmatter["status"].(map[string]interface{})["maturity"] = "exploring"
delete(art.Frontmatter, "requirements")
result := validate.ValidateBundle(art, bundleSchema())
assertNoViolationRule(t, result, "bundle/requirements-required")
}

func TestValidateBundle_Exploring_ValidIfPresent(t *testing.T) {
art := validBundleArtifact()
art.Frontmatter["status"].(map[string]interface{})["maturity"] = "exploring"
art.Frontmatter["requirements"] = []interface{}{
map[string]interface{}{"id": "REQ-001", "text": "Early requirement"},
}
result := validate.ValidateBundle(art, bundleSchema())
assertNoViolationRule(t, result, "bundle/requirement-id-pattern")
}
