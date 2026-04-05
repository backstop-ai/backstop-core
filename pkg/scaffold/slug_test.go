package scaffold

import (
	"strings"
	"testing"
)

func TestArtifactNew_ValidSlug_Accepted(t *testing.T) {
	err := ValidateSlug("my-artifact")
	if err != nil {
		t.Fatalf("expected valid slug to be accepted, got error: %v", err)
	}
}

func TestArtifactNew_SlugStartsWithDigit_Exit2(t *testing.T) {
	err := ValidateSlug("1bad")
	if err == nil {
		t.Fatal("expected error for slug starting with digit, got nil")
	}
}

func TestArtifactNew_SlugUppercase_Exit2(t *testing.T) {
	err := ValidateSlug("BadSlug")
	if err == nil {
		t.Fatal("expected error for uppercase slug, got nil")
	}
}

func TestArtifactNew_SlugTooShort_Exit2(t *testing.T) {
	err := ValidateSlug("a")
	if err == nil {
		t.Fatal("expected error for slug shorter than 2 chars, got nil")
	}
}

func TestArtifactNew_SlugTooLong_Exit2(t *testing.T) {
	longSlug := strings.Repeat("a", 65)
	err := ValidateSlug(longSlug)
	if err == nil {
		t.Fatal("expected error for slug longer than 64 chars, got nil")
	}
}

func TestArtifactNew_MissingSlug_Exit2(t *testing.T) {
	err := ValidateSlug("")
	if err == nil {
		t.Fatal("expected error for empty slug, got nil")
	}
}
