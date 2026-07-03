---
name: spec042-review1
description: SPEC-042 (coverage production engine, BUNDLE-011 Seed 4) review #1 verdict FAIL — two grounding defects
metadata:
  type: project
---

SPEC-042 (coverage PRODUCER: engine-model coverage-records channel + go-toolchain coverage engine; BUNDLE-011 Seed 4, REQ-014/015/016, DD-7) — review #1 verdict **FAIL** (2026-06-24).

**Why FAIL:** deliverables are correct and well-claimed (non-SARIF channel sound, language-agnostic record sound, non-vacuousness preserved, single-type contract declared, real-E2E mandated — all verified against main). But two stated-as-main-verified facts are false on main:
1. Spec cites SPEC-041's `CoverageRecord{Path, Pct, Measured, Excluded}` + re-implemented records-based coverage step as landed/"verified on main 2026-06-24". On main: no CoverageRecord type, no ParsePackCoverage, SPEC-041 still draft; live coverage step is the OLD baked Go `StepCoverageThresholdFunc` (pkg/gate/step_coverage.go). See [[sibling-draft-not-landed]].
2. Impl step 4 + Review-Q7 claim pack-`engines:`-block declaration is "consistent with the go-build/go-test bindings" — false: go-build/go-test/golangci are baked DefaultRegistry bindings (pkg/pack/engine/binding.go:270-306), go-toolchain pack.yml has no engines: block. Prescription is right (declare via engines: block), analogy is wrong.

**How to apply:** when re-reviewing, confirm both grounding corrections landed; the deliverable claims (REQ→CLM coverage, sharp edges, contracts) did NOT need rework. Validator PASSes. Related: [[feedback_coverage_producer_gap]] (the SPEC-040-side corrective on the consumer half).
