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
---

## Description

Implement gate step 7 (baseline comparison) and the CI baseline generation workflow. CI runs `backstop gate` post-merge and publishes the violation set as an immutable baseline artifact. Locally, `backstop gate` auto-pulls the latest baseline with TTL-based caching and reports differentials ("3 new violations beyond baseline") instead of absolute counts.

Includes: `backstop baseline pull` command, `.backstop/baseline.json` caching, TTL logic (default 15 minutes), GitHub Actions artifact publishing, structural diff algorithm for violation identity. Per ISSUE-086, the CI `baseline` job must install packs before generating, so the published artifact reflects pack-engine findings rather than a structurally empty engine set.

Depends on DIR-001 (release workflow — CI must exist to generate baselines).

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
