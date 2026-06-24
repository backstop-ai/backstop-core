package gate

import "testing"

// contract_migrated_absence_test.go (SPEC-038 TASK-021, REQ-010/CLM-036): the
// SURVIVING absence behavior previously covered by step_contract_absence_test.go /
// step_contract_absence_config_test.go (present->violation, absent->pass,
// missing/unscanned scope->loud config error) is MIGRATED to and still covered
// against the pack-SARIF consumer (VerifyContractVerdict / ContractEngineResult) —
// not dropped. This replaces the deleted analyzer-coupled coverage.

// TestContract_MigratedAbsenceBehaviorStillCovered exercises the three surviving
// absence dispositions against the new gate-side consumer.
func TestContract_MigratedAbsenceBehaviorStillCovered(t *testing.T) {
	entry := ContractEntry{File: "x.go", Name: "Forbidden", Absent: true, Scope: "x.go"}

	// present -> violation (forbidden symbol matched in a scanned scope).
	if _, raised := VerifyContractVerdict(ContractEngineResult{Entry: entry, Matched: true, Scanned: true}); !raised {
		t.Error("migrated absence: a present forbidden symbol must raise a violation")
	}

	// absent -> pass (genuinely absent from a confirmed-scanned scope).
	if _, raised := VerifyContractVerdict(ContractEngineResult{Entry: entry, Matched: false, Scanned: true}); raised {
		t.Error("migrated absence: a genuinely absent symbol in a scanned scope must PASS")
	}

	// missing/unscanned scope -> loud config error (never a silent pass).
	v, raised := VerifyContractVerdict(ContractEngineResult{Entry: entry, Matched: false, Scanned: false})
	if !raised {
		t.Fatal("migrated absence: an unscanned/missing scope must raise a loud config error, not pass")
	}
	if v.Severity != "error" {
		t.Errorf("migrated absence: config error must be severity error, got %q", v.Severity)
	}

	// And the present-signature surviving behavior: present->satisfied, mismatch->violation.
	sig := ContractEntry{File: "y.go", Name: "F", Signature: "func F()"}
	if _, raised := VerifyContractVerdict(ContractEngineResult{Entry: sig, Matched: true, Scanned: true}); raised {
		t.Error("migrated signature: a present signature (ast-grep match) must be SATISFIED")
	}
	if _, raised := VerifyContractVerdict(ContractEngineResult{Entry: sig, Matched: false, Scanned: true}); !raised {
		t.Error("migrated signature: an absent/mismatched signature must raise a violation")
	}
}
