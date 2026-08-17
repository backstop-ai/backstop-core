package main

import (
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/pack"
	"github.com/backstop-ai/backstop-core/pkg/pack/engine"
)

// doctor_engine_tools_test.go pins the five dispositions of doctor's engine-tools
// check (ISSUE-134). Every case that CAN be staged is driven through stageDoctorProject
// and the real doctor command, so the check is exercised the way the command exercises
// it; the trust-refusal case is the one exception, for the structural reason recorded on
// TestDoctor_EngineToolTrustRefusalFails.
//
// THE DEFECT THESE INVERT: before ISSUE-134, `backstop doctor` reported an all-green
// exit 0 on a project whose pack-declared, RULE-BOUND findings-engine tool was absent
// from PATH — the same project `backstop gate` refused at exit 2. Doctor consumed only
// packEntrypointProber, which selects by STAGE (test/build), so a findings engine was
// never selected, never probed, never mentioned.

// TestDoctor_EngineToolAbsentIsReported (CLM-001). Status fail, and the report names
// the probed argv[0], the declaring pack AND the engine key — asserted separately,
// because a status-only assertion passes on a report that tells the operator nothing
// actionable, which is half the defect.
func TestDoctor_EngineToolAbsentIsReported(t *testing.T) {
	project := stageDoctorProject(t, "engine-tool-absent")
	withBinaryResolver(t) // nothing resolves

	payload, _ := runDoctorJSON(t, project)

	if status := payload.statuses()[doctorCheckEngineTools]; status != doctorStatusFail {
		t.Fatalf("engine-tools status = %q, want %q — this is the exit-0 all-green report ISSUE-134 recorded", status, doctorStatusFail)
	}

	message := payload.field(t, doctorCheckEngineTools, "message")
	for _, want := range []string{"backstop-absent-engine-134", "backstop/engine-tool-absent", "absent-findings"} {
		if !strings.Contains(message, want) {
			t.Errorf("engine-tools message does not name %q; a report that cannot say WHAT is missing and WHO declared it sends the reader hunting through every installed pack\nmessage: %s", want, message)
		}
	}

	// THE REMEDIATION IS THE GATE'S OWN WORDS, BY CONSTRUCTION. Doctor reuses
	// absentToolMessage rather than rendering a second text, which is what keeps
	// doctor's advice and the gate's refusal from drifting apart.
	wantRemediation := absentToolMessage(requiredTool{
		name:   "backstop-absent-engine-134",
		pack:   "backstop/engine-tool-absent",
		engine: "absent-findings",
	})
	remediation := payload.field(t, doctorCheckEngineTools, "remediation")
	if !strings.Contains(remediation, wantRemediation) {
		t.Errorf("engine-tools remediation does not carry absentToolMessage's exact rendering\ngot:  %s\nwant: %s", remediation, wantRemediation)
	}
}

// TestDoctor_EngineToolPresentPasses (CLM-001). With the resolver reporting the tool
// found the status is pass, and the message still NAMES the probed tool — the
// anti-vacuity arm, so a check that probed nothing at all cannot pass this.
func TestDoctor_EngineToolPresentPasses(t *testing.T) {
	project := stageDoctorProject(t, "engine-tool-present")
	withBinaryResolver(t, "backstop-present-engine-134")

	payload, _ := runDoctorJSON(t, project)

	if status := payload.statuses()[doctorCheckEngineTools]; status != doctorStatusPass {
		t.Fatalf("engine-tools status = %q, want %q", status, doctorStatusPass)
	}
	message := payload.field(t, doctorCheckEngineTools, "message")
	if strings.TrimSpace(message) == "" {
		t.Fatal("engine-tools passed with an EMPTY message; a check that probed nothing reports exactly this")
	}
	if !strings.Contains(message, "backstop-present-engine-134") {
		t.Errorf("engine-tools passed without naming the probed tool: %s", message)
	}
}

// TestDoctor_EngineToolUnboundIsNotProbed (CLM-002). An engine no rule binds is never
// dispatched, so its tool's absence must not be reported. A walk over manifest.Engines
// reds here — and it would make doctor fail on a project the gate passes, which is a
// worse defect than the one being fixed.
func TestDoctor_EngineToolUnboundIsNotProbed(t *testing.T) {
	project := stageDoctorProject(t, "engine-tool-unbound")
	withBinaryResolver(t, "backstop-present-engine-134")

	payload, _ := runDoctorJSON(t, project)

	if status := payload.statuses()[doctorCheckEngineTools]; status != doctorStatusPass {
		t.Fatalf("engine-tools status = %q, want %q — the unbound engine's tool must not be probed", status, doctorStatusPass)
	}
	entry := payload.find(t, doctorCheckEngineTools)
	for _, key := range []string{"message", "remediation"} {
		text, _ := entry[key].(string)
		if strings.Contains(text, "backstop-unbound-engine-134") {
			t.Errorf("engine-tools %s names the UNBOUND engine's tool; the walk is going through manifest.Engines rather than through rules\n%s", key, text)
		}
	}
}

// TestDoctor_EngineToolNoneWarnsRatherThanPasses (CLM-006). A GATHERED pack set that
// binds no engine tool warns; it never passes silently. Mirrors checkToolchainRuns'
// outcome (d).
func TestDoctor_EngineToolNoneWarnsRatherThanPasses(t *testing.T) {
	project := stageDoctorProject(t, "engine-tool-none")
	withBinaryResolver(t)

	payload, _ := runDoctorJSON(t, project)

	status := payload.statuses()[doctorCheckEngineTools]
	if status == doctorStatusPass {
		t.Fatal("engine-tools PASSED on a pack set that binds no engine tool; a silent pass is the shape doctor exists to prevent")
	}
	if status != doctorStatusWarn {
		t.Fatalf("engine-tools status = %q, want %q", status, doctorStatusWarn)
	}
	message := payload.field(t, doctorCheckEngineTools, "message")
	if !strings.Contains(message, "no installed pack") {
		t.Errorf("engine-tools warn message must say no installed pack binds an engine tool, got: %s", message)
	}
}

// TestDoctor_EngineToolSkipsOnUngatheredPackSet (CLM-006). THREE subtests, not one.
//
// gatherDoctorContext calls loadInstalledPacks ONLY when the config was both discovered
// AND loaded, so an absent or unloadable backstop.yml leaves Packs nil with PacksErr
// ALSO nil. A PacksErr-only predicate reports "no installed pack binds an engine tool"
// on a project whose packs were never looked at — a wrong answer that reads like a right
// one — and passes two of these three subtests.
func TestDoctor_EngineToolSkipsOnUngatheredPackSet(t *testing.T) {
	cases := []struct {
		name    string
		fixture string
	}{
		{name: "config absent", fixture: "no-config"},
		{name: "config unloadable", fixture: "config-malformed"},
		{name: "pack set unloadable", fixture: "config-missing-pack"},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			project := stageDoctorProject(t, testCase.fixture)
			withBinaryResolver(t)

			payload, _ := runDoctorJSON(t, project)

			if status := payload.statuses()[doctorCheckEngineTools]; status != doctorStatusSkipped {
				t.Fatalf("engine-tools status = %q on %s, want %q", status, testCase.fixture, doctorStatusSkipped)
			}
			message := payload.field(t, doctorCheckEngineTools, "message")
			if !strings.Contains(message, doctorCheckPacksInstalled) {
				t.Errorf("the skip must name %s as the owner of the condition, got: %s", doctorCheckPacksInstalled, message)
			}
		})
	}
}

// TestDoctor_EngineToolTrustRefusalFails (CLM-006). A rule-bound engine whose
// `provision:` names a tool the allowlist does not admit fails, and the report says the
// tool was REFUSED BEFORE IT RAN rather than that it was missing.
//
// This is doctor's OWN disposition of a value the gate turns into a config error. The
// asymmetry is deliberate and shipped — describeEntrypointProbe's entrypointRefused
// branch does the same thing — and must not be "fixed" in either direction.
//
// ★ THIS IS THE ONE CASE THAT CANNOT BE STAGED AS A FIXTURE PROJECT, AND THE REASON IS
// WORTH RECORDING. The SAME trust gate fires at manifest PARSE time (pack.ParseManifest's
// per-rule validateEngine calls engine.CheckToolAllowed with the identical predicate), so
// a pack.yml declaring an un-allowlisted provision never survives loadInstalledPacks:
// PacksErr is set, packs-installed FAILS and owns the condition, and engine-tools SKIPS
// naming it — which is the correct one-condition-one-owner behavior, not a defect. The
// refusal branch inside the collection is therefore reachable only from an in-memory pack
// set, so this case is driven at the function level while every other case in this file
// goes through a staged project.
func TestDoctor_EngineToolTrustRefusalFails(t *testing.T) {
	withBinaryResolver(t)

	packs := []*pack.Manifest{collectFixtureManifest("fixture/refused",
		collectBinding{
			key:       "refused-findings",
			command:   "backstop-refused-engine-134 scan",
			bound:     true,
			provision: &engine.Provision{Tool: "backstop-refused-engine-134", Version: "9.9.9"},
		},
	)}

	result := checkEngineToolsPresent(doctorContext{Packs: packs})

	if result.Status != doctorStatusFail {
		t.Fatalf("engine-tools status = %q, want %q on a refused tool", result.Status, doctorStatusFail)
	}
	if !strings.Contains(result.Message, "refused") {
		t.Errorf("the refusal must be reported as a refusal, got: %s", result.Message)
	}
	if strings.Contains(result.Message, "not found on PATH") {
		t.Errorf("a tool backstop refuses to TRUST must not be reported as MISSING, got: %s", result.Message)
	}
	if !strings.Contains(result.Message, "backstop-refused-engine-134") {
		t.Errorf("the refusal must name the refused tool, got: %s", result.Message)
	}
}
