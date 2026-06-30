---
name: spec043-foundation-pass
description: SPEC-043 BUNDLE-012 Seed 1 (pack-declared classification globs + de-Go'd coverage measurable-path) reviewed PASS
metadata:
  type: project
---

SPEC-043 (BUNDLE-012 Spec Seed 1, the FOUNDATION contract) reviewed **PASS** on
`bundle/011-codecheck-cutover` (commits 4cb59b6..3eec0d1).

**Why:** First seed of BUNDLE-012 (language-neutral gate consumer + TS toolchain).
Closes the default-diff-scope vacuous-green hole: a changed non-Go source file was
silently skipped by the baked `.go` literal in `coverageMeasurablePath`.

**How to apply:** This is the clean template for de-language-X-ing a consumer.
What made it pass:
- Anti-vacuous-green guard is REAL and fail-on-revert: `coverage_unmeasured` (error)
  fires unconditionally in the path loop, distinct from `coverage_threshold`. The
  `if threshold<=0 {return pass}` early return is genuinely DISMANTLED (only in
  comments now). CLM-012 uses a bun classifier (`**/*.ts`) so it fails if reverted to
  the baked `.go`; CLM-014 fires at threshold 0 so it fails if the early return returns.
- Matcher is `bmatcuk/doublestar/v4` (direct dep), NOT gobwas; classification.go has
  zero gobwas. CLM-023 pins root-file `embed.go` measurable + root `foo_test.go` not.
- `SourceClassifier` stores BOTH source and test glob sets (IsTestFile/HasTestGlobs
  present) so SPEC-045 isn't orphaned; IsMeasurableSource = source ∧ ¬test.
- E2E (CLM-020) drives the REAL `buildGateSteps`/`runGate` seam (exercises live
  `mergeSourceClassifier(packs)` at gate.go:645), NOT a hand-merged classifier — the
  SPEC-035/037 integration-gap closure done right. E2E is hermetic (packs declare NO
  engine, so no record ⇒ guard reds) which is correct for THIS spec's no-record guard.
- Dogfood lockstep (TASK-014): go-toolchain classification block landed in the SAME
  change set as the literal deletion, in BOTH the in-repo .backstop copy and the
  cmd/backstop testdata copy; external pack-repo edit is sibling commit b398bd6.
- Reported scope deviation (`runCoverage` test helper now threads `goClassifier()`)
  is a legitimate compile necessity for the 4-arg signature; supplies `**/*.go` so the
  existing SPEC-042 `.go`-path assertions stay valid — no weakening.
- 24 mandated test fns all present + green; no hollow tests.

Contrast with [[spec035-phase4-gap]] / [[pack-step-provisioning-gap]]: those FAILED on
masked integration gaps; SPEC-043 explicitly drove the live wiring seam.
