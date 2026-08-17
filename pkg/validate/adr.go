package validate

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/backstop-ai/backstop-core/pkg/artifact"
	"github.com/backstop-ai/backstop-core/pkg/schema"
)

// adrNumberRe extracts ADR-NNNN from a filename.
var adrNumberRe = regexp.MustCompile(`^(ADR-\d{4})-`)

// ADR composes base validation with ADR-specific checks.
func ADR(art *artifact.ParsedArtifact, sch *schema.Schema) ValidationResult {
	base := Base(art, sch)
	var adrViolations []Violation

	// 1. Filename pattern
	filenameOK := false
	if sch.FilenamePattern != "" {
		re, err := regexp.Compile(sch.FilenamePattern)
		if err == nil {
			filenameOK = re.MatchString(art.Filename)
		}
		if !filenameOK {
			adrViolations = append(adrViolations, Violation{
				Rule:     "adr/filename-pattern",
				File:     art.Filename,
				Message:  fmt.Sprintf("filename does not match pattern %s", sch.FilenamePattern),
				Severity: "error",
			})
		}
	}

	// 2. Slug validation (only if filename matched the overall pattern)
	if filenameOK && sch.SlugPattern != "" {
		slug := extractSlug(art.Filename)
		if slug != "" {
			slugRe, err := regexp.Compile(sch.SlugPattern)
			valid := err == nil && slugRe.MatchString(slug)
			if valid && (sch.SlugMinLength > 0 && len(slug) < sch.SlugMinLength) {
				valid = false
			}
			if valid && (sch.SlugMaxLength > 0 && len(slug) > sch.SlugMaxLength) {
				valid = false
			}
			if !valid {
				adrViolations = append(adrViolations, Violation{
					Rule:     "adr/invalid-slug",
					File:     art.Filename,
					Message:  fmt.Sprintf("slug '%s' does not conform to D-078 spec", slug),
					Severity: "error",
				})
			}
		}
	}

	// 3. Number/filename consistency
	if filenameOK {
		m := adrNumberRe.FindStringSubmatch(art.Filename)
		if m != nil {
			fileNumber := m[1]
			if metaNumber, ok := art.Metadata["number"]; ok && metaNumber != fileNumber {
				adrViolations = append(adrViolations, Violation{
					Rule:     "adr/number-mismatch",
					File:     art.Filename,
					Message:  fmt.Sprintf("metadata Number '%s' does not match filename '%s'", metaNumber, fileNumber),
					Severity: "error",
				})
			}
		}
	}

	// 4. Title/number consistency
	if filenameOK {
		m := adrNumberRe.FindStringSubmatch(art.Filename)
		if m != nil && art.Title != "" {
			expectedPrefix := m[1] + ":"
			if !strings.HasPrefix(art.Title, expectedPrefix) {
				adrViolations = append(adrViolations, Violation{
					Rule:     "adr/title-number-mismatch",
					File:     art.Filename,
					Message:  fmt.Sprintf("title does not start with '%s'", expectedPrefix),
					Severity: "error",
				})
			}
		}
	}

	// 5. Status enum
	if len(sch.StatusEnum) > 0 {
		status := art.Metadata["status"]
		valid := false
		for _, s := range sch.StatusEnum {
			if s == status {
				valid = true
				break
			}
		}
		if !valid {
			adrViolations = append(adrViolations, Violation{
				Rule:     "adr/invalid-status",
				File:     art.Filename,
				Message:  fmt.Sprintf("status '%s' is not valid (allowed: %v)", status, sch.StatusEnum),
				Severity: "error",
			})
		}
	}

	// 6. Extension-specific required metadata (Deciders, Decisions)
	for _, key := range sch.ExtensionMetadata {
		found := false
		for k, v := range art.Metadata {
			if strings.EqualFold(k, key) && strings.TrimSpace(v) != "" {
				found = true
				break
			}
		}
		if !found {
			adrViolations = append(adrViolations, Violation{
				Rule:     "adr/metadata-required",
				File:     art.Filename,
				Message:  fmt.Sprintf("required metadata key '%s' is missing or empty", key),
				Severity: "error",
			})
		}
	}

	// 7. schema_version/artifact-type cross-check
	if sv, ok := art.Metadata["schema_version"]; ok && sv != "" {
		parts := strings.SplitN(sv, "/", 2)
		if len(parts) == 2 && sch.ArtifactType != "" && parts[0] != sch.ArtifactType {
			adrViolations = append(adrViolations, Violation{
				Rule:     "adr/schema-version-mismatch",
				File:     art.Filename,
				Message:  fmt.Sprintf("Schema-Version type '%s' does not match artifact type '%s'", parts[0], sch.ArtifactType),
				Severity: "error",
			})
		}
	}

	combined := make([]Violation, 0, len(base.Violations)+len(adrViolations))
	combined = append(combined, base.Violations...)
	combined = append(combined, adrViolations...)
	return ValidationResult{Violations: combined}
}

// extractSlug pulls the slug portion from an ADR filename.
// ADR-0001-my-slug.adr.md → my-slug
func extractSlug(filename string) string {
	// Strip prefix: ADR-NNNN-
	if len(filename) < 10 {
		return ""
	}
	rest := filename[9:] // after "ADR-NNNN-"
	// Strip suffix: .adr.md, read from the shared layout table (ISSUE-124).
	//
	// ★ THE LENGTH IS DERIVED, NOT WRITTEN. This used to close with `rest[:len(rest)-7]`,
	// where 7 was len(".adr.md") as a bare integer — a SECOND private copy of the same
	// fact, in a form no string scan can ever see. Converting only the HasSuffix call
	// would leave that copy behind and look complete.
	//
	// The ok is handled rather than discarded: a zero-value KindLayout's Extension is
	// EMPTY, HasSuffix against "" is always true, and `rest[:len(rest)-0]` would then
	// return the whole remainder INCLUDING its extension.
	adrLayout, ok := artifact.LayoutFor(artifact.KindADR)
	if !ok {
		return ""
	}
	if strings.HasSuffix(rest, adrLayout.Extension) {
		return rest[:len(rest)-len(adrLayout.Extension)]
	}
	return ""
}
