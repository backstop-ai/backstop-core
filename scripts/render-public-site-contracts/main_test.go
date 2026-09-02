package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testCommit = "0123456789abcdef0123456789abcdef01234567"

func writeFixture(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func makeRenderFixture(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	built := filepath.Join(root, "_site")
	writeFixture(t, filepath.Join(root, "docs/_data/content-topology.yml"), `journey_links:
  - {link_id: JLINK-001, source_route: /, source_anchor: start, destination_route: /model/, destination_anchor: target, label: Continue}
  - {link_id: JLINK-024, source_route: /, source_anchor: adjacent, destination_route: /contributing/, destination_anchor: external, label: Continue outside}
adoption_instructions:
  - {instruction_id: ADOPT-INSTALL, owner_route: /, owner_anchor: install, command_text: "echo ok", command_sha256: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
`)
	writeFixture(t, filepath.Join(root, "docs/_data/evidence-inventory.yml"), `claims:
  - {claim_id: CLAIM-001, owner: {route: /, anchor: start}, boundary_id: ""}
  - {claim_id: CLAIM-002, owner: {route: /, anchor: limit}, boundary_id: BOUNDARY-002}
  - {claim_id: CLAIM-005, owner: {route: /, anchor: adjacent}, boundary_id: BOUNDARY-005}
`)
	writeFixture(t, filepath.Join(root, "docs/_data/product-model.yml"), `boundaries:
  - {boundary_id: BOUNDARY-002, state: unsupported, owner: {route: /, anchor: limit}, claim_id: CLAIM-002, explanation_markdown: "Unsupported bytes.", continuation: null, guarantee_denial_markdown: null}
  - boundary_id: BOUNDARY-005
    state: adjacent-guidance
    owner: {route: /, anchor: adjacent}
    claim_id: CLAIM-005
    explanation_markdown: "Adjacent bytes."
    continuation: {journey_link_id: JLINK-024, route: /contributing/, anchor: external, label: Continue outside}
    guarantee_denial_markdown: "Denial bytes."
`)
	writeFixture(t, filepath.Join(built, "index.html"), `<html><main id="main"><h2 id="start">Start</h2>
<!-- backstop-claim: CLAIM-001 -->
<p>Evidence bytes.
<!-- /backstop-claim --></p>
<h2 id="limit">Limit</h2>
<!-- backstop-claim: CLAIM-002 -->
<p>Unsupported bytes.
<!-- /backstop-claim --></p>
<h2 id="adjacent">Adjacent</h2>
<!-- backstop-claim: CLAIM-005 -->
<p>Adjacent bytes.</p>
<!-- backstop-journey-link: JLINK-024 -->
<p><a href="/contributing/#external">Continue outside</a></p>
<p>Denial bytes.
<!-- /backstop-claim --></p>
<!-- backstop-journey-link: JLINK-001 -->
<p><a href="/model/#target">Continue</a></p>
<h2 id="install">Install</h2><pre><code>echo ok</code></pre>
<section data-generated-region="" data-product-truth-job="job"><ul><li data-generated-source-descriptor="" data-source-kind="blob" data-commit-binding="site" data-source-path="source">https://github.com/backstop-ai/backstop-core/blob/&lt;SITE-COMMIT&gt;/source</li></ul></section>
<table><tr><td>value</td></tr></table></main></html>`)
	return root, built
}

func TestRenderPublicSiteContracts_FullFixturePasses(t *testing.T) {
	root, built := makeRenderFixture(t)
	if findings := Render(root, built, testCommit); len(findings) != 0 {
		t.Fatalf("render failed: %#v", findings)
	}
	data, err := os.ReadFile(filepath.Join(built, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	doc := string(data)
	for _, expected := range []string{`<article data-evidence-card data-claim-id="CLAIM-001">`, `data-journey-link-id="JLINK-001"`, `data-adoption-instruction-id="ADOPT-INSTALL"`, `data-generated-source-link`, `data-source-path="source"`, `>source</a>`, testCommit, `data-overflow-region`} {
		if !strings.Contains(doc, expected) {
			t.Fatalf("rendered contract missing %q", expected)
		}
	}
	if strings.Contains(doc, "backstop-claim") || strings.Contains(doc, "SITE-COMMIT") || strings.Contains(doc, "</article></p>") {
		t.Fatalf("render left marker or invalid wrapper: %s", doc)
	}
}

func TestRenderPublicSiteContracts_RejectsBindingMatrix(t *testing.T) {
	for _, mutation := range []func(string, string){
		func(root, _ string) {
			writeFixture(t, filepath.Join(root, "docs/_data/content-topology.yml"), "journey_links: [")
		},
		func(_, built string) { writeFixture(t, filepath.Join(built, "index.html"), "missing owner markers") },
		func(_, built string) {
			path := filepath.Join(built, "index.html")
			data, _ := os.ReadFile(path)
			writeFixture(t, path, strings.Replace(string(data), "/model/#target", "/wrong/#target", 1))
		},
	} {
		root, built := makeRenderFixture(t)
		mutation(root, built)
		if findings := Render(root, built, testCommit); len(findings) == 0 {
			t.Fatal("invalid owner binding passed")
		}
	}
	root, built := makeRenderFixture(t)
	if findings := Render(root, built, "short"); len(findings) == 0 {
		t.Fatal("abbreviated site commit passed")
	}
}

func TestRenderPublicSiteContracts_HelperRefusals(t *testing.T) {
	if _, err := replaceOnce("two two", "two", "one", "identity"); err == nil {
		t.Fatal("duplicate replace binding passed")
	}
	if got := attribute(`data-source-kind="blob"`, "data-source-kind"); got != "blob" {
		t.Fatalf("attribute=%q", got)
	}
	if got := attribute(`data-source-kind="blob"`, "missing"); got != "" {
		t.Fatalf("missing attribute=%q", got)
	}
	if got := wrapTables("<table>unterminated", "/"); got != "<table>unterminated" {
		t.Fatalf("unterminated table changed: %q", got)
	}
}
