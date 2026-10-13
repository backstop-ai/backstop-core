---
title: "ConfigErr halt skips all later gate steps"
schema_version: issue/v1

issue:
  id: ISSUE-211
  title: "ConfigErr halt skips all later gate steps"
  type: bug
  status: open
  created: "2026-10-12"

complexity:
  scope: cross-cutting
  uncertainty: exploratory
  risk: moderate
---

# ConfigErr halt skips all later gate steps

## Problem

`Gate.Run` (`pkg/gate/gate.go:166-180`) breaks out of the step loop the first time a step returns
`ConfigErr: true`, sets `configErrHalt`, and returns exit code 2. Every step after the failing one
never runs and is absent from the result, so the run produces no evidence for them (tests,
coverage, contracts, traceability, drift, waiver, baseline, ledger, whichever come later).

This is general to all `ConfigErr` steps. It was recorded as deferred item (a) in ISSUE-209's
"Deferred / out of scope" section ("ConfigErr halt skips every later gate step"); ISSUE-209
deliberately kept it out as a gate-wide design question. This issue is that item.

It is fail-closed, not a vacuous green: the exit code is 2 and nothing downgrades it. But it is the
largest evidence loss in the gate: one broken pack or config hides whether everything else passes,
and each fix reveals the next failure one run at a time.

## Production ConfigErr sites

Step order as assembled (`cmd/backstop/gate.go:1218-1221`): `pack_lock_verification`,
`artifact_validation`, `pack_engines`, then `test_verification`, `test_substantiveness`,
`coverage_threshold`, `contract_signature`, drift and traceability steps, `waiver_resolution`,
`baseline_comparison`, `ledger_integrity`. The halt therefore skips the most downstream evidence
when it fires early.

Sites that set `ConfigErr: true` (non-test code), by step:

- `pack_loading`: `cmd/backstop/gate.go:925`, `:944` (pack load failure; this path returns a
  one-step list so nothing else was built anyway), `:988` (bad pack-declared test-name pattern).
- `pack_lock_verification`: `gate.go:1119`.
- `pack_engines`: `gate.go:1163` (engine provisioning refused), `:1183` (dispatch error;
  evidence aspect tracked in ISSUE-209).
- `artifact_status_drift`: `gate.go:1260`.
- `requirement_traceability`: `gate.go:1343`, `:1352`.
- `test_substantiveness`: `gate.go:1450`, `:1490` (resolving the pack), `:1521` (dispatch),
  `:1598` (evidence refusal, ISSUE-113).
- `coverage_threshold`: `gate.go:~1750-1770` (two returns; the `ConfigErr: true` is at `:1767`).
- `contract_signature`: `gate.go:1993`, `:2016` (two returns near it).
- Traceability polarity: `pkg/gate/traceability_polarity.go:244` (declared command broken) and
  `:258` (declared intent unmet), set per dimension step name.
- Delegated steps: `pkg/gate/step_delegate.go:51` (`artifact_validation`) and `:96`
  (`code_check`), set when the checker returns `*gate.ConfigError`
  (`gate.go:2251`, `:2266`, `:2289` in `cmd/backstop`).

Plus the construction-time path: `Gate.configErr` (`WithConfigError`, `gate.go:50`) returns exit 2
with zero steps before the loop starts (`gate.go:140-147`); it is a different mechanism and is not
the subject here.

## Why the halt may exist

Stated honestly, the halt is deliberate and specified, not an accident:

- **Specified behavior.** SPEC-010 REQ-001 and REQ-009 and its design text say a delegated config
  error "halts remaining steps", "Gate does not attempt to continue executing subsequent steps
  after a config error", and that exit 2 must take precedence "if a config error is detected, the
  command must not report partial gate results". The original implementation
  (`5aafb8f`, SPEC-010) introduced `configErrHalt`. Changing the halt means amending SPEC-010.
  Later specs reference the halt as a mechanism (SPEC-036's traceability-fail-loud design;
  ISSUE-113's refusal relies on it).
- **ConfigErr is doing three jobs.** The comment at `gate.go:1589-1593` records that the halt also
  prevents `waiver_resolution` from running (so a refusal cannot be waived) and, with
  `ApplyPolicy` skipping `ConfigErr` steps (`pkg/gate/policy.go:188`), prevents
  baseline grandfathering. Removing the halt must not lose either guarantee.
- **Waiver and baseline consume the accumulated set.** `waiver_resolution` and
  `baseline_comparison` are computed from the steps run so far (`gate.go:152-158`). If earlier
  steps were skipped, a baseline comparison would see their baselined findings as absent and
  report them as fixed; a waiver pass would adjudicate a partial set. These two cannot simply run
  over a partial result.
- **Real step dependencies.** `test_verification` consumes verdicts published by `pack_engines`
  through a collector populated only if `pack_engines` ran (`gate.go:1000-1012`: ordering "is what
  makes a supplier safe"). Substantiveness's Q2 join depends on engine streams (ISSUE-112
  recorded 397 misleading downstream violations from a partial stream).
- **Avoiding cascading noise.** A broken pack config can make many later steps fail for the same
  root cause; the halt reports one clear message.

What I did not find: any ADR or issue that weighs the halt against the evidence cost. SPEC-010
states the rule but not the trade-off. (Search covered specs/, issues/, and `git log` for
`gate.go`; the halt predates all of it.)

## Impact

- One broken pack, lock, or declaration hides the verdict of every later step; the operator
  cannot tell whether the rest of the tree is clean.
- Serial discovery: each config fix reveals the next failure in a new run (CI round-trips).
- Downstream consumers (JSON gate result, baseline publication) get no evidence for halted
  steps, and `configErrorDiagnostic` (`cmd/backstop/baseline.go:322-340`) can only describe the
  steps that ran.

## Direction

(Direction, not a decided design; options to evaluate.)

1. **Continue independent steps and report each.** Replace the unconditional `break` with
   "continue, but do not run steps that declare a dependency on a failed one". Requires steps to
   declare dependencies explicitly (for example `pack_engines` -> `test_verification`).
2. **Mark skipped steps explicitly.** Steps that do not run emit a `skipped` StepResult with a
   reason such as `skipped: halted by <step>` instead of vanishing, so the output is complete and
   cannot be mistaken for "ran and passed". `skipped` already exists as a status
   (`pkg/gate/result.go:138`, `:291`) and does not by itself produce exit 1.
3. **Keep the halt only for genuinely dependent steps.** At minimum `waiver_resolution` and
   `baseline_comparison` stay skipped when any `ConfigErr` fired (to preserve non-waivable,
   non-grandfatherable semantics), and a pack-loading failure keeps skipping everything that
   needs packs.
4. Keep exit 2 whenever any step set `ConfigErr`; fail-closed behavior must not change, only the
   evidence reported.

Questions to settle in the plan: whether SPEC-010's "must not report partial gate results" is
amended or reinterpreted (partial results labeled as such); whether the dependency declaration
lives on the step or in the assembly order; the JSON output contract for skipped-by-halt steps
and consumers that parse it; and interaction with ISSUE-209's `pack_engines` evidence
preservation.

## Complexity estimate

Cross-cutting (the gate runner, the step assembly in `cmd/backstop/gate.go`, about 15 production
`ConfigErr` sites, the result/JSON contract, and SPEC-010), exploratory (the dependency model and
spec amendment are open design questions), moderate risk (the risk is accidentally letting a
refusal be waived or grandfathered, or reporting a partial baseline comparison).

## Notes

- Repro: install a pack whose lock entry is stale (or declare a toolchain pack whose tool is
  absent from PATH) in a repo that also has an unrelated failing test or coverage shortfall. Run
  `backstop gate`: exit 2 with only the config error; the test and coverage steps are absent from
  the result.
- Related: ISSUE-209 (deferred item (a)), ISSUE-112, ISSUE-113.
