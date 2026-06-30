---
name: spec036-ready-to-plan
description: SPEC-036 (traceability fail-loud, BUNDLE-009 Seed 1) PASSED on review #2 (v1.1.0); the three FAIL findings are closed and verified against live code
metadata:
  type: project
---

SPEC-036 v1.1.0 (specs/SPEC-036-traceability-fail-loud.spec.md) PASSED backstop review on
2026-06-22 (second review) and is READY TO PLAN. Implements BUNDLE-009 Spec Seed 1 only
(fail-loud on undeclared/capability-absent traceability dimensions; traces to bundle REQ-001
via supports: stack-aware-traceability:REQ-001). NO analyzer touched, no engine, no pack.

**Why:** A prior FAIL had three findings; the v1.1.0 pass closed all three, each re-verified
against live code (not just the spec's assertions):
1. Wiring-verification gap — test_command now includes ./cmd/backstop/ (the wiring locus:
   buildGateSteps cmd/backstop/gate.go:288, builders :453/:475/:496). New CLM-028
   (TestWiring_ClassifierInterceptsClass123_AndFallsThroughWhenWorking) uses a spy on the
   analyzer delegate, so an unwired classifier FAILS it (unlike CLM-025/026/027 which a
   never-wired classifier would also pass). Genuinely closes the hole.
2. CapabilityState derivation — REQ-003 + CLM-029 pin the no-pack derivation: Present/Working
   only when cfg.Language=="go" AND baked Go analyzer exists; Absent when cfg.Language!="go"
   -> class 2. VERIFIED cfg.Language is real (pkg/config/config.go:22, `language` yaml field),
   so the grounding is honest and reads cfg.Language + baked-analyzer presence ONLY (no
   pack/engine). TS-runtime -> class-2 case can't be stubbed.
3. warning summary count — REQ-005 + CLM-030 mandate new GateResult.StepsWarned (contract
   declares it on result.go), populated by NewGateResultWithScope, rendered in FormatHuman
   summary line. Closes the summary-drop (live FormatHuman output.go:102 only had
   passed/failed/skipped; warning status falls through the result.go:127 switch silently).

Three-class matrix, exit polarities (ConfigErr->exit2 via gate.go:138 halt; warning non-failing
for Pass per result.go:127-135), waive matrix (silences class 2 only, CLM-022/023/024), and
back-door-vacuous-green guard (Sharp Edge 1 + CLM-015/017) were NOT churned. Validator PASS.

**How to apply:** If asked to review SPEC-036 again, this was the clean PASS — re-walk only if
version changed past 1.1.0. Note the grounding pattern that mattered: REQ-003's claim that
cfg.Language is "already read in the gate path" was the load-bearing assertion to verify
against config.go — it held. Sibling of [[spec035-ready-to-plan]] (same BUNDLE-009/035 lineage).
