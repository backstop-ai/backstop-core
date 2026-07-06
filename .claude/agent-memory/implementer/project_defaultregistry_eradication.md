---
name: defaultregistry-eradication
description: ISSUE-027 deleted engine.DefaultRegistry/DefaultFieldContracts/engineFieldClaim; built-ins now load from an embedded base-engines pack + external go-toolchain pack
metadata:
  type: project
---

ISSUE-027 (delivered 2026-07-06) eradicated the last baked engine knowledge from the binary.

**End state (built-ins are now DATA):**
- 4 generic engines (semgrep, ast-grep, sandbox, config-file) live in an EMBEDDED base pack: `packs/base-engines/pack.yml`, embedded via `//go:embed all:packs/base-engines` (root `embed.go`, `BaseEnginesFS`), loaded through normal `pack.ParseManifest` by `pkg/baseengines.Registry()`. Each carries its `field_contract` INLINE.
- 3 Go toolchain engines (go-build, go-test, golangci) live in the EXTERNAL `~/src/projects/backstop-go-toolchain-pack/pack.yml` (durable source; installed copy disposable).
- `engine.DefaultRegistry()`, `engine.DefaultFieldContracts()`, `engineFieldClaim` map + `claimFor()` DELETED. Field-contract violations report one generic code `CLM-020-engine-field-contract` (granular per-field codes intentionally dropped).

**Layering the plan under-specified (resolved):** `ValidateManifest` got the base registry by PARAM injection (option a, ratified). But `validateEngine` is called from `ParseManifest` (many prod callers, cannot import baseengines — cycle). Resolution: `validateEngine` validates ONLY the pack's own DECLARED engines and ACCEPTS undeclared names, DEFERRING unknown-engine + allowlist enforcement to the gate. Safe: gate `resolveEngineRegistry` merges `baseengines.Registry()`, dispatch `Lookup` fail-louds, `runFindingsEngine` runs `CheckToolAllowed` (Sharp Edge 1). WHY: go-standards/self packs use `semgrep` WITHOUT declaring it — parse must accept undeclared built-ins.

**Enabling changes not in the plan:** (1) `ParseManifest` now permits an engine-only pack (no content) when `len(Engines)>0` — base pack has no rules. (2) `EngineSpec`/`parseEngineSpec` gained `crash_guard` + `exempt_from_scope_filter` yaml fields (go-build/go-test need them as pack DATA; they existed only on the binding, not the yaml surface).

**Pack separation stays non-vacuous:** `pack_separation.go` `engineCategory` reads `binding.Category` via `resolveEngineRegistry(m)` = base union `m.Engines`; real packs declare their own engines with `category`, proven by `TestGoToolchainPack_MechanismOnlyNoStandards` running the check on the REAL go-toolchain manifest. Synthetic-pack tests using bare go-engine names must set `engineRegistry = builtinTestRegistry(t)` (a cmd/backstop test helper = baseengines.Registry() + goToolchainManifest engines).
