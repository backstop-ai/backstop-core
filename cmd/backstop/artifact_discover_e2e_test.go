package main

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/artifact"
)

// artifact_discover_e2e_test.go is ISSUE-122's regression proof: the artifact-corpus
// exclusion of ecosystem dependency directories comes from a PACK DECLARATION, not from
// a literal baked into core.
//
// Every test here builds the exclusion set THE WAY PRODUCTION DOES — loadInstalledPacks
// + mergeDependencyDirs over the fixture project root — rather than hand-constructing
// artifact.NewNonCorpusDirs. Hand-constructing it would prove the walk and skip the
// merge, and the merge is half of what this change is.

// noncorpusFixtureRoot resolves the ISSUE-122 fixture project.
func noncorpusFixtureRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("testdata", "noncorpus-declared"))
	if err != nil {
		t.Fatalf("resolving the noncorpus-declared fixture root: %v", err)
	}
	return root
}

// productionExclusionSet builds the exclusion set through the SAME two calls
// buildGateSteps, `artifact validate` and doctor each make.
func productionExclusionSet(t *testing.T, projectRoot string) artifact.NonCorpusDirs {
	t.Helper()
	packs, err := loadInstalledPacks(projectRoot)
	if err != nil {
		t.Fatalf("loadInstalledPacks(%s): %v", projectRoot, err)
	}
	declared := mergeDependencyDirs(packs)
	if len(declared) == 0 {
		t.Fatalf("the fixture pack declared no dependency_dirs; the exclusion under test would be vacuous")
	}
	return artifact.NewNonCorpusDirs(declared)
}

// discoverFixtureRelPaths runs discovery over projectRoot with the given exclusion set
// and returns sorted root-relative slash paths.
func discoverFixtureRelPaths(t *testing.T, projectRoot string, set artifact.NonCorpusDirs) []string {
	t.Helper()
	arts, err := DiscoverArtifacts(rootAtDir(t, projectRoot), nil, set)
	if err != nil {
		t.Fatalf("DiscoverArtifacts over %s: %v", projectRoot, err)
	}
	out := make([]string, 0, len(arts))
	for _, a := range arts {
		rel, relErr := filepath.Rel(projectRoot, a.Path)
		if relErr != nil {
			t.Fatalf("relativizing %s against %s: %v", a.Path, projectRoot, relErr)
		}
		out = append(out, filepath.ToSlash(rel))
	}
	sort.Strings(out)
	return out
}

// the artifact-shaped files planted inside declared dependency trees.
func plantedDependencyTreeArtifacts() []string {
	return []string{
		"vendor/SPEC-902-vendored-citer.spec.md",
		"node_modules/some-dep/ISSUE-902-node-dependency.issue.md",
		"pkg/sub/vendor/SPEC-903-nested-vendored.spec.md",
		"_thirdparty_deps/SPEC-904-invented-ecosystem.spec.md",
	}
}

func assertAbsent(t *testing.T, got []string, unwanted ...string) {
	t.Helper()
	for _, u := range unwanted {
		for _, g := range got {
			if g == u {
				t.Errorf("discovery returned %s, which sits inside a pack-declared dependency tree. discovered: %v", u, got)
			}
		}
	}
}

func assertPresent(t *testing.T, got []string, wanted ...string) {
	t.Helper()
	for _, w := range wanted {
		found := false
		for _, g := range got {
			if g == w {
				found = true
			}
		}
		if !found {
			t.Errorf("discovery did NOT return %s. discovered: %v", w, got)
		}
	}
}

// TestArtifactDiscoveryE2E_VendorAndNodeModulesExcludedByPackDeclaration (CLM-007):
// discovery over the fixture returns the legitimate corpus and NONE of the planted
// dependency-tree files.
func TestArtifactDiscoveryE2E_VendorAndNodeModulesExcludedByPackDeclaration(t *testing.T) {
	projectRoot := noncorpusFixtureRoot(t)
	got := discoverFixtureRelPaths(t, projectRoot, productionExclusionSet(t, projectRoot))

	assertPresent(t, got, "specs/SPEC-901-legitimate-corpus.spec.md")
	assertAbsent(t, got, plantedDependencyTreeArtifacts()...)
	if len(got) != 1 {
		t.Errorf("discovery returned %d artifacts, want exactly the 1 legitimate corpus file: %v", len(got), got)
	}
}

// TestArtifactDiscoveryE2E_ExclusionDisappearsWhenDeclarationRemoved (CLM-007): THE
// FALSIFIER. Copy the fixture, strip `dependency_dirs`, re-run discovery, and assert
// every planted file IS discovered. If a `vendor`/`node_modules` default survives
// anywhere in core, this test goes red — which is precisely its job.
//
// IT STRIPS THE INSTALLED COPY. loadInstalledPacks resolves each pack declared in
// backstop.yml to <projectRoot>/.backstop/packs/<name>/pack.yml and parses THAT; it
// never reads a pack's source tree. Editing any other manifest would leave the
// installed one still declaring `vendor`, the exclusion would still happen, and the
// falsifier would prove nothing while appearing to pass. The exact path edited is
// named below so a reviewer can see which manifest was touched.
func TestArtifactDiscoveryE2E_ExclusionDisappearsWhenDeclarationRemoved(t *testing.T) {
	projectRoot := copyFixtureToTemp(t, noncorpusFixtureRoot(t))

	installedManifest := filepath.Join(projectRoot, ".backstop", "packs", "backstop", "noncorpus-fixture", "pack.yml")
	stripDependencyDirs(t, installedManifest)

	packs, err := loadInstalledPacks(projectRoot)
	if err != nil {
		t.Fatalf("loadInstalledPacks after stripping the declaration: %v", err)
	}
	if declared := mergeDependencyDirs(packs); len(declared) != 0 {
		t.Fatalf("the declaration survived the strip of %s: still %v — the falsifier would prove nothing", installedManifest, declared)
	}

	got := discoverFixtureRelPaths(t, projectRoot, artifact.NewNonCorpusDirs(mergeDependencyDirs(packs)))

	assertPresent(t, got, "specs/SPEC-901-legitimate-corpus.spec.md")
	assertPresent(t, got, plantedDependencyTreeArtifacts()...)
}

// TestArtifactDiscoveryE2E_NestedDependencyDirExcludedAndFileNamedVendorUnaffected
// (CLM-007): the match is on the directory BASE NAME at any depth, and a plain FILE
// named `vendor` neither triggers a skip of its parent nor disappears. This is the
// "scoped wrong" guard — an implementation matching a root-relative path, or matching
// files as well as directories, fails here and nowhere else.
func TestArtifactDiscoveryE2E_NestedDependencyDirExcludedAndFileNamedVendorUnaffected(t *testing.T) {
	projectRoot := noncorpusFixtureRoot(t)

	// The premise, asserted so the test cannot pass for the wrong reason.
	vendorFile := filepath.Join(projectRoot, "specs", "vendor")
	info, statErr := os.Stat(vendorFile)
	if statErr != nil {
		t.Fatalf("the fixture's plain file named `vendor` is missing at %s: %v", vendorFile, statErr)
	}
	if info.IsDir() {
		t.Fatalf("%s is a DIRECTORY; this test needs a plain FILE there or it asserts nothing", vendorFile)
	}

	got := discoverFixtureRelPaths(t, projectRoot, productionExclusionSet(t, projectRoot))

	// Nested several levels deep, still excluded.
	assertAbsent(t, got, "pkg/sub/vendor/SPEC-903-nested-vendored.spec.md")
	// The plain file named `vendor` did not cause its parent specs/ to be skipped.
	assertPresent(t, got, "specs/SPEC-901-legitimate-corpus.spec.md")
}

// TestArtifactDiscoveryE2E_InventedEcosystemDirHonoredWithNoCoreEdit (CLM-008): the
// ECOSYSTEM-AGNOSTICISM proof. The fixture pack declares `_thirdparty_deps`, a name
// that appears in NO core source file, and a planted artifact under it is excluded.
// Adding the next ecosystem's vendoring convention is therefore a pack change, never a
// `case` in cmd/backstop.
//
// The absence of the name from core is ASSERTED here, not assumed: a name that happened
// to collide with something core already knows would make this test vacuous.
func TestArtifactDiscoveryE2E_InventedEcosystemDirHonoredWithNoCoreEdit(t *testing.T) {
	const invented = "_thirdparty_deps"

	assertNameAbsentFromCoreSources(t, invented)

	projectRoot := noncorpusFixtureRoot(t)
	packs, err := loadInstalledPacks(projectRoot)
	if err != nil {
		t.Fatalf("loadInstalledPacks: %v", err)
	}
	declaredInvented := false
	for _, name := range mergeDependencyDirs(packs) {
		if name == invented {
			declaredInvented = true
		}
	}
	if !declaredInvented {
		t.Fatalf("the fixture pack does not declare %q, so this test cannot prove CLM-008", invented)
	}

	got := discoverFixtureRelPaths(t, projectRoot, productionExclusionSet(t, projectRoot))
	assertAbsent(t, got, "_thirdparty_deps/SPEC-904-invented-ecosystem.spec.md")
}

// assertNameAbsentFromCoreSources walks the core Go sources under ../../pkg and
// ../../cmd and fails if name appears in any non-test, non-testdata file.
func assertNameAbsentFromCoreSources(t *testing.T, name string) {
	t.Helper()
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolving the repository root: %v", err)
	}
	for _, dir := range []string{"pkg", "cmd"} {
		walkErr := filepath.Walk(filepath.Join(repoRoot, dir), func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				if info.Name() == "testdata" {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			body, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			if strings.Contains(string(body), name) {
				rel, _ := filepath.Rel(repoRoot, path)
				t.Errorf("%q appears in core source %s; it is supposed to be a name core has never heard of, so this test would be vacuous", name, rel)
			}
			return nil
		})
		if walkErr != nil {
			t.Fatalf("walking %s: %v", dir, walkErr)
		}
	}
}

// copyFixtureToTemp copies a fixture project into a fresh temp dir so a test may edit
// it without mutating the committed fixture.
func copyFixtureToTemp(t *testing.T, src string) string {
	t.Helper()
	dst := t.TempDir()
	err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, relErr := filepath.Rel(src, path)
		if relErr != nil {
			return relErr
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		body, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		return os.WriteFile(target, body, 0o644)
	})
	if err != nil {
		t.Fatalf("copying fixture %s: %v", src, err)
	}
	return dst
}

// stripDependencyDirs removes the `dependency_dirs:` key and its list items from a
// pack manifest, leaving the rest of the `classification:` block intact.
func stripDependencyDirs(t *testing.T, manifestPath string) {
	t.Helper()
	body, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("reading %s: %v", manifestPath, err)
	}

	var kept []string
	inBlock := false
	for _, line := range strings.Split(string(body), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "dependency_dirs:") {
			inBlock = true
			continue
		}
		if inBlock {
			// The block's items are the immediately following `- ` entries.
			if strings.HasPrefix(trimmed, "- ") {
				continue
			}
			inBlock = false
		}
		kept = append(kept, line)
	}

	stripped := strings.Join(kept, "\n")
	if strings.Contains(stripped, "dependency_dirs") {
		t.Fatalf("stripping dependency_dirs from %s left the key behind:\n%s", manifestPath, stripped)
	}
	if err := os.WriteFile(manifestPath, []byte(stripped), 0o644); err != nil {
		t.Fatalf("writing the stripped manifest %s: %v", manifestPath, err)
	}
}
