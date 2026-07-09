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
		FileExtension: ".issue.md",
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
		FileExtension: ".capability.yml",
		BodySections:  nil,
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
			// spec_id is the plan schema's backing-artifact field for BOTH SPEC- and
			// ISSUE-sourced plans (the validator requires spec_id matching SPEC-NNN OR
			// ISSUE-NNN; every existing issue-plan uses spec_id). The scaffold formerly
			// emitted issue_id here, which the validator rejected (ISSUE-009).
			sb.WriteString(fmt.Sprintf("spec_id: %s\n", sourceID))
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
		// bundle/v2 (ISSUE-032 Defect E / CLM-009): the scaffolder builds a numbered
		// BUNDLE-NNN-<slug>.bundle.md filename that ONLY bundle/v2's filename_pattern
		// accepts (v1's pattern rejects the numbered prefix), so a freshly scaffolded
		// bundle must stamp v2 to validate against the schema its filename requires.
		sb.WriteString("schema_version: bundle/v2\n")
		sb.WriteString("\n")
		sb.WriteString("bundle:\n")
		sb.WriteString(fmt.Sprintf("  name: %s\n", slug))
		// version 0.1.0 keeps a fresh scaffold valid without an `updated` field (the v2
		// schema requires bundle.updated only beyond 0.1.0); category is a v2-enum value
		// (v1's `idea` is not in the v2 enum). A scaffolded bundle starts at `exploring`
		// (ISSUE-032 Defect E / CLM-009 — a fresh numbered bundle validates against v2).
		sb.WriteString("  version: \"0.1.0\"\n")
		sb.WriteString(fmt.Sprintf("  created: %q\n", date))
		sb.WriteString("  category: feature\n")
		sb.WriteString("\n")
		sb.WriteString("status:\n")
		sb.WriteString("  maturity: exploring\n")

	case "capability":
		sb.WriteString(fmt.Sprintf("# Capability: %s\n", title))
		sb.WriteString(fmt.Sprintf("# See artifacts/capability/v1/schema.json for format\n\n"))
		sb.WriteString("capability:\n")
		sb.WriteString(fmt.Sprintf("  id: CAP-%s\n", id))
		sb.WriteString(fmt.Sprintf("  title: %q\n", title))
		sb.WriteString("  status: draft\n")
		sb.WriteString("  strictness: relaxed\n")
		sb.WriteString("\ninfrastructure_specs: []\n")
		sb.WriteString("\nquality_gates:\n")
		sb.WriteString("  - type: acceptance\n")
		sb.WriteString("    command: \"\"\n")
		sb.WriteString("    must_pass: true\n")
		return []byte(sb.String()), nil
	}

	sb.WriteString("---\n")

	// Body sections
	if artifactType == "plan" {
		// Plan uses YAML phases, not markdown
		sb.WriteString("\nphases: []\n")
	} else {
		if artifactType == "bundle" {
			// base/title-required reads the artifact title from the body H1 heading
			// (art.Title), not the frontmatter title: key — so a fresh numbered bundle
			// needs an H1 to validate against bundle/v2 (ISSUE-032 Defect E / CLM-009).
			sb.WriteString(fmt.Sprintf("\n# %s\n", title))
		}
		for _, section := range cfg.BodySections {
			sb.WriteString(fmt.Sprintf("\n## %s\n", section))
		}
	}

	return []byte(sb.String()), nil
}
