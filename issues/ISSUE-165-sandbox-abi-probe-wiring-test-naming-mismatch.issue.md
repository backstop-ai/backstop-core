---
title: "Sandbox Abi Probe Wiring Test Naming Mismatch"
schema_version: issue/v1
delivered_by: PLAN-ISSUE-165

issue:
  id: ISSUE-165
  title: "Sandbox Abi Probe Wiring Test Naming Mismatch"
  type: bug
  status: closed
  created: "2026-08-18"
  closed: "2026-08-19"

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

## Resolution

Fixed across two commits, delivered by `PLAN-ISSUE-165` (`status: completed`).
`TestSandboxLinux_ProductionPathUsesTheRealABIProbe` was a structural wiring-guard test that
flat-matched call-site argument identifiers against the literal name `probeLandlockABI`, which
cannot distinguish "a parameter that legitimately holds the real prober, forwarded correctly"
from "a hardcoded wrong prober" — a test-authoring defect, not a production wiring bug.

- `8d357064` — moved the guard out of the `//go:build linux`-gated
  `sandbox_linux_errors_test.go` into an untagged file (`sandbox_wiring_guard_test.go`) so it's
  darwin-visible, and rewrote it to classify each tracked call site by its enclosing function
  (provenance-aware) rather than doing a flat identifier-name match.
- `fc2b8ce` — closed 2 more evasions an adversarial review found against the rewritten guard: a
  dispatch-seam re-bind gap (the `platformSandboxedRun` → `linuxSandboxedRunWith` seam had no
  re-bind protection) and a declaration-form re-bind gap (the re-bind scanner missed a `var x T =
  fake` `*ast.ValueSpec` form).

Confirmed genuinely proven, not just locally verified: real Linux CI runs `32314302525` and
`32315586649`, both `conclusion: success` on `main`.

**Not absorbed by this closure.** A third evasion found in the same adversarial review pass — a
`FuncLit` whose own parameter shadows the outer injected prober — was deliberately deferred, not
fixed here. It was filed separately as `ISSUE-170`, which remains open.

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

## Verification ceiling — fix landed, awaiting CI confirmation (2026-08-18)

**The fix has landed on `main`, across three commits. This issue stays `open` — do not read this
note as a close.**

- `8d35706` — the primary fix: moved the guard from `sandbox_linux_errors_test.go`
  (`//go:build linux`) into a new, untagged file, `pkg/packval/sandbox_wiring_guard_test.go`, and
  rewrote `proberWiringViolations` to classify each tracked call site by its ENCLOSING FUNCTION
  rather than doing a flat identifier-name match — closing the false positive this issue
  documents. `sandbox_linux.go` itself is untouched by this commit; only the guard's own logic
  changed.
- `fc2b8ce` — two more evasions closed after adversarial impl-review prototyped the rewritten
  guard against the real shipped code and found three new silent-green gaps beyond the two prior
  review rounds: (1) a **dispatch-seam re-bind gap** — the dispatch seam (`platformSandboxedRun`
  → `linuxSandboxedRunWith`, and its stdout sibling) had no re-bind protection at all, despite the
  guard's own docs calling it the more critical seam; (2) a **declaration-form re-bind gap** — the
  re-bind scanner matched only `*ast.AssignStmt` (`=`/`:=`), missing a `var x T = fake`
  `*ast.ValueSpec` declaration in a nested block, which is legal Go and evaded the scan entirely.
  A third, unrelated cosmetic fix landed alongside: a blank-identifier (`_`) prober parameter
  previously produced nonsense advice instead of being flagged as its own defect. All three are
  pinned by dedicated fixtures, each confirmed via mutation testing (fixture fails when its rule
  is removed). A fourth evasion found in the same review pass — a `FuncLit` whose own parameter
  shadows the outer injected prober — was explicitly deferred, not fixed here; see `ISSUE-170`.

### Why local verification is capped by construction

`pkg/packval/sandbox_linux_errors_test.go` remains `//go:build linux`-gated and is therefore
invisible to every native darwin command on this machine — proved, not assumed: a deliberate type
error introduced into that file left `go build ./...`, `go vet ./pkg/packval/`, and
`go test ./pkg/packval/` all clean on darwin, while `GOOS=linux GOARCH=amd64 go test -c` against
the same tree caught it immediately. So local evidence establishes only that the rewritten guard
— now living in the untagged `sandbox_wiring_guard_test.go`, which DOES run on darwin — passes
against the same `sandbox_linux.go` CI reads, and that the remaining linux-tagged test file
(`sandbox_linux_errors_test.go`) still compiles under a Linux cross-compile. It never establishes
that the CI failure this issue was filed from is resolved — that requires a real Linux CI run,
which this lane cannot produce locally.

### CI-watch criterion — state this precisely

Workflow `.github/workflows/ci.yml`, job `gate` (display name "Backstop Gate", `runs-on:
ubuntu-latest`), blocking step "Run the gate." Download the `gate-report` artifact from a CI run
at or after commit `fc2b8ce` and confirm: the `backstop-ai/go-toolchain/go-test` violations
contain **no result** on `sandbox_linux_errors_test.go` or `sandbox_wiring_guard_test.go` whose
message contains "does not pass probeLandlockABI." There is only ever one such row when present,
never two — the converter emits one result per failing test regardless of how many individual
`t.Errorf` assertions fire inside it. Do **not** gate this confirmation on `.scope.files`
containing `pkg/packval` — that package's `go-test` binding is `exempt_from_scope_filter: true`
and runs project-wide regardless of diff scope, so its absence from the scope file list proves
nothing either way. **Total violation count is not a valid confirmation criterion** — CI runs are
not comparably scoped to each other, so a lower (or higher) count establishes nothing about this
specific fix.

**Partial real confirmation already obtained, as of this writing.** CI run `32143000202` (commit
`8d35706`, the primary-fix commit, completed with overall conclusion `failure` for unrelated
reasons) was downloaded and read directly: `pkg/packval/sandbox_linux_errors_test.go` and
`pkg/packval/sandbox_wiring_guard_test.go` both appear in `.scope.files`, the string
`probeLandlockABI` does not appear anywhere in the report's violations, and neither file produces
any `backstop-ai/go-toolchain/go-test` violation. This confirms the PRIMARY fix (`8d35706`) alone
already cleared Linux CI's original false-positive. It does **not** yet confirm `fc2b8ce`'s two
additional evasion closures — the CI run for that commit (`32150146086`) was still `in_progress`
at authoring time, not completed. Treat the primary defect as CI-confirmed fixed and the two
adversarial-evasion closures as still awaiting their own CI read.

State plainly: fix landed, awaiting full CI confirmation. Do not claim or imply the complete fix
(through `fc2b8ce`) is confirmed working, and do not close this issue.
