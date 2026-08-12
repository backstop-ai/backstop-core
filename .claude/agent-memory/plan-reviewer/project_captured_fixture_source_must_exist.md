---
name: captured-fixture-source-must-exist
description: "Captured-never-authored fixture tasks must name a source that EXISTS on disk — verify it; 'a real X that never mentions Y' is often both unobtainable and vacuous"
metadata:
  type: project
---

For any plan task mandating CAPTURED (not authored) fixtures, check each named source
actually exists and actually has the property claimed. Two failure shapes, both found
in PLAN-SPEC-067 phase 4:

1. **Source contradicts criterion.** "a byte copy of a REAL workflow that never
   mentions `backstop gate` (backstop-core's own .github/workflows/ci.yml is the
   honest source)" — ci.yml invokes `backstop gate` three times.
2. **No source exists.** The same task family demanded "a real pipeline config" for
   GitLab / Bitbucket / Jenkins; no `.gitlab-ci.yml`, `bitbucket-pipelines.yml` or
   `Jenkinsfile` exists anywhere under `~/src/projects/`.

**Why:** the captured-never-authored rule exists because a fixture written to match
the rule proves only that the rule equals itself. An unobtainable source forces the
implementer to fabricate one silently — the exact defect the rule prevents. And a
"silence" fixture chosen to lack the rules' own anchor token is VACUOUSLY silent:
every rule anchored on that token is trivially quiet, proving nothing.

**How to apply:** `ls` the named source; grep it for the asserted property. If the
spec only requires one positive fixture per rule (the clean rendered payload already
supplies it), an unobtainable extra "silence" fixture should be dropped or given a
real named source, not left to the implementer's improvisation. Pairs with
[[fixtures-from-real-output]] discipline.
