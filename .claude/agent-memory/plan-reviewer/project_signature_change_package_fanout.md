---
name: signature-change-package-fanout
description: A signature change to a Go function radiates to EVERY caller in the same package (compile unit) + any `var Fn = thatFunc` seam + its prod call sites; scope them all
metadata:
  type: project
---

When a plan changes the SIGNATURE (arity) of a Go function, the change does not stay
local — Go compiles a package as one unit, so EVERY caller in the package must update or
the whole package fails to compile (and the Phase-1 red tests can't even build for the
intended reason). Empirically grep `grep -rln "funcName\b" pkg_or_dir/` BEFORE trusting a
plan's file scope.

**Three collateral surfaces to enumerate for any signature change:**
1. **Other production call sites** — especially a seam var `var fooFn = foo` (changing
   `foo`'s signature breaks the var binding) AND every call through that seam var.
2. **The seam's own behavioral contract** — a second prod caller may intentionally want the
   OLD behavior (e.g. code_check.go dispatches packs "always full scope"). The new param
   forces a decision there; the plan must STATE it (pass nil/full-scope) or an implementer
   silently narrows an untouched path.
3. **Every test file** that calls the function OR monkeypatches the seam var with a
   `fooFn = func(oldArgs){...}` closure — those closures must match the new signature.

**How to apply:** Run `grep -rln "<funcName>\b" <pkgdir>/*.go` (prod) and `..._test.go`
(tests). Confirm the plan's task `files` arrays cover the prod fan-out AND every existing
test file in the result — not just the new test file the plan adds. A plan that names only
the function's own file + a new test file, when 10 existing test files + a second prod
caller also reference it, is a compile-break FAIL.

Caught in PLAN-ISSUE-010 (dispatchPackEngines gains a scope param): plan scoped only
pack_gate.go + gate.go + 2 new test files, but the change radiates to code_check.go:18/140
(prod seam `dispatchPackEnginesFn`, "always full scope") and 10 existing test files
(bridge/code_check/code_check_errors/dispatch_engines/dogfood/gate_buildsteps/
pack_gate_errors/pack_gate_gotoolchain/pack_gate_helpers/pack_gate _test.go). Related to
[[retired-feeder-test-collateral]] and [[project-dispatch-consumer-edges]].
