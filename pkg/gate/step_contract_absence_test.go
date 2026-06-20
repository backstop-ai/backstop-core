package gate

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

// TestGate_ContractSignature_AbsentSymbolPasses (CLM-001) verifies that an
// {Absent:true} entry naming an identifier that is NOT present in the named file
// passes with zero violations — the asserted deletion genuinely held.
func TestGate_ContractSignature_AbsentSymbolPasses(t *testing.T) {
	clean, err := filepath.Abs("testdata/contract-absent-clean.go")
	if err != nil {
		t.Fatal(err)
	}

	contracts := []ContractEntry{
		{File: clean, Name: "LintExecutor", Kind: "function", Absent: true},
		{File: clean, Name: "BespokeExecutor", Kind: "type", Absent: true},
		{File: clean, Name: "GoBuiltinExecutors", Kind: "variable", Absent: true},
		{File: clean, Name: "(*BespokeExecutor).Probe", Kind: "method", Absent: true},
	}
	result := StepContractSignatureFunc(contracts)(context.Background())

	if result.StepName != StepContractSignature {
		t.Errorf("expected step_name %q, got %q", StepContractSignature, result.StepName)
	}
	if result.Status != "pass" {
		t.Errorf("expected status %q, got %q; violations: %v", "pass", result.Status, result.Violations)
	}
	if len(result.Violations) != 0 {
		t.Errorf("expected zero violations for genuinely-absent symbols, got %v", result.Violations)
	}
}

// TestGate_ContractSignature_ReappearedSymbolFails (CLM-002) verifies that an
// {Absent:true} entry naming an identifier that IS present in the named file
// fails, with a violation that states the symbol was expected absent (regression
// caught) and Rule == "contract_signature".
func TestGate_ContractSignature_ReappearedSymbolFails(t *testing.T) {
	present, err := filepath.Abs("testdata/contract-absent-present.go")
	if err != nil {
		t.Fatal(err)
	}

	contracts := []ContractEntry{
		{File: present, Name: "LintExecutor", Kind: "function", Absent: true},
	}
	result := StepContractSignatureFunc(contracts)(context.Background())

	if result.Status != "fail" {
		t.Fatalf("expected status %q for reappeared symbol, got %q", "fail", result.Status)
	}
	var matched bool
	for _, v := range result.Violations {
		if v.Rule != "contract_signature" {
			t.Errorf("expected rule %q, got %q", "contract_signature", v.Rule)
		}
		if strings.Contains(v.Message, "LintExecutor") && strings.Contains(strings.ToLower(v.Message), "absent") {
			matched = true
		}
	}
	if !matched {
		t.Errorf("expected a violation naming LintExecutor and stating it was expected absent, got %v", result.Violations)
	}
}

// TestGate_ContractSignature_AbsentScopedToNamedFile (CLM-005) verifies that an
// {Absent:true} entry is scoped to the NAMED file only: a symbol that exists in
// contract-absent-present.go but is asserted absent against contract-absent-clean.go
// passes, because the symbol is genuinely not in the named file.
func TestGate_ContractSignature_AbsentScopedToNamedFile(t *testing.T) {
	clean, err := filepath.Abs("testdata/contract-absent-clean.go")
	if err != nil {
		t.Fatal(err)
	}

	// LintExecutor exists in contract-absent-present.go, but we assert it absent
	// from contract-absent-clean.go where it genuinely is not declared.
	contracts := []ContractEntry{
		{File: clean, Name: "LintExecutor", Kind: "function", Absent: true},
	}
	result := StepContractSignatureFunc(contracts)(context.Background())

	if result.Status != "pass" {
		t.Errorf("expected status %q (present-elsewhere does not fail file-scoped absence), got %q; violations: %v",
			"pass", result.Status, result.Violations)
	}
	if len(result.Violations) != 0 {
		t.Errorf("expected zero violations, got %v", result.Violations)
	}
}
