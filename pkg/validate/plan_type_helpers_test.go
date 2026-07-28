package validate_test

import (
"github.com/backstop-ai/backstop-core/pkg/artifact"
)

// makeTask builds a task map with all required D-080 fields plus the type field.
func makeTask(id, taskType string, files, claims, deps []string) map[string]interface{} {
filesI := make([]interface{}, len(files))
for i, f := range files {
filesI[i] = f
}
claimsI := make([]interface{}, len(claims))
for i, c := range claims {
claimsI[i] = c
}
depsI := make([]interface{}, len(deps))
for i, d := range deps {
depsI[i] = d
}
return map[string]interface{}{
"id":          id,
"type":        taskType,
"title":       id + " title",
"description": id + " description",
"files":       filesI,
"claims":      claimsI,
"depends_on":  depsI,
}
}

// makePhase wraps tasks in a phase map.
func makePhase(id, name string, tasks ...map[string]interface{}) map[string]interface{} {
tasksI := make([]interface{}, len(tasks))
for i, t := range tasks {
tasksI[i] = t
}
return map[string]interface{}{
"id":    id,
"name":  name,
"tasks": tasksI,
}
}

// validTypedPlanArtifact returns a ParsedArtifact with properly typed tasks
// following the full TDD cycle: setup → test → implementation → verification.
func validTypedPlanArtifact() *artifact.ParsedArtifact {
return &artifact.ParsedArtifact{
Filename: "PLAN-SPEC-001-test-plan.plan.yml",
Metadata: map[string]string{
"plan_id": "PLAN-SPEC-001",
"spec_id": "SPEC-001",
"created": "2026-03-30",
"status":  "draft",
},
Frontmatter: map[string]interface{}{
"plan_id": "PLAN-SPEC-001",
"spec_id": "SPEC-001",
"created": "2026-03-30",
"status":  "draft",
"phases": []interface{}{
makePhase("phase-1", "Setup and TDD Cycle",
makeTask("setup-1", "setup", []string{"pkg/setup.go"}, []string{"CLM-001"}, []string{}),
makeTask("test-1", "test", []string{"pkg/foo_test.go"}, []string{"CLM-002"}, []string{"setup-1"}),
makeTask("impl-1", "implementation", []string{"pkg/foo.go"}, []string{"CLM-002"}, []string{"test-1"}),
makeTask("verify-1", "verification", []string{"pkg/foo_test.go"}, []string{"CLM-003"}, []string{"impl-1"}),
),
},
},
Sections: []string{},
}
}
