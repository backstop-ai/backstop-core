---
title: "Bundle Spec Promotion Gate Check"
schema_version: issue/v1

issue:
  id: ISSUE-057
  title: "Bundle Spec Promotion Gate Check"
  type: technical-debt
  status: open
  created: "2026-07-14"

complexity:
  scope: contained
  uncertainty: known
  risk: safe
---

# ISSUE-057: Bundle→Spec Promotion Gate Check

## Problem

**Original premise (2026-07-14, corrected below):** this issue was written
against a state where `pkg/validate/spec.go` only format-checked the
`supports` reference on a spec requirement and never resolved the referenced
bundle at all. That premise is now **false**. BUNDLE-014 (delivered
2026-07-15, see SPEC-050) shipped full both-direction resolution in
`pkg/validate/supports_resolution.go`, wired into `cmd/backstop/artifact_validate.go`
and the gate's `requirement_traceability` step:

- `supports` refs on spec (and issue) requirements now mandate an exact
  version pin — `bundle-name:REQ-NNN@MAJOR.MINOR.PATCH` — enforced by
  `supportsRe` in `pkg/validate/spec.go:16` (an unpinned or non-semver ref is
  now a format error, where it previously passed).
- `BuildBundleReqCatalog` + `CollectSupportRefs` + `ResolveSupports`
  (`pkg/validate/supports_resolution.go`) resolve every ref, corpus-wide,
  against the actual bundle: **bundle exists** (`supports/missing-bundle`),
  **REQ is declared** in that bundle's `requirements[]` (`supports/undeclared-req`),
  and **the pinned version is present** in that REQ's effective version log
  (`supports/version-unlogged`) — all three at `error` severity.

So "a spec can cite a REQ that doesn't exist, or a version that was never
logged" is closed. What is **not** closed, confirmed by re-reading
`ResolveSupports`/`BuildBundleReqCatalog` directly (`maturity` has zero hits
in `pkg/validate/supports_resolution.go`): **the cited bundle's own
`status.maturity` is never checked.** The resolution pass only keys on bundle
name, REQ id, and version-log membership — it is indifferent to whether the
citing bundle has ever been promoted past `exploring`.

This residual is real, not vacuous, but narrower than originally scoped.
Confirmed against `artifacts/bundle/v1/schema.json` and
`validateBundleRequirements` (`pkg/validate/bundle.go:527-570`): `requirements[]`
is only **required** from `defined` maturity onward — nothing in the schema or
validator **forbids** a bundle from carrying a populated `requirements[]`
block (with REQ ids and a version log) while still at `idea` or `exploring`.
So the concrete gap is: an `exploring` bundle that irregularly-but-validly
populates `requirements[]` early, and a spec that cites
`bundle-name:REQ-NNN@X.Y.Z` against it with a genuinely-logged pin, resolves
**fully clean** today — bundle exists, REQ declared, pin logged, zero errors.
Nothing anywhere asks whether the bundle was actually done exploring when it
was cited. This is the same ordering violation the BUNDLE-003 fossil exposed
(`SPEC-020`..`SPEC-029` machine-generated 2026-05-30 against an `exploring`
BUNDLE-003, purged in its 0.5.0 rewrite, 2026-07-13) — BUNDLE-014 fixed "does
the REQ exist," not "was the bundle promoted when it was cited." The
day-to-day exposure is small (a conventional bundle has no `requirements[]`
until promotion, so most `exploring` bundles have nothing to cite), but
nothing structural prevents an agent or auto-dispatch run from jumping ahead.

This remains the "prompts are vibes" gap (`project_prompts_are_vibes`): the
bundle→spec maturity ordering lives only in prose (CLAUDE.md, agent memory,
slash-command instructions) — no code enforces it.

## Solution

Add a maturity-floor check to the same resolution pass that already resolves
bundle/REQ/version (`ResolveSupports` in `pkg/validate/supports_resolution.go`,
or a sibling function invoked alongside it from the same call site in
`cmd/backstop/artifact_validate.go`):

1. After a ref resolves clean (bundle exists, REQ declared, pin logged),
   additionally resolve the cited bundle's `status.maturity` and require it
   be `defined`, `ready`, or `delivered` — the floor `feedback_bundle_workflow`
   / `feedback_artifact_tracks` already state in prose. This needs the
   catalog (or a parallel index) to retain each bundle's maturity, which
   `BuildBundleReqCatalog` currently discards.
2. A ref against a bundle still at `idea` or `exploring` is a new violation —
   e.g. `supports/bundle-not-promoted` — at `error` severity. Reasoning: every
   other violation this same pass emits (`missing-bundle`, `undeclared-req`,
   `version-unlogged`) is `error`; this is the same class of defect (a
   broken promise / workflow-ordering violation, not un-adopted capability),
   so per `feedback_loud_not_blocking` it belongs on the blocking side, not a
   warn, for severity/posture consistency with the rest of the pass.
3. A mandated test should build a fixture: an `exploring` bundle carrying a
   populated `requirements[]` + version log, and a citing spec with a
   validly-pinned ref against it, and confirm the new check fires where today
   it resolves clean.
4. Priority note from the founder (2026-07-14, still holds): this is
   explicitly **LOW / non-urgent** — current agent-driven, one-bundle-at-a-time
   working style doesn't exercise this gap day to day. Worth formalizing so
   the fossil-making mechanism is fully closed, not because it is actively
   burning anyone.

## References

- Discovered 2026-07-14 during the BUNDLE-003 OQ-resolution session, while
  auditing the bundle's own "Out of Scope / Dependencies" note about its own
  legacy SPEC-020..029
- BUNDLE-003 §"Out of Scope / Dependencies" — "Bundle→spec promotion gate
  check — an orthogonal workflow-integrity hole: a spec whose parent bundle is
  not promoted should be a violation but currently is not enforced"
- BUNDLE-003 §Version History, 0.5.0 (2026-07-13) — the purge of the
  never-committed SPEC-020..029 / 9 plans, the concrete incident this issue
  formalizes a fix for
- SPEC-050 (`specs/SPEC-050-requirement-versioning-and-supports-resolution.spec.md`)
  — BUNDLE-014's delivered spec: REQ-001 (both-direction resolution), REQ-002
  (mandatory exact pin), REQ-003 (version-log match), REQ-004 (per-REQ version
  log), REQ-005 (`requirements[]` required at `delivered`+). None of the five
  add a maturity-floor check on the cited bundle — confirmed by re-reading the
  requirement list and the delivered code
- `pkg/validate/supports_resolution.go` — `BuildBundleReqCatalog`,
  `CollectSupportRefs`, `ResolveSupports`; confirmed zero references to
  `maturity`
- `pkg/validate/bundle.go:527-570` (`validateBundleRequirements`) — confirms
  `requirements[]` is required from `defined` onward but never forbidden
  earlier, which is what makes the residual possible
- `.claude/hooks/backstop-agent-guard.sh` — confirmed scope: who-writes-what,
  not artifact-to-artifact workflow ordering
- `project_prompts_are_vibes` (agent memory) — foundational: prompts suggest,
  only executed code constrains; this issue is exactly a load-bearing rule
  that currently lives only in prompts
- `feedback_artifact_tracks`, `feedback_bundle_workflow` (agent memory) — the
  issue→plan / bundle→spec→plan→implementation track rules this check would
  make structurally enforced instead of prose-only
- `feedback_dogfood_rules_as_packs`, `feedback_loud_not_blocking` (agent
  memory) — the enforcement-philosophy constraints the plan should follow

## Version History

- **2026-07-15 — Reconciled against BUNDLE-014/SPEC-050.** The original
  premise ("no resolution at all") was corrected: BUNDLE-014 shipped full
  both-direction resolution (bundle-exists, REQ-declared, pin-in-log), closing
  most of what this issue named. Re-scoped to the one residual BUNDLE-014
  left open — a maturity-floor check on the cited bundle, needed because
  `requirements[]` is required from `defined` onward but never forbidden
  before it. Type/status/complexity unchanged (technical-debt, open,
  contained/known/safe); still LOW priority per the founder's 2026-07-14 note.
