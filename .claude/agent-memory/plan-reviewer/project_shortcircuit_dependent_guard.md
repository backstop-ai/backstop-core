---
name: shortcircuit-dependent-guard
description: An existing "already covers this" regression test can pass despite the live defect because an EARLIER branch in the predicate short-circuits; measure the guard's actual verdict at HEAD before accepting a plan's or an issue's claim that it suffices
metadata:
  type: project
---

When a plan (or the issue it implements) says "the existing mandated test already
guards this, no new coverage needed", measure that test's ACTUAL verdict at HEAD
before believing it. A guard routinely passes for the RIGHT ANSWER via the WRONG
REASON when the predicate it exercises has an earlier branch that short-circuits.

**Why:** ISSUE-130 (2026-08-16). `isReleasedModuleVersion` rejects on
`strings.Contains(v, "+")` BEFORE reaching `pseudoVersionSuffix.MatchString`. Any
uncommitted OR UNTRACKED file makes Go stamp `+dirty` on the recorded module
version, so `TestVersion_LdflagsInjectionReachesBuiltCLI` — whose own failure
message names the exact defect — passed in every working tree while the defect was
live. Only a pristine checkout reached the regex. ISSUE-130's own "Direction"
section asserted that test was a sufficient regression guard; it was not.

**How to apply:**
- Build/run the thing and read the real value, don't reason about it. Here:
  `go build -o /tmp/probe ./cmd/backstop && go version -m /tmp/probe | grep mod`
  showed `v0.1.3-0.<ts>-<hash>+dirty` and the CLI reporting `dev` — passing test,
  live defect, in one measurement.
- The fix shape to demand: a pure unit test driving the inner predicate directly
  with inputs that CANNOT trip the earlier branch, PLUS an assertion inside the
  integration test that neutralizes the short-circuit (strip the `+...` metadata,
  then assert the predicate still rejects). Either alone leaves a hole.
- Tree state is an input to the falsification pass. A plan telling the implementer
  to run `git status --porcelain` and report the observed state rather than assume
  it is doing this right.
- Untracked files count as dirty for Go's `vcs.modified` — an uncommitted plan
  artifact alone flips it.

Inverse of [[project_e2e_fixture_already_loud_at_head]]: there the acceptance test
was already loud at HEAD; here the guard was silently green at HEAD.
