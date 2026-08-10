---
name: information-existed-surfacing-did-not
description: Three defects in one lane shared a shape — the diagnostic was in hand and thrown away (helper stderr to /dev/null, policy ignoring Severity, baseline discarding result.Steps); when a failure reports only a code, suspect the wrapper before the cause
metadata:
  type: project
---

ISSUE-020 produced the same defect three times, each costing a CI iteration:

1. `platformSandboxedRunStdout` set Stdin and Stdout and left **Stderr nil** — os/exec
   routes that to /dev/null — so the helper's CLM-015 diagnostic naming Landlock, the
   kernel and ISSUE-020 was destroyed. Run 30381252600 reported only "exit status 126".
2. `pkg/gate/policy.go` never read `Violation.Severity` (see
   [[project_verdict_decided_after_the_step]]).
3. `cmd/backstop/baseline.go` held `result` and discarded it on the exit-2 branch,
   justified by a comment asserting `result.Steps` is always empty — TRUE only when the
   gate fails before building steps, FALSE when a step raises ConfigErr (provisionEngines
   refusing a missing Layer-0 tool). Run 30398137055 said "exit 2" and nothing else.

**Why:** in every case the fix was small and the COST was a full runner iteration spent
learning nothing. A failure that reports only a code is not a failure you have
diagnosed; it is one you have to reproduce.

**How to apply:** when a wrapper turns a rich result into a short message, check what it
is holding before you re-run anything. Grep the branch for the variable it ignores. And
distrust a comment that explains why evidence is not worth printing — comment 3 was
written true for one class and applied to all. Fix the SURFACING first, then re-run:
shipping the cause-fix alone leaves the next failure equally mute.
