---
name: cited-fixture-precedent-reads-other-file
description: A plan citing an existing testdata pack shape as precedent goes vacuous when the NEW check reads a different file class than the precedent check did — verify what the cited check actually reads
metadata:
  type: project
---

When a plan says "follow the manifest-only shape already used by `testdata/<X>/`", open the
check that consumes `<X>` and confirm it reads THE SAME FILES the new check will. A precedent
corpus is minimal for the precedent's reader, not for yours.

**Why:** PLAN-ISSUE-151 TASK-001 (2026-08-17) cited `pkg/packval/testdata/exempt-decision/`
as the shape to copy. That corpus is `pack.yml`-only because `ExemptDecisionPending` reads
only `pack.yml`. The new check reads each rule's `paths:` block out of
`packDir/<rule_path>` — a rule FILE — and needs declared fixture FILES for its mask claim.
The task's `files:` list carried seven `pack.yml` paths and nothing else, so the whole
advisory corpus would have yielded zero patterns and every phase-2 test would have gone
vacuously green while passing.

**How to apply:** for any fixture-corpus setup task, derive the required file set from the
check's own reads, then diff against the task's declared `files:`. Also check what the
SURROUNDING validator errors on for the fixture shape — here `phase2.go` emits
`fixtures-positive` / `fixtures-negative` / `fixture-exists` / `fixture-empty` errors, which
would have RED-ed the plan's own "advisory never blocks" claim (it asserts `Errors` is empty)
for the two packs that needed declared fixtures. Related:
[[project_inert_decoy_fixtures_vacuous]], [[project_captured_fixture_source_must_exist]].
