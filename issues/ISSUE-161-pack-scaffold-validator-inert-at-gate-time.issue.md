---
title: "backstop pack new's scaffolded sample validator genuinely discriminates at pack test, but is structurally inert at real gate time"
schema_version: issue/v1

issue:
  id: ISSUE-161
  title: "backstop pack new's scaffolded sample validator genuinely discriminates at pack test, but is structurally inert at real gate time"
  type: question
  status: open
  created: "2026-08-17"

complexity:
  scope: isolated
  uncertainty: novel
  risk: safe
---

# backstop pack new's scaffolded sample validator genuinely discriminates at pack test, but is structurally inert at real gate time

## Problem

`PLAN-ISSUE-146` fixed `ISSUE-146` (`backstop pack new` shipping a sample sandbox validator that
always exited 0): the scaffolded validator now flags a marker string
(`BACKSTOP-SAMPLE-VIOLATION`) and the paired fixtures carry/omit it, so `pack check`/`pack test`
on a freshly-scaffolded pack now exercise a genuine pass/fail signal (`pkg/pack/scaffold.go:120-122`,
`:167-182`, `:208-230`).

That fix does not extend to real `backstop gate` time, and this is documented candidly rather than
papered over — `pkg/pack/scaffold.go:124-132`, `scaffoldEnginePack`'s own doc comment:

> At GATE time it enforces nothing, and the reason is worth stating plainly rather than papering
> over. The sample rule declares no `input_scope`, so `runSandboxEngine` hands the validator
> exactly ONE argument — the project ROOT DIRECTORY — and the validator's `[ -f "$target" ] ||
> continue` guard skips it. The scaffolded pack is therefore gate-green with NO tool the author
> must install, but because the sample scans NOTHING there, not because it looked and found a
> project clean. Naming the guard here is deliberate: the no-op is by design, not a bug. An author
> who wants the rule to actually scan project files must declare `input_scope` and swap in real
> detection logic.

The generated `pack.yml` restates the same point inline (`scaffold.go:214-217`, the validator
script's own comment): at gate time the rule has no `input_scope`, so `runSandboxEngine` hands it
the project root directory as its sole argument, and `[ -f "$target" ] || continue` correctly
skips a directory target — meaning the sample scans zero files on the one surface (the real,
ongoing gate) that matters most in production. Functionally, at gate time, this is identical to
the pre-`ISSUE-146` always-`exit 0` validator: green, but for having looked at nothing.

## Why this is a question, not a bug report

`ISSUE-146`'s own scope was `pack test`/`pack check` discrimination, and that scope is fully
delivered — nothing here contradicts or regresses it. This issue is a distinct, open product
decision: should the scaffold ALSO wire the sample rule to be live at gate time, and if so, at
what durable cost to every consumer who runs `backstop pack new`? Two shapes exist and neither is
obviously correct:

**Option A — declare `input_scope: single-file` (or equivalent) on the scaffolded rule.**
`runSandboxEngine` would then walk and invoke the validator per project file instead of once
against the root directory, so a fresh scaffold's sample rule would genuinely scan real files at
every gate run from the moment it's created — onboarding-friendly, and "green because it looked
and found nothing" becomes true rather than misleading. The cost is real and durable: every
consumer who scaffolds a new pack now pays a per-file subprocess-spawn cost on every gate run,
forever, for a sample rule most authors will promptly replace — not a one-time onboarding cost.

**Option B — keep the current directory-argument shape, and be explicit in author-facing
docs/comments that the sample requires the author to add `input_scope` (and real detection logic)
themselves before it does anything at gate time.** No added per-gate-run cost for consumers who
never touch the sample rule; the tradeoff is that the honest doc-comment above lives in
`pkg/pack/scaffold.go`, a file scaffold authors never read — the CLI's own user-facing help text
and/or the generated `pack.yml`'s comments are the only surface that could actually reach them,
and today neither spells out "this rule is currently a gate-time no-op; add `input_scope` to make
it live."

This is a founder-level tradeoff (onboarding-friendliness / demonstrated correctness vs. real
per-gate-run cost imposed on every future pack author), not something an implementer should
decide unilaterally — hence filed as `type: question` rather than `bug`.

## Impact

Low — no active defect. The current behavior is honestly documented in code (see quoted comment
above), and nothing currently claims otherwise; `cmd/backstop/pack_new.go`'s help text was
corrected as part of `PLAN-ISSUE-146` to no longer overclaim discrimination it doesn't deliver.
The gap is that "discriminates at test time, inert at gate time" is a genuinely confusing
asymmetry for a first-time pack author who runs `pack new`, sees `pack test` pass, and reasonably
assumes the same rule is doing something at `backstop gate` — it isn't, by design, until they add
`input_scope` themselves.

## References

- `ISSUE-146` / `PLAN-ISSUE-146` — origin: fixed the `pack test`/`pack check`-time vacuous
  validator this issue's gate-time gap sits beside. This issue does not reopen or regress that
  fix.
- `pkg/pack/scaffold.go:114-132` (`scaffoldEnginePack` doc comment) — the code's own honest
  statement of the gate-time gap this issue surfaces.
- `pkg/pack/scaffold.go:208-230` — the scaffolded validator script; the `[ -f "$target" ] ||
  continue` guard at line ~218 and its adjacent comment are the exact mechanism.
- Source: observed by `implementer-issue146` during `PLAN-ISSUE-146` implementation
  (backstop-core, 2026-08-17).
- Existence-in-world check performed 2026-08-17 before filing: `grep -rl` over `issues/` and
  `bundles/` for `sample validator`, `scaffoldEnginePack`, and `input_scope` matched only
  `ISSUE-146` (the origin bug, now fixed) and unrelated hits (`ISSUE-092`, `BUNDLE-010`,
  `BUNDLE-020`, `BUNDLE-004`, `BUNDLE-005`) with no textual or charter overlap with this gate-time
  question. No open issue or bundle already owns this decision.
