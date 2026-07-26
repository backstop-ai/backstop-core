package distribution_test

import (
	"path/filepath"
	"testing"

	"github.com/bmanson/backstop-core/pkg/pack/distribution"
)

// command_helpers_test.go assembles the four lifecycle commands the distribution
// suites drive.
//
// The mocks these builders receive are unchanged; only their DELIVERY moved. A
// dependency used to reach the pipeline as a field on the options value, where a
// test could omit it and ride a nil-skip; it now reaches the pipeline through the
// constructor, which fails closed. Each builder t.Fatal-s on assembly failure so
// a test that mis-wires itself says so at the wiring site rather than failing
// later on a nil result.

// defaultTestPackCloner is the cloner the suites use when a test does not care
// what is cloned: it copies the valid-pack fixture. Tests exercising clone
// failure, a specific fixture, or a missing tag pass their own.
func defaultTestPackCloner() distribution.GitCloner {
	return &mockGitCloner{cloneDir: filepath.Join("testdata", "valid-pack")}
}

func newTestAddCommand(t *testing.T, git distribution.GitCloner, validator distribution.Validator) *distribution.AddCommand {
	t.Helper()

	cmd, err := distribution.NewAddCommand(git, validator)
	if err != nil {
		t.Fatalf("assembling the add command under test: %v", err)
	}
	return cmd
}

// newTestInstallCommand takes a cloner and no validator: install RESTORES what
// the lock already records and deliberately does not re-validate (DD-12).
//
// The cloner is required even by the --cache and local-pack paths, which never
// clone. That is DD-30's rule rather than an oversight — a dependency that is
// "optional when a flag is set" is the exact shape that let a nil reach a live
// path — and it costs those tests nothing, because assembling a cloner probes
// nothing.
func newTestInstallCommand(t *testing.T, git distribution.GitCloner) *distribution.InstallCommand {
	t.Helper()

	cmd, err := distribution.NewInstallCommand(git)
	if err != nil {
		t.Fatalf("assembling the install command under test: %v", err)
	}
	return cmd
}

func newTestUpdateCommand(t *testing.T, git distribution.GitCloner, validator distribution.Validator, resolver distribution.VersionResolver) *distribution.UpdateCommand {
	t.Helper()

	cmd, err := distribution.NewUpdateCommand(git, validator, resolver)
	if err != nil {
		t.Fatalf("assembling the update command under test: %v", err)
	}
	return cmd
}

func newTestUpgradeCommand(t *testing.T, git distribution.GitCloner, validator distribution.Validator, scanner distribution.Scanner, remediation distribution.RemediationGenerator) *distribution.UpgradeCommand {
	t.Helper()

	cmd, err := distribution.NewUpgradeCommand(git, validator, scanner, remediation)
	if err != nil {
		t.Fatalf("assembling the upgrade command under test: %v", err)
	}
	return cmd
}
