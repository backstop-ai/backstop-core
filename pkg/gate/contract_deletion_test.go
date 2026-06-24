package gate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// contract_deletion_test.go (SPEC-038 TASK-021, REQ-001/010/011): pins the deletion
// surface. No go/parser symbol extraction remains; the rendering + string-equality
// helpers are gone; the non-scoped StepContractSignatureFunc is deleted; and no
// pkg/gate test references a deleted symbol (the package compiles).

// gateGoSources returns the non-test .go source in pkg/gate as one string for the
// import/symbol guards.
func gateGoSources(t *testing.T) map[string]string {
	t.Helper()
	out := map[string]string{}
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("reading pkg/gate: %v", err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		data, rerr := os.ReadFile(e.Name())
		if rerr != nil {
			t.Fatalf("reading %s: %v", e.Name(), rerr)
		}
		out[e.Name()] = string(data)
	}
	return out
}

// importBlock returns just the import-relevant lines (to avoid matching the names
// in comments/docstrings). We check for the import path string literals.
func hasImport(src, path string) bool {
	return strings.Contains(src, `"`+path+`"`)
}

// TestContract_NoGoParserExtractionRemains: pkg/gate imports no go/parser, go/ast,
// or go/printer for contract verification, and the probeSymbol/find* extractors are
// gone (CLM-001).
func TestContract_NoGoParserExtractionRemains(t *testing.T) {
	for name, src := range gateGoSources(t) {
		// The contract files specifically must not import the Go language packages.
		if name == "step_contract.go" || name == "contract_verdict.go" {
			for _, banned := range []string{"go/parser", "go/ast", "go/printer"} {
				if hasImport(src, banned) {
					t.Errorf("%s must not import %q for contracts (CLM-001)", name, banned)
				}
			}
		}
	}
	// The extractor functions must be undefined (a definition would read "func name(").
	src := gateGoSources(t)["step_contract.go"]
	for _, sym := range []string{"func probeSymbol", "func findFunction", "func findMethod", "func findType", "func findVariable"} {
		if strings.Contains(src, sym) {
			t.Errorf("deleted extractor still defined in step_contract.go: %q (CLM-001)", sym)
		}
	}
}

// TestContract_SignatureRenderingHelpersDeleted: the Go-source signature rendering
// helpers are deleted and unreferenced anywhere in pkg/gate (CLM-002).
func TestContract_SignatureRenderingHelpersDeleted(t *testing.T) {
	helpers := []string{"formatFuncSignature", "formatMethodSignature", "underlyingTypeString", "printFieldList"}
	for name, src := range gateGoSources(t) {
		for _, h := range helpers {
			// Allow the name to appear inside a // comment line (documentation of the
			// deletion); flag only a real Go token use (definition or call).
			for _, line := range strings.Split(src, "\n") {
				trimmed := strings.TrimSpace(line)
				if strings.HasPrefix(trimmed, "//") {
					continue
				}
				if strings.Contains(line, h) {
					t.Errorf("%s still references deleted rendering helper %q (CLM-002): %s", name, h, trimmed)
				}
			}
		}
	}
}

// TestContract_StringEqualityComparisonDeleted: signaturesMatch / normalizeSignature
// are deleted; satisfaction is no longer decided by string equality (CLM-003).
func TestContract_StringEqualityComparisonDeleted(t *testing.T) {
	for name, src := range gateGoSources(t) {
		for _, sym := range []string{"signaturesMatch", "normalizeSignature"} {
			for _, line := range strings.Split(src, "\n") {
				trimmed := strings.TrimSpace(line)
				if strings.HasPrefix(trimmed, "//") {
					continue
				}
				if strings.Contains(line, sym) {
					t.Errorf("%s still references deleted compare helper %q (CLM-003): %s", name, sym, trimmed)
				}
			}
		}
	}
}

// TestContract_NoDeletedSymbolReferencedInGateTests: no pkg/gate *_test.go references
// any deleted analyzer symbol, and the package compiles (CLM-035). (That this test
// file compiles and runs is itself part of the proof.)
func TestContract_NoDeletedSymbolReferencedInGateTests(t *testing.T) {
	deleted := []string{
		"StepContractSignatureFunc(", "probeSymbol", "findFunction", "findMethod",
		"findType(", "findVariable", "formatFuncSignature", "formatMethodSignature",
		"underlyingTypeString", "printFieldList", "signaturesMatch", "normalizeSignature",
	}
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("reading pkg/gate: %v", err)
	}
	// The deletion/equivalence guard files name the deleted symbols as STRING DATA
	// (to assert their absence / document the proven verdicts) — that is not a Go
	// reference to the symbol. Exclude those self-referential guards; everywhere
	// else, a deleted-symbol token used OUTSIDE a string literal would be a real
	// compile-coupling reference (and the package would not compile).
	selfReferential := map[string]bool{
		"contract_deletion_test.go":         true,
		"contract_equivalence_test.go":      true,
		"contract_verdict_test.go":          true,
		"contract_migrated_absence_test.go": true,
	}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), "_test.go") || selfReferential[e.Name()] {
			continue
		}
		data, rerr := os.ReadFile(e.Name())
		if rerr != nil {
			t.Fatalf("reading %s: %v", e.Name(), rerr)
		}
		for _, line := range strings.Split(string(data), "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "//") {
				continue
			}
			for _, sym := range deleted {
				// Only an UNQUOTED occurrence is a real Go reference; a quoted token
				// is string data, not a symbol use.
				if strings.Contains(line, sym) && !inStringLiteralOnly(line, sym) {
					t.Errorf("%s references deleted symbol %q (CLM-035): %s", e.Name(), sym, trimmed)
				}
			}
		}
	}
}

// inStringLiteralOnly reports whether every occurrence of sym in line sits inside a
// double-quoted string literal (string DATA, not a Go identifier reference).
func inStringLiteralOnly(line, sym string) bool {
	idx := 0
	for {
		pos := strings.Index(line[idx:], sym)
		if pos < 0 {
			return true
		}
		abs := idx + pos
		// Count unescaped quotes before this position; odd => inside a string.
		quotes := strings.Count(line[:abs], `"`)
		if quotes%2 == 0 {
			return false // this occurrence is NOT inside a string literal
		}
		idx = abs + len(sym)
	}
}

// TestContract_DeletedInternalOnlyTestsRemoved: the analyzer-internal-only test files
// (Go-source rendering, string normalization, the dissolved non-.go-error) are
// removed — they do not survive as dead/skipped tests referencing removed symbols
// (CLM-037).
func TestContract_DeletedInternalOnlyTestsRemoved(t *testing.T) {
	removed := []string{
		"step_contract_test.go",
		"step_contract_absence_test.go",
		"step_contract_absence_config_test.go",
		"step_contract_noregress_test.go",
	}
	for _, f := range removed {
		if _, err := os.Stat(f); err == nil {
			t.Errorf("analyzer-coupled test file %s must be deleted in the same change as the deletion (CLM-037)", f)
		}
	}
	// The surviving file must remain (its surviving Absent-field extraction coverage
	// is preserved — not dropped).
	survivor := "step_contract_parser_absence_test.go"
	if _, err := os.Stat(survivor); err != nil {
		t.Errorf("the surviving extractor test %s must be PRESERVED (CLM-036): %v", survivor, err)
	}
}

// TestContract_NonScopedEntrypointDeleted: the non-scoped StepContractSignatureFunc
// is deleted and StepContractSignatureScopedFunc is the sole entrypoint (CLM-038).
func TestContract_NonScopedEntrypointDeleted(t *testing.T) {
	src := gateGoSources(t)["step_contract.go"]
	for _, line := range strings.Split(src, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") {
			continue
		}
		if strings.Contains(line, "func StepContractSignatureFunc(") {
			t.Errorf("non-scoped StepContractSignatureFunc must be DELETED (CLM-038): %s", trimmed)
		}
	}
	if !strings.Contains(src, "func StepContractSignatureScopedFunc(") {
		t.Error("StepContractSignatureScopedFunc must remain as the sole contract entrypoint (CLM-038)")
	}
}

// TestContract_NonScopedEntrypointHasNoCaller: no non-test caller of the deleted
// wrapper remains anywhere in the module (CLM-039).
func TestContract_NonScopedEntrypointHasNoCaller(t *testing.T) {
	// Walk up to module root and grep production .go files for the deleted call.
	root := eqRepoRoot(t)
	var offenders []string
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		data, rerr := os.ReadFile(path)
		if rerr != nil {
			return nil
		}
		for _, line := range strings.Split(string(data), "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "//") {
				continue
			}
			if strings.Contains(line, "StepContractSignatureFunc(") {
				offenders = append(offenders, path+": "+trimmed)
			}
		}
		return nil
	})
	if len(offenders) > 0 {
		t.Errorf("no non-test caller of the deleted StepContractSignatureFunc may remain (CLM-039): %v", offenders)
	}
}
