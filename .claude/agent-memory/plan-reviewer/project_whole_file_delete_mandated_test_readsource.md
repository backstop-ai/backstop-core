---
name: whole-file-delete-mandated-test-readsource
description: Deleting a file can break a SURVIVING-spec mandated test that reads that file as source (readCheckSource) at RUNTIME — deleted-test claim-drift via t.Fatalf, not compile error
metadata:
  type: project
---

Source-introspection tests (`readCheckSource(t, "semgrep.go")`, os.ReadFile of a
prod file, then string-Contains assertions) BREAK AT RUNTIME when the file they read
is deleted — `t.Fatalf("reading ...: %v")` on missing file. This is invisible to
`go build` and to the validator; it only fails when that test actually runs.

**Why:** PLAN-ISSUE-018 deletes pkg/check/semgrep.go. SPEC-034 CLM-029 mandates
`TestProvision_EnsureSemgrepBespokeInstallRetired`, which calls
`readCheckSource(t, "semgrep.go")` to assert the pip-install ladder is gone. Deleting
the file makes the surviving SPEC-034 claim's test t.Fatalf. The deletion plan must
either repoint/retire CLM-029's test or relocate the assertion — classic ISSUE-014
deleted-test claim drift, made sneakier because it's a runtime fatal, not a dangling
reference. test_verification is diff-scoped, so it surfaces exactly when semgrep.go
re-enters scope (i.e. this very plan), so the plan is the place to catch it.

**How to apply:** for any deleted prod file, grep mandated-test source for
`readCheckSource(... "<basename>")` / `os.ReadFile(.../<file>)`. If a SURVIVING spec
claim (grep specs/ for the test name) maps to such a test, the plan needs a task to
repoint or retire that claim. Sibling: [[whole-file-delete-shared-types]].
