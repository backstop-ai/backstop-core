---
name: contract-signature-block-drift
description: Any plan changing an exported func signature or adding a struct field must reconcile every `signature:` field in every implemented spec's contracts block naming that symbol — grep specs/ for the symbol, not just the spec the plan cites
metadata:
  type: project
---

A plan that changes an exported function's arity, or adds a field to a struct, silently
strands the `signature:` string in EVERY implemented spec's `contracts:` block that
declares that symbol under `provides:`. Planners reliably reconcile the ONE spec they
cite and miss the rest.

**Why:** contracts blocks pin literal signature strings (e.g. SPEC-068:862
`func FindUngatedArtifacts(projectRoot string, root artifact.Root) (...)`, SPEC-043:301
`type Classification struct { Source []string ...; Test []string ... }`). The gate's
contracts dimension checks declared-vs-real, and specs at `status: implemented` are the
ones it enforces (see [[project_gate_scoped_to_implemented]] in user memory). A plan
adding a parameter or a field drifts them all at once.

**The second axis planners miss: REQUIREMENT PROSE restates the same shape.** Grep the
bare SYMBOL, not `name: <Symbol>` — a spec's REQ text can carry an exhaustive, unhedged
struct literal (SPEC-043 REQ-001:59-60 `type Classification struct { Source []string;
Test []string }`, restated again in its design section at :540-541) far from any
contracts block. Same drift, not gate-enforced, and planners write "do NOT touch this
spec's requirements" while their own mandated sweep returns the hit. A plan that
mandates a corpus-wide sweep and then pre-dismisses one of its hits is contradicting
itself — flag it. (PLAN-ISSUE-122 round 4: reconciled SPEC-043:301 and forbade :59-60.)

**How to apply:** for every symbol whose signature the plan changes, run
`grep -rn "name: <Symbol>" specs/` AND `grep -rn "<Symbol>" specs/` — `provides:`
entries carry a `signature:` and DO drift; `consumes:` entries carry only name/kind and
do NOT; requirement/design prose restating the shape DOES drift. Verify each owning
spec's `status:`. Then confirm the plan's spec-author reconciliation task
names every drifted entry, not just the headline one. Related:
[[project_signature_change_package_fanout]] (the test-file compile fanout of the same
change) and [[project_repurposed_test_claim_text_drift]] (claim TEXT that still asserts
the pre-change enumeration even when the mandated test NAME is preserved).
