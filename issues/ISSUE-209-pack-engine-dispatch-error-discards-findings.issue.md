---
title: "Pack Engine Dispatch Error Discards Findings"
schema_version: issue/v1

issue:
  id: ISSUE-209
  title: "Pack Engine Dispatch Error Discards Findings"
  type: bug
  status: open
  created: "2026-10-12"

complexity:
  scope: contained
  uncertainty: exploratory
  risk: moderate
---

# Pack Engine Dispatch Error Discards Findings

## Problem

`dispatchPackEnginesWithEvidence` (`cmd/backstop/pack_gate.go:268`) loops over every declared
pack and, within each pack, every engine. On the FIRST engine that returns an error it returns
immediately, so two things are lost:

- **Earlier evidence is discarded.** Violations already accumulated in the local `violations`
  slice from earlier engines and earlier packs are thrown away: the early return passes `nil`
  as the violations (`newPackEngineDispatchResult(nil, nativeApplied || applied)`,
  `pack_gate.go:311` for a findings engine, `pack_gate.go:301` for a sandbox engine). The
  engine-registry lookup failure (`pack_gate.go:286-288`) returns an empty
  `packEngineDispatchResult{}`, which additionally drops the `NativeSandboxApplied` flag.
- **Later engines never run.** Every engine and pack after the failing one in the loop is
  skipped, so nothing is known about whether they would have passed or failed.

Discovered while filing ISSUE-208, where a SARIF parse error in one engine (gitleaks) erased the
whole `pack_engines` step's evidence. ISSUE-208 fixes why that one parse failed; it does not
change the fact that ANY engine error has this blast radius. This issue is the separate,
engine-agnostic defect.

### Exact control flow

Errors that trigger the early return (every non-nil error out of an engine's run):

- Findings engine, via `runFindingsEngine` (`pack_gate.go:661`): input gathering
  (`gatherEngineInputs`, `:674`), the trusted-tool allowlist refusal (`checkEngineToolAllowed`,
  `:687`), a process that never started (`check.NeverStarted`, ISSUE-112), a declared
  `stdout_artifact` not produced (`:895`), a missing or failing convert script, a strict-SARIF
  shape refusal, `check.ParsePackFindings` failure (`:927`, ISSUE-208's case), and the
  CrashGuard "non-zero exit with no parseable findings" refusal (`:940`).
- Sandbox (validator-driven) engine, via `runSandboxEngineWithEvidence` (`pack_gate.go:299`).
- Registry lookup failure for a rule's declared engine (`pack_gate.go:286`).

What the gate reports. The `pack_engines` step (`cmd/backstop/gate.go:1177-1186`) turns the
error into a single `StepResult{Status: "fail", ConfigErr: true}` carrying exactly ONE
violation: `{Rule: "pack_engines", Message: err.Error(), Severity: "error"}`.
`dispatchResult.Violations` is `nil` on that path, so no violation from any engine that did run
survives. The branch also returns before the unfiltered stream is published to the
test-verdict collector (`collectedVerdicts`, set just after this branch in `gate.go`), so
`test_verification` receives no verdicts from engines that did run.

Exit code. `ConfigErr: true` makes `Gate.Run` (`pkg/gate/gate.go:166-169`) set `configErrHalt`
and `break` out of the step loop, returning exit code 2. Consequences beyond `pack_engines`
itself: every later gate step does not run at all (absent from the result), and `ApplyPolicy`
skips `ConfigErr` steps (`pkg/gate/policy.go:188`), so the failure cannot be
baseline-grandfathered or policy-downgraded.

The same dispatcher also serves the substantiveness step (`gate.go:1513-1523`, same
`ConfigErr: true` plus a single message violation prefixed `dispatching substantiveness pack:`)
and, indirectly, the contract-entry dispatch (`gate.go:1944-1946`). That last site calls the
dispatcher with a single-rule manifest, so there is nothing to merge inside one call; its real
loss is in the loop that calls it (see "Deferred / out of scope"). `backstop pack
test` / packval do not call this dispatcher. It has exactly three production call sites, all
in `cmd/backstop/gate.go` (`:1174-1177`, `:1510-1513`, `:1944`). The first two check the
`dispatchPackEnginesFn` seam (`pack_gate.go:237`) themselves and otherwise call
`dispatchPackEnginesWithEvidence` directly. The wrapper `dispatchPackEngines`
(`pack_gate.go:248-255`, returns `(result.Violations, err)`) is test-only: it is the unused
fallback of `resolveDispatchPackEngines` (`:241`), whose only non-test references (outside its own
definition in `pack_gate.go`) are five comments: `cmd/backstop/gate.go:1438`, `:1809`, `:1882`,
and `cmd/backstop/gate_substantiveness_e2e.go:22` and `:488` (a non-`_test.go` file). Several of
them (`gate.go:1882`, `gate_substantiveness_e2e.go:22`) describe the chain
`resolveDispatchPackEngines` -> `dispatchPackEngines` -> `runFindingsEngine` as the live
dispatch path, which is stale; a plan touching these files should correct those comments. About 23 test files call it directly while tests stub the seam.

## Is there a vacuous green or warning-only path?

None found. This is fail-closed but evidence-destroying. Every error out of an engine reaches
`ConfigErr: true`, `Status: "fail"`, exit 2, and no branch downgrades it to a warning or skip.
Scope mode does not change it: diff-scoped and `--all` differ only in the inputs handed to
engines (`activeScope`), not in the error handling, and the dispatch error is returned before
`activeScope.FilterViolations` runs, so scope filtering cannot mask it. CrashGuard interplay
(ISSUE-207) runs the opposite direction: a non-CrashGuard engine whose run fails silently is
NOT an error at all (it yields zero findings and never reaches this early return), which is
ISSUE-207's vacuous-pass mechanism, a separate defect. This issue concerns only errors that DO
surface. Any green near this code would come from the ISSUE-207/ISSUE-112 class (an engine
failure not surfaced as an error), never from this one.

## Impact

- **One broken engine masks every other engine's evidence.** The operator sees only the first
  failing engine's message. Findings that earlier engines and packs really produced are lost
  for that run, even though those engines ran correctly.
- **The operator cannot tell which other engines would have failed.** Engines after the failure
  are never run, so a second broken engine stays invisible until the first is fixed; each fix
  reveals the next failure one gate run at a time (serial discovery).
- **It contradicts the principle that every check that ran reports what it found.** The gate
  reports evidence per dimension; the dispatcher collapses an entire step's evidence to one
  string.
- **Downstream consumers are starved.** The unfiltered stream for the test-verdict collector
  (ISSUE-118) is never published when dispatch errors, and the halt skips every later step.
- **Which evidence survives depends on declared order.** Pack order and per-pack rule order
  (the `order` slice, `pack_gate.go:279-282`) decide it: an erroring engine early in the order
  hides more than one late in it.

## Relationship to sibling issues

All of these are in one family: the engine-evidence path can lose or skip evidence. Each has a
distinct root cause and fix.

- ISSUE-206: `pack test` phase 3 never dispatches a command engine with `input_mode: none` and
  no `rule_path`/`pattern` (vacuous fixture pass). Evidence skipped because the engine is never
  invoked; packval path.
- ISSUE-207: a findings engine's pack-relative script runs with the wrong cwd; a non-CrashGuard
  engine reads the failure as a clean pass (vacuous green). Evidence skipped because a failure
  is not surfaced as an error.
- ISSUE-208: one non-string SARIF property rejects the whole log. Evidence lost inside a single
  engine's parse; that parse error is the trigger that this issue shows destroying the whole
  step. ISSUE-208's Impact section hands this behavior off to this issue ("tracked separately
  in ISSUE-209").
- ISSUE-209 (this issue): evidence lost and later engines skipped across engines when any one
  engine errors; fail-closed rather than vacuous green.
- ISSUE-112 (closed) is the precedent for the fail-loud half: making an unstartable engine an
  error is correct and must stay; this issue concerns what the dispatcher does with that error.

## Direction

(Direction, not a plan.)

1. Continue dispatching the remaining engines and packs after an engine fails. Do not return on
   the first error.
2. Record each failed engine as an error-severity violation naming the pack and engine (reusing
   the current wrapped message, `dispatching findings engine "X" for pack P: ...`), appended
   alongside all collected findings in dispatch order, so the step reports one consolidated
   result instead of one string.
   The `pack_engines` caller must change too, because it also discards findings independently
   of the dispatcher's return value: on `err != nil` the branch (`gate.go:1179-1185`) builds a
   step result from a single synthetic violation and never reads `dispatchResult.Violations`.
   Fixing only the dispatcher's return value would still surface nothing, so this one error
   branch must merge the preserved violations with the engine-failure violation(s).
   **Substantiveness stays fail-whole.** The substantiveness branch (`gate.go:1517-1524`)
   must explicitly IGNORE any violations the dispatcher now returns alongside an error and keep
   emitting exactly its single `ConfigErr` violation, so no Q2 join ever runs over a partial
   stream (see the ISSUE-112 lean below). The contract-entry path is likewise unchanged (see
   "Deferred / out of scope").
   - **Scope filtering still applies.** Today the error branch returns before
     `activeScope.FilterViolations` (ISSUE-070) runs. Preserved `pack_engines` findings must
     pass through that same filter (project-wide exempt violations retained as today), and
     engine-failure violations must be exempt from it so an out-of-scope engine error is never
     filtered away.
   - **Engine errors stay distinguishable from findings.** Downstream, `configErrorDiagnostic`
     (`cmd/backstop/baseline.go:322-340`) joins EVERY violation message of every `ConfigErr` step
     into the baseline-generate refusal text, so preserved findings would be mixed into an
     engine-error diagnostic. Keep engine-failure violations marked by the step's own rule name
     (`pack_engines`) with severity error, distinct from namespaced pack-rule findings, and have
     the plan decide whether the diagnostic should include only the engine-failure violations.
   - **Baseline and waiver semantics.** Because the step stays `ConfigErr`, the halt skips
     `waiver_resolution` and baseline comparison, and `ApplyPolicy` skips the step
     (`policy.go:188`). Preserved findings are therefore reported for information only: they get
     no baseline grandfathering and no waivers, and the step still fails on the engine error.
     This is the intended minimal semantics; changing it would be a separate decision.
   - **Only if a mark-partial alternative is later chosen for substantiveness:** preserved
     findings would also have to pass the Q1 scope skip (`gate.go` ~1536, `scope.Contains`) and
     the dependent joins would have to be suppressed. This is not the direction here.
3. The step still fails and the exit code stays 2 whenever any engine errored (retain
   `ConfigErr: true` semantics: not baseline-grandfatherable, not downgradable). Fail-closed
   behavior must not change; only the evidence preserved changes.
4. Tests: earlier engines' findings survive a later engine's error; later engines' findings
   survive an earlier engine's error; two erroring engines are both reported; mixed pack and
   engine orders; step status and exit code remain fail/2; a sandbox-engine error and a
   registry-lookup error behave the same. Fail-whole guards: on an engine error the
   substantiveness step still emits exactly one `ConfigErr` violation and no Q2 `noTarget`
   joins over a partial stream, and the contract-entry call (`gate.go:1944`) still fails whole.

Implications to settle in the plan:

- **Determinism and ordering.** Failed-engine violations are emitted in dispatch order (pack
  order, then rule-declared engine order) so output is stable run to run. Do not sort or
  de-duplicate across the error/finding boundary in a way that changes baseline identity for
  real findings.
- **Dispatch evidence recording.** `NativeSandboxApplied` must be the OR across every engine
  that ran, including those after a failure, and must also survive the registry-lookup failure
  path that currently returns an empty result. Extend the evidence tests around
  `newPackEngineDispatchResult` (`cmd/backstop/pack_sandbox_dispatch_result_test.go`) so a
  mid-loop error does not reset it.
- **Return contract.** The function returns `(result, error)`. Preserving findings while still
  signalling failure keeps the signature: the dispatcher returns non-nil violations alongside
  an aggregate error. The test-only wrapper `dispatchPackEngines` and the `dispatchPackEnginesFn`
  seam already pass violations and error through, so they need no signature change; the
  behavioral change is that the `pack_engines` caller reads the violations on error, while the
  substantiveness and contract callers ignore them. Alternatively the signature could change,
  touching the three production call sites plus the wrapper and the seam type.
- **Substantiveness: keep fail-whole (decided).** ISSUE-112 is the precedent: an empty or
  partial engine stream produced 397 misleading downstream join violations. A missing symbol
  extraction would cause false `noTarget` violations, and a missing hollow stream would silently
  drop hollow findings. Substantiveness therefore keeps today's behavior; preserving findings
  applies to `pack_engines` only. Marking the stream partial is recorded only as a possible
  later alternative.
- **Verdict collector.** On a partial stream, decide whether the unfiltered stream is still
  published to the test-verdict collector (ISSUE-118) so `test_verification` sees real verdicts.
- **Halt behavior.** Whether later gate steps (currently skipped by the `ConfigErr` halt) should
  run after a dispatch error is a separate question; keep this issue to the evidence inside
  `pack_engines`.

## Deferred / out of scope

Each item below is a same-class evidence-loss defect deliberately kept out of this issue.
Each is pending founder decision on whether to file as its own issue. No new issues were
created.

- **ConfigErr halt skips every later gate step** (`pkg/gate/gate.go:166-180`). General to all
  `ConfigErr` steps, and arguably the larger evidence loss: after any config error the remaining
  steps never run. Excluded because it is a gate-wide design question, not a dispatcher bug.
- **Coverage dispatcher early returns** (`cmd/backstop/pack_gate.go:381` registry lookup and
  `:394` engine error in `dispatchPackCoverageWithEvidence`, `:363`). Identical first-error
  abort on the coverage channel. Excluded to keep the findings-channel fix reviewable.
- **Contract-entry loop** (`produceContractEngineResultsWithSandbox`, `cmd/backstop/gate.go:1821`;
  the loop at ~`:1839-1845` returns `nil, err` on the first failing contract, discarding every
  other contract's results; `buildContractStepWithSandbox`, `:1999`, then emits one synthetic
  `ConfigErr` violation at ~`:2010-2017`). The loss is across contracts, not inside a single
  dispatcher call, so it needs its own fix. Excluded because it is a different loop with a
  different result type.

## Complexity estimate

Contained, exploratory (open design questions: error/finding distinguishability, verdict
collector on a partial stream, return-contract shape), moderate risk. The loop change itself is
small (accumulate errors instead of returning). The behavioral change is confined to the
`pack_engines` caller; the substantiveness and contract callers must explicitly ignore the
violations now returned alongside an error, which needs tests rather than design. Three production
call sites touch the dispatcher, all in `gate.go`. The test surface is wide (about 50 test files
reference the dispatcher), including about 23 files that call the test-only wrapper
`dispatchPackEngines` directly; those callers will now receive non-nil violations alongside
errors. Most assert success or a single error and should be unaffected, but each must be
checked. Roughly a half-day of focused work including tests; the
risk is in keeping the fail-whole callers truly unchanged, not in the loop.

## Notes

- Existing tests assert only that an error is returned (for example
  `TestDispatchPackEngines_BrokenSemgrepRuleFile` in `cmd/backstop/pack_gate_errors_test.go`,
  and `TestDispatch_UnstartableFindingsEngineFailsLoud` in
  `cmd/backstop/dispatch_unstartable_engine_test.go`); none asserts that other engines' findings
  survive, which is why the defect is untested.
- Repro (by code reading): declare one pack with two findings engines, the first emitting a
  valid SARIF finding and the second returning unparseable output. The gate reports a single
  `pack_engines` error violation, exit 2, and the first engine's finding is absent.
