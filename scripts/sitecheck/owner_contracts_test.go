package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const ownerTestCommit = "0123456789abcdef0123456789abcdef01234567"

func makeOwnerContractSite(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	built := filepath.Join(root, "_site")
	for _, route := range canonicalRoutes() {
		writeTestFile(t, builtRoutePath(built, route), "<html><main id=\"main\"></main></html>")
	}
	var content strings.Builder
	content.WriteString(`<html><main id="main">`)
	for index := 1; index <= 23; index++ {
		fmt.Fprintf(&content, `<a data-journey-link-id="JLINK-%03d" href="/model/#target">Journey %03d</a>`, index, index)
	}
	for index := 1; index <= 4; index++ {
		fmt.Fprintf(&content, `<aside data-boundary-callout data-boundary-id="BOUNDARY-%03d" data-boundary-state="out-of-scope"><p data-boundary-explanation>Boundary %03d explanation.</p></aside>`, index, index)
	}
	content.WriteString(`<aside data-boundary-callout data-boundary-id="BOUNDARY-005" data-boundary-state="adjacent-guidance"><p data-boundary-explanation>Explanation bytes.</p><p><a data-journey-link-id="JLINK-024" data-boundary-continuation href="/extend/#pack-shape">Continuation bytes</a></p><p data-boundary-guarantee-denial>Denial bytes.</p></aside>`)
	for _, id := range []string{"ADOPT-INSTALL", "ADOPT-CONFIGURE", "ADOPT-ENFORCE"} {
		fmt.Fprintf(&content, `<pre data-adoption-instruction-id="%s" data-command-sha256="sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"><code>echo ok</code></pre>`, id)
	}
	for _, job := range []string{"cli-command-catalog", "artifact-schema-catalog", "installed-pack-catalog", "release-history"} {
		fmt.Fprintf(&content, `<section data-generated-region="" data-product-truth-job="%s"><a data-generated-source-link data-source-kind="blob" data-source-commit="%s" href="https://github.com/backstop-ai/backstop-core/blob/%s/source">Source</a></section>`, job, ownerTestCommit, ownerTestCommit)
	}
	content.WriteString(`</main></html>`)
	writeTestFile(t, builtRoutePath(built, "/"), content.String())
	return root, built
}

func requireOwnerContractsPass(t *testing.T, root, built string) {
	t.Helper()
	if findings := VerifyRenderedOwnerContracts(root, built, ownerTestCommit); len(findings) != 0 {
		t.Fatalf("owner contracts rejected: %#v", findings)
	}
}

func requireOwnerMutationFails(t *testing.T, old, replacement string) {
	t.Helper()
	root, built := makeOwnerContractSite(t)
	path := builtRoutePath(built, "/")
	data, _ := os.ReadFile(path)
	changed := strings.Replace(string(data), old, replacement, 1)
	if changed == string(data) {
		t.Fatalf("owner mutation source absent: %q", old)
	}
	writeTestFile(t, path, changed)
	if findings := VerifyRenderedOwnerContracts(root, built, ownerTestCommit); len(findings) == 0 {
		t.Fatalf("owner mutation %q unexpectedly passed", old)
	}
}

func TestSiteCheck_RenderedJourneyLinkMatrixPasses(t *testing.T) {
	root, built := makeOwnerContractSite(t)
	requireOwnerContractsPass(t, root, built)
}

func TestSiteCheck_RenderedJourneyLinkMatrixRejectsInvalidCell(t *testing.T) {
	requireOwnerMutationFails(t, `data-journey-link-id="JLINK-007"`, `data-journey-link-id="JLINK-099"`)
}

func TestSiteCheck_StructuredBoundaryRenderingPasses(t *testing.T) {
	root, built := makeOwnerContractSite(t)
	requireOwnerContractsPass(t, root, built)
}

func TestSiteCheck_StructuredBoundaryRenderingRejectsInvalidMatrix(t *testing.T) {
	for _, mutation := range [][2]string{
		{`data-boundary-state="out-of-scope"`, `data-boundary-state=""`},
		{`data-boundary-explanation`, `data-unowned-explanation`},
		{`BOUNDARY-003`, `BOUNDARY-099`},
	} {
		requireOwnerMutationFails(t, mutation[0], mutation[1])
	}
}

func TestSiteCheck_GeneratedSourceLinkRenderingPasses(t *testing.T) {
	root, built := makeOwnerContractSite(t)
	requireOwnerContractsPass(t, root, built)
}

func TestSiteCheck_GeneratedSourceLinkRenderingRejectsInvalidMatrix(t *testing.T) {
	for _, mutation := range [][2]string{
		{ownerTestCommit + `/source`, `main/source`},
		{`data-product-truth-job="release-history"`, `data-product-truth-job="parallel-history"`},
		{`data-generated-source-link`, `data-unowned-source-link`},
	} {
		requireOwnerMutationFails(t, mutation[0], mutation[1])
	}
}

func TestSiteCheck_AdoptionInstructionRenderingPasses(t *testing.T) {
	root, built := makeOwnerContractSite(t)
	requireOwnerContractsPass(t, root, built)
}

func TestSiteCheck_AdoptionInstructionRenderingRejectsInvalidMatrix(t *testing.T) {
	for _, mutation := range [][2]string{
		{`ADOPT-CONFIGURE`, `ADOPT-UNKNOWN`},
		{`sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa`, `sha256:short`},
		{`<code>echo ok</code>`, `<code></code>`},
	} {
		requireOwnerMutationFails(t, mutation[0], mutation[1])
	}
}

func TestSiteCheck_RenderedOwnerContractsRejectAuthoritySubstitutes(t *testing.T) {
	for _, mutation := range [][2]string{
		{ownerTestCommit + `/source`, `HEAD/source`},
		{`data-command-sha256`, `data-locally-inferred-sha256`},
		{`data-boundary-state`, `data-prose-derived-state`},
	} {
		requireOwnerMutationFails(t, mutation[0], mutation[1])
	}
}

func TestSiteCheck_EmbeddedJLINK024DualIdentityPasses(t *testing.T) {
	root, built := makeOwnerContractSite(t)
	requireOwnerContractsPass(t, root, built)
}

func TestSiteCheck_EmbeddedJLINK024RejectsIdentityCardinalityMatrix(t *testing.T) {
	for _, mutation := range [][2]string{
		{`data-journey-link-id="JLINK-024" data-boundary-continuation`, `data-journey-link-id="JLINK-024"`},
		{`</main>`, `<a data-journey-link-id="JLINK-024" href="/extend/#pack-shape">Duplicate</a></main>`},
	} {
		requireOwnerMutationFails(t, mutation[0], mutation[1])
	}
}

func TestSiteCheck_EmbeddedJLINK024RejectsContainmentOrderAndVisibleBytesMatrix(t *testing.T) {
	for _, mutation := range [][2]string{
		{`Explanation bytes.`, ``},
		{`Continuation bytes`, ``},
		{`Denial bytes.`, ``},
		{`<p data-boundary-explanation>Explanation bytes.</p><p><a`, `<p><a`},
	} {
		requireOwnerMutationFails(t, mutation[0], mutation[1])
	}
}

func TestSiteCheck_UpstreamOwnershipAndGeneratedRegionsPass(t *testing.T) {
	root, built := makeOwnerContractSite(t)
	requireOwnerContractsPass(t, root, built)
}

func TestSiteCheck_RejectsUpstreamOwnershipViolationMatrix(t *testing.T) {
	requireOwnerMutationFails(t, `data-generated-region=""`, `data-generated-copy=""`)
}
