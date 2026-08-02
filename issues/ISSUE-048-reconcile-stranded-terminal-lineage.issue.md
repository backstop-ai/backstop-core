---
title: "Reconcile Stranded Terminal Lineage — ISSUE-018 / ISSUE-036 Residual"
schema_version: issue/v1

issue:
  id: ISSUE-048
  title: "Reconcile Stranded Terminal Lineage — ISSUE-018 / ISSUE-036 Residual"
  type: technical-debt
  status: closed
  created: "2026-07-08"
  closed: "2026-08-02"

resolved-by: 3a7a700

complexity:
  scope: contained
  uncertainty: known
  risk: moderate
---

# Reconcile Stranded Terminal Lineage — ISSUE-018 / ISSUE-036 Residual

## Resolution

All of this issue's scope — the four original problems and the two residual
Problem-1 artifacts (ISSUE-018, ISSUE-036) the 2026-07-28 Scope update
narrowed it to — is now delivered.

**Problems 2/3 (terminal vocab + plan-task-test drift coverage), the
keystone this issue's own scope narrowing depended on:** commit `3a7a700`
("feat(ISSUE-048): obsoleted/resolved-by close vocab + plan-task-test drift
coverage") landed the `obsoleted` terminal state and typed `obsoleted-by`
pointer in `pkg/validate/terminal.go` — `isTerminalStatus` includes
`"obsoleted"` in its terminal set at `:25`, `validateRetirementFields`
requires a typed `obsoleted-by` via `validateTypedRetirementRef` at
`:57-59`, and `extractObsoletedBy` reads the pointer at `:113`. All three
mandated tests are present and pass: `TestClassifyArtifactStatus_ObsoletedIsRetiredTerminal`
and `TestStatusDrift_ExcludesObsoletedArtifact`
(`pkg/gate/artifact_status_obsoleted_test.go`), and
`TestStatusDrift_CompletedPlanAbsentTaskTest_Blocks`
(`pkg/gate/status_drift_plantests_test.go`) — re-run 2026-08-02, all green.

**ISSUE-018 residual (CLM-007's 2 mandated tests):** resolved by repointing
CLM-007 in-tree to the real, present, passing guard tests
`TestCutover_NoCodeCheckStepInGateStepList` and
`TestCutover_LintBuildTestRunThroughDispatchPackEngines`
(`cmd/backstop/gate_cutover_step2_test.go:49,61`) — confirmed present at
those exact lines. See ISSUE-018's own Resolution section for the
mis-transcription history.

**ISSUE-036 residual (CLM-008's 5 mandated pack claim ids):** the resolver
gap this issue diagnosed (the drift resolver has no vocabulary for a
pack-declared claim id, only Go test functions) was handed off to and
delivered by ISSUE-098 (`closed`) — `PackClaimIndex` at
`pkg/gate/pack_claims.go:32`, wired into the drift step at
`cmd/backstop/gate.go:731`. Confirmed both land at the cited lines.
ISSUE-036's own artifact was correctly left untouched, per the interim
decision recorded in ISSUE-098: its five mandated claim ids resolve present
on their own merits now that the resolver understands pack-side evidence,
with no repoint needed.

**Honest caveat:** the drift resolver's fix (ISSUE-098) was verified
structurally here — code present at the cited lines, its own mandated tests
passing — but a live `./bin/backstop gate` run confirming the
`artifact_status_drift` violation count is actually 0 across the whole repo
was **not** executed as part of this closure. Stating that plainly rather
than implying a measured result.

**Record-keeping drift, left as-is:**
`plans/PLAN-ISSUE-048-obsoleted-resolvedby-vocab.plan.yml` is still
`status: draft` even though the code it plans shipped in the very same
commit (`3a7a700`) that authored the plan file. That plan file needs
reconciling to `completed` (or otherwise) but is out of scope for this
closure and was not edited here.

## Problem

This issue originally named four problems surfaced by ISSUE-042's
`artifact_status_drift` gate dimension. Three of the four are now fully
delivered (see "Delivered elsewhere" below) and the fourth (SPEC-002) has
been mooted by a broader, unrelated retirement. What remains — and what this
issue now scopes to — is the tail of Problem 1: two closed issues whose
mandated tests are still genuinely ABSENT, still flagged today by
`artifact_status_drift`, and never reconciled artifact-by-artifact the way
the other 6 of the original 8 were.

**Scope update (2026-07-28):** a cross-issue investigation disproved both of
the assumptions this section originally made about *why* the tests read
ABSENT (see the corrections inline below). ISSUE-018's 2 violations were a
genuine mis-transcription and are now RESOLVED (CLM-007 repointed to the
real, present, passing tests). ISSUE-036's 5 violations are NOT a naming or
retirement question at all — the mandated pack claim ids are present and
passing; the resolver simply cannot see pack-side evidence, a structural gap
now tracked as ISSUE-098. This issue's remaining live scope is therefore
narrower than originally framed: decide what, if anything, ISSUE-036's
CLM-008 should do pending ISSUE-098 (current interim position: leave it
mandating the pack claim ids as-is and let ISSUE-098 fix the resolver,
rather than force a Go-test repoint that would misrepresent what's being
verified — see ISSUE-036's own artifact, left untouched by this
correction).

Running `./bin/backstop gate` today reports exactly 7 live
`artifact_status_drift` violations (pass severity, non-blocking — baseline
grandfathered), all against two artifacts:

- **ISSUE-018** (`issues/ISSUE-018-remove-vestigial-baked-in-code.issue.md`,
  status `closed`) — CLM-007 mandates `TestCutover_GateNeverWiresStepCodeCheck`
  and `TestCutover_GateNeverCallsCheckRun`. Both are ABSENT (2 violations).
  ~~These were ISSUE-018's own cutover-assertion tests, later removed once the
  thing they asserted the absence of (`backstop code check` / `pkg/check`)
  was fully gone.~~ **Correction (2026-07-28):** disproven. `git log --all -S`
  for either literal string returns zero commits — these two names never
  existed as code, so they cannot have been "removed." They were fabricated
  in the closing commit `ca5b7ec` by mis-transcribing the doc-comment prose
  above the real, still-present, still-passing guard tests
  (`TestCutover_NoCodeCheckStepInGateStepList`,
  `TestCutover_LintBuildTestRunThroughDispatchPackEngines`, at
  `cmd/backstop/gate_cutover_step2_test.go:49` and `:61`). ISSUE-018's own
  CLM-007 has been repointed to those real names (2026-07-28); this drift is
  RESOLVED, not open reconciliation work — see ISSUE-018's Resolution
  section.
- **ISSUE-036**
  (`issues/ISSUE-036-contracts-pack-compiler-func-only-signatures.issue.md`,
  status `closed`) — CLM-008 mandates the contracts-pack compiler rule-id
  "tests" `type-signature-go`, `const-signature-go`, `var-signature-go`,
  `method-signature-go`, `interface-signature-go`. All 5 are ABSENT
  (5 violations).

Both are pre-existing, currently non-blocking (baseline-grandfathered), and
were explicitly deferred as "judgment cases" back when the bulk of Problem 1
was reconciled (5 of the original 8 artifacts moved to `obsoleted`) — they
need the same per-artifact call the others got: restore/repoint the mandated
test to a surviving equivalent, or retire the artifact honestly (now that the
`obsoleted` + `obsoleted-by` vocab exists) rather than leaving it under
permanent baseline grandfather. This issue does not prescribe which
resolution fits which artifact — that is planning work, read case by case.

~~Note: `packs/contracts/pack.yml`'s rule ids may have been renamed rather
than deleted when the compiler went kind-aware (see
`project_contracts_pack_kind_gaps` in agent memory, ISSUE-036/037/052) — the
planner should check for a rename/repoint candidate before assuming full
removal for ISSUE-036.~~ **Correction (2026-07-28):** neither renamed nor
deleted. All five mandated claim ids (`type-signature-go`,
`const-signature-go`, `var-signature-go`, `method-signature-go`,
`interface-signature-go`) are present, unchanged, and passing under their
original names at `packs/contracts/pack.yml:70-129` in all three copies
(durable source, tracked test-harness copy, installed copy); `backstop pack
test` confirms phase-3 fixtures pass. The false ABSENT report is structural,
not a naming drift: `artifact_status_drift`'s resolver
(`ResolveMandatedTestPaths`, `pkg/gate/step_testverify.go:609`) only scans
pack-declared test-FILE globs for Go test-FUNCTION names
(`collectTestFuncNames`, `:453`) — it has no vocabulary for a pack-declared
claim id at all, so a pack-side mandated test can never resolve as present
regardless of whether the pack itself is correct. This is now tracked as its
own defect, ISSUE-098, rather than a per-artifact reconciliation judgment
call — see that issue for the resolver fix and the interim CLM-008 decision
(no honest Go-test repoint exists; the file was left untouched, see
ISSUE-098's references).

## Delivered elsewhere (historical context — not remaining scope)

The issue's other three problems, and 6 of the original 8 Problem-1
artifacts, are already fully delivered:

- **Problem 2 (terminal-vocab gap)** — DELIVERED. Commit `3a7a700`
  (`feat(ISSUE-048): obsoleted/resolved-by close vocab + plan-task-test drift
  coverage (DIR-016)`, 2026-07-08, following `PLAN-ISSUE-048-obsoleted-resolvedby-vocab.plan.yml`)
  added the `obsoleted` terminal state + typed `obsoleted-by` pointer to
  `pkg/validate/terminal.go` — exactly the vocab this issue's Problem 2 asked
  the planner to decide on, resolved as "a new terminal state" rather than a
  looser convention.
- **Problem 3 (plan task-tests drift coverage)** — DELIVERED in the same
  commit `3a7a700`: the drift resolver now parses completed-plan
  `tasks[].test_names` and applies the same success-terminal-absent check
  already applied to issues/specs.
- **Problem 1, 5 of 8 artifacts** — DELIVERED. Commit `a7070bb`
  (`chore(backlog): reconcile 10 stragglers via resolved-by + obsoleted
  (ISSUE-048 dogfood)`, 2026-07-08) moved ISSUE-002/003/005/006/008 to
  `status: obsoleted` + `obsoleted-by: ISSUE-018`, dropping the drift count
  from 39 to 9 violations.
- **Problem 1, SPEC-041** — DELIVERED. Commit `43eeea8`
  (`chore(DIR-015): close out gate-checker-hardening — reconcile drift,
  backfill tests, mark done`, 2026-07-13) backfilled/repointed SPEC-041's 2
  previously-absent mandated tests
  (`TestExemption_BindingDeclaresExemptFromScopeFilterDecoupledFromScopeKind`,
  `TestExemption_ScopeKindDecoupledFromExemptDecision`) into
  `cmd/backstop/go_toolchain_engines_test.go`, dropping the drift count from
  9 to the current 7 (ISSUE-018 + ISSUE-036 only).
- **Problem 4 (SPEC-002 category-coverage stranding)** — MOOT, superseded by
  a broader fix. ISSUE-033 (`De-Go plan validation file classification`) has
  already landed (status `closed`); `fileCategory()` and the
  `plan/final-phase-missing-category` check are fully gone from
  `pkg/validate`. This issue had prescribed a surgical fix — narrow SPEC-002's
  REQ-004 and retire only CLM-026/CLM-027 while leaving the rest of the spec
  live. That never happened, and now can't meaningfully happen: SPEC-051
  (BUNDLE-014 Seed 2, "Legacy Reconciliation Version Backfill",
  `implemented` 2026-07-14/15) instead retired the ENTIRE `agent-definitions`
  scaffolding cluster — the bundle plus SPEC-002/003/004 (all three cited
  only `agent-definitions:REQ-004..018`) — to terminal `deprecated`, for a
  reason unrelated to ISSUE-033 (the citation-target bundle itself going
  terminal, requirement-traceability coverage debt). SPEC-002 is now terminal
  and fully exempt from live validation and the drift dimension regardless of
  REQ-004/CLM-026/CLM-027's content, which is a stronger outcome than the
  narrow repoint this issue asked for. Confirmed: SPEC-002's
  `TestPlan_FinalPhase_ComprehensiveVerification` /
  `TestPlan_FinalPhase_IncompleteVerification` are genuinely gone from the
  tree and SPEC-002 carries zero drift violations today (terminal exemption).
  The narrow-repoint approach this issue originally prescribed for Problem 4
  is now the wrong approach — do not re-open or hand-edit SPEC-002 to
  "narrow REQ-004"; it is retired.

## References

- ISSUE-042 (`0ce7f3d`) — the `artifact_status_drift` gate dimension whose
  full-sweep run originally surfaced all four problems here.
- `PLAN-ISSUE-048-obsoleted-resolvedby-vocab.plan.yml` / commit `3a7a700` —
  delivered Problems 2 and 3.
- Commit `a7070bb` — delivered 5 of 8 Problem 1 artifacts (ISSUE-002/003/005/006/008 → `obsoleted`).
- Commit `43eeea8` (DIR-015 close-out) — delivered the 6th Problem 1 artifact (SPEC-041, tests backfilled).
- **ISSUE-018** (closed) — residual Problem 1 artifact; CLM-007's 2 mandated tests absent.
- **ISSUE-036** (closed) — residual Problem 1 artifact; CLM-008's 5 mandated tests absent.
- ISSUE-037 / ISSUE-052 — adjacent, forward-looking contracts-pack capability
  work (relational-rule input mode, iota-member gap); not a fix for ISSUE-036's
  absent mandated tests, but useful context for whether a repoint is possible.
- ISSUE-033 (closed) — the eradication that made Problem 4 moot by removing
  `fileCategory()` / `plan/final-phase-missing-category` outright.
- SPEC-002/003/004 (`deprecated`) — retired as one legacy scaffolding cluster
  by SPEC-051, not by this issue's originally-prescribed narrow repoint.
- SPEC-051 (`Legacy Reconciliation Version Backfill`, BUNDLE-014 Seed 2,
  `implemented`) — the artifact that retired SPEC-002/003/004 and the
  `agent-definitions` bundle wholesale.
- ISSUE-031 — prior art for the terminal-state vocabulary; extended by
  `obsoleted` (delivered via this issue's own Problem 2, commit `3a7a700`).
- DIR-016 — parent directive (issue/plan lifecycle hardening); marked `done`
  2026-07-08, the same day this issue's Problems 2/3 and 5/8 of Problem 1
  landed, though this issue itself was never updated to reflect that at the
  time.
