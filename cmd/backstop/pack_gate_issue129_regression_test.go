package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/gate"
	"github.com/backstop-ai/backstop-core/pkg/pack"
)

// issue129Repo materializes the named files in a fresh temp dir, commits them as
// the git baseline, then MODIFIES the first one so a GateScopeModeDiff scope
// derived from real git state contains exactly that file. The scope is genuinely
// git-derived (not a diffScope literal), which is what makes the out-of-scope
// claim honest rather than assumed.
func issue129Repo(t *testing.T, changed string, alsoCommit ...string) *gate.GateScope {
	t.Helper()
	root := t.TempDir()
	for _, f := range append([]string{changed}, alsoCommit...) {
		abs := filepath.Join(root, f)
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, []byte("package p\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	diagInitCommitRepo(t, root)
	abs := filepath.Join(root, changed)
	if err := os.WriteFile(abs, []byte("package p\n\nfunc Changed() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	scope, err := gate.ComputeGateScope(root, gate.GateScopeModeDiff, nil)
	if err != nil {
		t.Fatalf("ComputeGateScope diff: %v", err)
	}
	if scope.Mode != gate.GateScopeModeDiff {
		t.Fatalf("expected a diff-mode scope, got %q", scope.Mode)
	}
	if !scope.Contains(changed) {
		t.Fatalf("the modified file %q must be in the git-derived diff scope; got %#v", changed, scope.Files)
	}
	return scope
}

// TestIssue129_OutOfScopeTestFailureRedsDiffScopedGate is the ISSUE-129 regression
// guard: a genuinely failing Go test whose file sits OUTSIDE the diff scope must
// still RED a diff-scoped `backstop gate` — the default invocation, and the only
// blocking check CI runs on a PR.
//
// It drives the REAL path end to end: the go-test binding is read from the DECLARED
// go-toolchain manifest (not a struct literal), the failure text is the REAL CAPTURED
// `go test` output fixture, it flows through the real convert script and the real
// dispatchPackEngines, and the survivors are decided by the real pkg/gate/scope.go
// filterViolations via filterThroughGate — a stamp-site-only assertion would not
// prove the gate's verdict.
//
// Before the fix, go-test declared no exempt_from_scope_filter, so every violation
// arrived ProjectWide=false and filterViolations dropped all of them: a full PASS
// with a broken test suite in the tree. After it, the violations survive
// (SPEC-041 CLM-015; ISSUE-129 CLM-002/CLM-003). The in-scope half (CLM-005) still
// reds too — exemption widens what is reported, it never narrows it.
func TestIssue129_OutOfScopeTestFailureRedsDiffScopedGate(t *testing.T) {
	m := onlyRules(goToolchainManifest(t), "go-test")
	stubSandboxedRunStdout(t, nil)
	runner := &fixtureRunner{byCmd: map[string][]byte{"go test": readFixture(t, "go-test-failures.txt")}}

	violations, err := dispatchPackEngines([]*pack.Manifest{m}, goToolchainPacksDir(t), t.TempDir(), nil, runner)
	if err != nil {
		t.Fatalf("dispatchPackEngines (test): %v", err)
	}
	if len(violations) == 0 {
		t.Fatal("expected go-test violations from the real captured failure output")
	}

	// Every violation must carry ProjectWide stamped from the DECLARED binding —
	// this is the pack-data bridge, resolved per-violation, with no engine-by-name
	// special case anywhere in core.
	for _, v := range violations {
		if !v.ProjectWide {
			t.Errorf("go-test violation %q (%s) must arrive ProjectWide=true, stamped from the declared exempt_from_scope_filter (ISSUE-129 CLM-002)", v.Message, v.File)
		}
	}

	// ── CLM-002/CLM-003: OUT of scope. The diff contains only an unrelated file. ──
	outScope := issue129Repo(t, filepath.Join("cmd", "unrelated", "main.go"))
	for _, v := range violations {
		if outScope.Contains(v.File) {
			t.Fatalf("fixture precondition broken: failing test file %q must NOT be in the diff scope %#v", v.File, outScope.Files)
		}
	}
	survived := filterThroughGate(t, outScope, violations)
	if len(survived) == 0 {
		t.Fatalf("a failing Go test in a file OUTSIDE the diff scope must SURVIVE the real filter and RED a diff-scoped gate; all %d violations were discarded (ISSUE-129 CLM-002/CLM-003)", len(violations))
	}

	// ── CLM-005: IN scope. A touched failing test file still reds, as it always did. ──
	inScope := issue129Repo(t, violations[0].File)
	if !inScope.Contains(violations[0].File) {
		t.Fatalf("fixture precondition broken: %q must be in the in-scope diff %#v", violations[0].File, inScope.Files)
	}
	if len(filterThroughGate(t, inScope, violations)) == 0 {
		t.Fatal("an IN-SCOPE test failure must still RED (you-touch-it-you-fix-it): the exemption widens what is reported, never narrows it (ISSUE-129 CLM-005)")
	}
}
