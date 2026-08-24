package main

import (
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/pack"
	"github.com/backstop-ai/backstop-core/pkg/packval"
)

// TestEngineDispatch_SandboxNoneExitCodeViolationSkipsSarif proves the sandbox
// engine (input_mode none, no command) takes the exit-code terminal branch and
// NEVER enters convert/parseSarif (CLM-066 / REQ-014): a non-zero validator exit
// yields a namespaced violation whose message is the captured (non-SARIF) output,
// and the convert seam is never touched. Substantive: the validator output is
// deliberately NON-SARIF text that would fail parseSarif if the path ever reached
// it, so a passing test proves the sandbox branch is exit-code-only.
func TestEngineDispatch_SandboxNoneExitCodeViolationSkipsSarif(t *testing.T) {
	// Capture convert invocations; the sandbox branch must touch NONE.
	convertCalls, _ := recordingConvertSeam(t)

	// Inject the CombinedOutput sandbox seam: a non-zero exit carrying non-SARIF
	// output (a human message). If dispatch fed this into parseSarif, the parse
	// would fail; the sandbox branch instead surfaces it as the violation message.
	validatorOutput := "MARKER file missing under /project (this is NOT sarif)"
	sandboxRunner := &recordingSandboxRunner{mode: packval.SandboxModeNative, runFn: func(string, []string, string) (packval.SandboxRunResult, error) {
		return packval.SandboxRunResult{Output: []byte(validatorOutput), NativeSandboxApplied: true}, &fakeExitError{code: 1}
	}}

	manifest := &pack.Manifest{
		NormalizedName: "test-org/engine-pack",
		Content: pack.Content{Ruleset: pack.Ruleset{Rules: []pack.Rule{
			{ID: "sandbox-presence", Engine: "sandbox", Validator: "scripts/check-presence.sh", InputScope: "multi-file", Category: "presence"},
		}}},
	}

	result, err := dispatchPackEnginesWithEvidence([]*pack.Manifest{manifest}, engineDispatchPacksDir(t), t.TempDir(), nil, emptySarifRunner{}, sandboxRunner)
	if err != nil {
		t.Fatalf("sandbox engine must surface a non-zero exit as a violation, not an error: %v", err)
	}
	if len(result.Violations) != 1 || !result.NativeSandboxApplied {
		t.Fatalf("expected exactly 1 acknowledged sandbox violation, got %#v", result)
	}
	v := result.Violations[0]
	if v.Rule != "test-org/engine-pack/sandbox-presence" {
		t.Errorf("sandbox violation must be namespaced, got %q", v.Rule)
	}
	if v.Message != validatorOutput {
		t.Errorf("sandbox violation message must be the captured validator output, got %q", v.Message)
	}
	// The exit-code branch must NEVER enter convert/parseSarif.
	if len(*convertCalls) != 0 {
		t.Errorf("sandbox/none exit-code branch must skip convert entirely, got convert calls %v", *convertCalls)
	}
	// Cross-check: the message is genuinely non-SARIF, so a passing test means the
	// path did not (and could not) parse it as SARIF.
	if strings.Contains(v.Message, "\"version\"") {
		t.Error("test precondition: validator output must be non-SARIF to prove parseSarif is skipped")
	}
}

// TestEngineDispatch_SandboxNoneZeroExitNoViolation proves the boundary: a
// zero-exit sandbox validator yields no violation (the marker is present), so the
// exit-code semantics are real and not a constant-fail stub. (Supports CLM-066's
// exit-code-driven contract.)
func TestEngineDispatch_SandboxNoneZeroExitNoViolation(t *testing.T) {
	sandboxRunner := &recordingSandboxRunner{mode: packval.SandboxModeNative, runFn: func(string, []string, string) (packval.SandboxRunResult, error) {
		return packval.SandboxRunResult{}, nil
	}}

	manifest := &pack.Manifest{
		NormalizedName: "test-org/engine-pack",
		Content: pack.Content{Ruleset: pack.Ruleset{Rules: []pack.Rule{
			{ID: "sandbox-presence", Engine: "sandbox", Validator: "scripts/check-presence.sh", InputScope: "multi-file", Category: "presence"},
		}}},
	}
	result, err := dispatchPackEnginesWithEvidence([]*pack.Manifest{manifest}, engineDispatchPacksDir(t), t.TempDir(), nil, emptySarifRunner{}, sandboxRunner)
	if err != nil {
		t.Fatalf("dispatchPackEngines: %v", err)
	}
	if len(result.Violations) != 0 {
		t.Fatalf("a zero-exit sandbox validator must yield no violations, got %#v", result.Violations)
	}
}
