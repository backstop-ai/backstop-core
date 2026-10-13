package engine

import (
	"reflect"
	"strings"
	"testing"
)

// success_exit_codes_test.go (PLAN-ISSUE-207 TASK-001). This file exercises NEW
// API (EffectiveSuccessExitCodes, ValidateSuccessExitCodes), so it is compile-red
// before TASK-005 by necessity — the one stated exception to the plan's
// behaviour-red discipline.

// TestEffectiveSuccessExitCodes_UndeclaredIsZeroOnly (CLM-005): an undeclared set
// resolves to exactly {0}; a declared set resolves to itself.
func TestEffectiveSuccessExitCodes_UndeclaredIsZeroOnly(t *testing.T) {
	undeclared := EngineBinding{Command: "tool"}
	if got := undeclared.EffectiveSuccessExitCodes(); !reflect.DeepEqual(got, []int{0}) {
		t.Errorf("undeclared success_exit_codes must resolve to exactly [0], got %v", got)
	}

	declared := EngineBinding{Command: "grep -rn", SuccessExitCodes: []int{0, 1}}
	if got := declared.EffectiveSuccessExitCodes(); !reflect.DeepEqual(got, []int{0, 1}) {
		t.Errorf("declared success_exit_codes [0, 1] must resolve to [0 1], got %v", got)
	}
}

// TestValidateSuccessExitCodes_RejectsInvalidSets (CLM-009, D-4): every invalid
// shape errors with a message naming the key and the reason; nil (undeclared) and
// a well-formed set pass.
func TestValidateSuccessExitCodes_RejectsInvalidSets(t *testing.T) {
	invalid := []struct {
		name   string
		codes  []int
		reason string
	}{
		{name: "empty non-nil", codes: []int{}, reason: "empty"},
		{name: "contains 256", codes: []int{0, 256}, reason: "256"},
		{name: "contains -1", codes: []int{0, -1}, reason: "-1"},
		{name: "duplicate 1", codes: []int{0, 1, 1}, reason: "duplicate"},
		{name: "missing 0", codes: []int{1}, reason: "must contain 0"},
	}
	for _, tc := range invalid {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateSuccessExitCodes(tc.codes)
			if err == nil {
				t.Fatalf("ValidateSuccessExitCodes(%v) must error", tc.codes)
			}
			if !strings.Contains(err.Error(), "success_exit_codes") {
				t.Errorf("error must name success_exit_codes, got %q", err)
			}
			if !strings.Contains(err.Error(), tc.reason) {
				t.Errorf("error must state the reason %q, got %q", tc.reason, err)
			}
		})
	}

	for _, ok := range [][]int{nil, {0, 1}, {0}, {0, 1, 2}} {
		if err := ValidateSuccessExitCodes(ok); err != nil {
			t.Errorf("ValidateSuccessExitCodes(%v) must pass, got %v", ok, err)
		}
	}
}
