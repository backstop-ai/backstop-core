---
name: severity-defect-has-two-hops
description: An issue naming one severity-discarding converter usually hides a second hop — the consuming step's raw-count status; fixing only the converter leaves the defect user-invisible
metadata:
  type: project
---

When an issue says "site X discards the pack-declared severity", the fix is almost never
one site. Check the CONSUMING step's status computation too: a step that still computes
`status := "pass"; if len(violations) > 0 { status = "fail" }` throws the preserved
severity away one line later, so the converter fix is real but produces no user-visible
change.

**Why:** ISSUE-106 (2026-08-16) named only `HollowFindingsToViolations` hardcoding
`Severity: "error"`. `buildTestSubstantivenessStep` (cmd/backstop/gate.go) still used a
raw count — the exact ISSUE-105 defect shape at a step ISSUE-105 did not convert. Measured
in a detached worktree: `StepVerdict` over a warning violation returns `"warning"` and
`NewGateResult` keeps `Pass=true`, while the raw count returns `"fail"` and `Pass=false`.
Hop A alone would have shipped an invisible fix.

**How to apply:** For any pack-severity / verdict-honesty issue, grep the consumers of the
converter for `StepVerdict` vs `len(violations)`. Note that `ApplyPolicy`'s docstring
already CLAIMS "the step builders now reach their verdict through StepVerdict" — that is an
overclaim written during ISSUE-105 with stragglers left behind, so it is a hypothesis to
check, not a fact to trust. Also expect a shipped test that LOCKS the defect ("Filed, not
fixed here") — see [[project_defect_pinned_by_shipped_tests]] — and expect the issue's
"either resolution is fine" half to need an explicit, written-into-the-docstring decision
rather than a silent choice.

Related: [[feedback_verify_issue_premises]], [[feedback_cite_by_name_in_contended_files]].
