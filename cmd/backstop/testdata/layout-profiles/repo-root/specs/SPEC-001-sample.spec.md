---
title: "SPEC-001: Layout Profile Sample Spec"
number: SPEC-001
created: "2026-08-14"
status: implemented
schema_version: spec/v1
spec_version: 1.0.0

implementation:
  summary: A layout-profile fixture spec used to prove artifact-root resolution end to end.
  subject: pkg/layoutprofile

verification:
  level: integration
  test_command: go test ./...
  coverage_threshold: 80

requirements:
  - id: REQ-001
    text: The layout-profile fixture corpus is discoverable under the resolved artifact root.

claims:
  - id: CLM-001
    requirement: REQ-001
    subject: pkg/layoutprofile
    text: The fixture corpus declares one mandated test name so a spec-directory read is distinguishable from an empty read
    tests:
      - TestLayoutProfileSampleMandatedNeverExists
---

# SPEC-001: Layout Profile Sample Spec

## Overview

A minimal implemented spec used by the layout-profile fixture projects.

## Requirements

Requirements are declared in frontmatter.

## Implementation

Implementation details are declared in frontmatter.

## Verification

Verification details are declared in frontmatter.
