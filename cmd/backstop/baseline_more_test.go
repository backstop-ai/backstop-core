package main

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRunBaselineGenerate_ConfigErrorWithoutBackstopYML verifies that
// runBaselineGenerate fails fast with a config-stage error when no backstop.yml
// is discoverable from the working directory.
func TestRunBaselineGenerate_ConfigErrorWithoutBackstopYML(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("BACKSTOP_CONFIG", filepath.Join(dir, "does-not-exist.yml"))
	defer chdirTemp(t, dir)()

	err := runBaselineGenerate(nil, nil)
	if err == nil {
		t.Fatal("expected config error from runBaselineGenerate without backstop.yml")
	}
	if !strings.Contains(err.Error(), "config") {
		t.Errorf("error = %q, want it to mention the config stage", err.Error())
	}
}

// TestResolveRepositoryFromOrigin_MalformedURL verifies that an origin remote
// whose URL is not a GitHub URL yields an "unable to resolve repository" error.
func TestResolveRepositoryFromOrigin_MalformedURL(t *testing.T) {
	projectRoot := t.TempDir()
	gitCmd(t, projectRoot, "init")
	gitCmd(t, projectRoot, "remote", "add", "origin", "https://example.com/not-github.git")

	_, err := resolveRepositoryFromOrigin(projectRoot)
	if err == nil {
		t.Fatal("expected error for a non-GitHub origin URL")
	}
	if !strings.Contains(err.Error(), "unable to resolve repository") {
		t.Errorf("error = %q, want it to mention unable to resolve repository", err.Error())
	}
}

// TestResolveRepositoryFromOrigin_HTTPSGitHubURL verifies the regex also
// resolves an https GitHub remote (not just the ssh form) to owner/repo.
func TestResolveRepositoryFromOrigin_HTTPSGitHubURL(t *testing.T) {
	projectRoot := t.TempDir()
	gitCmd(t, projectRoot, "init")
	gitCmd(t, projectRoot, "remote", "add", "origin", "https://github.com/acme/widgets.git")

	repo, err := resolveRepositoryFromOrigin(projectRoot)
	if err != nil {
		t.Fatalf("resolveRepositoryFromOrigin: %v", err)
	}
	if repo != "acme/widgets" {
		t.Errorf("repo = %q, want acme/widgets", repo)
	}
}

// TestDownloadBaselineArtifact_InvalidZipPayload verifies that a non-zip
// payload from gh yields an "invalid zip payload" error.
func TestDownloadBaselineArtifact_InvalidZipPayload(t *testing.T) {
	binDir := t.TempDir()
	writeFakeGh(t, filepath.Join(binDir, "gh"), "#!/bin/sh\nprintf 'not a zip'\n")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	_, err := downloadBaselineArtifact(t.TempDir(), "owner/repo", 1)
	if err == nil || !strings.Contains(err.Error(), "invalid zip payload") {
		t.Fatalf("expected invalid zip payload error, got %v", err)
	}
}

// TestDownloadBaselineArtifact_ExtractsBaselineEntry verifies the happy path:
// a zip containing a baseline.json entry is opened and its body returned.
func TestDownloadBaselineArtifact_ExtractsBaselineEntry(t *testing.T) {
	zipPath := filepath.Join(t.TempDir(), "artifact.zip")
	var buf bytes.Buffer
	writer := zip.NewWriter(&buf)
	entry, err := writer.Create("nested/baseline.json")
	if err != nil {
		t.Fatal(err)
	}
	payload := `{"schema_version":"baseline/v1"}`
	if _, err := entry.Write([]byte(payload)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(zipPath, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}

	binDir := t.TempDir()
	writeFakeGh(t, filepath.Join(binDir, "gh"), "#!/bin/sh\ncat \""+zipPath+"\"\n")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	body, err := downloadBaselineArtifact(t.TempDir(), "owner/repo", 99)
	if err != nil {
		t.Fatalf("downloadBaselineArtifact: %v", err)
	}
	if string(body) != payload {
		t.Errorf("body = %q, want %q", string(body), payload)
	}
}

// TestDownloadBaselineArtifact_GhFailurePropagates verifies that a gh API
// failure surfaces as an "artifact download failed" error.
func TestDownloadBaselineArtifact_GhFailurePropagates(t *testing.T) {
	binDir := t.TempDir()
	writeFakeGh(t, filepath.Join(binDir, "gh"), "#!/bin/sh\necho boom >&2\nexit 3\n")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	_, err := downloadBaselineArtifact(t.TempDir(), "owner/repo", 1)
	if err == nil || !strings.Contains(err.Error(), "artifact download failed") {
		t.Fatalf("expected artifact download failed error, got %v", err)
	}
}

// chdirTemp changes the working directory to dir and returns a restore func.
// Relocated from the deleted code_check_test.go (ISSUE-018); baseline_more_test
// is now its sole user.
func chdirTemp(t *testing.T, dir string) func() {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir %s: %v", dir, err)
	}
	return func() {
		if err := os.Chdir(wd); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	}
}
