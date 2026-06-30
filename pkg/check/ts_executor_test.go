package check

import (
	"testing"
)

// TestCodeCheck_TSTestCommand_ExplicitDeclarationRequired pins the post-SPEC-040
// TS posture: the baked TS builtin stack is DELETED, so an undeclared TS project
// no longer carries a built-in test pass that would demand enforcement.test_command
// — it resolves to an EMPTY executor set (the no-toolchain-pack baseline; the
// typescript-toolchain PACK is the enforcement surface). A TS project that
// DECLARES its toolchain still constructs working executors (the surviving code
// check subcommand path).
func TestCodeCheck_TSTestCommand_ExplicitDeclarationRequired(t *testing.T) {
	runner := &fakeRunner{}

	// Post-cutover: typescript with no declared toolchain yields an EMPTY executor
	// set (no baked stack), NOT a config error.
	noCmdCfg := loadConfigFromYAML(t, tsNoTestCommandBackstopYML)
	emptyExecs, err := buildExecutorsForConfigErr(Options{Language: "typescript", Config: noCmdCfg}, runner)
	if err != nil {
		t.Fatalf("an undeclared typescript project must resolve to an empty executor set, not error: %v", err)
	}
	if len(emptyExecs) != 0 {
		t.Fatalf("an undeclared typescript project must construct NO executors after the baked TS stack deletion, got %d", len(emptyExecs))
	}

	// Positive: typescript WITH a declared toolchain.
	cmdCfg := loadConfigFromYAML(t, tsBackstopYML)
	execs, posErr := buildExecutorsForConfigErr(Options{Language: "typescript", Config: cmdCfg}, runner)
	if posErr != nil {
		t.Fatalf("typescript stack with test_command errored: %v", posErr)
	}
	testExec, ok := execs[CheckTypeTest].(*commandExecutor)
	if !ok {
		t.Fatalf("test executor = %T, want *commandExecutor", execs[CheckTypeTest])
	}
	if testExec.command == "" {
		t.Error("test executor command is empty; want the declared test_command")
	}
}
