---
name: absence-tests-via-goast
description: Write `kind: absence` claims as go/ast scans of the package's non-test sources, not regex — and expect several to pass on arrival because earlier phases already made the defect unreachable
metadata:
  type: project
---

For a spec claim of `kind: absence` ("no options struct declares a dependency
field", "nothing calls the production constructor internally", "no package-level
entry point survives"), parse the package with `go/parser.ParseDir` filtered to
non-test files and walk the AST. Reduce field types through `*ast.StarExpr` and
`*ast.SelectorExpr` so `*GitCloner` and `distribution.GitCloner` are not read as
unrelated types. Fail loudly when the parse yields no files, and assert every
expected declaration was FOUND — a renamed struct otherwise drops silently out of
coverage and the guard passes while covering nothing.

**Why:** these defects are invisible at runtime. A reintroduced `GitCloner` field
breaks nothing until someone leaves it nil, and the contracts pack's signature
compiler reduces any struct to `type X $$$` without ever comparing field lists, so
the declared contract does NOT enforce it — the source scan is the only thing that
catches it (SPEC-055 CLM-030).

**How to apply:** expect a mixed red/green on arrival. In SPEC-055 phase 11, only
3 of 12 new tests failed pre-refactor; the other 9 already passed because phases
7-10 had routed every caller through fail-closed constructors, making the nil-skip
branches unreachable from any test. That is not a hollow test — it is the ratchet
that stops the branch coming back. Report which ones were genuinely red rather
than claiming the whole set was. Relates to
[[project_selfpack_absence_claim_tests]].
