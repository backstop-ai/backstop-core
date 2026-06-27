---
title: "TEST-102: Replaced Broken Spec"
number: TEST-102
created: "2026-06-27"
status: replaced
replaced-by: BUNDLE-011
schema_version: spec/v1
spec_version: "1.0.0"

implementation:
  summary: Replaced broken spec
  package: pkg/gate

verification:
  level: unit
  test_command: go test ./pkg/gate/...
  coverage_threshold: 80

requirements:
  - id: REQ-001
    text: Replaced requirement

claims:
  - id: CLM-001
    requirement: REQ-001
    text: Replaced claim
    tests:
      - TestReplaced_DoesNotExist_102

contracts:
  - file: pkg/gate/deleted_replaced.go
    provides:
      - name: DeletedReplacedSymbol
        kind: function
        signature: "func DeletedReplacedSymbol() error"
---

# TEST-102: Replaced Broken Spec
