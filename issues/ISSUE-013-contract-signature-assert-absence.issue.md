---
title: "contract_signature gate step can only verify a symbol EXISTS, not that it is ABSENT — deletions can't be enforced via contracts"
schema_version: issue/v1

issue:
  id: ISSUE-013
  title: "contract_signature gate step can only verify a symbol EXISTS, not that it is ABSENT — deletions can't be enforced via contracts"
  type: enhancement
  status: open
  created: "2026-06-20"

complexity:
  scope: contained
  uncertainty: known
  risk: moderate
---

# contract_signature gate step can only verify a symbol EXISTS, not that it is ABSENT — deletions can't be enforced via contracts

## Problem

The gate's `contract_signature` step (`pkg/gate/step_contract.go`, `StepContractSignatureScopedFunc`) verifies that every declared contract symbol exists in the named file with the declared signature. It has no "must be absent" semantic.

**The gap surfaced concretely during the SPEC-034 native-toolchain cutover.** The cutover's strangler pattern required declaring that specific bespoke symbols — `lintExecutor`, `buildExecutor`, `testExecutor`, `goBuiltinExecutors`, and others — must be deleted from `pkg/check` non-test source after phase 3. The natural expression of that intent in contract form would be a contract entry with a sentinel signature such as `REMOVED` (or an explicit `absent: true` flag). But the verifier's logic at `step_contract.go` lines 96–104 makes absence unconditionally a violation:

```go
if !found {
    violations = append(violations, Violation{
        Rule:     "contract_signature",
        File:     file,
        Message:  fmt.Sprintf("symbol %s not found in %s", entry.Name, file),
        Severity: "error",
    })
    continue
}
```

A correctly-deleted symbol is reported as "not found" → the step fails even though the deletion is exactly what was intended. There is no path through the verifier that treats absence as a passing state.

**Workaround used in SPEC-034.** The sentinel contracts (`REMOVED`, `MIGRATE-OR-DELETE`) were removed from the spec entirely. Absence enforcement was moved into hand-written deletion-assertion tests:

- `cmd/backstop/cutover_deletion_test.go` — `TestCutover_GoShortCircuitRemoved`, `TestCutover_BespokeExecutorTypesDeleted`, `TestCutover_GoBuildTestFormatsRemoved`, `TestCutover_BespokeLintPathRemoved`
- `cmd/backstop/strangler_guard_test.go` — `TestStrangler_DeletionGatedOnProvenEquivalence`, `TestEndState_NoBakedGoToolchainKnowledge`, `TestEndState_NoTestReferencesDeletedBespokeSymbol`

These tests use `containsIdent` AST/ident scans that correctly prove the symbols are gone. They work, and the dual-assertion pattern (bespoke absent AND engine path present) in `TestStrangler_DeletionGatedOnProvenEquivalence` is actually the right shape for a strangler guard. But they are ad hoc per deletion: each new strangler cutover will need its own bespoke test file instead of a first-class contract declaration.

**A secondary smell also surfaced.** In the original sentinel contracts, the "symbol name" for test-file absence assertions was `bespoke-toolchain-tests` — a placeholder slug, not a real Go identifier. Absence assertions probably should key on a real identifier (or an explicit file-level assertion keyed on a file path and pattern), so the gate can actually probe the AST for the right thing. The symbol-name-must-be-a-valid-identifier question is separable from the existence/absence semantic but is worth addressing together.

## Impact

Backstop's strangler-pattern story is "replace-then-delete with enforcement that never lapses." The framework's own SPEC-034 cutover demonstrated that deletions are a first-class part of that story, and that the enforcement layer has no native way to express them.

Without a first-class absence assertion:

- Every strangler cutover must ship its own ad hoc deletion-assertion tests. There is no standard contract form, no schema-enforced traceability between a spec's deletion intent and the test that verifies it, and no protection against accidentally omitting the assertion entirely.
- A symbol that re-appears in a later commit (regression) is only caught if someone happens to run the bespoke test file. A gate-level contract would catch it on every diff that touches the relevant file.
- The `contract_signature` step reports "symbol not found" as an error, which means a spec author who correctly expresses deletion intent via a sentinel signature will be punished with a gate failure, not a pass. The system actively discourages the correct behavior.

This is a dogfood gap the framework's own cutover exposed in the session that authored SPEC-034.

## Solution

Teach `contract_signature` to interpret an absence assertion. Two plausible spellings:

1. **Sentinel signature** — a contract entry with `signature: REMOVED` (or a defined constant such as `ABSENT`) passes iff the named symbol is genuinely not present in the file; fails if the symbol appears (regression guard). No schema change required beyond documenting the sentinel value.

2. **Explicit flag** — add an optional `absent: true` field to the contract entry YAML. The verifier checks this flag before the AST lookup; if set, it inverts the pass/fail logic. Requires a schema update but makes intent unambiguous in the contract declaration.

Either spelling requires the verifier loop at `step_contract.go` lines 96–104 to branch on the absence intent rather than treating "not found" as unconditionally a violation.

**Symbol-name constraint.** An absence assertion keyed on a real Go identifier (e.g. `lintExecutor`) allows the AST scanner to verify absence precisely. A placeholder slug (e.g. `bespoke-toolchain-tests`) is not scannable and should be rejected. The implementation should validate that absence-flagged contracts name a real identifier, not a descriptive slug.

**Strangler guard shape.** The `TestStrangler_DeletionGatedOnProvenEquivalence` pattern — bespoke absent AND replacement present — is the correct dual assertion. A future design might express both halves as a contract pair: one absence contract for the deleted symbol and one presence contract for the replacement. Whether contracts should support that pairing semantically is a design question for the plan.

The relevant logic to modify is contained in `pkg/gate/step_contract.go` (`StepContractSignatureScopedFunc`, lines 71–114) and `ContractEntry` (lines 16–21 — may need a new field). The `ExtractContractEntries` function in the artifact reader (wherever it lives) would also need to surface the new field.

## References

- `pkg/gate/step_contract.go` lines 16–21 (`ContractEntry` struct), lines 71–114 (verifier loop, the `!found` branch at 96–104 is the specific gap)
- `cmd/backstop/cutover_deletion_test.go` — the ad hoc deletion assertions that filled the contract gap for SPEC-034 (CLM-004, CLM-006, CLM-007, CLM-018, CLM-029)
- `cmd/backstop/strangler_guard_test.go` — the strangler guard and end-state scans (CLM-033, CLM-036, CLM-037, CLM-038); the dual-assertion shape in `TestStrangler_DeletionGatedOnProvenEquivalence` is the model for what a first-class contract pair would express
- SPEC-034 (`specs/SPEC-034-native-toolchain-cutover.spec.md`) — the spec whose cutover exposed this gap; the original sentinel contracts (`REMOVED`, `MIGRATE-OR-DELETE`) were the failed attempt to use the contract layer for absence enforcement
