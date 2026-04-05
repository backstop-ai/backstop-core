package pack

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// ValidPackTypes is the registry of valid pack types.
var ValidPackTypes = map[string]bool{
	"rule": true,
	"code": true,
}

var slugPattern = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)

// ValidateSlug checks that the slug conforms to ^[a-z][a-z0-9]*(-[a-z0-9]+)*$
// with a minimum length of 2 and maximum length of 64 characters.
func ValidateSlug(slug string) error {
	if slug == "" {
		return fmt.Errorf("slug is required")
	}
	if len(slug) < 2 {
		return fmt.Errorf("slug must be at least 2 characters, got %d", len(slug))
	}
	if len(slug) > 64 {
		return fmt.Errorf("slug must be at most 64 characters, got %d", len(slug))
	}
	if !slugPattern.MatchString(slug) {
		return fmt.Errorf("slug must match ^[a-z][a-z0-9]*(-[a-z0-9]+)*$, got %q", slug)
	}
	return nil
}

// ScaffoldOptions holds options for pack scaffolding.
type ScaffoldOptions struct {
	Type        string
	Language    string
	Slug        string
	Number      int
	ProjectRoot string
}

// ScaffoldResult holds the result of scaffolding.
type ScaffoldResult struct {
	Paths         []string `json:"paths"`
	PackID        string   `json:"pack_id"`
	Type          string   `json:"type"`
	Language      string   `json:"language"`
	Slug          string   `json:"slug"`
	SchemaVersion string   `json:"schema_version"`
}

// HumanString formats the result for human display.
func (r *ScaffoldResult) HumanString() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Created %s pack: %s\n", r.Type, r.PackID))
	for _, p := range r.Paths {
		sb.WriteString(fmt.Sprintf("  %s\n", p))
	}
	return sb.String()
}

// ScaffoldPack creates pack directory structure and template files for the
// given pack type and language.
func ScaffoldPack(opts ScaffoldOptions) (*ScaffoldResult, error) {
	switch opts.Type {
	case "rule":
		return scaffoldRulePack(opts)
	case "code":
		return scaffoldCodePack(opts)
	default:
		return nil, fmt.Errorf("unsupported pack type: %q", opts.Type)
	}
}

func scaffoldRulePack(opts ScaffoldOptions) (*ScaffoldResult, error) {
	langUpper := strings.ToUpper(opts.Language)
	packID := fmt.Sprintf("STD-%s-%03d", langUpper, opts.Number)
	filename := fmt.Sprintf("%s-%s.standard.md", packID, opts.Slug)

	dir := filepath.Join(opts.ProjectRoot, "standards", opts.Language)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("creating standards directory: %w", err)
	}

	filePath := filepath.Join(dir, filename)

	// Check for conflict
	if _, err := os.Stat(filePath); err == nil {
		return nil, fmt.Errorf("file already exists: %s", filePath)
	}

	// Render the standard file content
	title := slugToTitle(opts.Slug)
	today := time.Now().Format("2006-01-02")

	content := fmt.Sprintf(`---
title: "%s"
number: %s
created: "%s"
status: active
schema_version: standard/v1
language: %s
pack: %s
scope: language

rules:
  - id: %s-R001
    name: "%s Rule 1"
    category: correctness
    severity: error
    description: "TODO: Describe what this rule enforces"
    compliance_tier: required
    detection:
      strategy: semgrep
      pattern: "TODO: Add detection pattern"
---

# %s

TODO: Add standard description and rationale.
`, title, packID, today, opts.Language, opts.Language,
		packID, title, title)

	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		return nil, fmt.Errorf("writing standard file: %w", err)
	}

	return &ScaffoldResult{
		Paths:         []string{filePath},
		PackID:        packID,
		Type:          opts.Type,
		Language:      opts.Language,
		Slug:          opts.Slug,
		SchemaVersion: "pack-new/v1",
	}, nil
}

func scaffoldCodePack(opts ScaffoldOptions) (*ScaffoldResult, error) {
	dir := filepath.Join(opts.ProjectRoot, "recipes", opts.Language, opts.Slug)

	// Check for conflict
	if _, err := os.Stat(dir); err == nil {
		return nil, fmt.Errorf("directory already exists: %s", dir)
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("creating recipe directory: %w", err)
	}

	title := slugToTitle(opts.Slug)

	// Write README.md
	readmePath := filepath.Join(dir, "README.md")
	readmeContent := fmt.Sprintf("# %s\n\nDescription placeholder.\n", title)
	if err := os.WriteFile(readmePath, []byte(readmeContent), 0o644); err != nil {
		return nil, fmt.Errorf("writing README.md: %w", err)
	}

	// Write <slug>.recipe.md
	recipePath := filepath.Join(dir, opts.Slug+".recipe.md")
	recipeContent := fmt.Sprintf(`---
title: "%s"
slug: %s
language: %s
schema_version: recipe/v1
---

# %s

TODO: Add recipe implementation guidance.
`, title, opts.Slug, opts.Language, title)
	if err := os.WriteFile(recipePath, []byte(recipeContent), 0o644); err != nil {
		return nil, fmt.Errorf("writing recipe file: %w", err)
	}

	packID := fmt.Sprintf("%s/%s", opts.Language, opts.Slug)

	return &ScaffoldResult{
		Paths:         []string{readmePath, recipePath},
		PackID:        packID,
		Type:          opts.Type,
		Language:      opts.Language,
		Slug:          opts.Slug,
		SchemaVersion: "pack-new/v1",
	}, nil
}

// slugToTitle converts a kebab-case slug to a title-case string.
func slugToTitle(slug string) string {
	words := strings.Split(slug, "-")
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}
