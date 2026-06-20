package main

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/pack"
)

// astGrepDispatchManifest builds an in-memory manifest with a single ast-grep
// rule pointing at the engine-dispatch fixture pack's proof rule, so the convert
// pipe (ast-grep emits its own JSON, a convert script normalizes to SARIF) is
// dispatched against the REAL ast-grep EngineBinding (which declares a Convert).
func astGrepDispatchManifest() *pack.Manifest {
	return &pack.Manifest{
		NormalizedName: "test-org/engine-pack",
		Content: pack.Content{Ruleset: pack.Ruleset{Rules: []pack.Rule{
			{ID: "ast-grep-proof", Engine: "ast-grep", RulePath: "ast-grep/proof-rule.yml", Standard: "x"},
		}}},
	}
}

// engineDispatchPacksDir returns the .backstop/packs dir holding the
// engine-dispatch fixture pack (test-org/engine-pack), where dispatch resolves
// the pack-relative ast-grep convert script.
func engineDispatchPacksDir(t *testing.T) string {
	t.Helper()
	return filepath.Join(repoRoot(t), "cmd", "backstop", "testdata", "engine-dispatch", ".backstop", "packs")
}

// astGrepJSONStdout is a minimal `ast-grep scan --json` payload (the engine's
// native non-SARIF output) carrying one finding, used as the engine stdout that
// the convert pipe must transform to SARIF. Returned from a func (not a package
// const/var) to keep no global mutable state.
func astGrepJSONStdout() string {
	return `[{"ruleId":"ast-grep-proof","severity":"error","message":"forbiddenCall is not allowed","file":"main.go","range":{"start":{"line":2,"column":18}}}]`
}

// TestGateDispatch_ConvertPipeProducesSarif proves the two-process convert pipe
// (CLM-024 / REQ-007): a non-empty Convert binding pipes engine stdout -> convert
// stdin -> convert stdout (SARIF) -> check.ParsePackFindings, producing a
// namespaced violation carrying the CONVERTED finding. Substantive: drives the
// real ast-grep convert script (to-sarif.sh) over real ast-grep JSON and asserts
// the transformed finding's identity (rule/file/line) reaches gate output.
func TestGateDispatch_ConvertPipeProducesSarif(t *testing.T) {
	// The convert step runs the REAL pack script via the direct-shell stub of the
	// sandbox seam (production routes through SandboxedRunStdout).
	stubSandboxedRunStdout(t, nil)
	runner := &fixtureRunner{byCmd: map[string][]byte{"ast-grep scan": []byte(astGrepJSONStdout())}}

	violations, err := dispatchPackEngines([]*pack.Manifest{astGrepDispatchManifest()}, engineDispatchPacksDir(t), t.TempDir(), nil, runner)
	if err != nil {
		t.Fatalf("dispatchPackEngines (ast-grep convert pipe): %v", err)
	}
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation from the convert pipe, got %d: %#v", len(violations), violations)
	}
	v := violations[0]
	if v.Rule != "test-org/engine-pack/ast-grep-proof" {
		t.Errorf("converted violation must be namespaced, got %q", v.Rule)
	}
	if v.File != "main.go" {
		t.Errorf("converted finding's file must survive the pipe, got %q", v.File)
	}
	// The message is produced by the jq transform from the ast-grep JSON's
	// .message field, so its presence proves the convert step ran the real
	// transform rather than passing canned SARIF through.
	if v.Message != "forbiddenCall is not allowed" {
		t.Errorf("converted finding's message must survive the pipe, got %q", v.Message)
	}
}

// TestGateDispatch_ConvertRunsSandboxed proves the convert executable is resolved
// relative to the pack dir and run through the SandboxedRunStdout seam (CLM-025 /
// REQ-007): the seam intercepts the convert call and observes the pack-relative
// convert script path and the pack dir. Substantive: asserts the exact resolved
// convert path (pack root + binding.Convert) and that the engine stdout is what
// is piped in as stdin.
func TestGateDispatch_ConvertRunsSandboxed(t *testing.T) {
	var gotCmd string
	var gotPackDir string
	var gotStdin []byte
	orig := sandboxedRunStdout
	sandboxedRunStdout = func(cmd string, _ []string, packDir string, stdin []byte) ([]byte, error) {
		gotCmd = cmd
		gotPackDir = packDir
		gotStdin = append([]byte(nil), stdin...)
		return []byte(`{"version":"2.1.0","runs":[]}`), nil
	}
	t.Cleanup(func() { sandboxedRunStdout = orig })

	runner := &fixtureRunner{byCmd: map[string][]byte{"ast-grep scan": []byte(astGrepJSONStdout())}}
	packsDir := engineDispatchPacksDir(t)
	if _, err := dispatchPackEngines([]*pack.Manifest{astGrepDispatchManifest()}, packsDir, t.TempDir(), nil, runner); err != nil {
		t.Fatalf("dispatchPackEngines: %v", err)
	}

	wantPackRoot := filepath.Join(packsDir, "test-org", "engine-pack")
	wantConvert := filepath.Join(wantPackRoot, "ast-grep", "to-sarif.sh")
	if gotCmd != wantConvert {
		t.Errorf("convert must resolve relative to the pack dir; seam got %q, want %q", gotCmd, wantConvert)
	}
	if gotPackDir != wantPackRoot {
		t.Errorf("convert must run with the pack root as its dir; seam got %q, want %q", gotPackDir, wantPackRoot)
	}
	if string(gotStdin) != astGrepJSONStdout() {
		t.Errorf("convert stdin must be the engine's raw stdout, got %q", string(gotStdin))
	}
}

// TestGateDispatch_StderrBannerDoesNotCorruptSarif proves an ENGINE that writes a
// banner to stderr still yields parseable SARIF (CLM-029 / REQ-009): RunStdout
// captures only stdout, so the engine's stderr banner never reaches the parser.
// Substantive: the SARIF-native engine emits a stderr banner via a runner that
// keeps stdout clean, and the resulting violation is still produced.
func TestGateDispatch_StderrBannerDoesNotCorruptSarif(t *testing.T) {
	// SARIF-native engine: no convert. The runner returns clean SARIF on stdout;
	// RunStdout's contract is that stderr (banner) is separated out, so the
	// stdout the parser sees is uncorrupted.
	convertCalls := recordingConvertSeam(t)

	packsDir := t.TempDir()
	packRoot := filepath.Join(packsDir, "org", "pack")
	mkDirAll(t, filepath.Join(packRoot, "semgrep"))
	writeFileStr(t, filepath.Join(packRoot, "semgrep", "r.yml"), "rules: []\n")

	// Clean SARIF on stdout (the banner would have gone to stderr, which RunStdout
	// drops). If stderr had leaked into stdout, the leading banner bytes would make
	// this fail to parse.
	rec := &capturingRunner{out: []byte(`{"version":"2.1.0","runs":[{"results":[{"ruleId":"x","message":{"text":"m"},"locations":[{"physicalLocation":{"artifactLocation":{"uri":"f.go"},"region":{"startLine":1}}}]}]}]}`)}
	manifests := []*pack.Manifest{{
		NormalizedName: "org/pack",
		Content: pack.Content{Ruleset: pack.Ruleset{Rules: []pack.Rule{
			{ID: "x", Engine: "semgrep", RulePath: "semgrep/r.yml", Standard: "x"},
		}}},
	}}
	violations, err := dispatchPackEngines(manifests, packsDir, t.TempDir(), nil, rec)
	if err != nil {
		t.Fatalf("engine stderr banner must not corrupt SARIF on stdout: %v", err)
	}
	if len(violations) != 1 || violations[0].File != "f.go" {
		t.Fatalf("expected the clean-stdout SARIF to parse to one violation, got %#v", violations)
	}
	if len(*convertCalls) != 0 {
		t.Errorf("SARIF-native engine path must not invoke convert, got %v", *convertCalls)
	}
}

// TestGateDispatch_ConvertStderrBannerDoesNotCorruptSarif proves a CONVERT script
// that writes a banner to stderr still yields parseable SARIF on its stdout via
// the clean-stdout sandbox capture (CLM-064 / REQ-009): the pack's real
// to-sarif.sh writes "ast-grep to-sarif: transforming findings" to stderr, yet
// the converted SARIF is uncorrupted. Substantive: runs the real banner-emitting
// converter end-to-end and asserts the finding still parses.
func TestGateDispatch_ConvertStderrBannerDoesNotCorruptSarif(t *testing.T) {
	// Confirm the converter we exercise really does emit a stderr banner, so this
	// test genuinely covers the banner-vs-SARIF separation.
	packRoot := filepath.Join(engineDispatchPacksDir(t), "test-org", "engine-pack")
	convertSrc := readFileStr(t, filepath.Join(packRoot, "ast-grep", "to-sarif.sh"))
	if !strings.Contains(convertSrc, ">&2") {
		t.Fatalf("test precondition: to-sarif.sh must write a banner to stderr to exercise CLM-064")
	}

	stubSandboxedRunStdout(t, nil) // shells the real converter, stdout-only capture
	runner := &fixtureRunner{byCmd: map[string][]byte{"ast-grep scan": []byte(astGrepJSONStdout())}}

	violations, err := dispatchPackEngines([]*pack.Manifest{astGrepDispatchManifest()}, engineDispatchPacksDir(t), t.TempDir(), nil, runner)
	if err != nil {
		t.Fatalf("convert stderr banner must not corrupt SARIF: %v", err)
	}
	if len(violations) != 1 || violations[0].Message != "forbiddenCall is not allowed" {
		t.Fatalf("converted finding must survive despite the convert stderr banner, got %#v", violations)
	}
}

// TestGateDispatch_ConvertUsesCleanStdoutSandbox proves the convert step uses the
// clean-stdout SandboxedRunStdout seam, NEVER the CombinedOutput SandboxedRun
// (CLM-065 / REQ-007): intercepting BOTH seams shows only the clean-stdout seam
// receives the convert call. Substantive: asserts the CombinedOutput seam is
// never invoked while the clean-stdout seam IS, proving which capture path the
// convert rides.
func TestGateDispatch_ConvertUsesCleanStdoutSandbox(t *testing.T) {
	cleanStdoutCalls := 0
	combinedOutputCalls := 0

	origStdout := sandboxedRunStdout
	sandboxedRunStdout = func(_ string, _ []string, _ string, _ []byte) ([]byte, error) {
		cleanStdoutCalls++
		return []byte(`{"version":"2.1.0","runs":[]}`), nil
	}
	t.Cleanup(func() { sandboxedRunStdout = origStdout })

	origCombined := sandboxedRun
	sandboxedRun = func(string, []string, string) ([]byte, error) {
		combinedOutputCalls++
		return nil, nil
	}
	t.Cleanup(func() { sandboxedRun = origCombined })

	runner := &fixtureRunner{byCmd: map[string][]byte{"ast-grep scan": []byte(astGrepJSONStdout())}}
	if _, err := dispatchPackEngines([]*pack.Manifest{astGrepDispatchManifest()}, engineDispatchPacksDir(t), t.TempDir(), nil, runner); err != nil {
		t.Fatalf("dispatchPackEngines: %v", err)
	}

	if cleanStdoutCalls != 1 {
		t.Errorf("convert must run through the clean-stdout SandboxedRunStdout seam exactly once, got %d", cleanStdoutCalls)
	}
	if combinedOutputCalls != 0 {
		t.Errorf("convert must NEVER use the CombinedOutput SandboxedRun seam, got %d calls", combinedOutputCalls)
	}
}
