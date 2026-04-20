---
title: "Baseline Implementation"
number: DIR-003
created: "2026-04-19"
schema_version: directive/v1

directive:
  status: queued
  source:
    - "BUNDLE-007"
    - "SPEC-010"
---

## Description

Implement gate step 7 (baseline comparison) and the CI baseline generation workflow. CI runs `backstop gate` post-merge and publishes the violation set as an immutable baseline artifact. Locally, `backstop gate` auto-pulls the latest baseline with TTL-based caching and reports differentials ("3 new violations beyond baseline") instead of absolute counts.

Includes: `backstop baseline pull` command, `.backstop/baseline.json` caching, TTL logic (default 15 minutes), GitHub Actions artifact publishing, structural diff algorithm for violation identity.

Depends on DIR-001 (release workflow — CI must exist to generate baselines).
