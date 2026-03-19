package artifact

import (
	"os"
	"path/filepath"
	"strings"
)

// ParseFile reads a markdown artifact from disk and parses it.
func ParseFile(path string) (*ParsedArtifact, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(string(data), path)
}

// Parse extracts structure from backstop markdown content.
// Format: YAML frontmatter between --- markers, then H1 title, then H2 sections.
func Parse(content string, filename string) (*ParsedArtifact, error) {
	art := &ParsedArtifact{
		Filename: filepath.Base(filename),
		Metadata: make(map[string]string),
		Sections: []string{},
	}

	lines := strings.Split(content, "\n")
	i := 0

	// Skip leading whitespace
	for i < len(lines) && strings.TrimSpace(lines[i]) == "" {
		i++
	}

	// Extract YAML frontmatter if present
	if i < len(lines) && strings.TrimSpace(lines[i]) == "---" {
		i++ // skip opening ---
		for i < len(lines) && strings.TrimSpace(lines[i]) != "---" {
			line := strings.TrimSpace(lines[i])
			if colonIdx := strings.Index(line, ":"); colonIdx > 0 {
				key := strings.TrimSpace(line[:colonIdx])
				val := strings.TrimSpace(line[colonIdx+1:])
				// Strip surrounding quotes
				val = strings.Trim(val, `"'`)
				if key != "" && val != "" {
					art.Metadata[key] = val
				}
			}
			i++
		}
		if i < len(lines) {
			i++ // skip closing ---
		}
	}

	// Extract H1 title and H2 sections from body
	for i < len(lines) {
		trimmed := strings.TrimSpace(lines[i])

		if art.Title == "" && strings.HasPrefix(trimmed, "# ") {
			art.Title = strings.TrimPrefix(trimmed, "# ")
		} else if strings.HasPrefix(trimmed, "## ") {
			art.Sections = append(art.Sections, strings.TrimPrefix(trimmed, "## "))
		}

		i++
	}

	return art, nil
}
