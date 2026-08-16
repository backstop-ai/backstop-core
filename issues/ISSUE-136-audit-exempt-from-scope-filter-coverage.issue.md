---
title: "No audit of findings-engine bindings for missing exempt_from_scope_filter — ISSUE-129's own Direction item (b) is unaddressed"
schema_version: issue/v1

issue:
  id: ISSUE-136
  title: "No audit of findings-engine bindings for missing exempt_from_scope_filter — ISSUE-129's own Direction item (b) is unaddressed"
  type: technical-debt
  status: open
  created: "2026-08-16"

complexity:
  scope: cross-cutting
  uncertainty: exploratory
  risk: moderate
---

# No audit of findings-engine bindings for missing exempt_from_scope_filter — ISSUE-129's own Direction item (b) is unaddressed

## Problem

`exempt_from_scope_filter` (SPEC-041 REQ-004, `pkg/pack/engine/binding.go:261`) is a per-binding
boolean, declared individually on each pack engine, that decides whether an out-of-scope
violation still REDs a diff-scoped `backstop gate` (`ProjectWide` → survives
`filterViolations`, `pkg/gate/scope.go:302-326`) or gets silently filtered away. It is NOT
derived from `gate_type`, `scope_kind`, or any other structural signal — it is a bare,
independently-set flag per binding, so any pack engine that legitimately needs whole-module
reasoning (a build break or test failure in file A can be caused by a change to file B) but
lacks the flag reproduces exactly the ISSUE-129 class of bug: a diff-scoped gate PASS coexisting
with a real, silently-discarded, out-of-scope violation.

ISSUE-129 fixed exactly one instance — the `backstop-ai/go-toolchain` pack's `go-test` engine —
by declaring `exempt_from_scope_filter: true` to match its `go-build` sibling. ISSUE-129's own
"Direction" section named a second, broader action as explicitly out of its own scope:

> (b) audit EVERY other findings engine lacking the flag, since it is declared per-binding and
> NOT derived from `gate_type`, so any future pack engine can silently reintroduce this gap
> unless something asserts intent engine-by-engine.

(`issues/ISSUE-129-go-test-engine-scope-filtered-cross-file.issue.md`, Direction item 2; restated
verbatim in `directives/DIR-032-gate-verdict-honesty.directive.md:546` as item (b).)

That audit has not happened. Verified directly against the live tree (2026-08-16): across every
non-testdata `pack.yml` in the repo, only two engines currently declare
`exempt_from_scope_filter: true` — `go-build` and `go-test`
(`.backstop/packs/backstop-ai/go-toolchain/pack.yml:73,93`) and `go-arch-lint`
(`.backstop/packs/backstop-ai/backstop-core-architecture/pack.yml:17`, `gate_type: findings`).
Every other declared engine across the installed packs — `semgrep`, `ast-grep`, `sandbox`,
`config-file` (`packs/base-engines/pack.yml`), `grep`/`ast-grep-contracts`
(`packs/contracts/pack.yml`), `ast-grep-substantiveness` (`packs/substantiveness/pack.yml`),
`semgrep-ci` (`.backstop/packs/backstop-ai/ci-workflows/pack.yml`), and `golangci`
(`.backstop/packs/backstop-ai/go-toolchain/pack.yml:106`) — declares no
`exempt_from_scope_filter` key and therefore defaults to `false` (filtered by diff scope like
any ordinary in-scope-only finding).

No artifact currently asserts, engine-by-engine, whether that default-false state is CORRECT for
each of those engines or is itself an undetected instance of the ISSUE-129 gap. `go-arch-lint`
already independently arrived at `exempt_from_scope_filter: true` for a `gate_type: findings`
engine (architecture-lint findings can legitimately originate whole-module, same as build/test),
which is direct evidence the semantics are NOT tied to `gate_type` and that findings-family
engines are not automatically safe to leave unaudited just because `go-test`/`go-build` (the
`gate_type: build`/`gate_type: test` engines) were the ones fixed.

## Why this matters

Per SPEC-041, the flag is declared per-binding with no structural derivation and no gate-side
default-safety check — a future pack author (internal or third-party) can trivially omit it
without any signal, silently reproducing ISSUE-129's exact failure mode: `backstop gate`
(diff-scoped, the default and the only CI-blocking invocation) reports PASS while a
project-wide-scope engine's real, out-of-scope violation is discarded pre-verdict. This is the
same class of defect DIR-032 (Gate Verdict Honesty) exists to close, and it is currently
open-ended — nothing bounds how many of the eight-plus non-exempt engines above are
mis-declared versus correctly non-exempt.

## Solution

Not prescribed here — this is an audit/coverage task, not a fix for a specific engine. Direction:

1. For each currently-declared engine binding lacking `exempt_from_scope_filter` (enumerated
   above), determine whether its violations can legitimately originate from a file OTHER than
   the one they're reported against (whole-module/project-wide reasoning, per the `go-build`/
   `go-test`/`go-arch-lint` precedent) or whether file-scoped filtering is correct for it
   (e.g. a lint rule whose violation is intrinsically local to the reported file).
2. For any engine judged to need the flag, file it as its own defect (pack-manifest fix, version
   bump, relock — same mechanism as ISSUE-129's fix) rather than bundling fixes into this issue.
3. Consider whether `backstop pack check`/`pack test` (`pkg/packval`) should gain an advisory
   check that surfaces engines with `scope_kind: project-wide` and no explicit
   `exempt_from_scope_filter` key, so a pack author gets a prompt to make the decision
   consciously rather than silently defaulting to `false`. Not mandated here — a design decision
   for whoever picks this up, and per the zero-baked-language law any such check must be a pack
   rule, not baked core logic.

## References

- `issues/ISSUE-129-go-test-engine-scope-filtered-cross-file.issue.md` — the issue whose own
  Direction section names this audit as out-of-scope for it.
- `directives/DIR-032-gate-verdict-honesty.directive.md:546` — restates the same direction item
  (b) as part of ISSUE-129's charter; confirms (as of the note at line 614) zero in-flight
  coverage of the audit itself.
- `pkg/pack/engine/binding.go:251-261` — `ExemptFromScopeFilter` field declaration and doc
  comment.
- `pkg/gate/scope.go:302-326` — `filterViolations`, the consumer that keys the survive/drop
  decision on this flag (via `Violation.ProjectWide`).
- `specs/SPEC-041-coverage-reimpl-checktype-catalog.spec.md` — REQ-004, the spec that introduced
  the declared property and the engine-path bridge.
- Not a duplicate of ISSUE-129 (which fixes exactly one engine's declaration) or of any other
  open issue — searched `issues/` and `bundles/` for prior coverage of "audit every engine for
  exempt_from_scope_filter" and found none; this is the first artifact to own the audit itself.
