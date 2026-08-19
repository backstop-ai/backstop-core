---
title: "Ci Gate Missing Baseline Json"
schema_version: issue/v1
delivered_by: PLAN-ISSUE-176

issue:
  id: ISSUE-176
  title: "Ci Gate Missing Baseline Json"
  type: bug
  status: closed
  created: "2026-08-18"
  closed: "2026-08-18"

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

## Root cause, sharpened

Verified 2026-08-18 by reading `cmd/backstop/gate.go` and `cmd/backstop/baseline.go` in full,
confirming `.github/workflows/ci.yml`'s actual `permissions:`/env blocks, and pulling the real job
logs (`gh run view 32179966270 --log`, `gh run view 32172705491 --log`) plus both runs' downloaded
`gate-report.json` artifacts. This replaces the "not prescribed" framing below with a concrete
finding: the self-healing mechanism already exists in full; the gap is CI wiring around it.

**The self-healing pull is real and already wired into `backstop gate` itself — this is not a
"no mechanism exists" gap.** `resolveBaselineCache` (`cmd/backstop/gate.go:331`) runs
unconditionally at the top of every `backstop gate` invocation (`gate.go:208`), BEFORE `g.Run()`
(`gate.go:244`) executes the ordered steps — including `pack_engines`, which is what spawns the
`go test` subprocess that runs the three failing tests. When `.backstop/baseline.json` is absent
or stale, `resolveBaselineCache` calls `refreshBaselineFromRemote` → `runBaselinePull`
(`cmd/backstop/baseline.go:114`), a fully built pipeline: resolve the GitHub repo from `git remote
get-url origin`, check `gh auth status`, find the latest successful main-branch run via the GitHub
API, download that run's published `backstop-baseline-v1` artifact (the one the `baseline` job in
this same workflow already publishes), validate it, and write it atomically to
`.backstop/baseline.json`. On any failure along the way it degrades to a nil baseline plus a
warning string, never a crash.

**The real gap is CI wiring, and it is two independent blockers stacked, not one.**
`.github/workflows/ci.yml`'s `gate` job sets no `GH_TOKEN`/`GITHUB_TOKEN` anywhere (confirmed: zero
matches for either name in the whole file) and never runs `gh auth login`. GitHub Actions does not
inject a token into a step's environment automatically — only an explicit `env: GH_TOKEN: ${{
secrets.GITHUB_TOKEN }}` (or equivalent) does that, and nothing in this job does. So
`ensureGitHubAuth` (`baseline.go:193`), which shells to `gh auth status`, almost certainly fails on
its own, before the pull ever reaches the GitHub API — this is the FIRST and more proximate
blocker, ahead of the permissions question. Even if a bare token env were added, the workflow's
top-level `permissions:` block is `contents: read` only (`ci.yml:9-10`) — no `actions: read` — and
the auto-generated `GITHUB_TOKEN` is scoped by that block, so the pull's subsequent `gh api
repos/.../actions/runs` and `.../actions/artifacts` calls would then 403. Both facts need fixing
together: an authenticated token AND a `permissions:` grant of `actions: read`.

**Direct confirmation from existing CI artifacts was not possible, and the reason is itself a
finding worth recording.** Neither the human-readable job log nor the downloaded `gate-report.json`
for either cited run carries `resolveBaselineCache`'s warning text anywhere — not "no cached
baseline found," not "remote baseline fetch failed," nothing (`grep`ped for both across the full
1099-line log of run `32179966270`; zero hits). The reason: this repo's dogfood `backstop.yml`
configures per-dimension enforcement policy, and `ApplyPolicy` (`pkg/gate/policy.go:182`)
unconditionally overwrites the `baseline_comparison` step's `Reason` field to the literal string
`"superseded by per-dimension enforcement policy"` before either the human table or `--json` ever
renders it — regardless of whether `g.baseline` was nil (pull failed) or populated (pull
succeeded; comparison just isn't this repo's blocking mechanism). Both surfaces confirm the
identical overwritten string: the printed table (`baseline_comparison  skipped  (superseded by
per-dimension enforcement policy)`) and the downloaded `gate-report.json`'s
`baseline_comparison.reason` field. So the two blockers above are argued from reading the code and
the workflow file, not observed directly in a log line — I could not find positive evidence of the
pull failing in-band on real CI, only strong code-level grounds to expect it fails, plus a
concrete, cited reason (`ApplyPolicy`'s unconditional overwrite) why no existing artifact could
show it either way even if it had. Whoever picks this up should treat "does the pull actually fail
on CI today" as still worth a one-off direct confirmation — e.g. temporarily add a bare `run:
./bin/backstop baseline pull` step and read its own exit code/stderr, which bypasses the swallow —
rather than treating it as settled by this write-up.

**Separate, narrower gap: the 3 failing tests bypass the self-healing path entirely.**
`bun_ratchet_flip_test.go`'s `committedBaselineNeutralSpineFiles` (line ~18) reads
`.backstop/baseline.json` via a bare `os.ReadFile` with a hard `t.Fatalf` on any read error — it
never goes through `resolveBaselineCache`. Within the CI `gate` job specifically this doesn't
independently matter: `resolveBaselineCache` runs inside the SAME `backstop gate` process, before
the `pack_engines` step spawns the `go test` subprocess that runs this test, so a wiring fix alone
would already have the file on disk before these tests execute — no separate `baseline pull` CI
step is needed to make the `gate` job's three failures go away. It DOES matter for every path that
runs these tests without going through `backstop gate` first: `make test`, `make ci`, and a bare
`go test ./...` (`Makefile:8-9`) all invoke `go test -race ./...` directly, with no gate wrapper
and no baseline resolution of any kind. Anyone running the suite locally or via `make ci` without
having first run `backstop gate`/`backstop baseline pull` hits this same fatal error today, and
will keep hitting it regardless of whatever CI wiring fix lands.

## Solution

Two independent fixes, in this order — the first alone resolves the CI `gate` job's three failing
tests; the second is required only for paths that never invoke `backstop gate`:

1. **Wire GitHub auth + `actions: read` into the CI `gate` job** so `resolveBaselineCache`'s
   already-built self-healing pull can succeed with no new mechanism needed:
   - Give the "Run the gate" step (or the whole job) `GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}` (or
     `github.token`) in its `env:` so `gh auth status` succeeds.
   - Add `actions: read` to the `gate` job's `permissions:` block (job-level, not repo-wide, to
     keep the token's blast radius minimal) so the pull's `gh api .../actions/runs` and
     `.../actions/artifacts` calls don't 403.
   - Confirm the fix by checking that `.backstop/baseline.json` exists after "Run the gate"
     completes, or that the three named tests pass — NOT by reading `baseline_comparison`'s
     printed reason, which this investigation found is swallowed by `ApplyPolicy` regardless of
     outcome and so cannot confirm or refute a successful pull either way.
   - This composes with `ISSUE-086` rather than needing to resolve it first: whatever the pull
     downloads is exactly whatever the `baseline` job published, so `ISSUE-086`'s
     packless-generation gap flows through unchanged either way — fixing `ISSUE-086` separately
     still improves what lands here, and this fix does not need to wait on it.

2. **Separately, harden the non-gate test paths** (`make test`, `make ci`, bare `go test ./...`)
   against the same "no baseline present" condition, since `bun_ratchet_flip_test.go`'s
   `committedBaselineNeutralSpineFiles` reads the file directly with no fallback and none of these
   paths go through `backstop gate`'s `resolveBaselineCache`. Shape not prescribed here — a
   `TestMain`/package-level `backstop baseline pull` invocation gated on file absence and a
   documented local prerequisite are both plausible — but whichever direction is chosen should not
   make every local `go test` run hit the network by default, and a silent `t.Skip` on a missing
   baseline would be worse than today's loud `t.Fatalf`: it would quietly stop protecting the
   ratchet property these tests exist to guard.

## CI Confirmation (2026-08-18)

Recorded ahead of close-out (which is gated separately on `PLAN-ISSUE-176` TASK-010, once a real
CI push gates fully green) — these are observations from the first real pull on real Linux CI,
not a resolution.

`PLAN-ISSUE-176` landed the fix and CI run `32194863181` exercised it for real: the guarded
"Confirm the self-healing baseline pull landed a file" step confirmed `.backstop/baseline.json`
was present after the gate ran, fetched from the newest successful `main` run at the time.

1. **The pulled artifact was already ≥2 days stale, as predicted during planning
   (CLM-007(b)).** The artifact's `generated_at` is `2026-08-16T02:26:32Z` — from a run that
   predates this whole session's CI-hardening work, because `main`'s `gate` job had not passed
   since (it is `needs: gate`, and the gate job had failed on every `main` push for 21 consecutive
   runs as of planning). This is **expected and non-blocking, not a new defect**: it self-corrects
   the first time any push gates fully green on `main`, which triggers the separate `baseline`
   job to publish a fresh artifact. Recorded here as a confirmed prediction, not a finding.

2. **The published artifact carries no explicit retention override, so GitHub's 90-day default
   applies (CLM-007(c)).** Neither `upload-artifact` step in `.github/workflows/ci.yml` sets
   `retention-days:`. The artifact pulled by run `32194863181` was generated `2026-08-16T02:26:32Z`
   and so is on track to expire around **2026-11-14**. If `gate` is still red on `main` past that
   date, the pull will start failing for a **second, distinct** reason — `expired: true` on the
   located artifact, not an auth/permission problem (which is what this issue's own fix addressed)
   and not the workflow-name-selection gap tracked separately in `ISSUE-178`. This condition is
   self-diagnosing rather than silent: the guarded confirmation step's own bare `baseline pull`
   invocation surfaces the real `gh` error text in its stderr on failure, which is exactly why that
   step exists. Named here so it is not later mistaken for a regression when it eventually fires.

## Resolution

`PLAN-ISSUE-176` (`status: completed`) fixed `.github/workflows/ci.yml`'s `gate` job:

- Step-level `GH_TOKEN` on exactly the 3 steps that shell `gh` (never job-level), plus job-level
  `permissions: {contents: read, actions: read}` — authorizing `backstop gate`'s already-existing
  self-healing baseline pull (`resolveBaselineCache` / `runBaselinePull`) to actually succeed on CI
  instead of silently degrading to a nil baseline.
- A new guarded confirmation step (`id: run_gate` + `if: always() && steps.run_gate.conclusion !=
  'skipped'`) that reports whether the pull landed a file. It ships the **hard-fail** verdict mode —
  any baseline-unavailable condition now reds the blocking gate job — a deliberate choice, not the
  softer warn-only alternative that was also considered.
- Separately, `make baseline` (opt-in, depends on `build`, never wired into `test:`/`ci:`) plus a
  `readGeneratedBaseline` helper give the non-gate test paths (`make test`, bare `go test`) an
  actionable error naming the remedy instead of a bare file-not-found.

**Verified via real Linux CI** (PR #4, merged as commit `14d87ad`): run `32194863181`'s confirmation
step passed with "baseline present: 240442 bytes", and the three originally-failing ratchet tests
(`TestRatchet_CoverageMeasurablePathSiteUnGrandfatheredAfterDeGo`,
`TestRatchet_TestVerifyDiscoverySiteUnGrandfatheredAfterDeGo`,
`TestRatchet_GoPackageMatchersSiteUnGrandfatheredAfterDeGo`) appear zero times in that run's log —
gone, not merely different. A counterfactual on the pre-fix commit (`0943ec4`) confirmed 4
violations naming all three ratchet tests with the exact `read committed baseline: ... no such file
or directory` error; post-fix it's 1 violation (the unrelated, already-tracked `ISSUE-177`
anomaly) — 4 → 1, measured.

**Related, not absorbed:**

- The pulled baseline artifact was confirmed ≥2 days stale on first pull (expected, self-corrects
  once any push gates fully green — see the "CI Confirmation" section above).
- `ISSUE-178` was filed as a genuine follow-on: `resolveLatestSuccessfulMainRun` doesn't filter by
  workflow name — latent today, live the moment a second workflow runs on `main`.

**Scope of this close:** the three named ratchet failures are resolved. This explicitly does NOT
mean "CI is green" — the `gate` job is still red overall due to the separate, unrelated `ISSUE-177`.

## References

- `.github/workflows/ci.yml` — the `gate` job (step "Run the gate") and the separate `baseline`
  job, for contrast; `permissions:` block at lines 9-10 (`contents: read` only, confirmed no
  `actions: read` anywhere in the file); zero `GH_TOKEN`/`GITHUB_TOKEN` matches in the whole file.
- `cmd/backstop/gate.go:208,331` (`resolveBaselineCache`) and `cmd/backstop/baseline.go:114-200`
  (`runBaselinePull`, `ensureGitHubAuth`) — the self-healing pull mechanism read in full to confirm
  it already exists and where it sits relative to `g.Run()`.
- `pkg/gate/policy.go:182` (`ApplyPolicy`) — the unconditional `Reason` overwrite that makes
  `resolveBaselineCache`'s warning unrecoverable from either the human report or `--json` on this
  repo's dogfood config, which is why direct CI confirmation wasn't possible from existing
  artifacts.
- `Makefile:8-9` — `test`/`ci` targets invoke `go test -race ./...` directly, with no
  `backstop gate` wrapper, which is why the bare-`os.ReadFile` gap in `bun_ratchet_flip_test.go`
  is real and separate from the CI `gate` job's wiring gap.
- CI runs `32172705491` (commit `9aa278e`) and `32179966270` (commit `f8b3846`) — both re-read
  2026-08-18 via `gh run view <id> --log` (full job log, grepped for "baseline" and for
  auth/permission text) and via `gh run download <id> -n gate-report` (the actual
  `gate-report.json` artifact); the three-test failure set, its exact message, and the
  `baseline_comparison` reason string are byte-identical across both, which is what establishes
  this as pre-existing rather than caused by either commit, and confirms the `ApplyPolicy` swallow
  applies identically to both.
- `ISSUE-086` (`published-baseline-generated-packless`) — the adjacent but distinct `baseline`-job
  gap; cross-referenced, not subsumed.
- `PLAN-ISSUE-166` — the lane whose CI verification (`TASK-012`) surfaced this as one of the
  residual, pre-existing failures left after its own fix landed.

### Existence-in-world check

Performed 2026-08-18 before filing: read `ISSUE-086` in full (confirmed different job, different
mechanism, explicitly distinguished above). Searched `issues/` and `bundles/` for
"baseline.json", "bun_ratchet_flip", and "committed baseline". No open issue or bundle charter
already owns the `gate` job lacking a baseline file to read.
