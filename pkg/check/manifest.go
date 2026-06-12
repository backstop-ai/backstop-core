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

// languageExtensions maps a compiled-manifest language to the file extension
// its native toolchain passes (lint/build/test) apply to. Deliberately minimal
// (go → .go); extension-mapping breadth is out of scope for this fix.
var languageExtensions = map[string]string{
	"go": ".go",
}

// combinedRule decodes a single entry of the top-level "rules" array, capturing
// BOTH the legacy routing fields (extensions/path_patterns/check_types) and the
// compiled-schema enforcement field. Which fields are populated depends on the
// file's schema; the discriminator is the top-level standard/language, not the
// rule shape.
type combinedRule struct {
	ManifestRule
	Enforcement string `json:"enforcement"`
}

// compiledManifestFile captures BOTH the compiled standards schema and the
// legacy routing schema so a single decode can discriminate between them. A
// file is the compiled schema when it carries a non-empty top-level Standard
// AND Language; otherwise its Rules are interpreted as legacy routing rules.
type compiledManifestFile struct {
	// Compiled-schema fields.
	Standard      string `json:"standard"`
	Language      string `json:"language"`
	SemgrepConfig string `json:"semgrep_config"`

	// Rules decodes the shared "rules" array. For a compiled file each entry's
	// Enforcement drives semgrep derivation; for a legacy file each entry's
	// embedded ManifestRule carries the routing fields.
	Rules []combinedRule `json:"rules"`
}

// isCompiled reports whether the file carries the compiled standards schema:
// a non-empty top-level standard AND language. The legacy routing schema has
// neither.
func (f *compiledManifestFile) isCompiled() bool {
	return f.Standard != "" && f.Language != ""
}

// hasSemgrepSignal reports whether a compiled manifest should route semgrep:
// any rule has enforcement "semgrep", or a semgrep_config is set.
func (f *compiledManifestFile) hasSemgrepSignal() bool {
	if f.SemgrepConfig != "" {
		return true
	}
	for _, r := range f.Rules {
		if r.Enforcement == "semgrep" {
			return true
		}
	}
	return false
}

// legacyRules returns the routing-schema rules carried by a non-compiled
// manifest, with their check types parsed.
func (f *compiledManifestFile) legacyRules() []ManifestRule {
	rules := make([]ManifestRule, 0, len(f.Rules))
	for _, r := range f.Rules {
		rule := r.ManifestRule
		rule.parsed = parseCheckTypes(rule.CheckTypes)
		rules = append(rules, rule)
	}
	return rules
}

// deriveRules turns a compiled manifest into routing rules. A known language
// (go) routes its extension to lint/build/test, plus semgrep when a semgrep
// signal is present. An unknown language with a semgrep signal routes any file
// to semgrep-only via the "**" path pattern; an unknown language with no
// semgrep signal derives nothing.
func (f *compiledManifestFile) deriveRules() []ManifestRule {
	ext, known := languageExtensions[f.Language]
	if known {
		checks := []CheckType{CheckTypeLint, CheckTypeBuild, CheckTypeTest}
		if f.hasSemgrepSignal() {
			checks = append(checks, CheckTypeSemgrep)
		}
		return []ManifestRule{{
			Extensions: []string{ext},
			parsed:     checks,
		}}
	}

	if f.hasSemgrepSignal() {
		return []ManifestRule{{
			PathPatterns: []string{"**"},
			parsed:       []CheckType{CheckTypeSemgrep},
		}}
	}

	return nil
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
	manifestFilesPresent := false
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".manifest.json") {
			continue
		}
		manifestFilesPresent = true
		data, readErr := os.ReadFile(filepath.Join(dir, entry.Name()))
		if readErr != nil {
			return nil, fmt.Errorf("reading manifest %s: %w", entry.Name(), readErr)
		}
		var mf compiledManifestFile
		if jsonErr := json.Unmarshal(data, &mf); jsonErr != nil {
			return nil, fmt.Errorf("parsing manifest %s: %w", entry.Name(), jsonErr)
		}
		if mf.isCompiled() {
			// Compiled standards manifest: derive routing from language +
			// enforcement; its rule entries carry no routing fields.
			allRules = append(allRules, mf.deriveRules()...)
		} else {
			// Legacy routing-schema manifest: use its routing rules directly.
			allRules = append(allRules, mf.legacyRules()...)
		}
	}

	// No manifest files in the dir → built-in defaults (unchanged).
	if !manifestFilesPresent {
		return defaultManifest(), nil
	}

	// Manifest files present but nothing routable is a config error, never a
	// silently empty route table — an empty table skips every pass and renders
	// as a green result.
	if !hasRoutableRule(allRules) {
		return nil, &ConfigError{Message: fmt.Sprintf(
			"manifest files in %s yield no routable rules: declare extensions or path_patterns with valid check_types, or use a compiled standards manifest with a derivable language or semgrep signal", dir)}
	}

	return &Manifest{rules: allRules}, nil
}

// hasRoutableRule reports whether any rule can actually route a file: at
// least one matcher (extension or path pattern) and at least one parsed
// check type.
func hasRoutableRule(rules []ManifestRule) bool {
	for _, r := range rules {
		if (len(r.Extensions) > 0 || len(r.PathPatterns) > 0) && len(r.parsed) > 0 {
			return true
		}
	}
	return false
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
