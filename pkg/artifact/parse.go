package artifact

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ParseFile reads a markdown artifact from disk and parses it.
func ParseFile(path string) (*ParsedArtifact, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read artifact %s: %w", path, err)
	}
	return Parse(string(data), path)
}

// Parse extracts structure from backstop markdown content.
// Format: YAML frontmatter between --- markers, then H1 title, then H2 sections.
func Parse(content, filename string) (*ParsedArtifact, error) {
	art := &ParsedArtifact{
		Filename:    filepath.Base(filename),
		Metadata:    make(map[string]string),
		Frontmatter: make(map[string]interface{}),
		Sections:    []string{},
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
		var yamlLines []string
		for i < len(lines) && strings.TrimSpace(lines[i]) != "---" {
			yamlLines = append(yamlLines, lines[i])
			i++
		}
		if i < len(lines) {
			i++ // skip closing ---
		}

		yamlContent := strings.Join(yamlLines, "\n")
		if yamlContent != "" {
			if err := yaml.Unmarshal([]byte(yamlContent), &art.Frontmatter); err != nil {
				return nil, fmt.Errorf("invalid YAML frontmatter in %s: %w", filename, err)
			}
			// Populate flat Metadata from top-level scalar values
			for key, val := range art.Frontmatter {
				switch v := val.(type) {
				case string:
					art.Metadata[key] = v
				case int:
					art.Metadata[key] = fmt.Sprintf("%d", v)
				case float64:
					art.Metadata[key] = fmt.Sprintf("%g", v)
				case bool:
					art.Metadata[key] = fmt.Sprintf("%t", v)
				}
			}
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
