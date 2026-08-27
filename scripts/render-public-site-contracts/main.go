package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type Finding struct {
	Phase    string
	Identity string
	Message  string
}

type owner struct {
	Route  string `yaml:"route"`
	Anchor string `yaml:"anchor"`
}

type journeyLink struct {
	ID                string `yaml:"link_id"`
	SourceRoute       string `yaml:"source_route"`
	SourceAnchor      string `yaml:"source_anchor"`
	DestinationRoute  string `yaml:"destination_route"`
	DestinationAnchor string `yaml:"destination_anchor"`
	Label             string `yaml:"label"`
}

type adoptionInstruction struct {
	ID            string `yaml:"instruction_id"`
	OwnerRoute    string `yaml:"owner_route"`
	OwnerAnchor   string `yaml:"owner_anchor"`
	CommandText   string `yaml:"command_text"`
	CommandSHA256 string `yaml:"command_sha256"`
}

type topology struct {
	JourneyLinks         []journeyLink         `yaml:"journey_links"`
	AdoptionInstructions []adoptionInstruction `yaml:"adoption_instructions"`
}

type evidenceClaim struct {
	ID         string `yaml:"claim_id"`
	Owner      owner  `yaml:"owner"`
	BoundaryID string `yaml:"boundary_id"`
}

type evidenceInventory struct {
	Claims []evidenceClaim `yaml:"claims"`
}

type boundary struct {
	ID              string `yaml:"boundary_id"`
	State           string `yaml:"state"`
	Owner           owner  `yaml:"owner"`
	ClaimID         string `yaml:"claim_id"`
	Explanation     string `yaml:"explanation_markdown"`
	GuaranteeDenial string `yaml:"guarantee_denial_markdown"`
	Continuation    *struct {
		JourneyLinkID string `yaml:"journey_link_id"`
		Route         string `yaml:"route"`
		Anchor        string `yaml:"anchor"`
		Label         string `yaml:"label"`
	} `yaml:"continuation"`
}

type productModel struct {
	Boundaries []boundary `yaml:"boundaries"`
}

var fullCommitPattern = regexp.MustCompile(`^[0-9a-f]{40}$`)

func routeFile(root, route string) string {
	if route == "/" {
		return filepath.Join(root, "index.html")
	}
	return filepath.Join(root, strings.Trim(route, "/"), "index.html")
}

func loadYAML(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(target); err != nil {
		return err
	}
	return nil
}

func replaceOnce(input, before, after, identity string) (string, error) {
	count := strings.Count(input, before)
	if count != 1 {
		return input, fmt.Errorf("%s: expected one binding, observed %d", identity, count)
	}
	return strings.Replace(input, before, after, 1), nil
}

func bindJourneyLinks(doc string, links []journeyLink, route string) (string, error) {
	for _, link := range links {
		if link.SourceRoute != route {
			continue
		}
		marker := "<!-- backstop-journey-link: " + link.ID + " -->"
		pattern := regexp.MustCompile(`(?s)` + regexp.QuoteMeta(marker) + `\s*<p><a href="([^"]+)">([^<]+)</a></p>`)
		matches := pattern.FindAllStringSubmatchIndex(doc, -1)
		if len(matches) != 1 {
			return doc, fmt.Errorf("%s route=%s anchor=%s: expected one rendered owner link, observed %d", link.ID, route, link.SourceAnchor, len(matches))
		}
		match := pattern.FindStringSubmatch(doc)
		expectedHref := link.DestinationRoute + "#" + link.DestinationAnchor
		if match[1] != expectedHref || html.UnescapeString(match[2]) != link.Label {
			return doc, fmt.Errorf("%s route=%s anchor=%s: expected href=%q label=%q, observed href=%q label=%q", link.ID, route, link.SourceAnchor, expectedHref, link.Label, match[1], html.UnescapeString(match[2]))
		}
		extra := ""
		if link.ID == "JLINK-024" {
			extra = " data-boundary-continuation"
		}
		replacement := fmt.Sprintf(`<p><a data-journey-link-id="%s"%s href="%s">%s</a></p>`, html.EscapeString(link.ID), extra, html.EscapeString(expectedHref), match[2])
		doc = pattern.ReplaceAllString(doc, replacement)
	}
	return doc, nil
}

func bindClaims(doc string, claims []evidenceClaim, boundaries map[string]boundary, route string) (string, error) {
	for _, claim := range claims {
		if claim.Owner.Route != route {
			continue
		}
		opening := "<!-- backstop-claim: " + claim.ID + " -->"
		closing := "<!-- /backstop-claim -->"
		start := strings.Index(doc, opening)
		if start < 0 {
			return doc, fmt.Errorf("%s route=%s anchor=%s: opening marker missing", claim.ID, route, claim.Owner.Anchor)
		}
		endRelative := strings.Index(doc[start:], closing)
		if endRelative < 0 {
			return doc, fmt.Errorf("%s route=%s anchor=%s: closing marker missing", claim.ID, route, claim.Owner.Anchor)
		}
		end := start + endRelative + len(closing)
		closingNeedle := closing
		if strings.HasPrefix(doc[end:], "</p>") {
			end += len("</p>")
			closingNeedle += "</p>"
		}
		segment := doc[start:end]
		if claim.BoundaryID == "" {
			segment, _ = replaceOnce(segment, opening, `<article data-evidence-card data-claim-id="`+html.EscapeString(claim.ID)+`">`, claim.ID)
			segment, _ = replaceOnce(segment, closingNeedle, `</article>`, claim.ID)
			doc = doc[:start] + segment + doc[end:]
			continue
		}
		boundaryRecord, ok := boundaries[claim.BoundaryID]
		if !ok || boundaryRecord.ClaimID != claim.ID || boundaryRecord.Owner != claim.Owner {
			return doc, fmt.Errorf("%s: boundary owner binding is absent or inconsistent", claim.BoundaryID)
		}
		openingTag := fmt.Sprintf(`<aside data-boundary-callout data-boundary-id="%s" data-boundary-state="%s">`, html.EscapeString(boundaryRecord.ID), html.EscapeString(boundaryRecord.State))
		segment, _ = replaceOnce(segment, opening, openingTag, boundaryRecord.ID)
		if !strings.Contains(segment, "<p>") {
			return doc, fmt.Errorf("%s: explanation paragraph missing", boundaryRecord.ID)
		}
		segment = strings.Replace(segment, "<p>", `<p data-boundary-explanation>`, 1)
		if boundaryRecord.Continuation != nil {
			lastParagraph := strings.LastIndex(segment, "<p>")
			if lastParagraph < 0 {
				return doc, fmt.Errorf("%s: guarantee denial paragraph missing", boundaryRecord.ID)
			}
			segment = segment[:lastParagraph] + `<p data-boundary-guarantee-denial>` + segment[lastParagraph+3:]
			anchorNeedle := `data-journey-link-id="` + boundaryRecord.Continuation.JourneyLinkID + `" data-boundary-continuation`
			if strings.Count(segment, anchorNeedle) != 1 {
				return doc, fmt.Errorf("%s/%s: dual-identity continuation missing", boundaryRecord.ID, boundaryRecord.Continuation.JourneyLinkID)
			}
		}
		segment, _ = replaceOnce(segment, closingNeedle, `</aside>`, boundaryRecord.ID)
		doc = doc[:start] + segment + doc[end:]
	}
	return doc, nil
}

func bindAdoptionInstructions(doc string, instructions []adoptionInstruction, route string) (string, error) {
	for _, instruction := range instructions {
		if instruction.OwnerRoute != route {
			continue
		}
		before := "<pre><code>" + html.EscapeString(instruction.CommandText) + "</code></pre>"
		after := fmt.Sprintf(`<pre data-adoption-instruction-id="%s" data-command-sha256="%s"><code>%s</code></pre>`, html.EscapeString(instruction.ID), html.EscapeString(instruction.CommandSHA256), html.EscapeString(instruction.CommandText))
		var err error
		doc, err = replaceOnce(doc, before, after, instruction.ID)
		if err != nil {
			return doc, err
		}
	}
	return doc, nil
}

func bindGeneratedSources(doc, commit string) (string, error) {
	itemPattern := regexp.MustCompile(`(?s)<li (data-generated-source-descriptor[^>]*)>(https://github\.com/backstop-ai/backstop-core/[^<]+)</li>`)
	items := itemPattern.FindAllStringSubmatch(doc, -1)
	for _, item := range items {
		attrs := item[1]
		url := html.UnescapeString(item[2])
		url = strings.ReplaceAll(url, "<SITE-COMMIT>", commit)
		kind := attribute(attrs, "data-source-kind")
		path := attribute(attrs, "data-source-path")
		recordCommit := attribute(attrs, "data-source-commit")
		binding := attribute(attrs, "data-commit-binding")
		boundCommit := commit
		if binding == "record" {
			boundCommit = recordCommit
		}
		if kind == "" || binding == "" || !fullCommitPattern.MatchString(boundCommit) {
			return doc, fmt.Errorf("generated source descriptor: incomplete kind/binding/commit: %s", attrs)
		}
		link := fmt.Sprintf(`<li %s><a data-generated-source-link data-source-kind="%s" data-source-commit="%s"`, attrs, html.EscapeString(kind), html.EscapeString(boundCommit))
		if path != "" {
			link += ` data-source-path="` + html.EscapeString(path) + `"`
		}
		link += ` href="` + html.EscapeString(url) + `">Source</a></li>`
		doc = strings.Replace(doc, item[0], link, 1)
	}
	if strings.Contains(doc, "&lt;SITE-COMMIT&gt;") || strings.Contains(doc, "<SITE-COMMIT>") {
		return doc, errors.New("generated source descriptor: unresolved SITE-COMMIT")
	}
	return doc, nil
}

func attribute(attrs, name string) string {
	pattern := regexp.MustCompile(regexp.QuoteMeta(name) + `="([^"]*)"`)
	match := pattern.FindStringSubmatch(attrs)
	if len(match) == 2 {
		return html.UnescapeString(match[1])
	}
	return ""
}

func wrapTables(doc, route string) string {
	index := 0
	cursor := 0
	for {
		startRelative := strings.Index(doc[cursor:], "<table")
		if startRelative < 0 {
			break
		}
		start := cursor + startRelative
		endRelative := strings.Index(doc[start:], "</table>")
		if endRelative < 0 {
			break
		}
		end := start + endRelative + len("</table>")
		index++
		labelID := fmt.Sprintf("overflow-%s-%d", strings.Trim(strings.ReplaceAll(route, "/", "-"), "-"), index)
		if route == "/" {
			labelID = fmt.Sprintf("overflow-home-%d", index)
		}
		before := fmt.Sprintf(`<span class="visually-hidden" id="%s">Scrollable data table</span><div data-overflow-region role="region" aria-labelledby="%s" tabindex="0">`, labelID, labelID)
		doc = doc[:start] + before + doc[start:end] + "</div>" + doc[end:]
		cursor = end + len(before) + len("</div>")
	}
	return doc
}

func Render(root, builtRoot, siteCommit string) []Finding {
	if !fullCommitPattern.MatchString(siteCommit) {
		return []Finding{{Phase: "annotation", Identity: "site-commit", Message: "expected full lowercase 40-hex commit"}}
	}
	var topologyData topology
	var inventory evidenceInventory
	var model productModel
	for path, target := range map[string]any{
		filepath.Join(root, "docs/_data/content-topology.yml"):   &topologyData,
		filepath.Join(root, "docs/_data/evidence-inventory.yml"): &inventory,
		filepath.Join(root, "docs/_data/product-model.yml"):      &model,
	} {
		if err := loadYAML(path, target); err != nil {
			return []Finding{{Phase: "annotation", Identity: path, Message: err.Error()}}
		}
	}
	boundaries := make(map[string]boundary, len(model.Boundaries))
	for _, record := range model.Boundaries {
		boundaries[record.ID] = record
	}
	routes := map[string]bool{"/": true}
	for _, link := range topologyData.JourneyLinks {
		routes[link.SourceRoute] = true
	}
	for _, claim := range inventory.Claims {
		routes[claim.Owner.Route] = true
	}
	for _, instruction := range topologyData.AdoptionInstructions {
		routes[instruction.OwnerRoute] = true
	}
	orderedRoutes := make([]string, 0, len(routes))
	for route := range routes {
		orderedRoutes = append(orderedRoutes, route)
	}
	sort.Strings(orderedRoutes)
	var findings []Finding
	for _, route := range orderedRoutes {
		path := routeFile(builtRoot, route)
		data, err := os.ReadFile(path)
		if err != nil {
			findings = append(findings, Finding{Phase: "annotation", Identity: route, Message: err.Error()})
			continue
		}
		doc := string(data)
		if doc, err = bindJourneyLinks(doc, topologyData.JourneyLinks, route); err == nil {
			doc, err = bindClaims(doc, inventory.Claims, boundaries, route)
		}
		if err == nil {
			doc, err = bindAdoptionInstructions(doc, topologyData.AdoptionInstructions, route)
		}
		if err == nil {
			doc, err = bindGeneratedSources(doc, siteCommit)
		}
		if err != nil {
			findings = append(findings, Finding{Phase: "annotation", Identity: route, Message: err.Error()})
			continue
		}
		doc = wrapTables(doc, route)
		if err := os.WriteFile(path, []byte(doc), 0o644); err != nil {
			findings = append(findings, Finding{Phase: "annotation", Identity: route, Message: err.Error()})
		}
	}
	return findings
}

func main() {
	root := flag.String("root", ".", "repository root")
	builtRoot := flag.String("built-root", "_site", "built site root")
	commit := flag.String("site-commit", "", "full site commit")
	flag.Parse()
	findings := Render(*root, *builtRoot, *commit)
	for _, finding := range findings {
		_, _ = fmt.Fprintf(os.Stderr, "%s: %s: %s\n", finding.Phase, finding.Identity, finding.Message)
	}
	if len(findings) > 0 {
		os.Exit(1)
	}
	_, _ = fmt.Fprintf(os.Stdout, "annotation: rendered owner contracts for %s\n", *commit)
}
