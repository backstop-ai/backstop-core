package main

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ISSUE-099 / PLAN-ISSUE-099 — `--json-out FILE`: ONE gate invocation renders the
// human table to stdout AND writes the gate/v1 JSON envelope to FILE.
//
// ★ THE RED IN THIS FILE IS EIGHT OF NINE, NOT NINE OF NINE, AND THAT IS DELIBERATE.
//
// (a) through (h) FAIL before the flag exists. The RED is direct and blunt rather
// than subtle — `gate --json-out <path>` dies in Cobra's flag parser with
// `Error: unknown flag: --json-out` and exit 2 (MEASURED 2026-08-18). Because that
// refusal happens BEFORE runGate is entered, those eight fail at RUNTIME, not at
// compile time: the package still builds with this file in it, which is what makes
// it safe to land in a shared tree ahead of the fix.
//
// TestGateJSONOut_RunsTheKillChainExactlyOnce — (i) — is GREEN AT HEAD BY DESIGN.
// It is a REGRESSION LOCK, not a description of missing behaviour, and anyone who
// finds it passing before the implementation should read its own comment rather
// than "fix" it into a red.
//
// HARNESS. Every behavioural test here drives the BUILT BINARY through
// buildBackstopBinary + runBackstopStreams, never the merged-output runBinary:
// against a single CombinedOutput buffer "the table went to stdout while the JSON
// went to the file" is unfalsifiable, and "the diagnostic is on stderr" passes for
// a command that wrote it to stdout. That is the same reason
// exit_surfacing_streams_test.go exists. NO_COLOR=1 is set wherever stdout text is
// matched, or the table arrives wrapped in ANSI.
//
// FIXTURE. gitInitRepoWithBackstop(t) is a real git repo whose bare `gate` exits
// ExitViolations today — asserted independently by
// TestExitSurfacing_GateViolations_NoDuplicateErrorLine — which is exactly the
// discriminating verdict CLM-004 and CLM-005 need. A passing fixture would make
// both of those vacuous.

// requireHumanTable asserts stdout carries the human table.
//
// The three markers are the fragments FormatHuman always writes, and they stand in
// for "the table rendered" both where it MUST be present (the headline test and the
// late-write-failure test) and, as `Gate Results` alone, where its ABSENCE is the
// proof a refusal happened before the kill chain ran. They are declared INSIDE the
// helper rather than as a package var: a package-level slice is mutable state that
// any test in this package could rewrite mid-suite, and go-standards'
// no-global-mutable-state rule blocks it.
func requireHumanTable(t *testing.T, stdout string) {
	t.Helper()
	for _, marker := range []string{"Gate Results", "Total violations:", "Steps:"} {
		if !strings.Contains(stdout, marker) {
			t.Errorf("stdout does not carry the human table marker %q\nstdout: %s", marker, stdout)
		}
	}
}

// requireGateV1Envelope reads path, asserts it parses as a gate/v1 JSON envelope,
// and returns the decoded document.
//
// The `steps` leg is not garnish: an empty `{}` would satisfy "parses as JSON" and
// a hand-written `{"schema_version":"gate/v1"}` would satisfy the version check,
// while proving nothing about the file being a real gate result.
func requireGateV1Envelope(t *testing.T, path string) map[string]interface{} {
	t.Helper()

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading the --json-out destination %s: %v", path, err)
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("%s is not valid JSON: %v\ncontents: %s", path, err, raw)
	}
	if got, _ := parsed["schema_version"].(string); got != "gate/v1" {
		t.Errorf("%s has schema_version %q, want %q", path, got, "gate/v1")
	}
	steps, ok := parsed["steps"].([]interface{})
	if !ok || len(steps) == 0 {
		t.Errorf("%s carries no steps; an empty object satisfies every other assertion here while proving "+
			"nothing about the file being a real gate envelope\ncontents: %s", path, raw)
	}
	return parsed
}

// TestGateJSONOut_WritesTheEnvelopeToFileWhileTheHumanTableStillRendersToStdout —
// CLM-001. THE HEADLINE CLAIM, in ONE invocation: neither surface replaces the
// other. Pre-fix this dies at `unknown flag: --json-out`.
func TestGateJSONOut_WritesTheEnvelopeToFileWhileTheHumanTableStillRendersToStdout(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	bin := buildBackstopBinary(t)
	proj := gitInitRepoWithBackstop(t)

	stdout, stderr, code := runBackstopStreams(t, bin, proj, "gate", "--json-out", "report.json")

	if code == ExitConfigError {
		t.Fatalf("gate --json-out exited %d (ExitConfigError) — the flag is unrecognized or its destination was "+
			"refused\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	requireHumanTable(t, stdout)

	// STDOUT CARRIES NO JSON DOCUMENT. --json-out ADDS a destination; it never
	// changes what stdout receives (CLM-003).
	if strings.Contains(stdout, `"schema_version"`) {
		t.Errorf("stdout carries a JSON document as well as the table; --json-out must not alter stdout\nstdout: %s", stdout)
	}
	if trimmed := strings.TrimSpace(stdout); strings.HasPrefix(trimmed, "{") {
		t.Errorf("stdout begins with %q — it is a JSON document, not the human table\nstdout: %s", trimmed[:1], stdout)
	}

	requireGateV1Envelope(t, filepath.Join(proj, "report.json"))
}

// TestGateJSONOut_CombinesWithJSONWritingTheSameDocumentToBothSurfaces — CLM-002,
// CLM-003. The two flags are INDEPENDENT and COMBINABLE: neither refuses the
// other, and one invocation carrying both emits the document to each destination.
//
// WHY BYTE-IDENTITY IS THE ASSERTION, rather than "both parse as gate/v1": the
// envelope carries a `generated_at` stamp and per-step `duration_ms` values, so
// two kill-chain runs could not produce identical bytes. Equality here is evidence
// that ONE GateResult was rendered ONCE and delivered twice — which is CLM-002 —
// and not that two runs happened to agree. The trailing newline is part of that
// contract, which is why the comparison is over raw bytes and not parsed maps.
func TestGateJSONOut_CombinesWithJSONWritingTheSameDocumentToBothSurfaces(t *testing.T) {
	bin := buildBackstopBinary(t)
	proj := gitInitRepoWithBackstop(t)

	stdout, stderr, code := runBackstopStreams(t, bin, proj, "gate", "--json", "--json-out", "report.json")

	if code == ExitConfigError {
		t.Fatalf("gate --json --json-out exited %d (ExitConfigError); the two flags must combine, not refuse each "+
			"other\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	fileBytes, err := os.ReadFile(filepath.Join(proj, "report.json"))
	if err != nil {
		t.Fatalf("reading report.json: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}
	if string(fileBytes) != stdout {
		t.Errorf("the file's bytes differ from stdout's bytes. Both are renderings of ONE GateResult, so they must "+
			"be identical byte for byte, trailing newline included\nfile (%d bytes): %q\nstdout (%d bytes): %q",
			len(fileBytes), string(fileBytes), len(stdout), stdout)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &parsed); err != nil {
		t.Fatalf("stdout under --json is not valid JSON: %v\nstdout: %s", err, stdout)
	}
	if got, _ := parsed["schema_version"].(string); got != "gate/v1" {
		t.Errorf("stdout has schema_version %q, want %q", got, "gate/v1")
	}
	if strings.Contains(stdout, "Gate Results") {
		t.Errorf("stdout carries the human table under --json; --json-out must not change what --json selects for "+
			"stdout\nstdout: %s", stdout)
	}
}

// TestGateJSONOut_RefusesAnEmptyValueBeforeRunningTheKillChain — CLM-006(i).
//
// The load-bearing leg is the ABSENCE of the gate's own report from stdout. An
// exit-2 that arrives AFTER a twenty-minute kill chain satisfies the exit-code
// assertion and defeats the entire point of refusing early, and absence is the
// only way to tell the two apart from outside the process.
func TestGateJSONOut_RefusesAnEmptyValueBeforeRunningTheKillChain(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	bin := buildBackstopBinary(t)
	proj := gitInitRepoWithBackstop(t)

	stdout, stderr, code := runBackstopStreams(t, bin, proj, "gate", "--json-out", "")

	if code != ExitConfigError {
		t.Errorf("exit %d, want ExitConfigError (%d) — a destination the operator got wrong is never a violations "+
			"verdict\nstdout: %s\nstderr: %s", code, ExitConfigError, stdout, stderr)
	}
	if !strings.Contains(stderr, "--json-out") {
		t.Errorf("stderr does not name the flag it is refusing\nstderr: %s", stderr)
	}
	if !strings.Contains(stderr, "path") {
		t.Errorf("stderr does not tell the operator a path is required\nstderr: %s", stderr)
	}
	if strings.Contains(stdout, "Gate Results") {
		t.Errorf("the gate's report reached stdout, so the refusal came AFTER the kill chain ran. The refusal must "+
			"cost seconds, not the ~21 minutes the chain costs on this repo's CI\nstdout: %s", stdout)
	}
}

// TestGateJSONOut_RefusesAMissingParentDirectoryBeforeRunningTheKillChain —
// CLM-006(ii). Same three legs as the empty-value refusal, with the message
// naming the missing DIRECTORY rather than only the file, so the operator is told
// which half of the path is wrong.
func TestGateJSONOut_RefusesAMissingParentDirectoryBeforeRunningTheKillChain(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	bin := buildBackstopBinary(t)
	proj := gitInitRepoWithBackstop(t)

	stdout, stderr, code := runBackstopStreams(t, bin, proj, "gate", "--json-out", "no-such-dir/report.json")

	if code != ExitConfigError {
		t.Errorf("exit %d, want ExitConfigError (%d)\nstdout: %s\nstderr: %s", code, ExitConfigError, stdout, stderr)
	}
	if !strings.Contains(stderr, "--json-out") {
		t.Errorf("stderr does not name the flag it is refusing\nstderr: %s", stderr)
	}
	if !strings.Contains(stderr, "no-such-dir") {
		t.Errorf("stderr does not name the missing DIRECTORY; naming only the file leaves the operator guessing "+
			"which half of the path is wrong\nstderr: %s", stderr)
	}
	if strings.Contains(stdout, "Gate Results") {
		t.Errorf("the gate's report reached stdout, so a typo'd path cost a full kill chain rather than two "+
			"seconds\nstdout: %s", stdout)
	}
}

// TestGateJSONOut_ResolvesARelativePathAgainstTheProcessWorkingDirectory —
// CLM-001. The value is an ordinary file argument: the OS resolves it against the
// PROCESS working directory, not against the discovered project root.
//
// Running from a subdirectory is the only shape in which those two differ —
// config discovery walks UP, so the gate still finds backstop.yml — and therefore
// the only shape that can falsify the rule.
func TestGateJSONOut_ResolvesARelativePathAgainstTheProcessWorkingDirectory(t *testing.T) {
	bin := buildBackstopBinary(t)
	proj := gitInitRepoWithBackstop(t)
	sub := filepath.Join(proj, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatalf("creating the subdirectory: %v", err)
	}

	stdout, stderr, code := runBackstopStreams(t, bin, sub, "gate", "--json-out", "report.json")

	if code == ExitConfigError {
		t.Fatalf("gate exited %d (ExitConfigError) from a subdirectory\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	requireGateV1Envelope(t, filepath.Join(sub, "report.json"))
	if _, err := os.Stat(filepath.Join(proj, "report.json")); err == nil {
		t.Errorf("the envelope landed at <project>/report.json — the value was resolved against the discovered "+
			"project root rather than against the process working directory")
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat <project>/report.json: %v", err)
	}
}

// TestGateJSONOut_WriteFailureIsAConfigErrorAfterTheTableRenders — CLM-006(iii).
//
// The load-bearing leg is that stdout DOES carry the full table: the stdout
// surface is the primary contract and is never suppressed by a secondary
// destination's failure. The run's verdict stays readable and the exit-2
// diagnostic follows it on stderr.
//
// ★ THE FAILURE IS MANUFACTURED WITH A DIRECTORY, NEVER WITH chmod. A read-only
// directory does not fail for root and CI containers run as root, so a chmod-based
// version of this test would pass locally and go VACUOUSLY GREEN in CI. os.WriteFile
// onto an existing directory fails deterministically for every uid — and it passes
// the parent-directory preflight by construction, which is exactly why that
// preflight checks only the PARENT.
func TestGateJSONOut_WriteFailureIsAConfigErrorAfterTheTableRenders(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	bin := buildBackstopBinary(t)
	proj := gitInitRepoWithBackstop(t)
	if err := os.MkdirAll(filepath.Join(proj, "report.json"), 0o755); err != nil {
		t.Fatalf("creating the directory that occupies the destination path: %v", err)
	}

	stdout, stderr, code := runBackstopStreams(t, bin, proj, "gate", "--json-out", "report.json")

	if code != ExitConfigError {
		t.Errorf("exit %d, want ExitConfigError (%d) — an unwritable destination is a configuration error, never a "+
			"violations verdict\nstdout: %s\nstderr: %s", code, ExitConfigError, stdout, stderr)
	}
	if !strings.Contains(stderr, "report.json") {
		t.Errorf("stderr does not name the destination it could not write\nstderr: %s", stderr)
	}
	requireHumanTable(t, stdout)
}

// TestGateJSONOut_ExitCodeIsTheGateVerdictNotTheOutputFlags — CLM-005. The output
// flags choose DESTINATIONS; the verdict alone chooses the exit code.
//
// ★ THE JSON DESTINATIONS ARE WRITTEN OUTSIDE THE PROJECT, in a second t.TempDir().
// The gate's diff scope INCLUDES UNTRACKED FILES, so an a.json written into the
// fixture by the third run would be in scope for the fourth — the two runs would
// then be gating different file sets, and this test's equality claim would quietly
// mean something other than what it says.
//
// The non-zero leg is what stops four vacuous zeros from satisfying the equality.
func TestGateJSONOut_ExitCodeIsTheGateVerdictNotTheOutputFlags(t *testing.T) {
	bin := buildBackstopBinary(t)
	proj := gitInitRepoWithBackstop(t)
	out := t.TempDir()

	invocations := [][]string{
		{"gate"},
		{"gate", "--json"},
		{"gate", "--json-out", filepath.Join(out, "a.json")},
		{"gate", "--json", "--json-out", filepath.Join(out, "b.json")},
	}

	codes := make([]int, 0, len(invocations))
	for _, args := range invocations {
		stdout, stderr, code := runBackstopStreams(t, bin, proj, args...)
		if code == ExitConfigError {
			t.Fatalf("`%s` exited %d (ExitConfigError); this test compares VERDICTS and a config refusal is not "+
				"one\nstdout: %s\nstderr: %s", strings.Join(args, " "), code, stdout, stderr)
		}
		codes = append(codes, code)
	}

	for i, code := range codes {
		if code != codes[0] {
			t.Errorf("`%s` exited %d but `%s` exited %d — the exit code is decided by the gate verdict alone, never "+
				"by which output destinations were requested",
				strings.Join(invocations[i], " "), code, strings.Join(invocations[0], " "), codes[0])
		}
	}
	if codes[0] == 0 {
		t.Errorf("all four invocations exited 0, so the equality above is four vacuous passes. This fixture's bare "+
			"gate must FAIL, or the claim that the flags do not move the code is untested")
	}
}

// TestGateJSONOut_WritesTheFileEvenWhenTheGateVerdictFails — CLM-004, asserted BY
// NAME rather than left as a side effect of the headline test.
//
// The write happens BEFORE the process decides its exit, so a run whose verdict is
// ExitViolations still leaves a complete envelope on disk. That is precisely the
// property ci.yml's `always()` upload depends on: it publishes a report for exactly
// the run worth diagnosing. It is therefore the property the two-step retirement
// rests on, and the reason this lane can collapse CI's duplicate invocation at all.
func TestGateJSONOut_WritesTheFileEvenWhenTheGateVerdictFails(t *testing.T) {
	bin := buildBackstopBinary(t)
	proj := gitInitRepoWithBackstop(t)

	stdout, stderr, code := runBackstopStreams(t, bin, proj, "gate", "--json-out", "report.json")

	if code != ExitViolations {
		t.Fatalf("exit %d, want ExitViolations (%d) — this fixture's gate must FAIL, or the claim that the file "+
			"survives a failing verdict is untested\nstdout: %s\nstderr: %s", code, ExitViolations, stdout, stderr)
	}
	requireGateV1Envelope(t, filepath.Join(proj, "report.json"))
}

// TestGateJSONOut_RunsTheKillChainExactlyOnce — CLM-002, and A SOURCE PIN rather
// than a measurement.
//
// ★ THIS TEST PASSES AT HEAD, BEFORE THE IMPLEMENTATION, AND THAT IS CORRECT. It is
// a REGRESSION LOCK on a property that already holds (VERIFIED 2026-08-18: runGate
// contains exactly one `gate.New(` and one `.Run(` today), not a description of
// missing behaviour. Its job is to stop the file write from being implemented by
// RE-RUNNING the kill chain — the one structural mistake that would satisfy every
// behavioural test in this file while reintroducing the exact ~21-minute cost
// ISSUE-099 exists to remove. Anyone who finds it green before the fix should read
// this comment rather than weaken it into a red.
//
// WHY A SOURCE PIN. A doubled kill chain is observable only as wall time or as
// engine side effects, neither of which a hermetic unit test can assert without
// flake. "The file-emitting path was implemented by re-running the gate" is a
// STRUCTURAL mistake, and a structural pin catches it exactly. Its behavioural
// counterpart is TestGateJSONOut_CombinesWithJSONWritingTheSameDocumentToBothSurfaces,
// whose byte-identity leg covers the same claim from the outside.
//
// runGate is located BY NAME, never by a line number: cmd/backstop/gate.go is ~2000
// lines and has moved under three lanes this week.
func TestGateJSONOut_RunsTheKillChainExactlyOnce(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "gate.go", nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("parsing gate.go: %v", err)
	}

	var runGateDecl *ast.FuncDecl
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if ok && fn.Recv == nil && fn.Name.Name == "runGate" {
			runGateDecl = fn
			break
		}
	}
	if runGateDecl == nil {
		t.Fatalf("gate.go declares no func runGate; this pin locates its subject BY NAME and has just lost it")
	}

	newCalls, runCalls := 0, 0
	ast.Inspect(runGateDecl.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if pkg, ok := sel.X.(*ast.Ident); ok && pkg.Name == "gate" && sel.Sel.Name == "New" {
			newCalls++
		}
		if sel.Sel.Name == "Run" {
			runCalls++
		}
		return true
	})

	if newCalls != 1 {
		t.Errorf("runGate constructs %d gates (`gate.New(`), want exactly 1. A second construction means the "+
			"kill chain was doubled — the ~21-minute-per-push cost ISSUE-099 exists to remove", newCalls)
	}
	if runCalls != 1 {
		t.Errorf("runGate makes %d `.Run(` calls, want exactly 1. The --json-out destination must be a second "+
			"RENDERING of one result, never a second RUN of the gate", runCalls)
	}
}
