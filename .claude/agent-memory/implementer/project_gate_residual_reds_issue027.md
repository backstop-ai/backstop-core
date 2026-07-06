---
name: gate-residual-reds-issue027
description: After ISSUE-027, gate has pre-existing/structural reds surfaced by editing files (contract file-scope quirks, types-only + module-root coverage) — NOT eradication failures
metadata:
  type: project
---

After ISSUE-027 the gate's `pack_engines` (self-pack no-baked) dimension is GREEN (binding.go 6→0). Two OTHER dimensions red on PRE-EXISTING/STRUCTURAL issues surfaced only because editing a file pulls it into diff scope (see [[project_editing_file_pulls_it_into_gate_scope]]). None is the eradication logic failing.

**contract_signature (10):** contracts come from SPEC/PLAN `contracts:` blocks, checked file-scoped.
- `claimFor` (validate_manifest.go): GENUINELY STALE — SPEC-035 (`specs/SPEC-035-...spec.md:604`) contracts on `func claimFor(...)` which ISSUE-027 deleted. Needs SPEC-035 contract retirement via spec-author (align-predating-artifacts), NOT hand-edit.
- `ParseCoordinate`/`NamespacedRuleID`: contract `file` says manifest.go but symbols live in `coordinate.go` → func-contract file mislocation, pre-existing.
- `ToolRequirement`/`DependencyRequirement` (struct), `Manifest.Classification`/`TestNamePatterns` (field), `InputModePatternArg` (const), `ExemptFromScopeFilter` (field), `go-coverage-rule` (pack rule): the struct/const/field/pack-rule contract-compiler gap ([[project_struct_contract_compiler_gap]]) — compile-signature.sh matches only func sigs, so all non-func contracts red under --file scope. Codebase-wide, pre-existing.

**coverage_threshold (2, structural):**
- `pkg/pack/engine/fieldcontract.go`: now TYPES + CONSTS only (0 funcs) after DefaultFieldContracts deleted → no executable statements → "no coverage measurement". A types-only .go is structurally unmeasurable; the coverage dim flags any changed *.go with no record.
- `embed.go` (module root): `ListSchemas` IS tested (90% via `go test .`), but the go-coverage producer's per-file record for the MODULE-ROOT package path doesn't map to repo-relative `embed.go`, so the changed-file matcher sees it unmeasured. Narrow path-normalization gap for root-level files (subpackage `pkg/...` files match fine). Not fixable by adding tests.
