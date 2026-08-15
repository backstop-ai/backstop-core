package main

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// REQ-006: the four outcomes, the shared execution path, and the skip that protects
// outcome (d).
//
// ★ SCOPE. These tests assert DOCTOR's rollup, status mapping and message content — not
// the prober's mechanics. Selection by gate_type, the trust gate's ordering, the
// argv/no-shell property and the (b)/(c) split are the shared packEntrypointProber's, and
// SPEC-069's own tests own them at that level. Everything here is driven through
// `backstop doctor` so the claim is asserted on the reported RESULT, which is what
// SPEC-070 actually promises.

// toolchainResult runs the toolchain check over a staged fixture and returns the staged
// path plus the reported result fields.
func toolchainResult(t *testing.T, template string) (string, string, string, string) {
	t.Helper()

	project := stageDoctorProject(t, template)
	payload, _ := runDoctorJSON(t, project, "--check", doctorCheckToolchainRuns)
	return project,
		payload.statuses()[doctorCheckToolchainRuns],
		payload.field(t, doctorCheckToolchainRuns, "message"),
		payload.field(t, doctorCheckToolchainRuns, "remediation")
}

// TestDoctorToolchain_PassNamesPackAndCommandOnSuccess (CLM-031) — outcome (a).
func TestDoctorToolchain_PassNamesPackAndCommandOnSuccess(t *testing.T) {
	_, status, message, _ := toolchainResult(t, "clean")

	if status != "pass" {
		t.Fatalf("toolchain-runs status = %q on a succeeding entrypoint, want pass: %s", status, message)
	}
	if !strings.Contains(message, "backstop/doctor-fixture") {
		t.Errorf("message does not name the pack: %q", message)
	}
	if !strings.Contains(message, "go build -o clean.marker ./...") {
		t.Errorf("message does not name the command it ran: %q", message)
	}
}

// TestDoctorToolchain_MissingExecutableReportedAsOwedSetup (CLM-032) — outcome (b).
//
// It is SETUP THE CONSUMER STILL OWES, pointed at THAT PACK's own documented install
// steps. Doctor installs nothing, which is asserted as a filesystem fact.
func TestDoctorToolchain_MissingExecutableReportedAsOwedSetup(t *testing.T) {
	// The before/after snapshot is taken on the SAME staged tree, so it compares like
	// with like — and it names no path under testdata, which would be a corpus read in
	// place rather than a staged one.
	project := stageDoctorProject(t, "toolchain-missing-executable")
	before := projectFileSet(t, project)

	payload, _ := runDoctorJSON(t, project, "--check", doctorCheckToolchainRuns)
	status := payload.statuses()[doctorCheckToolchainRuns]
	message := payload.field(t, doctorCheckToolchainRuns, "message")
	remediation := payload.field(t, doctorCheckToolchainRuns, "remediation")

	if status != "fail" {
		t.Errorf("toolchain-runs status = %q on an unstartable entrypoint, want fail: %s", status, message)
	}
	if !strings.Contains(message, "backstop/missing-executable") {
		t.Errorf("message does not name the pack whose entrypoint could not run: %q", message)
	}
	combined := message + " " + remediation
	if !strings.Contains(strings.ToLower(combined), "setup") {
		t.Errorf("the report does not classify this as setup the consumer still owes: %q", combined)
	}
	if !strings.Contains(combined, "backstop/missing-executable") {
		t.Errorf("the remediation does not point at that pack's own install steps: %q", combined)
	}

	// DOCTOR INSTALLED NOTHING: the staged tree carries exactly the files the template
	// carried.
	after := projectFileSet(t, project)
	if !slices.Equal(before, after) {
		t.Errorf("doctor changed the project tree; before=%v after=%v", before, after)
	}
}

// TestDoctorToolchain_NonZeroExitReportsOutputVerbatimWithoutInferringCause (CLM-033) —
// outcome (c).
//
// ★★ THIS IS ALSO THE CAPTURE-METHOD TRIPWIRE, AND IT IS INVISIBLE IN A DIFF REVIEW. The
// fixture's `go build` failure writes its ENTIRE diagnostic to STDERR and nothing to
// stdout. An implementation bound to check.CommandRunner.RunStdout — which is what
// following runFindingsEngine's pattern would have produced — captures an EMPTY string
// here and passes every other assertion in this file. Consuming the shared prober binds
// Run (combined output) and settles it by construction, but it is asserted anyway,
// because an empty-string "verbatim" is a vacuous green on the claim's whole point.
func TestDoctorToolchain_NonZeroExitReportsOutputVerbatimWithoutInferringCause(t *testing.T) {
	_, status, message, remediation := toolchainResult(t, "toolchain-nonzero-exit")

	if status != "fail" {
		t.Fatalf("toolchain-runs status = %q on a nonzero exit, want fail: %s", status, message)
	}
	if !strings.Contains(message, "backstop/nonzero-exit") {
		t.Errorf("message does not attribute the failure to the pack: %q", message)
	}
	if !strings.Contains(message, "go build ./...") {
		t.Errorf("message does not name the command: %q", message)
	}
	if !strings.Contains(message, "exit") || !strings.Contains(message, "1") {
		t.Errorf("message does not report the exit code: %q", message)
	}
	// The compiler's OWN text, from stderr. A capture bound to stdout finds nothing here.
	if !strings.Contains(message, "broken.go") {
		t.Errorf("captured output is not the compiler's own stderr diagnostic — an implementation bound to RunStdout captures nothing here: %q", message)
	}
	// AND IT IS VERBATIM. The compiler emits its diagnostic lines at column zero, so a
	// report that indented or otherwise reformatted the capture — putting doctor's own
	// whitespace inside the bytes a consumer reads as the tool's output — fails here while
	// every "contains broken.go" assertion above still passes.
	if !strings.Contains(message, "\n./broken.go:") {
		t.Errorf("the captured output was reformatted rather than reported verbatim; the compiler's line does not start at column zero: %q", message)
	}

	// NO INFERRED CAUSE. The pnpm ERR_PNPM_IGNORED_BUILDS evidence is exactly a nonzero
	// exit whose obvious reading was wrong, so a nonzero exit is never reclassified as
	// owed setup even when it looks like a missing dependency.
	if strings.Contains(strings.ToLower(message+" "+remediation), "setup the consumer") {
		t.Errorf("a nonzero exit was reclassified as owed setup: %q / %q", message, remediation)
	}
}

// TestDoctorToolchain_WarnsWhenNoPackDeclaresToolchainEntrypoint (CLM-034) — outcome (d).
func TestDoctorToolchain_WarnsWhenNoPackDeclaresToolchainEntrypoint(t *testing.T) {
	_, status, message, _ := toolchainResult(t, "toolchain-no-entrypoint")

	if status == "pass" {
		t.Errorf("toolchain-runs PASSED with no declared entrypoint — a silent pass is what this claim forbids: %q", message)
	}
	if status != "warn" {
		t.Errorf("toolchain-runs status = %q, want warn: %s", status, message)
	}
	if !strings.Contains(message, "entrypoint") {
		t.Errorf("message does not state that no toolchain entrypoint is declared: %q", message)
	}
}

// TestDoctorToolchain_ReportsEachDeclaredEntrypointSeparately (CLM-035).
//
// Several PACKS, each entrypoint reported separately INSIDE the one result — not one
// result per entrypoint, and not one registered check per entrypoint.
func TestDoctorToolchain_ReportsEachDeclaredEntrypointSeparately(t *testing.T) {
	project, status, message, _ := toolchainResult(t, "toolchain-multi-entrypoint")

	if status != "pass" {
		t.Fatalf("toolchain-runs status = %q, want pass: %s", status, message)
	}
	for _, pack := range []string{"backstop/entrypoint-alpha", "backstop/entrypoint-beta"} {
		if !strings.Contains(message, pack) {
			t.Errorf("the one result does not report %s separately: %q", pack, message)
		}
	}
	markers := markerFiles(t, project)
	for _, marker := range []string{"alpha.marker", "beta.marker"} {
		if !slices.Contains(markers, marker) {
			t.Errorf("%s is absent, so that pack's entrypoint never ran; markers: %v", marker, markers)
		}
	}
}

// TestDoctorToolchain_PassesWhenEntrypointSucceedsDespiteFailingPackageManagerConfig
// (CLM-036) — DD-6 made testable: health comes from EXECUTION, never from a file on disk.
func TestDoctorToolchain_PassesWhenEntrypointSucceedsDespiteFailingPackageManagerConfig(t *testing.T) {
	project, status, message, _ := toolchainResult(t, "toolchain-failing-package-manager")

	if _, err := os.Stat(filepath.Join(project, "package.json")); err != nil {
		t.Fatalf("the failing package-manager configuration is missing from the staged project: %v", err)
	}
	if status != "pass" {
		t.Errorf("toolchain-runs status = %q despite a succeeding entrypoint, want pass: %s", status, message)
	}
	if strings.Contains(message, "package.json") {
		t.Errorf("the report reads the package-manager configuration; the check's whole point is that it does not: %q", message)
	}
}

// TestDoctorToolchain_RefusesCommandWhoseToolIsNotAllowlisted (CLM-037).
//
// The fixture's binding carries a `provision:` block, which is the ONLY shape that
// reaches the allowlist at all — checkEngineToolAllowed returns nil immediately on a nil
// Provision.
func TestDoctorToolchain_RefusesCommandWhoseToolIsNotAllowlisted(t *testing.T) {
	project, status, message, _ := toolchainResult(t, "toolchain-not-allowlisted")

	if status != "fail" {
		t.Errorf("toolchain-runs status = %q on a refused command, want fail: %s", status, message)
	}
	if !strings.Contains(message, "backstop/not-allowlisted") {
		t.Errorf("message does not attribute the refusal to the pack: %q", message)
	}
	if !strings.Contains(strings.ToLower(message), "allow") {
		t.Errorf("message does not report a trusted-tool refusal: %q", message)
	}
	// AND THE TOOL DID NOT RUN: a refused command is never split and never executed.
	if markers := markerFiles(t, project); len(markers) != 0 {
		t.Errorf("a refused entrypoint produced side effects: %v", markers)
	}
}

// TestDoctorToolchain_ShellMetacharactersArePassedAsLiteralArguments (CLM-038).
//
// BOTH DIRECTIONS, because one alone is not proof that no shell ran: the literally-named
// artifact must EXIST (the substitution was never performed) and the injected marker must
// be ABSENT (the separator was never honoured).
func TestDoctorToolchain_ShellMetacharactersArePassedAsLiteralArguments(t *testing.T) {
	project, status, message, _ := toolchainResult(t, "toolchain-shell-metacharacters")

	if status != "pass" {
		t.Fatalf("toolchain-runs status = %q, want pass — both fixture entrypoints should succeed with no shell: %s", status, message)
	}

	literal := filepath.Join(project, "out$(whoami).marker")
	if _, err := os.Stat(literal); err != nil {
		t.Errorf("the literally-named artifact is absent, so the $(...) token was not passed literally: %v; markers: %v",
			err, markerFiles(t, project))
	}
	if _, err := os.Stat(filepath.Join(project, "pwned.marker")); err == nil {
		t.Errorf("pwned.marker exists — the `;` was interpreted, so a shell ran")
	}
}

// TestDoctorToolchain_SkippedWhenPackSetCouldNotBeGathered (CLM-063).
//
// An ungathered pack set must never be read as outcome (d): "no toolchain entrypoint
// declared" requires a pack set that WAS gathered and declares none.
//
// ★ ALL THREE WAYS THE SET GOES UNGATHERED, not just the loader error. gatherDoctorContext
// calls loadInstalledPacks ONLY when the config was both discovered and loaded, so an
// absent or malformed backstop.yml leaves Packs nil with PacksErr ALSO nil. A skip
// predicate keyed on PacksErr alone therefore falls through to outcome (d) and reports
// "no installed pack declares a test or build entrypoint" about packs it never looked at
// — on a project whose real defect is the config. The missing-pack case alone cannot
// catch that, which is why this is a table.
func TestDoctorToolchain_SkippedWhenPackSetCouldNotBeGathered(t *testing.T) {
	cases := map[string]string{
		"loader errored on a declared pack missing from disk":     "config-missing-pack",
		"no backstop.yml at all, so the set was never loaded":     "no-config",
		"backstop.yml does not load, so the set was never loaded": "config-malformed",
	}
	for name, template := range cases {
		t.Run(name, func(t *testing.T) {
			_, status, message, _ := toolchainResult(t, template)

			if status != "skipped" {
				t.Errorf("toolchain-runs status = %q on an ungathered pack set, want skipped: %s", status, message)
			}
			if !strings.Contains(message, doctorCheckPacksInstalled) {
				t.Errorf("the skip does not name %q as the owner of that condition: %q", doctorCheckPacksInstalled, message)
			}
			if strings.Contains(message, "entrypoint") {
				t.Errorf("the skip reads as outcome (d) rather than as an ungathered pack set: %q", message)
			}
		})
	}
}

// projectFileSet lists every file under root, relative and sorted.
func projectFileSet(t *testing.T, root string) []string {
	t.Helper()

	var found []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		if rel == ".gitkeep" {
			return nil
		}
		found = append(found, rel)
		return nil
	})
	if err != nil {
		t.Fatalf("walking %s: %v", root, err)
	}
	slices.Sort(found)
	return found
}
