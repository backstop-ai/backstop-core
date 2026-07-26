---
title: "Recipe Authoring Surface Underspecified: Merge Fragment Form + No CLI Param Input"
schema_version: issue/v1

issue:
  id: ISSUE-081
  title: "Recipe Authoring Surface Underspecified: Merge Fragment Form + No CLI Param Input"
  type: technical-debt
  status: open
  created: "2026-07-25"

complexity:
  scope: contained
  uncertainty: known
  risk: moderate
---

# Recipe Authoring Surface Underspecified: Merge Fragment Form + No CLI Param Input

## Problem

SPEC-054 v1.2.1 delivered the recipe apply mechanism and manifest declaration, but the first
live dogfood of the authoring surface (2026-07-25) surfaced two coupled gaps: the `merge` op's
`fragment:` field has no pinned form, and `backstop recipe apply` has no way to supply params at
the CLI. Both were invisible to the existing test suite because the only fixture that exercises
the ambiguity is parsed but never applied, and every applied fixture already satisfies its
required params from declared defaults.

### Gap 1 — `fragment:` form is ambiguous across three artifacts, and two of them disagree

Three places declare or consume a merge op's fragment, and they do not agree with each other:

1. **The applier treats `Fragment` as a file path.** `pkg/recipe/apply.go`'s `applyMerge`
   resolves it under the recipe directory and reads it from disk:
   ```
   fragmentPath, err := resolveUnder(resolved.Dir, op.Fragment)   // apply.go:628
   rawFragment, err := os.ReadFile(fragmentPath)                  // apply.go:632
   ```
2. **The spec pins nothing.** SPEC-054's `Op` contract (`pkg/recipe/manifest.go`) declares only
   `Fragment string` with the note "One declared operation" — no wording anywhere in the spec
   says fragment is a path, an inline value, or either. REQ-002/CLM-005..008 say a merge op
   "deep-merges a declared fragment into a structured file" without saying how the fragment
   arrives.
3. **The committed fixture corpus declares it inline.** The `merge-settings` op in
   `pkg/recipe/testdata/packs/demo-org/demo-pack/recipes/starter/recipe.yml` uses a YAML block
   scalar with literal JSON content, not a path:
   ```yaml
   - id: merge-settings
     kind: merge
     target: "{{ config_dir }}/settings.json"
     format: json
     fragment: |
       {"adopted_by": "{{ app_name }}"}
   ```

That fixture is only ever fed to `ParseRecipeManifest` in tests — never to `Apply` — so the
contradiction between (1) and (3) survived every test in the suite. Applying it for real
produces exactly the failure the path-only implementation predicts:

```
open .../recipes/starter/{"owner": ...}: no such file or directory
```

(That the error is swallowed rather than surfaced to the operator is a separate, already-filed
defect — see References.) There is no committed recipe anywhere in the corpus that declares a
merge fragment as a real path and gets applied end to end either, so neither form is actually
proven through `Apply`.

### Gap 2 — no way to supply params at the CLI

`backstop recipe apply <pack:recipe@version>` (`cmd/backstop/recipe_apply.go`) takes exactly one
positional argument (`Args: cobra.ExactArgs(1)`) and no flags. `runRecipeApply` builds
`ApplyOptions{Mode: recipe.ModeDirect, ...}` without ever setting `Params`, so
`pkg/recipe/apply.go`'s `effectiveParams` (apply.go:900-913) has only the manifest's own
declared defaults to work with:

```go
for _, spec := range manifest.Params {
    if !spec.Required || spec.Default != "" {
        params[spec.Name] = spec.Default
    }
}
```

A param that is `required: true` with no `default:` is never added to the map at all — there is
no supplied-value path for it to fall into. SPEC-054 CLM-015 ("Direct mode self-applies the
recipe's ops from the recipe-declared defaults/params") is defensible as written for a recipe
whose required params all carry defaults, but it does not cover the case a param schema
explicitly supports: `ParamSpec.Required == true` with an empty `Default`. Any recipe that
declares such a param — e.g. the fixture's own `app_name` (`required: true`, no default) — is
unusable through the CLI: direct-mode apply has no way to provide the value the recipe demands,
and the resulting `{{ app_name }}` substitution failure is not distinguishable at the call site
from a genuinely undeclared placeholder.

### Why this matters together

Both gaps sit on the same surface a pack author or an operator hits on the very first realistic
recipe: a merge op that adds one settings key, and a required param naming what gets adopted.
Neither gap is exotic — they are the default shape of a useful scaffolding recipe — yet neither
is provable end to end today, and the fixture corpus itself only demonstrates the gap rather than
a working example.

## Solution

Two independent positions are needed; either can land first.

**Gap 1 — pick fragment's form and make the fixture and spec agree with the applier.**
- Option A (path-only): keep `applyMerge`'s current behavior, fix the `merge-settings` op in the
  committed fixture to point at a real fragment file under the recipe directory, and add a
  sentence to SPEC-054's `Op` contract note pinning `Fragment` as "a recipe-relative path,
  resolved under the recipe directory, read from disk" (mirroring how `Payload`/`Rule` are
  already documented as paths).
- Option B (support both): sniff or explicitly flag inline-vs-path (e.g. treat a value containing
  no path separator and starting with `{`/looking like a document as inline; or add a distinct
  field, e.g. `fragment_inline:`), update the spec's contract note to describe both forms, and add
  a captured-fixture regression test that exercises the inline path through `Apply` (per
  [[feedback_fixtures_from_real_output]] — the regression fixture must be captured from real
  applied output, not written to fit the fix).

Either option needs: the fixture corpus and the spec's `Op` contract note to describe the SAME
form the applier implements, and at least one fixture that is actually run through `Apply` (not
just `ParseRecipeManifest`) covering whichever form(s) are supported.

**Gap 2 — give direct-mode apply a way to supply required params.**
- Add `--param k=v` (repeatable) and/or `--params-file <path>` flags to
  `newRecipeApplyCommand()`, threaded into `ApplyOptions.Params` in `runRecipeApply`, OR
- Document that `backstop recipe apply` is defaults-only by design and that a required
  no-default param implies SDLC-mediated use (never direct CLI apply); if that's the position,
  `pkg/recipe.ParseRecipeManifest` should reject a `required: true` param with an empty `default:`
  on a `scaffolding`-kind recipe at `pack check` time, so the mismatch is caught at pack-publish
  time rather than surfacing as an unusable CLI invocation downstream.

Whichever position is taken for Gap 2, it determines whether the fixture's own `app_name` param
(required, no default) is a valid example of a CLI-applicable recipe or needs a default added to
stay a clean demo of direct mode.

## References

- `pkg/recipe/apply.go:617-659` (`applyMerge`) — path-only fragment resolution
- `pkg/recipe/apply.go:900-913` (`effectiveParams`) — required-no-default params are never added
  to the params map; `opts.Params` is the only other source and the CLI never populates it
- `pkg/recipe/manifest.go` — `Op.Fragment string`, `ParamSpec{Name, Required, Default}`
- `pkg/recipe/testdata/packs/demo-org/demo-pack/recipes/starter/recipe.yml` — the
  `merge-settings` op declaring `fragment:` as an inline YAML block scalar; the `app_name` param
  (`required: true`, no default)
- `cmd/backstop/recipe_apply.go` — `newRecipeApplyCommand()` (`Args: cobra.ExactArgs(1)`, no
  flags), `runRecipeApply()` (`ApplyOptions.Params` never set)
- `specs/SPEC-054-recipe-apply-and-manifest.spec.md` — REQ-002/CLM-005..008 (merge fragment,
  form unpinned), REQ-003/CLM-015 (direct mode self-applies from defaults, silent on
  required-no-default), `Op` contract note (no fragment-form pin)
- Filed in parallel from the same 2026-07-25 live dogfood session: the target-substitution and
  silent-apply-error issues (issue-author-target-subst, issue-author-silent-apply); a spec
  reshape from any of these should route through `/spec` per the ISSUE-078 pattern
