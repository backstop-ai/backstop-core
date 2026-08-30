package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type TraverseOptions struct {
	DirectLoad          bool
	GlobalNav           bool
	ScreenshotOnly      bool
	ParseSourceMetadata bool
}

type BuiltJourneyMutation struct {
	Name                string `yaml:"name"`
	JLink               string `yaml:"jlink"`
	Href                string `yaml:"href"`
	ExpectedError       string `yaml:"expected_error"`
	SplitIdentities     bool   `yaml:"split_identities"`
	DuplicateAnchor     bool   `yaml:"duplicate_anchor"`
	DropHref            bool   `yaml:"drop_href"`
	MoveOutsideBoundary bool   `yaml:"move_outside_boundary"`
	BeforeExplanation   bool   `yaml:"before_explanation"`
	AfterDenial         bool   `yaml:"after_denial"`
	EmptyExplanation    bool   `yaml:"empty_explanation"`
	LookalikeAnchor     bool   `yaml:"lookalike_anchor"`
	ParseSourceMetadata bool   `yaml:"parse_source_metadata"`
	DirectLoad          bool   `yaml:"direct_load"`
	GlobalNav           bool   `yaml:"global_nav"`
	ScreenshotOnly      bool   `yaml:"screenshot_only"`
	InjectRuntime       bool   `yaml:"inject_runtime"`
}

func TraverseBuiltJourneys(builtRoot string, m WebsiteCapabilityMap) error {
	return TraverseBuiltJourneysWithOptions(builtRoot, m, TraverseOptions{})
}

func TraverseBuiltJourneysWithOptions(builtRoot string, m WebsiteCapabilityMap, opts TraverseOptions) error {
	if opts.ParseSourceMetadata {
		return fmt.Errorf("CAP-014/@UJ-001: JLINK-024 BOUNDARY-005 Seed 5 must not parse or canonicalize source metadata")
	}
	documents, err := LoadBuiltDocuments(builtRoot)
	if err != nil {
		return err
	}
	if err := AssertNoPublishedRuntime(documents); err != nil {
		return err
	}
	if SourceMetadataPresent(documents) {
		return fmt.Errorf("CAP-014/@UJ-001: JLINK-024 BOUNDARY-005 Seed 5 must not parse or canonicalize source metadata")
	}
	for _, journey := range m.Journeys {
		if err := traverseJourney(documents, journey, opts); err != nil {
			return err
		}
	}
	return nil
}

func AssertNoPublishedRuntime(documents map[string]string) error {
	if PublishedRuntimePresent(documents) {
		return fmt.Errorf("runtime: published application JavaScript is prohibited")
	}
	return nil
}

func AssertCAP014DualIdentity(documents map[string]string, identity DualIdentity) error {
	links := FindRenderedJourneyLinks(documents, identity.JourneyLinkID)
	if len(links) != 1 {
		return fmt.Errorf("CAP-014/@UJ-001: JLINK-024 BOUNDARY-005 want one dual-identity anchor, found %d", len(links))
	}
	link := links[0]
	if !link.Continuation || link.Href != identity.Href {
		return fmt.Errorf("CAP-014/@UJ-001: JLINK-024 BOUNDARY-005 href/continuation mismatch")
	}
	boundary, err := FindRenderedBoundary(documents, identity.BoundaryID)
	if err != nil {
		return fmt.Errorf("CAP-014/@UJ-001: %v", err)
	}
	if !strings.Contains(boundary.InnerHTML, `data-journey-link-id="JLINK-024"`) || !strings.Contains(boundary.InnerHTML, "data-boundary-continuation") {
		return fmt.Errorf("CAP-014/@UJ-001: JLINK-024 BOUNDARY-005 anchor is outside the callout")
	}
	explanationIdx := strings.Index(boundary.InnerHTML, "data-boundary-explanation")
	linkIdx := strings.Index(boundary.InnerHTML, `data-journey-link-id="JLINK-024"`)
	denialIdx := strings.Index(boundary.InnerHTML, "data-boundary-guarantee-denial")
	if explanationIdx < 0 || linkIdx < 0 || denialIdx < 0 || explanationIdx >= linkIdx || linkIdx >= denialIdx {
		return fmt.Errorf("CAP-014/@UJ-001: JLINK-024 BOUNDARY-005 explanation-link-denial order is wrong")
	}
	if boundary.Explanation == "" || link.Text == "" || boundary.Denial == "" {
		return fmt.Errorf("CAP-014/@UJ-001: JLINK-024 BOUNDARY-005 visible bytes are empty")
	}
	if strings.Count(boundary.InnerHTML, `data-journey-link-id="JLINK-024"`) != 1 {
		return fmt.Errorf("CAP-014/@UJ-001: JLINK-024 BOUNDARY-005 duplicate or lookalike anchor")
	}
	return nil
}

func WriteAcceptedBuiltTree(dest string, m WebsiteCapabilityMap) error {
	pages := map[string]*strings.Builder{}
	for _, route := range CanonicalBuiltRoutes() {
		pages[route] = &strings.Builder{}
		if err := writePage(pages[route], "<!doctype html><html><body><main id=\"main\">"); err != nil {
			return err
		}
	}
	writtenLinks := map[string]bool{}
	writtenBounds := map[string]bool{}
	writtenJobs := map[string]bool{}
	status, err := pageFor(pages, "/status/")
	if err != nil {
		return err
	}
	if err := writePage(status, `<aside data-boundary-id="BOUNDARY-005" data-boundary-state="adjacent-guidance"><p data-boundary-explanation>owner-accepted-boundary-explanation</p><a data-journey-link-id="JLINK-024" data-boundary-continuation href="/contributing/#external-ownership">owner-accepted-continuation</a><p data-boundary-guarantee-denial>owner-accepted-guarantee-denial</p></aside>`); err != nil {
		return err
	}
	writtenLinks["JLINK-024"] = true
	writtenBounds["BOUNDARY-005"] = true
	for _, journey := range m.Journeys {
		for _, hop := range journey.Hops {
			route, fragment, err := ParseHop(hop)
			if err != nil {
				return err
			}
			page, err := pageFor(pages, route)
			if err != nil {
				return err
			}
			if fragment != "" && !strings.Contains(page.String(), `id="`+fragment+`"`) {
				if err := writePage(page, `<section id="%s"></section>`, fragment); err != nil {
					return err
				}
			}
		}
		for index, jlink := range journey.JLinks {
			if writtenLinks[jlink] {
				continue
			}
			startRoute, _, err := ParseHop(journey.Hops[index])
			if err != nil {
				return err
			}
			href := journey.Hops[index+1]
			if jlink == "JLINK-024" {
				continue
			}
			start, err := pageFor(pages, startRoute)
			if err != nil {
				return err
			}
			if err := writePage(start, `<nav data-next-action><a data-journey-link-id="%s" href="%s">next</a></nav>`, jlink, href); err != nil {
				return err
			}
			writtenLinks[jlink] = true
		}
		for _, obligation := range journey.Obligations {
			page, err := pageFor(pages, obligation.OwnerRoute)
			if err != nil {
				if obligation.OwnerRoute == "" {
					continue
				}
				return err
			}
			if obligation.Kind == "evidence" && obligation.OwnerAnchor != "" {
				if !strings.Contains(page.String(), `id="`+obligation.OwnerAnchor+`"`) {
					if err := writePage(page, `<section id="%s"></section>`, obligation.OwnerAnchor); err != nil {
						return err
					}
				}
			}
			if obligation.Kind == "boundary" && !writtenBounds[obligation.BoundaryID] {
				if err := writePage(page, `<aside data-boundary-id="%s" data-boundary-state="%s"><p data-boundary-explanation>owner-accepted-boundary-explanation</p></aside>`, obligation.BoundaryID, obligation.State); err != nil {
					return err
				}
				writtenBounds[obligation.BoundaryID] = true
			}
			if obligation.Kind == "generated" && !writtenJobs[obligation.JobID] {
				if err := writePage(page, `<section data-generated-region="" data-product-truth-job="%s"><a data-generated-source-link href="https://github.com/backstop-ai/backstop-core"></a></section>`, obligation.JobID); err != nil {
					return err
				}
				writtenJobs[obligation.JobID] = true
			}
		}
	}
	for _, route := range CanonicalBuiltRoutes() {
		if err := writePage(pages[route], "</main></body></html>"); err != nil {
			return err
		}
		path := BuiltRoutePath(dest, route)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte(pages[route].String()), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func ApplyBuiltJourneyMutation(src, dest string, mutation BuiltJourneyMutation) error {
	if err := copyBuiltTree(src, dest); err != nil {
		return err
	}
	documents, err := LoadBuiltDocuments(dest)
	if err != nil {
		return err
	}
	if mutation.JLink != "" && mutation.Href == "" {
		if err := rewriteAllRoutes(dest, documents, func(body string) string {
			return strings.ReplaceAll(body, `data-journey-link-id="`+mutation.JLink+`"`, `data-removed-link="`+mutation.JLink+`"`)
		}); err != nil {
			return err
		}
	}
	if mutation.JLink != "" && mutation.Href != "" {
		if err := rewriteAllRoutes(dest, documents, func(body string) string {
			return replaceJourneyHref(body, mutation.JLink, mutation.Href)
		}); err != nil {
			return err
		}
	}
	if mutation.InjectRuntime {
		if err := rewriteRoute(dest, "/", func(body string) string {
			return strings.Replace(body, "</main>", `<script src="/app.js"></script></main>`, 1)
		}); err != nil {
			return err
		}
	}
	edits := []struct {
		apply bool
		edit  func(string) string
	}{
		{mutation.SplitIdentities, func(body string) string {
			body = strings.Replace(body, ` data-boundary-continuation`, "", 1)
			return strings.Replace(body, `</aside>`, `<a data-boundary-continuation href="/contributing/#external-ownership">other</a></aside>`, 1)
		}},
		{mutation.DuplicateAnchor, func(body string) string {
			return strings.Replace(body, `</aside>`, `<a data-journey-link-id="JLINK-024" data-boundary-continuation href="/contributing/#external-ownership">dup</a></aside>`, 1)
		}},
		{mutation.DropHref, func(body string) string {
			return strings.Replace(body, ` href="/contributing/#external-ownership"`, "", 1)
		}},
		{mutation.MoveOutsideBoundary, func(body string) string {
			body = strings.Replace(body, `<a data-journey-link-id="JLINK-024" data-boundary-continuation href="/contributing/#external-ownership">owner-accepted-continuation</a>`, "", 1)
			return strings.Replace(body, `</main>`, `<a data-journey-link-id="JLINK-024" data-boundary-continuation href="/contributing/#external-ownership">owner-accepted-continuation</a></main>`, 1)
		}},
		{mutation.BeforeExplanation, func(body string) string {
			return strings.Replace(body, `<p data-boundary-explanation>owner-accepted-boundary-explanation</p><a data-journey-link-id="JLINK-024" data-boundary-continuation href="/contributing/#external-ownership">owner-accepted-continuation</a>`, `<a data-journey-link-id="JLINK-024" data-boundary-continuation href="/contributing/#external-ownership">owner-accepted-continuation</a><p data-boundary-explanation>owner-accepted-boundary-explanation</p>`, 1)
		}},
		{mutation.AfterDenial, func(body string) string {
			return strings.Replace(body, `<a data-journey-link-id="JLINK-024" data-boundary-continuation href="/contributing/#external-ownership">owner-accepted-continuation</a><p data-boundary-guarantee-denial>owner-accepted-guarantee-denial</p>`, `<p data-boundary-guarantee-denial>owner-accepted-guarantee-denial</p><a data-journey-link-id="JLINK-024" data-boundary-continuation href="/contributing/#external-ownership">owner-accepted-continuation</a>`, 1)
		}},
		{mutation.EmptyExplanation, func(body string) string {
			return strings.Replace(body, `>owner-accepted-boundary-explanation<`, `><`, 1)
		}},
		{mutation.LookalikeAnchor, func(body string) string {
			return strings.Replace(body, `data-journey-link-id="JLINK-024"`, `data-journey-link-id="jlink-024"`, 1)
		}},
	}
	for _, item := range edits {
		if !item.apply {
			continue
		}
		if err := rewriteRoute(dest, "/status/", item.edit); err != nil {
			return err
		}
	}
	return nil
}

func traverseJourney(documents map[string]string, journey MappedJourney, opts TraverseOptions) error {
	if opts.DirectLoad || opts.GlobalNav || opts.ScreenshotOnly {
		return fmt.Errorf("%s: direct load, global nav, and screenshot tours are prohibited", journey.GlobalKey)
	}
	if journey.GlobalKey == "CAP-014/@UJ-001" {
		if err := AssertCAP014DualIdentity(documents, journey.DualIdentity); err != nil {
			return err
		}
	}
	if len(journey.JLinks)+1 != len(journey.Hops) {
		return fmt.Errorf("%s: hop/JLINK cardinality", journey.GlobalKey)
	}
	for index, jlink := range journey.JLinks {
		startRoute, startFrag, err := ParseHop(journey.Hops[index])
		if err != nil {
			return fmt.Errorf("%s: %w", journey.GlobalKey, err)
		}
		endRoute, endFrag, err := ParseHop(journey.Hops[index+1])
		if err != nil {
			return fmt.Errorf("%s: %w", journey.GlobalKey, err)
		}
		if !DocumentHasID(documents[startRoute], startFrag) {
			return fmt.Errorf("%s: missing case-sensitive start anchor %s", journey.GlobalKey, startFrag)
		}
		if !DocumentHasID(documents[endRoute], endFrag) {
			return fmt.Errorf("%s: missing case-sensitive destination anchor %s", journey.GlobalKey, endFrag)
		}
		links := FindRenderedJourneyLinks(documents, jlink)
		if len(links) != 1 {
			return fmt.Errorf("%s: rendered %s count %d", journey.GlobalKey, jlink, len(links))
		}
		link := links[0]
		if link.Document != startRoute {
			return fmt.Errorf("%s: %s is not on first-route %s", journey.GlobalKey, jlink, startRoute)
		}
		if link.Href != journey.Hops[index+1] {
			return fmt.Errorf("%s: %s href %q does not match hop", journey.GlobalKey, jlink, link.Href)
		}
	}
	for _, obligation := range journey.Obligations {
		if obligation.Inferred || obligation.CopiedProse || obligation.Seed5Resolved || obligation.ReconstructEnvelope {
			return fmt.Errorf("%s: Seed 5 must remain an integration consumer", journey.GlobalKey)
		}
		switch obligation.Kind {
		case "evidence":
			if !DocumentHasID(documents[obligation.OwnerRoute], obligation.OwnerAnchor) {
				return fmt.Errorf("%s: missing evidence anchor %s", journey.GlobalKey, obligation.ClaimID)
			}
		case "boundary":
			boundary, err := FindRenderedBoundary(documents, obligation.BoundaryID)
			if err != nil {
				return fmt.Errorf("%s: %v", journey.GlobalKey, err)
			}
			if boundary.State != obligation.State {
				return fmt.Errorf("%s: boundary state %q", journey.GlobalKey, boundary.State)
			}
		case "generated":
			if obligation.SiteIdentity != "owner:SPEC-075:site-commit" || obligation.ReconstructionVerdict != "owner:SPEC-075:reconstruction" {
				return fmt.Errorf("%s: generated obligation missing owner verdict for %s", journey.GlobalKey, obligation.JobID)
			}
			if !HasGeneratedRegion(documents[obligation.OwnerRoute], obligation.JobID) {
				return fmt.Errorf("%s: missing generated region %s", journey.GlobalKey, obligation.JobID)
			}
			for _, template := range obligation.URLTemplates {
				if strings.Contains(template, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa") {
					return fmt.Errorf("%s: Seed 5 resolved SITE-COMMIT for %s", journey.GlobalKey, obligation.JobID)
				}
			}
		}
	}
	return nil
}

func copyBuiltTree(src, dest string) error {
	documents, err := LoadBuiltDocuments(src)
	if err != nil {
		return err
	}
	for route, body := range documents {
		path := BuiltRoutePath(dest, route)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func rewriteAllRoutes(dest string, documents map[string]string, edit func(string) string) error {
	for route := range documents {
		if err := rewriteRoute(dest, route, edit); err != nil {
			return err
		}
	}
	return nil
}

func rewriteRoute(dest, route string, edit func(string) string) error {
	path := BuiltRoutePath(dest, route)
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return os.WriteFile(path, []byte(edit(string(data))), 0o644)
}

func replaceJourneyHref(body, jlink, href string) string {
	old := FindRenderedJourneyLinks(map[string]string{"doc": body}, jlink)
	if len(old) == 0 {
		return body
	}
	needle := `data-journey-link-id="` + jlink + `"`
	start := strings.Index(body, needle)
	if start < 0 {
		return body
	}
	open := strings.LastIndex(body[:start], "<a ")
	close := strings.Index(body[start:], ">")
	if open < 0 || close < 0 {
		return body
	}
	tag := body[open : start+close]
	updated := strings.Replace(tag, `href="`+old[0].Href+`"`, `href="`+href+`"`, 1)
	return body[:open] + updated + body[start+close:]
}

func writePage(page *strings.Builder, format string, args ...any) error {
	if page == nil {
		return fmt.Errorf("missing built page buffer")
	}
	_, err := fmt.Fprintf(page, format, args...)
	return err
}

func pageFor(pages map[string]*strings.Builder, route string) (*strings.Builder, error) {
	page, ok := pages[route]
	if !ok {
		return nil, fmt.Errorf("unknown built route %s", route)
	}
	return page, nil
}
