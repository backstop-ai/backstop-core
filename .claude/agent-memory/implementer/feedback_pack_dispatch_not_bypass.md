---
name: pack-dispatch-not-bypass
description: Pack-consuming gate steps MUST route engine execution through the real dispatchPackEngines seam, never raw exec.Command — the recurring pack-provisioning integration gap
metadata:
  type: feedback
---

When a gate step consumes a pack's engine output (substantiveness, contracts, any
future pack-backed dimension), the engine + convert execution MUST go through the real
dispatch seam (`resolveDispatchPackEngines()` / `dispatchPackEnginesFn` ->
`runFindingsEngine`), NOT a parallel `exec.Command` + `/bin/sh <convert>` path.

**Why:** RECURRING defect — bit SPEC-035 P4, SPEC-037 Seed 3, and SPEC-038 Seed 4
(caught in impl-review). A raw-exec bypass makes the install real but the DISPATCH inert:
(1) the convert never runs under the real macOS sandbox (`packval.SandboxedRunStdout`),
(2) the tool never passes `CheckToolAllowed` (trusted-tool allowlist trust floor),
(3) the pack's declared `engines:` block + rule YAML are decorative. Tests pass green
while the production path is unproven.

**How to apply:**
- Build a per-invocation single-rule `pack.Manifest` carrying the pack's DECLARED
  `Engines`, set each rule's `Pattern` (pattern-arg) per work-item, scope to the target
  via `gate.ComputeGateScope(root, GateScopeModeFile, [target])`, and call
  `resolveDispatchPackEngines()([]*pack.Manifest{single}, root/.backstop/packs, ...)`.
  Dispatch resolves convert relative to `packDir/<normalizedName>`, so the pack must be
  INSTALLED under `.backstop/packs/backstop/<name>` (loadInstalledPacks resolves it).
  Shelling the COMPILER script (signature->pattern) is fine; shelling the ENGINE+convert
  is the bypass.
- Pattern-arg engines need a DECLARED engine in `engines:` (e.g. ast-grep-contracts with
  input_mode pattern-arg, input_flag --pattern). The built-in ast-grep is config-file
  mode and will NOT accept --pattern.
- The E2E "real engines + sandboxed convert" test MUST spy on the `sandboxedRunStdout`
  seam (wrap, don't replace) and assert >=1 convert per engine ran through it — so a
  raw-exec bypass records ZERO sandboxed converts and FAILS. Without this spy the test
  passes vacuously even on a bypass (how the defect shipped green).
- Engines dispatching over a file must scan a file UNDER projectRoot — an out-of-
  workspace absolute path returns no findings. Copy fixtures into the workspace.
- See [[feedback_agent_guard_testdata]] (use Bash for pack .yml/.sh + memory writes).
