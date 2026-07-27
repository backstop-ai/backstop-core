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
required params from declared defaults. A follow-up full recipe-scenario sweep (2026-07-26 — all
kinds/ops authored in a real local pack, applied via the real CLI in a scratch consumer) surfaced
a third, same-shaped gap on the `insert` op's placement semantics (Gap 3, below).

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

### Gap 3 — insert placement semantics are unpinned (found 2026-07-26)

Live-reproduced in the 2026-07-26 full recipe-scenario sweep (all kinds/ops authored in a real
local pack, applied via the real CLI in a scratch consumer): an `insert` op with anchor
`"registrations": [` and snippet `    "live-entry",` produced

```
"registrations": [    "live-entry",
```

— the snippet spliced INLINE onto the same line, immediately after the anchor text, almost
certainly not what a pack author reaching for "insert a new array entry" intends (the working
example needs a newline between the anchor and the snippet, or the snippet needs to carry its own
leading newline, either of which the current contract leaves the author to discover by trial).
`applyInsert` (`pkg/recipe/apply.go:445-478`) does exactly what it currently says it does:

```go
spliceAt := anchorAt + len(site.anchor)
updated := content[:spliceAt] + op.Snippet + content[spliceAt:]
```

`spliceAt` is the byte offset immediately after the anchor's last matched character — no line
boundary is considered. SPEC-054 doesn't pin an answer either way: the `Op` contract note
(`specs/SPEC-054-recipe-apply-and-manifest.spec.md:635`) and REQ text (line 295, "An insert op
inserts the declared snippet at the declared anchor") both describe WHAT gets spliced, never
WHERE relative to line boundaries — inline-after-anchor and new-line-after-anchor-line are equally
consistent readings of the current wording. The mandated test for this claim
(`TestApply_InsertOp_AtAnchor`, CLM-011) exercises only that the snippet appears somewhere at the
anchor, not which of the two placements it lands in, so this ambiguity is invisible to the test
suite the same way Gap 1's fragment-form ambiguity was.

For completeness, the same sweep also confirmed (not itself a defect, but worth recording so it
isn't rediscovered as one): SDLC-mediated mode is currently CLI-unreachable.
`cmd/backstop/recipe_apply.go:142` hardcodes `Mode: recipe.ModeDirect` on every `ApplyOptions` it
builds, and `runRecipeApply` never sets `InjectionSites`. `siteFor` (`pkg/recipe/apply.go:551-559`)
only consults `opts.InjectionSites` when `opts.Mode == ModeSDLCMediated`, so a CLI invocation can
never supply a site override — every insert/transform site through `backstop recipe apply` is the
recipe's own declared `target`/`anchor`, full stop. This is plausibly by design (per REQ-003,
injection sites are meant to come from a plan/agent driving SDLC-mediated apply, not the bare
CLI), but it is currently an accident of what the CLI happens not to expose rather than a
documented position — it needs an explicit stated position (e.g. "direct-mode-only CLI; SDLC-
mediated mode is a library-only entry point for plan-driven callers") rather than silence.

### Why this matters together

All three gaps sit on the same surface a pack author or an operator hits on the very first
realistic recipe: a merge op that adds one settings key, an insert op that adds one array entry,
and a required param naming what gets adopted. None of the gaps is exotic — they are the default
shape of a useful scaffolding recipe — yet none is provable end to end today, and the fixture
corpus itself only demonstrates the gaps rather than a working example.

## Solution

Three independent positions are needed; any can land first.

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

**Gap 3 — pin insert's line semantics and state SDLC-mediated mode's CLI reachability.**
- Pin one of the two placements in SPEC-054's `Op` contract note and `applyInsert`:
  - Inline-after-anchor (current behavior): keep `applyInsert` as-is, add a sentence to the
    contract note stating the splice point is the byte offset immediately after the anchor's last
    matched character with no line handling, and put the burden on the recipe author to include
    leading whitespace/newlines in `Snippet` when a new line is wanted (the sweep's own
    `"    \"live-entry\",\n"`-with-leading-newline would then be the correct authoring pattern,
    not a bug to fix in `apply.go`).
  - New-line-after-anchor-line (what the sweep's author intuitively expected): change
    `applyInsert` to splice after the END of the anchor's line (advance `spliceAt` to the next
    `\n` after `anchorAt`, or insert `"\n" + op.Snippet` when the anchor is mid-line), update the
    contract note to state the snippet always lands on its own line following the anchor's line,
    and add a captured-fixture regression test (per [[feedback_fixtures_from_real_output]]) that
    would have caught the sweep's inline-splice result.
  Whichever is chosen, extend `TestApply_InsertOp_AtAnchor` (or add a sibling) to assert on the
  EXACT resulting bytes around the splice point, not just that the snippet appears somewhere near
  the anchor — the current test's weak assertion is why this shipped unnoticed.
- Add an explicit sentence to SPEC-054 (or a follow-on ADR/spec note) stating whether
  `backstop recipe apply` is scoped to direct mode only by design, or whether SDLC-mediated mode
  is meant to be CLI-reachable and simply hasn't been wired yet
  (`cmd/backstop/recipe_apply.go:142` hardcodes `ModeDirect`). If direct-mode-only is the position,
  no code changes are required — just the stated position, so the gap isn't rediscovered as a bug.

Whichever position is taken for Gap 2, it determines whether the fixture's own `app_name` param
(required, no default) is a valid example of a CLI-applicable recipe or needs a default added to
stay a clean demo of direct mode.

## Decision (2026-07-27)

Founder-ratified positions for Gap 1 and Gap 2; Gap 3 remains open (see below).

**Gap 1 — DECIDED: Option A, path-only.** `fragment:` is canon as a **recipe-directory-relative
file path**, matching `applyMerge`'s existing behavior (`pkg/recipe/apply.go:628,632`) and how
`Payload`/`Rule` are already documented as paths. An inline block (as the committed
`merge-settings` fixture currently declares) is a **parse error with a clear message**, not a
silently-accepted alternate form — this dissolves the `{{ }}`-in-fragment substitution-timing
ambiguity Gap 1 raised, since inline fragments no longer exist as a form to disambiguate.
Follow-through: fix the `merge-settings` op in
`pkg/recipe/testdata/packs/demo-org/demo-pack/recipes/starter/recipe.yml` to point at a real
fragment file under the recipe directory; add the path-only sentence to SPEC-054's `Op` contract
note; add a captured-fixture regression test that runs the path form through `Apply` (not just
`ParseRecipeManifest`), per [[feedback_fixtures_from_real_output]]; and add the parse-time
rejection for an inline (non-path) `fragment:` value with a clear error message.

**Gap 2 — DECIDED: CLI gains `--param key=value`.** `newRecipeApplyCommand()` gets a repeatable
`--param key=value` flag, threaded into `ApplyOptions.Params` in `runRecipeApply`. Declared
defaults still fill any optional param left unsupplied; a `required: true` param with no default
becomes CLI-reachable via the flag instead of being permanently unusable through direct-mode apply.
This resolves the fixture's own `app_name` param (`required: true`, no default) as a valid example
of a CLI-applicable recipe once supplied via `--param app_name=...` — no fixture change needed to
give it a default.

**Gap 3 — remains open.** No position taken on insert placement semantics
(inline-after-anchor vs. new-line-after-anchor-line) or on stating SDLC-mediated mode's CLI
reachability. This is the issue's remaining open question.

Ready for `issue → plan` on Gaps 1 and 2.

## References

- `pkg/recipe/apply.go:617-659` (`applyMerge`) — path-only fragment resolution
- `pkg/recipe/apply.go:900-913` (`effectiveParams`) — required-no-default params are never added
  to the params map; `opts.Params` is the only other source and the CLI never populates it
- `pkg/recipe/manifest.go` — `Op.Fragment string`, `ParamSpec{Name, Required, Default}`
- `pkg/recipe/testdata/packs/demo-org/demo-pack/recipes/starter/recipe.yml` — the
  `merge-settings` op declaring `fragment:` as an inline YAML block scalar; the `app_name` param
  (`required: true`, no default)
- `cmd/backstop/recipe_apply.go` — `newRecipeApplyCommand()` (`Args: cobra.ExactArgs(1)`, no
  flags), `runRecipeApply()` (`ApplyOptions.Params` never set); line 142 hardcodes
  `Mode: recipe.ModeDirect` on every built `ApplyOptions` (Gap 3, SDLC-mediated unreachable)
- `pkg/recipe/apply.go:445-478` (`applyInsert`) — `spliceAt := anchorAt + len(site.anchor)` splices
  immediately after the anchor's matched text with no line-boundary handling (Gap 3)
- `pkg/recipe/apply.go:551-559` (`siteFor`) — only consults `opts.InjectionSites` when
  `opts.Mode == ModeSDLCMediated`, which the CLI never sets (Gap 3)
- `specs/SPEC-054-recipe-apply-and-manifest.spec.md:295,635` — insert's REQ text and `Op` contract
  note describe WHAT gets spliced, never WHERE relative to line boundaries (Gap 3); line 587/
  `TestApply_InsertOp_AtAnchor` (CLM-011) is the mandated test, currently weak on exact placement
- `specs/SPEC-054-recipe-apply-and-manifest.spec.md` — REQ-002/CLM-005..008 (merge fragment,
  form unpinned), REQ-003/CLM-015 (direct mode self-applies from defaults, silent on
  required-no-default), `Op` contract note (no fragment-form pin)
- Filed in parallel from the same 2026-07-25 live dogfood session: the target-substitution and
  silent-apply-error issues (issue-author-target-subst, issue-author-silent-apply); a spec
  reshape from any of these should route through `/spec` per the ISSUE-078 pattern
