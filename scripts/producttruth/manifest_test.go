package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func testRoot(t *testing.T) string {
	t.Helper()
	root, err := repositoryRoot()
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func testManifest(t *testing.T) Manifest {
	t.Helper()
	root := testRoot(t)
	manifest, err := LoadManifest(root, filepath.Join(root, manifestPath))
	if err != nil {
		t.Fatal(err)
	}
	return manifest
}

func cloneManifest(t *testing.T, mutate func(*Manifest)) (string, string) {
	t.Helper()
	root := t.TempDir()
	manifest := testManifest(t)
	mutate(&manifest)
	data, err := yamlMarshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "manifest.yml")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return root, path
}

func yamlMarshal(value any) ([]byte, error) { return yaml.Marshal(value) }

func assertManifestJob(t *testing.T, index int, id, output, owner string) {
	t.Helper()
	manifest := testManifest(t)
	job := manifest.Jobs[index]
	if job.ID != id || job.Output != output || job.OwnerRoute+"#"+job.OwnerAnchor != owner {
		t.Fatalf("unexpected job: %#v", job)
	}
	if job.Command != regenerate || job.Marker != "GENERATED PRODUCT TRUTH" || job.SourceLinkPolicy == "" {
		t.Fatalf("incomplete job: %#v", job)
	}
}

func TestProductTruth_ManifestCLIJobPasses(t *testing.T) {
	assertManifestJob(t, 0, "cli-command-catalog", "docs/_includes/generated/cli-command-catalog.md", "/reference/#cli-command-catalog")
	if got := testManifest(t).Jobs[0].Inputs; !reflect.DeepEqual(got, []string{"cmd/backstop"}) {
		t.Fatalf("inputs=%v", got)
	}
}

func TestProductTruth_ManifestArtifactSchemaJobPasses(t *testing.T) {
	assertManifestJob(t, 1, "artifact-schema-catalog", "docs/_includes/generated/artifact-schema-catalog.md", "/reference/#artifact-schema-catalog")
	if len(testManifest(t).Jobs[1].Inputs) != 2 {
		t.Fatal("schema inputs must be closed")
	}
}

func TestProductTruth_ManifestPublishedPackJobPasses(t *testing.T) {
	assertManifestJob(t, 2, "published-pack-catalog", "docs/_includes/generated/published-pack-catalog.md", "/pack/examples/#published-pack-catalog")
	if got := testManifest(t).Jobs[2].Inputs; !reflect.DeepEqual(got, []string{"docs/_data/published-pack-inventory.yml"}) {
		t.Fatalf("inputs=%v", got)
	}
}

func TestProductTruth_ManifestReleaseHistoryJobPasses(t *testing.T) {
	assertManifestJob(t, 3, "release-history", "docs/_includes/generated/release-history.md", "/status/#release-history")
	if got := testManifest(t).Jobs[3].Inputs; !reflect.DeepEqual(got, []string{"refs/tags/vMAJOR.MINOR.PATCH"}) {
		t.Fatalf("inputs=%v", got)
	}
}

func TestProductTruth_ManifestRejectsInvalidJobMatrix(t *testing.T) {
	root, path := cloneManifest(t, func(m *Manifest) { m.Jobs[1].Output = m.Jobs[0].Output })
	_, err := LoadManifest(root, path)
	if err == nil || !strings.Contains(err.Error(), "PT001_MANIFEST") {
		t.Fatalf("err=%v", err)
	}
}

func TestProductTruth_DefaultModeEqualsWrite(t *testing.T) {
	implicit, err := parseMode(nil)
	if err != nil {
		t.Fatal(err)
	}
	explicit, err := parseMode([]string{"--write"})
	if err != nil {
		t.Fatal(err)
	}
	if implicit != explicit || explicit != "write" {
		t.Fatalf("%q != %q", implicit, explicit)
	}
	if err := run(nil); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"--check"}); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"--recover"}); err != nil {
		t.Fatal(err)
	}
}

func TestProductTruth_RejectsInvalidModeArguments(t *testing.T) {
	for _, args := range [][]string{{"--wat"}, {"--write", "--check"}, {"--recover", "extra"}} {
		if _, err := parseMode(args); err == nil || !strings.Contains(err.Error(), "PT001_MANIFEST") {
			t.Fatalf("args=%v err=%v", args, err)
		}
	}
	if err := run([]string{"--wat"}); err == nil || !strings.Contains(err.Error(), "PT001_MANIFEST") {
		t.Fatalf("run err=%v", err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"--check"}); err == nil || !strings.Contains(err.Error(), "PT001_MANIFEST") {
		t.Fatalf("outside repository err=%v", err)
	}
	if err := os.Chdir(cwd); err != nil {
		t.Fatal(err)
	}
}

func TestProductTruth_GitLogicalTagStorageAndFormMatrix(t *testing.T) {
	if !stableTagPattern.MatchString("v1.2.3") || stableTagPattern.MatchString("v1.2.3-rc.1") {
		t.Fatal("stable tag filter")
	}
	if !semverGreater("v2.0.0", "v1.99.99") || semverGreater("v1.2.3", "v1.2.3") {
		t.Fatal("semantic order")
	}
	if strings.Contains(string(stableTagPattern.String()), ".git/refs") {
		t.Fatal("physical ref storage entered parser")
	}
}

func TestProductTruth_GitLogicalRefsRejectInvalidMatrix(t *testing.T) {
	invalid := []string{"1.2.3", "v01.2.3", "v1.02.3", "v1.2", "v1.2.3+meta", "backstop/spec/074"}
	for _, tag := range invalid {
		if stableTagPattern.MatchString(tag) {
			t.Fatalf("accepted %s", tag)
		}
	}
	if !stableTagPattern.MatchString("v0.0.0") {
		t.Fatal("valid zero tag rejected")
	}
}
