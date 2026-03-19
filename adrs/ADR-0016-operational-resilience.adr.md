---
number: ADR-0016
created: "2026-03-18"
status: Accepted
deciders: "@bmanson"
decisions: "D-062, D-065, D-066"
schema_version: adr/v2
---

# ADR-0016: Operational Resilience — Concurrency, Recovery, Evolution

## Context

ADRs 0001–0015 describe an ambitious architecture: specs, packs, validators, kill chains, ledgers, independent reviewers, standard libraries, runtime hooks, and supply chain enforcement. Architecture is easy when everything goes right. The hard question is: what happens when things go wrong?

Agents fail mid-task. Parallel agents conflict on shared files. Schema formats evolve and old artifacts need to remain valid. Stakeholders want dashboards and hosted layers. Each of these is a real-world concern that, if unaddressed, erodes trust in the system. This ADR addresses the operational realities that make the difference between a system that works in demos and one that works in production.

This is intentionally the last ADR — the "and it keeps working" guarantee after the ambitious architecture of ADRs 0001–0015.

## Decision

### Plan-level file overlap detection (D-062)

Parallel-eligible tasks in a plan DAG must have disjoint file sets. This is a static check performed on the plan before any agent begins work:

1. Each task declares the files it will create or modify.
2. The plan validator computes the intersection of file sets for all task pairs that could execute in parallel (i.e., tasks with no depends_on relationship between them).
3. If any intersection is non-empty, the plan is rejected with a diagnostic identifying the conflicting tasks and files.

Sequential tasks (those with a depends_on edge between them) can share files — the dependency edge guarantees ordering. Only parallel-eligible tasks must be file-exclusive.

This is a lightweight gate — a static analysis step, not a runtime lock. It prevents merge conflicts and race conditions before any agent starts work, rather than detecting them after the fact. The cost is that plan authors must decompose work along file boundaries, which is also good engineering practice.

### Agent-bounded tasks (D-080)

Each plan task must fit within a single agent session. A task that requires multiple sequential agent sessions is too large and must be decomposed. Combined with file exclusivity (D-062), this constraint makes the plan a safe DAG of parallelizable, self-contained agent sessions:

- Each node in the DAG is one agent session.
- Each node operates on a disjoint set of files (relative to its parallel siblings).
- Dependencies between nodes are explicit in the DAG.
- The entire plan can be validated statically before execution begins.

This constraint simplifies failure recovery (below) because the unit of failure is always a single agent session operating on a known set of files.

### Agent failure recovery (D-066)

When an agent session fails mid-task, the recovery procedure uses the plan and ledger together:

1. **Identify the failed task.** The plan DAG shows which task was in progress. The ledger records the last action the agent completed before failure.
2. **Assess completed work.** Tasks with depends_on edges pointing to completed tasks (tasks whose tests pass) are known-good. The ledger confirms their completion.
3. **Re-run the failed task.** Because tasks operate on disjoint file sets, a failed task's incomplete output does not corrupt other tasks' files. The failed task is re-run from scratch — its file set is wiped and regenerated.
4. **Record the failure and retry.** The ledger records the failure as one entry and the retry as a separate entry. The audit trail shows exactly what happened: which task failed, when, and what the retry produced.

Recovery is "re-run the failed task," not "start over from the beginning." The plan DAG and file exclusivity guarantee that completed tasks are unaffected by the failure. This makes failure recovery proportional to the size of the failed task, not the size of the entire project.

### Schema evolution

Backstop artifacts (specs, recipes, packs, ledgers) declare their schema version in the `Schema-Version` metadata field. Schema evolution follows these rules:

- **Within a major version:** changes are backward-compatible. New optional fields can be added. Existing fields cannot be removed or have their semantics changed. An artifact valid under `adr/v1` remains valid under all `adr/v1.x` versions.
- **Across major versions:** breaking changes are permitted. New major versions get new schema directories in the artifacts tree. An artifact declares which version it conforms to, and is validated against that version forever.
- **No implicit migration.** Artifacts are not silently upgraded to newer schema versions. If a project wants to adopt a new schema version, it explicitly updates the `Schema-Version` field and makes the necessary changes.

This ensures that old artifacts remain valid indefinitely. A project started under `adr/v1` does not break when backstop adds `adr/v2` — the v1 schema continues to exist and validate correctly.

### Hosted layer deferred (D-065)

A read-only dashboard or hosted rendering layer (e.g., a web UI showing ledger history, plan status, or validation results) is explicitly out of scope for backstop-core. The rationale:

- **The enforcement stack is the product.** Backstop's value proposition is mechanical enforcement, not visualization. A dashboard that shows pretty graphs of validation results adds no enforcement value.
- **Observability is a deep product discussion.** What to show, how to show it, who the audience is, and what actions they can take from the dashboard are product decisions that deserve their own dedicated design process.
- **CLI-first is sufficient for v1.** All information is available via `backstop validate`, `backstop status`, and ledger queries. A hosted layer is a convenience, not a requirement.

This decision can be revisited in a future ADR. It is deferred, not rejected.

## Consequences

### What this enables
- **Safe parallel agent execution.** File exclusivity and agent-bounded tasks make it safe to run multiple agents simultaneously. Conflicts are prevented by construction, not detected after the fact.
- **Proportional failure recovery.** When an agent fails, only its task is re-run. Completed tasks are unaffected. Recovery time is proportional to the failed task, not the total project size.
- **Long-term artifact validity.** Schema evolution rules ensure that artifacts written today will validate correctly in perpetuity. Projects are never broken by backstop upgrades.
- **Focused product scope.** Deferring the hosted layer keeps the team focused on enforcement — the core value proposition. Visualization can be built later without affecting the enforcement stack.

### What this requires
- **Discipline in plan decomposition.** Tasks must be decomposed along file boundaries. This is a constraint on plan authors (human or agent) that requires awareness of the file exclusivity rule.
- **Schema version discipline.** New features must be added as backward-compatible extensions within a major version. Breaking changes require a new major version — a high bar that should be cleared rarely.
- **Acceptance that visualization is deferred.** Teams that want dashboards must build them externally or wait for a future backstop release. The CLI is the only interface for v1.

## Alternatives Considered

| Approach | Why Rejected |
|----------|-------------|
| Runtime file locking instead of static analysis | Adds runtime complexity and failure modes. Static analysis is simpler, faster, and catches conflicts before any agent starts work. |
| Task-level checkpointing for recovery | Over-engineered for agent sessions. Agent sessions are short enough that re-running from scratch is faster than managing checkpoint state. Checkpointing adds complexity without proportional benefit. |
| Automatic schema migration | Breaks the principle that artifacts are immutable records. An artifact should validate against the schema version it was written under, forever. Automatic migration changes artifact semantics silently. |
| Building the dashboard now | Distracts from the enforcement stack, which is the core product. Dashboard features have infinite scope and would consume engineering bandwidth that should go toward making enforcement complete and reliable. |

## References

- D-062: Plan-level file overlap detection (formalized as D-081)
- D-065: Hosted layer deferred
- D-066: Agent failure recovery via plan + ledger cross-reference
- D-080: Agent-bounded tasks — one task per agent session
- ADR-0002: Canonical artifact primitives — schema versioning
- ADR-0004: Validation engine — plan validation
- ADR-0009: CI/CD pipeline — where plan execution occurs
- ADR-0011: Provenance ledger — failure and recovery recording
