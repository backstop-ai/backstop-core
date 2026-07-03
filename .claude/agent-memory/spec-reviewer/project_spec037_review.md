---
name: spec037-review
description: SPEC-037 v1.2.1 (BUNDLE-009 Seed 3, substantiveness pack) PASSED re-review; the 3 SPEC-036 capability re-key blockers are closed + verified vs live code
metadata:
  type: project
---

SPEC-037 v1.2.1 (specs/SPEC-037-traceability-substantiveness-pack.spec.md) PASSED backstop
re-review 2026-06-23 and is READY TO PLAN. v1.2.0 had FAILED on the SPEC-036 capability
re-key coupling (3 blockers); v1.2.1 closed all three. Validator PASS (10 REQ / 37 CLM,
no dup ids, no orphan claims, both new tests bound once).

**The 3 v1.2.0 blockers, now CLOSED (verified vs live code, not just spec assertions):**
1. Live locus named: REQ-009 + CLM-035 + Implementation + a NEW deriveCapabilityState
   contract entry all name deriveCapabilityState (cmd/backstop/gate.go:272) as the function
   whose SUBSTANTIVENESS arm re-keys onto installed-pack presence. Verified live: that
   function IS dimension-uniform today (returns Present iff lang=="go" for all 3 dims,
   string(dim) cosmetic) — so the premise + required split are exact.
2. Existing-test-coupling closed: REQ-008 EXTENDED beyond pkg/gate to name the shipped
   TestCapabilityState_NonGoProject_DerivesAbsentClass2 (cmd/backstop/gate_capability_test.go:17)
   for MIGRATE; new CLM-037 (TestCapability_ShippedSpec036Test_MigratedForSubstantivenessRekey,
   bound to REQ-008) pins: substantiveness arm -> installed-pack keying, coverage/contracts arms
   UNCHANGED, ./cmd/backstop/ stays green. Verified: that test's goCfg fixture (no packs) makes
   the substantiveness arm flip Present->Absent after re-key, so the migration is genuinely
   required, correctly scoped, not a no-op claim.
3. Dimension-asymmetry pinned: REQ-009 + CLM-036 (TestCapability_RekeyIsSubstantivenessOnly_
   CoverageContractsUnchanged, bound to REQ-009) + a Sharp Edge state ONLY the substantiveness
   arm re-keys; coverage descoped (BUNDLE-009 REQ-009), contracts keeps its baked analyzer
   until SPEC-038 ships its pack. Verified live: step_contract.go still has baked
   StepContractSignatureFunc -> contracts premise holds; re-keying contracts now would break
   it pre-pack. align-predating-artifacts: SPEC-036 NOT revised, aligned via impl.

**Sound core (NOT re-litigated, already passed v1.2.0):** REQ-009 provisioning faithful to
distribution.Add/VerifyLock; REQ-010 real over-installed-pack E2E genuinely unstubbable
(ISSUE-028/029 shipped); set-join / strangler-before-deletion / Q1 hollow / TS rule intact.

**How to apply:** SPEC-037 is cleared to plan. Re-walk only if version > 1.2.1. The capability
re-key is the load-bearing cross-spec coupling — it overturns [[spec036-ready-to-plan]]'s
deriveCapabilityState via impl (not a SPEC-036 rev). Sibling of [[project_spec038_review1_fail]]
(shares the TS proof pack).
