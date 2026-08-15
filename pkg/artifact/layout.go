package artifact

// The ONE artifact-layout table and the ONE artifact-root resolver (SPEC-068 REQ-006).
//
// THE IMPORT SET IS LOAD-BEARING. pkg/artifact imports only stdlib, which is exactly
// why every consumer — pkg/gate, pkg/validate, pkg/scaffold, pkg/config, cmd/backstop —
// can import it with no cycle. In particular this file must NOT import pkg/config:
// ResolveRoot takes the declared STRING rather than a *Config, and that decision is
// what keeps the package importable everywhere.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Kind is the artifact kind vocabulary. It is the SAME seven the codebase already
// recognized in three separate places (cmd/backstop's artifactPatterns map,
// pkg/scaffold's ValidArtifactTypes, pkg/validate's resolvedByTypeDir); this
// declaration exists so there is ONE, not a fourth.
type Kind string

const (
	KindSpec       Kind = "spec"
	KindPlan       Kind = "plan"
	KindADR        Kind = "adr"
	KindBundle     Kind = "bundle"
	KindIssue      Kind = "issue"
	KindDirective  Kind = "directive"
	KindCapability Kind = "capability"
)

// KindLayout is where one kind lives and what its files are called. Directory is the
// bare directory NAME ("specs"), never a path — joining it to a root is Root.Dir's
// job, which is what keeps the artifact root in exactly one place.
// @waiver:backstop-ai/go-standards/backstop.packs.backstop-ai.go-standards.rules.core.go.core.error-type-suffix:false-positive:2026-11-14 pack rule fix pending — KindLayout is not an error type; the rule's dotall regex spans declarations and anchors on whatever struct precedes the file's first Error() method, which here is the correctly-suffixed RootMissingError
type KindLayout struct {
	Directory string
	Extension string
}

// kindEntry is one row of the layout table. The table is an ORDERED slice rather than
// a map so Kinds() is deterministic without a second, separately-maintained ordering
// list that could drift from it.
type kindEntry struct {
	kind      Kind
	directory string
	extension string
}

// kindTable is the single type→{Directory, Extension} authority.
var kindTable = []kindEntry{ // nosemgrep: go.core.no-global-mutable-state — immutable lookup table, package idiom
	{KindSpec, "specs", ".spec.md"},
	{KindPlan, "plans", ".plan.yml"},
	{KindADR, "adrs", ".adr.md"},
	{KindBundle, "bundles", ".bundle.md"},
	{KindIssue, "issues", ".issue.md"},
	{KindDirective, "directives", ".directive.md"},
	{KindCapability, "capabilities", ".capability.yml"},
}

// LayoutFor returns the layout for kind. It returns ok=false for an unrecognized
// kind rather than a zero-value KindLayout, whose empty Directory would silently
// resolve to the artifact root itself.
func LayoutFor(kind Kind) (KindLayout, bool) {
	for _, e := range kindTable {
		if e.kind == kind {
			return KindLayout{Directory: e.directory, Extension: e.extension}, true
		}
	}
	return KindLayout{}, false
}

// Kinds enumerates every artifact kind, deterministically.
func Kinds() []Kind {
	out := make([]Kind, 0, len(kindTable))
	for _, e := range kindTable {
		out = append(out, e.kind)
	}
	return out
}

// ClassifyFilename maps a filename to the kind its name declares, and reports
// ok=false for anything that is not artifact-shaped. It is EXCLUSIVE by construction:
// the extensions do not overlap, so a filename matches at most one kind.
//
// The ADR case additionally requires the ADR- prefix, preserving the behavior of the
// artifactPatterns map this replaces.
//
// Consumed by BOTH CLI discovery and the REQ-008 ungated-artifact scan, so the set of
// files the gate picks up and the set it reports leaving out are defined by the same
// predicate. Note that the capability test is on the `.capability.yml` SUFFIX: a file
// named exactly `capability.yml` is deliberately NOT a capability artifact.
func ClassifyFilename(name string) (Kind, bool) {
	for _, e := range kindTable {
		if !strings.HasSuffix(name, e.extension) {
			continue
		}
		if e.kind == KindADR && !strings.HasPrefix(name, "ADR-") {
			continue
		}
		return e.kind, true
	}
	return "", false
}

// nonCorpusDirNames are the directory names no artifact corpus scan descends into.
// It has ONE home because both CLI discovery and the REQ-008 ungated scan need it and
// their two rules differ ONLY in how each treats `.backstop` — discovery skips
// `.backstop` wholesale except when it IS the root, while the ungated scan always
// walks it and excludes only `.backstop/packs` beneath it. Two hand-typed copies can
// drift, and a drift makes the set of files the gate picks up and the set it reports
// leaving out disagree. `.backstop` is deliberately ABSENT here; each caller adds its
// own rule on top.
var nonCorpusDirNames = []string{ // nosemgrep: go.core.no-global-mutable-state — immutable name list, package idiom
	".git", "vendor", "node_modules", "testdata", "prototype",
}

// NonCorpusDirNames returns the shared non-corpus directory names, deterministically
// ordered. It returns a copy so a caller cannot mutate the shared list.
func NonCorpusDirNames() []string {
	out := make([]string, len(nonCorpusDirNames))
	copy(out, nonCorpusDirNames)
	return out
}

// Root is a resolved artifact root.
//
// Path is ALWAYS ABSOLUTE AND CLEANED — ResolveRoot absolutizes a relative
// projectRoot rather than passing it through, which is the decision that keeps the
// REQ-008 per-kind directory comparison from degenerating. Declared is the raw
// backstop.yml value (empty when unconfigured). Configured distinguishes "the consumer
// chose this root" from "nobody said, so it is the project root" — REQ-008's
// loud-failure condition keys on it.
type Root struct {
	Path       string
	Declared   string
	Configured bool
}

// Dir is the only sanctioned way to name an artifact type directory. Every literal
// filepath.Join(projectRoot, "specs") in the codebase becomes a call to this.
func (r Root) Dir(kind Kind) string {
	layout, ok := LayoutFor(kind)
	if !ok {
		return ""
	}
	return filepath.Join(r.Path, layout.Directory)
}

// RootMissingError reports a CONFIGURED artifact root that is absent from disk. It is
// typed so callers can distinguish REQ-008's loud failure from a malformed declaration
// without string matching.
type RootMissingError struct {
	Declared string
	Path     string
}

func (e *RootMissingError) Error() string {
	return fmt.Sprintf("configured artifact_root %q does not exist at %s", e.Declared, e.Path)
}

// RootInvalidError reports a declared artifact root the resolver refuses: absolute,
// escaping the project root, or naming something that is not a directory. Reason is
// always populated and human-readable — a caller that cannot name WHY cannot report
// it, and SPEC-070's doctor reports exactly this field.
type RootInvalidError struct {
	Declared string
	Reason   string
}

func (e *RootInvalidError) Error() string {
	return fmt.Sprintf("invalid artifact_root %q: %s", e.Declared, e.Reason)
}

// ResolveRoot is the ONE artifact-root resolution.
//
// projectRoot is absolutized before anything else, so the returned Root.Path is
// absolute even when the caller passes "." — which runGate does whenever config-path
// discovery fails. The DECLARED value is project-relative by rule (an absolute one is
// rejected) while the RESOLVED Path is absolute by guarantee; these are deliberately
// different strings.
//
// An absent declaration resolves to the project root marked unconfigured, with no
// error — that is the framework exception that keeps a repo-root layout working
// without configuring anything.
func ResolveRoot(projectRoot, declared string) (Root, error) {
	absProject, err := filepath.Abs(projectRoot)
	if err != nil {
		return Root{}, &RootInvalidError{Declared: declared, Reason: fmt.Sprintf("resolving project root %q: %v", projectRoot, err)}
	}
	absProject = filepath.Clean(absProject)

	if declared == "" {
		return Root{Path: absProject, Configured: false}, nil
	}

	if filepath.IsAbs(declared) {
		return Root{}, &RootInvalidError{Declared: declared, Reason: "must be a project-relative path, not an absolute one"}
	}

	joined := filepath.Clean(filepath.Join(absProject, declared))

	// Escape detection via filepath.Rel, NOT a string prefix test: a ".." segment and
	// a sibling directory that merely shares a name prefix are different failures, and
	// a prefix test conflates them.
	rel, relErr := filepath.Rel(absProject, joined)
	if relErr != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return Root{}, &RootInvalidError{Declared: declared, Reason: "escapes the project root"}
	}

	info, statErr := os.Stat(joined)
	if statErr != nil {
		if os.IsNotExist(statErr) {
			return Root{}, &RootMissingError{Declared: declared, Path: joined}
		}
		return Root{}, &RootInvalidError{Declared: declared, Reason: fmt.Sprintf("cannot be read at %s: %v", joined, statErr)}
	}
	if !info.IsDir() {
		return Root{}, &RootInvalidError{Declared: declared, Reason: fmt.Sprintf("is not a directory at %s", joined)}
	}

	return Root{Path: joined, Declared: declared, Configured: true}, nil
}
