---
title: "SPEC-854: Union Source"
number: SPEC-854
created: "2026-08-17"
status: implemented
schema_version: spec/v1
spec_version: "1.0.0"

implementation:
  summary: Source spec whose single claim yields two tests, one of them absent
  package: pkg/gate

verification:
  level: unit
  test_command: go test ./pkg/gate/...
  coverage_threshold: 80

requirements:
  - id: REQ-001
    text: Union derivation requirement

claims:
  - id: CLM-001
    requirement: REQ-001
    text: The union claim
    tests:
      - TestPlanDerive_Present
      - TestPlanDerive_Absent
---

# SPEC-854: Union Source

Source artifact for PLAN-SPEC-854. The plan declares TestPlanDerive_Present
explicitly AND derives both names from CLM-001, so the union must dedupe to
exactly two entries.
