package gate

import (
	"os"
	"path/filepath"
	"testing"
)

// TestGate_ExtractContractEntries_AbsentFieldParsed (CLM-008) verifies that the
// new spec-contract `absent: true` field is surfaced onto ContractEntry.Absent,
// while a sibling provide entry without the field yields Absent==false. Asserts
// the parsed Absent boolean per entry (and its File/Name/Kind), not just len().
func TestGate_ExtractContractEntries_AbsentFieldParsed(t *testing.T) {
	specDir := t.TempDir()

	content := `---
title: "Absence Contract Spec"
number: SPEC-013
created: "2026-06-20"
status: implemented
schema_version: spec/v1
spec_version: 1.0.0

implementation:
  summary: Absence contract test
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
  - file: pkg/check/check.go
    provides:
      - name: lintExecutor
        kind: function
        absent: true
      - name: New
        kind: function
        signature: "func New(opts ...Option) *Gate"
---

# Absence Contract Spec
`
	if err := os.WriteFile(filepath.Join(specDir, "absence.spec.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	contracts, err := ExtractContractEntries(specDir, "/project/root")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(contracts) != 2 {
		t.Fatalf("expected 2 contract entries, got %d: %v", len(contracts), contracts)
	}

	byName := map[string]ContractEntry{}
	for _, c := range contracts {
		byName[c.Name] = c
	}

	absent, ok := byName["lintExecutor"]
	if !ok {
		t.Fatalf("expected an entry named lintExecutor, got %v", contracts)
	}
	if !absent.Absent {
		t.Errorf("expected lintExecutor entry Absent==true, got false")
	}
	if absent.Kind != "function" {
		t.Errorf("expected lintExecutor kind %q, got %q", "function", absent.Kind)
	}
	expectedPath := filepath.Join("/project/root", "pkg/check/check.go")
	if absent.File != expectedPath {
		t.Errorf("expected lintExecutor file %q, got %q", expectedPath, absent.File)
	}

	present, ok := byName["New"]
	if !ok {
		t.Fatalf("expected an entry named New, got %v", contracts)
	}
	if present.Absent {
		t.Errorf("expected New entry Absent==false (field omitted), got true")
	}
	if present.Signature != "func New(opts ...Option) *Gate" {
		t.Errorf("expected New signature preserved, got %q", present.Signature)
	}
}

// TestGate_ExtractContractEntries_AbsentDefaultsFalse (CLM-009) verifies that a
// spec with only conventional present contracts yields entries all with
// Absent==false — the new field is a backward-compatible default and existing
// specs are unchanged.
func TestGate_ExtractContractEntries_AbsentDefaultsFalse(t *testing.T) {
	specDir := t.TempDir()

	content := `---
title: "Present-Only Contract Spec"
number: SPEC-001
created: "2026-01-01"
status: implemented
schema_version: spec/v1
spec_version: 1.0.0

implementation:
  summary: Present contract test
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
  - file: pkg/gate/gate.go
    provides:
      - name: New
        kind: function
        signature: "func New(opts ...Option) *Gate"
      - name: Widget
        kind: type
        signature: "type Widget struct"
---

# Present-Only Contract Spec
`
	if err := os.WriteFile(filepath.Join(specDir, "present.spec.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	contracts, err := ExtractContractEntries(specDir, "/project/root")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(contracts) != 2 {
		t.Fatalf("expected 2 contract entries, got %d", len(contracts))
	}
	for _, c := range contracts {
		if c.Absent {
			t.Errorf("expected entry %q to default Absent==false, got true", c.Name)
		}
	}
}
