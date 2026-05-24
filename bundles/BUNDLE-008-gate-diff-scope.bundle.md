---
title: "Gate Diff Scoping — Changed-Files-Only Default for Usable Feedback"
number: BUNDLE-008
created: "2026-04-24"
schema_version: bundle/v2

bundle:
  name: gate-diff-scope
  version: "0.3.0"
  created: "2026-04-24"
  updated: "2026-04-27"
  category: feature

status:
  maturity: exploring

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

`pkg/check/scope.go` implements `ScopeModeDiff` with a 4-step git
merge-base cascade:

1. Try `origin/main` merge-base
2. Try `origin/master` merge-base
3. Fall back to local staged + unstaged changes
4. Fall back to full codebase scan (with warning)

`backstop code check` (no flags) already defaults to diff mode. The
gate just ignores this and hardcodes `ScopeModeAll`.

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

## References

- BUNDLE-007: Baseline (complementary noise reduction)
- SPEC-010: Gate (REQ-012 "no scope flags" — to be superseded)
- pkg/check/scope.go: Existing ScopeModeDiff with 4-step cascade
- cmd/backstop/gate.go: Current gate hardcoding ScopeModeAll
