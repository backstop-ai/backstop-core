package schema

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/bmanson/backstop-core/pkg/artifact"
)

var schemaVersionRe = regexp.MustCompile(`^[a-z]+/v\d+$`)

// rawSchema mirrors the JSON structure of schema files.
type rawSchema struct {
	ID               string   `json:"$id"`
	ArtifactType     string   `json:"artifact_type"`
	FilenamePattern  string   `json:"filename_pattern"`
	SlugPattern      string   `json:"slug_pattern"`
	SlugMinLength    int      `json:"slug_min_length"`
	SlugMaxLength    int      `json:"slug_max_length"`
	Extends          string   `json:"extends"`
	RequiredSections []string `json:"required_sections"`
	OptionalSections []string `json:"optional_sections"`
	Version          string   `json:"version"`

	// Top-level metadata (extension schemas use this directly)
	Metadata rawMetadata `json:"metadata"`

	// Base schema nests metadata under properties
	Properties struct {
		Metadata rawMetadata `json:"metadata"`
	} `json:"properties"`
}

type rawMetadata struct {
	Required   []string               `json:"required"`
	Properties map[string]rawMetaProp `json:"properties"`
}

type rawMetaProp struct {
	Pattern   string   `json:"pattern"`
	Const     string   `json:"const"`
	MinLength int      `json:"minLength"`
	Enum      []string `json:"enum"`
}

// LoadSchema reads a base schema JSON file and returns a Schema.
func LoadSchema(path string) (*Schema, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var raw rawSchema
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("invalid schema JSON at %s: %w", path, err)
	}

	// Resolve metadata — extension schemas have it at top level,
	// base schemas nest it under properties.metadata
	meta := raw.Metadata
	if len(meta.Required) == 0 && len(raw.Properties.Metadata.Required) > 0 {
		meta = raw.Properties.Metadata
	}

	sch := &Schema{
		ArtifactType:     raw.ArtifactType,
		FilenamePattern:  raw.FilenamePattern,
		SlugPattern:      raw.SlugPattern,
		SlugMinLength:    raw.SlugMinLength,
		SlugMaxLength:    raw.SlugMaxLength,
		RequiredMetadata: meta.Required,
		RequiredSections: raw.RequiredSections,
		OptionalSections: raw.OptionalSections,
		Extends:          raw.Extends,
		MetadataRules:    make(map[string]MetadataRule),
	}

	for key, prop := range meta.Properties {
		sch.MetadataRules[key] = MetadataRule{
			Pattern:   prop.Pattern,
			Const:     prop.Const,
			MinLength: prop.MinLength,
		}
		if len(prop.Enum) > 0 && key == "status" {
			sch.StatusEnum = prop.Enum
		}
	}

	return sch, nil
}

// LoadArtifactSchema reads an extension schema, resolves its base via extends,
// and returns a merged Schema. Extension-specific metadata keys are separated
// into ExtensionMetadata to prevent duplicate violations.
func LoadArtifactSchema(schemaPath, artifactsRoot string) (*Schema, error) {
	ext, err := LoadSchema(schemaPath)
	if err != nil {
		return nil, err
	}

	if ext.Extends == "" {
		return ext, nil
	}

	// Load the base schema
	basePath := filepath.Join(artifactsRoot, "base", "schema.json")
	base, err := LoadSchema(basePath)
	if err != nil {
		return nil, fmt.Errorf("loading base schema: %w", err)
	}

	// Separate extension keys into base-overlap and extension-only.
	// The extension schema is the authority on what flat metadata is required.
	// Base keys not listed in the extension's required set are NOT forced.
	baseKeySet := make(map[string]bool)
	for _, k := range base.RequiredMetadata {
		baseKeySet[k] = true
	}

	var baseKeys []string
	var extensionKeys []string
	for _, k := range ext.RequiredMetadata {
		if baseKeySet[k] {
			baseKeys = append(baseKeys, k)
		} else {
			extensionKeys = append(extensionKeys, k)
		}
	}

	// RequiredMetadata = only the base keys the extension actually declares
	ext.RequiredMetadata = baseKeys
	ext.ExtensionMetadata = extensionKeys

	// Merge metadata rules (extension overrides base)
	for key, rule := range base.MetadataRules {
		if _, exists := ext.MetadataRules[key]; !exists {
			ext.MetadataRules[key] = rule
		}
	}

	return ext, nil
}

// ResolveSchemaPath extracts schema_version metadata from a parsed artifact
// and returns the filesystem path to the schema file.
func ResolveSchemaPath(art *artifact.ParsedArtifact) (string, error) {
	version := ""
	for k, v := range art.Metadata {
		if strings.EqualFold(k, "schema_version") || strings.EqualFold(k, "schema-version") {
			version = v
			break
		}
	}

	if version == "" {
		return "", fmt.Errorf("missing schema_version metadata")
	}

	if !schemaVersionRe.MatchString(version) {
		return "", fmt.Errorf("invalid Schema-Version format: %q", version)
	}

	parts := strings.SplitN(version, "/", 2)
	return filepath.Join("artifacts", parts[0], parts[1], "schema.json"), nil
}
