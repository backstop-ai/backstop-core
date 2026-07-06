---
title: "Gate Flags Artifact Status Reality Drift"
schema_version: issue/v1

issue:
  id: ISSUE-042
  title: "Gate Flags Artifact Status Reality Drift"
  type: enhancement
  status: open
  created: "2026-07-06"

complexity:
  scope: cross-cutting
  uncertainty: exploratory
  risk: moderate
---

# Gate Flags Artifact Status Reality Drift

## Problem

Nothing today enforces that an artifact's `status` field matches observable
reality. The status state machines are defined in schema (issue: open → ready →
in-progress/blocked → closed; spec: draft → active → ... → implemented; directive:
queued → active → done) but transitions between those states are manual and
unverified — an author edits `status:` by hand (or forgets to) and nothing checks
it against what the codebase and test suite actually show.

This drifts in **both directions**, and both are real:

1. **Delivered but not closed.** An issue/spec/plan sits at a non-terminal status
   (`open`, `ready`, `in-progress`; `draft`, `active`) while every mandated test
   its claims name already exists and passes. The work is done; the artifact
   still claims it isn't.
2. **Done but not real.** An artifact is marked success-terminal (issue `closed`;
   spec `implemented`; directive `done`) but its mandated tests are absent or
   failing — a regression, a hand-edit that outran reality, or a close that
   happened before verification actually ran. This is the serious direction: a
   status field asserting work is proven when it isn't is a broken promise,
   structurally identical to the declared-but-absent capability class the gate
   already blocks for traceability dimensions (`pkg/gate/traceability_polarity.go`,
   `ClassDeclaredIntentUnmet`).

### Motivating evidence

At the 2026-07-06 session close, four issues — ISSUE-018, ISSUE-034, ISSUE-035,
ISSUE-036 — had all shipped on `main` in the squash commit `d5efd5b` ("feat:
eradicate `backstop code check` + un-vacuum gate dimensions") but still read
`status: open` in their frontmatter. Their mandated tests were present and
green; nothing had told anyone the status was stale. The mismatch was only
caught and reconciled by hand, artifact by artifact, because a human happened to
notice while closing out the session. There is no mechanical check that would
have caught it, and no check that would catch the inverse (a `closed` issue
whose mandated tests later regress or get deleted).

### Why it matters

This is exactly the kind of silent drift the gate exists to make loud
(CLAUDE.md: "Loud ≠ blocking. Block defects + broken promises, warn-with-
guidance for un-adopted capability. The enemy is silent/vacuous green, not
passing."). A `status` field is itself a promise — "this is done," "this is
still being worked," "this is blocked" — and today the gate verifies every
*other* kind of promise an artifact makes (requirements have claims, claims
mandate tests, tests must pass — `test_verification`) except the status label
wrapping all of it. An artifact whose status has drifted from reality
undermines exactly the traceability discipline the gate otherwise enforces:
readers (humans and agents) orient off `status` first, and a stale status
either hides finished work from planning (direction 1) or launders unverified
work as proven (direction 2, the more dangerous one).

## Solution

Add a new gate dimension — a dogfood check, following the existing "dogfood
rules as packs" convention — that resolves each artifact's mandated tests (the
same resolution `test_verification` already performs: issues/specs carry
claims → `tests` lists via `ExtractMandatedTests`; a plan's tasks carry their
own mandated test names) and cross-checks test existence + pass/fail against
the artifact's declared `status`.

### Proposed polarity, mirroring the broken-promise model

This mirrors `pkg/gate/traceability_polarity.go`'s existing three-class model
(`ClassBrokenDeclared` / `ClassCapabilityAbsent` / `ClassDeclaredIntentUnmet`),
reusing its shape rather than inventing new vocabulary:

- **Delivered-but-not-closed → WARN.** Non-terminal status (issue `open`/
  `ready`/`in-progress`; spec `draft`/`active`; directive `queued`/`active`) +
  ALL mandated tests present and passing → warn: "looks delivered — close it /
  advance status." Non-blocking, guidance-only — the report surface carries the
  loudness, nothing red.
- **Done-but-not-real → BLOCK.** Success-terminal status (issue `closed`; spec
  `implemented`; directive `done`) + any mandated test absent or failing →
  block: "claimed done, isn't." This is the broken-promise direction and should
  reuse `ClassDeclaredIntentUnmet`'s exit-2 treatment, not a softer warn — a
  closed artifact asserting proof that doesn't exist is the same category of
  defect as a declared traceability dimension with no wired capability.

### Caveat: "delivered" is a heuristic, not a proof — keep this honest

Passing mandated tests is strong evidence of completion but not definitive
proof: a task can be partially done with its named tests green (the tests
prove the claims they were written against, not the whole of the artifact's
intent). That asymmetry is exactly why the two directions get different
severities here — WARN when the heuristic merely *suggests* done (false
positives just nag), BLOCK when a status actively vouches for verification
that observably isn't there (false negatives launder unproven work as proven).
This should stay an explicit design constraint on whatever plan implements
this, not something a planner is free to silently upgrade to a blocking check
in both directions.

### Open scoping question for the planner

Diff-scoped (only artifacts whose files changed in the current diff) vs. a
full-artifact-set sweep:

- **Diff-scoped** is cheap and fits the gate's existing diff-scope model
  (`pkg/gate` mostly reasons about changed files), but only catches drift on
  artifacts touched in the current change — it would NOT have caught the
  ISSUE-018/034/035/036 case above, since none of those issue files themselves
  were part of the diff that shipped the fix (the code changed; the issue
  frontmatter didn't).
- **Full sweep** (every issue/spec/plan/directive, every gate run) is what
  actually catches org-wide drift like the motivating case, but is heavier —
  it re-resolves mandated tests for every artifact in the repo on every gate
  run, not just the ones in scope. Needs a cost/frequency tradeoff (every gate
  run vs. a separate periodic/CI-triggered sweep).

This tradeoff is left open for the plan to resolve; the motivating evidence
above argues for at least *some* full-sweep capability, since diff-scoping
alone provably would have missed it.

### Where this likely lives

A new gate step alongside `test_verification` (`pkg/gate/step_testverify.go`)
and the traceability polarity classifier (`pkg/gate/traceability_polarity.go`),
consuming the same `ExtractMandatedTests` / `MandatedTest` resolution machinery
`test_verification` already uses rather than re-implementing mandated-test
discovery. It additionally needs each artifact's parsed `status` field, which
`ExtractMandatedTests` and its sibling `parseSpecFrontmatter` do not currently
surface for issues/directives (today's frontmatter parsing in
`pkg/gate/step_testverify.go` is spec-shaped) — extending that parsing to
issues/plans/directives is likely a prerequisite, not a detail.

### Relationship to ISSUE-043

Complementary, not overlapping: this issue (042) makes status/reality drift
*loud*; ISSUE-043 ("Reduce Issue Close Friction Trace Plan Claims") makes the
*fix* for that drift cheap once it's flagged. A planner should be free to
sequence either first, but the WARN-direction UX of 042 (telling an author
"this looks delivered, advance its status") is most useful if closing is
already low-friction — worth reading 043 before designing 042's warning
message.

## References

- `pkg/gate/traceability_polarity.go` — the existing broken-promise / fail-loud
  polarity model (`PolarityClass`, `ClassDeclaredIntentUnmet` in particular) this
  issue's BLOCK direction should mirror
- `pkg/gate/step_testverify.go` — `MandatedTest`, `ExtractMandatedTests`,
  `ResolveMandatedTestPaths` — the existing mandated-test resolution this
  proposal reuses rather than reimplementing
- `artifacts/issue/v1/schema.json`, `artifacts/spec/v1/schema.json`,
  `artifacts/directive/v1/schema.json` — the status enums this check reads
- ISSUE-043 — sibling issue reducing close friction; complementary, see
  "Relationship to ISSUE-043" above
- ISSUE-018, ISSUE-034, ISSUE-035, ISSUE-036 — the motivating evidence: all four
  shipped in `d5efd5b` while still reading `status: open`, caught only by hand
  at 2026-07-06 session close
- DIR-016 — parent directive (issue/plan lifecycle hardening)
- CLAUDE.md, "Enforcement philosophy" — "Loud ≠ blocking"; "Dogfood rules as
  packs"
