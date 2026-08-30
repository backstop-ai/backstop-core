package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

type exhaustiveMutationsFile struct {
	GeneratedMutations []JourneyMutation `yaml:"generated_mutations"`
	CAP014Mutations    []JourneyMutation `yaml:"cap014_mutations"`
}

func loadExhaustiveMutations(t *testing.T) exhaustiveMutationsFile {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "exhaustive-journey-mutations.yml"))
	if err != nil {
		t.Fatal(err)
	}
	var document exhaustiveMutationsFile
	if err := yaml.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	if len(document.GeneratedMutations) == 0 || len(document.CAP014Mutations) < 10 {
		t.Fatal("exhaustive-journey-mutations.yml must cover generated and CAP-014 dual-identity cases")
	}
	return document
}

func TestWebsiteJourney_EveryCanonicalRouteRemovalBreaksJourney(t *testing.T) {
	built, m := mustAcceptedBuiltTree(t)
	mutations := RouteMutations()
	if len(mutations) != 10 {
		t.Fatalf("canonical routes = %d, want 10", len(mutations))
	}
	for _, mutation := range mutations {
		t.Run(mutation.Target, func(t *testing.T) {
			kill := KillJourneyMutation(built, m, mutation)
			if kill.Err == nil || !strings.Contains(kill.Err.Error(), "route") || kill.Journey == "" {
				t.Fatalf("route %s: %v journey=%s", mutation.Target, kill.Err, kill.Journey)
			}
			if !strings.Contains(kill.Err.Error(), kill.Journey) {
				t.Fatalf("error %q does not name journey %s", kill.Err, kill.Journey)
			}
		})
	}
}

func TestWebsiteJourney_EveryEvidenceEdgeRemovalBreaksJourney(t *testing.T) {
	built, m := mustAcceptedBuiltTree(t)
	mutations := EvidenceMutations(m)
	if len(mutations) == 0 {
		t.Fatal("expected evidence mutations")
	}
	for _, mutation := range mutations {
		t.Run(mutation.Class+"/"+mutation.Target, func(t *testing.T) {
			kill := KillJourneyMutation(built, m, mutation)
			if kill.Err == nil || kill.Journey == "" {
				t.Fatalf("%s %s still passed", mutation.Class, mutation.Target)
			}
			if !strings.Contains(kill.Err.Error(), kill.Journey) {
				t.Fatalf("error %q does not name %s", kill.Err, kill.Journey)
			}
		})
	}
}

func TestWebsiteJourney_EveryBoundaryExplanationRemovalBreaksJourney(t *testing.T) {
	built, m := mustAcceptedBuiltTree(t)
	mutations := BoundaryMutations(m)
	if len(mutations) == 0 {
		t.Fatal("expected boundary mutations")
	}
	for _, mutation := range mutations {
		t.Run(mutation.Class+"/"+mutation.Target, func(t *testing.T) {
			kill := KillJourneyMutation(built, m, mutation)
			if kill.Err == nil || kill.Journey == "" {
				t.Fatalf("%s %s still passed", mutation.Class, mutation.Target)
			}
			if !strings.Contains(kill.Err.Error(), kill.Journey) || !strings.Contains(kill.Err.Error(), mutation.Target) {
				t.Fatalf("error %q does not name %s and %s", kill.Err, kill.Journey, mutation.Target)
			}
		})
	}
}

func TestWebsiteJourney_MutationDiagnosticsAreJourneySpecific(t *testing.T) {
	built, m := mustAcceptedBuiltTree(t)
	first := KillJourneyMutation(built, m, JourneyMutation{Class: "route", Target: "/"})
	second := KillJourneyMutation(built, m, JourneyMutation{Class: "route", Target: "/extend/"})
	if first.Err == nil || second.Err == nil {
		t.Fatal("expected both route mutants to die")
	}
	if first.Journey == "" || second.Journey == "" || first.Journey == second.Journey {
		t.Fatalf("diagnostics must name different journeys, got %s and %s", first.Journey, second.Journey)
	}
	if !strings.Contains(first.Err.Error(), "route") || !strings.Contains(second.Err.Error(), "route") {
		t.Fatalf("diagnostics must name the mutation class: %v / %v", first.Err, second.Err)
	}
}

func TestWebsiteJourney_EveryGeneratedProvenanceMutationBreaksJourney(t *testing.T) {
	document := loadExhaustiveMutations(t)
	built, m := mustAcceptedBuiltTree(t)
	for _, mutation := range document.GeneratedMutations {
		mutation.Class = "generated"
		mutation.Target = mutation.JobID
		t.Run(mutation.Name, func(t *testing.T) {
			kill := KillJourneyMutation(built, m, mutation)
			if kill.Err == nil {
				t.Fatal("generated mutant still passed")
			}
			if !strings.Contains(kill.Err.Error(), mutation.ExpectedError) {
				t.Fatalf("error %q does not name %q", kill.Err, mutation.ExpectedError)
			}
		})
	}
}

func TestWebsiteJourney_EveryCAP014DualIdentityMutationBreaksNamedJourney(t *testing.T) {
	document := loadExhaustiveMutations(t)
	built, m := mustAcceptedBuiltTree(t)
	for _, mutation := range document.CAP014Mutations {
		mutation.Class = "cap014"
		mutation.Target = "JLINK-024/BOUNDARY-005"
		t.Run(mutation.Name, func(t *testing.T) {
			kill := KillJourneyMutation(built, m, mutation)
			if kill.Err == nil {
				t.Fatal("CAP-014 mutant still passed")
			}
			for _, token := range []string{"CAP-014/@UJ-001", "JLINK-024", "BOUNDARY-005"} {
				if !strings.Contains(kill.Err.Error(), token) {
					t.Fatalf("error %q does not name %q", kill.Err, token)
				}
			}
		})
	}
}
