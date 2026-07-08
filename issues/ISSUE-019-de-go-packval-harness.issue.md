---
title: "De-Go the packval pack-validation harness (cluster E)"
schema_version: issue/v1

issue:
  id: ISSUE-019
  title: "De-Go the packval pack-validation harness (cluster E)"
  type: technical-debt
  status: closed
  created: "2026-06-21"
  closed: "2026-07-08"

delivered_by: PLAN-ISSUE-019

complexity:
  scope: contained
  uncertainty: known
  risk: moderate
---

# De-Go the packval pack-validation harness (cluster E)

## Resolution

De-Go'd the packval harness so engine dispatch is derived from pack manifest data, not
baked semgrep/golangci/go-mod-tidy commands.

## Problem

`pkg/packval/` is the pack-authoring validation harness — a separate execution path
from the gate's engine-dispatch (`cmd/backstop/pack_gate.go`, which SPEC-035 is
fixing). packval still bakes tool-specific and Go-specific knowledge directly into
Go code. This violates the same invariant the eradication backlog targets everywhere
else: backstop must know zero tool names.

### Site 1 — tool-named interface method and hardcoded semgrep invocation

`pkg/packval/executor.go` lines 10, 25–26, 35–36:

```go
// Line 10 — tool name encoded in the interface contract itself
RunSemgrep(packDir, ruleFile, fixturePath string) (ExecutionResult, error)

// Lines 25–26 — hardcoded exec.Command baking the tool name
func (d *DefaultExecutor) RunSemgrep(packDir, ruleFile, fixturePath string) (ExecutionResult, error) {
    cmd := exec.Command("semgrep", "--config", ruleFile, fixturePath)

// Lines 35–36 — golangci-lint hardcoded in switch
case "golangci-lint":
    cmd = exec.Command("golangci-lint", "run", "--config", configFile, fixturePath)
```

The `FixtureExecutor` interface names a specific tool (`RunSemgrep`) rather than an
engine-dispatch abstraction. `RunToolConfig` avoids this at the interface level, but
its `DefaultExecutor` implementation dispatches on the tool string in a `switch` —
the same anti-pattern in a thinner disguise.

### Site 2 — semgrep call sites and Go-specific invocations in phase3

`pkg/packval/phase3.go`:

- Lines 51 and 65 — two `executor.RunSemgrep(...)` call sites that assume the engine
  is semgrep by name.
- Line 214 — `exec.Command("go", "mod", "tidy")` hardcoded inside `goModTidyTempCopy`.
  The `tool_config` fixture path runs a go-module-tidy step before invoking the tool
  config executor. This is Go-specific prep baked into the harness rather than
  pack-declared setup.
- Lines 171 and 184 — the skeleton scaffold check asserts presence of `_test.go` files
  containing `func Test`. This is a Go test convention encoded as a structural
  invariant for all packs, regardless of language.

### Site 3 — language gate and Language field

`pkg/packval/phase1.go` lines 32–36:

```go
if pack.Language == "" {
    res.Errors = append(res.Errors, ValidationError{...})
} else if pack.Language != "go" {
    res.Errors = append(res.Errors, ValidationError{..., Message: "unsupported language"})
}
```

packval silently rejects any pack whose `language` field is not `"go"`. This makes
the harness structurally incapable of validating a TypeScript, Python, or shell pack
— a hard ceiling that contradicts the multi-language roadmap.

`pkg/packval/manifest.go` line 13 — the `PackManifest.Language` field drives this
gate. The field itself is not the problem; the hard-wired `!= "go"` guard is.

### Why this is the same root cause

All three sites share a single root defect: the harness has opinions about tools and
languages that should be pack-declared data. The gate engine dispatch (`pkg/pack/engine`,
`cmd/backstop/dispatchPackEngines`) already models engines as data — each `EngineBinding`
declares the command to run and produces SARIF. packval predates that model and was
never refactored to use it.

## Solution

packval should converge onto the engine model SPEC-035 establishes rather than
inventing a parallel de-Go adapter.

**Design direction:**

1. Replace `RunSemgrep` in the `FixtureExecutor` interface with a generic
   `RunEngine(packDir string, binding EngineBinding, targets []string)` (or equivalent)
   that reuses the `EngineBinding` type from `pkg/pack/engine`. The engine command and
   any required flags are fully pack-declared; packval provides no tool knowledge.

2. Replace the `golangci-lint` switch in `RunToolConfig` with the same generic engine
   dispatch path. `tool_config` entries already declare their `tool` field — that
   declaration should map to a trusted-tool allowlist lookup, not a hard-coded switch.

3. Remove `goModTidyTempCopy` (the Go-specific pre-flight in `phase3.go:214`). If a
   pack requires module tidy before fixture execution, it declares that as a setup
   command on its `EngineBinding`; packval runs it generically or the pack's own
   scaffold handles it.

4. Generalize the skeleton scaffold check (phase3.go:171,184) to be pack-language-aware.
   The convention for a valid scaffold skeleton should come from the pack manifest (e.g.,
   a `test_indicator` field), not from a Go-hardwired `_test.go` / `func Test` scan.

5. Remove the `pack.Language != "go"` hard gate in `phase1.go:34`. The `language` field
   may remain for documentation, but the harness must not reject packs based on it.

**Dependency:** this work follows SPEC-035 and must be planned and implemented after
SPEC-035 lands. SPEC-035 establishes the shared `EngineBinding` substrate and the
SARIF contract that packval will reuse. Building on top of SPEC-035 avoids duplicating
the engine model in the harness.

## Non-scope

The following are explicitly excluded from this issue:

- **`pkg/packval/sandbox.go` `sandbox-exec`** — the macOS-only / Linux-no-op sandbox
  is cluster G, a security-elevated issue tracked separately.
- **Gate-path engine bindings / `CheckTypeSemgrep` / `DefaultRegistry`** — covered by
  SPEC-035 (cluster A).
- **Vestigial in-process `pkg/check/semgrep.go` executor / `EnsureSemgrep` / dead
  `.standard.md` validator** — covered by ISSUE-018 (clusters B and F). ISSUE-018
  targets `pkg/check/` (the in-process gate-path semgrep executor and config plumbing);
  this issue targets `pkg/packval/` (the pack-validation harness). These are distinct
  code paths: neither supersedes the other. ISSUE-018 lists `RunSemgrep` as a deletion
  target scoped to `pkg/packval/executor.go:25–30`; this issue replaces that narrow
  deletion with the full engine-convergence redesign for the packval harness. Both issues
  are required; ISSUE-019 is the cluster-E sibling of ISSUE-018's clusters B/F.

## References

- `pkg/packval/executor.go:10` — `FixtureExecutor` interface, tool-named `RunSemgrep` method
- `pkg/packval/executor.go:25–26` — `DefaultExecutor.RunSemgrep`, hardcoded `exec.Command("semgrep", "--config", ...)`
- `pkg/packval/executor.go:35–36` — `RunToolConfig` switch `case "golangci-lint":`
- `pkg/packval/phase3.go:51` — `executor.RunSemgrep(...)` call site (positive fixture)
- `pkg/packval/phase3.go:65` — `executor.RunSemgrep(...)` call site (negative fixture)
- `pkg/packval/phase3.go:214` — `exec.Command("go", "mod", "tidy")` in `goModTidyTempCopy`
- `pkg/packval/phase3.go:171,184` — `_test.go` / `func Test` skeleton assumption
- `pkg/packval/phase1.go:32–36` — `pack.Language != "go"` hard gate
- `pkg/packval/manifest.go:13` — `PackManifest.Language` field
- SPEC-035 — establishes `EngineBinding`, trusted-tool allowlist, SARIF contract that this issue will build on
- ISSUE-018 — clusters B/F (vestigial in-process semgrep executor in `pkg/check/` + legacy standards validator); sibling cluster-B/F deletion, distinct from this issue's cluster-E redesign of `pkg/packval/`; both converge onto SPEC-035's engine model
- Thin-executor eradication backlog (2026-06-20 audit) — cluster E designation
