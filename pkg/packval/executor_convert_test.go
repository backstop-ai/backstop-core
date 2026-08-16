package packval

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/pack/engine"
)

// ── FIXTURE CONSTRUCTION (ISSUE-141) ────────────────────────────────────────
// Every fixture here is written at RUNTIME into t.TempDir() rather than committed
// under testdata/: PLAN-ISSUE-092 owns several pkg/packval/testdata/ subtrees, and a
// committed script's exec bit is a durability hazard that presents as an unrelated
// failure. The sandbox confines reads to packDir, so the temp dir IS the pack dir.
// Precedent: sandbox_stdout_test.go, sandbox_realconvert_test.go.

// writeConvertFixtureScript writes an executable fixture script and re-stats it, so a
// fixture that silently lost its exec bit fails as a fixture problem rather than as a
// convert bug. It returns the ABSOLUTE path: a binding.Command without a path
// separator is resolved through PATH, and a relative one against the PROCESS cwd
// rather than cmd.Dir, either of which would trip check.NeverStarted and stay red
// even after the fix — indistinguishable from the defect these tests pin.
func writeConvertFixtureScript(t *testing.T, dir, name, body string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatalf("write fixture script %q: %v", path, err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat fixture script %q: %v", path, err)
	}
	if info.Mode().Perm()&0o111 == 0 {
		t.Fatalf("fixture invariant: %s MUST be executable, got mode %v", path, info.Mode().Perm())
	}
	return path
}

// convertFixturePackDir returns a symlink-RESOLVED temp dir to use as packDir. The
// sandbox's path rules (darwin subpath, Landlock path_beneath) match the
// KERNEL-resolved path, so an unresolved macOS /var/... packDir would fail to match
// the real /private/var/... and deny the convert script its own directory.
func convertFixturePackDir(t *testing.T) string {
	t.Helper()
	resolved, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatalf("EvalSymlinks on temp pack dir: %v", err)
	}
	return resolved
}

// requireSandboxPlatform skips ONLY where the sandbox genuinely does not exist,
// using the same-package predicate that already encodes that answer. It is
// deliberately NOT `runtime.GOOS != "darwin"`: that spelling skips on Linux, which
// is the only platform that gates merges, so every convert test would be a vacuous
// green in CI. On darwin and linux alike these tests RUN FOR REAL.
func requireSandboxPlatform(t *testing.T) {
	t.Helper()
	if err := sandboxPlatformSupported(runtime.GOOS); err != nil {
		t.Skipf("no sandbox implementation on this platform: %v", err)
	}
}

// nonSarifEngineScript emits the shape `ast-grep scan --json` really produces — a
// bare JSON ARRAY of match objects with no `runs` key — so check.ParsePackFindings
// cannot unmarshal it. recordCount records are emitted.
func nonSarifEngineScript(recordCount int) string {
	var b strings.Builder
	b.WriteString("#!/bin/sh\ncat <<'JSON'\n[")
	for i := 0; i < recordCount; i++ {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString(`{"ruleId":"fixture/non-sarif","file":"pkg/x/x.go","line":1}`)
	}
	b.WriteString("]\nJSON\n")
	return b.String()
}

// toSarifConvertScript reshapes the non-SARIF array on stdin into SARIF 2.1.0. Its
// OUTPUT IS DERIVED FROM ITS INPUT: it counts the `"ruleId"` occurrences it actually
// read and emits exactly that many results. A converter printing a constant SARIF
// document would satisfy "convert ran" without proving the engine's bytes traversed
// the pipe; here a broken pipe yields the WRONG answer (zero results) rather than the
// right one by accident.
//
// Pure POSIX shell builtins only (case/while/printf/[) — no jq, grep or tr — so the
// script needs nothing beyond its interpreter inside the sandbox on either platform.
const toSarifConvertScript = `#!/bin/sh
input=$(cat)
n=0
rest=$input
while :; do
  case "$rest" in
    *'"ruleId"'*) n=$((n+1)); rest=${rest#*'"ruleId"'} ;;
    *) break ;;
  esac
done
printf '{"runs":[{"results":['
i=0
while [ $i -lt $n ]; do
  if [ $i -gt 0 ]; then printf ','; fi
  printf '{"ruleId":"converted/rule-%s","level":"error","message":{"text":"converted finding %s"},"locations":[{"physicalLocation":{"artifactLocation":{"uri":"pkg/x/x.go"},"region":{"startLine":%s}}}]}' "$i" "$i" "$((i+1))"
  i=$((i+1))
done
printf ']}]}\n'
`

// runConvertFixture builds a one-off fixture pack (engine script + optional convert
// script) and runs the production DefaultExecutor over it.
func runConvertFixture(t *testing.T, engineBody, convertName, convertBody string) (ExecutionResult, error) {
	t.Helper()
	packDir := convertFixturePackDir(t)
	enginePath := writeConvertFixtureScript(t, packDir, "engine.sh", engineBody)
	if convertBody != "" {
		writeConvertFixtureScript(t, packDir, convertName, convertBody)
	}
	// NO Provision block (SE6): RunEngine runs engine.CheckToolAllowed FIRST when
	// Provision is non-nil, and a fixture script is not on the trusted-tool
	// allowlist — every test would fail at the allowlist gate before reaching the
	// code under test, and the failure would look like a convert bug.
	binding := engine.EngineBinding{Command: enginePath, Convert: convertName, Provision: nil}
	return (&DefaultExecutor{}).RunEngine(packDir, binding, []string{"fixture.txt"})
}

// TestPackVal_EngineConvert_NonSarifPipesThroughConvert is THE DEFECT (CLM-001).
// DefaultExecutor.RunEngine handed the engine's RAW stdout to
// check.ParsePackFindings and never applied the binding's declared Convert, so a pack
// whose engine emits non-SARIF output and ships a reshaper died with
// `engine %q produced no parseable SARIF` on output the convert would have made
// parseable.
//
// The second leg is what makes the first leg mean something: the SAME converter over
// an engine emitting ZERO records must report not-fired. A converter printing a
// constant document would pass the first leg and fail this one.
func TestPackVal_EngineConvert_NonSarifPipesThroughConvert(t *testing.T) {
	requireSandboxPlatform(t)

	res, err := runConvertFixture(t, nonSarifEngineScript(2), "to-sarif.sh", toSarifConvertScript)
	if err != nil {
		t.Fatalf("a non-SARIF engine with a declared convert must yield real findings; got error: %v (result %+v)", err, res)
	}
	if !res.Passed {
		t.Fatalf("expected the converted findings to yield Passed=true, got %+v", res)
	}

	empty, emptyErr := runConvertFixture(t, nonSarifEngineScript(0), "to-sarif.sh", toSarifConvertScript)
	if emptyErr != nil {
		t.Fatalf("a zero-record engine through the same convert must parse cleanly; got error: %v (result %+v)", emptyErr, empty)
	}
	if empty.Passed {
		t.Fatalf("THE DERIVATION FAILED: the converter reported findings for an engine that emitted ZERO records, "+
			"so its output is not derived from its input and the pipe is not proven. got %+v", empty)
	}
}

// TestPackVal_EngineConvert_NativeSarifNoConvert is the assertion that keeps CLM-001
// honest (CLM-002). A SARIF-native binding with an EMPTY Convert parses stdout
// directly, byte-for-byte, and spawns no convert process. The pack dir deliberately
// contains no convert script at all, so any attempt to resolve one would surface as
// a missing-script refusal rather than passing silently.
func TestPackVal_EngineConvert_NativeSarifNoConvert(t *testing.T) {
	sarifEngine := "#!/bin/sh\ncat <<'SARIF'\n" + `{
  "runs": [
    {
      "results": [
        {
          "ruleId": "fixture/native-sarif",
          "level": "error",
          "message": {"text": "native SARIF finding"},
          "locations": [
            {"physicalLocation": {"artifactLocation": {"uri": "pkg/x/x.go"}, "region": {"startLine": 7}}}
          ]
        }
      ]
    }
  ]
}` + "\nSARIF\n"

	res, err := runConvertFixture(t, sarifEngine, "", "")
	if err != nil {
		t.Fatalf("a SARIF-native engine with no declared convert must parse directly; got error: %v (result %+v)", err, res)
	}
	if !res.Passed {
		t.Fatalf("expected the native SARIF finding to yield Passed=true, got %+v", res)
	}
	if !strings.Contains(res.Output, "fixture/native-sarif") {
		t.Fatalf("the engine's own stdout must reach the caller unchanged on the no-convert path; got Output %q", res.Output)
	}
}

// TestPackVal_EngineConvert_MissingScriptFailsLoud pins CLM-003. A declared-but-absent
// convert script is a fail-loud refusal NAMING THE RESOLVED PATH, never a silent
// fall-through to the raw payload — a fall-through would re-open the exact defect as a
// soft failure, which is strictly worse than today because it would look fixed.
//
// Deliberately NOT platform-guarded: the stat refusal precedes any sandboxed run, so
// this holds on every platform.
func TestPackVal_EngineConvert_MissingScriptFailsLoud(t *testing.T) {
	t.Run("absent", func(t *testing.T) {
		packDir := convertFixturePackDir(t)
		enginePath := writeConvertFixtureScript(t, packDir, "engine.sh", nonSarifEngineScript(2))
		binding := engine.EngineBinding{Command: enginePath, Convert: "does-not-exist.sh", Provision: nil}

		res, err := (&DefaultExecutor{}).RunEngine(packDir, binding, []string{"fixture.txt"})
		if err == nil {
			t.Fatalf("a declared-but-missing convert script must fail loud, got nil error (result %+v)", res)
		}
		if res.Passed {
			t.Fatalf("expected a non-passing result alongside the error, got %+v", res)
		}
		want := filepath.Join(packDir, "does-not-exist.sh")
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("the refusal must name the RESOLVED convert path %q; got: %v", want, err)
		}
	})

	t.Run("directory", func(t *testing.T) {
		packDir := convertFixturePackDir(t)
		enginePath := writeConvertFixtureScript(t, packDir, "engine.sh", nonSarifEngineScript(2))
		convertDir := filepath.Join(packDir, "convert-is-a-dir")
		if err := os.Mkdir(convertDir, 0o755); err != nil {
			t.Fatalf("mkdir convert directory fixture: %v", err)
		}
		binding := engine.EngineBinding{Command: enginePath, Convert: "convert-is-a-dir", Provision: nil}

		res, err := (&DefaultExecutor{}).RunEngine(packDir, binding, []string{"fixture.txt"})
		if err == nil {
			t.Fatalf("a convert pointing at a DIRECTORY must refuse identically, got nil error (result %+v)", res)
		}
		if res.Passed {
			t.Fatalf("expected a non-passing result alongside the error, got %+v", res)
		}
		if !strings.Contains(err.Error(), convertDir) {
			t.Fatalf("the refusal must name the RESOLVED convert path %q; got: %v", convertDir, err)
		}
	})
}

// TestPackVal_EngineConvert_ConvertFailureAttributedToConvertStep pins CLM-004. A
// convert script that RUNS AND FAILS is attributed to the CONVERT STEP BY NAME.
// Misreporting it as the engine producing unparseable SARIF sends a pack author to
// debug the wrong program.
func TestPackVal_EngineConvert_ConvertFailureAttributedToConvertStep(t *testing.T) {
	requireSandboxPlatform(t)

	failing := "#!/bin/sh\ncat > /dev/null\necho 'converter blew up' 1>&2\nexit 3\n"
	res, err := runConvertFixture(t, nonSarifEngineScript(2), "broken-convert.sh", failing)
	if err == nil {
		t.Fatalf("a convert script exiting non-zero must surface as an error, got nil (result %+v)", res)
	}
	if res.Passed {
		t.Fatalf("expected a non-passing result alongside the error, got %+v", res)
	}
	if !strings.Contains(err.Error(), "convert") {
		t.Fatalf("the failure must be attributed to the CONVERT step by name; got: %v", err)
	}
	if !strings.Contains(err.Error(), "broken-convert.sh") {
		t.Fatalf("the failure must name the declared convert %q; got: %v", "broken-convert.sh", err)
	}
	if strings.Contains(err.Error(), "produced no parseable SARIF") {
		t.Fatalf("a FAILED CONVERT must not be reported as the engine producing unparseable SARIF — "+
			"that points the pack author at the wrong program; got: %v", err)
	}
}

// TestPackVal_EngineConvert_RunsThroughSandboxedRunStdout pins CLM-005: the convert
// runs through the PRODUCTION same-package sandbox seam, not an approximation of it.
// The converter writes a banner to stderr and the SARIF to stdout. SandboxedRun's
// CombinedOutput semantics would interleave the two and the SARIF parse would fail;
// SandboxedRunStdout's explicit stdout buffer is what makes this pass.
func TestPackVal_EngineConvert_RunsThroughSandboxedRunStdout(t *testing.T) {
	requireSandboxPlatform(t)

	noisy := "#!/bin/sh\necho 'WARNING: converter banner' 1>&2\n" + strings.TrimPrefix(toSarifConvertScript, "#!/bin/sh\n")
	res, err := runConvertFixture(t, nonSarifEngineScript(1), "noisy-convert.sh", noisy)
	if err != nil {
		t.Fatalf("the converter's stderr banner must not reach the SARIF parser — a CombinedOutput-based "+
			"sandbox call is the likely cause; got error: %v (result %+v)", err, res)
	}
	if !res.Passed {
		t.Fatalf("expected the converted finding to yield Passed=true, got %+v", res)
	}
}

// convertGuardMechanisms are the three mechanisms both convert-application sites must
// carry. `os.Stat` is the existence refusal; the sandbox call is the execution seam.
type convertGuardMechanisms struct {
	name          string
	body          string
	sandboxTokens []string
}

// TestPackVal_EngineConvert_NoDriftFromGateDispatch is the mechanical drift guard
// (CLM-006). Two call sites apply Convert — RunEngine here and runFindingsEngine in
// cmd/backstop/pack_gate.go — and consolidating them into one exported helper is
// residual R1, deliberately not taken here because pack_gate.go is another lane's
// exclusive scope. This guard holds the line until that consolidation is planned.
//
// IT IS A CONTENT SCAN, NOT A TREE-STATE CHECK: no git status, no git diff, no
// working-tree-cleanliness assertion and no line numbers. This is a shared tree with
// concurrent lanes, and a tree-state assertion blames whoever happens to run it.
//
// THE pack_gate.go HALF IS WINDOWED TO runFindingsEngine. That file contains TWO
// independent convert blocks — one in runCoverageEngine and one in runFindingsEngine
// — each carrying all three mechanisms, so a whole-file scan could not fail even if
// runFindingsEngine's block were deleted outright. The window is located by FUNCTION
// NAME, never by line number.
func TestPackVal_EngineConvert_NoDriftFromGateDispatch(t *testing.T) {
	executorSrc := readGuardSource(t, filepath.Join("executor.go"))
	gateSrc := readGuardSource(t, filepath.Join("..", "..", "cmd", "backstop", "pack_gate.go"))

	sites := []convertGuardMechanisms{
		{
			name:          "pkg/packval/executor.go RunEngine",
			body:          windowFunc(t, executorSrc, "func (d *DefaultExecutor) RunEngine("),
			sandboxTokens: []string{"SandboxedRunStdout"},
		},
		{
			name:          "cmd/backstop/pack_gate.go runFindingsEngine",
			body:          windowFunc(t, gateSrc, "func runFindingsEngine("),
			sandboxTokens: []string{"resolveSandboxedRunStdout", "SandboxedRunStdout"},
		},
	}

	const drift = "the two convert-application paths have DRIFTED. Both must stat-check the resolved " +
		"convert path and pipe the payload through the sandboxed stdout seam before parsing. Consolidating " +
		"them into one exported helper is residual R1 of PLAN-ISSUE-141 and is not yet planned; until it is, " +
		"this guard is what holds the line"

	for _, site := range sites {
		if !strings.Contains(site.body, "binding.Convert") {
			t.Fatalf("%s does not reference binding.Convert: %s", site.name, drift)
		}
		if !strings.Contains(site.body, "os.Stat") {
			t.Fatalf("%s performs no os.Stat existence refusal on the resolved convert path: %s", site.name, drift)
		}
		found := false
		for _, tok := range site.sandboxTokens {
			if strings.Contains(site.body, tok) {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("%s does not reach the sandboxed stdout seam (looked for %v): %s", site.name, site.sandboxTokens, drift)
		}
	}
}

func readGuardSource(t *testing.T, path string) string {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %q for the convert drift guard: %v", path, err)
	}
	return string(body)
}

// windowFunc returns the source between decl and the next top-level `func `
// declaration, so a guard scoped to one function cannot be satisfied by a sibling.
func windowFunc(t *testing.T, src, decl string) string {
	t.Helper()
	start := strings.Index(src, decl)
	if start < 0 {
		t.Fatalf("could not locate %q — the convert drift guard windows by FUNCTION NAME and the "+
			"declaration has been renamed or removed", decl)
	}
	rest := src[start+len(decl):]
	if end := strings.Index(rest, "\nfunc "); end >= 0 {
		return rest[:end]
	}
	return rest
}
