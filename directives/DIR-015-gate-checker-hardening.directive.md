---
title: "Gate Checker Hardening"
number: DIR-015
created: "2026-07-06"
schema_version: directive/v1

directive:
  status: done
  source:
    - "ISSUE-036"
    - "ISSUE-034"
    - "ISSUE-035"
    - "ISSUE-037"
    - "ISSUE-038"
    - "ISSUE-039"
    - "ISSUE-040"
    - "ISSUE-041"
    - "ISSUE-033"
---

## Description

Make backstop's own dogfood gate checks CORRECT and NON-VACUOUS. The 2026-07-06
thin-executor cutover (ISSUE-018 et al.) un-vacuumed the `contract_signature`
dimension and, together with a whole-repo re-scan under the packs-only spine,
exposed a cluster of gate defects: checks that were silently passing without
really checking, plus scope bugs producing false positives against a clean tree.
This directive owns fixing that cluster and draining the drift the un-vacuuming
surfaced — it is gate-hardening work, not new checker capability.

**DELIVERED — all nine source issues terminal (2026-07-13).**

Bug-fix cluster:
- ISSUE-036 — made the contracts `compile-signature.sh` kind-aware (was
  `func`-only; every non-func contract compiled to a vacuous always-pass).
- ISSUE-034 — `coverage_threshold` no longer measures git-deleted files
  (`os.Stat` existence guard).
- ISSUE-035 — `test_substantiveness` stopped false-flagging `TestMain` /
  structural tests via an opt-in `kind: absence` claim annotation.
- ISSUE-039 — restored the lost assertion in `TestGate_SucceedsWithoutStandards`
  (SPEC-030 CLM-015) that had gone vacuous.
- ISSUE-040 — stopped the dogfood substantiveness scan walking `testdata/`
  fixtures as source.
- ISSUE-041 — renamed the misnamed `enforcement.policy.<dimension>.baseline`
  key to `applies-to: new-code | all-code`.
- ISSUE-033 — de-Go'd `plan.go`'s baked `.go` file-classification.

Contract-correctness (037/038) — reconciled and handed off:
- ISSUE-038 — the contract-drift ratchet backlog. The exhaustive `--all` sweep
  drained all GENUINE drift (repointed `SourceClassifier`, shared-runner absence
  guards; retired `loadBridgedToolchainPacks` + stale `newSharedTestRunner`;
  fixed the `funcPattern`/`stackLabel` comment-match FPs). **Status `replaced`
  by ISSUE-052** — the 5 residual findings proved NOT to be drift but structural
  blind spots of the compiler (3 struct-field contracts it can't express, 2 non-Go
  prose contracts on testdata), re-scoped to ISSUE-052 (struct-field/iota
  capability) and ISSUE-053 (non-Go).
- ISSUE-037 — the iota-member gap. Found to have 0 live instances while its
  identical-root sibling (struct fields) has 3 live ones; **status `replaced`
  by ISSUE-052**, which now carries the full structural-presence capability
  (iota + struct fields + the fail-loud guard), justified by real instances.

Emergent gate-correctness fixes (not original sources; surfaced by the ISSUE-038
investigation and delivered under this theme) — ISSUE-051/ISSUE-054 scoped
`contract_signature` / `test_verification` / `test_substantiveness` /
`coverage_threshold` to `implemented` specs, correcting the gate's enforcement of
contracts and mandated-tests on planned-but-unbuilt draft specs
(contract_signature 143→9, test_verification 342→2 — the residual are genuine).

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

ISSUE-033 (de-Go `plan.go`'s baked `.go` file-classification via
pack-declared globs) re-homed here from DIR-014 (2026-07-06) — the
thin-executor residual whose literal is currently suppressed; DIR-015 owns
the real fix.
