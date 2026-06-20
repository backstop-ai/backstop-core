package main

import (
	"path/filepath"
	"sort"
	"testing"

	"github.com/bmanson/backstop-core/pkg/check"
)

// SPEC-034 phase-2 EQUIVALENCE GATE (REQ-009). These tests prove the NEW engine
// path (the go-toolchain pack convert scripts / golangci v2 native SARIF, parsed
// via check.ParsePackFindings) emits the SAME normalized violations as the
// STILL-PRESENT bespoke pkg/check parsers, for the SAME captured tool output.
//
// They are the deletion LICENSE: phase 3 may not delete the bespoke
// parsers/executors until these exist and pass. To avoid the DD-2 "removed, not
// extracted" risk, the engine side runs the REAL convert scripts against the REAL
// captured fixtures (never canned SARIF) — runConvertScriptDirect shells the
// actual pack script, the same transform production runs via SandboxedRunStdout.
//
// Equivalence is asserted on the located-finding tuple (File, Line, Message,
// Severity) — the normalization contract both paths must agree on. Rule/Pass
// differ by design (the engine namespaces to the pack; the bespoke parser stamps
// a CheckType), so they are excluded from the tuple, not silently ignored: the
// located-finding tuple IS the normalization the spec requires to be identical.

// normViolation is the comparison tuple: the located-finding fields both the
// bespoke parser and the engine path must normalize identically (CLM-030..032).
type normViolation struct {
	File     string
	Line     int
	Message  string
	Severity string
}

func normalize(vs []check.Violation) []normViolation {
	out := make([]normViolation, 0, len(vs))
	for _, v := range vs {
		out = append(out, normViolation{File: v.File, Line: v.Line, Message: v.Message, Severity: v.Severity})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].File != out[j].File {
			return out[i].File < out[j].File
		}
		if out[i].Line != out[j].Line {
			return out[i].Line < out[j].Line
		}
		return out[i].Message < out[j].Message
	})
	return out
}

// assertEquivalent fails with a precise diff when the two normalized sets differ.
func assertEquivalent(t *testing.T, bespoke, engine []check.Violation) {
	t.Helper()
	b := normalize(bespoke)
	e := normalize(engine)
	if len(b) == 0 {
		t.Fatal("bespoke path produced zero violations; an equivalence proof on an empty set is vacuous")
	}
	if len(e) == 0 {
		t.Fatal("engine path produced zero violations; an equivalence proof on an empty set is vacuous")
	}
	if len(b) != len(e) {
		t.Fatalf("violation COUNT differs: bespoke=%d engine=%d\n bespoke=%+v\n engine=%+v", len(b), len(e), b, e)
	}
	for i := range b {
		if b[i] != e[i] {
			t.Errorf("violation[%d] differs:\n bespoke=%+v\n engine =%+v", i, b[i], e[i])
		}
	}
}

// engineBuildFindings runs the REAL build convert script over the raw fixture and
// parses the resulting SARIF the way the engine dispatch path does
// (check.ParsePackFindings == parseSarif).
func engineBuildFindings(t *testing.T) []check.Violation {
	t.Helper()
	raw := readFixture(t, "go-build-errors.txt")
	scriptPath := buildConvertScript(t)
	sarif, err := runConvertScriptDirect(scriptPath, raw)
	if err != nil {
		t.Fatalf("build convert script failed: %v", err)
	}
	vs, perr := check.ParsePackFindings(sarif)
	if perr != nil {
		t.Fatalf("parsing convert SARIF: %v", perr)
	}
	return vs
}

// engineTestFindings runs the REAL test convert script over the raw fixture.
func engineTestFindings(t *testing.T) []check.Violation {
	t.Helper()
	raw := readFixture(t, "go-test-failures.txt")
	sarif, err := runConvertScriptDirect(testConvertScript(t), raw)
	if err != nil {
		t.Fatalf("test convert script failed: %v", err)
	}
	vs, perr := check.ParsePackFindings(sarif)
	if perr != nil {
		t.Fatalf("parsing convert SARIF: %v", perr)
	}
	return vs
}

func buildConvertScript(t *testing.T) string {
	t.Helper()
	return packScript(t, "build-to-sarif.sh")
}

func testConvertScript(t *testing.T) string {
	t.Helper()
	return packScript(t, "test-to-sarif.sh")
}

func packScript(t *testing.T, name string) string {
	t.Helper()
	return filepath.Join(goToolchainPackRoot(t), "scripts", name)
}

// TestGoToolchainConverter_BuildErrorsEquivalent proves the build convert script
// normalizes REAL `go build` compiler-error output into the SAME file:line:message
// findings the retired parseGoBuildErrors produced (CLM-008).
func TestGoToolchainConverter_BuildErrorsEquivalent(t *testing.T) {
	raw := readFixture(t, "go-build-errors.txt")
	bespoke := check.ParseGoBuildErrorsForTest(raw)
	engine := engineBuildFindings(t)
	assertEquivalent(t, bespoke, engine)
}

// TestGoToolchainConverter_TestFailuresEquivalent proves the test convert script
// normalizes REAL `go test` failure output into the SAME file:line:message
// findings the retired parseGoTestFailures produced — including the no-position
// FAIL block (TestNoPos -> "TestNoPos failed", File="", Line=0) (CLM-009).
func TestGoToolchainConverter_TestFailuresEquivalent(t *testing.T) {
	raw := readFixture(t, "go-test-failures.txt")
	bespoke := check.ParseGoTestFailuresForTest(raw)
	engine := engineTestFindings(t)
	assertEquivalent(t, bespoke, engine)
}

// TestEquivalence_BuildBespokeVsEngine is the phase-2 build equivalence gate: for
// the same go build output, the engine path equals the bespoke parseGoBuildErrors
// normalized violations (CLM-030).
func TestEquivalence_BuildBespokeVsEngine(t *testing.T) {
	raw := readFixture(t, "go-build-errors.txt")
	bespoke := check.ParseGoBuildErrorsForTest(raw)
	engine := engineBuildFindings(t)
	// Substantive sanity: the build fixture has exactly 3 parseable compiler
	// errors (two with columns, one column-less) — the equivalence is over a
	// non-trivial, mixed-shape set, not a single line.
	if len(bespoke) != 3 {
		t.Fatalf("expected the bespoke parser to find 3 build errors in the fixture, got %d", len(bespoke))
	}
	assertEquivalent(t, bespoke, engine)
}

// TestEquivalence_TestBespokeVsEngine is the phase-2 test equivalence gate
// (CLM-031): same go test output, engine == bespoke parseGoTestFailures.
func TestEquivalence_TestBespokeVsEngine(t *testing.T) {
	raw := readFixture(t, "go-test-failures.txt")
	bespoke := check.ParseGoTestFailuresForTest(raw)
	engine := engineTestFindings(t)
	if len(bespoke) != 3 {
		t.Fatalf("expected the bespoke parser to find 3 test failures (incl. the no-position block), got %d", len(bespoke))
	}
	assertEquivalent(t, bespoke, engine)
}

// TestEquivalence_LintBespokeVsEngine is the phase-2 lint equivalence gate
// (CLM-032): for the same golangci findings, the engine path (v2 SARIF ->
// parseSarif) equals the bespoke parseGolangciJSON (v1 JSON) normalized lint
// violations. The two fixtures encode the SAME two findings (errcheck error +
// ineffassign warning) in the two tool formats.
func TestEquivalence_LintBespokeVsEngine(t *testing.T) {
	v1 := readFixture(t, "golangci-v1.json")
	bespoke, err := check.ParseGolangciJSONForTest(v1)
	if err != nil {
		t.Fatalf("bespoke parseGolangciJSON: %v", err)
	}
	v2 := readFixture(t, "golangci-v2.sarif")
	engine, perr := check.ParsePackFindings(v2)
	if perr != nil {
		t.Fatalf("engine parseSarif on v2 SARIF: %v", perr)
	}
	if len(bespoke) != 2 {
		t.Fatalf("expected 2 bespoke lint violations (errcheck+ineffassign), got %d", len(bespoke))
	}
	assertEquivalent(t, bespoke, engine)
}

// TestGoLint_V2SarifEquivalentToRetiredJSON proves golangci v2 native SARIF parses
// into the SAME lint violations parseGolangciJSON produced for the equivalent v1
// findings (CLM-017) — the lint half of the parser-equivalence proof, asserted
// on the located-finding tuple including severity (error vs warning).
func TestGoLint_V2SarifEquivalentToRetiredJSON(t *testing.T) {
	v1 := readFixture(t, "golangci-v1.json")
	bespoke, err := check.ParseGolangciJSONForTest(v1)
	if err != nil {
		t.Fatalf("bespoke parseGolangciJSON: %v", err)
	}
	v2 := readFixture(t, "golangci-v2.sarif")
	engine, perr := check.ParsePackFindings(v2)
	if perr != nil {
		t.Fatalf("engine parseSarif: %v", perr)
	}
	// Severity must survive both paths: errcheck=error, ineffassign=warning.
	assertEquivalent(t, bespoke, engine)
	var sawWarning bool
	for _, v := range engine {
		if v.Severity == "warning" {
			sawWarning = true
		}
	}
	if !sawWarning {
		t.Error("expected the ineffassign finding to normalize to severity=warning through the engine path")
	}
}
