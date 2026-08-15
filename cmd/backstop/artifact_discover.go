package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/backstop-ai/backstop-core/pkg/artifact"
)

// DiscoveredArtifact represents a file identified as a backstop artifact
// during project directory scanning.
type DiscoveredArtifact struct {
	Path string // Full file path
	Type string // Artifact type: spec, plan, adr, bundle, issue, directive, capability
}

// DiscoverArtifacts scans the directory tree rooted at the RESOLVED ARTIFACT ROOT for
// files whose names declare an artifact kind. If typeFilters is non-empty, only
// artifacts of the specified types are returned. Files not matching any pattern are
// silently ignored.
//
// The parameter is an artifact.Root rather than a bare string so a caller cannot hand
// in the project root where the artifact root belongs — under a `.backstop/` layout the
// two are genuinely different directories, and passing the wrong one discovers a
// different corpus while still returning cleanly.
//
// Classification is artifact.ClassifyFilename — the SAME predicate the REQ-008 ungated
// scan uses — so the set of files the gate picks up and the set it reports leaving out
// are defined by one rule rather than two that can drift.
//
// THE SKIP RULE IS ROOT-RELATIVE. The shared non-corpus names still skip anywhere.
// `.backstop/packs` ALWAYS skips, wherever the root sits: several installed packs are
// themselves backstop repos carrying their own artifacts, and those are never part of
// the consumer's corpus. `.backstop` itself skips ONLY when it is not the root — a
// project that roots its artifacts there must be able to reach them.
//
// Sharp Edge 12, knowingly propagated rather than fixed here: `vendor` and
// `node_modules` are language nouns, and their presence in a shared list inside core is
// a real zero-baked-language defect tracked on its own issue. This function centralizes
// the list; it does not put that invariant on this seed's critical path.
func DiscoverArtifacts(root artifact.Root, typeFilters []string) ([]DiscoveredArtifact, error) {
	filterSet := make(map[string]bool)
	for _, t := range typeFilters {
		filterSet[t] = true
	}

	nonCorpus := make(map[string]bool)
	for _, name := range artifact.NonCorpusDirNames() {
		nonCorpus[name] = true
	}

	var artifacts []DiscoveredArtifact

	err := filepath.Walk(root.Path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			base := filepath.Base(path)
			if nonCorpus[base] {
				return filepath.SkipDir
			}
			// An installed-pack tree is never corpus, wherever the root sits. Keying on
			// the PARENT covers both layouts with one rule: `.backstop/packs` beneath a
			// repo root, and `packs` when `.backstop` IS the root.
			if base == "packs" && filepath.Base(filepath.Dir(path)) == ".backstop" {
				return filepath.SkipDir
			}
			// `.backstop` is not corpus UNLESS it is the root itself.
			if base == ".backstop" && path != root.Path {
				return filepath.SkipDir
			}
			return nil
		}

		kind, ok := artifact.ClassifyFilename(info.Name())
		if !ok {
			return nil
		}
		if len(filterSet) > 0 && !filterSet[string(kind)] {
			return nil
		}
		artifacts = append(artifacts, DiscoveredArtifact{Path: path, Type: string(kind)})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking artifact root %s: %w", root.Path, err)
	}

	return artifacts, nil
}
