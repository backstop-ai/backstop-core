package pack

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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

// --- Sample-validator discrimination tests (ISSUE-146) ---
//
// These drive the validator the scaffolder ACTUALLY WROTE, never a literal restated
// here: a test that re-declared the expected script bytes would stay green while the
// shipped scaffolder was broken. They execute it directly rather than through
// pkg/packval because pkg/packval imports pkg/pack, so a test in this package cannot
// reach the pipeline without an import cycle. The real-pipeline, real-sandbox proof
// lives in cmd/backstop/pack_new_falsification_test.go.

// scaffoldSamplePack scaffolds a pack of the given type into a fresh temp project and
// returns its pack directory.
func scaffoldSamplePack(t *testing.T, packType, slug string) string {
	t.Helper()
	root := tempProjectDir(t)
	if _, err := ScaffoldPack(ScaffoldOptions{Type: packType, Language: "go", Slug: slug, ProjectRoot: root}); err != nil {
		t.Fatalf("type %s: ScaffoldPack error: %v", packType, err)
	}
	return filepath.Join(root, slug)
}

// runScaffoldedValidator executes the scaffolded validator script against the given
// targets and returns its exit code plus combined output.
func runScaffoldedValidator(t *testing.T, packDir, slug string, targets ...string) (int, string) {
	t.Helper()
	args := append([]string{filepath.Join(packDir, "validators", slug+".sh")}, targets...)
	out, err := exec.Command("sh", args...).CombinedOutput()
	if err == nil {
		return 0, string(out)
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode(), string(out)
	}
	t.Fatalf("running scaffolded validator: %v (output: %s)", err, out)
	return 0, ""
}

// TestScaffoldPack_SampleValidatorFiresOnNegativeFixture proves the scaffolded
// validator FLAGS the scaffolded negative fixture (CLM-001, CLM-002). It is one half
// of a pair: a validator that exits non-zero unconditionally would also pass this, so
// it means nothing without TestScaffoldPack_SampleValidatorIsSilentOnPositiveFixture.
// The output assertion is not cosmetic — runSandboxEngine (cmd/backstop/pack_gate.go)
// uses the validator's combined output VERBATIM as the violation message, so a silent
// non-zero exit leaves a pack author nothing to act on.
func TestScaffoldPack_SampleValidatorFiresOnNegativeFixture(t *testing.T) {
	packDir := scaffoldSamplePack(t, "engine", "sample-check")
	negative := filepath.Join(packDir, "fixtures", "invalid", "example.txt")

	code, out := runScaffoldedValidator(t, packDir, "sample-check", negative)
	if code == 0 {
		t.Fatalf("scaffolded validator exited 0 on the negative fixture; it must flag it. output: %q", out)
	}
	if strings.TrimSpace(out) == "" {
		t.Error("scaffolded validator flagged the negative fixture silently; runSandboxEngine reports this output as the violation message")
	}
	if !strings.Contains(out, negative) {
		t.Errorf("validator output must name the target it flagged; got %q, want it to contain %q", out, negative)
	}
}

// TestScaffoldPack_SampleValidatorIsSilentOnPositiveFixture is the other half of the
// pair (CLM-001, CLM-002): the old `exit 0` validator passed THIS and failed its
// sibling. Only both together assert discrimination.
func TestScaffoldPack_SampleValidatorIsSilentOnPositiveFixture(t *testing.T) {
	packDir := scaffoldSamplePack(t, "engine", "sample-check")
	positive := filepath.Join(packDir, "fixtures", "valid", "example.txt")

	code, out := runScaffoldedValidator(t, packDir, "sample-check", positive)
	if code != 0 {
		t.Fatalf("scaffolded validator exited %d on the clean positive fixture; it must stay silent. output: %q", code, out)
	}
}

// TestScaffoldPack_SampleValidatorIgnoresNonFileTargets pins the GATE-TIME argument
// shape (CLM-004). The scaffolded rule declares no input_scope, so runSandboxEngine
// (cmd/backstop/pack_gate.go, the `targets := []string{projectRoot}` branch) hands the
// validator exactly ONE argument: the project ROOT DIRECTORY. A non-zero exit there
// becomes a blocking gate violation, so a marker-scanning validator without a
// regular-file guard would red every consumer's gate on its first run.
//
// THE DIRECTORY MUST BE THE SCAFFOLDED PACK DIR, NOT A BARE t.TempDir(). The pack dir
// CONTAINS a marker-bearing file (fixtures/invalid/example.txt), so passing it proves
// the validator refuses to recurse into a directory even when the marker is
// demonstrably reachable underneath. An empty temp dir would satisfy this assertion
// trivially — even a validator with no regular-file guard finds nothing in it — and
// CLM-004 would be pinned by nothing. Do not "simplify" this.
func TestScaffoldPack_SampleValidatorIgnoresNonFileTargets(t *testing.T) {
	packDir := scaffoldSamplePack(t, "engine", "sample-check")

	// Guard the premise: the marker really is reachable under this directory.
	negative, err := os.ReadFile(filepath.Join(packDir, "fixtures", "invalid", "example.txt"))
	if err != nil {
		t.Fatalf("reading scaffolded negative fixture: %v", err)
	}
	if code, _ := runScaffoldedValidator(t, packDir, "sample-check", filepath.Join(packDir, "fixtures", "invalid", "example.txt")); code == 0 {
		t.Fatalf("premise broken: the marker-bearing file under %s does not trip the validator, so the directory case proves nothing (fixture: %q)", packDir, negative)
	}

	if code, out := runScaffoldedValidator(t, packDir, "sample-check", packDir); code != 0 {
		t.Errorf("validator exited %d on the pack DIRECTORY; at gate time runSandboxEngine passes the project root directory and a non-zero exit is a blocking violation. output: %q", code, out)
	}

	missing := filepath.Join(packDir, "does-not-exist-anywhere")
	if code, out := runScaffoldedValidator(t, packDir, "sample-check", missing); code != 0 {
		t.Errorf("validator exited %d on a nonexistent path %s; it must ignore non-regular-file targets. output: %q", code, missing, out)
	}
}

// scaffoldProseNormalizer collapses every run of newline + leading whitespace +
// optional comment-continuation `#` + whitespace down to a single space.
var scaffoldProseNormalizer = regexp.MustCompile(`(?s)\s*\n\s*#?\s*`) // nosemgrep: go.core.no-global-mutable-state — compile-once immutable regexp, never reassigned

// normalizeScaffoldProse lower-cases text and unwraps its line breaks so a phrase
// split across a wrapped comment is matched as the single phrase it reads as.
func normalizeScaffoldProse(s string) string {
	return scaffoldProseNormalizer.ReplaceAllString(strings.ToLower(s), " ")
}

// TestScaffoldPack_ScaffoldedBytesDoNotClaimAlwaysPasses is a DRIFT GUARD over the
// scaffolder's own author-facing prose (CLM-005): no shipped byte may still advertise
// that the sample validator does nothing.
//
// ★ THE NORMALISATION IS LOAD-BEARING, NOT COSMETIC. A raw substring scan is VACUOUS
// on the pack.yml arm — the arm that matters most, because it is the comment every new
// pack author reads first. The manifest format string wraps the phrase across a line
// break, emitting `...currently always\n      # passes — replace it...`, so
// strings.Contains(lower(b), "always pass") reports GREEN there while the claim is
// plainly present. Unwrapping first is what makes a PARTIAL cleanup — fixing only the
// validator literal — stay red.
//
// The SUBSTRING is forbidden, not an exact sentence: an exact-sentence assertion is
// evaded by any rewording that keeps the claim, while the substring also catches
// "always passes", "always-pass" and "always passing".
//
// The packTypeBlurb arm starts GREEN and is a REGRESSION FENCE, not a red-first
// assertion — today's blurbs assert nothing about always passing. That is a legitimate
// reason for it to pass on arrival; do not delete it because it does not fail.
func TestScaffoldPack_ScaffoldedBytesDoNotClaimAlwaysPasses(t *testing.T) {
	const forbidden = "always pass"

	for _, typ := range []string{"engine", "mechanism", "toolchain"} {
		packDir := scaffoldSamplePack(t, typ, "sample-check")

		for _, artefact := range []struct{ label, rel string }{
			{"validator", filepath.Join("validators", "sample-check.sh")},
			{"pack.yml", "pack.yml"},
		} {
			data, err := os.ReadFile(filepath.Join(packDir, artefact.rel))
			if err != nil {
				t.Fatalf("type %s: reading scaffolded %s: %v", typ, artefact.label, err)
			}
			if got := normalizeScaffoldProse(string(data)); strings.Contains(got, forbidden) {
				t.Errorf("type %s: scaffolded %s still claims the sample %q (normalised text: %q)", typ, artefact.label, forbidden, got)
			}
		}

		if got := normalizeScaffoldProse(packTypeBlurb(typ)); strings.Contains(got, forbidden) {
			t.Errorf("type %s: packTypeBlurb still claims the sample %q: %q", typ, forbidden, got)
		}
	}
}
