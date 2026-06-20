---
title: "SPEC-007 contracts pin ResolveID/IDOptions to pkg/scaffold/scaffold.go, but they live in pkg/scaffold/idresolver.go — contract_signature fails whenever scaffold.go is in diff scope"
schema_version: issue/v1

issue:
  id: ISSUE-016
  title: "SPEC-007 contracts pin ResolveID/IDOptions to pkg/scaffold/scaffold.go, but they live in pkg/scaffold/idresolver.go — contract_signature fails whenever scaffold.go is in diff scope"
  type: bug
  status: open
  created: "2026-06-20"

complexity:
  scope: isolated
  uncertainty: known
  risk: safe
---

# SPEC-007 contracts pin ResolveID/IDOptions to pkg/scaffold/scaffold.go, but they live in pkg/scaffold/idresolver.go — contract_signature fails whenever scaffold.go is in diff scope

## Problem

SPEC-007 (`specs/SPEC-007-artifact-new.spec.md`) contains two contract entries filed under the wrong file block. The `## Contracts` section at line 761 opens a `file: pkg/scaffold/scaffold.go` block that, in addition to the symbols that genuinely belong there, includes:

```yaml
- name: ResolveID
  kind: function
  signature: "func ResolveID(artifactType string, opts IDOptions) (string, error)"

- name: IDOptions
  kind: type
  signature: "type IDOptions struct"
```

Both symbols are defined in `pkg/scaffold/idresolver.go`, not `scaffold.go`:

```
$ grep -n "func ResolveID\|type IDOptions" pkg/scaffold/scaffold.go pkg/scaffold/idresolver.go
pkg/scaffold/idresolver.go:53:  type IDOptions struct {
pkg/scaffold/idresolver.go:200: func ResolveID(artifactType string, opts IDOptions) (string, error) {
```

`scaffold.go` has zero matches. SPEC-007 even has a separate, correct `file: pkg/scaffold/idresolver.go` block (line 793) that correctly declares `GitTagResolver` and `LocalScanResolver` — the `ResolveID` and `IDOptions` entries were simply placed in the wrong block above it.

**How the gate fails.** The `contract_signature` step (`pkg/gate/step_contract.go`) verifies that each declared symbol exists in the NAMED file. When `scaffold.go` enters diff scope it runs the AST lookup against `scaffold.go` for all entries in its block. Because `ResolveID` and `IDOptions` are not in that file, the step produces:

```
symbol ResolveID not found in pkg/scaffold/scaffold.go   [error]
symbol IDOptions not found in pkg/scaffold/scaffold.go   [error]
```

This is a false negative — the symbols exist and are correctly implemented; only the spec's file attribution is wrong.

**Latency of the failure.** The gate's `contract_signature` step is diff-scoped: it only examines the contracts for files that appear in the current diff. `scaffold.go` is not routinely modified, so the mis-filed entries were harmless until the ISSUE-011 fix applied a one-line edit to `scaffold.go`. That brought `scaffold.go` into scope and triggered the spurious failures. The bug is pre-existing, confirmed via `git stash` — reverting the ISSUE-011 change restores gate green, proving the contract entries are the defect, not the ISSUE-011 fix.

**Caveat: editing SPEC-007 re-scopes it into `test_verification`.** Moving these two entries to the correct `idresolver.go` block requires a spec-author edit to SPEC-007. Any SPEC-007 edit will pull the spec file into diff scope during the gate run that validates the fix, which will trigger `test_verification` for SPEC-007's mandated tests. Whether those tests are complete or carry the same diff-scoped-test-debt described in ISSUE-012 should be checked before editing and addressed in the same fix window.

## References

- `specs/SPEC-007-artifact-new.spec.md` lines 761–774 — the `file: pkg/scaffold/scaffold.go` block where `ResolveID` (line 767–770) and `IDOptions` (line 771–774) are currently (incorrectly) declared
- `specs/SPEC-007-artifact-new.spec.md` lines 793–803 — the `file: pkg/scaffold/idresolver.go` block where both entries should be moved
- `pkg/scaffold/idresolver.go` line 53 (`type IDOptions struct`) and line 200 (`func ResolveID`) — the actual symbol locations that prove the mis-attribution
- `pkg/gate/step_contract.go` — `StepContractSignatureScopedFunc`, the gate step that performs file-scoped symbol lookup and produces the false-negative failure
- ISSUE-011 — the one-line `scaffold.go` edit whose diff scope activated the latent failure; the gate failure is pre-existing and unrelated to the ISSUE-011 fix itself
- ISSUE-012 — the diff-scoped `test_verification` debt pattern; relevant when editing SPEC-007 to fix this issue, as the edit will pull SPEC-007 into test_verification scope
