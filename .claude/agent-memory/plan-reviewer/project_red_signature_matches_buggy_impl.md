---
name: red-signature-matches-buggy-impl
description: Evaluate a restructured test's stated PRE-FIX failure set against BOTH the correct and the un-refactored implementation — a "these two assertions fail" signature is often satisfied only by the buggy one
metadata:
  type: project
---

When a plan restructures an existing guard test and states a PRE-FIX STATE ("assertions A and B
both fail"), run the plan's own derivation by hand and ask which assertions fail under (i) the
CORRECT restructure and (ii) the status quo with only the anti-vacuous floor widened. If the
stated signature matches (ii), the RED evidence requirement cannot distinguish a real fix from
a no-op.

**Why:** PLAN-ISSUE-180 restructured `TestSandboxHelperGate_PresentInEveryPackvalReachingTestMain`
so STEP 1 derives the packval-reaching set INDEPENDENT of TestMain, STEP 2 floors it at
{cmd/backstop, pkg/packval, pkg/pack/distribution}, STEP 3a asserts every member declares a
TestMain. It stated "STEP 2's floor and STEP 3a both fail" pre-fix. Measured: distribution IS in
the TestMain-independent set, so the floor is GREEN pre-fix and only STEP 3a reds. The stated
two-failure signature is instead what you get if the implementer KEEPS
`if pkg.testMain == nil { continue }` in STEP 1 — the exact non-fix, which then goes fully green
after the TestMain lands with the blind spot intact.

**How to apply:** reconstruct the derived set with a throwaway `go/parser` walk mirroring the
guard's own `scanGoPackages`, then evaluate each stated assertion. Demand the plan name WHICH
single assertion reds and with what message.
Related: [[single-authority-refactor-unfalsifiable]], [[datadriven-red-uniform-quantifier]].
