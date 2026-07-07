package main

import (
	"context"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/gate"
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
	dir := newNoStandardsProject(t)

	// Drive the ASSEMBLED gate steps directly over the no-standards project and
	// assert on the returned (GateResult, exitCode). This mirrors
	// TestGateIntegration_ReadOnlyExecution — the sandbox-safe in-process path that
	// bypasses runGate's baseline remote pull and (with NO packs declared) skips
	// pack-engine dispatch, so no external toolchain is invoked.
	scope, err := gate.ComputeGateScope(dir, gate.GateScopeModeAll, nil)
	if err != nil {
		t.Fatalf("compute gate scope: %v", err)
	}
	g := gate.New(gate.WithSteps(buildGateSteps(dir, scope)))
	result, exitCode := g.Run(context.Background())

	// Anti-vacuous guard: the gate must actually have RUN. The original hollow
	// test's sin was that NOTHING ran, so its "no error" was vacuously true.
	if len(result.Steps) == 0 {
		t.Fatalf("gate produced no steps — the gate did not run over the no-standards project")
	}

	// The gate SUCCEEDS on a no-standards project (SPEC-030 CLM-015).
	if exitCode != 0 {
		t.Fatalf("expected gate to succeed on no-standards project (exit 0), got exit=%d; steps=%s", exitCode, summarizeFailedSteps(result))
	}

	// No config error: config loading / pack loading must not fault on a project
	// with no STD-GO-001 artifact and no compiled standards dir.
	for _, step := range result.Steps {
		if step.ConfigErr {
			t.Fatalf("step %q reported a config error on no-standards project: %s", step.StepName, step.Reason)
		}
	}

	// No missing-standard error: no step violation may reference STD-GO-001, a
	// missing standard, or a compiled manifest — the routing tail that once faulted
	// when the standard was absent is gone, so its absence must be silent-success,
	// not a violation.
	for _, step := range result.Steps {
		for _, v := range step.Violations {
			if isMissingStandardViolation(v.Message) {
				t.Fatalf("step %q emitted a missing-standard violation on no-standards project: %q", step.StepName, v.Message)
			}
		}
	}
}

// newNoStandardsProject builds a temp project with NO packs declared, a main.go,
// an empty specs/ dir (no specs → no mandated tests), and an absent
// compiled-standards dir (no STD-GO-001 artifact) — the fixture CLM-015 names.
func newNoStandardsProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "backstop.yml"), []byte("project: no-standards\n"), 0o644); err != nil {
		t.Fatalf("write backstop.yml: %v", err)
	}
	// .backstop exists but rules/ is empty (no compiled standards dir contents).
	if err := os.MkdirAll(filepath.Join(dir, ".backstop"), 0o755); err != nil {
		t.Fatalf("mkdir .backstop: %v", err)
	}
	// An empty specs/ dir: mandated-test extraction finds zero specs and passes,
	// rather than faulting on a missing directory. This keeps the run about the
	// no-standards condition, not an incidental missing-specs error.
	if err := os.MkdirAll(filepath.Join(dir, "specs"), 0o755); err != nil {
		t.Fatalf("mkdir specs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}
	return dir
}

// isMissingStandardViolation reports whether a violation message signals a
// missing-standard / missing-manifest routing fault (the failure mode CLM-015
// guards against).
func isMissingStandardViolation(message string) bool {
	lower := strings.ToLower(message)
	return strings.Contains(lower, "std-go-001") ||
		strings.Contains(lower, "missing standard") ||
		strings.Contains(lower, "manifest")
}

// summarizeFailedSteps renders the failing/config-error steps for a legible
// assertion message.
func summarizeFailedSteps(result gate.GateResult) string {
	var parts []string
	for _, step := range result.Steps {
		if step.Status == "fail" || step.ConfigErr {
			parts = append(parts, step.StepName+"="+step.Status+"("+step.Reason+")")
		}
	}
	if len(parts) == 0 {
		return "(no failed steps)"
	}
	return strings.Join(parts, ", ")
}
