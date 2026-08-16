---
title: "SPEC-903: Nested Vendored Spec"
number: SPEC-903
created: "2026-08-16"
status: implemented
schema_version: spec/v1
spec_version: 1.0.0

implementation:
  summary: >
    Planted SEVERAL LEVELS DEEP, at pkg/sub/vendor/, so the match is proven to be
    on the directory BASE NAME at any depth rather than on a root-relative path.
  subject: cmd/backstop

verification:
  level: integration
  test_command: go test ./cmd/backstop/
  coverage_threshold: 80

requirements:
  - id: REQ-001
    text: a dependency directory nested several levels deep is excluded too

claims:
  - id: CLM-001
    requirement: REQ-001
    text: depth does not defeat the exclusion
    tests:
      - TestArtifactDiscoveryE2E_NestedDependencyDirExcludedAndFileNamedVendorUnaffected

contracts:
  - file: pkg/artifact/layout.go
    consumes:
      - name: NonCorpusDirs
        kind: type
        source: pkg/artifact
---

# SPEC-903: Nested Vendored Spec

## Overview

Planted at `pkg/sub/vendor/` rather than at the root, so an implementation that
matched a root-relative path instead of the directory base name would be caught.

## Requirements

REQ-001 — a dependency directory nested several levels deep is excluded too.

## Implementation

None. Fixture data.

## Verification

Asserted by the discovery tests in cmd/backstop.
