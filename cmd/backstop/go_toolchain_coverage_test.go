package main

import (
	"path/filepath"
	"testing"

	"github.com/bmanson/backstop-core/pkg/check"
	"github.com/bmanson/backstop-core/pkg/pack/engine"
)

// TestGoToolchain_CoverageEngineDeclaredWithCoverageGateType proves the go-toolchain
// pack declares a coverage engine whose binding fills gate_type coverage, whose
// command runs `go test -coverprofile`, and whose pack-relative convert
// (scripts/coverage-to-records.sh) is declared — asserted against the parsed pack.yml
// + binding records, AND asserted to be declared via the pack's engines: block (the
// SPEC-035 declared substrate), NOT the baked DefaultRegistry (CLM-020, Review Q7).
func TestGoToolchain_CoverageEngineDeclaredWithCoverageGateType(t *testing.T) {
	m := goToolchainManifest(t)

	// Declared via the pack's engines: block (the declared substrate), not the baked
	// DefaultRegistry. A coverage engine must appear in m.Engines.
	spec, ok := m.Engines["go-coverage"]
	if !ok {
		t.Fatalf("go-toolchain must declare a go-coverage engine in its engines: block (declared substrate), got engines %v", engineKeysOf(m.Engines))
	}

	// The engine must NOT be a baked DefaultRegistry binding — the baked registry has
	// no go-coverage engine.
	if _, err := engine.DefaultRegistry().Lookup("go-coverage"); err == nil {
		t.Errorf("go-coverage must be a PACK-declared engine, not a baked DefaultRegistry binding")
	}

	b := spec.Binding
	if b.GateType != engine.GateTypeCoverage {
		t.Errorf("go-coverage binding must declare gate_type coverage, got %v", b.GateType)
	}
	if !containsAll(b.Command, "go", "test", "coverprofile") {
		t.Errorf("go-coverage command must run `go test -coverprofile`, got %q", b.Command)
	}
	if b.Convert != "scripts/coverage-to-records.sh" {
		t.Errorf("go-coverage must declare convert scripts/coverage-to-records.sh, got %q", b.Convert)
	}

	// A coverage rule must bind the engine so the rule resolves to it.
	foundRule := false
	for _, r := range m.Content.Ruleset.Rules {
		if r.Engine == "go-coverage" {
			foundRule = true
		}
	}
	if !foundRule {
		t.Errorf("go-toolchain must carry a rule bound to the go-coverage engine")
	}
}

// TestGoToolchain_ConvertProfileToPerFileStatementRecords proves the go-toolchain
// convert script turns a REAL Go coverage profile into per-file CoverageRecords
// stamped Metric "statement" (Go's -coverprofile granularity), with a record for at
// least one measured-and-PASSING file (not only shortfalls) — proven by running the
// actual scripts/coverage-to-records.sh over the Phase-1 fixture and parsing its
// stdout via check.ParsePackCoverage (CLM-021).
func TestGoToolchain_ConvertProfileToPerFileStatementRecords(t *testing.T) {
	convertPath := filepath.Join(goToolchainPackRoot(t), "scripts", "coverage-to-records.sh")
	profile := readCoverageProfileFixture(t, "cover-combined.out")

	out, err := runConvertScriptDirect(convertPath, profile)
	if err != nil {
		t.Fatalf("running coverage-to-records.sh: %v\noutput: %s", err, out)
	}
	records, err := check.ParsePackCoverage(out)
	if err != nil {
		t.Fatalf("parsing convert output as coverage-records: %v\noutput: %s", err, out)
	}

	byFile := map[string]check.CoverageRecord{}
	for _, r := range records {
		byFile[r.Path] = r
		if r.Metric != "statement" {
			t.Errorf("Go's -coverprofile granularity is statement; record %q has metric %q", r.Path, r.Metric)
		}
		if !r.Measured {
			t.Errorf("every file in the profile was measured; %q has Measured=false", r.Path)
		}
	}

	// Per-FILE aggregation: the passing file (9 covered / 10 total) must be present as
	// a measured-and-PASSING record — not only the shortfall file.
	passing, ok := findRecordBySuffix(byFile, "passing.go")
	if !ok {
		t.Fatalf("convert must emit a record for the measured-and-PASSING file, got %v", keysOfRecords(byFile))
	}
	if passing.Total == 0 || passing.Covered*100 < 80*passing.Total {
		t.Errorf("the passing file must be measured-and-passing (covered/total high), got %#v", passing)
	}

	// The failing file is a shortfall record.
	failing, ok := findRecordBySuffix(byFile, "failing.go")
	if !ok {
		t.Fatalf("convert must emit a record for the measured-and-FAILING file")
	}
	if failing.Covered*100 >= 80*failing.Total {
		t.Errorf("the failing file must be below threshold, got %#v", failing)
	}

	// The no-executable-lines file aggregates to Total==0 (the N/A cell), never coerced.
	iface, ok := findRecordBySuffix(byFile, "iface.go")
	if !ok {
		t.Fatalf("convert must emit a record for the no-executable-lines file")
	}
	if iface.Total != 0 {
		t.Errorf("a no-executable-lines file must aggregate to Total==0 (N/A), got %#v", iface)
	}
}
