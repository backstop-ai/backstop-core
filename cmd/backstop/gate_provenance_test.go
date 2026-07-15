package main

import (
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// gitInitRepoWithBackstop creates a temp git repo carrying a minimal backstop.yml and one
// commit, returning the repo root. The provenance fields are sourced from the actual
// repository HEAD, so a real git repo (not a bare temp dir) is required.
func gitInitRepoWithBackstop(t *testing.T) string {
	t.Helper()
	dir := setupTestDir(t, minimalBackstopYML)
	runGit := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
		}
	}
	runGit("init")
	runGit("config", "user.email", "test@backstop.example")
	runGit("config", "user.name", "Backstop Test")
	runGit("add", "-A")
	runGit("commit", "-m", "init")
	return dir
}

// repoHEAD returns the HEAD commit SHA of the git repo rooted at dir.
func repoHEAD(t *testing.T, dir string) string {
	t.Helper()
	shaCmd := exec.Command("git", "rev-parse", "HEAD")
	shaCmd.Dir = dir
	shaOut, err := shaCmd.Output()
	if err != nil {
		t.Fatalf("git rev-parse HEAD: %v", err)
	}
	return strings.TrimSpace(string(shaOut))
}

func gateJSON(t *testing.T, dir string) map[string]interface{} {
	t.Helper()
	out, exitCode := runBinary(t, dir, "gate", "--json")
	if exitCode != 0 && exitCode != 1 {
		t.Fatalf("unexpected gate exit %d\noutput: %s", exitCode, out)
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &parsed); err != nil {
		t.Fatalf("gate output is not valid JSON: %v\noutput: %s", err, out)
	}
	return parsed
}

func TestGateCLI_JSONOutputIncludesGitSHAAndGeneratedAt(t *testing.T) {
	if binaryPath == "" {
		t.Skip("binary not built")
	}
	dir := gitInitRepoWithBackstop(t)
	parsed := gateJSON(t, dir)

	gitSHA, ok := parsed["git_sha"].(string)
	if !ok || gitSHA == "" {
		t.Fatalf("gate JSON missing non-empty git_sha, got %v", parsed["git_sha"])
	}
	generatedAt, ok := parsed["generated_at"].(string)
	if !ok || generatedAt == "" {
		t.Fatalf("gate JSON missing non-empty generated_at, got %v", parsed["generated_at"])
	}
	if _, err := time.Parse(time.RFC3339, generatedAt); err != nil {
		t.Fatalf("generated_at %q is not RFC 3339: %v", generatedAt, err)
	}
}

func TestGateCLI_GitSHAMatchesRepoHEAD(t *testing.T) {
	if binaryPath == "" {
		t.Skip("binary not built")
	}
	dir := gitInitRepoWithBackstop(t)
	head := repoHEAD(t, dir)
	parsed := gateJSON(t, dir)

	gitSHA, _ := parsed["git_sha"].(string)
	if gitSHA != head {
		t.Fatalf("git_sha = %q, want repo HEAD %q", gitSHA, head)
	}
}

func TestGateCLI_SchemaVersionUnchangedByTraceFields(t *testing.T) {
	if binaryPath == "" {
		t.Skip("binary not built")
	}
	dir := gitInitRepoWithBackstop(t)
	parsed := gateJSON(t, dir)

	if sv, _ := parsed["schema_version"].(string); sv != "gate/v1" {
		t.Fatalf("schema_version = %q, want gate/v1 (trace/provenance must not bump it)", sv)
	}
}
