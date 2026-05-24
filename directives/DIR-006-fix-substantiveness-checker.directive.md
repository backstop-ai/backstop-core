---
title: "Fix Substantiveness Checker"
number: DIR-006
created: "2026-04-19"
schema_version: directive/v1

directive:
  status: done
  source:
    - "SPEC-010"
---

## Description

The test substantiveness checker (gate step 4) produces 596 false positives because it flags "does not call target package" for same-package tests that call `Function()` directly instead of `package.Function()`. This also affects helper-mediated calls where a test calls a helper that calls the target.

Fix the heuristic to handle: same-package tests (no qualifier prefix), helper-mediated calls (transitive analysis), and the `package_test` external test package pattern.

This is blocking real signal from the gate — the 596 false positives drown out any real substantiveness issues.
