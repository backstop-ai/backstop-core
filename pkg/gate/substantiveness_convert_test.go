package gate

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
)

// substantiveness_convert_test.go pins the substantiveness pack's ast-grep -> SARIF
// convert (packs/substantiveness/ast-grep/to-sarif.sh, ISSUE-062 REQ-004). It feeds
// the REAL convert a captured `ast-grep scan --json` payload and asserts the emitted
// SARIF lifts the matched metavariables into result-level `properties` (func from $FN,
// symbol from $PKG) verbatim — including a test name with spaces AND quotes — while the
// human-readable message carries no machine-parsed contract. jq absence is a Fatal
// (never Skip): a skipped convert test is the vacuous green the anti-vacuous rule kills.

// repoFile resolves a path relative to the repository root (two levels up from the
// pkg/gate package dir), so the test can reach the pack's real convert + testdata.
func repoFile(parts ...string) string {
	return filepath.Join(append([]string{"..", ".."}, parts...)...)
}

// runConvertScript pipes stdin through the pack convert script via /bin/sh, returning
// its stdout (the SARIF). It mirrors the production clean-stdout convert step.
func runConvertScript(t *testing.T, script string, stdin []byte) []byte {
	t.Helper()
	if _, err := exec.LookPath("jq"); err != nil {
		t.Fatalf("jq not found on PATH: %v — the convert shells jq; this test MUST NOT be skipped (a skip is silent vacuous green)", err)
	}
	cmd := exec.CommandContext(context.Background(), "/bin/sh", script)
	cmd.Stdin = bytes.NewReader(stdin)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("convert script %s failed: %v (stderr: %s)", script, err, stderr.String())
	}
	return stdout.Bytes()
}

// TestSubstantivenessConvert_NullRuleId_NoCrash (ISSUE-069) is a regression guard: an
// ast-grep finding whose `ruleId` is null (or absent entirely) must flow through the
// convert WITHOUT crashing, yield valid SARIF, and stamp NO substantiveness_role — the
// role test is null-safe (`(.ruleId // "")`) so an unroutable finding falls through to
// the default (no role), never aborting the jq program mid-array on a null input.
func TestSubstantivenessConvert_NullRuleId_NoCrash(t *testing.T) {
	script := repoFile("packs", "substantiveness", "ast-grep", "to-sarif.sh")

	// Two adversarial findings: one with an explicit null ruleId, one omitting the key.
	// Both must survive convert and route to NO substantiveness_role.
	nullRuleID := map[string]any{
		"ruleId":   nil,
		"severity": "error",
		"message":  "finding with null ruleId",
		"file":     "mystery.go",
		"range":    map[string]any{"start": map[string]any{"line": 4, "column": 0}},
	}
	absentRuleID := map[string]any{
		"severity": "warning",
		"message":  "finding with absent ruleId",
		"file":     "mystery2.go",
		"range":    map[string]any{"start": map[string]any{"line": 9, "column": 0}},
	}
	astJSON, err := json.Marshal([]map[string]any{nullRuleID, absentRuleID})
	if err != nil {
		t.Fatalf("marshal ast-grep payload: %v", err)
	}

	// runConvertScript Fatals if the convert exits non-zero, so reaching here already
	// proves NO crash.
	got := runConvertScript(t, script, astJSON)

	// The output must be valid SARIF JSON.
	var sarif struct {
		Version string `json:"version"`
		Runs    []struct {
			Results []struct {
				RuleID     *string           `json:"ruleId"`
				Properties map[string]string `json:"properties"`
			} `json:"results"`
		} `json:"runs"`
	}
	if err := json.Unmarshal(got, &sarif); err != nil {
		t.Fatalf("convert output is not valid JSON SARIF: %v\noutput: %s", err, string(got))
	}
	if sarif.Version != "2.1.0" {
		t.Errorf("SARIF version = %q, want 2.1.0", sarif.Version)
	}
	if len(sarif.Runs) != 1 || len(sarif.Runs[0].Results) != 2 {
		t.Fatalf("expected 1 run with 2 results, got %+v", sarif.Runs)
	}

	// Neither unroutable finding may carry a substantiveness_role — it falls through to
	// the default (no role), which the gate partition treats as neither hollow nor
	// referenced.
	for i, res := range sarif.Runs[0].Results {
		if role, ok := res.Properties["substantiveness_role"]; ok {
			t.Errorf("result[%d] with null/absent ruleId must carry NO substantiveness_role, got %q", i, role)
		}
	}
}

// TestSubstantivenessConvertEmitsPropertiesFromMetavars (CLM-008): the convert lifts
// ast-grep metaVariables.single.FN/PKG into SARIF result properties func/symbol,
// preserving a spaced+quoted name verbatim and omitting symbol when $PKG was not
// captured, while retaining the human message.
func TestSubstantivenessConvertEmitsPropertiesFromMetavars(t *testing.T) {
	script := repoFile("packs", "substantiveness", "ast-grep", "to-sarif.sh")
	in, err := os.ReadFile(repoFile("packs", "substantiveness", "testdata", "convert", "metavars.json"))
	if err != nil {
		t.Fatalf("reading metavars.json: %v", err)
	}

	got := runConvertScript(t, script, in)

	// The emitted SARIF must equal the committed golden (order-independent, via map
	// comparison) — this is the machine contract REQ-004 pins.
	want, err := os.ReadFile(repoFile("packs", "substantiveness", "testdata", "convert", "expected.sarif.json"))
	if err != nil {
		t.Fatalf("reading expected.sarif.json: %v", err)
	}
	var gotAny, wantAny any
	if err := json.Unmarshal(got, &gotAny); err != nil {
		t.Fatalf("convert output is not valid JSON: %v\noutput: %s", err, string(got))
	}
	if err := json.Unmarshal(want, &wantAny); err != nil {
		t.Fatalf("golden is not valid JSON: %v", err)
	}
	if !reflect.DeepEqual(gotAny, wantAny) {
		t.Fatalf("convert output does not match golden expected.sarif.json\ngot:  %s\nwant: %s", string(got), string(want))
	}

	// Targeted assertions on the structured channel, independent of the golden.
	var sarif struct {
		Runs []struct {
			Results []struct {
				RuleID     string                `json:"ruleId"`
				Message    struct{ Text string } `json:"message"`
				Properties map[string]string     `json:"properties"`
			} `json:"results"`
		} `json:"runs"`
	}
	if err := json.Unmarshal(got, &sarif); err != nil {
		t.Fatalf("unmarshal convert SARIF: %v", err)
	}
	if len(sarif.Runs) != 1 || len(sarif.Runs[0].Results) != 2 {
		t.Fatalf("expected 1 run with 2 results; got %+v", sarif.Runs)
	}
	res := sarif.Runs[0].Results

	// Result 0: referenced-symbol carries both func and symbol from FN/PKG.
	if res[0].Properties["func"] != "TestReferencesSymbol" || res[0].Properties["symbol"] != "strings" {
		t.Errorf("result[0].properties = %v, want func=TestReferencesSymbol symbol=strings", res[0].Properties)
	}

	// Result 1: a hollow finding whose $FN is a spaced+quoted description carries that
	// name VERBATIM in properties.func, with NO symbol (PKG uncaptured) — the case the
	// deleted message parsers would have truncated at the first space.
	const spacedQuoted = `surfaces a plan "spec_id" in the response`
	if res[1].Properties["func"] != spacedQuoted {
		t.Errorf("result[1].properties[func] = %q, want the verbatim spaced+quoted name %q", res[1].Properties["func"], spacedQuoted)
	}
	if _, ok := res[1].Properties["symbol"]; ok {
		t.Errorf("result[1] must NOT carry a symbol property (PKG was not captured); got %v", res[1].Properties)
	}
	// The machine contract is NOT in the message (human-readable only).
	if res[1].Message.Text == "" {
		t.Errorf("result[1] must retain a human-readable message")
	}
}
