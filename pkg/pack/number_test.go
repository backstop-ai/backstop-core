package pack

import (
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

// The ResolvePackNumber tests are DELETED with the function itself (ISSUE-032 Defect
// A / ISSUE-030 fold): engine packs are named by slug, not auto-numbered from a
// standards/ scan. The deletion is pinned by deletion_assertion_test.go.
