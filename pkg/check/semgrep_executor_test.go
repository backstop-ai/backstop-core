package check

import (
	"context"
	"testing"
)

// containsConfigFor reports whether args contains a `--config <path>` pair for
// the given path.
func containsConfigFor(args []string, path string) bool {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == "--config" && args[i+1] == path {
			return true
		}
	}
	return false
}

func containsArg(args []string, want string) bool {
	for _, a := range args {
		if a == want {
			return true
		}
	}
	return false
}

// TestCodeCheck_SemgrepExecutor_RunsProjectAndPackConfigs verifies that
// semgrepExecutor invokes the EnsureSemgrep-resolved binary with --json, a
// --config for the project rules dir, AND a --config for every
// extraSemgrepConfigs path, then maps results[] to Violations with correct
// File (.path), Line (.start.line), Message (.extra.message), and Severity
// (.extra.severity ERROR->error, WARNING->warning). (CLM-005)
func TestCodeCheck_SemgrepExecutor_RunsProjectAndPackConfigs(t *testing.T) {
	const binPath = "/fake/tools/semgrep"
	runner := &fakeRunner{outputs: map[string][]byte{binPath: []byte(fixtureSemgrepFindings)}}
	ensurer := &mockSemgrepEnsurer{fn: func(_, _ string) (string, error) { return binPath, nil }}

	packConfigA := "/proj/.backstop/packs/slotly/go-standards/rules/a.yml"
	packConfigB := "/proj/.backstop/packs/acme/ts-standards/rules/b.yml"

	e := &semgrepExecutor{
		runner:              runner,
		ensurer:             ensurer,
		backstopDir:         "/proj/.backstop",
		manifestDir:         "/proj/.backstop/rules",
		extraSemgrepConfigs: []string{packConfigA, packConfigB},
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
	if !containsConfigFor(call.args, "/proj/.backstop/rules") {
		t.Errorf("args %v missing --config for project rules dir", call.args)
	}
	if !containsConfigFor(call.args, packConfigA) {
		t.Errorf("args %v missing --config for pack config A", call.args)
	}
	if !containsConfigFor(call.args, packConfigB) {
		t.Errorf("args %v missing --config for pack config B", call.args)
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
		manifestDir: "/proj/.backstop/rules",
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
