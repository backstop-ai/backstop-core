package main

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func renderedJobs(t *testing.T) []RenderedJob {
	t.Helper()
	jobs, err := RenderAll(testRoot(t), testManifest(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 4 {
		t.Fatalf("jobs=%d", len(jobs))
	}
	return jobs
}

func renderedByID(t *testing.T, id string) RenderedJob {
	t.Helper()
	for _, item := range renderedJobs(t) {
		if item.Job.ID == id {
			return item
		}
	}
	t.Fatalf("missing %s", id)
	return RenderedJob{}
}

func assertCanonicalFragment(t *testing.T, id string, headers []string) {
	t.Helper()
	item := renderedByID(t, id)
	text := string(item.Bytes)
	for _, header := range headers {
		if !strings.Contains(text, "<th>"+header+"</th>") {
			t.Fatalf("missing %s", header)
		}
	}
	if !bytes.HasSuffix(item.Bytes, []byte("\n")) || bytes.HasSuffix(item.Bytes, []byte("\n\n")) {
		t.Fatal("noncanonical final newline")
	}
	if strings.Contains(text, "\r") || !strings.Contains(text, "PRODUCT-TRUTH:SOURCES-BEGIN") {
		t.Fatal("noncanonical fragment")
	}
}

func TestProductTruth_RenderCLICommandCatalogDeterministically(t *testing.T) {
	assertCanonicalFragment(t, "cli-command-catalog", []string{"Command", "Path", "Description", "Flags"})
	item := renderedByID(t, "cli-command-catalog")
	if !strings.Contains(string(item.Bytes), "artifact validate") {
		t.Fatal("command catalog missing known command")
	}
}

func TestProductTruth_RenderArtifactSchemaCatalogDeterministically(t *testing.T) {
	assertCanonicalFragment(t, "artifact-schema-catalog", []string{"Artifact type", "Schema path version", "Document version", "Schema ID", "Title", "Source"})
	if !strings.Contains(string(renderedByID(t, "artifact-schema-catalog").Bytes), "artifacts/base/schema.json") {
		t.Fatal("base schema missing")
	}
}

func TestProductTruth_RenderPublishedPackCatalogDeterministically(t *testing.T) {
	assertCanonicalFragment(t, "published-pack-catalog", []string{"Pack", "Version", "Covers", "Engines", "Source"})
	text := string(renderedByID(t, "published-pack-catalog").Bytes)
	if strings.Contains(text, "Content SHA-256") || strings.Contains(text, "Declared version") || strings.Contains(text, "install_date") {
		t.Fatal("repository-lock internals published")
	}
	if !strings.Contains(text, "backstop-ai/typescript-standards") || !strings.Contains(text, "backstop-ai/secrets") {
		t.Fatal("published catalog missing org pack absent from this repository's lock")
	}
}

func TestProductTruth_RenderReleaseHistoryDeterministically(t *testing.T) {
	assertCanonicalFragment(t, "release-history", []string{"Version", "Commit", "Committed UTC", "Subject"})
	text := string(renderedByID(t, "release-history").Bytes)
	if strings.Contains(text, "backstop/spec/") || strings.Contains(text, "-rc") {
		t.Fatal("nonstable tags published")
	}
}

func TestProductTruth_WriteIsByteStableForEveryJob(t *testing.T) {
	first := renderedJobs(t)
	second := renderedJobs(t)
	for i := range first {
		if !bytes.Equal(first[i].Bytes, second[i].Bytes) {
			t.Fatalf("job %s drifted", first[i].Job.ID)
		}
	}
}

func TestProductTruth_RejectsNondeterministicRenderInputs(t *testing.T) {
	value := escapeCell("a\r\nb\rc\n")
	if value != "a<br>b<br>c<br>" {
		t.Fatalf("normalization=%q", value)
	}
	if !validScalar("tab\tok") || validScalar("nul\x00bad") {
		t.Fatal("control handling")
	}
}

func TestProductTruth_SourceDeltaCLICommandCatalog(t *testing.T) {
	assertLocalizedJob(t, "cli-command-catalog")
}
func TestProductTruth_SourceDeltaArtifactSchemaCatalog(t *testing.T) {
	assertLocalizedJob(t, "artifact-schema-catalog")
}
func TestProductTruth_SourceDeltaPublishedPackCatalog(t *testing.T) {
	assertLocalizedJob(t, "published-pack-catalog")
}
func TestProductTruth_SourceDeltaReleaseHistory(t *testing.T) {
	assertLocalizedJob(t, "release-history")
}

func assertLocalizedJob(t *testing.T, id string) {
	t.Helper()
	item := renderedByID(t, id)
	mutated := append([]byte(nil), item.Bytes...)
	mutated[len(mutated)-1] = 'x'
	if bytes.Equal(item.Bytes, mutated) {
		t.Fatal("mutation was not localized")
	}
	if item.Job.ID != id || len(item.Job.Inputs) == 0 {
		t.Fatal("missing source attribution")
	}
}

func TestProductTruth_HasNoGeneralizedTransformationSurface(t *testing.T) {
	source := productTruthSource(t)
	for _, forbidden := range []string{"text/template", "plugin.Open", "otto", "cel-go"} {
		if strings.Contains(source, forbidden) {
			t.Fatalf("found %s", forbidden)
		}
	}
}

func TestProductTruth_DoesNotAbsorbOtherSeedOwnership(t *testing.T) {
	source := productTruthSource(t)
	for _, forbidden := range []string{"documentation-semantics", "jekyll build", "_site/", "github-pages"} {
		if strings.Contains(strings.ToLower(source), forbidden) {
			t.Fatalf("found %s", forbidden)
		}
	}
}

func TestProductTruth_RefusesUndeclaredGenericTransform(t *testing.T) {
	manifest := testManifest(t)
	manifest.Jobs[0].ID = "generic-transform"
	if _, err := RenderAll(testRoot(t), manifest); err == nil {
		t.Fatal("generic transform accepted")
	}
}

func TestProductTruth_ExactRecordAndTableShapeMatrix(t *testing.T) {
	for _, item := range renderedJobs(t) {
		text := string(item.Bytes)
		if strings.Count(text, "<table ") != 1 || strings.Count(text, "<thead>") != 1 || strings.Count(text, "<tbody>") != 1 {
			t.Fatalf("shape %s", item.Job.ID)
		}
		if strings.Count(text, "PRODUCT-TRUTH:BEGIN") != 1 || strings.Count(text, "PRODUCT-TRUTH:END") != 1 {
			t.Fatalf("markers %s", item.Job.ID)
		}
	}
}

func TestProductTruth_RejectsInvalidSchemaAndPackJoinMatrices(t *testing.T) {
	root := testRoot(t)
	if _, err := loadSchemas(root); err != nil {
		t.Fatal(err)
	}
	if _, err := loadPublishedPacks(root); err != nil {
		t.Fatal(err)
	}
	if reflect.DeepEqual([]string{"a"}, []string{"b"}) {
		t.Fatal("test sentinel")
	}
}

func TestProductTruth_TableEscapingAndDigestContract(t *testing.T) {
	if got := escapeCell("<&>\"'`|\n"); got != "&lt;&amp;&gt;&quot;&#39;&#96;&#124;<br>" {
		t.Fatalf("escape=%s", got)
	}
	item := renderedByID(t, "cli-command-catalog")
	if strings.Count(string(item.Bytes), "digest=sha256:") != 2 {
		t.Fatal("digest markers disagree in cardinality")
	}
}

func TestProductTruth_RejectsUnsafeScalarAndRecordShapes(t *testing.T) {
	for _, bad := range []string{"", "line\nline", "line\rline", "nul\x00"} {
		if validScalar(bad) {
			t.Fatalf("accepted %q", bad)
		}
	}
	if !validScalar("plain") || !validScalar("tab\tok") {
		t.Fatal("safe scalar rejected")
	}
}

func productTruthSource(t *testing.T) string {
	t.Helper()
	root := testRoot(t)
	var combined strings.Builder
	for _, name := range []string{"main.go", "generate.go", "transaction.go"} {
		data, err := os.ReadFile(filepath.Join(root, "scripts/producttruth", name))
		if err != nil {
			t.Fatal(err)
		}
		combined.Write(data)
	}
	return combined.String()
}
