package main

import (
	"os"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/packval"
)

func TestSandboxAuthorization_StrippedFromEveryPackChildEnvironment(t *testing.T) {
	t.Setenv(packval.PackSandboxEnvVar, "external")
	t.Setenv("BACKSTOP_ENV_SENTINEL", "preserved")

	environment := packChildEnvironment()
	for _, entry := range environment {
		if strings.HasPrefix(entry, packval.PackSandboxEnvVar+"=") {
			t.Fatalf("authorization leaked into pack-child environment: %q", entry)
		}
	}
	if !containsEnvironmentEntry(environment, "BACKSTOP_ENV_SENTINEL=preserved") {
		t.Fatal("unrelated environment entry was removed")
	}
	if os.Getenv(packval.PackSandboxEnvVar) != "external" {
		t.Fatal("constructing a child environment mutated the parent")
	}
}

func TestSandboxAuthorization_AllCurrentPackChildConstructionSitesUseSanitizedEnvironment(t *testing.T) {
	gateSource := readFileStr(t, "gate.go")
	if got := strings.Count(gateSource, "Env: packChildEnvironment()"); got < 4 {
		t.Fatalf("gate pack runners with sanitized environments = %d, want at least 4", got)
	}
	if !strings.Contains(gateSource, "signatureCommand.Env = packChildEnvironment()") {
		t.Fatal("contracts signature compiler does not use the sanitized environment")
	}

	for file, snippet := range map[string]string{
		"recipe_apply.go": "&check.ExecCommandRunner{Dir: projectRoot, Env: packChildEnvironment()}",
		"init.go":         "&check.ExecCommandRunner{Dir: options.ProjectRoot, Env: packChildEnvironment()}",
		"doctor.go":       "&check.ExecCommandRunner{Dir: ctx.ProjectRoot, Env: packChildEnvironment()}",
	} {
		if source := readFileStr(t, file); !strings.Contains(source, snippet) {
			t.Errorf("%s does not construct its pack-child runner with the sanitized environment", file)
		}
	}
}

func containsEnvironmentEntry(environment []string, want string) bool {
	for _, entry := range environment {
		if entry == want {
			return true
		}
	}
	return false
}
