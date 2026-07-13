package gate

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// specWithStatusAndContract renders a minimal spec frontmatter whose status is
// the given value and which declares exactly one contract on a symbol that does
// not exist. Used by TestExtractContractEntries_OnlyImplementedSpecsAreExtracted
// to drive the full status vocabulary through ExtractContractEntries.
func specWithStatusAndContract(status string) string {
	return fmt.Sprintf(`---
title: "Scope Contract Spec"
number: SPEC-999
created: "2026-07-13"
status: %s
schema_version: spec/v1
spec_version: 1.0.0

implementation:
  summary: Scope contract test
  package: pkg/gate

verification:
  level: unit
  test_command: go test ./pkg/gate/... -race
  coverage_threshold: 80

requirements:
  - id: REQ-001
    text: Test req
    supports: cli:REQ-001

claims:
  - id: CLM-001
    requirement: REQ-001
    text: Test claim
    tests:
      - TestSomething

contracts:
  - file: pkg/gate/nonexistent.go
    provides:
      - name: SymbolThatDoesNotExist
        kind: function
        signature: "func SymbolThatDoesNotExist() error"
---

# Scope Contract Spec
`, status)
}

// TestExtractContractEntries_OnlyImplementedSpecsAreExtracted (CLM-002, CLM-003,
// CLM-004) proves that ExtractContractEntries extracts a spec's contracts ONLY
// when that spec is `implemented`. It is table-driven over the full status
// vocabulary. Each case stages a single spec declaring the SAME one contract on
// a symbol that does not exist:
//   - implemented                → 1 contract extracted (contracts are DUE).
//   - draft                      → 0 (pre-implementation; previously leaked a
//                                    false "symbol not found").
//   - ready-for-implementation   → 0 (same pre-implementation false pressure).
//   - replaced/canceled/deprecated → 0 (terminal, unbroken).
//   - obsoleted                  → 0 (terminal per schema; previously LEAKED
//                                    because isTerminalSpecStatus omits it —
//                                    now excluded via contractsAreDue).
func TestExtractContractEntries_OnlyImplementedSpecsAreExtracted(t *testing.T) {
	cases := []struct {
		status  string
		wantLen int
	}{
		{"implemented", 1},
		{"draft", 0},
		{"ready-for-implementation", 0},
		{"replaced", 0},
		{"canceled", 0},
		{"deprecated", 0},
		{"obsoleted", 0},
	}

	for _, tc := range cases {
		t.Run(tc.status, func(t *testing.T) {
			specDir := t.TempDir()
			content := specWithStatusAndContract(tc.status)
			if err := os.WriteFile(filepath.Join(specDir, "scope.spec.md"), []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}

			contracts, err := ExtractContractEntries(specDir, "/project/root")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(contracts) != tc.wantLen {
				t.Fatalf("status %q: expected %d contract entries, got %d: %v",
					tc.status, tc.wantLen, len(contracts), contracts)
			}
			if tc.wantLen == 1 {
				got := contracts[0]
				if got.Name != "SymbolThatDoesNotExist" {
					t.Errorf("status %q: expected contract on SymbolThatDoesNotExist, got %q",
						tc.status, got.Name)
				}
				wantFile := filepath.Join("/project/root", "pkg/gate/nonexistent.go")
				if got.File != wantFile {
					t.Errorf("status %q: expected file %q, got %q", tc.status, wantFile, got.File)
				}
			}
		})
	}
}

// TestContractsAreDue_TrueOnlyForImplemented (CLM-001) directly unit-tests the
// contractsAreDue predicate: it is true ONLY for "implemented" and false for
// every other status in the vocabulary (pre-implementation, terminal, obsoleted,
// and unknown/empty).
func TestContractsAreDue_TrueOnlyForImplemented(t *testing.T) {
	if !contractsAreDue("implemented") {
		t.Errorf("expected contractsAreDue(%q) == true", "implemented")
	}

	notDue := []string{
		"draft",
		"ready-for-implementation",
		"replaced",
		"canceled",
		"deprecated",
		"obsoleted",
		"unknown-status",
		"",
	}
	for _, status := range notDue {
		if contractsAreDue(status) {
			t.Errorf("expected contractsAreDue(%q) == false", status)
		}
	}
}
