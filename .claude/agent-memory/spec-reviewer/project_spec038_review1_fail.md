---
name: spec038-ready-to-plan
description: SPEC-038 (BUNDLE-009 Seed 4, contracts pack) PASSED review; v1.2.0 provisioning/E2E/capability-rekey update verified vs live code — ready to plan
metadata:
  type: project
---

SPEC-038 (`specs/SPEC-038-traceability-contracts-pack.spec.md`, v1.2.0, BUNDLE-009 Seed 4)
PASSED spec-review on 2026-06-23. Validator PASS (15 REQ / 52 CLM, no orphan claims, no dup
mandated test names, every claim binds one test). Ready to plan.

**v1.2.0 = TARGETED update (v1.1.0→1.2.0)** adapting SPEC-037 v1.2.2 / Seed 3's
provisioning + E2E + capability-rekey pattern to CONTRACTS. The eradication CORE (grep
pack-declared REQ-005, pack-side contract→ast-grep compiler REQ-002, absence-via-grep REQ-003,
strangler REQ-008, wiring spy REQ-006, TS rules REQ-007, delete-or-migrate REQ-010/011,
dogfood REQ-009) was UNCHANGED and was NOT re-litigated (already passed review #2 at v1.1.0,
see git history). The three NEW reqs all verified against live code:

1. **REQ-013 provisioning** — Go contracts pack as ordinary INSTALLED LOCAL pack (no //go:embed,
   no baked tier, no testdata-as-production). Verified: `add.go:102` sets sourceType="local",
   `add.go:235` writes lock entry, `verify.go:47-48` skips SourceType=="local". Pack-ownership
   precision correct: Go pack = NEW installable; TS contract rules CO-OWNED with SPEC-037's
   shared TS proof pack (rules ADDED, not a 2nd pack).
2. **REQ-014 real-over-installed-pack E2E** — genuinely unstubbable. CLM-046/047 both polarities
   through real pipeline (real `dispatchPackEngines` @ pack_gate.go:199 → real ast-grep + real
   grep → real sandboxed convert → SARIF → verdict); CLM-048 negative no-vacuous-green
   (uninstalled→no violation); CLM-049 asserts BOTH real ast-grep (ISSUE-028) AND real sandboxed
   convert (ISSUE-029). REQ text forbids stub / testdata-in-production / wiring-spy-alone.
3. **REQ-015 contracts-arm capability re-key** — the load-bearing existing-test coupling, the
   same class spec-review caught on SPEC-037. VERIFIED LIVE: `deriveCapabilityState`
   (cmd/backstop/gate.go:273) has a "HARD FENCE" comment (gate.go:280-281) — substantiveness
   already re-keyed (Seed 3), coverage+contracts still baked-Go. The shipped
   `TestCapabilityState_NonGoProject_DerivesAbsentClass2` (gate_capability_test.go:26-28) DOES
   iterate `{DimensionCoverage, DimensionContracts}` together in the baked-Go loop (lines 38-45
   assert Go→Present via baked analyzer) — so once contracts re-keys it goes RED. REQ-015/CLM-052
   correctly mandates splitting DimensionContracts OUT (leaving only DimensionCoverage), mirroring
   how Seed 3 split out DimensionSubstantiveness. 3-arm end state explicit (subst+contracts→pack,
   coverage→baked). `contractsPackInstalled`/`contractsPackName` do NOT yet exist (real additions
   mirroring live `substantivenessPackInstalled`@326/`substantivenessPackName`@574).

**#4 absent:true lesson applied cleanly:** NO `signature:` line carries a trailing `//` comment
(grep empty). Only ONE `absent: true` entry (line 649, StepContractSignatureFunc deletion guard);
the other deleted symbols (probeSymbol/signaturesMatch/formatFuncSignature/etc.) are prose-only in
REQ/CLM, NOT declared as provides[] — so the new pack-based contract check can't inherit a false-fail.

**#6 zero-baked intact:** grep ABSENT from DefaultRegistry (binding.go:179 = semgrep/ast-grep/
sandbox/config-file + native toolchain only) — REQ-005 stand-up is real work, not laundering a
baked entry. Compiler stays in pack (CLM-006, P0 Sharp Edge 1). No baked tool/lang knowledge added.

Deletion surface verified live in step_contract.go: probeSymbol:227, findFunction:274,
underlyingTypeString:388, formatFuncSignature:456, signaturesMatch:551, normalizeSignature:556,
go/ast+parser+printer imports:7-9, StepContractSignatureFunc:34 (thin wrapper → Scoped(…,nil)).

**How to apply:** SPEC-038 cleared to plan. Re-walk only if version > 1.2.0. Sibling of
[[spec037-review]] (shares the TS proof pack + the capability-rekey coupling pattern). The
capability re-key is the cross-spec keystone; aligns SPEC-036 via impl, not revision.
