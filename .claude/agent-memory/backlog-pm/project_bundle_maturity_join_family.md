---
name: bundle-maturity-join-family
description: Bundle-maturity-vs-citing-spec gaps (ISSUE-057 floor + ISSUE-181 ceiling) are ONE family in requirement_traceability.go, where the mirror check citing_spec_not_implemented already ships
metadata:
  type: project
---

The gate ALREADY performs the bundle-maturity × citing-spec join. Before homing or
sizing anything in this family, read `pkg/gate/requirement_traceability.go:107-123`.

**Why:** ISSUE-181 (2026-08-20) asked for "a new (or extended) gate dimension" joining
bundle records to their citing specs, asserting no check covers it — true for its
direction, misleading about the distance. That same per-ref loop already emits
`citing_spec_not_implemented` at `error`: *"delivered bundle X is cited by
non-implemented spec Y."* Both `bundle.Class` and `citer.Class` are in hand there, and
every `Trace` already stamps `BundleMaturity` (line ~274, populated at 4 sites). The
2×2 is half-shipped; the remaining quadrants are extensions, not new dimensions.

**The family (three quadrants, one join):**
- `ISSUE-057 "Bundle Spec Promotion Gate Check"` — the FLOOR: a spec citing a bundle
  still at `idea`/`exploring` resolves fully clean. Validate-side
  (`pkg/validate/supports_resolution.go`, zero hits for `maturity`). Homed: DIR-021 item 1.
- `ISSUE-181 "Bundle Maturity Promotion Detection Gap"` — the CEILING: a bundle whose
  cited specs are ALL `implemented` is never flagged promotion-ready. Needs an
  ALL-quantifier aggregate pass AFTER the ref loop (`covered`/`bundles`/`citers` already
  built there). Escalated 2026-08-20, unhomed pending a shape ruling.
- Shipped: `citing_spec_not_implemented` (delivered bundle + non-implemented citer).

**How to apply:** home this family under `DIR-021 "Traceability Hardening & Corpus
Drain"` (charter: residual gaps between what the gate structurally verifies and what it
takes on prose/trust), NOT DIR-034 — DIR-034's clauses are ID-collision and
CLOSED-artifact vacuity, and a stale-open bundle is neither. 057 and 181 should be
planned together; both are founder-flagged LOW, both trace to the same BUNDLE-003
SPEC-020..029 fossil incident. When an issue proposes "a documented backlog-pm sweep
habit" as an alternative shape, say plainly it has NO home (`.claude/` is orphaned —
see [[project_harness_config_has_no_home]]) and enforces nothing
(`project_prompts_are_vibes`). Related: [[project_check_the_siblings_plan]],
[[feedback_verify_the_loss_claim]].
