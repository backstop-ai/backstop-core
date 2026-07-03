package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/check"
	"github.com/bmanson/backstop-core/pkg/pack"
)

// SPEC-048 REQ-003 — THE LOAD-BEARING PROOF. A REAL end-to-end gate over the
// committed installed-pack fixture (cmd/backstop/testdata/bun-dispatch-e2e), driving
// the ACTUAL dispatch path (runFindingsEngine via dispatchPackEngines) with a REAL
// check.CommandRunner (check.ExecCommandRunner{Dir: projectRoot}) executing the
// committed POSIX fake-engine.sh — NOT a stubbed dispatcher and NOT the canned-stdout
// fixtureRunner that masked BOTH defects in SPEC-043..047. The fake self-targets,
// writes its findings to its declared stdout_artifact FILE, and prints noise to
// stdout, so this ONE test exercises BOTH fixes on the real path and FAILS against
// the pre-fix dispatch. No real bun/tsc/oxlint/go runs in Go CI.

// bunDispatchE2EFixtureDir is the committed installed-pack fixture root.
func bunDispatchE2EFixtureDir(t *testing.T) string {
	t.Helper()
	return filepath.Join(repoRoot(t), "cmd", "backstop", "testdata", "bun-dispatch-e2e")
}

// bunDispatchE2EManifest loads the committed fixture manifest and rewrites ONLY the
// engine command to the absolute path of the committed fake-engine.sh (a committed
// POSIX script cannot carry a machine-specific absolute path in its DATA, so the
// test resolves it). The self-targeting DATA under test — scope_kind project-wide,
// EMPTY project_target, declared stdout_artifact — stays authoritative from the
// committed pack.yml. Returns the manifest, the .backstop/packs dir, and the
// resolved script path.
func bunDispatchE2EManifest(t *testing.T) (*pack.Manifest, string, string) {
	t.Helper()
	fixtureDir := bunDispatchE2EFixtureDir(t)
	packsDir := filepath.Join(fixtureDir, ".backstop", "packs")
	packRoot := filepath.Join(packsDir, "backstop", "bun-dispatch-e2e")
	m, err := pack.ParseManifestFile(filepath.Join(packRoot, "pack.yml"))
	if err != nil {
		t.Fatalf("committed fixture manifest must parse: %v", err)
	}
	script := filepath.Join(packRoot, "scripts", "fake-engine.sh")
	if _, statErr := os.Stat(script); statErr != nil {
		t.Fatalf("committed fake-engine.sh must exist: %v", statErr)
	}
	spec := m.Engines["fake-engine"]
	// Guard the self-targeting DATA contract the e2e depends on (it is the point).
	if spec.Binding.ProjectTarget != "" {
		t.Fatalf("fixture engine must declare an EMPTY project_target (self-target), got %q", spec.Binding.ProjectTarget)
	}
	if spec.Binding.StdoutArtifact == "" {
		t.Fatal("fixture engine must declare a stdout_artifact (the FILE it writes findings to)")
	}
	spec.Binding.Command = script
	m.Engines["fake-engine"] = spec
	return m, packsDir, script
}

// bunDispatchE2EProjectRoot builds a fresh temp project the fake engine self-targets
// (its cwd). Its src/app.ts carries the SEEDED_DEFECT marker only when seeded, so the
// self-targeting fake reports a finding for the seeded variant and nothing for the
// clean one — mirroring the committed-clean vs seeded pattern without mutating the
// committed fixture.
func bunDispatchE2EProjectRoot(t *testing.T, seeded bool) string {
	t.Helper()
	projectRoot := t.TempDir()
	mkDirAll(t, filepath.Join(projectRoot, "src"))
	src := "export function greet(name: string): string {\n  return `hi ${name}`;\n}\n"
	if seeded {
		src += "// SEEDED_DEFECT: a seeded finding the self-targeting engine must report\n"
	}
	writeFileStr(t, filepath.Join(projectRoot, "src", "app.ts"), src)
	return projectRoot
}

// TestBunDispatchE2E_SeededFindingInStdoutArtifactRedsGate proves the seeded variant
// REDs over the REAL dispatch (CLM-009): the fake writes a finding to its
// stdout_artifact FILE while its stdout is noise, and the gate carries that finding
// (not vacuous green). Fails pre-fix — a bolted-on projectRoot arg suppresses the
// finding (DEFECT-1) and a stdout-fed convert sees no findings (DEFECT-2).
func TestBunDispatchE2E_SeededFindingInStdoutArtifactRedsGate(t *testing.T) {
	m, packsDir, _ := bunDispatchE2EManifest(t)
	projectRoot := bunDispatchE2EProjectRoot(t, true)
	runner := &check.ExecCommandRunner{Dir: projectRoot}

	violations, err := dispatchPackEngines([]*pack.Manifest{m}, packsDir, projectRoot, nil, runner)
	if err != nil {
		t.Fatalf("the seeded real-dispatch run must RED with a finding, not error out: %v", err)
	}
	if len(violations) != 1 {
		t.Fatalf("the seeded fixture must RED the gate carrying exactly the artifact's one finding, got %d: %#v", len(violations), violations)
	}
	if !strings.HasPrefix(violations[0].Rule, "backstop/bun-dispatch-e2e/") {
		t.Errorf("the finding must be namespaced to the pack, got %q", violations[0].Rule)
	}
	if !strings.Contains(violations[0].Message, "SEEDED_DEFECT") {
		t.Errorf("the RED must carry the seeded finding from the artifact FILE, got %#v", violations[0])
	}
	// Self-target proof on the real path: the dispatch appended NO scan target.
	if argc := strings.TrimSpace(readFileStr(t, filepath.Join(projectRoot, "argc.txt"))); argc != "0" {
		t.Errorf("DEFECT-1: the self-targeting engine must receive NO appended arg, argc=%q", argc)
	}
}

// TestBunDispatchE2E_CleanFixtureGreenGate proves the SAME fixture with NO seeded
// marker greens over the real dispatch (CLM-010) — for the RIGHT reason (Sharp Edge
// 2): the fake self-targeted and genuinely found nothing (argc==0), not because a
// stray arg made it scan nothing. This proves the CLM-009 RED is the finding and not
// a spurious dispatch failure.
func TestBunDispatchE2E_CleanFixtureGreenGate(t *testing.T) {
	m, packsDir, _ := bunDispatchE2EManifest(t)
	projectRoot := bunDispatchE2EProjectRoot(t, false)
	runner := &check.ExecCommandRunner{Dir: projectRoot}

	violations, err := dispatchPackEngines([]*pack.Manifest{m}, packsDir, projectRoot, nil, runner)
	if err != nil {
		t.Fatalf("the clean real-dispatch run must GREEN, not error out: %v", err)
	}
	if len(violations) != 0 {
		t.Fatalf("the clean fixture must GREEN over the real dispatch, got %d violations: %#v", len(violations), violations)
	}
	// GREEN for the RIGHT reason: the fake self-targeted (argc==0) and found nothing,
	// NOT because a bolted-on arg silenced it.
	if argc := strings.TrimSpace(readFileStr(t, filepath.Join(projectRoot, "argc.txt"))); argc != "0" {
		t.Errorf("the clean GREEN must come from a self-targeting scan (argc==0), got argc=%q", argc)
	}
}

// TestBunDispatchE2E_RealRunnerFakeEngineSelfTargetsAndReadsArtifactCatchesBothBugs is
// the capstone (CLM-011): the run uses a REAL check.ExecCommandRunner (NO stub, NO
// canned-stdout fixtureRunner) executing a committed POSIX fake (NO real
// bun/tsc/oxlint/go); the fake SELF-TARGETS (argc==0 — a bolted-on projectRoot would
// suppress even the seeded finding, exposing DEFECT-1); and findings come from the
// stdout_artifact FILE while stdout is human noise (a stdout-fed convert would expose
// DEFECT-2). This ONE test exercises BOTH fixes on the real path and FAILS pre-fix.
func TestBunDispatchE2E_RealRunnerFakeEngineSelfTargetsAndReadsArtifactCatchesBothBugs(t *testing.T) {
	m, packsDir, script := bunDispatchE2EManifest(t)
	projectRoot := bunDispatchE2EProjectRoot(t, true)

	// NO stubbed dispatcher, NO canned-stdout fixtureRunner — a REAL CommandRunner.
	runner := &check.ExecCommandRunner{Dir: projectRoot}
	if got := fmt.Sprintf("%T", runner); got != "*check.ExecCommandRunner" {
		t.Fatalf("the e2e must drive a REAL check.ExecCommandRunner, not a stub, got %s", got)
	}
	// The executed engine is the committed POSIX fake script under the fixture's
	// testdata tree — never a real toolchain binary (no real bun/tsc/oxlint/go runs
	// in Go CI). Asserting the resolved executable IS the committed fake script (a
	// .sh under the fixture dir) proves the absence of any real toolchain without
	// baking tool-name literals into the test.
	if filepath.Base(script) != "fake-engine.sh" || !strings.HasSuffix(script, ".sh") {
		t.Fatalf("the engine must be the committed fake POSIX script, got %q", script)
	}
	if !strings.HasPrefix(script, bunDispatchE2EFixtureDir(t)) {
		t.Fatalf("the engine must be the committed fake under the fixture testdata tree, not a real binary on PATH, got %q", script)
	}

	violations, err := dispatchPackEngines([]*pack.Manifest{m}, packsDir, projectRoot, nil, runner)
	if err != nil {
		t.Fatalf("real-dispatch run errored: %v", err)
	}
	if len(violations) != 1 || !strings.Contains(violations[0].Message, "SEEDED_DEFECT") {
		t.Fatalf("the dispatch must surface the artifact FILE's seeded finding, got %d: %#v", len(violations), violations)
	}

	// SELF-TARGET proof (DEFECT-1): the dispatch appended NO scan target. A bolted-on
	// projectRoot would make the fake suppress even the seeded finding.
	if argc := strings.TrimSpace(readFileStr(t, filepath.Join(projectRoot, "argc.txt"))); argc != "0" {
		t.Fatalf("DEFECT-1: the engine self-targets and must receive NO appended arg, argc=%q", argc)
	}

	// ARTIFACT-vs-STDOUT proof (DEFECT-2): the FILE carries the findings; stdout is
	// human noise with no SARIF. Run the same fake directly to inspect both streams.
	stdout, runErr := runner.RunStdout(context.Background(), script)
	// The fake exits non-zero when it emits a finding (like a real linter); the
	// stdout bytes are the contract we inspect, so a non-nil runErr here is EXPECTED.
	_ = runErr
	if strings.Contains(string(stdout), "runs") || strings.Contains(string(stdout), "results") || strings.Contains(string(stdout), "SEEDED_DEFECT") {
		t.Errorf("the engine's STDOUT must be human noise with NO SARIF/findings, got: %q", string(stdout))
	}
	artifact := readFileStr(t, filepath.Join(projectRoot, "findings.sarif"))
	if !strings.Contains(artifact, "results") || !strings.Contains(artifact, "SEEDED_DEFECT") {
		t.Errorf("the machine-readable findings must live in the stdout_artifact FILE, got: %q", artifact)
	}
}
