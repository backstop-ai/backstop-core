---
title: "Recipe Apply And Manifest"
number: SPEC-054
created: "2026-07-21"
updated: "2026-07-27"
status: implemented
schema_version: spec/v1
spec_version: 1.5.0

implementation:
  summary: >
    Seed 1+2 of BUNDLE-015 (pack-scaffolding-recipes): the GENERIC recipe APPLY
    mechanism and the MANIFEST DECLARATION it resolves against — one coupled surface,
    the piece `backstop init` (BUNDLE-003 DD-12) is blocked on. A recipe is a
    per-recipe DIRECTORY (recipe.yml + template payload + transform-rule files)
    indexed from a lightweight `pack.yml` `recipes:` block (stable recipe-id ->
    directory), NOT overloaded onto `content.scaffolds` or `pack scaffold`. A NEW
    `pkg/recipe` package carries: (1) recipe.yml manifest parse + structural
    validation (ops, kind, param schema, target paths, transform rules, enforcement
    pairing declaration, version, optional compat/variants); (2) a GENERIC applier
    that resolves a recipe by `<pack>:<recipe>@<recipe_version>` and runs its declared
    ORDERED ops — `create` / `merge` (json/yaml/toml/.env) / `transform` / `insert` —
    with declarative `{{ }}` substitution that is explicitly NOT Turing-complete, and
    RESERVES a fifth `step` op as an opaque sequencing seam (executor is BUNDLE-019's);
    (3) two application MODES (direct + SDLC-mediated) from one artifact; (4)
    non-destructive-toward-user-files + regenerate-by-default-for-recipe-owned-output
    with divergence as a `@waiver` (never a bespoke merge), and templating-kind apply
    as ONE-SHOT / consumer-owned; (5) a thin tracked ADOPTION RECORD
    `{recipe ref, @version, adopted}` written on apply (NOT the rich BUNDLE-017 ledger);
    (6) fail-loud on an unreachable `transform`/`insert` with the exact manual
    instruction; (7) strictly sequential deterministic multi-recipe apply in declared
    order. The applier carries ZERO language/platform/CI literals: `transform`
    dispatches to an allowlisted GENERAL engine through the SAME `engine.CheckToolAllowed`
    trust gate the gate's enforcement dispatch uses (match->fix reusing the match->finding
    substrate), and `backstop/self` guards the seam. The `recipes:` index is validated
    by pack-manifest validation (pkg/pack). Covers BUNDLE-015 REQ-001..REQ-010, REQ-021,
    REQ-023, REQ-024 ONLY. OUT OF SCOPE (later specs / bundles): the rev-guard (REQ-011),
    the three drift signals (REQ-012), enforcement activation / static scope / un-adopt
    (REQ-013/014/022), compat matrix / version-keyed variants / migration / chain
    traversal (REQ-015/016/017/020), the CI + provider packs (REQ-018/019), the `step`
    executor + probe-receipt engine (BUNDLE-019), and the rich provenance ledger +
    dynamic-`transform` output scoping (BUNDLE-017).
  subject: pkg/recipe

verification:
  level: integration
  test_command: go test ./pkg/recipe/... ./pkg/pack/... ./cmd/backstop/... -race -coverprofile=cover.out
  coverage_threshold: 80

requirements:
  - id: REQ-001
    text: >
      `pkg/recipe` must provide a GENERIC applier that runs a resolved recipe's
      declared ops in the recipe's DECLARED SEQUENCE order. The applier carries NO
      path or target knowledge — each op names its own target(s), read from the
      recipe manifest, and the applier writes exactly there and nowhere else. A
      recipe with zero ops applies as a clean no-op. The applier never contains a
      language-, platform-, or CI-aware branch (that seam is REQ-006).
    supports: pack-scaffolding-recipes:REQ-001@1.0.0
    follows: STD-GO-001:GO-010
  - id: REQ-002
    text: >
      The applier must support exactly four language-neutral OP families: `create`
      (materialize a templated file/tree at the declared target), `merge` (deep-merge
      a declared fragment into a STRUCTURED file — json, yaml, toml, or .env),
      `transform` (an AST rewrite dispatched to an allowlisted GENERAL engine running
      a declared rule — REQ-006), and `insert` (a snippet at a declared anchor). A
      `merge` op targeting a file whose format is not one of json/yaml/toml/.env is a
      fail-loud error. Value substitution is a declarative `{{ param }}` convention
      that resolves declared params ONLY and is explicitly NOT Turing-complete: it
      performs pure value interpolation with no conditionals, loops, or expression
      evaluation, and an undeclared placeholder is a fail-loud error (never silently
      blanked). Substitution SCOPE is part of the contract, not incidental: EVERY
      CONSUMER-FACING site/content field an op declares must be resolved through
      substitution BEFORE it is used to locate a path, match an anchor, splice bytes, or
      report a result — `Op.Target` (create / merge / transform / insert), `Op.Anchor`
      (insert, INCLUDING the half a caller supplies through SDLC-mediated
      `InjectionSites` — REQ-003), `Op.Snippet` (insert), the `create` payload and `merge`
      fragment CONTENT, and the relayed `Op.Manual` instruction (REQ-011, the one
      fail-SOFT field). `Op.Rule` is the ONE declared field that is NEVER substituted: it
      is validated at parse time by exact string equality against the recipe's declared
      `transform` rules allowlist and it selects which PACK ASSET an allowlisted engine
      executes in place over the consumer's tree, so a consumer-supplied param must not
      carry that authority; a placeholder in `rule:` is instead REFUSED at manifest
      validation (REQ-009). The resulting invariant: after a successful apply no literal
      `{{` / `}}` may reach the consumer's filesystem — not in any path an op writes,
      reads or creates, and not in the bytes it splices. `Op.Fragment` as a PATH joins
      `Op.Rule` on the NOT-substituted side: a merge op's `fragment:` is a
      recipe-directory-relative path resolved under the recipe directory and read from disk
      VERBATIM, and its hole is closed at parse instead — a non-path `fragment:` (inline
      content, a `{{` placeholder, or an empty value) is REFUSED at manifest validation
      (REQ-009). The fragment FILE'S CONTENT is still substituted, as it has been since
      1.0.0; path and content are separate fields of the contract and must not be read as
      contradicting each other. (`Op.Payload` as a PATH remains deliberately OUT OF SCOPE
      of this rule — its form and templatability stay ISSUE-081's open question, decided
      there and not here.)
    supports: pack-scaffolding-recipes:REQ-002@1.0.0
    follows: STD-GO-001:GO-010
  - id: REQ-003
    text: >
      The applier must support two application MODES from the SAME recipe artifact:
      DIRECT (self-applies its ops identically and deterministically every run from
      the recipe-declared defaults/params — the same recipe+params yields byte-identical
      output across runs) and SDLC-MEDIATED (applied through a spec/plan that supplies
      only the WHERE — the injection site(s) — while the recipe supplies the WHAT, i.e.
      the template + ops). In SDLC-mediated mode, `ApplyOptions.InjectionSites` is a map
      KEYED BY the declared op `id` whose value is the WHERE (target path and/or anchor
      locator) for the INJECTION-ACCEPTING op families ONLY — `transform` and `insert`;
      `create`, `merge`, and `step` are not injection-accepting and ignore
      `InjectionSites`. A supplied site is applied at exactly that op; a `transform`/`insert`
      op that in SDLC-mediated mode has neither a declared target/anchor nor a supplied
      injection site fails loud (it must not fall back to a guessed site — REQ-011). Both
      modes drive the same op executors over the same recipe artifact; the mode selects
      only where the WHERE comes from (recipe-declared defaults vs supplied sites). Direct
      mode's PARAM scope is the recipe-declared defaults OVERRIDDEN by operator-supplied
      values, and the shipped CLI must expose a way to supply them: a repeatable
      `--param key=value` threaded into `ApplyOptions.Params`, so a param declared
      `required: true` with no `default:` is reachable through the shipped command rather
      than permanently unusable in direct mode. A malformed argument (no `=`, or an empty
      key), a DUPLICATE key, and a key the resolved recipe does not declare are each a
      fail-loud config error naming the offending argument, refused BEFORE any op is
      applied — never silently ignored, never last-wins, and never degraded into an
      unattributable unresolvable-placeholder failure later. Splitting is on the FIRST `=`
      only (a value may contain `=`), an empty value is a legal supplied empty string, and
      values are never comma-split. This is a CLI-layer input surface only: `Apply` and
      every SDLC-mediated library caller are unchanged.
    supports: pack-scaffolding-recipes:REQ-003@1.0.0
    follows: STD-GO-001:GO-010
  - id: REQ-004
    text: >
      Application must be non-destructive toward USER-OWNED files: a `create` op must
      NEVER clobber a consumer-owned file already present at its target. For
      RECIPE-OWNED output of the scaffolding and implementing kinds the model is
      regenerate-by-default with an accountable-divergence hinge computed MECHANICALLY on
      re-apply: the applier computes the would-be-regenerated output and diffs it against
      the on-disk file. (a) NOT diverged → nothing to do. (b) DIVERGED AND a covering
      `@waiver` is ACTIVE — determined by adjudicating a synthetic divergence finding
      (keyed on the recipe's declared enforcement rule for that path) through the REAL
      `pkg/waiver` read/adjudication path (`waiver.Adjudicate` fed a `waiver.LineReader`
      over the consumer's on-disk file) — → PRESERVE the on-disk file and record the
      accountable divergence (the covering waiver) in `ApplyResult`. (c) DIVERGED AND no
      covering waiver → REGENERATE (overwrite) per the kind's model. The applier NEVER
      AUTHORS a waiver token: the `<reason>` and `<expiry>` of
      `@waiver:<rule>:<reason>:<expiry>` are human judgments, so the consumer adds the
      waiver FIRST and re-apply then preserves — auto-writing a token is the unaccountable
      anti-pattern this forbids, and there is NEVER a bespoke merge/upgrade path. "Never
      clobber" protects files the recipe does not own; recipe-owned output is regenerable
      and un-waivered consumer edits to it are overwritten, not silently preserved. (The
      templating-kind exception is REQ-012.)
    supports: pack-scaffolding-recipes:REQ-004@1.0.0
    follows: STD-GO-001:GO-010
  - id: REQ-005
    text: >
      Each application must write a MINIMAL, tracked ADOPTION RECORD entry —
      `{recipe ref, @version, adopted}`, the recipe analog of `backstop.lock` (a tracked
      file at the project root) — carrying the applied recipe `@version`. The applier
      must NOT emit any rich per-op or per-region provenance detail; that ledger is owned
      by BUNDLE-017 and is out of scope here. The adoption record contains only the thin
      fields.
    supports: pack-scaffolding-recipes:REQ-005@1.0.0
    follows: STD-GO-001:GO-010
  - id: REQ-006
    text: >
      The applier and its `transform`-engine dispatch must contain ZERO
      language/platform/CI literals — they resolve and run declared DATA only. A
      `transform` op must dispatch to an allowlisted GENERAL engine running the recipe's
      declared rule, routed through the SAME `engine.CheckToolAllowed` trust gate the
      gate's enforcement dispatch uses (an un-allowlisted or non-lock-pinned engine tool
      is rejected, exit-2 ConfigError shape), so that language-awareness lives in the
      engine, never in core — enforcement reads the AST (match -> finding); a recipe
      rewrites it (match -> fix); same engines, same declared-rule model. `backstop/self`
      must stay GREEN over the new applier + transform-dispatch code (no baked
      language/platform/CI noun).
    supports: pack-scaffolding-recipes:REQ-006@1.0.0
    follows: STD-GO-001:GO-010
  - id: REQ-007
    text: >
      The apply sequencing must RESERVE a `step` op as a fifth op family: a `step` op
      must be RECOGNIZED as valid and sequenced in the recipe's declared order alongside
      the file ops, but it must NOT be executed here — the applier defers/reserves it,
      because the step EXECUTOR and probe-receipt/precondition engine are owned by
      BUNDLE-019. The op-family set is a CLOSED allowlist: an op whose kind is not one of
      `create` / `merge` / `transform` / `insert` / `step` is a fail-loud error, never
      silently skipped.
    supports: pack-scaffolding-recipes:REQ-007@1.0.0
    follows: STD-GO-001:GO-010
  - id: REQ-008
    text: >
      A recipe must be declared as its OWN DIRECTORY colocating a `recipe.yml` manifest,
      its template payload, and its `transform`-rule files. `pack.yml` must carry a
      lightweight `recipes:` INDEX — a distinct top-level key mapping a stable recipe-id
      to that directory — that does NOT collide with `content.scaffolds` (declaring both
      is valid and independent) or with `pack scaffold` / `artifact new`. Multiple
      distinct recipes in one pack are multiple directories, each addressed by a stable
      id in the pack namespace. A `recipes:` entry pointing at a directory that is
      missing or lacks a `recipe.yml` is a pack-manifest validation error.
    supports: pack-scaffolding-recipes:REQ-008@1.0.0
    follows: STD-GO-001:GO-010
  - id: REQ-009
    text: >
      The `recipe.yml` manifest must declare the ordered ops, the KIND
      (scaffolding / implementing / templating), the param schema, the target paths, the
      `transform` rules, the per-op `manual:` instruction (REQ-011), the paired
      enforcement suite declaration, the recipe VERSION (well-formed semver), and —
      optionally — the compat matrix and version-keyed variants. Structural validation
      must run at parse time: a manifest missing `ops`, missing `version`, carrying a
      malformed (non-semver) version, or declaring a `kind` outside
      {scaffolding, implementing, templating} is a fail-loud error; each of the three
      valid kinds parses clean; an optionally-declared compat matrix and version-keyed
      variant block validate STRUCTURALLY (their apply-time behavior is out of scope for
      this spec). Three op-level cross-checks are also fail-loud at parse time: (1) every
      `transform` and `insert` op MUST carry a non-empty `manual:` instruction (the ops
      that can hit the injection limit — REQ-011 — must supply the human-actionable
      fallback text as DATA, since core cannot synthesize a language/framework-specific
      instruction without violating REQ-006); a `transform`/`insert` op missing `manual:`
      is a validation error. (2) every `transform` op's `rule` MUST be a pack-relative
      rule-file path that appears in the recipe's declared `transform` rules list; a
      `transform` op citing a rule file the recipe did not declare is a validation error.
      (3) op `id`s MUST be UNIQUE within a recipe, and MUST be non-empty on every
      injection-accepting op (`transform`/`insert`) — since `id` is the key
      `ApplyOptions.InjectionSites` routes the SDLC-mediated WHERE by (REQ-003), a
      duplicate id would misroute the site and an empty id on an injection-accepting op
      would make routing ambiguous; a duplicate id, or an empty id on a
      `transform`/`insert` op, is a fail-loud validation error naming the recipe and the
      duplicate/empty id. (4) a `transform` op's `rule` MUST NOT carry a substitution
      placeholder: `rule:` is the one op field the applier deliberately never substitutes
      (REQ-002), so a templated rule would otherwise resolve a brace-bearing path into an
      engine invocation at apply time; a placeholder opening delimiter appearing in
      `rule:` is a fail-loud validation error naming the op index, the op id and the
      FIELD, and it is reported BEFORE the declared-rules allowlist cross-check so a
      templated rule reports its real cause rather than "not among the declared rules".
      The rule-placeholder check remains scoped to `rule:` ONLY. `fragment:` is separately
      constrained by check (5) below; `payload:` is not checked here and its templatability
      remains ISSUE-081's open question. (5) a `merge` op's `fragment:` MUST be a
      recipe-directory-relative PATH: the applier resolves it under the recipe directory and
      reads the file, so a declaration that can only be something else is a fail-loud
      validation error naming the op index, the op id and the FIELD. Three shapes are
      refused, each with its OWN message so the diagnostic names the real cause: an empty or
      whitespace-only value (a merge op cannot merge nothing), a value containing a NEWLINE
      (the inline block-scalar form — checked BEFORE the placeholder check, since an inline
      block usually carries a placeholder too and diagnosing it as a templated path would
      name the wrong defect), and a value containing the `{{` opening delimiter (a fragment
      PATH is resolved verbatim and never substituted, so a templated path can only fail to
      open). The predicate is deliberately NOT widened: a single-line, placeholder-free value
      that is syntactically a valid relative filename is ACCEPTED here even if it is really
      inline content, because separating the two would require a content sniff on the
      document's first byte — syntax knowledge the manifest reader must not carry. That
      residual case fails at APPLY, where the message names this path-only contract rather
      than leaking a bare ENOENT. The check's scope is the recipe's own `ops` only; variant
      ops continue to validate structurally.
    supports: pack-scaffolding-recipes:REQ-009@1.0.0
    follows: STD-GO-001:GO-010
  - id: REQ-010
    text: >
      A recipe reference `<pack>:<recipe>@<recipe_version>` must RESOLVE at apply time to
      the pinned recipe: the named pack must exist and declare the recipe id in its
      `recipes:` index, and the pinned recipe version must match the target recipe's
      declared `version`. A ref naming a missing pack, a recipe id absent from the pack's
      `recipes:` index, or a `@version` that does not match the recipe's declared version
      is a fail-loud error; an unpinned or malformed ref (no `@X.Y.Z`) is a fail-loud
      error. (The publish-time rev-guard that keeps recipe versions trustworthy is
      REQ-011, out of scope here — this covers only the apply-time resolution half.)
    supports: pack-scaffolding-recipes:REQ-010@1.0.0
    follows: STD-GO-001:GO-010
  - id: REQ-011
    text: >
      Where a `transform` CANNOT reach its target — no convention to pattern against —
      apply must FAIL LOUD, NEVER silently skipping the op (apply must NOT report success)
      and NEVER guessing a site (no fallback write occurs). The fail-loud message MUST be
      the op's DECLARED `manual:` instruction text (REQ-009) emitted VERBATIM, plus a
      locator (the op id + intended target) — core NEVER synthesizes the instruction, because
      an actionable "wire it in by hand like THIS" instruction is inherently
      language/framework-specific and would violate REQ-006 if built in core; the recipe
      supplies it as DATA. The same fail-loud-with-declared-`manual:`-instruction rule
      applies to an `insert` op whose declared anchor is absent from the target. VERBATIM
      means core NEVER composes, paraphrases, re-wraps or synthesizes the instruction — it
      does NOT mean UNRENDERED: the relayed text is run through the recipe's OWN declared
      params (REQ-002) so the operator reads a resolved instruction rather than literal
      `{{ }}` braces; rendering the recipe author's declared params is not composition.
      Because the manual is emitted ONLY on an error path, this is the one field where
      substitution fails SOFT: if substituting the manual itself fails, the RAW declared
      text is relayed unchanged and the unreachable-site failure is still reported in
      full — replacing the operator's instruction with a second error, on a path that is
      already reporting a failure, is worse than an unrendered placeholder. ("When
      generation can't reach, enforcement does" — the paired-enforcement backstop is
      REQ-013/014, out of scope here; this spec owns the apply-time fail-loud half.)
    supports: pack-scaffolding-recipes:REQ-021@1.0.0
    follows: STD-GO-001:GO-010
  - id: REQ-012
    text: >
      Applying a TEMPLATING-kind recipe is ONE-SHOT: after initial application the output
      is fully CONSUMER-OWNED and must NEVER be regenerated or re-applied over the
      consumer's changes on a re-apply or pack upgrade. The regenerate-by-default model of
      REQ-004 applies ONLY to the scaffolding and implementing kinds, NEVER to templating —
      a consumer edit to templating-kind output must survive a re-apply. The kind is read
      from the recipe manifest.
    supports: pack-scaffolding-recipes:REQ-023@1.0.0
    follows: STD-GO-001:GO-010
  - id: REQ-013
    text: >
      Applying MULTIPLE recipes — within one pack or across packs — must be STRICTLY
      SEQUENTIAL and DETERMINISTIC in the consumer's DECLARED order; co-writes to the same
      structured file compose via `merge` (REQ-002) in that declared order as normal
      composition, not a conflict to arbitrate. The applier must NEVER interleave or
      reorder recipes.
    supports: pack-scaffolding-recipes:REQ-024@1.0.0
    follows: STD-GO-001:GO-010

claims:
  # REQ-001 — generic ordered-op applier, no baked target knowledge
  - id: CLM-001
    requirement: REQ-001
    text: The applier runs a recipe's ops in the recipe's declared sequence order
    tests:
      - TestApply_RunsOpsInDeclaredOrder
  - id: CLM-002
    requirement: REQ-001
    text: >
      The applier contributes no target path — a create op writes exactly at the
      recipe-declared target and nowhere else. "Recipe-declared target" means the declared
      target AFTER the recipe's own param substitution (REQ-002): the substituted form is
      what is written, and it is what `ApplyResult.Written` echoes — still the recipe's own
      path FORM, never a path the applier invented and never the applier's absolute
      resolution of it.
    tests:
      - TestApply_TargetComesFromRecipeNotApplier
  - id: CLM-003
    requirement: REQ-001
    text: A recipe with zero ops applies as a clean no-op (no error, no writes)
    tests:
      - TestApply_EmptyOpsNoop

  # REQ-002 — four op families + merge-format matrix + non-Turing substitution
  - id: CLM-004
    requirement: REQ-002
    text: A create op materializes a templated file/tree at the declared target (against a captured fixture payload)
    tests:
      - TestApply_CreateOp_MaterializesCapturedFixture
  - id: CLM-005
    requirement: REQ-002
    text: A merge op deep-merges a declared fragment into a structured JSON file
    tests:
      - TestApply_MergeOp_Json
  - id: CLM-006
    requirement: REQ-002
    text: A merge op deep-merges a declared fragment into a structured YAML file
    tests:
      - TestApply_MergeOp_Yaml
  - id: CLM-007
    requirement: REQ-002
    text: A merge op deep-merges a declared fragment into a structured TOML file
    tests:
      - TestApply_MergeOp_Toml
  - id: CLM-008
    requirement: REQ-002
    text: A merge op merges a declared fragment into a .env file
    tests:
      - TestApply_MergeOp_DotEnv
  - id: CLM-009
    requirement: REQ-002
    text: A merge op targeting a file whose format is not json/yaml/toml/.env fails loud (unsupported structured format)
    tests:
      - TestApply_MergeOp_UnsupportedFormatFailsLoud
  - id: CLM-088
    requirement: REQ-002
    text: >
      The RESIDUAL fragment failure that parse cannot catch — a single-line, placeholder-free
      value that is really inline content but is syntactically a valid relative filename, and a
      declared path naming nothing under the recipe directory — fails the APPLY with a message
      that NAMES the path-only contract (a recipe-directory-relative path read from disk) and
      echoes the declared value, keeping the underlying read failure wrapped rather than
      surfacing it bare. Asserting only that the error is non-nil would leave the undiagnosable
      bare-ENOENT message passing, which is the failure ISSUE-081 was filed about
    tests:
      - TestApply_MergeOp_UnreadableFragmentPathNamesThePathOnlyContract
  - id: CLM-010
    requirement: REQ-002
    text: A transform op dispatches an AST rewrite to an allowlisted general engine running the declared rule, transforming a CAPTURED before-fixture to its captured after-state
    tests:
      - TestApply_TransformOp_DispatchesToAllowlistedEngine_CapturedFixture
  - id: CLM-011
    requirement: REQ-002
    text: An insert op inserts the declared snippet at the declared anchor
    tests:
      - TestApply_InsertOp_AtAnchor
  - id: CLM-012
    requirement: REQ-002
    text: A "{{ param }}" placeholder is substituted from the declared params
    tests:
      - TestSubstitute_ResolvesDeclaredParam
  - id: CLM-013
    requirement: REQ-002
    text: Substitution is NOT Turing-complete — a logic/expression construct inside a placeholder is never evaluated as code
    tests:
      - TestSubstitute_NotTuringComplete_NoLogicEvaluated
  - id: CLM-014
    requirement: REQ-002
    text: An undeclared placeholder fails loud (never silently blanked)
    tests:
      - TestSubstitute_UndeclaredParamFailsLoud
  # REQ-002 — substitution SCOPE. CLM-012/013/014 above prove Substitute's SEMANTICS by
  # driving Substitute DIRECTLY, which cannot prove which fields the APPLIER routes
  # through it; the claims below are the Apply-level obligation and every one of them
  # drives the real Apply (or the real parse / the real CLI).
  - id: CLM-070
    requirement: REQ-002
    text: >
      A COMMITTED, resolution-driven recipe whose create / merge / transform / insert
      sites are all templated applies end to end with EVERY templated site resolved —
      the create target, the merge target, the transform target, the insert anchor and
      the insert snippet — proved through Apply over the resolved recipe, not through
      Substitute, and the result's Written paths carry the substituted project-relative
      form
    tests:
      - TestApply_CommittedFixtureRecipe_SubstitutesEveryTemplatedSite
  - id: CLM-071
    requirement: REQ-002
    text: >
      Driving the COMMITTED starter recipe through Apply resolves its templated create
      target AND merges its declared fragment PATH — the create op's file lands at the
      substituted path with no literal-brace directory created, and the merge op resolves
      `fragment:` under the recipe directory, reads that file, substitutes its CONTENT and
      merges it additively into the seeded consumer document (the pre-existing keys survive
      alongside the merged ones). The path is read verbatim and the content is substituted:
      both halves of the split are proved in one run over a committed artifact, not a
      test-local manifest
    tests:
      - TestApply_CommittedStarterRecipe_MergesTheDeclaredFragmentPath
  - id: CLM-072
    requirement: REQ-002
    text: >
      INVARIANT — after a successful apply, no path beneath the project root carries a
      `{{` or `}}` segment anywhere (asserted by walking the whole tree, not against a
      known-good path list, so an op family that forgets substitution is caught without
      anyone updating a list)
    tests:
      - TestApply_NoLiteralPlaceholderReachesTheFilesystem
  - id: CLM-073
    requirement: REQ-002
    text: >
      An UNDECLARED placeholder in ANY substituted site/content field — a create target, a
      merge target, an insert anchor, an insert snippet — fails the APPLY loud, with an
      error naming the recipe, the op INDEX and the op ID, and nothing is written (a
      partially-applied failure is as bad as a silent success)
    tests:
      - TestApply_UndeclaredPlaceholderInSiteFieldFailsLoud
  - id: CLM-074
    requirement: REQ-002
    text: >
      BOTH halves of a caller-supplied SDLC-mediated injection site are substituted — an
      InjectionSites value whose target AND anchor are templated writes at the substituted
      target and splices at the substituted anchor, and the recipe-DECLARED target is not
      written (so an applier that ignored the override cannot pass)
    tests:
      - TestApply_SDLCMediatedMode_SuppliedInjectionSiteIsSubstituted
  - id: CLM-075
    requirement: REQ-002
    text: >
      A transform op's declared `rule` is NOT substituted — with a templated target and a
      literal declared rule, the dispatch receives the substituted target and the
      byte-identical declared, transform-rules-validated rule path resolved under PackDir
      (guards the deliberate asymmetry against a later "fix" that substitutes everything
      uniformly)
    tests:
      - TestApply_TransformOp_DeclaredRuleIsNotSubstituted
  - id: CLM-077
    requirement: REQ-002
    subject: cmd/backstop
    text: >
      The shipped `backstop recipe apply` resolves a DEFAULTED param and the file lands at
      the real substituted path — the command succeeds, the declared payload is at the
      substituted target, and no `{{`-bearing path exists anywhere beneath the staged
      project root. Per-claim `subject: cmd/backstop` (like CLM-025/027/063): the test
      drives the shipped root command over an INSTALLED pack, so the inherited `pkg/recipe`
      subject would trip the substantiveness noTarget join on a correctly-placed test.
    tests:
      - TestRecipeApply_CLI_TemplatedTargetWritesAtTheSubstitutedPath

  # REQ-003 — two application modes from one artifact
  - id: CLM-015
    requirement: REQ-003
    text: Direct mode self-applies the recipe's ops from the recipe-declared defaults/params
    tests:
      - TestApply_DirectMode_SelfAppliesFromDefaults
  # Direct mode's OPERATOR-SUPPLIED param half (ISSUE-081 Gap 2). CLM-015 states what a
  # recipe supplies to itself; these state what the operator supplies to it through the
  # SHIPPED command. Each carries `subject: cmd/backstop` (like CLM-025/027/063/077): the
  # tests drive the shipped root command over an INSTALLED pack, so the inherited
  # `pkg/recipe` subject would trip the substantiveness noTarget join on a correctly-placed
  # test.
  - id: CLM-089
    requirement: REQ-003
    subject: cmd/backstop
    text: >
      A repeatable `--param key=value` on the shipped `backstop recipe apply` threads into
      `ApplyOptions.Params`, so a param declared `required: true` with NO `default:` resolves
      through the command — the file lands at the substituted target and the supplied value
      reaches the written BYTES, not only the path. The SAME invocation WITHOUT the flag
      fails, which is what proves the flag is load-bearing rather than decorative
    tests:
      - TestRecipeApply_CLI_ParamFlagSuppliesRequiredNoDefaultParam
  - id: CLM-090
    requirement: REQ-003
    subject: cmd/backstop
    text: >
      A malformed `--param` — no `=` at all, or an empty key — is a fail-loud config error
      (exit 2) QUOTING the offending argument, refused before any op runs so nothing is
      written beneath the project root; it is never silently ignored
    tests:
      - TestRecipeApply_CLI_MalformedParamFlagFailsLoud
  - id: CLM-091
    requirement: REQ-003
    subject: cmd/backstop
    text: >
      A DUPLICATE `--param` key is a fail-loud config error naming the repeated key, not
      last-wins — asserted with a key the recipe DECLARES, so the undeclared-name check
      cannot be what rejects it, and with nothing written beneath the project root (a map
      write would have silently kept the last value)
    tests:
      - TestRecipeApply_CLI_DuplicateParamKeyFailsLoud
  - id: CLM-092
    requirement: REQ-003
    subject: cmd/backstop
    text: >
      A `--param` naming a param the resolved recipe does not declare is a fail-loud config
      error naming the offending name, refused after resolution and before apply with nothing
      written — and the message must NOT read as an unresolvable-placeholder failure, since
      that undiagnosable shape is what the check exists to replace. The complement holds: an
      invocation whose supplied names are all DECLARED (including an explicit override of a
      param that carries a default) is accepted, so the check cannot pass by rejecting
      everything
    tests:
      - TestRecipeApply_CLI_UndeclaredParamFailsLoud
  - id: CLM-093
    requirement: REQ-003
    subject: cmd/backstop
    text: >
      A `--param` value carrying BOTH `,` and `=` survives intact end to end — the flag is a
      pflag StringArray (a StringSlice would comma-split the value into two params) and
      `key=value` splits on the FIRST `=` (splitting on the last would mangle the remainder).
      Asserted on the WRITTEN path and bytes rather than on an internal map
    tests:
      - TestRecipeApply_CLI_ParamValueWithSeparatorsSurvivesIntact
  - id: CLM-016
    requirement: REQ-003
    text: Direct mode is deterministic — the same recipe+params yields byte-identical output across two runs
    tests:
      - TestApply_DirectMode_Deterministic
  - id: CLM-017
    requirement: REQ-003
    text: SDLC-mediated mode applies an injection-accepting op (transform/insert) at the InjectionSites[op-id] site the plan supplies — a DIFFERENT supplied site produces a different, falsifiable write location, and the recipe still supplies the WHAT
    tests:
      - TestApply_SDLCMediatedMode_AppliesAtSuppliedInjectionSite
  - id: CLM-060
    requirement: REQ-003
    text: A transform/insert op in SDLC-mediated mode with neither a declared target/anchor nor a supplied injection site fails loud (no guessed fallback site)
    tests:
      - TestApply_SDLCMediatedMode_MissingInjectionSiteFailsLoud

  # REQ-004 — non-destructive toward user files; regenerate/waiver for recipe-owned (scaffolding+implementing)
  - id: CLM-018
    requirement: REQ-004
    text: A create op never clobbers a USER-OWNED file already present at its target
    tests:
      - TestApply_UserOwnedFileNeverClobbered
  - id: CLM-019
    requirement: REQ-004
    text: Recipe-owned output of a SCAFFOLDING recipe is regenerated by default on re-apply
    tests:
      - TestApply_Scaffolding_RegeneratesRecipeOwnedOutput
  - id: CLM-020
    requirement: REQ-004
    text: Recipe-owned output of an IMPLEMENTING recipe is regenerated by default on re-apply
    tests:
      - TestApply_Implementing_RegeneratesRecipeOwnedOutput
  - id: CLM-021
    requirement: REQ-004
    text: On re-apply, DIVERGED recipe-owned output WITH a covering active @waiver is PRESERVED because the divergence finding was adjudicated through the real pkg/waiver read path (waiver.Adjudicate over a LineReader on the consumer's file) — the covering waiver is recorded in ApplyResult, not synthesized
    tests:
      - TestApply_DivergedWithActiveWaiver_PreservedViaRealWaiverReadPath
  - id: CLM-061
    requirement: REQ-004
    text: On re-apply, DIVERGED recipe-owned output with NO covering waiver is REGENERATED (overwritten) per the kind's model
    tests:
      - TestApply_DivergedNoWaiver_Regenerates
  - id: CLM-062
    requirement: REQ-004
    text: Apply NEVER authors a @waiver token — a divergence with no pre-existing waiver never causes a token to be written into any file
    tests:
      - TestApply_NeverAuthorsWaiverToken
  - id: CLM-079
    requirement: REQ-004
    text: >
      An UNCOVERED divergence whose file carries a @waiver with an UNPARSEABLE reason
      code FAILS the apply loud — the error names the target as reported, the token's
      LINE and the illegal reason code — and the consumer's bytes are left on disk
      byte-for-byte. Regenerate-but-warn was rejected (the operator's edit is still
      destroyed, only more chattily), and preserve-silently was rejected (a typo in a
      reason code would become a permanent, unaudited opt-out of regeneration). This
      holds even when the malformed token names ANOTHER rule, and that asymmetry is
      DELIBERATE: a VALID token naming another rule does NOT block (it parsed, so its
      rule id is readable and it is knowably not about this recipe), while a MALFORMED
      one does (ParseToken fails before any field is trustworthy, so it cannot be
      dismissed as someone else's business). The two halves must not be "cleaned up"
      into symmetry
    tests:
      - TestApply_DivergedWithMalformedWaiverToken_FailsLoudAndPreservesConsumerBytes
      - TestApply_DivergedWithMalformedWaiverToken_DiagnosticNamesFileLineAndReasonCode
      - TestApply_MalformedTokenForAnotherRule_AlsoBlocks
      - TestRecipeApply_CLI_MalformedWaiverTokenSurfacesDiagnosticOnStderr
  - id: CLM-080
    requirement: REQ-004
    text: >
      A COVERED divergence that ALSO carries a separate malformed token is still
      PRESERVED and the apply still exits 0, AND the malformed token is still SURFACED
      as a warning naming the file, the line and the bad reason code — coverage and
      token hygiene are independent facts, and blocking here would punish the
      accountable path for an unrelated typo
    tests:
      - TestApply_CoveredDivergenceWithSeparateMalformedToken_PreservesAndReportsDiagnostic
      - TestRecipeApply_CLI_SuccessOutputNamesWrittenPreservedAndRegenerated
  - id: CLM-081
    requirement: REQ-004
    text: >
      ONE token yields ONE diagnostic regardless of how many enforcement rules the
      recipe declares — the same token is re-adjudicated once per declared rule, so
      the accumulation dedupes by token identity {File, Line, Message}
    tests:
      - TestApply_MultipleEnforcementRules_MalformedTokenReportedOnce
  - id: CLM-082
    requirement: REQ-004
    text: >
      `ApplyResult.Regenerated` is POPULATED and is a strict SUBSET of `Written` in the
      SAME value form, so the CLI marks a clobber from DATA rather than re-deriving the
      distinction: a re-apply over an UNCOVERED divergence records the target in both
      lists, while a write over nothing (first apply) and a re-apply over an UNCHANGED
      file record it in `Written` ONLY — the falsifier for the two lists being aliases
    tests:
      - TestApply_DivergedNoWaiver_RecordsRegeneratedOverDivergence
      - TestRecipeApply_CLI_SuccessOutputNamesWrittenPreservedAndRegenerated
  - id: CLM-083
    requirement: REQ-004
    text: >
      The malformed-token refusal has a BOUNDED blast radius — a recipe declaring NO
      enforcement rule has no rule id a consumer could waive against, so the waiver seam
      is never consulted, a malformed token in that recipe's output is invisible, the
      output still regenerates and no diagnostic is reported. Without this bound the
      refusal would start failing applies for recipes that never opted into divergence
      adjudication at all
    tests:
      - TestApply_NoEnforcementRules_MalformedTokenCannotBlock
  - id: CLM-084
    requirement: REQ-004
    text: >
      The type-level precondition every other divergence claim rests on — the seam
      returns a DivergenceVerdict, so "not covered because the token was MALFORMED" and
      "not covered because there was NO token" arrive at the decision point as different
      values (`Covered` is false in both; only `Diagnostics` differs) and produce
      opposite outcomes. A bare-bool seam cannot express that difference, and the real
      production reader — not only a stub — must actually produce such a verdict
    tests:
      - TestApply_WaiverReaderVerdict_CarriesAdjudicationDiagnostics

  # REQ-005 — thin adoption record
  - id: CLM-022
    requirement: REQ-005
    text: Apply writes an adoption record entry {recipe ref, @version, adopted}
    tests:
      - TestApply_WritesAdoptionRecord
  - id: CLM-023
    requirement: REQ-005
    text: The adoption record carries the applied recipe @version
    tests:
      - TestApply_AdoptionRecordCarriesAppliedVersion
  - id: CLM-024
    requirement: REQ-005
    text: The adoption record is THIN — it contains no rich per-op/per-region provenance (that is BUNDLE-017's)
    tests:
      - TestApply_AdoptionRecordIsThin_NoRichLedger

  # REQ-006 — zero literals; transform reuses the engine trust gate; backstop/self green
  - id: CLM-025
    requirement: REQ-006
    subject: cmd/backstop
    text: >
      Transform dispatch runs through the same engine.CheckToolAllowed trust gate — an
      un-allowlisted engine tool is rejected as a ConfigError before any command is built.
      Per-claim `subject: cmd/backstop` (like CLM-027/CLM-063): the trust gate is only
      reachable from the layer that can see the pack's `engines:` block, so the test drives
      the root command from cmd/backstop and would otherwise fail the substantiveness
      noTarget join against the inherited `pkg/recipe` subject.
    tests:
      - TestApply_TransformOp_UnallowlistedEngineRejected
  - id: CLM-026
    requirement: REQ-006
    text: Transform resolves and runs DECLARED rule data only — the allowlisted engine, not core, carries language-awareness
    tests:
      - TestApply_TransformOp_RunsDeclaredRuleNotBakedLogic
  - id: CLM-027
    requirement: REQ-006
    kind: absence
    subject: cmd/backstop
    text: backstop/self stays GREEN over the recipe applier + transform-dispatch code (zero language/platform/CI literals)
    tests:
      - TestSelfPack_GreenOverRecipeApplier
  - id: CLM-063
    requirement: REQ-006
    subject: cmd/backstop
    text: End-to-end, a REAL allowlisted engine (ast-grep) runs a declared rewrite rule against a CAPTURED before-fixture through the actual `backstop recipe apply` CLI and the file content transforms to the captured after-state (a no-op TransformDispatch would fail this)
    tests:
      - TestRecipeApply_E2E_RealEngineTransform_CapturedFixture

  # REQ-007 — the reserved step op seam + closed op-family allowlist
  - id: CLM-028
    requirement: REQ-007
    text: A step op is recognized as a valid fifth op family and sequenced in the recipe's declared order without failing as unknown
    tests:
      - TestApply_StepOp_RecognizedAndSequenced
  - id: CLM-029
    requirement: REQ-007
    text: A step op is NOT executed here — the applier reserves/defers it (executor is BUNDLE-019's)
    tests:
      - TestApply_StepOp_NotExecutedReservedSeam
  - id: CLM-030
    requirement: REQ-007
    text: An op whose kind is not one of create/merge/transform/insert/step fails loud (never silently skipped)
    tests:
      - TestApply_UnknownOpKindFailsLoud

  # REQ-008 — per-recipe directory + recipes: index, no collision with content.scaffolds
  - id: CLM-031
    requirement: REQ-008
    text: A recipe declared as its own directory (recipe.yml + payload + transform rules) is discovered and parsed
    tests:
      - TestRecipeDir_ParsesColocatedManifestAndPayload
  - id: CLM-032
    requirement: REQ-008
    subject: pkg/pack
    text: The pack.yml recipes index maps a stable recipe-id to its directory
    tests:
      - TestPackManifest_RecipesIndexMapsIdToDir
  - id: CLM-033
    requirement: REQ-008
    subject: pkg/pack
    text: The recipes index is a distinct top-level key from content.scaffolds — declaring both is valid and does not collide
    tests:
      - TestPackManifest_RecipesIndexDistinctFromScaffolds
  - id: CLM-034
    requirement: REQ-008
    subject: pkg/pack
    text: Multiple distinct recipes are multiple directories, each addressed by a stable id in the pack namespace
    tests:
      - TestPackManifest_MultipleRecipesMultipleDirs
  - id: CLM-035
    requirement: REQ-008
    subject: pkg/pack
    text: A recipes entry pointing at a missing directory (no recipe.yml) is a pack-manifest validation error
    tests:
      - TestPackManifest_RecipesIndexMissingDirErrors

  # REQ-009 — recipe.yml manifest fields + structural validation (kind matrix)
  - id: CLM-036
    requirement: REQ-009
    text: A well-formed recipe.yml (ops, scaffolding kind, param schema, target paths, transform rules, version) validates clean
    tests:
      - TestRecipeManifest_WellFormedValid
  - id: CLM-037
    requirement: REQ-009
    text: A recipe.yml missing its version field is a validation error
    tests:
      - TestRecipeManifest_MissingVersionErrors
  - id: CLM-038
    requirement: REQ-009
    text: A recipe.yml missing its ops list is a validation error
    tests:
      - TestRecipeManifest_MissingOpsErrors
  - id: CLM-039
    requirement: REQ-009
    text: A recipe.yml with a malformed (non-semver) version is a validation error
    tests:
      - TestRecipeManifest_MalformedVersionErrors
  - id: CLM-040
    requirement: REQ-009
    text: A recipe.yml with kind scaffolding validates clean
    tests:
      - TestRecipeManifest_KindScaffoldingValid
  - id: CLM-041
    requirement: REQ-009
    text: A recipe.yml with kind implementing validates clean
    tests:
      - TestRecipeManifest_KindImplementingValid
  - id: CLM-042
    requirement: REQ-009
    text: A recipe.yml with kind templating validates clean
    tests:
      - TestRecipeManifest_KindTemplatingValid
  - id: CLM-043
    requirement: REQ-009
    text: A recipe.yml with a kind outside {scaffolding, implementing, templating} is a validation error
    tests:
      - TestRecipeManifest_InvalidKindErrors
  - id: CLM-044
    requirement: REQ-009
    text: A recipe.yml declaring an optional compat matrix + version-keyed variants validates STRUCTURALLY (behavior out of scope)
    tests:
      - TestRecipeManifest_OptionalCompatVariantsValidateStructurally
  - id: CLM-064
    requirement: REQ-009
    text: A transform or insert op missing a non-empty manual field is a validation error (the injection-limit ops must declare the human-actionable fallback text)
    tests:
      - TestRecipeManifest_TransformInsertMissingManualErrors
  - id: CLM-065
    requirement: REQ-009
    text: A create/merge op with no manual field validates clean (manual is only required for the injection-limit op families)
    tests:
      - TestRecipeManifest_CreateMergeManualOptional
  - id: CLM-066
    requirement: REQ-009
    text: A transform op whose rule is not one of the recipe's declared transform rule files is a validation error (op-to-declared-rule cross-check)
    tests:
      - TestRecipeManifest_TransformOpUndeclaredRuleErrors
  - id: CLM-067
    requirement: REQ-009
    text: A transform op whose rule DOES appear in the recipe's declared transform rules validates clean
    tests:
      - TestRecipeManifest_TransformOpDeclaredRuleValid
  - id: CLM-068
    requirement: REQ-009
    text: A recipe with two ops sharing an id, or an injection-accepting (transform/insert) op with an empty id, is a validation error naming the recipe and the duplicate/empty id
    tests:
      - TestRecipeManifest_DuplicateOrEmptyOpIdErrors
  - id: CLM-069
    requirement: REQ-009
    text: A recipe whose op ids are unique (and non-empty on every injection-accepting op) validates clean
    tests:
      - TestRecipeManifest_UniqueOpIdsValid
  - id: CLM-076
    requirement: REQ-009
    text: >
      A substitution placeholder in `rule:` — and ONLY in `rule:` — is refused at manifest
      validation, fail-loud, naming the op index, the op id and the FIELD, and reported
      BEFORE the declared-rules allowlist cross-check; the SAME manifest with a literal
      `rule:` but a templated `payload:` still parses CLEANLY (the scope guard — `payload:`
      templatability is ISSUE-081's call, not this check's; `fragment:` is decided and is
      constrained separately by the fragment-form check, so it is NOT part of this guard),
      and a fully literal manifest parses cleanly too
    tests:
      - TestParseRecipeManifest_TemplatedTransformRuleFailsLoud
  - id: CLM-085
    requirement: REQ-009
    text: >
      A `merge` op whose `fragment:` value spans LINES — the inline block-scalar form — is
      refused at manifest validation, fail-loud, naming the op index, the op id and the
      FIELD, and stating the canon: `fragment` is a recipe-directory-relative path to a file
      read from disk, not inline content. The inline case is diagnosed AS inline even when it
      also carries a placeholder, so the message names the real defect rather than reporting
      a templated path
    tests:
      - TestParseRecipeManifest_InlineFragmentRefused
  - id: CLM-086
    requirement: REQ-009
    text: >
      A `merge` op whose single-line `fragment:` carries a `{{` placeholder is refused at
      parse — a fragment PATH is resolved under the recipe directory verbatim and never
      substituted, so a templated path can only fail to open — with the message directing the
      author to move the placeholder INTO the fragment file, whose content IS substituted.
      The rejection must come from the FRAGMENT check and not from the rule cross-check (the
      manifest declares its rule literally and lists it in `transform_rules`, so a
      rule-related diagnosis would mean the fragment check never ran). The complementary
      scope guard — the same manifest with a templated `payload:` still parsing cleanly —
      is CLM-076's
    tests:
      - TestParseRecipeManifest_TemplatedFragmentPathRefused
  - id: CLM-087
    requirement: REQ-009
    text: >
      A `merge` op whose `fragment:` is empty or whitespace-only is refused at parse — a
      merge op cannot merge nothing — rather than surviving to apply and surfacing there as
      the create/payload path's "declares no path" diagnostic, which names the wrong contract
    tests:
      - TestParseRecipeManifest_EmptyFragmentOnMergeOpRefused
  - id: CLM-094
    requirement: REQ-009
    text: >
      The MIRROR of the three refusals: a `merge` op declaring the canon form — a
      recipe-directory-relative fragment PATH — parses CLEAN and round-trips the declared
      string byte-exactly (it is handed to path resolution unchanged). Without this accept
      case the three refusals would be satisfied by a check that refused every fragment
    tests:
      - TestParseRecipeManifest_PathFormFragmentAccepted

  # REQ-010 — apply-time reference resolution of pack:recipe@version
  - id: CLM-045
    requirement: REQ-010
    text: A ref pack:recipe@version resolves to the pinned recipe via the pack's recipes index
    tests:
      - TestResolveRef_ResolvesPinnedRecipe
  - id: CLM-046
    requirement: REQ-010
    text: A ref naming a pack absent from the corpus fails loud
    tests:
      - TestResolveRef_MissingPackFailsLoud
  - id: CLM-047
    requirement: REQ-010
    text: A ref naming a recipe id absent from the pack's recipes index fails loud
    tests:
      - TestResolveRef_UndeclaredRecipeFailsLoud
  - id: CLM-048
    requirement: REQ-010
    text: A ref pinned to a @version that does not match the recipe's declared version fails loud
    tests:
      - TestResolveRef_NonexistentVersionFailsLoud
  - id: CLM-049
    requirement: REQ-010
    text: An unpinned or malformed ref (no @X.Y.Z) fails loud
    tests:
      - TestResolveRef_UnpinnedRefFailsLoud

  # REQ-011 (bundle REQ-021) — the injection limit: fail loud with the manual instruction
  - id: CLM-050
    requirement: REQ-011
    text: A transform op that cannot reach its target fails loud with a message whose instruction text EQUALS the op's declared manual field VERBATIM (plus a locator), never a core-synthesized instruction
    tests:
      - TestApply_TransformUnreachable_MessageEqualsDeclaredManualVerbatim
  - id: CLM-051
    requirement: REQ-011
    text: An unreachable transform is NEVER silently skipped — apply does not report success
    tests:
      - TestApply_TransformUnreachable_NeverSilentSkip
  - id: CLM-052
    requirement: REQ-011
    text: An unreachable transform NEVER guesses a site — no fallback write occurs
    tests:
      - TestApply_TransformUnreachable_NeverGuessesSite
  - id: CLM-053
    requirement: REQ-011
    text: An insert op whose declared anchor is absent fails loud with the op's declared manual field VERBATIM (the insert analog of the injection limit)
    tests:
      - TestApply_InsertMissingAnchor_MessageEqualsDeclaredManualVerbatim
  - id: CLM-078
    requirement: REQ-011
    text: >
      The relayed manual instruction is SUBSTITUTED and fails SOFT — on the injection-limit
      error path a manual templating a DECLARED param is relayed with the param resolved
      and carries no `{{`; the SAME manual templating an UNDECLARED param is relayed as the
      RAW declared text verbatim and the unreachable site is still identified (the
      falsifier for a naive substitute-everything-fail-loud implementation: the operator's
      instruction is never replaced by a second error on a path already reporting failure)
    tests:
      - TestApply_InjectionLimit_RelaysSubstitutedManualFailingSoftToRaw

  # REQ-012 (bundle REQ-023) — templating-kind one-shot / consumer-owned
  - id: CLM-054
    requirement: REQ-012
    text: A templating-kind recipe applied once produces consumer-owned output
    tests:
      - TestApply_Templating_OutputConsumerOwned
  - id: CLM-055
    requirement: REQ-012
    text: A templating-kind recipe on re-apply / pack upgrade is NOT regenerated over consumer changes
    tests:
      - TestApply_Templating_NotRegeneratedOnReapply
  - id: CLM-056
    requirement: REQ-012
    text: A consumer edit to templating-kind output survives a re-apply (regenerate-by-default does not apply to templating)
    tests:
      - TestApply_Templating_ConsumerEditSurvivesReapply

  # REQ-013 (bundle REQ-024) — strictly sequential deterministic multi-recipe apply
  - id: CLM-057
    requirement: REQ-013
    text: Multiple recipes apply strictly in the consumer's declared order (deterministic)
    tests:
      - TestApplyMulti_AppliesInDeclaredOrder
  - id: CLM-058
    requirement: REQ-013
    text: Same-file co-writes compose via merge in the declared order
    tests:
      - TestApplyMulti_SameFileCoWritesComposeViaMerge
  - id: CLM-059
    requirement: REQ-013
    text: The applier never reorders or interleaves recipes
    tests:
      - TestApplyMulti_NeverReordersOrInterleaves

contracts:
  - file: pkg/recipe/manifest.go
    provides:
      - name: RecipeManifest
        kind: type
        signature: "type RecipeManifest struct { Kind string; Version string; Params []ParamSpec; Ops []Op; TransformRules []string; Enforcement *EnforcementDecl; Compat []CompatSelector; Variants []Variant }"
        notes: "Parsed recipe.yml. Kind is one of scaffolding/implementing/templating (REQ-009), carried in source as the grouped untyped consts KindScaffolding/KindImplementing/KindTemplating. Those consts carry NO separate `provides` entry: a member of a grouped `const (…)` block is structurally inexpressible to the contracts pack's signature compiler, which handles only a standalone `const NAME = value` (the ast-grep pattern it emits does not bind a grouped block member) — the capability gap is ISSUE-078 (extending ISSUE-036's kind-aware compiler to const blocks), and the same disposition SPEC-035 v1.1.2 took for `CheckTypeFindings` applies: declaring an unverifiable entry buys a red, not a guarantee. The kind set is enforced BEHAVIORALLY and completely by the kind matrix — CLM-040/041/042 (each of the three kinds validates clean), CLM-043 (a kind outside the set is a validation error) — plus the ownership split CLM-019/CLM-020 (scaffolding+implementing regenerate) and CLM-054/055/056 (templating is one-shot). Compat/Variants are OPTIONAL and validated STRUCTURALLY only here — their apply-time behavior (REQ-015..017/020) is out of scope. Enforcement is the paired-suite DECLARATION (activation/scoping is REQ-013/014, out of scope)."
      - name: Op
        kind: type
        signature: "type Op struct { ID string; Kind string; Target string; Payload string; Fragment string; Format string; Rule string; Anchor string; Snippet string; Manual string }"
        notes: "One declared operation. ID is the stable op key SDLC-mediated InjectionSites is keyed by (REQ-003). Kind is a CLOSED allowlist: create/merge/transform/insert/step (REQ-002/REQ-007), carried in source as the grouped untyped consts OpCreate/OpMerge/OpTransform/OpInsert/OpStep. Those consts carry NO separate `provides` entry for the same reason RecipeManifest's kind consts do not — a grouped `const (…)` member is inexpressible to the contracts-pack signature compiler (ISSUE-078). The allowlist is enforced BEHAVIORALLY and exhaustively across every family: CLM-004 (create), CLM-005/006/007/008 (merge over the json/yaml/toml/.env matrix), CLM-010 (transform), CLM-011 (insert), CLM-028/029 (step recognized and sequenced, never executed), and CLM-030 (an op kind outside the set fails loud, never silently skipped). Rule (transform only) is a pack-relative rule-file path that MUST appear in the recipe's declared TransformRules — an op citing an undeclared rule is a manifest validation error (REQ-009/CLM-066). Fragment (merge only) is likewise a PATH — recipe-DIRECTORY-relative, resolved under ResolvedRecipe.Dir and read from disk VERBATIM, never substituted — the form ISSUE-081's Decision (2026-07-27) pinned; a declaration that can only be something else (an inline block, a `{{` placeholder, an empty value) is a manifest validation error (REQ-009/CLM-085/086/087), and the residual single-line inline value parse cannot distinguish from a filename fails at apply naming this contract (CLM-088). The fragment FILE'S CONTENT is still substituted (CLM-071): Fragment is the path, not the bytes. Payload's form remains ISSUE-081's residual open question and is deliberately unconstrained here. Manual is the human-actionable fallback instruction emitted VERBATIM when the injection limit is hit (REQ-011); it is REQUIRED for transform/insert and validated absent-is-error (CLM-064). A step op carries only its ID+Kind here and is never executed — its future payload schema is NOT round-tripped by the current non-strict YAML decode (unknown keys are dropped); BUNDLE-019 (which owns the step executor) will EXTEND this Op contract with the step payload fields (or a raw-passthrough), so this spec deliberately does not model them (Sharp Edges)."
      - name: ParamSpec
        kind: type
        signature: "type ParamSpec struct { Name string; Required bool; Default string }"
        notes: "One entry in the recipe's declared param schema; feeds {{ }} substitution (REQ-002) and direct-mode defaults (REQ-003)."
      - name: ParseRecipeManifest
        kind: function
        signature: "func ParseRecipeManifest(data []byte) (*RecipeManifest, error)"
        notes: "Parses + structurally validates recipe.yml: fail-loud on missing ops, missing/malformed-semver version, invalid kind, a transform/insert op missing its manual field (CLM-064), and a transform op whose rule is not among the declared TransformRules (CLM-066), a duplicate op id, an empty id on a transform/insert op (CLM-068), or a substitution placeholder in a transform op's `rule` — checked BEFORE the TransformRules cross-check and scoped to `rule:` ONLY, never payload (CLM-076) — and a merge op's `fragment:` that is not a PATH: empty/whitespace-only, spanning lines (the inline block form, diagnosed BEFORE the placeholder case so an inline block reports itself as inline), or carrying a `{{` placeholder, each with its own message naming the op index, id and field (CLM-085/086/087) while the canon path form parses clean (CLM-094) (REQ-009). The fragment predicate is deliberately not widened to a content sniff, so a single-line value that is syntactically a filename is accepted here and fails at apply instead (CLM-088); the check covers the recipe's own Ops only, not variant ops. Optional compat/variants validate structurally. No language knowledge — reads declared data."
    consumes:
      - source: gopkg.in/yaml.v3
        name: Unmarshal
        kind: function
  - file: pkg/recipe/resolve.go
    provides:
      - name: RecipeRef
        kind: type
        signature: "type RecipeRef struct { Pack string; Recipe string; Version string }"
        notes: "A parsed <pack>:<recipe>@<recipe_version> reference (REQ-010)."
      - name: ParseRecipeRef
        kind: function
        signature: "func ParseRecipeRef(raw string) (RecipeRef, error)"
        notes: "Parses the pinned ref shape; an unpinned or malformed ref (no @X.Y.Z) is an error (CLM-049)."
      - name: ResolvedRecipe
        kind: type
        signature: "type ResolvedRecipe struct { Ref RecipeRef; Dir string; PackDir string; Manifest *RecipeManifest }"
        notes: "A ref resolved to a recipe directory + parsed manifest (REQ-010). Dir is the RECIPE directory (recipe.yml + payloads); PackDir is the enclosing PACK root, carried because Op.Rule is a PACK-relative rule-file path (Op contract, REQ-009) — the transform executor resolves the rule under PackDir, which is what lets one pack share a rule file across several of its recipes. Both are set by ResolveRecipe; nothing downstream re-derives either."
      - name: ResolveRecipe
        kind: function
        signature: "func ResolveRecipe(ref RecipeRef, packs map[string]*pack.Manifest, packDir string) (*ResolvedRecipe, error)"
        notes: "Apply-time resolution (REQ-010): fail-loud on a missing pack, a recipe id absent from the pack's recipes: index, or a @version mismatch with the recipe's declared version. Reads the pack.Manifest.Recipes index. Sets both ResolvedRecipe.Dir (packDir joined with the index's declared directory) and ResolvedRecipe.PackDir (the packDir argument verbatim)."
    consumes:
      - source: pkg/pack
        name: Manifest
        kind: type
  - file: pkg/recipe/apply.go
    provides:
      - name: ApplyMode
        kind: type
        signature: "type ApplyMode string"
        notes: "The mode selector (REQ-003) — it selects only WHERE an op's WHERE comes from (recipe-declared defaults vs supplied InjectionSites); both modes drive the same op executors. Declared here as the named TYPE rather than as its two grouped value consts (ModeDirect = \"direct\", ModeSDLCMediated = \"sdlc-mediated\"), because a grouped `const (…)` member is structurally inexpressible to the contracts-pack signature compiler while `type ApplyMode string` is verifiable — this is the same shape/ISSUE-078 disposition the RecipeManifest and Op kind consts take, except that here a verifiable form of the surface exists, so the type is declared rather than the entry dropped. The two VALUES are enforced BEHAVIORALLY: CLM-015 (direct self-applies from recipe-declared defaults) and CLM-016 (direct is byte-identical across runs) for ModeDirect; CLM-017 (an injection-accepting op applies at the supplied InjectionSites[op-id], falsifiably) and CLM-060 (a transform/insert with neither a declared target/anchor nor a supplied site fails loud) for ModeSDLCMediated."
      - name: ApplyOptions
        kind: type
        signature: "type ApplyOptions struct { Mode ApplyMode; Params map[string]string; InjectionSites map[string]string; ProjectRoot string; Dispatch TransformDispatch; ReadWaivers WaiverReader }"
        notes: "Direct mode reads Params/defaults; SDLC-mediated mode reads InjectionSites — a map KEYED BY op id whose value is the WHERE (target/anchor) for the injection-accepting transform/insert ops only (REQ-003). Dispatch is the transform-engine seam (REQ-006). ReadWaivers is the waiver-adjudication seam (REQ-004); the production impl calls the real pkg/waiver read path (waiver.Adjudicate over a waiver.LineReader on the consumer's file). `ReadWaivers` now yields a `DivergenceVerdict` rather than a bool (ISSUE-080) — the field's TYPE NAME is unchanged, so this signature line is unchanged."
      - name: TransformDispatch
        kind: type
        signature: "type TransformDispatch func(rule string, target string) error"
        notes: "The injected transform-engine dispatch seam — and the ONLY transform-engine seam pkg/recipe has. pkg/recipe itself does NOT call engine.CheckToolAllowed: no type in this package carries a tool name or a locked version, so the trust gate is unimplementable here. The production Dispatch is built in cmd/backstop/recipe_apply.go, which runs the gate BEFORE constructing the closure, so an un-allowlisted tool's command is never built (REQ-006). Kept as a seam so the reject path is exercised on the real gate (CLM-025, a cmd/backstop test) without stubbing the allowlist open."
      - name: WaiverReader
        kind: type
        signature: "type WaiverReader func(rule string, file string) DivergenceVerdict"
        notes: "The waiver-adjudication seam (REQ-004): given the recipe's declared enforcement rule and a diverged path, it reports whether a covering @waiver is ACTIVE **and what the adjudication FOUND** — the coverage answer and the diagnostics travel together. It returns a VERDICT rather than a bool plus an optional reporting hook because an optional hook is nil by default and nil means \"drop it\" — reproducing by shape the very silent drop ISSUE-080 removed. Both facts come from ONE waiver.Result, so splitting them across two seams would let them disagree with nothing to catch it. The production impl builds a waiver.Finding for the divergence and adjudicates it via waiver.Adjudicate fed a waiver.LineReader over the on-disk file — it is still a READ path only and NEVER writes a token (CLM-062 unregressed). Kept as a seam so tests can drive the real read path (CLM-021) and the no-waiver path (CLM-061) without a stub-open bypass."
      - name: DivergenceVerdict
        kind: type
        signature: "type DivergenceVerdict struct { Covered bool; Diagnostics []waiver.Diagnostic }"
        notes: "The value the WaiverReader seam returns (REQ-004). It exists because narrowing adjudication to a bare `bool` silently DROPPED `waiver.DiagnosticMalformed`: a malformed @waiver token and an ABSENT token became indistinguishable at the decision point, so a hand-edited file marked with an unparseable token was regenerated over at exit 0, in silence (ISSUE-080). `Covered` is the coverage answer; `Diagnostics` carries what adjudication REPORTED — never anything the applier synthesized, on the same terms as `PreservedDivergence.CoveringWaiver`."
      - name: ApplyResult
        kind: type
        signature: "type ApplyResult struct { Written []string; Preserved []PreservedDivergence; Regenerated []string; Diagnostics []waiver.Diagnostic; Adoption AdoptionEntry }"
        notes: "Records what an apply wrote, the recipe-owned files PRESERVED because a covering active waiver was READ (each with the covering waiver that accounted for it — REQ-004; never a waiver the applier authored), which of the writes overwrote a divergence, what adjudication reported non-fatally, and the thin adoption entry (REQ-005). `Regenerated` is the strict SUBSET of `Written` whose diverged bytes were overwritten with NO covering waiver, recorded in the SAME value form `recordWritten` uses (the substituted declared target), so the subset property survives templated targets. `Diagnostics` carries the non-fatal adjudication findings the CLI surfaces as warnings on stderr. Both are REPORTING channels the CLI reads, never inputs to a decision. `Written` carries the recipe-declared target AFTER param substitution (REQ-002) — still the recipe's own path FORM (project-relative), never the applier's absolute resolution of it, and never a `{{`-bearing string: reporting `{{ config_dir }}/service.json` to an operator whose file landed at `config/service.json` names a path that does not exist. The same substituted form is the ownership key an apply uses to recognize a file it wrote in this run, because two ops templating differently can legitimately resolve to one file. `pkg/recipe/apply.go`'s ApplyResult doc comment states this identically and is updated in the SAME change — the two statements of the contract must not drift apart (CLM-002/CLM-070)."
      - name: PreservedDivergence
        kind: type
        signature: "type PreservedDivergence struct { Path string; Rule string; CoveringWaiver string }"
        notes: "One recipe-owned file left in place on re-apply because the divergence finding was adjudicated as covered by an ACTIVE waiver read from the consumer's file (REQ-004/CLM-021). CoveringWaiver is the token that was READ, not one the applier wrote. `Path` carries the recipe-declared target AFTER param substitution (REQ-002), on the same terms as `ApplyResult.Written`: the recipe's own project-relative path form, never the applier's absolute resolution and never a `{{`-bearing string — an operator asked to account for a divergence must be given a path that exists."
      - name: Apply
        kind: function
        signature: "func Apply(resolved *ResolvedRecipe, opts ApplyOptions) (ApplyResult, error)"
        notes: "Runs the recipe's ops in declared order (REQ-001); dispatches per op family (REQ-002/REQ-007); resolves every consumer-facing site/content field an op declares — target, anchor, snippet, payload/fragment content, and the relayed manual — through Substitute BEFORE using it to locate a path, match an anchor, splice bytes or report a result, while leaving Op.Rule RAW (REQ-002 scope; a templated rule is refused at parse instead, REQ-009); non-destructive toward user files, and on re-apply of recipe-owned output computes the would-be-regenerated bytes, diffs on-disk, and PRESERVES-on-covered-waiver (read via opts.ReadWaivers) / REGENERATES-otherwise, never authoring a token (REQ-004); one-shot for templating (REQ-012); fail-loud with the op's declared manual text VERBATIM on an unreachable transform/insert (REQ-011); writes the thin adoption record (REQ-005). Zero language/platform/CI literals (REQ-006)."
      - name: ApplyAll
        kind: function
        signature: "func ApplyAll(resolved []*ResolvedRecipe, opts ApplyOptions) ([]ApplyResult, error)"
        notes: "Strictly sequential, deterministic multi-recipe apply in the given declared order; never reorders/interleaves; same-file co-writes compose via merge in order (REQ-013)."
    consumes:
      - source: pkg/waiver
        name: Adjudicate
        kind: function
      - source: pkg/waiver
        name: LineReader
        kind: type
      - source: encoding/json
        name: Unmarshal
        kind: function
      - source: gopkg.in/yaml.v3
        name: Unmarshal
        kind: function
      - source: github.com/pelletier/go-toml/v2
        name: Unmarshal
        kind: function
  - file: pkg/recipe/substitute.go
    provides:
      - name: Substitute
        kind: function
        signature: "func Substitute(template string, params map[string]string) (string, error)"
        notes: "Declarative {{ param }} value interpolation, NOT Turing-complete (REQ-002): resolves declared params only, no logic/expression evaluation, fail-loud on an undeclared placeholder (CLM-013/CLM-014)."
  - file: pkg/recipe/adoption.go
    provides:
      - name: AdoptionEntry
        kind: type
        signature: "type AdoptionEntry struct { Recipe string; Version string; Adopted string }"
        notes: "The thin, tracked adoption record entry {recipe ref, @version, adopted} (REQ-005). No rich per-op/per-region fields (that is BUNDLE-017's)."
      - name: AdoptionRecord
        kind: type
        signature: "type AdoptionRecord struct { Recipes map[string]AdoptionEntry }"
        notes: "The recipe analog of backstop.lock: a tracked project-root file of adoption entries."
      - name: ReadAdoptions
        kind: function
        signature: "func ReadAdoptions(path string) (*AdoptionRecord, error)"
        notes: "Reads the tracked adoption record (missing file yields an empty record, not an error — same shape as ReadLockfile)."
      - name: WriteAdoptions
        kind: function
        signature: "func WriteAdoptions(path string, rec *AdoptionRecord) error"
        notes: "Writes the adoption record deterministically (sorted keys), rhyming with distribution.WriteLockfile."
  - file: pkg/pack/manifest.go
    provides:
      - name: Manifest.Recipes
        kind: variable
        signature: "Recipes map[string]string `yaml:\"recipes\"`"
        notes: "NEW field on the existing Manifest struct: the lightweight recipes: index (stable recipe-id -> directory), a distinct top-level key from Content.Scaffolds (REQ-008). Optional; zero-value (nil) when absent."
      - name: validateRecipesIndex
        kind: function
        signature: "func validateRecipesIndex(m *Manifest, packRoot string) error"
        notes: "Called from ParseManifest/pack validation: fail-loud on a recipes: entry pointing at a missing directory or one lacking a recipe.yml (CLM-035). The check is STRUCTURAL ONLY — directory exists under packRoot and contains a recipe.yml — and pkg/pack MUST NOT import pkg/recipe to do it: pkg/recipe imports pkg/pack (ResolveRecipe takes map[string]*pack.Manifest), so parsing recipe.yml here would invert that dependency into an import cycle. recipe.yml CONTENT is validated at resolve time by pkg/recipe.ParseRecipeManifest. Does not collide with the existing Content.Scaffolds validation."
    consumes:
      - source: os
        name: Stat
        kind: function
  - file: cmd/backstop/recipe_apply.go
    provides:
      - name: runRecipeApply
        kind: function
        signature: "func runRecipeApply(ref string, projectRoot string) (recipe.ApplyResult, error)"
        notes: "Thin CLI wiring for `backstop recipe apply <pack:recipe@version>`: parses+resolves the ref, selects the pack's single provisioned engine binding from declared data, RUNS THE TRUST GATE (checkEngineToolAllowed → engine.CheckToolAllowed over resolveTrustedToolAllowlist, the SAME gate the enforcement dispatch uses — REQ-006) and only then builds the production TransformDispatch, so an un-allowlisted tool's command is never constructed; runs Apply and writes the adoption record. It RETURNS the applier's result alongside the error (ISSUE-080) because the success line is printed in the cobra RunE closure, from which the result was otherwise unreachable — so the CLI now reports what the apply ACTUALLY did from DATA rather than printing one static line: each WRITTEN target, marking any that overwrote a divergence no active waiver accounted for; each PRESERVED path with the covering token, or marked as the consumer's own file; and each non-fatal adjudication diagnostic as a `warning:` on STDERR while the report itself goes undivided to stdout. Every failure path returns the ZERO result alongside the error, matching Apply's own contract (a run either produces a verdict or it fails, never both). The ConfigError-vs-violation exit split is UNCHANGED. THIS FILE IS WHERE THE TRUST GATE LIVES — pkg/recipe cannot host it (no type there carries a tool or a locked version). ReadWaivers is left nil so apply.go's own real pkg/waiver read path is used rather than forking adjudication into a second implementation (REQ-004). The dogfoodable surface the E2E real-engine transform test (CLM-063) drives, proving the transform-dispatch seam runs a REAL allowlisted engine (ast-grep) end to end — a wired-but-no-op dispatch would fail it."
    consumes:
      - source: pkg/recipe
        name: Apply
        kind: function
      - source: pkg/recipe
        name: ResolveRecipe
        kind: function
      - source: pkg/pack/engine
        name: CheckToolAllowed
        kind: function
      - source: pkg/pack/engine
        name: EngineBinding
        kind: type
      - source: pkg/check
        name: ExecCommandRunner
        kind: type
---

# SPEC-054: Recipe Apply And Manifest

## Overview

Backstop is a thin executor: detection, framework/language, and CI-platform knowledge
live in PACKS as data, never in core (`backstop/self` enforces it). But onboarding must
SCAFFOLD things that are inherently language- and platform-specific — a starter project
skeleton, a `.github/workflows/backstop.yml` gate workflow, a canonical client. The only
way to do that without baking knowledge into core is for PACKS to carry recipes (data)
and for a GENERIC core mechanism to apply them. That mechanism does not exist yet, and it
is the BLOCKING dependency `backstop init` (BUNDLE-003 DD-12) consumes but cannot be built
without.

This spec delivers the coupled first surface of BUNDLE-015: the **recipe APPLY mechanism**
and the **MANIFEST DECLARATION** it resolves against. A recipe is a per-recipe DIRECTORY
(`recipe.yml` + template payload + `transform`-rule files) indexed from a lightweight
`pack.yml` `recipes:` block. A new `pkg/recipe` package resolves a recipe by
`<pack>:<recipe>@<recipe_version>` and runs its declared ORDERED ops — `create` / `merge`
/ `transform` / `insert` — with declarative `{{ }}` substitution that is explicitly NOT
Turing-complete, reserving a fifth `step` op as an opaque sequencing seam. The genuinely
new primitive is GENERATION; everything that makes a recipe *backstop* reuses existing
substrate — the ENGINES (`transform` is match->fix on the same declared-rule model
enforcement's match->finding uses), the WAIVER (accountable divergence), and the
`backstop.lock`-shaped tracked record (here a THIN adoption record).

### Scope

Covers BUNDLE-015 **REQ-001..REQ-010, REQ-021, REQ-023, REQ-024** — the apply mechanism
and manifest declaration seeds — mapped to spec REQ-001..REQ-013 below.

Explicitly **OUT OF SCOPE** (named so downstream specs/bundles own them):

| Deferred | Owner |
|----------|-------|
| Publish-time rev-guard (bundle REQ-011) | later BUNDLE-015 spec (pack-authoring tooling, outside core) |
| The three drift signals (bundle REQ-012) | later BUNDLE-015 spec + BUNDLE-017 |
| Enforcement activation / static scope / un-adopt (bundle REQ-013/014/022) | Recipe-enforcement-scoping spec |
| Compat matrix / version-keyed variants / migration / chain traversal (bundle REQ-015/016/017/020) | Compat+variants+migration spec |
| CI recipe pack + provider standup packs (bundle REQ-018/019) | backstop-packs (not core) |
| The `step` EXECUTOR + probe-receipt/precondition engine | BUNDLE-019 (Runbooks) |
| Rich per-op/per-region provenance ledger + dynamic-`transform` output scoping | BUNDLE-017 |

This spec reserves the `step` op *seam* (REQ-007) and the *structural* validation of
compat/variants blocks (REQ-009); it does NOT execute steps or resolve variants.

## Requirements

Requirements REQ-001 through REQ-013 are defined in frontmatter and trace to BUNDLE-015
REQ-001..REQ-010, REQ-021, REQ-023, REQ-024 via pinned `supports` refs. Each requirement
has at least one claim; claims are defined in frontmatter.

### Op-family allowlist (REQ-002 / REQ-007)

The op-family set is a CLOSED allowlist. The applier recognizes exactly these kinds; every
cell below is covered by a claim (positive dispatch and negative reject):

| Op kind | Applier behavior | Claim |
|---------|------------------|-------|
| `create` | materialize templated file/tree at declared target | CLM-004 |
| `merge` | deep-merge fragment into a structured file | CLM-005..CLM-008 |
| `transform` | dispatch AST rewrite to an allowlisted engine (declared rule) | CLM-010 |
| `insert` | snippet at declared anchor | CLM-011 |
| `step` | RECOGNIZED + sequenced, but NOT executed here (reserved seam) | CLM-028 / CLM-029 |
| anything else | FAIL LOUD (never silently skipped) | CLM-030 |

### Merge target format (REQ-002)

`merge` supports exactly the universal structured formats; an unsupported format fails loud:

| Target format | `merge` behavior | Claim |
|---------------|------------------|-------|
| json | deep-merge | CLM-005 |
| yaml | deep-merge | CLM-006 |
| toml | deep-merge | CLM-007 |
| .env | merge | CLM-008 |
| anything else (unstructured/unknown) | FAIL LOUD | CLM-009 |

### Recipe kind → post-generation ownership (REQ-004 / REQ-012)

The recipe KIND selects the regenerate model. Every kind is covered:

| Kind | Re-apply behavior | Claim |
|------|-------------------|-------|
| scaffolding | regenerate-by-default (recipe-owned output reproduced) | CLM-019 |
| implementing | regenerate-by-default (recipe-owned output reproduced) | CLM-020 |
| templating | ONE-SHOT — consumer-owned, never regenerated over consumer changes | CLM-054..CLM-056 |

USER-OWNED files (files the recipe does not own) are NEVER clobbered by any kind (CLM-018);
divergence from *recipe-owned* output is an accountable `@waiver`, never a bespoke merge
(CLM-021).

### Substitution (REQ-002)

`{{ param }}` is pure value interpolation resolving declared params only:

| Case | Result | Claim |
|------|--------|-------|
| declared `{{ param }}` | substituted from params/defaults | CLM-012 |
| logic/expression construct in placeholder | never evaluated as code (not Turing-complete) | CLM-013 |
| undeclared placeholder | FAIL LOUD (never silently blanked) | CLM-014 |

#### Substitution SCOPE — which `Op` fields are resolved (REQ-002)

The semantics above say HOW a placeholder resolves; this matrix says WHERE. It is a
closed, field-by-field allowlist: a field is either SUBSTITUTED before use or it is
named here as deliberately not. Every row is covered by a claim that drives the real
`Apply` (or the real parse, or the real CLI) — never `Substitute` in isolation.

| `Op` field | Families | Substituted? | Failure policy | Claim |
|------------|----------|--------------|----------------|-------|
| `Op.Target` | create / merge / transform / insert | YES — before path resolution, existence check, read, write, ownership keying and result reporting | FAIL LOUD | CLM-070 / CLM-071 / CLM-073 / CLM-077 |
| `Op.Anchor` | insert (INCLUDING the half supplied through SDLC-mediated `InjectionSites`) | YES — before the anchor match | FAIL LOUD | CLM-070 / CLM-073 / CLM-074 |
| `Op.Snippet` | insert | YES — before the splice | FAIL LOUD | CLM-070 / CLM-073 |
| `Op.Manual` | transform / insert (relayed operator instruction) | YES — before it is relayed | FAIL **SOFT** → relay the RAW declared text | CLM-078 |
| payload / fragment CONTENT | create / merge | YES — before the bytes are written or merged (true since 1.0.0) | FAIL LOUD | CLM-070 / CLM-071 |
| `Op.Rule` | transform | **NO** — the declared, allowlist-validated path is executed unchanged | a placeholder in `rule:` is REFUSED at manifest validation (REQ-009) | CLM-075 / CLM-076 |
| `Op.Fragment` as a PATH | merge | **NO** — the declared path is resolved under the recipe directory and read from disk unchanged | a non-path value (inline content, a `{{` placeholder, or empty) is REFUSED at manifest validation (REQ-009); the residual single-line inline value parse cannot distinguish from a filename fails at APPLY naming the path-only contract | CLM-071 / CLM-085 / CLM-086 / CLM-087 / CLM-088 |

**The path/content split, stated so the last two rows are not read as contradicting each
other.** `Op.Fragment` is a PATH and is never substituted; the FILE it names has its
CONTENT substituted, exactly as it has since 1.0.0. A recipe author who wants a placeholder
in a merge fragment puts it inside the fragment file, never in the field that locates it.
The same split is what makes the parse-time refusal safe: refusing a templated `fragment:`
takes nothing away from the author, because the field was never a place a placeholder could
work — a brace-bearing path is resolved verbatim and can only fail to open. CLM-071 proves
both halves in one run over the committed `starter` recipe: the declared path is read
verbatim and its content arrives substituted.

`Op.Payload` as a PATH is deliberately **OUT OF SCOPE of this table**. Its form and
templatability remain ISSUE-081's open question. The omission is a DECISION, not a gap:
deciding it here would settle ISSUE-081 silently and from the wrong place, so this spec
takes no position either way. The asymmetry with `fragment:` is itself the reason —
`fragment:` had a LIVE three-way contradiction (the applier read a path, this spec said
nothing, and a committed fixture declared an inline block), so it was decided on evidence
by ISSUE-081's Decision (2026-07-27); `payload:` has no such contradiction, and settling it
here would be deciding from the wrong place on no evidence. (Payload CONTENT, once read, is
substituted; that is the content row above and is a separate matter from the path's form.)

**Why `Op.Rule` is not substituted — and why the reason differs from `Op.Fragment`'s.**
Both are paths refused-at-parse rather than substituted, but for DIFFERENT reasons, and
collapsing them would lose the stronger one. `fragment:` is verbatim because a substituted
path could only fail to open — there is nothing a placeholder there could usefully do.
`rule:` is verbatim for an AUTHORITY reason that would hold even if the substituted path
resolved perfectly. The rule path is validated at PARSE time by exact
string equality against the recipe's declared `transform` rules allowlist (REQ-009), and
it selects which PACK ASSET an allowlisted engine executes IN PLACE over the consumer's
tree. Substituting it at apply time would (a) execute a rule path that is not the one
validation approved — breaking, by construction, the declared-vs-executed correspondence
the allowlist exists to guarantee — and (b) hand a CONSUMER-supplied param authority over
which of the recipe's own code assets runs. Params are inputs to the recipe's OUTPUT,
never a selector for the recipe's own code. Because `rule:` is excluded from
substitution, its placeholder hole is closed at the earliest point instead: a `{{` in
`rule:` is a fail-loud manifest-validation error naming the op and the field (CLM-076),
scoped to `rule:` alone.

**Why `Op.Manual` fails SOFT.** The manual is emitted on ONE path only — the
injection-limit failure (REQ-011), where the loud failure has already happened. Relaying
literal braces to an operator is the same defect class in the diagnostic channel, so the
manual IS substituted; but a substitution failure there must not REPLACE the operator's
instruction with a second error. On failure the RAW declared text is relayed unchanged
and the unreachable site is still identified (CLM-078). This is the only field in the
matrix where fail-loud is the wrong policy, and the reason is precisely that the loud
failure is already in flight.

The invariant the matrix exists to produce: after a successful apply, **no path beneath
the project root carries a `{{` or `}}` segment** and no spliced bytes do either (CLM-072,
asserted by walking the tree rather than checking a known-good path list, so a future op
family that forgets substitution is caught without anyone updating a list).

### Reference resolution (REQ-010)

`<pack>:<recipe>@<recipe_version>` resolves iff all hold; any failure is a fail-loud error:

| Check | Fails when | Claim |
|-------|-----------|-------|
| Named pack exists | ref names a pack absent from the corpus | CLM-046 |
| Recipe id declared in the pack's `recipes:` index | ref cites a recipe the pack never indexed | CLM-047 |
| Pinned `@version` matches the recipe's declared version | pin names a version the recipe does not declare | CLM-048 |
| Ref is pinned + well-formed | ref is unpinned or malformed (no `@X.Y.Z`) | CLM-049 |

### Recipe manifest structural validation (REQ-009)

| Field | Required | Constraint | Claim |
|-------|----------|-----------|-------|
| `ops` | yes | non-empty; each op kind in the allowlist | CLM-038 |
| `version` | yes | well-formed semver `MAJOR.MINOR.PATCH` | CLM-037 / CLM-039 |
| `kind` | yes | one of scaffolding / implementing / templating | CLM-040..CLM-043 |
| `params`, `targets`, `transform` rules, `enforcement` | as declared | structural | CLM-036 |
| `compat`, `variants` | no | structural only (behavior out of scope) | CLM-044 |
| op `manual:` | yes for `transform`/`insert`, else no | non-empty on injection-limit ops; optional on `create`/`merge` | CLM-064 / CLM-065 |
| op `rule:` (transform) | yes for `transform` | pack-relative path present in the recipe's declared `transform` rules | CLM-066 / CLM-067 |
| op `rule:` placeholder (transform) | n/a | a `{{` in `rule:` — and ONLY in `rule:` — is refused, before the declared-rules cross-check; a `payload:` placeholder is NOT checked here (ISSUE-081's residual open question) | CLM-076 |
| op `fragment:` (merge) | yes for `merge` | a recipe-DIRECTORY-relative path; an inline (multi-line) value, a `{{` placeholder, or an empty/whitespace-only value is a fail-loud error naming the op index, id and field, each with its own message; the canon path form parses clean. Scope is the recipe's own `ops`, not variant ops | CLM-085 / CLM-086 / CLM-087 / CLM-094 |
| op `id:` | unique always; non-empty on `transform`/`insert` | duplicate id, or empty id on an injection-accepting op, is an error (it is the `InjectionSites` routing key — REQ-003) | CLM-068 / CLM-069 |

## Implementation

### Package layout

- `pkg/recipe/manifest.go` (NEW) — `RecipeManifest`, `Op`, `ParamSpec`, kind/op-kind
  constants, and `ParseRecipeManifest` with structural validation (REQ-009). The kind and
  op-kind constants ship as grouped `const (…)` blocks and therefore carry no standalone
  contract `provides` entry (ISSUE-078 — the contracts-pack signature compiler binds only a
  standalone `const NAME = value`); both sets are enforced behaviorally by the kind and
  op-family matrices, and the reasoning is recorded on the `RecipeManifest` and `Op` contract
  notes rather than as an unverifiable declaration.
- `pkg/recipe/resolve.go` (NEW) — `RecipeRef`, `ParseRecipeRef`, `ResolveRecipe`
  (apply-time reference resolution, REQ-010), reading `pack.Manifest.Recipes`. `ResolvedRecipe`
  carries BOTH the recipe directory (`Dir`) and the enclosing pack root (`PackDir`), because
  `Op.Rule` is pack-relative.
- `pkg/recipe/apply.go` (NEW) — `Apply` (single recipe, REQ-001/002/004/005/006/007/011/012)
  and `ApplyAll` (sequential multi-recipe, REQ-013), the two `ApplyMode`s (REQ-003), and the
  `TransformDispatch` seam (REQ-006).
- `pkg/recipe/substitute.go` (NEW) — `Substitute`, the non-Turing `{{ }}` interpolation
  (REQ-002).
- `pkg/recipe/adoption.go` (NEW) — `AdoptionEntry` / `AdoptionRecord` + read/write, the thin
  tracked record rhyming with `pkg/pack/distribution/lockfile.go` (REQ-005).
- `pkg/pack/manifest.go` (EXTEND) — add the `Recipes map[string]string` field (the
  `recipes:` index) and `validateRecipesIndex`, wired into `ParseManifest` next to (never
  colliding with) the existing `Content.Scaffolds` handling (REQ-008).
- `cmd/backstop/recipe_apply.go` (NEW) — thin `backstop recipe apply` wiring that RUNS the
  concrete trusted-tool allowlist gate and only then builds the production `TransformDispatch`,
  so the transform dispatch clears the SAME `engine.CheckToolAllowed` gate as enforcement
  (REQ-006). This file, not `pkg/recipe`, is where the trust gate lives.

### Transform dispatch reuses the existing engine trust gate (REQ-006)

A `transform` op is an AST rewrite (match->fix); backstop's existing enforcement dispatch is
an AST read (match->finding). They are the SAME shape: a general engine runs a DECLARED rule,
and the language-awareness lives in the engine, not core. The `transform` executor MUST NOT
introduce a second dispatch path — it routes through `engine.CheckToolAllowed`
(pkg/pack/engine), the identical trust gate `checkEngineToolAllowed`
(cmd/backstop/pack_gate.go:812) runs before dispatching an enforcement engine, and the
identical allowlist resolver (`resolveTrustedToolAllowlist`, cmd/backstop/pack_gate.go:46).
An un-allowlisted or non-lock-pinned engine tool is rejected as a `*check.ConfigError`
(exit 2) exactly as on the enforcement path.

**The gate lives at the CLI layer, and `pkg/recipe`'s only seam is `opts.Dispatch`.**
`pkg/recipe` does NOT call `engine.CheckToolAllowed`: no type in that package carries a tool
name or a locked version, so the gate is unimplementable there. `cmd/backstop/recipe_apply.go`
selects the pack's single provisioned engine binding from declared data, runs
`checkEngineToolAllowed` (→ `engine.CheckToolAllowed` over `resolveTrustedToolAllowlist`), and
only THEN builds the `TransformDispatch` closure — the same gate-before-command-construction
order `runFindingsEngine` uses, so an un-allowlisted tool's command is never constructed. The
closure runs the engine through `check.CommandRunner` (`check.ExecCommandRunner` with
`Dir = projectRoot`), the same runner the enforcement dispatch uses; it is NOT the
convert-step sandbox runner, whose deny-all-writes profile would make a `transform` — which by
definition writes a consumer file — structurally impossible. Keeping the dispatch injected
lets `pkg/recipe`'s op-level tests drive transform behavior without stubbing any gate open,
while the reject path itself (CLM-025) is tested at `cmd/backstop`, the only layer that can
see the pack's `engines:` block.

An op's `rule` is a PACK-relative path, so the transform executor resolves it under
`ResolvedRecipe.PackDir` (the pack root `ResolveRecipe` carried through), not under the recipe
directory — that is what lets one pack share a rule file across several of its recipes.

### The `merge` format matrix and its codecs (REQ-002)

The closed merge allowlist is {json, yaml, toml, .env}; anything else fails loud (never a text
append). The three STRUCTURED members decode to a generic tree, deep-merge name by name, and
re-encode with the same codec, so the merged file is the codec's canonical rendering of the
union; `.env` has no tree to decode and is merged over its KEY/VALUE LINE set instead. The
codecs are DATA-SHAPE readers, not language or tool knowledge (REQ-006): none of them implies
which language or toolchain wrote the file, and the recipe still supplies every path, fragment,
and format token.

Two of the four codecs are already in core (`encoding/json` from the standard library,
`gopkg.in/yaml.v3` from pack-manifest parsing) and `.env` needs none. TOML does — so this spec
promotes **`github.com/pelletier/go-toml/v2` to a DIRECT dependency of the module** (it was
previously only an indirect one, pulled in through the lint toolchain). That is recorded here
and in `pkg/recipe/apply.go`'s declared `consumes` edges rather than arriving silently in
`go.mod`: a new direct dependency is a contract-surface change, and the merge matrix's TOML
cells (CLM-007 and the format matrix) are what require it.

### Adoption record (REQ-005)

`AdoptionRecord` is the recipe analog of `backstop.lock`: a tracked project-root file whose
entries are exactly `{recipe ref, @version, adopted}`. `ReadAdoptions`/`WriteAdoptions`
mirror `distribution.ReadLockfile`/`WriteLockfile` — a missing file yields an empty record
(not an error), and writes are deterministic (sorted keys). It carries NONE of the rich
per-op/per-region provenance (BUNDLE-017 owns that). It is the applied-`@version` carrier the
downstream drift signals (out of scope) will read.

### Regenerate-vs-waiver mechanics reuse the REAL waiver read path (REQ-004)

The accountable-divergence hinge must not be a string appended to `ApplyResult`. On re-apply of
recipe-owned output the applier computes the would-be-regenerated bytes and diffs against on-disk;
a divergence is turned into a `waiver.Finding` (RuleID = the recipe's declared enforcement rule for
that path, File = the diverged path) and adjudicated through the REAL `pkg/waiver` read path —
`waiver.Adjudicate(findings, lineReader, policy, now)` (pkg/waiver/adjudicate.go:86) fed a
`waiver.LineReader` (pkg/waiver/adjudicate.go:27) that yields the consumer file's raw lines. If the
finding lands in `Result.Suppressed` (a covering `@waiver` token is present and active), the file is
PRESERVED and the covering waiver recorded in `ApplyResult.Preserved`; otherwise it is REGENERATED.
The applier calls NO waiver-authoring API — `waiver.ParseToken`/`Adjudicate` are read-only, and the
`<reason>`/`<expiry>` are human judgments the consumer supplies by adding the token FIRST. In
`pkg/recipe` this is the injected `WaiverReader` seam so CLM-021 (covered → preserved) and CLM-061
(uncovered → regenerated) drive the real adjudication, not a stub-open bypass; `cmd/backstop`
wires the concrete reader.

### The manual instruction is DATA, never synthesized (REQ-011)

Core cannot author the "wire it in by hand like THIS" text without knowing the target language and
framework — exactly the knowledge REQ-006 forbids in core. So each `transform`/`insert` op declares
a `manual:` string, `ParseRecipeManifest` requires it on those op families, and the injection-limit
failure emits that declared string VERBATIM (plus an op-id + intended-target locator). CLM-050/053
assert the emitted message EQUALS the declared field verbatim — a synthesized instruction would
fail them.

### The E2E real-engine transform binding (REQ-006, the anti-stub-green guard)

A wired-but-no-op `TransformDispatch` passes every op-level test while doing nothing — the exact
stub-green trap. `TestRecipeApply_E2E_RealEngineTransform_CapturedFixture` (cmd/backstop) closes it:
it runs the actual `backstop recipe apply` CLI over a recipe whose `transform` op declares a real
ast-grep rewrite rule, against a CAPTURED before-fixture, and asserts the on-disk file content
equals the captured after-state. The engine is real (allowlisted ast-grep through
`engine.CheckToolAllowed`), the fixture is captured (not fabricated), and the assertion is on
transformed file bytes — a no-op dispatch cannot pass it.

### The processing steps a planner maps tasks to

1. **Recipe manifest parse + validation (REQ-009).** `ParseRecipeManifest` reads `recipe.yml`
   and fail-louds on missing ops, missing/malformed-semver version, or invalid kind; requires a
   non-empty `manual:` on every `transform`/`insert` op (CLM-064) while leaving it optional on
   `create`/`merge` (CLM-065); cross-checks that every `transform` op's `rule` is a pack-relative
   path present in the declared `transform` rules (CLM-066/067); cross-checks that op `id`s are
   unique and non-empty on injection-accepting ops (CLM-068/069, the `InjectionSites` routing key);
   REFUSES a substitution placeholder in a `transform` op's `rule` — checked BEFORE the
   declared-rules cross-check so the diagnostic names the real cause, keyed on the placeholder
   delimiter the substitution implementation already declares (never a second literal), and scoped
   to `rule:` ONLY (CLM-076); REFUSES a `merge` op's non-path `fragment:` — empty/whitespace-only,
   multi-line (the inline block form, diagnosed BEFORE the placeholder case so an inline block
   reports itself as inline rather than as a templated path), or carrying the `{{` delimiter — each
   with its own message naming the op index, id and field (CLM-085/086/087), while the canon path
   form parses clean (CLM-094); validates optional compat/variants structurally. The fragment
   predicate stops where a CONTENT SNIFF would begin: a single-line, placeholder-free value that is
   syntactically a valid relative filename is accepted here even if it is really inline content,
   because distinguishing it means guessing at document syntax the manifest reader must not carry —
   that residual case fails at apply instead, naming this contract (CLM-088, step 5). The check
   covers `Ops` only; variant ops keep validating structurally.
2. **`recipes:` index on the pack manifest (REQ-008).** Add `Manifest.Recipes` and
   `validateRecipesIndex` (missing dir / missing `recipe.yml` -> error), wired into
   `ParseManifest` beside `Content.Scaffolds`.
3. **Reference resolution (REQ-010).** `ParseRecipeRef` + `ResolveRecipe`: missing pack /
   undeclared recipe / version mismatch / unpinned-or-malformed -> fail loud.
4. **Substitution (REQ-002).** `Substitute`: declared-param interpolation only; undeclared
   placeholder -> fail loud; no logic evaluation. Its SCOPE is a separate, equally load-bearing
   step: every consumer-facing site/content field an op declares — `Op.Target` (all families),
   `Op.Anchor` (insert, including the SDLC-mediated supplied half), `Op.Snippet`, the payload/
   fragment CONTENT, and the relayed `Op.Manual` — passes through substitution BEFORE it is used
   to resolve a path, match an anchor, splice bytes or populate `ApplyResult`. Every field goes
   through ONE door, so a field added later cannot be wired in without it, and the error names the
   FIELD that failed. `Op.Rule` alone is left RAW (its hole is closed at parse, step 1). The
   substitution-scope matrix is the authority for which field is which.
5. **Op dispatch (REQ-002/REQ-007).** `Apply` runs ops in declared order, dispatching per the
   closed allowlist: create/merge/transform/insert execute; `step` is recognized + sequenced
   but not executed; any other kind -> fail loud. `merge` routes by target format
   (json/yaml/toml/.env), unsupported format -> fail loud.
6. **Non-destructive + regenerate/waiver + one-shot (REQ-004/REQ-012).** User-owned files
   never clobbered. For recipe-owned output on re-apply (scaffolding/implementing): compute the
   would-be-regenerated bytes and diff against on-disk; (a) not diverged → nothing to do;
   (b) diverged AND a covering active `@waiver` is read via `opts.ReadWaivers` (production:
   `waiver.Adjudicate` over a `waiver.LineReader` on the consumer's file) → PRESERVE + record the
   covering waiver in `ApplyResult.Preserved`; (c) diverged, no covering waiver → REGENERATE
   (overwrite). The applier NEVER authors a token. Templating is one-shot / consumer-owned (step 8
   below is unaffected; REQ-012).
7. **Injection limit (REQ-011).** An unreachable `transform` (or an `insert` whose anchor is
   absent) fails loud with the op's DECLARED `manual:` text emitted VERBATIM plus a locator
   (op id + intended SUBSTITUTED target) — never silent-skip, never guess, never synthesize the
   instruction. The relayed manual is rendered through the recipe's declared params first
   (step 4); if THAT substitution fails, relay the RAW declared text and still report the
   unreachable site — the one fail-SOFT field, because this path is already a failure.
8. **Adoption record (REQ-005).** `Apply` writes the thin `{recipe ref, @version, adopted}`
   entry.
9. **Modes + multi-recipe (REQ-003/REQ-013).** Direct vs SDLC-mediated select where params/
   injection sites come from; `ApplyAll` runs multiple recipes strictly sequentially in
   declared order, same-file co-writes composing via `merge`.
10. **Operator param input at the CLI (REQ-003).** `backstop recipe apply` registers a
    repeatable `--param key=value` as a pflag **StringArray** — a StringSlice comma-splits its
    values and would silently truncate `--param greeting=a,b` into two params (CLM-093) — read
    inside `RunE` from the command's own flag set rather than from a captured variable, and
    parsed into the supplied param scope: split on the FIRST `=` only (a value may contain `=`),
    an empty value is a legal supplied empty string, a malformed argument or an empty key is a
    `*check.ConfigError` quoting the argument (CLM-090), and a repeated key is a `ConfigError`
    rather than last-wins (CLM-091). The UNDECLARED-name check runs AFTER `ResolveRecipe` — the
    recipe's declared param schema is only knowable then — and BEFORE `Apply` (CLM-092), so a
    typo'd name is refused at the invocation instead of resurfacing later as an unattributable
    unresolvable-placeholder failure. Every rejection is PRE-apply, which is why they take the
    ConfigError/exit-2 side of `runRecipeApply`'s existing exit-code split: nothing was applied,
    so the INVOCATION is what must change. `effectiveParams` is unchanged — supplied values
    already override declared defaults; the CLI simply never populated them (CLM-089).

### Fixtures must be CAPTURED, never fabricated

`create` payloads, `merge` fragments/targets, and `transform` before/after fixtures MUST be
captured from real output, not hand-written to match the applier. Sources: the
`typescript-toolchain/fixtures/captured/` convention in backstop-packs, and the bclabs-portal
go-live capture (the standing recipe fixture, BUNDLE-015 References) — e.g. a real
`vercel.json` and a real `package.json` merge fragment. A fabricated fixture that coincidentally
matches the applier's own output is a vacuous test (Sharp Edges).

## Verification

Verification is defined in frontmatter: integration level, 80% coverage threshold, targeting
`pkg/recipe`, `pkg/pack`, and `cmd/backstop`. Integration is chosen because the load-bearing
behavior is cross-component: the `transform` dispatch must route through the SAME
`engine.CheckToolAllowed` trust gate the gate's enforcement dispatch uses (REQ-006), the
`recipes:` index lives on `pkg/pack`'s manifest while resolution/apply live in `pkg/recipe`,
and the dogfoodable `backstop recipe apply` surface in `cmd/backstop` is where the transform
seam is proven end to end against a real allowlisted engine and captured fixtures — a unit-only
verification would prove the op executors in isolation while leaving exactly the wiring and the
trust-gate reuse (the places this could ship dark or bypass the allowlist) unproven. Claims are
defined in frontmatter; every requirement has at least one claim, and the op-family, merge-format,
recipe-kind, substitution, reference-resolution, and manifest-validation matrices are each
covered cell by cell.

The coverage threshold is **80** (the schema's integration-level floor), not 90: the load-bearing
risk here is cross-package wiring and real-engine/real-waiver behavior — proven by the E2E and
seam-driven tests (CLM-021/025/063) rather than by exhaustive line coverage of the pure op
executors — so the integration tier's 80 is the right bar; a 90 unit-tier threshold would push
toward mock-heavy line-chasing of exactly the stubs this spec is at pains to avoid.

## Sharp Edges

- **The op-family allowlist is CLOSED — `step` is reserved, unknown fails loud.** The applier
  must recognize exactly {create, merge, transform, insert, step}. The two failure modes to
  guard: silently skipping an unrecognized op kind (vacuous green — an authoring typo would
  drop an op with no signal), and EXECUTING a `step` op here (its executor is BUNDLE-019's — a
  step is recognized + sequenced but deferred). The seam recognizes-and-defers `step` and
  fail-louds everything outside the five. CLM-028/029/030 pin all three behaviors.

- **`transform` MUST reuse the existing engine trust gate — never a second dispatch path.** A
  bespoke transform runner would bypass `engine.CheckToolAllowed` and reopen the
  un-allowlisted-tool / baked-knowledge door. `transform` (match->fix) runs the same
  allowlisted-engine + lock-pin gate as enforcement (match->finding); the production dispatch
  in `cmd/backstop/recipe_apply.go` wires `resolveTrustedToolAllowlist` +
  `engine.CheckToolAllowed` exactly as `checkEngineToolAllowed` does. Any code that runs an
  engine command for a `transform` outside that gate is the defect. CLM-025 drives the
  un-allowlisted-reject on the real seam (not a stub-open allowlist).

- **Regenerate-vs-one-shot is KIND-gated — getting the kind wrong destroys consumer code.**
  Scaffolding/implementing regenerate-by-default; templating is one-shot / consumer-owned. If
  the applier regenerates a templating recipe on re-apply it clobbers consumer work; if it
  treats scaffolding as one-shot it lets recipe-owned output drift silently. The kind is the
  switch and is read from the recipe manifest. CLM-019/020 (regenerate) and CLM-054/055/056
  (one-shot) hold both sides.

- **"Non-destructive" protects USER-OWNED files, but recipe-owned output IS regenerable.** The
  subtle line: never-clobber applies to files the recipe does not own; recipe-owned output is
  regenerated and consumer edits to it are accountable `@waiver`s, not silently preserved.
  Conflating the two either clobbers a user's own file (CLM-018 guards) or freezes recipe
  output against re-apply (CLM-019/020 guard). Divergence is a waiver via the EXISTING
  subsystem, never a bespoke merge (CLM-021).

- **Fixtures must be CAPTURED from real output, never fabricated.** `create`/`merge`/`transform`
  fixtures come from real captures (`typescript-toolchain/fixtures/captured/`, the bclabs-portal
  go-live capture). A fixture hand-written to match the applier's own output proves nothing — it
  is a mirror, not a test. This is the single most likely way this spec's tests go vacuous.

- **The adoption record is THIN — do not smuggle the rich ledger in.** Exactly
  `{recipe ref, @version, adopted}`. Per-op/per-region provenance, forensic replay, and fleet
  dashboards are BUNDLE-017's; building any of them here couples this spec to a downstream bundle
  and violates the DD-20 split. CLM-024 guards the thinness.

- **Substitution must stay NON-Turing-complete.** `{{ param }}` is pure value interpolation. The
  temptation to add conditionals/loops/expressions is the code-in-data door reopening (an
  Nx-generator by the back door — the exact thing DD-3/DD-10 forbid). An undeclared placeholder
  must FAIL LOUD, not silently blank — silent blanking yields malformed output that looks
  applied. CLM-013/014 hold the line.

- **Substitution SEMANTICS and substitution SCOPE are different properties, and only one of
  them is provable by testing `Substitute`.** Every test that drives `Substitute` directly can
  be green while the applier routes almost nothing through it — which is exactly what happened:
  CLM-012/013/014 passed for the whole period in which `Op.Target`, `Op.Anchor` and `Op.Snippet`
  were used RAW, and a recipe templating its target wrote to a literal `{{ … }}` directory and
  exited 0 (silent wrong output, ISSUE-079). A scope claim MUST drive the real `Apply` (or the
  real parse, or the real CLI) over a COMMITTED recipe, never `Substitute` in isolation, and
  the general guard must be the filesystem-walk invariant (CLM-072) rather than a list of
  known-good paths — a list only catches the op families someone remembered to add to it.

- **The `Op.Rule` asymmetry is deliberate and must not be "cleaned up".** Uniformity is the
  seductive wrong answer here: substituting every declared path looks tidier, and it silently
  breaks two guarantees at once — the parse-time exact-match correspondence between the
  declared `transform` rules allowlist and what actually executes, and the rule that a
  CONSUMER-supplied param never selects which of the recipe's own assets an allowlisted engine
  runs in place over the consumer's tree. The compensating control is the parse-time refusal of
  a templated `rule:` (CLM-076), so the excluded field has no hole. CLM-075 exists specifically
  to fail a future uniformity refactor.

- **`Op.Payload`'s path form is STILL UNDECIDED here, and that silence stays load-bearing — while
  `Op.Fragment`'s is now DECIDED.** `fragment:` is a recipe-directory-relative PATH as of
  ISSUE-081's Decision (2026-07-27), refused at parse when it can only be something else. It was
  decided because it had a LIVE three-way contradiction: the applier read a path, this spec said
  nothing, and a committed fixture declared an inline block — a contradiction that survived the
  whole suite because the committed `starter` recipe was never driven through `Apply`. `payload:`
  has no such contradiction, so deciding it here would settle ISSUE-081's residual facet silently,
  from the wrong place, on no evidence. The scope matrix therefore still NAMES `Op.Payload` as out
  of scope rather than omitting it: the next reader must be able to tell a decision from an
  oversight. The falsifiable guard from below is CLM-076's payload-clean parse case — a future
  "uniformity" refactor that widens the placeholder check to `payload:` fails that ACCEPT case,
  which is exactly what the guard is for. The asymmetric risk to hold: the same symmetry argument
  that just landed on `fragment:` will look equally tidy applied to `payload:`, and it must not be
  taken here — it belongs to ISSUE-081.

- **A new `kind: type` contract entry for `pkg/recipe/apply.go` goes RED in Go while the artifact
  tooling stays GREEN — the spec author gets NO signal.** `pkg/recipe/contract_signature_test.go`
  compares this spec's declared `signature:` strings TEXTUALLY against `go/printer` output for the
  shipped declarations, and it walks a `contractedTypes()` list inside the test. Adding a `provides`
  entry to this spec's frontmatter for that file therefore makes `go test ./pkg/recipe/` FAIL until
  the same name is added to `contractedTypes()` — while `backstop artifact validate` passes
  throughout, because a contract entry is structurally valid on its own. The instruction currently
  lives only in that test's doc comment, discoverable only by someone already reading the test,
  which is why it is stated here where a spec author will meet it (PLAN-ISSUE-080 widened the guard
  and deferred this note to the next SPEC-054 touch — this one). The corollary for editors of THIS
  spec: contract `notes:` prose is free to change, but a `signature:` string is load-bearing text —
  field names, types and ORDER are compared byte-wise, so an "improvement" to a signature line is a
  code change in disguise.

- **The `--param` refusals are a CLI-LAYER policy, and they are deliberately stricter than the
  library.** `pkg/recipe.Apply` accepts any `Params` map: an undeclared key there is simply unused,
  and a duplicate is unrepresentable. The undeclared-name and duplicate-key refusals live at
  `cmd/backstop` ONLY, after `ResolveRecipe` and before `Apply`, so every SDLC-mediated library
  caller is unchanged. The reason to refuse rather than ignore: a dropped `--param app_nmae=x`
  resurfaces later as an unresolvable-`{{ app_name }}` failure the operator cannot attribute — the
  same undiagnosable shape ISSUE-081 was filed about — and a silently last-wins duplicate discards a
  value the operator typed. The consequence to hold: a recipe that REMOVES a param in a later
  version now breaks an invocation that still supplies it, loudly, at the invocation rather than in
  the output. That is the intended trade. The layering is the sharp part — a future refactor that
  "tidies" these checks down into `Apply` would impose a CLI ergonomics policy on every library
  caller, including SDLC-mediated apply, which supplies params programmatically.

- **`recipes:` index must not collide with `content.scaffolds` or `pack scaffold`.** "Scaffold"
  is already overloaded: `Content.Scaffolds` is a rule's paired TEST scaffold; `pack scaffold` /
  `artifact new` authors a NEW pack. Neither is "a pack ships a recipe a consumer materializes."
  The `recipes:` index is a DISTINCT top-level `pack.yml` key; declaring both `recipes:` and
  `content.scaffolds` is valid and independent. CLM-033 guards the non-collision.

- **Reference resolution is the APPLY-TIME half only.** `<pack>:<recipe>@<version>` -> pinned
  recipe is in scope; the publish-time rev-guard (bundle REQ-011) that forces a version bump on
  content change, and the drift signals (bundle REQ-012), are LATER specs. Do not build
  version-bump enforcement or drift comparison here.

- **Ordering is the consumer's declared order — the applier never reorders.** Multi-op apply
  within a recipe and multi-recipe apply are strictly sequential; determinism depends on never
  sorting or parallelizing. Same-file co-writes composing via `merge` only works because order is
  preserved. CLM-057/058/059 guard it.

- **The applier is exercised through a real CLI surface so the transform seam isn't dark.**
  `backstop recipe apply` is thin but load-bearing: without it, `pkg/recipe` is a library whose
  transform dispatch is never wired to the concrete allowlist gate, and the integration property
  (transform runs the SAME gate as enforcement) ships unproven. The CLI is the dogfood surface,
  not gold-plating. The E2E real-engine test (CLM-063) is the guard against a wired-but-no-op
  `TransformDispatch` passing all the op-level tests — the stub-green trap this project has paid
  for before.

- **The reserved `step` op payload is deliberately NOT modelled here — and the current parse does
  not round-trip it.** `Op` carries only `ID`+`Kind` for a `step` op, and the non-strict YAML
  decode drops any additional `step` payload keys silently. The CHOICE (over adding a raw-passthrough
  `map[string]any` now) is deliberate: this spec only SEQUENCES a step op (it is never executed
  here — the executor is BUNDLE-019's), so modelling a payload we cannot act on would invite a
  half-built schema, and an all-op `,inline` passthrough would swallow genuine typos on the file ops
  as "step payload." BUNDLE-019, which owns the step executor and its payload schema, will EXTEND
  the `Op` contract with the step fields (or a scoped passthrough) when it lands. The risk this
  accepts: a `recipe.yml` authored today with step payload keys will parse (they are dropped) rather
  than error — acceptable because no shipped recipe declares step payload before BUNDLE-019 defines
  it, and REQ-007's recognition/sequencing/deferral is unaffected. Documented so the dropped-keys
  behavior is a known seam, not a silent defect.

- **The transform engine is selected from the PACK, not the op — exactly one provisioned
  binding is required.** An `Op` declares its `rule`, never its ENGINE, so the only source of
  the engine is the pack's `engines:` block, and `cmd/backstop` requires the pack to declare
  EXACTLY ONE provisioned binding: `Manifest.Engines` is a map, so picking among several would
  be nondeterministic, and none means there is no tool to gate or run. Both are fail-loud
  config errors naming the pack and the count — never a silently-chosen engine. The consequence
  a pack author will hit: a pack that legitimately wants two provisioned engines cannot ship a
  `transform` recipe today. NAMED FOLLOW-UP (not built here): a per-op ENGINE SELECTOR on `Op`,
  which is a recipe.yml schema addition plus a resolution rule, not a change to this dispatch.

- **A RECIPES-ONLY pack does not validate yet — `recipes:` alone is not "content".** "Content
  is required" is asserted in THREE independent places (`pkg/pack/manifest.go`,
  `pkg/pack/validate_manifest.go`'s `ValidateManifest`, and `pkg/packval`'s phase-1 structural
  check), and NONE of them counts `recipes:`. So a pack shipping ONLY recipes fails pack
  validation, and every recipe-bearing pack must also declare `content:` and/or `engines:` —
  which is exactly how `recipes:` rides alongside existing content in CLM-033. This spec
  deliberately does not widen those checks: making `recipes:` sufficient means changing all
  three sites CONSISTENTLY (a half-change is a validator that accepts a pack one pipeline
  rejects), and no claim here asks for it. NAMED FOLLOW-UP, not smuggled in.

- **`backstop/self` covers the applier only at GLOBAL family strength (A/B1/B2), not
  spine-grade B3.** The zero-baked-language guarantee over `pkg/recipe/*.go` and
  `cmd/backstop/recipe_apply.go` currently rests on the globally-applied families; Family B3's
  spine include-list does not yet name these paths, so the applier gets the general neutrality
  sweep rather than the stricter spine treatment its position (a code path that WRITES consumer
  files from pack data) arguably warrants. CLM-027 asserts what is actually enforced today.
  NAMED FOLLOW-UP: extend Family B3's include list to these paths — a change in the
  `backstop/self` PACK repo, external by design (packs live outside core), not a core edit.

## Review Questions

1. Does the applier recognize exactly the five op families and FAIL LOUD on any other kind,
   while recognizing `step` as reserved (sequenced in declared order, NOT executed here — its
   executor is BUNDLE-019's)? (REQ-002/REQ-007 / CLM-028/029/030.)

2. Does the `transform` dispatch route through the SAME `engine.CheckToolAllowed` trust gate and
   `resolveTrustedToolAllowlist` resolver the enforcement dispatch uses (rejecting an
   un-allowlisted engine as a ConfigError), with NO second dispatch path and no stub-open
   allowlist on the tested seam? (REQ-006 / CLM-025.)

3. Is regenerate-vs-one-shot gated on the recipe KIND read from the recipe manifest, so
   templating output is never regenerated over consumer changes while scaffolding/implementing
   recipe-owned output always is? (REQ-004/REQ-012 / CLM-019/020/054/055/056.)

4. Are the `create`/`merge`/`transform` fixtures CAPTURED from real output (cite the capture
   source — `typescript-toolchain/fixtures/captured/` or the bclabs-portal go-live capture), not
   fabricated to match the applier's own output? (Fixtures-captured sharp edge.)

5. Is the adoption record strictly `{recipe ref, @version, adopted}` at the project root
   (backstop.lock-shaped), carrying the applied `@version` and NO rich per-op/per-region
   provenance? (REQ-005 / CLM-022/023/024.)

6. Does an unreachable `transform` (and an `insert` with an absent anchor) FAIL LOUD with the
   exact actionable manual instruction — never silently skipping (apply must not report success)
   and never guessing a site (no fallback write)? (REQ-011 / CLM-050/051/052/053.)

7. Is substitution non-Turing-complete (declared params only, no logic evaluation), and does an
   undeclared placeholder FAIL LOUD rather than silently blank? (REQ-002 / CLM-013/014.)

8. Is the `recipes:` index a DISTINCT top-level `pack.yml` key that does not collide with
   `content.scaffolds`, with a missing dir / missing `recipe.yml` a validation error? (REQ-008 /
   CLM-033/035.)

9. Does reference resolution cover ONLY the apply-time half (`pack:recipe@version` -> pinned
   recipe, fail-loud on missing pack / undeclared recipe / version mismatch / unpinned), leaving
   the rev-guard and drift signals to later specs? (REQ-010 / CLM-045..049.)

10. Does `backstop/self` stay GREEN over the applier + transform-dispatch code — zero
    language/platform/CI literals, all behavior from declared data? (REQ-006 / CLM-027.)

11. Is the injection-limit fail-loud message the op's DECLARED `manual:` field emitted VERBATIM
    (plus locator), with `manual:` VALIDATION-REQUIRED on every `transform`/`insert` op and core
    never synthesizing the instruction? (REQ-009/REQ-011 / CLM-050/053/064/065.)

12. Does the regenerate-vs-preserve decision run through the REAL `pkg/waiver` read path
    (`waiver.Adjudicate` over a `waiver.LineReader` on the consumer's file), preserving on a covered
    active waiver and regenerating otherwise — and does the applier NEVER author a `@waiver` token?
    (REQ-004 / CLM-021/061/062.)

13. Is there an END-TO-END test running a REAL allowlisted engine (ast-grep) over a CAPTURED
    before/after fixture through the actual `backstop recipe apply` CLI, so a wired-but-no-op
    `TransformDispatch` fails — and does every `transform` op's `rule` cross-check against the
    recipe's declared rule files? (REQ-006/REQ-009 / CLM-063/066/067.)

14. Does EVERY consumer-facing site/content field in the substitution-scope matrix pass through
    substitution before it is used to resolve a path, match an anchor, splice bytes or populate
    `ApplyResult` — target (all four executed families), anchor (including the SDLC-mediated
    SUPPLIED half, not just the declared one), snippet, payload/fragment content, and the relayed
    manual — with every scope assertion made through the real `Apply` over a COMMITTED recipe
    rather than against `Substitute`, and with the filesystem-walk invariant (no `{{`/`}}` path
    segment beneath the project root) asserted generically? (REQ-002 / CLM-070..074, CLM-077,
    CLM-078.)

15. Is `Op.Rule` left UNSUBSTITUTED — the dispatch receiving the declared, allowlist-validated
    rule path resolved under `PackDir` byte-identically — with its hole closed instead by a
    parse-time refusal scoped to `rule:` ALONE (a templated `payload:` still parsing cleanly,
    since ISSUE-081 still owns that question), and does `Op.Manual` fail SOFT to the raw
    declared text rather than replacing the operator's instruction with a second error?
    (REQ-002/REQ-009/REQ-011 / CLM-075/076/078.)

16. Is `Op.Fragment` treated as a PATH and only a path — resolved under the recipe directory and
    read VERBATIM, with the three non-path shapes (inline/multi-line, `{{`-bearing, empty)
    refused at parse under their OWN messages naming op index, id and field, the canon form
    accepted, and the residual single-line-inline value failing at APPLY with a message that
    names the path-only contract rather than a bare ENOENT — while the fragment FILE'S CONTENT
    is still substituted? Is the predicate free of any CONTENT SNIFF (no leading-`{`/`[` test),
    and scoped to the recipe's own `ops` rather than variant ops? (REQ-002/REQ-009 /
    CLM-071/085/086/087/088/094.)

17. Does `--param` reach `ApplyOptions.Params` through the SHIPPED command as a repeatable pflag
    **StringArray** (not a StringSlice, which would comma-split a value), splitting on the FIRST
    `=`, with malformed / duplicate / undeclared arguments each refused PRE-apply as a
    `*check.ConfigError` (exit 2) that quotes the offending argument and leaves nothing written
    beneath the project root — and are those refusals implemented at the CLI layer only, leaving
    `pkg/recipe.Apply` and every SDLC-mediated caller unchanged? (REQ-003 / CLM-089..093.)

## References

- **BUNDLE-015 (pack-scaffolding-recipes)** — source bundle (v0.11.0, `defined`). Seeds:
  recipe apply mechanism (REQ-001..007, 021, 023, 024) + manifest declaration (REQ-008/009) +
  reference resolution (REQ-010, apply-time half). Design decisions DD-1..DD-4 (inherited
  frame), DD-6 (three kinds), DD-8 (waiver dial), DD-9 (operation-set model), DD-10
  (transform/enforcement AST symmetry, no baked knowledge), DD-11 (two modes), DD-12
  (`pack:recipe@version`), DD-13 (injection limit), DD-18 (per-recipe directory + `recipes:`
  index), DD-20 (thin adoption record), DD-22 (sequential cross-pack ordering).
- **BUNDLE-019 (Runbooks)** — owns the `step` op EXECUTOR + probe-receipt/precondition engine.
  This spec reserves only the `step` sequencing seam (REQ-007).
- **BUNDLE-017 (recipe provenance ledger)** — owns the rich per-op/per-region ledger and
  dynamic-`transform` output scoping. This spec ships on the thin adoption record alone.
- `pkg/pack/manifest.go` — `Manifest`, `Content.Scaffolds` (the naming-collision hazard),
  `ParseManifest`; extended here with the `Recipes` index (REQ-008).
- `pkg/pack/scaffold.go` / `backstop artifact new` (`ValidPackTypes`) — the OTHER meaning of
  "scaffold" (pack authoring) the `recipes:` index must stay clear of.
- `cmd/backstop/pack_gate.go` — `checkEngineToolAllowed` (:812), `resolveTrustedToolAllowlist`
  (:46), `engine.CheckToolAllowed`: the enforcement-dispatch trust gate the `transform` dispatch
  reuses (REQ-006), and the `sandboxedRun`/allowlist seam pattern the `TransformDispatch` seam
  mirrors.
- `pkg/pack/distribution/lockfile.go` — `Lockfile`/`LockEntry`, `ReadLockfile`/`WriteLockfile`:
  the tracked-record read/write conventions the adoption record rhymes with (REQ-005).
- `backstop/self` pack — enforces the zero-baked-language boundary the applier + `transform`
  dispatch must respect (REQ-006 / DD-3/DD-10).
- Waiver subsystem (SPEC-049) — the `@waiver:<rule>:<reason>:<expiry>` machinery divergence is
  recorded through (REQ-004 / DD-8).
- **ISSUE-079 (recipe substitution scope)** — the defect that surfaced the scope gap this spec's
  substitution-scope matrix now pins: `Op.Target`/`Op.Anchor`/`Op.Snippet` were used RAW while
  `Substitute`-direct claims stayed green. Reconciled here in the same change that ships the
  behavior.
- **ISSUE-081 (recipe authoring surface: merge fragment form + no CLI param input)** — its
  **Decision (2026-07-27)** is the source for BOTH positions reconciled at 1.5.0: Gap 1 pinned
  `fragment:` as a recipe-directory-relative PATH with a non-path declaration a parse error, and
  Gap 2 gave `backstop recipe apply` a repeatable `--param key=value` threaded into
  `ApplyOptions.Params`. The reference NARROWS rather than retires: **Gap 3 remains OPEN** under
  ISSUE-081 (insert placement semantics — inline-after-anchor vs new-line-after-anchor-line — and
  whether SDLC-mediated mode's CLI reachability is stated), as does the residual **`Op.Payload`**
  facet (its path form and templatability). This spec takes no position on either, and CLM-076's
  payload-clean parse case is the guard that keeps the remaining boundary falsifiable.

## Version History

- **1.0.0 (2026-07-21, draft)** — Initial spec authored from BUNDLE-015 Seeds 1+2 (recipe apply
  mechanism + manifest declaration) plus the apply-time half of the reference-resolution seed:
  the generic ordered-op applier (REQ-001), the four op families + non-Turing `{{ }}` substitution
  (REQ-002), two application modes (REQ-003), non-destructive + regenerate/waiver (REQ-004), the
  thin adoption record (REQ-005), the zero-literal / transform-trust-gate seam (REQ-006), the
  reserved `step` op (REQ-007), the per-recipe directory + `recipes:` index (REQ-008), recipe.yml
  structural validation (REQ-009), apply-time reference resolution (REQ-010), the injection limit
  (REQ-011 / bundle REQ-021), templating one-shot (REQ-012 / bundle REQ-023), and strictly
  sequential multi-recipe apply (REQ-013 / bundle REQ-024). 13 requirements, 59 claims, 10 sharp
  edges, 10 review questions.
- **1.1.0 (2026-07-21, draft)** — Spec-reviewer FAIL fixes (3 material + 5 minor, one pass).
  M1 (coupling gap): added a declared per-op `manual:` instruction field (Op contract + recipe.yml
  REQ-009), VALIDATION-REQUIRED on `transform`/`insert`, and made REQ-011 emit it VERBATIM (never
  synthesized — synthesis would violate REQ-006); retargeted CLM-050/053 to "message == declared
  manual verbatim" and added CLM-064/065 (manual required/optional matrix). M2 (regenerate-vs-waiver
  mechanics): pinned the compute-diff-adjudicate hinge through the REAL `pkg/waiver` read path
  (`waiver.Adjudicate` + `waiver.LineReader`), added the `WaiverReader` seam + `PreservedDivergence`,
  reworked CLM-021 to exercise the real read path, and added CLM-061 (diverged+no-waiver →
  regenerate) and CLM-062 (never authors a token). M3 (anti-stub E2E): added CLM-063
  `TestRecipeApply_E2E_RealEngineTransform_CapturedFixture` — real ast-grep, captured fixture,
  through the CLI. Minors: (4) bound CAPTURED into the transform claim/test name (CLM-010);
  (5) specified the `Op.Rule` ↔ declared `TransformRules` cross-check (CLM-066/067); (6) documented
  the reserved `step` op payload non-round-trip choice (Sharp Edge); (7) pinned SDLC-mediated
  `InjectionSites` keying (op-id → WHERE for transform/insert), made CLM-017 falsifiable, and added
  CLM-060 (missing site fails loud); (8) added the coverage-threshold-80 rationale. Now 13
  requirements, 67 claims, 12 sharp edges, 13 review questions.
- **1.1.1 (2026-07-21, draft)** — Re-review residual R1 (MEDIUM): REQ-009 now enforces op-`id`
  integrity, mirroring its existing manual-present / rule-declared cross-checks — op `id`s MUST be
  unique within a recipe and non-empty on injection-accepting (`transform`/`insert`) ops, since
  `id` is the `ApplyOptions.InjectionSites` routing key (REQ-003); a duplicate id or an empty id on
  an injection-accepting op is a fail-loud validation error naming the recipe + offending id. Added
  the mirroring claim pair CLM-068 (duplicate/empty → error) and CLM-069 (unique → clean), the
  manifest-validation table row, the `ParseRecipeManifest` note, and Implementation step 1. Now 13
  requirements, 69 claims, 12 sharp edges, 13 review questions.
- **1.2.0 (2026-07-25, draft)** — Implementation-contact corrections (PLAN-SPEC-054 TASK-043),
  reconciling declared contracts to what was built and reviewer-settled through commit `3aee7db`.
  No requirement or claim was added, removed, or retargeted; the behavior contract is unchanged.
  (1) DROPPED the unimplementable `pkg/pack/manifest.go` → `pkg/recipe.ParseRecipeManifest`
  consumes edge: `pkg/recipe` imports `pkg/pack`, so parsing recipe.yml inside pack validation
  would invert that into an import cycle. The shipped `validateRecipesIndex` is STRUCTURAL only
  (directory present + contains `recipe.yml`), which is exactly what CLM-035 requires; the edge is
  retargeted to `os.Stat` and the import-cycle rule is stated in the contract note.
  (2) MOVED the `engine.CheckToolAllowed` consumes edge off `pkg/recipe/apply.go` onto
  `cmd/backstop/recipe_apply.go`, where the trust gate actually lives — no type in `pkg/recipe`
  carries a tool name or a locked version, so the gate is unimplementable there, and `pkg/recipe`'s
  only transform seam is `opts.Dispatch`. The CLI contract now also declares its
  `engine.EngineBinding` and `check.ExecCommandRunner` edges, and the Implementation section pins
  the gate-before-command-construction order plus the not-the-sandbox-runner constraint.
  (3) RECORDED the TOML codec (`github.com/pelletier/go-toml/v2`) as a DIRECT module dependency in
  a new "merge format matrix and its codecs" subsection and in `apply.go`'s consumes edges, rather
  than letting it arrive silently in `go.mod`.
  (4) ADDED per-claim `subject: cmd/backstop` to CLM-025 (BLOCKING): its test drives the root
  command from `cmd/backstop` — the only layer that can see the pack's `engines:` block — so the
  inherited `pkg/recipe` subject would trip the substantiveness noTarget join on a correctly-placed
  test. Same escape CLM-027/CLM-063 already use.
  (5) ADDED `PackDir` to the `ResolvedRecipe` contract (set by `ResolveRecipe` from its `packDir`
  argument): `Op.Rule` is a PACK-relative path per this spec's own `Op` contract, so the transform
  executor resolves the rule under the pack root — which is what lets one pack share a rule file
  across several of its recipes.
  (6) NAMED three follow-ups as sharp edges rather than building them: a per-op ENGINE SELECTOR on
  `Op` (today the pack must declare exactly one provisioned binding, fail-loud otherwise);
  RECIPES-ONLY PACKS (making `recipes:` satisfy "content is required" means changing
  `pkg/pack/manifest.go`, `pkg/pack/validate_manifest.go` and `pkg/packval` phase 1 together); and
  extending `backstop/self` Family B3's include list to `pkg/recipe/*.go` +
  `cmd/backstop/recipe_apply.go` for spine-grade neutrality (a change in the external
  `backstop/self` pack repo). Now 13 requirements, 69 claims, 15 sharp edges, 13 review questions.
- **1.2.1 (2026-07-25, implemented)** — Contract-declaration correction only: the three grouped
  const-block `provides` entries were structurally unverifiable and were the sole residual
  `contract_signature` reds. The source matches their declared values verbatim; the contracts
  pack's signature compiler simply cannot bind a member of a grouped `const (…)` block — it
  handles only a standalone `const NAME = value` — so the entries bought a red, never a
  guarantee. Filed as ISSUE-078 (const-signature support, extending ISSUE-036's kind-aware
  compiler) and dispositioned exactly as SPEC-035 v1.1.2 dispositioned `CheckTypeFindings`.
  (1) RETIRED the `RecipeKind constants` and `OpKind constants` entries from
  `pkg/recipe/manifest.go`, folding the constant names, the ISSUE-078 gap, and the naming of
  the claims that enforce each set into the `RecipeManifest` and `Op` notes — no coverage is
  lost: the kind set stays enforced by CLM-040/041/042/043 plus the ownership split
  CLM-019/020 and CLM-054/055/056, and the op-family allowlist by CLM-004/005/006/007/008/
  010/011/028/029/030, which cover every family and the outside-the-set failure.
  (2) RESHAPED `ApplyMode constants` on `pkg/recipe/apply.go` into a VERIFIED entry — the
  named type `type ApplyMode string`, which the compiler does bind — with the two values
  (ModeDirect / ModeSDLCMediated) and their enforcing claims (CLM-015/016 and CLM-017/060)
  named in the note. This is a reshape, not a retirement: net verified contract surface goes
  up by one entry. (3) Aligned the Implementation "Package layout" bullet so the body states
  the same grouped-const disposition as the frontmatter. No requirement, claim, mandated test,
  or behavior changed; still 13 requirements, 69 claims, 15 sharp edges, 13 review questions.
- **1.3.0 (2026-07-26, implemented)** — SUBSTITUTION SCOPE pinned + the result-path contract
  reconciled, both to behavior shipping in the SAME change (ISSUE-079). The spec previously pinned
  substitution SEMANTICS (declared params only, not Turing-complete, undeclared fails loud) but
  never its SCOPE — which `Op` fields the applier actually resolves — and the shipped applier ran
  only the create PAYLOAD and merge FRAGMENT content through `Substitute`. A recipe templating its
  target wrote to a literal `{{ … }}` path and exited 0. Crucially, the existing CLM-012/013/014
  tests (`TestSubstitute_ResolvesDeclaredParam`,
  `TestSubstitute_NotTuringComplete_NoLogicEvaluated`, `TestSubstitute_UndeclaredParamFailsLoud`)
  are **Substitute-DIRECT**: they exercise the function in isolation and were green throughout, so
  they cannot prove applier scope at all. The claims added here are the **Apply-level obligation**
  — every one drives the real `Apply` over a COMMITTED recipe, the real parse, or the real shipped
  CLI. (1) REQ-002 now states the scope: every consumer-facing site/content field — `Op.Target`
  (all four executed families), `Op.Anchor` (insert, INCLUDING the SDLC-mediated supplied half),
  `Op.Snippet`, payload/fragment CONTENT, and the relayed `Op.Manual` — is substituted before it
  locates, matches, splices or reports; `Op.Rule` is the one field never substituted; and no
  literal `{{`/`}}` may reach the consumer's filesystem. A new "Substitution SCOPE" matrix in the
  body carries one row per field with its failure policy, in the style of the neighbouring
  op-family / merge-format / kind / reference-resolution / manifest-validation matrices, plus the
  Rule rationale (parse-time exact-match allowlist correspondence + a consumer param must not
  select which PACK ASSET an allowlisted engine executes in place) and the Manual fail-SOFT
  rationale (the manual is emitted only on an error path, so a substitution failure there must not
  replace the operator's instruction with a second error). `Op.Payload`/`Op.Fragment` as PATHS are
  stated as explicitly OUT OF SCOPE of that matrix — their form and templatability are ISSUE-081's
  open question — so the omission reads as a decision, not a gap. (2) REQ-009 gained a FOURTH
  op-level parse cross-check: a placeholder in a `transform` op's `rule:` — and only `rule:` — is
  refused at manifest validation, before the declared-rules allowlist check, naming the op index,
  op id and field; `payload:`/`fragment:` placeholders are untouched. (3) REQ-011 now distinguishes
  VERBATIM (core never composes/paraphrases/synthesizes) from UNRENDERED: the relayed manual IS
  substituted, and fails SOFT to the raw declared text. (4) RESULT-PATH CONTRACT: `ApplyResult`
  `Written` and `PreservedDivergence.Path` are documented as carrying the recipe-declared target
  AFTER param substitution — still the recipe's own path FORM, never the applier's absolute
  resolution — and CLM-002 gained the matching substituted-form clarification while keeping its
  existing test (`TestApply_TargetComesFromRecipeNotApplier`). `pkg/recipe/apply.go`'s ApplyResult
  doc comment states the same contract and is updated in the same change; the two must not drift.
  (5) NINE claims added, each mandating a delivered test: CLM-070 (committed resolution-driven
  recipe, every templated site resolved end to end), CLM-071 (the byte-intact committed starter
  recipe — templated create target resolves before the apply halts on its inline fragment;
  ISSUE-081 marker, retires when the fragment form is pinned), CLM-072 (the walk-the-tree
  no-literal-placeholder invariant), CLM-073 (undeclared placeholder in any site field fails the
  APPLY loud, naming recipe + op index + op id, nothing written), CLM-074 (both halves of a
  supplied SDLC-mediated injection site substituted), CLM-075 (`Op.Rule` NOT substituted), CLM-076
  (templated `rule:` refused at parse, `payload:`/`fragment:` still clean), CLM-077 (shipped
  `backstop recipe apply` resolves a DEFAULTED param to the real path; `subject: cmd/backstop`),
  CLM-078 (relayed manual substituted, fail-soft to raw). (6) Three sharp edges added — semantics
  vs scope (a `Substitute`-direct test can never prove scope), the deliberate `Op.Rule` asymmetry
  that must not be "cleaned up", and the load-bearing silence on payload/fragment form — plus two
  review questions and ISSUE-079/ISSUE-081 in References. No requirement was removed and no
  existing test was dropped. Now 13 requirements, 78 claims, 18 sharp edges, 15 review questions.
- **1.4.0 (2026-07-26, implemented)** — WAIVER-ADJUDICATION SEAM WIDENED (ISSUE-080), reconciled
  to behavior shipping in the same change. `WaiverReader` returned a bare `bool`, which silently
  DROPPED `waiver.DiagnosticMalformed`: a MALFORMED @waiver token and an ABSENT one were
  indistinguishable at the decision point, so a hand-edited file marked
  `@waiver:<rule>:intentional-fork:2026-12-31` was regenerated over at exit 0, in silence. The seam
  now returns a new `DivergenceVerdict{Covered, Diagnostics}` carrying BOTH the coverage answer and
  the adjudication diagnostics. A second, OPTIONAL diagnostics channel was REJECTED: an optional
  hook is nil by default and nil means "drop it", reproducing the silent drop by shape — and both
  facts come from one `waiver.Result`, so two seams could disagree with nothing to catch it.
  `ApplyResult` gained `Regenerated` (the strict subset of `Written` overwritten with no covering
  waiver, in the same substituted value form) and `Diagnostics` (non-fatal findings the CLI warns
  on) — both REPORTING channels, never decision inputs. `ApplyOptions`' signature is unchanged: the
  field's type NAME is still `WaiverReader`. On the CLI side, `runRecipeApply` now returns
  `(recipe.ApplyResult, error)`: the success line is printed in the cobra RunE closure, which could
  not reach the result, so the CLI printed one static line instead of what the run did. It now
  reports from DATA — each written target (marking any that overwrote an unaccounted-for
  divergence), each preserved path with its covering token or marked consumer-owned, and each
  diagnostic as a stderr warning — with the ZERO result returned on every failure path and the
  ConfigError-vs-violation exit split UNCHANGED. SIX claims added, each mandating a delivered test:
  CLM-079 (uncovered divergence + unparseable reason code fails LOUD naming target/line/reason code,
  consumer bytes byte-for-byte intact — regenerate-but-warn and preserve-silently both rejected —
  including the DELIBERATE asymmetry that a malformed token naming ANOTHER rule also blocks while a
  VALID one for another rule does not), CLM-080 (covered divergence + a separate malformed token
  still preserves at exit 0 while surfacing the warning), CLM-081 (one token, one diagnostic —
  dedupe by {File, Line, Message} across N declared enforcement rules), CLM-082 (`Regenerated` is
  populated and a strict subset of `Written` in the same value form, so the CLI marks a clobber from
  data), CLM-083 (the refusal's BOUNDED blast radius — a recipe declaring no enforcement rule never
  consults the seam, so a malformed token there cannot block), CLM-084 (the type-level precondition:
  malformed-vs-absent reach the decision point as different verdicts, which a bare bool cannot
  express). CLM-082/083/084 bind tests that shipped with this change but were initially left
  unclaimed — `Regenerated` was declared as a contract with nothing mandating the only test that
  proves it is ever populated, which is the same unenforced-promise shape this change exists to
  close. A standing signature-drift test
  (`TestContract_SPEC054_DeclaredSignaturesMatchShippedTypes`) now compares these declared
  `signature:` strings TEXTUALLY against the shipped declarations, so field names, types and order
  in the contracts block are load-bearing. CLM-062 (never authors a token) is unregressed: the seam
  is still READ-only. No requirement or existing test was removed. Now 13 requirements, 84 claims,
  18 sharp edges, 15 review questions.
- **1.5.0 (2026-07-27, implemented)** — THE RECIPE AUTHORING SURFACE pinned to what shipped in the
  same change (ISSUE-081's **Decision, 2026-07-27**), on both gaps it decided.
  (1) **`fragment:` IS A PATH.** `Op.Fragment` (merge) is a recipe-DIRECTORY-relative path,
  resolved under the recipe directory and read from disk VERBATIM — never substituted. It joins
  `Op.Rule` on the NOT-substituted side of the substitution-scope matrix, which gains a row for it
  (families merge; Substituted? NO; failure policy: a non-path value is REFUSED at manifest
  validation) plus an explicit path-vs-content note, since the pre-existing "payload / fragment
  CONTENT — YES" row is CORRECT and stays: the FILE's content is still substituted; only the field
  that LOCATES it is verbatim. The decision resolved a live three-way contradiction — the applier
  read a path, this spec said nothing, and a committed fixture declared an inline block — which
  survived the whole suite because the committed `starter` recipe was never driven through `Apply`.
  (2) **REQ-009 gained a FIFTH op-level parse check**: a merge op's `fragment:` that is
  empty/whitespace-only, spans LINES (the inline block-scalar form), or carries a `{{` placeholder
  is a fail-loud validation error naming the op index, op id and FIELD, each cause under its OWN
  message; the inline case is diagnosed BEFORE the placeholder case so an inline block (which
  usually carries a placeholder too) reports itself as inline rather than as a templated path. The
  predicate deliberately stops short of a CONTENT SNIFF: a single-line, placeholder-free value that
  is syntactically a valid relative filename is ACCEPTED at parse, because separating it from a real
  filename means guessing at document syntax the manifest reader must not carry — that residual case
  fails at APPLY instead, with a message naming the path-only contract. Two layers, one canon, both
  stated. Scope is the recipe's own `ops`; variant ops keep validating structurally.
  (3) **`--param` (Gap 2).** REQ-003 now states direct mode's OPERATOR-supplied param half: the
  shipped `backstop recipe apply` exposes a repeatable `--param key=value` threaded into
  `ApplyOptions.Params`, so a `required: true` param with no `default:` is reachable instead of
  permanently unusable in direct mode. Malformed (no `=`, empty key), DUPLICATE, and UNDECLARED
  names are each refused PRE-apply as config errors quoting the offending argument — never silently
  ignored, never last-wins, and never degraded into an unattributable unresolvable-placeholder
  failure later. The flag is a pflag StringArray (a StringSlice would comma-split a value) and
  `key=value` splits on the FIRST `=`. The refusals are CLI-LAYER ONLY: `pkg/recipe.Apply` and every
  SDLC-mediated library caller are unchanged, and `effectiveParams` is untouched.
  (4) **CLM-071 REWRITTEN IN PLACE**, id reused so REQ-002 keeps its coverage. It claimed the
  starter apply HALTED on the recipe's inline fragment and carried "this claim retires when the
  fragment form is pinned"; the form is pinned, so it now claims the pinned behavior — the committed
  starter recipe resolves its templated create target AND merges its declared fragment PATH
  additively into the seeded consumer document. Its mandated test name moves from
  `TestApply_CommittedStarterRecipe_SubstitutesCreateTargetThenHaltsOnInlineFragment` to
  `TestApply_CommittedStarterRecipe_MergesTheDeclaredFragmentPath`. Retiring by REWRITE rather than
  by deletion keeps the property ISSUE-079 bought: the committed starter stays driven through
  `Apply`, which is how the contradiction hid in the first place.
  (5) **CLM-076 NARROWED** to a `payload:`-only clean-parse guard (its mandated test name
  `TestParseRecipeManifest_TemplatedTransformRuleFailsLoud` is unchanged), matching the split of the
  test's templated-payload-and-fragment manifest into a payload ACCEPT case and a fragment REJECT
  case. That accept case is what keeps the remaining boundary falsifiable from below: a future
  uniformity refactor widening the check to `payload:` fails it.
  (6) **CLAIMS ADDED**, each mandating a delivered test: CLM-085/086/087 (inline / templated /
  empty fragment refused at parse), CLM-094 (the mirroring ACCEPT — the canon path form parses clean
  and round-trips byte-exactly, without which the three refusals would be satisfied by a check that
  refused every fragment), CLM-088 (the residual apply-time failure names the path-only contract and
  echoes the declared value, keeping the underlying read error wrapped rather than bare), and
  CLM-089..093 under REQ-003 for the CLI param surface — supplies a required-no-default param,
  malformed refused, duplicate refused, undeclared refused (with its complementary all-declared
  accept case), and a value carrying `,` and `=` surviving intact. The five CLI claims carry
  per-claim `subject: cmd/backstop`, like CLM-025/027/063/077.
  (7) **REQ-002's out-of-scope parenthetical and the matrix paragraph NARROWED to `Op.Payload`
  alone**, with the evidence asymmetry stated: `fragment:` was decided because it had a live
  contradiction; `payload:` has none, so deciding it here would settle ISSUE-081's residual facet
  from the wrong place on no evidence.
  (8) **CONTRACT NOTES** on `Op` (Fragment pinned as a path read from disk, mirroring how Rule is
  already documented) and `ParseRecipeManifest` (the fragment-form rejection listed among its
  fail-loud checks). `notes:` prose only — NO `signature:` string changed, so
  `pkg/recipe/contract_signature_test.go`'s textual guard is unaffected.
  (9) **SHARP EDGES**: the payload/fragment silence bullet is replaced by a payload-only version
  carrying the fragment decision and its date; a new bullet records what PLAN-ISSUE-080 deferred to
  this touch — adding a `kind: type` entry for `pkg/recipe/apply.go` to this frontmatter makes
  `go test ./pkg/recipe/` RED until the name is also added to `contractedTypes()`, while
  `backstop artifact validate` stays GREEN, so the artifact tooling gives the spec author NO signal;
  and a third records that the `--param` refusals are a deliberately stricter CLI-layer policy that
  must not be tidied down into `Apply`. Two review questions added (the fragment path-only surface,
  the `--param` surface) and RQ-15 narrowed off `fragment:`.
  ISSUE-081 stays in References, NARROWED: Gap 3 (insert placement semantics, SDLC-mediated CLI
  reachability) and the residual `Op.Payload` facet remain OPEN there. No requirement was removed
  and no existing test was dropped.
