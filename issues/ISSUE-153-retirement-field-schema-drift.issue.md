---
title: "Retirement Field Schema Drift"
schema_version: issue/v1

issue:
  id: ISSUE-153
  title: "Retirement Field Schema Drift"
  type: technical-debt
  status: open
  created: "2026-08-16"

complexity:
  scope: cross-cutting
  uncertainty: known
  risk: safe
---

# Retirement Field Schema Drift

## Problem

Five of the seven artifact-type `schema.json` files declare `obsoleted-by` in their
`metadata.properties` block but omit `replaced-by`, even though `pkg/validate/terminal.go`
requires `replaced-by` (not `obsoleted-by`) whenever that artifact's `status` reaches the
`replaced` terminal state. Two of those five omit BOTH fields entirely, despite their
validator enforcing `replaced-by`.

Discovered 2026-08-16 while retiring `PLAN-SPEC-032` — `pkg/validate/plan.go:113`
(`validateRetirementFields`) correctly demanded and validated `replaced-by` on that plan even
though `artifacts/plan/v1/schema.json` never declares the key.

### Survey (verified against source)

| Artifact type | Validator requires `replaced-by`? | Schema declares `obsoleted-by`? | Schema declares `replaced-by`? |
|---|---|---|---|
| issue (`pkg/validate/issue.go:83`) | yes (status `replaced`) | yes (`artifacts/issue/v1/schema.json:23`) | **no** |
| spec (`pkg/validate/spec.go:135`) | yes | yes (`artifacts/spec/v1/schema.json:23`) | **no** |
| plan (`pkg/validate/plan.go:111`) | yes | yes (`artifacts/plan/v1/schema.json:25`) | **no** |
| bundle (`pkg/validate/bundle.go:78`) | yes (maturity `replaced`) | n/a (bundle has no `obsoleted` status) | **no** |
| directive (`pkg/validate/directive.go:62`) | yes (status `replaced`) | n/a (directive has no `obsoleted` status) | **no** |

All five terminal-status enforcement paths route through the same shared helper
(`pkg/validate/terminal.go:56`, `validateTypedRetirementRef` — the comment at
`terminal.go:49` notes this sharing is deliberate: "so replaced-by and obsoleted-by cannot
drift apart"). The Go enforcement logic is internally consistent; the JSON schema
declarations that are supposed to document that same logic are not.

### Root cause is documentation drift, not a live defect (verified against source)

Tracing how these schema declarations are actually consumed shows the omission has **no
functional effect today, for any of the five types** — not only for plans:

1. `pkg/schema/load.go` parses `metadata.properties` into `Schema.MetadataRules`
   (`load.go:82-91`), but grep across the repo shows `MetadataRules` is written in
   `pkg/schema/load.go` and **never read anywhere else** — no validator consults it.
2. `pkg/validate/base.go` (`Base()`, the only consumer of a loaded `*schema.Schema`) checks
   two things only: `sch.RequiredMetadata` key presence (`base.go:25-45`) and
   `sch.RequiredSections` presence (`base.go:47-63`). There is no `additionalProperties`-style
   rejection of undeclared keys anywhere in `pkg/schema` or `pkg/validate` — undeclared
   metadata keys are silently permitted, not rejected.
3. The actual shape/requiredness enforcement for `replaced-by`/`obsoleted-by` lives entirely
   in the hardcoded regex and logic in `pkg/validate/terminal.go`
   (`replacedByRefPattern`, `validateTypedRetirementRef`), independent of what the artifact's
   `schema.json` declares.
4. Plans additionally never reach schema validation for an unrelated reason: `plan/v1/schema.json`
   requires no `schema_version` key, and `ResolveSchemaPath` (`pkg/schema/load.go:150-169`)
   returns an error (never a schema path) when `schema_version` is absent — so plans are
   "schema-less" for a separate reason (no `schema_version` field at all), while issue/spec DO
   carry `schema_version` and DO run `LoadArtifactSchema`, yet the drift is still inert for them
   too, because of point 2 above.
5. `plan/v1/schema.json:11` additionally carries a `metadata.optional` allowlist array
   (`["spec_version", ..., "obsoleted-by"]`) that looks like it restricts allowed keys. It
   doesn't: `rawSchema`/`rawMetadata` in `pkg/schema/load.go:17-48` has no field mapped to
   `json:"optional"`, so this array is parsed as part of the raw JSON and then silently
   dropped — dead JSON, unique to the plan schema (no other artifact schema has this key).

### Impact

None today — no artifact of any of the five types can currently fail validation because of
this drift, since nothing enforces the schema's `properties` list as an allowlist or reads
`MetadataRules`. The impact is entirely latent: the schema files are inaccurate
documentation of what `pkg/validate/terminal.go` actually requires, and if a future change
ever wires `MetadataRules` into real enforcement (e.g. rejecting undeclared metadata keys),
it would immediately and incorrectly reject the exact `replaced-by` field the semantic
validator mandates for `status: replaced` — a self-contradiction between two validation
layers that would only surface once real, well after the schema was already inaccurate.

## Solution

Add `replaced-by` to the `metadata.properties` block of all five schemas (issue, spec, plan,
bundle, directive), using the same pattern already used for `obsoleted-by` where present
(`^(BUNDLE|SPEC|ISSUE|PLAN|DIR)-[0-9]{3}$`) so schema and semantic validator agree. Since
bundle and directive have no `obsoleted` status, only `replaced-by` is needed there.

Out of scope for this issue (separate concerns, noted for context only, not to be bundled
in): the dead `plan/v1/schema.json` `metadata.optional` array, and whether `MetadataRules`
should ever become a real enforcement mechanism. Fixing the field-list drift does not require
resolving either.

## References

- `pkg/validate/terminal.go:10-108` — shared `replacedByRefPattern`, `validateTypedRetirementRef`,
  `extractReplacedBy`
- `pkg/validate/issue.go:83`, `pkg/validate/spec.go:135`, `pkg/validate/plan.go:111`,
  `pkg/validate/bundle.go:78`, `pkg/validate/directive.go:62` — call sites requiring
  `replaced-by`
- `artifacts/issue/v1/schema.json:23`, `artifacts/spec/v1/schema.json:23`,
  `artifacts/plan/v1/schema.json:25` — existing `obsoleted-by` declarations to mirror
- `artifacts/bundle/v1/schema.json`, `artifacts/directive/v1/schema.json` — declare neither
  field
- `artifacts/plan/v1/schema.json:11` — dead `metadata.optional` array
- `pkg/schema/load.go:17-48,82-91` — `rawSchema`/`rawMetadata` structs; `MetadataRules`
  populated but never consumed
- `pkg/validate/base.go:25-63` — the only consumer of a loaded schema; checks presence only,
  no shape/allowlist enforcement
- `plans/PLAN-SPEC-032*.plan.yml` — the retirement that surfaced this drift
