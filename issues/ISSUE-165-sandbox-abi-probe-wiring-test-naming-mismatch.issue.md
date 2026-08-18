---
title: "Sandbox Abi Probe Wiring Test Naming Mismatch"
schema_version: issue/v1

issue:
  id: ISSUE-165
  title: "Sandbox Abi Probe Wiring Test Naming Mismatch"
  type: bug
  status: open
  created: "2026-08-18"

complexity:
  scope: isolated
  uncertainty: known
  risk: safe
---

# Sandbox Abi Probe Wiring Test Naming Mismatch

## Problem

`TestSandboxLinux_ProductionPathUsesTheRealABIProbe`
(`pkg/packval/sandbox_linux_errors_test.go:146`) is a structural "wiring guard" test: it parses
`sandbox_linux.go` as Go source (`go/parser`), walks the AST for every call site to three
tracked function names (`linuxSandboxedRunWith`, `linuxSandboxedRunStdoutWith`,
`newSandboxHelperCommand`), and asserts the LAST ARGUMENT at each call site is literally the
bare identifier `probeLandlockABI` — `t.Errorf` on any site that isn't. Real Linux CI now
reaches this test (once `ISSUE-163`'s `TestMain` fix let more of the suite run to completion on
GitHub's Linux runner) and it fails:

```
sandbox_linux_errors_test.go: TestSandboxLinux_ProductionPathUsesTheRealABIProbe: a production
call to newSandboxHelperCommand at sandbox_linux.go:221:17 does not pass probeLandlockABI as
its prober. The sandbox would negotiate its ABI through something other than the kernel, which
is the test/production divergence this guard exists to prevent
```

**This is a test-authoring defect, not a security or correctness bug.** The production sandbox
wiring is correct; only the assertion mechanism that checks it is wrong.

### What the test's own comment says it's checking, and where it actually breaks

The test's own comment (`sandbox_linux_errors_test.go:153-155`) states it checks two hops:

> "the two dispatch delegations that pass the prober down, and the two inner functions that
> hand it to `newSandboxHelperCommand`"

The first hop is fine. `platformSandboxedRun` (`sandbox_linux.go:211-213`), the actual
production entry point, calls:

```go
func platformSandboxedRun(command string, args []string, packDir string) ([]byte, error) {
	return linuxSandboxedRunWith(command, args, packDir, probeLandlockABI)
}
```

— the literal identifier `probeLandlockABI` at that call site, exactly as the guard expects.

The second hop is where it breaks. `linuxSandboxedRunWith` itself
(`sandbox_linux.go:218-228`) has the signature:

```go
func linuxSandboxedRunWith(command string, args []string, packDir string, probeABI LandlockABIProbe) ([]byte, error) {
	helper, err := newSandboxHelperCommand(command, args, packDir, probeABI)
	...
```

The call at `sandbox_linux.go:221` forwards `probeABI` — the function's OWN PARAMETER — to
`newSandboxHelperCommand`. That parameter is deliberately injectable: the function's doc
comment (`sandbox_linux.go:216-217`) states it is "`platformSandboxedRun`'s body with the ABI
prober injected," precisely so tests can drive it with a fake prober
(`TestLinuxSandboxedRunWith_WrapsThePrepareFailure` does exactly this, passing `unavailable` at
`sandbox_linux_errors_test.go:214`). `linuxSandboxedRunStdoutWith`
(`sandbox_linux.go:245-...`) has the identical shape at its own call to
`newSandboxHelperCommand` (`sandbox_linux.go:246`).

So the inner hop is a correctly-parametrized, testable delegation — by design it CANNOT also
pass the literal `probeLandlockABI` identifier, because the whole point of the parameter is to
hold whichever prober the caller supplied, real or fake. The test's naive check (bare
identifier-name equality against the literal `probeLandlockABI`) cannot distinguish "a parameter
that legitimately HOLDS the real prober value at runtime, forwarded correctly" from "a
hardcoded wrong prober." It structurally cannot pass for this delegation chain unless the
parameter happens to share the exact literal name `probeLandlockABI` at every hop — which is a
naming coincidence, not a property of correct wiring.

### Why this was never caught before tonight

Confirmed via `git log`: neither `sandbox_linux.go` nor `sandbox_linux_errors_test.go` has been
touched by any commit involved in tonight's investigation. Both files are `//go:build linux`
gated and have never compiled on the darwin machine this test suite is normally authored and
run on. `ISSUE-163`'s `TestMain` fix (landed at `970512b`) is what let Linux CI reach this test
for the first time — it did not cause the failure, only exposed a pre-existing one.

## Impact

None on the actual sandbox — `platformSandboxedRun`'s real dispatch chain does pass the real
`probeLandlockABI` at runtime; this was confirmed by direct code reading, not by trusting the
test. The impact is entirely to CI signal: this test currently fails loudly on every Linux CI
run once it's reached, misreporting a correctly-wired delegation as broken, which will train
whoever triages Linux CI failures to either distrust or waive this test rather than treat its
output as meaningful.

## References

- `pkg/packval/sandbox_linux_errors_test.go:138-200` —
  `TestSandboxLinux_ProductionPathUsesTheRealABIProbe`, the guard test with the false-positive
  assertion.
- `pkg/packval/sandbox_linux.go:211-213` — `platformSandboxedRun`, the real production entry
  point; its call to `linuxSandboxedRunWith` correctly passes the literal `probeLandlockABI`.
- `pkg/packval/sandbox_linux.go:218-228` — `linuxSandboxedRunWith`; its call to
  `newSandboxHelperCommand` at line 221 forwards its own `probeABI` parameter, which the guard
  misreads as a hardcoded wrong prober.
- `pkg/packval/sandbox_linux.go:245-262` (approx.) — `linuxSandboxedRunStdoutWith`, the sibling
  delegation with the identical shape.
- `ISSUE-163` / commit `970512b` — the `cmd/backstop` `TestMain` fix that let Linux CI reach
  this test for the first time tonight; not the cause of this defect, only what exposed it.
- CI run 32108003542 (`gate-report.json`, `pack_engines` step, rule
  `backstop-ai/go-toolchain/go-test`, file `sandbox_linux_errors_test.go`) — the real Linux CI
  failure this issue was filed from.

A fix likely means either (a) renaming `linuxSandboxedRunWith`'s and
`linuxSandboxedRunStdoutWith`'s parameter to literally `probeLandlockABI` so it happens to
satisfy the naive check at every hop, or (b) rewriting the guard to trace parameter provenance
(does this parameter's value originate from the real `probeLandlockABI` at the outermost
production call?) instead of doing a flat identifier-name match. The choice between these — or
another approach entirely — belongs to this issue's plan, not to this filing.

### Existence-in-world check

Performed 2026-08-18 before authoring: `grep -ril` over `issues/` and `bundles/` for
`probeLandlockABI`, "ABI probe", and `sandbox_linux_errors` matched only `ISSUE-020`
("Cross Platform Sandbox Linux Noop"), which is the historical issue that introduced the
injectable-prober seam this guard tests — not a duplicate of this specific test-naming defect —
and this issue's own not-yet-authored file. No open issue or bundle charter already owns this
surface.
