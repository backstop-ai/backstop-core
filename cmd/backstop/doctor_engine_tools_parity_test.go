package main

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/pack"
	"github.com/backstop-ai/backstop-core/pkg/pack/engine"
)

// doctor_engine_tools_parity_test.go carries ISSUE-134's STRUCTURAL claims — the ones
// that keep the fix from being correct today and wrong after the next edit.

// TestDoctor_EngineToolVerdictMatchesGateProvisioning is THE LOAD-BEARING TEST
// (CLM-002): over any pack set, doctor's engine-tools check reports `fail` if and only
// if provisionEngines returns an error. Neither answers a question the other does not.
//
// The biconditional is asserted in BOTH directions explicitly, so neither a doctor that
// always fails nor one that never does can pass. Under a collection widened to
// manifest.Engines the UNBOUND row breaks it: doctor fails where provisionEngines
// passes.
func TestDoctor_EngineToolVerdictMatchesGateProvisioning(t *testing.T) {
	present := "backstop-present-engine-134"

	cases := []struct {
		name  string
		packs []*pack.Manifest
	}{
		{
			name: "absent tool",
			packs: []*pack.Manifest{collectFixtureManifest("parity/absent",
				collectBinding{key: "absent-findings", command: "backstop-absent-engine-134 scan", bound: true},
			)},
		},
		{
			name: "present tool",
			packs: []*pack.Manifest{collectFixtureManifest("parity/present",
				collectBinding{key: "present-findings", command: present + " scan", bound: true},
			)},
		},
		{
			name: "unbound engine's tool is absent but never dispatched",
			packs: []*pack.Manifest{collectFixtureManifest("parity/unbound",
				collectBinding{key: "bound-findings", command: present + " scan", bound: true},
				collectBinding{key: "unbound-findings", command: "backstop-unbound-engine-134 scan", bound: false},
			)},
		},
		{
			name: "pack set binds no engine tool",
			packs: []*pack.Manifest{collectFixtureManifest("parity/none",
				collectBinding{key: "commandless", command: "", bound: true},
			)},
		},
		{
			name: "trust refusal",
			packs: []*pack.Manifest{collectFixtureManifest("parity/refused",
				collectBinding{
					key:       "refused-findings",
					command:   "backstop-refused-engine-134 scan",
					bound:     true,
					provision: &engine.Provision{Tool: "backstop-refused-engine-134", Version: "9.9.9"},
				},
			)},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			withBinaryResolver(t, present)

			gateErr := provisionEngines(testCase.packs)
			result := checkEngineToolsPresent(doctorContext{Packs: testCase.packs})

			doctorFails := result.Status == doctorStatusFail
			gateRefuses := gateErr != nil

			if doctorFails && !gateRefuses {
				t.Fatalf("doctor FAILS where provisionEngines PASSES — a diagnostic that cries wolf gets ignored\ndoctor: %s / %s", result.Status, result.Message)
			}
			if gateRefuses && !doctorFails {
				t.Fatalf("provisionEngines REFUSES (%v) where doctor reports %q — this is exactly the ISSUE-134 disagreement\ndoctor: %s", gateErr, result.Status, result.Message)
			}
		})
	}
}

// recordingCommandRunner records every invocation and refuses to run anything.
type recordingCommandRunner struct {
	invocations []string
}

func (r *recordingCommandRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	r.invocations = append(r.invocations, strings.TrimSpace(name+" "+strings.Join(args, " ")))
	return nil, context.Canceled
}

func (r *recordingCommandRunner) RunStdout(ctx context.Context, name string, args ...string) ([]byte, error) {
	return r.Run(ctx, name, args...)
}

// TestDoctor_EngineToolCheckRunsNoCommand (CLM-004). Doctor PRESENCE-PROBES; it never
// executes a findings-engine command.
//
// Executing pack-declared scanners at doctor time would make a diagnostic as slow as a
// gate run and would duplicate the gate's dispatch — and a findings tool run against
// nothing has no success signal distinguishable from a tool that scanned and found
// nothing. A future "let's actually run it to be sure" edit reds here, which is the
// point.
func TestDoctor_EngineToolCheckRunsNoCommand(t *testing.T) {
	present := "backstop-present-engine-134"
	withBinaryResolver(t, present)

	fixtures := []*pack.Manifest{
		collectFixtureManifest("norun/absent",
			collectBinding{key: "absent-findings", command: "backstop-absent-engine-134 scan", bound: true},
		),
		collectFixtureManifest("norun/present",
			collectBinding{key: "present-findings", command: present + " scan", bound: true},
		),
	}

	for _, manifest := range fixtures {
		runner := &recordingCommandRunner{}
		result := checkEngineToolsPresent(doctorContext{Packs: []*pack.Manifest{manifest}, Runner: runner})
		if result.Status == "" {
			t.Fatalf("%s: the check returned no status at all", manifest.NormalizedName)
		}
		if len(runner.invocations) != 0 {
			t.Errorf("%s: the engine-tools check invoked %d command(s) (%v); it presence-probes and executes nothing",
				manifest.NormalizedName, len(runner.invocations), runner.invocations)
		}
	}
}

// TestInit_ToolchainProbeStillSelectsOnlyTestAndBuild (CLM-005) is the `backstop init`
// non-regression guard, expressed two ways because each catches what the other misses.
func TestInit_ToolchainProbeStillSelectsOnlyTestAndBuild(t *testing.T) {
	// (a) BEHAVIORAL: a manifest declaring a test binding, a build binding and a
	// RULE-BOUND findings binding yields exactly the test and build probes.
	manifest := &pack.Manifest{
		Name:           "init-guard/pack",
		NormalizedName: "init-guard/pack",
		Engines: map[string]pack.EngineSpec{
			"probe-test":     {Binding: engine.EngineBinding{Command: "true", GateType: engine.GateTypeTest}},
			"probe-build":    {Binding: engine.EngineBinding{Command: "true", GateType: engine.GateTypeBuild}},
			"probe-findings": {Binding: engine.EngineBinding{Command: "backstop-absent-engine-134 scan", GateType: engine.GateTypeFindings}},
		},
	}
	manifest.Content.Ruleset.Rules = []pack.Rule{{ID: "findings-rule", Engine: "probe-findings"}}

	runner := &recordingCommandRunner{}
	probes := (&packEntrypointProber{Packs: []*pack.Manifest{manifest}, Runner: runner}).Probe(context.Background())

	selected := map[string]engine.GateType{}
	for _, probe := range probes {
		selected[probe.EngineKey] = probe.GateType
	}
	if len(selected) != 2 {
		t.Fatalf("Probe selected %d binding(s) (%v), want exactly the test and build ones", len(selected), selected)
	}
	if _, ok := selected["probe-findings"]; ok {
		t.Errorf("Probe selected the RULE-BOUND findings binding; init's selection is by STAGE, never by tool, and ISSUE-134 changed nothing about it")
	}
	for _, key := range []string{"probe-test", "probe-build"} {
		if _, ok := selected[key]; !ok {
			t.Errorf("Probe did not select %q", key)
		}
	}

	// (b) SOURCE-LEVEL ABSENCE. Arm (a) alone would still pass if someone widened the
	// filter and narrowed it somewhere downstream; this pins the file ISSUE-134
	// promised not to change.
	fset := token.NewFileSet()
	file, parseErr := parser.ParseFile(fset, "pack_entrypoint_prober.go", nil, 0)
	if parseErr != nil {
		t.Fatalf("parsing pack_entrypoint_prober.go: %v", parseErr)
	}

	var probeDecl *ast.FuncDecl
	for _, decl := range file.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if ok && funcDecl.Name.Name == "Probe" && funcDecl.Recv != nil {
			probeDecl = funcDecl
		}
	}
	if probeDecl == nil {
		t.Fatal("pack_entrypoint_prober.go declares no Probe method — the scan is looking at the wrong source")
	}

	named := map[string]bool{}
	ast.Inspect(probeDecl.Body, func(node ast.Node) bool {
		selector, ok := node.(*ast.SelectorExpr)
		if ok && strings.HasPrefix(selector.Sel.Name, "GateType") {
			named[selector.Sel.Name] = true
		}
		return true
	})

	for _, want := range []string{"GateTypeTest", "GateTypeBuild"} {
		if !named[want] {
			t.Errorf("Probe no longer tests %s; init's stage selection moved", want)
		}
	}
	for name := range named {
		if name != "GateTypeTest" && name != "GateTypeBuild" && name != "GateType" {
			t.Errorf("Probe names the gate type %q; its selection is exactly test and build, and widening it is ISSUE-134's explicit non-goal", name)
		}
	}
}

// TestDoctor_EngineToolCheckIsInsideTheDoctorSourceScan (CLM-008). The new check
// function must live in a file isDoctorOwnedFile admits, or CLM-051's stack-policy
// source scan silently stops covering it — a scan that quietly stops covering the thing
// it was pointed at is this directive's own subject matter.
//
// The file is resolved by FINDING the declaration rather than by hardcoding a name, so
// the day someone moves the check this reds and says why.
func TestDoctor_EngineToolCheckIsInsideTheDoctorSourceScan(t *testing.T) {
	declaringFile := ""
	for _, parsed := range parseNonTestPackageFiles(t) {
		for _, decl := range parsed.file.Decls {
			funcDecl, ok := decl.(*ast.FuncDecl)
			if ok && funcDecl.Recv == nil && funcDecl.Name.Name == "checkEngineToolsPresent" {
				declaringFile = parsed.path
			}
		}
	}
	if declaringFile == "" {
		t.Fatal("no non-test file declares checkEngineToolsPresent")
	}
	if !isDoctorOwnedFile(declaringFile) {
		t.Errorf("checkEngineToolsPresent is declared in %s, which isDoctorOwnedFile does NOT admit; the stack-policy source scan (CLM-051 arm (b)) no longer covers it. Either move the function into doctor_checks.go or widen isDoctorOwnedFile deliberately", declaringFile)
	}
}
