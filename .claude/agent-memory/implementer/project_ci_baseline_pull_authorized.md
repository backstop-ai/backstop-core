---
name: ci-baseline-pull-authorized
description: ISSUE-176 SHIPPED — CI's gate job now binds step-level GH_TOKEN + job-level actions:read, and the self-healing baseline pull WORKS on Linux (run 32194863181 printed "baseline present: 240442 bytes"); the three ratchet failures are gone
metadata:
  type: project
---

`backstop gate`'s self-healing pull (`resolveBaselineCache` →
`refreshBaselineFromRemote` → `runBaselinePull`) always existed and always
degraded SILENTLY to a nil baseline on CI. The gap was authorization only.
Delivered on 2026-08-18 (branch `fix/issue-176-ci-baseline-pull-wiring`, PR #4):

* **step-level** `GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}` on exactly the three
  gate-job steps that can shell `gh` — the two `backstop gate` invocations (both
  run `resolveBaselineCache`) and the confirmation step. NOT job-level: that
  would hand the token to the `curl | sh` installer, the `pipx install`, and the
  eight-repo pack clone.
* **job-level** `permissions: {contents: read, actions: read}` on `gate` only.
  `contents: read` is re-declared because a job-level block REPLACES the
  workflow-level one and unlisted scopes become `none`.

**MEASURED, not asserted** (run 32194863181, `pull_request`, ubuntu-latest):
step "Confirm the self-healing baseline pull landed a file" → **success**,
printing `baseline present: 240442 bytes` — the same size as the darwin
checkout's artifact (sha `0d45fbeb…`). Zero occurrences of `read committed
baseline` / `no such file or directory` in the whole run: the three
`TestRatchet_*UnGrandfatheredAfterDeGo` failures are GONE.

**Why the confirmation step exists at all:** `ApplyPolicy` overwrites
`baseline_comparison`'s Reason unconditionally under this repo's dogfood policy
— the CI log shows `baseline_comparison skipped (superseded by per-dimension
enforcement policy)` whether the baseline is nil or populated. Neither the
printed table nor `--json` can confirm or refute a pull. A guarded post-gate
`[ -f .backstop/baseline.json ]` step is the ONLY usable observable. Guard it
`always() && steps.<gate-id>.conclusion != 'skipped'` — a bare `always()`
misattributes an early job death to ISSUE-176.

**How to apply:** any future "did the pull work on CI" question is answered by
that step's output, never by `baseline_comparison`. If it ever reports absent,
its failure branch runs a bare `baseline pull` and prints that command's own
exit code and stderr. Watch for the ~2026-11-14 artifact-retention expiry of
run 31921681066's baseline if main is still red then. Related:
[[project_ciyml_byte_identity_guard]], [[project_local_baseline_makes_gate_permissive]].
