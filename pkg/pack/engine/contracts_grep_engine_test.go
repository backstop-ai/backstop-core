package engine

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

// contracts_grep_engine_test.go (SPEC-038 TASK-005, REQ-005): the grep engine is
// PACK-DECLARED, not baked. These tests assert (a) grep has NO baked
// engine.DefaultRegistry entry (CLM-015), (b) the traceability pack declares grep
// in its engines: block with pattern-arg input, a grep->SARIF convert, and
// gate_type contracts (CLM-015), and (c) the pack-relative grep convert script
// turns REAL grep output into valid SARIF with a physicalLocation (CLM-019) — no
// hand-written SARIF stub.

// repoRoot walks up from the test's working dir (pkg/pack/engine) to the module
// root so the test can resolve the in-repo traceability pack files.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for i := 0; i < 8; i++ {
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			return dir
		}
		dir = filepath.Dir(dir)
	}
	t.Fatal("could not locate module root (go.mod) from test working dir")
	return ""
}

// TestEngine_GrepPackDeclaredNotInDefaultRegistry asserts there is NO baked grep
// entry in engine.DefaultRegistry AND that the traceability pack DECLARES grep in
// its engines: block with the contract-required binding shape: pattern-arg input
// mode, a non-empty input_flag, a grep->SARIF convert script, and gate_type
// contracts (CLM-015). The engine exists ONLY because the pack declares it.
func TestEngine_GrepPackDeclaredNotInDefaultRegistry(t *testing.T) {
	// (a) No baked grep entry — backstop bakes no grep knowledge.
	if _, baked := DefaultRegistry()["grep"]; baked {
		t.Fatal("engine.DefaultRegistry must NOT contain a baked grep entry — grep is pack-declared (REQ-005/CLM-015)")
	}

	// (b) The traceability pack declares grep in its engines: block. Parse only the
	// engines block (the EngineBinding yaml surface) so this leaf-package test does
	// not import pkg/pack (cycle).
	root := repoRoot(t)
	packYML := filepath.Join(root, "pkg", "gate", "testdata", "traceability-pack", "pack.yml")
	data, err := os.ReadFile(packYML)
	if err != nil {
		t.Fatalf("reading traceability pack.yml (TASK-006 must author it): %v", err)
	}
	var manifest struct {
		Engines map[string]struct {
			Command   string `yaml:"command"`
			InputMode string `yaml:"input_mode"`
			InputFlag string `yaml:"input_flag"`
			Convert   string `yaml:"convert"`
			GateType  string `yaml:"gate_type"`
		} `yaml:"engines"`
	}
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("unmarshalling traceability pack engines block: %v", err)
	}
	grep, ok := manifest.Engines["grep"]
	if !ok {
		t.Fatal("traceability pack engines: block must declare a grep engine (CLM-015)")
	}
	if grep.InputMode != string(InputModePatternArg) {
		t.Errorf("grep engine input_mode = %q, want pattern-arg (CLM-015)", grep.InputMode)
	}
	if grep.InputFlag == "" {
		t.Error("grep engine must declare an input_flag for pattern-arg dispatch (CLM-015)")
	}
	if grep.Convert == "" {
		t.Error("grep engine must declare a grep->SARIF convert script (CLM-015)")
	}
	gt, err := ParseGateType(grep.GateType)
	if err != nil {
		t.Fatalf("grep engine gate_type %q must parse: %v", grep.GateType, err)
	}
	if gt != GateTypeContracts {
		t.Errorf("grep engine gate_type = %q, want contracts (CLM-015)", grep.GateType)
	}
}

// TestEngine_GrepConvertScriptEmitsValidSarif runs REAL grep over a real fixture,
// pipes its output through the pack's grep->SARIF convert script, and asserts the
// result is valid SARIF carrying a result with a physicalLocation (CLM-019). No
// hand-written SARIF: the SARIF is produced by converting genuine grep output.
func TestEngine_GrepConvertScriptEmitsValidSarif(t *testing.T) {
	if _, err := exec.LookPath("grep"); err != nil {
		t.Fatalf("real grep is required for this test (no t.Skip): %v", err)
	}
	root := repoRoot(t)
	convert := filepath.Join(root, "pkg", "gate", "testdata", "traceability-pack", "grep", "to-sarif.sh")
	if _, err := os.Stat(convert); err != nil {
		t.Fatalf("grep convert script must exist (TASK-006 authors it): %v", err)
	}
	fixture := filepath.Join(root, "pkg", "gate", "testdata", "contract-absence-present.go")

	// Run REAL grep: -rn -e <forbidden> <file> → "file:line:matchline".
	grepOut, _ := runGrep(t, "legacyProbeSymbol", fixture)
	if len(bytes.TrimSpace(grepOut)) == 0 {
		t.Fatal("real grep produced no match on a fixture that contains the forbidden symbol — fixture/grep invariant broken")
	}

	// Pipe genuine grep output through the pack's convert script.
	sarif := runConvertScript(t, convert, grepOut)

	var doc struct {
		Version string `json:"version"`
		Runs    []struct {
			Results []struct {
				Message struct {
					Text string `json:"text"`
				} `json:"message"`
				Locations []struct {
					PhysicalLocation struct {
						ArtifactLocation struct {
							URI string `json:"uri"`
						} `json:"artifactLocation"`
						Region struct {
							StartLine int `json:"startLine"`
						} `json:"region"`
					} `json:"physicalLocation"`
				} `json:"locations"`
			} `json:"results"`
		} `json:"runs"`
	}
	if err := json.Unmarshal(sarif, &doc); err != nil {
		t.Fatalf("convert output is not valid SARIF JSON: %v\noutput: %s", err, sarif)
	}
	if doc.Version == "" {
		t.Error("SARIF must declare a version")
	}
	if len(doc.Runs) == 0 || len(doc.Runs[0].Results) == 0 {
		t.Fatalf("SARIF must carry at least one result from the real grep match, got: %s", sarif)
	}
	res := doc.Runs[0].Results[0]
	if len(res.Locations) == 0 {
		t.Fatal("SARIF result must carry a physicalLocation (CLM-019)")
	}
	loc := res.Locations[0].PhysicalLocation
	if loc.ArtifactLocation.URI == "" {
		t.Error("physicalLocation must carry an artifactLocation.uri")
	}
	if loc.Region.StartLine <= 0 {
		t.Error("physicalLocation must carry a 1-indexed region.startLine")
	}
}

// runGrep runs real grep -rn -e <pattern> <target> and returns its stdout. grep
// exits non-zero when there is no match; a match yields exit 0 with stdout, so a
// non-nil err on a match-present fixture is unexpected but stdout is the contract.
func runGrep(t *testing.T, pattern, target string) ([]byte, error) {
	t.Helper()
	cmd := exec.Command("grep", "-rn", "-e", pattern, target)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.Bytes(), err
}

// runConvertScript shells the pack convert script with stdin and returns stdout.
func runConvertScript(t *testing.T, convert string, stdin []byte) []byte {
	t.Helper()
	cmd := exec.Command("/bin/sh", convert)
	cmd.Stdin = bytes.NewReader(stdin)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("grep convert script failed: %v (stderr: %s)", err, stderr.String())
	}
	return stdout.Bytes()
}
