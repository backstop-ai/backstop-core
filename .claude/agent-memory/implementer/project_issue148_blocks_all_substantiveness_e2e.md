---
name: issue148-blocks-all-substantiveness-e2e
description: "ISSUE-148 is CLOSED and packs/substantiveness now passes pack test — but the substantiveness e2e tests are STILL red at `phase3-fixtures`, now for a second, different cause: the e2e helper's own deliberate rule patch"
metadata:
  type: project
---

**Superseded in part, 2026-08-17.** ISSUE-148 (inverted phase3 fixture polarity) reads
`closed: 2026-08-17`, and `backstop pack test <abs>/packs/substantiveness` now passes all six
phases. **But the substantiveness e2e tests are still red at the same-looking
`phase3-fixtures` string, and the cause is now a DIFFERENT one** — so "grep for
`phase3-fixtures`" is still the right first move, and "therefore it's ISSUE-148" is no longer
the right conclusion.

**The live cause:** `cmd/backstop/gate_substantiveness_e2e.go`'s
`installZeroMatchSubstantivenessPack` deliberately appends a `files:` glob to
`ast-grep/rules/referenced-symbol-go.yml` to starve the rule, then calls `pack add`. A
load-bearing comment above that helper asserts *"packval never RUNS these fixtures at all…
that pack-validation hole is ALREADY TRACKED as ISSUE-092"*. **That premise is stale** —
ISSUE-092/141 made phase-3 dispatch actually execute, so the patch now trips
`ERROR [phase3-fixtures/semgrep-negative] negative fixture not triggered` and `pack add`
refuses. Measured red the same way: `TestE2E_HollowEvidenceBlocks*`,
`TestE2E_ZeroMatchClassification_*`, `TestPackAuthoringLoop_EndToEnd`.

**How to apply:** if a substantiveness/zero-match e2e reds in your lane, reproduce it OUTSIDE
the test — copy `packs/substantiveness` to scratch, append the helper's own `files:` patch,
and run `pack test` on both the pristine and patched copies. Pristine passes, patched fails,
under a control binary AND yours ⇒ inherited, not yours
([[project_go_overlay_control_and_mutation_harness]] builds the control binary). Fixing the
stale helper belongs to the substantiveness-e2e lane, not to a passing-through lane. Do NOT
`t.Skip` it: `requireAstGrepE2E` is fail-loud by design and a skipped real-engine test is
silent vacuous green. Related: [[project_smoke_darkpack_prefailures]],
[[project_gate_blind_to_nonmandated_test_failures]].
