---
name: schema-properties-block-unenforced
description: pkg/schema.MetadataRules (the metadata.properties block in every schema.json) is parsed but never consulted by any validator — only presence (RequiredMetadata) and required_sections are enforced
metadata:
  type: project
---

`pkg/schema/load.go` parses each artifact type's `metadata.properties` block (patterns,
consts, enums) into `Schema.MetadataRules`, but grep confirms `MetadataRules` is written
there and read nowhere else in the repo. `pkg/validate/base.go` — the only consumer of a
loaded `*schema.Schema` — checks only `RequiredMetadata` key presence and `RequiredSections`
presence; there is no `additionalProperties`-style rejection of undeclared metadata keys
anywhere in `pkg/schema` or `pkg/validate`.

**Why this matters for issue authoring:** a schema.json's `properties` block declaring (or
omitting) a field is currently pure documentation — it has zero live validation effect for
ANY artifact type, not just ones missing `schema_version`. Real shape/requiredness
enforcement for things like `replaced-by`/`obsoleted-by` lives in hardcoded Go logic
(`pkg/validate/terminal.go`), independent of the JSON schema file.

**How to apply:** when scoping a "schema says X but validator requires Y" issue, don't assume
the schema drift causes live rejections — verify by tracing whether `MetadataRules` or a
similar allowlist is actually consulted before claiming impact. As of 2026-08-16 it is not,
for any artifact type. This makes such drift `technical-debt` (latent/documentation), not
`bug`, unless something has since started consuming `MetadataRules`. See ISSUE-153.

Also: `artifacts/plan/v1/schema.json` carries a `metadata.optional` allowlist-looking array
that `pkg/schema/load.go`'s `rawMetadata` struct has no field for — it's silently dropped,
dead JSON unique to the plan schema.
