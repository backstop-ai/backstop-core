package main

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/check"
	"github.com/backstop-ai/backstop-core/pkg/pack"
	"github.com/backstop-ai/backstop-core/pkg/pack/engine"
)

// pack_gate_provision_collect_test.go pins collectRequiredEngineTools — the ONE
// collection authority the gate's provisionEngines and doctor's engine-tools check
// both consume (ISSUE-134 CLM-003) — and pins that extracting it left
// provisionEngines' externally observable behavior unmoved.
//
// THE WALK IS RULE-DRIVEN AND THAT IS A CORRECTNESS PROPERTY, not an implementation
// detail: an engine no rule binds is never dispatched, so probing its tool would
// refuse a run that was never going to happen. Widening the walk to manifest.Engines
// looks more thorough and makes doctor fail on a project the gate passes.

// collectFixtureManifest builds an in-test manifest whose rules bind the named
// engines, so the walk under test resolves engines THROUGH RULES rather than
// through the engines map.
//
// Every binding carries a NIL Provision unless a test replaces it: checkEngineToolAllowed
// returns nil immediately for a nil Provision (the Layer-0 exemption), so the
// selection — not the trust gate — is what the selection tests observe.
type collectBinding struct {
	key       string
	command   string
	bound     bool
	provision *engine.Provision
}

func collectFixtureManifest(name string, bindings ...collectBinding) *pack.Manifest {
	manifest := &pack.Manifest{
		Name:           name,
		NormalizedName: name,
		Engines:        map[string]pack.EngineSpec{},
	}
	for _, binding := range bindings {
		manifest.Engines[binding.key] = pack.EngineSpec{
			Binding: engine.EngineBinding{
				Command:   binding.command,
				InputMode: engine.InputModeNone,
				ScopeKind: engine.ScopeKindProjectWide,
				GateType:  engine.GateTypeFindings,
				Provision: binding.provision,
			},
		}
		if !binding.bound {
			continue
		}
		manifest.Content.Ruleset.Rules = append(manifest.Content.Ruleset.Rules, pack.Rule{
			ID:     binding.key + "-rule",
			Engine: binding.key,
		})
	}
	return manifest
}

// collectedNames returns the probed names in the order collection returned them.
func collectedNames(tools []requiredTool) []string {
	out := make([]string, 0, len(tools))
	for _, tool := range tools {
		out = append(out, tool.name)
	}
	return out
}

// recordingResolver installs a binaryResolver that records every name it was asked
// to resolve and reports the named set as present.
func recordingResolver(t *testing.T, present ...string) *[]string {
	t.Helper()
	set := map[string]bool{}
	for _, name := range present {
		set[name] = true
	}
	attempts := &[]string{}
	original := binaryResolver
	binaryResolver = func(name string) (string, error) {
		*attempts = append(*attempts, name)
		if set[name] {
			return "/fake/bin/" + name, nil
		}
		return "", errors.New("not found")
	}
	t.Cleanup(func() { binaryResolver = original })
	return attempts
}

// TestGate_CollectRequiredEngineToolsWalksRulesNotEngines is the guard against the
// tempting wrong implementation (CLM-003). A walk over manifest.Engines returns the
// unbound engine's tool and reds here.
func TestGate_CollectRequiredEngineToolsWalksRulesNotEngines(t *testing.T) {
	manifest := collectFixtureManifest("fixture/rule-walk",
		collectBinding{key: "bound-engine", command: "backstop-absent-engine-134-bound scan", bound: true},
		collectBinding{key: "unbound-engine", command: "backstop-absent-engine-134-unbound scan", bound: false},
	)

	tools, err := collectRequiredEngineTools([]*pack.Manifest{manifest})
	if err != nil {
		t.Fatalf("collectRequiredEngineTools: %v", err)
	}

	names := collectedNames(tools)
	if !slices.Contains(names, "backstop-absent-engine-134-bound") {
		t.Errorf("collection = %v, want the RULE-BOUND engine's tool present", names)
	}
	if slices.Contains(names, "backstop-absent-engine-134-unbound") {
		t.Errorf("collection = %v, want the unbound engine's tool ABSENT — an engine no rule binds is never dispatched, so probing it refuses a run that was never going to happen", names)
	}
}

// TestGate_CollectRequiredEngineToolsDedupesAndSorts (CLM-003). Two rules binding two
// engines whose commands share argv[0] collapse to ONE entry, and the returned order
// is sorted by probed name across a multi-pack set.
//
// Sorted order is not cosmetic: it is what makes the gate's first-absent refusal
// deterministic across runs, and doctor's report line order rides it too.
func TestGate_CollectRequiredEngineToolsDedupesAndSorts(t *testing.T) {
	shared := collectFixtureManifest("fixture/shared-argv",
		collectBinding{key: "go-build", command: "go build", bound: true},
		collectBinding{key: "go-test", command: "go test ./...", bound: true},
	)
	other := collectFixtureManifest("fixture/other",
		collectBinding{key: "alpha", command: "alpha-tool scan", bound: true},
	)

	tools, err := collectRequiredEngineTools([]*pack.Manifest{shared, other})
	if err != nil {
		t.Fatalf("collectRequiredEngineTools: %v", err)
	}

	names := collectedNames(tools)
	want := []string{"alpha-tool", "go"}
	if len(names) != len(want) {
		t.Fatalf("collection = %v, want exactly %v — two engines sharing argv[0] must dedupe to one entry", names, want)
	}
	for i, name := range names {
		if name != want[i] {
			t.Fatalf("collection = %v, want %v in sorted probed-name order", names, want)
		}
	}
}

// TestGate_CollectRequiredEngineToolsSkipsCommandlessEngine (CLM-003). A binding with
// an EMPTY command — the sandbox engine ships its own executable — contributes no
// required tool rather than probing an empty name.
func TestGate_CollectRequiredEngineToolsSkipsCommandlessEngine(t *testing.T) {
	manifest := collectFixtureManifest("fixture/commandless",
		collectBinding{key: "sandbox", command: "", bound: true},
		collectBinding{key: "real", command: "alpha-tool scan", bound: true},
	)

	tools, err := collectRequiredEngineTools([]*pack.Manifest{manifest})
	if err != nil {
		t.Fatalf("collectRequiredEngineTools: %v", err)
	}

	names := collectedNames(tools)
	if slices.Contains(names, "") {
		t.Errorf("collection = %v, want no empty probed name — a commandless engine ships its own executable", names)
	}
	if len(names) != 1 || names[0] != "alpha-tool" {
		t.Errorf("collection = %v, want exactly [alpha-tool]", names)
	}
}

// TestGate_CollectRequiredEngineToolsSurfacesTrustRefusal (CLM-003, CLM-006's refusal
// branch). A rule-bound engine whose provision: names a tool the allowlist does not
// admit makes collection return an ERROR, and it does so BEFORE any presence probe:
// a tool backstop refuses to TRUST must be reported as untrusted, not as missing.
func TestGate_CollectRequiredEngineToolsSurfacesTrustRefusal(t *testing.T) {
	attempts := recordingResolver(t)

	manifest := collectFixtureManifest("fixture/refused",
		collectBinding{
			key:       "refused-engine",
			command:   "not-allowlisted-tool scan",
			bound:     true,
			provision: &engine.Provision{Tool: "not-allowlisted-tool", Version: "9.9.9"},
		},
	)

	_, err := collectRequiredEngineTools([]*pack.Manifest{manifest})
	if err == nil {
		t.Fatal("an un-allowlisted provisioned tool must make collection return an error, got nil")
	}
	if !strings.Contains(err.Error(), "not-allowlisted-tool") {
		t.Errorf("refusal must name the refused tool, got: %v", err)
	}
	if len(*attempts) != 0 {
		t.Errorf("the trust gate fired AFTER %d presence probe(s) (%v); it must fire ahead of every probe", len(*attempts), *attempts)
	}
}

// TestGate_ProvisionEnginesBehaviorSurvivesExtraction is the non-regression half
// (CLM-003). provisionEngines still returns a *check.ConfigError whose message is
// EXACTLY absentToolMessage's rendering, and still returns on the FIRST absent name
// in sorted order rather than collecting them all.
func TestGate_ProvisionEnginesBehaviorSurvivesExtraction(t *testing.T) {
	// Two absent tools; `alpha-absent` sorts first, so it — and only it — is the one
	// the refusal names.
	manifest := collectFixtureManifest("fixture/first-absent",
		collectBinding{key: "alpha", command: "alpha-absent scan", bound: true},
		collectBinding{key: "zulu", command: "zulu-absent scan", bound: true},
	)
	attempts := recordingResolver(t)

	err := provisionEngines([]*pack.Manifest{manifest})
	if err == nil {
		t.Fatal("provisionEngines must fail loud on an absent tool, got nil")
	}
	var configErr *check.ConfigError
	if !errors.As(err, &configErr) {
		t.Fatalf("provisionEngines must return *check.ConfigError (exit 2), got %T: %v", err, err)
	}

	want := absentToolMessage(requiredTool{
		name:   "alpha-absent",
		pack:   "fixture/first-absent",
		engine: "alpha",
	})
	if configErr.Message != want {
		t.Errorf("provisionEngines message =\n%q\nwant the unchanged absentToolMessage rendering:\n%q", configErr.Message, want)
	}

	if len(*attempts) != 1 || (*attempts)[0] != "alpha-absent" {
		t.Errorf("provisionEngines probed %v; it must return on the FIRST absent name in sorted order", *attempts)
	}
}
