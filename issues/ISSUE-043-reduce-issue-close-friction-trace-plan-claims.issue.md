---
title: "Reduce Issue-Close Friction — Trace Claims to the Backing Plan Instead of Re-Authoring Them"
schema_version: issue/v1

issue:
  id: ISSUE-043
  title: "Reduce Issue-Close Friction — Trace Claims to the Backing Plan Instead of Re-Authoring Them"
  type: technical-debt
  status: closed
  created: "2026-07-06"
  closed: "2026-07-08"

delivered_by: PLAN-ISSUE-043

complexity:
  scope: contained
  uncertainty: exploratory
  risk: safe
---

# Reduce Issue-Close Friction — Trace Claims to the Backing Plan Instead of Re-Authoring Them

## Resolution

Added delivered_by cheap-close: a closed issue satisfies traceability by tracing to a
completed backing plan (path-anchored, fail-loud) instead of re-authoring claims.

## Problem

The issue schema (`artifacts/issue/v1/schema.json`) requires full spec-parity to reach
`status: closed`: a `requirements` array (REQ-NNN), a `claims` array (CLM-NNN, each with
mandated test names), a `verification` block, and an `implementation` scope — the same
rigor the schema demands of a spec (`closed-requires-traceability`: "Requirements and
claims arrays required and fully validated from 'ready' onward — every REQ must have ≥1
CLM, every CLM must reference a valid REQ").

That rigor is appropriate for issues that reach `ready` on their own (worked directly,
with no plan). But the dominant reactive-work shape in this repo is
`issue → plan → implementation`: the plan is scaffolded from the issue
(`plan.spec_id: ISSUE-NNN`), and the plan is where requirements/claims/verification are
actually authored and traced to real tests. For that shape, closing the issue means
**re-deriving** the plan's already-authored claims back onto the issue — a second
authoring pass over the same facts, producing a second copy that can drift from the
first.

### Repro (observed 2026-07-06, closing ISSUE-018/034/035/036)

All four issues closed in the 2026-07-06 session (ISSUE-018, ISSUE-034, ISSUE-035,
ISSUE-036) were plan-backed: each had a `PLAN-ISSUE-NNN` that already carried
requirements, claims, mandated test names, and a verification block, all satisfied by
real passing tests. Closing each issue still required:

1. Opening the plan to find its requirements/claims.
2. Re-typing (or paraphrasing) each REQ/CLM pair onto the issue frontmatter.
3. Re-stating the `verification` block and `implementation` summary on the issue,
   duplicating what the plan already recorded.

This is non-trivial per issue — see ISSUE-018's closed frontmatter
(`issues/ISSUE-018-remove-vestigial-baked-in-code.issue.md`), which carries 6
requirements and 7 claims that are a close paraphrase of `PLAN-ISSUE-018`'s own
requirements/claims. None of that content was newly authored at close time; it was
copied and reformatted from a document that already exists and is already durable.

### Why this matters — it is a root cause of status drift, not a side annoyance

DIR-016 (parent) names this as one of three symptoms hardening the issue/plan track:
"Closing a plan-backed issue re-authors work that already exists... That re-authoring
tax is exactly what causes issues to sit at `open` well past the point their work
actually landed." Concretely:

- **It is the direct cause of the drift ISSUE-042 detects.** ISSUE-042 makes
  delivered-but-still-`open` drift LOUD (a gate-side detection fix). This issue is the
  other half: even once drift is *detected*, closing correctly is expensive enough that
  people route around it — the fix for 042's finding is "pay the re-authoring tax," and
  the same tax is why the drift accumulated in the first place. 042 makes the symptom
  visible; 043 makes the cure cheap enough that people actually apply it. Fixing 042
  alone without 043 just makes the same friction louder, not smaller.
- **It duplicates a source of truth.** The plan and the issue can each claim to be
  authoritative for "what was actually verified," and nothing keeps them in sync after
  close — if the plan is later amended (e.g., a claim's test is renamed), the issue's
  copy silently goes stale with no mechanism to notice.
- **The rigor itself is not the problem.** Requirements/claims/verification at close time
  are exactly the right bar — the issue is *where that rigor is authored*, not *whether
  it should exist*. Per CLAUDE.md's "make the right thing easy": the friction here is
  incidental (paraphrasing an existing document by hand), not essential (proving the
  work was actually verified).

## Solution

Not resolved here — this issue documents the problem and constrains the direction; the
planner owns the design tradeoff.

**Direction:** let a `closed` issue satisfy the schema's requirements/claims/verification
obligations by **tracing** to a delivered backing plan instead of duplicating the plan's
content onto the issue. The plan already declares its backing issue
(`spec_id: ISSUE-NNN` in the plan schema) — the missing piece is the reverse edge (issue
→ plan) and a validator rule that accepts it as satisfying `closed-requires-traceability`.

Two candidate mechanisms for the planner to weigh, not a prescribed pick:

1. **Schema conditional relaxation.** When an issue reaches `closed`, the validator
   locates the plan(s) whose `spec_id` names this issue. If a plan is found, is itself
   `delivered` (or equivalent terminal-success plan status), and its own
   requirements/claims/tests validate clean, then the issue's own `requirements`/`claims`
   arrays are NOT required to be independently populated — the plan's claims are treated
   as the delivered claims of record. The issue would still need *some* minimal content
   (see "how much content" below) so it stays independently readable.
2. **Explicit `delivered_by` field.** Add a `delivered_by: PLAN-ISSUE-NNN` key (parallel
   to the existing `replaced-by` convention for the `replaced` terminal state — see
   ISSUE-031 / `project_artifact_terminal_states`) that the validator follows to locate
   the plan and perform the same delivered/claims-clean check. This is more explicit and
   greppable than schema-side plan-discovery, at the cost of one more field to keep
   correct.

**Open design questions the planner must resolve, not pre-empt:**

- **How much content must the closed issue still carry?** Zero content (pure pointer)
  trades away the issue's value as a standalone durable record — someone reading
  `issues/ISSUE-NNN-*.issue.md` in isolation should still learn what was fixed and how
  it was verified, without being forced to open the plan. A short `Resolution` prose
  section (already an optional section in the schema, and already used this way by
  ISSUE-034's closed form) plus a `delivered_by`/traced-plan pointer is likely the right
  minimum — but confirm against how a closed issue is actually consumed today (e.g., is
  it ever read without the plan open beside it?).
- **What "the plan is clean" means for trace purposes.** Does the validator need to
  re-run the plan's tests, or is it sufficient that the plan's own claims/tests
  validated at the time the plan reached its terminal state (trusting the plan's own
  gate history)? Re-verifying live is more correct but couples issue-close validation to
  running the plan's full test command — confirm this is acceptable validator cost.
- **Multiple plans per issue.** An issue could in principle be split across more than one
  plan (or a plan reopened/superseded). Decide whether trace-to-close requires exactly
  one delivered plan, or how partial coverage across plans is handled — do not assume
  the 1:1 case silently generalizes.
- **Non-plan-backed issues are unaffected.** An issue that reaches `ready`/`closed`
  without ever having a backing plan (worked directly) must still populate its own
  `requirements`/`claims`/`verification` in full — this relaxation is conditional on a
  delivered plan existing, not a general loosening of the `ready`-onward rigor.

**Acceptance (for the eventual plan, not claimed here):** closing a plan-backed issue is
a low-friction status+date+pointer transition (per DIR-016's acceptance criteria); an
issue with no backing plan is held to the full existing rigor with no regression; the
schema/validator changes are proven with real fixture issues+plans (a delivered-plan
case that validates `closed` with a thin issue body, and a non-plan-backed case that
still requires full requirements/claims) rather than asserted from reading the schema.

## References

- `artifacts/issue/v1/schema.json` — `requirements`/`claims`/`verification`/
  `implementation` "required when status is closed" rules; the
  `closed-requires-traceability` enforcement rule this issue proposes relaxing
  conditionally
- `issues/ISSUE-018-remove-vestigial-baked-in-code.issue.md` — closed issue whose 6
  requirements / 7 claims are a close paraphrase of its backing plan; the concrete
  evidence of the re-authoring tax
- `issues/ISSUE-034-gate-coverage-flags-deleted-files.issue.md` — closed issue with a
  `Resolution` section already summarizing outcome in prose; a reference point for "how
  much content should a closed issue still carry"
- `issues/ISSUE-035-gate-substantiveness-flags-testmain-absence-tests.issue.md` — sibling
  closed issue from the same session, same re-authoring shape
- `issues/ISSUE-036-contracts-pack-compiler-func-only-signatures.issue.md` — sibling
  closed issue from the same session, same re-authoring shape
- `plans/PLAN-ISSUE-018-remove-code-check.plan.yml`,
  `plans/PLAN-ISSUE-034-coverage-excludes-deleted-files.plan.yml`,
  `plans/PLAN-ISSUE-035-substantiveness-testmain-absence.plan.yml`,
  `plans/PLAN-ISSUE-036-contracts-compiler-kind-aware.plan.yml` — the four backing
  plans whose requirements/claims were re-derived onto their issues at close
  time
- `directives/DIR-016-directive-issue-plan-lifecycle-hardening.directive.md` — parent
  directive; names this re-authoring tax as one of three symptoms to fix, alongside
  ISSUE-042 (drift detection) and ISSUE-044 (agent-guard config self-check)
- `issues/ISSUE-042-gate-flags-artifact-status-reality-drift.issue.md` — complementary
  sibling: 042 makes delivered-but-open drift loud; 043 makes closing cheap enough that
  the loud finding actually gets resolved instead of routed around
- `issues/ISSUE-031-artifact-terminal-states.issue.md` — prior art for the
  `replaced`/`replaced-by` pointer-field convention this issue's `delivered_by` candidate
  mechanism would mirror
- `plans/` plan schema — `spec_id` field on plans, the existing forward edge
  (plan → issue) this issue's trace mechanism would read in reverse
- CLAUDE.md — "make the right thing easy" (toolkits respect willpower, bundles
  substitute structure); "loud ≠ blocking" enforcement philosophy this issue's
  companion (ISSUE-042) applies to the drift side of the same problem
