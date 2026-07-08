---
title: "Multi-rule ast-grep packs silently produce vacuous green: `--rule <DIR>` is rejected by ast-grep 0.43.0"
schema_version: issue/v1

issue:
  id: ISSUE-028
  title: "Multi-rule ast-grep packs silently produce vacuous green: `--rule <DIR>` is rejected by ast-grep 0.43.0"
  type: bug
  status: closed
  created: "2026-06-23"
  closed: "2026-07-08"

complexity:
  scope: contained
  uncertainty: known
  risk: critical

delivered_by: PLAN-ISSUE-028
---

# ISSUE-028: Multi-rule ast-grep packs silently produce vacuous green: `--rule <DIR>` is rejected by ast-grep 0.43.0

## Resolution

Retired the broken rule-dir input mode; multi-rule ast-grep now dispatches via config-file/--config with a pack-shipped sgconfig.yml, proven by a real un-stubbed multi-rule end-to-end test.

## Problem

The pack-engine dispatch's `InputModeRuleDir` case in `gatherEngineInputs`
(`cmd/backstop/pack_gate.go`, ~line 290) emits `--rule <RULE_DIR>` for the
`ast-grep` engine. The `ast-grep` `EngineBinding` in `DefaultRegistry`
(`pkg/pack/engine/binding.go:221–228`) declares:

```go
"ast-grep": {
    Command:   "ast-grep scan",
    InputMode: InputModeRuleDir,
    InputFlag: "--rule",
    ...
}
```

This is wrong for `ast-grep` 0.43.0. `ast-grep scan --rule` accepts exactly
**one rule FILE** — not a directory. Passing a directory fails at runtime:

```
ast-grep scan --rule <DIR>  →  "Is a directory (os error 21)"
```

The `InputModeRuleDir` dispatch also emits `--rule <dir>` once per distinct
directory when there are multiple dirs, but even a single-dir pack with one rule
**per the current code emits `--rule <dir>`** (the dir, not the file), which
fails. For the more common case of multiple rules in a single rule directory the
code deduplicates to one `--rule <dir>` emission — but that one emission is still
wrong because `--rule` requires a file. Even if two rule-dirs were present, two
`--rule` flags would also be rejected:

```
ast-grep scan --rule a.yml --rule b.yml
→ "the argument '--rule <RULE_FILE>' cannot be used multiple times"
```

### Consequence

A pack carrying more than one `ast-grep` rule cannot run through the production
`dispatchPackEngines` path. `ast-grep` exits with an OS error and produces zero
findings on stdout, which the dispatch treats as a clean (empty) result. The gate
silently produces a **vacuous green** — it does not block, does not error, and
does not report a broken-pack condition. Every finding from every `ast-grep` rule
in the pack is silently dropped.

### Why it was hidden

Every existing `ast-grep` dispatch test (`TestGateDispatch_AstGrepProofRuleEndToEnd`,
`TestGateDispatch_ReplacesSemgrepOnlyFeeder`, `TestGateDispatch_MixedEnginesNotCrossFed`
in `cmd/backstop/dispatch_astgrep_e2e_test.go`) uses a **stubbed runner**
(`fixtureRunner`) that intercepts the `ast-grep scan` command and returns canned
JSON — the `--rule <DIR>` argument never reaches real `ast-grep`. Additionally,
the existing proof pack contains only **one** `ast-grep` rule, so the defect was
not visible even in the argument construction logic.

### What exposed it

SPEC-037's substantiveness pack is the **first real multi-rule `ast-grep` pack**
in production: it carries two rules (Q1 hollow-bundle detection, Q2
referenced-symbol extraction). Attempting to wire SPEC-037's pack through
`dispatchPackEngines` against real `ast-grep` 0.43.0 produced the `os error 21`
failure above. SPEC-038's contracts pack will have the same shape (multiple
`ast-grep` rules). These packs are unrunnable until the dispatch is corrected.

### Thin-executor invariant at stake

Backstop's core invariant is: backstop runs what packs declare and speaks SARIF —
it bakes in no language or tool knowledge. A pack carrying two `ast-grep` rules
**must actually run both rules**. A dispatch that silently drops all findings
from a multi-rule pack violates this invariant just as badly as a baked-in check
would: the pack declares the rules, the executor fails to honor them, and the gate
goes green without measuring what it was supposed to measure.

## Solution

`ast-grep`'s correct multi-rule mechanism (verified working against 0.43.0) is:

```
ast-grep scan --config sgconfig.yml --json
```

where `sgconfig.yml` contains:

```yaml
ruleDirs:
  - <absolute-path-to-pack-ast-grep-rule-dir>
```

This runs **all rules** in the declared directory in a single invocation and emits
the correct JSON findings. The fix has two implementation options:

**Option A — new `InputMode` (`InputModeConfigDriven`):**
Introduce a new input mode that, at dispatch time, generates a temporary
`sgconfig.yml` referencing the pack's `ast-grep/` rule directory and emits
`--config <tmp-sgconfig.yml>` as the input arg. The `EngineBinding` for `ast-grep`
switches from `InputModeRuleDir` / `InputFlag: "--rule"` to the new mode and
`InputFlag: "--config"`. This is the cleanest model because the config-generation
logic is encapsulated in the mode, not in engine-specific dispatch code.

**Option B — fix `InputModeRuleDir` to generate an sgconfig proxy:**
Repurpose `InputModeRuleDir`: when the engine declares `InputFlag: "--config"`,
the `InputModeRuleDir` case writes a temporary `sgconfig.yml` and passes
`--config <path>` instead of `--rule <dir>`. This avoids a new mode constant but
conflates two different semantics under one mode.

Either option must update two files:
- `pkg/pack/engine/binding.go` — `DefaultRegistry` `ast-grep` entry
  (`InputMode`, `InputFlag`)
- `cmd/backstop/pack_gate.go` — `gatherEngineInputs` `InputModeRuleDir` case
  (or new mode case)

**Scope fence:** this issue covers only the dispatch fix. Two sibling substrate
blockers SPEC-037 also exposed are tracked separately and must NOT be folded in:

- ISSUE-020 — `convert` step SIGABRTs under `SandboxedRunStdout` (`jq`
  unavailable in sandbox; cluster G)
- ISSUE-027 — a backstop-owned pack cannot live under `.backstop/packs/` because
  `extra_unlocked` lock-verification fails (default-pack shipping; BUNDLE-011)

## Verification

Acceptance requires a **real, un-stubbed end-to-end test** that:

1. Constructs a pack fixture with **two distinct `ast-grep` rules** in its rule
   directory.
2. Dispatches that pack through the production `dispatchPackEngines` path with the
   **real `ast-grep` binary** (no `fixtureRunner` interception of the engine
   command).
3. Asserts findings from **both** rules arrive in the SARIF output — not one, not
   zero, both.

A test that stubs `ast-grep` or asserts findings from only one rule does not close
this issue. The recurring pattern of pack-provisioning integration gaps (unit green,
wiring broken — cf. SPEC-008, ISSUE-012) means stubbed coverage cannot be accepted
as proof here.

The existing `TestGateDispatch_AstGrepProofRuleEndToEnd` (CLM-030) tests the
single-rule proof pack through a fake runner and remains valid as a fast unit
path; it does not substitute for the new integration test.

## References

- `cmd/backstop/pack_gate.go` ~line 290 — `gatherEngineInputs` `InputModeRuleDir`
  case (emits `--rule <dir>`, the broken emission)
- `pkg/pack/engine/binding.go:221–228` — `DefaultRegistry` `ast-grep` binding
  (`InputMode: InputModeRuleDir`, `InputFlag: "--rule"`)
- `cmd/backstop/dispatch_astgrep_e2e_test.go` — existing tests all use
  `fixtureRunner`; none reach real `ast-grep`
- SPEC-035 — the pack-engine dispatch + `EngineBinding`/`InputMode` model this
  corrects
- SPEC-037 — first real multi-rule `ast-grep` pack; the consumer that exposed
  this defect
- ISSUE-020 — sibling blocker: `convert` under sandbox SIGABRTs (`jq`); cluster G
- ISSUE-027 — sibling blocker: backstop-owned pack under `.backstop/packs/`
  triggers `extra_unlocked`; BUNDLE-011
