---
title: "Extract Go Standards Pack"
number: DIR-005
created: "2026-04-19"
schema_version: directive/v1

directive:
  status: done
  source:
    - "BUNDLE-004"
    - "SPEC-012"
  completed: "2026-04-23"
---

## Description

Extract the embedded Go standards pack (SPEC-012) from backstop-core into its own standalone pack repo. This is the first real pack under the new manifest model — it proves the pack system works end-to-end. The pack gets its own `pack.yml`, rules/, fixtures/, and CI validation via the shared workflow.

After extraction, backstop-core becomes language-agnostic — it ships the gate and the pack infrastructure, not Go-specific opinions. The Go pack becomes the "core tap" that `backstop init` wires by default for Go projects.

## Completed

Pack repo: ~/src/projects/backstop-go-pack (github.com/bmanson/backstop-go-pack)
- 14 rules: 9 core (correctness), 4 security, 1 test (style)
- 32 fixtures: 14 valid, 14 invalid, 4 bypass-attempt (security)
- pack check: all 6 phases pass
- pack test: all 6 phases pass including fixture execution
- pack add + gate: installs and enforces in consuming project
