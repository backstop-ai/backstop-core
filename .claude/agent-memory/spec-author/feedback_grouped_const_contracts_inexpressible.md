---
name: grouped-const-contracts-inexpressible
description: A contract provides entry for a member of a grouped Go `const (…)`/`var (…)` block can never match; drop the entry with an ISSUE-078 forward reference instead of declaring it
metadata:
  type: feedback
---

Never declare a `kind: constant` (or `kind: variable`) `provides` entry for a symbol that lives
inside a grouped `const (…)` / `var (…)` block. Drop the entry and record WHY in a neighbouring
contract's `notes`, with a forward reference to **ISSUE-078**, which tracks the gap and folds it
into ISSUE-052's relational-rule `input_mode` as the general fix.

**Why:** the contracts pack's signature compiler
(`.backstop/packs/backstop-ai/go-contracts/scripts/compile-signature.sh`) emits an ast-grep
pattern `const NAME = $$$` from a `const NAME = value` signature. That pattern binds only a
STANDALONE declaration — it does not reach a `const_spec` nested inside a parenthesized
declaration. Verified with ast-grep directly: the same pattern matches
`pkg/gate/result.go:44`'s standalone const and matches nothing in a grouped block.

Two failure shapes, both real and both hit during SPEC-070's close-out:
- **The group-label entry.** One entry named for the whole block (e.g. `doctorCheckIDs`) with a
  `const ( A = …; B = … )` signature. A Go const block is ANONYMOUS — that symbol does not
  exist — and the compiler reads the token after `const ` as the name, so it emits `const ( = $$$`,
  which matches nothing anywhere.
- **The obvious repair, which is also wrong.** Splitting into N single-constant entries turns
  1 violation into N. Grouping is the blocker, not the signature form.

The repo's settled disposition, stated by SPEC-035 v1.1.2 and followed by SPEC-054
(`KindScaffolding`/`OpCreate`) and SPEC-070: **declaring an unverifiable entry buys a red, not a
guarantee.** Dropping it is not weakening enforcement, provided the invariant the constants exist
for keeps a real guard — for SPEC-070 that is CLM-059, a `kind: absence` source scan proving no id
is written as a literal outside the constants. Do NOT reshape source into standalone consts to
satisfy the compiler: that is a source edit with no plan in flight, and it shapes code to fit a
check's expressive gap. If `type ApplyMode string`-style named type wraps the values, declare the
TYPE instead (SPEC-054's third variant) — a verifiable form of the same surface.

**How to apply:** when authoring or amending a `contracts` block, grep the target file for
`const (` / `var (` before declaring any constant or variable provide. This is invisible to
`artifact validate` — `pkg/validate/contracts.go` checks only that name/kind/signature are PRESENT
and never whether the symbol exists or the signature compiles — so it survives spec review and
plan review and detonates at the `draft` -> `implemented` flip, when `contract_signature` first
collects the spec.

Related: [[kind-function-contracts-existence-only]] (the same compiler's `type`/`func` entries are
existence-only), [[omitted-subject-inherits-wrong-package]] and
[[claim-subject-is-one-package-only]] — the sibling close-out-only defect class on the
substantiveness side. See [[close-out-must-rerun-gate-after-flip]].
