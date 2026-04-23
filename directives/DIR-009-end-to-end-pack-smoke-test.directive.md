---
title: "End-to-End Pack Smoke Test"
number: DIR-009
created: "2026-04-19"
schema_version: directive/v1

directive:
  status: done
  source:
    - "BUNDLE-004"
    - "BUNDLE-005"
    - "BUNDLE-006"
  spec: SPEC-017
  completed: "2026-04-23"
---

## Description

Verify the pack system works end-to-end with real packs against a real project. This is the integration test that proves the unit-tested pieces actually work together.

## Verified scenarios

- `backstop pack check` on slotly-go-pack — 5 phases pass ✅
- `backstop pack test` on slotly-go-pack — 6 phases pass ✅
- `backstop pack add <local-path>` — copies to .backstop/packs/, writes backstop.yml + backstop.lock ✅
- `backstop gate` with pack enforcement — lock verification, rule merge, validators all pass ✅
- `backstop pack remove` — cleanly removes from backstop.yml, .backstop/packs/ ✅

## Issues found and fixed during smoke testing

- ISSUE-001: backstop.yml schema mismatch between config and distribution (fixed)
- main.go silently swallowing all CLI error messages (fixed)
- pack add not copying local packs to .backstop/packs/ (fixed)
- pack.yml using `rule:` instead of `rule_path:` field name (fixed in pack)

## Not yet verified

- `pack add` from git URL (no remote pack repo published yet)
- `pack update` / `pack upgrade` with version changes
- tool_config enforcement through gate (unit tested, not integration tested)
- Multiple packs composing in gate (unit tested, not integration tested)

## Test infrastructure

- slotly-go-pack: ~/src/projects/slotly-go-pack (standalone pack repo)
- backstop-test-project: ~/src/projects/backstop-test-project (consuming project)
