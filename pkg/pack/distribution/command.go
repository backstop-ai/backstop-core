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

// dependency names carried by *MissingDependencyError. They are the words an
// operator reads at a failed wiring site, so they are stated once here rather
// than spelled slightly differently at each of the ten guards below.
const (
	depGitCloner       = "git cloner"
	depValidator       = "validator"
	depVersionResolver = "version resolver"
	depScanner         = "scanner"
	depRemediation     = "remediation generator"
)

// AddCommand is `pack add`: it clones a pack's repository and validates the pack
// before anything is copied into the consumer project.
//
// The dependency fields are UNEXPORTED, so an AddCommand cannot be built by
// composite literal outside this package and NewAddCommand is the only assembly
// path. That is the structural half of REQ-006 — the constructor's nil guards
// only matter if there is no way around them.
type AddCommand struct {
	git       GitCloner
	validator Validator
}

// NewAddCommand assembles `pack add` over a cloner and a validator.
//
// Both are required because pack add clones AND validates. The cloner is
// required even though add's local-path branch never clones: NewExecGitCloner
// probes nothing at construction time, so a local-only consumer pays nothing for
// it, while a dependency that is "optional when a flag is set" is exactly the
// shape that let a nil reach a live code path in the first place.
func NewAddCommand(git GitCloner, validator Validator) (*AddCommand, error) {
	if git == nil {
		return nil, &MissingDependencyError{Command: "AddCommand", Dependency: depGitCloner}
	}
	if validator == nil {
		return nil, &MissingDependencyError{Command: "AddCommand", Dependency: depValidator}
	}

	return &AddCommand{git: git, validator: validator}, nil
}

// Run executes the pack add pipeline with the command's own dependencies.
//
// TRANSITIONAL (removed in Phase 11 of PLAN-SPEC-055): the pipeline body still
// lives in the package-level Add, so this method copies its dependencies onto the
// options value and delegates. Phase 11 moves the body here and deletes both the
// free function and the options structs' dependency fields; until then both APIs
// coexist so the existing suite keeps compiling. This delegation is NOT the
// intended end shape.
func (c *AddCommand) Run(packRef string, opts AddOptions) (*AddResult, error) {
	opts.GitCloner = c.git
	opts.Validator = c.validator

	return Add(packRef, opts)
}

// InstallCommand is `pack install`: the hash-verified RESTORE path that
// materializes what backstop.lock already records.
//
// Its only dependency is the cloner. Install does not re-validate — the packs it
// restores were validated when they were added, and their content hashes are
// what prove they are unchanged (DD-12).
type InstallCommand struct {
	git GitCloner
}

// NewInstallCommand assembles `pack install` over a cloner.
//
// There is deliberately no validator argument; adding one "for symmetry" would
// assert a re-validation install does not perform. The cloner is required even
// for the --cache path, which never clones, for the same reason add's is.
func NewInstallCommand(git GitCloner) (*InstallCommand, error) {
	if git == nil {
		return nil, &MissingDependencyError{Command: "InstallCommand", Dependency: depGitCloner}
	}

	return &InstallCommand{git: git}, nil
}

// Run executes the pack install pipeline with the command's own cloner.
//
// TRANSITIONAL (removed in Phase 11 of PLAN-SPEC-055) — see AddCommand.Run.
func (c *InstallCommand) Run(opts InstallOptions) (*InstallResult, error) {
	opts.GitCloner = c.git

	return Install(opts)
}

// UpdateCommand is `pack update`: it resolves the latest compatible version,
// clones it, validates it, and tamper-checks the installed copy.
//
// All three dependencies are required. The resolver in particular is no longer a
// field guarded at run time — an update with nothing to resolve versions with is
// not an update, so its absence is an assembly failure rather than a pipeline
// failure discovered after the operator has already invoked the command.
type UpdateCommand struct {
	git       GitCloner
	validator Validator
	resolver  VersionResolver
}

// NewUpdateCommand assembles `pack update` over a cloner, a validator, and a
// version resolver.
func NewUpdateCommand(git GitCloner, validator Validator, resolver VersionResolver) (*UpdateCommand, error) {
	if git == nil {
		return nil, &MissingDependencyError{Command: "UpdateCommand", Dependency: depGitCloner}
	}
	if validator == nil {
		return nil, &MissingDependencyError{Command: "UpdateCommand", Dependency: depValidator}
	}
	if resolver == nil {
		return nil, &MissingDependencyError{Command: "UpdateCommand", Dependency: depVersionResolver}
	}

	return &UpdateCommand{git: git, validator: validator, resolver: resolver}, nil
}

// Run executes the pack update pipeline with the command's own dependencies.
//
// TRANSITIONAL (removed in Phase 11 of PLAN-SPEC-055) — see AddCommand.Run.
func (c *UpdateCommand) Run(packName string, opts UpdateOptions) (*UpdateResult, error) {
	opts.GitCloner = c.git
	opts.Validator = c.validator
	opts.VersionResolver = c.resolver

	return Update(packName, opts)
}

// UpgradeCommand is `pack upgrade`: it crosses a major version, so beyond
// cloning and validating it scans the consumer codebase for violations the new
// major introduces and generates the remediation artifacts for them.
//
// All four dependencies are required. The scanner and remediation generator were
// previously skipped when nil, which turned an unwired upgrade into a silent
// partial success — the vacuous green this constructor exists to make impossible.
type UpgradeCommand struct {
	git         GitCloner
	validator   Validator
	scanner     Scanner
	remediation RemediationGenerator
}

// NewUpgradeCommand assembles `pack upgrade` over a cloner, a validator, a
// scanner, and a remediation generator.
func NewUpgradeCommand(git GitCloner, validator Validator, scanner Scanner, remediation RemediationGenerator) (*UpgradeCommand, error) {
	if git == nil {
		return nil, &MissingDependencyError{Command: "UpgradeCommand", Dependency: depGitCloner}
	}
	if validator == nil {
		return nil, &MissingDependencyError{Command: "UpgradeCommand", Dependency: depValidator}
	}
	if scanner == nil {
		return nil, &MissingDependencyError{Command: "UpgradeCommand", Dependency: depScanner}
	}
	if remediation == nil {
		return nil, &MissingDependencyError{Command: "UpgradeCommand", Dependency: depRemediation}
	}

	return &UpgradeCommand{git: git, validator: validator, scanner: scanner, remediation: remediation}, nil
}

// Run executes the pack upgrade pipeline with the command's own dependencies.
//
// TRANSITIONAL (removed in Phase 11 of PLAN-SPEC-055) — see AddCommand.Run.
func (c *UpgradeCommand) Run(packRef string, opts UpgradeOptions) (*UpgradeResult, error) {
	opts.GitCloner = c.git
	opts.Validator = c.validator
	opts.Scanner = c.scanner
	opts.RemediationGenerator = c.remediation

	return Upgrade(packRef, opts)
}
