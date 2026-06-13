---
title: "build/test passes hardcode ./... — no exclusion mechanism for intentionally-broken trees"
schema_version: issue/v1

issue:
  id: ISSUE-007
  title: "build/test passes hardcode ./... — no exclusion mechanism for intentionally-broken trees"
  type: enhancement
  status: open
  created: "2026-06-11"

complexity:
  scope: contained
  uncertainty: known
  risk: moderate
---

# build/test passes hardcode ./... — no exclusion mechanism for intentionally-broken trees

## Problem

buildExecutor and testExecutor run `go build ./...` / `go test ./...`
against the whole module with no exclusion mechanism. Repos legitimately
contain trees that must not gate the build: this repo's `prototype/`
holds pack rule fixtures that are *intentionally* uncompilable (missing
third-party modules are part of the fixture design). The first live
`code check --all` run correctly surfaced them as build violations:

```
prototype/code-pack/fixtures/rules/slotly-004/good-sig-verify.go
  [build] no required module provides package github.com/slack-go/slack...
```

These are true positives from the tool's perspective and permanent noise
from the project's perspective — the gate can never go green while they
gate the build.

**Update 2026-06-12:** the motivating case is gone — `prototype/` was
deleted (it was git-ignored scratch, never tracked), dropping the dogfood
baseline from 7 violations to 3 and making `go build ./...` clean
module-wide. This issue is now demoted to the generic mechanism: any
backstop-adopting repo with fixture trees, generated code, or vendored
exceptions will eventually need `exclude_paths`, but nothing in this repo
requires it today. Defer until a real consumer does.

## Fix sketch

Add an exclusion surface for the build/test passes — likely
`enforcement.exclude_paths` in backstop.yml (explicit, prompts-are-vibes
compliant) — translated to package-list filtering (`go list ./...` minus
excluded prefixes) rather than naked `./...`. Waivers are the wrong tool:
these aren't violations to forgive individually but trees that are out of
enforcement scope by design. Interaction to decide: should excluded paths
also be excluded from semgrep/lint routing, or build/test only?

## References

- pkg/check/check.go — buildExecutor/testExecutor (hardcoded ./...)
- Discovered during ISSUE-005 TASK-012 dogfood reckoning, 2026-06-11
- Related: ISSUE-003 (toolchain registry — exclusion config should be
  designed stack-generic, not Go-specific)
