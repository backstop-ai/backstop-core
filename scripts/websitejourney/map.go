package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

var sourceDigestPattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

type ownerClaim struct {
	ID     string
	Type   string
	Route  string
	Anchor string
}

type ownerBoundary struct {
	ID     string
	State  string
	Route  string
	Anchor string
	Claim  string
}

func LoadWebsiteCapabilityMap(root string) (WebsiteCapabilityMap, error) {
	data, err := os.ReadFile(filepath.Join(root, "docs", "_data", "website-capability-map.yml"))
	if err != nil {
		return WebsiteCapabilityMap{}, fmt.Errorf("read website-capability-map.yml: %w", err)
	}
	var document WebsiteCapabilityMap
	if err := yaml.Unmarshal(data, &document); err != nil {
		return WebsiteCapabilityMap{}, fmt.Errorf("parse website-capability-map.yml: %w", err)
	}
	document.Root = root
	return document, nil
}

func ValidateEvidenceAndBoundaryBindings(m WebsiteCapabilityMap) error {
	if m.PredecessorVersions["SPEC-072"] != "1.0.5" {
		return fmt.Errorf("SPEC-072: predecessor version %q, want 1.0.5", m.PredecessorVersions["SPEC-072"])
	}
	if m.PredecessorVersions["SPEC-074"] != "1.0.4" {
		return fmt.Errorf("SPEC-074: predecessor version %q, want 1.0.4", m.PredecessorVersions["SPEC-074"])
	}
	if m.PredecessorVersions["SPEC-075"] != "1.0.4" {
		return fmt.Errorf("SPEC-075: predecessor version %q, want 1.0.4", m.PredecessorVersions["SPEC-075"])
	}
	wantDeps := []string{"seed1-product-model", "seed2-documentation-semantics", "seed3-product-truth", "seed4-public-site"}
	if strings.Join(m.PrerequisiteIDs(), ",") != strings.Join(wantDeps, ",") {
		return fmt.Errorf("prerequisites = %v, want %v", m.PrerequisiteIDs(), wantDeps)
	}
	if strings.Join(m.AdoptionInstructionIDs(), ",") != "ADOPT-INSTALL,ADOPT-CONFIGURE,ADOPT-ENFORCE" {
		return fmt.Errorf("adoption refs = %v, want ADOPT-INSTALL,ADOPT-CONFIGURE,ADOPT-ENFORCE", m.AdoptionInstructionIDs())
	}
	expected := ExpectedWebsiteJourneys()
	if len(m.Journeys) != len(expected) {
		seen := map[string]bool{}
		for _, journey := range m.Journeys {
			seen[journey.GlobalKey] = true
		}
		for _, want := range expected {
			if !seen[want.GlobalKey] {
				return fmt.Errorf("%s: missing mapped journey", want.GlobalKey)
			}
		}
		for key := range seen {
			if !expectedJourneyKey(key) {
				return fmt.Errorf("%s: unexpected mapped journey", key)
			}
		}
		return fmt.Errorf("map journey cardinality: got %d, want %d", len(m.Journeys), len(expected))
	}
	claims, err := loadOwnerClaims(m.Root)
	if err != nil {
		return err
	}
	boundaries, err := loadOwnerBoundaries(m.Root)
	if err != nil {
		return err
	}
	for index, want := range expected {
		got := m.Journeys[index]
		if got.GlobalKey != want.GlobalKey {
			return fmt.Errorf("%s: out of matrix order, found %s", want.GlobalKey, got.GlobalKey)
		}
		if strings.Join(got.Hops, ",") != strings.Join(want.Hops, ",") {
			return fmt.Errorf("%s: hop sequence %v, want %v", want.GlobalKey, got.Hops, want.Hops)
		}
		if strings.Join(got.JLinks, ",") != strings.Join(want.JLinks, ",") {
			return fmt.Errorf("%s: JLINK binding %v, want %v", want.GlobalKey, got.JLinks, want.JLinks)
		}
		if err := validateSemanticObligations(got, claims, boundaries); err != nil {
			return err
		}
	}
	cap014 := m.Journey("CAP-014/@UJ-001")
	if cap014.DualIdentity.JourneyLinkID != "JLINK-024" || cap014.DualIdentity.BoundaryID != "BOUNDARY-005" || cap014.DualIdentity.Href != "/contributing/#external-ownership" {
		return fmt.Errorf("CAP-014/@UJ-001: dual identity must be one JLINK-024/BOUNDARY-005 owner relationship to /contributing/#external-ownership")
	}
	return nil
}

func validateSemanticObligations(journey MappedJourney, claims map[string]ownerClaim, boundaries map[string]ownerBoundary) error {
	states := map[string]bool{}
	for _, obligation := range journey.Obligations {
		switch obligation.Kind {
		case "evidence":
			if obligation.CopiedProse {
				return fmt.Errorf("%s: copied claim prose is prohibited", journey.GlobalKey)
			}
			if obligation.Inferred {
				return fmt.Errorf("%s: inferred evidence role is prohibited", journey.GlobalKey)
			}
			if obligation.Seed5Resolved {
				return fmt.Errorf("%s: Seed 5 must not resolve SITE-COMMIT or owner content", journey.GlobalKey)
			}
			claim, ok := claims[obligation.ClaimID]
			if !ok {
				return fmt.Errorf("%s: unknown claim_id %s", journey.GlobalKey, obligation.ClaimID)
			}
			if obligation.ClaimType != claim.Type {
				return fmt.Errorf("%s: claim_type %q does not match owner %s", journey.GlobalKey, obligation.ClaimType, claim.ID)
			}
			if obligation.OwnerRoute != claim.Route || obligation.OwnerAnchor != claim.Anchor {
				return fmt.Errorf("%s: evidence owner %s#%s does not match %s", journey.GlobalKey, obligation.OwnerRoute, obligation.OwnerAnchor, claim.ID)
			}
		case "boundary":
			if obligation.Inferred {
				return fmt.Errorf("%s: inferred boundary role is prohibited", journey.GlobalKey)
			}
			boundary, ok := boundaries[obligation.BoundaryID]
			if !ok {
				return fmt.Errorf("%s: unknown boundary_id %s", journey.GlobalKey, obligation.BoundaryID)
			}
			if obligation.State != boundary.State {
				return fmt.Errorf("%s: boundary state %q does not match %s", journey.GlobalKey, obligation.State, boundary.ID)
			}
			if obligation.ClaimID != boundary.Claim {
				return fmt.Errorf("%s: boundary claim %s does not match %s", journey.GlobalKey, obligation.ClaimID, boundary.ID)
			}
			if obligation.OwnerRoute != boundary.Route || obligation.OwnerAnchor != boundary.Anchor {
				return fmt.Errorf("%s: boundary owner does not match %s", journey.GlobalKey, boundary.ID)
			}
			states[obligation.State] = true
		case "generated":
			if obligation.Seed5Resolved {
				return fmt.Errorf("%s: Seed 5 must not resolve SITE-COMMIT", journey.GlobalKey)
			}
		default:
			return fmt.Errorf("%s: obligation kind %q is outside the closed set", journey.GlobalKey, obligation.Kind)
		}
	}
	if journey.GlobalKey == "CAP-006/@UJ-002" {
		for _, state := range []string{"supported", "limitation", "planned", "non-goal", "adjacent-guidance"} {
			if !states[state] {
				return fmt.Errorf("%s: missing rendered boundary state %s", journey.GlobalKey, state)
			}
		}
	}
	return nil
}

func ValidateGeneratedObligationMatrix(m WebsiteCapabilityMap) error {
	cap014 := m.Journey("CAP-014/@UJ-001")
	if cap014.DualIdentity.JourneyLinkID != "JLINK-024" || cap014.DualIdentity.BoundaryID != "BOUNDARY-005" || cap014.DualIdentity.Href != "/contributing/#external-ownership" {
		return fmt.Errorf("CAP-014/@UJ-001: dual identity must remain one JLINK-024/BOUNDARY-005 owner relationship")
	}
	for _, journey := range m.Journeys {
		jobs := journey.GeneratedJobIDs()
		switch journey.GlobalKey {
		case "CAP-011/@UJ-001":
			if strings.Join(jobs, ",") != "installed-pack-catalog" {
				return fmt.Errorf("%s: generated jobs %v, want installed-pack-catalog", journey.GlobalKey, jobs)
			}
		case "CAP-013/@UJ-002":
			if strings.Join(jobs, ",") != "cli-command-catalog,artifact-schema-catalog,installed-pack-catalog,release-history" {
				return fmt.Errorf("%s: generated jobs %v, want all four jobs", journey.GlobalKey, jobs)
			}
		default:
			if len(jobs) != 0 {
				return fmt.Errorf("%s: unexpected generated job %s", journey.GlobalKey, jobs[0])
			}
		}
		for _, job := range journey.GeneratedObligations() {
			if err := validateGeneratedJob(journey.GlobalKey, job); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateGeneratedJob(key string, job GeneratedObligation) error {
	if job.ReconstructEnvelope {
		return fmt.Errorf("%s: Seed 5 must not reconstruct a provenance envelope", key)
	}
	if !sourceDigestPattern.MatchString(job.SourceDigest) {
		return fmt.Errorf("%s: malformed source-level digest %q", key, job.SourceDigest)
	}
	if job.TruthBeginMarker == "" || job.TruthEndMarker == "" || job.SourcesBeginMarker == "" || job.SourcesEndMarker == "" {
		return fmt.Errorf("%s: missing SPEC-074 marker pair for %s", key, job.JobID)
	}
	if len(job.Descriptors) == 0 || len(job.URLTemplates) == 0 {
		return fmt.Errorf("%s: incomplete descriptor/template set for %s", key, job.JobID)
	}
	if job.SiteIdentity == "" || strings.Contains(job.SiteIdentity, "<SITE-COMMIT>") && !strings.Contains(job.SiteIdentity, "owner:") {
		if job.SiteIdentity == "" {
			return fmt.Errorf("%s: missing SPEC-075 site identity for %s", key, job.JobID)
		}
	}
	if len(job.RenderedAnchors) == 0 || job.ReconstructionVerdict == "" || job.RenderedDigest == "" {
		return fmt.Errorf("%s: incomplete SPEC-075 rendered contract for %s", key, job.JobID)
	}
	for _, template := range job.URLTemplates {
		if strings.Contains(template, "<SITE-COMMIT>") {
			continue
		}
		if job.JobID != "release-history" {
			return fmt.Errorf("%s: SITE-COMMIT was resolved in Seed 5 for %s", key, job.JobID)
		}
	}
	return nil
}

func ApplyMapBindingMutation(m WebsiteCapabilityMap, mutation MapBindingMutation) WebsiteCapabilityMap {
	cloned := cloneMap(m)
	if mutation.OmitKey != "" {
		filtered := cloned.Journeys[:0]
		for _, journey := range cloned.Journeys {
			if journey.GlobalKey != mutation.OmitKey {
				filtered = append(filtered, journey)
			}
		}
		cloned.Journeys = filtered
	}
	if mutation.ExtraKey != "" {
		cloned.Journeys = append(cloned.Journeys, MappedJourney{GlobalKey: mutation.ExtraKey})
	}
	if mutation.OverrideSpec != "" {
		if cloned.PredecessorVersions == nil {
			cloned.PredecessorVersions = map[string]string{}
		}
		cloned.PredecessorVersions[mutation.OverrideSpec] = mutation.OverrideVersion
	}
	if mutation.ResolveSiteCommit {
		for i := range cloned.Journeys {
			for j := range cloned.Journeys[i].Obligations {
				cloned.Journeys[i].Obligations[j].Seed5Resolved = true
			}
		}
	}
	if mutation.OmitAdoption != "" {
		filtered := cloned.AdoptionInstructions[:0]
		for _, id := range cloned.AdoptionInstructions {
			if id != mutation.OmitAdoption {
				filtered = append(filtered, id)
			}
		}
		cloned.AdoptionInstructions = filtered
	}
	for i := range cloned.Journeys {
		if mutation.Key != "" && cloned.Journeys[i].GlobalKey != mutation.Key {
			continue
		}
		if mutation.Key == "" && mutation.OverrideJobs == nil && mutation.OmitJob == "" && mutation.ExtraJob == "" && !mutation.ReconstructEnvelope && mutation.OverrideDigest == "" && !mutation.DropMarkers && !mutation.ResolveURLTemplate && !mutation.SplitDualIdentity && mutation.OverrideKind == "" && !mutation.CopyClaimStatement && !mutation.InferBoundaryFromProse && mutation.OverrideClaimType == "" && mutation.OverrideBoundaryState == "" {
			continue
		}
		if mutation.Key != "" && cloned.Journeys[i].GlobalKey != mutation.Key {
			continue
		}
		if mutation.SplitDualIdentity {
			cloned.Journeys[i].DualIdentity.JourneyLinkID = "JLINK-024"
			cloned.Journeys[i].DualIdentity.BoundaryID = ""
			cloned.Journeys[i].DualIdentity.Href = "/contributing/#external-ownership"
		}
		if mutation.OverrideJobs != nil {
			cloned.Journeys[i].Obligations = keepNonGenerated(cloned.Journeys[i].Obligations)
			for _, jobID := range mutation.OverrideJobs {
				cloned.Journeys[i].Obligations = append(cloned.Journeys[i].Obligations, stubGenerated(jobID))
			}
		}
		if mutation.OmitJob != "" {
			filtered := cloned.Journeys[i].Obligations[:0]
			for _, obligation := range cloned.Journeys[i].Obligations {
				if obligation.Kind == "generated" && obligation.JobID == mutation.OmitJob {
					continue
				}
				filtered = append(filtered, obligation)
			}
			cloned.Journeys[i].Obligations = filtered
		}
		if mutation.ExtraJob != "" {
			cloned.Journeys[i].Obligations = append(cloned.Journeys[i].Obligations, stubGenerated(mutation.ExtraJob))
		}
		for j := range cloned.Journeys[i].Obligations {
			obligation := &cloned.Journeys[i].Obligations[j]
			if mutation.OverrideKind != "" && obligation.Kind == "evidence" {
				obligation.Kind = mutation.OverrideKind
			}
			if mutation.CopyClaimStatement && obligation.Kind == "evidence" {
				obligation.CopiedProse = true
			}
			if mutation.InferBoundaryFromProse && obligation.Kind == "boundary" {
				obligation.Inferred = true
			}
			if mutation.OverrideClaimType != "" && obligation.Kind == "evidence" {
				obligation.ClaimType = mutation.OverrideClaimType
			}
			if mutation.OverrideBoundaryState != "" && obligation.Kind == "boundary" {
				obligation.State = mutation.OverrideBoundaryState
			}
			if obligation.Kind != "generated" {
				continue
			}
			if mutation.ReconstructEnvelope {
				obligation.ReconstructEnvelope = true
			}
			if mutation.OverrideDigest != "" {
				obligation.SourceDigest = mutation.OverrideDigest
			}
			if mutation.DropMarkers {
				obligation.TruthBeginMarker = ""
				obligation.TruthEndMarker = ""
				obligation.SourcesBeginMarker = ""
				obligation.SourcesEndMarker = ""
			}
			if mutation.ResolveURLTemplate {
				for k, template := range obligation.URLTemplates {
					obligation.URLTemplates[k] = strings.ReplaceAll(template, "<SITE-COMMIT>", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
				}
			}
		}
	}
	return cloned
}

func keepNonGenerated(obligations []MappedObligation) []MappedObligation {
	filtered := make([]MappedObligation, 0, len(obligations))
	for _, obligation := range obligations {
		if obligation.Kind != "generated" {
			filtered = append(filtered, obligation)
		}
	}
	return filtered
}

func stubGenerated(jobID string) MappedObligation {
	return MappedObligation{
		Kind:                  "generated",
		JobID:                 jobID,
		SourceDigest:          "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		TruthBeginMarker:      "BEGIN",
		TruthEndMarker:        "END",
		SourcesBeginMarker:    "SOURCES-BEGIN",
		SourcesEndMarker:      "SOURCES-END",
		Descriptors:           []GeneratedDescriptor{{Kind: "blob", CommitBinding: "site", Path: "backstop.yml"}},
		URLTemplates:          []string{"https://github.com/backstop-ai/backstop-core/blob/<SITE-COMMIT>/backstop.yml"},
		SiteIdentity:          "owner:SPEC-075:site-commit",
		RenderedAnchors:       []string{"a[data-generated-source-link]"},
		ReconstructionVerdict: "owner:SPEC-075:reconstruction",
		RenderedDigest:        "owner:SPEC-075:rendered-digest",
	}
}

func cloneMap(m WebsiteCapabilityMap) WebsiteCapabilityMap {
	cloned := m
	cloned.PredecessorVersions = map[string]string{}
	for key, value := range m.PredecessorVersions {
		cloned.PredecessorVersions[key] = value
	}
	cloned.Prerequisites = append([]MapPrerequisite(nil), m.Prerequisites...)
	cloned.AdoptionInstructions = append([]string(nil), m.AdoptionInstructions...)
	cloned.Journeys = make([]MappedJourney, len(m.Journeys))
	for i, journey := range m.Journeys {
		copyJourney := journey
		copyJourney.Hops = append([]string(nil), journey.Hops...)
		copyJourney.JLinks = append([]string(nil), journey.JLinks...)
		copyJourney.Obligations = make([]MappedObligation, len(journey.Obligations))
		for j, obligation := range journey.Obligations {
			copyOb := obligation
			copyOb.Descriptors = append([]GeneratedDescriptor(nil), obligation.Descriptors...)
			copyOb.URLTemplates = append([]string(nil), obligation.URLTemplates...)
			copyOb.RenderedAnchors = append([]string(nil), obligation.RenderedAnchors...)
			copyJourney.Obligations[j] = copyOb
		}
		cloned.Journeys[i] = copyJourney
	}
	return cloned
}

func loadOwnerClaims(root string) (map[string]ownerClaim, error) {
	data, err := os.ReadFile(filepath.Join(root, "docs", "_data", "evidence-inventory.yml"))
	if err != nil {
		return nil, fmt.Errorf("read evidence-inventory.yml: %w", err)
	}
	var document struct {
		Claims []struct {
			ClaimID   string `yaml:"claim_id"`
			ClaimType string `yaml:"claim_type"`
			Owner     struct {
				Route  string `yaml:"route"`
				Anchor string `yaml:"anchor"`
			} `yaml:"owner"`
		} `yaml:"claims"`
	}
	if err := yaml.Unmarshal(data, &document); err != nil {
		return nil, fmt.Errorf("parse evidence-inventory.yml: %w", err)
	}
	out := map[string]ownerClaim{}
	for _, claim := range document.Claims {
		out[claim.ClaimID] = ownerClaim{ID: claim.ClaimID, Type: claim.ClaimType, Route: claim.Owner.Route, Anchor: claim.Owner.Anchor}
	}
	return out, nil
}

func loadOwnerBoundaries(root string) (map[string]ownerBoundary, error) {
	data, err := os.ReadFile(filepath.Join(root, "docs", "_data", "product-model.yml"))
	if err != nil {
		return nil, fmt.Errorf("read product-model.yml: %w", err)
	}
	var document struct {
		Boundaries []struct {
			BoundaryID string `yaml:"boundary_id"`
			State      string `yaml:"state"`
			ClaimID    string `yaml:"claim_id"`
			Owner      struct {
				Route  string `yaml:"route"`
				Anchor string `yaml:"anchor"`
			} `yaml:"owner"`
		} `yaml:"boundaries"`
	}
	if err := yaml.Unmarshal(data, &document); err != nil {
		return nil, fmt.Errorf("parse product-model.yml: %w", err)
	}
	out := map[string]ownerBoundary{}
	for _, boundary := range document.Boundaries {
		out[boundary.BoundaryID] = ownerBoundary{ID: boundary.BoundaryID, State: boundary.State, Route: boundary.Owner.Route, Anchor: boundary.Owner.Anchor, Claim: boundary.ClaimID}
	}
	return out, nil
}
