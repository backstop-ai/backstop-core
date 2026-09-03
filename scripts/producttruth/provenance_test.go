package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func assertDescriptorContract(t *testing.T, id, kind, binding string, count int) {
	t.Helper()
	item := renderedByID(t, id)
	text := string(item.Bytes)
	if strings.Count(text, "data-generated-source-descriptor ") != count {
		t.Fatalf("descriptor count for %s", id)
	}
	if !strings.Contains(text, "data-source-kind=\""+kind+"\"") || !strings.Contains(text, "data-commit-binding=\""+binding+"\"") {
		t.Fatal("descriptor type mismatch")
	}
	if strings.Count(text, "digest=sha256:") != 2 {
		t.Fatal("digest markers missing")
	}
}

func assertDescriptorMutationDetected(t *testing.T, id, needle string) {
	t.Helper()
	item := renderedByID(t, id)
	mutated := bytes.Replace(item.Bytes, []byte(needle), nil, 1)
	if bytes.Equal(mutated, item.Bytes) {
		t.Fatalf("needle %q absent", needle)
	}
	if strings.Count(string(mutated), "data-generated-source-descriptor ") >= strings.Count(string(item.Bytes), "data-generated-source-descriptor ") {
		t.Fatal("mutation did not remove descriptor evidence")
	}
}

func TestProductTruth_CLIImmutableSourceLinkPasses(t *testing.T) {
	assertDescriptorContract(t, "cli-command-catalog", "tree", "site", 1)
	if !strings.Contains(string(renderedByID(t, "cli-command-catalog").Bytes), "/tree/&lt;SITE-COMMIT&gt;/cmd/backstop") {
		t.Fatal("immutable tree URL missing")
	}
}

func TestProductTruth_CLIImmutableSourceLinkRemovalFails(t *testing.T) {
	assertDescriptorMutationDetected(t, "cli-command-catalog", "data-generated-source-descriptor ")
	if strings.Count(string(renderedByID(t, "cli-command-catalog").Bytes), "data-source-path=\"cmd/backstop\"") != 1 {
		t.Fatal("source path cardinality")
	}
}

func TestProductTruth_CLIImmutableSourceLinkDriftFails(t *testing.T) {
	item := renderedByID(t, "cli-command-catalog")
	if !bytes.Contains(item.Bytes, []byte("owner=/reference/#cli-command-catalog")) {
		t.Fatal("owner missing")
	}
	if bytes.Contains(item.Bytes, []byte("/blob/&lt;SITE-COMMIT&gt;/cmd/backstop")) {
		t.Fatal("tree rendered as blob")
	}
}

func TestProductTruth_ArtifactSchemaImmutableSourceLinksPass(t *testing.T) {
	records, err := loadSchemas(testRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	assertDescriptorContract(t, "artifact-schema-catalog", "blob", "site", len(records))
	if len(records) < 2 {
		t.Fatal("schema corpus unexpectedly small")
	}
}

func TestProductTruth_ArtifactSchemaImmutableSourceLinkRemovalFails(t *testing.T) {
	assertDescriptorMutationDetected(t, "artifact-schema-catalog", "data-generated-source-descriptor ")
	if !strings.Contains(string(renderedByID(t, "artifact-schema-catalog").Bytes), "artifacts/base/schema.json") {
		t.Fatal("base descriptor absent")
	}
}

func TestProductTruth_ArtifactSchemaImmutableSourceLinkDriftFails(t *testing.T) {
	item := renderedByID(t, "artifact-schema-catalog")
	if strings.Contains(string(item.Bytes), "data-source-commit=") {
		t.Fatal("blob carries commit member")
	}
	if !strings.Contains(string(item.Bytes), "owner=/reference/#artifact-schema-catalog") {
		t.Fatal("owner missing")
	}
}

func TestProductTruth_PublishedPackImmutableSourceLinksPass(t *testing.T) {
	assertDescriptorContract(t, "published-pack-catalog", "blob", "site", 1)
	text := string(renderedByID(t, "published-pack-catalog").Bytes)
	if !strings.Contains(text, "data-source-path=\"docs/_data/published-pack-inventory.yml\"") {
		t.Fatal("inventory descriptor missing")
	}
}

func TestProductTruth_PublishedPackImmutableSourceLinkRemovalFails(t *testing.T) {
	item := renderedByID(t, "published-pack-catalog")
	withoutInventory := bytes.Replace(item.Bytes, []byte("data-source-path=\"docs/_data/published-pack-inventory.yml\""), nil, 1)
	if bytes.Equal(withoutInventory, item.Bytes) {
		t.Fatal("descriptor mutation did not apply")
	}
}

func TestProductTruth_PublishedPackImmutableSourceLinkDriftFails(t *testing.T) {
	item := renderedByID(t, "published-pack-catalog")
	if strings.Contains(string(item.Bytes), "data-source-kind=\"tree\"") {
		t.Fatal("pack source rendered as tree")
	}
	if strings.Contains(string(item.Bytes), "data-source-path=\"backstop.yml\"") || strings.Contains(string(item.Bytes), "data-source-path=\"backstop.lock\"") {
		t.Fatal("catalog still cites this repository's lock")
	}
	if !strings.Contains(string(item.Bytes), "owner=/pack/examples/#published-pack-catalog") {
		t.Fatal("owner missing")
	}
}

func TestProductTruth_ReleaseHistoryImmutableSourceLinksPass(t *testing.T) {
	records, err := loadReleases(testRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	assertDescriptorContract(t, "release-history", "commit", "record", len(records))
	if len(records) == 0 {
		t.Fatal("release history empty")
	}
}

func TestProductTruth_ReleaseHistoryImmutableSourceLinkRemovalFails(t *testing.T) {
	assertDescriptorMutationDetected(t, "release-history", "data-generated-source-descriptor ")
	if !strings.Contains(string(renderedByID(t, "release-history").Bytes), "data-source-commit=") {
		t.Fatal("commit descriptor absent")
	}
}

func TestProductTruth_ReleaseHistoryImmutableSourceLinkDriftFails(t *testing.T) {
	item := renderedByID(t, "release-history")
	if strings.Contains(string(item.Bytes), "data-source-path=") {
		t.Fatal("commit carries path member")
	}
	if !strings.Contains(string(item.Bytes), "owner=/status/#release-history") {
		t.Fatal("owner missing")
	}
}

func TestProductTruth_SourceLinkCanonicalJSONAbsentEmptyNullMutationMatrix(t *testing.T) {
	tree, err := json.Marshal(SourceLinkDescriptor{Kind: "tree", CommitBinding: "site", Path: "cmd/backstop"})
	if err != nil {
		t.Fatal(err)
	}
	commit, err := json.Marshal(SourceLinkDescriptor{Kind: "commit", CommitBinding: "record", Commit: strings.Repeat("a", 40)})
	if err != nil {
		t.Fatal(err)
	}
	if string(tree) != `{"kind":"tree","commit_binding":"site","path":"cmd/backstop"}` {
		t.Fatalf("tree=%s", tree)
	}
	if string(commit) != `{"kind":"commit","commit_binding":"record","commit":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}` {
		t.Fatalf("commit=%s", commit)
	}
}
