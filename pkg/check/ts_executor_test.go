package check

import (
	"errors"
	"testing"
)

// TestCodeCheck_TSTestCommand_ExplicitDeclarationRequired pins CLM-004: the TS
// test pass requires an explicitly declared enforcement.test_command. A
// typescript stack with no test command must yield a *check.ConfigError
// (errors.As) — exit 2, never a silent skip (constraint 1, no package.json
// detection). The positive case (test_command present) constructs the test
// executor successfully.
func TestCodeCheck_TSTestCommand_ExplicitDeclarationRequired(t *testing.T) {
	runner := &fakeRunner{}

	// Negative: typescript, no enforcement.test_command.
	noCmdCfg := loadConfigFromYAML(t, tsNoTestCommandBackstopYML)
	_, err := buildExecutorsForConfigErr(Options{Language: noCmdCfg.Language, Config: noCmdCfg}, runner)
	if err == nil {
		t.Fatal("typescript stack without test_command returned nil error; want a *check.ConfigError (exit 2), not a silent skip")
	}
	var cfgErr *ConfigError
	if !errors.As(err, &cfgErr) {
		t.Fatalf("error %T (%v) is not a *check.ConfigError", err, err)
	}

	// Positive: typescript WITH enforcement.test_command.
	cmdCfg := loadConfigFromYAML(t, tsBackstopYML)
	execs, posErr := buildExecutorsForConfigErr(Options{Language: cmdCfg.Language, Config: cmdCfg}, runner)
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
