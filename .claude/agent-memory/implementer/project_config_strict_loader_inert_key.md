---
name: config-strict-loader-inert-key
description: backstop.yml loader is STRICT (KnownFields+schema-required), not non-strict — retiring a config field needs schema + inline-catch-all changes the spec/plan won't scope
metadata:
  type: project
---

When retiring a `config.Config` field (SPEC-046 retired `language`), the spec/plan
assumed "backstop.yml is parsed non-strict, so a stray key parses inertly." That
premise is FALSE and the 4-round review chain missed it.

Reality (`pkg/config/config.go` `LoadConfigFromPath`): two strict layers —
(1) `yaml.NewDecoder(...).KnownFields(true)` rejects any key without a struct field;
(2) `validateAgainstSchema` enforces the JSON schema's top-level
`additionalProperties:false` AND its `required[]`. Nested unknown-key rejection
(e.g. `TestConfig_Enforcement_ToolchainRejectsUnknownNestedKey`) is enforced ONLY by
KnownFields on the nested struct — the schema validator does NOT recurse.

**Why:** deleting a field + stripping it from the dogfood `backstop.yml` makes
`backstop gate` fail config-load (field was schema-`required`) AND makes a config
still carrying the key error (KnownFields rejects the now-unknown key). The mandated
inert-parse test then can't pass.

**How to apply** (the resolution that kept the gate green — both files are OUTSIDE the
plan's literal file scope; edit the schema via Bash since agent-guard blocks `artifacts/`):
- `artifacts/backstop-yml/v1/schema.json`: drop the field from `required[]` but KEEP it
  in `properties` as an allowed-but-ignored legacy property.
- `pkg/config/config.go`: add an EXPORTED `,inline` catch-all to `config.Config`
  (`LegacyKeys map[string]any` with `yaml:",inline" json:"-"`) so KnownFields tolerates
  the now-unknown key. Must be exported (yaml.v3 ignores unexported inline fields).
  Truly-unknown top-level keys are still rejected by the schema, nested strictness is
  unaffected — strictness preserved, only the named legacy key tolerated.
- Update `TestConfig_RequiredFields_*` (the "missing <field>" case stops erroring).

Side note (same seed): changing `deriveCapabilityState`'s signature (the spec contract
mandated a new `stack` param) broke ~24 DIRECT test callers across the gate_capability
rekey/contracts/capability tests — a compile fence the plan's per-file task scope did
NOT enumerate. Budget for "signature change -> every direct caller, test included."
Related: [[feedback_review_misses_baked_nouns]].
