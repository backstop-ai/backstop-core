---
title: "Contracts Signature Engine Selected By Declared Capability, Not Engine-Key Name"
schema_version: issue/v1

issue:
  id: ISSUE-065
  title: "Contracts Signature Engine Selected By Declared Capability, Not Engine-Key Name"
  type: technical-debt
  status: open
  created: "2026-07-17"

complexity:
  scope: contained
  uncertainty: exploratory
  risk: moderate
---

# Contracts Signature Engine Selected By Declared Capability, Not Engine-Key Name

## Problem

`contractSignatureEngine` (cmd/backstop/gate.go) selects the engine that runs a compiled
signature pattern by testing the hardcoded engine key `manifest.Engines["ast-grep-contracts"]`
and falling back to `"ast-grep"`. This is a baked-name selection: a contracts pack that
declares its signature engine under any other key is not recognized and silently falls through
to the generic binding. Spun out of ISSUE-064 (which de-bakes the substantiveness rule-name
routing and the toolchain stack label); the ISSUE-064 backstop/self Family B7 rule flags this
site as a known instance.

## Root cause

Unlike ISSUE-064's two targets, there is **no existing declared field that discriminates the
signature engine** from its sibling. The contracts pack declares `gate_type: contracts` on BOTH
its engines — `ast-grep-contracts` (signature PRESENCE, structural pattern match) and `grep`
(symbol ABSENCE probe, literal search) — because gate_type is intentionally NON-exclusive: a
policy dimension is legitimately served by multiple engines with different mechanisms
(e.g. some rules enforced by semgrep, others by ast-grep, all part of one gate). Both engines
further share `input_mode: pattern-arg`, `scope_kind: file-args`, and `category: opinion`. The
ONLY things that distinguish them are the engine key, the `command` / `provision.tool` (baked
TOOL identity — Family A/B forbids keying on that), and the `convert` script path. So the
capability the signature dispatch actually requires — "consumes a compiled structural pattern"
vs "probes for a literal string" — is not declared as a first-class engine property; it is
implicit in the tool identity.

## Direction (to be specified)

Introduce a declared engine CAPABILITY/role the dispatch selects on, so the signature step
picks its engine by what the engine can DO, not by its key or tool. Candidate shapes (to
evaluate in the spec):

- An engine `capability:` / `role:` field (e.g. `signature-match` vs `absence-probe`) on the
  manifest engine spec, selected by the contracts dispatch.
- A pack-declared pairing between the signature COMPILER (`compile-signature.sh`, which emits an
  ast-grep pattern) and the engine that consumes its output — the compiler and match engine are
  a matched pair; declare the pairing rather than infer it by name.

Whichever shape wins, `contractSignatureEngine` selects by the declared capability, the
`"ast-grep-contracts"` key literal is deleted, and ISSUE-064's B7 rule can then be activated
live (it currently ships authored-but-not-enforced precisely because this site would trip it).

## Out of scope

- ISSUE-064's substantiveness routing and toolchain label de-bakes (shipping independently).
- Any change to gate_type exclusivity — gate_type stays non-exclusive by design; this issue adds
  a FINER declared discriminator, it does not repurpose gate_type.

## Notes / references

- Discovered by the post-ISSUE-063 baked-identity sweep; deferred out of ISSUE-064 after
  confirming (a) gate_type is non-exclusive and (b) no existing engine field discriminates the
  two contracts engines, so a clean fix needs a new declared capability rather than a name swap.
- Sibling to ISSUE-063 (capability by declaration, pack granularity) and ISSUE-064 (routing/label
  by declaration). This is the engine granularity, and unlike those it requires a new declaration
  rather than reusing an existing one — hence its own issue.
