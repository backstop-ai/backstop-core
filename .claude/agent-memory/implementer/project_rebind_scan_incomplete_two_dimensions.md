---
name: rebind-scan-incomplete-two-dimensions
description: An AST identifier-binding guard's re-bind scan is incomplete in TWO independent dimensions at once — node shape (AssignStmt misses ValueSpec) and seam coverage (applied to the injectable seam but not the dispatch seam) — and the strictest-sounding seam is the one that ends up undefended
metadata:
  type: project
---

When a guard asserts "this call forwards identifier X", the re-bind/shadow scan that
backs it is incomplete in **two independent dimensions**, and finding one does not
find the other. Both were live simultaneously in PLAN-ISSUE-165's guard and were
caught only by an impl-reviewer prototyping adversarially against the shipped code.

**Dimension 1 — NODE SHAPE.** `*ast.AssignStmt` covers `x = v` and `x := v`. It does
NOT cover `var x T = v`, which the parser gives as `*ast.DeclStmt` → `*ast.GenDecl` →
`*ast.ValueSpec` — a different node type entirely. In a nested block that is legal,
compiling Go that shadows the parameter for every statement after it. An
assignment-only scan returned ZERO violations on it. Match `*ast.ValueSpec` too.

**Dimension 2 — SEAM COVERAGE.** The scan was invoked only inside the branch for the
*injectable* seam (`if len(forwardCalls) > 0`). The *dispatch* seam — the one whose
own comment called it "the ONE place the real prober enters the chain, and nothing
downstream can recover from getting it wrong here" — had no re-bind protection at
all. `probeLandlockABI := someFake` one line above the call made the forwarded name
spell correctly while meaning something else: ZERO violations at the strictest seam.

**Still out of reach after both fixes:** a `FuncLit` whose OWN PARAMETER shadows the
outer prober identifier. Needs FuncLit-parameter-shadow detection; deferred to a
follow-on rather than absorbed.

**Why:** an identifier-binding guard is not dataflow provenance (that needs
`go/types`, which a guard over a file that does not build for the host platform
cannot use). The re-bind scan IS the whole mitigation, so a hole in it is a hole in
the guard's only defence against a correctly-spelled forward carrying a wrong value.

**How to apply:** when writing or reviewing any AST guard that resolves an identifier
against a declaration, enumerate BOTH axes explicitly — every node shape that can
create a binding (`AssignStmt`, `ValueSpec`, `FuncLit`/`FuncDecl` params, `RangeStmt`,
`TypeSwitchStmt`), and every seam class the guard classifies, not just the one whose
branch you happened to write the scan inside. A cheap tell for dimension 2: the scan
sits inside an `if len(someBucket) > 0` block. Also flag a blank-identifier (`_`)
parameter as its own violation — otherwise the "must forward its OWN parameter `_`"
message is nonsense advice that is itself evidence of the defect.
See [[absence-tests-via-goast]] and [[mutation-matrix-beats-sequence-red]].
