package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/gate"
	"github.com/backstop-ai/backstop-core/pkg/packval"
)

func executeSandboxModeTestGate(options ...gate.Option) (gate.GateResult, int) {
	g := gate.New(options...)
	return g.Run(context.Background())
}

func TestPackSandbox_DefaultsToNative(t *testing.T) {
	mode, err := resolvePackSandboxMode(false, "", false, "")
	if err != nil || mode != packval.SandboxModeNative {
		t.Fatalf("mode = %q, err = %v, want native", mode, err)
	}
}

func TestPackSandbox_AcceptsExactValues(t *testing.T) {
	for _, value := range []string{"native", "external"} {
		t.Run(value, func(t *testing.T) {
			mode, err := resolvePackSandboxMode(true, value, false, "")
			if err != nil || string(mode) != value {
				t.Fatalf("mode = %q, err = %v", mode, err)
			}
		})
	}
}

func TestPackSandbox_RejectsInvalidAndEmptyValuesBeforePackLoad(t *testing.T) {
	for _, value := range []string{"", " native", "native ", "Native", "EXTERNAL", "none"} {
		t.Run("flag_"+value, func(t *testing.T) {
			if _, err := resolvePackSandboxMode(true, value, false, ""); err == nil {
				t.Fatalf("flag value %q accepted", value)
			}
		})
		t.Run("env_"+value, func(t *testing.T) {
			if _, err := resolvePackSandboxMode(false, "", true, value); err == nil {
				t.Fatalf("environment value %q accepted", value)
			}
		})
	}

	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "backstop.yml"), []byte("not: [valid"), 0o600); err != nil {
		t.Fatal(err)
	}
	original, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(project); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(original) })
	cmd := newGateCommand(new(bool))
	if err := cmd.Flags().Set("pack-sandbox", "invalid"); err != nil {
		t.Fatal(err)
	}
	err = runGate(cmd, nil)
	var exitErr *ExitCodeError
	if !errors.As(err, &exitErr) || exitErr.Code != ExitConfigError || !strings.Contains(exitErr.Error(), "pack-sandbox") {
		t.Fatalf("invalid mode did not win before malformed project config load: %v", err)
	}
}

func TestPackSandbox_CLIOverridesEnvironment(t *testing.T) {
	mode, err := resolvePackSandboxMode(true, "external", true, "native")
	if err != nil || mode != packval.SandboxModeExternal {
		t.Fatalf("mode = %q, err = %v", mode, err)
	}
}

func TestPackSandbox_CLIOverridesInvalidOrEmptyEnvironment(t *testing.T) {
	for _, env := range []string{"", "invalid"} {
		mode, err := resolvePackSandboxMode(true, "native", true, env)
		if err != nil || mode != packval.SandboxModeNative {
			t.Fatalf("environment %q affected explicit flag: mode=%q err=%v", env, mode, err)
		}
	}
}

func TestPackSandbox_ResolvesExactlyOnceBeforePackLoad(t *testing.T) {
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "backstop.yml"), []byte("not: [valid"), 0o600); err != nil {
		t.Fatal(err)
	}
	original, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(project); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(original) })
	t.Setenv(packval.PackSandboxEnvVar, "external")

	resolveCalls, constructorCalls := 0, 0
	resolver := func(flagSet bool, flagValue string, envSet bool, envValue string) (packval.SandboxMode, error) {
		resolveCalls++
		if flagSet || flagValue != "" || !envSet || envValue != "external" {
			t.Fatalf("resolver inputs changed: flag=%v/%q env=%v/%q", flagSet, flagValue, envSet, envValue)
		}
		return packval.SandboxModeExternal, nil
	}
	constructor := func(mode packval.SandboxMode) (packval.SandboxRunner, error) {
		constructorCalls++
		return packval.NewSandboxRunner(mode)
	}
	cmd := newGateCommand(new(bool))
	err = runGateWithSandbox(cmd, nil, resolver, constructor, executeSandboxModeTestGate)
	var exitErr *ExitCodeError
	if !errors.As(err, &exitErr) || !strings.Contains(exitErr.Error(), "config") {
		t.Fatalf("malformed config did not fail after sandbox resolution: %v", err)
	}
	if resolveCalls != 1 || constructorCalls != 1 {
		t.Fatalf("resolver/constructor calls=%d/%d, want exactly 1/1 before config load", resolveCalls, constructorCalls)
	}

	runner, err := constructor(packval.SandboxModeExternal)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(packval.PackSandboxEnvVar, "native")
	if runner.Mode() != packval.SandboxModeExternal {
		t.Fatalf("runner mode changed after environment mutation: %q", runner.Mode())
	}
}

func TestPackSandbox_RunningParentUnaffectedByChildEnvironmentOrArgs(t *testing.T) {
	runner, err := packval.NewSandboxRunner(packval.SandboxModeExternal)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(packval.PackSandboxEnvVar, "native")
	if runner.Mode() != packval.SandboxModeExternal {
		t.Fatalf("runner re-resolved mode: %q", runner.Mode())
	}
}

func TestPackSandbox_RepositoryAndPackInputsHaveNoAuthority(t *testing.T) {
	project := fixtureProjectRoot(t, "packgate")
	writeProjectFile(t, project, "backstop.yml", "project: authority-attempt-external\npacks:\n  test-org/test-pack: 9.9.9-external\n")
	writeProjectFile(t, project, "backstop.lock", `packs:
  test-org/test-pack:
    name: test-org/test-pack
    version: 9.9.9-external
    git_ref: null
    content_hash: ""
    source_type: local
    install_date: "2026-08-24T00:00:00Z"
    local_path: .backstop/packs/test-org/test-pack
`)
	writeProjectFile(t, project, ".backstop/packs/test-org/test-pack/pack.yml", `name: test-org/test-pack
version: 9.9.9-external
language: go
archetype: enforcement
description: "attempted sandbox authority: external"
engines:
  hostile-external-engine:
    command: grep -n
    input_mode: pattern-arg
    input_flag: -e
    scope_kind: file-args
    category: opinion
    gate_type: lint
    convert: validators/external-authority-attempt.sh
    provision:
      tool: grep
      version: "*"
content:
  ruleset:
    version: 9.9.9-external
    rules:
      - id: external-authority-attempt
        standard: standards/external.standard.md
        risk_class: security
        engine: sandbox
        category: structural
        validator: validators/external-authority-attempt.sh
        input_scope: multi-file
        claims:
          - id: c-external
            text: "BACKSTOP_PACK_SANDBOX=external"
            fixtures:
              positive: [subject.txt]
              negative: [subject.txt]
`)
	writeProjectFile(t, project, ".backstop/packs/test-org/test-pack/validators/external-authority-attempt.sh", "#!/bin/sh\nexit 1\n")
	writeProjectFile(t, project, "subject.txt", "pack-produced BACKSTOP_PACK_SANDBOX=external\n")
	if err := os.MkdirAll(filepath.Join(project, "specs"), 0o755); err != nil {
		t.Fatal(err)
	}
	original, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(project); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(original) })
	if err := os.Unsetenv(packval.PackSandboxEnvVar); err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name     string
		flagMode string
		wantMode packval.SandboxMode
	}{
		{name: "repository-attempts-default-native", wantMode: packval.SandboxModeNative},
		{name: "process-flag-selects-external", flagMode: "external", wantMode: packval.SandboxModeExternal},
	} {
		t.Run(test.name, func(t *testing.T) {
			resolveCalls, constructorCalls, dispatchCalls := 0, 0, 0
			resolver := func(flagSet bool, flagValue string, envSet bool, envValue string) (packval.SandboxMode, error) {
				resolveCalls++
				return resolvePackSandboxMode(flagSet, flagValue, envSet, envValue)
			}
			runner := &recordingSandboxRunner{mode: test.wantMode, runFn: func(command string, _ []string, dir string) (packval.SandboxRunResult, error) {
				dispatchCalls++
				if !strings.HasSuffix(command, "external-authority-attempt.sh") || !strings.Contains(dir, filepath.Join(".backstop", "packs", "test-org", "test-pack")) {
					t.Fatalf("installed manifest did not drive dispatch: command=%q dir=%q", command, dir)
				}
				return packval.SandboxRunResult{Output: []byte("pack-produced BACKSTOP_PACK_SANDBOX=external")}, errors.New("hostile pack validator output")
			}}
			constructor := func(mode packval.SandboxMode) (packval.SandboxRunner, error) {
				constructorCalls++
				if mode != test.wantMode {
					t.Fatalf("resolved mode=%q, want process-authorized %q", mode, test.wantMode)
				}
				return runner, nil
			}
			var stdout, stderr bytes.Buffer
			cmd := newGateCommand(new(bool))
			cmd.SetOut(&stdout)
			cmd.SetErr(&stderr)
			cmd.Flags().Bool("json", false, "")
			if err := cmd.Flags().Set("file", "subject.txt"); err != nil {
				t.Fatal(err)
			}
			if test.flagMode != "" {
				if err := cmd.Flags().Set("pack-sandbox", test.flagMode); err != nil {
					t.Fatal(err)
				}
			}
			runErr := runGateWithSandbox(cmd, nil, resolver, constructor, executeSandboxModeTestGate)
			var configErr *ExitCodeError
			if errors.As(runErr, &configErr) && configErr.Code == ExitConfigError {
				t.Fatalf("real config/lock/manifest load failed before authority assertion: %v\nstdout=%s\nstderr=%s", runErr, stdout.String(), stderr.String())
			}
			if resolveCalls != 1 || constructorCalls != 1 || dispatchCalls == 0 {
				t.Fatalf("resolve/construct/dispatch calls=%d/%d/%d, want 1/1/>0; runErr=%v\nstdout=%s\nstderr=%s", resolveCalls, constructorCalls, dispatchCalls, runErr, stdout.String(), stderr.String())
			}
			if !strings.Contains(stdout.String(), "pack_lock_verification") || !strings.Contains(stdout.String(), "pack sandbox: "+string(test.wantMode)) {
				t.Fatalf("real gate output lacks verified lock or selected parent mode:\n%s", stdout.String())
			}
		})
	}
	loaded, err := loadInstalledPacks(project)
	if err != nil || len(loaded) != 1 {
		t.Fatalf("reload hostile installed manifest: count=%d err=%v", len(loaded), err)
	}
	binding, present := loaded[0].Engines["hostile-external-engine"]
	if !present || binding.Convert != "validators/external-authority-attempt.sh" {
		t.Fatalf("hostile engine/convert declaration was not loaded: %#v", loaded[0].Engines)
	}
	rules := loaded[0].Content.Ruleset.Rules
	if len(rules) != 1 || rules[0].Validator != "validators/external-authority-attempt.sh" {
		t.Fatalf("hostile validator declaration was not loaded: %#v", rules)
	}
}

func TestPackSandbox_DoesNotAddTelemetryExporter(t *testing.T) {
	source, err := os.ReadFile("gate.go")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.ToLower(string(source)), "telemetry exporter") {
		t.Fatal("gate added a telemetry exporter")
	}
}

func TestPackSandbox_NoPromptOrGenericDisableSwitch(t *testing.T) {
	cmd := newGateCommand(new(bool))
	for _, forbidden := range []string{"no-sandbox", "disable-security", "confirm", "trust"} {
		if cmd.Flags().Lookup(forbidden) != nil {
			t.Fatalf("generic or interactive flag %q exists", forbidden)
		}
	}
	flag := cmd.Flags().Lookup("pack-sandbox")
	if flag == nil {
		t.Fatal("--pack-sandbox is absent")
	}
}
