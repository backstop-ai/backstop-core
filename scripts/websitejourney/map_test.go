package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

type mapBindingMutationsFile struct {
	SemanticMutations  []MapBindingMutation `yaml:"semantic_mutations"`
	GeneratedMutations []MapBindingMutation `yaml:"generated_mutations"`
}

func loadMapBindingMutations(t *testing.T) mapBindingMutationsFile {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "map-binding-mutations.yml"))
	if err != nil {
		t.Fatal(err)
	}
	var document mapBindingMutationsFile
	if err := yaml.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	if len(document.SemanticMutations) == 0 || len(document.GeneratedMutations) == 0 {
		t.Fatal("map-binding-mutations.yml must declare semantic and generated cases")
	}
	return document
}

func mustLoadWebsiteCapabilityMap(t *testing.T) WebsiteCapabilityMap {
	t.Helper()
	m, err := LoadWebsiteCapabilityMap(websiteJourneyRepoRoot(t))
	if err != nil {
		t.Fatalf("LoadWebsiteCapabilityMap: %v", err)
	}
	return m
}

func TestWebsiteJourney_EvidenceAndBoundaryBindingsPass(t *testing.T) {
	m := mustLoadWebsiteCapabilityMap(t)
	if err := ValidateEvidenceAndBoundaryBindings(m); err != nil {
		t.Fatal(err)
	}

	if m.PredecessorVersions["SPEC-072"] != "1.0.5" || m.PredecessorVersions["SPEC-074"] != "1.0.4" || m.PredecessorVersions["SPEC-075"] != "1.0.4" {
		t.Fatalf("predecessor versions = %v, want SPEC-072 1.0.5, SPEC-074 1.0.4, SPEC-075 1.0.4", m.PredecessorVersions)
	}

	wantKeys := expectedJourneyKeys()
	if len(m.Journeys) != len(wantKeys) {
		t.Fatalf("map journey cardinality: got %d, want %d", len(m.Journeys), len(wantKeys))
	}
	for index, key := range wantKeys {
		if m.Journeys[index].GlobalKey != key {
			t.Fatalf("journeys[%d] = %s, want %s", index, m.Journeys[index].GlobalKey, key)
		}
		if strings.Join(m.Journeys[index].Hops, " -> ") != strings.Join(ExpectedWebsiteJourneys()[index].Hops, " -> ") {
			t.Fatalf("%s hops = %v", key, m.Journeys[index].Hops)
		}
		if strings.Join(m.Journeys[index].JLinks, ",") != strings.Join(ExpectedWebsiteJourneys()[index].JLinks, ",") {
			t.Fatalf("%s JLINKs = %v", key, m.Journeys[index].JLinks)
		}
		for _, obligation := range m.Journeys[index].Obligations {
			if obligation.Kind != "evidence" && obligation.Kind != "boundary" && obligation.Kind != "generated" {
				t.Fatalf("%s: obligation kind %q is outside the closed set", key, obligation.Kind)
			}
			if obligation.CopiedProse || obligation.Inferred || obligation.Seed5Resolved {
				t.Fatalf("%s: map must import owner IDs only, not copied/inferred/resolved content", key)
			}
		}
	}

	wantDeps := []string{"seed1-product-model", "seed2-documentation-semantics", "seed3-product-truth", "seed4-public-site"}
	if strings.Join(m.PrerequisiteIDs(), ",") != strings.Join(wantDeps, ",") {
		t.Fatalf("prerequisites = %v, want %v", m.PrerequisiteIDs(), wantDeps)
	}
	if strings.Join(m.AdoptionInstructionIDs(), ",") != "ADOPT-INSTALL,ADOPT-CONFIGURE,ADOPT-ENFORCE" {
		t.Fatalf("adoption refs = %v", m.AdoptionInstructionIDs())
	}

	cap014 := m.Journey("CAP-014/@UJ-001")
	if cap014.DualIdentity.JourneyLinkID != "JLINK-024" || cap014.DualIdentity.BoundaryID != "BOUNDARY-005" {
		t.Fatalf("CAP-014/@UJ-001 dual identity = %+v, want JLINK-024 and BOUNDARY-005 as one owner relationship", cap014.DualIdentity)
	}
	if cap014.DualIdentity.Href != "/contributing/#external-ownership" {
		t.Fatalf("CAP-014/@UJ-001 dual-identity href = %q", cap014.DualIdentity.Href)
	}
}

func TestWebsiteJourney_RejectsInvalidSemanticBindingMatrix(t *testing.T) {
	document := loadMapBindingMutations(t)
	base := mustLoadWebsiteCapabilityMap(t)
	for _, mutation := range document.SemanticMutations {
		t.Run(mutation.Name, func(t *testing.T) {
			err := ValidateEvidenceAndBoundaryBindings(ApplyMapBindingMutation(base, mutation))
			if err == nil {
				t.Fatal("accepted invalid semantic binding matrix")
			}
			if !strings.Contains(err.Error(), mutation.ExpectedError) {
				t.Fatalf("error %q does not name %q", err, mutation.ExpectedError)
			}
		})
	}
}

func TestWebsiteJourney_GeneratedObligationMatrixPasses(t *testing.T) {
	m := mustLoadWebsiteCapabilityMap(t)
	if err := ValidateGeneratedObligationMatrix(m); err != nil {
		t.Fatal(err)
	}

	cap011 := m.Journey("CAP-011/@UJ-001")
	if jobs := cap011.GeneratedJobIDs(); strings.Join(jobs, ",") != "installed-pack-catalog" {
		t.Fatalf("CAP-011/@UJ-001 jobs = %v, want installed-pack-catalog", jobs)
	}
	cap013 := m.Journey("CAP-013/@UJ-002")
	if jobs := cap013.GeneratedJobIDs(); strings.Join(jobs, ",") != "cli-command-catalog,artifact-schema-catalog,installed-pack-catalog,release-history" {
		t.Fatalf("CAP-013/@UJ-002 jobs = %v, want all four generated jobs", jobs)
	}
	for _, key := range []string{"CAP-011/@UJ-001", "CAP-013/@UJ-002"} {
		for _, job := range m.Journey(key).GeneratedObligations() {
			if job.SourceDigest == "" || !strings.HasPrefix(job.SourceDigest, "sha256:") {
				t.Fatalf("%s job %s missing canonical source-level digest", key, job.JobID)
			}
			if job.TruthBeginMarker == "" || job.TruthEndMarker == "" || job.SourcesBeginMarker == "" || job.SourcesEndMarker == "" {
				t.Fatalf("%s job %s missing SPEC-074 marker pair", key, job.JobID)
			}
			if len(job.Descriptors) == 0 || len(job.URLTemplates) == 0 {
				t.Fatalf("%s job %s missing typed descriptors or URL templates", key, job.JobID)
			}
			if job.SiteIdentity == "" || len(job.RenderedAnchors) == 0 {
				t.Fatalf("%s job %s missing SPEC-075 site identity or rendered-anchor set", key, job.JobID)
			}
			if job.ReconstructionVerdict == "" || job.RenderedDigest == "" {
				t.Fatalf("%s job %s missing SPEC-075 reconstruction or rendered-digest verdict", key, job.JobID)
			}
			for _, template := range job.URLTemplates {
				if strings.Contains(template, "<SITE-COMMIT>") {
					continue
				}
				if job.JobID != "release-history" {
					t.Fatalf("%s job %s resolved a SITE-COMMIT template in Seed 5: %s", key, job.JobID, template)
				}
			}
		}
	}
}

func TestWebsiteJourney_RejectsInvalidGeneratedObligationMatrix(t *testing.T) {
	document := loadMapBindingMutations(t)
	base := mustLoadWebsiteCapabilityMap(t)
	for _, mutation := range document.GeneratedMutations {
		t.Run(mutation.Name, func(t *testing.T) {
			err := ValidateGeneratedObligationMatrix(ApplyMapBindingMutation(base, mutation))
			if err == nil {
				t.Fatal("accepted invalid generated obligation matrix")
			}
			if !strings.Contains(err.Error(), mutation.ExpectedError) {
				t.Fatalf("error %q does not name %q", err, mutation.ExpectedError)
			}
		})
	}
}

func expectedJourneyKeys() []string {
	keys := make([]string, 0, 21)
	for _, journey := range ExpectedWebsiteJourneys() {
		keys = append(keys, journey.GlobalKey)
	}
	return keys
}
