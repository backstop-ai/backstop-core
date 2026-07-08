---
name: repurposed-test-claim-text-drift
description: Repurposing a mandated test in place (name kept, body flipped) satisfies the gate's existence check but silently strands the mandating claim's TEXT
metadata:
  type: project
---

When a plan CHANGES the behavior a function under an existing live spec asserts, and
"repurposes the mandated test in place" (keeps the test function NAME, rewrites the
body to assert the NEW behavior), the gate's mandated-test step stays GREEN — it only
checks the test NAME still exists, never that the body matches the claim text.

**Why:** the surviving claim's TEXT can still assert the OLD (now-removed) behavior.
Net: a live claim whose prose is false and whose mandated test now asserts the
contradiction, invisible to `test_verification` (existence-by-name only). This is the
text-vs-test variant of ISSUE-014 deleted-test-claim-drift.

**How to apply:** for any plan task that changes behavior AND repurposes a named
mandated test of a LIVE spec, grep the mandating claim's TEXT (and its REQ) for the
specific old behavior. If the claim prose still asserts it, the plan MUST include a
spec-author task to update that claim/REQ + bump spec_version — per
[[feedback_align_predating_artifacts]]. Concrete instance: ISSUE-047 flipped
pkg/gate.TargetPackageName (cmd/→"" removed) and repurposed
TestTargetPackageName_MigratedBehaviorPreserved, but SPEC-037 CLM-028/REQ-008 text
still said "cmd/... and non-pkg/ paths yield """ — no reconciliation task. Related:
[[project_coverage_rewrite_predating_spec_drift]], [[project_deletion_strands_spec_lineage]].
