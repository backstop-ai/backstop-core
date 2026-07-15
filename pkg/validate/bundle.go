package validate

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/bmanson/backstop-core/pkg/artifact"
	"github.com/bmanson/backstop-core/pkg/schema"
)

var (
	bundleNameRe     = regexp.MustCompile(`^[a-z0-9-]+$`)                     // nosemgrep: go.core.no-global-mutable-state — immutable compiled-regex singleton, package idiom
	semverRe         = regexp.MustCompile(`^\d+\.\d+\.\d+$`)                  // nosemgrep: go.core.no-global-mutable-state — immutable compiled-regex singleton, package idiom
	dateRe           = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)             // nosemgrep: go.core.no-global-mutable-state — immutable compiled-regex singleton, package idiom
	epicIDRe         = regexp.MustCompile(`^EPIC-[A-Z0-9-]+$`)                // nosemgrep: go.core.no-global-mutable-state — immutable compiled-regex singleton, package idiom
	placeholderRe    = regexp.MustCompile(`(?i)\b(TBD|TODO|FIXME|XXX)\b|\?{3}`) // nosemgrep: go.core.no-global-mutable-state — immutable compiled-regex singleton, package idiom
	bundleCategories = map[string]bool{ // nosemgrep: go.core.no-global-mutable-state — immutable enum lookup, package idiom
		"feature": true, "service": true, "integration": true,
		"recipe": true, "infrastructure": true, "tool": true, "epic": true,
	}
	maturityLevels = map[string]bool{ // nosemgrep: go.core.no-global-mutable-state — immutable enum lookup, package idiom
		"idea": true, "exploring": true, "defined": true, "ready": true,
		// Success terminal + retirement terminal states (ISSUE-031).
		"delivered": true, "replaced": true, "canceled": true, "deprecated": true,
	}
	// Sections required at defined/ready maturity
	matureSections = []string{ // nosemgrep: go.core.no-global-mutable-state — immutable section list, package idiom
		"Current Thinking", "Draft Requirements",
		"Draft Design Decisions", "Spec Seeds", "Version History",
	}
)

// Bundle composes base validation with bundle-specific checks
// including maturity-gated progressive requirements, epic validation,
// and placeholder bans.
func Bundle(art *artifact.ParsedArtifact, sch *schema.Schema) ValidationResult {
	base := Base(art, sch)
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
	violations = append(violations, validateStatusBlock(art)...)

	// 4b. Retirement fields — replaced-by required+typed when maturity==replaced;
	// reason optional for canceled/deprecated (ISSUE-031 DQ-2).
	violations = append(violations, validateRetirementFields(art, maturity, "bundle")...)

	// 5. Name/filename consistency
	if filenameOK {
		violations = append(violations, validateNameFilenameConsistency(art)...)
	}

	// 6. Version-gated: bundle.updated required when version > 0.1.0
	violations = append(violations, validateVersionGatedUpdated(art)...)

	// Terminal-state exemption (ISSUE-031 CLM-014): a retired bundle (delivered is
	// success-terminal; replaced/canceled/deprecated are retirement-terminal) is
	// not held to the defined/ready maturity gates or the requirements[] gate.
	// Core bundle-block + status-block + replaced-by checks above still apply.
	if !isTerminalStatus(maturity) {
		// 7. Maturity-gated requirements
		violations = append(violations, validateMaturityGates(art, maturity)...)

		// 8. Epic validation
		violations = append(violations, validateEpicBlock(art)...)

		// 9. Placeholder ban in problem.summary at defined/ready
		violations = append(violations, validatePlaceholderBan(art, maturity)...)

		// 10. Formal requirements array (required from defined onward)
		violations = append(violations, validateBundleRequirements(art, maturity)...)
	}

	combined := make([]Violation, 0, len(base.Violations)+len(violations))
	combined = append(combined, base.Violations...)
	combined = append(combined, violations...)
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
func validateStatusBlock(art *artifact.ParsedArtifact) []Violation {
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
			Message:  fmt.Sprintf("status.maturity '%s' is not valid (allowed: idea, exploring, defined, ready, delivered, replaced, canceled, deprecated)", m),
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
	// Strip BUNDLE-NNN- prefix if present (e.g., "BUNDLE-007-baseline" → "baseline")
	// This supports the numbered bundle convention from artifact new
	if idx := strings.Index(stem, "-"); idx > 0 {
		prefix := stem[:idx]
		if prefix == "BUNDLE" {
			rest := stem[idx+1:] // "007-baseline"
			if dashIdx := strings.Index(rest, "-"); dashIdx > 0 {
				stem = rest[dashIdx+1:] // "baseline"
			}
		}
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

	cat, ok := getStringField(bundle, "category")
	if !ok || cat != "epic" {
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

// Bundle requirement ID pattern: REQ-NNN
var bundleReqIDRe = regexp.MustCompile(`^REQ-\d{3}$`) // nosemgrep: go.core.no-global-mutable-state — immutable compiled-regex singleton, package idiom

// validateBundleRequirements checks the formal requirements array.
// Required from defined maturity onward. Each requirement needs id (REQ-NNN) and text.
func validateBundleRequirements(art *artifact.ParsedArtifact, maturity string) []Violation {
	var violations []Violation

	// REQ-005: requirements[] is required from `defined` through `delivered`.
	// `delivered` is NOT terminal (isTerminalStatus excludes it), so this
	// function already runs for it; extending the condition only ADDS the
	// requirement, never relaxing defined/ready. The replaced/canceled/deprecated
	// terminal exemption at bundle.go's Bundle() gate keeps them out entirely.
	requiresDefined := maturity == "defined" || maturity == "ready" || maturity == "delivered"

	reqsVal, ok := art.Frontmatter["requirements"]
	if !ok {
		if requiresDefined {
			violations = append(violations, Violation{
				Rule:     "bundle/requirements-required",
				File:     art.Filename,
				Message:  fmt.Sprintf("requirements array is required at '%s' maturity", maturity),
				Severity: "error",
			})
		}
		return violations
	}

	reqs, ok := reqsVal.([]interface{})
	if !ok {
		violations = append(violations, Violation{
			Rule:     "bundle/requirements-format",
			File:     art.Filename,
			Message:  "requirements is not a valid array",
			Severity: "error",
		})
		return violations
	}

	if len(reqs) == 0 && requiresDefined {
		violations = append(violations, Violation{
			Rule:     "bundle/requirements-required",
			File:     art.Filename,
			Message:  fmt.Sprintf("requirements array must not be empty at '%s' maturity", maturity),
			Severity: "error",
		})
		return violations
	}

	seen := make(map[string]bool)
	for i, item := range reqs {
		label := fmt.Sprintf("requirements[%d]", i)
		req, ok := item.(map[string]interface{})
		if !ok {
			violations = append(violations, Violation{
				Rule:     "bundle/requirement-format",
				File:     art.Filename,
				Message:  fmt.Sprintf("%s is not a valid map", label),
				Severity: "error",
			})
			continue
		}

		// id
		if id, ok := getStringField(req, "id"); !ok {
			violations = append(violations, Violation{
				Rule:     "bundle/requirement-id-required",
				File:     art.Filename,
				Message:  fmt.Sprintf("%s is missing 'id'", label),
				Severity: "error",
			})
		} else if !bundleReqIDRe.MatchString(id) {
			violations = append(violations, Violation{
				Rule:     "bundle/requirement-id-pattern",
				File:     art.Filename,
				Message:  fmt.Sprintf("%s id '%s' must match pattern REQ-NNN", label, id),
				Severity: "error",
			})
		} else {
			if seen[id] {
				violations = append(violations, Violation{
					Rule:     "bundle/requirement-id-duplicate",
					File:     art.Filename,
					Message:  fmt.Sprintf("duplicate requirement id '%s'", id),
					Severity: "error",
				})
			}
			seen[id] = true
		}

		// text
		if _, ok := getStringField(req, "text"); !ok {
			violations = append(violations, Violation{
				Rule:     "bundle/requirement-text-required",
				File:     art.Filename,
				Message:  fmt.Sprintf("%s is missing 'text'", label),
				Severity: "error",
			})
		}

		// REQ-004: per-REQ version + version-log well-formedness.
		violations = append(violations, validateReqVersionLog(art.Filename, label, req)...)
	}

	return violations
}

// validateReqVersionLog validates REQ-004's per-REQ version and version log for a
// single bundle requirement map. The strict `semverRe` (M.M.P, no prerelease) is
// the single source of truth shared with the supports pin. When `versions:` is
// absent the effective log is the implicit single entry {version, text} and no
// further ceremony is required (CLM-015). When present it must be non-empty, each
// entry well-formed, strictly ascending by NUMERIC semver with no duplicates, and
// the top-level version:/text: must equal the newest (semver-max) entry.
func validateReqVersionLog(filename, label string, req map[string]interface{}) []Violation {
	var violations []Violation

	version, hasVersion := getStringField(req, "version")
	switch {
	case !hasVersion:
		violations = append(violations, Violation{
			Rule:     "bundle/requirement-version-required",
			File:     filename,
			Message:  fmt.Sprintf("%s is missing 'version' (per-REQ semver MAJOR.MINOR.PATCH)", label),
			Severity: "error",
		})
	case !semverRe.MatchString(version):
		violations = append(violations, Violation{
			Rule:     "bundle/requirement-version-format",
			File:     filename,
			Message:  fmt.Sprintf("%s version '%s' must be strict semver (X.Y.Z, no prerelease/build)", label, version),
			Severity: "error",
		})
	}

	// versions: is optional; absent means the implicit single-entry log (valid).
	versionsVal, present := req["versions"]
	if !present {
		return violations
	}

	versions, ok := versionsVal.([]interface{})
	if !ok {
		violations = append(violations, Violation{
			Rule:     "bundle/requirement-versions-format",
			File:     filename,
			Message:  fmt.Sprintf("%s 'versions' must be a list of {version, text} entries", label),
			Severity: "error",
		})
		return violations
	}

	// An explicit but empty versions: [] is an error (asymmetry with absent).
	if len(versions) == 0 {
		violations = append(violations, Violation{
			Rule:     "bundle/requirement-versions-empty",
			File:     filename,
			Message:  fmt.Sprintf("%s 'versions' is present but empty (a version log must have at least one entry)", label),
			Severity: "error",
		})
		return violations
	}

	entryVersions := make([]string, 0, len(versions))
	entryTexts := make([]string, 0, len(versions))
	seen := make(map[string]bool)
	wellFormed := true
	for j, item := range versions {
		entry, ok := item.(map[string]interface{})
		if !ok {
			violations = append(violations, Violation{
				Rule:     "bundle/requirement-versions-entry-format",
				File:     filename,
				Message:  fmt.Sprintf("%s versions[%d] is not a valid map", label, j),
				Severity: "error",
			})
			wellFormed = false
			continue
		}

		ev, hasEV := getStringField(entry, "version")
		if !hasEV || !semverRe.MatchString(ev) {
			violations = append(violations, Violation{
				Rule:     "bundle/requirement-versions-entry-format",
				File:     filename,
				Message:  fmt.Sprintf("%s versions[%d] version '%s' must be strict semver (X.Y.Z)", label, j, ev),
				Severity: "error",
			})
			wellFormed = false
		}
		if _, hasText := getStringField(entry, "text"); !hasText {
			violations = append(violations, Violation{
				Rule:     "bundle/requirement-versions-entry-text",
				File:     filename,
				Message:  fmt.Sprintf("%s versions[%d] has empty or missing 'text'", label, j),
				Severity: "error",
			})
			wellFormed = false
		}
		if hasEV && seen[ev] {
			violations = append(violations, Violation{
				Rule:     "bundle/requirement-versions-duplicate",
				File:     filename,
				Message:  fmt.Sprintf("%s versions[%d] duplicate version '%s'", label, j, ev),
				Severity: "error",
			})
			wellFormed = false
		}
		if hasEV {
			seen[ev] = true
		}
		entryVersions = append(entryVersions, ev)
		entryTexts = append(entryTexts, stringField(entry, "text"))
	}

	// Strictly monotonically ascending by NUMERIC semver (1.10.0 > 1.9.0).
	for j := 1; j < len(entryVersions); j++ {
		prev, cur := entryVersions[j-1], entryVersions[j]
		if !semverRe.MatchString(prev) || !semverRe.MatchString(cur) {
			continue // malformed entries already flagged
		}
		if compareSemver(cur, prev) <= 0 {
			violations = append(violations, Violation{
				Rule:     "bundle/requirement-versions-nonmonotonic",
				File:     filename,
				Message:  fmt.Sprintf("%s versions must be strictly ascending by semver ('%s' does not exceed '%s')", label, cur, prev),
				Severity: "error",
			})
		}
	}

	// Top-level version:/text: must equal the newest (semver-max) entry. Only
	// cross-check when the log entries are well-formed enough to have a newest.
	if wellFormed && len(entryVersions) > 0 {
		newestIdx := 0
		for j := 1; j < len(entryVersions); j++ {
			if compareSemver(entryVersions[j], entryVersions[newestIdx]) > 0 {
				newestIdx = j
			}
		}
		if hasVersion && version != entryVersions[newestIdx] {
			violations = append(violations, Violation{
				Rule:     "bundle/requirement-version-not-newest",
				File:     filename,
				Message:  fmt.Sprintf("%s top-level version '%s' must equal the newest log entry version '%s'", label, version, entryVersions[newestIdx]),
				Severity: "error",
			})
		}
		// Compare meaning, not bytes: ParseFile clips the trailing newline of a
		// folded `>` scalar only when it is the last line before the closing `---`,
		// so byte-equal-authored texts can differ by surrounding whitespace by
		// position. Normalize both sides (CLM-033 stays meaning-equality).
		if strings.TrimSpace(stringField(req, "text")) != strings.TrimSpace(entryTexts[newestIdx]) {
			violations = append(violations, Violation{
				Rule:     "bundle/requirement-text-not-newest",
				File:     filename,
				Message:  fmt.Sprintf("%s top-level text must equal the newest log entry's text", label),
				Severity: "error",
			})
		}
	}

	return violations
}

// compareSemver compares two strict M.M.P semver strings numerically, returning
// -1, 0, or 1. Callers pass only strings already matched by semverRe, so the
// components parse cleanly; any parse anomaly compares as 0.
func compareSemver(a, b string) int {
	ap := strings.SplitN(a, ".", 3)
	bp := strings.SplitN(b, ".", 3)
	if len(ap) != 3 || len(bp) != 3 {
		return 0
	}
	for i := 0; i < 3; i++ {
		an := atoiSafe(ap[i])
		bn := atoiSafe(bp[i])
		if an < bn {
			return -1
		}
		if an > bn {
			return 1
		}
	}
	return 0
}

// atoiSafe parses a non-negative integer component, returning 0 on any anomaly.
func atoiSafe(s string) int {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0
		}
		n = n*10 + int(r-'0')
	}
	return n
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
	m, ok := getStringField(status, "maturity")
	if !ok {
		return ""
	}
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

// stringField returns the trimmed string value for key, or "" when the key is
// missing, empty, or non-string. It is the single-value companion to
// getStringField for callers that do not need the presence bool.
func stringField(m map[string]interface{}, key string) string {
	s, ok := getStringField(m, key)
	if !ok {
		return ""
	}
	return s
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
