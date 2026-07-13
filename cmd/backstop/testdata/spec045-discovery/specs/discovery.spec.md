---
title: "Discovery E2E Spec"
number: DISC-001
created: "2026-01-01"
status: implemented
schema_version: spec/v1
spec_version: 1.0.0

implementation:
  summary: discovery e2e
  package: app/widget

verification:
  level: integration
  test_command: bun test
  coverage_threshold: 80

requirements:
  - id: REQ-001
    text: req
    supports: x:REQ-001

claims:
  - id: CLM-001
    requirement: REQ-001
    text: the TS test is discovered via the bun globs + patterns
    tests:
      - renders the widget
  - id: CLM-002
    requirement: REQ-001
    text: the Go test is discovered via the go globs + patterns
    tests:
      - TestDiscoveredGoTest
---

# Discovery E2E Spec
