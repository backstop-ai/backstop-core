package distribution

import "fmt"

// MissingDependencyError is the fail-closed refusal a lifecycle command's
// constructor returns when a required dependency was explicitly passed as nil.
//
// A positional constructor already makes an OMITTED dependency a compile error;
// an explicitly written nil remains expressible, and this is what it becomes. It
// names BOTH the command being assembled and the dependency that was nil, so the
// diagnostic identifies the WIRING SITE rather than reporting a nil dereference
// from somewhere deep in an execution path.
type MissingDependencyError struct {
	Command    string
	Dependency string
}

func (e *MissingDependencyError) Error() string {
	return fmt.Sprintf("cannot assemble %s: its %s is nil; supply one at the wiring site", e.Command, e.Dependency)
}

// CapabilityUnavailableError is what a capability that is DECLARED but not yet
// BUILT returns instead of a vacuous success.
//
// Reference names the requirement tracking the gap, so the diagnostic points at
// the WORK rather than reading as a defect: an operator learns the capability is
// scheduled, not broken. Returning it — rather than an empty result — is what
// keeps an unbuilt capability from silently reporting "nothing to report".
//
// It is declared in this package, not in the assembly layer, because a command's
// Run must classify and propagate it and the CLI's error rendering keys a kind
// off it, while the production implementations that RETURN it live where the
// wiring lives.
type CapabilityUnavailableError struct {
	Capability string
	Reference  string
}

func (e *CapabilityUnavailableError) Error() string {
	return fmt.Sprintf("%s is declared but not yet available; it is tracked by %s", e.Capability, e.Reference)
}
