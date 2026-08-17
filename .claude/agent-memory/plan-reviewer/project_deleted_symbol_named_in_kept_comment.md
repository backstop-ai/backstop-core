---
name: deleted-symbol-named-in-kept-comment
description: A plan that DELETES a symbol plus mandates a zero-grep-hits evidence check collides with its own "KEEP this comment" instruction when the kept comment names the deleted symbol
metadata:
  type: project
---

When a plan deletes a function/symbol AND (a) instructs the implementer to KEEP a
neighboring comment verbatim, AND (b) mandates a verification like "a repo grep for
`<oldName>` returns ZERO hits" — grep the real source for `<oldName>` inside every
comment the plan says to keep. Comments routinely name the function they explain.

**Why:** Observed on PLAN-ISSUE-140 (2026-08-16): TASK-008 deleted
`runNeverStarted` from `cmd/backstop/pack_gate.go`, item 4 said KEEP the call-site
inline comment in `runCoverageEngine`, and item 2 + TASK-009 required zero repo
grep hits for the old name — but that kept comment literally reads "…the shape
reaching runNeverStarted here is…". The two instructions jointly determine the
answer (keep the substance, repoint the name), so a careful implementer resolves
it, but the plan never says so and the grep-evidence step can be reported as
failing.

**How to apply:** In any deletion/rename task, run
`grep -n '<oldName>' <every file in scope>` and check the hits that live in
comments the plan preserves. Ask the plan to state explicitly whether the kept
comment's reference is repointed. Related: [[whole-file-delete-shared-types]],
[[deletion-file-premise-audit]], [[gate-source-string-assertions]].
