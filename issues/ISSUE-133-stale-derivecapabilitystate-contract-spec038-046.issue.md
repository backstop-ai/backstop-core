---
title: "SPEC-038 and SPEC-046 Declare Stale deriveCapabilityState Signatures"
schema_version: issue/v1

issue:
  id: ISSUE-133
  title: "SPEC-038 and SPEC-046 Declare Stale deriveCapabilityState Signatures"
  type: bug
  status: open
  created: "2026-08-15"

complexity:
  scope: isolated
  uncertainty: known
  risk: safe
---

# ISSUE-133: SPEC-038 and SPEC-046 Declare Stale deriveCapabilityState Signatures

## Problem

`deriveCapabilityState` is a multiply-declared symbol across specs that touch the
capability-classification arm of the gate. Its **real, currently shipped** signature
(`cmd/backstop/gate.go:439`) is:

```
func deriveCapabilityState(packs []*pack.Manifest, dim gate.TraceabilityDimension, stack string) gate.CapabilityState
```

SPEC-037 (`specs/SPEC-037-traceability-substantiveness-pack.spec.md:702`, status
`implemented`) was just corrected during its own close-out and now declares this exact
signature — verified directly, it matches. That correction was scoped to SPEC-037's own
contract block only. Two OTHER specs that declare the same symbol were not touched and
remain stale:

**SPEC-046** (`specs/SPEC-046-retire-language-toolchain-bridge.spec.md:421`, status
`implemented`):

```
signature: "func deriveCapabilityState(cfg *config.Config, dim gate.TraceabilityDimension, stack string) gate.CapabilityState"
```

Wrong in the first parameter: `cfg *config.Config` instead of `packs []*pack.Manifest`.
This is an intermediate signature state — SPEC-046 evidently added the `stack string`
parameter correctly but never migrated the first parameter off the `cfg` config-object
form.

**SPEC-038** (`specs/SPEC-038-traceability-contracts-pack.spec.md:698`, status `draft`):

```
signature: "func deriveCapabilityState(cfg *config.Config, dim gate.TraceabilityDimension) gate.CapabilityState"
```

Wrong on both counts: still `cfg *config.Config` for the first parameter, and missing
the `stack string` parameter entirely — a signature two revisions behind current.

### Why this is invisible to the gate

`kind: function` contract entries compile to existence-only ast-grep queries (the
compiler emits a `func $NAME($$$) $$$`-shaped pattern that matches on the function name
existing, never on its parameter or return list — see
`.claude/agent-memory/spec-author/feedback_kind_function_contracts_existence_only.md`).
So the `contract_signature` gate dimension cannot and does not detect this drift: it
confirms `deriveCapabilityState` exists somewhere in `cmd/backstop/gate.go`, which is
true in all three specs, and stops there. There is no gate-red today and none is
expected from this drift under the current compiler — the specs are simply wrong
documentation of a real function's signature, silently, with no forcing function to
catch it.

To be precise about severity: SPEC-046 is `implemented`, so its contract entry IS
actively enforced by the existence-only presence check — but that check cannot and does
not catch a parameter-list mismatch, so this is not "currently causing a gate failure,"
it is a documentation-accuracy gap that the gate structurally cannot see. SPEC-038 is
still `draft`, so its stale entry has never even been enforcement-eligible.

### Root cause

Same lesson already on record in this repo: a multiply-declared symbol must be edited in
every declaring spec when its real shape changes, not just the spec doing the shipping.
SPEC-046 (which added the `stack` parameter) and whatever spec removed the `cfg
*config.Config` first parameter in favor of `packs []*pack.Manifest` each updated their
own contract block (or, in SPEC-046's case, updated it partially) without sweeping the
other specs that also declare `deriveCapabilityState`. SPEC-037's close-out only
corrected SPEC-037's own copy.

## Solution

Not committed here — issue-author does not edit specs. Needs a spec-author dispatch per
stale spec (same mechanical fix already applied to SPEC-037's contract block):

1. **SPEC-046** (`specs/SPEC-046-retire-language-toolchain-bridge.spec.md:421`): update
   the declared `deriveCapabilityState` signature's first parameter from
   `cfg *config.Config` to `packs []*pack.Manifest`, matching the real shipped function.
   No behavior change — spec-text-only correction.
2. **SPEC-038** (`specs/SPEC-038-traceability-contracts-pack.spec.md:698`): update the
   declared signature to add the missing `stack string` third parameter and change the
   first parameter from `cfg *config.Config` to `packs []*pack.Manifest`, matching the
   real shipped function.

Both corrections are independent and can ship separately or together. Neither touches
`cmd/backstop/gate.go` — the real function is already correct; only the specs' copies of
its signature are wrong.
