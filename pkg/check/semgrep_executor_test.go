package check

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// configPaths returns every value that follows a `--config` flag in args.
func configPaths(args []string) []string {
	var paths []string
	for i := 0; i+1 < len(args); i++ {
		if args[i] == "--config" {
			paths = append(paths, args[i+1])
		}
	}
	return paths
}

func containsArg(args []string, want string) bool {
	for _, a := range args {
		if a == want {
			return true
		}
	}
	return false
}

// hasConfigUnderRules reports whether any `--config` path points inside a
// `.backstop/rules` directory — the deleted compiled-standards rule source.
func hasConfigUnderRules(args []string) bool {
	for _, p := range configPaths(args) {
		if strings.Contains(filepath.ToSlash(p), ".backstop/rules") {
			return true
		}
	}
	return false
}

// TestSemgrepExecutor_NoConfigEver verifies that after SPEC-031 the in-process
// semgrepExecutor invokes semgrep with NO --config at all. Installed-pack
// semgrep rules are dispatched group-by-engine in cmd/backstop's
// dispatchPackEngines (the migrated pack-feeder coverage lives in the dispatch
// tests), so the executor's only job is the project's own semgrep pass:
// --json --quiet <files>, no rule-config source. (REQ-011)
func TestSemgrepExecutor_NoConfigEver(t *testing.T) {
	const binPath = "/fake/tools/semgrep"
	runner := &fakeRunner{outputs: map[string][]byte{binPath: []byte(`{"results":[]}`)}}
	ensurer := &mockSemgrepEnsurer{fn: func(_, _ string) (string, error) { return binPath, nil }}

	e := &semgrepExecutor{
		runner:      runner,
		ensurer:     ensurer,
		backstopDir: "/proj/.backstop",
	}

	if _, err := e.Execute(context.Background(), []string{"pkg/server/handler.go"}); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	call := runner.lastCall()
	if containsArg(call.args, "--config") {
		t.Errorf("args %v carry a --config flag; the executor no longer feeds pack configs", call.args)
	}
	if !containsArg(call.args, "--json") {
		t.Errorf("args %v missing --json", call.args)
	}
}

// TestSemgrepExecutor_StructHasNoPackFeeder constructs the semgrepExecutor with
// an UNKEYED (positional) composite literal of exactly four field values
// (runner, ensurer, backstopDir, pinnedVersion). If the retired
// extraSemgrepConfigs field — or any other field — is added back, this literal
// fails to compile, so the struct's field set is pinned exactly after the
// pack-feeder field is removed. (REQ-011)
func TestSemgrepExecutor_StructHasNoPackFeeder(t *testing.T) {
	const binPath = "/fake/tools/semgrep"
	runner := &fakeRunner{outputs: map[string][]byte{binPath: []byte(`{"results":[]}`)}}
	ensurer := &mockSemgrepEnsurer{fn: func(_, _ string) (string, error) { return binPath, nil }}

	// Unkeyed: runner, ensurer, backstopDir, pinnedVersion.
	e := &semgrepExecutor{
		runner,
		ensurer,
		"/proj/.backstop",
		"",
	}

	if _, err := e.Execute(context.Background(), []string{"pkg/server/handler.go"}); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	call := runner.lastCall()
	if containsArg(call.args, "--config") {
		t.Errorf("args %v carry a --config flag; the executor must yield none", call.args)
	}
}

// TestNoTestRequiresManifestDirOrPackFeeder is a source self-check over the
// pkg/check test files: it asserts neither the removed manifestDir field token
// nor the retired extraSemgrepConfigs field token survives in any
// semgrepExecutor struct literal, so the green go-test guarantee is enforced
// rather than assumed. (REQ-011)
func TestNoTestRequiresManifestDirOrPackFeeder(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("reading pkg/check dir: %v", err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, "_test.go") {
			continue
		}
		data, readErr := os.ReadFile(name)
		if readErr != nil {
			t.Fatalf("reading %s: %v", name, readErr)
		}
		src := string(data)
		// Build the search tokens from parts so this self-check file does not
		// match itself when it scans its own source.
		manifestFieldToken := "manifest" + "Dir:"
		if strings.Contains(src, manifestFieldToken) {
			t.Errorf("%s still constructs a semgrepExecutor with a %s field; the field is removed", name, manifestFieldToken)
		}
		packFeederToken := "extraSemgrep" + "Configs:"
		if strings.Contains(src, packFeederToken) {
			t.Errorf("%s still constructs a semgrepExecutor with the retired %s field", name, packFeederToken)
		}
	}
}

// TestCodeCheck_SemgrepExecutor_RunsProjectAndMapsResults verifies that the
// semgrepExecutor invokes the EnsureSemgrep-resolved binary with --json (no
// pack --config — that path moved to dispatch), then maps results[] to
// Violations with correct File (.path), Line (.start.line), Message
// (.extra.message), and Severity (.extra.severity ERROR->error,
// WARNING->warning). Re-keyed from the retired
// TestCodeCheck_SemgrepExecutor_RunsProjectAndPackConfigs: the pack-config
// assertions are dropped (their coverage moved to the dispatchPackEngines
// tests), the project-rules --json/results-to-Violations coverage is retained.
func TestCodeCheck_SemgrepExecutor_RunsProjectAndMapsResults(t *testing.T) {
	const binPath = "/fake/tools/semgrep"
	runner := &fakeRunner{outputs: map[string][]byte{binPath: []byte(fixtureSemgrepFindings)}}
	ensurer := &mockSemgrepEnsurer{fn: func(_, _ string) (string, error) { return binPath, nil }}

	e := &semgrepExecutor{
		runner:      runner,
		ensurer:     ensurer,
		backstopDir: "/proj/.backstop",
	}

	res, err := e.Execute(context.Background(), []string{"pkg/server/handler.go"})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if res.Pass != CheckTypeSemgrep {
		t.Errorf("Pass = %v, want %v", res.Pass, CheckTypeSemgrep)
	}

	// Binary resolved via EnsureSemgrep must be the invoked command.
	call := runner.lastCall()
	if call.name != binPath {
		t.Fatalf("invoked %q, want EnsureSemgrep-resolved binary %q", call.name, binPath)
	}
	if !containsArg(call.args, "--json") {
		t.Errorf("args %v missing --json", call.args)
	}
	// The pack-config feeder is retired: no --config and no standards-dir config.
	if containsArg(call.args, "--config") {
		t.Errorf("args %v carry a --config flag; pack rule dispatch is no longer fed here", call.args)
	}
	if hasConfigUnderRules(call.args) {
		t.Errorf("args %v carry a standards-dir (.backstop/rules) --config", call.args)
	}

	if len(res.Violations) != 3 {
		t.Fatalf("got %d violations, want 3: %+v", len(res.Violations), res.Violations)
	}

	v := findViolation(res.Violations, "pkg/server/handler.go", 31)
	if v == nil {
		t.Fatal("missing violation for handler.go:31")
	}
	if v.Message != "panic() is disallowed; return an error instead" {
		t.Errorf("Message = %q", v.Message)
	}
	if v.Severity != "error" {
		t.Errorf("Severity = %q, want error (ERROR->error)", v.Severity)
	}
	if v.Pass != CheckTypeSemgrep {
		t.Errorf("Pass = %v, want semgrep", v.Pass)
	}

	if w := findViolation(res.Violations, "cmd/app/main.go", 12); w == nil {
		t.Error("missing violation for main.go:12")
	} else if w.Severity != "warning" {
		t.Errorf("Severity = %q, want warning (WARNING->warning)", w.Severity)
	}
}

// TestCodeCheck_SemgrepExecutor_ToleratesNonJSONPreamble verifies that
// semgrepExecutor parses findings from combined output that carries non-JSON
// UTF-8 banner/progress bytes BEFORE the JSON document — reproducing the live
// `invalid character 'â' looking for beginning of value` failure (ISSUE-006)
// and pinning its fix: the JSON document is extracted from the preamble before
// unmarshaling. (CLM-001)
func TestCodeCheck_SemgrepExecutor_ToleratesNonJSONPreamble(t *testing.T) {
	const binPath = "/fake/tools/semgrep"
	runner := &fakeRunner{outputs: map[string][]byte{binPath: []byte(fixtureSemgrepFindingsWithPreamble)}}
	ensurer := &mockSemgrepEnsurer{fn: func(_, _ string) (string, error) { return binPath, nil }}

	e := &semgrepExecutor{
		runner:      runner,
		ensurer:     ensurer,
		backstopDir: "/proj/.backstop",
	}

	res, err := e.Execute(context.Background(), []string{"pkg/server/handler.go"})
	if err != nil {
		t.Fatalf("Execute returned error on output with non-JSON preamble: %v", err)
	}
	// The preamble fixture wraps the SAME JSON document as fixtureSemgrepFindings,
	// so the parsed findings must match exactly (three findings).
	if len(res.Violations) != 3 {
		t.Fatalf("got %d violations, want 3: %+v", len(res.Violations), res.Violations)
	}
	v := findViolation(res.Violations, "pkg/server/handler.go", 31)
	if v == nil {
		t.Fatal("missing violation for handler.go:31 — JSON document not extracted from preamble")
	}
	if v.Message != "panic() is disallowed; return an error instead" {
		t.Errorf("Message = %q, want the extracted finding message", v.Message)
	}
	if v.File != "pkg/server/handler.go" || v.Line != 31 {
		t.Errorf("File/Line = %q/%d, want pkg/server/handler.go/31", v.File, v.Line)
	}
}

// TestCodeCheck_SemgrepExecutor_QuietFlagPassed verifies that the semgrep
// invocation includes --quiet (suppresses non-JSON banner/progress output)
// alongside --json. (CLM-002)
func TestCodeCheck_SemgrepExecutor_QuietFlagPassed(t *testing.T) {
	const binPath = "/fake/tools/semgrep"
	runner := &fakeRunner{outputs: map[string][]byte{binPath: []byte(fixtureSemgrepFindings)}}
	ensurer := &mockSemgrepEnsurer{fn: func(_, _ string) (string, error) { return binPath, nil }}

	e := &semgrepExecutor{
		runner:      runner,
		ensurer:     ensurer,
		backstopDir: "/proj/.backstop",
	}

	if _, err := e.Execute(context.Background(), []string{"pkg/server/handler.go"}); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	call := runner.lastCall()
	if !containsArg(call.args, "--quiet") {
		t.Errorf("args %v missing --quiet", call.args)
	}
	if !containsArg(call.args, "--json") {
		t.Errorf("args %v missing --json", call.args)
	}
}

// TestCodeCheck_SemgrepExecutor_PreservesPackNamespacedRuleIDs verifies that a
// finding whose check_id is pack-namespaced (e.g. "slotly/go-standards/no-panic")
// preserves that exact rule ID on the resulting Violation for source-pack
// attribution. (CLM-006)
func TestCodeCheck_SemgrepExecutor_PreservesPackNamespacedRuleIDs(t *testing.T) {
	const binPath = "/fake/tools/semgrep"
	runner := &fakeRunner{outputs: map[string][]byte{binPath: []byte(fixtureSemgrepFindings)}}
	ensurer := &mockSemgrepEnsurer{fn: func(_, _ string) (string, error) { return binPath, nil }}

	e := &semgrepExecutor{
		runner:      runner,
		ensurer:     ensurer,
		backstopDir: "/proj/.backstop",
	}

	res, err := e.Execute(context.Background(), []string{"pkg/server/handler.go"})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	v := findViolation(res.Violations, "pkg/server/handler.go", 31)
	if v == nil {
		t.Fatal("missing violation for the pack-namespaced finding")
	}
	if v.Rule != "slotly/go-standards/no-panic" {
		t.Errorf("Rule = %q, want the verbatim pack-namespaced ID slotly/go-standards/no-panic", v.Rule)
	}

	// A second pack-namespaced rule must also survive verbatim.
	if c := findViolation(res.Violations, "pkg/server/client.go", 8); c == nil {
		t.Error("missing violation for client.go:8")
	} else if c.Rule != "slotly/go-standards/require-context" {
		t.Errorf("Rule = %q, want slotly/go-standards/require-context", c.Rule)
	}

	// A non-namespaced check_id is preserved as-is too.
	if p := findViolation(res.Violations, "cmd/app/main.go", 12); p == nil {
		t.Error("missing violation for main.go:12")
	} else if p.Rule != "no-fmt-println" {
		t.Errorf("Rule = %q, want no-fmt-println", p.Rule)
	}
}

// TestNoFallback_LeftoverCompiledRulesIgnored verifies that a planted
// STD-*.semgrep.yml in .backstop/rules/ is NOT collected as a --config path:
// the recorded argv contains no path under .backstop/rules/. The executor
// composes no --config at all. (CLM-018)
func TestNoFallback_LeftoverCompiledRulesIgnored(t *testing.T) {
	const binPath = "/fake/tools/semgrep"
	dir := t.TempDir()
	rulesDir := filepath.Join(dir, ".backstop", "rules")
	if err := os.MkdirAll(rulesDir, 0o755); err != nil {
		t.Fatalf("mkdir rules: %v", err)
	}
	leftover := filepath.Join(rulesDir, "STD-GO-001.semgrep.yml")
	if err := os.WriteFile(leftover, []byte("rules: []\n"), 0o644); err != nil {
		t.Fatalf("write leftover: %v", err)
	}

	runner := &fakeRunner{outputs: map[string][]byte{binPath: []byte(`{"results":[]}`)}}
	ensurer := &mockSemgrepEnsurer{fn: func(_, _ string) (string, error) { return binPath, nil }}

	e := &semgrepExecutor{
		runner:      runner,
		ensurer:     ensurer,
		backstopDir: filepath.Join(dir, ".backstop"),
	}

	if _, err := e.Execute(context.Background(), []string{"main.go"}); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	call := runner.lastCall()
	if hasConfigUnderRules(call.args) {
		t.Errorf("args %v collected a leftover .backstop/rules/ path as --config; it must be ignored", call.args)
	}
	if containsArg(call.args, "--config") {
		t.Errorf("args %v carry a --config flag; the executor composes none", call.args)
	}
}

// TestNoFallback_PopulatedRulesDirNotASource verifies that a populated
// .backstop/rules/ directory does not become an implicit rule source: semgrep
// still runs with no --config even when .backstop/rules/ contains files.
// (CLM-019)
func TestNoFallback_PopulatedRulesDirNotASource(t *testing.T) {
	const binPath = "/fake/tools/semgrep"
	dir := t.TempDir()
	rulesDir := filepath.Join(dir, ".backstop", "rules")
	if err := os.MkdirAll(rulesDir, 0o755); err != nil {
		t.Fatalf("mkdir rules: %v", err)
	}
	for _, name := range []string{"STD-GO-001.semgrep.yml", "STD-GO-001.manifest.json", "STD-GO-001.native.json"} {
		if err := os.WriteFile(filepath.Join(rulesDir, name), []byte("{}"), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	runner := &fakeRunner{outputs: map[string][]byte{binPath: []byte(`{"results":[]}`)}}
	ensurer := &mockSemgrepEnsurer{fn: func(_, _ string) (string, error) { return binPath, nil }}

	e := &semgrepExecutor{
		runner:      runner,
		ensurer:     ensurer,
		backstopDir: filepath.Join(dir, ".backstop"),
	}

	if _, err := e.Execute(context.Background(), []string{"main.go"}); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	call := runner.lastCall()
	if containsArg(call.args, "--config") {
		t.Errorf("args %v carry a --config flag; a populated .backstop/rules/ must not be an implicit source", call.args)
	}
}
