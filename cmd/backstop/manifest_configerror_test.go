package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/bmanson/backstop-core/pkg/check"
)

// SPEC-039 REQ-010 PRESERVATION tests. These guard that deleting the
// manifest-path zero-routable ConfigError EMISSION does NOT remove the
// ConfigError TYPE or any .manifest.json-INDEPENDENT trigger, and that the
// .standard.md scaffolder is out of scope. The MissingToolchain assertion holds
// BEFORE and AFTER the deletion: it drives the exit-2 via the registry path
// (resolveToolchain/validateToolchainKeys), never via a .manifest.json fixture.

// missingToolchainProjectNoManifest scaffolds a temp project whose backstop.yml
// declares a language (rust) with no built-in toolchain and no
// enforcement.toolchain declaration, and a single source file so scope is
// non-empty. Deliberately writes NO .manifest.json — the missing-toolchain
// ConfigError must come from the registry path alone.
func missingToolchainProjectNoManifest(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "backstop.yml"), []byte("project: no-toolchain\nlanguage: rust\n"), 0o644); err != nil {
		t.Fatalf("write backstop.yml: %v", err)
	}
	// .backstop/ must exist (ValidateBackstopDir) but carries NO .manifest.json —
	// the missing-toolchain ConfigError must come from the registry path alone.
	if err := os.MkdirAll(filepath.Join(dir, ".backstop"), 0o755); err != nil {
		t.Fatalf("mkdir .backstop: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "lib.rs"), []byte("fn main() {}\n"), 0o644); err != nil {
		t.Fatalf("write lib.rs: %v", err)
	}
	return dir
}

// TestCodeCheck_MissingToolchain_NoDeclaredToolchainIsCleanNotConfigError pins
// the post-SPEC-040 standalone-subcommand behavior: a declared language with NO
// enforcement.toolchain (and no baked builtin stack — the baked stacks are
// DELETED) resolves to an EMPTY executor set and runs CLEAN (exit 0), NOT an
// exit-2 config error. Enforcement is opt-in: lint/build/test come from a
// <lang>-toolchain pack through the engine path, and a project with none hits the
// no-toolchain-pack WARN-ONLY loud state on the gate — the standalone code check
// subcommand simply has no native passes to run. This REPLACES the pre-cutover
// "missing toolchain = config error" invariant the deleted builtin stack carried.
func TestCodeCheck_MissingToolchain_NoDeclaredToolchainIsCleanNotConfigError(t *testing.T) {
	dir := missingToolchainProjectNoManifest(t)
	restore := chdirTemp(t, dir)
	defer restore()

	origRun := checkRunFn
	defer func() { checkRunFn = origRun }()
	checkRunFn = func(ctx context.Context, opts check.Options) (*check.Result, error) {
		return check.RunWith(ctx, check.RunOptions{Options: opts})
	}

	root := NewRootCommand()
	root.SetArgs([]string{"code", "check", "--all"})
	err := root.Execute()
	if err != nil {
		t.Fatalf("code check --all over a declared language with no toolchain must run clean (empty executor set, exit 0) after the baked-stack deletion; got error: %v", err)
	}
}

// TestStandardScaffolder_Untouched pins CLM-018: the .standard.md scaffolder
// (pkg/pack/scaffold.go) is out of scope for SPEC-039 — it still exists and this
// change does not reference or remove it (scope fence for ISSUE-030).
func TestStandardScaffolder_Untouched(t *testing.T) {
	// The scaffolder file lives at <repo>/pkg/pack/scaffold.go. From the
	// cmd/backstop package dir (CWD during tests), that is ../../pkg/pack.
	p := filepath.Join("..", "..", "pkg", "pack", "scaffold.go")
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("pkg/pack/scaffold.go must remain (ISSUE-030 scope fence); stat error: %v", err)
	}
}
