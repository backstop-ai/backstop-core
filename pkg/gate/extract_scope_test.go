package gate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// extract_scope_test.go (SPEC-038 TASK-009, REQ-012): ExtractContractEntries is
// EXTENDED to populate ContractEntry.Scope from the declared provides entry (the
// file-OR-path absence parameter), so a path-scoped absence probe receives its
// scope through pattern-arg. Extraction stays a PURE data-record builder — no
// parse, no AST-walk, no signature compilation (CLM-042).

// writeSpec writes a .spec.md file with the given frontmatter body into specDir.
func writeSpec(t *testing.T, specDir, name, frontmatter string) {
	t.Helper()
	content := "---\n" + frontmatter + "\n---\n\n# fixture spec\n"
	if err := os.WriteFile(filepath.Join(specDir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("writing spec fixture: %v", err)
	}
}

// findEntry returns the first ContractEntry with the given Name.
func findEntry(entries []ContractEntry, name string) (ContractEntry, bool) {
	for _, e := range entries {
		if e.Name == name {
			return e, true
		}
	}
	return ContractEntry{}, false
}

// TestExtract_ContractEntryScopePopulatedFromDeclaration: a spec declaring a
// path-scoped absence yields a ContractEntry whose Scope equals the declared path
// (CLM-040).
func TestExtract_ContractEntryScopePopulatedFromDeclaration(t *testing.T) {
	specDir := t.TempDir()
	writeSpec(t, specDir, "absence-path.spec.md", strings.Join([]string{
		"number: SPEC-FIX",
		"status: implemented",
		"contracts:",
		"  - file: pkg/gate/testdata/contract-absence-present.go",
		"    provides:",
		"      - name: legacyProbeSymbol",
		"        kind: function",
		"        absent: true",
		"        scope: pkg/gate/testdata",
	}, "\n"))

	entries, err := ExtractContractEntries(specDir, "/project/root")
	if err != nil {
		t.Fatalf("ExtractContractEntries: %v", err)
	}
	e, ok := findEntry(entries, "legacyProbeSymbol")
	if !ok {
		t.Fatal("expected a ContractEntry for legacyProbeSymbol")
	}
	if !e.Absent {
		t.Error("entry must carry Absent=true")
	}
	// Scope is project-relative-joined like File; assert it ends with the declared path.
	if !strings.HasSuffix(filepath.ToSlash(e.Scope), "pkg/gate/testdata") {
		t.Errorf("Scope = %q, want it to carry the declared path pkg/gate/testdata (CLM-040)", e.Scope)
	}
}

// TestExtract_PathScopedAbsenceScopeReachesGrepProbe: a path-scoped absence
// ContractEntry carries its declared scope intact at the extraction boundary so
// it can flow to the grep pattern-arg (CLM-041) — the value is present and
// non-empty on the record, making the path-scoped probe buildable end to end.
func TestExtract_PathScopedAbsenceScopeReachesGrepProbe(t *testing.T) {
	specDir := t.TempDir()
	writeSpec(t, specDir, "absence-path.spec.md", strings.Join([]string{
		"number: SPEC-FIX",
		"status: implemented",
		"contracts:",
		"  - file: pkg/gate/testdata/contract-absence-present.go",
		"    provides:",
		"      - name: forbiddenSym",
		"        kind: function",
		"        absent: true",
		"        scope: pkg/some/dir",
	}, "\n"))

	entries, err := ExtractContractEntries(specDir, "/project/root")
	if err != nil {
		t.Fatalf("ExtractContractEntries: %v", err)
	}
	e, ok := findEntry(entries, "forbiddenSym")
	if !ok {
		t.Fatal("expected a ContractEntry for forbiddenSym")
	}
	if e.Scope == "" {
		t.Fatal("path-scoped absence ContractEntry must carry a non-empty Scope to reach the grep probe (CLM-041)")
	}
	if !strings.Contains(filepath.ToSlash(e.Scope), "pkg/some/dir") {
		t.Errorf("Scope = %q must carry the declared path so it flows to pattern-arg (CLM-041)", e.Scope)
	}
}

// TestExtract_ContractEntryExtractionDoesNotParseOrCompile: extraction is a pure
// data-record builder — a contract with a Signature is carried through UNMODIFIED
// (not rendered/compiled), with no parse/AST-walk (CLM-042).
func TestExtract_ContractEntryExtractionDoesNotParseOrCompile(t *testing.T) {
	specDir := t.TempDir()
	declaredSig := "func RouteFile(path string, mode int) (string, error)"
	writeSpec(t, specDir, "signature.spec.md", strings.Join([]string{
		"number: SPEC-FIX",
		"status: implemented",
		"contracts:",
		"  - file: pkg/gate/testdata/contract-sig-present.go",
		"    provides:",
		"      - name: RouteFile",
		"        kind: function",
		"        signature: " + `"` + declaredSig + `"`,
	}, "\n"))

	entries, err := ExtractContractEntries(specDir, "/project/root")
	if err != nil {
		t.Fatalf("ExtractContractEntries: %v", err)
	}
	e, ok := findEntry(entries, "RouteFile")
	if !ok {
		t.Fatal("expected a ContractEntry for RouteFile")
	}
	// The Signature must be carried through verbatim — extraction must NOT compile,
	// render, or normalize it.
	if e.Signature != declaredSig {
		t.Errorf("Signature must be carried through unmodified, got %q want %q (CLM-042)", e.Signature, declaredSig)
	}
	// A present (non-absent) signature contract carries no scope requirement.
	if e.Absent {
		t.Error("a signature contract must not be marked Absent")
	}

	// Structural guard: step_testverify.go's extraction must not gain go/ast/parser
	// usage in ExtractContractEntries (it stays a pure frontmatter reader).
	src, rerr := os.ReadFile(filepath.Join(".", "step_testverify.go"))
	if rerr != nil {
		t.Fatalf("reading step_testverify.go: %v", rerr)
	}
	if strings.Contains(string(src), "parser.ParseFile") {
		t.Error("ExtractContractEntries' file must not parse source (CLM-042) — no parser.ParseFile")
	}
}
