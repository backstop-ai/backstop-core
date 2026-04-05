---
title: "SPEC-002: Invalid Test Spec"
number: SPEC-002
created: "2026-04-01"
status: draft
schema_version: spec/v1
spec_version: 1.0.0

implementation:
  summary: A spec with missing required sections.
  package: pkg/test

verification:
  level: unit
  test_command: go test ./...
  coverage_threshold: 90
---

# SPEC-002: Invalid Test Spec

## Overview

This spec is intentionally missing required sections to trigger violations.
