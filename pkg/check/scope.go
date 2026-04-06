// Package check implements the backstop code check validation engine.
package check

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ScopeMode determines how the file list for validation is resolved.
type ScopeMode int

const (
	// ScopeModeDiff uses git merge-base cascade to find changed files.
	ScopeModeDiff ScopeMode = iota
	// ScopeModeAll walks the entire project directory.
	ScopeModeAll
	// ScopeModeFile checks a single file.
	ScopeModeFile
)

// errNoRemote signals that a remote branch does not exist.
var errNoRemote = errors.New("remote branch not found")

// GitExecutor abstracts git operations for testability.
type GitExecutor interface {
	IsGitRepo() bool
	MergeBase(remote string) (string, error)
	DiffNameOnly(base string) ([]string, error)
	DiffLocal() ([]string, error)
}

// DefaultGitExecutor shells out to git for scope resolution.
type DefaultGitExecutor struct {
	Dir string
}

// IsGitRepo checks if the working directory is a git repository.
func (g *DefaultGitExecutor) IsGitRepo() bool {
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	cmd.Dir = g.Dir
	out, err := cmd.Output()
	return err == nil && strings.TrimSpace(string(out)) == "true"
}

// MergeBase finds the merge-base between HEAD and the given remote branch.
func (g *DefaultGitExecutor) MergeBase(remote string) (string, error) {
	cmd := exec.Command("git", "merge-base", "HEAD", remote)
	cmd.Dir = g.Dir
	out, err := cmd.Output()
	if err != nil {
		return "", errNoRemote
	}
	return strings.TrimSpace(string(out)), nil
}

// DiffNameOnly returns files changed between HEAD and the given base commit.
func (g *DefaultGitExecutor) DiffNameOnly(base string) ([]string, error) {
	cmd := exec.Command("git", "diff", "--name-only", base+"..HEAD")
	cmd.Dir = g.Dir
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git diff --name-only: %w", err)
	}
	return splitLines(string(out)), nil
}

// DiffLocal returns staged and unstaged changed files.
func (g *DefaultGitExecutor) DiffLocal() ([]string, error) {
	cmd := exec.Command("git", "diff", "--name-only", "HEAD")
	cmd.Dir = g.Dir
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git diff --name-only HEAD: %w", err)
	}
	return splitLines(string(out)), nil
}

// scopeOption configures scope resolution behavior.
type scopeOption func(*scopeConfig)

type scopeConfig struct {
	projectDir string
}

func withProjectDir(dir string) scopeOption {
	return func(c *scopeConfig) {
		c.projectDir = dir
	}
}

// ResolveScope resolves the file list for a given scope mode.
// It uses a DefaultGitExecutor for git operations.
func ResolveScope(mode ScopeMode, filePath string) ([]string, []string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, nil, fmt.Errorf("getting working directory: %w", err)
	}
	git := &DefaultGitExecutor{Dir: cwd}
	return resolveScopeWithGit(mode, filePath, git, withProjectDir(cwd))
}

// resolveScopeWithGit resolves the file list using an injected GitExecutor.
func resolveScopeWithGit(mode ScopeMode, filePath string, git GitExecutor, opts ...scopeOption) ([]string, []string, error) {
	cfg := &scopeConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	switch mode {
	case ScopeModeFile:
		return resolveScopeFile(filePath)
	case ScopeModeAll:
		return resolveScopeAll(cfg.projectDir)
	case ScopeModeDiff:
		return resolveScopeDiff(git, cfg.projectDir)
	default:
		return nil, nil, fmt.Errorf("unknown scope mode: %d", mode)
	}
}

// resolveScopeFile verifies the file exists and returns it as a single-element slice.
func resolveScopeFile(filePath string) ([]string, []string, error) {
	if filePath == "" {
		return nil, nil, fmt.Errorf("--file requires a file path")
	}
	if _, err := os.Stat(filePath); err != nil {
		return nil, nil, fmt.Errorf("file %q: %w", filePath, err)
	}
	return []string{filePath}, nil, nil
}

// resolveScopeAll walks the project directory for all relevant files.
func resolveScopeAll(projectDir string) ([]string, []string, error) {
	if projectDir == "" {
		return nil, nil, fmt.Errorf("project directory not set for all-scope resolution")
	}
	var files []string
	err := filepath.Walk(projectDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip errors
		}
		// Skip hidden directories
		if info.IsDir() && strings.HasPrefix(info.Name(), ".") && info.Name() != "." {
			return filepath.SkipDir
		}
		if !info.IsDir() {
			rel, relErr := filepath.Rel(projectDir, path)
			if relErr != nil {
				rel = path
			}
			files = append(files, rel)
		}
		return nil
	})
	if err != nil {
		return nil, nil, fmt.Errorf("walking directory: %w", err)
	}
	return files, nil, nil
}

// resolveScopeDiff executes the 4-step git merge-base cascade (REQ-013).
func resolveScopeDiff(git GitExecutor, projectDir string) ([]string, []string, error) {
	// Step 4 check: is this a git repo?
	if git == nil || !git.IsGitRepo() {
		files, warnings, err := resolveScopeAll(projectDir)
		warnings = append(warnings, "not a git repository; falling back to full codebase scan (--all)")
		return files, warnings, err
	}

	// Step 1: try origin/main
	base, err := git.MergeBase("origin/main")
	if err == nil {
		files, diffErr := git.DiffNameOnly(base)
		if diffErr == nil {
			return files, nil, nil
		}
	}

	// Step 2: try origin/master
	base, err = git.MergeBase("origin/master")
	if err == nil {
		files, diffErr := git.DiffNameOnly(base)
		if diffErr == nil {
			return files, nil, nil
		}
	}

	// Step 3: local staged + unstaged
	files, err := git.DiffLocal()
	if err != nil {
		// Even local diff failed — fall back to all
		allFiles, warnings, allErr := resolveScopeAll(projectDir)
		warnings = append(warnings, fmt.Sprintf("git diff failed: %v; falling back to full codebase scan", err))
		return allFiles, warnings, allErr
	}
	warnings := []string{"no remote branch (origin/main or origin/master) found; using local changes only"}
	return files, warnings, nil
}

// splitLines splits output by newlines, filtering empty lines.
func splitLines(s string) []string {
	var lines []string
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}
