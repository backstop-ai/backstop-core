---
title: "SPEC-901: Deprecated Example Spec"
number: SPEC-901
created: "2026-06-27"
status: deprecated
schema_version: spec/v1
spec_version: "1.0.0"

implementation:
  summary: "Retired implementation target"
  package: "pkg/example"

verification:
  level: unit
  coverage_threshold: 90
  test_command: "go test ./..."

requirements:
  - id: REQ-001
    text: "Example requirement"

claims:
  - id: CLM-001
    requirement: REQ-001
    text: "Example claim"
    tests:
      - test_name: TestExample
---

# SPEC-901: Deprecated Example Spec

## Overview
Deprecated spec fixture.

## Requirements
See frontmatter.

## Implementation
See frontmatter.

## Verification
See frontmatter.
