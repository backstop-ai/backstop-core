---
name: import-premise-grep-matches-forbidden-lists
description: "\"package X imports pkg/Y\" premises derived from `grep -rl <import path>` are routinely FALSE — the hit is a string literal in a leaf package's own forbidden-import test asserting the opposite; re-run the guard's AST predicate"
metadata:
  type: project
---

When a plan justifies a scope decision with "package X imports pkg/Y" (or
publishes an importer roster), do NOT accept a `grep -rl "<import path>"`
derivation. Re-run the guard's OWN predicate — `parser.ParseFile` + walk
`file.Imports` comparing the quoted path — over the same walk exclusions the
guard uses (`testdata`, `vendor`, hidden dirs).

**Why:** leaf packages in this repo pin their leaf-ness with tests that embed the
FORBIDDEN import paths as string literals. `pkg/pack/engine` has two
(`import_cycle_test.go`'s `TestEngineBinding_NoImportCycle` via `go list -deps`,
and `binding_test.go`'s `TestEngine_NoForbiddenImports`), both listing
`github.com/backstop-ai/backstop-core/pkg/packval` in a `forbidden := []string{...}`
array, plus a doc comment in `binding.go`. A text grep reads all three as
"engine imports packval"; the AST says engine imports it nowhere and CANNOT
(REQ-013/CLM-033 pin the leaf invariant).

**Measured instance (PLAN-ISSUE-180 review, 2026-08-19).** The plan's heaviest
recorded reason for staying narrow — "generalizing the roster predicate goes RED
immediately for `pkg/pack/engine`, forcing an unmandated fix or a hardcoded
exemption" — was false. Running the guard's predicate gave real packval
importers = {`cmd/backstop`, `pkg/pack/distribution`, `pkg/packval`}; the
generalized predicate would have flagged exactly the ONE package the plan was
already fixing, with zero exemptions. The same bad grep is baked into ISSUE-164
and ISSUE-180 prose, and the planner had already saved it to memory as a
"measured" lesson.

**How to apply:** any plan whose scope fence, judgment call, or routed issue
disposition rests on an import edge — write the ~40-line AST walker and run it
before reviewing the decision. Also check whether the false premise is being
propagated into a routed `/issue` edit (that writes it into a committed
artifact) or into another agent's memory dir. Related:
[[project_census_through_real_parser]], [[project_verified_enumeration_do_not_rederive]],
[[project_sibling_precedent_cited_not_read]].
