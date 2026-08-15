package main

import (
	"strings"
	"testing"
)

// REQ-003: the four setup checks, one condition each.
//
// Every test here drives a REAL fixture project staged by the harness — never a
// hand-built doctorContext — because the property under test is that doctor's context
// GATHERING survives each broken project. A stubbed context proves nothing about the
// gathering that would abort.
//
// THE SHAPE TO REJECT, stated once so no test enshrines it: on a directory with no
// backstop.yml, exactly ONE check FAILS (config-present). config-loads,
// packs-installed and toolchain-runs are SKIPPED, each naming its owner. Four failures
// for one cause would also make the exit comparison in doctor_exit_test.go meaningless.

// TestDoctorConfigPresent_AbsentConfigFailsAsCheckResult (CLM-015).
//
// The load-bearing half is the last one: doctor still RAN. A refuse-to-start would pass
// a naive "did it say no backstop.yml" assertion.
func TestDoctorConfigPresent_AbsentConfigFailsAsCheckResult(t *testing.T) {
	project := stageDoctorProject(t, "no-config")
	payload, _ := runDoctorJSON(t, project)

	if status := payload.statuses()[doctorCheckConfigPresent]; status != "fail" {
		t.Errorf("config-present status = %q, want fail", status)
	}
	message := payload.field(t, doctorCheckConfigPresent, "message")
	if !strings.Contains(message, project) {
		t.Errorf("config-present message does not name the directory it searched from (%s): %q", project, message)
	}
	if payload.field(t, doctorCheckConfigPresent, "remediation") == "" {
		t.Errorf("config-present carries no remediation")
	}

	for _, entry := range doctorRegistry() {
		if _, ok := payload.statuses()[entry.ID]; !ok {
			t.Errorf("check %q produced no result — doctor refused to run the full registry", entry.ID)
		}
	}
}

// TestDoctorConfigLoads_SkippedWhenNoConfigFound (CLM-052): the absent-config condition
// is reported ONCE, by the check that owns it.
func TestDoctorConfigLoads_SkippedWhenNoConfigFound(t *testing.T) {
	project := stageDoctorProject(t, "no-config")
	payload, _ := runDoctorJSON(t, project)

	if status := payload.statuses()[doctorCheckConfigLoads]; status != "skipped" {
		t.Errorf("config-loads status = %q, want skipped", status)
	}
	if message := payload.field(t, doctorCheckConfigLoads, "message"); !strings.Contains(message, doctorCheckConfigPresent) {
		t.Errorf("config-loads skip does not name %q as the owning check: %q", doctorCheckConfigPresent, message)
	}
}

// TestDoctorConfig_PresentAndLoadableConfigPasses (CLM-053).
func TestDoctorConfig_PresentAndLoadableConfigPasses(t *testing.T) {
	project := stageDoctorProject(t, "clean")
	payload, _ := runDoctorJSON(t, project)

	statuses := payload.statuses()
	if statuses[doctorCheckConfigPresent] != "pass" {
		t.Errorf("config-present status = %q, want pass", statuses[doctorCheckConfigPresent])
	}
	if statuses[doctorCheckConfigLoads] != "pass" {
		t.Errorf("config-loads status = %q, want pass", statuses[doctorCheckConfigLoads])
	}
}

// TestDoctorPacks_DeclaredPackMissingFromDiskFails (CLM-017).
func TestDoctorPacks_DeclaredPackMissingFromDiskFails(t *testing.T) {
	project := stageDoctorProject(t, "config-missing-pack")
	payload, _ := runDoctorJSON(t, project)

	if status := payload.statuses()[doctorCheckPacksInstalled]; status != "fail" {
		t.Errorf("packs-installed status = %q, want fail", status)
	}
	if payload.field(t, doctorCheckPacksInstalled, "remediation") == "" {
		t.Errorf("packs-installed carries no remediation on a missing pack")
	}
}

// TestDoctorPacks_NoDeclaredPacksWarnsRatherThanFailingOrPassing (CLM-054).
//
// "backstop enforces nothing" is UN-ADOPTED CAPABILITY: loud and non-blocking. The
// assertion is explicit about it being neither pass nor fail.
func TestDoctorPacks_NoDeclaredPacksWarnsRatherThanFailingOrPassing(t *testing.T) {
	project := stageDoctorProject(t, "config-no-packs")
	payload, _ := runDoctorJSON(t, project)

	status := payload.statuses()[doctorCheckPacksInstalled]
	if status == "pass" {
		t.Errorf("packs-installed PASSED on a config declaring no packs — a silent pass is what this claim forbids")
	}
	if status == "fail" {
		t.Errorf("packs-installed FAILED on a config declaring no packs — un-adopted capability is not a broken promise")
	}
	if status != "warn" {
		t.Errorf("packs-installed status = %q, want warn", status)
	}
}

// TestDoctorPacks_SkippedWhenConfigAbsentOrUnloadable (CLM-055), on BOTH projects: a
// config problem must never be re-reported under a pack heading.
func TestDoctorPacks_SkippedWhenConfigAbsentOrUnloadable(t *testing.T) {
	for _, template := range []string{"no-config", "config-malformed"} {
		t.Run(template, func(t *testing.T) {
			project := stageDoctorProject(t, template)
			payload, _ := runDoctorJSON(t, project)

			if status := payload.statuses()[doctorCheckPacksInstalled]; status != "skipped" {
				t.Errorf("packs-installed status = %q on %s, want skipped", status, template)
			}
			message := payload.field(t, doctorCheckPacksInstalled, "message")
			if !strings.Contains(message, doctorCheckConfigPresent) && !strings.Contains(message, doctorCheckConfigLoads) {
				t.Errorf("packs-installed skip on %s names no owning check: %q", template, message)
			}
		})
	}
}

// TestDoctorPacks_AllDeclaredPacksPresentPassesNamingTheCount (CLM-064).
func TestDoctorPacks_AllDeclaredPacksPresentPassesNamingTheCount(t *testing.T) {
	project := stageDoctorProject(t, "clean")
	payload, _ := runDoctorJSON(t, project)

	if status := payload.statuses()[doctorCheckPacksInstalled]; status != "pass" {
		t.Errorf("packs-installed status = %q, want pass", status)
	}
	if message := payload.field(t, doctorCheckPacksInstalled, "message"); !strings.Contains(message, "1") {
		t.Errorf("packs-installed message does not name the count: %q", message)
	}
}

// TestDoctorGit_NonRepositoryWarnsNamingWhatDegrades (CLM-018).
//
// The NAMING is the claim, so the message content is asserted, not just the status. Per
// the spec's sharp edge this test asserts NO exit code: a non-repo directory alone must
// never be the reason doctor exits 1.
func TestDoctorGit_NonRepositoryWarnsNamingWhatDegrades(t *testing.T) {
	project := stageDoctorProject(t, "clean")
	if isGitWorkTree(project) {
		t.Fatalf("staged project %s is inside a git work tree; the non-repo case cannot be exercised", project)
	}
	payload, _ := runDoctorJSON(t, project)

	if status := payload.statuses()[doctorCheckGitRepository]; status != "warn" {
		t.Errorf("git-repository status = %q, want warn", status)
	}
	message := payload.field(t, doctorCheckGitRepository, "message") + " " +
		payload.field(t, doctorCheckGitRepository, "remediation")
	if !strings.Contains(message, "full sweep") {
		t.Errorf("git-repository does not name the diff-scope fallback to a full sweep: %q", message)
	}
	if !strings.Contains(message, "artifact new") {
		t.Errorf("git-repository does not name the loss of `artifact new` id reservation: %q", message)
	}
}

// TestDoctorGit_RepositoryPasses (CLM-056).
func TestDoctorGit_RepositoryPasses(t *testing.T) {
	project := stageDoctorProject(t, "clean")
	gitInitProject(t, project)
	payload, _ := runDoctorJSON(t, project)

	if status := payload.statuses()[doctorCheckGitRepository]; status != "pass" {
		t.Errorf("git-repository status = %q, want pass", status)
	}
}
