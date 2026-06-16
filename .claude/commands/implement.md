---
description: Execute a plan (implementer; gate-on-implement hook enforces green), then review
---
Execute the plan: $ARGUMENTS

1. Dispatch the **implementer** agent to execute the plan, following task ordering and
   file scope. The implementer is capability-bounded — do not hand-edit code as the
   orchestrator; let the agent do the work.
2. The `gate-on-implement` SubagentStop hook blocks the implementer from finishing while
   `backstop gate` is red — let that refactor-until-green loop run. Only step in if it gets
   stuck on something a refactor can't resolve.
3. Once the implementer returns and `./bin/backstop gate` is green, dispatch the
   **impl-reviewer** agent to verify the implementation against the spec claims and plan
   tasks (correctness, test substantiveness, contract fulfillment). Report its findings.

Report the gate result + reviewer findings.
