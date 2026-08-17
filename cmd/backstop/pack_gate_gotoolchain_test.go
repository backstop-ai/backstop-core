package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/pack"
	"github.com/backstop-ai/backstop-core/pkg/pack/engine"
)

// goToolchainPackRoot returns the absolute path to the in-worktree go-toolchain
// fixture pack directory (the reusable mechanism pack scaffolded in TASK-001).
func goToolchainPackRoot(t *testing.T) string {
	t.Helper()
	return filepath.Join(repoRoot(t), "cmd", "backstop", "testdata", "go-toolchain",
		".backstop", "packs", "backstop", "go-toolchain")
}

// goToolchainPacksDir returns the .backstop/packs dir holding the go-toolchain pack.
func goToolchainPacksDir(t *testing.T) string {
	t.Helper()
	return filepath.Join(repoRoot(t), "cmd", "backstop", "testdata", "go-toolchain", ".backstop", "packs")
}

// readFixture reads a shared captured-output fixture or fails.
func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	p := filepath.Join(repoRoot(t), "cmd", "backstop", "testdata", "go-toolchain", "fixtures", name)
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return b
}

// fixtureRunner is a CommandRunner whose RunStdout returns canned bytes keyed by
// the command name, recording every invocation's name+args so a test can assert
// the scope-kind-aware arg-shaping (build/test get `./...`, NOT projectRoot).
type fixtureRunner struct {
	byCmd    map[string][]byte
	byCmdErr map[string]error
	calls    []fixtureCall
}

type fixtureCall struct {
	name string
	args []string
}

func (r *fixtureRunner) Run(context.Context, string, ...string) ([]byte, error) { return nil, nil }

// producerCommandAlias maps a pack-declared PRODUCER SCRIPT'S BASENAME to the
// command key its canned output is registered under (ISSUE-067).
//
// WHY THE HARNESS NEEDS THIS. Declaring a producer changes the NAME the dispatch
// invokes: core swaps argv[0] for the packRoot-resolved script path, so a large
// existing corpus that keys canned output on `byCmd["go test"]` / `byCmd["go
// build"]` would miss on every lookup and get nil back — silently observing ZERO
// violations. Assertions that pass by GOING QUIET are the exact vacuous green this
// issue is about, so the indirection is absorbed HERE, in one place, rather than
// by re-keying (or deleting the assertions of) each affected file.
//
// TestFixtureRunner_ProducerAliasCoversEveryDeclaredProducer pins this table against
// the real fixture manifest, so adding a findings producer to the pack without
// teaching this double fails loudly instead of going quiet.
//
// It is a FUNCTION returning a fresh map, not a package-level var: the go-standards
// no-global-mutable-state rule forbids package-level mutable state, and a shared map
// is state one test could mutate out from under another.
func producerCommandAlias() map[string]string {
	return map[string]string{
		"test-produce.sh":  "go test",
		"build-produce.sh": "go build",
	}
}

func (r *fixtureRunner) RunStdout(_ context.Context, name string, args ...string) ([]byte, error) {
	r.calls = append(r.calls, fixtureCall{name: name, args: append([]string(nil), args...)})
	// A producer path resolves back to the command key it stands for, so every
	// existing registration keeps working through the producer indirection.
	if alias, aliased := producerCommandAlias()[filepath.Base(name)]; aliased {
		if out, ok := r.byCmd[alias]; ok {
			return out, r.byCmdErr[alias]
		}
	}
	key := name
	if len(args) > 0 {
		key = name + " " + args[0]
	}
	if out, ok := r.byCmd[key]; ok {
		return out, r.byCmdErr[key]
	}
	if out, ok := r.byCmd[name]; ok {
		return out, r.byCmdErr[name]
	}
	return nil, nil
}

// stubConvertStdin installs a sandboxedRunStdout stub that, instead of running
// the sandbox, executes the convert script's transform on the provided stdin by
// shelling the real script directly via /bin/sh — proving the convert pipe is
// exercised without a live sandbox. It records the stdin it received.
func stubSandboxedRunStdout(t *testing.T, gotStdin *[]byte) {
	t.Helper()
	orig := sandboxedRunStdout
	sandboxedRunStdout = func(cmd string, args []string, packDir string, stdin []byte) ([]byte, error) {
		if gotStdin != nil {
			*gotStdin = append([]byte(nil), stdin...)
		}
		return runConvertScriptDirect(cmd, stdin)
	}
	t.Cleanup(func() { sandboxedRunStdout = orig })
}

// goToolchainManifest loads the real go-toolchain pack manifest, skipping if the
// engine bindings it declares are not yet registered (TDD red before impl).
func goToolchainManifest(t *testing.T) *pack.Manifest {
	t.Helper()
	m, err := pack.ParseManifestFile(filepath.Join(goToolchainPackRoot(t), "pack.yml"))
	if err != nil {
		t.Fatalf("go-toolchain pack must parse (its engines must be registered): %v", err)
	}
	return m
}

// onlyRules returns a shallow copy of the manifest keeping only rules whose
// engine is in keep, so a test can drive one pass in isolation.
func onlyRules(m *pack.Manifest, keep ...string) *pack.Manifest {
	set := map[string]bool{}
	for _, k := range keep {
		set[k] = true
	}
	cp := *m
	var rules []pack.Rule
	for _, r := range m.Content.Ruleset.Rules {
		if set[r.Engine] {
			rules = append(rules, r)
		}
	}
	cp.Content.Ruleset.Rules = rules
	return &cp
}

// TestGoToolchain_BuildFindingsEngineWithConvert proves build is a go-toolchain
// findings engine whose pack-relative convert script normalizes `go build`
// output to SARIF via runFindingsEngine (CLM-011).
func TestGoToolchain_BuildFindingsEngineWithConvert(t *testing.T) {
	m := onlyRules(goToolchainManifest(t), "go-build")
	stubSandboxedRunStdout(t, nil)
	runner := &fixtureRunner{byCmd: map[string][]byte{"go build": readFixture(t, "go-build-errors.txt")}}

	violations, err := dispatchPackEngines([]*pack.Manifest{m}, goToolchainPacksDir(t), t.TempDir(), nil, runner)
	if err != nil {
		t.Fatalf("dispatchPackEngines (build): %v", err)
	}
	if len(violations) != 3 {
		t.Fatalf("expected 3 build violations from the convert, got %d: %#v", len(violations), violations)
	}
	if violations[0].File != "pkg/widget/widget.go" || violations[0].Message != "undefined: Frobnicate" {
		t.Errorf("build violation not normalized via convert: %#v", violations[0])
	}
	if !strings.HasPrefix(violations[0].Rule, "backstop/go-toolchain/") {
		t.Errorf("build violation must be namespaced to the pack, got %q", violations[0].Rule)
	}
}

// TestGoToolchain_TestFindingsEngineWithConvert proves test is a go-toolchain
// findings engine with a convert script via runFindingsEngine (CLM-012).
func TestGoToolchain_TestFindingsEngineWithConvert(t *testing.T) {
	m := onlyRules(goToolchainManifest(t), "go-test")
	stubSandboxedRunStdout(t, nil)
	runner := &fixtureRunner{byCmd: map[string][]byte{"go test": readFixture(t, "go-test-failures.txt")}}

	violations, err := dispatchPackEngines([]*pack.Manifest{m}, goToolchainPacksDir(t), t.TempDir(), nil, runner)
	if err != nil {
		t.Fatalf("dispatchPackEngines (test): %v", err)
	}
	if len(violations) != 3 {
		t.Fatalf("expected 3 test violations from the convert, got %d: %#v", len(violations), violations)
	}
	if violations[0].Message != "TestWidgetFrobnicate: expected 5, got 7" {
		t.Errorf("test violation not normalized via convert: %#v", violations[0])
	}
}

// TestGoToolchain_ConvertUsesSandboxedRunStdout proves the convert step runs via
// the sandboxed clean-stdout capture (CLM-013): a converter's stderr banner
// cannot corrupt the SARIF because the bytes flow through SandboxedRunStdout.
func TestGoToolchain_ConvertUsesSandboxedRunStdout(t *testing.T) {
	m := onlyRules(goToolchainManifest(t), "go-build")
	var gotStdin []byte
	stubSandboxedRunStdout(t, &gotStdin)
	raw := readFixture(t, "go-build-errors.txt")
	runner := &fixtureRunner{byCmd: map[string][]byte{"go build": raw}}

	if _, err := dispatchPackEngines([]*pack.Manifest{m}, goToolchainPacksDir(t), t.TempDir(), nil, runner); err != nil {
		t.Fatalf("dispatchPackEngines: %v", err)
	}
	if string(gotStdin) != string(raw) {
		t.Fatalf("convert did not receive the tool's raw stdout via SandboxedRunStdout; got %q", string(gotStdin))
	}
}

// TestGoToolchain_BuildTestCrashNotSilentGreen proves a build/test run that
// exits non-zero with NO parseable findings surfaces as a crash on the convert
// path, not a silent green (CLM-010). N1: runFindingsEngine today discards
// runErr — this guard is the new behavior.
func TestGoToolchain_BuildTestCrashNotSilentGreen(t *testing.T) {
	m := onlyRules(goToolchainManifest(t), "go-build")
	stubSandboxedRunStdout(t, nil)
	// A crash: non-zero exit with output that yields no parseable compiler errors.
	crash := []byte("go: cannot find main module; see 'go help modules'\n")
	runner := &fixtureRunner{
		byCmd:    map[string][]byte{"go build": crash},
		byCmdErr: map[string]error{"go build": &fakeExitError{code: 1}},
	}

	_, err := dispatchPackEngines([]*pack.Manifest{m}, goToolchainPacksDir(t), t.TempDir(), nil, runner)
	if err == nil {
		t.Fatal("expected a fail-loud crash error (non-zero exit, no parseable findings), got nil — that is a silent green")
	}
	if !strings.Contains(err.Error(), "go-toolchain") && !strings.Contains(err.Error(), "go-build") && !strings.Contains(err.Error(), "go build") {
		t.Errorf("crash error must attribute the failing engine, got: %v", err)
	}
}

// TestGoToolchain_BuildTestNonZeroWithFindingsIsNormal proves the crash guard
// does NOT fire when the non-zero run DID produce parseable findings (the normal
// compile-error case): those surface as violations, not a crash (CLM-010 boundary).
func TestGoToolchain_BuildTestNonZeroWithFindingsIsNormal(t *testing.T) {
	m := onlyRules(goToolchainManifest(t), "go-build")
	stubSandboxedRunStdout(t, nil)
	runner := &fixtureRunner{
		byCmd:    map[string][]byte{"go build": readFixture(t, "go-build-errors.txt")},
		byCmdErr: map[string]error{"go build": &fakeExitError{code: 1}},
	}
	violations, err := dispatchPackEngines([]*pack.Manifest{m}, goToolchainPacksDir(t), t.TempDir(), nil, runner)
	if err != nil {
		t.Fatalf("non-zero exit WITH parseable findings must not be a crash, got err: %v", err)
	}
	if len(violations) != 3 {
		t.Fatalf("expected the 3 compiler errors as violations, got %d", len(violations))
	}
}

// TestGoToolchain_BuildTestArgShapingScopeKindAware proves the N1 arg-shaping:
// build/test do NOT get projectRoot appended as a scan target the way
// semgrep/ast-grep do — `go build ./...` not `go build <root>` (REQ-010).
func TestGoToolchain_BuildTestArgShapingScopeKindAware(t *testing.T) {
	m := onlyRules(goToolchainManifest(t), "go-build")
	stubSandboxedRunStdout(t, nil)
	root := t.TempDir()
	runner := &fixtureRunner{byCmd: map[string][]byte{"go build": readFixture(t, "go-build-errors.txt")}}

	if _, err := dispatchPackEngines([]*pack.Manifest{m}, goToolchainPacksDir(t), root, nil, runner); err != nil {
		t.Fatalf("dispatchPackEngines: %v", err)
	}
	if len(runner.calls) != 1 {
		t.Fatalf("expected one build invocation, got %d", len(runner.calls))
	}
	call := runner.calls[0]
	// The invoked NAME is the binding's declared producer once it declares one
	// (ISSUE-067) — core swaps argv[0] and nothing else. What this test PROVES is
	// the arg shaping below, which the swap deliberately leaves untouched; both
	// forms are accepted so the grounding tracks the declaration rather than
	// pinning a tool name the pack is free to front with a producer.
	if call.name != "go" && producerCommandAlias()[filepath.Base(call.name)] != "go build" {
		t.Fatalf("expected the `go` tool or its declared producer, got %q", call.name)
	}
	for _, a := range call.args {
		if a == root {
			t.Errorf("project root must NOT be appended as a scan target for a project-wide toolchain pass; args=%v", call.args)
		}
	}
	joined := strings.Join(call.args, " ")
	if !strings.Contains(joined, "./...") {
		t.Errorf("build pass must target ./..., got args=%v", call.args)
	}
}

// TestFixtureRunner_ProducerAliasCoversEveryDeclaredProducer pins the harness
// alias against the REAL fixture manifest (ISSUE-067, sharp edge 2). A
// hand-written producer→command alias in a test double can silently diverge from
// the actual declaration, and the failure mode of that divergence is the corpus
// going QUIET — which looks like passing. So: the set of producer basenames the
// manifest declares across its FINDINGS-typed engine bindings must be EXACTLY the
// alias map's key set.
//
// THE ENUMERATION IS SCOPED TO FINDINGS BINDINGS, ON PURPOSE. The pack also
// declares a coverage producer (go-coverage / coverage-produce.sh), which rides a
// STRUCTURALLY DIFFERENT dispatch: runCoverageEngine invokes it BARE — no args —
// and its output goes to the coverage-RECORDS channel, never through the
// `name + " " + args[0]` command-key resolution this alias exists to double.
// Requiring it in a findings alias map would pin a relationship that does not
// exist. The filter uses the SAME predicate production routes on
// (dispatchPackCoverage's `GateType != engine.GateTypeCoverage`) so the pin tracks
// the real routing rule rather than a second, drifting definition of it.
//
// The expected set is DERIVED from the parsed manifest, never hardcoded — a
// hardcoded list would go inert exactly when it is needed.
func TestFixtureRunner_ProducerAliasCoversEveryDeclaredProducer(t *testing.T) {
	manifest := goToolchainManifest(t)

	declared := map[string]string{}
	for name, spec := range manifest.Engines {
		if spec.Binding.GateType == engine.GateTypeCoverage {
			continue
		}
		if spec.Binding.Producer == "" {
			continue
		}
		declared[filepath.Base(filepath.FromSlash(spec.Binding.Producer))] = name
	}

	if len(declared) == 0 {
		t.Fatal("no findings-typed binding declares a producer — the pin has nothing to protect, which means the pack data regressed")
	}

	for base, engineName := range declared {
		if _, ok := producerCommandAlias()[base]; !ok {
			t.Errorf("findings engine %q declares producer %q, which the fixture harness alias does not cover — every test keyed on that engine's command would silently observe ZERO violations", engineName, base)
		}
	}
	for base := range producerCommandAlias() {
		if _, ok := declared[base]; !ok {
			t.Errorf("the harness aliases producer %q, which no findings-typed binding declares — the alias is stale", base)
		}
	}
}
