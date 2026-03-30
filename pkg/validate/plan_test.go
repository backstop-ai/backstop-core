package validate_test

import (
	"testing"

	"github.com/bmanson/backstop-core/pkg/artifact"
	"github.com/bmanson/backstop-core/pkg/validate"
)

func validPlanArtifact() *artifact.ParsedArtifact {
	return &artifact.ParsedArtifact{
		Filename: "PLAN-SPEC-001-standards-compiler.plan.yml",
		Metadata: map[string]string{
			"plan_id": "PLAN-SPEC-001",
			"spec_id": "SPEC-001",
			"created": "2026-03-30",
			"status":  "draft",
		},
		Frontmatter: map[string]interface{}{
			"plan_id": "PLAN-SPEC-001",
			"spec_id": "SPEC-001",
			"created": "2026-03-30",
			"status":  "draft",
			"phases": []interface{}{
				map[string]interface{}{
					"id":   "phase-1",
					"name": "Core Types",
					"tasks": []interface{}{
						map[string]interface{}{
							"id":          "types-test",
							"title":       "Write type tests",
							"description": "Create tests for Rule, ManifestRule, etc.",
							"files":       []interface{}{"pkg/compile/types_test.go"},
							"claims":      []interface{}{"CLM-003"},
							"depends_on":  []interface{}{},
						},
						map[string]interface{}{
							"id":          "types-impl",
							"title":       "Implement core types",
							"description": "Create Rule, ManifestRule, CompileOptions, etc.",
							"files":       []interface{}{"pkg/compile/types.go"},
							"claims":      []interface{}{"CLM-003"},
							"depends_on":  []interface{}{"types-test"},
						},
					},
				},
			},
		},
		Sections: []string{},
	}
}

func TestPlan_FullyValid(t *testing.T) {
	result := validate.Plan(validPlanArtifact(), nil)
	if !result.Pass() {
		for _, v := range result.Violations {
			t.Errorf("[%s] %s: %s", v.Severity, v.Rule, v.Message)
		}
	}
}

// --- Filename ---

func TestPlan_InvalidFilename(t *testing.T) {
	art := validPlanArtifact()
	art.Filename = "bad-plan.yml"
	result := validate.Plan(art, nil)
	assertHasViolation(t, result, "plan/filename-pattern")
}

func TestPlan_MarkdownFilenameRejected(t *testing.T) {
	art := validPlanArtifact()
	art.Filename = "PLAN-SPEC-001-standards-compiler.plan.md"
	result := validate.Plan(art, nil)
	assertHasViolation(t, result, "plan/filename-pattern")
}

// --- plan_id ---

func TestPlan_MissingPlanID(t *testing.T) {
	art := validPlanArtifact()
	delete(art.Frontmatter, "plan_id")
	result := validate.Plan(art, nil)
	assertHasViolation(t, result, "plan/plan-id-required")
}

func TestPlan_InvalidPlanID(t *testing.T) {
	art := validPlanArtifact()
	art.Frontmatter["plan_id"] = "PLAN-001"
	result := validate.Plan(art, nil)
	assertHasViolation(t, result, "plan/plan-id-pattern")
}

func TestPlan_PlanIDFilenameMismatch(t *testing.T) {
	art := validPlanArtifact()
	art.Frontmatter["plan_id"] = "PLAN-SPEC-099"
	result := validate.Plan(art, nil)
	assertHasViolation(t, result, "plan/id-filename-mismatch")
}

// --- spec_id ---

func TestPlan_MissingSpecID(t *testing.T) {
	art := validPlanArtifact()
	delete(art.Frontmatter, "spec_id")
	result := validate.Plan(art, nil)
	assertHasViolation(t, result, "plan/spec-id-required")
}

func TestPlan_InvalidSpecID(t *testing.T) {
	art := validPlanArtifact()
	art.Frontmatter["spec_id"] = "PLAN-001"
	result := validate.Plan(art, nil)
	assertHasViolation(t, result, "plan/spec-id-pattern")
}

func TestPlan_SpecIDAcceptsIssue(t *testing.T) {
	art := validPlanArtifact()
	art.Frontmatter["spec_id"] = "ISSUE-042"
	result := validate.Plan(art, nil)
	assertNoViolationRule(t, result, "plan/spec-id-pattern")
	assertNoViolationRule(t, result, "plan/spec-id-required")
}

// --- status ---

func TestPlan_InvalidStatus(t *testing.T) {
	art := validPlanArtifact()
	art.Frontmatter["status"] = "abandoned"
	result := validate.Plan(art, nil)
	assertHasViolation(t, result, "plan/invalid-status")
}

func TestPlan_MissingStatus(t *testing.T) {
	art := validPlanArtifact()
	delete(art.Frontmatter, "status")
	result := validate.Plan(art, nil)
	assertHasViolation(t, result, "plan/status-required")
}

func TestPlan_StatusReady(t *testing.T) {
	art := validPlanArtifact()
	art.Frontmatter["status"] = "ready"
	result := validate.Plan(art, nil)
	assertNoViolationRule(t, result, "plan/invalid-status")
}

// --- created ---

func TestPlan_MissingCreated(t *testing.T) {
	art := validPlanArtifact()
	delete(art.Frontmatter, "created")
	result := validate.Plan(art, nil)
	assertHasViolation(t, result, "plan/created-required")
}

// --- phases ---

func TestPlan_MissingPhases(t *testing.T) {
	art := validPlanArtifact()
	delete(art.Frontmatter, "phases")
	result := validate.Plan(art, nil)
	assertHasViolation(t, result, "plan/phases-required")
}

func TestPlan_EmptyPhases(t *testing.T) {
	art := validPlanArtifact()
	art.Frontmatter["phases"] = []interface{}{}
	result := validate.Plan(art, nil)
	assertHasViolation(t, result, "plan/phases-empty")
}

func TestPlan_PhasesNotArray(t *testing.T) {
	art := validPlanArtifact()
	art.Frontmatter["phases"] = "not an array"
	result := validate.Plan(art, nil)
	assertHasViolation(t, result, "plan/phases-required")
}

func TestPlan_PhaseNotMap(t *testing.T) {
	art := validPlanArtifact()
	art.Frontmatter["phases"] = []interface{}{"not a map"}
	result := validate.Plan(art, nil)
	assertHasViolation(t, result, "plan/phase-format")
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
	result := validate.Plan(art, nil)
	assertHasViolation(t, result, "plan/phase-id-required")
}

func TestPlan_DuplicatePhaseID(t *testing.T) {
	art := validPlanArtifact()
	art.Frontmatter["phases"] = []interface{}{
		map[string]interface{}{
			"id": "p1", "name": "First",
			"tasks": []interface{}{
				map[string]interface{}{
					"id": "t1", "title": "task", "description": "desc",
					"files": []interface{}{"a.go"}, "claims": []interface{}{"CLM-001"},
					"depends_on": []interface{}{},
				},
			},
		},
		map[string]interface{}{
			"id": "p1", "name": "Duplicate",
			"tasks": []interface{}{
				map[string]interface{}{
					"id": "t2", "title": "task", "description": "desc",
					"files": []interface{}{"b.go"}, "claims": []interface{}{"CLM-002"},
					"depends_on": []interface{}{},
				},
			},
		},
	}
	result := validate.Plan(art, nil)
	assertHasViolation(t, result, "plan/phase-id-duplicate")
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
	result := validate.Plan(art, nil)
	assertHasViolation(t, result, "plan/task-description-required")
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
	result := validate.Plan(art, nil)
	assertHasViolation(t, result, "plan/task-files-required")
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
	result := validate.Plan(art, nil)
	assertHasViolation(t, result, "plan/task-files-empty")
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
	result := validate.Plan(art, nil)
	assertHasViolation(t, result, "plan/task-claims-required")
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
	result := validate.Plan(art, nil)
	assertHasViolation(t, result, "plan/task-depends-on-required")
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
	result := validate.Plan(art, nil)
	assertHasViolation(t, result, "plan/task-id-duplicate")
}

// D-081: File exclusivity

func TestPlan_FileExclusivity_ParallelConflict(t *testing.T) {
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
					"files": []interface{}{"shared.go"}, "claims": []interface{}{"CLM-002"},
					"depends_on": []interface{}{},
				},
			},
		},
	}
	result := validate.Plan(art, nil)
	assertHasViolation(t, result, "plan/file-exclusivity")
}

func TestPlan_FileExclusivity_SequentialOK(t *testing.T) {
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
					"files": []interface{}{"shared.go"}, "claims": []interface{}{"CLM-002"},
					"depends_on": []interface{}{"task-a"},
				},
			},
		},
	}
	result := validate.Plan(art, nil)
	assertNoViolationRule(t, result, "plan/file-exclusivity")
}

func TestPlan_FileExclusivity_TransitiveDep(t *testing.T) {
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
	result := validate.Plan(art, nil)
	assertNoViolationRule(t, result, "plan/file-exclusivity")
}

func TestPlan_FileExclusivity_DiamondDAG(t *testing.T) {
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
	result := validate.Plan(art, nil)
	assertHasViolation(t, result, "plan/file-exclusivity")
}

// --- Real file test ---

func TestPlan_RealPlanFile(t *testing.T) {
	art, err := artifact.ParseFile("../../plans/PLAN-SPEC-001-standards-compiler.plan.yml")
	if err != nil {
		t.Skipf("skipping real plan test — ParseFile: %v", err)
	}
	result := validate.Plan(art, nil)
	if !result.Pass() {
		for _, v := range result.Violations {
			t.Errorf("[%s] %s: %s", v.Severity, v.Rule, v.Message)
		}
	}
}

// Helpers assertHasViolation and assertNoViolationRule are in bundle_test.go

// --- F4: created date format ---

func TestPlan_CreatedBadFormat(t *testing.T) {
	art := validPlanArtifact()
	art.Frontmatter["created"] = "not-a-date"
	result := validate.Plan(art, nil)
	assertHasViolation(t, result, "plan/created-format")
}

func TestPlan_CreatedValidFormat(t *testing.T) {
	art := validPlanArtifact()
	art.Frontmatter["created"] = "2026-03-30"
	result := validate.Plan(art, nil)
	assertNoViolationRule(t, result, "plan/created-format")
}

// --- F5: coverage_threshold range ---

func TestPlan_CoverageThresholdOutOfRange(t *testing.T) {
	art := validPlanArtifact()
	art.Frontmatter["coverage_threshold"] = 999
	result := validate.Plan(art, nil)
	assertHasViolation(t, result, "plan/coverage-threshold-range")
}

func TestPlan_CoverageThresholdNegative(t *testing.T) {
	art := validPlanArtifact()
	art.Frontmatter["coverage_threshold"] = -1
	result := validate.Plan(art, nil)
	assertHasViolation(t, result, "plan/coverage-threshold-range")
}

func TestPlan_CoverageThresholdValid(t *testing.T) {
	art := validPlanArtifact()
	art.Frontmatter["coverage_threshold"] = 90
	result := validate.Plan(art, nil)
	assertNoViolationRule(t, result, "plan/coverage-threshold-range")
}

func TestPlan_CoverageThresholdWrongType(t *testing.T) {
	art := validPlanArtifact()
	art.Frontmatter["coverage_threshold"] = "ninety"
	result := validate.Plan(art, nil)
	assertHasViolation(t, result, "plan/coverage-threshold-range")
}

// --- F7: dangling depends_on ---

func TestPlan_DanglingDependency(t *testing.T) {
	art := validPlanArtifact()
	art.Frontmatter["phases"] = []interface{}{
		map[string]interface{}{
			"id": "p1", "name": "Phase",
			"tasks": []interface{}{
				map[string]interface{}{
					"id": "t1", "title": "task", "description": "desc",
					"files": []interface{}{"a.go"}, "claims": []interface{}{"CLM-001"},
					"depends_on": []interface{}{"nonexistent-task"},
				},
			},
		},
	}
	result := validate.Plan(art, nil)
	assertHasViolation(t, result, "plan/unknown-dependency")
}

// --- F14: optional field type checks ---

func TestPlan_OptionalFieldWrongType(t *testing.T) {
	art := validPlanArtifact()
	art.Frontmatter["notes"] = 42
	result := validate.Plan(art, nil)
	assertHasViolation(t, result, "plan/field-type")
}

func TestPlan_OptionalFieldStringOK(t *testing.T) {
	art := validPlanArtifact()
	art.Frontmatter["notes"] = "some notes"
	result := validate.Plan(art, nil)
	assertNoViolationRule(t, result, "plan/field-type")
}

// --- F15: cycle detection ---

func TestPlan_DependencyCycle(t *testing.T) {
	art := validPlanArtifact()
	art.Frontmatter["phases"] = []interface{}{
		map[string]interface{}{
			"id": "p1", "name": "Phase",
			"tasks": []interface{}{
				map[string]interface{}{
					"id": "a", "title": "A", "description": "d",
					"files": []interface{}{"a.go"}, "claims": []interface{}{"CLM-001"},
					"depends_on": []interface{}{"b"},
				},
				map[string]interface{}{
					"id": "b", "title": "B", "description": "d",
					"files": []interface{}{"b.go"}, "claims": []interface{}{"CLM-002"},
					"depends_on": []interface{}{"a"},
				},
			},
		},
	}
	result := validate.Plan(art, nil)
	assertHasViolation(t, result, "plan/dependency-cycle")
}

func TestPlan_SelfDependency(t *testing.T) {
	art := validPlanArtifact()
	art.Frontmatter["phases"] = []interface{}{
		map[string]interface{}{
			"id": "p1", "name": "Phase",
			"tasks": []interface{}{
				map[string]interface{}{
					"id": "a", "title": "A", "description": "d",
					"files": []interface{}{"a.go"}, "claims": []interface{}{"CLM-001"},
					"depends_on": []interface{}{"a"},
				},
			},
		},
	}
	result := validate.Plan(art, nil)
	assertHasViolation(t, result, "plan/dependency-cycle")
}

// --- F16: empty claims and empty title ---

func TestPlan_TaskEmptyClaims(t *testing.T) {
	art := validPlanArtifact()
	art.Frontmatter["phases"] = []interface{}{
		map[string]interface{}{
			"id": "p1", "name": "Phase",
			"tasks": []interface{}{
				map[string]interface{}{
					"id": "t1", "title": "task", "description": "desc",
					"files": []interface{}{"f.go"}, "claims": []interface{}{},
					"depends_on": []interface{}{},
				},
			},
		},
	}
	result := validate.Plan(art, nil)
	assertHasViolation(t, result, "plan/task-claims-empty")
}

func TestPlan_TaskEmptyTitle(t *testing.T) {
	art := validPlanArtifact()
	art.Frontmatter["phases"] = []interface{}{
		map[string]interface{}{
			"id": "p1", "name": "Phase",
			"tasks": []interface{}{
				map[string]interface{}{
					"id": "t1", "title": "", "description": "desc",
					"files": []interface{}{"f.go"}, "claims": []interface{}{"CLM-001"},
					"depends_on": []interface{}{},
				},
			},
		},
	}
	result := validate.Plan(art, nil)
	assertHasViolation(t, result, "plan/task-title-required")
}
