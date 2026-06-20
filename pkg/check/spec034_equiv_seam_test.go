package check

import (
	"os"
	"path/filepath"
	"testing"
)

// readEquivFixture reads a shared SPEC-034 captured-output fixture from the
// go-toolchain testdata so the seam wrappers are exercised on the SAME bytes the
// cmd/backstop equivalence gate uses.
func readEquivFixture(t *testing.T, name string) []byte {
	t.Helper()
	p := filepath.Join("..", "..", "cmd", "backstop", "testdata", "go-toolchain", "fixtures", name)
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return b
}

// TestEquivSeam_ParseGoBuildErrorsForTest covers the build-parser seam wrapper and
// asserts it normalizes the captured `go build` output into the expected located
// findings (so the wrapper is genuinely exercised in-package, not just by the
// cross-package equivalence gate).
func TestEquivSeam_ParseGoBuildErrorsForTest(t *testing.T) {
	vs := ParseGoBuildErrorsForTest(readEquivFixture(t, "go-build-errors.txt"))
	if len(vs) != 3 {
		t.Fatalf("expected 3 build violations, got %d: %+v", len(vs), vs)
	}
	if vs[0].File != "pkg/widget/widget.go" || vs[0].Line != 14 || vs[0].Message != "undefined: Frobnicate" {
		t.Errorf("first build violation not normalized as expected: %+v", vs[0])
	}
	// The column-less line-8 error must still be located.
	last := vs[len(vs)-1]
	if last.File != "pkg/gadget/gadget.go" || last.Line != 8 {
		t.Errorf("column-less compiler error not located: %+v", last)
	}
}

// TestEquivSeam_ParseGoTestFailuresForTest covers the test-parser seam wrapper,
// including the no-position FAIL block (TestNoPos -> "TestNoPos failed", File="",
// Line=0).
func TestEquivSeam_ParseGoTestFailuresForTest(t *testing.T) {
	vs := ParseGoTestFailuresForTest(readEquivFixture(t, "go-test-failures.txt"))
	if len(vs) != 3 {
		t.Fatalf("expected 3 test violations, got %d: %+v", len(vs), vs)
	}
	var sawNoPos bool
	for _, v := range vs {
		if v.Message == "TestNoPos failed" {
			sawNoPos = true
			if v.File != "" || v.Line != 0 {
				t.Errorf("no-position failure must have empty File/zero Line, got %+v", v)
			}
		}
	}
	if !sawNoPos {
		t.Error("expected the no-position FAIL block to normalize to `TestNoPos failed`")
	}
}

// TestEquivSeam_ParsePackFindingsSarif covers ParsePackFindings — the SARIF entry
// the go-toolchain engine path parses convert/lint output through (it STAYS after
// the cutover, unlike the bespoke parsers). Asserts the located findings and the
// SARIF level->severity mapping (error/warning) on the golangci v2 SARIF fixture,
// so the engine-side of the equivalence gate is exercised in-package too.
func TestEquivSeam_ParsePackFindingsSarif(t *testing.T) {
	vs, err := ParsePackFindings(readEquivFixture(t, "golangci-v2.sarif"))
	if err != nil {
		t.Fatalf("ParsePackFindings: %v", err)
	}
	if len(vs) != 2 {
		t.Fatalf("expected 2 SARIF findings, got %d: %+v", len(vs), vs)
	}
	var gotError, gotWarning bool
	for _, v := range vs {
		switch v.Severity {
		case "error":
			gotError = true
			if v.File != "pkg/widget/widget.go" || v.Line != 14 {
				t.Errorf("error finding not located: %+v", v)
			}
		case "warning":
			gotWarning = true
		}
	}
	if !gotError || !gotWarning {
		t.Errorf("SARIF level must map to error+warning severities, got %+v", vs)
	}

	// Empty input is a clean (no-findings) parse, not an error.
	empty, eerr := ParsePackFindings([]byte("  \n"))
	if eerr != nil {
		t.Fatalf("empty SARIF must parse cleanly, got: %v", eerr)
	}
	if len(empty) != 0 {
		t.Errorf("empty SARIF must yield zero findings, got %+v", empty)
	}

	// Malformed (non-JSON) input must fail loud, not silently read as zero.
	if _, berr := ParsePackFindings([]byte("not json at all")); berr == nil {
		t.Error("malformed SARIF must return a parse error, not a silent zero-findings green")
	}
}

// TestEquivSeam_ParseGolangciJSONForTest covers the lint-parser seam wrapper on
// golangci v1 JSON, asserting severity survives (errcheck=error, ineffassign=warning).
func TestEquivSeam_ParseGolangciJSONForTest(t *testing.T) {
	vs, err := ParseGolangciJSONForTest(readEquivFixture(t, "golangci-v1.json"))
	if err != nil {
		t.Fatalf("ParseGolangciJSONForTest: %v", err)
	}
	if len(vs) != 2 {
		t.Fatalf("expected 2 lint violations, got %d: %+v", len(vs), vs)
	}
	bySev := map[string]string{}
	for _, v := range vs {
		bySev[v.Severity] = v.Message
	}
	if bySev["error"] == "" || bySev["warning"] == "" {
		t.Errorf("expected one error and one warning severity, got %+v", vs)
	}
}
