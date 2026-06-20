package main

import (
	"context"
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/pack"
)

// lintRunner is a CommandRunner that returns the golangci v2 SARIF fixture ONLY
// through RunStdout (clean stdout), recording every RunStdout invocation so a
// test can assert the capture method and the absence of a version probe. Its Run
// (the CombinedOutput-shaped seam) returns nothing, so a lint path that captured
// via CombinedOutput instead of RunStdout would see EMPTY output and produce zero
// findings — making the RunStdout requirement (CLM-016) observable.
type lintRunner struct {
	sarif         []byte
	runStdoutErr  error
	runStdoutCall []fixtureCall
	runCall       []fixtureCall
}

func (r *lintRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	r.runCall = append(r.runCall, fixtureCall{name: name, args: append([]string(nil), args...)})
	return nil, nil
}

func (r *lintRunner) RunStdout(_ context.Context, name string, args ...string) ([]byte, error) {
	r.runStdoutCall = append(r.runStdoutCall, fixtureCall{name: name, args: append([]string(nil), args...)})
	return r.sarif, r.runStdoutErr
}

// TestGoLint_ConfigFileEngineNativeSarif proves lint runs golangci-lint as a
// config-file engine whose SARIF output parses directly through parseSarif with
// NO converter (CLM-015). The go-lint rule resolves to the golangci binding
// (InputModeConfigFile, empty Convert); dispatch parses the v2 SARIF fixture and
// emits the two normalized lint violations namespaced to the pack.
func TestGoLint_ConfigFileEngineNativeSarif(t *testing.T) {
	m := onlyRules(goToolchainManifest(t), "golangci")
	runner := &lintRunner{sarif: readFixture(t, "golangci-v2.sarif")}

	violations, err := dispatchPackEngines([]*pack.Manifest{m}, goToolchainPacksDir(t), t.TempDir(), nil, runner)
	if err != nil {
		t.Fatalf("dispatchPackEngines (lint): %v", err)
	}
	if len(violations) != 2 {
		t.Fatalf("expected 2 lint violations from native v2 SARIF, got %d: %#v", len(violations), violations)
	}
	if violations[0].File != "pkg/widget/widget.go" || violations[0].Message != "Error return value is not checked" {
		t.Errorf("lint violation not parsed from native v2 SARIF: %#v", violations[0])
	}
	if !strings.HasPrefix(violations[0].Rule, "backstop/go-toolchain/") {
		t.Errorf("lint violation must be namespaced to the pack, got %q", violations[0].Rule)
	}

	// No converter: the golangci binding declares no Convert, so the sandboxed
	// convert seam is never the source of the SARIF — the tool's stdout IS the
	// SARIF. Assert by confirming dispatch produced findings without a convert stub
	// being installed (sandboxedRunStdout is nil here; a Convert would have nil-pivoted
	// to the real sandbox and failed in this unit test).
	bind := resolveEngineRegistry()["golangci"]
	if bind.Convert != "" {
		t.Errorf("the golangci config-file engine must declare NO converter (v2 native SARIF), got Convert=%q", bind.Convert)
	}
}

// TestGoLint_SarifCapturedViaRunStdoutNotCombined proves the lint engine output
// is captured via RunStdout (clean stdout), NOT CombinedOutput, so a
// golangci-lint stderr banner cannot corrupt the SARIF (CLM-016, Sharp Edge 4).
// The lintRunner returns the SARIF only through RunStdout; a CombinedOutput-based
// capture would see empty output and yield zero findings.
func TestGoLint_SarifCapturedViaRunStdoutNotCombined(t *testing.T) {
	m := onlyRules(goToolchainManifest(t), "golangci")
	runner := &lintRunner{sarif: readFixture(t, "golangci-v2.sarif")}

	violations, err := dispatchPackEngines([]*pack.Manifest{m}, goToolchainPacksDir(t), t.TempDir(), nil, runner)
	if err != nil {
		t.Fatalf("dispatchPackEngines (lint): %v", err)
	}
	if len(violations) == 0 {
		t.Fatal("lint produced zero findings — the SARIF must be captured via RunStdout, not the empty CombinedOutput seam (Sharp Edge 4)")
	}
	if len(runner.runStdoutCall) == 0 {
		t.Fatal("lint engine never called RunStdout; it must capture the SARIF via the clean-stdout runner, not CombinedOutput")
	}
	if len(runner.runCall) != 0 {
		t.Errorf("lint engine used the CombinedOutput-shaped Run seam (%d calls); golangci SARIF must be captured via RunStdout only", len(runner.runCall))
	}
}

// TestGoLint_NoVersionProbeOrV1Branch proves the lint invocation assumes
// golangci-lint v2 SARIF and performs NO `golangci-lint version` probe or v1/v2
// flag branch (CLM-019). The invocation is a single `golangci-lint run ...` with
// the SARIF output flags; no `version` subcommand is ever invoked.
func TestGoLint_NoVersionProbeOrV1Branch(t *testing.T) {
	m := onlyRules(goToolchainManifest(t), "golangci")
	runner := &lintRunner{sarif: readFixture(t, "golangci-v2.sarif")}

	if _, err := dispatchPackEngines([]*pack.Manifest{m}, goToolchainPacksDir(t), t.TempDir(), nil, runner); err != nil {
		t.Fatalf("dispatchPackEngines (lint): %v", err)
	}

	for _, c := range append(append([]fixtureCall{}, runner.runStdoutCall...), runner.runCall...) {
		if c.name != "golangci-lint" {
			continue
		}
		for _, a := range c.args {
			if a == "version" {
				t.Errorf("lint must not probe `golangci-lint version`; the invocation assumes v2 SARIF (args=%v)", c.args)
			}
		}
	}
	if len(runner.runStdoutCall) != 1 {
		t.Fatalf("expected exactly one golangci-lint invocation (no version probe), got %d RunStdout calls", len(runner.runStdoutCall))
	}
	call := runner.runStdoutCall[0]
	if call.name != "golangci-lint" || len(call.args) == 0 || call.args[0] != "run" {
		t.Fatalf("lint invocation must be `golangci-lint run ...`, got %s %v", call.name, call.args)
	}
	// The v2 SARIF output flags must be present (no v1 `--out-format json` branch).
	joined := strings.Join(call.args, " ")
	if !strings.Contains(joined, "--output.sarif.path") {
		t.Errorf("lint invocation must use the v2 SARIF output flag, got args=%v", call.args)
	}
	if strings.Contains(joined, "--out-format") {
		t.Errorf("lint invocation must NOT use the v1 `--out-format` flag (no v1/v2 branch), got args=%v", call.args)
	}
}

// TestGoLint_V1JSONFailsLoudNotSilentZero proves the strict-SARIF guard: feeding
// the lint config-file engine golangci v1 JSON (`{"Issues":[...]}`, NOT SARIF)
// fails loud as unparseable-SARIF attributed to the engine, rather than the
// lenient SARIF parser silently reading zero findings — a vacuous green
// (REQ-005/CLM-019, Sharp Edge 5). This is the v1/too-old golangci fail-loud the
// generic findings path does not provide on its own.
func TestGoLint_V1JSONFailsLoudNotSilentZero(t *testing.T) {
	m := onlyRules(goToolchainManifest(t), "golangci")
	// v1 JSON is well-formed JSON but NOT SARIF; the lenient parser would yield
	// zero findings and pass green without the strict-SARIF guard.
	runner := &lintRunner{sarif: readFixture(t, "golangci-v1.json")}

	_, err := dispatchPackEngines([]*pack.Manifest{m}, goToolchainPacksDir(t), t.TempDir(), nil, runner)
	if err == nil {
		t.Fatal("expected a fail-loud error for golangci v1 JSON (not SARIF), got nil — that is a silent zero-findings green")
	}
	if !strings.Contains(err.Error(), "go-toolchain") && !strings.Contains(err.Error(), "golangci-lint") {
		t.Errorf("strict-SARIF failure must attribute the lint engine, got: %v", err)
	}
}
