package scaffold

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/artifact"
)

// TestTargetDir_PlacesEveryKindUnderResolvedRoot pins CLM-049. It ranges over
// artifact.Kinds() rather than a hand-written type list, so a kind added to the shared
// table without a scaffold entry goes red here.
//
// TargetDir's SECOND PARAMETER IS an artifact.Root, not a bare projectRoot string, so
// a caller cannot accidentally hand it the project root where the artifact root
// belongs. The compile break is the point.
func TestTargetDir_PlacesEveryKindUnderResolvedRoot(t *testing.T) {
	project := t.TempDir()
	if err := os.MkdirAll(filepath.Join(project, ".backstop"), 0o755); err != nil {
		t.Fatalf("creating .backstop: %v", err)
	}

	configured, err := artifact.ResolveRoot(project, ".backstop")
	if err != nil {
		t.Fatalf("ResolveRoot(configured): %v", err)
	}

	for _, kind := range artifact.Kinds() {
		artifactType := string(kind)
		got := TargetDir(artifactType, configured)
		want := configured.Dir(kind)
		if got != want {
			t.Errorf("TargetDir(%q, <configured root>) = %q, want %q", artifactType, got, want)
		}
		if got == filepath.Join(project, string(kind)) {
			t.Errorf("TargetDir(%q) placed the artifact at the PROJECT root %q rather than the configured artifact root", artifactType, got)
		}
	}

	// The control: an UNCONFIGURED root keeps the repo-root placement backstop-core
	// itself depends on.
	unconfigured, err := artifact.ResolveRoot(project, "")
	if err != nil {
		t.Fatalf("ResolveRoot(unconfigured): %v", err)
	}
	for _, kind := range artifact.Kinds() {
		artifactType := string(kind)
		layout, ok := artifact.LayoutFor(kind)
		if !ok {
			t.Fatalf("artifact.LayoutFor(%q) returned ok=false", kind)
		}
		got := TargetDir(artifactType, unconfigured)
		want := filepath.Join(unconfigured.Path, layout.Directory)
		if got != want {
			t.Errorf("TargetDir(%q, <unconfigured root>) = %q, want %q — the repo-root layout must be unchanged", artifactType, got, want)
		}
	}
}

// TestArtifactTypeFor_UnknownTypeReportsNotFound covers the lookup's negative arm.
//
// ArtifactTypeFor replaced an exported package-level map, and a map index silently
// yields a ZERO config for an unknown key — which is how an unrecognized artifact type
// used to produce a filename with an empty prefix and no extension instead of an error.
// The boolean is what makes that unrepresentable, so every caller's not-found branch is
// exercised here.
func TestArtifactTypeFor_UnknownTypeReportsNotFound(t *testing.T) {
	cfg, ok := ArtifactTypeFor("no-such-type")
	if ok {
		t.Fatalf("ArtifactTypeFor(%q) reported the type as known", "no-such-type")
	}
	if cfg.IDPrefix != "" || cfg.DigitCount != 0 || cfg.DefaultStatus != "" || cfg.FileExtension != "" || cfg.BodySections != nil {
		t.Errorf("ArtifactTypeFor returned a populated config for an unknown type: %+v", cfg)
	}

	// Every known type resolves, driven from the shared kind table so a kind added
	// there without a scaffold entry goes red here rather than silently.
	for _, kind := range artifact.Kinds() {
		if _, known := ArtifactTypeFor(string(kind)); !known {
			t.Errorf("ArtifactTypeFor(%q) reported a shared-table kind as unknown", kind)
		}
	}

	// Filename refuses rather than composing a name out of a zero config.
	if got := Filename("no-such-type", "001", "slug", ""); got != "" {
		t.Errorf("Filename for an unknown type = %q, want the empty string; a zero config would compose %q-shaped garbage", got, "-001-slug")
	}

	// Scaffold errors, naming the type.
	if _, err := Scaffold("no-such-type", "001", "slug", "2026-08-14", ""); err == nil {
		t.Error("Scaffold accepted an unknown artifact type")
	}
}

// TestLocalScanResolver_UnknownTypeErrors covers the ID resolver's not-found arm. It
// used to read a zero config and resolve an ID with a zero digit count, which formats
// as an unpadded number — a silently wrong ID rather than a refusal.
func TestLocalScanResolver_UnknownTypeErrors(t *testing.T) {
	r := &LocalScanResolver{}
	if _, err := r.Resolve("no-such-type", t.TempDir()); err == nil {
		t.Error("LocalScanResolver.Resolve accepted an unknown artifact type")
	}

	// The control: a known type over an empty directory resolves the first ID, so the
	// refusal above is attributable to the type and not to the directory.
	id, err := r.Resolve("spec", filepath.Join(t.TempDir(), "absent"))
	if err != nil {
		t.Fatalf("resolving the first ID for a known type: %v", err)
	}
	if id != "001" {
		t.Errorf("first spec ID = %q, want %q", id, "001")
	}
}

// TestGitTagResolver_UnknownTypeErrorsBeforeReachingGit covers the tag resolver's
// not-found arm. It refuses on the TYPE before it reads any tag, so an unrecognized
// type can never reach the digit-count formatting with a zero config.
func TestGitTagResolver_UnknownTypeErrorsBeforeReachingGit(t *testing.T) {
	// Assigned rather than composed in a keyed literal: `isRepo` matches the
	// constructor-injection rule's dependency-field regex, and this mock is test
	// scaffolding rather than production wiring.
	mock := &mockGitExecutor{}
	mock.isRepo = true
	mock.isAvailable = true

	resolver := &GitTagResolver{}
	resolver.executor = mock
	resolver.maxRetries = 3

	if _, err := resolver.Resolve("no-such-type", "some-slug"); err == nil {
		t.Error("GitTagResolver.Resolve accepted an unknown artifact type")
	} else if isFallbackError(err) {
		t.Errorf("an unknown artifact type was reported as a FALLBACK (an offline/git condition) rather than a bad type: %v", err)
	}
}
