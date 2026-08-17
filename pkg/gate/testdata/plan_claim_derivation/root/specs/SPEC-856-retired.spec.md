---
title: "SPEC-856: Retired Source"
number: SPEC-856
created: "2026-08-17"
status: obsoleted
obsoleted-by: ISSUE-114
schema_version: spec/v1
spec_version: "1.0.0"

implementation:
  summary: Retired-terminal source spec whose claims are withdrawn
  package: pkg/gate

verification:
  level: unit
  test_command: go test ./pkg/gate/...
  coverage_threshold: 80

requirements:
  - id: REQ-001
    text: Retired derivation requirement

claims:
  - id: CLM-001
    requirement: REQ-001
    text: The withdrawn claim
    tests:
      - TestPlanDerive_RetiredOnly
---

# SPEC-856: Retired Source

`obsoleted` ON PURPOSE. isTerminalSpecStatus returns FALSE for `obsoleted`, so
ExtractMandatedTests does NOT drop this spec and its record genuinely carries
TestPlanDerive_RetiredOnly. That is what makes the retired-source guard in the
plan derivation a real test rather than an assertion about an already-empty set.
