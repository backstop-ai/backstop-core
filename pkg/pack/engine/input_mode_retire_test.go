package engine

import (
	"strings"
	"testing"
)

// TestInputModeRuleDir_Retired pins the end state of the ISSUE-028 mode
// retirement (CLM-008): NO EngineBinding in DefaultRegistry declares the
// retired "rule-dir" input mode. ast-grep was the sole user; after its flip to
// config-file the mode has zero remaining users and is deleted. This re-reds if
// a future binding reintroduces the dead mode.
func TestInputModeRuleDir_Retired(t *testing.T) {
	for name, binding := range DefaultRegistry() {
		if binding.InputMode == InputMode("rule-dir") {
			t.Errorf("engine %q still declares the retired rule-dir input mode; "+
				"ast-grep was its sole user and now uses config-file (ISSUE-028)", name)
		}
	}
}

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
