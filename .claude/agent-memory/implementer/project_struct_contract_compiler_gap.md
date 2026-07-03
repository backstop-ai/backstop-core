---
name: struct-contract-compiler-gap
description: contracts pack compile-signature.sh only handles func signatures; every `type X struct` contract reds under --file gate scope (pre-existing, codebase-wide)
metadata:
  type: project
---

The backstop/contracts pack's `scripts/compile-signature.sh` (the contract->ast-grep
pattern compiler, SPEC-038) ONLY handles `func` signatures. Given a `type X struct`
contract it strips `func `/`(`/`)` that aren't there and emits a malformed pattern,
so the ast-grep presence query never matches -> `contract_signature` reds with
"symbol X signature not found or mismatched ... expected \"type X struct\"".

**Why:** the compiler was written for function contracts; struct/type `kind: type`
contracts were never wired into it. This is codebase-wide: `gate --file
pkg/pack/manifest.go` reds on `type Manifest struct`, `type Content struct`,
`type Ruleset struct`, etc. -- all PRE-EXISTING struct contracts, not anything a
given change introduced. SPEC-043 added `type SourceClassifier struct` /
`type Classification struct` which join the same failing population.

**How to apply:** Do NOT panic or try to "fix" your struct definition when a
`--file`-scoped gate reds on a `type X struct` contract -- your code is fine; the
pack compiler can't verify struct contracts. The DIFF-scope gate (the
gate-on-implement hook's mode, and the real enforcement path) does NOT hit this:
with no `origin/main` remote the diff scope resolves empty, so no contract entries
are in scope and `contract_signature` passes (full gate stays PASS, exit 0). The
real fix belongs in the contracts pack (BUNDLE-009/SPEC-038 territory: teach
compile-signature.sh to emit `type X struct { $$$ }` for `kind: type` contracts),
NOT in consumer specs. Flag it; don't absorb it as scope creep. See
[[feedback_netnegative_gate_baseline]].
