---
title: "Backstop Doctor"
number: SPEC-070
created: "2026-08-13"
updated: "2026-08-16"
status: implemented
schema_version: spec/v1
spec_version: 1.1.6

implementation:
  summary: >
    BUNDLE-003's `backstop doctor` seed: the diagnostic command init delegates to
    when a setup is off. Four bundle requirements, three of which are checks
    doctor runs — REQ-020 (the command itself, one check per ranked sharp edge,
    and init naming it), REQ-023 (the toolchain-execution check, re-runnable
    outside init), REQ-025 (artifact-layout validation against the resolved
    artifact root) — and one, REQ-024, that this spec deliberately does NOT
    implement because the pack surface it reads does not exist (see Dependencies
    and Sharp Edges; escalated, not silently dropped). REQ-021 and REQ-022 were
    moved OUT of this seed at bundle v0.10.0 and belong to SPEC-068; doctor
    surfaces what they stamp and owns neither.
    What lands is one cobra command in `cmd/backstop` plus one declared check
    REGISTRY that every consumer reads — the `--check` selector, the human
    renderer, the `--json` renderer, the exit-code computation, and init's
    guidance — so a check cannot exist in one of those and not the others, and
    init cannot name a check id doctor does not register. EIGHT checks ship, in
    three groups. Four exist because REQ-003's refuse-to-run conditions are only
    reportable if some registered check produces them: `config-present`,
    `config-loads`, `git-repository`, and `packs-installed` turn the exact
    conditions that make `gate` exit 2 into ordinary check results carrying
    remediation. Three are the ranked sharp edges: `build-identity` SURFACES the
    running binary's version, build commit, build
    date, and schema cohort (the highest-ranked sharp edge — a stale binary
    reporting bare `dev` — made visible, without re-deciding BUNDLE-020's
    capability comparison); `toolchain-runs` EXECUTES every pack-declared
    `gate_type: test` / `gate_type: build` entrypoint once through the same
    allowlist and command-splitting path the gate's engine dispatch already uses,
    because DD-6's evidence is that package-manager configuration lies and only
    execution is ground truth; `artifact-layout` reports each artifact-shaped
    file that lies outside the resolved artifact root, naming the path it
    expected. The EIGHTH check — `engine-tools-present` — was NOT in the set this
    spec originally declared, and is not a quiet addition: it was authorized by
    DIR-002's founder-ruled scope expansion of 2026-08-16, which brought doctor's
    findings-engine tool-detection diagnostic coverage into that directive's
    charter as an ongoing concern, and it was delivered under ISSUE-134 /
    PLAN-ISSUE-134. It PROBES that every pack-declared, RULE-BOUND findings-engine
    tool resolves on PATH — closing the gap where doctor reported all-green on a
    project `backstop gate` refuses at exit 2 naming the same absent tool — and it
    is a PRESENCE PROBE ONLY: doctor invokes no findings-engine command.
    The load-bearing posture across all eight is that doctor is the
    command you run when everything else refuses: it reports an absent,
    unparseable, or gate-fatal `backstop.yml` as a CHECK RESULT and has no exit-2
    path at all, because a diagnostic that will not start on a broken setup
    cannot diagnose a broken setup.
  subject: cmd/backstop

verification:
  level: integration
  test_command: go test ./cmd/backstop/... -race -coverprofile=cover.out
  coverage_threshold: 80

requirements:
  - id: REQ-001
    text: >
      `backstop doctor` must be registered on the root command and must appear in
      the `backstop commands` agent-discovery output. A bare invocation must run
      EVERY registered check; the only way to report on a subset is an explicit
      `--check <id>`, and an unknown `--check` id must be a loud error naming the
      registered ids rather than an empty successful run. Doctor must render its
      results in human form and, under `--json`, as a payload declaring
      `schema_version: doctor/v1` that carries, for every check that ran, its id,
      title, status, message, and remediation. The two renderings must report the
      same check set and the same per-check statuses.
    supports:
      - onboarding-experience:REQ-020@1.0.0
  - id: REQ-002
    text: >
      Doctor's checks must live in ONE declared registry holding exactly the
      eight checks this spec declares, in this report order — `config-present`,
      `config-loads`, `git-repository`, `packs-installed`, `build-identity`,
      `toolchain-runs`, `engine-tools-present`, `artifact-layout` — and no other.
      `engine-tools-present` sits between `toolchain-runs` and `artifact-layout`
      because the two tool-execution/tool-presence checks read the same installed
      pack set and belong adjacent in the report. Each entry carries a
      stable, unique, lowercase-kebab id, a human title, and a function returning
      exactly one result whose status is one of `pass`, `warn`, `fail`,
      `skipped`. Registry order is report order and must be deterministic across
      runs — never Go map-iteration order. No check may be reachable except
      through that registry: the `--check` selector, the human renderer, the
      `--json` renderer, the exit-code computation, and init's guidance must all
      read the same set, so a check cannot be present in one consumer and missing
      from another. The mechanism that makes that provable rather than asserted
      is that exactly ONE non-test call site ENUMERATES `doctorRegistry()` —
      `runDoctor` — and every consumer reads the results it enumerates there. The
      registry has exactly two non-test readers, and their roles are distinct:
      `runDoctor`, the one site that ENUMERATES the full registry, from which every
      report's check SET comes; and `doctorGuidance`, which resolves a SINGLE check
      id against it, returns no set, and feeds no report. Any third non-test call
      site, and any second site that enumerates, is prohibited — a second
      enumeration is what would let two consumers disagree about the check set,
      whereas a keyed single-id lookup cannot. Every check id any code
      path can print must come from the one declared constant block the registry
      itself is built from, never from a second string literal.
    supports:
      - onboarding-experience:REQ-020@1.0.0
  - id: REQ-003
    text: >
      Doctor must REPORT, as check results carrying remediation, the conditions
      that make other commands refuse to run. Four REGISTERED checks own them,
      one condition each, so every condition below is produced by a named entry
      in the REQ-002 registry rather than by command-level error handling:
      `config-present` FAILS when no `backstop.yml` is discoverable from the
      working directory, naming the directory it searched from;
      `config-loads` FAILS naming the loader's own error when the discovered
      `backstop.yml` does not load or validate — the very error `runGate` turns
      into exit 2 — and is SKIPPED when no config was found at all, because
      `config-present` already reported that and no condition may be reported
      twice; `packs-installed` FAILS when a declared pack is missing from
      `.backstop/packs/` or its `pack.yml` cannot be parsed, WARNS when the
      config declares no packs at all (backstop enforces nothing — un-adopted
      capability, not a broken promise), and is SKIPPED when the config is absent
      or unloadable; `git-repository` WARNS when the project root is not inside a
      git work tree, naming what degrades — the diff scope falls back to a full
      sweep and `artifact new` cannot reserve an id — and never fails, because
      the gate itself already treats this as a loud fallback rather than a
      refusal. Doctor's exit code must be 0 when no check has status
      `fail` — warnings and skips included — and 1 when at least one check does.
      Doctor must have NO exit-2 config-error path: a `backstop.yml` that makes
      `backstop gate` exit 2 must make `backstop doctor` exit 1 with that
      condition reported as a failing check, because a diagnostic that refuses to
      start on a broken setup cannot diagnose a broken setup. No check may report
      a condition another check owns: a check whose input could not be gathered
      is SKIPPED with the owning check named, never failed a second time.
    supports:
      - onboarding-experience:REQ-020@1.0.0
  - id: REQ-004
    text: >
      Whenever an init step cannot complete AND a registered doctor check
      diagnoses that failure, init's printed guidance must name `backstop doctor`
      together with that check's id, read from the same registry doctor runs
      rather than written as an independent string literal. Every check id init
      can print must resolve to a registered check, and the ids must have exactly
      one source in the code so a registry rename cannot leave init pointing at a
      check that no longer exists. That guidance must be rendered in
      `cmd/backstop/init.go` — the cobra layer that already renders init's
      report from the `initialize.Result` SPEC-069's `Runner.Run` returns — and
      NOT inside `pkg/initialize`, which cannot reach the registry: the registry
      is unexported in package `main` and an import from `pkg/initialize` would
      invert the dependency. The rendering must map a FAILED `toolchain` step
      report to the toolchain check's id constant and produce guidance for no
      other step, and it must obtain the printable text through the one lookup
      that resolves an id against the registry, so an id no entry carries cannot
      be printed at all. The registered check set is the TEST of
      "diagnosable", and it is narrow: the toolchain-execution failure (REQ-006
      here) is diagnosable, and no registered check diagnoses a CI recipe ref
      that will not resolve or a brownfield CI preserve — so init must add NO
      doctor guidance to its CI steps, whose text bundle REQ-017 v1.2.0 confines
      to `recipe apply`'s own surfaced error and bundle REQ-035 confines to its
      own next action. That prohibition is SPEC-069's to enforce (its
      `TestInit_ImplementsNoCIDetectionOrBespokeGuidance`); this requirement's
      obligation is only that doctor never becomes the reason init violates it.
    supports:
      - onboarding-experience:REQ-020@1.0.0
  - id: REQ-005
    text: >
      The `build-identity` check must SURFACE the running binary's identity — its
      version string, the build commit and build date, and its schema cohort —
      as a reported result, and must WARN when that build identity is absent,
      which is exactly the stale-binary case behind the highest-ranked sharp edge
      (a weeks-old binary reporting bare `dev`, whose skew was misdiagnosed as a
      pack producer error). An absent build identity is a WARN, never a FAIL: the
      binary still runs. This check must NOT compare the binary against any pack.
      Capability-set comparison is BUNDLE-020's mechanism, stated as a diagnostic
      outcome by bundle REQ-022 and specced in SPEC-068; doctor surfaces what
      those stamp and re-decides nothing.
    supports:
      - onboarding-experience:REQ-020@1.0.0
  - id: REQ-006
    text: >
      The `toolchain-runs` check must verify the toolchain by EXECUTING it — once
      per pack-declared entrypoint — and must never infer toolchain health from
      package-manager configuration or from any other file on disk. Its subject
      is exactly the engines installed packs declare with `gate_type: test` or
      `gate_type: build`, the pack-declared test/compile entrypoints. Engines
      declared with `gate_type` `lint`, `findings`, `coverage`,
      `substantiveness`, or `contracts` must NOT be executed by this check. Each
      executed entrypoint produces one reported outcome: (a) executed and exited
      zero — pass, naming the pack and the command; (b) the declared executable
      cannot be executed at all — fail, classified as SETUP THE CONSUMER STILL
      OWES, naming the pack whose entrypoint could not run and pointing at that
      pack's own documented install steps, installing nothing itself; (c)
      executed and exited nonzero — fail, reporting the exit code and captured
      output VERBATIM, attributed to the pack and the command, with NO inference
      about the cause; (d) no installed pack declares a `test` or `build` engine
      — warn, stating that no installed pack declares a toolchain entrypoint,
      never a silent pass. Every declared entrypoint is reported separately when
      several packs declare one. When the installed pack set could not be
      gathered at all, this check is SKIPPED naming `packs-installed` as the
      check that owns that condition: it must never re-report a pack-loading
      failure as a toolchain failure, and must never read an ungathered pack set
      as outcome (d). Execution must go through the SAME trusted-tool
      allowlist check and command-splitting path the gate's engine dispatch uses:
      doctor must not introduce a second, weaker way to run pack-declared
      commands, and must never run one through a shell.
    supports:
      - onboarding-experience:REQ-023@1.0.0
  - id: REQ-007
    text: >
      The `artifact-layout` check must resolve the artifact root through the ONE
      config-resolved resolver bundle REQ-029 introduces (`artifact.ResolveRoot`,
      SPEC-068) and must hold no artifact directory name, extension, or root of
      its own — three independent hardcodings of the root layout are the defect
      REQ-029 removes, and this check must not become a fourth. A DEVIATION is an
      artifact-shaped file that is not located DIRECTLY in the directory the
      resolution expects for its own kind (`Root.Dir(kind)`) — directly, because
      discovery reads that directory with `os.ReadDir` and skips subdirectories,
      so an artifact nested one level below its expected directory is as ungated
      as one in the wrong tree. Every deviation must be reported with BOTH its
      actual path and that expected path. The deviation set must come from
      SPEC-068's shared `FindUngatedArtifacts`, whose `ExpectedDir` field IS the
      expected path this check prints, and doctor must not walk the corpus
      itself. The exclusion set that helper applies is no longer a list the
      helper CARRIES: it is the TOOL-AGNOSTIC BASE core holds (`.git`,
      `testdata`, `prototype`, plus the walk's own `.backstop/packs` rule)
      UNIONED with the ecosystem-specific dependency directory names installed
      packs declare via `classification.dependency_dirs`, and that union arrives
      as an INJECTED PARAMETER (ISSUE-122 — core bakes no ecosystem noun of its
      own, so neither `vendor` nor `node_modules` is core's to hold). Doctor must
      obtain that set the same way every other call site does — from the packs it
      has already loaded (`ctx.Packs`) — which is what keeps doctor and the gate
      from disagreeing about which trees are corpus. When the pack load failed
      (`ctx.PacksErr != nil`) doctor must inject the ZERO VALUE rather than
      defaulting any name back in, and the NAMED CONSEQUENCE is that the set is
      then the generic base ONLY, so artifact-shaped files inside dependency
      trees ARE reported as deviations that doctor does not report today. That is
      the HONEST failure and not a regression to fix: the only alternative is core
      knowing the ecosystem nouns, which is the exact bake ISSUE-122 removes, and
      the packs check FAILING on the same run is the diagnosis that explains the
      extra deviations. A deviation is loud, never blocking. Without the exclusion
      set at all this repository alone contributes 99 artifact-shaped files that
      are deliberately not corpus — 67 when this requirement was first written;
      the figure is this repo's own testdata and grows with it, and every one of
      the 99 sits under a `testdata` ancestor, so the tool-agnostic base alone
      still excludes all of them here even on the degraded path. Defining
      deviation per-KIND rather than as "outside the resolved root" is required,
      not stylistic: when no root is configured the resolved root IS the project
      root, so a repo holding artifacts under `.backstop/` — the live
      backstop-runtime shape this check exists for — has no file outside the root
      at all and a root-containment test reports nothing. A file that is not
      artifact-shaped is never reported; an expected type directory that is
      absent is not a deviation; a resolved root that exists but is empty is a
      pass; and a repository whose artifacts already sit in their expected
      directories passes with NO warning whether or not a root is configured,
      which is how backstop-core's unconfigured root layout stays clean without
      the check naming that repository. Remediation on a deviation must state
      both remedies — move the file to the expected path, or declare the root
      that makes its current location expected — and must not assert one as
      correct, because the choice between them is the consumer's layout decision.
      The resolution's typed failures must be reported as distinct results: a
      configured root that is missing from disk is a FAIL naming the declared
      value, and an invalid declaration is a FAIL naming the reason.
    supports:
      - onboarding-experience:REQ-025@1.0.0

claims:
  # REQ-001 — the command surface and its two renderings.
  - id: CLM-001
    requirement: REQ-001
    text: doctor is registered on the root command and appears in the `backstop commands` discovery tree
    tests:
      - TestDoctor_RegisteredAndDiscoverableInCommandTree
  - id: CLM-002
    requirement: REQ-001
    text: A bare `backstop doctor` runs and reports every check the registry declares, not a subset
    tests:
      - TestDoctor_BareRunReportsEveryRegisteredCheck
  - id: CLM-003
    requirement: REQ-001
    text: The `--json` payload declares schema_version doctor/v1 and carries id, title, status, message, and remediation for every check that ran
    tests:
      - TestDoctor_JSONPayloadCarriesSchemaVersionAndPerCheckFields
  - id: CLM-004
    requirement: REQ-001
    text: The human rendering and the `--json` rendering report the same check set with the same per-check statuses
    tests:
      - TestDoctor_HumanAndJSONReportSameCheckSetAndStatuses
  - id: CLM-061
    requirement: REQ-001
    text: "`backstop --json doctor` — the flag in ROOT position — renders the JSON payload, proving doctor reads the root persistent flag rather than shadowing it with a local one that only `backstop doctor --json` could reach"
    tests:
      - TestDoctor_RootPositionJSONFlagRendersJSONPayload
  - id: CLM-005
    requirement: REQ-001
    text: An explicit `--check <id>` runs only the named check and reports only its result
    tests:
      - TestDoctor_CheckSelectorRunsOnlyTheNamedCheck
  - id: CLM-006
    requirement: REQ-001
    text: An unknown `--check` id fails loudly naming the registered ids rather than reporting an empty successful run
    tests:
      - TestDoctor_UnknownCheckSelectorFailsNamingRegisteredIDs

  # REQ-002 — the registry is the single source every consumer reads.
  - id: CLM-007
    requirement: REQ-002
    text: Report order is the declared registry order and is byte-identical across repeated runs, so it cannot ride map iteration
    tests:
      - TestDoctor_ReportOrderIsDeterministicAcrossRuns
  - id: CLM-008
    requirement: REQ-002
    text: Every registry id is unique and lowercase-kebab
    tests:
      - TestDoctor_RegistryIDsAreUniqueAndKebabCase
  - id: CLM-009
    requirement: REQ-002
    text: Every registered check returns exactly one result whose status is one of pass, warn, fail, skipped
    tests:
      - TestDoctor_EveryCheckReturnsOneResultWithDeclaredStatus
  - id: CLM-010
    requirement: REQ-002
    text: The `--check` selector, both renderers, and the exit-code computation enumerate the SAME registry, so a check added to one is present in all
    tests:
      - TestDoctor_AllConsumersEnumerateTheSameRegistry
  - id: CLM-058
    requirement: REQ-002
    kind: absence
    text: "`doctorRegistry()` has no non-test call site other than `runDoctor` (the ONE site that ENUMERATES the full registry) and `doctorGuidance` (a keyed lookup of a single id, returning no set and feeding no report) — the mechanism behind CLM-010: a second enumeration is what would let two consumers disagree, and it cannot exist without this failing"
    tests:
      - TestDoctor_RegistryHasNoCallSiteOtherThanRunDoctorAndGuidance
  - id: CLM-059
    requirement: REQ-002
    kind: absence
    text: Every check id in non-test code is the declared constant rather than an inline string literal, so a rename cannot leave one consumer pointing at a retired id
    tests:
      - TestDoctor_CheckIDsAppearOnlyAsDeclaredConstants
  - id: CLM-051
    requirement: REQ-002
    kind: absence
    text: The registry holds exactly the declared set of check ids and registers NO stack-policy check, the tripwire on bundle REQ-024's carve-out — a check reading a pack stack-policy surface cannot appear without this failing
    tests:
      - TestDoctor_RegistersNoStackPolicyCheckAndReadsNoStackPolicySurface

  # REQ-003 — exit-code matrix, then the refuse-to-start conditions doctor must
  # instead diagnose.
  - id: CLM-011
    requirement: REQ-003
    text: A run whose checks all pass exits 0
    tests:
      - TestDoctor_ExitZeroWhenNoCheckFails
  - id: CLM-012
    requirement: REQ-003
    text: A run carrying warnings but no failure exits 0 — a warning is loud, not blocking
    tests:
      - TestDoctor_ExitZeroWhenWarningsPresent
  - id: CLM-013
    requirement: REQ-003
    text: A run carrying skipped checks but no failure exits 0
    tests:
      - TestDoctor_ExitZeroWhenChecksSkipped
  - id: CLM-014
    requirement: REQ-003
    text: A run in which at least one check fails exits 1
    tests:
      - TestDoctor_ExitOneWhenAnyCheckFails
  - id: CLM-015
    requirement: REQ-003
    text: An absent backstop.yml makes the registered `config-present` check FAIL with remediation, naming the directory it searched from, rather than making doctor refuse to run
    tests:
      - TestDoctorConfigPresent_AbsentConfigFailsAsCheckResult
  - id: CLM-016
    requirement: REQ-003
    text: The very backstop.yml that makes `backstop gate` exit 2 makes `backstop doctor` exit 1 with `config-loads` failing and carrying the loader's own error
    tests:
      - TestDoctor_UnparseableConfigExitsOneWhereGateExitsTwo
  - id: CLM-052
    requirement: REQ-003
    text: When no backstop.yml is found at all, `config-loads` is SKIPPED rather than failing — the absent-config condition is reported once, by the check that owns it
    tests:
      - TestDoctorConfigLoads_SkippedWhenNoConfigFound
  - id: CLM-053
    requirement: REQ-003
    text: A discoverable backstop.yml that loads and validates makes `config-present` and `config-loads` both pass
    tests:
      - TestDoctorConfig_PresentAndLoadableConfigPasses
  - id: CLM-017
    requirement: REQ-003
    text: A config declaring a pack that is missing from `.backstop/packs/` makes `packs-installed` FAIL with remediation, rather than making doctor refuse to run
    tests:
      - TestDoctorPacks_DeclaredPackMissingFromDiskFails
  - id: CLM-054
    requirement: REQ-003
    text: A config declaring no packs at all makes `packs-installed` WARN that backstop enforces nothing — un-adopted capability is loud, never a silent pass and never a failure
    tests:
      - TestDoctorPacks_NoDeclaredPacksWarnsRatherThanFailingOrPassing
  - id: CLM-055
    requirement: REQ-003
    text: An absent or unloadable backstop.yml makes `packs-installed` SKIPPED, so a config problem is never re-reported as a pack problem
    tests:
      - TestDoctorPacks_SkippedWhenConfigAbsentOrUnloadable
  - id: CLM-064
    requirement: REQ-003
    text: A config whose declared packs are all present and parseable makes `packs-installed` pass, naming the count
    tests:
      - TestDoctorPacks_AllDeclaredPacksPresentPassesNamingTheCount
  - id: CLM-018
    requirement: REQ-003
    text: A directory that is not a git repository makes `git-repository` WARN, naming the diff-scope fallback and the id-reservation loss, rather than failing, refusing, or crashing
    tests:
      - TestDoctorGit_NonRepositoryWarnsNamingWhatDegrades
  - id: CLM-056
    requirement: REQ-003
    text: A project root inside a git work tree makes `git-repository` pass
    tests:
      - TestDoctorGit_RepositoryPasses
  - id: CLM-057
    requirement: REQ-003
    text: No doctor code path returns ExitConfigError and no check gathers its own input by a route that can abort before the registry runs — the whole registry reports on a project with no backstop.yml, no packs, and no git repository
    tests:
      - TestDoctor_NoExitConfigErrorPathAndFullRegistryRunsOnAnEmptyDirectory

  # REQ-004 — init names the check, from the one id source.
  - id: CLM-019
    requirement: REQ-004
    text: An init run whose `toolchain` step report is a failure prints, from `cmd/backstop/init.go`, `backstop doctor` together with the registered id of the toolchain check that re-runs it standalone
    tests:
      - TestInit_ToolchainFailureNamesTheToolchainRunsCheckID
  - id: CLM-020
    requirement: REQ-004
    text: Every doctor check id init can print resolves to a check the registry registers, because the printable text is obtained through the registry lookup and an unregistered id yields no text at all
    tests:
      - TestInit_DoctorCheckIDsResolveToRegisteredChecks
      - TestDoctorGuidance_UnregisteredIDYieldsNoPrintableText
  - id: CLM-060
    requirement: REQ-004
    text: An init run whose steps all succeed, and one whose only failure is a step no registered check diagnoses, print no doctor guidance at all — so doctor never becomes the reason init's CI steps carry guidance SPEC-069 forbids
    tests:
      - TestInit_NoDoctorGuidanceForStepsNoRegisteredCheckDiagnoses

  # REQ-005 — build identity is SURFACED, never compared.
  - id: CLM-021
    requirement: REQ-005
    text: On a stamped binary the build-identity check reports the version string, build commit, build date, and schema cohort
    tests:
      - TestDoctorBuildIdentity_ReportsStampedVersionCommitDateAndCohort
  - id: CLM-022
    requirement: REQ-005
    text: A binary carrying no build identity warns rather than fails — the stale-binary case is surfaced, not blocked
    tests:
      - TestDoctorBuildIdentity_WarnsWhenBuildIdentityAbsent
  - id: CLM-023
    requirement: REQ-005
    text: The build-identity check performs no pack capability comparison and reads no pack manifest, leaving that mechanism to BUNDLE-020 via SPEC-068
    tests:
      - TestDoctorBuildIdentity_PerformsNoPackCapabilityComparison

  # REQ-006 — the gate_type matrix: all seven declared types, run or not run.
  - id: CLM-024
    requirement: REQ-006
    text: "An engine declared `gate_type: test` IS executed by the toolchain-runs check"
    tests:
      - TestDoctorToolchain_ExecutesTestGateTypeEntrypoint
  - id: CLM-025
    requirement: REQ-006
    text: "An engine declared `gate_type: build` IS executed by the toolchain-runs check"
    tests:
      - TestDoctorToolchain_ExecutesBuildGateTypeEntrypoint
  - id: CLM-026
    requirement: REQ-006
    text: "An engine declared `gate_type: lint` is NOT executed by the toolchain-runs check"
    tests:
      - TestDoctorToolchain_DoesNotExecuteLintEngine
  - id: CLM-027
    requirement: REQ-006
    text: "An engine declared `gate_type: findings` is NOT executed by the toolchain-runs check"
    tests:
      - TestDoctorToolchain_DoesNotExecuteFindingsEngine
  - id: CLM-028
    requirement: REQ-006
    text: "An engine declared `gate_type: coverage` is NOT executed by the toolchain-runs check"
    tests:
      - TestDoctorToolchain_DoesNotExecuteCoverageEngine
  - id: CLM-029
    requirement: REQ-006
    text: "An engine declared `gate_type: substantiveness` is NOT executed by the toolchain-runs check"
    tests:
      - TestDoctorToolchain_DoesNotExecuteSubstantivenessEngine
  - id: CLM-030
    requirement: REQ-006
    text: "An engine declared `gate_type: contracts` is NOT executed by the toolchain-runs check"
    tests:
      - TestDoctorToolchain_DoesNotExecuteContractsEngine

  # REQ-006 — the four outcomes, and the execution path they ride.
  - id: CLM-031
    requirement: REQ-006
    text: An entrypoint that executes and exits zero is reported as a pass naming the pack and the command it ran
    tests:
      - TestDoctorToolchain_PassNamesPackAndCommandOnSuccess
  - id: CLM-032
    requirement: REQ-006
    text: An entrypoint whose executable cannot be executed at all fails as SETUP OWED, naming the pack and pointing at that pack's own documented install steps, and doctor installs nothing
    tests:
      - TestDoctorToolchain_MissingExecutableReportedAsOwedSetup
  - id: CLM-033
    requirement: REQ-006
    text: An entrypoint that runs and exits nonzero fails reporting the exit code and captured output verbatim, with no inferred cause in the message
    tests:
      - TestDoctorToolchain_NonZeroExitReportsOutputVerbatimWithoutInferringCause
  - id: CLM-034
    requirement: REQ-006
    text: A project whose installed packs declare no test or build engine warns that no toolchain entrypoint is declared rather than passing silently
    tests:
      - TestDoctorToolchain_WarnsWhenNoPackDeclaresToolchainEntrypoint
  - id: CLM-035
    requirement: REQ-006
    text: When several packs declare an entrypoint, each is executed and reported separately
    tests:
      - TestDoctorToolchain_ReportsEachDeclaredEntrypointSeparately
  - id: CLM-036
    requirement: REQ-006
    text: A repo whose package-manager configuration is in the failing state still passes when the declared entrypoint executes successfully — health comes from execution, not configuration
    tests:
      - TestDoctorToolchain_PassesWhenEntrypointSucceedsDespiteFailingPackageManagerConfig
  - id: CLM-037
    requirement: REQ-006
    text: A declared command whose tool is not on the trusted-tool allowlist is refused rather than executed, on the same check the gate dispatch applies
    tests:
      - TestDoctorToolchain_RefusesCommandWhoseToolIsNotAllowlisted
  - id: CLM-038
    requirement: REQ-006
    text: Shell metacharacters in a declared command are passed as literal arguments and never interpreted, because doctor runs no shell
    tests:
      - TestDoctorToolchain_ShellMetacharactersArePassedAsLiteralArguments
  - id: CLM-063
    requirement: REQ-006
    text: A project whose pack set could not be gathered SKIPS the toolchain check naming `packs-installed`, rather than warning "no toolchain entrypoint declared" — outcome (d) requires a gathered pack set that declares none
    tests:
      - TestDoctorToolchain_SkippedWhenPackSetCouldNotBeGathered

  # REQ-007 — layout deviations against the RESOLVED root.
  - id: CLM-039
    requirement: REQ-007
    text: The check reports against the resolved root — declaring a different artifact root changes which files are deviations — proving it consumes the shared resolution rather than a root of its own
    tests:
      - TestDoctorLayout_ReportsAgainstResolvedArtifactRoot
  - id: CLM-040
    requirement: REQ-007
    text: A deviating artifact is reported with BOTH its actual path and the expected path for its kind under the resolved root
    tests:
      - TestDoctorLayout_DeviationReportsActualAndExpectedPath
  - id: CLM-041
    requirement: REQ-007
    text: Every artifact suffix in the seven-kind vocabulary is classified — .spec.md, .plan.yml, .adr.md, .bundle.md, .issue.md, .directive.md, .capability.yml — with none silently unrecognized
    tests:
      - TestDoctorLayout_ClassifiesEveryArtifactSuffix
  - id: CLM-062
    requirement: REQ-007
    text: A bare `capability.yml` is NOT reported, because doctor adds no filename pattern of its own on top of the shared classifier — widening it here would report a deviation on this repository's own CAP-001 file, which neither this spec nor SPEC-068 was asked to touch
    tests:
      - TestDoctorLayout_BareCapabilityYmlIsNotReportedAndDoctorAddsNoPattern
  - id: CLM-042
    requirement: REQ-007
    text: A file that is not artifact-shaped is never reported as a layout deviation
    tests:
      - TestDoctorLayout_NonArtifactFilesAreNotReported
  - id: CLM-043
    requirement: REQ-007
    text: Artifacts sitting in the directory expected for their kind are not deviations
    tests:
      - TestDoctorLayout_ArtifactsInTheirExpectedDirectoryAreNotDeviations
  - id: CLM-044
    requirement: REQ-007
    text: An expected type directory that is absent is not a deviation
    tests:
      - TestDoctorLayout_AbsentTypeDirectoryIsNotADeviation
  - id: CLM-045
    requirement: REQ-007
    text: A resolved artifact root that exists but is empty passes, preserving the validated greenfield outcome
    tests:
      - TestDoctorLayout_EmptyArtifactRootPasses
  - id: CLM-046
    requirement: REQ-007
    text: A repo with NO configured root holding artifacts under `.backstop/` reports them as deviations naming the expected default path — the backstop-runtime shape a root-containment test cannot see
    tests:
      - TestDoctorLayout_UnconfiguredRootReportsDotBackstopArtifactsAsDeviations
  - id: CLM-047
    requirement: REQ-007
    text: A repo with NO configured root whose artifacts sit in their expected directories passes with no warning, which is backstop-core's framework exception arising from the rule rather than from a special case
    tests:
      - TestDoctorLayout_UnconfiguredRootWithExpectedLayoutPassesWithoutWarning
  - id: CLM-048
    requirement: REQ-007
    text: Remediation on a deviation offers BOTH remedies — move the file, or declare the root that makes its location expected — asserting neither as the correct one
    tests:
      - TestDoctorLayout_RemediationOffersBothMoveAndDeclareRoot
  - id: CLM-049
    requirement: REQ-007
    text: A configured artifact root that is missing from disk fails, naming the declared value
    tests:
      - TestDoctorLayout_ConfiguredRootMissingFromDiskFails
  - id: CLM-050
    requirement: REQ-007
    text: An invalid artifact-root declaration fails naming the reason, distinctly from the missing-root failure
    tests:
      - TestDoctorLayout_InvalidRootDeclarationFailsNamingTheReason

contracts:
  - file: cmd/backstop/doctor.go
    provides:
      - name: newDoctorCommand
        kind: function
        signature: "func newDoctorCommand(jsonFlag *bool) *cobra.Command"
        notes: "The cobra surface, wired into NewRootCommand's AddCommand list alongside artifactCmd/packCmd/gateCmd/baselineCmd/versionCmd/commandsCmd/waiverCmd/recipeCmd (root.go:154). It takes the ROOT PERSISTENT --json flag by pointer, exactly as newGateCommand and all nine newPack*Command constructors do (root.go:25,42,91): --json is declared once on the root command, so declaring a local --json here would SHADOW it and `backstop --json doctor` would never reach the JSON renderer (CLM-061). It owns exactly one flag of its own, --check, and nothing else; every decision it renders comes from runDoctor."
      - name: doctorCheck
        kind: type
        signature: "type doctorCheck struct { ID string; Title string; Run func(ctx doctorContext) doctorResult }"
        notes: "One registry entry. ID is the stable lowercase-kebab identifier init prints (REQ-004) and --check selects on (REQ-001). The eight ids themselves are carried in source as the grouped untyped consts doctorCheckConfigPresent/doctorCheckConfigLoads/doctorCheckGitRepository/doctorCheckPacksInstalled/doctorCheckBuildIdentity/doctorCheckToolchainRuns/doctorCheckEngineTools/doctorCheckArtifactLayout, and those consts carry NO `provides` entry of their own: a member of a grouped `const (…)` block is structurally inexpressible to the contracts pack's signature compiler, which binds only a standalone `const NAME = value` — the capability gap is ISSUE-078, and the disposition SPEC-054 and SPEC-035 v1.1.2 already took applies unchanged, because declaring an unverifiable entry buys a red, not a guarantee. The one-id-source invariant those consts exist for is NOT left unenforced: CLM-059 is a `kind: absence` source scan asserting that no check id is written as a literal anywhere outside them, which pins the property the dropped entries would only have gestured at."
      - name: doctorResult
        kind: type
        signature: "type doctorResult struct { ID string `json:\"id\"`; Title string `json:\"title\"`; Status string `json:\"status\"`; Message string `json:\"message\"`; Remediation string `json:\"remediation\"` }"
        notes: "Status is one of pass|warn|fail|skipped (REQ-002). The json tags ARE the doctor/v1 payload's per-check shape (REQ-001) — one struct feeds both renderers so they cannot disagree (CLM-004). Remediation carries NO omitempty deliberately: REQ-001 requires the key present for every check that ran, so a passing check emits an empty string rather than dropping the key and making the payload's shape depend on its status (CLM-003)."
      - name: doctorRegistry
        kind: function
        signature: "func doctorRegistry() []doctorCheck"
        notes: "THE single source. A SLICE, not a map, because report order is a requirement (CLM-007). It holds exactly the eight declared checks in report order: config-present, config-loads, git-repository, packs-installed, build-identity, toolchain-runs, engine-tools-present, artifact-layout (REQ-002, CLM-051) — the eighth added under DIR-002's founder-ruled scope expansion of 2026-08-16 and delivered by PLAN-ISSUE-134, never by an implementer's own judgement. It has exactly TWO non-test readers with distinct roles, and no third (CLM-058): runDoctor, the ONE site that ENUMERATES it — every report's check SET comes from that one enumeration — and doctorGuidance, which resolves a SINGLE id against it, returns no set and feeds no report. That is the mechanism behind CLM-010: every consumer reads the results runDoctor enumerates, so a divergence would require a second ENUMERATION, and a keyed single-id lookup is not one."
      - name: doctorGuidance
        kind: function
        signature: "func doctorGuidance(checkID string) (string, bool)"
        notes: "REQ-004's one lookup, and the registry's SECOND permitted non-test reader (REQ-002, CLM-058) — a keyed lookup of a single id, never an enumeration: it returns no check set and feeds no report, so it cannot make two consumers disagree about the set. Resolves checkID against doctorRegistry() and returns the printable `backstop doctor --check <id>` guidance, or (\"\", false) when no entry carries it — so an unregistered id is unprintable rather than printed wrong (CLM-020). Any code outside doctor.go that names a doctor check goes through this."
      - name: runDoctor
        kind: function
        signature: "func runDoctor(cmd *cobra.Command, jsonFlag *bool, args []string) error"
        notes: "Gathers the doctorContext, enumerates doctorRegistry() once, and returns *ExitCodeError{Code: ExitViolations} when any result is fail and nil otherwise. It must NOT return ExitConfigError on any path: config load failure is a check result (CLM-016, CLM-057), which is the inverse of runGate's first act (gate.go:67-73)."
    consumes:
      - source: github.com/spf13/cobra
        name: Command
        kind: type
      - source: cmd/backstop
        name: ExitCodeError
        kind: type
      - source: pkg/config
        name: DiscoverConfigPath
        kind: function
      - source: pkg/config
        name: LoadConfig
        kind: function
      - source: cmd/backstop
        name: loadInstalledPacks
        kind: function
      - source: pkg/check
        name: CommandRunner
        kind: interface
  - file: cmd/backstop/doctor_checks.go
    provides:
      - name: checkConfigPresent
        kind: function
        signature: "func checkConfigPresent(ctx doctorContext) doctorResult"
        notes: "REQ-003. Reads ctx.ConfigPath/ctx.ConfigPathErr — the result of the ONE config.DiscoverConfigPath call runDoctor already made — and never calls a loader itself. FAILS naming the directory searched from when discovery found nothing; passes naming the resolved path otherwise (CLM-015, CLM-053)."
      - name: checkConfigLoads
        kind: function
        signature: "func checkConfigLoads(ctx doctorContext) doctorResult"
        notes: "REQ-003. Reads ctx.ConfigErr as DATA. FAILS carrying the loader's own error text — the very error runGate converts to exit 2 (CLM-016) — and is SKIPPED naming config-present when no config was discovered at all, so the absent-config condition is reported exactly once (CLM-052)."
      - name: checkGitRepository
        kind: function
        signature: "func checkGitRepository(ctx doctorContext) doctorResult"
        notes: "REQ-003. Detects the work tree through the existing exported `(&check.DefaultGitExecutor{Dir: projectRoot}).IsGitRepo()` rather than a fourth `git rev-parse` shell-out. WARNS, never fails (CLM-018): the gate itself falls back to a full sweep with a warning when git is absent (pkg/gate/scope.go:220-222), so a non-repo is degraded capability, not a broken promise — the message names that fallback and the loss of tag-based id reservation."
      - name: checkPacksInstalled
        kind: function
        signature: "func checkPacksInstalled(ctx doctorContext) doctorResult"
        notes: "REQ-003. Reads ctx.Packs/ctx.PacksErr as DATA. Note the shape of loadInstalledPacks (pack_gate.go:147-176): a config declaring no packs returns an EMPTY slice with a nil error, while a declared-but-missing pack or an unparseable pack.yml returns an error — so empty-with-nil-error is the WARN case (CLM-054) and the error is the FAIL case (CLM-017). It is SKIPPED when the config is absent or unloadable, because loadInstalledPacks loads backstop.yml itself and would otherwise surface a config problem as a pack problem (CLM-055)."
      - name: checkBuildIdentity
        kind: function
        signature: "func checkBuildIdentity(ctx doctorContext) doctorResult"
        notes: "REQ-005. Reads the same resolved version/cohort values the version command renders (root.go:106-127) plus the build commit/date SPEC-068 stamps. It performs no pack read at all, which is what keeps CLM-023 falsifiable."
      - name: checkToolchainRuns
        kind: function
        signature: "func checkToolchainRuns(ctx doctorContext) doctorResult"
        notes: "REQ-006. It CONSUMES the shared packEntrypointProber (cmd/backstop/pack_entrypoint_prober.go) and owns no mechanical step of its own: selection by engine.GateTypeTest and engine.GateTypeBuild ONLY, and the three execution steps — the allowlist check, the command split, then the shared CommandRunner — happen one layer down inside the prober, in that order, the same three steps runFindingsEngine takes (pack_gate.go:573) minus the SARIF parse. Because init consumes that same type through its own thin adapter, there is exactly ONE execution route to audit rather than one per command. This function contributes the rollup: constructing the prober from ctx.Packs and ctx.Runner, calling Probe once, and mapping the returned probes onto the (a)/(b)/(c)/(d) outcomes and the single doctorResult. It is SKIPPED when ctx.PacksErr is non-nil, naming packs-installed as the owner of that condition, so an ungathered pack set is never read as outcome (d) (CLM-063)."
      - name: checkEngineToolsPresent
        kind: function
        signature: "func checkEngineToolsPresent(ctx doctorContext) doctorResult"
        notes: "REQ-002's eighth registered check, authorized by DIR-002's founder-ruled scope expansion of 2026-08-16 and delivered under ISSUE-134 / PLAN-ISSUE-134, which owns its behavioral claims and mandated test names; this spec owns its REGISTRY MEMBERSHIP (the enumeration in REQ-002, the order CLM-007 asserts, and the exactly-this-set tripwire CLM-051). Declared here rather than dropped because this `provides` block is kept 1:1 with the registry — one entry per check function — and an eighth check with no entry would silently break that correspondence. It CONSUMES the shared `collectRequiredEngineTools` (cmd/backstop/pack_gate_provision.go) and owns no registry walk and no trust gate of its own: the manifest-rule walk, the `resolveEngineRegistry` lookup, the `checkEngineToolAllowed` refusal and the argv[0] extraction all live in that one collection authority, whose OTHER consumer is `provisionEngines` — the gate's own disposition half — so doctor and the gate cannot disagree about WHICH tools are required. Dispositions: `skipped` on an ungathered pack set (ConfigPathErr || ConfigErr || PacksErr), naming packs-installed as that condition's owner, the identical predicate checkToolchainRuns carries for the identical reason; `fail` on a trust-gate refusal, because reporting a refused tool as merely missing would send the reader to install a binary backstop declines to run; `warn` when a gathered pack set binds no engine tool at all, mirroring checkToolchainRuns' outcome (d) rather than passing silently; `fail` enumerating EVERY absent tool where the gate stops at the FIRST, so an operator is not made to re-run once per missing tool; `pass` otherwise. Presence is probed through `resolveBinaryResolver()` ONLY — no findings-engine command is executed — and each absence's remediation is rendered by the gate's own `absentToolMessage`, so doctor's advice and the gate's refusal are the same words by construction rather than two texts that drift."
      - name: checkArtifactLayout
        kind: function
        signature: "func checkArtifactLayout(ctx doctorContext) doctorResult"
        notes: "REQ-007. Consumes SPEC-068's `artifact.ResolveRoot(projectRoot, declared)` for the root and `gate.FindUngatedArtifacts(projectRoot, root, nonCorpus)` for the deviation set, printing each `UngatedArtifact`'s `ExpectedDir` as the expected path — no `specs`/`bundles`/`issues`/`directives`/`plans` join, no suffix list, no walk, and no exclusion list of its own, or it becomes the fourth hardcoding REQ-029 exists to remove. The `nonCorpus` argument is an `artifact.NonCorpusDirs` DERIVED from `ctx.Packs` — the pack-declared `classification.dependency_dirs` names unioned onto core's tool-agnostic base — and INJECTED into the helper, which is the same derivation every other call site performs and is what keeps doctor and the gate from disagreeing about which trees are corpus; the ZERO VALUE is injected when `ctx.PacksErr` is non-nil, degrading to the tool-agnostic base only (ISSUE-122). That injection does NOT change this function's own signature: it derives the set from the `doctorContext` it already receives. `declared` comes from the `Config.ArtifactRoot` field SPEC-068 adds; a nil/failed config yields the empty declaration, which resolves to the project root and must still produce a report rather than an error (REQ-003). The typed failures `*artifact.RootMissingError` and `*artifact.RootInvalidError` are distinguished by type, never by string match."
      - name: doctorContext
        kind: type
        signature: "type doctorContext struct { ProjectRoot string; SearchDir string; ConfigPath string; ConfigPathErr error; Config *config.Config; ConfigErr error; Packs []*pack.Manifest; PacksErr error; Runner check.CommandRunner }"
        notes: "Gathered ONCE before the registry runs and passed to every check; no check gathers its own input. That is why this file's `consumes` names the DATA types the checks read — `config.Config`, `pack.Manifest`, and the `check.CommandRunner` this struct carries and `checkToolchainRuns` hands to the shared `packEntrypointProber` — and NOT the loaders that produce them: `pkg/config.LoadConfig` and `loadInstalledPacks` are consumed by `doctor.go`, the file that gathers (Implementation §2). A loader call appearing in this file is the defect REQ-003 exists to forbid — a second abort path before the registry. ConfigPathErr, ConfigErr and PacksErr are carried as DATA rather than raised, which is the mechanism behind REQ-003: every load failure reaches the checks as a condition to report, so no check can turn one into an exit-2 (CLM-057). SearchDir is the working directory discovery started from, which checkConfigPresent names in its failure; ConfigPath is empty exactly when ConfigPathErr is non-nil, which is the SKIP signal checkConfigLoads and checkPacksInstalled read."
    consumes:
      - source: pkg/config
        name: Config
        kind: type
      - source: pkg/pack
        name: Manifest
        kind: type
      - source: pkg/check
        name: DefaultGitExecutor
        kind: type
      - source: cmd/backstop
        name: packEntrypointProber
        kind: type
        notes: "Declared in cmd/backstop/pack_entrypoint_prober.go, built by SPEC-069/PLAN-SPEC-069 phase 14 as THE ONE execution route for pack-declared test/build entrypoints. `checkToolchainRuns` constructs it from ctx.Packs and ctx.Runner and calls Probe once, consuming the same shared type `cmd/backstop/init_toolchain.go` consumes as a thin adapter. The three execution steps — allowlist check, command splitting, execution through check.CommandRunner.Run — plus the binding selection, the deterministic sorted-key walk, and the started-versus-exited-nonzero split all live INSIDE the prober, so this file names neither checkEngineToolAllowed nor splitCommand: it does not call them. That is the point of the extraction — the number of independent execution routes to audit is exactly one, not one per command. What this file adds on top of the raw []entrypointProbe is doctor's own rollup: the (a)/(b)/(c)/(d) mapping REQ-006 states and the one-result-per-check aggregation, which is the mirror image of init's owed-setup-versus-verbatim classification."
      - source: cmd/backstop
        name: collectRequiredEngineTools
        kind: function
        notes: "Declared in cmd/backstop/pack_gate_provision.go as THE ONE authority on which findings-engine tools a pack set requires: it walks each manifest's RULES, resolves each rule's engine through resolveEngineRegistry(manifest).Lookup, applies the checkEngineToolAllowed trust gate, extracts argv[0] and returns the deduped set sorted by probed name. `checkEngineToolsPresent` calls it and adds only doctor's disposition; the gate's `provisionEngines` calls the same function and adds only the gate's. That is the same two-consumers-one-authority shape `packEntrypointProber` has, and it is why doctor cannot report a required-tool set the gate does not use."
      - source: cmd/backstop
        name: absentToolMessage
        kind: function
        notes: "The gate's own renderer for an absent required tool (pack_gate_provision.go). `checkEngineToolsPresent` reuses it verbatim as its per-tool remediation rather than writing a second text, so doctor's advice and the gate's refusal cannot drift apart."
      - source: pkg/pack/engine
        name: GateType
        kind: type
      - source: pkg/check
        name: CommandRunner
        kind: interface
      - source: pkg/artifact
        name: ResolveRoot
        kind: function
      - source: pkg/artifact
        name: Root
        kind: type
      - source: pkg/gate
        name: FindUngatedArtifacts
        kind: function
      - source: pkg/gate
        name: UngatedArtifact
        kind: type
  - file: cmd/backstop/root.go
    provides:
      - name: NewRootCommand
        kind: function
        signature: "func NewRootCommand() *cobra.Command"
        notes: "Signature UNCHANGED. The single edit is adding newDoctorCommand(&jsonFlag)'s result to the existing AddCommand call (root.go:154), passing the same persistent-flag pointer gateCmd and the nine pack commands already receive (root.go:25,42,91). That is what makes CLM-001's discovery assertion hold through buildCommandTree with no discovery-specific code, and CLM-061's root-position --json reach the renderer."
    consumes:
      - source: cmd/backstop
        name: newDoctorCommand
        kind: function
  - file: cmd/backstop/init.go
    provides:
      - name: doctorGuidanceForSteps
        kind: function
        signature: "func doctorGuidanceForSteps(steps []initialize.StepReport) []string"
        notes: "REQ-004. The ONLY place init text names doctor. It lives in cmd/backstop rather than pkg/initialize because doctorRegistry is unexported in package main and pkg/initialize importing it would invert the dependency — SPEC-069 already renders init's report here from the initialize.Result its Runner returns, so this is a rendering concern in the layer that already does the rendering. It maps a FAILED `toolchain` step report to doctorGuidance(doctorCheckToolchainRuns) and returns nothing for every other step (CLM-019, CLM-060), which is what keeps init's CI steps free of doctor guidance — SPEC-069's TestInit_ImplementsNoCIDetectionOrBespokeGuidance is the guard. It carries no check-id literal and no guidance-text literal: both come from doctor.go."
    consumes:
      - source: pkg/initialize
        name: StepReport
        kind: type
      - source: cmd/backstop
        name: doctorGuidance
        kind: function
---

# SPEC-070: Backstop Doctor

## Overview

BUNDLE-003 partitions into three seeds. The guards seed makes version skew unable
to CERTIFY; the init seed automates the validated happy path; this seed is what a
consumer runs when the setup is off anyway. It is the smallest of the three — four
bundle requirements — and its shape is set by one line of DD-8: each ranked sharp
edge from the two hand-onboarding write-ups is either an init guardrail or a
`doctor` check. Doctor's checks are therefore not invented; they are the deviations
those write-ups actually ranked.

Three things about this spec are worth stating before the requirements, because
each is a place a reader could reasonably expect something that is deliberately
not here.

First, **REQ-021 and REQ-022 are not in this spec.** They were moved to the guards
seed at bundle v0.10.0 — REQ-021 changes what `cmd/backstop/version.go` stamps and
REQ-022 changes the `pack add` / `gate` failure path, so neither is a check doctor
runs. Doctor SURFACES what they produce (REQ-005 here) and owns neither.

Second, **bundle REQ-024 is not implemented by this spec.** It requires doctor to
check the installed runtime/toolchain version against "the stack policy declared by
the installed packs", with the policy values as pack DATA. Verified at HEAD:
`pkg/pack/manifest.go` declares no stack-policy surface of any kind, and the real
installed `typescript-toolchain@1.2.1` declares no runtime version anywhere. This
is the same shape as REQ-005 v1.0.0's retired clause, which named a pack-manifest
field that did not exist and was corrected with the ruling that "manifest surface
design is BUNDLE-004's, not this bundle's." That precedent is followed rather than
re-litigated: this spec does not invent the surface, and the requirement is carried
in Dependencies with its escalation, not silently dropped.

Third, **doctor is a diagnostic, not a gate.** Nothing in the kill chain consumes
its output, its `--check` selector is not a filtering mechanism that could
manufacture a green, and its warnings do not block. What it owes is the opposite
property from the gate's: it must start and report on precisely the broken setups
that make every other command refuse.

## Requirements

Requirements, claims, and bundle pins are defined in frontmatter. In summary:

| Requirement | Bundle pin | Surface |
| --- | --- | --- |
| REQ-001 — the command, its two renderings, and the `--check` selector | `REQ-020@1.0.0` | `cmd/backstop/doctor.go` |
| REQ-002 — one declared registry every consumer reads | `REQ-020@1.0.0` | `cmd/backstop/doctor.go` |
| REQ-003 — four checks that diagnose what other commands refuse; no exit-2 path | `REQ-020@1.0.0` | `cmd/backstop/doctor.go`, `cmd/backstop/doctor_checks.go` |
| REQ-004 — init names the registered check id, from one id source | `REQ-020@1.0.0` | `cmd/backstop/init.go` + the registry |
| REQ-005 — `build-identity` surfaces, never compares | `REQ-020@1.0.0` | `cmd/backstop/doctor_checks.go` |
| REQ-006 — `toolchain-runs` executes the declared entrypoint | `REQ-023@1.0.0` | `cmd/backstop/doctor_checks.go` |
| REQ-007 — `artifact-layout` reports each artifact not in its expected directory under the resolved root | `REQ-025@1.0.0` | `cmd/backstop/doctor_checks.go` |

### One check per ranked sharp edge

REQ-020's coverage bar is "one check per ranked sharp edge from the hand-onboarding
write-ups". The full-SDLC write-up ranks nine; DD-8's corollary is that each is a
doctor check OR an init guardrail. The mapping this spec implements, and where the
non-doctor ones went:

| Ranked sharp edge | Disposition |
| --- | --- |
| 1 — CLI/pack version skew, silently misdiagnosed | `build-identity` check (REQ-005); the guards that stamp and compare are SPEC-068 |
| 2 — artifact-dir convention, consuming profile undiscovered | `artifact-layout` check (REQ-007) |
| 3 — package-manager guidance stale; toolchain health inferred, not executed | `toolchain-runs` check (REQ-006) |
| 4 — Node LTS decision unenforced | bundle REQ-024 — CARVED OUT (ruled 2026-08-13); no pack surface exists to read (Dependencies) |
| 5 — `.gitignore` divergence between onboarded repos | init guardrail, bundle REQ-005 (SPEC-069) |
| 6 — self-contradictory remoteless `baseline_comparison` message | ISSUE-056, consumed not built (bundle REQ-013) |
| 7 — greenfield defaults are clean (positive finding) | not a defect; it is the two-profile fork, bundle REQ-003 |
| 8 — observations, explicitly not blockers | no check |
| 9 — `artifact new` cannot gap-fill; orphan reservation tags | NOT taken: the bundle records that whether REQ-020 sweeps in a doctor-side orphan-tag check "is a founder call, not one this pass took" |

Row 9 is the one place this spec could have quietly widened REQ-020 and did not.
Row 4 is the one place it could have quietly invented a pack surface and did not.

### The eighth check, and the authority that admitted it

`engine-tools-present` is in the REQ-002 registry and is produced by NO row of the
table above and by no REQ-003 condition. That is deliberate and it is recorded
rather than inferred, because the sharp edge below — "growth belongs to the
bundle" — makes an unexplained eighth check indistinguishable from the quiet
addition CLM-051 exists to catch.

- **The gap.** `backstop doctor` reported an all-green exit 0 on a project whose
  pack-declared, RULE-BOUND findings-engine tool was absent from PATH, while
  `backstop gate` on the same project refused at exit 2 naming that tool. Doctor
  consumed only the shared `packEntrypointProber`, whose selection is BY STAGE
  (`gate_type: test` / `gate_type: build`), so a findings engine was never
  selected, never probed, and never mentioned. Filed as ISSUE-134.
- **The authority.** Not this spec's own judgement and not an implementer's:
  DIR-002's founder-ruled scope expansion of 2026-08-16
  (`directives/DIR-002-backstop-init.directive.md` — the ISSUE-134 follow-on and
  the "Founder-approved home and framing" paragraph) states that this directive's
  charter now includes doctor's tool-detection diagnostic reliability going
  forward as an ongoing concern, not merely BUNDLE-003 residue.
- **The ownership split.** This spec owns the check's REGISTRY MEMBERSHIP —
  REQ-002's enumeration, its report position, the order CLM-007 asserts and the
  exactly-this-set tripwire CLM-051 — and declares its function in the contracts
  block. Its BEHAVIORAL claims and mandated test names are owned by ISSUE-134 /
  `PLAN-ISSUE-134`, on the reactive `issue -> plan` track; this spec does not
  restate them, because two owners for one guarantee is how a guarantee stops
  having one.
- **What did NOT change.** Bundle REQ-024 remains CARVED OUT and ISSUE-121 still
  owns it; CLM-051's stack-policy source scan is intact and unweakened, and its
  mandated test name is unchanged. Only the SIZE of the declared set moved.

### The four checks no ranked sharp edge produced

The ranked list is a coverage FLOOR, not the registry's whole membership. REQ-020
also obliges doctor to report the conditions that make other commands refuse —
REQ-003 here — and a condition is only reportable if some REGISTERED check
produces it, because the registry is the only thing either renderer or the
exit-code computation reads. Command-level error handling that printed "no
backstop.yml" outside the registry would be exactly the refuse-to-start behavior
REQ-003 forbids, and would appear in neither the `--json` payload nor the
`--check` selector. So four checks exist for REQ-003's four conditions, one each:

| Check | Condition it owns | Statuses it can produce |
| --- | --- | --- |
| `config-present` | no `backstop.yml` discoverable | `pass`, `fail` |
| `config-loads` | the discovered `backstop.yml` does not load or validate | `pass`, `fail`, `skipped` |
| `git-repository` | the project root is not inside a git work tree | `pass`, `warn` |
| `packs-installed` | a declared pack is missing or unparseable; or none is declared | `pass`, `warn`, `fail`, `skipped` |

`git-repository` warns rather than fails because the gate already treats a
missing work tree as a loud fallback to a full sweep rather than a refusal
(`pkg/gate/scope.go:220-222`) — un-adopted capability is loud, not blocking. The
other three can fail, which is what makes CLM-016's exit-1-where-gate-exits-2
assertion reachable at all.

One condition, one owner. A check whose input could not be gathered is `skipped`
naming the check that owns the condition, never failed a second time — otherwise
a single absent `backstop.yml` would produce four failures and the report would
say four things are broken when one is.

## Implementation

Everything lands in `cmd/backstop` — the package that already holds
`loadInstalledPacks`, the shared `packEntrypointProber` (SPEC-069's extracted
execution route, `pack_entrypoint_prober.go`), and the command wiring — so no new
package boundary is introduced for eight checks. The eighth check additionally
consumes `collectRequiredEngineTools` from `pack_gate_provision.go`, which is in
that same package.

1. **The registry (REQ-002).** `doctorRegistry() []doctorCheck` in `doctor.go`
   returns a SLICE literal of the eight checks in report order: `config-present`,
   `config-loads`, `git-repository`, `packs-installed`, `build-identity`,
   `toolchain-runs`, `engine-tools-present`, `artifact-layout`. The four setup
   checks come first because
   they are what the rest depend on, so a reader sees why a later check skipped
   before reading the skip. A slice rather than a map
   is load-bearing: order is asserted (CLM-007), and map iteration would make the
   report non-deterministic. Each entry's `ID` is one of the eight declared
   constants (`doctorCheckConfigPresent` … `doctorCheckArtifactLayout`, including
   `doctorCheckEngineTools`) — no id
   is written as a literal anywhere, including in init's guidance (CLM-059).
   Nothing may enumerate checks any other way: `doctorRegistry()` has exactly two
   non-test call sites and no third (CLM-058) — `runDoctor`, the ONE that
   ENUMERATES it, from which every consumer reads the results it produced; and
   `doctorGuidance`, which resolves a single id against it (REQ-004), returning no
   set and feeding no report, so it is a keyed lookup rather than a second
   enumeration.

2. **Context gathering (REQ-003).** Before any check runs, `runDoctor` builds one
   `doctorContext`: the working directory discovery started from, the discovered
   `backstop.yml` path AND its discovery error, the project root (the directory
   of that path, falling back to the working directory exactly as `runGate` does
   at `gate.go:75-80`), the loaded config AND its error, the installed packs AND
   their error, and the shared `check.CommandRunner`. The errors are CARRIED, not
   raised. This inversion is the whole of REQ-003: `runGate` turns a config load
   failure into an immediate exit-2 (`gate.go:67-73`); `runDoctor` turns the same
   failure into a reported condition and continues. No check performs its own
   gathering, which is what makes CLM-057 provable — there is no second place an
   abort could originate.

3. **The eight checks, in registry order.**
   - `checkConfigPresent` (REQ-003) reports the discovered config path, or fails
     naming the directory the search started from. Its remediation names creating
     a `backstop.yml` or running `backstop init`.
   - `checkConfigLoads` (REQ-003) reports `ctx.ConfigErr` verbatim as a failure —
     this is the identical error text `runGate` wraps into its exit-2 — or skips
     naming `config-present` when nothing was discovered to load.
   - `checkGitRepository` (REQ-003) calls
     `(&check.DefaultGitExecutor{Dir: ctx.ProjectRoot}).IsGitRepo()`, the existing
     exported detector, and warns when it is false. Doctor adds no fourth
     `git rev-parse` shell-out of its own.
   - `checkPacksInstalled` (REQ-003) reads `ctx.Packs`/`ctx.PacksErr`. The three
     outcomes follow the shape `loadInstalledPacks` already has
     (`pack_gate.go:147-176`): an error means a declared pack is missing from
     `.backstop/packs/` or its `pack.yml` did not parse — fail, remediation
     `backstop pack install`; an empty slice with no error means the config
     declares no packs — warn, because backstop then enforces nothing; anything
     else passes naming the count. It skips when the config is absent or
     unloadable, since `loadInstalledPacks` reads `backstop.yml` itself and would
     otherwise re-report a config failure under a pack heading.
   - `checkBuildIdentity` (REQ-005) reads the resolved version string, the build
     commit and build date SPEC-068 stamps, and the schema cohort — the same
     values the `version` command renders (`root.go:106-127`) — and reports them.
     Absent build identity warns. It reads no pack manifest, executes nothing, and
     performs no comparison.
   - `checkToolchainRuns` (REQ-006) constructs the shared `packEntrypointProber`
     from the gathered manifests and `ctx.Runner` and calls `Probe` ONCE. The
     prober walks those manifests, selects every engine binding whose `GateType`
     is `engine.GateTypeTest` or `engine.GateTypeBuild`, and for each applies the
     trusted-tool allowlist check, splits the declared command, then executes it
     once from the project root through the shared runner — the same three steps
     in the same order as before the extraction, just one layer down, and now the
     SAME route `backstop init` takes through its own thin adapter, so the number
     of independent execution routes to audit is one rather than two. What this
     check adds is the rollup from the returned probes to a single
     `doctorResult`. The five other declared gate
     types are not selected, and the whole check skips when `ctx.PacksErr` is
     non-nil rather than reading an ungathered pack set as "no entrypoint
     declared". The outcome mapping is (a)/(b)/(c)/(d) exactly as
     REQ-006 states, and case (b) is distinguished from case (c) by the ONE signal
     that is mechanically honest: the executable could not be started at all
     versus it started and exited nonzero. No other classification is attempted.
   - `checkEngineToolsPresent` (REQ-002's eighth registered entry; behavior owned
     by ISSUE-134 / `PLAN-ISSUE-134` under DIR-002's 2026-08-16 scope expansion)
     calls the shared `collectRequiredEngineTools(ctx.Packs)` ONCE and adds only
     doctor's disposition. The collection authority — the manifest RULE walk, the
     `resolveEngineRegistry(manifest).Lookup(rule.Engine)` resolution, the
     `checkEngineToolAllowed` trust gate, the argv[0] extraction, the dedupe and
     the sort by probed name — lives in `pack_gate_provision.go` and is shared
     with `provisionEngines`, the gate's own disposition half, so the two cannot
     disagree about WHICH tools are required. This check is SKIPPED on
     `ctx.ConfigPathErr != nil || ctx.ConfigErr != nil || ctx.PacksErr != nil`,
     naming `packs-installed` as that condition's owner — the identical predicate
     `checkToolchainRuns` carries, for the identical reason: a `PacksErr`-only
     predicate would report "no installed pack binds an engine tool" on a project
     whose packs were never looked at. A trust-gate refusal FAILS distinctly from
     an absence, because a refused tool is one backstop declines to run even once
     installed. A gathered pack set binding no engine tool WARNS rather than
     passing silently, mirroring `checkToolchainRuns`' outcome (d). Otherwise each
     required tool is resolved with `resolveBinaryResolver()` — a PRESENCE PROBE;
     no findings-engine command is executed — and the single result enumerates
     EVERY probed tool with its pack and engine attribution, taking the worst
     status among them. That enumeration is the one deliberate divergence from the
     gate, which stops at the FIRST absence in sorted order: a diagnostic that did
     the same would make the operator re-run it once per missing tool. Each
     absence's remediation is rendered by the gate's own `absentToolMessage`, so
     doctor's advice and the gate's refusal are the same words by construction.
   - `checkArtifactLayout` (REQ-007) resolves the artifact root through
     `artifact.ResolveRoot(projectRoot, declared)`, hands that resolution to
     `gate.FindUngatedArtifacts`, and reports each returned `UngatedArtifact` as a
     deviation, printing its `ExpectedDir` as the expected path. It performs no
     walk of its own, holds no suffix list, joins no type-directory name, and
     holds no exclusion list of its own — the first three come from the shared
     helper, and the fourth is DERIVED from `ctx.Packs` (the pack-declared
     `classification.dependency_dirs` names unioned onto core's tool-agnostic
     base) and INJECTED into that same helper as a parameter (ISSUE-122). That is
     still — and more strongly — what makes "what doctor calls a deviation" and
     "what the gate calls ungated" one predicate by construction rather than two
     implementations that agree today: both now derive the injected set the same
     way, from the same installed-pack manifests, and hand it to the same helper.
     On the degraded path (`ctx.PacksErr != nil`) doctor injects the zero value,
     so the set is the tool-agnostic base ONLY — the named consequence REQ-007
     states, not a defensive default.

4. **Rendering and exit (REQ-001, REQ-003).** Both renderers consume the same
   `[]doctorResult`. Human form prints one line per check plus its remediation;
   `--json` marshals the payload with `schema_version: doctor/v1`, and every
   per-check object carries all five keys including `remediation` — no
   `omitempty`, so the payload's shape does not vary with a check's status. The
   `--json` flag is the ROOT persistent one, taken by pointer in
   `newDoctorCommand(jsonFlag *bool)` exactly as `newGateCommand` and the nine
   pack commands take it (`root.go:25,42,91`); declaring a local one would shadow
   it and break `backstop --json doctor`. The exit code is
   computed from the same slice: `ExitViolations` if any status is `fail`, nil
   otherwise. There is no branch that returns `ExitConfigError`.

5. **The `--check` selector (REQ-001).** Selection filters the registry by id
   before running. An id no entry carries is an error listing the registered ids —
   never an empty, successful run, which would be a diagnostic reporting nothing
   and calling it fine.

6. **Init's guidance (REQ-004).** `doctorGuidanceForSteps(steps
   []initialize.StepReport) []string` in `cmd/backstop/init.go` maps a FAILED
   `toolchain` step report to `doctorGuidance(doctorCheckToolchainRuns)` and
   returns nothing for every other step. Two facts fix where this lives:
   `doctorRegistry` is unexported in package `main`, so `pkg/initialize` cannot
   reach it and an export-for-import would invert the dependency; and
   `cmd/backstop/init.go` is already where SPEC-069 renders init's report from
   the `initialize.Result` its `Runner.Run` returns, so the guidance is rendered
   in the layer that already does the rendering, from the same structured
   `StepReport` data. `doctorGuidance` resolves the id against the registry and
   returns no text for an unregistered one, so a rename cannot leave init
   printing a check that no longer exists. Init's own steps and their failure
   text belong to SPEC-069; what belongs here is that the id it prints is a
   registered one and that no other step attracts guidance.

7. **Out of scope, by name.** The build-identity STAMP and the capability-gap
   diagnostic (bundle REQ-021 / REQ-022, SPEC-068 and BUNDLE-020); the artifact
   root RESOLVER itself (bundle REQ-029, SPEC-068); every init step (SPEC-069);
   the runtime-version-versus-stack-policy check (bundle REQ-024 — see
   Dependencies); and any orphan-reservation-tag check (ranked sharp edge 9, a
   founder call the bundle explicitly did not take).

## Verification

Verification configuration is in frontmatter: integration level, an 80% coverage
threshold, and `go test ./cmd/backstop/... -race`. Integration rather than unit is
the honest level — the toolchain check EXECUTES real commands and the layout check
walks a real tree, and a doctor proven only against stubs would repeat the exact
mistake DD-6 is about. 80 is `cmd/backstop`'s spec-derived floor, not a discount.

Three verification properties are worth stating so a plan does not weaken them. The
seven `gate_type` claims must be proven against fixture packs declaring each of the
seven types with an observably side-effecting command, so "not executed" is
asserted by the side effect's ABSENCE rather than by reading the implementation's
selection list. The gate-exits-2 / doctor-exits-1 claim must run the SAME broken
`backstop.yml` through both `gate` and `doctor` in one test, since the claim is
about the difference between the two, not about doctor's exit code alone. And the
four setup checks need fixture PROJECTS, not stubs — an empty directory with no
`backstop.yml` and no `.git`, one with a malformed `backstop.yml`, one declaring a
pack absent from `.backstop/packs/`, one declaring no packs — because the property
under test is that doctor's context gathering survives each of them, and a stubbed
context proves nothing about the gathering that would abort.

CLM-058 and CLM-059 are structural: they read this package's non-test Go source
(every `doctorRegistry(` call site and the function enclosing it, which must be
`runDoctor` or `doctorGuidance` and nothing else; check-id string literals) rather than running
doctor, which is why they carry `kind: absence`. They are cheap and they are the
only mechanical proof that CLM-010's "all consumers read the same registry" holds
by construction rather than by today's code happening to agree.

## Sharp Edges

- **REQ-024 has no data source, and no owner anywhere.** Doctor is required to
  compare the installed runtime against pack-declared stack policy. No such
  manifest surface exists (`pkg/pack/manifest.go`, verified at HEAD), the real
  `typescript-toolchain@1.2.1` declares no runtime version, and no bundle,
  directive, spec, or issue claims the surface. Implementing it means DEFINING it,
  which is BUNDLE-004's territory by the bundle's own REQ-005 ruling. There is a
  second-order problem even if the surface existed: reading a runtime's version
  means running a command against software already on the machine, which is the
  ungoverned third execution surface BUNDLE-021 (`exploring`) exists to settle and
  the same reasoning that retired REQ-032. This is escalated in Dependencies, not
  absorbed. A reviewer should treat a future spec that adds a `stack_policy:` block
  without BUNDLE-004 adopting it as scope creep, not as this gap being closed.

- **The setup-owed / entrypoint-failed distinction is one signal wide.** REQ-011
  v1.1.0 obliges naming an uninstalled toolchain as owed SETUP, but the only
  mechanically honest signal is "the executable could not be started" versus "it
  started and exited nonzero". A repo whose dependencies are missing in a way that
  still lets the launcher start — an installed `npx` resolving nothing, a test
  runner exiting nonzero on a missing module — lands in case (c) and is reported
  verbatim rather than classified as owed setup. That is deliberate. The pnpm
  `ERR_PNPM_IGNORED_BUILDS` evidence is exactly a nonzero exit whose obvious
  reading was wrong, and a doctor that guesses causes from exit codes reproduces
  the misdiagnosis it exists to end.

- **Executing pack-declared commands is the sharpest thing doctor does.** The
  toolchain check runs consumer-supplied commands with ambient permissions. It is
  bounded to the gate's own path, reached through the shared
  `packEntrypointProber` — trusted-tool allowlist gate FIRST, then
  `strings.Fields`-based splitting of the declared command, then the shared
  runner, and never a shell — so doctor adds no new attack surface, but it does add
  a new TRIGGER for that surface: a user who has not run `gate` can now execute pack
  commands by running a command whose name promises diagnosis. Those guarantees are
  now the PROBER's to hold rather than this check's own inline logic, which
  strengthens rather than weakens them: one bounded route serves both `doctor` and
  `init`, so there is a single place to audit and a single place a regression could
  appear. If BUNDLE-021 lands a posture on pack command execution, this check is one
  of the call sites it must cover, and a plan must not "simplify" it to a direct
  `exec.Command` — nor re-inline the prober's steps here, which would recreate the
  second execution route the extraction removed.

- **"Outside the resolved root" is the WRONG test, and it fails silently on the
  exact repo shape this check exists for.** An unconfigured root resolves to the
  project root, which contains everything — so a consumer holding artifacts under
  `.backstop/` with no `artifact_root` declared, precisely the live
  backstop-runtime shape bundle REQ-030 cites, has zero files "outside the root"
  while none of them are discovered or gated. Both specs now decide per KIND
  instead: SPEC-068's first draft carried the containment reading (and a claim
  asserting the vacuity as if it were correct), corrected in its 1.1.0 after this
  was raised, which is also when `FindOutOfRootArtifacts` became
  `FindUngatedArtifacts` because the old name encoded the wrong predicate. An
  implementer who "simplifies" back to root containment gets a check that passes
  everywhere and detects the one thing it was built for nowhere — and no in-repo
  run will show it, because backstop-core is clean under both readings (measured:
  116 issues, 102 plans, 51 specs, 27 directives, 26 bundles, 18 adrs, zero
  strays). Both specs need fixture projects; neither can prove this by dogfood.

- **The classification trap in this repository's own corpus: do not "fix" the
  capability pattern.** `capabilities/CAP-001-pack-gate-enforcement/capability.yml`
  is BOTH nested a directory deeper than the layout expects and named bare rather
  than `CAP-001-….capability.yml` (verified on disk). It escapes classification
  today only because a `.capability.yml` suffix test does not match a bare
  `capability.yml`, and it must keep escaping: the moment the pattern is widened
  to catch the bare name, this repository's own gate reports a deviation on a file
  neither this spec nor SPEC-068 was asked to touch. Doctor inherits the behavior
  by consuming the shared classifier, and must not add a pattern of its own.
  Directness matters for the same reason and is not pedantry — `walkArtifactDir`
  reads its directory with `os.ReadDir` and skips subdirectories
  (`pkg/gate/artifact_status.go:379-404`), so an artifact one level below its
  expected directory is exactly as ungated as one in the wrong tree.

- **The layout check states two remedies and picks neither, deliberately.** A
  deviation can be fixed by moving the file or by declaring the root that makes
  its location expected, and which is right is the consumer's layout decision.
  SPEC-068's resolution is policy-FREE — it resolves whatever is declared and has
  no notion that `.backstop/` is canonical — so any canonical-layout opinion is
  doctor's own. This spec keeps that opinion in remediation TEXT rather than in a
  status, because a warn-when-unconfigured rule would fire forever in
  backstop-core, and the fix for that must never be a special case naming this
  repository inside core.

- **A layout deviation is reported, never repaired, and never widens discovery.**
  Bundle REQ-030's rule for the gate is that out-of-root artifacts are REPORTED,
  not adopted. Doctor must hold the same line: naming an out-of-root spec is not
  an invitation to scan it, move it, or count it. A "helpful" doctor that widened
  the root would manufacture exactly the silent-undiscovery-inverse — files gated
  from a location the gate does not read.

- **`--check` must never become a way to claim green.** It exists because REQ-023
  wants the toolchain check re-runnable standalone. Doctor produces no verdict the
  gate or any artifact consumes, so a filtered run cannot certify anything today.
  If doctor output is ever wired into a gate step or a baseline, this flag becomes
  a filtering mechanism over enforcement and the razor against check filtering
  applies to it.

- **Eight checks is the declared set, not a starting point — and growth requires a
  RECORDED authorization, not an implementer's judgement.** REQ-020's "one per
  ranked sharp edge" is satisfied by the mapping table above, including the rows
  that route elsewhere; four more exist because REQ-003's conditions need an
  owner; and the eighth, `engine-tools-present`, exists because DIR-002's
  founder-ruled scope expansion of 2026-08-16 admitted doctor's findings-engine
  tool-detection coverage into that directive's charter and ISSUE-134 /
  `PLAN-ISSUE-134` delivered it. THAT is the shape a legitimate ninth must take:
  a directive or bundle that admits it, an artifact that delivers it, and this
  section amended to say so. The temptation for an implementer is to add the
  obvious missing ones (orphan reservation tags, gitleaks presence, a `.gitignore`
  completeness check) on their own initiative. Each of those is either an init
  guardrail already assigned or an explicitly untaken founder call. CLM-051 is the
  tripwire: it asserts the registry holds exactly the declared ids, so an
  unauthorized addition reds rather than ships — the eighth check passed through
  that tripwire deliberately, by amending the declared set here first.

- **A local `--json` flag on doctor would work in every test that types it and
  fail the one invocation users type.** `--json` is a ROOT PERSISTENT flag
  (`root.go:25,42`) and every JSON-rendering command takes it by pointer
  (`newGateCommand(&jsonFlag)` and the nine `newPack*Command(&jsonFlag)`,
  `root.go:91`). A `BoolVar` on doctor's own flag set would shadow it: `backstop
  doctor --json` would pass, `backstop --json doctor` would silently render human
  text. CLM-061 asserts the ROOT position specifically, because the sub-command
  position is the one that keeps working.

- **A single absent `backstop.yml` could plausibly fail five checks, and must
  fail exactly one.** `config-loads`, `packs-installed`, `toolchain-runs` and
  `engine-tools-present` all
  read inputs that a missing config makes unavailable, and the reflexive
  implementation reports each as its own failure. That report tells a consumer
  five things are broken when one is, and it makes CLM-016's exit comparison
  meaningless because doctor would exit 1 for reasons unrelated to the condition
  under test. The rule is one condition, one owner, everything downstream
  `skipped` naming that owner — which is also the only natural producer of the
  `skipped` status REQ-002 declares. A plan that folds these back into fewer
  checks, or that lets each check gather its own config, reintroduces this.

- **`git-repository` warning rather than failing is a decision, not an
  oversight.** No git means the diff scope falls back to a full sweep with a
  warning (`pkg/gate/scope.go:220-222`) and `artifact new` cannot reserve an id
  through a tag — degraded, but backstop runs. Making it a failure would exit 1
  in every non-repo directory, including deliberate ones. Making it silent would
  hide the id-reservation loss. The consequence a reviewer should hold this to:
  no test may assert exit 1 from a non-repo directory alone.

- **Registering `doctor` edits ANOTHER SPEC'S ANTI-REGRESSION PIN, in a file no
  task in this spec's plan declares — and both halves of that are correct.**
  `cmd/backstop/ci_recipes_mechanism_test.go` belongs to SPEC-067 (its
  REQ-006/REQ-007/REQ-008 proof). Its
  `TestCIRecipes_RegisteredCommandSurfaceUnchanged` (CLM-052) enumerates the CLI's
  ENTIRE top-level command set and asserts exact whole-set equality, so the
  one-line `root.go` registration that makes `doctor` reachable necessarily turns
  that test red until the expected set names it. It now does: `doctor` was added
  to the pinned list and the attributing comment extended to read "`init` is
  SPEC-069's and `doctor` is SPEC-070's, not this spec's." **The assertion was NOT
  weakened** — no exemption predicate, no membership-only relaxation, still exact
  whole-set equality — so an unexplained future addition still fails it. Only the
  expected set grew by one, attributed by name. That is the pin working as its own
  comment prescribes, not the pin being violated; SPEC-069 set the precedent for
  `init` (its Sharp Edge 22) and this is the same move. **The non-obvious half:
  this file appears in NO task's `files:` list in PLAN-SPEC-070**, so any
  file-scope audit of that plan will report the edit as out of scope. That is an
  inherent property of a whole-set anti-regression pin rather than a defect in the
  plan's scoping — the pin lives in the spec that WROTE it, and every later spec
  that adds a top-level command hits the identical mismatch, however honestly its
  own file scope is drawn. A reviewer should read an out-of-scope touch of THIS
  file, adding THIS spec's command by name with attribution, as expected; what
  should still draw scrutiny is a touch that removes an entry, relaxes the equality
  to a subset check, or adds a name without saying whose it is.

## Review Questions

1. Does `checkArtifactLayout` contain any literal artifact directory name —
   `specs`, `bundles`, `issues`, `directives`, `plans` — or does it obtain every
   path from SPEC-068's resolver? A single literal makes it the fourth hardcoding
   REQ-029 exists to delete, and it will not fail any test that only asserts
   output.

2. Do the five `gate_type` non-execution claims assert the ABSENCE of a side effect
   from a real fixture command, or do they merely assert the selection list?
   Asserting the list tests the implementation against itself.

3. Does `artifact-layout` decide deviation per KIND (`Root.Dir(kind)`), or has it
   been reduced to "outside the resolved root"? The reduced form passes on a
   `.backstop/`-rooted repo with no configured root — the one shape the check
   exists for — and no in-repo run will reveal it, because backstop-core is clean
   under both readings.

4. Does any doctor path return `ExitConfigError`, or call a config/pack loader in a
   way that aborts before the registry runs? REQ-003 is falsified by one such path,
   and it will look like ordinary defensive error handling in review.

5. Does the toolchain check execute anything other than `gate_type: test` and
   `gate_type: build` bindings — in particular, does it run a `coverage` engine
   because the same command happens to appear there? The real
   `typescript-toolchain` declares near-identical vitest commands under both `test`
   and `coverage`; selection must key on the declared type, never on the command
   string.

6. Does init's guidance obtain check ids from `doctorRegistry()`, or does it carry
   its own string literals that merely happen to match today (REQ-004/CLM-020)?

7. Was anything added to close bundle REQ-024 — a `stack_policy:` manifest block, a
   runtime-version probe, a hardcoded LTS list? Any of those is out of scope for
   this spec and must be escalated rather than merged.

8. Does any of the four setup conditions get reported OUTSIDE the registry — an
   early `if cfgErr != nil { return ... }` in `runDoctor`, a printed warning
   before the checks run? Such a path passes a human-output test and is invisible
   in the `--json` payload and to `--check`, which is precisely the
   refuse-to-start behavior REQ-003 forbids.

9. On a directory with no `backstop.yml`, how many checks report a FAILURE? The
   answer must be one (`config-present`); `config-loads`, `packs-installed`,
   `toolchain-runs` and `engine-tools-present` must be `skipped`, each naming the
   check that owns the condition. Five failures for one cause is the shape to
   reject. `engine-tools-present` in particular must skip on ANY of
   `ConfigPathErr`, `ConfigErr`, `PacksErr` — a `PacksErr`-only predicate leaves
   it reporting "no installed pack binds an engine tool" on a project whose packs
   were never gathered, which is a WARN masquerading as a finding.

10. Does `newDoctorCommand` declare its own `--json`? A local declaration shadows
    the root persistent flag, and every sub-command-position test still passes.

11. Does `doctorGuidanceForSteps` live in `cmd/backstop/init.go`, and does it
    obtain both the id and the guidance text from `doctor.go` — the constant and
    `doctorGuidance` — rather than formatting its own string? A hand-formatted
    `"run backstop doctor --check toolchain-runs"` is the second literal REQ-004
    exists to prevent, and it will match the registry on the day it is written.

12. Does the `remediation` JSON tag carry `omitempty`? REQ-001 requires the key
    for every check that ran, so a passing check must emit an empty string rather
    than drop the key.

## Dependencies

- **SPEC-068 (trustworthy-green guards) must land first.** REQ-005 surfaces the
  build commit and build date bundle REQ-021 stamps. REQ-007 consumes bundle
  REQ-029's resolution as SPEC-068 declares it: `artifact.ResolveRoot(projectRoot,
  declared string) (Root, error)` in `pkg/artifact`, with `Root{Path, Declared,
  Configured}`, `Kind`/`LayoutFor`/`ClassifyFilename`/`Root.Dir(kind)`, the typed
  `*RootMissingError` / `*RootInvalidError`, the new `Config.ArtifactRoot` field
  and its `artifact_root` schema entry, and `pkg/gate`'s
  `FindUngatedArtifacts(projectRoot string, root artifact.Root, nonCorpus artifact.NonCorpusDirs) ([]UngatedArtifact, error)`
  returning `UngatedArtifact{Path, Kind, ExpectedDir, Root}` (the third
  parameter is not SPEC-068's: `artifact.NonCorpusDirs` and the injection were
  added by ISSUE-122 after both specs landed, and the return type and field list
  are unchanged by it). Those signatures are
  DECIDED-IN-SPEC, not built — SPEC-068 is `status: draft` and they can still move
  in its review, as the helper already did once (it was
  `FindOutOfRootArtifacts`/`ExcludedBy` in that spec's 1.0.0), so this
  spec's plan must re-read them rather than trust this citation. Doctor needs no
  prefix→Kind helper: it classifies by filename, not by typed-ref prefix. This is
  also the bundle's declared seed order (guards → init → doctor) and a real
  build-order dependency, not a preference.
- **SPEC-069 (`backstop init`) must land first for REQ-004.** CLM-019, CLM-020 and
  CLM-060 assert init's printed guidance, rendered in `cmd/backstop/init.go` —
  the file SPEC-069 declares for `newInitCommand` — from the
  `initialize.Result{Steps []StepReport}` its `Runner.Run` returns. Two of that
  spec's decisions are consumed here and must be re-read rather than trusted from
  this citation, since SPEC-069 is also `draft`: the step identifier for the
  toolchain probe (its step table names it `toolchain`, step 7 of ten, one
  `StepReport` each) and the `Outcome` constant that denotes a failed step, which
  SPEC-069 declares as a type without naming its values. If either moves,
  `doctorGuidanceForSteps` moves with it; the doctor-side contract — the id
  constant and the `doctorGuidance` lookup — does not.
  Doctor is last in the declared seed order, so
  init exists by then; if the order is changed, REQ-004's claims move with
  init. Confirmed with SPEC-069: it specs no init-points-at-doctor requirement,
  claim, or test, so REQ-004 is solely owned here. The boundary runs the other
  way too — SPEC-069's `TestInit_ImplementsNoCIDetectionOrBespokeGuidance` forbids
  init adding guidance to its CI steps, which is why REQ-004 makes the registered
  check set the test of "diagnosable" and excludes the CI steps explicitly.
- **No `follows:` standard-rule bindings appear on any requirement, deliberately.**
  The in-repo `standards/` tree carries only empty `core/` and `typescript/`
  skeletons, and `STD-GO-001` — the standard older specs bind to — was deleted when
  SPEC-030 retired the native standards compiler. Binding to it would cite a
  standard that no longer exists. Per escalation-over-guessing this is stated
  rather than filled with an invented mapping; if a standards pack covering Go CLI
  code is adopted, these requirements are the ones to bind.
- **Bundle REQ-024 — CARVED OUT, blocked on an unowned pack-manifest surface.**
  Escalated during authoring and ruled 2026-08-13: carve it out rather than invent
  the field, on the same rule the bundle already applied to REQ-005 v1.1.0 —
  manifest surface design is BUNDLE-004's. The pack-declared stack-policy surface
  does not exist (verified at HEAD) and no artifact owns creating it. Closing it
  requires (a) BUNDLE-004 adopting a manifest surface for stack policy, and
  probably (b) BUNDLE-021 settling the posture toward executing a version probe
  against software already installed on the machine. Until both have owners a
  doctor check for it would either read nothing — a vacuous check — or invent
  another bundle's surface. REQ-024 therefore carries NO requirement and NO
  mandated claim in this spec, and should be specced once an owner exists.
  ISSUE-121 was filed for the gap and is being homed under BUNDLE-004, so it does
  not live only as a note here. One tripwire IS carried, mirroring how SPEC-069
  guards its own unowned coverage-floor knob: the absence claim on REQ-002 asserts
  the registry holds exactly the declared set of check ids and reads no
  stack-policy surface, so an implementer cannot close REQ-024 from the wrong
  artifact without a test going red.

## References

- `bundles/BUNDLE-003-onboarding-experience.bundle.md` — source bundle
  (v0.10.2, `defined`). The doctor seed (~line 1489) and its 0.10.0 correction
  moving REQ-021/REQ-022 out; the `backstop doctor` requirement narrative
  (~line 1021); REQ-020 / REQ-023 / REQ-024 / REQ-025 text (~lines 532, 637-656);
  DD-6 (toolchain runs, ~line 1163), DD-8 and its corollary (~line 1194), DD-9
  (~line 1216), DD-13 (~line 1281), DD-15 (~line 1292); the REQ-005 v1.1.0
  correction establishing that manifest surface design is BUNDLE-004's
  (~line 131); the untaken orphan-tag founder call (~line 1105).
- `~/src/projects/backstop-packs/BACKSTOP-INIT-DOGFOOD-FULL-SDLC.md` — the
  full-SDLC write-up whose "Sharp edges (ranked by pain)" list is REQ-020's
  coverage bar; and `~/src/projects/backstop-packs/BACKSTOP-INIT-REQUIREMENTS.md`
  — the pack-only sibling. Both are untracked field notes; the bundle is their
  durable home.
- `specs/SPEC-068-trustworthy-green-guards.spec.md` — the build-identity stamp and
  the `pkg/artifact` resolution (`ResolveRoot`, `Root`, `Root.Dir`,
  `ClassifyFilename`, the typed root errors, `Config.ArtifactRoot`) plus
  `pkg/gate`'s `FindUngatedArtifacts`, all consumed here and none restated. Its
  REQ-007 also fixes a fourth root hardcoding the bundle does not name, verified
  here when this spec was written: `cmd/backstop/artifact_discover.go` then
  skipped `.backstop` from an UNCONDITIONAL `switch base` list (`testdata`,
  `vendor`, `node_modules`, `.git`, `.backstop`, `prototype`) that no
  configuration reached, and `artifact_validate.go:132-133` returns
  `ValidateResult{Pass: true}` on an empty discovery, which `gate.go:1015` feeds
  from as well — so a `.backstop/`-rooted repo validated zero artifacts and
  reported green. BOTH halves have since landed and the citation is historical:
  SPEC-068 replaced the switch with the shared resolution, and ISSUE-122 removed
  the two baked ecosystem names on it (`node_modules`, `vendor`) that were a
  DD-13 concern raised separately here — that concern is RESOLVED by ISSUE-122,
  not outstanding. `DiscoverArtifacts` now takes the exclusion set as an
  `artifact.NonCorpusDirs` PARAMETER — core's tool-agnostic base (`.git`,
  `testdata`, `prototype`) unioned with the pack-declared
  `classification.dependency_dirs` names — and keeps only its two root-relative
  `.backstop` rules local to the walk.
- `specs/SPEC-069-backstop-init.spec.md` — init. It specs no init-points-at-doctor
  requirement or claim (REQ-004 is solely this spec's), and its
  `TestInit_ImplementsNoCIDetectionOrBespokeGuidance` is the guard REQ-004 must
  not put init in violation of.
- `issues/ISSUE-121` — the filed gap for bundle REQ-024's absent pack stack-policy
  surface, being homed under BUNDLE-004.
- `cmd/backstop/root.go:106-127` — the `version` command's resolved version and
  cohort, what `build-identity` surfaces. `root.go:154` — the `AddCommand` list
  doctor joins (NOT `:144`, which is inside the `commands` discovery handler).
  `root.go:25,42,91` — the persistent `--json` declaration and the by-pointer
  convention `newDoctorCommand` follows.
- `cmd/backstop/gate.go:67-73` — `runGate`'s exit-2-on-config-failure, which
  doctor inverts; `gate.go:75-80` — its project-root fallback, which doctor
  mirrors.
- `cmd/backstop/pack_gate.go:147-176,573` — `loadInstalledPacks` (whose
  empty-slice-vs-error split is what `packs-installed` reads) and
  `runFindingsEngine` (the three execution steps `toolchain-runs` reaches without
  the SARIF parse).
- `cmd/backstop/pack_entrypoint_prober.go` — `packEntrypointProber`, THE one
  execution route for pack-declared test/build entrypoints, extracted by
  SPEC-069/PLAN-SPEC-069 phase 14 and consumed BY NAME here. It holds the
  selection, the allowlist gate, the command split, the runner call and the
  started-versus-exited-nonzero classification, so `toolchain-runs` performs none
  of them itself. `cmd/backstop/init_toolchain.go` is its other caller — a thin
  adapter over the same type — which is what makes "one execution route, two
  callers" true rather than aspirational.
- `cmd/backstop/pack_gate_provision.go` — `collectRequiredEngineTools`, THE one
  authority on which findings-engine tools a pack set requires (the manifest RULE
  walk, the `resolveEngineRegistry` lookup, the `checkEngineToolAllowed` trust
  gate, the argv[0] extraction, the dedupe and the sort), consumed BY NAME by
  `engine-tools-present`; `provisionEngines` is its other consumer and keeps its
  exact prior signature and observable behavior, owning only the gate's
  disposition — the same "one authority, two consumers" shape
  `packEntrypointProber` has. `absentToolMessage` is the shared remediation
  renderer both use. `resolveBinaryResolver` is the presence probe doctor calls
  and the only tool contact it makes.
- `directives/DIR-002-backstop-init.directive.md` — the founder-ruled scope
  expansion of 2026-08-16 (the ISSUE-134 follow-on paragraph and "Founder-approved
  home and framing") that authorized the eighth check by bringing doctor's
  findings-engine tool-detection diagnostic coverage into that directive's charter
  as an ongoing concern. Without this citation an eighth check reads as exactly the
  quiet addition CLM-051 exists to catch.
- `issues/ISSUE-134` and `plans/PLAN-ISSUE-134-doctor-findings-engine-tool-blindspot.plan.yml`
  — the defect (doctor exit 0 all-green while `backstop gate` refused at exit 2
  naming the absent tool) and the plan that delivered `engine-tools-present`,
  which OWNS that check's behavioral claims and mandated test names; this spec
  owns only its registry membership.
- `pkg/check/scope.go:37-48` — `DefaultGitExecutor.IsGitRepo`, the existing
  exported work-tree detector `git-repository` consumes; `pkg/gate/scope.go:220-222`
  — the gate's own warn-and-fall-back-to-`--all` behavior on a non-repo, which is
  why that check warns rather than fails.
- `cmd/backstop/artifact_discover.go:18-26` — the SEVEN-kind filename vocabulary
  (`spec`, `plan`, `adr`, `bundle`, `issue`, `directive`, `capability`) CLM-041
  enumerates; the `capability` entry matches `.capability.yml` on `:25`, which is
  why the bare `capability.yml` of CAP-001 escapes and must keep escaping.
- `pkg/pack/engine/gatetype.go` — the seven declared gate types the REQ-006 matrix
  covers.
- `pkg/pack/manifest.go` — verified at HEAD to carry NO stack-policy surface, which
  is why bundle REQ-024 is escalated rather than specced.
- `bundles/BUNDLE-004-pack-manifest-authoring.bundle.md` (`ready`) — owner of pack
  manifest surface design; `bundles/BUNDLE-021-pack-command-execution-governance.bundle.md`
  (`exploring`) — owner of the posture toward executing pack-declared commands
  against machine-installed software.

## Version History

- **1.1.6** (2026-08-16) — **THE EIGHTH CHECK: `engine-tools-present` is admitted to the
  declared set, with its authorization recorded.** This is a RECONCILIATION of an
  `implemented` spec against work that has already landed (PLAN-ISSUE-134, TASK-010), not
  new design. No claim is added, deleted or reworded; no mandated test name changes; no
  requirement is added or removed.
  **What moved.** REQ-002's enumeration now holds eight ids in report order, with
  `engine-tools-present` between `toolchain-runs` and `artifact-layout` — the position the
  registry actually ships. Every count of the declared set was corrected from seven to
  eight: the implementation summary, the narrative intro, REQ-002, the `doctorCheck` and
  `doctorRegistry` contract notes, Implementation §1 and §3, and the growth sharp edge. The
  `doctorCheck` note's grouped-const enumeration gains `doctorCheckEngineTools`.
  Implementation §3 gains the check's own bullet; Requirements gains a subsection recording
  the gap, the authority and the ownership split; References gains
  `pack_gate_provision.go`, DIR-002 and ISSUE-134/PLAN-ISSUE-134.
  **THREE unrelated "seven"s were deliberately NOT touched**, because a mechanical
  seven→eight substitution corrupts them: the SEVEN `gate_type` values the REQ-006 matrix
  covers (`pkg/pack/engine/gatetype.go` still declares seven and this change adds none); the
  SEVEN-kind artifact filename vocabulary CLM-041 enumerates; and every "seven" inside a
  version-history entry below, which records what a PAST revision did and is wrong the
  moment it is edited to match the present.
  **THE `provides` DISPOSITION — (a), DECLARED.** An eighth `provides` entry for
  `checkEngineToolsPresent` (`kind: function`, matching the existing seven's format) was
  ADDED to the `cmd/backstop/doctor_checks.go` contract, because that block is kept 1:1 with
  the registry and an eighth check with no entry would silently break the correspondence
  that makes the block readable as the check inventory. No grep of this file could have
  surfaced that gap — the word "seven" appears nowhere near those entries. Two `consumes`
  entries were added alongside it, `collectRequiredEngineTools` and `absentToolMessage`,
  both genuinely referenced by the file. The grouped id-const block still carries NO
  `provides` entry of its own: the v1.1.4 disposition (ISSUE-078 — a member of a grouped
  `const (…)` block is structurally inexpressible to the contracts pack's signature
  compiler) is UNCHANGED and was not reversed while adding the eighth function entry.
  **THE AUTHORITY.** DIR-002's founder-ruled scope expansion of 2026-08-16
  (`directives/DIR-002-backstop-init.directive.md`, the ISSUE-134 follow-on and the
  "Founder-approved home and framing" paragraph) brought doctor's findings-engine
  tool-detection diagnostic coverage into that directive's charter as an ongoing concern.
  It is cited in three places rather than one, because the spec's own growth sharp edge
  says growth belongs to the bundle, and an eighth check found with no recorded
  authorization would be read — correctly — as the quiet addition CLM-051 forbids.
  **OWNERSHIP, stated so it is not inferred.** This spec owns the eighth check's REGISTRY
  MEMBERSHIP only: REQ-002's enumeration, its report position (CLM-007), the
  exactly-this-set tripwire (CLM-051) and its contract entry. Its BEHAVIORAL claims and
  mandated test names are owned by ISSUE-134 / PLAN-ISSUE-134 on the reactive
  `issue -> plan` track and are deliberately NOT restated here — two owners for one
  guarantee is how a guarantee stops having one.
  **CLM-051 is UNCHANGED and still true.** Its text asserts the registry holds exactly the
  DECLARED set and registers no stack-policy check; only the size of that set moved. Its
  mandated test name, `TestDoctor_RegistersNoStackPolicyCheckAndReadsNoStackPolicySurface`,
  is untouched, and arm (b)'s stack-policy source scan is intact and unweakened. Bundle
  REQ-024 remains CARVED OUT with ISSUE-121 still owning it — nothing here closes it.
  **Two prose consequences of the eighth check that are behavior, not bookkeeping.** The
  absent-`backstop.yml` sharp edge and Review Question 9 now name FOUR downstream checks
  that must skip rather than three: `engine-tools-present` skips on ANY of `ConfigPathErr`,
  `ConfigErr`, `PacksErr`, the identical predicate `checkToolchainRuns` carries, because a
  `PacksErr`-only predicate reports "no installed pack binds an engine tool" on a project
  whose packs were never gathered.

- **1.1.5** (2026-08-15) — **CLOSE-OUT: status `draft` -> `implemented`.** No requirement,
  claim, contract, test, or mechanism is added, removed, or reworded; all 64 claims and 65
  mandated test names stand exactly as 1.1.3 left them. This entry records the evidence the
  flip rests on, and that evidence was RE-VERIFIED in this session rather than copied forward
  from the implementation report.
  **Lineage.** `PLAN-SPEC-070-backstop-doctor.plan.yml` is `completed`, with all 28 tasks
  across all 7 phases delivered. The implementation went through TWO independent impl-review
  passes. The first returned FAIL on three findings, one of them a real SPEC CONTRADICTION
  rather than a mechanical slip: `checkToolchainRuns` read an UNGATHERED pack set as outcome
  (d), "no entrypoint declared", instead of SKIPPING — contradicting REQ-006's explicit
  prohibition and Review Question 9's stated matrix. The other two were a guidance line
  printed once per failing entrypoint instead of once per CHECK, and CLM-051's stack-policy
  scan matching `lts` as a bare substring (false-positiving on `results`, `defaults`,
  `faults`). All three were fixed and independently re-verified in a targeted delta re-review
  that returned CLEAN. The blocker fix was FALSIFIED rather than asserted: reverting to the
  narrower skip predicate in a scratch copy reds exactly the two new table rows while every
  original row stays green, proving the fix load-bearing.
  **Build and tests.** `go build ./...` and `go vet ./...` are clean across the repo. This
  spec's own declared `test_command` — `go test ./cmd/backstop/... -race` — passes with zero
  failures at close-out.
  **Mandated tests.** Every test name in this spec's `claims` block was confirmed PRESENT in
  the tree by name at close-out time, by scanning the tree rather than trusting the
  implementation report: 64 claims carrying 65 distinct test names (one claim mandates two).
  Zero missing. The two impl-review rounds established that none are hollow, by mutation
  rather than by presence.
  **Claim subjects — the SPEC-069 defect class, checked and ABSENT here.** Closing SPEC-069
  surfaced a class of latent defect (its 1.3.3): a claim carrying no `subject:` inherits the
  spec-level `implementation.subject`, and `test_substantiveness` enforces only at
  `implemented` status, so a wrong inherited subject stays invisible until the flip. Every one
  of this spec's 64 claims was audited against that: none declares a `subject:`, all 65
  mandated tests live in `cmd/backstop`, and this spec's `implementation.subject` IS
  `cmd/backstop` — so the substantiveness subject join is satisfied by structural colocation
  for every claim, not by incidental symbol reference. Three claims additionally carry
  `kind: absence` (CLM-051, CLM-058 and CLM-059) and are exempt from the join by design. The
  defect does not recur here.
  **Coverage.** Measured at close-out against this spec's declared 80 floor: `cmd/backstop`
  91.9%. Every new or edited file also clears the repo's per-file floor.
  **Gate (DIFF-SCOPED).** The diff-scoped `./bin/backstop gate` — bare, i.e. scoped to the
  diff vs merge-base plus untracked files, which is the scope actually run and verified for
  this close-out — exits 0 with every blocking dimension green AFTER the 1.1.4 fix:
  `pack_lock_verification`, `artifact_validation`, `test_verification`, `test_substantiveness`,
  `coverage_threshold`, `contract_signature`, `artifact_status_drift`,
  `requirement_traceability` and `waiver_resolution` all pass with zero violations. Every
  residual finding is a non-blocking warning and each was ATTRIBUTED rather than waved past:
  23 `pack_engines` warnings, all 23 in `pkg/scaffold/idresolver_test.go` — a different plan's
  file, and ISSUE-125's known constructor-injection false positive; 178
  `requirement_traceability_advisory` and 2 `artifact_status_drift_advisory` entries, this
  project's standing repo-wide advisories. Following SPEC-068 1.2.9's and SPEC-069 1.3.4's
  precedent, this is deliberately NOT a claim about `./bin/backstop gate --all`: the full sweep
  is red, and that red was PROVEN inherited rather than assumed — a HEAD control run with none
  of this spec's code present reports the same ~194 violations.
  **What the flip closed.** Bundle `onboarding-experience` REQ-020, REQ-023 and REQ-025 —
  this spec's three `supports` targets — now have implemented-spec coverage and have dropped
  out of the traceability advisory set. Two of the bundle's requirements in this neighbourhood
  remain uncovered and are NOT this spec's to close: REQ-024, the carve-out named below, and
  REQ-022, which moved to SPEC-068 at bundle v0.10.0 but carries no `supports` ref there
  (SPEC-068 declares REQ-021 and REQ-026..029). The REQ-022 gap is surfaced here rather than
  silently absorbed; it belongs to SPEC-068's seam.
  **Contract enforcement — and the defect it caught.** Contract declarations are collected
  only for `implemented` specs, so this flip is the first moment this spec's `contracts` block
  — including the `doctor_checks.go` `consumes:` entry 1.1.2 rewrote around the shared
  `packEntrypointProber` — is enforced at all. PLAN-SPEC-070's AS-BUILT block flagged exactly
  that gap and said every green it reported was silent about it. It was right: the flip turned
  `contract_signature` RED with one real violation, a provide naming a symbol Go cannot have.
  That is fixed in 1.1.4 above, spec-side only, and the dimension is green after it. Both
  `artifact validate` and the diff-scoped gate were therefore run AFTER the flip, not before,
  and those post-flip runs are the readings recorded here.
  **Known open items, expected and NOT resolved here.** (1) Bundle REQ-024 remains the
  CARVED-OUT, UNOWNED gap this spec has named since 1.0.0: the pack-declared stack-policy
  surface it would read does not exist and no bundle, directive, spec, or issue owns it, so it
  carries no requirement and no claim here. Closing this spec does not close it, and CLM-051's
  tripwire remains the mechanical guard against a future spec quietly adding a `stack_policy:`
  block. (2) ISSUE-129 is OPEN and is explicitly not this spec's to fix: `backstop gate`'s
  diff-scoped mode can report PASS while a real test in an UNCHANGED file is failing, because
  the go-toolchain pack's go-test findings engine lacks the `exempt_from_scope_filter` flag its
  go-build sibling has (`--all` is confirmed unaffected). This delivery is the live instance,
  and it sits directly on the Sharp Edge 1.1.3 added: registering `doctor` as a top-level
  command breaks SPEC-067's CLM-052 whole-command-set pin in a file no task's `files:` list
  declares, and every diff-scoped gate run stayed green over that break until an unfiltered
  `go test ./...` was run. The diff-scoped PASS recorded above should be read with that caveat
  attached — the unfiltered repo-wide race run is what actually carries the verdict here.
  (3) This spec's twelve Review Questions were framed as implementation-checkable probes and
  were answered during review rather than left standing: the plan maps 1, 2, 4, 5, 6, 9, 11
  and 12 onto named tasks, the impl-review round answered 9 against a real empty directory,
  and 3, 8 and 10 were re-checked against the shipped source at close-out — `checkArtifactLayout`
  delegates per-KIND deviation to `gate.FindUngatedArtifacts` and performs no root-containment
  reduction of its own, `runDoctor` reports every setup condition through the one registry
  enumeration with no pre-registry abort or print, and `newDoctorCommand` takes the root
  persistent `--json` by pointer and declares only `--check`.
- **1.1.4** (2026-08-15) — **ACCURACY FIX, forced by the close-out: one contract entry named
  a symbol Go does not have.** No requirement, claim, mandated test, or mechanism is added,
  removed, or reworded, and NO source file changed — the only edit is to this spec's
  `contracts` block. Like SPEC-069's 1.3.3, this was found by ATTEMPTING the flip: contract
  declarations are collected only for `implemented` specs, so while this spec sat at `draft`
  the defect was structurally invisible, and the flip turned the diff-scoped
  `contract_signature` dimension RED with exactly one violation.
  **The defect.** `cmd/backstop/doctor.go` declared a single `kind: constant` provide named
  `doctorCheckIDs`, whose signature was the whole seven-constant block
  (`const ( doctorCheckConfigPresent = "config-present"; … )`). There is no `doctorCheckIDs`
  symbol anywhere in the tree and there cannot be: a Go `const (...)` block is ANONYMOUS, so
  the name was a documentation label for a group, written into a field that means "a symbol
  that exists". The contract dimension proved it mechanically. `dispatchContractEntry`
  (`cmd/backstop/gate.go`) hands the declared signature to the contracts pack's own compiler,
  `.backstop/packs/backstop-ai/go-contracts/scripts/compile-signature.sh`, whose `const`
  branch takes the token after `const ` as the name — so the group form compiles to the
  pattern `const ( = $$$`, which matches nothing, in any file, ever. Verified by running the
  compiler directly on both forms.
  **The fix, and the wrong fix that was tried first and rejected on evidence.** The obvious
  repair — split the group entry into seven single-constant entries, one per real constant,
  each in the `const NAME = value` form the compiler documents — was made, run, and FOUND
  INSUFFICIENT: the gate went from one violation to SEVEN. The compiler's emitted pattern
  `const doctorCheckConfigPresent = $$$` binds only a STANDALONE const declaration and does
  not reach a `const_spec` nested inside a parenthesized `const (…)` block. Verified directly
  with ast-grep against both shapes: the same pattern matches `pkg/gate/result.go:44`'s
  standalone `const StepRequirementTraceability = "requirement_traceability"` and matches
  nothing in `doctor.go`'s grouped block. So the seven entries are DROPPED instead. That is
  not this spec inventing an escape: it is the disposition SPEC-054 already took for
  `KindScaffolding`/`OpCreate` and SPEC-035 v1.1.2 took for `CheckTypeFindings`, under the
  rule those specs state and this one adopts — declaring an unverifiable entry buys a red, not
  a guarantee. The reason is recorded on the `doctorCheck` contract note with the standing
  forward reference, ISSUE-078, which tracks this exact gap (grouped const/var block members)
  and folds it into ISSUE-052's relational-rule `input_mode` as the general fix.
  **What is NOT weakened by dropping them.** The one-id-source invariant the constants exist
  for keeps its real, purpose-built guard: CLM-059 is a `kind: absence` source scan asserting
  that no check id appears as a literal anywhere outside those constants — a stronger property
  than a signature-existence probe, and one that never depended on the contract dimension. NO
  source file changed, no test changed, no assertion was weakened, and no waiver was taken.
  Contorting `doctor.go` into seven standalone consts to satisfy a compiler limitation was
  considered and rejected: it is a source edit with no plan in flight, and shaping code to fit
  a check's expressive gap is the inverse of what the check is for.
  **Why this was invisible until now, and where else it can hide.** The gap is not specific to
  this spec. `pkg/validate/contracts.go` validates a provides entry's SHAPE — name, kind, and
  signature present, kind in the allowed enum — and never whether the named symbol exists or
  whether its signature is compilable, which is why `artifact validate` was clean on this entry
  through four spec revisions and two plan-review rounds. Only the `implemented`-gated
  `contract_signature` dimension probes the real file. So any spec that declares a const GROUP
  under one invented label carries the same latent defect, and it will surface at its own
  close-out and never before. This is the second distinct defect class in two consecutive
  close-outs whose only trigger is the `draft` -> `implemented` flip — SPEC-069 1.3.3 was the
  first, a claim inheriting the wrong `subject:` — which is worth stating plainly: the flip is
  not bookkeeping, it is the moment two whole gate dimensions start reading the artifact for
  the first time, and a close-out that does not RE-RUN the gate after flipping has verified
  nothing about either.
- **1.1.3** (2026-08-15): DOCUMENTATION-ONLY amendment recording an edit already
  made and verified — no requirement, no claim, and no mandated test name added,
  removed, or changed; all 64 claims stand exactly as PLAN-SPEC-070 maps them, and
  no behavior, contract, or scope changed. One Sharp Edge is added: registering the
  `doctor` command in `root.go` necessarily edits SPEC-067's anti-regression pin
  `TestCIRecipes_RegisteredCommandSurfaceUnchanged` (CLM-052,
  `cmd/backstop/ci_recipes_mechanism_test.go`), which asserts exact whole-set
  equality over the CLI's top-level command set. `doctor` was added to the expected
  set with an attributing comment naming this spec, mirroring what SPEC-069 did for
  `init` (its Sharp Edge 22) and exactly what the pin's own comment prescribes. The
  assertion itself was NOT weakened — no exemption, still whole-set equality — so an
  unexplained future addition still fails. The edge also records the non-obvious
  half implementer-070 flagged: the pin lives in a file NO task in PLAN-SPEC-070's
  `files:` list declares, so an honest narrow file scope will always appear to
  exclude it. That is an inherent property of a whole-set pin — every spec adding a
  top-level command hits it — not a scoping defect in the plan. Verified at the time
  of writing: `TestCIRecipes_RegisteredCommandSurfaceUnchanged` passes and
  whole-repo `go test ./...` is clean.
- **1.1.2** (2026-08-15): Contract and prose amendment ONLY — no requirement, no
  claim, and no mandated test name added, removed, or changed; all 64 claims stand
  exactly as PLAN-SPEC-070 maps them, and no behavior or scope changed. This spec's
  text still described a PRE-EXTRACTION implementation shape: it named
  `checkEngineToolAllowed` and `splitCommand` as things `checkToolchainRuns` calls
  directly, in `cmd/backstop/doctor_checks.go`'s `consumes` block and in five prose
  places (the `checkToolchainRuns` contract note, the Implementation preamble, the
  Implementation entry for that check, Sharp Edges, References). SPEC-069/
  PLAN-SPEC-069 phase 14 extracted those three steps into the shared
  `packEntrypointProber` (`cmd/backstop/pack_entrypoint_prober.go`), which
  `backstop init` reaches through a thin adapter and which `checkToolchainRuns`
  now consumes as its second caller — a sequencing PLAN-SPEC-070 explicitly
  anticipated and ordered this amendment behind, not a change of direction. Nothing
  automated would have caught the drift: `pkg/validate/contracts.go`'s
  `validateConsumesItem` checks a consumes entry's SHAPE (source/name/kind present)
  and never whether the named symbol is actually called. So the two function entries
  are dropped and `{source: cmd/backstop, name: packEntrypointProber, kind: type}`
  takes their place; `engine.GateType` and `check.CommandRunner` STAY, since the
  rollup still formats `probe.GateType` and `doctorContext` still carries the runner
  handed to the prober. The substance of each prose passage is preserved: the three
  steps still happen in that order (allowlist gate first, then splitting, then
  execution), one layer down, and the security posture Sharp Edges asserts —
  allowlist-gate-before-split, `strings.Fields`-based tokenization, no shell — remains
  in force, now attributed to the prober rather than to inline logic in
  `doctor_checks.go`. The audit surface shrinks accordingly: independent execution
  routes go from two (doctor's inline flow plus init's) to exactly one.
- **1.1.1** (2026-08-14): Second review round, two text-only corrections; no
  requirement and no claim added or removed. (1) CLM-058's predicate contradicted
  `doctorGuidance`'s own contract: four sites mandated ONE non-test call site for
  `doctorRegistry()` while `doctorGuidance` is by design a second one, so the
  source scan would have failed on arrival. The predicate said "call site" where it
  meant "enumeration that feeds a report" — REQ-002, CLM-058 (renamed to
  `TestDoctor_RegistryHasNoCallSiteOtherThanRunDoctorAndGuidance`), the
  `doctorRegistry` and `doctorGuidance` contract notes, Implementation §1 and the
  Verification note now all permit exactly two non-test readers with named distinct
  roles — `runDoctor` ENUMERATES, `doctorGuidance` does a keyed single-id lookup
  that returns no set and feeds no report — and prohibit any third. (2)
  `doctor_checks.go`'s `consumes` listed `pkg/config.LoadConfig` and
  `loadInstalledPacks`, contradicting the spec's own "no check gathers its own
  input" invariant and inviting an implementer reading contracts literally to put a
  loader inside a check — the second abort path REQ-003 forbids. Both loaders, plus
  `pkg/check.CommandRunner`, now sit in `doctor.go`'s `consumes` (the file that
  gathers, Implementation §2); `doctor_checks.go` consumes the DATA types the
  checks read instead — `config.Config`, `pack.Manifest`, and the
  `check.CommandRunner` its `doctorContext` field carries and `checkToolchainRuns`
  invokes — with the split recorded on the `doctorContext` note so it cannot drift
  back.
- **1.1.0** (2026-08-14): Review corrections, six findings. (1) REQ-003 mandated
  four reportable conditions no registered check could produce — the registry now
  holds SEVEN checks, adding `config-present`, `config-loads`, `git-repository`
  and `packs-installed`, one condition each, with the one-condition-one-owner
  skip rule that keeps a single absent `backstop.yml` from failing four of them;
  REQ-002 and CLM-051 name the declared set rather than a count, so the
  stack-policy carve-out tripwire still holds. (2) REQ-004's guidance is now
  located: rendered in `cmd/backstop/init.go` (package `main`, where SPEC-069
  already renders init's report from `initialize.Result`) rather than in
  `pkg/initialize`, which cannot import the unexported registry — with a declared
  id-constant block and a `doctorGuidance` lookup as the single id source.
  (3) `newDoctorCommand` now takes the ROOT persistent `--json` by pointer like
  every sibling command, rather than shadowing it. (4) CLM-041's suffix
  enumeration was missing `.capability.yml` — the vocabulary is seven kinds, not
  six — and a claim now pins the bare `capability.yml` near-miss the Sharp Edges
  section already described. (5) `remediation` drops `omitempty`, so the payload
  shape does not vary with status as CLM-003 requires. (6) Three stale line
  citations corrected against HEAD: `root.go:144`→`:154` (three sites),
  `gate.go:66-72`→`:67-73`, `gate.go:78-82`→`:75-80`. The cross-spec
  toolchain-failure-classification conflict with SPEC-069 was re-verified against
  bundle REQ-011 v1.1.0 and resolved in this spec's favor — nonzero exit is
  reported verbatim, never classified as owed setup — so REQ-006 is unchanged.
- **1.0.0** (2026-08-13): Initial spec, authored against BUNDLE-003's `backstop
  doctor` seed. Covers bundle REQ-020, REQ-023, and REQ-025. Authored with three
  seams resolved against their owners rather than assumed: SPEC-068's real
  resolution signatures replace a placeholder dependency and drive REQ-007's
  per-KIND deviation rule (a root-containment rule cannot see the unconfigured
  `.backstop/` case) — SPEC-068 adopted the same reading in its 1.1.0 and renamed
  the shared helper to `FindUngatedArtifacts`, which this spec now consumes
  outright, so doctor performs no corpus walk, classification, or exclusion of its
  own; SPEC-069 confirmed it owns no init-points-at-doctor
  requirement, and its CI-guidance prohibition scoped REQ-004's diagnosable set to
  the toolchain failure. Bundle REQ-024 is
  CARVED OUT — ruled 2026-08-13, consistent with the bundle's own REQ-005 v1.1.0
  ruling that manifest surface design is BUNDLE-004's — because the pack-declared
  stack-policy surface it reads does not exist and has no owner; it carries no
  requirement and no claim here and is to be specced once an owner exists. Ranked
  sharp edge 9 (orphan reservation tags) is left untaken per the bundle's own
  record that it is a founder call.
