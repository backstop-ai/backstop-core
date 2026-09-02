package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"html"
	"net"
	"net/url"
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
	Expected string
	Observed string
}

func (finding Finding) Error() string {
	return fmt.Sprintf("%s: %s: expected %s, observed %s", finding.Phase, finding.Identity, finding.Expected, finding.Observed)
}

type PresentationPage struct {
	Route        string   `yaml:"route"`
	PageKind     string   `yaml:"page_kind"`
	HeroQuestion string   `yaml:"hero_question"`
	Treatments   []string `yaml:"treatments"`
	NextAction   string   `yaml:"next_action"`
}

type Presentation struct {
	SchemaVersion string             `yaml:"schema_version"`
	Pages         []PresentationPage `yaml:"pages"`
}

type OwnerAcceptanceExport struct {
	SchemaVersion string `yaml:"schema_version"`
	Subject       struct {
		ManifestIdentity string `yaml:"manifest_identity"`
		Version          string `yaml:"version"`
		RulesetVersion   string `yaml:"ruleset_version"`
	} `yaml:"subject"`
	ExportFingerprintBinding string `yaml:"export_fingerprint_binding"`
	Cells                    []struct {
		ID      string `yaml:"id"`
		RuleID  string `yaml:"rule_id"`
		Filters struct {
			Include []string `yaml:"include"`
			Exclude []string `yaml:"exclude"`
		} `yaml:"path_filters"`
		CleanFixture    string `yaml:"clean_fixture"`
		NegativeFixture string `yaml:"negative_fixture"`
		Mutation        struct {
			TargetRelativePath string `yaml:"target_relative_path"`
			UniqueBeforeBase64 string `yaml:"unique_before_base64"`
			ReplacementBase64  string `yaml:"replacement_base64"`
		} `yaml:"mutation"`
		PathFidelity struct {
			FixtureRelativePath string `yaml:"fixture_relative_path"`
			TargetRelativePath  string `yaml:"target_relative_path"`
			DispatchEvidenceRef string `yaml:"dispatch_evidence_ref"`
		} `yaml:"path_fidelity"`
	} `yaml:"cells"`
	TokenAsset struct {
		InstalledRelativePath string `yaml:"installed_relative_path"`
		MediaType             string `yaml:"media_type"`
		SHA256                string `yaml:"sha256"`
		PublicOutput          string `yaml:"public_output"`
	} `yaml:"token_asset"`
	ProtectedFileFingerprints []struct {
		Path   string `yaml:"path"`
		SHA256 string `yaml:"sha256"`
	} `yaml:"protected_file_fingerprints"`
}

func journeyLinkIDs() []string {
	return []string{"JLINK-001", "JLINK-002", "JLINK-003", "JLINK-004", "JLINK-007", "JLINK-009", "JLINK-013", "JLINK-015", "JLINK-016", "JLINK-020", "JLINK-023", "JLINK-024"}
}

func canonicalRoutes() []string {
	return []string{"/", "/evaluate/", "/model/", "/adopt/", "/use-cases/", "/pack/examples/", "/pack/guide/", "/reference/", "/status/", "/contributing/"}
}

func primaryNavigation() []string {
	return []string{"/evaluate/", "/model/", "/adopt/", "/pack/"}
}

func extraLinkableRoutes() []string {
	return []string{"/pack/"}
}

func utilityNavigation() []string {
	return []string{"/contributing/"}
}

func legacyRedirects() map[string]string {
	return map[string]string{
		"getting-started.html":   "/adopt/",
		"concepts.html":          "/model/",
		"artifact-workflow.html": "/model/",
		"cli-reference.html":     "/reference/",
	}
}

func navigationLabel(route string) string {
	switch route {
	case "/evaluate/":
		return "Evaluate"
	case "/model/":
		return "Model"
	case "/adopt/":
		return "Adopt"
	case "/use-cases/":
		return "Use Cases"
	case "/pack/":
		return "Pack"
	case "/reference/":
		return "Reference"
	case "/status/":
		return "Status"
	case "/contributing/":
		return "Contributing"
	default:
		return ""
	}
}

func builtRoutePath(root, route string) string {
	if route == "/" {
		return filepath.Join(root, "index.html")
	}
	return filepath.Join(root, strings.Trim(route, "/"), "index.html")
}

func loadSitePresentation(path string) (Presentation, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Presentation{}, err
	}
	var presentation Presentation
	decoder := yaml.NewDecoder(strings.NewReader(string(data)))
	decoder.KnownFields(true)
	if err := decoder.Decode(&presentation); err != nil {
		return Presentation{}, err
	}
	if presentation.SchemaVersion != "backstop-core/site-presentation/v1" {
		return Presentation{}, fmt.Errorf("schema_version=%q", presentation.SchemaVersion)
	}
	return presentation, nil
}

func routeIDs(doc string) map[string]int {
	ids := map[string]int{}
	pattern := regexp.MustCompile(`\sid="([^"]+)"`)
	for _, match := range pattern.FindAllStringSubmatch(doc, -1) {
		ids[html.UnescapeString(match[1])]++
	}
	return ids
}

func attributeCount(doc, name, value string) int {
	if value == "" {
		return strings.Count(doc, name)
	}
	return strings.Count(doc, name+`="`+value+`"`)
}

func addCardinality(findings *[]Finding, phase, identity, expected string, observed int) {
	if expected == fmt.Sprintf("%d", observed) {
		return
	}
	*findings = append(*findings, Finding{Phase: phase, Identity: identity, Expected: expected, Observed: fmt.Sprintf("%d", observed)})
}

func verifyRouteDocument(route string, page PresentationPage, doc string) []Finding {
	var findings []Finding
	addCardinality(&findings, "rendered-route", route+" shell", "1", attributeCount(doc, "data-site-shell", "field-guide-v1"))
	addCardinality(&findings, "rendered-route", route+" page-kind", "1", attributeCount(doc, "data-page-kind", page.PageKind))
	addCardinality(&findings, "rendered-route", route+" main", "1", strings.Count(doc, `<main id="main" data-page-route="`+route+`"`))
	addCardinality(&findings, "rendered-route", route+" hero", "1", attributeCount(doc, "data-page-hero", ""))
	if route == "/" {
		findings = append(findings, verifyHomepageDirection(doc)...)
	} else {
		addCardinality(&findings, "rendered-route", route+" question", "1", strings.Count(doc, `data-page-question>`+html.EscapeString(page.HeroQuestion)+`</h1>`))
		if strings.Contains(doc, "data-home-") {
			findings = append(findings, Finding{Phase: "rendered-route", Identity: route + " homepage fence", Expected: "non-home shared shell without homepage composition markers", Observed: "data-home-* marker present"})
		}
	}
	addCardinality(&findings, "rendered-route", route+" next-action", "1", strings.Count(doc, `href="`+page.NextAction+`">Next`))
	addCardinality(&findings, "rendered-route", route+" primary-navigation", "1", strings.Count(doc, `<nav aria-label="Primary">`))
	addCardinality(&findings, "rendered-route", route+" utility-navigation", "1", strings.Count(doc, `<nav aria-label="Utility">`))
	wordmark := `<span>./b</span><span>backstop</span><span>.sh</span>`
	addCardinality(&findings, "rendered-route", route+" wordmark", "1", strings.Count(doc, wordmark))
	addCardinality(&findings, "rendered-route", route+" wordmark owner marker", "1", strings.Count(doc, `data-backstop-wordmark`))
	addCardinality(&findings, "rendered-route", route+" canonical", "1", strings.Count(doc, `<link rel="canonical" href="https://backstop.sh`+route+`">`))
	if strings.Contains(strings.ToLower(doc), "<script") || regexp.MustCompile(`(?i)(src|href)="[^"]+\.js(?:[?#][^"]*)?"`).MatchString(doc) {
		findings = append(findings, Finding{Phase: "runtime-absence", Identity: route, Expected: "no published JavaScript", Observed: "script or JavaScript asset reference"})
	}
	for id, count := range routeIDs(doc) {
		if count != 1 {
			findings = append(findings, Finding{Phase: "anchor-cardinality", Identity: route + "#" + id, Expected: "1", Observed: fmt.Sprintf("%d", count)})
		}
	}
	currentExpected := 0
	for _, destination := range append(primaryNavigation(), utilityNavigation()...) {
		count := strings.Count(doc, `href="`+destination+`"`)
		if count < 1 {
			findings = append(findings, Finding{Phase: "navigation", Identity: route + " -> " + destination, Expected: "present", Observed: "missing"})
		}
		if route == destination {
			currentExpected = 1
		}
	}
	verifyNavigationOrder := func(label string, destinations []string) {
		start := strings.Index(doc, `<nav aria-label="`+label+`">`)
		if start < 0 {
			return
		}
		end := strings.Index(doc[start:], `</nav>`)
		if end < 0 {
			return
		}
		nav := doc[start : start+end]
		position := -1
		for _, destination := range destinations {
			destinationLabel := navigationLabel(destination)
			pattern := regexp.MustCompile(`<a\s+[^>]*href="` + regexp.QuoteMeta(destination) + `"[^>]*>` + regexp.QuoteMeta(destinationLabel) + `</a>`)
			location := pattern.FindStringIndex(nav)
			if location == nil || location[0] <= position {
				findings = append(findings, Finding{Phase: "navigation", Identity: route + " " + label + " order", Expected: destinationLabel + " -> " + destination, Observed: "missing, mislabeled, or reordered"})
				return
			}
			position = location[0]
		}
	}
	verifyNavigationOrder("Primary", primaryNavigation())
	verifyNavigationOrder("Utility", utilityNavigation())
	addCardinality(&findings, "navigation", route+" current-page", fmt.Sprintf("%d", currentExpected), strings.Count(doc, `aria-current="page"`))
	return findings
}

func verifyHomepageDirection(doc string) []Finding {
	var findings []Finding
	add := func(identity, expected, observed string) {
		findings = append(findings, Finding{Phase: "homepage-canonical", Identity: identity, Expected: expected, Observed: observed})
	}
	wordmark := `<span>./b</span><span>backstop</span><span>.sh</span>`
	if strings.Count(doc, wordmark) != 1 {
		add("wordmark", "one ordered source-visible ./b backstop .sh owner", fmt.Sprintf("%d", strings.Count(doc, wordmark)))
	}
	for _, expected := range []string{"Define the work.", "Enforce your standards.", "Detect drift."} {
		if !strings.Contains(doc, expected) {
			add("hero", "canonical Define/Enforce/Detect hero", "missing "+expected)
		}
	}
	if attributeCount(doc, "data-home-gate-proof", "") != 1 || !strings.Contains(doc, "backstop gate") {
		add("gate proof", "one substantive backstop gate proof", "missing or duplicated")
	}
	if strings.Contains(doc, "data-page-question") || strings.Contains(doc, ">Why Backstop<") || strings.Contains(doc, ">Choose your path<") {
		add("forbidden scaffold", "no field-guide question or scaffold headings", "legacy scaffold present")
	}
	sections := []struct{ id, number, title string }{{"define-work", "01", "Define the work"}, {"enforce-standards", "02", "Enforce your standards"}, {"detect-drift", "03", "Detect drift"}}
	position := -1
	if attributeCount(doc, "data-home-system-section", "") != len(sections) {
		add("system sections", "exactly three ordered canonical sections", fmt.Sprintf("%d", attributeCount(doc, "data-home-system-section", "")))
	} else {
		for _, section := range sections {
			index := strings.Index(doc, `id="`+section.id+`" data-home-system-section`)
			pattern := regexp.MustCompile(`(?s)id="` + regexp.QuoteMeta(section.id) + `" data-home-system-section[^>]*>.*?<span[^>]*>` + regexp.QuoteMeta(section.number) + `</span>.*?>` + regexp.QuoteMeta(section.title) + `<`)
			if index <= position || !pattern.MatchString(doc[index:]) {
				add("system sections", "ordered 01/02/03 Define/Enforce/Detect sections", section.id+" missing or reordered")
				break
			}
			position = index
		}
	}
	modeStart := strings.Index(doc, "data-home-modes")
	modeEnd := -1
	if modeStart >= 0 {
		modeEnd = strings.Index(doc[modeStart:], `</section>`)
	}
	expectedModes := []string{"Full framework", "Artifact workflow", "Standards enforcement", "Deterministic scaffolding"}
	if modeStart < 0 || modeEnd < 0 {
		add("composability modes", "exactly four canonical modes", "mode region missing")
	} else {
		region := doc[modeStart : modeStart+modeEnd]
		matches := regexp.MustCompile(`<h3>([^<]+)</h3>`).FindAllStringSubmatch(region, -1)
		observed := make([]string, 0, len(matches))
		for _, match := range matches {
			observed = append(observed, html.UnescapeString(match[1]))
		}
		if strings.Join(observed, "|") != strings.Join(expectedModes, "|") {
			add("composability modes", strings.Join(expectedModes, " | "), strings.Join(observed, " | "))
		}
		for _, distinction := range []string{"Artifacts + packs + recipes + gates", "Use the whole chain or only what you need", "Packs + gate", "Recipe packs"} {
			if !strings.Contains(region, distinction) {
				add("composability modes", "four distinguishing mode contracts", "missing "+distinction)
			}
		}
	}
	if strings.Count(doc, "CLAIM-017") != 1 || strings.Count(doc, "JLINK-001") != 1 {
		add("owner markers", "one CLAIM-017 and one JLINK-001", fmt.Sprintf("claim=%d journey=%d", strings.Count(doc, "CLAIM-017"), strings.Count(doc, "JLINK-001")))
	}
	if !strings.Contains(doc, `href="/evaluate/#failure-fit"`) && !strings.Contains(doc, `href="/evaluate/#target"`) {
		add("journey destination", "canonical root-relative evaluate anchor", "missing")
	}
	if strings.Count(doc, `data-next-action`) != 1 || !strings.Contains(doc, `href="/evaluate/">Next`) {
		add("next action", "one /evaluate/ next action", "missing or drifted")
	}
	if strings.Count(doc, "<footer") != 1 || !strings.Contains(doc, "Open source under the MIT License.") {
		add("footer", "one established footer", "missing or drifted")
	}
	if !strings.Contains(doc, `data-required-blocks="define-work,enforce-standards,detect-drift,composable-modes"`) {
		add("required blocks", "canonical required-block IDs on substantive containers", "missing or drifted")
	}
	return findings
}

func collectBuiltDocuments(builtRoot string) (map[string]string, map[string]map[string]int, []Finding) {
	documents := map[string]string{}
	ids := map[string]map[string]int{}
	var findings []Finding
	for _, route := range canonicalRoutes() {
		path := builtRoutePath(builtRoot, route)
		data, err := os.ReadFile(path)
		if err != nil {
			findings = append(findings, Finding{Phase: "canonical-route", Identity: route, Expected: "one built file", Observed: err.Error()})
			continue
		}
		documents[route] = string(data)
		ids[route] = routeIDs(string(data))
	}
	for _, route := range extraLinkableRoutes() {
		path := builtRoutePath(builtRoot, route)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		documents[route] = string(data)
		ids[route] = routeIDs(string(data))
	}
	return documents, ids, findings
}

func verifyLinks(documents map[string]string, ids map[string]map[string]int) []Finding {
	var findings []Finding
	anchorPattern := regexp.MustCompile(`<a\s+[^>]*href="([^"]*)"[^>]*>`)
	for route, doc := range documents {
		for _, match := range anchorPattern.FindAllStringSubmatch(doc, -1) {
			href := html.UnescapeString(match[1])
			if href == "" {
				findings = append(findings, Finding{Phase: "link-resolution", Identity: route, Expected: "nonempty href", Observed: "empty"})
				continue
			}
			parsed, err := url.Parse(href)
			if err != nil {
				findings = append(findings, Finding{Phase: "link-resolution", Identity: route + " -> " + href, Expected: "parseable link", Observed: err.Error()})
				continue
			}
			if parsed.Scheme == "mailto" {
				if parsed.Opaque == "" || strings.Contains(parsed.Opaque, " ") {
					findings = append(findings, Finding{Phase: "link-resolution", Identity: route + " -> " + href, Expected: "valid mailto recipient", Observed: "empty or invalid"})
				}
				continue
			}
			if parsed.IsAbs() || parsed.Host != "" || strings.HasPrefix(href, "//") {
				hostname := strings.ToLower(parsed.Hostname())
				ip := net.ParseIP(hostname)
				allowed := parsed.Scheme == "https" && parsed.Host != "" && hostname != "backstop.sh" && hostname != "localhost" && (ip == nil || !ip.IsLoopback())
				if !allowed {
					findings = append(findings, Finding{Phase: "link-resolution", Identity: route + " -> " + href, Expected: "cross-origin HTTPS or valid mailto", Observed: "forbidden absolute link"})
				}
				continue
			}
			if parsed.Path != "" && !strings.HasPrefix(parsed.Path, "/") {
				findings = append(findings, Finding{Phase: "link-resolution", Identity: route + " -> " + href, Expected: "root-relative internal path", Observed: "path-relative"})
				continue
			}
			targetRoute := route
			if parsed.Path != "" {
				targetRoute = parsed.Path
			}
			fragment := parsed.Fragment
			if _, ok := documents[targetRoute]; !ok {
				findings = append(findings, Finding{Phase: "link-resolution", Identity: route + " -> " + href, Expected: "canonical route", Observed: "missing or alias target"})
				continue
			}
			if fragment != "" && ids[targetRoute][fragment] != 1 {
				findings = append(findings, Finding{Phase: "link-resolution", Identity: route + " -> " + href, Expected: "one case-sensitive target ID", Observed: fmt.Sprintf("%d", ids[targetRoute][fragment])})
			}
		}
	}
	return findings
}

func VerifyRenderedOwnerContracts(_ string, builtRoot string, siteCommit string) []Finding {
	documents, _, findings := collectBuiltDocuments(builtRoot)
	joined := ""
	for _, route := range canonicalRoutes() {
		joined += documents[route]
	}
	for _, id := range journeyLinkIDs() {
		addCardinality(&findings, "owner-contracts", id, "1", strings.Count(joined, `data-journey-link-id="`+id+`"`))
		if !regexp.MustCompile(`<a\s+[^>]*data-journey-link-id="` + regexp.QuoteMeta(id) + `"[^>]*href="/[^"]+#[^"]+"[^>]*>[^<]+</a>`).MatchString(joined) {
			findings = append(findings, Finding{Phase: "owner-contracts", Identity: id, Expected: "one labeled root-relative route/anchor link", Observed: "missing or malformed anchor"})
		}
	}
	for index := 1; index <= 5; index++ {
		id := fmt.Sprintf("BOUNDARY-%03d", index)
		addCardinality(&findings, "owner-contracts", id, "1", strings.Count(joined, `data-boundary-id="`+id+`"`))
		pattern := regexp.MustCompile(`(?s)<aside\s+[^>]*data-boundary-id="` + regexp.QuoteMeta(id) + `"[^>]*data-boundary-state="[^"]+"[^>]*>(.*?)</aside>`)
		match := pattern.FindStringSubmatch(joined)
		if len(match) != 2 || !regexp.MustCompile(`(?s)data-boundary-explanation[^>]*>\s*[^<]`).MatchString(match[1]) {
			findings = append(findings, Finding{Phase: "owner-contracts", Identity: id, Expected: "one structured callout with state and nonempty explanation", Observed: "missing or malformed"})
		}
	}
	for _, id := range []string{"ADOPT-INSTALL", "ADOPT-CONFIGURE", "ADOPT-ENFORCE"} {
		addCardinality(&findings, "owner-contracts", id, "1", strings.Count(joined, `data-adoption-instruction-id="`+id+`"`))
		pattern := regexp.MustCompile(`(?s)<pre\s+data-adoption-instruction-id="` + regexp.QuoteMeta(id) + `"\s+data-command-sha256="sha256:[0-9a-f]{64}"><code>[^<]+</code></pre>`)
		if !pattern.MatchString(joined) {
			findings = append(findings, Finding{Phase: "owner-contracts", Identity: id, Expected: "digest-bound nonempty code bytes", Observed: "missing or malformed"})
		}
	}
	for _, job := range []string{"cli-command-catalog", "artifact-schema-catalog", "published-pack-catalog", "release-history"} {
		addCardinality(&findings, "owner-contracts", job+" region", "1", strings.Count(joined, `data-generated-region="" data-product-truth-job="`+job+`"`))
		pattern := regexp.MustCompile(`(?s)<section\s+data-generated-region=""\s+data-product-truth-job="` + regexp.QuoteMeta(job) + `"[^>]*>(.*?)</section>`)
		match := pattern.FindStringSubmatch(joined)
		if len(match) != 2 || !strings.Contains(match[1], "data-generated-source-link") {
			findings = append(findings, Finding{Phase: "owner-contracts", Identity: job, Expected: "one generated region containing immutable source links", Observed: "missing or malformed"})
		}
	}
	dualIdentity := regexp.MustCompile(`(?s)<aside\s+[^>]*data-boundary-id="BOUNDARY-005"[^>]*>.*?data-boundary-explanation[^>]*>[^<].*?<a\s+data-journey-link-id="JLINK-024"\s+data-boundary-continuation[^>]*>[^<]+</a>.*?data-boundary-guarantee-denial[^>]*>[^<].*?</aside>`)
	if strings.Count(joined, `data-journey-link-id="JLINK-024" data-boundary-continuation`) != 1 || !dualIdentity.MatchString(joined) {
		findings = append(findings, Finding{Phase: "owner-contracts", Identity: "JLINK-024/BOUNDARY-005", Expected: "one dual-identity anchor", Observed: "missing, split, or duplicated"})
	}
	generatedLink := regexp.MustCompile(`<a\s+data-generated-source-link\s+data-source-kind="[^"]+"\s+data-source-commit="([0-9a-f]{40})"[^>]*href="(https://github\.com/backstop-ai/backstop-core/[^"]+)"`)
	for _, match := range generatedLink.FindAllStringSubmatch(joined, -1) {
		if !strings.Contains(match[2], match[1]) {
			findings = append(findings, Finding{Phase: "owner-contracts", Identity: "generated provenance", Expected: "href bound to data-source-commit", Observed: match[0]})
		}
	}
	if !regexp.MustCompile(`^[0-9a-f]{40}$`).MatchString(siteCommit) || strings.Contains(joined, "SITE-COMMIT") {
		findings = append(findings, Finding{Phase: "owner-contracts", Identity: "generated provenance", Expected: siteCommit, Observed: "unresolved or invalid commit binding"})
	}
	return findings
}

func LoadOwnerAcceptanceExport(root string) (OwnerAcceptanceExport, error) {
	path := filepath.Join(root, ".backstop/packs/backstop-ai/backstop-design-system/contracts/public-site-acceptance.yml")
	data, err := os.ReadFile(path)
	if err != nil {
		return OwnerAcceptanceExport{}, err
	}
	var export OwnerAcceptanceExport
	if err := yaml.Unmarshal(data, &export); err != nil {
		return OwnerAcceptanceExport{}, err
	}
	expectedCells := []string{"token", "inline-style", "focus", "reduced-motion", "accessibility", "wordmark", "reusable-presentation"}
	if export.SchemaVersion != "backstop-design-system/public-site-acceptance/v1" || export.Subject.ManifestIdentity != "backstop-ai/backstop-design-system" || export.Subject.Version != "0.1.5" || export.Subject.RulesetVersion != "1.3.1" || export.ExportFingerprintBinding != "release-evidence/v0.1.5.yml#public_site_acceptance" || len(export.Cells) != len(expectedCells) {
		return OwnerAcceptanceExport{}, errors.New("owner export identity, release, or seven-cell cardinality mismatch")
	}
	seenRules := map[string]bool{}
	for index, id := range expectedCells {
		cell := export.Cells[index]
		before, beforeErr := base64.StdEncoding.DecodeString(cell.Mutation.UniqueBeforeBase64)
		replacement, replacementErr := base64.StdEncoding.DecodeString(cell.Mutation.ReplacementBase64)
		included := false
		for _, pattern := range cell.Filters.Include {
			if pattern == cell.Mutation.TargetRelativePath {
				included = true
			}
		}
		if cell.ID != id || cell.RuleID == "" || seenRules[cell.RuleID] || !included || cell.CleanFixture == "" || cell.NegativeFixture == "" || cell.Mutation.TargetRelativePath == "" || len(before) == 0 || len(replacement) == 0 || beforeErr != nil || replacementErr != nil || string(before) == string(replacement) || cell.PathFidelity.TargetRelativePath != cell.Mutation.TargetRelativePath || cell.PathFidelity.FixtureRelativePath != cell.NegativeFixture || !strings.HasPrefix(cell.PathFidelity.DispatchEvidenceRef, "release-evidence/v0.1.5.yml#") {
			return OwnerAcceptanceExport{}, fmt.Errorf("owner export cell %q is incomplete or reordered", id)
		}
		if id == "wordmark" && string(before) != `<span>./b</span><span>backstop</span><span>.sh</span>` {
			return OwnerAcceptanceExport{}, errors.New("owner wordmark mutation source is not the exact three-part markup")
		}
		seenRules[cell.RuleID] = true
	}
	if export.TokenAsset.InstalledRelativePath == "" || export.TokenAsset.MediaType != "text/css" || !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(export.TokenAsset.SHA256) || export.TokenAsset.PublicOutput != "assets/css/design-system-tokens.css" {
		return OwnerAcceptanceExport{}, errors.New("owner token asset contract is incomplete")
	}
	packRoot := filepath.Join(root, ".backstop/packs/backstop-ai/backstop-design-system")
	if len(export.ProtectedFileFingerprints) == 0 {
		return OwnerAcceptanceExport{}, errors.New("owner protected fingerprint set is empty")
	}
	for _, fingerprint := range export.ProtectedFileFingerprints {
		if fingerprint.Path == "" || !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(fingerprint.SHA256) {
			return OwnerAcceptanceExport{}, errors.New("owner protected fingerprint is incomplete")
		}
		data, err := os.ReadFile(filepath.Join(packRoot, filepath.FromSlash(fingerprint.Path)))
		if err != nil {
			return OwnerAcceptanceExport{}, fmt.Errorf("owner protected file %s: %w", fingerprint.Path, err)
		}
		digest := sha256.Sum256(data)
		if hex.EncodeToString(digest[:]) != fingerprint.SHA256 {
			return OwnerAcceptanceExport{}, fmt.Errorf("owner protected file %s digest mismatch", fingerprint.Path)
		}
	}
	return export, nil
}

func BuildGateCorpus(_ string, builtRoot string) ([]string, error) {
	var paths []string
	err := filepath.WalkDir(builtRoot, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		extension := strings.ToLower(filepath.Ext(path))
		if extension != ".html" && extension != ".css" && extension != ".svg" {
			return nil
		}
		relative, err := filepath.Rel(builtRoot, path)
		if err != nil || strings.HasPrefix(relative, "..") {
			return fmt.Errorf("built path %q is outside built root", path)
		}
		paths = append(paths, filepath.ToSlash(filepath.Join("_site", relative)))
		return nil
	})
	sort.Slice(paths, func(left, right int) bool { return paths[left] < paths[right] })
	return paths, err
}

func VerifyInstalledDesignSystem(root, builtRoot string) []Finding {
	export, err := LoadOwnerAcceptanceExport(root)
	if err != nil {
		return []Finding{{Phase: "design-system", Identity: "owner export", Expected: "valid same-release seven-cell export", Observed: err.Error()}}
	}
	tokenPath := filepath.Join(builtRoot, filepath.FromSlash(export.TokenAsset.PublicOutput))
	data, err := os.ReadFile(tokenPath)
	if err != nil {
		return []Finding{{Phase: "design-system", Identity: export.TokenAsset.PublicOutput, Expected: export.TokenAsset.SHA256, Observed: err.Error()}}
	}
	digest := sha256.Sum256(data)
	observed := hex.EncodeToString(digest[:])
	if observed != export.TokenAsset.SHA256 {
		return []Finding{{Phase: "design-system", Identity: export.TokenAsset.PublicOutput, Expected: export.TokenAsset.SHA256, Observed: observed}}
	}
	indexData, err := os.ReadFile(filepath.Join(builtRoot, "index.html"))
	if err != nil {
		return []Finding{{Phase: "design-system", Identity: "index.html", Expected: "built shell", Observed: err.Error()}}
	}
	index := string(indexData)
	tokenLink := strings.Index(index, `href="/assets/css/design-system-tokens.css"`)
	siteLink := strings.Index(index, `href="/assets/css/site.css"`)
	if tokenLink < 0 || siteLink < 0 || tokenLink >= siteLink {
		return []Finding{{Phase: "design-system", Identity: "token linkage", Expected: "owner tokens linked before site.css", Observed: "missing or wrong order"}}
	}
	return nil
}

func Verify(root, builtRoot string) []Finding {
	presentation, err := loadSitePresentation(filepath.Join(root, "docs/_data/site-presentation.yml"))
	if err != nil {
		return []Finding{{Phase: "presentation", Identity: "site-presentation", Expected: "valid exact matrix", Observed: err.Error()}}
	}
	routes := canonicalRoutes()
	if len(presentation.Pages) != len(routes) {
		return []Finding{{Phase: "presentation", Identity: "route cardinality", Expected: fmt.Sprintf("%d", len(routes)), Observed: fmt.Sprintf("%d", len(presentation.Pages))}}
	}
	documents, ids, findings := collectBuiltDocuments(builtRoot)
	for index, route := range routes {
		page := presentation.Pages[index]
		if page.Route != route {
			findings = append(findings, Finding{Phase: "presentation", Identity: fmt.Sprintf("row %d", index), Expected: route, Observed: page.Route})
			continue
		}
		if doc, ok := documents[route]; ok {
			findings = append(findings, verifyRouteDocument(route, page, doc)...)
		}
	}
	findings = append(findings, verifyLinks(documents, ids)...)
	for alias, destination := range legacyRedirects() {
		data, err := os.ReadFile(filepath.Join(builtRoot, alias))
		if err != nil {
			findings = append(findings, Finding{Phase: "legacy-redirect", Identity: alias, Expected: destination, Observed: err.Error()})
			continue
		}
		doc := string(data)
		for _, expected := range []string{`href="https://backstop.sh` + destination + `"`, `content="0; url=` + destination + `"`, `href="` + destination + `"`} {
			if !strings.Contains(doc, expected) {
				findings = append(findings, Finding{Phase: "legacy-redirect", Identity: alias, Expected: expected, Observed: "missing"})
			}
		}
		if strings.Contains(strings.ToLower(doc), "<script") {
			findings = append(findings, Finding{Phase: "legacy-redirect", Identity: alias, Expected: "serverless redirect", Observed: "script"})
		}
	}
	expectedCNAME := []byte("backstop.sh\n")
	for _, path := range []string{filepath.Join(root, "docs/CNAME"), filepath.Join(builtRoot, "CNAME")} {
		data, err := os.ReadFile(path)
		if err != nil || string(data) != string(expectedCNAME) {
			findings = append(findings, Finding{Phase: "custom-domain", Identity: path, Expected: fmt.Sprintf("%q", expectedCNAME), Observed: fmt.Sprintf("%q err=%v", data, err)})
		}
	}
	findings = append(findings, VerifyInstalledDesignSystem(root, builtRoot)...)
	return findings
}
