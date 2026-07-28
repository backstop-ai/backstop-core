package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/pack/distribution"
)

// gate_substantiveness_provisioning_test.go pins the PROVISIONING model (SPEC-037
// REQ-009 / CLM-030 / CLM-031): the substantiveness pack is an ORDINARY INSTALLED pack —
// not embedded, not testdata-as-production — and backstop-core dogfood-installs it into
// itself as a LOCAL pack via the real distribution.Add path (declared `local` + locked,
// VerifyLock passes without a remote artifact).

// writeFile writes content to dir/name and returns nil/err for the E2E helpers.
func writeFile(t *testing.T, dir, name, content string) error {
	t.Helper()
	return os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644)
}

// appendMandatedTest adds a claim mandating funcName to the e2e spec so the
// substantiveness step keys an additional test.
func appendMandatedTest(t *testing.T, specDir, funcName string) {
	t.Helper()
	path := filepath.Join(specDir, "e2e.spec.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading spec for append: %v", err)
	}
	// Insert a second claim before the closing frontmatter delimiter's body.
	extra := "  - id: CLM-002\n    requirement: REQ-001\n    text: claim2\n    tests:\n      - " + funcName + "\n"
	s := string(data)
	idx := strings.Index(s, "\n---\n")
	if idx < 0 {
		t.Fatalf("spec frontmatter delimiter not found")
	}
	out := s[:idx] + "\n" + extra + s[idx:]
	if err := os.WriteFile(path, []byte(out), 0o644); err != nil {
		t.Fatalf("writing appended spec: %v", err)
	}
}

// TestProvisioning_SubstantivenessPackNotEmbeddedNorTestdata (CLM-030) — the
// substantiveness rules are absent from the binary: no //go:embed (or other compiled-in
// asset) carries the substantiveness rule YAML, and no PRODUCTION gate code path resolves
// the pack from a testdata directory — the rules are present only in an installed pack
// (the packs/substantiveness/ source).
func TestProvisioning_SubstantivenessPackNotEmbeddedNorTestdata(t *testing.T) {
	repoRoot := repoRoot(t)

	// (1) No //go:embed of the substantiveness rule YAML anywhere in production sources.
	// (2) No production gate code path resolves the pack from a testdata directory.
	prodFiles := []string{
		filepath.Join(repoRoot, "cmd", "backstop", "gate.go"),
		filepath.Join(repoRoot, "cmd", "backstop", "gate_substantiveness_e2e.go"),
		filepath.Join(repoRoot, "cmd", "backstop", "pack_gate.go"),
		filepath.Join(repoRoot, "cmd", "backstop", "code_check.go"),
		filepath.Join(repoRoot, "pkg", "gate", "substantiveness_join.go"),
		filepath.Join(repoRoot, "pkg", "gate", "step_testverify.go"),
	}
	for _, f := range prodFiles {
		data, err := os.ReadFile(f)
		if err != nil {
			continue // file may not exist (e.g. before a phase) — skip, not fail
		}
		src := string(data)
		// Look for an ACTUAL //go:embed directive (a line whose first non-space token
		// is the directive), not the substring appearing in prose/comments.
		for _, line := range strings.Split(src, "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "//go:embed") {
				t.Errorf("%s carries a //go:embed directive — substantiveness rules must live in an installed pack, not the binary: %q", f, trimmed)
			}
		}
		if strings.Contains(src, "testdata/substantiveness-pack") {
			t.Errorf("%s resolves the substantiveness pack from testdata — production must resolve from the INSTALLED pack only (CLM-030)", f)
		}
	}

	// The installable pack SOURCE exists in-repo at packs/substantiveness/ (location B),
	// carrying its own rule YAML + convert script — the only place the rules live.
	for _, rel := range []string{
		"packs/substantiveness/pack.yml",
		"packs/substantiveness/ast-grep/sgconfig.yml",
		"packs/substantiveness/ast-grep/rules/hollow-test-go.yml",
		"packs/substantiveness/ast-grep/rules/referenced-symbol-go.yml",
		"packs/substantiveness/ast-grep/to-sarif.sh",
	} {
		if _, err := os.Stat(filepath.Join(repoRoot, rel)); err != nil {
			t.Errorf("installable pack source missing %s: %v", rel, err)
		}
	}
}

// TestProvisioning_SubstantivenessInstalledAsLocalPack_DeclaredAndLocked (CLM-031) —
// after a production-assembled pack add over the packs/substantiveness/ local source, backstop.yml
// declares the pack with the `local` source value and the lockfile carries a `local`
// SourceType entry, and VerifyLock PASSES WITHOUT a remote artifact (local packs skipped).
func TestProvisioning_SubstantivenessInstalledAsLocalPack_DeclaredAndLocked(t *testing.T) {
	repoRoot := repoRoot(t)
	tmp := t.TempDir()

	// Minimal backstop.yml so distribution.Add can read-modify-write it.
	if err := os.WriteFile(filepath.Join(tmp, "backstop.yml"), []byte("project: prov\nlanguage: go\npacks: {}\n"), 0o644); err != nil {
		t.Fatalf("writing backstop.yml: %v", err)
	}

	add, err := newProductionAddCommand()
	if err != nil {
		t.Fatalf("assembling the production pack add command: %v", err)
	}

	_, err = add.Run(filepath.Join(repoRoot, "packs", "substantiveness"), distribution.AddOptions{ProjectDir: tmp})
	if err != nil {
		t.Fatalf("pack add (local pack): %v", err)
	}

	// backstop.yml declares the pack with the `local` source value.
	ymlData, err := os.ReadFile(filepath.Join(tmp, "backstop.yml"))
	if err != nil {
		t.Fatalf("reading backstop.yml: %v", err)
	}
	yml := string(ymlData)
	if !strings.Contains(yml, "backstop/substantiveness") {
		t.Errorf("backstop.yml must declare the substantiveness pack; got:\n%s", yml)
	}
	if !strings.Contains(yml, "local") {
		t.Errorf("backstop.yml must declare the substantiveness pack with the `local` source value; got:\n%s", yml)
	}

	// The lockfile carries a `local` SourceType entry.
	lf, err := distribution.ReadLockfile(filepath.Join(tmp, "backstop.lock"))
	if err != nil {
		t.Fatalf("reading lockfile: %v", err)
	}
	entry, ok := lf.Packs["backstop/substantiveness"]
	if !ok {
		t.Fatalf("lockfile missing the substantiveness pack entry; got %#v", lf.Packs)
	}
	if entry.SourceType != "local" {
		t.Errorf("lockfile SourceType = %q, want \"local\"", entry.SourceType)
	}

	// VerifyLock PASSES without a remote artifact (local packs are skipped).
	result, err := distribution.VerifyLock(lf, filepath.Join(tmp, ".backstop", "packs"), []string{"backstop/substantiveness"})
	if err != nil {
		t.Fatalf("VerifyLock: %v", err)
	}
	if !result.Pass {
		t.Errorf("VerifyLock must PASS for a local pack without a remote artifact; failures: %#v", result.Failures)
	}
}
