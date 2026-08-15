---
name: feedback-flip-injects-notarget
description: Before calling a spec promotable, simulate the status flip — mandated tests enter the noTarget join only at terminal status and can inject new blocking violations
metadata:
  type: feedback
---

A spec whose mandated tests all exist AND pass is still not promotable until you check
what the flip to `implemented` TURNS ON. Two gate dimensions are status-gated and inert
while a spec is `draft`:

1. **contract_signature** — `provides` entries are extracted only at terminal status
   (`ExtractContractEntries` → `contractsAreDue`). This is what blocked SPEC-036.
2. **test_substantiveness Q2 noTarget** — `buildTestSubstantivenessStep` filters mandated
   tests through `ContractsAreDue(mt.Status)` (cmd/backstop/gate.go:1177) BEFORE the join.
   A draft spec's tests are invisible to it; flipping makes every one of them join against
   `TargetPackageName(implementation.subject)`.

**Why:** the gate's own `artifact_status_drift_advisory` only checks that mandated test
NAMES exist. It has no idea whether promoting would go red — SPEC-037 (2026-08-15) had all
37 tests existing and passing, and would still have injected 10 new blocking noTarget
violations, because tests living outside the subject package must reference the subject
token in the PACK-EXTRACTED symbol set (call/selector position only — a type-position
`*gate.GateScope` does not count) and are otherwise flagged.

**How to apply:** when asked "is this spec promotable", verify in this order —
(a) mandated tests exist AND pass for real; (b) each `provides` signature compiles through
the contracts pack's `compile-signature.sh` and MATCHES live code (run ast-grep with the
compiled pattern; `absent:` entries probe by grep on the NAME and ignore Signature);
(c) for every mandated test NOT colocated with the subject package, confirm the subject
token is in the extraction set, and confirm any predicted violation's absence from
`.backstop/baseline.json` — `applies-to: new-code` grandfathers against the BASELINE
(pkg/gate/policy.go:219), not against changed files, so a violation missing from the
baseline blocks. Per-claim `subject:` and `kind: absence` are the legitimate escape hatches.

Editing the spec to simulate the flip is blocked by the agent-guard (artifact writes route
through the authoring agents), so simulate dimension-by-dimension instead of end-to-end.
See [[spec037-review]].
