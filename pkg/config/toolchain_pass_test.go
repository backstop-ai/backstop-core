package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bmanson/backstop-core/pkg/config"
)

// writeToolchainTempConfig writes content to a backstop.yml in a fresh temp dir and
// returns the path, so each table case validates against a real on-disk file
// through the same strict-decode + embedded-schema path production uses.
func writeToolchainTempConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "backstop.yml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing temp config: %v", err)
	}
	return path
}

// TestToolchainPass_GateTypeAndWaived_DecodeAndValidate is the lockstep test
// (Sharp Edge 4): a backstop.yml setting enforcement.toolchain gate_type and
// waived MUST decode under KnownFields(true) AND pass embedded-schema
// validation; a config OMITTING both MUST still decode and validate
// (zero-value backward compatibility). These are the DECLARED-join inputs the
// classifier consumes (CLM-001 substantiveness-declared, CLM-002
// coverage-undeclared).
func TestToolchainPass_GateTypeAndWaived_DecodeAndValidate(t *testing.T) {
	tests := []struct {
		name          string
		content       string
		wantGateType  string
		wantWaived    bool
		toolchainPass string // key in enforcement.toolchain to inspect
	}{
		{
			name: "gate_type declared on test pass decodes",
			content: "project: p\nlanguage: go\n" +
				"enforcement:\n  toolchain:\n    test:\n      command: \"go test ./...\"\n" +
				"      format: \"go-test\"\n      gate_type: \"substantiveness\"\n",
			toolchainPass: "test",
			wantGateType:  "substantiveness",
			wantWaived:    false,
		},
		{
			name: "gate_type and waived both set decode",
			content: "project: p\nlanguage: typescript\n" +
				"enforcement:\n  toolchain:\n    test:\n      command: \"npm test\"\n" +
				"      format: \"tap\"\n      gate_type: \"coverage\"\n      waived: true\n",
			toolchainPass: "test",
			wantGateType:  "coverage",
			wantWaived:    true,
		},
		{
			name: "omitting both fields stays valid with zero values",
			content: "project: p\nlanguage: go\n" +
				"enforcement:\n  toolchain:\n    lint:\n      command: \"golangci-lint run\"\n" +
				"      format: \"sarif\"\n",
			toolchainPass: "lint",
			wantGateType:  "",
			wantWaived:    false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := writeToolchainTempConfig(t, tc.content)
			cfg, err := config.LoadConfigFromPath(path)
			if err != nil {
				t.Fatalf("LoadConfigFromPath: strict decode + schema validation failed: %v", err)
			}
			tp, ok := cfg.Enforcement.Toolchain[tc.toolchainPass]
			if !ok {
				t.Fatalf("toolchain pass %q missing from decoded config", tc.toolchainPass)
			}
			if tp.GateType != tc.wantGateType {
				t.Errorf("GateType = %q, want %q", tp.GateType, tc.wantGateType)
			}
			if tp.Waived != tc.wantWaived {
				t.Errorf("Waived = %v, want %v", tp.Waived, tc.wantWaived)
			}
		})
	}
}

// TestToolchainPass_ExistingConfigsWithoutNewFields_StillValid pins backward
// compatibility: a backstop.yml that predates the additive fields (no
// gate_type / waived anywhere) MUST continue to decode and validate, proving
// the additive fields default to the zero value and do not break the strict
// KnownFields(true) decode (CLM-001/CLM-002).
func TestToolchainPass_ExistingConfigsWithoutNewFields_StillValid(t *testing.T) {
	path := filepath.Join("..", "..", "cmd", "backstop", "testdata", "full-backstop.yml")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("fixture %s unavailable: %v", path, err)
	}
	cfg, err := config.LoadConfigFromPath(path)
	if err != nil {
		t.Fatalf("LoadConfigFromPath on a pre-existing config: %v", err)
	}
	for name, tp := range cfg.Enforcement.Toolchain {
		if tp.GateType != "" {
			t.Errorf("toolchain[%q].GateType = %q, want empty for a legacy config", name, tp.GateType)
		}
		if tp.Waived {
			t.Errorf("toolchain[%q].Waived = true, want false for a legacy config", name)
		}
	}
}
