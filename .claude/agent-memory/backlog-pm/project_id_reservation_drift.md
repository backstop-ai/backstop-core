---
name: id-reservation-drift
description: The predicted post-remote artifact-ID collision FIRED on 2026-08-18 (ISSUE-167) and RECURRED hours later as an untagged phantom (ISSUE-171); the remote now exists, ISSUE-090 never landed, and both issues are RELEASED with no live directive home
metadata:
  type: project
---

Artifact ID allocation reads **git tags first** (`GitTagResolver.Resolve`,
`pkg/scaffold/idresolver.go` — `max(backstop/<type>/NNN) + 1`, never looks at
disk), with a silent `LocalScanResolver` filesystem fallback on any git
failure. The two views never reconcile, so a fallback allocation is
**tag-lossy**.

**THE PREDICTION FIRED — 2026-08-18.** DIR-001's ISSUE-090 note said the
hazard's trigger was its own launch action ("the moment the remote is added,
`FetchTags` starts succeeding and the tag path reactivates"). `git remote -v`
now returns `origin https://github.com/backstop-ai/backstop-core.git`;
ISSUE-090 never landed; `backstop artifact new issue` then handed out an
already-committed ID (**164**) and re-issued **167**, in ordinary
**sequential** one-at-a-time calls. Filed as `ISSUE-167 "Artifact New Issue Id
Collision On Fallback Path"` (deliberately evidence-only, no root-cause
claim). No duplicate IDs reached disk — caught pre-commit, damage nil so far.

**★ SECOND SIGHTING, SAME DAY — the OTHER polarity (2026-08-18T14:52Z).**
`issues/ISSUE-171-sandbox-helper-doc-comment-stale-probe-claim.issue.md` was
written to disk **two minutes before** `ISSUE-169`, whose slug it duplicates
character for character, with an **empty body** and **no `backstop/issue/171`
tag** — while 169 and 170, created *after* it, both got real tags. So the
allocator can also **skip ahead and leave no tag**: a numbered file exists that
the ledger has never heard of, and 171 is still free to be handed out again.
Escalated 2026-08-18T15:12Z recommending outright deletion (unledgered, empty,
unreferenced, uncommitted — nothing to strand); NOT acted on, retiring an
artifact is outside the PM grant.

**★ THIRD SIGHTING, SAME DAY (2026-08-18T20:32Z) — now BURNT TAGS with no
file.** `git tag -l 'backstop/issue/17*'` returns 170,171,172,173,174,175 while
`ls issues/` shows 170,172,174,175 only: **171 and 173 are tagged with no
artifact on disk.** 171 is the empty phantom above (its file has since
disappeared, the tag has since appeared); 173 was never seen in any form. So
within 48h the allocator has produced all three polarities — reused ID, file
without tag, and now tag without file. ISSUE-175 itself IS correctly tagged and
allocated, so this is intermittent, not a total break.

**Neither issue has a directive home, and that is correct.** ISSUE-090 is
cited only by `DIR-001` (`done`), so it is **released** under the 2026-08-17
ruling — see [[project_homed_but_orphaned_bundles]]. ISSUE-167 inherits the
same status (same file, same mechanism). Escalated 2026-08-18 recommending the
two be planned as ONE lane on `issue → plan`. Rejected homes worth not
re-deriving: DIR-016 (lifecycle hardening — the natural fit) is `done`;
DIR-021 is the gate's traceability substrate, not the scaffold allocator.

**First suspect for whoever plans it:** `idresolver.go` was modified
**2026-08-15** (`124374b`, SPEC-068) — its first change since 2026-04-05, three
days before the collisions. PLAN-SPEC-068 TASK-019's "AS BUILT" note admits an
unplanned round-two edit to *the local-scan fallback* (`ProjectRoot` string →
typed `Root`, resolution moved above `ResolveID`) because the read path
"restarted numbering at 001." Not a diagnosis; just the only recent change to
the exact path.

**How to apply:** the tag ledger currently **lags** disk again (max tag 167 vs
`ISSUE-168` on disk, untagged). So: never infer that a missing
`backstop/<type>/NNN` tag means an artifact was hand-created, and never reason
about ID burn/reuse from tags alone — check `git tag -l "backstop/issue/*"`
against `ls issues/` before any ID-safety claim, and check
`ls issues/ | grep -oE '^ISSUE-[0-9]+' | sort | uniq -d` for real duplicates
before ranking off a collision report ([[feedback_verify_the_loss_claim]]).
Contrast the inverse failure in [[project_phantom_filed_issues]], where tags
LED files against artifacts nobody wrote — and never reclaim a burnt ID.
