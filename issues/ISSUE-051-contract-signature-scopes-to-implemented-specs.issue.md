---
title: "Contract Signature Scopes To Implemented Specs"
schema_version: issue/v1

issue:
  id: ISSUE-051
  title: "Contract Signature Scopes To Implemented Specs"
  type: technical-debt
  status: open
  created: "2026-07-13"

complexity:
  scope: contained
  uncertainty: known
  risk: moderate
---

# ISSUE-051: Contract Signature Scopes To Implemented Specs

## Problem

`contract_signature` is diff-scoped in normal gate runs (`applies-to:
new-code`, `backstop.yml:15-17`), so day to day it only evaluates contracts
declared by specs whose files are actually touched. Running the gate with
`--all` — full-repo scan, ignoring diff scope — surfaces **143 violations**,
all of them currently silent because they fall under the existing baseline
and ISSUE-050's strict file-level ratchet (a violation on an untouched file
is inherited debt, not a new-work block).

Splitting the 143 by whether the contract's declared `file` actually exists
on disk:

- **104 point at files that don't exist.**
- **39 point at files that exist.**

### Verified: the 104 are planned-not-yet-built work, not drift

All 104 missing-file contracts are declared by the init/release spec suite
**SPEC-020 through SPEC-029** (init-state detection, default-pack discovery,
consentful dependency/toolchain install, non-destructive repair, greenfield
scaffolding, CI setup, onboarding orchestration, binary distribution), every
one a child of BUNDLE-003. Their statuses, checked directly:

| Spec | Status |
|---|---|
| SPEC-020 | `ready-for-implementation` |
| SPEC-021 | `draft` |
| SPEC-022 | `ready-for-implementation` |
| SPEC-023 | `draft` |
| SPEC-024 | `draft` |
| SPEC-025 | `draft` |
| SPEC-026 | `draft` |
| SPEC-027 | `draft` |
| SPEC-028 | `draft` |
| SPEC-029 | `draft` |

Per the spec schema's lifecycle vocabulary (`artifacts/spec/v1/schema.json`),
LIVE statuses run `draft` → `ready-for-implementation` → `implemented`;
TERMINAL statuses are `replaced` / `canceled` / `deprecated` / `obsoleted`.
None of SPEC-020..029 is `implemented`, and none is terminal either — they
are simply **not built yet**. Confirmed directly: `pkg/init/*`,
`pkg/release/distribution/*`, `pkg/initrepair/*`, `pkg/initstate/*`, and
`cmd/backstop/init.go` do not exist in the tree today.

### Root cause

`ExtractContractEntries` (`pkg/gate/step_testverify.go:509`) extracts
contract declarations from **every spec that is not terminal**:

```go
if isTerminalSpecStatus(fm.Status) {
    continue // terminal specs are excluded from enforcement (ISSUE-031)
}
```

`isTerminalSpecStatus` (`pkg/gate/step_testverify.go:166`) returns `true`
only for `replaced` / `canceled` / `deprecated` — it says nothing about
`draft` or `ready-for-implementation`. So a `draft` spec's contracts get
extracted and handed to the Go signature compiler exactly like an
`implemented` spec's would, and the compiler correctly reports "symbol not
found" for each — because the symbol was never supposed to exist yet.

This is the same "the gate is asking the wrong question about a
non-terminal-but-not-live status" family as ISSUE-036/037 (which fixed the
compiler itself), but **inverted**: instead of a check silently passing when
it shouldn't (vacuous green), this is a check silently **failing** when it
shouldn't — false pressure that makes non-debt (planned, unbuilt work) read
as grandfathered debt, polluting the baseline and the ratchet with entries
that were never real violations.

## Solution

Not committed — left open for the plan. Clear lean: contracts are only
"due" once a spec reaches `implemented`; a `draft` or
`ready-for-implementation` spec's contracts describe intended future code,
not a broken promise. `ExtractContractEntries` should extract contracts only
from specs whose status is `implemented`, skipping pre-implementation
statuses (`draft`, `ready-for-implementation`) in addition to the existing
terminal skip. Mechanically this is a single predicate change — either a new
`isPreImplementationSpecStatus` helper alongside `isTerminalSpecStatus`, or
inverting the check to a positive `contractsAreDue(status)` that requires
`implemented` — mirroring the shape `isTerminalSpecStatus` already
establishes as the single source of truth pattern for this kind of
status-gated exclusion.

A mandated test should prove the fix directly: a `draft`-status spec
declaring a contract on a nonexistent symbol must NOT be extracted by
`ExtractContractEntries`, while an `implemented`-status spec's contract
still is.

**Why this is safe / not a vacuous-green hole.** Scoping contract
enforcement to `implemented` specs does not open a way to dodge it by
parking a spec in `draft` forever — the `artifact_status_drift` gate
dimension independently enforces honest status, so a spec whose code is
actually built cannot legitimately linger in `draft` or
`ready-for-implementation`. The two dimensions compose: `artifact_status_drift`
keeps status honest, `contract_signature` then enforces contracts only on
specs that are honestly `implemented`.

**Expected result.** 104 of the 143 evaporate correctly — not by retiring a
single real contract, but by correctly not-yet-enforcing planned work whose
code doesn't exist yet. The residual ~39 is the genuine worklist: roughly 30
real Go drift on already-`implemented` specs (tracked as ISSUE-038) and
roughly 7 non-Go-artifact contracts that can never compile under the current
Go-oriented signature compiler regardless of spec status (tracked as
ISSUE-053).

**Sibling observation — flagged here, not fixed here.**
`ExtractSpecVerifications` (`pkg/gate/step_testverify.go:465`), which feeds
`test_verification` and `coverage_threshold`, has the identical
terminal-only filter (same `isTerminalSpecStatus` call at line ~484), so it
likewise pulls mandated-tests/coverage expectations from unbuilt `draft`
specs. This is very likely a chunk of the already-parked, judgment-heavy
`test_verification` broken-promise backlog — ISSUE-012 (58 mandated-but-never-
written SPEC-017/SPEC-031 tests concealed by diff scope) is a documented
instance of exactly this family: a gate dimension pulling enforcement from
specs/claims whose code was never built. It is named here as a candidate
sibling fix using the same predicate, and explicitly **deferred** — the
founder has parked that backlog and this issue's scope is
`contract_signature` only.

## References

- `pkg/gate/step_testverify.go:509` — `ExtractContractEntries`, the function
  this issue's fix targets
- `pkg/gate/step_testverify.go:166` — `isTerminalSpecStatus`, the single
  source of truth this fix extends with a pre-implementation predicate
- `pkg/gate/step_testverify.go:465` — `ExtractSpecVerifications`, the
  sibling function with the identical filter; named and deferred, not fixed
  here
- `backstop.yml:15-17` — `contract_signature: { applies-to: new-code, level:
  block }`, the ratchet policy under which the 143 currently sit silent
- `artifacts/spec/v1/schema.json` — spec status lifecycle vocabulary
  (`draft` → `ready-for-implementation` → `implemented`; terminal:
  `replaced`/`canceled`/`deprecated`/`obsoleted`)
- `specs/SPEC-020-*.spec.md` through `specs/SPEC-029-*.spec.md` — the
  init/release spec suite responsible for all 104 missing-file contracts,
  all children of BUNDLE-003
- `bundles/BUNDLE-003-onboarding-experience.bundle.md` — parent bundle for
  the still-unbuilt init/release suite
- ISSUE-036 — the kind-aware compiler fix that made `contract_signature`
  non-vacuous and, in doing so, made this scoping gap observable for the
  first time
- ISSUE-038 — the residual real-drift worklist (~30 of the 39) this fix
  leaves behind once the 104 false positives are removed
- ISSUE-053 — the residual non-Go-artifact worklist (~7 of the 39) this fix
  leaves behind, orthogonal to spec status
- ISSUE-031 — the terminal-exclusion precedent (`isTerminalSpecStatus`) this
  issue extends with a pre-implementation exclusion
- ISSUE-012 — a documented instance of the parked `test_verification`
  broken-promise backlog family the sibling observation above is likely a
  chunk of (58 mandated-but-never-written SPEC-017/SPEC-031 tests concealed
  by diff scope)
- `directives/DIR-015-gate-checker-hardening.directive.md` — this issue is
  part of DIR-015's gate-correctness cluster
- Project memory `artifact_status_drift` dimension — the anti-gaming guard
  that makes scoping contracts to `implemented` status safe
- CLAUDE.md — "Loud ≠ blocking" enforcement philosophy; this is the false
  positive form of what that principle guards against
