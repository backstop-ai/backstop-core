package validate_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/bmanson/backstop-core/pkg/validate"
)

// REQ-009: Schema includes task type

func TestPlan_SchemaIncludesTaskType(t *testing.T) {
	data, err := os.ReadFile("../../artifacts/plan/v1/schema.json")
	if err != nil {
		t.Fatalf("cannot read schema: %v", err)
	}
	var schema map[string]interface{}
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatalf("cannot parse schema: %v", err)
	}

	// Check task_type_enum exists with all 6 values
	phases, ok := schema["phases"].(map[string]interface{})
	if !ok {
		t.Fatal("schema missing 'phases' object")
	}
	enumVal, ok := phases["task_type_enum"]
	if !ok {
		t.Fatal("schema phases missing 'task_type_enum'")
	}
	enumArr, ok := enumVal.([]interface{})
	if !ok {
		t.Fatalf("task_type_enum is not an array: %T", enumVal)
	}
	expected := map[string]bool{
		"setup": false, "test": false, "implementation": false,
		"verification": false, "refactor": false, "documentation": false,
	}
	for _, v := range enumArr {
		s, ok := v.(string)
		if !ok {
			continue
		}
		expected[s] = true
	}
	for k, found := range expected {
		if !found {
			t.Errorf("task_type_enum missing '%s'", k)
		}
	}

	// Check task_required_keys includes "type"
	reqKeys, ok := phases["task_required_keys"]
	if !ok {
		t.Fatal("schema phases missing 'task_required_keys'")
	}
	reqArr, ok := reqKeys.([]interface{})
	if !ok {
		t.Fatalf("task_required_keys is not an array: %T", reqKeys)
	}
	hasType := false
	for _, v := range reqArr {
		if s, ok := v.(string); ok && s == "type" {
			hasType = true
		}
	}
	if !hasType {
		t.Error("task_required_keys does not include 'type'")
	}
}

// REQ-008: Existing rules still work

func TestPlan_ExistingRules_PlanIDPattern(t *testing.T) {
	art := validTypedPlanArtifact()
	art.Frontmatter["plan_id"] = "BAD-ID"
	result := validate.Plan(art, nil)
	assertHasViolation(t, result, "plan/plan-id-pattern")
}

func TestPlan_ExistingRules_FileExclusivity(t *testing.T) {
	art := validTypedPlanArtifact()
	art.Frontmatter["phases"] = []interface{}{
		makePhase("phase-1", "Phase",
			makeTask("setup-1", "setup", []string{"pkg/shared.go"}, []string{"CLM-001"}, []string{}),
			makeTask("test-1", "test", []string{"pkg/foo_test.go"}, []string{"CLM-002"}, []string{"setup-1"}),
			makeTask("impl-a", "implementation", []string{"pkg/shared.go"}, []string{"CLM-002"}, []string{"test-1"}),
			makeTask("impl-b", "implementation", []string{"pkg/shared.go"}, []string{"CLM-003"}, []string{"test-1"}),
			makeTask("verify-1", "verification", []string{"pkg/foo_test.go"}, []string{"CLM-004"}, []string{"impl-a", "impl-b"}),
		),
	}
	result := validate.Plan(art, nil)
	assertHasViolation(t, result, "plan/file-exclusivity")
}

func TestPlan_ExistingRules_CycleDetection(t *testing.T) {
	art := validTypedPlanArtifact()
	art.Frontmatter["phases"] = []interface{}{
		makePhase("phase-1", "Phase",
			makeTask("test-a", "test", []string{"pkg/a_test.go"}, []string{"CLM-001"}, []string{"test-b"}),
			makeTask("test-b", "test", []string{"pkg/b_test.go"}, []string{"CLM-002"}, []string{"test-a"}),
			makeTask("impl-1", "implementation", []string{"pkg/a.go"}, []string{"CLM-001"}, []string{"test-a"}),
			makeTask("verify-1", "verification", []string{"pkg/a_test.go"}, []string{"CLM-003"}, []string{"impl-1"}),
		),
	}
	result := validate.Plan(art, nil)
	assertHasViolation(t, result, "plan/dependency-cycle")
}

// --- Defensive branch coverage for validatePhases ---
// These exercise pre-existing branches that typed-task tests don't reach.

func TestPlan_PhaseMissingName_Typed(t *testing.T) {
	art := validTypedPlanArtifact()
	phases := art.Frontmatter["phases"].([]interface{})
	phase := phases[0].(map[string]interface{})
	delete(phase, "name")
	result := validate.Plan(art, nil)
	assertHasViolation(t, result, "plan/phase-name-required")
}

func TestPlan_PhaseEmptyName_Typed(t *testing.T) {
	art := validTypedPlanArtifact()
	phases := art.Frontmatter["phases"].([]interface{})
	phase := phases[0].(map[string]interface{})
	phase["name"] = "   "
	result := validate.Plan(art, nil)
	assertHasViolation(t, result, "plan/phase-name-required")
}

func TestPlan_PhaseMissingTasks_Typed(t *testing.T) {
	art := validTypedPlanArtifact()
	art.Frontmatter["phases"] = []interface{}{
		map[string]interface{}{"id": "phase-1", "name": "No Tasks"},
	}
	result := validate.Plan(art, nil)
	assertHasViolation(t, result, "plan/phase-tasks-required")
}

func TestPlan_PhaseEmptyTasks_Typed(t *testing.T) {
	art := validTypedPlanArtifact()
	art.Frontmatter["phases"] = []interface{}{
		map[string]interface{}{"id": "phase-1", "name": "Empty", "tasks": []interface{}{}},
	}
	result := validate.Plan(art, nil)
	assertHasViolation(t, result, "plan/phase-tasks-empty")
}

func TestPlan_TaskNotMap_Typed(t *testing.T) {
	art := validTypedPlanArtifact()
	art.Frontmatter["phases"] = []interface{}{
		map[string]interface{}{
			"id": "phase-1", "name": "Bad Task",
			"tasks": []interface{}{"not a map"},
		},
	}
	result := validate.Plan(art, nil)
	assertHasViolation(t, result, "plan/task-format")
}

func TestPlan_TaskMissingID_Typed(t *testing.T) {
	art := validTypedPlanArtifact()
	art.Frontmatter["phases"] = []interface{}{
		makePhase("phase-1", "Phase",
			map[string]interface{}{
				"type": "setup", "title": "No ID", "description": "desc",
				"files": []interface{}{"a.go"}, "claims": []interface{}{"CLM-001"},
				"depends_on": []interface{}{},
			},
		),
	}
	result := validate.Plan(art, nil)
	assertHasViolation(t, result, "plan/task-id-required")
}

func TestPlan_TaskMissingTitle_Typed(t *testing.T) {
	art := validTypedPlanArtifact()
	art.Frontmatter["phases"] = []interface{}{
		makePhase("phase-1", "Phase",
			map[string]interface{}{
				"id": "t1", "type": "setup", "description": "desc",
				"files": []interface{}{"a.go"}, "claims": []interface{}{"CLM-001"},
				"depends_on": []interface{}{},
			},
		),
	}
	result := validate.Plan(art, nil)
	assertHasViolation(t, result, "plan/task-title-required")
}

func TestPlan_TaskEmptyDescription_Typed(t *testing.T) {
	art := validTypedPlanArtifact()
	art.Frontmatter["phases"] = []interface{}{
		makePhase("phase-1", "Phase",
			map[string]interface{}{
				"id": "t1", "type": "setup", "title": "T", "description": "   ",
				"files": []interface{}{"a.go"}, "claims": []interface{}{"CLM-001"},
				"depends_on": []interface{}{},
			},
		),
	}
	result := validate.Plan(art, nil)
	assertHasViolation(t, result, "plan/task-description-required")
}
