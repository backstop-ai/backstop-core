---
name: phase3-fixture-polarity-inverted
description: Plans citing packval phase-3 "silent pass" impact routinely invert which fixture polarity is silent — POSITIVE is the silent one on the findings seam, NEGATIVE goes loud
metadata:
  type: project
---

On the FINDINGS seam in `pkg/packval/phase3.go` (`RunFixtures`), `RunEngine`'s
`Passed` means "the engine FIRED", so the polarity is the INVERSE of the naive
reading:

- POSITIVE fixture: error raised only `case r.Passed:` (`semgrep-positive`,
  "positive fixture triggered the rule (false positive)"). So
  `Passed=false, err=nil` is a POSITIVE fixture's SUCCESS condition — the silent
  clean pass.
- NEGATIVE fixture: error raised `else if !r.Passed` (`semgrep-negative`,
  "negative fixture not triggered"). So `Passed=false, err=nil` REDS a negative
  fixture — it is loud, not silent.

The VALIDATOR seam is deliberately the opposite shape (`RunValidator.Passed`
means "exited zero"), which is what seeds the confusion. `phase3.go` carries a
"DO NOT HARMONIZE" comment on that asymmetry.

**Why:** PLAN-ISSUE-160 (2026-08-17) stated in five places that
`Passed=false` + nil error "IS the success condition for a NEGATIVE phase-3
fixture" — inverted — including in two implementer-facing task descriptions and
one claim body. DIR-032 item 21 had already spelled out the correct polarity and
explicitly warned "a planner must not misread it"; the plan restated the exact
misreading the directive named.

**How to apply:** any plan whose impact/homing argument rests on packval
phase-3 verdict honesty — the ISSUE-092/140/141/144/160 drift family — gets its
polarity sentences read literally against `phase3.go`'s two `for _, f := range
claim.Fixtures.{Positive,Negative}` loops. Grep the plan for "negative fixture",
"clean negative" and "positive fixture" and evaluate each against the branch
that actually raises the error. Citing the source directive is not evidence the
directive was read correctly.

**THE INVERSION IS ALSO IN COMMITTED SOURCE, WHICH IS WHY IT KEEPS RECURRING.**
`pkg/packval/executor.go`'s stdout_artifact comment (delivered by PLAN-ISSUE-144,
locate by `if binding.StdoutArtifact != ""`) reads "a Passed=false/nil-error
verdict that IS the success condition for a negative fixture" — inverted the same
way. A planner mirroring the neighbouring lane's prose inherits the error. Do not
treat "it matches the existing comment" as evidence; evaluate against
`phase3.go`'s loops only.

Related: [[registry_derived_premise_per_test]] (per-site, not per-file, premise
checking), [[verified_enumeration_do_not_rederive]].
