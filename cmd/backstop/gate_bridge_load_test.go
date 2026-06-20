package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bmanson/backstop-core/pkg/pack"
)

// installFixtureToolchainPack copies the testdata go-toolchain mechanism pack
// into projectRoot/.backstop/packs/backstop/go-toolchain so the bridge loader
// resolves it from disk (the convert scripts must live on disk for the real
// SandboxedRunStdout path).
func installFixtureToolchainPack(t *testing.T, projectRoot string) {
	t.Helper()
	src := goToolchainPackRoot(t)
	dst := filepath.Join(projectRoot, ".backstop", "packs", "backstop", "go-toolchain")
	if err := os.MkdirAll(filepath.Join(dst, "scripts"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{"pack.yml", ".golangci.yml", "scripts/build-to-sarif.sh", "scripts/test-to-sarif.sh"} {
		b, err := os.ReadFile(filepath.Join(src, rel))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		if err := os.WriteFile(filepath.Join(dst, rel), b, 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
}

// TestLoadBridgedToolchainPacks_NonGoLanguageYieldsNone proves the bridge only
// resolves the Go native-toolchain pack for a Go project; a non-Go language gets
// no bridged packs (a future <lang>-toolchain pack is that language's concern).
func TestLoadBridgedToolchainPacks_NonGoLanguageYieldsNone(t *testing.T) {
	got, err := loadBridgedToolchainPacks(t.TempDir(), "typescript", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected no bridged packs for a non-go language, got %d", len(got))
	}
}

// TestLoadBridgedToolchainPacks_MissingPackDirYieldsNone proves a Go project with
// no installed go-toolchain pack gets no bridged packs (no error) — the bespoke
// path stays live, so there is no enforcement-lapse window in phase 1.
func TestLoadBridgedToolchainPacks_MissingPackDirYieldsNone(t *testing.T) {
	got, err := loadBridgedToolchainPacks(t.TempDir(), "go", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected no bridged packs when the pack dir is absent, got %d", len(got))
	}
}

// TestLoadBridgedToolchainPacks_PresentPackIsLoaded proves the bridge loads the
// on-disk go-toolchain pack for a Go project, so its lint/build/test engine
// bindings dispatch through the existing substrate.
func TestLoadBridgedToolchainPacks_PresentPackIsLoaded(t *testing.T) {
	root := t.TempDir()
	installFixtureToolchainPack(t, root)
	got, err := loadBridgedToolchainPacks(root, "go", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected the native toolchain pack to be loaded, got %d", len(got))
	}
	if got[0].NormalizedName != "backstop/go-toolchain" {
		t.Errorf("loaded pack = %q, want backstop/go-toolchain", got[0].NormalizedName)
	}
}

// TestLoadBridgedToolchainPacks_DeclaredPackIsDeduped proves the bridge does NOT
// re-add the native pack when it is ALSO a declared project pack (dispatch
// already covers it) — preventing a double-run of lint/build/test.
func TestLoadBridgedToolchainPacks_DeclaredPackIsDeduped(t *testing.T) {
	root := t.TempDir()
	installFixtureToolchainPack(t, root)
	declared := []*pack.Manifest{{NormalizedName: "backstop/go-toolchain"}}
	got, err := loadBridgedToolchainPacks(root, "go", declared)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected the bridge to dedupe an already-declared native pack, got %d", len(got))
	}
}

// TestLoadBridgedToolchainPacks_MalformedPackFailsLoud proves a PRESENT but
// malformed native pack fails loud (a config error), not a silent skip — the
// bridge never swallows a broken pack into a vacuous green.
func TestLoadBridgedToolchainPacks_MalformedPackFailsLoud(t *testing.T) {
	root := t.TempDir()
	dst := filepath.Join(root, ".backstop", "packs", "backstop", "go-toolchain")
	if err := os.MkdirAll(dst, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dst, "pack.yml"), []byte("name: : not valid yaml ::\n\t- broken"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := loadBridgedToolchainPacks(root, "go", nil)
	if err == nil {
		t.Fatal("expected a fail-loud error for a malformed native toolchain pack, got nil")
	}
}

// TestGateLanguage_DefaultsAndReads proves gateLanguage reads the declared
// language and defaults to go for an unreadable/empty config.
func TestGateLanguage_DefaultsAndReads(t *testing.T) {
	// Missing config -> defaults to go.
	if got := gateLanguage(t.TempDir()); got != "go" {
		t.Errorf("missing config: gateLanguage = %q, want go", got)
	}
	// Declared language is read back.
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "backstop.yml"), []byte("project: p\nlanguage: typescript\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := gateLanguage(root); got != "typescript" {
		t.Errorf("declared language: gateLanguage = %q, want typescript", got)
	}
}
