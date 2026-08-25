package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/packval"
)

const recursiveSandboxProbeEnv = "BACKSTOP_RECURSIVE_SANDBOX_PROBE"

func TestSandboxAuthorization_RecursiveBackstopDoesNotPassivelyInheritExternalMode(t *testing.T) {
	if os.Getenv(recursiveSandboxProbeEnv) == "1" {
		value, present := os.LookupEnv(packval.PackSandboxEnvVar)
		mode, err := resolvePackSandboxMode(false, "", present, value)
		if err != nil {
			if _, writeErr := fmt.Fprint(os.Stdout, err); writeErr != nil {
				os.Exit(3)
			}
			os.Exit(2)
		}
		if _, err := fmt.Fprint(os.Stdout, mode); err != nil {
			os.Exit(3)
		}
		os.Exit(0)
	}

	t.Setenv(packval.PackSandboxEnvVar, "external")
	base := append(packChildEnvironment(), recursiveSandboxProbeEnv+"=1")

	run := func(environment []string) string {
		t.Helper()
		cmd := exec.Command(os.Args[0], "-test.run=^TestSandboxAuthorization_RecursiveBackstopDoesNotPassivelyInheritExternalMode$")
		cmd.Env = environment
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("recursive probe: %v: %s", err, output)
		}
		return strings.TrimSpace(string(output))
	}

	if got := run(base); got != "native" {
		t.Fatalf("recursive invocation passively inherited mode %q, want native", got)
	}
	if got := run(append(base, packval.PackSandboxEnvVar+"=external")); got != "external" {
		t.Fatalf("fresh authorization resolved mode %q, want external", got)
	}
}
