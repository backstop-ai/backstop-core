package pack_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/pack"
)

// manifest_success_exit_codes_test.go (PLAN-ISSUE-207 TASK-001, CLM-009). The
// manifests are YAML TEXT driven through the production pack.ParseManifestFile, so
// these tests compile before the field exists: an unwired success_exit_codes key is
// silently dropped by the non-strict decode, and that silence is the red.

// secEnginesYAML returns a pack.yml whose single declared engine `tool` carries the
// given success_exit_codes line (empty line = key absent).
func secEnginesYAML(successLine string) string {
	block := "engines:\n  tool:\n    command: grep -rn -H -I\n    input_mode: pattern-arg\n    input_flag: -e\n    gate_type: findings\n"
	if successLine != "" {
		block += "    " + successLine + "\n"
	}
	return enginesPackYAML(block)
}

// secRequireParseError parses body and requires an error naming the engine and
// the success_exit_codes key.
func secRequireParseError(t *testing.T, successLine string) {
	t.Helper()
	m, err := pack.ParseManifestFile(writePackYAML(t, secEnginesYAML(successLine)))
	if err == nil {
		t.Fatalf("%q must be a parse error, got manifest with binding %+v", successLine, m.Engines["tool"].Binding)
	}
	if !strings.Contains(err.Error(), "success_exit_codes") {
		t.Errorf("%q: parse error must name the success_exit_codes key, got %q", successLine, err)
	}
}

// TestParseManifest_SuccessExitCodesRejectsEmptyList: an explicit empty list is
// not "undeclared" — it would make every exit, including 0, a failure.
func TestParseManifest_SuccessExitCodesRejectsEmptyList(t *testing.T) {
	_, err := pack.ParseManifestFile(writePackYAML(t, secEnginesYAML("success_exit_codes: []")))
	if err == nil {
		t.Fatal("success_exit_codes: [] must be a parse error")
	}
	for _, want := range []string{`engine "tool"`, "success_exit_codes"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("parse error must contain %q, got %q", want, err)
		}
	}
}

// TestParseManifest_SuccessExitCodesRejectsWrongType: the value must be a YAML
// sequence of integers.
func TestParseManifest_SuccessExitCodesRejectsWrongType(t *testing.T) {
	for _, line := range []string{`success_exit_codes: 1`, `success_exit_codes: [0, "x"]`} {
		t.Run(line, func(t *testing.T) {
			// A non-sequence or non-integer value is rejected by the YAML decode of
			// the typed []int field itself, whose error names the offending value
			// and target type rather than the key.
			_, err := pack.ParseManifestFile(writePackYAML(t, secEnginesYAML(line)))
			if err == nil {
				t.Fatalf("%q must be a parse error", line)
			}
			if !strings.Contains(err.Error(), "cannot unmarshal") {
				t.Errorf("%q: want a typed decode error, got %q", line, err)
			}
		})
	}
}

// TestParseManifest_SuccessExitCodesRejectsOutOfRangeDuplicateAndMissingZero: out-of-range,
// duplicate and 0-less sets are config errors.
func TestParseManifest_SuccessExitCodesRejectsOutOfRangeDuplicateAndMissingZero(t *testing.T) {
	for _, line := range []string{
		"success_exit_codes: [0, 256]",
		"success_exit_codes: [0, 1, 1]",
		"success_exit_codes: [1]",
	} {
		t.Run(line, func(t *testing.T) {
			secRequireParseError(t, line)
		})
	}
}

// TestParseManifest_SuccessExitCodesReachBinding: a valid declaration reaches the
// resolved binding (not silently dropped), and an absent key leaves none. Asserted
// through the binding's printed form so the test names no new field.
func TestParseManifest_SuccessExitCodesReachBinding(t *testing.T) {
	m, err := pack.ParseManifestFile(writePackYAML(t, secEnginesYAML("success_exit_codes: [0, 1]")))
	if err != nil {
		t.Fatalf("success_exit_codes: [0, 1] must parse: %v", err)
	}
	if got := fmt.Sprintf("%v", m.Engines["tool"].Binding); !strings.Contains(got, "[0 1]") {
		t.Errorf("declared [0, 1] must reach the resolved binding, got %s", got)
	}

	absent, err := pack.ParseManifestFile(writePackYAML(t, secEnginesYAML("")))
	if err != nil {
		t.Fatalf("absent key must parse: %v", err)
	}
	if got := fmt.Sprintf("%v", absent.Engines["tool"].Binding); strings.Contains(got, "[0 1]") || strings.Contains(got, "[0]") {
		t.Errorf("an absent key must leave the binding with no declared set, got %s", got)
	}
}
