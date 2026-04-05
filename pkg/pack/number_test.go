package pack

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// --- Language validation tests (REQ-002) ---

func TestPackNew_ValidLanguage_Accepted(t *testing.T) {
	for _, lang := range []string{"go", "typescript", "python", "rust"} {
		if err := ValidateLanguage(lang); err != nil {
			t.Errorf("ValidateLanguage(%q) returned error: %v", lang, err)
		}
	}
}

func TestPackNew_MissingLanguage_Exit2(t *testing.T) {
	// Missing language is represented as empty string from flag parsing
	err := ValidateLanguage("")
	if err == nil {
		t.Fatal("ValidateLanguage(\"\") should return an error for missing language")
	}
}

func TestPackNew_EmptyLanguage_Exit2(t *testing.T) {
	err := ValidateLanguage("")
	if err == nil {
		t.Fatal("ValidateLanguage(\"\") should return an error for empty language")
	}
}

func TestPackNew_LanguageUppercase_Exit2(t *testing.T) {
	for _, lang := range []string{"Go", "TypeScript", "PYTHON"} {
		err := ValidateLanguage(lang)
		if err == nil {
			t.Errorf("ValidateLanguage(%q) should return an error for uppercase", lang)
		}
	}
}

func TestPackNew_LanguageInvalidChars_Exit2(t *testing.T) {
	for _, lang := range []string{"c-sharp", "go2", "type_script", "c++"} {
		err := ValidateLanguage(lang)
		if err == nil {
			t.Errorf("ValidateLanguage(%q) should return an error for invalid chars", lang)
		}
	}
}

// --- Number resolution tests (REQ-006) ---

func TestPackNew_NumberAssign_StartsAt001(t *testing.T) {
	root := tempProjectDir(t)
	// No standards exist for "go" — should return 1
	num, err := ResolvePackNumber("go", root)
	if err != nil {
		t.Fatalf("ResolvePackNumber returned error: %v", err)
	}
	if num != 1 {
		t.Fatalf("expected number 1 for empty language dir, got %d", num)
	}
}

func TestPackNew_NumberAssign_ScansExisting(t *testing.T) {
	root := tempProjectDir(t)
	seedStandard(t, root, "go", 1, "error-handling")
	seedStandard(t, root, "go", 3, "concurrency")

	num, err := ResolvePackNumber("go", root)
	if err != nil {
		t.Fatalf("ResolvePackNumber returned error: %v", err)
	}
	if num != 4 {
		t.Fatalf("expected number 4 (max 3 + 1), got %d", num)
	}
}

func TestPackNew_NumberAssign_GapsPreserved(t *testing.T) {
	root := tempProjectDir(t)
	seedStandard(t, root, "go", 1, "first")
	seedStandard(t, root, "go", 3, "third")
	// Gap at 2 should NOT be filled

	num, err := ResolvePackNumber("go", root)
	if err != nil {
		t.Fatalf("ResolvePackNumber returned error: %v", err)
	}
	if num == 2 {
		t.Fatal("ResolvePackNumber should not fill gaps; expected 4, got 2")
	}
	if num != 4 {
		t.Fatalf("expected number 4, got %d", num)
	}
}

func TestPackNew_NumberAssign_ZeroPadded(t *testing.T) {
	root := tempProjectDir(t)
	seedStandard(t, root, "go", 2, "existing")

	num, err := ResolvePackNumber("go", root)
	if err != nil {
		t.Fatalf("ResolvePackNumber returned error: %v", err)
	}
	// Verify the number when formatted as %03d produces zero-padded string
	formatted := fmt.Sprintf("%03d", num)
	if formatted != "003" {
		t.Fatalf("expected zero-padded '003', got %q", formatted)
	}
}

func TestPackNew_NumberAssign_NoStandardsDir(t *testing.T) {
	// Project root with no standards/ directory at all
	root := t.TempDir()
	num, err := ResolvePackNumber("go", root)
	if err != nil {
		t.Fatalf("ResolvePackNumber returned error: %v", err)
	}
	if num != 1 {
		t.Fatalf("expected 1 when standards dir does not exist, got %d", num)
	}
}

func TestPackNew_NumberAssign_NonMatchingFiles(t *testing.T) {
	root := tempProjectDir(t)
	// Create standards/go/ with files that don't match the pattern
	dir := filepath.Join(root, "standards", "go")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "other.txt"), []byte("world"), 0o644); err != nil {
		t.Fatal(err)
	}

	num, err := ResolvePackNumber("go", root)
	if err != nil {
		t.Fatalf("ResolvePackNumber returned error: %v", err)
	}
	if num != 1 {
		t.Fatalf("expected 1 when no matching files, got %d", num)
	}
}

func TestPackNew_NumberAssign_IgnoresOtherLanguages(t *testing.T) {
	root := tempProjectDir(t)
	// Seed standards for different language
	seedStandard(t, root, "typescript", 5, "security")
	// go should still start at 1
	num, err := ResolvePackNumber("go", root)
	if err != nil {
		t.Fatalf("ResolvePackNumber returned error: %v", err)
	}
	if num != 1 {
		t.Fatalf("expected 1 for go (typescript standards should not count), got %d", num)
	}
}
