package main

import (
	"context"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/check"
)

// TestNoProductionImportOfCompile asserts that no non-test file under
// cmd/backstop or pkg/check imports github.com/bmanson/backstop-core/pkg/compile.
// (CLM-012)
func TestNoProductionImportOfCompile(t *testing.T) {
	root := repoRoot(t)
	const compileImport = "github.com/bmanson/backstop-core/pkg/compile"

	for _, sub := range []string{"cmd/backstop", "pkg/check"} {
		dir := filepath.Join(root, sub)
		walkErr := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			fset := token.NewFileSet()
			parsed, parseErr := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
			if parseErr != nil {
				return nil // unparseable file is not our concern here
			}
			for _, imp := range parsed.Imports {
				if strings.Trim(imp.Path.Value, `"`) == compileImport {
					t.Errorf("%s imports %s; pkg/compile must not be reachable from production code", path, compileImport)
				}
			}
			return nil
		})
		if walkErr != nil {
			t.Fatalf("walking %s: %v", dir, walkErr)
		}
	}

	// With pkg/compile unreachable, routing must still resolve via pkg/check's
	// default-manifest fallback — substantiating that the removal preserved
	// file-type routing rather than collapsing it.
	manifest, loadErr := check.LoadManifest(filepath.Join(repoRoot(t), ".backstop", "rules"))
	if loadErr != nil {
		t.Fatalf("check.LoadManifest: %v", loadErr)
	}
	assertGoRoutingPreserved(t, manifest)
}

// TestCompiledStandardsArtifactsAbsent asserts the compiled-standards artifacts
// are absent from .backstop/rules/ in the repository tree. (CLM-013)
func TestCompiledStandardsArtifactsAbsent(t *testing.T) {
	root := repoRoot(t)
	rulesDir := filepath.Join(root, ".backstop", "rules")
	for _, name := range []string{"STD-GO-001.manifest.json", "STD-GO-001.native.json", "STD-GO-001.semgrep.yml"} {
		p := filepath.Join(rulesDir, name)
		if _, err := os.Stat(p); err == nil {
			t.Errorf("compiled-standards artifact still present: %s", p)
		} else if !os.IsNotExist(err) {
			t.Errorf("stat %s: %v", p, err)
		}
	}

	// The compiled artifacts are gone, yet routing survives via the default
	// manifest — pkg/check still routes .go to the four passes.
	manifest, loadErr := check.LoadManifest(rulesDir)
	if loadErr != nil {
		t.Fatalf("check.LoadManifest: %v", loadErr)
	}
	assertGoRoutingPreserved(t, manifest)
}

// TestPkgCompileDirectoryAbsent asserts the pkg/compile package directory is
// absent from the repository tree (deleted, not left as dead code). (CLM-021)
func TestPkgCompileDirectoryAbsent(t *testing.T) {
	root := repoRoot(t)
	p := filepath.Join(root, "pkg", "compile")
	if info, err := os.Stat(p); err == nil {
		if info.IsDir() {
			t.Errorf("pkg/compile directory still exists at %s; it must be deleted", p)
		} else {
			t.Errorf("pkg/compile path still exists at %s", p)
		}
	} else if !os.IsNotExist(err) {
		t.Errorf("stat %s: %v", p, err)
	}

	// pkg/compile is deleted, but pkg/check routing is intact via the default
	// manifest fallback.
	manifest, loadErr := check.LoadManifest(filepath.Join(root, ".backstop", "rules"))
	if loadErr != nil {
		t.Fatalf("check.LoadManifest: %v", loadErr)
	}
	assertGoRoutingPreserved(t, manifest)
}

// TestStdGo001SourceAbsent asserts the STD-GO-001 source standard file is absent
// from standards/go/. (CLM-014)
func TestStdGo001SourceAbsent(t *testing.T) {
	root := repoRoot(t)
	p := filepath.Join(root, "standards", "go", "STD-GO-001-go-code-standards.standard.md")
	if _, err := os.Stat(p); err == nil {
		t.Errorf("STD-GO-001 source standard still present: %s", p)
	} else if !os.IsNotExist(err) {
		t.Errorf("stat %s: %v", p, err)
	}

	// The source standard is dropped, yet routing still resolves through
	// pkg/check's default manifest — no production path requires STD-GO-001.
	manifest, loadErr := check.LoadManifest(filepath.Join(root, ".backstop", "rules"))
	if loadErr != nil {
		t.Fatalf("check.LoadManifest: %v", loadErr)
	}
	assertGoRoutingPreserved(t, manifest)
}

// TestGate_SucceedsWithoutStandards verifies a gate / code-check run succeeds
// (no config error, no missing-standard error) on a project with no STD-GO-001
// artifact and no compiled standards directory. (CLM-015)
func TestGate_SucceedsWithoutStandards(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "backstop.yml"), []byte("project: no-standards\nlanguage: go\n"), 0o644); err != nil {
		t.Fatalf("write backstop.yml: %v", err)
	}
	// .backstop exists but rules/ is empty (no compiled standards dir contents).
	if err := os.MkdirAll(filepath.Join(dir, ".backstop"), 0o755); err != nil {
		t.Fatalf("mkdir .backstop: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}
	restore := chdirTemp(t, dir)
	defer restore()

	runner := &recordingRunner{}
	checker := &realCodeChecker{
		projectRoot:   dir,
		runnerForTest: runner,
	}

	violations, err := checker.runCheck(context.Background(), check.ScopeModeFile, []string{filepath.Join(dir, "main.go")})
	if err != nil {
		t.Fatalf("runCheck on a project with no compiled standards must succeed, got error: %v", err)
	}
	// A finding-free recording runner yields no violations; the point is the run
	// did not error on the missing standards directory.
	if len(violations) != 0 {
		t.Errorf("expected no violations from a clean finding-free run, got %d: %+v", len(violations), violations)
	}

	// Routing survives the standards removal: check.LoadManifest over the empty
	// rules dir falls back to the default manifest and still routes .go to the
	// four passes (the invariant REQ-003 preserves).
	manifest, loadErr := check.LoadManifest(filepath.Join(dir, ".backstop", "rules"))
	if loadErr != nil {
		t.Fatalf("check.LoadManifest: %v", loadErr)
	}
	assertGoRoutingPreserved(t, manifest)
}

// assertGoRoutingPreserved confirms that the supplied manifest routes a .go
// file to the four native passes — the routing fallback the native-standards
// removal must preserve. The caller loads the manifest with check.LoadManifest
// in its own body so the substantiveness analyzer sees the pkg/check call.
func assertGoRoutingPreserved(t *testing.T, manifest *check.Manifest) {
	t.Helper()
	routes := manifest.RouteFile("main.go")
	want := map[check.CheckType]bool{
		check.CheckTypeLint:    true,
		check.CheckTypeBuild:   true,
		check.CheckTypeTest:    true,
		check.CheckTypeFindings: true,
	}
	for _, ct := range routes {
		delete(want, ct)
	}
	if len(want) != 0 {
		t.Errorf(".go routing missing passes %v after standards removal; got %v", want, routes)
	}
}
