package main

import (
	"fmt"
	"os"
	"strings"
)

type JourneyMutation struct {
	Class                      string
	Target                     string
	ExpectedError              string
	JobID                      string `yaml:"job_id"`
	RemoveRegion               bool   `yaml:"remove_region"`
	RemoveJobID                bool   `yaml:"remove_job_id"`
	RemoveOwnerAnchor          bool   `yaml:"remove_owner_anchor"`
	DropBeginMarker            bool   `yaml:"drop_begin_marker"`
	DropEndMarker              bool   `yaml:"drop_end_marker"`
	DropDigest                 bool   `yaml:"drop_digest"`
	RemoveSourceLink           bool   `yaml:"remove_source_link"`
	RemoveSharedAnchor         bool   `yaml:"remove_shared_anchor"`
	RemoveJLinkIdentity        bool   `yaml:"remove_jlink_identity"`
	RemoveContinuationIdentity bool   `yaml:"remove_continuation_identity"`
	Href                       string `yaml:"href"`
	SplitIdentities            bool   `yaml:"split_identities"`
	DuplicateAnchor            bool   `yaml:"duplicate_anchor"`
	DropHref                   bool   `yaml:"drop_href"`
	MoveOutsideBoundary        bool   `yaml:"move_outside_boundary"`
	BeforeExplanation          bool   `yaml:"before_explanation"`
	AfterDenial                bool   `yaml:"after_denial"`
	EmptyExplanation           bool   `yaml:"empty_explanation"`
	EmptyLinkLabel             bool   `yaml:"empty_link_label"`
	EmptyDenial                bool   `yaml:"empty_denial"`
	Name                       string `yaml:"name"`
}

type MutationKill struct {
	Class   string
	Target  string
	Journey string
	Err     error
}

func KillJourneyMutation(builtRoot string, m WebsiteCapabilityMap, mutation JourneyMutation) MutationKill {
	dest, err := os.MkdirTemp("", "websitejourney-mutant-")
	if err != nil {
		return MutationKill{Class: mutation.Class, Target: mutation.Target, Err: err}
	}
	defer func() { _ = os.RemoveAll(dest) }()
	mutatedMap := cloneMap(m)
	if err := ApplyJourneyMutation(builtRoot, dest, &mutatedMap, mutation); err != nil {
		return MutationKill{Class: mutation.Class, Target: mutation.Target, Err: err}
	}
	return diagnoseMutant(dest, mutatedMap, mutation)
}

func ApplyJourneyMutation(src, dest string, m *WebsiteCapabilityMap, mutation JourneyMutation) error {
	if mutation.Class == "cap014" {
		if err := ApplyBuiltJourneyMutation(src, dest, BuiltJourneyMutation{
			JLink:               cap014JLink(mutation),
			Href:                mutation.Href,
			SplitIdentities:     mutation.SplitIdentities,
			DuplicateAnchor:     mutation.DuplicateAnchor,
			DropHref:            mutation.DropHref,
			MoveOutsideBoundary: mutation.MoveOutsideBoundary,
			BeforeExplanation:   mutation.BeforeExplanation,
			AfterDenial:         mutation.AfterDenial,
			EmptyExplanation:    mutation.EmptyExplanation,
			LookalikeAnchor:     mutation.RemoveJLinkIdentity,
		}); err != nil {
			return err
		}
		return applyCAP014Extra(dest, mutation)
	}
	if err := copyBuiltTree(src, dest); err != nil {
		return err
	}
	switch mutation.Class {
	case "route":
		path := BuiltRoutePath(dest, mutation.Target)
		return os.Remove(path)
	case "evidence":
		return rewriteRoute(dest, evidenceRoute(mutation.Target), func(body string) string {
			return strings.ReplaceAll(body, `id="`+evidenceAnchor(mutation.Target)+`"`, `id="removed-`+evidenceAnchor(mutation.Target)+`"`)
		})
	case "evidence-source":
		return rewriteRoute(dest, evidenceRoute(mutation.Target), func(body string) string {
			if strings.Contains(body, "data-generated-source-link") {
				return strings.ReplaceAll(body, "data-generated-source-link", "data-removed-source-link")
			}
			return strings.ReplaceAll(body, `id="`+evidenceAnchor(mutation.Target)+`"`, `id="removed-`+evidenceAnchor(mutation.Target)+`"`)
		})
	case "boundary":
		return rewriteAllLoaded(dest, func(body string) string {
			return strings.ReplaceAll(body, `data-boundary-id="`+mutation.Target+`"`, `data-removed-boundary="`+mutation.Target+`"`)
		})
	case "boundary-state":
		return rewriteAllLoaded(dest, func(body string) string {
			return strings.Replace(body, `data-boundary-id="`+mutation.Target+`" data-boundary-state="`, `data-boundary-id="`+mutation.Target+`" data-boundary-state="mutated-`, 1)
		})
	case "boundary-explanation":
		return rewriteAllLoaded(dest, func(body string) string {
			return strings.Replace(body, `data-boundary-explanation`, `data-removed-explanation`, 1)
		})
	case "boundary-continuation":
		return rewriteAllLoaded(dest, func(body string) string {
			return strings.Replace(body, ` data-boundary-continuation`, "", 1)
		})
	case "boundary-denial":
		return rewriteAllLoaded(dest, func(body string) string {
			return strings.Replace(body, `data-boundary-guarantee-denial`, `data-removed-denial`, 1)
		})
	case "prerequisite":
		*m = ApplyDependencyVerdictMutation(*m, DependencyVerdictMutation{OmitID: mutation.Target})
		return nil
	case "generated":
		applyGeneratedMapMutation(m, mutation)
		return applyGeneratedTreeMutation(dest, mutation)
	default:
		return fmt.Errorf("unknown mutation class %s", mutation.Class)
	}
}

func diagnoseMutant(dest string, m WebsiteCapabilityMap, mutation JourneyMutation) MutationKill {
	kill := MutationKill{Class: mutation.Class, Target: mutation.Target}
	if mutation.Class == "prerequisite" {
		tree, err := LoadCapabilityTree(m.Root)
		if err != nil {
			kill.Err = err
			return kill
		}
		_, err = EvaluatePrerequisites(m, tree, FreshZeroPrerequisiteRunner())
		if err == nil {
			kill.Err = fmt.Errorf("%s: %s: still passed", mutation.Class, mutation.Target)
			return kill
		}
		kill.Err = fmt.Errorf("%s: %s: %v", mutation.Class, mutation.Target, err)
		return kill
	}
	documents, loadErr := LoadBuiltDocuments(dest)
	if loadErr != nil {
		for _, journey := range m.Journeys {
			if journeyUsesRoute(journey, mutation.Target) || strings.Contains(loadErr.Error(), journey.GlobalKey) {
				kill.Journey = journey.GlobalKey
				kill.Err = fmt.Errorf("%s: %s: %s: %v", mutation.Class, mutation.Target, journey.GlobalKey, loadErr)
				return kill
			}
		}
		if len(m.Journeys) > 0 {
			kill.Journey = m.Journeys[0].GlobalKey
			kill.Err = fmt.Errorf("%s: %s: %s: %v", mutation.Class, mutation.Target, m.Journeys[0].GlobalKey, loadErr)
			return kill
		}
		kill.Err = loadErr
		return kill
	}
	for _, journey := range m.Journeys {
		if err := inspectJourneyMutation(documents, journey, mutation); err != nil {
			kill.Journey = journey.GlobalKey
			kill.Err = fmt.Errorf("%s: %s: %v", mutation.Class, mutation.Target, err)
			return kill
		}
	}
	kill.Err = fmt.Errorf("%s: %s: mutant still passed", mutation.Class, mutation.Target)
	return kill
}

func inspectJourneyMutation(documents map[string]string, journey MappedJourney, mutation JourneyMutation) error {
	if mutation.Class == "generated" {
		if err := assertGeneratedConsumer(journey); err != nil {
			return err
		}
	}
	if mutation.Class == "cap014" && journey.GlobalKey == "CAP-014/@UJ-001" {
		if mutation.EmptyLinkLabel {
			links := FindRenderedJourneyLinks(documents, "JLINK-024")
			if len(links) == 1 && links[0].Text == "" {
				return fmt.Errorf("CAP-014/@UJ-001: JLINK-024 BOUNDARY-005 visible bytes are empty")
			}
		}
		if mutation.EmptyDenial {
			boundary, err := FindRenderedBoundary(documents, "BOUNDARY-005")
			if err == nil && boundary.Denial == "" {
				return fmt.Errorf("CAP-014/@UJ-001: JLINK-024 BOUNDARY-005 visible bytes are empty")
			}
		}
		if mutation.RemoveSharedAnchor || mutation.RemoveContinuationIdentity {
			return ApplyBuiltJourneyMutationInspect(documents, mutation)
		}
	}
	return traverseJourney(documents, journey, TraverseOptions{})
}

func ApplyBuiltJourneyMutationInspect(documents map[string]string, mutation JourneyMutation) error {
	if mutation.RemoveSharedAnchor {
		if len(FindRenderedJourneyLinks(documents, "JLINK-024")) == 0 {
			return fmt.Errorf("CAP-014/@UJ-001: JLINK-024 BOUNDARY-005 shared anchor is missing")
		}
	}
	if mutation.RemoveContinuationIdentity {
		links := FindRenderedJourneyLinks(documents, "JLINK-024")
		if len(links) == 1 && !links[0].Continuation {
			return fmt.Errorf("CAP-014/@UJ-001: JLINK-024 BOUNDARY-005 continuation identity is missing")
		}
	}
	return AssertCAP014DualIdentity(documents, DualIdentity{JourneyLinkID: "JLINK-024", BoundaryID: "BOUNDARY-005", Href: "/contributing/#external-ownership"})
}

func assertGeneratedConsumer(journey MappedJourney) error {
	for _, job := range journey.GeneratedObligations() {
		if job.TruthBeginMarker == "" || job.TruthEndMarker == "" || job.SourcesBeginMarker == "" || job.SourcesEndMarker == "" || job.SourceDigest == "" {
			return fmt.Errorf("%s: generated %s marker or digest is missing", journey.GlobalKey, job.JobID)
		}
	}
	return nil
}

func applyGeneratedMapMutation(m *WebsiteCapabilityMap, mutation JourneyMutation) {
	for i := range m.Journeys {
		for j := range m.Journeys[i].Obligations {
			obligation := &m.Journeys[i].Obligations[j]
			if obligation.Kind != "generated" || obligation.JobID != mutation.JobID {
				continue
			}
			if mutation.DropBeginMarker {
				obligation.TruthBeginMarker = ""
			}
			if mutation.DropEndMarker {
				obligation.TruthEndMarker = ""
			}
			if mutation.DropDigest {
				obligation.SourceDigest = ""
			}
			if mutation.RemoveOwnerAnchor {
				obligation.OwnerAnchor = ""
			}
		}
	}
}

func applyGeneratedTreeMutation(dest string, mutation JourneyMutation) error {
	return rewriteAllLoaded(dest, func(body string) string {
		if mutation.RemoveRegion {
			body = strings.ReplaceAll(body, `data-generated-region="" data-product-truth-job="`+mutation.JobID+`"`, `data-removed-region="`+mutation.JobID+`"`)
		}
		if mutation.RemoveJobID {
			body = strings.ReplaceAll(body, `data-product-truth-job="`+mutation.JobID+`"`, `data-product-truth-job=""`)
		}
		if mutation.RemoveSourceLink {
			body = strings.ReplaceAll(body, "data-generated-source-link", "data-removed-source-link")
		}
		if mutation.RemoveOwnerAnchor {
			body = strings.ReplaceAll(body, `id="`+mutation.JobID+`"`, `id="removed-`+mutation.JobID+`"`)
		}
		return body
	})
}

func applyCAP014Extra(dest string, mutation JourneyMutation) error {
	return rewriteRoute(dest, "/status/", func(body string) string {
		if mutation.EmptyLinkLabel {
			body = strings.Replace(body, `>owner-accepted-continuation<`, `><`, 1)
		}
		if mutation.EmptyDenial {
			body = strings.Replace(body, `>owner-accepted-guarantee-denial<`, `><`, 1)
		}
		if mutation.RemoveContinuationIdentity {
			body = strings.Replace(body, ` data-boundary-continuation`, "", 1)
		}
		return body
	})
}

func rewriteAllLoaded(dest string, edit func(string) string) error {
	documents, err := LoadBuiltDocuments(dest)
	if err != nil {
		return err
	}
	return rewriteAllRoutes(dest, documents, edit)
}

func journeyUsesRoute(journey MappedJourney, route string) bool {
	for _, hop := range journey.Hops {
		hopRoute, _, err := ParseHop(hop)
		if err == nil && hopRoute == route {
			return true
		}
	}
	for _, obligation := range journey.Obligations {
		if obligation.OwnerRoute == route {
			return true
		}
	}
	return false
}

func evidenceRoute(target string) string {
	route, _, _ := strings.Cut(target, "#")
	if route == "" {
		return "/"
	}
	if !strings.HasSuffix(route, "/") && route != "/" {
		route += "/"
	}
	return route
}

func evidenceAnchor(target string) string {
	_, anchor, _ := strings.Cut(target, "#")
	return anchor
}

func cap014JLink(mutation JourneyMutation) string {
	if mutation.Href != "" || mutation.RemoveSharedAnchor || mutation.RemoveJLinkIdentity {
		return "JLINK-024"
	}
	return ""
}

func RouteMutations() []JourneyMutation {
	var mutations []JourneyMutation
	for _, route := range CanonicalBuiltRoutes() {
		mutations = append(mutations, JourneyMutation{Class: "route", Target: route})
	}
	return mutations
}

func EvidenceMutations(m WebsiteCapabilityMap) []JourneyMutation {
	seen := map[string]bool{}
	var mutations []JourneyMutation
	for _, journey := range m.Journeys {
		for _, obligation := range journey.Obligations {
			if obligation.Kind != "evidence" || obligation.OwnerAnchor == "" {
				continue
			}
			target := strings.TrimSuffix(obligation.OwnerRoute, "/") + "/#" + obligation.OwnerAnchor
			if obligation.OwnerRoute == "/" {
				target = "/#" + obligation.OwnerAnchor
			}
			if seen[target] {
				continue
			}
			seen[target] = true
			mutations = append(mutations, JourneyMutation{Class: "evidence", Target: target})
			mutations = append(mutations, JourneyMutation{Class: "evidence-source", Target: target})
		}
	}
	return mutations
}

func BoundaryMutations(m WebsiteCapabilityMap) []JourneyMutation {
	seen := map[string]bool{}
	var mutations []JourneyMutation
	for _, journey := range m.Journeys {
		for _, obligation := range journey.Obligations {
			if obligation.Kind != "boundary" || seen[obligation.BoundaryID] {
				continue
			}
			seen[obligation.BoundaryID] = true
			mutations = append(mutations, JourneyMutation{Class: "boundary", Target: obligation.BoundaryID})
			mutations = append(mutations, JourneyMutation{Class: "boundary-state", Target: obligation.BoundaryID})
			mutations = append(mutations, JourneyMutation{Class: "boundary-explanation", Target: obligation.BoundaryID})
			if obligation.BoundaryID == "BOUNDARY-005" {
				mutations = append(mutations, JourneyMutation{Class: "boundary-continuation", Target: obligation.BoundaryID})
				mutations = append(mutations, JourneyMutation{Class: "boundary-denial", Target: obligation.BoundaryID})
			}
		}
	}
	return mutations
}
