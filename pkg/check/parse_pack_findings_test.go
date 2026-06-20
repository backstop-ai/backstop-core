package check

import (
	"os"
	"path/filepath"
	"testing"
)

// ParsePackFindings is the SARIF entry the go-toolchain engine path (convert
// scripts + golangci v2 native SARIF) parses its normalized output through. It
// STAYS after the SPEC-034 cutover (the bespoke Go parsers it once sat beside
// were deleted), so it must remain genuinely tested in-package with hardcoded
// expected findings — not a bespoke comparison.

// readGoToolchainFixture reads a shared SPEC-034 captured-output fixture from the
// go-toolchain testdata.
func readGoToolchainFixture(t *testing.T, name string) []byte {
	t.Helper()
	p := filepath.Join("..", "..", "cmd", "backstop", "testdata", "go-toolchain", "fixtures", name)
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return b
}

// TestParsePackFindings_GolangciV2Sarif asserts ParsePackFindings normalizes the
// golangci v2 native SARIF fixture into the located findings and SARIF
// level->severity mapping (error/warning), plus the empty (clean) and malformed
// (fail-loud) branches.
func TestParsePackFindings_GolangciV2Sarif(t *testing.T) {
	vs, err := ParsePackFindings(readGoToolchainFixture(t, "golangci-v2.sarif"))
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
