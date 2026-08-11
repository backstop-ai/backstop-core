---
name: project-spec067-review2
description: SPEC-067 (CI recipe pack, BUNDLE-015 Seed 6) re-review 2026-08-10 — FAILED on a fabricated phase-3 justification introduced by the v1.0.1 fix pass
metadata:
  type: project
---

SPEC-067 v1.0.1 re-review verdict: FAIL. 5 of 6 claimed fixes verified good; the REQ-009
path-scoping fix introduced a new fabrication.

**Why:** the v1.0.1 fix correctly switched the `paths:` include to basename globs — I measured
all four sets against real semgrep 1.156.0 and they behave exactly as the spec says (multi-segment
include matches ZERO, basename globs match both the deployed target and the in-place fixture,
no cross-platform bleed, union matches no tracked core file). But the SECOND justification the
fix introduced — the trailing `*` is mandatory because packval phase 3 runs fixtures in place and
would otherwise fail "negative fixture not triggered" — is not a fact. Phase 3 executes nothing
(see [[feedback_packval_phase3_is_inert]]), and the named error cannot fire from a filtered
fixture anyway. CLM-058 is vacuous as written, and REQ-009's positive/negative assignment is
inverted vs the code it cites.

**How to apply:** on the next SPEC-067 pass, check that the trailing `*` was RE-justified on
grounds that survive (basename globs are the only form semgrep honours; fixture naming is
forward-compatible hygiene), that CLM-058 either mandates `file:` + packval's polarity or drops
its "phase-3 fixture EXECUTION" language, and that the gitlab two-pattern rationale ("a fixture
file name cannot be dot-prefixed" — measured FALSE) is corrected.
