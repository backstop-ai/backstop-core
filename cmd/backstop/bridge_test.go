package main

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/pack"
)

// TestGoToolchain_NoEmbeddedBuildTestParser proves backstop-core embeds no Go
// build/test parser on the engine path: the build/test SARIF is produced by the
// pack convert script, not an in-binary parse of compiler/test output (CLM-014).
// It drives the build engine with a convert stub that records it received the
// RAW tool stdout (so the transform happens in the pack), and asserts the result
// reflects the convert, not any in-binary parse.
func TestGoToolchain_NoEmbeddedBuildTestParser(t *testing.T) {
	m := onlyRules(goToolchainManifest(t), "go-build")
	var gotStdin []byte
	stubSandboxedRunStdout(t, &gotStdin)
	raw := readFixture(t, "go-build-errors.txt")
	runner := &fixtureRunner{byCmd: map[string][]byte{"go build": raw}}

	violations, err := dispatchPackEngines([]*pack.Manifest{m}, goToolchainPacksDir(t), t.TempDir(), nil, runner)
	if err != nil {
		t.Fatalf("dispatchPackEngines: %v", err)
	}
	// The convert script received the raw tool stdout — the transform is pack
	// code, not an in-binary parser.
	if string(gotStdin) != string(raw) {
		t.Fatalf("the pack convert script must receive the raw `go build` stdout; the transform is NOT embedded in core")
	}
	if len(violations) == 0 {
		t.Fatal("expected the pack convert to produce findings")
	}
}

// TestGoToolchain_NoEmbeddedBuildTestParserSource is a source self-check: the
// bridge file (pack_gate.go) must not parse compiler/test output itself — it
// owns no go-build/go-test parser. The transform resolves from the pack convert
// script run via SandboxedRunStdout (CLM-014 / Sharp Edge 7).
func TestGoToolchain_NoEmbeddedBuildTestParserSource(t *testing.T) {
	src := readFileStr(t, "pack_gate.go")
	for _, banned := range []string{"parseGoBuildErrors", "parseGoTestFailures", "goBuildErrorRe", "goTestFailRe"} {
		if strings.Contains(src, banned) {
			t.Errorf("pack_gate.go references %q; the build/test transform must live in the pack convert script, not the binary", banned)
		}
	}
}

// TestBridge_NativePassesRunThroughDispatchPackEngines proves the native
// lint/build/test passes are dispatched through dispatchPackEngines as engine
// bindings, not through bespoke pkg/check PassExecutors (CLM-001). The
// go-toolchain pack's three rules each resolve to a registered engine binding
// that dispatch runs; a fixtureRunner feeds captured tool output.
func TestBridge_NativePassesRunThroughDispatchPackEngines(t *testing.T) {
	m := goToolchainManifest(t)
	stubSandboxedRunStdout(t, nil)
	runner := &fixtureRunner{byCmd: map[string][]byte{
		"go build":      readFixture(t, "go-build-errors.txt"),
		"go test":       readFixture(t, "go-test-failures.txt"),
		"golangci-lint": readFixture(t, "golangci-v2.sarif"),
	}}

	// Partition dedicated-step gate-types (the SPEC-042 coverage producer) out of the
	// SARIF findings dispatch as the production gate does — coverage routes to the
	// coverage-records channel, leaving the three native SARIF passes here.
	violations, err := dispatchPackEngines(excludeDedicatedStepRules([]*pack.Manifest{m}), goToolchainPacksDir(t), t.TempDir(), nil, runner)
	if err != nil {
		t.Fatalf("dispatchPackEngines (all three native passes): %v", err)
	}
	// 3 build + 3 test + 2 lint = 8 normalized violations, all namespaced to the
	// go-toolchain pack — proving all three native passes ran through the SAME
	// dispatch substrate.
	if len(violations) != 8 {
		t.Fatalf("expected 8 violations across lint+build+test via dispatch, got %d: %#v", len(violations), violations)
	}
	for _, v := range violations {
		if !strings.HasPrefix(v.Rule, "backstop/go-toolchain/") {
			t.Errorf("native pass violation must be namespaced to the pack engine path, got %q", v.Rule)
		}
	}
}

// TestBridge_NoParallelDispatcher proves the bridge reuses dispatchPackEngines
// and introduces no second native dispatcher (CLM-003). The gate wiring
// (gate.go) and the bridge (pack_gate.go) must reference dispatchPackEngines and
// must NOT define a parallel native-dispatch function.
func TestBridge_NoParallelDispatcher(t *testing.T) {
	gateSrc := readFileStr(t, "gate.go")
	if !strings.Contains(gateSrc, "dispatchPackEngines") {
		t.Error("gate.go must route the native passes through the existing dispatchPackEngines, not a new dispatcher")
	}
	// No alternate native-dispatch entry point may exist across the package.
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	banned := []string{"dispatchNativeEngines", "dispatchToolchain", "runNativeDispatch", "dispatchGoToolchain"}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		src := readFileStr(t, e.Name())
		for _, b := range banned {
			if strings.Contains(src, "func "+b) {
				t.Errorf("%s defines a parallel native dispatcher %q; the bridge must reuse dispatchPackEngines", e.Name(), b)
			}
		}
	}
}

// TestBridge_NoCheckToEngineImport proves pkg/check does NOT import
// pkg/pack/engine after the bridge (CLM-002 / Sharp Edge 1). Orchestration of
// both substrates lives at cmd/backstop, which already imports both; pkg/check
// stays a disjoint leaf. Implemented as an import scan over every non-test Go
// file in pkg/check.
func TestBridge_NoCheckToEngineImport(t *testing.T) {
	checkDir := filepath.Join(repoRoot(t), "pkg", "check")
	entries, err := os.ReadDir(checkDir)
	if err != nil {
		t.Fatalf("reading pkg/check: %v", err)
	}
	fset := token.NewFileSet()
	bannedImport := "github.com/backstop-ai/backstop-core/pkg/pack/engine"
	scanned := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		f, perr := parser.ParseFile(fset, filepath.Join(checkDir, e.Name()), nil, parser.ImportsOnly)
		if perr != nil {
			t.Fatalf("parsing %s: %v", e.Name(), perr)
		}
		scanned++
		for _, imp := range f.Imports {
			if strings.Trim(imp.Path.Value, `"`) == bannedImport {
				t.Errorf("%s imports %s; pkg/check must NOT import pkg/pack/engine (import-cycle / leaf-placement guard)", e.Name(), bannedImport)
			}
		}
	}
	if scanned == 0 {
		t.Fatal("expected to scan pkg/check non-test sources")
	}
}
