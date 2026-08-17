package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/check"
	"github.com/backstop-ai/backstop-core/pkg/gate"
	"github.com/backstop-ai/backstop-core/pkg/pack"
	"github.com/backstop-ai/backstop-core/pkg/pack/engine"
)

// ISSUE-067 REQ-001 — the FINDINGS dispatch honors a pack-declared `producer:`.
//
// The seam already existed as pack DATA (EngineBinding.Producer) and the COVERAGE
// path already honored it (runCoverageEngine); runFindingsEngine ignored it
// entirely, so a pack declaring one silently got the plain command. These tests
// pin the fix AND its ONE deliberate asymmetry with the coverage producer: the
// coverage producer is invoked BARE (it shapes its own whole invocation), while
// the findings producer REPLACES THE TOOL AND NOTHING ELSE — core computes
// cmdArgs exactly as it does for the plain command and hands the SAME args to the
// producer. That is what keeps the findings path's four scope contracts alive
// (diff-scoped file list + ISSUE-010 CLM-003 anti-fallback, excludeTestdataPaths,
// fileModeTestTarget, ProjectTarget vs self-targeting).
//
// Every manifest here is SYNTHETIC (`acme/*`). No Go/tool literal appears in any
// subject: the core change is language- and tool-blind, and a `go test` command in
// these tests would make it read as go-specific.

// producerFindingsPack builds a scratch packs dir holding an `acme/producer` pack
// root with a real producer script on disk (so the dispatch's os.Stat resolution
// passes; the fake runner intercepts the exec), and returns the packs dir plus a
// manifest whose ONE findings engine declares that producer. The plain Command is
// a MUST-NOT-RUN sentinel: when a producer is declared the dispatch must invoke
// the packRoot-resolved script instead of splitCommand(binding.Command)'s tool.
func producerFindingsPack(t *testing.T, producerRel string, mutate func(*engine.EngineBinding)) (string, *pack.Manifest) {
	t.Helper()
	packsDir := t.TempDir()
	packRoot := filepath.Join(packsDir, "acme", "producer")
	mkDirAll(t, filepath.Join(packRoot, "scripts"))
	writeFileStr(t, filepath.Join(packRoot, "scripts", "produce.sh"),
		"#!/bin/sh\n# producer stub (the fake runner intercepts the exec)\n")
	// A REAL, EXECUTABLE, SUCCEEDING stand-in for the plain command, used by the
	// tests that drive the genuine ExecCommandRunner. It matters that this one
	// WORKS: if the plain command were itself unstartable, the never-started test
	// below would pass pre-fix on the COMMAND's failure and prove nothing about the
	// producer branch.
	writeExecutableScript(t, filepath.Join(packRoot, "scripts", "plain-ok.sh"),
		"#!/bin/sh\ncat <<'J'\n"+emptySarif+"\nJ\n")

	binding := engine.EngineBinding{
		Command:   "must-not-run --plain",
		Producer:  producerRel,
		InputMode: engine.InputModeNone,
		ScopeKind: engine.ScopeKindProjectWide,
		Category:  engine.EngineCategoryOpinion,
		GateType:  engine.GateTypeFindings,
	}
	if mutate != nil {
		mutate(&binding)
	}

	return packsDir, &pack.Manifest{
		NormalizedName: "acme/producer",
		Engines: map[string]pack.EngineSpec{
			"acme-producer": {Binding: binding},
		},
		Content: pack.Content{Ruleset: pack.Ruleset{Rules: []pack.Rule{
			{ID: "p", Engine: "acme-producer", Standard: "x"},
		}}},
	}
}

// writeExecutableScript writes a script with the exec bit set and verifies the
// mode, so a test that depends on the script RUNNING cannot silently degrade into
// a test of an unstartable process.
func writeExecutableScript(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("write executable script %s: %v", path, err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	if info.Mode().Perm()&0o111 == 0 {
		t.Fatalf("fixture invariant: %s must be executable, got mode %v", path, info.Mode().Perm())
	}
}

// emptySarif is a well-formed SARIF log carrying zero results — what a converter
// emits for output it recognizes but finds nothing in.
const emptySarif = `{"version":"2.1.0","runs":[{"results":[]}]}`

// oneResultSarif is a well-formed SARIF log carrying exactly one located result,
// so a test can prove the producer's payload reached the parser.
const oneResultSarif = `{"version":"2.1.0","runs":[{"results":[{"ruleId":"acme-rule","level":"error",` +
	`"message":{"text":"produced by the producer"},"locations":[{"physicalLocation":` +
	`{"artifactLocation":{"uri":"src/thing.txt"},"region":{"startLine":7}}}]}]}]}`

// assertNoPlainCommandRun fails if the binding's plain command tool was ever
// handed to the runner. The producer SUBSUMES the command; running both would
// double the work and defeat the whole point of the swap.
func assertNoPlainCommandRun(t *testing.T, runner *fixtureRunner) {
	t.Helper()
	for _, c := range runner.calls {
		if c.name == "must-not-run" {
			t.Errorf("the plain command must NEVER run when a producer is declared; ran %q with args %v", c.name, c.args)
		}
	}
}

// TestFindingsEngine_RunsDeclaredProducerInPlaceOfCommand (CLM-001): a findings
// binding declaring producer: gets the packRoot-resolved script run via the runner
// (un-sandboxed, exactly as the coverage producer is), and the binding's own
// command tool is never invoked. The producer's payload still flows through the
// unchanged convert/parse tail.
func TestFindingsEngine_RunsDeclaredProducerInPlaceOfCommand(t *testing.T) {
	if dispatchPackEnginesFn != nil {
		t.Fatal("dispatchPackEnginesFn must be nil — this must drive the REAL dispatch, not a stub")
	}
	packsDir, m := producerFindingsPack(t, "scripts/produce.sh", func(b *engine.EngineBinding) {
		b.ProjectTarget = "ALL-TARGETS"
	})
	producerPath := filepath.Join(packsDir, "acme", "producer", "scripts", "produce.sh")

	runner := &fixtureRunner{byCmd: map[string][]byte{producerPath: []byte(oneResultSarif)}}

	violations, err := dispatchPackEngines([]*pack.Manifest{m}, packsDir, t.TempDir(), nil, runner)
	if err != nil {
		t.Fatalf("producer findings dispatch: %v", err)
	}

	if len(runner.calls) != 1 {
		t.Fatalf("expected exactly one runner invocation (the producer), got %d: %#v", len(runner.calls), runner.calls)
	}
	if runner.calls[0].name != producerPath {
		t.Errorf("dispatch must invoke the packRoot-resolved producer script; got %q want %q", runner.calls[0].name, producerPath)
	}
	assertNoPlainCommandRun(t, runner)

	// The convert/parse tail is untouched: the producer's stdout is the payload.
	if len(violations) != 1 {
		t.Fatalf("the producer's payload must flow through the unchanged parse tail; got %d violations: %#v", len(violations), violations)
	}
	if violations[0].Message != "produced by the producer" || violations[0].File != "src/thing.txt" {
		t.Errorf("producer payload not parsed as findings: %#v", violations[0])
	}
	if !strings.HasPrefix(violations[0].Rule, "acme/producer/") {
		t.Errorf("violation must stay namespaced to the pack, got %q", violations[0].Rule)
	}
}

// TestFindingsEngine_ProducerReceivesTheSameShapedArgsAsTheCommand (CLM-001) is
// THE LOAD-BEARING assertion and the deliberate point of departure from the
// coverage producer, which is invoked bare. The findings path owns four scope
// contracts that would ALL silently evaporate the moment a pack declared a
// producer if the producer were invoked bare. Each subtest pins one of them
// against the args the producer actually received.
//
// EVERY EXPECTED VECTOR HERE BEGINS WITH `--plain`, AND THAT IS THE POINT.
// splitCommand is a plain strings.Fields tokenizer: ONLY argv[0] becomes cmdName,
// so the declared command's REMAINING tokens are already sitting in cmdArgs before
// any scope shaping happens. The swap replaces argv[0] alone, so those tokens ride
// through to the producer. That is not incidental — it is the exact mechanism the
// real go-toolchain producers depend on: `command: go test` tokenizes to
// cmdName="go" + cmdArgs=["test"], so the producer receives ["test", "./..."] and
// its body must be `go "$@"`, never `go test "$@"` (which would double the
// subcommand and exit non-zero on a green tree). Asserting the FULL vector here is
// what keeps that contract pinned in core rather than only in the pack.
func TestFindingsEngine_ProducerReceivesTheSameShapedArgsAsTheCommand(t *testing.T) {
	// (1) A project-wide binding with a ProjectTarget: the producer gets that target.
	t.Run("project-wide binding gets its project_target", func(t *testing.T) {
		packsDir, m := producerFindingsPack(t, "scripts/produce.sh", func(b *engine.EngineBinding) {
			b.ScopeKind = engine.ScopeKindProjectWide
			b.ProjectTarget = "ALL-TARGETS"
		})
		producerPath := filepath.Join(packsDir, "acme", "producer", "scripts", "produce.sh")
		runner := &fixtureRunner{byCmd: map[string][]byte{producerPath: []byte(emptySarif)}}

		if _, err := dispatchPackEngines([]*pack.Manifest{m}, packsDir, t.TempDir(), nil, runner); err != nil {
			t.Fatalf("dispatch: %v", err)
		}
		call := onlyProducerCall(t, runner, producerPath)
		if strings.Join(call.args, " ") != "--plain ALL-TARGETS" {
			t.Errorf("producer must receive the command's own tokens plus the project_target, exactly as the plain command would; got args %v", call.args)
		}
	})

	// (2) A diff-scoped binding: the producer gets the explicit in-scope file list,
	// NOT the project root.
	t.Run("diff-scoped binding gets the explicit in-scope file list", func(t *testing.T) {
		packsDir, m := producerFindingsPack(t, "scripts/produce.sh", func(b *engine.EngineBinding) {
			b.ScopeKind = engine.ScopeKindFileArgs
			b.ProjectTarget = ""
		})
		producerPath := filepath.Join(packsDir, "acme", "producer", "scripts", "produce.sh")
		runner := &fixtureRunner{byCmd: map[string][]byte{producerPath: []byte(emptySarif)}}
		scope := &gate.GateScope{Mode: gate.GateScopeModeDiff, Files: []string{"src/one.txt", "src/two.txt"}}

		if _, err := dispatchPackEngines([]*pack.Manifest{m}, packsDir, t.TempDir(), scope, runner); err != nil {
			t.Fatalf("dispatch: %v", err)
		}
		call := onlyProducerCall(t, runner, producerPath)
		if strings.Join(call.args, " ") != "--plain src/one.txt src/two.txt" {
			t.Errorf("producer must receive the diff-scoped file list unchanged; got args %v", call.args)
		}
	})

	// (3) THE ANTI-FALLBACK (ISSUE-010 CLM-003): a testdata-only diff yields an
	// EMPTY arg list. It must NOT silently become the project root — that would
	// turn a scan-nothing run into a whole-repo scan the moment a producer was
	// declared.
	t.Run("testdata-only diff yields empty args, never the project root", func(t *testing.T) {
		packsDir, m := producerFindingsPack(t, "scripts/produce.sh", func(b *engine.EngineBinding) {
			b.ScopeKind = engine.ScopeKindFileArgs
			b.ProjectTarget = ""
		})
		producerPath := filepath.Join(packsDir, "acme", "producer", "scripts", "produce.sh")
		runner := &fixtureRunner{byCmd: map[string][]byte{producerPath: []byte(emptySarif)}}
		projectRoot := t.TempDir()
		scope := &gate.GateScope{Mode: gate.GateScopeModeDiff, Files: []string{"pkg/x/testdata/planted.txt"}}

		if _, err := dispatchPackEngines([]*pack.Manifest{m}, packsDir, projectRoot, scope, runner); err != nil {
			t.Fatalf("dispatch: %v", err)
		}
		call := onlyProducerCall(t, runner, producerPath)
		// The command's own tokens remain; what must NOT appear is any TARGET.
		if strings.Join(call.args, " ") != "--plain" {
			t.Errorf("a testdata-only diff must append NO targets (scan nothing), leaving only the command's own tokens; got args %v", call.args)
		}
		for _, a := range call.args {
			if a == projectRoot {
				t.Fatalf("anti-fallback breach: the producer received the project root %q as a target", projectRoot)
			}
		}
	})

	// (4) File-mode PACKAGE scoping (SPEC-034 REQ-010/CLM-035): a package_scoped
	// project-wide binding under a file-mode scope gets fileModeTestTarget's
	// package selector, not its ProjectTarget.
	t.Run("package_scoped binding under file-mode gets the changed file's package", func(t *testing.T) {
		packsDir, m := producerFindingsPack(t, "scripts/produce.sh", func(b *engine.EngineBinding) {
			b.ScopeKind = engine.ScopeKindProjectWide
			b.ProjectTarget = "ALL-TARGETS"
			b.PackageScoped = true
		})
		producerPath := filepath.Join(packsDir, "acme", "producer", "scripts", "produce.sh")
		runner := &fixtureRunner{byCmd: map[string][]byte{producerPath: []byte(emptySarif)}}
		scope := &gate.GateScope{Mode: gate.GateScopeModeFile, Files: []string{"pkg/widget/widget_test.go"}}

		if _, err := dispatchPackEngines([]*pack.Manifest{m}, packsDir, t.TempDir(), scope, runner); err != nil {
			t.Fatalf("dispatch: %v", err)
		}
		call := onlyProducerCall(t, runner, producerPath)
		joined := strings.Join(call.args, " ")
		if joined != "--plain ./pkg/widget" {
			t.Errorf("file-mode package scoping must survive the producer swap; want the package selector ./pkg/widget, got args %v", call.args)
		}
		if strings.Contains(joined, "ALL-TARGETS") {
			t.Errorf("file-mode run must NOT fall back to the project-wide target; got args %v", call.args)
		}
	})
}

// onlyProducerCall returns the single recorded invocation of the producer path,
// failing if the producer did not run exactly once.
func onlyProducerCall(t *testing.T, runner *fixtureRunner, producerPath string) fixtureCall {
	t.Helper()
	var found []fixtureCall
	for _, c := range runner.calls {
		if c.name == producerPath {
			found = append(found, c)
		}
	}
	if len(found) != 1 {
		t.Fatalf("expected exactly one producer invocation at %q, got %d: %#v", producerPath, len(found), runner.calls)
	}
	assertNoPlainCommandRun(t, runner)
	return found[0]
}

// TestFindingsEngine_MissingDeclaredProducerFailsLoud (CLM-002): a declared-but-
// missing producer is a fail-loud broken-pack error naming pack + path — never a
// silent fall-back to the plain command, which would make a mis-typed declaration
// look like it worked. Both absence shapes are covered: no such file, and a path
// that resolves to a DIRECTORY.
func TestFindingsEngine_MissingDeclaredProducerFailsLoud(t *testing.T) {
	cases := []struct {
		name string
		rel  string
		prep func(t *testing.T, packRoot string)
	}{
		{name: "absent file", rel: "scripts/does-not-exist.sh"},
		{
			name: "path resolves to a directory",
			rel:  "scripts/a-directory",
			prep: func(t *testing.T, packRoot string) { mkDirAll(t, filepath.Join(packRoot, "scripts", "a-directory")) },
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			packsDir, m := producerFindingsPack(t, tc.rel, func(b *engine.EngineBinding) {
				b.ProjectTarget = "ALL-TARGETS"
			})
			if tc.prep != nil {
				tc.prep(t, filepath.Join(packsDir, "acme", "producer"))
			}
			runner := &fixtureRunner{byCmd: map[string][]byte{}}

			violations, err := dispatchPackEngines([]*pack.Manifest{m}, packsDir, t.TempDir(), nil, runner)
			if err == nil {
				t.Fatalf("a declared-but-missing producer must fail loud; got %d violations and no error", len(violations))
			}
			if !strings.Contains(err.Error(), "acme/producer") {
				t.Errorf("broken-pack error must name the pack, got: %v", err)
			}
			wantPath := filepath.Join(packsDir, "acme", "producer", filepath.FromSlash(tc.rel))
			if !strings.Contains(err.Error(), wantPath) {
				t.Errorf("broken-pack error must name the resolved producer path %q, got: %v", wantPath, err)
			}
			if len(runner.calls) != 0 {
				t.Errorf("THE RUNNER MUST NEVER BE REACHED when the producer is missing; got %#v", runner.calls)
			}
		})
	}
}

// TestFindingsEngine_ProducerNeverStartedIsRefused (CLM-003): the ISSUE-112
// never-started refusal still applies on the producer branch. A producer script
// that exists but cannot be exec'd (no exec bit) fails at fork/exec, produced
// nothing, and must be refused as a broken run — distinctly from the crash-guard
// message, which would mis-describe a process that never ran at all. Driven
// through the REAL ExecCommandRunner, because a stub runner cannot produce the
// fork/exec error shape.
func TestFindingsEngine_ProducerNeverStartedIsRefused(t *testing.T) {
	packsDir, m := producerFindingsPack(t, "scripts/produce.sh", nil)
	packRoot := filepath.Join(packsDir, "acme", "producer")
	producerPath := filepath.Join(packRoot, "scripts", "produce.sh")

	// THE PLAIN COMMAND MUST WORK. Point it at the real executable stand-in that
	// exits 0 with an empty findings document, so this test can ONLY go red on the
	// producer branch. With an unstartable sentinel command it would pass pre-fix
	// on the COMMAND's own failure and assert nothing about the producer.
	sp := m.Engines["acme-producer"]
	sp.Binding.Command = filepath.Join(packRoot, "scripts", "plain-ok.sh")
	sp.Binding.ProjectTarget = "ALL-TARGETS"
	m.Engines["acme-producer"] = sp

	// Strip the exec bit: the file EXISTS (so the os.Stat guard passes) but cannot
	// be exec'd — the shape a pack whose script lost its mode in transit produces.
	if err := os.Chmod(producerPath, 0o644); err != nil {
		t.Fatalf("chmod producer: %v", err)
	}
	info, err := os.Stat(producerPath)
	if err != nil {
		t.Fatalf("stat producer: %v", err)
	}
	if info.Mode().Perm()&0o111 != 0 {
		t.Fatalf("fixture invariant: %s must NOT be executable", producerPath)
	}

	projectRoot := t.TempDir()
	runner := &check.ExecCommandRunner{Dir: projectRoot}

	violations, dispErr := dispatchPackEngines([]*pack.Manifest{m}, packsDir, projectRoot, nil, runner)
	if dispErr == nil {
		t.Fatalf("a producer that never started must be refused; got %d violations and no error", len(violations))
	}
	var cfgErr *check.ConfigError
	if !errors.As(dispErr, &cfgErr) {
		t.Errorf("the never-started refusal must be a *check.ConfigError (exit 2), got %T: %v", dispErr, dispErr)
	}
	if !strings.Contains(dispErr.Error(), "never started") {
		t.Errorf("refusal must use the never-started wording, not the crash-guard wording; got: %v", dispErr)
	}
	if strings.Contains(dispErr.Error(), "crashed") {
		t.Errorf("a process that never ran must NOT be reported as a crash; got: %v", dispErr)
	}
	if len(violations) != 0 {
		t.Errorf("a refused run must yield no violations that could read as a clean scan; got %#v", violations)
	}
}

// TestFindingsEngine_ProducerCrashGuardStillFiresOnZeroFindings (CLM-003,
// CLM-012): the SPEC-034 crash guard is NOT disabled by the producer branch. A
// crash_guard binding whose producer RAN, exited non-zero, and converted to zero
// parseable findings is still a loud failure — that is the genuine-crash case
// this whole issue keeps intact while removing the false one.
func TestFindingsEngine_ProducerCrashGuardStillFiresOnZeroFindings(t *testing.T) {
	packsDir, m := producerFindingsPack(t, "scripts/produce.sh", func(b *engine.EngineBinding) {
		b.ProjectTarget = "ALL-TARGETS"
		b.CrashGuard = true
	})
	producerPath := filepath.Join(packsDir, "acme", "producer", "scripts", "produce.sh")

	// The producer STARTED and exited non-zero (a plain exit error, not a
	// never-started shape) while emitting a well-formed but EMPTY findings
	// document: no diagnostic at all.
	runner := &fixtureRunner{
		byCmd:    map[string][]byte{producerPath: []byte(emptySarif)},
		byCmdErr: map[string]error{producerPath: &fakeExitError{code: 2}},
	}

	violations, err := dispatchPackEngines([]*pack.Manifest{m}, packsDir, t.TempDir(), nil, runner)
	if err == nil {
		t.Fatalf("crash_guard must still fire on a non-zero producer run with zero findings; got %d violations and no error", len(violations))
	}
	if !strings.Contains(err.Error(), "crashed") {
		t.Errorf("the crash-guard error wording must survive the producer branch, got: %v", err)
	}
	if !strings.Contains(err.Error(), "acme/producer") {
		t.Errorf("crash-guard error must name the pack, got: %v", err)
	}
}

// TestFindingsEngine_NoProducerDeclaredPathUnchanged (CLM-004): an engine that
// declares NO producer takes a behaviourally unchanged path. The plain command's
// tool is invoked with exactly the args it received before, and the payload /
// convert / parse tail is untouched. This is the assertion that keeps the change
// a swap rather than a rewrite — the whole existing corpus depends on it.
func TestFindingsEngine_NoProducerDeclaredPathUnchanged(t *testing.T) {
	packsDir, m := producerFindingsPack(t, "", func(b *engine.EngineBinding) {
		b.Command = "plain-tool --flag"
		b.ProjectTarget = "ALL-TARGETS"
	})
	runner := &fixtureRunner{byCmd: map[string][]byte{"plain-tool": []byte(oneResultSarif)}}

	violations, err := dispatchPackEngines([]*pack.Manifest{m}, packsDir, t.TempDir(), nil, runner)
	if err != nil {
		t.Fatalf("no-producer dispatch: %v", err)
	}
	if len(runner.calls) != 1 {
		t.Fatalf("expected exactly one invocation (the plain command), got %d: %#v", len(runner.calls), runner.calls)
	}
	call := runner.calls[0]
	if call.name != "plain-tool" {
		t.Errorf("with no producer declared, splitCommand(binding.Command)'s tool must run; got %q", call.name)
	}
	if strings.Join(call.args, " ") != "--flag ALL-TARGETS" {
		t.Errorf("no-producer arg shaping must be unchanged; want `--flag ALL-TARGETS`, got %v", call.args)
	}
	if len(violations) != 1 || violations[0].Message != "produced by the producer" {
		t.Fatalf("the payload/convert/parse tail must be untouched; got %#v", violations)
	}
}
