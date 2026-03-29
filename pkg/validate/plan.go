package validate

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/bmanson/backstop-core/pkg/artifact"
	"github.com/bmanson/backstop-core/pkg/schema"
)

var (
	planNumberRe = regexp.MustCompile(`^(PLAN-[A-Z]+-\d+)-`)
	implementsRe = regexp.MustCompile(`^(SPEC|ISSUE)-\d{3}$`)
)

// Plan composes base validation with plan-specific checks
// including D-080 (agent-bounded tasks) and D-081 (file exclusivity).
func Plan(art *artifact.ParsedArtifact, sch *schema.Schema) ValidationResult {
	base := Base(art, sch)
	var planViolations []Violation

	// 1. Filename pattern
	filenameOK := false
	if sch.FilenamePattern != "" {
		re, err := regexp.Compile(sch.FilenamePattern)
		if err == nil {
			filenameOK = re.MatchString(art.Filename)
		}
		if !filenameOK {
			planViolations = append(planViolations, Violation{
				Rule:     "plan/filename-pattern",
				File:     art.Filename,
				Message:  fmt.Sprintf("filename does not match pattern %s", sch.FilenamePattern),
				Severity: "error",
			})
		}
	}

	// 2. Number/filename consistency
	if filenameOK {
		m := planNumberRe.FindStringSubmatch(art.Filename)
		if m != nil {
			fileNumber := m[1]
			if metaNumber, ok := art.Metadata["number"]; ok && metaNumber != fileNumber {
				planViolations = append(planViolations, Violation{
					Rule:     "plan/number-mismatch",
					File:     art.Filename,
					Message:  fmt.Sprintf("metadata number '%s' does not match filename '%s'", metaNumber, fileNumber),
					Severity: "error",
				})
			}
		}
	}

	// 3. Status enum
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
			planViolations = append(planViolations, Violation{
				Rule:     "plan/invalid-status",
				File:     art.Filename,
				Message:  fmt.Sprintf("status '%s' is not valid (allowed: %v)", status, sch.StatusEnum),
				Severity: "error",
			})
		}
	}

	// 4. schema_version/artifact-type cross-check
	if sv, ok := art.Metadata["schema_version"]; ok && sv != "" {
		parts := strings.SplitN(sv, "/", 2)
		if len(parts) == 2 && sch.ArtifactType != "" && parts[0] != sch.ArtifactType {
			planViolations = append(planViolations, Violation{
				Rule:     "plan/schema-version-mismatch",
				File:     art.Filename,
				Message:  fmt.Sprintf("schema_version type '%s' does not match artifact type '%s'", parts[0], sch.ArtifactType),
				Severity: "error",
			})
		}
	}

	// 5. Implements — required reference to SPEC-NNN or ISSUE-NNN
	planViolations = append(planViolations, validatePlanImplements(art)...)

	// 6. Phases validation (D-080 + D-081)
	planViolations = append(planViolations, validatePhases(art)...)

	combined := make([]Violation, 0, len(base.Violations)+len(planViolations))
	combined = append(combined, base.Violations...)
	combined = append(combined, planViolations...)
	return ValidationResult{Violations: combined}
}

// validatePlanImplements checks that the plan declares what artifact it implements.
func validatePlanImplements(art *artifact.ParsedArtifact) []Violation {
	var violations []Violation

	implVal, ok := art.Frontmatter["implements"]
	if !ok {
		violations = append(violations, Violation{
			Rule:     "plan/implements-required",
			File:     art.Filename,
			Message:  "implements field is required (SPEC-NNN or ISSUE-NNN)",
			Severity: "error",
		})
		return violations
	}

	impl, ok := implVal.(string)
	if !ok || strings.TrimSpace(impl) == "" {
		violations = append(violations, Violation{
			Rule:     "plan/implements-required",
			File:     art.Filename,
			Message:  "implements must be a non-empty string",
			Severity: "error",
		})
		return violations
	}

	if !implementsRe.MatchString(impl) {
		violations = append(violations, Violation{
			Rule:     "plan/implements-pattern",
			File:     art.Filename,
			Message:  fmt.Sprintf("implements '%s' must match SPEC-NNN or ISSUE-NNN", impl),
			Severity: "error",
		})
	}

	return violations
}

// planTask is an internal representation of a task extracted from frontmatter.
type planTask struct {
	id        string
	files     []string
	dependsOn []string
	phaseID   string
}

// validatePhases checks the phases array, enforces D-080 (agent-bounded tasks)
// and D-081 (parallel-eligible file exclusivity).
func validatePhases(art *artifact.ParsedArtifact) []Violation {
	var violations []Violation

	phasesVal, ok := art.Frontmatter["phases"]
	if !ok {
		violations = append(violations, Violation{
			Rule:     "plan/phases-required",
			File:     art.Filename,
			Message:  "phases array is missing from frontmatter",
			Severity: "error",
		})
		return violations
	}

	phases, ok := phasesVal.([]interface{})
	if !ok {
		violations = append(violations, Violation{
			Rule:     "plan/phases-required",
			File:     art.Filename,
			Message:  "phases is not a valid array",
			Severity: "error",
		})
		return violations
	}

	if len(phases) == 0 {
		violations = append(violations, Violation{
			Rule:     "plan/phases-empty",
			File:     art.Filename,
			Message:  "phases array must contain at least one phase",
			Severity: "error",
		})
		return violations
	}

	var allTasks []planTask
	seenPhaseIDs := make(map[string]bool)
	seenTaskIDs := make(map[string]bool)

	for i, item := range phases {
		phase, ok := item.(map[string]interface{})
		if !ok {
			violations = append(violations, Violation{
				Rule:     "plan/phase-format",
				File:     art.Filename,
				Message:  fmt.Sprintf("phases[%d] is not a valid map", i),
				Severity: "error",
			})
			continue
		}

		// Phase ID
		phaseID := ""
		if idVal, ok := phase["id"]; ok {
			if s, ok := idVal.(string); ok {
				phaseID = s
			}
		}
		switch {
		case phaseID == "":
			violations = append(violations, Violation{
				Rule:     "plan/phase-id-required",
				File:     art.Filename,
				Message:  fmt.Sprintf("phases[%d] is missing 'id'", i),
				Severity: "error",
			})
		case seenPhaseIDs[phaseID]:
			violations = append(violations, Violation{
				Rule:     "plan/phase-id-duplicate",
				File:     art.Filename,
				Message:  fmt.Sprintf("duplicate phase id '%s'", phaseID),
				Severity: "error",
			})
		default:
			seenPhaseIDs[phaseID] = true
		}

		// Phase name
		if nameVal, ok := phase["name"]; !ok {
			violations = append(violations, Violation{
				Rule:     "plan/phase-name-required",
				File:     art.Filename,
				Message:  fmt.Sprintf("phases[%d] is missing 'name'", i),
				Severity: "error",
			})
		} else if name, ok := nameVal.(string); ok && strings.TrimSpace(name) == "" {
			violations = append(violations, Violation{
				Rule:     "plan/phase-name-required",
				File:     art.Filename,
				Message:  fmt.Sprintf("phases[%d] 'name' is empty", i),
				Severity: "error",
			})
		}

		// Phase tasks
		tasksVal, hasTasks := phase["tasks"]
		if !hasTasks {
			violations = append(violations, Violation{
				Rule:     "plan/phase-tasks-required",
				File:     art.Filename,
				Message:  fmt.Sprintf("phases[%d] is missing 'tasks'", i),
				Severity: "error",
			})
			continue
		}

		tasks, ok := tasksVal.([]interface{})
		if !ok || len(tasks) == 0 {
			violations = append(violations, Violation{
				Rule:     "plan/phase-tasks-empty",
				File:     art.Filename,
				Message:  fmt.Sprintf("phases[%d] must have at least one task", i),
				Severity: "error",
			})
			continue
		}

		for j, taskItem := range tasks {
			task, ok := taskItem.(map[string]interface{})
			if !ok {
				violations = append(violations, Violation{
					Rule:     "plan/task-format",
					File:     art.Filename,
					Message:  fmt.Sprintf("phases[%d].tasks[%d] is not a valid map", i, j),
					Severity: "error",
				})
				continue
			}

			pt := planTask{phaseID: phaseID}
			taskLabel := fmt.Sprintf("phases[%d].tasks[%d]", i, j)

			// Task ID
			if idVal, ok := task["id"]; ok {
				if s, ok := idVal.(string); ok {
					pt.id = s
				}
			}
			switch {
			case pt.id == "":
				violations = append(violations, Violation{
					Rule:     "plan/task-id-required",
					File:     art.Filename,
					Message:  fmt.Sprintf("%s is missing 'id'", taskLabel),
					Severity: "error",
				})
			case seenTaskIDs[pt.id]:
				violations = append(violations, Violation{
					Rule:     "plan/task-id-duplicate",
					File:     art.Filename,
					Message:  fmt.Sprintf("duplicate task id '%s'", pt.id),
					Severity: "error",
				})
			default:
				seenTaskIDs[pt.id] = true
			}

			// D-080: title required
			if titleVal, ok := task["title"]; !ok {
				violations = append(violations, Violation{
					Rule:     "plan/task-title-required",
					File:     art.Filename,
					Message:  fmt.Sprintf("%s is missing 'title'", taskLabel),
					Severity: "error",
				})
			} else if title, ok := titleVal.(string); ok && strings.TrimSpace(title) == "" {
				violations = append(violations, Violation{
					Rule:     "plan/task-title-required",
					File:     art.Filename,
					Message:  fmt.Sprintf("%s 'title' is empty", taskLabel),
					Severity: "error",
				})
			}

			// D-080: description required (agent needs clear context)
			if descVal, ok := task["description"]; !ok {
				violations = append(violations, Violation{
					Rule:     "plan/task-description-required",
					File:     art.Filename,
					Message:  fmt.Sprintf("%s is missing 'description' (D-080: agent-bounded tasks require clear context)", taskLabel),
					Severity: "error",
				})
			} else if desc, ok := descVal.(string); ok && strings.TrimSpace(desc) == "" {
				violations = append(violations, Violation{
					Rule:     "plan/task-description-required",
					File:     art.Filename,
					Message:  fmt.Sprintf("%s 'description' is empty", taskLabel),
					Severity: "error",
				})
			}

			// D-080: files required (agent needs to know what it's touching)
			filesVal, hasFiles := task["files"]
			if !hasFiles {
				violations = append(violations, Violation{
					Rule:     "plan/task-files-required",
					File:     art.Filename,
					Message:  fmt.Sprintf("%s is missing 'files' (D-080: agent-bounded tasks must declare file ownership)", taskLabel),
					Severity: "error",
				})
			} else {
				files, ok := filesVal.([]interface{})
				if !ok || len(files) == 0 {
					violations = append(violations, Violation{
						Rule:     "plan/task-files-empty",
						File:     art.Filename,
						Message:  fmt.Sprintf("%s 'files' must list at least one file", taskLabel),
						Severity: "error",
					})
				} else {
					for _, f := range files {
						if s, ok := f.(string); ok {
							pt.files = append(pt.files, s)
						}
					}
				}
			}

			// D-080: claims required (ties task to spec)
			claimsVal, hasClaims := task["claims"]
			if !hasClaims {
				violations = append(violations, Violation{
					Rule:     "plan/task-claims-required",
					File:     art.Filename,
					Message:  fmt.Sprintf("%s is missing 'claims' (D-080: tasks must reference spec claims)", taskLabel),
					Severity: "error",
				})
			} else {
				claims, ok := claimsVal.([]interface{})
				if !ok || len(claims) == 0 {
					violations = append(violations, Violation{
						Rule:     "plan/task-claims-empty",
						File:     art.Filename,
						Message:  fmt.Sprintf("%s 'claims' must reference at least one CLM", taskLabel),
						Severity: "error",
					})
				}
			}

			// depends_on required (can be empty array)
			depsVal, hasDeps := task["depends_on"]
			if !hasDeps {
				violations = append(violations, Violation{
					Rule:     "plan/task-depends-on-required",
					File:     art.Filename,
					Message:  fmt.Sprintf("%s is missing 'depends_on'", taskLabel),
					Severity: "error",
				})
			} else if deps, ok := depsVal.([]interface{}); ok {
				for _, d := range deps {
					if s, ok := d.(string); ok {
						pt.dependsOn = append(pt.dependsOn, s)
					}
				}
			}

			allTasks = append(allTasks, pt)
		}
	}

	// D-081: File exclusivity for parallel-eligible tasks
	violations = append(violations, checkFileExclusivity(art.Filename, allTasks)...)

	return violations
}

// checkFileExclusivity implements D-081: parallel-eligible tasks must have
// disjoint file sets. Two tasks are parallel-eligible if neither depends
// (transitively) on the other.
func checkFileExclusivity(filename string, tasks []planTask) []Violation {
	var violations []Violation

	// Build task index
	taskIdx := make(map[string]int)
	for i, t := range tasks {
		if t.id != "" {
			taskIdx[t.id] = i
		}
	}

	// Build transitive dependency closure
	// depends[i] = set of all task indices that task i transitively depends on
	n := len(tasks)
	depends := make([]map[int]bool, n)
	for i := range depends {
		depends[i] = make(map[int]bool)
	}

	// Direct dependencies
	for i, t := range tasks {
		for _, depID := range t.dependsOn {
			if j, ok := taskIdx[depID]; ok {
				depends[i][j] = true
			}
		}
	}

	// Transitive closure (Floyd-Warshall style)
	for k := 0; k < n; k++ {
		for i := 0; i < n; i++ {
			if depends[i][k] {
				for j := range depends[k] {
					depends[i][j] = true
				}
			}
		}
	}

	// Check all parallel-eligible pairs
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			// Skip if either depends on the other (sequential)
			if depends[i][j] || depends[j][i] {
				continue
			}

			// Parallel-eligible — check file disjointness
			filesI := make(map[string]bool)
			for _, f := range tasks[i].files {
				filesI[f] = true
			}
			for _, f := range tasks[j].files {
				if filesI[f] {
					violations = append(violations, Violation{
						Rule:     "plan/file-exclusivity",
						File:     filename,
						Message:  fmt.Sprintf("D-081: parallel-eligible tasks '%s' and '%s' both touch file '%s'", tasks[i].id, tasks[j].id, f),
						Severity: "error",
					})
				}
			}
		}
	}

	return violations
}
