package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/gate"
)

// cmdTerminalFixtureDir returns the dir holding the deprecated broken spec
// fixture used to exercise the cmd-level exclusion notice.
func cmdTerminalFixtureDir(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return filepath.Join(dir, "cmd", "backstop", "testdata", "terminal")
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find repo root (no go.mod)")
		}
		dir = parent
	}
}

// TestGateTerminalExclusion_PrintsCountAndStaysGreen asserts that, for a fixture
// tree whose only spec is a terminal (deprecated) spec with a broken mandated
// test + dead contract:
//
//	(a) the gate emits the informational excluded-count line,
//	(b) the excluded spec produces NO test_verification / contract_signature
//	    failure (its tests/contracts are not extracted),
//	(c) the count line is purely informational (CLM-017).
func TestGateTerminalExclusion_PrintsCountAndStaysGreen(t *testing.T) {
	specDir := cmdTerminalFixtureDir(t)

	// (a) Informational notice is produced and names the excluded count.
	notice := terminalExclusionNotice(specDir)
	if notice == "" {
		t.Fatal("expected an informational excluded-count line for a terminal spec, got empty")
	}
	if !strings.Contains(notice, "1") {
		t.Errorf("notice should report the excluded count (1), got %q", notice)
	}
	if !strings.Contains(strings.ToLower(notice), "retired") && !strings.Contains(strings.ToLower(notice), "excluded") {
		t.Errorf("notice should describe retirement/exclusion, got %q", notice)
	}

	// (b) The deprecated spec's broken mandated test is NOT extracted, so it
	// cannot produce a test_verification failure.
	tests, err := gate.ExtractMandatedTests(specDir)
	if err != nil {
		t.Fatalf("ExtractMandatedTests: %v", err)
	}
	for _, mt := range tests {
		if mt.FuncName == "TestCmd_Deprecated_DoesNotExist_201" {
			t.Error("deprecated spec's mandated test must not be extracted/enforced")
		}
	}

	// (b cont.) The deprecated spec's dead contract is NOT extracted.
	contracts, err := gate.ExtractContractEntries(specDir, t.TempDir())
	if err != nil {
		t.Fatalf("ExtractContractEntries: %v", err)
	}
	for _, c := range contracts {
		if c.Name == "DeletedCmdDeprecatedSymbol" {
			t.Error("deprecated spec's dead contract must not be extracted/enforced")
		}
	}

	// (c) When there are zero terminal specs, the notice is silent (no noise) —
	// proves the line is conditional and does not itself fail anything.
	empty := t.TempDir()
	if got := terminalExclusionNotice(empty); got != "" {
		t.Errorf("expected no notice when no terminal specs present, got %q", got)
	}
}
