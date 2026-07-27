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
    - "ISSUE-083"
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
What remains under this directive is the BUNDLE-006 code-side seed work
SPEC-055 didn't scope:

- **REQ-039 (version/identity validation)** — manifest semver must equal
  the effective tag; a live counterexample already exists (the harness
  toolchain pack's manifest declares `0.1.3` against tags `v0.1.0`/`v0.1.1`,
  cited in DIR-027 item 2). Now the highest-value remaining seed per PM
  assessment.
- **Transactions and the parity suite** — the remaining BUNDLE-006 DDs not
  covered by SPEC-055.
- **REQ-041 (legacy-hash migration), DEMOTED** — remove+re-add under
  DIR-027's fleet migration writes fresh lock entries, so the migration
  mechanism's real-world exposure is now approximately zero; still seeded
  here for BUNDLE-006 traceability, not treated as launch-blocking.

`status` remains `active` — SPEC-055 is implemented but the remaining
code-side seeds above have not yet been specced. Update to `done` (with
`directive.completed`) once they land and the gate is proven green.

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
