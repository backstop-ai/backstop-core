package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/pack"
	"github.com/backstop-ai/backstop-core/pkg/pack/engine"
)

// SPEC-046 shared test fixtures + helpers. The four spec046-*.yml gate configs in
// testdata declare toolchain packs ONLY via `packs:` (the declared-pack path) with
// NO `language:` field (except the language-key pair fixture, which carries a stray
// `language:` key to prove inert-parse + verdict-invariance). Declared-pack manifest
// STUBS let the dispatch/count/label assertions run without installing the
// not-yet-built bun-toolchain pack (that pack is SPEC-047's concern).

// spec046FixturePath returns the absolute path to a spec046 testdata gate config.
func spec046FixturePath(t *testing.T, name string) string {
	t.Helper()
	return filepath.Join(repoRoot(t), "cmd", "backstop", "testdata", name)
}

// goToolchainProjectRoot returns the testdata project root whose .backstop/packs
// holds the go-toolchain fixture pack — the Go worked example. (Relocated here from
// the deleted bridge tests; surviving cutover/seam tests still use it. After the
// SPEC-046 bridge deletion the project DECLARES backstop/go-toolchain in its
// backstop.yml `packs:`, so the declared-pack path resolves it.)
func goToolchainProjectRoot(t *testing.T) string {
	t.Helper()
	root := filepath.Join(repoRoot(t), "cmd", "backstop", "testdata", "go-toolchain")
	if _, err := os.Stat(filepath.Join(root, ".backstop", "packs", "backstop", "go-toolchain")); err != nil {
		t.Fatalf("go-toolchain fixture pack missing: %v", err)
	}
	return root
}

// spec046ToolchainManifest builds a declared-pack manifest STUB for a toolchain
// pack named normalizedName (e.g. "backstop/go-toolchain", "backstop/bun-toolchain").
// It lets the count/enforcement/stack-label assertions exercise the declared-pack
// path without a real on-disk pack — the polyglot bun-toolchain pack does not exist
// until SPEC-047, so its dispatch/count/label behavior is proven over a stub here.
func spec046ToolchainManifest(normalizedName string) *pack.Manifest {
	// A toolchain pack is recognized BY DECLARATION (ISSUE-063 principle): it declares an
	// enforcement-mechanism engine. Declare a build (typecheck) engine so
	// countToolchainPacks counts it, and DECLARE the pack's language so the cosmetic stack
	// label reads it by declaration (ISSUE-064) — no longer name-derived. A real toolchain
	// pack ships `language:`; this stub simulates that by declaring the language the pack's
	// name implies (backstop/go-toolchain -> "go", backstop/bun-toolchain -> "bun"), so the
	// polyglot label union stays meaningful while the VALUE now flows from manifest.Language.
	return &pack.Manifest{
		Name:           normalizedName,
		NormalizedName: normalizedName,
		Language:       spec046StubLanguage(normalizedName),
		Engines: map[string]pack.EngineSpec{
			"typecheck": {Binding: engine.EngineBinding{GateType: engine.GateTypeBuild}},
		},
	}
}

// spec046StubLanguage returns the language a stub toolchain pack DECLARES, simulating a
// real pack's `language:` field. It derives the value from the pack name only to keep the
// stub convenient — production reads manifest.Language directly and never inspects the
// name (proven by TestToolchainStackLabel_ByDeclaredLanguageNotName, which declares a
// language that diverges from the name).
func spec046StubLanguage(normalizedName string) string {
	name := normalizedName
	if i := strings.LastIndex(name, "/"); i >= 0 {
		name = name[i+1:]
	}
	return strings.TrimSuffix(name, "-toolchain")
}

// spec046InstallGoToolchainProject builds a temp project root that DECLARES
// backstop/go-toolchain in `packs:` (the fixtureName config, copied to backstop.yml)
// and installs the go-toolchain fixture pack on disk under .backstop/packs, so
// loadInstalledPacks resolves it via the DECLARED-pack path (no bridge, no language).
// It is the live-gate no-regression shape: a toolchain pack acquired ONLY through
// `packs:`.
func spec046InstallGoToolchainProject(t *testing.T, fixtureName string) string {
	t.Helper()
	root := t.TempDir()
	src, err := os.ReadFile(spec046FixturePath(t, fixtureName))
	if err != nil {
		t.Fatalf("reading spec046 fixture %s: %v", fixtureName, err)
	}
	if err := os.WriteFile(filepath.Join(root, "backstop.yml"), src, 0o644); err != nil {
		t.Fatalf("writing backstop.yml: %v", err)
	}
	dst := filepath.Join(root, ".backstop", "packs", "backstop", "go-toolchain")
	copyDirRecForTest(t, goToolchainPackRoot(t), dst)
	return root
}

// copyDirRecForTest recursively copies a directory tree (used to install a fixture
// pack on disk so the declared-pack path resolves it).
func copyDirRecForTest(t *testing.T, src, dst string) {
	t.Helper()
	entries, err := os.ReadDir(src)
	if err != nil {
		t.Fatalf("reading %s: %v", src, err)
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dst, err)
	}
	for _, e := range entries {
		s := filepath.Join(src, e.Name())
		d := filepath.Join(dst, e.Name())
		if e.IsDir() {
			copyDirRecForTest(t, s, d)
			continue
		}
		b, err := os.ReadFile(s)
		if err != nil {
			t.Fatalf("reading %s: %v", s, err)
		}
		if err := os.WriteFile(d, b, 0o644); err != nil {
			t.Fatalf("writing %s: %v", d, err)
		}
	}
}
