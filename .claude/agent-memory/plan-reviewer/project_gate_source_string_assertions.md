---
name: gate-source-string-assertions
description: cmd/backstop has tests that string-match exact call forms in gate.go source; changing those call sites reds them even though the code is correct
metadata:
  type: project
---
cmd/backstop/gate_transitional_seams_test.go and gate_bridge_deletion_test.go read
gate.go as a STRING (readFileStr) and assert exact literal call/def forms are present
or absent — e.g. gate_transitional_seams_test.go:27 pins `coverageRecordsProducer(packs, projectRoot)`;
gate_bridge_deletion_test.go pins absence of "bridged", presence of `mergeSourceClassifier(packs)`,
`loadInstalledPacks(projectRoot)`, etc.

**Why:** a plan that rewires a pinned call site (e.g. ISSUE-068 threading a shared
cache by changing `coverageRecordsProducer(packs, projectRoot)` to pass the cache)
reds these guards even though the runtime code is correct. The break is invisible in
the plan's own new tests and in the validator — it only shows when the pinned test runs.

**How to apply:** when reviewing a cmd/backstop plan that changes a function's call
form or signature in gate.go, grep gate_transitional_seams_test.go + gate_bridge_deletion_test.go
for the exact symbol. If a source-string test pins it, the plan MUST either scope +
update that test, or thread the change WITHOUT altering the pinned literal (e.g. inject
at the coverageRecordsFn INVOCATION / buildCoverageStep rather than the producer
construction). Sibling of [[project_dispatch_consumer_edges]] / [[shared-cache-seam-wiring]].
