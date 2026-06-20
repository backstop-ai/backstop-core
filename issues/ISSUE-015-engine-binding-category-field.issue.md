---
title: "Mechanism/opinion classification hardcodes engine sets in two places — add EngineBinding.Category as single source of truth"
schema_version: issue/v1

issue:
  id: ISSUE-015
  title: "Mechanism/opinion classification hardcodes engine sets in two places — add EngineBinding.Category as single source of truth"
  type: technical-debt
  status: open
  created: "2026-06-20"

complexity:
  scope: contained
  uncertainty: known
  risk: moderate
---

# Mechanism/opinion classification hardcodes engine sets in two places — add EngineBinding.Category as single source of truth

## Problem

SPEC-034 Phase 6 introduced pack separation enforcement in `cmd/backstop/pack_separation.go`. The enforcer classifies each pack rule by its declared engine name to determine whether the rule belongs to the mechanism pack (`backstop/go-toolchain`) or the opinion pack (`backstop/go-standards`). That classification is implemented as two switch functions:

- `isToolchainMechanismEngine` (line 29–36): hardcodes `go-build`, `go-test`, `golangci` as mechanism engines.
- `isStandardsOpinionEngine` (line 41–48): hardcodes `semgrep`, `ast-grep` as opinion engines.

Both functions MIRROR the engine registrations in `pkg/pack/engine/binding.go` (`DefaultRegistry`, lines 141–216) but are entirely separate from them. There is no programmatic link between the two locations.

**Drift risk:** adding a new engine to `DefaultRegistry` does NOT update `pack_separation.go`. The enforcer will silently mis-classify any rule bound to that engine — treating it as neither mechanism nor opinion, making it invisible to both boundary checks. Because `classifyPack` (line 58) gates the entire violation-detection pass, a mis-classified engine produces a structural false green: packs that mix the new engine with other engines pass the separation check even when they should be flagged.

**Root cause:** engine category ("is this engine mechanism or opinion?") is a property of the engine, but it lives outside the engine's binding record. The engine registry (`EngineBinding` struct) has no field for it. Classification is reconstructed at the call site via hardcoded switch statements, creating a second source of truth that can silently diverge from the registry.

**Phase 6 scope exclusion:** the comment on `isToolchainMechanismEngine` (line 28) explicitly notes these sets "mirror the SPEC-034 EngineBinding records the bridge registers" — acknowledging the duplication. Moving the category into `EngineBinding` was deliberately out of scope for the pure-separation task. This issue captures the follow-on work.

### Affected locations

| Location | What it does |
|---|---|
| `cmd/backstop/pack_separation.go:29–36` | `isToolchainMechanismEngine` — hardcoded mechanism engine set |
| `cmd/backstop/pack_separation.go:41–48` | `isStandardsOpinionEngine` — hardcoded opinion engine set |
| `pkg/pack/engine/binding.go:76–108` | `EngineBinding` struct — where the category field should live |
| `pkg/pack/engine/binding.go:141–216` | `DefaultRegistry` — where each engine's category should be declared |

## Solution

Add a `Category` field (or equivalent) to `EngineBinding` in `pkg/pack/engine/binding.go`. Reasonable shape:

```go
// EngineCategory classifies an engine as toolchain mechanism or coding-standards
// opinion for the pack-separation enforcer.
type EngineCategory int

const (
    EngineCategoryUnknown  EngineCategory = iota
    EngineCategoryMechanism               // native toolchain: go-build, go-test, golangci
    EngineCategoryOpinion                 // swappable coding standards: semgrep, ast-grep
)

// On EngineBinding:
Category EngineCategory
```

Each `DefaultRegistry` entry sets its `Category` at declaration time. `pack_separation.go` then drops both switch functions and instead calls `registry.Lookup(r.Engine)` to read `binding.Category`, so the enforcer derives mechanism/opinion classification entirely from the registry — no separate classification logic, no drift surface.

Adding a future engine (e.g. `eslint`, `ruff`) requires one declaration in `DefaultRegistry` with an explicit `Category`; the separation enforcer picks it up automatically with no edit needed in `pack_separation.go`.

## References

- `cmd/backstop/pack_separation.go` — `isToolchainMechanismEngine` (lines 29–36), `isStandardsOpinionEngine` (lines 41–48), `classifyPack` (lines 58–72), `packSeparationViolations` (lines 81–116)
- `pkg/pack/engine/binding.go` — `EngineBinding` struct (lines 76–108), `DefaultRegistry` (lines 141–216)
- SPEC-034 REQ-007 — the mechanism/opinion boundary requirement that `pack_separation.go` implements
