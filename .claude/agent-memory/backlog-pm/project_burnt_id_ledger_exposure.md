---
name: burnt-id-ledger-exposure
description: ISSUE-167's real blast radius is 37 silently-burnt IDs across 4 namespaces, and the ledger_integrity gate step is a permanent stub that was never going to catch it
metadata:
  type: project
---

**ISSUE-167 is not a 3-instance bug. Measured 2026-08-19: 37 IDs are silently
burnt** — a `backstop/<type>/NNN` git tag exists, and `git log --all
--diff-filter=A` shows the artifact file was NEVER created in any commit on any
branch. Only ONE burnt ID in the whole corpus (`spec/012`) is a genuine purge.

Per namespace (tag exists, file never created):
- issues: **102, 103, 171, 173** (the sweep only ever found 171/173; a fork
  re-checking also missed 102/103 — scan all namespaces with `comm`, don't
  eyeball)
- directives: **010, 028, 029, 030, 031**
- bundles: **022-030, 032**
- specs: **020-029** (the known BUNDLE-003 fossil incident), **058-065**
- plans: 154 tags vs 144 files

The **reverse direction is now CLEAN everywhere** — zero files without tags. The
old ISSUE-089/090 "tags lag disk" hazard was reconciled by a manual backfill, so
the collision risk flipped: the ledger now runs AHEAD of disk. Burnt-ahead is
safe for collisions (next ID = max+1) but silently consumes IDs forever.

**The signature that identifies it:** a burnt tag is allocated during a commit
that creates a DIFFERENT artifact type, or none at all. `bundle/030` was
allocated during a `feat(SPEC-055) phase 8` commit; `directive/031` during a
`chore(SPEC-055): status implemented` commit; `spec/065` during a
`fix(ISSUE-116)` commit; `issue/173` during `ccc267a` "annotate ISSUE-099".
Check the allocating commit's subject — if it doesn't announce that artifact
type, it's this defect.

**Why nothing caught it — the kicker.** `ledger_integrity` is the 12th and final
gate step and has ALWAYS been a stub: `pkg/gate/step_deferred.go:30-38`
unconditionally returns `Status: "skipped", Reason: "ledger not implemented"`.
Backstop ships a gate step named for exactly this check that has never checked
anything. Frame it that way — it is a vacuous-green in backstop's own gate, not
a small scaffolder bug, and that framing is what makes it directive-sized.

**Why:** the founder needed a fast home ruling on ISSUE-167 and the filed
evidence understated it by an order of magnitude.

**How to apply:** never quote "3 instances" from the INBOX; re-run the
tag-vs-file `comm` scan across ALL namespaces before reporting. Root code is
`pkg/scaffold/idresolver.go` — `GitTagResolver` and `LocalScanResolver` never
reconcile their maxima. Related: [[project_id_reservation_drift]],
[[project_phantom_filed_issues]].
