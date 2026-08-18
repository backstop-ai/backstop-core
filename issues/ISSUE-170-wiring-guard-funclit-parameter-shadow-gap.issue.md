---
title: "Wiring Guard Funclit Parameter Shadow Gap"
schema_version: issue/v1

issue:
  id: ISSUE-170
  title: "Wiring Guard Funclit Parameter Shadow Gap"
  type: bug
  status: open
  created: "2026-08-18"

complexity:
  scope: isolated
  uncertainty: known
  risk: safe
---

# Wiring Guard Funclit Parameter Shadow Gap

## Problem

The prober-wiring guard rewritten by ISSUE-165 (`proberWiringViolations` and its rebind scanner
`proberWiringRebindsOf`, both in `pkg/packval/sandbox_wiring_guard_test.go`) has a known,
deliberately-deferred blind spot: a closure whose OWN parameter shadows the outer function's
injected prober parameter defeats the guard silently, producing zero violations even though the
real ABI prober never reaches the call it appears to guard.

### The evasion shape, concretely

```go
func linuxSandboxedRunWith(command string, args []string, packDir string, probeABI LandlockABIProbe) ([]byte, error) {
	run := func(probeABI LandlockABIProbe) ([]byte, error) {
		// uses THIS probeABI -- the closure's own parameter, not the
		// outer function's injected one
		return newSandboxHelperCommand(command, args, packDir, probeABI)
	}
	return run(someFakeProber)
}
```

This is a legitimate, plausible refactor shape — a retry wrapper, a timeout wrapper, or any
closure that happens to reuse the parameter name `probeABI` for its own signature — not a
contrived attack. Go does not warn about the resulting shadowing, and does not warn about the
outer parameter going unused inside the closure (it is still "used" as an argument to `run`).

### Why the guard cannot see it, precisely

`proberWiringViolations` (`pkg/packval/sandbox_wiring_guard_test.go:81`) classifies every tracked
call site by its ENCLOSING `*ast.FuncDecl` — the walk descends into `ast.FuncLit` bodies (a
documented strength: a tracked call nested inside a closure is still attributed to the enclosing
declaration, per the comment at lines 75-80) but resolves WHICH parameter name counts as "the
injected prober" from that same enclosing `*ast.FuncDecl`'s parameter list
(`proberWiringProberParamNames`, line 311), never from the innermost enclosing scope. In the
shape above, the call to `newSandboxHelperCommand` is correctly attributed to the enclosing
`linuxSandboxedRunWith`, and its last argument is the identifier `probeABI` — which matches
`linuxSandboxedRunWith`'s own parameter NAME by spelling, so the guard reports no violation. The
guard has no way to know that the `probeABI` actually in scope at that call site is the closure's
parameter, bound to `someFakeProber`, not the outer function's.

The rebind scanner (`proberWiringRebindsOf`, line 356) does not help either: it looks for
`*ast.AssignStmt`/`*ast.ValueSpec` rebinds of the name within the enclosing function's body, and
a `FuncLit` parameter declaration is neither node shape — it is a new binding introduced by the
function literal's own parameter list, which Go resolves by ordinary lexical scoping, not by
assignment.

### Provenance — found during ISSUE-165, explicitly deferred, not an oversight

This is the third of three silent-green evasions the ISSUE-165 impl-review found against the
rewritten guard (recorded in `.claude/agent-memory/impl-reviewer/project_issue165_wiring_guard_review.md`
as "FuncLit-parameter shadow"). The other two — a dispatch-seam re-bind gap and a
declaration-form (`var`) re-bind gap — were closed in commit `fc2b8ce`. This third one was
deliberately left open: closing it needs FuncLit-parameter-shadow detection in the AST walk (the
guard would need to track, per scope, which parameter binding is currently in effect at each call
site, essentially small-scale scope resolution), which is disproportionate for what is a
test-only regression guard that never runs in production. The guard's own source documents the
deferral in place: `sandbox_wiring_guard_test.go:353-355`, directly above
`proberWiringRebindsOf`'s declaration — "STILL OUT OF REACH, DELIBERATELY: a FuncLit whose own
PARAMETER shadows the prober identifier. That needs FuncLit-parameter-shadow detection and is
tracked as a follow-on to ISSUE-165, not silently absorbed."

## Impact

None today, and none urgent. The guard's existing coverage (dispatch-seam identifier match at the
two neutral seams, enclosing-parameter match plus assignment/declaration re-bind detection at the
two injectable seams) already closes the loopholes that mattered for the original false-positive
ISSUE-165 was filed to fix, and this guard exercises no production code path — it only parses
`sandbox_linux.go` as text at test time. The residual risk is narrow: a future refactor that
introduces a same-named closure parameter around one of the two injectable seams would silently
pass this guard while genuinely defeating the prober injection it exists to protect. That is a
real, if unlikely, regression class — worth tracking, not worth the scope this lane declined to
take on.

## References

- `pkg/packval/sandbox_wiring_guard_test.go:353-356` — the guard's own comment documenting this
  exact deferral, directly above `proberWiringRebindsOf`.
- `pkg/packval/sandbox_wiring_guard_test.go:75-80` — the comment establishing that a tracked call
  nested inside a `FuncLit` IS attributed to its enclosing `FuncDecl` (the property that makes the
  gap possible: attribution happens, but parameter-name resolution does not follow scope).
- `pkg/packval/sandbox_wiring_guard_test.go:81` (`proberWiringViolations`) and `:311`
  (`proberWiringProberParamNames`) — where the enclosing-`FuncDecl`-only parameter resolution
  lives.
- `pkg/packval/sandbox_wiring_guard_test.go:356` (`proberWiringRebindsOf`) — the rebind scanner
  that matches `*ast.AssignStmt` and `*ast.ValueSpec` but has no case for a `FuncLit` parameter
  list.
- `.claude/agent-memory/impl-reviewer/project_issue165_wiring_guard_review.md` — the impl-review
  finding that surfaced this as the third of three new evasions against the rewritten guard,
  naming it "FuncLit-parameter shadow" and recording that the guard "descends into FuncLits
  (advertised as a strength) but resolves the prober name from the enclosing FuncDecl, so the
  names match. Go does not flag the unused outer param."
- `ISSUE-165` (`issues/ISSUE-165-sandbox-abi-probe-wiring-test-naming-mismatch.issue.md`) — origin
  issue whose fix (commits `8d35706`, `fc2b8ce`) closed the other two evasions found in the same
  review pass and explicitly deferred this one.
- `PLAN-ISSUE-165-sandbox-abi-probe-test-parameter-naming.plan.yml` TASK-005 — the closeout task
  that named this follow-on as owed but had no issue-authoring capability to file it.

A fix would extend `proberWiringRebindsOf` (or a sibling function) with scope-aware resolution:
for each tracked call site, walk outward from the call through any enclosing `ast.FuncLit`
boundaries and check whether any of them declares a parameter with the tracked name before
reaching the outer `FuncDecl` — if so, the call site's "enclosing" prober binding is the
closure's own parameter, not the outer function's, and the value passed to `run(...)` (or
equivalent) at the closure's call site becomes the thing that needs checking instead. The exact
mechanism and how much scope-resolution machinery is proportionate is left to this issue's plan,
not decided here.

### Existence-in-world check

Performed 2026-08-18 before authoring: `grep -ril` over `issues/` and `bundles/` for "FuncLit",
"funclit", and "parameter shadow" matched only this issue's own not-yet-authored file and
`ISSUE-165` (which names the deferral but does not own fixing it — its own text and the guard's
own comment both point here). No open issue or bundle charter already owns this surface.
