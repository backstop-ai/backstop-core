---
title: "Gate Test Verification Runs Full Package, Not a Plan's Narrow -run Filter"
schema_version: issue/v1

issue:
  id: ISSUE-066
  title: "Gate Test Verification Runs Full Package, Not a Plan's Narrow -run Filter"
  type: technical-debt
  status: closed
  created: "2026-07-17"
  closed: "2026-08-17"

resolved-by: ISSUE-129

complexity:
  scope: contained
  uncertainty: known
  risk: moderate
---

# Gate Test Verification Runs Full Package, Not a Plan's Narrow -run Filter

## Problem

A spec/plan `test_command` commonly scopes to `go test ... -run '<claim-name-pattern>'` to name
the tests that prove that artifact's claims. The gate's test verification honors that filter, so
a regression in ANY test whose name does NOT match the pattern stays invisible: the scoped run is
green while the full package is red. The narrow `-run` filter — meant only to MAP tests to claims
— silently doubles as the bound on WHAT MUST PASS, which it should never do.

Discovered in ISSUE-064: `TestWiring_NoBakedAnalyzerDelegateInvoked` was broken by the routing
change (it fed a synthetic finding into the migrated routing path without the new role property)
and failed deterministically under `go test ./cmd/backstop/...`, but matched none of the
`test_command`'s `-run 'Substantiveness|ToolchainStackLabel|IsToolchainPack|ByDeclaration|SelfRule'`
patterns. Every mechanical check — the scoped mandated-test run AND the gate — reported green; the
regression was only visible via an unfiltered `go test`, and (see ISSUE-067) was further masked in
the gate as an opaque engine crash.

## Root cause

Two distinct concerns are conflated onto one `-run` filter: (a) "which tests prove THIS artifact's
claims" (a subset, for claim-mapping / substantiveness) and (b) "which tests must pass for the gate
to be green" (the full package(s) any changed code lives in). (a) is legitimately a subset; (b)
must never be. The gate currently derives (b) from (a).

## Direction (to be specified)

The gate's test step must run the FULL test package(s) in the change's scope (e.g. `go test` over
each touched package with no `-run` filter), independent of the plan's claim-mapping filter. The
`-run`/mandated-test-name mapping stays as the claim→test evidence link (test_verification /
substantiveness), but a green gate must require the whole package green, not just the mapped subset.
Evaluate whether this is enforced in the test-verification step, the toolchain go-test engine, or
both.

## Resolution

Not delivered by a plan against this issue — no `PLAN-ISSUE-066` was ever authored, correctly:
the defect this issue describes no longer reproduces against current code, dissolved by three
unrelated deliveries, none of which names ISSUE-066 directly.

**Verification (measured, 2026-08-17):** a fresh `./bin/backstop` binary was built in an isolated
`git worktree` at HEAD (`a60015c`), with `.backstop/` copied in. A production-only defect was
injected — the pinned semgrep version string in `pkg/pack/engine/allowlist.go` changed to
`9.99.9`, the only file touched (`git status --porcelain` showed exactly one modified file) — and
the default diff-scoped `./bin/backstop gate` was run. No clean control run was taken on the same
worktree with the defect reverted, so this does NOT claim the gate would otherwise have been
green — the tree carries pre-existing, unrelated warn-level reds (mostly
`requirement_traceability_advisory` against un-implemented bundle requirements). What the run does
establish: **exit 1, FAIL, 323 total violations**, of which **108 quote the injected `9.99.9`
literal verbatim** — a string that existed nowhere in the tree before the injection, so
attribution of those 108 to the injected change is unambiguous without a control. 138 of the 323
violations came from the `backstop-ai/go-toolchain/go-test` binding; the 108 attributable ones
span **29 distinct `_test.go` files**, none in the diff (the diff was one non-test file) —
representative examples: `workflows_test.go`, `by_declaration_regression_test.go`,
`capability_regression_test.go`, `ci_recipes_bitbucket_pipelines_test.go`,
`baseline_identity_dispatch_e2e_test.go`, `phase3_rulepath_dispatch_test.go`,
`pipeline_dispatch_e2e_test.go`. Four `mandated_test_failed` critical violations also fired,
joining the failures back to SPEC-037 CLM-031/032/033/034. The 108 are not 108 independent
regressions — they're a mix of direct assertion failures (e.g.
`TestCIWorkflow_InstallsProvisionedToolsAtAllowlistPins` asserting the pin itself) and cascade
failures (the changed pin made semgrep fail pack validation's engine trust gate, which failed
phase3-fixtures, which failed a large e2e family) — but both modes are genuine instances of "a
production change breaks tests, by name, in files the diff never touched," which is exactly this
issue's scenario: a production change breaking tests whose names match no `-run` claim-mapping
pattern, in files the plan's `test_command` filter never named. The worktree was discarded after
the injected change was reverted; the main tree was never touched. (Full captured gate output:
`/tmp/g066.txt` at measurement time — transient, not part of this repo.)

**Why it no longer reproduces — three deliveries, accumulated:**

- **SPEC-034/040/042** demoted `verification.test_command` to inert metadata. Its VALUE (the
  `-run` filter) is never read or executed by core anymore — the only consumers are a struct-field
  declaration (`pkg/gate/step_coverage.go:17`) and an assignment (`pkg/gate/step_testverify.go:675`)
  — only its *presence* still gates coverage extraction.
- **ISSUE-070** made the diff-scope filter apply to dispatched pack violations at all, closing a
  prerequisite gap in the scoping mechanism this issue's fix would have needed anyway.
- **ISSUE-129** declared `exempt_from_scope_filter: true` on the go-toolchain `go-test` engine
  binding, so a project-wide test failure REDs a diff-scoped gate even when the failing test's file
  sits outside the current diff.

The pass/fail bound is now the go-test engine's `project_target: "./..."` (the whole module) —
`fileModeTestTarget` only narrows a package-scoped engine under explicit `gate --file`, never
under the default diff scope this issue's scenario runs under. The plan's `-run` filter is now
purely a claim→test evidence mapping (test_verification / substantiveness), never the bound on
what must pass — which is precisely the fix this issue's Direction section called for, arrived at
by a different route than a dedicated plan.

**Residual, explicitly not closed here:** `verification.test_command` remains a REQUIRED
spec/issue field that core never executes, conventionally authored with a narrow `-run` filter
that still reads (misleadingly, to a human or agent reader) as "the pass bound." The gate itself
is honest; only the field's documentation-value is stale. Worth a small follow-up issue if wanted
— not filed as part of this closure.

**Also noted, not a defect:** the installed `go-toolchain` pack has drifted to v1.6.0
(`.backstop/packs/backstop-ai/go-toolchain/pack.yml:111`), past both the v1.4.0 named in
`PLAN-ISSUE-129`'s TASK-005 and the v1.5.0 recorded in ISSUE-129's own Resolution — the
`exempt_from_scope_filter: true` declaration survives intact at v1.6.0, so nothing is broken, but
it's recorded here so a future relock doesn't get surprised by stale version-drift bookkeeping.

## Notes / references

- Surfaced by ISSUE-064's impl-review. Sibling to ISSUE-067 (the same regression was ALSO masked at
  the gate layer by the go-test engine's opaque-crash reporting) — the two failures compounded: a
  narrow filter hid it from the mandated run, and an opaque crash hid it from the gate. Either fix
  alone would have surfaced it.
- Sibling to the gate-verdict-honesty cluster (ISSUE-067, ISSUE-091, ISSUE-118, ISSUE-129) named in
  ISSUE-129's own Notes/references.
