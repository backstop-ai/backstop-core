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

**How to apply:** for every symbol whose signature the plan changes, run
`grep -rn "name: <Symbol>" specs/` and check each hit — `provides:` entries carry a
`signature:` and DO drift; `consumes:` entries carry only name/kind and do NOT. Verify
each owning spec's `status:`. Then confirm the plan's spec-author reconciliation task
names every drifted entry, not just the headline one. Related:
[[project_signature_change_package_fanout]] (the test-file compile fanout of the same
change) and [[project_repurposed_test_claim_text_drift]] (claim TEXT that still asserts
the pre-change enumeration even when the mandated test NAME is preserved).
