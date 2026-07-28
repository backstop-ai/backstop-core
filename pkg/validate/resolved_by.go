package validate

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/backstop-ai/backstop-core/pkg/artifact"
)

// resolved-by (ISSUE-048) lets a `closed` issue that was fixed DIRECTLY — a real
// commit, with NO issue->plan track and possibly NO test — satisfy
// close-requires-traceability by pointing at the resolving work. It is LIGHTER
// than delivered_by: it requires neither a backing plan nor a mandated test. Like
// delivered_by, it is STATIC and trust-based (liveness is ISSUE-042's gate job):
// this file does static filesystem reads ONLY (filepath, os.Stat, filepath.Glob),
// never os/exec, and never imports pkg/gate — pkg/gate consumes validators, so
// importing it here would invert the layering.
//
// resolved-by must be a STRUCTURED ref, NEVER free text. A bare `resolved-by: done`
// plus a Resolution heading would otherwise be a vacuous-close hatch that bypasses
// ALL REQ->CLM->test rigor AND is invisible to the drift dimension (a resolved-by
// close carries no claims). So the accept-boundary is:
//
//	(a) a TYPED artifact ref (BUNDLE|SPEC|ISSUE|PLAN|DIR)-NNN — additionally the
//	    referenced artifact FILE must EXIST (the same static sibling-file read
//	    delivered_by does); else issue/resolved-by-artifact-not-found; OR
//	(b) a COMMIT/PR ref — a hex SHA or a PR URL — accepted SHAPE-ONLY (existence
//	    is skipped: a static validator cannot confirm a SHA).
//
// Anything else (empty/blank/arbitrary prose) is issue/resolved-by-malformed.
//
// NB the asymmetry vs obsoleted-by/replaced-by, which are SHAPE-ONLY typed refs
// with NO existence check (the ISSUE-031 retirement-field trust level). resolved-by
// checks typed-ref existence to stay no looser than delivered_by (which resolves to
// a real completed plan).
var (
	resolvedByTypedRefRe = regexp.MustCompile(`^(BUNDLE|SPEC|ISSUE|PLAN|DIR)-[0-9]{3}$`) // nosemgrep: go.core.no-global-mutable-state — immutable compiled-regex singleton, package idiom
	resolvedByCommitRe   = regexp.MustCompile(`^[0-9a-f]{7,40}$`)                        // nosemgrep: go.core.no-global-mutable-state — immutable compiled-regex singleton, package idiom
)

// resolvedByTypeDir maps a typed-ref prefix to the sibling artifact directory and
// filename extension its files use (all artifacts are named <PREFIX>-NNN-slug.<ext>).
// The existence check globs <parent>/<dir>/<REF>-*<ext>.
var resolvedByTypeDir = map[string]struct{ dir, ext string }{ // nosemgrep: go.core.no-global-mutable-state — immutable lookup table, package idiom
	"BUNDLE": {"bundles", ".bundle.md"},
	"SPEC":   {"specs", ".spec.md"},
	"ISSUE":  {"issues", ".issue.md"},
	"PLAN":   {"plans", ".plan.yml"},
	"DIR":    {"directives", ".directive.md"},
}

// validateResolvedBy runs the static structured-ref conditions that let a closed
// issue satisfy traceability by tracing to its resolving work (ISSUE-048). It
// returns a (possibly empty) slice of fail-loud violations; an empty result means
// the ref is accepted. Every failure mode has a DISTINCT rule name at error
// severity — a bad resolved-by NEVER silently satisfies the close.
func validateResolvedBy(art *artifact.ParsedArtifact, resolvedBy string) []Violation {
	ref := strings.TrimSpace(resolvedBy)

	// Empty / blank is malformed — this is the accept-boundary that closes the
	// vacuous-close hatch (CLM-005).
	if ref == "" {
		return []Violation{{
			Rule:     "issue/resolved-by-malformed",
			File:     art.Filename,
			Message:  "resolved-by must be a structured ref: a typed artifact id (BUNDLE/SPEC/ISSUE/PLAN/DIR-NNN) or a commit/PR ref (hex SHA or PR URL); empty/blank is not permitted",
			Severity: "error",
		}}
	}

	// Typed artifact ref: statically verify the referenced artifact file EXISTS.
	if resolvedByTypedRefRe.MatchString(ref) {
		if !typedRefArtifactExists(art, ref) {
			return []Violation{{
				Rule:     "issue/resolved-by-artifact-not-found",
				File:     art.Filename,
				Message:  fmt.Sprintf("resolved-by '%s' is a typed artifact ref but no matching artifact file resolves from the issue's source path", ref),
				Severity: "error",
			}}
		}
		return nil
	}

	// Commit/PR ref: shape-only, existence is not statically checkable — accept.
	if resolvedByCommitRe.MatchString(ref) || isPullRequestURL(ref) {
		return nil
	}

	// Anything else (arbitrary prose) is malformed.
	return []Violation{{
		Rule:     "issue/resolved-by-malformed",
		File:     art.Filename,
		Message:  fmt.Sprintf("resolved-by '%s' is neither a typed artifact id (BUNDLE/SPEC/ISSUE/PLAN/DIR-NNN) nor a commit/PR ref (hex SHA or PR URL)", ref),
		Severity: "error",
	}}
}

// typedRefArtifactExists reports whether the typed ref resolves to a real sibling
// artifact file, anchored on the issue's OWN source path (never the ambient CWD),
// exactly like delivered_by's resolvePlansDir. Production layout:
// <root>/issues/ISSUE-NNN.issue.md, so an ISSUE ref resolves to <root>/issues, a
// SPEC ref to <root>/specs, etc. A directory-less SourcePath, an unknown prefix,
// or zero glob matches all yield false (the caller flags artifact-not-found).
// Static reads only (filepath.Dir/Join, filepath.Glob) — no os/exec.
func typedRefArtifactExists(art *artifact.ParsedArtifact, ref string) bool {
	if art.SourcePath == "" {
		return false
	}
	dir := filepath.Dir(art.SourcePath)
	if dir == "." || dir == "" {
		return false
	}
	prefix := strings.SplitN(ref, "-", 2)[0]
	td, ok := resolvedByTypeDir[prefix]
	if !ok {
		return false
	}
	siblingDir := filepath.Join(dir, "..", td.dir)
	// ref has no glob metacharacters (typed-ref regex), so a Glob error can only
	// mean a bad pattern we did not build — treat like zero matches. A non-existent
	// or non-directory siblingDir also yields zero matches, so the glob result alone
	// answers existence (no separate os.Stat needed).
	matches, err := filepath.Glob(filepath.Join(siblingDir, ref+"-*"+td.ext))
	return err == nil && len(matches) > 0
}

// isPullRequestURL reports whether ref looks like a GitHub-style PR URL — an
// https URL whose path contains /pull/ or /pulls/. Shape-only, like a commit SHA.
func isPullRequestURL(ref string) bool {
	if !strings.HasPrefix(ref, "https://") {
		return false
	}
	return strings.Contains(ref, "/pull/") || strings.Contains(ref, "/pulls/")
}
