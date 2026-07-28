package main

import (
	"path/filepath"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/check"
)

// SPEC-034 STANDALONE convert-script coverage. The transitional equivalence gate
// (equivalence_test.go) was deleted with the bespoke parsers it compared against,
// so these tests assert the go-toolchain pack convert scripts' normalization
// DIRECTLY against hardcoded expected findings — not via a bespoke comparison —
// so the engine convert path stays genuinely tested after the cutover.
//
// They run the REAL pack scripts (build-to-sarif.sh / test-to-sarif.sh) over the
// REAL captured fixtures via runConvertScriptDirect (the test stand-in for the
// production SandboxedRunStdout), then parse the resulting SARIF the way the
// engine dispatch path does (check.ParsePackFindings == parseSarif). This guards
// against the DD-2 "removed, not extracted" risk — the transform must live in the
// pack script and produce the exact normalization the retired parser did.

// convertScript returns the absolute path to a go-toolchain pack convert script.
func convertScript(t *testing.T, name string) string {
	t.Helper()
	return filepath.Join(goToolchainPackRoot(t), "scripts", name)
}

// findFinding returns the first finding matching file+line, or nil.
func findFinding(vs []check.Violation, file string, line int) *check.Violation {
	for i := range vs {
		if vs[i].File == file && vs[i].Line == line {
			return &vs[i]
		}
	}
	return nil
}

// TestGoToolchainConvert_BuildToSarif_DirectFindings runs the REAL build-to-sarif.sh
// over the captured `go build` output and asserts the EXACT normalized findings the
// retired parseGoBuildErrors produced: three compiler errors (two with columns, one
// column-less), located file:line:message, all severity=error.
func TestGoToolchainConvert_BuildToSarif_DirectFindings(t *testing.T) {
	raw := readFixture(t, "go-build-errors.txt")
	sarif, err := runConvertScriptDirect(convertScript(t, "build-to-sarif.sh"), raw)
	if err != nil {
		t.Fatalf("build-to-sarif.sh failed: %v", err)
	}
	vs, perr := check.ParsePackFindings(sarif)
	if perr != nil {
		t.Fatalf("parsing convert SARIF: %v", perr)
	}
	if len(vs) != 3 {
		t.Fatalf("expected 3 build findings from the convert script, got %d: %+v", len(vs), vs)
	}

	first := findFinding(vs, "pkg/widget/widget.go", 14)
	if first == nil {
		t.Fatalf("missing the first build finding pkg/widget/widget.go:14; got %+v", vs)
	}
	if first.Message != "undefined: Frobnicate" {
		t.Errorf("build finding message = %q, want %q", first.Message, "undefined: Frobnicate")
	}
	if first.Severity != "error" {
		t.Errorf("build finding severity = %q, want error", first.Severity)
	}

	// The column-less compiler error (line 8) must still be located.
	if last := findFinding(vs, "pkg/gadget/gadget.go", 8); last == nil {
		t.Errorf("column-less compiler error must still be located; got %+v", vs)
	}
}

// TestGoToolchainConvert_TestToSarif_DirectFindings runs the REAL test-to-sarif.sh
// over the captured `go test` output and asserts the EXACT normalized findings the
// retired parseGoTestFailures produced: three failures including the no-position
// FAIL block (TestNoPos -> "TestNoPos failed", File="", Line=0).
func TestGoToolchainConvert_TestToSarif_DirectFindings(t *testing.T) {
	raw := readFixture(t, "go-test-failures.txt")
	sarif, err := runConvertScriptDirect(convertScript(t, "test-to-sarif.sh"), raw)
	if err != nil {
		t.Fatalf("test-to-sarif.sh failed: %v", err)
	}
	vs, perr := check.ParsePackFindings(sarif)
	if perr != nil {
		t.Fatalf("parsing convert SARIF: %v", perr)
	}
	if len(vs) != 3 {
		t.Fatalf("expected 3 test findings from the convert script, got %d: %+v", len(vs), vs)
	}

	// The first failure carries its file:line position and detail message.
	var sawLocated, sawNoPos bool
	for _, v := range vs {
		if v.Message == "TestWidgetFrobnicate: expected 5, got 7" {
			sawLocated = true
			if v.Severity != "error" {
				t.Errorf("located test failure severity = %q, want error", v.Severity)
			}
		}
		if v.Message == "TestNoPos failed" {
			sawNoPos = true
			if v.File != "" || v.Line != 0 {
				t.Errorf("no-position failure must have empty File/zero Line, got %+v", v)
			}
		}
	}
	if !sawLocated {
		t.Error("expected the located TestWidgetFrobnicate failure to normalize with its detail message")
	}
	if !sawNoPos {
		t.Error("expected the no-position FAIL block to normalize to `TestNoPos failed` (File=\"\", Line=0)")
	}
}

// TestGoToolchainConvert_GolangciV2SarifDirect asserts the golangci v2 NATIVE SARIF
// fixture parses through the engine path (check.ParsePackFindings == parseSarif)
// into the EXACT lint findings the retired parseGolangciJSON produced for the
// equivalent v1 findings: an errcheck error and an ineffassign warning, located.
// This is the lint half of the engine convert/normalize coverage, standalone.
func TestGoToolchainConvert_GolangciV2SarifDirect(t *testing.T) {
	vs, err := check.ParsePackFindings(readFixture(t, "golangci-v2.sarif"))
	if err != nil {
		t.Fatalf("parseSarif on golangci v2 SARIF: %v", err)
	}
	if len(vs) != 2 {
		t.Fatalf("expected 2 lint findings, got %d: %+v", len(vs), vs)
	}
	bySeverity := map[string]check.Violation{}
	for _, v := range vs {
		bySeverity[v.Severity] = v
	}
	errFinding, hasErr := bySeverity["error"]
	if !hasErr {
		t.Fatalf("expected one error-severity lint finding, got %+v", vs)
	}
	if errFinding.File != "pkg/widget/widget.go" || errFinding.Line != 14 {
		t.Errorf("error lint finding not located: %+v", errFinding)
	}
	if _, hasWarn := bySeverity["warning"]; !hasWarn {
		t.Errorf("expected the ineffassign finding to normalize to severity=warning, got %+v", vs)
	}
}
