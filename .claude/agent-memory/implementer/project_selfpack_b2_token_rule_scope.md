---
name: selfpack-b2-token-rule-scope
description: backstop/self Family B2 no-baked-language-token is the ONLY family without exclude *_test.go, so any test naming go.mod trips it; precedent fix is a line-scoped nosemgrep, and a founder posture decision is already escalated
metadata:
  type: project
---

`backstop/self` rule `no-baked-language-token` (Family B2, `rules/no-baked.yml`)
is a bare repo-wide `pattern-regex` with NO `paths:` block. Its four sibling
families (B3 neutral-spine, B4 repo-layout, B5 name-split, B6 pack-name-keyed)
ALL carry `exclude: ["*_test.go"]`. B2 does not — so it fires on ANY Go file
naming `go.mod`, `package.json`, `tsconfig`, `Cargo.toml`, `requirements.txt`,
`pom.xml`, `build.gradle`, or a foreign extension, test files included. (`.go`
is deliberately omitted from B2's regex; `.sh` is not in it at all.)

Measured 2026-07-27: all 12 files in backstop-core carrying a `"go.mod"`
literal are `_test.go` files. ZERO production files bake it. They sit dormant
only because file-level diff scope has not touched them — open any one and it
goes blocking.

**Why:** backstop-core IS the module under test, so its own repo-meta tests must
name its module file to assert on it. That is not the baked-routing-in-the-binary
hazard B2 targets, but B2 cannot tell the difference without a path scope.

**How to apply:** when a new/touched test legitimately names `go.mod`, use the
established in-repo precedent — a line-scoped
`// nosemgrep: backstop.packs.backstop.self.rules.no-baked-language-token — <reason>`
as in `pkg/pack/distribution/contracts_provisioning_test.go:26` and
`spec015_lineage_test.go:133`. Do NOT obfuscate the literal (`"go" + ".mod"`) —
that defeats the check textually while keeping the knowledge. Do NOT edit
`.backstop/packs/backstop/self/` (installed, non-durable); the pack source is
`local_path: ../backstop-self-pack`.

The broader question — self-pack scoping over backstop-core's OWN test harness —
is ALREADY ESCALATED and pending a founder posture decision, recorded in a
`deferred` waiver at `tests/smoke/smoke_test.go:53,246`. Cite that escalation
rather than re-opening it. Related: [[feedback_gostandards_rule_mechanics]],
[[project_editing_file_pulls_it_into_gate_scope]].
