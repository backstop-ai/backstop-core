---
name: phantom-filed-issues
description: INBOX entries can describe issues that were never actually filed — verify every ID against issues/ and git before citing one in a proposal
metadata:
  type: project
---

An INBOX FYI or PROPOSAL is NOT evidence that an artifact exists. When a
directive-author or issue-author dispatch is refused mid-flight (harness
permissions, guard, denied tool), the PM entry still gets written describing
the artifact as real and homed — but nothing was created. Discovered
2026-08-02: `ISSUE-102` (typescript-substantiveness harness-baked globs) and
`ISSUE-103` (typescript-contracts bare-const idiom) had three INBOX entries
between them, including a PRIORITY PROPOSAL asking Brandon to work them ahead
of the entire ranked backlog — and neither file has ever existed (no file in
`issues/`, no add-commit in `git log --all --diff-filter=A`, ID sequence
jumps 101 → 104).

The nasty half: reservation tags `backstop/issue/102` and `/103` DO exist.
Tags LEAD files by two, so the numbers are permanently burnt against
artifacts nobody wrote. This is the inverse of the drift in
[[project_id_reservation_drift]], where tags LAGGED files.

**RULING (Brandon, 2026-08-02): 102 and 103 are DEAD — never refile under
those numbers, never reconstruct them, and leave the tags burnt.** Recorded
in DIR-024's Notes (where the 101 → 104 gap and ISSUE-113's harness-baked-globs
mention would otherwise send a reader hunting) and closed out across all four
INBOX entries. Precedent worth reusing: when a phantom is found, the founder's
instinct was NOT to recover the artifacts but to make the gap self-explaining
— cheaper, and it removes the recurring "is this corruption?" question. What
survives matters though: ISSUE-113 partially recaptures dead 102's substance.

**Second correction, same day, and the more reusable lesson:** I closed dead
103's `typescript-contracts` bare-const defect as a loose thread needing "a
fresh id under DIR-022 if it resurfaces." Brandon corrected it — that defect is
a PACK defect, and **packs live outside core, so it gets NO backstop-core issue
under any ID, ever**. Not "no home yet" but **"not core's to track."** Its home
is `backstop-ai/typescript-contracts`'s own tracking (working copy
`~/src/projects/backstop-typescript-contracts-pack-local`, whose `pack.yml`
still reads the pre-rename `backstop/contracts` 1.1.0 — a stale mirror).

**Why this keeps catching me:** a defect OBSERVED through a core consumer
(bclabs-portal's gate going red) reads like core's problem, and the triage
reflex is to find it a directive. Ask first whether the FIX SITE is a pack — if
the fix is a rule/manifest/glob change plus a re-tag, it is pack-side and this
backlog should not track it at all. See [[feedback_packs_always_external]] and
[[feedback_zero_baked_checks]]. Applies to homing too: never route a
pack-content defect to a core directive because that directive's theme sounds
adjacent.

**Why:** the PM writes its INBOX entry from the dispatch's *intent*, and a
subagent that reports "content staged, write refused" reads as a deferral
rather than a failure. Nothing in the loop re-checks the filesystem
afterward.

**How to apply:** before citing any issue ID in a recommendation, escalation,
or priority proposal, confirm the file exists — `ls issues/ISSUE-NNN-*`. Do
this especially for IDs you learned about from the INBOX rather than from the
pm-trigger hook, and for any entry whose Action-taken line mentions a blocked,
refused, or staged write. When a dispatch reports a refused write, re-verify
and log the FAILURE explicitly rather than the intent. Never try to reclaim a
burnt ID — file under a fresh number; re-using a tagged number is the exact
collision [[project_id_reservation_drift]] exists to prevent.
