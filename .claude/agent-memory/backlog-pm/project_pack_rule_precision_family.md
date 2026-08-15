---
name: pack-rule-precision-family
description: Pack rule-precision false positives (go-standards GO-005/GO-021) home under DIR-024 by charter, but ISSUE-061 sits in DIR-021 as a deadline-driven exception — check both before ruling
metadata:
  type: project
---

Pack RULE-PRECISION defects (a pack rule's pattern is too broad → false
positives on correct code) home under **DIR-024 "Gate/Engine Quality"** by
charter, on the ISSUE-096 precedent that DIR-024's own Notes spell out. Not
DIR-027 (owns which packs exist / where they point, explicitly disclaims rule
content), not DIR-032 (verdict honesty — a too-broad rule reports exactly the
verdict its pattern earns), not DIR-005 (`done`).

**The trap:** `ISSUE-061` (go-standards `error-type-suffix`, GO-021) — the same
defect class — is homed in **DIR-021 "Traceability Hardening & Corpus Drain"**,
where DIR-021's text introduces it *outside* its four numbered threads as a
deadline-driven exception. Hosting ≠ charter. Don't read 061's placement as
precedent for DIR-021 owning rule precision.

**Live family in `backstop-ai/go-standards` (as of 2026-08-15):** GO-021
(ISSUE-061, DIR-021) and GO-005 (ISSUE-125, DIR-024) are both WARNING, both in
`rules/core/go-core.yml`, both live at installed v1.2.1. One version bump fixes
both. ISSUE-061's inline waiver at `cmd/backstop/artifact_validate.go`
**expires 2026-10-12**, after which the gate goes RED on a false positive —
that expiry is why DIR-021 holds its backlog position. Pack source repo is
`~/src/projects/backstop-go-pack` (dir name ≠ pack name `backstop-ai/go-standards`).

**Why:** Brandon asked (via team-lead, 2026-08-15) that pack-rule issues be
ruled on subject matter, not on the recent DIR-002 slotting pattern.

**How to apply:** on any pack-rule FP, slot to DIR-024, then check whether a
sibling defect in the SAME pack is already homed elsewhere — surface the
one-bump batching and any waiver expiry as a sequencing finding. Re-homing
061/125 together is an unruled founder call.

Related: [[gate_verdict_honesty_cluster]], [[project_packs_learn_from_scenarios]]
