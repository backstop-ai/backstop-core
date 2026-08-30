package main

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	journeyLinkPattern = regexp.MustCompile(`<a\s+([^>]*data-journey-link-id="(JLINK-[0-9]{3})"[^>]*)>([^<]*)</a>`)
	attrPattern        = regexp.MustCompile(`([a-zA-Z0-9:-]+)="([^"]*)"`)
	idPattern          = regexp.MustCompile(`\sid="([^"]+)"`)
	scriptPattern      = regexp.MustCompile(`(?i)<script[\s>]`)
	sourceClaimPattern = regexp.MustCompile(`<!--\s*backstop-claim:`)
	sourceJLinkPattern = regexp.MustCompile(`<!--\s*backstop-journey-link:`)
)

type RenderedJourneyLink struct {
	ID           string
	Href         string
	Text         string
	Continuation bool
	Document     string
}

type RenderedBoundary struct {
	ID          string
	State       string
	Document    string
	InnerHTML   string
	Explanation string
	Denial      string
}

func BuiltRoutePath(root, route string) string {
	if route == "/" {
		return filepath.Join(root, "index.html")
	}
	return filepath.Join(root, strings.Trim(route, "/"), "index.html")
}

func CanonicalBuiltRoutes() []string {
	return []string{"/", "/evaluate/", "/model/", "/adopt/", "/use-cases/", "/packs/", "/extend/", "/reference/", "/status/", "/contributing/"}
}

func LoadBuiltDocuments(builtRoot string) (map[string]string, error) {
	documents := map[string]string{}
	for _, route := range CanonicalBuiltRoutes() {
		data, err := os.ReadFile(BuiltRoutePath(builtRoot, route))
		if err != nil {
			return nil, fmt.Errorf("%s: built document: %w", route, err)
		}
		documents[route] = string(data)
	}
	return documents, nil
}

func ParseHop(hop string) (route, fragment string, err error) {
	parsed, parseErr := url.Parse(hop)
	if parseErr != nil {
		return "", "", parseErr
	}
	route = parsed.Path
	if route == "" {
		route = "/"
	}
	if !strings.HasSuffix(route, "/") && route != "/" {
		route += "/"
	}
	return route, parsed.Fragment, nil
}

func DocumentHasID(document, id string) bool {
	if id == "" {
		return document != ""
	}
	for _, match := range idPattern.FindAllStringSubmatch(document, -1) {
		if match[1] == id {
			return true
		}
	}
	return false
}

func FindRenderedJourneyLinks(documents map[string]string, id string) []RenderedJourneyLink {
	var found []RenderedJourneyLink
	for route, document := range documents {
		for _, match := range journeyLinkPattern.FindAllStringSubmatch(document, -1) {
			if match[2] != id {
				continue
			}
			attrs := parseAttrs(match[1])
			found = append(found, RenderedJourneyLink{
				ID:           id,
				Href:         attrs["href"],
				Text:         match[3],
				Continuation: strings.Contains(match[1], "data-boundary-continuation"),
				Document:     route,
			})
		}
	}
	return found
}

func FindRenderedBoundary(documents map[string]string, id string) (RenderedBoundary, error) {
	pattern := regexp.MustCompile(`(?s)<aside\s+([^>]*data-boundary-id="` + regexp.QuoteMeta(id) + `"[^>]*)>(.*?)</aside>`)
	for route, document := range documents {
		match := pattern.FindStringSubmatch(document)
		if match == nil {
			continue
		}
		attrs := parseAttrs(match[1])
		return RenderedBoundary{
			ID:          id,
			State:       attrs["data-boundary-state"],
			Document:    route,
			InnerHTML:   match[2],
			Explanation: firstElementText(match[2], "data-boundary-explanation"),
			Denial:      firstElementText(match[2], "data-boundary-guarantee-denial"),
		}, nil
	}
	return RenderedBoundary{}, fmt.Errorf("%s: missing rendered boundary", id)
}

func PublishedRuntimePresent(documents map[string]string) bool {
	for _, document := range documents {
		if scriptPattern.MatchString(document) {
			return true
		}
		if strings.Contains(document, "application/javascript") {
			return true
		}
	}
	return false
}

func SourceMetadataPresent(documents map[string]string) bool {
	for _, document := range documents {
		if sourceClaimPattern.MatchString(document) || sourceJLinkPattern.MatchString(document) {
			return true
		}
	}
	return false
}

func HasGeneratedRegion(document, jobID string) bool {
	return strings.Contains(document, `data-product-truth-job="`+jobID+`"`) && strings.Contains(document, "data-generated-source-link")
}

func parseAttrs(raw string) map[string]string {
	attrs := map[string]string{}
	for _, match := range attrPattern.FindAllStringSubmatch(raw, -1) {
		attrs[match[1]] = match[2]
	}
	return attrs
}

func firstElementText(inner, attr string) string {
	pattern := regexp.MustCompile(`(?s)[^<]*` + regexp.QuoteMeta(attr) + `[^>]*>([^<]*)<`)
	match := pattern.FindStringSubmatch(inner)
	if match == nil {
		return ""
	}
	return strings.TrimSpace(match[1])
}
