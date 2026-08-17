---
name: parity-test-cannot-catch-shared-walk
description: A biconditional parity test between two consumers of ONE extracted authority cannot detect a wrong walk inside that authority — both verdicts move together; the guard must live at the collection level
metadata:
  type: project
---

When you extract a shared authority and give it two consumers, a test asserting
`(consumer A fails) == (consumer B fails)` **cannot falsify a wrong implementation
inside the shared authority.** Widening the shared walk moves BOTH verdicts in
lockstep, so the biconditional stays true and the test stays green.

**Why:** PLAN-ISSUE-134 extracted `collectRequiredEngineTools` out of
`provisionEngines` and gave it a second consumer (doctor's engine-tools check). The
plan's falsification bar asserted that repointing the collection at `manifest.Engines`
instead of at rules would red `TestDoctor_EngineToolVerdictMatchesGateProvisioning`
"on the unbound row, where doctor fails and provisionEngines passes." Measured
2026-08-16: it did **not** red. provisionEngines consumes the same collection, so it
failed on the unbound row too and the biconditional held. The plan's premise silently
assumed doctor's walk could be widened independently of the gate's — which is exactly
what the extraction made impossible.

The mutation DID red the two tests that assert the walk directly:
`TestGate_CollectRequiredEngineToolsWalksRulesNotEngines` (collection level) and
`TestDoctor_EngineToolUnboundIsNotProbed` (report level).

**How to apply:** when a plan claims a parity/equivalence test is the guard against a
wrong shared implementation, RUN the mutation before believing it. Expect the parity
test to catch *divergence between the consumers* (a consumer that warns where the other
refuses, or that re-derives part of the walk) and nothing else — it is still worth
having, and it stayed non-vacuous here: neutering the presence probe to always pass DID
red it. But the guard against the wrong walk has to sit where the walk is: one test on
the authority's own output, plus one at each consumer's report. Report the corrected
premise rather than quietly substituting a different mutation. See
[[feedback_choose_compile_red_or_behavioral_red]] and
[[project_lock_the_chain_falsify_per_hop]].
