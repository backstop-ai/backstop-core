package artifact

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var metadataRe = regexp.MustCompile(`^\*\*(.+?):\*\*\s*(.+)$`)

// ParseFile reads a markdown artifact from disk and parses it.
func ParseFile(path string) (*ParsedArtifact, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(string(data), path)
}

// Parse extracts structure from backstop markdown content.
func Parse(content string, filename string) (*ParsedArtifact, error) {
	art := &ParsedArtifact{
		Filename: filepath.Base(filename),
		Metadata: make(map[string]string),
		Sections: []string{},
	}

	lines := strings.Split(content, "\n")
	pastSeparator := false
	titleFound := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Extract H1 title
		if !titleFound && strings.HasPrefix(trimmed, "# ") {
			art.Title = strings.TrimPrefix(trimmed, "# ")
			titleFound = true
			continue
		}

		// Before separator: extract metadata
		if !pastSeparator {
			if trimmed == "---" {
				pastSeparator = true
				continue
			}
			if titleFound {
				if m := metadataRe.FindStringSubmatch(trimmed); m != nil {
					art.Metadata[strings.TrimSpace(m[1])] = strings.TrimSpace(m[2])
				}
			}
			continue
		}

		// After separator: extract H2 sections
		if strings.HasPrefix(trimmed, "## ") {
			art.Sections = append(art.Sections, strings.TrimPrefix(trimmed, "## "))
		}
	}

	return art, nil
}
