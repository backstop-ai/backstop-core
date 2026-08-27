package main

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
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
	for _, destination := range primaryNavigation {
		current := ""
		if page.Route == destination {
			current = ` aria-current="page"`
		}
		fmt.Fprintf(&primary, `<a href="%s"%s>%s</a>`, destination, current, destination)
	}
	for _, destination := range utilityNavigation {
		current := ""
		if page.Route == destination {
			current = ` aria-current="page"`
		}
		fmt.Fprintf(&utility, `<a href="%s"%s>%s</a>`, destination, current, destination)
	}
	return fmt.Sprintf(`<!doctype html><html><head><link rel="canonical" href="https://backstop.sh%s"><link rel="stylesheet" href="/assets/css/design-system-tokens.css"><link rel="stylesheet" href="/assets/css/site.css"></head><body data-site-shell="field-guide-v1"><header><a data-backstop-wordmark href="/"><span>./b</span><span>.sh</span></a><nav aria-label="Primary">%s</nav><nav aria-label="Utility">%s</nav></header><main id="main" data-page-route="%s"><section data-page-hero><h1 data-page-question>%s</h1></section><article class="prose" data-page-kind="%s"><h2 id="target">Target</h2><a href="#target">fragment</a><a href="?mode=read">query</a><a href="https://example.org/reference">external</a><a href="mailto:maintainer@example.org">mail</a></article></main><nav data-next-action><a href="%s">Next</a></nav></body></html>`, page.Route, primary.String(), utility.String(), page.Route, page.HeroQuestion, page.PageKind, page.NextAction)
}

func makeSyntheticSite(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	built := filepath.Join(root, "_site")
	writeTestFile(t, filepath.Join(root, "docs/_data/site-presentation.yml"), "schema_version: backstop-core/site-presentation/v1\npages:\n")
	var presentation strings.Builder
	presentation.WriteString("schema_version: backstop-core/site-presentation/v1\npages:\n")
	for _, page := range canonicalPresentation() {
		fmt.Fprintf(&presentation, "  - route: %s\n    page_kind: %s\n    hero_question: %s\n    treatments: [%s]\n    next_action: %s\n", page.Route, page.PageKind, page.HeroQuestion, strings.Join(page.Treatments, ", "), page.NextAction)
		writeTestFile(t, builtRoutePath(built, page.Route), syntheticDocument(page))
	}
	writeTestFile(t, filepath.Join(root, "docs/_data/site-presentation.yml"), presentation.String())
	for alias, destination := range legacyRedirects {
		writeTestFile(t, filepath.Join(built, alias), fmt.Sprintf(`<link rel="canonical" href="https://backstop.sh%s"><meta http-equiv="refresh" content="0; url=%s"><a href="%s">Continue</a>`, destination, destination, destination))
	}
	writeTestFile(t, filepath.Join(root, "docs/CNAME"), "backstop.sh\n")
	writeTestFile(t, filepath.Join(built, "CNAME"), "backstop.sh\n")
	token := "/* owner token */\n"
	digest := sha256.Sum256([]byte(token))
	writeTestFile(t, filepath.Join(built, "assets/css/design-system-tokens.css"), token)
	writeTestFile(t, filepath.Join(built, "assets/css/site.css"), "body { color: var(--ds-text); }\n:focus-visible { outline: var(--ds-focus-ring); }\n@media (prefers-reduced-motion: reduce) {}\n")
	packManifest := "name: backstop-ai/backstop-design-system\nversion: 0.1.2\n"
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
			before, replacement = "PHNwYW4+Li9iPC9zcGFuPjxzcGFuPi5zaDwvc3Bhbj4=", "PHNwYW4+Li94PC9zcGFuPjxzcGFuPi5zaDwvc3Bhbj4="
		case "reusable-presentation":
			before, replacement = "PHNlY3Rpb24gZGF0YS1wYWdlLWhlcm8+", "PHNlY3Rpb24gZGF0YS1wYWdlLWhlcm8+PGgyPkR1cGxpY2F0ZTwvaDI+PC9zZWN0aW9uPjxzZWN0aW9uIGRhdGEtcGFnZS1oZXJvPg=="
		}
		fmt.Fprintf(&cells, `  - id: %s
    rule_id: %s
    path_filters: {include: [%s], exclude: ["vendor/**"]}
    clean_fixture: fixtures/%s-clean.html
    negative_fixture: fixtures/%s-bad.html
    mutation: {target_relative_path: %s, unique_before_base64: %s, replacement_base64: %s}
    path_fidelity: {fixture_relative_path: fixtures/%s-bad.html, target_relative_path: %s, dispatch_evidence_ref: release-evidence/v0.1.2.yml#%s}
`, id, rule, target, id, id, target, before, replacement, id, target, id)
	}
	export := fmt.Sprintf(`schema_version: backstop-design-system/public-site-acceptance/v1
subject: {manifest_identity: backstop-ai/backstop-design-system, version: 0.1.2, ruleset_version: 1.2.0}
export_fingerprint_binding: release-evidence/v0.1.2.yml#public_site_acceptance
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

func TestSiteCheck_LinkPrecedenceRejectsRelativeSameOriginAbsoluteAndForbiddenSchemes(t *testing.T) {
	for _, href := range []string{"relative.html", "https://backstop.sh/model/", "http://example.org/", "//example.org/", "file:///tmp/a", "http://127.0.0.1/"} {
		root, built := makeSyntheticSite(t)
		path := builtRoutePath(built, "/")
		data, _ := os.ReadFile(path)
		writeTestFile(t, path, strings.Replace(string(data), `href="#target"`, `href="`+href+`"`, 1))
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
	if export, err := LoadOwnerAcceptanceExport(root); err != nil || len(export.Cells) != 7 {
		t.Fatalf("export cells=%d err=%v", len(export.Cells), err)
	}
}

func TestSiteCheck_OwnerAcceptanceExportRejectsSchemaAndFidelityMatrix(t *testing.T) {
	for _, oldNew := range [][2]string{{"public-site-acceptance/v1", "wrong/v1"}, {"version: 0.1.2", "version: 9.9.9"}, {"target_relative_path: index.html", "target_relative_path: other.html"}} {
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
    git_ref: v0.1.2
    name: backstop-ai/backstop-design-system
    source_coordinate: backstop-ai/backstop-design-system
    source_type: git
    version: 0.1.2
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
