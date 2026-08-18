---
name: ast-wiring-guards-assert-spelling
description: AST "wiring guard" tests in pkg/packval assert an identifier's SPELLING at a call site, not a value's provenance — so they false-red on any correctly-parametrized delegation, and the obvious rename "fix" is a vacuous green
metadata:
  type: project
---

`pkg/packval`'s Linux sandbox lane ships **AST wiring-guard tests** — tests that
`go/parser`-parse a production `.go` file and assert that the last argument at every
tracked call site is a specific bare identifier. `TestSandboxLinux_ProductionPathUsesTheRealABIProbe`
(`sandbox_linux_errors_test.go`, `//go:build linux`) is the archetype, authored by
`PLAN-ISSUE-020`; it produced `ISSUE-165` (DIR-024 item 21).

**The structural defect class:** such a guard can only pass at a hop where the real
value is written literally. At any hop that forwards an **injectable parameter** — the
seam that exists so tests can drive a fake — the assertion is **unsatisfiable by
construction**. In ISSUE-165: `sandbox_linux.go:214`/`:240` pass literal
`probeLandlockABI` ✓; `:221`/`:246` forward `probeABI` ✗. Production was correct the
whole time; only the assertion was wrong.

**★ The trap worth catching every time:** the obvious fix — *rename the parameter to
match the literal* — makes the test **green without making it true**. The parameter
shadows the package-level function, so the assertion then passes for any caller-supplied
prober including a fake: it checks a declaration's *spelling*, not a value's
*provenance*. That is a vacuous green under the founder's never-hack-the-gate-green law.
A plan slug naming "parameter naming" is the tell.

**Two companion hazards in the same test shape:**
- `t.Errorf` (not Fatalf) means **every** bad site reports — an issue quoting one
  message usually understates the count.
- These guards close with an exact `callSites != N` assertion whose stated purpose is
  anti-vacuity. Any fix changing the tracked-identifier set must update `N` in the same
  change, or a false red becomes a *different* false red. **Post-rewrite the assertion is
  `dispatch != 2 || forward != 2 || dispatch+forward != 4` (`:434`) AND a table-driven
  falsification harness asserting per-case `wantDispatch`/`wantForward` (`:1131`,
  `:1136`) — there are now TWO places to keep coherent, not one.**

**★ UPDATE 2026-08-18 — ISSUE-165 IS FIXED, and the guard MOVED.** Commits
`8d35706` (rewrite, seam-aware) + `fc2b8ce` (two more evasions closed) replaced the
linux-tagged guard with **`pkg/packval/sandbox_wiring_guard_test.go`, which carries NO
build tag and no `_linux` filename component, deliberately** (12-line header says so) —
so this guard now **runs and falsifies on darwin**. DIR-024 item 21's "darwin-invisible,
only Linux CI falsifies it" line is stale and note-superseded. The rewrite also resolves
the prober parameter **BY TYPE** (`*ast.Field` of type `LandlockABIProbe`), not by name;
option (a) is refused mechanically. `ISSUE-165` is still `status: open` and
`PLAN-ISSUE-165` still `draft` BY DESIGN — TASK-005 forbids closing on local evidence
("fix landed, awaiting CI confirmation").

**The residual family it filed (both 2026-08-18, ~30s apart, from TASK-005):**
`ISSUE-169` (stale `newSandboxHelperCommand` doc comment the fix inverted → DIR-024 item
25, sibling PM) and **`ISSUE-170` (FuncLit-parameter shadow → DIR-024 item 24, mine)** —
the third of three impl-review evasions, deferred on purpose and documented verbatim in
the guard at `:353-355`. **ISSUE-170 inverts DIR-024's usual polarity: it is a SILENT
green, not a loud red.** It still homes DIR-024 on the ISSUE-137 line — *product-surface
false-green → DIR-032; repo test-harness false-green → DIR-024* — with item 4 (ISSUE-075)
as affirmative precedent. Say the inversion out loud in any slot; a reader will notice.

**★ THE LANE'S CLOSEOUT FILES A BATCH — expect exactly two (2026-08-18).**
`PLAN-ISSUE-165` TASK-005 clause (5) mandates `ISSUE-169` (the stale doc comment
on `newSandboxHelperCommand`, `sandbox_linux.go:163-175`) and clause (6)
("silence is not a discharge") produced `ISSUE-170` (the guard's `FuncLit`
parameter-shadow blind spot). Both slotted **DIR-024** as items **25** and
**24** — by *provenance* (filed by a DIR-024 item-21 lane), NOT by the usual
"loud red" charter test, because **neither reds anything**: 169 is prose, 170 is
a *silent green*. Item 21 + 24 + 25 are ONE family in ~40 lines of `pkg/packval`
and the cheapest possible single lane. Reject `DIR-021` for 169 — its charter is
artifact-corpus debt, not prose in a Go file.

**The correction to re-make if that comment is ever re-triaged:** ISSUE-169 says
the sentence was accurate until ISSUE-165 "inverted" it. It welds TWO claims —
*"the test asserts this"* (true at authoring, false now) and *"both production
call sites hand it the real `probeLandlockABI`"* (**never true**; `:221`/`:246`
always forwarded `probeABI`, which is why the guard went red). Two falsehoods,
one of them original. A planner reading only the issue fixes one.

**How to apply:** false-positive loud reds home to **DIR-024** (see
[[project_dir033_and_the_three_way_gate_boundary]] — goes RED ⇒ DIR-024). Enumerate the
call sites yourself with grep before accepting the issue's framing; and always say in
the item that a `//go:build linux` test's only falsifier is Linux CI, never a darwin
local run.
