package main

import (
	"encoding/json"
	"strings"
	"testing"
)

// REQ-001: the command surface, its two renderings, and the --check selector.

// TestDoctor_RegisteredAndDiscoverableInCommandTree asserts BOTH that doctor is on the
// root command's list AND that it appears in the `backstop commands` agent-discovery
// JSON (CLM-001).
//
// Asserting only one of the two would miss a command registered somewhere the discovery
// tree does not walk. The tree is built by buildCommandTree FROM the registered
// commands, so both hold with no discovery-specific code.
func TestDoctor_RegisteredAndDiscoverableInCommandTree(t *testing.T) {
	root := NewRootCommand()

	found, _, err := root.Find([]string{"doctor"})
	if err != nil || found.Name() != "doctor" {
		t.Fatalf("doctor is not registered on the root command: cmd=%v err=%v", found, err)
	}

	out, execErr := executeCommand(NewRootCommand(), "commands")
	if execErr != nil {
		t.Fatalf("backstop commands: %v", execErr)
	}
	var descriptors []CommandDescriptor
	start := strings.Index(out, "[")
	if start < 0 {
		t.Fatalf("commands output carries no JSON array:\n%s", out)
	}
	if unmarshalErr := json.Unmarshal([]byte(out[start:]), &descriptors); unmarshalErr != nil {
		t.Fatalf("decoding commands output: %v\n%s", unmarshalErr, out)
	}
	for _, descriptor := range descriptors {
		if descriptor.Path == "doctor" {
			return
		}
	}
	t.Fatalf("doctor is absent from the agent-discovery command tree")
}

// TestDoctor_BareRunReportsEveryRegisteredCheck compares the reported id SET against the
// ids doctorRegistry() declares (CLM-002).
//
// REGISTRY-RELATIVE BY CONSTRUCTION: it hardcodes no count and no id list, so it holds
// while the registry grows across phases. The absolute "exactly these seven" assertion
// is CLM-051's, in its own file.
func TestDoctor_BareRunReportsEveryRegisteredCheck(t *testing.T) {
	project := stageDoctorProject(t, "clean")
	payload, _ := runDoctorJSON(t, project)

	declared := make(map[string]bool)
	for _, entry := range doctorRegistry() {
		declared[entry.ID] = true
	}
	reported := make(map[string]bool)
	for _, id := range payload.ids() {
		reported[id] = true
	}

	for id := range declared {
		if !reported[id] {
			t.Errorf("registered check %q was not reported by a bare run", id)
		}
	}
	for id := range reported {
		if !declared[id] {
			t.Errorf("run reported check %q that the registry does not declare", id)
		}
	}
}

// TestDoctor_JSONPayloadCarriesSchemaVersionAndPerCheckFields asserts the doctor/v1
// declaration and the five per-check keys (CLM-003).
//
// KEY PRESENCE ON A PASSING CHECK is the load-bearing half: an empty `remediation` must
// still emit its key, which is what the dropped `omitempty` buys and the only assertion
// that catches the tag being re-added.
func TestDoctor_JSONPayloadCarriesSchemaVersionAndPerCheckFields(t *testing.T) {
	project := stageDoctorProject(t, "clean")
	gitInitProject(t, project)
	payload, _ := runDoctorJSON(t, project)

	if payload.SchemaVersion != "doctor/v1" {
		t.Errorf("schema_version = %q, want %q", payload.SchemaVersion, "doctor/v1")
	}
	if len(payload.Checks) == 0 {
		t.Fatalf("payload carries no checks")
	}

	sawPassing := false
	for _, entry := range payload.Checks {
		for _, key := range []string{"id", "title", "status", "message", "remediation"} {
			if _, ok := entry[key]; !ok {
				t.Errorf("check %v is missing the %q key", entry["id"], key)
			}
		}
		if status, _ := entry["status"].(string); status == "pass" {
			sawPassing = true
			if remediation, ok := entry["remediation"].(string); !ok || remediation != "" {
				t.Errorf("passing check %v should carry an EMPTY remediation string, got %#v", entry["id"], entry["remediation"])
			}
		}
	}
	if !sawPassing {
		t.Fatalf("no check passed on the clean project, so the empty-remediation assertion never ran")
	}
}

// TestDoctor_HumanAndJSONReportSameCheckSetAndStatuses runs the SAME project through
// both renderings and compares set and per-check status (CLM-004).
//
// Both must come from ONE []doctorResult; a test constructing its own expectation twice
// would pass over two renderers that agree only by accident.
func TestDoctor_HumanAndJSONReportSameCheckSetAndStatuses(t *testing.T) {
	project := stageDoctorProject(t, "config-missing-pack")

	payload, jsonCode := runDoctorJSON(t, project)
	human, humanCode := runDoctorInProject(t, project, "doctor")

	if jsonCode != humanCode {
		t.Errorf("exit codes differ: json=%d human=%d", jsonCode, humanCode)
	}
	for id, status := range payload.statuses() {
		if !strings.Contains(human, id) {
			t.Errorf("human rendering omits check %q\n%s", id, human)
		}
		if !humanLineCarries(human, id, status) {
			t.Errorf("human rendering does not report status %q for check %q\n%s", status, id, human)
		}
	}
}

// humanLineCarries reports whether the human line naming id also names status.
func humanLineCarries(human, id, status string) bool {
	for _, line := range strings.Split(human, "\n") {
		if strings.Contains(line, id) && strings.Contains(line, status) {
			return true
		}
	}
	return false
}

// TestDoctor_RootPositionJSONFlagRendersJSONPayload drives the flag in ROOT position:
// `backstop --json doctor` (CLM-061).
//
// This is the invocation a local BoolVar would break while every sub-command-position
// test kept passing, which is why the claim names the root position specifically.
func TestDoctor_RootPositionJSONFlagRendersJSONPayload(t *testing.T) {
	project := stageDoctorProject(t, "clean")

	stdout, _ := runDoctorInProject(t, project, "--json", "doctor")
	payload := decodeDoctorJSON(t, stdout)

	if payload.SchemaVersion != "doctor/v1" {
		t.Fatalf("root-position --json did not render the JSON payload; got:\n%s", stdout)
	}
}

// TestDoctor_CheckSelectorRunsOnlyTheNamedCheck asserts one result, and the other
// registered checks ABSENT from the payload (CLM-005).
func TestDoctor_CheckSelectorRunsOnlyTheNamedCheck(t *testing.T) {
	project := stageDoctorProject(t, "clean")
	payload, _ := runDoctorJSON(t, project, "--check", doctorCheckConfigPresent)

	if got := payload.ids(); len(got) != 1 || got[0] != doctorCheckConfigPresent {
		t.Fatalf("--check reported %v, want exactly [%s]", got, doctorCheckConfigPresent)
	}
}

// TestDoctor_UnknownCheckSelectorFailsNamingRegisteredIDs asserts a loud error naming
// the registered ids and a NON-ZERO exit (CLM-006).
//
// A diagnostic that reports nothing and calls it fine is the exact failure mode this
// claim exists for, so the empty-successful-run shape is asserted against explicitly.
func TestDoctor_UnknownCheckSelectorFailsNamingRegisteredIDs(t *testing.T) {
	project := stageDoctorProject(t, "clean")
	stdout, code := runDoctorInProject(t, project, "doctor", "--check", "no-such-check")

	if code == 0 {
		t.Errorf("unknown --check id exited 0; an empty successful run is the shape this claim forbids\n%s", stdout)
	}
	if code == ExitConfigError {
		t.Errorf("unknown --check id exited %d (config error); doctor has no exit-2 path", code)
	}
	if strings.Contains(stdout, "doctor/v1") {
		t.Errorf("unknown --check id still rendered a report:\n%s", stdout)
	}
	for _, entry := range doctorRegistry() {
		if !strings.Contains(stdout, entry.ID) {
			t.Errorf("error text does not name registered id %q\n%s", entry.ID, stdout)
		}
	}
}
