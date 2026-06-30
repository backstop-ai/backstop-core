---
name: project-dispatch-consumer-edges
description: For consumer-impl tasks, depends_on must include the impl task of every symbol they import, not just their own test task — validator does not catch this
metadata:
  type: project
---

When a plan's implementation task consumes symbols produced by other tasks (e.g. a
dispatch/wiring task that imports a leaf package's type, a new runner method, or a new
manifest field), its `depends_on` must include the *implementation* task that produces
each symbol — not merely the task's own test task.

**Why:** The plan validator passes purely on TDD shape (impl depends on a test). It does
NOT verify that consumer→producer build edges exist. Plans are executed with parallelism
where the DAG allows, so a consumer-impl task becomes dispatchable the moment its test task
finishes — which can precede the producer-impl task in another phase. Result: compile break,
or a phase verification ("build and pass") that runs before the imported symbols exist.

**How to apply:** For each implementation/verification task, read its description for
"consumes / looks up / runs via / reads field X" language, map each consumed symbol to the
task that implements it, and confirm a transitive `depends_on` edge exists. Caught in
PLAN-SPEC-031: dispatch impl (TASK-013/014) imported pkg/pack/engine.Registry (TASK-003),
RunStdout/SandboxedRunStdout (TASK-009), and rule.Engine (TASK-006) but depended on none of
them; only the final full-gate task converged. Related to [[project-retirement-claim-scope]].
