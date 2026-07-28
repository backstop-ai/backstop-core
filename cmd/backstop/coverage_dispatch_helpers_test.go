package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/check"
	"github.com/backstop-ai/backstop-core/pkg/pack"
	"github.com/backstop-ai/backstop-core/pkg/pack/engine"
)

// readCoverageProfileFixture reads a real Go -coverprofile fixture from the
// go-toolchain-coverage testdata dir (the inputs the convert script runs over).
func readCoverageProfileFixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(repoRoot(t), "cmd", "backstop", "testdata", "go-toolchain-coverage", name))
	if err != nil {
		t.Fatalf("read coverage profile fixture %s: %v", name, err)
	}
	return b
}

// engineKeysOf returns the keys of an engines map for diagnostic messages.
func engineKeysOf(m map[string]pack.EngineSpec) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// keysOfRecords returns the paths of a record map for diagnostic messages.
func keysOfRecords(m map[string]check.CoverageRecord) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// containsAll reports whether s contains every substring.
func containsAll(s string, subs ...string) bool {
	for _, sub := range subs {
		if !strings.Contains(s, sub) {
			return false
		}
	}
	return true
}

// findRecordBySuffix returns the record whose path ends with suffix.
func findRecordBySuffix(byFile map[string]check.CoverageRecord, suffix string) (check.CoverageRecord, bool) {
	for path, r := range byFile {
		if strings.HasSuffix(path, suffix) {
			return r, true
		}
	}
	return check.CoverageRecord{}, false
}

// coverageRecordsJSON is a minimal coverage-records payload (the SECOND normalized
// output type, NOT SARIF) carrying one measured-and-passing file and one
// measured-and-failing file. Returned from a func (not a package const/var) to keep
// no global mutable state. It is the convert stdout a coverage engine emits.
func coverageRecordsJSON() string {
	return `[{"path":"pkg/svc/handler.go","covered":92,"total":100,"measured":true,"excluded":false,"metric":"statement"},` +
		`{"path":"pkg/svc/shortfall.go","covered":30,"total":100,"measured":true,"excluded":false,"metric":"statement"}]`
}

// sarifFindingsJSON is a minimal SARIF document carrying one finding — the
// normalized output a lint/build/test/findings engine emits. Used to assert the
// four findings/output gate-types route to ParsePackFindings, never the coverage
// records channel.
func sarifFindingsJSON() string {
	return `{"version":"2.1.0","runs":[{"results":[{"ruleId":"x","level":"error","message":{"text":"finding"},` +
		`"locations":[{"physicalLocation":{"artifactLocation":{"uri":"main.go"},"region":{"startLine":1}}}]}]}]}`
}

// engineBindingForGateType returns an in-memory EngineBinding declaring the given
// gate_type, with a convert script so the dispatch routes engine stdout through the
// pack's declared convert (resolveSandboxedRunStdout) before parsing. The routing
// key under test is the DECLARED GateType ONLY — the command/convert names are
// deliberately neutral so no command-string or tool-name sniff could select the
// channel.
func engineBindingForGateType(gt engine.GateType) engine.EngineBinding {
	return engine.EngineBinding{
		Command:       "neutral-tool run",
		InputMode:     engine.InputModeNone,
		Convert:       "scripts/convert.sh",
		ScopeKind:     engine.ScopeKindProjectWide,
		ProjectTarget: ".",
		GateType:      gt,
	}
}

// gateTypeRoutingManifest builds an in-memory pack manifest carrying ONE rule bound
// to a pack-declared engine of the given gate_type. The engine is declared via the
// manifest's Engines map (the SPEC-035 declared substrate) so resolveEngineRegistry
// resolves the rule's engine to the gate_type under test — never the baked
// DefaultRegistry. The routing tests dispatch this manifest and assert the channel.
func gateTypeRoutingManifest(engineName string, gt engine.GateType) *pack.Manifest {
	return &pack.Manifest{
		NormalizedName: "test-org/coverage-routing",
		Engines: map[string]pack.EngineSpec{
			engineName: {Binding: engineBindingForGateType(gt)},
		},
		Content: pack.Content{Ruleset: pack.Ruleset{Rules: []pack.Rule{
			{ID: engineName + "-rule", Engine: engineName, Standard: "x"},
		}}},
	}
}

// coverageRoutingPacksDir returns a scratch .backstop/packs dir holding a pack root
// for the routing manifest plus a convert script that echoes the supplied normalized
// output. The convert is the REAL on-disk script the sandboxed-convert seam shells,
// so the routing tests exercise the convert pipe, not a stubbed parser. The
// normalizedOut bytes are what the convert emits on stdout (coverage-records or SARIF).
func coverageRoutingPacksDir(t *testing.T, normalizedOut string) string {
	t.Helper()
	packsDir := t.TempDir()
	packRoot := filepath.Join(packsDir, "test-org", "coverage-routing")
	mkDirAll(t, filepath.Join(packRoot, "scripts"))
	// The convert reads stdin (the engine's raw stdout) and emits the canned
	// normalized output — modeling a real convert that transforms the tool output.
	writeFileStr(t, filepath.Join(packRoot, "scripts", "convert.sh"),
		"#!/bin/sh\ncat >/dev/null\ncat <<'NORMALIZED'\n"+normalizedOut+"\nNORMALIZED\n")
	return packsDir
}
