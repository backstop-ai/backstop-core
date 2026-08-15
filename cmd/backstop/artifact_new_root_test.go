package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/scaffold"
)

// runArtifactNewIn executes `artifact new <type> --slug <slug>` rooted at project.
// BACKSTOP_CONFIG is cleared so config discovery resolves from the injected project
// root rather than from whatever the ambient environment points at.
func runArtifactNewIn(t *testing.T, project, artifactType, slug string) error {
	t.Helper()
	t.Setenv("BACKSTOP_CONFIG", "")

	cmd := newArtifactNewCommandWithDeps(scaffold.ArtifactNewDeps{
		Executor:    &noopGitExecutor{},
		ProjectRoot: project,
		DateFunc:    func() string { return "2026-08-14" },
	})
	cmd.SetArgs([]string{artifactType, "--slug", slug})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	return cmd.Execute()
}

func writeProjectConfig(t *testing.T, project, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(project, "backstop.yml"), []byte(body), 0o644); err != nil {
		t.Fatalf("writing backstop.yml: %v", err)
	}
}

// TestArtifactNew_WritesIntoResolvedRoot pins CLM-050. BOTH halves are asserted — the
// file exists where expected AND no file was created at the project-root location. A
// one-sided assertion passes for an implementation that writes to both.
func TestArtifactNew_WritesIntoResolvedRoot(t *testing.T) {
	project := t.TempDir()
	writeProjectConfig(t, project, "project: artifact-new-configured\nartifact_root: .backstop\n")
	if err := os.MkdirAll(filepath.Join(project, ".backstop"), 0o755); err != nil {
		t.Fatalf("creating .backstop: %v", err)
	}

	if err := runArtifactNewIn(t, project, "spec", "configured-root-sample"); err != nil {
		t.Fatalf("artifact new under a configured root: %v", err)
	}

	configuredDir := filepath.Join(project, ".backstop", "specs")
	entries, err := os.ReadDir(configuredDir)
	if err != nil {
		t.Fatalf("reading %s: %v", configuredDir, err)
	}
	if len(entries) != 1 {
		t.Fatalf("found %d files under the configured root's specs directory, want 1", len(entries))
	}

	projectRootDir := filepath.Join(project, "specs")
	if _, statErr := os.Stat(projectRootDir); statErr == nil {
		t.Errorf("artifact new ALSO created %s; writing to both locations is not a fix", projectRootDir)
	}

	// The control: with NO artifact_root the file lands at the project root exactly as
	// it does today, which is the behavior backstop-core itself depends on.
	unconfigured := t.TempDir()
	writeProjectConfig(t, unconfigured, "project: artifact-new-unconfigured\n")

	if err := runArtifactNewIn(t, unconfigured, "spec", "repo-root-sample"); err != nil {
		t.Fatalf("artifact new under an unconfigured root: %v", err)
	}

	entries, err = os.ReadDir(filepath.Join(unconfigured, "specs"))
	if err != nil {
		t.Fatalf("reading the unconfigured project's specs directory: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("found %d files at the project-root specs directory, want 1", len(entries))
	}
	if _, statErr := os.Stat(filepath.Join(unconfigured, ".backstop", "specs")); statErr == nil {
		t.Error("artifact new created a .backstop/specs directory for a project that configures no artifact root")
	}
}

// TestArtifactNew_IDNumberingContinuesUnderConfiguredRoot is the regression guard for a
// defect the WRITE-path tests above cannot see: ID resolution and the write must agree
// about where the corpus lives.
//
// ResolveID's local-scan fallback counts existing artifacts to pick the next number. It
// used to be handed the PROJECT root while the write used the RESOLVED root, so under a
// configured `.backstop` root the scan read a nonexistent <project>/specs, found
// nothing, and restarted numbering at 001 — silently colliding with the existing
// SPEC-002 rather than continuing past it. Nothing about the file's LOCATION was wrong,
// which is why every location assertion stayed green while the numbering was broken.
//
// The git resolver is unavailable here (noopGitExecutor), so the fallback is the path
// under test rather than an incidental branch.
func TestArtifactNew_IDNumberingContinuesUnderConfiguredRoot(t *testing.T) {
	project := t.TempDir()
	writeProjectConfig(t, project, "project: artifact-new-numbering\nartifact_root: .backstop\n")

	configuredSpecs := filepath.Join(project, ".backstop", "specs")
	if err := os.MkdirAll(configuredSpecs, 0o755); err != nil {
		t.Fatalf("creating the configured specs directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configuredSpecs, "SPEC-002-existing.spec.md"), []byte("x"), 0o644); err != nil {
		t.Fatalf("planting the existing spec: %v", err)
	}

	if err := runArtifactNewIn(t, project, "spec", "next-number"); err != nil {
		t.Fatalf("artifact new under a configured root: %v", err)
	}

	entries, err := os.ReadDir(configuredSpecs)
	if err != nil {
		t.Fatalf("reading %s: %v", configuredSpecs, err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	if len(names) != 2 {
		t.Fatalf("found %d files under the configured specs directory, want 2 (the planted SPEC-002 and the new one): %v", len(names), names)
	}

	var created string
	for _, n := range names {
		if n != "SPEC-002-existing.spec.md" {
			created = n
		}
	}
	if !strings.HasPrefix(created, "SPEC-003-") {
		t.Errorf("the new artifact is %q, want a SPEC-003-* name: the ID scan must count the artifacts under the RESOLVED root, not a project-root directory that does not exist", created)
	}
}

// TestArtifactNew_ConfiguredAbsentRootIsAConfigError pins the other half of the same
// wiring: a declared root that does not exist must surface as a config error rather
// than silently falling back to the project root, which is the silent-wrong-place
// failure CLM-033 forbids one layer down.
func TestArtifactNew_ConfiguredAbsentRootIsAConfigError(t *testing.T) {
	project := t.TempDir()
	writeProjectConfig(t, project, "project: artifact-new-absent-root\nartifact_root: .backstop\n")

	err := runArtifactNewIn(t, project, "spec", "absent-root-sample")
	if err == nil {
		t.Fatal("artifact new succeeded against a configured root that does not exist")
	}
	exitErr, ok := err.(*ExitCodeError)
	if !ok {
		t.Fatalf("error %v (%T) is not an *ExitCodeError", err, err)
	}
	if exitErr.Code != ExitConfigError {
		t.Errorf("exit code = %d, want %d (config error)", exitErr.Code, ExitConfigError)
	}
	if _, statErr := os.Stat(filepath.Join(project, "specs")); statErr == nil {
		t.Error("artifact new fell back to the project root and wrote there anyway")
	}
}
