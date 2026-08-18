---
name: gotest-sarif-one-violation-per-test
description: "go-toolchain's test-to-sarif emits ONE violation per failing test (first file:line only) and go-test is exempt_from_scope_filter — so 'both t.Errorf sites absent' and 'the package must appear in .scope.files' are both wrong CI criteria"
metadata:
  type: project
---

Two facts about `backstop-ai/go-toolchain`'s `go-test` channel that plans routinely get
wrong when writing CI-confirmation criteria against `gate-report.json`:

1. **One violation per failing TEST, not per `t.Errorf`.**
   `scripts/test-to-sarif.sh` captures the FIRST `file.go:NN:` position inside each
   `--- FAIL: TestName` block (`have_pos == 0` guard) and `flush()`es exactly one SARIF
   result. A test that emits N `t.Errorf`s at N positions still yields ONE gate
   violation, carrying the first position. Corroborate against `.backstop/baseline.json`
   — it stores what CI actually produced.
2. **`go-test` declares `scope_kind: project-wide`, `project_target: "./..."` and
   `exempt_from_scope_filter: true`.** Its findings BYPASS diff-scope filtering, so
   "if `pkg/X` is not in `.scope.files` the run proves nothing" is a false premise —
   the tests run and report regardless of what the diff touched.

**Why:** PLAN-ISSUE-165 (2026-08-18) built its CI-confirmation conjunction on both
errors — "BOTH `:221` AND `:246` must be absent" (only `:221` ever existed at the gate
layer) and "package must be in `.scope.files`". Also note gate-report violations are
FAILURES ONLY: no criterion of the form "confirm the test appears as a PASS" is
discharge­able from that artifact.
**How to apply:** whenever a plan names `gate-report.json` as the evidence source, read
the pack's convert script in `.backstop/packs/backstop-ai/go-toolchain/scripts/` and the
engine block's scope flags before accepting the criteria. Related:
[[project_sarif_suppressions_measurement_layer]], [[project_diffscoped_ci_count_confound]].
