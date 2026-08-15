---
title: "go-standards constructor-injection rule matches field-name substrings, not whole words"
schema_version: issue/v1

issue:
  id: ISSUE-125
  title: "go-standards constructor-injection rule matches field-name substrings, not whole words"
  type: technical-debt
  status: open
  created: "2026-08-15"

complexity:
  scope: isolated
  uncertainty: known
  risk: safe
---

# ISSUE-125: go-standards constructor-injection rule matches field-name substrings, not whole words

## Problem

The `backstop-ai/go-standards` pack's `go.core.constructor-injection` rule (`rule_id: GO-005`,
`severity: WARNING`, non-blocking to the gate) is defined at
`.backstop/packs/backstop-ai/go-standards/rules/core/go-core.yml:49-56`:

```yaml
- id: go.core.constructor-injection
  patterns:
    - pattern: |
        $TYPE{
          $FIELD: $VALUE,
          ...
        }
    - metavariable-regex:
        metavariable: $FIELD
        regex: (?i).*(repo|client|store|db|database|cache|logger|service|gateway|adapter|provider).*
    - pattern-not: |
        $TYPE{}
  message: "Direct struct wiring detected. Prefer constructor-based dependency injection."
```

The rule's intent (per its `rationale`) is to flag struct literals that directly wire a
dependency field (e.g. a field literally named `repo`, `client`, `db`, `logger`) instead of
going through a constructor. But the metavariable-regex matches the substring anywhere inside
`$FIELD`'s name, case-insensitively — not the field name as a whole word. Any struct field
whose name happens to CONTAIN one of those substrings fires the rule, regardless of whether
the field is a dependency at all.

**Concrete false positive, verified at HEAD (2026-08-15):** `pkg/scaffold/idresolver_test.go`
has a test-fixture struct field `isRepo bool` (declared line 16) — a boolean flag meaning "is
this directory a git repo," unrelated to dependency injection. The case-insensitive regex
matches `"repo"` as a substring of `"isRepo"`, firing the rule on every table-driven struct
literal that sets it — 15 occurrences (lines 70, 86, 107, 139, 156, 173, 190, 220, 240, 270,
292, 336, 363, 382, plus the field declaration itself at line 16).

Because the rule is WARNING severity, this does not block the gate — but it is noise: every
hit is a false positive that trains reviewers to skim past `GO-005` findings, which weakens
the rule's real signal on genuine direct-wiring violations.

This is a substring-matching design defect in the rule itself, not a naming problem in
`idresolver_test.go` — `isRepo` is a legitimate, correctly-named boolean field and should not
need to be renamed to satisfy an overly broad pattern.

## Solution

Fix the pack (`backstop-ai/go-standards`), not the consuming code. The
`metavariable-regex` should require a whole-word match on `$FIELD` rather than bare substring
matching — e.g. anchoring each alternative with `\b...\b` (adjusted for case-insensitive
camelCase boundaries, since Go field names are typically camelCase rather than
underscore-delimited, so a literal `\b` may not reliably split `isRepo` from `Repo` — verify
the chosen regex actually fails to match `isRepo`/`dbTimeout`-shaped names before landing it,
not just that it still matches `repo`/`db` alone). Bump the pack's rule version, relock in
this repo (`backstop pack update backstop-ai/go-standards` or equivalent), and confirm the
`isRepo` false positive at `pkg/scaffold/idresolver_test.go:16` no longer fires while a genuine
direct-wiring case (e.g. `Service{repo: someRepo}`) still does.

## References

- `.backstop/packs/backstop-ai/go-standards/rules/core/go-core.yml:49-56` — the rule definition
- `pkg/scaffold/idresolver_test.go:16` — the false-positive field (`isRepo bool`), part of
  PLAN-SPEC-068's implementation
- `pkg/scaffold/idresolver_test.go:70,86,107,139,156,173,190,220,240,270,292,336,363,382` — the
  15 struct-literal occurrences that each trigger a false GO-005 warning
