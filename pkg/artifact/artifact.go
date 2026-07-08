package artifact

// ParsedArtifact represents a backstop markdown artifact parsed into its
// structural components: title, metadata key-value pairs, and section headings.
type ParsedArtifact struct {
	Filename string
	// SourcePath is the full path this artifact was parsed from (the exact
	// argument handed to Parse/ParseFile), retained so callers can resolve
	// sibling artifacts relative to THIS file rather than the ambient working
	// directory. Filename stays the base name; SourcePath keeps the directory.
	// Enables ISSUE-043's CWD-independent, source-path-anchored plan resolution.
	SourcePath  string
	Title       string
	Metadata    map[string]string      // Flat top-level key-value pairs
	Frontmatter map[string]interface{} // Full YAML tree for nested access
	Sections    []string
}
