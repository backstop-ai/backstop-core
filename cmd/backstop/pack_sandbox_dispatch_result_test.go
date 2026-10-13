package main

import (
	"path/filepath"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/pack"
	"github.com/backstop-ai/backstop-core/pkg/pack/engine"
	"github.com/backstop-ai/backstop-core/pkg/packval"
)

func TestPackSandbox_DispatchResultsCarryDeterministicEvidence(t *testing.T) {
	runner := &recordingSandboxRunner{mode: packval.SandboxModeExternal}
	engines, err := dispatchPackEnginesWithEvidence(nil, t.TempDir(), t.TempDir(), nil, emptySarifRunner{}, runner)
	if err != nil {
		t.Fatal(err)
	}
	if engines.Violations == nil || engines.NativeSandboxApplied {
		t.Fatalf("engine dispatch result = %#v", engines)
	}

	coverage, err := dispatchPackCoverageWithEvidence(nil, t.TempDir(), t.TempDir(), nil, emptySarifRunner{}, runner)
	if err != nil {
		t.Fatal(err)
	}
	if coverage.Records == nil || coverage.NativeSandboxApplied {
		t.Fatalf("coverage dispatch result = %#v", coverage)
	}
}

// TestPackSandbox_DispatchEvidenceSurvivesMidLoopEngineError (ISSUE-209 CLM-004):
// NativeSandboxApplied is the OR across EVERY engine that ran, including engines
// after a failure, and a registry-lookup failure mid-loop must not reset it (it
// used to return an empty result, dropping the evidence of a sandboxed convert
// that had already run).
func TestPackSandbox_DispatchEvidenceSurvivesMidLoopEngineError(t *testing.T) {
	sarif := `{"version":"2.1.0","runs":[{"tool":{"driver":{"name":"cont"}},"results":[]}]}`
	convertBinding := func(command string) engine.EngineBinding {
		return engine.EngineBinding{
			Command:   command,
			InputMode: engine.InputModeNone,
			ScopeKind: engine.ScopeKindProjectWide,
			Category:  engine.EngineCategoryOpinion,
			Convert:   "scripts/convert.sh",
		}
	}
	plainBinding := func(command string) engine.EngineBinding {
		return engine.EngineBinding{
			Command:   command,
			InputMode: engine.InputModeNone,
			ScopeKind: engine.ScopeKindProjectWide,
			Category:  engine.EngineCategoryOpinion,
		}
	}
	cases := []struct {
		name    string
		engines map[string]engine.EngineBinding
		order   []string
	}{
		{
			name: "applied before the failure is not reset",
			engines: map[string]engine.EngineBinding{
				"sbx-convert": convertBinding("sbx-convert"),
				"sbx-after":   plainBinding("sbx-after"),
			},
			order: []string{"sbx-convert", "sbx-unregistered-209", "sbx-after"},
		},
		{
			name: "applied only after the failure still counts",
			engines: map[string]engine.EngineBinding{
				"sbx-convert": convertBinding("sbx-convert"),
			},
			order: []string{"sbx-unregistered-209", "sbx-convert"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			packsDir := t.TempDir()
			packRoot := filepath.Join(packsDir, "org", "sbx")
			mkDirAll(t, filepath.Join(packRoot, "scripts"))
			writeFileStr(t, filepath.Join(packRoot, "scripts", "convert.sh"), "#!/bin/sh\ncat\n")

			m := &pack.Manifest{NormalizedName: "org/sbx", Engines: map[string]pack.EngineSpec{}}
			for name, b := range tc.engines {
				m.Engines[name] = pack.EngineSpec{Binding: b}
			}
			for _, name := range tc.order {
				m.Content.Ruleset.Rules = append(m.Content.Ruleset.Rules, pack.Rule{ID: name + "-rule", Engine: name, Standard: "x"})
			}
			runner := &fixtureRunner{byCmd: map[string][]byte{
				"sbx-convert": []byte(sarif),
				"sbx-after":   []byte(sarif),
			}}

			result, err := dispatchPackEnginesWithEvidence([]*pack.Manifest{m}, packsDir, t.TempDir(), nil, runner, directConvertSandboxRunner(nil))
			if err == nil {
				t.Fatal("the registry-lookup failure must surface a non-nil error")
			}
			if !result.NativeSandboxApplied {
				t.Errorf("NativeSandboxApplied must be true: a sandboxed convert ran in this dispatch; got %#v", result)
			}
		})
	}
}
