---
title: "gate --file Falsely REDs Non-Go Files Whose Directory Holds No Go Package"
schema_version: issue/v1

issue:
  id: ISSUE-093
  title: "gate --file Falsely REDs Non-Go Files Whose Directory Holds No Go Package"
  type: bug
  status: closed
  created: "2026-07-27"
  closed: "2026-08-17"

delivered_by: PLAN-ISSUE-093

complexity:
  scope: contained
  uncertainty: known
  risk: moderate
---

# gate --file Falsely REDs Non-Go Files Whose Directory Holds No Go Package

## Resolution

Fixed by PLAN-ISSUE-093 (completed 2026-08-17, commit 6c32a17 on `main`).

`pkg/gate/classification.go` gained two predicates on `SourceClassifier`: `ClaimsPath` (the
source-glob OR test-glob union — deliberately not `IsMeasurableSource`, so a `_test.go` file is
still claimed) and `DeclaresAnyGlobs` (distinguishes "pack declared nothing" from "declared,
no match"). `cmd/backstop/pack_gate_filemode.go`'s file-mode package-target derivation
(`fileModeTestTarget`, now `fileModeTestTargets`) went from a two-state, `scope.Files[0]`-only
derivation to a structural three-state `fileModeDecision`: not-applicable (diff/`--all`/nil scope
unchanged), targets-derived (one deduped package selector per claimed file, with a
capability-absent guard for packs that declare `package_scoped: true` but no classification
globs), and claims-nothing (the engine is not dispatched at all, and a non-blocking `warning`
advisory says so — never a silent skip, never a `./...` fallback). This is what stops a
`package_scoped` engine like go-toolchain's `go-test` from being dispatched against a directory
it has no interest in (e.g. `.github/workflows/ci.yml`), which was the root cause of the false RED.

Separately, `cmd/backstop/gate.go`'s `--file` flag is now genuinely repeatable
(`StringArrayVar`, accumulating with positional args) and refuses an empty value as a config
error, instead of silently keeping only the last occurrence of a repeated flag or falling
through to a diff-scoped sweep on an empty value.

One real deviation from the plan was found and fixed during implementation: `pflag`'s
`GetStringArray` silently drops a lone empty `--file` value via its CSV round-trip. The shipped
code reads the raw accumulated slice via `pflag.SliceValue.GetSlice()` instead.

The founder-level policy question this fix opens — may CI's blocking job now use `--file`, since
both defects are fixed? — is filed separately as ISSUE-156 and was deliberately not decided or
absorbed here; the existing ban on `--file`/`--all` in CI's blocking job stays in force.

## Problem

`backstop gate --file <path>` false-REDs a file with an opaque go-test engine crash whenever the
file's directory contains no Go package — regardless of the file's own language or whether it was
touched at all. Verified directly on an untracked-clean, unmodified file:

```
$ ./bin/backstop gate --file .github/workflows/ci.yml
  pack_engines    fail  (19304ms)  (1 violations)
    - [pack_engines] dispatching findings engine "go-test" for pack backstop/go-toolchain:
      pack backstop/go-toolchain engine "go test" crashed: non-zero exit with no parseable
      findings: exit status 1
FAIL
Error: gate: exit code 2
```

`.github/workflows/ci.yml` is long-tracked and untouched. The same command against `README.md`
passes, because the repo root happens to hold a Go package. So a per-file gate verdict on a
non-Go file is not a property of the file — it depends on whether the file's *directory*
incidentally contains Go source, which is an accident of repo layout and has nothing to do with
whether the file itself is correct. This directly poisons the "verify each file you touched"
workflow documented in implementer memory (`feedback_never_stash_shared_tree.md`, "Then make the
positive case too") — an agent scoping `--file` to a genuinely clean non-Go file gets a false RED
it cannot fix, because there is nothing wrong with the file.

## Root cause

`fileModeTestTarget` (`cmd/backstop/pack_gate_filemode.go:31-45`) fires whenever the gate is in
`GateScopeModeFile` and the dispatched binding declares `PackageScoped: true` — which the
`go-toolchain` pack's `go-test` engine does (`.backstop/packs/backstop/go-toolchain/pack.yml`,
`go-test:` block, `package_scoped: true`). It unconditionally calls `goTestPackageSelector` on
`scope.Files[0]` (`pack_gate_filemode.go:44`):

```go
func goTestPackageSelector(file string) string {
    dir := filepath.Dir(file)
    if dir == "" || dir == "." {
        return "."
    }
    ...
    return "./" + dir
}
```

This derives a `go test` package target from the file's directory with no check that the
directory contains any `.go` files at all. For `.github/workflows/ci.yml` the target becomes
`./.github/workflows`; `go test ./.github/workflows` finds no Go package there and exits non-zero
with no test output. The `go-test` engine binding declares `crash_guard: true`
(`pack.yml`, `go-test:` block), so the dispatch treats "non-zero exit, zero parseable findings" as
an engine crash rather than a clean/no-op result — the same opaque-crash surfacing ISSUE-067
documents for a different trigger (a real test failure the converter can't parse). Here the
trigger is different: there is no Go code in scope at all, so `go test` legitimately has nothing
to run, and that legitimate no-op is what gets reported as a crash violation.

The selector function has no guard for "does this directory contain a Go package" and no
distinction between "file I'm scoping to is itself Go" vs "file is some other language that
happens to share a directory tree with Go code" vs "file's directory has zero Go files anywhere
in the repo." All three collapse to the same blind `./` + directory target.

## Secondary defect — `--file` silently collapses repeated flags

Same command surface, independent bug. `--file` is declared as a plain string flag, not a
repeatable one (`cmd/backstop/gate.go:35,52`):

```go
var fileFlag string
...
cmd.Flags().StringVar(&fileFlag, "file", "", "scope gate to one or more explicit files")
```

despite the `Use:` string advertising `gate [--all | --file FILE [FILE...]]`
(`cmd/backstop/gate.go:37`). Passing `--file` twice does not error and does not accumulate — pflag's
`StringVar` semantics mean the second occurrence silently overwrites the first. Verified directly:

```
$ ./bin/backstop gate --file README.md --file .github/workflows/ci.yml
  Gate running against 1 explicit files.
```

Only the last `--file` value survives; the first is silently dropped with no warning, no error,
and no indication in the output that a flag was discarded. (A single `--file X` followed by bare
positional args DOES accumulate — `runGate`, `cmd/backstop/gate.go:92`, appends `args...` after
the one `fileValue` — so the flag-repetition case is the only broken shape.) An agent looping
`--file` calls under the mistaken belief the flag is repeatable (the `Use:` string invites exactly
that reading) silently verifies only its last file, believing it verified all of them.

## Impact

Both defects sit in the same command surface an agent is told to use for the "positive case" of
per-file verification (prove each touched file is clean). The first defect produces a false RED
that cannot be fixed by editing the named file — it depends on unrelated repo topology. The second
produces a false sense of coverage — a multi-`--file` invocation silently checks only the last
file named, with no error. Neither is a false GREEN (the honest-failure direction), but both
corrupt exactly the verification discipline (`feedback_never_stash_shared_tree.md`) built to
compensate for shared-tree risk during concurrent multi-agent work.

## Notes / references

- Filed at team-lead's request; evidence gathered by implementer-087-p4 (2026-07-28) against
  untracked-clean files, independently reproduced during authoring (2026-07-27) with identical
  output.
- Sibling to the gate-verdict-honesty cluster named in DIR-024 (ISSUE-066, ISSUE-067, ISSUE-091,
  ISSUE-092) — same failure family (a gate signal that reads authoritative and silently isn't) —
  but a distinct trigger from all four: this is package-target derivation for `--file` scoping,
  not run-filter scoping (ISSUE-066), crash-vs-failure formatting (ISSUE-067), `--all` undercount
  (ISSUE-091), or fixture execution being dead code (ISSUE-092). Per DIR-024's own notes, that
  cluster is "currently cited by no directive" pending a founder home decision — this issue is not
  slotted into DIR-024 and awaits the same decision.
- Distinct from ISSUE-067: ISSUE-067 is about a REAL test failure surfacing as an opaque crash;
  here there is no test failure at all — `go test` has zero Go code in scope and that legitimate
  no-op is what surfaces as the crash.

## Additional evidence

- Corroborated by the orchestrator (2026-08-02/03) while verifying SPEC-018's closure: `gate
  --file specs/SPEC-018-gate-diff-scope.spec.md` crashed with the identical signature
  (`dispatching findings engine "go-test" ... crashed: non-zero exit with no parseable findings:
  exit status 1`). Control run against a different, already-shipped, unrelated spec
  (`specs/SPEC-030-packs-only-native-standards-removal.spec.md`, implemented days prior) on the
  live tree produced the same crash with the same signature — confirming the defect is universal
  to `specs/` targets (a directory with no Go package, the same root cause already documented
  above for directories generally) and is not tied to any one file's content or freshness. A
  separate `go test ./...` run in an isolated detached worktree confirmed the underlying suite
  itself is 100% clean (17 packages, zero failures) — the crash is purely a `--file`
  dispatch-scoping defect, not a real test failure being surfaced.
