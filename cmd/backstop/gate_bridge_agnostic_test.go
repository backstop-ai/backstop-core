package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/pack"
)

// typescriptToolchainProjectRoot returns the testdata project root whose
// .backstop/packs holds the non-Go typescript-toolchain fixture pack (TASK-001).
func typescriptToolchainProjectRoot(t *testing.T) string {
	t.Helper()
	return filepath.Join(repoRoot(t), "cmd", "backstop", "testdata", "typescript-toolchain")
}

// TestToolchainPack_BridgeResolutionLanguageAgnostic proves the bridge resolves
// the <lang>-toolchain pack for a NON-Go declared language from disk, deriving
// the on-disk pack name from the language rather than hardcoding go-toolchain —
// the go-only short-circuit is gone (CLM-014, Sharp Edge 7).
func TestToolchainPack_BridgeResolutionLanguageAgnostic(t *testing.T) {
	root := typescriptToolchainProjectRoot(t)

	bridged, err := loadBridgedToolchainPacks(root, "typescript", nil)
	if err != nil {
		t.Fatalf("loadBridgedToolchainPacks(typescript): %v", err)
	}
	if len(bridged) != 1 {
		t.Fatalf("expected the typescript-toolchain pack to resolve for a typescript project, got %d packs", len(bridged))
	}
	if bridged[0].NormalizedName != "backstop/typescript-toolchain" {
		t.Fatalf("bridge resolved the wrong pack for typescript: got %q, want backstop/typescript-toolchain — the on-disk name must derive from the declared language, not hardcode go", bridged[0].NormalizedName)
	}
}

// TestToolchainPack_LintBuildTestAsLayer0EnginePasses proves a resolved
// <lang>-toolchain pack's lint/build/test passes dispatch through
// dispatchPackEngines as Layer-0 engine passes (CLM-013). The go-toolchain pack
// is leveraged unchanged as the Go worked example.
func TestToolchainPack_LintBuildTestAsLayer0EnginePasses(t *testing.T) {
	m := goToolchainManifest(t)
	stubSandboxedRunStdout(t, nil)
	runner := &fixtureRunner{byCmd: map[string][]byte{
		"go build":      readFixture(t, "go-build-errors.txt"),
		"go test":       readFixture(t, "go-test-failures.txt"),
		"golangci-lint": readFixture(t, "golangci-v2.sarif"),
	}}

	// Partition dedicated-step gate-types (coverage, substantiveness, contracts) out
	// of the SARIF findings dispatch exactly as the production gate does — the
	// SPEC-042 go-coverage engine routes to the coverage-records channel, not SARIF,
	// so it must not be fed to dispatchPackEngines.
	violations, err := dispatchPackEngines(excludeDedicatedStepRules([]*pack.Manifest{m}), goToolchainPacksDir(t), t.TempDir(), nil, runner)
	if err != nil {
		t.Fatalf("dispatchPackEngines over the toolchain pack: %v", err)
	}
	if len(violations) != 8 {
		t.Fatalf("expected 8 lint+build+test violations dispatched as engine passes, got %d", len(violations))
	}
	for _, v := range violations {
		if !strings.HasPrefix(v.Rule, "backstop/go-toolchain/") {
			t.Errorf("toolchain pass violation must be namespaced to the pack engine path, got %q", v.Rule)
		}
	}
}

// TestToolchainPack_NewLanguageNeedsNoCoreChange proves adding a new language's
// toolchain needs only a <lang>-toolchain pack — the bridge resolution keys off
// the DECLARED language generically, not a per-language code branch (CLM-015).
// It asserts the SAME loadBridgedToolchainPacks call resolves two DIFFERENT
// languages' packs purely from their declared language argument, with no
// language literal hardcoded into the resolution path beyond the convention.
func TestToolchainPack_NewLanguageNeedsNoCoreChange(t *testing.T) {
	// Go resolves go-toolchain; typescript resolves typescript-toolchain — same
	// function, language-derived name, no per-language branch.
	tsRoot := typescriptToolchainProjectRoot(t)
	tsBridged, err := loadBridgedToolchainPacks(tsRoot, "typescript", nil)
	if err != nil {
		t.Fatalf("typescript bridge: %v", err)
	}
	if len(tsBridged) != 1 || tsBridged[0].NormalizedName != "backstop/typescript-toolchain" {
		t.Fatalf("typescript must resolve its own pack via the generic language-derived name, got %#v", tsBridged)
	}

	goRoot := goToolchainProjectRoot(t)
	goBridged, err := loadBridgedToolchainPacks(goRoot, "go", nil)
	if err != nil {
		t.Fatalf("go bridge: %v", err)
	}
	if len(goBridged) != 1 || goBridged[0].NormalizedName != "backstop/go-toolchain" {
		t.Fatalf("go must resolve its own pack via the generic language-derived name, got %#v", goBridged)
	}

	// Source guard: the bridge must not re-introduce a `language == "go"` /
	// `language != "go"` short-circuit (Sharp Edge 7). The resolution must be
	// generic over the declared language.
	src := readFileStr(t, "gate.go")
	for _, banned := range []string{`language != "go"`, `language == "go"`} {
		if strings.Contains(src, banned) {
			t.Errorf("gate.go contains a go-only short-circuit %q; the bridge must be language-agnostic (CLM-014/CLM-015)", banned)
		}
	}
}

// goToolchainProjectRoot returns the testdata project root whose .backstop/packs
// holds the go-toolchain fixture pack — the Go worked example.
func goToolchainProjectRoot(t *testing.T) string {
	t.Helper()
	root := filepath.Join(repoRoot(t), "cmd", "backstop", "testdata", "go-toolchain")
	if _, err := os.Stat(filepath.Join(root, ".backstop", "packs", "backstop", "go-toolchain")); err != nil {
		t.Fatalf("go-toolchain fixture pack missing: %v", err)
	}
	return root
}
