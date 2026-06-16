---
description: Create a TDD plan from a spec or issue, then review it
---
Create an implementation plan from: $ARGUMENTS

1. Confirm the source: a **spec** (feature track) or an **issue** (reactive track). If
   neither is identified, stop and ask.
2. Dispatch the **planner** agent to produce a TDD-compliant plan (task ordering, file
   scope, mandated test names, gate cadence). The agent writes the plan — do not hand-edit.
3. Run `./bin/backstop artifact validate`; hand any violations back to the planner until clean.
4. Dispatch the **plan-reviewer** agent to check the plan against its source for congruence
   (claim coverage, TDD ordering, file scope, gate cadence). Report its findings.

Report the validated plan + reviewer findings. Do not start implementation until the user says go.
