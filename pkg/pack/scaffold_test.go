package pack

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- Pack type validation tests (ISSUE-032 CLM-002) ---

func TestPackNew_ValidType_Engine(t *testing.T) {
	for _, typ := range []string{"engine", "mechanism", "toolchain"} {
		if !ValidPackTypes[typ] {
			t.Errorf("expected %q to be a valid pack type", typ)
		}
	}
}

// TestPackNew_RetiredTypesRejected pins the removal of the native-standards
// "rule"/"code" types (CLM-002): they must no longer be valid pack types.
func TestPackNew_RetiredTypesRejected(t *testing.T) {
	for _, typ := range []string{"rule", "code", "standard", "recipe", "bundle", "bogus", ""} {
		if ValidPackTypes[typ] {
			t.Errorf("expected %q to NOT be a valid pack type", typ)
		}
	}
}

// --- Slug validation tests ---

func TestPackNew_ValidSlug_Accepted(t *testing.T) {
	for _, slug := range []string{"error-handling", "go", "my-pack", "ab", "a1", "test-2-things"} {
		if err := ValidateSlug(slug); err != nil {
			t.Errorf("ValidateSlug(%q) returned error: %v", slug, err)
		}
	}
}

func TestPackNew_SlugStartsWithDigit_Exit2(t *testing.T) {
	if err := ValidateSlug("1bad"); err == nil {
		t.Fatal("ValidateSlug(\"1bad\") should error for slug starting with digit")
	}
}

func TestPackNew_SlugTooShort_Exit2(t *testing.T) {
	if err := ValidateSlug("a"); err == nil {
		t.Fatal("ValidateSlug(\"a\") should error for slug shorter than 2 chars")
	}
}

func TestPackNew_SlugTooLong_Exit2(t *testing.T) {
	slug := strings.Repeat("a", 65)
	if err := ValidateSlug(slug); err == nil {
		t.Fatal("ValidateSlug should error for slug longer than 64 chars")
	}
}

func TestPackNew_MissingSlug_Exit2(t *testing.T) {
	if err := ValidateSlug(""); err == nil {
		t.Fatal("ValidateSlug(\"\") should error for missing slug")
	}
}

// --- Engine-pack scaffolding tests (CLM-001) ---

// TestScaffoldPack_WritesEnginePackYml proves ScaffoldPack writes a pack.yml carrying
// name/version/language/archetype + an engines: block with STRING enum values + a
// content.ruleset with a sample engine rule (engine/risk_class/claims with positive+
// negative fixtures) + the referenced validator + fixture files, and NO .standard.md /
// .recipe.md (CLM-001/CLM-003).
func TestScaffoldPack_WritesEnginePackYml(t *testing.T) {
	root := tempProjectDir(t)
	result, err := ScaffoldPack(ScaffoldOptions{Type: "engine", Language: "go", Slug: "error-handling", ProjectRoot: root})
	if err != nil {
		t.Fatalf("ScaffoldPack returned error: %v", err)
	}

	packYml := filepath.Join(root, "error-handling", "pack.yml")
	data, readErr := os.ReadFile(packYml)
	if readErr != nil {
		t.Fatalf("expected pack.yml at %s: %v", packYml, readErr)
	}
	s := string(data)
	for _, want := range []string{
		"name: local/error-handling", "version:", "language: go", "archetype: enforcement",
		"description:", "engines:", "input_mode: none", "scope_kind: file-args", "gate_type: findings",
		"content:", "ruleset:", "engine:", "risk_class:", "claims:", "validator:", "fixtures:", "positive:", "negative:",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("pack.yml missing %q", want)
		}
	}

	// Referenced validator + fixtures exist.
	for _, rel := range []string{
		filepath.Join("validators", "error-handling.sh"),
		filepath.Join("fixtures", "valid", "example.txt"),
		filepath.Join("fixtures", "invalid", "example.txt"),
	} {
		if _, statErr := os.Stat(filepath.Join(root, "error-handling", rel)); statErr != nil {
			t.Errorf("expected scaffolded file %s: %v", rel, statErr)
		}
	}

	// No legacy artifacts anywhere under the project root.
	walkErr := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.HasSuffix(path, ".standard.md") || strings.HasSuffix(path, ".recipe.md") {
			t.Errorf("scaffolder must not write legacy artifact %s", path)
		}
		return nil
	})
	if walkErr != nil {
		t.Fatalf("walking scaffold output: %v", walkErr)
	}

	if result.PackID != "local/error-handling" {
		t.Errorf("PackID = %q, want local/error-handling", result.PackID)
	}
}

// TestScaffoldPack_ParsesUnderConsumer proves the scaffolded pack.yml parses under the
// real consumer parser (pkg/pack.ParseManifest) — the same parser the gate and pack
// add use — for every valid type (CLM-001). The engines: block's string enum values
// must resolve, and the sample rule must satisfy the manifest validations.
func TestScaffoldPack_ParsesUnderConsumer(t *testing.T) {
	for _, typ := range []string{"engine", "mechanism", "toolchain"} {
		root := tempProjectDir(t)
		if _, err := ScaffoldPack(ScaffoldOptions{Type: typ, Language: "go", Slug: "sample-check", ProjectRoot: root}); err != nil {
			t.Fatalf("type %s: ScaffoldPack error: %v", typ, err)
		}
		m, err := ParseManifestFile(filepath.Join(root, "sample-check", "pack.yml"))
		if err != nil {
			t.Fatalf("type %s: consumer ParseManifest failed on scaffolded pack: %v", typ, err)
		}
		if len(m.Content.Ruleset.Rules) != 1 {
			t.Fatalf("type %s: expected 1 sample rule, got %d", typ, len(m.Content.Ruleset.Rules))
		}
		if len(m.Engines) != 1 {
			t.Fatalf("type %s: expected 1 declared engine, got %d", typ, len(m.Engines))
		}
		rule := m.Content.Ruleset.Rules[0]
		if rule.Engine == "" || len(rule.Claims) == 0 {
			t.Errorf("type %s: sample rule must carry an engine and claims, got %+v", typ, rule)
		}
	}
}

// --- Conflict + error tests ---

func TestScaffoldPack_ConflictDirExists(t *testing.T) {
	root := tempProjectDir(t)
	if _, err := ScaffoldPack(ScaffoldOptions{Type: "engine", Language: "go", Slug: "dup", ProjectRoot: root}); err != nil {
		t.Fatalf("first scaffold error: %v", err)
	}
	_, err := ScaffoldPack(ScaffoldOptions{Type: "engine", Language: "go", Slug: "dup", ProjectRoot: root})
	if err == nil {
		t.Fatal("expected error when pack directory already exists")
	}
	if !strings.Contains(err.Error(), filepath.Join(root, "dup")) {
		t.Errorf("conflict error should name the path, got %q", err.Error())
	}
}

func TestScaffoldPack_UnsupportedType(t *testing.T) {
	root := tempProjectDir(t)
	_, err := ScaffoldPack(ScaffoldOptions{Type: "rule", Language: "go", Slug: "x", ProjectRoot: root})
	if err == nil {
		t.Fatal("expected error for retired pack type 'rule'")
	}
	if !strings.Contains(err.Error(), "unsupported pack type") {
		t.Errorf("expected 'unsupported pack type' in error, got %q", err.Error())
	}
}

func TestScaffoldPack_WriteError(t *testing.T) {
	// A file at the pack-dir path blocks directory creation.
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "blocked"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := ScaffoldPack(ScaffoldOptions{Type: "engine", Language: "go", Slug: "blocked", ProjectRoot: root})
	if err == nil {
		t.Fatal("expected error when pack path is blocked by a file")
	}
}

// --- Output tests ---

func TestScaffoldPack_ResultFieldsAndOutput(t *testing.T) {
	root := tempProjectDir(t)
	result, err := ScaffoldPack(ScaffoldOptions{Type: "engine", Language: "go", Slug: "error-handling", ProjectRoot: root})
	if err != nil {
		t.Fatalf("ScaffoldPack error: %v", err)
	}

	if result.Type != "engine" || result.Language != "go" || result.Slug != "error-handling" {
		t.Errorf("unexpected result fields: %+v", result)
	}
	if result.SchemaVersion != "pack-new/v1" {
		t.Errorf("SchemaVersion = %q, want pack-new/v1", result.SchemaVersion)
	}
	if len(result.Paths) == 0 {
		t.Fatal("expected scaffolded paths in result")
	}

	// JSON round-trip carries the documented fields.
	data, jsonErr := json.Marshal(result)
	if jsonErr != nil {
		t.Fatalf("marshaling result: %v", jsonErr)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	for _, field := range []string{"type", "language", "slug", "paths", "schema_version", "pack_id"} {
		if _, ok := m[field]; !ok {
			t.Errorf("JSON output missing field %q", field)
		}
	}

	// Human output references the pack id and a created path.
	human := result.HumanString()
	if !strings.Contains(human, result.PackID) {
		t.Error("human output missing PackID")
	}
	if !strings.Contains(human, result.Paths[0]) {
		t.Error("human output missing created path")
	}
	// ISSUE-049: the output must give the next-step cd hint, because pack check/test read
	// pack.yml from the CURRENT dir — `pack check <path>` after `pack new` otherwise fails
	// confusingly. Assert the hint names the pack dir (slug) and the check command.
	if !strings.Contains(human, "cd "+result.Slug) || !strings.Contains(human, "backstop pack check") {
		t.Errorf("human output missing the `cd %s && backstop pack check` next-step hint (ISSUE-049); got:\n%s", result.Slug, human)
	}
}
