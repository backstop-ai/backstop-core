---
title: "Backstop Init"
number: SPEC-069
created: "2026-08-13"
updated: "2026-08-14"
status: draft
schema_version: spec/v1
spec_version: 1.3.0

source:
  bundle: BUNDLE-003
  directive: DIR-002

implementation:
  summary: >
    BUNDLE-003's `backstop init` seed and nothing else: a NEW `backstop init` command
    (`cmd/backstop/init.go`) over a NEW orchestration package (`pkg/initialize`) that takes a
    consuming project from "the binary is present" to first useful output in one prompt-free
    invocation. Init resolves a SUBTRACTABLE capability set, writes the profile-correct
    `backstop.yml`, runs `git init` only when there is no `.git`, scaffolds the
    `.backstop/`-rooted artifact layout for the full-SDLC profile, installs only the pack refs
    the consumer named, THEN emits the canonical `.gitignore` (whose entry set is a function of
    those installed packs), applies a SCAFFOLD recipe when the consumer names one so the
    project holds at least one source file before anything compiles it (DD-7), executes the
    pack-declared
    test/build entrypoints once as ground truth through the gate's own allowlisted execution
    path, delegates local-baseline seeding, applies a CI
    recipe ONLY when the consumer passes a full pinned `<pack>:<recipe>@<version>` ref to
    `--ci`, runs the gate once and reports its findings as OBSERVATION, and prints a per-step
    report whose exit code carries broken promises and nothing else. The hard invariant this
    spec is built around: init performs ZERO language, framework, ecosystem, or CI-platform
    detection, and core holds no language name, no framework name, no platform name, no pack
    name, no recipe id, and no version literal anywhere on the init path — every one of those
    arrives as consumer input or as pack-declared data. Init reuses the SHIPPED mechanisms
    rather than reimplementing them: `pkg/recipe`'s `ParseRecipeRef`/`ResolveRecipe`/`Apply`
    for BOTH the CI wiring and the source-file scaffold (one applier seam, two consumer-named
    refs, zero core-constructed ref parts), `pkg/pack/distribution`'s add path for pack installs, `pkg/gate` for the
    observation run, and `pkg/config` for the config it writes. THREE requirements in the seed
    are consumed, not built: REQ-012 and REQ-013 (local-baseline seeding and the remoteless
    `baseline_comparison` message) belong to ISSUE-056, and REQ-018's local-provenance half
    belongs to ISSUE-055 — this spec's obligation for all three is a delegation seam plus
    ABSENCE claims proving init did not re-implement them. REQ-033 (a spec-independent coverage
    floor for the pack-only profile) is a DOCUMENTED, UNSATISFIED GAP: the knob it would wire
    does not exist, `artifacts/backstop-yml/v1/schema.json` declares
    `enforcement.additionalProperties: false`, so writing it would be rejected at load — init
    reports the forfeiture instead of inventing an enforcement-policy surface. OUT OF SCOPE and
    named as such: `backstop doctor` (SPEC-070), the trustworthy-green guards including the
    config-resolved artifact root init CONSUMES (SPEC-068), baseline generation mechanics
    (BUNDLE-007), binary distribution (DIR-001), pack-declared dependency INSTALLATION (retired
    bundle REQ-032), and any change to `pkg/gate`, `pkg/recipe`, or the `backstop.yml` schema.
  subject: pkg/initialize

verification:
  level: integration
  test_command: go test ./pkg/initialize/... ./cmd/backstop/ -race -coverprofile=cover.out
  coverage_threshold: 80

requirements:
  - id: REQ-001
    supports: onboarding-experience:REQ-001@1.0.0
    text: >
      A single `backstop init` invocation must take a project directory from "the binary is
      present" to first useful output with no second command required in between. One
      invocation writes `backstop.yml`, establishes the profile's layout, emits the canonical
      `.gitignore`, performs every capability in the resolved set, and prints one report
      naming every step and its outcome. Init must not instruct the consumer to run a further
      command in order to complete a step init itself claimed.
  - id: REQ-002
    supports: onboarding-experience:REQ-002@1.0.0
    text: >
      Bare `backstop init` must run the FULL default capability set with no prompting and no
      read of standard input, and subtraction flags must be the ONLY way to narrow it. The
      capability set is exactly seven backstop-vocabulary names — `git`, `sdlc`, `gitignore`,
      `packs`, `toolchain`, `baseline`, `observe` — and `backstop.yml` generation is NOT a
      capability (it is unconditional; init without it produces nothing). `--no-<cap>` removes
      one capability; `--only <cap>` (repeatable) narrows the set to exactly the named
      capabilities. `--only` may never ADD a capability outside the default seven, `--only`
      and `--no-` may not be combined (config error, exit 2), and an unrecognized capability
      name is a config error (exit 2) naming the seven valid names. Because init never
      prompts, it must behave identically with stdin closed and with no TTY. There is no
      `--no-ci` capability: CI is governed solely by `--ci` presence (REQ-016).
  - id: REQ-003
    supports: onboarding-experience:REQ-003@1.0.0
    text: >
      Init must generate the `backstop.yml` correct for the profile without the consumer
      hand-writing policy, where the profile is DERIVED from the resolved capability set
      rather than selected by a separate flag: `sdlc` present is the full-SDLC greenfield
      profile, `sdlc` subtracted (`--no-sdlc`) is the pack-only profile. NAMED EXACTLY,
      because "the artifact-pipeline keys" names nothing the schema has: the shipped
      `artifacts/backstop-yml/v1/schema.json` declares exactly six top-level properties —
      `project`, `language` (retired/legacy), `runtimes`, `enforcement`, `packs`,
      `registries` — under `additionalProperties: false`, and SPEC-068 (bundle REQ-029) adds
      a seventh, `artifact_root`. The full-SDLC config therefore carries `project:` plus
      `artifact_root: .backstop` (the write itself is REQ-004's step) and sets NO SDLC
      dimension to `off`. Neither profile writes any other top-level key.
      The pack-only config must additionally set `enforcement.policy` `level: off` for EVERY
      one of the five SDLC dimensions — `test_verification`, `coverage_threshold`,
      `contract_signature`, `test_substantiveness`, `artifact_status_drift` — because those
      dimensions hard-error on a missing `specs/` directory rather than skipping. The retired
      `language:` key must NEVER be written by either profile. The `project:` value is the
      target directory's basename and is derived from nothing else — no file in the project is
      read to name it.
  - id: REQ-004
    supports: onboarding-experience:REQ-004@1.0.0
    text: >
      For the full-SDLC profile init must scaffold the `.backstop/`-rooted artifact layout —
      `.backstop/bundles/`, `.backstop/specs/`, `.backstop/plans/`, `.backstop/issues/`,
      `.backstop/adrs/`, `.backstop/directives/` — and must write the `artifact_root` key
      SPEC-068 (bundle REQ-029) adds to the schema and typed loader, with the value
      `.backstop`, so discovery resolves the layout init created. Init must NEVER create
      root-level `specs/`, `bundles/`, `plans/`, `issues/`, `adrs/`, or `directives/` in a
      consumer repo; backstop-core's own root layout is a framework exception init does not
      produce and does not police.
      SIX OF SEVEN, DELIBERATELY. SPEC-068's layout table covers seven artifact kinds — spec,
      plan, adr, bundle, issue, directive, AND capability — and init scaffolds no
      `capabilities/` directory. This is a decision, not an omission: the six init creates are
      exactly the work products of the two consumer tracks (issue→plan and
      bundle→spec→plan→implementation), whereas a capability artifact declares a named
      contract at the pack↔core wire seam (backstop-core's own
      `capabilities/CAP-001-pack-gate-enforcement/`, whose on-disk shape is a DIRECTORY PER
      ARTIFACT holding `capability.yml` plus a `.feature` file, unlike the flat file-per-
      artifact shape of the other six). It is authored by framework and pack authors, is not
      produced by any step of a consuming project's onboarding, and its governing mechanism is
      BUNDLE-020's, so init pre-creating an empty `capabilities/` would scaffold a directory
      no verb in the flow init just ran can fill.
      The pack-only profile scaffolds no artifact directories at all.
  - id: REQ-005
    supports: onboarding-experience:REQ-005@1.1.0
    text: >
      Init must emit one canonical `.gitignore` whose backstop-owned entries are exactly three
      literals stated in core — `.backstop/packs/`, `.backstop/baseline.json`,
      `.backstop/pack-config-provenance.json` — PLUS, for each installed pack, every engine's
      declared `stdout_artifact` path (`pkg/pack/manifest.go:97`, yaml key `stdout_artifact`),
      which is the only generated-output declaration a pack manifest carries. No language,
      framework, or tool-specific path may be enumerated in core. An engine declaring no
      `stdout_artifact` contributes no entry. ACCEPTED RESIDUE: `stdout_artifact` names only
      what an engine writes for the gate to read, so generated paths a toolchain produces
      incidentally are not covered; init must STATE that residue in its report and leave those
      paths to the consumer rather than guessing them. An existing `.gitignore` is appended to,
      never rewritten: every pre-existing byte survives, missing entries are appended, and an
      entry already present is not duplicated.
      ORDERING IS PART OF THIS REQUIREMENT, not an implementation detail: because the entry set
      is a function of the INSTALLED packs, the gitignore step must run AFTER the pack-install
      step (REQ-019), or a `--pack` run on a fresh repo emits a `.gitignore` missing every
      pack-derived entry — precisely the cross-repo ignore divergence DD-7 exists to end.
  - id: REQ-006
    supports: onboarding-experience:REQ-006@1.0.0
    text: >
      Init must run `git init` only when the target directory contains no `.git`, and must
      leave an existing repository's git state entirely untouched — no re-init, no config
      write, no ref or HEAD mutation.
  - id: REQ-007
    supports: onboarding-experience:REQ-007@1.0.0
    text: >
      Re-running init must converge and never clobber. Init detects only backstop-neutral
      facts — the presence of `.git`, of `backstop.yml`, of the artifact directories, of
      `.gitignore` — adds only what is missing, overwrites no existing consumer file, and
      reports its findings purely in backstop terms. An existing `backstop.yml` is preserved
      byte-for-byte and reported as already present. A second init run immediately after a
      first must write no new file and must report every step as converged. No file outside
      backstop's own surface may be read to make any of these decisions.
  - id: REQ-008
    supports: onboarding-experience:REQ-008@1.0.0
    text: >
      Init must perform ZERO language, framework, ecosystem, or CI-platform detection. No
      language name, framework name, ecosystem-manifest filename, or CI-platform name may
      appear in the init source set (`cmd/backstop/init*.go` and `pkg/initialize/**`, excluding
      `_test.go` files); that knowledge lives in packs as data and in the consumer's own
      arguments, and reaches init only through pack manifests, recipe refs, and flags. Init's
      observable behavior must be identical across two projects whose only difference is the
      ecosystem marker files they contain.
  - id: REQ-009
    supports: onboarding-experience:REQ-009@1.0.0
    text: >
      Init must apply pack-supplied recipes ONLY through the shipped generic mechanism —
      `recipe.ParseRecipeRef` → `recipe.ResolveRecipe` → `recipe.Apply` — and must not
      interpret, rewrite, render, or language-specialize template content. Init contains no
      template engine, no payload rewriting, and no path defaulting of its own: a recipe
      payload lands byte-identical to what the pack declared, at the path the recipe declared.
      EVERY LANGUAGE-SPECIFIC ARTIFACT INIT PRODUCES ORIGINATES IN A PACK RECIPE, AND A FIRST
      SOURCE FILE IS ONE OF THEM. This is bundle REQ-009's own scope, restored verbatim after a
      draft of this spec narrowed it: the bundle requires that "every language-specific artifact
      init produces (a first source file, toolchain config) must originate in a pack recipe,
      never in core", and DD-7 states the obligation with its evidence — init "scaffolds at
      least one source file" because a compiler run over an empty repo can RED on "no inputs"
      (observed: `tsc` TS18003), and that scaffolded file "comes from a pack recipe, never from
      core". The 2026-08-12 bundle correction struck DD-7's TypeScript-flavored ignore LIST and
      explicitly reaffirmed this half with its rationale, and retiring bundle REQ-032 removed
      DD-8 step 1's "install dependencies" clause, not its "scaffold" clause. Init therefore
      CARRIES A SCAFFOLD STEP, and that step is a recipe apply and nothing else: init writes no
      source-file CONTENT of its own, renders no template, invents no path, and holds no
      filename, extension, language name, or payload literal for it. What REQ-009 forbids is
      core AUTHORING the file; what it requires is init DELIVERING it through a pack.
      THE FLAG TAKES THE `--ci` SHAPE, DELIBERATELY. The scaffold recipe is named by the
      consumer through `--scaffold <pack>:<recipe>@<version>` — the FULL PINNED REF, OPAQUE to
      core, handed to the same shipped resolve+apply path VERBATIM. Core constructs no part of
      it (not the pack name, not the recipe id, not the version), never interprets, defaults,
      maps, or completes it, and holds no default scaffold recipe and no scaffold-pack roster. A
      consumer naming an entirely different pack is equally valid, precisely because core never
      inspects the ref. SCAFFOLD IS NOT A CAPABILITY: it adds no eighth name to REQ-002's
      seven-name vocabulary, there is no `--no-scaffold`, and no `scaffold` verb or subcommand
      is added — presence of the flag is the whole of the opt-in, exactly as with `--ci`.
      WHEN `--scaffold` IS OMITTED, the step is SKIPPED and init must SAY SO: state that no
      source file was scaffolded, name `backstop recipe apply` plus the pinned
      `<pack>:<recipe>@<version>` ref shape as the way to do it later, and exit as a deliberate
      no-op. This is REQ-016's omitted-`--ci` posture for the same reason — not every pack
      ecosystem ships a scaffold recipe, and a skipped optional step is not an error. The
      failure mode DD-7 cites (a compiler reding on an empty project) then arrives, correctly,
      as REQ-011's own case (b)/(c) report rather than as an init-time hard block: init reports
      what the entrypoint did and does not diagnose why.
      WHEN `--scaffold` IS SUPPLIED AND THE REF CANNOT BE RESOLVED, the step fails LOUDLY on
      REQ-017's posture applied to this step: the shipped error surfaces VERBATIM attributed to
      the SCAFFOLD step, every other init step still completes, init adds no guidance of its own
      and classifies no failure mode differently, and init exits non-zero. Preserves a scaffold
      apply returns are classified by the SAME classifier REQ-035 defines, over the same three
      observable classes with the same exit-code consequences — init holds ONE preserve
      classifier, never one per step.
      INIT SUPPLIES NO RECIPE PARAM, TO EITHER RECIPE. Neither `--ci` nor `--scaffold` carries a
      param surface, so a recipe declaring a param that is required with no default cannot be
      applied by init at all: the shipped apply's own error surfaces verbatim and the consumer
      completes it with `backstop recipe apply --param`. Init must not invent a param value and
      must specifically not derive one from the target directory's name or from anything it
      found on disk — deriving one would be core constructing recipe input, which is the same
      defect as core constructing half a recipe ref.
  - id: REQ-010
    supports: onboarding-experience:REQ-010@1.0.0
    text: >
      Packs must enter a consumer project only through an explicit consumer act. Init installs
      exactly the pack refs supplied via the repeatable `--pack <ref>` flag, in the order
      given, and installs nothing else. Init holds no pack roster and no pack-name literal, has
      no concept of a "primary language", and must never select a pack from anything it finds
      on disk. Bare init (no `--pack`) installs zero packs and must SAY SO rather than skip
      silently, naming `backstop pack add` as the way to add them.
  - id: REQ-011
    supports: onboarding-experience:REQ-011@1.1.0
    text: >
      Init must verify the toolchain actually RUNS by executing, once each, the command of
      every installed pack engine whose declared `gate_type` is `test` or `build`, and taking
      that engine's outcome from THAT command's own exit status and from nothing adjacent —
      never from a package-manager command init ran itself (it runs none), never from a
      configuration file, and never from another engine's command in the same pack. Each
      executed entrypoint produces its own separately-reported outcome, and the outcomes are
      independent: one entrypoint exiting non-zero does not change another's.
      THE FAILURE OUTCOME IS SPLIT IN TWO, AND THE LABELS DIFFER. Bundle REQ-011 v1.1.0
      forbids inferring toolchain health "from package-manager configuration OR EXIT CODE" and
      conditions its owed-setup clause on cause — "WHEN THE CHECK FAILS BECAUSE THOSE
      DEPENDENCIES ARE ABSENT". Treating any non-zero exit as owed setup would commit exactly
      the exit-code cause-inference the requirement exists to forbid, and would re-enact the
      pnpm `ERR_PNPM_IGNORED_BUILDS` misdiagnosis that is the requirement's own evidence. So,
      matching SPEC-070 REQ-006's (b)/(c) split for the same check:
      (b) THE DECLARED EXECUTABLE CANNOT BE STARTED AT ALL — reported as a SETUP step the
      consumer still owes, naming the pack whose entrypoint could not run and pointing at that
      pack's own documented install steps, inventing no install command and installing nothing.
      (c) THE COMMAND STARTED AND EXITED NON-ZERO — reported with its exit code and its
      captured output VERBATIM, attributed to the pack and the command, with NO cause claimed:
      init must not call this owed setup, must not name dependencies, and must not attribute it
      to a wrapper, because init cannot tell.
      Both (b) and (c) make init exit non-zero (DD-15: init could not verify the toolchain runs,
      so it must not report success). WHAT INIT CANNOT DISTINGUISH, STATED HONESTLY: a pack
      declares ONE command per engine, so if the declared entrypoint is itself a wrapper
      invocation the exit status init reads IS the wrapper's. Core cannot see past it and must
      not pretend to — which is why (c) reports and does not diagnose, and why choosing a
      command whose exit status means what the pack intends is the PACK AUTHOR's obligation.
      EXECUTION SAFETY IS BOUND, NOT ASSUMED. Every entrypoint command runs through the SAME
      three steps the gate's own engine dispatch takes — `checkEngineToolAllowed`
      (`cmd/backstop/pack_gate.go:812`, the trusted-tool allowlist plus lock-pin trust gate),
      then `splitCommand` (`pack_gate.go:887`, whitespace argv tokenization), then the shared
      `check.CommandRunner` — which is `runFindingsEngine`'s path (`pack_gate.go:573-600`)
      minus the SARIF parse. Init must introduce NO second, weaker way to run a pack-declared
      command and must NEVER run one through a shell. A command whose tool the allowlist
      refuses is not executed at all: the trust gate sits BEFORE `splitCommand` and before the
      runner, and its refusal is reported as a config error rather than as a toolchain verdict.
      THE RUNNER METHOD IS NAMED, AND IT IS THE ONE DELIBERATE DIFFERENCE FROM
      `runFindingsEngine`. Init enters the shared runner through `check.CommandRunner.Run`
      (`pkg/check/runner.go:17`, COMBINED stdout+stderr), NOT through `RunStdout`
      (`runner.go:21`, stdout only) which is what `runFindingsEngine` calls
      (`pack_gate.go:648`). The two methods exist for opposite reasons and the shipped comments
      say so: `RunStdout` exists so a tool's stderr banner cannot corrupt the SARIF bytes on
      stdout, and `Run` exists for "the build/test executors whose violation messages may
      legitimately include stderr" — which is exactly case (c). A failing build or test
      entrypoint commonly writes its diagnostics to stderr, so binding init's capture to
      `RunStdout` would print an EMPTY "captured output" for precisely the failures case (c)
      exists to surface. Init parses nothing and renders to a human, so stderr contamination of
      a machine format is not a hazard here; losing the diagnostic is. The allowlist gate and the
      splitter are shared UNCHANGED — only the capture method differs, and it differs on purpose.
      When no installed pack declares a `test` or `build` engine there is nothing to execute:
      the step reports capability-absent and does not fail the run.
  - id: REQ-012
    supports: onboarding-experience:REQ-012@1.1.0
    text: >
      CONSUMED, NOT BUILT HERE. The gitignored local baseline at `.backstop/baseline.json` is
      owned by ISSUE-056; this spec designs none of that machinery. Init's `baseline`
      capability is a DELEGATION SEAM only: when a seeding implementation is available init
      invokes it exactly once and reports what it returned; when none is available the step
      reports the gap naming ISSUE-056 as its owner and does NOT fail the run — an un-adopted
      capability is a missing benefit, not a broken promise. The init source set must contain
      no code that writes `.backstop/baseline.json` or computes a baseline fingerprint.
  - id: REQ-013
    supports: onboarding-experience:REQ-013@1.1.0
    text: >
      CONSUMED, NOT BUILT HERE. The self-consistency of the `baseline_comparison` message on a
      remoteless repository is pure `pkg/gate` machinery owned by ISSUE-056. This spec must
      change no file under `pkg/gate`, and init must not paper over that message by rewriting,
      suppressing, or substituting for it in its own report.
  - id: REQ-014
    supports: onboarding-experience:REQ-014@1.1.0
    text: >
      Init must run the gate once and present the result as OBSERVATION, not failure: findings
      grouped by gate dimension with counts, phrased as what was noticed, with no verdict
      language in the summary. PRE-EXISTING FINDINGS ALONE MUST NOT MAKE INIT EXIT NON-ZERO —
      inherited violations are not an init failure. That rule governs the classification of
      findings only; it does not make init exit 0 when an init STEP failed to deliver what it
      promised. Where one run triggers both this and a REQ-035 preserve of the USER-OWNED or
      INDETERMINATE class (a WAIVER-COVERED preserve is not a step failure and does not enter
      this precedence rule at all), or both
      this and a REQ-017 CI resolve failure or a REQ-009 SCAFFOLD resolve failure, the failed
      step WINS on the exit code and BOTH facts are reported in full.
  - id: REQ-015
    supports: onboarding-experience:REQ-015@1.0.0
    text: >
      After init, a `backstop gate` run with no scope flags must be diff-scoped — the shipped
      default (`cmd/backstop/gate.go`, `gate.GateScopeModeDiff`). Init must write no
      configuration that changes, overrides, or pins the gate's default scope, so this
      requirement is satisfied by not regressing it, and a mandated test must hold that line.
  - id: REQ-016
    supports: onboarding-experience:REQ-016@1.2.0
    text: >
      Init wires CI only when the consumer passes `--ci` EXPLICITLY, and the flag's value is
      the FULL PINNED RECIPE REF in `<pack>:<recipe>@<version>` form — e.g.
      `backstop init --ci backstop-ai/ci-workflows:github-actions-gate@1.0.0`. That whole
      string is OPAQUE to core and is handed to the shipped resolve+apply path VERBATIM: core
      constructs no part of it — not the pack name, not the recipe id, not the version — and
      never interprets, defaults, maps, or completes it. Core holds no platform literal, no
      list of platform names, no pack name, and no version. A consumer naming an entirely
      different pack is equally valid, precisely because core never inspects the ref. When
      `--ci` is OMITTED, CI wiring is SKIPPED and init must SAY SO: state that no CI was
      wired, name `backstop recipe apply` plus the pinned `<pack>:<recipe>@<version>` ref
      shape as the way to wire it later, and exit as a deliberate no-op — a skipped optional
      step is not an error. No `ci` verb or subcommand may be added.
  - id: REQ-017
    supports: onboarding-experience:REQ-017@1.2.0
    text: >
      When CI was asked for and the ref cannot be resolved, the CI step must fail LOUDLY while
      every other init step still completes, and init must exit non-zero. All four failure
      modes are covered and each surfaces the SHIPPED error VERBATIM, attributed to the CI
      step: a malformed or unpinned ref (`recipe.ParseRecipeRef`), a pack that is not installed
      ("is not among the installed packs"), a pack that declares no such recipe ("declares no
      recipe"), and a pinned version that does not equal the recipe's declared version. Any
      OTHER fail-loud error the shipped resolve path returns — notably an unreadable or
      unparseable colocated `recipe.yml`, which is a pack defect rather than a bad consumer ref
      — is governed by the same rule and surfaced identically; init classifies none of them
      differently. Init implements NO detection of its own: it never identifies "the CI pack", never probes for
      installation, and adds no guidance text beyond attributing the error to the CI step —
      any further guidance belongs to `recipe apply`, not to init. There is no baked
      per-platform fallback, so silent success is unrepresentable. This governs only the
      asked-for-and-unresolvable case; `--ci` omitted is REQ-016's reported no-op.
  - id: REQ-018
    supports: onboarding-experience:REQ-018@1.1.0
    text: >
      Init must install its packs through PORTABLE GIT-REF references only, so the
      `backstop.lock` it produces contains no machine-specific path: a `--pack` value that is a
      local filesystem path is REFUSED as a config error (exit 2) naming the portability reason
      and pointing at `backstop pack add` after init, and no lock entry init writes carries a
      `local_path` value. THERE IS EXACTLY ONE AUTHORITY ON WHAT A LOCAL PATH IS, and init must
      not become a second: the shipped classifier is `isLocalPath`
      (`pkg/pack/distribution/add.go:96`), which the add path already uses to fork local from
      remote. Init must call THAT function — exported from `pkg/pack/distribution` for the
      purpose if it is still unexported when this lands — and must define no predicate of its
      own. A second definition of "is this a local path" is the divergence SPEC-056 removed
      elsewhere by making `pack.ValidatePackName` "one authority, not a copy"; the two
      definitions drifting would let a ref init accepted be classified local downstream, which
      is the machine-specific lock entry this requirement exists to prevent.
      The LOCAL-PROVENANCE half is CONSUMED, NOT BUILT HERE — owner
      ISSUE-055 — so the init source set must contain no local-provenance cache, no
      pack-sources record, and no lock-schema change.
  - id: REQ-019
    supports: onboarding-experience:REQ-019@1.0.0
    text: >
      Init's happy path must execute the transcribed hand-onboarding sequence in this order —
      git init if needed → write the profile-correct `backstop.yml` → create the artifact
      directories → install each named pack → write the canonical `.gitignore` → apply the
      scaffold recipe when asked → verify the toolchain runs → seed the baseline → wire CI when
      asked → run the gate.
      TWO DEVIATIONS FROM DD-8'S TRANSCRIBED ORDER, EACH DECLARED WITH ITS REASON, AND BOTH
      FORCED BY BUNDLE CORRECTIONS THAT POST-DATE THE TRANSCRIPTION. Transcription fidelity is
      preserved everywhere it is still satisfiable; no step other than these two moves, and no
      step is dropped.
      DEVIATION 1 — `.gitignore` AFTER `pack add`. DD-8 lists the canonical `.gitignore` as step
      4 and `pack add` as step 5; this spec swaps them. DD-8 was transcribed when the ignore
      list was a fixed literal set with no pack dependency — the list DD-7's 2026-08-12
      correction has since struck. REQ-005 v1.1.0 made the entry set a function of the INSTALLED
      packs' declared `stdout_artifact` values, at which point emitting the file before any pack
      is installed became unsatisfiable: a `--pack` run on a fresh repo would write a
      `.gitignore` missing every pack-derived entry.
      DEVIATION 2 — DD-8 STEP 1'S SCAFFOLD MOVES FROM FIRST TO AFTER THE PACK INSTALL, AND ITS
      DEPENDENCY-INSTALL CLAUSE IS GONE FOR A DIFFERENT REASON. DD-8 step 1 reads "scaffold a
      minimal project with ≥1 source file + one test + minimal config … and install
      dependencies", and it is TWO obligations, not one. The INSTALL-DEPENDENCIES half is
      DROPPED, because bundle REQ-032 was retired: init installs no project dependencies and
      says so (REQ-011's denylist). The SCAFFOLD half SURVIVES — DD-7 states it with its own
      evidence and the 2026-08-12 correction reaffirmed it — and it MOVES, because DD-7 also
      requires that the scaffolded file "come from a pack recipe, never from core" (REQ-009): a
      recipe cannot be resolved out of a pack that is not installed yet, so the scaffold step is
      unsatisfiable at position 1 and lands immediately after the pack step. It sits BEFORE the
      toolchain step, and that placement is load-bearing rather than tidy: DD-7's motivating
      evidence is a compiler reding on an empty project ("no inputs", observed as `tsc`
      TS18003), and the toolchain step (REQ-011) is precisely the run that would hit it. A
      scaffold step ordered after the toolchain step would re-manufacture the exact failure
      DD-7 exists to prevent. An earlier version of this spec declared "one deviation" while
      DD-8 step 1 had in fact been dropped entirely and unreported; that misstatement is
      corrected here.
      THE ACCEPTANCE BAR is the outcome that sequence already achieved by hand, WITH REAL PACKS:
      on a fresh directory, `backstop init` installing REAL packs by git ref, followed by
      `backstop gate`, reaches PASS with zero violations, for BOTH the full-SDLC greenfield
      profile and the pack-only profile. A packless acceptance run does NOT satisfy this
      requirement: with zero packs installed the gate has no pack engines to dispatch and the
      toolchain step reports capability-absent, so its PASS asserts almost nothing about the
      sequence this requirement transcribes.
      THE ACCEPTANCE PACK MUST BE ONE WHOSE FINDINGS CAN REACH THE VERDICT UNDER BOTH PROFILES,
      AND THAT CONSTRAINT PICKS THE PACK. A prior version of this spec named `packs/contracts`
      and `packs/substantiveness` and was UNFALSIFIABLE, for two compounding reasons now
      corrected. (i) Those two packs declare ONLY `gate_type: contracts` and
      `gate_type: substantiveness` engines, and `excludeDedicatedStepRules`
      (`cmd/backstop/pack_gate.go:116-133`, called at `gate.go:828`) strips exactly those gate
      types out of the generic `pack_engines` dispatch, so their findings can reach the verdict
      ONLY through the `contract_signature` and `test_substantiveness` dimensions — the two the
      pack-only profile sets to `level: off` (REQ-003). Under that profile no finding from
      either pack could EVER change the outcome, so a fixture deliberately containing what their
      rules flag would still have PASSED. (ii) The contracts step dispatches PER CONTRACT ENTRY
      (`buildContractStep` → `gate.ExtractContractEntries` → the loop in
      `produceContractEngineResults`), and a freshly-initialized consumer repo has zero specs and
      therefore zero contract entries, so `packs/contracts` was never dispatched in EITHER
      acceptance run despite the spec claiming real dispatch.
      THE ACCEPTANCE RUNS THEREFORE INSTALL ONE PURPOSE-BUILT FIXTURE PACK SOURCE WHOSE ENGINE
      DECLARES `gate_type: lint`, published by the hermetic remote harness and installed by
      genuine git ref. A `lint` gate type has NO dedicated step (`gateTypeHasDedicatedStep`
      returns true for substantiveness, contracts and coverage ONLY), so its rules survive
      `excludeDedicatedStepRules` and dispatch through the generic `pack_engines` path — a
      dimension neither profile disables, since the pack-only profile turns off exactly the five
      SDLC dimensions and `pack_engines` is not among them. The same engine is therefore LIVE
      under both the full-SDLC and the pack-only acceptance run, which is what makes CLM-112's
      can-go-red half provable at all.
      THE HARNESS CONSTRAINT PRIOR DRAFTS ASSERTED IS FALSE AND IS WITHDRAWN. `newHermeticRemote`
      (`cmd/backstop/hermetic_remote_harness_test.go`) publishes an ARBITRARY directory as a
      tagged local repository; it is not limited to `packs/`, and SIX purpose-built fixture pack
      SOURCE trees already live under `cmd/backstop/testdata/hermetic-remote/` (`valid-pack`,
      `fixture-fail-pack`, `invalid-pack`, `scaffold-config-pack`, `divergent-name-pack`,
      `version-drift-pack`). The claim that "exactly three exist" counted only `packs/` and is
      struck. A TEST-FIXTURE pack source is not a vendored external pack, so publishing one
      violates no packs-are-always-external invariant — those six are the standing precedent.
      NONE OF THE SIX CAN SERVE, WHICH IS WHY A SEVENTH IS ADDED. Every one of the six declares
      its engine with `command: ""` — deliberately, as their own manifests say, so that nothing
      is ever executed through them and any future routing of execution fails loud rather than
      silently running a real binary. An engine with no command cannot be dispatched, so none of
      the six can produce a finding, and a PASS over them would assert nothing. Two are
      additionally unusable at install time: `invalid-pack` fails `pack check` and
      `fixture-fail-pack` fails `pack test`, and `pack add` runs both pipelines unconditionally,
      so init could not install either. The acceptance runs therefore install a NEW fixture pack
      source in that same directory, declaring one `gate_type: lint` engine with a REAL,
      allowlisted, hermetic command and one rule carrying a distinctive marker pattern that
      appears nowhere in a freshly-initialized project. Real manifest, real command, real
      generic dispatch, and a rule that demonstrably fires when its marker is present.
      ONE CONSEQUENCE IS STATED RATHER THAN HIDDEN, AND ONE SHORTFALL REMAINS. The fixture pack
      declares no `test` or `build` engine, so the acceptance runs' toolchain step reports
      capability-absent; REQ-011's execution path is proven by its own claims against manifests
      declaring those gate types, not by this run. And this is still not DD-8 step 5's five-pack
      hand-onboarding set: four of those five (toolchain, standards, secrets, and the external
      halves) live in their own repositories and install into gitignored `.backstop/packs/`, and
      REQ-018 forbids a local-path `--pack` value, so restoring the five-pack bar needs those
      packs installable by git ref from their own published repositories inside a hermetic test —
      a test-infrastructure capability this spec does not build. The difference is recorded here
      rather than silently narrowed.
  - id: REQ-033
    supports: onboarding-experience:REQ-033@1.0.0
    text: >
      DOCUMENTED GAP — NOT SATISFIED BY THIS SPEC. The pack-only profile turns
      `coverage_threshold` off (REQ-003), which forfeits coverage enforcement for a consumer
      whose packs still emit coverage records. The spec-independent coverage-floor knob that
      would replace it DOES NOT EXIST and has no owner: verified against
      `artifacts/backstop-yml/v1/schema.json`, `enforcement` declares
      `additionalProperties: false` over exactly `security`, `waiver_warning_days`,
      `semgrep_version`, `baseline_ttl`, `test_command`, `toolchain`, `policy` — so a
      `coverage.min_pct`-shaped key written by init would be REJECTED at config load. Init must
      therefore NOT invent that surface: the config it writes must validate against the shipped
      schema unchanged, and instead of wiring a floor the pack-only profile must REPORT the
      forfeiture as an unwired gap naming the absent capability, without failing the run. The
      full-SDLC profile emits no such notice.
  - id: REQ-035
    supports: onboarding-experience:REQ-035@1.0.0
    text: >
      Init's recipe steps must report a BROWNFIELD PRESERVE as a gap, not a success — but each
      must first CLASSIFY the preserve by its producer, because `preserveOrRegenerate`
      (`pkg/recipe/apply.go:348-390`) returns the same `PreservedDivergence` value for THREE
      materially different situations and only ONE of them is the brownfield gap. Treating all
      three alike would make init tell a consumer "no backstop gate was wired into this file"
      about a file where the gate demonstrably IS wired. The three PRODUCERS below are what the
      CODE does; what init can OBSERVE of them is a separate and coarser question, answered
      immediately after, and the two must not be conflated:
      (a) USER-OWNED (`apply.go:349-355`, `!own.adopted`) — no apply of this recipe ever
      produced the file, so the `create` family's never-clobber protection left the consumer's
      own file in place. `Rule` and `CoveringWaiver` are both EMPTY because nothing was
      adjudicated. NO backstop gate is wired into that file. THIS is the REQ-035 gap: init must
      name every such file, state in words that no gate was wired into it, give the consumer
      the next action, and exit non-zero — DD-15's "on 'I cannot tell', REFUSE" posture governs,
      because silent success here is a broken promise rather than un-adopted capability.
      (b) ONE-SHOT ALREADY MATERIALIZED (`apply.go:357-363`, reached ONLY when `own.adopted` is
      TRUE and `own.kind == KindTemplating`) — a `kind: templating` recipe's output carries no
      regeneration obligation, so there is no divergence to account for. `Rule` and
      `CoveringWaiver` are ALSO empty here, so the field pair does not separate (b) from (a).
      (c) COVERED BY AN ACTIVE WAIVER (`apply.go:377-388`) — recipe-owned output the consumer
      legitimately customized and accounted for with a valid `@waiver` token. `Rule` and
      `CoveringWaiver` are both POPULATED. The gate IS wired and the customization is
      accountable: init reports it naming the rule and the covering token, must NOT say no gate
      was wired, does not report it as a gap, and does not fail the run.
      WHAT INIT CAN ACTUALLY OBSERVE, AND WHERE THE OBSERVATION RUNS OUT. The declared recipe
      `kind` does NOT separate (a) from (b), and any statement that it does is false: the branch
      order at `apply.go:349` tests `!own.adopted` FIRST and returns immediately, so the kind
      test at `:357` is unreachable for a recipe no prior apply adopted. A `kind: templating`
      recipe that was never adopted therefore takes branch (a) and returns a value BYTE-IDENTICAL
      to branch (b)'s — `Path` set, `Rule` and `CoveringWaiver` empty — and the adoption bit that
      separates them is not carried by anything the apply returns. What init holds is exactly two
      observables: the `Rule`/`CoveringWaiver` pair per preserve, and the resolved recipe's
      declared `kind`. They yield THREE observable classes, and the third is an admitted
      unknown:
      WAIVER-COVERED — pair POPULATED, at ANY declared kind. Unambiguously producer (c): report
      the accountable customization naming the rule and the covering token, no gap, exit 0.
      USER-OWNED — pair EMPTY and declared kind is `scaffolding` or `implementing`. Unambiguously
      producer (a), because branch (b) is unreachable for those kinds. This is the REQ-035 gap:
      init names every such file, states in words that no backstop gate was wired into it, gives
      the consumer the next action, and exits non-zero.
      INDETERMINATE — pair EMPTY and declared kind is `templating`. Producer (a) and producer (b)
      are INDISTINGUISHABLE here from what init can see. Init must not resolve this by guessing
      in either direction: claiming success would hide a real brownfield gap, and asserting "no
      backstop gate was wired into this file" would be a false positive assertion about a
      one-shot that already materialized — the same species of false report REQ-035 exists to
      prevent, pointed the other way. DD-15's "on 'I cannot tell', REFUSE" posture governs, so
      init scores it CONSERVATIVELY as a gap — it names the file, reports that it CANNOT
      DETERMINE whether the recipe's output is present in it, gives the next action, and exits
      non-zero — while using NO "no gate was wired" language, because that is the half init
      cannot know. A gap it names honestly is the cost of an ambiguity core cannot resolve.
      INIT MUST NOT WIDEN THE SEAM TO CARRY ADOPTION STATE. `preserveOrRegenerate`'s adoption bit
      is recipe-level (`apply.go:166`) and reaches it through `adoptionKey`, which is UNEXPORTED;
      reconstructing it in init would be a second derivation of a recipe's adoption identity —
      the same "one authority, not a copy" hazard REQ-018 refuses for local-path classification —
      and surfacing the applier's own bit would require editing `pkg/recipe`, which REQ-009
      forbids. The ambiguity is therefore RECORDED, not engineered around. If a later artifact
      makes prior-adoption state available at this seam without either cost, the INDETERMINATE
      class collapses into (a) and (b) and this requirement should be revised then.
      Init must therefore never use "no gate was wired" language for a WAIVER-COVERED or an
      INDETERMINATE preserve; that sentence belongs to the USER-OWNED class alone. A USER-OWNED
      or INDETERMINATE preserve occurring alongside successfully written files is still reported
      as a gap; only an apply whose every preserve is WAIVER-COVERED (or which preserves nothing
      at all) is reported as success.
      ONE CLASSIFIER, EVERY RECIPE APPLY INIT PERFORMS. This requirement governs the CI step
      (REQ-016/REQ-017) and the SCAFFOLD step (REQ-009) identically — both go through the same
      `recipe.Apply` and both can return preserves — and the init source set must contain exactly
      one implementation of the classification above. A second, step-local classifier is the
      "one authority, not a copy" hazard this spec refuses everywhere else. What differs between
      the two steps is the SENTENCE, not the class, because the CI step's wording is a statement
      about a gate and the scaffold step's is not: for a USER-OWNED preserve the CI step states
      that no backstop gate was wired into that file, and the scaffold step states that the
      consumer's own file was left in place and the recipe's declared source file was therefore
      NOT written. Both name every such file, give the consumer a next action, and exit
      non-zero; the scaffold step must NEVER borrow the "no gate was wired" sentence, which
      would assert something about CI wiring the scaffold step knows nothing about. An
      INDETERMINATE scaffold preserve is likewise a gap that exits non-zero and reports that
      init cannot determine whether the recipe's output is present, and a WAIVER-COVERED
      scaffold preserve is an accountable customization: no gap, exit 0, naming the rule and the
      covering token.
      THE SEAM MUST CARRY THE DISCRIMINATORS. Init's `RecipeApplier` seam must surface the
      applier's `[]recipe.PreservedDivergence` values with their `Rule` and `CoveringWaiver`
      fields intact, plus the resolved recipe's declared `kind` — a `[]string` of paths
      discards exactly the fields the classification needs. VERIFIED FEASIBLE WITHOUT TOUCHING
      `pkg/recipe`: `recipe.PreservedDivergence` (with both fields), `recipe.ApplyResult.Preserved`,
      and `recipe.ResolvedRecipe.Manifest.Kind` are ALL already exported at HEAD, so this is a
      widening of this spec's OWN seam type and requires no change to the recipe mechanism,
      whose files this spec must not edit (REQ-009).

claims:
  # ── REQ-001 — one command, no manual step in between ──
  - id: CLM-001
    requirement: REQ-001
    subject: cmd/backstop
    text: >
      A single `backstop init` invocation in an empty directory leaves `backstop.yml`, the
      `.backstop/` artifact layout, and a `.gitignore` on disk and prints one report — no
      second command is required to reach that state
    tests:
      - TestInit_SingleInvocationReachesFirstValue
  - id: CLM-002
    requirement: REQ-001
    text: >
      The printed report names EVERY capability in the resolved set with an outcome for each,
      so no step is silently absent from the account of what init did
    tests:
      - TestInit_ReportNamesEveryResolvedCapabilityWithAnOutcome
  # ── REQ-002 — omakase default, subtraction-only narrowing, headless parity ──
  - id: CLM-003
    requirement: REQ-002
    text: >
      Bare init resolves EXACTLY the seven default capabilities — git, sdlc, gitignore, packs,
      toolchain, baseline, observe — and runs all seven
    tests:
      - TestInit_BareRunResolvesAllSevenDefaultCapabilities
  - id: CLM-004
    requirement: REQ-002
    text: >
      `--no-git` removes the git capability and leaves the other six resolved and executed
    tests:
      - TestInit_NoGitSubtractsOnlyTheGitCapability
  - id: CLM-005
    requirement: REQ-002
    text: >
      `--no-sdlc` removes the sdlc capability and leaves the other six resolved and executed
    tests:
      - TestInit_NoSdlcSubtractsOnlyTheSdlcCapability
  - id: CLM-006
    requirement: REQ-002
    text: >
      `--no-gitignore` removes the gitignore capability and leaves the other six resolved and
      executed
    tests:
      - TestInit_NoGitignoreSubtractsOnlyTheGitignoreCapability
  - id: CLM-007
    requirement: REQ-002
    text: >
      `--no-packs` removes the packs capability and leaves the other six resolved and executed
    tests:
      - TestInit_NoPacksSubtractsOnlyThePacksCapability
  - id: CLM-008
    requirement: REQ-002
    text: >
      `--no-toolchain` removes the toolchain capability and leaves the other six resolved and
      executed
    tests:
      - TestInit_NoToolchainSubtractsOnlyTheToolchainCapability
  - id: CLM-009
    requirement: REQ-002
    text: >
      `--no-baseline` removes the baseline capability and leaves the other six resolved and
      executed
    tests:
      - TestInit_NoBaselineSubtractsOnlyTheBaselineCapability
  - id: CLM-010
    requirement: REQ-002
    text: >
      `--no-observe` removes the observe capability and leaves the other six resolved and
      executed
    tests:
      - TestInit_NoObserveSubtractsOnlyTheObserveCapability
  - id: CLM-011
    requirement: REQ-002
    text: >
      `--only` narrows the set to exactly the capabilities named and to no others, and is
      repeatable so several may be named
    tests:
      - TestInit_OnlyNarrowsToExactlyTheNamedCapabilities
  - id: CLM-012
    requirement: REQ-002
    subject: cmd/backstop
    text: >
      Combining `--only` and any `--no-<cap>` in one invocation is a config error exiting 2,
      because the two express contradictory intents about the same set
    tests:
      - TestInit_OnlyAndNoFlagsCombinedIsAConfigError
  - id: CLM-013
    requirement: REQ-002
    subject: cmd/backstop
    text: >
      An unrecognized capability name supplied to `--only` or `--no-` is a config error
      exiting 2 whose message lists the seven valid capability names
    tests:
      - TestInit_UnknownCapabilityNameIsAConfigErrorNamingTheValidSet
  - id: CLM-014
    requirement: REQ-002
    subject: cmd/backstop
    text: >
      Init never reads standard input: with stdin closed and no TTY the run produces the same
      files, the same report, and the same exit code as an interactive run
    tests:
      - TestInit_HeadlessRunIsIdenticalToInteractiveRun
  # ── REQ-003 — profile-correct backstop.yml (the five-dimension matrix) ──
  - id: CLM-015
    requirement: REQ-003
    text: >
      The full-SDLC profile's generated config carries `project:` and `artifact_root:
      .backstop` and NO other top-level key, and sets NONE of the five SDLC dimensions to
      `level: off`. EXPECTED RED UNTIL SPEC-068 LANDS — see CLM-021
    tests:
      - TestInit_FullSdlcConfigLeavesAllFiveSdlcDimensionsEnforced
  - id: CLM-016
    requirement: REQ-003
    text: >
      The pack-only profile writes `enforcement.policy.test_verification.level: off`
    tests:
      - TestInit_PackOnlyDisablesTestVerification
  - id: CLM-017
    requirement: REQ-003
    text: >
      The pack-only profile writes `enforcement.policy.coverage_threshold.level: off`
    tests:
      - TestInit_PackOnlyDisablesCoverageThreshold
  - id: CLM-018
    requirement: REQ-003
    text: >
      The pack-only profile writes `enforcement.policy.contract_signature.level: off`
    tests:
      - TestInit_PackOnlyDisablesContractSignature
  - id: CLM-019
    requirement: REQ-003
    text: >
      The pack-only profile writes `enforcement.policy.test_substantiveness.level: off`
    tests:
      - TestInit_PackOnlyDisablesTestSubstantiveness
  - id: CLM-020
    requirement: REQ-003
    text: >
      The pack-only profile writes `enforcement.policy.artifact_status_drift.level: off`
    tests:
      - TestInit_PackOnlyDisablesArtifactStatusDrift
  - id: CLM-021
    requirement: REQ-003
    text: >
      The full-SDLC config init writes loads cleanly through `config.Load` and passes the
      shipped backstop.yml JSON-schema pass. EXPECTED RED UNTIL SPEC-068 LANDS: the config
      carries `artifact_root`, which TODAY is rejected TWICE — by `config.Load`'s strict
      typed decode and by the schema's `additionalProperties: false` over the six declared
      top-level properties. A `config.Load` failure here is the stated prerequisite, NOT an
      implementer bug
    tests:
      - TestInit_FullSdlcConfigRoundTripsThroughConfigLoad
  - id: CLM-022
    requirement: REQ-003
    text: >
      The pack-only config init writes loads cleanly through `config.Load` and passes the
      shipped backstop.yml JSON-schema pass. EXPECTED RED UNTIL SPEC-068 LANDS for the same
      reason as CLM-021 — the pack-only profile writes no `artifact_root`, but this claim is
      asserted against the SAME shipped loader and schema pass the full-SDLC claim uses, so it
      is listed here for symmetry of diagnosis rather than left to look like a distinct failure
    tests:
      - TestInit_PackOnlyConfigRoundTripsThroughConfigLoad
  - id: CLM-023
    requirement: REQ-003
    kind: absence
    text: >
      DENYLIST — neither profile's generated config contains the retired `language:` key,
      which SPEC-046 removed and which would bake a language into the first file a consumer
      sees
    tests:
      - TestInit_GeneratedConfigNeverWritesTheRetiredLanguageKey
  - id: CLM-024
    requirement: REQ-003
    text: >
      The `project:` value equals the target directory's basename, and is unchanged when the
      directory is populated with differing project-identity files — nothing in the project is
      read to name it
    tests:
      - TestInit_ProjectNameComesFromDirectoryBasenameOnly
  # ── REQ-004 — the .backstop/-rooted layout ──
  - id: CLM-025
    requirement: REQ-004
    text: >
      The full-SDLC profile creates all six artifact directories under `.backstop/` —
      bundles, specs, plans, issues, adrs, directives
    tests:
      - TestInit_FullSdlcScaffoldsAllSixArtifactDirectoriesUnderBackstop
  - id: CLM-026
    requirement: REQ-004
    text: >
      The full-SDLC profile writes `artifact_root: .backstop`, so discovery resolves the layout
      init just created rather than the repo root. EXPECTED RED UNTIL SPEC-068 LANDS — the key
      does not exist in the schema or the typed loader yet (same prerequisite as CLM-021)
    tests:
      - TestInit_FullSdlcWritesArtifactRootPointingAtBackstopDir
  - id: CLM-027
    requirement: REQ-004
    text: >
      The pack-only profile creates no artifact directory anywhere and writes no artifact-root
      key
    tests:
      - TestInit_PackOnlyScaffoldsNoArtifactDirectories
  - id: CLM-028
    requirement: REQ-004
    kind: absence
    text: >
      DENYLIST — init creates no root-level `specs/`, `bundles/`, `plans/`, `issues/`,
      `adrs/`, or `directives/` directory in a consumer repo under either profile
    tests:
      - TestInit_NeverCreatesRootLevelArtifactDirectories
  - id: CLM-104
    requirement: REQ-004
    kind: absence
    text: >
      DENYLIST — init creates no `capabilities/` directory under either profile or at either
      location (`.backstop/capabilities/` or root). The seventh kind SPEC-068's layout table
      covers is deliberately not scaffolded, and this claim makes that a decision the corpus
      records rather than an omission a later reader "fixes"
    tests:
      - TestInit_ScaffoldsNoCapabilitiesDirectory
  - id: CLM-103
    requirement: REQ-004
    subject: cmd/backstop
    text: >
      NON-VACUITY GUARD — an artifact placed under the `.backstop/` root init created is
      DISCOVERED and validated, not skipped: discovery reports a non-zero artifact count and
      a deliberately invalid artifact placed there makes validation FAIL. The layout init
      scaffolds must not be invisible to the tooling that gates it, and a PASS over zero
      discovered artifacts must not be reachable from a correctly-initialized repo.
      EXPECTED RED UNTIL SPEC-068 LANDS — this is the deliberate tripwire, red for a DIFFERENT
      reason than CLM-021/022/026: those fail at config load on the `artifact_root` key, this
      one fails at DISCOVERY, because `.backstop` sits on the unconditional skip list at
      `cmd/backstop/artifact_discover.go:47-49`
    tests:
      - TestInit_ArtifactsUnderTheScaffoldedRootAreDiscoveredAndGated
  # ── REQ-005 — the canonical .gitignore ──
  - id: CLM-029
    requirement: REQ-005
    text: >
      The emitted `.gitignore` contains all three backstop-owned entries — `.backstop/packs/`,
      `.backstop/baseline.json`, `.backstop/pack-config-provenance.json`
    tests:
      - TestInit_GitignoreCarriesTheThreeBackstopOwnedEntries
  - id: CLM-030
    requirement: REQ-005
    text: >
      For each installed pack, every engine's declared `stdout_artifact` path appears as a
      `.gitignore` entry, sourced from the pack manifest rather than from any core list
    tests:
      - TestInit_GitignoreIncludesEveryInstalledPackEngineStdoutArtifact
  - id: CLM-031
    requirement: REQ-005
    text: >
      A pack whose engines declare no `stdout_artifact` contributes no entry — init does not
      invent a path for it
    tests:
      - TestInit_PackWithoutStdoutArtifactContributesNoGitignoreEntry
  - id: CLM-032
    requirement: REQ-005
    kind: absence
    text: >
      DENYLIST — core contributes no language-, framework-, or tool-specific ignore path; the
      entry set is exactly the three backstop literals plus pack-declared values
    tests:
      - TestInit_GitignoreHoldsNoToolOrLanguageSpecificPathFromCore
  - id: CLM-033
    requirement: REQ-005
    text: >
      Init's report states the ACCEPTED RESIDUE — that incidentally generated toolchain paths
      are not covered by `stdout_artifact` and remain the consumer's to ignore
    tests:
      - TestInit_ReportStatesTheUncoveredGitignoreResidue
  - id: CLM-034
    requirement: REQ-005
    text: >
      An existing `.gitignore` survives byte-for-byte and only the missing entries are
      appended — no pre-existing line is rewritten, reordered, or dropped
    tests:
      - TestInit_ExistingGitignoreIsAppendedToNeverRewritten
  - id: CLM-035
    requirement: REQ-005
    text: >
      An entry already present in the consumer's `.gitignore` is not appended a second time
    tests:
      - TestInit_GitignoreEntryAlreadyPresentIsNotDuplicated
  # ── REQ-006 — git init only when there is no .git ──
  - id: CLM-036
    requirement: REQ-006
    text: >
      In a directory with no `.git`, init creates a git repository
    tests:
      - TestInit_CreatesGitRepositoryWhenNoneExists
  - id: CLM-037
    requirement: REQ-006
    text: >
      In a directory that is already a git repository, init runs no git initialization and the
      existing `.git` directory's HEAD, config, and refs are unchanged
    tests:
      - TestInit_ExistingGitRepositoryIsLeftUntouched
  # ── REQ-007 — converge, never clobber, framework-blind ──
  - id: CLM-038
    requirement: REQ-007
    text: >
      An existing `backstop.yml` is preserved byte-for-byte and reported as already present
      rather than overwritten
    tests:
      - TestInit_ExistingBackstopYmlIsPreservedNotOverwritten
  - id: CLM-039
    requirement: REQ-007
    text: >
      When some artifact directories already exist, init creates only the missing ones and
      leaves the existing ones and their contents untouched
    tests:
      - TestInit_CreatesOnlyTheMissingArtifactDirectories
  - id: CLM-040
    requirement: REQ-007
    text: >
      No consumer-owned file is overwritten by any init step: every file present before the
      run and not owned by backstop is byte-identical after it
    tests:
      - TestInit_OverwritesNoPreExistingConsumerFile
  - id: CLM-041
    requirement: REQ-007
    text: >
      A second init run immediately after a first writes no new file, changes no existing byte,
      and reports every step as converged
    tests:
      - TestInit_SecondRunConvergesAndWritesNothing
  - id: CLM-042
    requirement: REQ-007
    kind: absence
    text: >
      DENYLIST — the facts init inspects are exactly `.git`, `backstop.yml`, the artifact
      directories, and `.gitignore`; no other path in the project is read to decide what to do
    tests:
      - TestInit_ReadsOnlyBackstopNeutralFactsToDecideWhatToDo
  - id: CLM-043
    requirement: REQ-007
    text: >
      Init's report is phrased purely in backstop vocabulary — capability names, gate
      dimensions, artifact paths — and names no ecosystem, language, or framework
    tests:
      - TestInit_ReportUsesOnlyBackstopVocabulary
  # ── REQ-008 — the zero-detection invariant ──
  - id: CLM-044
    requirement: REQ-008
    kind: absence
    text: >
      DENYLIST — a structural scan of the init source set (`cmd/backstop/init*.go` and
      `pkg/initialize/**`, excluding `_test.go`) finds no language name, framework name,
      ecosystem-manifest filename, or CI-platform name
    tests:
      - TestInit_SourceSetHoldsNoLanguageFrameworkOrPlatformLiteral
  - id: CLM-045
    requirement: REQ-008
    text: >
      Two fixture projects differing only in which ecosystem marker files they contain produce
      identical init reports, identical written files, and identical exit codes
    tests:
      - TestInit_BehaviorIsIdenticalAcrossDifferingEcosystemMarkers
  # ── REQ-009 — recipes applied only through the shipped generic mechanism ──
  - id: CLM-046
    requirement: REQ-009
    text: >
      Init applies a recipe only by calling the shipped resolve+apply path, and the target
      files written are exactly the ones the recipe declared — init contributes no path
    tests:
      - TestInit_AppliesRecipesOnlyThroughTheShippedResolveApplyPath
  - id: CLM-047
    requirement: REQ-009
    text: >
      A recipe payload containing arbitrary bytes lands byte-identical on disk; init performs
      no rendering, rewriting, or specialization of payload content
    tests:
      - TestInit_RecipePayloadLandsByteIdentical
  - id: CLM-048
    requirement: REQ-009
    kind: absence
    text: >
      DENYLIST — init AUTHORS no source file: every file init produces is either a
      backstop-owned artifact (config, layout, gitignore) or a recipe-declared target written
      by the shipped apply path, and the init source set holds no source-file payload, no
      template, no filename or extension literal, and no path for one. CORRECTED — an earlier
      wording ("init writes no source file of its own") read as a denial that init produces a
      first source file AT ALL, which contradicts bundle REQ-009 and DD-7; the denylist is on
      core AUTHORING the content, never on init DELIVERING it through a pack recipe
    tests:
      - TestInit_AuthorsNoSourceFileContentOfItsOwn
  # ── REQ-009 (cont.) — the scaffold step: DD-7's source file, delivered by recipe ──
  - id: CLM-126
    requirement: REQ-009
    text: >
      The `--scaffold` value is handed to the resolve+apply path byte-identical to what the
      consumer typed — no trimming, completing, defaulting, or normalization — exactly as
      CLM-072 requires of `--ci`
    tests:
      - TestInit_ScaffoldRefIsPassedThroughVerbatim
  - id: CLM-127
    requirement: REQ-009
    text: >
      With `--scaffold` omitted, no recipe resolution or apply is attempted for the scaffold
      step at all
    tests:
      - TestInit_ScaffoldOmittedAttemptsNoRecipeResolution
  - id: CLM-128
    requirement: REQ-009
    text: >
      With `--scaffold` omitted, init states that no source file was scaffolded and names
      `backstop recipe apply` plus the pinned `<pack>:<recipe>@<version>` ref shape as the way
      to do it later — the honest skip, not a silent absence
    tests:
      - TestInit_ScaffoldOmittedReportsTheSkipAndHowToDoItLater
  - id: CLM-129
    requirement: REQ-009
    subject: cmd/backstop
    text: >
      With `--scaffold` omitted and every other requested step delivered, init exits 0 — a
      skipped optional step is not an error, and not every pack ecosystem ships a scaffold
      recipe
    tests:
      - TestInit_ScaffoldOmittedIsADeliberateNoOpExitingZero
  - id: CLM-130
    requirement: REQ-009
    text: >
      With `--scaffold` supplied and resolvable, the recipe's declared targets land on disk at
      the paths the RECIPE declared, byte-identical to the payload the pack declared — init
      contributes no path and no byte
    tests:
      - TestInit_ScaffoldRecipeTargetsLandAtTheRecipeDeclaredPaths
  - id: CLM-131
    requirement: REQ-009
    kind: absence
    text: >
      DENYLIST — the init source set holds nothing that could author or name a scaffolded source
      file: no pack name, recipe id or version literal for the scaffold ref, and no source-file
      payload, template body, filename, or file-extension literal anywhere. Core constructs no
      part of the ref and no part of the file
    tests:
      - TestInit_SourceSetConstructsNoPartOfTheScaffoldRefOrItsPayload
  - id: CLM-132
    requirement: REQ-009
    text: >
      `scaffold` is NOT a capability name — `ResolveCapabilities` rejects it as unrecognized,
      exactly as it rejects any other eighth name, so the vocabulary stays the seven CLM-003
      pins and the scaffold step is governed solely by the flag's presence
    tests:
      - TestInit_ScaffoldIsNotACapabilityName
  - id: CLM-133
    requirement: REQ-009
    kind: absence
    subject: cmd/backstop
    text: >
      DENYLIST — no `--no-scaffold` flag exists and no `scaffold` verb or subcommand is added to
      the command tree; `--scaffold` is a flag on init and nothing else, mirroring CLM-077
    tests:
      - TestInit_AddsNoScaffoldVerbOrNegationFlag
  - id: CLM-134
    requirement: REQ-009
    text: >
      A `--scaffold` ref that cannot be resolved surfaces the SHIPPED error VERBATIM attributed
      to the SCAFFOLD step — init adds no guidance of its own and classifies no resolve failure
      differently — and every other init step still completes and is reported with its own
      outcome
    tests:
      - TestInit_UnresolvableScaffoldRefSurfacesTheResolveErrorVerbatim
  - id: CLM-135
    requirement: REQ-009
    subject: cmd/backstop
    text: >
      A `--scaffold` ref that cannot be resolved makes init exit non-zero: the consumer asked
      for a source file and init did not deliver one, which is a broken promise rather than an
      un-adopted capability
    tests:
      - TestInit_UnresolvableScaffoldRefExitsNonZero
  - id: CLM-136
    requirement: REQ-009
    kind: absence
    text: >
      DENYLIST — init supplies NO recipe param to either apply: the init source set constructs
      no param map, and no value is derived from the target directory's name or from anything
      init found on disk. A recipe declaring a param required with no default therefore fails
      through the shipped apply's own error rather than through an init-invented value
    tests:
      - TestInit_SuppliesNoRecipeParamToEitherApply
  - id: CLM-137
    requirement: REQ-009
    text: >
      A `--scaffold` ref naming a pack that appears nowhere in core is dispatched normally,
      because core holds no allowlist of acceptable pack names — the same property CLM-078
      asserts for `--ci`
    tests:
      - TestInit_ArbitraryPackNameInTheScaffoldRefIsDispatchedNormally
  # ── REQ-010 — packs enter only by explicit consumer act ──
  - id: CLM-049
    requirement: REQ-010
    text: >
      Bare init with no `--pack` installs zero packs and reports the skip, naming
      `backstop pack add` as the way to add them
    tests:
      - TestInit_WithoutPackFlagInstallsNothingAndSaysSo
  - id: CLM-050
    requirement: REQ-010
    text: >
      `--pack` is repeatable and installs exactly the refs supplied, in the order supplied, and
      no others
    tests:
      - TestInit_InstallsExactlyTheSuppliedPackRefsInOrder
  - id: CLM-051
    requirement: REQ-010
    kind: absence
    text: >
      DENYLIST — the init source set contains no pack-name literal and no default pack roster,
      so there is nothing for init to install absent a consumer-supplied ref
    tests:
      - TestInit_SourceSetHoldsNoPackNameLiteralOrDefaultRoster
  - id: CLM-052
    requirement: REQ-010
    text: >
      Two fixture projects with different on-disk contents and the same `--pack` arguments
      install the identical pack set — nothing on disk influences pack selection
    tests:
      - TestInit_PackSelectionIsUnaffectedByProjectContents
  # ── REQ-011 — the toolchain-execution check (the gate_type matrix) ──
  - id: CLM-053
    requirement: REQ-011
    subject: cmd/backstop
    text: >
      An installed pack declaring an engine with `gate_type: test` has that engine's command
      executed exactly once, and a zero exit passes the step
    tests:
      - TestInit_ExecutesDeclaredTestEntrypointOnceAndPassesOnZeroExit
  - id: CLM-054
    requirement: REQ-011
    subject: cmd/backstop
    text: >
      An installed pack declaring an engine with `gate_type: build` has that engine's command
      executed exactly once, and a zero exit passes the step
    tests:
      - TestInit_ExecutesDeclaredBuildEntrypointOnceAndPassesOnZeroExit
  - id: CLM-055
    requirement: REQ-011
    subject: cmd/backstop
    text: >
      An engine declared `gate_type: lint` is NOT executed by the toolchain step
    tests:
      - TestInit_LintGateTypeEngineIsNotExecuted
  - id: CLM-118
    requirement: REQ-011
    subject: cmd/backstop
    text: >
      An engine declared `gate_type: findings` is NOT executed by the toolchain step
    tests:
      - TestInit_FindingsGateTypeEngineIsNotExecuted
  - id: CLM-119
    requirement: REQ-011
    subject: cmd/backstop
    text: >
      An engine declared `gate_type: coverage` is NOT executed by the toolchain step
    tests:
      - TestInit_CoverageGateTypeEngineIsNotExecuted
  - id: CLM-120
    requirement: REQ-011
    subject: cmd/backstop
    text: >
      An engine declared `gate_type: substantiveness` is NOT executed by the toolchain step
    tests:
      - TestInit_SubstantivenessGateTypeEngineIsNotExecuted
  - id: CLM-121
    requirement: REQ-011
    subject: cmd/backstop
    text: >
      An engine declared `gate_type: contracts` is NOT executed by the toolchain step
    tests:
      - TestInit_ContractsGateTypeEngineIsNotExecuted
  - id: CLM-056
    requirement: REQ-011
    subject: cmd/backstop
    text: >
      CASE (b) — CANNOT BE STARTED AT ALL. A declared entrypoint whose executable cannot be
      executed (absent, or not executable) is reported as an owed SETUP step naming the pack
      whose entrypoint could not run and pointing at that pack's own documented install steps,
      and init exits non-zero
    tests:
      - TestInit_UnstartableEntrypointIsReportedAsOwedSetupAndExitsNonZero
  - id: CLM-105
    requirement: REQ-011
    subject: cmd/backstop
    text: >
      CASE (c) — STARTED AND EXITED NON-ZERO. A declared entrypoint that RAN and exited
      non-zero is reported with its exit code and its captured output VERBATIM, attributed to
      the pack and the command, and init exits non-zero. VERBATIM INCLUDES STDERR: an entrypoint
      that writes its diagnostics ONLY to stderr and nothing to stdout has those bytes present
      in the report, which is what fails if the implementation reaches the runner through
      `RunStdout` instead of `Run`
    tests:
      - TestInit_NonZeroEntrypointExitIsReportedVerbatimWithItsExitCode
  - id: CLM-122
    requirement: REQ-011
    subject: cmd/backstop
    text: >
      CASE (c), THE CAPTURE-METHOD TRIPWIRE — an entrypoint whose output goes ENTIRELY to
      stderr and whose stdout is empty still produces a non-empty captured-output report,
      because init enters the shared runner through `check.CommandRunner.Run` (combined
      output) rather than `RunStdout` (stdout only). An implementation that copies
      `runFindingsEngine`'s method choice along with its three steps fails this claim
    tests:
      - TestInit_CapturedOutputIncludesStderrOnlyDiagnostics
  - id: CLM-106
    requirement: REQ-011
    kind: absence
    subject: cmd/backstop
    text: >
      DENYLIST — a case-(c) report claims NO cause: it does not use the owed-setup label, does
      not name dependencies or a package manager, and attributes the failure to nothing beyond
      the pack and the command it ran. This is the exit-code cause-inference bundle REQ-011
      forbids, and it is the pnpm `ERR_PNPM_IGNORED_BUILDS` misdiagnosis made unrepresentable
    tests:
      - TestInit_NonZeroEntrypointExitClaimsNoCause
  - id: CLM-057
    requirement: REQ-011
    subject: cmd/backstop
    text: >
      Entrypoint outcomes are INDEPENDENT: a pack declaring two entrypoint engines, one exiting
      non-zero and one exiting zero, produces two separately-reported outcomes and the failing
      one does not turn the passing one's outcome into a failure — each engine's verdict is its
      own command's exit status and nothing adjacent
    tests:
      - TestInit_EachEntrypointOutcomeIsIndependentOfTheOthers
  - id: CLM-107
    requirement: REQ-011
    subject: cmd/backstop
    text: >
      An entrypoint whose declared tool the trusted-tool allowlist refuses is NOT executed at
      all: `checkEngineToolAllowed` runs BEFORE `splitCommand` and before the runner, and its
      refusal surfaces as a config error rather than as a toolchain pass or fail
    tests:
      - TestInit_UnallowlistedEntrypointToolIsRefusedBeforeExecution
  - id: CLM-108
    requirement: REQ-011
    subject: cmd/backstop
    text: >
      An entrypoint command is tokenized by the shared `splitCommand` and handed to the shared
      `check.CommandRunner` as an argv, so a command whose text contains shell metacharacters
      is passed through as literal arguments and no shell interprets it
    tests:
      - TestInit_EntrypointRunsAsArgvThroughTheSharedRunnerNeverAShell
  - id: CLM-109
    requirement: REQ-011
    kind: absence
    subject: cmd/backstop
    text: >
      DENYLIST — init's toolchain path introduces no second execution route: the init source
      set contains no `exec.Command` construction, no shell invocation, and no allowlist or
      command-splitting logic of its own; it reaches the runner only through the same
      `checkEngineToolAllowed` → `splitCommand` → `check.CommandRunner` sequence
      `runFindingsEngine` takes, differing from it in the CAPTURE METHOD alone (`Run` rather
      than `RunStdout`, CLM-122) and in nothing else
    tests:
      - TestInit_SourceSetHoldsNoSecondCommandExecutionPath
  - id: CLM-058
    requirement: REQ-011
    subject: cmd/backstop
    text: >
      When no installed pack declares a `test` or `build` engine, the toolchain step reports
      capability-absent and init does not exit non-zero for it
    tests:
      - TestInit_NoDeclaredEntrypointReportsCapabilityAbsentWithoutFailing
  - id: CLM-059
    requirement: REQ-011
    kind: absence
    subject: cmd/backstop
    text: >
      DENYLIST — init runs no package-manager or dependency-installation command and writes
      nothing into the project beyond its own artifacts and recipe-declared targets
    tests:
      - TestInit_InstallsNoProjectDependencies
  # ── REQ-012 — baseline seeding is CONSUMED from ISSUE-056 ──
  - id: CLM-060
    requirement: REQ-012
    kind: absence
    text: >
      DENYLIST — nothing in the init source set writes `.backstop/baseline.json` or computes a
      baseline fingerprint; the seeding machinery is ISSUE-056's, not this spec's
    tests:
      - TestInit_ImplementsNoBaselineSeedingMachinery
  - id: CLM-061
    requirement: REQ-012
    text: >
      When a seeding implementation is supplied to the delegation seam, init invokes it exactly
      once and reports what it returned
    tests:
      - TestInit_DelegatesBaselineSeedingExactlyOnce
  - id: CLM-062
    requirement: REQ-012
    text: >
      When no seeding implementation is available, the baseline step reports the gap naming
      ISSUE-056 as its owner and init does NOT exit non-zero for it — an un-adopted capability
      is not a broken promise
    tests:
      - TestInit_AbsentBaselineSeederIsReportedAsAGapWithoutFailing
  # ── REQ-013 — the remoteless gate message is CONSUMED from ISSUE-056 ──
  - id: CLM-063
    requirement: REQ-013
    kind: absence
    text: >
      DENYLIST — this spec's implementation changes no file under `pkg/gate`, and init neither
      rewrites, suppresses, nor substitutes for the remoteless `baseline_comparison` message
    tests:
      - TestInit_ChangesNoGatePackageFileAndDoesNotMaskTheRemotelessMessage
  # ── REQ-014 — observation framing and exit-code precedence ──
  - id: CLM-064
    requirement: REQ-014
    text: >
      Gate findings are grouped by gate dimension with a count per dimension, so the report
      presents structure rather than a wall of individual findings
    tests:
      - TestInit_GateFindingsAreGroupedByDimensionWithCounts
  - id: CLM-065
    requirement: REQ-014
    text: >
      The observation summary is phrased as what was noticed and carries no failure or
      violation verdict language
    tests:
      - TestInit_ObservationSummaryCarriesNoVerdictLanguage
  - id: CLM-066
    requirement: REQ-014
    subject: cmd/backstop
    text: >
      A run whose only negative signal is pre-existing gate findings exits 0 — inherited
      violations are not an init failure
    tests:
      - TestInit_PreExistingFindingsAloneExitZero
  - id: CLM-067
    requirement: REQ-014
    subject: cmd/backstop
    text: >
      A run whose gate is clean and whose every requested step delivered exits 0
    tests:
      - TestInit_CleanGateAndAllStepsDeliveredExitZero
  - id: CLM-068
    requirement: REQ-014
    subject: cmd/backstop
    text: >
      A run with BOTH pre-existing findings and a REQ-035 USER-OWNED brownfield preserve
      exits non-zero, and both facts appear in full in the report
    tests:
      - TestInit_BrownfieldPreserveWinsOverObservationOnTheExitCode
  - id: CLM-069
    requirement: REQ-014
    subject: cmd/backstop
    text: >
      A run with BOTH pre-existing findings and a REQ-017 CI resolve failure exits non-zero,
      and both facts appear in full in the report
    tests:
      - TestInit_CIResolveFailureWinsOverObservationOnTheExitCode
  # ── REQ-015 — post-init default scope stays diff-based ──
  - id: CLM-070
    requirement: REQ-015
    subject: cmd/backstop
    text: >
      A `backstop gate` run with no scope flags in a project init produced resolves
      `GateScopeModeDiff`
    tests:
      - TestInit_PostInitGateWithNoScopeFlagsIsDiffScoped
  - id: CLM-071
    requirement: REQ-015
    kind: absence
    text: >
      DENYLIST — the config init writes carries no key that changes, overrides, or pins the
      gate's default scope
    tests:
      - TestInit_GeneratedConfigCarriesNoScopeOverride
  # ── REQ-016 — --ci takes the full pinned ref, verbatim ──
  - id: CLM-072
    requirement: REQ-016
    text: >
      The `--ci` value is handed to the resolve+apply path byte-identical to what the consumer
      typed — no trimming, completing, defaulting, or normalization
    tests:
      - TestInit_CIRefIsPassedThroughVerbatim
  - id: CLM-073
    requirement: REQ-016
    kind: absence
    text: >
      DENYLIST — the init source set holds no pack name, no recipe id, no version literal, and
      no CI-platform name; core constructs no part of the ref
    tests:
      - TestInit_SourceSetConstructsNoPartOfTheCIRef
  - id: CLM-074
    requirement: REQ-016
    text: >
      With `--ci` omitted, no recipe resolution or apply is attempted at all
    tests:
      - TestInit_CIOmittedAttemptsNoRecipeResolution
  - id: CLM-075
    requirement: REQ-016
    text: >
      With `--ci` omitted, init states that no CI was wired and names `backstop recipe apply`
      plus the pinned `<pack>:<recipe>@<version>` ref shape as the way to wire it later
    tests:
      - TestInit_CIOmittedReportsTheSkipAndHowToWireItLater
  - id: CLM-076
    requirement: REQ-016
    subject: cmd/backstop
    text: >
      With `--ci` omitted and every other requested step delivered, init exits 0 — a skipped
      optional step is not an error
    tests:
      - TestInit_CIOmittedIsADeliberateNoOpExitingZero
  - id: CLM-077
    requirement: REQ-016
    kind: absence
    subject: cmd/backstop
    text: >
      DENYLIST — no `ci` verb or subcommand is added to the command tree; `--ci` is a flag on
      init and nothing else
    tests:
      - TestInit_AddsNoCIVerbToTheCommandTree
  - id: CLM-078
    requirement: REQ-016
    text: >
      A ref naming a pack that appears nowhere in core is dispatched normally, because core
      holds no allowlist of acceptable pack names
    tests:
      - TestInit_ArbitraryPackNameInTheCIRefIsDispatchedNormally
  # ── REQ-017 — asked-for-and-unresolvable: all four failure modes ──
  - id: CLM-079
    requirement: REQ-017
    text: >
      A malformed or unpinned `--ci` value surfaces `ParseRecipeRef`'s own error verbatim,
      attributed to the CI step; init performs no pin defaulting
    tests:
      - TestInit_UnpinnedCIRefSurfacesTheParseErrorVerbatim
  - id: CLM-080
    requirement: REQ-017
    text: >
      A `--ci` ref naming a pack that is not installed surfaces the shipped
      "is not among the installed packs" error verbatim, attributed to the CI step
    tests:
      - TestInit_UninstalledPackInCIRefSurfacesTheResolveErrorVerbatim
  - id: CLM-081
    requirement: REQ-017
    text: >
      A `--ci` ref naming a recipe the installed pack does not index surfaces the shipped
      "declares no recipe" error verbatim, attributed to the CI step
    tests:
      - TestInit_UndeclaredRecipeInCIRefSurfacesTheResolveErrorVerbatim
  - id: CLM-082
    requirement: REQ-017
    text: >
      A `--ci` ref whose pinned version differs from the recipe's declared version surfaces the
      shipped version-mismatch error verbatim, naming both versions
    tests:
      - TestInit_PinMismatchInCIRefSurfacesTheResolveErrorVerbatim
  - id: CLM-083
    requirement: REQ-017
    text: >
      In every one of the four CI failure modes, every other init step still completes and is
      reported with its own outcome
    tests:
      - TestInit_EveryOtherStepCompletesWhenTheCIStepFails
  - id: CLM-084
    requirement: REQ-017
    subject: cmd/backstop
    text: >
      Each of the four CI failure modes makes init exit non-zero
    tests:
      - TestInit_EveryCIResolveFailureExitsNonZero
  - id: CLM-085
    requirement: REQ-017
    kind: absence
    text: >
      DENYLIST — init implements no CI detection: it never enumerates installed packs looking
      for a CI pack, never probes for a platform config file, and adds no guidance text of its
      own beyond attributing the surfaced error to the CI step
    tests:
      - TestInit_ImplementsNoCIDetectionOrBespokeGuidance
  # ── REQ-018 — portable git-ref installs; local provenance is ISSUE-055's ──
  - id: CLM-086
    requirement: REQ-018
    text: >
      A git-ref `--pack` value installs and lands in `backstop.lock` as a git-source entry
      carrying its source coordinate
    tests:
      - TestInit_GitRefPackInstallLandsAsAPortableLockEntry
  - id: CLM-087
    requirement: REQ-018
    subject: cmd/backstop
    text: >
      A `--pack` value that is a local filesystem path is refused as a config error exiting 2,
      naming the lock-portability reason and pointing at `backstop pack add` after init
    tests:
      - TestInit_LocalPathPackRefIsRefusedForLockPortability
  - id: CLM-110
    requirement: REQ-018
    subject: cmd/backstop
    text: >
      Init's local-path decision is the SHIPPED `isLocalPath`'s and not a second one: every
      form that classifier accepts (`/abs`, `./rel`, `../rel`, and a platform-absolute path)
      is refused by init, and every form it rejects (a bare `org/pack@version` git ref) is
      accepted — so the two cannot disagree about any ref
    tests:
      - TestInit_LocalPathClassificationMatchesTheShippedClassifier
  - id: CLM-088
    requirement: REQ-018
    kind: absence
    text: >
      DENYLIST — no lock entry init produces carries a `local_path` value, so the committed
      lock holds no machine-specific path
    tests:
      - TestInit_ProducesNoLockEntryCarryingALocalPath
  - id: CLM-089
    requirement: REQ-018
    kind: absence
    text: >
      DENYLIST — the init source set implements no local-provenance cache and writes no
      pack-sources record; that half is ISSUE-055's
    tests:
      - TestInit_ImplementsNoLocalProvenanceCache
  # ── REQ-019 — the transcribed sequence and its acceptance bar ──
  - id: CLM-090
    requirement: REQ-019
    text: >
      The executed step order is exactly the sequence REQ-019 states — git, config, layout,
      packs, gitignore, scaffold, toolchain, baseline, ci, observe — with the pack step BEFORE
      the gitignore step and the scaffold step BEFORE the toolchain step, and subtracting a
      capability removes its step without reordering the rest
    tests:
      - TestInit_ExecutesTheTranscribedStepSequenceInOrder
  - id: CLM-111
    requirement: REQ-019
    text: >
      ORDERING IS LOAD-BEARING, NOT COSMETIC — a `--pack` run on a fresh repo emits a
      `.gitignore` that ALREADY CARRIES the installed pack's declared `stdout_artifact`
      entries at the moment the gitignore step completes, which is only reachable because the
      pack step ran first; the same run with the two steps swapped would emit a file missing
      every pack-derived entry
    tests:
      - TestInit_GitignoreCarriesPackEntriesBecausePacksInstallFirst
  - id: CLM-091
    requirement: REQ-019
    subject: cmd/backstop
    text: >
      ACCEPTANCE — on a fresh directory, `backstop init` (full-SDLC profile) installing the
      purpose-built fixture pack source via a `--pack` git ref, followed by `backstop gate`,
      reaches PASS with zero violations. The pack is published as a hermetic remote by the
      shipped harness (`newHermeticRemote`, `cmd/backstop/hermetic_remote_harness_test.go`) and
      installed by genuine git ref, and its engine declares `gate_type: lint`, so the gate under
      assertion has a real pack engine with a real declared command dispatched through the
      generic `pack_engines` path — a packless run does not satisfy this claim
    tests:
      - TestInit_FullSdlcFreshRepoWithRealPacksThenGateReachesPassWithZeroViolations
  - id: CLM-092
    requirement: REQ-019
    subject: cmd/backstop
    text: >
      ACCEPTANCE — on a fresh directory, `backstop init --no-sdlc` (pack-only profile) installing
      the SAME fixture pack source via a `--pack` git ref, followed by `backstop gate`, reaches
      PASS with zero violations, under the same hermetic remote harness and the same
      no-packless-run condition as CLM-091. THE PROFILE IS WHY THE GATE TYPE MATTERS: this
      profile writes `level: off` for `contract_signature` and `test_substantiveness`, so a pack
      declaring only those gate types could not have affected this verdict at all, and its PASS
      would have been unfalsifiable. A `lint` engine dispatches through `pack_engines`, which
      this profile leaves enforcing
    tests:
      - TestInit_PackOnlyFreshRepoWithRealPacksThenGateReachesPassWithZeroViolations
  - id: CLM-112
    requirement: REQ-019
    subject: cmd/backstop
    text: >
      NON-VACUITY GUARD on the acceptance claims, IN THREE PARTS, because dispatch alone is too
      weak a bar — a fixture engine with no real content would satisfy "an engine ran" while
      asserting nothing. (1) The gate run each acceptance claim asserts PASS over actually
      DISPATCHED the installed pack's engine, so a PASS from a gate with zero dispatched pack
      engines cannot satisfy CLM-091 or CLM-092. (2) That dispatched engine is demonstrably
      CAPABLE of producing a violation: the same pack, installed the same way, run over a
      fixture project deliberately containing the marker its rule flags, produces at least one
      violation attributed to that engine — so the acceptance PASS is a pack that CAN go red and
      did not, never a pack that cannot go red at all. (3) That demonstration is run under BOTH
      acceptance profiles, because a violation that cannot reach the verdict under the pack-only
      profile's `enforcement.policy` would leave CLM-092 unfalsifiable even with parts (1) and
      (2) satisfied
    tests:
      - TestInit_AcceptanceGateRunsDispatchedRealPackEngines
      - TestInit_AcceptanceDispatchedEngineCanProduceAViolation
      - TestInit_AcceptanceEngineCanProduceAViolationUnderBothProfiles
  - id: CLM-138
    requirement: REQ-019
    text: >
      DEVIATION 2'S ORDERING IS LOAD-BEARING, NOT COSMETIC — a run supplying both `--pack` and
      `--scaffold` has the scaffold recipe's declared file ALREADY ON DISK at the moment the
      toolchain step executes its first entrypoint, which is only reachable because the scaffold
      step runs after the pack step (so the recipe resolves) and before the toolchain step (so
      the entrypoint sees the file). The claim asserts the observable state at the toolchain
      step's boundary, not the step list alone, so the order cannot be satisfied vacuously
    tests:
      - TestInit_ScaffoldedFileIsOnDiskBeforeTheToolchainStepRuns
  - id: CLM-139
    requirement: REQ-019
    subject: cmd/backstop
    text: >
      DD-7'S MOTIVATING EVIDENCE, IN PROVABLE FORM — an installed pack whose declared
      `test`/`build` entrypoint exits NON-ZERO over a project containing no source file exits
      ZERO over the same project once the scaffold recipe has run, with the scaffolded file as
      the only difference between the two runs. This is the "a compiler run over an empty repo
      can RED on no inputs" failure DD-7 cites, bound to a test rather than to a story, and it
      is what would regress silently if a later refactor moved the scaffold step after the
      toolchain step
    tests:
      - TestInit_ScaffoldedProjectTurnsAnEmptyProjectEntrypointFailureGreen
  # ── REQ-033 — the coverage floor is a documented, unsatisfied gap ──
  - id: CLM-093
    requirement: REQ-033
    kind: absence
    text: >
      DENYLIST — init writes no coverage-floor key of any shape; the generated config's
      `enforcement` block contains only keys the shipped schema declares, and the pack-only
      config is accepted by the schema's `additionalProperties: false` pass unchanged
    tests:
      - TestInit_InventsNoCoverageFloorEnforcementKey
  - id: CLM-094
    requirement: REQ-033
    text: >
      The pack-only profile's report names the forfeited coverage enforcement as an unwired
      gap, states that the spec-independent floor knob does not exist, and does not fail the
      run
    tests:
      - TestInit_PackOnlyReportsTheCoverageFloorGapWithoutFailing
  - id: CLM-095
    requirement: REQ-033
    text: >
      The full-SDLC profile emits no coverage-floor gap notice, because `coverage_threshold`
      is live under that profile
    tests:
      - TestInit_FullSdlcEmitsNoCoverageFloorGapNotice
  # ── REQ-035 — the preserve-producer matrix: all three producers, reported and scored ──
  # Every branch of preserveOrRegenerate (apply.go:348-390) gets its own report claim and its
  # own exit-code claim, so no producer can inherit another's language or verdict.
  - id: CLM-096
    requirement: REQ-035
    text: >
      USER-OWNED CLASS — a CI recipe of `kind: scaffolding` or `kind: implementing` whose
      `create` target is a file NO prior apply produced yields a preserve with `Rule` and
      `CoveringWaiver` both EMPTY (the unambiguous producer-(a) observation, CLM-123/CLM-124),
      and init names every such file in its own output
    tests:
      - TestInit_UserOwnedPreserveNamesEveryPreservedFile
  - id: CLM-097
    requirement: REQ-035
    text: >
      USER-OWNED CLASS — init states in words that no backstop gate was wired into that CI
      configuration. This sentence belongs to this class alone and appears for no other
    tests:
      - TestInit_UserOwnedPreserveStatesNoGateWasWired
  - id: CLM-098
    requirement: REQ-035
    text: >
      USER-OWNED CLASS — init gives the consumer a concrete next action rather than leaving the
      gap unresolved
    tests:
      - TestInit_UserOwnedPreserveGivesTheConsumerANextAction
  - id: CLM-099
    requirement: REQ-035
    subject: cmd/backstop
    text: >
      USER-OWNED CLASS — a user-owned preserve makes init exit non-zero even though the
      underlying apply succeeded; the broken promise is what the exit code carries
    tests:
      - TestInit_UserOwnedPreserveExitsNonZeroDespiteSuccessfulApply
  - id: CLM-113
    requirement: REQ-035
    text: >
      INDETERMINATE CLASS — a preserve with `Rule` and `CoveringWaiver` both EMPTY under a
      resolved recipe whose declared `kind` is `templating` is reported as a GAP that names the
      file and states that init CANNOT DETERMINE whether the recipe's output is present in it,
      with a next action. The declared kind does NOT separate producer (a) from producer (b)
      here — `preserveOrRegenerate` tests `!own.adopted` first (`apply.go:349`) and returns
      before the kind test at `:357`, so a never-adopted templating recipe yields a value
      identical to an adopted-and-materialized one — and this claim is what holds init to
      reporting that unknown instead of resolving it by guess
    tests:
      - TestInit_TemplatingKindEmptyPairPreserveIsReportedAsIndeterminateGap
  - id: CLM-114
    requirement: REQ-035
    subject: cmd/backstop
    text: >
      INDETERMINATE CLASS — an indeterminate preserve makes init exit NON-ZERO on DD-15's
      refuse posture (init cannot confirm it delivered the CI wiring it was asked for), and
      init's output for it carries NO "no gate was wired" assertion, because that is precisely
      the half init cannot know. Both halves are asserted together: an implementation that
      exits 0 fails, and one that borrows the USER-OWNED sentence fails
    tests:
      - TestInit_IndeterminatePreserveExitsNonZeroAndAssertsNoUnwiredGate
  - id: CLM-115
    requirement: REQ-035
    text: >
      PRODUCER (c) WAIVER-COVERED — a recipe-owned target whose divergence is covered by an
      ACTIVE waiver yields a preserve with `Rule` and `CoveringWaiver` both POPULATED, and init
      reports the accountable customization naming that rule and that covering token
    tests:
      - TestInit_WaiverCoveredPreserveNamesTheRuleAndCoveringToken
  - id: CLM-116
    requirement: REQ-035
    kind: absence
    subject: cmd/backstop
    text: >
      PRODUCER (c) — a waiver-covered preserve does NOT make init exit non-zero, is NOT reported
      as a gap, and init's output for it carries NO "no gate was wired" language: the gate IS
      wired and the consumer accounted for the divergence. This is the false report the
      undiscriminated reading produced, and this claim is what makes it unrepresentable
    tests:
      - TestInit_WaiverCoveredPreserveExitsZeroAndClaimsNoUnwiredGate
  - id: CLM-100
    requirement: REQ-035
    text: >
      A partial apply — some targets written, at least one USER-OWNED preserve — is reported
      as a gap, not as success
    tests:
      - TestInit_PartialApplyWithAUserOwnedPreserveIsReportedAsAGap
  - id: CLM-101
    requirement: REQ-035
    text: >
      An apply with zero preserves of ANY producer (every declared target absent beforehand) is
      reported as success and contributes no gap
    tests:
      - TestInit_ApplyWithNoPreservesIsReportedAsSuccess
  - id: CLM-117
    requirement: REQ-035
    subject: cmd/backstop
    text: >
      An apply whose every preserve is WAIVER-COVERED (`Rule` and `CoveringWaiver` populated) is
      reported as SUCCESS, contributes no gap, and exits 0 — the accountable class is the ONLY
      preserve class that leaves an apply successful. An apply carrying any USER-OWNED or any
      INDETERMINATE preserve is a gap and exits non-zero regardless of how many targets were
      written alongside it
    tests:
      - TestInit_ApplyWithOnlyWaiverCoveredPreservesIsSuccessAndExitsZero
  - id: CLM-123
    requirement: REQ-035
    text: >
      USER-OWNED CLASS, `scaffolding` — a preserve with the field pair EMPTY under a resolved
      recipe declaring `kind: scaffolding` is UNAMBIGUOUSLY producer (a), because branch (b) is
      unreachable for that kind, and init states in words that no backstop gate was wired into
      the file
    tests:
      - TestInit_ScaffoldingKindEmptyPairPreserveIsClassifiedUserOwned
  - id: CLM-124
    requirement: REQ-035
    text: >
      USER-OWNED CLASS, `implementing` — a preserve with the field pair EMPTY under a resolved
      recipe declaring `kind: implementing` is UNAMBIGUOUSLY producer (a) for the same reason,
      and init states in words that no backstop gate was wired into the file
    tests:
      - TestInit_ImplementingKindEmptyPairPreserveIsClassifiedUserOwned
  - id: CLM-125
    requirement: REQ-035
    subject: cmd/backstop
    text: >
      THE PAIR OUTRANKS THE KIND — a preserve whose `Rule` and `CoveringWaiver` are POPULATED is
      classified WAIVER-COVERED at EVERY declared kind, `templating` included: init reports the
      accountable customization, uses no "no gate was wired" language, reports no gap, and exits
      0. The kind is consulted only to split the empty-pair case
    tests:
      - TestInit_PopulatedWaiverPairIsWaiverCoveredAtEveryRecipeKind
  # The SAME three observable classes over the SCAFFOLD step's apply. The classifier is shared;
  # what differs is the sentence, because "no gate was wired" is a statement about CI wiring the
  # scaffold step knows nothing about.
  - id: CLM-140
    requirement: REQ-035
    text: >
      SCAFFOLD STEP, USER-OWNED CLASS — a scaffold recipe of `kind: scaffolding` or
      `kind: implementing` whose `create` target is a file no prior apply produced yields a
      preserve with the field pair EMPTY, and init names every such file and states that the
      consumer's own file was left in place and the recipe's declared source file was therefore
      NOT written. Init must NOT use the CI step's "no backstop gate was wired" sentence here —
      that sentence asserts something about CI wiring this step knows nothing about
    tests:
      - TestInit_ScaffoldUserOwnedPreserveNamesTheFileAndSaysNoSourceFileWasWritten
  - id: CLM-141
    requirement: REQ-035
    subject: cmd/backstop
    text: >
      SCAFFOLD STEP, USER-OWNED CLASS — a user-owned scaffold preserve makes init exit non-zero
      even though the underlying apply succeeded, and gives the consumer a concrete next action.
      The consumer asked for a scaffolded source file and does not have one
    tests:
      - TestInit_ScaffoldUserOwnedPreserveExitsNonZeroWithANextAction
  - id: CLM-142
    requirement: REQ-035
    subject: cmd/backstop
    text: >
      SCAFFOLD STEP, INDETERMINATE CLASS — a scaffold preserve with the field pair EMPTY under a
      resolved recipe whose declared `kind` is `templating` is reported as a GAP naming the file
      and stating that init CANNOT DETERMINE whether the recipe's output is present in it, with
      a next action, and init exits NON-ZERO on DD-15's refuse posture. Both halves are asserted
      together: an implementation that exits 0 fails, and one that asserts the output IS or is
      NOT present fails
    tests:
      - TestInit_ScaffoldTemplatingKindEmptyPairPreserveIsReportedAsIndeterminateGap
  - id: CLM-143
    requirement: REQ-035
    subject: cmd/backstop
    text: >
      SCAFFOLD STEP, WAIVER-COVERED CLASS — a scaffold preserve whose `Rule` and
      `CoveringWaiver` are POPULATED is reported as an accountable customization naming that
      rule and that covering token, contributes NO gap, and exits 0. The accountable class is
      the only preserve class that leaves either apply successful, at either step
    tests:
      - TestInit_ScaffoldWaiverCoveredPreserveIsSuccessAndExitsZero
  - id: CLM-144
    requirement: REQ-035
    kind: absence
    text: >
      DENYLIST — the init source set contains EXACTLY ONE preserve classifier, and both the CI
      step and the scaffold step call it. A second, step-local classification of the
      `Rule`/`CoveringWaiver` pair and the declared kind is the "one authority, not a copy"
      hazard this spec refuses for local-path classification (REQ-018) and for the adoption bit
      (REQ-035), and two classifiers drifting would let one step report a class the other would
      not
    tests:
      - TestInit_HoldsExactlyOnePreserveClassifierSharedByBothRecipeSteps
  - id: CLM-102
    requirement: REQ-017
    text: >
      A `--ci` ref whose pack indexes the recipe but whose colocated `recipe.yml` is unreadable
      or unparseable surfaces that shipped error verbatim too — a pack defect is classified no
      differently from a bad consumer ref, and init adds no interpretation to either
    tests:
      - TestInit_UnparseableRecipeManifestSurfacesTheResolveErrorVerbatim

contracts:
  - file: pkg/initialize/capability.go
    provides:
      - name: Capability
        kind: type
        signature: "type Capability string"
      - name: DefaultCapabilities
        kind: function
        signature: "func DefaultCapabilities() []Capability"
      - name: ResolveCapabilities
        kind: function
        signature: "func ResolveCapabilities(only []string, excluded []string) (map[Capability]bool, error)"
  - file: pkg/initialize/initialize.go
    provides:
      - name: Options
        kind: type
        signature: "type Options struct { ProjectRoot string; Capabilities map[Capability]bool; PackRefs []string; CIRecipeRef string; ScaffoldRecipeRef string }"
        notes: "ScaffoldRecipeRef is REQ-009's `--scaffold <pack>:<recipe>@<version>` value, carried EXACTLY as CIRecipeRef is: a whole opaque string, empty when the flag was omitted (the honest-skip case), never decomposed into pack/recipe/version parts anywhere in core. Two ref-shaped fields and no ref-shaped CONSTRUCTOR is the point — a field that held a pack name or a version separately would be core holding half a ref, which is the DD-13 bake REQ-016 v1.2.0 was corrected to remove. Neither field has a param companion: init supplies no recipe param to either apply (CLM-136)."
      - name: Outcome
        kind: type
        signature: "type Outcome int"
      - name: StepReport
        kind: type
        signature: "type StepReport struct { Step string; Outcome Outcome; Detail string }"
      - name: Result
        kind: type
        signature: "type Result struct { Steps []StepReport; Observations []DimensionCount; Preserved []ClassifiedPreserve; BrokenPromise bool }"
        notes: "Preserved carries CLASSIFIED preserves, not paths (REQ-035). A []string discarded the Rule/CoveringWaiver pair and the recipe kind, which are the only two observables that separate an accountable waiver-covered customization from an unambiguous user-owned brownfield gap from the case init cannot resolve at all — and reporting them all as the brownfield gap was the false 'no gate was wired' this type exists to prevent."
      - name: PreserveClass
        kind: type
        signature: "type PreserveClass int"
        notes: "REQ-035's OBSERVABLE classes, which are deliberately NOT the code's three producers: preserveOrRegenerate (apply.go:348-390) has three branches, but its `!own.adopted` test at :349 returns BEFORE the kind test at :357, so a never-adopted templating recipe emits a value byte-identical to an adopted-and-materialized one. The classes init can actually derive from what it holds are three: PreserveWaiverCovered (Rule/CoveringWaiver populated, any kind — no gap, exit 0), PreserveUserOwned (pair empty, kind scaffolding or implementing — the brownfield gap, 'no gate was wired', exit non-zero), and PreserveIndeterminate (pair empty, kind templating — producers (a) and (b) indistinguishable; scored as a gap on DD-15's refuse posture, exit non-zero, and reported with NO 'no gate was wired' assertion). Naming the type for producers would re-assert the discrimination init cannot make."
      - name: ClassifiedPreserve
        kind: type
        signature: "type ClassifiedPreserve struct { Path string; Class PreserveClass; Rule string; CoveringWaiver string }"
      - name: DimensionCount
        kind: type
        signature: "type DimensionCount struct { Dimension string; Count int }"
      - name: Runner
        kind: type
        signature: "type Runner struct"
      - name: NewRunner
        kind: function
        signature: "func NewRunner(packs PackInstaller, recipes RecipeApplier, gates GateRunner, tools ToolchainProber, seeds BaselineSeeder) (*Runner, error)"
      - name: Run
        kind: method
        signature: "func (r *Runner) Run(opts Options) (Result, error)"
    consumes:
      - source: pkg/config
        name: Config
        kind: type
      - source: pkg/pack
        name: Manifest
        kind: type
      - source: pkg/recipe
        name: ParseRecipeRef
        kind: function
      - source: pkg/recipe
        name: PreservedDivergence
        kind: type
      - source: pkg/pack/distribution
        name: IsLocalPath
        kind: function
  - file: pkg/initialize/seams.go
    provides:
      - name: PackInstaller
        kind: interface
        signature: "type PackInstaller interface { Install(projectRoot string, ref string) error }"
      - name: ApplyOutcome
        kind: type
        signature: "type ApplyOutcome struct { Written []string; Preserved []recipe.PreservedDivergence; RecipeKind string }"
        notes: "REQ-035. Carries the applier's OWN PreservedDivergence values with Rule and CoveringWaiver intact, plus the resolved recipe's declared kind. All three are already exported at HEAD (recipe.PreservedDivergence, recipe.ApplyResult.Preserved, recipe.ResolvedRecipe.Manifest.Kind), so this widening touches no file under pkg/recipe — which REQ-009 forbids editing. These two fields are the WHOLE of what init can classify on: the applier's recipe-level adoption bit (apply.go:166, keyed by the unexported adoptionKey) is not carried here and must not be reconstructed, so the empty-pair-plus-templating case stays INDETERMINATE by design rather than being resolved by a second derivation of a recipe's adoption identity."
      - name: RecipeApplier
        kind: interface
        signature: "type RecipeApplier interface { Apply(projectRoot string, ref string) (ApplyOutcome, error) }"
        notes: "ONE seam, TWO callers (REQ-009): the CI step and the scaffold step both apply through this single interface with the ref their own flag supplied, and the returned preserves run through the single shared classifier CLM-144 pins. The interface takes the ref as an opaque string precisely so adding the scaffold step required no new method, no per-step variant, and no place for core to construct a ref part."
      - name: GateRunner
        kind: interface
        signature: "type GateRunner interface { Run(projectRoot string) ([]DimensionCount, error) }"
      - name: ToolchainProber
        kind: interface
        signature: "type ToolchainProber interface { Probe(projectRoot string) ([]StepReport, error) }"
        notes: "Interface ONLY. Its production implementation is packToolchainProber in cmd/backstop/init_toolchain.go, NOT here, and the location is forced: the allowlist trust gate and the command splitter REQ-011 binds execution to (checkEngineToolAllowed, splitCommand) are unexported in package main, so a pkg/initialize implementation could not reach them and would have to write a second copy — the exact second execution path REQ-011 forbids."
      - name: BaselineSeeder
        kind: interface
        signature: "type BaselineSeeder interface { Seed(projectRoot string) (string, error) }"
  - file: cmd/backstop/init_toolchain.go
    provides:
      - name: packToolchainProber
        kind: type
        signature: "type packToolchainProber struct { Packs []*pack.Manifest; Runner check.CommandRunner }"
        notes: "REQ-011's concrete ToolchainProber — the file a plan hangs the allowlist binding on. Selects bindings by engine.GateTypeTest and engine.GateTypeBuild ONLY, then runs each through checkEngineToolAllowed -> splitCommand -> the shared check.CommandRunner: the same three steps runFindingsEngine takes (pack_gate.go:573-600) minus the SARIF parse, so there is no second execution path to audit and no shell anywhere. IT ENTERS THE RUNNER THROUGH Run, NOT RunStdout, and that is the one deliberate divergence from runFindingsEngine (which calls RunStdout at pack_gate.go:648): Run returns combined stdout+stderr — the method pkg/check/runner.go:17 documents as being for 'the build/test executors whose violation messages may legitimately include stderr' — and REQ-011 case (c) must report captured output VERBATIM, which a stdout-only capture would render empty for a failing entrypoint that diagnoses on stderr. Init parses nothing, so the SARIF-contamination reason RunStdout exists for does not apply here. CLM-122 is the tripwire. Mirrors SPEC-070's checkToolchainRuns deliberately — one check, two callers, one execution route."
      - name: Probe
        kind: method
        signature: "func (p *packToolchainProber) Probe(projectRoot string) ([]initialize.StepReport, error)"
    consumes:
      - source: cmd/backstop
        name: checkEngineToolAllowed
        kind: function
      - source: cmd/backstop
        name: splitCommand
        kind: function
      - source: cmd/backstop
        name: loadInstalledPacks
        kind: function
      - source: pkg/check
        name: CommandRunner
        kind: interface
      - source: pkg/pack/engine
        name: GateType
        kind: type
  - file: cmd/backstop/init.go
    provides:
      - name: newInitCommand
        kind: function
        signature: "func newInitCommand() *cobra.Command"
    consumes:
      - source: pkg/initialize
        name: NewRunner
        kind: function
      - source: pkg/initialize
        name: Options
        kind: type
  - file: pkg/pack/distribution/add.go
    provides:
      - name: IsLocalPath
        kind: function
        signature: "func IsLocalPath(ref string) bool"
        notes: "REQ-018. The SHIPPED classifier at add.go:96, EXPORTED (isLocalPath -> IsLocalPath, all in-package callers updated) so init calls the one authority instead of writing a second definition of 'is this a local path'. Behavior is UNCHANGED — this is a rename, not a reimplementation; the SPEC-056 precedent is pack.ValidatePackName as 'one authority, not a copy'. This is the ONLY edit this spec makes to pkg/pack/distribution."
---

# SPEC-069: Backstop Init

## Overview

`backstop init` is the command that takes a consuming project from "the binary is present" to
first useful output. It does not exist today: a consumer hand-writes `backstop.yml`, guesses an
artifact layout, adds packs one by one, hand-writes a `.gitignore` that matches what the packs
emit, and then debugs whatever the first gate run surfaces with no diagnostic help. Two profiles
were onboarded by hand and both reached a clean gate PASS, so the flow is validated; this spec
transcribes that flow rather than inventing a new one (bundle DD-8).

The spec is organized around one hard invariant and three ownership boundaries.

**The invariant (bundle DD-13).** Init performs zero language, framework, ecosystem, or
CI-platform detection, and core holds no literal naming any of them anywhere on the init path.
Every language-shaped decision arrives either as pack-declared data (a manifest's engines, a
recipe's declared target) or as a consumer argument (`--pack`, `--ci`, `--scaffold`). This is why
the profile
fork in REQ-003 keys on a capability name rather than an inspected project identity, why the
`.gitignore` set in REQ-005 is three backstop literals plus pack-declared `stdout_artifact`
values, why the toolchain check in REQ-011 selects entrypoints by `gate_type` rather than by tool,
and why `--ci` in REQ-016 and `--scaffold` in REQ-009 each take the whole pinned recipe ref rather
than a platform or language name. The scaffold step is the sharpest instance of the invariant
paying off: DD-7 requires init to produce a first SOURCE file, and init does — without core
holding a single byte of it, because the file arrives as a recipe payload the consumer named.
Several
claims in this spec are DENYLIST claims whose entire content is that a literal is absent; they are
the invariant's teeth.

**Boundary one — three requirements are consumed, not built.** REQ-012 (a gitignored local
baseline at `.backstop/baseline.json`) and REQ-013 (the self-contradictory remoteless
`baseline_comparison` message) are owned by ISSUE-056, which is open on the issue→plan track and
cites this bundle's OQ-3 as its direction. REQ-018's local-provenance half (restoring local-path
packs from a gitignored record) is owned by ISSUE-055, citing OQ-4. This spec designs none of the
three. It provides a delegation seam for baseline seeding and otherwise proves the boundary with
absence claims: no code here writes `.backstop/baseline.json`, no file under `pkg/gate` changes,
and no pack-sources cache is written.

**Boundary two — REQ-033 is a documented, unsatisfied gap.** REQ-033 requires init to wire a
spec-independent coverage floor for the pack-only profile, since REQ-003 turns
`coverage_threshold` off there. The knob it would wire has no owner anywhere and does not exist:
`artifacts/backstop-yml/v1/schema.json` declares `enforcement` with
`additionalProperties: false` over exactly `security`, `waiver_warning_days`, `semgrep_version`,
`baseline_ttl`, `test_command`, `toolchain`, and `policy`. A `coverage.min_pct`-shaped key written
by init would not be tolerated-and-ignored — it would be REJECTED at config load, so the config
init generated would not load. Inventing the schema field here would be init designing an
enforcement-policy surface, which the bundle explicitly says it does not do. The spec therefore
implements the honest fallback: init reports the forfeiture as an unwired gap and proves, by
absence claim, that it invented nothing.

**Boundary three — the guards seed lands first.** REQ-004's `.backstop/`-rooted layout is
unimplementable without the config-resolved artifact root that SPEC-068 (bundle REQ-029) owns:
today `pkg/gate/artifact_status.go`, `pkg/validate/resolved_by.go`, and `pkg/scaffold/scaffold.go`
each hardcode the root layout independently and no artifact-root key exists in the `backstop.yml`
schema. Init scaffolding `.backstop/` before that lands would create a layout discovery cannot
see — the silent-undiscovery false green DD-15 exists to prevent. SPEC-068 is a hard prerequisite,
not a preference.

Out of scope, and named: `backstop doctor` (SPEC-070), the guards themselves (SPEC-068), baseline
generation mechanics (BUNDLE-007), binary and toolchain acquisition (DIR-001), pack-declared
dependency installation (retired bundle REQ-032 — the MVP is a pack-authoring convention, so init
reports an uninstalled toolchain rather than repairing it), and any edit to `pkg/gate`,
`pkg/recipe`, or the `backstop.yml` schema.

## Requirements

Twenty-one requirements, carried in frontmatter, each bound by `supports` to its BUNDLE-003
requirement at that requirement's CURRENT version. Their character differs and the difference is
load-bearing:

| Character | Requirements | What the spec owes |
|---|---|---|
| Init behavior built here | REQ-001 – REQ-011, REQ-014, REQ-016, REQ-017, REQ-018 (git-ref half), REQ-019, REQ-035 | Full implementation plus claims |
| Consumed from another artifact | REQ-012, REQ-013, REQ-018 (local-provenance half) | A delegation seam plus absence claims proving nothing was re-implemented |
| Satisfied by not regressing | REQ-015 | A claim holding the shipped diff-scope default |
| Documented gap, unsatisfied | REQ-033 | An honest report plus an absence claim proving no schema surface was invented |

The capability set REQ-002 defines is the spine of the command, and every other requirement hangs
off one of its members:

| Capability | Flag to subtract | Requirements it carries |
|---|---|---|
| `git` | `--no-git` | REQ-006 |
| `sdlc` | `--no-sdlc` | REQ-003 (profile fork), REQ-004 |
| `gitignore` | `--no-gitignore` | REQ-005 |
| `packs` | `--no-packs` | REQ-010, REQ-018 |
| `toolchain` | `--no-toolchain` | REQ-011 |
| `baseline` | `--no-baseline` | REQ-012 (delegated) |
| `observe` | `--no-observe` | REQ-014 |

`backstop.yml` generation (REQ-003) is unconditional and is NOT a capability — an init that does
not write the config produces nothing a consumer can use. CI wiring is NOT a capability either:
per bundle OQ-7 there is no `--no-ci` to need, because omitting `--ci` IS the opt-out. Adding a
`ci` capability would give one outcome two different reports, and CLM-077 forbids the `ci` verb
outright.

Claims are defined in frontmatter, one block per requirement.

## Implementation

### 1. Package layout

Four new files across two packages, plus one one-line export in a third:

- **`pkg/initialize/`** — the orchestration engine. `capability.go` holds the capability
  vocabulary and its resolution; `seams.go` holds the five injected dependency interfaces and
  `ApplyOutcome`; `initialize.go` holds `Options`, `Result`, the preserve-classification types,
  and the `Runner` that executes the step sequence.
- **`cmd/backstop/init.go`** — the cobra command: flag definitions, flag→`Options` translation,
  report rendering, and exit-code mapping. It is thin by construction, mirroring
  `cmd/backstop/recipe_apply.go`.
- **`cmd/backstop/init_toolchain.go`** — `packToolchainProber`, the concrete `ToolchainProber`.
  It lives in package `main` because that is where `checkEngineToolAllowed` and `splitCommand`
  are, and REQ-011 binds execution to those exact functions (§7).
- **`pkg/pack/distribution/add.go`** — the sole edit outside the two new packages: `isLocalPath`
  is exported as `IsLocalPath` so REQ-018 has one authority rather than a copy (§6). Behavior
  unchanged.

`NewRunner` is a FAIL-CLOSED positional constructor that errors naming any nil dependency, the
same shape `pkg/pack/distribution/command.go` uses for `NewAddCommand`. This is deliberate reuse
of an established pattern in this repo: it makes a half-wired runner unconstructable rather than a
runtime nil-deref, and it is what lets the mandated tests drive real implementations for some
seams and fakes for others without a global registry.

### 2. Capability resolution (REQ-002)

`ResolveCapabilities(only, excluded)` is the ONLY entry point that produces a capability set.

1. If both `only` and `excluded` are non-empty → error. Two contradictory expressions of the same
   set is a config defect in the invocation, not a precedence puzzle to resolve silently. This
   mirrors the shipped `--file`/`--all` mutual exclusion in `backstop gate`.
2. Every supplied name is checked against the seven-member vocabulary; an unrecognized name errors
   listing all seven.
3. With neither flag → all seven. With `--only` → exactly the named subset. With `--no-` → the
   seven minus the named.

`--only` cannot ADD, because its input is validated against the same seven-name vocabulary — there
is no eighth name to add.

### 3. The step sequence (REQ-019)

`Runner.Run` executes ten steps in a fixed order, each producing exactly one `StepReport`.
Subtracting a capability removes its step and reorders nothing.

| # | Step | Capability | Behavior |
|---|---|---|---|
| 1 | `git` | `git` | `git init` only when no `.git` (REQ-006) |
| 2 | `config` | (unconditional) | Write the profile-correct `backstop.yml` (REQ-003) |
| 3 | `layout` | `sdlc` | Create the six `.backstop/` artifact dirs (REQ-004) |
| 4 | `packs` | `packs` | Install each `--pack` ref via the git-ref path (REQ-010, REQ-018) |
| 5 | `gitignore` | `gitignore` | Emit/append the canonical ignore set (REQ-005) |
| 6 | `scaffold` | (governed by `--scaffold`) | Apply the pinned scaffold ref verbatim, or report the skip (REQ-009, REQ-035) |
| 7 | `toolchain` | `toolchain` | Execute each declared `test`/`build` entrypoint once (REQ-011) |
| 8 | `baseline` | `baseline` | Delegate to the seeding seam (REQ-012) |
| 9 | `ci` | (governed by `--ci`) | Apply the pinned ref verbatim, or report the skip (REQ-016, REQ-017, REQ-035) |
| 10 | `observe` | `observe` | Run the gate, group findings by dimension (REQ-014) |

Steps 2 and 3 are separate even though both belong to the profile fork, because the pack-only
profile writes a config and creates no directories — folding them would make the pack-only path
express its difference as a conditional inside one step rather than as a step that is simply
absent.

Steps 6 and 9 are the two flag-governed steps and they are NOT capabilities: like `--ci`, the
presence of `--scaffold` is the whole of the opt-in, there is no `--no-scaffold`, and
`ResolveCapabilities` rejects `scaffold` as an eighth name (CLM-132). The capability vocabulary
stays at exactly the seven CLM-003 pins.

**Two deviations from DD-8's transcribed order, both forced, both declared.** Everything else is
transcribed as written, and no step is dropped without saying so.

*Deviation 1 — steps 4 and 5 are swapped.* DD-8 lists the canonical `.gitignore` as its step 4
and `backstop pack add` as its step 5. That transcription was made when the ignore list was a
fixed literal set with no pack dependency — the TypeScript-specific list DD-7's 2026-08-12
correction has since struck. REQ-005 v1.1.0 made the entry set a function of the INSTALLED packs'
declared `stdout_artifact` values, at which point emitting the file before any pack is installed
became unsatisfiable: step 4 on a fresh `--pack` run would write a `.gitignore` missing every
pack-derived entry, which is exactly the cross-repo ignore divergence DD-7 exists to end.
Swapping the two is the minimum change that makes REQ-005 and REQ-019 both true at once, and
CLM-111 asserts the consequence directly (the emitted file already carries the pack entries)
rather than asserting the order alone, so the ordering cannot be satisfied vacuously.

*Deviation 2 — DD-8 step 1's scaffold half becomes step 6.* DD-8 step 1 carries two obligations:
"scaffold a minimal project with ≥1 source file … **and install dependencies**". The
install-dependencies half is DROPPED because bundle REQ-032 was retired — init installs no
project dependencies (CLM-059). The SCAFFOLD half survives, and an earlier version of this spec
lost it: it declared "one deviation" while DD-8 step 1 had in fact been dropped whole and
unreported, and spec REQ-009 had been quietly narrowed from the bundle's "every language-specific
artifact **init produces** (a first source file, toolchain config)" to "every artifact a consumer
ends up with" — a re-scope that deleted init's own obligation, which CLM-048 then hardened into a
denylist. Both are corrected here. The step MOVES rather than staying at position 1 because DD-7
also requires the scaffolded file to "come from a pack recipe, never from core": a recipe cannot
resolve out of a pack that is not installed, so position 1 is unsatisfiable and the step lands
immediately after the pack step. It sits BEFORE the toolchain step, and that is load-bearing —
DD-7's evidence is a compiler reding on an empty project ("no inputs", `tsc` TS18003), and the
toolchain step is exactly the run that would hit it, so a scaffold step ordered after it would
re-manufacture the failure DD-7 exists to prevent. CLM-138 asserts the file is on disk at the
toolchain step's boundary, and CLM-139 binds the claim to DD-7's evidence directly: the same
entrypoint that reds over an empty project greens over the scaffolded one.

### 4. Profile derivation and config generation (REQ-003, REQ-033)

The profile is READ OFF the resolved capability set — `sdlc` present is full-SDLC, `sdlc` absent
is pack-only — and never off the project. Both profiles write `project:` set to the target
directory's basename and never write `language:` (retired by SPEC-046; `pkg/config/config.go`
absorbs it as a legacy key and ignores it, so writing it would emit a dead key that bakes a
language into the first file a consumer sees).

The pack-only profile additionally writes `enforcement.policy.<dim>.level: off` for exactly five
dimensions: `test_verification`, `coverage_threshold`, `contract_signature`,
`test_substantiveness`, `artifact_status_drift`. These are the dimensions that hard-error on a
missing `specs/` directory rather than skipping, and each is a real policy key (backstop-core's
own `backstop.yml` carries all five). The full-SDLC profile sets NONE of them to `off`.

Nothing else is written into `enforcement`. In particular, no coverage-floor key of any shape is
written (REQ-033): the schema's `additionalProperties: false` would reject it, so the pack-only
step instead appends a GAP line to its report naming the forfeited enforcement and the absent
knob.

Both generated configs are round-tripped through `config.Load` and the JSON-schema pass by
mandated tests, so a config init writes can never be one the binary refuses to read.

### 5. The `.gitignore` (REQ-005)

This step runs AFTER pack installation (step 5, not step 4) for the reason §3 states: part 2 of
the entry set below does not exist until the packs are on disk.

The entry set is built in two parts and in this order:

1. Three literals stated in core: `.backstop/packs/`, `.backstop/baseline.json`,
   `.backstop/pack-config-provenance.json`.
2. For each installed pack manifest, for each engine in its `engines:` map, the engine's declared
   `stdout_artifact` value when non-empty (`pkg/pack/manifest.go:97`). Nothing is derived, guessed,
   or defaulted for an engine that declares none.

The write is APPEND-ONLY against an existing file — the same posture
`pkg/pack/distribution/add.go`'s `ensureGitignore` already takes: read, skip entries already
present, append the rest, never rewrite. The report states the accepted residue in words:
`stdout_artifact` names what an engine writes for the gate to READ, not everything a toolchain
leaves on disk, so dependency directories and native build output remain the consumer's to ignore.
Closing that residue would require a new pack-manifest field, which is BUNDLE-004's surface, not
this spec's.

### 6. Pack installation (REQ-010, REQ-018)

Init installs exactly the refs supplied via the repeatable `--pack` flag, in the order given.
Before any install runs, each ref is classified: a value that resolves as a local filesystem path
is REFUSED as a config error (exit 2) naming the lock-portability reason and pointing at
`backstop pack add` after init.

**That classification has exactly one authority and init does not become a second.** The shipped
predicate is `isLocalPath` (`pkg/pack/distribution/add.go:96`) — the same one the add path
already forks local from remote on. This spec EXPORTS it as `IsLocalPath` (a rename with no
behavior change; in-package callers updated) and init calls it. That is the only edit this spec
makes to `pkg/pack/distribution`. Writing init's own predicate would be a second definition of
"is this a local path", and the two drifting apart is not hypothetical: a ref init classified
remote and the add path classified local would produce precisely the machine-specific
`local_path` lock entry REQ-018 exists to prevent, and it would fail nowhere near init.
SPEC-056 settled this shape elsewhere by making `pack.ValidatePackName` "one authority, not a
copy"; CLM-110 holds the two in agreement across every form the classifier recognizes.

This is the whole of REQ-018's init-owned half — the committed
lock carries only portable git-ref entries, so no `local_path` value ever originates from init.
The local-provenance cache that would let such a pack be restored on the same machine is
ISSUE-055's, and CLM-089 asserts its absence here.

With no `--pack` supplied, the step is a reported no-op: zero packs installed, and a line naming
`backstop pack add` as the way to add them. This mirrors the `--ci`-omitted posture exactly, and
it is what keeps a pack roster out of core — there is nothing for init to install that a consumer
did not name.

### 7. The toolchain-execution check (REQ-011)

The entrypoint set is derived from installed pack manifests: every engine whose declared
`gate_type` is `test` or `build`. Those two spellings are backstop's own kill-chain vocabulary
(`pkg/pack/engine/gatetype.go` defines exactly seven: lint, build, test, findings, coverage,
substantiveness, contracts), so selecting on them names a STAGE, never a tool.

**Execution goes through the gate's own path, and this is the sharpest thing init does.** Each
selected engine's declared command runs through exactly three steps: `checkEngineToolAllowed`
(`cmd/backstop/pack_gate.go:812` — the trusted-tool allowlist plus the lock-resolved version
pin), then `splitCommand` (`pack_gate.go:887` — whitespace argv tokenization), then the shared
`check.CommandRunner`. That is `runFindingsEngine`'s sequence (`pack_gate.go:573-600`) minus the
SARIF parse. Init introduces no second, weaker way to run a pack-declared command and never runs
one through a shell — init is executing arbitrary pack-supplied command strings, so an unbound
execution path here would be a hole in the trusted-tool invariant, not a style preference. The
trust gate sits BEFORE the splitter and the runner, so a refused tool's command is never handed
to anything; the refusal is a config error, not a toolchain verdict. SPEC-070's `checkToolchainRuns`
binds the same three steps for doctor's copy of this check, deliberately: one check, two callers,
one execution route to audit.

**The capture method is `Run`, not `RunStdout`, and that is the one deliberate divergence from
`runFindingsEngine`.** `check.CommandRunner` declares two methods with opposite purposes and the
shipped comments state them (`pkg/check/runner.go:14-21`): `RunStdout` returns ONLY stdout so a
tool's stderr banner cannot corrupt the SARIF bytes a findings engine writes to stdout, and `Run`
returns combined stdout+stderr for "the build/test executors whose violation messages may
legitimately include stderr". `runFindingsEngine` calls `RunStdout` (`pack_gate.go:648`) for the
first reason. Init's case (c) needs the second: a failing build or test entrypoint routinely puts
its whole diagnostic on stderr, so capturing stdout alone would render an EMPTY "captured output
VERBATIM" for exactly the failures REQ-011 case (c) exists to surface — a report that looks like
the tool said nothing when it said everything. Init parses nothing and renders to a human, so the
contamination hazard `RunStdout` guards against does not apply here while the lost-diagnostic
hazard does. The allowlist gate and the splitter are shared unchanged; only the capture method
differs, CLM-122 is the tripwire that fails if an implementer copies the method along with the
sequence, and this is the ONLY difference from `runFindingsEngine`'s path anywhere on init's
toolchain route. SPEC-070's `checkToolchainRuns` owes the same verbatim capture for the same case
and should make the same choice; that spec does not name a method today, and if it binds
`RunStdout` the two reports diverge on stderr-only failures.

The concrete `ToolchainProber` therefore lives in `cmd/backstop/init_toolchain.go`, not in
`pkg/initialize`. The location is forced rather than chosen — `checkEngineToolAllowed` and
`splitCommand` are unexported in package `main`, so a `pkg/initialize` implementation could not
reach them and would have to write a second copy of both, which is the thing this requirement
forbids. `pkg/initialize` holds the interface only.

**What the verdict is taken from.** Each engine's outcome is its own command's exit status and
nothing adjacent: not a package-manager command init ran itself (it runs none), not a
configuration file, not another engine's command in the same pack. Outcomes are independent —
one entrypoint failing does not change another's (CLM-057).

**The failure outcome is split in two, because one label cannot honestly carry both cases.**

| Case | Init's report | Exit |
|---|---|---|
| (a) executed, exited zero | pass, naming the pack and the command | 0 |
| (b) executable cannot be STARTED at all | owed SETUP, naming the pack and pointing at that pack's own documented install steps — no install command invented, nothing installed | non-zero |
| (c) started, exited NON-ZERO | exit code plus captured output VERBATIM — combined stdout+stderr, via `Run` — attributed to the pack and the command, NO cause claimed | non-zero |
| (d) no pack declares a `test`/`build` engine | capability-absent | 0 |

The (b)/(c) split is not a refinement — it is the requirement. Bundle REQ-011 v1.1.0's first
sentence forbids inferring toolchain health "from package-manager configuration OR EXIT CODE",
and its owed-setup clause is CONDITIONAL on cause: "when the check fails BECAUSE THOSE
DEPENDENCIES ARE ABSENT". Labeling every non-zero exit as owed setup would drop that condition
and commit the exact exit-code cause-inference the sentence forbids — re-enacting the pnpm
`ERR_PNPM_IGNORED_BUILDS` misdiagnosis that is the requirement's own cited evidence. SPEC-070's
REQ-006 splits the same check the same way for doctor; the two must not diverge. Both (b) and
(c) still exit non-zero on DD-15's posture — init promised to verify the toolchain RUNS and did
not — but the LABEL differs, and CLM-106 makes the (c) report's silence about cause a testable
denylist rather than a hope.

**What init cannot distinguish, stated rather than promised.** An earlier draft of this section
claimed that "a wrapper exiting non-zero while the real entrypoint ran fine must not produce a
false init failure, and the only way to guarantee that is to read the entrypoint's own status."
That guarantee is RETRACTED, because core cannot deliver it: a pack declares ONE command per
engine, so if the pack declares `<package-manager> <test-runner> run`, the entrypoint's own
status IS whatever the package manager reports, and there is no second status underneath for
init to read. What init actually guarantees is narrower and true — the verdict comes from the
declared entrypoint command and from nothing adjacent to it. Where the wrapper's status is
misleading, init reports case (c) and diagnoses nothing, which is why (c) prints the exit code
and the captured output verbatim and stops. Choosing a command whose exit status means what the
pack intends is the PACK AUTHOR's obligation; core's obligation is not to invent a cause it
cannot see. This is also the honest answer to the pnpm evidence: init cannot save a pack author
from a wrapper they declared, but it can refuse to tell the consumer a story about why the
command failed.

### 8. The source-file scaffold (REQ-009, REQ-019, REQ-035)

This section documents step **6**, which executes before §7's step. The document orders these two
sections by dependency (the scaffold is a recipe apply, and §10 below is where the apply mechanism
is described in full) rather than by step index; §3's table is the normative order.

DD-7 states two obligations and this spec had been carrying only one. The first — one canonical
`.gitignore` — is §5. The second is that init "scaffolds at least one source file … because a
compiler run over an empty repo can RED on 'no inputs' (observed: `tsc` TS18003)", and that "that
scaffolded file comes from a pack recipe, never from core (REQ-009)". Bundle REQ-009 says the same
thing from the other direction: "every language-specific artifact init produces (a first source
file, toolchain config) must originate in a pack recipe, never in core". The 2026-08-12 bundle
correction struck DD-7's TypeScript-flavored ignore LIST and reaffirmed the scaffold half with its
rationale; retiring bundle REQ-032 removed DD-8 step 1's dependency-install clause, not its
scaffold clause. Neither is stale, and the obligation is built here rather than deferred.

**The mechanism is a recipe apply and nothing else.** The consumer names the recipe with
`--scaffold <pack>:<recipe>@<version>`; that whole string is opaque to core and is handed to
`ParseRecipeRef` → `ResolveRecipe` → `Apply` byte-identically, exactly as `--ci` is (§10). Core
constructs no part of the ref and, critically, no part of the FILE: no template, no payload, no
filename, no extension, no path. What REQ-009 forbids is core AUTHORING a language-specific
artifact; what it REQUIRES is init DELIVERING one through a pack. Reading the prohibition as
"init produces no source file at all" is what deleted this step from an earlier draft, and
CLM-048 is re-worded so the denylist scans for authored CONTENT rather than for the file's
existence.

**Why a flag and not a capability.** `--scaffold` follows `--ci`'s governance shape for the reason
OQ-7 gave for CI: omission IS the opt-out, so there is no `--no-scaffold` to need, and adding
`scaffold` to the capability set would give one outcome (no source file scaffolded) two report
paths and two justifications. The vocabulary stays at exactly seven names (CLM-003, CLM-132), and
no `scaffold` verb enters the command tree (CLM-133).

**The three outcomes, and they mirror the CI step exactly.**

| Case | Init's report | Exit |
|---|---|---|
| `--scaffold` omitted | no source file scaffolded; names `backstop recipe apply` plus the pinned `<pack>:<recipe>@<version>` shape as the way to do it later | 0 |
| supplied, resolved, applied | the recipe's declared targets, at the recipe's declared paths | 0 |
| supplied, unresolvable | the shipped resolve error VERBATIM, attributed to the SCAFFOLD step, with no added guidance and no init-side classification | non-zero |

The omitted case is a deliberate no-op rather than a failure because not every pack ecosystem
ships a scaffold recipe, and init must not manufacture one. DD-7's failure mode does not go
unreported when the step is skipped — it arrives at §7 as REQ-011's own case (b)/(c) report, which
prints what the entrypoint did and diagnoses nothing. That is the honest routing: the empty
project is a fact about the project, not a fact init is entitled to explain.

**Preserves are classified by the shared classifier.** A scaffold recipe's `create` op in a
brownfield repo hits `preserveOrRegenerate`'s never-clobber path exactly as a CI recipe does, so
init holds ONE classifier over the three observable classes §10 describes, called by both steps
(CLM-144). Only the SENTENCE differs: the CI step's user-owned wording is a statement about a
gate ("no backstop gate was wired into this file"), and the scaffold step must never borrow it —
it states instead that the consumer's own file was left in place and the recipe's declared source
file was therefore not written (CLM-140). Class, gap-ness and exit code are identical across the
two steps; only the words the class is reported in are per-step.

**No params, to either recipe.** Neither flag carries a param surface, so a recipe declaring a
param that is required with no default cannot be applied by init at all: the shipped apply's own
error surfaces verbatim and the consumer finishes with `backstop recipe apply --param`. The
alternative — init deriving a param, most temptingly the project name it already computed for
`project:` — is refused on the same ground REQ-016 v1.2.0 refused core supplying the `<pack>:`
half of a ref: core constructing recipe input is the DD-13 bake, one layer in. CLM-136 is the
tripwire, and Review Question 6 puts the ergonomic cost in front of the founder rather than
resolving it here.

### 9. Baseline delegation (REQ-012, REQ-013)

The `baseline` step calls `BaselineSeeder.Seed` and reports what it returned. When no seeder is
available, the step reports the gap naming ISSUE-056 as its owner and does not fail the run — an
un-adopted capability is a missing benefit, not a broken promise, which is this codebase's
standing loud-≠-blocking rule and is exactly the distinction the `--ci`-omitted no-op draws.
No code in this spec writes `.backstop/baseline.json`, computes a fingerprint, or touches
`pkg/gate`; REQ-013 is satisfied entirely by that absence.

### 10. CI wiring (REQ-016, REQ-017, REQ-035)

Verified against the shipped code rather than transcribed: `recipe.ParseRecipeRef`
(`pkg/recipe/resolve.go`) accepts ONLY `<pack>:<recipe>@<version>` with a MANDATORY strict-semver
pin — "there is no 'latest', no default version, and no tolerance branch" — and
`recipe.ResolveRecipe` produces its own distinct fail-loud errors: an uninstalled pack
("is not among the installed packs", listing what IS installed), an undeclared recipe
("declares no recipe", listing what IS indexed), an unreadable/unparseable colocated manifest, and
a pin that does not equal the recipe's declared version (naming both). Each already names what was
missing and what is available, which is precisely why init adds nothing.

The step therefore does exactly three things. It hands the `--ci` value to `ParseRecipeRef`
byte-identically. On any resolve failure it surfaces that error VERBATIM, attributed to the CI
step, lets every other step complete, and makes init exit non-zero. On success it applies through
the shipped `recipe.Apply` and reads the result's `Written` and `Preserved` slices.

`Preserved` is the REQ-035 case, and it is why the apply's own success is not init's success —
but a preserve must be CLASSIFIED before it is reported, because `preserveOrRegenerate`
(`pkg/recipe/apply.go:348-390`) returns the same `PreservedDivergence` value from three
different branches:

| Producer (what the code does) | Branch | `Rule` / `CoveringWaiver` | Is a gate wired? |
|---|---|---|---|
| (a) user-owned | `!own.adopted` (`:349-355`) | both EMPTY | **no** |
| (b) one-shot materialized | `own.adopted` AND `own.kind == KindTemplating` (`:357-363`) | both EMPTY | yes — the one-shot's output is already on disk |
| (c) waiver-covered | `covered` (`:377-388`) | both POPULATED | **yes** |

**The branch order is the trap, and an earlier draft of this spec fell into it.** That draft
claimed the resolved recipe's declared `kind` separates (a) from (b). It does not.
`preserveOrRegenerate` tests `!own.adopted` FIRST (`apply.go:349`) and returns immediately, so the
kind test at `:357` is unreachable for any recipe no prior apply adopted. A `kind: templating`
recipe that was never adopted therefore takes branch (a) and returns a value byte-identical to
branch (b)'s. The adoption bit that actually separates them is recipe-level (`apply.go:166`),
keyed by the UNEXPORTED `adoptionKey`, and is carried by nothing the apply returns.

So init holds exactly two observables — the `Rule`/`CoveringWaiver` pair per preserve, and the
resolved recipe's declared `kind` — and they yield three OBSERVABLE CLASSES, not three producers:

| Observable class | Pair | Declared kind | Producers it could be | Init's report | Exit |
|---|---|---|---|---|---|
| WAIVER-COVERED | POPULATED | any | (c) only | accountable customization, naming the rule and the covering token | 0 |
| USER-OWNED | empty | `scaffolding`, `implementing` | (a) only — (b) is unreachable for these kinds | name the file, state no gate was wired, give the next action | non-zero |
| INDETERMINATE | empty | `templating` | (a) or (b) — indistinguishable | name the file, state that init CANNOT DETERMINE whether the recipe's output is present, give the next action | non-zero |

The INDETERMINATE class is an admitted unknown, and admitting it is the point. Claiming success
there would hide a real brownfield gap; asserting "no backstop gate was wired into this file"
there would be a false statement about a one-shot that already materialized — the same species of
false report REQ-035 exists to prevent, pointed the other way. DD-15's "on 'I cannot tell',
REFUSE" governs, so init scores it as a gap and exits non-zero while withholding the sentence it
cannot support. CLM-113/CLM-114 hold both halves; CLM-116 keeps the false report unrepresentable
for the waiver-covered class; CLM-123/CLM-124 pin the two unambiguous kinds; CLM-125 pins that a
populated pair outranks the kind at every kind.

**Init must not engineer around the ambiguity**, and specifically must not reconstruct the
applier's adoption bit: `adoptionKey` is unexported, so init would be writing a second derivation
of a recipe's adoption identity — the same "one authority, not a copy" hazard REQ-018 refuses for
local-path classification — and surfacing the applier's own bit would mean editing `pkg/recipe`,
which REQ-009 forbids. Both observables the classifier DOES use are already exported at HEAD, so
the seam widening this needs (`ApplyOutcome` carrying `[]recipe.PreservedDivergence` plus the
kind) is a change to THIS spec's own seam type and touches no file under `pkg/recipe`. If a later
artifact makes prior-adoption state available at this seam without either cost, INDETERMINATE
collapses into (a) and (b) and REQ-035 should be revised then, deliberately.

A partial apply — some targets written, at least one USER-OWNED or INDETERMINATE preserve — is a
gap, not a success. An apply whose every preserve is WAIVER-COVERED is a success and exits 0
(CLM-117); the accountable class is the only preserve class that leaves an apply successful.

**This classifier is init's ONLY one, and the scaffold step (§8) calls the same one.** A scaffold
recipe's `create` op hits the identical never-clobber path, so the three observable classes, the
gap-ness of each and the exit code each carries are the same at both steps; CLM-144 forbids a
second, step-local classifier on the "one authority, not a copy" ground REQ-018 applies to
local-path classification. What is per-step is the SENTENCE alone: "no backstop gate was wired
into this file" is a claim about CI wiring and belongs to the CI step's USER-OWNED class only —
the scaffold step says instead that the consumer's own file was left in place and the recipe's
declared source file was therefore not written (CLM-140). Sharing the classifier while splitting
the wording is what keeps both reports true.

With `--ci` omitted, no resolution or apply is attempted at all, init states that no CI was wired,
names `backstop recipe apply` plus the pinned ref shape, and the step contributes nothing to the
exit code.

### 11. Observation and exit-code precedence (REQ-014, REQ-015)

The `observe` step runs the gate once and reduces its findings to `[]DimensionCount` — the gate
dimension and how many findings it produced. Dimension names are backstop's universal vocabulary,
so the grouping introduces no tool or language noun. The summary is phrased as what was noticed
and carries no verdict language.

The exit code is computed from step outcomes only, never from the observation:

| Condition | Exit |
|---|---|
| Every requested step delivered; gate clean | 0 |
| Every requested step delivered; gate found pre-existing findings | 0 |
| `--ci` omitted (deliberate no-op) | 0 |
| `--scaffold` omitted (deliberate no-op, REQ-009) | 0 |
| A capability was subtracted by flag | 0 |
| Baseline seeder unavailable (capability absent, ISSUE-056) | 0 |
| Pack-only coverage-floor gap (capability absent, REQ-033) | 0 |
| No pack declares a `test`/`build` entrypoint (capability absent) | 0 |
| A preserve of the WAIVER-COVERED class, at EITHER recipe step (REQ-035) | 0 |
| A declared toolchain entrypoint could not be STARTED — case (b) (REQ-011) | non-zero |
| A declared toolchain entrypoint STARTED and exited non-zero — case (c) (REQ-011) | non-zero |
| A `--ci` ref could not be resolved (REQ-017) | non-zero |
| A `--scaffold` ref could not be resolved (REQ-009) | non-zero |
| A CI target was preserved in the USER-OWNED class, no gate wired (REQ-035) | non-zero |
| A CI target was preserved in the INDETERMINATE class, wiring unconfirmable (REQ-035) | non-zero |
| A SCAFFOLD target was preserved in the USER-OWNED class, no source file written (REQ-035) | non-zero |
| A SCAFFOLD target was preserved in the INDETERMINATE class, output unconfirmable (REQ-035) | non-zero |
| `--only` + `--no-` combined, unknown capability, local-path `--pack`, un-allowlisted entrypoint tool | 2 (config error) |

The organizing rule is bundle REQ-014's precedence clause: pre-existing findings are never an init
failure, but an init STEP that failed to deliver what it promised always is. Capability-absent
outcomes are neither — nothing promised them.

Post-init, the gate's default scope is unchanged: `cmd/backstop/gate.go` already resolves
`gate.GateScopeModeDiff` when no scope flag is given, so REQ-015 is satisfied by writing no config
that overrides it, and CLM-070/CLM-071 hold that line against regression.

## Verification

Level `integration`, because the spec spans two packages and its acceptance claims drive real
mechanisms — the shipped recipe resolve/apply path, real pack manifests, and a real gate run —
rather than stubs. Coverage threshold 80, the integration-level floor and the standing
`cmd/backstop` floor.

Test placement follows claim subject exactly, and no claim straddles packages:

- **`pkg/initialize`** (spec default subject) — step behavior, report content, file writes,
  capability resolution, the REQ-035 preserve CLASSIFICATION and its report text, and the
  structural DENYLIST scans that do not concern command execution. A structural scan reads the
  init source set by path but the TEST lives in one package, so its claim has one satisfiable
  subject.
- **`cmd/backstop`** (per-claim override) — flag parsing, config-error exits, exit-code mapping,
  headless parity, the command-tree assertion (including `--scaffold`'s, CLM-133), the scaffold
  step's exit-code claims and its three preserve-class claims, CLM-139's
  entrypoint-over-a-scaffolded-project demonstration (it executes a declared entrypoint, so it
  lives where every other REQ-011 claim does), the two REQ-019 acceptance e2e claims, and
  **every REQ-011 toolchain claim**. That last group is not a preference: the concrete
  `ToolchainProber` lives in `cmd/backstop/init_toolchain.go` because the allowlist gate and the
  command splitter it must bind to are unexported in package `main` (§7), so its tests live
  there too. `pkg/initialize` holds only the `ToolchainProber` interface, and the Runner's
  toolchain STEP is exercised through a fake prober.

Absence and structural claims carry `kind: absence` so the substantiveness noTarget join does not
demand they reference the subject — a test proving a literal is ABSENT cannot reference the thing
it proves absent.

Fixtures: the acceptance claims (CLM-091, CLM-092) stage a fresh directory and drive the built
root command end to end WITH REAL PACKS INSTALLED. That is a hard condition, not a nicety: a bare
`backstop init` with no `--pack` installs zero packs, so the toolchain step reports
capability-absent and the gate it then asserts PASS over has no pack engines to dispatch — a
"PASS with zero violations" over a packless repo would assert almost nothing.

**The pack set is named, and the previous naming was unfalsifiable.** REQ-018 forbids local-path
`--pack` values, so the acceptance runs cannot point at anything on disk; they publish a pack
SOURCE TREE as a hermetic remote (`newHermeticRemote`,
`cmd/backstop/hermetic_remote_harness_test.go` and `cmd/backstop/testdata/hermetic-remote/` — the
`GIT_CONFIG_GLOBAL` insteadOf redirect SPEC-055 built, a real clone at a real tag with no network)
and install from it by git ref.

A prior version of this spec named `packs/contracts` and `packs/substantiveness`, and that choice
made both acceptance claims unfalsifiable. Three findings, all verified against the shipped code:

1. **Their findings could not reach the verdict under the pack-only profile.** Those two packs
   declare only `gate_type: contracts` and `gate_type: substantiveness` engines.
   `gateTypeHasDedicatedStep` (`cmd/backstop/pack_gate.go:107`) returns true for exactly
   substantiveness, contracts and coverage, and `excludeDedicatedStepRules` (`:123`, called at
   `cmd/backstop/gate.go:828`) strips those rules out of the generic `pack_engines` dispatch. Their
   findings therefore arrive only through the `contract_signature` and `test_substantiveness`
   dimensions — the two REQ-003 sets to `level: off` under that profile. A fixture deliberately
   containing what their rules flag would still have PASSED, which is precisely the vacuity
   CLM-112 exists to prevent.
2. **`packs/contracts` was never dispatched in either run.** `buildContractStep`
   (`cmd/backstop/gate.go:1486`) calls `gate.ExtractContractEntries` and hands the result to
   `produceContractEngineResults` (`:1329`), whose body is `for _, c := range contracts`. A
   freshly-initialized consumer repo has zero specs, hence zero contract entries, hence zero loop
   iterations — so `grep -rn` and `ast-grep run --json` never ran at all, despite the spec
   claiming "real manifests, real commands, real dispatch".
3. **The harness constraint the spec asserted is false.** `newHermeticRemote(t, packSourceDir,
   tags...)` publishes an ARBITRARY directory, not only a tree under `packs/`, and SIX
   purpose-built fixture pack sources already live under `cmd/backstop/testdata/hermetic-remote/`
   (`valid-pack`, `fixture-fail-pack`, `invalid-pack`, `scaffold-config-pack`,
   `divergent-name-pack`, `version-drift-pack`). "Exactly three exist" counted only `packs/` and
   is struck. A test-fixture pack source is not a vendored external pack, so publishing one
   violates no packs-are-always-external invariant — those six are the standing precedent.

**So the acceptance runs install ONE purpose-built fixture pack source, and it is a seventh rather
than one of the six.** All six existing fixtures declare their engine with `command: ""`
deliberately — their manifests state the reason, which is that nothing must ever execute through
them — and an engine with no command cannot dispatch or produce a finding, so a PASS over any of
them asserts nothing. Two are additionally uninstallable: `invalid-pack` fails `pack check` and
`fixture-fail-pack` fails `pack test`, and `pack add` runs both pipelines unconditionally. The new
fixture source therefore declares:

- one engine with `gate_type: lint` — a type with NO dedicated step, so its rules survive
  `excludeDedicatedStepRules` and dispatch through the generic `pack_engines` path, a dimension
  neither profile disables (the pack-only profile turns off exactly the five SDLC dimensions, and
  `pack_engines` is not among them). This is the property that makes the same engine live under
  BOTH acceptance profiles;
- a REAL command built from an allowlisted, hermetic, no-provisioning tool with a pack-declared
  convert normalizing its output to SARIF — the shape `packs/contracts`' own `grep` engine already
  proves end to end in this repository (`command: grep -rn`, `input_mode: pattern-arg`,
  `input_flag: -e`, `convert: grep/to-sarif.sh`, `provision: {tool: grep, version: "*"}`), lifted
  to `gate_type: lint`. Nothing is downloaded and no network is touched;
- one rule carrying a distinctive marker pattern that appears nowhere in a freshly-initialized
  project, so the acceptance run finds nothing — and fires when a fixture project contains it.

That last property is CLM-112's second and third parts: the dispatched engine is demonstrably able
to go red, proven under BOTH profiles, so each acceptance PASS is a pack that COULD have failed
and did not, never a pack that could never fail. One consequence is deliberate and stated: the
fixture pack declares no `test` or `build` engine, so the acceptance runs' toolchain step reports
capability-absent, and REQ-011's execution path is verified by its own `cmd/backstop` claims
against manifests that DO declare those gate types. REQ-019 records the remaining shortfall against
DD-8's five-pack transcription rather than pretending it away.

**The scaffold claims need two fixtures beyond the acceptance pack, and both are real rather than
stubbed.** CLM-130 (the recipe's declared targets land at the recipe's declared paths) needs a
pack declaring a scaffold RECIPE — one `create` op with a payload and a declared target — and the
existing `scaffold-config-pack` fixture is NOT it: its `scaffolds:` block is packval's
`tier: complete` scaffold declaration, a different mechanism from `pkg/recipe`'s recipes. CLM-139
(DD-7's evidence) needs one pack that declares BOTH a scaffold recipe and a `gate_type: test` or
`build` engine whose command exits non-zero over a project with no source file and zero over the
same project once the recipe has run — the pack author's choice of command is what encodes the
"no inputs" failure, and core supplies none of it. Both fixtures are pack DATA under
`cmd/backstop/testdata/`, so neither puts a language or filename literal in the init source set
that CLM-131 scans.

Other pack-dependent claims resolve REAL installed pack manifests where one is
available and otherwise use manifest fixtures written in the pack's own declared YAML shape —
never a fabricated Go struct that bypasses `pack.ParseManifest`, so a manifest-schema change
cannot leave these tests passing against a shape no pack can produce.

**Four claims are EXPECTED RED until SPEC-068 lands, and they go red for two different reasons.**
An implementer must not read either failure as their own bug:

- **CLM-021, CLM-022, CLM-026** fail at CONFIG LOAD. The config init writes carries
  `artifact_root`, which is rejected TWICE today — once by `config.Load`'s strict typed decode,
  and once by `artifacts/backstop-yml/v1/schema.json`'s `additionalProperties: false` over the
  six declared top-level properties. SPEC-068 REQ-006 adds the key to both.
- **CLM-103** fails at DISCOVERY, not at load: `.backstop` sits on the unconditional skip list at
  `cmd/backstop/artifact_discover.go:47-49`, so artifacts under the root init creates are not
  found at all.

Naming all four (rather than only CLM-103, the deliberate tripwire) is the point: presenting one
red and leaving three unexplained would make an implementer debug a `config.Load` failure that is
a stated prerequisite.

## Sharp Edges

1. **ISSUE-119 (open) — brownfield CI adoption reports success and wires nothing.** Init is this
   gap's primary consumer. Three of the four shipped CI platforms (`.gitlab-ci.yml`,
   `bitbucket-pipelines.yml`, `Jenkinsfile`) have exactly ONE conventional entry point apiece, so
   every existing project adopting backstop hits the never-clobber preserve; only GitHub's
   per-workflow-file layout escapes it. REQ-035 makes init report the outcome honestly, but it
   does NOT fix it — a brownfield consumer still ends up with no gate in CI until ISSUE-119's
   merge/insert-op work lands. The temptation this spec must resist is closing the gap by having
   init edit the consumer's existing CI file itself; that would put init in the business of
   rewriting foreign config, which REQ-009 forbids outright.

2. **ISSUE-081 Gap 3 (open) — `insert` op placement semantics are unpinned.** `applyInsert`
   (`pkg/recipe/apply.go:639`) splices a snippet inline immediately after the anchor text,
   producing artifacts like `"registrations": [    "live-entry",` on one line; newline handling is
   left for the pack author to discover by trial. Init must not depend on `insert` placement for
   any step it promises to deliver, because the output is currently unpredictable. This is a live
   constraint on any future init step that wants to extend an existing file rather than write a
   whole one.

3. **ISSUE-110 (open) — no escape syntax for foreign `{{ }}` template content.**
   `pkg/recipe/substitute.go` reads EVERY `{{ ... }}` span in a payload as a param name and
   hard-fails on any name no param declares, emitting nothing at all rather than partial output —
   and it fires even inside an explanatory comment. CI templates for platforms whose own syntax
   uses `{{ }}` cannot currently quote those spans. Init inherits this whole-cloth: a `--ci` ref
   pointing at such a recipe fails at substitution time, and init's only correct move is to
   surface that failure, not to work around it.

4. **The artifact-root key name couples this spec to SPEC-068 — RESOLVED, the spelling is
   confirmed.** REQ-004 requires init to write the artifact-root configuration key, which is
   DEFINED by SPEC-068 (bundle REQ-029) and does not exist in
   `artifacts/backstop-yml/v1/schema.json` today. The spelling is no longer an open coupling:
   SPEC-068 v1.2.0's REQ-006 states verbatim that "`backstop.yml` gains a top-level
   `artifact_root` key, in both the JSON schema (which is `additionalProperties: false`, so an
   unlisted key is rejected) and the typed loader", and its contracts carry
   `Config.ArtifactRoot` with the `yaml:"artifact_root"` tag. CLM-026 and the config round-trip
   claims are therefore written against a CONFIRMED spelling, not a guessed one. What survives
   as the sharp edge is the SEQUENCING, which is not optional: init that scaffolds `.backstop/`
   before discovery can resolve it manufactures exactly the silent-undiscovery false green DD-15
   exists to prevent. If SPEC-068 changes the spelling before it lands, those claims break
   together — loud, at test time — and must be reconciled rather than patched around.

5. **`.backstop` is on a hardcoded discovery SKIP LIST today, so the layout REQ-004 mandates is
   currently invisible — verified, not inferred.** `cmd/backstop/artifact_discover.go:47-49`
   skips directories named `testdata`, `vendor`, `node_modules`, `.git`, `.backstop`,
   `prototype` unconditionally (the `switch base` at `:47`, its literals at `:48`, the
   `SkipDir` at `:49`), and `cmd/backstop/artifact_validate.go:132-133` returns
   `ValidateResult{Pass: true}` on zero discovered artifacts. The same discovery feeds the gate
   (`cmd/backstop/gate.go:1015`). Composed, that means a consumer repo laid out exactly the way
   init is required to lay it out has every artifact skipped and BOTH `artifact validate` and
   the gate's artifact steps report green having checked nothing — the silent-undiscovery false
   green DD-15 exists to prevent, manufactured by init itself. The fix belongs to SPEC-068
   (bundle REQ-029/REQ-030, one config-resolved root shared by gate, validate, and scaffold) and
   is NOT specced here. What is specced here is the tripwire: CLM-103 fails until the layout init
   creates is actually discoverable, so this spec cannot be implemented into a vacuous green.
   Note also, for whoever owns that file: `node_modules` in that same skip list is a baked
   ecosystem literal in core CLI code, which DD-13 forbids. That is out of this spec's lane and
   is ALREADY FILED as **ISSUE-122** (`issues/ISSUE-122-baked-ecosystem-literals-in-artifact-
   discover.issue.md`), homed under DIR-002 and sequenced after SPEC-068 — no new issue is
   needed and none should be filed.

6. **REQ-033's dependency is genuinely dangling and this spec does not close it.** No artifact
   anywhere owns the spec-independent coverage-floor knob. The failure mode to guard against is
   quiet scope creep: a future implementer reading "wire a coverage floor" and adding
   `enforcement.coverage.min_pct` to the schema to make the requirement satisfiable. That would be
   init designing an enforcement-policy surface, which the bundle explicitly excludes, and it
   would also be a schema change made from the wrong artifact. CLM-093 is the tripwire —
   it fails the moment init writes an `enforcement` key the shipped schema does not declare.

7. **"Omakase base bundle" does not mean a baked pack roster, and the reading is load-bearing.**
   Bundle REQ-002 says bare init installs "the full opinionated base bundle", which reads at first
   glance as a default set of PACKS. It cannot be: the unit REQ-002's own subtraction flags
   operate on is `<cap>`, a capability; REQ-010 says packs enter only via an explicit act and init
   must never select packs on inspected identity; and REQ-016 v1.2.0 established the precedent
   that core may hold NO pack name at all — that correction exists precisely because core
   supplying the `<pack>:` half of a recipe ref was itself the DD-13 bake. A default pack roster
   in core would be the same defect, one layer up. So the base bundle is the seven-capability
   STEP set, and pack refs come from the consumer. This is the single most consequential
   interpretation in the spec and it is called out in Review Questions for an explicit ruling.

8. **`--no-ci` deliberately does not exist, and its absence is easy to "fix" wrongly.** OQ-7 is
   explicit: omission IS the opt-out, so there is no `--no-ci` to need. Adding `ci` to the
   capability set would give one outcome (no CI wired) two different report paths and two
   different justifications, and a future reader would reasonably assume symmetric flags. CLM-003
   pins the capability set at exactly seven names, which fails if `ci` is added.

9. **The toolchain check's exit-code posture is the one place this spec chose between two defensible
   readings — and it is now decoupled from the LABEL, which was a separate defect.** REQ-011 says
   an uninstalled toolchain is reported as "a SETUP step the consumer still owes", never silently
   passing — which could be read as a warning. This spec exits non-zero for BOTH failure cases,
   on DD-15's "on 'I cannot tell', REFUSE" posture: init promised to verify the toolchain RUNS,
   and it did not verify it. The consequence is real and worth stating plainly — a greenfield
   consumer who runs `backstop init` before installing project dependencies gets a non-zero init.
   That is the intended signal, but it is a UX cost and a founder could rule the other way.
   What must NOT be collapsed along with it is the (b)/(c) distinction (§7): the exit code being
   the same for both is exactly what makes it tempting to give them the same label, and an
   earlier draft did, which is how "any non-zero exit is owed setup" got written — the precise
   exit-code cause-inference bundle REQ-011's first sentence forbids. If the founder rules the
   exit code down to a warning, the labels still stay split, because they answer a different
   question (WHAT HAPPENED) than the exit code does (DID INIT DELIVER).

10. **Absence claims are only as good as their scan boundary.** Every DENYLIST claim here scans "the
   init source set" — `cmd/backstop/init*.go` plus `pkg/initialize/**`, excluding `_test.go`. A
   future implementer who moves init logic into a helper file outside that glob (a new
   `pkg/initcore/`, a function parked in `cmd/backstop/root.go`) silently empties the claims
   without failing them. The scan must be defined by a package/prefix set the implementation
   cannot drift out of without also failing an import-graph assertion.

11. **Two structurally-identical no-ops carry different exit codes and the distinction must survive
    refactoring.** `--ci` omitted (exit 0) and `--ci` supplied-but-unresolvable (non-zero) both end
    with no CI wired. So do "no packs supplied" (exit 0) and "pack ref refused as local path"
    (exit 2). The organizing rule — did the consumer ASK for something init then failed to deliver
    — is not visible in the file system state afterward, only in the invocation. A refactor that
    computes the exit code from the RESULT rather than from the request will collapse these pairs.
    The REQ-035 class matrix is the same hazard in a second place: WAIVER-COVERED, USER-OWNED and
    INDETERMINATE preserves all leave a file on disk that init did not write, and only the CLASS
    separates an accountable customization from a broken promise from an unresolvable unknown.

12. **DECLARED CROSS-SPEC SEAM: SPEC-070 owns `doctorGuidanceForSteps` inside
    `cmd/backstop/init.go`, and the residual hazard is CONCURRENCY, not ownership.** SPEC-070
    v1.1.0 DECLARES `cmd/backstop/init.go` in its own contracts, providing
    `doctorGuidanceForSteps(steps []initialize.StepReport) []string`, with a notes block that
    names SPEC-069 explicitly, explains why the function lives in package `main`
    (`doctorRegistry` is unexported there and `pkg/initialize` importing it would invert the
    dependency), and defers to this spec's `TestInit_ImplementsNoCIDetectionOrBespokeGuidance` as
    the guard keeping doctor guidance off init's CI steps. This spec's own contracts declare the
    same file. The ownership question is therefore SETTLED and the earlier reading — that the
    seam was undeclared and that SPEC-070's contracts listed only `doctor.go`,
    `doctor_checks.go` and `root.go` — is superseded. SPEC-070 mandates THREE tests asserting
    INIT's output, not two: `TestInit_ToolchainFailureNamesTheToolchainRunsCheckID`,
    `TestInit_DoctorCheckIDsResolveToRegisteredChecks`, and
    `TestInit_NoDoctorGuidanceForStepsNoRegisteredCheckDiagnoses`. What REMAINS sharp is that two
    specs declare one file as contract surface and can be implemented concurrently: SPEC-070
    writes the doctor-guidance renderer into `cmd/backstop/init.go` while this spec writes the
    command, its flags, its report rendering and its exit-code mapping into the same file, so
    concurrent implementation collides in the editor even though neither spec's scope overlaps.
    Sequencing is the mitigation — whichever lands second edits a file the first already created
    — and the split of responsibility inside it is fixed: doctor guidance attaches to the
    TOOLCHAIN step only and must never reach the CI steps (SPEC-070 REQ-004, guarded by this
    spec's CLM-085). A second, quieter divergence rides here too: SPEC-070's
    `checkToolchainRuns` owes the same verbatim capture for the same non-zero-exit case but does
    not name a runner method, so if it binds `RunStdout` while init binds `Run` (§7), the two
    reports disagree on any entrypoint that diagnoses only on stderr.

13. **A pack author's wrapper is a limit on init, not a bug in it.** §7 retracts the guarantee an
    earlier draft made — that init can tell "the wrapper failed" from "the toolchain failed" —
    because a pack declares one command per engine and core reads one exit status. The temptation
    to "fix" this is to add wrapper-aware parsing of captured output, or a second probing command,
    or a package-manager-shaped retry. All three bake ecosystem knowledge into core and all three
    would be caught by CLM-044 or CLM-109 only if the literal happened to be recognizable — so
    the durable guard is the posture, not the scan: case (c) reports and does not diagnose.

14. **The preserve classifier is honest about an ambiguity it cannot close, and the tempting
    "fix" reintroduces a second authority.** §10 records why the declared recipe `kind` cannot
    separate producer (a) from producer (b): `preserveOrRegenerate` returns on `!own.adopted`
    before it ever reads the kind, so a never-adopted `kind: templating` recipe and an
    adopted-and-materialized one produce identical values. A future reader will find the
    INDETERMINATE class unsatisfying and reach for one of three closures, and all three are
    wrong here. Reading `backstop-recipes.lock` through the exported `recipe.ReadAdoptions`
    requires reproducing the applier's UNEXPORTED `adoptionKey` derivation, which is a second
    definition of a recipe's adoption identity — the hazard REQ-018 refuses for local-path
    classification. Adding the adoption bit to `recipe.PreservedDivergence` edits `pkg/recipe`,
    which REQ-009 forbids. Defaulting the class to (b) so init exits 0 is the silent success
    DD-15 exists to prevent. What is legitimate is a later artifact that makes prior-adoption
    state available at this seam properly, at which point REQ-035 should be revised on purpose
    rather than reinterpreted quietly.

15. **The one method that differs from `runFindingsEngine` is invisible in a diff review.** §7
    binds init's execution to the same allowlist gate, the same splitter and the same runner as
    the gate's engine dispatch — and then departs on a single call, `Run` instead of
    `RunStdout`. An implementer copying the sequence will copy the method with it, and every
    structural claim still passes: the allowlist is bound, no shell is invoked, no second
    execution path exists, the exit code is right. Only the CONTENT of case (c)'s report goes
    empty, and only for entrypoints that diagnose on stderr, which is most of them. CLM-122 is
    the tripwire and it must assert a stderr-ONLY failure; a fixture that also writes to stdout
    passes under either method and asserts nothing.

16. **The acceptance pack's `gate_type` is load-bearing, not metadata, and re-tagging it silently
    re-vacuums both acceptance claims.** `lint` was chosen because `gateTypeHasDedicatedStep`
    (`pack_gate.go:107`) excludes substantiveness/contracts/coverage from the generic dispatch,
    and because the pack-only profile disables the five SDLC dimensions but not `pack_engines`.
    Those two facts together are the ONLY reason a finding from this pack can reach the verdict
    under both profiles. A later reader tidying the fixture toward "a more realistic pack" by
    tagging it `contracts` or `substantiveness` reproduces exactly the defect this revision
    fixes — and reproduces it INVISIBLY, because both acceptance tests still pass. Anyone
    changing that field must re-derive the routing, not just the manifest.

17. **The acceptance marker must not appear anywhere the gate scopes — including the pack's own
    manifest, which necessarily contains it.** The rule's pattern is declared in the pack's
    `pack.yml`, which lands under `.backstop/packs/`. The `.gitignore` step (step 5) is what
    keeps that tree out of the observe step's scope, so the acceptance PASS quietly depends on a
    step the same run performed. A run that subtracts the `gitignore` capability, or a scope
    change that stops honoring ignore entries, would have the pack find its own marker and the
    acceptance claim would go red for a reason having nothing to do with init. The marker string
    must be chosen so this is detectable rather than confusing, and the dependency is recorded
    here rather than discovered later. A second, quieter host dependency rides along: a
    grep-shaped convert normalizes tool output to SARIF with ordinary POSIX text tools, which is
    the same dependency `packs/contracts` already carries in this repository's own suite — real,
    already-paid, and worth knowing about before a CI image is trimmed.

18. **Scaffold-before-toolchain is invisible in a step list, and the obvious refactor breaks it.**
    Grouping "the two recipe steps" together — moving `scaffold` down beside `ci` at step 9 —
    reads as a tidy-up and passes any assertion that merely compares a list of step names in
    order, because the list is still internally consistent. What it destroys is the reason the
    step exists: the toolchain step would then run over a project with no source file, which is
    DD-7's cited `tsc` TS18003 failure, manufactured by init. This is why CLM-138 asserts the
    file is ON DISK at the toolchain step's boundary and CLM-139 asserts the entrypoint's verdict
    flips, rather than either asserting the order alone.

19. **The scaffold step's USER-OWNED sentence is one copy-paste from being false.** CLM-144
    requires ONE preserve classifier shared by both recipe steps, which makes sharing the report
    STRING look like the same kind of de-duplication. It is not: "no backstop gate was wired into
    this file" is an assertion about CI wiring, and the scaffold step has no knowledge of CI at
    all. A shared string would make init tell a consumer something it cannot know about a file it
    was never asked to wire a gate into — the same species of false report REQ-035 exists to
    prevent, in a third place. CLM-140 asserts the scaffold wording specifically, and the split is
    stated in REQ-035 so it survives a refactor that unifies the classifier further.

20. **`--scaffold` carries no recipe params, and the tempting fix is a bake.** A scaffold recipe
    that declares a param required with no default cannot be applied through init at all; the
    consumer must finish with `backstop recipe apply --param`. The fix that looks free is for
    init to pass the project basename it already computed for `project:` — and that is core
    constructing recipe INPUT, which is the same defect one layer in as core constructing the
    `<pack>:` half of a ref, the thing REQ-016 v1.2.0 was corrected to remove. CLM-136 is the
    tripwire. This is a genuine ergonomic cost rather than a free win, which is why Review
    Question 6 puts it in front of the founder instead of resolving it here.

21. **A requirement can be deleted by a rewording, and this spec proves it.** Spec REQ-009 had
    quietly narrowed bundle REQ-009's "every language-specific artifact INIT PRODUCES (a first
    source file, toolchain config)" to "every artifact a consumer ends up with" — a phrase that
    reads as a faithful paraphrase and is not, because it moves the obligation from init to
    whoever else happens to write the file. CLM-048 then hardened the narrowed reading into a
    denylist ("init writes no source file of its own"), at which point the missing step had a
    passing TEST guarding its absence. Nothing in the corpus went red. The durable guard is not a
    scan but a habit: when a spec requirement paraphrases a bundle requirement, the paraphrase is
    a diff to be reviewed, and a denylist claim derived from a paraphrase inherits whatever the
    paraphrase dropped.

## Review Questions

1. **Is the capability-set reading of "omakase base bundle" the intended one?** This spec resolves
   bundle REQ-002's "full opinionated base bundle" as the seven-capability step set, with pack
   refs supplied by the consumer via `--pack` and bare init installing zero packs (reported, not
   silent). The alternative — a default pack roster in core — would put pack-name literals in core
   CLI code, which REQ-016 v1.2.0 corrected against for exactly this reason. Does the founder
   affirm this reading, or is there a third source for the base roster (e.g. a bootstrap recipe
   ref the consumer passes) that should be specced instead?

2. **Does the toolchain-check failure really warrant a non-zero init exit?** Sharp Edge 9 states
   the tradeoff: DD-15's refuse posture argues yes, but the practical consequence is that a
   greenfield consumer who runs init before installing dependencies sees a failing first
   experience — which is the exact emotional outcome DD-3 was written to avoid. Should this be a
   loud exit-0 warning instead? Note the question is narrower than it was: it asks ONLY about the
   exit code, and specifically about case (b) (the executable could not be started, which is the
   uninstalled-dependencies case). Case (c) — a toolchain that ran and failed — is a genuine red
   the consumer should see regardless, and the (b)/(c) LABELS are not on the table either way,
   since bundle REQ-011's own text forbids collapsing them.

3. **CLOSED — SPEC-068 names the key `artifact_root`, verified, no ruling needed.** This question
   asked whether CLM-026 and the config round-trip claims were written against a guessed
   spelling. They are not. SPEC-068 v1.2.0's REQ-006 states it verbatim — "`backstop.yml` gains a
   top-level `artifact_root` key, in both the JSON schema (which is `additionalProperties:
   false`, so an unlisted key is rejected) and the typed loader" — and its contracts declare
   `Config.ArtifactRoot` with the `yaml:"artifact_root,omitempty"` tag. Recorded as answered
   rather than deleted so a later reader does not re-open it; the residual coupling is sequencing
   only and is carried by Sharp Edge 4 and the Dependencies section.

4. **Is the `sdlc` capability the right carrier for the profile fork?** REQ-003's fork is
   config-shaped and REQ-004's is layout-shaped, and this spec ties both to one capability so that
   `--no-sdlc` is the whole pack-only profile. The alternative is an explicit `--profile` selector,
   which REQ-002's "subtraction flags are the ONLY way to narrow" appears to forbid. Confirm the
   subtraction framing is intended to carry the profile fork.

5. **How should an implementer prove the DENYLIST scans cannot be evaded?** Sharp Edge 10 names the
   drift risk. Should the scan boundary be pinned by an import-graph assertion (nothing outside
   `pkg/initialize` and `cmd/backstop/init*.go` may be reachable from the init command's entry),
   or is a path glob plus review sufficient?

6. **Should `--scaffold` be able to pass recipe params?** REQ-009 gives it the same no-param
   posture `--ci` has, so a scaffold recipe declaring a param that is required with no default is
   inapplicable through init and the consumer must finish with `backstop recipe apply --param`.
   The alternative is a repeatable passthrough flag (`--scaffold-param k=v`, values supplied
   wholly by the consumer, zero core-side construction), which stays inside DD-13 but adds a
   surface and breaks symmetry with `--ci` unless `--ci` gains one too. This spec chose the
   narrow, symmetric option and recorded the cost (Sharp Edge 20). Founder call: accept the
   limitation, or add the passthrough to BOTH flags?

7. **Is a flag the right governance for the scaffold step, or should it be an eighth
   capability?** This spec models `--scaffold` on `--ci`: presence is the opt-in, there is no
   `--no-scaffold`, and the capability vocabulary stays at seven (CLM-003, CLM-132). The argument
   for a capability is that scaffolding is a thing init DOES, like `gitignore` or `layout`,
   whereas CI is a thing init WIRES elsewhere. The argument against — and the one taken — is that
   a capability with no consumer-supplied ref has nothing to do, so `--no-scaffold` and an omitted
   `--scaffold` would be two flags expressing one outcome, which is precisely the two-report-paths
   defect Sharp Edge 8 records for `--no-ci`. Confirm the flag framing.

8. **`follows` bindings were deliberately omitted.** No standards corpus is discoverable in this
   repo — `standards/core/rules/` contains only a `.gitkeep`, and standards live in packs — so
   binding requirements to `STD-*` rule IDs would mean inventing rule identifiers rather than
   citing them. Per the escalation-over-guessing rule this spec escalates instead: if a standards
   pack should be bound here, name it and the bindings will be added.

## Dependencies

- **SPEC-068 (Trustworthy-Green Guards) — HARD PREREQUISITE, and the blocking condition is
  MEASURED, not assumed.** Owns bundle REQ-029's config-resolved artifact root, which REQ-004
  writes and depends on. Today `.backstop` sits on an unconditional discovery skip list
  (`cmd/backstop/artifact_discover.go:47-49`, literals at `:48`) and zero discovered artifacts
  returns `Pass: true` (`cmd/backstop/artifact_validate.go:132-133`), so the layout init is required to
  create is invisible to both `artifact validate` and the gate. FOUR claims go RED against the
  current binary, for TWO distinct reasons, and all four are expected: **CLM-021, CLM-022 and
  CLM-026** fail at CONFIG LOAD, because `artifact_root` is rejected twice today — by
  `config.Load`'s strict typed decode and by the schema's `additionalProperties: false` over its
  six declared top-level properties — and **CLM-103** fails at DISCOVERY, on the skip list above.
  SPEC-068 REQ-006 adds the key to both the schema and the typed loader and supplies the one
  config-resolved root. Init must not ship until SPEC-068 turns all four green; an implementer
  seeing any of them red before that is seeing the prerequisite, not a defect in their work.
- **SPEC-054 (Recipe Apply And Manifest) — DELIVERED (`implemented`).** Supplies
  `ParseRecipeRef`/`ResolveRecipe`/`Apply`, which REQ-009 and REQ-016 consume unchanged.
- **SPEC-067 (CI Recipe Pack) — DELIVERED (`implemented`).** Published as
  `backstop-ai/ci-workflows@v0.1.0`; the concrete recipes a consumer may name in `--ci`. Core
  holds no reference to it.
- **ISSUE-056 (open) — owns REQ-012 and REQ-013.** Init provides a delegation seam; until the
  issue lands, the baseline step reports capability-absent.
- **ISSUE-055 (open) — owns REQ-018's local-provenance half.** Init refuses local-path pack refs
  in the meantime.
- **SPEC-070 (Backstop Doctor) — CONCURRENT, sharing one DECLARED file.** Not a prerequisite.
  SPEC-070 v1.1.0 declares `cmd/backstop/init.go` in its own contracts and owns
  `doctorGuidanceForSteps` there; this spec declares the same file and owns the command, its
  flags, its report rendering and its exit-code mapping. Ownership is settled on both sides, so
  what is left is a concurrency hazard, not an undeclared seam. SPEC-070 mandates THREE tests
  asserting INIT's own output — `TestInit_ToolchainFailureNamesTheToolchainRunsCheckID`,
  `TestInit_DoctorCheckIDsResolveToRegisteredChecks`, and
  `TestInit_NoDoctorGuidanceForStepsNoRegisteredCheckDiagnoses` — and confines that guidance to
  the toolchain step, deferring to this spec's `TestInit_ImplementsNoCIDetectionOrBespokeGuidance`
  (CLM-085) to keep it off the CI steps. The two also share one execution route by design —
  SPEC-070's `checkToolchainRuns` and this spec's `packToolchainProber` both bind
  `checkEngineToolAllowed` → `splitCommand` → `check.CommandRunner` — with one open detail:
  SPEC-070 names no runner METHOD, and this spec binds `Run` rather than `RunStdout` (§7). See
  Sharp Edge 12; the landing order is a sequencing call, not a spec edit.
- **ISSUE-122 (open, homed under DIR-002, sequenced after SPEC-068)** — owns the baked ecosystem
  literals (`node_modules`) on the discovery skip list this spec's Sharp Edge 5 observes. Not
  this spec's to fix and already filed.
- **REQ-033's coverage-floor knob — NO OWNER.** Documented gap; init reports rather than wires.

## References

- `bundles/BUNDLE-003-onboarding-experience.bundle.md` (v0.10.2, `defined`) — the source bundle;
  the `backstop init` seed and its 21 requirements.
- `directives/DIR-002-backstop-init.directive.md` — the directive this spec delivers against. Note
  the bundle's own recorded observation that DIR-002's prose still describes baked language
  detection and needs a dated correction through the directive track; this spec does not edit it.
- `issues/ISSUE-056-local-first-baseline-seeding.issue.md` (open) — owner of REQ-012/REQ-013.
- `issues/ISSUE-055-local-provenance-cache-for-local-packs.issue.md` (open) — owner of REQ-018's
  local-provenance half.
- `pkg/recipe/resolve.go` — `ParseRecipeRef` (mandatory strict-semver pin) and `ResolveRecipe`
  (the four fail-loud resolve errors REQ-017 surfaces verbatim).
- `pkg/recipe/apply.go` — `Apply`, `ApplyResult`, `PreservedDivergence` (with the `Rule` /
  `CoveringWaiver` pair REQ-035 classifies on), `preserveOrRegenerate` at `:348-390` (the THREE
  branches that produce a preserve: `!own.adopted` at `:349-355`, `KindTemplating` at `:357-363`,
  waiver-covered at `:377-388` — note the ORDER: the adoption test returns before the kind test
  is reached, which is why the kind cannot separate the first two), and `applyInsert` (line 639,
  the unpinned-placement sharp edge).
- `pkg/recipe/manifest.go` — `RecipeManifest.Kind` and the three declared kinds
  `KindScaffolding` / `KindImplementing` / `KindTemplating` (`:19-21`), REQ-035's second
  observable. It is a PARTIAL discriminator: it settles the empty-pair case for the first two
  kinds and leaves `templating` INDETERMINATE, because the branch that would have used it is
  unreachable for an unadopted recipe (§10).
- `cmd/backstop/pack_gate.go` — `checkEngineToolAllowed` (`:812`, the trusted-tool allowlist plus
  lock-pin trust gate), `splitCommand` (`:887`), and `runFindingsEngine` (`:573-600`), whose
  allowlist → split → runner sequence REQ-011 binds init's toolchain execution to.
- `pkg/check/runner.go` — `CommandRunner` / `ExecCommandRunner`, the shared execution seam; no
  shell is involved at any point on this path. `Run` (`:17`) returns COMBINED stdout+stderr and is
  documented as the build/test executors' method; `RunStdout` (`:21`) returns stdout only so
  stderr cannot corrupt SARIF bytes. Init's toolchain step binds `Run`; `runFindingsEngine` binds
  `RunStdout` (`pack_gate.go:648`). That is the single difference between the two paths (§7).
- `pkg/pack/distribution/add.go:96` — `isLocalPath`, the shipped local-vs-remote classifier
  REQ-018 exports and reuses rather than reimplementing.
- `cmd/backstop/hermetic_remote_harness_test.go` (`newHermeticRemote`) and
  `cmd/backstop/testdata/hermetic-remote/` — the `GIT_CONFIG_GLOBAL` insteadOf redirect SPEC-055
  built; how CLM-091/CLM-092 install a REAL pack by git ref with no network and without violating
  REQ-018's local-path refusal. The constructor takes an ARBITRARY `packSourceDir`, so the
  acceptance pack set is NOT bounded to the trees under `packs/` — the six fixture pack sources
  already in that directory are the precedent, and the acceptance runs add a seventh.
- `cmd/backstop/testdata/hermetic-remote/{valid-pack,fixture-fail-pack,invalid-pack,
  scaffold-config-pack,divergent-name-pack,version-drift-pack}` — the six existing fixture pack
  sources. Every one declares its engine with `command: ""` on purpose (their manifests state the
  reason), so none can dispatch or produce a finding; `invalid-pack` fails `pack check` and
  `fixture-fail-pack` fails `pack test`, so neither can even be installed by `pack add`. This is
  why the acceptance runs need a new fixture source rather than reusing one.
- `packs/contracts/pack.yml` — NOT an acceptance pack (its `gate_type: contracts` engines are
  excluded from the generic dispatch and disabled outright under the pack-only profile), but the
  reference implementation of the ENGINE SHAPE the acceptance fixture copies: an allowlisted,
  no-provisioning `grep -rn` command with `input_mode: pattern-arg`, `input_flag: -e` and a
  pack-declared `grep/to-sarif.sh` convert normalizing matches to SARIF.
- `cmd/backstop/pack_gate.go:107` (`gateTypeHasDedicatedStep`) and `:123`
  (`excludeDedicatedStepRules`), called at `cmd/backstop/gate.go:828` — the routing that strips
  substantiveness/contracts/coverage rules out of the generic `pack_engines` dispatch, and
  therefore the reason the acceptance fixture's engine declares `gate_type: lint`.
- `cmd/backstop/gate.go:1486` (`buildContractStep`) and `:1329`
  (`produceContractEngineResults`, whose body is `for _, c := range contracts`) — the per-entry
  contracts dispatch that runs zero times in a repo with zero specs, which is why the previous
  acceptance pack was never dispatched at all.
- `artifacts/backstop-yml/v1/schema.json` `enforcement.policy` — keyed by gate DIMENSION
  (`pack_engines`, `coverage_threshold`, …). The pack-only profile writes `level: off` for the
  five SDLC dimensions and not for `pack_engines`, which is the second half of why a `lint`
  engine stays live under both acceptance profiles.
- `bundles/BUNDLE-003-onboarding-experience.bundle.md` DD-7 and DD-8 step 1 — the scaffold
  obligation REQ-009 restores ("scaffolds at least one source file", "comes from a pack recipe,
  never from core"), its evidence (`tsc` TS18003 over an empty repo), and the 2026-08-12
  correction that struck DD-7's ignore LIST while reaffirming this half.
- `pkg/recipe/adoption.go` — `AdoptionRecord` / `ReadAdoptions` and the UNEXPORTED `adoptionKey`
  (`apply.go:1252`) behind the applier's recipe-level adoption bit (`apply.go:166`); the reason
  REQ-035's INDETERMINATE class stays open rather than being closed by a second derivation.
- `issues/ISSUE-122-baked-ecosystem-literals-in-artifact-discover.issue.md` (open, DIR-002,
  sequenced after SPEC-068) — owns the `node_modules` skip-list literal Sharp Edge 5 observes.
- `cmd/backstop/recipe_apply.go` — the shipped CLI shape init's CI step mirrors.
- `pkg/pack/manifest.go` — `EngineSpec.StdoutArtifact` (line 97), the only generated-output
  declaration a manifest carries.
- `pkg/pack/engine/gatetype.go` — the seven neutral gate-type stages; `test` and `build` are the
  entrypoint selectors REQ-011 uses.
- `pkg/pack/distribution/add.go` — `ensureGitignore` (the append-only posture REQ-005 extends) and
  the git-ref install path REQ-018 requires.
- `pkg/pack/distribution/command.go` — `NewAddCommand`, the fail-closed constructor pattern
  `NewRunner` follows.
- `artifacts/backstop-yml/v1/schema.json` — `enforcement` with `additionalProperties: false`; the
  evidence that REQ-033's knob cannot be written without a schema change.
- `pkg/config/config.go` — `Config`, `Enforcement`, `DimensionPolicy`, and the retired-`language:`
  note SPEC-046 left behind.
- `cmd/backstop/gate.go` — `gate.GateScopeModeDiff` as the shipped no-flag default REQ-015 holds,
  and `:1015`, where the gate consumes the same `DiscoverArtifacts` the skip list governs.
- `cmd/backstop/artifact_discover.go:47-49` — the unconditional directory skip list (literals at
  `:48`) that makes the `.backstop/` layout REQ-004 mandates invisible today; CLM-103's tripwire
  points here. The `node_modules` / `vendor` entries on that same line are baked ecosystem
  literals raised separately (see Sharp Edge 5) and are not this spec's to fix.
- `cmd/backstop/artifact_validate.go:132-133` — `ValidateResult{Pass: true}` on zero discovered
  artifacts, the second half of the vacuous-green composition.
- `specs/SPEC-068-trustworthy-green-guards.spec.md` — the prerequisite seed.
- `specs/SPEC-070-backstop-doctor.spec.md` — the sibling seed init delegates diagnosis to.
