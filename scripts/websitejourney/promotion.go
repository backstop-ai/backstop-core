package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	promotionFixtureCommit = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	promotionFixtureRunID  = "12345"
)

type PromotionEvidence struct {
	Commit          string
	RunID           string
	DeployedCommit  string
	DeployedRunID   string
	Artifacts       bool
	BuiltJourneys   bool
	MutationClasses []string
	Adoption        bool
	Deployed        bool
	Dependency      bool
	Workflows       bool
	Blocking        bool
}

type PromotionEvidenceMutation struct {
	Name              string `yaml:"name"`
	ArtifactsPass     *bool  `yaml:"artifacts_pass"`
	BuiltPass         *bool  `yaml:"built_pass"`
	AdoptionPass      *bool  `yaml:"adoption_pass"`
	DeployedPass      *bool  `yaml:"deployed_pass"`
	DependencyPass    *bool  `yaml:"dependency_pass"`
	WorkflowsPass     *bool  `yaml:"workflows_pass"`
	OmitMutationClass string `yaml:"omit_mutation_class"`
	StaleCommit       bool   `yaml:"stale_commit"`
	MixedRun          bool   `yaml:"mixed_run"`
	Nonblocking       bool   `yaml:"nonblocking"`
	KeepDraft         bool   `yaml:"keep_draft"`
	PromotePartial    string `yaml:"promote_partial"`
	ExpectedError     string `yaml:"expected_error"`
}

func RequiredMutationClasses() []string {
	return []string{"route", "evidence", "boundary", "generated", "dual-identity"}
}

func CompletePromotionEvidence() PromotionEvidence {
	return PromotionEvidence{
		Commit:          promotionFixtureCommit,
		RunID:           promotionFixtureRunID,
		DeployedCommit:  promotionFixtureCommit,
		DeployedRunID:   promotionFixtureRunID,
		Artifacts:       true,
		BuiltJourneys:   true,
		MutationClasses: append([]string(nil), RequiredMutationClasses()...),
		Adoption:        true,
		Deployed:        true,
		Dependency:      true,
		Workflows:       true,
		Blocking:        true,
	}
}

func ValidatePromotion(tree CapabilityTree, evidence PromotionEvidence) error {
	var verifiedIDs, draftIDs []string
	for _, artifact := range tree.Artifacts {
		if !websiteCapabilityID(artifact.ID) {
			continue
		}
		switch artifact.Status {
		case "verified":
			verifiedIDs = append(verifiedIDs, artifact.ID)
		default:
			draftIDs = append(draftIDs, artifact.ID)
		}
	}
	fault := promotionEvidenceFault(evidence)
	if len(verifiedIDs) > 0 && fault != nil {
		if len(verifiedIDs) != len(ExpectedWebsiteCapabilities()) {
			return fmt.Errorf("%s: status %q, want draft until blocking gates pass", verifiedIDs[0], "verified")
		}
		return fault
	}
	if fault == nil && len(draftIDs) > 0 {
		return fmt.Errorf("%s: status %q, want verified after same-release evidence", draftIDs[0], "draft")
	}
	return nil
}

func ApplyPromotionMutation(tree CapabilityTree, evidence PromotionEvidence, mutation PromotionEvidenceMutation) (CapabilityTree, PromotionEvidence) {
	cloned := cloneTree(tree)
	next := evidence
	if mutation.ArtifactsPass != nil {
		next.Artifacts = *mutation.ArtifactsPass
	}
	if mutation.BuiltPass != nil {
		next.BuiltJourneys = *mutation.BuiltPass
	}
	if mutation.AdoptionPass != nil {
		next.Adoption = *mutation.AdoptionPass
	}
	if mutation.DeployedPass != nil {
		next.Deployed = *mutation.DeployedPass
	}
	if mutation.DependencyPass != nil {
		next.Dependency = *mutation.DependencyPass
	}
	if mutation.WorkflowsPass != nil {
		next.Workflows = *mutation.WorkflowsPass
	}
	if mutation.OmitMutationClass != "" {
		var filtered []string
		for _, class := range next.MutationClasses {
			if class != mutation.OmitMutationClass {
				filtered = append(filtered, class)
			}
		}
		next.MutationClasses = filtered
	}
	if mutation.StaleCommit {
		next.DeployedCommit = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	}
	if mutation.MixedRun {
		next.DeployedRunID = "9" + next.RunID
	}
	if mutation.Nonblocking {
		next.Blocking = false
	}
	if mutation.KeepDraft {
		for i := range cloned.Artifacts {
			cloned.Artifacts[i].Status = "draft"
		}
	}
	if mutation.PromotePartial != "" {
		for i := range cloned.Artifacts {
			if cloned.Artifacts[i].ID == mutation.PromotePartial {
				cloned.Artifacts[i].Status = "verified"
			} else {
				cloned.Artifacts[i].Status = "draft"
			}
		}
	}
	return cloned, next
}

func CollectPromotionEvidence(root string) (PromotionEvidence, error) {
	evidence := PromotionEvidence{
		Commit:         promotionFixtureCommit,
		RunID:          promotionFixtureRunID,
		DeployedCommit: promotionFixtureCommit,
		DeployedRunID:  promotionFixtureRunID,
		Blocking:       true,
	}
	tree, err := LoadCapabilityTree(root)
	if err != nil {
		return evidence, err
	}
	if err := ValidateCapabilityArtifactMatrix(tree); err != nil {
		return evidence, nil
	}
	if err := ValidateScenarioAndCoverageMatrix(tree); err != nil {
		return evidence, nil
	}
	evidence.Artifacts = true
	journeyMap, err := LoadWebsiteCapabilityMap(root)
	if err != nil {
		return evidence, err
	}
	if _, err := EvaluatePrerequisites(journeyMap, tree, FreshZeroPrerequisiteRunner()); err == nil {
		evidence.Dependency = true
	}
	built, err := os.MkdirTemp("", "websitejourney-promotion-built-")
	if err != nil {
		return evidence, err
	}
	defer func() { _ = os.RemoveAll(built) }()
	if err := WriteAcceptedBuiltTree(built, journeyMap); err != nil {
		return evidence, err
	}
	if err := TraverseBuiltJourneys(built, journeyMap); err == nil {
		evidence.BuiltJourneys = true
	}
	if classes, err := proveMutationClasses(built, journeyMap); err == nil {
		evidence.MutationClasses = classes
	}
	instructions, err := LoadOwnerAdoptionInstructions(root)
	if err == nil && len(instructions) == 3 {
		evidence.Adoption = true
	}
	stamped, err := os.MkdirTemp("", "websitejourney-promotion-deployed-")
	if err != nil {
		return evidence, err
	}
	defer func() { _ = os.RemoveAll(stamped) }()
	if err := copyBuiltTree(built, stamped); err != nil {
		return evidence, err
	}
	if err := StampDeployedIdentity(stamped, promotionFixtureCommit, promotionFixtureRunID); err != nil {
		return evidence, err
	}
	if err := TraverseDeployedSite(journeyMap, DeployedRequest{
		Origin: CanonicalDeployedOrigin,
		Commit: promotionFixtureCommit,
		RunID:  promotionFixtureRunID,
		Fetch:  FixtureDeployedFetcher(stamped, CanonicalDeployedOrigin, ""),
	}); err == nil {
		evidence.Deployed = true
	}
	wired, err := workflowsWired(root)
	if err != nil {
		return evidence, err
	}
	evidence.Workflows = wired
	return evidence, nil
}

func proveMutationClasses(built string, m WebsiteCapabilityMap) ([]string, error) {
	representatives := []JourneyMutation{
		RouteMutations()[0],
		firstClassMutation(EvidenceMutations(m), "evidence"),
		firstClassMutation(BoundaryMutations(m), "boundary"),
		{Class: "generated", JobID: "installed-pack-catalog", RemoveRegion: true, Target: "installed-pack-catalog"},
		{Class: "cap014", RemoveSharedAnchor: true, Target: "JLINK-024"},
	}
	proved := make([]string, 0, len(RequiredMutationClasses()))
	for index, mutation := range representatives {
		if mutation.Class == "" {
			return proved, fmt.Errorf("promotion: mutation class %s has no representative", RequiredMutationClasses()[index])
		}
		kill := KillJourneyMutation(built, m, mutation)
		if kill.Err == nil {
			return proved, fmt.Errorf("promotion: mutation class %s was not killed", RequiredMutationClasses()[index])
		}
		proved = append(proved, RequiredMutationClasses()[index])
	}
	return proved, nil
}

func firstClassMutation(mutations []JourneyMutation, class string) JourneyMutation {
	for _, mutation := range mutations {
		if mutation.Class == class {
			return mutation
		}
	}
	return JourneyMutation{}
}

func promotionEvidenceFault(evidence PromotionEvidence) error {
	if !evidence.Artifacts {
		return fmt.Errorf("promotion: artifact evidence is incomplete")
	}
	if !evidence.BuiltJourneys {
		return fmt.Errorf("promotion: built journey evidence is incomplete")
	}
	if !hasRequiredMutationClasses(evidence.MutationClasses) {
		return fmt.Errorf("promotion: mutation class evidence is incomplete")
	}
	if !evidence.Adoption {
		return fmt.Errorf("promotion: CAP-009 adoption evidence is incomplete")
	}
	if !evidence.Deployed {
		return fmt.Errorf("promotion: deployed evidence is incomplete")
	}
	if !evidence.Dependency {
		return fmt.Errorf("promotion: dependency evidence is incomplete")
	}
	if !evidence.Workflows {
		return fmt.Errorf("promotion: workflow evidence is incomplete")
	}
	if !evidence.Blocking {
		return fmt.Errorf("promotion: nonblocking evidence cannot promote")
	}
	if !sameReleaseIdentity(evidence) {
		return fmt.Errorf("promotion: stale or mixed identity")
	}
	return nil
}

func hasRequiredMutationClasses(got []string) bool {
	seen := map[string]bool{}
	for _, class := range got {
		seen[class] = true
	}
	for _, want := range RequiredMutationClasses() {
		if !seen[want] {
			return false
		}
	}
	return true
}

func sameReleaseIdentity(evidence PromotionEvidence) bool {
	return fullCommitSHA(evidence.Commit) && validRunID(evidence.RunID) &&
		evidence.Commit == evidence.DeployedCommit && evidence.RunID == evidence.DeployedRunID
}

func workflowsWired(root string) (bool, error) {
	ci, err := os.ReadFile(filepath.Join(root, ".github", "workflows", "ci.yml"))
	if err != nil {
		return false, err
	}
	pages, err := os.ReadFile(filepath.Join(root, ".github", "workflows", "pages.yml"))
	if err != nil {
		return false, err
	}
	ciText := string(ci)
	pagesText := string(pages)
	if !strings.Contains(ciText, "./scripts/verify-website-capabilities.sh") ||
		!strings.Contains(ciText, "website-capabilities") ||
		strings.Contains(ciText, "--deployed-origin") {
		return false, nil
	}
	if !strings.Contains(pagesText, "./scripts/verify-pages-deployment.sh") ||
		!strings.Contains(pagesText, "--deployed-origin https://backstop.sh") ||
		!strings.Contains(pagesText, "BACKSTOP_DEPLOY_COMMIT") ||
		strings.Contains(strings.ToLower(pagesText), "rollback") ||
		strings.Contains(strings.ToLower(ciText), "rollback") {
		return false, nil
	}
	return true, nil
}

func websiteCapabilityCohortStatus(tree CapabilityTree) (string, error) {
	if len(tree.Artifacts) == 0 {
		return "", fmt.Errorf("capability cardinality: got 0")
	}
	want := tree.Artifacts[0].Status
	for _, artifact := range tree.Artifacts {
		if artifact.Status != want {
			if artifact.Status == "verified" {
				return "", fmt.Errorf("%s: status %q, want draft until blocking gates pass", artifact.ID, artifact.Status)
			}
			return "", fmt.Errorf("%s: status %q, want %s", artifact.ID, artifact.Status, want)
		}
	}
	if want != "draft" && want != "verified" {
		return "", fmt.Errorf("%s: status %q, want draft until blocking gates pass", tree.Artifacts[0].ID, want)
	}
	return want, nil
}
