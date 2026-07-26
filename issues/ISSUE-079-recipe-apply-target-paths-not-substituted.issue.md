---
title: "`recipe apply` Writes Recipe-Declared Sites Under Their LITERAL `{{ }}` Path, Anchor, and Snippet Text — Silent Wrong Output"
schema_version: issue/v1

issue:
  id: ISSUE-079
  title: "`recipe apply` Writes Recipe-Declared Sites Under Their LITERAL `{{ }}` Path, Anchor, and Snippet Text — Silent Wrong Output"
  type: bug
  status: open
  created: "2026-07-25"

complexity:
  scope: contained
  uncertainty: known
  risk: critical
---

# `recipe apply` Writes Recipe-Declared Sites Under Their LITERAL `{{ }}` Path, Anchor, and Snippet Text — Silent Wrong Output

## Problem

`backstop recipe apply` runs every op's declared **payload** and **fragment** content
through `Substitute` (`pkg/recipe/substitute.go`), so a `{{ param }}` inside file
CONTENT resolves correctly. But the **site strings that locate WHERE an op writes** —
`Op.Target` (create/merge/transform/insert), `Op.Anchor` and `Op.Snippet` (insert) —
are never passed through `Substitute` at all. When a recipe templates one of those
fields, the applier writes to, reads from, or splices in the LITERAL text containing
the still-unresolved `{{ ... }}` placeholder, and **exits 0 reporting success**. This
is silent wrong output, not a crash: nothing in the apply path notices that a `{{`
reached the filesystem.

### Live repro (first real non-fixture use of `recipe apply`, 2026-07-25)

In a scratch consumer project with a local pack installed via `pack add`, applying a
recipe whose `starter` op declares `target: "{{ config_dir }}/service.json"` (param
`config_dir`, default `"config"`):

```
$ backstop recipe apply demo/live-pack:starter@1.0.0
applied recipe
$ echo $?
0
$ find . -maxdepth 2
./{{ config_dir }}/service.json     # literal braces in the path — never resolved
```

`config_dir` is a declared param with a default; no override was needed for
substitution to apply. The command never mentions the literal path, never warns, and
exits 0. Payload/fragment CONTENT substitution worked correctly in the SAME run —
`{{ service_name }}` resolved inside both the create payload and the merge fragment,
and the merge's deep-merge sibling-preservation behavior was correct.

### Root cause — every site field bypasses `Substitute`; only two content fields go through it

`pkg/recipe/apply.go` calls `Substitute` from exactly two places:

- `renderPayload` (apply.go:410-426, called from `applyCreate` at apply.go:248) —
  substitutes the create op's PAYLOAD content.
- `applyMerge` (apply.go:639) — substitutes the merge op's FRAGMENT content.

Every other field that carries `{{ }}` in the shared fixture corpus is read and used
RAW:

- `applyCreate` (apply.go:224, 230, 252, 256, 258) resolves and writes
  `op.Target` directly — `resolveUnder(opts.ProjectRoot, op.Target)`,
  `targetExists(op.Target, target)`, `writeRendered(op.Target, target, rendered)` —
  with no substitution step anywhere in between.
- `applyMerge` (apply.go:618, 644, 646, 654, 656) resolves, reads, and writes
  `op.Target` the same way — raw, even though the fragment it merges INTO that
  target is correctly substituted three lines later.
- `applyInsert` (apply.go:446-477) resolves `site.target` raw (apply.go:454) AND
  splices `op.Snippet` verbatim into the file (apply.go:470,
  `content[:spliceAt] + op.Snippet + content[spliceAt:]`) with no `Substitute` call —
  so a templated snippet, not just a templated target, would also land in the
  consumer file as literal `{{ }}` text. `site.anchor` (apply.go:446-452) is
  likewise matched raw against file content.
- `applyTransform` (apply.go:496-526) resolves `site.target` raw (apply.go:510) and
  hands it straight to the injected `Dispatch`.
- `siteFor` (apply.go:551-574), the function that produces `site.target`/`site.anchor`
  for both insert and transform in either apply mode, never substitutes either half —
  a caller-supplied `InjectionSites` override in `sdlc-mediated` mode has the same gap.

The fixture corpus already demonstrates the full blast radius. The committed demo
recipe `pkg/recipe/testdata/packs/demo-org/demo-pack/recipes/starter/recipe.yml`
templates `target:` on THREE of its four executed ops (`create-config`,
`merge-settings`, `rename-config-key` all use
`target: "{{ config_dir }}/..."`) and templates `Op.Snippet` on its insert op
(`register-app`: `snippet: '"{{ app_name }}"'`) — every op family this recipe
exercises has a templated site, content, or both.

### Why every test missed it

No mandated test for CLM-012/013/014 (the substitution claims, SPEC-054 REQ-002)
ever calls `Apply` with a templated site field:

- `TestSubstitute_ResolvesDeclaredParam`, `TestSubstitute_NotTuringComplete_NoLogicEvaluated`,
  and `TestSubstitute_UndeclaredParamFailsLoud` (mandated by CLM-012/013/014,
  `specs/SPEC-054-recipe-apply-and-manifest.spec.md:298-312`) all exercise
  `Substitute()` directly in `substitute_test.go` — never through `Apply` end to end,
  and never against `Op.Target`/`Op.Anchor`/`Op.Snippet`.
- The one committed fixture recipe that templates a target
  (`pkg/recipe/testdata/packs/demo-org/demo-pack/recipes/starter/recipe.yml`, e.g.
  `target: "{{ config_dir }}/app.config.json"`) is consumed only by resolution/parse
  tests (recipe loading, reference resolution) — never actually run through `Apply`.
  Every `apply_*_test.go` file (`apply_core_test.go`, `apply_merge_test.go`,
  `apply_modes_test.go`, `apply_multi_test.go`, `apply_opfamily_test.go`,
  `apply_regenerate_test.go`, `apply_templating_test.go`, `apply_transform_test.go`,
  `apply_injection_limit_test.go`) builds its own ops inline with literal,
  non-templated `Target`/`Anchor`/`Snippet` strings, so no test constructs the
  combination that would have failed: a templated site field run through `Apply`.

### Spec gap contributing to the miss

`specs/SPEC-054-recipe-apply-and-manifest.spec.md` §"Substitution (REQ-002)"
(around line 873) states `{{ param }}` is "pure value interpolation resolving
declared params only" and CLM-012's text says only "A placeholder is substituted
from the declared params" — neither pins the substitution SCOPE, i.e. which `Op`
fields actually get resolved (payload/fragment content vs. target/anchor/snippet/rule
site fields). The spec's own fixture corpus (the `starter` demo recipe) implies
targets and snippets are in scope; the implementation disagrees. A substitution-scope
table, analogous to the existing kind/reference-resolution/manifest-validation
matrices in the same spec, would have made this gap visible at spec-review time
instead of at first real dogfood use.

### Expected

Every `Op` field that a recipe author can template — `Target`, `Anchor`, `Snippet`,
and (for symmetry, since `Rule` is also a declared path) `Rule` — is resolved through
`Substitute(field, effectiveParams(...))` BEFORE it is used to resolve a filesystem
path, match an anchor, or splice content, for every op family and both apply modes
(including an `sdlc-mediated` `InjectionSites` override). An unresolvable placeholder
in a site field fails loud with the same "unresolvable placeholder" error `Substitute`
already produces for payload/fragment content — never a literal `{{` reaching the
filesystem, and never a silent exit 0.

## References

- `pkg/recipe/apply.go:224-258` — `applyCreate`; resolves/writes `op.Target` raw
- `pkg/recipe/apply.go:410-426` — `renderPayload`; the ONE correctly-substituted
  content path (create payload)
- `pkg/recipe/apply.go:445-478` — `applyInsert`; `site.target` raw (line 454),
  `site.anchor` raw match (lines 450-452, 464-466), `op.Snippet` spliced raw
  (line 470) — a content field with the same gap as target
- `pkg/recipe/apply.go:496-526` — `applyTransform`; `site.target` raw (line 510)
- `pkg/recipe/apply.go:540-574` — `siteFor`; produces target/anchor for insert and
  transform in both apply modes, substitutes neither half, including the
  `sdlc-mediated` `InjectionSites` override
- `pkg/recipe/apply.go:617-659` — `applyMerge`; `op.Target` raw (lines 618, 644,
  646, 654, 656), fragment content correctly substituted at line 639
- `pkg/recipe/substitute.go:31` — `Substitute(template string, params map[string]string) (string, error)`,
  the function every site field needs to be routed through
- `pkg/recipe/testdata/packs/demo-org/demo-pack/recipes/starter/recipe.yml` — the
  committed fixture recipe that already templates `target:` on 3 ops and `snippet:`
  on 1, but is exercised only by resolution/parse tests, never by `Apply`
- `specs/SPEC-054-recipe-apply-and-manifest.spec.md:298-312` — CLM-012/013/014 and
  their mandated `TestSubstitute_*` tests, all `Substitute()`-direct, none through
  `Apply`
- `specs/SPEC-054-recipe-apply-and-manifest.spec.md:873-881` — "Substitution
  (REQ-002)" section; states scope as "declared params only," never pins WHICH `Op`
  fields are in scope
- Cross-ref: a parallel issue filed the same live-dogfood session covers a separate
  silent-error defect surfaced in the same run (SPEC-054 v1.2.1)
