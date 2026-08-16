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
// THE EXCLUSION SET ARRIVES AS A PARAMETER, AND THE `.backstop` RULES DO NOT.
// nonCorpus is the TOOL-AGNOSTIC BASE core carries (artifact.NonCorpusDirNames) unioned
// with the ecosystem-specific dependency directory names installed packs declare via
// classification.dependency_dirs. Core holds no ecosystem noun of its own: `vendor` is a
// Go noun and `node_modules` a Node/npm one, and a thin executor that bakes no language
// or tool knowledge cannot carry either (ISSUE-122, resolving SPEC-068's Sharp Edge 12).
// Its zero value still excludes the tool-agnostic base, so a caller that does not wire
// the installed packs degrades to today-minus-declarations rather than walking `.git`.
//
// THE `.backstop` SKIP RULES REMAIN ROOT-RELATIVE AND LOCAL TO THIS WALK.
// `.backstop/packs` ALWAYS skips, wherever the root sits: several installed packs are
// themselves backstop repos carrying their own artifacts, and those are never part of
// the consumer's corpus. `.backstop` itself skips ONLY when it is not the root — a
// project that roots its artifacts there must be able to reach them.
func DiscoverArtifacts(root artifact.Root, typeFilters []string, nonCorpus artifact.NonCorpusDirs) ([]DiscoveredArtifact, error) {
	filterSet := make(map[string]bool)
	for _, t := range typeFilters {
		filterSet[t] = true
	}

	var artifacts []DiscoveredArtifact

	err := filepath.Walk(root.Path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			base := filepath.Base(path)
			if nonCorpus.Excludes(base) {
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
