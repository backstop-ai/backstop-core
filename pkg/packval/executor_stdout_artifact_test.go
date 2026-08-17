package packval

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/pack/engine"
)

// ── FIXTURE CONSTRUCTION (ISSUE-144) ────────────────────────────────────────
// Every fixture here is built at RUNTIME into a temp dir, reusing the helpers
// executor_convert_test.go already owns in this package —
// writeConvertFixtureScript (returns the ABSOLUTE path, which buildEngineArgv
// needs so exec.Command resolves against neither PATH nor the PROCESS cwd),
// convertFixturePackDir, requireSandboxPlatform, readGuardSource and windowFunc.
// Re-declaring any of them would be a duplicate declaration in package packval.
//
// The engine scripts below write their artifact via a path RELATIVE TO THEIR OWN
// CWD. RunEngine sets cmd.Dir = packDir and the engine run is not sandboxed, so a
// plain `> results.sarif` lands in packDir — which is exactly what makes the
// base-directory assertions discriminating rather than incidentally true.

// stdoutArtifactSarif renders a SARIF 2.1.0 document carrying resultCount results.
func stdoutArtifactSarif(resultCount int) string {
	var b strings.Builder
	b.WriteString(`{"runs":[{"results":[`)
	for i := 0; i < resultCount; i++ {
		if i > 0 {
			b.WriteString(",")
		}
		fmt.Fprintf(&b, `{"ruleId":"fixture/artifact-%d","level":"error","message":{"text":"artifact finding %d"},`+
			`"locations":[{"physicalLocation":{"artifactLocation":{"uri":"pkg/x/x.go"},"region":{"startLine":%d}}}]}`,
			i, i, i+1)
	}
	b.WriteString(`]}]}`)
	return b.String()
}

// stdoutArtifactEngineScript emits stdoutBody to stdout and, when artifactRel is
// non-empty, writes artifactBody to that pack-relative path (creating its parent
// directory) from the engine's own cwd.
func stdoutArtifactEngineScript(stdoutBody, artifactRel, artifactBody string) string {
	var b strings.Builder
	b.WriteString("#!/bin/sh\n")
	if artifactRel != "" {
		if dir := filepath.ToSlash(filepath.Dir(artifactRel)); dir != "." && dir != "" {
			fmt.Fprintf(&b, "mkdir -p '%s'\n", dir)
		}
		fmt.Fprintf(&b, "cat > '%s' <<'ARTIFACT'\n%s\nARTIFACT\n", artifactRel, artifactBody)
	}
	if stdoutBody != "" {
		fmt.Fprintf(&b, "cat <<'STDOUT'\n%s\nSTDOUT\n", stdoutBody)
	}
	return b.String()
}

// runStdoutArtifactFixture builds a one-off fixture pack and runs the production
// DefaultExecutor over it, returning the packDir alongside the result so a caller
// can assert on the RESOLVED artifact path. No Provision block: RunEngine runs
// engine.CheckToolAllowed first when Provision is non-nil, and a fixture script is
// not on the trusted-tool allowlist, so every test would die at the allowlist gate
// before reaching the code under test.
func runStdoutArtifactFixture(t *testing.T, engineBody, stdoutArtifact, convertName, convertBody string) (string, ExecutionResult, error) {
	t.Helper()
	packDir := convertFixturePackDir(t)
	enginePath := writeConvertFixtureScript(t, packDir, "engine.sh", engineBody)
	if convertBody != "" {
		writeConvertFixtureScript(t, packDir, convertName, convertBody)
	}
	binding := engine.EngineBinding{
		Command:        enginePath,
		StdoutArtifact: stdoutArtifact,
		Convert:        convertName,
		Provision:      nil,
	}
	res, err := (&DefaultExecutor{}).RunEngine(packDir, binding, []string{"fixture.txt"})
	return packDir, res, err
}

// TestPackVal_EngineStdoutArtifact_ArtifactBytesParsedNotStdout is THE DEFECT
// (CLM-001). DefaultExecutor.RunEngine initialized its payload unconditionally from
// stdout and never consulted binding.StdoutArtifact, so an engine that writes its
// real machine-readable output to the declared FILE and prints nothing to stdout had
// its output thrown away.
//
// The verdict that produced was a LIE, not a crash: check.ParsePackFindings trims its
// input and reads empty stdout as ZERO findings with NO error, so this run returned
// Passed=false, ExitCode=0, err=nil — and Passed=false is the SUCCESS condition for a
// negative phase3 fixture. A clean pass over a run whose output was never read.
func TestPackVal_EngineStdoutArtifact_ArtifactBytesParsedNotStdout(t *testing.T) {
	script := stdoutArtifactEngineScript("", "results.sarif", stdoutArtifactSarif(1))

	_, res, err := runStdoutArtifactFixture(t, script, "results.sarif", "", "")
	if err != nil {
		t.Fatalf("an engine writing real SARIF to its declared stdout_artifact must yield findings; got error: %v (result %+v)", err, res)
	}
	if !res.Passed {
		t.Fatalf("THE LYING VERDICT: the engine wrote one real SARIF result to its declared stdout_artifact "+
			"and printed nothing to stdout, yet the run reports not-fired — which is the SUCCESS condition for a "+
			"negative fixture. The declared artifact was never read. got %+v", res)
	}
}

// TestPackVal_EngineStdoutArtifact_AbsentDeclarationParsesStdout is the assertion
// that keeps CLM-001 honest (CLM-002). A binding with an EMPTY StdoutArtifact parses
// stdout directly and touches no file. The pack dir deliberately contains no artifact
// file at all, so an implementation that reads a file unconditionally surfaces as a
// refusal rather than passing silently.
//
// Green both before and after the fix, by design.
func TestPackVal_EngineStdoutArtifact_AbsentDeclarationParsesStdout(t *testing.T) {
	script := stdoutArtifactEngineScript(stdoutArtifactSarif(1), "", "")

	packDir, res, err := runStdoutArtifactFixture(t, script, "", "", "")
	if err != nil {
		t.Fatalf("a binding with no declared stdout_artifact must parse stdout directly; got error: %v (result %+v)", err, res)
	}
	if !res.Passed {
		t.Fatalf("expected the stdout SARIF finding to yield Passed=true, got %+v", res)
	}
	if !strings.Contains(res.Output, "fixture/artifact-0") {
		t.Fatalf("the engine's own stdout must reach the caller unchanged on the no-declaration path; got Output %q", res.Output)
	}
	if entries, readErr := os.ReadDir(packDir); readErr == nil {
		for _, e := range entries {
			if e.Name() != "engine.sh" {
				t.Fatalf("the no-declaration path must write and read no artifact; found %q in the pack dir", e.Name())
			}
		}
	}
}

// TestPackVal_EngineStdoutArtifact_FileWinsOverConflictingStdout pins CLM-003:
// SELECTION, not merge and not fallback. Both legs declare the artifact and put
// CONFLICTING SARIF in the two sources.
//
// Leg (b) is the one that cannot be faked. An implementation that concatenates the
// two sources, or prefers whichever is non-empty, or falls back to stdout when the
// file parses to nothing, reports fired there and fails.
func TestPackVal_EngineStdoutArtifact_FileWinsOverConflictingStdout(t *testing.T) {
	t.Run("file_fires_stdout_empty_of_results", func(t *testing.T) {
		script := stdoutArtifactEngineScript(stdoutArtifactSarif(0), "results.sarif", stdoutArtifactSarif(1))

		_, res, err := runStdoutArtifactFixture(t, script, "results.sarif", "", "")
		if err != nil {
			t.Fatalf("the declared artifact's one result must be the payload; got error: %v (result %+v)", err, res)
		}
		if !res.Passed {
			t.Fatalf("expected the ARTIFACT's single result to decide the verdict, got %+v", res)
		}
	})

	t.Run("file_clean_stdout_fires", func(t *testing.T) {
		script := stdoutArtifactEngineScript(stdoutArtifactSarif(1), "results.sarif", stdoutArtifactSarif(0))

		_, res, err := runStdoutArtifactFixture(t, script, "results.sarif", "", "")
		if err != nil {
			t.Fatalf("a clean artifact must parse cleanly, not error; got: %v (result %+v)", err, res)
		}
		if res.Passed {
			t.Fatalf("SELECTION FAILED: the declared artifact carried ZERO results and stdout carried one, yet the "+
				"run reports fired — the implementation merges the two sources, prefers whichever is non-empty, or "+
				"falls back to stdout. Stdout must not be consulted at all when an artifact is declared. got %+v", res)
		}
	})
}

// TestPackVal_EngineStdoutArtifact_MissingArtifactFailsLoud pins CLM-004. A
// declared-but-unproduced artifact is a BROKEN RUN naming both the declared value and
// the resolved path — never a silent fallback to stdout, which would re-open the
// defect as a soft failure that merely looks fixed.
//
// Pre-fix this test is only PARTIALLY red, and that is correct: today's RunEngine
// parses the human summary off stdout, fails to read it as SARIF and returns a parse
// error, so the non-nil-error and not-Passed legs already hold. The two SUBSTRING
// assertions are what discriminate "failed for the right reason, naming what a pack
// author needs to fix" from "failed by accident".
//
// Deliberately NOT platform-guarded: no sandbox is involved on this path.
func TestPackVal_EngineStdoutArtifact_MissingArtifactFailsLoud(t *testing.T) {
	script := stdoutArtifactEngineScript("ran 3 checks, 0 problems", "", "")

	packDir, res, err := runStdoutArtifactFixture(t, script, "results.sarif", "", "")
	if err == nil {
		t.Fatalf("a declared-but-unproduced stdout_artifact must fail loud — a nil error here is the SOFT-FAILURE "+
			"regression CLM-004 forbids, because the run silently succeeded off stdout instead. got result %+v", res)
	}
	if res.Passed {
		t.Fatalf("expected a non-passing result alongside the error, got %+v", res)
	}
	if !strings.Contains(err.Error(), "results.sarif") {
		t.Fatalf("the refusal must name the DECLARED value %q so a pack author can find it in pack.yml; got: %v", "results.sarif", err)
	}
	want := filepath.Join(packDir, "results.sarif")
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("the refusal must name the RESOLVED path %q; got: %v", want, err)
	}
}

// TestPackVal_EngineStdoutArtifact_ResolvedAgainstPackDir pins CLM-005: the base is
// packDir — the run's own working directory, per the StdoutArtifact field's own
// contract ("relative to the run's working dir") and RunEngine's cmd.Dir = packDir.
// runFindingsEngine joins against projectRoot because THAT is its run's working dir;
// same rule, different value.
//
// Leg (a) proves it operationally through a NESTED relative path the engine writes
// from its own cwd: a read joined against the TEST PROCESS's cwd (pkg/packval during
// go test) finds nothing. Leg (b) pins the base BY NAME rather than by inference.
func TestPackVal_EngineStdoutArtifact_ResolvedAgainstPackDir(t *testing.T) {
	const declared = "out/results.sarif"

	t.Run("nested_relative_path_read_from_pack_dir", func(t *testing.T) {
		script := stdoutArtifactEngineScript("", declared, stdoutArtifactSarif(1))

		_, res, err := runStdoutArtifactFixture(t, script, declared, "", "")
		if err != nil {
			t.Fatalf("a nested artifact written relative to the engine's cwd must resolve against packDir; got error: %v (result %+v)", err, res)
		}
		if !res.Passed {
			t.Fatalf("expected the nested artifact's result to decide the verdict, got %+v", res)
		}
	})

	t.Run("refusal_names_the_pack_dir_resolved_path", func(t *testing.T) {
		script := stdoutArtifactEngineScript("no problems found", "", "")

		packDir, res, err := runStdoutArtifactFixture(t, script, declared, "", "")
		if err == nil {
			t.Fatalf("a declared-but-unproduced nested artifact must fail loud, got nil error (result %+v)", res)
		}
		want := filepath.Join(packDir, filepath.FromSlash(declared))
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("the refusal must name the packDir-resolved path %q — naming any other base means the read "+
				"is joined against the wrong directory; got: %v", want, err)
		}
	})
}

// TestPackVal_EngineStdoutArtifact_ConvertConsumesArtifactBytes pins CLM-006: payload
// SELECTION precedes CONVERT, so the declared converter reshapes the ARTIFACT's bytes
// rather than raw stdout.
//
// This is the ONE test here that may skip: only the convert step runs sandboxed.
//
// toSarifConvertScript's output is DERIVED from its input — it counts the "ruleId"
// occurrences it actually read — so a pipeline wired the wrong way round gives the
// WRONG answer rather than the right one by accident. Leg (b) fails loudly if
// selection runs AFTER convert, or not at all.
func TestPackVal_EngineStdoutArtifact_ConvertConsumesArtifactBytes(t *testing.T) {
	requireSandboxPlatform(t)

	nonSarifBody := func(recordCount int) string {
		var b strings.Builder
		b.WriteString("[")
		for i := 0; i < recordCount; i++ {
			if i > 0 {
				b.WriteString(",")
			}
			b.WriteString(`{"ruleId":"fixture/non-sarif","file":"pkg/x/x.go","line":1}`)
		}
		b.WriteString("]")
		return b.String()
	}

	t.Run("artifact_records_reach_the_converter", func(t *testing.T) {
		script := stdoutArtifactEngineScript(nonSarifBody(0), "results.json", nonSarifBody(2))

		_, res, err := runStdoutArtifactFixture(t, script, "results.json", "to-sarif.sh", toSarifConvertScript)
		if err != nil {
			t.Fatalf("the converter must consume the ARTIFACT's bytes; got error: %v (result %+v)", err, res)
		}
		if !res.Passed {
			t.Fatalf("expected the artifact's two records to convert into findings, got %+v", res)
		}
	})

	t.Run("stdout_records_never_reach_the_converter", func(t *testing.T) {
		script := stdoutArtifactEngineScript(nonSarifBody(2), "results.json", nonSarifBody(0))

		_, res, err := runStdoutArtifactFixture(t, script, "results.json", "to-sarif.sh", toSarifConvertScript)
		if err != nil {
			t.Fatalf("a zero-record artifact through the same convert must parse cleanly; got error: %v (result %+v)", err, res)
		}
		if res.Passed {
			t.Fatalf("ORDER FAILED: the declared artifact carried ZERO records and stdout carried two, yet the "+
				"converter reported findings — selection runs AFTER convert, or not at all. got %+v", res)
		}
	})
}

// TestPackVal_EngineStdoutArtifact_NeverStartedPrecedesArtifactRead pins CLM-007: the
// never-started refusal still comes FIRST. A process that never ran produced no
// artifact, so blaming the missing file would misattribute the failure — the same
// misattribution class ISSUE-112 and ISSUE-140 both exist to kill.
//
// Green both before and after the fix: it guards the INSERTION POINT rather than the
// new behaviour.
func TestPackVal_EngineStdoutArtifact_NeverStartedPrecedesArtifactRead(t *testing.T) {
	packDir := convertFixturePackDir(t)
	binding := engine.EngineBinding{
		Command:        filepath.Join(packDir, "engine-that-does-not-exist.sh"),
		StdoutArtifact: "results.sarif",
		Provision:      nil,
	}

	res, err := (&DefaultExecutor{}).RunEngine(packDir, binding, []string{"fixture.txt"})
	if err == nil {
		t.Fatalf("an engine whose process never started must fail loud, got nil error (result %+v)", res)
	}
	if res.Passed {
		t.Fatalf("expected a non-passing result alongside the error, got %+v", res)
	}
	if !strings.Contains(err.Error(), "never started") {
		t.Fatalf("the refusal must name the NEVER-STARTED condition; got: %v", err)
	}
	if strings.Contains(err.Error(), "stdout_artifact") {
		t.Fatalf("MISATTRIBUTION: a process that never ran produced no artifact, so the failure must not be "+
			"reported as a missing stdout_artifact — that sends a pack author to create a file when the real "+
			"problem is an unexecutable command; got: %v", err)
	}
	if strings.Contains(err.Error(), filepath.Join(packDir, "results.sarif")) {
		t.Fatalf("MISATTRIBUTION: the never-started refusal must not name the artifact path; got: %v", err)
	}
}

// stdoutArtifactGuardSite is one of the two payload-selection sites the drift guard
// windows by FUNCTION NAME.
type stdoutArtifactGuardSite struct {
	name string
	body string
}

// TestPackVal_EngineStdoutArtifact_NoDriftFromGateDispatch is the mechanical drift
// guard (CLM-008) — the payload-selection twin of PLAN-ISSUE-141's convert guard. Two
// call sites select the payload from a declared stdout_artifact: RunEngine here and
// runFindingsEngine in cmd/backstop/pack_gate.go. Consolidating them is ISSUE-143's
// subject; it is open and unplanned, and this guard holds the line until it is picked
// up.
//
// IT IS A CONTENT SCAN, NOT A TREE-STATE CHECK: no git status, no git diff, no
// working-tree-cleanliness assertion and no line numbers. This is a shared tree with
// concurrent lanes, and a tree-state assertion blames whoever happens to run it.
//
// THE pack_gate.go HALF IS WINDOWED TO runFindingsEngine. That file contains TWO
// independent payload-selection blocks — one in runCoverageEngine and one in
// runFindingsEngine — each carrying binding.StdoutArtifact and an os.ReadFile refusal,
// so a whole-file scan could not fail even if runFindingsEngine's block were deleted
// outright. cmd/backstop/pack_gate.go is READ-ONLY in this lane; the guard only reads
// it.
func TestPackVal_EngineStdoutArtifact_NoDriftFromGateDispatch(t *testing.T) {
	executorSrc := readGuardSource(t, filepath.Join("executor.go"))
	gateSrc := readGuardSource(t, filepath.Join("..", "..", "cmd", "backstop", "pack_gate.go"))

	sites := []stdoutArtifactGuardSite{
		{
			name: "pkg/packval/executor.go RunEngine",
			body: windowFunc(t, executorSrc, "func (d *DefaultExecutor) RunEngine("),
		},
		{
			name: "cmd/backstop/pack_gate.go runFindingsEngine",
			body: windowFunc(t, gateSrc, "func runFindingsEngine("),
		},
	}

	const drift = "the two payload-selection paths have DRIFTED. Both must read the binding's declared " +
		"stdout_artifact from its resolved path and refuse loudly when that file cannot be read, so a declared " +
		"artifact is never silently replaced by stdout. Consolidating them into one authority is ISSUE-143, " +
		"which is open and unplanned; until it lands, this guard is what holds the line"

	for _, site := range sites {
		if !strings.Contains(site.body, "binding.StdoutArtifact") {
			t.Fatalf("%s does not reference binding.StdoutArtifact: %s", site.name, drift)
		}
		if !strings.Contains(site.body, "os.ReadFile") {
			t.Fatalf("%s performs no os.ReadFile on the resolved artifact path: %s", site.name, drift)
		}
		if !strings.Contains(site.body, "readErr") {
			t.Fatalf("%s does not return an error on artifact read failure: %s", site.name, drift)
		}
	}
}
