package gate

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

// TestGate_ContractSignature_PresentPathUnchangedByAbsence (CLM-006) fences the
// present path against the absence branch. With Absent==false (the default), an
// entry must retain its exact pre-absence behavior:
//   - a matching present symbol passes;
//   - a missing present-only symbol still reports "symbol not found";
//   - a present-only entry on a missing file is still silently skipped;
//   - a present-only entry on a non-Go file is still silently skipped.
//
// It drives the existing contract-target.go fixture plus the new absence
// fixtures, guarding against the absence branch leaking into the present path.
func TestGate_ContractSignature_PresentPathUnchangedByAbsence(t *testing.T) {
	target, err := filepath.Abs("testdata/contract-target.go")
	if err != nil {
		t.Fatal(err)
	}
	present, err := filepath.Abs("testdata/contract-absent-present.go")
	if err != nil {
		t.Fatal(err)
	}

	// 1. Matching present symbol → pass (Absent defaults false).
	pass := StepContractSignatureFunc([]ContractEntry{
		{File: target, Name: "DoSomething", Kind: "function", Signature: "func DoSomething(ctx context.Context, name string) error"},
	})(context.Background())
	if pass.Status != "pass" || len(pass.Violations) != 0 {
		t.Fatalf("matching present symbol: expected pass with no violations, got status=%q violations=%v",
			pass.Status, pass.Violations)
	}

	// 2. Missing present-only symbol → "symbol not found".
	notFound := StepContractSignatureFunc([]ContractEntry{
		{File: present, Name: "NeverDeclaredSymbol", Kind: "function", Signature: "func NeverDeclaredSymbol()"},
	})(context.Background())
	if notFound.Status != "fail" {
		t.Fatalf("missing present-only symbol: expected fail, got %q", notFound.Status)
	}
	var notFoundMsg bool
	for _, v := range notFound.Violations {
		if strings.Contains(v.Message, "NeverDeclaredSymbol") && strings.Contains(v.Message, "not found") {
			notFoundMsg = true
		}
	}
	if !notFoundMsg {
		t.Errorf("missing present-only symbol: expected a \"symbol not found\" violation, got %v", notFound.Violations)
	}

	// 3. Present-only entry on a missing file → silently skipped (pass, no violation).
	missing := filepath.Join(t.TempDir(), "gone.go")
	skipMissing := StepContractSignatureFunc([]ContractEntry{
		{File: missing, Name: "Whatever", Kind: "function", Signature: "func Whatever()"},
	})(context.Background())
	if skipMissing.Status != "pass" || len(skipMissing.Violations) != 0 {
		t.Errorf("present-only on missing file: expected silent skip, got status=%q violations=%v",
			skipMissing.Status, skipMissing.Violations)
	}

	// 4. Present-only entry on a non-Go file → silently skipped (pass, no violation).
	skipNonGo := StepContractSignatureFunc([]ContractEntry{
		{File: "config/schema.json", Name: "Whatever", Kind: "function", Signature: "x"},
	})(context.Background())
	if skipNonGo.Status != "pass" || len(skipNonGo.Violations) != 0 {
		t.Errorf("present-only on non-Go file: expected silent skip, got status=%q violations=%v",
			skipNonGo.Status, skipNonGo.Violations)
	}
}
