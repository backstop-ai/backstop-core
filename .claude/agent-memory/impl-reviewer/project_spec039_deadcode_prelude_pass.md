---
name: spec039-deadcode-prelude-pass
description: SPEC-039 (BUNDLE-011 Seed 1) dead-code prelude PASSED review — clean net-negative deletion, substantive orphan reconciliations, faithful CLM adaptation
metadata:
  type: project
---

SPEC-039 (BUNDLE-011 Seed 1, branch bundle/011-codecheck-cutover @ 003f3e3) PASSED impl review — a contrast to the Seed-4/SPEC-035 gap memories.

What made it clean (pattern worth repeating):
- Single production file edited (pkg/check/manifest.go); deletion exactly matched spec scope (dead .manifest.json reader + stranded rule-matching path + non-Go findings catch-all). No out-of-scope deletion; no Seed-2/3/4 production bleed (realCodeChecker/builtinToolchain/gate step/errors.go/step_coverage.go/scaffold.go all untouched — verified via `git diff --name-only`).
- Orphan-test reconciliations were SUBSTANTIVE, not gutted: rewritten tests assert real post-deletion behavior (non-Go→empty slice, skipped-pass inventory), helpers (writeManifest/writeRawManifest) fully removed, no test stubbed-to-skip.
- **CLM-006 adaptation was faithful, not a dodge:** claim wanted "executor map has lint/build/test, never findings." Go's builtinToolchain returns empty Entries (Go's native passes run via go-toolchain pack, not pkg/check), so testing on Go would build an empty map and prove nothing. Implementer correctly switched the test to the TypeScript stack (real lint/build/test entries) to make the present-vs-absent contrast substantive. Lesson: a CLM adaptation that moves to a DIFFERENT fixture can be MORE faithful to claim intent, not less — check whether the original fixture could even exercise the assertion.
- Sharp Edge 1 honored: CLM-005 gate-result test pins violations+exit ONLY, never a PassResults list (the skipped-findings entry intentionally vanishes for non-Go).

**Why the false-positive gate-red was a NON-issue:** pack_engines red was a pre-existing no-global-mutable-state FALSE-FIRE on the `const(...iota)` CheckType enum (fired at baseline too); Seed 1 actually removed a real `var languageExtensions` global (2→1). Net-zero new violations. Verify net-negative gate-reds against baseline before treating as a blocker.

Coverage: pkg/check 93.0%, cmd/backstop 90.4% (threshold 90); every surviving manifest.go function at 100%.
