---
name: struct-contract-compiler-gap
description: RESOLVED — the contracts pack compiler is kind-aware since ISSUE-036 (d5efd5b); `type X struct` contracts now COMPILE AND PASS. Kept as a correction because the old entry's advice ("your code is fine, ignore the red") would now hide a real finding.
metadata:
  type: project
---

STATUS CORRECTED 2026-07-28. Both premises of the original entry are now FALSE. It is
kept rather than deleted because its advice — "do not fix your struct; the pack cannot
verify struct contracts" — would today tell you to ignore a finding that is REAL.

WHAT WAS CLAIMED, AND WHAT IS TRUE NOW:

1. "compile-signature.sh only handles func signatures" — DISPROVEN. ISSUE-036 shipped
   kind-awareness in d5efd5b. The installed
   `.backstop/packs/backstop-ai/go-contracts/scripts/compile-signature.sh` dispatches on
   SEVEN shapes: method (`func (recv T)`), func, type (with an `interface` sub-case
   emitting `type X interface { $$$ }`, else `type X $$$`), const (`const X = $$$` — the
   `=` is required or ast-grep errors), var (RHS-preserving), a bare struct FIELD wrapped
   as `struct {\n$$$\n<field>\n$$$\n}`, and a func-shaped fallback.
2. "every `type X struct` contract reds under --file gate scope" — RE-CHECKED EMPIRICALLY
   and no longer true. `./bin/backstop gate --file pkg/pack/manifest.go` (the exact file
   the old entry named, carrying `type Manifest struct`, `type Content struct`,
   `type Ruleset struct`) now returns `contract_signature: pass, 0 violations`, exit 0.

ALSO STALE IN THE ORIGINAL: its escape hatch reasoned that diff scope "resolves empty
with no origin/main remote". backstop-core HAS had a real `origin/main` since 2026-07-28,
so diff scope resolves normally and that sentence no longer describes anything.

**Why:** this entry was written from a true observation and outlived its subject by two
pack versions. A memory that names a script's capabilities is a claim about the version
installed WHEN IT WAS WRITTEN — packs are versioned and external, so they move
underneath it.

**How to apply:** if a `type X struct` contract reds today, treat it as a REAL finding
and read the message, rather than reaching for this entry. Before trusting any memory
that describes pack-script behaviour, re-verify against the INSTALLED pack under
`.backstop/packs/` (the source under `packs/` can differ from what the gate runs) — the
two were byte-identical at 130 lines when this was checked, but that is a fact to
confirm, not assume. Related: [[project_pack_copies_and_stale_gate_binary]],
[[project_local_baseline_makes_gate_permissive]], [[feedback_netnegative_gate_baseline]].
