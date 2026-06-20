package gate

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

// TestGate_ContractSignature_AbsentMissingFileIsConfigError (CLM-003) verifies
// that an {Absent:true} entry whose File does not exist on disk is a LOUD config
// error (NOT silently skipped the way a present-only entry on a missing file is).
// It also asserts the mixed-entry split: a present-only entry on the SAME missing
// file still skips with no violation, proving the skip change is absence-scoped.
func TestGate_ContractSignature_AbsentMissingFileIsConfigError(t *testing.T) {
	missing, err := filepath.Abs("testdata/contract-absent-does-not-exist.go")
	if err != nil {
		t.Fatal(err)
	}

	// Absence assertion on a missing file → loud config error.
	absentResult := StepContractSignatureFunc([]ContractEntry{
		{File: missing, Name: "LintExecutor", Kind: "function", Absent: true},
	})(context.Background())

	if absentResult.Status != "fail" {
		t.Fatalf("expected status %q for absence assertion on missing file, got %q",
			"fail", absentResult.Status)
	}
	var loud bool
	for _, v := range absentResult.Violations {
		if v.Rule != "contract_signature" {
			t.Errorf("expected rule %q, got %q", "contract_signature", v.Rule)
		}
		if strings.Contains(v.Message, missing) && strings.Contains(strings.ToLower(v.Message), "absence") {
			loud = true
		}
	}
	if !loud {
		t.Errorf("expected a config-error violation naming the missing file and flagging an absence assertion, got %v",
			absentResult.Violations)
	}

	// Present-only entry on the SAME missing file still skips (no violation) —
	// proving the missing-file skip change is absence-scoped, not global.
	presentResult := StepContractSignatureFunc([]ContractEntry{
		{File: missing, Name: "LintExecutor", Kind: "function", Signature: "func LintExecutor(ctx context.Context) error"},
	})(context.Background())

	if presentResult.Status != "pass" {
		t.Errorf("expected present-only entry on missing file to still skip (status %q), got %q; violations: %v",
			"pass", presentResult.Status, presentResult.Violations)
	}
	if len(presentResult.Violations) != 0 {
		t.Errorf("expected zero violations for skipped present-only entry on missing file, got %v",
			presentResult.Violations)
	}
}

// TestGate_ContractSignature_AbsentNonIdentifierIsConfigError (CLM-004) verifies
// that an {Absent:true} entry whose Name is a placeholder slug (containing '-')
// is a LOUD config error requiring a real Go identifier — the slug must NOT
// silently pass.
func TestGate_ContractSignature_AbsentNonIdentifierIsConfigError(t *testing.T) {
	clean, err := filepath.Abs("testdata/contract-absent-clean.go")
	if err != nil {
		t.Fatal(err)
	}

	result := StepContractSignatureFunc([]ContractEntry{
		{File: clean, Name: "bespoke-toolchain-tests", Kind: "function", Absent: true},
	})(context.Background())

	if result.Status != "fail" {
		t.Fatalf("expected status %q for non-identifier absence name, got %q",
			"fail", result.Status)
	}
	var loud bool
	for _, v := range result.Violations {
		if v.Rule != "contract_signature" {
			t.Errorf("expected rule %q, got %q", "contract_signature", v.Rule)
		}
		if strings.Contains(v.Message, "bespoke-toolchain-tests") &&
			strings.Contains(strings.ToLower(v.Message), "identifier") {
			loud = true
		}
	}
	if !loud {
		t.Errorf("expected a config-error violation naming the slug and requiring a real Go identifier, got %v",
			result.Violations)
	}
}

// TestGate_ContractSignature_AbsentUnknownKindIsError verifies the semantics-table
// row for an {Absent:true} entry with an unrecognized kind: it is reported as an
// unknown-kind error (the same provenance as the present path), not a silent pass.
func TestGate_ContractSignature_AbsentUnknownKindIsError(t *testing.T) {
	clean, err := filepath.Abs("testdata/contract-absent-clean.go")
	if err != nil {
		t.Fatal(err)
	}

	result := StepContractSignatureFunc([]ContractEntry{
		{File: clean, Name: "LintExecutor", Kind: "gremlin", Absent: true},
	})(context.Background())

	if result.Status != "fail" {
		t.Fatalf("expected status %q for unknown-kind absence entry, got %q", "fail", result.Status)
	}
	var loud bool
	for _, v := range result.Violations {
		if v.Rule == "contract_signature" &&
			strings.Contains(v.Message, "gremlin") &&
			strings.Contains(strings.ToLower(v.Message), "unknown contract kind") {
			loud = true
		}
	}
	if !loud {
		t.Errorf("expected an unknown-kind violation naming the bad kind, got %v", result.Violations)
	}
}

// TestGate_ContractSignature_AbsentNonGoFileIsConfigError (CLM-007) verifies that
// an {Absent:true} entry whose File is a non-.go path is a LOUD config error
// (cannot AST-probe), distinct from the present-path's documentation skip of
// non-Go files.
func TestGate_ContractSignature_AbsentNonGoFileIsConfigError(t *testing.T) {
	result := StepContractSignatureFunc([]ContractEntry{
		{File: "docs/some-schema.json", Name: "LintExecutor", Kind: "function", Absent: true},
	})(context.Background())

	if result.Status != "fail" {
		t.Fatalf("expected status %q for absence assertion on non-Go file, got %q",
			"fail", result.Status)
	}
	var loud bool
	for _, v := range result.Violations {
		if v.Rule != "contract_signature" {
			t.Errorf("expected rule %q, got %q", "contract_signature", v.Rule)
		}
		if strings.Contains(v.Message, "docs/some-schema.json") &&
			strings.Contains(strings.ToLower(v.Message), "non-go") {
			loud = true
		}
	}
	if !loud {
		t.Errorf("expected a config-error violation naming the non-Go file, got %v", result.Violations)
	}

	// Sibling check: a present-only entry on the same non-Go file still skips,
	// proving the non-Go skip change is absence-scoped.
	presentResult := StepContractSignatureFunc([]ContractEntry{
		{File: "docs/some-schema.json", Name: "LintExecutor", Kind: "function", Signature: "x"},
	})(context.Background())
	if presentResult.Status != "pass" || len(presentResult.Violations) != 0 {
		t.Errorf("expected present-only entry on non-Go file to still skip, got status=%q violations=%v",
			presentResult.Status, presentResult.Violations)
	}
}
