---
title: "Zero Match Harness Patch Makes Pack Unvalidatable"
schema_version: issue/v1

issue:
  id: ISSUE-158
  title: "Zero Match Harness Patch Makes Pack Unvalidatable"
  type: bug
  status: open
  created: "2026-08-17"

complexity:
  scope: contained
  uncertainty: known
  risk: moderate
---

# Zero Match Harness Patch Makes Pack Unvalidatable

## Problem

This is a mandated follow-on filing from `PLAN-ISSUE-148` TASK-005 item 1. It was **surfaced**,
not caused, by that plan's fixture-polarity fix. Everything below is measured by
implementer-issue148 on 2026-08-17 at repo HEAD `c586af3`, with real ast-grep 0.43.0.

`(*e2eWorkspace).installZeroMatchSubstantivenessPack`
(`cmd/backstop/gate_substantiveness_e2e.go:297`) copies `packs/substantiveness` to a temp dir
and **appends** to the copy's `ast-grep/rules/referenced-symbol-go.yml`:

```yaml
files:
  - "harness/fixtures/**/*.go"
```

`ISSUE-113`'s intent for that patch is legitimate: make the Q2 `referenced-symbol` rule match
**zero** test files in the consumer's workspace, producing a healthy-looking pack that yields no
Q2 evidence. But that same glob **also** takes the pack's **own** fixture tree
(`testdata/fixtures/rules/**`) out of the rule's scope. So the pack's declared negative fixture
for `referenced-symbol-go` can no longer trigger, and `pack add` — which runs the full packval
pipeline unconditionally on a scratch copy — **refuses** the patched copy at `phase3-fixtures`.

### Measured

Applying the harness's exact patch to a scratch copy of the pack and running
`./bin/backstop pack test <abs path>`:

- **On the PRE-`ISSUE-148` pack** (HEAD `c586af3` fixtures): exit 1, `phase3-fixtures` fail, 3
  errors:
  ```
  ERROR [phase3-fixtures/semgrep-positive] positive fixture triggered the rule (false positive)
  ERROR [phase3-fixtures/semgrep-negative] negative fixture not triggered
  ERROR [phase3-fixtures/semgrep-negative] negative fixture not triggered
  ```
- **On the `ISSUE-148`-corrected pack**: exit 1, `phase3-fixtures` fail, 1 error:
  ```
  ERROR [phase3-fixtures/semgrep-negative] negative fixture not triggered
  ```

So the `ISSUE-148` lane took this copy from 3 errors to 1 and did **not** clear it. The residual
is this harness's rule patch colliding with packval, not a polarity problem — `ISSUE-148` is
correctly scoped and this is a distinct defect.

### Blast radius — four tests, not three

Every test that installs via this harness stays red, all failing identically at `pack add` ->
`pack test ... failed: pack validation (test) of the validation copy failed in phase3-fixtures: 1
validation error(s)`, none reaching the code under test:

- `TestE2E_ZeroMatchClassification_RefusesInsteadOfPerTestViolations`
  (`cmd/backstop/gate_substantiveness_zero_match_e2e_test.go:86`)
- `TestE2E_ZeroMatchClassification_RefusalIsNotWaivable`
  (`cmd/backstop/gate_substantiveness_zero_match_e2e_test.go:155`)
- `TestE2E_HollowEvidenceBlocksZeroMatchRefusal`
  (`cmd/backstop/gate_substantiveness_refusal_boundary_e2e_test.go:52`)
- `TestE2E_HollowEvidenceBlocksRefusal_IsNotVacuous`
  (`cmd/backstop/gate_substantiveness_refusal_boundary_e2e_test.go:112`)

`PLAN-ISSUE-148`'s own notes predicted only the three `TestE2E_ZeroMatchClassification_*` tests;
the two `TestE2E_HollowEvidenceBlocks*` tests in the refusal-boundary file share the same harness
and the same failure mode, and were not in the original prediction. So the measured residual set
is these **four**, not three.

`TestE2E_ZeroMatchClassification_ControlPackReportsNoViolations` went green under the `ISSUE-148`
fix because it installs the pristine pack via `installSubstantivenessLocalPack`, not the patched
copy — it does not exercise this harness at all and is unaffected.

### A second, smaller finding worth recording: the harness's docstring is now stale and wrong

The comment block immediately above `installZeroMatchSubstantivenessPack`, beginning "WHY THE
PATCHED PACK STILL PASSES `pack test`", asserts that packval never runs these fixtures because
packval's `Rule` struct reads the YAML key `file` while `packs/substantiveness/pack.yml`
declares `rule_path:`, so `rule.File == ""` and phase3 skips every fixture-execution site — and
it cites `ISSUE-092` as tracking that hole. `ISSUE-092` **closed** that hole. `phase3` now reads
`rule.RuleSourcePath()`, does run the fixtures, and does notice the patch — which is exactly why
this issue exists: the comment's premise is what changed underneath it. The comment should be
corrected as part of whatever fixes this; it currently tells a future reader the exact opposite
of the truth.

### The design judgment this issue must not pre-decide

The fix needs a glob (or another mechanism) that preserves `ISSUE-113`'s meaning — "this rule
matches nothing in the consumer's workspace" — while leaving the pack's own fixture tree in
scope so packval can still validate the copy. That is stated here as a constraint, not a
solution; the choice belongs to this issue's own plan/implementation lane. Do **not** propose
weakening a fixture, and do **not** propose `t.Skip` on any of the four tests —
`requireAstGrepE2E` is a `t.Fatalf` by design, and a skipped real-engine E2E is silent vacuous
green.

### Adjacency, stated without conflation

Note the adjacency to `ISSUE-151` (path-scoped pack rules going dark under file dispatch), but
these are **not** the same defect. `ISSUE-151` is about the live consumer gate's dispatch
shape (directory dispatch vs. explicit-file dispatch against a real repo). This issue is about a
**test harness's deliberate rule patch** colliding with packval's unconditional
validate-on-`pack-add` pipeline against the **pack's own fixture tree**. Different mechanism,
different surface — do not merge or route one's fix through the other.

## References

- `PLAN-ISSUE-148` — TASK-005 item 1, the mandated follow-on filing that produced this issue.
- `ISSUE-148` — "Substantiveness Fixture Polarity Inverted"
  (`issues/ISSUE-148-substantiveness-fixture-polarity-inverted.issue.md`) — the fix whose
  verification surfaced this residual; not itself at fault.
- `ISSUE-113` — owns the original intent of the zero-match harness patch (classification globs
  baked to the pack author's layout rather than the consumer's).
- `ISSUE-092` — "Pack Test Phase3 Fixtures Cannot Fail" — closed; its closure is what makes the
  harness docstring's premise stale and is why this defect is now visible.
- `ISSUE-151` — "Path Scoped Pack Rules Dark Under File Dispatch" — adjacent surface (dispatch
  shape vs. pack rule scoping), explicitly not the same defect as this one.
- `cmd/backstop/gate_substantiveness_e2e.go:297` —
  `(*e2eWorkspace).installZeroMatchSubstantivenessPack`, the patch site.
- `cmd/backstop/gate_substantiveness_zero_match_e2e_test.go`,
  `cmd/backstop/gate_substantiveness_refusal_boundary_e2e_test.go` — the four affected tests.
