package main

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// TestDogfood_BackstopYmlHasNoLanguageKey (CLM-013): the dogfood backstop.yml carries
// NO `language:` key after the retirement — backstop-core models the post-retirement
// world (a project is described by its declared packs, not one baked language).
func TestDogfood_BackstopYmlHasNoLanguageKey(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "backstop.yml"))
	if err != nil {
		t.Fatalf("reading dogfood backstop.yml: %v", err)
	}
	for _, line := range strings.Split(string(raw), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "language:") {
			t.Errorf("the dogfood backstop.yml must carry NO `language:` key after the retirement, found: %q", strings.TrimSpace(line))
		}
	}
}

// TestGate_NoConfigLanguageReaderRemains (CLM-014): NO cfg.Language reader remains
// anywhere in cmd/backstop or pkg/gate non-test source — the de-language'd readers
// (gateConfig fallback, the deriveCapabilityState coverage-arm message, the
// code_check.go assignment, the e2e helper yml literal) are all updated.
func TestGate_NoConfigLanguageReaderRemains(t *testing.T) {
	cmdDir := filepath.Join(repoRoot(t), "cmd", "backstop")
	gateDir := filepath.Join(repoRoot(t), "pkg", "gate")
	// The needle is concatenated so the contiguous "cfg.Language" never appears as a
	// literal in this file — the CLM-025 completeness guard scans the test tree for it.
	needle := "cfg." + "Language"
	for _, dir := range []string{cmdDir, gateDir} {
		if grepNonTestSource(t, dir, needle) {
			t.Errorf("a config language reader survives in %s non-test source — every reader must be de-language'd before the field is removed (CLM-014)", dir)
		}
	}
}

// gateStepOutcome runs buildGateSteps over a temp project carrying the given
// backstop.yml content and returns the ordered "stepName=status" pairs — the
// observable for verdict-invariance.
func gateStepOutcome(t *testing.T, ymlContent string) []string {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "backstop.yml"), []byte(ymlContent), 0o644); err != nil {
		t.Fatal(err)
	}
	steps := buildGateSteps(root, emptyDiffScope())
	out := make([]string, 0, len(steps))
	for _, step := range steps {
		res := step(context.Background())
		out = append(out, res.StepName+"="+res.Status)
	}
	return out
}

// TestGate_LanguageKeyPresenceDoesNotChangeVerdict (CLM-015): the gate verdict is
// IDENTICAL whether or not a `language:` key is present in backstop.yml — the field
// is inert, so adding/removing it changes nothing about the gate result.
func TestGate_LanguageKeyPresenceDoesNotChangeVerdict(t *testing.T) {
	withLang := gateStepOutcome(t, "project: inv\nlanguage: go\n")
	without := gateStepOutcome(t, "project: inv\n")
	if !reflect.DeepEqual(withLang, without) {
		t.Errorf("the gate verdict must be IDENTICAL with/without a `language:` key (the field is inert)\n with=%v\n without=%v", withLang, without)
	}
}
