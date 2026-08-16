---
title: "pkg/packval/executor.go's *exec.Error-only never-started check misses path-ful engine commands — silent vacuous pass on backstop pack test/check"
schema_version: issue/v1

issue:
  id: ISSUE-140
  title: "pkg/packval/executor.go's *exec.Error-only never-started check misses path-ful engine commands — silent vacuous pass on backstop pack test/check"
  type: bug
  status: open
  created: "2026-08-16"

complexity:
  scope: contained
  uncertainty: known
  risk: critical
---

# packval executor's narrow never-started check misses path-ful engine commands

## Problem

`DefaultExecutor.RunEngine` (`pkg/packval/executor.go:62-97`, the command dispatch behind
`backstop pack test` / `backstop pack check` phase3 fixtures) decides whether an engine command
was a "broken run" using only:

```go
runErr := cmd.Run()
var execErr *exec.Error
if errors.As(runErr, &execErr) {
    return ExecutionResult{Passed: false, ...}, fmt.Errorf("engine %q failed to run: %w", ...)
}
```

`*exec.Error` is produced ONLY when `exec.Command` resolves a BARE command name via `LookPath`
and that lookup fails. It is never produced for a PATH-FUL command (`filepath.Base(name) !=
name`, e.g. `./scripts/lint.sh`, `packs/foo/bin/checker`) — that shape instead fails at fork/exec
time with `*fs.PathError{Op: "fork/exec"}`. `buildEngineArgv` (`executor.go:40-51`) builds `name`
directly from `binding.Command`, which is pack-declared DATA — nothing prevents (or excludes) a
path-ful command there.

When such a command is absent, non-executable, or has a bad interpreter line, `cmd.Run()` returns
the `*fs.PathError` shape, `errors.As(runErr, &execErr)` is false, and execution falls through to:

```go
findings, parseErr := check.ParsePackFindings(stdout.Bytes())
...
return ExecutionResult{Passed: len(findings) > 0, ...}, nil
```

`stdout` is empty (the process never ran). `check.ParsePackFindings` → `parseSarif`
(`pkg/check/parsers.go:130-133`) has a deliberate lenient case for this: `bytes.TrimSpace(out)`
empty returns `(nil, nil)` — no error. So `RunEngine` returns `ExecutionResult{Passed: false,
ExitCode: 0}, nil` — a clean, error-free result. For a NEGATIVE fixture (the case expected to
produce zero findings), `Passed: false` is exactly the SUCCESS condition phase3 checks for. The
fixture reads as correctly passing even though the declared engine binary never started.

This is the exact "narrow error-shape" defect class ISSUE-112 fixed on the `backstop gate`
dispatch path (`cmd/backstop/pack_gate.go`'s `runNeverStarted`, which matches BOTH `*exec.Error`
and a fork/exec-shaped `*fs.PathError`) — but ISSUE-112's own Direction section explicitly assumed
the packval executor already covered this ("the packval executor already does this; the gate
dispatch path does not"). That assumption is false for the path-ful case: `pack_gate.go:408-414`
(the `runNeverStarted` doc comment) itself calls out that packval's check is "*exec.Error-only"
and narrower by construction — the gate's widened check was written to be strictly broader than
packval's, but packval's own check was never widened to match.

## Impact

`backstop pack test` / `backstop pack check` — the pack-quality gate that is supposed to prove a
pack's declared engine actually runs — can report a fixture (and therefore the whole phase3 step)
clean when the pack's engine command never executed at all. A path-ful engine command that is
missing, non-executable, or carries a bad interpreter line is indistinguishable from "engine ran,
found nothing" on a negative fixture. This is the same vacuous-green failure mode ISSUE-112
diagnosed taking hours to track down on the gate path, now open on the pack-authoring path
instead — where it is arguably worse, since `pack test`/`pack check` is the tool an author is
supposed to trust BEFORE a pack ever reaches `backstop gate`.

## Direction

Widen `pkg/packval/executor.go`'s `RunEngine` never-started check to match `*fs.PathError` with
`Op == "fork/exec"` in addition to `*exec.Error`, the same two-shape check `runNeverStarted`
(`cmd/backstop/pack_gate.go:418-428`) already implements for the gate dispatch path. Check
whether the predicate can be shared (e.g. hoisted into a package both `pkg/packval` and
`cmd/backstop` can import without violating an existing import-boundary constraint — PLAN-ISSUE-118
hit a comparable constraint where `pkg/gate` could not import `pkg/pack/engine`) or whether
packval needs its own copy for architectural reasons; either is acceptable, but two copies must
not drift in which shapes they catch (that drift is the root cause here — `pack_gate.go`'s
already-widened check documents the gap in packval's without packval itself ever being updated).

## Notes

- Sibling/parent: ISSUE-112 (`issues/ISSUE-112-engine-tool-missing-silent-vacuous.issue.md`) —
  fixed this exact error-shape narrowness on the `backstop gate` dispatch path
  (`runNeverStarted` in `cmd/backstop/pack_gate.go`); this issue is the same defect class on the
  `backstop pack test`/`pack check` path that ISSUE-112 assumed was already safe.
- Repro sketch: a pack declaring a findings engine via a path-ful `command:` (e.g.
  `./scripts/checker.sh`) pointing at a non-executable or missing file, exercised by phase3 against
  a negative fixture. Expected: `pack test`/`pack check` fails loud, naming the broken command.
  Actual (current code): the fixture step reports pass.
- Traced mechanism confirmed by reading current code 2026-08-16: `executor.go:86-89` (narrow
  check), `pkg/check/parsers.go:130-133` (`parseSarif`'s empty-input `(nil, nil)` case — the
  specific reason the fallthrough is silent rather than merely mislabeled).
