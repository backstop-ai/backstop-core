package validate_test

import (
	"testing"

	"github.com/bmanson/backstop-core/pkg/artifact"
	"github.com/bmanson/backstop-core/pkg/schema"
	"github.com/bmanson/backstop-core/pkg/validate"
)

func planSchema() *schema.Schema {
	return &schema.Schema{
		ArtifactType:     "plan",
		FilenamePattern:  `^PLAN-[A-Z]+-[0-9]+-[a-z][a-z0-9]*(-[a-z0-9]+)*\.plan\.md$`,
		RequiredMetadata: []string{"title", "number", "created", "status", "schema_version"},
		RequiredSections: []string{"Overview"},
		StatusEnum:       []string{"draft", "ready", "implementing", "completed"},
	}
}

func validPlanArtifact() *artifact.ParsedArtifact {
	return &artifact.ParsedArtifact{
		Filename: "PLAN-SPEC-023-backstop-base-validator.plan.md",
		Title:    "PLAN-SPEC-023: Backstop Base Validator Implementation",
		Metadata: map[string]string{
			"number":         "PLAN-SPEC-023",
			"created":        "2026-03-18",
			"status":         "ready",
			"schema_version": "plan/v1",
		},
		Frontmatter: map[string]interface{}{
			"number":         "PLAN-SPEC-023",
			"created":        "2026-03-18",
			"status":         "ready",
			"schema_version": "plan/v1",
			"implements":     "SPEC-023",
			"spec_reference": "SPEC-023",
			"phases": []interface{}{
				map[string]interface{}{
					"id":   "phase-1",
					"name": "Result Types",
					"tasks": []interface{}{
						map[string]interface{}{
							"id":          "result-types-test",
							"title":       "Write result type tests",
							"description": "Create tests for Violation and ValidationResult types",
							"files":       []interface{}{"pkg/validate/result_test.go"},
							"claims":      []interface{}{"CLM-018", "CLM-019"},
							"depends_on":  []interface{}{},
						},
						map[string]interface{}{
							"id":          "result-types-impl",
							"title":       "Implement result types",
							"description": "Create Violation struct and ValidationResult with Pass()",
							"files":       []interface{}{"pkg/validate/result.go"},
							"claims":      []interface{}{"CLM-018", "CLM-019"},
							"depends_on":  []interface{}{"result-types-test"},
						},
					},
				},
				map[string]interface{}{
					"id":   "phase-2",
					"name": "Parser",
					"tasks": []interface{}{
						map[string]interface{}{
							"id":          "parser-test",
							"title":       "Write parser tests",
							"description": "Create tests for Parse and ParseFile functions",
							"files":       []interface{}{"pkg/artifact/parse_test.go"},
							"claims":      []interface{}{"CLM-001", "CLM-002"},
							"depends_on":  []interface{}{"result-types-impl"},
						},
					},
				},
			},
		},
		Sections: []string{"Overview"},
	}
}

func TestPlan_FullyValid(t *testing.T) {
	result := validate.Plan(validPlanArtifact(), planSchema())
	if !result.Pass() {
		for _, v := range result.Violations {
			t.Errorf("[%s] %s: %s", v.Severity, v.Rule, v.Message)
		}
	}
}

func TestPlan_InvalidFilename(t *testing.T) {
	art := validPlanArtifact()
	art.Filename = "bad-plan.md"

	result := validate.Plan(art, planSchema())
	found := false
	for _, v := range result.Violations {
		if v.Rule == "plan/filename-pattern" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'plan/filename-pattern' violation, got: %v", result.Violations)
	}
}

func TestPlan_NumberMismatch(t *testing.T) {
	art := validPlanArtifact()
	art.Metadata["number"] = "PLAN-SPEC-999"

	result := validate.Plan(art, planSchema())
	found := false
	for _, v := range result.Violations {
		if v.Rule == "plan/number-mismatch" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'plan/number-mismatch' violation, got: %v", result.Violations)
	}
}

func TestPlan_InvalidStatus(t *testing.T) {
	art := validPlanArtifact()
	art.Metadata["status"] = "abandoned"

	result := validate.Plan(art, planSchema())
	found := false
	for _, v := range result.Violations {
		if v.Rule == "plan/invalid-status" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'plan/invalid-status' violation, got: %v", result.Violations)
	}
}

func TestPlan_MissingPhases(t *testing.T) {
	art := validPlanArtifact()
	delete(art.Frontmatter, "phases")

	result := validate.Plan(art, planSchema())
	found := false
	for _, v := range result.Violations {
		if v.Rule == "plan/phases-required" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'plan/phases-required' violation, got: %v", result.Violations)
	}
}

func TestPlan_EmptyPhases(t *testing.T) {
	art := validPlanArtifact()
	art.Frontmatter["phases"] = []interface{}{}

	result := validate.Plan(art, planSchema())
	found := false
	for _, v := range result.Violations {
		if v.Rule == "plan/phases-empty" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'plan/phases-empty' violation, got: %v", result.Violations)
	}
}

func TestPlan_PhaseMissingID(t *testing.T) {
	art := validPlanArtifact()
	art.Frontmatter["phases"] = []interface{}{
		map[string]interface{}{
			"name": "Test Phase",
			"tasks": []interface{}{
				map[string]interface{}{
					"id": "t1", "title": "task", "description": "desc",
					"files": []interface{}{"f.go"}, "claims": []interface{}{"CLM-001"},
					"depends_on": []interface{}{},
				},
			},
		},
	}

	result := validate.Plan(art, planSchema())
	found := false
	for _, v := range result.Violations {
		if v.Rule == "plan/phase-id-required" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'plan/phase-id-required' violation, got: %v", result.Violations)
	}
}

func TestPlan_DuplicatePhaseID(t *testing.T) {
	art := validPlanArtifact()
	art.Frontmatter["phases"] = []interface{}{
		map[string]interface{}{
			"id": "phase-1", "name": "First",
			"tasks": []interface{}{
				map[string]interface{}{
					"id": "t1", "title": "task", "description": "desc",
					"files": []interface{}{"a.go"}, "claims": []interface{}{"CLM-001"},
					"depends_on": []interface{}{},
				},
			},
		},
		map[string]interface{}{
			"id": "phase-1", "name": "Duplicate",
			"tasks": []interface{}{
				map[string]interface{}{
					"id": "t2", "title": "task", "description": "desc",
					"files": []interface{}{"b.go"}, "claims": []interface{}{"CLM-002"},
					"depends_on": []interface{}{},
				},
			},
		},
	}

	result := validate.Plan(art, planSchema())
	found := false
	for _, v := range result.Violations {
		if v.Rule == "plan/phase-id-duplicate" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'plan/phase-id-duplicate' violation, got: %v", result.Violations)
	}
}

// D-080: Agent-bounded task checks

func TestPlan_TaskMissingDescription(t *testing.T) {
	art := validPlanArtifact()
	art.Frontmatter["phases"] = []interface{}{
		map[string]interface{}{
			"id": "p1", "name": "Phase",
			"tasks": []interface{}{
				map[string]interface{}{
					"id": "t1", "title": "task",
					"files": []interface{}{"f.go"}, "claims": []interface{}{"CLM-001"},
					"depends_on": []interface{}{},
				},
			},
		},
	}

	result := validate.Plan(art, planSchema())
	found := false
	for _, v := range result.Violations {
		if v.Rule == "plan/task-description-required" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'plan/task-description-required' violation, got: %v", result.Violations)
	}
}

func TestPlan_TaskMissingFiles(t *testing.T) {
	art := validPlanArtifact()
	art.Frontmatter["phases"] = []interface{}{
		map[string]interface{}{
			"id": "p1", "name": "Phase",
			"tasks": []interface{}{
				map[string]interface{}{
					"id": "t1", "title": "task", "description": "desc",
					"claims": []interface{}{"CLM-001"}, "depends_on": []interface{}{},
				},
			},
		},
	}

	result := validate.Plan(art, planSchema())
	found := false
	for _, v := range result.Violations {
		if v.Rule == "plan/task-files-required" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'plan/task-files-required' violation, got: %v", result.Violations)
	}
}

func TestPlan_TaskEmptyFiles(t *testing.T) {
	art := validPlanArtifact()
	art.Frontmatter["phases"] = []interface{}{
		map[string]interface{}{
			"id": "p1", "name": "Phase",
			"tasks": []interface{}{
				map[string]interface{}{
					"id": "t1", "title": "task", "description": "desc",
					"files": []interface{}{}, "claims": []interface{}{"CLM-001"},
					"depends_on": []interface{}{},
				},
			},
		},
	}

	result := validate.Plan(art, planSchema())
	found := false
	for _, v := range result.Violations {
		if v.Rule == "plan/task-files-empty" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'plan/task-files-empty' violation, got: %v", result.Violations)
	}
}

func TestPlan_TaskMissingClaims(t *testing.T) {
	art := validPlanArtifact()
	art.Frontmatter["phases"] = []interface{}{
		map[string]interface{}{
			"id": "p1", "name": "Phase",
			"tasks": []interface{}{
				map[string]interface{}{
					"id": "t1", "title": "task", "description": "desc",
					"files": []interface{}{"f.go"}, "depends_on": []interface{}{},
				},
			},
		},
	}

	result := validate.Plan(art, planSchema())
	found := false
	for _, v := range result.Violations {
		if v.Rule == "plan/task-claims-required" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'plan/task-claims-required' violation, got: %v", result.Violations)
	}
}

func TestPlan_TaskMissingDependsOn(t *testing.T) {
	art := validPlanArtifact()
	art.Frontmatter["phases"] = []interface{}{
		map[string]interface{}{
			"id": "p1", "name": "Phase",
			"tasks": []interface{}{
				map[string]interface{}{
					"id": "t1", "title": "task", "description": "desc",
					"files": []interface{}{"f.go"}, "claims": []interface{}{"CLM-001"},
				},
			},
		},
	}

	result := validate.Plan(art, planSchema())
	found := false
	for _, v := range result.Violations {
		if v.Rule == "plan/task-depends-on-required" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'plan/task-depends-on-required' violation, got: %v", result.Violations)
	}
}

// D-081: File exclusivity tests

func TestPlan_FileExclusivity_ParallelConflict(t *testing.T) {
	// Two tasks with no dependency relationship touch the same file
	art := validPlanArtifact()
	art.Frontmatter["phases"] = []interface{}{
		map[string]interface{}{
			"id": "p1", "name": "Phase",
			"tasks": []interface{}{
				map[string]interface{}{
					"id": "task-a", "title": "Task A", "description": "desc",
					"files": []interface{}{"shared.go"}, "claims": []interface{}{"CLM-001"},
					"depends_on": []interface{}{},
				},
				map[string]interface{}{
					"id": "task-b", "title": "Task B", "description": "desc",
					"files": []interface{}{"shared.go"}, "claims": []interface{}{"CLM-002"},
					"depends_on": []interface{}{},
				},
			},
		},
	}

	result := validate.Plan(art, planSchema())
	found := false
	for _, v := range result.Violations {
		if v.Rule == "plan/file-exclusivity" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'plan/file-exclusivity' violation for parallel tasks sharing shared.go, got: %v", result.Violations)
	}
}

func TestPlan_FileExclusivity_SequentialOK(t *testing.T) {
	// Task B depends on Task A — they CAN share files (TDD pattern)
	art := validPlanArtifact()
	art.Frontmatter["phases"] = []interface{}{
		map[string]interface{}{
			"id": "p1", "name": "Phase",
			"tasks": []interface{}{
				map[string]interface{}{
					"id": "task-a", "title": "Task A", "description": "desc",
					"files": []interface{}{"shared.go"}, "claims": []interface{}{"CLM-001"},
					"depends_on": []interface{}{},
				},
				map[string]interface{}{
					"id": "task-b", "title": "Task B", "description": "desc",
					"files": []interface{}{"shared.go"}, "claims": []interface{}{"CLM-002"},
					"depends_on": []interface{}{"task-a"},
				},
			},
		},
	}

	result := validate.Plan(art, planSchema())
	for _, v := range result.Violations {
		if v.Rule == "plan/file-exclusivity" {
			t.Errorf("unexpected file-exclusivity violation for sequential tasks: %v", v)
		}
	}
}

func TestPlan_FileExclusivity_TransitiveDep(t *testing.T) {
	// A → B → C: A and C share a file, but C transitively depends on A — OK
	art := validPlanArtifact()
	art.Frontmatter["phases"] = []interface{}{
		map[string]interface{}{
			"id": "p1", "name": "Phase",
			"tasks": []interface{}{
				map[string]interface{}{
					"id": "task-a", "title": "A", "description": "d",
					"files": []interface{}{"shared.go"}, "claims": []interface{}{"CLM-001"},
					"depends_on": []interface{}{},
				},
				map[string]interface{}{
					"id": "task-b", "title": "B", "description": "d",
					"files": []interface{}{"other.go"}, "claims": []interface{}{"CLM-002"},
					"depends_on": []interface{}{"task-a"},
				},
				map[string]interface{}{
					"id": "task-c", "title": "C", "description": "d",
					"files": []interface{}{"shared.go"}, "claims": []interface{}{"CLM-003"},
					"depends_on": []interface{}{"task-b"},
				},
			},
		},
	}

	result := validate.Plan(art, planSchema())
	for _, v := range result.Violations {
		if v.Rule == "plan/file-exclusivity" {
			t.Errorf("unexpected file-exclusivity violation for transitively dependent tasks: %v", v)
		}
	}
}

func TestPlan_FileExclusivity_DiamondDAG(t *testing.T) {
	// Diamond: A → B, A → C, B → D, C → D
	// B and C are parallel-eligible — must not share files
	art := validPlanArtifact()
	art.Frontmatter["phases"] = []interface{}{
		map[string]interface{}{
			"id": "p1", "name": "Phase",
			"tasks": []interface{}{
				map[string]interface{}{
					"id": "a", "title": "A", "description": "d",
					"files": []interface{}{"a.go"}, "claims": []interface{}{"CLM-001"},
					"depends_on": []interface{}{},
				},
				map[string]interface{}{
					"id": "b", "title": "B", "description": "d",
					"files": []interface{}{"conflict.go"}, "claims": []interface{}{"CLM-002"},
					"depends_on": []interface{}{"a"},
				},
				map[string]interface{}{
					"id": "c", "title": "C", "description": "d",
					"files": []interface{}{"conflict.go"}, "claims": []interface{}{"CLM-003"},
					"depends_on": []interface{}{"a"},
				},
				map[string]interface{}{
					"id": "d", "title": "D", "description": "d",
					"files": []interface{}{"d.go"}, "claims": []interface{}{"CLM-004"},
					"depends_on": []interface{}{"b", "c"},
				},
			},
		},
	}

	result := validate.Plan(art, planSchema())
	found := false
	for _, v := range result.Violations {
		if v.Rule == "plan/file-exclusivity" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'plan/file-exclusivity' violation for diamond DAG parallel tasks B and C, got: %v", result.Violations)
	}
}

func TestPlan_ComposesBaseAndPlan(t *testing.T) {
	art := &artifact.ParsedArtifact{
		Filename:    "bad-name.md",
		Title:       "",
		Metadata:    map[string]string{},
		Frontmatter: map[string]interface{}{},
		Sections:    []string{},
	}

	result := validate.Plan(art, planSchema())
	hasBase := false
	hasPlan := false
	for _, v := range result.Violations {
		if v.Rule == "base/title-required" {
			hasBase = true
		}
		if v.Rule == "plan/filename-pattern" || v.Rule == "plan/phases-required" {
			hasPlan = true
		}
	}
	if !hasBase {
		t.Error("expected base violation (base/title-required)")
	}
	if !hasPlan {
		t.Error("expected plan violation")
	}
}

func TestPlan_PhasesNotAnArray(t *testing.T) {
	art := validPlanArtifact()
	art.Frontmatter["phases"] = "not an array"

	result := validate.Plan(art, planSchema())
	found := false
	for _, v := range result.Violations {
		if v.Rule == "plan/phases-required" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'plan/phases-required' violation, got: %v", result.Violations)
	}
}

func TestPlan_PhaseNotAMap(t *testing.T) {
	art := validPlanArtifact()
	art.Frontmatter["phases"] = []interface{}{"not a map"}

	result := validate.Plan(art, planSchema())
	found := false
	for _, v := range result.Violations {
		if v.Rule == "plan/phase-format" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'plan/phase-format' violation, got: %v", result.Violations)
	}
}

func TestPlan_TaskNotAMap(t *testing.T) {
	art := validPlanArtifact()
	art.Frontmatter["phases"] = []interface{}{
		map[string]interface{}{
			"id": "p1", "name": "Phase",
			"tasks": []interface{}{"not a map"},
		},
	}

	result := validate.Plan(art, planSchema())
	found := false
	for _, v := range result.Violations {
		if v.Rule == "plan/task-format" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'plan/task-format' violation, got: %v", result.Violations)
	}
}

func TestPlan_DuplicateTaskID(t *testing.T) {
	art := validPlanArtifact()
	art.Frontmatter["phases"] = []interface{}{
		map[string]interface{}{
			"id": "p1", "name": "Phase",
			"tasks": []interface{}{
				map[string]interface{}{
					"id": "dup", "title": "A", "description": "d",
					"files": []interface{}{"a.go"}, "claims": []interface{}{"CLM-001"},
					"depends_on": []interface{}{},
				},
				map[string]interface{}{
					"id": "dup", "title": "B", "description": "d",
					"files": []interface{}{"b.go"}, "claims": []interface{}{"CLM-002"},
					"depends_on": []interface{}{},
				},
			},
		},
	}

	result := validate.Plan(art, planSchema())
	found := false
	for _, v := range result.Violations {
		if v.Rule == "plan/task-id-duplicate" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'plan/task-id-duplicate' violation, got: %v", result.Violations)
	}
}

func TestPlan_SchemaVersionMismatch(t *testing.T) {
	art := validPlanArtifact()
	art.Metadata["schema_version"] = "adr/v2"

	result := validate.Plan(art, planSchema())
	found := false
	for _, v := range result.Violations {
		if v.Rule == "plan/schema-version-mismatch" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'plan/schema-version-mismatch' violation, got: %v", result.Violations)
	}
}

// --- Implements field tests ---

func TestPlan_MissingImplements(t *testing.T) {
	art := validPlanArtifact()
	delete(art.Frontmatter, "implements")
	result := validate.Plan(art, planSchema())
	found := false
	for _, v := range result.Violations {
		if v.Rule == "plan/implements-required" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'plan/implements-required' violation")
	}
}

func TestPlan_ImplementsEmpty(t *testing.T) {
	art := validPlanArtifact()
	art.Frontmatter["implements"] = ""
	result := validate.Plan(art, planSchema())
	found := false
	for _, v := range result.Violations {
		if v.Rule == "plan/implements-required" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'plan/implements-required' violation")
	}
}

func TestPlan_ImplementsBadPattern(t *testing.T) {
	art := validPlanArtifact()
	art.Frontmatter["implements"] = "PLAN-001"
	result := validate.Plan(art, planSchema())
	found := false
	for _, v := range result.Violations {
		if v.Rule == "plan/implements-pattern" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'plan/implements-pattern' violation")
	}
}

func TestPlan_ImplementsSpec(t *testing.T) {
	art := validPlanArtifact()
	art.Frontmatter["implements"] = "SPEC-023"
	result := validate.Plan(art, planSchema())
	for _, v := range result.Violations {
		if v.Rule == "plan/implements-pattern" || v.Rule == "plan/implements-required" {
			t.Errorf("unexpected violation: [%s] %s", v.Rule, v.Message)
		}
	}
}

func TestPlan_ImplementsIssue(t *testing.T) {
	art := validPlanArtifact()
	art.Frontmatter["implements"] = "ISSUE-042"
	result := validate.Plan(art, planSchema())
	for _, v := range result.Violations {
		if v.Rule == "plan/implements-pattern" || v.Rule == "plan/implements-required" {
			t.Errorf("unexpected violation: [%s] %s", v.Rule, v.Message)
		}
	}
}
