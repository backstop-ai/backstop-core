---
title: "Ci Gate Missing Baseline Json"
schema_version: issue/v1

issue:
  id: ISSUE-176
  title: "Ci Gate Missing Baseline Json"
  type: bug
  status: open
  created: "2026-08-18"

complexity:
  scope: contained
  uncertainty: known
  risk: safe
---

# Ci Gate Missing Baseline Json

## Problem

CI's `gate` job (`.github/workflows/ci.yml`, step "Run the gate") runs `backstop gate` against a
worktree with no `.backstop/baseline.json` present, and three ratchet tests that read the
committed baseline fail identically as a result:

- `TestRatchet_CoverageMeasurablePathSiteUnGrandfatheredAfterDeGo`
- `TestRatchet_TestVerifyDiscoverySiteUnGrandfatheredAfterDeGo`
- `TestRatchet_GoPackageMatchersSiteUnGrandfatheredAfterDeGo`

All three (`bun_ratchet_flip_test.go`) fail with the same error shape:

```
read committed baseline: open .../.backstop/baseline.json: no such file or directory
```

Confirmed on run `32172705491` (commit `9aa278e`) and run `32179966270` (commit `f8b3846`) — the
error is byte-identical between the two, present on every recent CI `gate`-job run checked as of
2026-08-18. `.backstop/baseline.json` is gitignored (consistent with the pack-distribution/
baseline model — it is a generated artifact, not committed source), and nothing in
`.github/workflows/ci.yml`'s `gate` job restores, downloads, or generates it before "Run the
gate" runs.

## Not the same as ISSUE-086

`ISSUE-086` ("The Published Baseline Artifact Is Generated With Zero Packs Installed") covers a
DIFFERENT gap: the separate `baseline` job's `backstop baseline generate` step running with no
packs installed, so the PUBLISHED baseline artifact is missing pack-engine-sourced findings.
This issue is about a different job entirely — the `gate` job has NO baseline file to read AT
ALL, packless or otherwise, because nothing ever puts one there before the gate step runs.
`ISSUE-086`'s fix (installing packs before `baseline generate` in the `baseline` job) does not
touch the `gate` job and would not resolve this. The two are adjacent (both are baseline/CI
wiring gaps) but do not subsume each other.

## Impact

Every CI `gate` job run currently fails these three ratchet tests unconditionally, regardless of
what the diff under test actually changes — a standing, always-present false-red in the gate
job's `go-test` findings that has nothing to do with the correctness of whatever change triggered
that run. This was directly observed as 3 of the "5 blocking errors remaining after the
`ISSUE-166` fix" — confirmed pre-existing (byte-identical before and after that fix) rather than
caused by it, but real: anyone reading a CI `gate` job's violation list has to already know to
mentally discount these three, which is exactly the kind of "kept aligned by hope" attribution
burden this repo's own conventions try to eliminate.

## Solution

Not prescribed here. The shape is narrow: the `gate` job needs SOME committed or generated
baseline present before "Run the gate" executes — either restore a published
`backstop-baseline-v1` artifact from a prior run (mirroring how the `baseline` job publishes one),
or generate one in-job the same way the `baseline` job does. Whichever direction is chosen should
account for `ISSUE-086`'s packless-generation gap so the two aren't fixed independently into a
baseline that is present but still incomplete.

## References

- `.github/workflows/ci.yml` — the `gate` job (step "Run the gate") and the separate `baseline`
  job, for contrast.
- CI runs `32172705491` (commit `9aa278e`) and `32179966270` (commit `f8b3846`) — both read
  directly via `gate-report.json`; the three-test failure set and its exact message are
  byte-identical across both, which is what establishes this as pre-existing rather than caused
  by either commit.
- `ISSUE-086` (`published-baseline-generated-packless`) — the adjacent but distinct `baseline`-job
  gap; cross-referenced, not subsumed.
- `PLAN-ISSUE-166` — the lane whose CI verification (`TASK-012`) surfaced this as one of the
  residual, pre-existing failures left after its own fix landed.

### Existence-in-world check

Performed 2026-08-18 before filing: read `ISSUE-086` in full (confirmed different job, different
mechanism, explicitly distinguished above). Searched `issues/` and `bundles/` for
"baseline.json", "bun_ratchet_flip", and "committed baseline". No open issue or bundle charter
already owns the `gate` job lacking a baseline file to read.
