---
title: "Drift Resolver Blind To Pack Claim Ids"
schema_version: issue/v1

issue:
  id: ISSUE-098
  title: "Drift Resolver Blind To Pack Claim Ids"
  type: bug
  status: open
  created: "2026-07-28"

complexity:
  scope: contained
  uncertainty: known
  risk: safe
---

# ISSUE-098: Drift Resolver Blind To Pack Claim Ids

## Problem

The `artifact_status_drift` gate dimension's mandated-test vocabulary is "test function
declared in a source file" — full stop. When a claim's `tests:` entry names a **pack claim
id** instead of a Go test function (the pack-native evidence unit — a `claims[].id` under a
pack.yml rule, verified by `backstop pack test`'s fixture pipeline, not by `go test`), the
resolver can never mark it present, no matter how correct and passing the pack claim actually
is. That produces a false broken-promise block on entirely healthy, packs-only evidence — the
opposite of what the dimension exists to catch.

### First CI-blocking instance (2026-07-28)

ISSUE-036's CLM-008 mandates five pack claim ids —
`type-signature-go`, `const-signature-go`, `var-signature-go`, `method-signature-go`,
`interface-signature-go` — all five present, unchanged, and passing under their original
names at `packs/contracts/pack.yml:70-129` in all three copies (durable source, tracked
test-harness copy, installed copy); `backstop pack test` confirms phase-3 fixtures pass for
every one of them. None of that matters to the resolver: it reports all five ABSENT anyway,
contributing 5 of the 7 pre-existing `artifact_status_drift` violations catalogued by
ISSUE-048. This had been sitting locally baseline-grandfathered (pass severity,
non-blocking) — until CI, which does not inherit the local baseline cache
(see ISSUE-086, "the published baseline artifact is generated with zero packs installed" —
a related but distinct gap in the same neighborhood: CI's baseline generation and CI's gate
enforcement are not seeing the same evidence a local run sees), turned this from a quiet
grandfathered warning into a hard CI failure: run `30384984403`
(`fix/issue-020-sandbox-diagnostic`, "Run the gate" step, job `90361904638`) is RED on this
exact class of false positive. This was first surfaced as discovery context during
PLAN-ISSUE-020's TASK-019 (Linux-runner sandbox gate validation) — the CI run that finally
exercised the gate on a runner without a pre-warmed local baseline is what made a
year-old-in-spirit blind spot visible for the first time.

### Root cause

The resolver's whole-repo existence check runs through one path with no branch for pack-side
evidence:

- `cmd/backstop/gate.go:942` `computeDriftSurfaces` collects every record's
  `MandatedTests` and calls `gate.ResolveMandatedTestPaths` over the **whole repo** (not
  scope-limited, by design — CLM-005/008 of the dimension itself: an out-of-diff stale
  artifact must still be caught).
- `pkg/gate/step_testverify.go:609` `ResolveMandatedTestPaths` resolves each mandated test's
  `FuncName` against a map built by `collectTestFuncNames`.
- `pkg/gate/step_testverify.go:453` `collectTestFuncNames` (→ `:457`
  `collectTestFuncNamesScoped`) walks `codeDir`, keeps only files the pack-declared
  `SourceClassifier.IsTestFile` recognizes as TEST files, and within those files applies the
  pack-declared `TestNameMatcher.FindName` **per line** to harvest test-function names.

A `pack.yml` is not a file the classifier will ever call a "test file" (it is pack
configuration, not source under test), and even if it were, `- id: type-signature-go` is not
a line shape any `TestNameMatcher` built for source-language test declarations would ever
recognize as a test name. There is no code path anywhere in this chain that reads an
installed pack's manifest and checks whether it declares a `claims[].id` matching the
mandated name. The vocabulary is Go-test-function-shaped by construction; pack claim ids are
categorically outside it, not merely unmatched by it.

This is a **thin-executor gap**, not a Go-specific one: any language's pack-native evidence
unit (a pack claim id, a fixture id — whatever a given pack's `archetype` uses as its unit of
proof) hits the identical blind spot the moment a claim's `tests:` mandates it instead of a
source-level test function. ISSUE-036 is simply the first artifact to have actually done so.

### Why this reads as a per-artifact naming/retirement question but isn't

ISSUE-048 originally treated ISSUE-036's 5 violations as a candidate for the same
per-artifact "repoint or retire" triage its other 6 stragglers got, on the theory the pack's
rule ids might have been renamed when the compiler went kind-aware. That theory is
disproven (2026-07-28 investigation) — the ids were never touched. Renaming, retiring, or
"repointing" ISSUE-036's CLM-008 cannot fix this: whatever pack claim id the claim names,
the resolver still has no vocabulary to see it. The only fix that generalizes is teaching
the resolver pack-side evidence is first-class, which is this issue's scope. ISSUE-048 has
been corrected accordingly (2026-07-28) — its own text no longer frames this as a
naming/retirement judgment call.

### Interim CLM-008 decision, and its restoration trigger

Evaluated whether a subset of the 9 Go tests in
`pkg/pack/engine/contracts_kind_signature_test.go` (the `TestContractCompiler_*` suite,
shipped alongside CLM-008 in the same commit `d5efd5b`) could honestly stand in for CLM-008's
mandated pack claim ids as an interim Go-test repoint, the way CLM-007 above was repointed.
**Decision: no repoint.** Those 9 tests shell `compile-signature.sh` directly with
hand-picked signature strings and assert on the *compiler's* dynamically emitted pattern —
they substantiate CLM-001 through CLM-007 (the compiler is declaration-kind-aware), which is
a different claim already carrying its own dedicated tests. None of them invoke
`backstop pack test`, read `pack.yml`, or verify that the five **static** per-kind rules
declared there (`type-signature`, `const-signature`, `var-signature`, `method-signature`,
`interface-signature`) pass their own positive/negative fixtures — which is what CLM-008
actually claims: that the pack's own self-test suite, not the compiler in isolation, exercises
every non-func kind. Repointing CLM-008 to the compiler tests would substitute "the compiler
is correct" for "the pack's own frozen regression suite is correct" — a different, weaker
guarantee dressed up as the same claim, and exactly the kind of claim-weakening-to-clear-a-gate
CLAUDE.md's enforcement philosophy exists to prevent. ISSUE-036 was therefore left untouched.
**Restoration trigger:** once this issue's resolver fix ships, CLM-008 needs no repoint at
all — its existing `tests:` (the five pack claim ids) will resolve as present on their own
merits, which is the honest outcome this issue exists to deliver.

## Solution

Teach `ResolveMandatedTestPaths` (or a sibling resolution path it delegates to) to recognize a
mandated test name that matches an installed pack's declared `claims[].id` (walking installed
pack manifests — `backstop.lock` + `.backstop/packs/*/pack.yml` — the same set `pack test`
already reads) as PRESENT when that pack's own validation (fixtures) confirms the claim is
live, rather than only ever searching for it as a Go (or any source-language) test function.
Packs-only grain: a pack's fixtures ARE that pack's tests: this is not a special case, it is
the resolver learning the second half of the vocabulary that already exists on the pack side.

Suggested shape (open to the planner):

1. Build a pack-claim-id index once per gate run from installed pack manifests (name → set of
   `claims[].id` per rule), alongside the existing Go test-function index
   `collectTestFuncNames` already builds.
2. In `ResolveMandatedTestPaths`, check a mandated test's `FuncName` against BOTH indices —
   source-function match (existing behavior, unchanged) OR pack-claim-id match (new) — before
   concluding ABSENT.
3. Decide, explicitly, whether a pack-claim-id match should also require confirmation that the
   pack's own fixture for that claim passes (i.e., cross-check against a `pack test` result),
   or whether claim-id PRESENCE in an installed, validated pack manifest is sufficient
   evidence on its own (the manifest only validates/installs when its fixtures already pass —
   `pkg/packval` phase 3 — so a second live fixture re-run at drift-resolution time may be
   redundant work, not additional rigor; the planner should confirm this rather than assume
   either way).
4. Re-run the gate and confirm ISSUE-036's 5 violations clear without any change to
   `packs/contracts/pack.yml` or ISSUE-036's own artifact — proving the fix is in the resolver,
   not a repoint dressed up as one.

## References

- `cmd/backstop/gate.go:942` — `computeDriftSurfaces`, the whole-repo (non-scope-limited)
  existence-resolution entry point
- `pkg/gate/step_testverify.go:609` — `ResolveMandatedTestPaths`, the function with no
  pack-claim-id branch
- `pkg/gate/step_testverify.go:453,457` — `collectTestFuncNames` /
  `collectTestFuncNamesScoped`, the Go-test-function-only harvest (pack-declared TEST-file
  globs + pack-declared `TestNameMatcher`, but categorically source-test-shaped, never
  pack-manifest-shaped)
- `packs/contracts/pack.yml:70-129` — the five live, passing, unrenamed claim ids
  (`type-signature-go`, `const-signature-go`, `var-signature-go`, `method-signature-go`,
  `interface-signature-go`) that prove the pack side is healthy and the gap is purely in the
  resolver
- `pkg/pack/engine/contracts_kind_signature_test.go` — the 9 `TestContractCompiler_*` tests
  evaluated (and rejected) as a CLM-008 repoint candidate; they substantiate CLM-001–CLM-007,
  not CLM-008
- ISSUE-036 (`issues/ISSUE-036-contracts-pack-compiler-func-only-signatures.issue.md`, closed)
  — CLM-008's 5 mandated pack claim ids, left untouched pending this issue
- ISSUE-048 (`issues/ISSUE-048-reconcile-stranded-terminal-lineage.issue.md`, open) — corrected
  2026-07-28 to stop framing ISSUE-036's violations as a naming/retirement judgment call; now
  points here for the actual fix
- ISSUE-086 (`issues/ISSUE-086-published-baseline-generated-packless.issue.md`, open) — the
  adjacent local-baseline-vs-CI gap that let this false positive stay a quiet local warning
  until CI (which does not share the local baseline cache) turned it into a hard block
- PLAN-ISSUE-020 (`plans/PLAN-ISSUE-020-linux-sandbox-gate-in-ci.plan.yml`), TASK-019 — the
  Linux-runner sandbox gate validation whose real CI run first made this blind spot visible
- CI run `30384984403` (`fix/issue-020-sandbox-diagnostic`, job `90361904638`, "Run the gate"
  step) — first CI-blocking instance, confirmed RED via `gh run view` 2026-07-28
- `pkg/packval/` — the phase-3 fixture pipeline that already validates a pack's claim ids are
  live; the natural source of truth this issue's fix should read from rather than
  re-implementing
- CLAUDE.md — thin-executor first principle (this is a gap in resolver *vocabulary*, not a
  Go-specific bug: any pack-native evidence unit hits it) and "Loud ≠ blocking" enforcement
  philosophy (a false ABSENT on healthy evidence is exactly the silent-rot-adjacent failure
  mode the philosophy targets, inverted — here it's noisy-false-positive, not silent-gap, but
  the same "don't let mechanism cost mask truth" principle applies)
