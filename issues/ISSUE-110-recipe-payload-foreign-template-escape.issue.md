---
title: "Recipe Payload Foreign Template Escape"
schema_version: issue/v1

issue:
  id: ISSUE-110
  title: "Recipe Payload Foreign Template Escape"
  type: enhancement
  status: open
  created: "2026-07-29"

complexity:
  scope: contained
  uncertainty: known
  risk: safe
---

# Recipe Payload Foreign Template Escape

## Problem

`pkg/recipe/substitute.go` (`Substitute`, lines 31-62) reads EVERY `{{ ... }}` span in a recipe
payload as a param name, with no escape syntax. Its own doc comment states the design plainly:
"the span between the delimiters is trimmed and looked up as a param NAME. There is no expression
parser... a logic construct written inside a placeholder is never evaluated as code — it is
simply a name no param declares, and it fails loud." An undeclared name returns no output at all
(`fmt.Errorf("unresolvable placeholder %s %s %s: no declared param named %q", ...)`) — the apply
hard-fails rather than emitting anything partial.

This is a real gap for any recipe payload that legitimately needs to emit a **foreign** template
syntax verbatim — GitHub Actions' `${{ ... }}`, goreleaser's `{{ .X }}`, Helm, Jinja, or any other
`{{ }}`-delimited DSL a payload file happens to be written in. The shipped workaround, first
proven at scale by `backstop-ai/go-distribution` (now with **public consumers**, since the pack
is intended for `pack add` by any Go CLI project): declare **eight self-emitting pass-through
params**, one per foreign template span the payload needs to survive substitution —
`.Version`, `.ProjectName`, `.Os`, `.Arch`, `.Env.HOMEBREW_TAP_TOKEN`, `github.ref_name`,
`secrets.GITHUB_TOKEN`, `secrets.HOMEBREW_TAP_TOKEN`. Each param's **name** is the template's
inner text and its **default** is that same text wrapped back in delimiters, e.g.:

```
params:
  - name: ".Version"
    required: false
    default: "{{ .Version }}"
```

This works because substituted values are NEVER RESCANNED (`substitute.go:22-23` comment: "a
value that itself contains the delimiters cannot smuggle a second pass in") — `Substitute` looks
up `.Version`, emits the literal string `{{ .Version }}`, and moves on without re-parsing that
output. It is honest and it is functioning in production, but it is a workaround for a missing
first-class escape, not a feature: it is also a mild hazard, since a caller passing
`--param .Version=anything` would rewrite the emitted template rather than leaving it alone.

**New measured finding, not previously recorded**: the substituter does not distinguish comments
from code. It scans the raw template string byte-range for `{{`/`}}` regardless of surrounding
YAML comment syntax (`#`) or any other context — there is no comment-aware skip. A payload
comment that merely **mentions** a template in prose, with real `{{ }}` delimiters, hard-fails the
apply with the same "unresolvable placeholder" error as a real unresolved substitution. This cost
a real debugging cycle during the `go-distribution` recipe payload's authoring (the release.yml
payload's manual cross-read note, which needs to say "cross-read every `.goreleaser.yml`
`.Env.<NAME>` reference against this file's exports"). The workaround now in place
(`recipes/go-release/payload/release.yml:16-20` in `backstop-ai/go-distribution`) writes
`.Env.<NAME>` **delimiter-less** in the prose, with an explanatory note that the delimiters were
dropped on purpose because the file is a recipe payload and any `{{ }}` span — even inside a
comment describing one — has to be declared as a param to survive the apply.

Any recipe that wants to document its own foreign templates in a comment — which is exactly the
kind of comment a hard-won, footgun-avoiding payload like this one needs — hits this same gap.

**Distinct from ISSUE-081.** ISSUE-081 ("Recipe Authoring Surface Underspecified") covers a
different, already-open cluster: the merge op's `fragment:` field having no pinned form, no CLI
param input for `recipe apply`, and the `insert` op's placement semantics. This issue is a
**language-level escape gap** in the substitution grammar itself, with its own shipped workaround
distinct from any of ISSUE-081's three gaps. It is filed separately rather than folded in, but the
founder may prefer to consolidate it as ISSUE-081's "Gap 4" instead — recorded here explicitly so
that consolidation, if wanted, is a deliberate choice rather than a rediscovery.

## Solution

Not resolved here — recording fix directions for a future spec/plan to pick between:

1. **A first-class escape syntax** in the substitution grammar — e.g. a triple-brace form
   (`{{{ literal }}}`) or a backslash-escape (`\{{ literal }}`) that `Substitute` recognizes and
   emits verbatim without a param lookup. Removes the need for pass-through params entirely for
   the common case; requires deciding and documenting one canonical escape form.
2. **Comment-aware skipping.** Have `Substitute` recognize YAML/common comment prefixes and skip
   `{{ }}` spans that appear only in prose, so documentation about a template doesn't require the
   same declaration machinery as the template itself. Narrower fix, addresses only the
   newly-measured comment-specific finding, and is language-syntax-aware in a way the current
   substituter deliberately is not (see its "deliberately NOT Turing-complete" framing) — would
   need care not to reintroduce ambiguity about what counts as a comment across payload file
   types.
3. **Do nothing, document the pass-through-param idiom as the sanctioned pattern.** Lowest-cost
   option; leaves the hazard (a param name collision with a real declared param) and the
   footgun (documenting templates safely requires knowing to drop delimiters) as permanent
   authoring surface.

## References

- `pkg/recipe/substitute.go:31-62` — `Substitute`, the mechanism with no escape syntax.
- `issues/ISSUE-081-recipe-authoring-surface-underspecified.issue.md` — the adjacent, open,
  differently-shaped cluster of recipe-authoring gaps; cross-referenced, not merged.
- `bundles/BUNDLE-015-pack-scaffolding-recipes.bundle.md` (SPEC-054) — the recipe apply/manifest
  machinery this substitution behavior is part of.
- `plans/PLAN-ISSUE-101-go-distribution-pack.plan.yml` — TASK-016 (naming this issue as a
  deliberate deferral), and the notes' "RECIPES ARE FULLY PARAMETERIZED, and the substituter has
  NO ESCAPE" section (the original eight-pass-through-param derivation) plus TASK-008's payload
  authoring notes (the newly-measured comment-delimiter finding and its workaround).
- `backstop-ai/go-distribution` pack (`~/src/projects/backstop-go-distribution-pack`) —
  `recipes/go-release/payload/release.yml:16-20`, the delimiter-less-prose workaround in
  production.
