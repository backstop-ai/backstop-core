---
name: path-scans-need-symlink-resolution
description: "Any scan that COMPARES a walk path against a separately-derived expected directory must EvalSymlinks both sides, not just filepath.Abs — on macOS t.TempDir() is /var/... while os.Getwd() resolves to /private/var/..., and the mismatch yields one finding per artifact, which looks exactly like a working implementation"
metadata:
  type: project
---

`filepath.Abs` is NOT enough to make two paths comparable. `os.Getwd()` returns a symlink-RESOLVED
path, so a caller that passes `"."` (which `runGate` does whenever `config.DiscoverConfigPath` fails)
produces walk paths under `/private/var/...`, while a `Root` resolved from the same directory's
`t.TempDir()` spelling stays under `/var/...`. Nothing matches.

**Why it is dangerous rather than merely annoying:** the failure mode is not a crash. A
per-kind location predicate comparing `filepath.Dir(path)` against `Root.Dir(kind)` reports EVERY
artifact as misplaced (or, mirrored, none at all) — and both outcomes are indistinguishable from a
correct implementation unless a test compares the relative and absolute forms of ONE project root
against each other. That comparison is the only thing that catches it.

**How to apply:** in a scan of this shape, resolve symlinks on entry for BOTH the walk root and the
root the expected directories are derived from, with a fallback that returns the path unchanged when
it cannot be resolved (it may not exist yet). Then report in the RESOLVED vocabulary consistently —
mixing a resolved walk path with an unresolved expected directory in one finding is self-contradictory
in the report. Tests over such a scan must anchor their relative-path expectations on the same
resolved form, or they fail for a reason that has nothing to do with the predicate.

Landed in `pkg/gate/artifact_status.go` (`FindUngatedArtifacts` / `resolveSymlinks`), SPEC-068
CLM-067 and the spec's trap C.
