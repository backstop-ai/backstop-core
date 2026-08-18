---
title: "Go Contracts Mirror Inverted Fixture Polarity"
schema_version: issue/v1
delivered_by: PLAN-ISSUE-157

issue:
  id: ISSUE-157
  title: "Go Contracts Mirror Inverted Fixture Polarity"
  type: bug
  status: closed
  created: "2026-08-17"
  closed: "2026-08-18"

complexity:
  scope: contained
  uncertainty: known
  risk: moderate
---

# Go Contracts Mirror Inverted Fixture Polarity

## Problem

The published mirror `backstop-ai/go-contracts` carries the same inverted
positive/negative fixture polarity as the in-repo `packs/contracts` source
that `PLAN-ISSUE-142` corrects (its TASK-005) — and where TASK-005's fix is
purely a source-tree correction that costs nothing until dispatched, this
mirror's copy of the same defect becomes an active installation failure the
moment `PLAN-ISSUE-142`'s phase 1 lands, because `pkg/packval` starts
dispatching pattern-arg-rule fixtures for real for the first time. This issue
is filed as `PLAN-ISSUE-142` TASK-009 — a first-class residual (R1), not a
note — matching the rigor `PLAN-ISSUE-148` applies to its own analogous
mirror residual (its TASK-005, for `backstop-ai/go-substantiveness`). The two
mirrors carry the SAME DEFECT SHAPE but are DIFFERENT PACKS, so this issue
and `ISSUE-148` do not collide and neither supersedes the other.

All four facts below were measured firsthand by the `PLAN-ISSUE-142`
implementer on 2026-08-17, against the real installed manifest and real
ast-grep/packval output — not predicted.

### 1. The defect, as installed

`.backstop/packs/backstop-ai/go-contracts/pack.yml` — the mirror declared as
`backstop-ai/go-contracts: 1.2.0` in `backstop.yml` and present at that
version in `backstop.lock` — was read directly on 2026-08-17. The installed
manifest file is dated Jul 28 09:53 and self-declares `version: "1.2.0"`. It
carries the same 7 pattern-arg rules as the in-repo `packs/contracts` source,
with the same inverted fixture slots. Measured by grep of the installed
manifest: 7 occurrences of `pattern:`, 0 of `rule_path:`.

Slot readings, read line-by-line from that manifest:

- `contract-signature`: positive slot -> `testdata/fixtures/sig-mismatch.go`,
  negative slot -> `testdata/fixtures/sig-present.go` (manifest lines 59-62).
- The five kind rules (`type-signature`, `const-signature`, `var-signature`,
  `method-signature`, `interface-signature`): all five positive slots ->
  `testdata/fixtures/sig-kinds-present.go`, all five negative slots ->
  `testdata/fixtures/sig-kinds-mismatch.go` (manifest lines 78-129).
- `contract-absence`: already correct — positive `absence-clean.go`, negative
  `absence-present.go` (manifest lines 139-142). Not part of this defect.

The manifest even carries the same stale comment at line 68 explaining the
inversion as intentional: `"positive fixtures DECLARE the kind, negatives
OMIT the declared names"` — which IS the inversion, written down as if it
were the rule. Backstop's actual convention, per `BUNDLE-005` REQ-011, is the
opposite: a POSITIVE fixture is the CLEAN case a rule must NOT fire on; a
NEGATIVE fixture is the VIOLATING case it MUST fire on.

### 2. The newly created consequence

Once `PLAN-ISSUE-142`'s phase 1 lands — widening `pkg/packval`'s
`RunFixtures` dispatch-eligibility guard from "declares a rule source path"
to "declares a rule source path OR an inline pattern" — the mirror becomes
**uninstallable** via `pack add`, `pack update` and `pack upgrade`. All three
route through `RunValidationOnScratchCopy`
(`pkg/pack/distribution/command.go`, called from `AddCommand.Run` at line
257, `UpdateCommand.Run` at line 684, and `UpgradeCommand.Run` at line 828),
which runs the same `pkg/packval` pipeline this lane makes dispatch-live.

The in-repo source, which carries the identical inverted slots, was measured
failing with **9** `phase3-fixtures` validation errors across 6 of its 7
rules:

- 6x `semgrep-positive: positive fixture triggered the rule (false
  positive)` — one each for `contract-signature`, `type-signature`,
  `const-signature`, `var-signature`, `method-signature`,
  `interface-signature`.
- 3x `semgrep-negative: negative fixture not triggered` — for
  `const-signature`, `var-signature`, `interface-signature`: their declared
  negative `sig-kinds-mismatch.go` yields 0 ast-grep matches for the const,
  var and interface patterns respectively.

Record that the error count is 9, not 6: `PLAN-ISSUE-142`'s own
falsification-bar prediction said "6 of 7 rules fail with `semgrep-positive`"
and under-counted, because it omitted the three non-firing negatives that its
own measured table had already recorded. The mirror, carrying the same
slots, will fail the same way — 9 errors, not 6 — when it is next validated.

### 3. Why this is not urgent, and what would make it urgent

`pack install` does **not** validate: `NewInstallCommand(git GitCloner)`
(`pkg/pack/distribution/command.go:399`) takes no `Validator` parameter at
all, and `InstallCommand.Run` never calls `RunValidationOnScratchCopy` —
confirmed by grep of every call site in that file. CI's fleet step is
exactly `./bin/backstop pack install` (`.github/workflows/ci.yml` line 86 in
the gate job, line 249 in the baseline job). So the installed mirror keeps
restoring unchanged and the gate keeps consuming it unchanged; **CI and the
gate are unaffected** by this issue today or after `PLAN-ISSUE-142` lands.

It becomes blocking the first time anyone runs `pack update
backstop-ai/go-contracts`, `pack upgrade`, or adds `backstop-ai/go-contracts`
to a new consumer repo — at that point `pack check`'s phase-3 fixtures step
fails loud and the operation refuses.

### 4. The fix shape

The same correction `PLAN-ISSUE-142`'s TASK-005 applies to the in-repo
`packs/contracts` source, applied in the `backstop-ai/go-contracts` repo:
add two new genuinely-clean fixtures, re-point the affected positive/negative
slots at them, and correct the stale comment.

Verified with real ast-grep 0.43.0 on 2026-08-17 against the in-repo
equivalents:

- A new `testdata/fixtures/sig-absent.go` — a valid Go file in `package fix`
  with no function declaration at all — measures **0** matches for `func
  $NAME($$$PARAMS)`, and becomes `contract-signature`'s positive slot; its
  negative slot stays `sig-present.go` (measures 1 match).
- A new `testdata/fixtures/sig-kinds-absent.go` — `package fix` plus exactly
  one plain function, with no type, no `const NAME =`, no `var NAME =`, no
  method and no interface — measures **0** for all five kind patterns (`type
  $NAME $$$`, `const $NAME = $$$`, `var $NAME = $$$`, `func ($R $T)
  $NAME($$$PARAMS) $$$`, `type $NAME interface { $$$ }`), and becomes the
  shared positive slot for all five kind rules; their negative slots all
  become `sig-kinds-present.go`, which measures 3/1/1/1/1 matches
  respectively.

A simple slot swap is **not** sufficient, and the fix in that repo must not
attempt one: these self-test patterns are deliberately NAME-AGNOSTIC, so the
existing "mismatch" contrast files still match under them —
`sig-kinds-mismatch.go` itself measures 2 matches for `type $NAME $$$` and 1
for the method pattern. A name-specific contrast cannot discriminate under a
name-agnostic pattern; the clean case needs the construct wholly ABSENT, not
merely differently named.

Keep `sig-mismatch.go` and `sig-kinds-mismatch.go` on disk in that repo even
after `pack.yml` stops referencing them from these slots, if that repo has a
consumer equivalent to this one's: in backstop-core,
`pkg/pack/engine/contracts_kind_signature_test.go` reads both files by name
to assert the kind-aware compiler's name-specific output finds zero matches
in them. Deleting them reds that suite for an unrelated reason.

Then a version bump in the `backstop-ai/go-contracts` repo with a matching
git tag — the mirror is already at 1.2.0, so the next version must be
greater than 1.2.0. Note that the in-repo `packs/contracts` source was
independently bumped 1.1.0 -> 1.2.0 by `PLAN-ISSUE-142` TASK-005; these are
two DIFFERENT packs (`backstop/contracts` vs `backstop-ai/go-contracts`), so
the coinciding version numbers are not a conflict, but it is worth stating
here so nobody conflates the two bumps as one event. After the tag exists,
`pack update backstop-ai/go-contracts` + relock in backstop-core adopts it.

Publishing to that repo is FOUNDER-GATED, exactly as `PLAN-ISSUE-148`'s
phase 5 establishes for the analogous `go-substantiveness` mirror fix — this
issue does not authorize a push to `backstop-ai/go-contracts` on its own.

## Notes

- Same defect SHAPE as `ISSUE-148` (`backstop-ai/go-substantiveness`
  inverted fixture polarity), but a DIFFERENT pack (`backstop-ai/
  go-contracts`) — the two issues do not collide and neither should be
  closed by the other's fix.
- Source/parent: `ISSUE-142` (`Packval Pattern Arg Fixtures Never Dispatch`),
  filed as `PLAN-ISSUE-142` TASK-009. `DIR-032` item 16 is the directive
  roster entry covering `ISSUE-142`'s own defect family; this issue is a
  residual of the plan that fixes it, not itself a `DIR-032` roster member.
- R2 (unifying `pkg/pack.Rule` and `pkg/packval.Rule`), R3 (a phase-1
  structural check for rules declaring no engine input at all), and R4
  (`RunValidator` dispatch still gating on `rule.Layer`) are explicitly OUT
  OF SCOPE for this issue — they are `PLAN-ISSUE-142`'s other residuals,
  restated for the founder to route separately, not filed here.
- Existence-in-world check performed 2026-08-17 before filing: searched
  `issues/` and `bundles/` for `go-contracts`. No open issue or bundle
  charter already owns this defect; the two hits (`ISSUE-111`,
  `ISSUE-115`, `BUNDLE-031`) are unrelated to the mirror's fixture content.

## Resolution

The external mirror `backstop-ai/go-contracts` was bumped `1.3.0` -> `1.4.0`
(tag `v1.4.0`, commit `e21e41c`), fixing the inverted positive/negative
fixture polarity on the 6 pattern-arg signature rules
(`contract-signature`, `type-signature`, `const-signature`,
`var-signature`, `method-signature`, `interface-signature`): two new
genuinely-clean fixtures were added (`sig-absent.go`,
`sig-kinds-absent.go`), the six affected slots were re-pointed (never
swapped) to them, and the stale comment that wrote the inversion down as
if it were the rule was corrected to the real `BUNDLE-005` REQ-011
convention. `contract-absence` was untouched — it was already correctly
polarized and not part of this defect.

Verified via `pack test` going from `status: fail`, exit 1 (9
`phase3-fixtures` errors — 6 `semgrep-positive` + 3 `semgrep-negative`) to
`status: pass`, exit 0 (all six phases green), with per-rule real-ast-grep
match counts reproduced exactly: `contract-signature` 0/1,
`type-signature` 0/3, `const-signature` 0/1, `var-signature` 0/1,
`method-signature` 0/1, `interface-signature` 0/1 (positive matches /
negative matches).

A new mandated test,
`TestInstalledGoContractsPack_FixturePolarityIsCorrect`
(`pkg/pack/engine/contracts_installed_pack_polarity_test.go`), pins the
installed mirror's fixture polarity from core and went RED -> GREEN across
this fix.

Core was relocked via `pack update backstop-ai/go-contracts`, jumping
`1.2.0` -> `1.4.0` directly — skipping the still-defective `1.3.0`, which
core had never adopted precisely because it still carried this defect.

`gate --all` was verified clean of new defects via a differential
comparison against a clean-HEAD control worktree: 385 vs 380
path-normalized violation lines, with the only difference being the
removed `TestInstalledGoContractsPack_CarriesFilenameHeaderFix` failure in
the control — the differential proof that this fix removed something and
added nothing.

**Not absorbed by this close, recorded for the next reader:**

- `ISSUE-166` is now unblocked — its `pack update` blocker is gone — but
  stays open. This issue does not close it; it keeps its own issue, plan,
  claims and close-out.
- `ISSUE-174` (no guard keeps an in-repo pack source and its mirror in
  sync) stays open. This fix is a second concrete instance of the general
  gap that issue names, not a fix for the gap itself.
- `ISSUE-176` and `ISSUE-177` are unrelated and were untouched by this
  work.
- Correction to this issue's own hedge (section 7, above): the mirror's
  `sig-mismatch.go` and `sig-kinds-mismatch.go` have NO core consumer —
  `pkg/pack/engine/contracts_kind_signature_test.go` reads only the
  in-repo `packs/contracts` source's copies, via its `durablePackRel =
  "packs/contracts"` constant. They were kept for source/mirror parity,
  not because a consumer requires it.

Delivered by `PLAN-ISSUE-157` (`status: completed`).
