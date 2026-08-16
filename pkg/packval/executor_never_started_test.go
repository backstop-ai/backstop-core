package packval

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/pack/engine"
)

// writeUnstartableEngineScript writes a REAL non-executable file and guards the
// fixture's own premise by re-stat'ing it. Without that guard a fixture that became
// executable would silently stop testing the fork/exec shape while still passing.
// Copied from writeNonExecutableScript in cmd/backstop/dispatch_unstartable_engine_test.go.
func writeUnstartableEngineScript(t *testing.T, dir, name string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\necho should-never-run\n"), 0o644); err != nil {
		t.Fatalf("write non-executable script: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat script: %v", err)
	}
	if info.Mode().Perm()&0o111 != 0 {
		t.Fatalf("fixture invariant: %s must NOT be executable, got mode %v", path, info.Mode().Perm())
	}
	return path
}

// TestExecutor_RunEngineRefusesPathfulUnstartableEngine is THE DEFECT (CLM-003,
// CLM-004). binding.Command is pack-declared DATA and may carry a path separator; a
// PATH-FUL command never consults LookPath, so its failure is an *fs.PathError with
// Op "fork/exec" and NEVER an *exec.Error. Before ISSUE-140 the executor classified
// only *exec.Error, so this run fell through to a parse of EMPTY stdout and returned
// ExecutionResult{Passed:false}, nil — a silent pass for an engine that never ran.
//
// Every binding here has a NIL Provision so engine.CheckToolAllowed returns early and
// THE RUN is what fails; a non-nil Provision would fail at the allowlist gate instead,
// a red that keeps firing after the fix.
func TestExecutor_RunEngineRefusesPathfulUnstartableEngine(t *testing.T) {
	dir := t.TempDir()
	script := writeUnstartableEngineScript(t, dir, "unstartable-engine.sh")
	binding := engine.EngineBinding{Command: script + " --json", Provision: nil}

	res, err := (&DefaultExecutor{}).RunEngine(dir, binding, []string{"fixture.txt"})
	if err == nil {
		t.Fatalf("expected an error, got nil — a path-ful engine that never started read as a finding-free pass (result: %+v)", res)
	}
	if res.Passed {
		t.Fatalf("expected a non-passing result alongside the error, got Passed=true (%+v)", res)
	}
	// CLM-004: the refusal must NAME the broken command. Substring only, so the
	// wording can improve without churning this test.
	if !strings.Contains(err.Error(), binding.Command) {
		t.Fatalf("error must name the declared command %q; got: %v", binding.Command, err)
	}
}

// TestExecutor_RunEngineRefusesBareAbsentEngine pins the *exec.Error behavior that
// ALREADY works (CLM-003, continuity), so widening the classification cannot silently
// trade one shape for the other. This MUST pass at HEAD; if it reds there the fixture
// is wrong, not production.
func TestExecutor_RunEngineRefusesBareAbsentEngine(t *testing.T) {
	// A fixed nonsense bare name — no path separator, so exec.Command routes it
	// through LookPath, which misses. PATH is never mutated.
	binding := engine.EngineBinding{Command: "backstop-absent-engine-140", Provision: nil}

	res, err := (&DefaultExecutor{}).RunEngine(t.TempDir(), binding, []string{"fixture.txt"})
	if err == nil {
		t.Fatalf("expected an error for a bare absent engine, got nil (result: %+v)", res)
	}
	if res.Passed {
		t.Fatalf("expected a non-passing result alongside the error, got Passed=true (%+v)", res)
	}
	if !strings.Contains(err.Error(), binding.Command) {
		t.Fatalf("error must name the declared command %q; got: %v", binding.Command, err)
	}
}

// TestExecutor_RunEngineStartedNonZeroExitStillReportsFindings is the assertion that
// keeps CLM-003 honest (CLM-005). A findings engine legitimately exits non-zero
// precisely WHEN it reports findings, so the exit code is not the contract — the SARIF
// on stdout is. An implementation reduced to `runErr != nil` fails here.
//
// SKIP-DISCIPLINE: if the host cannot run a /bin/sh script this test FAILS rather than
// skipping — a skip would silently remove the only guard against that mistake.
func TestExecutor_RunEngineStartedNonZeroExitStillReportsFindings(t *testing.T) {
	dir := t.TempDir()
	// A minimal SARIF log with one result, shaped to what check.ParsePackFindings
	// (pkg/check/parsers.go) actually consumes: runs[].results[] with ruleId, level,
	// message.text and a physicalLocation.
	script := "#!/bin/sh\ncat <<'SARIF'\n" + `{
  "runs": [
    {
      "results": [
        {
          "ruleId": "org/pack/no-panic",
          "level": "error",
          "message": {"text": "panic is forbidden"},
          "locations": [
            {"physicalLocation": {"artifactLocation": {"uri": "pkg/widget/widget.go"}, "region": {"startLine": 42}}}
          ]
        }
      ]
    }
  ]
}` + "\nSARIF\nexit 1\n"

	path := filepath.Join(dir, "findings-engine.sh")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write findings engine script: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat script: %v", err)
	}
	if info.Mode().Perm()&0o111 == 0 {
		t.Fatalf("fixture invariant: %s MUST be executable, got mode %v", path, info.Mode().Perm())
	}

	binding := engine.EngineBinding{Command: path, Provision: nil}
	res, runErr := (&DefaultExecutor{}).RunEngine(dir, binding, []string{"fixture.txt"})
	if runErr != nil {
		t.Fatalf("a STARTED engine exiting non-zero with SARIF must not error: %v", runErr)
	}
	if !res.Passed {
		t.Fatalf("expected the reported findings to yield Passed=true, got %+v", res)
	}
}
