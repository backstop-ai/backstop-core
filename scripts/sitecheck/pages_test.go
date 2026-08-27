package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func repositoryRoot() string {
	return filepath.Clean(filepath.Join("..", ".."))
}

func copyPagesContract(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, relative := range []string{".github/pages-actions.lock.yml", ".github/workflows/pages.yml"} {
		data, err := os.ReadFile(filepath.Join(repositoryRoot(), relative))
		if err != nil {
			t.Fatal(err)
		}
		writeTestFile(t, filepath.Join(root, relative), string(data))
	}
	return root
}

func TestSiteCheck_PagesWorkflowPinnedContractPasses(t *testing.T) {
	if findings := VerifyPagesWorkflow(repositoryRoot()); len(findings) != 0 {
		t.Fatalf("Pages workflow rejected: %#v", findings)
	}
}

func TestSiteCheck_PagesWorkflowRejectsWorkflowAndActionPinMatrix(t *testing.T) {
	mutations := []struct{ file, old, replacement string }{
		{".github/pages-actions.lock.yml", "3d3c42e5aac5ba805825da76410c181273ba90b1", "v7"},
		{".github/workflows/pages.yml", "include-hidden-files: true", "include-hidden-files: false"},
		{".github/workflows/pages.yml", "branches: [main]", "tags: ['v*']"},
		{".github/workflows/pages.yml", "cancel-in-progress: false", "cancel-in-progress: true"},
		{".github/workflows/pages.yml", "actions/deploy-pages@", "third-party/deploy@"},
		{".github/workflows/pages.yml", "ruby-version: \"3.3.4\"", "ruby-version: \"3.3\""},
		{".github/pages-actions.lock.yml", "45bfe0192ca1faeb007ade9deae92b16b8254a0d", "983d7736d9b0ae728b81ab479565c72886d7745b"},
	}
	for _, mutation := range mutations {
		root := copyPagesContract(t)
		path := filepath.Join(root, mutation.file)
		data, _ := os.ReadFile(path)
		changed := strings.Replace(string(data), mutation.old, mutation.replacement, 1)
		if changed == string(data) {
			t.Fatalf("mutation source absent: %s", mutation.old)
		}
		writeTestFile(t, path, changed)
		if findings := VerifyPagesWorkflow(root); len(findings) == 0 {
			t.Fatalf("workflow mutation %q unexpectedly passed", mutation.old)
		}
	}
}

func TestSiteCheck_RejectsParallelTruthOrPublication(t *testing.T) {
	workflow, err := os.ReadFile(filepath.Join(repositoryRoot(), ".github/workflows/pages.yml"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(workflow)
	for _, forbidden := range []string{"peaceiris/actions-gh-pages", "jekyll-action", "docs/index.html --destination", "git push gh-pages"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("parallel publication surface %q", forbidden)
		}
	}
}

func TestSiteCheck_RejectsPublishedVerificationRuntime(t *testing.T) {
	err := filepath.WalkDir(filepath.Join(repositoryRoot(), "docs"), func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		relative, _ := filepath.Rel(filepath.Join(repositoryRoot(), "docs"), path)
		if filepath.Ext(path) == ".js" || relative == "package.json" || relative == "package-lock.json" || strings.HasPrefix(relative, "tests"+string(filepath.Separator)) {
			t.Errorf("verification runtime is inside the publish source: %s", relative)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestSiteCheck_RejectsUnearnedRuntimeMatrix(t *testing.T) {
	for _, relative := range []string{"docs/_config.yml", "docs/_layouts/default.html", "package.json"} {
		data, err := os.ReadFile(filepath.Join(repositoryRoot(), relative))
		if err != nil {
			t.Fatal(err)
		}
		lower := strings.ToLower(string(data))
		for _, forbidden := range []string{"express", "next.js", "react", "database", "localstorage", "serviceworker", "<script"} {
			if strings.Contains(lower, forbidden) {
				t.Fatalf("%s contains prohibited runtime class %q", relative, forbidden)
			}
		}
	}
}

func TestSiteCheck_StaticLockedJekyllBuildPasses(t *testing.T) {
	lock, err := os.ReadFile(filepath.Join(repositoryRoot(), "Gemfile.lock"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(lock), "github-pages (232)") || !strings.Contains(string(lock), "jekyll (3.10.0)") {
		t.Fatal("locked GitHub Pages/Jekyll graph is absent")
	}
	workflow, _ := os.ReadFile(filepath.Join(repositoryRoot(), ".github/workflows/pages.yml"))
	if !strings.Contains(string(workflow), "./scripts/verify-public-site.sh") {
		t.Fatal("Pages does not use the canonical verifier")
	}
}
