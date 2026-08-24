package main

import (
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/pack"
	"github.com/backstop-ai/backstop-core/pkg/pack/engine"
)

// TestDispatch_CoverageGateTypeRoutesToRecordsParser proves a coverage engine
// (binding GateType == engine.GateTypeCoverage) has its convert output routed
// through check.ParsePackCoverage and produces []check.CoverageRecord — records,
// NOT SARIF violations, come back (CLM-001). The routing key is the DECLARED
// gate_type; the convert emits coverage-records JSON and dispatchPackCoverage
// terminates in the records parser.
func TestDispatch_CoverageGateTypeRoutesToRecordsParser(t *testing.T) {
	sandboxRunner := directConvertSandboxRunner(nil)
	packsDir := coverageRoutingPacksDir(t, coverageRecordsJSON())
	manifest := gateTypeRoutingManifest("cov-engine", engine.GateTypeCoverage)
	runner := &fixtureRunner{byCmd: map[string][]byte{"neutral-tool": []byte("raw coverage profile")}}

	result, err := dispatchPackCoverageWithEvidence([]*pack.Manifest{manifest}, packsDir, t.TempDir(), nil, runner, sandboxRunner)
	if err != nil {
		t.Fatalf("dispatchPackCoverage (coverage gate type): %v", err)
	}
	if len(result.Records) != 2 {
		t.Fatalf("a coverage engine must route to the records parser and yield records, got %d: %#v", len(result.Records), result.Records)
	}
	// The records carry RAW COUNTS and the pack-declared metric — proof the
	// coverage-records parser (not the SARIF parser) produced them.
	if result.Records[0].Metric != "statement" || result.Records[0].Total == 0 {
		t.Errorf("expected real coverage records with metric+counts, got %#v", result.Records[0])
	}
}

// TestDispatch_CoverageEngineOutputNotParsedAsSarif proves a coverage engine's
// convert output is NEVER parsed as SARIF — ParsePackFindings is not invoked for a
// GateTypeCoverage engine, so a coverage-records payload is never silently read as
// zero findings (CLM-002). dispatchPackEngines (the SARIF channel) must produce NO
// violations for a coverage-only pack: the coverage engine is partitioned out of the
// findings dispatch.
func TestDispatch_CoverageEngineOutputNotParsedAsSarif(t *testing.T) {
	sandboxRunner := directConvertSandboxRunner(nil)
	packsDir := coverageRoutingPacksDir(t, coverageRecordsJSON())
	manifest := gateTypeRoutingManifest("cov-engine", engine.GateTypeCoverage)
	runner := &fixtureRunner{byCmd: map[string][]byte{"neutral-tool": []byte("raw coverage profile")}}

	// The SARIF channel must NOT absorb the coverage engine: dispatchPackEngines runs
	// only the generic findings stages over rules whose engine is NOT a dedicated-step
	// gate-type, so a coverage-only pack yields zero SARIF violations.
	result, err := dispatchPackEnginesWithEvidence(excludeDedicatedStepRules([]*pack.Manifest{manifest}), packsDir, t.TempDir(), nil, runner, sandboxRunner)
	if err != nil {
		t.Fatalf("dispatchPackEngines over a coverage-only pack: %v", err)
	}
	if len(result.Violations) != 0 {
		t.Fatalf("a coverage engine's output must NEVER be parsed as SARIF; got %d findings: %#v", len(result.Violations), result.Violations)
	}
}

// TestDispatch_LintEngineRoutesToSarifNotCoverage proves a LINT engine (golangci,
// GateTypeLint) routes to SARIF (ParsePackFindings), NOT the records channel — the
// records channel does not absorb a lint engine (CLM-003). dispatchPackCoverage must
// produce NO records for a lint engine.
func TestDispatch_LintEngineRoutesToSarifNotCoverage(t *testing.T) {
	assertNonCoverageGateTypeYieldsNoRecords(t, "lint-engine", engine.GateTypeLint)
}

// TestDispatch_BuildEngineRoutesToSarifNotCoverage proves a BUILD engine (go-build,
// GateTypeBuild) routes to SARIF, NOT the records channel (CLM-004).
func TestDispatch_BuildEngineRoutesToSarifNotCoverage(t *testing.T) {
	assertNonCoverageGateTypeYieldsNoRecords(t, "build-engine", engine.GateTypeBuild)
}

// TestDispatch_TestEngineRoutesToSarifNotCoverage proves a TEST engine (go-test,
// GateTypeTest) routes to SARIF, NOT the records channel — a test-failure engine and
// a coverage engine over the same `go test` family stay distinct channels (CLM-005).
func TestDispatch_TestEngineRoutesToSarifNotCoverage(t *testing.T) {
	assertNonCoverageGateTypeYieldsNoRecords(t, "test-engine", engine.GateTypeTest)
}

// TestDispatch_FindingsEngineRoutesToSarifNotCoverage proves a FINDINGS engine
// (semgrep/ast-grep, GateTypeFindings) routes to SARIF, NOT the records channel
// (CLM-006).
func TestDispatch_FindingsEngineRoutesToSarifNotCoverage(t *testing.T) {
	assertNonCoverageGateTypeYieldsNoRecords(t, "findings-engine", engine.GateTypeFindings)
}

// assertNonCoverageGateTypeYieldsNoRecords dispatches a manifest carrying an engine
// of a NON-coverage gate_type through dispatchPackCoverage and asserts it yields zero
// coverage records — the records channel routes SOLELY on the declared
// GateTypeCoverage and never absorbs a lint/build/test/findings engine. Even though
// the (irrelevant) convert here would emit SARIF, the engine is skipped by the
// coverage dispatch entirely.
func assertNonCoverageGateTypeYieldsNoRecords(t *testing.T, engineName string, gt engine.GateType) {
	t.Helper()
	sandboxRunner := directConvertSandboxRunner(nil)
	packsDir := coverageRoutingPacksDir(t, sarifFindingsJSON())
	manifest := gateTypeRoutingManifest(engineName, gt)
	runner := &fixtureRunner{byCmd: map[string][]byte{"neutral-tool": []byte("raw output")}}

	result, err := dispatchPackCoverageWithEvidence([]*pack.Manifest{manifest}, packsDir, t.TempDir(), nil, runner, sandboxRunner)
	if err != nil {
		t.Fatalf("dispatchPackCoverage over a %s engine: %v", gt, err)
	}
	if len(result.Records) != 0 {
		t.Fatalf("a %s engine must route to SARIF, NOT the coverage-records channel; got %d records: %#v", gt, len(result.Records), result.Records)
	}
}
