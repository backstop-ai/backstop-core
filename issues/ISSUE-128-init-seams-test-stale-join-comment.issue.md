---
title: "cmd/backstop/init_seams_test.go header comment claims file-level package reference satisfies the substantiveness subject join, which is false"
schema_version: issue/v1

issue:
  id: ISSUE-128
  title: "cmd/backstop/init_seams_test.go header comment claims file-level package reference satisfies the substantiveness subject join, which is false"
  type: bug
  status: open
  created: "2026-08-15"

complexity:
  scope: isolated
  uncertainty: known
  risk: safe
---

# init_seams_test.go header comment falsely claims file-level package reference satisfies the substantiveness subject join

## Problem

`cmd/backstop/init_seams_test.go` lines 25-27 carry this header comment:

```go
// Referencing `initialize` here is also what satisfies the substantiveness subject join
// for the five claims whose declared subject stays pkg/initialize while their mandated
// tests live in cmd/backstop, where the production adapters are forced to live.
```

This asserts that referencing the `initialize` package somewhere in the FILE is what satisfies
the `test_substantiveness` gate dimension's subject join for every test in that file. That is
false, and is the exact false belief that produced a real defect, just found and fixed while
closing out SPEC-069 (`specs/SPEC-069-backstop-init.spec.md`) to `implemented`.

Five claims — CLM-046, CLM-047, CLM-086, CLM-088, CLM-130 — had no explicit `subject:` field and
so inherited the spec-level default `pkg/initialize`, while their mandated tests live in this
file, which is `package main` (not `package initialize`). The join logic in
`pkg/gate/substantiveness_join.go` (`NoTargetViolation`) and `cmd/backstop/gate.go`
(`testFileColocatedWithTarget`) evaluates colocation/symbol-reference **per test function**, not
per file — there is no mechanism by which one import statement or one reference to `initialize`
anywhere in the file extends coverage to every other test function in that file. A file-level
"this file references package X" comment describes a check that does not exist in the join
implementation.

The defect on the SPEC-069 side is already resolved: a spec-author fix added explicit
`subject: cmd/backstop` to the five affected claims (shipped in SPEC-069 v1.3.3), which satisfies
the per-claim join correctly. But the comment in `init_seams_test.go` itself was never touched —
it is still there, still asserts the false per-file belief, and will mislead the next person who
reads it (author or reviewer) into thinking a claim can go without an explicit `subject:` as long
as *something* in the test file references the spec-level default package.

## Why this matters

This is the load-bearing false belief behind a real gate defect, sitting in a comment inside the
very file whose claims it mis-describes. Anyone reading this comment while adding a new claim or
test to `init_seams_test.go` (or copying its pattern into a similar cross-package wiring test
elsewhere) will reasonably conclude they can omit an explicit `subject:` field and rely on file
colocation — reproducing the same defect SPEC-069's close-out just paid to find and fix.

## Solution

Correct or remove the false claim in the lines 25-27 comment block. The comment can keep
explaining why the file exists and why it references `initialize` directly (that part is true and
useful — it documents the real-adapter wiring-proof rationale), but must not claim that doing so
satisfies the substantiveness subject join for claims declared elsewhere. If a replacement
sentence is wanted, it should say the opposite: each claim whose mandated tests live in this file
must declare an explicit `subject: cmd/backstop` (or equivalent) itself — file-level references
do not propagate to other test functions in the join.

This is a source-code comment correction, not a behavior change — no plan/spec lineage needed; fix
directly and note the resolution here.
