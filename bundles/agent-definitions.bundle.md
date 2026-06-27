---
title: Agent Definitions and Plan Schema Evolution
schema_version: bundle/v1

bundle:
  name: agent-definitions
  version: "0.4.0"
  created: "2026-03-31"
  updated: "2026-06-27"
  category: feature

status:
  maturity: delivered
  note: >
    Delivered. The specialized agent roles this bundle defined (spec-author,
    planner, implementer, spec/plan/impl reviewers, artifact authors) now exist
    and are in active use. Its work shipped across SPEC-002 (plan-schema-evolution),
    SPEC-003 (agent-hooks), and SPEC-004 (spec-schema-evolution); the plan-schema /
    D-081 file-exclusivity enforcement is marked IMPLEMENTED in the bundle body.
    Success terminal — not a retirement.

problem:
  summary: >
    Backstop's workflow state machine (ADR-0018) requires specialized agent roles
    at each phase — spec authors, planners, implementers, reviewers — each with
    different tool access, constraints, and review responsibilities. ADR-0012
    mandates separate sessions for review. Currently no agent definitions exist.
    Additionally, the plan validator enforces structural rules (D-080 file scope,
    D-081 disjoint files) but lacks enforcement of TDD discipline (test tasks
    before implementation), gate verification cadence, and task type
    classification. Without these, agents must remember to follow discipline
    rather than having it mechanically enforced — exactly the failure mode
    backstop exists to prevent.

  user_story: >
    As a developer using backstop, I want specialized agent definitions that
    know their role in the workflow — a planner that produces TDD-compliant
    plans, an implementer that follows them, a reviewer that evaluates
    independently, and artifact authors that produce schema-valid output.
    I want the plan validator to mechanically enforce TDD ordering and gate
    cadence so the implementer cannot skip tests or ignore verification.

  success_criteria:
    - Ten agent definitions exist covering all workflow roles
    - Plan validator rejects plans with two implementation tasks in a row
    - Plan validator rejects plans missing verification tasks in implementation phases
    - Plan validator rejects plans without comprehensive verification in final phase
    - PreToolUse hooks enforce file-type restrictions per agent role
    - Reviewer agents run in isolated subagent sessions with no write access
    - All existing plan tests continue to pass with task type additions
    - Agents can discover active artifacts via session state and convention

solution:
  approach: >
    Define agent roles as .claude/agents/ markdown files with tool restrictions,
    model selection, and behavioral instructions aligned to ADR-0018 phases.
    Extend the plan schema and validator to enforce task types, TDD ordering
    (implementation directly blocked by test — no two implementation tasks in
    a row), gate verification cadence, and parallel phase eligibility scoped by
    file overlap. Port proven patterns from mechsuit agent definitions while
    adapting to backstop conventions (Go, backstop artifacts, claim-based
    traceability). PreToolUse hooks enforce file-type write restrictions per
    agent, serving as the first real backstop hooks in production. Use Claude
    Code subagents (stable) for agent isolation; note agent teams (experimental)
    as a future upgrade path for parallel plan execution.

  assumptions:
    - Claude Code subagent model provides genuine context isolation (confirmed)
    - Claude Code supports per-agent hooks in frontmatter (confirmed)
    - The CLI will scaffold artifacts before authors need to fill them in
    - Existing plan validator tests provide a regression safety net
    - Spec and bundle validators are sufficient for current needs aside from
      spec-to-bundle cross-validation (future enhancement)
---

# Agent Definitions and Plan Schema Evolution

## Current Thinking

### Agent Roles (Ten Total)

Based on ADR-0018's workflow phases, ADR-0012's review model, and mechsuit
experience:

**Artifact Authors** (INTAKE and SPEC phases):
- **bundle-author** — facilitates context bundle creation, drives OQ resolution,
  manages maturity progression
- **spec-author** — writes specs from bundle seeds, produces claims with mandated
  test names, sharp edges, contracts
- **adr-author** — formal architecture decision records
- **issue-author** — reactive work items (bugs, tech debt, enhancements)

**Planners** (PLAN phase):
- **planner** — decomposes specs into phased, file-scoped tasks with TDD ordering,
  gate verification cadence, and claim traceability

**Implementers** (IMPL phase):
- **implementer** — executes plan tasks, writes code and tests using mandated
  names, updates session state. Hooks handle gate enforcement automatically.

**Reviewers** (three specialized, per D-102):
- **spec-reviewer** — reads bundle + spec(s), checks coverage and coherence.
  "Does the spec fully cover the bundle? Gaps? Ambiguities? Missing claims?"
- **plan-reviewer** — reads spec + plan, checks congruence and structural
  validity. "Is the plan congruent with the spec? Every claim mapped? TDD
  ordering sound? File scopes right?"
- **impl-reviewer** — reads spec + plan + code diff, checks correctness and
  completeness. "Does the implementation satisfy the plan tasks and spec claims?"

All reviewers share core discipline: separate session (subagent with isolated
context), structured verdict, no write access, review/fix loop until clean.

### Claude Code Subagents vs Agent Teams

**Subagents (using now):** Stable, isolated context windows, per-agent hooks,
no parent conversation leakage. The reviewer cannot see the implementer's
chain of thought — bias prevention by architecture per ADR-0012. Cannot spawn
sub-agents (one level deep).

**Agent Teams (future path):** Experimental feature requiring feature flag.
Completely separate Claude Code instances running in parallel with direct
inter-agent messaging and shared task lists. Potential fit for parallel plan
phase execution where D-081's disjoint file sets prevent conflicts. A main
agent could maintain plan-level state machine coordination, dispatching
phases to teammates and tracking completion.

**Decision:** Use subagents for alpha. The isolation model maps cleanly to
ADR-0012's requirements. Agent teams are a future upgrade path when the
feature stabilizes — the abstractions we're building (isolated review,
task-scoped file ownership, phase-level parallelism) map to either model.

**Note on Copilot SDK difference:** Copilot SDK agents inherit everything
from the main agent — context, conversation history, all of it. Agent-level
filtering and hooks only work on background agents. To achieve ADR-0012's
isolation on Copilot SDK, you need an entirely separate session, not a
sub-agent. The runtime provider abstraction must express "isolated agent
execution" as a capability, not assume the underlying mechanism.

### What We Learned from Mechsuit

1. **The reviewer must verify against BOTH plan and spec** — mechsuit's SPEC-015
   failure happened because the reviewer checked requirements but not plan tasks.
   Deliverable counts, task completion, and claim coverage all matter.

2. **TDD must be enforced in the plan, not just instructed** — mechsuit uses OPA
   policies to validate that implementation tasks are blocked by test tasks. Two
   implementation tasks in a row gets rejected. Hardcore TDD.

3. **Gate cadence matters** — running all gates after every phase wastes tokens.
   Running none lets drift compound. Smart cadence: verification tasks in
   implementation phases, comprehensive verification in final phase.

4. **Task types drive validation rules** — setup, test, implementation,
   verification, refactor, documentation each have different constraints.
   Implementation requires test dependency. Verification requires gate commands.
   Setup has no test requirement.

5. **The implementer agent is simpler when hooks exist** — half of mechsuit's
   implementer instructions are "remember to run gates." With backstop hooks
   firing automatically, the implementer just follows the plan.

6. **Never write reports to the repo** — every mechsuit agent has this rule.
   All summaries, checklists, and verification reports stay in session state.

7. **The CLI scaffolds artifacts** — `backstop new spec` handles frontmatter,
   IDs, filenames, required sections. Authors focus on content quality (claims,
   sharp edges, design decisions), not schema compliance.

8. **Standards must be prescribed left of implementation** — a Java-savant
   engineer running mechsuit found that agents constantly need reminding about
   code standards during implementation. The fix: bind standards to requirements
   in the spec, not just at the spec level. REQ-003 doesn't just "follow
   STD-JAVA-001" — it follows STD-JAVA-001:J-042 specifically for mock chain
   handling. The agent knows exactly which rule applies to which requirement.

9. **Ambiguity needs an escalation path, not a guess** — when standards don't
   fully cover a case (e.g., deeply nested API mocking patterns), agents
   shouldn't improvise. They should escalate to the human or a principal
   engineer agent that has a crib sheet of judgment calls. Better to ask than
   to guess wrong and have it caught in review.

10. **Spec-level review questions drive impl review** — like bundle OQs but
    for code review. The spec author generates adversarial questions that the
    impl-reviewer should check: "Did the implementation mock the nested chain
    correctly?" "Did it handle the null case at depth 3?" Forces the reviewer
    to check things the spec author already knows are risky.

### Plan Schema Evolution

The current plan validator (D-080, D-081) needs these additions:

**Task type field** (new required field per task):
```yaml
tasks:
  - id: TASK-001
    type: test              # NEW: setup|test|implementation|verification|refactor|documentation
    title: "Write tests for RuntimeProvider interface"
    description: "..."
    files: [...]
    claims: [...]
    depends_on: []
```

**TDD enforcement rules (strict):**
- Every `implementation` task must directly depend on at least one `test` task.
  Two implementation tasks in a row is a validation failure. No exceptions.
- `refactor` tasks can depend on `implementation` or other `refactor` tasks.
- `verification` tasks must depend on at least one `implementation` or `refactor`
  task.
- `setup` and `documentation` tasks have no dependency type constraints.

**Gate cadence rules:**
- Every phase containing `implementation` tasks must also contain at least one
  `verification` task.
- The final phase must contain verification tasks covering every category of
  work the plan's tasks actually perform — determined by file types touched
  and artifact types produced. Not a hardcoded gate list, but comprehensive
  verification relevant to the work done.

**Parallel phase rules (extending D-081):**
- Phases without dependency chains between them are parallel-eligible.
- Parallel-eligible phases must have disjoint file sets across ALL their tasks.
- Already partially implemented at task level; needs phase-level extension.

### Agent File Restrictions via PreToolUse Hooks

Each agent definition includes PreToolUse hooks that enforce file-type
restrictions. This is the first real backstop hook in production — a
lightweight wedge that instantiates the control surface before the full
runtime hooks ship.

| Agent | Allowed write patterns |
|-------|----------------------|
| bundle-author | `*.bundle.md` |
| spec-author | `*.spec.md` |
| adr-author | `*.adr.md` |
| issue-author | `*.issue.md` |
| planner | `*.plan.yml` |
| implementer | `*.go`, `*_test.go`, `go.mod`, `go.sum`, `Makefile` |
| spec-reviewer | nothing (disallowedTools: Edit, Write) |
| plan-reviewer | nothing (disallowedTools: Edit, Write) |
| impl-reviewer | nothing (disallowedTools: Edit, Write) |

Hook scripts live at `.claude/hooks/backstop-agent-guard.sh` and check the
file path against allowed patterns based on the agent type passed via
environment or stdin context.

### Standards Binding and Review Questions

**Standards at the requirement level:** The spec schema should support an optional
`follows` field on requirements, referencing specific standard rules or recipes
that constrain the implementation. Format: `STD-LANG-NNN:RULE-ID` or `recipe-name`.
This is not just cataloging which standards apply — it's traceability from the
standard down to the specific requirement that uses it.

Example:
```yaml
requirements:
  - id: REQ-003
    text: Mock chain handling for nested API calls
    follows: STD-JAVA-001:J-042
```

The planner then knows REQ-003's implementation must follow J-042. The implementer
knows which rule to read. The reviewer knows which rule to check against.

**Hook-injected standards context:** The spec-author agent shouldn't have to
discover available standards manually. A SessionStart or SubagentStart hook reads
the compiled manifests from `.backstop/rules/` and injects them as context:
"Available standards: STD-GO-001 (23 rules covering core, test, security)..."
The agent then binds relevant rules to requirements as it writes the spec.

**Review questions on specs:** Specs should include an optional "Review Questions"
section — adversarial questions the spec author generates for the impl-reviewer.
These capture risks the author already sees that might not be obvious from the
claims alone. The impl-reviewer agent is instructed to check each review question
during code review.

**Escalation for ambiguity:** When a standard doesn't cover the exact case, or
when there's a judgment call (e.g., how to mock a deeply nested Java API chain),
the agent escalates rather than guessing. Escalation goes to the human or a
future principal engineer agent that maintains a crib sheet — an indexed
collection of judgment calls for common ambiguous situations.

### Validator Hardening

Beyond plan schema evolution, the existing validators need:

- **spec.go** — cross-validation that spec requirements trace back to a bundle's
  requirements via the `supports` field. Not blocking for alpha but important
  for the spec-reviewer agent to have mechanical backing.
- **bundle.go** — no changes needed. Maturity gates are sufficient.
- **adr.go** — sufficient for current needs.
- **issue.go** — sufficient for current needs.

### Resolved Design Questions

**OQ-1: Separate artifact authors vs single author** → Separate agents per
artifact type. Each focuses on content quality, not schema mechanics (the CLI
scaffolds, the validator enforces). More targeted instructions and examples
per agent. Same reasoning as separate validators per artifact type.

**OQ-2: TDD enforcement strictness** → Strict. Every implementation task must
directly depend on at least one test task. Two implementation tasks in a row
is a validation failure. No transitive chains, no phase-level flexibility.
Hardcore TDD, mechanically enforced.

**OQ-3: Comprehensive gates in final phase** → The final phase must run every
verification mechanism relevant to the work performed — determined by which
file types were touched, which artifact types were produced, and which
enforcement manifests apply. Not a hardcoded list. Gaps are expected and
surfaced — that's the point of the phase.

**OQ-4: Phase-aware reviewer** → Three specialized reviewer agents:
spec-reviewer (bundle→spec coverage), plan-reviewer (spec→plan congruence),
impl-reviewer (plan+spec→code correctness). Each has narrow scope, specific
inputs, and a focused question to answer. All share core discipline: separate
session, structured verdict, review/fix loop.

**OQ-5: How agents discover artifacts** → Convention and session context. The
agent reads active artifacts from the conversation or session state. No formal
discovery mechanism needed — "review that" works when context is obvious.

**OQ-6: Ordering** → Ship together. Agent definitions + plan validator evolution
in the same release. The planner produces TDD-compliant plans, the validator
enforces it. Neither is useful without the other. Also audit spec and bundle
validators for needed hardening.

**OQ-7: Mechanical enforcement of no reports in repo** → PreToolUse hooks
enforce file-type restrictions per agent. Each agent can only write to file
patterns matching its role. This serves double duty as the first real backstop
hook in production — the lightweight wedge for the control surface before full
runtime hooks ship.

## Design Decisions

- **DD-1:** Ten specialized agents: four artifact authors (bundle, spec, ADR,
  issue), one planner, one implementer, three reviewers (spec, plan, impl).
  Specificity beats generality.
- **DD-2:** Strict TDD enforcement in plan validator. Implementation tasks must
  directly depend on test tasks. Two implementation tasks in a row is rejected.
  No exceptions. Mechanically enforced, not instructed.
- **DD-3:** Smart gate cadence — verification tasks required in implementation
  phases, comprehensive relevant verification in final phase. Not a hardcoded
  gate list — determined by work actually performed.
- **DD-4:** Subagents for isolation (stable). Agent teams noted as future
  upgrade path for parallel plan execution when experimental status lifts.
- **DD-5:** PreToolUse hooks enforce per-agent file-type restrictions. First
  real backstop hooks in production. Lightweight wedge for the control surface.
- **DD-6:** Reviewer agents are read-only (disallowedTools: Edit, Write).
  Separate subagent sessions provide context isolation per ADR-0012.
- **DD-7:** CLI scaffolds artifacts before authors fill them in. Authors focus
  on content quality, not schema compliance.
- **DD-8:** Ship agent definitions and plan validator evolution together.
  Neither is useful without the other.
- **DD-9:** Copilot SDK requires separate sessions (not sub-agents) to achieve
  ADR-0012 isolation. Runtime provider abstraction must express "isolated agent
  execution" as a capability, not assume mechanism.
- **DD-10:** Standards bound at the requirement level, not just spec level. The
  `follows` field on requirements traces to specific standard rules (e.g.,
  STD-JAVA-001:J-042), giving implementers and reviewers precise references.
- **DD-11:** Spec-level review questions — adversarial questions authored during
  spec creation that the impl-reviewer must check. Captures risk the spec author
  sees that claims alone don't express.
- **DD-12:** Hook-injected standards context — SessionStart/SubagentStart hooks
  read compiled manifests and inject available standards as agent context. Agents
  don't discover standards manually; the hook tells them what's available.
- **DD-13:** Escalation over guessing — when standards don't cover the exact case,
  agents escalate to the human or a principal engineer agent rather than
  improvising. A future crib sheet artifact indexes common judgment calls.

## Draft Requirements

requirements:
  - id: REQ-001
    text: >
      Agent definitions must exist for all ten workflow roles: bundle-author,
      spec-author, adr-author, issue-author, planner, implementer,
      spec-reviewer, plan-reviewer, impl-reviewer
  - id: REQ-002
    text: >
      Each agent definition must include tool restrictions matching its role —
      reviewers cannot write, authors can only write their artifact type
  - id: REQ-003
    text: >
      Reviewer agents must operate as subagents with isolated context windows
      per ADR-0012 / D-102 — no access to implementation context
  - id: REQ-004
    text: >
      Plan schema must include a task type field with enum validation:
      setup, test, implementation, verification, refactor, documentation
  - id: REQ-005
    text: >
      Plan validator must enforce strict TDD: every implementation task must
      directly depend on at least one test task. Two implementation tasks in
      a row is a validation failure.
  - id: REQ-006
    text: >
      Plan validator must enforce gate cadence: every phase containing
      implementation tasks must also contain at least one verification task
  - id: REQ-007
    text: >
      Plan validator must enforce comprehensive verification in the final
      phase — verification tasks covering every category of work performed,
      determined by file types touched and artifact types produced
  - id: REQ-008
    text: >
      PreToolUse hooks must enforce file-type write restrictions per agent
      role — each agent can only write to file patterns matching its role
  - id: REQ-009
    text: >
      Agent definitions must reference backstop artifact conventions, not
      mechsuit — Go project, backstop CLI, claim-based traceability
  - id: REQ-010
    text: >
      All agents must include instruction to never write summary, report, or
      status files to the repository
  - id: REQ-011
    text: >
      The planner agent must produce plans that pass all validator rules
      mechanically — task types, TDD ordering, gate cadence, file scope
  - id: REQ-012
    text: >
      Spec validator should cross-validate that spec requirements trace to
      bundle requirements via the supports field (enhancement, not blocking)
  - id: REQ-013
    text: >
      All existing plan validator tests must continue to pass after task
      type additions — backward compatibility for plans without type field
      is NOT required (plans must be updated)
  - id: REQ-014
    text: >
      Spec schema must support an optional follows field on requirements
      that references specific standard rules or recipes
      (format: STD-LANG-NNN:RULE-ID or recipe-name)
  - id: REQ-015
    text: >
      Spec schema must support an optional Review Questions section for
      adversarial questions the spec author generates for the impl-reviewer
  - id: REQ-016
    text: >
      The spec-author agent must be instructed to bind applicable standards
      to requirements using the follows field and generate review questions
  - id: REQ-017
    text: >
      The impl-reviewer agent must be instructed to check review questions
      during code review in addition to claim verification
  - id: REQ-018
    text: >
      SessionStart or SubagentStart hooks must inject available project
      standards as agent context by reading compiled manifests from
      .backstop/rules/

## Spec Seeds

- **SPEC-002:** Plan Schema Evolution — task type field, TDD enforcement rules,
  gate cadence validation, comprehensive final phase requirement, phase-level
  parallel file exclusivity. Extends plan.go and plan schema. **IMPLEMENTED.**
  Covers bundle REQ-004 through REQ-007, REQ-013.
- **SPEC-003:** Agent Hooks — PreToolUse guard script (.claude/hooks/backstop-
  agent-guard.sh) enforcing per-agent file-type write restrictions, settings.json
  registration, exit code semantics, default-deny for unknown agents. Covers
  bundle REQ-002, REQ-008.
- **SPEC-004:** Spec Schema Evolution — optional follows field on requirements
  binding to specific standard rules (STD-LANG-NNN:RULE-ID), optional Review
  Questions section for impl-reviewer, agent instruction updates for spec-author
  and impl-reviewer, SessionStart hook for standards context injection. Covers
  bundle REQ-014 through REQ-018.

## Open Questions

None remaining. All seven original OQs resolved 2026-03-31.

## Version History

- 0.1.0 (2026-03-31): Initial bundle. Agent roles identified, plan schema
  gaps documented, seven open questions. Exploring maturity.
- 0.2.0 (2026-03-31): All 7 questions resolved. Maturity advanced to defined.
  Key decisions: ten specialized agents (four authors, planner, implementer,
  three reviewers), strict TDD enforcement (two impl tasks in a row rejected),
  PreToolUse hooks as first real backstop hooks, subagents for isolation with
  agent teams as future path, ship agents and validator together.
- 0.3.0 (2026-04-02): Added standards binding at requirement level (follows
  field), spec-level review questions for impl-reviewer, hook-injected standards
  context, and escalation-over-guessing pattern. Driven by real-world feedback
  from a Java engineer running mechsuit daily. Five new requirements (REQ-014
  through REQ-018) and four new design decisions (DD-10 through DD-13).
- 0.4.0 (2026-06-27): Marked delivered. Work shipped via SPEC-002
  (plan-schema-evolution), SPEC-003 (agent-hooks), and SPEC-004
  (spec-schema-evolution); the specialized agent roles are now in active use and
  the plan-schema / D-081 file-exclusivity enforcement is IMPLEMENTED. Success
  terminal maturity (per ISSUE-031) — exempt from defined/ready completeness gates.

## Notes / Ideas

- Agent teams (experimental) could enable parallel plan phase execution with
  a main agent maintaining plan-level state machine coordination. D-081's
  disjoint file sets prevent conflicts. Worth revisiting when the feature
  stabilizes.
- The PreToolUse hook for agent file restrictions is the same hook surface
  the runtime-hooks bundle will use. Building it now validates the hook
  pattern before we get fancy with post-write validation and session state.
- Copilot SDK's lack of subagent isolation means the runtime provider must
  express "isolated agent execution" abstractly. This feeds back into the
  runtime-hooks bundle's provider interface design.

## References

- ADR-0012: Review Model — Independent Reviewer
- ADR-0018: Workflow State Machine
- D-067: Independent reviewer — separate session
- D-080: Agent-bounded tasks (file scope, claims)
- D-081: Disjoint file sets for parallel tasks
- D-100 through D-104: Workflow state machine decisions
- D-102: Review is always a separate agent session
- Mechsuit agent definitions (mechsuit-reviewer, mechsuit-planner,
  mechsuit-implementer, mechsuit-technical-writer)
- Mechsuit plan validation pipeline (SPEC-021, SPEC-022)
- Claude Code subagent documentation (isolated context, per-agent hooks)
- Claude Code agent teams documentation (experimental, parallel execution)
