---
title: "backstop-ai/go-toolchain leaves golangci and go-coverage as undeclared exempt_from_scope_filter decisions"
schema_version: issue/v1

issue:
  id: ISSUE-155
  title: "backstop-ai/go-toolchain leaves golangci and go-coverage as undeclared exempt_from_scope_filter decisions"
  type: technical-debt
  status: open
  created: "2026-08-17"

complexity:
  scope: isolated
  uncertainty: known
  risk: safe
---

# backstop-ai/go-toolchain leaves golangci and go-coverage as undeclared exempt_from_scope_filter decisions

## Problem

PLAN-ISSUE-136 shipped (backstop-core commit `f0dd714`) a `pack check` advisory that fires when
a project-wide-scoped findings engine omits the `exempt_from_scope_filter` key — i.e. no pack
author ever recorded an explicit decision for whether that engine's violations should survive
diff-scope filtering (`pkg/gate/scope.go` `filterViolations`) or be silently dropped when they
originate outside the changed file set. Running that audit against the installed
`backstop-ai/go-toolchain` pack (`.backstop/packs/backstop-ai/go-toolchain/pack.yml`, currently
locked in this repo's `backstop.lock`) surfaces exactly two pending engines:

- `golangci` (`pack.yml:116-125`, `scope_kind: project-wide`, `gate_type: lint`) — no
  `exempt_from_scope_filter` key.
- `go-coverage` (`pack.yml:46-58`, `scope_kind: project-wide`, `gate_type: coverage`) — no
  `exempt_from_scope_filter` key.

Both are advisory-only findings from `pack check` — neither blocks a gate today. On inspection,
both are ALREADY correctly non-exempt in practice; this issue is pure record-keeping debt, not a
behavioral bug:

- **`golangci` should stay non-exempt.** ISSUE-070 (`gate-diffscope-leaks-projectwide-lint`)
  established that lint violations are intrinsically file-local (unlike `go-build`/`go-test`,
  where a change to file A can break file B) and that diff-scope filtering is the CORRECT
  behavior for this engine — flipping `golangci` to `exempt_from_scope_filter: true` now would
  regress that already-shipped/delivered fix.
- **`go-coverage` should be recorded false, but for a different reason than "filtering is
  correct."** Its `gate_type: coverage` routes it through the separate coverage-records channel
  (`dispatchPackCoverage` / `ParsePackCoverage`, per `docs/CODEBASE-MAP.md`'s Engines section),
  which never reaches `filterViolations` at all — the scope-filter question does not apply to it
  in practice. The key should still be recorded explicitly (`exempt_from_scope_filter: false`,
  mirroring its actual non-participation in `filterViolations`) purely so the `pack check` audit
  stops flagging it as an open decision; there is no scope-filtering behavior to change.

## Why this matters

Low severity — advisory-only, no blocking impact today. But it is exactly the kind of silent gap
ISSUE-136's audit was built to surface: an engine binding with no recorded intent is
indistinguishable, at a glance, from an engine that was simply never considered. Leaving it
unrecorded means every future `pack check` run against `backstop-ai/go-toolchain` keeps
re-raising the same two advisories with no way to tell a reviewer "this was already decided" from
"this was never looked at."

## Solution

Not prescribed in detail — the decision is already made (see Problem); this is a mechanical
pack-side edit:

1. In the `backstop-ai/go-toolchain` pack's OWN repo (not backstop-core), add
   `exempt_from_scope_filter: false` to both the `golangci` and `go-coverage` engine bindings in
   `pack.yml`, with a short comment recording the rationale above (lint is file-local per
   ISSUE-070; coverage never reaches `filterViolations` per its `gate_type: coverage` routing).
2. Bump the pack's version and tag the release (git-tag-based resolution — no matching tag means
   `pack add`/`pack update` cannot resolve the bump).
3. Run `backstop pack relock` in this repo to pick up the new version and confirm `pack check`'s
   advisory no longer fires for either engine.

**Dependency / related issue, not owned here:** ISSUE-137 tracks the separate, known lockstep
hazard that the in-repo fixture mirror at
`cmd/backstop/testdata/go-toolchain/.backstop/packs/backstop/go-toolchain/pack.yml` can drift
from whatever real pack version gets tagged. Whoever picks up this issue should keep that
fixture in sync with the released `pack.yml` change as part of step 1-2 above, but the guard
itself (or lack thereof) is ISSUE-137's scope, not this issue's.

## References

- PLAN-ISSUE-136 / ISSUE-136 — the audit that shipped the `pack check` advisory this issue is
  responding to (backstop-core commit `f0dd714`).
- `issues/ISSUE-070-gate-diffscope-leaks-projectwide-lint.issue.md` — establishes why `golangci`
  must remain non-exempt; flipping it would regress that fix.
- `issues/ISSUE-137-pack-fixture-drift-no-guard.issue.md` — the related-but-distinct fixture/
  release parity hazard; referenced as a dependency, not duplicated here.
- `.backstop/packs/backstop-ai/go-toolchain/pack.yml:46-58,116-125` — the two pending engine
  bindings.
- Not a duplicate: searched `issues/` and `bundles/` for prior coverage of applying
  `exempt_from_scope_filter` decisions to the `golangci`/`go-coverage` bindings specifically and
  found none — ISSUE-136 is the audit that surfaced this, ISSUE-137 covers only fixture drift.
