---
title: "Plan Verification Commands Unresolvable"
schema_version: issue/v1

issue:
  id: ISSUE-076
  title: "Plan Verification Commands Unresolvable"
  type: technical-debt
  status: open
  created: "2026-07-25"

complexity:
  scope: cross-cutting
  uncertainty: known
  risk: moderate
---

# Plan Verification Commands Unresolvable

## Problem

Validated plans mandate a CLI subcommand that no longer exists, and the agent contract that
executes those plans mandates the same dead command as its inner loop.

`backstop code check` was deleted by ISSUE-018 (closed 2026-07-06, commit `d5efd5b` "feat:
eradicate `backstop code check` + un-vacuum gate dimensions"). `backstop --help` today lists only:
`artifact, baseline, commands, completion, gate, help, pack, version, waiver`. Invoking
`backstop code check` exits non-zero with `unknown command "code" for "backstop"` — confirmed
live, not from memory.

Despite the deletion, the command is still mandated in two live places:

- **`plans/PLAN-SPEC-054-recipe-apply-and-manifest.plan.yml`** — 16 occurrences, in
  verification-task titles and descriptions (e.g. line 802
  `title: "backstop code check — manifest validation matrix + substitution suite green"`, and
  lines 953, 1083). The same plan mentions `go test` zero times. This plan is validated and
  currently in flight — phase 5 landed as commit `c305f38`, with phases referencing the dead
  command still ahead.
- **`.claude/agents/implementer.md`** — the implementer agent contract itself: "Run
  `backstop code check` after every implementation/refactor task", "`backstop code check` is
  your inner loop", "Do NOT use raw `go test` or `golangci-lint` directly". This is the contract
  every `/implement` dispatch loads.

Other plans should be swept for the same pattern; PLAN-SPEC-054 is the confirmed instance, not
necessarily the only one. Note that the mandated strings also invoke the wrong binary name
(bare `backstop` instead of `./bin/backstop` / the built binary) — tracked separately as
ISSUE-077, not fixed here.

### Why this is currently masked, not fixed

The implementer agent does not actually fail when it hits this — it silently substitutes the
right command, because the session that discovered the gap wrote a Claude-Code-specific memory
file (`.claude/agent-memory/implementer/project_no_code_check_command.md`) correcting it. That
correction only exists in one vendor's memory directory; it is not in the repo, not enforced by
any gate, and does not follow the plan or the agent contract to a fresh session, a different
agent runtime, or a contributor who never loaded that memory. The plan and the agent contract
still say the wrong thing today.

### Why review did not catch it (structural, not a lapse)

The `plan-reviewer` agent contract runs nine checks, and all nine are *congruence* checks:
validator run, claim coverage, TDD ordering, gate cadence, file scope, dependency graph,
sharp-edge coverage. Step 6 ("Verify Gate Cadence") only asks whether a verification task is
**present** — "Does every phase with implementation tasks also have verification tasks?" — never
whether the command that task names actually **resolves**. Its blocker list is structural only
(claim with no task, impl task without a test dependency, phase without verification). The
reviewer has Bash available but nothing in its contract directs it to resolve or execute a
declared command string against the real CLI.

This is the same failure class as the existing `review_misses_baked_nouns` finding (agent
memory): review validates relationships *between artifacts*, and never validates an artifact's
claims *against the world*. A plan can be internally consistent — every phase has a verification
task, every task names a command — and still mandate a command that has not existed for weeks.

## Solution

Two parts. The first clears today's instance; the second is the durable fix that prevents the
class from recurring.

1. **Sweep and correct.** Update `plans/PLAN-SPEC-054-recipe-apply-and-manifest.plan.yml` (and any
   other plans found to have the same pattern) plus `.claude/agents/implementer.md` to name the
   real verification path: `go test ./pkg/<pkg>/...` (or the package under test) for red/green
   unit evidence, then `./bin/backstop gate` (diff-scoped by default) as the real check. Route
   the plan edit through the plan-authoring path, not a hand edit, per the "never hand-edit
   artifacts" rule — re-run `/plan` or the equivalent correction flow so the artifact stays
   schema-valid and re-validated, not patched with a text editor.
2. **Add a validator check (durable fix).** `backstop artifact validate` (plan validator,
   `pkg/validate`) should resolve every command string named in a plan's verification tasks
   against the real, discoverable command surface — the same `backstop commands` agent-discovery
   JSON already emitted at `cmd/backstop/root.go:126` — and fail loud when a verification task
   names a command that does not resolve. This closes the exact gap the plan-reviewer structurally
   cannot close: a better reviewer prompt only re-checks relationships between artifacts; only a
   deterministic check against the live command surface can catch a claim that has drifted from
   reality. This belongs in the gate/validator path (dogfooded, deterministic), not in an LLM
   reviewer's judgment call.

## Lineage

- **ISSUE-018** (closed) performed the removal of `backstop code check` itself.
- **ISSUE-048** (open, "Reconcile Stranded Terminal Lineage — ISSUE-018 / ISSUE-036 Residual")
  reconciles the *backward-looking* residue of that same deletion: closed artifacts whose
  `mandated_tests` are stranded (7 live `artifact_status_drift` violations, baseline-grandfathered).
  ISSUE-076 is the *forward-looking* strand of the same deletion — live instructions (a validated,
  in-flight plan and the implementer agent contract) still pointing at the deleted command — and
  is deliberately filed separately so ISSUE-048 stays contained and closeable. The fix shapes also
  differ: ISSUE-048 is a hand-reconciliation call on two already-closed artifacts; ISSUE-076 is a
  sweep of live artifacts plus a new validator check.
- **ISSUE-075** (open) line 98 already parked "`backstop code check` naming (noted, out of
  scope)" when it found the same string in a stale smoke-test skip — precedent for not absorbing
  this fix into that issue either.
- **ISSUE-077** (open) — the bare-`backstop`-vs-built-binary naming issue visible in the same
  plan text; a related but distinct defect, not fixed here.

## Verification

verification:
  level: static
  test_command: go test ./pkg/validate/... -run TestPlan -count=1

## References

- `plans/PLAN-SPEC-054-recipe-apply-and-manifest.plan.yml:802,953,1083` (and 13 more occurrences)
  — `backstop code check` named as the verification command
- `.claude/agents/implementer.md` — implementer contract mandating `backstop code check` as the
  inner loop and forbidding raw `go test` / `golangci-lint`
- `.claude/agent-memory/implementer/project_no_code_check_command.md` — the non-durable,
  Claude-Code-specific correction currently masking this gap
- ISSUE-018 (closed 2026-07-06, commit `d5efd5b`) — deleted `backstop code check`
- ISSUE-048 (open) — backward-looking reconciliation of the same deletion's stranded tests
- ISSUE-075 (open) — parked the same dead-command string as out of scope in a smoke-test fixture
- ISSUE-077 (open) — bare-`backstop`-vs-built-binary naming, visible in the same plan text
- `cmd/backstop/root.go:126` — `commands` agent-discovery JSON, the resolution source the new
  validator check should use
- `.claude/agents/plan-reviewer.md` Step 6 ("Verify Gate Cadence") — the structural-only check
  that cannot and does not catch this class
- agent memory `feedback_review_misses_baked_nouns` — same failure class precedent: review checks
  congruence between artifacts, never an artifact's claims against the world
