package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

type promotionEvidenceMutationsFile struct {
	InvalidPromotions []PromotionEvidenceMutation `yaml:"invalid_promotions"`
}

func loadPromotionEvidenceMutations(t *testing.T) promotionEvidenceMutationsFile {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "promotion-evidence-mutations.yml"))
	if err != nil {
		t.Fatal(err)
	}
	var document promotionEvidenceMutationsFile
	if err := yaml.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	if len(document.InvalidPromotions) < 8 {
		t.Fatal("promotion-evidence-mutations.yml must cover partial, stale, mixed, and nonblocking cases")
	}
	return document
}

func verifiedWebsiteTree(t *testing.T) CapabilityTree {
	t.Helper()
	tree := mustLoadCapabilityTree(t)
	for i := range tree.Artifacts {
		tree.Artifacts[i].Status = "verified"
	}
	return tree
}

func TestWebsiteJourney_SameReleasePromotionPasses(t *testing.T) {
	root := websiteJourneyRepoRoot(t)
	tree := mustLoadCapabilityTree(t)
	evidence, err := CollectPromotionEvidence(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidatePromotion(tree, evidence); err != nil {
		t.Fatal(err)
	}
	if err := promotionEvidenceFault(evidence); err != nil {
		t.Fatal(err)
	}
	for _, artifact := range tree.Artifacts {
		if artifact.Status != "verified" {
			t.Fatalf("%s: status %q, want verified after same-release evidence", artifact.ID, artifact.Status)
		}
	}
}

func TestWebsiteJourney_RejectsInvalidPromotionEvidence(t *testing.T) {
	document := loadPromotionEvidenceMutations(t)
	base := verifiedWebsiteTree(t)
	for _, mutation := range document.InvalidPromotions {
		t.Run(mutation.Name, func(t *testing.T) {
			tree, next := ApplyPromotionMutation(base, CompletePromotionEvidence(), mutation)
			err := ValidatePromotion(tree, next)
			if err == nil {
				t.Fatal("accepted invalid promotion evidence")
			}
			if !strings.Contains(err.Error(), mutation.ExpectedError) {
				t.Fatalf("error %q does not name %q", err, mutation.ExpectedError)
			}
		})
	}
}
