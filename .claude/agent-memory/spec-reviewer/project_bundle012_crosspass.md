---
name: bundle012-crosspass
description: BUNDLE-012 (lang-neutral consumer + bun toolchain) 5-spec cross-consistency review #1 — FAIL with 3 blockers
metadata:
  type: project
---

BUNDLE-012 sliced into SPEC-043 (pack-declared globs + coverage measurable-path), 044 (multi-metric `(path,metric)` records), 045 (de-Go test-verify discovery + matchers), 046 (retire `language:` bridge + field), 047 (bun pack + two-surface proof + ratchet→block flip). Cross-consistency review #1 = **FAIL**.

**Why:** three blocking seam inconsistencies (verified vs live code 2026-06-28):
1. `StepCoverageThresholdScopedFunc` — 043 adds `classifier` (4-arg) but 044 declares 3-arg "signature UNCHANGED". Reconcile to 4-arg; fix 044's contract. [[shared-function-signature-conflict]]
2. `mergeSourceClassifier(bridged, packs)` — 043 sources from `bridged` which 046 deletes; 046's edit-set omits mergeSourceClassifier → orphan. Re-source from declared packs only; assign owner. [[deleted-concept-orphans-consumer]]
3. Zero-baseline flip (047 REQ-006/CLM-033, bundle REQ-008 success criterion): live `enforcement.Policy` keys ONLY per-dimension (StepName); backstop/self neutral-spine findings ride shared `pack_engines` dim. No per-pack key exists; 047 punts to a sharp edge. Fix = route backstop/self to its own gate dimension OR scope a finer policy key as a real requirement.

Plus SHOULD-FIX: 044 miscites bun producer as "SPEC-045" (is 047); step_testverify.go co-edit by 044+045 unflagged; go-toolchain pack.yml needs classification(043)+test_name_patterns(045) in lockstep + in the EXTERNAL repo (durability).

**How to apply:** when re-review arrives, confirm the 3 blockers closed against live `pkg/gate/step_coverage.go`, `cmd/backstop/gate.go`, `pkg/config/config.go` + `pkg/gate/policy.go` (per-dimension only). All 9 bundle REQs were covered; structure validated PASS — the failure was purely cross-spec seams.
