package validate

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/bmanson/backstop-core/pkg/artifact"
	"github.com/bmanson/backstop-core/pkg/schema"
)

var (
	bundleNameRe    = regexp.MustCompile(`^[a-z0-9-]+$`)
	semverRe        = regexp.MustCompile(`^\d+\.\d+\.\d+$`)
	dateRe          = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	epicIDRe        = regexp.MustCompile(`^EPIC-[A-Z0-9-]+$`)
	placeholderRe   = regexp.MustCompile(`(?i)\b(TBD|TODO|FIXME|XXX)\b|\?\?\?`)
	bundleCategories = map[string]bool{
		"feature": true, "service": true, "integration": true,
		"recipe": true, "infrastructure": true, "tool": true, "epic": true,
	}
	maturityLevels = map[string]bool{
		"idea": true, "exploring": true, "defined": true, "ready": true,
	}
	// Sections required at defined/ready maturity
	matureSections = []string{
		"Current Thinking", "Draft Requirements",
		"Draft Design Decisions", "Spec Seeds", "Version History",
	}
)

// ValidateBundle composes base validation with bundle-specific checks
// including maturity-gated progressive requirements, epic validation,
// and placeholder bans.
func ValidateBundle(art *artifact.ParsedArtifact, sch *schema.Schema) ValidationResult {
	base := ValidateBase(art, sch)
	var violations []Violation

	// 1. Filename pattern
	filenameOK := false
	if sch.FilenamePattern != "" {
		re, err := regexp.Compile(sch.FilenamePattern)
		if err == nil {
			filenameOK = re.MatchString(art.Filename)
		}
		if !filenameOK {
			violations = append(violations, Violation{
				Rule:     "bundle/filename-pattern",
				File:     art.Filename,
				Message:  fmt.Sprintf("filename does not match pattern %s", sch.FilenamePattern),
				Severity: "error",
			})
		}
	}

	// 2. schema_version cross-check
	if sv, ok := art.Metadata["schema_version"]; ok && sv != "" {
		parts := strings.SplitN(sv, "/", 2)
		if len(parts) == 2 && sch.ArtifactType != "" && parts[0] != sch.ArtifactType {
			violations = append(violations, Violation{
				Rule:     "bundle/schema-version-mismatch",
				File:     art.Filename,
				Message:  fmt.Sprintf("schema_version type '%s' does not match artifact type '%s'", parts[0], sch.ArtifactType),
				Severity: "error",
			})
		}
	}

	// 3. Core bundle fields
	violations = append(violations, validateBundleBlock(art)...)

	// 4. Status block
	maturity := extractMaturity(art)
	violations = append(violations, validateStatusBlock(art, maturity)...)

	// 5. Name/filename consistency
	if filenameOK {
		violations = append(violations, validateNameFilenameConsistency(art)...)
	}

	// 6. Version-gated: bundle.updated required when version > 0.1.0
	violations = append(violations, validateVersionGatedUpdated(art)...)

	// 7. Maturity-gated requirements
	violations = append(violations, validateMaturityGates(art, maturity)...)

	// 8. Epic validation
	violations = append(violations, validateEpicBlock(art)...)

	// 9. Placeholder ban in problem.summary at defined/ready
	violations = append(violations, validatePlaceholderBan(art, maturity)...)

	combined := append(base.Violations, violations...)
	return ValidationResult{Violations: combined}
}

// validateBundleBlock checks the required bundle.* frontmatter fields.
func validateBundleBlock(art *artifact.ParsedArtifact) []Violation {
	var violations []Violation

	bundleVal, ok := art.Frontmatter["bundle"]
	if !ok {
		violations = append(violations, Violation{
			Rule:     "bundle/block-required",
			File:     art.Filename,
			Message:  "bundle block is missing from frontmatter",
			Severity: "error",
		})
		return violations
	}

	bundle, ok := bundleVal.(map[string]interface{})
	if !ok {
		violations = append(violations, Violation{
			Rule:     "bundle/block-required",
			File:     art.Filename,
			Message:  "bundle block is not a valid map",
			Severity: "error",
		})
		return violations
	}

	// bundle.name
	if name, ok := getStringField(bundle, "name"); !ok {
		violations = append(violations, Violation{
			Rule:     "bundle/name-required",
			File:     art.Filename,
			Message:  "bundle.name is required",
			Severity: "error",
		})
	} else if !bundleNameRe.MatchString(name) {
		violations = append(violations, Violation{
			Rule:     "bundle/name-pattern",
			File:     art.Filename,
			Message:  fmt.Sprintf("bundle.name '%s' must be kebab-case (pattern: ^[a-z0-9-]+$)", name),
			Severity: "error",
		})
	}

	// bundle.version
	if version, ok := getStringField(bundle, "version"); !ok {
		violations = append(violations, Violation{
			Rule:     "bundle/version-required",
			File:     art.Filename,
			Message:  "bundle.version is required",
			Severity: "error",
		})
	} else if !semverRe.MatchString(version) {
		violations = append(violations, Violation{
			Rule:     "bundle/version-pattern",
			File:     art.Filename,
			Message:  fmt.Sprintf("bundle.version '%s' must be semver (X.Y.Z)", version),
			Severity: "error",
		})
	}

	// bundle.created
	if created, ok := getStringField(bundle, "created"); !ok {
		violations = append(violations, Violation{
			Rule:     "bundle/created-required",
			File:     art.Filename,
			Message:  "bundle.created is required",
			Severity: "error",
		})
	} else if !dateRe.MatchString(created) {
		violations = append(violations, Violation{
			Rule:     "bundle/created-pattern",
			File:     art.Filename,
			Message:  fmt.Sprintf("bundle.created '%s' must be YYYY-MM-DD", created),
			Severity: "error",
		})
	}

	// bundle.category
	if cat, ok := getStringField(bundle, "category"); !ok {
		violations = append(violations, Violation{
			Rule:     "bundle/category-required",
			File:     art.Filename,
			Message:  "bundle.category is required",
			Severity: "error",
		})
	} else if !bundleCategories[cat] {
		violations = append(violations, Violation{
			Rule:     "bundle/category-enum",
			File:     art.Filename,
			Message:  fmt.Sprintf("bundle.category '%s' is not valid (allowed: feature, service, integration, recipe, infrastructure, tool, epic)", cat),
			Severity: "error",
		})
	}

	return violations
}

// validateStatusBlock checks the required status.* frontmatter fields.
func validateStatusBlock(art *artifact.ParsedArtifact, maturity string) []Violation {
	var violations []Violation

	statusVal, ok := art.Frontmatter["status"]
	if !ok {
		violations = append(violations, Violation{
			Rule:     "bundle/status-required",
			File:     art.Filename,
			Message:  "status block is missing from frontmatter",
			Severity: "error",
		})
		return violations
	}

	status, ok := statusVal.(map[string]interface{})
	if !ok {
		violations = append(violations, Violation{
			Rule:     "bundle/status-required",
			File:     art.Filename,
			Message:  "status block is not a valid map",
			Severity: "error",
		})
		return violations
	}

	if m, ok := getStringField(status, "maturity"); !ok {
		violations = append(violations, Violation{
			Rule:     "bundle/maturity-required",
			File:     art.Filename,
			Message:  "status.maturity is required",
			Severity: "error",
		})
	} else if !maturityLevels[m] {
		violations = append(violations, Violation{
			Rule:     "bundle/maturity-enum",
			File:     art.Filename,
			Message:  fmt.Sprintf("status.maturity '%s' is not valid (allowed: idea, exploring, defined, ready)", m),
			Severity: "error",
		})
	}

	return violations
}

// validateNameFilenameConsistency checks that bundle.name matches the filename stem.
func validateNameFilenameConsistency(art *artifact.ParsedArtifact) []Violation {
	var violations []Violation

	bundleVal, ok := art.Frontmatter["bundle"]
	if !ok {
		return violations
	}
	bundle, ok := bundleVal.(map[string]interface{})
	if !ok {
		return violations
	}
	name, ok := getStringField(bundle, "name")
	if !ok {
		return violations
	}

	// Extract stem: strip .epic.bundle.md or .bundle.md
	stem := art.Filename
	if strings.HasSuffix(stem, ".epic.bundle.md") {
		stem = strings.TrimSuffix(stem, ".epic.bundle.md")
	} else if strings.HasSuffix(stem, ".bundle.md") {
		stem = strings.TrimSuffix(stem, ".bundle.md")
	}

	if stem != name {
		violations = append(violations, Violation{
			Rule:     "bundle/name-filename-mismatch",
			File:     art.Filename,
			Message:  fmt.Sprintf("bundle.name '%s' does not match filename stem '%s'", name, stem),
			Severity: "error",
		})
	}

	return violations
}

// validateVersionGatedUpdated checks that bundle.updated is present when version > 0.1.0.
func validateVersionGatedUpdated(art *artifact.ParsedArtifact) []Violation {
	var violations []Violation

	bundleVal, ok := art.Frontmatter["bundle"]
	if !ok {
		return violations
	}
	bundle, ok := bundleVal.(map[string]interface{})
	if !ok {
		return violations
	}

	version, ok := getStringField(bundle, "version")
	if !ok || version == "0.1.0" {
		return violations
	}

	// Version is beyond 0.1.0 — updated is required
	if _, ok := getStringField(bundle, "updated"); !ok {
		violations = append(violations, Violation{
			Rule:     "bundle/updated-required",
			File:     art.Filename,
			Message:  fmt.Sprintf("bundle.updated is required when version is beyond 0.1.0 (current: %s)", version),
			Severity: "error",
		})
	}

	return violations
}

// validateMaturityGates enforces progressive requirements based on status.maturity.
func validateMaturityGates(art *artifact.ParsedArtifact, maturity string) []Violation {
	var violations []Violation

	if maturity != "defined" && maturity != "ready" {
		return violations
	}

	// Frontmatter path requirements for defined maturity
	definedPaths := []string{
		"problem.summary", "problem.user_story", "solution.approach",
	}
	// Additional paths required only at ready
	readyPaths := []string{
		"problem.success_criteria", "solution.assumptions",
	}

	paths := definedPaths
	if maturity == "ready" {
		paths = append(paths, readyPaths...)
	}

	for _, path := range paths {
		if !resolveFrontmatterPath(art.Frontmatter, path) {
			violations = append(violations, Violation{
				Rule:     "bundle/maturity-gate",
				File:     art.Filename,
				Message:  fmt.Sprintf("%s is required at '%s' maturity", path, maturity),
				Severity: "error",
			})
		}
	}

	// Required sections at defined/ready
	sectionSet := make(map[string]bool)
	for _, s := range art.Sections {
		sectionSet[s] = true
	}
	for _, section := range matureSections {
		if !sectionSet[section] {
			violations = append(violations, Violation{
				Rule:     "bundle/maturity-section",
				File:     art.Filename,
				Message:  fmt.Sprintf("section '%s' is required at '%s' maturity", section, maturity),
				Severity: "error",
			})
		}
	}

	return violations
}

// validateEpicBlock checks epic-specific requirements when bundle.category is "epic".
func validateEpicBlock(art *artifact.ParsedArtifact) []Violation {
	var violations []Violation

	bundleVal, ok := art.Frontmatter["bundle"]
	if !ok {
		return violations
	}
	bundle, ok := bundleVal.(map[string]interface{})
	if !ok {
		return violations
	}

	cat, _ := getStringField(bundle, "category")
	if cat != "epic" {
		return violations
	}

	epicVal, ok := art.Frontmatter["epic"]
	if !ok {
		violations = append(violations, Violation{
			Rule:     "bundle/epic-required",
			File:     art.Filename,
			Message:  "epic block is required when bundle.category is 'epic'",
			Severity: "error",
		})
		return violations
	}

	epic, ok := epicVal.(map[string]interface{})
	if !ok {
		violations = append(violations, Violation{
			Rule:     "bundle/epic-required",
			File:     art.Filename,
			Message:  "epic block is not a valid map",
			Severity: "error",
		})
		return violations
	}

	// epic.id
	if id, ok := getStringField(epic, "id"); !ok {
		violations = append(violations, Violation{
			Rule:     "bundle/epic-id-required",
			File:     art.Filename,
			Message:  "epic.id is required",
			Severity: "error",
		})
	} else if !epicIDRe.MatchString(id) {
		violations = append(violations, Violation{
			Rule:     "bundle/epic-id-pattern",
			File:     art.Filename,
			Message:  fmt.Sprintf("epic.id '%s' must match pattern EPIC-[A-Z0-9-]+", id),
			Severity: "error",
		})
	}

	// epic.goal
	if _, ok := getStringField(epic, "goal"); !ok {
		violations = append(violations, Violation{
			Rule:     "bundle/epic-goal-required",
			File:     art.Filename,
			Message:  "epic.goal is required",
			Severity: "error",
		})
	}

	// epic.success_metric
	if _, ok := getStringField(epic, "success_metric"); !ok {
		violations = append(violations, Violation{
			Rule:     "bundle/epic-success-metric-required",
			File:     art.Filename,
			Message:  "epic.success_metric is required",
			Severity: "error",
		})
	}

	// epic.children (must be non-empty array)
	childrenVal, hasChildren := epic["children"]
	if !hasChildren {
		violations = append(violations, Violation{
			Rule:     "bundle/epic-children-required",
			File:     art.Filename,
			Message:  "epic.children is required",
			Severity: "error",
		})
	} else {
		children, ok := childrenVal.([]interface{})
		if !ok || len(children) == 0 {
			violations = append(violations, Violation{
				Rule:     "bundle/epic-children-empty",
				File:     art.Filename,
				Message:  "epic.children must contain at least one child bundle",
				Severity: "error",
			})
		}
	}

	return violations
}

// validatePlaceholderBan checks for TBD/TODO/FIXME/XXX/??? in problem.summary
// at defined/ready maturity.
func validatePlaceholderBan(art *artifact.ParsedArtifact, maturity string) []Violation {
	var violations []Violation

	if maturity != "defined" && maturity != "ready" {
		return violations
	}

	problemVal, ok := art.Frontmatter["problem"]
	if !ok {
		return violations
	}
	problem, ok := problemVal.(map[string]interface{})
	if !ok {
		return violations
	}

	summary, ok := getStringField(problem, "summary")
	if !ok {
		return violations
	}

	if placeholderRe.MatchString(summary) {
		violations = append(violations, Violation{
			Rule:     "bundle/placeholder-ban",
			File:     art.Filename,
			Message:  "problem.summary contains placeholder text (TBD/TODO/FIXME/XXX/???)",
			Severity: "error",
		})
	}

	return violations
}

// extractMaturity returns the status.maturity value or empty string.
func extractMaturity(art *artifact.ParsedArtifact) string {
	statusVal, ok := art.Frontmatter["status"]
	if !ok {
		return ""
	}
	status, ok := statusVal.(map[string]interface{})
	if !ok {
		return ""
	}
	m, _ := getStringField(status, "maturity")
	return m
}

// resolveFrontmatterPath checks if a dot-separated path exists in frontmatter.
// For example, "problem.summary" checks frontmatter["problem"]["summary"].
func resolveFrontmatterPath(fm map[string]interface{}, path string) bool {
	parts := strings.Split(path, ".")
	current := fm
	for i, part := range parts {
		val, ok := current[part]
		if !ok {
			return false
		}
		if i == len(parts)-1 {
			// Terminal — check it's non-empty
			switch v := val.(type) {
			case string:
				return strings.TrimSpace(v) != ""
			case []interface{}:
				return len(v) > 0
			default:
				return val != nil
			}
		}
		// Intermediate — must be a map
		next, ok := val.(map[string]interface{})
		if !ok {
			return false
		}
		current = next
	}
	return false
}

// getStringField extracts a string value from a map, returning ("", false) if
// the key is missing or the value is empty after trimming.
func getStringField(m map[string]interface{}, key string) (string, bool) {
	val, ok := m[key]
	if !ok {
		return "", false
	}
	s, ok := val.(string)
	if !ok || strings.TrimSpace(s) == "" {
		return "", false
	}
	return s, true
}
