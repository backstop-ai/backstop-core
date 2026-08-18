---
name: doctored-fixture-must-still-parse
description: A test that doctors artifact/manifest DATA to reach a refusal branch must prove the doctored input still parses, and assert the branch's OWN message — "an error was returned" passes on an upstream parse failure
metadata:
  type: project
---

When a plan mandates a test that **doctors artifact or manifest data** to drive a
fail-loud branch, two things must be specified or the test is a vacuous green:

1. **Prove the doctored input still PARSES.** Loaders validate more than you remember.
   Concretely: `pack.ParseManifestFile` runs `validateFixtures`, which hard-requires
   ≥1 positive AND ≥1 negative fixture per rule claim — so "remove the `fixtures:`
   block" makes the *parse* fail, not the branch under test. Removing the whole
   `claims:` block parses clean (there is no rule-level claims-non-empty check; the
   `len(Claims) == 0` requirement in that file is for `tool_configs`) and reaches the
   branch.
2. **Assert the branch's OWN distinguishing message**, not "a non-nil error naming the
   rule id and the path" — a parse error satisfies that too. Mandate distinct,
   unique-to-the-branch phrasing in the implementation task, and have the test pin that
   literal (stated once, per [[feedback_state_a_sweep_once]]).

**Why:** in PLAN-ISSUE-158 review, the first recipe produced a subtest that went GREEN
while exercising an entirely different code path than its name claimed. It was caught by
a reviewer running the doctoring for real, not by the loop.

**How to apply:** whenever a task says "strip X from the fixture/manifest so the
derivation is empty", make the plan state (a) the measured proof that the stripped input
parses, and (b) the exact substring the assertion targets. Cheap to specify, invisible
once wrong.
