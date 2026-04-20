---
title: "Baseline — CI-Generated Immutable Violation Reference"
number: BUNDLE-007
created: "2026-04-19"
schema_version: bundle/v2

bundle:
  name: baseline
  version: "0.1.0"
  created: "2026-04-19"
  category: feature

status:
  maturity: exploring

problem:
  summary: >
    Gate step 7 (baseline comparison) is deferred. Without a baseline,
    every gate run reports ALL violations — pre-existing and new alike.
    Agents can't distinguish "violations I caused" from "violations that
    existed before I touched anything." This makes the gate noisy and
    unusable as a feedback signal during implementation. The baseline is
    the reference point that turns absolute violation counts into
    differential signals: "did you make it worse?" A key lesson from
    the founder's day-job project: letting developers generate baselines
    locally leads to gaming, inconsistency (different machines produce
    different results), merge conflicts on the baseline file, and
    baselines that hide rather than track violations. The baseline must
    be CI-generated, immutable, and distributed — never hand-edited.

  user_story: >
    As an implementer agent, I need to know whether my changes introduced
    new violations beyond what already exists on main. I pull the current
    baseline at the start of my session, run backstop gate after each
    implementation task, and the gate tells me "0 new violations" or
    "3 new violations beyond baseline — fix before proceeding." I never
    produce or edit the baseline — CI does that after every merge to main.

solution:
  approach: >
    Baseline is a CI post-merge artifact, not a committed file. After
    every merge to main, CI runs backstop gate and publishes the full
    violation set as an immutable baseline artifact. Locally, backstop
    gate pulls the latest baseline (with TTL-based caching) and compares
    the current violation set against it. The differential is the signal:
    new violations = fail, fixed violations = progress. The baseline file
    is cached at .backstop/baseline.json (gitignored) and never committed
    to the repo.
---

# Baseline

## Current Thinking

### The problem with local baselines

The founder's day-job project allowed developers to generate baselines
locally and commit them. This created multiple failure modes:

- **Inconsistency**: different machines, tooling versions, and OS
  differences produce different violation sets. Two developers running
  the same code get different baselines.
- **Gaming**: developers can regenerate the baseline after introducing
  violations, making them "pre-existing." The baseline becomes a way
  to hide regressions rather than track them.
- **Merge conflicts**: two PRs that both update the baseline file
  conflict. The baseline becomes a noisy diff in every PR.
- **Staleness**: committed baselines go stale the moment someone else
  merges. The local baseline doesn't reflect what's actually on main.

### The CI-generated model

The baseline is a CI output, not a CI input:

1. Developer merges PR to main
2. CI runs `backstop gate` on the merged code
3. CI publishes the violation set as an immutable baseline artifact
4. The baseline is the truth until the next merge

Developers and agents never produce the baseline. They only consume it.

### Local distribution and caching

Agents need the baseline locally to compare during implementation.
The flow:

- `backstop gate` checks for a fresh baseline as part of its run
- If cached baseline is stale (TTL expired, default ~15 minutes):
  check GitHub for a new baseline artifact
- If a newer baseline exists: download, cache at
  `.backstop/baseline.json`, use it
- If the cached baseline is fresh (within TTL): use cache, skip network
- If offline: use stale cache, warn
- If no cache exists: pull, cache, proceed (first run bootstraps)

The TTL prevents hammering GitHub on every `backstop code check` during
a tight implementation loop. The baseline only changes on merge to main,
not every minute.

### Explicit pull command

`backstop baseline pull` fetches the latest baseline from CI and caches
it locally, bypassing TTL. Useful at session start or when the agent
knows a merge just landed.

### The differential signal

With a baseline, the gate output changes from:

> "748 violations found" (unusable — is this my fault?)

to:

> "0 new violations beyond baseline" (I'm clean)

or:

> "3 new violations beyond baseline — fix before proceeding"

The agent knows exactly what it introduced. Pre-existing violations are
invisible. The ratchet works naturally: each merge produces a new
baseline. If a PR adds violations, the PR gate fails. If a PR fixes
violations, the post-merge baseline drops. It can only go down.

### Baseline format

The baseline artifact contains:
- The full violation set from the gate run (all 9 steps)
- The git SHA of the merge commit
- The timestamp
- The backstop version used
- Per-step violation counts and rule IDs

This is enough to do a structural diff: same rule + same file + same
line range = pre-existing. New rule or new file or new line range =
regression.

### Interaction with pack upgrades

When a pack version bumps and introduces new rules, the post-merge
baseline captures all the new violations. The remediation bundle
(DD-42 from BUNDLE-004) scopes the work. New code is enforced
immediately; existing violations are in the baseline and chipped away
over time.

### Interaction with `backstop init`

BUNDLE-003 describes the onboarding experience where `backstop init`
captures the initial state as "here's what we noticed." That initial
capture IS the first baseline. `backstop init` runs the gate, publishes
the result as the baseline, and from that point forward everything is
differential.

## Draft Design Decisions

- **DD-1:** Baseline is a CI post-merge artifact, not a committed file.
  CI runs `backstop gate` after every merge to main and publishes the
  violation set as an immutable artifact. No developer or agent produces
  or edits the baseline. Rationale: local baselines lead to gaming,
  inconsistency, merge conflicts, and staleness.

- **DD-2:** Baseline is cached locally at `.backstop/baseline.json`
  (gitignored). `backstop gate` auto-pulls the latest baseline with
  TTL-based caching (default 15 minutes). `backstop baseline pull`
  forces a fresh pull bypassing TTL. Offline mode uses stale cache
  with a warning.

- **DD-3:** Gate step 7 (baseline comparison) compares the current
  violation set against the cached baseline. New violations (same rule
  but new file/line, or new rule entirely) are regressions. Fixed
  violations (in baseline but not in current) are progress. The gate
  reports the differential, not the absolute count.

- **DD-4:** The baseline ratchet: baselines can only go down. Each
  merge to main produces a new baseline. If a PR adds violations, the
  PR gate fails (more than baseline). If a PR fixes violations, the
  post-merge baseline reflects the improvement. There is no mechanism
  to raise the baseline without a corresponding code change.

- **DD-5:** `backstop init` produces the first baseline. The initial
  gate run captures the starting violation set and publishes it as the
  baseline. From that point forward, all gate runs are differential.

- **DD-6:** Pack version upgrades interact with the baseline through
  the remediation bundle (BUNDLE-004 DD-42). New pack rules produce
  new violations that land in the post-merge baseline. Existing code
  is not blocked; new code is enforced immediately.

- **DD-7:** The baseline artifact includes: full violation set, git SHA,
  timestamp, backstop version, per-step violation counts with rule IDs.
  This enables structural diffing: same rule + same file + same line
  range = pre-existing.

## Open Questions

- **OQ-1: Baseline artifact storage.** Where does CI publish the
  baseline? GitHub Actions artifact storage (free, 90-day retention)?
  GitHub release asset (permanent)? S3 (requires infra)? Git tag with
  attached artifact? The storage choice affects TTL, availability, and
  cost. Lean: GitHub Actions artifacts for v1 — free, simple, good
  enough. Permanent storage can come later.

- **OQ-2: Violation identity for diffing.** How do you determine
  whether a violation in the current run is "the same" as one in the
  baseline? Same rule + same file + same line is fragile (line numbers
  shift on any edit). Same rule + same file + same code snippet is
  more robust but requires storing code context in the baseline.
  Same rule + same file only is too coarse (a file with 10 violations
  of the same rule — fixing one doesn't register). Needs a stable
  identity scheme.

- **OQ-3: Branch baselines.** Long-lived feature branches diverge from
  main. Should each branch have its own baseline? Or always compare
  against main's baseline? Branch baselines add complexity; main-only
  is simpler but may produce noise on large branches that accumulate
  violations before merging.

- **OQ-4: Baseline TTL configurability.** Default 15 minutes. Should
  this be configurable in backstop.yml? Too short = excessive network
  checks. Too long = stale baseline during active development when
  teammates are merging. Maybe configurable with a sane default.

- **OQ-5: First-run bootstrap.** If no baseline exists anywhere (new
  project, no CI runs yet), what does the gate do? Options: (a) treat
  everything as new (harsh), (b) skip baseline comparison with a
  warning (permissive), (c) auto-generate a local baseline on first
  run (defeats the CI-only model). Lean: (b) — skip with warning,
  document that CI needs to run first.

- **OQ-6: Baseline and the ledger.** Gate step 9 is the append-only
  ledger. Should each baseline be a ledger entry? The ledger would
  then contain the full history of violation counts over time — useful
  for trend analysis. But the ledger is also deferred. Should baseline
  block on ledger, or proceed independently?

## Spec Seeds

- **`backstop baseline pull`** — fetch latest from CI, cache locally,
  bypass TTL
- **Gate step 7 implementation** — baseline comparison logic,
  structural diffing, differential reporting
- **CI workflow for baseline generation** — GitHub Actions workflow
  that runs gate post-merge and publishes the artifact
- **Baseline format schema** — the JSON structure for the cached
  baseline file

## Notes / Ideas

- The baseline is the mechanism that makes pack upgrades and onboarding
  non-hostile. Without it, every new rule is a wall of violations.
  With it, new rules produce a differential signal that can be chipped
  away over time.
- The 15-minute TTL is a guess. Real-world usage will determine the
  right value. Teams merging frequently need shorter TTL; solo
  developers can use longer.
- The "same rule + same file + same line" identity scheme is the
  weakest part of the design. Line numbers shift on every edit. A
  content-hash-based identity (hash of the violating code region)
  would be more stable but requires the gate to emit richer violation
  data than it currently does.

## Version History

- 0.1.0 (2026-04-19): Initial bundle. Captured the CI-generated
  baseline model, local caching with TTL, differential gate reporting,
  ratchet semantics, pack upgrade interaction. 7 DDs, 6 OQs, 4 spec
  seeds. Motivated by real-world pain from local baseline generation
  at the founder's day-job project.

## References

- BUNDLE-003: Onboarding experience (baseline as first capture)
- BUNDLE-004: Pack distribution (DD-42 remediation bundles on upgrade)
- SPEC-010: Gate (step 7 baseline currently deferred)
- ADR-0001: Agent-first (baseline consumed by agents, not humans)
