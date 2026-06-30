---
name: deleted-concept-orphans-consumer
description: When one spec deletes a concept (e.g. `bridged`) another spec still sources from, check that the deletion spec's edit-set covers the consumer's call site — or it's an orphan
metadata:
  type: feedback
---

In a parallel cutover, Spec X deletes a concept while Spec Y still builds on it; the bug is an OWNERSHIP GAP at the consumer's call site that neither spec's edit-set covers.

**Why:** BUNDLE-012 SPEC-046 deleted the `bridged` toolchain-pack list everywhere, but SPEC-043's `mergeSourceClassifier(bridged, packs)` call site still referenced it, and SPEC-046's `bridged`-removal edit-set listed coverageRecordsProducer/toolchainEnforcementStatus/etc. but NOT mergeSourceClassifier (a 043 symbol). Result: a dangling reference no spec owns fixing. Contrast SPEC-045, which hedged ("read whatever manifest set survives SPEC-046") — that hedge is what 043 lacked.

**How to apply:** When Spec X removes a variable/type/field, grep every sibling spec for that token. Each consuming site must appear in EITHER X's removal edit-set OR the consuming spec's "defer to what survives" language. A consuming spec that hard-codes the soon-deleted concept (no hedge) AND is absent from the deleter's edit-set = BLOCKING orphan. Name the single owner for the call-site reconciliation.
