---
title: "Substantiveness Fixture Polarity Inverted"
schema_version: issue/v1
delivered_by: PLAN-ISSUE-148

issue:
  id: ISSUE-148
  title: "Substantiveness Fixture Polarity Inverted"
  type: bug
  status: closed
  created: "2026-08-16"
  closed: "2026-08-17"

complexity:
  scope: contained
  uncertainty: known
  risk: critical
---

# Substantiveness Fixture Polarity Inverted

## Problem

`packs/substantiveness`'s own phase3 fixtures have inverted positive/negative polarity, so the
pack has never actually passed a real `phase3-fixtures` validation — the check was only ever
masked by the packval-vs-gate dispatch-machinery gaps `ISSUE-141`/`ISSUE-092` just fixed.

Proven tonight (2026-08-16) by implementer-issue092 during `PLAN-ISSUE-092` verification. This was
previously only a suspicion from a body-level code reading (a "follow-on #4/F4" note in
`PLAN-ISSUE-092`'s own authoring, explicitly WITHDRAWN pending real measurement because it wasn't
runnable until `ISSUE-141`'s Convert-application fix landed tonight). It is now measured, by running
the real engines against the real fixtures, and F4's original suspicion is confirmed correct.

Running `backstop pack test packs/substantiveness` (with an absolute path — the relative-`packDir`
sandbox behavior is a separate, already-filed issue) now correctly executes real engine dispatch
(post-`ISSUE-141`) and fails with exactly 2 errors, one per rule:

```
ERROR [phase3-fixtures/semgrep-positive] positive fixture triggered the rule (false positive)
```

### Root cause, read from the fixture bodies

`packs/substantiveness/testdata/fixtures/rules/hollow-test-go/positive.go` is declared in the pack
manifest (`packs/substantiveness/pack.yml`) as the POSITIVE fixture — i.e., by backstop's
convention, the CLEAN case that must NOT trigger the rule — but its own comment reads:

```go
// Q1 positive: a hollow test (calls a subject, asserts nothing) → RED finding.
```

That is a violating example, not a clean one.

`packs/substantiveness/testdata/fixtures/rules/hollow-test-go/negative.go` is declared as the
NEGATIVE fixture — i.e. the case that must trigger the rule — but its own comment reads:

```go
// Q1 negative: a substantive test (has an assertion) → no finding (GREEN).
```

That is a clean example, not a violating one.

The pack author used "positive/negative" to mean "positive EXAMPLE OF THE VIOLATION" rather than
backstop's actual convention (positive fixture = the clean case that should NOT trigger; negative
fixture = the violating case that SHOULD trigger, per `BUNDLE-005` REQ-011). The two fixture files'
CONTENTS are each internally correct examples of what they claim to be — the defect is purely which
manifest key (`positive`/`negative`) each is filed under. `phase3-fixtures` correctly detects the
inversion and fails, which is the check working exactly as designed.

### Two owners, not one

Per this repo's pack-distribution model (packs are external, GitHub-hosted, installed into
gitignored `.backstop/packs/`), fixing this requires two changes:

1. The in-repo copy at `packs/substantiveness/pack.yml` and its fixture files (swap the
   `positive`/`negative` fixture-path assignment under each rule's `claims[].fixtures`, or swap the
   file contents to match their declared slot — either resolves the inversion).
2. The external mirror `backstop-ai/go-substantiveness` needs the same fix, a version bump, and a
   `pack update`/relock in this repo to adopt it — the in-repo copy alone does not fix what is
   actually installed and consumed by the gate.

### Explicitly not fixed by PLAN-ISSUE-092

`PLAN-ISSUE-092`'s own design states it "no longer corrects any in-repo pack manifest" — it
deliberately stayed out of pack-content territory. Confirmed `packs/substantiveness/pack.yml`
untouched by that lane.

### Impact

12 `cmd/backstop` tests currently fail against this pack for this reason (post-`ISSUE-141`,
pre-this-fix) — `pack add` reports "2 validation error(s)" (3 for zero-match variants). This is a
real, currently-failing state, not hypothetical.

## Notes

- Same "vacuous/wrong-verdict" defect family as `ISSUE-146` (`pack new`'s vacuous scaffold
  validator) — both are "fixture content doesn't actually discriminate correctly," but this one is
  a polarity swap on an otherwise-correct pair, not a validator that can never discriminate at all.
- Likely fit is `DIR-032` ("Gate Verdict Honesty") given the vacuous/wrong-verdict shape shared with
  `ISSUE-092`/`ISSUE-146`/other `DIR-032` members; the charter call belongs to backlog-pm/
  directive-author triage, not to this filing.
- Existence-in-world check performed 2026-08-16 before filing: searched `issues/` for
  `substantiveness`, `hollow-test`, and `polarity`, and `bundles/` for the same. No open issue or
  bundle charter already owns this defect — `ISSUE-092` (phase3 dispatch dead code) and `ISSUE-146`
  (`pack new`'s vacuous validator) are related-family siblings, not duplicates: both are
  dispatch/authoring gaps in different code, neither touches
  `packs/substantiveness/pack.yml`'s fixture polarity.

## Resolution

The real root cause was narrower — and different — than this issue's own originally-stated fix
menu ("swap the `positive`/`negative` fixture-path assignment under each rule's `claims[].fixtures`,
or swap the file contents to match their declared slot — either resolves the inversion"). That
menu was measured wrong: both `packs/substantiveness` rules (`hollow-test-go`,
`weak-assertion-go`) ship in one shared `ast-grep/sgconfig.yml`, so a fixture's `pack test`
`Passed` result was answering "did the whole ast-grep engine fire on this file", not "did THIS
rule fire" — a manifest-key swap alone changes nothing about which rule's pattern a given fixture
body actually trips. Direct measurement confirmed this: the key-swap and a content-swap produced
byte-identical failure output. Correcting this framing is filed separately as `ISSUE-159` (a prose
correction to this issue and to `DIR-032`, not a code defect in its own right — nothing shipped
under the wrong menu).

Compounding the polarity swap, both of the two currently-declared negative fixtures were ALSO
passing for the wrong reason — a sibling rule's pattern was accidentally matching them, producing
a coincidental correct-looking result that was really a second, pre-existing vacuous green. This
was corrected alongside the polarity fix rather than left latent. All four fixture bodies (positive
and negative, both rules) were re-authored so each one discriminates against the COMBINED
ruleset in the shared sgconfig, not just its own rule in isolation. `packs/substantiveness/pack.yml`
itself was left untouched — still frozen at version `1.1.0` in-repo, per this repo's pack
distribution model (the in-repo copy is not what the gate consumes).

The external mirror `backstop-ai/go-substantiveness`, which carries the identical bug, was
published at `v1.2.1` and adopted into this repo's `backstop.yml`/`backstop.lock`. That
publication was founder-authorized directly — not via a relayed agent message — and the
implementer correctly refused an earlier relayed-authorization attempt per the plan's own explicit
publication gate; that refusal was respected rather than overridden. The adoption was proven
necessary, not cosmetic, via a control: a worktree with `go-substantiveness@1.2.1` installed but
`backstop.lock` still pinned at `1.2.0` hard-fails at the `pack_lock_verification` gate step; with
the lock updated to `1.2.1`, the same worktree passes.

Measured end-to-end impact: 14 pre-existing `cmd/backstop` test failures dropped to 4, including
both of `PLAN-ISSUE-106`'s tests that had been deliberately left red pending exactly this fix —
both now pass.

The remaining 4 residual reds are a distinct, unrelated cause (a test harness's own glob-matching
patch darkening the pack's own fixture tree, not a fixture-polarity or dispatch defect) — filed
separately as `ISSUE-158` rather than fixed under this issue.

Delivered by `PLAN-ISSUE-148` (`status: completed`, committed at `94f04c0`/`2cd3945`).
