---
name: spec035-ready-to-plan
description: SPEC-035 (pack-declared engines + trusted allowlist) PASSED on review #3 (v1.1.0); the four residual findings are closed and verified against live code
metadata:
  type: project
---

SPEC-035 v1.1.0 (specs/SPEC-035-pack-declared-engines-trusted-allowlist.spec.md) PASSED
backstop review on 2026-06-21 (third review) and is READY TO PLAN.

**Why:** A prior FAIL had four residual findings; the v1.1.0 corrective pass closed all
four, and I re-verified each against live code (not just the spec's assertions):
1. FieldContract folded onto the binding — DefaultFieldContracts (pkg/pack/engine/
   fieldcontract.go) + engineFieldClaim (pkg/pack/validate_manifest.go:114) both real,
   both placed under OQ-1 (CLM-036/037).
2. SemgrepVersion/PinnedSemgrepVersion named as superseded by the allowlist, deleted by
   ISSUE-018 not SPEC-035 (no new delete-req added).
3. ISSUE-018 ordering stated (REQ-005 + Sharp Edge 11); the two collision sites
   (check.go:322 delete, registry.go:249 execs[]) confirmed executor-internal.
4. Compiled-standards-manifest routing (hasSemgrepSignal/deriveRules/compiledManifestFile,
   pkg/check/manifest.go:74-217) explicitly OUT OF SCOPE / delegated.
Full tool-name literal partition across cmd/backstop+pkg/pack+pkg/check left ZERO orphans.

**How to apply:** If asked to review SPEC-035 again, this was the clean PASS — re-walk only
if the spec version changed past 1.1.0. The one loose phrase ("PinnedSemgrepVersion has no
other consumer" in REQ-002) is imprecise — check.go:311 Run() is a second reader — but it's
an ISSUE-018 scoping nuance, not a SPEC-035 obligation. Builds on
[[feedback_toolname_eradication_surface]] (the line-collision sequencing gap that this
corrective pass closed).
