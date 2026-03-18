package artifact

// ParsedArtifact represents a backstop markdown artifact parsed into its
// structural components: title, metadata key-value pairs, and section headings.
type ParsedArtifact struct {
	Filename string
	Title    string
	Metadata map[string]string
	Sections []string
}
