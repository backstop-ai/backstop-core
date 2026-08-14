---
title: "Gate Diff Scoping — Changed-Files-Only Default for Usable Feedback"
number: BUNDLE-008
created: "2026-04-24"
schema_version: bundle/v2

bundle:
  name: gate-diff-scope
  version: "1.0.0"
  created: "2026-04-24"
  updated: "2026-08-14"
  category: feature

status:
  maturity: delivered
  note: >
    DELIVERED 2026-08-14 via SPEC-018 (`status: implemented`) and its closed-out plan.
    All three spec seeds landed: gate scope infrastructure (`--all` / `--file` flags,
    changed-file computation, scope threading), per-step diff filtering, and the
    SPEC-010 REQ-012 supersession (recorded in SPEC-018's `supersedes:` block). At HEAD
    the gate's default IS diff mode (`cmd/backstop/gate.go:101`,
    `scopeMode := gate.GateScopeModeDiff`), scoping runs through `pkg/gate/scope.go`
    (`GateScope` / `ComputeGateScope` / `ComputeGateScopeWithBase`), and the diff-mode
    scope summary is emitted by `pkg/gate/output.go` in both its cases — the
    changed-file count at `:44` and the empty-set line at `:42`. All 6 OQs were resolved before
    delivery and remain correct against the shipped design. `requirements[]` below was
    authored at delivery time (v1.0.0) from the bundle's existing DDs and resolved OQs —
    this bundle predates the requirements-array convention — so the delivered corpus has
    a real top-of-chain to trace against. See also the two dated prose corrections in the
    body, where the pre-delivery text described state that no longer holds at HEAD.

problem:
  summary: >
    The backstop gate command runs all verification steps against the
    entire project every time, producing 500+ violations. Most of these
    are pre-existing noise from specs covering dead or unimplemented
    features. Agents cannot distinguish violations they caused from
    violations that already existed, making the gate unusable as a
    feedback signal during implementation.

  user_story: >
    As an implementer agent, I run backstop gate after making changes
    and see only violations relevant to the files I touched. Instead of
    500+ violations, I see 0-5 that I can actually fix.

solution:
  approach: >
    Gate defaults to diff mode using the existing ScopeModeDiff
    infrastructure from pkg/check/scope.go (4-step git merge-base
    cascade). Changed files are computed once at gate start and each
    step runs its checks against that file set. The gate answers "are
    these files clean?" — not "are these specs satisfied?" Spec-level
    verification (is the implementation complete relative to the plan?)
    is a separate concern. The --all flag restores current full-sweep
    behavior, and --file accepts an explicit file list as a manual
    override.

requirements:
  - id: REQ-001
    version: "1.0.0"
    text: >
      `backstop gate` with no scope flags must default to DIFF mode: it runs
      against the set of files changed relative to the git merge-base rather
      than the whole project. The changed-file set must include BOTH tracked
      modifications (vs the merge-base) AND untracked files, so a newly added
      file is never silently skipped. (DD-1.)
  - id: REQ-002
    version: "1.0.0"
    text: >
      `backstop gate --all` must run the full-project sweep — the behavior that
      was the unconditional default before this bundle. This is the mode for CI
      post-merge runs and comprehensive audits, and it is what preserves the
      original "gate is comprehensive" intent once diff mode becomes the
      default. (DD-1; OQ-1 — an explicit flag, never CI env-var detection.)
  - id: REQ-003
    version: "1.0.0"
    text: >
      `backstop gate --file <files...>` must accept an explicit file list and
      scope the file-scoped steps to exactly those files — a manual override for
      when the git diff does not reflect reality. `--file` and `--all` are
      mutually exclusive; supplying both is a config error, not a silent
      precedence rule. (DD-1; OQ-6.)
  - id: REQ-004
    version: "1.0.0"
    text: >
      The changed-file set must be computed ONCE at gate start and shared across
      every step. No step may independently re-derive it, so all steps see an
      identical file set and the gate performs no redundant git work. (DD-2.)
  - id: REQ-005
    version: "1.0.0"
    text: >
      In diff and file modes the gate must answer "are THESE FILES clean?", not
      "are these specs satisfied?": each file-scoped step (artifact validation,
      code check, test verification, substantiveness, coverage, contract
      signatures) runs against the shared changed-file set, with no
      spec-matching or `implementation.package` correlation logic. Artifact
      validation is strict file-change only — an artifact that did not change is
      not validated. Spec-level "is this implementation complete relative to the
      plan?" verification is explicitly NOT the gate's job. (DD-3; OQ-3, OQ-4.)
  - id: REQ-006
    version: "1.0.0"
    text: >
      Pack lock verification and pack validators must ALWAYS run, in every scope
      mode. They are structural checks on project shape rather than per-file
      checks, so scoping must never be able to skip them. (DD-4; OQ-2.)
  - id: REQ-007
    version: "1.0.0"
    text: >
      In diff mode the gate must print a scope summary before step results, so a
      reader can never mistake a narrow scope for a clean project. The summary
      must state the scope the run actually had, in BOTH cases: when files
      changed, how many it is running against and how to get a full sweep (e.g.
      "Gate running against 12 changed files (use --all for full sweep)"); and
      when the changed set is EMPTY, that no changed files were found and the
      scoped checks therefore had nothing to inspect. The empty case is the more
      dangerous of the two — an unexplained green over zero files is the exact
      "confusion about missing violations" this requirement exists to prevent —
      so it must be reported explicitly rather than rendered as a zero-count or
      omitted. No summary is required in `--all` mode. (DD-5; OQ-5.)
  - id: REQ-008
    version: "1.0.0"
    text: >
      SPEC-010 REQ-012 ("gate accepts no scope flags") must be formally
      SUPERSEDED rather than silently contradicted: the superseding spec records
      the supersession and states that REQ-012's original intent — the gate is
      comprehensive — is preserved through `--all`. (REQ-012 supersession.)
---

# Gate Diff Scoping

## Current Thinking

### The noise problem

The gate currently runs every step against the entire project. With
specs covering features at various stages of implementation, this
produces 500+ violations on every run. An agent implementing a single
feature sees violations from the standards compiler, pack distribution,
onboarding experience, and every other spec — none of which are
relevant to its work.

### The diff infrastructure already exists

~~`pkg/check/scope.go` implements `ScopeModeDiff` with a 4-step git
merge-base cascade:~~

1. Try `origin/main` merge-base
2. Try `origin/master` merge-base
3. Fall back to local staged + unstaged changes
4. Fall back to full codebase scan (with warning)

~~`backstop code check` (no flags) already defaults to diff mode. The
gate just ignores this and hardcodes `ScopeModeAll`.~~

**CORRECTION (2026-08-14, v1.0.0): the attribution above is wrong about
what actually shipped.** The pre-delivery text assumed the gate would
REUSE `pkg/check/scope.go`'s `ScopeModeDiff`. The delivered design
instead introduced a PARALLEL gate-side mechanism in `pkg/gate/scope.go`
— the `GateScope` value plus `ComputeGateScope` / `ComputeGateScopeWithBase`
— and that is what `cmd/backstop/gate.go` threads through the steps.
`pkg/check.ScopeModeDiff` still exists in the tree, but the gate does not
use it: the gate's only reference is to `gate.GateScopeModeDiff`
(`cmd/backstop/gate.go:101`). The four-step cascade shape described above
is a fair description of the SEMANTICS the gate scope provides; only the
"already exists, just reuse it" framing was wrong. Everything else in this
section stands.

### Simplified model: gate checks files, not specs

The gate's job in diff mode is straightforward: run checks against
the changed files. It doesn't need to figure out which specs are
relevant or match files to `implementation.package` paths. Each step
just operates on the changed file set:

- **Artifact validation**: validate changed artifact files (specs,
  bundles, plans that are in the diff)
- **Code check**: lint, build, test, semgrep against changed files
  (ScopeModeDiff already does this)
- **Test verification**: check mandated tests exist — scoped to test
  files in the changed set
- **Substantiveness**: check test quality — scoped to test files in
  the changed set
- **Coverage**: run coverage for changed packages
- **Contracts**: check signatures in changed files
- **Pack lock/validators**: always run (structural checks)

Spec-level verification ("is this implementation complete relative
to the plan?") is a separate command — something like
`backstop verify-implementation --plan <PLAN_ID>`. That's not the
gate's job.

### Three scope modes

1. **Diff mode** (default, no flags): changed files from git
   merge-base cascade
2. **File mode** (`--file a.go b.go ...`): explicit file list,
   manual override for when the diff is wrong
3. **All mode** (`--all`): full project sweep, for CI post-merge

**ADDITION (2026-08-14, v1.0.0): there is a FOURTH selector at HEAD.**
`--base REV` (`cmd/backstop/gate.go:54`) scopes the gate to files changed
since the merge-base with `REV`, plus untracked files. It postdates this
bundle and exists for CI: a fresh checkout has a clean working tree, so
the bare diff default resolves to nothing and would check zero files —
`--base` takes the pull-request base sha or the push before-sha instead,
and an unresolvable `REV` is a config error rather than a silent full
sweep. The three explicit selectors (`--all`, `--file`, `--base`) are
pairwise mutually exclusive; diff mode is what a run gets when none is
passed.

### REQ-012 supersession

SPEC-010 REQ-012 states "gate accepts no scope flags." This would
need to be superseded — gate now accepts `--all` and `--file`. The
original intent (gate is comprehensive) is preserved via `--all`.

## Draft Design Decisions

- **DD-1:** Gate defaults to diff mode (changed files only). `--all`
  flag enables full sweep. `--file` accepts an explicit file list.
  Changed files must include both tracked modifications (vs merge-base)
  and untracked files, so new files are never silently skipped.
  Rationale: diff mode produces actionable feedback (0-5 violations
  vs 500+). Matches existing `backstop code check` behavior. `--file`
  provides a manual override when the git diff doesn't reflect reality.

- **DD-2:** Changed files computed once at gate start, shared across
  all steps. Rationale: avoids redundant git operations and ensures
  consistent file set across steps.

- **DD-3:** Gate checks files, not specs. Each step runs its checks
  against the changed file set. The gate answers "are these files
  clean?" Spec-level verification is a separate concern. Rationale:
  simpler model, no spec-matching logic, language-agnostic.

- **DD-4:** Pack lock verification and pack validators always run in
  all modes. Rationale: structural checks on project shape, not
  per-file checks. Fast and always relevant.

- **DD-5:** Gate outputs scope summary in diff mode, e.g., "Gate
  running against 12 changed files (use --all for full sweep)."
  Rationale: helps agents understand the scope and avoids confusion
  about missing violations.

## Resolved Design Questions

- **OQ-1 (CI behavior):** `--all` is an explicit flag, no CI env-var
  detection. CI runs both modes: diff on PR checks (blocking), full
  sweep post-merge (advisory + baseline). Full advisory pipeline is
  BUNDLE-007 territory.

- **OQ-2 (pack validator scoping):** Always run all pack validators.
  Structural checks on project shape, not per-file.

- **OQ-3 (specs without implementation.package):** N/A — gate doesn't
  do spec matching. It checks files. Artifact validation fires for
  changed artifact files regardless of their content.

- **OQ-4 (artifact validation scope):** Strict file-change only.
  Artifact validation checks document structure. If the file didn't
  change, it doesn't get validated.

- **OQ-5 (output messaging):** Yes. Scope summary in diff mode output.

- **OQ-6 (--file flag):** Gate supports `--file` accepting a list of
  files. Manual override for when git diff is wrong. Mutually
  exclusive with `--all`.

## Out of Scope

- Baseline comparison (BUNDLE-007) — complementary but separate.

- Spec-level implementation verification — "is this implementation
  complete relative to the plan?" is a separate command, not the gate.

- Waiver/suppression mechanism — different concern from file scoping.

## Spec Seeds

- Gate scope infrastructure (--all flag, --file flag, changed-file
  computation, scope context threading through steps)

- Per-step diff filtering (each step's logic for operating on the
  changed file set instead of the full project)

- SPEC-010 REQ-012 update (spec evolution to allow scope flags)

## Notes / Ideas

- This is the quick win that makes the gate usable NOW. Baseline
  (BUNDLE-007) is the long-term CI solution. Together they're two
  layers of noise reduction.

- The `--file` flag accepting a list makes the gate composable with
  external tools — e.g., a GitHub Action could compute the PR's
  changed files and pass them to `backstop gate --file`.

## Version History

- 0.1.0 (2026-04-24): Initial bundle at exploring. Captured the
  diff scoping problem, existing ScopeModeDiff infrastructure, 3
  DDs, 6 OQs, 3 spec seeds.

- 0.3.0 (2026-04-27): Confirmed scope boundaries and readiness for promotion. Bundle scope is pure file selection - input to gate pipeline, not violation interpretation. Clean separation from baseline work (BUNDLE-007). Ready to advance to next maturity level.

- 1.0.0 (2026-08-14): **DELIVERED.** Disposition change, founder-approved
  after a review that verified this bundle against real code at HEAD.

  - Maturity `exploring` → `delivered`. The substance shipped via SPEC-018
    (`status: implemented`) and its closed-out plan; this bundle had simply
    never been dispositioned. Deliberately NOT a retired-terminal status
    (`replaced`/`canceled`/`deprecated`) — those mean abandoned or superseded,
    which would be factually wrong. The work landed.
  - All THREE spec seeds landed: gate scope infrastructure (`--all`, `--file`,
    changed-file computation, scope threading through the steps), per-step diff
    filtering, and the SPEC-010 REQ-012 supersession.
  - All SIX OQs (OQ-1…OQ-6) were re-checked against the shipped design and
    remain correctly resolved — no OQ was invalidated by delivery.
  - `requirements[]` AUTHORED FRESH at 1.0.0 (REQ-001…REQ-008), derived from
    the existing DD-1…DD-5, the resolved OQs, and the REQ-012 supersession —
    no new scope invented. This bundle predates the requirements-array
    convention, and `delivered` maturity mechanically requires a non-empty
    array (`pkg/validate/bundle.go:536` — `delivered` is not exempted the way
    the retired-terminal statuses are). The version bump to 1.0.0 reflects a
    requirements array authored at 1.0.0, not an evolution of a prior one.
  - TWO prose corrections, applied under the dated-correction convention
    (old text struck, correction appended) because a `delivered` bundle must
    not record pre-fix state as current: (a) the References line
    "cmd/backstop/gate.go: Current gate hardcoding ScopeModeAll" is FALSE at
    HEAD — `gate.go:101` is `scopeMode := gate.GateScopeModeDiff`; (b) "The
    diff infrastructure already exists" attributed the implementation to
    `pkg/check/scope.go`'s `ScopeModeDiff`, but the shipped design uses a
    parallel `pkg/gate` mechanism (`GateScope` / `ComputeGateScopeWithBase`)
    and the gate does not consume `pkg/check.ScopeModeDiff` at all.
  - ONE addition: the "Three scope modes" section gained a fourth selector,
    `--base REV` (`cmd/backstop/gate.go:54`), a CI-fresh-checkout selector
    that postdates the bundle.
  - REQ-007 text AMENDED, still at 1.0.0, before this version was ever
    committed. The first drafting promoted DD-5's illustrative string into the
    obligation ("naming how many changed files it is running against"), which
    silently narrowed DD-5 — DD-5 says "e.g.", so the changed-file sentence is
    an EXAMPLE and the obligation is "gate outputs scope summary in diff mode."
    An empty changed set is still diff mode, and the shipped code emits its
    empty-diff line from the same branch as the count line
    (`pkg/gate/output.go:42` vs `:44`), so it is the scope summary for the zero
    case rather than a separate output. Surfaced by the spec-author
    back-annotating SPEC-018, who correctly found REQ-009's output obligation
    unanchored but placed the gap one level too low: the defect was this
    bundle's REQ-007 text, not a missing REQ. NOT version-bumped, deliberately —
    REQ-007@1.0.0 had no consumer outside the same uncommitted working tree
    (verified: no committed artifact at HEAD pins any `gate-diff-scope:REQ-*`),
    and a bump would have made the just-authored SPEC-018 pin stale, whose
    remedy on a success-terminal bundle is `new_spec`
    (`pkg/gate/requirement_traceability.go:142-144`) — blocking ceremony for a
    defect minutes old with no history to preserve.
  - FOLLOW-UP, out of this agent's lane: `BACKLOG.yml` still re-lists
    BUNDLE-008 in its `bundles:` section (added earlier on 2026-08-14 during
    an unrelated pass). A `delivered` bundle is not an open backlog slot, so
    that entry should be removed — flagged here for routing, not edited here.

## References

- BUNDLE-007: Baseline (complementary noise reduction)
- SPEC-010: Gate (REQ-012 "no scope flags" — SUPERSEDED by SPEC-018,
  recorded in that spec's `supersedes:` block)
- SPEC-018: Gate diff scoping — the spec that delivered this bundle
  (`status: implemented`)
- pkg/gate/scope.go: The DELIVERED gate scope mechanism — `GateScope`,
  `ComputeGateScope`, `ComputeGateScopeWithBase`
- pkg/check/scope.go: `ScopeModeDiff` with a 4-step cascade — present in
  the tree but NOT the mechanism the gate uses (see the dated correction
  under "The diff infrastructure already exists")
- ~~cmd/backstop/gate.go: Current gate hardcoding ScopeModeAll~~
  **CORRECTION (2026-08-14, v1.0.0): false at HEAD.** `cmd/backstop/gate.go:101`
  reads `scopeMode := gate.GateScopeModeDiff` — diff IS the default, and
  `--all` / `--file` / `--base` select otherwise. The struck line described
  the pre-delivery state and must not stand as "current" in a `delivered`
  bundle.
- pkg/gate/output.go: The delivered diff-mode scope summary — one obligation,
  two arms of the same switch branch: the empty-set line at `:42`, the
  changed-file count at `:44` (REQ-007 covers both)
