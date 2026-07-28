package validate_test

import (
"testing"

"github.com/backstop-ai/backstop-core/pkg/validate"
)

// REQ-001: Task type classification

func TestPlan_TaskTypeValid(t *testing.T) {
// Plan with all six valid types should pass
art := validTypedPlanArtifact()
// Add tasks of remaining types (refactor, documentation) to ensure all pass
phases := art.Frontmatter["phases"].([]interface{})
phase := phases[0].(map[string]interface{})
tasks := phase["tasks"].([]interface{})
tasks = append(tasks,
makeTask("refactor-1", "refactor", []string{"pkg/refactor.go"}, []string{"CLM-004"}, []string{"impl-1"}),
makeTask("docs-1", "documentation", []string{"docs/readme.md"}, []string{"CLM-005"}, []string{"verify-1"}),
// Need another verification for the refactor
makeTask("verify-2", "verification", []string{"pkg/refactor_test.go"}, []string{"CLM-006"}, []string{"refactor-1"}),
)
phase["tasks"] = tasks
result := validate.Plan(art, nil)
assertNoViolationRule(t, result, "plan/task-type-required")
assertNoViolationRule(t, result, "plan/task-type-invalid")
}

func TestPlan_TaskTypeMissing(t *testing.T) {
art := validTypedPlanArtifact()
// Remove "type" from the first task
phases := art.Frontmatter["phases"].([]interface{})
phase := phases[0].(map[string]interface{})
tasks := phase["tasks"].([]interface{})
task := tasks[0].(map[string]interface{})
delete(task, "type")
result := validate.Plan(art, nil)
assertHasViolation(t, result, "plan/task-type-required")
}

func TestPlan_TaskTypeInvalid(t *testing.T) {
art := validTypedPlanArtifact()
// Set first task type to invalid value
phases := art.Frontmatter["phases"].([]interface{})
phase := phases[0].(map[string]interface{})
tasks := phase["tasks"].([]interface{})
task := tasks[0].(map[string]interface{})
task["type"] = "unknown"
result := validate.Plan(art, nil)
assertHasViolation(t, result, "plan/task-type-invalid")
}

// REQ-002: TDD enforcement

func TestPlan_TDD_ImplDependsOnTest(t *testing.T) {
// Implementation with test dependency should pass
art := validTypedPlanArtifact()
result := validate.Plan(art, nil)
assertNoViolationRule(t, result, "plan/tdd-impl-requires-test")
}

func TestPlan_TDD_ImplWithoutTestDependency(t *testing.T) {
art := validTypedPlanArtifact()
// Change impl-1 to depend on setup-1 only (another implementation) instead of test-1
phases := art.Frontmatter["phases"].([]interface{})
phase := phases[0].(map[string]interface{})
tasks := phase["tasks"].([]interface{})
// Add another impl task depending on impl-1 (no test dep)
tasks = append(tasks,
makeTask("impl-2", "implementation", []string{"pkg/bar.go"}, []string{"CLM-007"}, []string{"impl-1"}),
)
phase["tasks"] = tasks
result := validate.Plan(art, nil)
assertHasViolation(t, result, "plan/tdd-impl-requires-test")
}

func TestPlan_TDD_TwoImplInRow(t *testing.T) {
art := validTypedPlanArtifact()
phases := art.Frontmatter["phases"].([]interface{})
phase := phases[0].(map[string]interface{})
tasks := phase["tasks"].([]interface{})
// impl-2 depends on impl-1, impl-1 depends on test-1 — impl-2 has no direct test dep
tasks = append(tasks,
makeTask("impl-2", "implementation", []string{"pkg/baz.go"}, []string{"CLM-008"}, []string{"impl-1"}),
)
phase["tasks"] = tasks
result := validate.Plan(art, nil)
assertHasViolation(t, result, "plan/tdd-impl-requires-test")
}

func TestPlan_TDD_ImplDependsOnSetupOnly(t *testing.T) {
// Implementation depending only on setup (no test) should fail
art := validTypedPlanArtifact()
phases := art.Frontmatter["phases"].([]interface{})
phase := phases[0].(map[string]interface{})
phase["tasks"] = []interface{}{
makeTask("setup-1", "setup", []string{"pkg/setup.go"}, []string{"CLM-001"}, []string{}),
makeTask("impl-bad", "implementation", []string{"pkg/bad.go"}, []string{"CLM-002"}, []string{"setup-1"}),
makeTask("verify-1", "verification", []string{"pkg/bad_test.go"}, []string{"CLM-003"}, []string{"impl-bad"}),
}
result := validate.Plan(art, nil)
assertHasViolation(t, result, "plan/tdd-impl-requires-test")
}
