package validate_test

import (
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/validate"
)

// REQ-010: Test task dependency constraints
// Test tasks may only depend on setup, test, or verification tasks.
// Dependencies on implementation, refactor, or documentation are rejected.

func TestPlan_TestTask_DependsOnSetup(t *testing.T) {
	// test-1 already depends on setup-1 in validTypedPlanArtifact → should pass
	art := validTypedPlanArtifact()
	result := validate.Plan(art, nil)
	assertNoViolationRule(t, result, "plan/test-invalid-dependency")
}

func TestPlan_TestTask_DependsOnTest(t *testing.T) {
	art := validTypedPlanArtifact()
	phases := art.Frontmatter["phases"].([]interface{})
	phase := phases[0].(map[string]interface{})
	tasks := phase["tasks"].([]interface{})
	tasks = append(tasks,
		makeTask("test-2", "test", []string{"pkg/bar_test.go"}, []string{"CLM-010"}, []string{"test-1"}),
	)
	phase["tasks"] = tasks
	result := validate.Plan(art, nil)
	assertNoViolationRule(t, result, "plan/test-invalid-dependency")
}

func TestPlan_TestTask_DependsOnImplFails(t *testing.T) {
	art := validTypedPlanArtifact()
	phases := art.Frontmatter["phases"].([]interface{})
	phase := phases[0].(map[string]interface{})
	tasks := phase["tasks"].([]interface{})
	tasks = append(tasks,
		makeTask("test-bad", "test", []string{"pkg/bad_test.go"}, []string{"CLM-010"}, []string{"impl-1"}),
	)
	phase["tasks"] = tasks
	result := validate.Plan(art, nil)
	assertHasViolation(t, result, "plan/test-invalid-dependency")
}

func TestPlan_TestTask_DependsOnRefactorFails(t *testing.T) {
	art := validTypedPlanArtifact()
	phases := art.Frontmatter["phases"].([]interface{})
	phase := phases[0].(map[string]interface{})
	tasks := phase["tasks"].([]interface{})
	tasks = append(tasks,
		makeTask("refactor-1", "refactor", []string{"pkg/r.go"}, []string{"CLM-010"}, []string{"impl-1"}),
		makeTask("verify-2", "verification", []string{"pkg/r_test.go"}, []string{"CLM-011"}, []string{"refactor-1"}),
		makeTask("test-bad", "test", []string{"pkg/bad_test.go"}, []string{"CLM-012"}, []string{"refactor-1"}),
	)
	phase["tasks"] = tasks
	result := validate.Plan(art, nil)
	assertHasViolation(t, result, "plan/test-invalid-dependency")
}

func TestPlan_TestTask_DependsOnVerification(t *testing.T) {
	// Test depending on verification is allowed (gates are sequencing points)
	art := validTypedPlanArtifact()
	phases := art.Frontmatter["phases"].([]interface{})
	phase := phases[0].(map[string]interface{})
	tasks := phase["tasks"].([]interface{})
	tasks = append(tasks,
		makeTask("test-2", "test", []string{"pkg/bar_test.go"}, []string{"CLM-010"}, []string{"verify-1"}),
	)
	phase["tasks"] = tasks
	result := validate.Plan(art, nil)
	assertNoViolationRule(t, result, "plan/test-invalid-dependency")
}

func TestPlan_TestTask_DependsOnDocsFails(t *testing.T) {
	art := validTypedPlanArtifact()
	phases := art.Frontmatter["phases"].([]interface{})
	phase := phases[0].(map[string]interface{})
	tasks := phase["tasks"].([]interface{})
	tasks = append(tasks,
		makeTask("docs-1", "documentation", []string{"docs/readme.md"}, []string{"CLM-010"}, []string{"verify-1"}),
		makeTask("test-bad", "test", []string{"pkg/bad_test.go"}, []string{"CLM-011"}, []string{"docs-1"}),
	)
	phase["tasks"] = tasks
	result := validate.Plan(art, nil)
	assertHasViolation(t, result, "plan/test-invalid-dependency")
}

// REQ-007: Setup and documentation have no dependency constraints

func TestPlan_Setup_NoDependencyConstraint(t *testing.T) {
	// setup-1 with empty depends_on should pass (no constraints)
	art := validTypedPlanArtifact()
	result := validate.Plan(art, nil)
	// The fact that setup-1 has empty deps and passes is sufficient
	if !result.Pass() {
		// Only check for type-related violations, not existing ones
		for _, v := range result.Violations {
			if v.Rule == "plan/setup-invalid-dependency" {
				t.Errorf("unexpected violation: %s", v.Message)
			}
		}
	}
}

func TestPlan_Documentation_NoDependencyConstraint(t *testing.T) {
	art := validTypedPlanArtifact()
	phases := art.Frontmatter["phases"].([]interface{})
	phase := phases[0].(map[string]interface{})
	tasks := phase["tasks"].([]interface{})
	// Documentation depending on every type should be fine
	tasks = append(tasks,
		makeTask("docs-1", "documentation", []string{"docs/readme.md"}, []string{"CLM-010"}, []string{"setup-1", "test-1", "impl-1", "verify-1"}),
	)
	phase["tasks"] = tasks
	result := validate.Plan(art, nil)
	assertNoViolationRule(t, result, "plan/documentation-invalid-dependency")
}
