---
title: "Gate Checker Hardening"
number: DIR-015
created: "2026-07-06"
schema_version: directive/v1

directive:
  status: active
  source:
    - "ISSUE-036"
    - "ISSUE-034"
    - "ISSUE-035"
    - "ISSUE-037"
    - "ISSUE-038"
    - "ISSUE-039"
    - "ISSUE-040"
    - "ISSUE-041"
---

## Description

Make backstop's own dogfood gate checks CORRECT and NON-VACUOUS. The 2026-07-06
thin-executor cutover (ISSUE-018 et al.) un-vacuumed the `contract_signature`
dimension and, together with a whole-repo re-scan under the packs-only spine,
exposed a cluster of gate defects: checks that were silently passing without
really checking, plus scope bugs producing false positives against a clean tree.
This directive owns fixing that cluster and draining the drift the un-vacuuming
surfaced — it is gate-hardening work, not new checker capability.

**Delivered (3):**
- ISSUE-036 — the contracts pack's signature compiler
  (`compile-signature.sh`) was `func`-only; every `type`/`const`/`method`/
  `variable`/`interface` contract silently compiled into a vacuous always-pass
  rule. Made the compiler kind-aware, which for the first time makes
  `contract_signature` actually verify non-function contracts.
- ISSUE-034 — the `coverage_threshold` step treated git-**deleted** files as
  in-scope for coverage measurement, producing spurious
  `coverage_unmeasured` violations. Fixed with an `os.Stat` existence guard.
- ISSUE-035 — `test_substantiveness` false-flagged `TestMain` and legitimate
  structural/absence-style tests as hollow. Fixed via an opt-in
  `kind: absence` claim annotation (default-off — no existing check is
  blinded by the fix).

**Remaining (5):**
- ISSUE-037 — the kind-aware compiler from ISSUE-036 still can't express
  `const` contracts whose value is an `iota` member; those remain
  unverifiable by design until this lands.
- ISSUE-038 — audit and reconcile the pre-existing contract drift that the
  kind-aware compiler (ISSUE-036) exposed now that `contract_signature` runs
  with `baseline: true` for real. This is the ratchet-down backlog: specs
  whose contracts had silently diverged from the implementation while the
  check was vacuous.
- ISSUE-039 — a genuine hollow test: `TestGate_SucceedsWithoutStandards`
  (the mandated test for SPEC-030 claim CLM-015) lost its assertion during
  the cutover and now passes vacuously.
- ISSUE-040 — the dogfood substantiveness scan walks `testdata/` fixture
  directories as ordinary source, flagging intentional fixture violations
  that exist to test the rule engine itself.
- ISSUE-041 — the `enforcement.policy.<dimension>.baseline` key in
  `backstop.yml` is misnamed: every dimension is always compared against the
  baseline, so the boolean actually controls scope (new-code only vs.
  all-code), not whether baseline comparison happens. Rename to
  `applies-to: new-code | all-code`.

## Acceptance Criteria

- The gate's quantitative dimensions — `contract_signature`,
  `coverage_threshold`, `test_substantiveness` — are non-vacuous (they can
  and do fail on a real violation) AND free of the known false-positive
  classes (deleted files, TestMain/absence tests, testdata fixtures).
- The contract-drift ratchet backlog opened by ISSUE-038 is drained: no
  outstanding drift between specs' declared contracts and the implementation
  they claim to describe.
- The `enforcement.policy.<dimension>` key is renamed from `baseline` to
  `applies-to` (`new-code | all-code`) across schema, code, and config.

## Notes

Distinct from the older `done` directives DIR-006 (Fix Substantiveness
Checker) and DIR-007 (Fix Contract Verifier) — those directives completed
their *original* scope (the 2026-04 false-positive waves) and are correctly
closed. This directive does not reopen them; it owns the *new* defects that
the 2026-07-06 packs-only cutover and kind-aware compiler exposed. Genuinely
new work gets a new directive rather than resurrecting a done one, per
standing practice.
