---
name: spec038-dispatch-bypass
description: SPEC-038 Seed 4 contract-path dispatch bypass — found in first review, FIXED + verified in re-review; both guard tests mutation-pass
metadata:
  type: project
---

SPEC-038 (BUNDLE-009 Seed 4, contracts pack) — RESOLVED.

**First review (FAIL):** the contract gate path ran ast-grep/grep/convert via RAW `exec.Command` (`gate.PackContractResult` in pkg/gate/contract_equivalence.go), bypassing `dispatchPackEngines` + the macOS sandbox + `CheckToolAllowed`. CLM-049 (sandboxed convert) + REQ-005 (allowlist trust-floor) unmet; pack engine bindings inert.

**Fix (re-review PASS):** `produceContractEngineResults` (cmd/backstop/gate.go) now → `dispatchContractEntry` → `resolveDispatchPackEngines()` (the SAME dispatch seam declared packs use), so convert runs under `packval.SandboxedRunStdout` and grep clears `CheckToolAllowed`. The only remaining shell-out is `compileContractSignature` (the legitimate pack-side COMPILER script — binary never renders a pattern). `gate.PackContractResult`/`PackVerdict` (raw exec) survive ONLY as the strangler TEST equivalence oracle (callers all `_test.go`), NOT production.

**Two new guard tests, BOTH mutation-verified load-bearing:**
- `TestE2E_ContractsRealAstGrepAndGrep_AndSandboxedConvert` — spies the real `sandboxedRunStdout` seam, asserts BOTH ast-grep/to-sarif.sh AND grep/to-sarif.sh ran under it. Mutation (revert dispatchContractEntry to raw exec) → FAILS "zero sandboxed converts".
- `TestE2E_ContractsGrepGatedByAllowlist` — removes grep/rg via the `trustedToolAllowlist` seam, asserts loud ConfigErr. Mutation (raw exec) → FAILS (raw exec ignores allowlist).

Gate: 6 passed / 0 failed / 2 warned (contract_signature ⚠ + test_substantiveness ⚠, both correct class-2 capability-absent — packs not installed into backstop-core itself) / 3 skipped (pre-existing infra). pack_engines pass, no new dogfood violations. Coverage 90.3%.

**Review trick that worked (reuse this):** to verify a "runs under sandbox / allowlist" claim, MUTATE the production dispatch back to raw exec and confirm the spy test FAILS — a spy that wraps the real seam and counts invocations is load-bearing; one that only asserts a violation surfaced is not. See [[project_pack_provisioning_integration_gap]] — this is the gap finally closed at the dispatch (not just install) layer.
