package gate

import (
	"context"
	"encoding/json"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// self_rule_test.go pins the backstop/self Family B5 rule (ISSUE-062 REQ-005 / CLM-009):
// no-structural-name-split-on-spine flags a whitespace-split name extractor on the
// neutral gate spine (the tokenValue defect) and leaves a structured-property read clean.
// It runs the REAL semgrep engine over a testdata copy of the pack rule + its two
// fixtures (whose filenames match the rule's include hooks). semgrep absence is a Fatal
// (never Skip): a skipped rule test is the vacuous green the anti-vacuous rule kills.

// runSelfRule runs semgrep with the self-pack rule file over target and returns the set
// of matched check_ids.
func runSelfRule(t *testing.T, rule, target string) []string {
	t.Helper()
	if _, err := exec.LookPath("semgrep"); err != nil {
		t.Fatalf("semgrep not found on PATH: %v — install semgrep; this real-engine rule test MUST NOT be skipped (a skip is silent vacuous green)", err)
	}
	cmd := exec.CommandContext(context.Background(), "semgrep", "--config", rule, "--json", "--quiet", target)
	out, err := cmd.Output()
	if err != nil {
		// semgrep exits non-zero only on error here (findings are exit 0 with --json);
		// surface stderr for diagnosis.
		if ee, ok := err.(*exec.ExitError); ok {
			t.Fatalf("semgrep failed: %v (stderr: %s)", err, string(ee.Stderr))
		}
		t.Fatalf("semgrep failed: %v", err)
	}
	var parsed struct {
		Results []struct {
			CheckID string `json:"check_id"`
		} `json:"results"`
	}
	if err := json.Unmarshal(out, &parsed); err != nil {
		t.Fatalf("parsing semgrep JSON: %v\noutput: %s", err, string(out))
	}
	ids := make([]string, 0, len(parsed.Results))
	for _, r := range parsed.Results {
		ids = append(ids, r.CheckID)
	}
	return ids
}

// containsRule reports whether any matched check_id names the given rule. semgrep
// dot-namespaces the id by the config path (e.g.
// testdata.self-rule.no-structural-name-split-on-spine), so the bare rule id is the
// trailing dot-segment.
func containsRule(ids []string, ruleID string) bool {
	for _, id := range ids {
		if id == ruleID || strings.HasSuffix(id, "."+ruleID) {
			return true
		}
	}
	return false
}

// TestSelfRuleFlagsStructuralNameSplitOnSpine (CLM-009): the B5 self rule flags a
// whitespace-split name extractor (the tokenValue defect) on the neutral spine and does
// NOT flag a structured-property read.
func TestSelfRuleFlagsStructuralNameSplitOnSpine(t *testing.T) {
	const ruleID = "no-structural-name-split-on-spine"
	rule := filepath.Join("testdata", "self-rule", "no-baked.yml")

	// Negative fixture (a whitespace-split name extractor) MUST trigger the rule.
	violation := filepath.Join("testdata", "self-rule", "structural-name-split.go")
	got := runSelfRule(t, rule, violation)
	if !containsRule(got, ruleID) {
		t.Errorf("the whitespace-split name extractor must be flagged by %s; got check_ids %v", ruleID, got)
	}

	// Positive fixture (a structured Properties read) must be CLEAN of this rule.
	clean := filepath.Join("testdata", "self-rule", "structured-property-read.go")
	gotClean := runSelfRule(t, rule, clean)
	if containsRule(gotClean, ruleID) {
		t.Errorf("a structured-property read must NOT be flagged by %s; got check_ids %v", ruleID, gotClean)
	}
}
