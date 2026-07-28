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

	backstopcore "github.com/backstop-ai/backstop-core"
	"gopkg.in/yaml.v3"
)

const defaultBaselineTTL = 15 * time.Minute

// Config is the typed representation of backstop.yml.
//
// SPEC-046: the single-language `language` field is RETIRED — a project is described
// by its declared packs, not one baked language. An existing backstop.yml may still
// carry a `language:` key; it is absorbed by LegacyKeys below and ignored. The
// look-alikes pack.Manifest.Language and check.Options.Language are SEPARATE fields on
// other structs and are unaffected.
type Config struct {
	Project     string            `yaml:"project" json:"project"`
	Runtimes    []string          `yaml:"runtimes,omitempty" json:"runtimes,omitempty"`
	Enforcement Enforcement       `yaml:"enforcement,omitempty" json:"enforcement,omitempty"`
	Packs       Packs             `yaml:"packs,omitempty" json:"packs,omitempty"`
	Registries  map[string]string `yaml:"registries,omitempty" json:"registries,omitempty"`
	// LegacyKeys absorbs retired/legacy top-level keys (the SPEC-046 `language:` key)
	// so an existing backstop.yml carrying one parses cleanly under the strict
	// (KnownFields) decoder rather than erroring — the field is GONE, not rejected
	// (CLM-012). Truly-unknown top-level keys are still rejected by the JSON-schema
	// additionalProperties:false pass (which lists `language` as the only allowed-but-
	// ignored legacy property), so strictness is preserved: only schema-allowed legacy
	// keys are tolerated here. It carries no behavior and is never read (json:"-").
	LegacyKeys map[string]any `yaml:",inline" json:"-"`
}

// Enforcement holds the enforcement configuration block.
type Enforcement struct {
	Security          Security                 `yaml:"security,omitempty" json:"security,omitempty"`
	WaiverWarningDays int                      `yaml:"waiver_warning_days,omitempty" json:"waiver_warning_days,omitempty"`
	BaselineTTL       string                   `yaml:"baseline_ttl,omitempty" json:"baseline_ttl,omitempty"`
	TestCommand       string                   `yaml:"test_command,omitempty" json:"test_command,omitempty"`
	Toolchain         map[string]ToolchainPass `yaml:"toolchain,omitempty" json:"toolchain,omitempty"`
	// Policy is the per-dimension enforcement policy, keyed by gate dimension
	// (the step/gate_type name, e.g. "pack_engines", "coverage_threshold"). Each
	// entry sets the enforcement level and whether pre-existing findings are
	// grandfathered (applies-to: new-code) or the total is blocked (applies-to:
	// all-code, the absent default). A dimension with no entry keeps the default
	// behavior (block, all-code). The keys are backstop's universal dimension
	// vocabulary — never a tool or language name.
	Policy map[string]DimensionPolicy `yaml:"policy,omitempty" json:"policy,omitempty"`
}

// DimensionPolicy is one row of the enforcement policy table: how strictly a gate
// dimension is enforced (level) and WHICH violations count (applies-to). Level is
// "off" (don't enforce), "warn" (surface, non-blocking), or "block" (fail the gate);
// empty defaults to "block". AppliesTo is "new-code" (grandfather pre-existing
// findings against the baseline — block only net-new, the ratchet) or "all-code"
// (block on the TOTAL, zero tolerance); an ABSENT/empty applies-to defaults to
// all-code (the strict floor — a bare dimension is never silently grandfathered).
// Level and applies-to are orthogonal: applies-to decides which violations count,
// level decides what happens once they do.
//
// Sources is the OPTIONAL per-PACK / per-rule-SOURCE scoping (SPEC-047 REQ-007),
// keyed by the pack/rule-source name (matched against gate.Violation.SourcePack). A
// source-scoped override applies its level+applies-to ONLY to that pack's findings
// within the dimension; every OTHER pack's findings keep the dimension default (or
// their own scoped override). This lets `backstop/self` flip to block + all-code
// (zero baseline) on the shared `pack_engines` dimension WITHOUT disturbing
// go-standards/go-toolchain's new-code-grandfathered style debt. Absent Sources ⇒ the
// entry is dimension-only and behaves exactly as before (backward compatible, CLM-036).
type DimensionPolicy struct {
	Level     string                     `yaml:"level,omitempty" json:"level,omitempty"`
	AppliesTo string                     `yaml:"applies-to,omitempty" json:"applies-to,omitempty"`
	Sources   map[string]DimensionPolicy `yaml:"sources,omitempty" json:"sources,omitempty"`
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
	// GateType names the traceability dimension this toolchain entry declares
	// (substantiveness | coverage | contracts). Its presence with a matching
	// value is what makes a dimension DECLARED for the SPEC-036 polarity
	// classifier. Optional; the zero value ("") means the entry declares no
	// traceability dimension.
	GateType string `yaml:"gate_type,omitempty" json:"gate_type,omitempty"`
	// Waived opts a declared dimension out of its class-2 capability-absent
	// advisory (SPEC-036 REQ-007). A waive silences ONLY the class-2 warning; it
	// never silences a class-1 broken-declared or class-3 declared-intent-unmet
	// failure. Optional; the zero value (false) means not waived.
	Waived bool `yaml:"waived,omitempty" json:"waived,omitempty"`
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
		return nil, fmt.Errorf("load config: %w", err)
	}
	return LoadConfigFromPath(path)
}

// LoadConfigFromDir discovers and loads backstop.yml starting from dir.
func LoadConfigFromDir(dir string) (*Config, error) {
	path, err := DiscoverConfigPathFrom(dir)
	if err != nil {
		return nil, fmt.Errorf("load config from %q: %w", dir, err)
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
			// Validate the per-dimension enforcement.policy block (applies-to / level
			// enums + the nested per-source overrides). The top-level strict YAML decode
			// already rejects unknown dimension keys (e.g. the retired `baseline:`); this
			// closes the enum gap the hand-rolled schema walk otherwise leaves open.
			if polMap, ok := enfMap["policy"].(map[string]interface{}); ok {
				for dim, entry := range polMap {
					entryMap, _ := entry.(map[string]interface{})
					if err := validatePolicyEntry(fmt.Sprintf("enforcement.policy.%s", dim), entryMap); err != nil {
						return err
					}
					srcMap, _ := entryMap["sources"].(map[string]interface{})
					for src, sEntry := range srcMap {
						sEntryMap, _ := sEntry.(map[string]interface{})
						if err := validatePolicyEntry(fmt.Sprintf("enforcement.policy.%s.sources.%s", dim, src), sEntryMap); err != nil {
							return err
						}
					}
				}
			}
		}
	}

	return nil
}

// validatePolicyEntry enforces the applies-to / level ENUM constraints on one
// enforcement.policy dimension entry (or one per-source override). The strict YAML
// decode already rejects unknown/retired keys (e.g. `baseline:`) and the wrong value
// TYPE; this closes the one gap it leaves — a syntactically-valid string that is not a
// member of the level (off|warn|block) or applies-to (new-code|all-code) enum.
func validatePolicyEntry(path string, entry map[string]interface{}) error {
	if lvlVal, ok := entry["level"]; ok {
		lvl, _ := lvlVal.(string)
		if lvl != "off" && lvl != "warn" && lvl != "block" {
			return fmt.Errorf("%s.level %q is not valid; must be one of: off, warn, block", path, lvl)
		}
	}
	if atVal, ok := entry["applies-to"]; ok {
		at, _ := atVal.(string)
		if at != "new-code" && at != "all-code" {
			return fmt.Errorf("%s.applies-to %q is not valid; must be one of: new-code, all-code", path, at)
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
