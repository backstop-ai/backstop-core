package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// gotoolchain_single_run_test.go pins the SINGLE-RUN CONVENTION in the go-toolchain
// pack (ISSUE-172, discharging the go-toolchain follow-on ISSUE-068 parked).
//
// THE DEFECT IT GUARDS. `go-test` and `go-coverage` each dispatched an independent
// whole-module `go test ./...`, so the two dominant gate steps were ONE suite run
// TWICE — measured on CI run 32151610956 as pack_engines 629797ms against
// coverage_threshold 612148ms, a ~17.6s delta that is just the build and lint passes
// riding in pack_engines. The convention removes the duplication: `go-test`'s
// command carries `-coverprofile=cover.out`, so ONE run emits both the test output
// and the profile, and the coverage producer REUSES that profile instead of
// re-running the suite.
//
// EVERYTHING HERE ASSERTS AGAINST THE CORE FIXTURE, NOT THE INSTALLED PACK. The
// fixture at cmd/backstop/testdata/go-toolchain/.backstop/packs/backstop/go-toolchain
// is a deliberately DIVERGED older snapshot (name backstop/go-toolchain, version
// 1.1.0, no consumer-exclusions block) and it is the root for the whole go-toolchain
// test corpus — see goToolchainPackRoot/goToolchainManifest in
// pack_gate_gotoolchain_test.go. The INSTALLED backstop-ai/go-toolchain under
// .backstop/packs/ is a different tree at a different name and version, pinned
// separately by pkg/pack/engine/gotoolchain_installed_pack_singlerun_test.go. The two
// are never synced; the same SEMANTIC change is applied to each independently.

// goCoverageFreshStamp is the freshness stamp the test producer writes and the
// coverage producer consumes. Declared once here so the two harnesses below and the
// pack scripts cannot drift apart silently.
const goCoverageFreshStamp = ".backstop/go-coverage-fresh"

// TestGoToolchainSingleRun_TestEngineProducesCoverageProfile pins the manifest half
// of the convention: the TEST engine is what produces the profile.
//
// It asserts the two engines are pinned to the SAME profile path in the same test,
// because that agreement is the whole mechanism — `go-coverage` declares cover.out
// as its stdout_artifact (the file the dispatch feeds to the convert), so a `go-test`
// command writing anywhere else would produce a profile nothing reads while the
// coverage producer silently kept re-running the suite.
func TestGoToolchainSingleRun_TestEngineProducesCoverageProfile(t *testing.T) {
	m := goToolchainManifest(t)

	testEngine, ok := m.Engines["go-test"]
	if !ok {
		t.Fatalf("the go-toolchain fixture must declare a go-test engine; got engines %v", engineNames(m.Engines))
	}
	if !strings.Contains(testEngine.Command, "-coverprofile=cover.out") {
		t.Fatalf("go-test's command must carry -coverprofile=cover.out so ONE whole-module run emits both the test "+
			"output and the coverage profile (ISSUE-172; ISSUE-068's parked go-toolchain follow-on); got command %q",
			testEngine.Command)
	}

	coverageEngine, ok := m.Engines["go-coverage"]
	if !ok {
		t.Fatalf("the go-toolchain fixture must declare a go-coverage engine; got engines %v", engineNames(m.Engines))
	}
	if coverageEngine.StdoutArtifact != "cover.out" {
		t.Fatalf("go-coverage's stdout_artifact must stay cover.out — it is the file go-test now writes and the "+
			"coverage producer reuses; the two engines drifting apart makes the reuse dark; got %q",
			coverageEngine.StdoutArtifact)
	}
}

// engineNames returns the declared engine names, for failure messages.
func engineNames[T any](engines map[string]T) []string {
	names := make([]string, 0, len(engines))
	for n := range engines {
		names = append(names, n)
	}
	return names
}

// singleRunHarness builds a hermetic temp project with a recording `go` SHIM first on
// PATH, and returns the project dir plus the shim's log path.
//
// THE SHIM IS THE FALSIFIER. The producers' only observable behaviour that matters
// here is WHETHER THEY INVOKE THE TOOL, so the harness records every `go` invocation's
// argv and lets the test assert on the presence or absence of a `test` subcommand. It
// also answers `go list` plausibly, so the coverage producer's enrichment tail runs
// for real rather than being skipped into a vacuous pass.
func singleRunHarness(t *testing.T, exitCode int) (project, shimLog string) {
	t.Helper()
	project = t.TempDir()
	shimDir := t.TempDir()
	shimLog = filepath.Join(shimDir, "invocations.log")

	// `go list -m` yields a module path; `go list -f ...` yields one package's go
	// files; everything else (notably `test`) is recorded and exits with exitCode.
	shim := "#!/bin/sh\n" +
		"printf '%s\\n' \"$*\" >> '" + shimLog + "'\n" +
		"case \"$1\" in\n" +
		"  list)\n" +
		"    case \"$2\" in\n" +
		"      -m) echo example.com/harness ;;\n" +
		"      *) echo example.com/harness/pkg/widget/widget.go ;;\n" +
		"    esac\n" +
		"    exit 0 ;;\n" +
		"esac\n" +
		"exit " + strconv.Itoa(exitCode) + "\n"
	shimPath := filepath.Join(shimDir, "go")
	if err := os.WriteFile(shimPath, []byte(shim), 0o755); err != nil {
		t.Fatalf("writing the go shim: %v", err)
	}
	return project, shimLog
}

// runProducer runs one of the fixture pack's producer scripts with the project as
// cwd and the shim first on PATH, returning its exit status.
func runProducer(t *testing.T, script, project, shimLog string, args ...string) int {
	t.Helper()
	cmd := exec.Command("/bin/sh", append([]string{filepath.Join(goToolchainPackRoot(t), "scripts", script)}, args...)...)
	cmd.Dir = project
	cmd.Env = append(os.Environ(), "PATH="+filepath.Dir(shimLog)+string(os.PathListSeparator)+os.Getenv("PATH"))
	out, err := cmd.CombinedOutput()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return exitErr.ExitCode()
		}
		t.Fatalf("running %s: %v (output %s)", script, err, out)
	}
	return 0
}

// shimInvokedTest reports whether the shim log records a `go test ...` invocation.
func shimInvokedTest(t *testing.T, shimLog string) bool {
	t.Helper()
	raw, err := os.ReadFile(shimLog)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
		t.Fatalf("reading the shim log: %v", err)
	}
	for _, line := range strings.Split(string(raw), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "test") {
			return true
		}
	}
	return false
}

// TestGoToolchainSingleRun_CoverageProducerReusesAFreshProfile is the executable
// falsifier for the producer half: given a stamp written by a test run in THIS gate
// invocation and a matching cover.out, the coverage producer must NOT run the suite
// again — and must still perform the enrichment the sandboxed convert depends on.
func TestGoToolchainSingleRun_CoverageProducerReusesAFreshProfile(t *testing.T) {
	project, shimLog := singleRunHarness(t, 0)

	if err := os.MkdirAll(filepath.Join(project, ".backstop"), 0o755); err != nil {
		t.Fatalf("creating .backstop: %v", err)
	}
	// A real (if tiny) profile, then the stamp — written in that order and with a
	// deliberate mtime nudge so cover.out is NOT older than the stamp, which is the
	// freshness relation the producer tests.
	profile := filepath.Join(project, "cover.out")
	if err := os.WriteFile(profile, []byte("mode: set\nexample.com/harness/pkg/widget/widget.go:1.1,2.2 1 1\n"), 0o644); err != nil {
		t.Fatalf("writing cover.out: %v", err)
	}
	stamp := filepath.Join(project, filepath.FromSlash(goCoverageFreshStamp))
	if err := os.WriteFile(stamp, nil, 0o644); err != nil {
		t.Fatalf("writing the stamp: %v", err)
	}
	older := time.Now().Add(-1 * time.Second)
	if err := os.Chtimes(stamp, older, older); err != nil {
		t.Fatalf("aging the stamp: %v", err)
	}

	if status := runProducer(t, "coverage-produce.sh", project, shimLog); status != 0 {
		t.Fatalf("the coverage producer must exit 0; got %d", status)
	}

	if shimInvokedTest(t, shimLog) {
		raw, _ := os.ReadFile(shimLog)
		t.Fatalf("with a FRESH stamped profile the coverage producer must REUSE it and run NO second suite — "+
			"that redundant run is the whole defect ISSUE-172 removes; the shim recorded: %s", raw)
	}

	// The reuse path must not skip the enrichment: the convert is parse-only and
	// sandboxed, so everything it knows arrives as these #backstop-* lines.
	enriched, err := os.ReadFile(profile)
	if err != nil {
		t.Fatalf("reading the enriched profile: %v", err)
	}
	if !strings.Contains(string(enriched), "#backstop-module example.com/harness") {
		t.Fatalf("the reused profile must still carry the #backstop-module line the convert parses; got:\n%s", enriched)
	}
	if !strings.Contains(string(enriched), "#backstop-gofile ") {
		t.Fatalf("the reused profile must still carry the #backstop-gofile lines the convert parses; got:\n%s", enriched)
	}
	// The stamp is CONSUMED, so a later coverage-only invocation cannot honour it.
	if _, err := os.Stat(stamp); !os.IsNotExist(err) {
		t.Fatalf("the stamp must be consumed (deleted) so reuse is impossible without a test run in THIS invocation; stat err=%v", err)
	}
}

// TestGoToolchainSingleRun_CoverageProducerRerunsWithoutAStamp is the paired guard
// for the DEGRADED path, and it must stay green forever.
//
// It is green today by construction — the producer currently always runs the tool —
// and that is correct and expected. It exists so a later "optimization" cannot turn
// the fallback off: with no stamp there is no evidence a whole-module profile was
// produced in this invocation, and the producer must run the suite itself.
// Slow-but-correct, never wrong.
func TestGoToolchainSingleRun_CoverageProducerRerunsWithoutAStamp(t *testing.T) {
	project, shimLog := singleRunHarness(t, 0)

	// A cover.out is present but UNSTAMPED — a leftover from an earlier invocation.
	// This is the case that must NOT be reused: the profile could be from a
	// file-scoped (partial) run, or arbitrarily stale.
	if err := os.WriteFile(filepath.Join(project, "cover.out"), []byte("mode: set\n"), 0o644); err != nil {
		t.Fatalf("writing the stale cover.out: %v", err)
	}

	if status := runProducer(t, "coverage-produce.sh", project, shimLog); status != 0 {
		t.Fatalf("the coverage producer must exit 0 even on the degraded path; got %d", status)
	}

	if !shimInvokedTest(t, shimLog) {
		raw, _ := os.ReadFile(shimLog)
		t.Fatalf("with NO stamp the coverage producer must run the suite itself — reusing an unstamped profile would "+
			"honour a possibly partial or stale one (a file-scoped run's profile is PARTIAL); the shim recorded: %s", raw)
	}
}

// TestGoToolchainSingleRun_TestProducerPropagatesFailureExit is the most important
// guard in this change.
//
// test-produce.sh ENDED on `go "$@" 2>&1`, so that line's status IS the script's
// status, and the file's own comment forbids weakening it: crash_guard and the
// non-zero-exit contract both read it. Appending a stamp-writing step after that line
// silently REPLACES the script's exit status with the stamp step's — and every
// FAILING SUITE WOULD THEN READ GREEN. That is the worst possible outcome of
// ISSUE-172 and it is invisible.
//
// The leg pins PROPAGATION, not "non-zero": the exit-0 sub-case is what stops a
// script that hardcodes `exit 1` from satisfying it.
func TestGoToolchainSingleRun_TestProducerPropagatesFailureExit(t *testing.T) {
	for _, tc := range []struct {
		name string
		code int
	}{
		{"failing suite propagates non-zero", 1},
		{"passing suite propagates zero", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			project, shimLog := singleRunHarness(t, tc.code)

			// The arguments the REAL dispatch passes: core swaps only argv[0], so the
			// `test` subcommand arrives IN the argument list and the script must not
			// repeat it (writing `go test "$@"` would run `go test test ./...`).
			status := runProducer(t, "test-produce.sh", project, shimLog, "test", "./...")

			if status != tc.code {
				t.Fatalf("test-produce.sh must propagate the tool's exit status VERBATIM — crash_guard and the "+
					"non-zero-exit contract both read it, and swallowing it turns every failing suite GREEN "+
					"(ISSUE-172 sharp edge 1); tool exited %d, script exited %d", tc.code, status)
			}
			if !shimInvokedTest(t, shimLog) {
				t.Fatal("the harness must actually reach the tool, or this leg proves nothing about propagation")
			}
		})
	}
}
