package main

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

type presentationDocument struct {
	SchemaVersion string             `yaml:"schema_version"`
	Pages         []presentationPage `yaml:"pages"`
}

type presentationPage struct {
	Route        string   `yaml:"route"`
	PageKind     string   `yaml:"page_kind"`
	HeroQuestion string   `yaml:"hero_question"`
	Treatments   []string `yaml:"treatments"`
	NextAction   string   `yaml:"next_action"`
}

func canonicalPresentation() []presentationPage {
	return []presentationPage{
		{Route: "/", PageKind: "home", HeroQuestion: "What failure does Backstop prevent?", Treatments: []string{"evidence-cards"}, NextAction: "/evaluate/"},
		{Route: "/evaluate/", PageKind: "evaluation", HeroQuestion: "Your agent already writes the code.", Treatments: []string{"evidence-cards", "boundary-callouts"}, NextAction: "/model/"},
		{Route: "/model/", PageKind: "model", HeroQuestion: "How it works", Treatments: []string{"evidence-cards", "local-overflow"}, NextAction: "/adopt/"},
		{Route: "/adopt/", PageKind: "adoption", HeroQuestion: "Try it out.", Treatments: []string{"evidence-cards"}, NextAction: "/use-cases/"},
		{Route: "/use-cases/", PageKind: "use-cases", HeroQuestion: "Which problem-oriented adoption path applies?", Treatments: []string{"evidence-cards", "boundary-callouts"}, NextAction: "/packs/"},
		{Route: "/packs/", PageKind: "ecosystem", HeroQuestion: "Which maintained pack already owns this standard?", Treatments: []string{"evidence-cards", "generated-regions", "local-overflow"}, NextAction: "/extend/"},
		{Route: "/extend/", PageKind: "extension", HeroQuestion: "When should this concern become a pack?", Treatments: []string{"evidence-cards", "boundary-callouts"}, NextAction: "/reference/"},
		{Route: "/reference/", PageKind: "reference", HeroQuestion: "What exact interface or behavior do I need?", Treatments: []string{"generated-regions", "local-overflow"}, NextAction: "/status/"},
		{Route: "/status/", PageKind: "status", HeroQuestion: "What is supported, limited, planned, or intentionally outside Backstop?", Treatments: []string{"evidence-cards", "boundary-callouts", "generated-regions", "local-overflow"}, NextAction: "/contributing/"},
		{Route: "/contributing/", PageKind: "contributing", HeroQuestion: "How can I participate in Backstop and its ecosystem?", Treatments: []string{"boundary-callouts"}, NextAction: "/"},
	}
}

func loadPresentation(t *testing.T) presentationDocument {
	t.Helper()
	path := filepath.Join("..", "..", "docs", "_data", "site-presentation.yml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var document presentationDocument
	if err := yaml.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	return document
}

func validatePresentation(document presentationDocument) bool {
	return document.SchemaVersion == "backstop-core/site-presentation/v1" && reflect.DeepEqual(document.Pages, canonicalPresentation())
}

func TestSiteCheck_NavigationMatrixPasses(t *testing.T) {
	if document := loadPresentation(t); !validatePresentation(document) {
		t.Fatal("site presentation does not byte-match the accepted ten-row matrix")
	}
	home, err := os.ReadFile(filepath.Join("..", "..", "docs", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	ordered := []string{`href="/evaluate/">Evaluate`, `href="/model/">Model`, `href="/adopt/">Adopt`, `href="/use-cases/">Use Cases`, `href="/packs/">Packs`, `href="/extend/">Extend`, `href="/reference/">Reference`, `href="/status/">Status`, `href="/contributing/">Contributing`}
	position := -1
	for _, expected := range ordered {
		next := strings.Index(string(home), expected)
		if next <= position {
			t.Fatalf("navigation label/destination absent or reordered: %s", expected)
		}
		position = next
	}
}

func TestSiteCheck_NavigationMatrixRejectsInvalidCell(t *testing.T) {
	mutations := []func(*presentationDocument){
		func(d *presentationDocument) { d.Pages[0].Route = "/home/" },
		func(d *presentationDocument) { d.Pages[1].PageKind = "home" },
		func(d *presentationDocument) { d.Pages[2].HeroQuestion += " changed" },
		func(d *presentationDocument) {
			d.Pages[3].Treatments = append(d.Pages[3].Treatments, "boundary-callouts")
		},
		func(d *presentationDocument) { d.Pages[4].NextAction = "/" },
		func(d *presentationDocument) { d.Pages = append(d.Pages, d.Pages[0]) },
		func(d *presentationDocument) { d.Pages[0], d.Pages[1] = d.Pages[1], d.Pages[0] },
	}
	for index, mutate := range mutations {
		document := presentationDocument{SchemaVersion: "backstop-core/site-presentation/v1", Pages: append([]presentationPage(nil), canonicalPresentation()...)}
		mutate(&document)
		if validatePresentation(document) {
			t.Fatalf("mutation %d unexpectedly passed", index)
		}
	}
}

func readRepoFile(t *testing.T, relative string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", relative))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestSiteCheck_ModelPaperInkChromeAndHomeReflowPasses(t *testing.T) {
	layout := readRepoFile(t, "docs/_layouts/default.html")
	css := readRepoFile(t, "docs/assets/css/site.css")

	paperKinds := map[string]bool{}
	for _, kind := range strings.Split(extractAssignSplit(layout, "paper_kinds"), ",") {
		kind = strings.TrimSpace(kind)
		if kind != "" {
			paperKinds[kind] = true
		}
	}
	if !paperKinds["model"] {
		t.Fatal("paper-kind list must contain model")
	}
	if paperKinds["home"] {
		t.Fatal("paper-kind list must not contain home")
	}
	if !strings.Contains(layout, `<meta name="theme-color" content="#0c0d0d">`) {
		t.Fatal("paper branch must emit theme-color #0c0d0d")
	}
	if !strings.Contains(layout, `<link rel="stylesheet" href="/assets/css/backstop-tokens.css">`) {
		t.Fatal("paper branch must link backstop-tokens.css")
	}
	if !strings.Contains(layout, `<meta name="color-scheme" content="dark">`) {
		t.Fatal("non-paper branch must retain dark color-scheme")
	}

	if !strings.Contains(css, `html:has([data-page-kind="evaluation"], [data-page-kind="model"], [data-page-kind="adoption"]`) {
		t.Fatal("html:has paper remap must include evaluation, model, and adoption page kinds")
	}
	for _, token := range []string{"--ds-canvas", "--ds-text", "--ds-accent", "--ds-border"} {
		if !strings.Contains(css, token) {
			t.Fatalf("paper remap missing token mapping for %s", token)
		}
	}
	if !strings.Contains(css, `[data-page-kind="model"] .canonical-anchors`) || !strings.Contains(css, `clip: rect(0, 0, 0, 0)`) {
		t.Fatal("model canonical hidden pattern missing clip rule")
	}
	for _, surface := range []string{".work-topology", ".work-paths", ".model-figure", ".figure-bridge", ".loop-core"} {
		if !strings.Contains(css, `[data-page-kind="model"] `+surface) {
			t.Fatalf("model stylesheet missing surface %s", surface)
		}
	}
	workPathsRule := regexp.MustCompile(`\[data-page-kind="model"\] \.work-paths[^}]*\{[^}]*\}`)
	if match := workPathsRule.FindString(css); match == "" || !strings.Contains(match, "border-top") || !strings.Contains(match, "var(--ds-border)") || !strings.Contains(match, "padding-top") {
		t.Fatal("work-paths must carry hairline divider using ds-border token")
	}
	labelRule := regexp.MustCompile(`\[data-page-kind="model"\] \.work-path-label[^}]*\{[^}]*\}`)
	if match := labelRule.FindString(css); match == "" || !strings.Contains(match, "var(--ds-muted)") || !strings.Contains(match, "font-size") {
		t.Fatal("work-path-label must be muted and smaller than body lede")
	}

	media56 := extractMediaBlock(css, "max-width: 56rem")
	if !strings.Contains(media56, `[data-page-kind="home"] .nav `) || !strings.Contains(media56, `[data-page-kind="home"] .nav-links`) {
		t.Fatal("56rem media block must retain legacy home nav rules")
	}
	if !strings.Contains(css, "grid-template-columns: minmax(0, 1fr) minmax(0, 1fr)") {
		t.Fatal("30rem navigation grid literal missing")
	}
	for _, surface := range []string{".model-figure", ".work-topology"} {
		rule := regexp.MustCompile(regexp.QuoteMeta(`[data-page-kind="model"] `+surface) + `[^{]*\{[^}]*\}`)
		for _, match := range rule.FindAllString(css, -1) {
			if strings.Contains(match, "background:") || strings.Contains(match, "border:") {
				t.Fatalf("slide-frame chrome forbidden on %s", surface)
			}
		}
	}
	modelStart := strings.Index(css, `[data-page-kind="model"]`)
	modelEnd := strings.Index(css, `html:has([data-page-kind="evaluation"])`)
	modelBlock := css
	if modelStart >= 0 && modelEnd > modelStart {
		modelBlock = css[modelStart:modelEnd]
	}
	if regexp.MustCompile(`#[0-9a-fA-F]{3,8}\b`).MatchString(modelBlock) {
		t.Fatal("model stylesheet block must not contain raw hex color literals")
	}
}

func extractAssignSplit(layout, name string) string {
	pattern := regexp.MustCompile(`\{% assign ` + name + ` = "([^"]+)" \| split: "," %\}`)
	match := pattern.FindStringSubmatch(layout)
	if len(match) < 2 {
		return ""
	}
	return match[1]
}

func TestSiteCheck_AdoptionPaperInkChromeIsDeclared(t *testing.T) {
	layout := readRepoFile(t, "docs/_layouts/default.html")
	css := readRepoFile(t, "docs/assets/css/site.css")

	paperKinds := map[string]bool{}
	for _, kind := range strings.Split(extractAssignSplit(layout, "paper_kinds"), ",") {
		kind = strings.TrimSpace(kind)
		if kind != "" {
			paperKinds[kind] = true
		}
	}
	if !paperKinds["adoption"] {
		t.Fatal("paper-kind list must contain adoption")
	}
	if !strings.Contains(layout, `<link rel="stylesheet" href="/assets/css/backstop-tokens.css">`) {
		t.Fatal("paper branch must link backstop-tokens.css")
	}

	if !strings.Contains(css, `[data-page-kind="adoption"]`) || !strings.Contains(css, `html:has([data-page-kind="evaluation"], [data-page-kind="model"], [data-page-kind="adoption"]`) {
		t.Fatal("html:has paper remap must include adoption page kind")
	}
	usedModelRule := regexp.MustCompile(`\[data-page-kind="adoption"\] #used-the-model[^}]*\{[^}]*\}`)
	if match := usedModelRule.FindString(css); match == "" || !strings.Contains(match, "border-top") {
		t.Fatal("adoption #used-the-model must carry border-top")
	}
	canonicalRule := regexp.MustCompile(`\[data-page-kind="adoption"\] \.canonical-note[^}]*\{[^}]*\}`)
	if match := canonicalRule.FindString(css); match == "" || !strings.Contains(match, "clip:") || strings.Contains(match, "display: none") {
		t.Fatal("adoption canonical-note must use clip visually-hidden pattern")
	}

	media56 := extractMediaBlock(css, "max-width: 56rem")
	if !strings.Contains(media56, `[data-page-kind="home"] .nav `) || !strings.Contains(media56, `[data-page-kind="home"] .nav-links`) {
		t.Fatal("56rem media block must retain legacy home nav rules")
	}

	adoptionStart := strings.Index(css, `[data-page-kind="adoption"]`)
	adoptionEnd := strings.Index(css, `html:has([data-page-kind="evaluation"]`)
	adoptionBlock := css
	if adoptionStart >= 0 && adoptionEnd > adoptionStart {
		adoptionBlock = css[adoptionStart:adoptionEnd]
	}
	if regexp.MustCompile(`#[0-9a-fA-F]{3,8}\b`).MatchString(adoptionBlock) {
		t.Fatal("adoption stylesheet block must not contain raw hex color literals")
	}
}

func TestSiteCheck_EntityPaperInkChromeIsInLockstep(t *testing.T) {
	layout := readRepoFile(t, "docs/_layouts/default.html")
	hero := readRepoFile(t, "docs/_includes/page-hero.html")
	css := readRepoFile(t, "docs/assets/css/site.css")
	presentation := loadPresentation(t)

	if len(presentation.Pages) != len(canonicalRoutes()) {
		t.Fatalf("presentation rows = %d, want %d", len(presentation.Pages), len(canonicalRoutes()))
	}
	if !reflect.DeepEqual(presentation.Pages, canonicalPresentation()) {
		t.Fatal("site presentation does not byte-match the accepted ten-row matrix")
	}
	entityRoutes := []string{"/plan/", "/issue/", "/spec/", "/bundle/", "/pack/", "/directive/", "/adr/", "/capability/"}
	for _, route := range entityRoutes {
		for _, page := range presentation.Pages {
			if page.Route == route {
				t.Fatalf("entity route %q appears in site-presentation.yml", route)
			}
		}
	}

	if !strings.Contains(layout, `presentation.page_kind | default: page.page_kind`) {
		t.Fatal("layout must fall back to page.page_kind when presentation lookup is empty")
	}
	paperKinds := map[string]bool{}
	for _, kind := range strings.Split(extractAssignSplit(layout, "paper_kinds"), ",") {
		kind = strings.TrimSpace(kind)
		if kind != "" {
			paperKinds[kind] = true
		}
	}
	for _, kind := range []string{"evaluation", "model", "adoption", "entity"} {
		if !paperKinds[kind] {
			t.Fatalf("paper-kind list must contain %q", kind)
		}
	}
	if !strings.Contains(layout, `data-page-kind="{{ page_kind }}"`) {
		t.Fatal("layout must emit derived page_kind in data-page-kind")
	}
	if !strings.Contains(layout, `{% if presentation.next_action %}`) {
		t.Fatal("next-action block must be guarded on presentation.next_action")
	}
	if strings.Contains(layout, `href="{{ presentation.next_action }}">Next`) && !strings.Contains(layout, `{% if presentation.next_action %}`) {
		t.Fatal("unguarded next-action would emit empty href on entity pages")
	}
	if !strings.Contains(hero, `include.presentation.hero_question | default: page.hero_question`) {
		t.Fatal("page-hero must fall back to page.hero_question")
	}
	if !strings.Contains(hero, `page.hero_lede`) {
		t.Fatal("page-hero must render page.hero_lede channel")
	}

	for _, surface := range []string{"entity-meta", "entity-table", "entity-illegal", "entity-also"} {
		if !strings.Contains(css, `[data-page-kind="entity"] .`+surface) {
			t.Fatalf("entity stylesheet missing .%s", surface)
		}
	}
	if !strings.Contains(css, `html:has([data-page-kind="evaluation"], [data-page-kind="model"], [data-page-kind="adoption"], [data-page-kind="entity"]`) {
		t.Fatal("html:has paper remap must include entity page kind")
	}
	media56 := extractMediaBlock(css, "max-width: 56rem")
	if !strings.Contains(media56, `[data-page-kind="home"] .nav `) || !strings.Contains(media56, `[data-page-kind="home"] .nav-links`) || !strings.Contains(media56, `[data-page-kind="home"] h1`) {
		t.Fatal("56rem media block must retain legacy home nav rules")
	}
}

func extractMediaBlock(css, query string) string {
	start := strings.Index(css, "@media ("+query+")")
	if start < 0 {
		return ""
	}
	depth := 0
	for index := start; index < len(css); index++ {
		switch css[index] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return css[start : index+1]
			}
		}
	}
	return css[start:]
}
