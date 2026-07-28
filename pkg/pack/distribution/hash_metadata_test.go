package distribution_test

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/pack/distribution"
)

// authoredTree writes the shared authored fixture these tests hash. Every name
// sorts AFTER ".git", so a walk that skipped the remainder of the root instead
// of just the metadata entry drops all of them and the equality assertions red.
func authoredTree(t *testing.T, dir string) string {
	t.Helper()
	writeFile(t, filepath.Join(dir, "pack.yml"), "name: fixture\nversion: 1.0.0\n")
	writeFile(t, filepath.Join(dir, "rules", "rule.semgrep.yml"), "rules:\n  - id: fixture\n")
	writeFile(t, filepath.Join(dir, "zz-last.txt"), "sorts after .git\n")
	return dir
}

// writeRootGitDirectory builds a real .git directory with files at two different
// depths, so skipping the entry without pruning the subtree still hashes bytes.
func writeRootGitDirectory(t *testing.T, dir string) {
	t.Helper()
	writeFile(t, filepath.Join(dir, ".git", "HEAD"), "ref: refs/heads/main\n")
	writeFile(t, filepath.Join(dir, ".git", "objects", "ab", "cdef"), "object payload\n")
}

// symlinkOrSkip creates a real symlink or skips. Degrading to a regular file
// would turn the symlink claims into vacuous passes.
func symlinkOrSkip(t *testing.T, target, link string) {
	t.Helper()
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("os.Symlink is unsupported on this host: %v", err)
	}
}

// documentedManifestDigest recomputes the documented content-hash algorithm from
// the spec's description alone: for each file "relpath:hex(sha256(content))",
// forward slashes, sorted, joined with newlines, then sha256 of that string.
func documentedManifestDigest(t *testing.T, root string, relPaths []string) string {
	t.Helper()
	entries := make([]string, 0, len(relPaths))
	for _, rel := range relPaths {
		content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("reading manifest input %s: %v", rel, err)
		}
		fileSum := sha256.Sum256(content)
		entries = append(entries, rel+":"+hex.EncodeToString(fileSum[:]))
	}
	sort.Strings(entries)
	manifestSum := sha256.Sum256([]byte(strings.Join(entries, "\n")))
	return hex.EncodeToString(manifestSum[:])
}

func TestComputeContentHash_RootGitDirectoryExcluded(t *testing.T) {
	withGit := authoredTree(t, t.TempDir())
	writeRootGitDirectory(t, withGit)
	metadataFree := authoredTree(t, t.TempDir())

	if got, want := mustContentHash(t, withGit), mustContentHash(t, metadataFree); got != want {
		t.Fatalf("root .git directory contributed to the hash: with=%s without=%s", got, want)
	}

	// Churn inside the subtree must not move the hash. This is what separates a
	// pruned subtree from an entry that was merely skipped and then descended.
	before := mustContentHash(t, withGit)
	writeFile(t, filepath.Join(withGit, ".git", "objects", "ab", "cdef"), "rewritten object payload\n")
	writeFile(t, filepath.Join(withGit, ".git", "logs", "HEAD"), "reflog churn\n")
	if after := mustContentHash(t, withGit); after != before {
		t.Errorf("churn inside the root .git subtree changed the hash: before=%s after=%s", before, after)
	}

	// Positive control: the authored files must still be hashed, so an exclusion
	// that dropped everything cannot pass this test.
	writeFile(t, filepath.Join(withGit, "pack.yml"), "name: fixture\nversion: 2.0.0\n")
	if after := mustContentHash(t, withGit); after == before {
		t.Error("editing an authored file did not change the hash")
	}
}

func TestComputeContentHash_RootGitFileExcluded(t *testing.T) {
	withPointer := authoredTree(t, t.TempDir())
	writeFile(t, filepath.Join(withPointer, ".git"), "gitdir: /somewhere/.git/worktrees/x\n")
	metadataFree := authoredTree(t, t.TempDir())

	got := mustContentHash(t, withPointer)
	if want := mustContentHash(t, metadataFree); got != want {
		t.Fatalf("root .git worktree pointer file contributed to the hash: with=%s without=%s", got, want)
	}

	// Rewriting the pointer must not move the hash.
	writeFile(t, filepath.Join(withPointer, ".git"), "gitdir: /elsewhere/.git/worktrees/y\n")
	if after := mustContentHash(t, withPointer); after != got {
		t.Errorf("rewriting the root .git pointer changed the hash: before=%s after=%s", got, after)
	}
}

func TestComputeContentHash_RootGitSymlinkExcludedAndNotFollowed(t *testing.T) {
	outside := t.TempDir()
	target := filepath.Join(outside, "real-git-dir-contents")
	writeFile(t, target, "distinctive content that must never reach the manifest\n")

	withSymlink := authoredTree(t, t.TempDir())
	symlinkOrSkip(t, target, filepath.Join(withSymlink, ".git"))
	metadataFree := authoredTree(t, t.TempDir())

	got := mustContentHash(t, withSymlink)
	if want := mustContentHash(t, metadataFree); got != want {
		t.Fatalf("root .git symlink contributed to the hash: with=%s without=%s", got, want)
	}

	// Rewriting the symlink's target must not move the hash — the link was never
	// followed, so its bytes are not part of this pack's identity.
	writeFile(t, target, "different distinctive content\n")
	if after := mustContentHash(t, withSymlink); after != got {
		t.Errorf("root .git symlink was followed: hash moved with its target: before=%s after=%s", got, after)
	}
}

func TestComputeContentHash_DanglingRootGitSymlinkDoesNotFailTheWalk(t *testing.T) {
	outside := t.TempDir()
	target := filepath.Join(outside, "target-that-goes-away")
	writeFile(t, target, "about to be removed\n")

	dangling := authoredTree(t, t.TempDir())
	symlinkOrSkip(t, target, filepath.Join(dangling, ".git"))
	if err := os.Remove(target); err != nil {
		t.Fatalf("removing symlink target: %v", err)
	}

	hash, err := distribution.ComputeContentHash(dangling)
	if err != nil {
		t.Fatalf("dangling root .git symlink failed the walk: %v", err)
	}
	if hash == "" {
		t.Fatal("expected a usable hash for a tree carrying a dangling root .git symlink")
	}

	metadataFree := authoredTree(t, t.TempDir())
	if want := mustContentHash(t, metadataFree); hash != want {
		t.Errorf("dangling root .git symlink changed the hash: with=%s without=%s", hash, want)
	}
}

func TestComputeContentHash_IdenticalWithAndWithoutRootGitMetadata(t *testing.T) {
	dir := authoredTree(t, t.TempDir())

	before := mustContentHash(t, dir)

	// The same authored files, now checked out with their own repository beside
	// them. This is the measured 639f74fb/bb86715c divergence.
	writeRootGitDirectory(t, dir)

	after := mustContentHash(t, dir)
	if before != after {
		t.Fatalf("one authored tree hashed differently with a root .git present: without=%s with=%s", before, after)
	}
	if before == "" {
		t.Fatal("expected a non-empty hash for an authored tree")
	}
}

func TestComputeContentHash_AuthoredDotfilesRemainContent(t *testing.T) {
	dir := authoredTree(t, t.TempDir())

	dotfiles := []struct {
		name    string
		relPath string
		content string
		revised string
	}{
		{"gitignore", ".gitignore", "*.tmp\n", "*.tmp\n*.log\n"},
		{"gitattributes", ".gitattributes", "* text=auto\n", "* -text\n"},
		{"github workflow", filepath.Join(".github", "workflows", "ci.yml"), "on: push\n", "on: pull_request\n"},
	}

	previous := mustContentHash(t, dir)
	for _, dotfile := range dotfiles {
		writeFile(t, filepath.Join(dir, dotfile.relPath), dotfile.content)
		added := mustContentHash(t, dir)
		if added == previous {
			t.Errorf("adding %s did not change the hash", dotfile.name)
		}

		writeFile(t, filepath.Join(dir, dotfile.relPath), dotfile.revised)
		revised := mustContentHash(t, dir)
		if revised == added {
			t.Errorf("editing %s did not change the hash", dotfile.name)
		}
		previous = revised
	}
}

func TestComputeContentHash_RootGitPrefixedNamesAreNotMetadata(t *testing.T) {
	// Every root entry whose name merely BEGINS with ".git". No root ".git" at
	// all, so a prefix-matching predicate silently drops authored content here.
	prefixed := []string{
		".gitignore",
		".gitattributes",
		filepath.Join(".github", "workflows", "ci.yml"),
		filepath.Join(".git-hooks", "pre-commit"),
	}

	buildTree := func() string {
		dir := authoredTree(t, t.TempDir())
		for _, rel := range prefixed {
			writeFile(t, filepath.Join(dir, rel), "content of "+filepath.ToSlash(rel)+"\n")
		}
		return dir
	}

	full := mustContentHash(t, buildTree())

	for _, omitted := range prefixed {
		dir := buildTree()
		// Remove the top-level entry, directory or file alike.
		top := strings.Split(filepath.ToSlash(omitted), "/")[0]
		if err := os.RemoveAll(filepath.Join(dir, top)); err != nil {
			t.Fatalf("removing %s: %v", top, err)
		}
		if reduced := mustContentHash(t, dir); reduced == full {
			t.Errorf("removing root entry %s did not change the hash, so it was treated as metadata", top)
		}
	}
}

func TestComputeContentHash_NestedGitDirectoryRemainsContent(t *testing.T) {
	// A .git BELOW the walk root is authored content, deliberately: the exclusion
	// mirrors the cloner's one-path strip and is a property of the root alone.
	dir := authoredTree(t, t.TempDir())
	nested := filepath.Join(dir, "subdir", ".git", "config")
	writeFile(t, nested, "[core]\n\tbare = false\n")

	withNested := mustContentHash(t, dir)

	writeFile(t, nested, "[core]\n\tbare = true\n")
	if edited := mustContentHash(t, dir); edited == withNested {
		t.Error("editing a nested .git file did not change the hash, so it was wrongly excluded")
	}

	if err := os.RemoveAll(filepath.Join(dir, "subdir", ".git")); err != nil {
		t.Fatalf("removing nested .git: %v", err)
	}
	if removed := mustContentHash(t, dir); removed == withNested {
		t.Error("removing the nested .git did not change the hash, so it never contributed")
	}
}

func TestComputeContentHash_DirectoryHoldingOnlyRootGitHashesAsEmpty(t *testing.T) {
	onlyMetadata := t.TempDir()
	writeRootGitDirectory(t, onlyMetadata)

	// Computed here rather than pasted, so this asserts against the algorithm's
	// own empty-directory value rather than against a captured digest.
	empty := mustContentHash(t, t.TempDir())

	if got := mustContentHash(t, onlyMetadata); got != empty {
		t.Fatalf("a directory holding only root repository metadata did not hash as empty: got=%s want=%s", got, empty)
	}
}

func TestComputeContentHash_MetadataFreeTreeMatchesDocumentedManifest(t *testing.T) {
	dir := t.TempDir()
	relPaths := []string{
		"pack.yml",
		"rules/rule.semgrep.yml",
		"docs/README.md",
	}
	for _, rel := range relPaths {
		writeFile(t, filepath.Join(dir, filepath.FromSlash(rel)), "content of "+rel+"\n")
	}

	want := documentedManifestDigest(t, dir, relPaths)
	if got := mustContentHash(t, dir); got != want {
		t.Fatalf("ComputeContentHash diverged from the documented manifest algorithm: got=%s want=%s", got, want)
	}
}
