package gate

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/backstop-ai/backstop-core/pkg/check"
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
	Mode     GateScopeMode `json:"mode"`
	Files    []string      `json:"files"`
	Warnings []string      `json:"warnings,omitempty"`
	// RequestedBase is the revision the operator asked to diff against, recorded
	// VERBATIM as written. MergeBase is what it actually resolved to. Both are
	// reported so an empty CI scope is VISIBLE rather than silent — the difference
	// between a legible "green over 12 files since <sha>" and an unexplained green
	// over zero. Empty for every scope that did not use an explicit base.
	RequestedBase string `json:"requested_base,omitempty"`
	MergeBase     string `json:"merge_base,omitempty"`
	ProjectRoot   string `json:"-"`
	fileSet       map[string]struct{}
}

// ComputeGateScope resolves the gate scope once for a gate run.
//
// It is a THIN WRAPPER over ComputeGateScopeWithBase with an empty base, and that
// is deliberate: 40+ call sites depend on this signature, and there must be exactly
// ONE implementation of "what changed". A second resolver grown alongside this one
// is how the two silently disagree.
func ComputeGateScope(projectRoot string, mode GateScopeMode, files []string) (*GateScope, error) {
	return ComputeGateScopeWithBase(projectRoot, mode, files, "")
}

// ComputeGateScopeWithBase is the full-arity entry point. An EMPTY base reproduces
// bare diff mode exactly; a non-empty base scopes the run to the files changed since
// merge-base(HEAD, base) plus untracked files.
//
// WHY AN EXPLICIT BASE EXISTS AT ALL: a CI checkout is pristine. Bare diff mode
// resolves merge-base(HEAD, origin/main), which on a push to main IS HEAD, so
// `git diff --name-only HEAD` returns nothing — and there are no untracked files
// either. That is not a wrong scope, it is an EMPTY one, and an empty scope passes
// every dimension. On a pull_request at the default fetch-depth there is no
// origin/main remote-tracking ref at all, so both merge-base attempts fail and the
// code falls through to the same empty result carrying only a warning. An explicit
// base is what makes a CI run check anything.
//
// THERE IS NO FALLBACK PATH when a base is given — not to --all, not to HEAD, not
// to an empty list. Bare diff mode's fallbacks are precisely what make CI silent
// today, so inheriting them here would defeat the flag.
func ComputeGateScopeWithBase(projectRoot string, mode GateScopeMode, files []string, base string) (*GateScope, error) {
	// A base with any non-diff mode is a programming error the CLI layer should
	// already have rejected. Erroring beats ignoring it: silently dropping the base
	// would run a scope the caller did not ask for while it believed otherwise.
	if base != "" && mode != GateScopeModeDiff {
		return nil, fmt.Errorf("--base applies to diff scope only, but scope mode is %q", mode)
	}

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
		if base != "" {
			resolved, mergeBase, err := resolveGateScopeExplicitBase(projectRoot, base)
			if err != nil {
				// Return NO scope: a caller must not be handed something it could
				// mistake for a valid, merely-empty result. The wrap names the layer
				// that failed; the wrapped error already names the ref and the reason,
				// and the CLI surfaces the whole chain unchanged.
				return nil, fmt.Errorf("computing gate scope: %w", err)
			}
			scope := newGateScope(projectRoot, mode, resolved, nil)
			scope.RequestedBase = base
			scope.MergeBase = mergeBase
			return scope, nil
		}
		resolved, warnings, err := resolveGateScopeDiff(projectRoot)
		return newGateScope(projectRoot, mode, resolved, warnings), err
	default:
		return nil, fmt.Errorf("unknown gate scope mode: %s", mode)
	}
}

// resolveGateScopeExplicitBase resolves an EXPLICIT diff base. Every failure is loud
// and names what could not resolve; none of them degrades into a usable scope.
//
// It returns the in-scope files and the RESOLVED MERGE-BASE SHA, so the run can
// report which commit it actually compared against.
func resolveGateScopeExplicitBase(projectRoot, base string) ([]string, string, error) {
	// STEP 1: does the rev resolve to a commit at all? The `^{commit}` peel matters —
	// a tag object resolves under a bare rev-parse but is not something merge-base
	// can use.
	resolvedOut, err := gitOutput(projectRoot, "rev-parse", "--verify", base+"^{commit}")
	if err != nil {
		return nil, "", fmt.Errorf("--base %q does not resolve to a commit in this repository "+
			"(in CI this usually means the checkout is shallow — set fetch-depth: 0): %w", base, err)
	}
	resolved := strings.TrimSpace(resolvedOut)

	// HEAD is read only so the STEP 2 failure can name both sides: "no merge base"
	// without saying between what is unactionable in a CI log.
	headOut, headErr := gitOutput(projectRoot, "rev-parse", "--verify", "HEAD^{commit}")
	if headErr != nil {
		return nil, "", fmt.Errorf("--base %q resolved to %s, but HEAD does not resolve to a commit: %w",
			base, resolved, headErr)
	}
	head := strings.TrimSpace(headOut)

	// STEP 2: do they share history? The rev RESOLVES by here, so a step-1-only guard
	// would pass — this is what catches unrelated histories, the shape a force-push or
	// a grafted shallow clone produces.
	mergeBaseOut, mergeErr := gitOutput(projectRoot, "merge-base", "HEAD", resolved)
	if mergeErr != nil {
		return nil, "", fmt.Errorf("--base %q (resolved %s) shares no merge-base with HEAD (%s): "+
			"the two revisions have unrelated histories, which a force-push or a grafted "+
			"shallow clone produces: %w", base, resolved, head, mergeErr)
	}
	mergeBase := strings.TrimSpace(mergeBaseOut)
	if mergeBase == "" {
		return nil, "", fmt.Errorf("--base %q (resolved %s) shares no merge-base with HEAD (%s): "+
			"git merge-base returned no commit", base, resolved, head)
	}

	// STEP 3: the same two commands bare diff mode runs, with no fallback around them.
	tracked, diffErr := gitLines(projectRoot, "diff", "--name-only", mergeBase)
	if diffErr != nil {
		return nil, "", fmt.Errorf("--base %q: diffing against merge-base %s failed: %w", base, mergeBase, diffErr)
	}
	untracked, untrackedErr := gitLines(projectRoot, "ls-files", "--others", "--exclude-standard")
	if untrackedErr != nil {
		// Parity with bare diff mode: the tracked diff is still authoritative, so
		// proceed tracked-only rather than aborting the scope computation.
		untracked = nil
	}
	return append(tracked, untracked...), mergeBase, nil
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
