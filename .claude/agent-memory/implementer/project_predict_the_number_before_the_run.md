---
name: predict-the-number-before-the-run
description: Commit to a predicted measurement BEFORE the CI run — it turns an expensive runner into an experiment; ISSUE-020's predictions went 81.6% (exact), 92.3% (held) and one wrong 3-for-1 the profile falsified in a single command
metadata:
  type: project
---

Practice that paid repeatedly in ISSUE-020, where each CI iteration cost minutes and
a push:

  "split alone -> 40/49 = 81.6%, NOT the floor"      -> measured 43/50 = 86.0%, short
  "seam -> 46/50 = 92%"                              -> measured 44/50 = 88.0%, WRONG
  "delegation -> 48/52 = 92.3%"                      -> HELD, floor cleared

**Why:** the wrong one was the most valuable. Predicting +3 forced the reasoning into
the open ("covering the refusal also drives its callers' wraps"), so when the profile
showed +1 the error was locatable in one command — the wraps only fire when the failure
happens INSIDE the callers, and the callers pass the real prober. An unpredicted number
would have been a surprise to explain rather than a hypothesis to check.

**How to apply:** before any expensive verification, state the expected number and what
would falsify it. Say plainly when the prediction does NOT reach the bar — "I do not
predict 90%" preserved everyone's ability to plan. Never report a range you have not
computed, and re-derive from the artifact rather than from the last report.
Related: [[project_buildtag_file_never_measurable]],
[[project_ast_locks_untestable_wiring]].
