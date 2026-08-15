---
name: feedback_omitted_subject_inherits_wrong_package
description: A claim with no explicit `subject:` inherits the spec-level package by default — if its mandated test actually lives in a different package, the substantiveness join can still pass through an incidental symbol reference or a kind:absence exemption, hiding the mis-subject until the spec flips to implemented
metadata:
  type: feedback
---

Found flipping SPEC-069 to `implemented` (2026-08-15): CLM-046, CLM-047,
CLM-086, CLM-088 and CLM-130 carried no `subject:`, so they inherited the
spec-level default (`pkg/initialize`) — but all five claims' mandated tests
actually live in `cmd/backstop/init_seams_test.go`, package `main`. This sat
green through the spec's entire `draft` lifetime because
`test_substantiveness` only enforces at `implemented` status
([[project_review_state_keystone]]-adjacent: enforcement gated on a status
value, not on the claim being wrong).

**Three non-obvious details that let this hide:**

1. **The substantiveness join is per TEST FUNCTION, not per file.** A file
   header comment asserting "this file references package X, so every test
   in it satisfies X's claims" is false — `pkg/gate/substantiveness_join.go`'s
   `NoTargetViolation` and `cmd/backstop/gate.go`'s
   `testFileColocatedWithTarget` evaluate colocation or symbol-reference
   per function body, not per file. `cmd/backstop/init_seams_test.go` had
   exactly this false comment, left uncorrected during this fix (source
   edit with no plan in flight — filed as an issue instead of hand-edited).

2. **Composite literals and constants do NOT count as a referenced symbol.**
   The four non-`kind: absence` claims' test bodies touch `pkg/initialize`
   only via `initialize.Options{…}` (a composite literal) and
   `initialize.OutcomeDelivered` (a constant) — neither is recorded by the
   join's symbol-extraction as "this test references package X," so the
   fallback path that would otherwise rescue an incidentally-correct
   mis-subject doesn't fire either.

3. **`kind: absence` is EXEMPT from the join entirely**, which looks like a
   pass but is actually a non-check — CLM-088 (an absence claim) "passed"
   this whole time for the wrong reason, the same mis-subject as its four
   siblings, just invisible because the join never ran on it at all.

**How to apply:** when authoring or auditing a claim with no explicit
`subject:`, don't trust the spec-level default to be right just because the
claim is siblings with others in the same requirement group — check where
its OWN mandated test(s) actually live. If a sibling claim in the same test
file already declares an explicit `subject:` different from the spec
default (as CLM-087/CLM-110 did here), that's a strong signal the omitted
ones are wrong too, not that the join will sort it out. Detect this the
same way [[feedback_claim_subject_is_one_package_only]] recommends for
straddling claims: force `implemented` on a scratch copy and run the real
join, don't wait for a live status flip to find out.
