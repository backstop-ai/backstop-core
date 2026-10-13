---
title: "Darwin Convert Sandbox Denies Framework Python"
schema_version: issue/v1

issue:
  id: ISSUE-224
  title: "Darwin Convert Sandbox Denies Framework Python"
  type: bug
  status: open
  created: "2026-10-13"

complexity:
  scope: contained
  uncertainty: known
  risk: moderate
---

# Darwin Convert Sandbox Denies Framework Python

## Problem

On macOS, a pack convert script that runs `python3` aborts inside the convert sandbox when the
`python3` first on `PATH` is the python.org installer build rather than Homebrew or the system
Python.

Mechanism, verified 2026-10-13 while confirming the bash-toolchain fix for ISSUE-219:

- `/usr/local/bin/python3` is a symlink into `/Library/Frameworks/Python.framework/Versions/3.12/`,
  where the python.org installer puts its framework build.
- `darwinSandboxProfile` (`pkg/packval/sandbox_nonlinux.go`) grants file reads only for the pack
  directory, `/usr/lib`, `/System/Library`, and the Intel and Apple-Silicon Homebrew prefixes.
  `/Library/Frameworks` is not in the list.
- dyld therefore cannot load the interpreter's own framework:
  `dyld: Library not loaded: /Library/Frameworks/Python.framework/Versions/3.12/Python ...
  (file system sandbox blocked open())`, exit 134 (SIGABRT).
- The same convert succeeded once Homebrew's `python@3.13` came first on `PATH`.

### Reproduction

macOS with a python.org Python first on `PATH`: run any pack whose convert invokes `python3`
(for example `backstop-ai/bash-toolchain`'s `scripts/test-to-sarif.sh` on its failure branch)
through `backstop gate` or `backstop pack test`.

## Impact

- Any python-based convert fails on macOS machines using the python.org installer, a common way to
  install Python. Which Python you have, not what your code does, decides whether the gate works.
- The failure is reported as a convert-step failure (`exit status 1` or a SIGABRT), and the
  sandbox's stderr is not shown, so the cause is hard to find. This is the same evidence-loss class
  as ISSUE-209, ISSUE-216 and ISSUE-219.

## Root cause

The read allowlist names Homebrew and system locations only. It has no entry for framework installs
under `/Library/Frameworks`.

## Direction

(Direction, not a plan.)

1. Add a scoped read subpath for interpreter frameworks, for example
   `/Library/Frameworks/Python.framework`, and keep it narrow rather than all of `/Library`.
   Alternatively, document that converts require a Homebrew or system interpreter and detect the
   framework case up front with a clear message.
2. Carried over from ISSUE-219's optional core item: surface the sandboxed run's stderr in the
   convert-failure message, and consider a private writable `TMPDIR` for converts that legitimately
   need scratch space.

## Notes

- Related: ISSUE-029 (the original darwin read allowlist), ISSUE-216 (relative pack dir),
  ISSUE-219 (here-document temp file).
