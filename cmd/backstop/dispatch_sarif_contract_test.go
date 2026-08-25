package main

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/pack"
	"github.com/backstop-ai/backstop-core/pkg/packval"
)

// recordingConvertSeam installs a sandboxedRunStdout seam that records every
// convert invocation (the convert script path) for the duration of a test, so a
// test can assert whether the convert pipe ran at all. It returns the canned
// SARIF on stdout so any convert that DOES run still yields a parseable result.
// It is the proof seam for "empty Convert => no second process" (CLM-026/CLM-065).
func recordingConvertSeam(t *testing.T) (*[]string, *recordingSandboxRunner) {
	t.Helper()
	calls := &[]string{}
	runner := &recordingSandboxRunner{mode: packval.SandboxModeNative, stdoutFn: func(cmd string, _ []string, _ string, _ []byte) (packval.SandboxRunResult, error) {
		*calls = append(*calls, cmd)
		return packval.SandboxRunResult{Output: []byte(`{"version":"2.1.0","runs":[]}`)}, nil
	}}
	return calls, runner
}

// TestGateDispatch_SarifNativeNoConvert proves the findings-engine SARIF contract
// for a SARIF-native engine (CLM-022 / REQ-006): a binding with an EMPTY Convert
// sends the engine's stdout straight to check.ParsePackFindings, so native SARIF
// becomes a real namespaced violation with no convert step. Substantive: asserts
// the violation's namespaced rule id, file, and message all reach gate output
// from the engine's raw SARIF.
func TestGateDispatch_SarifNativeNoConvert(t *testing.T) {
	// semgrep is the built-in SARIF-native engine (Convert == ""). Confirm the
	// binding really carries no convert script, so this test asserts the
	// no-convert path and not an accidental convert.
	if b := resolveEngineRegistry(nil)["semgrep"]; b.Convert != "" {
		t.Fatalf("test precondition: semgrep must be SARIF-native (empty Convert), got %q", b.Convert)
	}
	// Inject a convert seam that records calls; for the empty-Convert path it must
	// never be touched.
	convertCalls, sandboxRunner := recordingConvertSeam(t)

	packsDir := t.TempDir()
	packRoot := filepath.Join(packsDir, "org", "pack")
	mkDirAll(t, filepath.Join(packRoot, "semgrep"))
	writeFileStr(t, filepath.Join(packRoot, "semgrep", "r.yml"), "rules: []\n")

	// Native SARIF straight from the engine (no convert needed).
	rec := &capturingRunner{out: []byte(`{"version":"2.1.0","runs":[{"results":[{"ruleId":"no-eval","message":{"text":"eval forbidden"},"locations":[{"physicalLocation":{"artifactLocation":{"uri":"danger.go"},"region":{"startLine":9}}}]}]}]}`)}
	manifests := []*pack.Manifest{{
		NormalizedName: "org/pack",
		Content: pack.Content{Ruleset: pack.Ruleset{Rules: []pack.Rule{
			{ID: "no-eval", Engine: "semgrep", RulePath: "semgrep/r.yml", Standard: "x"},
		}}},
	}}

	result, err := dispatchPackEnginesWithEvidence(manifests, packsDir, t.TempDir(), nil, rec, sandboxRunner)
	violations := result.Violations
	if err != nil {
		t.Fatalf("dispatchPackEngines: %v", err)
	}
	if len(violations) != 1 {
		t.Fatalf("expected exactly 1 violation from native SARIF, got %d: %#v", len(violations), violations)
	}
	v := violations[0]
	if v.Rule != "org/pack/no-eval" {
		t.Errorf("native SARIF violation must be namespaced, got %q", v.Rule)
	}
	if v.File != "danger.go" || v.Message != "eval forbidden" {
		t.Errorf("native SARIF fields must reach gate output unchanged, got %#v", v)
	}
	if len(*convertCalls) != 0 {
		t.Errorf("SARIF-native engine (empty Convert) must NOT run any convert step, got convert calls %v", *convertCalls)
	}
}

// TestGateDispatch_NonSarifWithoutConvertFails proves the SARIF contract is
// enforced, not assumed (CLM-023 / REQ-006): when an engine with an EMPTY Convert
// emits stdout that is NOT SARIF, the bytes reach check.ParsePackFindings and the
// dispatch FAILS LOUD with an error attributed to the pack/engine — never a
// silent zero-finding green. Substantive: asserts a non-nil error, that it names
// the pack, and that zero violations are returned (no vacuous green).
func TestGateDispatch_NonSarifWithoutConvertFails(t *testing.T) {
	packsDir := t.TempDir()
	packRoot := filepath.Join(packsDir, "org", "pack")
	mkDirAll(t, filepath.Join(packRoot, "semgrep"))
	writeFileStr(t, filepath.Join(packRoot, "semgrep", "r.yml"), "rules: []\n")

	// Engine stdout that is NOT SARIF (a raw human banner) with an empty Convert:
	// the contract says this must fail loud at parseSarif, not read as zero findings.
	rec := &capturingRunner{out: []byte("error: something went wrong\nnot json at all\n")}
	manifests := []*pack.Manifest{{
		NormalizedName: "org/pack",
		Content: pack.Content{Ruleset: pack.Ruleset{Rules: []pack.Rule{
			{ID: "r", Engine: "semgrep", RulePath: "semgrep/r.yml", Standard: "x"},
		}}},
	}}

	violations, err := dispatchPackEngines(manifests, packsDir, t.TempDir(), nil, rec)
	if err == nil {
		t.Fatalf("non-SARIF engine stdout with empty Convert must fail loud, got nil error and violations %#v", violations)
	}
	if len(violations) != 0 {
		t.Errorf("a fail-loud parse error must not also return violations, got %#v", violations)
	}
	if !strings.Contains(err.Error(), "org/pack") {
		t.Errorf("parse error must be attributed to the pack, got: %v", err)
	}
	if !strings.Contains(strings.ToLower(err.Error()), "sarif") {
		t.Errorf("error must name the SARIF parse failure (the contract), got: %v", err)
	}
}

// TestGateDispatch_EmptyConvertNoPipe proves that when a binding declares no
// Convert, dispatch invokes NO convert executable at all (CLM-026 / REQ-007): the
// recording convert seam observes zero invocations across a full dispatch of a
// SARIF-native engine. Substantive: directly asserts the absence of a second
// process on the empty-Convert path (not merely that a violation appeared).
func TestGateDispatch_EmptyConvertNoPipe(t *testing.T) {
	convertCalls, sandboxRunner := recordingConvertSeam(t)

	packsDir := t.TempDir()
	packRoot := filepath.Join(packsDir, "org", "pack")
	mkDirAll(t, filepath.Join(packRoot, "semgrep"))
	writeFileStr(t, filepath.Join(packRoot, "semgrep", "r.yml"), "rules: []\n")

	rec := &capturingRunner{out: []byte(`{"version":"2.1.0","runs":[]}`)}
	manifests := []*pack.Manifest{{
		NormalizedName: "org/pack",
		Content: pack.Content{Ruleset: pack.Ruleset{Rules: []pack.Rule{
			{ID: "r", Engine: "semgrep", RulePath: "semgrep/r.yml", Standard: "x"},
		}}},
	}}

	if _, err := dispatchPackEnginesWithEvidence(manifests, packsDir, t.TempDir(), nil, rec, sandboxRunner); err != nil {
		t.Fatalf("dispatchPackEngines: %v", err)
	}
	if len(*convertCalls) != 0 {
		t.Fatalf("empty Convert must pipe through NO convert executable, got %d convert invocation(s): %v", len(*convertCalls), *convertCalls)
	}
	// And the engine command itself DID run (so the zero-convert result is real
	// dispatch, not a skipped pack).
	if rec.calls != 1 {
		t.Errorf("expected the SARIF-native engine to run once, got %d invocations", rec.calls)
	}
}
