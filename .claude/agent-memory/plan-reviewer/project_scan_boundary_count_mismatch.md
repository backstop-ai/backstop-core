---
name: scan-boundary-count-mismatch
description: A plan's measured "N files under <dir>" label often excludes testdata while the prescribed walk includes it — re-derive BOTH counts, because the boundary IS the claim
metadata:
  type: project
---

When a plan introduces a new directory-walking scan ("enumerate every `.go` file under
`pkg/X`") and cites a measured file count, re-derive the count BOTH ways before trusting
the label:

```
find pkg/X -name '*.go' | wc -l
find pkg/X -name '*.go' -not -path '*testdata*' | wc -l
```

PLAN-ISSUE-139 (2026-08-16) said "all 108 `.go` files under `pkg/gate`" and told the
implementer a recursive walk "yields 108". Reality: 133 recursive, 108 excluding
`pkg/gate/testdata/`. The planner's grep WAS recursive (so the zero-match finding held),
but the count label came from a different command.

**Why:** these plans routinely carry a sharp edge saying "THE SCAN BOUNDARY IS THE CLAIM"
and require an emptiness guard. A count that doesn't match the prescribed walk leaves the
implementer to reconcile — and the cheap reconciliation is to ADD a testdata exclusion,
silently narrowing the claim nobody reviewed. Related: [[new-guard-predicate-measure-existing-fixtures]].

**How to apply:** whenever a plan names a file count for a new scan, run both finds. If they
differ, demand the plan state testdata IN or OUT explicitly. There is usually a substantive
answer: `pkg/gate/testdata` holds synthetic fixture projects with invented spec ids
(SPEC-900..906, SPEC-999 all present), so scanning it adds false-positive surface for
prose/spec-id predicates while adding no real signal — fixtures are not the machinery the
claim is about.
