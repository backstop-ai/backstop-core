---
name: gated-subcheck-unreachable-pin
description: A plan pinning an edge-case input "through the exported validator" is vacuous when the target sub-check sits behind an earlier pass/fail guard — grep the call site, not just the function
metadata:
  type: project
---

When a plan says "pin behavior X by driving the EXPORTED entry point with input I"
(and forbids calling the unexported helper), verify that I actually REACHES the code
implementing X. Composite validators in `pkg/validate` gate later sub-checks on
earlier results.

Canonical case: `pkg/validate/bundle.go:84` calls `validateNameFilenameConsistency`
only `if filenameOK` — i.e. only when the filename already matched the schema's
`filename_pattern` (`^[a-z0-9-]+(\.epic)?\.bundle\.md$`). So through `validate.Bundle`:

- `"foo.epic"` → pattern fails → stem code NEVER RUNS (the if/else fall-through
  branch it was meant to pin is DEAD in production)
- `"BUNDLE-007-baseline.bundle.md"` → uppercase rejected by `[a-z0-9-]+` → also never
  runs, so the BUNDLE-NNN- prefix strip is unexercised through the exported path
- with `sch.FilenamePattern == ""`, `filenameOK` stays false too — there is NO input
  that reaches the sub-check except a real pattern match

**Why:** PLAN-ISSUE-124 round 2 built its whole bundle falsification story on
`"foo.epic"` — a sharp edge, a table row, and a dedicated mutation (2b) it called
"the mutation that matters most in this lane." All three were unreachable, so the
mutation could never go red and the row was green-because-unexecuted.

**How to apply:** for every "drive the exported X with input I" instruction in a plan,
`grep -n "<helperName>"` to find the single call site and read its guard. If the guard
filters I out, the pin is vacuous — say so and offer the fix: an INTERNAL test package
(`package validate`, as `resolved_by_layout_test.go` already is) may call the
unexported helper directly, which is not "exporting" it; or pass a deliberately
permissive `*schema.Schema`. Related: [[project_shortcircuit_dependent_guard]],
[[project_inert_decoy_fixtures_vacuous]].

**RESOLVED FORM (PLAN-ISSUE-124 round 4, accepted).** The direct in-package call won
over the permissive-schema option, and the reasoning generalizes: a fabricated
permissive schema pins the logic against an input shape production can never produce
AND stops tracking the real schema if it changes — a second fiction layered on the one
being removed. The accepted shape also (a) restates the sharp edge honestly ("harmless
only because an upstream schema pattern filters out the distinguishing input — a
coupling nothing states or enforces", NOT "a real verdict change"), and (b) adds a
mutation guard: if the observed red is `bundle/filename-pattern` rather than the row's
own assertion, the TEST is wrong (written against the exported path), not evidence of
a defect. Demand both halves, not just the direct call.
