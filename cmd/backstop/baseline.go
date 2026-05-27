package main

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/bmanson/backstop-core/pkg/config"
	"github.com/bmanson/backstop-core/pkg/gate"
	"github.com/spf13/cobra"
)

const baselineArtifactName = "backstop-baseline-v1"

func newBaselineCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "baseline",
		Short: "Baseline cache and artifact commands",
		Long:  "Commands for fetching and managing baseline artifacts used by gate baseline comparison.",
	}
	pullCmd := &cobra.Command{
		Use:   "pull",
		Short: "Fetch latest successful main baseline artifact",
		Long: `Fetches the baseline artifact from the latest successful main-branch CI run.
Artifact lookup uses GitHub Actions runs and artifact naming semantics, bypasses local TTL,
and writes .backstop/baseline.json atomically without corrupting existing cache on failure.`,
		RunE: runBaselinePull,
	}
	generateCmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate baseline JSON from full-scope gate",
		Long: `Runs backstop gate in full-scope mode equivalent to --all and writes
.backstop/baseline.json as baseline/v1 JSON. This command is intended for CI
baseline publication and does not depend on local baseline cache TTL.`,
		RunE: runBaselineGenerate,
	}
	cmd.AddCommand(pullCmd, generateCmd)
	return cmd
}

func runBaselineGenerate(_ *cobra.Command, _ []string) error {
	if _, err := config.LoadConfig(); err != nil {
		return fmt.Errorf("baseline generate: config: %w", err)
	}
	cfgPath, err := config.DiscoverConfigPath()
	if err != nil {
		return fmt.Errorf("baseline generate: config: %w", err)
	}
	projectRoot := filepath.Dir(cfgPath)
	scope, err := gate.ComputeGateScope(projectRoot, gate.GateScopeModeAll, nil)
	if err != nil {
		return fmt.Errorf("baseline generate: scope: %w", err)
	}
	allowSeeding, changedFiles := ruleSetChangeSeedingContext(projectRoot, scope)
	g := gate.New(gate.WithSteps(buildGateSteps(projectRoot, scope)), gate.WithScope(scope), gate.WithRuleSetChangeSeedingAllowed(allowSeeding), gate.WithRuleSetChangeFiles(changedFiles))
	result, _ := g.Run(context.Background())
	artifact := gate.NewBaselineArtifactFromSteps(result.Steps, time.Now().UTC().Format(time.RFC3339), gitSHA(projectRoot), version)
	baselinePath := filepath.Join(projectRoot, ".backstop", "baseline.json")
	if err := gate.WriteBaseline(baselinePath, &artifact); err != nil {
		return fmt.Errorf("baseline generate: writing baseline cache: %w", err)
	}
	return nil
}

func gitSHA(projectRoot string) string {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = projectRoot
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func runBaselinePull(cmd *cobra.Command, args []string) error {
	_ = cmd
	_ = args
	projectRoot, err := resolveProjectRoot()
	if err != nil {
		return fmt.Errorf("baseline pull: %w", err)
	}
	repo, err := resolveRepositoryFromOrigin(projectRoot)
	if err != nil {
		return fmt.Errorf("baseline pull: %w", err)
	}
	if err := ensureGitHubAuth(projectRoot); err != nil {
		return fmt.Errorf("baseline pull: %w", err)
	}
	runID, err := resolveLatestSuccessfulMainRun(projectRoot, repo)
	if err != nil {
		return fmt.Errorf("baseline pull: %w", err)
	}
	artifactID, err := resolveBaselineArtifactID(projectRoot, repo, runID)
	if err != nil {
		return fmt.Errorf("baseline pull: %w", err)
	}
	baselineBytes, err := downloadBaselineArtifact(projectRoot, repo, artifactID)
	if err != nil {
		return fmt.Errorf("baseline pull: %w", err)
	}
	if err := validateBaselineBytes(baselineBytes); err != nil {
		return fmt.Errorf("baseline pull: artifact download failed: invalid baseline payload: %w", err)
	}
	baselinePath := filepath.Join(projectRoot, ".backstop", "baseline.json")
	if err := writeBaselineAtomically(baselinePath, baselineBytes); err != nil {
		return fmt.Errorf("baseline pull: %w", err)
	}
	return nil
}

func resolveProjectRoot() (string, error) {
	cfgPath, err := discoverConfigPath()
	if err != nil {
		return "", fmt.Errorf("unable to resolve project for baseline pull: %w", err)
	}
	return filepath.Dir(cfgPath), nil
}

func discoverConfigPath() (string, error) {
	path, err := config.DiscoverConfigPath()
	if err != nil {
		return "", fmt.Errorf("backstop.yml not found; baseline pull requires repository root config")
	}
	return path, nil
}

func validateBaselineBytes(data []byte) error {
	var artifact gate.BaselineArtifact
	if err := json.Unmarshal(data, &artifact); err != nil {
		return err
	}
	if artifact.SchemaVersion != "" && artifact.SchemaVersion != gate.BaselineSchemaV1 {
		return fmt.Errorf("unsupported baseline schema_version %q", artifact.SchemaVersion)
	}
	return nil
}

func resolveRepositoryFromOrigin(projectRoot string) (string, error) {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = projectRoot
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("missing origin remote; cannot resolve GitHub repository")
	}
	url := strings.TrimSpace(string(out))
	re := regexp.MustCompile(`github\.com[:/]([^/]+)/([^/.]+)(\.git)?$`)
	match := re.FindStringSubmatch(url)
	if len(match) < 3 {
		return "", fmt.Errorf("unable to resolve repository from origin remote %q", url)
	}
	return match[1] + "/" + match[2], nil
}

func ensureGitHubAuth(projectRoot string) error {
	cmd := exec.Command("gh", "auth", "status")
	cmd.Dir = projectRoot
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("missing GitHub authentication; run `gh auth login` for baseline pull")
	}
	return nil
}

func resolveLatestSuccessfulMainRun(projectRoot, repo string) (int64, error) {
	out, err := ghAPI(projectRoot, "repos/"+repo+"/actions/runs?branch=main&status=success&per_page=20")
	if err != nil {
		return 0, fmt.Errorf("workflow/run selection miss: unable to query successful main runs: %w", err)
	}
	var payload struct {
		WorkflowRuns []struct {
			ID         int64  `json:"id"`
			Name       string `json:"name"`
			Conclusion string `json:"conclusion"`
			HeadBranch string `json:"head_branch"`
		} `json:"workflow_runs"`
	}
	if err := json.Unmarshal(out, &payload); err != nil {
		return 0, fmt.Errorf("workflow/run selection miss: invalid run listing payload: %w", err)
	}
	for _, run := range payload.WorkflowRuns {
		if run.HeadBranch == "main" && run.Conclusion == "success" {
			return run.ID, nil
		}
	}
	return 0, fmt.Errorf("workflow/run selection miss: no latest successful main run found")
}

func resolveBaselineArtifactID(projectRoot, repo string, runID int64) (int64, error) {
	out, err := ghAPI(projectRoot, fmt.Sprintf("repos/%s/actions/runs/%d/artifacts", repo, runID))
	if err != nil {
		return 0, fmt.Errorf("missing artifact: unable to list artifacts for selected run: %w", err)
	}
	var payload struct {
		Artifacts []struct {
			ID   int64  `json:"id"`
			Name string `json:"name"`
		} `json:"artifacts"`
	}
	if err := json.Unmarshal(out, &payload); err != nil {
		return 0, fmt.Errorf("missing artifact: invalid artifact listing payload: %w", err)
	}
	for _, artifact := range payload.Artifacts {
		if artifact.Name == baselineArtifactName {
			return artifact.ID, nil
		}
	}
	return 0, fmt.Errorf("missing artifact: %q not found in selected run", baselineArtifactName)
}

func downloadBaselineArtifact(projectRoot, repo string, artifactID int64) ([]byte, error) {
	zipBytes, err := ghAPI(projectRoot, fmt.Sprintf("repos/%s/actions/artifacts/%d/zip", repo, artifactID))
	if err != nil {
		return nil, fmt.Errorf("artifact download failed: %w", err)
	}
	reader, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
	if err != nil {
		return nil, fmt.Errorf("artifact download failed: invalid zip payload: %w", err)
	}
	for _, file := range reader.File {
		if strings.HasSuffix(file.Name, "baseline.json") {
			rc, err := file.Open()
			if err != nil {
				return nil, fmt.Errorf("artifact download failed: open baseline entry: %w", err)
			}
			defer rc.Close()
			body, err := io.ReadAll(rc)
			if err != nil {
				return nil, fmt.Errorf("artifact download failed: read baseline entry: %w", err)
			}
			return body, nil
		}
	}
	return nil, fmt.Errorf("missing artifact: baseline.json entry absent from downloaded artifact")
}

func writeBaselineAtomically(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func ghAPI(projectRoot, endpoint string) ([]byte, error) {
	cmd := exec.Command("gh", "api", endpoint)
	cmd.Dir = projectRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%s", strings.TrimSpace(string(output)))
	}
	return output, nil
}
