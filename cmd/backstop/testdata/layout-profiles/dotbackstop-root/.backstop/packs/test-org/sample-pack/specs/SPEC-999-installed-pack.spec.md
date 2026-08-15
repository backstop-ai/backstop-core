---
title: "SPEC-999: Installed Pack Spec"
number: SPEC-999
created: "2026-08-14"
status: draft
schema_version: spec/v1
spec_version: 1.0.0

implementation:
  summary: An installed pack's OWN artifact — it must never be adopted as the consumer's corpus.
  subject: pkg/samplepack

verification:
  level: unit
  test_command: go test ./...
  coverage_threshold: 80
---

# SPEC-999: Installed Pack Spec

## Overview

An installed pack's own spec, planted inside .backstop/packs so the discovery
exclusion can be proven.

## Requirements

None.

## Implementation

None.

## Verification

None.
