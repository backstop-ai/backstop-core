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
    - "ISSUE-082"
    - "ISSUE-075"
    - "ISSUE-077"
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
3. **Tool allowlist unreachable entries + overstated guarantee (ISSUE-082).**
   `engine.TrustedToolAllowlist()` (`pkg/pack/engine/allowlist.go`) declares
   eight pinned tool entries, but every call site of `CheckToolAllowed`
   (`cmd/backstop/pack_gate.go:813`, `pkg/pack/manifest.go:547`,
   `pkg/packval/executor.go:63`, `cmd/backstop/pack_gate_provision.go:85`)
   exempts bindings with a nil `Provision`. A sweep of every pack.yml in
   this repo and in `~/src/projects/backstop-packs` found `provision:`
   blocks for only three tools (grep, ast-grep, semgrep) — so `rg`,
   `oxlint`, `bun`, `tsc`, `prettier` are dead entries. `typescript-toolchain`
   (the pack those four were added for) declares no `provision:` at all; it
   shells out via `npx --no-install`.
   Two defects, both cleanup-only: dead entries carrying a
   `// nosemgrep: no-baked-language-token` suppression (the `backstop/self`
   rule fired correctly on a TypeScript tool name in a core Go file, and the
   fix was a suppression rather than not adding the entry — removing the
   entries removes the suppression), and a doc comment claiming a tool
   absent from the map "may not be run by any pack-declared command, no
   matter what a pack declares," which is false given the nil-`Provision`
   exemption. What the map actually governs is the trust floor for tools
   backstop provisions and pins on the user's behalf via the Provision/lock
   path.
   Scope boundary: this is deletion + doc correction only. The governance
   question for arbitrary pack-declared commands outside the Provision path
   belongs to bundles/BUNDLE-021-pack-command-execution-governance.bundle.md
   (exploring, not yet scoped). Do NOT design or implement an enforcement
   mechanism against ISSUE-082.
4. **Smoke fixture vacuous green on a missing mandated test (ISSUE-075).**
   `TestSmoke_GateFailsMissingMandatedTest` (`tests/smoke/smoke_test.go:486`)
   is meant to prove that `test_verification` blocks on a spec's missing
   mandated test, but its `createSpec` helper hardcodes `status: draft` with
   no override — so ISSUE-054's implemented-only mandated-test scoping
   filters the test out before enforcement ever runs, and the scenario
   passes green while proving nothing. The fix is scoped and small: add a
   status override to `specOpts`/`createSpec`, force `SPEC-999` to
   `implemented` for this scenario, and re-run the smoke suite with
   `-count=1` to surface any sibling scenarios cache-masked the same way.
5. **Stale gate binary produces phantom violations (ISSUE-077).** Bare
   `backstop` on `PATH` resolves to whichever copy was last manually
   rebuilt, which can be days stale relative to the working tree; a stale
   binary parsing a pack's updated rule-message format has already produced
   3 phantom violations against a correct diff (the ISSUE-062 incident),
   while `go test` — which always builds from source — passed clean. Two
   complementary fixes are proposed in the issue: make the `PATH` entry a
   shim that rebuilds-then-execs the fresh binary, and a defense-in-depth
   self-check at gate startup that exits 2 (not a phantom violation) if the
   running binary is older than the newest tracked `.go` file. Primarily a
   contributor on-ramp problem for this repo, not consumer-facing — a
   released install has only one binary copy and cannot diverge from itself.

## Notes

Grouped together as the catch-all for gate/engine-quality gaps that aren't
contracts-engine, pack-distribution, or traceability themed. The original
three issues are pre-existing and low-urgency (ISSUE-007 was already
self-demoted by its author to "defer until a real consumer needs it";
ISSUE-020's real-world exposure is explicitly noted as low pre-launch/
local-only/no-Linux-CI-yet, though it is rated `risk: critical` in its own
file once that changes; ISSUE-082 is dead-code + doc cleanup with no
runtime exposure). Positioned last of the four newly-added directives — no
deadline pressure, no bundle depending on any of them landing.

ISSUE-075 and ISSUE-077 (backlog-pm slotted, 2026-07-26) are both scoped
and low-urgency in the same vein — a smoke-fixture gap and a local
dev/contributor on-ramp trap, neither consumer-facing nor blocking other
work — so they ride along here for thematic fit (gate/engine quality)
rather than displacing ISSUE-020's priority within this directive.

Sequencing caveat: DIR-024 holds position 3 in BACKLOG.yml on the strength
of ISSUE-020 — Linux/CI viability is one of the four founder-designated
launch blockers (the other three being recipes/SPEC-054/DIR-019, remote
pack consumption/DIR-026/SPEC-055, and CI-driven releases/DIR-001, tiered
up 2026-07-27). ISSUE-082 is tier-2 by both the founder's
launch razor and its own priority note; it rides along in this directive
for thematic fit only (gate/engine quality), not because it shares
ISSUE-020's urgency. Anyone working this directive top-down should land
ISSUE-020 first — ISSUE-082 (and ISSUE-007) must not be picked up ahead of
it.
