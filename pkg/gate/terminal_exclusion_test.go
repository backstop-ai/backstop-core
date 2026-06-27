package gate

import (
	"os"
	"path/filepath"
	"testing"
)

// gateRepoRoot walks up from cwd to the directory containing go.mod.
func gateRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find repo root (no go.mod)")
		}
		dir = parent
	}
}

// terminalFixtureDir returns the dir holding the active/deprecated/replaced
// gate-exclusion spec fixtures.
func terminalFixtureDir(t *testing.T) string {
	t.Helper()
	return filepath.Join(gateRepoRoot(t), "pkg", "gate", "testdata", "terminal")
}

// TestExtractMandatedTests_SkipsTerminalSpecs proves the mandated-test
// extraction returns the active spec's mandated test but NOT the deprecated or
// replaced specs' tests (CLM-015).
func TestExtractMandatedTests_SkipsTerminalSpecs(t *testing.T) {
	dir := terminalFixtureDir(t)
	tests, err := ExtractMandatedTests(dir)
	if err != nil {
		t.Fatalf("ExtractMandatedTests: %v", err)
	}

	names := map[string]bool{}
	for _, mt := range tests {
		names[mt.FuncName] = true
	}

	if !names["TestActive_DoesNotExist_100"] {
		t.Error("active spec's mandated test must be extracted")
	}
	if names["TestDeprecated_DoesNotExist_101"] {
		t.Error("deprecated spec's mandated test must be SKIPPED")
	}
	if names["TestReplaced_DoesNotExist_102"] {
		t.Error("replaced spec's mandated test must be SKIPPED")
	}
}

// TestExtractContractEntries_SkipsTerminalSpecs proves contract extraction
// returns the active spec's contract but NOT the terminal specs' dead contracts
// (CLM-016).
func TestExtractContractEntries_SkipsTerminalSpecs(t *testing.T) {
	dir := terminalFixtureDir(t)
	contracts, err := ExtractContractEntries(dir, "/project/root")
	if err != nil {
		t.Fatalf("ExtractContractEntries: %v", err)
	}

	names := map[string]bool{}
	for _, c := range contracts {
		names[c.Name] = true
	}

	if !names["DeletedActiveSymbol"] {
		t.Error("active spec's contract must be extracted")
	}
	if names["DeletedDeprecatedSymbol"] {
		t.Error("deprecated spec's dead contract must be SKIPPED")
	}
	if names["DeletedReplacedSymbol"] {
		t.Error("replaced spec's dead contract must be SKIPPED")
	}
}

// TestExtractSpecVerifications_SkipsTerminalSpecs proves the verification
// extraction skips terminal specs (CLM-016).
func TestExtractSpecVerifications_SkipsTerminalSpecs(t *testing.T) {
	dir := terminalFixtureDir(t)
	specs, err := ExtractSpecVerifications(dir)
	if err != nil {
		t.Fatalf("ExtractSpecVerifications: %v", err)
	}

	ids := map[string]bool{}
	for _, s := range specs {
		ids[s.SpecID] = true
	}

	if !ids["TEST-100"] {
		t.Error("active spec's verification must be extracted")
	}
	if ids["TEST-101"] {
		t.Error("deprecated spec's verification must be SKIPPED")
	}
	if ids["TEST-102"] {
		t.Error("replaced spec's verification must be SKIPPED")
	}
}

// TestGateExclusion_DeprecatedSpecMandatedTestsNotEnforced proves a deprecated
// spec's missing mandated test does NOT produce a test_verification failure
// (CLM-015), while the active control spec's missing test DOES.
func TestGateExclusion_DeprecatedSpecMandatedTestsNotEnforced(t *testing.T) {
	dir := terminalFixtureDir(t)
	tests, err := ExtractMandatedTests(dir)
	if err != nil {
		t.Fatalf("ExtractMandatedTests: %v", err)
	}
	for _, mt := range tests {
		if mt.SpecID == "TEST-101" {
			t.Fatalf("deprecated spec TEST-101 must not contribute mandated tests, got %s", mt.FuncName)
		}
	}
	// Active control still present → still enforced.
	foundActive := false
	for _, mt := range tests {
		if mt.SpecID == "TEST-100" {
			foundActive = true
		}
	}
	if !foundActive {
		t.Error("active control spec TEST-100 must still be enforced")
	}
}

// TestGateExclusion_ReplacedSpecContractsNotEnforced proves a replaced spec's
// dead contract is NOT enforced (CLM-016).
func TestGateExclusion_ReplacedSpecContractsNotEnforced(t *testing.T) {
	dir := terminalFixtureDir(t)
	contracts, err := ExtractContractEntries(dir, "/project/root")
	if err != nil {
		t.Fatalf("ExtractContractEntries: %v", err)
	}
	for _, c := range contracts {
		if c.Name == "DeletedReplacedSymbol" {
			t.Fatal("replaced spec's dead contract must not be enforced")
		}
	}
}

// TestGateExclusion_ReportsExcludedCount proves the extraction surfaces a count
// of excluded terminal specs equal to the number of terminal fixtures (CLM-017).
func TestGateExclusion_ReportsExcludedCount(t *testing.T) {
	dir := terminalFixtureDir(t)
	count, err := CountTerminalSpecs(dir)
	if err != nil {
		t.Fatalf("CountTerminalSpecs: %v", err)
	}
	// Two terminal fixtures: deprecated + replaced.
	if count != 2 {
		t.Errorf("expected 2 excluded terminal specs, got %d", count)
	}
}
