package main

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/check"
	"github.com/bmanson/backstop-core/pkg/pack"
	"github.com/bmanson/backstop-core/pkg/pack/engine"
)

// coverageProducerManifest builds an in-memory pack manifest whose ONE coverage
// engine declares the NEW producer: field (ISSUE-045 option (ii)). The Binding is
// set directly (not YAML-parsed) so the dispatch resolves the rule's engine to it
// via resolveEngineRegistry. The plain Command is a MUST-NOT-RUN sentinel: when a
// producer is declared, the dispatch must run the packRoot-resolved producer script
// INSTEAD of splitCommand(binding.Command). The declared stdout_artifact + convert
// mirror the real go-coverage binding so the sandboxed convert still runs on the
// producer's payload afterward.
func coverageProducerManifest(producerRel string) *pack.Manifest {
	return &pack.Manifest{
		NormalizedName: "test-org/cov-producer",
		Engines: map[string]pack.EngineSpec{
			"cov": {Binding: engine.EngineBinding{
				Command:        "must-not-run plain-command",
				Producer:       producerRel,
				StdoutArtifact: "cover.out",
				Convert:        "scripts/convert.sh",
				InputMode:      engine.InputModeNone,
				ScopeKind:      engine.ScopeKindProjectWide,
				ProjectTarget:  "./...",
				GateType:       engine.GateTypeCoverage,
			}},
		},
		Content: pack.Content{Ruleset: pack.Ruleset{Rules: []pack.Rule{
			{ID: "cov-rule", Engine: "cov", Standard: "x"},
		}}},
	}
}

// coverageProducerPacksDir returns a scratch .backstop/packs dir holding the
// cov-producer pack root with a producer script (on disk so the dispatch's
// os.Stat resolution passes; the fake runner intercepts its exec) and a convert
// script that echoes one canned coverage record.
func coverageProducerPacksDir(t *testing.T) string {
	t.Helper()
	packsDir := t.TempDir()
	packRoot := filepath.Join(packsDir, "test-org", "cov-producer")
	mkDirAll(t, filepath.Join(packRoot, "scripts"))
	writeFileStr(t, filepath.Join(packRoot, "scripts", "enrich.sh"), "#!/bin/sh\n# producer stub (fake runner intercepts exec)\n")
	writeFileStr(t, filepath.Join(packRoot, "scripts", "convert.sh"),
		"#!/bin/sh\ncat >/dev/null\ncat <<'J'\n[{\"path\":\"embed.go\",\"covered\":9,\"total\":10,\"measured\":true,\"excluded\":false,\"metric\":\"statement\"}]\nJ\n")
	return packsDir
}

// TestCoverageProducer_DispatchRunsPackRootResolvedProducerUnsandboxed (CLM-007):
// when a coverage binding declares producer:, the dispatch resolves it under packRoot
// and runs THAT script via the runner (un-sandboxed) INSTEAD of the plain Command,
// then feeds the declared stdout_artifact into the sandboxed convert. Proves the
// producer path is pack DATA the manifest supplied — the dispatch bakes no
// go/cover/profile literal.
func TestCoverageProducer_DispatchRunsPackRootResolvedProducerUnsandboxed(t *testing.T) {
	if dispatchPackEnginesFn != nil {
		t.Fatal("dispatchPackEnginesFn must be nil — the producer dispatch must run the REAL path, not a stub")
	}
	packsDir := coverageProducerPacksDir(t)
	packRoot := filepath.Join(packsDir, "test-org", "cov-producer")
	producerPath := filepath.Join(packRoot, "scripts", "enrich.sh")

	projectRoot := t.TempDir()
	// Simulate the producer having written the enriched artifact (the fake runner
	// does not actually exec the producer, so pre-place its declared output).
	writeFileStr(t, filepath.Join(projectRoot, "cover.out"),
		"mode: atomic\ngithub.com/x/embed.go:1.1,2.2 1 1\n")

	var convertStdin []byte
	stubSandboxedRunStdout(t, &convertStdin)

	runner := &fixtureRunner{byCmd: map[string][]byte{}}
	m := coverageProducerManifest("scripts/enrich.sh")

	records, err := dispatchPackCoverage([]*pack.Manifest{m}, packsDir, projectRoot, nil, runner)
	if err != nil {
		t.Fatalf("producer dispatch: %v", err)
	}

	// (1) The packRoot-resolved producer ran via the runner (un-sandboxed), exactly once.
	if len(runner.calls) != 1 {
		t.Fatalf("expected exactly one producer invocation via the runner, got %d: %#v", len(runner.calls), runner.calls)
	}
	if runner.calls[0].name != producerPath {
		t.Errorf("dispatch must run the packRoot-resolved producer script, got %q want %q", runner.calls[0].name, producerPath)
	}
	// The producer shapes its own invocation: no inputs / ProjectTarget bolted on.
	if len(runner.calls[0].args) != 0 {
		t.Errorf("producer must run un-decorated (no inputs/target appended), got args %v", runner.calls[0].args)
	}
	// (2) The plain Command was NEVER run — the producer subsumes it.
	for _, c := range runner.calls {
		if c.name == "must-not-run" {
			t.Errorf("dispatch must NOT run splitCommand(binding.Command) when a producer is declared; ran %q", c.name)
		}
	}
	// (3) The sandboxed convert still ran on the producer's stdout_artifact payload.
	if len(convertStdin) == 0 {
		t.Fatal("the sandboxed convert never received the producer's artifact payload — the producer->convert pipe did not run")
	}
	if !strings.Contains(string(convertStdin), "embed.go") {
		t.Errorf("convert must receive the producer's cover.out payload, got %q", string(convertStdin))
	}
	if len(records) != 1 {
		t.Fatalf("expected the one record the convert emitted, got %d: %#v", len(records), records)
	}
}

// TestCoverageProducer_TrustGateBlocksProducerBeforeExec (CLM-007, Sharp Edge 3):
// the trusted-tool allowlist gate runs BEFORE the producer executes. A binding whose
// Provision is non-nil and un-allowlisted returns a *check.ConfigError (exit 2) and
// the producer is NEVER run. Driven via the trustedToolAllowlist seam (an empty
// allowlist), NOT a stub-open allowlist.
func TestCoverageProducer_TrustGateBlocksProducerBeforeExec(t *testing.T) {
	orig := trustedToolAllowlist
	trustedToolAllowlist = func() map[string]string { return map[string]string{} }
	t.Cleanup(func() { trustedToolAllowlist = orig })

	packsDir := coverageProducerPacksDir(t)
	projectRoot := t.TempDir()
	runner := &fixtureRunner{byCmd: map[string][]byte{}}

	m := coverageProducerManifest("scripts/enrich.sh")
	sp := m.Engines["cov"]
	sp.Binding.Provision = &engine.Provision{Tool: "sneaky-tool", Version: "1.0.0"}
	m.Engines["cov"] = sp

	_, err := dispatchPackCoverage([]*pack.Manifest{m}, packsDir, projectRoot, nil, runner)
	if err == nil {
		t.Fatal("an un-allowlisted provisioned tool must fail the trust gate, not run the producer")
	}
	var cfgErr *check.ConfigError
	if !errors.As(err, &cfgErr) {
		t.Errorf("trust-gate rejection must be a *check.ConfigError (exit 2), got %T: %v", err, err)
	}
	if len(runner.calls) != 0 {
		t.Errorf("the producer must NEVER run when the trust gate rejects; got calls %#v", runner.calls)
	}
}

// TestCoverageProducer_MissingProducerScriptFailsLoud (CLM-007): a declared-but-
// missing producer script is a fail-loud broken-pack error naming the pack (mirroring
// the convert-missing error), and the runner is never reached.
func TestCoverageProducer_MissingProducerScriptFailsLoud(t *testing.T) {
	packsDir := coverageProducerPacksDir(t)
	projectRoot := t.TempDir()
	runner := &fixtureRunner{byCmd: map[string][]byte{}}

	m := coverageProducerManifest("scripts/does-not-exist.sh")

	_, err := dispatchPackCoverage([]*pack.Manifest{m}, packsDir, projectRoot, nil, runner)
	if err == nil {
		t.Fatal("a declared-but-missing producer script must fail loud, not silently fall back")
	}
	if !strings.Contains(err.Error(), "test-org/cov-producer") {
		t.Errorf("broken-pack error must name the pack, got: %v", err)
	}
	if len(runner.calls) != 0 {
		t.Errorf("the runner must never be reached when the producer script is missing; got %#v", runner.calls)
	}
}
