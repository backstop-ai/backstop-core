---
title: "SPEC-852: Partial Source"
number: SPEC-852
created: "2026-08-17"
status: implemented
schema_version: spec/v1
spec_version: "1.0.0"

implementation:
  summary: Source spec with one present and one absent claim test
  package: pkg/gate

verification:
  level: unit
  test_command: go test ./pkg/gate/...
  coverage_threshold: 80

requirements:
  - id: REQ-001
    text: Partial derivation requirement

claims:
  - id: CLM-001
    requirement: REQ-001
    text: The present claim
    tests:
      - TestPlanDerive_Present
  - id: CLM-002
    requirement: REQ-001
    text: The absent claim
    tests:
      - TestPlanDerive_Absent
---

# SPEC-852: Partial Source

Source artifact for PLAN-SPEC-852. CLM-002's test is deliberately absent from the
present-set, so the derived plan does NOT look delivered.
