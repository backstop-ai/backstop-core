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
	"github.com/backstop-ai/backstop-core/pkg/initialize"
	"github.com/backstop-ai/backstop-core/pkg/schema"
)

// init_acceptance_test.go is REQ-019's acceptance bar: a fresh directory, `backstop
// init` installing REAL packs BY GIT REF, then `backstop gate`.
//
// ★ A PACKLESS ACCEPTANCE RUN DOES NOT SATISFY REQ-019. With zero packs installed the
// gate has no pack engines to dispatch and the toolchain step reports capability-absent,
// so a "PASS with zero violations" would assert almost nothing. REQ-018 forbids a
// local-path `--pack` value, so these runs PUBLISH the fixture pack source as a hermetic
// remote and install it BY GENUINE GIT REF — a real clone at a real tag, no network.

// acceptanceMarker is the literal the fixture pack's rule flags.
//
// ★ SHARP EDGE 17, RECORDED HERE BECAUSE IT IS A DEPENDENCY ON A STEP THE SAME RUN
// PERFORMED. This literal necessarily appears in the pack's own pack.yml, which lands
// under `.backstop/packs/`. The acceptance PASS therefore depends on init's GITIGNORE
// step keeping that tree out of the observe step's scope. A run that subtracted the
// `gitignore` capability, or a scope change that stopped honoring ignore entries, would
// have the pack find its own manifest — and the claim would red for a reason having
// nothing to do with init.
const acceptanceMarker = "BACKSTOP_ACCEPTANCE_MARKER_7F3A"

// acceptanceCapabilityAbsentRules are the gate's own class-2 capability-absent
// advisories.
//
// ★ WHY "ZERO VIOLATIONS" IS READ AS ZERO BLOCKING VIOLATIONS FOR THE GREENFIELD
// PROFILE. These three are warning-severity advisories whose own message text ends
// "This advisory is non-blocking (exit 0)", and they are TRUE: a consumer who installed
// only a lint pack genuinely has not wired substantiveness, coverage or contracts.
// Init CANNOT silence them, and that is structural rather than incidental — the
// advisories' own remedy is to declare `enforcement.toolchain`, and REQ-003 states the
// full-SDLC config carries `project` plus `artifact_root` and NO other top-level key.
// So the acceptance bar is asserted as the GREEN VERDICT plus a closed set: every
// violation reported must be one of exactly these three, and ANY other finding — very
// much including a real one from the fixture pack's own engine — fails the claim.
func acceptanceCapabilityAbsentRules() map[string]bool {
	return map[string]bool{
		"substantiveness_capability_absent": true,
		"coverage_capability_absent":        true,
		"contracts_capability_absent":       true,
	}
}

// acceptanceInit runs `backstop init` in a fresh project with the fixture pack
// installed by genuine git ref, and returns the project root.
func acceptanceInit(t *testing.T, fixture string, args ...string) string {
	t.Helper()

	ref, project := initCmdHermeticPack(t, fixture)
	output, code := runInitCommand(t, project, append([]string{"--pack", ref}, args...)...)
	if code != ExitPass {
		t.Fatalf("the acceptance init run exited %d\n%s", code, output)
	}
	return project
}

// acceptanceGate runs the SHIPPED gate over an initialized project, assembled exactly
// as `backstop gate` assembles it, and returns the real result.
func acceptanceGate(t *testing.T, project string) gate.GateResult {
	t.Helper()

	cfg, err := config.LoadConfigFromPath(filepath.Join(project, "backstop.yml"))
	if err != nil {
		t.Fatalf("loading the initialized project's config: %v", err)
	}
	root, rootErr := artifact.ResolveRoot(project, cfg.ArtifactRoot)
	if rootErr != nil {
		t.Fatalf("resolving the artifact root init declared: %v", rootErr)
	}
	scope, scopeErr := gate.ComputeGateScope(project, gate.GateScopeModeDiff, nil)
	if scopeErr != nil {
		t.Fatalf("computing the default gate scope: %v", scopeErr)
	}

	options := []gate.Option{
		gate.WithSteps(buildGateSteps(project, root, scope)),
		gate.WithScope(scope),
	}
	if policy := gatePolicyFromConfig(cfg); len(policy) > 0 {
		options = append(options, gate.WithPolicy(policy))
	}

	result, _ := gate.New(options...).Run(context.Background())
	return result
}

// assertAcceptanceBar asserts the gate reached PASS and reported no violation outside
// the closed capability-absent set.
func assertAcceptanceBar(t *testing.T, profile string, result gate.GateResult) {
	t.Helper()

	sanctioned := acceptanceCapabilityAbsentRules()
	for _, step := range result.Steps {
		for _, violation := range step.Violations {
			if !sanctioned[violation.Rule] {
				t.Fatalf("the %s acceptance gate reported %s in dimension %s: %s\nThe only violations this bar tolerates are the three non-blocking capability-absent advisories, which init is structurally unable to silence.",
					profile, violation.Rule, step.StepName, violation.Message)
			}
		}
	}
	if !result.Pass {
		t.Fatalf("the %s acceptance gate did not PASS.\nsteps: %s", profile, renderStepVerdicts(result))
	}
	if result.StepsFailed != 0 {
		t.Fatalf("the %s acceptance gate had %d failed step(s).\nsteps: %s", profile, result.StepsFailed, renderStepVerdicts(result))
	}
}

// renderStepVerdicts summarizes a gate result for a failure message.
func renderStepVerdicts(result gate.GateResult) string {
	parts := []string{}
	for _, step := range result.Steps {
		parts = append(parts, step.StepName+"="+step.Status)
	}
	return strings.Join(parts, " ")
}

// TestInit_FullSdlcFreshRepoWithRealPacksThenGateReachesPassWithZeroViolations
// (SPEC-069 CLM-091).
func TestInit_FullSdlcFreshRepoWithRealPacksThenGateReachesPassWithZeroViolations(t *testing.T) {
	project := acceptanceInit(t, "acceptance-lint-pack")

	// The greenfield layout really was created, so the profile under test is the one
	// the claim names.
	for _, dir := range []string{".backstop/specs", ".backstop/plans", ".backstop/bundles"} {
		if _, err := os.Stat(filepath.Join(project, dir)); err != nil {
			t.Fatalf("the full-SDLC profile did not create %s: %v", dir, err)
		}
	}

	assertAcceptanceBar(t, "full-SDLC", acceptanceGate(t, project))
}

// TestInit_PackOnlyFreshRepoWithRealPacksThenGateReachesPassWithZeroViolations
// (SPEC-069 CLM-092).
//
// The pack-only profile reaches LITERALLY zero: it sets the five SDLC dimensions to
// `level: off`, which is what suppresses the capability-absent advisories its sibling
// cannot.
func TestInit_PackOnlyFreshRepoWithRealPacksThenGateReachesPassWithZeroViolations(t *testing.T) {
	project := acceptanceInit(t, "acceptance-lint-pack", "--no-sdlc")

	if _, err := os.Stat(filepath.Join(project, ".backstop", "specs")); err == nil {
		t.Fatal("the pack-only profile created an artifact directory; the profile under test is not the one the claim names")
	}

	result := acceptanceGate(t, project)
	assertAcceptanceBar(t, "pack-only", result)

	if result.TotalViolations != 0 {
		t.Fatalf("the pack-only acceptance gate reported %d violations, want zero — this profile turns the five SDLC dimensions off, so nothing should remain",
			result.TotalViolations)
	}
}

// TestInit_AcceptanceGateRunsDispatchedRealPackEngines is NON-VACUITY GUARD (1).
//
// A PASS from a gate with ZERO dispatched pack engines cannot satisfy either acceptance
// claim: it would be a green over an empty kill chain. So the gate each claim asserts
// PASS over must actually have DISPATCHED the installed pack's engine.
func TestInit_AcceptanceGateRunsDispatchedRealPackEngines(t *testing.T) {
	project := acceptanceInit(t, "acceptance-lint-pack")

	// The pack really is installed, by git ref, with its manifest on disk.
	manifest := filepath.Join(project, ".backstop", "packs", remoteE2EOrg, "acceptance-lint-pack", "pack.yml")
	body, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatalf("the acceptance pack was not installed: %v", err)
	}
	if !strings.Contains(string(body), acceptanceMarker) {
		t.Fatal("the installed manifest does not carry the marker, so the dispatch below would assert nothing")
	}

	result := acceptanceGate(t, project)

	dispatched := false
	for _, step := range result.Steps {
		if step.StepName == "pack_engines" && step.Status != "skipped" {
			dispatched = true
		}
	}
	if !dispatched {
		t.Fatalf("the acceptance gate did not dispatch pack_engines at all.\nsteps: %s", renderStepVerdicts(result))
	}
}

// TestInit_AcceptanceDispatchedEngineCanProduceAViolation is NON-VACUITY GUARD (2).
//
// Dispatch alone is too weak a bar: a fixture engine with no real content satisfies
// "an engine ran" while asserting nothing. So the SAME pack, installed the SAME way,
// over a fixture project deliberately containing the marker its rule flags, must
// produce at least one violation attributed to that engine — which makes the acceptance
// PASS a pack that CAN go red and did not.
func TestInit_AcceptanceDispatchedEngineCanProduceAViolation(t *testing.T) {
	project := acceptanceInit(t, "acceptance-lint-pack")

	// Plant the marker where the gate WILL look: a tracked-scope file in the project,
	// not under the gitignored pack tree.
	if err := os.WriteFile(filepath.Join(project, "carries-the-marker.txt"),
		[]byte("this file deliberately contains "+acceptanceMarker+"\n"), 0o644); err != nil {
		t.Fatalf("planting the marker: %v", err)
	}

	result := acceptanceGate(t, project)

	if !acceptanceEngineFired(result) {
		t.Fatalf("the pack's engine produced NO violation over a file that carries the very marker its rule flags. The acceptance PASS is therefore not a pack that CAN go red — it is a pack that cannot.\nsteps: %s",
			renderStepVerdicts(result))
	}
}

// TestInit_AcceptanceEngineCanProduceAViolationUnderBothProfiles is NON-VACUITY GUARD
// (3).
//
// The demonstration must run under BOTH profiles, because a violation that could not
// reach the verdict under the pack-only profile's enforcement policy would leave that
// profile's acceptance claim unfalsifiable even with guards (1) and (2) satisfied. This
// is the whole reason the fixture engine's gate_type is `lint`: `lint` has no dedicated
// step, so its rules survive the dedicated-step exclusion and dispatch through the
// generic pack_engines path — a dimension NEITHER profile disables.
func TestInit_AcceptanceEngineCanProduceAViolationUnderBothProfiles(t *testing.T) {
	for _, profile := range []struct {
		name string
		args []string
	}{
		{"full-SDLC", nil},
		{"pack-only", []string{"--no-sdlc"}},
	} {
		t.Run(profile.name, func(t *testing.T) {
			project := acceptanceInit(t, "acceptance-lint-pack", profile.args...)
			if err := os.WriteFile(filepath.Join(project, "carries-the-marker.txt"),
				[]byte("this file deliberately contains "+acceptanceMarker+"\n"), 0o644); err != nil {
				t.Fatalf("planting the marker: %v", err)
			}

			result := acceptanceGate(t, project)
			if !acceptanceEngineFired(result) {
				t.Fatalf("under the %s profile the pack's engine produced no violation over a file carrying its marker; re-tagging the engine's gate_type would reproduce exactly this, invisibly.\nsteps: %s",
					profile.name, renderStepVerdicts(result))
			}
		})
	}
}

// acceptanceEngineFired reports whether the fixture pack's own rule produced a
// violation.
func acceptanceEngineFired(result gate.GateResult) bool {
	for _, step := range result.Steps {
		for _, violation := range step.Violations {
			if strings.Contains(violation.Rule, "acceptance-marker") {
				return true
			}
		}
	}
	return false
}

// TestInit_ArtifactsUnderTheScaffoldedRootAreDiscoveredAndGated (SPEC-069 CLM-103).
//
// ★ THE DELIBERATE TRIPWIRE THAT STOPS THIS SPEC BEING IMPLEMENTED INTO A VACUOUS
// GREEN. An artifact placed under the root init created must actually be DISCOVERED and
// validated, and a deliberately INVALID artifact placed there must make validation
// FAIL. Init that scaffolds `.backstop/` before discovery can resolve it manufactures
// exactly the silent-undiscovery false green it exists to prevent — and both halves are
// asserted, because a discovery that found the file but validated nothing would satisfy
// the first alone.
func TestInit_ArtifactsUnderTheScaffoldedRootAreDiscoveredAndGated(t *testing.T) {
	project := t.TempDir()
	if output, code := runInitCommand(t, project); code != ExitPass {
		t.Fatalf("the init run exited %d\n%s", code, output)
	}

	cfg, err := config.LoadConfigFromPath(filepath.Join(project, "backstop.yml"))
	if err != nil {
		t.Fatalf("loading the generated config: %v", err)
	}
	root, rootErr := artifact.ResolveRoot(project, cfg.ArtifactRoot)
	if rootErr != nil {
		t.Fatalf("resolving the artifact root init declared: %v", rootErr)
	}

	t.Run("an artifact under the scaffolded root is DISCOVERED", func(t *testing.T) {
		writeAcceptanceIssue(t, project, "ISSUE-001-discovered.issue.md", validAcceptanceIssue())

		found, discoverErr := DiscoverArtifacts(root, nil, artifact.NonCorpusDirs{})
		if discoverErr != nil {
			t.Fatalf("discovering artifacts under %s: %v", root.Path, discoverErr)
		}
		if len(found) == 0 {
			t.Fatalf("ZERO artifacts were discovered under the root init created (%s). A consumer laid out exactly the way init lays them out would have every artifact skipped, and both `artifact validate` and the gate would report green having checked nothing.",
				root.Path)
		}
	})

	t.Run("a deliberately INVALID artifact under that root FAILS validation", func(t *testing.T) {
		// Without this half, a discovery that found the file and asserted nothing
		// against it would satisfy the claim above.
		// It DECLARES its schema so routing SUCCEEDS, and then violates that schema. An
		// artifact that failed to ROUTE would error out before validation ever ran,
		// which would prove the router works rather than that the corpus beneath the
		// scaffolded root is genuinely being asserted against anything.
		writeAcceptanceIssue(t, project, "ISSUE-002-invalid.issue.md", invalidAcceptanceIssue())

		// SchemaFS and the cohort travel WITH the config, exactly as the shipped
		// `artifact validate` builds it — a nil schema filesystem is not a degraded
		// validation, it is an unusable one.
		cohort, cohortErr := schema.ComputeCohort(SchemaFS)
		if cohortErr != nil {
			t.Fatalf("computing the schema cohort: %v", cohortErr)
		}
		result, validateErr := ValidateArtifacts(ValidateConfig{
			ProjectRoot: project,
			Root:        root,
			All:         true,
			SchemaFS:    SchemaFS,
			Cohort:      cohort,
		})
		if validateErr != nil {
			t.Fatalf("validating the scaffolded root: %v", validateErr)
		}
		if result.ArtifactsFound == 0 {
			t.Fatal("validation discovered no artifacts, so its verdict is about nothing")
		}
		if result.Pass {
			t.Fatalf("validation PASSED over a deliberately invalid artifact under the scaffolded root (%d found, %d asserted). That is the silent-undiscovery false green this tripwire exists to catch.",
				result.ArtifactsFound, result.ArtifactsAsserted)
		}
	})
}

// writeAcceptanceIssue lays an artifact under the scaffolded root's issues directory.
func writeAcceptanceIssue(t *testing.T, project, name, body string) {
	t.Helper()
	dir := filepath.Join(project, ".backstop", "issues")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("creating %s: %v", dir, err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatalf("writing %s: %v", name, err)
	}
}

// validAcceptanceIssue is a structurally complete issue artifact.
func validAcceptanceIssue() string {
	return `---
title: "An artifact placed under the root init created"
number: ISSUE-001
created: "2026-08-15"
status: open
schema_version: issue/v1
severity: low
category: tech-debt
---

# ISSUE-001: An artifact placed under the root init created

## Summary

This artifact exists so discovery has something to find beneath the layout init
scaffolded. Its content is irrelevant; its LOCATION is the whole point.

## Impact

None. It is a fixture.
`
}

// invalidAcceptanceIssue is an artifact that ROUTES and then FAILS.
//
// It declares its schema (so the router resolves it) and violates that schema (so
// validation has something to reject). The distinction is the whole point of the
// sub-test that uses it: an artifact missing `schema_version` errors out during
// ROUTING, before validation runs at all, and a test built on one would show that the
// router works while saying nothing about whether artifacts under the scaffolded root
// are actually asserted against their schema.
func invalidAcceptanceIssue() string {
	return strings.Join([]string{
		"---",
		"schema_version: issue/v1",
		"title: \"Declares its schema and then violates it\"",
		"number: ISSUE-002",
		"status: not-a-status-the-schema-declares",
		"---",
		"",
		"# ISSUE-002",
		"",
	}, "\n")
}

// TestInit_ScaffoldedProjectTurnsAnEmptyProjectEntrypointFailureGreen (SPEC-069
// CLM-139).
//
// ★ DD-7's MOTIVATING EVIDENCE, BOUND TO A TEST RATHER THAN TO A STORY. The SAME
// declared entrypoint that exits NON-ZERO over a project with no source file exits ZERO
// over the same project once the scaffold recipe has run, with the scaffolded file as
// the ONLY difference between the two runs.
//
// This is what regresses SILENTLY if a later refactor moves the scaffold step after the
// toolchain step: the step list would still read consistently, and the toolchain step
// would simply start running over an empty project again.
func TestInit_ScaffoldedProjectTurnsAnEmptyProjectEntrypointFailureGreen(t *testing.T) {
	ref, project := initCmdHermeticPack(t, "scaffold-entrypoint-pack")
	packName := remoteE2EOrg + "/scaffold-entrypoint-pack"

	// RUN A — the pack is installed, the scaffold recipe is NOT applied. The declared
	// entrypoint runs over a project with no source file.
	beforeOutput, beforeCode := runInitCommand(t, project, "--pack", ref)

	toolchainBefore := reportLineFor(t, beforeOutput, "toolchain")
	if !strings.Contains(toolchainBefore, "not delivered") {
		t.Fatalf("the declared entrypoint did NOT fail over a project with no source file, so this test cannot demonstrate the flip.\n%s", toolchainBefore)
	}
	if beforeCode == ExitPass {
		t.Fatalf("init exited 0 despite an entrypoint that could not verify the toolchain\n%s", beforeOutput)
	}

	// RUN B — the SAME project, the SAME entrypoint, with the scaffold recipe applied.
	afterOutput, afterCode := runInitCommand(t, project, "--scaffold", packName+":first-source@1.0.0")

	toolchainAfter := reportLineFor(t, afterOutput, "toolchain")
	if strings.Contains(toolchainAfter, "not delivered") {
		t.Fatalf("the declared entrypoint STILL fails after the scaffold recipe ran.\nbefore: %s\nafter:  %s", toolchainBefore, toolchainAfter)
	}
	if afterCode != ExitPass {
		t.Fatalf("init exited %d after the scaffold recipe delivered the source file\n%s", afterCode, afterOutput)
	}

	// THE SCAFFOLDED FILE IS THE ONLY DIFFERENCE, asserted rather than assumed.
	scaffolded := filepath.Join(project, "src", "first-source.txt")
	if _, err := os.Stat(scaffolded); err != nil {
		t.Fatalf("the scaffold recipe's declared target is not on disk: %v", err)
	}
}

// TestInit_ScaffoldedFileIsOnDiskWhenTheRealProberRuns is CLM-138's production-shape
// twin: the pkg/initialize claim proves the ORDER against a fake prober that stats the
// file, and this one proves the same boundary with the REAL prober executing a REAL
// pack-declared entrypoint.
//
// It is additive, and it is what makes the ordering claim about the shipped command
// rather than about the runner in isolation.
func TestInit_ScaffoldedFileIsOnDiskWhenTheRealProberRuns(t *testing.T) {
	ref, project := initCmdHermeticPack(t, "scaffold-entrypoint-pack")
	packName := remoteE2EOrg + "/scaffold-entrypoint-pack"

	// ONE invocation supplying BOTH --pack and --scaffold: the scaffold step (6) runs
	// before the toolchain step (7) within that single run, which is the boundary under
	// test. If the two were swapped, the entrypoint would run over an empty project and
	// report a failure.
	output, code := runInitCommand(t, project, "--pack", ref, "--scaffold", packName+":first-source@1.0.0")

	toolchain := reportLineFor(t, output, "toolchain")
	if strings.Contains(toolchain, "not delivered") {
		t.Fatalf("the toolchain step ran BEFORE the scaffold step delivered the source file, which re-manufactures the empty-project entrypoint failure the ordering exists to prevent.\n%s", toolchain)
	}
	if code != ExitPass {
		t.Fatalf("the combined run exited %d\n%s", code, output)
	}
}

// TestInit_AcceptanceToolchainStepIsCapabilityAbsentForTheLintOnlyPack records a
// CONSEQUENCE of the acceptance fixture rather than a defect.
//
// The acceptance pack declares no test or build engine, so the acceptance runs' toolchain
// step reports capability-absent. REQ-011's execution path is proven by its own claims
// against manifests that DO declare those gate types, not by this run — and stating it
// here keeps a future reader from reading the absence as coverage.
func TestInit_AcceptanceToolchainStepIsCapabilityAbsentForTheLintOnlyPack(t *testing.T) {
	ref, project := initCmdHermeticPack(t, "acceptance-lint-pack")

	output, code := runInitCommand(t, project, "--pack", ref)
	if code != ExitPass {
		t.Fatalf("the acceptance init run exited %d\n%s", code, output)
	}

	toolchain := reportLineFor(t, output, "toolchain")
	if !strings.Contains(toolchain, "capability absent") {
		t.Fatalf("the acceptance pack declares no test or build engine, so the toolchain step must report capability-absent.\n%s", toolchain)
	}

	// And the prober really was the production one, over the really-installed corpus.
	prober := &packToolchainProber{Runner: &check.ExecCommandRunner{Dir: project}}
	reports, probeErr := prober.Probe(project)
	if probeErr != nil {
		t.Fatalf("the production prober errored over the initialized project: %v", probeErr)
	}
	if len(reports) != 1 || reports[0].Outcome != initialize.OutcomeCapabilityAbsent {
		t.Fatalf("the production prober reported %+v, want a single capability-absent report", reports)
	}
}
