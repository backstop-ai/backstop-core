package main

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// gate_severity_policyless_e2e_test.go — the ISSUE-105 A/B probe, driven through the
// REAL shipped runGate over a fixture consumer that declares NO enforcement.policy.
//
// WHY THE EXISTING CONTRACT TEST DID NOT CATCH THIS. pack_severity_contract_test.go
// drives verdictFor, which hands ApplyPolicy a policy map that ALWAYS contains a
// pack_engines entry — so both ISSUE-104 and ISSUE-105 shipped past it green. The
// missing half is the population the defect actually hit: a project that adopted a pack
// before writing any per-dimension enforcement config. That absence is the fixture.

// severityABFixtureRoot returns the committed policy-absent consumer tree.
func severityABFixtureRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("testdata", "severity-policy-ab"))
	if err != nil {
		t.Fatalf("resolving severity-ab fixture root: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(root, "backstop.yml")); statErr != nil {
		t.Fatalf("severity-ab fixture is missing its backstop.yml: %v", statErr)
	}
	return root
}

// severityABProject copies the fixture into a per-run temp dir so one run's config edit
// cannot leak into another, and returns the copy's path.
func severityABProject(t *testing.T) string {
	t.Helper()
	temp := t.TempDir()
	copyTree(t, severityABFixtureRoot(t), temp)
	return temp
}

// appendPackEnginesBlockPolicy writes the ONE delta that separates the A and B runs:
// a single `level: block` enforcement entry for the pack_engines dimension.
func appendPackEnginesBlockPolicy(t *testing.T, projectDir string) {
	t.Helper()
	path := filepath.Join(projectDir, "backstop.yml")
	existing, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading fixture backstop.yml: %v", err)
	}
	if strings.Contains(string(existing), "enforcement:") {
		t.Fatal("the fixture already declares an enforcement block; its whole purpose is that it " +
			"does NOT, and the A/B pair is only a probe while the policy entry is the sole delta")
	}
	updated := string(existing) + "enforcement:\n  policy:\n    pack_engines:\n      level: block\n"
	if writeErr := os.WriteFile(path, []byte(updated), 0o644); writeErr != nil {
		t.Fatalf("writing policy entry into fixture backstop.yml: %v", writeErr)
	}
}

// packEnginesStatus extracts the pack_engines step's reported status from the human gate
// output. The assertion is deliberately on THE STEP and never on a global warning count:
// the fixture declares no toolchain pack, so every run also carries the non-failing
// "enforcement not configured (0 toolchain packs)" warning step by design, plus
// capability-absent warnings on the traceability dimensions. A count assertion would fail
// for reasons that have nothing to do with this fix.
func packEnginesStatus(t *testing.T, output string) string {
	t.Helper()
	line := ""
	for _, candidate := range strings.Split(output, "\n") {
		if strings.HasPrefix(strings.TrimSpace(stripANSI(candidate)), "pack_engines ") {
			line = stripANSI(candidate)
			break
		}
	}
	if line == "" {
		t.Fatalf("the gate output has no pack_engines step line at all — the fixture is not reaching "+
			"the dimension under test, which would make this probe vacuous. Output:\n%s", output)
	}
	switch {
	case strings.Contains(line, "fail"):
		return "fail"
	case strings.Contains(line, "warning"):
		return "warning"
	case strings.Contains(line, "skipped"):
		return "skipped"
	case strings.Contains(line, "pass"):
		return "pass"
	}
	t.Fatalf("could not read a status from the pack_engines line %q", line)
	return ""
}

// ansiEscape matches the SGR colour codes the human renderer emits around statuses.
func ansiEscape() *regexp.Regexp {
	return regexp.MustCompile("\x1b\\[[0-9;]*m")
}

func stripANSI(s string) string {
	return ansiEscape().ReplaceAllString(s, "")
}

// TestGateCLI_DeclaredWarningDoesNotBlockWithoutPolicyEntry IS THE FALSIFIER (CLM-008).
//
// A pack finding declaring SARIF severity `warning` must not block a consumer that has
// written no enforcement.policy. RED before the fix: runGate returns a non-nil error and
// pack_engines reports "fail", because the step decided by RAW COUNT and never read the
// severity the pack declared.
func TestGateCLI_DeclaredWarningDoesNotBlockWithoutPolicyEntry(t *testing.T) {
	project := severityABProject(t)
	t.Setenv("SEVERITY_AB_LEVEL", "warning")

	out, err := runGateInDir(t, project)
	if err != nil {
		t.Fatalf("a DECLARED-WARNING pack finding blocked a consumer with NO enforcement.policy; "+
			"the severity contract belongs to the finding, not to adopter configuration. err=%v\noutput:\n%s",
			err, out)
	}
	if got := packEnginesStatus(t, out); got != "warning" {
		t.Errorf("expected the pack_engines step to report %q, got %q; a warning that reports \"pass\" "+
			"would be invisible and one that reports \"fail\" would block\noutput:\n%s", "warning", got, out)
	}
	if !strings.Contains(out, "capture.capture-sample-panic") {
		t.Errorf("non-blocking must not mean invisible: the finding is missing from the report\noutput:\n%s", out)
	}
}

// TestGateCLI_DeclaredWarningStaysNonBlockingWithPolicyEntry is the NON-REGRESSION half.
//
// THE ONLY DELTA FROM THE PREVIOUS TEST is the appended
//
//	enforcement:
//	  policy:
//	    pack_engines:
//	      level: block
//
// block. Same fixture, same pack, same captured bytes, same SEVERITY_AB_LEVEL. That
// identity is what makes the pair a PROBE rather than two unrelated assertions: it
// isolates the policy entry as the single variable, exactly as ISSUE-105 measured it.
// This run is green today and must stay green, proving the fix did not move the behavior
// consumers already have.
func TestGateCLI_DeclaredWarningStaysNonBlockingWithPolicyEntry(t *testing.T) {
	project := severityABProject(t)
	appendPackEnginesBlockPolicy(t, project)
	t.Setenv("SEVERITY_AB_LEVEL", "warning")

	out, err := runGateInDir(t, project)
	if err != nil {
		t.Fatalf("a declared-warning finding blocked a consumer WITH a level:block policy entry; "+
			"this path was green before the fix and must stay green. err=%v\noutput:\n%s", err, out)
	}
	if got := packEnginesStatus(t, out); got != "warning" {
		t.Errorf("expected pack_engines %q with a policy entry present, got %q\noutput:\n%s",
			"warning", got, out)
	}
}

// TestGateCLI_DeclaredErrorBlocksWithAndWithoutPolicyEntry guards that the fix narrowed
// the verdict by SEVERITY and did not simply stop blocking pack findings (CLM-004/008).
//
// Both configurations must fail, before AND after. If either goes green, the lane
// disarmed the gate rather than teaching it to read severity.
func TestGateCLI_DeclaredErrorBlocksWithAndWithoutPolicyEntry(t *testing.T) {
	cases := []struct {
		name       string
		withPolicy bool
	}{
		{"no policy entry", false},
		{"level:block policy entry", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			project := severityABProject(t)
			if tc.withPolicy {
				appendPackEnginesBlockPolicy(t, project)
			}
			t.Setenv("SEVERITY_AB_LEVEL", "error")

			out, err := runGateInDir(t, project)
			if err == nil {
				t.Fatalf("a DECLARED-ERROR pack finding did not block (%s); severity-aware must never "+
					"mean relaxed\noutput:\n%s", tc.name, out)
			}
			if got := packEnginesStatus(t, out); got != "fail" {
				t.Errorf("expected pack_engines %q for a declared-error finding (%s), got %q\noutput:\n%s",
					"fail", tc.name, got, out)
			}
		})
	}
}

// TestSeverityABFixture_SarifBytesAreTheCapturedSemgrepOutput keeps the fixture honest
// (CLM-008).
//
// The fixture's engine `cat`s copies of ISSUE-104's captured real semgrep output —
// severity on the RULE DESCRIPTOR with no result-level `level`, which is the wire shape
// semgrep actually emits and the reason the ISSUE-104 parser hop exists. A copied capture
// that drifts is a fabrication wearing a provenance file's name, so byte-equality with the
// canonical captures is asserted rather than assumed. cmd/backstop/testdata/semgrep/
// fixtures/PROVENANCE.md is the provenance of record.
func TestSeverityABFixture_SarifBytesAreTheCapturedSemgrepOutput(t *testing.T) {
	fixtureRoot := severityABFixtureRoot(t)
	canonicalRoot := filepath.Join("testdata", "semgrep", "fixtures")

	for _, name := range []string{"descriptor-warning.sarif", "descriptor-error.sarif"} {
		copied, err := os.ReadFile(filepath.Join(fixtureRoot, name))
		if err != nil {
			t.Fatalf("reading the fixture's %s: %v", name, err)
		}
		canonical, canonicalErr := os.ReadFile(filepath.Join(canonicalRoot, name))
		if canonicalErr != nil {
			t.Fatalf("reading the canonical capture %s: %v", name, canonicalErr)
		}
		if !bytes.Equal(copied, canonical) {
			t.Errorf("%s has DRIFTED from the canonical capture (%d vs %d bytes); fixtures are "+
				"CAPTURED from real tool output, never fabricated, and a drifted copy is both",
				name, len(copied), len(canonical))
		}
	}
}
