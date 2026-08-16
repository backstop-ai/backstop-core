---
title: "SPEC-904: Invented Ecosystem Spec"
number: SPEC-904
created: "2026-08-16"
status: implemented
schema_version: spec/v1
spec_version: 1.0.0

implementation:
  summary: >
    CLM-008's payload. It sits under `_thirdparty_deps`, a directory name that
    appears in NO core source file — invented for this fixture. Core excludes it
    solely because the fixture pack declares it, which is what makes "adding the
    next ecosystem's vendoring convention costs zero core edits" falsifiable
    rather than asserted.
  subject: cmd/backstop

verification:
  level: integration
  test_command: go test ./cmd/backstop/
  coverage_threshold: 80

requirements:
  - id: REQ-001
    text: a pack-declared directory name unknown to core is honored end to end

claims:
  - id: CLM-001
    requirement: REQ-001
    text: an invented dependency directory name is excluded with no core edit
    tests:
      - TestArtifactDiscoveryE2E_InventedEcosystemDirHonoredWithNoCoreEdit

contracts:
  - file: cmd/backstop/pack_dependency_dirs.go
    consumes:
      - name: mergeDependencyDirs
        kind: function
        source: cmd/backstop
---

# SPEC-904: Invented Ecosystem Spec

## Overview

Sits under `_thirdparty_deps`, a name no core source file contains. If core ever
regains a hardcoded ecosystem list, this file's exclusion cannot come from it.

## Requirements

REQ-001 — a pack-declared directory name unknown to core is honored end to end.

## Implementation

None. Fixture data.

## Verification

Asserted by the discovery tests in cmd/backstop.
