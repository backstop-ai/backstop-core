---
title: "SPEC-901: Legitimate Corpus Spec"
number: SPEC-901
created: "2026-08-16"
status: implemented
schema_version: spec/v1
spec_version: 1.0.0

implementation:
  summary: the one artifact in this fixture that is genuinely corpus
  subject: cmd/backstop

verification:
  level: integration
  test_command: go test ./cmd/backstop/
  coverage_threshold: 80

requirements:
  - id: REQ-001
    text: discovery returns this file and none of the planted dependency-tree files

claims:
  - id: CLM-001
    requirement: REQ-001
    text: the legitimate corpus is discovered
    tests:
      - TestArtifactDiscoveryE2E_VendorAndNodeModulesExcludedByPackDeclaration

contracts:
  - file: cmd/backstop/artifact_discover.go
    provides:
      - name: DiscoverArtifacts
        kind: function
        signature: "func DiscoverArtifacts(root artifact.Root, typeFilters []string, nonCorpus artifact.NonCorpusDirs) ([]DiscoveredArtifact, error)"
---

# SPEC-901: Legitimate Corpus Spec

## Overview

The single artifact in this ISSUE-122 fixture that is genuinely part of the
consumer's corpus. Every other artifact-shaped file here is planted inside a
dependency tree a pack declares.

## Requirements

REQ-001 — discovery returns this file and none of the planted dependency-tree
files, so an assertion can distinguish "the walk worked" from "the walk found
nothing".

## Implementation

None. This is fixture data consumed by the cmd/backstop e2e tests.

## Verification

Asserted by the discovery and wiring tests in cmd/backstop.
