---
title: "Wire Packs Into Gate"
number: DIR-004
created: "2026-04-19"
schema_version: directive/v1

directive:
  status: queued
  source:
    - "BUNDLE-004"
    - "BUNDLE-005"
---

## Description

The gate currently runs the old code check pipeline without loading installed packs. Pack rules, tool_config, and custom validators need to be loaded from `.backstop/packs/` and merged into the gate's check pipeline at runtime.

Includes: pack loader reads backstop.yml packs list, resolves installed packs from `.backstop/packs/`, merges semgrep rules into the semgrep pass, merges tool_config into lint pass, executes layer 3 validators, runs `VerifyLock` as part of gate step 1. Consumer never thinks about packs — they just run `backstop gate` and pack enforcement is included.
