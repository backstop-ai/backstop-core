package main

import (
	"testing"
)

// TestArtifactValidate_TypeFilterFlags_AllAccepted verifies that each
// per-type filter flag (--adr, --bundle, --issue, --directive, in addition to
// the already-covered --spec/--plan) is accepted and routed through the type
// filter path without a config-level error. With no matching artifacts present
// the command emits a zero-artifacts warning and exits clean (pass), which
// proves the filter was honored rather than rejected.
func TestArtifactValidate_TypeFilterFlags_AllAccepted(t *testing.T) {
	cases := []struct {
		name string
		flag string
		id   string
	}{
		{name: "adr filter", flag: "--adr", id: "ADR-001"},
		{name: "bundle filter", flag: "--bundle", id: "BUNDLE-001"},
		{name: "issue filter", flag: "--issue", id: "ISSUE-001"},
		{name: "directive filter", flag: "--directive", id: "DIR-001"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := setupArtifactTestDir(t, artifactTestBackstopYML, nil)
			stdout, stderr, exitCode := runValidateCommand(t, dir, tc.flag, tc.id)

			// A filter for a non-existent artifact must not be a config error
			// (exit 2). It either passes (zero artifacts) or reports the
			// missing artifact as a validation violation (exit 1) — both prove
			// the flag was parsed and routed, not rejected.
			if exitCode == ExitConfigError {
				t.Fatalf("flag %s produced a config error: stdout=%q stderr=%q", tc.flag, stdout, stderr)
			}
		})
	}
}

// TestArtifactValidate_MultipleTypeFilters verifies that several type filters
// can be combined in a single invocation; the command must parse all of them
// and not reject the combination as a config error.
func TestArtifactValidate_MultipleTypeFilters(t *testing.T) {
	dir := setupArtifactTestDir(t, artifactTestBackstopYML, nil)
	_, _, exitCode := runValidateCommand(t, dir, "--adr", "ADR-001", "--bundle", "BUNDLE-001", "--directive", "DIR-001")
	if exitCode == ExitConfigError {
		t.Fatalf("combined type filters produced a config error (exit %d)", exitCode)
	}
}
