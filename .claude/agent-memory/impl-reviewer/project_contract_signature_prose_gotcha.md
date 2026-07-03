---
name: contract-signature-prose-gotcha
description: contract gate false-fails when a spec's signature carries inline // prose (checker only collapses whitespace, never strips comments); deletion contracts must use absent:true not a prose "DELETED—" signature
metadata:
  type: project
---

The contract_signature gate step (`signaturesMatch` at pkg/gate/step_contract.go) only
collapses whitespace via `normalizeSignature` — it does NOT strip trailing `// prose`
comments. Any spec `contracts[].provides[].signature:` that embeds inline `// explanation`
prose will FALSE-FAIL even when the Go source signature is correct, because the checker
compares the full prose-laden string against the bare source signature.

Two recurring spec-authoring defects this surfaces (seen in SPEC-037 v1.2.1 review):
1. **Prose in signature:** strip `// ...` from the signature field; put rationale in
   surrounding prose instead. Also write result groups Go-canonically:
   `(hollow, extraction []Violation)` NOT `(hollow []Violation, extraction []Violation)`.
2. **Deletion contracts:** a contract asserting a symbol is GONE must use
   `{name: X, kind: function, absent: true}` — NOT `kind: function` + a prose
   `"DELETED — ..."` signature. The schema DOES support `absent` (ContractEntry.Absent,
   read from `provides[].absent` in ExtractMandatedTests/ExtractContractEntries). The
   absent path is a real deletion-regression guard ("symbol expected absent but present →
   fail"); a prose signature instead makes the checker report "symbol not found" — a
   false-fail that also fails to guard the deletion.

**Why:** these read as gate failures but are SPEC-AUTHOR fixes (route to spec author),
not implementation defects — distinguish them from genuine signature mismatches (a
function whose ACTUAL source signature diverges from the contract).
**How to apply:** when a contract_signature violation's expected vs got differ ONLY by a
trailing `// ...` or by `(a T, b T)` vs `(a, b T)`, it's the checker limitation, not a real
mismatch. For "symbol not found" on a symbol the spec INTENDS to delete, the fix is
absent:true. Confirm baseline suppression status: with no origin remote, baseline_comparison
SKIPS, so normally-suppressed pre-existing artifacts surface raw — don't accept "pre-existing"
without checking which spec OWNS the contract (untouched older spec = genuinely pre-existing;
the spec under review = its own authoring defect). See [[pack_provisioning_integration_gap]].
