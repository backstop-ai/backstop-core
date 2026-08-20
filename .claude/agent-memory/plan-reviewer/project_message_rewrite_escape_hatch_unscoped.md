---
name: message-rewrite-escape-hatch-unscoped
description: A plan rewriting an error/warning string often hands the impl task a "this task owns the consequence if you can't preserve the substring" escape hatch its files[] forbids — grep the literal for pre-existing assertions and check whose scope can fix them
metadata:
  type: project
---

When a plan task rewrites a production **error or warning string**, grep the OLD
literal across `*_test.go` and check whether the task whose `files:` could repair
a broken assertion is the SAME task doing the rewrite. Plans frequently write an
escape hatch — "preserve the substring if you can; this task owns the consequence
if you cannot" — while the task's `files:` lists only the production file. The
test file is usually owned by the earlier RED task, which has already run.

**Why:** PLAN-ISSUE-178 (2026-08-20) rewrote `resolveLatestSuccessfulMainRun`'s
terminal miss message for CLM-003. Its prescribed replacement dropped the
substring `no latest successful main run` that `cmd/backstop/baseline_test.go:331`
asserts on via the `runs-empty` fixture (empty array → loop never runs → terminal
return fires). TASK-002's `files:` was `cmd/backstop/baseline.go` only; TASK-001
(which scoped `baseline_test.go`) ran first; TASK-003/004 were verification. The
breakage would surface at verification with no authorized remedy — inviting an
out-of-scope edit or a weakened assertion, both of which the repo forbids.

**How to apply:** For any task prescribing new error text —
1. `grep -rn "<old literal fragment>" --include="*_test.go" .` — find every
   assertion on the message FAMILY, not just the exact sentence. Prefix-shared
   messages (`workflow/run selection miss:`) have several siblings.
2. Trace which INPUT reaches the rewritten return. An empty-collection fixture
   often reaches the terminal `return` the plan is rewriting, even though the
   plan only reasoned about the populated case.
3. If a pre-existing assertion breaks, demand the plan prescribe ONE concrete
   message that satisfies both the new claim's required substrings AND the
   surviving assertion — that is nearly always possible and is a one-line fix.
   Treat any "the task owns the consequence" hedge as a scope defect, not
   latitude.

Related: [[project_gate_source_string_assertions]] (tests string-matching source
text), [[project_retired_feeder_test_collateral]] (collateral test breakage from
a seam change).
