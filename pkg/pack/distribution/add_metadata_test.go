package distribution_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bmanson/backstop-core/pkg/pack/distribution"
)

// metadataPackName is the pack these fixtures author. Every add runs in its own
// project directory, so the same name may be installed by several tests.
const metadataPackName = "internal/metadata-pack"

// writeMetadataPackSource authors a valid local pack source. Every file name
// sorts AFTER ".git", so a copy that abandoned the rest of the root instead of
// just the metadata entry would drop all of them.
func writeMetadataPackSource(t *testing.T, dir string) {
	t.Helper()
	writeFile(t, filepath.Join(dir, "pack.yml"),
		"name: "+metadataPackName+"\n"+
			"archetype: rule-pack\n"+
			"description: A local pack fixture for the root-metadata boundary\n"+
			"rules:\n"+
			"  - id: META-001\n"+
			"    severity: error\n"+
			"    risk_class: high\n"+
			"    description: Metadata rule one\n"+
			"    pattern: \"meta-pattern-1\"\n"+
			"scaffolds: []\n")
	writeFile(t, filepath.Join(dir, "rules", "r1.yml"), "rules:\n  - id: META-001\n")
	writeFile(t, filepath.Join(dir, "zz-last.txt"), "sorts after .git\n")
}

// addLocalPackSource runs a real local-path add through the command constructor
// and returns the recorded content hash.
func addLocalPackSource(t *testing.T, projectDir, sourceDir string) string {
	t.Helper()
	add := newTestAddCommand(t, defaultTestPackCloner(), &mockValidator{})
	result, err := add.Run(sourceDir, distribution.AddOptions{ProjectDir: projectDir})
	if err != nil {
		t.Fatalf("add local pack from %s: %v", sourceDir, err)
	}
	return result.ContentHash
}

// lockedHashAfterAdd adds sourceDir into a FRESH project and returns the hash the
// lock recorded. A separate project per add is required: an add refuses a pack
// that is already installed.
func lockedHashAfterAdd(t *testing.T, sourceDir string) string {
	t.Helper()
	projectDir := setupAddProject(t)
	addLocalPackSource(t, projectDir, sourceDir)

	lf := mustReadLock(t, filepath.Join(projectDir, "backstop.lock"))
	entry, ok := lf.Packs[metadataPackName]
	if !ok {
		t.Fatalf("pack %s missing from the lockfile after add", metadataPackName)
	}
	return entry.ContentHash
}

func TestAddCommand_Run_LocalSourceRootGitNotInstalled(t *testing.T) {
	projectDir := setupAddProject(t)
	sourceDir := t.TempDir()
	writeMetadataPackSource(t, sourceDir)
	writeRootGitDirectory(t, sourceDir)

	addLocalPackSource(t, projectDir, sourceDir)

	installed := installedPackDir(projectDir, metadataPackName)
	if _, err := os.Stat(filepath.Join(installed, ".git")); !os.IsNotExist(err) {
		t.Errorf("root .git was copied into the installed pack at %s (stat error = %v), want it absent", installed, err)
	}

	// The authored tree must still be installed, so a copy that dropped
	// everything cannot pass this test.
	for _, rel := range []string{"pack.yml", filepath.Join("rules", "r1.yml"), "zz-last.txt"} {
		if _, err := os.Stat(filepath.Join(installed, rel)); err != nil {
			t.Errorf("authored file %s missing from the installed pack: %v", rel, err)
		}
	}
}

func TestAddCommand_Run_LocalSourceHashMatchesMetadataFreeCopy(t *testing.T) {
	// Two byte-identical authored sources; one is a checkout with its own
	// repository beside it, the other is not.
	withGitSource := t.TempDir()
	writeMetadataPackSource(t, withGitSource)
	writeRootGitDirectory(t, withGitSource)

	metadataFreeSource := t.TempDir()
	writeMetadataPackSource(t, metadataFreeSource)

	withGitHash := lockedHashAfterAdd(t, withGitSource)
	metadataFreeHash := lockedHashAfterAdd(t, metadataFreeSource)

	if withGitHash == "" || metadataFreeHash == "" {
		t.Fatalf("expected non-empty recorded hashes, got withGit=%q metadataFree=%q", withGitHash, metadataFreeHash)
	}
	if withGitHash != metadataFreeHash {
		t.Errorf("a .git-carrying local source recorded a different hash than the identical metadata-free content: withGit=%s metadataFree=%s", withGitHash, metadataFreeHash)
	}
}

func TestAddCommand_Run_LocalSourceNestedGitStillInstalled(t *testing.T) {
	projectDir := setupAddProject(t)
	sourceDir := t.TempDir()
	writeMetadataPackSource(t, sourceDir)

	// NESTED, not root: authored content, deliberately still copied.
	const nestedContent = "[core]\n\tbare = false\n"
	writeFile(t, filepath.Join(sourceDir, "rules", ".git", "config"), nestedContent)

	addLocalPackSource(t, projectDir, sourceDir)

	nestedPath := filepath.Join(installedPackDir(projectDir, metadataPackName), "rules", ".git", "config")
	got, err := os.ReadFile(nestedPath)
	if err != nil {
		t.Fatalf("nested .git was not installed: %v", err)
	}
	if string(got) != nestedContent {
		t.Errorf("installed nested .git content = %q, want %q", string(got), nestedContent)
	}
}

func TestInstallCommand_Run_LocalSourceGitChurnStillVerifies(t *testing.T) {
	projectDir := setupAddProject(t)
	sourceDir := t.TempDir()
	writeMetadataPackSource(t, sourceDir)
	writeRootGitDirectory(t, sourceDir)

	addLocalPackSource(t, projectDir, sourceDir)

	// Install can only materialize a local pack that has a recorded local_path.
	// A run erroring with "no recorded source path" would prove nothing, so this
	// is asserted as a fixture precondition rather than left to chance.
	lf := mustReadLock(t, filepath.Join(projectDir, "backstop.lock"))
	entry, ok := lf.Packs[metadataPackName]
	if !ok {
		t.Fatalf("pack %s missing from the lockfile after add", metadataPackName)
	}
	if entry.LocalPath == "" {
		t.Fatalf("fixture precondition failed: the add recorded no local_path for %s", metadataPackName)
	}

	// Repository churn in the operator's source directory after the add, standing
	// in for reflog and object writes from ordinary git use.
	writeFile(t, filepath.Join(sourceDir, ".git", "logs", "HEAD"), "reflog churn after the add\n")
	writeFile(t, filepath.Join(sourceDir, ".git", "HEAD"), "ref: refs/heads/other\n")
	writeFile(t, filepath.Join(sourceDir, ".git", "objects", "cd", "ef01"), "a new object\n")

	install := newTestInstallCommand(t, defaultTestPackCloner())
	if _, err := install.Run(distribution.InstallOptions{ProjectDir: projectDir}); err != nil {
		t.Fatalf("install failed after repository churn in the local source: %v", err)
	}
}

func TestRelock_LocalPackReproducesAddTimeHash(t *testing.T) {
	projectDir := setupAddProject(t)
	sourceDir := t.TempDir()
	writeMetadataPackSource(t, sourceDir)
	writeRootGitDirectory(t, sourceDir)

	addTimeHash := addLocalPackSource(t, projectDir, sourceDir)
	if addTimeHash == "" {
		t.Fatal("expected a non-empty add-time content hash")
	}

	// Relock takes a filesystem PATH to a directory containing a pack.yml, not a
	// pack name (the ISSUE-074 residual this spec does not touch).
	installed := installedPackDir(projectDir, metadataPackName)
	result, err := distribution.Relock(projectDir, installed)
	if err != nil {
		t.Fatalf("Relock: %v", err)
	}
	if result.ContentHash != addTimeHash {
		t.Errorf("relock recomputed %s, want the add-time hash %s", result.ContentHash, addTimeHash)
	}
}
