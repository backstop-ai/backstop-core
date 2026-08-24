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

// TestDispatch_CoverageNotTunneledThroughSarifProperties proves the coverage channel
// is a DISTINCT typed output ([]check.CoverageRecord), never coverage tunneled
// through a SARIF result's properties — a convert emitting a SARIF document carrying
// coverage-shaped result.properties is NOT accepted by the coverage path (the
// records parser rejects the SARIF object), so coverage cannot be smuggled through
// SARIF (CLM-007).
func TestDispatch_CoverageNotTunneledThroughSarifProperties(t *testing.T) {
	sandboxRunner := directConvertSandboxRunner(nil)
	// The convert emits a SARIF document with coverage in result.properties — the
	// cheap-but-wrong tunneling shape the coverage path must REJECT.
	tunneled, err := os.ReadFile(filepath.Join(repoRoot(t), "pkg", "check", "testdata", "coverage-tunneled-through-sarif-properties.json"))
	if err != nil {
		t.Fatalf("read tunneling fixture: %v", err)
	}
	packsDir := coverageRoutingPacksDir(t, string(tunneled))
	manifest := gateTypeRoutingManifest("cov-engine", engine.GateTypeCoverage)
	runner := &fixtureRunner{byCmd: map[string][]byte{"neutral-tool": []byte("raw")}}

	_, dErr := dispatchPackCoverageWithEvidence([]*pack.Manifest{manifest}, packsDir, t.TempDir(), nil, runner, sandboxRunner)
	if dErr == nil {
		t.Fatal("a SARIF document carrying coverage in result.properties must be REJECTED by the coverage path, not tunneled as records")
	}
	if !strings.Contains(strings.ToLower(dErr.Error()), "sarif") && !strings.Contains(strings.ToLower(dErr.Error()), "array") {
		t.Errorf("rejection must name the SARIF/records-shape mismatch, got %v", dErr)
	}
}

// TestCoverage_ChannelCarriesMeasuredPassingFiles proves the channel carries a record
// for a MEASURED-AND-PASSING file (Measured true, Covered/Total above any threshold)
// — the census includes passing files, not only shortfalls; an engine emitting only
// below-threshold records would be a SARIF-findings stream in disguise (CLM-008).
func TestCoverage_ChannelCarriesMeasuredPassingFiles(t *testing.T) {
	sandboxRunner := directConvertSandboxRunner(nil)
	packsDir := coverageRoutingPacksDir(t, coverageRecordsJSON())
	manifest := gateTypeRoutingManifest("cov-engine", engine.GateTypeCoverage)
	runner := &fixtureRunner{byCmd: map[string][]byte{"neutral-tool": []byte("raw")}}

	result, err := dispatchPackCoverageWithEvidence([]*pack.Manifest{manifest}, packsDir, t.TempDir(), nil, runner, sandboxRunner)
	if err != nil {
		t.Fatalf("dispatchPackCoverage: %v", err)
	}
	var passing *check.CoverageRecord
	for i := range result.Records {
		if result.Records[i].Measured && result.Records[i].Covered*100 >= 90*result.Records[i].Total && result.Records[i].Total > 0 {
			passing = &result.Records[i]
			break
		}
	}
	if passing == nil {
		t.Fatalf("the census MUST carry a measured-and-passing record, got %#v", result.Records)
	}
	if !passing.Measured {
		t.Errorf("the passing record must be Measured=true, got %#v", *passing)
	}
}

// TestCoverage_MeasuredPassedVsNotMeasuredDistinguished proves a file the engine
// should have measured but did NOT (absent from the records) is distinguishable as
// not-measured from measured-and-passed — the load-bearing reason coverage cannot be
// SARIF (CLM-009). Feeding a record-set that OMITS an expected file, the
// not-measured signal (no record / Measured false) survives to the consumer boundary.
func TestCoverage_MeasuredPassedVsNotMeasuredDistinguished(t *testing.T) {
	sandboxRunner := directConvertSandboxRunner(nil)
	// Only handler.go is measured; an EXPECTED file (omitted.go) has no record.
	convert := `[{"path":"pkg/svc/handler.go","covered":92,"total":100,"measured":true,"excluded":false,"metric":"statement"}]`
	packsDir := coverageRoutingPacksDir(t, convert)
	manifest := gateTypeRoutingManifest("cov-engine", engine.GateTypeCoverage)
	runner := &fixtureRunner{byCmd: map[string][]byte{"neutral-tool": []byte("raw")}}

	result, err := dispatchPackCoverageWithEvidence([]*pack.Manifest{manifest}, packsDir, t.TempDir(), nil, runner, sandboxRunner)
	if err != nil {
		t.Fatalf("dispatchPackCoverage: %v", err)
	}
	measured := map[string]bool{}
	for _, r := range result.Records {
		measured[r.Path] = r.Measured
	}
	if !measured["pkg/svc/handler.go"] {
		t.Errorf("the measured-and-passed file must carry Measured=true, got %#v", result.Records)
	}
	// The omitted file is not-measured: it has NO record, distinguishable from a
	// measured-and-passed record. SARIF-as-findings cannot carry this distinction.
	if _, present := measured["pkg/svc/omitted.go"]; present {
		t.Errorf("an unmeasured file must be ABSENT from the records (not-measured), got a record for it")
	}
}

// TestDispatch_NoBakedGoCoverageKnowledge is the thin-executor guard: it asserts
// cmd/backstop/pack_gate.go constructs no `go test -coverprofile` command and parses
// no Go coverage-profile text in the coverage path — the Go-toolchain coverage
// knowledge lives in the pack (binding command + convert script), not the binary
// (CLM-022). It is a source guard over the dispatch file.
func TestDispatch_NoBakedGoCoverageKnowledge(t *testing.T) {
	src, err := os.ReadFile(filepath.Join(repoRoot(t), "cmd", "backstop", "pack_gate.go"))
	if err != nil {
		t.Fatalf("read pack_gate.go: %v", err)
	}
	text := string(src)
	for _, baked := range []string{"-coverprofile", "coverprofile", "go test -cover", "mode: atomic", "mode: set", "mode: count"} {
		if strings.Contains(text, baked) {
			t.Errorf("pack_gate.go must bake NO Go-coverage knowledge; found %q in the dispatch source", baked)
		}
	}
}
