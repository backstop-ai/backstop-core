package gate

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/artifact"
)

// step_testverify_layout_test.go pins that the four spec-discovery filters in this file
// still DISCRIMINATE after ISSUE-124 routed them through the shared artifact layout
// table.
//
// ★ THESE ARE GREEN BEFORE THE SWAP AND GREEN AFTER, BY DESIGN. They do not falsify the
// duplication — that is the source scan in pkg/artifact/layout_consumer_scan_test.go.
// They falsify a BOTCHED swap, specifically the one way this change can go
// catastrophically wrong:
//
//	layout, _ := artifact.LayoutFor(artifact.KindSpec)   // ok DISCARDED
//	if entry.IsDir() || !strings.HasSuffix(entry.Name(), layout.Extension) {
//
// LayoutFor returns a ZERO-VALUE KindLayout when ok is false, and its Extension is the
// EMPTY STRING. strings.HasSuffix(anything, "") is ALWAYS TRUE, so that one discarded
// bool inverts a "specs only" filter into "accept every file in the directory" —
// silently, with no error, with ExtractMandatedTests then parsing README.md as spec
// frontmatter. Every scoped call site must handle ok=false explicitly.
//
// ★★ THE DECOYS' FRONTMATTER IS LOAD-BEARING. DO NOT SIMPLIFY THEM INTO EMPTY FILES AND
// DO NOT COLLAPSE THE SET DOWN TO ONE SPEC. All four extractors SKIP a file they cannot
// use for reasons that have nothing to do with the extension filter — unparseable
// frontmatter, wrong status, missing verification block, missing contracts. So a bare
// decoy like an empty README.md contributes NOTHING even when the filter is fully
// widened, both outputs come back byte-identical, and this guard passes in a world where
// the bug is live. Every field below exists to clear one specific downstream filter.
//
// ★ AND THE STATUS WINDOWS ARE MUTUALLY EXCLUSIVE, which is why there are TWO real
// specs rather than one:
//
//	contractsAreDue(s)      ≡  s == "implemented"
//	isTerminalSpecStatus(s) ≡  s ∈ {replaced, canceled, deprecated}
//
// `implemented` is therefore NOT terminal. An all-`implemented` fixture set leaves
// CountTerminalSpecs returning 0 whether the filter is correct or fully widened, and a
// quarter of the guard goes blind. R2 is deprecated and D2 is canceled for exactly that
// reason.

// layoutFixtureExtension returns a kind's extension or fails the test. Every
// artifact-shaped fixture name below is COMPOSED from this rather than typed, so a
// fixture cannot drift from the table the production code now reads.
func layoutFixtureExtension(t *testing.T, kind artifact.Kind) string {
	t.Helper()
	layout, ok := artifact.LayoutFor(kind)
	if !ok {
		t.Fatalf("artifact.LayoutFor(%q) reported ok=false; the fixture names cannot be composed", kind)
	}
	return layout.Extension
}

// The two real targets. R1 is `implemented` and carries everything the three
// implemented-scoped extractors demand; R2 is genuinely TERMINAL and is the only file
// CountTerminalSpecs may count.
const (
	layoutFixtureRealSpec = `---
number: SPEC-900
status: implemented
implementation:
  subject: pkg/layoutfixture
verification:
  test_command: "go test ./..."
  coverage_threshold: 90
claims:
  - id: CLM-901
    tests:
      - TestFixtureAlpha
contracts:
  - file: pkg/layoutfixture/alpha.go
    provides:
      - name: AlphaProvide
        kind: function
        signature: "func AlphaProvide() error"
---

The implemented spec. Feeds ExtractMandatedTests, ExtractSpecVerifications and
ExtractContractEntries. NOT counted by CountTerminalSpecs.
`

	// R2 deliberately carries a mandated test it must NOT contribute, which additionally
	// pins ExtractMandatedTests' terminal exclusion.
	layoutFixtureTerminalSpec = `---
number: SPEC-901
status: deprecated
claims:
  - id: CLM-902
    tests:
      - TestFixtureTerminal
---

The terminal spec. The ONLY file CountTerminalSpecs counts.
`
)

// The four decoys. Three are SUBSTANTIVE — parseable frontmatter carrying exactly the
// fields their target extractor demands — because parseSpecFrontmatter reads any file
// opening with `---` regardless of extension, which is what makes the extension filter
// the only thing standing between these and the output.
const (
	// D1 would feed THREE extractors at once if the filter widened.
	layoutFixtureDecoyPlan = `---
number: PLAN-SPEC-900
status: implemented
implementation:
  subject: pkg/layoutfixture
verification:
  test_command: "go test ./..."
  coverage_threshold: 90
claims:
  - id: CLM-903
    tests:
      - TestDecoyPlan
contracts:
  - file: pkg/layoutfixture/plan.go
    provides:
      - name: DecoyPlanProvide
        kind: function
        signature: "func DecoyPlanProvide() error"
---

Decoy 1.
`

	// D2 is TERMINAL, and it is the ONLY thing that makes CountTerminalSpecs
	// discriminate. Without it that extractor returns 0 both ways.
	layoutFixtureDecoyADR = `---
number: ADR-0900
status: canceled
---

Decoy 2.
`

	// D3 is the file SHARP EDGE 1 names by name. It must actually be parseable for that
	// scenario to be real rather than rhetorical.
	layoutFixtureDecoyReadme = `---
number: SPEC-903
status: implemented
implementation:
  subject: pkg/layoutfixture
verification:
  test_command: "go test ./..."
  coverage_threshold: 90
claims:
  - id: CLM-904
    tests:
      - TestDecoyReadme
contracts:
  - file: pkg/layoutfixture/readme.go
    provides:
      - name: DecoyReadmeProvide
        kind: function
        signature: "func DecoyReadmeProvide() error"
---

Decoy 3.
`

	// D4 is the honest control: genuinely unparseable, contributes nothing either way. It
	// proves the guard is not accidentally leaning on the unparseable-skip path for its
	// discrimination. KEEP IT, and do NOT let it stand in for D1-D3.
	layoutFixtureDecoyNotes = `just some notes, with no frontmatter at all
`
)

// writeLayoutFixtureDir builds the six-file fixture directory and returns it along with
// R1's full path.
//
// decoySuffix is appended to every DECOY name and to nothing else. Passing "" produces
// the real directory, where only the two spec-shaped names are discoverable. Passing the
// spec extension produces exactly what a WIDENED filter would see — the same six files,
// with every decoy now spec-shaped — which is how the discrimination proof below runs
// without touching production code.
func writeLayoutFixtureDir(t *testing.T, decoySuffix string) (dir string, realSpecPath string) {
	t.Helper()

	dir = t.TempDir()
	specExt := layoutFixtureExtension(t, artifact.KindSpec)

	realSpecPath = filepath.Join(dir, "SPEC-900-fixture"+specExt)
	files := map[string]string{
		realSpecPath: layoutFixtureRealSpec,
		filepath.Join(dir, "SPEC-901-terminal"+specExt): layoutFixtureTerminalSpec,

		filepath.Join(dir, "PLAN-SPEC-900-fixture"+layoutFixtureExtension(t, artifact.KindPlan)+decoySuffix): layoutFixtureDecoyPlan,
		filepath.Join(dir, "ADR-0900-fixture"+layoutFixtureExtension(t, artifact.KindADR)+decoySuffix):       layoutFixtureDecoyADR,
		filepath.Join(dir, "README.md"+decoySuffix):                                                          layoutFixtureDecoyReadme,
		filepath.Join(dir, "notes.txt"+decoySuffix):                                                          layoutFixtureDecoyNotes,
	}

	for path, body := range files {
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatalf("writing fixture %s: %v", filepath.Base(path), err)
		}
	}
	return dir, realSpecPath
}

// mandatedTestNames reduces an extraction to its sorted func names.
func mandatedTestNames(tests []MandatedTest) []string {
	names := make([]string, 0, len(tests))
	for _, test := range tests {
		names = append(names, test.FuncName)
	}
	sort.Strings(names)
	return names
}

// contractNames reduces an extraction to its sorted provide names.
func contractNames(entries []ContractEntry) []string {
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name)
	}
	sort.Strings(names)
	return names
}

// equalStrings reports whether two sorted slices hold the same values.
func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestTestVerify_SpecDiscoveryAcceptsOnlySpecFilesFromSharedLayout (ISSUE-124 CLM-003,
// CLM-006).
//
// All four extractors, over one directory holding two real specs and four decoys, return
// EXACTLY what the two real specs mandate and nothing the decoys carry.
//
// ★ EVERY ASSERTION IS AN EQUALITY OVER THE FULL RESULT, never a "contains the expected
// one" check. A superset assertion stays green under the widening mutation and re-vacuums
// the whole guard.
func TestTestVerify_SpecDiscoveryAcceptsOnlySpecFilesFromSharedLayout(t *testing.T) {
	dir, realSpecPath := writeLayoutFixtureDir(t, "")

	t.Run("ExtractMandatedTests sees only the implemented spec's tests", func(t *testing.T) {
		tests, err := ExtractMandatedTests(dir)
		if err != nil {
			t.Fatalf("ExtractMandatedTests: %v", err)
		}
		want := []string{"TestFixtureAlpha"}
		if got := mandatedTestNames(tests); !equalStrings(got, want) {
			t.Fatalf("ExtractMandatedTests returned %v, want exactly %v.\nR2 is terminal and must be excluded; D1 and D3 are not spec-shaped and must never be read. Extra names here mean the extension filter stopped discriminating — the SHARP EDGE 1 empty-extension failure.", got, want)
		}
	})

	t.Run("CountTerminalSpecs counts only the terminal spec", func(t *testing.T) {
		count, err := CountTerminalSpecs(dir)
		if err != nil {
			t.Fatalf("CountTerminalSpecs: %v", err)
		}
		if count != 1 {
			t.Fatalf("CountTerminalSpecs returned %d, want exactly 1.\nOnly R2 (deprecated) is a terminal SPEC. D2 (canceled) is an ADR and must not be reached; a 2 here means the filter widened.", count)
		}
	})

	t.Run("ExtractSpecVerifications sees only the implemented spec", func(t *testing.T) {
		verifications, err := ExtractSpecVerifications(dir)
		if err != nil {
			t.Fatalf("ExtractSpecVerifications: %v", err)
		}
		if len(verifications) != 1 {
			t.Fatalf("ExtractSpecVerifications returned %d entries, want exactly 1.\nD1 and D3 both carry a test_command and a coverage_threshold at status implemented, so any extra entry means they were read.", len(verifications))
		}
		if verifications[0].File != realSpecPath {
			t.Fatalf("ExtractSpecVerifications returned the entry for %s, want %s", verifications[0].File, realSpecPath)
		}
	})

	t.Run("ExtractContractEntries sees only the implemented spec's provides", func(t *testing.T) {
		entries, err := ExtractContractEntries(dir, dir)
		if err != nil {
			t.Fatalf("ExtractContractEntries: %v", err)
		}
		want := []string{"AlphaProvide"}
		if got := contractNames(entries); !equalStrings(got, want) {
			t.Fatalf("ExtractContractEntries returned %v, want exactly %v.\nD1 and D3 both declare contracts.provides at status implemented; either name appearing here means the filter widened.", got, want)
		}
	})
}

// TestTestVerify_DecoyFixturesWouldChangeEveryExtractorIfWidened (ISSUE-124 CLM-003).
//
// ★ THE PROOF THAT THE GUARD ABOVE IS NOT DECORATIVE, and it runs on every gate rather
// than once in a reviewer's terminal.
//
// A guard nobody has watched go red is a guard nobody has confirmed is wired to
// anything. This one is the plan's ONLY defense against its worst failure mode, so its
// discriminating power is proven HERE — without touching production code — by building
// the same six fixtures with every decoy given a spec-shaped name. That directory is
// precisely what a widened filter would see, and all four extractors must return
// something DIFFERENT over it.
//
// If any of the four comes back identical, that decoy is INERT and the corresponding
// quarter of CLM-003 is vacuous: the fix is to make the fixture substantive again, never
// to relax the assertion.
func TestTestVerify_DecoyFixturesWouldChangeEveryExtractorIfWidened(t *testing.T) {
	widened, _ := writeLayoutFixtureDir(t, layoutFixtureExtension(t, artifact.KindSpec))

	t.Run("ExtractMandatedTests would gain the decoys' tests", func(t *testing.T) {
		tests, err := ExtractMandatedTests(widened)
		if err != nil {
			t.Fatalf("ExtractMandatedTests: %v", err)
		}
		got := mandatedTestNames(tests)
		if equalStrings(got, []string{"TestFixtureAlpha"}) {
			t.Fatal("ExtractMandatedTests returned the SAME result over the widened directory, so D1 and D3 are inert as mandated-test decoys. That quarter of CLM-003 is vacuous: the SHARP EDGE 1 mutation would pass it silently. Make the decoys substantive rather than relaxing this.")
		}
	})

	t.Run("CountTerminalSpecs would count the terminal decoy too", func(t *testing.T) {
		count, err := CountTerminalSpecs(widened)
		if err != nil {
			t.Fatalf("CountTerminalSpecs: %v", err)
		}
		if count != 2 {
			t.Fatalf("CountTerminalSpecs returned %d over the widened directory, want 2 (R2 plus the canceled ADR decoy).\nIf it returned 1, D2 is inert — most likely because its status is not genuinely terminal — and CountTerminalSpecs' quarter of CLM-003 checks nothing.", count)
		}
	})

	t.Run("ExtractSpecVerifications would gain the decoys' entries", func(t *testing.T) {
		verifications, err := ExtractSpecVerifications(widened)
		if err != nil {
			t.Fatalf("ExtractSpecVerifications: %v", err)
		}
		if len(verifications) <= 1 {
			t.Fatalf("ExtractSpecVerifications returned %d entries over the widened directory, want more than 1.\nD1 and D3 are inert as verification decoys — check that each carries BOTH a test_command and a positive coverage_threshold at status implemented.", len(verifications))
		}
	})

	t.Run("ExtractContractEntries would gain the decoys' provides", func(t *testing.T) {
		entries, err := ExtractContractEntries(widened, widened)
		if err != nil {
			t.Fatalf("ExtractContractEntries: %v", err)
		}
		if len(entries) <= 1 {
			t.Fatalf("ExtractContractEntries returned %d entries over the widened directory, want more than 1.\nD1 and D3 are inert as contract decoys — check that each declares contracts.provides at status implemented.", len(entries))
		}
	})
}
