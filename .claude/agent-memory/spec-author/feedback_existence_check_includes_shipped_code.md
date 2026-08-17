---
name: existence-check-includes-shipped-code
description: The existence-in-world check must probe the SHIPPED CODE, not just specs/ — a stale draft spec's scope is often already delivered by the issue track under different test names
metadata:
  type: feedback
---

The existence-in-world check before authoring or reconciling a spec must include a
probe of the **shipped implementation**, not only a search of `specs/` for an
overlapping artifact.

**Why:** SPEC-032 (fixture engine execution) sat at `draft` from 2026-06-16 to
2026-08-16. Reconciling it, the assumption was that two sibling plans landed that
night had overlapped it. The truth was larger: ~9 of its 10 requirements were
already fully implemented in `pkg/packval` — mostly by ISSUE-019 months earlier,
with ISSUE-092/ISSUE-141 finishing the rest. No sibling *spec* overlapped it, so an
artifact-only existence check would have found nothing and waved it through. The
duplicate was the codebase.

Two failure modes this catches, both of which appeared in that one spec:

1. **Silent delivery under different names.** The spec's mandated test names
   (`TestPackVal_EngineContract_*`, `TestPackVal_EngineDispatch_*`) mostly do not
   exist, yet the behavior is covered by differently-named shipped tests
   (`TestPackVal_P3_*`, `TestExecutor_*`). Grepping for the spec's OWN mandated test
   names reports "missing" and reads as residual scope. It is not. Read the
   implementation and enumerate what its real tests assert.
2. **Requirements that shipped code actively contradicts.** SPEC-032 REQ-010 asked to
   *retain and condition* a `go mod tidy` pre-check; ISSUE-019 had **deleted** it as a
   baked Go path, and a shipped test now asserts the identifier stays absent.
   Implementing that requirement as written would have broken a passing test and
   re-baked a language into the thin executor. A stale spec is not merely incomplete —
   parts of it can be actively wrong.

**How to apply:** before authoring from a bundle seed, and always before reconciling a
draft spec older than a few weeks, read the target package's real source and test-function
list first. Map requirement → shipped behavior → the test that actually pins it. Only what
survives that mapping is residual scope. When most of it is absorbed, retire rather than
reduce: `status: replaced` accepts a LIST in `replaced-by`, so multi-artifact delivery
(here `[ISSUE-019, ISSUE-092, ISSUE-141]`) is expressible and is more honest than
`deprecated`. Retire the spec's plan in lockstep — but note spec-author is **not permitted
to write plan artifacts**, so hand that edit to a planner and say plainly that it is not
done. See [[feedback_align_predating_artifacts]].
