---
title: "End-to-End Pack Smoke Test"
number: DIR-009
created: "2026-04-19"
schema_version: directive/v1

directive:
  status: queued
  source:
    - "BUNDLE-004"
    - "BUNDLE-005"
    - "BUNDLE-006"
---

## Description

Verify the pack system works end-to-end with real packs against a real project. This is the integration test that proves the unit-tested pieces actually work together.

Scenarios:
- `backstop pack add <git-url>` clones, validates, installs a real pack from a git repo
- `backstop pack check` validates the installed pack
- `backstop pack test` runs fixtures
- `backstop gate` enforces the pack's rules on real code
- `backstop pack remove` cleanly uninstalls
- `backstop pack update` detects new versions
- Lock verification catches tampered packs

Use the prototype packs extracted from Slotly as the test subject. This should run as part of the smoke test suite alongside the existing gate smoke tests.
