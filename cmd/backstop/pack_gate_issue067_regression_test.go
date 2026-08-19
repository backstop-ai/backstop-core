package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/gate"
	"github.com/backstop-ai/backstop-core/pkg/pack"
	"github.com/backstop-ai/backstop-core/pkg/pack/engine"
)

// ISSUE-067 — the acceptance surface. `go test` splits its output across two
// streams and core captures only stdout by design (SPEC-031 CLM-028), so for every
// COMPILE or VET failure the located diagnostic was discarded before the convert
// ever ran. The converter was handed `FAIL <pkg> [build failed]`, emitted zero
// results, and crash_guard turned that into `engine "go test" crashed: non-zero
// exit with no parseable findings` — the opaque crash this issue is named for.
// go-build was worse: `go build` writes NOTHING to stdout, so build-to-sarif.sh
// received an empty payload on every production run and has never once fired.
//
// The fix is pack DATA, not core language knowledge: the pack declares producer
// scripts that fold the tool's own stderr into its own stdout, and the test
// converter learns to normalize the compiler/vet diagnostics that now arrive.
//
// These tests drive REAL artifacts throughout — the real fixture manifest, the
// real converter scripts (shelled by stubSandboxedRunStdout), the real captured
// fixtures, and the real dispatchPackEngines.

// issue067TestRunner returns a fixtureRunner feeding the given captured bytes to
// the go-test engine along with a non-zero exit, which is what the real tool does
// on any failure. The non-zero exit matters: it satisfies crash_guard's
// `runErr != nil` precondition, so these tests prove the guard is no longer
// REACHED rather than that it was disabled.
func issue067TestRunner(payload []byte) *fixtureRunner {
	return &fixtureRunner{
		byCmd:    map[string][]byte{"go test": payload},
		byCmdErr: map[string]error{"go test": &fakeExitError{code: 1}},
	}
}

// issue067BuildRunner is the go-build analogue of issue067TestRunner.
func issue067BuildRunner(payload []byte) *fixtureRunner {
	return &fixtureRunner{
		byCmd:    map[string][]byte{"go build": payload},
		byCmdErr: map[string]error{"go build": &fakeExitError{code: 1}},
	}
}

// buildFailedLines selects from a captured payload exactly the lines naming a
// `[build failed]` package. Used to DERIVE a test payload from real captured
// bytes rather than hand-spelling the shape — the byte layout is load-bearing
// (ONE TAB after FAIL, a SPACE before `[build failed]`) and a hand-typed literal
// is how a wrong shape gets reintroduced in a way the test then agrees with.
func buildFailedLines(payload []byte) []byte {
	var kept []string
	for _, line := range strings.Split(string(payload), "\n") {
		if strings.Contains(line, "[build failed]") {
			kept = append(kept, line)
		}
	}
	return []byte(strings.Join(kept, "\n") + "\n")
}

// buildFailedPackage extracts the import path from the first `[build failed]`
// line of a captured payload, so the expected package name is derived from the
// same real bytes rather than restated.
func buildFailedPackage(t *testing.T, payload []byte) string {
	t.Helper()
	for _, line := range strings.Split(string(payload), "\n") {
		if !strings.Contains(line, "[build failed]") {
			continue
		}
		fields := strings.Fields(line)
		// `FAIL\t<import-path> [build failed]` — the import path is field 1.
		if len(fields) >= 2 {
			return fields[1]
		}
	}
	t.Fatalf("capture carries no [build failed] line to derive a package from: %q", string(payload))
	return ""
}

// TestIssue067_StdoutOnlyCaptureYieldsAFindingNotTheOpaqueCrash is the plan's
// CENTRAL FALSIFIER. It feeds the stdout-only half of a real compile-failure run —
// literally what core sees TODAY — and requires a finding rather than the opaque
// crash (CLM-009).
//
// It deliberately does NOT assert that this payload still crashes after the fix.
// These bytes NAME A FAILED PACKAGE, and CLM-009's floor says a run whose output
// still names a failure always yields at least one finding. CLM-012's genuine-crash
// pin is narrower by its own wording ("no diagnostic output at all") and is held
// on a genuinely empty payload by TestFindingsEngine_ProducerCrashGuardStillFires
// OnZeroFindings; a second, conflicting pin here could not coexist with CLM-009.
func TestIssue067_StdoutOnlyCaptureYieldsAFindingNotTheOpaqueCrash(t *testing.T) {
	m := onlyRules(goToolchainManifest(t), "go-test")
	stubSandboxedRunStdout(t, nil)
	payload := readFixture(t, "go-test-build-failure-stdout-only.txt")
	runner := issue067TestRunner(payload)

	violations, err := dispatchPackEngines([]*pack.Manifest{m}, goToolchainPacksDir(t), t.TempDir(), nil, runner)
	if err != nil {
		t.Fatalf("the stdout-only capture must NOT reach the crash path — a run whose output names a failed package always yields a finding; got: %v", err)
	}
	if len(violations) == 0 {
		t.Fatal("expected at least one finding naming the [build failed] package, got none — this is ISSUE-067's zero-findings crash path")
	}
	wantPkg := buildFailedPackage(t, payload)
	found := false
	for _, v := range violations {
		if strings.Contains(v.Message, wantPkg) {
			found = true
		}
	}
	if !found {
		t.Errorf("no finding names the failed package %q; got %#v", wantPkg, violations)
	}
}

// TestIssue067_TestFileCompileFailureYieldsLocatedFindingsNotCrash (CLM-006): the
// MERGED capture — what the producer now hands the converter — yields findings
// located at the compiler's own file and line, with the compiler's text.
func TestIssue067_TestFileCompileFailureYieldsLocatedFindingsNotCrash(t *testing.T) {
	m := onlyRules(goToolchainManifest(t), "go-test")
	stubSandboxedRunStdout(t, nil)
	runner := issue067TestRunner(readFixture(t, "go-test-build-failure.txt"))

	violations, err := dispatchPackEngines([]*pack.Manifest{m}, goToolchainPacksDir(t), t.TempDir(), nil, runner)
	if err != nil {
		t.Fatalf("a test-file compile failure must yield located findings, not a dispatch error: %v", err)
	}
	if len(violations) == 0 {
		t.Fatal("expected located compile findings, got none")
	}

	var located *gate.Violation
	for i := range violations {
		if strings.HasSuffix(violations[i].File, "b_test.go") {
			located = &violations[i]
			break
		}
	}
	if located == nil {
		t.Fatalf("no finding carries the NESTED path the compiler printed (sub/pkg/b_test.go); got %#v", violations)
	}
	// The compiler printed a nested path; the finding must keep it, not collapse
	// to a bare basename.
	if !strings.Contains(located.File, "/") {
		t.Errorf("compiler diagnostics carry a nested path; got File %q", located.File)
	}
	if located.Line == 0 {
		t.Errorf("a compiler diagnostic is the most precisely located output the toolchain produces; got Line 0 in %#v", *located)
	}
	if !strings.Contains(located.Message, "undefined") {
		t.Errorf("finding must carry the compiler's own message text; got %q", located.Message)
	}
	if !strings.HasPrefix(located.Rule, "backstop/go-toolchain/") {
		t.Errorf("finding must be namespaced to the pack, got %q", located.Rule)
	}
}

// TestIssue067_VetFailureYieldsLocatedFinding (CLM-006): a `go vet` failure during
// `go test` splits the same way and its diagnostic also sits under a `#` header,
// which is the discriminator CLM-008 keys on.
func TestIssue067_VetFailureYieldsLocatedFinding(t *testing.T) {
	m := onlyRules(goToolchainManifest(t), "go-test")
	stubSandboxedRunStdout(t, nil)
	runner := issue067TestRunner(readFixture(t, "go-test-vet-failure.txt"))

	violations, err := dispatchPackEngines([]*pack.Manifest{m}, goToolchainPacksDir(t), t.TempDir(), nil, runner)
	if err != nil {
		t.Fatalf("a vet failure must yield a located finding, not a dispatch error: %v", err)
	}
	if len(violations) == 0 {
		t.Fatal("expected a located vet finding, got none")
	}
	var located *gate.Violation
	for i := range violations {
		if strings.HasSuffix(violations[i].File, "c_test.go") {
			located = &violations[i]
			break
		}
	}
	if located == nil {
		t.Fatalf("no finding carries the vet diagnostic's file; got %#v", violations)
	}
	if located.Line == 0 {
		t.Errorf("vet reports a precise line; got Line 0 in %#v", *located)
	}
	if !strings.Contains(located.Message, "wrong type") {
		t.Errorf("finding must carry vet's own message; got %q", located.Message)
	}
}

// TestIssue067_BuildFailureReachesTheBuildConverter (CLM-007): the converter that
// has never fired in production fires.
//
// This asserts BOTH halves, because either alone is vacuous. The converter half
// (located violations out of the real capture) has always been green in the test
// corpus — the corpus injects as STDOUT the bytes production only ever produces on
// STDERR, which is exactly why this defect survived component-level green. The
// REACHABILITY half — that the go-build engine actually invokes a producer, which
// is the only reason those bytes can arrive at all — is the part that was false in
// production and is what makes the claim about production rather than about the
// fixture.
func TestIssue067_BuildFailureReachesTheBuildConverter(t *testing.T) {
	manifest := goToolchainManifest(t)

	// (1) REACHABILITY: without a declared producer, `go build` writes nothing to
	// the stdout core captures, so build-to-sarif.sh receives an empty payload on
	// every real run no matter how well it parses.
	buildBinding, err := resolveEngineRegistry(manifest).Lookup("go-build")
	if err != nil {
		t.Fatalf("resolve go-build binding: %v", err)
	}
	if buildBinding.Producer == "" {
		t.Error("go-build must declare a producer — `go build` writes NOTHING to stdout, so without one the convert receives an empty payload on every production run")
	}

	// (2) NORMALIZATION: the real captured bytes yield located findings.
	m := onlyRules(manifest, "go-build")
	stubSandboxedRunStdout(t, nil)
	runner := issue067BuildRunner(readFixture(t, "go-build-stderr-capture.txt"))

	violations, dispErr := dispatchPackEngines([]*pack.Manifest{m}, goToolchainPacksDir(t), t.TempDir(), nil, runner)
	if dispErr != nil {
		t.Fatalf("a production-code compile failure must yield located findings, not a dispatch error: %v", dispErr)
	}
	if len(violations) == 0 {
		t.Fatal("build-to-sarif.sh produced no findings from a real `go build` capture")
	}
	for _, v := range violations {
		if v.File == "" || v.Line == 0 {
			t.Errorf("every compiler finding must be located; got %#v", v)
		}
	}
}

// TestIssue067_DiagnosticShapedTestOutputManufacturesNoFinding (CLM-008): merging
// stderr widens the converter's input, so anything a test writes to stderr now
// reaches it. The `# <import-path>` header is the discriminator — a passing test
// that prints a `file.go:12: something`-shaped line manufactures NO finding,
// because no header is in effect.
func TestIssue067_DiagnosticShapedTestOutputManufacturesNoFinding(t *testing.T) {
	m := onlyRules(goToolchainManifest(t), "go-test")
	stubSandboxedRunStdout(t, nil)
	runner := issue067TestRunner(readFixture(t, "go-test-hostile-stderr.txt"))

	violations, err := dispatchPackEngines([]*pack.Manifest{m}, goToolchainPacksDir(t), t.TempDir(), nil, runner)
	if err != nil {
		t.Fatalf("dispatch: %v", err)
	}

	// The EXACT violation set, not merely a count: the genuinely failing test
	// produces one finding, and the hostile stderr line produces none.
	if len(violations) != 1 {
		t.Fatalf("expected exactly one finding (the genuinely failing test); got %d: %#v", len(violations), violations)
	}
	if !strings.Contains(violations[0].Message, "TestGenuinelyFails") {
		t.Errorf("the one finding must be the genuinely failing test; got %q", violations[0].Message)
	}
	for _, v := range violations {
		if strings.Contains(v.File, "notes.go") || strings.Contains(v.Message, "check this") {
			t.Errorf("a passing test's diagnostic-shaped stderr line MANUFACTURED a finding — the header discriminator is not holding: %#v", v)
		}
	}
}

// TestIssue067_BuildFailedSummaryAlwaysYieldsAtLeastOneFinding (CLM-009) is the
// FLOOR that keeps a partially-captured failure off the crash path: a payload
// carrying ONLY the `[build failed]` summary — no `#` header, no positional line —
// still yields a finding naming the package.
//
// The payload is DERIVED from the real capture, never hand-spelled: the byte shape
// (one TAB after FAIL, a SPACE before `[build failed]`) is exactly what an awk
// pattern gets wrong, and a hand-typed literal would make this test agree with a
// broken converter and pass vacuously.
func TestIssue067_BuildFailedSummaryAlwaysYieldsAtLeastOneFinding(t *testing.T) {
	capture := readFixture(t, "go-test-build-failure-stdout-only.txt")
	payload := buildFailedLines(capture)
	if !strings.Contains(string(payload), "[build failed]") {
		t.Fatalf("derived payload lost the summary line: %q", string(payload))
	}
	if strings.Contains(string(payload), "#") {
		t.Fatalf("derived payload must carry NO header line; got %q", string(payload))
	}

	m := onlyRules(goToolchainManifest(t), "go-test")
	stubSandboxedRunStdout(t, nil)
	runner := issue067TestRunner(payload)

	violations, err := dispatchPackEngines([]*pack.Manifest{m}, goToolchainPacksDir(t), t.TempDir(), nil, runner)
	if err != nil {
		t.Fatalf("a run naming a failed package must never reach the crash path: %v", err)
	}
	if len(violations) < 1 {
		t.Fatal("the [build failed] summary alone must yield at least one finding — this is the floor that keeps a partial capture off the crash path")
	}
	wantPkg := buildFailedPackage(t, capture)
	found := false
	for _, v := range violations {
		if strings.Contains(v.Message, wantPkg) {
			found = true
		}
	}
	if !found {
		t.Errorf("the floor finding must NAME the failed package %q; got %#v", wantPkg, violations)
	}
}

// TestIssue067_ExistingCapturedShapesUnchanged (CLM-010): the shapes that already
// worked are unchanged. This is a pure no-regression guard — it passes both before
// and after the widening BY DESIGN, and a change in either count means the
// converter's existing machinery moved when it was required not to.
func TestIssue067_ExistingCapturedShapesUnchanged(t *testing.T) {
	t.Run("go-test failures still yield exactly 3", func(t *testing.T) {
		m := onlyRules(goToolchainManifest(t), "go-test")
		stubSandboxedRunStdout(t, nil)
		runner := &fixtureRunner{byCmd: map[string][]byte{"go test": readFixture(t, "go-test-failures.txt")}}

		violations, err := dispatchPackEngines([]*pack.Manifest{m}, goToolchainPacksDir(t), t.TempDir(), nil, runner)
		if err != nil {
			t.Fatalf("dispatch: %v", err)
		}
		if len(violations) != 3 {
			t.Fatalf("the existing test capture must still yield exactly 3 violations; got %d: %#v", len(violations), violations)
		}
		if violations[0].Message != "TestWidgetFrobnicate: expected 5, got 7" {
			t.Errorf("assertion-failure normalization changed; got %q", violations[0].Message)
		}
		// TestNoPos keeps its NAMED-but-unlocated finding — legible without a
		// position is the accepted outcome; making it located is ISSUE-135's lane.
		var noPos *gate.Violation
		for i := range violations {
			if strings.Contains(violations[i].Message, "TestNoPos") {
				noPos = &violations[i]
			}
		}
		if noPos == nil {
			t.Fatalf("TestNoPos's unlocated finding disappeared; got %#v", violations)
		}
		if noPos.File != "" || noPos.Line != 0 {
			t.Errorf("TestNoPos must stay NAMED-but-unlocated; got %#v", *noPos)
		}
	})

	t.Run("go-build errors still yield exactly 3", func(t *testing.T) {
		m := onlyRules(goToolchainManifest(t), "go-build")
		stubSandboxedRunStdout(t, nil)
		runner := &fixtureRunner{byCmd: map[string][]byte{"go build": readFixture(t, "go-build-errors.txt")}}

		violations, err := dispatchPackEngines([]*pack.Manifest{m}, goToolchainPacksDir(t), t.TempDir(), nil, runner)
		if err != nil {
			t.Fatalf("dispatch: %v", err)
		}
		if len(violations) != 3 {
			t.Fatalf("the existing build capture must still yield exactly 3 violations; got %d: %#v", len(violations), violations)
		}
		if violations[0].File != "pkg/widget/widget.go" || violations[0].Message != "undefined: Frobnicate" {
			t.Errorf("build normalization changed; got %#v", violations[0])
		}
	})
}

// TestIssue067_MultipleDiagnosticsUnderOneHeaderAllBecomeFindings (CLM-008,
// sharp edge 8a) is THE ONLY TEST HERE THAT CATCHES A NON-STICKY HEADER. Go emits
// MANY diagnostics under ONE `#` header; a discriminator that checks only whether
// the IMMEDIATELY-PRECEDING line was a header converts the first and silently
// drops the rest — an under-report, the same failure class this plan exists to
// close, just quieter. It asserts a COUNT, because a non-sticky implementation
// passes every other test in this file.
func TestIssue067_MultipleDiagnosticsUnderOneHeaderAllBecomeFindings(t *testing.T) {
	capture := readFixture(t, "go-test-build-failure.txt")

	// Pin the FIXTURE'S OWN premise: this test is meaningless unless the capture
	// really does carry several diagnostics under a single header.
	var headers, diagnostics int
	for _, line := range strings.Split(string(capture), "\n") {
		switch {
		case strings.HasPrefix(line, "#"):
			headers++
		case strings.Contains(line, "b_test.go:"):
			diagnostics++
		}
	}
	if headers != 1 || diagnostics < 2 {
		t.Fatalf("fixture invariant: the capture must carry MULTIPLE diagnostics under ONE header; got %d headers, %d diagnostics", headers, diagnostics)
	}

	m := onlyRules(goToolchainManifest(t), "go-test")
	stubSandboxedRunStdout(t, nil)
	runner := issue067TestRunner(capture)

	violations, err := dispatchPackEngines([]*pack.Manifest{m}, goToolchainPacksDir(t), t.TempDir(), nil, runner)
	if err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	if len(violations) != diagnostics {
		t.Fatalf("ALL %d diagnostics under the single header must become findings (a non-sticky header converts only the first); got %d: %#v",
			diagnostics, len(violations), violations)
	}
	// Every reported line must be represented, not just the count.
	for _, want := range []string{"undefinedOne", "undefinedTwo", "undefinedThree"} {
		found := false
		for _, v := range violations {
			if strings.Contains(v.Message, want) {
				found = true
			}
		}
		if !found {
			t.Errorf("diagnostic %q under the shared header produced no finding; got %#v", want, violations)
		}
	}
}

// TestIssue067_MultiEngineDispatchSurvivesOneEnginesCompileBreak (CLM-011) is the
// BLAST-RADIUS claim. runFindingsEngine returning an error propagates out of
// dispatchPackEngines, which returns `nil, err` on the FIRST engine error — so one
// un-compilable test file discarded EVERY other engine's findings in the same run.
func TestIssue067_MultiEngineDispatchSurvivesOneEnginesCompileBreak(t *testing.T) {
	m := onlyRules(goToolchainManifest(t), "go-build", "go-test")
	stubSandboxedRunStdout(t, nil)

	// TWO engines in ONE dispatch: go-test fed the compile-failure capture (the
	// engine that used to abort the loop), and go-build fed output that genuinely
	// yields findings (the engine whose results used to be discarded with it).
	runner := &fixtureRunner{
		byCmd: map[string][]byte{
			"go test":  readFixture(t, "go-test-build-failure.txt"),
			"go build": readFixture(t, "go-build-errors.txt"),
		},
		byCmdErr: map[string]error{"go test": &fakeExitError{code: 1}},
	}

	violations, err := dispatchPackEngines([]*pack.Manifest{m}, goToolchainPacksDir(t), t.TempDir(), nil, runner)
	if err != nil {
		t.Fatalf("one engine's compile break must NOT abort the whole pack loop: %v", err)
	}

	var buildCount, testCount int
	for _, v := range violations {
		switch v.GateType {
		case "build":
			buildCount++
		case "test":
			testCount++
		}
	}
	if buildCount == 0 {
		t.Errorf("the WORKING engine's findings were discarded by the other engine's failure — that is the blast radius; got %#v", violations)
	}
	if testCount == 0 {
		t.Errorf("the compile-broken engine must itself contribute located findings; got %#v", violations)
	}
}

// TestIssue067_VerdictCollectorIsPopulatedNotStarved (CLM-011) is the second half
// of the blast radius, asserted directly rather than narrated. ISSUE-118's verdict
// collector is fed from the dispatch's unfiltered stream; when the dispatch errors,
// `violations` is nil and MandatedTestFailures has nothing to join — so
// test_verification cannot report the failing mandated test it was built to report.
// 118 is not a partial fix of 067; 067 is the supply failure upstream of 118's join.
func TestIssue067_VerdictCollectorIsPopulatedNotStarved(t *testing.T) {
	m := onlyRules(goToolchainManifest(t), "go-build", "go-test")
	stubSandboxedRunStdout(t, nil)
	runner := &fixtureRunner{
		byCmd: map[string][]byte{
			"go test":  readFixture(t, "go-test-build-failure.txt"),
			"go build": readFixture(t, "go-build-errors.txt"),
		},
		byCmdErr: map[string]error{"go test": &fakeExitError{code: 1}},
	}

	violations, err := dispatchPackEngines([]*pack.Manifest{m}, goToolchainPacksDir(t), t.TempDir(), nil, runner)
	if err != nil {
		t.Fatalf("dispatch must not error, or the collector is starved by construction: %v", err)
	}

	routed := gate.RouteTestVerdictFindings(violations)
	if len(routed) == 0 {
		t.Fatalf("ISSUE-118's verdict collector is STARVED — the test-typed stream is empty, so MandatedTestFailures has nothing to join; dispatch produced %#v", violations)
	}
}

// TestIssue067_ProducerScriptsExitZeroOnAGreenTree is THE B1 CLASS FALSIFIER and
// the only test here that executes the real producer scripts. Every other test in
// this file routes through fixtureRunner, which returns canned bytes and IGNORES
// the real args, so all of them stay green even if the producer scripts are
// catastrophically wrong.
//
// The arg vector is THE REAL ONE core hands the producer. splitCommand("go test")
// yields cmdName="go" + cmdArgs=["test"], and the swap replaces cmdName only — so
// the producer receives ["test", "./..."], subcommand INCLUDED. A producer whose
// body is `go test "$@"` therefore runs `go test test ./...`, which exits non-zero
// on a GREEN tree with `package test is not in std`. With crash_guard on both
// bindings that makes every gate run in every consumer permanently and opaquely
// RED — behind a pushed public tag. This test is what stops that from shipping.
func TestIssue067_ProducerScriptsExitZeroOnAGreenTree(t *testing.T) {
	module := writeGreenGoModule(t)
	packRoot := goToolchainPackRoot(t)
	manifest := goToolchainManifest(t)
	registry := resolveEngineRegistry(manifest)

	cases := []struct {
		engineName string
		argv       []string
	}{
		// Exactly what splitCommand(binding.Command) + the project-wide target
		// shaping produce for each binding.
		//
		// ISSUE-172: go-test's declared command gained -coverprofile=cover.out (the
		// single-run convention), so its real vector gained that flag — the drift
		// guard below is what caught it. This case now also exercises the stamp path
		// for real: the target IS `./...`, so a green run writes cover.out and the
		// .backstop/go-coverage-fresh stamp inside the throwaway module.
		{engineName: "go-test", argv: []string{"test", "-coverprofile=cover.out", "./..."}},
		{engineName: "go-build", argv: []string{"build", "./..."}},
	}

	for _, tc := range cases {
		t.Run(tc.engineName, func(t *testing.T) {
			binding, err := registry.Lookup(tc.engineName)
			if err != nil {
				t.Fatalf("resolve %s binding: %v", tc.engineName, err)
			}
			if binding.Producer == "" {
				t.Fatalf("%s declares no producer — there is no script to prove green", tc.engineName)
			}
			script := filepath.Join(packRoot, filepath.FromSlash(binding.Producer))

			// Confirm the invocation this test asserts is the one core builds, so a
			// change to the declared command cannot silently make this vector stale.
			gotName, gotArgs := splitCommand(binding.Command)
			gotArgs = append(gotArgs, binding.ProjectTarget)
			if gotName != "go" {
				t.Fatalf("premise: the swap replaces argv[0]; expected the tool to be argv[0], got %q", gotName)
			}
			if strings.Join(gotArgs, " ") != strings.Join(tc.argv, " ") {
				t.Fatalf("the real arg vector moved: core builds %v, this test asserts %v", gotArgs, tc.argv)
			}

			cmd := exec.Command(script, tc.argv...)
			cmd.Dir = module
			out, runErr := cmd.CombinedOutput()
			if runErr != nil {
				t.Fatalf("the producer MUST exit 0 on a green tree; running %s %v exited %v.\n"+
					"A body of `go %s \"$@\"` doubles the subcommand and fails exactly like this.\noutput:\n%s",
					script, tc.argv, runErr, tc.argv[0], string(out))
			}
		})
	}
}

// writeGreenGoModule writes a minimal, genuinely PASSING Go module into a temp dir
// and returns its path. It must compile, vet and test clean, or the producer test
// above would be measuring the module rather than the script.
func writeGreenGoModule(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	mkDirAll(t, filepath.Join(dir, "sub", "pkg"))
	writeFileStr(t, filepath.Join(dir, "go.mod"), "module greenprobe\n\ngo 1.24\n")
	writeFileStr(t, filepath.Join(dir, "sub", "pkg", "a.go"),
		"package pkg\n\n// Add sums two ints.\nfunc Add(a, b int) int { return a + b }\n")
	writeFileStr(t, filepath.Join(dir, "sub", "pkg", "a_test.go"),
		"package pkg\n\nimport \"testing\"\n\nfunc TestAdd(t *testing.T) {\n\tif Add(2, 3) != 5 {\n\t\tt.Fatal(\"bad sum\")\n\t}\n}\n")
	return dir
}

// TestIssue067_FixturePackDeclaresExecutableProducersOnBothEngines (CLM-013) is
// the in-repo half of the lockstep, asserted rather than assumed. A producer is
// invoked directly by path, so a script that lost its exec bit in transit fails as
// a never-started process — correctly refused, but the intent is that it not
// happen (sharp edge 6).
func TestIssue067_FixturePackDeclaresExecutableProducersOnBothEngines(t *testing.T) {
	manifest := goToolchainManifest(t)
	packRoot := goToolchainPackRoot(t)
	registry := resolveEngineRegistry(manifest)

	for _, engineName := range []string{"go-test", "go-build"} {
		binding, err := registry.Lookup(engineName)
		if err != nil {
			t.Fatalf("resolve %s binding: %v", engineName, err)
		}
		if binding.Producer == "" {
			t.Errorf("%s must declare a producer — its located diagnostics arrive on stderr, which core does not capture", engineName)
			continue
		}
		script := filepath.Join(packRoot, filepath.FromSlash(binding.Producer))
		info, statErr := os.Stat(script)
		if statErr != nil {
			t.Errorf("%s declares producer %q, which does not resolve under the pack root (%s): %v", engineName, binding.Producer, script, statErr)
			continue
		}
		if info.IsDir() {
			t.Errorf("%s's declared producer %q resolves to a directory", engineName, binding.Producer)
			continue
		}
		if info.Mode().Perm()&0o111 == 0 {
			t.Errorf("%s's producer %s must be executable (it is invoked directly by path); got mode %v", engineName, script, info.Mode().Perm())
		}
	}

	// The declaration must NOT spread by copy-paste into an engine that emits
	// native SARIF on stdout: merging stderr into golangci's payload would corrupt
	// the document. Enumerate the FINDINGS-typed bindings (the same predicate
	// dispatchPackCoverage routes on) and require exactly the two.
	wantProducers := map[string]bool{"go-test": true, "go-build": true}
	for name, spec := range manifest.Engines {
		if spec.Binding.GateType == engine.GateTypeCoverage {
			continue
		}
		hasProducer := spec.Binding.Producer != ""
		if hasProducer != wantProducers[name] {
			t.Errorf("findings engine %q: producer declared = %v, want %v", name, hasProducer, wantProducers[name])
		}
	}
}
