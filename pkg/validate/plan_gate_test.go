package validate_test

import (
"testing"

"github.com/bmanson/backstop-core/pkg/validate"
)

// REQ-003: Gate cadence — phase with impl/refactor must have verification

func TestPlan_GateCadence_PhaseWithVerification(t *testing.T) {
// validTypedPlanArtifact has impl + verify in same phase → should pass
art := validTypedPlanArtifact()
result := validate.Plan(art, nil)
assertNoViolationRule(t, result, "plan/gate-cadence-missing")
}

func TestPlan_GateCadence_PhaseWithoutVerification(t *testing.T) {
art := validTypedPlanArtifact()
// Replace phase with one that has impl but no verification
art.Frontmatter["phases"] = []interface{}{
makePhase("phase-1", "No Verification",
makeTask("setup-1", "setup", []string{"pkg/setup.go"}, []string{"CLM-001"}, []string{}),
makeTask("test-1", "test", []string{"pkg/foo_test.go"}, []string{"CLM-002"}, []string{"setup-1"}),
makeTask("impl-1", "implementation", []string{"pkg/foo.go"}, []string{"CLM-002"}, []string{"test-1"}),
),
}
result := validate.Plan(art, nil)
assertHasViolation(t, result, "plan/gate-cadence-missing")
}

func TestPlan_GateCadence_RefactorPhaseWithoutVerification(t *testing.T) {
art := validTypedPlanArtifact()
art.Frontmatter["phases"] = []interface{}{
makePhase("phase-1", "Refactor No Verify",
makeTask("test-1", "test", []string{"pkg/foo_test.go"}, []string{"CLM-001"}, []string{}),
makeTask("impl-1", "implementation", []string{"pkg/foo.go"}, []string{"CLM-001"}, []string{"test-1"}),
makeTask("refactor-1", "refactor", []string{"pkg/bar.go"}, []string{"CLM-002"}, []string{"impl-1"}),
),
}
result := validate.Plan(art, nil)
assertHasViolation(t, result, "plan/gate-cadence-missing")
}

// REQ-005: Refactor dependency constraints

func TestPlan_Refactor_DependsOnImpl(t *testing.T) {
art := validTypedPlanArtifact()
phases := art.Frontmatter["phases"].([]interface{})
phase := phases[0].(map[string]interface{})
tasks := phase["tasks"].([]interface{})
tasks = append(tasks,
makeTask("refactor-1", "refactor", []string{"pkg/refactor.go"}, []string{"CLM-004"}, []string{"impl-1"}),
makeTask("verify-2", "verification", []string{"pkg/refactor_test.go"}, []string{"CLM-005"}, []string{"refactor-1"}),
)
phase["tasks"] = tasks
result := validate.Plan(art, nil)
assertNoViolationRule(t, result, "plan/refactor-invalid-dependency")
}

func TestPlan_Refactor_DependsOnRefactor(t *testing.T) {
art := validTypedPlanArtifact()
phases := art.Frontmatter["phases"].([]interface{})
phase := phases[0].(map[string]interface{})
tasks := phase["tasks"].([]interface{})
tasks = append(tasks,
makeTask("refactor-1", "refactor", []string{"pkg/r1.go"}, []string{"CLM-004"}, []string{"impl-1"}),
makeTask("refactor-2", "refactor", []string{"pkg/r2.go"}, []string{"CLM-005"}, []string{"refactor-1"}),
makeTask("verify-2", "verification", []string{"pkg/r_test.go"}, []string{"CLM-006"}, []string{"refactor-2"}),
)
phase["tasks"] = tasks
result := validate.Plan(art, nil)
assertNoViolationRule(t, result, "plan/refactor-invalid-dependency")
}

func TestPlan_Refactor_DependsOnTest(t *testing.T) {
art := validTypedPlanArtifact()
phases := art.Frontmatter["phases"].([]interface{})
phase := phases[0].(map[string]interface{})
tasks := phase["tasks"].([]interface{})
tasks = append(tasks,
makeTask("refactor-1", "refactor", []string{"pkg/refactor.go"}, []string{"CLM-004"}, []string{"test-1"}),
makeTask("verify-2", "verification", []string{"pkg/refactor_test.go"}, []string{"CLM-005"}, []string{"refactor-1"}),
)
phase["tasks"] = tasks
result := validate.Plan(art, nil)
assertNoViolationRule(t, result, "plan/refactor-invalid-dependency")
}

func TestPlan_Refactor_DependsOnSetupFails(t *testing.T) {
art := validTypedPlanArtifact()
phases := art.Frontmatter["phases"].([]interface{})
phase := phases[0].(map[string]interface{})
tasks := phase["tasks"].([]interface{})
tasks = append(tasks,
makeTask("refactor-bad", "refactor", []string{"pkg/refactor.go"}, []string{"CLM-004"}, []string{"setup-1"}),
makeTask("verify-2", "verification", []string{"pkg/refactor_test.go"}, []string{"CLM-005"}, []string{"refactor-bad"}),
)
phase["tasks"] = tasks
result := validate.Plan(art, nil)
assertHasViolation(t, result, "plan/refactor-invalid-dependency")
}

func TestPlan_Refactor_DependsOnDocsFails(t *testing.T) {
art := validTypedPlanArtifact()
phases := art.Frontmatter["phases"].([]interface{})
phase := phases[0].(map[string]interface{})
tasks := phase["tasks"].([]interface{})
tasks = append(tasks,
makeTask("docs-1", "documentation", []string{"docs/readme.md"}, []string{"CLM-004"}, []string{"verify-1"}),
makeTask("refactor-bad", "refactor", []string{"pkg/refactor.go"}, []string{"CLM-005"}, []string{"docs-1"}),
makeTask("verify-2", "verification", []string{"pkg/refactor_test.go"}, []string{"CLM-006"}, []string{"refactor-bad"}),
)
phase["tasks"] = tasks
result := validate.Plan(art, nil)
assertHasViolation(t, result, "plan/refactor-invalid-dependency")
}

func TestPlan_Refactor_DependsOnVerificationFails(t *testing.T) {
art := validTypedPlanArtifact()
phases := art.Frontmatter["phases"].([]interface{})
phase := phases[0].(map[string]interface{})
tasks := phase["tasks"].([]interface{})
tasks = append(tasks,
makeTask("refactor-bad", "refactor", []string{"pkg/refactor.go"}, []string{"CLM-004"}, []string{"verify-1"}),
makeTask("verify-2", "verification", []string{"pkg/refactor_test.go"}, []string{"CLM-005"}, []string{"refactor-bad"}),
)
phase["tasks"] = tasks
result := validate.Plan(art, nil)
assertHasViolation(t, result, "plan/refactor-invalid-dependency")
}

// REQ-006: Verification dependency constraints

func TestPlan_Verification_DependsOnImpl(t *testing.T) {
// validTypedPlanArtifact verify-1 depends on impl-1 → should pass
art := validTypedPlanArtifact()
result := validate.Plan(art, nil)
assertNoViolationRule(t, result, "plan/verification-requires-impl")
}

func TestPlan_Verification_DependsOnRefactor(t *testing.T) {
art := validTypedPlanArtifact()
phases := art.Frontmatter["phases"].([]interface{})
phase := phases[0].(map[string]interface{})
tasks := phase["tasks"].([]interface{})
tasks = append(tasks,
makeTask("refactor-1", "refactor", []string{"pkg/refactor.go"}, []string{"CLM-004"}, []string{"impl-1"}),
makeTask("verify-2", "verification", []string{"pkg/refactor_test.go"}, []string{"CLM-005"}, []string{"refactor-1"}),
)
phase["tasks"] = tasks
result := validate.Plan(art, nil)
assertNoViolationRule(t, result, "plan/verification-requires-impl")
}

func TestPlan_Verification_NoDependency(t *testing.T) {
art := validTypedPlanArtifact()
// Replace verify-1 to depend only on setup-1 (no impl/refactor dep)
phases := art.Frontmatter["phases"].([]interface{})
phase := phases[0].(map[string]interface{})
phase["tasks"] = []interface{}{
makeTask("setup-1", "setup", []string{"pkg/setup.go"}, []string{"CLM-001"}, []string{}),
makeTask("test-1", "test", []string{"pkg/foo_test.go"}, []string{"CLM-002"}, []string{"setup-1"}),
makeTask("impl-1", "implementation", []string{"pkg/foo.go"}, []string{"CLM-002"}, []string{"test-1"}),
makeTask("verify-bad", "verification", []string{"pkg/verify.go"}, []string{"CLM-003"}, []string{"setup-1"}),
}
result := validate.Plan(art, nil)
assertHasViolation(t, result, "plan/verification-requires-impl")
}
