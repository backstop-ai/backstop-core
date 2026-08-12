---
title: "CI Recipe Pack"
number: SPEC-067
created: "2026-08-11"
updated: "2026-08-11"
status: implemented
schema_version: spec/v1
spec_version: 1.0.4

implementation:
  summary: >
    BUNDLE-015 Seed 6, the CI-recipe-pack half of it (bundle REQ-018) and NOTHING else:
    a NEW external pack, `backstop-ai/ci-workflows`, holding FOUR per-platform
    gate-workflow recipes as pack DATA — `github-actions-gate`, `gitlab-ci-gate`,
    `bitbucket-pipelines-gate`, `jenkins-gate` — each a `scaffolding`-kind recipe with a
    single `create` op at its platform's declared target, applied through the ALREADY
    SHIPPED generic executor (`backstop recipe apply <pack>:<recipe>@<version>`,
    SPEC-054, implemented). This spec adds ZERO core binary code: no new command, no new
    Go package, no new production symbol, no gate step, and no branch anywhere in
    `pkg/recipe` or `cmd/backstop` that knows the word github, gitlab, bitbucket or
    jenkins. That absence is the POINT — DD-5 names this pack "the invariant's acceptance
    test," so the four platforms differing ONLY in the reference argument passed to one
    unchanged command is the evidence the mechanism is genuinely thin. backstop-core's own
    new artifacts are therefore exactly two lines of fleet declaration
    (`backstop.yml` + `backstop.lock`) plus the mandated tests in `cmd/backstop`, which
    resolve the REAL installed pack out of `.backstop/packs/backstop-ai/ci-workflows/`,
    stage a scratch consumer project, and drive the SHIPPED root command to apply each of
    the four recipes for real — no stubbed dispatch, no committed copy of the pack under
    testdata, no fabricated fixture. Every rendered workflow is then asserted against the
    five invariants that make a CI gate non-vacuous: full history, a pinned backstop
    install, `backstop pack install` BEFORE `backstop gate`, a blocking un-swallowed gate
    invocation, and a diff base resolved from the platform's own environment variables.
    The pack additionally ships TWELVE semgrep enforcement rules (three per platform),
    each `paths:`-scoped by a BASENAME GLOB anchored on its recipe's target filename (the
    only include form that works under BOTH of the gate's real dispatch modes — measured
    against the gate's own live path, `runFindingsEngine` at
    `cmd/backstop/pack_gate.go:573`: under the DEFAULT diff-scoped gate, which hands
    semgrep EXPLICIT FILE targets, a multi-segment path include matches NOTHING, while
    under `--all`'s directory-target dispatch that same include DOES match its one file,
    so only the basename form is reliable across both), which is what makes adopting the
    pack into backstop-core verdict-neutral BY CONSTRUCTION rather than by measurement — bundle REQ-013's adoption-gating is NOT implemented yet, so path scoping
    is the only thing standing between an installed-but-unapplied recipe and someone
    else's red gate. OUT OF SCOPE and named as such: the OIDC emitter workflow and the
    cross-pack OIDC-audience parity check (the 2026-07-20 seed extension, which no bundle
    requirement formalizes and whose other side lives in the unbuilt vercel pack), the
    provider standup packs (bundle REQ-019, a DIFFERENT seed), adoption-gating and
    un-adopt (bundle REQ-013/REQ-014/REQ-022), the publish-time rev-guard (bundle
    REQ-011), compat/variants (bundle REQ-015..REQ-017, REQ-020), and any change to
    backstop-core's OWN `.github/workflows/ci.yml`, which stays bespoke and unmanaged by
    this pack.
  subject: cmd/backstop

verification:
  level: integration
  test_command: go test ./cmd/backstop/ -race -coverprofile=cover.out
  coverage_threshold: 80

requirements:
  - id: REQ-001
    text: >
      The capability must ship as a NEW pack OUTSIDE backstop-core, named
      `backstop-ai/ci-workflows` in its `pack.yml` (cloned locally as
      `backstop-ci-workflows-pack`, published as the GitHub repository
      `backstop-ai/ci-workflows`), declaring `archetype: recipes` and a top-level
      `recipes:` index whose key set is EXACTLY these four stable ids mapped to their
      per-recipe directories — `github-actions-gate` -> `recipes/github-actions-gate`,
      `gitlab-ci-gate` -> `recipes/gitlab-ci-gate`, `bitbucket-pipelines-gate` ->
      `recipes/bitbucket-pipelines-gate`, `jenkins-gate` -> `recipes/jenkins-gate`. A
      fifth indexed recipe of any kind is PROHIBITED by this spec: specifically no OIDC /
      ingest / emitter recipe and no supabase, vercel or nextjs provider recipe, all of
      which belong to later specs. The pack must declare EXACTLY ONE engine binding
      carrying `provision:`, named `semgrep-ci` (never the bare name `semgrep`, which
      would override the embedded base-engines binding every other installed pack resolves
      through) and pinned to the version `engine.TrustedToolAllowlist()` pins for semgrep.
    supports: pack-scaffolding-recipes:REQ-018@1.0.0
  - id: REQ-002
    text: >
      backstop-core must DECLARE the pack in its own committed fleet — the same version
      string in `backstop.yml` and in `backstop.lock` — so `backstop pack install`
      materializes it at `.backstop/packs/backstop-ai/ci-workflows/` and the mandated
      tests run against a REAL installed pack rather than a committed copy under testdata.
      Adopting it must NOT move backstop-core's gate verdict, and the guarantee must be
      STRUCTURAL rather than measured: every rule the pack ships declares a `paths:`
      include set, and the union of those include sets must not match any file tracked in
      backstop-core. backstop-core must NOT apply any of the four recipes to itself — its
      own `.github/workflows/ci.yml` stays bespoke, is not a recipe target, and is not
      edited by this spec.
    supports: pack-scaffolding-recipes:REQ-018@1.0.0
  - id: REQ-003
    text: >
      Each of the four recipes must declare, in its own `recipe.yml`: `kind: scaffolding`
      (the kind DD-6 assigns CI recipes, and the kind whose packval teeth REQUIRE the
      pairing below), its OWN semver `version` distinct from the pack version, a NON-EMPTY
      `enforcement.rules` list, and EXACTLY ONE `create` op whose `target` is that
      platform's declared path — `github-actions-gate` ->
      `.github/workflows/backstop-gate.yml`, `gitlab-ci-gate` -> `.gitlab-ci.yml`,
      `bitbucket-pipelines-gate` -> `bitbucket-pipelines.yml`, `jenkins-gate` ->
      `Jenkinsfile`. `merge`, `transform`, `insert` and `step` ops are PROHIBITED in all
      four recipes: a gate workflow is a whole file the recipe owns, and an op family that
      edits a consumer's existing file would put a recipe-owned promise inside
      consumer-owned bytes. Applying a recipe must write that one declared target and no
      other file.
    supports: pack-scaffolding-recipes:REQ-018@1.0.0
  - id: REQ-004
    text: >
      Every rendered gate workflow must satisfy all FIVE non-vacuity invariants, and each
      is a separate falsifiable property of the rendered bytes. (a) FULL HISTORY — the
      file declares its platform's full-clone directive (`fetch-depth: 0`, `GIT_DEPTH: 0`,
      `clone: depth: full`, and a `CloneOption` with `shallow: false` respectively), so a
      diff base is resolvable. (b) PINNED INSTALL — the file installs the backstop CLI
      from a release archive at the version the `backstop_version` param supplies, never
      from an unpinned or floating source. (c) ORDERING — `backstop pack install` appears
      BEFORE `backstop gate` in the file's execution order; a gate run against an empty
      pack directory reports capability_absent on every dimension and passes having checked
      nothing, which is the exact vacuous green ISSUE-020 was filed about. (d) BLOCKING —
      the `backstop gate` invocation is the step whose exit code is the job's verdict, and
      it is never followed by `|| true`, `|| exit 0`, `continue-on-error: true`,
      `allow_failure: true`, or the platform equivalent. (e) BASE — the file resolves a
      diff base from the platform's OWN environment variables plus the `default_branch`
      param, passes it to `backstop gate --base`, and exits non-zero when no base
      resolves; it must never silently degrade to an unscoped or empty-scope run.
    supports: pack-scaffolding-recipes:REQ-018@1.0.0
  - id: REQ-005
    text: >
      Each rendered file must be VALID INPUT for the platform it targets, proven by
      parsing rather than asserted. The three YAML targets
      (`.github/workflows/backstop-gate.yml`, `.gitlab-ci.yml`,
      `bitbucket-pipelines.yml`) must parse as YAML documents whose top-level shape
      carries that platform's required keys — `on` and `jobs`; a job or `stages` list; a
      `pipelines` block, respectively. The `Jenkinsfile` target is checked STRUCTURALLY,
      not parsed: balanced braces across the whole file plus the declarative-pipeline
      blocks `pipeline`, `agent`, `stages`, at least one `stage(`, and `steps`. The
      Jenkins check is weaker than the other three ON PURPOSE and the weakness is
      recorded, not hidden: no Groovy parser is reachable from a Go test without a JVM
      dependency this repository does not carry.
    supports: pack-scaffolding-recipes:REQ-018@1.0.0
  - id: REQ-006
    text: >
      Substitution must be COMPLETE and the payloads must not fight the substituter. No
      payload may contain a GitHub Actions `${{ ... }}` expression: the applier reads
      every `{{ ... }}` span as a param NAME and hard-errors on an undeclared one, so an
      expression-bearing payload would force a set of pass-through params whose only job
      is to re-emit themselves. The GitHub template therefore reads
      `GITHUB_BASE_REF` / `GITHUB_EVENT_NAME` / `GITHUB_SHA` as ordinary environment
      variables instead. Every `{{ ... }}` span that DOES appear in a payload must name a
      param the recipe declares, and after a successful apply no literal `{{` or `}}` may
      survive anywhere in the rendered file.
    supports: pack-scaffolding-recipes:REQ-018@1.0.0
  - id: REQ-007
    text: >
      Apply must be DETERMINISTIC and must fail loud rather than write an unresolved site.
      Applying the same recipe twice with the same params into the same project must
      produce byte-identical output (the scaffolding kind's regenerate-by-default model,
      with no drift introduced by the second run). On that second run nothing is written
      and nothing is reported as regenerated: `preserveOrRegenerate` short-circuits on
      byte-equality, returning `Final: true` without writing, so the CLI reports that
      nothing was written or preserved. `backstop_version` is declared `required: true`
      with NO default — a defaulted version silently pins whatever was current when the
      recipe was authored — so an apply that does not supply it must FAIL LOUD with a
      NON-ZERO verdict, in an error NAMING the unresolved param, BEFORE the declared
      target is written. The exact shape is INHERITED from SPEC-054 as shipped, because
      this spec adds no core code that could change it: `effectiveParams` deliberately
      leaves a required-with-no-default param ABSENT from the substitution scope, so the
      failure surfaces inside `Substitute` as an ordinary op failure that `recipe apply`
      returns as `&ExitCodeError{Code: ExitViolations}` — EXIT 1, and specifically NOT the
      exit-2 `*check.ConfigError` shape, which is reserved for malformed, duplicate or
      undeclared `--param` input. `renderPayload` runs before `writeRendered`, which is
      what makes the failure precede the write.
    supports: pack-scaffolding-recipes:REQ-018@1.0.0
  - id: REQ-008
    text: >
      backstop-core must contain ZERO baked platform knowledge for this capability, proven
      three ways. (a) DENYLIST — no non-test Go file under `pkg/recipe/` or
      `cmd/backstop/` may contain the literals `gitlab`, `bitbucket`, `jenkins`,
      `Jenkinsfile`, `.github/workflows`, `.gitlab-ci` or `bitbucket-pipelines`
      (case-insensitive), and every occurrence of the lowercase token `github`
      (matched CASE-SENSITIVELY) must fall inside a module-path reference in one of its
      two spellings — the plain `github.com/` or the regex-escaped `github\.com`, which
      denote the same thing and are both exempt. (b) ONE PATH — all four recipes must apply
      successfully through the SAME shipped `backstop recipe apply` invocation, the
      reference argument being the ONLY input that differs between them. (c) NO NEW
      SURFACE — the CLI's registered top-level command set must be unchanged by this spec,
      and no new core command, subcommand or flag may be added for CI, workflows or any
      platform.
    supports: pack-scaffolding-recipes:REQ-018@1.0.0
  - id: REQ-009
    text: >
      The pack must ship TWELVE enforcement rules, three per platform, whose ids are the
      recipe's declared `enforcement.rules` — `<platform>-gate-workflow-missing-pack-install`
      (ERROR), `<platform>-gate-workflow-verdict-swallowed` (ERROR), and
      `<platform>-gate-workflow-shallow-clone` (ERROR). Every rule's rule file must declare
      a semgrep `paths:` include set built ONLY from BASENAME GLOBS anchored on its
      recipe's target FILENAME, and nothing else, so a recipe an installed pack merely
      SHIPS polices nothing beyond files bearing that recipe's own target name. The four
      include sets are exactly: `backstop-gate*.yml` (github-actions-gate);
      `.gitlab-ci*.yml` PLUS `gitlab-ci*.yml` (gitlab-ci-gate — two patterns because the
      deployed target name is dot-prefixed while the fixture names this spec mandates are
      not, and one glob cannot cover both spellings; a dot-prefixed fixture name WOULD
      match `.gitlab-ci*.yml` and produce a finding — that was measured — so the
      two-pattern set is justified by the undotted fixture names this spec actually
      mandates, not by any inability to dot-prefix a fixture);
      `bitbucket-pipelines*.yml` (bitbucket-pipelines-gate); `Jenkinsfile*`
      (jenkins-gate). A MULTI-SEGMENT path include such as
      `.github/workflows/backstop-gate.yml` is PROHIBITED, and not as a style preference:
      measured against real semgrep 1.156.0 in the gate's own live dispatch shape
      (`runFindingsEngine`, `cmd/backstop/pack_gate.go:573`, EXPLICIT FILE targets under the
      DEFAULT diff-scoped gate), that include matches ZERO files — not even the file it
      names — so mandating it would mandate a rule that never fires under the everyday bare
      `backstop gate`. Basename globs are the only include form real semgrep honours across
      BOTH of the gate's dispatch modes, which is the whole justification for the
      form. The trailing `*` is equally mandatory, and each fixture must be NAMED so its
      own rule's include glob matches it — `backstop-gate-missing-pack-install.yml`,
      `gitlab-ci-verdict-swallowed.yml`, `Jenkinsfile-shallow-clone`, and so on. That
      naming rule is justified as FORWARD-COMPATIBLE GROUNDWORK and explicitly NOT as a
      protection that exists today. The mechanism it anticipates: packval phase 3 WOULD
      execute each fixture IN PLACE under `fixtures/rules/{valid,invalid}/`, never at the
      deployed target path, so a fixture outside its rule's include scope would be filtered
      out and read as a rule that failed to fire. That execution does not happen — for
      this pack or for any pack in the fleet: phase 3 guards fixture execution on a
      manifest field that real packs, including this one, do not declare, so the step
      skips silently everywhere. That is ISSUE-092 (Pack Test Phase3 Fixtures Cannot Fail),
      tracked separately and NOT fixed by this spec; this spec
      therefore claims no mis-scoping protection from `pack test` and names the fixtures
      under the glob because doing so costs nothing now and makes the pack correct the day
      that defect is fixed. The naming convention is in any case the one
      `backstop-self-pack`'s `rules/no-baked.yml` already establishes, pairing a
      `*neutral_spine_*.go` include with fixtures named `neutral_spine_violation.go` /
      `neutral_spine_clean.go`. Scoping
      must nevertheless stay TIGHT: no platform's include set may match any other
      platform's declared target, and the union of the four must match no file tracked in
      backstop-core. Every id named in a recipe's `enforcement.rules` must exist in the pack's ruleset
      (packval's archetype check is presence-only and never resolves those strings, so the
      cross-reference is this spec's obligation). Every rule must carry at least one
      positive and one negative fixture; each positive fixture is a byte copy of the
      recipe's own rendered payload or another real workflow, each negative is that same
      file with exactly one deliberate defect, and the whole pack must clear the real
      `backstop pack test` pipeline with zero errors — a STRUCTURAL result only (the
      manifest parses and every declared recipe, rule and engine binding is well-formed
      against the pack schema), since the pipeline's fixture-execution step is the silent
      no-op described above.
    supports: pack-scaffolding-recipes:REQ-018@1.0.0
  - id: REQ-010
    text: >
      The templates must TRAVEL: a rendered file must carry no consumer-project identity
      and no language or toolchain knowledge. Concretely, no rendered file may contain a
      language-runtime setup step or a language package-manager invocation — the denylist
      is `actions/setup-`, `go install`, `go build`, `npm `, `pnpm `, `yarn `, `pip `,
      `bundle install`, `cargo `, `mvn `, `gradle`. The ONLY organization-or-repository
      literals permitted anywhere in a rendered file are (i) the backstop release download
      coordinate itself and (ii) the single literal `actions/checkout` (with any version
      suffix), which on GitHub Actions is the only way to express REQ-004(a)'s
      `fetch-depth: 0` at all, so forbidding it would make REQ-004(a) unsatisfiable.
      Permission (ii) is CLOSED to exactly that one action: no OTHER `actions/*`
      first-party action is permitted either, and the closed form is deliberate — it is
      mechanically decidable from the rendered bytes ("is this literal `actions/checkout`
      and nothing else"), whereas an open "any legitimate platform mechanic" permission is
      not testable. Everything else stays PROHIBITED and that is the
      requirement's actual teeth: no consumer-project organization or repository name may
      appear anywhere in a rendered file, and no third-party action,
      pipe or plugin may be referenced at all — including any that would install a
      language toolchain, which permission (ii) never reaches because it names one action
      and because `actions/setup-` is already on the denylist above. The three non-GitHub
      platforms need no action reference whatsoever, so for them the permitted set is the
      release coordinate alone.
      A consumer whose packs need Layer-0 analyzer binaries
      adds those steps itself, at the documented anchor comment the template carries;
      baking any of them into a supposedly generic template would reintroduce, in pack
      data, exactly the language assumption core was cleared of.
    supports: pack-scaffolding-recipes:REQ-018@1.0.0

claims:
  # ── REQ-001 — the pack: where it lives, what it indexes, what engine it declares ──
  - id: CLM-001
    requirement: REQ-001
    text: >
      The installed pack is present and parseable at
      `.backstop/packs/backstop-ai/ci-workflows/pack.yml`, declares the manifest name
      `backstop-ai/ci-workflows` and `archetype: recipes`, and its absence FAILS the suite
      loudly rather than skipping it
    tests:
      - TestCIRecipes_PackInstalledAndParseableAtExpectedPath
  - id: CLM-002
    requirement: REQ-001
    text: >
      The pack's `recipes:` index key set is EXACTLY the four declared ids —
      github-actions-gate, gitlab-ci-gate, bitbucket-pipelines-gate, jenkins-gate — each
      mapped to a directory containing a readable `recipe.yml`; a fifth id fails the test
    tests:
      - TestCIRecipes_RecipeIndexIsExactlyTheFourPlatforms
  - id: CLM-003
    requirement: REQ-001
    kind: absence
    text: >
      DENYLIST — the pack indexes no OIDC, ingest or emitter recipe and no supabase,
      vercel or nextjs provider recipe; those surfaces belong to later specs and their
      absence here is deliberate
    tests:
      - TestCIRecipes_PackShipsNoOIDCOrProviderRecipe
  - id: CLM-004
    requirement: REQ-001
    text: >
      The pack declares exactly ONE engine binding carrying `provision:`, its key is
      `semgrep-ci` rather than the bare `semgrep` that would shadow the base-engines
      binding, and its pinned version equals the semgrep pin in
      `engine.TrustedToolAllowlist()`
    tests:
      - TestCIRecipes_SingleProvisionedEngineBindingUnderDistinctName
  # ── REQ-002 — fleet declaration and structural verdict-neutrality ──
  - id: CLM-005
    requirement: REQ-002
    text: >
      `backstop.yml` and `backstop.lock` both declare `backstop-ai/ci-workflows` at the
      SAME version string, so `backstop pack install` materializes the pack the mandated
      tests resolve
    tests:
      - TestCIRecipes_FleetDeclaresPackAtOneVersionInBothFiles
  - id: CLM-006
    requirement: REQ-002
    text: >
      Every rule file the pack ships declares a non-empty semgrep `paths:` include set,
      and the union of those include patterns matches ZERO files tracked in
      backstop-core — the verdict-neutrality guarantee proven structurally, not by
      re-running the gate and hoping
    tests:
      - TestCIRecipes_PackRuleIncludePathsMatchNoTrackedCoreFile
  - id: CLM-007
    requirement: REQ-002
    kind: absence
    text: >
      DENYLIST — backstop-core's own `.github/workflows/ci.yml` is byte-unchanged by this
      spec and is not any recipe's declared target, and the repository root carries no
      `.gitlab-ci.yml`, `bitbucket-pipelines.yml` or `Jenkinsfile`, so core adopts the
      pack without adopting any recipe
    tests:
      - TestCIRecipes_CoreAdoptsThePackWithoutApplyingAnyRecipe
  # ── REQ-003 — recipe declaration shape, per platform ──
  - id: CLM-008
    requirement: REQ-003
    text: >
      The `github-actions-gate` recipe declares kind scaffolding, its own semver version,
      a non-empty `enforcement.rules`, and exactly one op — kind `create`, target
      `.github/workflows/backstop-gate.yml` — with no merge, transform, insert or step op
    tests:
      - TestCIRecipes_GitHubActions_RecipeDeclarationShape
  - id: CLM-009
    requirement: REQ-003
    text: >
      The `gitlab-ci-gate` recipe declares kind scaffolding, its own semver version, a
      non-empty `enforcement.rules`, and exactly one op — kind `create`, target
      `.gitlab-ci.yml` — with no merge, transform, insert or step op
    tests:
      - TestCIRecipes_GitLabCI_RecipeDeclarationShape
  - id: CLM-010
    requirement: REQ-003
    text: >
      The `bitbucket-pipelines-gate` recipe declares kind scaffolding, its own semver
      version, a non-empty `enforcement.rules`, and exactly one op — kind `create`, target
      `bitbucket-pipelines.yml` — with no merge, transform, insert or step op
    tests:
      - TestCIRecipes_BitbucketPipelines_RecipeDeclarationShape
  - id: CLM-011
    requirement: REQ-003
    text: >
      The `jenkins-gate` recipe declares kind scaffolding, its own semver version, a
      non-empty `enforcement.rules`, and exactly one op — kind `create`, target
      `Jenkinsfile` — with no merge, transform, insert or step op
    tests:
      - TestCIRecipes_Jenkins_RecipeDeclarationShape
  - id: CLM-012
    requirement: REQ-003
    text: >
      Applying `github-actions-gate` through the shipped CLI into a scratch consumer
      project creates `.github/workflows/backstop-gate.yml` and leaves the rest of the
      project's file set unchanged apart from the adoption record
    tests:
      - TestCIRecipes_GitHubActions_ApplyWritesOnlyItsDeclaredTarget
  - id: CLM-013
    requirement: REQ-003
    text: >
      Applying `gitlab-ci-gate` through the shipped CLI into a scratch consumer project
      creates `.gitlab-ci.yml` and leaves the rest of the project's file set unchanged
      apart from the adoption record
    tests:
      - TestCIRecipes_GitLabCI_ApplyWritesOnlyItsDeclaredTarget
  - id: CLM-014
    requirement: REQ-003
    text: >
      Applying `bitbucket-pipelines-gate` through the shipped CLI into a scratch consumer
      project creates `bitbucket-pipelines.yml` and leaves the rest of the project's file
      set unchanged apart from the adoption record
    tests:
      - TestCIRecipes_BitbucketPipelines_ApplyWritesOnlyItsDeclaredTarget
  - id: CLM-015
    requirement: REQ-003
    text: >
      Applying `jenkins-gate` through the shipped CLI into a scratch consumer project
      creates `Jenkinsfile` and leaves the rest of the project's file set unchanged apart
      from the adoption record
    tests:
      - TestCIRecipes_Jenkins_ApplyWritesOnlyItsDeclaredTarget
  # ── REQ-004(a) — full history, per platform ──
  - id: CLM-016
    requirement: REQ-004
    text: >
      The rendered GitHub Actions workflow declares a checkout step whose `fetch-depth` is
      0, so the full history a diff base needs is present
    tests:
      - TestCIRecipes_GitHubActions_RenderedDeclaresFullHistory
  - id: CLM-017
    requirement: REQ-004
    text: >
      The rendered GitLab CI config declares the `GIT_DEPTH` variable with the value 0, so
      the full history a diff base needs is present
    tests:
      - TestCIRecipes_GitLabCI_RenderedDeclaresFullHistory
  - id: CLM-018
    requirement: REQ-004
    text: >
      The rendered Bitbucket Pipelines config declares a `clone` block whose `depth` is
      full, so the full history a diff base needs is present
    tests:
      - TestCIRecipes_BitbucketPipelines_RenderedDeclaresFullHistory
  - id: CLM-019
    requirement: REQ-004
    text: >
      The rendered Jenkinsfile declares a checkout `CloneOption` whose `shallow` setting is
      false, so the full history a diff base needs is present
    tests:
      - TestCIRecipes_Jenkins_RenderedDeclaresFullHistory
  # ── REQ-004(b) — pinned backstop install, per platform ──
  - id: CLM-020
    requirement: REQ-004
    text: >
      The rendered GitHub Actions workflow installs the backstop CLI from a release
      archive whose URL carries the exact `backstop_version` value supplied to the apply,
      and carries no unpinned or `latest` install form
    tests:
      - TestCIRecipes_GitHubActions_RenderedInstallsPinnedBackstop
  - id: CLM-021
    requirement: REQ-004
    text: >
      The rendered GitLab CI config installs the backstop CLI from a release archive whose
      URL carries the exact `backstop_version` value supplied to the apply, and carries no
      unpinned or `latest` install form
    tests:
      - TestCIRecipes_GitLabCI_RenderedInstallsPinnedBackstop
  - id: CLM-022
    requirement: REQ-004
    text: >
      The rendered Bitbucket Pipelines config installs the backstop CLI from a release
      archive whose URL carries the exact `backstop_version` value supplied to the apply,
      and carries no unpinned or `latest` install form
    tests:
      - TestCIRecipes_BitbucketPipelines_RenderedInstallsPinnedBackstop
  - id: CLM-023
    requirement: REQ-004
    text: >
      The rendered Jenkinsfile installs the backstop CLI from a release archive whose URL
      carries the exact `backstop_version` value supplied to the apply, and carries no
      unpinned or `latest` install form
    tests:
      - TestCIRecipes_Jenkins_RenderedInstallsPinnedBackstop
  # ── REQ-004(c) — pack install precedes gate, per platform ──
  - id: CLM-024
    requirement: REQ-004
    text: >
      In the rendered GitHub Actions workflow the byte offset of `backstop pack install`
      is strictly less than the byte offset of the `backstop gate` invocation, and both
      appear exactly once
    tests:
      - TestCIRecipes_GitHubActions_PackInstallPrecedesGate
  - id: CLM-025
    requirement: REQ-004
    text: >
      In the rendered GitLab CI config the byte offset of `backstop pack install` is
      strictly less than the byte offset of the `backstop gate` invocation, and both
      appear exactly once
    tests:
      - TestCIRecipes_GitLabCI_PackInstallPrecedesGate
  - id: CLM-026
    requirement: REQ-004
    text: >
      In the rendered Bitbucket Pipelines config the byte offset of `backstop pack
      install` is strictly less than the byte offset of the `backstop gate` invocation,
      and both appear exactly once
    tests:
      - TestCIRecipes_BitbucketPipelines_PackInstallPrecedesGate
  - id: CLM-027
    requirement: REQ-004
    text: >
      In the rendered Jenkinsfile the byte offset of `backstop pack install` is strictly
      less than the byte offset of the `backstop gate` invocation, and both appear exactly
      once
    tests:
      - TestCIRecipes_Jenkins_PackInstallPrecedesGate
  # ── REQ-004(d) — the gate verdict is not swallowed, per platform ──
  - id: CLM-028
    requirement: REQ-004
    kind: absence
    text: >
      DENYLIST — the rendered GitHub Actions workflow's gate step carries none of
      `|| true`, `|| exit 0`, `continue-on-error`, or a trailing `exit 0`, so the gate's
      exit code is the job's verdict
    tests:
      - TestCIRecipes_GitHubActions_GateVerdictNotSwallowed
  - id: CLM-029
    requirement: REQ-004
    kind: absence
    text: >
      DENYLIST — the rendered GitLab CI config's gate job carries none of `|| true`,
      `|| exit 0`, `allow_failure: true`, or a trailing `exit 0`, so the gate's exit code
      is the job's verdict
    tests:
      - TestCIRecipes_GitLabCI_GateVerdictNotSwallowed
  - id: CLM-030
    requirement: REQ-004
    kind: absence
    text: >
      DENYLIST — the rendered Bitbucket Pipelines config's gate step carries none of
      `|| true`, `|| exit 0`, or a trailing `exit 0`, so the gate's exit code is the
      step's verdict
    tests:
      - TestCIRecipes_BitbucketPipelines_GateVerdictNotSwallowed
  - id: CLM-031
    requirement: REQ-004
    kind: absence
    text: >
      DENYLIST — the rendered Jenkinsfile's gate step carries none of `|| true`,
      `|| exit 0`, `returnStatus: true`, or `catchError`, so the gate's exit code fails
      the build
    tests:
      - TestCIRecipes_Jenkins_GateVerdictNotSwallowed
  # ── REQ-004(e) — the diff base is resolved from platform environment, per platform ──
  - id: CLM-032
    requirement: REQ-004
    text: >
      The rendered GitHub Actions workflow assigns a base variable from platform
      environment variables and the `default_branch` param, passes that same variable to
      `backstop gate --base`, and exits non-zero when the base does not resolve
    tests:
      - TestCIRecipes_GitHubActions_ResolvesDiffBaseAndFailsWhenUnresolvable
  - id: CLM-033
    requirement: REQ-004
    text: >
      The rendered GitLab CI config assigns a base variable from platform environment
      variables and the `default_branch` param, passes that same variable to
      `backstop gate --base`, and exits non-zero when the base does not resolve
    tests:
      - TestCIRecipes_GitLabCI_ResolvesDiffBaseAndFailsWhenUnresolvable
  - id: CLM-034
    requirement: REQ-004
    text: >
      The rendered Bitbucket Pipelines config assigns a base variable from platform
      environment variables and the `default_branch` param, passes that same variable to
      `backstop gate --base`, and exits non-zero when the base does not resolve
    tests:
      - TestCIRecipes_BitbucketPipelines_ResolvesDiffBaseAndFailsWhenUnresolvable
  - id: CLM-035
    requirement: REQ-004
    text: >
      The rendered Jenkinsfile assigns a base variable from platform environment variables
      and the `default_branch` param, passes that same variable to `backstop gate --base`,
      and exits non-zero when the base does not resolve
    tests:
      - TestCIRecipes_Jenkins_ResolvesDiffBaseAndFailsWhenUnresolvable
  # ── REQ-005 — platform validity of the rendered artifact ──
  - id: CLM-036
    requirement: REQ-005
    text: >
      The rendered `.github/workflows/backstop-gate.yml` parses as a YAML document whose
      top level carries the GitHub Actions keys `on` and `jobs`, with at least one job
      declaring `runs-on` and a non-empty `steps` list
    tests:
      - TestCIRecipes_GitHubActions_RenderedParsesAsWorkflowYAML
  - id: CLM-037
    requirement: REQ-005
    text: >
      The rendered `.gitlab-ci.yml` parses as a YAML document declaring at least one job
      mapping that carries a non-empty `script` list
    tests:
      - TestCIRecipes_GitLabCI_RenderedParsesAsPipelineYAML
  - id: CLM-038
    requirement: REQ-005
    text: >
      The rendered `bitbucket-pipelines.yml` parses as a YAML document carrying a
      `pipelines` block with at least one branch or pull-request pipeline whose step
      declares a non-empty `script` list
    tests:
      - TestCIRecipes_BitbucketPipelines_RenderedParsesAsPipelinesYAML
  - id: CLM-039
    requirement: REQ-005
    text: >
      The rendered `Jenkinsfile` has balanced braces across the whole file and declares
      the declarative blocks `pipeline`, `agent`, `stages`, at least one `stage(`, and
      `steps` — a structural check, deliberately not a Groovy parse
    tests:
      - TestCIRecipes_Jenkins_RenderedIsStructurallyWellFormedDeclarativePipeline
  # ── REQ-006 — substitution completeness ──
  - id: CLM-040
    requirement: REQ-006
    kind: absence
    text: >
      DENYLIST — the rendered GitHub Actions workflow contains no literal `{{` and no
      literal `}}`, so no placeholder reached the consumer's filesystem
    tests:
      - TestCIRecipes_GitHubActions_RenderedCarriesNoResidualPlaceholder
  - id: CLM-041
    requirement: REQ-006
    kind: absence
    text: >
      DENYLIST — the rendered GitLab CI config contains no literal `{{` and no literal
      `}}`, so no placeholder reached the consumer's filesystem
    tests:
      - TestCIRecipes_GitLabCI_RenderedCarriesNoResidualPlaceholder
  - id: CLM-042
    requirement: REQ-006
    kind: absence
    text: >
      DENYLIST — the rendered Bitbucket Pipelines config contains no literal `{{` and no
      literal `}}`, so no placeholder reached the consumer's filesystem
    tests:
      - TestCIRecipes_BitbucketPipelines_RenderedCarriesNoResidualPlaceholder
  - id: CLM-043
    requirement: REQ-006
    kind: absence
    text: >
      DENYLIST — the rendered Jenkinsfile contains no literal `{{` and no literal `}}`, so
      no placeholder reached the consumer's filesystem
    tests:
      - TestCIRecipes_Jenkins_RenderedCarriesNoResidualPlaceholder
  - id: CLM-044
    requirement: REQ-006
    text: >
      No payload of any of the four recipes contains a `${{` expression, and every
      `{{ ... }}` span appearing in any payload names a param that recipe declares — the
      two halves that together make the apply resolvable without pass-through params
    tests:
      - TestCIRecipes_PayloadsCarryNoActionsExpressionAndOnlyDeclaredParams
  # ── REQ-007 — determinism and the required-param refusal ──
  - id: CLM-045
    requirement: REQ-007
    text: >
      Applying `github-actions-gate` twice with identical params leaves the target file
      byte-identical to what the first apply wrote, and the second apply reports that
      nothing was written or preserved — the byte-equality short-circuit, which is what
      catches nondeterministic rendering
    tests:
      - TestCIRecipes_GitHubActions_ReapplyIsByteIdentical
  - id: CLM-046
    requirement: REQ-007
    text: >
      Applying `gitlab-ci-gate` twice with identical params leaves the target file
      byte-identical to what the first apply wrote, and the second apply reports that
      nothing was written or preserved — the byte-equality short-circuit, which is what
      catches nondeterministic rendering
    tests:
      - TestCIRecipes_GitLabCI_ReapplyIsByteIdentical
  - id: CLM-047
    requirement: REQ-007
    text: >
      Applying `bitbucket-pipelines-gate` twice with identical params leaves the target
      file byte-identical to what the first apply wrote, and the second apply reports that
      nothing was written or preserved — the byte-equality short-circuit, which is what
      catches nondeterministic rendering
    tests:
      - TestCIRecipes_BitbucketPipelines_ReapplyIsByteIdentical
  - id: CLM-048
    requirement: REQ-007
    text: >
      Applying `jenkins-gate` twice with identical params leaves the target file
      byte-identical to what the first apply wrote, and the second apply reports that
      nothing was written or preserved — the byte-equality short-circuit, which is what
      catches nondeterministic rendering
    tests:
      - TestCIRecipes_Jenkins_ReapplyIsByteIdentical
  - id: CLM-049
    requirement: REQ-007
    text: >
      An apply that omits the required `backstop_version` param fails with EXIT CODE 1 —
      an op failure through the normal apply path, NOT the exit-2 config-error shape — in
      a message naming the unresolved `backstop_version` placeholder, and the declared
      target file does not exist afterwards, so the failure precedes any write
    tests:
      - TestCIRecipes_ApplyWithoutRequiredVersionParamFailsExitOneBeforeWriting
  # ── REQ-008 — the zero-baked-platform-knowledge proof ──
  - id: CLM-050
    requirement: REQ-008
    kind: absence
    text: >
      DENYLIST — no non-test Go file under `pkg/recipe/` or `cmd/backstop/` contains
      `gitlab`, `bitbucket`, `jenkins`, `Jenkinsfile`, `.github/workflows`, `.gitlab-ci`
      or `bitbucket-pipelines` (case-insensitive), and every occurrence of the lowercase
      token `github` (matched CASE-SENSITIVELY) falls inside a module-path reference in
      one of its two spellings — the plain `github.com/` or the regex-escaped
      `github\.com`. Both spellings denote the same module path and are equally exempt;
      the case-sensitive scope means capitalized mentions such as `GitHubActions` in
      prose, identifiers or error strings are OUTSIDE this claim and neither pass nor
      fail it
    tests:
      - TestCIRecipes_CoreProductionSourceCarriesNoPlatformLiteral
  - id: CLM-051
    requirement: REQ-008
    text: >
      All four recipes apply successfully through one identical shipped `recipe apply`
      invocation shape in which the reference argument is the only input that differs —
      the positive half of the thin-executor proof
    tests:
      - TestCIRecipes_AllFourPlatformsApplyThroughOneUnchangedInvocation
  - id: CLM-052
    requirement: REQ-008
    kind: absence
    text: >
      DENYLIST — the CLI's registered top-level command set is unchanged by this spec and
      the `recipe` namespace registers no new subcommand or flag; no CI-, workflow- or
      platform-named command exists anywhere in the tree
    tests:
      - TestCIRecipes_RegisteredCommandSurfaceUnchanged
  # ── REQ-009 — the enforcement rules and their scoping ──
  - id: CLM-053
    requirement: REQ-009
    text: >
      The `github-actions-gate` rules declare a semgrep `paths:` include set of exactly
      the basename glob `backstop-gate*.yml` and no other pattern — in particular no
      multi-segment path pattern, which semgrep matches nothing against — and that glob
      matches the recipe's deployed target `.github/workflows/backstop-gate.yml`, matches
      the rule's own in-place fixtures, and matches neither the other three platforms'
      targets nor any file tracked in backstop-core
    tests:
      - TestCIRecipes_GitHubActions_RulesScopedByTargetBasenameGlob
  - id: CLM-054
    requirement: REQ-009
    text: >
      The `gitlab-ci-gate` rules declare a semgrep `paths:` include set of exactly the two
      basename globs `.gitlab-ci*.yml` and `gitlab-ci*.yml` and no other pattern — in
      particular no multi-segment path pattern — and that set matches the recipe's
      deployed target `.gitlab-ci.yml`, matches the rule's own in-place fixtures, and
      matches neither the other three platforms' targets nor any file tracked in
      backstop-core
    tests:
      - TestCIRecipes_GitLabCI_RulesScopedByTargetBasenameGlob
  - id: CLM-055
    requirement: REQ-009
    text: >
      The `bitbucket-pipelines-gate` rules declare a semgrep `paths:` include set of
      exactly the basename glob `bitbucket-pipelines*.yml` and no other pattern — in
      particular no multi-segment path pattern — and that glob matches the recipe's
      deployed target `bitbucket-pipelines.yml`, matches the rule's own in-place fixtures,
      and matches neither the other three platforms' targets nor any file tracked in
      backstop-core
    tests:
      - TestCIRecipes_BitbucketPipelines_RulesScopedByTargetBasenameGlob
  - id: CLM-056
    requirement: REQ-009
    text: >
      The `jenkins-gate` rules declare a semgrep `paths:` include set of exactly the
      basename glob `Jenkinsfile*` and no other pattern — in particular no multi-segment
      path pattern — and that glob matches the recipe's deployed target `Jenkinsfile`,
      matches the rule's own in-place fixtures, and matches neither the other three
      platforms' targets nor any file tracked in backstop-core
    tests:
      - TestCIRecipes_Jenkins_RulesScopedByTargetBasenameGlob
  - id: CLM-057
    requirement: REQ-009
    text: >
      Every rule id named in any of the four recipes' `enforcement.rules` resolves to a
      rule declared in the pack's ruleset, and the pack declares exactly the twelve rules
      those four lists name — closing the hole packval's presence-only check leaves open
    tests:
      - TestCIRecipes_EveryDeclaredEnforcementRuleResolvesInThePackRuleset
  - id: CLM-058
    requirement: REQ-009
    text: >
      STRUCTURAL ONLY — the installed pack clears the real `backstop pack test` pipeline
      with zero errors: the manifest parses, it validates against the pack schema, every
      declared recipe, rule and engine binding is well-formed, and no phase reports an
      error. This claim asserts NOTHING about rule firing behaviour and must not be written
      or read as proof of any: the pipeline's phase-3 fixture-EXECUTION step is guarded on
      a manifest field this pack does not declare and is therefore a silent no-op for this
      pack, as it is for every pack in the current fleet. Real fixture-firing proof is
      blocked on ISSUE-092 (Pack Test Phase3 Fixtures Cannot Fail), out of scope
      here
    tests:
      - TestCIRecipes_InstalledPackClearsRealPackTestStructurally
  # ── REQ-010 — the templates travel ──
  - id: CLM-059
    requirement: REQ-010
    kind: absence
    text: >
      DENYLIST — the rendered GitHub Actions workflow contains no language-runtime setup
      step or package-manager invocation from the declared denylist, and every
      organization-or-repository literal it carries is either the backstop release
      download coordinate or the single permitted action literal `actions/checkout` (any
      version suffix), which carries REQ-004(a)'s `fetch-depth: 0`; the permission is
      CLOSED to that one action, so no OTHER `actions/*` action, no third-party action,
      and no consumer-project organization or repository name appears anywhere in the
      rendered file
    tests:
      - TestCIRecipes_GitHubActions_RenderedCarriesNoToolchainOrConsumerLiteral
  - id: CLM-060
    requirement: REQ-010
    kind: absence
    text: >
      DENYLIST — the rendered GitLab CI config contains no language-runtime setup step or
      package-manager invocation from the declared denylist, and no organization or
      repository literal other than the backstop release download coordinate — GitLab
      needs no action reference, so REQ-010's first-party-action permission has no
      instance here and the permitted set is the coordinate alone
    tests:
      - TestCIRecipes_GitLabCI_RenderedCarriesNoToolchainOrConsumerLiteral
  - id: CLM-061
    requirement: REQ-010
    kind: absence
    text: >
      DENYLIST — the rendered Bitbucket Pipelines config contains no language-runtime
      setup step or package-manager invocation from the declared denylist, and no
      organization or repository literal other than the backstop release download
      coordinate — no Bitbucket pipe is referenced, so REQ-010's first-party-action
      permission has no instance here and the permitted set is the coordinate alone
    tests:
      - TestCIRecipes_BitbucketPipelines_RenderedCarriesNoToolchainOrConsumerLiteral
  - id: CLM-062
    requirement: REQ-010
    kind: absence
    text: >
      DENYLIST — the rendered Jenkinsfile contains no language-runtime setup step or
      package-manager invocation from the declared denylist, and no organization or
      repository literal other than the backstop release download coordinate — no Jenkins
      plugin coordinate is referenced, so REQ-010's first-party-action permission has no
      instance here and the permitted set is the coordinate alone
    tests:
      - TestCIRecipes_Jenkins_RenderedCarriesNoToolchainOrConsumerLiteral

contracts:
  # No entry declares a `provides:` signature, and that is REQ-008's design rather than an
  # omission: this spec adds no Go production symbol anywhere in backstop-core, so there is
  # no signature for the presence probe to match. Each entry declares what the file
  # CONSUMES. Stated plainly because the body calls some of these seams load-bearing:
  # `consumes:` entries carry NO gate enforcement (ExtractContractEntries iterates
  # `Provides` only), so every entry below is DOCUMENTARY. What falsifies these seams is
  # the claim set — CLM-051 and CLM-058 above all.
  - file: backstop.yml
    consumes:
      - source: pkg/config
        name: Config
        kind: type
  - file: backstop.lock
    consumes:
      - source: pkg/pack/distribution
        name: LockEntry
        kind: type
  - file: .backstop/packs/backstop-ai/ci-workflows/pack.yml
    consumes:
      - source: pkg/pack
        name: Manifest
        kind: type
      # `Recipes` is a struct FIELD on `Manifest` (`pkg/pack/manifest.go:63`), not a
      # package-level variable. The schema's kind enum is
      # [function, type, interface, method, constant, variable] and carries no `field`
      # kind, so the entry is written as the qualified field name under `kind: type` —
      # the least-wrong available label — rather than mislabelled `variable`.
      - source: pkg/pack
        name: Manifest.Recipes
        kind: type
      - source: pkg/pack/engine
        name: TrustedToolAllowlist
        kind: function
  - file: .backstop/packs/backstop-ai/ci-workflows/recipes/github-actions-gate/recipe.yml
    consumes:
      - source: pkg/recipe
        name: RecipeManifest
        kind: type
      - source: pkg/recipe
        name: KindScaffolding
        kind: constant
  - file: .backstop/packs/backstop-ai/ci-workflows/recipes/gitlab-ci-gate/recipe.yml
    consumes:
      - source: pkg/recipe
        name: RecipeManifest
        kind: type
      - source: pkg/recipe
        name: KindScaffolding
        kind: constant
  - file: .backstop/packs/backstop-ai/ci-workflows/recipes/bitbucket-pipelines-gate/recipe.yml
    consumes:
      - source: pkg/recipe
        name: RecipeManifest
        kind: type
      - source: pkg/recipe
        name: KindScaffolding
        kind: constant
  - file: .backstop/packs/backstop-ai/ci-workflows/recipes/jenkins-gate/recipe.yml
    consumes:
      - source: pkg/recipe
        name: RecipeManifest
        kind: type
      - source: pkg/recipe
        name: KindScaffolding
        kind: constant
  - file: .backstop/packs/backstop-ai/ci-workflows/recipes/github-actions-gate/payload/backstop-gate.yml
    consumes:
      - source: cmd/backstop
        name: gate
        kind: function
      - source: cmd/backstop
        name: pack install
        kind: function
  - file: .backstop/packs/backstop-ai/ci-workflows/recipes/gitlab-ci-gate/payload/gitlab-ci.yml
    consumes:
      - source: cmd/backstop
        name: gate
        kind: function
      - source: cmd/backstop
        name: pack install
        kind: function
  - file: .backstop/packs/backstop-ai/ci-workflows/recipes/bitbucket-pipelines-gate/payload/bitbucket-pipelines.yml
    consumes:
      - source: cmd/backstop
        name: gate
        kind: function
      - source: cmd/backstop
        name: pack install
        kind: function
  - file: .backstop/packs/backstop-ai/ci-workflows/recipes/jenkins-gate/payload/Jenkinsfile
    consumes:
      - source: cmd/backstop
        name: gate
        kind: function
      - source: cmd/backstop
        name: pack install
        kind: function
---

# SPEC-067: CI Recipe Pack

## Overview

`backstop init` (DIR-002) is blocked on this spec. BUNDLE-003 resolved that init delegates
ALL CI-workflow scaffolding to a pack through the already-shipped recipe-apply mechanism —
init itself bakes zero language, platform or CI-provider knowledge. SPEC-054 delivered the
mechanism (`pkg/recipe` + `backstop recipe apply <pack>:<recipe>@<version>`, status
`implemented`). What does not exist is a CI recipe pack for init to apply. This spec is
that pack.

It is also, in DD-5's words, **the invariant's acceptance test**. The bundle states the
falsification condition plainly: *"If wiring GitHub-vs-GitLab CI needs a branch in core,
DD-3 is violated."* So the spec's centre of gravity is not the templates — it is the
demonstration that four materially different CI platforms flow through ONE unchanged
command, differing only in the reference argument, with backstop-core's own binary
containing no occurrence of the words gitlab, bitbucket or jenkins at all.

### What is in scope, and what the seed carries that this spec does not

The source is BUNDLE-015's sixth spec seed, *CI recipe pack (backstop-packs, first
consumer)*, whose sole formalized requirement is **REQ-018**: per-platform gate-workflow
recipes for github / gitlab / bitbucket / jenkins, as DATA of the scaffolding kind,
gating a project packs-only, shipping on the thin adoption record alone.

The seed's 2026-07-20 extension also mentions the **OIDC emitter workflow** and one side of
a **cross-pack OIDC-audience parity check**. Both are OUT OF SCOPE here, and the reason is
structural rather than a matter of appetite. The bundle itself says the authoritative
requirements are the frontmatter array, and no requirement in that array formalizes the
emitter or the parity check — REQ-018 does not mention them, and REQ-019, which does cover
provider-pack file recipes, names supabase / vercel / nextjs and is explicitly a different
seed. The parity check additionally needs its other side, which lives in the unbuilt vercel
pack. Shipping half a parity check would be a check that cannot fail. Both wait for a
follow-on spec once REQ-019's provider packs exist.

Also out of scope, each owned by a different seed of the same bundle: adoption-gated
enforcement and un-adopt (bundle REQ-013 / REQ-014 / REQ-022), the publish-time rev-guard
(REQ-011), the three drift signals (REQ-012), and compat matrix / version-keyed variants /
chain traversal (REQ-015 / REQ-016 / REQ-017 / REQ-020).

### The one consequence of adoption-gating being unbuilt

Bundle REQ-013 says a recipe's enforcement is active only for ADOPTED recipes, so a recipe
an installed pack merely ships is inert. That is not implemented. Today, installing a pack
means its rules dispatch through `pack_engines` against whatever files the gate has in
scope, adopted or not. A CI pack is the worst possible place to discover that: every
consumer of this pack has CI config files, and most of them were not written by a recipe.

So this spec does not rely on adoption-gating. Every rule the pack ships is `paths:`-scoped
in its own rule file by a basename glob anchored on its recipe's target FILENAME (REQ-009),
so a recipe an installed pack merely ships polices nothing except files bearing that
recipe's own target name — inert BY CONSTRUCTION rather than by a gate feature that does
not exist yet. This is the same static-scope answer DD-19 gives, implemented one layer
down — in the rule data instead of in the gate — and it is also what makes backstop-core's
own adoption of the pack verdict-neutral without measuring anything (REQ-002, CLM-006):
core's only workflow is `ci.yml`, which no include set names.

The glob form is not a stylistic choice and the spec is precise about it because the
obvious form does not work. Measured against real semgrep in the GATE'S OWN live dispatch
shape — `runFindingsEngine` (`cmd/backstop/pack_gate.go:573`) under the DEFAULT
diff-scoped gate, which passes EXPLICIT FILE targets — an include of
`.github/workflows/backstop-gate.yml` matches ZERO files, not even the file it names,
while the basename form `backstop-gate.yml` matches. Confirmed on both semgrep 1.156.0 and
on 1.96.0, the version `pkg/pack/engine/allowlist.go:22` actually pins and therefore what
the gate provisions. That zero-match result is SPECIFIC TO THAT DISPATCH MODE: under
`--all`, which passes the project root as a DIRECTORY target, the same multi-segment
include DOES match its one file. The asymmetry is exactly why the basename form is
mandated — it is the only include form that works under BOTH dispatch modes, so a rule
written that way cannot be silently dead under whichever mode a consumer happens to run.
(packval phase 3, were ISSUE-092 ever fixed, would use the same explicit-file invocation
shape — but phase 3 never executes today, so it is not why this matters.)

The trailing `*` and the matching fixture NAMES are justified separately and more weakly,
and the spec is explicit about the difference. Were phase 3 executing fixtures, it would
run each one IN PLACE under `fixtures/rules/…` rather than at the deployed target path, so
the include would have to match the fixture too. It is not executing them: phase 3 guards
fixture execution on a manifest field real packs do not declare, so the step skips silently
for every pack in the fleet — ISSUE-092 (Pack Test Phase3 Fixtures Cannot Fail),
separate from this spec and not fixed by it. Naming fixtures under the glob is therefore
forward-compatible groundwork, not a protection in force today, and nothing in this spec
may be read as claiming `pack test` would catch a mis-scoped include. The convention is in
any case `backstop-self-pack`'s, whose `rules/no-baked.yml` scopes a location rule with
`*neutral_spine_*.go` and names its fixtures `neutral_spine_violation.go` /
`neutral_spine_clean.go`. See Sharp Edge 8 for what the basename form costs and Sharp
Edge 10 for the Go-side testing trap it carries.

### Why the GitHub target has its own filename and the other three do not

`.github/workflows/backstop-gate.yml` is a NEW file GitHub Actions discovers alongside
whatever workflows a consumer already has. The other three platforms each have exactly one
conventional entry point — `.gitlab-ci.yml`, `bitbucket-pipelines.yml`, `Jenkinsfile` — and
there is no discovered-directory equivalent to write beside. That asymmetry is platform
DATA expressed in four recipe manifests; it is emphatically not a branch in core, and the
fact that it costs core nothing is a small piece of the acceptance-test evidence.

The consequence for an existing project is real and stated rather than hidden: on a repo
that already owns a `.gitlab-ci.yml`, `create`'s never-clobber rule preserves the
consumer's file and the apply writes nothing there. See Sharp Edge 2.

## Requirements

Requirements, claims and bundle pins are defined in frontmatter. In summary:

| Requirement | Boundary | Bundle pin |
| --- | --- | --- |
| REQ-001 — a NEW external pack `backstop-ai/ci-workflows`, `archetype: recipes`, indexing EXACTLY four recipes; exactly one provisioned engine binding named `semgrep-ci` | pack repo | `REQ-018@1.0.0` |
| REQ-002 — backstop-core's fleet declares the pack at one version in both files; verdict-neutrality is structural, not measured; core applies no recipe to itself | `backstop.yml` / `backstop.lock` | `REQ-018@1.0.0` |
| REQ-003 — each recipe is `kind: scaffolding`, own semver version, non-empty `enforcement.rules`, exactly one `create` op at its declared target; merge / transform / insert / step are prohibited | four `recipe.yml` files | `REQ-018@1.0.0` |
| REQ-004 — five non-vacuity invariants per rendered workflow: full history, pinned install, pack-install-before-gate, blocking un-swallowed gate, base resolved from platform environment | four payloads | `REQ-018@1.0.0` |
| REQ-005 — rendered validity: YAML parse plus required top-level keys for the three YAML targets; structural well-formedness for the Jenkinsfile | four payloads | `REQ-018@1.0.0` |
| REQ-006 — no `${{ }}` in any payload; every payload `{{ }}` span names a declared param; no residual `{{` / `}}` in rendered output | four payloads | `REQ-018@1.0.0` |
| REQ-007 — re-apply is byte-identical and writes nothing the second time; `backstop_version` is required with no default and its omission fails exit-1 through the normal apply path, naming the param, before any write | apply behaviour | `REQ-018@1.0.0` |
| REQ-008 — zero baked platform knowledge, proven three ways: source denylist, one-invocation-four-platforms, unchanged command surface | `pkg/recipe` + `cmd/backstop` | `REQ-018@1.0.0` |
| REQ-009 — twelve rules, three per platform, each `paths:`-scoped by a basename glob anchored on its recipe's target filename (multi-segment path includes prohibited; fixtures named to fall under the same glob, as forward-compatible groundwork rather than as a protection in force); every declared enforcement id resolves; the pack clears real `pack test` STRUCTURALLY | pack ruleset | `REQ-018@1.0.0` |
| REQ-010 — rendered files carry no language-runtime setup step, no package-manager invocation, and no consumer identity literal; the only org/repo literals permitted are the backstop release coordinate and the single action literal `actions/checkout` — no other `actions/*` action and no third-party action | four payloads | `REQ-018@1.0.0` |

Bundle REQ-019 (the provider standup packs) has no requirement, no claim and no test here;
it is a different spec seed and remains uncovered by any spec.

### The three enforcement rules, per platform

Exactly these three ids exist for each of the four platforms, so twelve in total. The table
is the same content as REQ-009 and REQ-004 state; if it ever disagrees with them, the
frontmatter wins and the table is the bug.

| Rule id suffix | Severity | Fires when | Guards |
| --- | --- | --- | --- |
| `-gate-workflow-missing-pack-install` | ERROR | the file invokes `backstop gate` with no preceding `backstop pack install` | REQ-004(c) — the vacuous green of gating an empty pack directory |
| `-gate-workflow-verdict-swallowed` | ERROR | the gate invocation is followed by a swallow form for that platform | REQ-004(d) — a pipeline claiming a guarantee it does not have |
| `-gate-workflow-shallow-clone` | ERROR | the platform's full-history directive is absent | REQ-004(a) — an unresolvable diff base |

All three are ERROR because each names a BROKEN PROMISE rather than un-adopted capability:
a workflow file that exists and says it gates, but does not. That is the severity ledger
this repository already applies — loud-not-blocking is for capability a consumer has not
adopted, and none of these three is that.

## Implementation

### 1. The pack repository (external — the primary deliverable)

Published as the GitHub repository `backstop-ai/ci-workflows`; cloned locally as
`~/src/projects/backstop-ci-workflows-pack`, matching the `backstop-<name>-pack` local
convention the other pack clones use. The manifest name — which IS the install identity —
is `backstop-ai/ci-workflows`.

**Why that name — and note that it INTRODUCES a naming category for the PUBLISHED fleet
rather than fitting one.** Be precise about the scope of that claim. No PUBLISHED
`backstop-ai/*` pack in the remote fleet is cross-cutting and non-language-scoped today:
every one is either language-prefixed (`go-toolchain`, `go-standards`, `go-contracts`,
`go-substantiveness`, `go-distribution`, `bun-toolchain`, `cobra-cli-standards`) or
project-named (`backstop-self`, `backstop-core-architecture`, `backstop-harness-*`).
`ci-workflows` is neither, so this spec establishes the cross-cutting category FOR THE
PUBLISHED FLEET, and a future maintainer should read the name as the first published
member of that category rather than as precedent it inherited.

Non-language-scoped packs do already exist in this repository, in a different category:
backstop-core's own root `packs/` directory holds `backstop/base-engines`,
`backstop/contracts` and `backstop/substantiveness`, none of them language-scoped, and
`base-engines` is embedded and live rather than hypothetical. They are local/embedded
rather than published-and-installable through `pack add` from a remote coordinate, which is
why they do not settle the naming question for a published pack — but the claim being made
here is about the published fleet specifically, not about the absence of the shape
anywhere.

The reasoning for the name itself stands on its own. The pack is cross-cutting, so it takes
no language prefix — a `go-` or `bun-` prefix would be a lie about what it contains.
`ci-workflows` says both halves of what it is: CI, and workflow files. `ci` alone is too
generic for a namespace that will later hold an ingest emitter and possibly per-platform
variants; `gate-ci` overloads the word `gate`, which already names the product's central
command and every gate step. The plural is deliberate — this pack holds four recipes on day
one.

```
backstop-ci-workflows-pack/
  pack.yml                       # name, version 0.1.0, archetype: recipes, engines:,
                                 # recipes: index (4 ids), content.ruleset (12 rules)
  recipes/
    github-actions-gate/         recipe.yml + payload/backstop-gate.yml
    gitlab-ci-gate/              recipe.yml + payload/gitlab-ci.yml
    bitbucket-pipelines-gate/    recipe.yml + payload/bitbucket-pipelines.yml
    jenkins-gate/                recipe.yml + payload/Jenkinsfile
  rules/
    github-actions/gate-workflow.yml
    gitlab-ci/gate-workflow.yml
    bitbucket-pipelines/gate-workflow.yml
    jenkins/gate-workflow.yml
  fixtures/rules/{valid,invalid}/…
```

`language:` is a required documentation-only field in `pack.yml` (phase-1 structural check;
it is never used to reject a pack). This pack declares `language: any`, because a gate
workflow is CI configuration and belongs to no programming language — which is the whole
point of it living in a pack rather than in core.

**The engine binding is load-bearing twice**, exactly as `backstop-ai/go-distribution`
records: `recipe apply` calls `provisionedEngineBinding` UNCONDITIONALLY before a recipe
runs and requires the pack to declare exactly ONE binding carrying `provision:`, so a
create-only recipes pack that declared none would fail with a config error before its first
op executed; and the twelve rules need semgrep anyway. The binding is named `semgrep-ci`
rather than `semgrep` because pack bindings OVERRIDE the embedded base-engines defaults, and
a pack that redefined `semgrep` would silently change engine resolution for every other pack
installed in the same consumer project.

### 2. The four recipes

Each `recipe.yml` declares `kind: scaffolding`, `version: 1.0.0`, a param schema, ONE
`create` op, and its `enforcement.rules`. Scaffolding is not a free choice: packval's
phase-4 recipes-archetype check REQUIRES `enforcement.rules` for a scaffolding- or
implementing-kind recipe, and that pairing is the design — the recipe writes the invariant,
the rule guards it against later drift. It also selects regenerate-by-default on re-apply,
which is what REQ-007's byte-identical property rests on.

| Recipe id | Directory | Declared target | Op |
| --- | --- | --- | --- |
| `github-actions-gate` | `recipes/github-actions-gate` | `.github/workflows/backstop-gate.yml` | one `create` |
| `gitlab-ci-gate` | `recipes/gitlab-ci-gate` | `.gitlab-ci.yml` | one `create` |
| `bitbucket-pipelines-gate` | `recipes/bitbucket-pipelines-gate` | `bitbucket-pipelines.yml` | one `create` |
| `jenkins-gate` | `recipes/jenkins-gate` | `Jenkinsfile` | one `create` |

**Params** — the same three NAMES in all four recipes, which is itself evidence the platform
differences are data. Only `runner`'s default varies, and it varies as a declared default in
four manifests, not as a branch anywhere:

| Param | Required | Default | Purpose |
| --- | --- | --- | --- |
| `backstop_version` | yes | none | the release version the CI job installs; no default, because a defaulted version silently pins whatever was current when the recipe was authored |
| `default_branch` | no | `main` | the branch a merge-base is computed against |
| `runner` | no | platform-appropriate | GitHub `ubuntu-latest`; GitLab / Bitbucket the container image; Jenkins the `agent` label |

### 3. What each rendered workflow does

Four ordered actions, the same four everywhere, spelled in each platform's own dialect:

1. **Check out with full history** — the platform's full-clone directive (REQ-004(a)). A
   shallow clone makes the base commit unreachable, and every diff-scoped run then either
   falls back or exits 2. On GitHub Actions that directive has exactly one expression —
   `fetch-depth: 0` as an input to `actions/checkout@vN` — which is why REQ-010 permits
   naming that first-party action explicitly. It is the only org/repo literal in the
   template besides the backstop release coordinate, and the three other platforms express
   full history in their own config keys and reference nothing at all.
2. **Install the pinned backstop CLI** — download the release archive for the runner's
   OS/arch from the backstop release coordinate at `{{ backstop_version }}`, unpack, put it
   on `PATH`. The archive path is the generic install route: it assumes no language
   toolchain, which `go install` would and `brew` would (REQ-010).
3. **`backstop pack install`** — MUST precede the gate (REQ-004(c)). Against an empty
   `.backstop/packs/` every dimension reports capability_absent and the job passes having
   checked nothing.
4. **Resolve a base, then `backstop gate --base "$BASE"`** — the blocking invocation
   (REQ-004(d)/(e)). The base comes from the platform's own environment variables plus the
   `default_branch` param; when it cannot be resolved the step exits non-zero rather than
   running unscoped.

A commented ANCHOR marks where a consumer adds Layer-0 toolchain setup its own packs
require. Nothing language-shaped is shipped at that anchor — see Sharp Edge 3 for what
regenerate-by-default means for edits made there.

**The GitHub template contains no `${{ ... }}` expression at all**, and that is a design
decision, not an accident of drafting. The substituter reads every `{{ ... }}` span as a
param name and hard-errors on an undeclared one, with no escape syntax; the
`backstop-ai/go-distribution` pack works around that by declaring pass-through params whose
default is the template text they re-emit. That workaround is explicitly recorded there as a
workaround, not a pattern to imitate. GitHub Actions exposes everything this template needs
as ordinary environment variables — `GITHUB_BASE_REF`, `GITHUB_EVENT_NAME`, `GITHUB_SHA` —
so the template reads those instead and the trap simply does not apply. GitLab (`$CI_*`),
Bitbucket (`$BITBUCKET_*`) and Jenkins (`${env.…}`, single-brace) never had the problem.

### 4. The twelve rules

One rule FILE per platform, three rules in each, following the idiom the go-distribution
pack established and measured: a whole-file anchor regex that doubles as the trigger
condition, paired with a whole-file `pattern-not-regex` at the same span (composing a
narrow regex under `patterns:` intersects ranges and breaks the cancellation). Two
differences from that precedent, both required here:

- **Every rule declares `paths: include:`**, built only from basename globs anchored on its
  recipe's target filename (REQ-009). go-distribution deliberately carries no `paths:`
  filter because its anchor doubles as the trigger; here the anchor cannot do that job,
  because a rule that fired on any file shaped like a gate workflow would fire on files no
  recipe wrote — including backstop-core's own `ci.yml`, whose diagnostic capture step
  legitimately ends in `|| echo …`. Path scoping is what makes REQ-002's verdict-neutrality
  structural. The exact include sets, all four measured against real semgrep in the gate's
  OWN live dispatch shape (`runFindingsEngine`, `cmd/backstop/pack_gate.go:573`, DEFAULT
  diff-scoped gate = explicit file targets):

  | Recipe | `paths: include:` | Matches the deployed target | Matches its own fixtures | Matches core |
  | --- | --- | --- | --- | --- |
  | `github-actions-gate` | `backstop-gate*.yml` | `.github/workflows/backstop-gate.yml` | `backstop-gate-*.yml` | no |
  | `gitlab-ci-gate` | `.gitlab-ci*.yml`, `gitlab-ci*.yml` | `.gitlab-ci.yml` | `gitlab-ci-*.yml` | no |
  | `bitbucket-pipelines-gate` | `bitbucket-pipelines*.yml` | `bitbucket-pipelines.yml` | `bitbucket-pipelines-*.yml` | no |
  | `jenkins-gate` | `Jenkinsfile*` | `Jenkinsfile` | `Jenkinsfile-*` | no |

  One mechanical fact drives that whole column of globs and it is not negotiable: a
  MULTI-SEGMENT include is not reliable. Under the gate's DEFAULT diff-scoped dispatch,
  which hands semgrep EXPLICIT FILE targets, `.github/workflows/backstop-gate.yml` as an
  include pattern matches ZERO files — not even the file it names — while the basename
  form matches. (Under `--all`, which passes the project root as a DIRECTORY target, that
  multi-segment include DOES match its one file; the basename form matches under BOTH.)
  Confirmed on semgrep 1.156.0 and on 1.96.0, the pinned version
  (`pkg/pack/engine/allowlist.go:22`). Basename globs are the only include form that works
  across both dispatch modes, so the form is forced.

  The trailing `*` and the fixture NAMING rule rest on a weaker, forward-looking
  justification and this spec states it as such. The mechanism they anticipate: packval
  phase 3 would run each fixture IN PLACE at `fixtures/rules/{valid,invalid}/…`, never at
  the deployed path, so an include matching only the deployed filename would filter out the
  rule's own NEGATIVE fixture and fail packval's "negative fixture not triggered" check.
  That is NOT what happens today. Phase 3 guards fixture execution on a manifest field that
  this pack — following the convention every real pack follows — does not declare, so the
  execution step skips silently for this pack and for every other pack in the fleet. It is
  ISSUE-092 (Pack Test Phase3 Fixtures Cannot Fail), out of scope here and not fixed
  by this spec. The implementer must therefore NOT treat `pack test` as a check that would
  catch a mis-scoped include; the naming rule is groundwork that costs nothing now and makes
  the pack correct once that defect is fixed. It is also the convention
  `backstop-self-pack`'s `rules/no-baked.yml` already uses: a `*neutral_spine_*.go` include
  paired with fixtures named `neutral_spine_violation.go` and `neutral_spine_clean.go`.
- **The Jenkins rules declare `languages: [generic]`**, since a Jenkinsfile is Groovy and
  the regex idiom needs no language support; the other three declare `languages: [yaml]`.

Fixtures follow the captured-fixture convention without exception: each positive fixture is
a byte copy of a real file (the recipe's own rendered payload, and a real unrelated workflow
that must stay silent), each negative is that same file with exactly ONE deliberate defect,
so a fixture isolates precisely the invariant its rule guards. Their FILE NAMES are
load-bearing rather than descriptive — each must fall under its rule's include glob
(`backstop-gate-verdict-swallowed.yml`, `Jenkinsfile-shallow-clone`, …), so that they sit in
scope for the day phase 3 actually runs them. Until then they are not executed at all, and
CLM-058 runs `pack test` for a STRUCTURAL verdict only. An implementer who wants real
firing evidence during development must run semgrep against a fixture directly, outside
`pack test`; that is a development aid, not a claim this spec makes.

### 5. backstop-core's own changes (the entire in-repo delta)

1. `backstop.yml` — one `packs:` entry, `backstop-ai/ci-workflows: <version>`.
2. `backstop.lock` — the matching entry, written by `pack add` / `pack install`, at the same
   version.
3. The mandated tests in `cmd/backstop`.

Nothing else. No Go production file is added or edited. `.github/workflows/ci.yml` is not
touched and is not a recipe target (CLM-007): it is bespoke, carries Go-specific Layer-0
tool installs no generic template can, and replacing it with recipe output is neither
required by REQ-018 nor desirable.

The fleet declaration is what makes the proof real rather than staged: the tests resolve the
pack out of `.backstop/packs/backstop-ai/ci-workflows/` — the directory `backstop pack
install` writes — instead of a committed copy under `testdata/`, which would rot the moment
the pack moved.

### 6. The test harness

Following the shape `cmd/backstop/recipe_apply_e2e_test.go` already uses, with one
substitution: the pack comes from the REAL install, not from a fixture project.

1. **Locate the installed pack.** Resolve `.backstop/packs/backstop-ai/ci-workflows` from
   the repository root, parse its `pack.yml`, and FAIL (never skip) when it is absent — a
   skip here would recreate exactly the vacuous green the pack exists to prevent.
2. **Stage a scratch consumer.** Copy the installed pack into a fresh `t.TempDir()` under
   `.backstop/packs/backstop-ai/ci-workflows`, write a minimal `backstop.yml` declaring it,
   and `t.Chdir` into that root — `recipeProjectRoot()` resolves the project from the
   discovered `backstop.yml`, so the apply writes into the scratch tree and never into the
   working repository.
3. **Drive the shipped command.** `NewRootCommand()` with
   `recipe apply backstop-ai/ci-workflows:<id>@<version> --param backstop_version=<v>`,
   reading every id, version and target from the parsed manifests rather than retyping them
   in a test literal.
4. **Assert against the rendered bytes.** YAML parses via `gopkg.in/yaml.v3` (already a
   direct dependency); the Jenkinsfile check is structural; the ordering claims compare byte
   offsets within one file; the denylist claims scan the rendered text.
5. **Source denylist (CLM-050)** walks `pkg/recipe/` and `cmd/backstop/`, skips `_test.go`
   files, and searches for the literal set — with the lowercase token `github` matched
   CASE-SENSITIVELY and only OUTSIDE a module-path reference, since every import in the
   repository contains that string and a naive scan would report the whole tree. The
   module-path exemption covers BOTH spellings of the same reference: the plain
   `github.com/` and the regex-escaped `github\.com` (the form
   `cmd/backstop/baseline.go:171` uses inside a `regexp.MustCompile` pattern). Capitalized
   mentions are outside the token the scan looks for at all.

Every mandated test lives in `cmd/backstop` (package `main`), which is the spec's
`implementation.subject`. No claim's tests straddle packages.

## Verification

`go test ./cmd/backstop/ -race -coverprofile=cover.out`, level `integration`, coverage
threshold 80 — the floor `cmd/backstop` already carries from SPEC-047, and the schema's
threshold for the integration level.

Integration is the honest level, not unit: every load-bearing claim runs the SHIPPED root
command against a REAL installed pack and asserts on files a real applier wrote. There is no
stubbed dispatch anywhere in the suite, because a stub would pass whether or not the pack
exists — which is the failure mode this spec is the acceptance test against.

This spec adds no production Go code, so it moves the `cmd/backstop` coverage number only
through the paths its tests exercise (`recipe apply`, `pack test`, the root command
assembly). The threshold is the existing floor and is not lowered.

Additionally, and outside the Go suite, the pack itself must clear `backstop pack check` and
`backstop pack test` in its own repository before publication. CLM-058 runs the same
pipeline from inside this repository against the installed copy, so the two cannot silently
diverge. Both runs are STRUCTURAL verdicts: neither executes a fixture, because phase 3's
execution step is a no-op for every pack in the fleet pending ISSUE-092 (Pack Test Phase3
Fixtures Cannot Fail). This spec's verification therefore contains NO evidence
that any of the twelve rules fires on a defective file, and that gap is deliberate and named
rather than papered over — see Sharp Edge 8 and the Review Questions.

## Sharp Edges

1. **Enforcement is NOT adoption-gated yet, and a CI pack is the worst place to learn
   that.** Bundle REQ-013 (a recipe an installed pack merely ships is inert) is unbuilt. Any
   consumer installing this pack gets its rules dispatched against in-scope files regardless
   of whether they ever applied a recipe. The `paths:` scoping in REQ-009 is the ONLY thing
   keeping that from turning the pack into an unsolicited policy engine over every CI file
   in someone else's repository — one rule file shipped without it, or widened later "to
   catch more", and it becomes one. State the residual honestly, though: scoping bounds the
   blast radius, it does not eliminate it. A consumer who installs this pack and has a
   hand-written `.gitlab-ci.yml` still gets it policed, because that file sits AT the
   recipe's target name; what scoping buys is that nothing ELSE is. Fully fixing the
   same-path case needs REQ-013, and whoever implements it should revisit this scoping
   rather than inherit it as permanent.

2. **On an existing project, three of the four recipes quietly do nothing.**
   `.gitlab-ci.yml`, `bitbucket-pipelines.yml` and `Jenkinsfile` are each their platform's
   only conventional entry point, so on a repo that already owns one, `create`'s
   never-clobber rule preserves the consumer's file and the apply reports `preserved … (the
   consumer's own file)`. The consumer ends up with an adoption record and NO gate in CI.
   That is correct non-destructive behaviour and a genuinely bad outcome, and there is no
   `merge`-into-an-existing-pipeline op in this spec to fix it. Init's greenfield case is
   unaffected. A follow-on that adds an `insert`/`merge` variant for brownfield adoption
   should be filed rather than improvised during implementation.

3. **Regenerate-by-default eats consumer edits at the toolchain anchor.** The templates
   carry a commented anchor for the Layer-0 setup steps a consumer's own packs need — and a
   Go consumer WILL need them, since `golangci-lint` and friends must be real binaries on
   `PATH`. Scaffolding kind means the next apply regenerates the file and overwrites those
   steps unless a `@waiver` covers the divergence. That is the designed dial (DD-8), but it
   is a trap for anyone who edits the file and re-applies months later, and the recipe's
   documentation must say so in the file itself, not only in the pack README.

4. **Structural is not syntactic for Jenkins.** CLM-039 checks balanced braces and the
   presence of declarative blocks. It cannot catch an unbalanced quote, an invalid step
   argument, or anything a Groovy parser would. Nothing in this repository can, short of a
   JVM dependency. Do not let the passing test be read as "the Jenkinsfile is valid" — it is
   "the Jenkinsfile is not obviously malformed". If Jenkins support ever matters
   commercially, the honest upgrade is a containerized `declarative-linter` run, not a
   better regex.

5. **The `github` denylist has one legitimate exception and it is the whole difficulty.**
   Every import path in the repository contains `github.com`. CLM-050 must therefore exempt
   that reference specifically, and an implementer who takes the shortcut of dropping
   `github` from the denylist entirely converts the strongest claim in the spec into a claim
   that proves nothing about the platform this pack's first recipe targets. The exemption is
   the module-path reference, not the substring `github`.

   The exemption has TWO spellings and a scan that knows only one is wrong in a way that
   looks right. `cmd/backstop/baseline.go:171` writes the same module path REGEX-ESCAPED —
   ``regexp.MustCompile(`github\.com[:/]...`)`` — so the characters following `github` are
   `\.com`, not `.com/`. A literal-prefix scan flags this genuine module-path reference and
   the claim fails on pre-existing, entirely innocent code. Both spellings are exempt.
   Conversely, the token is matched CASE-SENSITIVELY, which is a deliberate narrowing and
   not an oversight: `baseline.go` also carries five capitalized mentions (a "GitHub Actions"
   comment, `ensureGitHubAuth` twice, and two error strings naming GitHub). Those are the
   baseline-pull feature's own vocabulary, they are not what this claim was ever measuring,
   and CLM-050 neither passes nor fails them. An implementer who "helpfully" makes the scan
   case-insensitive turns a green claim red against code this spec does not govern; one who
   widens the exemption to CI-platform-shaped literals generally guts the claim instead.

6. **Two specs are editing `backstop.yml` and `backstop.lock` in the same window.**
   SPEC-066 adds `backstop-ai/go-distribution` to the same two files. The edits are
   independent map entries and compose, but a plan that regenerates either file wholesale
   rather than adding one entry will silently drop the other spec's entry. Whichever lands
   second must re-read both files rather than write from a remembered baseline.

7. **`archetype: recipes` still requires non-empty `content`.** Phase-1's structural check
   demands a ruleset, a scaffold, or an SDK regardless of archetype — a recipes-ONLY pack
   fails validation. This pack satisfies it with its twelve rules, so the constraint is
   invisible here; a future maintainer stripping the rules "because the recipes are the
   product" would break `pack check` for a reason phase 4's diagnostics never mention.

8. **The include glob is a BASENAME match, so it is deliberately wider than the target
   path — accepted because it is the only form that works, and with NO safety net under
   it.** `backstop-gate*.yml` matches that basename ANYWHERE in a consumer's tree: a
   consumer with an unrelated `deploy/backstop-gate.yml.bak`-shaped file, or a vendored copy
   under `third_party/`, gets it policed. That extra width is a real cost, wider than an
   exact path include would be if exact paths worked reliably. They do not: under the
   gate's DEFAULT diff-scoped dispatch (`runFindingsEngine`,
   `cmd/backstop/pack_gate.go:573`, EXPLICIT FILE targets) semgrep matches ZERO files
   against a multi-segment include — measured on 1.156.0 and on the pinned 1.96.0
   (`pkg/pack/engine/allowlist.go:22`) — so the natural tightening to the real path
   silently stops the rule firing at all, which reads as a clean pack rather than a dead
   one. Under `--all`'s DIRECTORY-target dispatch the same multi-segment include WOULD
   match its one file, which makes the trap worse rather than better: a tightened rule
   would look alive under a full sweep and be dead under the everyday bare `backstop
   gate`. The basename form is chosen because it is the only form that works under BOTH
   dispatch modes, not because it is tight.

   What makes this genuinely sharp is that nothing catches the mistake for you. A
   mis-scoped include is NOT caught by `backstop pack test` today: phase 3's fixture
   execution is guarded on a manifest field real packs do not declare, so it skips silently
   for every pack in the fleet — ISSUE-092 (Pack Test Phase3 Fixtures Cannot Fail),
   tracked separately. A green `pack test` is a structural verdict and says nothing about
   whether any rule fires. Anyone who edits these `paths:` blocks must verify firing by
   running semgrep against the fixture directly, and must not read the green pipeline as
   confirmation.

   One dependency underneath all of this is worth recording explicitly, because it is
   unstated elsewhere and currently TRUE rather than guaranteed. REQ-009 deliberately NAMES
   each fixture so its own rule's include glob MATCHES it
   (`backstop-gate-missing-pack-install.yml`, `Jenkinsfile-shallow-clone`, …) — and once a
   consumer installs this pack, those glob-matching fixtures sit inside the consumer's own
   tree at `.backstop/packs/backstop-ai/ci-workflows/fixtures/rules/invalid/`.
   REQ-002/CLM-006 measures verdict-neutrality only over files TRACKED in backstop-core,
   and installed packs are gitignored, so CLM-006 never looks at those fixtures at all.
   They stay inert for a different reason, measured: semgrep SKIPS git-ignored paths, and
   `.backstop/packs/` is gitignored (`.gitignore:41`), so even the DIRECTORY-target
   dispatch of `backstop gate --all` never scans them. Verdict-neutrality therefore holds
   today — but it holds on semgrep's git-ignore awareness PLUS the consumer keeping
   `.backstop/packs/` gitignored, which is the standard state this repo's own conventions
   assume and which nothing in this spec requires. A consumer who un-ignores their pack
   directory would begin scanning this pack's own deliberate NEGATIVE fixtures and would
   collect findings on them.

9. **Rule severity is declared in the rule FILE, not the manifest.** semgrep's per-rule
   `severity:` becomes the SARIF level, and `warning` is non-blocking by contract while
   `error` and an ABSENT level both block. All twelve rules are ERROR. A rule authored with
   no `severity:` at all still blocks — fail-closed, and therefore silently
   indistinguishable from a deliberate ERROR when reading the manifest alone.

10. **A Go test asserting the glob semantics must compare against the BASENAME, or it
    asserts something semgrep does not do.** CLM-053..056 assert semgrep's matching
    behaviour for the four basename patterns, and the obvious way to test that in Go is
    `filepath.Match`. It does not reproduce semgrep's semantics for a slashless pattern:
    semgrep matches a slashless include against the file's basename anywhere in the tree,
    whereas `filepath.Match` requires the WHOLE path to match. Measured —
    `filepath.Match("backstop-gate*.yml", ".github/workflows/backstop-gate.yml")` returns
    `false, nil`, while the same pattern against `filepath.Base(path)` returns `true`. Any
    Go-side assertion of these claims must therefore match against `filepath.Base(path)`,
    never the full path; written the naive way, the test for "the glob matches the deployed
    target" fails against a correct pack, and an implementer's likely repair — widening the
    mandated pattern until `filepath.Match` accepts the full path — would replace a working
    include with one semgrep matches nothing against (Sharp Edge 8). The Go helper is a
    model of semgrep's behaviour, not the behaviour itself; the only authority is semgrep.

## Review Questions

- Does any rule file ship without a `paths: include:` block, or with a pattern outside the
  four include sets REQ-009 fixes? A widened include is Sharp Edge 1 detonating; a
  multi-segment path include is Sharp Edge 8 — it matches nothing and the rule is dead while
  looking healthy.
- Was a green `backstop pack test` written up, in a test name, a comment or a commit
  message, as evidence that any rule FIRES? It is not, and CLM-058 says so: phase 3's
  fixture-execution step is a silent no-op for this pack and for every pack in the fleet,
  pending ISSUE-092 (Pack Test Phase3 Fixtures Cannot Fail), open since 2026-07-27 and
  risk-classed critical. REAL fixture-firing proof is BLOCKED
  on that defect — the honest position for this spec is that the twelve rules are proven
  well-formed and unproven to fire.
- OPEN QUESTION, deliberately unresolved here and tied to that same defect (ISSUE-092): does
  this spec's fixture POLARITY match real-world pack convention? This spec and packval's
  phase-3 code agree — a POSITIVE fixture is the clean file the rule must stay silent on,
  a NEGATIVE fixture is the defective file the rule must fire on (`phase3.go` errors with
  "positive fixture failed" when the run does not pass, and "negative fixture not
  triggered" when it does). But the in-repo `packs/contracts/pack.yml` declares the
  INVERSE, pairing `positive: sig-mismatch.go` (the defect) with `negative:
  sig-present.go` (the clean file), and nothing has ever caught the disagreement because
  the execution that would surface it never runs. Whoever resolves ISSUE-092 must
  read both artifacts together and decide which polarity is canonical before these twelve
  rules' fixtures are trusted; do not silently flip them on the strength of this spec
  alone.
- Was `backstop_version`-omission asserted at exit code 1 (an op failure naming the
  unresolved placeholder), or did the test assert an exit-2 config error? The shipped
  `effectiveParams` leaves the param absent and lets `Substitute` fail, so exit 2 is the
  wrong expectation and a test asserting it would be made to pass by changing core — which
  this spec forbids.
- Does the rendered GitHub workflow reference any action other than `actions/checkout`?
  REQ-010 permits that one first-party action for `fetch-depth: 0` and nothing further; a
  second action is scope creep into toolchain territory.
- Do the CLM-053..056 glob tests compare the pattern against `filepath.Base(path)`, or
  against the full path? `filepath.Match` does not model semgrep's slashless-pattern
  semantics (Sharp Edge 10), so a full-path comparison asserts something semgrep never
  does — and the mandated include set, not the test, is what tends to get "fixed".
- Is `CLM-050`'s `github` exemption written as the prefix `github.com/`, or was the token
  dropped from the denylist to make the test pass?
- Does any payload contain a `${{` sequence? One re-introduces the pass-through-param
  workaround REQ-006 exists to avoid, and the failure mode is an apply-time hard error, not
  a rendering defect.
- Is `backstop pack install` genuinely BEFORE `backstop gate` in each rendered file, or only
  before it in the source YAML while the platform's execution order differs (Bitbucket's
  `script` list, Jenkins' stage order)? The claim compares byte offsets; confirm byte order
  and execution order agree for each platform.
- Does the engine binding use the key `semgrep-ci`? A binding keyed `semgrep` overrides the
  base-engines default for every other pack in any consumer project that installs this one —
  a coupling defect that shows up in someone else's repository, not in this pack's tests.
- Is `backstop_version` still `required: true` with no default? A default added later "for
  convenience" pins a stale release into every future consumer's CI and CLM-049 would need
  to be deleted to allow it.
- Did the implementer apply any of the four recipes to backstop-core itself? CLM-007 forbids
  it; `.github/workflows/ci.yml` must be byte-unchanged.
- Do all four negative fixtures differ from their positive counterpart by exactly one
  defect, and are the positives byte copies of real files rather than authored samples?
- No `follows:` binding appears on any requirement. The in-repo `standards/` tree carries no
  rule files (only `core/` and `typescript/` skeletons), and no CI-configuration or shell
  standard exists to bind to; per the escalation-over-guessing rule this is escalated rather
  than filled with an invented mapping. If a standards pack covering CI configuration is
  adopted before implementation, these requirements should be re-bound to it.

## References

- **BUNDLE-015 (Pack Scaffolding Recipes)**, v0.11.0 `defined` — the source. Seed 6, *CI
  recipe pack (backstop-packs, first consumer)*; requirement **REQ-018**; design decisions
  **DD-5** (the CI pack is the first and canonical consumer, and the forcing function),
  **DD-3** (all language/platform knowledge lives in data), **DD-6** (CI recipes are the
  scaffolding kind), **DD-7/DD-8** (paired enforcement, the waiver as the dial), **DD-18**
  (per-recipe directory + `recipes:` index), **DD-19** (static scope from declared target
  paths), **DD-20** (the thin adoption record — this pack ships on it alone), **DD-22**
  (provider decomposition; the CI pack is one of the four). REQ-019, the provider standup
  packs, is a DIFFERENT seed and is not covered here.
- **SPEC-054 (Recipe Apply And Manifest)**, `implemented` — the mechanism this spec consumes
  and does not re-specify: `pkg/recipe`, `backstop recipe apply <pack>:<recipe>@<version>`,
  the per-recipe directory shape, the `{{ }}` substituter, the adoption record, and the
  regenerate-by-default / never-clobber model.
- **BUNDLE-003 (Onboarding Experience) / DIR-002 (`backstop init`)** — the blocked consumer.
  Init delegates all CI scaffolding to this pack through `recipe apply`.
- **`backstop-ai/go-distribution`** (`~/src/projects/backstop-go-distribution-pack`) — the
  only other real recipe-declaring pack, and the working precedent for the `recipes:` index,
  the single provisioned engine binding under a non-shadowing name, the scaffolding-kind /
  `enforcement.rules` pairing, the whole-file anchor rule idiom, and the captured-fixture
  convention. Its `${{ }}` pass-through params are the workaround REQ-006 avoids rather than
  imitates.
- **SPEC-066 (CI Release Auto Tag)** — the precedent for a spec whose deliverable lives in an
  external pack while `cmd/backstop` is the subject, proven by in-repo tests against the real
  installed pack. Also the other spec currently editing `backstop.yml` / `backstop.lock`.
- **SPEC-047 (Bun Toolchain Pack And Two-Surface Proof)** — the earlier external-pack
  precedent and the source of `cmd/backstop`'s 80 coverage floor.
- **ISSUE-020** — why `backstop pack install` must precede `backstop gate`: a gate over an
  empty pack directory reports capability_absent everywhere and passes having checked
  nothing.
- **ISSUE-092 (Pack Test Phase3 Fixtures Cannot Fail)**, `open`, risk `critical`, filed
  2026-07-27 — the core defect this spec repeatedly defers to: `backstop pack test`'s
  phase-3 fixture-EXECUTION step is guarded on a manifest field real packs do not declare,
  so it skips silently for every pack in the fleet. It is why CLM-058 is a STRUCTURAL claim
  only, why REQ-009's matching-fixture-name rule is forward-compatible groundwork rather
  than present-day protection, and why this spec carries no evidence that any of its twelve
  rules fires. Also the owner of the unresolved fixture-POLARITY question between packval's
  phase-3 semantics and `packs/contracts/pack.yml`. Not fixed by this spec.
- **`.github/workflows/ci.yml`** (backstop-core) — the real, working reference for what a
  CI job running `backstop gate` looks like. NOT a template: its Layer-0 tool installs and
  pinned analyzer versions are specific to backstop-core gating itself in Go.

## Version History

- **1.0.4** (2026-08-11) — **CLOSE-OUT: status `draft` -> `implemented`.** No requirement,
  claim, contract, test or mechanism is added, removed or reworded; this entry records the
  evidence the flip rests on. **Verification.** The impl-reviewer independently re-ran all
  62 mandated tests (62 pass, 0 fail) and the full gate (green, exit 0), and then MUTATED
  the delivered code — 13 separate deliberate mutations applied to the REAL installed pack,
  each confirmed to turn its affected claim red with the expected message. Every claim in
  this spec is therefore falsification-verified rather than read-verified, which is the bar
  that matters for a spec whose whole subject is the difference between a check that fires
  and a check that merely exists. **Publication.** The deliverable ships as
  `backstop-ai/ci-workflows@v0.1.0`; the reviewer cloned that published tag fresh from
  GitHub and confirmed it byte-identical to the copy installed under
  `.backstop/packs/backstop-ai/ci-workflows/` that the mandated tests exercise, so the
  tested artifact and the published artifact are the same bytes. **Core delta.** ZERO
  backstop-core production code changed — the in-repo delta is the two fleet-declaration
  lines (`backstop.yml`, `backstop.lock`) plus the mandated tests in `cmd/backstop`, exactly
  as Implementation §5 scopes it. The `contracts` block's absence of any `provides:`
  signature was confirmed ACCURATE rather than assumed: there is no new Go production symbol
  for a presence probe to match, which is REQ-008's design landing rather than an omission.
  **One non-blocking finding, routed OUT of this spec.** Two of the four platform recipes —
  `gitlab-ci-gate` and `bitbucket-pipelines-gate`, the two whose `runner` param defaults to
  the container image `debian:stable-slim` — render a job whose default base image lacks
  `curl`, `git` and `ca-certificates`, so a real pipeline on the default runner would fail
  at the CLI-install step before reaching the gate. It is filed as a native GitHub issue on
  the pack's own repo — https://github.com/backstop-ai/ci-workflows/issues/1 (open, filed
  2026-08-12), "gitlab-ci-gate and bitbucket-pipelines-gate default runner
  (debian:stable-slim) lacks curl/git/ca-certificates the rendered pipeline itself requires"
  — rather than as a backstop-core artifact issue, following the `backstop-ai/go-contracts#1`
  precedent that pack-data defects are filed against the pack that owns the data. It is
  explicitly NOT a SPEC-067 defect: nothing in this spec asserts that a rendered file
  EXECUTES in a real pipeline. REQ-005 asserts parse-level validity of the rendered bytes
  (YAML parse plus required top-level keys; structural well-formedness for the Jenkinsfile)
  and REQ-004 asserts the five non-vacuity invariants are PRESENT in those bytes — runtime
  executability is outside every requirement and every claim here, and no claim weakens or
  changes as a result. Whoever picks that issue up should treat it as a pack-data fix
  (runner default, or a documented prerequisite step) in a `ci-workflows` version bump, not
  as a re-opening of this spec.
- **1.0.3** (2026-08-11) — Scoped correction to CLM-050's module-path exemption, closing the
  conflict PLAN-SPEC-067 surfaced during planning. The finding: CLM-050 required every
  occurrence of `github` to fall inside the literal prefix `github.com/`, but
  `cmd/backstop/baseline.go:171` writes that module path REGEX-ESCAPED —
  ``regexp.MustCompile(`github\.com[:/]([^/]+)/([^/.]+)(\.git)?$`)`` — so a strict
  literal-prefix reading failed one real, pre-existing occurrence that is genuinely nothing
  but a module-path reference. **Founder ruling (2026-08-11):** widen the exemption to
  recognize BOTH spellings of the same reference — the plain `github.com/` and the
  regex-escaped `github\.com`. Everything else about the claim is untouched: the check stays
  scoped to the literal lowercase token `github` matched CASE-SENSITIVELY, which was already
  its scope. The ruling explicitly did NOT broaden the claim case-insensitively, so the five
  capitalized mentions in `baseline.go` (a "GitHub Actions" comment, `ensureGitHubAuth` ×2,
  two error strings) remain OUT of scope — they were never what the lowercase-token check
  measured; and it explicitly did NOT adopt the rejected alternative of narrowing to
  "CI-platform-shaped literals" generally, which would have let those five slide by a route
  that also weakened the claim. **REQ-008(a)** carries the identical widening, since the
  requirement states the same rule the claim proves and the two must not drift.
  **CLM-050**'s mandated test name is UNCHANGED
  (`TestCIRecipes_CoreProductionSourceCarriesNoPlatformLiteral`): the test's purpose — core
  production source carries no platform literal — is exactly what it was, and only the
  exemption predicate inside it moves. Prose: Implementation §5 and Sharp Edge 5 restate the
  two-spelling exemption and record why the case-sensitive scope is a deliberate narrowing
  rather than an oversight. No requirement, claim, contract or test is added or removed.
- **1.0.2** (2026-08-11) — Honesty correction pass on 1.0.1's JUSTIFICATIONS; every
  requirement, claim and mandated mechanism from 1.0.1 is retained unchanged except where
  noted. The finding: `backstop pack test`'s phase-3 fixture-EXECUTION step never runs for
  any pack in the current fleet — it is guarded on a manifest field real packs do not
  declare — so 1.0.1's repeated claim that phase 3 "executes every fixture in place, so a
  mis-scoped include would fail the negative-fixture check" described a protection that does
  not exist. **REQ-009**: the basename-glob form and the matching-fixture-name rule are
  KEPT (basename globs are the only include form real semgrep honours, measured; the naming
  rule is now justified as forward-compatible groundwork for whenever the execution defect
  is fixed, explicitly not as present-day protection). Its GitLab two-pattern justification
  is corrected: a dot-prefixed fixture name does match and does produce a finding — measured
  — so the two patterns are justified by the undotted fixture names this spec mandates, not
  by an inability to dot-prefix. **CLM-058** is narrowed from a fixture-execution claim to a
  STRUCTURAL `pack test` claim and its test renamed accordingly; Implementation §4, the
  Overview and Sharp Edge 8 drop the same fabricated mechanism, Sharp Edge 8 restating
  basename width as a real accepted cost with no safety net under it. **REQ-010 / CLM-059**:
  the first-party-action permission is CLOSED to exactly `actions/checkout` (previously "at
  minimum"), matching the Review Question that already treated it as closed and keeping the
  claim decidable from rendered bytes. Prose: the naming-precedent claim is narrowed from
  "no cross-cutting pack anywhere in the fleet" to the published `backstop-ai/*` fleet, since
  root `packs/` already holds non-language-scoped `backstop/base-engines`,
  `backstop/contracts` and `backstop/substantiveness`. New Sharp Edge 10 records that
  `filepath.Match` does not model semgrep's slashless-pattern semantics, so CLM-053..056's
  Go assertions must compare against `filepath.Base(path)`. New Review Questions record that
  real fixture-firing proof is BLOCKED on ISSUE-092 (Pack Test Phase3 Fixtures Cannot
  Fail — open, critical, filed 2026-07-27; the ID was backfilled here after the fact), and
  flag the unresolved fixture-POLARITY disagreement between packval's phase-3 semantics and
  `packs/contracts/pack.yml` as tied to that same defect.
- **1.0.1** (2026-08-11) — Review-driven correction pass; no re-scoping, no requirement
  added or removed. **REQ-009 / CLM-053..056**: the mandated `paths:` scoping form changed
  from "equal to exactly the recipe's target PATH" to a per-platform BASENAME GLOB, because
  the former is empirically unsatisfiable — real semgrep 1.156.0 matches zero files against
  a multi-segment include. (This entry also cited packval phase 3's in-place fixture
  execution as a second reason; that half was fabricated and is retracted in 1.0.2 — the
  semgrep measurement alone carries the change.) The
  underlying reason scoping exists is unchanged. **REQ-007 / CLM-049**: the missing-required-
  param failure is restated as exit 1 through the normal apply path (an op failure inside
  `Substitute`) rather than an exit-2 config error, matching the shipped `pkg/recipe`
  behaviour this spec inherits without modifying. **REQ-010 / CLM-059**: naming a
  first-party platform action (`actions/checkout`) is now explicitly permitted, since the
  previous blanket org/repo prohibition contradicted REQ-004(a)'s `fetch-depth: 0`, which
  has no other expression. **CLM-045..048**: dropped the false "reports the regeneration"
  clause — `preserveOrRegenerate` short-circuits on byte-equality and writes nothing.
  Prose: the `backstop-ai/ci-workflows` name is now stated as INTRODUCING the cross-cutting
  pack category rather than fitting an existing convention; new Sharp Edge 8 records the
  basename glob's cost and the tightening trap; the `Recipes` contract entry is corrected
  from `kind: variable` to the qualified field name.
- **1.0.0** (2026-08-11) — Initial spec, authored from BUNDLE-015 Seed 6 / REQ-018. Scoped
  to the four per-platform gate-workflow recipes only; the seed's OIDC emitter and
  cross-pack parity-check extension are named and deferred for want of a formalized bundle
  requirement and of the vercel pack that owns the other side.
