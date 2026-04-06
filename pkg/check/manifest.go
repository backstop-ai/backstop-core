package check

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CheckType represents a validation pass type.
type CheckType int

const (
	// CheckTypeLint runs golangci-lint.
	CheckTypeLint CheckType = iota
	// CheckTypeBuild runs go build.
	CheckTypeBuild
	// CheckTypeTest runs go test.
	CheckTypeTest
	// CheckTypeSemgrep runs semgrep rules.
	CheckTypeSemgrep
)

// String returns the string representation of a CheckType.
func (ct CheckType) String() string {
	switch ct {
	case CheckTypeLint:
		return "lint"
	case CheckTypeBuild:
		return "build"
	case CheckTypeTest:
		return "test"
	case CheckTypeSemgrep:
		return "semgrep"
	default:
		return fmt.Sprintf("unknown(%d)", int(ct))
	}
}

// parseCheckType converts a string to a CheckType.
func parseCheckType(s string) (CheckType, bool) {
	switch strings.ToLower(s) {
	case "lint":
		return CheckTypeLint, true
	case "build":
		return CheckTypeBuild, true
	case "test":
		return CheckTypeTest, true
	case "semgrep":
		return CheckTypeSemgrep, true
	default:
		return 0, false
	}
}

// ManifestRule maps file extensions and path patterns to check types.
type ManifestRule struct {
	Extensions   []string    `json:"extensions,omitempty"`
	PathPatterns []string    `json:"path_patterns,omitempty"`
	CheckTypes   []string    `json:"check_types"`
	parsed       []CheckType // lazily populated
}

// manifestFile is the JSON structure of a .manifest.json file.
type manifestFile struct {
	Rules []ManifestRule `json:"rules"`
}

// Manifest holds compiled enforcement rules for file-type routing.
type Manifest struct {
	rules      []ManifestRule
	isDefaults bool
}

// LoadManifest reads compiled enforcement manifests from dir.
// If no .manifest.json files exist, returns a Manifest with built-in defaults.
func LoadManifest(dir string) (*Manifest, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		// Directory doesn't exist or can't be read — use defaults
		return defaultManifest(), nil
	}

	var allRules []ManifestRule
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".manifest.json") {
			continue
		}
		data, readErr := os.ReadFile(filepath.Join(dir, entry.Name()))
		if readErr != nil {
			return nil, fmt.Errorf("reading manifest %s: %w", entry.Name(), readErr)
		}
		var mf manifestFile
		if jsonErr := json.Unmarshal(data, &mf); jsonErr != nil {
			return nil, fmt.Errorf("parsing manifest %s: %w", entry.Name(), jsonErr)
		}
		allRules = append(allRules, mf.Rules...)
	}

	if len(allRules) == 0 {
		return defaultManifest(), nil
	}

	// Parse check types for all rules
	for i := range allRules {
		allRules[i].parsed = parseCheckTypes(allRules[i].CheckTypes)
	}

	return &Manifest{rules: allRules}, nil
}

// RouteFile returns the applicable check types for a given file path.
// Files matching no rules return an empty slice.
func (m *Manifest) RouteFile(path string) []CheckType {
	if m.isDefaults {
		return m.routeFileDefaults(path)
	}

	// Check each rule for a match
	for _, rule := range m.rules {
		if matchesRule(path, rule) {
			return rule.parsed
		}
	}

	return nil
}

// routeFileDefaults applies built-in default routing.
func (m *Manifest) routeFileDefaults(path string) []CheckType {
	ext := filepath.Ext(path)
	if ext == ".go" {
		return []CheckType{CheckTypeLint, CheckTypeBuild, CheckTypeTest, CheckTypeSemgrep}
	}
	// All other files get semgrep only
	return []CheckType{CheckTypeSemgrep}
}

// matchesRule checks if a file path matches a manifest rule by extension or
// path pattern.
func matchesRule(path string, rule ManifestRule) bool {
	// Check extensions
	ext := filepath.Ext(path)
	for _, ruleExt := range rule.Extensions {
		if ext == ruleExt {
			return true
		}
	}

	// Check path patterns
	for _, pattern := range rule.PathPatterns {
		if matchGlobPattern(path, pattern) {
			return true
		}
	}

	return false
}

// matchGlobPattern matches a file path against a glob pattern.
// Supports ** for recursive directory matching and * for single segment.
func matchGlobPattern(path, pattern string) bool {
	// Convert ** glob to a simpler matching approach
	// Split both path and pattern by /
	if strings.Contains(pattern, "**") {
		return matchDoubleStarPattern(path, pattern)
	}

	matched, err := filepath.Match(pattern, path)
	if err != nil {
		return false
	}
	return matched
}

// matchDoubleStarPattern handles ** in glob patterns.
func matchDoubleStarPattern(path, pattern string) bool {
	// Split pattern around **
	parts := strings.SplitN(pattern, "**", 2)
	if len(parts) != 2 {
		return false
	}

	prefix := parts[0]
	suffix := strings.TrimPrefix(parts[1], "/")

	// Path must start with prefix
	if prefix != "" && !strings.HasPrefix(path, prefix) {
		return false
	}

	// The remaining path after prefix must end with a match against suffix
	remaining := strings.TrimPrefix(path, prefix)

	// If suffix is empty, everything matches
	if suffix == "" {
		return true
	}

	// Try matching suffix against each possible tail of remaining
	segments := strings.Split(remaining, "/")
	for i := range segments {
		tail := strings.Join(segments[i:], "/")
		matched, err := filepath.Match(suffix, tail)
		if err == nil && matched {
			return true
		}
	}

	return false
}

// defaultManifest returns the built-in default manifest.
func defaultManifest() *Manifest {
	return &Manifest{isDefaults: true}
}

// parseCheckTypes converts string check type names to CheckType values.
func parseCheckTypes(names []string) []CheckType {
	var types []CheckType
	for _, name := range names {
		if ct, ok := parseCheckType(name); ok {
			types = append(types, ct)
		}
	}
	return types
}
