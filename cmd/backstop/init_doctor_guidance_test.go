package main

import (
	"bytes"
	"go/ast"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/initialize"
	"github.com/spf13/cobra"
)

// REQ-004: init names the registered check id, from the ONE id source.

// renderInitReportForTest drives the SHIPPED renderer and returns what it printed.
//
// Driving the renderer rather than doctorGuidanceForSteps alone is what locks the WIRING
// hop: a guidance function that returned the right strings while nothing printed them
// would satisfy a function-level test and deliver nothing to a consumer.
func renderInitReportForTest(t *testing.T, steps []initialize.StepReport) string {
	t.Helper()

	cmd := &cobra.Command{}
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	renderInitReport(cmd, initialize.Options{ProjectRoot: "/tmp/doctor-guidance-fixture"}, initialize.Result{Steps: steps})
	return buf.String()
}

// TestInit_ToolchainFailureNamesTheToolchainRunsCheckID (CLM-019).
//
// The printed id is compared against the CONSTANT, never against a literal typed into
// this test: a test carrying its own copy of the string is the second source REQ-004
// exists to prevent, and it would agree with the registry on the day it was written and
// stop agreeing the day someone renamed the check.
func TestInit_ToolchainFailureNamesTheToolchainRunsCheckID(t *testing.T) {
	printed := renderInitReportForTest(t, []initialize.StepReport{
		{Step: initialize.StepToolchain, Outcome: initialize.OutcomeBrokenPromise, Detail: "the declared entrypoint could not be started"},
	})

	if !strings.Contains(printed, "backstop doctor") {
		t.Errorf("a failed toolchain step printed no `backstop doctor` guidance:\n%s", printed)
	}
	if !strings.Contains(printed, doctorCheckToolchainRuns) {
		t.Errorf("the guidance does not name the registered toolchain check id:\n%s", printed)
	}

	// AND IT IS THE REGISTRY'S TEXT, not a hand-formatted twin: the exact string
	// doctorGuidance returns must appear.
	guidance, ok := doctorGuidance(doctorCheckToolchainRuns)
	if !ok {
		t.Fatalf("doctorGuidance does not resolve the toolchain check id, so init has nothing registered to print")
	}
	if !strings.Contains(printed, guidance) {
		t.Errorf("init printed guidance that is not the registry's own text %q:\n%s", guidance, printed)
	}

	// ★ ONCE, NOT ONCE PER ENTRYPOINT. The toolchain probe emits one StepReport per probed
	// entrypoint, so two failing packs produce two failed `toolchain` steps — and the
	// advice is per CHECK, not per entrypoint, because the one command re-runs all of
	// them. Printing it twice is noise a consumer reads as two different problems.
	t.Run("two failing toolchain steps print the guidance once", func(t *testing.T) {
		printed := renderInitReportForTest(t, []initialize.StepReport{
			{Step: initialize.StepToolchain, Outcome: initialize.OutcomeBrokenPromise, Detail: "pack alpha's entrypoint could not be started"},
			{Step: initialize.StepToolchain, Outcome: initialize.OutcomeBrokenPromise, Detail: "pack beta's entrypoint exited 1"},
		})
		if count := strings.Count(printed, guidance); count != 1 {
			t.Errorf("the guidance line was printed %d times, want exactly 1:\n%s", count, printed)
		}
	})
}

// TestInit_DoctorCheckIDsResolveToRegisteredChecks (CLM-020, init-side half).
//
// EVERY id init can print is enumerated STRUCTURALLY — the doctorCheck* identifiers
// init.go actually names — rather than from a list this test wrote, so an id added to
// init.go later is covered without anyone remembering to extend the test.
func TestInit_DoctorCheckIDsResolveToRegisteredChecks(t *testing.T) {
	named := map[string]bool{}
	for _, file := range parseNonTestPackageFiles(t) {
		if !strings.HasSuffix(file.path, "init.go") {
			continue
		}
		ast.Inspect(file.file, func(node ast.Node) bool {
			if ident, ok := node.(*ast.Ident); ok && strings.HasPrefix(ident.Name, "doctorCheck") {
				named[ident.Name] = true
			}
			return true
		})
	}
	if len(named) == 0 {
		t.Fatalf("init.go names no doctor check id constant at all, so nothing was checked")
	}

	// The identifiers init.go names must be exactly the declared constants, and each
	// must resolve through the ONE lookup.
	declared := map[string]string{
		"doctorCheckConfigPresent":  doctorCheckConfigPresent,
		"doctorCheckConfigLoads":    doctorCheckConfigLoads,
		"doctorCheckGitRepository":  doctorCheckGitRepository,
		"doctorCheckPacksInstalled": doctorCheckPacksInstalled,
		"doctorCheckBuildIdentity":  doctorCheckBuildIdentity,
		"doctorCheckToolchainRuns":  doctorCheckToolchainRuns,
		"doctorCheckArtifactLayout": doctorCheckArtifactLayout,
	}
	for identifier := range named {
		value, isDeclared := declared[identifier]
		if !isDeclared {
			t.Errorf("init.go names %q, which is not one of the declared check-id constants", identifier)
			continue
		}
		if _, ok := doctorGuidance(value); !ok {
			t.Errorf("init.go can print the id %q (%s), which resolves to NO registered check", value, identifier)
		}
	}
}

// TestInit_NoDoctorGuidanceForStepsNoRegisteredCheckDiagnoses (CLM-060).
//
// THE REGISTERED CHECK SET IS THE TEST OF "DIAGNOSABLE", AND IT IS NARROW. No registered
// check diagnoses a CI recipe ref that will not resolve or a brownfield CI preserve, so
// init's CI steps must attract NO guidance — which is what keeps doctor from becoming the
// reason init violates SPEC-069's own TestInit_ImplementsNoCIDetectionOrBespokeGuidance.
func TestInit_NoDoctorGuidanceForStepsNoRegisteredCheckDiagnoses(t *testing.T) {
	t.Run("every step succeeded", func(t *testing.T) {
		printed := renderInitReportForTest(t, []initialize.StepReport{
			{Step: initialize.StepToolchain, Outcome: initialize.OutcomeDelivered, Detail: "ran the declared entrypoint"},
			{Step: "ci", Outcome: initialize.OutcomeDelivered, Detail: "applied the pinned recipe"},
		})
		if strings.Contains(printed, "backstop doctor") {
			t.Errorf("an all-succeeding run printed doctor guidance:\n%s", printed)
		}
	})

	t.Run("the only failure is a step no registered check diagnoses", func(t *testing.T) {
		printed := renderInitReportForTest(t, []initialize.StepReport{
			{Step: initialize.StepToolchain, Outcome: initialize.OutcomeDelivered, Detail: "ran the declared entrypoint"},
			{Step: "ci", Outcome: initialize.OutcomeBrokenPromise, Detail: "the recipe ref would not resolve"},
		})
		if strings.Contains(printed, "backstop doctor") {
			t.Errorf("a failed CI step attracted doctor guidance; no registered check diagnoses it:\n%s", printed)
		}
	})

	// A CAPABILITY-ABSENT toolchain step is not a failure either: nothing promised it, so
	// there is no failed promise to diagnose.
	t.Run("the toolchain step reports capability-absent", func(t *testing.T) {
		printed := renderInitReportForTest(t, []initialize.StepReport{
			{Step: initialize.StepToolchain, Outcome: initialize.OutcomeCapabilityAbsent, Detail: "no installed pack declares an entrypoint"},
		})
		if strings.Contains(printed, "backstop doctor") {
			t.Errorf("a capability-absent toolchain step attracted doctor guidance:\n%s", printed)
		}
	})
}
