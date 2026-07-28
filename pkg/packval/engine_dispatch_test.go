package packval

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/pack/engine"
)

// readSource reads a packval source file for deletion/absence assertions.
func readSource(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(name)
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return string(data)
}

// TestExecutor_RunEngineDispatchesFromBinding (CLM-002): the argv RunEngine builds
// comes ENTIRELY from the pack-declared binding DATA (command + input_flag) with the
// targets appended — never a hardcoded tool name.
func TestExecutor_RunEngineDispatchesFromBinding(t *testing.T) {
	binding := engine.EngineBinding{
		Command:   "toolx --scan --sarif",
		InputMode: engine.InputModeRuleFlags,
		InputFlag: "--config",
	}
	name, args := buildEngineArgv(binding, []string{"rules/r.yml", "fixtures/p.txt"})
	if name != "toolx" {
		t.Fatalf("engine name must come from binding.Command, got %q", name)
	}
	want := []string{"--scan", "--sarif", "--config", "rules/r.yml", "fixtures/p.txt"}
	if strings.Join(args, " ") != strings.Join(want, " ") {
		t.Fatalf("argv from binding wrong:\n got %v\nwant %v", args, want)
	}
	// No baked tool name may appear in the constructed argv.
	joined := name + " " + strings.Join(args, " ")
	for _, banned := range []string{"semgrep", "golangci-lint"} {
		if strings.Contains(joined, banned) {
			t.Fatalf("constructed argv leaked a baked tool literal %q: %s", banned, joined)
		}
	}
}

// TestExecutor_RunEngineGatedByAllowlist (CLM-002, CLM-009): a binding whose
// provisioned tool is NOT on the trusted-tool allowlist is NEVER executed —
// RunEngine fail-louds from engine.CheckToolAllowed before touching the runner.
func TestExecutor_RunEngineGatedByAllowlist(t *testing.T) {
	d := &DefaultExecutor{}
	binding := engine.EngineBinding{
		Command:   "definitely-not-a-real-binary --go",
		InputMode: engine.InputModeRuleFlags,
		InputFlag: "--config",
		Provision: &engine.Provision{Tool: "definitely-not-a-real-binary", Version: "9.9.9"},
	}
	_, err := d.RunEngine(t.TempDir(), binding, []string{"x"})
	if err == nil {
		t.Fatal("expected fail-loud error for un-allowlisted provisioned tool")
	}
	if !strings.Contains(err.Error(), "allowlist") {
		t.Fatalf("expected allowlist trust-floor error, got %q", err.Error())
	}
}

// TestExecutor_NoRunSemgrepMethod (CLM-004): the tool-named RunSemgrep method is
// gone from the interface, DefaultExecutor, and MockExecutor.
func TestExecutor_NoRunSemgrepMethod(t *testing.T) {
	src := readSource(t, "executor.go")
	if strings.Contains(src, "RunSemgrep") {
		t.Fatal("RunSemgrep must be removed from executor.go (generic RunEngine replaces it)")
	}
	if strings.Contains(src, "SemgrepFn") {
		t.Fatal("MockExecutor.SemgrepFn must be removed in favor of EngineFn")
	}
	// The interface must expose the generic RunEngine seam.
	if !strings.Contains(src, "RunEngine(") {
		t.Fatal("executor.go must declare the generic RunEngine method")
	}
}

// TestExecutor_ToolConfigResolvesViaBindingNotSwitch (CLM-003): no baked tool-name
// switch or exec.Command tool literal remains; dispatch is generic.
func TestExecutor_ToolConfigResolvesViaBindingNotSwitch(t *testing.T) {
	src := readSource(t, "executor.go")
	for _, banned := range []string{
		`case "golangci-lint"`,
		`exec.Command("golangci-lint"`,
		`exec.Command("semgrep"`,
	} {
		if strings.Contains(src, banned) {
			t.Fatalf("executor.go still contains baked tool literal: %s", banned)
		}
	}
	ph3 := readSource(t, "phase3.go")
	for _, banned := range []string{`"semgrep"`, `"golangci-lint"`} {
		if strings.Contains(ph3, banned) {
			t.Fatalf("phase3.go still contains baked tool literal: %s", banned)
		}
	}
}

// TestPhase3_ResolvesRuleEngineFromRegistry (CLM-005, CLM-009): RunFixtures resolves
// a rule's declared engine against baseengines.Registry() MERGED with the pack's
// engines: block, a pack-declared engine winning over the same-named base binding,
// and dispatches the resolved binding through RunEngine.
func TestPhase3_ResolvesRuleEngineFromRegistry(t *testing.T) {
	// A pack-declared "semgrep" override with a marker input_flag, distinct from the
	// base semgrep binding, proves precedence and that the resolver was consulted.
	pack := &PackManifest{
		Name:      "acme/x",
		Version:   "1.0.0",
		Language:  "generic",
		Archetype: "enforcement",
		Engines: map[string]engine.EngineBinding{
			"semgrep": {
				Command:   "semgrep --sarif --quiet",
				InputMode: engine.InputModeRuleFlags,
				InputFlag: "--MARKER-pack-declared-wins",
			},
		},
		Content: Content{Ruleset: Ruleset{Rules: []Rule{{
			ID: "R1", Engine: "semgrep", File: "rules/r.yml", RiskClass: "correctness",
			Claims: []Claim{{ID: "C1", Fixtures: Fixtures{
				Positive: []FixtureRef{{Path: "fixtures/p.txt"}},
			}}},
		}}}},
	}
	dir := t.TempDir()
	writeSrc(t, dir, "rules/r.yml", "rules:\n  - id: R1\n")
	writeSrc(t, dir, "fixtures/p.txt", "x")

	var captured engine.EngineBinding
	var calls int
	mock := &MockExecutor{EngineFn: func(_ string, b engine.EngineBinding, _ []string) (ExecutionResult, error) {
		captured = b
		calls++
		return ExecutionResult{Passed: true}, nil
	}}
	RunFixtures(pack, dir, mock)
	if calls == 0 {
		t.Fatal("RunFixtures never dispatched through RunEngine")
	}
	if captured.InputFlag != "--MARKER-pack-declared-wins" {
		t.Fatalf("resolver did not prefer pack-declared engine; got InputFlag %q", captured.InputFlag)
	}
}

// TestPhase3_UnknownEngineFailsLoud (CLM-009, CLM-010): a rule naming an engine that
// is in NEITHER the base registry NOR the pack's engines: block fails loud through
// the REAL DefaultExecutor — never a silent pass.
func TestPhase3_UnknownEngineFailsLoud(t *testing.T) {
	dir, err := filepath.Abs("testdata/engine-unknown-pack")
	if err != nil {
		t.Fatal(err)
	}
	pack, err := ParseManifest(filepath.Join(dir, "pack.yml"))
	if err != nil {
		t.Fatalf("parse engine-unknown pack: %v", err)
	}
	res := RunFixtures(pack, dir, &DefaultExecutor{})
	if res.Status != "fail" {
		t.Fatalf("unknown engine must fail loud, got status %q", res.Status)
	}
	found := false
	for _, e := range res.Errors {
		if strings.Contains(e.Message, "no-such-engine") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected an error naming the unknown engine; errors: %+v", res.Errors)
	}
}

// writeSrc is a local file-writing helper (this file is package packval, so it
// cannot use the _test package writeFile helper).
func writeSrc(t *testing.T, root, rel, data string) {
	t.Helper()
	full := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
}
