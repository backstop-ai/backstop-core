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
