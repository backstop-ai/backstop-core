package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/pack/distribution"
	"github.com/spf13/cobra"
)

// Coverage tests for the thin CLI adapter commands: each invokes a cobra RunE with
// minimal setup and asserts what the operator actually gets back.
//
// They used to invoke and discard (`_ = cmd.Execute()`), which exercised the path
// without ever failing when it misbehaved — the shape that let `pack add` ship a nil
// dependency and reach production. Every one now asserts on the verdict.

// packCLIProject scaffolds a temp project carrying manifest and makes it the working
// directory for the rest of the test.
func packCLIProject(t *testing.T, manifest string) string {
	t.Helper()

	dir := t.TempDir()
	writeFileForTest(t, dir, "backstop.yml", manifest)
	t.Cleanup(chdirForTest(t, dir))
	return dir
}

// emptyProjectManifest is a project that declares no packs.
const emptyProjectManifest = "project: test\npacks: {}\n"

// runPackCLI executes a constructed command with its output captured, so a test can
// assert on what the operator reads rather than only on the error.
func runPackCLI(t *testing.T, cmd *cobra.Command, args ...string) (string, error) {
	t.Helper()

	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

func TestCLI_PackAdd_NoArgs(t *testing.T) {
	cmd := newPackAddCommand(boolPtr(false))
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for pack add with no args")
	}
}

func TestCLI_PackRemove_NoArgs(t *testing.T) {
	cmd := newPackRemoveCommand(boolPtr(false))
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for pack remove with no args")
	}
}

func TestCLI_PackRemove_NonexistentPack(t *testing.T) {
	packCLIProject(t, emptyProjectManifest)

	_, err := runPackCLI(t, newPackRemoveCommand(boolPtr(false)), "nonexistent/pack")
	if err == nil {
		t.Fatal("removing a pack the project does not declare must fail, not report a silent success")
	}
	if !strings.Contains(err.Error(), "nonexistent/pack") {
		t.Errorf("the diagnostic must name the pack that could not be removed, got: %v", err)
	}
}

func TestCLI_PackInstall_NoLockfile(t *testing.T) {
	packCLIProject(t, emptyProjectManifest)

	_, err := runPackCLI(t, newPackInstallCommand(boolPtr(false)))
	if err == nil {
		t.Fatal("installing with no backstop.lock must fail: there is nothing recorded to restore from")
	}
	if !strings.Contains(err.Error(), "backstop.lock") {
		t.Errorf("the diagnostic must name the missing lock, got: %v", err)
	}
}

func TestCLI_PackUpdate_NoArgs(t *testing.T) {
	cmd := newPackUpdateCommand(boolPtr(false))
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for pack update with no args")
	}
}

func TestCLI_PackUpdate_NonexistentPack(t *testing.T) {
	packCLIProject(t, emptyProjectManifest)

	_, err := runPackCLI(t, newPackUpdateCommand(boolPtr(false)), "nonexistent/pack")
	if err == nil {
		t.Fatal("updating a pack the project does not declare must fail")
	}
	if !strings.Contains(err.Error(), "nonexistent/pack") {
		t.Errorf("the diagnostic must name the pack that could not be updated, got: %v", err)
	}
}

// TestCLI_PackUpdate_LocalPackIsHonestNoOp covers the branch nothing else reaches: a
// pack declared as a local path has no remote to resolve a newer version from, so update
// reports an honest no-op rather than silently doing nothing or claiming an update.
func TestCLI_PackUpdate_LocalPackIsHonestNoOp(t *testing.T) {
	packCLIProject(t, "project: test\npacks:\n  internal/local-rules: local\n")

	out, err := runPackCLI(t, newPackUpdateCommand(boolPtr(false)), "internal/local-rules")
	if err != nil {
		t.Fatalf("updating a local-path pack must be a no-op, not a failure: %v", err)
	}
	if !strings.Contains(out, "internal/local-rules") {
		t.Errorf("the no-op message must name the pack, got: %q", out)
	}
	if !strings.Contains(out, "local") {
		t.Errorf("the no-op message must say why there is nothing to resolve, got: %q", out)
	}
}

// TestCLI_PackUpdate_ResolvesNewerTagInProcess drives a REAL hermetic update through the
// real Cobra command: a genuine clone, a genuine tag listing, a genuine install.
//
// It exists IN PROCESS on purpose. TestE2E_PackUpdate_ResolvesNewerCompatibleTag asserts
// the same behavior through the BUILT binary, which is the stronger proof but runs as a
// subprocess and therefore contributes NOTHING to this package's coverage profile — the
// success branch of pack_update.go read as dead code with an end-to-end test sitting
// right on top of it. This is the in-process twin that closes that gap; neither replaces
// the other.
//
// Three independent assertions make it falsifiable: a resolver that returned the current
// version would print the no-op message instead, a resolver that reported the new version
// without installing it would leave the old manifest on disk, and a lock left at 1.0.0
// would mean the pin was never rewritten.
func TestCLI_PackUpdate_ResolvesNewerTagInProcess(t *testing.T) {
	packName, project := remoteE2ESetup(t, validPackFixture, "v1.0.0", "v1.1.0")
	t.Cleanup(chdirForTest(t, project))

	if out, err := runPackCLI(t, newPackAddCommand(boolPtr(false)), packName+"@1.0.0"); err != nil {
		t.Fatalf("seeding the project with a real add: %v (out: %s)", err, out)
	}

	out, err := runPackCLI(t, newPackUpdateCommand(boolPtr(false)), packName)
	if err != nil {
		t.Fatalf("pack update must resolve and apply the newer compatible tag: %v (out: %s)", err, out)
	}
	if !strings.Contains(out, fmt.Sprintf("Updated %s: 1.0.0 -> 1.1.0", packName)) {
		t.Errorf("expected the update to report 1.0.0 -> 1.1.0, got: %q", out)
	}

	lockfile, lockErr := distribution.ReadLockfile(filepath.Join(project, "backstop.lock"))
	if lockErr != nil {
		t.Fatalf("reading backstop.lock: %v", lockErr)
	}
	if entry := lockfile.Packs[packName]; entry.Version != "1.1.0" {
		t.Errorf("lock entry version = %q after update, want 1.1.0", entry.Version)
	}

	// The newer tag rewrites the manifest's own version, so an update that resolved but
	// reinstalled the old tree is caught here rather than passing on the message alone.
	manifest, readErr := os.ReadFile(filepath.Join(project, ".backstop", "packs", packName, "pack.yml"))
	if readErr != nil {
		t.Fatalf("reading the updated pack manifest: %v", readErr)
	}
	if !strings.Contains(string(manifest), "version: 1.1.0") {
		t.Errorf("the installed pack still carries the old tag's content: %q", string(manifest))
	}
}

func TestCLI_PackUpgrade_NoArgs(t *testing.T) {
	cmd := newPackUpgradeCommand(boolPtr(false))
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for pack upgrade with no args")
	}
}

func TestCLI_PackUpgrade_NonexistentPack(t *testing.T) {
	packCLIProject(t, emptyProjectManifest)

	_, err := runPackCLI(t, newPackUpgradeCommand(boolPtr(false)), "nonexistent/pack@2.0.0")
	if err == nil {
		t.Fatal("upgrading a pack the project does not declare must fail")
	}
	if !strings.Contains(err.Error(), "nonexistent/pack") {
		t.Errorf("the diagnostic must name the pack that could not be upgraded, got: %v", err)
	}
}

// TestCLI_PackAdd_NonexistentPack exercises pack add against a repository that does not
// exist, HERMETICALLY (SPEC-055 REQ-010).
//
// It used to pass `nonexistent/pack@1.0.0` straight through and survive on a bare
// `recover()`, because the unassembled command nil-dereferenced before it ever reached
// git. Now that production assembles a real cloner, the same invocation would clone a
// live URL — so it is redirected at a local path that is not a repository, and the
// recover is gone: there is no longer a panic to catch, and a `recover` here would hide
// the regression if one came back.
//
// This is the MISSING-REPOSITORY case, distinct from the missing-TAG case
// TestE2E_PackAdd_MissingTagDiagnosticNotPanic drives through the built binary.
func TestCLI_PackAdd_NonexistentPack(t *testing.T) {
	// A directory that exists but holds no repository: git fails offline, deterministically.
	redirectPackURL(t, "nonexistent", "pack", t.TempDir())
	packCLIProject(t, emptyProjectManifest)

	_, err := runPackCLI(t, newPackAddCommand(boolPtr(false)), "nonexistent/pack@1.0.0")
	if err == nil {
		t.Fatal("pack add of a repository that does not exist must return an error, not succeed")
	}
	if !strings.Contains(err.Error(), "nonexistent/pack") {
		t.Errorf("the diagnostic must name the pack that could not be cloned, got: %v", err)
	}
}

func TestCLI_PackList_EmptyProject(t *testing.T) {
	packCLIProject(t, emptyProjectManifest)

	out, err := runPackCLI(t, newPackListCommand(boolPtr(false)))
	if err != nil {
		t.Fatalf("listing a project that declares no packs must succeed: %v", err)
	}
	if !strings.Contains(out, "VERSION") {
		t.Errorf("the listing must render its header even when empty, got: %q", out)
	}
}

func TestCLI_PackList_JSON(t *testing.T) {
	packCLIProject(t, emptyProjectManifest)

	jsonFlag := true
	out, err := runPackCLI(t, newPackListCommand(&jsonFlag))
	if err != nil {
		t.Fatalf("listing a project that declares no packs must succeed: %v", err)
	}

	var decoded interface{}
	if decodeErr := json.Unmarshal([]byte(out), &decoded); decodeErr != nil {
		t.Errorf("--json output must parse as JSON, got %q: %v", out, decodeErr)
	}
}

func TestCLI_FormatNewResult_JSON(t *testing.T) {
	f := &JSONArtifactNewFormatter{}
	result := ArtifactNewResult{ArtifactType: "spec", ID: "001", FilePath: "specs/SPEC-001.spec.md"}
	out, err := f.FormatNewResult(result)
	if err != nil {
		t.Fatalf("FormatNewResult JSON: %v", err)
	}
	if out == "" {
		t.Error("expected non-empty JSON output")
	}
}

func TestCLI_FormatNewResult_Human(t *testing.T) {
	f := &HumanArtifactNewFormatter{}
	result := ArtifactNewResult{ArtifactType: "spec", ID: "001", FilePath: "specs/SPEC-001.spec.md"}
	out, err := f.FormatNewResult(result)
	if err != nil {
		t.Fatalf("FormatNewResult human: %v", err)
	}
	if out == "" {
		t.Error("expected non-empty human output")
	}
}

func TestCLI_SpecsExist_NoDir(t *testing.T) {
	if specsExist(filepath.Join(t.TempDir(), "nonexistent")) {
		t.Error("expected false for nonexistent dir")
	}
}

func TestCLI_SpecsExist_DirExists(t *testing.T) {
	dir := t.TempDir()
	writeFileForTest(t, dir, filepath.Join("specs", ".keep"), "")
	if !specsExist(filepath.Join(dir, "specs")) {
		t.Error("expected true when specs/ exists")
	}
}

func TestCLI_SpecsExist_FileNotDir(t *testing.T) {
	dir := t.TempDir()
	writeFileForTest(t, dir, "specs", "not a dir")
	if specsExist(filepath.Join(dir, "specs")) {
		t.Error("expected false when specs is a file not a directory")
	}
}

func boolPtr(b bool) *bool { return &b }
