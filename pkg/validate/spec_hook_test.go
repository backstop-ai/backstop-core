package validate_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestStandardsHook_OutputsManifestSummary(t *testing.T) {
	script := standardsHookPath(t)
	tmp := t.TempDir()
	rulesDir := filepath.Join(tmp, ".backstop", "rules")
	if err := os.MkdirAll(rulesDir, 0o755); err != nil {
		t.Fatalf("mkdir rules dir: %v", err)
	}

	manifest := `{"standard_id":"STD-GO-001","language":"go","rules":[{"id":"GO-001"},{"id":"GO-002"}]}`
	manifestPath := filepath.Join(rulesDir, "STD-GO-001.manifest.json")
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	cmd := exec.Command("bash", script)
	cmd.Dir = tmp
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("hook failed: %v output=%s", err, string(out))
	}

	s := string(out)
	if !strings.Contains(s, "Available standards:") {
		t.Fatalf("expected output to contain header, got: %q", s)
	}
	if !strings.Contains(s, "STD-GO-001 (go, 2 rules)") {
		t.Fatalf("expected output to contain standard summary, got: %q", s)
	}
}

func TestStandardsHook_MissingDirectoryGraceful(t *testing.T) {
	script := standardsHookPath(t)
	tmp := t.TempDir()
	cmd := exec.Command("bash", script)
	cmd.Dir = tmp
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("hook should exit zero when directory missing, err=%v output=%s", err, string(out))
	}
	if strings.TrimSpace(string(out)) != "" {
		t.Fatalf("expected no output when directory missing, got: %q", string(out))
	}
}

func TestStandardsHook_EmptyDirectoryGraceful(t *testing.T) {
	script := standardsHookPath(t)
	tmp := t.TempDir()
	rulesDir := filepath.Join(tmp, ".backstop", "rules")
	if err := os.MkdirAll(rulesDir, 0o755); err != nil {
		t.Fatalf("mkdir rules dir: %v", err)
	}
	cmd := exec.Command("bash", script)
	cmd.Dir = tmp
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("hook should exit zero when directory empty, err=%v output=%s", err, string(out))
	}
	if strings.TrimSpace(string(out)) != "" {
		t.Fatalf("expected no output when directory empty, got: %q", string(out))
	}
}

func TestSettingsJson_SessionStartHookRegistered(t *testing.T) {
	data, err := os.ReadFile("../../.claude/settings.json")
	if err != nil {
		t.Fatalf("cannot read settings.json: %v", err)
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatalf("cannot parse settings.json: %v", err)
	}

	hooks, ok := settings["hooks"].(map[string]interface{})
	if !ok {
		t.Fatal("settings hooks missing")
	}
	sessionStart, ok := hooks["SessionStart"].([]interface{})
	if !ok {
		t.Fatal("hooks.SessionStart missing or not array")
	}

	if !hasHookCommand(sessionStart, ".claude/hooks/backstop-standards-context.sh") {
		t.Fatal("SessionStart hook missing standards context command")
	}
}

func TestSettingsJson_SubagentStartHookRegistered(t *testing.T) {
	data, err := os.ReadFile("../../.claude/settings.json")
	if err != nil {
		t.Fatalf("cannot read settings.json: %v", err)
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatalf("cannot parse settings.json: %v", err)
	}

	hooks, ok := settings["hooks"].(map[string]interface{})
	if !ok {
		t.Fatal("settings hooks missing")
	}
	subagentStart, ok := hooks["SubagentStart"].([]interface{})
	if !ok {
		t.Fatal("hooks.SubagentStart missing or not array")
	}

	if !hasHookCommand(subagentStart, ".claude/hooks/backstop-standards-context.sh") {
		t.Fatal("SubagentStart hook missing standards context command")
	}
}

func hasHookCommand(hooks []interface{}, expected string) bool {
	for _, h := range hooks {
		entry, ok := h.(map[string]interface{})
		if !ok {
			continue
		}
		command, _ := entry["command"].(string)
		if command == expected {
			return true
		}
	}
	return false
}

func standardsHookPath(t *testing.T) string {
	t.Helper()
	p, err := filepath.Abs("../../.claude/hooks/backstop-standards-context.sh")
	if err != nil {
		t.Fatalf("resolve hook path: %v", err)
	}
	return p
}
