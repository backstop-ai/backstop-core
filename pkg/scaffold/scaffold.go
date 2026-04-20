package scaffold

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ArtifactNewDeps holds injectable dependencies for the artifact new command.
type ArtifactNewDeps struct {
	Executor    GitExecutor
	ProjectRoot string
	DateFunc    func() string
}

// ArtifactTypeConfig holds the configuration for a single artifact type.
type ArtifactTypeConfig struct {
	// Directory is the target directory relative to the project root.
	Directory string
	// IDPrefix is the uppercase prefix for the artifact ID (e.g., "SPEC", "ADR").
	IDPrefix string
	// DigitCount is the number of zero-padded digits in the numeric ID.
	DigitCount int
	// DefaultStatus is the initial status for new artifacts of this type.
	DefaultStatus string
	// FileExtension is the file extension including the dot (e.g., ".spec.md").
	FileExtension string
	// BodySections lists the markdown section headings for the artifact body.
	BodySections []string
}

// ValidArtifactTypes maps artifact type names to their configuration.
var ValidArtifactTypes = map[string]ArtifactTypeConfig{
	"spec": {
		Directory:     "specs",
		IDPrefix:      "SPEC",
		DigitCount:    3,
		DefaultStatus: "draft",
		FileExtension: ".spec.md",
		BodySections:  []string{"Overview", "Requirements", "Implementation", "Verification"},
	},
	"plan": {
		Directory:     "plans",
		IDPrefix:      "PLAN",
		DigitCount:    3,
		DefaultStatus: "draft",
		FileExtension: ".plan.yml",
		BodySections:  nil, // Plan uses YAML phases, not markdown sections
	},
	"issue": {
		Directory:     "issues",
		IDPrefix:      "ISSUE",
		DigitCount:    3,
		DefaultStatus: "open",
		FileExtension: ".md",
		BodySections:  []string{"Problem"},
	},
	"adr": {
		Directory:     "adrs",
		IDPrefix:      "ADR",
		DigitCount:    4,
		DefaultStatus: "draft",
		FileExtension: ".adr.md",
		BodySections:  []string{"Context", "Decision", "Consequences"},
	},
	"directive": {
		Directory:     "directives",
		IDPrefix:      "DIR",
		DigitCount:    3,
		DefaultStatus: "queued",
		FileExtension: ".directive.md",
		BodySections:  []string{"Description"},
	},
	"bundle": {
		Directory:     "bundles",
		IDPrefix:      "BUNDLE",
		DigitCount:    3,
		DefaultStatus: "idea",
		FileExtension: ".bundle.md",
		BodySections:  []string{"Overview", "Components"},
	},
	"capability": {
		Directory:     "capabilities",
		IDPrefix:      "CAP",
		DigitCount:    3,
		DefaultStatus: "draft",
		FileExtension: ".capability.md",
		BodySections:  []string{"Overview", "Requirements"},
	},
}

// TargetDir returns the full target directory path for the given artifact type.
func TargetDir(artifactType string, projectRoot string) string {
	cfg := ValidArtifactTypes[artifactType]
	return filepath.Join(projectRoot, cfg.Directory)
}

// Filename returns the filename for the given artifact type, ID, slug, and sourceID.
// For plan types, sourceID determines the filename prefix (PLAN-SPEC- vs PLAN-ISSUE-).
func Filename(artifactType string, id string, slug string, sourceID string) string {
	cfg := ValidArtifactTypes[artifactType]

	if artifactType == "plan" {
		// Plan filenames use the source prefix and source's numeric ID.
		// PLAN-SPEC-002-slug.plan.yml or PLAN-ISSUE-005-slug.plan.yml
		var prefix string
		if strings.HasPrefix(sourceID, "SPEC-") {
			prefix = "PLAN-SPEC-"
		} else if strings.HasPrefix(sourceID, "ISSUE-") {
			prefix = "PLAN-ISSUE-"
		}
		// Extract numeric portion from sourceID (e.g., "002" from "SPEC-002")
		parts := strings.SplitN(sourceID, "-", 2)
		sourceNum := parts[1]
		return prefix + sourceNum + "-" + slug + cfg.FileExtension
	}

	// Standard filename: PREFIX-ID-slug.ext
	return cfg.IDPrefix + "-" + id + "-" + slug + cfg.FileExtension
}

// slugToTitle converts a kebab-case slug to a title case string.
func slugToTitle(slug string) string {
	words := strings.Split(slug, "-")
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}

// Scaffold renders the complete artifact content with frontmatter and body sections.
func Scaffold(artifactType string, id string, slug string, date string, sourceID string) ([]byte, error) {
	cfg, ok := ValidArtifactTypes[artifactType]
	if !ok {
		return nil, fmt.Errorf("unknown artifact type: %s", artifactType)
	}

	title := slugToTitle(slug)
	var sb strings.Builder

	sb.WriteString("---\n")

	switch artifactType {
	case "spec":
		sb.WriteString(fmt.Sprintf("title: %q\n", title))
		sb.WriteString(fmt.Sprintf("number: SPEC-%s\n", id))
		sb.WriteString(fmt.Sprintf("created: %q\n", date))
		sb.WriteString("status: draft\n")
		sb.WriteString("schema_version: spec/v1\n")
		sb.WriteString("spec_version: 1.0.0\n")

	case "plan":
		if strings.HasPrefix(sourceID, "SPEC-") {
			parts := strings.SplitN(sourceID, "-", 2)
			sb.WriteString(fmt.Sprintf("plan_id: PLAN-SPEC-%s\n", parts[1]))
			sb.WriteString(fmt.Sprintf("spec_id: %s\n", sourceID))
		} else if strings.HasPrefix(sourceID, "ISSUE-") {
			parts := strings.SplitN(sourceID, "-", 2)
			sb.WriteString(fmt.Sprintf("plan_id: PLAN-ISSUE-%s\n", parts[1]))
			sb.WriteString(fmt.Sprintf("issue_id: %s\n", sourceID))
		}
		sb.WriteString(fmt.Sprintf("created: %q\n", date))
		sb.WriteString("status: draft\n")

	case "issue":
		sb.WriteString(fmt.Sprintf("title: %q\n", title))
		sb.WriteString("schema_version: issue/v1\n")
		sb.WriteString("\n")
		sb.WriteString("issue:\n")
		sb.WriteString(fmt.Sprintf("  id: ISSUE-%s\n", id))
		sb.WriteString(fmt.Sprintf("  title: %q\n", title))
		sb.WriteString("  type: bug\n")
		sb.WriteString("  status: open\n")
		sb.WriteString(fmt.Sprintf("  created: %q\n", date))

	case "adr":
		sb.WriteString(fmt.Sprintf("title: %q\n", title))
		sb.WriteString(fmt.Sprintf("number: ADR-%s\n", id))
		sb.WriteString(fmt.Sprintf("created: %q\n", date))
		sb.WriteString("status: draft\n")
		sb.WriteString("schema_version: adr/v1\n")
		sb.WriteString("deciders: []\n")
		sb.WriteString("decisions: []\n")

	case "directive":
		sb.WriteString(fmt.Sprintf("title: %q\n", title))
		sb.WriteString(fmt.Sprintf("number: DIR-%s\n", id))
		sb.WriteString(fmt.Sprintf("created: %q\n", date))
		sb.WriteString("schema_version: directive/v1\n")
		sb.WriteString("\n")
		sb.WriteString("directive:\n")
		sb.WriteString("  status: queued\n")
		sb.WriteString("  source:\n")
		sb.WriteString("    - \"\"\n")

	case "bundle":
		sb.WriteString(fmt.Sprintf("title: %q\n", title))
		sb.WriteString(fmt.Sprintf("number: BUNDLE-%s\n", id))
		sb.WriteString(fmt.Sprintf("created: %q\n", date))
		sb.WriteString("schema_version: bundle/v1\n")
		sb.WriteString("\n")
		sb.WriteString("bundle:\n")
		sb.WriteString(fmt.Sprintf("  name: %s\n", slug))
		sb.WriteString("  version: \"1.0.0\"\n")
		sb.WriteString(fmt.Sprintf("  created: %q\n", date))
		sb.WriteString("  category: idea\n")

	case "capability":
		sb.WriteString(fmt.Sprintf("title: %q\n", title))
		sb.WriteString(fmt.Sprintf("number: CAP-%s\n", id))
		sb.WriteString(fmt.Sprintf("created: %q\n", date))
		sb.WriteString("status: draft\n")
		sb.WriteString("schema_version: capability/v1\n")
	}

	sb.WriteString("---\n")

	// Body sections
	if artifactType == "plan" {
		// Plan uses YAML phases, not markdown
		sb.WriteString("\nphases: []\n")
	} else {
		for _, section := range cfg.BodySections {
			sb.WriteString(fmt.Sprintf("\n## %s\n", section))
		}
	}

	return []byte(sb.String()), nil
}
