package distribution_test

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// spec015_lineage_test.go is the mechanical guard (CLM-100) that a 139-test
// migration cannot quietly sever SPEC-015's claim-to-test lineage.
//
// BUNDLE-006 forbids rewriting SPEC-015's historical requirement pins; renaming
// a test the spec names in a claim's `tests:` list severs the same lineage just
// as effectively, and the failure is QUIET — the suite still passes and only
// SPEC-015's traceability goes red much later. This is a RATCHET: it is green
// before the migration and must stay green through it.

// spec015MandatedNamesInSuites is the FLOOR: how many of SPEC-015's mandated
// test names live in the six migrated suites. It is a count, deliberately not a
// name list — the names themselves are read from the spec so they cannot drift,
// while the count is what makes the comparison non-tautological (an intersection
// recomputed from the current files would otherwise agree with itself).
//
// Raising this floor when the spec gains claims is expected. LOWERING it is the
// thing this file exists to make impossible without a deliberate edit.
const spec015MandatedNamesInSuites = 62

// spec015MigratedSuites returns the six suites Phase 10 of PLAN-SPEC-055 migrates
// onto constructor-assembled commands. It is a function rather than a package
// variable so nothing can mutate the set the guard checks.
func spec015MigratedSuites() []string {
	return []string{
		"add_test.go",
		"update_test.go",
		"upgrade_test.go",
		"install_test.go",
		"install_materialize_test.go",
		"install_reconcile_test.go",
	}
}

// A claim's mandated tests are YAML sequence entries in the spec frontmatter, so
// a mandated name is a list item whose whole value is a Go test identifier.
var spec015MandatedEntry = regexp.MustCompile(`(?m)^[ \t]*-[ \t]+(Test[A-Za-z0-9_]+)[ \t]*$`)

// A declared test function, matched at the start of a line so a name merely
// MENTIONED in a comment or called from another test does not count as declared.
var spec015DeclaredTestFunc = regexp.MustCompile(`(?m)^func[ \t]+(Test[A-Za-z0-9_]+)[ \t]*\(`)

// TestDistribution_Spec015MandatedTestNamesPreserved asserts that every
// SPEC-015-mandated test name living in the six migrated distribution suites is
// still declared there VERBATIM.
//
// The mandated names are read out of the spec, never from a list embedded here:
// a hardcoded list drifts silently the moment the spec moves and turns the guard
// into decoration. If the spec cannot be read, this FAILS — a guard that passes
// when it cannot see its own input is worse than no guard.
func TestDistribution_Spec015MandatedTestNamesPreserved(t *testing.T) {
	root := spec015RepoRoot(t)

	specPath := filepath.Join(root, "specs", "SPEC-015-pack-distribution.spec.md")
	specSource, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("reading the mandated names from %s: %v; this guard cannot run without its input and must not pass silently", specPath, err)
	}

	mandated := spec015UniqueMatches(spec015MandatedEntry, string(specSource))
	if len(mandated) < spec015MandatedNamesInSuites {
		t.Fatalf("extracted only %d mandated test names from %s, fewer than the %d expected to live in the suites alone; the extraction is broken, not the corpus",
			len(mandated), specPath, spec015MandatedNamesInSuites)
	}

	declared := spec015DeclaredInSuites(t, root)

	var present []string
	for _, name := range mandated {
		if _, ok := declared[name]; ok {
			present = append(present, name)
		}
	}
	sort.Strings(present)

	if len(present) < spec015MandatedNamesInSuites {
		var missing []string
		for _, name := range mandated {
			if _, ok := declared[name]; !ok {
				missing = append(missing, name)
			}
		}
		sort.Strings(missing)
		t.Fatalf("only %d of SPEC-015's mandated test names are still declared in the six migrated suites, below the floor of %d — a mandated test was renamed, merged, or deleted and SPEC-015's claim-to-test lineage is severed.\n%d mandated names are not declared in these suites (most legitimately live elsewhere, e.g. cmd/backstop; compare against the floor, not this list):\n  %s",
			len(present), spec015MandatedNamesInSuites, len(missing), strings.Join(missing, "\n  "))
	}
}

// spec015DeclaredInSuites maps every test function declared across the six
// migrated suites to the suite declaring it.
func spec015DeclaredInSuites(t *testing.T, root string) map[string]string {
	t.Helper()

	declared := map[string]string{}
	for _, suite := range spec015MigratedSuites() {
		suitePath := filepath.Join(root, "pkg", "pack", "distribution", suite)
		suiteSource, err := os.ReadFile(suitePath)
		if err != nil {
			t.Fatalf("reading migrated suite %s: %v; a suite that cannot be read cannot be checked for renames", suitePath, err)
		}
		for _, name := range spec015UniqueMatches(spec015DeclaredTestFunc, string(suiteSource)) {
			declared[name] = suite
		}
	}
	if len(declared) == 0 {
		t.Fatal("found no declared test functions across the six migrated suites; the declaration scan is broken")
	}

	return declared
}

// spec015AbsentDependencyPremiseTests returns the tests whose assertion SUBJECT
// is that an absent dependency makes the pipeline skip something.
//
// The constructors make their premise unrepresentable, so they cannot be
// migrated — but the skips they assert are STILL LIVE until REQ-008/REQ-009
// delete them. Retiring them before then would leave live behavior untested.
func spec015AbsentDependencyPremiseTests() []string {
	return []string{
		"TestPackAdd_NilValidatorSkipsValidation",
		"TestPackUpdate_NilValidatorSkipsValidation",
		"TestPackUpgrade_SkipsValidationWhenValidatorNil",
		"TestPackUpgrade_NoRemediationWhenGeneratorNil",
		"TestPackUpgrade_SkipsScanningWhenScannerNil",
	}
}

// TestDistribution_AbsentDependencyPremiseTestsAwaitSkipRemoval asserts these
// five tests are STILL PRESENT — their survival through the suite migration is a
// deliberate decision, not an oversight.
//
// They ride the package-level entry points one phase longer so that they die in
// the SAME change that deletes the skips they assert, with the inverse claims
// (the validator, scanner, and remediation generator running unconditionally)
// landing simultaneously. Coverage never has a window where the skip behavior is
// live and untested.
//
// DELETE THIS GUARD together with the five, in the change that removes the
// nil-dependency skips. Until then a failure here means someone retired them
// early and took live behavior's only coverage with them.
func TestDistribution_AbsentDependencyPremiseTestsAwaitSkipRemoval(t *testing.T) {
	declared := spec015DeclaredInSuites(t, spec015RepoRoot(t))

	for _, name := range spec015AbsentDependencyPremiseTests() {
		if _, ok := declared[name]; !ok {
			t.Errorf("%s is gone; it asserts a skip that is still live, and nothing else covers that behavior until the skip itself is deleted", name)
		}
	}
}

// spec015RepoRoot walks up from the working directory to the repository root so
// the spec is located by structure rather than by a baked absolute path.
func spec015RepoRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("resolving the working directory to walk up from: %v", err)
	}

	for {
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil { // nosemgrep: backstop.packs.backstop.self.rules.no-baked-language-token — test walks up to the repo root by module file, not baked routing
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("walked to the filesystem root without finding the repository root above %q; this guard cannot locate SPEC-015 and must not pass silently", dir)
		}
		dir = parent
	}
}

// spec015UniqueMatches returns the deduplicated first capture group of every
// match, in first-seen order.
func spec015UniqueMatches(re *regexp.Regexp, source string) []string {
	seen := map[string]bool{}
	var names []string
	for _, match := range re.FindAllStringSubmatch(source, -1) {
		name := match[1]
		if seen[name] {
			continue
		}
		seen[name] = true
		names = append(names, name)
	}
	return names
}
