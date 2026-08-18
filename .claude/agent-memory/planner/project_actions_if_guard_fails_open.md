---
name: actions-if-guard-fails-open
description: A GitHub Actions `if:` referencing a nonexistent step id fails OPEN (step always runs), not closed — the coercion path that makes this true, and why it inverts a guard's stated rationale
metadata:
  type: project
---

`if: always() && steps.<id>.conclusion != 'skipped'` with a WRONG OR MISSING `<id>` does not
skip the step. It makes the guard **inert** — the step runs on every run, including exactly the
runs the guard existed to suppress.

The coercion path, which is the part that gets stated backwards:
a missing context property resolves to `null` → comparing `null != 'skipped'` is a type
mismatch → GitHub coerces both sides **to number** (`null` → 0, `'skipped'` → NaN, since it
does not parse) → every comparison involving NaN is false → `null == 'skipped'` is FALSE →
`!=` is TRUE → the condition collapses to `always() && true`.

**Why:** PLAN-ISSUE-176 review round 3 (2026-08-18). The plan added a confirmation step guarded
on the gate step's conclusion, and asserted the guard's id against the step's declared id — a
correct assertion — but justified it with "a bad id silently evaluates to a permanent skip."
That is the opposite of what happens, and it mattered because this repo's convention is to
encode the rationale into the assertion's failure message: the message would have told a future
reader a typo fails SAFE when it fails OPEN, which is the more dangerous direction and the whole
reason the assertion is there.

**How to apply:** when planning any workflow `if:` guard, state the failure mode as fails-OPEN
and assert the id-consistency (extract the id the `if:` references; require the target step to
declare that exact value — do not match both against a hardcoded literal, which passes a pair
that agrees only with the test). Related, and a better argument than a bare "skipped means
skipped": in backstop-core, `TestCIWorkflow_PackInstallFailureFailsTheJob` forbids ANY `if:` on
a step whose script contains `backstop gate`, so condition-false is unreachable for the gate
step and `conclusion == 'skipped'` can only mean an earlier step killed the job — which is
precisely the discriminator worth guarding on. See
[[project_run_the_command_you_prescribe]] for the general "verify the mechanism, don't reason
about it" habit this belongs to.
