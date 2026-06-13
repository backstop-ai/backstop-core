package check

import (
	"context"
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/config"
)

// TestCodeCheck_Registry_SelectsToolchainByLanguage pins CLM-001: the registry
// selects the toolchain by Config.Language; "go" and an absent language bind
// the Go built-in entries (the bespoke lint/build/test/semgrep executors),
// while "typescript" binds the TS built-in entries (distinct commands via the
// generic commandExecutor). Asserts on the constructed executors' identity, not
// live execution.
func TestCodeCheck_Registry_SelectsToolchainByLanguage(t *testing.T) {
	runner := &fakeRunner{}

	// Go stack (explicit language).
	goExec := buildExecutorsForConfig(Options{Language: "go"}, runner)
	assertGoBuiltinExecutors(t, "language=go", goExec)

	// Absent language defaults to the Go stack (preserves current behavior).
	defaultExec := buildExecutorsForConfig(Options{Language: ""}, runner)
	assertGoBuiltinExecutors(t, "language absent", defaultExec)

	// TypeScript stack: distinct executor types from Go (generic commandExecutor
	// bound to eslint/tsc/declared-test) plus the shared semgrep executor.
	tsExec := buildExecutorsForConfig(Options{
		Language: "typescript",
		Config: &config.Config{
			Language: "typescript",
			Enforcement: config.Enforcement{
				TestCommand: "vitest run",
			},
		},
	}, runner)
	if _, ok := tsExec[CheckTypeLint].(*commandExecutor); !ok {
		t.Errorf("TS lint executor type = %T, want *commandExecutor", tsExec[CheckTypeLint])
	}
	if _, ok := tsExec[CheckTypeBuild].(*commandExecutor); !ok {
		t.Errorf("TS build executor type = %T, want *commandExecutor", tsExec[CheckTypeBuild])
	}
	if _, ok := tsExec[CheckTypeTest].(*commandExecutor); !ok {
		t.Errorf("TS test executor type = %T, want *commandExecutor", tsExec[CheckTypeTest])
	}
	if _, ok := tsExec[CheckTypeSemgrep].(*semgrepExecutor); !ok {
		t.Errorf("TS semgrep executor type = %T, want shared *semgrepExecutor", tsExec[CheckTypeSemgrep])
	}

	// The TS lint command must be eslint-derived, distinct from golangci-lint.
	tsLint, _ := tsExec[CheckTypeLint].(*commandExecutor)
	if tsLint != nil && !strings.HasPrefix(tsLint.command, "eslint") {
		t.Errorf("TS lint command = %q, want an eslint command", tsLint.command)
	}
	tsBuild, _ := tsExec[CheckTypeBuild].(*commandExecutor)
	if tsBuild != nil && !strings.Contains(tsBuild.command, "tsc") {
		t.Errorf("TS build command = %q, want a tsc command", tsBuild.command)
	}
}

func assertGoBuiltinExecutors(t *testing.T, ctx string, execs map[CheckType]PassExecutor) {
	t.Helper()
	if _, ok := execs[CheckTypeLint].(*lintExecutor); !ok {
		t.Errorf("%s: lint executor = %T, want *lintExecutor (Go built-in)", ctx, execs[CheckTypeLint])
	}
	if _, ok := execs[CheckTypeBuild].(*buildExecutor); !ok {
		t.Errorf("%s: build executor = %T, want *buildExecutor (Go built-in)", ctx, execs[CheckTypeBuild])
	}
	if _, ok := execs[CheckTypeTest].(*testExecutor); !ok {
		t.Errorf("%s: test executor = %T, want *testExecutor (Go built-in)", ctx, execs[CheckTypeTest])
	}
	if _, ok := execs[CheckTypeSemgrep].(*semgrepExecutor); !ok {
		t.Errorf("%s: semgrep executor = %T, want *semgrepExecutor", ctx, execs[CheckTypeSemgrep])
	}
}

// TestCodeCheck_Registry_CustomToolchainFromConfig pins CLM-002: a custom stack
// declared in backstop.yml (command + format per pass) produces working
// executors with no code changes. Each declared pass is driven through a
// fakeRunner returning fixture output and the declared format parses it into
// violations; the declared command string is the one passed to the runner.
func TestCodeCheck_Registry_CustomToolchainFromConfig(t *testing.T) {
	cfg := loadConfigFromYAML(t, declaredStackBackstopYML)

	// regex-lines fixture output the declared format must parse.
	runner := &fakeRunner{outputs: map[string][]byte{
		"cargo": []byte(regexLinesSampleTxt),
	}}

	execs := buildExecutorsForConfig(Options{Language: cfg.Language, Config: cfg}, runner)

	for _, ct := range []CheckType{CheckTypeLint, CheckTypeBuild, CheckTypeTest} {
		ce, ok := execs[ct].(*commandExecutor)
		if !ok {
			t.Fatalf("declared %v executor = %T, want *commandExecutor", ct, execs[ct])
		}
		// The declared command begins with "cargo".
		if !strings.HasPrefix(ce.command, "cargo") {
			t.Errorf("declared %v command = %q, want a cargo command", ct, ce.command)
		}

		pr, err := ce.Execute(context.Background(), []string{"src/lib.rs"})
		if err != nil {
			t.Fatalf("declared %v Execute: %v", ct, err)
		}
		if len(pr.Violations) != 2 {
			t.Errorf("declared %v produced %d violations, want 2 (regex-lines fixture)", ct, len(pr.Violations))
		}
		if len(pr.Violations) > 0 && pr.Violations[0].Pass != ct {
			t.Errorf("declared %v violation Pass = %v, want %v", ct, pr.Violations[0].Pass, ct)
		}
	}

	// The runner was actually invoked with the "cargo" command.
	sawCargo := false
	for _, c := range runner.calls {
		if c.name == "cargo" {
			sawCargo = true
		}
	}
	if !sawCargo {
		t.Error("fakeRunner never received the declared 'cargo' command")
	}
}

// loadConfigFromYAML writes the given YAML to a temp backstop.yml and loads it
// through the real config loader, exercising the actual decode path. The config
// fixtures live as Go literals because the implementer cannot write testdata/
// files in this environment (see fixtures_toolchain_test.go).
func loadConfigFromYAML(t *testing.T, yaml string) *config.Config {
	t.Helper()
	dir := t.TempDir()
	path := dir + "/backstop.yml"
	if err := writeFileForTest(path, yaml); err != nil {
		t.Fatalf("write temp backstop.yml: %v", err)
	}
	cfg, err := config.LoadConfigFromPath(path)
	if err != nil {
		t.Fatalf("LoadConfigFromPath: %v", err)
	}
	return cfg
}
