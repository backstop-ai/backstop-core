package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/pack/distribution"
)

// gate_substantiveness_provisioning_test.go pins the PROVISIONING model (SPEC-037
// REQ-009 / CLM-030 / CLM-031): the substantiveness pack is an ORDINARY INSTALLED pack —
// not embedded, not testdata-as-production — and backstop-core dogfood-installs it into
// itself through the real distribution.Add path, where it is DECLARED, LOCKED and
// RESOLVABLE. EITHER source type satisfies that: a local install (declared `local`, a
// `local` lock entry VerifyLock skips so no remote artifact is required) or a remote one
// (a `git` lock entry carrying its source coordinate and tag). What is pinned is
// installed-and-resolvable via the distribution path, never that the source is local.

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

// substantivenessPackName is the pack's MANIFEST name, which is the install identity
// (SPEC-056) — the backstop.yml key, the lock key, and the engine asset root all read
// it. The git arm below is served at a DIFFERENT source coordinate on purpose; both
// arms still look the pack up here.
const substantivenessPackName = "backstop/substantiveness"

// assertPackDeclaredAs requires backstop.yml's DECLARATION LINE for packName to carry
// want. Scanning for the key's own line rather than the value anywhere in the file is
// what makes this falsifiable: `strings.Contains(yml, "local")` passes on the word
// appearing in any unrelated key, value or comment.
func assertPackDeclaredAs(t *testing.T, projectDir, packName, want string) {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(projectDir, "backstop.yml"))
	if err != nil {
		t.Fatalf("reading backstop.yml: %v", err)
	}
	yml := string(data)
	for _, line := range strings.Split(yml, "\n") {
		key, value, found := strings.Cut(strings.TrimSpace(line), ":")
		if !found || key != packName {
			continue
		}
		if got := strings.TrimSpace(value); got != want {
			t.Errorf("backstop.yml declares %s as %q, want %q; got:\n%s", packName, got, want, yml)
		}
		return
	}
	t.Errorf("backstop.yml carries no declaration line for %s; got:\n%s", packName, yml)
}

// assertLockVerifies requires VerifyLock to PASS for packName over the materialized
// .backstop/packs tree — the property REQ-009 calls "resolvable".
func assertLockVerifies(t *testing.T, projectDir string, lf *distribution.Lockfile, packName string) {
	t.Helper()

	result, err := distribution.VerifyLock(lf, filepath.Join(projectDir, ".backstop", "packs"), []string{packName})
	if err != nil {
		t.Fatalf("VerifyLock: %v", err)
	}
	if !result.Pass {
		t.Errorf("VerifyLock must PASS for the installed %s pack; failures: %#v", packName, result.Failures)
	}
}

// TestProvisioning_SubstantivenessInstalledViaDistributionPath_LocalOrGit_DeclaredAndLocked
// (CLM-031) — backstop-core dogfood-installs the substantiveness pack through the STANDARD
// DISTRIBUTION PATH, and EITHER source type satisfies the claim. Both arms drive the same
// production-assembled `pack add` and assert the same three properties — backstop.yml
// DECLARES the pack, backstop.lock carries a RESOLVABLE entry, VerifyLock PASSES — then the
// specifics of their source type.
//
// The claim is a DISJUNCTION, so a single-arm test leaves the half the amendment was written
// for unfalsified: backstop-core's own pack migrated local → git in 905120f while the
// local-only test stayed green.
func TestProvisioning_SubstantivenessInstalledViaDistributionPath_LocalOrGit_DeclaredAndLocked(t *testing.T) {
	repoRoot := repoRoot(t)
	packSource := filepath.Join(repoRoot, "packs", "substantiveness")

	t.Run("local-source", func(t *testing.T) {
		project := t.TempDir()

		// Minimal backstop.yml so distribution.Add can read-modify-write it.
		if err := os.WriteFile(filepath.Join(project, "backstop.yml"), []byte("project: prov\nlanguage: go\npacks: {}\n"), 0o644); err != nil {
			t.Fatalf("writing backstop.yml: %v", err)
		}

		add, err := newProductionAddCommand()
		if err != nil {
			t.Fatalf("assembling the production pack add command: %v", err)
		}
		if _, err := add.Run(packSource, distribution.AddOptions{ProjectDir: project}); err != nil {
			t.Fatalf("pack add (local source): %v", err)
		}

		assertPackDeclaredAs(t, project, substantivenessPackName, "local")

		lf, err := distribution.ReadLockfile(filepath.Join(project, "backstop.lock"))
		if err != nil {
			t.Fatalf("reading lockfile: %v", err)
		}
		entry, ok := lf.Packs[substantivenessPackName]
		if !ok {
			t.Fatalf("lockfile missing the substantiveness pack entry; got %#v", lf.Packs)
		}
		if entry.SourceType != "local" {
			t.Errorf("lockfile SourceType = %q, want \"local\"", entry.SourceType)
		}

		// VerifyLock PASSES without a remote artifact (local packs are skipped).
		assertLockVerifies(t, project, lf, substantivenessPackName)
	})

	t.Run("git-source", func(t *testing.T) {
		// FAIL, never skip, when git is absent. The harness's own requireGit calls
		// t.Skip, and a SKIPPED subtest leaves the PARENT test PASSING — the suite
		// would report this mandated test green with the git arm never executed,
		// which is the vacuous green this spec exists to prevent.
		if _, err := exec.LookPath("git"); err != nil {
			t.Fatalf("git is not on PATH: CLM-031 is a claim about the distribution path, and the distribution path IS git — this environment cannot make a statement about it either way: %v", err)
		}

		// newConsumerProject, NOT a bare t.TempDir(): updateBackstopYml READS
		// backstop.yml and returns the read error when it is missing, so an add into
		// an empty directory fails before it declares anything.
		project := newConsumerProject(t)

		// v1.1.0 is the manifest's own declared version; the harness rewrites pack.yml's
		// version to the tag anyway, so the two cannot drift.
		remote := newHermeticRemote(t, packSource, "v1.1.0")
		redirectPackURL(t, remoteE2EOrg, "substantiveness", remote.Path)
		// PROVE the redirect reached a child process BEFORE any other assertion: a
		// mismatched pair does not fail, it MISSES, and the arm quietly reaches the
		// network and passes for the wrong reason.
		assertPackURLRedirected(t, remoteE2EOrg, "substantiveness", remote)

		add, err := newProductionAddCommand()
		if err != nil {
			t.Fatalf("assembling the production pack add command: %v", err)
		}
		// The hermetic remote serves the pack at <org>/substantiveness while its
		// manifest declares backstop/substantiveness. That divergence is DIAGNOSTIC,
		// not a refusal (SPEC-056): the add SUCCEEDS and installs under the manifest
		// name, which is why both arms look up the same lock key.
		if _, err := add.Run(remoteE2EOrg+"/substantiveness@1.1.0", distribution.AddOptions{ProjectDir: project}); err != nil {
			t.Fatalf("pack add (git source): %v", err)
		}

		assertPackDeclaredAs(t, project, substantivenessPackName, "1.1.0")

		// Read the lock OFF DISK rather than trusting what Add returned — the
		// COMMITTED lock is what REQ-009 is about.
		lf, err := distribution.ReadLockfile(filepath.Join(project, "backstop.lock"))
		if err != nil {
			t.Fatalf("reading lockfile: %v", err)
		}
		entry, ok := lf.Packs[substantivenessPackName]
		if !ok {
			t.Fatalf("lockfile missing the substantiveness pack entry; got %#v", lf.Packs)
		}
		if entry.SourceType != "git" {
			t.Errorf("lockfile SourceType = %q, want \"git\" — a remote install that recorded a local source proves nothing was cloned", entry.SourceType)
		}
		if entry.SourceCoordinate == "" {
			t.Error("the lock entry carries no source_coordinate; without it a fresh clone cannot resolve where the pack came from")
		}
		if entry.GitRef == nil || *entry.GitRef != "v1.1.0" {
			t.Errorf("lock entry git_ref = %v, want v1.1.0", entry.GitRef)
		}
		if entry.ContentHash == "" {
			t.Error("the lock entry carries no content hash; there would be nothing for a later install to verify against")
		}

		assertLockVerifies(t, project, lf, substantivenessPackName)
	})
}
