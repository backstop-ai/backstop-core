---
title: "Artifact schemas lack terminal / end-of-life status states across all artifact types"
schema_version: issue/v1
number: "031"

issue:
  id: ISSUE-031
  title: "Artifact schemas lack terminal / end-of-life status states across all artifact types"
  type: technical-debt
  status: closed
  created: "2026-06-24"
  closed: "2026-07-08"

complexity:
  scope: cross-cutting
  uncertainty: known
  risk: moderate

delivered_by: PLAN-ISSUE-031
---

# Artifact schemas lack terminal / end-of-life status states across all artifact types

## Resolution

Added terminal/end-of-life status states (replaced/canceled/deprecated; bundle delivered) across all artifact schemas, with validation and gate exclusion of retired artifacts.

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

Design questions are resolved (see Resolved Design Decisions below). This section states the
implementation-ready specification.

### Terminal-state vocabulary

Two flavors of terminal state, applied per-type:

**Retirement terminals** (new across all SDLC types):

- `replaced` — replaced by a named successor; work lives on elsewhere. Validation MUST
  enforce that a `replaced-by:` field is present and well-formed when status is `replaced`.
  Format: a typed reference matching `BUNDLE-NNN`, `SPEC-NNN`, `ISSUE-NNN`, `PLAN-NNN`, or
  `DIR-NNN`; may be an array for multi-absorber cases.
- `canceled` — abandoned, no successor, work will not happen. Reason field is optional
  free-text. Spelling: one L (`canceled`), matching Go's `context.Canceled`.
- `deprecated` — was live or shipped, now discouraged. Optional reason field. Applies ONLY
  to spec and bundle (and the already-present adr/standard/capability). Do NOT add
  `deprecated` to plan, directive, or issue — those are tactical execution artifacts; they
  get only `replaced`/`canceled`.

**Success terminals** (one gap to fill):

All success terminals already exist for most types (`spec: implemented`, `plan: completed`,
`directive: done`, `issue: closed`, `adr: Accepted`, `capability: verified`) and are NOT
renamed or unified. The one gap: bundle has no success terminal. Its maturity ladder ends at
`ready` (= ready-to-spec), so a fully-delivered bundle permanently stalls at `ready`. Add a
`delivered` maturity value to bundle/v1 and bundle/v2. This gap exists because bundle-as-
unit-of-work post-dates the original maturity ladder design.

### Schema changes per type

| Type | Change |
|------|--------|
| `spec/v1` | Add `replaced`, `canceled`, `deprecated` to `status` enum |
| `bundle/v1` + `bundle/v2` | Add `delivered` (success terminal) AND `replaced`, `canceled`, `deprecated` to `status.maturity` |
| `plan/v1` | Add `replaced`, `canceled` to `status` enum |
| `directive/v1` | Add `replaced`, `canceled` to `directive.status` enum |
| `issue/v1` | Add `replaced`, `canceled` to `issue.status` enum (keep `closed` — `replaced`/`canceled` distinguish "absorbed by a bundle" / "abandoned" from "fixed") |
| `adr/v2`, `standard/v1`, `capability/v1` | No change — already have terminal vocabulary |

### Validation rules

- When `status` (or `status.maturity`) is `replaced`: `replaced-by` field MUST be present
  and MUST match one of the typed-ref patterns above. Validation fails if absent or
  malformed.
- When status is `canceled` or `deprecated`: an optional `reason` field may be present;
  validation does not require it.
- ADR keeps its existing Title-Case `Superseded`/`Deprecated` — renaming it would break
  existing ADRs and the ADR convention is standard. The new SDLC-type states are lowercase.

### Gate-exclusion behavior

`backstop gate` and planners MUST exclude terminal-state artifacts from enforcement. A
`replaced`, `canceled`, or `deprecated` artifact must not have its mandated tests or
contracts enforced; it must not be surfaced as outstanding work.

The gate outputs a count of excluded terminal artifacts (e.g., `N retired artifacts
excluded`) as an informational line — not a warning, since retirement is deliberate.

This exclusion is the load-bearing change that lets the standards-compiler-era stale
artifacts stop failing the gate. The current broken-promise backlog is dominated by
~622 red findings from removed-standards-compiler mandated tests and deleted-`pkg/compile`
contracts. Once those specs and bundles can be marked `replaced` or `deprecated`, those
findings are gated out entirely.

### Reconciliations (unblocked once spec schema is updated)

These are explicit downstream tasks, not part of the schema change itself:

1. **SPEC-001**: change `status: active` (currently invalid — causes live `[spec/invalid-status]`
   failure today) to `status: deprecated`. Rationale: the standards-compiler model is
   strategically retired per ISSUE-030 and BUNDLE-011; the spec was never superseded by a
   single named successor, it was made obsolete by a strategy change.

2. **SPEC-034**: change `status: draft` to `status: replaced` with
   `replaced-by: BUNDLE-011`. BUNDLE-011 REQ-008 explicitly requires this tombstoning. The
   bridge work landed; the deletion scope was absorbed into BUNDLE-011.

## Resolved design decisions

These questions were open at time of filing and are now ratified.

**DQ-1 (gate exclusion): RESOLVED.**
`backstop gate` and planners exclude terminal-state artifacts from enforcement entirely. A
`replaced`/`canceled`/`deprecated` spec does not have its tests or contracts enforced; it is
not surfaced as outstanding work. The gate reports a count of excluded artifacts as
informational (not a warning). Rationale: the alternative (enforcing dead contracts) is what
produces the ~622 vacuous-red findings currently polluting the broken-promise backlog.
Deliberately dead artifacts should be gated out, not noisily complained about.

**DQ-2 (validation requires replaced-by): RESOLVED.**
Validation requires `replaced-by` when status is `replaced`. The reference must be a typed
artifact ID matching `BUNDLE-NNN`, `SPEC-NNN`, `ISSUE-NNN`, `PLAN-NNN`, or `DIR-NNN`; an
array is allowed for multi-absorber cases. `canceled` and `deprecated` take an optional
free-text `reason` field; validation does not require it.

**DQ-3 (per-type vs. uniform vocabulary): RESOLVED.**
Per-type vocabulary, not a forced uniform rename. ADR keeps its existing Title-Case
`Superseded`/`Deprecated` — renaming it would break existing ADRs and violate the ADR
convention (these states are an established standard). The new SDLC-type states are
lowercase (`replaced`, `canceled`, `deprecated`, `delivered`). The intentional split is:
ADR convention-cased values for ADRs, lowercase for everything else.

**DQ-4 (deprecated vs. replaced distinction): RESOLVED.**
`deprecated` and `replaced` are distinct states. `replaced` = has a named successor (a
specific artifact absorbs or continues the work; `replaced-by` is mandatory). `deprecated` =
aged out or made obsolete by a strategy change, but no single named successor exists. This
mirrors the ADR `Deprecated`/`Superseded` distinction and applies it consistently to SDLC
types. `deprecated` is intentionally limited to spec and bundle (types with a "live/shipped"
concept); tactical execution artifacts (plan, directive, issue) cannot be deprecated, only
replaced or canceled.

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
