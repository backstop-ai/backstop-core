package gate

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/artifact"
)

// plantFile writes an empty file at <root>/<rel>, creating parents. Content is
// irrelevant to the scan — classification is by FILENAME — so the fixtures stay minimal
// and the assertions stay about location.
func plantFile(t *testing.T, root, rel string) string {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("creating %s: %v", filepath.Dir(full), err)
	}
	if err := os.WriteFile(full, []byte("fixture\n"), 0o644); err != nil {
		t.Fatalf("writing %s: %v", full, err)
	}
	return full
}

// realPath is the absolute, symlink-RESOLVED form of a path — the vocabulary
// FindUngatedArtifacts reports in. On macOS t.TempDir() hands back a /var/... path whose
// real location is /private/var/..., so a test that anchors on the unresolved form
// compares two spellings of one directory and fails for a reason that has nothing to do
// with the predicate under test.
func realPath(t *testing.T, path string) string {
	t.Helper()
	abs, err := filepath.Abs(path)
	if err != nil {
		t.Fatalf("absolutizing %s: %v", path, err)
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return abs
	}
	return resolved
}

// ungatedRelPaths renders findings as sorted root-relative slash paths.
func ungatedRelPaths(t *testing.T, projectRoot string, found []UngatedArtifact) []string {
	t.Helper()
	abs := realPath(t, projectRoot)
	out := make([]string, 0, len(found))
	for _, f := range found {
		rel, relErr := filepath.Rel(abs, f.Path)
		if relErr != nil {
			t.Fatalf("relativizing %s against %s: %v", f.Path, abs, relErr)
		}
		out = append(out, filepath.ToSlash(rel))
	}
	sort.Strings(out)
	return out
}

// unconfiguredRootAt resolves the root a project declaring no artifact_root gets.
func unconfiguredRootAt(t *testing.T, dir string) artifact.Root {
	t.Helper()
	root, err := artifact.ResolveRoot(dir, "")
	if err != nil {
		t.Fatalf("resolving an unconfigured root at %s: %v", dir, err)
	}
	return root
}

// TestFindUngatedArtifacts_SurfacesFileOutsideItsExpectedTypeDirectory pins CLM-057.
// ExpectedDir is what makes the report ACTIONABLE rather than merely accusatory, so it
// is asserted, not just the path.
func TestFindUngatedArtifacts_SurfacesFileOutsideItsExpectedTypeDirectory(t *testing.T) {
	project := t.TempDir()
	plantFile(t, project, "docs/SPEC-001-stray.spec.md")
	plantFile(t, project, "specs/SPEC-002-correct.spec.md")

	root := unconfiguredRootAt(t, project)
	found, err := FindUngatedArtifacts(project, root, artifact.NonCorpusDirs{})
	if err != nil {
		t.Fatalf("FindUngatedArtifacts: %v", err)
	}

	got := ungatedRelPaths(t, project, found)
	want := []string{"docs/SPEC-001-stray.spec.md"}
	if len(got) != 1 || got[0] != want[0] {
		t.Fatalf("findings = %v, want exactly %v (the correctly placed spec must NOT be surfaced)", got, want)
	}

	f := found[0]
	if f.Kind != artifact.KindSpec {
		t.Errorf("finding Kind = %q, want %q", f.Kind, artifact.KindSpec)
	}
	wantExpected := filepath.Join(realPath(t, root.Path), "specs")
	if f.ExpectedDir != wantExpected {
		t.Errorf("finding ExpectedDir = %q, want %q — the report must name where discovery WOULD have found it", f.ExpectedDir, wantExpected)
	}
	if f.Root != realPath(t, root.Path) {
		t.Errorf("finding Root = %q, want %q", f.Root, realPath(t, root.Path))
	}
}

// TestFindUngatedArtifacts_SurfacesInsideDefaultRootButWrongTypeDirectory pins CLM-058,
// THE CLAIM THAT SEPARATES A CORRECT IMPLEMENTATION FROM A PLAUSIBLE ONE.
//
// The root is UNCONFIGURED, so it IS the project root, and the bundles under
// .backstop/bundles/ therefore sit INSIDE it — the project root contains itself. A
// root-CONTAINMENT predicate reports nothing here while passing every other test in
// this family. This is the backstop-runtime shape REQ-008 was written for: if this is
// the only red, the predicate is containment rather than per kind.
func TestFindUngatedArtifacts_SurfacesInsideDefaultRootButWrongTypeDirectory(t *testing.T) {
	project := t.TempDir()
	plantFile(t, project, ".backstop/bundles/BUNDLE-001-runtime.bundle.md")
	plantFile(t, project, ".backstop/bundles/BUNDLE-002-runtime.bundle.md")

	root := unconfiguredRootAt(t, project)
	if root.Configured {
		t.Fatal("the fixture root resolved as CONFIGURED; this claim is about the unconfigured default")
	}

	// The premise, asserted so the test cannot pass for the wrong reason: the planted
	// files really are INSIDE the resolved root, which is what defeats containment.
	inside := filepath.Join(project, ".backstop", "bundles", "BUNDLE-001-runtime.bundle.md")
	if !strings.HasPrefix(inside, root.Path+string(filepath.Separator)) {
		t.Fatalf("the planted bundle %q is not inside the resolved root %q, so a containment predicate would ALSO surface it and this test would prove nothing", inside, root.Path)
	}

	found, err := FindUngatedArtifacts(project, root, artifact.NonCorpusDirs{})
	if err != nil {
		t.Fatalf("FindUngatedArtifacts: %v", err)
	}

	got := ungatedRelPaths(t, project, found)
	want := []string{".backstop/bundles/BUNDLE-001-runtime.bundle.md", ".backstop/bundles/BUNDLE-002-runtime.bundle.md"}
	if len(got) != len(want) {
		t.Fatalf("findings = %v, want %v — a root-containment predicate reports NOTHING here", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("finding %d = %q, want %q", i, got[i], want[i])
		}
	}
	wantBundleDir := filepath.Join(realPath(t, root.Path), "bundles")
	for _, f := range found {
		if f.ExpectedDir != wantBundleDir {
			t.Errorf("finding %s carries ExpectedDir %q, want %q", f.Path, f.ExpectedDir, wantBundleDir)
		}
	}
}

// TestFindUngatedArtifacts_SurfacesFileOutsideConfiguredRoot pins CLM-059: the SAME
// per-kind rule covers the configured shape. One rule, two shapes.
func TestFindUngatedArtifacts_SurfacesFileOutsideConfiguredRoot(t *testing.T) {
	project := t.TempDir()
	plantFile(t, project, ".backstop/specs/SPEC-001-correct.spec.md")
	plantFile(t, project, "specs/SPEC-002-outside.spec.md")

	root, err := artifact.ResolveRoot(project, ".backstop")
	if err != nil {
		t.Fatalf("resolving the configured root: %v", err)
	}

	found, err := FindUngatedArtifacts(project, root, artifact.NonCorpusDirs{})
	if err != nil {
		t.Fatalf("FindUngatedArtifacts: %v", err)
	}

	got := ungatedRelPaths(t, project, found)
	if len(got) != 1 || got[0] != "specs/SPEC-002-outside.spec.md" {
		t.Fatalf("findings = %v, want exactly [specs/SPEC-002-outside.spec.md] — the file inside the CONFIGURED root's specs/ is correctly placed", got)
	}
}

// TestFindUngatedArtifacts_CorrectlyPlacedCorpusProducesNoFindings pins CLM-060, and it
// is why backstop-core passes without a framework-exception carve-out: because its
// files are placed correctly, not because it is exempt.
func TestFindUngatedArtifacts_CorrectlyPlacedCorpusProducesNoFindings(t *testing.T) {
	project := t.TempDir()
	root := unconfiguredRootAt(t, project)

	// One correctly placed file of EVERY kind, driven from the shared table rather than
	// a hand-written list, so a new kind cannot silently escape this test.
	for _, kind := range artifact.Kinds() {
		layout, ok := artifact.LayoutFor(kind)
		if !ok {
			t.Fatalf("artifact.LayoutFor(%q) returned ok=false", kind)
		}
		name := "SAMPLE-001-correct" + layout.Extension
		if kind == artifact.KindADR {
			name = "ADR-0001-correct" + layout.Extension
		}
		plantFile(t, project, layout.Directory+"/"+name)
	}

	found, err := FindUngatedArtifacts(project, root, artifact.NonCorpusDirs{})
	if err != nil {
		t.Fatalf("FindUngatedArtifacts: %v", err)
	}
	if len(found) != 0 {
		t.Errorf("a correctly placed corpus produced %d findings: %v", len(found), ungatedRelPaths(t, project, found))
	}
}

// TestFindUngatedArtifacts_ReportsWithoutWideningRootOrTypeDirectories pins CLM-061:
// surfacing REPORTS without ADOPTING. The scanned root and type directories are
// unchanged afterwards, and the surfaced files are not thereby treated as corpus.
func TestFindUngatedArtifacts_ReportsWithoutWideningRootOrTypeDirectories(t *testing.T) {
	project := t.TempDir()
	plantFile(t, project, "docs/SPEC-001-stray.spec.md")

	root := unconfiguredRootAt(t, project)
	rootBefore := root
	specDirBefore := root.Dir(artifact.KindSpec)

	found, err := FindUngatedArtifacts(project, root, artifact.NonCorpusDirs{})
	if err != nil {
		t.Fatalf("FindUngatedArtifacts: %v", err)
	}
	if len(found) == 0 {
		t.Fatal("the scan surfaced nothing, so the no-adoption assertions below would be vacuous")
	}

	if root != rootBefore {
		t.Errorf("the scan mutated the root it was given: %+v -> %+v", rootBefore, root)
	}
	if root.Dir(artifact.KindSpec) != specDirBefore {
		t.Errorf("the scan widened the spec type directory: %q -> %q", specDirBefore, root.Dir(artifact.KindSpec))
	}

	// The surfaced file is NOT adopted as corpus: the non-recursive status walk over
	// the resolved root still resolves zero records from docs/.
	res, err := ResolveArtifactStatus(root.Path)
	if err != nil {
		t.Fatalf("ResolveArtifactStatus: %v", err)
	}
	for _, rec := range res.Records {
		if strings.Contains(rec.Path, "docs") {
			t.Errorf("the surfaced file at %s was ADOPTED into the status corpus; surfacing reports, it does not widen", rec.Path)
		}
	}
}

// TestFindUngatedArtifacts_ExcludesEnumeratedNonCorpusTreesButWalksDotBackstop pins
// CLM-062, and BOTH halves are in one test on purpose (trap B): the exclusions must not
// be so wide that the unconfigured motivating case is excluded away before it can be
// found. `.backstop` itself is WALKED; only `.backstop/packs` beneath it is excluded.
//
// THE SET IS ENUMERATED FROM TWO SOURCES SINCE ISSUE-122, and this fixture spans both:
// `.git`, `testdata` and `prototype` are the TOOL-AGNOSTIC BASE core carries, while
// `vendor` and `node_modules` are ECOSYSTEM NOUNS that arrive INJECTED from installed
// packs' classification.dependency_dirs. Core bakes neither noun; the injection below
// is what a Go or Node toolchain pack declares. The `.backstop/packs` rule is local to
// this walk and unchanged either way.
func TestFindUngatedArtifacts_ExcludesEnumeratedNonCorpusTreesButWalksDotBackstop(t *testing.T) {
	project := t.TempDir()
	excluded := []string{
		".git/SPEC-900-git.spec.md",
		"vendor/SPEC-901-vendor.spec.md",
		"node_modules/SPEC-902-node.spec.md",
		"testdata/SPEC-903-testdata.spec.md",
		"prototype/SPEC-904-prototype.spec.md",
		".backstop/packs/test-org/sample/specs/SPEC-905-pack.spec.md",
	}
	for _, rel := range excluded {
		plantFile(t, project, rel)
	}
	plantFile(t, project, ".backstop/SPEC-906-reachable.spec.md")

	root := unconfiguredRootAt(t, project)
	found, err := FindUngatedArtifacts(project, root, artifact.NewNonCorpusDirs([]string{"vendor", "node_modules"}))
	if err != nil {
		t.Fatalf("FindUngatedArtifacts: %v", err)
	}

	got := ungatedRelPaths(t, project, found)
	for _, rel := range excluded {
		for _, g := range got {
			if g == rel {
				t.Errorf("the scan descended into an ENUMERATED non-corpus tree and surfaced %s", rel)
			}
		}
	}
	reachable := false
	for _, g := range got {
		if g == ".backstop/SPEC-906-reachable.spec.md" {
			reachable = true
		}
	}
	if !reachable {
		t.Errorf("the scan did NOT reach an artifact-shaped file directly under .backstop/; excluding .backstop wholesale removes the unconfigured motivating case before it can be found. findings: %v", got)
	}
}

// TestFindUngatedArtifacts_NestedArtifactIsUngatedButStillDiscovered pins CLM-063.
// UNGATED IS NOT UNDISCOVERED: the status walk is non-recursive (os.ReadDir), so a spec
// nested one level BELOW specs/ genuinely never reaches it, while the recursive CLI
// discovery does reach and schema-validate it.
func TestFindUngatedArtifacts_NestedArtifactIsUngatedButStillDiscovered(t *testing.T) {
	project := t.TempDir()
	plantFile(t, project, "specs/SPEC-001-top.spec.md")
	plantFile(t, project, "specs/archive/SPEC-002-nested.spec.md")
	// A bare SUBDIRECTORY inside a type directory is not itself artifact-shaped and
	// must never be a finding — only FILES' locations are judged.
	if err := os.MkdirAll(filepath.Join(project, "specs", "empty-subdir"), 0o755); err != nil {
		t.Fatalf("creating the empty subdirectory: %v", err)
	}

	root := unconfiguredRootAt(t, project)
	found, err := FindUngatedArtifacts(project, root, artifact.NonCorpusDirs{})
	if err != nil {
		t.Fatalf("FindUngatedArtifacts: %v", err)
	}

	got := ungatedRelPaths(t, project, found)
	if len(got) != 1 || got[0] != "specs/archive/SPEC-002-nested.spec.md" {
		t.Fatalf("findings = %v, want exactly [specs/archive/SPEC-002-nested.spec.md]; a bare subdirectory is not a finding and the top-level spec is correctly placed", got)
	}

	// The non-recursive status walk genuinely does NOT gate the nested file, which is
	// what makes the finding true rather than pedantic.
	res, err := ResolveArtifactStatus(root.Path)
	if err != nil {
		t.Fatalf("ResolveArtifactStatus: %v", err)
	}
	for _, rec := range res.Records {
		if strings.Contains(filepath.ToSlash(rec.Path), "specs/archive/") {
			t.Errorf("the status walk reached the nested spec at %s; if the walk became recursive this finding set would silently empty", rec.Path)
		}
	}

	// The report must describe it as UNGATED, never as undiscovered — CLI discovery
	// walks the resolved root recursively and does reach it.
	for _, f := range found {
		if strings.Contains(strings.ToLower(f.Message()), "undiscovered") || strings.Contains(strings.ToLower(f.Message()), "not discovered") {
			t.Errorf("the finding for %s describes the file as undiscovered; discovery is recursive and DOES reach it. message: %s", f.Path, f.Message())
		}
		if !strings.Contains(strings.ToLower(f.Message()), "ungated") {
			t.Errorf("the finding for %s does not describe the file as ungated: %s", f.Path, f.Message())
		}
	}
}

// TestFindUngatedArtifacts_RelativeAndAbsoluteProjectRootAgree pins CLM-067 and trap C.
// A relative projectRoot compared against an absolute Root.Dir yields either zero
// findings or one per artifact — and BOTH look like a working implementation, which is
// why the two forms are compared rather than each checked alone.
func TestFindUngatedArtifacts_RelativeAndAbsoluteProjectRootAgree(t *testing.T) {
	project := t.TempDir()
	plantFile(t, project, "docs/SPEC-001-stray.spec.md")
	plantFile(t, project, "specs/SPEC-002-correct.spec.md")

	root := unconfiguredRootAt(t, project)

	absFound, err := FindUngatedArtifacts(project, root, artifact.NonCorpusDirs{})
	if err != nil {
		t.Fatalf("FindUngatedArtifacts(absolute): %v", err)
	}

	// Drive the RELATIVE form by chdir'ing into the project and passing ".", which is
	// exactly the shape runGate produces when config-path discovery fails.
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(project); err != nil {
		t.Fatalf("chdir to %s: %v", project, err)
	}
	relFound, relErr := FindUngatedArtifacts(".", root, artifact.NonCorpusDirs{})
	if chErr := os.Chdir(origDir); chErr != nil {
		t.Fatalf("chdir back: %v", chErr)
	}
	if relErr != nil {
		t.Fatalf("FindUngatedArtifacts(relative): %v", relErr)
	}

	if len(absFound) == 0 {
		t.Fatal("the absolute form surfaced nothing, so the agreement below would be vacuous")
	}
	absPaths := ungatedRelPaths(t, project, absFound)
	relPaths := ungatedRelPaths(t, project, relFound)
	if len(absPaths) != len(relPaths) {
		t.Fatalf("the two project-root forms disagree: absolute %v, relative %v", absPaths, relPaths)
	}
	for i := range absPaths {
		if absPaths[i] != relPaths[i] {
			t.Errorf("finding %d differs: absolute %q, relative %q", i, absPaths[i], relPaths[i])
		}
	}
}
