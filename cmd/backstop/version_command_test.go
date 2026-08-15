package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/schema"
)

// TestVersionCommand_ReportsContentDerivedCohort pins CLM-006. The cohort the command
// prints is schema.ComputeCohort(SchemaFS).ID, computed here independently rather than
// read back from the command's own output.
//
// The NEGATIVE half is the point (spec Review Question 4): a presence-only assertion
// passes while the old path-derived `N-schemas[…]` string survives alongside the new
// one, which would leave the guard optional. So the output must ALSO not contain
// `-schemas[`.
func TestVersionCommand_ReportsContentDerivedCohort(t *testing.T) {
	want, err := schema.ComputeCohort(SchemaFS)
	if err != nil {
		t.Fatalf("computing the expected cohort: %v", err)
	}
	if want.ID == "" {
		t.Fatal("the expected cohort ID is empty; the assertion below would be vacuous")
	}

	root := NewRootCommand()
	out, err := executeCommand(root, "version")
	if err != nil {
		t.Fatalf("version command error: %v", err)
	}

	if !strings.Contains(out, want.ID) {
		t.Errorf("version output does not carry the content-derived cohort %q:\n%s", want.ID, out)
	}
	if strings.Contains(out, "-schemas[") {
		t.Errorf("version output still carries the legacy path-derived cohort shape:\n%s", out)
	}

	// The JSON rendering must agree — the two cannot be allowed to report different
	// cohorts, which is what the resolve-once-share-both shape exists to guarantee.
	jsonRoot := NewRootCommand()
	jsonOut, err := executeCommand(jsonRoot, "version", "--json")
	if err != nil {
		t.Fatalf("version --json error: %v", err)
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(jsonOut), &parsed); err != nil {
		t.Fatalf("version --json output is not valid JSON: %v\noutput: %s", err, jsonOut)
	}
	if got, _ := parsed["schema_cohort"].(string); got != want.ID {
		t.Errorf("version --json schema_cohort = %q, want %q", got, want.ID)
	}
}

// TestVersionCommand_ReportsBuildCommitAndBuildDate pins CLM-025: the HUMAN rendering
// carries a commit and a build date. The values are asserted against the ONE resolved
// build identity the command reads, so this cannot pass by printing a second,
// separately resolved value that disagrees with `backstop version --json`.
func TestVersionCommand_ReportsBuildCommitAndBuildDate(t *testing.T) {
	want := effectiveBuildIdentity()
	if want.Commit == "" || want.BuildDate == "" {
		t.Fatalf("the resolved build identity has an empty field (commit=%q date=%q); CLM-029 requires the literal \"unknown\" instead", want.Commit, want.BuildDate)
	}

	root := NewRootCommand()
	out, err := executeCommand(root, "version")
	if err != nil {
		t.Fatalf("version command error: %v", err)
	}

	if !strings.Contains(out, want.Commit) {
		t.Errorf("version output does not carry the resolved commit %q:\n%s", want.Commit, out)
	}
	if !strings.Contains(out, want.BuildDate) {
		t.Errorf("version output does not carry the resolved build date %q:\n%s", want.BuildDate, out)
	}
	// The version line itself must not have regressed while the fields were added.
	if !strings.Contains(out, want.Version) {
		t.Errorf("version output does not carry the resolved version %q:\n%s", want.Version, out)
	}
}

// TestVersionCommand_JSONCarriesBuildIdentity pins CLM-030. The commit and build date
// are ADDITIVE: SPEC-005's version, schema_cohort and go_version keys must all still be
// present, since dropping one is a silent break of CLM-023.
func TestVersionCommand_JSONCarriesBuildIdentity(t *testing.T) {
	root := NewRootCommand()
	out, err := executeCommand(root, "version", "--json")
	if err != nil {
		t.Fatalf("version --json error: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("version --json output is not valid JSON: %v\noutput: %s", err, out)
	}

	// The pre-existing keys survive.
	for _, field := range []string{"version", "schema_cohort", "go_version", "schema_version"} {
		if _, ok := parsed[field]; !ok {
			t.Errorf("version --json dropped the pre-existing field %q; these are additive fields", field)
		}
	}

	want := effectiveBuildIdentity()
	if got, _ := parsed["commit"].(string); got != want.Commit {
		t.Errorf("version --json commit = %q, want %q", got, want.Commit)
	}
	if got, _ := parsed["build_date"].(string); got != want.BuildDate {
		t.Errorf("version --json build_date = %q, want %q", got, want.BuildDate)
	}
	if got, _ := parsed["version"].(string); got != want.Version {
		t.Errorf("version --json version = %q, want %q", got, want.Version)
	}
}
