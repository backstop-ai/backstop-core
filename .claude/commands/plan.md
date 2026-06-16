---
description: Create a TDD plan from a spec or issue, then review it
---
Create an implementation plan from: $ARGUMENTS

1. Confirm the source: a **spec** (feature track) or an **issue** (reactive track). If
   neither is identified, stop and ask.
2. **Scaffold the plan file via the CLI FIRST** so it starts from a compliant template with
   a valid id (plans require a backing source):
   `./bin/backstop artifact new plan --slug <kebab-slug> --source <SPEC-NNN|ISSUE-NNN>`
   Several plans at once? Reserve all ids serially first to avoid id races. Note each path/id.
3. Dispatch the **planner** agent (one per plan; parallel is fine) to author INTO its
   pre-created file — TDD ordering, file scope, mandated test names, gate cadence. Tell it
   NOT to run `backstop artifact new`. The agent writes the plan — do not hand-edit.
4. Run `./bin/backstop artifact validate`; hand any violations back to the planner until clean.
5. Dispatch the **plan-reviewer** agent (one per plan) to check congruence against the source
   (claim coverage, TDD ordering, file scope, gate cadence). Report findings.

Report the validated plan(s) + reviewer findings. Do not start implementation until the user says go.
