---
title: "TEST-201: Deprecated Broken Spec (cmd)"
number: TEST-201
created: "2026-06-27"
status: deprecated
schema_version: spec/v1
spec_version: "1.0.0"

implementation:
  summary: Deprecated broken spec for cmd-level exclusion test
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
      - TestCmd_Deprecated_DoesNotExist_201

contracts:
  - file: pkg/gate/deleted_cmd_deprecated.go
    provides:
      - name: DeletedCmdDeprecatedSymbol
        kind: function
        signature: "func DeletedCmdDeprecatedSymbol() error"
---

# TEST-201: Deprecated Broken Spec (cmd)
