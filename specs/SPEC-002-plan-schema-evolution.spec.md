---
title: "SPEC-002: Plan Schema Evolution — Task Types, TDD Enforcement, Gate Cadence"
number: SPEC-002
created: "2026-03-31"
status: draft
schema_version: spec/v1
spec_version: 1.0.0

implementation:
  summary: >
    Extend the plan artifact schema and validator to enforce task type
    classification, strict TDD ordering (implementation tasks must directly
    depend on test tasks), gate verification cadence (verification tasks
    required in implementation phases), and comprehensive relevant verification
    in the final phase. These additions make the plan validator the mechanical
    enforcement layer for development discipline — agents cannot produce plans
    that skip tests, omit verification, or bundle implementation without
    preceding test work. All existing D-080 and D-081 enforcement is preserved.
  package: pkg/validate

verification:
  level: unit
  test_command: go test ./pkg/validate/ -run TestPlan -race -coverprofile=cover.out
  coverage_threshold: 90

requirements:
  - id: REQ-001
    text: >
      Every task in a plan must include a type field with one of six valid
      values: setup, test, implementation, verification, refactor, documentation.
      Missing or invalid type is a validation error.
    supports: agent-definitions:REQ-004

  - id: REQ-002
    text: >
      Every implementation task must directly depend on at least one test task
      via its depends_on field. "Directly" means at least one entry in
      depends_on must reference a task whose type is "test". Two implementation
      tasks in a row (an implementation task whose depends_on contains only
      non-test tasks) is a validation failure.
    supports: agent-definitions:REQ-005

  - id: REQ-003
    text: >
      Every phase that contains at least one implementation or refactor task
      must also contain at least one verification task. A phase with
      code-changing work but no verification is a validation failure.
    supports: agent-definitions:REQ-006

  - id: REQ-004
    text: >
      The final phase in the phases array must contain verification tasks
      covering every category of work the plan performs. Categories are
      determined by file extensions across all tasks in the plan: .go files
      mean code verification is required, artifact files (.spec.md, .plan.yml,
      .adr.md, .bundle.md, .issue.md, .standard.md) mean artifact verification
      is required. The validator collects all file extensions from all tasks,
      maps them to categories, then checks that the final phase contains at
      least one verification task whose files include each required category.
      A final phase with no verification tasks is rejected. A final phase
      missing a required category is rejected.
    supports: agent-definitions:REQ-007

  - id: REQ-005
    text: >
      Refactor tasks may depend on implementation tasks, other refactor tasks,
      or test tasks. Refactor tasks do not require a direct test dependency
      (they modify existing code, not create new functionality). A refactor
      task depending on a setup, documentation, or verification task is a
      validation error — refactors operate on code, not scaffolding or docs.

  - id: REQ-006
    text: >
      Verification tasks must depend on at least one implementation or refactor
      task. A verification task with no implementation or refactor dependency
      is a validation error — there is nothing to verify.

  - id: REQ-007
    text: >
      Setup and documentation tasks have no task-type dependency constraints.
      They may depend on any task type or have an empty depends_on.

  - id: REQ-008
    text: >
      All existing plan validation rules must continue to function: plan_id
      pattern, spec_id pattern, filename consistency, status enum, created
      date, coverage_threshold range, phase structure (id, name, tasks),
      D-080 task fields (id, title, description, files, claims, depends_on),
      D-081 file exclusivity for parallel-eligible tasks, dependency reference
      validation, and cycle detection.

  - id: REQ-009
    text: >
      The plan JSON schema (artifacts/plan/v1/schema.json) must be updated
      to document the task type field, its enum values, and the TDD/gate
      enforcement rules.

  - id: REQ-010
    text: >
      Test tasks may only depend on setup, test, or verification tasks. A
      test task depending on implementation, refactor, or documentation is
      a validation error — tests must be writable before any code-changing
      work exists. Verification dependencies are allowed because gates are
      non-code-changing sequencing points, not TDD violations.

  - id: REQ-011
    text: >
      Phases without dependency chains between them are parallel-eligible.
      Parallel-eligible phases must have disjoint file sets across all their
      tasks combined. This extends D-081 from task-level to phase-level:
      if phase A and phase B have no transitive dependency between any of
      their tasks, the union of file sets in phase A must not overlap with
      the union of file sets in phase B. Overlapping files in parallel-eligible
      phases is a validation error.

claims:
  - id: CLM-001
    requirement: REQ-001
    text: Tasks with valid type field pass validation
    tests:
      - TestPlan_TaskTypeValid

  - id: CLM-002
    requirement: REQ-001
    text: Tasks with missing type field produce validation error
    tests:
      - TestPlan_TaskTypeMissing

  - id: CLM-003
    requirement: REQ-001
    text: Tasks with invalid type value produce validation error
    tests:
      - TestPlan_TaskTypeInvalid

  - id: CLM-004
    requirement: REQ-002
    text: Implementation task with direct test dependency passes validation
    tests:
      - TestPlan_TDD_ImplDependsOnTest

  - id: CLM-005
    requirement: REQ-002
    text: Implementation task without direct test dependency fails validation
    tests:
      - TestPlan_TDD_ImplWithoutTestDependency

  - id: CLM-006
    requirement: REQ-002
    text: Two implementation tasks in a row fails validation
    tests:
      - TestPlan_TDD_TwoImplInRow

  - id: CLM-007
    requirement: REQ-002
    text: Implementation depending on setup only (no test) fails validation
    tests:
      - TestPlan_TDD_ImplDependsOnSetupOnly

  - id: CLM-008
    requirement: REQ-003
    text: Phase with implementation and verification tasks passes
    tests:
      - TestPlan_GateCadence_PhaseWithVerification

  - id: CLM-009
    requirement: REQ-003
    text: Phase with implementation but no verification fails
    tests:
      - TestPlan_GateCadence_PhaseWithoutVerification

  - id: CLM-030
    requirement: REQ-003
    text: Phase with refactor but no verification fails
    tests:
      - TestPlan_GateCadence_RefactorPhaseWithoutVerification

  - id: CLM-010
    requirement: REQ-004
    text: Plan with verification in final phase passes
    tests:
      - TestPlan_FinalPhase_HasVerification

  - id: CLM-011
    requirement: REQ-004
    text: Plan without verification in final phase fails
    tests:
      - TestPlan_FinalPhase_NoVerification

  - id: CLM-012
    requirement: REQ-005
    text: Refactor task depending on implementation task passes
    tests:
      - TestPlan_Refactor_DependsOnImpl

  - id: CLM-013
    requirement: REQ-005
    text: Refactor task depending on another refactor task passes
    tests:
      - TestPlan_Refactor_DependsOnRefactor

  - id: CLM-014
    requirement: REQ-006
    text: Verification task depending on implementation passes
    tests:
      - TestPlan_Verification_DependsOnImpl

  - id: CLM-032
    requirement: REQ-006
    text: Verification task depending on refactor task passes
    tests:
      - TestPlan_Verification_DependsOnRefactor

  - id: CLM-015
    requirement: REQ-006
    text: Verification task with no impl or refactor dependency fails
    tests:
      - TestPlan_Verification_NoDependency

  - id: CLM-016
    requirement: REQ-007
    text: Setup task with empty depends_on passes
    tests:
      - TestPlan_Setup_NoDependencyConstraint

  - id: CLM-017
    requirement: REQ-007
    text: Documentation task depending on any type passes
    tests:
      - TestPlan_Documentation_NoDependencyConstraint

  - id: CLM-018
    requirement: REQ-008
    text: >
      Representative existing plan validation rules still function: plan_id
      pattern enforcement, D-081 file exclusivity, and cycle detection
    tests:
      - TestPlan_ExistingRules_PlanIDPattern
      - TestPlan_ExistingRules_FileExclusivity
      - TestPlan_ExistingRules_CycleDetection

  - id: CLM-019
    requirement: REQ-009
    text: Plan schema JSON includes task type field and enum values
    tests:
      - TestPlan_SchemaIncludesTaskType

  - id: CLM-020
    requirement: REQ-010
    text: Test task depending on setup task passes
    tests:
      - TestPlan_TestTask_DependsOnSetup

  - id: CLM-021
    requirement: REQ-010
    text: Test task depending on another test task passes
    tests:
      - TestPlan_TestTask_DependsOnTest

  - id: CLM-022
    requirement: REQ-010
    text: Test task depending on implementation task fails
    tests:
      - TestPlan_TestTask_DependsOnImplFails

  - id: CLM-033
    requirement: REQ-010
    text: Test task depending on refactor task fails
    tests:
      - TestPlan_TestTask_DependsOnRefactorFails

  - id: CLM-034
    requirement: REQ-010
    text: Test task depending on verification task passes
    tests:
      - TestPlan_TestTask_DependsOnVerification

  - id: CLM-035
    requirement: REQ-010
    text: Test task depending on documentation task fails
    tests:
      - TestPlan_TestTask_DependsOnDocsFails

  - id: CLM-023
    requirement: REQ-011
    text: Parallel-eligible phases with disjoint file sets pass
    tests:
      - TestPlan_PhaseParallel_DisjointFiles

  - id: CLM-024
    requirement: REQ-011
    text: Parallel-eligible phases with overlapping file sets fail
    tests:
      - TestPlan_PhaseParallel_OverlappingFiles

  - id: CLM-031
    requirement: REQ-005
    text: Refactor task depending on test task passes
    tests:
      - TestPlan_Refactor_DependsOnTest

  - id: CLM-025
    requirement: REQ-005
    text: Refactor task depending on setup task fails
    tests:
      - TestPlan_Refactor_DependsOnSetupFails

  - id: CLM-028
    requirement: REQ-005
    text: Refactor task depending on documentation task fails
    tests:
      - TestPlan_Refactor_DependsOnDocsFails

  - id: CLM-029
    requirement: REQ-005
    text: Refactor task depending on verification task fails
    tests:
      - TestPlan_Refactor_DependsOnVerificationFails

  - id: CLM-026
    requirement: REQ-004
    text: Final phase with verification covering all work categories passes
    tests:
      - TestPlan_FinalPhase_ComprehensiveVerification

  - id: CLM-027
    requirement: REQ-004
    text: Final phase with verification missing a work category fails
    tests:
      - TestPlan_FinalPhase_IncompleteVerification

contracts:
  - file: pkg/validate/plan.go
    provides:
      - name: Plan
        kind: function
        signature: "func Plan(art *artifact.ParsedArtifact, _ *schema.Schema) ValidationResult"
        notes: "Extended with task type validation, TDD enforcement, gate cadence checks"
    consumes:
      - source: pkg/artifact
        name: ParsedArtifact
        kind: type
      - source: pkg/schema
        name: Schema
        kind: type
---

# SPEC-002: Plan Schema Evolution — Task Types, TDD Enforcement, Gate Cadence

## Overview

The plan validator (pkg/validate/plan.go) currently enforces D-080 (agent-bounded
tasks with file scope, claims, and dependencies) and D-081 (disjoint file sets for
parallel-eligible tasks). These are structural rules — they ensure plans are well-
formed but say nothing about development discipline.

This spec adds three enforcement layers:

1. **Task type classification** — every task declares what kind of work it is
2. **TDD ordering** — implementation must follow tests, mechanically enforced
3. **Gate cadence** — verification tasks must accompany implementation work

Together these make the plan validator the single mechanical authority for
development discipline. An agent cannot produce a plan that skips tests, omits
verification, or bundles implementation without preceding test work. The validator
rejects it before any code is written.

## Requirements

Requirements are defined in frontmatter (REQ-001 through REQ-011).

### Task Type Classification

Six task types, each with different validation rules:

| Type | Purpose | Dependency constraints |
|------|---------|----------------------|
| `setup` | Infrastructure, scaffolding | None — may depend on anything or nothing |
| `test` | Write test files (TDD red phase) | May depend on setup, test, or verification tasks. Must not depend on implementation, refactor, or documentation. |
| `implementation` | Write new code (TDD green phase) | Must directly depend on at least one test task |
| `verification` | Run gates (lint, test, coverage) | Must depend on at least one implementation or refactor task |
| `refactor` | Modify existing code | May depend on implementation, refactor, or test tasks. Setup/docs/verification dependency is an error. |
| `documentation` | Write docs, comments | None — may depend on anything or nothing |

### TDD Enforcement

The core rule: **every implementation task must directly depend on at least one
test task.** This is strict — not transitive, not phase-level. The depends_on
array must contain the ID of a task whose type is "test".

This means the TDD cycle is:

```
setup → test → implementation → verification
                     ↓
               refactor (optional)
                     ↓
               verification
```

Two implementation tasks in a row — where an implementation task's depends_on
contains only setup, implementation, refactor, verification, or documentation
tasks — is a validation failure.

### Gate Cadence

Every phase with implementation work must include verification. The final phase
must include comprehensive verification covering every category of work the plan
performs — if the plan has implementation tasks, the final phase must verify code
(test, lint, build); if it modifies artifacts, the final phase must verify artifacts.

### Phase-Level Parallel File Exclusivity

Extending D-081 from task-level to phase-level: phases without dependency chains
between them are parallel-eligible. The union of all file sets across all tasks
in one parallel-eligible phase must be disjoint from the union in another. This
prevents file conflicts when phases execute concurrently.

## Implementation

### Changes to plan.go

The validator adds six new validation passes after existing checks:

1. **Task type validation** — iterate all tasks, verify type field exists and
   contains a valid enum value.

2. **TDD enforcement** — for each task with type "implementation", verify that
   at least one entry in depends_on references a task with type "test". For each
   task with type "test", verify that no entry in depends_on references a task
   with type "implementation".

3. **Gate cadence** — for each phase, if any task has type "implementation" or
   "refactor", verify at least one task in the same phase has type "verification".
   For the last phase in the array, collect all file extensions across the entire
   plan, map to categories (code vs artifact), and verify the final phase has
   verification tasks covering each category.

4. **Verification dependency** — for each task with type "verification", verify
   at least one entry in depends_on references a task with type "implementation"
   or "refactor".

5. **Refactor dependency validation** — for each task with type "refactor", verify
   all entries in depends_on reference tasks with type "implementation", "refactor",
   or "test". Dependencies on setup, documentation, or verification are rejected.

6. **Phase-level parallel file exclusivity** — identify parallel-eligible phases
   (no transitive dependency between any of their tasks), compute the union of
   file sets per phase, and verify disjoint file sets across parallel-eligible
   phase pairs. Extends the existing task-level D-081 check.

### Changes to plan schema

Update artifacts/plan/v1/schema.json to add:

```json
"task_required_keys": ["id", "type", "title", "description", "files", "claims", "depends_on"],
"task_type_enum": ["setup", "test", "implementation", "verification", "refactor", "documentation"]
```

### Internal changes

The `planTask` struct gains a `taskType string` field. A `taskTypeMap` helper
builds a lookup from task ID to task type, called once per plan and used by
TDD and gate validation passes.

```go
var validTaskTypes = map[string]bool{
    "setup": true, "test": true, "implementation": true,
    "verification": true, "refactor": true, "documentation": true,
}
```

## Verification

Verification is defined in frontmatter. Unit-level verification with 90% coverage
threshold on the plan validator.

Claims CLM-001 through CLM-035 map each requirement to specific test functions.

## Sharp Edges

- **Existing plans must be updated.** Plans without the type field will fail
  validation. There is no backward compatibility — the bundle's REQ-013
  explicitly states this. The existing PLAN-SPEC-001 must be updated with
  task types before it passes the new validator.

- **Refactor vs implementation ambiguity.** The distinction matters for TDD:
  implementation requires a test dependency, refactor does not. The rule of
  thumb: if it's the first time writing the code, it's implementation. If it's
  modifying existing code, it's refactor. The planner agent must get this right.

- **Phase-level verification vs task-level verification.** The gate cadence
  rule checks phases, not individual tasks. A phase with 5 implementation
  tasks and 1 verification task at the end is valid. The smart cadence pattern
  from mechsuit (verification every 2-3 implementation phases) is an agent
  instruction, not a validator rule — enforcing exact cadence intervals is
  too rigid.

- **Final phase identification.** "Final phase" means the last entry in the
  phases array. If a plan's logical final phase isn't last in the array, the
  validator will check the wrong phase. Plan authors must order phases correctly.

- **Comprehensive verification is gameable.** REQ-004 requires verification
  covering every category of work performed. A malicious or lazy planner could
  add a no-op verification task that claims to cover a category without actually
  doing anything. The validator checks that verification tasks exist and reference
  the right categories, but cannot verify the tasks are substantive — that's the
  reviewer's job.

- **Phase-level parallelism depends on correct dependency declarations.** If
  tasks that should be sequential have no depends_on relationship, the phase-level
  file exclusivity check treats them as parallel-eligible. Incorrect dependency
  declarations can lead to false positives (rejecting valid plans) or false
  negatives (allowing conflicting plans).

- **Test-to-implementation dependency direction.** REQ-010 prevents test tasks
  from depending on implementation tasks. This catches inverted TDD — writing
  implementation first and tests after. However, a test task can depend on a
  setup task that's listed alongside implementation tasks in the same phase.
  The validator checks direct dependency types, not phase membership.

## References

- Bundle: agent-definitions (plan schema evolution seed)
- ADR-0018: Workflow State Machine
- D-080: Agent-bounded tasks
- D-081: Disjoint file sets for parallel tasks
- SPEC-001: Standards Compiler (reference for spec format)
- Mechsuit SPEC-021/022: Plan validation pipeline (TDD and gate OPA policies)
