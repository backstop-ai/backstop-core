---
title: "TEST-101: Deprecated Broken Spec"
number: TEST-101
created: "2026-06-27"
status: deprecated
schema_version: spec/v1
spec_version: "1.0.0"

implementation:
  summary: Deprecated broken spec
  package: pkg/gate

verification:
  level: unit
  test_command: go test ./pkg/gate/...
  coverage_threshold: 80

requirements:
  - id: REQ-001
    text: Deprecated requirement

claims:
  - id: CLM-001
    requirement: REQ-001
    text: Deprecated claim
    tests:
      - TestDeprecated_DoesNotExist_101

contracts:
  - file: pkg/gate/deleted_deprecated.go
    provides:
      - name: DeletedDeprecatedSymbol
        kind: function
        signature: "func DeletedDeprecatedSymbol() error"
---

# TEST-101: Deprecated Broken Spec
