package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// cmdBackstopDir returns the cmd/backstop package directory.
func cmdBackstopDir(t *testing.T) string {
	t.Helper()
	return filepath.Join(repoRoot(t), "cmd", "backstop")
}

// TestSharedRunner_Eradicated asserts cmd/backstop/shared_testrun.go is deleted
// and none of its symbols (sharedTestRunner/newSharedTestRunner/
// isWholeModuleGoTest/wholeModuleTest) survive anywhere in cmd/backstop non-test
// source (CLM-004).
func TestSharedRunner_Eradicated(t *testing.T) {
	if _, err := os.Stat(filepath.Join(cmdBackstopDir(t), "shared_testrun.go")); !os.IsNotExist(err) {
		t.Errorf("cmd/backstop/shared_testrun.go must be deleted (CLM-004); stat err=%v", err)
	}
	for _, sym := range []string{"sharedTestRunner", "newSharedTestRunner", "isWholeModuleGoTest", "wholeModuleTest"} {
		if grepNonTestSource(t, cmdBackstopDir(t), sym) {
			t.Errorf("shared-runner symbol %q still present in cmd/backstop non-test source — it must be eradicated (CLM-004)", sym)
		}
	}
}

// TestSharedRunner_WiringRemovedFromGate asserts gate.go no longer constructs the
// shared runner, injects it, or threads it into buildCoverageStep (CLM-005).
func TestSharedRunner_WiringRemovedFromGate(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(cmdBackstopDir(t), "gate.go"))
	if err != nil {
		t.Fatalf("read gate.go: %v", err)
	}
	src := string(data)
	for _, banned := range []string{
		"newSharedTestRunner",
		"sharedRunner",
		"sharedTest",
	} {
		if strings.Contains(src, banned) {
			t.Errorf("gate.go still wires the shared runner via %q — the feed must be removed (CLM-005)", banned)
		}
	}
}

// TestSharedRunner_NoRenamedWholeModuleGoTestRunner asserts no cmd/backstop or
// pkg/gate non-test source constructs a whole-module `go test ./...` runner under
// ANY name — a renamed shared runner is REQ-002 violated in disguise and FAILS
// (CLM-006).
func TestSharedRunner_NoRenamedWholeModuleGoTestRunner(t *testing.T) {
	gateDir := filepath.Join(repoRoot(t), "pkg", "gate")
	for _, dir := range []string{cmdBackstopDir(t), gateDir} {
		for _, p := range nonTestGoSources(t, dir) {
			b, err := os.ReadFile(p)
			if err != nil {
				t.Fatalf("read %s: %v", p, err)
			}
			src := string(b)
			// A whole-module go-test coverage runner is the literal coupling
			// REQ-002 eradicates: `go test ./...` plus a -coverprofile read. Any
			// surviving construction of that exec is the renamed-runner regression.
			if strings.Contains(src, `"./..."`) && strings.Contains(src, "-coverprofile") {
				t.Errorf("%s constructs a whole-module `go test ./... -coverprofile` runner — a renamed shared runner is forbidden (CLM-006)", p)
			}
		}
	}
}
