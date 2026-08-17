---
title: "pm-trigger hook's PostToolUse matcher is Write-only, so it never fires for the mandated artifact-authoring workflow (scaffold via CLI, then Edit)"
schema_version: issue/v1

issue:
  id: ISSUE-123
  title: "pm-trigger hook's PostToolUse matcher is Write-only, so it never fires for the mandated artifact-authoring workflow (scaffold via CLI, then Edit)"
  type: bug
  status: closed
  created: "2026-08-14"
  closed: "2026-08-17"

resolved-by: bee8873dcb62b08c6686fda4bd50773247e1cccf

complexity:
  scope: isolated
  uncertainty: known
  risk: moderate
---

# pm-trigger hook's PostToolUse matcher is Write-only, so it never fires for the mandated artifact-authoring workflow

## Problem

The `backstop-pm-trigger.sh` hook is supposed to auto-route every newly-filed issue/bundle into a
directive by logging it to `.backstop/pm/pending.log` and dispatching a detached `backlog-pm`
triage run. In practice it silently fails to fire for the standard, mandated artifact-authoring
workflow — `./bin/backstop artifact new <type> --slug <kebab>` (a Bash call that scaffolds an
empty templated file) followed by an authoring agent filling in the content via the `Edit` tool.
Neither step is a `Write` tool call, and the hook's `PostToolUse` matcher only fires on `Write`.

`.claude/settings.json`'s current `hooks.PostToolUse` array:

```json
[
  {
    "matcher": "Write|Edit|MultiEdit",
    "hooks": [
      { "type": "command", "command": ".claude/hooks/backstop-validate-artifact.sh" }
    ]
  },
  {
    "matcher": "Write",
    "hooks": [
      { "type": "command", "command": ".claude/hooks/backstop-pm-trigger.sh" }
    ]
  }
]
```

The sibling `backstop-validate-artifact.sh` hook matches `Write|Edit|MultiEdit`.
`backstop-pm-trigger.sh` matches `Write` only — an inconsistency between two hooks that are meant
to fire on the same class of event (an artifact file being written to).

`.claude/settings.local.json` was checked and does not touch `hooks` at all — it contains only a
`permissions.allow` list, so nothing there masks or already fixes this gap.

## Reproduction / evidence

Same session, both same-day (2026-08-14), both filed via the standard `artifact new` + `Edit`
workflow:

- **ISSUE-121** (`issues/ISSUE-121-pack-manifest-missing-stack-policy-surface.issue.md`) — never
  appeared in `.backstop/pm/pending.log`. Confirmed directly: `grep -c "ISSUE-121"
  .backstop/pm/pending.log` returned `0` before manual intervention.
- **ISSUE-122** (`issues/ISSUE-122-baked-ecosystem-literals-in-artifact-discover.issue.md`) — found
  completely untriaged, floating with no directive attachment. Discovered by accident when it
  briefly turned `artifact validate --all` red for an unrelated reason, not because any pending-log
  entry or notification pointed at it.

Both required a manual `backlog-pm` triage dispatch to get homed into a directive — the automation
the hook exists to provide did not happen.

## Root cause

Two independent gaps compound:

1. **Bash-created files get no `PostToolUse` coverage at all.** `artifact new` scaffolds the file
   via a `Bash` subprocess. No `PostToolUse` hook — not `backstop-pm-trigger.sh`, not
   `backstop-validate-artifact.sh` — fires for a Bash-created file at the moment of its creation.
2. **The subsequent `Edit` call (filling in the scaffold) is excluded by matcher.** The hook body
   (`.claude/hooks/backstop-pm-trigger.sh:13-16`) also gates on `tool_input.file_path` matching
   `*issues/*.issue.md` or `*bundles/*.bundle.md`, and on the file being untracked
   (`git ls-files --error-unmatch`, lines 19-20) — both checks are compatible with an `Edit` on a
   freshly scaffolded, not-yet-committed file. The ONLY reason `Edit` doesn't reach that logic is
   the matcher itself (`"matcher": "Write"` in settings.json) filtering the event out before the
   script ever runs.

Net effect: the only path that reaches `backstop-pm-trigger.sh` today is an artifact written
start-to-finish in a single `Write` call — a workflow shortcut that is itself off the mandated
`artifact new` + `Edit` convention (see project CLAUDE.md: "Never hand-create an issue file" and
"scaffold via CLI first").

## Why this matters

This is worse than "occasionally misses an issue" — the gap correlates *inversely* with correct
workflow adherence. The more faithfully an author follows the scaffold-via-CLI convention this
repo itself mandates, the LESS likely their artifact ever gets auto-triaged. Only artifacts
written off-convention (one-shot `Write`) get caught by the automation meant to catch all of them.
Untriaged issues/bundles sit with no directive lineage and no visibility until someone notices by
accident (as happened with ISSUE-122) or a manual sweep catches them.

## Resolution

Fixed directly as harness config (this repo treats `.claude/` hook scripts as harness config,
not product code needing the full artifact pipeline — no plan/spec lineage). Commit
`bee8873dcb62b08c6686fda4bd50773247e1cccf`, "fix(ISSUE-123,ISSUE-127): pm-trigger fires on
Edit; agent-guard implementer allow-list is language-agnostic."

**Mechanism:** `.claude/settings.json`'s `PostToolUse` matcher for
`backstop-pm-trigger.sh` was widened from `"Write"` to `"Write|Edit|MultiEdit"`, matching
`backstop-validate-artifact.sh`'s own matcher — option 1 of the "Direction" section below. The
hook body needed no change: `tool_input.file_path` has the same shape across `Write`, `Edit`,
and `MultiEdit` events, so the existing glob and untracked-file dedupe logic
(`.claude/hooks/backstop-pm-trigger.sh:13-20`) behaves identically once the event reaches it.

**Verification:** manually confirmed the hook now fires on an `Edit` call against a freshly
scaffolded, not-yet-committed issue file (the exact scaffold-via-CLI-then-Edit workflow named
in the Reproduction section above).

Option 2 (Bash-tool coverage scoped to `artifact new` invocations) was not pursued — widening
the matcher closes the gap without adding a second hook-coverage surface to maintain.

## Direction (original triage, superseded by Resolution above)

At minimum, the eventual plan should weigh:

1. Widening `backstop-pm-trigger.sh`'s matcher to `Write|Edit|MultiEdit`, matching
   `backstop-validate-artifact.sh`'s own matcher, so it fires on `Edit` too.
2. And/or adding `PostToolUse` coverage for the `Bash` tool scoped to `artifact new` invocations,
   so the artifact is caught at scaffold time regardless of how it's subsequently filled in.
3. Whichever is chosen, verify (don't assume) whether the hook body itself needs adjustment: it
   currently reads `tool_input.file_path` off the raw hook payload, which has a different shape for
   `Write` vs `Edit` vs `Bash` tool calls — confirm the dedupe-by-untracked-file logic
   (lines 18-20) and the pending-log dedupe (line 24) still behave correctly for whichever event(s)
   the fix adds coverage for.
4. A regression fixture proving the fix: scaffold an artifact via `artifact new` then fill it via
   `Edit` (or Bash, per whichever direction is chosen) and assert it lands in
   `.backstop/pm/pending.log` — absent that proof, any fix risks becoming another silent-gap
   regression.

## Notes / references

- Reported by `team-lead` mid-session (2026-08-14), verified directly rather than taken as a
  hypothesis: `.claude/settings.json`'s `hooks.PostToolUse` block, `.claude/hooks/backstop-pm-
  trigger.sh`, and `.backstop/pm/pending.log` were all read directly as part of authoring this
  issue.
- Not related to `bundles/runtime-hooks.bundle.md` — that bundle designs hook architecture for the
  separate `backstop-runtime` product; this issue is about a live misconfiguration in backstop-
  core's own `.claude/hooks/` tooling for this repo's Claude Code sessions.
