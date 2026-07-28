package main

// SPEC-035 TASK-002 — trusted-tool allowlist + lock-pin test harness.
//
// This file scaffolds the harness the Phase-2/Phase-4 allowlist tests consume to
// drive the trust-gate matrix WITHOUT a live tool and WITHOUT stubbing the
// allowlist OPEN (Sharp Edge 3 / the spec's Verification section forbids a
// stub-open allowlist on the dispatch path under test). It provides three things
// the trust-gate claims (CLM-005..008, CLM-029) require:
//
//   1. loadAllowlistFixtures: a REAL stub allowlist map {tool -> pinned version}
//      and a stub lock-pinned-version source {tool -> locked version}, loaded
//      from testdata/allowlist-fixtures.yml — with a genuine ABSENT-tool cell
//      (absentTool) and a genuine present-but-unpinned / version-divergent cell.
//   2. allowlistFixtures.lockedVersion: the stub lock read that mirrors the
//      backstop.lock / VerifyLock path, so a caller feeds the locked version into
//      CheckToolAllowed as the `lockedVersion` argument (the pin rides the lock,
//      never a second literal — CLM-029).
//   3. recordingStdoutRunner: a fake check.CommandRunner that RECORDS whether
//      RunStdout was ever called, so CLM-008 can assert an un-allowlisted tool's
//      command was NEVER handed to the runner (the gate sits BEFORE
//      splitCommand/RunStdout).
//
// The allowlist PRODUCTION code (engine.TrustedToolAllowlist / CheckToolAllowed)
// is Phase 2 — this file references neither, so it compiles before Phase 2 lands.
// The Phase-4 dispatch tests wire the loaded allowlist + lockedVersion into the
// real CheckToolAllowed gate.

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/check"

	yaml "gopkg.in/yaml.v3"
)

// allowlistFixtures is the parsed testdata/allowlist-fixtures.yml: a real stub
// allowlist (the backstop-owned {tool -> pinned version} trust floor) and a stub
// lock source (the {tool -> locked version} read mirroring backstop.lock).
type allowlistFixtures struct {
	// Allowlist is the {tool -> pinned version} trust floor. A tool absent from
	// this map may not be run by any pack-declared command (CLM-006). It is NOT
	// stubbed open — absentTool() names a tool deliberately missing from it.
	Allowlist map[string]string `yaml:"allowlist"`
	// Lock is the {tool -> locked version} stub mirroring the backstop.lock /
	// VerifyLock read. A caller feeds Lock[tool] into CheckToolAllowed as the
	// lockedVersion argument (CLM-029) — the pin rides the lock, not a literal.
	Lock map[string]string `yaml:"lock"`
}

// loadAllowlistFixtures reads testdata/allowlist-fixtures.yml into a real (non
// stub-open) allowlist + lock pair. It fatals rather than returning an error so
// callers stay terse; the fixture is part of the test's contract.
func loadAllowlistFixtures(t *testing.T) allowlistFixtures {
	t.Helper()
	path := filepath.Join("testdata", "allowlist-fixtures.yml")
	raw, err := os.ReadFile(path) // #nosec G304 — fixed in-repo test fixture path
	if err != nil {
		t.Fatalf("read allowlist fixtures %s: %v", path, err)
	}
	var f allowlistFixtures
	if err := yaml.Unmarshal(raw, &f); err != nil {
		t.Fatalf("unmarshal allowlist fixtures: %v", err)
	}
	if len(f.Allowlist) == 0 {
		t.Fatalf("allowlist fixture is empty — a stub-open/empty allowlist proves nothing (Sharp Edge 3)")
	}
	return f
}

// absentTool returns a tool name that is GENUINELY absent from the allowlist —
// the real ABSENT-tool cell CLM-006/CLM-008 require (not a stub-open hole). It
// asserts the absence so a future fixture edit that accidentally adds the tool
// fails loudly here instead of silently weakening the test.
func (f allowlistFixtures) absentTool(t *testing.T) string {
	t.Helper()
	const tool = "acme-absent"
	if _, present := f.Allowlist[tool]; present {
		t.Fatalf("fixture invariant broken: %q must be ABSENT from the allowlist to drive the un-allowlisted cell (CLM-006)", tool)
	}
	return tool
}

// lockedVersion returns the stub lock-pinned version for a tool (mirroring the
// backstop.lock / VerifyLock read), and whether the tool is pinned in the lock
// at all. A caller passes this value into CheckToolAllowed as the lockedVersion
// argument so the version pin rides the lock and cannot drift from a second
// source (CLM-029). A tool present on the allowlist but absent from the lock
// (pinned=false) is the allowlisted-but-unpinned cell (CLM-007).
func (f allowlistFixtures) lockedVersion(tool string) (version string, pinned bool) {
	v, ok := f.Lock[tool]
	return v, ok
}

// recordingStdoutRunner is a check.CommandRunner that records whether RunStdout
// was ever invoked (and with what command), so the dispatch trust-gate test
// (CLM-008) can assert an un-allowlisted tool's command is NEVER handed to the
// runner — the gate sits BEFORE splitCommand/RunStdout. It is intentionally
// distinct from the package's existing recordingRunner: that type folds Run and
// RunStdout into one slice, whereas CLM-008 specifically needs to know whether
// RunStdout (the dispatch path) ran. It never shells out to a live tool.
type recordingStdoutRunner struct {
	// runStdoutCalls records every RunStdout invocation's command name. Its
	// emptiness is the CLM-008 assertion surface (gate-blocked => no call).
	runStdoutCalls []string
	// runCalls records Run invocations for completeness; the gate guards the
	// RunStdout dispatch path, but recording both keeps the fake honest.
	runCalls []string
	// stdout is the canned stdout returned on a (gate-passed) RunStdout call —
	// empty by default (a clean, finding-free run).
	stdout []byte
}

// Run records the invocation and returns the canned (empty) output. The trust
// gate guards the RunStdout dispatch path; Run is recorded only for honesty.
func (r *recordingStdoutRunner) Run(_ context.Context, name string, _ ...string) ([]byte, error) {
	r.runCalls = append(r.runCalls, name)
	return r.stdout, nil
}

// RunStdout records that the dispatch path actually handed a command to the
// runner. If the trust gate blocks (un-allowlisted/unpinned tool), this MUST
// never be called — len(runStdoutCalls) == 0 is the CLM-008 assertion.
func (r *recordingStdoutRunner) RunStdout(_ context.Context, name string, _ ...string) ([]byte, error) {
	r.runStdoutCalls = append(r.runStdoutCalls, name)
	return r.stdout, nil
}

// runStdoutWasCalled reports whether the dispatch path ever reached RunStdout.
// The CLM-008 gate-blocks-before-RunStdout test asserts this is false for an
// un-allowlisted tool.
func (r *recordingStdoutRunner) runStdoutWasCalled() bool {
	return len(r.runStdoutCalls) > 0
}

// Compile-time assertion that the recording fake satisfies the same
// check.CommandRunner contract the real dispatch path consumes — so the harness
// can be dropped into runFindingsEngine's runner slot once the trust gate lands.
var _ check.CommandRunner = (*recordingStdoutRunner)(nil)
