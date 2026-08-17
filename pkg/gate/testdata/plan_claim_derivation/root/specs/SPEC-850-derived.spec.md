---
title: "SPEC-850: Derived Source"
number: SPEC-850
created: "2026-08-17"
status: implemented
schema_version: spec/v1
spec_version: "1.0.0"

implementation:
  summary: Source spec whose single claim test a plan derives
  package: pkg/gate

verification:
  level: unit
  test_command: go test ./pkg/gate/...
  coverage_threshold: 80

requirements:
  - id: REQ-001
    text: Derivation source requirement

claims:
  - id: CLM-001
    requirement: REQ-001
    text: The derivation source claim
    tests:
      - TestPlanDerive_Present
---

# SPEC-850: Derived Source

Source artifact for PLAN-SPEC-850. Its CLM-001 tests are what the plan's task
`claims: [CLM-001]` resolves to.
