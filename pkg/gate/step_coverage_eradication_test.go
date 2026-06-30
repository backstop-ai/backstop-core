package gate

import (
	"os"
	"strings"
	"testing"
)

// readStepCoverageSource reads pkg/gate/step_coverage.go (the coverage step
// successor) as text so the eradication guards can assert the baked Go-toolchain
// machinery is gone by source inspection — the only faithful way to prove a
// constructor/parser/reader no longer exists in the binary.
func readStepCoverageSource(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile("step_coverage.go")
	if err != nil {
		t.Fatalf("read step_coverage.go: %v", err)
	}
	return string(data)
}

// TestCoverage_NoBakedGoTestCommandRemains asserts the coverage step constructs
// NO `go test` command and contains none of the Go-toolchain coverage-target
// constructors (CLM-001). After the eradication the gate-side coverage step
// holds no `go test` invocation — coverage's signal is the declared toolchain
// test pass (SPEC-040/SPEC-042), not an in-binary exec.
func TestCoverage_NoBakedGoTestCommandRemains(t *testing.T) {
	src := readStepCoverageSource(t)
	banned := []string{
		"goCoverageTarget",
		"goCoveragePackagesTarget",
		"repoSweepCoverageTarget",
		"goCoverageTargetsForScope",
		"changedGoCoverageTargets",
		"-coverprofile",
		`"go", "test"`,
		`go", "test"`,
	}
	for _, b := range banned {
		if strings.Contains(src, b) {
			t.Errorf("step_coverage.go still contains baked Go-coverage machinery %q — it must be eradicated (CLM-001)", b)
		}
	}
}

// TestCoverage_NoGoCoverageParsingOrGoModReaderRemains asserts the Go-coverage
// output parser, the whole-module dedup read, and the go.mod reader are gone
// (CLM-002): the step parses no Go-coverage text and holds no go.mod/Go-package
// knowledge.
func TestCoverage_NoGoCoverageParsingOrGoModReaderRemains(t *testing.T) {
	src := readStepCoverageSource(t)
	banned := []string{
		"coverageRe",
		"parseCoverageLine",
		"wholeModulePackageCoverage",
		"packageLabelFromLine",
		"goModulePath",
		"coverage:",          // the `coverage: N% of statements` literal
		"of statements",      // the Go-coverage summary suffix
		"CoverageTarget",     // the baked command descriptor type
	}
	for _, b := range banned {
		if strings.Contains(src, b) {
			t.Errorf("step_coverage.go still contains baked Go-coverage parsing/go.mod knowledge %q — it must be eradicated (CLM-002)", b)
		}
	}
}

// TestCoverageRelevance_NoBakedGoLiteralsInSpecRelevance (CLM-030, source guard):
// no baked `.go`/`_testdata.go`/`./...` string literal remains in
// coverageSpecRelevantToFile or packagePathMatches (the whole step_coverage.go
// file is free of them after the de-Go) — the relevance path keys only on
// directory matching.
func TestCoverageRelevance_NoBakedGoLiteralsInSpecRelevance(t *testing.T) {
	src := readStepCoverageSource(t)
	banned := []string{
		`.go"`,           // a baked Go source-extension suffix literal
		"_testdata.go",   // the baked testdata-suffix gate
		"./...",          // the baked Go-package recursive glob convention
	}
	for _, b := range banned {
		if strings.Contains(src, b) {
			t.Errorf("step_coverage.go still contains a baked Go literal %q in the spec-relevance path — relevance must key only on directory matching (CLM-030)", b)
		}
	}
}
