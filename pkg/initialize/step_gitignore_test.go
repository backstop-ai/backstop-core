package initialize

import (
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/pack"
)

// manifestFromYAML parses a pack manifest through the SHIPPED pack.ParseManifest, so
// every gitignore assertion is made against the pack's OWN declared YAML shape.
//
// A fabricated Go struct would let these claims keep passing against a shape no real
// pack could produce, which is the whole failure mode "parsed through ParseManifest,
// never fabricated" exists to close.
func manifestFromYAML(t *testing.T, body string) *pack.Manifest {
	t.Helper()
	manifest, err := pack.ParseManifest([]byte(body))
	if err != nil {
		t.Fatalf("parsing the fixture manifest: %v\n---\n%s", err, body)
	}
	return manifest
}

// gitignoreLines returns the written .gitignore split into non-empty, non-comment
// entries.
func gitignoreLines(t *testing.T, root string) []string {
	t.Helper()
	lines := []string{}
	for _, line := range strings.Split(readFile(t, root, ".gitignore"), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		lines = append(lines, trimmed)
	}
	return lines
}

// hasLine reports whether entry appears in lines.
func hasLine(lines []string, entry string) bool {
	for _, line := range lines {
		if line == entry {
			return true
		}
	}
	return false
}

// TestInit_GitignoreCarriesTheThreeBackstopOwnedEntries (SPEC-069 CLM-029).
//
// EXACTLY three literals are stated in core, and they are backstop's own paths.
func TestInit_GitignoreCarriesTheThreeBackstopOwnedEntries(t *testing.T) {
	root := t.TempDir()

	report := stepGitignore(root, nil)
	if report.Outcome != OutcomeDelivered {
		t.Fatalf("gitignore step reported %v (%s), want OutcomeDelivered", report.Outcome, report.Detail)
	}

	lines := gitignoreLines(t, root)
	for _, entry := range []string{".backstop/packs/", ".backstop/baseline.json", ".backstop/pack-config-provenance.json"} {
		if !hasLine(lines, entry) {
			t.Fatalf("the generated .gitignore is missing the backstop-owned entry %q.\ngot: %v", entry, lines)
		}
	}
	if len(lines) != 3 {
		t.Fatalf("with no installed pack the entry set is %v; it must be exactly the three backstop-owned literals and nothing else", lines)
	}
}

// TestInit_GitignoreIncludesEveryInstalledPackEngineStdoutArtifact (SPEC-069 CLM-030).
//
// For each installed pack, EVERY engine's declared `stdout_artifact` appears as an
// entry, sourced from the MANIFEST rather than from any core list. That field is the
// only generated-output declaration a pack manifest carries, which is precisely why
// the entry set can be pack-derived at all.
func TestInit_GitignoreIncludesEveryInstalledPackEngineStdoutArtifact(t *testing.T) {
	root := t.TempDir()

	first := manifestFromYAML(t, `
name: fixture/first
version: 1.0.0
language: neutral
archetype: enforcement
description: A fixture manifest whose only job is to declare engines for the gitignore step.
engines:
  one:
    command: grep -rn
    input_mode: none
    scope_kind: project-wide
    gate_type: test
    stdout_artifact: build/first-engine-output.json
  two:
    command: grep -rn
    input_mode: none
    scope_kind: project-wide
    gate_type: coverage
    stdout_artifact: build/second-engine-output.xml
content:
  ruleset:
    version: 1.0.0
    rules: []
`)
	second := manifestFromYAML(t, `
name: fixture/second
version: 1.0.0
language: neutral
archetype: enforcement
description: A fixture manifest whose only job is to declare engines for the gitignore step.
engines:
  only:
    command: grep -rn
    input_mode: none
    scope_kind: project-wide
    gate_type: lint
    stdout_artifact: reports/second-pack-output.txt
content:
  ruleset:
    version: 1.0.0
    rules: []
`)

	if report := stepGitignore(root, []*pack.Manifest{first, second}); report.Outcome != OutcomeDelivered {
		t.Fatalf("gitignore step reported %v (%s)", report.Outcome, report.Detail)
	}

	lines := gitignoreLines(t, root)
	for _, declared := range []string{
		"build/first-engine-output.json",
		"build/second-engine-output.xml",
		"reports/second-pack-output.txt",
	} {
		if !hasLine(lines, declared) {
			t.Fatalf("the generated .gitignore is missing the pack-declared stdout_artifact %q.\ngot: %v", declared, lines)
		}
	}
}

// TestInit_PackWithoutStdoutArtifactContributesNoGitignoreEntry (SPEC-069 CLM-031).
//
// An engine declaring none contributes NOTHING: init invents no path for it. Guessing
// one would be core holding generated-output knowledge about a tool it knows nothing
// about.
func TestInit_PackWithoutStdoutArtifactContributesNoGitignoreEntry(t *testing.T) {
	root := t.TempDir()

	silent := manifestFromYAML(t, `
name: fixture/declares-nothing
version: 1.0.0
language: neutral
archetype: enforcement
description: A fixture manifest whose only job is to declare engines for the gitignore step.
engines:
  quiet:
    command: grep -rn
    input_mode: none
    scope_kind: project-wide
    gate_type: test
content:
  ruleset:
    version: 1.0.0
    rules: []
`)

	stepGitignore(root, []*pack.Manifest{silent})

	lines := gitignoreLines(t, root)
	if len(lines) != 3 {
		t.Fatalf("a pack whose engine declares no stdout_artifact contributed %d extra entries (%v); init must invent no path for it",
			len(lines)-3, lines)
	}
}

// TestInit_GitignoreHoldsNoToolOrLanguageSpecificPathFromCore (SPEC-069 CLM-032,
// denylist).
//
// The entry set is exactly the three backstop literals plus pack-declared values. No
// language, framework or tool path from core — the ignore list DD-7's own correction
// struck.
func TestInit_GitignoreHoldsNoToolOrLanguageSpecificPathFromCore(t *testing.T) {
	root := t.TempDir()
	stepGitignore(root, nil)

	body := readFile(t, root, ".gitignore")
	// A representative set of the paths a language-specific ignore list would carry.
	// They live in this TEST, not in the source set, so naming them here is an
	// assertion rather than a bake.
	forbidden := []string{
		"node_modules", "vendor/", "target/", "__pycache__", "dist/", "build/",
		".venv", "Cargo.lock", "*.pyc", "*.class", "*.o",
	}
	for _, path := range forbidden {
		if strings.Contains(body, path) {
			t.Fatalf("the generated .gitignore carries the tool/language-specific path %q, which core must not enumerate.\n---\n%s", path, body)
		}
	}
}

// TestInit_ReportStatesTheUncoveredGitignoreResidue (SPEC-069 CLM-033).
//
// `stdout_artifact` names what an engine writes FOR THE GATE TO READ, not everything a
// toolchain leaves on disk. The residue is real and init states it in WORDS rather
// than closing it by guessing — closing it needs a new pack-manifest field, which is
// another bundle's surface.
func TestInit_ReportStatesTheUncoveredGitignoreResidue(t *testing.T) {
	root := t.TempDir()
	report := stepGitignore(root, nil)

	detail := strings.ToLower(report.Detail)
	if !strings.Contains(detail, "stdout_artifact") {
		t.Fatalf("the report does not name stdout_artifact as the source of the pack-derived entries.\ngot: %s", report.Detail)
	}
	if !strings.Contains(detail, "gate") {
		t.Fatalf("the report does not state that stdout_artifact names what an engine writes for the GATE to read, which is the whole reason the residue exists.\ngot: %s", report.Detail)
	}
	for _, cue := range []string{"yours", "your own", "consumer"} {
		if strings.Contains(detail, cue) {
			return
		}
	}
	t.Fatalf("the report does not leave the uncovered paths to the consumer in words.\ngot: %s", report.Detail)
}

// TestInit_ExistingGitignoreIsAppendedToNeverRewritten (SPEC-069 CLM-034).
//
// Every pre-existing BYTE survives; no line is rewritten, reordered or dropped. This
// is the same posture the shipped ensureGitignore already takes on the pack-add path:
// read, skip what is present, append the rest.
func TestInit_ExistingGitignoreIsAppendedToNeverRewritten(t *testing.T) {
	root := t.TempDir()
	original := "# the consumer's own header\n\nsecret.env\n*.local\n\n# a trailing comment\n"
	writeFile(t, root, ".gitignore", original)

	report := stepGitignore(root, nil)
	if report.Outcome == OutcomeBrokenPromise {
		t.Fatalf("the gitignore step failed over an existing file: %s", report.Detail)
	}

	after := readFile(t, root, ".gitignore")
	if !strings.HasPrefix(after, original) {
		t.Fatalf("the pre-existing .gitignore was not preserved byte-for-byte as a PREFIX; entries are appended, never merged in place.\nbefore:\n%s\nafter:\n%s", original, after)
	}
	for _, entry := range []string{".backstop/packs/", ".backstop/baseline.json", ".backstop/pack-config-provenance.json"} {
		if !hasLine(gitignoreLines(t, root), entry) {
			t.Fatalf("the missing entry %q was not appended to the existing file.\n---\n%s", entry, after)
		}
	}
}

// TestInit_GitignoreEntryAlreadyPresentIsNotDuplicated (SPEC-069 CLM-035).
func TestInit_GitignoreEntryAlreadyPresentIsNotDuplicated(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, ".gitignore", "# already partly adopted\n.backstop/packs/\n.backstop/baseline.json\n")

	stepGitignore(root, nil)

	lines := gitignoreLines(t, root)
	counts := map[string]int{}
	for _, line := range lines {
		counts[line]++
	}
	for entry, count := range counts {
		if count > 1 {
			t.Fatalf("%q appears %d times in the .gitignore; an entry already present must not be duplicated.\ngot: %v", entry, count, lines)
		}
	}
	if !hasLine(lines, ".backstop/pack-config-provenance.json") {
		t.Fatalf("the one genuinely missing entry was not appended.\ngot: %v", lines)
	}
}

// TestInit_GitignoreConvergesWhenEveryEntryIsPresent asserts the step reports
// converged and writes nothing when there is nothing to add. Additive: it is what
// makes the runner's second-run convergence claim satisfiable.
func TestInit_GitignoreConvergesWhenEveryEntryIsPresent(t *testing.T) {
	root := t.TempDir()

	if first := stepGitignore(root, nil); first.Outcome != OutcomeDelivered {
		t.Fatalf("first gitignore run reported %v", first.Outcome)
	}
	before := readFile(t, root, ".gitignore")

	second := stepGitignore(root, nil)
	if second.Outcome != OutcomeConverged {
		t.Fatalf("second gitignore run reported %v (%s), want OutcomeConverged", second.Outcome, second.Detail)
	}
	if after := readFile(t, root, ".gitignore"); after != before {
		t.Fatalf("the second run rewrote the file.\nbefore:\n%s\nafter:\n%s", before, after)
	}
}
