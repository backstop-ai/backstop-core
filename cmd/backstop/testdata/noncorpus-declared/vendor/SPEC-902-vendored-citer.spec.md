---
title: "SPEC-902: Vendored Citer Spec"
number: SPEC-902
created: "2026-08-16"
status: draft
schema_version: spec/v1
spec_version: 1.0.0

implementation:
  summary: >
    THE FALSIFIER PAYLOAD. A SCHEMA-VALID spec planted inside a vendored
    dependency tree. It is valid on purpose: an invalid file would fail the
    per-artifact pass and mask the signal this fixture exists to produce. Its
    supports ref names `absent-fixture-bundle`, which exists NOWHERE in this
    fixture (the plan said "BUNDLE-999", but supportsRe requires a lowercase-kebab
    bundle NAME — an uppercase id fails the format rule and would make this very
    file invalid, defeating the "planted artifacts must be valid" requirement the
    same plan sets; the name changed, the semantics did not), and its
    status is deliberately NON-TERMINAL so CollectSupportRefs does not skip it.
    That combination is what lets the corpus resolution pass emit a violation
    naming this file the moment the exclusion set stops excluding vendor/ — which
    is how the ValidateConfig.NonCorpus hop and the collectTraceRefs hop are
    falsifiable at all. Its basename collides with nothing else here, because
    ResolveSupports reports its citing file by BASENAME.
  subject: cmd/backstop

verification:
  level: integration
  test_command: go test ./cmd/backstop/
  coverage_threshold: 80

requirements:
  - id: REQ-001
    text: this requirement exists only to carry the unresolvable supports pin below
    supports: absent-fixture-bundle:REQ-001@1.0.0

claims:
  - id: CLM-001
    requirement: REQ-001
    text: this file is never discovered while a pack declares vendor
    tests:
      - TestArtifactDiscoveryE2E_VendorAndNodeModulesExcludedByPackDeclaration

contracts:
  - file: cmd/backstop/artifact_validate.go
    consumes:
      - name: buildResolutionViolations
        kind: function
        source: cmd/backstop
---

# SPEC-902: Vendored Citer Spec

## Overview

Planted inside `vendor/`. Schema-valid so it produces no per-artifact violation,
and therefore the ONLY way it can be observed is through the corpus resolution
pass — which is exactly the observable the wiring tests need.

## Requirements

REQ-001 — carry a `supports` pin to a bundle that does not exist, so discovering
this file produces a resolution violation naming it.

## Implementation

None. Fixture data.

## Verification

Asserted by the discovery and wiring tests in cmd/backstop.
