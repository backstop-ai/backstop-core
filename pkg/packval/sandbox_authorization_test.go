package packval

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/pack/engine"
)

func TestSandboxAuthorization_DirectExecPathsUseSanitizedEnvironment(t *testing.T) {
	t.Setenv(PackSandboxEnvVar, "external")
	t.Setenv("BACKSTOP_ENV_SENTINEL", "preserved")
	packDir := t.TempDir()
	script := filepath.Join(packDir, "environment-engine.sh")
	body := `#!/bin/sh
printf '{"version":"2.1.0","runs":[{"tool":{"driver":{"name":"fixture"}},"results":[{"ruleId":"environment","message":{"text":"%s|%s"}}]}]}' "${BACKSTOP_PACK_SANDBOX-unset}" "${BACKSTOP_ENV_SENTINEL-unset}"
`
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name    string
		binding engine.EngineBinding
	}{
		{name: "engine", binding: engine.EngineBinding{Command: script}},
		{name: "producer", binding: engine.EngineBinding{Command: "/bin/false", Producer: filepath.Base(script)}},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := (&DefaultExecutor{}).RunEngine(packDir, test.binding, nil)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(result.Output, `unset|preserved`) {
				t.Fatalf("child environment output = %q", result.Output)
			}
		})
	}

	result, err := (&DefaultExecutor{}).RunScaffoldTest(packDir, ".", `printf '%s|%s' "${BACKSTOP_PACK_SANDBOX-unset}" "${BACKSTOP_ENV_SENTINEL-unset}" > scaffold-environment`)
	if err != nil || !result.Passed {
		t.Fatalf("scaffold result = %#v, err = %v", result, err)
	}
	output, err := os.ReadFile(filepath.Join(packDir, "scaffold-environment"))
	if err != nil {
		t.Fatal(err)
	}
	if string(output) != "unset|preserved" {
		t.Fatalf("scaffold child environment output = %q", output)
	}
}
