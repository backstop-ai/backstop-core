package main

import (
	"strings"
	"testing"
)

// doctor_engine_tools_e2e_test.go drives the REAL cobra command.
//
// The unit tests call the check function. What ISSUE-134 reports is not a function
// returning the wrong value — it is `backstop doctor` printing an all-green report and
// exiting 0 on a project the gate refuses. A fix verified only at the function level has
// not been verified against the artifact the incident named.

// TestDoctor_CommandReportsAbsentEngineToolAndExitsNonZero (CLM-001).
func TestDoctor_CommandReportsAbsentEngineToolAndExitsNonZero(t *testing.T) {
	project := stageDoctorProject(t, "engine-tool-absent")
	withBinaryResolver(t)

	// (a) THE ASSERTION THAT MOST DIRECTLY INVERTS THE OBSERVED DEFECT: exit 0 today.
	human, humanCode := runDoctorInProject(t, project, "doctor")
	if humanCode != ExitViolations {
		t.Fatalf("doctor exited %d on a project whose rule-bound engine tool is absent, want %d — exit 0 is the recorded defect\n%s", humanCode, ExitViolations, human)
	}

	// (b) The HUMAN rendering names the absent tool.
	if !strings.Contains(human, "backstop-absent-engine-134") {
		t.Errorf("the human report does not name the absent tool:\n%s", human)
	}

	// (c) The --json payload carries the failure too. Both renderers, because doctor's
	// one-struct-feeds-both property is what keeps them from disagreeing — and a JSON
	// consumer must see the failure as well.
	payload, jsonCode := runDoctorJSON(t, project)
	if jsonCode != ExitViolations {
		t.Errorf("--json exited %d, want %d", jsonCode, ExitViolations)
	}
	entry := payload.find(t, doctorCheckEngineTools)
	if status, _ := entry["status"].(string); status != doctorStatusFail {
		t.Errorf("--json engine-tools status = %q, want %q", status, doctorStatusFail)
	}
	if remediation, _ := entry["remediation"].(string); strings.TrimSpace(remediation) == "" {
		t.Errorf("--json engine-tools carries an EMPTY remediation; a failure a consumer cannot act on is half a report")
	}

	// The exit-1 must be attributable to THIS check rather than to some other failure
	// the fixture happens to trip.
	for id, status := range payload.statuses() {
		if status == doctorStatusFail && id != doctorCheckEngineTools {
			t.Errorf("check %q also failed on this fixture, so the exit code proves less than it appears to", id)
		}
	}
}

// TestDoctor_CommandCheckSelectorResolvesEngineToolsCheck (CLM-007). --check <new-id>
// returns exactly that one result rather than the unknown-selector error, which proves
// the entry is genuinely REGISTERED rather than merely rendered — and is the cheapest
// guard against an id registered under one spelling and printed under another.
func TestDoctor_CommandCheckSelectorResolvesEngineToolsCheck(t *testing.T) {
	project := stageDoctorProject(t, "engine-tool-present")
	withBinaryResolver(t, "backstop-present-engine-134")

	payload, code := runDoctorJSON(t, project, "--check", doctorCheckEngineTools)
	if code == ExitConfigError {
		t.Fatalf("--check %s exited with a config error; the id resolves to no registered entry", doctorCheckEngineTools)
	}
	got := payload.ids()
	if len(got) != 1 || got[0] != doctorCheckEngineTools {
		t.Fatalf("--check %s reported %v, want exactly [%s]", doctorCheckEngineTools, got, doctorCheckEngineTools)
	}
}
