---
title: "Substantiveness Fixture Polarity Inverted"
schema_version: issue/v1

issue:
  id: ISSUE-148
  title: "Substantiveness Fixture Polarity Inverted"
  type: bug
  status: open
  created: "2026-08-16"

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
