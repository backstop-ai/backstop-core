---
title: "Artifact Ledger Lifecycle Hardening"
number: DIR-034
created: "2026-08-19"
schema_version: directive/v1

directive:
  status: queued
  source:
    - "ISSUE-167"
    - "ISSUE-090"
    - "ISSUE-071"
    - "ISSUE-072"
    - "ISSUE-089"
    - "ISSUE-074"
---

## Description

Backstop dogfoods its own "no ledger-numbered artifact ever collides, no
closed artifact is ever vacuous" invariants — this directive is where the
gaps in that dogfooding live. Six issues, two themes:

**Theme A — the ID ledger itself doesn't reconcile (ISSUE-167 +
ISSUE-090).** `pkg/scaffold/idresolver.go` runs two independent ID
resolvers — `GitTagResolver` (primary, tag-based) and `LocalScanResolver`
(fallback, file-scan-based) — that never reconcile their maxima.
`ResolveID` only falls back from tag-path to local-scan on a
`*FallbackError`; it never merges the two counts. ISSUE-090 is the
original root-cause filing for this defect (released under DIR-001, which
shipped only a manual tag backfill, never the resolver fix itself — the
underlying bug is still live). ISSUE-167 is a fresh, independently-observed
recurrence from 2026-08-18: two ID collisions in a short run of ordinary
sequential `artifact artifact new` calls, plus a full tag-vs-file sweep of
every artifact namespace that found **37 IDs silently burnt** — a
`backstop/<type>/NNN` tag exists but the file backing it was never created
in any commit, on any branch. Burnt: issues 102, 103, 171, 173; directives
010, 028, 029, 030, 031; bundles 022-030, 032; specs 020-029 and 058-065;
plans (154 tags vs. 144 files). Only one of the 37 (`spec/012`) is a
legitimate purge — the rest are the resolver defect's fingerprint. The
reverse direction (a file with no backing tag) is clean everywhere today —
a prior manual backfill already fixed that direction, but only that
direction. What makes this directive-sized rather than a small bug fix:
`pkg/gate/step_deferred.go:30-38` is the gate's 12th and final step,
`ledger_integrity`, and it unconditionally returns `Status: "skipped",
Reason: "ledger not implemented"`. It has never once actually checked
anything — the mechanism this directive needs to build has a step already
wired into the gate pipeline waiting for a real implementation, not a
step to add from scratch.

**Theme B — closed-issue vacuity and its relaxation list (ISSUE-071 +
ISSUE-072).** `artifact_status_drift`'s `ClassSuccessTerminal`
(`pkg/gate/status_drift.go:39-52`) only fires by iterating a closed
artifact's declared `MandatedTests` — an issue closed with an EMPTY test
list produces no violation at all, so a bare `resolved-by` close currently
sails through with zero mechanical proof it was ever actually done
(ISSUE-071). ISSUE-072 (Gap 2 specifically — Gaps 1 and 3 are a separate,
later-sequenced concern, see Notes) wants to add a THIRD closure-relaxation
path to that same conditional list: retirement-by-deletion, so a
sub-artifact can close honestly when its home artifact is deliberately
removed rather than fixed. These two issues must be planned together as
one extension of the closure-relaxation list, not sequentially — shipping
ISSUE-071's tightening alone would immediately red every retirement-close
ISSUE-072 exists to enable, and ISSUE-072's own Acceptance section already
requires that a bare vacuous close still trips ISSUE-071's check. Flagged
as a scoping question for whoever plans this pair, not resolved here: per
this repo's own CLAUDE.md convention ("no plan/spec lineage to close
against → stays open-but-fixed with a `## Resolution` section instead of
forcing ceremony"), ISSUE-072 Gap 2's formal schema/validator work may be
lower-value than it looks — worth revisiting scope before committing to
the full mechanism.

**Two smaller, independent ledger/tooling defects round out the
directive:**

- **ISSUE-089** — `backstop artifact validate <path>` silently ignores the
  path argument and validates the entire corpus instead. A vacuous-green
  command shape that a new user encounters first, on the exact command
  this directive's own process depends on.
- **ISSUE-074 (residual only)** — `pack relock` takes a filesystem PATH
  argument where every sibling pack command (`remove`/`update`/`upgrade`)
  takes a pack NAME. The silent-failure half of ISSUE-074 is already
  fixed (relock failures now print to stderr); this is only the
  arg-shape asymmetry left over. See `docs/CODEBASE-MAP.md`'s "Known gap
  — `pack relock` arg-shape asymmetry (ISSUE-074, residual)" for the
  existing writeup and the reason relock's read-modify-write shape makes
  it structurally different from its siblings.

## Acceptance Criteria

- `pkg/gate/step_deferred.go`'s `ledger_integrity` step performs a real
  check (tag-vs-file reconciliation across every artifact namespace)
  instead of unconditionally skipping.
- The 36 confirmed-burnt IDs (all but `spec/012`) are dispositioned —
  either the resolver fix prevents recurrence and the historical gaps are
  documented as accepted history, or a backfill closes them; either way
  the ledger stops silently drifting further.
- `GitTagResolver` and `LocalScanResolver` reconcile their maxima before
  `ResolveID` commits to a number, closing the collision class ISSUE-167
  observed twice in one session.
- A closed issue with zero declared mandated tests and no retirement
  marker produces a `artifact_status_drift`-family signal (ISSUE-071),
  while a genuinely retired sub-artifact closes clean under the new
  retirement path (ISSUE-072 Gap 2) — both proven together, not
  sequentially.
- `backstop artifact validate <path>` validates only the artifact at
  `<path>`, not the whole corpus (ISSUE-089).
- `pack relock` accepts a pack name, consistent with `remove`/`update`/
  `upgrade` (ISSUE-074 residual).

## Notes

Six issues, two themes, grouped under one directive per this repo's own
directive-authoring convention (granular work rolls up under a directive;
it doesn't each get its own one-issue directive). ISSUE-090 and ISSUE-167
are cited together deliberately — same root-cause code
(`pkg/scaffold/idresolver.go`), same fix, and DIR-001 (now `done`) already
demonstrates that shipping only a tag backfill without the resolver fix
lets the defect recur; splitting them would obscure that lineage.

ISSUE-072's Gaps 1 (no structural retirement primitive) and 3 (claim-grain
version blindness) are real but separate concerns from Gap 2 — they
should sequence AFTER the 071+072-Gap-2 work lands, not block it. They are
not cited as directive source here; if they mature into committed work,
they belong either as a later addition to this directive or as their own
issue-derived directive item, decided when that work is actually
prioritized.

ISSUE-089 and ISSUE-074 (residual) are independent, contained CLI-defect
fixes with no sequencing dependency on the ledger or closure-relaxation
threads above — they can be picked up in any order relative to Theme A/B.
