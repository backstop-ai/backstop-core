package distribution_test

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/pack/distribution"
)

// upgrade_capability_test.go covers the upgrade pipeline's obligations toward
// its scan and remediation capabilities (REQ-009).
//
// These are DISTRIBUTION-level claims driven by TEST-DECLARED doubles: the
// production unavailable-capability implementations live in package main, which
// this package cannot import. The production-wiring proof is CLM-091/CLM-060 on
// the cmd/backstop side; what is proven here is the pipeline's own behavior —
// that it always asks, that it aborts when the answer is a failure, and that it
// asks before it has changed anything.

// recordingScanner counts its invocations, because "the scan reported no
// violations" and "the scan never ran" are indistinguishable from the result
// alone — and the second was the vacuous success this phase exists to close.
type recordingScanner struct {
	calls      int
	violations []string
	err        error
}

func (s *recordingScanner) ScanViolations(_, _ string) ([]string, error) {
	s.calls++
	return s.violations, s.err
}

// recordingRemediationGenerator counts its invocations and can fail on demand.
type recordingRemediationGenerator struct {
	calls   int
	bundle  string
	failErr error
}

func (g *recordingRemediationGenerator) GenerateBundle(_ string, _ []string) (string, error) {
	g.calls++
	if g.failErr != nil {
		return "", fmt.Errorf("generating remediation bundle: %w", g.failErr)
	}
	return g.bundle, nil
}

// TestUpgradeCommand_ScannerInvokedUnconditionally asserts the scanner runs even
// when it has nothing to report (CLM-056).
func TestUpgradeCommand_ScannerInvokedUnconditionally(t *testing.T) {
	projectDir := setupUpgradeProject(t)

	scanner := &recordingScanner{}
	generator := &recordingRemediationGenerator{bundle: "remediation-bundle.md"}

	upgrade := newTestUpgradeCommand(t,
		&mockGitCloner{cloneDir: filepath.Join("testdata", "valid-pack-v3")},
		&mockValidator{},
		scanner,
		generator,
	)

	result, err := upgrade.Run("acme/valid-pack@2.0.0", distribution.UpgradeOptions{ProjectDir: projectDir})
	if err != nil {
		t.Fatalf("upgrade whose scan reports nothing: %v", err)
	}

	if scanner.calls != 1 {
		t.Errorf("scanner invoked %d times, want 1; a zero-violation report must come from an actual scan, not from a skip", scanner.calls)
	}
	if result.BaselinedViolations != 0 {
		t.Errorf("BaselinedViolations = %d, want 0", result.BaselinedViolations)
	}
	if generator.calls != 0 {
		t.Errorf("remediation generator invoked %d times with no violations to remediate, want 0", generator.calls)
	}
}

// TestUpgradeCommand_RemediationFailureAbortsUpgrade asserts the remediation
// generator runs whenever the scan reports violations, and that a generation
// failure aborts the upgrade rather than being absorbed (CLM-057).
func TestUpgradeCommand_RemediationFailureAbortsUpgrade(t *testing.T) {
	projectDir := setupUpgradeProject(t)

	generator := &recordingRemediationGenerator{failErr: fmt.Errorf("bundle writer refused")}

	upgrade := newTestUpgradeCommand(t,
		&mockGitCloner{cloneDir: filepath.Join("testdata", "valid-pack-v3")},
		&mockValidator{},
		&recordingScanner{violations: []string{"v1", "v2"}},
		generator,
	)

	_, err := upgrade.Run("acme/valid-pack@2.0.0", distribution.UpgradeOptions{ProjectDir: projectDir})
	if err == nil {
		t.Fatal("upgrade must fail when remediation generation fails")
	}
	if !strings.Contains(err.Error(), "bundle writer refused") {
		t.Errorf("error = %v, want it to carry the generator's own diagnostic", err)
	}

	if generator.calls != 1 {
		t.Errorf("remediation generator invoked %d times for a scan reporting 2 violations, want 1", generator.calls)
	}

	manifest := string(mustReadFile(t, filepath.Join(projectDir, "backstop.yml")))
	if !strings.Contains(manifest, "1.0.0") {
		t.Errorf("backstop.yml no longer pins the old version after an aborted upgrade:\n%s", manifest)
	}
}

// TestUpgradeCommand_PropagatesCapabilityUnavailableError asserts a scanner's
// *CapabilityUnavailableError reaches the caller intact (CLM-058).
//
// Both halves matter: errors.As must recover the typed error, and its Capability
// and Reference must still name the capability and the requirement tracking it.
// An error flattened into opaque text still fails an upgrade, but it fails one
// telling an operator nothing about what is missing or when it is coming.
func TestUpgradeCommand_PropagatesCapabilityUnavailableError(t *testing.T) {
	projectDir := setupUpgradeProject(t)

	unavailable := &distribution.CapabilityUnavailableError{
		Capability: "pack upgrade violation scanning",
		Reference:  "BUNDLE-006 REQ-014",
	}

	upgrade := newTestUpgradeCommand(t,
		&mockGitCloner{cloneDir: filepath.Join("testdata", "valid-pack-v3")},
		&mockValidator{},
		&recordingScanner{err: unavailable},
		&recordingRemediationGenerator{},
	)

	_, err := upgrade.Run("acme/valid-pack@2.0.0", distribution.UpgradeOptions{ProjectDir: projectDir})
	if err == nil {
		t.Fatal("upgrade must fail when its scan capability is unavailable")
	}

	var recovered *distribution.CapabilityUnavailableError
	if !errors.As(err, &recovered) {
		t.Fatalf("error %v does not carry a *CapabilityUnavailableError; the typed error was flattened on its way out", err)
	}
	if recovered.Capability != unavailable.Capability {
		t.Errorf("Capability = %q, want %q", recovered.Capability, unavailable.Capability)
	}
	if recovered.Reference != unavailable.Reference {
		t.Errorf("Reference = %q, want %q", recovered.Reference, unavailable.Reference)
	}
}

// TestUpgradeCommand_UnavailableCapabilityFailsBeforeMutation asserts the
// failure lands before ANY consumer state has changed (CLM-059).
//
// This is the assertion that fails if the violation scan and the tool-config
// merge are swapped back: the merge writes .golangci.yml into the project, so a
// scan running after it would leave that file behind on a failure. All five
// artifacts are snapshotted rather than just the obvious one, because a pipeline
// that mutates four of them and rolls back the fifth would otherwise pass.
func TestUpgradeCommand_UnavailableCapabilityFailsBeforeMutation(t *testing.T) {
	projectDir := setupUpgradeProject(t)

	before := snapshotUpgradeArtifacts(t, projectDir)

	scanner := &recordingScanner{err: &distribution.CapabilityUnavailableError{
		Capability: "pack upgrade violation scanning",
		Reference:  "BUNDLE-006 REQ-014",
	}}
	upgrade := newTestUpgradeCommand(t,
		&mockGitCloner{cloneDir: filepath.Join("testdata", "valid-pack-v3")},
		&mockValidator{},
		scanner,
		&recordingRemediationGenerator{},
	)

	if _, err := upgrade.Run("acme/valid-pack@2.0.0", distribution.UpgradeOptions{ProjectDir: projectDir}); err == nil {
		t.Fatal("upgrade must fail when its scan capability is unavailable")
	}

	// THE SCAN MUST HAVE BEEN REACHED, and this assertion is why the test still means
	// something (SPEC-056 edit-set F).
	//
	// Everything below asserts only that Run errored and that five artifacts are
	// unchanged. A *VersionMismatchError from SPEC-056's identity gate satisfies BOTH —
	// it errors, and it refuses before any mutation — so when the gate landed, this test
	// went on PASSING while the property it exists for (the scanner is reached, and its
	// UNAVAILABILITY is what fails the upgrade) silently stopped being tested. Nothing
	// turned red to prompt the fix.
	//
	// The double has always counted its calls; nothing read the count. Reading it is what
	// stops the same vacuity returning the next time anything is inserted ahead of the
	// scan.
	if scanner.calls == 0 {
		t.Fatal("the scanner was never invoked, so this test is not proving what it claims: something now refuses the upgrade BEFORE the scan, and the capability failure is no longer what fails it")
	}

	after := snapshotUpgradeArtifacts(t, projectDir)

	for _, artifact := range upgradeMutableArtifacts() {
		if before[artifact.name] != after[artifact.name] {
			t.Errorf("%s changed despite the upgrade failing on an unavailable capability;\nbefore: %s\nafter:  %s",
				artifact.name, before[artifact.name], after[artifact.name])
		}
	}
}

// upgradeMutableArtifacts names the five pieces of consumer state a completed
// upgrade writes, each relative to the project directory.
func upgradeMutableArtifacts() []struct{ name, relPath string } {
	return []struct{ name, relPath string }{
		{"tool config (.golangci.yml)", ".golangci.yml"},
		{"provenance", filepath.Join(".backstop", "pack-config-provenance.json")},
		{"installed content", filepath.Join(".backstop", "packs", "acme", "valid-pack", "pack.yml")},
		{"backstop.yml", "backstop.yml"},
		{"backstop.lock", "backstop.lock"},
	}
}

// snapshotUpgradeArtifacts records the content of each artifact, representing an
// absent file distinctly from an empty one so a file that appears where none
// existed registers as a change.
func snapshotUpgradeArtifacts(t *testing.T, projectDir string) map[string]string {
	t.Helper()

	snapshot := map[string]string{}
	for _, artifact := range upgradeMutableArtifacts() {
		data, err := os.ReadFile(filepath.Join(projectDir, artifact.relPath))
		switch {
		case err == nil:
			snapshot[artifact.name] = string(data)
		case os.IsNotExist(err):
			snapshot[artifact.name] = "<absent>"
		default:
			t.Fatalf("snapshotting %s: %v", artifact.name, err)
		}
	}
	return snapshot
}
