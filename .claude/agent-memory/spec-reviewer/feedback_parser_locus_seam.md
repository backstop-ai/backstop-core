---
name: parser-locus-seam
description: Pack engine dispatch and the native toolchain code-check registry are SEPARATE call paths; a claim that scopes work to one may be vacuous if the target lives in the other
metadata:
  type: feedback
---

backstop has two parser/executor loci that are easy to conflate: (1) the PACK engine
dispatch path (cmd/backstop pack_gate `mergePackRules` → semgrep, plus the new
`dispatchPackEngines`), and (2) the NATIVE toolchain code-check registry (pkg/check
`registry.go`, ISSUE-003, where `golangci-json`/`eslint-json`/`tsc` formats are wired
per-stack). They do not share an executor.

**Why:** SPEC-031 CLM-036 claimed `golangci-json`/`eslint-json` parsers are "removed from
the engine dispatch path." Those parsers were NEVER in the pack dispatch path — they live
only in registry.go:70 (native TS toolchain). So the claim is vacuous/untestable as scoped,
and the bundle's real parser-retirement intent (REQ-014) risks falling between SPEC-030
(packs-only/native removal) and SPEC-031.

**How to apply:** When a spec asserts removal/retirement of a parser or executor, grep for
the symbol's actual call sites and confirm the named locus matches. If the symbol lives in
the native registry but the spec scopes the change to pack dispatch (or vice versa), flag a
mislocated/vacuous claim and check whether a sibling spec actually owns it.

**Stronger corollary (SPEC-034, 2026-06-18):** the seam is not just per-parser — the
whole ENGINE MODEL (`EngineBinding`/`Registry`/`Provision`/`config-file`/`sandbox` shapes/
the sandboxed convert pipe) lives in `pkg/pack/engine` + `cmd/backstop/pack_gate.go` and has
NO import edge into `pkg/check` (`grep "pack/engine" pkg/check/` is empty). The native
`commandExecutor` is command + named in-binary parser only; it cannot run a pack-relative
`Convert` script or use `RunStdout`. The `config-file`/`sandbox` bindings in `DefaultRegistry()`
carry `Command:""` — they are pack-dispatch descriptors, not native executors. So a spec that
says "declare the native build/test pass as a `sandbox` engine entry + pack converter" or
"lint as a `config-file` engine" is asserting a bridge that does not exist; the bridge IS the
work and must be named (new type/path, or re-route native passes through dispatch). SPEC-031
REQ-013 explicitly disclaimed this native-ToolchainEntry→engine conversion as "owned
elsewhere" — confirm "elsewhere" actually builds it, don't assume the substrate already reaches
pkg/check.
