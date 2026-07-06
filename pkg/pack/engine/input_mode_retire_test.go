package engine

import (
	"strings"
	"testing"
)

// TestParseInputMode_RejectsRuleDir proves the rule-dir value is GONE, not merely
// unused (CLM-008): ParseInputMode("rule-dir") fail-louds as an unknown value
// after retirement, and the accepted-values list no longer advertises it.
func TestParseInputMode_RejectsRuleDir(t *testing.T) {
	_, err := ParseInputMode("rule-dir")
	if err == nil {
		t.Fatal("ParseInputMode(rule-dir) must error after the mode is retired, got nil")
	}
	if !strings.Contains(err.Error(), "rule-dir") {
		t.Errorf("fail-loud message must name the offending value, got: %v", err)
	}
	if strings.Contains(err.Error(), "rule-dir,") || strings.Contains(err.Error(), ", rule-dir") {
		t.Errorf("retired rule-dir must not appear in the accepted-values list, got: %v", err)
	}
}
