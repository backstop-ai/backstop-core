---
title: "Artifact schemas lack terminal / end-of-life status states across all artifact types"
schema_version: issue/v1
number: "031"

issue:
  id: ISSUE-031
  title: "Artifact schemas lack terminal / end-of-life status states across all artifact types"
  type: technical-debt
  status: open
  created: "2026-06-24"

complexity:
  scope: cross-cutting
  uncertainty: known
  risk: moderate
---

# Artifact schemas lack terminal / end-of-life status states across all artifact types

## Problem

Backstop's artifact schemas have no way to express that an artifact is dead — superseded,
absorbed, withdrawn, or deliberately closed without implementation. The lifecycle enums for
most types end at "done" or "implemented" with no terminal variant. This forces dishonest
status choices whenever work is retired rather than completed.

The problem is confirmed across every schema in the tree. Findings below are verified against
the schema files directly.

### Per-type status audit

| Type | Schema path | Status enum | Terminal state? |
|------|-------------|-------------|----------------|
| **spec** | `artifacts/spec/v1/schema.json` | `["draft", "ready-for-implementation", "implemented"]` | **None** |
| **bundle** | `artifacts/bundle/v1/schema.json`, `artifacts/bundle/v2/schema.json` | `status.maturity: ["idea", "exploring", "defined", "ready"]` | **None** |
| **plan** | `artifacts/plan/v1/schema.json` | `["draft", "ready", "implementing", "completed"]` | **None** |
| **issue** | `artifacts/issue/v1/schema.json` | `["open", "ready", "in-progress", "blocked", "closed"]` | `closed` covers this case |
| **directive** | `artifacts/directive/v1/schema.json` | `directive.status: ["queued", "active", "specced", "done"]` | **None** (no `withdrawn`/`superseded`) |
| **adr** | `artifacts/adr/v2/schema.json` | `["Proposed", "Accepted", "Deprecated", "Superseded"]` | `Deprecated` + `Superseded` present |
| **standard** | `artifacts/standard/v1/schema.json` | `["draft", "active", "deprecated"]` | `deprecated` present |
| **capability** | `artifacts/capability/v1/schema.json` | `["draft", "defined", "ready", "in-progress", "verified", "broken", "deprecated"]` | `deprecated` present |

**Summary of the gap:** the four artifact types at the center of the SDLC pipeline — spec,
bundle, plan, directive — all lack a terminal state. Issues have `closed` which covers the
use case adequately. ADRs, standards, and capabilities already have terminal vocabulary.

### Concrete victim 1: SPEC-034 cannot be honestly marked superseded

SPEC-034 (`specs/SPEC-034-native-toolchain-engine-cutover.spec.md`, `status: draft`) is a
zombie. Its bridge work (`loadBridgedToolchainPacks`) landed; its deletion scope (eradicating
the bespoke Go path) was absorbed by BUNDLE-011 REQ-008, which explicitly states:

> "This bundle must ABSORB SPEC-034's unfinished deletion scope ... SPEC-034 must be marked
> SUPERSEDED/absorbed." — `bundles/BUNDLE-011-collapse-legacy-codecheck-into-packs.bundle.md`,
> REQ-008

There is no valid `spec/v1` status value for this. The options are:

- `draft` — the current lie-by-omission: SPEC-034 looks like live work-in-progress;
  implementers may pick it up
- `implemented` — a factual lie: the deletion never reached `main`
- `ready-for-implementation` — a factual lie: nothing is ready; the work was absorbed elsewhere

None of these is honest. SPEC-034 must stay at `draft` with a prose tombstone note until this
issue lands a valid `superseded` status. That workaround is fragile and invisible to tooling.

### Concrete victim 2: SPEC-001 is ALREADY FAILING VALIDATION today

SPEC-001 (`specs/SPEC-001-standards-compiler.spec.md`) carries `status: active`. This value
does not exist in the `spec/v1` status enum. The artifact is live in the tree, checked in,
and fails validation on every run:

```
SPEC-001-standards-compiler.spec.md
  ✗ [spec/invalid-status] status 'active' is not valid (allowed: [draft ready-for-implementation implemented])
  ✗ [spec/claim-requirement-invalid] claims[8] references unknown requirement 'REQ-009'
  ✗ [spec/claim-tests-empty] claims[8] must have at least one test
```

Confirmed by running `./bin/backstop artifact validate` on 2026-06-24. The `active` status
failure is directly caused by the missing terminal/historical-state vocabulary in the spec
schema. SPEC-001 documents the standards-compiler model that is now strategically retired
(per ISSUE-030 and BUNDLE-011). Its correct honest status would be `superseded` or
`deprecated` — neither of which exists.

SPEC-001's reconciliation (moving it to a valid status that accurately reflects its retired
state) is a direct unblocking dependency on this issue.

### Bundle and plan gaps: same structural problem

Bundles have a forward-only maturity ladder (`idea → exploring → defined → ready`) with no
way to mark a bundle abandoned, absorbed into another, or superseded. A bundle whose work was
fully absorbed by a different bundle must stay at whatever maturity it reached, which gives no
signal to tooling or planners that the bundle is inert.

Plans have `completed` but no `withdrawn` or `superseded`. A plan authored for a spec that
gets superseded becomes stranded: it cannot be marked done (the implementation never ran) and
cannot be marked withdrawn.

Directives have `done` but no `withdrawn`. A directive pulled from the backlog without
implementation has no valid end state.

### Scope

This is cross-cutting technical debt in the schema layer. It blocks honest artifact lifecycle
management across the four most active artifact types and is already causing a live validation
failure (SPEC-001) in the tree.

## Solution

Each artifact type's status enum gains the terminal state(s) it needs:

- **spec/v1**: add `superseded` and/or `withdrawn` to the status enum
- **bundle/v1 + v2**: add a terminal maturity value (e.g., `superseded`, `withdrawn`,
  `absorbed`) to `status.maturity`
- **plan/v1**: add `withdrawn` (or `superseded`) to the status enum
- **directive/v1**: add `withdrawn` to the `directive.status` enum

Once the spec schema has a valid terminal status:

- SPEC-001's `status: active` is reconciled to the correct terminal value (resolving the live
  validation failure)
- SPEC-034 is marked `superseded` with a `superseded-by: BUNDLE-011` reference (unblocking
  honest tombstoning that BUNDLE-011 REQ-008 explicitly requires)

### Open design questions (not pre-resolved — user drives these)

**DQ-1: Should the gate / planners SKIP artifacts in a terminal state?**
A `superseded` spec should probably not be scanned for implementation gaps or surfaced as
outstanding work. But should `backstop gate` silently ignore superseded artifacts, warn about
them, or require an explicit exclusion? This affects the gate's coverage semantics.

**DQ-2: Should a terminal status REQUIRE a supersession reference, and should validation
enforce it?**
A `superseded` spec with no `superseded-by:` field is less useful than one that says
`superseded-by: BUNDLE-011`. Should validation require the reference field when status is
`superseded`? What's the reference format — free string, typed BUNDLE-NNN/ISSUE-NNN pattern,
or an array for multi-absorber cases?

**DQ-3: Uniform terminal vocabulary across types, or per-type states?**
ADRs use `Superseded`/`Deprecated` (title-case). Standards/capabilities use `deprecated`
(lowercase). A consistent vocabulary across all types (`superseded`, `withdrawn`,
`deprecated`) would make tooling simpler and the mental model cleaner, but may not map
cleanly to each type's semantics. Should a schema-wide convention be established?

**DQ-4: Is `deprecated` distinct from `superseded` for specs and bundles?**
For ADRs the distinction is meaningful (`Deprecated` = no longer recommended but not
replaced; `Superseded` = replaced by a specific other ADR). Should specs and bundles carry
both variants, or is a single `superseded` sufficient?

## References

- `artifacts/spec/v1/schema.json` — spec status enum `["draft", "ready-for-implementation", "implemented"]`; no terminal state
- `artifacts/bundle/v1/schema.json`, `artifacts/bundle/v2/schema.json` — maturity enum `["idea", "exploring", "defined", "ready"]`; no terminal state
- `artifacts/plan/v1/schema.json` — status enum `["draft", "ready", "implementing", "completed"]`; no terminal state
- `artifacts/directive/v1/schema.json` — `directive.status: ["queued", "active", "specced", "done"]`; no withdrawn/superseded
- `artifacts/issue/v1/schema.json` — `status: ["open", "ready", "in-progress", "blocked", "closed"]`; `closed` is adequate
- `artifacts/adr/v2/schema.json` — `["Proposed", "Accepted", "Deprecated", "Superseded"]`; terminal states present (reference model)
- `artifacts/standard/v1/schema.json` — `["draft", "active", "deprecated"]`; terminal state present
- `artifacts/capability/v1/schema.json` — includes `deprecated`; terminal state present
- `specs/SPEC-001-standards-compiler.spec.md:5` — `status: active` (invalid value, fails `[spec/invalid-status]` today)
- `specs/SPEC-034-native-toolchain-engine-cutover.spec.md:5` — `status: draft` (honest value forced by absence of `superseded`)
- `bundles/BUNDLE-011-collapse-legacy-codecheck-into-packs.bundle.md` REQ-008 — explicitly requires SPEC-034 be marked superseded/absorbed
- ISSUE-030 — eradicating the standards-compiler lineage (also makes SPEC-001 a candidate for terminal status)
