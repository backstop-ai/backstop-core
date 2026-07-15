---
title: "Gate/Engine Quality"
number: DIR-024
created: "2026-07-15"
schema_version: directive/v1

directive:
  status: queued
  source:
    - "ISSUE-007"
    - "ISSUE-020"
---

## Description

Two gate/engine-quality gaps that don't fit the other three newly-added
directives' themes:

1. **Cross-platform sandbox — Linux is a hard no-op (ISSUE-020).**
   `pkg/packval/sandbox.go`'s `SandboxedRun` / `SandboxedRunStdout` dispatch
   on `runtime.GOOS`: the `darwin` branch wraps pack-supplied convert
   scripts and sandbox-validators under `sandbox-exec` (deny-default,
   deny-network, deny-file-write); the `linux` branch unconditionally
   returns `"sandbox unavailable on linux in this build"` — confirmed still
   the case in the current tree. Since the caller treats any sandbox error
   as a hard failure, no pack convert script or validator can run on Linux
   at all today: the gate is non-functional on Linux, not merely
   unsandboxed. This is security-elevated — the OS sandbox is the only
   trust boundary between arbitrary pack-supplied scripts and the host —
   and must close before CI runs on any Linux host or any third-party pack
   is installed. Candidate mechanisms (bubblewrap, Landlock, seccomp, user
   namespaces) are noted in the issue for the planner, not decided; any
   implementation must hold deny-network/deny-write parity with the macOS
   profile and fail loud rather than silently pass through if the chosen
   mechanism is unavailable on the host kernel.
2. **Build/test pass exclusion mechanism (ISSUE-007).** Originally: no
   `exclude_paths`-style mechanism exists for trees that must not gate
   build/test (e.g. intentionally-uncompilable fixture directories). The
   motivating case in this repo (`prototype/`) was deleted 2026-06-12, and
   the issue itself demoted this to "defer until a real consumer needs it."
   **Flagging for the founder:** this issue's own file still cites
   `buildExecutor`/`testExecutor` in `pkg/check/check.go` as the mechanism
   to extend — a `grep` against the current tree found neither identifier;
   they were removed by the later native-toolchain cutover
   (`project_native_toolchain_cutover`, 2026-07-03) that deleted
   `builtinToolchain`/`realCodeChecker` outright. The underlying concern
   (repos need a way to declare out-of-scope trees for pack-driven
   build/test steps) is still generically valid post-packs-only, but the
   concrete fix site this issue names no longer exists — it needs
   re-scoping to wherever pack-declared build/test steps live today before
   a plan is written against it, not a literal read of the issue's current
   "Fix sketch."

## Notes

Grouped together as the catch-all for gate/engine-quality gaps that aren't
contracts-engine, pack-distribution, or traceability themed. Both issues
are pre-existing and low-urgency (ISSUE-007 was already self-demoted by its
author to "defer until a real consumer needs it"; ISSUE-020's real-world
exposure is explicitly noted as low pre-launch/local-only/no-Linux-CI-yet,
though it is rated `risk: critical` in its own file once that changes).
Positioned last of the four newly-added directives — no deadline pressure,
no bundle depending on either landing.
