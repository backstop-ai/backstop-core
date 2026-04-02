package validate_test

import (
	"testing"

	"github.com/bmanson/backstop-core/pkg/validate"
)

// REQ-004: Final phase must contain verification

func TestPlan_FinalPhase_HasVerification(t *testing.T) {
	// validTypedPlanArtifact has verify-1 in its only (= final) phase → pass
	art := validTypedPlanArtifact()
	result := validate.Plan(art, nil)
	assertNoViolationRule(t, result, "plan/final-phase-no-verification")
}

func TestPlan_FinalPhase_NoVerification(t *testing.T) {
	art := validTypedPlanArtifact()
	// Two phases: first has verification, second (final) has only impl + no verification
	art.Frontmatter["phases"] = []interface{}{
		makePhase("phase-1", "First",
			makeTask("setup-1", "setup", []string{"pkg/setup.go"}, []string{"CLM-001"}, []string{}),
			makeTask("test-1", "test", []string{"pkg/foo_test.go"}, []string{"CLM-002"}, []string{"setup-1"}),
			makeTask("impl-1", "implementation", []string{"pkg/foo.go"}, []string{"CLM-002"}, []string{"test-1"}),
			makeTask("verify-1", "verification", []string{"pkg/foo_test.go"}, []string{"CLM-003"}, []string{"impl-1"}),
		),
		makePhase("phase-2", "Final No Verify",
			makeTask("test-2", "test", []string{"pkg/bar_test.go"}, []string{"CLM-004"}, []string{"verify-1"}),
			makeTask("impl-2", "implementation", []string{"pkg/bar.go"}, []string{"CLM-004"}, []string{"test-2"}),
		),
	}
	result := validate.Plan(art, nil)
	assertHasViolation(t, result, "plan/final-phase-no-verification")
}

// REQ-004: Comprehensive verification — final phase must cover all work categories

func TestPlan_FinalPhase_ComprehensiveVerification(t *testing.T) {
	// Plan touches .go files and .plan.yml → final phase has verification covering both
	art := validTypedPlanArtifact()
	art.Frontmatter["phases"] = []interface{}{
		makePhase("phase-1", "Code Work",
			makeTask("setup-1", "setup", []string{"pkg/setup.go"}, []string{"CLM-001"}, []string{}),
			makeTask("test-1", "test", []string{"pkg/foo_test.go"}, []string{"CLM-002"}, []string{"setup-1"}),
			makeTask("impl-1", "implementation", []string{"pkg/foo.go"}, []string{"CLM-002"}, []string{"test-1"}),
			makeTask("verify-1", "verification", []string{"pkg/foo_test.go"}, []string{"CLM-003"}, []string{"impl-1"}),
		),
		makePhase("phase-2", "Artifact Work",
			makeTask("test-2", "test", []string{"pkg/bar_test.go"}, []string{"CLM-004"}, []string{"verify-1"}),
			makeTask("impl-2", "implementation", []string{"plans/my.plan.yml"}, []string{"CLM-004"}, []string{"test-2"}),
			makeTask("verify-final", "verification", []string{"pkg/bar_test.go", "plans/my.plan.yml"}, []string{"CLM-005"}, []string{"impl-2"}),
		),
	}
	result := validate.Plan(art, nil)
	assertNoViolationRule(t, result, "plan/final-phase-missing-category")
}

func TestPlan_FinalPhase_IncompleteVerification(t *testing.T) {
	// Plan touches .go and .plan.yml but final verification only covers .go
	art := validTypedPlanArtifact()
	art.Frontmatter["phases"] = []interface{}{
		makePhase("phase-1", "Code Work",
			makeTask("setup-1", "setup", []string{"pkg/setup.go"}, []string{"CLM-001"}, []string{}),
			makeTask("test-1", "test", []string{"pkg/foo_test.go"}, []string{"CLM-002"}, []string{"setup-1"}),
			makeTask("impl-1", "implementation", []string{"pkg/foo.go"}, []string{"CLM-002"}, []string{"test-1"}),
			makeTask("verify-1", "verification", []string{"pkg/foo_test.go"}, []string{"CLM-003"}, []string{"impl-1"}),
		),
		makePhase("phase-2", "Artifact Work",
			makeTask("test-2", "test", []string{"pkg/bar_test.go"}, []string{"CLM-004"}, []string{"verify-1"}),
			makeTask("impl-2", "implementation", []string{"plans/my.plan.yml"}, []string{"CLM-004"}, []string{"test-2"}),
			// Final verification only covers code (.go), not artifacts (.plan.yml)
			makeTask("verify-final", "verification", []string{"pkg/bar_test.go"}, []string{"CLM-005"}, []string{"impl-2"}),
		),
	}
	result := validate.Plan(art, nil)
	assertHasViolation(t, result, "plan/final-phase-missing-category")
}

// REQ-011: Phase-level parallel file exclusivity

func TestPlan_PhaseParallel_DisjointFiles(t *testing.T) {
	// Two phases with no dependency chain and disjoint file sets → pass
	art := validTypedPlanArtifact()
	art.Frontmatter["phases"] = []interface{}{
		makePhase("phase-1", "First",
			makeTask("setup-1", "setup", []string{"pkg/a.go"}, []string{"CLM-001"}, []string{}),
			makeTask("test-1", "test", []string{"pkg/a_test.go"}, []string{"CLM-002"}, []string{"setup-1"}),
			makeTask("impl-1", "implementation", []string{"pkg/a.go"}, []string{"CLM-002"}, []string{"test-1"}),
			makeTask("verify-1", "verification", []string{"pkg/a_test.go"}, []string{"CLM-003"}, []string{"impl-1"}),
		),
		makePhase("phase-2", "Second",
			makeTask("setup-2", "setup", []string{"pkg/b.go"}, []string{"CLM-004"}, []string{}),
			makeTask("test-2", "test", []string{"pkg/b_test.go"}, []string{"CLM-005"}, []string{"setup-2"}),
			makeTask("impl-2", "implementation", []string{"pkg/b.go"}, []string{"CLM-005"}, []string{"test-2"}),
			makeTask("verify-2", "verification", []string{"pkg/b_test.go"}, []string{"CLM-006"}, []string{"impl-2"}),
		),
	}
	result := validate.Plan(art, nil)
	assertNoViolationRule(t, result, "plan/phase-file-exclusivity")
}

func TestPlan_PhaseParallel_OverlappingFiles(t *testing.T) {
	// Two phases with no dependency chain but overlapping files → fail
	art := validTypedPlanArtifact()
	art.Frontmatter["phases"] = []interface{}{
		makePhase("phase-1", "First",
			makeTask("setup-1", "setup", []string{"pkg/shared.go"}, []string{"CLM-001"}, []string{}),
			makeTask("test-1", "test", []string{"pkg/a_test.go"}, []string{"CLM-002"}, []string{"setup-1"}),
			makeTask("impl-1", "implementation", []string{"pkg/shared.go"}, []string{"CLM-002"}, []string{"test-1"}),
			makeTask("verify-1", "verification", []string{"pkg/a_test.go"}, []string{"CLM-003"}, []string{"impl-1"}),
		),
		makePhase("phase-2", "Second",
			makeTask("setup-2", "setup", []string{"pkg/shared.go"}, []string{"CLM-004"}, []string{}),
			makeTask("test-2", "test", []string{"pkg/b_test.go"}, []string{"CLM-005"}, []string{"setup-2"}),
			makeTask("impl-2", "implementation", []string{"pkg/shared.go"}, []string{"CLM-005"}, []string{"test-2"}),
			makeTask("verify-2", "verification", []string{"pkg/b_test.go"}, []string{"CLM-006"}, []string{"impl-2"}),
		),
	}
	result := validate.Plan(art, nil)
	assertHasViolation(t, result, "plan/phase-file-exclusivity")
}
