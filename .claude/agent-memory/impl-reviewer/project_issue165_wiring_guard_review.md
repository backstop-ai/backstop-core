---
name: issue165-wiring-guard-review
description: ISSUE-165 AST wiring-guard rewrite — code sound but 3 NEW identifier-shadow evasions found; the reusable recipe is extracting the real checker verbatim into a scratch module and driving it with your own fixtures
metadata:
  type: project
---

ISSUE-165 (commit `8d35706`) rewrote `pkg/packval/sandbox_wiring_guard_test.go`'s
`proberWiringViolations` to classify prober-carrying call sites by ENCLOSING FuncDecl.
Round-1 review had already closed three loopholes (positional-last decoy, `=` re-bind,
else-branch precedence). Reviewing it I found **three more, all silent green**:

1. **Dispatch-seam shadow** — `probeLandlockABI := someFake` (or the `var` form) inside
   `platformSandboxedRun`, then forwarded. B3's re-bind scan exists ONLY for bucket B.
2. **`var`-form re-bind** — B3 scans only `*ast.AssignStmt`; a `var x T = fake` DeclStmt
   in a nested block evades it.
3. **FuncLit-parameter shadow** — a closure inside the seam declaring its OWN
   `probeABI LandlockABIProbe` param, forwarding it, called with a fake. The checker
   descends into FuncLits (advertised as a strength) but resolves the prober name from
   the enclosing FuncDecl, so the names match. Go does not flag the unused outer param.

**Why:** any AST guard that asserts an identifier SPELLING is defeated by re-binding or
shadowing that spelling. Closing one bucket's re-bind hole and not the other's, or closing
`=`/`:=` and not `var`/closure-param, leaves the same class open under a different syntax.

**How to apply:** when reviewing any `go/ast` wiring/spelling guard, enumerate every Go
construct that can re-bind a name — `=`, `:=`, `var`, `const`, a FuncLit parameter, a
nested-block redeclaration, a range/type-switch binding — and test each. Reuse the recipe:
`head -N <guard_test.go> > scratch/checker.go`, rewrite `package packval` -> `package main`,
drop `testing`, add a tiny driver that walks `*.gosrc` fixtures. That runs the REAL checker
code, not a reimplementation. Then compile the evasion shapes in a throwaway module to prove
they are legal vet-clean Go, not reviewer fantasy.

Also recurring on this lane family: TASK-005 (record the ceiling in the issue artifact +
file the surfaced follow-on) went undischarged again — same as [[project_issue163_sandbox_guard_review]].
Check `issues/` mtimes and `status:` before signing off on any lane whose final task is
documentation. See [[reference_sandbox_verification]].
