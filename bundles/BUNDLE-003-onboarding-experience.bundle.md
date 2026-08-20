---
title: "Onboarding Experience — Zero to First Value"
number: BUNDLE-003
created: "2026-04-09"
schema_version: bundle/v2

bundle:
  name: onboarding-experience
  version: "0.11.0"
  created: "2026-04-09"
  updated: "2026-08-20"
  category: feature

status:
  maturity: delivered

problem:
  summary: >
    Backstop has no onboarding path. A consumer must hand-create backstop.yml,
    figure out the artifact-directory layout, know which packs to add, and their
    first run either errors on missing SDLC directories or produces a wall of
    violations. Two profiles onboarded by hand (a full-SDLC greenfield TypeScript
    repo and a pack-only consumer) both reached a clean gate PASS — but only after
    a human absorbed a ranked list of sharp edges the tool gave no help with.
    Every one of those sharp edges is adoption friction. The objective is minimal
    time from having the binary to first value derived: install, one command,
    "this caught something real in my code" — without hand-writing config,
    guessing the layout, or being told the whole codebase is broken.

  user_story: >
    As a tech lead evaluating backstop for my team, I want to run one command in
    my project and see useful observations about my codebase in minutes — without
    hand-writing config, without guessing where artifacts go, and without the tool
    hard-erroring on directories I never created. I want the tool to feel like it
    understands my project, not like it is judging it.

solution:
  approach: >
    A single `backstop init` command that takes a consuming project from zero to
    first value, scaffolded for one of two validated profiles: full-SDLC greenfield
    (adopting backstop's artifact pipeline) or pack-only (adopting packs without the
    SDLC artifacts). Init installs an opinionated base bundle (omakase — bare `init`
    is the full bundle you subtract from via flags) and does ZERO language/framework
    detection; languages enter only via explicit `pack add`. It writes the
    profile-correct backstop.yml, scaffolds the `.backstop/`-rooted artifact layout,
    emits a canonical .gitignore, seeds a local baseline, verifies the toolchain
    actually runs, wires CI via a recipe pack when `--ci <pack>:<recipe>@<version>` is passed
    explicitly (no default platform; the full pinned ref goes to `recipe apply` verbatim and
    omitting the flag skips CI wiring and says so), and captures the first gate
    result as a baseline framed as observation ("here's what we noticed"), not
    judgment. Post-init, default scope is diff-based — you are accountable for new
    patterns, not inherited ones. The init algorithm is specified by transcribing the
    two validated hand-onboarding checklists, not by inventing a flow; each ranked
    sharp edge becomes either an init guardrail or a `backstop doctor` diagnostic.
    Detection, framework recognition, and CI-platform knowledge live in packs as data,
    never in core (a HARD INVARIANT).
    Binary + system-toolchain acquisition and baseline generation are owned elsewhere
    (DIR-001, BUNDLE-007) and are out of scope here.

requirements:
  - id: REQ-001
    version: "1.0.0"
    text: >
      `backstop init` must take a consuming project from "the binary is present" to
      first useful output in a single command, with no manual step required between
      the two (DD-1).
  - id: REQ-002
    version: "1.0.0"
    text: >
      Bare `backstop init` must install the full opinionated base bundle without
      prompting for any input, and must expose subtraction flags (`--only <cap>` /
      `--no-<cap>`) as the only way to narrow that default set. Because it never
      prompts, init must run identically in an interactive terminal and in a headless
      or CI environment (OQ-2, DD-2).
  - id: REQ-003
    version: "1.0.0"
    text: >
      Init must generate the `backstop.yml` correct for the selected profile without
      the consumer hand-writing policy. The full-SDLC greenfield profile gets a minimal
      config (`project:` plus the artifact pipeline enabled). The pack-only profile must
      additionally set `enforcement.policy` `level: off` for every SDLC dimension —
      `test_verification`, `coverage_threshold`, `contract_signature`,
      `test_substantiveness`, `artifact_status_drift` — because those dimensions
      hard-error on a missing `specs/` directory rather than skipping (DD-2).
  - id: REQ-004
    version: "1.0.0"
    text: >
      Init must scaffold the `.backstop/`-rooted artifact layout in consumer repos,
      leaving only `backstop.yml` / `backstop.lock` and `.backstop/` visible at the
      repo root. backstop-core's own root-level artifact layout is a recognized
      framework exception and must not be treated as a layout violation (OQ-1).
  - id: REQ-005
    version: "1.1.0"
    text: >
      Init must emit one canonical `.gitignore` covering the backstop-owned entries the
      validated onboarding produced — `.backstop/packs/`, `.backstop/baseline.json`,
      `.backstop/pack-config-provenance.json` — plus, for each installed pack, every
      engine's declared `stdout_artifact` path, which is the ONLY generated-output
      declaration a pack manifest actually carries (the `StdoutArtifact` field at
      `pkg/pack/manifest.go:97`, yaml key `stdout_artifact`). No language-, framework-, or
      tool-specific path may be enumerated in core. ACCEPTED RESIDUE for v1:
      `stdout_artifact` names only the artifact an engine writes for the gate to read, so
      generated paths a toolchain produces incidentally (dependency directories, native
      coverage output, build output) are NOT covered by it; init leaves those to the
      consumer and says so. That gap is ACCEPTED rather than closed by inventing a new
      pack-manifest "generated output" field — manifest surface design is BUNDLE-004's,
      not this bundle's (DD-7 as corrected 2026-08-12).
    versions:
      - version: "1.0.0"
        text: >
          Init must emit one canonical `.gitignore` covering the full set the validated
          onboarding produced — `.backstop/packs/`, `.backstop/baseline.json`,
          `.backstop/pack-config-provenance.json`, plus every path the installed packs
          declare as generated output — so that ignore contents no longer diverge between
          onboarded repos (DD-7).
      - version: "1.1.0"
        text: >
          Init must emit one canonical `.gitignore` covering the backstop-owned entries the
          validated onboarding produced — `.backstop/packs/`, `.backstop/baseline.json`,
          `.backstop/pack-config-provenance.json` — plus, for each installed pack, every
          engine's declared `stdout_artifact` path, which is the ONLY generated-output
          declaration a pack manifest actually carries (the `StdoutArtifact` field at
          `pkg/pack/manifest.go:97`, yaml key `stdout_artifact`). No language-, framework-, or
          tool-specific path may be enumerated in core. ACCEPTED RESIDUE for v1:
          `stdout_artifact` names only the artifact an engine writes for the gate to read, so
          generated paths a toolchain produces incidentally (dependency directories, native
          coverage output, build output) are NOT covered by it; init leaves those to the
          consumer and says so. That gap is ACCEPTED rather than closed by inventing a new
          pack-manifest "generated output" field — manifest surface design is BUNDLE-004's,
          not this bundle's (DD-7 as corrected 2026-08-12).
        correction: >
          CORRECTION (2026-08-12, v0.10.0). The v1.0.0 clause "every path the installed packs
          declare as generated output" named a pack-manifest field that DOES NOT EXIST —
          verified against `pkg/pack/manifest.go` at HEAD: there is no generated-output
          declaration on the manifest at all. The nearest real surface is the per-engine
          `stdout_artifact` (manifest.go:97), which the requirement now names explicitly. The
          coverage difference is stated rather than papered over: `stdout_artifact` is what an
          engine promises to WRITE FOR THE GATE, not an inventory of everything a toolchain
          leaves on disk, so a residue of un-ignored generated paths remains and is accepted
          for v1. Inventing a new manifest field to close it would be this bundle designing
          another bundle's surface. DD-7 carried the mirrored defect — a literal canonical
          list with TypeScript-specific paths baked into core reasoning — and is corrected in
          the same pass.
  - id: REQ-006
    version: "1.0.0"
    text: >
      Init must run `git init` only when the target directory has no `.git`, and must
      leave an existing repository's git state untouched (DD-11).
  - id: REQ-007
    version: "1.0.0"
    text: >
      Re-running init must converge and never clobber: it detects only
      backstop-neutral facts (presence of `.git`, of `backstop.yml`, of the artifact
      directories), adds only what is missing, overwrites no existing consumer file,
      and reports its findings purely in backstop terms (OQ-6, DD-11, DD-14).
  - id: REQ-008
    version: "1.0.0"
    text: >
      Init must perform ZERO language, framework, or CI-platform detection. No
      language, framework, or CI-platform name may appear in core CLI code on the init
      path; that knowledge lives in packs as data and reaches init only through pack
      manifests and recipes. The `backstop/self` pack enforces this (DD-13, OQ-5, OQ-6).
  - id: REQ-009
    version: "1.0.0"
    text: >
      Init must apply pack-supplied recipes through a single generic mechanism — copy
      a template to the path the recipe itself declares — and must not interpret,
      rewrite, or language-specialize template content. Every language-specific
      artifact init produces (a first source file, toolchain config) must originate in
      a pack recipe, never in core; init adds the backstop layer only and never
      reimplements what an ecosystem scaffolder (`create-next-app`, `cargo new`, …)
      already does (DD-12, DD-7, DD-14).
  - id: REQ-010
    version: "1.0.0"
    text: >
      Languages must enter a consumer project only via an explicit `pack add`;
      multi-language projects are simply multiple packs. Init must have no concept of
      a "primary language" and must never select packs on a project's inspected
      identity (OQ-5, DD-11).
  - id: REQ-011
    version: "1.1.0"
    text: >
      Init must verify the toolchain actually RUNS by executing the pack-declared
      test/compile entrypoint once and confirming success, rather than inferring
      health from package-manager configuration or exit code. Evidence: pnpm 11.11
      exited nonzero with `ERR_PNPM_IGNORED_BUILDS` while vitest ran fine — trusting
      the package manager would have produced a false init failure (DD-6). Init does NOT
      install the project-level dependencies that entrypoint needs (retired REQ-032, see
      Version History 0.10.0): when the check fails because those dependencies are absent,
      init must report the failure as a SETUP step the consumer still owes, naming the pack
      whose entrypoint could not run and pointing at that pack's own documented install
      steps — never silently passing, and never installing anything itself.
    versions:
      - version: "1.0.0"
        text: >
          Init must verify the toolchain actually RUNS by executing the pack-declared
          test/compile entrypoint once and confirming success, rather than inferring
          health from package-manager configuration or exit code. Evidence: pnpm 11.11
          exited nonzero with `ERR_PNPM_IGNORED_BUILDS` while vitest ran fine — trusting
          the package manager would have produced a false init failure (DD-6).
      - version: "1.1.0"
        text: >
          Init must verify the toolchain actually RUNS by executing the pack-declared
          test/compile entrypoint once and confirming success, rather than inferring
          health from package-manager configuration or exit code. Evidence: pnpm 11.11
          exited nonzero with `ERR_PNPM_IGNORED_BUILDS` while vitest ran fine — trusting
          the package manager would have produced a false init failure (DD-6). Init does NOT
          install the project-level dependencies that entrypoint needs (retired REQ-032, see
          Version History 0.10.0): when the check fails because those dependencies are absent,
          init must report the failure as a SETUP step the consumer still owes, naming the pack
          whose entrypoint could not run and pointing at that pack's own documented install
          steps — never silently passing, and never installing anything itself.
        correction: >
          CORRECTION (2026-08-12, v0.10.0, founder-ruled). v1.0.0 was authored while REQ-032
          (init auto-installs pack-declared project dependencies) was in scope, and it silently
          ASSUMED that predecessor: REQ-032 explicitly ordered itself "before the REQ-011
          toolchain-execution check runs", so REQ-011 as written had no answer for a repo whose
          devDependencies are simply not installed. REQ-032 is retired this pass — its mechanism
          needed a pack-command executor BUNDLE-019 reserves but does not build, plus an
          unresolved governance posture in BUNDLE-021 — and the founder chose a docs-only MVP:
          "I think an MVP solution for this is simply including a dependency install guide in a
          pack readme." So the assumption is replaced with the explicit posture above: the check
          still runs and still refuses to infer health, but an uninstalled toolchain is reported
          as owed setup rather than repaired by init.
  - id: REQ-012
    version: "1.1.0"
    text: >
      CONSUMED, NOT BUILT HERE — owner: ISSUE-056 (open, `issues/ISSUE-056-local-first-
      baseline-seeding.issue.md`), which cites this bundle's OQ-3 as its source of direction.
      A gitignored local baseline at `.backstop/baseline.json` must exist so a solo or
      remoteless consumer gets the ratchet from day zero, superseded without migration by the
      CI-generated baseline BUNDLE-007 owns once a team adopts it. THIS BUNDLE'S obligation is
      the consumption only: init seeds that baseline through whatever mechanism ISSUE-056
      delivers, and init's spec must not restate or re-design the seeding machinery (OQ-3,
      DD-10).
    versions:
      - version: "1.0.0"
        text: >
          Init must seed a gitignored local baseline at `.backstop/baseline.json` so a
          solo or remoteless consumer gets the ratchet from day zero, and that local
          baseline must be superseded without migration by the CI-generated baseline
          BUNDLE-007 owns once a team adopts it (OQ-3, DD-10).
      - version: "1.1.0"
        text: >
          CONSUMED, NOT BUILT HERE — owner: ISSUE-056 (open, `issues/ISSUE-056-local-first-
          baseline-seeding.issue.md`), which cites this bundle's OQ-3 as its source of direction.
          A gitignored local baseline at `.backstop/baseline.json` must exist so a solo or
          remoteless consumer gets the ratchet from day zero, superseded without migration by the
          CI-generated baseline BUNDLE-007 owns once a team adopts it. THIS BUNDLE'S obligation is
          the consumption only: init seeds that baseline through whatever mechanism ISSUE-056
          delivers, and init's spec must not restate or re-design the seeding machinery (OQ-3,
          DD-10).
        correction: >
          CORRECTION (2026-08-12, v0.10.0). v1.0.0 restated as this bundle's own build obligation
          a mechanism ISSUE-056 already owns on the issue→plan track — the issue is open, cites
          BUNDLE-003 OQ-3 as its direction, and its Solution items ARE this requirement and
          REQ-013. Two artifacts owning one mechanism is exactly the divergence hazard the
          existence-in-world rule exists to prevent, so ownership is settled in ISSUE-056's favor
          (it is already filed, already scoped, and pure baseline machinery, which this bundle's
          own Out of Scope section separately excludes). Nothing is dropped: the requirement
          survives as the CONSUMPTION contract init's spec must honor.
  - id: REQ-013
    version: "1.1.0"
    text: >
      CONSUMED, NOT BUILT HERE — owner: ISSUE-056 (open), same lineage as REQ-012. The
      `baseline_comparison` message emitted on a remoteless repository must be self-consistent
      — it must not simultaneously claim a baseline is required and that none can exist. This is
      pure gate machinery with no init code in it; it is recorded here only because init is the
      moment a consumer first meets the broken message, and init's spec must not carry it as
      work (OQ-3).
    versions:
      - version: "1.0.0"
        text: >
          The `baseline_comparison` message emitted on a remoteless repository must be
          self-consistent — it must not simultaneously claim a baseline is required and
          that none can exist (OQ-3).
      - version: "1.1.0"
        text: >
          CONSUMED, NOT BUILT HERE — owner: ISSUE-056 (open), same lineage as REQ-012. The
          `baseline_comparison` message emitted on a remoteless repository must be self-consistent
          — it must not simultaneously claim a baseline is required and that none can exist. This is
          pure gate machinery with no init code in it; it is recorded here only because init is the
          moment a consumer first meets the broken message, and init's spec must not carry it as
          work (OQ-3).
        correction: >
          CORRECTION (2026-08-12, v0.10.0). v1.0.0 sat in the `backstop init` spec seed while
          containing no init code whatsoever — it is a `pkg/gate` message fix — and while this
          bundle's own Out of Scope section excluded baseline machinery, a direct self-
          contradiction. ISSUE-056 quotes the exact contradictory string from
          `pkg/gate/gate.go` and owns the fix. Reclassified to consumption, same as REQ-012.
  - id: REQ-014
    version: "1.1.0"
    text: >
      Init must capture the first gate run as the baseline and present it as
      observation, not failure: findings grouped by category with counts, phrased as
      what was noticed, and an exit code of 0 for "baseline captured" rather than 1
      for "violations found" (DD-3). PRECEDENCE: this exit-0 rule governs only the
      CLASSIFICATION OF PRE-EXISTING FINDINGS — it says inherited violations are not an init
      failure. It does NOT make init's overall exit 0 when an init STEP failed to deliver what
      it promised. Where one run triggers both this and REQ-035 (brownfield CI preserve,
      DD-15's REFUSE posture), REQ-035 WINS on the exit code: baseline-captured and
      no-gate-actually-wired are orthogonal facts, and the broken promise is the one the exit
      code must carry. Both are still reported in full.
    versions:
      - version: "1.0.0"
        text: >
          Init must capture the first gate run as the baseline and present it as
          observation, not failure: findings grouped by category with counts, phrased as
          what was noticed, and an exit code of 0 for "baseline captured" rather than 1
          for "violations found" (DD-3).
      - version: "1.1.0"
        text: >
          Init must capture the first gate run as the baseline and present it as
          observation, not failure: findings grouped by category with counts, phrased as
          what was noticed, and an exit code of 0 for "baseline captured" rather than 1
          for "violations found" (DD-3). PRECEDENCE: this exit-0 rule governs only the
          CLASSIFICATION OF PRE-EXISTING FINDINGS — it says inherited violations are not an init
          failure. It does NOT make init's overall exit 0 when an init STEP failed to deliver what
          it promised. Where one run triggers both this and REQ-035 (brownfield CI preserve,
          DD-15's REFUSE posture), REQ-035 WINS on the exit code: baseline-captured and
          no-gate-actually-wired are orthogonal facts, and the broken promise is the one the exit
          code must carry. Both are still reported in full.
  - id: REQ-015
    version: "1.0.0"
    text: >
      After init, an unflagged gate run must be diff-scoped — evaluating changed files
      against the seeded baseline so inherited patterns are separated from introduced
      ones — with full-codebase evaluation available on demand (DD-4).
  - id: REQ-016
    version: "1.2.0"
    text: >
      Init wires CI only when the consumer passes `--ci` EXPLICITLY, and the flag's value is
      the FULL PINNED RECIPE REF in `<pack>:<recipe>@<version>` form. The exact command-line
      shape is:
      `backstop init --ci backstop-ai/ci-workflows:github-actions-gate@1.0.0`.
      That whole string is opaque to core and is handed to `recipe apply` VERBATIM: core
      constructs no part of it — not the pack name, not the recipe id, not the version — and
      never interprets, defaults, maps, or completes it. There is no default platform and no
      inferred one; core holds no platform literal, no list of platform names, no pack name,
      and no version. ~~The flag's value is passed STRAIGHT THROUGH to the CI recipe pack as
      `recipe apply <pack>:<value>` — whatever recipe names that pack declares are the accepted
      values~~ — **CORRECTED 2026-08-13 (v1.2.0):** that form is not a valid recipe ref and left
      core constructing the `<pack>:` half, which is itself a core-side pack-name literal — the
      DD-13 bake this requirement exists to prevent. Verified against
      `pkg/recipe/resolve.go:60-88` (`ParseRecipeRef`): the only accepted ref format is
      `<pack>:<recipe>@<version>`, where the pin is MANDATORY and strict semver — "there is no
      'latest', no default version, and no tolerance branch (CLM-049)" — and an unpinned ref is
      a hard parse error, so a literal reading of the v1.1.0 form produced a call that always
      errors. The installed `backstop-ai/ci-workflows@v0.1.0` declares four discrete, separately
      named recipes (`github-actions-gate`, `gitlab-ci-gate`, `bitbucket-pipelines-gate`,
      `jenkins-gate`, each at recipe version `1.0.0`) with no variant abstraction and no
      pack-declared default; init requires no such abstraction to exist, and a consumer naming
      an entirely different pack is equally valid precisely because core never inspects the ref.
      When `--ci` is OMITTED, CI wiring is SKIPPED and init must SAY SO rather than skip
      silently: state that no CI was wired, name `backstop recipe apply` plus the pinned
      `<pack>:<recipe>@<version>` ref shape as the way to wire it later, and exit as a
      deliberate no-op — a skipped optional step is not an error (contrast REQ-035's broken
      promise). No `ci` verb may be added: the generated job chains the existing
      platform-agnostic commands (`pack install` → `baseline pull` → `gate`), which is what
      gives local and CI runs identical semantics (OQ-7, DD-13). Founder ruling
      2026-08-12 (see Version History): the CI pack exists to help consumers scaffold CI
      quickly, not to be opinionated about which platform they use, so the platform is an
      explicit flag.
    versions:
      - version: "1.0.0"
        text: >
          Init must wire CI by default by applying the selected variant from a CI recipe
          pack that holds per-platform templates as data (`--ci github` as the default
          variant, `--no-ci` to opt out). No `ci` verb may be added: the generated job
          chains the existing platform-agnostic commands (`pack install` → `baseline pull`
          → `gate`), which is what gives local and CI runs identical semantics (OQ-7,
          DD-13).
      - version: "1.1.0"
        text: >
          Init wires CI only when the consumer passes `--ci <recipe>` EXPLICITLY. There is no
          default platform and no inferred one: core holds no platform literal, no list of
          platform names, and no name→recipe mapping. The flag's value is passed STRAIGHT
          THROUGH to the CI recipe pack as `recipe apply <pack>:<value>` — whatever recipe names
          that pack declares are the accepted values, and core never interprets them. The
          installed `backstop-ai/ci-workflows@v0.1.0` declares four discrete, separately named
          recipes (`github-actions-gate`, `gitlab-ci-gate`, `bitbucket-pipelines-gate`,
          `jenkins-gate`) with no variant abstraction and no pack-declared default; init requires
          no such abstraction to exist. When `--ci` is OMITTED, CI wiring is SKIPPED and init must
          SAY SO rather than skip silently: state that no CI was wired, name `backstop recipe
          apply` plus the installed CI pack's declared recipe names as the way to wire it later,
          and exit as a deliberate no-op — a skipped optional step is not an error (contrast
          REQ-035's broken promise). No `ci` verb may be added: the generated job chains the
          existing platform-agnostic commands (`pack install` → `baseline pull` → `gate`), which
          is what gives local and CI runs identical semantics (OQ-7, DD-13). Founder ruling
          2026-08-12 (see Version History): the CI pack exists to help consumers scaffold CI
          quickly, not to be opinionated about which platform they use, so the platform is an
          explicit flag.
      - version: "1.2.0"
        text: >
          Init wires CI only when the consumer passes `--ci` EXPLICITLY, and the flag's value is
          the FULL PINNED RECIPE REF in `<pack>:<recipe>@<version>` form. The exact command-line
          shape is:
          `backstop init --ci backstop-ai/ci-workflows:github-actions-gate@1.0.0`.
          That whole string is opaque to core and is handed to `recipe apply` VERBATIM: core
          constructs no part of it — not the pack name, not the recipe id, not the version — and
          never interprets, defaults, maps, or completes it. There is no default platform and no
          inferred one; core holds no platform literal, no list of platform names, no pack name,
          and no version. ~~The flag's value is passed STRAIGHT THROUGH to the CI recipe pack as
          `recipe apply <pack>:<value>` — whatever recipe names that pack declares are the accepted
          values~~ — **CORRECTED 2026-08-13 (v1.2.0):** that form is not a valid recipe ref and left
          core constructing the `<pack>:` half, which is itself a core-side pack-name literal — the
          DD-13 bake this requirement exists to prevent. Verified against
          `pkg/recipe/resolve.go:60-88` (`ParseRecipeRef`): the only accepted ref format is
          `<pack>:<recipe>@<version>`, where the pin is MANDATORY and strict semver — "there is no
          'latest', no default version, and no tolerance branch (CLM-049)" — and an unpinned ref is
          a hard parse error, so a literal reading of the v1.1.0 form produced a call that always
          errors. The installed `backstop-ai/ci-workflows@v0.1.0` declares four discrete, separately
          named recipes (`github-actions-gate`, `gitlab-ci-gate`, `bitbucket-pipelines-gate`,
          `jenkins-gate`, each at recipe version `1.0.0`) with no variant abstraction and no
          pack-declared default; init requires no such abstraction to exist, and a consumer naming
          an entirely different pack is equally valid precisely because core never inspects the ref.
          When `--ci` is OMITTED, CI wiring is SKIPPED and init must SAY SO rather than skip
          silently: state that no CI was wired, name `backstop recipe apply` plus the pinned
          `<pack>:<recipe>@<version>` ref shape as the way to wire it later, and exit as a
          deliberate no-op — a skipped optional step is not an error (contrast REQ-035's broken
          promise). No `ci` verb may be added: the generated job chains the existing
          platform-agnostic commands (`pack install` → `baseline pull` → `gate`), which is what
          gives local and CI runs identical semantics (OQ-7, DD-13). Founder ruling
          2026-08-12 (see Version History): the CI pack exists to help consumers scaffold CI
          quickly, not to be opinionated about which platform they use, so the platform is an
          explicit flag.
  - id: REQ-017
    version: "1.2.0"
    text: >
      When the consumer asked for CI (a full pinned ref was passed to `--ci`, REQ-016 v1.2.0)
      and that ref cannot be resolved — the named pack is not installed, the pack declares no
      such recipe, or the pinned version does not match the one the recipe declares — the CI
      step of init must fail LOUDLY, while every other init step still completes. Init
      implements NO detection of its own: it neither identifies "the CI pack" nor probes for
      installation. ~~the CI step of init must fail LOUDLY with guidance naming the pack to
      add~~ — **CORRECTED 2026-08-13 (v1.2.0):** since the consumer names the pack explicitly in
      the ref (REQ-016 v1.2.0), core has nothing to detect, and the not-installed case is
      already covered by `recipe apply`'s own existing error path. Verified against
      `pkg/recipe/resolve.go:100-131` (`ResolveRecipe`): an uninstalled pack errors with
      "pack %q is not among the installed packs (installed: ...)", an undeclared recipe errors
      with "pack %q declares no recipe %q in its recipes: index (indexed: ...)", and a pin
      mismatch names both versions — each already naming what was missing and what IS available.
      Init's obligation is therefore to SURFACE that error verbatim, attributed to the CI step,
      and to carry on with the remaining steps; it must not add bespoke detection machinery, and
      any actionable guidance beyond what the resolve error already prints belongs to `recipe
      apply`, not to init. There is no baked per-platform fallback, so silent success is
      unrepresentable. This requirement governs only the asked-for-and-unavailable case; `--ci`
      omitted is REQ-016's skipped-and-said-so no-op, not a failure (OQ-7).
    versions:
      - version: "1.0.0"
        text: >
          When no CI recipe pack is installed, the CI step of init must fail LOUDLY with
          guidance naming the pack to add, while every other init step still completes.
          There is no baked per-platform fallback, so silent success is unrepresentable
          (OQ-7).
      - version: "1.1.0"
        text: >
          When the consumer asked for CI (`--ci <recipe>` was passed, REQ-016 v1.1.0) and no CI
          recipe pack is installed, the CI step of init must fail LOUDLY with guidance naming the
          pack to add, while every other init step still completes. There is no baked
          per-platform fallback, so silent success is unrepresentable. This requirement governs
          only the asked-for-and-unavailable case; `--ci` omitted is REQ-016's skipped-and-said-so
          no-op, not a failure (OQ-7).
      - version: "1.2.0"
        text: >
          When the consumer asked for CI (a full pinned ref was passed to `--ci`, REQ-016 v1.2.0)
          and that ref cannot be resolved — the named pack is not installed, the pack declares no
          such recipe, or the pinned version does not match the one the recipe declares — the CI
          step of init must fail LOUDLY, while every other init step still completes. Init
          implements NO detection of its own: it neither identifies "the CI pack" nor probes for
          installation. ~~the CI step of init must fail LOUDLY with guidance naming the pack to
          add~~ — **CORRECTED 2026-08-13 (v1.2.0):** since the consumer names the pack explicitly in
          the ref (REQ-016 v1.2.0), core has nothing to detect, and the not-installed case is
          already covered by `recipe apply`'s own existing error path. Verified against
          `pkg/recipe/resolve.go:100-131` (`ResolveRecipe`): an uninstalled pack errors with
          "pack %q is not among the installed packs (installed: ...)", an undeclared recipe errors
          with "pack %q declares no recipe %q in its recipes: index (indexed: ...)", and a pin
          mismatch names both versions — each already naming what was missing and what IS available.
          Init's obligation is therefore to SURFACE that error verbatim, attributed to the CI step,
          and to carry on with the remaining steps; it must not add bespoke detection machinery, and
          any actionable guidance beyond what the resolve error already prints belongs to `recipe
          apply`, not to init. There is no baked per-platform fallback, so silent success is
          unrepresentable. This requirement governs only the asked-for-and-unavailable case; `--ci`
          omitted is REQ-016's skipped-and-said-so no-op, not a failure (OQ-7).
  - id: REQ-018
    version: "1.1.0"
    text: >
      Init must install its omakase base through portable git-ref pack references, so the
      `backstop.lock` it commits contains no machine-specific paths — that half is init's own
      and stays here. The local-provenance half is CONSUMED, NOT BUILT HERE — owner: ISSUE-055
      (open, `issues/ISSUE-055-local-provenance-cache-for-local-packs.issue.md`), which cites
      this bundle's OQ-4 as its source of direction: local-path packs a consumer adds afterward
      are restored on the same machine from a gitignored local-provenance record, and promoting
      one to shareable stays an explicit git-ref step. Init's spec must not design or restate
      that lock-schema/pack-CLI change (OQ-4).
    versions:
      - version: "1.0.0"
        text: >
          Init must install its omakase base through portable git-ref pack references, so
          the `backstop.lock` it commits contains no machine-specific paths. Local-path
          packs a consumer adds afterward are restored on the same machine from a
          gitignored local-provenance cache, and promoting one to shareable stays an
          explicit git-ref step (OQ-4).
      - version: "1.1.0"
        text: >
          Init must install its omakase base through portable git-ref pack references, so the
          `backstop.lock` it commits contains no machine-specific paths — that half is init's own
          and stays here. The local-provenance half is CONSUMED, NOT BUILT HERE — owner: ISSUE-055
          (open, `issues/ISSUE-055-local-provenance-cache-for-local-packs.issue.md`), which cites
          this bundle's OQ-4 as its source of direction: local-path packs a consumer adds afterward
          are restored on the same machine from a gitignored local-provenance record, and promoting
          one to shareable stays an explicit git-ref step. Init's spec must not design or restate
          that lock-schema/pack-CLI change (OQ-4).
        correction: >
          CORRECTION (2026-08-12, v0.10.0). Same shape as the REQ-012/REQ-013 correction: v1.0.0
          restated a mechanism ISSUE-055 already owns (open, OQ-4-sourced, quoting the real
          `LockEntry.LocalPath` state in `pkg/pack/distribution/lockfile.go`), and this bundle's
          own Out of Scope section already listed "gitignored local-provenance lock mechanism" as
          NOT designed here — so the requirement contradicted its own bundle. Split rather than
          deleted, because the two halves have different owners: "init's own installs use portable
          git-refs" is genuinely init behavior and is retained as this bundle's requirement; the
          local-provenance cache is ISSUE-055's and is now marked consumed.
  - id: REQ-019
    version: "1.0.0"
    text: >
      Init's happy path must execute the transcribed hand-onboarding sequence
      (scaffold via recipe → write profile-correct `backstop.yml` → create artifact
      dirs → write `.gitignore` → add each pack → run the gate), and the acceptance
      bar is the outcome that sequence already achieved by hand: on a fresh repo, init
      followed by `backstop gate` reaches PASS with zero violations, for both the
      full-SDLC greenfield and the pack-only profile (DD-8, DD-2).
  - id: REQ-020
    version: "1.0.0"
    text: >
      `backstop doctor` must exist as the diagnostic command for a setup that is off,
      with one check per ranked sharp edge from the hand-onboarding write-ups, and
      init must name it in the guidance it prints whenever a step it cannot complete
      is diagnosable (DD-8 corollary, DD-9).
  - id: REQ-021
    version: "1.1.0"
    text: >
      `backstop version` must stamp the build commit and build date, so a stale binary is
      identifiable on sight. Reporting a bare `dev` as the VERSION STRING of a non-release
      build is CORRECT and must be preserved, not eliminated: `cmd/backstop/version.go`
      deliberately refuses to report anything release-shaped for a locally built binary —
      it rejects `(devel)`, any `+` build metadata (so `v0.11.0+dirty` is a tag plus
      uncommitted changes, not a release), and Go pseudo-versions — precisely so a local
      build cannot pretend to be an official release. What must become identifiable is the
      BUILD, not the release: `dev` plus a commit and a build date distinguishes a fresh
      dev build from a stale one, which is the diagnosability the dogfood incident actually
      needed (DD-9 as corrected 2026-08-11).
    versions:
      - version: "1.0.0"
        text: >
          `backstop version` must stamp the build commit and build date, and must never
          report a bare `dev`, so a stale binary is identifiable on sight (DD-9).
      - version: "1.1.0"
        text: >
          `backstop version` must stamp the build commit and build date, so a stale binary is
          identifiable on sight. Reporting a bare `dev` as the VERSION STRING of a non-release
          build is CORRECT and must be preserved, not eliminated: `cmd/backstop/version.go`
          deliberately refuses to report anything release-shaped for a locally built binary —
          it rejects `(devel)`, any `+` build metadata (so `v0.11.0+dirty` is a tag plus
          uncommitted changes, not a release), and Go pseudo-versions — precisely so a local
          build cannot pretend to be an official release. What must become identifiable is the
          BUILD, not the release: `dev` plus a commit and a build date distinguishes a fresh
          dev build from a stale one, which is the diagnosability the dogfood incident actually
          needed (DD-9 as corrected 2026-08-11).
        correction: >
          CORRECTION (2026-08-11, v0.8.1, founder-ruled). The v1.0.0 clause "must never report a
          bare `dev`" was WRONG and is withdrawn: it contradicted a deliberate anti-spoofing
          design already shipped in `cmd/backstop/version.go`, whose whole purpose is that a
          non-release build reports `dev` rather than something that looks like a release.
          Founder ruling, verbatim: "it can be dev for a local build if that's standard
          practice." Verified against the code and the running binary the same day: `./bin/backstop
          version` reports `backstop version dev`, and `resolveVersion` returns `dev` for every
          input that is not an injected goreleaser stamp or a strictly released module version.
          Nothing is lost by the withdrawal — the underlying stale-binary concern is carried by
          the surviving build-commit/build-date clause above, by REQ-022 (which names the stale
          binary as the cause instead of surfacing the downstream engine error — the exact
          dogfood failure mode), and by REQ-028 (which stamps producing-binary identity onto every
          result so a stale green can be quarantined). No NEW mechanism was ordered by the ruling
          and none is invented here.
  - id: REQ-022
    version: "1.1.0"
    text: >
      When a pack declares core capabilities the running binary does not provide, the
      consumer-facing failure must NAME THAT GAP — identifying the missing capability and the
      pack that requires it — instead of surfacing only the downstream engine error (e.g.
      `declared stdout_artifact ... not produced`), which gives no hint the binary is the
      cause. That misdiagnosis, not the comparison algorithm, is what this bundle observed and
      what it requires be fixed. The MECHANISM is DEFERRED WHOLLY TO BUNDLE-020 (Pack Core
      Version Compatibility) and must be consumed from it, not re-decided here: BUNDLE-020 DD-4
      (founder-resolved 2026-07-26) settles that a pack declares named CAPABILITY CONTRACTS
      scoped to the pack↔core wire seam and that compatibility is a CAPABILITY-SET comparison,
      deliberately NOT a version-ordering comparison; BUNDLE-020's OQ-2 (where it is enforced —
      add time, lock verification, gate preflight) and OQ-3 (failure posture) are still OPEN and
      are not answered here. This requirement therefore states the DIAGNOSTIC OUTCOME only
      (DD-9, as corrected 2026-08-12).
    versions:
      - version: "1.0.0"
        text: >
          `pack add` and `gate` must compare the binary's capability against the features
          the pack manifest declares it requires, and on skew must fail with a diagnostic
          naming the binary as older than the pack requires — instead of surfacing the
          downstream engine error (e.g. `declared stdout_artifact ... not produced`) that
          gives no hint the binary is the cause (DD-9).
      - version: "1.1.0"
        text: >
          When a pack declares core capabilities the running binary does not provide, the
          consumer-facing failure must NAME THAT GAP — identifying the missing capability and the
          pack that requires it — instead of surfacing only the downstream engine error (e.g.
          `declared stdout_artifact ... not produced`), which gives no hint the binary is the
          cause. That misdiagnosis, not the comparison algorithm, is what this bundle observed and
          what it requires be fixed. The MECHANISM is DEFERRED WHOLLY TO BUNDLE-020 (Pack Core
          Version Compatibility) and must be consumed from it, not re-decided here: BUNDLE-020 DD-4
          (founder-resolved 2026-07-26) settles that a pack declares named CAPABILITY CONTRACTS
          scoped to the pack↔core wire seam and that compatibility is a CAPABILITY-SET comparison,
          deliberately NOT a version-ordering comparison; BUNDLE-020's OQ-2 (where it is enforced —
          add time, lock verification, gate preflight) and OQ-3 (failure posture) are still OPEN and
          are not answered here. This requirement therefore states the DIAGNOSTIC OUTCOME only
          (DD-9, as corrected 2026-08-12).
        correction: >
          CORRECTION (2026-08-12, v0.10.0). v1.0.0 reached into a DIFFERENT bundle's problem space
          and pre-decided two things it had no standing to decide. (1) It prescribed the comparison
          shape — "fail with a diagnostic naming the binary as OLDER THAN the pack requires" — which
          is a version-ordering comparison, precisely the model BUNDLE-020 DD-4 deliberately rejected
          in favor of capability SETS ("a capability set is derived from what the binary actually
          implements"); it also named the enforcement points (`pack add` and `gate`), which is
          BUNDLE-020's still-OPEN OQ-2, and implied a hard-fail posture, which is its still-OPEN OQ-3.
          (2) It SELF-CONFLICTED with REQ-021: `cmd/backstop/version.go` deliberately reports bare
          `dev` for every non-release build, so the common case has no orderable version to be
          "older" than anything — the v1.0.0 text was unsatisfiable for exactly the binary shape
          REQ-021 says is correct. Rewritten to state the outcome (name the capability gap) and
          consume BUNDLE-020's eventual mechanism. BUNDLE-020's open questions are NOT resolved
          here and must not be resolved from this bundle.
  - id: REQ-023
    version: "1.0.0"
    text: >
      `backstop doctor` must expose the REQ-011 toolchain-execution check as a
      standalone, re-runnable diagnostic with actionable remediation, so a consumer
      can diagnose a broken toolchain without re-running init (DD-6, doctor seed).
  - id: REQ-024
    version: "1.0.0"
    text: >
      `backstop doctor` must check the installed runtime/toolchain version against the
      stack policy declared by the installed packs and warn on deviation. The policy
      values are pack data — no runtime version may be baked into core. Evidence: the
      dogfood machine ran only non-LTS Node against a documented Node-LTS stack
      decision and nothing warned (doctor seed, DD-13).
  - id: REQ-025
    version: "1.0.0"
    text: >
      `backstop doctor` must validate a consumer repo's artifact layout against the
      canonical `.backstop/`-rooted layout and report each deviation with the path it
      expected (OQ-1, doctor seed).
  - id: REQ-026
    version: "1.0.0"
    text: >
      The schema cohort a binary reports must be derived from the CONTENT of its
      embedded schemas, not from their names or count, so that any change to an
      embedded schema's bytes changes the cohort identifier. Evidence (verified
      2026-08-11): `computeCohortID` (`cmd/backstop/root.go`) builds the cohort string
      from schema file PATHS alone — `11-schemas[adr/v1,...,spec/v1]` — so the in-place
      revision of `bundle/v2` that BUNDLE-014 shipped (adding the `Draft`-prefixed
      section names, `text:` in place of `statement:`, per-REQ `version:`, and
      `bundle.updated`) left the reported cohort byte-identical. A name-derived cohort
      cannot detect the skew that produced the false green in REQ-027, which is why
      content-derivation is a requirement and not an implementation detail (DD-9).
  - id: REQ-027
    version: "1.0.0"
    text: >
      `backstop artifact validate` and `backstop gate` must assert the validating
      binary's schema cohort against each artifact's declared `schema_version` and MUST
      REFUSE to report green when they cannot prove the cohort covers the schema
      features the artifact uses. A validator too old — or too new — for an artifact's
      schema must fail loudly; "cannot determine" is a failure, never a pass. Evidence:
      BUNDLE-001 in bclabs-portal was authored, promoted to `defined`, and revised five
      times (0.2.0 → 0.2.5) across multiple sessions with `artifact validate` reporting
      GREEN throughout, while violating the then-current `bundle/v2` schema in 41
      places; the nonconformance surfaced instantly under a current binary. The only
      schema cross-check performed today is `schema_version` PREFIX vs artifact type
      (`pkg/validate/{bundle,spec,issue,adr}.go`) — nothing compares schema revisions.
      This guard belongs in core's own validate/gate rather than any single harness,
      because the skew bit in a different toolchain than the one that caught it (DD-9).
  - id: REQ-028
    version: "1.0.0"
    text: >
      Every `gate` and `artifact validate` result, in both human and `--json` form, must
      record the version and schema cohort of the binary that produced it, so a stored
      or forwarded result can be quarantined rather than projected as truth when it was
      produced by a validator that could not have caught the violation. Evidence: gate
      results already stamp `GitSHA` and `GeneratedAt` (`cmd/backstop/gate.go`) but carry
      no binary identity, so a recorded green is indistinguishable from a green a stale
      validator could not have withheld (DD-9).
  - id: REQ-029
    version: "1.0.0"
    text: >
      Artifact discovery must resolve the artifact root from configuration and apply
      that one resolution uniformly across gate status resolution, validation reference
      resolution, and scaffolding — so a consuming repo can hold the whole artifact
      chain under `.backstop/` and be discovered, while backstop-core keeps the root
      layout (the OQ-1 framework exception) by configuring the repo root. Evidence
      (verified 2026-08-11, unchanged since the 2026-07-17 write-up): three independent
      hardcodings of the root layout — `pkg/gate/artifact_status.go` joins projectRoot
      with `specs`/`bundles`/`issues`/`directives`/`plans`; `pkg/validate/resolved_by.go`
      maps BUNDLE→`bundles`, SPEC→`specs`, ISSUE→`issues`, DIR→`directives`;
      `pkg/scaffold/scaffold.go` scaffolds into root `specs`/`issues`/`directives`/
      `bundles`. No `artifact_root` key exists in the `backstop.yml` schema or loader.
      REQ-004 is unimplementable — and actively harmful — until this lands (OQ-1).
  - id: REQ-030
    version: "1.1.0"
    text: >
      Artifact discovery must name the artifact root it actually scanned in gate output, and
      must fail loudly when a CONFIGURED artifact root does not exist on disk rather than
      walking absent directories and reporting a passing dimension. It must ALSO surface
      artifact-shaped files it finds OUTSIDE the resolved root — reporting them as
      undiscovered-and-ungated, naming the path and the root that excluded them — rather than
      silently ignoring them. An artifact root that EXISTS but is empty remains a legitimate
      pass, and out-of-root artifacts are REPORTED, not adopted: discovery does not widen its
      root, it tells the truth about what it left out. Evidence: backstop-runtime already
      placed `.backstop/bundles/` while its `specs/` stayed at root; because the binary only
      discovers `projectRoot/bundles`, those bundles are almost certainly not gated and nothing
      says so — the same false-green family as REQ-027 (OQ-1).
    versions:
      - version: "1.0.0"
        text: >
          Artifact discovery must name the artifact root it actually scanned in gate output,
          and must fail loudly when a configured artifact root does not exist on disk rather
          than walking absent directories and reporting a passing dimension. An artifact root
          that EXISTS but is empty remains a legitimate pass — the validated greenfield
          profile reached gate PASS with empty artifact dirs and that outcome must not
          regress. Evidence: backstop-runtime already placed `.backstop/bundles/` while its
          `specs/` stayed at root; because the binary only discovers `projectRoot/bundles`,
          those bundles are almost certainly not gated and nothing says so — the same
          false-green family as REQ-027 (OQ-1).
      - version: "1.1.0"
        text: >
          Artifact discovery must name the artifact root it actually scanned in gate output, and
          must fail loudly when a CONFIGURED artifact root does not exist on disk rather than
          walking absent directories and reporting a passing dimension. It must ALSO surface
          artifact-shaped files it finds OUTSIDE the resolved root — reporting them as
          undiscovered-and-ungated, naming the path and the root that excluded them — rather than
          silently ignoring them. An artifact root that EXISTS but is empty remains a legitimate
          pass, and out-of-root artifacts are REPORTED, not adopted: discovery does not widen its
          root, it tells the truth about what it left out. Evidence: backstop-runtime already
          placed `.backstop/bundles/` while its `specs/` stayed at root; because the binary only
          discovers `projectRoot/bundles`, those bundles are almost certainly not gated and nothing
          says so — the same false-green family as REQ-027 (OQ-1).
        correction: >
          CORRECTION (2026-08-12, v0.10.0). v1.0.0 did NOT close the backstop-runtime case it cites
          — verified against `pkg/gate/artifact_status.go` at HEAD. Two behaviors defeat it: a
          missing type-directory is not an error (`walkArtifactDir`/`walkBundleDir`/`walkPlanDir`
          each return nil on `os.IsNotExist`, lines 382/409/437), and an unconfigured root defaults
          to `projectRoot`, which by definition EXISTS. So the only condition v1.0.0 fails loudly on
          — an absent CONFIGURED root — is never reached in the very scenario the requirement was
          written for: backstop-runtime's `.backstop/bundles/` stay silently undiscovered under a
          perfectly existent default root, and nothing in v1.0.0's text obliges a word about them.
          The surfacing clause is what actually closes it.
  - id: REQ-033
    version: "1.0.0"
    text: >
      The pack-only profile must not silently forfeit coverage enforcement. Because
      REQ-003 sets `coverage_threshold` to `level: off` for a consumer that has no specs
      to source thresholds from, init's pack-only profile must instead wire a
      spec-independent coverage floor (a global and/or per-glob minimum under
      `enforcement`) over the coverage records the installed packs already produce.
      Evidence: `ts-coverage` emits per-file coverage records, but `coverage_threshold`
      sources its thresholds from SDLC specs, so the pack-only consumer ends up holding
      coverage data with no "fail under N%" knob. The knob itself is a core
      enforcement-policy capability this bundle CONSUMES and does not design (see Out of
      Scope / Dependencies) (DD-2).
  - id: REQ-034
    version: "1.0.0"
    text: >
      An artifact's schema identity, as asserted at validation time, must be CONTENT-DERIVED —
      a digest of the schema file(s) actually used to validate it — and not the bare
      `schema_version` const string. Either the artifact itself carries that digest, or the
      validation record produced when it is checked does; this bundle requires one of the two
      and leaves the choice between them to the guards spec. Without it REQ-027 cannot detect
      the incident it exists for: `artifacts/bundle/v2/schema.json` declares
      `"schema_version": {"const": "bundle/v2"}` — REVISION-FREE — so BUNDLE-014's IN-PLACE
      revision of bundle/v2 left every artifact's declared value byte-identical before and
      after, and a stale validator asserting "my cohort covers bundle/v2" reports green on both
      sides of the change. REQ-026 makes the BINARY's cohort content-derived; this requirement
      makes the ARTIFACT's schema identity revision-bearing, and only the pair makes the
      comparison in REQ-027 meaningful — a same-named schema that changed underneath is
      detectably different, not merely same-named (DD-15).
  - id: REQ-035
    version: "1.0.0"
    text: >
      Init's CI step must report a BROWNFIELD PRESERVE as a gap, not a success. When the CI
      recipe's target file already exists, the recipe applier's `create` op family is
      never-clobber by design — it preserves the consumer's file and reports `preserved`, so
      the apply succeeds, an adoption record is written, and NO backstop gate is wired into
      that consumer's CI. Init must state that outcome explicitly in its own output and in its
      exit posture: name the preserved file, state that no gate was wired, and give the
      consumer the next action. Silent success is a broken promise, not un-adopted capability,
      so DD-15's "on 'I cannot tell', REFUSE" posture governs here rather than
      loud-≠-blocking. Evidence: ISSUE-119 (open) — three of the four CI platforms
      (`.gitlab-ci.yml`, `bitbucket-pipelines.yml`, `Jenkinsfile`) have exactly ONE
      conventional entry point apiece, so ANY existing project adopting backstop hits this;
      only GitHub's per-workflow-file layout escapes it. REQ-017 does not cover this case — it
      covers only "no CI recipe pack installed" — and the underlying merge/insert-op gap is
      ISSUE-119's to close, not this bundle's; init owes the honest report either way (OQ-7,
      DD-15).
---

# Onboarding Experience

## Current Thinking

### Provenance: dogfood-grounded

This bundle began (0.1.0, April 2026) as a thought exercise — analyzing onboarding
from a tech lead's perspective. It is now **dogfood-grounded**: the requirements for
`backstop init` and `backstop doctor` derive from onboarding two real repos by hand,
one write-up per profile (see References). Those two write-ups are the empirical
requirements corpus. Both currently live as untracked field notes in other repos, so
the load-bearing substance is lifted into the design decisions below (especially DD-8)
rather than left to survive by citation alone.

### Grounding pass, 2026-08-11 — requirements re-checked against the real dogfood evidence

The 25 requirements authored on 2026-08-10 were derived from this bundle's own resolved OQs
and DDs. On 2026-08-11 they were re-checked line by line against the two hand-onboarding
source documents themselves, rather than against this bundle's summary of them:

- `~/src/projects/backstop-packs/BACKSTOP-INIT-REQUIREMENTS.md` — the **pack-only** profile.
- `~/src/projects/backstop-packs/BACKSTOP-INIT-DOGFOOD-FULL-SDLC.md` — the **full-SDLC
  greenfield** profile (bclabs-portal, 2026-07-12, with a second version-skew instance
  appended 2026-07-16 and the artifact-layout founder ruling appended 2026-07-17).

Note the second document is the same corpus the References section lists as
`bclabs-portal/docs/dogfood/init-sharp-edges.md`; it now lives in `backstop-packs/`
alongside its pack-only sibling. Both remain untracked field notes — this bundle is still
their durable home.

Fifteen concrete findings were checked. Ten were already covered by REQ-001..025. Two were
ruled out of scope with reasons recorded below (semgrep rule-id readability; `artifact new`
number reservation). Eight new requirements (REQ-026..033) closed the remaining gaps, and the
code state behind each was verified against HEAD on 2026-08-11 rather than trusted from notes
written in July — `computeCohortID` is still path-derived, all three artifact-discovery
hardcodings are still present, no `artifact_root` config key exists, and gate results still
carry no producing-binary identity. The two serious ones (schema-cohort false green,
artifact-root silent undiscovery) are generalized into DD-15.

One finding was RETIRED by its own successor evidence: the pack-only document's recommendation
that init write `onlyBuiltDependencies: [esbuild]` into `pnpm-workspace.yaml` was superseded
three weeks later when pnpm 11.11 ignored that key entirely and the failure proved cosmetic.
That reversal is precisely the argument for REQ-011 (verify the toolchain RUNS) — and writing
a package-manager-specific key from core would violate DD-13 regardless.

### The current state

There is no `backstop init`. A consumer must:
1. Hand-create `backstop.yml` — and know the profile-correct contents (a pack-only
   consumer must set `enforcement.policy` `level: off` for every SDLC dimension or the
   gate hard-errors on a missing `specs/` directory).
2. Decide an artifact-directory layout with no canonical answer (live repos already
   diverge — see OQ-1).
3. Know which packs to add and add each one.
4. Write a `.gitignore` that matches what the packs and gate actually emit.
5. Run `backstop gate` — and debug whatever it surfaces with no diagnostic help.

Both profiles reached gate PASS by hand, which proves the flow exists; each step above
is a sharp edge the tool should absorb.

### The target experience

1. Obtain the binary — owned by DIR-001, out of scope here.
2. `backstop init` — installs the omakase base (prompt-free; subtract via flags),
   writes profile-correct config, scaffolds the `.backstop/`-rooted layout, emits the
   canonical `.gitignore`, seeds a local baseline, verifies the toolchain runs, and
   wires CI when a full pinned recipe ref is passed explicitly — e.g.
   `backstop init --ci backstop-ai/ci-workflows:github-actions-gate@1.0.0` (~~by default~~ /
   ~~`--ci <recipe>`~~ — **corrected 2026-08-13, v0.10.2**: there is no default platform, and the
   flag's value is the whole `<pack>:<recipe>@<version>` ref, passed to `recipe apply` verbatim;
   omitting `--ci` skips CI wiring and init reports the skip plus how to wire it later).
   Zero language/framework detection —
   languages arrive via explicit `pack add`.
3. Fix one thing, run a diff-scoped check, watch it go green. That is first value.

### Key principles (product thesis)

- **Baseline as observation, not judgment.** If the first thing backstop says is "427
  violations, exit 1," the tech lead hears "your code sucks." Init captures the first
  run as a baseline and presents it as "here's what we noticed," grouped by category —
  same data, opposite emotional response.
- **Diff-scoped by default post-init.** After init, an unflagged check looks only at
  changed files. The baseline records what existed at init so the gate separates
  inherited patterns from introduced ones. You are accountable for new code from day one.
- **Two profiles, not one generic flow.** The single most important empirical finding:
  onboarding is not one-size-fits-all. Full-SDLC and pack-only consumers need
  materially different config, and picking the wrong one silently produces the
  hard-error experience (see DD-2).

## Draft Requirements

**33 requirements** are carried in the frontmatter `requirements` block: REQ-001..030 plus
REQ-033..035. REQ-031 and REQ-032 are RETIRED (0.10.0 — see Version History); requirement IDs
are never renumbered, so the gap is deliberate and permanent. REQ-001..025 are each derived from
an already-resolved OQ, an existing DD, or a spec seed in this bundle. REQ-026..033 were added in
0.8.0 by the grounding pass described under Current Thinking. REQ-034..035 were added in 0.10.0
by the review-correction pass.

They partition cleanly across **three** spec seeds (ranges revised 0.10.0):

| Seed | Requirements | Count |
|---|---|---|
| Trustworthy-green guards (core) | REQ-021, REQ-022, REQ-026 – REQ-030, REQ-034 | 8 |
| `backstop init` | REQ-001 – REQ-019, REQ-033, REQ-035 | 21 |
| `backstop doctor` | REQ-020, REQ-023, REQ-024, REQ-025 | 4 |

The guards seed is new in 0.8.0. It exists because those requirements are changes to core's
validate / gate / discovery paths that init and doctor both DEPEND on but neither OWNS —
folding them into either seed would have made that seed's scope dishonest. See Spec Seeds.

**Seed reassignment, 0.10.0:** REQ-021 (build-identity stamp on `version`) and REQ-022 (name the
capability gap instead of the downstream engine error) MOVED from the doctor seed to the guards
seed. Neither is a doctor CHECK — REQ-021 changes `cmd/backstop/version.go`, REQ-022 changes the
`pack add` / `gate` failure path — and both are exactly what the guards seed's carve-out rule
describes: core changes init and doctor both depend on but neither owns. Doctor is left as
REQ-020/023/024/025, a clean independently-specable diagnostics command whose every requirement
is a check it runs.

**OWNERSHIP CORRECTION 2026-08-14 (v0.10.3) — the guards seed's 8 are owned 7-by-SPEC-068,
1-unowned.** The table row above still reads "REQ-021, REQ-022, REQ-026 – REQ-030, REQ-034 | 8",
which is the correct SEED partition and stays as written — but it must not be read as "SPEC-068
covers these." SPEC-068 (Trustworthy Green Guards, v1.2.3) pins SEVEN of the eight: **REQ-021,
REQ-026, REQ-027, REQ-028, REQ-029, REQ-030, REQ-034**. **REQ-022 is UNOWNED — blocked on
BUNDLE-020.** REQ-022 v1.1.0 defers its mechanism wholly to BUNDLE-020 (Pack Core Version
Compatibility), whose DD-4 is founder-resolved (capability-SET comparison, not version ordering)
but whose **OQ-2** (where the comparison is enforced — add time, lock verification, or gate
preflight) and **OQ-3** (failure posture) are both still OPEN, with BUNDLE-020 itself still
`exploring`. SPEC-068 declined to pin it deliberately and says so in its own Dependencies section:
specifying the diagnostic outcome with no producer would mean either shipping a stub renderer or
answering another bundle's open questions — which this bundle's own consumption boundary forbids.
**REQ-022's future owner is a delta spec against SPEC-068, written once BUNDLE-020 resolves OQ-2
and OQ-3.** This is the same treatment REQ-033 carries as DANGLING and REQ-012 / REQ-013 / REQ-018
carry as CONSUMED: the requirement stays in the seed, but no spec owns it yet.

**OWNERSHIP CORRECTION 2026-08-20 (v0.11.0) — the doctor seed's 4 are owned 3-by-SPEC-070,
1-CARVED-OUT.** The table row above reads "`backstop doctor` | REQ-020, REQ-023, REQ-024,
REQ-025 | 4", which is the correct SEED partition and stays as written — but, exactly as with the
guards row, it must not be read as "SPEC-070 covers these." SPEC-070 (Backstop Doctor) implements
THREE: **REQ-020, REQ-023, REQ-025**. **REQ-024 is CARVED OUT — blocked on an unowned
pack-manifest surface.** The requirement asks doctor to check the installed runtime/toolchain
version against the stack policy the installed packs declare; verified at HEAD, no pack-declared
stack-policy surface exists and no artifact owns creating one. SPEC-070 escalated this during
authoring and it was ruled 2026-08-13 to carve it out rather than invent the field — the SAME
rule this bundle already applied to REQ-005 v1.1.0, that pack-manifest surface design is
BUNDLE-004's, not this bundle's. Closing it needs (a) BUNDLE-004 adopting a stack-policy manifest
surface and probably (b) BUNDLE-021 settling the posture toward probing software already
installed on the machine. Until then a doctor check would either read nothing (a vacuous check)
or invent another bundle's surface. **REQ-024 carries NO requirement and NO mandated claim in
SPEC-070**, which instead holds a TRIPWIRE — an absence claim asserting the check registry holds
exactly the declared ids and reads no stack-policy surface — so REQ-024 cannot be closed from the
wrong artifact without a test going red. **`issues/ISSUE-121-pack-manifest-missing-stack-policy-
surface.issue.md` was filed for the gap (open, created 2026-08-13) and is being homed under
BUNDLE-004**, so it does not live only as a note. Same posture as REQ-022: the requirement stays
in the seed, but no spec owns it yet.

**DELIVERY NOTE 2026-08-20 (v0.11.0) — REQ-033 is OWNED but SHIPPED UNSATISFIED.** Unlike
REQ-022 and REQ-024, REQ-033 (spec-independent coverage floor for the pack-only profile) IS
pinned by SPEC-069 — but what shipped is an honest REPORT of the gap, not the wired knob. The
DANGLING flag recorded at 0.10.0 was never cleared: no artifact ever adopted the coverage-floor
knob REQ-033 depends on, and `backstop.yml`'s `additionalProperties: false` rejects writing it,
so init reports the gap rather than wiring it. SPEC-069 carries absence claims proving no schema
surface was invented to fake it. Recorded here so the three postures this bundle now holds are
distinguishable at a glance: **CONSUMED** (REQ-012 / REQ-013 / REQ-018's local-provenance half —
another artifact builds it), **CARVED OUT / UNOWNED** (REQ-022, REQ-024 — no spec owns it yet),
and **OWNED-BUT-UNSATISFIED** (REQ-033 — a spec owns it and shipped a truthful report instead of
the mechanism).

**Consumed-not-built requirements.** REQ-012, REQ-013 and REQ-018's local-provenance half are
marked CONSUMED (0.10.0): the mechanisms belong to ISSUE-056 and ISSUE-055 respectively, both
open on the issue→plan track and both already citing this bundle's own resolved OQ-3/OQ-4 as
their source of direction. They stay listed because init's spec must honor them as inputs; they
are not this bundle's work to build, and a spec author must not re-scope them. This also removes
a self-contradiction: REQ-013 is pure `pkg/gate` message machinery and sat in the init seed while
Out of Scope separately excluded baseline machinery.

### `backstop init` (REQ-001 – REQ-019, REQ-033, REQ-035)

- **Shape of the command** (REQ-001, REQ-002): one command from binary to first value with no
  manual step in between (DD-1); omakase base installed prompt-free with subtract-via-flags,
  which is also what makes init headless/CI-safe (OQ-2).
- **Profile correctness** (REQ-003): the headline two-profile fork — full-SDLC greenfield gets
  a minimal config; pack-only additionally gets `enforcement.policy` `level: off` on all five
  SDLC dimensions, because they hard-error rather than skip on a missing `specs/` (DD-2).
- **What init writes** (REQ-004, REQ-005): the `.backstop/`-rooted consumer layout with
  backstop-core's root layout as the explicit framework exception (OQ-1), and one canonical
  `.gitignore` that ends the divergence observed across onboarded repos (DD-7). **CORRECTED
  2026-08-12 (v0.10.0):** REQ-005 v1.0.0 deferred the non-backstop entries to "every path the
  installed packs declare as generated output" — no such pack-manifest field exists. It now
  defers to each engine's declared `stdout_artifact` (the real surface) and STATES the residue
  it does not cover, rather than inventing a manifest field. DD-7's literal TypeScript-flavored
  list is correspondingly demoted to an example of what a pack might declare.
- **Thin-executor boundary** (REQ-008, REQ-009, REQ-010): zero language/framework/CI-platform
  detection, no such literal in core init code, recipes applied by a generic
  copy-template-to-declared-path mechanism, and languages entering only via explicit
  `pack add` (DD-13, DD-12, OQ-5). REQ-009 is also where DD-14 lands: init adds the backstop
  layer, ecosystem scaffolders own the project.
- **Idempotency** (REQ-006, REQ-007): `git init` only when there is no `.git`; re-init
  converges, never clobbers, and stays framework-blind (DD-11, OQ-6).
- **Ground truth over configuration** (REQ-011): init executes the pack-declared toolchain
  entrypoint once rather than trusting a package manager's exit code (DD-6). **CORRECTED
  2026-08-12 (v0.10.0):** with REQ-032 retired, init no longer installs the project
  dependencies that entrypoint needs — REQ-011 v1.1.0 states the replacement posture, which is
  to report an uninstalled toolchain as owed SETUP (naming the pack and pointing at its
  documented steps), never to pass silently and never to install anything itself.
- **Baseline and scope** (REQ-012, REQ-013, REQ-014, REQ-015): a gitignored local baseline
  exists day-zero (OQ-3) and the self-contradictory remoteless message is fixed (OQ-3) — both
  now marked CONSUMED, owner ISSUE-056, per the 0.10.0 correction — while init itself frames the
  first run as observation with exit 0 (DD-3) and makes post-init default scope diff-based
  (DD-4).
- **CI** (REQ-016, REQ-017, REQ-035): wired via a CI recipe pack whose templates are
  data, no `ci` verb (the job chains `pack install` → `baseline pull` → `gate`), and a loud but
  non-blocking failure when CI was asked for and the named recipe cannot be resolved (OQ-7).
  ~~wired by default~~ **CORRECTED 2026-08-13 (v0.10.1):** REQ-016 drops the default entirely —
  `--ci` must be passed explicitly, because a `--ci github` default would put a platform literal
  plus a name→recipe mapping in core, which DD-13 calls the bug. ~~its value goes straight
  through to the pack's own recipe name~~ **FURTHER CORRECTED 2026-08-13 (v0.10.2):** REQ-016
  v1.2.0 makes the flag's value the FULL PINNED REF —
  `--ci backstop-ai/ci-workflows:github-actions-gate@1.0.0` — handed to `recipe apply` verbatim,
  because `ParseRecipeRef` (`pkg/recipe/resolve.go:60-88`) accepts only
  `<pack>:<recipe>@<version>` with a mandatory strict-semver pin (CLM-049); a bare recipe name
  would both always error and leave core holding the pack name. Omitting `--ci` is a reported
  skip, not an error; REQ-017 v1.2.0 is scoped to the asked-for-and-unresolvable case and
  satisfied by surfacing `recipe apply`'s own resolve error rather than any init-side detection.
  **REQ-035, added 0.10.0**, closes the
  case REQ-017 does not reach: a BROWNFIELD consumer whose platform config already exists gets
  `create`'s never-clobber preserve, a successful-looking apply, an adoption record, and no gate
  wired at all (ISSUE-119). Init must say so in words rather than report success.
- **Lock portability** (REQ-018): init's own installs use portable git-refs so the committed
  lock carries no machine-specific paths — init's own requirement. The gitignored
  local-provenance cache that restores locally-added packs is CONSUMED, owner ISSUE-055 (OQ-4;
  split 0.10.0).
- **Acceptance** (REQ-019): the transcribed hand-onboarding sequence is the happy path, and
  the bar is the outcome it already produced by hand — init then `backstop gate` reaching PASS
  with zero violations on a fresh repo, for both profiles (DD-8).
- **Pack-only coverage floor** (REQ-033, added 0.8.0): REQ-003 turns `coverage_threshold`
  off for the pack-only profile, which would otherwise leave that consumer holding coverage
  records with no threshold to fail against. Init wires a spec-independent floor instead.
  **NOTE (0.10.0): the knob REQ-033 wires has no owner.** See Out of Scope / Dependencies —
  this is a genuinely dangling dependency, not an assumed-covered one.
- **~~Layout sequencing guard (REQ-031)~~ — RETIRED 2026-08-12 (v0.10.0).** REQ-031 said init
  must keep consuming-repo artifacts at the ROOT layout "until REQ-029 ships", gating REQ-004 on
  it. The declared seed ORDER makes that condition dead on arrival: the guards seed (which owns
  REQ-029) is sequenced FIRST, ahead of init, so by the time init's spec is written REQ-029 has
  already landed and REQ-031's blocking condition is already false — while REQ-031 as written
  formally contradicts REQ-004, which mandates the `.backstop/` layout. The sequencing intent is
  NOT lost: it is exactly what the Spec Seeds ordering rationale states ("REQ-004 is
  unimplementable without REQ-029"), which is the right place for a sequencing constraint. If
  the seeds are ever re-ordered, that rationale is where the constraint must be re-asserted.
- **~~Pack-declared project dependencies (REQ-032)~~ — RETIRED 2026-08-12 (v0.10.0),
  founder-ruled.** REQ-032 had init install the devDeps a pack's engines invoke but do not
  vendor, from pack recipe data. It required a pack-command-execution mechanism that does not
  exist and is not this bundle's to build: BUNDLE-019 explicitly RESERVES the `step` op without
  executing it ("BUNDLE-019 owns its executor"), and BUNDLE-021 (Pack Command Execution
  Governance, `exploring`) holds the unresolved governance question of what posture core takes
  toward pack-declared command execution at all. Building it here would have meant standing up
  new execution infrastructure across two unbuilt bundles' open surfaces. Founder ruling,
  verbatim: *"I think an MVP solution for this is simply including a dependency install guide in
  a pack readme."* The MVP path is therefore a PACK-AUTHORING CONVENTION, not a backstop
  mechanism — see DD-12's 0.10.0 note. REQ-011 v1.1.0 absorbs the consequence: init reports an
  uninstalled toolchain rather than repairing it.

### `backstop doctor` (REQ-020, REQ-023 – REQ-025)

- **The command** (REQ-020): one check per ranked sharp edge; init points at it rather than
  absorbing diagnosis (DD-8 corollary).
- **Standalone diagnostics** (REQ-023, REQ-024, REQ-025): re-run the toolchain-execution check
  outside init (DD-6), check runtime version against pack-declared stack policy (the
  unenforced Node-LTS observation), and validate the artifact layout against the canonical
  `.backstop/` root (OQ-1).
  **CORRECTION (2026-08-20, v0.11.0):** REQ-024 — the runtime-version-vs-stack-policy check — is
  CARVED OUT and did NOT ship with SPEC-070. The pack-declared stack-policy surface this bullet
  presumes does not exist at HEAD, and creating it is BUNDLE-004's call, not this bundle's
  (ruled 2026-08-13; ISSUE-121 filed and open). REQ-023 and REQ-025 shipped as described. See the
  ownership correction under the seed table above.
- **MOVED OUT 2026-08-12 (v0.10.0):** the version-skew pair REQ-021 / REQ-022 previously lived
  here. Neither is a check doctor runs — REQ-021 changes what `cmd/backstop/version.go` stamps,
  REQ-022 changes what the `pack add` / `gate` failure path says — so both now sit in the guards
  seed, whose stated charter is exactly this: core changes init and doctor both depend on but
  neither owns. Doctor still SURFACES their output; it does not own the change. The 2026-08-11
  correction to REQ-021 (bare `dev` is the CORRECT version string for a non-release build; what
  must be identifiable is the BUILD, via commit + build date) travels with it — see the REQ-021
  v1.1.0 correction and DD-9 mechanic (a).

### Trustworthy-green guards (REQ-021, REQ-022, REQ-026 – REQ-030, REQ-034)

The doctor requirements above make version skew **diagnosable**. These make it **unable to
certify**. The distinction is the whole point of DD-15: a misleading error stops you, but a
false green propagates — promotions, hand-offs, and derived specs and plans accrue on a
foundation that was never actually validated.

- **Cohort integrity** (REQ-026, REQ-027, REQ-028, REQ-034): the reported cohort must be derived
  from schema CONTENT (today it is derived from schema paths, so an in-place `bundle/v2`
  revision is invisible); `validate` and `gate` must refuse green when they cannot prove their
  cohort covers the artifact's schema; and every result must carry the producing binary's
  identity so a stale green can be quarantined downstream.
  **REQ-034, added 0.10.0 — the missing other half.** REQ-027 as written asserts the binary's
  cohort against "each artifact's declared `schema_version`" — but that declared value is
  revision-FREE: `artifacts/bundle/v2/schema.json` pins `"schema_version": {"const":
  "bundle/v2"}`, and the incident REQ-027 is named after was an IN-PLACE revision of bundle/v2
  itself. The artifact said `bundle/v2` before the revision and `bundle/v2` after it, so a stale
  validator asserting "my cohort covers bundle/v2" is telling the truth about the string and
  lying about the schema — green on both sides. REQ-026 makes the BINARY side content-derived;
  REQ-034 makes the ARTIFACT side revision-bearing (a digest of the schema files actually used,
  carried by the artifact or by the validation record). Neither alone detects an in-place
  revision; the pair does, which is why REQ-034 is a requirement rather than an implementation
  note on REQ-027.
- **Artifact-root resolution** (REQ-029, REQ-030): discovery resolves the artifact root from
  config through ONE resolver shared by gate, validate, and scaffold; discovery names the root
  it scanned, fails loudly on a configured root that is absent, and — per the 0.10.0 correction
  — SURFACES artifact-shaped files it finds outside the resolved root. That last clause is what
  actually closes the backstop-runtime case: a missing type-directory is not an error in
  `pkg/gate/artifact_status.go`, and an unconfigured root defaults to `projectRoot`, which
  always exists, so "fail on an absent configured root" never fires there. An
  existing-but-empty root stays a pass, preserving the validated greenfield outcome.
- **Version-skew guards** (REQ-021, REQ-022 — moved here 0.10.0): the build-identity stamp on
  `version`, and naming the capability gap instead of surfacing the downstream engine error.
  REQ-022 was also RESTATED this pass to stop pre-deciding BUNDLE-020's open questions — it now
  states the diagnostic outcome and consumes BUNDLE-020's capability-set mechanism (DD-4) rather
  than prescribing a version-ordering comparison BUNDLE-020 deliberately rejected.
  **CORRECTED 2026-08-14 (v0.10.3) — these two are NOT jointly owned.** SPEC-068 (v1.2.3) pins
  REQ-021 and leaves **REQ-022 UNOWNED**: its mechanism is BUNDLE-020's, and BUNDLE-020's OQ-2
  (enforcement point) and OQ-3 (failure posture) are still open, so there is no producer for the
  diagnostic REQ-022 describes. REQ-022's owner is a future delta spec against SPEC-068, not this
  seed's first spec. See the ownership correction under Draft Requirements.

### Deliberately NOT requirements here

- Baseline generation mechanics, binary distribution, and system-toolchain acquisition —
  owned by BUNDLE-007 and DIR-001 (DD-10). REQ-012 and REQ-013 are init's *consumption* of
  the baseline, not its machinery — and as of 0.10.0 they say so in their own text, with
  ISSUE-056 named as owner.
- The pack-recipe capability itself, the concrete CI recipe pack, and the gitignored
  local-provenance lock-schema change — consumed by REQ-009/REQ-016/REQ-018 but designed
  elsewhere (see Out of Scope / Dependencies). ~~The recipe capability remains a BLOCKING
  dependency for the init spec.~~ **CORRECTION (2026-08-12, v0.10.0): NOT BLOCKING — DELIVERED.**
  The recipe capability shipped as SPEC-054 (`implemented`) under BUNDLE-015 / DIR-019, and the
  CI recipe pack shipped as SPEC-067 (`implemented`), published and installed as
  `backstop-ai/ci-workflows@v0.1.0`. DIR-019's own 2026-08-12 correction clears the DIR-002
  blocking-dependency claim explicitly. Both remain out of THIS bundle's scope — they are
  consumed, not designed here — but the reason is ownership, no longer absence.
- Pack-declared project dependency INSTALLATION — retired as REQ-032 (0.10.0, founder-ruled).
  The MVP is a pack-authoring convention (packs document their own install steps in their own
  README), not a backstop mechanism; the executor it would have required belongs to BUNDLE-019
  and its governance posture to BUNDLE-021.
- The cascading `coverage_threshold` misdiagnosis when tests fail — recorded under
  Observations as a pack-side messaging concern, not init scope.
- Registry-era pack auto-detection — explicitly dissolved by OQ-5.
- The `enforcement.coverage.min_pct` knob ITSELF — a core enforcement-policy capability.
  REQ-033 requires init to WIRE a spec-independent coverage floor for the pack-only profile;
  designing the knob belongs to enforcement policy (see Out of Scope / Dependencies).
- Semgrep's path-derived rule IDs (e.g.
  `backstop/typescript-standards/backstop.packs...rules.security.ts.security.no-eval`) — a
  core-side output/waiver-token readability cleanup, explicitly logged as cosmetic and
  non-blocking in the pack-only write-up. Real, but it is gate-output ergonomics, not
  onboarding; it needs its own issue.
- `artifact new` number reservation (no gap-fill, no `--number`, stray reservation tags) — a
  `pkg/scaffold` CLI defect, not init behavior. Recorded under Observations with its verified
  root cause so it can be filed separately. REQ-020's "one check per ranked sharp edge" would
  sweep a doctor-side orphan-tag check in by implication; whether to make that explicit is a
  founder call, not one this pass took.

## Draft Design Decisions

- **DD-1: `backstop init` is the single zero-to-first-value command.** It installs the
  opinionated base bundle (omakase; DD-2 / OQ-2), writes profile-correct config, scaffolds
  the `.backstop/`-rooted artifact layout (OQ-1), emits the canonical `.gitignore`, seeds a
  local baseline (OQ-3), verifies the toolchain runs, and wires CI on an explicit
  `--ci <pack>:<recipe>@<version>` (~~by default~~ / ~~`--ci <recipe>`~~ — **corrected
  2026-08-13, v0.10.1 and v0.10.2**; OQ-7). It
  does ZERO language/framework detection (DD-11 / DD-13). No manual steps between having the
  binary and first useful output. Owned downstream by DIR-002.

- **DD-2 (headline): Init scaffolds for one of TWO validated profiles.** The **full-SDLC
  greenfield** profile (repo adopting backstop's artifact pipeline: artifact dirs + a
  minimal `backstop.yml` carrying ~~`project:` + `language:`~~ **`project:` only — CORRECTED
  2026-08-12 (v0.10.0)**) reaches gate PASS on clean
  defaults with zero policy boilerplate. The **pack-only** profile (a consumer adopting
  packs but NOT the SDLC artifacts) must instead set `enforcement.policy` `level: off`
  for every SDLC dimension (`test_verification`, `coverage_threshold`,
  `contract_signature`, `test_substantiveness`, `artifact_status_drift`), because those
  dimensions hard-error on a missing `specs/` directory rather than skipping. Init must
  generate the correct config for the chosen profile so neither consumer hand-writes
  policy. Evidence: the full-SDLC dogfood of bclabs-portal (empty repo → gate PASS, 10
  passed / 2 skipped / 0 violations, 2026-07-12) and the pack-only profile captured in
  `backstop-packs/BACKSTOP-INIT-REQUIREMENTS.md`. This asymmetry is a genuine init design
  fork; earlier versions of this bundle imagined only one generic flow.
  **CORRECTION (2026-08-12, v0.10.0): `language:` is struck from the greenfield config.** The
  key is RETIRED — `pkg/config/config.go:21-36` states it outright: "SPEC-046: the
  single-language `language` field is RETIRED — a project is described by its declared packs,
  not one baked language. An existing backstop.yml may still carry a `language:` key; it is
  absorbed by `LegacyKeys` below and ignored." Init writing it would emit a dead key that bakes
  a language into the very first file a consumer sees, contradicting REQ-003 (which already
  correctly omits it), REQ-008, REQ-010, and the zero-baked-language invariant. The same strike
  is applied to DD-8 step 2, which matters more: REQ-019 makes DD-8's sequence the NORMATIVE
  transcription source for init's happy path, so a spec author would otherwise have copied the
  dead key straight into the spec.

- **DD-3: The first run is framed as baseline observation, not failure.** Output uses
  "here's what we noticed," grouped by category and count; exit code is 0 (baseline
  captured), not 1 (violations found). The emotional framing is a product decision that
  matters as much as the data.

- **DD-4: Post-init default scope is diff-based.** An unflagged check operates on changed
  files only; the baseline records what existed at init so the gate distinguishes
  inherited from introduced patterns. Full-codebase checks are available on demand.

- **DD-5 (RETRACTED — superseded by OQ-5 / DD-11):** ~~Language detection drives default
  pack selection.~~ Init does ZERO language detection. Languages enter only via explicit
  `pack add`; multi-language = multiple packs. Retained here as a record of the reversal:
  detection was a baked-language assumption (a thin-executor violation), and removing it
  dissolves the multi-language question entirely (no "primary language" to pick). See the
  OQ-5 resolution and the hard invariant DD-13.

- **DD-6: Init verifies the toolchain actually RUNS.** It executes the language's
  test/compile entrypoint once (e.g. vitest / tsc) and confirms success rather than
  trusting package-manager configuration, whose semantics shift between majors. Evidence:
  pnpm 11.11 ignored `onlyBuiltDependencies` and exited nonzero with
  `ERR_PNPM_IGNORED_BUILDS`, which turned out cosmetic (vitest ran fine). Trusting the
  package-manager exit code would have produced a false init failure; executing the
  toolchain is ground truth.

- **DD-7: Init emits one canonical `.gitignore` and scaffolds at least one source file.**
  ~~The canonical ignore list is the superset that worked in the dogfood: `.backstop/packs/`,
  `.backstop/baseline.json`, `.backstop/pack-config-provenance.json`,
  `.backstop/ts-coverage/`, `.backstop/ts-test-results.json`, `node_modules/`,
  `coverage/`.~~ At least one source file is scaffolded because a compiler run over an empty
  repo can RED on "no inputs" (observed: `tsc` TS18003) — that scaffolded file comes from a
  pack recipe, never from core (REQ-009). Evidence: `.gitignore` diverged across onboarded
  repos (one ignored provenance + baseline; another ignored only `packs/`); a single canonical
  list removes the divergence.
  **CORRECTION (2026-08-12, v0.10.0): the literal list above is DEMOTED to an example and is no
  longer the canonical list.** It enumerated TypeScript-specific paths (`node_modules/`,
  `coverage/`, `.backstop/ts-coverage/`, `.backstop/ts-test-results.json`) as core reasoning,
  which DD-13 forbids outright. The canonical list is now: the three BACKSTOP-OWNED entries
  (`.backstop/packs/`, `.backstop/baseline.json`, `.backstop/pack-config-provenance.json`)
  stated literally in core, PLUS whatever each installed pack's engines declare as their
  `stdout_artifact`. Read the struck paths as an ILLUSTRATION of what a TypeScript pack's
  `stdout_artifact` declarations might supply — `.backstop/ts-coverage/…` and
  `.backstop/ts-test-results.json` are plausibly exactly that — and `node_modules/` /
  `coverage/` as the ACCEPTED RESIDUE: `stdout_artifact` names what an engine writes for the
  gate to read, not everything a toolchain leaves behind, so those stay the consumer's to
  ignore. See REQ-005 v1.1.0 for the full statement, including why no new pack-manifest field
  is invented to close the residue.

- **DD-8: The init algorithm is specified BY transcribing the validated hand-onboarding
  checklist, not by re-inventing a flow.** The two hand write-ups are the empirical
  requirements corpus (one document per profile — see References); the greenfield doc's
  "Manual steps performed" list IS init's happy-path step sequence. Captured here so it
  survives the source docs being deleted, the full-SDLC greenfield sequence is:
  1. Scaffold a minimal project with ≥1 source file + one test + minimal config
     (toolchain devDeps, `tsconfig.json`, workspace config) and install dependencies.
  2. Write a minimal `backstop.yml` (~~`project:` + `language:`~~ **`project:` only —
     CORRECTED 2026-08-12 (v0.10.0); `language:` is a RETIRED, ignored key per SPEC-046 /
     `pkg/config/config.go:21-36`, and transcribing it here would bake a language into init's
     normative happy path** — for greenfield; the pack-only profile adds the
     `enforcement.policy` `level: off` lines per DD-2).
  3. Create the artifact dirs (layout per OQ-1).
  4. Write the canonical `.gitignore` (DD-7).
  5. `backstop pack add <ref>` for each pack (toolchain, standards, secrets, contracts,
     substantiveness).
  6. `backstop gate` → iterate to PASS.
  Corollary: each ranked sharp edge from the write-ups is either a `doctor` check or an
  init guardrail — init automates the happy path, `doctor` diagnoses the deviations.
  Rationale: the flow is already validated by reaching gate PASS on a real repo by hand;
  transcribing "what the human did" is lower-risk than designing a new sequence.

- **DD-9: Init and `doctor` make CLI/pack version skew diagnosable.** Two mechanics:
  (a) `backstop version` stamps the build commit and date — **CORRECTED 2026-08-11 (v0.8.1,
  founder-ruled): the original trailing clause "never a bare `dev`" is WITHDRAWN.** It
  contradicted the shipped anti-spoofing design in `cmd/backstop/version.go`, which reports
  `dev` for any non-release build on purpose so a local build cannot masquerade as a release.
  Founder ruling: "it can be dev for a local build if that's standard practice." The mechanic
  survives as build IDENTITY (commit + date) making a stale dev build distinguishable from a
  fresh one — not as a prohibition on the string `dev`; (b)
  `pack add` and `gate` compare the binary's capability against the pack manifest's
  required features and fail loudly with ~~"your backstop is older than this pack requires"~~
  a diagnostic NAMING THE MISSING CAPABILITY and the pack that requires it, instead of
  surfacing a downstream engine error. **CORRECTION (2026-08-12, v0.10.0):** the struck
  phrasing prescribed a VERSION-ORDERING comparison, which is a mechanism BUNDLE-020 (Pack Core
  Version Compatibility) owns and has already founder-resolved AGAINST — its DD-4 settles that
  compatibility is a CAPABILITY-SET comparison over named contracts at the pack↔core wire seam,
  deliberately not an ordering of versions; its OQ-2 (where enforced) and OQ-3 (failure posture)
  remain OPEN and are not answered from this bundle. "Older than" is also unsatisfiable against
  the binary shape REQ-021 endorses, since a local `dev` build has no orderable version at all.
  What survives here is the OUTCOME — the diagnostic must name the gap — with the mechanism
  consumed from BUNDLE-020. See REQ-022 v1.1.0. Evidence: the highest-pain sharp edge in
  the 2026-07-12 dogfood — a stale binary reporting bare `dev` predated the pack
  producer/convert split, and the gate reported `declared stdout_artifact ... not produced`
  quoting the plain engine command, giving no hint the binary was simply too old. This
  consumed most of the debugging time. Feeds the `backstop doctor` spec seed.

- **DD-10: Adjacent scope is owned elsewhere and is out of scope here (settled).**
  Baseline mechanics are owned by **BUNDLE-007** — resolved as: baseline is a CI-generated
  post-merge artifact, cached locally at `.backstop/baseline.json` (gitignored), pulled by
  the gate, ratchet-only (can only go down), never committed or hand-edited. Binary
  distribution (Homebrew tap, GitHub releases, cross-platform builds) and system-toolchain
  acquisition are owned by **DIR-001** (Release Workflow), now the top backlog priority.
  This bundle references those boundaries and does not re-litigate them; init consumes their
  outputs (a binary is present; a baseline can be generated), it does not implement them.

- **DD-11: Greenfield init = `git init` (only if not already a repo) + backstop base.**
  Init's job is to add the backstop layer, not the project. It runs `git init` only when
  there is no `.git`, then installs the omakase base. Zero language/framework detection —
  the project's identity is never inspected (emergent from OQ-5 / OQ-6).

- **DD-12: Packs may carry SCAFFOLDING RECIPES.** A recipe is a template plus a
  self-declared target path. `pack add` can bootstrap a repo for its language via a recipe,
  and init applies recipes through a GENERIC copy-template-to-declared-path mechanism —
  never language- or platform-aware code. ~~(Depends on the pack-recipe capability, which
  does not yet exist — see Out of Scope / Dependencies.)~~
  **CORRECTION (2026-08-12, v0.10.0): the capability EXISTS — this dependency is satisfied,
  not pending.** BUNDLE-015 (Pack Scaffolding Recipes, `defined`) owns it; SPEC-054 (Recipe
  Apply And Manifest, `status: implemented`) shipped the mechanism as `pkg/recipe` plus the
  `recipe apply` CLI; SPEC-067 (CI Recipe Pack, `status: implemented`) shipped the first
  gate-workflow consumer, published and installed as `backstop-ai/ci-workflows@v0.1.0` — a real
  external pack, not a fixture. DIR-019's own 2026-08-12 correction states the consequence
  directly: "This directive's DIR-002 blocking-dependency claim … is correspondingly CLEARED."
  Init's spec is therefore NOT blocked on recipes; it consumes a shipped capability, and the
  live sharp edges in that capability (see Sharp Edges) are what its spec author must plan
  around instead.
  **MVP note on pack-declared setup steps (0.10.0, founder-ruled).** Recipes copy files; they do
  not run commands. The `step` op that would run one is RESERVED-BUT-NOT-EXECUTED (BUNDLE-019
  owns its executor) and its governance posture is an open question in BUNDLE-021. Rather than
  build across two unbuilt bundles, the founder ruled the MVP for "the packages a pack's engines
  need but do not vendor" is documentation: *"I think an MVP solution for this is simply
  including a dependency install guide in a pack readme."* That is a PACK-AUTHORING CONVENTION
  — pack authors document their own install/setup steps in their own README — not a new
  backstop mechanism, so it creates no requirement in this bundle. Recorded here so the intent
  is not lost when REQ-032's retirement is read later. Init's side of it is REQ-011 v1.1.0:
  report the uninstalled toolchain and point at the pack.

- **DD-13: HARD INVARIANT — init's thin-executor boundary.** Detection, framework
  recognition, and CI-platform knowledge live in PACKS as data, NEVER in core/CLI. Core
  dispatches; packs know. A language, framework, or platform name appearing in core CLI code
  IS the bug; `backstop/self` enforces it. This is the invariant that makes every other init
  resolution safe — omakase, recipes, and CI templates are all data supplied by packs.

- **DD-14: backstop COMPOSES with ecosystem scaffolders — it does not own project
  scaffolding.** `create-next-app`, `rails new`, `cargo new`, etc. produce the project;
  `backstop init` adds the backstop layer on top (converge-not-clobber, DD/OQ-6). Init never
  reimplements what an ecosystem scaffolder already does.

- **DD-15 (added 0.8.0): A false green is a strictly worse failure than a loud error, and
  onboarding is where both are manufactured.** DD-9 treats version skew as a DEBUGGING tax —
  the 2026-07-12 instance produced a misleading error that cost most of a day. The 2026-07-16
  instance produced something else: `artifact validate` reported GREEN across five revisions
  and a promotion to `defined` on a bundle that violated its own schema in 41 places. A
  misleading error at least stops you; a false green lets nonconformant governed state
  propagate and accrue derived work on an invalid foundation. Two distinct mechanisms in this
  bundle's evidence produce that class of failure, and both surface during onboarding, because
  onboarding is exactly when a consumer's binary, packs, and layout are least likely to agree:
  (a) **cohort skew** — the validator's embedded schemas silently predate the artifact's
  (REQ-026, REQ-027, REQ-028); (b) **silent undiscovery** — the gate walks an artifact root the
  consumer isn't using and reports a passing dimension over nothing (REQ-029, REQ-030).
  Rationale for making this a decision rather than a note: it establishes the acceptance
  posture for all five requirements — the correct behavior on "I cannot tell" is REFUSE, not
  warn and not pass. That inverts the default this codebase otherwise favors
  (loud-≠-blocking / warn-on-un-adopted-capability), and the inversion is deliberate: an
  un-adopted capability is a missing benefit, whereas an unverifiable green is an active lie
  about work that was already accepted on its strength. It also constrains init directly —
  ~~REQ-031 forbids init from scaffolding the layout (b) would silently swallow, even though
  REQ-004 calls for that layout, until REQ-029 makes it discoverable.~~ **CORRECTED 2026-08-12
  (v0.10.0):** REQ-031 is RETIRED and the constraint now lives where it belongs, in the Spec
  Seeds ORDERING — the guards seed (REQ-029) is specced and shipped BEFORE the init seed, so
  init never faces the choice REQ-031 was written to forbid. As a requirement it was formally
  contradictory (REQ-004 mandates the `.backstop/` layout; REQ-031 forbade it) and dead on
  arrival under the bundle's own declared order. If that order is ever changed, this constraint
  must be re-asserted — it is the reason the order is what it is.

## Open Questions

All seven open questions were driven to resolution by the founder in a 2026-07-13
working session (recorded in 0.6.0 below). None remain open. They are kept here — marked
RESOLVED with the decision and rationale — rather than deleted, so the reasoning survives.
The founder ruled promotion to `defined` on 2026-08-12 (see Version History 0.9.0).

- **OQ-1 (load-bearing) — RESOLVED: Canonical artifact-dir layout.** Decision: `.backstop/`
  is the artifact root for consumer repos; **backstop-core keeps the root layout as the
  framework exception** (it IS the framework — its artifacts are primary content, not
  governance overhead). Init scaffolds and validates the `.backstop/`-rooted layout for
  consumers. Rationale: one canonical consumer layout ends the live-repo divergence
  (backstop-core root vs backstop-runtime `.backstop/` vs stale test projects) and keeps the
  consumer root clean (only `backstop.yml`/`.lock` + `.backstop/` visible). This deliberately
  re-adopts the previously-retracted `.backstop/` convention, now with the framework
  exception made explicit rather than implicit.

- **OQ-2 — RESOLVED: How the init profile / capability set is chosen.** Decision: **omakase
  default** — bare `backstop init` installs the full opinionated bundle, PROMPT-FREE; you
  SUBTRACT capabilities via flags (`--only X` / `--no-X`, Cargo-features style: a default
  feature set you strip from). Rationale: an opinionated default is the fastest path to value
  and removes wrong-default guessing; subtraction is explicit and scriptable. Because it is
  prompt-free, init is headless/CI-safe by default — the interactive-vs-CI tension dissolves
  entirely.

- **OQ-3 — RESOLVED: baseline on a remoteless repo.** Decision: init **seeds a GITIGNORED
  LOCAL baseline** so solo/local-first users get the ratchet from day zero; the CI-generated
  baseline (BUNDLE-007) is the later team upgrade (tamper-resistance, no concurrent
  stomping). Also fix the self-contradictory remoteless `baseline_comparison` message.
  Rationale: local-first users should not need a remote to get the ratchet; CI baseline is an
  upgrade, not a prerequisite. Reflects back to BUNDLE-007 / DIR-003 as a new day-zero-local
  baseline mode (see Out of Scope / Dependencies).

- **OQ-4 — RESOLVED: local-pack restore from a fresh clone.** Decision: the committed
  `backstop.lock` records **only portable git-ref packs**; a GITIGNORED local provenance
  cache records local-path pack sources so `pack install` restores them **on the same
  machine**. Promoting a local pack to shareable is an explicit git-ref step. Rationale: keeps
  the committed lock portable (no machine-specific paths leak into shared history) while a
  single machine can still restore local packs; sharing is a deliberate act. The gitignored
  local-provenance lock mechanism is a pack-CLI / lock-schema change (see Out of Scope /
  Dependencies).

- **OQ-5 — RESOLVED (DISSOLVED): multi-language init.** Decision: init does **ZERO language
  detection**. Languages enter only via explicit `pack add` (which may optionally scaffold
  via a recipe); multi-language = multiple packs. Registry-era auto-detect is out of
  scope / future. Rationale: language detection was a baked-language assumption (a
  thin-executor violation); removing it dissolves the question — there is no "primary
  language" to pick. Supersedes the former DD-5; codified as DD-11 / DD-13.

- **OQ-6 — RESOLVED: re-init / idempotency.** Decision: **converge, never clobber,
  FRAMEWORK-BLIND.** Init detects only backstop-neutral facts (is there a `.git`? a
  `backstop.yml`? artifact dirs?), adds only what is missing, reports only in backstop terms,
  and NEVER interprets the repo's language/framework identity. Rationale: idempotent and
  non-destructive re-runs, and staying framework-blind keeps init inside the thin-executor
  boundary (DD-13). Codified as DD-11 / DD-14.

- **OQ-7 — RESOLVED: consumer CI-workflow scaffolding.** Decision: ~~CI is wired **by default**
  (omakase), delivered as a **CI RECIPE PACK** holding per-platform templates as DATA; init
  applies the selected variant via the generic recipe mechanism (`--ci github` default,
  `--no-ci` opts out).~~ **CORRECTION (2026-08-13, v0.10.1) — THE DEFAULT IS DROPPED; `--ci` IS
  AN EXPLICIT FLAG.** CI is delivered as a **CI RECIPE PACK** and applied through the generic
  recipe mechanism, but init wires CI ONLY when `--ci` is passed explicitly. There is
  no default platform, no inferred one, and no `--no-ci` opt-out to need — omission IS the
  opt-out. ~~Init passes the flag's value straight through to `recipe apply <pack>:<value>`; the
  accepted values are whatever recipe names the installed pack declares.~~ **FURTHER CORRECTION
  (2026-08-13, v0.10.2) — THE FLAG'S VALUE IS THE FULL PINNED REF, NOT A BARE RECIPE NAME:**
  `--ci <pack>:<recipe>@<version>`, e.g.
  `backstop init --ci backstop-ai/ci-workflows:github-actions-gate@1.0.0`, handed to
  `recipe apply` VERBATIM with core constructing no part of it. `ParseRecipeRef`
  (`pkg/recipe/resolve.go:60-88`) accepts only that shape and the semver pin is MANDATORY —
  "there is no 'latest', no default version, and no tolerance branch (CLM-049)" — so a bare
  recipe name would have produced a call that always errors, and having core supply the
  `<pack>:` half would have put a pack-name literal in core, the same DD-13 bake this
  resolution was correcting. Omitting `--ci` SKIPS
  CI wiring and init must SAY SO — naming `backstop recipe apply` and the pinned
  `<pack>:<recipe>@<version>` ref shape as the way to wire it later — as a deliberate no-op,
  not an error exit,
  consistent with how REQ-011 reports an uninstalled toolchain and REQ-035 reports a
  brownfield preserve. **Why the reversal:** the original resolution's "per-platform templates
  / selected variant" framing assumed a variant abstraction with a core-side default. The
  delivered pack has no such thing — `backstop-ai/ci-workflows@v0.1.0` declares four discrete,
  separately named recipes (`github-actions-gate`, `gitlab-ci-gate`,
  `bitbucket-pipelines-gate`, `jenkins-gate`), no variant layer and no pack-declared default —
  so `--ci github` as a default would have required core to hold BOTH the literal `github` AND
  a `github → github-actions-gate` mapping. That is a platform name in core CLI code, which
  DD-13 names as the bug itself. **Founder ruling 2026-08-12, verbatim:** *"normally option 1
  is the answer [pack-declares-its-own-default], but my intent at least for this ci pack is
  just to help people scaffold ci quickly. not be opinionated about default. so i guess it's
  an explicit flag?"* An interactive "which platform?" prompt was considered and rejected: no
  prompt mechanism of any kind exists in `cmd/backstop/` today, so it would be new,
  unjustified scope. REQ-016 v1.2.0 and REQ-017 v1.2.0 carry the corrected behavior.
  **The rest of the original resolution stands unchanged:** **No `ci` verb** — the CI job
  chains existing agnostic commands
  (`pack install` → `baseline pull` → `gate`), which also gives local/CI parity. If CI was
  ASKED FOR and the named ref cannot be resolved, the CI STEP fails LOUDLY (no
  baked fallback exists — the architecture makes silent success impossible) while the rest of
  init succeeds — and per REQ-017 v1.2.0 that loudness is `recipe apply`'s OWN resolve error
  surfaced, since the consumer named the pack in the ref and core therefore has nothing to
  detect.
  Rationale: CI-platform knowledge is data in a pack, never baked into core (DD-13);
  ~~default-on makes enforcement real fast;~~ **(2026-08-13: the default-on rationale is
  withdrawn per the correction above — an opinionated default cannot be held anywhere but in
  core, which DD-13 forbids)**; the loud-but-non-blocking failure honors
  loud-≠-blocking. ~~Depends on the recipe capability + a concrete CI recipe pack (see Out of
  Scope / Dependencies).~~ **CORRECTION (2026-08-12, v0.10.0): both dependencies are
  DELIVERED** — the recipe capability as SPEC-054 (`implemented`, BUNDLE-015/DIR-019), the CI
  recipe pack as SPEC-067 (`implemented`), live at `backstop-ai/ci-workflows@v0.1.0` with all
  four platform templates. The resolution is unchanged; only its dependency status is. One
  thing the 2026-07-13 resolution did NOT anticipate has since surfaced: on a BROWNFIELD repo
  the apply preserves an existing platform config and wires no gate while reporting success
  (ISSUE-119). REQ-035 (added 0.10.0) makes init report that outcome honestly.

### Questions closed earlier (recorded, not open)

- **Baseline storage / granularity / progressive reduction** — RESOLVED via BUNDLE-007;
  captured in DD-10. Not reopened here.
- **Binary distribution** and **dependency/toolchain installation strategy** — MOVED to
  DIR-001 (Release Workflow), the owner of "how users get the binary + system toolchain."
  Captured in DD-10. Not litigated here.

## Spec Seeds

Three non-overlapping seeds, in suggested implementation order: **trustworthy-green guards →
`backstop init` → `backstop doctor`**. Baseline, binary distribution, and toolchain
acquisition are deliberately NOT seeds here — they belong to BUNDLE-007 and DIR-001 (DD-10).

- **Trustworthy-green guards (core)** — new in 0.8.0, and FIRST in order.
  **Requirements (revised 0.10.0): REQ-021, REQ-022, REQ-026, REQ-027, REQ-028, REQ-029,
  REQ-030, REQ-034 — 8 total.** Covers: a content-derived schema cohort (REQ-026), a
  content-derived schema identity on the ARTIFACT side or its validation record (REQ-034),
  cohort assertion with refuse-to-green in `artifact validate` and `gate` (REQ-027),
  producing-binary provenance on every result (REQ-028), config-resolved artifact root shared
  by gate / validate / scaffold (REQ-029), scanned-root reporting with loud failure on an absent
  configured root plus surfacing of artifact-shaped files outside the resolved root (REQ-030),
  the build-identity stamp on `version` (REQ-021), and naming the capability gap rather than the
  downstream engine error (REQ-022). Rationale for sequencing it first: REQ-004 (`.backstop/`
  layout) is unimplementable without REQ-029 — this ordering IS the constraint retired REQ-031
  used to state — and every acceptance claim the other two seeds make is only as trustworthy as
  the validator asserting it (DD-15). Touches `cmd/backstop/root.go`, `cmd/backstop/version.go`,
  `pkg/validate/`, `pkg/gate/artifact_status.go`, `pkg/scaffold/`, and the `backstop.yml` schema
  — no init code and no doctor code.
  **Consumption boundary (0.10.0):** REQ-022 states the diagnostic OUTCOME only. Its mechanism
  — what a pack declares and how it is compared — is BUNDLE-020's (capability sets per DD-4,
  NOT version ordering), and BUNDLE-020's OQ-2/OQ-3 are still open. This seed's spec must
  consume whatever BUNDLE-020 lands and must not decide it.

- **`backstop init` command** — the critical-path spec.
  **Requirements (revised 0.10.0): REQ-001 – REQ-019, REQ-033, REQ-035 — 21 total**, of which
  REQ-012, REQ-013 and REQ-018's local-provenance half are CONSUMED from ISSUE-056 / ISSUE-055
  rather than built here. Covers: omakase base install with subtract-via-flags (DD-2 / OQ-2),
  `git init`-if-needed + converge-not-clobber re-init (DD-11 / DD-14 / OQ-6), `.backstop/`-rooted
  artifact layout scaffolding (OQ-1), profile-correct `backstop.yml` generation, canonical
  `.gitignore` emission + scaffold ≥1 source file (DD-7), verify the toolchain runs and report
  an uninstalled one as owed setup (DD-6, REQ-011 v1.1.0), local baseline seeding (OQ-3),
  explicit-flag CI wiring via recipe (no default platform; `--ci` takes the full pinned
  `<pack>:<recipe>@<version>` ref and passes it to `recipe apply` verbatim, with a reported skip
  when the flag is omitted) including the honest brownfield-preserve report (OQ-7, REQ-035),
  a spec-independent coverage floor for the pack-only profile (REQ-033), and first-gate baseline
  capture with observation framing (DD-3 / DD-4). Zero language/framework detection (DD-13).
  Specified by transcribing the greenfield hand-onboarding checklist (DD-8).
  ~~**Blocked on** the pack-recipe capability (init consumes recipes but cannot be built until
  it exists — see Out of Scope / Dependencies).~~
  **CORRECTION (2026-08-12, v0.10.0): NOT BLOCKED — the dependency is DELIVERED.** SPEC-054
  (`implemented`) shipped the recipe mechanism under BUNDLE-015 / DIR-019; SPEC-067
  (`implemented`) shipped the CI recipe pack, published and installed as
  `backstop-ai/ci-workflows@v0.1.0`; DIR-019's own 2026-08-12 correction states that DIR-002's
  blocking-dependency claim "is correspondingly CLEARED." The init spec's real prerequisite is
  now the guards seed (sequencing, above), and its real risk surface is the set of live recipe
  sharp edges listed under Sharp Edges — not the absence of the capability.

- **`backstop doctor`** — the "help me fix my setup" diagnostic init delegates to when
  something is off. **Requirements (revised 0.10.0): REQ-020, REQ-023, REQ-024, REQ-025 — 4
  total.** Covers: the command itself with one check per ranked sharp edge (REQ-020); the
  toolchain actually executes, re-runnable outside init (REQ-023, DD-6); runtime/toolchain
  version vs the stack policy the installed packs declare — the unenforced Node-LTS observation
  (REQ-024); and artifact-layout validation against the canonical `.backstop/` root once OQ-1's
  resolution lands via REQ-029 (REQ-025). Each check is the diagnosis of a ranked sharp edge
  from the write-ups (DD-8 corollary). Doctor SURFACES the version-skew signals but no longer
  owns them.
  **CORRECTION (2026-08-20, v0.11.0) — this seed is DELIVERED 3-of-4, not 4-of-4.** SPEC-070
  (`implemented`) covers REQ-020, REQ-023 and REQ-025. **REQ-024 is CARVED OUT** and carries no
  requirement and no mandated claim in that spec: the "stack policy the installed packs declare"
  named above is a pack-manifest surface that does not exist at HEAD, and inventing it would be
  this bundle designing BUNDLE-004's surface — the same rule that produced REQ-005 v1.1.0's
  accepted residue. Ruled 2026-08-13 during spec authoring; `issues/ISSUE-121-pack-manifest-
  missing-stack-policy-surface.issue.md` owns the gap (open) and is being homed under BUNDLE-004.
  SPEC-070 guards the carve-out with an absence claim asserting its check registry reads no
  stack-policy surface, so no implementer can close REQ-024 from the wrong artifact silently. The
  seed PARTITION is correct and unchanged — REQ-024 belongs here; it simply has no owner yet.
  ~~Covers: binary version stamp present (commit+date, not bare `dev`) and binary-vs-pack
  capability skew (DD-9).~~ **CORRECTION (2026-08-12, v0.10.0), two defects in one clause:**
  (1) the phrase "not bare `dev`" is the withdrawn wording — REQ-021 was corrected on 2026-08-11
  (v0.8.1, founder-ruled) because `cmd/backstop/version.go` reports `dev` for every non-release
  build ON PURPOSE, as anti-spoofing; what must be identifiable is the BUILD (commit + build
  date), not the release. This seed still carried the withdrawn phrasing and now matches
  REQ-021 v1.1.0. (2) REQ-021 and REQ-022 have MOVED to the guards seed — they are core changes
  to `version.go` and to the `pack add`/`gate` failure path, not checks doctor runs.

## Notes / Ideas

### Out of Scope / Dependencies

The 0.6.0 OQ resolutions lean on capabilities that this bundle consumes but does NOT design
here. These are recorded so spec authoring does not accidentally absorb them:

- ~~**Pack-recipe capability (BLOCKING DEPENDENCY)**~~ **Pack-recipe capability — DELIVERED
  2026-08-12; consumed here, owned elsewhere.** How packs declare and ship scaffolding + CI
  templates (template + self-declared target path; generic copy-to-path apply). ~~Init consumes
  recipes but **cannot be built until this exists**. Likely its own bundle.~~
  **CORRECTION (2026-08-12, v0.10.0):** it exists, and it does have its own bundle — **it IS
  BUNDLE-015** (Pack Scaffolding Recipes, `defined`), directed by **DIR-019**, delivered by
  **SPEC-054** (Recipe Apply And Manifest, `status: implemented`) as `pkg/recipe` + the
  `recipe apply` CLI. Nothing here is blocked on its absence any more. It stays on this list
  because it is still not DESIGNED here — the distinction is ownership, not existence.
- ~~**CI recipe pack**~~ **CI recipe pack — DELIVERED 2026-08-12.** A concrete pack deliverable
  holding github / gitlab / bitbucket / jenkins templates as data. ~~Depends on the recipe
  capability.~~ **CORRECTION (2026-08-12, v0.10.0):** delivered by **SPEC-067** (CI Recipe Pack,
  `status: implemented`) against BUNDLE-015 REQ-018; published and live at
  `backstop-ai/ci-workflows@v0.1.0` (a real GitHub tag) and installed into backstop-core itself
  from that source — a real external consumer, not a fixture. Per this project's
  packs-live-outside-core invariant it lives in its OWN pack repo, so core's empty `recipes/`
  directories are NOT evidence it is unbuilt (DIR-019 says this in as many words). Its known
  residual gap is brownfield adoption — ISSUE-119, which REQ-035 now answers on init's side.
- **Gitignored local-provenance lock mechanism (OQ-4)** — a pack-CLI / lock-schema change so
  local-path pack sources restore on the same machine while the committed lock stays portable.
  **OWNER (recorded 2026-08-12, v0.10.0): ISSUE-055** (open, technical-debt), which cites this
  bundle's OQ-4 as its source of direction and quotes the real `LockEntry.LocalPath` state. This
  is the half of REQ-018 marked CONSUMED.
- **Local-first baseline seeding (OQ-3)** — a new day-zero-local baseline mode; reflects back
  to BUNDLE-007 / DIR-003 (the baseline subsystem). **OWNER (recorded 2026-08-12, v0.10.0):
  ISSUE-056** (open, technical-debt), which cites this bundle's OQ-3 as its source of direction
  and whose Solution items ARE REQ-012 and REQ-013 — both now marked CONSUMED rather than
  restated as this bundle's build obligation.
- **Spec-independent coverage floor knob (`enforcement.coverage.min_pct` or equivalent) —
  DANGLING: NO OWNER (flagged 2026-08-12, v0.10.0).** REQ-033 requires init to WIRE a
  spec-independent coverage floor for the pack-only profile, and this bundle correctly excludes
  DESIGNING the knob. But unlike every other entry on this list, no artifact anywhere owns
  building it: verified 2026-08-12 that `artifacts/backstop-yml/v1/schema.json` contains no such
  key (the only `coverage` occurrences are the traceability-dimension enum values and the
  per-dimension `enforcement.policy` description), and no bundle, directive, spec, or issue
  claims it. REQ-033 is therefore UNSATISFIABLE until some artifact adopts it. This is flagged,
  not assigned — picking its home is a founder/PM call, not one this pass takes. Whoever specs
  the init seed must confirm an owner exists before scoping REQ-033, or REQ-033 will silently
  become "init invents an enforcement-policy surface", which is exactly what this bundle says it
  does not do.
- **Pack registry + pack-declared `detect:` field** — registry-era auto-detection; deferred,
  relates to pack distribution (BUNDLE-001 / BUNDLE-002). Explicitly NOT how languages enter
  in this bundle (OQ-5 dissolved detection).
- **Bundle→spec promotion gate check** — an orthogonal workflow-integrity hole: a spec whose
  parent bundle is not promoted should be a violation but currently is not enforced (this
  bundle's own legacy SPEC-020..029 were auto-generated against an unpromoted parent). Belongs
  to gate/workflow hardening, not init.

### Sharp Edges (added 2026-08-12, v0.10.0)

Init is the recipe mechanism's PRIMARY CONSUMER, and that mechanism shipped with known,
already-filed gaps. They are not hypothetical risks — each is an open issue with a live
reproduction — and the init spec author will hit them, so they are named here rather than
rediscovered mid-spec. None of them is this bundle's to FIX; all of them are this bundle's to
PLAN AROUND.

- **ISSUE-119 (open) — brownfield CI adoption reports success and wires nothing.** The applier's
  `create` op family is never-clobber by design: an existing target file is preserved and
  reported as `preserved`, the apply succeeds, and an adoption record is written — with no
  backstop gate in the consumer's CI at all. Three of the four platforms
  (`.gitlab-ci.yml`, `bitbucket-pipelines.yml`, `Jenkinsfile`) have exactly ONE conventional
  entry point each, so every existing project adopting backstop hits this; only GitHub's
  per-workflow-file layout escapes. **Init impact:** greenfield init is unaffected, but init on
  an existing repo is the common adoption path. REQ-035 requires init to report the gap; the
  merge/insert-op fix itself is ISSUE-119's.
- **ISSUE-081 Gap 3 (open) — `insert` op placement semantics are unpinned.** `applyInsert`
  (`pkg/recipe/apply.go:639-683`) splices the snippet inline immediately after the anchor text,
  producing e.g. `"registrations": [    "live-entry",` on one line — live-reproduced in the
  2026-07-26 recipe-scenario sweep. The contract leaves newline handling for the pack author to
  discover by trial. **Init impact:** any init step that relies on an `insert` op to extend an
  existing file inherits unpredictable output; init should not depend on `insert` placement
  until the semantics are pinned. (ISSUE-081 also carries two sibling gaps on the `merge` op's
  `fragment:` form being ambiguous across applier, spec, and fixtures.)
- **ISSUE-110 (open) — no escape syntax for foreign `{{ }}` templates.** `pkg/recipe/
  substitute.go` reads EVERY `{{ ... }}` span in a payload as a param name and hard-fails on
  any name no param declares — emitting nothing at all, not partial output. A payload that must
  legitimately carry a FOREIGN template (a GitHub Actions expression, a Helm/Jinja/Handlebars
  fragment) cannot, and the failure fires even inside an explanatory comment. **Init impact:**
  init's CI wiring applies templates for platforms whose own syntax uses `{{ }}`; a
  recipe payload cannot currently quote those without tripping the substituter.

Two further live-risk items are recorded elsewhere in this bundle and belong on the same list
for a spec author's purposes: the **dangling coverage-floor knob** REQ-033 depends on (no owner
anywhere — see Out of Scope / Dependencies), and **BUNDLE-020's open OQ-2/OQ-3**, which REQ-022
consumes and must not pre-empt.

### Observations and evidence

- The emotional framing of first-run output (DD-3) is a product decision, not a technical
  one. The exact wording of the baseline summary determines whether a tech lead feels
  welcomed or attacked; it deserves design attention and possibly user-testing outside the
  project.
- **Cascading coverage error masks the real failure (pack-side, first-run polish).** When
  a test FAILS, `coverage_threshold` fails with a misleading secondary error — `declared
  stdout_artifact ".backstop/ts-coverage/backstop-summary.json" not produced` — because
  vitest emits no coverage summary on a failed run. The real failure is caught loudly in
  `pack_engines`, but the coverage step points at the pack producer rather than saying
  "tests failed, so coverage was not computed." Same misdiagnosis family as DD-9. A
  pack-side messaging concern, not strictly init, but it degrades the clean first-run
  experience init aims to deliver. (2026-07-12 verification pass.)
- **Node LTS is unenforced.** The dogfood machine had only non-LTS Node (23 + 25) despite a
  documented "Node LTS" stack decision, and nothing warned. Feeds the `doctor` seed as a
  node-version-vs-stack-policy check.
- **Positive — enforcement is real, not vacuous green.** On the live bclabs-portal repo, an
  injected `eval()` tripped the standards no-eval rule, a type error tripped tsc TS2322, a
  hardcoded AWS key tripped no-hardcoded-credentials, and a failing test tripped vitest —
  all drove the gate RED; a clean repo goes GREEN. First validation of the "real-project
  e2e / RED-when-it-should" bar in an external consumer.
- **Positive — waiver-hint renders in the wild.** The waiver-hint wiring
  (`↳ to waive: @waiver:...`) renders correctly in a live external consumer; first
  confirmation of that requirement outside the framework repo.
- **Observation — `ledger_integrity: skipped (ledger not implemented)`.** Core already
  reserves the concept a consumer is about to implement.
- **`artifact new` number reservation — out of scope here, but root-caused (2026-08-11).**
  The write-up reports that `artifact new spec` cannot gap-fill a number, that git tags are
  the undocumented reservation mechanism, and that an orphan tag got reused while a stray tag
  one higher was created. Reading `pkg/scaffold/idresolver.go` explains the observed symptom
  exactly: `GitTagResolver.Resolve` takes max+1 over `backstop/<type>/*` tags, CREATES the
  annotated tag, and only THEN pushes it — and a push failure that is not a `TagConflictError`
  (a remoteless repo, for instance) returns `FallbackError`, at which point `ResolveID` falls
  through to `LocalScanResolver`, which numbers from FILES on disk. So the run leaves behind
  the locally-created tag at max+1 while returning a lower, file-derived number. That is the
  "produced SPEC-007, stray-tagged 008" pair. It is worth noting this fires hardest on exactly
  the repo shape init targets — freshly created and remoteless. Still a `pkg/scaffold` defect
  rather than an init requirement, so it is recorded here for a separate issue rather than
  requirement-ized.
- **Semgrep rule-id readability — out of scope here.** The pack-only write-up logs it as
  cosmetic, core-side, and non-blocking; it degrades waiver tokens and gate output generally,
  not onboarding specifically. Recorded so it is not lost, not adopted as init scope.
- **Predating artifact — SPEC-005's cohort claim is SUPERSEDED by REQ-026 (recorded
  2026-08-12, v0.10.0).** `specs/SPEC-005-cli-foundation.spec.md` (`status: draft`) states the
  cohort is "derived from the set of embedded schemas" — i.e. from the SET, which is what
  `computeCohortID` implements today (paths only: `11-schemas[adr/v1,...,spec/v1]`). REQ-026
  INVERTS that: the cohort must be derived from schema CONTENT, precisely because a set-derived
  cohort could not see BUNDLE-014's in-place `bundle/v2` revision. Naming it here per the
  align-predating-artifacts convention: SPEC-005 is not wrong about what was built, it is a
  claim the guards seed will supersede, and whoever specs REQ-026 must update SPEC-005's claim
  openly rather than leaving two artifacts asserting different cohort derivations. SPEC-005 is
  still `draft`, so this is a cheap correction if taken with the guards spec.
- **Out of lane, flagged for its own owner — DIR-002's directive text still describes baked
  language detection (2026-08-12, v0.10.0).** `directives/DIR-002-backstop-init.directive.md`
  is the directive THIS bundle sources, and its own prose predates OQ-5's dissolution of
  language detection (this bundle's DD-13 hard invariant). It needs a dated correction of its
  own, authored through the directive track — deliberately NOT edited from this bundle, which
  owns neither that file nor the directive vocabulary.
- **Go-forward.** Derive the `backstop init` and `backstop doctor` specs directly from the
  two per-profile hand-onboarding checklists (DD-8), NOT by re-inventing the flow — after
  OQ-1 and OQ-2 are resolved and the bundle is promoted. Spec authoring is a
  transcription-and-hardening exercise against a validated flow.

## Version History

- 0.1.0 (2026-04-09): Initial bundle. Problem (no onboarding path), target experience,
  5 design decisions (init command, auto-install deps, baseline as observation, diff-based
  default scope, language detection), 8 open questions, 5 spec seeds. Maturity: exploring.
  A thought exercise analyzing onboarding from a tech lead's perspective.
- 0.2.0 (2026-04-19): Added DD-6..9 mandating `.backstop/` as the consuming-repo artifact
  root. Resolved baseline OQs via BUNDLE-007.
- 0.3.0 (2026-07-12): Evidence intake from the first real greenfield onboarding
  (bclabs-portal: empty repo → gate PASS). Added the two-profile decision, version-skew
  diagnosability, verify-the-toolchain-runs, canonical `.gitignore` + scaffold source file;
  added OQs for profile selection, artifact layout, remoteless baseline, local-pack restore.
- 0.4.0 (2026-07-12): Provenance reframe — established the two hand-onboarding write-ups as
  the empirical requirements corpus; added the "specify init by transcription" decision.
- 0.5.0 (2026-07-13): **From-scratch rewrite to current discipline.** Purged the legacy
  auto-generated SPEC-020..029 and their 9 plans (never committed — untracked cruft from a
  2026-05-30 auto-dispatch experiment run against this then-unpromoted bundle) and reset to
  a clean `exploring` foundation from which init will be properly re-speced in order
  (resolve OQs → promote → spec). Kept the dogfood gold: two-profile model (now the headline
  DD-2), version-skew diagnosability (DD-9), verify-the-toolchain-runs (DD-6), canonical
  `.gitignore` + scaffold source file (DD-7), init-by-transcription (DD-8), and the
  requirements-corpus framing. **Retracted the `.backstop/`-root mandate** (old DD-6/7/8):
  the dogfood used the ROOT convention and passed, and live repos diverge, so the layout is
  reopened as OQ-1 (the load-bearing open question) — root vs `.backstop/` vs detect-both.
  **Moved** binary distribution and dependency/toolchain installation OUT to DIR-001 (their
  owner); dropped the corresponding OQs and the binary/dependency spec seeds. **Recorded**
  the baseline OQs as resolved-via-BUNDLE-007 rather than open. Distilled the DD set to the
  ten dogfood-grounded decisions and the two spec seeds (init, doctor). Maturity unchanged:
  exploring — no self-promotion, no pre-resolved OQs.
- 0.6.0 (2026-07-13): **All 7 OQs resolved** in a founder-driven working session; each
  converted open → RESOLVED with decision + rationale (none remain open). Decisions: OQ-1
  `.backstop/` root for consumers, root for backstop-core (framework exception); OQ-2 omakase
  prompt-free default with subtract-via-flags (headless-safe, tension dissolved); OQ-3 seed a
  gitignored local baseline day-zero (CI baseline is the team upgrade); OQ-4 committed lock
  holds only portable git-refs + a gitignored local-provenance cache for local packs; OQ-5
  DISSOLVED — zero language detection, languages enter via `pack add`; OQ-6 converge /
  never-clobber / framework-blind re-init; OQ-7 CI wired by default via a CI recipe pack, no
  `ci` verb, loud failure if the pack is absent. Added emergent DD-11 (`git init` + backstop
  base, zero detection), DD-12 (pack scaffolding recipes via a generic copy-to-path apply),
  DD-13 (HARD INVARIANT — detection / framework / CI-platform knowledge lives in packs as
  data, never in core; `backstop/self` enforces), DD-14 (composes with ecosystem scaffolders,
  does not own project scaffolding). RETRACTED DD-5 (language detection) as superseded.
  Revised DD-1, the solution approach, and the init spec seed to drop language detection and
  reflect omakase + recipes + local baseline + default CI. Added an "Out of Scope /
  Dependencies" note recording the pack-recipe capability (a BLOCKING dependency init can't be
  built without), the CI recipe pack, the local-provenance lock change, local-first baseline
  seeding, registry-era detection, and the orthogonal bundle→spec promotion gate hole.
  Maturity unchanged: exploring — promotion is founder-triggered separately.
- 0.7.0 (2026-08-10): **Draft Requirements authored** — 25 formal requirements
  (REQ-001..025) added to frontmatter plus a matching `## Draft Requirements` section,
  each derived from an already-resolved OQ, an existing DD, or a spec seed; no new scope
  introduced. Partitioned non-overlappingly across the two seeds: REQ-001..019 to
  `backstop init` (command shape and omakase, two-profile config, `.backstop/` layout,
  canonical `.gitignore`, thin-executor boundary and generic recipe apply, `git init`
  -if-needed and converge-not-clobber, toolchain-runs verification, local baseline +
  observation framing + diff-scoped default, default CI via recipe pack with loud absent-pack
  failure, portable-lock installs, and the transcribed-sequence acceptance bar), REQ-020..025
  to `backstop doctor` (the command itself, version stamp, binary-vs-pack capability skew,
  standalone toolchain check, pack-declared stack-policy version check, layout validation).
  Recorded what is deliberately NOT a requirement here (baseline mechanics, binary
  distribution, the recipe capability and CI recipe pack, the local-provenance lock change,
  the pack-side cascading coverage message, registry-era detection). **One tension resolved
  and flagged:** DD-7's literal `.gitignore` list contains TypeScript-specific paths, which
  DD-13 forbids in core, so REQ-005 states the backstop-owned entries literally and defers
  the rest to what installed packs declare as generated output. Maturity unchanged:
  exploring — promotion remains founder-triggered.

- 0.8.0 (2026-08-11): **Grounding pass — requirements reconciled against the two
  hand-onboarding source documents themselves**, rather than against this bundle's summary of
  them. Fifteen concrete findings checked: ten already covered by REQ-001..025, two ruled out
  of scope with reasons recorded (semgrep rule-id readability — cosmetic core output
  ergonomics; `artifact new` number reservation — a `pkg/scaffold` defect, root-caused under
  Observations), one retired by its own successor evidence (the `pnpm-workspace.yaml`
  `onlyBuiltDependencies` recommendation, superseded when pnpm 11.11 ignored the key and the
  failure proved cosmetic — which is the argument FOR REQ-011). **Eight new requirements
  (REQ-026..033)**, each with its code state re-verified against HEAD on 2026-08-11 rather
  than trusted from July notes. The two serious gaps: (1) **schema-cohort false green** —
  REQ-026 content-derived cohort (`computeCohortID` is still path-derived, so BUNDLE-014's
  in-place `bundle/v2` revision was invisible), REQ-027 refuse-to-green when cohort coverage
  is unprovable (the only schema cross-check today is prefix-vs-artifact-type), REQ-028
  producing-binary provenance on every result (gate stamps `GitSHA`/`GeneratedAt` but no
  binary identity); (2) **artifact-root silent undiscovery** — REQ-029 config-resolved
  artifact root through one resolver (all three hardcodings confirmed present; no
  `artifact_root` key exists), REQ-030 scanned-root reporting with loud failure on an absent
  configured root while existing-but-empty stays a pass. Plus REQ-031 (init must not scaffold
  a layout discovery cannot see — REQ-004 is GATED on REQ-029), REQ-032 (install
  pack-declared project devDeps from recipe data before REQ-011 runs — a hole BETWEEN REQ-009
  and REQ-011), REQ-033 (pack-only coverage floor, since REQ-003 turns `coverage_threshold`
  off). Added **DD-15** generalizing both false-green mechanisms and setting the acceptance
  posture — on "I cannot tell", REFUSE rather than warn — a deliberate inversion of the
  loud-≠-blocking default, justified because an unverifiable green is an active lie about
  already-accepted work. Added a **third spec seed**, "Trustworthy-green guards (core)"
  (REQ-026..030), sequenced FIRST; the seeds now partition as init REQ-001..019 + 031..033,
  doctor REQ-020..025, guards REQ-026..030. Maturity unchanged: exploring — this pass changed
  requirements content only; promotion remains founder-triggered, and the prior structural
  assessment is untouched.
- 0.8.1 (2026-08-11): **Single-requirement correction — REQ-021 → v1.1.0, founder-ruled.** The
  clause "must never report a bare `dev`" was found to contradict shipped, deliberate behavior:
  `cmd/backstop/version.go` reports `dev` for ANY non-release build on purpose (rejecting
  `(devel)`, `+` build metadata, and Go pseudo-versions) so a local build cannot masquerade as an
  official release — verified against the code and against `./bin/backstop version`, which prints
  `backstop version dev`. Founder ruling: "it can be dev for a local build if that's standard
  practice" — the CODE is right and the requirement text was wrong. REQ-021 keeps its
  build-commit/build-date clause (build IDENTITY is what separates a stale dev build from a fresh
  one) and withdraws the prohibition; the prior text is preserved under `versions:` with a dated
  `correction`. Matching dated corrections applied to DD-9 mechanic (a) and the Current Thinking
  version-skew bullet, which both repeated the withdrawn phrasing. **Nothing is lost:** the
  2026-07-12 dogfood concern (a weeks-old binary reporting only `dev`, costing significant
  debugging time) is carried by REQ-022 — which names the stale binary as the cause instead of
  surfacing the downstream engine error, the exact observed failure mode — and by REQ-028, which
  stamps producing-binary identity onto every result. No new mechanism was invented beyond the
  one-line ruling. No requirement added or retired; no OQ resolved; maturity unchanged: exploring.
- 0.9.0 (2026-08-12): **Promoted to `defined` — founder-ruled.** No content authored in this
  pass: every structural requirement for `defined` was already in place from the prior passes —
  33 requirements (REQ-001..033) across two authoring passes (0.7.0's 25 plus 0.8.0's 8
  dogfood-grounded additions, with 0.8.1's REQ-021 correction), all 7 original open questions
  resolved since 2026-07-13, and populated Draft Requirements / Draft Design Decisions (DD-1..15) /
  Spec Seeds / Version History sections plus `solution.approach`. Promotion-readiness was verified
  independently before the ruling by flipping maturity on a throwaway copy and running
  `artifact validate` clean. This entry records the maturity change, the `bundle.updated` bump, and
  a one-line correction to the Open Questions preamble, which still asserted maturity would stay
  `exploring`. Unblocks `/spec` against the three seeds, in order: trustworthy-green guards →
  `backstop init` → `backstop doctor`. Context: DIR-002 sits at BACKLOG position 1, and its
  blocking dependency (BUNDLE-015 REQ-018, the CI recipe pack) was delivered via SPEC-067.

- 0.10.0 (2026-08-12): **Review-correction pass — five blocker fixes, one founder ruling, eight
  concern fixes, one structural move. Maturity UNCHANGED (`defined`); this pass corrects content,
  it does not regress readiness.** The bundle was reviewed FAIL with the reviewer explicit that
  the defects are load-bearing corrections and dependency reconciliation, not a redesign.

  **BLOCKER 1 — REQ-027 could not detect its own incident. Added REQ-034.** REQ-027 asserts the
  binary's cohort against "each artifact's declared `schema_version`", but that value is
  revision-FREE: `artifacts/bundle/v2/schema.json` pins `"schema_version": {"const": "bundle/v2"}`,
  and the incident REQ-027 is named for was an IN-PLACE revision of bundle/v2 — byte-identical
  declared value on both sides, so a stale validator reports green either way. REQ-026 makes the
  BINARY's cohort content-derived; nothing made the ARTIFACT's schema identity revision-bearing.
  REQ-034 (guards seed) requires a content-derived schema identity — a digest of the schema files
  actually used — carried by the artifact OR by the validation record produced when it is checked.
  Stated as a new requirement rather than forced onto REQ-027, which keeps its own scope.

  **BLOCKER 2 — REQ-022 pre-decided BUNDLE-020's open questions. Restated → v1.1.0.** v1.0.0
  required naming "the binary as older than the pack requires", a VERSION-ORDERING comparison —
  but BUNDLE-020 (`exploring`) founder-resolved in DD-4 that compatibility is a CAPABILITY-SET
  comparison, deliberately NOT version ordering, and its OQ-2 (where enforced) and OQ-3 (failure
  posture) are still OPEN. It also self-conflicted with REQ-021: a `dev` binary has no orderable
  version. REQ-022 v1.1.0 states the DIAGNOSTIC OUTCOME only — name the capability gap and the
  pack requiring it, instead of surfacing the downstream engine error — and defers the mechanism
  wholly to BUNDLE-020. DD-9 mechanic (b) and the guards seed carry matching dated corrections.
  BUNDLE-020's OQs are NOT resolved here.

  **BLOCKER 3 — stale "blocking dependency" claims, contradicted by delivery.** DD-12, the init
  spec seed, OQ-7, the "Deliberately NOT requirements" list, and two Out of Scope entries all
  asserted the pack-recipe capability "does not yet exist" / init "cannot be built until it
  exists" / "likely its own bundle". ALL FALSE as of 2026-08-12: SPEC-054 (`implemented`) shipped
  the mechanism, SPEC-067 (`implemented`) shipped the CI recipe pack — published and installed as
  `backstop-ai/ci-workflows@v0.1.0` — and DIR-019's own 2026-08-12 correction states DIR-002's
  blocking-dependency claim "is correspondingly CLEARED". Every occurrence corrected in place via
  the dated-correction convention (old text struck, correction appended); "likely its own bundle"
  now records that it IS BUNDLE-015. The capability remains out of scope here — on ownership
  grounds, no longer absence.

  **BLOCKER 4 — DD-2 and DD-8 step 2 instructed init to write a dead, language-baking config
  key.** Both described the greenfield `backstop.yml` as `project:` + `language:`. `language:` is
  RETIRED (`pkg/config/config.go:21-36`, SPEC-046: "absorbed by LegacyKeys below and ignored").
  REQ-003 already omitted it correctly, but REQ-019 makes DD-8's sequence the NORMATIVE
  transcription source for init's happy path, so a spec author would have copied a dead key that
  bakes a language into the consumer's first file — contradicting REQ-003/008/010 and the
  zero-baked-language invariant. Struck from both, dated corrections applied.

  **BLOCKER 5 — REQ-005 deferred to a pack-manifest field that does not exist. → v1.1.0.**
  "Every path the installed packs declare as generated output" names nothing real; `pkg/pack/
  manifest.go` has no generated-output declaration, and the nearest real surface is the
  per-engine `stdout_artifact` (manifest.go:97). REQ-005 v1.1.0 names `stdout_artifact` as the
  source it defers to AND states the ACCEPTED RESIDUE — `stdout_artifact` is what an engine
  writes for the gate to read, not an inventory of everything a toolchain leaves behind, so
  incidentally-generated paths stay the consumer's to ignore and that gap is accepted for v1
  rather than closed by inventing a manifest field (which would be BUNDLE-004's surface, not
  this bundle's). DD-7's literal canonical list — which baked `node_modules/`, `coverage/`,
  `.backstop/ts-coverage/`, `.backstop/ts-test-results.json` into core reasoning, a DD-13
  violation — is demoted to an EXAMPLE of what a pack's `stdout_artifact` declarations might
  supply.

  **FOUNDER RULING — REQ-032 RETIRED (init auto-installs pack-declared devDependencies).** Its
  mechanism required pack-command execution that does not exist and is not this bundle's to
  build: BUNDLE-019 RESERVES the `step` op without executing it ("BUNDLE-019 owns its executor"),
  and BUNDLE-021 (`exploring`) holds the unresolved governance posture for pack-declared command
  execution. Rather than stand up new execution infrastructure across two unbuilt bundles, the
  founder ruled it out of scope, verbatim: *"I think an MVP solution for this is simply including
  a dependency install guide in a pack readme."* The MVP path is recorded as a PACK-AUTHORING
  CONVENTION under DD-12 — pack authors document their own install/setup steps in their own
  README — which creates no requirement in this bundle but preserves the intent. REQ-011 silently
  ASSUMED REQ-032 (which ordered itself "before the REQ-011 toolchain-execution check runs") and
  is corrected to v1.1.0: the check still runs and still refuses to infer health, but an
  uninstalled toolchain is reported as owed SETUP — naming the pack, pointing at its documented
  steps — never silently passed and never repaired by init.

  **CONCERN 1 — shipped false-success path on brownfield CI. Added REQ-035.** Three of four CI
  platforms (`.gitlab-ci.yml`, `bitbucket-pipelines.yml`, `Jenkinsfile`) have one conventional
  entry point each, so brownfield adoption hits `create`'s never-clobber rule: apply reports
  `preserved`, init succeeds, an adoption record is written, and NO gate is wired (ISSUE-119,
  open). REQ-017 covers only "no CI recipe pack installed". By this bundle's own DD-15 posture —
  a false green is a broken promise, not un-adopted capability — REQ-035 requires init to name
  the preserved file, state that no gate was wired, and give the next action.

  **CONCERN 2 — REQ-030 did not close the backstop-runtime case it cites. → v1.1.0.** Verified
  against `pkg/gate/artifact_status.go`: a missing type-directory is not an error (each walker
  returns nil on `os.IsNotExist`, lines 382/409/437) and an unconfigured root defaults to
  `projectRoot`, which always exists — so "fail loudly on an ABSENT CONFIGURED root" never fires
  for artifacts sitting under an unadopted `.backstop/`. REQ-030 v1.1.0 adds the clause that
  actually closes it: artifact-shaped files found OUTSIDE the resolved root must be SURFACED as
  undiscovered-and-ungated, naming path and excluding root. Discovery reports; it does not widen.

  **CONCERN 3 — REQ-031 RETIRED as formally contradictory and dead on arrival.** REQ-004 mandates
  the `.backstop/` layout; REQ-031 forbade it "until REQ-029 ships" — but the declared Spec Seeds
  order puts the guards seed (which owns REQ-029) BEFORE init, so REQ-031's blocking condition is
  already false by the time init's spec is written. The sequencing INTENT is preserved where it
  belongs, in the seed-ordering rationale, with an explicit note that re-ordering the seeds means
  re-asserting the constraint. DD-15's closing sentence, which cited REQ-031, corrected to match.

  **CONCERN 4 — duplicate ownership with open issues resolved in the ISSUES' favor.** ISSUE-056
  (open) cites this bundle's OQ-3 as its source and its Solution items ARE REQ-012/REQ-013;
  ISSUE-055 (open) is the same shape against OQ-4 / REQ-018. REQ-013 was additionally a direct
  self-contradiction — pure `pkg/gate` message machinery sitting in the init seed while Out of
  Scope separately excluded baseline machinery. REQ-012, REQ-013 and REQ-018's local-provenance
  half are now marked **CONSUMED, NOT BUILT HERE** with the owning issue named in the requirement
  text; REQ-018's genuinely-init half (portable git-ref installs so the committed lock carries no
  machine-specific paths) is retained as this bundle's own. Both issues are added to Out of Scope
  as named OWNERS and to References.

  **CONCERN 5 — the 2026-08-11 REQ-021 correction was not applied everywhere.** The doctor spec
  seed still carried the withdrawn "binary version stamp present (commit+date, not bare `dev`)"
  phrasing. Corrected to match REQ-021 v1.1.0: bare `dev` is the CORRECT version string for a
  non-release build (deliberate anti-spoofing in `cmd/backstop/version.go`); what must be
  identifiable is the BUILD, via commit + build date.

  **CONCERN 6 — REQ-033's sole dependency has no owner anywhere.** Verified 2026-08-12 that no
  `enforcement.coverage.min_pct`-style key exists in `artifacts/backstop-yml/v1/schema.json`, and
  no bundle, directive, spec, or issue claims building it. The bundle correctly excludes DESIGNING
  the knob but named no owner — so REQ-033 is unsatisfiable until one exists. Flagged explicitly
  under Out of Scope as **DANGLING: NO OWNER**, with a warning that the init spec author must
  confirm an owner before scoping REQ-033. No owner invented here; that is a founder/PM call.

  **CONCERN 7 — added a Sharp Edges section.** Init is the recipe mechanism's primary consumer and
  that mechanism shipped with three open, live-reproduced gaps: ISSUE-119 (brownfield CI false
  success), ISSUE-081 Gap 3 (`insert` op splices inline — placement semantics unpinned,
  `pkg/recipe/apply.go:639-683`), ISSUE-110 (no escape syntax for foreign `{{ }}`, hard-fails even
  inside comments). Each is named with its init impact; none is this bundle's to fix, all are its
  to plan around. The dangling coverage knob and BUNDLE-020's open OQs are cross-listed there.

  **CONCERN 8 — References reconciled.** Added BUNDLE-015, BUNDLE-019, BUNDLE-020, BUNDLE-021,
  DIR-019, SPEC-054, SPEC-067, ISSUE-055, ISSUE-056, ISSUE-119, ISSUE-081, ISSUE-110, SPEC-005 —
  none appeared before, despite several owning dependencies this bundle relies on.

  **STRUCTURAL — REQ-021 and REQ-022 MOVED from the doctor seed to the guards seed.** Neither is a
  check doctor runs: REQ-021 changes what `cmd/backstop/version.go` stamps, REQ-022 changes the
  `pack add`/`gate` failure path. Both meet the guards seed's own carve-out rule (core changes init
  and doctor both depend on but neither owns) better than the doctor seed's charter. Doctor is left
  as REQ-020/023/024/025 — a clean, independently-specable diagnostics command whose every
  requirement is a check it performs.

  **NITS.** SPEC-005 (`draft`) is named as a PREDATING artifact whose "cohort derived from the set
  of embedded schemas" claim REQ-026 supersedes, per the align-predating-artifacts convention.
  "Likely its own bundle" now records that it IS BUNDLE-015, delivered. DIR-002's own directive
  text still describes baked language detection — flagged under Observations as needing its own
  dated correction through the directive track, and deliberately NOT edited from here.

  **Net requirement change: 33 → 33.** Retired REQ-031, REQ-032; added REQ-034, REQ-035. IDs are
  never renumbered, so REQ-031/032 are a permanent, documented gap. Amended to new versions with
  dated corrections: REQ-005, REQ-011, REQ-012, REQ-013, REQ-018, REQ-022, REQ-030 (all → v1.1.0).
  Seed partition is now guards REQ-021/022/026–030/034 (8), init REQ-001–019 + 033 + 035 (21),
  doctor REQ-020/023/024/025 (4). Implementation order unchanged: guards → init → doctor.
- 0.10.1 (2026-08-13): **Confirming-review follow-up — one blocker (a baked platform literal),
  one founder ruling, three nits.** Narrow pass; nothing else in the bundle was touched and
  maturity is unchanged at `defined`.

  **BLOCKER — REQ-016 baked a platform literal into core's default flag value.** REQ-016 v1.0.0
  mandated `--ci github` as the DEFAULT when the consumer names no platform, framed around "the
  selected variant" of a pack holding "per-platform templates". This bundle's own DD-13 says a
  platform name appearing in core CLI code IS the bug, and the delivered pack makes the
  contradiction concrete: `backstop-ai/ci-workflows@v0.1.0` (verified in
  `.backstop/packs/backstop-ai/ci-workflows/pack.yml`) declares FOUR discrete, separately named
  recipes — `github-actions-gate`, `gitlab-ci-gate`, `bitbucket-pipelines-gate`, `jenkins-gate`
  — with no variant abstraction and no pack-declared default. `--ci github` as a default
  therefore required core to hold BOTH the literal `github` AND a `github → github-actions-gate`
  mapping: a live bake, not a theoretical one. Because the default originated in founder-resolved
  OQ-7, this needed a founder ruling rather than an author call.

  **FOUNDER RULING (2026-08-12), verbatim:** *"normally option 1 is the answer
  [pack-declares-its-own-default], but my intent at least for this ci pack is just to help people
  scaffold ci quickly. not be opinionated about default. so i guess it's an explicit flag?"* An
  interactive "which platform?" prompt was raised and rejected — no prompt mechanism of any kind
  exists in `cmd/backstop/` today (verified), so it would be new, unjustified scope; the founder
  accepted that.

  **REQ-016 → v1.1.0.** The default is DROPPED. `--ci <recipe>` must be passed explicitly; core
  holds no platform literal, no name list, and no name→recipe mapping, passing the flag's value
  straight through to `recipe apply <pack>:<value>`. The accepted values are whatever recipe
  names the installed pack declares — the "per-platform templates / selected variant" framing is
  gone, since it implied a mapping layer core would own. Omitting `--ci` SKIPS CI wiring and
  init must SAY SO — naming `backstop recipe apply` and the pack's declared recipe names as the
  way to wire it later — as a deliberate no-op with an honest report, NOT an error exit,
  matching how REQ-011 reports an uninstalled toolchain and REQ-035 reports a brownfield
  preserve. OQ-7's resolution text carries the matching dated correction (old text struck,
  correction appended, ruling cited), including withdrawal of its now-dead "default-on makes
  enforcement real fast" rationale. **REQ-017 → v1.1.0** as a consequence: its loud failure is
  scoped to the ASKED-FOR-and-unavailable case, because under the old default-on model the CI
  step always ran and now it does not. Four live-prose restatements of "wires CI by default"
  (solution approach, target experience step 2, the Draft Requirements CI grouping, DD-1) and
  the init spec seed's summary were corrected in place; Version History entries describing the
  0.6.0/0.10.0 passes are left as the historical record they are.

  **NIT — REQ-014 / REQ-035 exit-code precedence was unstated.** One init run can capture a
  baseline (REQ-014: exit 0) and hit a brownfield CI preserve (REQ-035: reflected "in its exit
  posture" under DD-15's REFUSE). **REQ-014 → v1.1.0** states the precedence without inventing
  behavior: its exit-0 rule governs only the CLASSIFICATION OF PRE-EXISTING FINDINGS — inherited
  violations are not an init failure — and does not make init's overall exit 0 when an init STEP
  failed to deliver what it promised. Where both fire, REQ-035 wins on the exit code; both facts
  are still reported in full.

  **NIT — stale source citation.** Sharp Edges and the 0.10.0 CONCERN 7 entry both cited
  `pkg/recipe/apply.go:445-478` for `applyInsert`. Verified against the tree: `applyInsert` is
  `pkg/recipe/apply.go:639-683` (line 445 is now inside `coveredDivergence`). Both occurrences
  corrected. ISSUE-081 carries the same stale range at its line 111 and in three sibling
  citations (`applyMerge` cited 617-659, actual 877-939; `siteFor` cited 551-559, actual
  774-806; `effectiveParams` cited 900-913, actual 1175-1191) — NOT edited from here, since
  issue text is the issue track's to correct; flagged for that track.

  **NIT — REQ-032's retirement was invisible in live prose.** The References entry for
  `backstop-packs/BACKSTOP-INIT-REQUIREMENTS.md` described it as the source of "REQ-032 /
  REQ-033" with no indication REQ-032 is retired; it now says so inline and points at Version
  History 0.10.0.

  **Net requirement change: 33 → 33.** No requirements added or retired. Amended to new versions
  with dated corrections: REQ-014, REQ-016, REQ-017 (all → v1.1.0). Seed partition, requirement
  IDs, and implementation order are unchanged.
- 0.10.3 (2026-08-14): **REQ-022 ownership note — the guards seed is owned 7-of-8, not 8-of-8.**
  Narrow correction pass; no requirement added, retired, or amended; no seed re-partitioned; no OQ
  resolved; maturity unchanged at `defined`.

  **STALENESS — the Spec Seeds table read as if SPEC-068 covered all 8 guards requirements.**
  SPEC-068 (Trustworthy Green Guards, v1.2.3) pins SEVEN: REQ-021, REQ-026, REQ-027, REQ-028,
  REQ-029, REQ-030, REQ-034. REQ-022 was deliberately NOT pinned during spec authoring — its
  v1.1.0 text defers the mechanism wholly to BUNDLE-020 (Pack Core Version Compatibility), whose
  OQ-2 (enforcement point) and OQ-3 (failure posture) are still OPEN with BUNDLE-020 itself still
  `exploring`. With no producer for the capability-gap diagnostic, specifying it would have meant
  shipping a stub or answering BUNDLE-020's open questions, which this bundle's own consumption
  boundary forbids. SPEC-068's Dependencies section records the deferral and names a delta spec
  against itself as REQ-022's future owner, once OQ-2 and OQ-3 resolve. Found and confirmed during
  PLAN-SPEC-068's implementation (its Review Question 13 flagged it report-not-fix, correctly: a
  spec author reclassifying a requirement in its own source bundle is the drift the bundle-agent
  convention exists to prevent). Two dated corrections added — one under Draft Requirements naming
  the 7 SPEC-068 owns and marking REQ-022 unowned, one on the Version-skew guards bullet in the
  guards Spec Seed. The seed partition itself is CORRECT and unchanged: REQ-022 belongs to this
  seed; it simply has no spec owner yet, the same posture REQ-033 carries as DANGLING and
  REQ-012 / REQ-013 / REQ-018 carry as CONSUMED.
- 0.10.2 (2026-08-13): **Third-review follow-up — one blocker: REQ-016's `--ci` value shape did
  not match the recipe-ref API contract.** Narrow pass; nothing else in the bundle was touched
  and maturity is unchanged at `defined`.

  **BLOCKER — REQ-016 v1.1.0 mandated a command the shipped code hard-rejects.** v1.1.0 said the
  `--ci` value is passed "STRAIGHT THROUGH to the CI recipe pack as `recipe apply
  <pack>:<value>`". Verified against `pkg/recipe/resolve.go:60-88` (`ParseRecipeRef`): the only
  accepted ref format is `<pack>:<recipe>@<version>`, the semver pin is MANDATORY — "there is no
  'latest', no default version, and no tolerance branch (CLM-049)" — and an unpinned ref is a
  hard parse error. So v1.1.0 left two things undefined and both resolved toward a core-side
  literal, the exact DD-13 bake the requirement exists to prevent: (1) the `@<version>` pin was
  absent from the mandated command form, meaning a literal implementation always errors and an
  implementer would have to invent a default or hardcoded version; (2) the `<pack>` half was
  implied to be constructed by core, meaning core would hold a pack-name literal — and REQ-017
  compounded it by presupposing a "which installed pack is THE CI pack" detection rule that
  exists nowhere in the tree.

  **REQ-016 → v1.2.0.** `--ci` now takes the FULL PINNED RECIPE REF as its value —
  `backstop init --ci backstop-ai/ci-workflows:github-actions-gate@1.0.0` is stated in the
  requirement as the exact command-line shape so it cannot be misread as a bare recipe name
  again — handed to `recipe apply` VERBATIM. Core constructs and interprets no part of the
  string: not the pack name, not the recipe id, not the version. This is a UX-shape correction
  consistent with the founder's 2026-08-12 ruling already recorded at 0.10.1 (no core-side
  platform knowledge, explicit flag, no interactive prompt) and needed no new founder check-in;
  it strengthens that ruling by removing the last two core-held literals it had left implicit.
  The four recipes in `backstop-ai/ci-workflows@v0.1.0` each declare recipe version `1.0.0`
  (verified in the pack's `recipes/*/recipe.yml`), which is where the example's pin comes from.

  **REQ-017 → v1.2.0 — no new detection machinery; the existing error path already covers it.**
  Because the consumer now names the pack explicitly in the ref, core has nothing to detect.
  Verified against `pkg/recipe/resolve.go:100-131` (`ResolveRecipe`): an uninstalled pack, an
  undeclared recipe id, and a pin/manifest version mismatch are each already a distinct
  fail-loud error naming what was missing and listing what IS available. REQ-017 is restated to
  say plainly that init SURFACES that error, attributed to the CI step, while the other init
  steps still complete — replacing the old "guidance naming the pack to add" language, which
  described detection logic init would have had to own.

  Live-prose restatements of the old bare-name form were corrected in place with dated
  corrections: solution approach, target experience step 2, the Draft Requirements CI grouping,
  DD-1, OQ-7's resolution (both its command-form sentence and its asked-for-and-unavailable
  tail), and the init spec seed's summary. Archived requirement version-log entries (REQ-016
  v1.0.0/v1.1.0, REQ-017 v1.0.0/v1.1.0) and Version History entries for the 0.6.0/0.10.0/0.10.1
  passes are left as the historical record they are.

  **Net requirement change: 33 → 33.** No requirements added or retired. Amended to new versions
  with dated corrections: REQ-016, REQ-017 (both → v1.2.0). Seed partition, requirement IDs,
  and implementation order are unchanged.

- 0.11.0 (2026-08-20): **Promoted to `delivered` — founder-ruled.** All three spec seeds are
  built and their specs are `status: implemented`: **SPEC-068** (Trustworthy Green Guards),
  **SPEC-069** (Backstop Init), **SPEC-070** (Backstop Doctor) — implemented in that order, which
  is the sequencing this bundle's Spec Seeds section prescribed (guards first, because REQ-004 is
  unimplementable without REQ-029 and every acceptance claim the other two seeds make is only as
  trustworthy as the validator asserting it). No requirement was added, retired, or amended in
  this pass; no seed was re-partitioned; no OQ was reopened. All seven OQs have been resolved
  since 2026-07-13 (0.6.0). This entry records the maturity change, the `bundle.updated` bump, and
  the two staleness corrections below. `delivered` is the schema's success-terminal maturity
  (`artifacts/bundle/v2/schema.json` enum); note it is NOT treated as terminal by the validator's
  exemption path, so the `requirements[]` array remains required and remains populated at 33.

  **PROMOTION IS FOUNDER-RULED, NOT SELF-ASSESSED.** The founder ordered this promotion after the
  underlying work was jointly verified as done. Facts were re-verified independently before
  writing rather than taken on the brief's word — each spec's `status` read from its own
  frontmatter, and every `onboarding-experience:REQ-NNN` in the three specs' `supports:` fields
  diffed against this bundle's 33 requirement ids. That diff is what surfaced the second carve-out
  below, which the brief had not named; it was escalated to the founder and the promotion was
  re-authorized on the corrected three-item picture rather than proceeding on the original
  one-item one.

  **READ `delivered` PRECISELY: 30 of 33 requirements have a shipping mechanism, and the
  remaining three are accounted for in three DIFFERENT postures.** This bundle is delivered in the
  sense that its seeds are specced and implemented and no further spec work is queued against it —
  NOT in the sense that every requirement has running code behind it. The three:

  1. **REQ-022 — UNOWNED, deferred to BUNDLE-020.** Recorded at 0.10.3. SPEC-068 pins seven of
     the guards seed's eight and deliberately declined this one: its v1.1.0 text defers the
     mechanism wholly to BUNDLE-020 (Pack Core Version Compatibility), whose DD-4 is
     founder-resolved (capability-SET comparison, not version ordering) but whose OQ-2
     (enforcement point) and OQ-3 (failure posture) are both still OPEN, with BUNDLE-020 itself
     still `exploring`. Future owner: a delta spec against SPEC-068 once those resolve.
  2. **REQ-024 — CARVED OUT, blocked on an unowned pack-manifest surface. NEW IN THIS PASS.**
     SPEC-070 implements REQ-020, REQ-023 and REQ-025, and carries no requirement and no mandated
     claim for REQ-024. The pack-declared stack-policy surface it would read does not exist at
     HEAD and no artifact owns creating it; ruled 2026-08-13 to carve it out rather than invent
     the field, on the same rule that produced REQ-005 v1.1.0's accepted residue — pack-manifest
     surface design is BUNDLE-004's. `issues/ISSUE-121-pack-manifest-missing-stack-policy-
     surface.issue.md` is filed (open, 2026-08-13) and is being homed under BUNDLE-004. SPEC-070
     holds a tripwire absence claim asserting its check registry reads no stack-policy surface, so
     the carve-out cannot be closed from the wrong artifact without a test going red.
  3. **REQ-033 — OWNED but SHIPPED UNSATISFIED.** Distinct from the other two: SPEC-069 DOES pin
     it, but what shipped is an honest report of the gap rather than the wired coverage floor. The
     DANGLING flag raised at 0.10.0 was never cleared — no artifact adopted the knob, and
     `backstop.yml`'s `additionalProperties: false` rejects writing it — so init reports and
     SPEC-069 carries absence claims proving no schema surface was invented to fake it.

  **STALENESS — the doctor seed read as 4-of-4 covered.** The same defect 0.10.3 fixed for the
  guards seed was still live for the doctor seed: the Draft Requirements table row, the Standalone
  diagnostics bullet, and the `backstop doctor` spec-seed bullet all described REQ-024 as in scope
  and delivered, with nothing recording the 2026-08-13 carve-out. Three dated corrections added
  (one ownership block under the seed table, one on the diagnostics bullet, one on the seed
  bullet). The seed PARTITION is correct in every case and is unchanged — REQ-024 belongs to the
  doctor seed; it simply has no spec owner. A fourth note was added under the seed table
  distinguishing the three postures this bundle now holds — CONSUMED (REQ-012 / REQ-013 /
  REQ-018's local-provenance half), CARVED OUT / UNOWNED (REQ-022, REQ-024), and
  OWNED-BUT-UNSATISFIED (REQ-033) — so a future reader is not left to infer them.

  **Net requirement change: 33 → 33.** Nothing added, retired, or amended. Requirement IDs, seed
  partition, and implementation order unchanged. What changed is maturity, `bundle.updated`, and
  the honesty of three stale prose locations.

## References

### Requirements corpus (hand-onboarding write-ups — one per profile)

These are the empirical requirements corpus for the `backstop init` and `backstop doctor`
specs, not incidental links. Both are currently UNTRACKED field notes living in OTHER repos;
treat them as transient. Their durable home is this bundle — DD-8 and the sharp-edge
decisions/OQs/notes above lift the load-bearing substance in so it survives the docs being
deleted.

- **`backstop-packs/BACKSTOP-INIT-DOGFOOD-FULL-SDLC.md`** — the **full-SDLC greenfield**
  profile hand-onboarding (empty repo → `backstop gate` PASS, 2026-07-12; version-skew
  instance #2 appended 2026-07-16; artifact-layout founder ruling appended 2026-07-17). Its
  "Manual steps performed" list is transcribed into DD-8 as init's happy-path sequence; its
  ranked sharp edges are the source of DD-9, DD-15, and REQ-026..031. Previously cited here
  as `bclabs-portal/docs/dogfood/init-sharp-edges.md` — same corpus, relocated alongside its
  pack-only sibling (path corrected 2026-08-11).
- **`backstop-packs/BACKSTOP-INIT-REQUIREMENTS.md`** — the **pack-only** profile
  hand-onboarding (consumer adopting packs without SDLC artifacts). Source of the
  `enforcement.policy` `level: off` requirement in DD-2, and of REQ-032 / REQ-033 (REQ-032 is
  RETIRED — see Version History 0.10.0; this write-up remains its source of record, not a live
  requirement).

### Related artifacts

- DIR-001: Release Workflow — owns binary distribution + system-toolchain acquisition +
  post-merge baseline generation (out of scope here; DD-10).
- DIR-002: `backstop init` command — the directive this bundle sources.
- BUNDLE-007: Baseline — owns baseline mechanics (CI-generated post-merge, gitignored local
  cache, ratchet-only); resolves this bundle's baseline questions (DD-10).
- BUNDLE-001: Pack distribution — default pack wiring depends on packs being distributable.
- SPEC-008: Code check (diff-based scope via ResolveScope).
- SPEC-010: Gate (baseline step consumes BUNDLE-007's output).

#### Added 2026-08-12 (v0.10.0) — dependencies this bundle relies on that were not cited

- **BUNDLE-015: Pack Scaffolding Recipes** (`defined`) — OWNS the pack-recipe capability
  REQ-009 / REQ-016 consume and DD-12 depends on. Previously described here as a
  not-yet-existing dependency "likely its own bundle"; it IS that bundle, and it is delivered.
- **DIR-019: Pack recipe capability** — the directive driving BUNDLE-015. Its 2026-08-12
  correction explicitly CLEARS the DIR-002 blocking-dependency claim this bundle repeated in
  four places.
- **SPEC-054: Recipe Apply And Manifest** (`status: implemented`) — the delivered mechanism
  (`pkg/recipe` + `recipe apply`). Its own text names `backstop init` (BUNDLE-003 DD-12) as the
  consumer it unblocks.
- **SPEC-067: CI Recipe Pack** (`status: implemented`) — the delivered CI recipe pack, published
  and installed as `backstop-ai/ci-workflows@v0.1.0`; satisfies BUNDLE-015 REQ-018 and OQ-7's
  "concrete CI recipe pack" dependency.
- **BUNDLE-020: Pack Core Version Compatibility** (`exploring`) — OWNS the pack↔core
  compatibility mechanism REQ-022 now CONSUMES. DD-4 (founder-resolved 2026-07-26) settles
  capability-SET comparison over version ordering; OQ-2 (where enforced) and OQ-3 (failure
  posture) remain OPEN and must not be resolved from this bundle.
- **BUNDLE-021: Pack Command Execution Governance** (`exploring`) — holds the unresolved
  posture on pack-declared command execution that, together with BUNDLE-019's reserved-but-
  unbuilt `step` executor, made retired REQ-032 unbuildable here.
- **BUNDLE-019: Runbooks** (`exploring`) — owns the step EXECUTOR that a pack-declared install
  command would have required; the `step` op is reserved, not executed.
- **ISSUE-056: Local-first baseline seeding** (open) — OWNS the mechanism behind REQ-012 /
  REQ-013; cites this bundle's OQ-3 as its source of direction.
- **ISSUE-055: Local provenance cache for local packs** (open) — OWNS the local-provenance half
  of REQ-018; cites this bundle's OQ-4 as its source of direction.
- **ISSUE-119: Recipe payloads have no merge/insert op** (open) — the brownfield CI false-success
  path REQ-035 answers on init's side. See Sharp Edges.
- **ISSUE-081: Recipe authoring surface underspecified** (open) — Gap 3 (insert placement
  semantics) is a live risk for any init step using an `insert` op. See Sharp Edges.
- **ISSUE-110: Recipe payload foreign template escape** (open) — no escape syntax for foreign
  `{{ }}`; bites CI templates whose own syntax collides. See Sharp Edges.
- **SPEC-005: CLI Foundation** (`status: draft`) — PREDATING artifact whose "cohort derived from
  the set of embedded schemas" claim REQ-026 supersedes with content-derivation. See
  Observations.
