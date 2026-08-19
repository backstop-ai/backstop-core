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

// writeReuseState writes a real (if tiny) profile and the freshness stamp into
// project, in the PRODUCTION ORDER — the profile FIRST, the stamp SECOND. That order
// is not incidental: it is the order test-produce.sh produces, and it is the whole
// subject of these legs.
func writeReuseState(t *testing.T, project string) (profile, stamp string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(project, ".backstop"), 0o755); err != nil {
		t.Fatalf("creating .backstop: %v", err)
	}
	profile = filepath.Join(project, "cover.out")
	if err := os.WriteFile(profile, []byte("mode: set\nexample.com/harness/pkg/widget/widget.go:1.1,2.2 1 1\n"), 0o644); err != nil {
		t.Fatalf("writing cover.out: %v", err)
	}
	stamp = filepath.Join(project, filepath.FromSlash(goCoverageFreshStamp))
	if err := os.WriteFile(stamp, nil, 0o644); err != nil {
		t.Fatalf("writing the stamp: %v", err)
	}
	return profile, stamp
}

// setMtime pins one file's mtime to an explicit offset from now, so a leg can state
// the timestamp relation it is testing OUTRIGHT instead of hoping the filesystem's
// granularity happens to resolve a natural write gap.
func setMtime(t *testing.T, path string, offset time.Duration) {
	t.Helper()
	at := time.Now().Add(offset)
	if err := os.Chtimes(path, at, at); err != nil {
		t.Fatalf("setting the mtime of %s: %v", path, err)
	}
}

// assertReusedAndEnriched asserts the producer took the REUSE path: no second suite
// run, and the enrichment the sandboxed convert depends on still performed.
func assertReusedAndEnriched(t *testing.T, profile, shimLog string) {
	t.Helper()
	if shimInvokedTest(t, shimLog) {
		raw, _ := os.ReadFile(shimLog)
		t.Fatalf("with a FRESH stamped profile the coverage producer must REUSE it and run NO second suite — "+
			"that redundant run is the whole defect ISSUE-172 removes and ISSUE-179 restored; the shim recorded: %s", raw)
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
}

// assertStampConsumed asserts the stamp was deleted on the way through, which is what
// makes reuse structurally impossible without a test run in THIS invocation.
func assertStampConsumed(t *testing.T, stamp string) {
	t.Helper()
	if _, err := os.Stat(stamp); !os.IsNotExist(err) {
		t.Fatalf("the stamp must be consumed (deleted) so reuse is impossible without a test run in THIS invocation; stat err=%v", err)
	}
}

// TestGoToolchainSingleRun_CoverageProducerReusesAFreshProfile is the executable
// falsifier for the producer half: given a stamp written by a test run in THIS gate
// invocation and a matching cover.out, the coverage producer must NOT run the suite
// again — and must still perform the enrichment the sandboxed convert depends on.
//
// ★ WHAT THE ORIGINAL BODY GOT WRONG (ISSUE-179). It arranged the state by AGEING THE
// STAMP BACKWARDS ONE SECOND (`os.Chtimes(stamp, now-1s)`) so that cover.out came out
// NEWER than the stamp, and called that "the freshness relation the producer tests".
// It is the INVERSE of the relation production actually produces. test-produce.sh
// writes cover.out via `go "$@"` and touches the stamp AFTER, in the same script, so on
// a genuine same-invocation success the STAMP is the newer of the two — always, without
// exception. By fabricating the opposite ordering at a whole-second magnitude, the old
// body satisfied the shipped (backwards) comparison on EVERY platform, and so stayed
// green on Linux while the mechanism it claimed to guard was a complete no-op there.
// The relation the producer must actually test is "the STAMP is not older than the
// PROFILE", because the profile is written first.
//
// The three legs below therefore arrange the REAL direction, at three different
// magnitudes, and one deliberately-inverted state that must be REFUSED.
func TestGoToolchainSingleRun_CoverageProducerReusesAFreshProfile(t *testing.T) {
	// LEG 1 — PRIMARY, PLATFORM-INDEPENDENT. The true production direction (profile
	// older than stamp) stated EXPLICITLY at a two-whole-second magnitude, so even a
	// `test -ot` that truncates to whole seconds resolves it correctly. This is the leg
	// that makes the defect catchable on a darwin workstation, where the natural
	// sub-second gap ties and hides it.
	t.Run("production_chronology_coarse_visible", func(t *testing.T) {
		project, shimLog := singleRunHarness(t, 0)
		profile, stamp := writeReuseState(t, project)
		setMtime(t, profile, -2*time.Second)
		setMtime(t, stamp, 0)

		if status := runProducer(t, "coverage-produce.sh", project, shimLog); status != 0 {
			t.Fatalf("the coverage producer must exit 0; got %d", status)
		}
		assertReusedAndEnriched(t, profile, shimLog)
		assertStampConsumed(t, stamp)
	})

	// LEG 2 — PRECISION-SENSITIVE, AND ITS DARWIN PASS IS CORRECT, NOT A BUG. Natural
	// write-then-touch with NO os.Chtimes at all: the real few-hundred-microsecond gap
	// production produces. It runs under /bin/sh, which on darwin truncates both
	// timestamps to the same whole second — so this leg PASSES on darwin today even
	// with the comparison backwards, and is red only where /bin/sh reads nanoseconds
	// (CI's ubuntu-24.04, whose /bin/sh is dash). Do not "fix" its darwin pass and do
	// not delete it as redundant with leg 1: leg 1 deliberately exaggerates the
	// magnitude, and this leg is the one that pins the REAL production magnitude.
	t.Run("production_chronology_subsecond", func(t *testing.T) {
		project, shimLog := singleRunHarness(t, 0)
		profile, stamp := writeReuseState(t, project)

		if status := runProducer(t, "coverage-produce.sh", project, shimLog); status != 0 {
			t.Fatalf("the coverage producer must exit 0; got %d", status)
		}
		assertReusedAndEnriched(t, profile, shimLog)
		assertStampConsumed(t, stamp)
	})

	// LEG 3 — THE PROPERTY THE FLIPPED COMPARISON BUYS. A stamp left behind by an
	// invocation that aborted between the test dispatch and the coverage dispatch, over
	// a cover.out a LATER file-scoped run has since overwritten with a PARTIAL profile.
	// That state is reachable (the stamp is gitignored, so nothing surfaces it) and
	// reusing it would hand the coverage dimension an incomplete measurement with
	// nothing red — the exact silent-narrowing class test-produce.sh's `./...` guard
	// exists to prevent. The producer must REFUSE it and run the suite itself.
	//
	// This is the OLD body's fabricated scenario with its expectation CORRECTED: the
	// same arranged state, the opposite expected outcome.
	t.Run("stale_stamp_older_than_profile_is_refused", func(t *testing.T) {
		project, shimLog := singleRunHarness(t, 0)
		profile, stamp := writeReuseState(t, project)
		setMtime(t, stamp, -2*time.Second)
		setMtime(t, profile, 0)

		if status := runProducer(t, "coverage-produce.sh", project, shimLog); status != 0 {
			t.Fatalf("the coverage producer must exit 0; got %d", status)
		}
		if !shimInvokedTest(t, shimLog) {
			raw, _ := os.ReadFile(shimLog)
			t.Fatalf("a stamp OLDER than cover.out means a later run overwrote the profile without stamping it — "+
				"a file-scoped run's profile is PARTIAL — so the producer must REFUSE the reuse and run the suite "+
				"itself; reusing it reports an incomplete measurement as a complete one with nothing red "+
				"(ISSUE-179 CLM-006); the shim recorded: %s", raw)
		}
		assertStampConsumed(t, stamp)
	})
}

// probeShell is one shell the reuse condition is evaluated under. args carries any
// words that must precede `-c` (busybox needs `sh`), so a multi-word invocation is a
// data difference rather than a second code path.
type probeShell struct {
	name string
	path string
	args []string
}

// resolveProbeShells returns every shell on this machine the condition can be
// evaluated under.
//
// ★ zsh AND ksh ARE LOAD-BEARING — DO NOT TRIM THIS LIST BACK TO POSIX SHELLS. The
// obvious guess, "dash is the precise one", is FALSE, and it was measured rather than
// assumed (ISSUE-179): macOS's own /bin/dash truncates mtimes to whole seconds exactly
// like /bin/sh and /bin/bash, so a {sh, dash, bash} matrix is GREEN ON DARWIN WITH THE
// DEFECT FULLY LIVE — decoration, incapable of failing where it is being written.
// Whether a shell's `test` builtin reads the seconds field or the nanoseconds field of
// stat is a property of THAT SHELL AS BUILT FOR THAT PLATFORM, not of its name. On
// darwin, zsh and ksh are the shells that read nanoseconds, and they are what make this
// test a real falsifier on the authoring machine. On CI's ubuntu-24.04 /bin/sh IS dash
// and reads nanoseconds, so the same test runs for real on the platform that matters.
func resolveProbeShells(t *testing.T) []probeShell {
	t.Helper()

	shells := []probeShell{}
	if _, err := os.Stat("/bin/sh"); err != nil {
		t.Fatalf("/bin/sh must exist for this guard to mean anything: %v", err)
	}
	shells = append(shells, probeShell{name: "/bin/sh", path: "/bin/sh"})

	for _, name := range []string{"dash", "bash", "zsh", "ksh"} {
		path, err := exec.LookPath(name)
		if err != nil {
			// Not on PATH; fall back to the conventional location before giving up,
			// since /bin is not always on a test process's PATH.
			candidate := filepath.Join("/bin", name)
			if _, statErr := os.Stat(candidate); statErr != nil {
				continue
			}
			path = candidate
		}
		shells = append(shells, probeShell{name: name, path: path})
	}
	if path, err := exec.LookPath("busybox"); err == nil {
		shells = append(shells, probeShell{name: "busybox sh", path: path, args: []string{"sh"}})
	}
	return shells
}

// extractReuseCondition reads the REUSE CONDITION OUT OF THE FIXTURE PRODUCER SCRIPT
// at run time and returns a self-contained shell fragment that prints its verdict.
//
// ★★ THE CONDITION IS NEVER HARDCODED IN GO, AND BOTH WAYS OF HARDCODING IT ARE WRONG
// (ISSUE-179 CLM-010). Embedding the CORRECT condition as a Go literal makes this test
// report REUSE under every shell from the moment it is written — permanently green,
// structurally incapable of telling a fixed producer from a defective one, which is
// this very defect's class reproduced inside the test written to prevent it. Embedding
// the BACKWARDS condition makes it permanently red, because no task that fixes the
// producer edits a Go string literal. Reading the live script is the only shape that
// tracks the thing under test.
//
// The extracted block is self-contained — it defines $stamp itself and yields $reuse —
// and extraction deliberately STOPS AT `fi`, excluding the `rm -f "$stamp"` line below
// it, so evaluating the fragment does not delete the state under test.
func extractReuseCondition(t *testing.T) string {
	t.Helper()

	scriptPath := filepath.Join(goToolchainPackRoot(t), "scripts", "coverage-produce.sh")
	raw, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("reading the fixture coverage producer %s: %v", scriptPath, err)
	}

	// ★ EVERY FAILURE BELOW IS A HARD FAIL, NEVER A SKIP AND NEVER A FALLBACK. A silent
	// degradation here — an empty fragment, a skip, a hardcoded default — would turn a
	// future reshaping of the producer into a vacuously green guard with no signal,
	// which is exactly the failure class ISSUE-179 exists to correct.
	lines := strings.Split(string(raw), "\n")
	start := -1
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "stamp=") {
			start = i
			break
		}
	}
	if start < 0 {
		t.Fatalf("no line beginning `stamp=` in %s — the reuse condition could not be extracted. This guard evaluates "+
			"the condition READ OUT OF the producer at run time precisely so it cannot go vacuously green when the "+
			"producer is reshaped; re-anchor the extraction rather than hardcoding the condition in Go", scriptPath)
	}
	end := -1
	for i := start + 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "fi" {
			end = i
			break
		}
	}
	if end < 0 {
		t.Fatalf("no closing `fi` after the `stamp=` line in %s — the reuse condition could not be extracted; "+
			"re-anchor the extraction rather than hardcoding the condition in Go", scriptPath)
	}

	fragment := strings.Join(lines[start:end+1], "\n")
	if !strings.Contains(fragment, "-ot") {
		t.Fatalf("the block extracted from %s carries no `-ot` freshness comparison, so this guard would be "+
			"evaluating nothing:\n%s", scriptPath, fragment)
	}
	return fragment + "\nprintf '%s\\n' \"$reuse\"\n"
}

// TestGoToolchainSingleRun_CoverageProducerReuseIsShellPrecisionIndependent pins the
// dimension the rest of this file had none of: the reuse decision must not depend on
// how precisely the evaluating shell compares mtimes.
//
// It evaluates the CONDITION — not the whole producer script — under every shell it can
// resolve, against the natural sub-second production chronology. Because it runs only
// the extracted condition, a non-POSIX shell is a perfectly legitimate probe here; that
// is what lets zsh and ksh participate, and they are the only darwin shells that
// resolve the comparison at nanosecond precision.
//
// A backwards comparison makes the precise shells disagree with the coarse ones. A
// correct one makes every shell agree, which is the property being pinned.
func TestGoToolchainSingleRun_CoverageProducerReuseIsShellPrecisionIndependent(t *testing.T) {
	fragment := extractReuseCondition(t)

	shells := resolveProbeShells(t)
	if len(shells) < 1 {
		t.Fatal("resolved NO shell to evaluate the reuse condition under — this guard would be vacuous, so it fails " +
			"rather than skipping")
	}

	type verdict struct {
		shell  string
		reuse  string
		detail string
	}
	verdicts := make([]verdict, 0, len(shells))

	for _, sh := range shells {
		// A fresh directory per shell, arranged in the PRODUCTION ORDER with no
		// os.Chtimes: the profile first, the stamp immediately after. That is the real
		// few-hundred-microsecond gap, which is precisely the magnitude a coarse `-ot`
		// ties on and a precise one resolves.
		dir := t.TempDir()
		writeReuseState(t, dir)

		cmd := exec.Command(sh.path, append(append([]string{}, sh.args...), "-c", fragment)...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("evaluating the reuse condition under %s (%s) failed: %v (output %s)", sh.name, sh.path, err, out)
		}
		verdicts = append(verdicts, verdict{shell: sh.name, reuse: strings.TrimSpace(string(out)), detail: sh.path})
	}

	// LOG THE FULL MATRIX unconditionally, so a green can never be vacuous and a future
	// reader can see immediately which shells were actually probed.
	for _, v := range verdicts {
		label := "NO-REUSE"
		if v.reuse == "1" {
			label = "REUSE"
		}
		t.Logf("reuse condition under %-10s (%s): %s", v.shell, v.detail, label)
	}

	for _, v := range verdicts {
		if v.reuse != "1" {
			t.Errorf("shell %s (%s) evaluated the reuse condition to NO-REUSE against the REAL production chronology "+
				"(cover.out written first, stamp touched immediately after). The freshness comparison is "+
				"directionally backwards: test-produce.sh writes the profile and THEN touches the stamp, so demanding "+
				"the profile be no-older-than the stamp asks for a state a successful run never produces. A shell that "+
				"compares mtimes at whole-second resolution ties and hides this; one that reads nanoseconds does not "+
				"— which is why CI's ubuntu-24.04 /bin/sh (dash) never reused and CI run 32275399064 measured no "+
				"improvement at all (ISSUE-179)", v.shell, v.detail)
		}
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
