---
title: "Baseline — CI-Generated Immutable Violation Reference"
number: BUNDLE-007
created: "2026-04-19"
schema_version: bundle/v2

bundle:
  name: baseline
  version: "0.3.0"
  created: "2026-04-19"
  updated: "2026-04-24"
  category: feature

status:
  maturity: ready

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

  success_criteria:
    - Gate reports differential violations (new vs baseline), not absolute counts
    - Agents can determine whether their changes introduced regressions
    - Baseline is CI-generated and immutable — no local generation or editing
    - Baseline ratchet ensures violation counts can only decrease over time
    - First-run experience is non-hostile — missing baseline skips comparison with warning
    - Local caching with configurable TTL avoids excessive network calls
    - Pack upgrades produce new baseline violations that are chipped away, not blocked

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

  assumptions:
    - GitHub Actions is available as the CI provider
    - CI runs on every merge to main (post-merge trigger)
    - Network is available for baseline pull (offline fallback uses stale cache with warning)
    - GitHub Actions artifact storage provides sufficient retention (90-day default)
    - Gate emits violation data with enough structure for content-hash-based identity

requirements:
  - id: REQ-001
    text: >
      CI must run backstop gate after every merge to main and publish
      the full violation set as an immutable GitHub Actions artifact.
      The artifact includes git SHA, timestamp, backstop version, and
      per-step violation counts with rule IDs.
  - id: REQ-002
    text: >
      backstop gate must cache the baseline locally at
      .backstop/baseline.json (gitignored) with TTL-based freshness.
      If the cached baseline is within TTL, use it without network
      access. If expired, check GitHub for a newer artifact. If
      offline, use stale cache with a warning.
  - id: REQ-003
    text: >
      Gate step 7 (baseline comparison) must compute a differential
      between the current violation set and the cached baseline.
      Violations are matched by rule + file + content-hash of the
      violating region. New violations (not in baseline) are
      regressions. Violations in the baseline but not in current are
      progress. The gate reports the differential, not absolute counts.
  - id: REQ-004
    text: >
      Baselines must ratchet: each post-merge baseline can only have
      equal or fewer violations than the previous one. If a PR adds
      net-new violations, the PR gate fails. If a PR fixes violations,
      the post-merge baseline reflects the improvement. There is no
      mechanism to raise the baseline without a corresponding code change.
  - id: REQ-005
    text: >
      backstop baseline pull must fetch the latest baseline artifact
      from GitHub Actions, cache it locally at .backstop/baseline.json,
      and bypass TTL. This command is for explicit use at session start
      or after a known merge.
  - id: REQ-006
    text: >
      When no baseline exists (first run, no CI runs yet), gate step 7
      must skip baseline comparison and emit a warning indicating that
      CI needs to run first. The gate does not treat all violations as
      new and does not auto-generate a local baseline.
  - id: REQ-007
    text: >
      The baseline artifact must be a JSON file containing: full
      violation set from all gate steps, git SHA of the merge commit,
      timestamp, backstop version, per-step violation counts, and
      per-violation identity (rule + file + content-hash of violating
      region). The schema must be versioned.
  - id: REQ-008
    text: >
      When a pack version upgrade introduces new rules, the post-merge
      baseline must capture all new violations from those rules. New
      code is enforced immediately; existing violations enter the
      baseline and are remediated over time via the remediation bundle
      (BUNDLE-004 DD-42).
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
- Per-violation identity: rule + file + content-hash of violating region

This is enough to do a structural diff: same rule + same file + same
content-hash = pre-existing. New rule or new file or different
content-hash = regression.

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

## Draft Requirements

Requirements are formally captured in the frontmatter `requirements`
block (REQ-001 through REQ-008). Summary:

- **REQ-001**: CI post-merge baseline generation via GitHub Actions
- **REQ-002**: Local baseline caching with TTL at .backstop/baseline.json
- **REQ-003**: Differential violation reporting in gate step 7 using
  rule + file + content-hash identity
- **REQ-004**: Baseline ratchet — violation counts can only decrease
- **REQ-005**: `backstop baseline pull` command for explicit fetch
- **REQ-006**: First-run bootstrap skips comparison with warning
- **REQ-007**: Baseline artifact JSON schema (versioned)
- **REQ-008**: Pack upgrade interaction — new rules captured in baseline

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
  timestamp, backstop version, per-step violation counts with rule IDs,
  and per-violation content-hash identity. This enables structural
  diffing: same rule + same file + same content-hash = pre-existing.

- **DD-8:** Baseline artifact storage uses GitHub Actions artifacts
  for v1. Free, simple, 90-day retention is sufficient. Permanent
  storage (release assets, S3) can be added later if needed.
  Rationale: avoids infrastructure requirements; 90-day retention
  exceeds practical need since baselines are replaced on every merge.

- **DD-9:** Violation identity for diffing uses rule + file +
  content-hash of the violating region. More stable than line numbers
  (which shift on any edit). Requires the gate to emit richer violation
  data including the violating code region content. Rationale: line-based
  identity is too fragile for real-world use where edits above a
  violation shift its line number without changing the violation itself.

- **DD-10:** Main-only baselines for v1. All branches compare against
  main's baseline. Long-lived feature branches may accumulate noise but
  the simplicity tradeoff is worth it. Branch-specific baselines can be
  added in a future version if demand warrants. Rationale: branch
  baselines add significant complexity (storage, selection logic,
  staleness) for an edge case.

- **DD-11:** Baseline TTL is configurable in backstop.yml under
  `enforcement.baseline_ttl`, default 15 minutes. Rationale: teams
  merging frequently need shorter TTL; solo developers can use longer.
  Configurable with a sane default avoids one-size-fits-all friction.

- **DD-12:** First-run bootstrap skips baseline comparison with a
  warning. The gate does not treat all violations as new (too harsh)
  and does not auto-generate a local baseline (defeats the CI-only
  model). Rationale: permissive first-run avoids blocking adoption
  while making clear that CI needs to run first.

- **DD-13:** Baseline proceeds independently of the ledger (gate
  step 9). No dependency between baseline generation and ledger
  entries. Integration can come later. Rationale: the ledger is also
  deferred; coupling two deferred features creates unnecessary
  blocking. Each can ship independently.

## Resolved Design Questions

- **OQ-1 (storage):** Resolved as GitHub Actions artifacts for v1.
  Free, simple, 90-day retention is sufficient since baselines are
  replaced on every merge. Permanent storage deferred. See DD-8.

- **OQ-2 (violation identity):** Resolved as rule + file +
  content-hash of the violating region. More stable than line numbers
  which shift on any edit. Requires richer violation data from the
  gate. See DD-9.

- **OQ-3 (branch baselines):** Resolved as main-only for v1. All
  branches compare against main's baseline. Simpler model; branch
  baselines deferred. See DD-10.

- **OQ-4 (TTL configurability):** Resolved as configurable in
  backstop.yml under `enforcement.baseline_ttl`, default 15 minutes.
  See DD-11.

- **OQ-5 (first-run bootstrap):** Resolved as skip baseline
  comparison with warning. Document that CI needs to run first. No
  local baseline generation. See DD-12.

- **OQ-6 (baseline and ledger):** Resolved as independent. Baseline
  proceeds without waiting for ledger. Can integrate later. See DD-13.

## Spec Seeds

- **Baseline artifact schema** — the JSON structure for the cached
  baseline file, versioned schema, content-hash identity format.
  Covers REQ-007. Implement first as foundation for other specs.

- **CI workflow for baseline generation** — GitHub Actions workflow
  that runs gate post-merge and publishes the artifact. GitHub Actions
  artifact API integration. Covers REQ-001, REQ-008.

- **`backstop baseline pull`** — fetch latest from CI, cache locally,
  bypass TTL. TTL configuration from backstop.yml. Offline fallback.
  Covers REQ-002, REQ-005.

- **Gate step 7 implementation** — baseline comparison logic,
  content-hash structural diffing, differential reporting, first-run
  bootstrap behavior. Covers REQ-003, REQ-004, REQ-006.

## Notes / Ideas

- The baseline is the mechanism that makes pack upgrades and onboarding
  non-hostile. Without it, every new rule is a wall of violations.
  With it, new rules produce a differential signal that can be chipped
  away over time.
- The 15-minute TTL is a guess. Real-world usage will determine the
  right value. Teams merging frequently need shorter TTL; solo
  developers can use longer.
- Content-hash identity requires the gate to emit the violating code
  region, not just file + line. This is a prerequisite for baseline
  diffing and may require changes to existing gate step outputs.

## Version History

- 0.1.0 (2026-04-19): Initial bundle at exploring. Captured the
  CI-generated baseline model, local caching with TTL, differential
  gate reporting, ratchet semantics, pack upgrade interaction. 7 DDs,
  6 OQs, 4 spec seeds. Motivated by real-world pain from local
  baseline generation at the founder's day-job project.

- 0.2.0 (2026-04-24): Advanced to defined. Resolved all 6 OQs:
  GitHub Actions artifacts for storage (OQ-1), content-hash violation
  identity (OQ-2), main-only baselines (OQ-3), configurable TTL in
  backstop.yml (OQ-4), skip-with-warning bootstrap (OQ-5), independent
  of ledger (OQ-6). Added 6 new DDs (DD-8 through DD-13) from OQ
  resolutions. Added formal requirements REQ-001 through REQ-008.
  Added Draft Requirements section. Moved OQs to Resolved Design
  Questions section.

- 0.3.0 (2026-04-24): Advanced to ready. Added success_criteria and
  assumptions to frontmatter. Refined spec seeds with requirement
  traceability and implementation ordering. Updated baseline format
  notes to reflect content-hash identity decision.

## References

- BUNDLE-003: Onboarding experience (baseline as first capture)
- BUNDLE-004: Pack distribution (DD-42 remediation bundles on upgrade)
- SPEC-010: Gate (step 7 baseline currently deferred)
- ADR-0001: Agent-first (baseline consumed by agents, not humans)
