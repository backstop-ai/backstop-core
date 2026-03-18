package schema

// Schema represents the loaded validation rules for an artifact type.
type Schema struct {
	ArtifactType      string
	FilenamePattern   string
	SlugPattern       string
	SlugMinLength     int
	SlugMaxLength     int
	RequiredMetadata  []string // Base keys checked by ValidateBase
	ExtensionMetadata []string // Type-specific keys checked by type validator
	MetadataRules     map[string]MetadataRule
	RequiredSections  []string
	OptionalSections  []string
	Extends           string
	StatusEnum        []string
}

// MetadataRule defines validation constraints for a single metadata key.
type MetadataRule struct {
	Pattern   string
	Const     string
	MinLength int
}
