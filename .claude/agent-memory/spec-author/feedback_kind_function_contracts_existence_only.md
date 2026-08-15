---
name: kind-function-contracts-existence-only
description: A `kind: function` contract entry compiles to an existence-only ast-grep query — the gate never compares parameter/return lists, so signature drift is invisible until an unrelated change drags the file into diff scope; a shape change must edit every declaring spec in the same pass or the others rot silently
metadata:
  type: feedback
---

Found fixing SPEC-054's `runRecipeApply` contract (2026-08-15): the declared
signature had been missing a parameter and a return value for weeks — `--param`
(CLM-089..093, SPEC-054's own v1.5.0 work) added `suppliedParams []string`
without updating the contract string alongside it — and nothing ever went red.

**Why it stays hidden:** a `kind: function` contract entry compiles to an
EXISTENCE-ONLY ast-grep query. The gate confirms a function by that name
exists and never compares parameter or return lists; `artifact validate`
doesn't either. The one textual guard in this repo
(`pkg/recipe/contract_signature_test.go`) is deliberately narrow to
`kind: type` entries in one file — it does not generalize. So a signature can
drift arbitrarily far from its contract block and stay fully green until an
unrelated change drags the file into gate diff-scope for the first time,
which can be weeks or months later and look like a fresh regression when it
is actually old debt surfacing.

**How to apply:** when authoring or auditing a `kind: function` contract
entry, re-verify the declared signature against the real source yourself —
a green gate is not evidence the string is right, only that the symbol still
exists under that name. Two specs declaring the same symbol is an
established, accepted pattern in this corpus (e.g. `loadInstalledPacks` is
declared by SPEC-017 + SPEC-069 + SPEC-070) — not something to resolve by
deferring one spec to another. That means a real shape change to a
multiply-declared symbol must edit EVERY declaring spec's contract string in
the same change, or the specs you didn't touch rot the same way, silently,
with no red anywhere to catch it. `absent: true` is not a substitute fix for
stale-but-live signatures — in this corpus (SPEC-045, SPEC-043, SPEC-038) it
specifically asserts a symbol was deleted; pointing it at a function that
still exists inverts the deletion guard into a false alarm.
