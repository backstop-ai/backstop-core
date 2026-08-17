---
name: registry-derived-premise-per-test
description: A plan's "that test absorbs the new entry because it derives from the registry" premise must be checked PER TEST, never per file — sibling tests in the same file often hardcode the enumeration
metadata:
  type: project
---

When a plan grows a registry/enumeration (an Nth check, an Nth kind, an Nth
step) and its notes list tests as "CONFIRMED UNAFFECTED because they derive
their expected set from the live registry," open EACH named test and grep the
file for the enumerating call (`doctorRegistry()`, `Steps()`, etc.). A file can
mix both shapes: `cmd/backstop/doctor_registry_test.go` has two tests that
`range doctorRegistry()` AND `TestDoctor_CheckIDsAppearOnlyAsDeclaredConstants`
whose `ids := []string{…}` is HARDCODED — both its `wanted` filter and its final
assertion loop iterate the hardcoded slice, so a new id is silently outside the
scan's coverage while the test stays green over the old set.

**Why:** caught twice on PLAN-ISSUE-134 (2026-08-16). Round 2 found the false
"unaffected" bullet; the planner's own fix then revealed the SAME false premise
restated verbatim in an implementation task's body. A hardcoded-set scan that
quietly stops covering the thing it was pointed at is the exact defect class
these directives exist to prevent — and it lands on an `implemented` spec's
mandated test, so nothing in the gate reds.

**How to apply:** for every "unaffected" bullet, (1) read the named test's
actual enumeration source, (2) if hardcoded, it is a MUST-DISTURB item needing
an explicit edit task with the file in `files:` and the test in `test_names:`,
and (3) sweep the WHOLE plan for the same premise restated elsewhere — planners
repeat their notes' reasoning inside task descriptions, so fixing the notes
alone leaves the false claim live where the implementer actually reads it.
Related: [[verified-enumeration-do-not-rederive]],
[[completeness-claimed-comment-set]].
