package main

import (
	"bytes"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/bmanson/backstop-core/pkg/check"
)

// engineFixturePackRoot returns the absolute path to the in-worktree
// engine-dispatch fixture pack directory.
func engineFixturePackRoot(t *testing.T) string {
	t.Helper()
	return filepath.Join(repoRoot(t), "cmd", "backstop", "testdata", "engine-dispatch",
		".backstop", "packs", "test-org", "engine-pack")
}

// TestAstGrepConverter_RealJSONToValidSarif is the converter-correctness proof
// (CLM-068 / REQ-008, Sharp Edge 6 / Review Question 9). It runs the REAL
// converter script the pack ships (ast-grep/to-sarif.sh, NOT the stub) against a
// checked-in sample of REAL `ast-grep scan --json` output (ast-grep/real-scan.json)
// and asserts the script's stdout is SARIF that the gate's SARIF parser accepts
// AND that the emitted SARIF carries the sample's finding: the rule id and the
// file/line location survive the transform. This proves the genuine JSON->SARIF
// transform, so "ast-grep wired end-to-end" cannot be satisfied by canned bytes.
func TestAstGrepConverter_RealJSONToValidSarif(t *testing.T) {
	if _, err := exec.LookPath("jq"); err != nil {
		t.Skip("jq not installed; the real ast-grep converter requires jq")
	}
	packRoot := engineFixturePackRoot(t)
	converter := filepath.Join(packRoot, "ast-grep", "to-sarif.sh")
	sample := readFileStr(t, filepath.Join(packRoot, "ast-grep", "real-scan.json"))

	// Run the REAL converter directly: stdin = real ast-grep JSON, capture stdout.
	cmd := exec.Command("/bin/sh", converter)
	cmd.Stdin = bytes.NewReader([]byte(sample))
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		t.Fatalf("running real converter %s: %v", converter, err)
	}

	// The converter's stdout must be SARIF the gate's parser accepts, via the
	// SAME lookupParser/parseSarif path dispatch uses.
	violations, err := check.ParsePackFindings(stdout.Bytes())
	if err != nil {
		t.Fatalf("converter stdout is not parseSarif-valid: %v\noutput: %s", err, stdout.String())
	}
	if len(violations) != 1 {
		t.Fatalf("expected exactly 1 finding from the real sample, got %d: %#v", len(violations), violations)
	}
	v := violations[0]
	// The sample's finding identity must survive the transform.
	if v.Rule != "ast-grep-proof" {
		t.Errorf("ruleId not preserved: got %q, want ast-grep-proof", v.Rule)
	}
	if v.File != "main.go" {
		t.Errorf("file not preserved: got %q, want main.go", v.File)
	}
	// ast-grep reports 0-indexed lines (sample line 2); the converter +1 → 3.
	if v.Line != 3 {
		t.Errorf("line not transformed 0-indexed->1-indexed: got %d, want 3", v.Line)
	}
	if v.Message != "forbiddenCall is not allowed" {
		t.Errorf("message not preserved: got %q", v.Message)
	}
}

// TestAstGrepPack_ShipsOwnConverter proves the ast-grep pack supplies its own
// stdin->SARIF converter referenced by the engine binding's convert field
// (CLM-027 / REQ-008): the ast-grep EngineBinding declares a convert script and
// that script exists inside the pack directory.
func TestAstGrepPack_ShipsOwnConverter(t *testing.T) {
	b, err := resolveEngineRegistry(nil).Lookup("ast-grep")
	if err != nil {
		t.Fatalf("ast-grep engine not registered: %v", err)
	}
	if b.Convert == "" {
		t.Fatal("ast-grep EngineBinding declares no convert script; the pack must ship its own converter")
	}
	packRoot := engineFixturePackRoot(t)
	converter := filepath.Join(packRoot, filepath.FromSlash(b.Convert))
	// The pack ships the real converter at the binding's declared path.
	if got := readFileStr(t, converter); got == "" {
		t.Fatalf("converter %s is empty", converter)
	}
}
