---
name: pm-trigger-hook-is-wrong-in-both-directions
description: The pm-trigger hook both MISSES real artifacts (Write-only matcher vs `backstop artifact new`+Edit) and FABRICATES them from test fixtures (rootless path glob matches testdata/) — pending.log is neither complete nor trustworthy
metadata:
  type: project
---

`.claude/hooks/backstop-pm-trigger.sh` fails in **both** directions. Confirmed 2026-08-14.

## Direction 1 — MISSES real artifacts (false negatives)

Registered in `.claude/settings.json` as **`PostToolUse` with `matcher: "Write"`** — fires
only on the **Write tool**. But the mandated creation path is `backstop artifact new` (see
[[scaffold-via-cli]]), which scaffolds through the CLI in **Bash**. If the author then fills
the scaffold with **Edit**, Write never touches the file and the hook **never fires**.

Confirmed misses: ISSUE-121 (2026-08-13) and ISSUE-122 (2026-08-14), both absent from
`pending.log`, both triaged only because the team lead spotted them by hand. Mechanism read
out of the hook + settings, not proven against those two files — strong hypothesis. Nasty
because it correlates with *correct* workflow: the more faithfully an author follows the
scaffold-via-CLI rule, the likelier their artifact is never triaged.

## Direction 2 — FABRICATES artifacts from test fixtures (false positives)

The artifact test is a **bare path glob with no notion of a project root**:
`case "$FILE_PATH" in *issues/*.issue.md|*bundles/*.bundle.md)`. So any `.bundle.md` under
any `bundles/` dir **anywhere** qualifies — `testdata/`, nested fixture projects,
`.backstop/packs/`. The untracked-file guard does not save you: fixtures are untracked too,
which is exactly why they pass it.

2026-08-14T19:09Z: three `cmd/backstop/testdata/layout-profiles/**/BUNDLE-001-sample.bundle.md`
fixtures fired three detached PM runs 7s apart. They were TASK output of
`PLAN-SPEC-068-trustworthy-green-guards` (the plan names the paths verbatim in its file
scope). **This recurs** — SPEC-069 (`init`) and SPEC-070 (`doctor`) also create fixture
projects. **Confirmed 2026-08-15T18:19Z**: SPEC-070's `cmd/backstop/testdata/doctor/projects/`
fixtures fired two more runs 15s apart (only 2 of its 24 fixture projects are artifact-shaped).
SPEC-069 (`init`) is still pending and will make three. Both 18:19Z lines were marked and one
consolidated INBOX entry written by the FIRST sibling — the second sibling (me) lost the race,
had already appended a duplicate INBOX entry, and deleted its own. **Check the INBOX for a
sibling entry immediately before writing, not just after** — the race window is the whole triage.

**Recurrence confirmed 2026-08-15T18:19Z, exactly as predicted:** two
`cmd/backstop/testdata/doctor/projects/{layout-unconfigured-dot-backstop/.backstop,
layout-unconfigured-expected-layout}/bundles/BUNDLE-001-sample.bundle.md` fixtures from
`PLAN-SPEC-070-backstop-doctor` (file scope lines 2107/2111) fired two more runs 15s apart.
Nothing had changed in the hook; the 2026-08-14 ask was still unruled. When this fires again,
the entry writes itself in ~3 tool calls: confirm the path shape, `grep -rln` `plans/` for the
path to name the plan, write ONE terse entry that **leads with the prediction record** (the
recurrence is the finding; the root cause is already on file), mark the lines. Do not re-derive
the root cause or re-read the corpus.

**Direction-1 recurrence, 2026-08-17T00:24Z — worst ratio yet: a FOUR-artifact batch, and the
hook logged ONE.** `PLAN-ISSUE-091` TASK-006 filed ISSUE-149/150/151/152 in the same minute;
only 150 reached `pending.log`. When a hook-delivered issue cites a plan's "file, don't absorb"
task, **immediately enumerate that task's mandated filings** (`grep -n "TASK-00N" plans/<plan>`)
and check each against `pending.log` — the batch, not the one artifact you were handed, is the
real triage unit. See [[record-only-consequence-filings]].

Two mechanical details that matter when the fix is finally scoped: the dedupe guard is
**per-path** (`grep -qF "$REL"`, line 24), so N fixture projects = N detached runs, never one;
and the scoping fix is already half-written in the script — line 19 computes `REL` by
stripping `$PWD`, but the `case` on line 13 tests the raw absolute `$FILE_PATH`. Moving that
test below line 19 and anchoring it (`issues/*.issue.md|bundles/*.bundle.md`, no leading `*`)
root-scopes the hook in one line.

The irony names the fix: SPEC-068 REQ-007 gives backstop's own `DiscoverArtifacts` an
`artifact.Root` + root-relative exclusion precisely so the CLI stops confusing nested/testdata
trees with real ones. The hook needs the same root-awareness the binary is getting.

**Why:** one narrow trigger, wrong both ways, means `pending.log` is neither a complete nor a
clean stream — and every false-fire burns a real headless PM run.

**How to apply:**
- Never state or imply "the backlog is clean" / "nothing is pending" from the INBOX alone.
  On any sweep, **enumerate `issues/` and `bundles/` against `pending.log`** and report every
  artifact the hook never logged. Standard sweep step. Also check `git status --porcelain
  issues/` — the misses so far were all untracked.
- On any hook-delivered path, **check the path shape FIRST, before grounding in BACKLOG.yml**.
  If it contains `testdata/`, `.backstop/packs/`, or sits under a nested fixture project, it
  is almost certainly not an artifact — grep `plans/` for the path to find the plan that
  created it, write one consolidated escalation, mark `[triaged]`, stop. Cheap triage;
  don't read the directive corpus for a fixture.
- Fixture bursts arrive as **siblings seconds apart** (one PM run per file). Cover them all
  in ONE INBOX entry and say so, so duplicate sibling entries read as one finding.
  See [[concurrent-pm-triage-races]].
- **Before appending anything, check whether a sibling already covered the burst**: `grep -c
  "<burst marker>" .backstop/pm/INBOX.md` and `tail` `pending.log` for `[triaged]` on your own
  line. If both are already handled — as happened at 19:09Z, where the first sibling wrote the
  consolidated entry AND marked all three lines — **write nothing and stop.** A duplicate
  entry is pure noise in a file Brandon reads by hand. Losing the race is the good outcome.
- Do NOT file the fix issue yourself (recursion through the same hook) and do NOT edit the
  hook — escalate. Fold both symptoms into ONE issue: add `Edit`/Bash capture for the misses,
  add root-scoping for the false-fires.

Related: [[homed-but-orphaned-bundles]], [[phantom-filed-issues]] (INBOX presence is not
proof of existence; direction 1 is the converse), [[triage-races-plan-scaffold]] (an artifact
can be mid-write when you read it).
