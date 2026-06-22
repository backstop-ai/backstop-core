---
title: "Eradicate baked engine defaults: move DefaultRegistry + DefaultFieldContracts into pack data"
schema_version: issue/v1
number: "027"

issue:
  id: ISSUE-027
  title: "Eradicate baked engine defaults: move DefaultRegistry + DefaultFieldContracts into pack data"
  type: technical-debt
  status: open
  created: "2026-06-22"

complexity:
  scope: cross-cutting
  uncertainty: exploratory
  risk: moderate
---

# Eradicate baked engine defaults: move `DefaultRegistry` + `DefaultFieldContracts` into pack data

## Problem

SPEC-035 shipped pack-declared engine bindings and the trusted-tool allowlist (Phase 1–5). For OQ-1 it chose **option (i)**: the 7 baked bindings in `DefaultRegistry()` and the name-keyed contracts in `DefaultFieldContracts()` / the `engineFieldClaim` map remain as an overridable baked fallback. That deferral was deliberate — the mechanism for a pack to own built-in definitions did not yet exist. The deferred work is now this issue.

### What is baked and where

**`pkg/pack/engine/binding.go` — `DefaultRegistry()`** (7 bindings):

| Engine | Nature |
|---|---|
| `semgrep` | Generic opinion engine — appropriate in a base/default ENGINE pack |
| `ast-grep` | Generic opinion engine — appropriate in a base/default ENGINE pack |
| `sandbox` | Generic none-input engine — appropriate in a base/default ENGINE pack |
| `config-file` | Generic config-input shape — appropriate in a base/default ENGINE pack |
| `go-build` | Go-toolchain mechanism — belongs in a `go-toolchain` pack |
| `go-test` | Go-toolchain mechanism — belongs in a `go-toolchain` pack |
| `golangci` | Go-toolchain mechanism — belongs in a `go-toolchain` pack |

The last three (`go-build`, `go-test`, `golangci`) each carry Go-specific commands (`go build`, `go test`, `golangci-lint run`) and Go-specific Convert scripts (`scripts/build-to-sarif.sh`, `scripts/test-to-sarif.sh`). Baking Go toolchain bindings into the binary is a zero-baked-LANGUAGE violation independent of the general engine-defaults problem.

**`pkg/pack/engine/fieldcontract.go` — `DefaultFieldContracts()`**: 4 baked engine name → `FieldContract` entries (semgrep, ast-grep, sandbox, config-file).

**`pkg/pack/validate_manifest.go` — `engineFieldClaim` map**: 16 baked `(engine|field|kind) → CLM-NNN` entries. Currently used in `claimFor()` as the fallback claim code when the pack-declared `FieldContract` on a binding carries no entry.

### Four production call sites to migrate

1. `pkg/pack/manifest.go:419` (`validateEngine`) — fallback `engine.DefaultRegistry().Lookup(name)` when the engine is not in the pack-declared engines block.
2. `pkg/pack/validate_manifest.go:237` (`resolveEngineRegistryForValidation`) — seeds the validation registry with `engine.DefaultRegistry()` before overlaying pack-declared engines.
3. `pkg/pack/validate_manifest.go:254` (`contractForEngine`) — calls `engine.DefaultFieldContracts()[name]` when the binding carries no declared `FieldContract`.
4. The `engineFieldClaim` map and `claimFor()` at `pkg/pack/validate_manifest.go:130–158` — baked claim codes for 4 engines × field+kind combinations; consumed by `validateEngineFields`.

### Why this is a violation of the zero-baked-checks invariant

From CLAUDE.md (first principle):

> Backstop bakes in ZERO language/tool knowledge, for ANY language. … New language = a new pack, never new CLI code. A baked language/tool branch is a defect to eradicate, never to extend.

`DefaultRegistry()` is baked Go code that knows `semgrep`'s flags, `ast-grep`'s converter path, and three Go toolchain commands. The Go entries are the most acute violation — they are language-specific commands inside the binary. The generic entries (semgrep, ast-grep, sandbox, config-file) are a lesser but still real violation: they make the binary the authority on what engines exist and what their defaults are, rather than pack data.

### Entanglement with BUNDLE-011

The `go-toolchain` pack that should receive `go-build`, `go-test`, and `golangci` is the SAME pack that BUNDLE-011 is building to retire `pkg/check` as a second enforcement engine. This issue cannot be planned or implemented independently of BUNDLE-011's scope decisions — specifically, which artifact (BUNDLE-011 spec or this issue's plan) owns authoring the `go-toolchain` pack that holds the native toolchain bindings. The plan for this issue MUST be sequenced after BUNDLE-011 resolves that ownership question, or the two workstreams must be formally merged.

### Missing mechanism: embedded default/base pack

No "default pack always-loaded" mechanism exists today. The path to zero baked defaults requires:

1. A base/default ENGINE pack shipped with backstop (likely embedded in the binary or adjacent to schemas) that declares the 4 generic engines (semgrep, ast-grep, sandbox, config-file) with their `FieldContract`s inline on the binding.
2. The `go-toolchain` pack (owned with BUNDLE-011) that declares the 3 Go-specific engines with their commands + Convert scripts.
3. The pack loader loading the embedded base pack through the normal pack path — so `resolveEngineRegistry`, `validateEngine`, `contractForEngine`, and the gate's dispatch all resolve built-ins from pack data, not from `engine.DefaultRegistry()` / `engine.DefaultFieldContracts()`.

`cmd/backstop/embed.go` + the root-level `embed.go` (schema embedding via `//go:embed`) are prior art for the embedding mechanism. The same pattern applies to an embedded pack directory.

### End state

- `engine.DefaultRegistry()` deleted.
- `engine.DefaultFieldContracts()` deleted.
- `engineFieldClaim` map and `claimFor()` deleted (claim codes migrate into the base pack's binding `FieldContract` entries as documentation or a separate CLM mapping in pack YAML).
- All 4 call sites resolved against pack data through the normal registry path.
- `pkg/pack/engine/binding.go` and `pkg/pack/engine/fieldcontract.go` contain no runtime data — only types, `ParseInputMode`, and `Registry.Lookup`.
- Completing this issue (alongside BUNDLE-011's `go-toolchain` pack delivery) is the thin-executor program's milestone: zero baked engine or language knowledge in the binary.

## Solution

**Split the 7 baked bindings by nature at authoring time:**

- `semgrep`, `ast-grep`, `sandbox`, `config-file` → a `backstop/engines` base pack (or `default-engines`), embedded and always-loaded. Each binding carries its `FieldContract` inline (eliminating the `DefaultFieldContracts` need).
- `go-build`, `go-test`, `golangci` → the `go-toolchain` pack (BUNDLE-011's deliverable). This pack already houses the Convert scripts (`scripts/build-to-sarif.sh`, `scripts/test-to-sarif.sh`, `ast-grep/to-sarif.sh`) which belong alongside the binding declarations.

**Wire the embedded pack through the normal load path:**

Add an embedded pack dir (via `//go:embed`) that the pack loader discovers before user-declared packs. The gate's engine registry and the manifest validator's `resolveEngineRegistryForValidation` both resolve through the unified pack-sourced registry. No special "fallback to DefaultRegistry" branch remains.

**Migrate `engineFieldClaim` claim codes into pack data or eliminate the indirection:**

Once the base pack's bindings carry inline `FieldContract`s, `contractForEngine` no longer needs the `DefaultFieldContracts()` fallback. The `engineFieldClaim` map's CLM codes exist for claim traceability; those codes may migrate to comments on the `FieldContract` entries in the base pack YAML, or the `claimFor()` function can be deleted and its callers replaced with a generic claim code (`CLM-003-engine-fit`).

**Sequence:**

1. Resolve BUNDLE-011's ownership of the `go-toolchain` pack (before or concurrent with planning this issue).
2. Implement the embedded base pack + loader mechanism.
3. Remove `DefaultRegistry()`, `DefaultFieldContracts()`, `engineFieldClaim`, and `claimFor()` from the binary.
4. Delete tests that assert on the baked defaults (they become pack-level tests).

## References

- `pkg/pack/engine/binding.go` — `DefaultRegistry()`, 7 baked `EngineBinding` literals
- `pkg/pack/engine/fieldcontract.go` — `DefaultFieldContracts()`, 4 baked `FieldContract` literals
- `pkg/pack/validate_manifest.go:130–158` — `engineFieldClaim` map and `claimFor()` (baked CLM codes)
- `pkg/pack/validate_manifest.go:235–254` — `resolveEngineRegistryForValidation` (call site 2) and `contractForEngine` (call site 4)
- `pkg/pack/manifest.go:419` — `validateEngine` fallback (call site 1)
- `cmd/backstop/embed.go` + root `embed.go` — prior art for `//go:embed` mechanism
- SPEC-035 OQ-1 — option (i) decision that deferred this work to Phase 6
- BUNDLE-011 — owns `go-toolchain` pack delivery; this issue is entangled with its scope
- CLAUDE.md — zero-baked-checks first invariant; eradication backlog (items C/D)
