---
name: pm-trigger-hook-misses-cli-scaffolded-artifacts
description: The pm-trigger hook is PostToolUse(Write)-only, so artifacts scaffolded by `backstop artifact new` (Bash) and filled in with Edit never trigger triage — the INBOX is NOT a complete artifact stream
metadata:
  type: project
---

`.claude/hooks/backstop-pm-trigger.sh` is registered in `.claude/settings.json` as
**`PostToolUse` with `matcher: "Write"`** — it fires only on the **Write tool**. But the
mandated creation path is `backstop artifact new` (see [[scaffold-via-cli]]), which
scaffolds the file **through the CLI in Bash**. If the author then fills the scaffold with
**Edit**, Write never touches the file and the hook **never fires**.

**Why:** confirmed misses — ISSUE-121 (2026-08-13) and ISSUE-122 (2026-08-14) are both
absent from `pending.log`; ISSUE-121 was triaged only because the team lead spotted it by
hand. The mechanism above was read out of the hook + settings; it was NOT proven against
those two files (that needs the authoring sessions' transcripts), so treat it as a strong
hypothesis. The gap is nasty because it correlates with *correct* workflow: the more
faithfully an author follows the scaffold-via-CLI rule, the likelier their artifact is
never triaged.

**How to apply:** never state or imply "the backlog is clean" / "nothing is pending" on the
strength of the INBOX alone. On any sweep, **enumerate `issues/` and `bundles/` against
`pending.log`** and report every artifact the hook never logged — that is now a standard
sweep step, not an optional one. Also check untracked artifacts (`git status --porcelain
issues/`), since the hook additionally skips anything already tracked and the misses so far
were all untracked. Do NOT file the fix issue yourself (recursion through the same hook)
and do NOT edit the hook — escalate; the fix is small (add `Edit` to the matcher, or
capture on a Bash `PostToolUse` for `artifact new`).

Related: [[homed-but-orphaned-bundles]], [[phantom-filed-issues]] (INBOX presence is not
proof of existence; this is the converse — absence is not proof of nonexistence),
[[triage-races-plan-scaffold]] (an artifact can be mid-write when you read it; ISSUE-122
briefly failed `validate --all` with a missing title, then went green on its own).
