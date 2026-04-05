package pack

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- Pack type validation tests (REQ-001) ---

func TestPackNew_ValidType_Rule(t *testing.T) {
	if !ValidPackTypes["rule"] {
		t.Fatal("expected 'rule' to be a valid pack type")
	}
}

func TestPackNew_ValidType_Code(t *testing.T) {
	if !ValidPackTypes["code"] {
		t.Fatal("expected 'code' to be a valid pack type")
	}
}

func TestPackNew_InvalidType_Exit2(t *testing.T) {
	for _, typ := range []string{"bundle", "recipe", "standard", "bogus"} {
		if ValidPackTypes[typ] {
			t.Errorf("expected %q to NOT be a valid pack type", typ)
		}
	}
}

func TestPackNew_MissingType_Exit2(t *testing.T) {
	if ValidPackTypes[""] {
		t.Fatal("expected empty string to NOT be a valid pack type")
	}
}

// --- Slug validation tests (REQ-003) ---

func TestPackNew_ValidSlug_Accepted(t *testing.T) {
	for _, slug := range []string{"error-handling", "go", "my-pack", "ab", "a1", "test-2-things"} {
		if err := ValidateSlug(slug); err != nil {
			t.Errorf("ValidateSlug(%q) returned error: %v", slug, err)
		}
	}
}

func TestPackNew_SlugStartsWithDigit_Exit2(t *testing.T) {
	err := ValidateSlug("1bad")
	if err == nil {
		t.Fatal("ValidateSlug(\"1bad\") should return an error for slug starting with digit")
	}
}

func TestPackNew_SlugTooShort_Exit2(t *testing.T) {
	err := ValidateSlug("a")
	if err == nil {
		t.Fatal("ValidateSlug(\"a\") should return an error for slug shorter than 2 chars")
	}
}

func TestPackNew_SlugTooLong_Exit2(t *testing.T) {
	// 65 characters — exceeds 64 max
	slug := "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklm"
	if len(slug) != 65 {
		t.Fatalf("test slug should be 65 chars, got %d", len(slug))
	}
	err := ValidateSlug(slug)
	if err == nil {
		t.Fatal("ValidateSlug should return an error for slug longer than 64 chars")
	}
}

func TestPackNew_MissingSlug_Exit2(t *testing.T) {
	err := ValidateSlug("")
	if err == nil {
		t.Fatal("ValidateSlug(\"\") should return an error for missing slug")
	}
}

// --- Rule pack scaffolding tests (REQ-004) ---

func TestPackNew_RulePack_CreatesStandardFile(t *testing.T) {
	root := tempProjectDir(t)
	result, err := ScaffoldPack(ScaffoldOptions{
		Type:        "rule",
		Language:    "go",
		Slug:        "error-handling",
		Number:      1,
		ProjectRoot: root,
	})
	if err != nil {
		t.Fatalf("ScaffoldPack returned error: %v", err)
	}
	expectedPath := filepath.Join(root, "standards", "go", "STD-GO-001-error-handling.standard.md")
	if _, err := os.Stat(expectedPath); err != nil {
		t.Fatalf("expected standard file at %s, got error: %v", expectedPath, err)
	}
	if len(result.Paths) == 0 || result.Paths[0] != expectedPath {
		t.Fatalf("expected result.Paths[0] = %s, got %v", expectedPath, result.Paths)
	}
}

func TestPackNew_RulePack_UppercasedLangPrefix(t *testing.T) {
	root := tempProjectDir(t)
	_, err := ScaffoldPack(ScaffoldOptions{
		Type:        "rule",
		Language:    "typescript",
		Slug:        "security",
		Number:      1,
		ProjectRoot: root,
	})
	if err != nil {
		t.Fatalf("ScaffoldPack returned error: %v", err)
	}
	expectedPath := filepath.Join(root, "standards", "typescript", "STD-TYPESCRIPT-001-security.standard.md")
	if _, err := os.Stat(expectedPath); err != nil {
		t.Fatalf("expected file with uppercased lang prefix at %s, got error: %v", expectedPath, err)
	}
}

func TestPackNew_RulePack_FrontmatterFields(t *testing.T) {
	root := tempProjectDir(t)
	_, err := ScaffoldPack(ScaffoldOptions{
		Type:        "rule",
		Language:    "go",
		Slug:        "error-handling",
		Number:      1,
		ProjectRoot: root,
	})
	if err != nil {
		t.Fatalf("ScaffoldPack returned error: %v", err)
	}
	filePath := filepath.Join(root, "standards", "go", "STD-GO-001-error-handling.standard.md")
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("reading standard file: %v", err)
	}
	s := string(content)
	for _, field := range []string{"title:", "number:", "created:", "status:", "schema_version:", "language:", "pack:", "scope:"} {
		if !strings.Contains(s, field) {
			t.Errorf("frontmatter missing field %q", field)
		}
	}
}

func TestPackNew_RulePack_FrontmatterDefaults(t *testing.T) {
	root := tempProjectDir(t)
	_, err := ScaffoldPack(ScaffoldOptions{
		Type:        "rule",
		Language:    "go",
		Slug:        "error-handling",
		Number:      1,
		ProjectRoot: root,
	})
	if err != nil {
		t.Fatalf("ScaffoldPack returned error: %v", err)
	}
	filePath := filepath.Join(root, "standards", "go", "STD-GO-001-error-handling.standard.md")
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("reading standard file: %v", err)
	}
	s := string(content)
	if !strings.Contains(s, "status: active") {
		t.Error("expected status: active in frontmatter")
	}
	if !strings.Contains(s, "created:") {
		t.Error("expected created field in frontmatter")
	}
}

func TestPackNew_RulePack_TemplateRuleBody(t *testing.T) {
	root := tempProjectDir(t)
	_, err := ScaffoldPack(ScaffoldOptions{
		Type:        "rule",
		Language:    "go",
		Slug:        "error-handling",
		Number:      1,
		ProjectRoot: root,
	})
	if err != nil {
		t.Fatalf("ScaffoldPack returned error: %v", err)
	}
	filePath := filepath.Join(root, "standards", "go", "STD-GO-001-error-handling.standard.md")
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("reading standard file: %v", err)
	}
	s := string(content)
	// Must contain rules array with template rule fields
	for _, field := range []string{"rules:", "id:", "name:", "category:", "severity:", "description:", "compliance_tier:", "detection:"} {
		if !strings.Contains(s, field) {
			t.Errorf("template rule body missing field %q", field)
		}
	}
}

func TestPackNew_RulePack_CreatesDirectoryIfMissing(t *testing.T) {
	// Use a temp dir WITHOUT pre-created standards/go/
	root := t.TempDir()
	_, err := ScaffoldPack(ScaffoldOptions{
		Type:        "rule",
		Language:    "go",
		Slug:        "error-handling",
		Number:      1,
		ProjectRoot: root,
	})
	if err != nil {
		t.Fatalf("ScaffoldPack returned error: %v", err)
	}
	dir := filepath.Join(root, "standards", "go")
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("expected standards/go/ to be created, got error: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("expected standards/go/ to be a directory")
	}
}

// --- Code pack scaffolding tests (REQ-005) ---

func TestPackNew_CodePack_CreatesRecipeDirectory(t *testing.T) {
	root := tempProjectDir(t)
	result, err := ScaffoldPack(ScaffoldOptions{
		Type:        "code",
		Language:    "go",
		Slug:        "error-handling",
		ProjectRoot: root,
	})
	if err != nil {
		t.Fatalf("ScaffoldPack returned error: %v", err)
	}
	dir := filepath.Join(root, "recipes", "go", "error-handling")
	info, statErr := os.Stat(dir)
	if statErr != nil {
		t.Fatalf("expected recipe dir at %s, got error: %v", dir, statErr)
	}
	if !info.IsDir() {
		t.Fatal("expected recipe path to be a directory")
	}
	if len(result.Paths) != 2 {
		t.Fatalf("expected 2 paths in result, got %d", len(result.Paths))
	}
}

func TestPackNew_CodePack_ReadmeContents(t *testing.T) {
	root := tempProjectDir(t)
	_, err := ScaffoldPack(ScaffoldOptions{
		Type:        "code",
		Language:    "go",
		Slug:        "error-handling",
		ProjectRoot: root,
	})
	if err != nil {
		t.Fatalf("ScaffoldPack returned error: %v", err)
	}
	readmePath := filepath.Join(root, "recipes", "go", "error-handling", "README.md")
	content, readErr := os.ReadFile(readmePath)
	if readErr != nil {
		t.Fatalf("reading README.md: %v", readErr)
	}
	s := string(content)
	if !strings.Contains(s, "Error Handling") {
		t.Error("README.md should contain pack name derived from slug")
	}
	if !strings.Contains(s, "Description placeholder") {
		t.Error("README.md should contain description placeholder")
	}
}

func TestPackNew_CodePack_TemplateRecipeFile(t *testing.T) {
	root := tempProjectDir(t)
	_, err := ScaffoldPack(ScaffoldOptions{
		Type:        "code",
		Language:    "go",
		Slug:        "error-handling",
		ProjectRoot: root,
	})
	if err != nil {
		t.Fatalf("ScaffoldPack returned error: %v", err)
	}
	recipePath := filepath.Join(root, "recipes", "go", "error-handling", "error-handling.recipe.md")
	if _, statErr := os.Stat(recipePath); statErr != nil {
		t.Fatalf("expected recipe file at %s, got error: %v", recipePath, statErr)
	}
	content, readErr := os.ReadFile(recipePath)
	if readErr != nil {
		t.Fatalf("reading recipe file: %v", readErr)
	}
	if len(content) == 0 {
		t.Fatal("recipe file should not be empty")
	}
}

func TestPackNew_CodePack_CreatesDirectoriesIfMissing(t *testing.T) {
	// Use a bare temp dir without pre-created recipes/
	root := t.TempDir()
	_, err := ScaffoldPack(ScaffoldOptions{
		Type:        "code",
		Language:    "go",
		Slug:        "error-handling",
		ProjectRoot: root,
	})
	if err != nil {
		t.Fatalf("ScaffoldPack returned error: %v", err)
	}
	dir := filepath.Join(root, "recipes", "go", "error-handling")
	info, statErr := os.Stat(dir)
	if statErr != nil {
		t.Fatalf("expected directories to be created, got error: %v", statErr)
	}
	if !info.IsDir() {
		t.Fatal("expected path to be a directory")
	}
}

// --- Conflict detection tests (REQ-009) ---

func TestPackNew_Conflict_RulePackFileExists(t *testing.T) {
	root := tempProjectDir(t)
	seedStandard(t, root, "go", 1, "error-handling")

	_, err := ScaffoldPack(ScaffoldOptions{
		Type:        "rule",
		Language:    "go",
		Slug:        "error-handling",
		Number:      1,
		ProjectRoot: root,
	})
	if err == nil {
		t.Fatal("ScaffoldPack should return error when standard file already exists")
	}
}

func TestPackNew_Conflict_CodePackDirExists(t *testing.T) {
	root := tempProjectDir(t)
	seedRecipeDir(t, root, "go", "error-handling")

	_, err := ScaffoldPack(ScaffoldOptions{
		Type:        "code",
		Language:    "go",
		Slug:        "error-handling",
		ProjectRoot: root,
	})
	if err == nil {
		t.Fatal("ScaffoldPack should return error when recipe directory already exists")
	}
}

func TestPackNew_Conflict_ErrorIdentifiesPath(t *testing.T) {
	root := tempProjectDir(t)
	seedStandard(t, root, "go", 1, "error-handling")

	_, err := ScaffoldPack(ScaffoldOptions{
		Type:        "rule",
		Language:    "go",
		Slug:        "error-handling",
		Number:      1,
		ProjectRoot: root,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	expectedPath := filepath.Join(root, "standards", "go", "STD-GO-001-error-handling.standard.md")
	if !strings.Contains(err.Error(), expectedPath) {
		t.Errorf("error should contain conflicting path %q, got %q", expectedPath, err.Error())
	}
}

// --- Output tests (REQ-007) ---

func TestPackNew_Output_JSON_Fields(t *testing.T) {
	root := tempProjectDir(t)
	result, err := ScaffoldPack(ScaffoldOptions{
		Type:        "rule",
		Language:    "go",
		Slug:        "error-handling",
		Number:      1,
		ProjectRoot: root,
	})
	if err != nil {
		t.Fatalf("ScaffoldPack returned error: %v", err)
	}

	// Marshal to JSON and verify required fields
	data, jsonErr := json.Marshal(result)
	if jsonErr != nil {
		t.Fatalf("marshaling result: %v", jsonErr)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}

	for _, field := range []string{"type", "language", "slug", "paths", "schema_version"} {
		if _, ok := m[field]; !ok {
			t.Errorf("JSON output missing field %q", field)
		}
	}
	if m["schema_version"] != "pack-new/v1" {
		t.Errorf("expected schema_version 'pack-new/v1', got %v", m["schema_version"])
	}
}

func TestPackNew_Output_Human_Display(t *testing.T) {
	root := tempProjectDir(t)
	result, err := ScaffoldPack(ScaffoldOptions{
		Type:        "rule",
		Language:    "go",
		Slug:        "error-handling",
		Number:      1,
		ProjectRoot: root,
	})
	if err != nil {
		t.Fatalf("ScaffoldPack returned error: %v", err)
	}

	human := result.HumanString()
	if !strings.Contains(human, "STD-GO-001") {
		t.Error("human output should contain pack identifier")
	}
	if !strings.Contains(human, "standards") {
		t.Error("human output should contain created path")
	}
}

func TestPackNew_Output_DataParity(t *testing.T) {
	root := tempProjectDir(t)
	result, err := ScaffoldPack(ScaffoldOptions{
		Type:        "rule",
		Language:    "go",
		Slug:        "error-handling",
		Number:      1,
		ProjectRoot: root,
	})
	if err != nil {
		t.Fatalf("ScaffoldPack returned error: %v", err)
	}

	// JSON output should contain the same data as human output
	human := result.HumanString()
	data, _ := json.Marshal(result)
	jsonStr := string(data)

	// Both should reference the pack ID and paths
	if !strings.Contains(human, result.PackID) {
		t.Error("human output missing PackID")
	}
	if !strings.Contains(jsonStr, result.PackID) {
		t.Error("JSON output missing PackID")
	}
	if !strings.Contains(human, result.Paths[0]) {
		t.Error("human output missing path")
	}
	if !strings.Contains(jsonStr, result.Paths[0]) {
		t.Error("JSON output missing path")
	}
}

// --- Additional coverage tests ---

func TestPackNew_ScaffoldPack_UnsupportedType(t *testing.T) {
	root := tempProjectDir(t)
	_, err := ScaffoldPack(ScaffoldOptions{
		Type:        "bogus",
		Language:    "go",
		Slug:        "my-pack",
		ProjectRoot: root,
	})
	if err == nil {
		t.Fatal("expected error for unsupported pack type")
	}
	if !strings.Contains(err.Error(), "unsupported pack type") {
		t.Errorf("expected 'unsupported pack type' in error, got %q", err.Error())
	}
}

func TestPackNew_RulePack_InvalidProjectRoot(t *testing.T) {
	// Use a path that cannot be created (file as parent)
	tmpDir := t.TempDir()
	blockingFile := filepath.Join(tmpDir, "standards")
	if err := os.WriteFile(blockingFile, []byte("block"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := ScaffoldPack(ScaffoldOptions{
		Type:        "rule",
		Language:    "go",
		Slug:        "my-pack",
		Number:      1,
		ProjectRoot: tmpDir,
	})
	if err == nil {
		t.Fatal("expected error when standards path is blocked by a file")
	}
}

func TestPackNew_CodePack_InvalidProjectRoot(t *testing.T) {
	tmpDir := t.TempDir()
	blockingFile := filepath.Join(tmpDir, "recipes")
	if err := os.WriteFile(blockingFile, []byte("block"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := ScaffoldPack(ScaffoldOptions{
		Type:        "code",
		Language:    "go",
		Slug:        "my-pack",
		ProjectRoot: tmpDir,
	})
	if err == nil {
		t.Fatal("expected error when recipes path is blocked by a file")
	}
}
