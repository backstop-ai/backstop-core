package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ISSUE-018 deletion-assertion tests for the `backstop code check` command
// removal. The absence assertions pin that the vestigial standalone code-check
// command and its in-process engine entry points are GONE from cmd/backstop
// production source so they cannot be silently reintroduced (SPEC-034-style
// deletion guard). They are red while the symbols still exist and go green only
// after TASK-003 lands.

// cmdBackstopProductionSource concatenates every non-test .go file under
// cmd/backstop, so the absence assertions scan production source only.
func cmdBackstopProductionSource(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(repoRoot(t), "cmd", "backstop")
	var b strings.Builder
	for _, p := range nonTestGoSources(t, dir) {
		raw, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("reading %s: %v", p, err)
		}
		b.WriteString("// FILE: " + filepath.Base(p) + "\n")
		b.Write(raw)
		b.WriteString("\n")
	}
	return b.String()
}

// TestCodeCheckCommand_Removed proves the `backstop code check` command surface
// and its in-process check-engine entry points are gone from cmd/backstop
// production source. CLM-001.
func TestCodeCheckCommand_Removed(t *testing.T) {
	src := cmdBackstopProductionSource(t)
	for _, sym := range []string{
		"newCodeCheckCommand",
		"codeCheckCmd",
		"resolveCheckRun",
		"checkRunFn",
	} {
		if strings.Contains(src, sym) {
			t.Errorf("cmd/backstop production source still references removed code-check symbol %q; it must be deleted", sym)
		}
	}
	// The in-process engine entry (check.Run / check.Options construction) must no
	// longer be invoked from cmd/backstop production source.
	if strings.Contains(src, "check.Run") {
		t.Error("cmd/backstop production source still invokes check.Run; the in-process check engine entry must be gone with the command")
	}
	if strings.Contains(src, "check.Options{") {
		t.Error("cmd/backstop production source still constructs check.Options{}; the in-process check engine entry must be gone with the command")
	}
	// The `code` cobra namespace registration must be gone from root.go.
	if strings.Contains(src, "codeCmd") {
		t.Error("root.go still references the codeCmd namespace; the code-check command registration must be removed")
	}
	// The whole command file must be deleted.
	if _, err := os.Stat("code_check.go"); err == nil {
		t.Error("cmd/backstop/code_check.go still exists; the code-check command file must be deleted")
	}
}

// TestCodeCheckSubcommand_AbsentFromCLI proves the CLI surface itself is gone:
// building the root command and resolving `code check` yields an
// unknown-command outcome, so the command is not merely symbol-detached but
// actually unreachable. CLM-001.
func TestCodeCheckSubcommand_AbsentFromCLI(t *testing.T) {
	root := NewRootCommand()
	// cobra Find returns an error (or resolves to the root) when the path does not
	// name a real command. `code check` must not resolve to a runnable command.
	cmd, _, err := root.Find([]string{"code", "check"})
	if err == nil && cmd != nil && cmd.Name() == "check" {
		t.Fatalf("`code check` still resolves to a runnable command %q; the subcommand must be removed from the CLI", cmd.CommandPath())
	}
	// The bare `code` namespace must also be gone (it had no other children).
	codeCmd, _, codeErr := root.Find([]string{"code"})
	if codeErr == nil && codeCmd != nil && codeCmd.Name() == "code" {
		t.Errorf("the `code` namespace still exists on the CLI; with its only child removed the namespace must be gone")
	}
}
