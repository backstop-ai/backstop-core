package main

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func syntheticDocument(page presentationPage) string {
	var primary, utility strings.Builder
	labels := map[string]string{"/evaluate/": "Evaluate", "/model/": "Model", "/adopt/": "Adopt", "/pack/": "Pack", "/contributing/": "Contributing"}
	for _, destination := range primaryNavigation() {
		current := ""
		if page.Route == destination || (destination == "/pack/" && packFleetCurrent(page.Route)) {
			current = ` aria-current="page"`
		}
		fmt.Fprintf(&primary, `<a href="%s"%s>%s</a>`, destination, current, labels[destination])
	}
	for _, destination := range utilityNavigation() {
		current := ""
		if page.Route == destination {
			current = ` aria-current="page"`
		}
		fmt.Fprintf(&utility, `<a href="%s"%s>%s</a>`, destination, current, labels[destination])
	}
	wordmark := `<a data-backstop-wordmark href="/"><span>./b</span><span>backstop</span><span>.sh</span></a>`
	if page.Route == "/" {
		return fmt.Sprintf(`<!doctype html><html><head><link rel="canonical" href="https://backstop.sh/"><link rel="stylesheet" href="/assets/css/design-system-tokens.css"><link rel="stylesheet" href="/assets/css/site.css"></head><body data-site-shell="field-guide-v1"><a class="skip-link" href="#main">Skip to content</a><header>%s<nav aria-label="Primary">%s</nav><nav aria-label="Utility">%s</nav></header><main id="main" data-page-route="/" data-page-kind="home"><section data-page-hero><h1>Define the work. <span>Enforce your standards.</span> Detect drift.</h1><div data-home-gate-proof>backstop gate PASS</div></section><div data-page-content data-required-blocks="define-work,enforce-standards,detect-drift,composable-modes"><section id="define-work" data-home-system-section><span>01</span><h2>Define the work</h2><p>Find problems before implementation. <!-- backstop-claim: CLAIM-017 --> Requirements trace through a validated plan to implementation evidence before source changes begin. <!-- /backstop-claim --> <!-- backstop-journey-link: JLINK-001 --><a href="/evaluate/#target">Evaluate the failure fit</a></p></section><section id="enforce-standards" data-home-system-section><span>02</span><h2>Enforce your standards</h2><p>Encode engineering decisions once in versioned packs. Agents and CI run the same deterministic gate, with positive and negative fixtures proving each rule.</p></section><section id="detect-drift" data-home-system-section><span>03</span><h2>Detect drift</h2><p>Measure standards and requirements drift independently. Existing debt remains visible, while touched code and broken completion claims fail loudly.</p></section><section id="composable-modes"><h2>Use the framework. Or use the parts.</h2><div data-home-modes><article><span>01</span><h3>Full framework</h3><code>Artifacts + packs + recipes + gates</code><p>Define, scaffold, enforce, and verify the complete promise.</p></article><article><span>02</span><h3>Artifact workflow</h3><code>Use the whole chain or only what you need</code><p>Keep requirements, plans, decisions, and completion claims traceable.</p></article><article><span>03</span><h3>Standards enforcement</h3><code>Packs + gate</code><p>Install versioned rules and run the same result in development and CI.</p></article><article><span>04</span><h3>Deterministic scaffolding</h3><code>Recipe packs</code><p>Apply a pinned recipe without requiring enforcement rules.</p></article></div></section><a href="#define-work">fragment</a><a href="?mode=read">query</a><a href="https://example.org/reference">external</a><a href="mailto:maintainer@example.org">mail</a></div><nav data-next-action><a href="/evaluate/">Next</a></nav><footer>Open source under the MIT License.</footer></main></body></html>`, wordmark, primary.String(), utility.String())
	}
	targetID := "target"
	return fmt.Sprintf(`<!doctype html><html><head><link rel="canonical" href="https://backstop.sh%s"><link rel="stylesheet" href="/assets/css/design-system-tokens.css"><link rel="stylesheet" href="/assets/css/site.css"></head><body data-site-shell="field-guide-v1"><header>%s<nav aria-label="Primary">%s</nav><nav aria-label="Utility">%s</nav></header><main id="main" data-page-route="%s"><section data-page-hero><h1 data-page-question>%s</h1></section><article class="prose" data-page-kind="%s"><h2 id="%s">Target</h2><a href="#%s">fragment</a><a href="?mode=read">query</a><a href="https://example.org/reference">external</a><a href="mailto:maintainer@example.org">mail</a></article></main><nav data-next-action><a href="%s">Next</a></nav><footer>Open source under the MIT License.</footer></body></html>`, page.Route, wordmark, primary.String(), utility.String(), page.Route, page.HeroQuestion, page.PageKind, targetID, targetID, page.NextAction)
}

func makeSyntheticSite(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	built := filepath.Join(root, "_site")
	writeTestFile(t, filepath.Join(root, "docs/_data/site-presentation.yml"), "schema_version: backstop-core/site-presentation/v1\npages:\n")
	var presentation strings.Builder
	presentation.WriteString("schema_version: backstop-core/site-presentation/v1\npages:\n")
	for _, page := range canonicalPresentation() {
		fmt.Fprintf(&presentation, "  - route: %s\n    page_kind: %s\n    hero_question: %q\n    treatments: [%s]\n    next_action: %s\n", page.Route, page.PageKind, page.HeroQuestion, strings.Join(page.Treatments, ", "), page.NextAction)
		writeTestFile(t, builtRoutePath(built, page.Route), syntheticDocument(page))
	}
	writeTestFile(t, filepath.Join(root, "docs/_data/site-presentation.yml"), presentation.String())
	writeTestFile(t, builtRoutePath(built, "/pack/"), `<!doctype html><html><body><main id="main" data-page-route="/pack/"><h2 id="target">Pack</h2></main></body></html>`)
	for alias, destination := range legacyRedirects() {
		writeTestFile(t, filepath.Join(built, alias), fmt.Sprintf(`<link rel="canonical" href="https://backstop.sh%s"><meta http-equiv="refresh" content="0; url=%s"><a href="%s">Continue</a>`, destination, destination, destination))
	}
	writeTestFile(t, filepath.Join(root, "docs/CNAME"), "backstop.sh\n")
	writeTestFile(t, filepath.Join(built, "CNAME"), "backstop.sh\n")
	ownerStylesheet := "/* owner stylesheet */\n"
	digest := sha256.Sum256([]byte(ownerStylesheet))
	writeTestFile(t, filepath.Join(built, "assets/css/design-system-tokens.css"), ownerStylesheet)
	writeTestFile(t, filepath.Join(built, "assets/css/site.css"), "body { color: var(--ds-text); }\n:focus-visible { outline: var(--ds-focus-ring); }\n@media (prefers-reduced-motion: reduce) {}\n")
	packManifest := "name: backstop-ai/backstop-design-system\nversion: 0.1.5\n"
	packDigest := sha256.Sum256([]byte(packManifest))
	writeTestFile(t, filepath.Join(root, ".backstop/packs/backstop-ai/backstop-design-system/pack.yml"), packManifest)
	var cells strings.Builder
	for _, id := range []string{"token", "inline-style", "focus", "reduced-motion", "accessibility", "wordmark", "reusable-presentation"} {
		rule := map[string]string{"token": "no-raw-colors", "inline-style": "no-inline-styles", "focus": "focus-visible-required", "reduced-motion": "reduced-motion-required", "accessibility": "accessible-site-shell", "wordmark": "canonical-wordmark", "reusable-presentation": "reusable-page-hero"}[id]
		target := "index.html"
		before, replacement := "PGJvZHkgZGF0YS1zaXRlLXNoZWxsPSJmaWVsZC1ndWlkZS12MSI+", "PGJvZHkgZGF0YS1zaXRlLXNoZWxsPSJmaWVsZC1ndWlkZS12MSIgc3R5bGU9ImNvbG9yOiBpbmhlcml0Ij4="
		switch id {
		case "token":
			target, before, replacement = "assets/css/site.css", "Y29sb3I6IHZhcigtLWRzLXRleHQpOw==", "Y29sb3I6ICMxMTE4Mjc7"
		case "focus":
			target, before, replacement = "assets/css/site.css", "OmZvY3VzLXZpc2libGUgeyBvdXRsaW5lOiB2YXIoLS1kcy1mb2N1cy1yaW5nKTsgfQ==", "OmZvY3VzIHsgb3V0bGluZTogdmFyKC0tZHMtZm9jdXMtcmluZyk7IH0="
		case "reduced-motion":
			target, before, replacement = "assets/css/site.css", "QG1lZGlhIChwcmVmZXJzLXJlZHVjZWQtbW90aW9uOiByZWR1Y2Up", "QG1lZGlhIChtaW4td2lkdGg6IDAp"
		case "accessibility":
			before, replacement = "PG5hdiBhcmlhLWxhYmVsPSJQcmltYXJ5Ij4=", "PG5hdj4="
		case "wordmark":
			before, replacement = "PHNwYW4+Li9iPC9zcGFuPjxzcGFuPmJhY2tzdG9wPC9zcGFuPjxzcGFuPi5zaDwvc3Bhbj4=", "PHNwYW4+Li94PC9zcGFuPjxzcGFuPmJhY2tzdG9wPC9zcGFuPjxzcGFuPi5zaDwvc3Bhbj4="
		case "reusable-presentation":
			before, replacement = "PHNlY3Rpb24gZGF0YS1wYWdlLWhlcm8+", "PHNlY3Rpb24gZGF0YS1wYWdlLWhlcm8+PGgyPkR1cGxpY2F0ZTwvaDI+PC9zZWN0aW9uPjxzZWN0aW9uIGRhdGEtcGFnZS1oZXJvPg=="
		}
		fmt.Fprintf(&cells, `  - id: %s
    rule_id: %s
    path_filters: {include: [%s], exclude: ["vendor/**"]}
    clean_fixture: fixtures/%s-clean.html
    negative_fixture: fixtures/%s-bad.html
    mutation: {target_relative_path: %s, unique_before_base64: %s, replacement_base64: %s}
    path_fidelity: {fixture_relative_path: fixtures/%s-bad.html, target_relative_path: %s, dispatch_evidence_ref: release-evidence/v0.1.5.yml#%s}
`, id, rule, target, id, id, target, before, replacement, id, target, id)
	}
	export := fmt.Sprintf(`schema_version: backstop-design-system/public-site-acceptance/v1
subject: {manifest_identity: backstop-ai/backstop-design-system, version: 0.1.5, ruleset_version: 1.3.1}
export_fingerprint_binding: release-evidence/v0.1.5.yml#public_site_acceptance
cells:
%stoken_asset: {installed_relative_path: assets/design-system-tokens.css, media_type: text/css, sha256: %x, public_output: assets/css/design-system-tokens.css}
protected_file_fingerprints:
  - {path: pack.yml, sha256: %x}
`, cells.String(), digest, packDigest)
	writeTestFile(t, filepath.Join(root, ".backstop/packs/backstop-ai/backstop-design-system/contracts/public-site-acceptance.yml"), export)
	return root, built
}

func requireCleanSite(t *testing.T, root, built string) {
	t.Helper()
	if findings := Verify(root, built); len(findings) != 0 {
		t.Fatalf("expected clean site, observed %#v", findings)
	}
}

func TestSiteCheck_CanonicalRouteMatrixPasses(t *testing.T) {
	root, built := makeSyntheticSite(t)
	requireCleanSite(t, root, built)
}

func TestSiteCheck_RouteCatalogsAreImmutableAndExhaustive(t *testing.T) {
	wantRoutes := []string{"/", "/evaluate/", "/model/", "/adopt/", "/use-cases/", "/pack/examples/", "/pack/guide/", "/reference/", "/status/", "/contributing/"}
	wantPrimary := []string{"/evaluate/", "/model/", "/adopt/", "/pack/"}
	wantUtility := []string{"/contributing/"}
	wantRedirects := map[string]string{"getting-started.html": "/adopt/", "concepts.html": "/model/", "artifact-workflow.html": "/model/", "cli-reference.html": "/reference/"}

	assertSlice := func(name string, got, want []string) {
		t.Helper()
		if strings.Join(got, "|") != strings.Join(want, "|") {
			t.Fatalf("%s = %#v, want %#v", name, got, want)
		}
		got[0] = "/mutated/"
		if strings.Join(got, "|") == strings.Join(want, "|") {
			t.Fatalf("%s mutation did not alter the test copy", name)
		}
	}
	assertSlice("canonical routes", canonicalRoutes(), wantRoutes)
	assertSlice("primary navigation", primaryNavigation(), wantPrimary)
	assertSlice("utility navigation", utilityNavigation(), wantUtility)
	for _, catalog := range []struct {
		name string
		got  []string
		want []string
	}{
		{"canonical routes", canonicalRoutes(), wantRoutes},
		{"primary navigation", primaryNavigation(), wantPrimary},
		{"utility navigation", utilityNavigation(), wantUtility},
	} {
		if strings.Join(catalog.got, "|") != strings.Join(catalog.want, "|") {
			t.Fatalf("%s retained caller mutation: %#v", catalog.name, catalog.got)
		}
	}
	redirects := legacyRedirects()
	if len(redirects) != len(wantRedirects) {
		t.Fatalf("redirect cardinality = %d, want %d", len(redirects), len(wantRedirects))
	}
	for alias, destination := range wantRedirects {
		if redirects[alias] != destination {
			t.Fatalf("redirect %q = %q, want %q", alias, redirects[alias], destination)
		}
	}
	redirects["concepts.html"] = "/mutated/"
	if got := legacyRedirects()["concepts.html"]; got != "/model/" {
		t.Fatalf("redirect catalog retained caller mutation: %q", got)
	}
}

func TestSiteCheck_PackFleetCurrentPredicate(t *testing.T) {
	for _, route := range []string{"/pack/", "/pack/guide/", "/pack/examples/"} {
		if !packFleetCurrent(route) {
			t.Fatalf("packFleetCurrent(%q) = false, want true", route)
		}
	}
	for _, route := range []string{"/pack/examplesx/", "/packs/", "/extend/", "/reference/", "/"} {
		if packFleetCurrent(route) {
			t.Fatalf("packFleetCurrent(%q) = true, want false", route)
		}
	}
}

func TestSiteCheck_EntityRoutesAreLinkableAndOutsideCanonicalTopology(t *testing.T) {
	wantExtra := []string{"/pack/", "/plan/", "/issue/", "/spec/", "/bundle/", "/directive/", "/adr/", "/capability/"}
	gotExtra := extraLinkableRoutes()
	if len(gotExtra) != len(wantExtra) {
		t.Fatalf("extraLinkableRoutes len = %d, want %d", len(gotExtra), len(wantExtra))
	}
	seen := map[string]int{}
	for _, route := range gotExtra {
		seen[route]++
	}
	for _, route := range wantExtra {
		if seen[route] != 1 {
			t.Fatalf("route %q count = %d, want 1", route, seen[route])
		}
	}
	gotExtra[0] = "/mutated/"
	if extraLinkableRoutes()[0] == "/mutated/" {
		t.Fatal("extraLinkableRoutes retained caller mutation")
	}

	baselineCanonical := append([]string(nil), canonicalRoutes()...)
	baselinePrimary := append([]string(nil), primaryNavigation()...)
	baselineUtility := append([]string(nil), utilityNavigation()...)
	baselineRedirects := legacyRedirects()
	for _, route := range wantExtra {
		if route != "/pack/" && containsString(canonicalRoutes(), route) {
			t.Fatalf("entity route %q appears in canonicalRoutes", route)
		}
	}
	for _, route := range []string{"/plan/", "/issue/", "/spec/", "/bundle/", "/directive/", "/adr/", "/capability/"} {
		if containsString(primaryNavigation(), route) || containsString(utilityNavigation(), route) {
			t.Fatalf("entity route %q appears in navigation", route)
		}
	}
	if strings.Join(canonicalRoutes(), "|") != strings.Join(baselineCanonical, "|") {
		t.Fatal("canonicalRoutes changed")
	}
	if strings.Join(primaryNavigation(), "|") != strings.Join(baselinePrimary, "|") {
		t.Fatal("primaryNavigation changed")
	}
	if strings.Join(utilityNavigation(), "|") != strings.Join(baselineUtility, "|") {
		t.Fatal("utilityNavigation changed")
	}
	if len(legacyRedirects()) != len(baselineRedirects) {
		t.Fatal("legacyRedirects cardinality changed")
	}

	root := t.TempDir()
	built := filepath.Join(root, "_site")
	for _, route := range extraLinkableRoutes() {
		if route == "/pack/" {
			continue
		}
		if _, err := os.Stat(builtRoutePath(built, route)); err == nil {
			t.Fatalf("unexpected prebuilt entity route %s", route)
		}
	}
	documents, _, findings := collectBuiltDocuments(built)
	if len(findings) == 0 {
		t.Fatal("missing canonical routes should produce canonical-route findings")
	}
	if _, ok := documents["/issue/"]; ok {
		t.Fatal("missing entity route must be skipped without a finding")
	}

	for _, route := range canonicalRoutes() {
		writeTestFile(t, builtRoutePath(built, route), syntheticDocument(canonicalPresentationPage(route)))
	}
	writeTestFile(t, builtRoutePath(built, "/pack/"), `<!doctype html><html><body><main id="main" data-page-route="/pack/"><h2 id="target">Pack</h2></main></body></html>`)
	writeTestFile(t, builtRoutePath(built, "/issue/"), `<!doctype html><html><body><main data-page-route="/issue/"><h2 id="target">Issue</h2></main></body></html>`)
	documents, ids, findings := collectBuiltDocuments(built)
	if len(findings) != 0 {
		t.Fatalf("full canonical tree findings = %#v", findings)
	}
	if _, ok := documents["/issue/"]; !ok {
		t.Fatal("entity route must load when present in built tree")
	}
	evaluate := strings.Replace(documents["/evaluate/"], `</main>`, `<a href="/issue/#target">Issue</a></main>`, 1)
	documents["/evaluate/"] = evaluate
	if linkFindings := verifyLinks(documents, ids); len(linkFindings) != 0 {
		t.Fatalf("entity link should resolve: %#v", linkFindings)
	}
	delete(documents, "/issue/")
	if linkFindings := verifyLinks(documents, ids); len(linkFindings) == 0 {
		t.Fatal("missing entity document should produce link-resolution finding")
	}
	bad := documents["/evaluate/"]
	bad = strings.Replace(bad, `href="/issue/#target"`, `href="issue/"`, 1)
	documents["/evaluate/"] = bad
	if linkFindings := verifyLinks(documents, ids); len(linkFindings) == 0 {
		t.Fatal("path-relative href should produce link-resolution finding")
	}
}

func canonicalPresentationPage(route string) presentationPage {
	for _, page := range canonicalPresentation() {
		if page.Route == route {
			return page
		}
	}
	return presentationPage{Route: route, PageKind: "reference", HeroQuestion: "Q", NextAction: "/"}
}

func containsString(slice []string, value string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}

func TestSiteCheck_HomepageCanonicalDirectionPasses(t *testing.T) {
	root, built := makeSyntheticSite(t)
	if findings := Verify(root, built); len(findings) != 0 {
		t.Fatalf("canonical homepage rejected: %#v", findings)
	}
	doc, err := os.ReadFile(filepath.Join(repositoryRoot(), "docs/index.html"))
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{`<span>./b</span><span>backstop</span><span`, "data-home-gate-proof", "data-home-system-section", "Full framework", "Artifact workflow", "Standards enforcement", "Deterministic scaffolding", "CLAIM-017", "JLINK-001", "data-next-action", "<footer"} {
		if !strings.Contains(string(doc), expected) {
			t.Fatalf("current homepage missing canonical marker %q", expected)
		}
	}
}

func TestSiteCheck_HomepageCanonicalDirectionRejectsDriftMatrix(t *testing.T) {
	replaceOnce := func(old, replacement string) func(string) (string, bool) {
		return func(document string) (string, bool) {
			if strings.Count(document, old) != 1 {
				return document, false
			}
			return strings.Replace(document, old, replacement, 1), true
		}
	}
	modeArticle := func(document, title string) (string, bool) {
		titleAt := strings.Index(document, "<h3>"+title+"</h3>")
		if titleAt < 0 {
			return "", false
		}
		start := strings.LastIndex(document[:titleAt], "<article>")
		endOffset := strings.Index(document[titleAt:], "</article>")
		if start < 0 || endOffset < 0 {
			return "", false
		}
		return document[start : titleAt+endOffset+len("</article>")], true
	}
	removeMode := func(title string) func(string) (string, bool) {
		return func(document string) (string, bool) {
			article, ok := modeArticle(document, title)
			if !ok {
				return document, false
			}
			return strings.Replace(document, article, "", 1), true
		}
	}
	reorderModes := func(firstTitle, secondTitle string) func(string) (string, bool) {
		return func(document string) (string, bool) {
			first, firstOK := modeArticle(document, firstTitle)
			second, secondOK := modeArticle(document, secondTitle)
			if !firstOK || !secondOK || !strings.Contains(document, first+second) {
				return document, false
			}
			return strings.Replace(document, first+second, second+first, 1), true
		}
	}
	duplicateMode := func(title string) func(string) (string, bool) {
		return func(document string) (string, bool) {
			article, ok := modeArticle(document, title)
			if !ok {
				return document, false
			}
			return strings.Replace(document, article, article+article, 1), true
		}
	}

	const canonicalModes = "Full framework | Artifact workflow | Standards enforcement | Deterministic scaffolding"
	mutations := []struct {
		name, phase, identity, expected, observed string
		mutate                                    func(string) (string, bool)
	}{
		{"truncated-wordmark", "homepage-canonical", "wordmark", "one ordered source-visible ./b backstop .sh owner", "0", replaceOnce(`<span>backstop</span>`, "")},
		{"field-guide-question", "homepage-canonical", "forbidden scaffold", "no field-guide question or scaffold headings", "legacy scaffold present", replaceOnce(`<section data-page-hero>`, `<section data-page-hero><h1 data-page-question>What failure does Backstop prevent?</h1>`)},
		{"missing-section", "homepage-canonical", "system sections", "exactly three ordered canonical sections", "2", replaceOnce(`data-home-system-section><span>02`, `><span>02`)},
		{"reordered-section", "homepage-canonical", "system sections", "ordered 01/02/03 Define/Enforce/Detect sections", "define-work missing or reordered", replaceOnce(`<span>01</span><h2>Define the work`, `<span>03</span><h2>Define the work`)},
		{"replacement-mode", "homepage-canonical", "composability modes", canonicalModes, "Full framework | Understand | Standards enforcement | Deterministic scaffolding", replaceOnce("Artifact workflow", "Understand")},
		{"extra-mode", "homepage-canonical", "composability modes", canonicalModes, "Full framework | Artifact workflow | Standards enforcement | Deterministic scaffolding | Extra mode", replaceOnce(`</div></section><a href="#define-work">`, `<article><h3>Extra mode</h3></article></div></section><a href="#define-work">`)},
		{"removed-canonical-mode", "homepage-canonical", "composability modes", canonicalModes, "Full framework | Standards enforcement | Deterministic scaffolding", removeMode("Artifact workflow")},
		{"truly-reordered-canonical-modes", "homepage-canonical", "composability modes", canonicalModes, "Artifact workflow | Full framework | Standards enforcement | Deterministic scaffolding", reorderModes("Full framework", "Artifact workflow")},
		{"duplicated-canonical-mode", "homepage-canonical", "composability modes", canonicalModes, "Full framework | Full framework | Artifact workflow | Standards enforcement | Deterministic scaffolding", duplicateMode("Full framework")},
		{"owner-marker", "homepage-canonical", "owner markers", "one CLAIM-017 and one JLINK-001", "claim=0 journey=1", replaceOnce("CLAIM-017", "CLAIM-REMOVED")},
		{"fragment-only-action", "homepage-canonical", "next action", "one /evaluate/ next action", "missing or drifted", replaceOnce(`href="/evaluate/">Next`, `href="#define-work">Next`)},
		{"empty-next-action", "homepage-canonical", "next action", "one /evaluate/ next action", "missing or drifted", replaceOnce(`href="/evaluate/">Next`, `href="">Next`)},
		{"path-relative-link", "homepage-canonical", "journey destination", "canonical root-relative evaluate anchor", "missing", replaceOnce(`href="/evaluate/#target"`, `href="evaluate/#target"`)},
		{"alias-link", "homepage-canonical", "journey destination", "canonical root-relative evaluate anchor", "missing", replaceOnce(`href="/evaluate/#target"`, `href="/getting-started.html"`)},
		{"empty-journey-destination", "homepage-canonical", "journey destination", "canonical root-relative evaluate anchor", "missing", replaceOnce(`href="/evaluate/#target"`, `href=""`)},
		{"wrong-primary-navigation-destination", "navigation", "/ Primary order", "Evaluate -> /evaluate/", "missing, mislabeled, or reordered", replaceOnce(`<a href="/evaluate/">Evaluate</a>`, `<a href="/wrong-evaluate/">Evaluate</a>`)},
		{"wrong-utility-navigation-destination", "navigation", "/ Utility order", "Status -> /status/", "missing, mislabeled, or reordered", replaceOnce(`<a href="/status/">Status</a>`, `<a href="/wrong-status/">Status</a>`)},
	}
	for _, mutation := range mutations {
		t.Run(mutation.name, func(t *testing.T) {
			root, built := makeSyntheticSite(t)
			path := builtRoutePath(built, "/")
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			mutated, ok := mutation.mutate(string(data))
			if !ok || mutated == string(data) {
				t.Fatalf("mutation setup %s did not alter the homepage", mutation.name)
			}
			writeTestFile(t, path, mutated)
			findings := Verify(root, built)
			matched := false
			for _, finding := range findings {
				if finding.Phase == mutation.phase && finding.Identity == mutation.identity && finding.Expected == mutation.expected && finding.Observed == mutation.observed {
					matched = true
				}
			}
			if !matched {
				t.Fatalf("homepage drift %s missing diagnostic phase=%q identity=%q expected=%q observed=%q: %#v", mutation.name, mutation.phase, mutation.identity, mutation.expected, mutation.observed, findings)
			}
		})
	}
}

func TestSiteCheck_HomepageCanonicalChangeFence(t *testing.T) {
	for _, mutation := range []struct{ old, replacement string }{
		{`data-page-kind="evaluation"`, `data-page-kind="home"`},
		{`data-page-question>Your agent already writes the code.`, `>Define the work.`},
		{`<section data-page-hero>`, `<section data-page-hero data-home-system-section>`},
	} {
		root, built := makeSyntheticSite(t)
		path := builtRoutePath(built, "/evaluate/")
		data, _ := os.ReadFile(path)
		writeTestFile(t, path, strings.Replace(string(data), mutation.old, mutation.replacement, 1))
		if findings := Verify(root, built); len(findings) == 0 {
			t.Fatalf("non-home route accepted homepage drift %q", mutation.old)
		}
	}
}

func TestSiteCheck_CanonicalRouteMatrixRejectsInvalidCell(t *testing.T) {
	for _, mutate := range []func(string, string){
		func(_ string, built string) { _ = os.Remove(builtRoutePath(built, "/model/")) },
		func(_ string, built string) { writeTestFile(t, builtRoutePath(built, "/"), "duplicate owner") },
		func(root, _ string) {
			writeTestFile(t, filepath.Join(root, "docs/_data/site-presentation.yml"), "schema_version: wrong\npages: []\n")
		},
	} {
		root, built := makeSyntheticSite(t)
		mutate(root, built)
		if findings := Verify(root, built); len(findings) == 0 {
			t.Fatal("invalid canonical route matrix passed")
		}
	}
}

func TestSiteCheck_LinkPrecedenceAcceptsFragmentQueryRootRelativeCrossOriginHTTPSAndMailto(t *testing.T) {
	root, built := makeSyntheticSite(t)
	requireCleanSite(t, root, built)
}

func TestSiteCheck_LinkPolicyFixtureAvoidsCredentialShape(t *testing.T) {
	root, built := makeSyntheticSite(t)
	source := readRepositoryFile(t, "scripts/sitecheck/site_contract_test.go")
	credentialIdentifier := regexp.MustCompile(`(?m)\b(token|password|secret)\s*:=`)
	if credentialIdentifier.MatchString(source) {
		t.Fatal("synthetic fixture construction contains a credential-shaped identifier")
	}
	mutated := strings.Replace(source, "ownerStylesheet :=", "to"+"ken :=", 1)
	if mutated == source || !credentialIdentifier.MatchString(mutated) {
		t.Fatal("credential-shaped identifier source mutation was not rejected")
	}

	hrefPattern := regexp.MustCompile(`href="([^"]*)"`)
	for _, page := range canonicalPresentation() {
		for _, match := range hrefPattern.FindAllStringSubmatch(syntheticDocument(page), -1) {
			parsed, err := url.Parse(match[1])
			if err != nil {
				t.Fatalf("clean fixture href %q is not parseable: %v", match[1], err)
			}
			if parsed.User != nil {
				t.Fatalf("clean fixture contains credential-shaped user-info bytes in %q", match[1])
			}
		}
	}
	path := builtRoutePath(built, "/")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, path, strings.Replace(string(data), `href="#define-work"`, `href="mailto:invalid recipient"`, 1))
	findings := Verify(root, built)
	for _, finding := range findings {
		if finding.Phase == "link-resolution" && finding.Expected == "valid mailto recipient" && finding.Observed == "empty or invalid" {
			return
		}
	}
	t.Fatalf("neutral malformed-mail fixture did not exercise link policy: %#v", findings)
}

func TestSiteCheck_LinkPrecedenceRejectsRelativeSameOriginAbsoluteAndForbiddenSchemes(t *testing.T) {
	for _, href := range []string{"relative.html", "https://backstop.sh/model/", "http://example.org/", "//example.org/", "file:///tmp/a", "http://127.0.0.1/"} {
		root, built := makeSyntheticSite(t)
		path := builtRoutePath(built, "/")
		data, _ := os.ReadFile(path)
		writeTestFile(t, path, strings.Replace(string(data), `href="#define-work"`, `href="`+href+`"`, 1))
		if findings := Verify(root, built); len(findings) == 0 {
			t.Fatalf("forbidden href %q passed", href)
		}
	}
}

func TestSiteCheck_LegacyRedirectMatrixPasses(t *testing.T) {
	root, built := makeSyntheticSite(t)
	requireCleanSite(t, root, built)
}

func TestSiteCheck_LegacyRedirectMatrixRejectsInvalidCell(t *testing.T) {
	root, built := makeSyntheticSite(t)
	writeTestFile(t, filepath.Join(built, "concepts.html"), `<script>location='/model/'</script>`)
	if findings := Verify(root, built); len(findings) == 0 {
		t.Fatal("script redirect passed")
	}
}

func TestSiteCheck_CustomDomainBytesPass(t *testing.T) {
	root, built := makeSyntheticSite(t)
	requireCleanSite(t, root, built)
}

func TestSiteCheck_CustomDomainRejectsInvalidMatrix(t *testing.T) {
	for _, value := range []string{"backstop.sh", "www.backstop.sh\n", "backstop.sh\r\n", ""} {
		root, built := makeSyntheticSite(t)
		writeTestFile(t, filepath.Join(root, "docs/CNAME"), value)
		if findings := Verify(root, built); len(findings) == 0 {
			t.Fatalf("invalid CNAME %q passed", value)
		}
	}
}

func TestSiteCheck_DistinguishesCanonicalMetadataFromAnchorLinks(t *testing.T) {
	root, built := makeSyntheticSite(t)
	path := builtRoutePath(built, "/model/")
	data, _ := os.ReadFile(path)
	writeTestFile(t, path, strings.Replace(string(data), `https://backstop.sh/model/`, `/model/`, 1))
	if findings := Verify(root, built); len(findings) == 0 {
		t.Fatal("root-relative canonical metadata passed")
	}
}

func TestSiteCheck_OwnerTokenAssetConsumptionPasses(t *testing.T) {
	root, built := makeSyntheticSite(t)
	if findings := VerifyInstalledDesignSystem(root, built); len(findings) != 0 {
		t.Fatalf("owner token rejected: %#v", findings)
	}
}

func TestSiteCheck_OwnerTokenAssetConsumptionRejectsInvalidMatrix(t *testing.T) {
	for _, mutate := range []func(string, string){
		func(_ string, built string) {
			writeTestFile(t, filepath.Join(built, "assets/css/design-system-tokens.css"), "changed")
		},
		func(_ string, built string) {
			path := builtRoutePath(built, "/")
			data, _ := os.ReadFile(path)
			writeTestFile(t, path, strings.Replace(string(data), "design-system-tokens.css", "missing.css", 1))
		},
	} {
		root, built := makeSyntheticSite(t)
		mutate(root, built)
		if findings := VerifyInstalledDesignSystem(root, built); len(findings) == 0 {
			t.Fatal("invalid owner token consumption passed")
		}
	}
}

func TestSiteCheck_OwnerAcceptanceExportBindingPasses(t *testing.T) {
	root, _ := makeSyntheticSite(t)
	export, err := LoadOwnerAcceptanceExport(root)
	if err != nil || len(export.Cells) != 7 {
		t.Fatalf("export cells=%d err=%v", len(export.Cells), err)
	}
	wordmark := export.Cells[5]
	before, decodeErr := base64.StdEncoding.DecodeString(wordmark.Mutation.UniqueBeforeBase64)
	if decodeErr != nil || string(before) != `<span>./b</span><span>backstop</span><span>.sh</span>` {
		t.Fatalf("wordmark mutation source=%q err=%v", before, decodeErr)
	}
}

func TestSiteCheck_OwnerAcceptanceExportRejectsSchemaAndFidelityMatrix(t *testing.T) {
	for _, oldNew := range [][2]string{{"public-site-acceptance/v1", "wrong/v1"}, {"version: 0.1.5", "version: 9.9.9"}, {"target_relative_path: index.html", "target_relative_path: other.html"}} {
		root, _ := makeSyntheticSite(t)
		path := filepath.Join(root, ".backstop/packs/backstop-ai/backstop-design-system/contracts/public-site-acceptance.yml")
		data, _ := os.ReadFile(path)
		writeTestFile(t, path, strings.Replace(string(data), oldNew[0], oldNew[1], 1))
		if _, err := LoadOwnerAcceptanceExport(root); err == nil {
			t.Fatalf("export mutation %q unexpectedly passed", oldNew)
		}
	}
}

func TestSiteCheck_EightIsolatedProjectRootsAndCleanCorpusPass(t *testing.T) {
	root, built := makeSyntheticSite(t)
	writeTestFile(t, filepath.Join(root, "backstop.lock"), `packs:
  backstop-ai/backstop-design-system:
    content_hash: aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
    git_ref: v0.1.5
    name: backstop-ai/backstop-design-system
    source_coordinate: backstop-ai/backstop-design-system
    source_type: git
    version: 0.1.5
`)
	fake := `#!/usr/bin/env bash
set -euo pipefail
if [[ ${1-} == pack && ${2-} == install ]]; then mkdir -p .backstop/packs; exit 0; fi
identity=${PWD##*/}
if [[ $identity == clean ]]; then
  printf '%s\n' '{"schema_version":"gate/v1","pass":true,"total_violations":0,"steps":[]}' > gate-report.json
  exit 0
fi
cell=${identity#negative-}
case "$cell" in
  token) rule=no-raw-colors; file=_site/assets/css/site.css ;;
  inline-style) rule=no-inline-styles; file=_site/index.html ;;
  focus) rule=focus-visible-required; file=_site/assets/css/site.css ;;
  reduced-motion) rule=reduced-motion-required; file=_site/assets/css/site.css ;;
  accessibility) rule=accessible-site-shell; file=_site/index.html ;;
  wordmark) rule=canonical-wordmark; file=_site/index.html ;;
  reusable-presentation) rule=reusable-page-hero; file=_site/index.html ;;
esac
printf '{"schema_version":"gate/v1","pass":false,"total_violations":1,"steps":[{"step_name":"pack_engines","violations":[{"rule":"%s","file":"%s","source_pack":"backstop-ai/backstop-design-system"}]}]}\n' "$rule" "$file" > gate-report.json
exit 1
`
	fakePath := filepath.Join(root, "bin/backstop")
	writeTestFile(t, fakePath, fake)
	if err := os.Chmod(fakePath, 0o755); err != nil {
		t.Fatal(err)
	}
	export, _ := LoadOwnerAcceptanceExport(root)
	if findings := VerifyEightIsolatedCorpora(root, built, export); len(findings) != 0 {
		t.Fatalf("corpus rejected: %#v", findings)
	}
}
