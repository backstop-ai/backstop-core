package gate

import (
	"context"
	"go/parser"
	"go/token"
	"path/filepath"
	"testing"
)

// TestGate_ContractSignature_MatchingSignaturePasses verifies pass when a
// declared contract function exists with matching signature.
func TestGate_ContractSignature_MatchingSignaturePasses(t *testing.T) {
	target, err := filepath.Abs("testdata/contract-target.go")
	if err != nil {
		t.Fatal(err)
	}

	contracts := []ContractEntry{
		{File: target, Name: "DoSomething", Kind: "function", Signature: "func DoSomething(ctx context.Context, name string) error"},
	}
	step := StepContractSignatureFunc(contracts)
	result := step(context.Background())

	if result.StepName != StepContractSignature {
		t.Errorf("expected step_name %q, got %q", StepContractSignature, result.StepName)
	}
	if result.Status != "pass" {
		t.Errorf("expected status %q, got %q; violations: %v", "pass", result.Status, result.Violations)
	}
}

// TestGate_ContractSignature_MissingFunctionFails verifies fail when a
// declared contract function is missing from the file.
func TestGate_ContractSignature_MissingFunctionFails(t *testing.T) {
	target, err := filepath.Abs("testdata/contract-target.go")
	if err != nil {
		t.Fatal(err)
	}

	contracts := []ContractEntry{
		{File: target, Name: "NonExistentFunc", Kind: "function", Signature: "func NonExistentFunc() error"},
	}
	step := StepContractSignatureFunc(contracts)
	result := step(context.Background())

	if result.Status != "fail" {
		t.Errorf("expected status %q, got %q", "fail", result.Status)
	}
	if len(result.Violations) == 0 {
		t.Error("expected at least one violation for missing function")
	}
}

// TestGate_ContractSignature_WrongSignatureFails verifies fail when a
// declared function exists but signature differs.
func TestGate_ContractSignature_WrongSignatureFails(t *testing.T) {
	target, err := filepath.Abs("testdata/contract-target.go")
	if err != nil {
		t.Fatal(err)
	}

	// DoSomething exists but we declare a wrong signature
	contracts := []ContractEntry{
		{File: target, Name: "DoSomething", Kind: "function", Signature: "func DoSomething(name string) string"},
	}
	step := StepContractSignatureFunc(contracts)
	result := step(context.Background())

	if result.Status != "fail" {
		t.Errorf("expected status %q, got %q", "fail", result.Status)
	}
	if len(result.Violations) == 0 {
		t.Error("expected at least one violation for wrong signature")
	}
}

// TestGate_ContractSignature_TypeAndInterfaceVerified verifies that contract
// types and interfaces are verified, not just functions.
func TestGate_ContractSignature_TypeAndInterfaceVerified(t *testing.T) {
	target, err := filepath.Abs("testdata/contract-target.go")
	if err != nil {
		t.Fatal(err)
	}

	contracts := []ContractEntry{
		{File: target, Name: "Widget", Kind: "type", Signature: "type Widget struct"},
		{File: target, Name: "Runner", Kind: "interface", Signature: "type Runner interface"},
	}
	step := StepContractSignatureFunc(contracts)
	result := step(context.Background())

	if result.Status != "pass" {
		t.Errorf("expected status %q, got %q; violations: %v", "pass", result.Status, result.Violations)
	}
}

// TestGate_ContractSignature_VariableVerified verifies that contract variable
// declarations are verified by the contract signature step.
func TestGate_ContractSignature_VariableVerified(t *testing.T) {
	target, err := filepath.Abs("testdata/contract-target.go")
	if err != nil {
		t.Fatal(err)
	}

	contracts := []ContractEntry{
		{File: target, Name: "DefaultTimeout", Kind: "variable", Signature: "var DefaultTimeout int"},
	}
	step := StepContractSignatureFunc(contracts)
	result := step(context.Background())

	if result.Status != "pass" {
		t.Errorf("expected status %q, got %q; violations: %v", "pass", result.Status, result.Violations)
	}
}

// TestGate_FindVariable_Found verifies that findVariable locates a var
// declaration and returns its signature.
func TestGate_FindVariable_Found(t *testing.T) {
	target, err := filepath.Abs("testdata/contract-target.go")
	if err != nil {
		t.Fatal(err)
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, target, nil, 0)
	if err != nil {
		t.Fatalf("parsing file: %v", err)
	}

	sig, ok := findVariable(fset, f, "DefaultTimeout")
	if !ok {
		t.Fatal("expected to find variable DefaultTimeout")
	}
	if sig != "var DefaultTimeout int" {
		t.Errorf("expected signature %q, got %q", "var DefaultTimeout int", sig)
	}
}

// TestGate_FindVariable_NotFound verifies that findVariable returns false
// for a variable name that does not exist.
func TestGate_FindVariable_NotFound(t *testing.T) {
	target, err := filepath.Abs("testdata/contract-target.go")
	if err != nil {
		t.Fatal(err)
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, target, nil, 0)
	if err != nil {
		t.Fatalf("parsing file: %v", err)
	}

	_, ok := findVariable(fset, f, "NonExistentVar")
	if ok {
		t.Error("expected findVariable to return false for missing variable")
	}
}
