package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

type builtJourneyMutationsFile struct {
	JLinkRemovals   []BuiltJourneyMutation `yaml:"jlink_removals"`
	JLinkWrongBinds []BuiltJourneyMutation `yaml:"jlink_wrong_binds"`
	CAP014Mutations []BuiltJourneyMutation `yaml:"cap014_mutations"`
	CheatMutations  []BuiltJourneyMutation `yaml:"cheat_mutations"`
}

func loadBuiltJourneyMutations(t *testing.T) builtJourneyMutationsFile {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "built-journey-mutations.yml"))
	if err != nil {
		t.Fatal(err)
	}
	var document builtJourneyMutationsFile
	if err := yaml.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	if len(document.JLinkRemovals) != 24 || len(document.JLinkWrongBinds) == 0 || len(document.CAP014Mutations) == 0 {
		t.Fatal("built-journey-mutations.yml must cover JLINK-001..024 and CAP-014 cases")
	}
	return document
}

func mustAcceptedBuiltTree(t *testing.T) (string, WebsiteCapabilityMap) {
	t.Helper()
	m := mustLoadWebsiteCapabilityMap(t)
	dest := t.TempDir()
	if err := WriteAcceptedBuiltTree(dest, m); err != nil {
		t.Fatal(err)
	}
	return dest, m
}

func TestWebsiteJourney_BuiltSiteAllJourneysPass(t *testing.T) {
	built, m := mustAcceptedBuiltTree(t)
	if err := TraverseBuiltJourneys(built, m); err != nil {
		t.Fatal(err)
	}
	if len(m.Journeys) != 22 {
		t.Fatalf("journey cardinality: got %d, want 22", len(m.Journeys))
	}
	for _, journey := range m.Journeys {
		t.Run(journey.GlobalKey, func(t *testing.T) {
			if err := traverseJourney(mustLoadDocuments(t, built), journey, TraverseOptions{}); err != nil {
				t.Fatal(err)
			}
		})
	}
	document := loadBuiltJourneyMutations(t)
	for _, mutation := range document.JLinkRemovals {
		t.Run("remove-"+mutation.JLink, func(t *testing.T) {
			copyDir := t.TempDir()
			if err := ApplyBuiltJourneyMutation(built, copyDir, mutation); err != nil {
				t.Fatal(err)
			}
			err := TraverseBuiltJourneys(copyDir, m)
			if err == nil {
				t.Fatalf("removing %s still passed", mutation.JLink)
			}
			if !strings.Contains(err.Error(), mutation.ExpectedError) {
				t.Fatalf("error %q does not name %q", err, mutation.ExpectedError)
			}
		})
	}
	for _, mutation := range document.JLinkWrongBinds {
		t.Run("wrong-bind-"+mutation.JLink, func(t *testing.T) {
			copyDir := t.TempDir()
			if err := ApplyBuiltJourneyMutation(built, copyDir, mutation); err != nil {
				t.Fatal(err)
			}
			err := TraverseBuiltJourneys(copyDir, m)
			if err == nil {
				t.Fatal("wrong-bound JLINK still passed")
			}
			if !strings.Contains(err.Error(), mutation.ExpectedError) {
				t.Fatalf("error %q does not name %q", err, mutation.ExpectedError)
			}
		})
	}
	for _, mutation := range document.CheatMutations {
		t.Run(mutation.Name, func(t *testing.T) {
			if mutation.InjectRuntime {
				copyDir := t.TempDir()
				if err := ApplyBuiltJourneyMutation(built, copyDir, mutation); err != nil {
					t.Fatal(err)
				}
				err := TraverseBuiltJourneys(copyDir, m)
				if err == nil || !strings.Contains(err.Error(), mutation.ExpectedError) {
					t.Fatalf("runtime mutation: %v", err)
				}
				return
			}
			err := TraverseBuiltJourneysWithOptions(built, m, TraverseOptions{
				DirectLoad:     mutation.DirectLoad,
				GlobalNav:      mutation.GlobalNav,
				ScreenshotOnly: mutation.ScreenshotOnly,
			})
			if err == nil || !strings.Contains(err.Error(), mutation.ExpectedError) {
				t.Fatalf("cheat %s: %v", mutation.Name, err)
			}
		})
	}
}

func TestWebsiteJourney_HasNoPublishedRuntime(t *testing.T) {
	built, m := mustAcceptedBuiltTree(t)
	documents := mustLoadDocuments(t, built)
	if err := AssertNoPublishedRuntime(documents); err != nil {
		t.Fatal(err)
	}
	if err := TraverseBuiltJourneys(built, m); err != nil {
		t.Fatal(err)
	}
	copyDir := t.TempDir()
	if err := ApplyBuiltJourneyMutation(built, copyDir, BuiltJourneyMutation{InjectRuntime: true}); err != nil {
		t.Fatal(err)
	}
	if err := AssertNoPublishedRuntime(mustLoadDocuments(t, copyDir)); err == nil {
		t.Fatal("accepted a published runtime")
	}
}

func TestWebsiteJourney_CAP014DualIdentityAnchorPasses(t *testing.T) {
	built, m := mustAcceptedBuiltTree(t)
	documents := mustLoadDocuments(t, built)
	identity := m.Journey("CAP-014/@UJ-001").DualIdentity
	if err := AssertCAP014DualIdentity(documents, identity); err != nil {
		t.Fatal(err)
	}
	if identity.Href != "/contributing/#external-ownership" {
		t.Fatalf("dual-identity href = %q", identity.Href)
	}
}

func TestWebsiteJourney_RejectsCAP014DualIdentityAnchorMatrix(t *testing.T) {
	document := loadBuiltJourneyMutations(t)
	built, m := mustAcceptedBuiltTree(t)
	for _, mutation := range document.CAP014Mutations {
		t.Run(mutation.Name, func(t *testing.T) {
			if mutation.ParseSourceMetadata {
				err := TraverseBuiltJourneysWithOptions(built, m, TraverseOptions{ParseSourceMetadata: true})
				if err == nil || !strings.Contains(err.Error(), "CAP-014/@UJ-001") || !strings.Contains(err.Error(), "JLINK-024") || !strings.Contains(err.Error(), "BOUNDARY-005") {
					t.Fatalf("source canonicalize: %v", err)
				}
				return
			}
			copyDir := t.TempDir()
			if err := ApplyBuiltJourneyMutation(built, copyDir, mutation); err != nil {
				t.Fatal(err)
			}
			err := TraverseBuiltJourneys(copyDir, m)
			if err == nil {
				t.Fatal("accepted invalid CAP-014 dual-identity tree")
			}
			for _, token := range []string{"CAP-014/@UJ-001", "JLINK-024", "BOUNDARY-005", mutation.ExpectedError} {
				if !strings.Contains(err.Error(), token) {
					t.Fatalf("error %q does not name %q", err, token)
				}
			}
		})
	}
}

func cloneJourney(journey MappedJourney) MappedJourney {
	cloned := journey
	cloned.Hops = append([]string(nil), journey.Hops...)
	cloned.JLinks = append([]string(nil), journey.JLinks...)
	cloned.Obligations = append([]MappedObligation(nil), journey.Obligations...)
	for i, obligation := range cloned.Obligations {
		cloned.Obligations[i].URLTemplates = append([]string(nil), obligation.URLTemplates...)
		cloned.Obligations[i].Descriptors = append([]GeneratedDescriptor(nil), obligation.Descriptors...)
		cloned.Obligations[i].RenderedAnchors = append([]string(nil), obligation.RenderedAnchors...)
	}
	return cloned
}

func mustLoadDocuments(t *testing.T, built string) map[string]string {
	t.Helper()
	documents, err := LoadBuiltDocuments(built)
	if err != nil {
		t.Fatal(err)
	}
	return documents
}

func TestWebsiteJourney_BuiltTraversalRejectsObligationCheats(t *testing.T) {
	built, m := mustAcceptedBuiltTree(t)
	documents := mustLoadDocuments(t, built)
	t.Run("missing-built-tree", func(t *testing.T) {
		if err := TraverseBuiltJourneys(t.TempDir(), m); err == nil {
			t.Fatal("accepted a missing built tree")
		}
	})
	t.Run("source-metadata-in-tree", func(t *testing.T) {
		copyDir := t.TempDir()
		if err := copyBuiltTree(built, copyDir); err != nil {
			t.Fatal(err)
		}
		if err := rewriteRoute(copyDir, "/", func(body string) string {
			return strings.Replace(body, "</main>", `<!-- backstop-claim: CLAIM-005 --></main>`, 1)
		}); err != nil {
			t.Fatal(err)
		}
		err := TraverseBuiltJourneys(copyDir, m)
		if err == nil || !strings.Contains(err.Error(), "CAP-014/@UJ-001") || !strings.Contains(err.Error(), "JLINK-024") || !strings.Contains(err.Error(), "BOUNDARY-005") {
			t.Fatalf("source metadata: %v", err)
		}
	})
	t.Run("hop-cardinality", func(t *testing.T) {
		journey := m.Journey("CAP-004/@UJ-001")
		journey.Hops = journey.Hops[:1]
		if err := traverseJourney(documents, journey, TraverseOptions{}); err == nil || !strings.Contains(err.Error(), "CAP-004/@UJ-001") {
			t.Fatalf("cardinality: %v", err)
		}
	})
	t.Run("invalid-hop", func(t *testing.T) {
		journey := m.Journey("CAP-004/@UJ-001")
		journey.Hops = []string{"http://[::1]:namedport", "/evaluate/#failure-fit"}
		if err := traverseJourney(documents, journey, TraverseOptions{}); err == nil || !strings.Contains(err.Error(), "CAP-004/@UJ-001") {
			t.Fatalf("invalid hop: %v", err)
		}
	})
	t.Run("missing-start-anchor", func(t *testing.T) {
		journey := m.Journey("CAP-004/@UJ-001")
		journey.Hops = []string{"/#Define-work", "/evaluate/#failure-fit"}
		if err := traverseJourney(documents, journey, TraverseOptions{}); err == nil || !strings.Contains(err.Error(), "CAP-004/@UJ-001") {
			t.Fatalf("case-sensitive start: %v", err)
		}
	})
	t.Run("copied-prose", func(t *testing.T) {
		journey := cloneJourney(m.Journey("CAP-004/@UJ-001"))
		journey.Obligations[0].CopiedProse = true
		if err := traverseJourney(documents, journey, TraverseOptions{}); err == nil || !strings.Contains(err.Error(), "integration consumer") {
			t.Fatalf("copied prose: %v", err)
		}
	})
	t.Run("missing-evidence", func(t *testing.T) {
		journey := cloneJourney(m.Journey("CAP-004/@UJ-001"))
		journey.Obligations[0].OwnerAnchor = "missing-evidence"
		if err := traverseJourney(documents, journey, TraverseOptions{}); err == nil || !strings.Contains(err.Error(), "CLAIM-017") {
			t.Fatalf("missing evidence: %v", err)
		}
	})
	t.Run("wrong-boundary-state", func(t *testing.T) {
		journey := cloneJourney(m.Journey("CAP-004/@UJ-001"))
		journey.Obligations[1].State = "supported"
		if err := traverseJourney(documents, journey, TraverseOptions{}); err == nil || !strings.Contains(err.Error(), "boundary state") {
			t.Fatalf("boundary state: %v", err)
		}
	})
	t.Run("missing-generated-region", func(t *testing.T) {
		journey := cloneJourney(m.Journey("CAP-011/@UJ-001"))
		for i := range journey.Obligations {
			if journey.Obligations[i].Kind == "generated" {
				journey.Obligations[i].JobID = "missing-job"
			}
		}
		if err := traverseJourney(documents, journey, TraverseOptions{}); err == nil || !strings.Contains(err.Error(), "missing-job") {
			t.Fatalf("generated region: %v", err)
		}
	})
	t.Run("resolved-site-commit", func(t *testing.T) {
		journey := cloneJourney(m.Journey("CAP-011/@UJ-001"))
		for i := range journey.Obligations {
			if journey.Obligations[i].Kind == "generated" {
				journey.Obligations[i].URLTemplates = []string{"https://example.invalid/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
			}
		}
		if err := traverseJourney(documents, journey, TraverseOptions{}); err == nil || !strings.Contains(err.Error(), "SITE-COMMIT") {
			t.Fatalf("resolved template: %v", err)
		}
	})
	t.Run("missing-owner-verdict", func(t *testing.T) {
		journey := cloneJourney(m.Journey("CAP-011/@UJ-001"))
		for i := range journey.Obligations {
			if journey.Obligations[i].Kind == "generated" {
				journey.Obligations[i].SiteIdentity = ""
			}
		}
		if err := traverseJourney(documents, journey, TraverseOptions{}); err == nil || !strings.Contains(err.Error(), "owner verdict") {
			t.Fatalf("owner verdict: %v", err)
		}
	})
	t.Run("link-off-first-route", func(t *testing.T) {
		journey := m.Journey("CAP-004/@UJ-001")
		journey.Hops = []string{"/evaluate/#failure-fit", "/#define-work"}
		if err := traverseJourney(documents, journey, TraverseOptions{}); err == nil || !strings.Contains(err.Error(), "first-route") {
			t.Fatalf("off-route link: %v", err)
		}
	})
	t.Run("unknown-boundary", func(t *testing.T) {
		journey := cloneJourney(m.Journey("CAP-004/@UJ-001"))
		journey.Obligations[1].BoundaryID = "BOUNDARY-999"
		if err := traverseJourney(documents, journey, TraverseOptions{}); err == nil || !strings.Contains(err.Error(), "BOUNDARY-999") {
			t.Fatalf("unknown boundary: %v", err)
		}
	})
	t.Run("write-unknown-route", func(t *testing.T) {
		broken := m
		broken.Journeys = []MappedJourney{{
			GlobalKey: "CAP-004/@UJ-001",
			Hops:      []string{"/missing/#anchor"},
			JLinks:    []string{"JLINK-001"},
		}}
		if err := WriteAcceptedBuiltTree(t.TempDir(), broken); err == nil {
			t.Fatal("accepted an unknown built route")
		}
	})
	t.Run("html-helpers", func(t *testing.T) {
		if SourceMetadataPresent(documents) {
			t.Fatal("accepted tree must not carry source metadata")
		}
		if !HasGeneratedRegion(documents["/pack/examples/"], "installed-pack-catalog") {
			t.Fatal("accepted tree missing generated region")
		}
		route, fragment, err := ParseHop("/status/#adjacent-guidance")
		if err != nil || route != "/status/" || fragment != "adjacent-guidance" {
			t.Fatalf("ParseHop: %s %s %v", route, fragment, err)
		}
		if _, _, err := ParseHop("http://[::1]:namedport"); err == nil {
			t.Fatal("ParseHop accepted an invalid hop")
		}
		if DocumentHasID(documents["/"], "Define-work") {
			t.Fatal("DocumentHasID must be case-sensitive")
		}
		if !DocumentHasID(documents["/"], "") {
			t.Fatal("empty fragment must accept a present document")
		}
		if replaceJourneyHref("<p>no-link</p>", "JLINK-001", "/x") != "<p>no-link</p>" {
			t.Fatal("replaceJourneyHref must no-op without the JLINK")
		}
		if err := writePage(nil, "x"); err == nil {
			t.Fatal("writePage accepted a nil buffer")
		}
		if _, err := pageFor(map[string]*strings.Builder{}, "/"); err == nil {
			t.Fatal("pageFor accepted a missing route")
		}
	})
}
