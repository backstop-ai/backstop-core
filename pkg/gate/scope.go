package gate

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmanson/backstop-core/pkg/check"
)

// GateScopeMode is the gate-level scope mode serialized in gate output.
type GateScopeMode string

const (
	GateScopeModeDiff GateScopeMode = "diff"
	GateScopeModeAll  GateScopeMode = "all"
	GateScopeModeFile GateScopeMode = "file"
)

// GateScope describes the single file scope computed at gate startup.
type GateScope struct {
	Mode        GateScopeMode `json:"mode"`
	Files       []string      `json:"files"`
	Warnings    []string      `json:"warnings,omitempty"`
	ProjectRoot string        `json:"-"`
	fileSet     map[string]struct{}
}

// ComputeGateScope resolves the gate scope once for a gate run.
func ComputeGateScope(projectRoot string, mode GateScopeMode, files []string) (*GateScope, error) {
	switch mode {
	case GateScopeModeAll:
		resolved, warnings, err := resolveGateScopeAll(projectRoot)
		return newGateScope(projectRoot, mode, resolved, warnings), err
	case GateScopeModeFile:
		if len(files) == 0 {
			return nil, fmt.Errorf("--file requires at least one file path")
		}
		return newGateScope(projectRoot, mode, files, nil), nil
	case GateScopeModeDiff:
		resolved, warnings, err := resolveGateScopeDiff(projectRoot)
		return newGateScope(projectRoot, mode, resolved, warnings), err
	default:
		return nil, fmt.Errorf("unknown gate scope mode: %s", mode)
	}
}

// GateScopeModeFromCheck maps existing check scope modes to gate scope modes.
func GateScopeModeFromCheck(mode check.ScopeMode) GateScopeMode {
	switch mode {
	case check.ScopeModeAll:
		return GateScopeModeAll
	case check.ScopeModeFile:
		return GateScopeModeFile
	default:
		return GateScopeModeDiff
	}
}

// Contains reports whether path is in scope. All mode contains every path.
func (s *GateScope) Contains(path string) bool {
	if s == nil || s.Mode == GateScopeModeAll {
		return true
	}
	_, ok := s.fileSet[NormalizePath(s.ProjectRoot, path)]
	return ok
}

func (s *GateScope) Empty() bool {
	return s != nil && s.Mode != GateScopeModeAll && len(s.Files) == 0
}

func newGateScope(projectRoot string, mode GateScopeMode, files []string, warnings []string) *GateScope {
	set := make(map[string]struct{})
	for _, file := range files {
		if normalized := NormalizePath(projectRoot, file); normalized != "" {
			set[normalized] = struct{}{}
		}
	}
	stable := make([]string, 0, len(set))
	for file := range set {
		stable = append(stable, file)
	}
	sort.Strings(stable)
	return &GateScope{Mode: mode, Files: stable, Warnings: warnings, ProjectRoot: projectRoot, fileSet: set}
}

// NormalizePath canonicalizes a file path into the ONE repo-relative form every
// violation identity and scope decision uses. It is path-string-only — no
// language noun, no extension literal, no toolchain knowledge (CLM-010): it
// applies filepath.Clean + filepath.ToSlash and, when projectRoot is non-empty,
// rel-ifies an absolute path against it. With projectRoot=="" the abs→rel step is
// a no-op, so it degrades to the idempotent Clean+ToSlash+strip-"./" subset — the
// SAME function used at the projectRoot-free identity chokepoint (baseline.go), so
// there is exactly ONE normalization implementation with no drift.
func NormalizePath(projectRoot, file string) string {
	if strings.TrimSpace(file) == "" {
		return ""
	}
	clean := filepath.Clean(file)
	if filepath.IsAbs(clean) && projectRoot != "" {
		if rel, err := filepath.Rel(projectRoot, clean); err == nil {
			clean = rel
		}
	}
	return filepath.ToSlash(clean)
}

func resolveGateScopeDiff(projectRoot string) ([]string, []string, error) {
	if !gitOK(projectRoot, "rev-parse", "--is-inside-work-tree") {
		files, warnings, err := resolveGateScopeAll(projectRoot)
		return files, append(warnings, "not a git repository; falling back to full codebase scan (--all)"), err
	}

	for _, remote := range []string{"origin/main", "origin/master"} {
		base, err := gitOutput(projectRoot, "merge-base", "HEAD", remote)
		if err == nil && strings.TrimSpace(base) != "" {
			tracked, diffErr := gitLines(projectRoot, "diff", "--name-only", strings.TrimSpace(base))
			if diffErr == nil {
				untracked, untrackedErr := gitLines(projectRoot, "ls-files", "--others", "--exclude-standard")
				if untrackedErr != nil {
					return tracked, nil, nil
				}
				return append(tracked, untracked...), nil, nil
			}
		}
	}

	tracked, err := gitLines(projectRoot, "diff", "--name-only", "HEAD")
	if err != nil {
		files, warnings, allErr := resolveGateScopeAll(projectRoot)
		return files, append(warnings, fmt.Sprintf("git diff failed: %v; falling back to full codebase scan", err)), allErr
	}
	untracked, untrackedErr := gitLines(projectRoot, "ls-files", "--others", "--exclude-standard")
	if untrackedErr != nil {
		// Untracked enumeration failed; the tracked diff is still authoritative, so
		// proceed with tracked-only rather than aborting the scope computation.
		untracked = nil
	}
	return append(tracked, untracked...), []string{"no remote branch (origin/main or origin/master) found; using local changes only"}, nil
}

func resolveGateScopeAll(projectRoot string) ([]string, []string, error) {
	var files []string
	err := filepath.Walk(projectRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil {
			return nil
		}
		if info.IsDir() && strings.HasPrefix(info.Name(), ".") && info.Name() != "." {
			return filepath.SkipDir
		}
		if info.IsDir() {
			return nil
		}
		if rel, relErr := filepath.Rel(projectRoot, path); relErr == nil {
			files = append(files, rel)
		}
		return nil
	})
	return files, nil, err
}

func gitOK(dir string, args ...string) bool {
	out, err := gitOutput(dir, args...)
	return err == nil && strings.TrimSpace(out) == "true"
}

func gitLines(dir string, args ...string) ([]string, error) {
	out, err := gitOutput(dir, args...)
	if err != nil {
		return nil, fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	var lines []string
	for _, line := range strings.Split(out, "\n") {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			lines = append(lines, trimmed)
		}
	}
	return lines, nil
}

func gitOutput(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return string(out), nil
}

func filterViolations(scope *GateScope, violations []Violation) []Violation {
	if scope == nil || scope.Mode == GateScopeModeAll {
		return violations
	}
	filtered := make([]Violation, 0, len(violations))
	for _, violation := range violations {
		// Project-wide (build/typecheck) violations are NEVER scope-filtered:
		// they may legitimately reference out-of-scope files and must still
		// fail a diff-scoped gate (Ratified Design Constraint 3). This is a
		// structural exemption on the ProjectWide field, not a Rule string
		// match — the tsc/sarif parsers populate Rule (e.g. "TS2304"), so a
		// Rule=="build" match would miss them.
		if violation.ProjectWide {
			filtered = append(filtered, violation)
			continue
		}
		if violation.File != "" && scope.Contains(violation.File) {
			filtered = append(filtered, violation)
		}
	}
	if filtered == nil {
		return []Violation{}
	}
	return filtered
}

// FilterViolations is the exported wrapper over filterViolations so out-of-package
// callers — specifically cmd/backstop's pack_engines step (packValidatorStep) — apply
// the SAME diff-scope filter the delegate and baseline paths already use. ISSUE-070:
// packValidatorStep set status on the raw dispatch output WITHOUT filtering, so
// project-wide NON-exempt lint violations (golangci errcheck/unused/…) on UNCHANGED
// files leaked past diff-scope and redded the gate on any change. A nil or ModeAll
// scope returns the violations unchanged (whole-repo sweep); ProjectWide (exempt)
// violations are still kept regardless of scope.
func (s *GateScope) FilterViolations(violations []Violation) []Violation {
	return filterViolations(s, violations)
}
