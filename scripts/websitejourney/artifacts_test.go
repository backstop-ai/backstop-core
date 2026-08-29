package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

type artifactScenarioMutationsFile struct {
	CapabilityMutations []CapabilityMutation `yaml:"capability_mutations"`
	ScenarioMutations   []ScenarioMutation   `yaml:"scenario_mutations"`
}

func websiteJourneyRepoRoot(t *testing.T) string {
	t.Helper()
	root := filepath.Clean(filepath.Join("..", ".."))
	if _, err := os.Stat(filepath.Join(root, "capabilities")); err != nil {
		t.Fatalf("capabilities directory: %v", err)
	}
	return root
}

func loadArtifactScenarioMutations(t *testing.T) artifactScenarioMutationsFile {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "artifact-scenario-mutations.yml"))
	if err != nil {
		t.Fatal(err)
	}
	var document artifactScenarioMutationsFile
	if err := yaml.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	if len(document.CapabilityMutations) == 0 || len(document.ScenarioMutations) == 0 {
		t.Fatal("artifact-scenario-mutations.yml must declare capability and scenario cases")
	}
	return document
}

func mustLoadCapabilityTree(t *testing.T) CapabilityTree {
	t.Helper()
	tree, err := LoadCapabilityTree(websiteJourneyRepoRoot(t))
	if err != nil {
		t.Fatalf("LoadCapabilityTree: %v", err)
	}
	return tree
}

func TestWebsiteJourney_ExactCapabilityArtifactMatrixPasses(t *testing.T) {
	tree := mustLoadCapabilityTree(t)
	if err := ValidateCapabilityArtifactMatrix(tree); err != nil {
		t.Fatal(err)
	}

	want := ExpectedWebsiteCapabilities()
	if len(tree.Artifacts) != len(want) {
		t.Fatalf("capability cardinality: got %d, want %d", len(tree.Artifacts), len(want))
	}
	for index, expected := range want {
		got := tree.Artifacts[index]
		if got.ID != expected.ID || got.Slug != expected.Slug || got.Title != expected.Title {
			t.Fatalf("artifacts[%d]: got %s %s %q, want %s %s %q", index, got.ID, got.Slug, got.Title, expected.ID, expected.Slug, expected.Title)
		}
		if got.Status != "draft" || got.Strictness != "strict" {
			t.Fatalf("%s: status/strictness = %s/%s, want draft/strict", got.ID, got.Status, got.Strictness)
		}
		if strings.Join(got.InfrastructureSpecs, ",") != "SPEC-072,SPEC-073,SPEC-074,SPEC-075" {
			t.Fatalf("%s: infrastructure_specs = %v", got.ID, got.InfrastructureSpecs)
		}
		if got.IntegrationSpec != "SPEC-076" {
			t.Fatalf("%s: integration_spec = %q", got.ID, got.IntegrationSpec)
		}
		if err := validateExactCapabilityGates(got); err != nil {
			t.Fatal(err)
		}
	}
}

func TestWebsiteJourney_RejectsInvalidCapabilityArtifactMatrix(t *testing.T) {
	document := loadArtifactScenarioMutations(t)
	base := mustLoadCapabilityTree(t)
	for _, mutation := range document.CapabilityMutations {
		t.Run(mutation.Name, func(t *testing.T) {
			err := ValidateCapabilityArtifactMatrix(ApplyCapabilityMutation(base, mutation))
			if err == nil {
				t.Fatal("accepted invalid capability matrix")
			}
			if !strings.Contains(err.Error(), mutation.ExpectedError) {
				t.Fatalf("error %q does not name %q", err, mutation.ExpectedError)
			}
		})
	}
}

func TestWebsiteJourney_ExactScenarioAndCoverageMatrixPasses(t *testing.T) {
	tree := mustLoadCapabilityTree(t)
	if err := ValidateScenarioAndCoverageMatrix(tree); err != nil {
		t.Fatal(err)
	}

	want := ExpectedWebsiteJourneys()
	got := tree.Journeys()
	if len(got) != len(want) {
		t.Fatalf("journey cardinality: got %d, want %d", len(got), len(want))
	}
	for index, expected := range want {
		actual := got[index]
		if actual.GlobalKey != expected.GlobalKey {
			t.Fatalf("journeys[%d]: got %s, want %s", index, actual.GlobalKey, expected.GlobalKey)
		}
		if actual.Title != expected.Title {
			t.Fatalf("%s title = %q, want %q", expected.GlobalKey, actual.Title, expected.Title)
		}
		if strings.Join(actual.JLinks, ",") != strings.Join(expected.JLinks, ",") {
			t.Fatalf("%s JLINKs = %v, want %v", expected.GlobalKey, actual.JLinks, expected.JLinks)
		}
		if strings.Join(actual.Hops, " -> ") != strings.Join(expected.Hops, " -> ") {
			t.Fatalf("%s hops = %v, want %v", expected.GlobalKey, actual.Hops, expected.Hops)
		}
		if !actual.HasGiven || !actual.HasWhen || !actual.HasThen {
			t.Fatalf("%s is not an executable Given/When/Then scenario", expected.GlobalKey)
		}
		if !strings.Contains(actual.ExecutableTestTitle, expected.GlobalKey) {
			t.Fatalf("%s missing executable browser coverage title containing the global key", expected.GlobalKey)
		}
		if actual.UsesDirectLoad {
			t.Fatalf("%s uses a direct load", expected.GlobalKey)
		}
		if actual.UsesGlobalNavigation {
			t.Fatalf("%s uses global navigation", expected.GlobalKey)
		}
	}
}

func TestWebsiteJourney_RejectsScenarioAndCoverageDrift(t *testing.T) {
	document := loadArtifactScenarioMutations(t)
	base := mustLoadCapabilityTree(t)
	for _, mutation := range document.ScenarioMutations {
		t.Run(mutation.Name, func(t *testing.T) {
			err := ValidateScenarioAndCoverageMatrix(ApplyScenarioMutation(base, mutation))
			if err == nil {
				t.Fatal("accepted invalid scenario/coverage matrix")
			}
			if !strings.Contains(err.Error(), mutation.ExpectedError) {
				t.Fatalf("error %q does not name %q", err, mutation.ExpectedError)
			}
			if mutation.Name == "local-only-id" && !strings.Contains(err.Error(), "CAP-004/@UJ-001") {
				t.Fatalf("local-only ID diagnostic must carry the global key, got %q", err)
			}
			if strings.Contains(err.Error(), mutation.ExpectedError) && !strings.Contains(mutation.ExpectedError, "/") {
				return
			}
			if strings.HasPrefix(mutation.ExpectedError, "CAP-") && strings.Contains(mutation.ExpectedError, "/@UJ-") {
				if !strings.Contains(err.Error(), mutation.ExpectedError) {
					t.Fatalf("diagnostic must name global journey key %s: %v", mutation.ExpectedError, err)
				}
			}
		})
	}

	for _, journey := range ExpectedWebsiteJourneys() {
		for _, link := range journey.JLinks {
			t.Run("omit-"+link+"-from-"+strings.ReplaceAll(journey.GlobalKey, "/", "_"), func(t *testing.T) {
				err := ValidateScenarioAndCoverageMatrix(OmitJourneyLink(base, journey.GlobalKey, link))
				if err == nil {
					t.Fatalf("omitting %s from %s was accepted", link, journey.GlobalKey)
				}
				if !strings.Contains(err.Error(), journey.GlobalKey) {
					t.Fatalf("omitting %s: error %q does not name %s", link, err, journey.GlobalKey)
				}
			})
			t.Run("wrong-bind-"+link+"-on-"+strings.ReplaceAll(journey.GlobalKey, "/", "_"), func(t *testing.T) {
				err := ValidateScenarioAndCoverageMatrix(WrongBindJourneyLink(base, journey.GlobalKey, link, "JLINK-099"))
				if err == nil {
					t.Fatalf("wrong-binding %s on %s was accepted", link, journey.GlobalKey)
				}
				if !strings.Contains(err.Error(), journey.GlobalKey) {
					t.Fatalf("wrong-binding %s: error %q does not name %s", link, err, journey.GlobalKey)
				}
			})
		}
	}
}

func validateExactCapabilityGates(artifact CapabilityArtifact) error {
	built := fmt.Sprintf("./scripts/verify-website-capabilities.sh --capability %s", artifact.ID)
	deployed := fmt.Sprintf("./scripts/verify-website-capabilities.sh --capability %s --deployed-origin https://backstop.sh --commit \"$BACKSTOP_DEPLOY_COMMIT\" --run-id \"$BACKSTOP_DEPLOY_RUN_ID\"", artifact.ID)
	if len(artifact.Gates) != 2 {
		return fmt.Errorf("%s: want exactly two blocking gates, got %d", artifact.ID, len(artifact.Gates))
	}
	if artifact.Gates[0].Command != built || !artifact.Gates[0].MustPass {
		return fmt.Errorf("%s: built gate = %q must_pass=%v", artifact.ID, artifact.Gates[0].Command, artifact.Gates[0].MustPass)
	}
	if artifact.Gates[1].Command != deployed || !artifact.Gates[1].MustPass {
		return fmt.Errorf("%s: deployed gate = %q must_pass=%v", artifact.ID, artifact.Gates[1].Command, artifact.Gates[1].MustPass)
	}
	if artifact.FeatureFile != "user-journeys.feature" || artifact.IDPattern != `^@UJ-[0-9]{3}$` {
		return fmt.Errorf("%s: scenario file/pattern = %s %s", artifact.ID, artifact.FeatureFile, artifact.IDPattern)
	}
	if len(artifact.AppOrigins) != 1 || artifact.AppOrigins[0] != "https://backstop.sh" {
		return fmt.Errorf("%s: app_origins = %v", artifact.ID, artifact.AppOrigins)
	}
	if artifact.AuthStrategy != "none" || artifact.SetupCommand != "" || artifact.TeardownCommand != "" {
		return fmt.Errorf("%s: test_configuration auth=%q setup=%q teardown=%q", artifact.ID, artifact.AuthStrategy, artifact.SetupCommand, artifact.TeardownCommand)
	}
	return nil
}
