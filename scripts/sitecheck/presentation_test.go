package main

import (
	"os"
	"path/filepath"
	"reflect"
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
		{Route: "/reference/", PageKind: "reference", HeroQuestion: "Exact interfaces, schemas, lifecycle rules, and integration behavior.", Treatments: []string{"generated-regions", "local-overflow"}, NextAction: "/status/"},
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
