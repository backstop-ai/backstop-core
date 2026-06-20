package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/pack/engine"
)

// SPEC-034 phase-3 DELETION guard + end-state invariants (REQ-009/REQ-011).
// These assert the strangler completed WITHOUT an enforcement-lapse window: the
// bespoke Go toolchain symbols are gone from pkg/check non-test source, AND the
// engine path that replaced them is present and dispatching. They are red while
// the bespoke code still exists and go green only after the phase-3 deletions.

// bespokeNonTestSymbols are the baked-in Go-toolchain identifiers that REQ-011
// requires absent from pkg/check non-test source after the cutover.
var bespokeNonTestSymbols = []string{
	"goBuiltinExecutors",
	"lintExecutor",
	"buildExecutor",
	"testExecutor",
	"parseGoBuildErrors",
	"parseGoTestFailures",
	"parseGolangciJSON",
	"golangciOutputArgs",
	"golangciMajorVersion",
	"golangciVersionRe",
	"goBuildErrorRe",
	"goTestFailRe",
	"goTestPosRe",
}

// pkgCheckNonTestSources returns the absolute paths of every non-test .go file in
// pkg/check, for the end-state source scans.
func pkgCheckNonTestSources(t *testing.T) []string {
	t.Helper()
	dir := filepath.Join(repoRoot(t), "pkg", "check")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading pkg/check: %v", err)
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		out = append(out, filepath.Join(dir, e.Name()))
	}
	if len(out) == 0 {
		t.Fatal("expected to scan pkg/check non-test sources")
	}
	return out
}

// TestStrangler_DeletionGatedOnProvenEquivalence is the deletion-gate guard
// (CLM-033). N2 ONE-TREE invariant: in this working tree the bespoke symbols are
// ABSENT (the deletion completed) AND the engine path that replaced them is
// present-and-enforcing — the go-toolchain build/test/lint engine bindings are
// registered and dispatchable. So bespoke-absent implies engine-present: a
// deletion that outran the replacement (an enforcement-lapse window, Sharp Edge
// 2) fails this guard rather than shipping green. The equivalence PROOF that
// licensed the deletion is the committed phase-4 history (a1b5c69); it is
// transitional and retires with the bespoke parsers it compared against.
func TestStrangler_DeletionGatedOnProvenEquivalence(t *testing.T) {
	// (1) Bespoke absent from pkg/check non-test source.
	for _, path := range pkgCheckNonTestSources(t) {
		src, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading %s: %v", path, err)
		}
		for _, sym := range bespokeNonTestSymbols {
			if containsIdent(string(src), sym) {
				t.Errorf("%s still references bespoke symbol %q; deletion is not complete", filepath.Base(path), sym)
			}
		}
	}

	// (2) Engine path present-and-enforcing: the three native Go toolchain engine
	// bindings replaced the bespoke executors and are registered + dispatchable.
	// Absence here while the bespoke is gone would be the enforcement-lapse window.
	reg := engine.DefaultRegistry()
	for _, eng := range []string{"go-build", "go-test", "golangci"} {
		b, err := reg.Lookup(eng)
		if err != nil {
			t.Fatalf("engine %q must remain registered (enforcement must not lapse when bespoke is deleted): %v", eng, err)
		}
		if !strings.Contains(b.Command, "go ") && !strings.HasPrefix(b.Command, "golangci-lint") {
			t.Errorf("engine %q binding looks wrong: %q", eng, b.Command)
		}
	}
}

// TestEndState_NoBakedGoToolchainKnowledge is the binary-wide backstop grep
// (CLM-036): pkg/check non-test source contains none of the bespoke executors,
// parsers, version-adaptive flag logic, named formats, or the `language == "go"`
// short-circuit.
func TestEndState_NoBakedGoToolchainKnowledge(t *testing.T) {
	sources := pkgCheckNonTestSources(t)
	for _, path := range sources {
		src, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading %s: %v", path, err)
		}
		text := string(src)
		for _, sym := range bespokeNonTestSymbols {
			if containsIdent(text, sym) {
				t.Errorf("%s: bespoke symbol %q must be deleted from pkg/check non-test source", filepath.Base(path), sym)
			}
		}
		// The named-format entries must be gone from formatParsers.
		for _, fmtName := range []string{`"go-build"`, `"go-test"`, `"golangci-json"`} {
			if strings.Contains(text, fmtName) {
				t.Errorf("%s: named format %s must be removed from formatParsers", filepath.Base(path), fmtName)
			}
		}
		// The `language == "go"` short-circuit must be gone.
		if strings.Contains(text, `language == "go"`) {
			t.Errorf("%s: the `language == \"go\"` short-circuit must be deleted", filepath.Base(path))
		}
	}
}

// TestEndState_NoTestReferencesDeletedBespokeSymbol (CLM-037) proves no test file
// references a deleted bespoke symbol — the bespoke-asserting test files are each
// migrated to the engine path or deleted with their target, so the package
// compiles with the bespoke symbols gone. A dangling reference would be a compile
// break; this scan catches a stale reference even in a file that happened to
// compile (e.g. inside a string).
func TestEndState_NoTestReferencesDeletedBespokeSymbol(t *testing.T) {
	checkDir := filepath.Join(repoRoot(t), "pkg", "check")
	entries, err := os.ReadDir(checkDir)
	if err != nil {
		t.Fatalf("reading pkg/check: %v", err)
	}
	// Symbols that are fully removed (incl. the ForTest equivalence seams) — no
	// test may reference them after the cutover.
	deleted := append([]string{
		"ParseGoBuildErrorsForTest", "ParseGoTestFailuresForTest", "ParseGolangciJSONForTest",
	}, bespokeNonTestSymbols...)
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		src, rerr := os.ReadFile(filepath.Join(checkDir, e.Name()))
		if rerr != nil {
			t.Fatalf("reading %s: %v", e.Name(), rerr)
		}
		for _, sym := range deleted {
			if containsIdent(string(src), sym) {
				t.Errorf("%s still references deleted bespoke symbol %q; migrate or delete it", e.Name(), sym)
			}
		}
	}
}

// TestEndState_NewStackNeedsNoCoreChange (CLM-038) proves adding a new stack
// requires no pkg/check change: a Go project with NO bespoke special-casing now
// constructs executors purely from the data-driven toolchain/engine path. The
// proof: buildExecutorsForConfigErr no longer special-cases Go (no
// goBuiltinExecutors call), so the construction is the same generic path every
// declared stack uses — adding a stack is a pack/declaration, not a core edit.
func TestEndState_NewStackNeedsNoCoreChange(t *testing.T) {
	regPath := filepath.Join(repoRoot(t), "pkg", "check", "registry.go")
	src, err := os.ReadFile(regPath)
	if err != nil {
		t.Fatalf("reading registry.go: %v", err)
	}
	text := string(src)
	if strings.Contains(text, "goBuiltinExecutors") {
		t.Error("registry.go must not construct goBuiltinExecutors; stack support is data-driven, no core special-casing")
	}
	if strings.Contains(text, `language == "go"`) {
		t.Error("registry.go must not special-case `language == \"go\"`; a new stack needs no core change")
	}
	// The generic data-driven path (resolveToolchain + commandExecutor) must remain
	// the single construction route.
	if !strings.Contains(text, "resolveToolchain") {
		t.Error("registry.go must retain the generic resolveToolchain path that serves every stack")
	}
}

// TestCutover_CheckRunNoLongerRunsToolchainPasses (CLM-005) proves the gate's
// realCodeChecker -> check.Run step no longer constructs the native
// lint/build/test passes for a Go project: buildExecutorsForConfigErr for Go
// yields ONLY the shared semgrep executor (the lint/build/test passes now run
// through the engine bridge). Asserted structurally: with the short-circuit and
// goBuiltinExecutors deleted and the Go built-in toolchain carrying no
// lint/build/test entries, no Go-specific native executor is constructed.
func TestCutover_CheckRunNoLongerRunsToolchainPasses(t *testing.T) {
	// Source-level: the construction path has no bespoke Go executor wiring.
	regPath := filepath.Join(repoRoot(t), "pkg", "check", "registry.go")
	src, err := os.ReadFile(regPath)
	if err != nil {
		t.Fatalf("reading registry.go: %v", err)
	}
	for _, banned := range []string{"goBuiltinExecutors", "lintExecutor{", "buildExecutor{", "testExecutor{"} {
		if strings.Contains(string(src), banned) {
			t.Errorf("registry.go still wires a native Go toolchain executor (%q); the engine bridge owns lint/build/test now", banned)
		}
	}
}

// containsIdent reports whether text contains sym as a whole Go identifier
// (word-bounded), so a substring inside a longer name does not false-positive.
// The scan covers comments/strings too, which the end-state grep intentionally
// includes per CLM-036/CLM-037.
func containsIdent(text, sym string) bool {
	if !strings.Contains(text, sym) {
		return false
	}
	for _, idx := range allIndexes(text, sym) {
		left := idx == 0 || !isIdentRune(rune(text[idx-1]))
		rightPos := idx + len(sym)
		right := rightPos >= len(text) || !isIdentRune(rune(text[rightPos]))
		if left && right {
			return true
		}
	}
	return false
}

func allIndexes(s, sub string) []int {
	var out []int
	for i := 0; ; {
		j := strings.Index(s[i:], sub)
		if j < 0 {
			return out
		}
		out = append(out, i+j)
		i += j + len(sub)
	}
}

func isIdentRune(r rune) bool {
	return r == '_' ||
		(r >= 'a' && r <= 'z') ||
		(r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9')
}
