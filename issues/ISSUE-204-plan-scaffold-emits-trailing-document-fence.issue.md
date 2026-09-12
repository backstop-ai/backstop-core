---
title: "Plan Scaffold Emits Trailing Document Fence"
schema_version: issue/v1

issue:
  id: ISSUE-204
  title: "Plan Scaffold Emits Trailing Document Fence"
  type: bug
  status: open
  created: "2026-09-12"

complexity:
  scope: isolated
  uncertainty: known
  risk: safe
---

# Plan Scaffold Emits Trailing Document Fence

## Problem

`backstop artifact new plan` scaffolds a `.plan.yml` file whose `phases` field lives
**outside** the YAML frontmatter block, because the scaffold closes that block with an
unconditional second `---` before writing `phases: []`. `pkg/scaffold/scaffold.go:261-266`
(current HEAD, `98c5918`-era):

```go
sb.WriteString("---\n")   // line 261, unconditional for every artifact type incl. plan

// Body sections
if artifactType == "plan" {
    // Plan uses YAML phases, not markdown
    sb.WriteString("\nphases: []\n")
} else {
    ...
}
```

This is a real defect with two distinct, verified consequences — one immediate and loud,
one delayed and silent.

### Consequence 1 (verified): every fresh plan scaffold fails validate immediately

`pkg/artifact/parse.go:39-52`'s `Parse()` is a line-based scanner: it captures YAML
frontmatter as everything between the FIRST bare `---` and the NEXT bare `---`, and never
looks at anything after the second fence — by design, for markdown artifact types where
that trailing content is a Markdown body. But a plan is declared "pure YAML... not
human-readable markdown" (`pkg/validate/plan.go:30-31`'s own doc comment), and its
`.plan.yml` extension invites treating the whole file as one YAML document. Because
scaffold.go closes the fence BEFORE emitting `phases`, `Parse()` never sees it, so
`art.Frontmatter["phases"]` is absent and `pkg/validate/plan.go:192-197`'s
`plan/phases-required` check fires on an untouched, zero-content scaffold.

Reproduced against current HEAD (built binary, scratch repo):

```
$ backstop artifact new plan --source ISSUE-001 --slug fence-repro
Created .../plans/PLAN-ISSUE-001-fence-repro.plan.yml (ID: 001)

$ cat plans/PLAN-ISSUE-001-fence-repro.plan.yml
---
plan_id: PLAN-ISSUE-001
spec_id: ISSUE-001
created: "2026-09-12"
status: draft
---

phases: []

$ backstop artifact validate --plan PLAN-ISSUE-001
PLAN-ISSUE-001-fence-repro.plan.yml
  ✗ [plan/phases-required] phases array is missing from frontmatter
✗ Checks failed
```

Every planner agent hits this on the very first save and must hand-merge `phases:` up
into the frontmatter block (deleting the closing fence) to pass validate at all — this is
the forcing function behind Consequence 2.

### Consequence 2 (verified, the reported symptom): the hand-fix habit leaves a vestigial trailing `---`, producing a 2-document YAML stream that passes validate but breaks strict single-document loaders

In `bclabs-brief-builder` (a consumer repo), of 68 committed `.plan.yml` files, 37 carry a
bare `---` as their literal last line with nothing after it, and 31 do not:

```
$ for f in plans/*.plan.yml; do grep -c '^---$' "$f"; done | sort | uniq -c
     31 1
     37 2
```

The 31 "clean" files merge `phases:` into the frontmatter block and delete the closing
fence entirely (single document, opening `---` only — matches `PLAN-ISSUE-075`). The 37
"vestigial" files also merge `phases:` up, but leave the ORIGINAL closing `---` in place,
now sitting at end-of-file with nothing following it (e.g. `PLAN-ISSUE-001`: `---` at line
1, all frontmatter+phases content flattened through line 540, `---` again at line 541,
EOF).

`pkg/artifact/parse.go`'s scanner treats this shape as fully valid: it reads everything
between the first `---` (line 1) and the LAST `---` (line 541) as one YAML blob — nothing
intervenes, so `phases` is captured and `yaml.Unmarshal` succeeds. `backstop artifact
validate --plan` PASSES.

But per YAML's own document-separator semantics, EVERY bare `---` line starts a new
document, including the trailing one. A strict single-document loader sees the trailing
`---` as the start of a genuinely separate (empty) second document and refuses to treat
the file as one document. Reproduced directly against a real committed file:

```
$ python3 -c "
import yaml
content = open('plans/PLAN-ISSUE-001-ask-question-schema-flatten.plan.yml').read()
yaml.safe_load(content)
"
yaml.composer.ComposerError: expected a single document in the stream
  in "<unicode string>", line 2, column 1:
    plan_id: PLAN-ISSUE-001
    ^
but found another document
  in "<unicode string>", line 541, column 1:
    ---
    ^

$ python3 -c "
import yaml
docs = list(yaml.safe_load_all(content))
print(len(docs), docs[1])
"
2 None
```

A reviewer's tooling in the consumer repo hit exactly this: it reported a plan as "empty"
once, with its real task list living in "a second document" — the shape defeats any
consumer-side audit that loads a plan with a single-document parser, silently
(`safe_load_all` → `[plan, None]`, second entry discarded/misread) or loudly
(`safe_load` → `ComposerError`), depending on the loader.

## Scope check: does this affect other scaffolded artifact types?

No — verified by scaffolding one of each type against current HEAD and checking fence
count:

| Type | Extension | Bare `---` count | Whole-file single-doc YAML load |
|---|---|---|---|
| `plan` | `.plan.yml` | 2 (scaffold), 1 or 2 (post-hand-fix) | Fails by design if misused this way — plan IS meant to be one YAML document |
| `capability` | `.capability.yml` | 1 (opens, never closes) | Succeeds — genuinely one document |
| `spec` | `.spec.md` | 2 | `ComposerError` on the whole file, same as any frontmatter+Markdown file |
| `issue` | `.issue.md` | 2 | same |
| `adr` | `.adr.md` | 2 | same |
| `directive` | `.directive.md` | 2 | same |
| `bundle` | `.bundle.md` | 2 | same |

The `.md` types are NOT the same defect: they are frontmatter-plus-Markdown by design, a
bare second `---` is the correct, intentional frontmatter delimiter, and no reasonable
consumer loads the WHOLE file as one YAML document — they split off the frontmatter first
(see `pkg/artifact/parse.go`'s own approach, and this repo's existing "spec frontmatter
needs anchored delimiter split" lesson about naive splitting on the wrong `---`). `plan`
and `capability` are the two types whose declared contract is "the whole file is one YAML
document," and only `plan`'s scaffold breaks that contract — `capability`'s scaffold gets
it right by never emitting a closing fence at all.

## Fix shape (for the planner to derive)

Two independent layers, both worth doing:

1. **Stop emitting the trailing fence in the plan scaffold** (`pkg/scaffold/scaffold.go`).
   Either write `phases: []` INSIDE the frontmatter block (before the closing `---`), or —
   matching `capability`'s already-correct pattern and the 31 clean committed files — never
   emit a closing `---` for `plan` at all, since a plan file is meant to be exactly one YAML
   document from the opening fence to EOF. This removes the Consequence-1 forced failure
   and the resulting hand-edit habit that produces Consequence 2.
2. **Make `validate` catch a multi-document plan/spec file.** Parse (or a dedicated check)
   should count YAML documents in the artifact file and WARN (or error) when there is more
   than one, or when a bare `---` appears with nothing after it — so a vestigial trailing
   fence surfaces at `backstop artifact validate` time instead of silently passing while
   remaining hostile to any standard single-document YAML consumer. This is defense in
   depth independent of fix (1), since existing committed files already carry the shape.
3. **Consumer-side migration.** A `backstop artifact fix` (or a documented one-liner) to
   strip a trailing bare `---` from already-committed plan files, for consumers carrying
   the vestigial pattern today (37 of 68 in `bclabs-brief-builder` as of 2026-09-12).

## Related

- ISSUE-201 (`baseline_comparison` severity-blind) and ISSUE-203 (spec sourcing below
  success-criteria bar) — same consumer (`bclabs-brief-builder`), same pattern: the
  validator is looser than the practice it's meant to gate, and the gap surfaces only when
  something outside the validator's own path (a policy block, a reviewer's tooling) treats
  the artifact more strictly than `backstop artifact validate` does.
- ISSUE-009 (closed) — an earlier plan-scaffold defect (`issue_id` vs `spec_id`) fixed in
  `fb5d2ac`. Same file (`pkg/scaffold/scaffold.go`), same pattern of "scaffold emits a shape
  the validator/schema doesn't actually accept until an agent hand-corrects it."
