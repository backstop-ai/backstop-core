package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/artifact"
	"github.com/backstop-ai/backstop-core/pkg/check"
	"github.com/backstop-ai/backstop-core/pkg/config"
	"github.com/backstop-ai/backstop-core/pkg/gate"
	"github.com/backstop-ai/backstop-core/pkg/pack"
	"github.com/backstop-ai/backstop-core/pkg/pack/engine"
)

// gate_test_only_diff_verdict_e2e_test.go is ISSUE-118's ACCEPTANCE SURFACE: a diff
// whose changed files are ENTIRELY test files, a mandated test that genuinely
// EXISTS and genuinely FAILED, and a gate run that must come back non-zero.
//
// WHY THIS LIVES IN cmd/backstop AND NOT pkg/gate. The collector, the step
// ordering, and buildGateSteps are all cmd/backstop's; pkg/gate cannot reach them,
// and the whole defect is a wiring defect — the verdict is computed correctly and
// then discarded between two steps. A pure pkg/gate test cannot see that.
//
// WHY THE DISPATCH IS SEAMED BUT THE PACK SET IS REAL — READ BOTH HALVES.
//   - REAL: the pack manifests are read OFF DISK from committed fixture workspaces,
//     so the `gate_type: test` capability is genuinely DECLARED and genuinely
//     DISCOVERED. That is what makes the capability-absent branch below reachable
//     by an honest input rather than by an injected boolean.
//   - SEAMED: only the dispatch OUTPUT. The root `.backstop/packs/` is gitignored,
//     a stub engine command would not clear the trust allowlist, and a `grep`
//     engine cannot be made to emit go-test SARIF. So the seam delivers the REAL
//     converter output (TASK-001's derived artifact) with its REAL bare-basename
//     paths, arriving exactly as production delivers it.
//
// Between TestPackDispatch_StampsDeclaredGateTypeOnBridgedViolations — which drives
// the REAL runFindingsEngine over the REAL committed capture through the REAL
// committed converter — and this test, every hop from captured tool output to gate
// verdict is covered by something real.

// verdictE2ESarif returns the derived REAL converter output as gate.Violations
// stamped the way a `gate_type: test` binding's dispatch stamps them. The paths it
// carries are the converter's own bare basenames (widget_test.go), plus one finding
// with no path at all.
func verdictE2ESarif(t *testing.T) []gate.Violation {
	t.Helper()
	path := filepath.Join(repoRoot(t), "pkg", "gate", "testdata", "test-verdict", "go-test-failure.sarif.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading the derived converter output: %v", err)
	}
	parsed, err := check.ParsePackFindings(raw)
	if err != nil {
		t.Fatalf("parsing the derived converter output: %v", err)
	}
	out := make([]gate.Violation, 0, len(parsed))
	for _, v := range parsed {
		out = append(out, gate.Violation{
			Rule:        pack.NamespacedRuleID("acme/verdict-probe", v.Rule),
			File:        v.File,
			Line:        v.Line,
			Message:     v.Message,
			Severity:    v.Severity,
			SourcePack:  "acme/verdict-probe",
			GateType:    engine.GateTypeTest.String(),
			ProjectWide: false,
		})
	}
	return out
}

// seamDispatch replaces the dispatch seam with one returning violations, restoring
// it on cleanup.
func seamDispatch(t *testing.T, violations []gate.Violation) {
	t.Helper()
	orig := dispatchPackEnginesFn
	t.Cleanup(func() { dispatchPackEnginesFn = orig })
	dispatchPackEnginesFn = func([]*pack.Manifest, string, string, *gate.GateScope, check.CommandRunner) ([]gate.Violation, error) {
		return violations, nil
	}
}

// verdictE2EWorkspace copies a committed fixture workspace into a temp dir (never
// mutating committed testdata) and adds the spec + test file the acceptance case
// needs. It returns the project root and the repo-relative path of the test file.
func verdictE2EWorkspace(t *testing.T, fixture string) (string, string) {
	t.Helper()
	project := t.TempDir()
	copyTree(t, filepath.Join(repoRoot(t), "cmd", "backstop", "testdata", fixture), project)

	// A spec that genuinely VALIDATES and is `implemented`, mandating the test that
	// will be reported as failing. It must validate: artifact validation is the
	// first assembled step, and a broken spec would red the run for the wrong
	// reason and hollow out assertion 1.
	specDir := filepath.Join(project, "specs")
	if err := os.MkdirAll(specDir, 0o755); err != nil {
		t.Fatalf("creating specs dir: %v", err)
	}
	spec := `---
title: "Verdict E2E Spec"
number: VERD-001
created: "2026-01-01"
status: implemented
schema_version: spec/v1
spec_version: 1.0.0

implementation:
  summary: verdict e2e
  package: pkg/widget

verification:
  level: unit
  test_command: go test ./...
  coverage_threshold: 80

requirements:
  - id: REQ-001
    text: a failing mandated test reds the gate
    supports: x:REQ-001

claims:
  - id: CLM-001
    requirement: REQ-001
    text: the mandated test's verdict is joined by name
    tests:
      - TestWidgetFrobnicate
---

# Verdict E2E Spec
`
	if err := os.WriteFile(filepath.Join(specDir, "verdict.spec.md"), []byte(spec), 0o644); err != nil {
		t.Fatalf("writing the spec fixture: %v", err)
	}

	// COMMIT THE BASELINE BEFORE PLANTING THE TEST FILE. The diff scope is resolved
	// from GIT, not from a file list handed to ComputeGateScope — so the only way to
	// get a genuine diff-mode scope containing exactly one file is to make everything
	// else tracked and clean first. Committing here, and writing the test file after,
	// is what makes the planted file the SOLE entry `git ls-files --others` appends.
	// (Precedent: gitInitCommitAll / newDiffScopedPackGateProject in gate_scope_test.go.)
	gitInitCommitAll(t, project)

	// The mandated test EXISTS, in a SUBDIRECTORY, matching the fixture pack's own
	// declared test glob and test-name pattern — so it is both discoverable and
	// name-verifiable. This is the "exists, is substantive, and is RED" case that
	// reads as a clean pass at HEAD.
	relTest := filepath.Join("pkg", "widget", "widget_test.go")
	full := filepath.Join(project, relTest)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("creating the test package dir: %v", err)
	}
	body := "package widget\n\nimport \"testing\"\n\nfunc TestWidgetFrobnicate(t *testing.T) {\n\tif 5 != 7 {\n\t\tt.Fatal(\"expected 5, got 7\")\n\t}\n}\n"
	if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
		t.Fatalf("writing the mandated test file: %v", err)
	}
	return project, filepath.ToSlash(relTest)
}

// runVerdictE2EGate assembles the SHIPPED steps over project and runs them, exactly
// as `backstop gate` does.
func runVerdictE2EGate(t *testing.T, project string, scope *gate.GateScope) gate.GateResult {
	t.Helper()
	cfg, err := config.LoadConfigFromPath(filepath.Join(project, "backstop.yml"))
	if err != nil {
		t.Fatalf("loading the fixture config: %v", err)
	}
	root, rootErr := artifact.ResolveRoot(project, cfg.ArtifactRoot)
	if rootErr != nil {
		t.Fatalf("resolving the artifact root: %v", rootErr)
	}
	result, _ := gate.New(
		gate.WithSteps(buildGateSteps(project, root, scope)),
		gate.WithScope(scope),
	).Run(context.Background())
	return result
}

// verdictStepNamed returns the result of the named step, and whether the run
// reached it.
// Asserting PRESENCE by name matters: an aborted run leaves the step absent, which
// an empty-violation-list check would silently read as clean.
func verdictStepNamed(result gate.GateResult, name string) (gate.StepResult, bool) {
	for _, s := range result.Steps {
		if s.StepName == name {
			return s, true
		}
	}
	return gate.StepResult{}, false
}

// TestE2E_GateRedsOnFailingMandatedTestInTestOnlyDiff (CLM-005, CLM-004, CLM-003):
// the assembled gate goes RED on a genuinely failing mandated test when the diff's
// changed files are ENTIRELY test files. This is ISSUE-118's own reproduction.
func TestE2E_GateRedsOnFailingMandatedTestInTestOnlyDiff(t *testing.T) {
	project, relTest := verdictE2EWorkspace(t, "test-verdict-e2e")
	seamDispatch(t, verdictE2ESarif(t))

	// ISSUE-118's shape: a REAL diff-mode scope, resolved from git, whose files are
	// ENTIRELY test files — that one planted file and nothing else. Asserted
	// explicitly so a later edit cannot quietly turn it into a mixed diff and void
	// the fixture.
	scope, err := gate.ComputeGateScope(project, gate.GateScopeModeDiff, nil)
	if err != nil {
		t.Fatalf("computing the diff scope: %v", err)
	}
	if len(scope.Files) != 1 || scope.Files[0] != relTest {
		t.Fatalf("the diff scope must be exactly the one planted test file, got %v", scope.Files)
	}
	classifier := gate.NewSourceClassifier([]string{"**/*.go"}, []string{"**/*_test.go"})
	for _, f := range scope.Files {
		if !classifier.IsTestFile(f) {
			t.Fatalf("the diff must be ENTIRELY test files; %q is not classified as one", f)
		}
	}

	result := runVerdictE2EGate(t, project, scope)

	// 1. The run must not pass, and must have REACHED the verdict rather than
	//    aborting early on a config error.
	if result.Pass {
		t.Fatalf("the gate PASSED with a genuinely failing mandated test in an all-test-file diff — this is ISSUE-118.\nsteps: %s", renderStepVerdicts(result))
	}
	for _, s := range result.Steps {
		if s.ConfigErr {
			t.Fatalf("step %s reported ConfigErr, so the run aborted before the verdict rather than reaching it: %#v", s.StepName, s.Violations)
		}
	}

	verify, reached := verdictStepNamed(result, gate.StepTestVerification)
	if !reached {
		t.Fatalf("the run never reached test_verification.\nsteps: %s", renderStepVerdicts(result))
	}

	// 2. test_verification carries the blocking verdict violation naming the test.
	var verdict gate.Violation
	found := false
	for _, v := range verify.Violations {
		if v.Rule == "mandated_test_failed" {
			verdict, found = v, true
		}
	}
	if !found {
		t.Fatalf("test_verification reported no mandated_test_failed violation: %#v", verify.Violations)
	}
	if !strings.Contains(verdict.Message, "TestWidgetFrobnicate") {
		t.Fatalf("the verdict does not name the failing mandated test: %q", verdict.Message)
	}
	if verdict.Severity != "critical" {
		t.Fatalf("verdict Severity = %q, want \"critical\"", verdict.Severity)
	}

	// 3. Attribution is the mandated test's OWN path, never the converter's
	//    unresolvable bare basename.
	if verdict.File == "widget_test.go" {
		t.Fatalf("the finding's bare-basename path leaked into attribution: %q", verdict.File)
	}
	if verdict.File != relTest {
		t.Fatalf("verdict File = %q, want the mandated test's own path %q", verdict.File, relTest)
	}

	// 4. THE SCOPE-FILTER FENCE — deliberately NOT "the ISSUE-129 fence". The seam
	//    replaces dispatch wholesale, so NO binding declaration
	//    (exempt_from_scope_filter included) can reach this test at all. What this
	//    pins is that THIS lane leaves activeScope.FilterViolations and the
	//    pack_engines step's own reported set exactly as they are: the seamed
	//    finding, whose path resolves nowhere, is still filtered out of what
	//    pack_engines reports. If a later change makes pack_engines red here too,
	//    this assertion tells the next reader which lane did it.
	engines, reachedEngines := verdictStepNamed(result, "pack_engines")
	if !reachedEngines {
		t.Fatalf("the run never reached pack_engines.\nsteps: %s", renderStepVerdicts(result))
	}
	if len(engines.Violations) != 0 {
		t.Fatalf("pack_engines' own reported violations changed; the scope filter is not this lane's to alter: %#v", engines.Violations)
	}
}

// TestE2E_VerdictCapabilityAbsentIsAdvisoryNotSilentPass (CLM-006): due mandated
// tests exist, but NO installed pack declares a `gate_type: test` engine. The
// dimension must say so, non-blockingly, rather than reporting an unqualified pass
// having verified nothing but spelling.
//
// THE SECOND FIXTURE IS WHAT MAKES THIS HONEST. Capability presence is derived from
// the REAL declared bindings read out of the installed manifests, so a seam-only
// injection cannot reach this branch; the workspace declaring no `gate_type: test`
// binding anywhere is the only input that does.
//
// ★ ASSERTED ON THE STEP, NOT ON result.Pass — AND THAT IS DELIBERATE. The in-tree
// precedent for the closed-rule-set shape is init_acceptance_test.go's
// acceptanceCapabilityAbsentRules + its step/violation loop. This test copies that
// SHAPE and deliberately LEAVES BEHIND that precedent's companion result.Pass /
// result.StepsFailed assertions: buildGateSteps assembles twelve steps
// unconditionally, and drift, requirement traceability and ledger integrity all have
// opinions about this fixture's implemented-spec-with-a-mandated-test shape — which
// is the very input required to reach the branch under test. Hanging CLM-006 on
// eleven unrelated dimensions being globally clean over a hand-built workspace is a
// flake waiting to happen, and is not what CLM-006 claims.
func TestE2E_VerdictCapabilityAbsentIsAdvisoryNotSilentPass(t *testing.T) {
	project, relTest := verdictE2EWorkspace(t, "test-verdict-absent-e2e")
	// No test-typed violations: the workspace declares no test-verdict engine, so
	// production would have none to collect either.
	seamDispatch(t, nil)

	scope, err := gate.ComputeGateScope(project, gate.GateScopeModeDiff, nil)
	if err != nil {
		t.Fatalf("computing the diff scope: %v", err)
	}
	if len(scope.Files) != 1 || scope.Files[0] != relTest {
		t.Fatalf("the diff scope must be exactly the one planted test file, got %v", scope.Files)
	}

	result := runVerdictE2EGate(t, project, scope)

	verify, reached := verdictStepNamed(result, gate.StepTestVerification)
	if !reached {
		t.Fatalf("the run never reached test_verification.\nsteps: %s", renderStepVerdicts(result))
	}
	if verify.ConfigErr {
		t.Fatalf("test_verification reported ConfigErr; an un-adopted capability is a warning, not a config error")
	}
	if verify.Status != "warning" {
		t.Fatalf("test_verification Status = %q, want \"warning\"; violations: %#v", verify.Status, verify.Violations)
	}

	advisory := false
	for _, v := range verify.Violations {
		if v.Rule == "test_verification_verdict_capability_absent" {
			advisory = true
			if v.Severity != "warning" {
				t.Fatalf("the advisory Severity = %q, want \"warning\"", v.Severity)
			}
		}
		if v.Rule == "mandated_test_failed" {
			t.Fatalf("a verdict violation was emitted with no verdict engine declared: %#v", v)
		}
		// EVERY violation this dimension reports must be non-blocking, not just the
		// advisory — an extra blocking finding sneaking into the dimension fails the
		// claim.
		if !strings.EqualFold(strings.TrimSpace(v.Severity), "warning") {
			t.Fatalf("test_verification reported a BLOCKING violation on the capability-absent path: %s (%s)", v.Rule, v.Severity)
		}
		// The name-presence check must still have reported cleanly: the mandated test
		// genuinely exists in the workspace.
		if v.Rule == "test_verification" {
			t.Fatalf("the name-presence check reported a violation for a test that exists: %#v", v)
		}
	}
	if !advisory {
		t.Fatalf("no test_verification_verdict_capability_absent advisory; an unqualified pass here is the silent green CLM-006 forbids: %#v", verify.Violations)
	}
}
