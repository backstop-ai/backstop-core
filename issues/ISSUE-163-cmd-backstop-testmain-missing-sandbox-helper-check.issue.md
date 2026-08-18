---
title: "Cmd Backstop Testmain Missing Sandbox Helper Check"
schema_version: issue/v1

issue:
  id: ISSUE-163
  title: "Cmd Backstop Testmain Missing Sandbox Helper Check"
  type: bug
  status: open
  created: "2026-08-17"

complexity:
  scope: contained
  uncertainty: known
  risk: moderate
---

# Cmd Backstop Testmain Missing Sandbox Helper Check

## Problem

Every test in package `cmd/backstop` that triggers real Linux sandboxed dispatch against a
local pack dies before the dispatched command ever runs, because that package's `TestMain`
(`cmd/backstop/integration_test.go`) never checks whether it was re-exec'd as a sandbox helper
before unconditionally running `go build`. This is the confirmed root cause of the majority of
the 62 CI violations that failed the v0.2.0 release attempt on GitHub's `ubuntu-latest` Linux
runner — it does not reproduce on darwin and never has, because the mechanism it collides with
is Linux-only.

### The two mechanisms that collide

**Mechanism A — the Linux sandbox's re-exec design.** `pkg/packval/sandbox_linux.go`'s
`newSandboxHelperCommand` implements sandboxed dispatch by RE-EXECUTING THE CURRENTLY RUNNING
BINARY as a "helper" subprocess: it resolves `self, _ := os.Executable()` (the path to whatever
binary or test binary is currently running), builds `exec.Command(self)` with `Dir` set to the
pack directory being validated (e.g. `packs/substantiveness`), and sets the environment variable
`BACKSTOP_SANDBOX_HELPER_SPEC` to tell that re-exec'd copy "you are now the sandbox helper —
apply Landlock/seccomp restrictions, then exec the real target command." The dispatch happens in
`pkg/packval/sandbox_linux_helper.go`'s `runSandboxHelper`.

**Mechanism B — `cmd/backstop/integration_test.go`'s `TestMain`.** This is the first code that
runs whenever the compiled `cmd/backstop` test binary starts, for any reason. It unconditionally
runs `go build -o <tmp> .` (with `cmd.Dir = "."`, i.e. the process's current working directory)
as its very first action, with no prior check of any kind.

**The collision.** For any test living in package `cmd/backstop` that triggers real sandboxed
dispatch (concretely: `TestSubstantivenessFixtures_RealPackTestPassesPhase3` in
`cmd/backstop/substantiveness_fixture_polarity_test.go`, which drives
`packval.NewPipeline(absPackDir, ...).Run()` against `packs/substantiveness`, causing a real
sandboxed `ast-grep`-convert dispatch), the re-exec'd "helper" copy IS THAT SAME `cmd/backstop`
test binary — so it goes through `TestMain` again on startup. `TestMain` has zero awareness it
might be running as a sandbox helper: it unconditionally tries `go build -o <tmp> .` FIRST,
using its current working directory — which the sandbox trampoline has already set to the PACK
directory (e.g. `packs/substantiveness`) via `Dir` on the parent's `exec.Cmd`. That directory has
no `.go` files, so `go build` fails immediately with Go's own `"no Go files in <dir>"`, which
`TestMain` wraps as `"failed to build binary: %v"` and calls `os.Exit(1)`. The process dies here,
entirely BEFORE the `BACKSTOP_SANDBOX_HELPER_SPEC` env var check (`runSandboxHelper`) ever gets a
chance to run — so the actual intended command (the pack's real `ast-grep/to-sarif.sh` convert
script) never executes at all, and packval reports it up the stack as a generic engine-run
failure.

### Proof this exact problem was already solved once, for a sibling package

`pkg/packval/main_test.go`'s own `TestMain` (a different package that also has integration-style
tests triggering the sandbox) does this correctly, as the first statement in the function:

```go
func TestMain(m *testing.M) {
	// FIRST STATEMENT. When this process was spawned as a sandbox helper a
	// successful call never returns; otherwise it returns nil immediately, having
	// done nothing.
	if err := MaybeRunSandboxHelper(); err != nil {
		fmt.Fprintf(os.Stderr, "backstop sandbox helper: %v\n", err)
		os.Exit(126)
	}
	os.Exit(m.Run())
}
```

`cmd/backstop/main.go` confirms the REAL production binary's `main()` also calls this the same
way — `runWith(stdout, stderr, packval.MaybeRunSandboxHelper, NewRootCommand)` — and its own
comment states explicitly that the intended test-side twin of this call is
`packval.MaybeRunSandboxHelper()` in `pkg/packval`'s `TestMain`. The pattern was known and
documented; it was simply never propagated to `cmd/backstop/integration_test.go`'s `TestMain`,
which goes straight into the `go build` step with no `MaybeRunSandboxHelper()` call anywhere in
it.

### How this was confirmed, not guessed

1. Read the real CI failure log and downloaded the actual `gate-report.json` artifact from the
   failed v0.2.0 CI run — extracted 62 distinct violations, the large majority resident in
   `cmd/backstop` E2E/integration tests dispatching against `packs/contracts` and
   `packs/substantiveness` as local packs.
2. Reproduced partially in a local Docker Linux container. Confirmed Docker Desktop's `linuxkit`
   kernel lacks Landlock entirely, which rules that environment out for a full repro but did
   confirm the sandbox mechanism's refusal path behaves correctly when Landlock is genuinely
   absent — i.e. the container result is a different, expected failure mode, not this one.
3. Opened a temporary, throwaway debug PR (`#2` against `backstop-ai/backstop-core`, since closed
   and its branch deleted, never merged) that added diagnostic `printf`s inside
   `pkg/packval/sandbox_linux_helper.go`'s `applyRestrictionsAndExec`, immediately before the
   final `unix.Exec` call, then watched it run on real GitHub Actions infrastructure. The
   diagnostic printfs DID appear for two other sandboxed tests living in `pkg/packval` itself
   (`TestLinuxSandbox_NetworkAllowedControlLegSucceeds`,
   `TestLinuxSandbox_RealInterpreterRunsUnderTheFilter` — which have their own correctly-behaving
   `TestMain`), but NEVER appeared for `TestSubstantivenessFixtures_RealPackTestPassesPhase3`,
   which still showed the exact same `"no Go files ... failed to build binary"` error, completely
   unchanged by the added instrumentation. That proves the re-exec'd helper process for that test
   dies before ever reaching the instrumented code path, which is only reachable AFTER
   `runSandboxHelper` has successfully taken over.
4. Grepped the whole repo for the literal string `"failed to build binary"` and found exactly one
   source: `cmd/backstop/integration_test.go:33`, inside `TestMain` — confirming the mechanism
   precisely.

### Blast radius

Every test living in package `cmd/backstop` (not just
`TestSubstantivenessFixtures_RealPackTestPassesPhase3`) that triggers real sandboxed dispatch
against a local pack on Linux hits this same collision. This accounts for the majority of the 62
CI violations seen on the v0.2.0 release attempt — dozens of distinct test names across
contracts, substantiveness, and init dimensions, all `cmd/backstop`-resident, all failing with
variations of `"pack add ... pack test for ... failed"` that trace back to this same root
mechanism.

It does NOT reproduce on darwin — confirmed across this whole investigation's local testing,
which never hit it, since `pkg/packval/sandbox_linux.go` and
`pkg/packval/sandbox_linux_helper.go` are `//go:build linux`-gated and never compile on macOS —
and it does not affect `pkg/packval`'s own tests, which already carry the correct `TestMain`
guard.

### Context — why this surfaced only now

This was discovered during the same v0.2.0 release investigation that produced `ISSUE-158`
(already closed). It is a separate, deeper layer that surfaced only once `ISSUE-158`'s fix and
the earlier semgrep-pin fix (`7851d5f`) cleared the way for CI to run far enough on Linux to
reach it for the first time ever — prior CI failures on this release attempt were masking it.

### What the fix is (stated for context only — not implemented here)

Add the same `MaybeRunSandboxHelper()` first-statement check to
`cmd/backstop/integration_test.go`'s `TestMain`, mirroring `pkg/packval/main_test.go`'s exact
pattern, before its `go build` step. Issues never carry the fix — this belongs in a plan.

## References

- `pkg/packval/main_test.go` — the correct precedent: `TestMain` calling
  `MaybeRunSandboxHelper()` as its first statement.
- `cmd/backstop/main.go` — the production binary's `main()`, which documents the intended
  pattern (`runWith(stdout, stderr, packval.MaybeRunSandboxHelper, NewRootCommand)`) and names
  `pkg/packval`'s `TestMain` as its test-side twin.
- `cmd/backstop/integration_test.go` — the defect site: `TestMain` runs `go build -o <tmp> .`
  unconditionally with no `MaybeRunSandboxHelper()` call, and is the sole source in the repo of
  the literal error string `"failed to build binary"`.
- `cmd/backstop/substantiveness_fixture_polarity_test.go` —
  `TestSubstantivenessFixtures_RealPackTestPassesPhase3`, the primary reproducing test, driving
  `packval.NewPipeline(absPackDir, ...).Run()` against `packs/substantiveness`.
- `pkg/packval/sandbox_linux.go` (`newSandboxHelperCommand`) and
  `pkg/packval/sandbox_linux_helper.go` (`runSandboxHelper`, `applyRestrictionsAndExec`) — the
  Linux sandbox re-exec mechanism this collides with. Both `//go:build linux`-gated.
- `ISSUE-158` — the prior, now-closed issue from the same v0.2.0 release investigation; its fix
  (and the earlier semgrep-pin fix at `7851d5f`) is what let CI run far enough on Linux to reach
  this defect for the first time.
