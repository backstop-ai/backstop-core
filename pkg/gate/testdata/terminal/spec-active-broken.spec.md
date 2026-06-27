---
title: "TEST-100: Active Broken Spec"
number: TEST-100
created: "2026-06-27"
status: draft
schema_version: spec/v1
spec_version: "1.0.0"

implementation:
  summary: Active broken spec
  package: pkg/gate

verification:
  level: unit
  test_command: go test ./pkg/gate/...
  coverage_threshold: 80

requirements:
  - id: REQ-001
    text: Active requirement

claims:
  - id: CLM-001
    requirement: REQ-001
    text: Active claim
    tests:
      - TestActive_DoesNotExist_100

contracts:
  - file: pkg/gate/deleted_active.go
    provides:
      - name: DeletedActiveSymbol
        kind: function
        signature: "func DeletedActiveSymbol() error"
---

# TEST-100: Active Broken Spec
