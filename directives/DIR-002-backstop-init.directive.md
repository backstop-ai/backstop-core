---
title: "backstop init Command"
number: DIR-002
created: "2026-04-19"
schema_version: directive/v1

directive:
  status: queued
  source:
    - "BUNDLE-003"
---

## Description

Implement `backstop init` — the single command that takes a consuming project from zero to first value. Scaffolds `.backstop/` directory structure, creates `backstop.yml` and `backstop.lock` at root, detects language, auto-installs dependencies (semgrep, golangci-lint, ruff), wires the default language pack, runs the first gate, and captures the result as a baseline presented as observation ("here's what we noticed") not judgment.

Target: under 2 minutes from install to first useful output. Zero manual config steps.

Depends on DIR-001 (release workflow — users need the binary) and DIR-009 (pack smoke test — init wires packs, packs need to work).
