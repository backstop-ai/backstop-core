package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func repositoryFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", path))
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

func releaseHistoryJob(t *testing.T) string {
	t.Helper()
	release := repositoryFile(t, filepath.Join(".github", "workflows", "release.yml"))
	match := regexp.MustCompile(`(?ms)^  release-history-current:\n(.*?)(?:^  [A-Za-z0-9_-]+:\n|\z)`).FindString(release)
	if match == "" {
		t.Fatal("release-history-current job is absent")
	}
	return match
}

func TestProductTruth_ReleaseBlocksStaleLatestMain(t *testing.T) {
	job := releaseHistoryJob(t)
	for _, required := range []string{"generate-product-truth.sh --check", "origin/main"} {
		if !strings.Contains(job, required) {
			t.Errorf("release-history-current job is missing %q", required)
		}
	}
}

func TestProductTruth_ReleaseHandshakePassesAfterMainRegeneration(t *testing.T) {
	release := repositoryFile(t, filepath.Join(".github", "workflows", "release.yml"))
	for _, required := range []string{"fetch-depth: 0", "github.ref_name", "release-history-current"} {
		if !strings.Contains(release, required) {
			t.Errorf("release workflow is missing %q", required)
		}
	}
}

func TestProductTruth_ReleaseWorkflowRejectsTagCheckoutSubstitution(t *testing.T) {
	job := releaseHistoryJob(t)
	for _, required := range []string{"ref: main", "git fetch", "refs/tags"} {
		if !strings.Contains(job, required) {
			t.Errorf("release-history-current job is missing %q", required)
		}
	}
	if regexp.MustCompile(`ref:[^\n]*github\.ref`).MatchString(job) {
		t.Error("release-history-current job substitutes the triggering tag checkout for main")
	}
}

func TestProductTruth_VerifierAcceptsCoverageAtThreshold(t *testing.T) {
	verifier := repositoryFile(t, filepath.Join("scripts", "verify-product-truth.sh"))
	for _, required := range []string{"80.00", "PT205_COVERAGE", "cover -func"} {
		if !strings.Contains(verifier, required) {
			t.Errorf("product-truth verifier is missing %q", required)
		}
	}
}

func TestProductTruth_VerifierRejectsCoverageFailureMatrix(t *testing.T) {
	verifier := repositoryFile(t, filepath.Join("scripts", "verify-product-truth.sh"))
	for _, required := range []string{"total_count", "numeric_total", "coverage below 80.00"} {
		if !strings.Contains(verifier, required) {
			t.Errorf("product-truth verifier is missing %q", required)
		}
	}
}

func TestProductTruth_VerifierCoversIndependentSourcePipeline(t *testing.T) {
	verifier := repositoryFile(t, filepath.Join("scripts", "verify-product-truth.sh"))
	for _, required := range []string{"go test ./scripts/producttruth/...", "generate-product-truth.sh --check", "SourceIncludes"} {
		if !strings.Contains(verifier, required) {
			t.Errorf("product-truth verifier is missing %q", required)
		}
	}
}
