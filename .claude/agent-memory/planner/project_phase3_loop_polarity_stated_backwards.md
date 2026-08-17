---
name: phase3-loop-polarity-stated-backwards
description: In packval phase3, Passed=false + nil err is a POSITIVE fixture's SILENT pass and a NEGATIVE fixture's LOUD red — plans keep writing it inverted and reviewers BLOCK on it
metadata:
  type: project
---

`pkg/packval/phase3.go`'s two fixture loops are deliberately opposite shapes: the positive
loop appends an error only on `case r.Passed:` (a finding on the clean example is a false
positive); the negative loop appends on `else if !r.Passed`. So the verdict
`Passed=false, err=nil` — what every `RunEngine` divergence defect produces — is a POSITIVE
fixture's **silent** success and a NEGATIVE fixture's **loud** red (misattributed to the
fixture). The silent half is the one that makes these DIR-032 verdict-honesty defects.

**Why:** PLAN-ISSUE-160 stated it backwards in five places and was FAILed for it — the
"success condition for a negative fixture" phrasing reads plausibly and propagates by
copy-paste into claim bodies and task descriptions. DIR-032's own item 21 warns against
exactly this misreading, which is itself evidence of how easy it is to make.

**How to apply:** any time a plan explains what a lying `RunEngine` verdict means for phase
3, re-derive the sentence from a fresh read of the two loops, never by restating prior plan
prose (yours or a sibling's). Grep the finished plan for `negative` before handing off. Note
that tests calling `RunEngine` directly do not exercise either loop — only the *prose*
framing is at risk, so a polarity fix is edit-the-notes, not re-author-the-tasks. Related:
[[project_packval_verdict_is_whole_ruleset]], [[feedback_verify_issue_premises]].
