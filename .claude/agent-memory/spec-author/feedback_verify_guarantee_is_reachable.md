---
name: verify-guarantee-is-reachable
description: Before writing a claim asserting a security/trust guarantee, verify the check is actually REACHABLE for the subject — guard conditions make claims vacuous from inception, not by drift
metadata:
  type: feedback
---

Before writing a claim that asserts a guarantee ("X is allowlisted", "Y clears the trust
floor", "Z is validated"), read every call site of the enforcing function and confirm no
guard condition short-circuits it for THIS claim's subject. A claim whose check is
unreachable is vacuous **from inception** — it never protected anything, and it is worse
than no claim because it reads as coverage.

**Why:** SPEC-047 CLM-007 asserted `oxlint`/`bun`/`tsc`/`prettier` "are allowlisted."
`engine.CheckToolAllowed` has exactly three call sites (`checkEngineToolAllowed` in
`cmd/backstop/pack_gate.go`, `validateEngine` in `pkg/pack/manifest.go`,
`DefaultExecutor.RunEngine` in `pkg/packval/executor.go`) and every one is guarded by
`binding.Provision != nil`. The bun pack declares no `provision:`, so the gate was never
reached. The claim's test passed for two spec versions while proving nothing, and it
blocked a legitimate dead-code cleanup (ISSUE-082) that wanted to delete the unreachable
allowlist entries. Corrected 2026-08-15 at spec_version 1.2.1 under founder-approved scope.

**How to apply:** when a claim's subject is a trust/validation/authorization guarantee,
grep the enforcing symbol for ALL call sites and read each guard before writing the claim.
If a guard excludes the subject, either drop that half of the claim or write the claim
against what is genuinely enforced. Split a two-part claim ("A is allowlisted AND B is
pack-declared data") — one half being real does not rescue the other, and the vacuous half
will eventually block someone's cleanup. When correcting such a claim, also check the
mandated test bodies of NEIGHBORING claims: a stray assertion often leaks into a sibling
test whose claim text never mentioned it (SPEC-047 CLM-003 carried the same allowlist
assertion despite being a `gate_type` claim).

Related: [[feedback_kind_function_contracts_existence_only]] — another case where the
gate checks less than the spec text implies.
