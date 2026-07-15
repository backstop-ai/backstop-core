---
title: "Baseline — CI-Generated Immutable Violation Reference"
number: BUNDLE-007
created: "2026-04-19"
schema_version: bundle/v2

bundle:
  name: baseline
  version: "0.4.0"
  created: "2026-04-19"
  updated: "2026-05-25"
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
    - Baseline ratchet prevents developer-caused increases, with rule-set-change seeding as the v1 exception
    - First-run experience is non-hostile — missing baseline skips comparison with warning
    - Local caching with configurable TTL avoids excessive network calls
    - Pack and rule-set upgrades can seed existing-code violations that are chipped away, not blocked
    - Baseline implementation resolves SPEC-010's stale placeholder semantics

solution:
  approach: >
    Baseline is a CI post-merge artifact, not a committed file. After
    every merge to main, CI runs backstop gate, generates a portable JSON
    baseline artifact from the full raw violation set, and publishes it
    through GitHub Actions artifacts. Locally, backstop
    gate pulls the latest baseline (with TTL-based caching) and compares
    the current scoped violation set against it. The differential is the signal:
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
    version: "1.0.0"
    text: >
      CI must run backstop gate after every merge to main, generate a
      portable JSON baseline artifact containing the full violation set,
      and publish it through GitHub Actions artifacts for v1. The artifact
      includes git SHA, timestamp, backstop version, and per-step violation
      counts with rule IDs.
  - id: REQ-002
    version: "1.0.0"
    text: >
      backstop gate must cache the baseline locally at
      .backstop/baseline.json (gitignored) with TTL-based freshness.
      If the cached baseline is within TTL, use it without network
      access. If expired, check GitHub for a newer artifact. If
      offline, use stale cache with a warning.
  - id: REQ-003
    version: "1.0.0"
    text: >
      Gate step 7 (baseline comparison) must compute a differential
      between the current scoped violation set and the cached full-project
      baseline. Violations are matched by stable identity fields: rule,
      file, and content hash of the violating region. New scoped violations
      not in the baseline are regressions. Baseline violations absent from
      the current scoped run are progress only when they are inside the
      gate's current scope; diff and --file runs must not imply fixes for
      files they did not evaluate. The gate reports the differential, not
      absolute counts.
  - id: REQ-004
    version: "1.0.0"
    text: >
      Baselines must ratchet for developer changes: each post-merge
      baseline can only have equal or fewer violations than the previous
      one except for explicit rule-set changes. If a PR adds net-new
      violations in changed code, the PR gate fails. If a PR fixes
      violations, the post-merge baseline reflects the improvement.
  - id: REQ-005
    version: "1.0.0"
    text: >
      backstop baseline pull must fetch the latest baseline artifact
      from GitHub Actions, cache it locally at .backstop/baseline.json,
      and bypass TTL. This command is for explicit use at session start
      or after a known merge.
  - id: REQ-006
    version: "1.0.0"
    text: >
      When no baseline exists (first run, no CI runs yet), gate step 7
      must skip baseline comparison and emit a warning indicating that
      CI needs to run first. The gate does not treat all violations as
      new and does not auto-generate a local baseline.
  - id: REQ-007
    version: "1.0.0"
    text: >
      The baseline artifact must be a JSON file containing the full raw
      violation set from the gate's evaluating steps, git SHA of the merge commit,
      timestamp, backstop version, per-step violation counts, and
      per-violation identity (rule + file + content-hash of violating
      region). The schema must be versioned and portable so upload/publish
      remains separable from baseline generation.
  - id: REQ-008
    version: "1.0.0"
    text: >
      When a pack upgrade or other explicit rule-set change introduces new
      rules, the post-merge baseline may capture existing-code violations
      from those rules. New or changed code is enforced immediately;
      existing violations enter the baseline and are remediated over time
      via the remediation bundle (BUNDLE-004 DD-42).
  - id: REQ-009
    version: "1.0.0"
    text: >
      Baseline implementation must supersede SPEC-010 REQ-005. The v1
      cached baseline file is .backstop/baseline.json, not
      .backstop/baseline.yml, and violation matching uses stable identity
      fields rather than rule+file counts.
  - id: REQ-010
    version: "1.0.0"
    text: >
      Gate must support baseline comparison with access to prior gate step
      results. The implementation may use an accumulated-results-aware step,
      a gate-run context, or a post-processing phase, but step 7 must compare
      against violations produced by steps 1-6 without rerunning them.
  - id: REQ-011
    version: "1.0.0"
    text: >
      Gate violation output must add stable identity data required by
      baselines without breaking the existing gate/v1 contract. At minimum,
      violations need enough data to compute or carry a baseline identity:
      rule, file, optional line/range when available, and a content hash or
      source-region hash. Existing consumers must continue to work with the
      additive fields.
  - id: REQ-012
    version: "1.0.0"
    text: >
      CI baseline generation must run the full gate scope, equivalent to
      backstop gate --all. Local default diff mode and explicit --file mode
      compare only the current scoped violations against the full cached
      baseline.
  - id: REQ-013
    version: "1.0.0"
    text: >
      DIR-003/BUNDLE-007 v1 baseline comparison must ignore waivers only
      because the waiver subsystem has not been built yet. This v1 behavior
      must not define baseline comparison as permanently pre-waiver. When
      waivers are implemented, waived violations must participate in the
      baseline calculation semantics rather than being treated as permanent
      regressions.
  - id: REQ-014
    version: "1.0.0"
    text: >
      GitHub Actions baseline retrieval must define and gracefully handle repo
      resolution, missing origin remotes, missing authentication, missing
      artifacts, offline network failures, artifact naming, workflow/run
      selection, and branch filtering. Missing or unreachable remote baseline
      data must use a fresh or stale cached baseline when available; only when
      no fresh or stale cached baseline is available may the gate skip
      comparison with a warning rather than treating all violations as new.
  - id: REQ-015
    version: "1.0.0"
    text: >
      Pack or rule-set upgrades may explicitly seed new existing-code
      violations into the next full baseline. This is the only v1 exception
      to the ratchet rule: seeded violations are attributed to a rule-set
      change, while new violations in changed code still fail the PR gate.
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
3. CI generates a portable JSON baseline artifact from the violation set
4. CI publishes that JSON through GitHub Actions artifacts
5. The baseline is the truth until the next merge

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
- If no cache exists and a CI artifact exists: pull, cache, proceed
- If no cache exists and no CI artifact is available: skip comparison,
  warn, and continue without generating a local baseline

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
baseline. If a PR adds changed-code violations, the PR gate fails. If a
PR fixes violations, the post-merge baseline drops. Explicit rule-set
changes are the v1 exception that can seed existing-code violations into
the next full baseline.

### Baseline format

The baseline artifact contains:
- The full raw violation set from the gate's evaluating steps, before
  baseline comparison, waiver suppression, and ledger recording
- The git SHA of the merge commit
- The timestamp
- The backstop version used
- Per-step violation counts and rule IDs
- Per-violation identity: rule + file + content-hash of violating region

The JSON artifact is portable by design: baseline generation produces the
file, and upload/publish is a separable workflow concern. v1 publishes via
GitHub Actions artifacts, but later consumers can wire a different
distribution mechanism without changing baseline comparison semantics.

This is enough to do a structural diff: same rule + same file + same
content-hash = pre-existing. New rule or new file or different
content-hash = regression.

### Interaction with pack and rule-set upgrades

When a pack version bumps or another explicit rule-set change introduces
new rules, the next full baseline may seed existing-code violations from
those rules. The remediation bundle (DD-42 from BUNDLE-004) scopes the
work. New or changed code remains enforced immediately; existing
violations are in the baseline and chipped away over time. This is the
only v1 exception to the ratchet rule.

### Boundary with `backstop init`

BUNDLE-003 owns the onboarding experience. This bundle does not require
`backstop init` to generate, upload, or commit a baseline. For v1, missing
baseline behavior is intentionally simple: gate step 7 skips comparison
with a warning until CI has published the first baseline artifact.

## Draft Requirements

Requirements are formally captured in the frontmatter `requirements`
block (REQ-001 through REQ-015). Summary:

- **REQ-001**: CI post-merge baseline generation via GitHub Actions
  artifact publication, with generation producing portable JSON before upload
- **REQ-002**: Local baseline caching with TTL at .backstop/baseline.json
- **REQ-003**: Differential violation reporting in gate step 7 using
  rule + file + content-hash identity
- **REQ-004**: Baseline ratchet for developer changes, with rule-set-change exception
- **REQ-005**: `backstop baseline pull` command for explicit fetch
- **REQ-006**: First-run bootstrap skips comparison with warning
- **REQ-007**: Baseline artifact JSON schema (versioned and portable)
- **REQ-008**: Pack/rule-set upgrade interaction — existing-code violations can be seeded
- **REQ-009**: Supersede SPEC-010's stale baseline file/path and matching model
- **REQ-010**: Baseline step consumes accumulated gate results
- **REQ-011**: Add stable violation identity fields without breaking gate/v1
- **REQ-012**: CI uses `--all`; local diff/file runs compare scoped violations
- **REQ-013**: Ignore waivers in v1 only until the waiver subsystem exists;
  future waived violations participate in baseline semantics
- **REQ-014**: Define GitHub artifact retrieval, stale-cache fallback, and graceful no-baseline behavior
- **REQ-015**: Rule-set changes are the v1 ratchet exception

## Draft Design Decisions

- **DD-1:** Baseline is a CI post-merge artifact, not a committed file.
  CI runs `backstop gate` after every merge to main, generates a portable
  JSON violation set artifact, and publishes it through GitHub Actions
  artifacts. No developer or agent produces or edits the baseline. Rationale:
  local baselines lead to gaming, inconsistency, merge conflicts, and staleness.

- **DD-2:** Baseline is cached locally at `.backstop/baseline.json`
  (gitignored). `backstop gate` auto-pulls the latest baseline with
  TTL-based caching (default 15 minutes). `backstop baseline pull`
  forces a fresh pull bypassing TTL. Offline mode uses stale cache
  with a warning.

- **DD-3:** Gate step 7 (baseline comparison) compares the current scoped
  violation set against the cached full-project baseline. New scoped
  violations are regressions. Baseline violations absent from the current
  scoped run are progress only when they are inside the evaluated scope;
  diff and `--file` runs do not claim fixes for files outside their scope.
  The gate reports the differential, not the absolute count.

- **DD-4:** The baseline ratchet applies to developer changes. Each merge
  to main produces a new baseline. If a PR adds changed-code violations,
  the PR gate fails. If a PR fixes violations, the post-merge baseline
  reflects the improvement. Explicit rule-set changes are handled by DD-20.

- **DD-5:** `backstop init` integration is out of scope for v1 baseline.
  BUNDLE-003 may describe onboarding capture, but this bundle does not
  require local baseline generation or upload during init. Until CI
  publishes the first artifact, step 7 skips comparison with a warning.

- **DD-6:** Pack and rule-set upgrades interact with the baseline through
  the remediation bundle (BUNDLE-004 DD-42). New rules may seed
  existing-code violations into the next full baseline. Existing code is
  not blocked; new or changed code is enforced immediately.

- **DD-7:** The baseline artifact includes: full violation set, git SHA,
  timestamp, backstop version, per-step violation counts with rule IDs,
  and per-violation content-hash identity. The violation set is the raw
  output from evaluating gate steps before baseline comparison, waiver
  suppression, and ledger recording. This enables structural diffing:
  same rule + same file + same content-hash = pre-existing.

- **DD-8:** Baseline artifact storage uses GitHub Actions artifacts
  for v1. Free, simple, 90-day retention is sufficient. The CI workflow
  keeps generation modular: `backstop gate --all` produces a portable JSON
  artifact, and upload/publish is a separable step. Permanent storage
  backends (release assets, S3, GCS, JFrog, or another durable store) are
  out of scope for this bundle and can be added later if needed.
  Rationale: avoids infrastructure requirements; 90-day retention
  exceeds practical need since baselines are replaced on every merge, while
  separable publishing preserves a later migration path.

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
  step 9). Ledger state must not affect baseline comparison or baseline
  pass/fail semantics. Future ledger work may record or audit baseline
  events, but it must not gate the baseline differential. Rationale: the
  ledger is also deferred; coupling two deferred features creates
  unnecessary blocking, and auditability must not become hidden gating.

- **DD-14:** BUNDLE-007 supersedes SPEC-010 REQ-005 for baseline
  semantics. SPEC-010's deferred step text predates this bundle and
  names `.backstop/baseline.yml` with rule+file/count matching. The
  baseline implementation spec must use `.backstop/baseline.json` and
  stable identity matching. Rationale: SPEC-010 owned the placeholder;
  this bundle owns the real subsystem design.

- **DD-15:** Gate step 7 needs access to accumulated results from steps
  1-6. The implementation must not rerun earlier gate steps to compute
  the current violation set. A post-processing comparison phase or
  context-aware step is acceptable as long as the public step name remains
  `baseline_comparison`. Rationale: the current StepFunc signature is too
  narrow for a comparison step that consumes previous output.

- **DD-16:** Violation identity fields are additive to the gate JSON
  contract. Existing `rule`, `file`, `message`, `severity`, and
  `source_pack` fields remain valid. Baseline adds optional fields such as
  `line`, `end_line`, `fingerprint`, and/or `region_hash`; steps that lack
  exact source regions may derive a best-effort identity from the data they
  have. Rationale: baseline should not force a breaking gate/v2 schema.

- **DD-17:** CI generates baselines from `backstop gate --all`. Local
  default diff mode and explicit `--file` mode compare scoped current
  violations against the full baseline but only report fixed violations for
  the current scope. Rationale: SPEC-018 made diff mode the usable local
  default, but a baseline must represent the whole project.

- **DD-18:** DIR-003/BUNDLE-007 v1 baseline comparison ignores waivers only
  because the waiver subsystem has not been built yet. This is not a permanent
  pre-waiver baseline model. When waivers are implemented, waived violations
  must participate in baseline calculation semantics rather than being treated
  as permanent regressions. Rationale: coupling v1 baseline comparison to an
  unimplemented waiver subsystem would block DIR-003, but the temporary v1
  deferral must not lock in incorrect future waiver semantics.

- **DD-19:** GitHub Actions artifact lookup is best-effort locally. Missing
  remote configuration, missing authentication, no matching artifact, or
  network failure must skip baseline comparison with an actionable warning
  unless a fresh or stale cache is available. Rationale: local development
  must not fail closed just because CI metadata is unavailable.

- **DD-20:** Rule-set changes are the v1 ratchet exception. A pack upgrade
  or other explicit rule-set change may seed existing-code violations into
  the next full baseline, while changed-code regressions still fail the PR
  gate. Rationale: this preserves the baseline ratchet for developer
  changes without making rule adoption impossible.

## Resolved Design Questions

- **OQ-1 (storage):** Resolved as GitHub Actions artifacts for v1.
  Free, simple, 90-day retention is sufficient since baselines are
  replaced on every merge. Baseline generation still produces portable JSON
  before the separable publish step, so future storage backends can be wired
  without changing comparison semantics. Permanent storage deferred. See DD-8.

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
  proceeds without waiting for ledger, and ledger state must not affect
  baseline comparison or baseline pass/fail semantics. Later ledger work may
  record or audit baseline events, but must not gate the differential. See DD-13.

- **OQ-7 (SPEC-010 conflict):** Resolved by treating BUNDLE-007 as the
  authoritative baseline design and superseding SPEC-010 REQ-005 during the
  baseline spec. See DD-14.

- **OQ-8 (baseline step inputs):** Resolved by requiring step 7 to consume
  accumulated results from steps 1-6 without rerunning them. See DD-15.

- **OQ-9 (gate scope):** Resolved as full-scope CI generation with scoped
  local comparison. See DD-17.

- **OQ-10 (waivers):** Resolved as ignored by DIR-003/BUNDLE-007 v1 only
  because the waiver subsystem has not been built yet. This does not make
  baseline comparison permanently pre-waiver; future waived violations must
  participate in baseline calculation semantics. See DD-18.

- **OQ-11 (GitHub artifact failures):** Resolved as best-effort local lookup
  that uses any available fresh or stale cache first, and skips comparison with
  a warning only when remote metadata is unavailable and no cache exists. See DD-19.

## Spec Seeds

- **Baseline artifact schema** — the JSON structure for the cached
  baseline file, versioned schema, content-hash identity format, and portable
  generation output independent of the publish mechanism.
  Covers REQ-007 and REQ-011. Implement first as foundation for other specs.

- **CI workflow for baseline generation** — GitHub Actions workflow
  that runs gate post-merge, generates the portable JSON artifact, and
  publishes it through GitHub Actions artifacts as a separable step. GitHub
  Actions artifact API integration, including explicit rule-set-change seeding.
  Covers REQ-001, REQ-008, REQ-014, and REQ-015.

- **`backstop baseline pull`** — fetch latest from CI, cache locally,
  bypass TTL. TTL configuration from backstop.yml. Offline fallback.
  Covers REQ-002, REQ-005, and REQ-014.

- **Gate step 7 implementation** — baseline comparison logic,
  content-hash structural diffing, differential reporting, first-run
  bootstrap behavior, accumulated result access, and scoped comparison.
  Covers REQ-003, REQ-004, REQ-006, REQ-009, REQ-010, REQ-012, and REQ-013.

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
- Out-of-scope boundaries for v1 are intentional: no init-generated
  baselines, no ledger gating or dependency, no waiver-aware comparison until
  the waiver subsystem exists, no branch-specific baselines, and no permanent
  baseline storage backends such as release assets, S3, GCS, or JFrog. The CI
  workflow should still keep JSON generation separate from upload/publish so a
  later backend can be added without redefining baseline semantics.

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

- 0.4.0 (2026-05-25): Added DIR-003 implementation-readiness decisions.
  Resolved SPEC-010 baseline conflict, gate accumulated-result access,
  scoped comparison behavior after SPEC-018, waiver deferral, GitHub artifact
  failure behavior, additive violation identity fields, and the rule-set-change
  ratchet exception. Expanded requirements through REQ-015. Signoff clarified
  that ledger never gates baseline comparison/pass-fail and that v1 uses
  GitHub Actions artifacts while keeping JSON generation separable from publish.

## References

- BUNDLE-003: Onboarding experience (init integration deferred beyond v1)
- BUNDLE-004: Pack distribution (DD-42 remediation bundles on upgrade)
- SPEC-010: Gate (step 7 baseline currently deferred)
- ADR-0001: Agent-first (baseline consumed by agents, not humans)
