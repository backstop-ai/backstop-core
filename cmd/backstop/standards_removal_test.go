package main

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
}

// TestGate_SucceedsWithoutStandards verifies a project with no STD-GO-001
// artifact and no compiled standards directory scaffolds cleanly (no config
// error, no missing-standard error). (CLM-015)
//
// The former assertion that drove a *realCodeChecker.runCheck was removed by the
// SPEC-040 cutover, and the check.LoadManifest routing tail was removed by
// ISSUE-018 (the in-process routing manifest is deleted along with the code
// check engine).
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
	if _, statErr := os.Stat(filepath.Join(dir, ".backstop")); statErr != nil {
		t.Fatalf(".backstop scaffold missing: %v", statErr)
	}
}
