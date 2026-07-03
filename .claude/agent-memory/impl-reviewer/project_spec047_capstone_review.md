---
name: spec047-capstone-review
description: SPEC-047 (BUNDLE-012 capstone) review — in-repo code/data/tests all sound + substantive; the one gap is REQ-005 external executed acceptance never performed (fork unwired, no run-evidence)
metadata:
  type: project
---

SPEC-047 (BUNDLE-012 Seed 5, bun-toolchain pack + two-surface proof) reviewed 2026-06-30 on `bundle/011-codecheck-cutover` (commits f74eb91..d06dffa + external repo ~/src/projects/backstop-bun-toolchain-pack @1147df0).

VERDICT: FAIL on one blocking gap only; everything else PASS.

What's SOUND (verified, all 40 mandated tests exist + pass/skip-as-designed):
- Bun pack: 5 engines correct (oxlint→lint, prettier→lint as format-as-lint with NO new dimension, tsc→build, bun test→test, bun coverage→coverage); classification globs; lcov convert emits SPEC-044 line+branch two-record shape through existing check.ParsePackCoverage (strict-parser negative case tested). External repo pack.yml/scripts BYTE-IDENTICAL to in-repo testdata copy.
- REQ-007 per-pack policy (pkg/gate/policy.go `applyScopedPolicy` + config Sources map + gatePolicyFromConfig): matrix real + fail-on-revert. backstop/self flip blocks fresh neutral-spine; go-standards/go-toolchain baselined debt stays grandfathered; no whole-dim zero-baseline (CLM-037..040).
- REQ-006 ratchet/flip: baseline un-grandfathered for the 3 de-Go'd sites; terminal flip via per-pack key in backstop.yml; WALL test discriminating (baselined finding, pre-flip pass, post-flip block); ordering-guard real.

The BLOCKING gap — REQ-005 external executed proof NOT performed:
- The fork ~/src/projects/backstop-runtime STILL carries a stale backstop.yml with `language: typescript` (the SPEC-046-RETIRED field) and NO bun-toolchain pack declared/installed (.backstop/packs/backstop empty).
- No RED-then-green run-evidence recorded anywhere. Spec is explicit: "the executed proof, not the skipped Go-CI stub, is what closes REQ-005" + requires captured run-evidence in the verification log. The guarded test + bunAcceptanceEnabled() + docs are all present and correct (CLM-027/028/029 skip cleanly), but the REQUIRED acceptance itself is unmet.

RECURRING PATTERN (ties to [[project_pack_provisioning_integration_gap]]): "two-surface proof" specs land the in-repo stubbed surface cleanly but leave the EXTERNAL executed surface unperformed. On capstone reviews, always check the fork is actually wired + evidence captured, not just that the guarded test skips.

Should-fix (non-blocking): applyScopedPolicy grandfathers ALL findings when baseline==nil (newSet empty ⇒ baseline:true sources count nothing), whereas the unscoped path blocks all when baseline==nil. Divergence toward silent-green under a missing-baseline degraded condition. Mirror the unscoped nil-baseline semantics.

UPDATE 2026-06-30 (commit 5bef5f9 re-review): all three findings addressed. VERDICT FLIPPED TO PASS.
- nil-baseline should-fix FIXED: applyScopedPolicy now gates net-new lookup on `eff.Baseline && baseline != nil`, mirroring the unscoped fail-loud path. New test TestPolicy_ScopedNilBaselineBlocksFailLoudNotSilentGreen is substantive + fail-on-revert (asserts BOTH the baseline:true go-standards finding AND the zero-baseline self finding land in NewViolations under nil baseline). All 5 TestPolicy_* pass.
- nit FIXED: pillarASitesClean now counts nosemgrep-suppressed literals as not-clean (whole-file Contains), with a comment that the REAL wall is the live gate/CLM-034.
- REQ-005 wiring DONE (verified independently, not on coordinator's word): fork backstop.yml packs-only, retired language: gone (grep 0), bun pack installed under .backstop/packs matching external repo. The ACTUAL executed RED-then-green run is a user-environment MANUAL acceptance (no bun in sandbox; OQ-4 makes it a user step) — implementation obligation was to make it READY (wired + documented + guarded test), which it now is. Captured run-evidence is the one pending USER-manual step, NOT an implementation/code defect.
