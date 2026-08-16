package artifact

import (
	"sort"
	"strings"
	"testing"
)

// TestLayout_EverySevenKindsResolveDirectoryAndExtension pins CLM-038: every one of
// the seven kinds the codebase already recognizes resolves to a directory and an
// extension from the ONE layout table. The assertion is DRIVEN FROM Kinds() rather
// than a hand-written list — ranging over a local list would test the list, not the
// table, which is what the Kinds() contract note says this claim exists to prevent.
func TestLayout_EverySevenKindsResolveDirectoryAndExtension(t *testing.T) {
	kinds := Kinds()
	if len(kinds) != 7 {
		t.Fatalf("Kinds() returned %d kinds, want exactly 7: %v", len(kinds), kinds)
	}

	got := make([]string, 0, len(kinds))
	for _, k := range kinds {
		got = append(got, string(k))
	}
	sort.Strings(got)

	want := []string{"adr", "bundle", "capability", "directive", "issue", "plan", "spec"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Kinds() set = %v, want %v", got, want)
		}
	}

	for _, k := range kinds {
		layout, ok := LayoutFor(k)
		if !ok {
			t.Errorf("LayoutFor(%q) returned ok=false; every enumerated kind must resolve", k)
			continue
		}
		if layout.Directory == "" {
			t.Errorf("LayoutFor(%q).Directory is empty; a zero directory would resolve to the root itself", k)
		}
		if layout.Extension == "" {
			t.Errorf("LayoutFor(%q).Extension is empty", k)
		}
	}
}

// kindsMatching returns every layout-table entry whose OWN pattern matches name,
// independently of table order.
//
// It applies each row's rule directly rather than asking ClassifyFilename, because
// ClassifyFilename returns the first match and therefore cannot reveal a second one.
// The rule it applies is the same one ClassifyFilename applies — suffix, plus the ADR
// prefix requirement — and it is deliberately the ONLY duplication of that logic in the
// suite: it exists so an overlap between two rows is observable at all.
func kindsMatching(name string) []Kind {
	var out []Kind
	for _, e := range kindTable {
		if !strings.HasSuffix(name, e.extension) {
			continue
		}
		if e.kind == KindADR && !strings.HasPrefix(name, "ADR-") {
			continue
		}
		out = append(out, e.kind)
	}
	return out
}

// TestLayout_ExtensionsCannotShadowOneAnother is the whole-class version of the
// exclusivity property: it holds for every possible filename, not only the sampled ones.
//
// Two rows collide when one extension is a SUFFIX of another (".md" would shadow
// ".spec.md"; ".spec.md" would shadow ".my.spec.md"), because then any filename ending
// in the longer one also ends in the shorter. Sampling cannot prove the absence of such
// a pair — a new kind added with a too-general extension would pass every sampled case
// while silently capturing another kind's files.
func TestLayout_ExtensionsCannotShadowOneAnother(t *testing.T) {
	for _, a := range kindTable {
		for _, b := range kindTable {
			if a.kind == b.kind {
				continue
			}
			if !strings.HasSuffix(a.extension, b.extension) {
				continue
			}
			// The ADR row is allowed to be shadowed on extension alone because its
			// prefix requirement disambiguates it; every other pair is a real defect.
			if a.kind == KindADR || b.kind == KindADR {
				continue
			}
			t.Errorf("kind %q's extension %q ends with kind %q's extension %q, so every %s filename is also matched by %s and classification depends on table order",
				a.kind, a.extension, b.kind, b.extension, a.kind, b.kind)
		}
	}

	// Every extension is non-empty, since an empty one matches EVERY filename.
	for _, e := range kindTable {
		if e.extension == "" {
			t.Errorf("kind %q declares an empty extension, which matches every filename", e.kind)
		}
	}
}

// TestLayout_FilenameClassificationIsExclusiveAndRejectsNonArtifacts pins CLM-039:
// classification assigns AT MOST one kind per filename and no kind to a non-artifact
// filename. Exclusivity is asserted against the layout TABLE — every row's own pattern
// is applied to each filename and exactly one may match — rather than by re-asking
// ClassifyFilename, whose first-match-wins answer is precisely what hides an overlap.
func TestLayout_FilenameClassificationIsExclusiveAndRejectsNonArtifacts(t *testing.T) {
	cases := map[string]Kind{
		"SPEC-068-trustworthy-green-guards.spec.md":       KindSpec,
		"PLAN-SPEC-068-trustworthy-green-guards.plan.yml": KindPlan,
		"ADR-0001-some-decision.adr.md":                   KindADR,
		"BUNDLE-003-onboarding-experience.bundle.md":      KindBundle,
		"ISSUE-121-some-defect.issue.md":                  KindIssue,
		"DIR-002-some-theme.directive.md":                 KindDirective,
		"CAP-001-pack-gate-enforcement.capability.yml":    KindCapability,
	}

	for name, wantKind := range cases {
		gotKind, ok := ClassifyFilename(name)
		if !ok {
			t.Errorf("ClassifyFilename(%q) returned ok=false, want kind %q", name, wantKind)
			continue
		}
		if gotKind != wantKind {
			t.Errorf("ClassifyFilename(%q) = %q, want %q", name, gotKind, wantKind)
		}

		// Exclusivity is asserted against the TABLE, not against ClassifyFilename.
		//
		// Polling ClassifyFilename once per kind cannot discharge this claim: the
		// function returns the FIRST entry whose pattern matches, so a genuine overlap
		// between two kinds is exactly what it HIDES. Counting its answer can only ever
		// yield 0 or 1 no matter how broken the table is. So this counts the table rows
		// whose own pattern matches, which is the thing that can actually be wrong.
		claimants := kindsMatching(name)
		if len(claimants) != 1 {
			t.Errorf("filename %q is matched by %d layout entries %v, want exactly 1 — two kinds whose patterns both match one real filename make ClassifyFilename's answer depend on table ORDER", name, len(claimants), claimants)
		}
	}

	// And the cross product: no kind's sample filename may be claimed by any OTHER
	// kind's pattern. This is the check the old per-name poll could not make.
	for name, wantKind := range cases {
		for _, claimant := range kindsMatching(name) {
			if claimant != wantKind {
				t.Errorf("filename %q (a %s) is ALSO matched by the %s layout entry", name, wantKind, claimant)
			}
		}
	}

	// Non-artifact filenames classify to NO kind. "capability.yml" is the trap: the
	// repo's own capabilities/CAP-001-pack-gate-enforcement/capability.yml is named
	// exactly that and is nested a level deeper than the layout expects. It escapes
	// the `.capability.yml` suffix test today and must keep escaping it — widening the
	// pattern to also match the bare name turns this repo's own gate red.
	nonArtifacts := []string{
		"README.md",
		"backstop.yml",
		"schema.json",
		"capability.yml",
		"layout.go",
		"CHANGELOG.md",
		"pack.yml",
	}
	for _, name := range nonArtifacts {
		if k, ok := ClassifyFilename(name); ok {
			t.Errorf("ClassifyFilename(%q) = (%q, true), want ok=false — it is not an artifact filename", name, k)
		}
	}

	// The ADR case additionally requires the ADR- prefix, preserving the behavior of
	// the artifactPatterns map this replaces: a bare `.adr.md` suffix is not enough.
	if k, ok := ClassifyFilename("notes.adr.md"); ok {
		t.Errorf("ClassifyFilename(\"notes.adr.md\") = (%q, true), want ok=false — the ADR- prefix is required", k)
	}

	// An unrecognized kind returns ok=false, never a zero-value KindLayout. A zero
	// Directory would silently resolve to the root itself.
	layout, ok := LayoutFor(Kind("not-a-kind"))
	if ok {
		t.Errorf("LayoutFor(\"not-a-kind\") returned ok=true with layout %+v, want ok=false", layout)
	}
	if layout.Directory != "" || layout.Extension != "" {
		t.Errorf("LayoutFor on an unrecognized kind returned a populated layout %+v, want the zero value alongside ok=false", layout)
	}

	// Dir on an unrecognized kind yields no directory rather than silently returning
	// the root itself, which is the same failure a zero-value KindLayout would cause.
	if dir := (Root{Path: "/tmp/project"}).Dir(Kind("not-a-kind")); dir != "" {
		t.Errorf("Root.Dir on an unrecognized kind = %q, want empty", dir)
	}
}

// TestLayout_NonCorpusDirNamesAreSharedAndExcludeDotBackstop pins the shared
// non-corpus directory list Sharp Edge 12 requires to have exactly ONE home. The
// ABSENCE of ".backstop" is the load-bearing half: CLI discovery skips `.backstop`
// wholesale except when it IS the root, while the ungated scan always walks it and
// excludes only `.backstop/packs` beneath it. Baking `.backstop` into the shared list
// would make the ungated scan report nothing in the UNCONFIGURED case, which is
// precisely its motivating case.
func TestLayout_NonCorpusDirNamesAreSharedAndExcludeDotBackstop(t *testing.T) {
	got := NonCorpusDirNames()
	want := []string{".git", "testdata", "prototype"}

	if len(got) != len(want) {
		t.Fatalf("NonCorpusDirNames() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("NonCorpusDirNames() = %v, want %v", got, want)
		}
	}

	for _, name := range got {
		if name == ".backstop" {
			t.Error("NonCorpusDirNames() contains \".backstop\"; each caller must add its own .backstop rule, or the unconfigured motivating case is excluded away")
		}
	}

	// The accessor returns a copy, so a caller cannot mutate the shared list out from
	// under the other caller.
	got[0] = "mutated"
	if NonCorpusDirNames()[0] != ".git" {
		t.Error("mutating the returned slice changed the shared list; NonCorpusDirNames must return a copy")
	}
}
