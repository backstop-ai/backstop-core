---
name: prior-lane-planted-fence
description: Grep the Go/test corpus for the plan's OWN issue ID — a completed sibling lane often planted an assertion fencing exactly this change, and it is invisible to the plan's symbol-grep
metadata:
  type: project
---

Before accepting any plan's "I grepped for `<Symbol>` and the only other touchpoint is X"
premise, run TWO greps yourself:

    grep -rn "<Symbol>" --include='*_test.go' .
    grep -rn "ISSUE-NNN" --include='*.go' pkg cmd      # the plan's OWN issue id

The second one is the cheap, high-yield detector. A completed sibling lane that touched the
same function frequently plants a guard naming the FUTURE issue by ID — and the planner's
symbol-grep either misses the file or misreads the assertion as severity-agnostic.

**Why:** PLAN-ISSUE-106 (2026-08-17) claimed the only other test touching
`HollowFindingsToViolations` was severity-agnostic. False:
`pkg/gate/substantiveness_hollow_violation_test.go:98` asserts `v.Severity == "error"` on
three inputs that carry NO severity — so forwarding `v.Severity` yields `""` and reds it.
PLAN-ISSUE-116 had planted it ON PURPOSE, comment at line 58 reading "this fix must NOT
become ISSUE-106's severity change". A single `grep -rn ISSUE-106 --include='*.go'` found it.

**How to apply:** run both greps, then OPEN each hit and read what the fixture INPUTS carry,
not just what the assertion says — an assertion that passes today only because the production
code hardcodes the value is a lock, even when the test's name is about something else
(Line carry, path normalization). A planted fence must be retired DELIBERATELY by the lane it
fences (scoped file + explicit task), never tripped at a verification step. Keep the test NAME
if a prior plan lists it in `test_names`. See [[project_retired_feeder_test_collateral]] and
[[project_repurposed_test_claim_text_drift]].
