---
title: "Gate Test Verification Runs Full Package, Not a Plan's Narrow -run Filter"
schema_version: issue/v1

issue:
  id: ISSUE-066
  title: "Gate Test Verification Runs Full Package, Not a Plan's Narrow -run Filter"
  type: technical-debt
  status: open
  created: "2026-07-17"

complexity:
  scope: contained
  uncertainty: known
  risk: moderate
---

# Gate Test Verification Runs Full Package, Not a Plan's Narrow -run Filter

## Problem

A spec/plan `test_command` commonly scopes to `go test ... -run '<claim-name-pattern>'` to name
the tests that prove that artifact's claims. The gate's test verification honors that filter, so
a regression in ANY test whose name does NOT match the pattern stays invisible: the scoped run is
green while the full package is red. The narrow `-run` filter — meant only to MAP tests to claims
— silently doubles as the bound on WHAT MUST PASS, which it should never do.

Discovered in ISSUE-064: `TestWiring_NoBakedAnalyzerDelegateInvoked` was broken by the routing
change (it fed a synthetic finding into the migrated routing path without the new role property)
and failed deterministically under `go test ./cmd/backstop/...`, but matched none of the
`test_command`'s `-run 'Substantiveness|ToolchainStackLabel|IsToolchainPack|ByDeclaration|SelfRule'`
patterns. Every mechanical check — the scoped mandated-test run AND the gate — reported green; the
regression was only visible via an unfiltered `go test`, and (see ISSUE-067) was further masked in
the gate as an opaque engine crash.

## Root cause

Two distinct concerns are conflated onto one `-run` filter: (a) "which tests prove THIS artifact's
claims" (a subset, for claim-mapping / substantiveness) and (b) "which tests must pass for the gate
to be green" (the full package(s) any changed code lives in). (a) is legitimately a subset; (b)
must never be. The gate currently derives (b) from (a).

## Direction (to be specified)

The gate's test step must run the FULL test package(s) in the change's scope (e.g. `go test` over
each touched package with no `-run` filter), independent of the plan's claim-mapping filter. The
`-run`/mandated-test-name mapping stays as the claim→test evidence link (test_verification /
substantiveness), but a green gate must require the whole package green, not just the mapped subset.
Evaluate whether this is enforced in the test-verification step, the toolchain go-test engine, or
both.

## Notes / references

- Surfaced by ISSUE-064's impl-review. Sibling to ISSUE-067 (the same regression was ALSO masked at
  the gate layer by the go-test engine's opaque-crash reporting) — the two failures compounded: a
  narrow filter hid it from the mandated run, and an opaque crash hid it from the gate. Either fix
  alone would have surfaced it.
