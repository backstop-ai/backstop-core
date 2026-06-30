package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bmanson/backstop-core/pkg/config"
	"github.com/bmanson/backstop-core/pkg/pack"
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

// spec046LoadFixture loads a spec046 testdata gate config through the real config
// loader (exercising the actual strict-decode + schema path, which after SPEC-046
// accepts a config with no `language:` field and ignores a stray `language:` key).
func spec046LoadFixture(t *testing.T, name string) *config.Config {
	t.Helper()
	cfg, err := config.LoadConfigFromPath(spec046FixturePath(t, name))
	if err != nil {
		t.Fatalf("loading spec046 fixture %s: %v", name, err)
	}
	return cfg
}

// spec046ToolchainManifest builds a declared-pack manifest STUB for a toolchain
// pack named normalizedName (e.g. "backstop/go-toolchain", "backstop/bun-toolchain").
// It lets the count/enforcement/stack-label assertions exercise the declared-pack
// path without a real on-disk pack — the polyglot bun-toolchain pack does not exist
// until SPEC-047, so its dispatch/count/label behavior is proven over a stub here.
func spec046ToolchainManifest(normalizedName string) *pack.Manifest {
	return &pack.Manifest{Name: normalizedName, NormalizedName: normalizedName}
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
