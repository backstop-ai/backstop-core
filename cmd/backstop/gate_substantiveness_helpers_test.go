package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bmanson/backstop-core/pkg/config"
)

// gate_substantiveness_helpers_test.go covers the cmd/backstop-side substantiveness
// helper edge cases the re-wired step + capability re-key depend on: the installed-pack
// presence signal (substantivenessPackInstalled), the resolution filter
// (resolveSubstantivenessPacks empty case), and the string-only same-package derivation
// (goFilePackageMatchesTarget). Real behavior assertions — each pins a disposition the
// gate verdict turns on.

// TestSubstantivenessPackInstalled_ReadsPacksMap — the installed-pack signal keys on the
// backstop.yml packs map: declared → true, absent/nil → false.
func TestSubstantivenessPackInstalled_ReadsPacksMap(t *testing.T) {
	if substantivenessPackInstalled(nil) {
		t.Errorf("nil config must report the pack as NOT installed")
	}
	if substantivenessPackInstalled(&config.Config{}) {
		t.Errorf("a config with no packs map must report NOT installed")
	}
	if substantivenessPackInstalled(&config.Config{Packs: config.Packs{"other/pack": "1.0.0"}}) {
		t.Errorf("a config without the substantiveness pack must report NOT installed")
	}
	if !substantivenessPackInstalled(&config.Config{Packs: config.Packs{"backstop/substantiveness": "local"}}) {
		t.Errorf("a config declaring the substantiveness pack must report INSTALLED")
	}
}

// TestResolveSubstantivenessPacks_FiltersInstalled — with no installed substantiveness
// pack in a temp project, the resolver returns an empty set (the step's no-op path).
func TestResolveSubstantivenessPacks_FiltersInstalled(t *testing.T) {
	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, "backstop.yml"), []byte("project: p\nlanguage: go\npacks: {}\n"), 0o644); err != nil {
		t.Fatalf("writing backstop.yml: %v", err)
	}
	packs, err := resolveSubstantivenessPacks(tmp)
	if err != nil {
		t.Fatalf("resolveSubstantivenessPacks: %v", err)
	}
	if len(packs) != 0 {
		t.Errorf("with no substantiveness pack installed, the resolver must return an empty set; got %d", len(packs))
	}
}

// TestGoFilePackageMatchesTarget_CmdEdgeCases — the cmd/backstop string-only same-package
// derivation: empty target / missing file / external _test variant / same package /
// unrelated package.
func TestGoFilePackageMatchesTarget_CmdEdgeCases(t *testing.T) {
	if goFilePackageMatchesTarget("/nonexistent/x_test.go", "") {
		t.Errorf("empty target is never same-package")
	}
	if goFilePackageMatchesTarget("/nonexistent/missing_test.go", "gate") {
		t.Errorf("a missing file is never same-package")
	}

	dir := t.TempDir()
	ext := filepath.Join(dir, "ext_test.go")
	if err := os.WriteFile(ext, []byte("package gate_test\n\nfunc f() {}\n"), 0o644); err != nil {
		t.Fatalf("writing ext: %v", err)
	}
	if !goFilePackageMatchesTarget(ext, "gate") {
		t.Errorf("package gate_test must match target gate")
	}

	same := filepath.Join(dir, "same_test.go")
	if err := os.WriteFile(same, []byte("package gate\n\nfunc g() {}\n"), 0o644); err != nil {
		t.Fatalf("writing same: %v", err)
	}
	if !goFilePackageMatchesTarget(same, "gate") {
		t.Errorf("package gate must match target gate")
	}

	other := filepath.Join(dir, "other_test.go")
	if err := os.WriteFile(other, []byte("package widget_test\n\nfunc h() {}\n"), 0o644); err != nil {
		t.Fatalf("writing other: %v", err)
	}
	if goFilePackageMatchesTarget(other, "gate") {
		t.Errorf("package widget_test must NOT match target gate")
	}
}
