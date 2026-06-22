package engine

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

// TestGateType_SevenNeutralValuesNoCheckImport asserts the backstop-owned,
// tool-NEUTRAL gate-type enum has EXACTLY the seven values lint, build, test,
// findings, coverage, substantiveness, contracts (REQ-005/CLM-019) — and that
// the source file declaring it imports no pkg/check, keeping the leaf-package
// placement that mirrors EngineCategory/ScopeKind. The seven string spellings
// are the YAML surface the pack engines: block declares (the Phase-1 fixtures
// use exactly these), so a drift here desyncs the parser from the fixtures.
func TestGateType_SevenNeutralValuesNoCheckImport(t *testing.T) {
	// Each of the seven neutral values must round-trip through ParseGateType to a
	// DISTINCT GateType — proving the enum is exhaustive and the parser accepts
	// every declared value.
	want := []string{"lint", "build", "test", "findings", "coverage", "substantiveness", "contracts"}
	seen := map[GateType]string{}
	for _, s := range want {
		gt, err := ParseGateType(s)
		if err != nil {
			t.Fatalf("ParseGateType(%q): %v", s, err)
		}
		if prev, dup := seen[gt]; dup {
			t.Errorf("gate_type %q and %q resolved to the same GateType value %d — values must be distinct", s, prev, gt)
		}
		seen[gt] = s
		// The String() round-trips back to the canonical spelling so the binding's
		// gate-type can be re-serialized to the exact YAML the pack declared.
		if gt.String() != s {
			t.Errorf("GateType(%q).String() = %q, want %q", s, gt.String(), s)
		}
	}
	if len(seen) != 7 {
		t.Fatalf("GateType enum has %d distinct values, want exactly 7", len(seen))
	}

	// Leaf-package placement: gatetype.go must not IMPORT pkg/check (it would
	// reintroduce the cycle the leaf package exists to prevent). Parsed via the
	// import set — not a substring scan — so a doc comment mentioning pkg/check
	// does not false-fire while a real import is still caught (the transitive
	// proof lives in import_cycle_test.go).
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "gatetype.go", nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("parse gatetype.go: %v", err)
	}
	for _, imp := range file.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		if strings.Contains(path, "pkg/check") {
			t.Fatalf("gatetype.go must not import %q (leaf-package invariant)", path)
		}
	}
}

// TestManifest_UnknownGateTypeFailsLoud asserts ParseGateType fails loud on an
// unrecognized gate_type value — no silent default, parallel to ParseInputMode.
// CLM-021. Drives the value used by the Phase-1 bad-gatetype fixture.
func TestManifest_UnknownGateTypeFailsLoud(t *testing.T) {
	_, err := ParseGateType("teleport")
	if err == nil {
		t.Fatal("expected error for unknown gate_type, got nil")
	}
	if !strings.Contains(err.Error(), "teleport") {
		t.Errorf("error must name the offending value, got: %v", err)
	}
}
