// Package config provides backstop.yml discovery, loading, and validation.
package config

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	backstopcore "github.com/bmanson/backstop-core"
	"gopkg.in/yaml.v3"
)

const defaultBaselineTTL = 15 * time.Minute

// Config is the typed representation of backstop.yml.
type Config struct {
	Project       string            `yaml:"project" json:"project"`
	Language      string            `yaml:"language" json:"language"`
	Runtimes      []string          `yaml:"runtimes,omitempty" json:"runtimes,omitempty"`
	StandardsDirs []string          `yaml:"standards_dirs,omitempty" json:"standards_dirs,omitempty"`
	Enforcement   Enforcement       `yaml:"enforcement,omitempty" json:"enforcement,omitempty"`
	Packs         Packs             `yaml:"packs,omitempty" json:"packs,omitempty"`
	Registries    map[string]string `yaml:"registries,omitempty" json:"registries,omitempty"`
}

// Enforcement holds the enforcement configuration block.
type Enforcement struct {
	Security          Security                  `yaml:"security,omitempty" json:"security,omitempty"`
	WaiverWarningDays int                       `yaml:"waiver_warning_days,omitempty" json:"waiver_warning_days,omitempty"`
	SemgrepVersion    string                    `yaml:"semgrep_version,omitempty" json:"semgrep_version,omitempty"`
	BaselineTTL       string                    `yaml:"baseline_ttl,omitempty" json:"baseline_ttl,omitempty"`
	TestCommand       string                    `yaml:"test_command,omitempty" json:"test_command,omitempty"`
	Toolchain         map[string]ToolchainPass `yaml:"toolchain,omitempty" json:"toolchain,omitempty"`
}

// ToolchainPass is a single declared pass binding in enforcement.toolchain: the
// command to run, the named output format that parses its output, the file
// extensions the stack routes, and an optional per-toolchain test
// dependency-mapping command for the test pass. It is an extensible object (not
// a positional tuple) so future stack-generic knobs (e.g. ISSUE-007's
// exclude_paths) can slot in without reshaping the registry.
type ToolchainPass struct {
	Command               string   `yaml:"command" json:"command"`
	Format                string   `yaml:"format" json:"format"`
	Extensions            []string `yaml:"extensions,omitempty" json:"extensions,omitempty"`
	TestDependencyCommand string   `yaml:"test_dependency_command,omitempty" json:"test_dependency_command,omitempty"`
}

// Security holds the security enforcement settings.
type Security struct {
	Tier string `yaml:"tier,omitempty" json:"tier,omitempty"`
}

// Packs is a map of pack ref → version. The ref is either the pack name
// (public, e.g. "acme/go-standards") or a generated opaque ref (private,
// e.g. "gentle-river-k8x2"). Version is the pinned semver string.
// Local packs use "local" as the version.
type Packs map[string]string

// DiscoverConfigPath finds backstop.yml by checking BACKSTOP_CONFIG env var
// first, then walking up from the current working directory.
func DiscoverConfigPath() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting working directory: %w", err)
	}
	return DiscoverConfigPathFrom(cwd)
}

// DiscoverConfigPathFrom finds backstop.yml by checking BACKSTOP_CONFIG env var
// first, then walking up from startDir. Exported for testing.
func DiscoverConfigPathFrom(startDir string) (string, error) {
	// Check BACKSTOP_CONFIG env var first
	envPath := os.Getenv("BACKSTOP_CONFIG")
	if envPath != "" {
		// Resolve relative paths from cwd
		if !filepath.IsAbs(envPath) {
			cwd, err := os.Getwd()
			if err != nil {
				return "", fmt.Errorf("resolving BACKSTOP_CONFIG: %w", err)
			}
			envPath = filepath.Join(cwd, envPath)
		}
		info, err := os.Stat(envPath)
		if err != nil {
			return "", fmt.Errorf("BACKSTOP_CONFIG path %q: %w", envPath, err)
		}
		if info.IsDir() {
			return "", fmt.Errorf("BACKSTOP_CONFIG path %q is a directory, expected a file", envPath)
		}
		return envPath, nil
	}

	// Walk up from startDir
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", fmt.Errorf("resolving start directory: %w", err)
	}

	for {
		candidate := filepath.Join(dir, "backstop.yml")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root
			return "", fmt.Errorf("backstop.yml not found in %q or any parent directory; create one or set BACKSTOP_CONFIG", startDir)
		}
		dir = parent
	}
}

// LoadConfig discovers and loads backstop.yml from the current working
// directory using walk-up discovery or BACKSTOP_CONFIG override.
func LoadConfig() (*Config, error) {
	path, err := DiscoverConfigPath()
	if err != nil {
		return nil, err
	}
	return LoadConfigFromPath(path)
}

// LoadConfigFromDir discovers and loads backstop.yml starting from dir.
func LoadConfigFromDir(dir string) (*Config, error) {
	path, err := DiscoverConfigPathFrom(dir)
	if err != nil {
		return nil, err
	}
	return LoadConfigFromPath(path)
}

// LoadConfigFromPath loads and validates backstop.yml from a specific path.
func LoadConfigFromPath(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config %s: %w", path, err)
	}

	// Strict YAML decode to reject unknown keys
	var cfg Config
	decoder := yaml.NewDecoder(strings.NewReader(string(data)))
	decoder.KnownFields(true)
	if err := decoder.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("parsing backstop.yml: %w", err)
	}

	// Validate against embedded JSON schema
	if err := validateAgainstSchema(data); err != nil {
		return nil, fmt.Errorf("backstop.yml schema validation: %w", err)
	}

	// Apply defaults
	if cfg.Enforcement.WaiverWarningDays == 0 {
		cfg.Enforcement.WaiverWarningDays = 30
	}

	return &cfg, nil
}

// BaselineTTLDuration resolves enforcement.baseline_ttl with defaulting.
// Empty baseline_ttl defaults to 15 minutes.
func (c *Config) BaselineTTLDuration() (time.Duration, error) {
	if c == nil {
		return defaultBaselineTTL, nil
	}
	raw := strings.TrimSpace(c.Enforcement.BaselineTTL)
	if raw == "" {
		return defaultBaselineTTL, nil
	}
	ttl, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid enforcement.baseline_ttl: %w", err)
	}
	if ttl <= 0 {
		return 0, fmt.Errorf("invalid enforcement.baseline_ttl: must be greater than 0")
	}
	return ttl, nil
}

// schemaData is the raw backstop-yml schema loaded from the embedded FS.
// Loaded once on first use.
var schemaData []byte

func loadSchemaData() ([]byte, error) {
	if schemaData != nil {
		return schemaData, nil
	}
	data, err := fs.ReadFile(backstopcore.SchemaFS, "artifacts/backstop-yml/v1/schema.json")
	if err != nil {
		return nil, fmt.Errorf("reading embedded backstop-yml schema: %w", err)
	}
	schemaData = data
	return schemaData, nil
}

// validateAgainstSchema validates YAML data against the backstop-yml JSON
// schema. It converts YAML to JSON then validates using the schema.
func validateAgainstSchema(yamlData []byte) error {
	sd, err := loadSchemaData()
	if err != nil {
		return err
	}

	// Parse the JSON schema
	var schemaMap map[string]interface{}
	if err := json.Unmarshal(sd, &schemaMap); err != nil {
		return fmt.Errorf("parsing backstop-yml schema: %w", err)
	}

	// Parse YAML to generic map for JSON schema validation
	var doc interface{}
	if err := yaml.Unmarshal(yamlData, &doc); err != nil {
		return fmt.Errorf("parsing YAML for schema validation: %w", err)
	}

	// Convert YAML map to JSON-compatible structure
	doc = convertYAMLToJSON(doc)

	// Validate required fields
	docMap, ok := doc.(map[string]interface{})
	if !ok {
		return fmt.Errorf("backstop.yml must be a YAML mapping")
	}

	// Check required fields from schema
	required, _ := schemaMap["required"].([]interface{})
	for _, r := range required {
		key, _ := r.(string)
		if key == "" {
			continue
		}
		if _, exists := docMap[key]; !exists {
			return fmt.Errorf("required field %q is missing", key)
		}
	}

	// Check additionalProperties: false
	if ap, ok := schemaMap["additionalProperties"]; ok {
		if allowed, ok := ap.(bool); ok && !allowed {
			props, _ := schemaMap["properties"].(map[string]interface{})
			for key := range docMap {
				if _, defined := props[key]; !defined {
					return fmt.Errorf("unknown field %q in backstop.yml (additionalProperties is false)", key)
				}
			}
		}
	}

	// Validate enforcement tier enum
	if enfObj, ok := docMap["enforcement"]; ok {
		if enfMap, ok := enfObj.(map[string]interface{}); ok {
			if secObj, ok := enfMap["security"]; ok {
				if secMap, ok := secObj.(map[string]interface{}); ok {
					if tierVal, ok := secMap["tier"]; ok {
						tier, _ := tierVal.(string)
						validTiers := map[string]bool{"baseline": true, "standard": true, "compliance": true}
						if !validTiers[tier] {
							return fmt.Errorf("enforcement.security.tier %q is not valid; must be one of: baseline, standard, compliance", tier)
						}
					}
				}
			}
		}
	}

	return nil
}

// convertYAMLToJSON converts YAML-parsed values to JSON-compatible types.
// YAML parses maps as map[string]interface{} but some values may use
// map[interface{}]interface{} which JSON doesn't support.
func convertYAMLToJSON(v interface{}) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{}, len(val))
		for k, v := range val {
			result[k] = convertYAMLToJSON(v)
		}
		return result
	case map[interface{}]interface{}:
		result := make(map[string]interface{}, len(val))
		for k, v := range val {
			result[fmt.Sprintf("%v", k)] = convertYAMLToJSON(v)
		}
		return result
	case []interface{}:
		for i, item := range val {
			val[i] = convertYAMLToJSON(item)
		}
		return val
	default:
		return val
	}
}
