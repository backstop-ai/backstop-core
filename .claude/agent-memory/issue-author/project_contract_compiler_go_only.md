---
name: project-contract-compiler-go-only
description: ISSUE-053 — contracts compiler (compile-signature.sh) is Go-source-only; any contract on a non-Go file (pack.yml, JSON, MD, shell) is permanently unverifiable regardless of signature phrasing
metadata:
  type: project
---

`packs/contracts/scripts/compile-signature.sh` only emits Go-syntax ast-grep patterns
(func/method/type/const/var/struct-field, keyed on the signature's leading token). If a
contract's declared `file` is not Go source — a `pack.yml`, JSON schema, agent `.md`, shell
script — and its signature is prose-shaped (no Go leading keyword), the compiler falls
through to the struct-field wrap path and emits a Go `struct { ... }` pattern that can
never match non-Go content. This is a distinct, broader defect from
[[project_contracts_pack_kind_gaps]] (which is about kind-awareness *within* Go source,
e.g. bare iota consts) — this one is about the *target file's language*, not the
signature's Go declaration kind.

**Confirmed 2026-08-16 (via a duplicate-issue investigation, not filed — see below):**
SPEC-042's `go-coverage-rule` contract (declares `kind: variable`, target
`cmd/backstop/testdata/go-toolchain/.backstop/packs/backstop/go-toolchain/pack.yml`,
signature `"rule id: go-coverage, engine: go-coverage, gate_type: coverage"`) hits this
exact failure — reported as `contract_signature not found or mismatched` on a genuinely
correct pack.yml (the `id`/`engine` live on a `rules:` entry, `gate_type` lives on the
referenced `engines:` block — normal, valid split). The apparent "can't join facts split
across two YAML blocks" framing is a red herring; the real cause is the compiler being
Go-only, independent of whether the facts are joined or split.

**Already owned by ISSUE-053** (`issues/ISSUE-053-contract-signature-non-go-artifact-contracts.issue.md`,
open, technical-debt). ISSUE-053 explicitly named this exact SPEC-042 contract in its
"Important nuance" section as the re-audit trigger once ISSUE-051 (status-scoping to
`implemented` specs) landed — ISSUE-051 is now closed and SPEC-042 is `implemented`, so
this is that predicted recurrence arriving as a real gate blocker (surfaced via
PLAN-ISSUE-129), not a new defect.

**How to apply:** before filing ANY issue about `contract_signature` false-mismatch on a
non-`.go` target file, check ISSUE-053 first — it is the owning artifact. Annotate it with
the new evidence instead of filing a duplicate. Its three solution directions (skip non-Go
contracts / route to artifact-appropriate verification / retire prose signatures to
`notes`) are already drafted, not yet decided.
