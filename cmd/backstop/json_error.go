package main

import (
	"encoding/json"
	"errors"
	"io"

	"github.com/bmanson/backstop-core/pkg/pack/distribution"
)

// jsonErrorDocument is the structured error a failing pack lifecycle command renders
// under --json (SPEC-055 REQ-012).
//
// Three fields, all always present: the COMMAND PATH so a consumer running several
// lifecycle steps knows which one failed, the KIND so it can route without parsing
// prose, and the human MESSAGE so an operator reading the raw document still learns
// what happened.
type jsonErrorDocument struct {
	Command string `json:"command"`
	Kind    string `json:"kind"`
	Message string `json:"message"`
}

// The kinds a consumer switches on. They are derived from distribution's TYPED errors
// rather than from message text, which is what makes them stable across reworded
// diagnostics.
const (
	jsonErrorKindGit        = "git"
	jsonErrorKindValidation = "validation"
	jsonErrorKindDependency = "dependency"
	jsonErrorKindCapability = "capability"
	// jsonErrorKindVersion covers BOTH *VersionUnresolvedError and
	// *VersionMismatchError: each means "the version is wrong", and a consumer's
	// response to either is the same.
	jsonErrorKindVersion = "version"
	// jsonErrorKindIdentity is SEPARATE from version on purpose. Its remedy differs —
	// fix the manifest's name versus retag the repository — and a consumer switching on
	// kind must be able to tell those apart.
	jsonErrorKindIdentity = "identity"
	// jsonErrorKindUnknown is the DEFAULT, and it is deliberately a real value rather
	// than the empty string: a consumer's switch needs something to land on, and an
	// empty kind turns every unclassified failure into a silent fall-through.
	jsonErrorKindUnknown = "unknown"
)

// writeJSONError renders ONE structured JSON error document for a failing command to w.
//
// One document and nothing else: under --json the caller writes this to stdout and then
// returns an ExitCodeError with Explained set, so stdout holds exactly this object and
// stderr stays empty. ADR-0001's machine-first posture means a consumer pipes stdout
// straight into a parser — a second object, a log line, or a trailing human sentence
// would break it.
//
// err must be non-nil: every call site is inside an `if err != nil` on a lifecycle
// command's failure path, and a document rendering a nil failure would be a lie a
// consumer could act on.
func writeJSONError(w io.Writer, command string, err error) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(jsonErrorDocument{
		Command: command,
		Kind:    classifyJSONErrorKind(err),
		Message: err.Error(),
	})
}

// classifyJSONErrorKind maps a lifecycle failure onto its wire kind.
//
// errors.As, never a bare type assertion: the pipelines wrap their typed errors for
// context (fmt.Errorf("resolving version: %w", …)), and a type assertion would classify
// the bare values correctly while silently sending every wrapped one — which is most of
// what a real failure looks like — to the default kind.
func classifyJSONErrorKind(err error) string {
	var gitErr *distribution.GitError
	if errors.As(err, &gitErr) {
		return jsonErrorKindGit
	}

	var validationErr *distribution.ValidationError
	if errors.As(err, &validationErr) {
		return jsonErrorKindValidation
	}

	var dependencyErr *distribution.MissingDependencyError
	if errors.As(err, &dependencyErr) {
		return jsonErrorKindDependency
	}

	var capabilityErr *distribution.CapabilityUnavailableError
	if errors.As(err, &capabilityErr) {
		return jsonErrorKindCapability
	}

	var versionUnresolvedErr *distribution.VersionUnresolvedError
	if errors.As(err, &versionUnresolvedErr) {
		return jsonErrorKindVersion
	}

	var versionMismatchErr *distribution.VersionMismatchError
	if errors.As(err, &versionMismatchErr) {
		return jsonErrorKindVersion
	}

	var identityErr *distribution.IdentityError
	if errors.As(err, &identityErr) {
		return jsonErrorKindIdentity
	}

	return jsonErrorKindUnknown
}

// packLifecycleFailure renders a failing pack lifecycle command's error under the
// disposition --json selects, and returns the ExitCodeError its RunE must return
// (SPEC-055 REQ-012).
//
// It is one function rather than four inlined copies because the DECISION is identical
// at all four sites; what differs is the command path, which stays a per-call argument
// precisely so each command's document names ITSELF (CLM-089). A shared constant there
// would make that claim unfalsifiable.
func packLifecycleFailure(out io.Writer, jsonFlag *bool, command string, err error) *ExitCodeError {
	if jsonFlag != nil && *jsonFlag {
		// Explained: the document on stdout IS the diagnostic, so reportError adds no
		// human line and stdout holds exactly one parseable object (CLM-083, CLM-088).
		//
		// A render that FAILED falls through to the loud return below rather than
		// claiming to have explained anything — an unwritable stdout must leave the
		// operator with a message on stderr, never with silence.
		if writeErr := writeJSONError(out, command, err); writeErr == nil {
			return &ExitCodeError{Code: ExitViolations, Message: err.Error(), Explained: true}
		}
	}
	return &ExitCodeError{Code: ExitViolations, Message: err.Error()}
}
