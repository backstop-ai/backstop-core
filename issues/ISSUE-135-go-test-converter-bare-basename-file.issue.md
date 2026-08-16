---
title: "go-test converter emits a bare basename for nested-package test failures, not a repo-relative path"
schema_version: issue/v1

issue:
  id: ISSUE-135
  title: "go-test converter emits a bare basename for nested-package test failures, not a repo-relative path"
  type: bug
  status: open
  created: "2026-08-15"

complexity:
  scope: contained
  uncertainty: known
  risk: moderate
---

# go-test converter emits a bare basename for nested-package test failures, not a repo-relative path

## Problem

The `backstop-ai/go-toolchain` pack's `go-test` findings engine converts `go test`'s raw stdout to
SARIF via `scripts/test-to-sarif.sh` (bound at `pack.yml:82`, `convert: scripts/test-to-sarif.sh`).
For a failing test in a package below the repo root (e.g. `sub/pkg`), the SARIF result's
`locations[].physicalLocation.artifactLocation.uri` — which becomes the violation's `File` field
once bridged into backstop's internal model — is a bare basename (`"a_test.go"`), not a
repo-relative path (`"sub/pkg/a_test.go"`).

Reproduced directly against the real converter, run from the repo root exactly as
`cmd/backstop/pack_gate.go` invokes it (no `Dir` override — the command runs in the process cwd,
i.e. the project root, with `project_target: "./..."` appended):

```
$ go test ./...
--- FAIL: TestFails (0.00s)
    a_test.go:6: boom
FAIL    gotest-repro/sub/pkg   0.472s

$ go test ./... 2>&1 | sh scripts/test-to-sarif.sh
{"version":"2.1.0","runs":[{"results":[{"ruleId":"go-test","level":"error",
"message":{"text":"TestFails: boom"},
"locations":[{"physicalLocation":{"artifactLocation":{"uri":"a_test.go"},
"region":{"startLine":6}}}]}]}]}
```

`uri` is `a_test.go`, not `sub/pkg/a_test.go`, even though the failing package is two levels
below the repo root.

## Root cause

The converter (`scripts/test-to-sarif.sh`) is faithful to its input: it scans each `--- FAIL:`
block for the first `<file>.go:<line>: <detail>` line and copies that token verbatim into `uri`
(see the `awk` script's `file = substr(line, 1, idx+2)` capture). The bug is upstream of the
converter, in what `go test` itself prints. Go's `testing` package locates a failure via
`runtime.Caller` and formats it through an internal `decorate()` step that takes
`filepath.Base()` of the caller file — this is Go's own behavior, not something backstop's
tooling adds or could suppress by invoking `go test` differently. So `go test`'s raw per-failure
line is ALWAYS a bare basename regardless of how deep the package is nested; only the per-package
`FAIL\t<import-path>\t<time>` summary line (also present in the raw output, but currently ignored
by the converter) carries the package's location, as a Go import path rather than a filesystem
path.

The converter has everything it needs to fix this: each `--- FAIL:` block belongs to exactly one
package (delimited by that package's own `FAIL\t<import-path>\t<time>` summary line, or a `PASS`
transition to the next package's output), and the import path is derivable to a repo-relative
directory (either by stripping the module path prefix declared in `go.mod`, or via `go list`).
The converter currently discards this information — it never looks past the `--- FAIL:` block's
own `.go:NN:` line — so the fix is a converter change, not a `go test` invocation change.

## Impact

`pkg/gate/scope.go`'s `filterViolations` does exact repo-relative-path membership matching against
the diff's file set (`scope.Contains(violation.File)`). A bare-basename `File` can never match a
real diff-scope entry, so on its face this looks like it would make a nested-package go-test
violation silently unattributable to the diff even when the failing file genuinely IS in scope —
worse than being merely out-of-scope, because there is no repo-relative path at all to check.

In the CURRENT tree this is not the operative failure mode: ISSUE-129 established that `go-test`
violations carry `ProjectWide: true` (the `go-test` binding declares `exempt_from_scope_filter:
true`), and `filterViolations` keeps a violation when `ProjectWide` is true OR the file-membership
check passes (`pkg/gate/scope.go`) — the OR means every go-test violation already survives
scope-filtering regardless of `File`, so this defect is not currently the reason any go-test
finding is dropped from a diff-scoped gate result.

The wrong `File` value is still live and reachable through other consumers of `violation.File`
that are not scope-filtering:

- Human-readable gate output names the wrong (or an ambiguous/incomplete) location for a failing
  test — `a_test.go:6` instead of `sub/pkg/a_test.go:6`, which is actively misleading whenever a
  repo has more than one file with the same basename across packages (common for `a_test.go`-style
  or generically-named test files).
- Baseline comparison (`pkg/gate/testdata` baseline machinery, `backstop baseline`) is keyed on
  finding identity, which includes `File`; a collision between same-named test files in different
  packages could conflate or fail to distinguish genuinely different violations.
- Any future feature that reads `violation.File` for a go-test finding inherits the same
  ambiguity — this is a correctness gap in the finding's own data, not a gap specific to how
  scope-filtering happens to read it today.

## Solution

Not prescribed here — the converter needs the failing test's PACKAGE, not just its bare filename,
to build a repo-relative path. Direction only: correlate each `--- FAIL:` block with the
per-package `FAIL\t<import-path>\t<time>` summary line that closes its package's output block (or
with `PASS\t<import-path>` boundaries between packages), then resolve that import path to a
repo-relative directory — via the module path declared in `go.mod`, or `go list -f
'{{.Dir}}' <import-path>` relative to the repo root — and join it with the bare basename the
`--- FAIL:` block already captures. An alternative worth considering when this is actually spec'd:
running `go test -json` instead of parsing human-readable stdout gives a `Package` field on every
test event natively, removing the need for this correlation entirely.

## References

- Lives in the `backstop-ai/go-toolchain` pack
  (`/Users/bmanson/src/projects/backstop-go-toolchain-pack`), not backstop-core — any fix is a
  pack change (version bump + relock), same as ISSUE-129's fix.
- Found during ISSUE-129 investigation (scope-filtering blind spot on go-test violations) and
  ISSUE-118 (gate blind spot on test-only diffs). Homed in DIR-024 (corrected 2026-08-16 — the
  scope-suppression risk this issue originally cited was neutralized by ISSUE-129's
  exempt_from_scope_filter fix; what remains is a finding-data precision defect, DIR-024's
  charter, not a wrong-verdict defect under DIR-032). A gate signal (`violation.File` for a
  go-test finding) that reads as an authoritative repo-relative location and is not, for any
  package below the repo root.
- Sibling but distinct from ISSUE-067 (go-test failures surfacing as an opaque crash instead of
  parseable findings — a findings-existence defect) and ISSUE-129 (go-test findings dropped by
  diff-scope filtering — a filtering-logic defect keyed on `ProjectWide`, not on `File`). This
  issue is about the correctness of the `File` value itself once a finding is already produced.
