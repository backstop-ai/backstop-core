---
title: "Tool Allowlist Unreachable Entries"
schema_version: issue/v1
delivered_by: PLAN-ISSUE-082

issue:
  id: ISSUE-082
  title: "Tool Allowlist Unreachable Entries"
  type: technical-debt
  status: closed
  created: "2026-07-26"
  closed: "2026-08-19"

complexity:
  scope: isolated
  uncertainty: known
  risk: safe
---

# Tool Allowlist Unreachable Entries

## Problem

`engine.TrustedToolAllowlist()` (`pkg/pack/engine/allowlist.go`) declares eight
`{tool -> pinned version}` entries — `semgrep`, `ast-grep`, `grep`, `rg`, `oxlint`, `bun`,
`tsc`, `prettier` — and its doc comment claims: "A tool ABSENT from this map may not be
run by any pack-declared command, no matter what a pack declares — this is the
non-negotiable security gate." Neither the entry set nor the guarantee holds up against
the code that actually consults the map.

### Five of eight entries are unreachable

Every call site of `CheckToolAllowed` gates on the engine binding carrying a **non-nil**
`Provision` block, verified directly against source:

- `cmd/backstop/pack_gate.go:813` `checkEngineToolAllowed` — `if binding.Provision == nil
  { return nil }`
- `pkg/pack/manifest.go:547` `validateEngine` — `if binding.Provision != nil { ... }`
- `pkg/packval/executor.go:63` `RunEngine` — same guard
- `cmd/backstop/pack_gate_provision.go:85` — routes through the same
  `checkEngineToolAllowed`, comment stating nil-`Provision` bindings are exempt

A `provision:` block is what makes an entry reachable. A sweep of every `pack.yml` in
this repo (`packs/`) and in the separate `backstop-packs` repo
(`~/src/projects/backstop-packs`) found `provision:` blocks for exactly three tools:

- `grep` — `packs/contracts/pack.yml`, `typescript-contracts/pack.yml`
- `ast-grep` — `packs/contracts/pack.yml`, `packs/substantiveness/pack.yml`,
  `packs/base-engines/pack.yml`, `typescript-contracts/pack.yml`,
  `typescript-substantiveness/pack.yml`
- `semgrep` — `packs/base-engines/pack.yml`

`rg`, `oxlint`, `bun`, `tsc`, `prettier` have **zero** provision blocks anywhere in
either repo. Notably, `typescript-toolchain/pack.yml` — the pack whose lint/typecheck/
format commands (`oxlint`, `tsc`, `prettier`, and implicitly `bun` as the package-manager
front door) motivated adding those four entries in the first place — declares no
`provision:` block at all; it invokes every tool via `npx --no-install`, which never
builds a `Provision`. So all five of these entries are dead: nothing in the codebase
reaches `CheckToolAllowed` with `tool` set to any of them.

### The suppressed dogfood rule was correct

The `tsc` entry carries `// nosemgrep: no-baked-language-token`, justified in an adjacent
comment as: "The allowlist KEY is a tool-name lookup datum ..., NOT a baked
routing/command literal ... it never sources a command." That argument is reasonable in
the abstract, but it was used here to suppress a `backstop/self` rule that had correctly
fired on a TypeScript tool name landing in a core Go file — the self-pack rule was right,
and the fix was to add dead code under a suppression rather than to not add the entry.
Removing the entry removes the suppression along with it; no re-justification is needed
because there is no longer anything to justify.

### The doc comment overstates the guarantee

Given the nil-`Provision` exemption at every call site, "a tool absent from this map may
not be run by any pack-declared command, no matter what a pack declares" is false as
written — a pack can declare (and does declare, via `typescript-toolchain`) commands
running tools that are not on the map at all, with zero gate involvement, because those
bindings carry no `Provision`. What `TrustedToolAllowlist` actually governs is narrower
and still legitimate: it is the trust floor for tools **backstop itself downloads and
pins on the user's behalf** (the `Provision` path — semgrep/ast-grep today, both riding
`backstop.lock`). Pack-declared commands against Layer-0/runtime tools the pack invokes
directly (grep as a POSIX tool, the whole Bun/TypeScript native toolchain) are outside
this gate's reach today, by construction, regardless of what the comment claims.

## Solution

1. Remove the five unreachable entries — `rg`, `oxlint`, `bun`, `tsc`, `prettier` — from
   `TrustedToolAllowlist()`, including the `// nosemgrep: no-baked-language-token`
   suppression that rides with the `tsc` entry and the comment block that justified it.
   Leave `semgrep`, `ast-grep`, and `grep` (the three entries with at least one real
   `provision:` consumer today) in place.
2. Rewrite the function's doc comment to describe what the gate actually covers: the
   trust floor for tools backstop provisions/pins on the user's behalf via the
   `Provision`/lock path, gated by a non-nil `Provision` on the engine binding — not a
   blanket guarantee over every pack-declared command. Remove the "no matter what a pack
   declares" language; it is not true today and this issue is not making it true (see
   Scope boundary below).
3. Re-run the provision-block sweep (`grep -rn "provision:" packs/*/pack.yml` in this
   repo and the equivalent in `~/src/projects/backstop-packs`) as the verification check
   that no pack anywhere references `rg`, `oxlint`, `bun`, `tsc`, or `prettier` through a
   `Provision`-bearing binding, so the removal cannot silently break an installed pack.

### Acceptance criteria

- `TrustedToolAllowlist()` contains only `semgrep` and `ast-grep` (Provision-backed) and
  `grep` (the one Layer-0 entry with a live `provision:` consumer via the presence-pin
  convention already documented in the surrounding comment) — `rg`, `oxlint`, `bun`,
  `tsc`, `prettier` are gone, along with their comment blocks.
- The `// nosemgrep: no-baked-language-token` suppression is gone — not moved, not
  re-justified elsewhere.
- `backstop/self` runs green over `pkg/pack/engine/allowlist.go` on its own, with **no**
  suppression comment anywhere in the file. Green must be achieved by the removal itself,
  not by adding a new or reworded suppression to cover the same or a different line.
- The doc comment on `TrustedToolAllowlist` no longer claims coverage of "any
  pack-declared command" — it states the narrower Provision/lock-pin scope this issue
  verified.
- The provision-block sweep against both `packs/*/pack.yml` (this repo) and
  `~/src/projects/backstop-packs/*/pack.yml` shows zero references to the five removed
  tool names after the change (i.e., nothing depended on them being present).
- Full test suite green (`go test ./...` unaffected — no test currently asserts on the
  five removed keys; if one is found during implementation it is asserting on dead code
  and should be removed alongside it, not preserved).

## Scope boundary

This issue is the cleanup only: delete unreachable allowlist entries and correct the doc
comment to match what the code does. It does **not** propose a mechanism for governing
arbitrary pack-declared commands against tools outside the Provision/lock path (the
`grep`-as-Layer-0-presence-pin convention, and the wholly ungated `typescript-toolchain`
tools). That governance question belongs to
`bundles/BUNDLE-021-pack-command-execution-governance.bundle.md` (filed in the same sweep
that surfaced this issue, `exploring`, not yet scoped). Do not design or implement a new
enforcement mechanism against this issue — if BUNDLE-021's eventual design wants
`typescript-toolchain`'s tools back on an allowlist-shaped structure, that is BUNDLE-021's
call to make from a real requirement, not a defensive re-add here.

## Priority note

Tier-2 by the founder's launch razor — real dead code and a false guarantee in a
security-adjacent comment, worth draining, but backstop is not unusable without this fix.
The three launch blockers remain recipes (SPEC-054), remote pack consumption (DIR-026 /
SPEC-055), and Linux/CI viability (ISSUE-020). This issue should not be triaged ahead of
those three.

## Resolution

Fixed at commit `d534e2a`, delivered by `PLAN-ISSUE-082` (`status: completed`). `TrustedToolAllowlist()`
(`pkg/pack/engine/allowlist.go`) now declares only `semgrep`, `ast-grep`, and `grep` — the five
unreachable entries (`rg`, `oxlint`, `bun`, `tsc`, `prettier`) and their accompanying `//
nosemgrep: no-baked-language-token` suppression are deleted outright, not reworded or relocated.
The doc comment now states the actual guarantee (trust floor for tools backstop itself
provisions/pins via the Provision/lock path) rather than the false "no matter what a pack
declares" blanket claim.

A new falsifying test, `pkg/pack/engine/allowlist_reachability_test.go`, asserts both directions
(the five removed names are absent, the three kept names are present) and that no suppression
marker (`nosem`/`@waiver:`) rides anywhere in the file. Two dependent specs whose mandated tests
had asserted membership of the removed entries were corrected in the same effort (SPEC-047
v1.2.1, SPEC-038 v1.2.2) — those assertions were vacuous from inception (the covered tools never
carried a Provision block, so they never reached the allowlist check at runtime), not drift.

Verified via an isolated-worktree HEAD control diff: zero net-new blocking violations; `gate
--all`'s pre-existing reds are byte-identical to HEAD's pre-existing debt.

## References

- `pkg/pack/engine/allowlist.go` — `TrustedToolAllowlist()` (the eight-entry map and its
  doc comment) and `CheckToolAllowed` (the pure gate function)
- `cmd/backstop/pack_gate.go:813` `checkEngineToolAllowed` — nil-`Provision` exemption
- `pkg/pack/manifest.go:547` `validateEngine` — same guard
- `pkg/packval/executor.go:63` `RunEngine` — same guard
- `cmd/backstop/pack_gate_provision.go:85` — same guard, routed through the shared check
- `packs/base-engines/pack.yml`, `packs/contracts/pack.yml`,
  `packs/substantiveness/pack.yml` — the only `provision:` blocks in this repo (semgrep,
  ast-grep, grep)
- `~/src/projects/backstop-packs/typescript-contracts/pack.yml`,
  `typescript-substantiveness/pack.yml` — the only `provision:` blocks in the sibling
  packs repo (grep, ast-grep)
- `~/src/projects/backstop-packs/typescript-toolchain/pack.yml` — zero `provision:`
  blocks; the pack the `oxlint`/`bun`/`tsc`/`prettier` entries were added for, and the
  proof they are unreachable
- `bundles/BUNDLE-020-pack-core-version-compatibility.bundle.md` — OQ-1 resolved that the
  tool allowlist is out of scope for pack↔core version compatibility; the allowlist's
  actual reachability was examined as part of that resolution and surfaced this issue
- `bundles/BUNDLE-021-pack-command-execution-governance.bundle.md` — owns the governance
  question this issue explicitly does not answer (see Scope boundary)
