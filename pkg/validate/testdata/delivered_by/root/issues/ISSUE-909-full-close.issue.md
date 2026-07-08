---
title: "Full non-plan-backed close carrying the whole traceability chain"
schema_version: issue/v1

issue:
  id: ISSUE-909
  title: "Full non-plan-backed close carrying the whole traceability chain"
  type: bug
  status: closed
  created: "2026-07-08"
  closed: "2026-07-08"

verification:
  level: unit
  coverage_threshold: 90
  test_command: "go test ./pkg/example/..."

implementation:
  summary: >
    Fix the off-by-one in the example accumulator and add regression coverage.
  package: pkg/example

requirements:
  - id: REQ-001
    text: >
      The example accumulator must not drop the final element when the input
      length is odd.

claims:
  - id: CLM-001
    requirement: REQ-001
    text: >
      Accumulate includes the final element for odd-length inputs.
    tests:
      - TestAccumulate_OddLengthIncludesFinalElement

contracts:
  - file: pkg/example/example.go
    provides:
      - name: Accumulate
        kind: function
        signature: "func Accumulate(xs []int) int"
---

# Full non-plan-backed close carrying the whole traceability chain

## Problem

A normal closed issue that carries its own requirements, claims, verification,
implementation, and contracts — no delivered_by pointer. This must still
validate clean after the relaxation, proving the unconditional path is intact.

## Resolution

Fixed and covered by a regression test.
