package main

import "github.com/backstop-ai/backstop-core/pkg/pack"

// mergeDependencyDirs unions the classification.dependency_dirs names across the
// FULL installed-pack manifest set into one deduplicated list (ISSUE-122
// CLM-003). It mirrors mergeSourceClassifier's contract exactly: it takes the
// wholesale []*pack.Manifest set loadInstalledPacks resolves over
// `backstop.yml packs:` with NO toolchain-only pre-filter — a manifest with no
// `classification:` block contributes zero to the union, so no filter is needed
// or wanted — skips nil manifests, and carries no pack-name literal and no
// language branch.
//
// Zero packs yields an EMPTY list, never a defaulted one: defaulting
// `vendor`/`node_modules` here would re-bake the ecosystem nouns one layer down
// from where ISSUE-122 removed them from pkg/artifact.
//
// First-seen order is preserved so the merged list is deterministic across runs
// without a separate sort that would decouple it from declaration order.
func mergeDependencyDirs(packs []*pack.Manifest) []string {
	var merged []string
	seen := make(map[string]bool)
	for _, manifest := range packs {
		if manifest == nil {
			continue
		}
		for _, name := range manifest.Classification.DependencyDirs {
			if name == "" || seen[name] {
				continue
			}
			seen[name] = true
			merged = append(merged, name)
		}
	}
	return merged
}
