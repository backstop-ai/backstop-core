---
name: homed-but-orphaned-bundles
description: BUNDLE-004/005/008 are cited ONLY by done directives, so residue issues pointing at them have no live home — check citing directives' status before calling a bundle "homed"
metadata:
  type: project
---

A bundle cited by a directive is removed from BACKLOG.yml's `bundles:` section — but
nothing removes it from limbo when that directive reaches `done`. Three bundles are in
this state: **BUNDLE-004 "Pack Manifest and Authoring"** (`ready`, cited only by DIR-004
/DIR-005/DIR-009, all `done`), **BUNDLE-005** (`ready`), **BUNDLE-008** (`exploring`).
They read as covered while nothing active carries them.

**RULED 2026-08-14 (founder, relayed via team-lead): citation by a directive that has
since reached `done` does NOT count as homed.** Such a bundle returns to BACKLOG.yml's
`bundles:` section. Applied to BUNDLE-004 (appended at the END of the section, position =
append order, with a rationale comment naming ISSUE-121 as its visible dependent).
**BUNDLE-005 and BUNDLE-008 were NOT re-listed** — the ruling as relayed named BUNDLE-004
only, and extending it to two more bundles is a ranking act outside the grant. That
extension is still pending as of 2026-08-14; check BACKLOG.yml before assuming either way.

**Why it mattered:** BUNDLE-004 is the corpus-designated owner of every pack-manifest
surface question, and BUNDLE-003's requirements keep deferring to it (REQ-005 via the DD-7
correction; REQ-024 via ISSUE-121) — so a live requirement in the position-1 directive was
blocked on a bundle nothing carried.

**How to apply:** when an artifact's stated design owner is a BUNDLE, `grep -l BUNDLE-NNN
directives/` and CHECK EACH CITING DIRECTIVE'S `status:` — "cited" is not "homed." If all
citers are `done`, that is a NO FIT, not a slot: the fix is re-listing the bundle, not
minting a directive over scope nobody has enumerated. Re-listing stays **propose-only**
(retroactive addition to a founder-ratified list is a ranking act, not the
append-on-creation grant) — but the precedent above is now the recommendation to make. The
residue issue rides the bundle entry as a comment and stays cited by no directive; never
park it in the REQUIRING bundle's directive. Related: [[orphaned-issue-backlog]],
[[issue101-home-ruling-pending]], [[pm-write-path]] (BACKLOG.yml is hard-blocked for
backlog-pm — always dispatch directive-author with an append-only brief).

**Founder DD-7 precedent that constrains slotting here (2026-08-12):** a bundle whose
requirement needs a field in ANOTHER bundle's surface states the residue rather than
inventing the field. So residue issues must never be slotted into the REQUIRING bundle's
directive (e.g. ISSUE-121 into DIR-002) — that undoes the ruling by the back door.

**Applied 2026-08-14: all three are now listed** (BUNDLE-004 first, then 005/008 after the
founder extended the ruling). Caveat recorded in BACKLOG.yml itself — 005's and 008's
substance already SHIPPED (`pkg/packval`; `GateScopeModeAll` in `cmd/backstop/gate.go`),
and 005's residual defect ISSUE-092 is already carried by DIR-024/DIR-032. So those two
entries mark unhomed *intent*, not uncarried *work*, and BUNDLE-008 may deserve a terminal
state instead — escalated, not decided. See [[pm-trigger-hook-misses-cli-scaffolded-artifacts]].
