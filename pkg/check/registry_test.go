package check

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/config"
)

// unknownKeyRustBackstopYML is a non-go declared stack (language: rust) with an
// out-of-vocabulary enforcement.toolchain key (`typecheck:`) alongside a valid
// `lint:` entry. The typo path is the silent-skip bug ISSUE-008 closes.
const unknownKeyRustBackstopYML = `project: rust-example
language: rust
enforcement:
  toolchain:
    lint:
      command: "cargo clippy --message-format short"
      format: regex-lines
      extensions: [".rs"]
    typecheck:
      command: "cargo check"
      format: regex-lines
      extensions: [".rs"]
`

// unknownKeyGoBackstopYML is a GO-language project carrying a typo'd
// enforcement.toolchain key (`lnit:`). The go early-return must NOT bypass the
// key-vocabulary guard — this is the dominant-path silent-skip hole.
const unknownKeyGoBackstopYML = `project: go-example
language: go
enforcement:
  toolchain:
    lnit:
      command: "golangci-lint run"
      format: regex-lines
`

// semgrepKeyRustBackstopYML declares an in-vocabulary `semgrep:` key alongside a
// valid `lint:` entry for a non-go language. parseCheckType accepts "semgrep",
// so the guard must accept it (accept-semgrep decision) even though it has no
// toolchain-overlay effect today.
const semgrepKeyRustBackstopYML = `project: rust-example
language: rust
enforcement:
  toolchain:
    lint:
      command: "cargo clippy --message-format short"
      format: regex-lines
      extensions: [".rs"]
    semgrep:
      command: "semgrep --config auto"
      format: regex-lines
`

// TestCodeCheck_Registry_SelectsToolchainByLanguage pins CLM-001: the registry
// selects the toolchain by Config.Language; "go" and an absent language bind
// the Go built-in entries (which, post-cutover, construct no pkg/check
// executors), while "typescript" binds the TS built-in entries (distinct commands via the
// generic commandExecutor). Asserts on the constructed executors' identity, not
// live execution.
func TestCodeCheck_Registry_SelectsToolchainByLanguage(t *testing.T) {
	runner := &fakeRunner{}

	// Go stack (explicit language): after the SPEC-034 cutover the Go stack
	// constructs ONLY the shared semgrep executor — build/test/lint run through
	// the go-toolchain pack engines, not pkg/check.
	goExec := buildExecutorsForConfig(Options{Language: "go"}, runner)
	assertGoStackConstructsNoExecutors(t, "language=go", goExec)

	// Absent language defaults to the Go stack (preserves current behavior).
	defaultExec := buildExecutorsForConfig(Options{Language: ""}, runner)
	assertGoStackConstructsNoExecutors(t, "language absent", defaultExec)

	// TypeScript stack: distinct executor types from Go (generic commandExecutor
	// bound to eslint/tsc/declared-test). Semgrep runs through the pack engine,
	// so the registry constructs no semgrep executor.
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
	if _, ok := tsExec[CheckTypeSemgrep]; ok {
		t.Errorf("TS stack must construct no semgrep executor (it runs through the pack engine), got %T", tsExec[CheckTypeSemgrep])
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

// assertGoStackConstructsNoExecutors pins the post-cutover + post-ISSUE-018 Go
// stack shape: the registry constructs NO pkg/check executor for any pass. The
// lint/build/test passes run through the go-toolchain pack engines, and after the
// in-process semgrep path was removed (ISSUE-018) the semgrep pass runs through
// the pack engine too — so the registry's Go executor map is empty.
func assertGoStackConstructsNoExecutors(t *testing.T, ctx string, execs map[CheckType]PassExecutor) {
	t.Helper()
	for _, ct := range []CheckType{CheckTypeLint, CheckTypeBuild, CheckTypeTest, CheckTypeSemgrep} {
		if _, ok := execs[ct]; ok {
			t.Errorf("%s: Go stack must construct no native %v executor (it runs through a pack engine), got %T", ctx, ct, execs[ct])
		}
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

// TestCodeCheck_Registry_UnknownToolchainPassKeyIsConfigError pins CLM-001 /
// REQ-001: an out-of-vocabulary enforcement.toolchain pass key is a fail-loud
// *check.ConfigError naming the offending key AND enumerating the allowed
// vocabulary (lint/build/test/semgrep) — never a silent skip that disables a
// pass. The guard must hold REGARDLESS of language (the go-path early-return
// must not bypass it), and an in-vocabulary `semgrep:` key must be ACCEPTED.
func TestCodeCheck_Registry_UnknownToolchainPassKeyIsConfigError(t *testing.T) {
	runner := &fakeRunner{}

	// --- Non-go language with a typo'd key alongside valid entries. ---
	t.Run("non_go_typo_key_is_config_error", func(t *testing.T) {
		cfg := loadConfigFromYAML(t, unknownKeyRustBackstopYML)

		execs, err := buildExecutorsForConfigErr(Options{Language: cfg.Language, Config: cfg}, runner)
		if err == nil {
			t.Fatalf("buildExecutorsForConfigErr returned nil error for an out-of-vocabulary key; got executors %v (silent skip is the bug)", execs)
		}
		var cfgErr *ConfigError
		if !errors.As(err, &cfgErr) {
			t.Fatalf("error %T (%v) is not a *check.ConfigError", err, err)
		}
		// Must name the offending key so the author can self-correct.
		if !strings.Contains(cfgErr.Message, "typecheck") {
			t.Errorf("message %q must name the offending key %q", cfgErr.Message, "typecheck")
		}
		// Must enumerate the allowed vocabulary.
		for _, allowed := range []string{"lint", "build", "test", "semgrep"} {
			if !strings.Contains(cfgErr.Message, allowed) {
				t.Errorf("message %q must enumerate allowed vocabulary value %q", cfgErr.Message, allowed)
			}
		}
		// It must NOT be a partial executor map silently missing the bad pass.
		if len(execs) != 0 {
			t.Errorf("on a config error the executor map must be empty, got %d entries (partial map masks the silent-skip bug)", len(execs))
		}
	})

	// --- Sharp edge: a GO-language project with a typo'd key ALSO errors. ---
	// This proves the guard runs BEFORE the go/empty-language early-return at
	// registry.go's language branch, closing the dominant-path silent-skip hole.
	t.Run("go_language_typo_key_is_config_error", func(t *testing.T) {
		cfg := loadConfigFromYAML(t, unknownKeyGoBackstopYML)

		execs, err := buildExecutorsForConfigErr(Options{Language: cfg.Language, Config: cfg}, runner)
		if err == nil {
			t.Fatalf("go-language project with a typo'd toolchain key returned nil error; the guard must run before the go early-return (silent non-enforcement on the dominant path)")
		}
		var cfgErr *ConfigError
		if !errors.As(err, &cfgErr) {
			t.Fatalf("go-path error %T (%v) is not a *check.ConfigError", err, err)
		}
		if !strings.Contains(cfgErr.Message, "lnit") {
			t.Errorf("go-path message %q must name the offending key %q", cfgErr.Message, "lnit")
		}
		if len(execs) != 0 {
			t.Errorf("go-path config error must yield an empty executor map, got %d entries", len(execs))
		}
	})

	// --- Accept-semgrep decision: `semgrep:` is in parseCheckType's vocabulary
	// and must NOT be rejected, even though it has no toolchain-overlay effect
	// today (semgrep runs through the pack engine, not a pkg/check executor). Pins
	// the decision so a future tightening that wrongly rejects it fails loud here. ---
	t.Run("semgrep_key_is_accepted", func(t *testing.T) {
		cfg := loadConfigFromYAML(t, semgrepKeyRustBackstopYML)

		execs, err := buildExecutorsForConfigErr(Options{Language: cfg.Language, Config: cfg}, runner)
		if err != nil {
			t.Fatalf("a `semgrep:` toolchain key must be accepted as in-vocabulary, got error: %v", err)
		}
		// The semgrep key has no toolchain-overlay effect: it constructs no
		// pkg/check executor (semgrep runs through the pack engine).
		if _, ok := execs[CheckTypeSemgrep]; ok {
			t.Errorf("semgrep key must construct no pkg/check executor, got %T", execs[CheckTypeSemgrep])
		}
		// The valid lint entry alongside it still builds.
		if _, ok := execs[CheckTypeLint].(*commandExecutor); !ok {
			t.Errorf("lint executor = %T, want *commandExecutor (valid entry must still build)", execs[CheckTypeLint])
		}
	})
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
