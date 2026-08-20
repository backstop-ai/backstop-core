---
name: run-the-generalized-predicate-never-grep-for-imports
description: Decide "should this plan also generalize the guard?" by RUNNING the generalized predicate through the guard's own AST check — a grep for an import path matches forbidden-import literals in tests asserting the opposite
metadata:
  type: project
---

When a plan is offered "also generalize the structural guard while you're in here", derive the
generalized predicate's ACTUAL verdict against the current tree before deciding. Do not reason
about what it would flag — run it.

★ AND RUN IT THE WAY THE GUARD DOES: an AST membership test (a real `*ast.ImportSpec`), never
`grep -rl "<import path>"`. **A grep for an import path matches STRING LITERALS, and leaf
packages keep the import path of what they must NOT import in a forbidden-list literal inside
their own invariant tests.** The grep hit is literally the test proving the opposite of what
the grep implies.

**Measured instance (PLAN-ISSUE-180, 2026-08-19 — the mistake, and the correction).**
My first draft decided "stay narrow" because generalizing `cmd/backstop/sandbox_helper_testmain_guard_test.go`
would supposedly red `pkg/pack/engine`, forcing an unmandated fix or a hardcoded exemption.
FALSE. `grep -rl "backstop-core/pkg/packval"` reported `pkg/pack/engine`; the hits were
forbidden-list literals in `import_cycle_test.go` and `binding_test.go`, whose
`TestEngineBinding_NoImportCycle` (`go list -deps`) and `TestEngine_NoForbiddenImports` (AST)
assert packval is ABSENT — dead by construction, not unmeasured. Run through the guard's own
AST predicate, the packval-reaching set is exactly `cmd/backstop`, `pkg/packval`,
`pkg/pack/distribution`, and the generalized check flags ONE package — the one the plan already
fixes — with ZERO exemptions. Cost went from "forces scope creep" to "~3 lines", and the
decision flipped to generalize. Caught by the plan-reviewer, not by me.

★ ALSO: a precedent's OUTCOME is not its reasoning. PLAN-ISSUE-163 kept narrow, but its stated
rule was "do not widen the roster until something MEASURES that package reaching sandboxed
dispatch." When your lane IS that measurement, honouring the precedent means widening. Copying
the outcome while ignoring its condition is cargo-culting.

**Two points that survived the correction, both verified:**
- ★ A NARROW FIX USUALLY OPENS ITS OWN HOLE. A derived roster that SKIPS absent members picks
  the fixed package up for free — and drops it again silently if someone deletes the fix.
  Either close it by generalizing (better: replaces a hardcoded name with the predicate) or by
  adding the package to the required-member floor. Closing it is in scope; it is caused by the fix.
- ★ CHECK DIFF-SCOPE REACHABILITY BEFORE RELYING ON A CROSS-PACKAGE PIN. go-toolchain's
  `go-test` binding is `package_scoped: true`, so `fileModeTestTargets` narrows a diff-scoped
  run to the CHANGED FILES' packages. The realistic regression (deleting package X's guard) is
  by definition a diff in X — which never executes a pin living elsewhere. A package-local pin
  is what actually fires; accept the duplicated AST helpers and say why in the file header.
  The binding's adjacent `exempt_from_scope_filter: true` does NOT rescue this — it governs
  violation FILTERING, not target DERIVATION, and a test that never runs emits nothing to exempt.

**How to apply:** at any "should this plan also generalize?" fork — run the predicate through
the guard's own mechanism, paste the verdict and its count into the plan's notes, name the
packages it flags, and justify the call there rather than picking silently.
Related: [[project_extending_a_shipped_plan]], [[feedback_enumerations_assert_exhaustiveness]],
[[feedback_verify_issue_premises]].
