package main

import (
	"os"
	"path/filepath"
	"strings"
)

// DiscoveredArtifact represents a file identified as a backstop artifact
// during project directory scanning.
type DiscoveredArtifact struct {
	Path string // Full file path
	Type string // Artifact type: spec, plan, adr, bundle, issue, standard
}

// artifactPatterns maps artifact types to their filename matching functions.
// Each function returns true if the filename matches the pattern for that type.
var artifactPatterns = map[string]func(string) bool{
	"spec":     func(name string) bool { return strings.HasSuffix(name, ".spec.md") },
	"plan":     func(name string) bool { return strings.HasSuffix(name, ".plan.yml") },
	"adr":      func(name string) bool { return strings.HasPrefix(name, "ADR-") && strings.HasSuffix(name, ".adr.md") },
	"bundle":   func(name string) bool { return strings.HasSuffix(name, ".bundle.md") },
	"issue":    func(name string) bool { return strings.HasSuffix(name, ".issue.md") },
	"standard":  func(name string) bool { return strings.HasSuffix(name, ".standard.md") },
	"directive": func(name string) bool { return strings.HasSuffix(name, ".directive.md") },
}

// DiscoverArtifacts scans the directory tree rooted at root for files matching
// known artifact filename patterns. If typeFilters is non-empty, only artifacts
// of the specified types are returned. Files not matching any pattern are
// silently ignored.
func DiscoverArtifacts(root string, typeFilters []string) ([]DiscoveredArtifact, error) {
	filterSet := make(map[string]bool)
	for _, t := range typeFilters {
		filterSet[t] = true
	}

	var artifacts []DiscoveredArtifact

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			// Skip directories that should not be scanned for artifacts
			base := filepath.Base(path)
			switch base {
			case "testdata", "vendor", "node_modules", ".git", ".backstop", "prototype":
				return filepath.SkipDir
			}
			return nil
		}

		name := info.Name()
		for artType, matcher := range artifactPatterns {
			if !matcher(name) {
				continue
			}
			// If type filters are set, skip non-matching types
			if len(filterSet) > 0 && !filterSet[artType] {
				continue
			}
			artifacts = append(artifacts, DiscoveredArtifact{
				Path: path,
				Type: artType,
			})
			break // A file matches at most one type
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return artifacts, nil
}
