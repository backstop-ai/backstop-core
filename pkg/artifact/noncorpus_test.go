package artifact

import "testing"

// TestNonCorpusDirNames_ContainsNoEcosystemNoun (ISSUE-122 CLM-001): the shared
// list core carries holds ONLY tool-agnostic names. `vendor` is a Go noun and
// `node_modules` is a Node/npm noun; a thin executor that bakes no language or
// tool knowledge cannot carry either. They arrive injected from pack
// declarations instead (classification.dependency_dirs).
func TestNonCorpusDirNames_ContainsNoEcosystemNoun(t *testing.T) {
	got := NonCorpusDirNames()
	want := []string{".git", "testdata", "prototype"}

	if len(got) != len(want) {
		t.Fatalf("NonCorpusDirNames() = %v, want exactly the tool-agnostic names %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("NonCorpusDirNames() = %v, want exactly the tool-agnostic names %v", got, want)
		}
	}

	for _, name := range got {
		if name == "vendor" || name == "node_modules" {
			t.Errorf("NonCorpusDirNames() carries the ecosystem noun %q; core bakes no language or tool knowledge — it must arrive from a pack's classification.dependency_dirs", name)
		}
	}
}

// TestNewNonCorpusDirs_UnionsGenericBaseWithPackDeclared (ISSUE-122 CLM-004):
// the injected set is the generic base UNIONED with the pack-declared names. It
// deduplicates an injected name that duplicates a base name and tolerates an
// empty or nil injection.
func TestNewNonCorpusDirs_UnionsGenericBaseWithPackDeclared(t *testing.T) {
	t.Run("excludes both the base and the injected names", func(t *testing.T) {
		set := NewNonCorpusDirs([]string{"vendor", "node_modules", "_thirdparty"})

		for _, name := range []string{".git", "testdata", "prototype"} {
			if !set.Excludes(name) {
				t.Errorf("Excludes(%q) = false, want true (tool-agnostic base)", name)
			}
		}
		for _, name := range []string{"vendor", "node_modules", "_thirdparty"} {
			if !set.Excludes(name) {
				t.Errorf("Excludes(%q) = false, want true (pack-declared)", name)
			}
		}
		for _, name := range []string{"specs", "pkg", ".backstop", "", "vendored"} {
			if set.Excludes(name) {
				t.Errorf("Excludes(%q) = true, want false — the set must exclude only the base plus the declarations", name)
			}
		}
	})

	t.Run("an injected name duplicating a base name is harmless", func(t *testing.T) {
		set := NewNonCorpusDirs([]string{"testdata", ".git"})
		for _, name := range []string{".git", "testdata", "prototype"} {
			if !set.Excludes(name) {
				t.Errorf("Excludes(%q) = false, want true", name)
			}
		}
		if set.Excludes("vendor") {
			t.Error("Excludes(\"vendor\") = true with no declaration of it; the noun must come from a pack")
		}
	})

	t.Run("nil and empty injections yield the base", func(t *testing.T) {
		for name, set := range map[string]NonCorpusDirs{
			"nil":             NewNonCorpusDirs(nil),
			"empty":           NewNonCorpusDirs([]string{}),
			"empty name only": NewNonCorpusDirs([]string{""}),
		} {
			t.Run(name, func(t *testing.T) {
				if !set.Excludes(".git") {
					t.Error("Excludes(\".git\") = false, want true")
				}
				if set.Excludes("") {
					t.Error("Excludes(\"\") = true; an empty declared name must be ignored, not matched")
				}
				if set.Excludes("vendor") {
					t.Error("Excludes(\"vendor\") = true with no declaration; that would be the bake ISSUE-122 removes")
				}
			})
		}
	})
}

// TestNonCorpusDirs_ZeroValueFallsBackToGenericBase (ISSUE-122 CLM-004): the
// ZERO VALUE — what a caller that forgets to wire the packs produces — excludes
// the generic base rather than nothing. A zero value excluding nothing would
// send the walk into `.git`, and the resulting garbage would be a silent,
// confusing failure at a call site the author forgot. Degrading to
// "today's behavior minus the pack-declared names" is the honest failure.
func TestNonCorpusDirs_ZeroValueFallsBackToGenericBase(t *testing.T) {
	var zero NonCorpusDirs

	for _, name := range []string{".git", "testdata", "prototype"} {
		if !zero.Excludes(name) {
			t.Errorf("zero-value Excludes(%q) = false, want true — a mis-wired walk must not descend into %s", name, name)
		}
	}
	for _, name := range []string{"vendor", "node_modules", "specs", ".backstop", ""} {
		if zero.Excludes(name) {
			t.Errorf("zero-value Excludes(%q) = true, want false — the zero value is the generic base and NOTHING else", name)
		}
	}
}
