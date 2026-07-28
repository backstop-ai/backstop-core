package distribution_test

// Constructor matrix for the four lifecycle commands (SPEC-055 REQ-006).
//
// The point of these tests is not that assembly can fail — it is WHICH failure an
// operator reads. A positional constructor already turns an OMITTED dependency
// into a compile error; an explicitly written nil is the residue, and every cell
// below asserts the residue names the dependency and the command being assembled.
// A constructor that reports the wrong dependency satisfies "err != nil" and
// defeats the entire diagnostic purpose of a typed assembly error, so no test
// here stops at the presence of an error.
//
// The mocks are the package suite's existing ones (mockGitCloner/mockValidator
// from add_test.go, mockVersionResolver from update_test.go, mockScanner and
// mockRemediationGenerator from upgrade_test.go) — a parallel mock set would let
// this suite drift from what the pipelines are actually exercised against.

import (
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/pack/distribution"
)

// assertMissingDependency is the shared shape of every nil cell: a failed
// assembly yields NOTHING usable and an error identifying both the command and
// the dependency that was nil.
//
// assembled carries the caller's typed nil-check because a typed nil pointer
// stuffed into an interface here would compare non-nil and silently pass.
func assertMissingDependency(t *testing.T, constructor string, assembled bool, err error, wantCommand, wantDependency string) {
	t.Helper()

	if err == nil {
		t.Fatalf("%s returned no error for a nil %s; the nil would surface later as a dereference far from the wiring site", constructor, wantDependency)
	}
	if assembled {
		t.Errorf("%s returned a command alongside its error; a failed assembly must never yield a partially built command", constructor)
	}

	var missing *distribution.MissingDependencyError
	if !errors.As(err, &missing) {
		t.Fatalf("%s returned %T (%v), want a *distribution.MissingDependencyError", constructor, err, err)
	}

	if !strings.Contains(strings.ToLower(missing.Dependency), strings.ToLower(wantDependency)) {
		t.Errorf("%s names the missing dependency as %q, which does not identify the %s", constructor, missing.Dependency, wantDependency)
	}
	if !strings.Contains(strings.ToLower(missing.Command), strings.ToLower(wantCommand)) {
		t.Errorf("%s names the command as %q, which does not identify %s — the diagnostic points at the wrong wiring site", constructor, missing.Command, wantCommand)
	}

	rendered := missing.Error()
	if !strings.Contains(rendered, missing.Dependency) {
		t.Errorf("the rendered message %q does not carry the dependency name %q", rendered, missing.Dependency)
	}
	if !strings.Contains(rendered, missing.Command) {
		t.Errorf("the rendered message %q does not carry the command name %q", rendered, missing.Command)
	}
}

// assertDependenciesStored proves a successful assembly actually RETAINED what it
// was handed: every dependency field is populated and there are exactly as many
// of them as the matrix says.
//
// Without this a constructor returning an empty &AddCommand{} passes a bare
// non-nil check while every Run against it nil-dereferences.
func assertDependenciesStored(t *testing.T, constructor string, cmd any, wantFields int) {
	t.Helper()

	v := reflect.ValueOf(cmd)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		t.Fatalf("%s returned %#v, want a non-nil command pointer", constructor, cmd)
	}

	s := v.Elem()
	if s.NumField() != wantFields {
		t.Errorf("%s built a command with %d fields, want %d — the dependency matrix and the struct disagree", constructor, s.NumField(), wantFields)
	}

	for i := range s.NumField() {
		f := s.Field(i)
		if f.Kind() != reflect.Interface {
			t.Errorf("%s: field %q is a %s, want an interface — a command must depend on abstractions, not concretions", constructor, s.Type().Field(i).Name, f.Kind())
			continue
		}
		if f.IsNil() {
			t.Errorf("%s left field %q nil despite a complete assembly; the dependency was dropped on the floor", constructor, s.Type().Field(i).Name)
		}
	}
}

func realCloner() distribution.GitCloner { return &mockGitCloner{} }

func realValidator() distribution.Validator { return &mockValidator{} }

func realResolver() distribution.VersionResolver { return &mockVersionResolver{} }

func realScanner() distribution.Scanner { return &mockScanner{} }

func realRemediation() distribution.RemediationGenerator { return &mockRemediationGenerator{} }

// --- AddCommand (CLM-032/033/034/035) ---

// TestNewAddCommand_CompleteAssemblySucceeds proves pack add's two-dependency
// assembly succeeds and retains both (CLM-032). Add clones AND validates, so a
// command missing either is not an add command.
func TestNewAddCommand_CompleteAssemblySucceeds(t *testing.T) {
	cmd, err := distribution.NewAddCommand(realCloner(), realValidator())
	if err != nil {
		t.Fatalf("NewAddCommand with a real cloner and validator returned %v, want an assembled command", err)
	}
	if cmd == nil {
		t.Fatal("NewAddCommand returned a nil command with a nil error")
	}
	assertDependenciesStored(t, "NewAddCommand", cmd, 2)
}

// TestNewAddCommand_NilGitClonerNamesDependency covers CLM-033: the cloner is
// required even though pack add's local-path branch never clones, because an
// "optional when a flag is set" dependency is the shape DD-30 forbids.
func TestNewAddCommand_NilGitClonerNamesDependency(t *testing.T) {
	cmd, err := distribution.NewAddCommand(nil, realValidator())
	assertMissingDependency(t, "NewAddCommand(nil, validator)", cmd != nil, err, "AddCommand", "git cloner")
}

// TestNewAddCommand_NilValidatorNamesDependency covers CLM-034. The validator is
// nilled with a REAL cloner present so the constructor cannot pass by naming the
// first dependency it happens to check.
func TestNewAddCommand_NilValidatorNamesDependency(t *testing.T) {
	cmd, err := distribution.NewAddCommand(realCloner(), nil)
	assertMissingDependency(t, "NewAddCommand(cloner, nil)", cmd != nil, err, "AddCommand", "validator")
}

// TestNewAddCommand_AllDependenciesNilReturnsError covers CLM-035: with nothing
// supplied the constructor must still refuse outright rather than hand back a
// half-built command alongside an error.
func TestNewAddCommand_AllDependenciesNilReturnsError(t *testing.T) {
	cmd, err := distribution.NewAddCommand(nil, nil)
	if err == nil {
		t.Fatal("NewAddCommand(nil, nil) returned no error")
	}
	if cmd != nil {
		t.Errorf("NewAddCommand(nil, nil) returned %#v alongside its error; a wholly unassembled command must never escape the constructor", cmd)
	}

	var missing *distribution.MissingDependencyError
	if !errors.As(err, &missing) {
		t.Fatalf("NewAddCommand(nil, nil) returned %T (%v), want a *distribution.MissingDependencyError", err, err)
	}
	if missing.Dependency == "" {
		t.Error("the error names no dependency, so it does not tell the operator what to supply")
	}
}

// --- InstallCommand (CLM-036/037/048) ---

// TestNewInstallCommand_CompleteAssemblySucceeds covers CLM-036: install's ONLY
// dependency is the cloner.
func TestNewInstallCommand_CompleteAssemblySucceeds(t *testing.T) {
	cmd, err := distribution.NewInstallCommand(realCloner())
	if err != nil {
		t.Fatalf("NewInstallCommand with a real cloner returned %v, want an assembled command", err)
	}
	if cmd == nil {
		t.Fatal("NewInstallCommand returned a nil command with a nil error")
	}
	assertDependenciesStored(t, "NewInstallCommand", cmd, 1)
}

// TestNewInstallCommand_NilGitClonerNamesDependency covers CLM-037. The cloner is
// required even for `pack install --cache`, which never clones: NewExecGitCloner
// probes nothing, so requiring it costs an airgapped consumer nothing.
func TestNewInstallCommand_NilGitClonerNamesDependency(t *testing.T) {
	cmd, err := distribution.NewInstallCommand(nil)
	assertMissingDependency(t, "NewInstallCommand(nil)", cmd != nil, err, "InstallCommand", "git cloner")
}

// TestNewInstallCommand_RequiresNoValidator covers CLM-048: install is the
// hash-verified RESTORE path and does not re-validate (DD-12), so a validator is
// not merely optional — it is absent from the constructor's signature and from
// the command entirely.
//
// This is asserted structurally rather than by a successful call, because a
// constructor that accepted a validator and defaulted it would still construct.
func TestNewInstallCommand_RequiresNoValidator(t *testing.T) {
	validatorType := reflect.TypeOf((*distribution.Validator)(nil)).Elem()

	ctor := reflect.TypeOf(distribution.NewInstallCommand)
	if got := ctor.NumIn(); got != 1 {
		t.Fatalf("NewInstallCommand takes %d arguments, want exactly 1 (the cloner); install does not re-validate", got)
	}
	if in := ctor.In(0); in == validatorType {
		t.Errorf("NewInstallCommand's only argument is a %s; install must not take a validator", in)
	}

	cmd, err := distribution.NewInstallCommand(realCloner())
	if err != nil {
		t.Fatalf("NewInstallCommand(cloner) returned %v; a validator-free assembly must succeed", err)
	}

	s := reflect.ValueOf(cmd).Elem()
	for i := range s.NumField() {
		if s.Field(i).Type() == validatorType {
			t.Errorf("InstallCommand carries a %s field %q; install verifies content hashes and does not re-validate", validatorType, s.Type().Field(i).Name)
		}
	}
}

// --- UpdateCommand (CLM-038/039/040/041) ---

// TestNewUpdateCommand_CompleteAssemblySucceeds covers CLM-038: update resolves a
// compatible version, clones, and validates, so it requires all three.
func TestNewUpdateCommand_CompleteAssemblySucceeds(t *testing.T) {
	cmd, err := distribution.NewUpdateCommand(realCloner(), realValidator(), realResolver())
	if err != nil {
		t.Fatalf("NewUpdateCommand with all three dependencies returned %v, want an assembled command", err)
	}
	if cmd == nil {
		t.Fatal("NewUpdateCommand returned a nil command with a nil error")
	}
	assertDependenciesStored(t, "NewUpdateCommand", cmd, 3)
}

// TestNewUpdateCommand_NilGitClonerNamesDependency covers CLM-039.
func TestNewUpdateCommand_NilGitClonerNamesDependency(t *testing.T) {
	cmd, err := distribution.NewUpdateCommand(nil, realValidator(), realResolver())
	assertMissingDependency(t, "NewUpdateCommand(nil, validator, resolver)", cmd != nil, err, "UpdateCommand", "git cloner")
}

// TestNewUpdateCommand_NilValidatorNamesDependency covers CLM-040.
func TestNewUpdateCommand_NilValidatorNamesDependency(t *testing.T) {
	cmd, err := distribution.NewUpdateCommand(realCloner(), nil, realResolver())
	assertMissingDependency(t, "NewUpdateCommand(cloner, nil, resolver)", cmd != nil, err, "UpdateCommand", "validator")
}

// TestNewUpdateCommand_NilVersionResolverNamesDependency covers CLM-041. This
// replaces update.go's runtime "version resolver required for update" check,
// which was a fail-closed guard on a field that was structurally optional.
func TestNewUpdateCommand_NilVersionResolverNamesDependency(t *testing.T) {
	cmd, err := distribution.NewUpdateCommand(realCloner(), realValidator(), nil)
	assertMissingDependency(t, "NewUpdateCommand(cloner, validator, nil)", cmd != nil, err, "UpdateCommand", "version resolver")
}

// --- UpgradeCommand (CLM-042/043/044/045/046) ---

// TestNewUpgradeCommand_CompleteAssemblySucceeds covers CLM-042: upgrade clones,
// validates, scans the consumer codebase, and generates remediation artifacts.
func TestNewUpgradeCommand_CompleteAssemblySucceeds(t *testing.T) {
	cmd, err := distribution.NewUpgradeCommand(realCloner(), realValidator(), realScanner(), realRemediation())
	if err != nil {
		t.Fatalf("NewUpgradeCommand with all four dependencies returned %v, want an assembled command", err)
	}
	if cmd == nil {
		t.Fatal("NewUpgradeCommand returned a nil command with a nil error")
	}
	assertDependenciesStored(t, "NewUpgradeCommand", cmd, 4)
}

// TestNewUpgradeCommand_NilGitClonerNamesDependency covers CLM-043.
func TestNewUpgradeCommand_NilGitClonerNamesDependency(t *testing.T) {
	cmd, err := distribution.NewUpgradeCommand(nil, realValidator(), realScanner(), realRemediation())
	assertMissingDependency(t, "NewUpgradeCommand(nil, validator, scanner, remediation)", cmd != nil, err, "UpgradeCommand", "git cloner")
}

// TestNewUpgradeCommand_NilValidatorNamesDependency covers CLM-044.
func TestNewUpgradeCommand_NilValidatorNamesDependency(t *testing.T) {
	cmd, err := distribution.NewUpgradeCommand(realCloner(), nil, realScanner(), realRemediation())
	assertMissingDependency(t, "NewUpgradeCommand(cloner, nil, scanner, remediation)", cmd != nil, err, "UpgradeCommand", "validator")
}

// TestNewUpgradeCommand_NilScannerNamesDependency covers CLM-045. The scanner sits
// in the third position with real dependencies on either side, so a constructor
// checking only its first or last argument fails this cell.
func TestNewUpgradeCommand_NilScannerNamesDependency(t *testing.T) {
	cmd, err := distribution.NewUpgradeCommand(realCloner(), realValidator(), nil, realRemediation())
	assertMissingDependency(t, "NewUpgradeCommand(cloner, validator, nil, remediation)", cmd != nil, err, "UpgradeCommand", "scanner")
}

// TestNewUpgradeCommand_NilRemediationGeneratorNamesDependency covers CLM-046.
func TestNewUpgradeCommand_NilRemediationGeneratorNamesDependency(t *testing.T) {
	cmd, err := distribution.NewUpgradeCommand(realCloner(), realValidator(), realScanner(), nil)
	assertMissingDependency(t, "NewUpgradeCommand(cloner, validator, scanner, nil)", cmd != nil, err, "UpgradeCommand", "remediation generator")
}

// --- Run reads the RECEIVER's dependencies ---
//
// These tests once handed Run an options value carrying SABOTAGED dependencies
// and asserted the pipeline succeeded anyway, which was the only way to tell
// whether the transitional delegation had copied the receiver's dependencies
// across. The options no longer HAVE dependency fields, so that sabotage is
// unrepresentable and the delegation it guarded is gone.
//
// What each test asserts is unchanged and still falsifiable: the assembled
// dependency is the one that DROVE the outcome — the receiver's resolver decides
// which version the update lands on, the receiver's scanner decides how many
// violations are baselined. A Run reading dependencies from anywhere else
// produces different values here.

// TestAddCommandRun_UsesReceiverDependencies proves AddCommand.Run runs the add
// pipeline on the cloner and validator the command was ASSEMBLED with.
func TestAddCommandRun_UsesReceiverDependencies(t *testing.T) {
	projectDir := setupAddProject(t)

	cmd, err := distribution.NewAddCommand(
		&mockGitCloner{cloneDir: filepath.Join("testdata", "valid-pack")},
		&mockValidator{},
	)
	if err != nil {
		t.Fatalf("NewAddCommand: %v", err)
	}

	opts := distribution.AddOptions{
		ProjectDir: projectDir,
		Version:    "1.0.0",
	}

	result, err := cmd.Run("acme/valid-pack@1.0.0", opts)
	if err != nil {
		t.Fatalf("AddCommand.Run: %v — the pipeline did not run on the command's own dependencies", err)
	}
	if result.PackName != "acme/valid-pack" {
		t.Errorf("PackName = %q, want %q", result.PackName, "acme/valid-pack")
	}
}

// TestInstallCommandRun_UsesReceiverCloner proves InstallCommand.Run restores
// with the command's cloner.
func TestInstallCommandRun_UsesReceiverCloner(t *testing.T) {
	projectDir := setupInstallProject(t)

	sourceDir := t.TempDir()
	writeFile(t, filepath.Join(sourceDir, "pack.yml"), "name: acme/valid-pack\nversion: \"1.0.0\"\n")

	cmd, err := distribution.NewInstallCommand(&mockGitCloner{cloneDir: sourceDir})
	if err != nil {
		t.Fatalf("NewInstallCommand: %v", err)
	}

	opts := distribution.InstallOptions{
		ProjectDir: projectDir,
	}

	result, err := cmd.Run(opts)
	if err != nil {
		t.Fatalf("InstallCommand.Run: %v — the pipeline did not run on the command's own cloner", err)
	}
	if len(result.InstalledPacks) == 0 {
		t.Error("InstallCommand.Run installed nothing; the restore did not run")
	}
}

// TestUpdateCommandRun_UsesReceiverDependencies proves UpdateCommand.Run resolves
// and validates with all three of the command's dependencies — the resolver in
// particular, which decides WHICH version the update lands on.
func TestUpdateCommandRun_UsesReceiverDependencies(t *testing.T) {
	projectDir := setupUpdateProject(t)

	cmd, err := distribution.NewUpdateCommand(
		&mockGitCloner{cloneDir: filepath.Join("testdata", "valid-pack-v2")},
		&mockValidator{},
		&mockVersionResolver{latestMinor: "1.1.0"},
	)
	if err != nil {
		t.Fatalf("NewUpdateCommand: %v", err)
	}

	opts := distribution.UpdateOptions{
		ProjectDir: projectDir,
	}

	result, err := cmd.Run("acme/valid-pack", opts)
	if err != nil {
		t.Fatalf("UpdateCommand.Run: %v — the pipeline did not run on the command's own dependencies", err)
	}
	if result.NewVersion != "1.1.0" {
		t.Errorf("NewVersion = %q, want %q — the command's resolver did not drive the resolution", result.NewVersion, "1.1.0")
	}
}

// TestUpgradeCommandRun_UsesReceiverDependencies proves UpgradeCommand.Run runs on
// all four of the command's dependencies, including the scanner and remediation
// generator that a nil-skipping pipeline used to silently omit.
func TestUpgradeCommandRun_UsesReceiverDependencies(t *testing.T) {
	projectDir := setupUpgradeProject(t)

	cmd, err := distribution.NewUpgradeCommand(
		&mockGitCloner{cloneDir: filepath.Join("testdata", "valid-pack-v3")},
		&mockValidator{},
		&mockScanner{violations: []string{"violation-1"}},
		&mockRemediationGenerator{},
	)
	if err != nil {
		t.Fatalf("NewUpgradeCommand: %v", err)
	}

	opts := distribution.UpgradeOptions{
		ProjectDir: projectDir,
	}

	result, err := cmd.Run("acme/valid-pack@2.0.0", opts)
	if err != nil {
		t.Fatalf("UpgradeCommand.Run: %v — the pipeline did not run on the command's own dependencies", err)
	}
	if result.NewVersion != "2.0.0" {
		t.Errorf("NewVersion = %q, want %q", result.NewVersion, "2.0.0")
	}
	if result.BaselinedViolations != 1 {
		t.Errorf("BaselinedViolations = %d, want 1 — the command's scanner did not drive the scan", result.BaselinedViolations)
	}
}
