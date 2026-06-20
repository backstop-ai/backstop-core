---
title: "pack_engines gate step is not diff-scoped — rule-fed findings engines scan the whole repository"
schema_version: issue/v1

issue:
  id: ISSUE-010
  title: "pack_engines gate step is not diff-scoped — rule-fed findings engines scan the whole repository"
  type: bug
  status: open
  created: "2026-06-20"

complexity:
  scope: contained
  uncertainty: known
  risk: moderate
---

# pack_engines gate step is not diff-scoped — rule-fed findings engines scan the whole repository

## Problem

`backstop gate` computes a diff scope on every run. The header it prints confirms this:

```
Gate running against 15 changed files (use --all for full sweep)
```

All other gate steps (e.g. `code_check`) honor that 15-file scope and pass. The `pack_engines` step does not — it returns **245 violations across 20+ files that are not in the changed set**, of which only ~22 touch the two files actually changed (`cmd/backstop/gate.go`: 9, `cmd/backstop/pack_gate.go`: 13).

**Files emitting violations entirely outside the diff scope (sample):**

| File | Violations |
|---|---|
| pkg/check/check.go | 24 |
| pkg/pack/distribution/upgrade.go | 14 |
| pkg/pack/distribution/update.go | 13 |
| pkg/pack/manifest.go | 11 |
| pkg/validate/standard.go | 10 |
| pkg/gate/result.go | 10 |
| pkg/pack/distribution/add.go | 10 |
| ... | ... |

**Root cause — `dispatchPackEngines` receives no scope parameter.**

The call site in `cmd/backstop/gate.go` line 398:

```go
violations, err := dispatchPackEngines(dispatchPacks, filepath.Join(projectRoot, ".backstop", "packs"), projectRoot, runner)
```

The function signature (`cmd/backstop/pack_gate.go` line 130):

```go
func dispatchPackEngines(packs []*pack.Manifest, packDir, projectRoot string, runner check.CommandRunner) ([]gate.Violation, error)
```

No changed-file list is threaded in. Inside `runFindingsEngine` (lines 287–291), when the engine is not project-wide-toolchain (the `else` branch), the scan target is unconditionally set to `projectRoot`:

```go
if binding.ScopeKind == engine.ScopeKindProjectWide && binding.ProjectTarget != "" {
    cmdArgs = append(cmdArgs, binding.ProjectTarget)
} else {
    cmdArgs = append(cmdArgs, projectRoot)   // scans the whole repo
}
```

This is the correct branch for rule-fed findings engines (semgrep, ast-grep) — except it must be scoped to the gate's changed-file list, not `projectRoot`. The `if` branch (project-wide toolchain: `go build ./...`, `go test ./...`, `golangci-lint run ./...`) must stay project-wide by design (RDC-3: a change to `a.go` that breaks unchanged `b.go` must still fail the gate).

**The scope leak is specific to rule-fed findings engines in the `else` branch.**

## Impact

The `gate-on-implement` hook requires `backstop gate` to be green before an implementation can proceed. With `pack_engines` returning 245 violations from the pre-existing repo-wide backlog (emitted by the `backstop/go-standards` pack against untouched files), no diff-scoped feature work can ever turn the gate green — even when every changed file is clean.

**This is currently blocking implementation of PLAN-SPEC-034** (native toolchain engine cutover), which is the active implementation track for BUNDLE-010.

The issue is directly related to the gate scope & coverage debt previously fixed in mechsuit and slated for re-implementation in backstop-core (captured in project memory under "Gate scope & coverage").

## References

- `cmd/backstop/pack_gate.go` line 130 — `dispatchPackEngines` signature (no scope param)
- `cmd/backstop/pack_gate.go` lines 287–291 — `runFindingsEngine`, the unconditional `projectRoot` scan target
- `cmd/backstop/gate.go` line 398 — call site, scope not passed through
- ISSUE-004 — parallel pattern: `pkg/check` diff-scope bug fixed by threading the changed-file list through
- "Gate scope & coverage" project memory — known mechsuit-to-backstop-core port debt; both halves (untracked file inclusion and per-file enforcement) relevant here
