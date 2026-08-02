---
title: "Remote Dependency Assembly"
number: DIR-026
created: "2026-07-26"
schema_version: directive/v1

directive:
  status: active
  source:
    - "BUNDLE-006"
    - "SPEC-055"
    - "SPEC-056"
    - "ISSUE-083"
    - "ISSUE-095"
---

## Description

Make remote (git-hosted) pack consumption actually work in production.
Packs are external by design — every pack lives in its own repository and
is installed via `pack add org/pack-name@version` — so this is not one
feature among many, it is the mechanism the whole distribution model
depends on. Today it does not run: `pack add` on any non-local ref
dereferences a nil `GitCloner` and panics with a raw stack trace, `pack
update` and `pack upgrade` share the same nil-wiring exposure, and the
`GitCloner` interface has no production implementation anywhere in the
tree — only mocked in tests. Every real pack install to date, including
this repo's own, is `source_type: local`. Until this closes, "install a
pack by name from its tap" is an advertised capability, not a delivered
one.

ISSUE-073 documented the defect (root-caused against source: nil
`GitCloner`/`Validator` in `cmd/backstop/pack_add.go`, the panic site at
`pkg/pack/distribution/add.go:153`, the same exposure in `update.go`/
`upgrade.go`, and zero production `GitCloner` implementations — only
mocks). That surface has since moved onto the BUNDLE-006 →
SPEC-055 → PLAN-SPEC-055 chain: BUNDLE-006 (`pack-distribution-lifecycle`,
`defined`, v0.9.0) reopened around live 2026-07-25/26 re-verification
evidence that reproduced the panic from the current tree; SPEC-055
("Production Remote Dependency Assembly", v1.3.0) scopes the concrete
production `ExecGitCloner`/`PackvalValidator`/`TagVersionResolver`, the
fail-closed constructor redesign (no command can be assembled with a
missing dependency), and the cross-cutting silent-exit-1 repair; and
PLAN-SPEC-055 (52 tasks / 12 phases) is the active implementation, with
phases landing in sequence (hermetic remote substrate, reportError seam +
stream-separated runner, per-site error surfacing, `ExecGitCloner` with a
real clone/remote-tags/Clone-strip, `PackvalValidator` +
`TagVersionResolver`, and fail-closed positional constructors all merged
as of 2026-07-26). PLAN-ISSUE-073 was retired `replaced`, by SPEC-055 —
its analysis was substantially adopted by the successor, not discarded;
the surface moved from the reactive issue→plan track onto the feature
bundle→spec→plan track once BUNDLE-006 was already open around it.

**Why a new directive rather than folding into an existing one** (considered
and rejected, recorded here since it isn't obvious): DIR-023 (Pack
Distribution Hardening) cites ISSUE-055 and ISSUE-058 — the local-pack
provenance cache and registry-era pack-detection, a different slice of pack
distribution entirely, at a different time horizon (one near-term, one
explicitly deferred pending registry infra). DIR-009 (End-to-End Pack Smoke
Test) is the only existing directive that cites BUNDLE-006, but it is a
`done` (2026-04-23) directive describing an earlier integration-smoke-test
milestone, not this reliability workstream — and it correctly carries no
BACKLOG.yml entry, since done directives may be dropped. Neither is the
right home; this workstream needed its own directive.

**Launch status: blocking.** The founder tiers launch blockers as four (as
of 2026-07-27): pack recipes (BUNDLE-015 / SPEC-054, under DIR-019), remote
pack consumption (this directive), Linux/CI viability (ISSUE-020, under
DIR-024), and CI-driven releases (DIR-001, tiered up 2026-07-27). Everything
else — explicitly including `backstop init` (DIR-002) — is tier-2
ergonomics: wanted, but not what makes backstop unusable if it slips.

## Notes

SPEC-055 implemented 2026-07-26 — all 12 phases of PLAN-SPEC-055 landed.
Ten packs are published under `backstop-ai` with real remote installs
proven (see DIR-027, which owns the ecosystem/fleet side of this work).
SPEC-056 implemented 2026-07-27 — all 13 phases of PLAN-SPEC-056 landed
(closing commit `067bd37`, "chore(SPEC-056): close delivered — spec
implemented v1.1.1, plan completed"). What remains under this directive is
the BUNDLE-006 code-side seed work SPEC-055 and SPEC-056 didn't scope:

- **REQ-039 (version/identity validation) — DELIVERED.** SPEC-056
  ("Remote Identity Version Validation", `implemented` at spec_version
  1.1.1, PLAN-SPEC-056 `completed`) shipped this: the manifest `name` —
  never the requested repository coordinate — is now the pack's
  install/runtime identity (install path, `backstop.yml` key, lock key,
  engine asset root); a new remote identity module
  (`pkg/pack/distribution/identity.go`) gates every cloning command with
  typed refusals (`*VersionUnresolvedError`, `*VersionMismatchError`,
  `*IdentityError`) before any consumer state is touched; the requested
  coordinate is recorded verbatim and preserved through the lock
  lifecycle (`LockEntry.SourceCoordinate`, DD-31) and read back through
  one shared accessor on install/update/upgrade/version-resolution;
  identity now gates on `pack update` and `pack upgrade`, not just `pack
  add`; and a hermetic identity E2E suite runs over the built binary.
  12+ of SPEC-056's claims carry
  `supports: pack-distribution-lifecycle:REQ-039@1.1.0`. The harness
  toolchain pack's manifest/tag drift (manifest declares `0.1.3` against
  tags stopping at `v0.1.1`, cited in DIR-027 item 2) was the shape this
  spec was built to catch and is now what the gate is validated against
  — SPEC-056 did not itself retag or amend that published pack, so the
  live drift persists as a fixture-proven case rather than a remediated
  one.
- **Transactions and the parity suite** — the remaining BUNDLE-006 DDs not
  covered by SPEC-055 or SPEC-056. This is now the actual code-side
  remainder under this directive.
- **ISSUE-095 (`pack add` silently no-ops a source conversion) — OPEN,
  unspecced.** `pack add <org>/<pack>@<version>` against a name already
  installed from a *local* source reports `Pack <name> is already
  installed and up to date` (`cmd/backstop/pack_add.go:76`) and exits 0
  while doing nothing: the lock keeps `source_type: local` and its
  out-of-repo `local_path`, records no `source_coordinate`, and no clone
  happens. The operator's evidence says the conversion succeeded; it did
  not. Measured during the PLAN-ISSUE-020 fleet migration on
  `backstop-ai/backstop-self@1.1.2` and `backstop-ai/go-standards@1.2.1`.
  Root cause, and why it is this directive's: the short-circuit gate is
  `isPackInstalledAndCurrent(projectDir, packName)`
  (`pkg/pack/distribution/add.go:135-151`) — it answers "does a non-empty
  dir exist at this name, and does the lock hold *an* entry for this
  name," taking no other input. `LockEntry`
  (`pkg/pack/distribution/lockfile.go:17-37`) carries `SourceType`,
  `SourceCoordinate`, `Version` and `GitRef` — precisely the fields
  SPEC-056 introduced and threaded through the lock lifecycle — and the
  gate reads none of them. Critically, the git-branch call site
  (`pkg/pack/distribution/command.go:242-244`) is the one SPEC-056
  *deliberately moved* to sit after identity resolution, with an in-code
  comment (`command.go:237-241`) explaining that it is keyed on the
  install name and warning "do not optimize it back." So `packName`,
  `version` and `gitRef` are all already resolved and in hand at the
  moment of the short-circuit, and simply are not compared against the
  entry the gate just found. This is the SPEC-056 identity surface with
  one comparison missing — not a new surface — which is why it is homed
  here rather than in DIR-023 (whose two threads are local-provenance
  caching and registry-era detection) or DIR-027 (which explicitly
  disclaims mechanism design: "Dependency, not scope"). The function
  already documents the behavior it lacks: its own doc comment
  (`add.go:128-134`) states that a pack "whose lock entry is
  missing/**diverged** is NOT installed-and-current and must be
  (re)installed." Divergence is written into the contract and never
  implemented; the fix restores the function to its stated contract
  rather than extending it. Wider than the issue's title, flagged for
  the fix planner: the gate is *version*-blind by the same omission — it
  accepts only `(projectDir, packName)`, and neither call site compares
  the resolved version against `lf.Packs[packName].Version`. By
  source-read (this half was NOT reproduced — the local→git half is the
  measured one), a version-differing `pack add` on an already-installed
  name reaches the same short-circuit and prints the same "already
  installed and up to date" line. Whether `pack add` *should* convert a
  version — as opposed to directing the operator to `pack
  update`/`pack upgrade` — is a semantics question for whoever specs the
  fix; the success message is false in either case. Blast radius,
  measured today: `backstop-core`'s own `backstop.lock` now holds six
  entries, all `source_type: git` under `backstop-ai` coordinates
  (including the newly extracted `go-contracts` and
  `go-substantiveness`) — its own migration completed *through* the
  confirmed `pack remove` + `pack add` workaround. Live exposure is
  therefore the remaining consumers (`bclabs-portal`, `stash`,
  `backstop-harness`) and any future operator who does not already know
  the workaround. Where it belongs in the remainder: with transactions
  and the parity suite (BUNDLE-006 REQ-040/REQ-042), not as a standalone
  thread — a hermetic lifecycle parity suite over
  `add`/`update`/`upgrade`/`relock` is the shape that catches this class
  of gate-blindness, and REQ-042 is still unspecced. No `PLAN-ISSUE-095`
  exists and no other directive cites the issue. Effect on the
  done-readiness question: this directive's Notes already flag, as a
  founder call, whether DIR-026 is close enough to `done` pending only
  transactions/parity — ISSUE-095 adds an open, unspecced,
  operator-visible silent-failure defect to that inventory, so that call
  should not be made against the pre-ISSUE-095 remainder.
- **REQ-041 (legacy-hash migration), DEMOTED** — remove+re-add under
  DIR-027's fleet migration writes fresh lock entries, so the migration
  mechanism's real-world exposure is now approximately zero; that low
  exposure is a consequence of a defect, not of a chosen workflow —
  remove+re-add is not merely the convenient path, it is currently the
  *only* path, since a bare re-add silently does nothing (ISSUE-095);
  still seeded here for BUNDLE-006 traceability, not treated as
  launch-blocking.

`status` remains `active`. SPEC-055 and SPEC-056 are both implemented;
what still holds this directive open is the remainder above —
transactions and the parity suite, neither yet specced — plus REQ-041
(demoted, not blocking). ISSUE-083 is cited here but is explicitly
post-launch and not-to-be-planned (see guard below), so it may not be
holding this directive open on merit; whether DIR-026 is now close
enough to `done` pending only transactions/parity is a founder call this
reconciliation pass does not make. Update to `done` (with
`directive.completed`) once the remainder lands and the gate is proven
green.

**ISSUE-083 (`resolveGitURL` hardcodes the GitHub host) — cited here, not
scoped for work yet, with two guards traveling with the citation:**

Sequencing: ISSUE-083 is post-launch. Every launch pack is first-party on
`github.com/backstop-ai` and every consumer installs rather than authors —
host-generality has zero real-world exposure before launch. It must not be
picked up ahead of REQ-039 (version/identity validation, the highest-value
remaining seed per PM assessment) within this directive's work.

Do not plan ISSUE-083 until the founder picks the resolution model. The
issue's own `uncertainty: exploratory` flag is load-bearing, not
decorative: its three candidate mechanisms — a host field in `backstop.yml`,
full-URL-as-coordinate accepted alongside the shorthand, or resolver
indirection (a host-prefix convention) — are three different architectures,
not variations on one. Full-URL-as-coordinate in particular would reopen
the `name == coordinate` convention ratified 2026-07-26 (the ten-pack
publication this directive's Notes section describes above), since a URL
is not a name. The founder question to put verbatim when planning is
proposed: "does a pack's identity stay its GitHub coordinate, or does the
coordinate become separable from the name?"

## References

- ISSUE-073 — original defect report (nil `GitCloner`, panic site, no
  production implementation)
- ISSUE-083 (`pack-resolution-hardcodes-github-host`) — `resolveGitURL`
  fixed-host defect; post-launch, unhomed pending founder resolution-model
  decision (see Notes guard above)
- `docs/CODEBASE-MAP.md` "Known gap — remote pack resolution" section
