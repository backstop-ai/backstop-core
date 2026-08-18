---
title: "A relative packDir silently breaks every macOS sandboxed convert step (exit 71, stderr discarded)"
schema_version: issue/v1

issue:
  id: ISSUE-147
  title: "A relative packDir silently breaks every macOS sandboxed convert step (exit 71, stderr discarded)"
  type: bug
  status: open
  created: "2026-08-16"

complexity:
  scope: contained
  uncertainty: known
  risk: critical
---

# ISSUE-147: A relative packDir silently breaks every macOS sandboxed convert step (exit 71, stderr discarded)

## Problem

On macOS, passing a **relative** `packDir` to the sandboxed convert path makes
`sandbox-exec` refuse to apply its profile at all, and the caller sees only an
opaque `exit status 71` with zero diagnostic content. The same pack, same
bytes, same binary, invoked with an **absolute** `packDir` succeeds.

Discovered and isolated 2026-08-16 by `implementer-issue092` during
PLAN-ISSUE-092 verification. Reproduced back-to-back, stable:

```
$ backstop pack test packs/substantiveness                                    # relative — FAILS
... 4x opaque "exit status 71" convert failures ...

$ backstop pack test /Users/bmanson/src/projects/backstop-core/packs/substantiveness   # absolute — SUCCEEDS
... convert succeeds ...
```

This affects **every** `pack test`/`pack check` invocation on macOS where the
pack directory is passed as a relative path — which is the common case (e.g.
`backstop pack test packs/substantiveness` run from the repo root, exactly as
shown above). It is producing bogus failures for anyone using a relative path
tonight; the only workaround currently circulating is "use absolute paths."

### Root cause

`darwinSandboxProfile` (`pkg/packval/sandbox_nonlinux.go:72-92`) calls
`filepath.EvalSymlinks(packDir)` directly on the argument it is given, with no
`filepath.Abs` call first:

```go
func darwinSandboxProfile(packDir string) string {
	resolved := packDir
	if r, err := filepath.EvalSymlinks(packDir); err == nil {
		resolved = r
	}
	readSubpaths := []string{
		resolved, // the pack directory itself (the only project path)
		...
```

Go's `filepath.EvalSymlinks` preserves relativity: given a relative path with
no symlinks to resolve in its existing prefix, it returns a relative path
back (or a path resolved only up to the point it exists, still relative). A
relative `resolved` value is then embedded verbatim into the generated
`sandbox-exec` profile as a `(subpath "packs/substantiveness")` clause
(`sandbox_nonlinux.go:93-96`).

`sandbox-exec` **rejects a relative `subpath` clause** in a profile. It does
not treat it as relative-to-cwd; it fails to apply the profile at all and
exits with **71 (`EX_OSERR`, "could not apply the profile")** — a shell-level
sandbox setup failure that has nothing to do with the convert script's actual
logic or output. This is why the failure looks opaque: the exit code names a
sandbox bootstrap failure, not a script/interpreter error, but nothing in the
current code path says so.

This is the same file, same profile-construction function, that ISSUE-029
fixed for a different reason (dyld read denials) — ISSUE-029's fix already
established that `resolved` must be the kernel-resolved path for a
`sandbox-exec` subpath rule to match at all; it did not also establish that
`resolved` must be absolute, which is the missing half.

### Ruled out before concluding (per the discovering implementer)

- Profile-with-empty-stdin behaves identically on both paths (not an
  input/stdin issue).
- The ast-grep payload is byte-identical (2145 bytes) on both paths.
- File modes are identical (0755) on both paths.
- `diff -rq` of the two pack trees is clean.
- A copy of the pack placed at an absolute-path location works; a copy of the
  same pack placed under a relative-path location fails.
- The only variable that moves the outcome is whether the `packDir` argument
  passed in is absolute or relative.

### Compounding defect, same file: sandbox-exec's stderr is discarded

`platformSandboxedRunStdout` (`sandbox_nonlinux.go:133-149`) sets `c.Stdout`
to a buffer but never sets `c.Stderr`:

```go
c := sandboxExecCommand(cmd, args, packDir)
if stdin != nil {
	c.Stdin = bytes.NewReader(stdin)
}
var stdout bytes.Buffer
c.Stdout = &stdout
if err := c.Run(); err != nil {
	return stdout.Bytes(), fmt.Errorf("sandboxed run (stdout) failed: %w", err)
}
```

`sandbox-exec` writes its own diagnostic (something to the effect of "could
not apply the profile") to stderr on this failure; that text is discarded
before it ever reaches the operator, leaving only the bare, unexplained
`exit status 71`. This is the same "information existed, surfacing did not"
pattern as ISSUE-145 (a separate stderr-blindspot issue) — noted here because
it shares the exact file and call site as the primary defect, and because a
future opaque exit-71 will be just as undiagnosable as this one was unless
both are fixed together.

### Also manifests on the validator seam, with a different visible symptom (added 2026-08-17)

The root cause described above is not confined to the convert step. It also breaks the
**validator** seam (`RunValidator`, `pkg/packval/executor.go`) — same relative-packDir cause,
same silently-mismatching sandbox `subpath` clause, but a *different* visible failure signature,
which is what let this go unrecognized as ISSUE-147 the first time it was hit on this seam.

**Why the two seams look unrelated even though they share one cause:**

- **Convert seam** (`platformSandboxedRunStdout`, `sandbox_nonlinux.go:133-149`): when
  `sandbox-exec` refuses to apply a profile with a relative `subpath` clause, the command exits
  71 and its stderr — which would carry `sandbox-exec`'s own "could not apply profile" text — is
  discarded (the compounding defect documented above). The caller sees a bare, opaque
  `exit status 71`.
- **Validator seam** (`RunValidator`, `pkg/packval/executor.go:261-267`): here the profile *does*
  apply — it's just a relative `subpath` clause that matches nothing, so the sandboxed process
  gets `Operation not permitted` reading its own pack files. `RunValidator` collapses that run
  failure into `Passed:false` with a **nil error**, and the phase-3 fixture harness
  (`pkg/packval/phase3.go:141`) fires the same generic message —
  `ERROR [phase3-fixtures/validator-positive] layer3 positive failed` — whether the validator ran
  and returned a genuinely wrong verdict, or never usefully ran at all because the sandbox denied
  it its own pack. (The harness's **negative**-fixture branch, `phase3.go:163-168`, does
  distinguish these cases; only the positive branch does not.) A relative packDir therefore
  produces a message that reads exactly like a real fixture-polarity bug in the scaffolded
  validator, not like a sandbox access failure — which is what makes it a convincing false lead.

**Concrete instance: `TestPackAuthoringLoop_EndToEnd`.** This darwin-only acceptance test
(`cmd/backstop/pack_authoring_loop_test.go`) runs `pack test` from *inside* the freshly-scaffolded
pack via its `runBackstop` helper (`cmd.Dir = packDir`, no path argument), which is exactly the
relative/implicit-packDir shape this issue describes (`pack test` with no argument defaults
`packDir` to `"."`, `cmd/backstop/pack_test_cmd.go:29`). It fails at
`phase3-fixtures/validator-positive` with the generic message above. This was misdiagnosed once
as a distinct, untracked defect and filed as `ISSUE-162` — retracted and closed as a duplicate of
this issue once both the repo's backlog-pm auto-triage hook (which traced this exact mechanism
from source, `.claude/agent-memory/backlog-pm/project_relative_packdir_masquerades.md`) and a
direct reproduction (same scaffolded pack, relative packDir fails at
`phase3-fixtures/validator-positive`; identical pack, absolute packDir passes clean across all
six phases) independently corroborated it. See `ISSUE-162`'s Resolution section for the full
retraction record.

The backlog-pm memory file's standing triage guidance is now independently corroborated and
should be treated as settled, not merely asserted: **any darwin `pack test` failure reading
`ERROR [phase3-fixtures/validator-positive] layer3 positive failed` is this issue's mechanism
until an absolute-packDir re-run says otherwise** — one command (re-run the same pack with an
absolute path) distinguishes a real fixture-polarity bug from this sandbox defeat, and should be
run *before* suspecting the fixture/validator itself.

**Blast radius correction.** This issue was originally filed as convert-step-only. The actual
blast radius is larger: `pack test`/`pack check` run from inside any validator-bearing pack on
macOS is exposed on the validator seam too, not just at the convert step. Still invisible to CI
(darwin-skip on the test side, Landlock — not `sandbox-exec` — on Linux), so this remains a
local-dev-experience defect on macOS, not a CI blocker, but the fix in this issue's Solution
section (guarantee `darwinSandboxProfile`'s embedded path is absolute) resolves both seams at
once since it corrects the shared root cause; no separate fix is needed for the validator seam.

### Platform scope: darwin-only, not shared with Linux

Verified against `pkg/packval/sandbox_linux.go`: the Linux path
(`newSandboxHelperCommand` → Landlock-based helper) does not string-embed
`packDir` into a textual profile clause that a sandbox mechanism parses and
matches — Landlock rules are established via file descriptors
(`ConvertValidatorCapability`, `helper.Dir = packDir`), which resolve
relative paths against the process's actual working directory the normal
POSIX way. There is no equivalent "relative subpath literal rejected by the
sandboxing mechanism" failure mode on Linux. This issue is darwin-specific,
scoped to `pkg/packval/sandbox_nonlinux.go`.

## Solution

Not prescribed in full, direction only:

1. In `darwinSandboxProfile`, call `filepath.Abs(packDir)` before (or as part
   of) the `filepath.EvalSymlinks` call, so the value embedded in the
   `sandbox-exec` profile's `subpath` clause is always absolute regardless of
   what the caller passed in. `sandbox-exec` profile subpaths must be
   absolute; the function should guarantee that invariant internally instead
   of depending on every caller to pass an absolute `packDir`.
2. In `platformSandboxedRunStdout` (and audit `platformSandboxedRun`'s
   `CombinedOutput` path for the same gap), capture `sandbox-exec`'s stderr
   on failure and fold it into the returned error so an opaque exit-71 (or
   any other sandbox-exec failure) carries a legible diagnostic instead of a
   bare exit code.

## Verification

Not prescribed in full — for the planner. At minimum:

- A regression test that invokes the real macOS sandboxed convert path
  (`platformSandboxedRunStdout` or the `pack test`/`pack check` entry point
  above it) with a **relative** `packDir` and asserts it succeeds, mirroring
  the way ISSUE-029's `TestSandboxConvertWithRealInterpreter` proved the dyld
  fix via a real, un-stubbed `sandbox-exec` call rather than the
  `stubSandboxedRunStdout` bypass.
- A test asserting that a `sandbox-exec` failure's stderr is surfaced in the
  returned error (not merely that an error is returned).
- Confirm `platformSandboxedRun`'s `CombinedOutput()` path is unaffected by
  the stderr-capture fix (it already interleaves stderr via
  `CombinedOutput`), so the fix to `platformSandboxedRunStdout` doesn't
  regress it.

## References

- `pkg/packval/sandbox_nonlinux.go:72-92` — `darwinSandboxProfile`, the
  `filepath.EvalSymlinks` call with no preceding `filepath.Abs`.
- `pkg/packval/sandbox_nonlinux.go:133-149` — `platformSandboxedRunStdout`,
  the discarded-stderr compounding defect.
- ISSUE-029 (closed) — fixed a different defect in this same function
  (dyld read denials from an under-scoped read-allowlist); established that
  `resolved` must be the kernel-resolved path, not that it must be absolute.
- ISSUE-020 (closed) — the sibling Linux facet of the sandbox interface;
  confirmed unaffected by this defect (see Platform scope above).
- ISSUE-145 — separate stderr-blindspot issue; same failure pattern
  ("information existed, surfacing did not"), different call site. Not
  folded in here; noted because this issue's compounding defect is an
  instance of the same class, in a file ISSUE-145 does not cover.
- Discovered during PLAN-ISSUE-092 verification, 2026-08-16, by
  `implementer-issue092`.
- `pkg/packval/executor.go:261-267` — `RunValidator`, where the validator-seam manifestation
  collapses a sandbox run failure into `Passed:false` with a nil error (added 2026-08-17).
- `pkg/packval/phase3.go:141` — the positive-fixture check that fires the generic
  `layer3 positive failed` message regardless of cause; contrast with the negative branch at
  `phase3.go:163-168`, which does distinguish.
- `cmd/backstop/pack_authoring_loop_test.go` — `TestPackAuthoringLoop_EndToEnd`, the concrete
  darwin-only instance of the validator-seam manifestation.
- `cmd/backstop/pack_test_cmd.go:29` — `pack test` with no path argument defaults `packDir` to
  `"."`, the relative-packDir trigger this test's `runBackstop` helper hits by construction.
- `ISSUE-162` (retired 2026-08-17, `status: replaced`, `replaced-by: ISSUE-147`) — filed as a
  distinct defect for the validator-seam symptom above, retracted once traced to this issue's
  mechanism. Its Resolution section carries the full retraction record and both lines of
  corroborating evidence.
- `.claude/agent-memory/backlog-pm/project_relative_packdir_masquerades.md` — the backlog-pm
  auto-triage memory that first traced the validator-seam mechanism in tree while triaging
  `ISSUE-162`, and states the standing rule this section restates: treat
  `phase3-fixtures/validator-positive` on darwin as this issue's mechanism until an
  absolute-packDir re-run says otherwise.
