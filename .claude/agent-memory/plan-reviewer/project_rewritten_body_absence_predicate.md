---
name: rewritten-body-absence-predicate
description: A close-out "prove the BODY was rewritten, not just the name kept" predicate must be PRESENCE-only over the assertion/subtest names — an ABSENCE check on arrange code fires on a fully correct implementation
metadata:
  type: project
---

When a plan keeps a test's NAME and rewrites its BODY, it usually adds a
close-out predicate to prove the rewrite happened (a name check alone cannot
tell the old body from the new one). Check the predicate's polarity against the
NEW body the plan itself specifies:

- ARRANGE code is often byte-identical between the old fabricated body and the
  new correct one. If the new design says a leg is "the old scenario with its
  expectation INVERTED", the old fabrication's setup call (`os.Chtimes(stamp,
  older, older)`, a seeded fixture, a doctored config) is REPRODUCED verbatim on
  purpose. An ABSENCE predicate on that call reds a correct implementation.
- The real discriminator is the ASSERTION polarity plus the new subtest names.
  A NEGATED form of the old assertion (`!shimInvokedTest(`) inside the specific
  leg is a genuine discriminator IFF the old body contains only the positive
  form — grep the current file and confirm, and confirm the negated form does
  not already appear in the same function.
- Scope matters: the negated form often DOES appear elsewhere in the same file
  (a sibling test asserting the opposite outcome), so the predicate must name
  the enclosing function, and a hand-grep must honour that.

**Why:** PLAN-ISSUE-179 rounds 3-4 (2026-08-19) — TASK-015's first predicate
forbade `os.Chtimes(stamp`, which leg 3 legitimately emits as its arrange step;
it would have failed a correct fix. Replaced with presence-only: three subtest
names + `!shimInvokedTest(` inside the rewritten function, verified against the
shipped body (positive `if shimInvokedTest(` at one site, no `t.Run`, so a
reverted body fails all four).

**How to apply:** any plan that preserves a mandated test name while replacing
its body, or any close-out task with a "distinguishing structure" grep. Run the
predicate mentally against BOTH the reverted body and the plan's own specified
new body — it must fail the first and pass the second.

Related: [[repurposed-test-claim-text-drift]], [[matrix-test-subject-provenance]].
