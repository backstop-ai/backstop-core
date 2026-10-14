---
title: "Global State Rule False Positives And SPEC-019 Test Subject"
schema_version: issue/v1

issue:
  id: ISSUE-227
  title: "Global State Rule False Positives And SPEC-019 Test Subject"
  type: bug
  status: open
  created: "2026-10-14"

complexity:
  scope: contained
  uncertainty: known
  risk: moderate
---

# Global State Rule False Positives And SPEC-019 Test Subject

## Problem

ISSUE-226 cleared the go-toolchain 1.10 lint debt. Afterwards, `backstop gate --file` over the files
it changed still reports 5 blocking findings, all there before that change. They come from two
separate causes. Neither has a mechanical fix in this repository.

### 1. go-standards `no-global-mutable-state` flags declarations that are not mutable globals

`backstop-ai/go-standards` rule `go.core.no-global-mutable-state` (GO-003) blocks on
`cmd/backstop/embed.go`:

```go
var SchemaFS embed.FS = backstopcore.SchemaFS
```

SPEC-005 declares this exact signature as a contract (`var SchemaFS embed.FS`). It is the CLI
package's re-export of the module's embedded schema filesystem. Nothing assigns to it after
initialization, and `embed.FS` is read-only. The module-root `var SchemaFS embed.FS` with a
`//go:embed` directive is not flagged, only because it has no initializer.

The rule's only positive pattern is `var $X = ...`, with no constraint to package scope. It
therefore also matches `var` declarations inside function bodies. In ISSUE-226 it flagged a
function-local `var _ fs.FS = ...` type assertion in `embed_test.go`, which was rewritten to get
past it. Local variables are not package-level state.

### 2. SPEC-019's declared subject does not match where four of its tests live

SPEC-019 declares `implementation.package: cmd/backstop`. CLM-001 and CLM-002 override this with
`subject: pkg/gate`. CLM-003, CLM-005 and CLM-008 have no override, so they inherit
`cmd/backstop`. Their mandated tests are in `pkg/gate`:

- CLM-003: `TestGate_BaselineComparison_WiredAfterAccumulatedChecks` (`pkg/gate/gate_test.go`)
- CLM-005: `TestGate_BaselineComparison_MissingBaselineSkipsWithReason` (`pkg/gate/gate_test.go`)
- CLM-008: `TestGateOutput_BaselineDifferential_HumanMessaging` and
  `TestGateOutput_BaselineDifferential_JSONDiagnostics` (`pkg/gate/output_test.go`)

`test_substantiveness` reports each one as "does not call package backstop". The tests exercise
`pkg/gate` code, so the finding comes from the declared subject, not from a hollow test.

## Impact

- `backstop gate` cannot pass for a change that touches `cmd/backstop/embed.go`, `embed_test.go`,
  `pkg/gate/gate_test.go` or `pkg/gate/output_test.go`. The diff-scoped and `--file` modes check
  each touched file whole.
- A contracted, read-only re-export and local type assertions are reported as mutable global state.

## Direction

(Direction, not a plan.)

1. In the `backstop-ai/go-standards` pack, limit GO-003 to package-scope declarations, so `var`
   inside a function body no longer matches. Decide whether a read-only `embed.FS` re-export
   counts as mutable state. One option is to exclude `embed.FS`-typed declarations, as
   `regexp.MustCompile` already is. Add fixtures for both cases. Then update the pack version in
   this repository.
2. Correct SPEC-019 so these claims name the unit they test. Either add `subject: pkg/gate` to
   CLM-003, CLM-005 and CLM-008, or change the spec-level subject if `pkg/gate` is the right
   default. Check the remaining claims while doing this. SPEC-019 is `implemented`, so the change
   should follow the repository's process for amending an implemented spec.

## Notes
