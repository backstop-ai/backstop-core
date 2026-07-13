---
title: "Scope Test Verification Substantiveness Coverage To Implemented"
schema_version: issue/v1

issue:
  id: ISSUE-054
  title: "Scope Test Verification Substantiveness Coverage To Implemented"
  type: technical-debt
  status: closed
  created: "2026-07-13"
  closed: "2026-07-13"

delivered_by: PLAN-ISSUE-054

complexity:
  scope: cross-cutting
  uncertainty: known
  risk: moderate
---

# ISSUE-054: Scope Test Verification Substantiveness Coverage To Implemented

## Problem

`contract_signature` was just fixed (ISSUE-051) to enforce contracts only
from `implemented` specs, because `draft` / `ready-for-implementation` specs
describe planned-but-unbuilt code, not broken promises. The **same
premature-enforcement bug** exists in three sibling gate dimensions that
still enforce on non-terminal (including `draft`) specs — `isTerminalSpecStatus`
excludes only `replaced` / `canceled` / `deprecated`, saying nothing about
pre-implementation statuses:

- **`test_verification`** (mandated-test-presence). Verified directly: of
  its 342 findings under `backstop gate --all`, **316 are on `draft` specs,
  24 on `ready-for-implementation`, and only 2 on `implemented` specs**
  (both `SPEC-041`). So **340 of 342 are false pressure** — mandated tests
  demanded for code that isn't written yet. This is the bulk of what's been
  carried as the "342 broken-promise backlog" (ISSUE-012); it was never 342
  real broken promises.
- **`test_substantiveness`** (hollow-test detection). The draft-spec
  mandated tests it checks produce false "does not call package X" findings
  — e.g. 9 surfaced when the ISSUE-050 ratchet pulled draft `SPEC-010`'s
  tests into scope.
- **`coverage_threshold`** (per-file coverage). Same premature enforcement
  applied to draft-spec coverage thresholds.

**Out of scope, explicitly not fixed here.** The 2 genuinely-missing tests
on `SPEC-041` (`implemented`) are real and remain a legitimate — but tiny —
residual backlog item. This issue does not drain them; do not close them as
part of this work.

## Architecture — why this is not a one-line copy of ISSUE-051

There are three extraction functions in `pkg/gate/step_testverify.go`, each
feeding different dimensions, and they have **different status needs**:

1. **`ExtractContractEntries`** (`pkg/gate/step_testverify.go:521`) →
   `contract_signature`. Already scoped to `implemented` by ISSUE-051 via the
   `contractsAreDue(status)` predicate (`step_testverify.go:180`, currently
   `status == "implemented"`). Reuse this predicate — do not reinvent it.
2. **`ExtractSpecVerifications`** (`pkg/gate/step_testverify.go:477`, called
   from `cmd/backstop/gate.go:1152` in `buildCoverageStep`) →
   `coverage_threshold` **only**. Not shared with anything else. It CAN be
   scoped directly to implemented-only exactly like `ExtractContractEntries`
   — e.g. `if !contractsAreDue(fm.Status) { continue }` alongside its
   existing terminal-status skip.
3. **`ExtractMandatedTests`** (`pkg/gate/step_testverify.go:106`) feeds
   **three** consumers, not one:
   - `StepTestVerificationScopedFunc` (`pkg/gate/step_testverify.go:311`) →
     `test_verification`
   - `buildTestSubstantivenessStep` (`cmd/backstop/gate.go:974`) →
     `test_substantiveness`
   - `ResolveArtifactStatus` (`pkg/gate/artifact_status.go:160`) →
     `artifact_status_drift`

**The constraint that makes this issue non-trivial:** `artifact_status_drift`
**requires** draft specs — it is the dimension that raises the "this spec is
`draft` (non-terminal) but all its mandated tests are PRESENT — it looks
delivered" advisory, which is only detectable if it can see draft specs'
mandated tests. Therefore `ExtractMandatedTests` itself must **NOT** be
scoped to implemented-only — doing so would blind `artifact_status_drift`
and silently break its own advisory.

**Design lean (final mechanism left to the plan).** The implemented-only
filter for `test_verification` and `test_substantiveness` must be applied at
the **consumer/step level**, not inside the shared `ExtractMandatedTests`.
The `MandatedTest` struct (`pkg/gate/step_testverify.go:16`) already carries
`SpecID` / `SpecFile` but not status — so either (a) add a `Status` field
populated by `ExtractMandatedTests`, and have the `test_verification` and
`test_substantiveness` consumers filter with `contractsAreDue(status)`
before acting on a mandated test, or (b) introduce an
enforcement-scoped variant of the extractor/consumer path that wraps the
shared one. `ResolveArtifactStatus` (`artifact_status_drift`) must keep
consuming the **unfiltered** list unconditionally. The plan must
demonstrate — with a test — that `artifact_status_drift`'s
draft-looks-delivered advisory still fires correctly after the change.

## Blast radius (plan must trace exhaustively, not sample)

ISSUE-051's plan review flagged blast-radius under-counting as its #1 risk;
this issue's radius is roughly 2x larger because it spans three consumers
(test_verification, test_substantiveness, coverage_threshold) instead of
one, plus the status_drift fixtures that must be proven UNCHANGED. Known
affected test files staging `draft` / statusless specs with a test command
or mandated tests, which may flip once extraction/consumption scopes to
implemented:

- `pkg/gate/step_testverify_test.go`
- `pkg/gate/step_coverage_test.go`
- `pkg/gate/step_testverify_metric_test.go`
- `pkg/gate/coverage_permetric_verdict_test.go`
- `pkg/gate/step_coverage_eradication_test.go`
- multiple `cmd/backstop/*_test.go` files exercising the gate command

The plan MUST grep every caller of `ExtractMandatedTests`,
`ExtractSpecVerifications`, and `ExtractContractEntries` (production and
test code) and every fixture that stages a spec consumed by the three
sibling dimensions, and reconcile each — typically by flipping the fixture's
`status:` to `implemented`, or by asserting the new draft-exclusion
behavior directly where that's the point of the test. `artifact_status_drift`
fixtures are the one category that must be verified to need **no** change.

## Anti-gaming / safety

Identical composition argument to ISSUE-051: scoping `test_verification` /
`test_substantiveness` / `coverage_threshold` to `implemented` specs is not
a way to dodge enforcement by parking a built spec in `draft` forever —
`artifact_status_drift` independently keeps status honest (a spec whose code
is actually built and tested cannot legitimately linger `draft` without
raising the drift advisory). The dimensions compose: `artifact_status_drift`
keeps status honest, the three sibling dimensions then enforce mandated
tests / substantiveness / coverage only on specs honestly `implemented`.

## Expected result

- `test_verification`: 342 → 2 (the genuine `SPEC-041` residual, left open).
- `test_substantiveness`: draft-spec false "does not call package X"
  findings drop (e.g. the 9 on `SPEC-010`).
- `coverage_threshold`: draft-spec coverage-threshold false positives drop.
- `artifact_status_drift`: unchanged — draft-looks-delivered advisory still
  fires, proven by a test.
- `backstop gate`: `new_violations: 0`; baseline refreshed to the corrected
  set.
- Largely (not fully — the `SPEC-041` residual remains) resolves ISSUE-012.

## References

- `pkg/gate/step_testverify.go:16` — `MandatedTest` struct; carries
  `SpecID`/`SpecFile` but no status field today
- `pkg/gate/step_testverify.go:106` — `ExtractMandatedTests`; must stay
  unfiltered by implementation status (feeds `artifact_status_drift`)
- `pkg/gate/step_testverify.go:180` — `contractsAreDue`, the ISSUE-051
  predicate (`status == "implemented"`) this issue reuses rather than
  reinventing
- `pkg/gate/step_testverify.go:311` — `StepTestVerificationScopedFunc`,
  the `test_verification` consumer that needs the implemented-only filter
- `pkg/gate/step_testverify.go:477` — `ExtractSpecVerifications`, the sole
  `coverage_threshold` feed; scope directly like `ExtractContractEntries`
- `pkg/gate/step_testverify.go:521` — `ExtractContractEntries`, the
  ISSUE-051 sibling already fixed; the reference implementation
- `pkg/gate/artifact_status.go:160` — `ResolveArtifactStatus`'s
  `ExtractMandatedTests` call; the consumer that MUST keep seeing draft specs
- `cmd/backstop/gate.go:974` — `buildTestSubstantivenessStep`, the
  `test_substantiveness` consumer that needs the implemented-only filter
- `cmd/backstop/gate.go:1152` — `buildCoverageStep`, calls
  `ExtractSpecVerifications`
- ISSUE-051 — the `contract_signature` sibling fix; source of
  `contractsAreDue` and the anti-gaming composition argument reused here
- ISSUE-012 — the 342 mandated-test broken-promise backlog this issue
  largely (not fully) drains
- ISSUE-050 — the strict file-level ratchet whose scope-pull surfaced the
  `test_substantiveness` false positives on draft `SPEC-010`
- SPEC-041 — the 2 genuine residual `test_verification` findings, explicitly
  out of scope for this issue
- `directives/DIR-015-gate-checker-hardening.directive.md` — this issue is
  part of DIR-015's gate-correctness cluster
- Project memory `artifact_status_drift` dimension — the anti-gaming guard
  this issue's scoping composes with, and the reason `ExtractMandatedTests`
  itself cannot be filtered
- CLAUDE.md — "Loud ≠ blocking" enforcement philosophy; this issue removes
  false loudness (draft-spec pressure), not real enforcement

## Resolution

Delivered by PLAN-ISSUE-054 (commit 2164994). Scoped `test_verification`,
`test_substantiveness`, and `coverage_threshold` to `implemented` specs.
`ExtractSpecVerifications` (coverage, unshared) filters directly; for the shared
`ExtractMandatedTests` — also consumed by `artifact_status_drift`, which REQUIRES
draft-spec visibility — a `MandatedTest.Status` field was added and the
`implemented`-only filter applied at the test_verification + substantiveness
CONSUMERS, leaving the shared extractor and status_drift untouched.
`test_verification` dropped 342 → 2 (the 2 real broken promises on SPEC-041);
draft-spec substantiveness/coverage FPs cleared; `artifact_status_drift` unchanged
(9 → 9), proving no leak into the shared path. Largely drains the ISSUE-012
backlog. Guard test `TestStatusDriftAdvisory_DraftLooksDeliveredStillFires`
exercises the real `ResolveArtifactStatus → ExtractMandatedTests` path over a temp
projectRoot.
