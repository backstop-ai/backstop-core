package packval

import (
	"os"
	"strings"
	"testing"
)

// TestPhase1_NonGoLanguageNotRejected (CLM-008): a pack whose language is not "go"
// is NOT rejected on the language field — the harness is language-neutral, so it
// validates a TypeScript or Rust pack structurally as far as the language field is
// concerned.
func TestPhase1_NonGoLanguageNotRejected(t *testing.T) {
	for _, lang := range []string{"typescript", "rust"} {
		m := &PackManifest{
			Name: "a/b", Version: "1.0.0", Language: lang, Archetype: "enforcement",
			Content: Content{Ruleset: Ruleset{Rules: []Rule{{ID: "r1", RiskClass: "correctness"}}}},
		}
		r := RunStructural(m, t.TempDir())
		for _, e := range r.Errors {
			if e.Check == "language" || e.ManifestPath == "language" && strings.Contains(e.Message, "unsupported") {
				t.Fatalf("language %q must not be rejected, got error: %+v", lang, e)
			}
		}
	}
}

// TestPhase1_LanguageGateSourceGone (CLM-008): the `!= "go"` comparison and the
// "unsupported language" message are gone from phase1.go, while the
// PackManifest.Language field itself is RETAINED (docs).
func TestPhase1_LanguageGateSourceGone(t *testing.T) {
	src, err := os.ReadFile("phase1.go")
	if err != nil {
		t.Fatalf("read phase1.go: %v", err)
	}
	s := string(src)
	if strings.Contains(s, `!= "go"`) {
		t.Fatal(`phase1.go still contains the != "go" language gate`)
	}
	if strings.Contains(s, "unsupported language") {
		t.Fatal(`phase1.go still contains the "unsupported language" rejection`)
	}
	// The Language field must remain on the manifest for documentation.
	mfst, err := os.ReadFile("manifest.go")
	if err != nil {
		t.Fatalf("read manifest.go: %v", err)
	}
	if !strings.Contains(string(mfst), "Language") {
		t.Fatal("PackManifest.Language field must be retained for docs")
	}
}
