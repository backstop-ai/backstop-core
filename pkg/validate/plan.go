package validate

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/bmanson/backstop-core/pkg/artifact"
	"github.com/bmanson/backstop-core/pkg/schema"
)

var (
	planIDRe     = regexp.MustCompile(`^PLAN-(SPEC|ISSUE)-\d{3}$`)
	planFileRe   = regexp.MustCompile(`^PLAN-[A-Z]+-[0-9]+-[a-z][a-z0-9]*(-[a-z0-9]+)*\.plan\.yml$`)
	specIDRe     = regexp.MustCompile(`^(SPEC|ISSUE)-\d{3}$`)
	planStatuses = map[string]bool{
		"draft": true, "ready": true, "implementing": true, "completed": true,
	}
)

// Plan validates a pure YAML plan artifact. Plans do NOT extend the base
// artifact schema — they have their own top-level fields (plan_id, spec_id,
// status, created) and are machine-consumed, not human-readable markdown.
// Enforces D-080 (agent-bounded tasks) and D-081 (file exclusivity).
func Plan(art *artifact.ParsedArtifact, _ *schema.Schema) ValidationResult {
	var violations []Violation

	// 1. Filename pattern
	if !planFileRe.MatchString(art.Filename) {
		violations = append(violations, Violation{
			Rule:     "plan/filename-pattern",
			File:     art.Filename,
			Message:  fmt.Sprintf("filename must match PLAN-*-NNN-slug.plan.yml pattern"),
			Severity: "error",
		})
	}

	// 2. plan_id — required, must match pattern
	planID := getFrontmatterString(art, "plan_id")
	if planID == "" {
		violations = append(violations, Violation{
			Rule:     "plan/plan-id-required",
			File:     art.Filename,
			Message:  "plan_id is required",
			Severity: "error",
		})
	} else if !planIDRe.MatchString(planID) {
		violations = append(violations, Violation{
			Rule:     "plan/plan-id-pattern",
			File:     art.Filename,
			Message:  fmt.Sprintf("plan_id '%s' must match PLAN-SPEC-NNN or PLAN-ISSUE-NNN", planID),
			Severity: "error",
		})
	}

	// 3. plan_id / filename consistency
	if planID != "" && planFileRe.MatchString(art.Filename) {
		if !strings.HasPrefix(art.Filename, planID+"-") {
			violations = append(violations, Violation{
				Rule:     "plan/id-filename-mismatch",
				File:     art.Filename,
				Message:  fmt.Sprintf("filename must start with plan_id '%s-'", planID),
				Severity: "error",
			})
		}
	}

	// 4. spec_id — required, must match SPEC-NNN or ISSUE-NNN
	specID := getFrontmatterString(art, "spec_id")
	if specID == "" {
		violations = append(violations, Violation{
			Rule:     "plan/spec-id-required",
			File:     art.Filename,
			Message:  "spec_id is required (SPEC-NNN or ISSUE-NNN)",
			Severity: "error",
		})
	} else if !specIDRe.MatchString(specID) {
		violations = append(violations, Violation{
			Rule:     "plan/spec-id-pattern",
			File:     art.Filename,
			Message:  fmt.Sprintf("spec_id '%s' must match SPEC-NNN or ISSUE-NNN", specID),
			Severity: "error",
		})
	}

	// 5. status — required, must be in enum
	status := getFrontmatterString(art, "status")
	if status == "" {
		violations = append(violations, Violation{
			Rule:     "plan/status-required",
			File:     art.Filename,
			Message:  "status is required (draft, ready, implementing, completed)",
			Severity: "error",
		})
	} else if !planStatuses[status] {
		violations = append(violations, Violation{
			Rule:     "plan/invalid-status",
			File:     art.Filename,
			Message:  fmt.Sprintf("status '%s' is not valid (draft, ready, implementing, completed)", status),
			Severity: "error",
		})
	}

	// 6. created — required, must match YYYY-MM-DD
	created := getFrontmatterString(art, "created")
	if created == "" {
		violations = append(violations, Violation{
			Rule:     "plan/created-required",
			File:     art.Filename,
			Message:  "created date is required (YYYY-MM-DD)",
			Severity: "error",
		})
	} else if !dateRe.MatchString(created) {
		violations = append(violations, Violation{
			Rule:     "plan/created-format",
			File:     art.Filename,
			Message:  fmt.Sprintf("created '%s' must match YYYY-MM-DD format", created),
			Severity: "error",
		})
	}

	// 7. coverage_threshold — optional, integer 0-100 when present (F5)
	if ctVal, ok := art.Frontmatter["coverage_threshold"]; ok {
		valid := false
		switch ct := ctVal.(type) {
		case int:
			valid = ct >= 0 && ct <= 100
		case float64:
			valid = ct >= 0 && ct <= 100
		}
		if !valid {
			violations = append(violations, Violation{
				Rule:     "plan/coverage-threshold-range",
				File:     art.Filename,
				Message:  fmt.Sprintf("coverage_threshold must be an integer 0-100, got %v", ctVal),
				Severity: "error",
			})
		}
	}

	// 8. Optional string field type checks (F14)
	for _, field := range []string{"spec_version", "target_repo", "target_module", "test_command", "notes"} {
		if val, ok := art.Frontmatter[field]; ok {
			if _, ok := val.(string); !ok {
				violations = append(violations, Violation{
					Rule:     "plan/field-type",
					File:     art.Filename,
					Message:  fmt.Sprintf("'%s' must be a string, got %T", field, val),
					Severity: "error",
				})
			}
		}
	}

	// 9. Phases validation (D-080 + D-081)
	violations = append(violations, validatePhases(art)...)

	return ValidationResult{Violations: violations}
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

	// Validate depends_on references exist (F7)
	for _, t := range allTasks {
		for _, dep := range t.dependsOn {
			if !seenTaskIDs[dep] {
				violations = append(violations, Violation{
					Rule:     "plan/unknown-dependency",
					File:     art.Filename,
					Message:  fmt.Sprintf("task '%s' depends on unknown task '%s'", t.id, dep),
					Severity: "error",
				})
			}
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

	// Cycle detection (F15): if task depends on itself, it's in a cycle
	for i := 0; i < n; i++ {
		if depends[i][i] {
			violations = append(violations, Violation{
				Rule:     "plan/dependency-cycle",
				File:     filename,
				Message:  fmt.Sprintf("task '%s' is part of a dependency cycle", tasks[i].id),
				Severity: "error",
			})
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
