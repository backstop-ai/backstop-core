package scaffold

import (
	"fmt"
	"os/exec"
	"strings"
)

// RealGitExecutor shells out to git for real operations.
type RealGitExecutor struct{}

func (r *RealGitExecutor) ListTags(pattern string) ([]string, error) {
	out, err := exec.Command("git", "tag", "-l", pattern).Output()
	if err != nil {
		return nil, fmt.Errorf("git tag -l: %w", err)
	}
	raw := strings.TrimSpace(string(out))
	if raw == "" {
		return nil, nil
	}
	return strings.Split(raw, "\n"), nil
}

func (r *RealGitExecutor) CreateAnnotatedTag(name, message string) error {
	if err := exec.Command("git", "tag", "-a", name, "-m", message).Run(); err != nil {
		return fmt.Errorf("git tag -a: %w", err)
	}
	return nil
}

func (r *RealGitExecutor) PushTag(name string) error {
	out, err := exec.Command("git", "push", "origin", name).CombinedOutput()
	if err != nil {
		outStr := string(out)
		if strings.Contains(outStr, "already exists") || strings.Contains(outStr, "rejected") {
			return &TagConflictError{Tag: name}
		}
		return fmt.Errorf("git push origin %s: %s", name, outStr)
	}
	return nil
}

func (r *RealGitExecutor) FetchTags() error {
	if err := exec.Command("git", "fetch", "--tags").Run(); err != nil {
		return fmt.Errorf("git fetch --tags: %w", err)
	}
	return nil
}

func (r *RealGitExecutor) IsGitRepo() bool {
	err := exec.Command("git", "rev-parse", "--is-inside-work-tree").Run()
	return err == nil
}

func (r *RealGitExecutor) IsGitAvailable() bool {
	_, err := exec.LookPath("git")
	return err == nil
}
