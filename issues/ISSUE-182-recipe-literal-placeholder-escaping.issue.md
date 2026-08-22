---
title: "Recipe Literal Placeholder Escaping"
schema_version: issue/v1

issue:
  id: ISSUE-182
  title: "Recipe Literal Placeholder Escaping"
  type: bug
  status: open
  created: "2026-08-22"

complexity:
  scope: contained
  uncertainty: known
  risk: moderate
---

# Recipe Literal Placeholder Escaping

## Problem

Recipe payload substitution treats every `{{ ... }}` sequence as a Backstop
parameter reference and provides no observed way to emit that sequence
literally. Applying `backstop-ai/backstop-design-system:jekyll-landing-page`
against the Backstop site failed before its first write with:

```text
unresolvable placeholder {{ page.title | default: site.title }}:
no declared param named "page.title | default: site.title"
```

The payload was a valid Jekyll layout. Jekyll/Liquid uses the same delimiter as
Backstop recipes, so a create-only recipe could not scaffold a file containing
ordinary downstream template expressions. The design-system recipe had to
abandon a shared Jekyll layout and emit a complete static `index.html` instead.

This is broader than Jekyll: Liquid, Jinja, Helm, Go templates, GitHub Actions
expressions embedded inside generated shell, and other template systems can
legitimately require delimiter-shaped bytes that belong to the generated file,
not to Backstop.

The current fail-closed behavior for genuinely unresolved Backstop parameters
is correct. The missing capability is a deterministic distinction between
"substitute this declared recipe parameter" and "preserve these bytes for the
downstream consumer."

## Expected Behavior

A recipe author can encode a literal placeholder using a documented escape or
raw mechanism. Applying the recipe emits the intended downstream syntax
byte-for-byte, while an unescaped reference to an undeclared Backstop parameter
continues to fail before any write.

## Acceptance Evidence

- A fixture recipe emits at least one literal Liquid/Jinja-style `{{ ... }}`
  expression byte-for-byte.
- A declared Backstop parameter in the same payload still substitutes.
- An unescaped undeclared Backstop parameter still fails closed before writes.
- Re-applying the recipe remains byte-identical.
- Recipe-authoring documentation defines the escaping contract and its behavior
  across create and transform payloads.

## Existence-in-world Check

Searched the current Backstop Core issue/artifact corpus and repository source
for `unresolvable placeholder`, recipe placeholder escaping, and literal
template delimiters before filing. No existing issue owns this surface.
