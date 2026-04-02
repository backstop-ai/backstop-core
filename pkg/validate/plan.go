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
	validTaskTypes = map[string]bool{
		"setup": true, "test": true, "implementation": true,
		"verification": true, "refactor": true, "documentation": true,
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
			valid = ct >= 0 && ct <= 100 && ct == float64(int(ct))
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
	taskType  string
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
			// Task type (SPEC-002: REQ-001)
			if typeVal, ok := task["type"]; ok {
				if s, ok := typeVal.(string); ok {
					pt.taskType = s
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

	// SPEC-002: Task type validation (REQ-001)
	typeMap := make(map[string]string)
	for _, t := range allTasks {
		if t.id != "" {
			typeMap[t.id] = t.taskType
		}
	}
	for _, t := range allTasks {
		switch {
		case t.taskType == "":
			violations = append(violations, Violation{
				Rule:     "plan/task-type-required",
				File:     art.Filename,
				Message:  fmt.Sprintf("task '%s' is missing required 'type' field", t.id),
				Severity: "error",
			})
		case !validTaskTypes[t.taskType]:
			violations = append(violations, Violation{
				Rule:     "plan/task-type-invalid",
				File:     art.Filename,
				Message:  fmt.Sprintf("task '%s' has invalid type '%s' (valid: setup, test, implementation, verification, refactor, documentation)", t.id, t.taskType),
				Severity: "error",
			})
		}
	}

	// SPEC-002: TDD enforcement (REQ-002)
	violations = append(violations, validateTDD(art.Filename, allTasks, typeMap)...)

	// SPEC-002: Gate cadence (REQ-003)
	violations = append(violations, validateGateCadence(art.Filename, phases, typeMap)...)

	// SPEC-002: Verification dependency validation (REQ-006)
	violations = append(violations, validateVerificationDeps(art.Filename, allTasks, typeMap)...)

	// SPEC-002: Refactor dependency validation (REQ-005)
	violations = append(violations, validateRefactorDeps(art.Filename, allTasks, typeMap)...)

	// SPEC-002: Test task dependency validation (REQ-010)
	violations = append(violations, validateTestTaskDeps(art.Filename, allTasks, typeMap)...)

	// SPEC-002: Final phase verification (REQ-004)
	violations = append(violations, validateFinalPhase(art.Filename, phases, allTasks, typeMap)...)

	// SPEC-002: Phase-level parallel file exclusivity (REQ-011)
	violations = append(violations, checkPhaseFileExclusivity(art.Filename, phases, allTasks)...)

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

// validateFinalPhase enforces REQ-004: the final phase must contain verification
// tasks covering every category of work the plan performs.
func validateFinalPhase(filename string, phases []interface{}, allTasks []planTask, typeMap map[string]string) []Violation {
	var violations []Violation
	if len(phases) == 0 {
		return violations
	}

	// Get last phase
	lastPhase, ok := phases[len(phases)-1].(map[string]interface{})
	if !ok {
		return violations
	}

	// Check final phase has at least one verification task
	tasksVal, ok := lastPhase["tasks"]
	if !ok {
		return violations
	}
	tasks, ok := tasksVal.([]interface{})
	if !ok {
		return violations
	}

	lastPhaseID := ""
	if id, ok := lastPhase["id"]; ok {
		if s, ok := id.(string); ok {
			lastPhaseID = s
		}
	}

	// Collect verification task files in final phase
	var verifyFiles []string
	hasVerification := false
	for _, taskItem := range tasks {
		task, ok := taskItem.(map[string]interface{})
		if !ok {
			continue
		}
		taskID := ""
		if id, ok := task["id"]; ok {
			if s, ok := id.(string); ok {
				taskID = s
			}
		}
		if typeMap[taskID] == "verification" {
			hasVerification = true
			if filesVal, ok := task["files"]; ok {
				if files, ok := filesVal.([]interface{}); ok {
					for _, f := range files {
						if s, ok := f.(string); ok {
							verifyFiles = append(verifyFiles, s)
						}
					}
				}
			}
		}
	}

	if !hasVerification {
		violations = append(violations, Violation{
			Rule:     "plan/final-phase-no-verification",
			File:     filename,
			Message:  fmt.Sprintf("final phase '%s' must contain at least one verification task", lastPhaseID),
			Severity: "error",
		})
		return violations
	}

	// Collect all categories from all tasks across entire plan
	requiredCategories := make(map[string]bool)
	for _, t := range allTasks {
		for _, f := range t.files {
			cat := fileCategory(f)
			if cat != "" {
				requiredCategories[cat] = true
			}
		}
	}

	// Collect categories covered by final phase verification tasks
	coveredCategories := make(map[string]bool)
	for _, f := range verifyFiles {
		cat := fileCategory(f)
		if cat != "" {
			coveredCategories[cat] = true
		}
	}

	// Check each required category is covered
	for cat := range requiredCategories {
		if !coveredCategories[cat] {
			violations = append(violations, Violation{
				Rule:     "plan/final-phase-missing-category",
				File:     filename,
				Message:  fmt.Sprintf("final phase '%s' verification tasks do not cover '%s' category", lastPhaseID, cat),
				Severity: "error",
			})
		}
	}

	return violations
}

// fileCategory maps a file path to a work category based on extension.
func fileCategory(path string) string {
	artifactExts := []string{".spec.md", ".plan.yml", ".adr.md", ".bundle.md", ".issue.md", ".standard.md"}
	for _, ext := range artifactExts {
		if strings.HasSuffix(path, ext) {
			return "artifact"
		}
	}
	if strings.HasSuffix(path, ".go") {
		return "code"
	}
	// Other extensions (e.g. .md for docs) don't map to a required category
	return ""
}

// checkPhaseFileExclusivity enforces REQ-011: parallel-eligible phases must
// have disjoint file sets. Two phases are parallel-eligible if no task in
// one phase transitively depends on any task in the other phase.
func checkPhaseFileExclusivity(filename string, phases []interface{}, allTasks []planTask) []Violation {
	var violations []Violation
	if len(phases) < 2 {
		return violations
	}

	// Build phase info: ID and set of all files
	type phaseInfo struct {
		id      string
		files   map[string]bool
		taskIDs map[string]bool
	}

	var phaseInfos []phaseInfo
	for _, item := range phases {
		phase, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		pi := phaseInfo{
			files:   make(map[string]bool),
			taskIDs: make(map[string]bool),
		}
		if id, ok := phase["id"]; ok {
			if s, ok := id.(string); ok {
				pi.id = s
			}
		}
		if tasksVal, ok := phase["tasks"]; ok {
			if tasks, ok := tasksVal.([]interface{}); ok {
				for _, taskItem := range tasks {
					task, ok := taskItem.(map[string]interface{})
					if !ok {
						continue
					}
					if id, ok := task["id"]; ok {
						if s, ok := id.(string); ok {
							pi.taskIDs[s] = true
						}
					}
					if filesVal, ok := task["files"]; ok {
						if files, ok := filesVal.([]interface{}); ok {
							for _, f := range files {
								if s, ok := f.(string); ok {
									pi.files[s] = true
								}
							}
						}
					}
				}
			}
		}
		phaseInfos = append(phaseInfos, pi)
	}

	// Build task-to-phase index
	taskToPhase := make(map[string]int)
	for i, pi := range phaseInfos {
		for tid := range pi.taskIDs {
			taskToPhase[tid] = i
		}
	}

	// Build phase dependency graph using task dependencies
	// Phase A depends on Phase B if any task in A depends on any task in B
	numPhases := len(phaseInfos)
	phaseDeps := make([]map[int]bool, numPhases)
	for i := range phaseDeps {
		phaseDeps[i] = make(map[int]bool)
	}
	for _, t := range allTasks {
		tPhase, ok := taskToPhase[t.id]
		if !ok {
			continue
		}
		for _, dep := range t.dependsOn {
			dPhase, ok := taskToPhase[dep]
			if !ok {
				continue
			}
			if tPhase != dPhase {
				phaseDeps[tPhase][dPhase] = true
			}
		}
	}

	// Transitive closure
	for k := 0; k < numPhases; k++ {
		for i := 0; i < numPhases; i++ {
			if phaseDeps[i][k] {
				for j := range phaseDeps[k] {
					phaseDeps[i][j] = true
				}
			}
		}
	}

	// Check parallel-eligible phase pairs
	for i := 0; i < numPhases; i++ {
		for j := i + 1; j < numPhases; j++ {
			if phaseDeps[i][j] || phaseDeps[j][i] {
				continue // sequential — skip
			}
			// Parallel-eligible — check file disjointness
			for f := range phaseInfos[i].files {
				if phaseInfos[j].files[f] {
					violations = append(violations, Violation{
						Rule:     "plan/phase-file-exclusivity",
						File:     filename,
						Message:  fmt.Sprintf("parallel-eligible phases '%s' and '%s' both touch file '%s'", phaseInfos[i].id, phaseInfos[j].id, f),
						Severity: "error",
					})
				}
			}
		}
	}

	return violations
}

// validateTDD enforces REQ-002: every implementation task must directly depend
// on at least one test task.
func validateTDD(filename string, tasks []planTask, typeMap map[string]string) []Violation {
	var violations []Violation
	for _, t := range tasks {
		if t.taskType != "implementation" {
			continue
		}
		hasTestDep := false
		for _, dep := range t.dependsOn {
			if typeMap[dep] == "test" {
				hasTestDep = true
				break
			}
		}
		if !hasTestDep {
			violations = append(violations, Violation{
				Rule:     "plan/tdd-impl-requires-test",
				File:     filename,
				Message:  fmt.Sprintf("implementation task '%s' must directly depend on at least one test task (TDD enforcement)", t.id),
				Severity: "error",
			})
		}
	}
	return violations
}

// validateGateCadence enforces REQ-003: every phase with implementation or
// refactor tasks must also contain at least one verification task.
func validateGateCadence(filename string, phases []interface{}, typeMap map[string]string) []Violation {
	var violations []Violation
	for _, item := range phases {
		phase, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		tasksVal, ok := phase["tasks"]
		if !ok {
			continue
		}
		tasks, ok := tasksVal.([]interface{})
		if !ok {
			continue
		}

		hasCodeWork := false
		hasVerification := false
		phaseID := ""
		if id, ok := phase["id"]; ok {
			if s, ok := id.(string); ok {
				phaseID = s
			}
		}

		for _, taskItem := range tasks {
			task, ok := taskItem.(map[string]interface{})
			if !ok {
				continue
			}
			taskID := ""
			if id, ok := task["id"]; ok {
				if s, ok := id.(string); ok {
					taskID = s
				}
			}
			tt := typeMap[taskID]
			if tt == "implementation" || tt == "refactor" {
				hasCodeWork = true
			}
			if tt == "verification" {
				hasVerification = true
			}
		}

		if hasCodeWork && !hasVerification {
			violations = append(violations, Violation{
				Rule:     "plan/gate-cadence-missing",
				File:     filename,
				Message:  fmt.Sprintf("phase '%s' has implementation/refactor tasks but no verification task (gate cadence enforcement)", phaseID),
				Severity: "error",
			})
		}
	}
	return violations
}

// validateVerificationDeps enforces REQ-006: every verification task must
// depend on at least one implementation or refactor task.
func validateVerificationDeps(filename string, tasks []planTask, typeMap map[string]string) []Violation {
	var violations []Violation
	for _, t := range tasks {
		if t.taskType != "verification" {
			continue
		}
		hasImplOrRefactor := false
		for _, dep := range t.dependsOn {
			dt := typeMap[dep]
			if dt == "implementation" || dt == "refactor" {
				hasImplOrRefactor = true
				break
			}
		}
		if !hasImplOrRefactor {
			violations = append(violations, Violation{
				Rule:     "plan/verification-requires-impl",
				File:     filename,
				Message:  fmt.Sprintf("verification task '%s' must depend on at least one implementation or refactor task", t.id),
				Severity: "error",
			})
		}
	}
	return violations
}

// validateRefactorDeps enforces REQ-005: refactor tasks may only depend on
// implementation, refactor, or test tasks. Dependencies on setup, documentation,
// or verification are rejected.
func validateRefactorDeps(filename string, tasks []planTask, typeMap map[string]string) []Violation {
	var violations []Violation
	validRefactorDeps := map[string]bool{
		"implementation": true,
		"refactor":       true,
		"test":           true,
	}
	for _, t := range tasks {
		if t.taskType != "refactor" {
			continue
		}
		for _, dep := range t.dependsOn {
			dt := typeMap[dep]
			if dt != "" && !validRefactorDeps[dt] {
				violations = append(violations, Violation{
					Rule:     "plan/refactor-invalid-dependency",
					File:     filename,
					Message:  fmt.Sprintf("refactor task '%s' depends on %s task '%s' (allowed: implementation, refactor, test)", t.id, dt, dep),
					Severity: "error",
				})
			}
		}
	}
	return violations
}

// validateTestTaskDeps enforces REQ-010: test tasks may only depend on
// setup, test, or verification tasks. Dependencies on implementation,
// refactor, or documentation are rejected.
func validateTestTaskDeps(filename string, tasks []planTask, typeMap map[string]string) []Violation {
	var violations []Violation
	validTestDeps := map[string]bool{
		"setup":        true,
		"test":         true,
		"verification": true,
	}
	for _, t := range tasks {
		if t.taskType != "test" {
			continue
		}
		for _, dep := range t.dependsOn {
			dt := typeMap[dep]
			if dt != "" && !validTestDeps[dt] {
				violations = append(violations, Violation{
					Rule:     "plan/test-invalid-dependency",
					File:     filename,
					Message:  fmt.Sprintf("test task '%s' depends on %s task '%s' (allowed: setup, test, verification)", t.id, dt, dep),
					Severity: "error",
				})
			}
		}
	}
	return violations
}
