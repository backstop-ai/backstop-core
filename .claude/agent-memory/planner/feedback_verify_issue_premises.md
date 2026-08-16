---
name: verify-issue-premises
description: "Verify an issue's factual premises (especially 'no test asserts on this') against source before planning — they are claims, not facts"
metadata:
  type: feedback
---

**An issue's stated premises are claims to verify, not facts to inherit.** Before
planning a removal or a change an issue calls "inert," check the premise against
source yourself.

**Why:** ISSUE-082 (2026-08-15) asserted in its Acceptance Criteria that "no test
currently asserts on the five removed keys" and that the fix was confined to one
file. Both were false. Four tests read the map and asserted membership directly,
bypassing the code path the issue had analyzed — and three of the five keys were
load-bearing for MANDATED test names of spec claims, one on a spec with status
`implemented`. A code-only removal would have failed the suite AND tripped the
`artifact_status_drift` gate dimension as a broken promise. The issue was not
careless; it analyzed the runtime path correctly and simply never grepped for
direct-membership assertions. Founder approved widening scope to amend both specs
(SPEC-038 v1.2.2, SPEC-047 v1.2.1) before the cleanup could proceed.

**How to apply:** when an issue proposes deleting a symbol, map entry, or config
key, grep for every reader of it before scoping the plan — not just the runtime
call path the issue describes. Then check whether any test touching it is a
mandated test name of a spec, and what that spec's `status` is: `implemented`
means `status_drift` will treat a deleted mandated test as a broken promise, so
the spec must be amended first via spec-author. Deleting or weakening a mandated
test to make a plan executable is never the answer.

Corollary that also paid off on the same issue: an issue's own verification
command can be too narrow. ISSUE-082's sweep glob (`packs/*/pack.yml`) missed
every testdata manifest; sweeping wider confirmed the conclusion but the issue's
command could not have proven it.

Related: [[code-check-command-removed]]
