---
title: "bundle-010 substrate (SPEC-017/SPEC-031) mandates ~58 test functions that were never written — diff-scoped test_verification hid it"
schema_version: issue/v1

issue:
  id: ISSUE-012
  title: "bundle-010 substrate (SPEC-017/SPEC-031) mandates ~58 test functions that were never written — diff-scoped test_verification hid it"
  type: bug
  status: open
  created: "2026-06-20"

complexity:
  scope: cross-cutting
  uncertainty: known
  risk: critical
---

# bundle-010 substrate (SPEC-017/SPEC-031) mandates ~58 test functions that were never written — diff-scoped test_verification hid it

## Problem

Two independent but related findings surfaced during contract reconciliation on ISSUE-010. They are reported together because Finding 2 is the mechanism that concealed Finding 1.

### Finding 1 — ~58 mandated tests for SPEC-017 and SPEC-031 were never written

SPEC-017 (pack-gate-integration) and SPEC-031 (pluggable-engine-dispatch) are reviewed-and-approved substrate specs from BUNDLE-010. Their `claims` blocks declare mandated test functions. Those test functions do not exist in the codebase and, critically, never did.

Verification method: for each of the 58 function names the gate's `test_verification` step reports as "not found," the git command below returned no output:

```
git log -S"func <name>" --all
```

55 of 58 names have zero appearances in git history. The remaining 3 appeared briefly and were removed. None are renames — the mandated contracts were simply never implemented.

**Representative missing tests (spec / claim / function):**

| Spec | Claim | Missing test |
|---|---|---|
| SPEC-017 | CLM-010 | `TestGateIntegration_Layer3ValidatorExecuted` |
| SPEC-017 | CLM-011 | `TestGateIntegration_Layer3NamespacedIDs` |
| SPEC-017 | CLM-021 | `TestGateIntegration_CodeCheckLayer3FullProject` |
| SPEC-031 | CLM-022 | `TestGateDispatch_SarifNativeNoConvert` |
| SPEC-031 | CLM-023 | `TestGateDispatch_NonSarifWithoutConvertFails` |
| SPEC-031 | CLM-033 | `TestEngineBinding_NoImportCycle` |
| SPEC-031 | CLM-037 | `TestMigration_GoPackEngineSemgrep` |
| SPEC-031 | CLM-038 | `TestMigration_NoSilentGrandfather` |
| SPEC-031 | CLM-067 | `TestEngineDispatch_SandboxBranchReKeyedFromLayer3` |

The full 58-entry list is the exact output of the gate's `test_verification` step when SPEC-017 and SPEC-031 are in diff scope.

**Impact:** BUNDLE-010's engine-dispatch and pack-gate substrate shipped without the integration tests its own specs mandate. Every claim in SPEC-017 and SPEC-031 asserts behavior that has no verifying test. This is precisely the vacuous-green / integration-gap pattern the project guards against.

### Finding 2 — `test_verification` is diff-scoped, which concealed Finding 1

The gate's `test_verification` step skips mandated-test existence checks for any spec whose file is not in the current diff scope. The guard in `pkg/gate/step_testverify.go` (lines 158–164):

```go
for _, mt := range mandated {
    if scope != nil && scope.Mode != GateScopeModeAll {
        if mt.FilePath != "" && !scope.Contains(mt.FilePath) && !scope.Contains(mt.SpecFile) {
            continue   // skipped: spec not in diff
        }
        if mt.FilePath == "" && !scope.Contains(mt.SpecFile) {
            continue   // skipped: spec not in diff
        }
    }
    ...
}
```

A spec's mandated tests are only checked when `mt.SpecFile` (the spec's `.spec.md` path) appears in the diff. Because SPEC-017 and SPEC-031 were untouched after BUNDLE-010 merged, no subsequent gate run ever pulled them into scope — even though their mandated tests were absent the entire time.

The 58 missing tests became visible only when an unrelated contract reconciliation edited SPEC-017 and SPEC-031 directly, accidentally dragging their spec files into the diff and triggering the check.

**This is a vacuous-green hole in the gate itself.** A spec can be marked implemented and merged without its mandated tests existing; the gate will never flag it until some unrelated change happens to touch that spec file. Mandated-test existence is a repo-wide invariant, not a per-diff property. The pass/fail evaluation of those tests may reasonably stay diff-scoped, but existence-checking must be gate-wide (or at minimum triggered on all specs whose `status` is `implemented`).

This design question may warrant its own issue and spec; it is noted here so it is not lost.

## Impact

- SPEC-017 and SPEC-031 claims have no verifying tests. The behavioral guarantees they assert (SARIF conversion, engine dispatch, migration safety, pack-gate integration) are unverified.
- BUNDLE-010 passes its gate in all normal runs because the spec files are not in the diff — a structural false green, not a passing green.
- The same gap likely exists for other specs that landed in an earlier bundle and have since been untouched; the scope of exposure is unknown without a `--all` sweep.
- Finding 2 is a gate-design defect that will recur on future specs unless the existence check is decoupled from diff scope.

## References

- `pkg/gate/step_testverify.go` lines 131–187 — `StepTestVerificationScopedFunc`, the scoped gate step; lines 158–164 are the diff-scope guard
- SPEC-017 (`specs/SPEC-017-pack-gate-integration.spec.md`) — claims CLM-010, CLM-011, CLM-021 and others with no corresponding test functions
- SPEC-031 (`specs/SPEC-031-pluggable-engine-dispatch.spec.md`) — claims CLM-022, CLM-023, CLM-033, CLM-037, CLM-038, CLM-067 and ~44 others with no corresponding test functions
- ISSUE-010 — the contract reconciliation work that accidentally pulled SPEC-017/031 into diff scope and surfaced this debt; ISSUE-010 is a separate fix (pack_engines diff-scope) and is NOT blocked by this issue
