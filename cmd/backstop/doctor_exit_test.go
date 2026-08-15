package main

import (
	"go/ast"
	"strings"
	"testing"
)

// REQ-003: the exit-code matrix, and the absence of any exit-2 path.
//
// Every test reads the INTEGER exit code from the harness. None infers one from text.

// TestDoctor_ExitZeroWhenNoCheckFails (CLM-011) — the clean project.
//
// ★ CROSS-PHASE OBLIGATION. This is registry-relative: every phase that registers a new
// check owns keeping the clean fixture all-passing. The fix for a red here is always the
// FIXTURE, never a --check filter — a filtered run cannot stand in for "a run whose
// checks all pass".
func TestDoctor_ExitZeroWhenNoCheckFails(t *testing.T) {
	project := stageDoctorProject(t, "clean")
	gitInitProject(t, project)

	payload, code := runDoctorJSON(t, project)
	for id, status := range payload.statuses() {
		if status == "fail" {
			t.Errorf("check %q failed on the clean project: %s", id, payload.field(t, id, "message"))
		}
	}
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
}

// TestDoctor_ExitZeroWhenWarningsPresent (CLM-012) — warnings are loud, not blocking.
func TestDoctor_ExitZeroWhenWarningsPresent(t *testing.T) {
	project := stageDoctorProject(t, "config-no-packs")
	payload, code := runDoctorJSON(t, project)

	sawWarning := false
	for _, status := range payload.statuses() {
		if status == "warn" {
			sawWarning = true
		}
		if status == "fail" {
			t.Fatalf("the no-packs project carries a FAILURE, so this fixture cannot isolate the warning case: %v", payload.statuses())
		}
	}
	if !sawWarning {
		t.Fatalf("no check warned, so the claim never ran: %v", payload.statuses())
	}
	if code != 0 {
		t.Errorf("exit code = %d with warnings and no failure, want 0", code)
	}
}

// TestDoctor_ExitZeroWhenChecksSkipped (CLM-013).
//
// THE FIXTURE CHOICE IS DELIBERATE. The no-config project also FAILS config-present, so
// it cannot serve here. `--check config-loads` on the no-config project produces the SKIP
// (config-loads skips naming config-present as the owner) with no failing check in the
// report at all — a skip-without-failure run, which is the exact condition this claim
// names and the only one that isolates it.
func TestDoctor_ExitZeroWhenChecksSkipped(t *testing.T) {
	project := stageDoctorProject(t, "no-config")
	payload, code := runDoctorJSON(t, project, "--check", doctorCheckConfigLoads)

	if status := payload.statuses()[doctorCheckConfigLoads]; status != "skipped" {
		t.Fatalf("config-loads status = %q, want skipped", status)
	}
	if code != 0 {
		t.Errorf("exit code = %d for a run carrying a skip and no failure, want 0", code)
	}
}

// TestDoctor_ExitOneWhenAnyCheckFails (CLM-014) — the missing-pack project.
func TestDoctor_ExitOneWhenAnyCheckFails(t *testing.T) {
	project := stageDoctorProject(t, "config-missing-pack")
	payload, code := runDoctorJSON(t, project)

	if status := payload.statuses()[doctorCheckPacksInstalled]; status != "fail" {
		t.Fatalf("packs-installed status = %q, want fail — the fixture is not exercising the claim", status)
	}
	if code != ExitViolations {
		t.Errorf("exit code = %d with a failing check, want %d", code, ExitViolations)
	}
}

// TestDoctor_UnparseableConfigExitsOneWhereGateExitsTwo (CLM-016).
//
// ONE test running the SAME staged project through BOTH commands, because the claim is
// about the DIFFERENCE between them. Two separate tests would let the gate half rot
// silently while the doctor half kept passing.
func TestDoctor_UnparseableConfigExitsOneWhereGateExitsTwo(t *testing.T) {
	project := stageDoctorProject(t, "config-malformed")

	gateOut, gateErr := runGateInDir(t, project)
	if gateCode := doctorExitCode(gateErr); gateCode != ExitConfigError {
		t.Fatalf("gate exited %d on the malformed config, want %d — the fixture no longer drives the condition under test\n%s",
			gateCode, ExitConfigError, gateOut)
	}

	payload, doctorCode := runDoctorJSON(t, project)
	if doctorCode != ExitViolations {
		t.Errorf("doctor exited %d on the SAME config, want %d", doctorCode, ExitViolations)
	}
	if status := payload.statuses()[doctorCheckConfigLoads]; status != "fail" {
		t.Errorf("config-loads status = %q, want fail", status)
	}
	// The loader's OWN error text, not a doctor paraphrase: this is the very error
	// runGate wraps into its exit-2.
	message := payload.field(t, doctorCheckConfigLoads, "message")
	if !strings.Contains(message, "backstop.yml") {
		t.Errorf("config-loads message does not carry the loader's own error: %q", message)
	}
}

// TestDoctor_NoExitConfigErrorPathAndFullRegistryRunsOnAnEmptyDirectory (CLM-057) has
// THREE halves, one per conjunct of the claim.
//
// (a) and (b) cover "no doctor code path returns ExitConfigError". (c) covers the OTHER
// conjunct — "no check gathers its own input by a route that can abort before the
// registry runs" — and it is not redundant: a check that calls config.LoadConfig itself
// and converts the error into its own `fail` result returns no ExitConfigError (so (a)
// stays green) and still produces a result for every check (so (b) stays green) while
// violating the second conjunct outright. Only a source scan catches that, and it reads
// as ordinary defensive error handling in review.
func TestDoctor_NoExitConfigErrorPathAndFullRegistryRunsOnAnEmptyDirectory(t *testing.T) {
	// (a) SOURCE SCAN: no doctor non-test file names ExitConfigError.
	for _, file := range parseNonTestPackageFiles(t) {
		if !isDoctorOwnedFile(file.path) {
			continue
		}
		ast.Inspect(file.file, func(node ast.Node) bool {
			ident, ok := node.(*ast.Ident)
			if ok && ident.Name == "ExitConfigError" {
				t.Errorf("%s names ExitConfigError; doctor has no exit-2 path", file.path)
			}
			return true
		})
	}

	// (b) REAL RUN on the simultaneous no-config + no-packs + no-git project: EVERY
	// registered check produced a result.
	project := stageDoctorProject(t, "no-config")
	if isGitWorkTree(project) {
		t.Fatalf("staged project %s is inside a git work tree; the no-git half cannot be exercised", project)
	}
	payload, code := runDoctorJSON(t, project)
	if code == ExitConfigError {
		t.Errorf("doctor exited %d on an empty directory; a diagnostic that refuses to start cannot diagnose a broken setup", code)
	}
	statuses := payload.statuses()
	for _, entry := range doctorRegistry() {
		if _, ok := statuses[entry.ID]; !ok {
			t.Errorf("check %q produced no result on the empty project", entry.ID)
		}
	}

	// ONE CONDITION, ONE OWNER: exactly one FAILURE on a directory whose single defect
	// is the absent config. Four failures for one cause is the shape to reject.
	failures := []string{}
	for id, status := range statuses {
		if status == "fail" {
			failures = append(failures, id)
		}
	}
	if len(failures) != 1 || failures[0] != doctorCheckConfigPresent {
		t.Errorf("failures on the empty project = %v, want exactly [%s]", failures, doctorCheckConfigPresent)
	}

	// (c) SOURCE SCAN of doctor_checks.go SPECIFICALLY: it calls NO loader. doctor.go is
	// the file that legitimately calls all four, exactly once, before the registry runs
	// — a package-wide scan would be red on arrival.
	loaders := map[string]bool{
		"LoadConfig":         true,
		"LoadConfigFromPath": true,
		"DiscoverConfigPath": true,
		"loadInstalledPacks": true,
	}
	for _, file := range parseNonTestPackageFiles(t) {
		if !strings.HasSuffix(file.path, "doctor_checks.go") {
			continue
		}
		ast.Inspect(file.file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			switch fun := call.Fun.(type) {
			case *ast.Ident:
				if loaders[fun.Name] {
					t.Errorf("doctor_checks.go calls the loader %s — a check that gathers its own input is the second abort path REQ-003 forbids", fun.Name)
				}
			case *ast.SelectorExpr:
				if loaders[fun.Sel.Name] {
					t.Errorf("doctor_checks.go calls the loader %s — a check that gathers its own input is the second abort path REQ-003 forbids", fun.Sel.Name)
				}
			}
			return true
		})
	}
}

// isDoctorOwnedFile reports whether path is one of the non-test files this spec owns.
func isDoctorOwnedFile(path string) bool {
	return strings.HasSuffix(path, "doctor.go") || strings.HasSuffix(path, "doctor_checks.go")
}
