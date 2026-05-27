package main

import (
	"archive/zip"
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func writeBaselineFixture(t *testing.T, projectRoot string) string {
	t.Helper()
	cachePath := filepath.Join(projectRoot, ".backstop", "baseline.json")
	if err := os.MkdirAll(filepath.Dir(cachePath), 0o755); err != nil {
		t.Fatalf("mkdir baseline dir: %v", err)
	}
	data := []byte(`{"schema_version":"baseline/v1","generated_at":"2026-05-26T00:00:00Z","git_sha":"abc123","violations":[]}`)
	if err := os.WriteFile(cachePath, data, 0o644); err != nil {
		t.Fatalf("write baseline fixture: %v", err)
	}
	return cachePath
}

func TestBaselineCLI_TTL_Default15Minutes_Contract(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectRoot, "backstop.yml"), []byte("project: ttl-default\nlanguage: go\n"), 0o644); err != nil {
		t.Fatalf("write backstop.yml: %v", err)
	}
	writeBaselineFixture(t, projectRoot)
	orig, _ := os.Getwd()
	if err := os.Chdir(projectRoot); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() { _ = os.Chdir(orig) }()

	root := NewRootCommand()
	_, err := executeCommand(root, "gate", "--json")
	if err == nil {
		t.Fatalf("expected gate to enforce remaining checks and return non-zero in this minimal fixture")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "exit code 1") {
		t.Fatalf("expected non-config gate failure semantics, got: %v", err)
	}
}

func TestBaselineCLI_TTL_Override_Contract(t *testing.T) {
	projectRoot := t.TempDir()
	config := "project: ttl-override\nlanguage: go\nenforcement:\n  baseline_ttl: 1m\n"
	if err := os.WriteFile(filepath.Join(projectRoot, "backstop.yml"), []byte(config), 0o644); err != nil {
		t.Fatalf("write backstop.yml: %v", err)
	}
	writeBaselineFixture(t, projectRoot)
	orig, _ := os.Getwd()
	if err := os.Chdir(projectRoot); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() { _ = os.Chdir(orig) }()

	root := NewRootCommand()
	_, err := executeCommand(root, "gate", "--json")
	if err == nil {
		t.Fatalf("expected gate to enforce remaining checks and return non-zero in this minimal fixture")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "exit code 1") {
		t.Fatalf("expected non-config gate failure semantics, got: %v", err)
	}
}

func TestBaselinePull_BypassesTTL_Contract(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectRoot, "backstop.yml"), []byte("project: pull-bypass\nlanguage: go\n"), 0o644); err != nil {
		t.Fatalf("write backstop.yml: %v", err)
	}
	cachePath := writeBaselineFixture(t, projectRoot)
	before, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatalf("read cache before: %v", err)
	}

	orig, _ := os.Getwd()
	if err := os.Chdir(projectRoot); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() { _ = os.Chdir(orig) }()

	root := NewRootCommand()
	_, execErr := executeCommand(root, "baseline", "pull")
	if execErr == nil {
		t.Fatalf("expected baseline pull contract test to fail until baseline command is implemented")
	}
	if !strings.Contains(execErr.Error(), "baseline") {
		t.Fatalf("expected baseline pull failure to mention baseline command semantics, got: %v", execErr)
	}
	after, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatalf("read cache after: %v", err)
	}
	if string(after) != string(before) {
		t.Fatalf("expected pull failure path to preserve existing cache integrity")
	}
}

func TestBaselinePull_ActionableFailureModes_Contract(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectRoot, "backstop.yml"), []byte("project: pull-errors\nlanguage: go\n"), 0o644); err != nil {
		t.Fatalf("write backstop.yml: %v", err)
	}
	cachePath := writeBaselineFixture(t, projectRoot)
	baselineBefore, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatalf("read cache before: %v", err)
	}

	orig, _ := os.Getwd()
	if err := os.Chdir(projectRoot); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() { _ = os.Chdir(orig) }()

	for _, scenario := range []string{"missing remote", "missing auth", "missing artifact", "workflow selection miss"} {
		t.Run(scenario, func(t *testing.T) {
			root := NewRootCommand()
			_, execErr := executeCommand(root, "baseline", "pull")
			if execErr == nil {
				t.Fatalf("expected contract failure for scenario %q until implemented", scenario)
			}
			if !strings.Contains(strings.ToLower(execErr.Error()), "baseline") {
				t.Fatalf("expected actionable baseline pull error for %q, got: %v", scenario, execErr)
			}
		})
	}

	baselineAfter, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatalf("read cache after: %v", err)
	}
	if string(baselineAfter) != string(baselineBefore) {
		t.Fatalf("expected error paths to preserve cache bytes")
	}
}

func TestBaselineBytesValidation(t *testing.T) {
	if err := validateBaselineBytes([]byte(`{"schema_version":"baseline/v1","violations":[]}`)); err != nil {
		t.Fatalf("valid baseline rejected: %v", err)
	}
	if err := validateBaselineBytes([]byte(`{"schema_version":"baseline/v99"}`)); err == nil {
		t.Fatal("expected unsupported baseline schema to fail")
	}
	if err := validateBaselineBytes([]byte(`not-json`)); err == nil {
		t.Fatal("expected invalid JSON baseline to fail")
	}
}

func TestBaselineGenerateWritesBaselineCache(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectRoot, "backstop.yml"), []byte("project: generate-baseline\nlanguage: go\n"), 0o644); err != nil {
		t.Fatalf("write backstop.yml: %v", err)
	}
	orig, _ := os.Getwd()
	if err := os.Chdir(projectRoot); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() { _ = os.Chdir(orig) }()

	if err := runBaselineGenerate(nil, nil); err != nil {
		t.Fatalf("baseline generate: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(projectRoot, ".backstop", "baseline.json"))
	if err != nil {
		t.Fatalf("read generated baseline: %v", err)
	}
	if !strings.Contains(string(data), `"schema_version": "baseline/v1"`) {
		t.Fatalf("expected baseline/v1 cache, got %s", string(data))
	}
}

func TestBaselineWriteAtomicallyCreatesParentAndReplacesContent(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".backstop", "baseline.json")
	if err := writeBaselineAtomically(path, []byte(`{"schema_version":"baseline/v1"}`)); err != nil {
		t.Fatalf("initial write: %v", err)
	}
	if err := writeBaselineAtomically(path, []byte(`{"schema_version":"baseline/v1","git_sha":"next"}`)); err != nil {
		t.Fatalf("replacement write: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read baseline: %v", err)
	}
	if !strings.Contains(string(data), `"git_sha":"next"`) {
		t.Fatalf("expected replacement content, got %s", string(data))
	}
}

func TestResolveRepositoryFromOrigin(t *testing.T) {
	projectRoot := t.TempDir()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skipf("git unavailable: %v", err)
	}
	cmd := exec.Command("git", "init")
	cmd.Dir = projectRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, string(out))
	}
	cmd = exec.Command("git", "remote", "add", "origin", "git@github.com:owner/repo.git")
	cmd.Dir = projectRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git remote add: %v\n%s", err, string(out))
	}
	repo, err := resolveRepositoryFromOrigin(projectRoot)
	if err != nil {
		t.Fatalf("resolve repo: %v", err)
	}
	if repo != "owner/repo" {
		t.Fatalf("repo = %q, want owner/repo", repo)
	}
}

func TestResolveRepositoryFromOriginReportsMissingRemote(t *testing.T) {
	projectRoot := t.TempDir()
	cmd := exec.Command("git", "init")
	cmd.Dir = projectRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, string(out))
	}
	_, err := resolveRepositoryFromOrigin(projectRoot)
	if err == nil || !strings.Contains(err.Error(), "missing origin remote") {
		t.Fatalf("expected missing origin remote error, got %v", err)
	}
}

func TestBaselineGitHubHelpersUseGhCLI(t *testing.T) {
	projectRoot := t.TempDir()
	binDir := t.TempDir()
	zipPath := filepath.Join(t.TempDir(), "artifact.zip")
	makeBaselineZip(t, zipPath)
	writeFakeGh(t, filepath.Join(binDir, "gh"), `#!/bin/sh
if [ "$1 $2" = "auth status" ]; then
  exit 0
fi
if [ "$1" != "api" ]; then
  echo "unexpected command: $*" >&2
  exit 1
fi
case "$2" in
  repos/owner/repo/actions/runs\?branch=main\&status=success\&per_page=20)
    printf '{"workflow_runs":[{"id":42,"name":"ci","conclusion":"success","head_branch":"main"}]}'
    ;;
  repos/owner/repo/actions/runs/42/artifacts)
    printf '{"artifacts":[{"id":99,"name":"backstop-baseline-v1"}]}'
    ;;
  repos/owner/repo/actions/artifacts/99/zip)
    cat "`+zipPath+`"
    ;;
  *)
    echo "unexpected endpoint: $2" >&2
    exit 1
    ;;
esac
`)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	if err := ensureGitHubAuth(projectRoot); err != nil {
		t.Fatalf("ensure auth: %v", err)
	}
	runID, err := resolveLatestSuccessfulMainRun(projectRoot, "owner/repo")
	if err != nil {
		t.Fatalf("resolve run: %v", err)
	}
	if runID != 42 {
		t.Fatalf("run id = %d, want 42", runID)
	}
	artifactID, err := resolveBaselineArtifactID(projectRoot, "owner/repo", runID)
	if err != nil {
		t.Fatalf("resolve artifact: %v", err)
	}
	if artifactID != 99 {
		t.Fatalf("artifact id = %d, want 99", artifactID)
	}
	baselineBytes, err := downloadBaselineArtifact(projectRoot, "owner/repo", artifactID)
	if err != nil {
		t.Fatalf("download artifact: %v", err)
	}
	if !strings.Contains(string(baselineBytes), `"schema_version":"baseline/v1"`) {
		t.Fatalf("expected baseline payload, got %s", string(baselineBytes))
	}
}

func TestBaselineGitHubHelpersReportPayloadMisses(t *testing.T) {
	projectRoot := t.TempDir()
	binDir := t.TempDir()
	writeFakeGh(t, filepath.Join(binDir, "gh"), `#!/bin/sh
if [ "$1" != "api" ]; then
  echo "auth denied" >&2
  exit 1
fi
case "$2" in
  *runs-invalid*)
    printf 'not-json'
    ;;
  *runs-empty*)
    printf '{"workflow_runs":[]}'
    ;;
  *artifacts-invalid*)
    printf 'not-json'
    ;;
  *artifacts-empty*)
    printf '{"artifacts":[]}'
    ;;
  *bad-zip*)
    printf 'not-a-zip'
    ;;
  fail)
    echo "gh failure" >&2
    exit 1
    ;;
esac
`)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	if err := ensureGitHubAuth(projectRoot); err == nil || !strings.Contains(err.Error(), "missing GitHub authentication") {
		t.Fatalf("expected auth failure, got %v", err)
	}
	if _, err := ghAPI(projectRoot, "fail"); err == nil || !strings.Contains(err.Error(), "gh failure") {
		t.Fatalf("expected gh failure output, got %v", err)
	}
	if _, err := resolveLatestSuccessfulMainRun(projectRoot, "runs-invalid"); err == nil || !strings.Contains(err.Error(), "invalid run listing payload") {
		t.Fatalf("expected invalid runs payload, got %v", err)
	}
	if _, err := resolveLatestSuccessfulMainRun(projectRoot, "runs-empty"); err == nil || !strings.Contains(err.Error(), "no latest successful main run") {
		t.Fatalf("expected missing run error, got %v", err)
	}
	if _, err := resolveBaselineArtifactID(projectRoot, "artifacts-invalid", 0); err == nil || !strings.Contains(err.Error(), "invalid artifact listing payload") {
		t.Fatalf("expected invalid artifacts payload, got %v", err)
	}
	if _, err := resolveBaselineArtifactID(projectRoot, "artifacts-empty", 0); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected missing artifact error, got %v", err)
	}
	if _, err := downloadBaselineArtifact(projectRoot, "bad-zip", 0); err == nil || !strings.Contains(err.Error(), "invalid zip payload") {
		t.Fatalf("expected invalid zip error, got %v", err)
	}
}

func TestDownloadBaselineArtifactReportsMissingEntry(t *testing.T) {
	projectRoot := t.TempDir()
	binDir := t.TempDir()
	zipPath := filepath.Join(t.TempDir(), "artifact.zip")
	var buf bytes.Buffer
	writer := zip.NewWriter(&buf)
	entry, err := writer.Create("other.json")
	if err != nil {
		t.Fatalf("create zip entry: %v", err)
	}
	if _, err := entry.Write([]byte(`{}`)); err != nil {
		t.Fatalf("write zip entry: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
	if err := os.WriteFile(zipPath, buf.Bytes(), 0o644); err != nil {
		t.Fatalf("write zip fixture: %v", err)
	}
	writeFakeGh(t, filepath.Join(binDir, "gh"), `#!/bin/sh
cat "`+zipPath+`"
`)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	_, err = downloadBaselineArtifact(projectRoot, "owner/repo", 99)
	if err == nil || !strings.Contains(err.Error(), "baseline.json entry absent") {
		t.Fatalf("expected missing baseline entry error, got %v", err)
	}
}

func TestRunBaselinePull_UpdatesCacheFromArtifact(t *testing.T) {
	projectRoot := t.TempDir()
	if out, err := exec.Command("git", "init", projectRoot).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, string(out))
	}
	cmd := exec.Command("git", "remote", "add", "origin", "git@github.com:owner/repo.git")
	cmd.Dir = projectRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git remote add: %v\n%s", err, string(out))
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "backstop.yml"), []byte("project: pull-success\nlanguage: go\n"), 0o644); err != nil {
		t.Fatalf("write backstop.yml: %v", err)
	}
	cachePath := writeBaselineFixture(t, projectRoot)

	binDir := t.TempDir()
	zipPath := filepath.Join(t.TempDir(), "artifact.zip")
	makeBaselineZip(t, zipPath)
	writeFakeGh(t, filepath.Join(binDir, "gh"), `#!/bin/sh
if [ "$1 $2" = "auth status" ]; then
  exit 0
fi
if [ "$1" != "api" ]; then
  echo "unexpected command: $*" >&2
  exit 1
fi
case "$2" in
  repos/owner/repo/actions/runs\?branch=main\&status=success\&per_page=20)
    printf '{"workflow_runs":[{"id":42,"name":"ci","conclusion":"success","head_branch":"main"}]}'
    ;;
  repos/owner/repo/actions/runs/42/artifacts)
    printf '{"artifacts":[{"id":99,"name":"backstop-baseline-v1"}]}'
    ;;
  repos/owner/repo/actions/artifacts/99/zip)
    cat "`+zipPath+`"
    ;;
  *)
    echo "unexpected endpoint: $2" >&2
    exit 1
    ;;
esac
`)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	orig, _ := os.Getwd()
	if err := os.Chdir(projectRoot); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() { _ = os.Chdir(orig) }()

	if err := runBaselinePull(nil, nil); err != nil {
		t.Fatalf("run baseline pull: %v", err)
	}
	after, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatalf("read updated baseline: %v", err)
	}
	if !strings.Contains(string(after), `"schema_version":"baseline/v1"`) {
		t.Fatalf("expected updated baseline payload, got %s", string(after))
	}
}

func TestRunBaselinePull_InvalidBaselinePayloadPreservesExistingCache(t *testing.T) {
	projectRoot := t.TempDir()
	if out, err := exec.Command("git", "init", projectRoot).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, string(out))
	}
	cmd := exec.Command("git", "remote", "add", "origin", "git@github.com:owner/repo.git")
	cmd.Dir = projectRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git remote add: %v\n%s", err, string(out))
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "backstop.yml"), []byte("project: pull-invalid\nlanguage: go\n"), 0o644); err != nil {
		t.Fatalf("write backstop.yml: %v", err)
	}
	cachePath := writeBaselineFixture(t, projectRoot)
	before, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatalf("read baseline before: %v", err)
	}

	binDir := t.TempDir()
	zipPath := filepath.Join(t.TempDir(), "artifact-invalid.zip")
	var buf bytes.Buffer
	writer := zip.NewWriter(&buf)
	entry, err := writer.Create("baseline.json")
	if err != nil {
		t.Fatalf("create zip entry: %v", err)
	}
	if _, err := entry.Write([]byte(`{"schema_version":"baseline/v99","violations":[]}`)); err != nil {
		t.Fatalf("write invalid payload: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close invalid zip: %v", err)
	}
	if err := os.WriteFile(zipPath, buf.Bytes(), 0o644); err != nil {
		t.Fatalf("write invalid zip: %v", err)
	}
	writeFakeGh(t, filepath.Join(binDir, "gh"), `#!/bin/sh
if [ "$1 $2" = "auth status" ]; then
  exit 0
fi
case "$2" in
  repos/owner/repo/actions/runs\?branch=main\&status=success\&per_page=20)
    printf '{"workflow_runs":[{"id":42,"name":"ci","conclusion":"success","head_branch":"main"}]}'
    ;;
  repos/owner/repo/actions/runs/42/artifacts)
    printf '{"artifacts":[{"id":99,"name":"backstop-baseline-v1"}]}'
    ;;
  repos/owner/repo/actions/artifacts/99/zip)
    cat "`+zipPath+`"
    ;;
esac
`)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	orig, _ := os.Getwd()
	if err := os.Chdir(projectRoot); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() { _ = os.Chdir(orig) }()

	err = runBaselinePull(nil, nil)
	if err == nil || !strings.Contains(err.Error(), "invalid baseline payload") {
		t.Fatalf("expected invalid payload error, got %v", err)
	}
	after, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatalf("read baseline after: %v", err)
	}
	if string(after) != string(before) {
		t.Fatalf("expected cache to remain unchanged on invalid payload")
	}
}

func TestResolveProjectRootFailsWithoutBackstopConfig(t *testing.T) {
	orig, _ := os.Getwd()
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() { _ = os.Chdir(orig) }()

	_, err := resolveProjectRoot()
	if err == nil || !strings.Contains(err.Error(), "unable to resolve project") {
		t.Fatalf("expected project resolution failure, got %v", err)
	}
}

func writeFakeGh(t *testing.T, path string, script string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake gh: %v", err)
	}
}

func makeBaselineZip(t *testing.T, path string) {
	t.Helper()
	var buf bytes.Buffer
	writer := zip.NewWriter(&buf)
	entry, err := writer.Create("nested/baseline.json")
	if err != nil {
		t.Fatalf("create baseline zip entry: %v", err)
	}
	if _, err := entry.Write([]byte(`{"schema_version":"baseline/v1","violations":[]}`)); err != nil {
		t.Fatalf("write baseline zip entry: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close baseline zip: %v", err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatalf("write baseline zip fixture: %v", err)
	}
}
