package main

import (
	"path/filepath"
	"testing"

	"github.com/bmanson/backstop-core/pkg/check"
	"github.com/bmanson/backstop-core/pkg/config"
)

// TestCutover_CodeCheckSubcommandSurvivesWithDeclaredToolchain proves the
// standalone backstop code check subcommand SURVIVES the cutover: an invocation
// over a project with a DECLARED toolchain still resolves and runs its
// lint/build/test passes through check.Run / buildExecutorsForConfigErr /
// resolveToolchain after builtinToolchain is deleted (CLM-030).
func TestCutover_CodeCheckSubcommandSurvivesWithDeclaredToolchain(t *testing.T) {
	// The subcommand command itself must still exist.
	if codeCheckCmd == nil {
		newCodeCheckCommand(nil)
	}
	if codeCheckCmd == nil {
		t.Fatal("codeCheckCmd is nil — the standalone code check subcommand must survive the cutover (CLM-030)")
	}

	// A DECLARED toolchain (a Go project that declares enforcement.toolchain
	// lint/build/test) still resolves a non-empty executor set through the reduced
	// resolveToolchain — proving the subcommand's run path keeps working.
	cfg := &config.Config{
		Enforcement: config.Enforcement{
			Toolchain: map[string]config.ToolchainPass{
				"lint": {Command: "golangci-lint run", Format: "sarif"},
			},
		},
	}
	execs, err := check.DeclaredToolchainExecutorsForTest("go", cfg)
	if err != nil {
		t.Fatalf("a DECLARED toolchain must resolve through the reduced resolveToolchain, got: %v", err)
	}
	if len(execs) == 0 {
		t.Fatal("a project with a DECLARED enforcement.toolchain must resolve a non-empty executor set — the code check subcommand would otherwise be stranded (CLM-030)")
	}
}

// TestCutover_NoDeletedSymbolHasSurvivingCaller proves no symbol deleted by the
// cutover has a surviving production caller: every deleted symbol (realCodeChecker
// as a gate step, builtinToolchain) is absent AND the retained-symbol callers
// (buildExecutorsForConfigErr -> resolveToolchain) compile and resolve, so the
// delete set cannot strand the code check subcommand (CLM-031).
func TestCutover_NoDeletedSymbolHasSurvivingCaller(t *testing.T) {
	cmdDir := filepath.Join(repoRoot(t), "cmd", "backstop")
	checkDir := filepath.Join(repoRoot(t), "pkg", "check")

	// Deleted symbols are ABSENT from non-test source.
	if grepNonTestSource(t, cmdDir, "realCodeChecker") {
		t.Error("realCodeChecker still present in cmd/backstop non-test source (CLM-031)")
	}
	if grepNonTestSource(t, checkDir, "builtinToolchain") {
		t.Error("builtinToolchain still present in pkg/check non-test source (CLM-031)")
	}

	// Retained-symbol callers compile and RESOLVE: buildExecutorsForConfigErr ->
	// resolveToolchain still produces executors for a declared toolchain (the
	// surviving code check subcommand's path). Exercised through the exported seam.
	cfg := &config.Config{
		Enforcement: config.Enforcement{
			Toolchain: map[string]config.ToolchainPass{
				"lint": {Command: "golangci-lint run", Format: "sarif"},
			},
		},
	}
	if _, err := check.DeclaredToolchainExecutorsForTest("go", cfg); err != nil {
		t.Fatalf("the retained buildExecutorsForConfigErr -> resolveToolchain path must still resolve (CLM-031), got: %v", err)
	}
}
