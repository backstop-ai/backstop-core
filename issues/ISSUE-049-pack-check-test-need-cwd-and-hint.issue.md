---
title: "pack check / pack test Ignore a Positional Pack-Dir Argument"
schema_version: issue/v1

issue:
  id: ISSUE-049
  title: "pack check / pack test Ignore a Positional Pack-Dir Argument"
  type: technical-debt
  status: closed
  created: "2026-07-09"
  closed: "2026-07-09"

resolved-by: 34312d9

complexity:
  scope: isolated
  uncertainty: known
  risk: safe
---

# ISSUE-049: pack check / pack test Ignore a Positional Pack-Dir Argument

## Problem

`backstop pack check` and `backstop pack test` take no positional path
argument — both hardcode `.` as the pack directory and read `pack.yml` from
the current working directory only, silently discarding any path a caller
passes. This breaks the natural first-use flow that immediately follows
`pack new`, exactly the launch-onboarding path.

### Repro (live smoke test of the pack authoring loop, 2026-07-09)

```
$ backstop pack new --type engine --language go --slug smoke-check
...scaffolds ./smoke-check/...

$ backstop pack check ./smoke-check
ERROR [phase1-structural/parse-manifest] reading manifest pack.yml: open pack.yml: no such file or directory
```

`./smoke-check` is accepted on the command line but never reaches the
manifest reader — it is silently dropped. The error names the bare `pack.yml`
with no indication that a path argument was even considered, which reads as
"the scaffold produced a broken pack" when the scaffold is actually fine. The
only way to make `pack check` (or `pack test`) succeed is to `cd smoke-check`
first and re-run with no argument:

```
$ cd smoke-check && backstop pack check
# passes
```

### Root cause

Both commands hardcode `"."` as the pipeline root and declare their args
parameter as `_ []string`, i.e. the RunE signature explicitly discards
whatever positional arguments cobra parsed:

```go
// cmd/backstop/pack_check.go:16,28
RunE: func(cmd *cobra.Command, _ []string) error {
    ...
    p := packval.NewPipeline(".", packval.PipelineOptions{Mode: "check", Format: format})
```

```go
// cmd/backstop/pack_test_cmd.go:16,27
RunE: func(cmd *cobra.Command, _ []string) error {
    ...
    p := packval.NewPipeline(".", packval.PipelineOptions{Mode: "test", Format: format})
```

Neither command declares `Args: cobra.MaximumNArgs(1)` (or any `Args`
validator) either, so cobra doesn't even reject an unexpected extra argument
— it is accepted syntactically and then thrown away semantically.

### Why it matters

This is a UX papercut, not a correctness bug — the scaffolded pack itself is
valid: run from inside the pack directory, both `pack check` and `pack test`
pass with exit 0, and a genuinely broken manifest still exits 1 (fail-loud
works). But every new pack author hits this on their very first command after
`pack new`, and the error message actively misleads — it looks like the
scaffold is broken, not like the CLI dropped an argument.

It went undetected by ISSUE-032's end-to-end authoring-loop test
(`cmd/backstop/pack_authoring_loop_test.go`,
`TestPackAuthoringLoop_EndToEnd`) because that test drives `pack check` and
`pack test` as a subprocess with `cmd.Dir` set to the pack directory (i.e. it
"cd's in" via subprocess cwd) rather than invoking `pack check <path>` from
the project root:

```go
// cmd/backstop/pack_authoring_loop_test.go:91
if out, code := runBackstop(t, bin, packDir, "pack", "check"); code != 0 {
```

That exercises the no-arg / run-from-pack-dir path exclusively and never
exercises `pack check <path>` from a different cwd — the exact ergonomics a
real author reaches for first.

### Partial mitigation already shipped

DIR-017's `pack new` hardening added a next-step hint to scaffold output
(`pkg/pack/scaffold.go`, `ScaffoldResult.HumanString`, guarded by a
`scaffold_test.go` assertion) so the `cd`-first requirement is at least
discoverable:

```go
// pkg/pack/scaffold.go:84
sb.WriteString(fmt.Sprintf("\nNext: cd %s && backstop pack check   # then: backstop pack test\n", r.Slug))
```

This makes the workaround visible but does not fix the underlying gap —
`pack check <path>` and `pack test <path>` still silently ignore the
argument. This issue tracks the underlying fix; the hint is a stopgap, not a
resolution.

## Solution

Direction for the eventual plan (not prescribed — the planner owns the
design):

- Give both `pack check` and `pack test` an optional positional pack-dir
  argument (e.g. `pack check ./smoke-check`) that resolves `<path>/pack.yml`
  — either by passing `<path>` as the pipeline root instead of the hardcoded
  `"."`, or by changing into `<path>` before reading. Preserve the existing
  no-arg (run-from-pack-dir) behavior for back-compat — this is additive, not
  a breaking change.
- Add an `Args` validator (e.g. `cobra.MaximumNArgs(1)`) so an unexpected
  second argument is rejected instead of silently ignored, matching the
  "silently dropped input" theme this issue is about.
- When a path is given but has no `pack.yml`, the error should name the given
  path (e.g. `no pack.yml in ./smoke-check`), not the bare cwd-relative
  `pack.yml` — the current message is what made this look like a scaffold
  defect instead of an argument-handling gap.
- Verify with a REAL CLI test (subprocess binary, matching ISSUE-032's
  `pack_authoring_loop_test.go` style) that runs `pack check <path>` and
  `pack test <path>` from the project root — i.e. `cmd.Dir` is the project
  root, not the pack directory — the exact gap ISSUE-032's e2e test missed by
  cd'ing in via `cmd.Dir`.

## References

- `cmd/backstop/pack_check.go:16,28` — `newPackCheckCommand`, hardcodes `"."`
  and discards args (`_ []string`)
- `cmd/backstop/pack_test_cmd.go:16,27` — `newPackTestCommand`, same pattern
- `cmd/backstop/pack_authoring_loop_test.go:91` — `TestPackAuthoringLoop_EndToEnd`,
  the ISSUE-032 e2e that cd's in via `cmd.Dir` and never exercises
  `pack check <path>` from a different cwd
- `pkg/pack/scaffold.go:84` — `ScaffoldResult.HumanString`, the DIR-017
  next-step hint that mitigates (but does not fix) this gap
- DIR-017 (Pack CLI Authoring & Distribution Hardening) — the directive this
  issue rounds out; DIR-017 itself is done, this is a follow-on papercut
  surfaced by live verification of its acceptance criteria
- ISSUE-032 (pack-cli-authoring-loop-reboot) — closed; its e2e test's
  cd-in style masked this gap

## Resolution

`pack check` and `pack test` now accept an optional `[pack-dir]` positional
argument (`cobra.MaximumNArgs(1)`), defaulting to the current directory when
omitted. The argument is passed straight through to
`packval.NewPipeline(packDir, ...)` as the pipeline root — the same struct
that reads `<packDir>/pack.yml` — so `pack check ./<slug>` immediately after
`pack new` now works from the project root instead of silently ignoring the
path and failing. A missing manifest now names the given path
(`<path>/pack.yml`) rather than the bare cwd, so the error correctly points
at an argument-handling gap instead of reading like a broken scaffold.
No-arg invocation from inside the pack directory is unchanged (back-compat
preserved — additive, not breaking).

A real CLI regression test, `TestPackVal_PackCheckCommand_PathArg` in
`cmd/backstop/pack_val_cmd_test.go`, runs `pack check <path>` from the
*parent* cwd (not `cmd.Dir`-into-the-pack-dir), closing the exact blind spot
ISSUE-032's e2e test left open (it set `cmd.Dir` to the pack directory,
masking the missing-arg-handling gap). It asserts a valid pack at the given
subdir passes and that a bad path's report names `nonexistent/pack.yml`.

Live-verified end-to-end from the project root: `pack new` → `pack check
./<slug>` and `pack test ./<slug>` both pass (exit 0); a bad path errors
naming the given path; no-arg-from-inside still works. Gate is green (exit
0); artifact validate is green.

This supersedes/completes the d78b30a mitigation (the `pack new` cd-hint),
which remains a helpful nicety but is no longer the only recourse — the
underlying argument-handling gap it worked around is now fixed directly.
