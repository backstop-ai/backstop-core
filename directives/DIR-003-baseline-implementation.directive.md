---
title: "Baseline Implementation"
number: DIR-003
created: "2026-04-19"
schema_version: directive/v1

directive:
  status: active
  source:
    - "BUNDLE-007"
    - "SPEC-010"
    - "ISSUE-056"
    - "ISSUE-086"
    - "ISSUE-120"
    - "ISSUE-178"
---

## Description

Implement gate step 7 (baseline comparison) and the CI baseline generation workflow. CI runs `backstop gate` post-merge and publishes the violation set as an immutable baseline artifact. Locally, `backstop gate` auto-pulls the latest baseline with TTL-based caching and reports differentials ("3 new violations beyond baseline") instead of absolute counts.

Includes: `backstop baseline pull` command, `.backstop/baseline.json` caching, TTL logic (default 15 minutes), GitHub Actions artifact publishing, structural diff algorithm for violation identity. Per ISSUE-086, the CI `baseline` job must install packs before generating, so the published artifact reflects pack-engine findings rather than a structurally empty engine set.

Depends on DIR-001 (release workflow — CI must exist to generate baselines).

Per ISSUE-120, `backstop baseline pull`'s GitHub-Actions-specific knowledge (GitHub Actions
runs/artifact lookup, `gh auth status`, GitHub-naming error strings) is a candidate zero-baked-
platform-knowledge violation whose disposition — accepted narrow exception vs. an extracted
provider seam — is an open founder decision this directive owns.

Per ISSUE-178, `backstop baseline pull`'s run selection is incomplete: `resolveLatestSuccessfulMainRun` (`cmd/backstop/baseline.go:202-224`) decodes each candidate run's `Name` field but never filters on it, returning the newest successful run on `main` from ANY workflow rather than specifically the `CI` workflow whose `baseline` job publishes the `backstop-baseline-v1` artifact. Latent today, and its fix is entangled with the ISSUE-120 decision above — the obvious repair bakes a fourth GitHub-Actions-specific literal (the workflow NAME) into the same file whose GitHub coupling ISSUE-120 already asks the founder to rule on. These two should be planned together, not separately.

## Notes

- **ISSUE-086 gates this directive's completion, independent of backlog position.** DIR-003's
  pull model presumes CI generates a baseline `backstop gate` can trust and pull from. The
  currently published `backstop-baseline-v1` artifact is generated with zero packs installed
  (`.github/workflows/ci.yml:39-64` runs `./backstop baseline generate` with no `pack install`
  step), so every pack-engine-sourced dimension is structurally absent from it — not clean,
  never evaluated. Any DIR-003 work that consumes today's artifact inherits that vacuum.
  Reordering this directive in BACKLOG.yml does not change this fact. ISSUE-086 must be fixed
  before or as part of DIR-003's delivery, not after.
- **Correction, 2026-08-02: the bullet above is now factually false — verified against the
  current `.github/workflows/ci.yml`.** The `baseline` job now spans `ci.yml:174-258` (not
  `39-64` — the file has grown) and it DOES install packs before generating. `./backstop pack
  install` runs at `ci.yml:241` (step "Install the pack fleet", `ci.yml:233-241`), preceded by
  pinned Layer-0 analyzer installs — golangci-lint v2.6.0 and go-arch-lint v1.16.0
  (`ci.yml:192-208`) — and the provisioned engine-tool installs at their allowlist pins —
  semgrep 1.96.0 and ast-grep 0.43.0 (`ci.yml:210-228`) — all ahead of `./backstop baseline
  generate` at `ci.yml:250`. ISSUE-086's own Verification section lists two criteria; the first
  ("the `baseline` job installs packs before `./backstop baseline generate` runs") is now met.
  ISSUE-086 itself, however, is still `status: open` in its own file despite this — its record
  has not caught up with the CI change either.
  **This correction does NOT lift the hold in the next bullet, and must not be read as doing
  so.** The hold's stated precondition reads satisfied now, but the hold's actual purpose was
  never "wait for that sentence to become true" — it exists to prevent a silent ratchet-clean of
  pack-engine dimensions that were never evaluated before. ISSUE-086's own SECOND Verification
  criterion — "a subsequent gate run against the refreshed baseline does not silently
  ratchet-clean any pack-engine dimension that was previously unevaluated" — is exactly that
  protection, and it is **not yet demonstrated**. That is an independent reason the hold stands:
  a satisfied precondition on the first criterion is not evidence on the second. Lifting the
  hold is a separate, substantive founder decision and is intentionally out of scope for this
  correction; do not self-lift it here or on any future read of this file.
- **The approved coverage-baseline refresh is HELD, founder-ratified 2026-07-27, until the CI
  `baseline` job installs packs.** Refreshing the tracked coverage baseline against today's
  packless artifact would ratchet-declare every pack-engine dimension clean without ever having
  evaluated it — the silent/vacuous green this project exists to prevent (see CLAUDE.md
  "Enforcement philosophy"). This is a hold, not a PM suggestion, and should not be lifted
  without an explicit founder go once ISSUE-086 lands.
- **Founder-reaffirmed, 2026-08-10: the hold stands.** Brandon was asked directly whether to
  lift the hold now that its stated CI precondition reads met (per the 2026-08-02 correction
  above). He said keep it. The CI precondition being satisfied is not sufficient on its own —
  the deeper, still-undemonstrated risk is the one the 2026-08-02 correction identified: a
  baseline refresh could silently ratchet-clean a pack-engine dimension that was never
  evaluated, indistinguishable in outward shape from one that ran and passed. That distinction
  needs to actually be proven safe (e.g. a test/fixture demonstrating never-evaluated vs.
  ran-and-clean are told apart) before anyone refreshes the baseline for real. This closes out
  the 2026-08-02T14:10Z PM-inbox escalation asking for a ruling on lifting the hold — the
  ruling is: not yet, precondition-met is not the same as risk-proven-safe.
- **ISSUE-120, filed 2026-08-11: `backstop baseline pull` bakes GitHub-Actions-specific
  knowledge into core** — `ensureGitHubAuth` (shells out to `gh auth status`), the `Long` help
  text's "Artifact lookup uses GitHub Actions runs and artifact naming semantics", and two
  GitHub-naming error strings, all in `cmd/backstop/baseline.go`. This is a candidate
  zero-baked-checks violation, not yet a ruling: it is NOT a re-litigation of SPEC-067's
  CLM-050 (that claim's case-sensitive `github` scan is settled GREEN per the founder's
  2026-08-11 v1.0.3 ruling, which exempted `github.com/`/`github\.com` module-path references
  and deliberately left these five capitalized mentions out of scope). The open question the
  issue raises and does not answer: keep the GitHub Actions coupling as a documented, narrow
  accepted exception (baseline pull may inherently need to "talk to a CI provider's API" in a
  way that isn't findings-engine-shaped), or extract a `baseline-pull` provider seam a pack or
  config value supplies. That choice is an unmade founder decision. Precedent for the seam
  option exists in-repo: pack distribution already removed a GitHub host assumption under
  SPEC-056 DD-31 (see the comment at `pkg/pack/distribution/identity.go:216`) — read DD-31
  first if planning this. Measured today: `pkg/validate/resolved_by.go:129`'s
  `isPullRequestURL` is a weaker, GitHub-shaped instance (`/pull/`/`/pulls/` vs. GitLab's
  `/merge_requests/` or Bitbucket's `/pull-requests/`) that should not be scope-crept into this
  work; `pkg/gate/result.go:3` and `pkg/packval/sandbox_linux.go:21` are comment-only mentions;
  `pkg/pack/distribution/lockfile.go:32` documents a GitHub property deliberately. Adjacent to
  DIR-019: SPEC-067's four per-platform CI gate-workflow recipes (github-actions, gitlab-ci,
  bitbucket-pipelines, jenkins) create backstop's first non-GitHub CI consumers, but none of
  their rendered workflows invoke `baseline pull` — nothing breaks when they land. The real
  consequence is narrower: a GitLab/Bitbucket/Jenkins consumer wired up by that recipe has no
  path to adopt the baseline ratchet at all.
- **ISSUE-178, filed 2026-08-18: `backstop baseline pull` selects a workflow run without filtering on workflow name.** Filed as the deferred follow-on from PLAN-ISSUE-176's CLM-007(a), which recorded the defect and explicitly declined to fix it in that lane. Verified in tree by the PM, not relayed: the loop at `cmd/backstop/baseline.go:218-222` tests only `HeadBranch == "main" && Conclusion == "success"` while `Name` is decoded at `:207-213` and never read. THREE PM-MEASURED CORRECTIONS a planner needs, none of them in the issue:
  (a) **The issue's premise "this repository currently runs exactly one workflow against `main`" is wrong as stated.** `.github/workflows/` holds THREE workflows — `CI` (ci.yml), `release` (release.yml), and `tag-integrity` (tag-integrity.yml). The landmine is nonetheless still dormant, but for a reason the issue never gives: `release` and `tag-integrity` are both triggered by `push: tags: ['v*']`, so their runs carry the TAG as `head_branch` and are excluded by the `?branch=main` query parameter — not because they do not exist. A planner who "verifies" the issue's premise by listing workflow files will find three and may wrongly conclude the defect is already live; a planner who trusts the premise will not know the tag-trigger fact is the only thing holding it dormant. Either way the correct trigger to watch for is the first BRANCH-triggered second workflow on `main`, not the first second workflow.
  (b) **The issue's proposed one-line fix, applied verbatim, goes RED against the existing test suite.** `cmd/backstop/baseline_test.go:248` already exercises this function through a fake `gh` whose fixture payload emits `"name":"ci"` — LOWERCASE — while `.github/workflows/ci.yml` declares `name: CI`, uppercase. So adding `run.Name == "CI"` breaks the currently-passing assertion at `baseline_test.go:267` (`runID != 42`). The fixture is also, by that same mismatch, demonstrably not captured from real GitHub API output, which this repo's fixtures-from-real-output convention requires — so the fixture needs correcting on its own merits, and that correction must land in the SAME change as any name filter, or the fix is red on arrival.
  (c) **"Redundant" must not be read as "delete it."** The issue calls the client-side `HeadBranch == "main"` check redundant given the server-side `branch=main` parameter. That client-side check is the only in-process guard should the server-side filter's treatment of tag-triggered runs ever differ from the assumption in (a); removing it as dead code would convert a documented assumption into an undocumented one. Keep it.
  The issue itself declines to prescribe the fix shape, correctly, offering a literal-name match versus a "select the run whose artifacts actually include `backstop-baseline-v1`" scan; the latter has no workflow-name coupling and so is the option that does NOT enlarge ISSUE-120's surface. That trade-off is a founder/planner call, not settled here. Sequencing is clear: PLAN-ISSUE-176 is `status: draft` and in flight in this shared tree, but its file scope is `.github/workflows/ci.yml`, `cmd/backstop/bun_ratchet_flip_test.go` and `cmd/backstop/baseline_prerequisite_test.go` — it does NOT hold `cmd/backstop/baseline.go` or `cmd/backstop/baseline_test.go`, so this work does not collide with it.
