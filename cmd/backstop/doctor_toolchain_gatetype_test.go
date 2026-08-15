package main

import (
	"slices"
	"testing"
)

// REQ-006: the gate_type matrix — two executed, five not.
//
// ★ EXECUTION AND NON-EXECUTION ARE READ OFF THE FILESYSTEM, NEVER OFF THE
// IMPLEMENTATION. Each of the fixture pack's seven bindings writes a DISTINCT marker file
// when it runs, so "was executed" is a file existing and "was not executed" is a file the
// command WOULD have created being absent. Asserting the implementation's selection list,
// or a recorded command log, tests the implementation against itself.

// runToolchainMatrix stages the seven-gate_type fixture, runs the toolchain check over
// it, and returns the marker files that exist afterwards.
func runToolchainMatrix(t *testing.T) []string {
	t.Helper()

	project := stageDoctorProject(t, "toolchain-matrix")
	payload, _ := runDoctorJSON(t, project, "--check", doctorCheckToolchainRuns)
	if status := payload.statuses()[doctorCheckToolchainRuns]; status == "skipped" {
		t.Fatalf("toolchain-runs was SKIPPED on the matrix project, so nothing ran: %s",
			payload.field(t, doctorCheckToolchainRuns, "message"))
	}
	return markerFiles(t, project)
}

// TestDoctorToolchain_ExecutesTestGateTypeEntrypoint (CLM-024).
func TestDoctorToolchain_ExecutesTestGateTypeEntrypoint(t *testing.T) {
	markers := runToolchainMatrix(t)
	if !slices.Contains(markers, "test.marker") {
		t.Errorf("the gate_type: test entrypoint did not run; markers on disk: %v", markers)
	}
}

// TestDoctorToolchain_ExecutesBuildGateTypeEntrypoint (CLM-025).
func TestDoctorToolchain_ExecutesBuildGateTypeEntrypoint(t *testing.T) {
	markers := runToolchainMatrix(t)
	if !slices.Contains(markers, "build.marker") {
		t.Errorf("the gate_type: build entrypoint did not run; markers on disk: %v", markers)
	}
}

// assertNotExecuted fails when the named marker exists, and Fatals when NO marker exists
// at all.
//
// THE SECOND GUARD IS WHAT KEEPS THESE FIVE CLAIMS HONEST. If nothing ran — a pack parked
// where loadInstalledPacks cannot reach it, a fixture module that will not build, a
// skipped check — every absence assertion passes while the mechanism under test never
// executed. Requiring the two EXPECTED markers to be present turns that vacuous green
// into a failure.
func assertNotExecuted(t *testing.T, marker string) {
	t.Helper()

	markers := runToolchainMatrix(t)
	for _, expected := range []string{"test.marker", "build.marker"} {
		if !slices.Contains(markers, expected) {
			t.Fatalf("%s is absent, so NOTHING ran and the absence of %s proves nothing; markers: %v",
				expected, marker, markers)
		}
	}
	if slices.Contains(markers, marker) {
		t.Errorf("%s exists, so the engine that writes it was executed; markers: %v", marker, markers)
	}
}

// TestDoctorToolchain_DoesNotExecuteLintEngine (CLM-026).
func TestDoctorToolchain_DoesNotExecuteLintEngine(t *testing.T) {
	assertNotExecuted(t, "lint.marker")
}

// TestDoctorToolchain_DoesNotExecuteFindingsEngine (CLM-027).
func TestDoctorToolchain_DoesNotExecuteFindingsEngine(t *testing.T) {
	assertNotExecuted(t, "findings.marker")
}

// TestDoctorToolchain_DoesNotExecuteCoverageEngine (CLM-028).
//
// ★ THIS IS THE TRAP ROW. The coverage binding's command differs from the test binding's
// ONLY in the -o name, reproducing the real typescript-toolchain shape where
// near-identical vitest commands sit under both `test` and `coverage`. An implementation
// selecting on the command STRING rather than the declared gate_type passes CLM-024 and
// reds here. Do not "clean up" the similarity.
func TestDoctorToolchain_DoesNotExecuteCoverageEngine(t *testing.T) {
	assertNotExecuted(t, "coverage.marker")
}

// TestDoctorToolchain_DoesNotExecuteSubstantivenessEngine (CLM-029).
func TestDoctorToolchain_DoesNotExecuteSubstantivenessEngine(t *testing.T) {
	assertNotExecuted(t, "subst.marker")
}

// TestDoctorToolchain_DoesNotExecuteContractsEngine (CLM-030).
func TestDoctorToolchain_DoesNotExecuteContractsEngine(t *testing.T) {
	assertNotExecuted(t, "contracts.marker")
}
