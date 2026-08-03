---
title: "Contract Block Drift On Nonterminal Specs"
schema_version: issue/v1

issue:
  id: ISSUE-115
  title: "Contract Block Drift On Nonterminal Specs"
  type: bug
  status: open
  created: "2026-08-02"
---

# Contract block drift is invisible until closure — validate at the status flip, not after

## Problem

A spec's `contracts:` block can drift silently out of sync with the tree for the entire time the
spec is non-terminal, and nothing observes the drift until someone tries to close the spec — the
worst possible moment to discover it.

The mechanism is `ExtractContractEntries` (`pkg/gate/step_testverify.go:556`), gated by
`contractsAreDue` (`pkg/gate/step_testverify.go:190`):

```go
func contractsAreDue(status string) bool {
	return status == "implemented"
}
```

`ExtractContractEntries` skips any spec where `!contractsAreDue(fm.Status)`
(`step_testverify.go:573`). So a `contracts:` block is inert while the spec is `draft` or
`ready-for-implementation`, and goes live the instant status flips to `implemented`. The code the
contracts describe keeps evolving the whole time — symbols get renamed, exported, deleted, or gain
fields — and nothing checks. Staleness accumulates unobserved and detonates at closure: the person
closing a spec that shipped correctly weeks earlier suddenly gets a red gate for it.

**Evidence — found in a single afternoon's closure pass, 2026-08-02.** Five specs were examined for
closure; **SPEC-019 was NOT stale** (all 8 declared signatures verified clean against the tree,
closed without incident). Drift was found in three:

- **SPEC-018** declared a package-level `var gateCmd *cobra.Command`. No such variable exists —
  `gateCmd` is a function-local at `cmd/backstop/root.go:91`. The real package-level surface is the
  constructor `newGateCommand` (`cmd/backstop/gate.go:33`), already declared immediately below it,
  so the stale entry was also a duplicate.
- **SPEC-030** declared `(*realCodeChecker).runCheck`. That symbol was deliberately deleted — and
  its deletion is part of what the spec delivered.
  `TestCutover_RealCodeCheckerDeleted` (`cmd/backstop/cutover_deletion_test.go:102-110`) asserts its
  absence. A `provides:` entry asserting a symbol EXISTS is the exact inverse of a spec whose
  deliverable is that symbol's ABSENCE. Root cause: the entry belonged to REQ-002, retired at
  spec_version 2.0.0 — the contract block outlived the requirement that justified it.
- **SPEC-036** declared `CapabilityState` with four fields; the tree has five (`Stack string`, added
  by SPEC-046).

**Two findings that sharpen the issue — both change what the fix must cover:**

1. **`consumes:` entries drift too, and are never enforced at all.** `ExtractContractEntries`
   iterates `c.Provides` only (`step_testverify.go:582`) — `Consumes` is never read. So
   `consumes:` staleness is permanently invisible; it will not even fail at closure. Confirmed
   instances: SPEC-018 consumes `ComputeChangedFiles` from `pkg/check/scope.go`, a function that
   exists nowhere in the repo; SPEC-030 carries a whole contract block for
   `cmd/backstop/code_check.go`, a file that does not exist, consuming `check.Options` and
   `mergePackRules` (both deleted). These are stale text sitting inside terminal specs today.

2. **Not all drift is equally dangerous, and the fix should know the difference.** Verified by
   running the real `go-contracts` compiler: a `kind: type` signature compiles to
   `type CapabilityState $$$`, which is field-agnostic — the four-vs-five field drift on SPEC-036
   would NOT have failed the gate. It is a documentation defect. Whereas a declared symbol that
   does not exist at all (SPEC-018's var, SPEC-030's deleted method) DOES fail. The surface splits
   into "wrong but harmless" and "wrong and blocking," and a fix that only catches the blocking
   kind still leaves the corpus lying.

## Direction

Validate contract blocks against the tree at closure time, BEFORE the status flip to `implemented`
is accepted, rather than after. That way a spec cannot reach terminal status carrying contracts
that don't match, and the person closing it gets a precise diagnostic instead of a red gate later.
The natural home is wherever artifact validation already runs (`backstop artifact validate` / the
`artifact_validation` gate step), since that already guards artifact-shape invariants — but this is
a recommendation, not a prescription: whether it's a new validator, an extension of the existing
contract dispatch, or a pre-flight check is for the planner to decide.

Open question to carry into planning: should `consumes:` become enforced, or be dropped as
decorative? Today it is neither — declared, parsed, and never checked.

Likely home for routing: **DIR-024 "Gate/Engine Quality"** looks like the right directive for this
— flagging it here as a suggestion for the backlog-PM to route, not a self-assignment.

## Scope boundary

A separate defect was found in the same pass and must NOT be folded into this issue: the
`go-contracts` pack's `compile-signature.sh` cannot express a grouped `const ( ... )` block member
at all — it dispatches on the signature's leading token with no `const_spec` pattern, so SPEC-036's
`DimensionSubstantiveness` / `DimensionCoverage` / `DimensionContracts` entries match nothing no
matter how they're written. That is a pack defect with a pack-side fix (fix the pack, bump,
relock), and it is why SPEC-036 could not close today. Different fix site, different repo —
referenced here as related context only.
