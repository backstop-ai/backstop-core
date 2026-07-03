---
name: symmetric-field-removal-claims
description: Flag-day removal of two parallel fields (manifestDir + Options.ManifestDir) needs symmetric claim attribution; check both "no-test-requires-X" self-check claims map to the same task shape
metadata:
  type: project
---

When a spec removes two parallel struct fields under one flag-day (SPEC-030: `semgrepExecutor.manifestDir` + `check.Options.ManifestDir`), it usually has two symmetric "no remaining pkg/check test constructs X" self-check claims (CLM-023 for the executor field, CLM-024 for the Options field). Verify BOTH map to the same task shape: test-task (deletes the literals + adds the self-check) AND impl-task (removes the field, which is the production truth-maker).

**Why:** Re-review of PLAN-SPEC-030 (2026-06-16). The plan mapped CLM-024 to TASK-002 (test) AND TASK-003 (impl), but its twin CLM-023 only to TASK-001 (test) + TASK-014 (final) — NOT to TASK-003, even though TASK-003 removes the `manifestDir` field that makes CLM-023's self-check pass. Not a blocker (validator passes; both files scoped — `semgrep_executor_test.go` in TASK-001, `check.go` in TASK-003; no work/scope gap), but an attribution inconsistency that's easy to miss because the validator only requires >=1 task per claim.

**How to apply:** For paired field-removal claims, line up the claim->task map for both twins side by side; if one carries the impl task and the other doesn't, flag the asymmetry as minor. Also verify the field-removal collateral surface empirically: `grep -ln "manifestDir:\|ManifestDir:" pkg/.../*_test.go` should return EXACTLY the files the plan scopes (here only 2: semgrep_executor_test.go + check_test.go); other tests using `defaultManifest()` / `LoadManifest(tempdir)` directly are NOT collateral. Related to [[field-removal-fixture-scope]] and [[retired-feeder-test-collateral]].
