---
name: issue116-line-carry-pass
description: PLAN-ISSUE-116 (substantiveness waiver line carry) reviewed PASS — revert-experiment methodology confirmed all three tests are true falsifiers, incl. the behavioral assertion behind a Fatalf
metadata:
  type: project
---

PLAN-ISSUE-116 (`Line: v.Line` in `HollowFindingsToViolations`) passed review 2026-08-10.
One-line production fix + two docstring corrections + 3 tests; scope fence vs ISSUE-106
(hardcoded Severity) and ISSUE-117 (noTarget has no line) held — noTarget dodged in the
FIXTURE (`gate.Build()` satisfies the set-join) rather than via `kind: absence` or a code
change.

**Why:** this lane arrived with an unusually confident implementer report claiming a
from-scratch red-proof. Rather than trusting the narrative, the review reproduced it: rsync
the working tree (uncommitted changes included) to scratchpad, delete ONLY the `Line: v.Line`
line, rebuild, rerun. All three tests went red with the exact predicted values.

**How to apply:** two techniques worth reusing on any "did they actually prove the red" review.
1. The rsync-to-scratchpad revert is the way to red-prove UNCOMMITTED work read-only —
   `git stash` would mutate the tree and violate the reviewer's read-only contract.
2. When a test's first `t.Fatalf` is a MECHANISM assertion (here `v.Line == 0`), it MASKS
   whether the downstream BEHAVIORAL assertions are independently red. Neuter the mechanism
   assert in the scratchpad copy and rerun — that is what proves the user-visible claim
   (suppression actually happens) rather than just the field-plumbing claim. Here assertion 3
   was independently red pre-fix, so CLM-002 is genuinely earned.

Non-blocking note carried forward: the unit test pins `Severity == "error"` (plan-mandated,
CLM-005), so ISSUE-106's lane must update that assertion when it lands. See
[[project_spec048_dispatch_fix_pass]] for the sibling "real-path-or-it-didn't-happen" pattern.
