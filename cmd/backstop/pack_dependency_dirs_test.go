package main

import (
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/pack"
)

// TestMergeDependencyDirs_UnionsAcrossPacksDedupedAndNilSafe (ISSUE-122
// CLM-003): the merge unions classification.dependency_dirs across the FULL
// installed-pack set, deduplicated and first-seen-ordered, skipping nil
// manifests. It mirrors mergeSourceClassifier's contract — wholesale pack set,
// no toolchain pre-filter, no language branch.
func TestMergeDependencyDirs_UnionsAcrossPacksDedupedAndNilSafe(t *testing.T) {
	packs := []*pack.Manifest{
		{Name: "backstop-ai/go-toolchain", Classification: pack.Classification{DependencyDirs: []string{"vendor", "_thirdparty"}}},
		nil,
		{Name: "backstop-ai/no-declaration"},
		{Name: "backstop-ai/ts-toolchain", Classification: pack.Classification{DependencyDirs: []string{"node_modules", "vendor"}}},
	}

	got := mergeDependencyDirs(packs)

	want := []string{"vendor", "_thirdparty", "node_modules"}
	if len(got) != len(want) {
		t.Fatalf("merged len = %d, want %d (%#v)", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("merged[%d] = %q, want %q (deduplicated, first-seen order)", i, got[i], w)
		}
	}
}

// TestMergeDependencyDirs_ZeroPacksYieldsEmpty (ISSUE-122 CLM-003): no
// installed packs yields an EMPTY result, never a defaulted list. A default
// here would re-bake `vendor`/`node_modules` one layer down from where
// ISSUE-122 removed them, which is the exact failure this change exists to
// prevent.
func TestMergeDependencyDirs_ZeroPacksYieldsEmpty(t *testing.T) {
	for name, packs := range map[string][]*pack.Manifest{
		"nil slice":           nil,
		"empty slice":         {},
		"only a nil manifest": {nil},
		"pack declaring none": {{Name: "backstop-ai/self"}},
	} {
		t.Run(name, func(t *testing.T) {
			if got := mergeDependencyDirs(packs); len(got) != 0 {
				t.Errorf("mergeDependencyDirs must yield no defaulted names, got %#v", got)
			}
		})
	}
}
