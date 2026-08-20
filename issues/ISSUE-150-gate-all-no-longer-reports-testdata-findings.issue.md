---
title: "Gate All No Longer Reports Testdata Findings"
schema_version: issue/v1

issue:
  id: ISSUE-150
  title: "Gate All No Longer Reports Testdata Findings"
  type: technical-debt
  status: closed
  created: "2026-08-16"
  closed: "2026-08-19"

complexity:
  scope: isolated
  uncertainty: known
  risk: safe
resolved-by: b113d1286a3e7376d0cd40e85e4d5b61f405c76
---

# Gate All No Longer Reports Testdata Findings

## Problem

This is a deliberate, founder-visible **behavior change**, not a bug fix, and it is
**subtractive**: `gate --all` no longer reports findings that live inside `testdata/`
directories. It is filed per PLAN-ISSUE-091 TASK-006 item 2 / CLM-009 ("the consequences are
filed, not absorbed"), rather than folded into that plan's own tasks.

### Mechanism

PLAN-ISSUE-091 collapsed the all-scope dispatch in `runFindingsEngine` (`cmd/backstop/
pack_gate.go`) onto the same explicit-file-list path the diff scope already used. That path
runs `excludeTestdataPaths` — the ISSUE-040 filter, an exact `testdata` path-SEGMENT match
(look-alikes such as `testdata_util.go` are NOT excluded). Before ISSUE-091, that filter applied
only to the diff-scoped branch; `--all` took a separate `projectRoot` directory-target branch
that was never testdata-filtered. After ISSUE-091, both scopes dispatch through the same
explicit-file-list path, so the filter now applies to **both** consistently.

### The measured cost

Re-measured by the ISSUE-091 implementer at the current working tree, real semgrep 1.156.0, on
the `cmd/backstop` subtree, at the **ACTIVE layer** (suppression-filtered — i.e. after
`parseSarif` drops rows carrying an in-source `nosemgrep` suppression, which is what the gate
actually prints). The change removes exactly **3** real rows. All three are `go-standards` rows,
so the figure is the same under either pack set measured (go-standards alone, or all four
installed semgrep packs — backstop-self, ci-workflows, cobra-cli-standards, go-standards):

- `go.security.no-hardcoded-credentials` @ `cmd/backstop/testdata/dogfood_enforcement/known_bad.go`
- `go.core.no-panic-in-library-code` @ `cmd/backstop/testdata/semgrep/fixtures/capture/sample.go`
- `go.core.no-panic-in-library-code` @ `cmd/backstop/testdata/severity-policy-ab/capture/sample.go`

Identify these by `(File, Rule)`, not by line number — line numbers are not stable across
concurrent edits in this repo. Any future re-measurement of this figure must state its pack set,
subtree, and measurement layer alongside the count; a bare number is not reproducible without
all three (PLAN-ISSUE-091's notes are emphatic on this point).

### Rationale (this is a real one, not a hand-wave)

A `testdata` directory is inert data by universal Go convention (`go help packages`: never
compiled, never vetted), and the three rows above are exactly what that convention exists to
protect: deliberately-planted rule violations that exist so a test can prove enforcement
transfers (the same class ISSUE-040 excluded from the diff-scoped gate in the first place). This
issue is ISSUE-040's rationale now applied consistently to both scopes instead of only one — not
a new argument.

### Open question for the founder — do NOT resolve here

Anyone who has been using `gate --all` to audit fixture **content** — i.e. relying on the
whole-repo sweep to surface exactly this class of planted/negative fixture — loses that view
entirely as of this fix. Is that view worth a separate flag? This issue does not propose or
design one; it only raises the question for founder judgment.

## Completed-plan claim retraction

This item also carries a completed-plan claim-prose retraction, given the same treatment as the
sibling filing `ISSUE-149` — recorded here, never by rewriting the completed plan.

`plans/PLAN-ISSUE-040-testdata-findings-exclusion.plan.yml` (status: `completed`) CLM-005 states,
verbatim:

> The nil-scope / GateScopeModeAll whole-repo escape hatch is unchanged (still scans projectRoot;
> NOT testdata-filtered), and the exact-segment match does not exclude look-alike paths (e.g.
> `testdata_util.go`).

PLAN-ISSUE-091 falsifies **both halves of that sentence for the all-scope arm only**: post-fix,
`--all` no longer takes the nil-scope escape-hatch shape the claim describes (it now dispatches
an explicit, testdata-pruned file list instead of the bare `projectRoot` directory target), and
that all-scope path **is** now testdata-filtered.

What survives unchanged:

- **The nil-scope arm** — a `nil` scope still hands the engine the bare `projectRoot` directory
  target, unfiltered. This half of CLM-005 is accurate today exactly as written.
- **The look-alike-path clause** — the exact-segment match still does not exclude paths like
  `testdata_util.go`. This half of CLM-005 is also accurate today exactly as written.

The retraction is scoped strictly to the `GateScopeModeAll` arm of CLM-005's first clause. Do not
read this as retracting the nil-scope behavior or the look-alike-path behavior — both survive.

Note also that PLAN-ISSUE-040's TASK-001 mandated a "re-assert unaffected" assertion covering the
all-scope arm; that assertion was delivered as the `all scope` subtest inside
`TestPackEngines_AllScope_RestoresWholeRepoScan`, which PLAN-ISSUE-091's TASK-003 has now
inverted — the same enclosing test name was preserved (it stays accurate because the surviving
`nil scope` subtest genuinely still restores the whole-repo scan), but the `all scope` subtest's
assertion direction flipped.

## Enforcement scope

Stated explicitly so the next reader does not go hunting for a mandated test: **this is a
claim-prose retraction only, and not a broken mandated-test promise.**

`plans/PLAN-ISSUE-040-testdata-findings-exclusion.plan.yml`, exactly like `plans/PLAN-ISSUE-010-
pack-engines-diff-scope.plan.yml`, declares **no** `test_names:` field anywhere — verified:
`grep -c test_names plans/PLAN-ISSUE-040-testdata-findings-exclusion.plan.yml` returns `0`. It
names its tests only in task-description prose. `pkg/gate/artifact_status.go` builds a plan's
`MandatedTests` solely from `phases[].tasks[].test_names`, so PLAN-ISSUE-040 contributes zero
gate-enforced mandated tests. There is therefore **no asymmetry** between this filing and
`ISSUE-149` (which retracts PLAN-ISSUE-010's CLM-004 prose under the identical no-`test_names:`
condition) — both completed plans are pure prose-only mandates on this point, and neither
retraction reds any gate dimension.

## Resolution

Like `ISSUE-149`, this issue's filed text is itself the complete deliverable — per
`PLAN-ISSUE-091` TASK-006 item 2 / CLM-009. The "measured cost" and "completed-plan claim
retraction" sections above ARE the resolution: a founder-visible record of the subtractive
behavior change (`gate --all` no longer reports testdata findings) and the corresponding
retraction of `PLAN-ISSUE-040` CLM-005's all-scope claim. The open question for the founder
(whether the lost testdata-content-audit view is worth a separate flag) remains genuinely open
and is NOT resolved by this closure — closing this issue records the consequence, it does not
answer that question.

Filed and committed at `b113d1286a3e7376d0cd40e85e4d5b61f405c76`, alongside its sibling
`ISSUE-149`. No backing plan implements this issue, so it closes via `resolved-by` rather than
`delivered_by`.

## References

- `ISSUE-091` — "gate --all under-reports test findings" (`issues/ISSUE-091-*.issue.md`), the
  source issue whose fix produced this consequence.
- `PLAN-ISSUE-091` — `plans/PLAN-ISSUE-091-gate-all-underreports-test-findings.plan.yml`,
  TASK-006 item 2 (files this issue) and CLM-003 (all-scope now prunes `testdata` segments),
  CLM-009 (the "consequences are filed, not absorbed" claim this issue satisfies).
- `PLAN-ISSUE-040` — `plans/PLAN-ISSUE-040-testdata-findings-exclusion.plan.yml` (status:
  `completed`), the completed plan carrying the CLM-005 prose partially retracted above.
- `ISSUE-040` — "Gate Substantiveness Scans Testdata Fixtures" (closed, `delivered_by:
  PLAN-ISSUE-040`), the origin of the testdata-exclusion rationale this issue extends to
  `--all`.
- `ISSUE-149` — sibling filing (PLAN-ISSUE-091 TASK-006 item 1), the CLM-004 supersession on
  `PLAN-ISSUE-010`; this issue follows the same "retract via new issue, never rewrite the
  completed plan" pattern.
- `DIR-032` — Gate Verdict Honesty, this issue's home directive; slot alongside the rest of the
  ISSUE-091 follow-on cluster.
