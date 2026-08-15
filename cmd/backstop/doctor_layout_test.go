package main

import (
	"go/ast"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// REQ-007: layout deviations decided PER KIND against the resolved root.
//
// EVERY TEST RUNS AGAINST A STAGED FIXTURE CORPUS. Neither this spec nor SPEC-068 can
// prove this by dogfood: backstop-core is clean under BOTH the correct per-kind reading
// and the wrong root-containment reading, so an in-repo run shows nothing either way.
//
// ★★ THE ZERO-DEVIATION TRIPWIRE, and it is invisible in a diff review.
// gate.FindUngatedArtifacts SKIPS `testdata` among its non-corpus exclusion trees — that
// exclusion is load-bearing for this repository and is not going away. So if a layout
// corpus is read IN PLACE instead of staged into t.TempDir, EVERY case returns ZERO
// deviations: the four tests asserting an empty set pass, the two asserting an absence
// WITHIN the set pass, and only the deviation-REPORTING tests red. It looks like a
// partially-working implementation rather than a staging bug.
//
// SO: whenever a deviation-expecting case reports ZERO deviations, suspect IN-PLACE
// STAGING FIRST — before the check, the resolver, or the fixture's contents. Confirm the
// path the test read is under t.TempDir() and NOT under cmd/backstop/testdata/.

// layoutResult runs the layout check over a staged corpus and returns the staged path plus
// the reported status, message and remediation.
func layoutResult(t *testing.T, template string) (string, string, string, string) {
	t.Helper()

	project := stageDoctorProject(t, template)
	if strings.Contains(project, filepath.Join("testdata", "doctor")) {
		t.Fatalf("corpus %q was read IN PLACE at %s; FindUngatedArtifacts excludes testdata, so every case would report zero deviations", template, project)
	}
	payload, _ := runDoctorJSON(t, project, "--check", doctorCheckArtifactLayout)
	return project,
		payload.statuses()[doctorCheckArtifactLayout],
		payload.field(t, doctorCheckArtifactLayout, "message"),
		payload.field(t, doctorCheckArtifactLayout, "remediation")
}

// TestDoctorLayout_ReportsAgainstResolvedArtifactRoot (CLM-039).
//
// The twin corpora hold the SAME spec file and declare the SAME root; only its location
// differs. That the verdict flips between them is the proof the check consumes the shared
// resolution rather than a root of its own.
func TestDoctorLayout_ReportsAgainstResolvedArtifactRoot(t *testing.T) {
	_, cleanStatus, cleanMessage, _ := layoutResult(t, "layout-configured-root-clean")
	if cleanStatus != "pass" {
		t.Errorf("the spec inside the DECLARED root was reported as a deviation: %s", cleanMessage)
	}

	_, deviatingStatus, deviatingMessage, _ := layoutResult(t, "layout-configured-root-deviating")
	if deviatingStatus == "pass" {
		t.Errorf("the SAME spec outside the declared root was not reported: %s", deviatingMessage)
	}
	if !strings.Contains(deviatingMessage, "SPEC-001-sample.spec.md") {
		t.Errorf("the deviation does not name the file: %s", deviatingMessage)
	}
}

// TestDoctorLayout_DeviationReportsActualAndExpectedPath (CLM-040).
func TestDoctorLayout_DeviationReportsActualAndExpectedPath(t *testing.T) {
	project, status, message, _ := layoutResult(t, "layout-nested-artifact")

	if status == "pass" {
		t.Fatalf("a nested artifact was not reported; the status walk reads type directories with os.ReadDir and SKIPS subdirectories, so it is exactly as ungated as one in the wrong tree")
	}
	actual := filepath.Join(project, "specs", "archive", "SPEC-001-sample.spec.md")
	expected := filepath.Join(project, "specs")
	if !strings.Contains(message, actual) {
		t.Errorf("the deviation does not name its ACTUAL path %s: %s", actual, message)
	}
	if !strings.Contains(message, expected) {
		t.Errorf("the deviation does not name the EXPECTED path %s: %s", expected, message)
	}
}

// TestDoctorLayout_ClassifiesEveryArtifactSuffix (CLM-041).
//
// The seven artifacts sit MISPLACED together in misc/, so the deviation set is exactly
// seven entries spanning seven distinct kinds. Placed correctly, a perfect classifier and
// one recognizing nothing would both report zero and this test would prove nothing.
func TestDoctorLayout_ClassifiesEveryArtifactSuffix(t *testing.T) {
	_, _, message, _ := layoutResult(t, "layout-all-suffixes")

	suffixes := []string{
		"SPEC-001-sample.spec.md",
		"PLAN-SPEC-001-sample.plan.yml",
		"ADR-001-sample.adr.md",
		"BUNDLE-001-sample.bundle.md",
		"ISSUE-001-sample.issue.md",
		"DIR-001-sample.directive.md",
		"CAP-001-sample.capability.yml",
	}
	for _, name := range suffixes {
		if !strings.Contains(message, name) {
			t.Errorf("the classifier silently dropped %s — it is misplaced in misc/ and must appear as a deviation:\n%s", name, message)
		}
	}

	// The expected directory for each kind must appear too, so a classifier that
	// recognized a suffix but resolved it to the wrong kind still reds.
	for _, dir := range []string{"specs", "plans", "adrs", "bundles", "issues", "directives", "capabilities"} {
		if !strings.Contains(message, string(filepath.Separator)+dir) {
			t.Errorf("no deviation names the expected directory %q, so one kind resolved wrongly:\n%s", dir, message)
		}
	}
}

// TestDoctorLayout_BareCapabilityYmlIsNotReportedAndDoctorAddsNoPattern (CLM-062).
//
// Asserted as an ABSENCE WITHIN a non-empty set, never as an empty set: this repository's
// own capabilities/CAP-001-pack-gate-enforcement/capability.yml has exactly this shape and
// must keep escaping, and doctor must add no filename pattern of its own.
func TestDoctorLayout_BareCapabilityYmlIsNotReportedAndDoctorAddsNoPattern(t *testing.T) {
	_, _, message, _ := layoutResult(t, "layout-all-suffixes")

	if !strings.Contains(message, "CAP-001-sample.capability.yml") {
		t.Fatalf("the deviation set does not carry the suffixed capability, so this absence assertion would pass over an empty set:\n%s", message)
	}
	if strings.Contains(message, filepath.Join("CAP-002-bare-name", "capability.yml")) {
		t.Errorf("the bare capability.yml was reported; widening the pattern reds this repository's own gate on CAP-001:\n%s", message)
	}

	// THE STRUCTURAL HALF: doctor holds no filename pattern, suffix, directory name or
	// exclusion list of its own — all four come from the shared helper, which is what
	// makes "what doctor calls a deviation" and "what the gate calls ungated" ONE
	// predicate by construction.
	assertLayoutCheckHoldsNoLayoutKnowledge(t)
}

// assertLayoutCheckHoldsNoLayoutKnowledge is the structural half of CLM-062: the check
// holds no artifact directory name, no suffix or filename pattern, no corpus walk of its
// own, and no exclusion list.
//
// ALL FOUR COME FROM THE SHARED HELPER, and a single one of them here makes this the
// FOURTH hardcoding bundle REQ-029 exists to delete — which no test asserting only output
// would catch, because a hand-rolled copy agrees with the shared table on the day it is
// written.
func assertLayoutCheckHoldsNoLayoutKnowledge(t *testing.T) {
	t.Helper()

	found := false
	for _, file := range parseNonTestPackageFiles(t) {
		ast.Inspect(file.file, func(node ast.Node) bool {
			decl, ok := node.(*ast.FuncDecl)
			if !ok || decl.Body == nil || decl.Name.Name != "checkArtifactLayout" {
				return true
			}
			found = true

			forbidden := map[string]string{
				"specs": "an artifact directory name", "plans": "an artifact directory name",
				"adrs": "an artifact directory name", "bundles": "an artifact directory name",
				"issues": "an artifact directory name", "directives": "an artifact directory name",
				"capabilities": "an artifact directory name",
				".spec.md":     "an artifact suffix", ".plan.yml": "an artifact suffix",
				".adr.md": "an artifact suffix", ".bundle.md": "an artifact suffix",
				".issue.md": "an artifact suffix", ".directive.md": "an artifact suffix",
				".capability.yml": "an artifact suffix",
				"testdata":        "an exclusion-list entry", "vendor": "an exclusion-list entry",
				"node_modules": "an exclusion-list entry", "prototype": "an exclusion-list entry",
			}
			ast.Inspect(decl.Body, func(inner ast.Node) bool {
				if lit, isLit := inner.(*ast.BasicLit); isLit && lit.Kind == token.STRING {
					value, err := strconv.Unquote(lit.Value)
					if err == nil {
						if kind, isForbidden := forbidden[value]; isForbidden {
							t.Errorf("checkArtifactLayout carries the literal %q — %s. All four come from the shared helper, or this becomes the fourth hardcoding REQ-029 exists to delete", value, kind)
						}
					}
				}
				// No corpus walk of its own.
				if selector, isSelector := inner.(*ast.SelectorExpr); isSelector {
					switch selector.Sel.Name {
					case "Walk", "WalkDir", "ReadDir":
						t.Errorf("checkArtifactLayout calls filepath/os.%s — it must perform no corpus walk of its own", selector.Sel.Name)
					}
				}
				return true
			})
			return true
		})
	}
	if !found {
		t.Fatalf("checkArtifactLayout was not found in the non-test sources — the scan proved nothing")
	}
}

// TestDoctorLayout_NonArtifactFilesAreNotReported (CLM-042).
func TestDoctorLayout_NonArtifactFilesAreNotReported(t *testing.T) {
	_, _, message, _ := layoutResult(t, "layout-all-suffixes")

	if !strings.Contains(message, "SPEC-001-sample.spec.md") {
		t.Fatalf("the deviation set is empty, so this absence assertion proves nothing:\n%s", message)
	}
	for _, name := range []string{"README.md", "notes.txt"} {
		if strings.Contains(message, name) {
			t.Errorf("%s was reported as a layout deviation; it is not artifact-shaped:\n%s", name, message)
		}
	}
}

// TestDoctorLayout_ArtifactsInTheirExpectedDirectoryAreNotDeviations (CLM-043).
func TestDoctorLayout_ArtifactsInTheirExpectedDirectoryAreNotDeviations(t *testing.T) {
	_, status, message, _ := layoutResult(t, "layout-configured-root-clean")

	if status != "pass" {
		t.Errorf("an artifact in the directory expected for its kind was reported: %s", message)
	}
}

// TestDoctorLayout_AbsentTypeDirectoryIsNotADeviation (CLM-044).
func TestDoctorLayout_AbsentTypeDirectoryIsNotADeviation(t *testing.T) {
	_, status, message, _ := layoutResult(t, "layout-absent-type-directory")

	if status != "pass" {
		t.Errorf("a root holding some type directories and missing others was reported: %s", message)
	}
}

// TestDoctorLayout_EmptyArtifactRootPasses (CLM-045) — the validated greenfield outcome.
func TestDoctorLayout_EmptyArtifactRootPasses(t *testing.T) {
	project, status, message, _ := layoutResult(t, "layout-empty-root")

	// The harness must have removed .backstop/.gitkeep RECURSIVELY, or the root is not
	// empty and this case asserts the wrong condition.
	entries, err := filepath.Glob(filepath.Join(project, ".backstop", "*"))
	if err != nil || len(entries) != 0 {
		t.Fatalf("the staged root is not empty (%v), so the empty-root condition was never exercised", entries)
	}
	if status != "pass" {
		t.Errorf("an existing but empty artifact root did not pass: %s", message)
	}
}

// TestDoctorLayout_UnconfiguredRootReportsDotBackstopArtifactsAsDeviations (CLM-046).
//
// ★ THE ONE THAT MATTERS MOST. A root-CONTAINMENT implementation passes every other test
// in this file and reds only this one: with no configured root the resolved root IS the
// project root, which contains everything, so containment reports nothing while none of
// these files is discovered or gated. The count assertion is what turns the in-place
// staging tripwire from advice into a failing test.
func TestDoctorLayout_UnconfiguredRootReportsDotBackstopArtifactsAsDeviations(t *testing.T) {
	project, status, message, _ := layoutResult(t, "layout-unconfigured-dot-backstop")

	if status == "pass" {
		t.Fatalf("the .backstop/-rooted corpus reported NO deviation — this is the containment reading, or the corpus was read in place")
	}
	for _, name := range []string{
		filepath.Join(".backstop", "specs", "SPEC-001-sample.spec.md"),
		filepath.Join(".backstop", "bundles", "BUNDLE-001-sample.bundle.md"),
		filepath.Join(".backstop", "plans", "PLAN-SPEC-001-sample.plan.yml"),
	} {
		if !strings.Contains(message, filepath.Join(project, name)) {
			t.Errorf("the report does not name %s by path:\n%s", name, message)
		}
	}
	// AND THE EXPECTED DEFAULT PATH, which is what makes the report actionable rather
	// than merely accusatory.
	if !strings.Contains(message, filepath.Join(project, "specs")) {
		t.Errorf("the report does not name the expected default path for the spec:\n%s", message)
	}
}

// TestDoctorLayout_UnconfiguredRootWithExpectedLayoutPassesWithoutWarning (CLM-047).
//
// NO WARNING, because a warn-when-unconfigured rule would fire forever in backstop-core —
// and the fix for that must never be a special case naming this repository inside core.
// This corpus differs from CLM-046's by the `.backstop/` prefix and nothing else.
func TestDoctorLayout_UnconfiguredRootWithExpectedLayoutPassesWithoutWarning(t *testing.T) {
	_, status, message, _ := layoutResult(t, "layout-unconfigured-expected-layout")

	if status != "pass" {
		t.Errorf("an unconfigured root whose artifacts sit in their expected directories did not pass cleanly (status %q): %s", status, message)
	}
}

// TestDoctorLayout_RemediationOffersBothMoveAndDeclareRoot (CLM-048).
//
// BOTH remedies, asserting NEITHER as correct: the choice is the consumer's layout
// decision, and SPEC-068's resolution is policy-FREE — it has no notion that `.backstop/`
// is canonical, so any such opinion would be doctor's own.
func TestDoctorLayout_RemediationOffersBothMoveAndDeclareRoot(t *testing.T) {
	_, status, _, remediation := layoutResult(t, "layout-nested-artifact")

	if status == "pass" {
		t.Fatalf("no deviation was reported, so there is no remediation to assert on")
	}
	lower := strings.ToLower(remediation)
	if !strings.Contains(lower, "move") {
		t.Errorf("the remediation does not offer moving the file: %q", remediation)
	}
	if !strings.Contains(lower, "artifact_root") {
		t.Errorf("the remediation does not offer declaring the root that makes the location expected: %q", remediation)
	}
	for _, asserted := range []string{"you should", "the correct", "must move", "we recommend"} {
		if strings.Contains(lower, asserted) {
			t.Errorf("the remediation asserts one remedy as correct (%q): %q", asserted, remediation)
		}
	}
}

// TestDoctorLayout_ConfiguredRootMissingFromDiskFails (CLM-049).
func TestDoctorLayout_ConfiguredRootMissingFromDiskFails(t *testing.T) {
	_, status, message, _ := layoutResult(t, "layout-configured-root-missing")

	if status != "fail" {
		t.Errorf("a configured root absent from disk did not fail (status %q): %s", status, message)
	}
	if !strings.Contains(message, "artifacts-that-do-not-exist") {
		t.Errorf("the failure does not name the DECLARED value: %s", message)
	}
}

// TestDoctorLayout_InvalidRootDeclarationFailsNamingTheReason (CLM-050).
//
// DISTINCT from the missing case, and distinguished by ERROR TYPE rather than by string
// match. The two messages are compared so a single collapsed branch fails.
func TestDoctorLayout_InvalidRootDeclarationFailsNamingTheReason(t *testing.T) {
	_, invalidStatus, invalidMessage, _ := layoutResult(t, "layout-configured-root-invalid")
	_, _, missingMessage, _ := layoutResult(t, "layout-configured-root-missing")

	if invalidStatus != "fail" {
		t.Errorf("an invalid root declaration did not fail (status %q): %s", invalidStatus, invalidMessage)
	}
	if !strings.Contains(invalidMessage, "/absolute/is/rejected") {
		t.Errorf("the failure does not name the declared value: %s", invalidMessage)
	}
	if !strings.Contains(strings.ToLower(invalidMessage), "relative") && !strings.Contains(strings.ToLower(invalidMessage), "absolute") {
		t.Errorf("the failure does not name the REASON: %s", invalidMessage)
	}
	if invalidMessage == missingMessage {
		t.Errorf("the invalid and missing cases produce the identical message, so a single collapsed branch would pass: %s", invalidMessage)
	}
}
