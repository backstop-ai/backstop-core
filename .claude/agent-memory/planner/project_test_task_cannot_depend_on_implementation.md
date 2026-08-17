---
name: test-task-cannot-depend-on-implementation
description: A test task needing a symbol from an implementation task must depend on the VERIFICATION task downstream of it — test→implementation is rejected by validateTestTaskDeps
metadata:
  type: project
---

When a `test` task needs a symbol that a `implementation` task creates (so it would
otherwise fail to compile), route the edge through the VERIFICATION task that depends on
that implementation — never add the implementation task to the test's `depends_on`.

**Why:** `validateTestTaskDeps` in `pkg/validate/plan.go` (REQ-010) allows a test task to
depend only on `setup`, `test`, or `verification`. A test→implementation edge is rejected
as `plan/test-invalid-dependency`. Since every implementation task is followed by a
verification task under the gate-cadence rule, that verification task is always available
as a legal proxy and delivers the same ordering guarantee.

**How to apply:** Reviewers routinely ask for the illegal edge directly ("TASK-008 needs
the predicate from TASK-003, add TASK-003 to its depends_on"). Apply the INTENT via the
downstream verification task and say in the reply why the literal instruction could not be
followed — the reviewer is usually reasoning about compile order, not about the validator.
The same substitution answers the mirror-image suggestion "point this test at the
implementation instead of the verification to save a serialization hop": that hop is not
optional.

Related: [[project_extending_a_shipped_plan]] — the other half of the same rule, that a
test task IS permitted to depend on a verification task, which is what makes reconciliation
phases legal at all.
