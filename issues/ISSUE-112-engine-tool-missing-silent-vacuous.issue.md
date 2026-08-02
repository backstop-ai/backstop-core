---
title: "Missing engine tool + no CrashGuard = silently empty SARIF, vacuous pack_engines pass, and misleading downstream join violations"
schema_version: issue/v1

issue:
  id: ISSUE-112
  title: "Missing engine tool + no CrashGuard = silently empty SARIF, vacuous pack_engines pass, and misleading downstream join violations"
  type: bug
  status: open
  created: "2026-07-29"
---

# Missing engine tool: silent vacuous pass + misleading joins

## Problem

A findings engine whose tool is ABSENT from PATH fails in the worst possible way when the binding has
no CrashGuard: the runner's non-fatal runErr is discarded, the empty stdout flows through convert
(jq over empty stdin emits nothing), the LENIENT SARIF parse reads zero findings, and pack_engines
PASSES — while every consumer of that evidence lies downstream. Observed live (bclabs-portal, first
CI run on a GitHub runner, 2026-07-29): ast-grep absent -> typescript-substantiveness produced empty
SARIF -> pack_engines green -> the test_substantiveness join starved -> 397 false "does not call
package" violations attributed to innocent tests. Diagnosis took hours because nothing named the
missing tool.

Two aggravators:
1. The assume-present fail-loud (pack_gate_provision.go) EXEMPTS provision-declared tools ("auto-
   provisioned") — but provision is a TRUST ALLOWLIST pin only; no code path installs anything. So
   provision-pinned tools (ast-grep, semgrep) get NEITHER install NOR presence check.
2. Non-CrashGuard engines treat every non-zero/failed run as finding-free (runErr discarded) —
   an exec-not-found error is indistinguishable from a clean scan.

## Direction

- Presence-check provision-pinned tools exactly like assume-present ones (fail loud naming the tool
  + the install expectation), since backstop never auto-provisions; or implement provisioning.
- An *exec.Error-class failure (binary could not start) must fail loud for EVERY engine regardless
  of CrashGuard — a missing tool is never a finding-free pass (the packval executor already does
  this; the gate dispatch path does not).

## Notes

- Repro: any repo + typescript-substantiveness on a PATH without ast-grep.
- Sibling diagnosability issue: zero-match classification refusal (filed alongside).
- Portal-side workaround shipped: explicit gitleaks+ast-grep installs in its CI workflow.
