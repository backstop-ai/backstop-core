---
title: "Agent Guard Missing Role Self Check"
schema_version: issue/v1

issue:
  id: ISSUE-044
  title: "Agent Guard Missing Role Self Check"
  type: technical-debt
  status: open
  created: "2026-07-06"

complexity:
  scope: contained
  uncertainty: known
  risk: moderate
---

# ISSUE-044: Agent Guard Missing Role Self Check

## Problem

Adding a new sub-agent under `.claude/agents/*.md` requires ALSO adding a
matching `case` branch to the write-permission hook
`.claude/hooks/backstop-agent-guard.sh` — otherwise the new agent's
`agent_type` falls through to the hook's default `*) block` branch and the
agent silently cannot write ANY file, via a hard PreToolUse deny. This
coupling between the agent roster and the guard's case statement is
undocumented and, today, entirely unenforced: nothing checks that the two
lists agree.

### Hit live, 2026-07-06

A new `directive-author` agent was added (`.claude/agents/directive-author.md`)
without a corresponding case in `backstop-agent-guard.sh`. The agent was
blocked from writing `.directive.md` files until the case was added by hand.
The failure mode was confusing in two directions at once:

- **Silent** — the block reason (`"agent directive-author not permitted to
  write ..."`) reads like a deliberate policy decision, not a missing-config
  bug. Nothing pointed at the actual cause (the hook's case statement doesn't
  know this agent exists).
- **Inconsistent** — the hook only fires on the `Write`/`Edit` tools (its
  `PreToolUse` matcher does not cover `Bash`). One `directive-author` instance
  hit the block on `Write` and stalled; a second instance in the same
  situation worked around it by writing the file via a `Bash` heredoc, which
  the hook never sees, and proceeded as if nothing were wrong. So the same
  missing-case bug produced two different observed behaviors depending on
  which tool the agent happened to reach for — a hard block for one instance,
  a silent bypass for another.

### Current state of the coupling (verified 2026-07-06)

`.claude/agents/*.md` currently defines 11 agents: `adr-author`,
`bundle-author`, `bundle-reviewer`, `directive-author`, `impl-reviewer`,
`implementer`, `issue-author`, `plan-reviewer`, `planner`, `spec-author`,
`spec-reviewer`. `backstop-agent-guard.sh`'s `case "$AGENT_NAME" in` block
now has explicit branches for `bundle-author`, `spec-author`, `adr-author`,
`issue-author`, `planner`, `directive-author`, `implementer`, and a combined
`spec-reviewer|plan-reviewer|impl-reviewer` branch, plus a `general-purpose`
branch for an agent that has no `.claude/agents/*.md` file at all.
`directive-author`'s case was added by hand as the fix for the incident
above — the roster and the guard agree right now, but only because a human
closed the gap manually this session, with nothing left behind to catch the
next drift.

Notably, `bundle-reviewer` (one of the 11 defined agents) has **no** explicit
case — it falls through to the default `*) block` branch, the same branch a
genuinely-unregistered agent name would hit. That may well be the correct
behavior (reviewers arguably shouldn't write files at all), but it is
indistinguishable, by inspection or at runtime, from an oversight — which is
exactly the ambiguity this issue is about. There is currently no way to tell
"this agent is deliberately denied" from "this agent was never wired up."

## Impact

- A new agent role is unusable for its core purpose (writing its artifact
  type) until someone notices the silent block and manually patches the
  hook — a discovery tax paid by whichever session happens to hit it first.
- The Bash-bypass path means the guard's enforcement is not just
  fail-closed-but-silent, it is fail-open-and-silent for any agent that
  reaches for a shell command instead of `Write`/`Edit` — a bigger problem
  than the missing case alone, since it means the guard can be defeated by
  accident, not just blocked by accident.
- No signal exists today to distinguish "this agent role is deliberately
  denied all writes" from "this agent role was never registered in the
  guard" — both look identical (fall-through to default block).

## Fix direction (for the planner — not decided here)

A self-consistency check that fails loudly when the agent roster and the
guard drift apart:

- For every agent defined in `.claude/agents/*.md`, assert a corresponding
  `case` branch exists in `backstop-agent-guard.sh` — either a dedicated
  branch or explicit inclusion in a combined branch (e.g.
  `spec-reviewer|plan-reviewer|impl-reviewer`). Optionally assert the
  reverse too: no guard case should reference an agent name that no longer
  has a `.claude/agents/*.md` file.
- Where the check should live is an open question for the planner to weigh,
  not a decision made by this issue:
  - a Go test in the repo's test suite (runs under `go test ./...` /
    the gate's test step) — probably the lightest first cut, since it needs
    no new machinery;
  - a dogfood gate rule (consistent with backstop's "dogfood rules as
    packs" posture, but heavier to stand up for a config-file pairing check
    that has nothing to do with source code);
  - a lint the hook script itself runs at hook-invocation time (catches
    drift at the point of failure, but is the least visible — it would only
    fire *after* an agent is already trying to write and getting blocked).
- Do not scope this issue as also covering the Bash-bypass gap (guard only
  matches `Write`/`Edit`, not `Bash`) — that is a related but separate hole
  in the guard's coverage, worth its own issue if it isn't already tracked,
  not bundled into a config self-consistency check.

### Deeper smell (note only, not this issue's scope)

The guard is an allow-list keyed by agent name, so it is inherently coupled
to the agent roster by construction — the self-check above only makes that
existing coupling visible and loud, it doesn't remove it. A future refactor
could instead have each agent declare its writable path glob(s) in its own
`.claude/agents/*.md` frontmatter, with the guard reading that declaration
directly instead of maintaining a parallel case statement. That would
collapse the two lists into one and make the drift structurally impossible
rather than merely detected. That refactor is out of scope here — this
issue is about making today's drift loud, not about redesigning the guard's
mechanism.

## References

- `.claude/hooks/backstop-agent-guard.sh` — the `case "$AGENT_NAME" in`
  block that must stay in sync with the agent roster
- `.claude/agents/` — the 11 agent definitions (`adr-author`,
  `bundle-author`, `bundle-reviewer`, `directive-author`, `impl-reviewer`,
  `implementer`, `issue-author`, `plan-reviewer`, `planner`, `spec-author`,
  `spec-reviewer`) the guard must have a case for
- DIR-016 (directive-issue-plan-lifecycle-hardening) — parent directive;
  this issue is one of three sourced from it (alongside ISSUE-042, ISSUE-043),
  covering DIR-016's "config self-consistency is unenforced" acceptance
  criterion
- Incident: 2026-07-06 session authoring `directive-author` — the missing
  case was discovered and hand-patched, and one `directive-author` instance
  bypassed the guard entirely via `Bash` while another correctly stalled on
  `Write`, surfacing both the silent-block and the Bash-bypass behavior in
  the same session
