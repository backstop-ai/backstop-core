---
title: "Extract Go Standards Pack"
number: DIR-005
created: "2026-04-19"
schema_version: directive/v1

directive:
  status: queued
  source:
    - "BUNDLE-004"
    - "SPEC-012"
---

## Description

Extract the embedded Go standards pack (SPEC-012) from backstop-core into its own standalone pack repo. This is the first real pack under the new manifest model — it proves the pack system works end-to-end. The pack gets its own `pack.yml`, rules/, fixtures/, and CI validation via the shared workflow.

After extraction, backstop-core becomes language-agnostic — it ships the gate and the pack infrastructure, not Go-specific opinions. The Go pack becomes the "core tap" that `backstop init` wires by default for Go projects.

Depends on DIR-004 (packs must be loadable by the gate) and DIR-009 (end-to-end smoke test).
