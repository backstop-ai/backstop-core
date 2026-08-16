---
title: "Trustworthy Green Guards"
number: SPEC-068
created: "2026-08-13"
updated: "2026-08-16"
status: implemented
schema_version: spec/v1
spec_version: 1.3.0

implementation:
  summary: >
    BUNDLE-003's FIRST seed — the trustworthy-green guards — and the prerequisite
    for the other two. It pins seven of the seed's eight bundle requirements at
    their current editions: REQ-026@1.0.0 (content-derived schema cohort),
    REQ-034@1.0.0 (content-derived schema identity on the validation record),
    REQ-027@1.0.0 (cohort assertion with refuse-to-green), REQ-028@1.0.0
    (producing-binary provenance on every result), REQ-021@1.1.0 (build identity
    on `version`, `dev` PRESERVED), REQ-029@1.0.0 (one config-resolved artifact
    root and one artifact-layout table, applied uniformly), and REQ-030@1.1.0
    (name the scanned root, fail loudly on a configured root that is absent,
    surface per kind the artifact-shaped files the status walk cannot reach). The eighth, REQ-022, is
    DELIBERATELY NOT PINNED — it is blocked on BUNDLE-020 and the reasoning is
    recorded in Dependencies rather than resolved here.
    Three defect families are closed. (1) COHORT SKEW: `computeCohortID`
    (`cmd/backstop/root.go:206`) builds its identifier from schema PATHS —
    verified at HEAD, it emits `11-schemas[adr/v1,…,spec/v1]` — so BUNDLE-014's
    in-place revision of `bundle/v2` left the reported cohort byte-identical, and
    `artifacts/bundle/v2/schema.json` pins `"schema_version": {"const":
    "bundle/v2"}`, revision-FREE, so the artifact side was byte-identical too.
    Neither side could name the change; both sides become content-derived here.
    (2) NO PRODUCING-BINARY IDENTITY: `GateResult` stamps `GitSHA`/`GeneratedAt`
    (`pkg/gate/result.go:144-145`) and the validate envelope
    (`cmd/backstop/output.go:47-52`) stamps neither, so a stored green is
    indistinguishable from a green a stale validator could not have withheld.
    (3) SILENT UNDISCOVERY: the artifact root is hardcoded in three places the
    bundle names — `pkg/gate/artifact_status.go:171,193,211,226,241` joins
    projectRoot with the five type directories, `pkg/scaffold/scaffold.go:35-75`
    scaffolds into the same root names, `pkg/validate/resolved_by.go:46-52`
    carries its own type→directory map — and in a FOURTH the bundle does not:
    `DiscoverArtifacts` (`cmd/backstop/artifact_discover.go:48`) SkipDirs
    `.backstop` outright, so a `.backstop/`-rooted consumer discovers zero
    artifacts and `ValidateArtifacts` returns `Pass: true` on the empty set
    (`artifact_validate.go:129`). No `artifact_root` key exists in
    `artifacts/backstop-yml/v1/schema.json` or `pkg/config`, and that schema is
    `additionalProperties: false`, so the key must be added before any consumer
    can declare one.
    What lands: one layout authority and one root resolver in `pkg/artifact`
    (zero-cycle — the package imports only stdlib and yaml today), consumed by
    gate status resolution, CLI discovery, typed-ref resolution and scaffolding;
    a content-derived cohort in `pkg/schema` consumed by `version`, `validate`
    and `gate`; build identity in `cmd/backstop/version.go` that keeps the
    anti-spoofing `dev` string intact; and, on the artifact_validation step, the
    per-kind ungated-artifact surfacing that actually closes the
    backstop-runtime case.
  subject: pkg/artifact

verification:
  level: integration
  test_command: go test ./pkg/artifact/... ./pkg/schema/... ./pkg/config/... ./pkg/validate/... ./pkg/gate/... ./pkg/scaffold/... ./cmd/backstop/... -race -coverprofile=cover.out
  coverage_threshold: 80

requirements:
  - id: REQ-001
    text: >
      The schema cohort a binary reports must be derived from the CONTENT of its
      embedded schemas. A single cohort computation must produce (a) one cohort
      IDENTIFIER over the whole embedded schema set and (b) a per-schema DIGEST
      keyed by the `<type>/v<N>` identity artifacts declare in `schema_version`.
      Changing any embedded schema's bytes must change both that schema's digest
      and the cohort identifier, even when the set of schema PATHS is unchanged —
      the exact condition BUNDLE-014's in-place `bundle/v2` revision met and the
      current path-derived `computeCohortID` cannot detect. Adding or removing a
      schema must also change the identifier. The computation must be
      deterministic for identical content and independent of filesystem walk
      order, and it must be reachable by `version`, `artifact validate` and
      `gate` from one implementation rather than three. A `schema_version` the
      embedded set does not contain must resolve to NO digest and be reported as
      uncovered — never to a zero value, an empty string, or a default.
    supports:
      - onboarding-experience:REQ-026@1.0.0
    follows:
      - STD-GO-001:GO-010
      - STD-GO-001:GO-011
  - id: REQ-002
    text: >
      `backstop artifact validate` and `backstop gate` must assert the validating
      binary's cohort against each discovered artifact's declared
      `schema_version` and must REFUSE to report a passing result when they
      cannot prove the cohort covers it. A declared `schema_version` with no
      schema in the cohort is a refusal, and its diagnostic must name the
      artifact path, the declared `schema_version`, and the binary's cohort
      identifier — "cannot determine" is a failure, never a pass. An artifact
      type that is routed WITHOUT a schema (plans, which route by discovery type
      and carry no `schema_version`) must be recorded as schema-less rather than
      silently counted as covered. Discovering ZERO artifacts under an artifact
      root that EXISTS is NOT a refusal — it stays a pass, preserving the
      validated greenfield outcome REQ-008 also protects — but the result must
      report the number of artifacts asserted and the root they were sought in,
      so an empty pass is legible as empty rather than as verified.
    supports:
      - onboarding-experience:REQ-027@1.0.0
    follows:
      - STD-GO-001:GO-010
      - STD-GO-001:GO-011
  - id: REQ-003
    text: >
      An artifact's schema identity, as asserted at validation time, must be
      CONTENT-DERIVED rather than the bare revision-free `schema_version` const,
      and this spec discharges the bundle's either/or by choosing the VALIDATION
      RECORD side: the record produced when an artifact is checked carries
      `<schema_version>@<digest>`, and the artifact files themselves are not
      mutated. The digest must cover every schema file actually used to validate
      the artifact, INCLUDING the base schema resolved through `extends`, so a
      base-schema revision changes the recorded identity even when the extension
      schema is untouched. Both `artifact validate` and `gate` must carry these
      identities in their human and `--json` renderings, from one resolved value
      shared by both so the two cannot drift.
    supports:
      - onboarding-experience:REQ-034@1.0.0
    follows:
      - STD-GO-001:GO-011
  - id: REQ-004
    text: >
      Every `gate` and `artifact validate` result, in both human and `--json`
      form, must record the version and the schema cohort identifier of the
      binary that produced it, so a stored or forwarded result can be quarantined
      rather than projected as truth. On the gate side these are additive fields
      on the existing result under the unchanged `gate/v1` schema, alongside the
      `GitSHA`/`GeneratedAt` provenance already present; on the validate side
      they are additive fields on the existing output envelope. The value
      reported must be the SAME version string `backstop version` reports and the
      SAME cohort identifier REQ-001 computes — resolved once per run and shared
      by both renderings.
    supports:
      - onboarding-experience:REQ-028@1.0.0
    follows:
      - STD-GO-001:GO-010
  - id: REQ-005
    text: >
      `backstop version` must stamp the build commit and the build date so a
      stale binary is identifiable on sight, in both human and `--json` form. The
      existing anti-spoofing VERSION STRING behavior is PRESERVED, not
      eliminated: a non-release build must still report bare `dev`, and
      `resolveVersion`'s rejection of `(devel)`, of any `+` build metadata, and of
      Go pseudo-versions is unchanged. Build identity is derived from the build
      information the toolchain records (VCS revision, VCS time, VCS modified),
      with link-time injected values taking precedence when non-empty; a modified
      working tree must be marked on the COMMIT, never by making the version
      string look released. When neither source supplies a commit — a binary
      installed from the module cache, which has no VCS data — the field must
      report `unknown` explicitly rather than an empty or omitted value.
    supports:
      - onboarding-experience:REQ-021@1.1.0
    follows:
      - STD-GO-001:GO-003
      - STD-GO-001:GO-010
  - id: REQ-006
    text: >
      There must be ONE artifact-root resolution and ONE artifact-layout table,
      both living in a package every consumer can import. The layout table is the
      single authority mapping each artifact kind to its directory name and file
      extension, and to the filename pattern that classifies a file as that kind;
      it must cover every kind the codebase already recognizes — spec, plan, adr,
      bundle, issue, directive, capability — and classify any filename to at most
      one kind. The root resolver takes the project root and the value declared
      in `backstop.yml` and returns the resolved root plus whether it was
      configured. `backstop.yml` gains a top-level `artifact_root` key, in both
      the JSON schema (which is `additionalProperties: false`, so an unlisted key
      is rejected) and the typed loader; an ABSENT key resolves to the project
      root marked unconfigured, which is how backstop-core keeps its own
      repo-root layout as the OQ-1 framework exception without configuring
      anything. THE RESOLVED ROOT PATH IS ALWAYS ABSOLUTE AND CLEANED: the
      resolver absolutizes a relative project root rather than passing it
      through, so every directory the layout names is absolute for every caller
      and no consumer has to know which form it was handed. This is a contract
      guarantee, not a caller convention, because the two entry points disagree
      today — gate roots at the `backstop.yml` directory and falls back to `"."`
      when that discovery fails, while validate roots at the working directory —
      and REQ-008's per-kind rule is a directory-string comparison that degrades
      silently, never loudly, when one side is relative and the other absolute.
      The resolver must reject a declared value that is absolute or
      that escapes the project root, and must return a TYPED error — never a
      silent fallback to the project root — when a configured root is absent from
      disk or is not a directory. A configured root that exists and is empty
      resolves normally.
    supports:
      - onboarding-experience:REQ-029@1.0.0
    follows:
      - STD-GO-001:GO-010
      - STD-GO-001:GO-011
      - STD-GO-001:GO-021
  - id: REQ-007
    text: >
      Every consumer of the artifact layout must read that one resolution, so a
      consuming repo can hold the whole artifact chain under `.backstop/` and be
      discovered. ~~Four consumers adopt it~~ FIVE consumers adopt it (see the
      2026-08-14 correction below for the fifth): gate status resolution
      (`pkg/gate/artifact_status.go`, which today joins the project root with
      five literal directory names), CLI artifact discovery
      (`cmd/backstop/artifact_discover.go`, which today walks the project root
      and SkipDirs `.backstop` outright — the reason a `.backstop/`-rooted repo
      validates zero artifacts and reports a pass), validation typed-reference
      resolution (`pkg/validate/resolved_by.go`, which anchors on the citing
      artifact's own source path and therefore needs the shared type→directory
      table, NOT a root change), and scaffolding (`pkg/scaffold`, which places
      new artifacts). Discovery's exclusion of installed pack trees under
      `.backstop/packs/` must SURVIVE the change: when the artifact root is
      `.backstop`, that root is walked but installed packs within it are still
      excluded, so a pack's own artifact-shaped files are never adopted as the
      consumer's corpus. backstop-core's unconfigured repo-root layout must be
      discovered exactly as it is today.
      CORRECTION (2026-08-14, v1.2.1) — A FIFTH CONSUMER, FOUND DURING PLANNING.
      The v1.2.0 clause enumerated exactly four consumers and MISSED a real
      hardcoding that the other four fixes do not reach: `buildGateSteps`
      (`cmd/backstop/gate.go:652`) computes `specDir := filepath.Join(projectRoot,
      "specs")` and feeds it to FOUR gate dimensions — `test_verification`
      (`gate.go:706`), `test_substantiveness` (`gate.go:710`), `coverage`
      (`gate.go:718`) and `contracts` (`gate.go:721`). `specDir` MUST be the
      resolved artifact root's spec directory — `Root.Dir(artifact.KindSpec)` —
      exactly as the other four consumers read theirs, and every call site of
      `buildGateSteps` (`gate.go:124`, `baseline.go:64`, `waiver.go:88`) must pass
      the resolved root rather than leaving one of them on the project root.
      WHY THIS IS A DEFECT AND NOT A REFINEMENT: `gate.ExtractMandatedTests`
      (`pkg/gate/step_testverify.go:113-116`) does an `os.ReadDir` on `specDir` and
      returns a wrapped error on a missing directory — it does NOT tolerate absence
      the way the status walkers do. So once this spec's other fixes land, a
      `.backstop/`-rooted consumer has `artifact_validation`, `status_drift` and
      `requirement_traceability` correctly rooted while these four dimensions still
      read a literal `<project>/specs` that does not exist, and the gate HARD-BREAKS
      on them. That consumer is not hypothetical: SPEC-069's full-SDLC `backstop
      init` profile writes `artifact_root: .backstop`, creates `.backstop/specs/`,
      and DELIBERATELY leaves exactly these five dimensions enabled (it disables
      them only for the narrower pack-only profile), so without this fix SPEC-068 and
      SPEC-069 together ship a flagship-profile project whose gate breaks on four
      dimensions the first time it runs. Absent-directory tolerance is NOT changed
      here — the loud failure for a configured-but-absent root stays at the ROOT
      (REQ-008), an EXISTING but empty spec directory still reads cleanly because
      `os.ReadDir` of an empty directory succeeds, and SPEC-069's profile creates the
      directory it configures.
    supports:
      - onboarding-experience:REQ-029@1.0.0
    follows:
      - STD-GO-001:GO-010
      - STD-GO-001:GO-011
  - id: REQ-008
    text: >
      Gate output must NAME the artifact root it actually scanned and whether
      that root was configured, in both human and `--json` form, and the
      configured/unconfigured fact must be present in the `--json` rendering in
      BOTH states — an unconfigured root is the default and the motivating shape,
      so a serialization that omits the field when it is false does not satisfy
      this. A CONFIGURED artifact root that does not exist on disk must fail the
      run loudly rather than walking absent directories and reporting a passing
      dimension. An artifact root that EXISTS but is empty remains a legitimate
      pass — SCOPED (2026-08-14, v1.2.2) to root resolution and the
      `artifact_validation` dimension: an empty root must not itself be an error,
      and the zero-artifact corpus must validate clean. It does NOT mandate an
      aggregate `Pass: true` for a run over an empty root, since dimensions
      unrelated to root resolution — notably the four REQ-007 spec-directory
      consumers — may legitimately fail over the same tree. MECHANISM CORRECTED
      (2026-08-14, v1.2.3): the v1.2.2 wording attributed that to those consumers
      erroring on a spec directory that does not exist. Under the shape real
      `backstop init` produces, the configured root's spec directory EXISTS and
      is empty, so they read it cleanly; what may still fail over that tree is
      whatever those dimensions independently require of a zero-artifact corpus.
      The SCOPING is unchanged by this correction.
      Artifact-shaped files the gate's ARTIFACT-STATUS WALK would not reach must
      be SURFACED as ungated, each reported with its path, the directory it was
      expected in, and the resolved root. The status walk is the calibration
      point and it is NON-RECURSIVE — it reads a type directory with a single
      directory read — so the surfaced set is exactly the artifact-shaped files
      that are not DIRECTLY in `Root.Dir(kind)` for the kind their filename
      classifies to. Ungated and undiscovered are two different facts and the
      report must not conflate them: CLI discovery walks the resolved root
      RECURSIVELY, so a file nested one level below its own type directory is
      discovered and schema-validated while still being ungated by the status
      walk, whereas a file outside the resolved root or under an excluded tree is
      both undiscovered and ungated. Every file in the surfaced set is ungated;
      the report must describe it as such, and may only call a file undiscovered
      when discovery genuinely does not reach it.
      THE TEST IS PER KIND, NOT ROOT CONTAINMENT: a file deviates when it is not
      directly in `Root.Dir(kind)`. Root containment alone is insufficient and
      would make this clause vacuous on its own motivating case — backstop-runtime
      places `.backstop/bundles/` in a repo that configures no artifact root, so
      those bundles sit INSIDE the default root while `bundles/` is where
      discovery looks, and a containment test reports nothing. The per-kind test
      catches that case, catches a file under a configured root that is in the
      wrong type directory, and catches one outside a configured root, with one
      rule. Because that rule is a directory comparison, both sides of it must be
      normalized to the absolute, cleaned form REQ-006 guarantees before they are
      compared; a comparison between a relative and an absolute form silently
      yields either no findings at all or a finding for every artifact, and both
      degenerate outcomes look like a working implementation.
      Surfaced files must NOT be adopted: discovery does not widen its root or its
      type directories, it tells the truth about what it left out. Those findings
      are corpus-integrity facts, not per-file findings, so they must survive a
      diff-scoped run rather than being filtered away with the files they name.
      The scan must exclude a DETERMINED set of non-corpus trees — the
      tool-agnostic base (`.git`, `testdata`, `prototype`) plus whatever
      installed packs declare via `classification.dependency_dirs`, and installed
      pack trees under `.backstop/packs` — so fixtures and installed packs cannot
      manufacture findings. Ecosystem-specific dependency directory names are NOT
      core's to name (ISSUE-122): they arrive as pack-declared data and are
      unioned onto the tool-agnostic base, so the exclusion set is determined by
      what is installed rather than by a literal in core. That set is this
      requirement's own and must
      NOT be expressed as "whatever discovery skips": discovery skips `.backstop`
      wholesale except when `.backstop` IS the resolved root, and inheriting that
      skip by reference would make this clause report nothing in the UNCONFIGURED
      case, which is precisely its motivating case. `.backstop` itself is
      therefore WALKED by this scan whether or not it is the root; only
      `.backstop/packs` is excluded beneath it. A subdirectory inside a type
      directory is not itself a deviation — only the artifact-shaped FILES'
      locations are judged.
    supports:
      - onboarding-experience:REQ-030@1.1.0
    follows:
      - STD-GO-001:GO-010
      - STD-GO-001:GO-011

claims:
  # REQ-001 — the cohort becomes a function of schema BYTES.
  - id: CLM-001
    requirement: REQ-001
    subject: pkg/schema
    text: The cohort identifier changes when an embedded schema's bytes change while the set of schema paths is unchanged — the in-place bundle/v2 revision the current path-derived id cannot see
    tests:
      - TestComputeCohort_IDChangesOnInPlaceSchemaRevision
  - id: CLM-002
    requirement: REQ-001
    subject: pkg/schema
    text: The cohort identifier is deterministic for identical content and independent of walk order, so two runs over the same embedded set agree
    tests:
      - TestComputeCohort_DeterministicAndOrderIndependent
  - id: CLM-003
    requirement: REQ-001
    subject: pkg/schema
    text: Adding a schema and removing a schema each change the cohort identifier
    tests:
      - TestComputeCohort_IDChangesOnSchemaSetAddition
      - TestComputeCohort_IDChangesOnSchemaSetRemoval
  - id: CLM-004
    requirement: REQ-001
    subject: pkg/schema
    text: The cohort exposes a per-schema digest for every embedded schema, keyed by the type/vN identity artifacts declare in schema_version
    tests:
      - TestComputeCohort_PerSchemaDigestForEveryEmbeddedSchema
  - id: CLM-005
    requirement: REQ-001
    subject: pkg/schema
    text: A schema_version the embedded set does not contain resolves to no digest and reports uncovered, never a zero value or default
    tests:
      - TestCohort_UnknownSchemaVersionReportsUncoveredNotZeroValue
  - id: CLM-006
    requirement: REQ-001
    subject: cmd/backstop
    text: backstop version reports the content-derived cohort identifier in human and --json output, replacing the path-derived schema-name string
    tests:
      - TestVersionCommand_ReportsContentDerivedCohort

  # REQ-002 — assertion, and the refusal boundary in both directions.
  - id: CLM-007
    requirement: REQ-002
    subject: cmd/backstop
    text: An artifact declaring a schema_version absent from the binary's cohort refuses to report green, naming the artifact path, the declared schema_version and the cohort identifier
    tests:
      - TestValidateArtifacts_UncoveredSchemaVersionRefusesGreen
  - id: CLM-008
    requirement: REQ-002
    subject: cmd/backstop
    text: That refusal exits non-zero rather than reporting a pass with zero violations
    tests:
      - TestValidateArtifacts_UncoveredSchemaVersionExitsNonZero
  - id: CLM-009
    requirement: REQ-002
    subject: cmd/backstop
    text: A plan, routed without a schema, is recorded as schema-less rather than counted as cohort-covered
    tests:
      - TestValidateArtifacts_PlanRecordedAsSchemaless
  - id: CLM-010
    requirement: REQ-002
    subject: cmd/backstop
    text: Zero artifacts discovered under an artifact root that exists remains a PASS, and the result reports the asserted count and the root sought in
    tests:
      - TestValidateArtifacts_ZeroArtifactsUnderExistingRootStillPasses
  - id: CLM-011
    requirement: REQ-002
    subject: cmd/backstop
    text: The gate inherits the same refusal through its artifact_validation step rather than passing where the CLI would refuse
    tests:
      - TestGate_ArtifactValidation_UncoveredSchemaVersionRefusesGreen
  - id: CLM-012
    requirement: REQ-002
    subject: cmd/backstop
    text: Every artifact whose schema_version IS covered validates exactly as it does today, so the assertion adds a guard without changing accepted corpora
    tests:
      - TestValidateArtifacts_CoveredSchemaVersionsValidateUnchanged

  # REQ-003 — the record-side schema identity (the bundle's either/or, discharged).
  - id: CLM-013
    requirement: REQ-003
    subject: cmd/backstop
    text: validate --json records a schema_version@digest identity for each validated artifact
    tests:
      - TestValidateOutput_JSONRecordsSchemaIdentityPerArtifact
  - id: CLM-014
    requirement: REQ-003
    subject: cmd/backstop
    text: validate human output records the same identities the JSON rendering carries, from one resolved value
    tests:
      - TestValidateOutput_HumanRecordsSameSchemaIdentityAsJSON
  - id: CLM-015
    requirement: REQ-003
    subject: cmd/backstop
    text: The recorded identity changes across an in-place schema revision whose schema_version const is byte-identical before and after
    tests:
      - TestValidateOutput_SchemaIdentityChangesOnInPlaceRevision
  - id: CLM-016
    requirement: REQ-003
    subject: cmd/backstop
    text: gate --json carries the same per-schema identities its artifact validation asserted
    tests:
      - TestGateOutput_JSONRecordsSchemaIdentities
  - id: CLM-017
    requirement: REQ-003
    subject: pkg/schema
    text: The identity digest covers the base schema resolved through extends, so revising only the base schema changes it
    tests:
      - TestCohort_SchemaIdentityCoversResolvedBaseSchema
  - id: CLM-018
    requirement: REQ-003
    subject: cmd/backstop
    text: No artifact file is written or mutated by validation, which is what makes the record rather than the artifact the identity carrier
    tests:
      - TestValidateArtifacts_LeavesArtifactFilesUnmodified

  # REQ-004 — producing-binary provenance on every result surface.
  - id: CLM-019
    requirement: REQ-004
    subject: pkg/gate
    text: GateResult carries the producing binary's version and cohort identifier as additive fields under the unchanged gate/v1 schema
    tests:
      - TestGateResult_CarriesProducingBinaryIdentity
  - id: CLM-020
    requirement: REQ-004
    subject: cmd/backstop
    text: gate --json output carries the producing binary's version and cohort identifier
    tests:
      - TestGate_JSONOutputCarriesBinaryIdentity
  - id: CLM-021
    requirement: REQ-004
    subject: cmd/backstop
    text: gate human output prints the producing binary's version and cohort identifier
    tests:
      - TestGate_HumanOutputPrintsBinaryIdentity
  - id: CLM-022
    requirement: REQ-004
    subject: cmd/backstop
    text: The validate output envelope carries the producing binary's version and cohort identifier in --json form
    tests:
      - TestValidateOutput_JSONCarriesBinaryIdentity
  - id: CLM-023
    requirement: REQ-004
    subject: cmd/backstop
    text: The validate human rendering prints the same binary identity the JSON rendering carries
    tests:
      - TestValidateOutput_HumanCarriesSameBinaryIdentityAsJSON
  - id: CLM-024
    requirement: REQ-004
    subject: cmd/backstop
    text: The version reported on a result equals the string backstop version reports for the same binary
    tests:
      - TestResultBinaryIdentity_MatchesVersionCommandOutput

  # REQ-005 — build identity, with the anti-spoofing dev string preserved.
  - id: CLM-025
    requirement: REQ-005
    subject: cmd/backstop
    text: version reports a build commit and a build date derived from recorded build information for a locally built binary
    tests:
      - TestVersionCommand_ReportsBuildCommitAndBuildDate
  - id: CLM-026
    requirement: REQ-005
    subject: cmd/backstop
    text: The version STRING for a non-release build is still bare dev, and the existing rejections of (devel), + build metadata and pseudo-versions are unchanged
    tests:
      - TestResolveVersion_NonReleaseBuildStillReportsDev
  - id: CLM-027
    requirement: REQ-005
    subject: cmd/backstop
    text: A modified working tree is marked on the commit field and never by making the version string look released
    tests:
      - TestBuildIdentity_DirtyTreeMarkedOnCommitNotVersion
  - id: CLM-028
    requirement: REQ-005
    subject: cmd/backstop
    text: A non-empty link-time injected commit or date takes precedence over the recorded build information
    tests:
      - TestBuildIdentity_InjectedStampTakesPrecedence
  - id: CLM-029
    requirement: REQ-005
    subject: cmd/backstop
    text: With neither an injected stamp nor VCS build information the commit reports unknown explicitly rather than empty or omitted
    tests:
      - TestBuildIdentity_UnknownWhenNoInjectedStampAndNoVCSData
  - id: CLM-030
    requirement: REQ-005
    subject: cmd/backstop
    text: version --json carries the commit and build date alongside the version and cohort fields
    tests:
      - TestVersionCommand_JSONCarriesBuildIdentity

  # REQ-006 — the one resolver and the one layout table; every root state enumerated.
  - id: CLM-031
    requirement: REQ-006
    text: An absent artifact_root resolves to the project root marked unconfigured, which is how the framework exception keeps the repo-root layout
    tests:
      - TestResolveRoot_AbsentDeclarationResolvesToProjectRootUnconfigured
  - id: CLM-032
    requirement: REQ-006
    text: A configured root naming an existing directory resolves to that directory marked configured
    tests:
      - TestResolveRoot_ConfiguredExistingDirectoryResolvesConfigured
  - id: CLM-033
    requirement: REQ-006
    text: A configured root that is absent from disk returns a typed missing-root error and never falls back to the project root
    tests:
      - TestResolveRoot_ConfiguredAbsentRootReturnsTypedErrorNotFallback
  - id: CLM-034
    requirement: REQ-006
    text: A configured root that exists and is empty resolves normally, so an empty artifact tree is representable
    tests:
      - TestResolveRoot_ConfiguredEmptyDirectoryResolves
  - id: CLM-035
    requirement: REQ-006
    text: A configured root pointing at a file rather than a directory returns a typed error
    tests:
      - TestResolveRoot_ConfiguredRootThatIsAFileReturnsTypedError
  - id: CLM-036
    requirement: REQ-006
    text: An absolute declared root is rejected
    tests:
      - TestResolveRoot_AbsoluteDeclaredRootRejected
  - id: CLM-037
    requirement: REQ-006
    text: A declared root that escapes the project root is rejected
    tests:
      - TestResolveRoot_EscapingDeclaredRootRejected
  - id: CLM-038
    requirement: REQ-006
    text: Each of the seven artifact kinds — spec, plan, adr, bundle, issue, directive, capability — resolves to a directory and an extension from the one layout table, with no kind missing
    tests:
      - TestLayout_EverySevenKindsResolveDirectoryAndExtension
  - id: CLM-039
    requirement: REQ-006
    text: Filename classification assigns at most one kind per filename and no kind to a non-artifact filename
    tests:
      - TestLayout_FilenameClassificationIsExclusiveAndRejectsNonArtifacts
  - id: CLM-040
    requirement: REQ-006
    subject: pkg/config
    text: A backstop.yml declaring artifact_root parses through the strict typed loader
    tests:
      - TestLoadConfig_ArtifactRootParses
  - id: CLM-041
    requirement: REQ-006
    subject: pkg/config
    text: A backstop.yml declaring artifact_root passes the JSON-schema pass whose additionalProperties is false
    tests:
      - TestLoadConfig_ArtifactRootAcceptedByJSONSchema
  - id: CLM-042
    requirement: REQ-006
    subject: pkg/config
    text: A backstop.yml with no artifact_root leaves the field empty, so no default root is baked into the loader
    tests:
      - TestLoadConfig_AbsentArtifactRootLeavesFieldEmpty

  # REQ-007 — every one of the consumers, plus both layout profiles end to end.
  # The fifth consumer (gate.go's specDir) is CLM-069, grouped with the 1.2.1 addition below.
  - id: CLM-043
    requirement: REQ-007
    subject: pkg/gate
    text: Gate status resolution walks the resolved artifact root, so artifacts under a .backstop/-rooted layout are resolved rather than missed
    tests:
      - TestResolveArtifactStatus_WalksResolvedArtifactRoot
  - id: CLM-044
    requirement: REQ-007
    subject: pkg/gate
    text: Gate status resolution takes its type directory names from the shared layout table rather than its own literal joins
    tests:
      - TestResolveArtifactStatus_TakesTypeDirectoriesFromSharedLayout
  - id: CLM-045
    requirement: REQ-007
    subject: cmd/backstop
    text: CLI artifact discovery walks the resolved root, including .backstop when that is the configured root
    tests:
      - TestDiscoverArtifacts_WalksResolvedRootIncludingDotBackstop
  - id: CLM-046
    requirement: REQ-007
    subject: cmd/backstop
    text: Discovery under a .backstop root still excludes installed pack trees, so a pack's own artifact-shaped files are never adopted as the consumer's corpus
    tests:
      - TestDiscoverArtifacts_DotBackstopRootStillExcludesInstalledPacks
  - id: CLM-047
    requirement: REQ-007
    subject: pkg/validate
    text: Typed-reference resolution takes its directory name and file extension from the shared layout table for all five typed-ref prefixes, carrying no private directory or extension values of its own
    tests:
      - TestResolvedBy_TakesTypeDirectoriesFromSharedLayout
  - id: CLM-048
    requirement: REQ-007
    subject: pkg/validate
    text: A typed reference still resolves for an artifact living under a non-root layout, preserving the source-path-anchored behavior the shared table must not break
    tests:
      - TestResolvedBy_ResolvesUnderNonRootLayout
  - id: CLM-049
    requirement: REQ-007
    subject: pkg/scaffold
    text: Scaffolding places a new artifact of each kind under the resolved artifact root
    tests:
      - TestTargetDir_PlacesEveryKindUnderResolvedRoot
  - id: CLM-050
    requirement: REQ-007
    subject: cmd/backstop
    text: artifact new writes its scaffolded file into the resolved root rather than the project root
    tests:
      - TestArtifactNew_WritesIntoResolvedRoot
  - id: CLM-051
    requirement: REQ-007
    subject: cmd/backstop
    text: An unconfigured repo-root layout — backstop-core's own framework exception — discovers the same corpus it does today
    tests:
      - TestE2E_UnconfiguredRepoRootLayoutDiscoversUnchangedCorpus
  - id: CLM-052
    requirement: REQ-007
    subject: cmd/backstop
    text: A .backstop/-rooted project discovers the same corpus a byte-identical repo-root project does, which is the outcome REQ-004 of the init seed depends on
    tests:
      - TestE2E_DotBackstopRootedProjectDiscoversEquivalentCorpus

  # REQ-008 — naming, loud absence, and the surfacing clause that closes the runtime case.
  - id: CLM-053
    requirement: REQ-008
    subject: pkg/gate
    text: The gate result names the artifact root actually scanned and whether it was configured
    tests:
      - TestGateResult_NamesScannedArtifactRootAndConfiguredFlag
  - id: CLM-054
    requirement: REQ-008
    subject: cmd/backstop
    text: Both gate renderings print the scanned artifact root
    tests:
      - TestGate_HumanAndJSONOutputNameScannedArtifactRoot
  - id: CLM-055
    requirement: REQ-008
    subject: cmd/backstop
    text: A configured artifact root that is absent from disk fails the run loudly instead of walking nothing and reporting a passing dimension
    tests:
      - TestGate_ConfiguredAbsentArtifactRootFailsLoudly
  - id: CLM-056
    requirement: REQ-008
    subject: cmd/backstop
    text: >
      A configured artifact root that exists and is empty does not fail ROOT
      RESOLUTION OR THE artifact_validation DIMENSION — ResolveRoot returns
      normally and artifact_validation reports clean over the zero-artifact
      corpus rather than erroring on the empty root — preserving the validated
      greenfield outcome. NARROWED (2026-08-14, v1.2.2): this claim does NOT
      assert that the whole gate run returns Pass: true. Other dimensions may
      legitimately fail over the same fixture for reasons independent of root
      resolution — with REQ-007's fifth consumer (CLM-069) the test_verification,
      test_substantiveness, coverage and contracts dimensions read the resolved
      root's SPEC DIRECTORY. MECHANISM CORRECTED (2026-08-14, v1.2.3): the
      v1.2.2 wording illustrated that with ExtractMandatedTests erroring on a
      spec directory that does not exist, which no longer describes the fixture
      this claim's mandated test runs against — that fixture's spec directory
      EXISTS and is empty, the shape real `backstop init` produces. Those four
      dimensions therefore READ the spec directory cleanly here, and may still
      report failure for reasons of their own over a zero-artifact corpus,
      independent of root resolution. The NARROWING is unchanged by this
      correction. The mandated test must therefore assert the
      root-resolution/artifact_validation outcome specifically, never the
      aggregate verdict.
    tests:
      - TestGate_ConfiguredEmptyArtifactRootPasses
  - id: CLM-057
    requirement: REQ-008
    subject: pkg/gate
    text: An artifact-shaped file not directly in Root.Dir(kind) is surfaced, naming its path, the directory it was expected in, and the resolved root
    tests:
      - TestFindUngatedArtifacts_SurfacesFileOutsideItsExpectedTypeDirectory
  - id: CLM-058
    requirement: REQ-008
    subject: pkg/gate
    text: The backstop-runtime shape — .backstop/bundles/ in a repo configuring no artifact root, so the files sit INSIDE the default root — is surfaced, which a root-containment test cannot do
    tests:
      - TestFindUngatedArtifacts_SurfacesInsideDefaultRootButWrongTypeDirectory
  - id: CLM-059
    requirement: REQ-008
    subject: pkg/gate
    text: A file outside a CONFIGURED root is surfaced by the same per-kind rule, so one rule covers both the configured and unconfigured shapes
    tests:
      - TestFindUngatedArtifacts_SurfacesFileOutsideConfiguredRoot
  - id: CLM-060
    requirement: REQ-008
    subject: pkg/gate
    text: A corpus whose artifact-shaped files all sit directly in their expected type directories produces no findings, which is why backstop-core's unconfigured repo-root layout passes without a framework-exception carve-out
    tests:
      - TestFindUngatedArtifacts_CorrectlyPlacedCorpusProducesNoFindings
  - id: CLM-061
    requirement: REQ-008
    subject: pkg/gate
    text: Surfacing reports without adopting — the scanned root and type directories are unchanged and the surfaced files are not validated as corpus
    tests:
      - TestFindUngatedArtifacts_ReportsWithoutWideningRootOrTypeDirectories
  - id: CLM-062
    requirement: REQ-008
    subject: pkg/gate
    text: The scan excludes exactly the determined non-corpus trees — the tool-agnostic base (.git, testdata, prototype) plus whatever installed packs declare via classification.dependency_dirs, and .backstop/packs — while still walking .backstop itself, so fixtures and installed packs cannot manufacture findings and the unconfigured-root case is not excluded away
    tests:
      - TestFindUngatedArtifacts_ExcludesEnumeratedNonCorpusTreesButWalksDotBackstop
  - id: CLM-063
    requirement: REQ-008
    subject: pkg/gate
    text: A subdirectory inside a type directory is not itself a deviation, and an artifact-shaped file nested inside one IS surfaced as ungated — the non-recursive status walk does not reach it — while remaining discovered and schema-validated by the recursive CLI discovery, so the finding does not call it undiscovered
    tests:
      - TestFindUngatedArtifacts_NestedArtifactIsUngatedButStillDiscovered
  - id: CLM-064
    requirement: REQ-008
    subject: pkg/gate
    text: Ungated findings survive diff-scoped filtering, because a corpus-integrity fact must not be filtered away with the files it names
    tests:
      - TestUngatedArtifactFindings_SurviveDiffScopedFiltering
  - id: CLM-065
    requirement: REQ-008
    subject: cmd/backstop
    text: The backstop-runtime case is surfaced in gate output end to end — bundles under .backstop/bundles in a repo configuring no artifact root, which discovery does not reach and the status walk does not gate, so the finding names both facts
    tests:
      - TestGate_UngatedBundlesUnderDotBackstopSurfacedInOutput

  # REQ-006 and REQ-008 — the path-normalization decision the per-kind comparison rides on,
  # and the false-state serialization the configured flag rides on. Grouped last because
  # they were added in 1.2.0; claim ids are identifiers, not ordinals.
  - id: CLM-066
    requirement: REQ-006
    text: ResolveRoot returns an absolute cleaned Path when handed a relative project root, so Root.Dir never yields a relative directory to a consumer
    tests:
      - TestResolveRoot_RelativeProjectRootResolvesToAbsolutePath
  - id: CLM-067
    requirement: REQ-008
    subject: pkg/gate
    text: FindUngatedArtifacts returns identical findings whether it is handed the relative or the absolute form of the same project root, so the per-kind directory comparison cannot silently degenerate to zero findings or to one per artifact
    tests:
      - TestFindUngatedArtifacts_RelativeAndAbsoluteProjectRootAgree
  - id: CLM-068
    requirement: REQ-008
    subject: cmd/backstop
    text: gate --json carries the artifact-root-configured fact as an explicit false for an UNCONFIGURED root rather than omitting the field, which is the default and the motivating shape
    tests:
      - TestGate_JSONCarriesArtifactRootConfiguredFalseWhenUnconfigured

  # REQ-007 — the fifth consumer, added in 1.2.1 (see REQ-007's 2026-08-14 correction).
  - id: CLM-069
    requirement: REQ-007
    subject: cmd/backstop
    text: The spec directory buildGateSteps hands to the test_verification, test_substantiveness, coverage and contracts dimensions is the resolved artifact root's spec directory, so under a .backstop/-rooted project all four read the specs that exist instead of erroring on an absent project-root specs/
    tests:
      - TestBuildGateSteps_SpecDirectoryConsumersReadResolvedArtifactRoot

  # REQ-007 — the ID-resolution fallback, added in 1.2.7. The write path always read the
  # resolved root; the fallback SCAN did not, and the two disagreeing silently restarted
  # numbering at 001 under a configured root.
  - id: CLM-070
    requirement: REQ-007
    subject: cmd/backstop
    text: Under a configured artifact root, artifact ID numbering continues from the artifacts already on disk rather than silently restarting at 001, because the ID-resolution fallback scan reads the same resolved root the write path writes into
    tests:
      - TestArtifactNew_IDNumberingContinuesUnderConfiguredRoot

contracts:
  - file: pkg/artifact/layout.go
    provides:
      - name: Kind
        kind: type
        signature: "type Kind string"
        notes: "The artifact kind vocabulary — spec, plan, adr, bundle, issue, directive, capability. It is the SAME seven the codebase already recognizes in three separate places (cmd/backstop/artifact_discover.go:18-26, pkg/scaffold/scaffold.go:33-89, pkg/validate/resolved_by.go:46-52); this declaration exists so there is one, not a fourth."
      - name: LayoutFor
        kind: function
        signature: "func LayoutFor(kind Kind) (KindLayout, bool)"
        notes: "The single type→{Directory, Extension} authority (REQ-006). Returns ok=false for an unrecognized kind — never a zero-value KindLayout that would silently resolve to the project root itself."
      - name: KindLayout
        kind: type
        signature: "type KindLayout struct { Directory string; Extension string }"
        notes: "Directory is the bare directory NAME (\"specs\"), never a path — joining it to a root is the resolver's job, which is what keeps the root in exactly one place."
      - name: Kinds
        kind: function
        signature: "func Kinds() []Kind"
        notes: "Deterministic enumeration of every kind, so the exhaustiveness claim (CLM-038) tests the table rather than a hand-written list that can drift from it."
      - name: ClassifyFilename
        kind: function
        signature: "func ClassifyFilename(name string) (Kind, bool)"
        notes: "The filename→kind authority replacing cmd/backstop's artifactPatterns map. Exclusive by construction (CLM-039): a filename matches at most one kind. Consumed by BOTH CLI discovery and the REQ-008 ungated-artifact scan, so the set of files the gate says it left out is defined by the same predicate as the set it picks up."
      - name: NonCorpusDirNames
        kind: function
        signature: "func NonCorpusDirNames() []string"
        notes: "ADDED 2026-08-14 (v1.2.4) — declared during PLAN-SPEC-068's implementation, which found this symbol NECESSARILY exported and absent from this contract block. SIGNATURE UNCHANGED BY ISSUE-122; only the names it returns changed. It returns the shared TOOL-AGNOSTIC BASE of non-corpus directory names no artifact corpus scan descends into, deterministically ordered: .git, testdata, prototype. The ecosystem-specific dependency directory names it once also carried are GONE from core (ISSUE-122, resolved 2026-08-16) — core bakes no language or tool knowledge, so those names now arrive as data from installed packs' `classification.dependency_dirs` and are unioned onto this base by `artifact.NonCorpusDirs`, which is what each corpus walk actually consumes. `.backstop` is DELIBERATELY ABSENT from the list, because its exclusion is ROOT-RELATIVE and each caller layers its own rule on top — CLI discovery skips `.backstop` wholesale EXCEPT when it IS the resolved root, while the REQ-008 ungated scan always walks it and excludes only `.backstop/packs` beneath it (see the FindUngatedArtifacts notes). It returns a COPY so a caller cannot mutate the shared list. EXPORTED for the same reason as the other symbols in this file: its consumers live in DIFFERENT packages — `cmd/backstop/artifact_discover.go` and `pkg/gate`'s ungated-artifact scan (`pkg/gate/artifact_status.go`) — and one hand-typed copy per caller is exactly the drift this file exists to prevent, since a drift would make the set of files the gate picks up and the set it reports leaving out disagree."
      - name: Root
        kind: type
        signature: "type Root struct { Path string; Declared string; Configured bool }"
        notes: "Path is ALWAYS ABSOLUTE AND CLEANED — ResolveRoot absolutizes a relative projectRoot rather than passing it through (REQ-006), which is the decision that keeps FindUngatedArtifacts' directory comparison from degenerating. Declared is the raw backstop.yml value (empty when unconfigured); Configured distinguishes 'the consumer chose this root' from 'nobody said, so it is the project root'. REQ-008's loud-failure condition keys on Configured, which is precisely the distinction the bundle's 0.10.0 correction says v1.0.0 was missing."
      - name: ResolveRoot
        kind: function
        signature: "func ResolveRoot(projectRoot, declared string) (Root, error)"
        notes: "The ONE root resolution (REQ-006). Absolutizes projectRoot via filepath.Abs before joining, so the returned Root.Path is absolute even when the caller passes `\".\"` — which runGate does whenever DiscoverConfigPath fails (gate.go:77) while runArtifactValidate passes an absolute os.Getwd() (artifact_validate.go:284). Rejects an absolute or escaping DECLARED value (the declaration is project-relative by rule; the resolved Path is absolute by guarantee — these are different strings and the distinction is deliberate) and returns a typed error for a configured root that is absent or is not a directory. CORRECTION (2026-08-14, v1.2.4) — WHICH typed error, made explicit, because this note previously implied the same wrong mapping as Implementation step 2 and the drift is LIVE (PLAN-SPEC-070's doctor branches on the two error types). The mapping is: ABSENT (os.IsNotExist) → *RootMissingError; ABSOLUTE, ESCAPING, and NOT-A-DIRECTORY → *RootInvalidError with a populated Reason. That matches the RootInvalidError contract note below (CLM-035/036/037) and was confirmed as the shipped, correct behavior during PLAN-SPEC-068's implementation, where CLM-035's mandated test pins not-a-directory to *RootInvalidError. It takes the declared STRING rather than a *config.Config so pkg/artifact keeps its stdlib-only import set and every consumer can import it without a cycle."
      - name: RootMissingError
        kind: type
        signature: "type RootMissingError struct { Declared string; Path string }"
        notes: "Typed so callers can distinguish 'configured root absent' (REQ-008 loud failure, exit non-zero) from 'declared value malformed' (a config defect) without string matching."
      - name: RootInvalidError
        kind: type
        signature: "type RootInvalidError struct { Declared string; Reason string }"
        notes: "Covers absolute, escaping, and not-a-directory declarations (CLM-035/036/037)."
      - name: Dir
        kind: method
        signature: "func (r Root) Dir(kind Kind) string"
        notes: "The only sanctioned way to name an artifact type directory. Every literal filepath.Join(projectRoot, \"specs\") in the codebase becomes a call to this."
    consumes:
      - source: path/filepath
        name: Join
        kind: function
      - source: os
        name: Stat
        kind: function
  - file: pkg/schema/cohort.go
    provides:
      - name: Cohort
        kind: type
        signature: "type Cohort struct { ID string; Digests map[string]string }"
        notes: "ID is the content-derived cohort identifier (REQ-001); Digests is keyed by the `<type>/v<N>` identity artifacts declare in schema_version, so REQ-003's per-artifact record is a lookup rather than a second computation."
      - name: ComputeCohort
        kind: function
        signature: "func ComputeCohort(fsys fs.FS) (Cohort, error)"
        notes: "Takes an fs.FS (backstopcore.SchemaFS in production) so the in-place-revision claims can be proven against a fixture FS without rewriting the embedded schemas. Folds a sorted path→file-digest manifest into the ID, which is what makes it deterministic and order-independent (CLM-002) while still changing on any byte (CLM-001)."
      - name: DigestFor
        kind: method
        signature: "func (c Cohort) DigestFor(schemaVersion string) (string, bool)"
        notes: "ok=false is the UNCOVERED signal REQ-002 refuses on (CLM-005). It must never return an empty string with ok=true — that shape is what would let an uncovered schema read as covered-with-no-content."
      - name: SchemaIdentity
        kind: method
        signature: "func (c Cohort) SchemaIdentity(schemaVersion string) (string, bool)"
        notes: "Renders the REQ-003 record value `<schema_version>@<digest>`. The digest folds the extension schema AND the base schema resolved through `extends` (CLM-017) — LoadArtifactSchema merges the two (pkg/schema/load.go:99-144), so an identity covering only the extension would miss a base revision entirely."
    consumes:
      - source: crypto/sha256
        name: Sum256
        kind: function
      - source: io/fs
        name: WalkDir
        kind: function
  - file: pkg/gate/artifact_status.go
    provides:
      - name: ResolveArtifactStatus
        kind: function
        signature: "func ResolveArtifactStatus(artifactRoot string) (*ArtifactStatusResolution, error)"
        notes: "SIGNATURE UNCHANGED, MEANING CHANGED (REQ-007): the parameter is now the RESOLVED artifact root, not the project root, and both call sites (cmd/backstop/gate.go:902 and :983) pass the resolved value. The five literal joins at lines 171/193/211/226/241 become Root.Dir calls. The walkers' os.IsNotExist tolerance (lines 382/409/437) is DELIBERATELY KEPT: a missing type directory under an existing root stays a non-error, which is what preserves the empty-root pass (CLM-056). Loud failure moves to the ROOT, where the bundle's 0.10.0 correction says it belongs."
      - name: FindUngatedArtifacts
        kind: function
        signature: "func FindUngatedArtifacts(projectRoot string, root artifact.Root, nonCorpus artifact.NonCorpusDirs) ([]UngatedArtifact, error)"
        notes: "REQ-008's surfacing scan. RENAMED from an earlier FindOutOfRootArtifacts, and the rename is the point: the predicate is PER KIND, not root containment. It absolutizes projectRoot on entry — Root.Path is absolute by ResolveRoot's guarantee, and comparing an absolute Root.Dir(kind) against a relative walk path yields either zero findings or one per artifact, both of which look like a working implementation (CLM-067). THIRD PARAMETER ADDED 2026-08-16 (ISSUE-122): the non-corpus exclusion set arrives AS A PARAMETER rather than being read from a core-local literal, because core bakes no ecosystem nouns. It walks projectRoot, skips the trees that set determines — the tool-agnostic base (.git, testdata, prototype) plus whatever installed packs declare via classification.dependency_dirs — and, by a rule LOCAL to this walk and unchanged by ISSUE-122, installed pack trees under .backstop/packs (note that .backstop ITSELF is walked, unlike in discovery's skip set, or the unconfigured motivating case is excluded away before it can be found). The zero-value NonCorpusDirs excludes the tool-agnostic base, so a caller that wires no packs degrades to today-minus-declarations rather than walking .git. It classifies each remaining file with artifact.ClassifyFilename, and reports every one whose parent directory is not Root.Dir(kind). A containment test would return EMPTY whenever the root is unconfigured — projectRoot contains itself — which is exactly the backstop-runtime shape the bundle cites, so containment made the clause vacuous on its own motivating example (CLM-058). Directness matters: the status walk reads type directories with os.ReadDir and does not recurse (walkArtifactDir, artifact_status.go:379-404), so a file nested one level below specs/ is genuinely UNGATED and is reported — though it is still DISCOVERED and schema-validated, because DiscoverArtifacts is a recursive filepath.Walk; the two facts are distinct and CLM-063 pins the distinction. A mere subdirectory is not a finding."
      - name: UngatedArtifact
        kind: type
        signature: "type UngatedArtifact struct { Path string; Kind artifact.Kind; ExpectedDir string; Root string }"
        notes: "ExpectedDir is Root.Dir(Kind) — where discovery WOULD have found it — and Root is the resolved root that produced that expectation. The bundle requires the report name both the path and the root; ExpectedDir is what makes the report actionable rather than merely accusatory, and it is the same string a doctor-side layout report needs (SPEC-070 REQ-025 consumes this helper)."
    consumes:
      - source: pkg/artifact
        name: Root
        kind: type
      - source: pkg/artifact
        name: ClassifyFilename
        kind: function
      - source: pkg/artifact
        name: LayoutFor
        kind: function
  - file: pkg/gate/result.go
    provides:
      - name: GateResult
        kind: type
        signature: "type GateResult struct { /* existing fields */ BinaryVersion string `json:\"binary_version,omitempty\"`; SchemaCohort string `json:\"schema_cohort,omitempty\"`; SchemaIdentities []string `json:\"schema_identities,omitempty\"`; ArtifactRoot string `json:\"artifact_root,omitempty\"`; ArtifactRootConfigured bool `json:\"artifact_root_configured\"` }"
        notes: "Five ADDITIVE fields (REQ-003/REQ-004/REQ-008) alongside the existing GitSHA/GeneratedAt provenance at result.go:144-145, additive under the unchanged gate/v1 schema exactly as ISSUE-059 added the existing pair. No existing field changes type or meaning. ArtifactRootConfigured DELIBERATELY CARRIES NO omitempty: encoding/json drops a false bool entirely under omitempty, and false is the DEFAULT and the motivating state — an unconfigured root — so an omitempty here would make REQ-008's 'name whether that root was configured' unsatisfiable in --json for the exact case the requirement was written for (CLM-068). The other five are string/slice fields whose empty value means genuinely absent, so omitempty is correct for them."
    consumes:
      - source: pkg/artifact
        name: Root
        kind: type
  - file: pkg/config/config.go
    provides:
      - name: Config
        kind: type
        signature: "type Config struct { /* existing fields */ ArtifactRoot string `yaml:\"artifact_root,omitempty\" json:\"artifact_root,omitempty\"` }"
        notes: "One additive field (REQ-006). It is the RAW declared value; resolution is pkg/artifact's. Note the two-pass strictness at config.go:36 — the yaml decoder uses KnownFields and a separate JSON-schema pass enforces additionalProperties:false — so the key MUST be added to artifacts/backstop-yml/v1/schema.json as well or a config declaring it fails the schema pass (CLM-041)."
    consumes: []
  - file: pkg/scaffold/scaffold.go
    provides:
      - name: TargetDir
        kind: function
        signature: "func TargetDir(artifactType string, root artifact.Root) string"
        notes: "SIGNATURE CHANGED (REQ-007): the second parameter becomes the resolved Root rather than a bare projectRoot string, so a caller cannot accidentally pass the project root where the artifact root belongs. The body is root.Dir(artifact.Kind(artifactType))."
      - name: ArtifactTypeFor
        kind: function
        signature: "func ArtifactTypeFor(artifactType string) (ArtifactTypeConfig, bool)"
        notes: "REPLACES the exported package-level ValidArtifactTypes map (2026-08-15 correction — this contract previously said the map was KEPT with its Directory values re-sourced from artifact.LayoutFor, which is not what shipped). The map is gone and Directory is REMOVED from ArtifactTypeConfig entirely, so no per-type directory field remains for LayoutFor to feed; callers needing a directory use artifact.LayoutFor / Root.Dir directly. A lookup function rather than a table because an exported package-level map is writable by every importer — the hazard go-standards' no-global-mutable-state names, raised as a pack finding during implementation. The rest of ArtifactTypeConfig (IDPrefix, DigitCount, DefaultStatus, FileExtension, BodySections) is scaffold's own and stays."
    consumes:
      - source: pkg/artifact
        name: Root
        kind: type
      - source: pkg/artifact
        name: LayoutFor
        kind: function
  - file: pkg/scaffold/idresolver.go
    provides:
      - name: IDOptions
        kind: type
        signature: "type IDOptions struct { Root artifact.Root; Executor GitExecutor; MaxRetries int }"
        notes: "FIELD CHANGED (REQ-007, 2026-08-15 correction — an exported-API change the Implementation narrative did not originally describe): ProjectRoot string becomes Root artifact.Root. ResolveID's local-scan fallback counts existing artifacts to choose the next number, so it must read the SAME directory artifact new writes to; holding a project root let the two disagree, and under a configured .backstop root the fallback scanned a nonexistent <project>/specs, found nothing and silently restarted numbering at 001. Taking an artifact.Root — which ResolveRoot guarantees absolute — makes passing the wrong directory unrepresentable. Callers hoist the ResolveRoot call ABOVE ResolveID (cmd/backstop/artifact_new.go:92) so one resolved Root feeds both the ID scan and TargetDir."
    consumes:
      - source: pkg/artifact
        name: Root
        kind: type
  - file: pkg/validate/resolved_by.go
    provides:
      - name: typedRefArtifactExists
        kind: function
        signature: "func typedRefArtifactExists(art *artifact.ParsedArtifact, ref string) bool"
        notes: "Signature unchanged. The private resolvedByTypeDir map (resolved_by.go:46-52) carries THREE things in one literal — a typed-ref PREFIX (BUNDLE/SPEC/ISSUE/PLAN/DIR), a directory, and an extension — and only the last two are layout. Those two are DELETED and read from artifact.LayoutFor; the PREFIX→Kind mapping STAYS LOCAL to this file, because artifact.LayoutFor is keyed by Kind and nothing in pkg/artifact maps a ref prefix to one. That is deliberate, not an oversight: the five prefixes are the resolved-by GRAMMAR's accepted vocabulary (they match resolvedByTypedRefRe, and DIR→directive would defeat any naive lowercase coercion anyway), which is a validation concern rather than a layout one. The remaining local map is prefix→artifact.Kind only and holds no directory or extension string. The SIBLING-RELATIVE anchoring (filepath.Join(dir, \"..\", typeDir), line 120) is PRESERVED — see Sharp Edges: contrary to the bundle's phrasing this function never hardcoded the artifact ROOT, and rewriting it to consume a configured root would BREAK the .backstop/ case it already handles."
    consumes:
      - source: pkg/artifact
        name: LayoutFor
        kind: function
  - file: cmd/backstop/artifact_discover.go
    provides:
      - name: DiscoverArtifacts
        kind: function
        signature: "func DiscoverArtifacts(root artifact.Root, typeFilters []string, nonCorpus artifact.NonCorpusDirs) ([]DiscoveredArtifact, error)"
        notes: "SIGNATURE CHANGED (REQ-007), then WIDENED AGAIN 2026-08-16 (ISSUE-122). The private artifactPatterns map (artifact_discover.go:18-26) is DELETED in favor of artifact.ClassifyFilename. The non-corpus exclusion set no longer lives in a core-local literal: it arrives AS A PARAMETER — the tool-agnostic base (.git, testdata, prototype) unioned with whatever installed packs declare via classification.dependency_dirs — because core bakes no ecosystem nouns. The ROOT-RELATIVE `.backstop` rule stays LOCAL to this walk and is UNCHANGED by that: installed pack trees (.backstop/packs) are always excluded, while `.backstop` itself is walked when it IS the resolved root, which is what makes a .backstop/-rooted repo discoverable (CLM-045/CLM-046). The zero-value NonCorpusDirs excludes the tool-agnostic base, so a caller that wires no packs degrades to today-minus-declarations rather than walking .git."
    consumes:
      - source: pkg/artifact
        name: Root
        kind: type
      - source: pkg/artifact
        name: ClassifyFilename
        kind: function
  - file: cmd/backstop/artifact_validate.go
    provides:
      - name: ValidateResult
        kind: type
        signature: "type ValidateResult struct { Pass bool; ViolationsCount int; Violations []validate.Violation; ArtifactsFound int; ArtifactsAsserted int; ScannedRoot string; Records []ArtifactValidationRecord }"
        notes: "The four existing fields (artifact_validate.go:27-32) are unchanged. ArtifactsAsserted and ScannedRoot are REQ-002's legibility pair — they are what makes the zero-artifact pass read as EMPTY rather than as VERIFIED, and the zero case must carry them too, so the early `return ValidateResult{Pass: true}` at line 129 stops being a bare literal. Records is REQ-003's per-artifact carrier (CLM-007-013): the flat GateResult.SchemaIdentities list cannot express an identity for EACH validated artifact, so the validate side needs its own shape rather than reusing it."
      - name: ArtifactValidationRecord
        kind: type
        signature: "type ArtifactValidationRecord struct { Path string; Type string; SchemaVersion string; SchemaIdentity string; Schemaless bool }"
        notes: "One record per discovered artifact. SchemaIdentity is cohort.SchemaIdentity's `<schema_version>@<digest>` (REQ-003). Schemaless is REQ-002's plan marker — a plan routes by discovery type and declares no schema_version, and the record must say so rather than leaving an empty SchemaIdentity to be read as covered-with-no-content (CLM-009). Both renderings read these same records, which is what keeps human and --json from drifting (CLM-014)."
      - name: ValidateArtifacts
        kind: function
        signature: "func ValidateArtifacts(cfg ValidateConfig) (ValidateResult, error)"
        notes: "Signature unchanged; behavior gains REQ-002's assertion. Before loading a schema it looks the declared schema_version up in the cohort and REFUSES on uncovered, with a diagnostic naming path, schema_version and cohort id — today's failure is a generic wrapped `loading schema for %s` (line ~150) naming none of the three. cfg.ProjectRoot becomes the RESOLVED artifact root, so this function and the gate cannot scan different corpora."
      - name: ValidateConfig
        kind: type
        signature: "type ValidateConfig struct { /* existing fields */ Root artifact.Root }"
        notes: "The resolved root travels with the config rather than being re-derived inside, so runArtifactValidate (which roots at os.Getwd(), artifact_validate.go:284) and realArtifactValidator (which roots at the backstop.yml directory) hand in one resolved value and Sharp Edge 10's two-roots divergence has exactly one place to be reconciled."
    consumes:
      - source: pkg/artifact
        name: Root
        kind: type
      - source: pkg/schema
        name: Cohort
        kind: type
  - file: cmd/backstop/output.go
    provides:
      - name: jsonEnvelope
        kind: type
        signature: "type jsonEnvelope struct { SchemaVersion string `json:\"schema_version\"`; Pass bool `json:\"pass\"`; ViolationsCount int `json:\"violations_count\"`; Violations []jsonViolation `json:\"violations\"`; BinaryVersion string `json:\"binary_version,omitempty\"`; SchemaCohort string `json:\"schema_cohort,omitempty\"`; ArtifactsAsserted int `json:\"artifacts_asserted\"`; ScannedRoot string `json:\"scanned_root,omitempty\"`; Artifacts []jsonArtifactRecord `json:\"artifacts,omitempty\"` }"
        notes: "The validate-side equivalent of GateResult's additive fields (REQ-003, REQ-004), on the envelope at output.go:47-52. ArtifactsAsserted carries NO omitempty for the same reason ArtifactRootConfigured does not: zero is the value that must stay legible, since an omitted count on an empty pass is exactly the illegibility REQ-002 names."
      - name: jsonArtifactRecord
        kind: type
        signature: "type jsonArtifactRecord struct { Path string `json:\"path\"`; Type string `json:\"type\"`; SchemaIdentity string `json:\"schema_identity,omitempty\"`; Schemaless bool `json:\"schemaless,omitempty\"` }"
        notes: "The wire form of ArtifactValidationRecord. It exists as its own type rather than reusing GateResult's flat SchemaIdentities []string because CLM-013 requires an identity for EACH validated artifact and a flat list of schema identities cannot express that binding — GateResult's list is per SCHEMA (CLM-016), a genuinely different claim, and collapsing the two would silently weaken this one."
      - name: FormatResult
        kind: method
        signature: "func (f *JSONFormatter) FormatResult(result validate.ValidationResult) (string, error)"
        notes: "The formatter must reach the ValidateResult fields the envelope now carries; whether that is a widened parameter or a second method is the planner's call, but the human and JSON renderings must read ONE resolved value (CLM-014/CLM-023), mirroring how the version command already shares reportedVersion across its two branches (root.go:107)."
    consumes:
      - source: cmd/backstop
        name: ArtifactValidationRecord
        kind: type
  - file: cmd/backstop/root.go
    provides:
      - name: computeCohortID
        kind: function
        absent: true
        signature: "func computeCohortID(schemas []string) string"
        notes: "ABSENCE assertion (absent: true) — the signature value is documentary only; an absence entry is probed as a grep text-presence check keyed on the NAME over cmd/backstop/root.go, and the compiled ast-grep signature pattern is never used for it. DELETED (REQ-001). It is the path-derived cohort at root.go:205-220 that BUNDLE-014's in-place bundle/v2 revision walked straight past. Deleting it is a COMPILE BREAK for TestCLI_ComputeCohortID_Empty (root_test.go:323) and TestCLI_ComputeCohortID_NonEmpty (root_test.go:329), which call it directly and assert its literal `1-schemas[spec/v1]` shape — see Sharp Edges. No legacy cohort string survives anywhere; a second one would make the guard optional."
      - name: NewRootCommand
        kind: function
        signature: "func NewRootCommand() *cobra.Command"
        notes: "SIGNATURE UNCHANGED; it is the declarable surface for this file's rewrite. `versionCmd` is a LOCAL variable inside this constructor (root.go:97) and its RunE is an inline closure — neither is a package-level declaration, so neither is nameable as a contract entry of its own; the constructor that builds them is. The `version` command's RunE is REWRITTEN (REQ-001, REQ-005). It stops calling ListSchemas/computeCohortID and reads schema.ComputeCohort(SchemaFS).ID, and it renders BuildIdentity's commit and build date alongside the version in both branches. The existing resolve-once-share-both-renderings shape at root.go:107 is PRESERVED and extended — it is why the JSON and text outputs cannot drift. TestCLI_Version_HumanOutput (root_test.go:132) and TestCLI_Version_JSONOutput (:148) assert only loose content and field PRESENCE, never the cohort string's shape, so they keep passing across this rewrite and must keep passing — they are the regression floor for it."
    consumes:
      - source: pkg/schema
        name: ComputeCohort
        kind: function
  - file: cmd/backstop/gate.go
    provides:
      - name: realArtifactValidator
        kind: type
        signature: "type realArtifactValidator struct { projectRoot string; root artifact.Root; cohort schema.Cohort; nonCorpus artifact.NonCorpusDirs }"
        notes: "Gains the resolved artifact root and the cohort (gate.go:1568) so the gate's artifact_validation step asserts on the same values the CLI does rather than re-deriving them per step. FOURTH FIELD ADDED 2026-08-16 (ISSUE-122): nonCorpus is the artifact-corpus exclusion set derived ONCE in buildGateSteps from the installed packs, carried as a FIELD for the same reason cohort is — so this struct's two corpus consumers, the per-artifact validation config and the ungated scan, cannot measure different corpora. Its zero value is reachable and correct: every keyed test construction of this struct wires no packs and gets artifact.NonCorpusDirs{}, which excludes the tool-agnostic base and nothing else."
      - name: ValidateAll
        kind: method
        signature: "func (v *realArtifactValidator) ValidateAll(ctx context.Context) ([]gate.Violation, error)"
        notes: "Signature unchanged, RETURN SET WIDENED (REQ-008): after converting ValidateArtifacts' violations it appends the gate.FindUngatedArtifacts(projectRoot, root, nonCorpus) findings — the third argument being the struct's own exclusion-set field, so the scan and the validation config measure one corpus — each marked ProjectWide so filterViolations (pkg/gate/scope.go:302-326) cannot drop them from a diff-scoped run — the file a finding names is by definition not in the diff. This is the ONLY place the ungated scan is invoked; a second call site would let the two disagree about the root."
      - name: runGate
        kind: function
        signature: "func runGate(cmd *cobra.Command, args []string) error"
        notes: "Resolves the artifact root ONCE per run (REQ-006/REQ-007), immediately after config load and project-root discovery (gate.go:65-80, which already holds cfg and projectRoot), and hands the resolved Root to realArtifactValidator, to both ResolveArtifactStatus call sites (:902, :983) and to buildGateSteps (:124). It passes `\".\"` as projectRoot whenever DiscoverConfigPath fails (gate.go:77) — ResolveRoot's absolutization is what makes that safe. A *artifact.RootMissingError from this resolution is REQ-008's loud failure and exits non-zero; it is distinguished by TYPE, never by string match."
      - name: buildGateSteps
        kind: function
        signature: "func buildGateSteps(projectRoot string, root artifact.Root, scope ...*gate.GateScope) []gate.StepFunc"
        notes: "SIGNATURE CHANGED (REQ-007, 2026-08-14 correction — the FIFTH consumer). `specDir := filepath.Join(projectRoot, \"specs\")` (gate.go:652) becomes `root.Dir(artifact.KindSpec)`; the four dimensions it feeds are unchanged in every other respect (test_verification :706, test_substantiveness :710, coverage :718, contracts :721). projectRoot STAYS as a separate parameter — it is the CODE directory those same steps walk for test files and the two are genuinely different roots under a `.backstop/` layout, so collapsing them would break test discovery. The variadic scope tail stays last. All three call sites pass the resolved root: gate.go:124 (from runGate's one resolution), baseline.go:64 and waiver.go:88 (which resolve it the same way runGate does, or a widened signature silently re-introduces the project-root default on two of three paths)."
    consumes:
      - source: pkg/artifact
        name: ResolveRoot
        kind: function
      - source: pkg/gate
        name: FindUngatedArtifacts
        kind: function
  - file: cmd/backstop/version.go
    provides:
      - name: BuildIdentity
        kind: type
        signature: "type BuildIdentity struct { Version string; Commit string; BuildDate string }"
        notes: "The one resolved value both the human and --json renderings of `version` read, and the source of the REQ-004 stamp on results — so a result cannot report a different version than `backstop version` does (CLM-024)."
      - name: resolveBuildIdentity
        kind: function
        signature: "func resolveBuildIdentity(injectedVersion, injectedCommit, injectedDate string, info *debug.BuildInfo, ok bool) BuildIdentity"
        notes: "Pure function of its arguments, mirroring the existing resolveVersion (version.go:38) so the whole matrix is testable without building a binary per case. It DELEGATES the version string to the existing resolveVersion unchanged — the anti-spoofing behavior is reused, not reimplemented (CLM-026). Commit and date come from info.Settings vcs.revision / vcs.time, with vcs.modified appended to the COMMIT as a dirty marker (CLM-027) and `unknown` when no source supplies one (CLM-029)."
      - name: resolveVersion
        kind: function
        signature: "func resolveVersion(injected string, info *debug.BuildInfo, ok bool) string"
        notes: "UNCHANGED, declared to keep this file's surface complete and to make the preservation explicit: REQ-005 adds fields around it and must not alter its precedence or its rejections."
    consumes:
      - source: runtime/debug
        name: ReadBuildInfo
        kind: function
---

# SPEC-068: Trustworthy Green Guards

## Overview

BUNDLE-003 splits into three seeds and this one goes first, for a stated reason:
the init seed's REQ-004 (`.backstop/`-rooted layout) is unimplementable until the
artifact root is resolvable, and every acceptance claim the other two seeds make
is only as trustworthy as the validator asserting it. The bundle's DD-15 is the
posture: a misleading error stops you, but a false green lets nonconformant
governed state propagate and accrue derived work on an invalid foundation, so on
"I cannot tell" the answer is REFUSE — deliberately inverting this codebase's
usual loud-≠-blocking default.

Two mechanisms in the dogfood evidence manufacture that class of failure, and
this spec closes both.

**Cohort skew.** A binary's schema cohort is currently a function of schema
PATHS. `computeCohortID` (`cmd/backstop/root.go:206`) strips `artifacts/` and
`/schema.json` off each embedded path and prints `11-schemas[adr/v1,…,spec/v1]`.
BUNDLE-014 revised `bundle/v2` IN PLACE, adding `Draft`-prefixed section names,
`text:` in place of `statement:`, per-REQ `version:`, and `bundle.updated` — the
path set never moved, so the reported cohort was byte-identical before and after.
The artifact side was no better: `artifacts/bundle/v2/schema.json` pins
`"schema_version": {"const": "bundle/v2"}`, revision-free, so an artifact said
`bundle/v2` on both sides of the change. A stale validator asserting "my cohort
covers bundle/v2" told the truth about the string and lied about the schema. That
is how BUNDLE-001 in bclabs-portal was authored, promoted to `defined`, and
revised five times with `artifact validate` reporting GREEN while violating the
then-current schema in 41 places. Making the binary side content-derived and the
record side revision-bearing is the pair; neither alone detects an in-place
revision.

**Silent undiscovery.** The artifact root is hardcoded, and not in three places
but four. The bundle names three and all three are still present at HEAD:
`pkg/gate/artifact_status.go` joins the project root with `specs`/`bundles`/
`issues`/`directives`/`plans` (lines 171, 193, 211, 226, 241);
`pkg/validate/resolved_by.go` carries its own `BUNDLE`→`bundles` type map (lines
46–52); `pkg/scaffold/scaffold.go` scaffolds into the same root names (lines
35–89). The fourth was found while grounding this spec and is worse than any of
them: `DiscoverArtifacts` (`cmd/backstop/artifact_discover.go:48`) `SkipDir`s
`.backstop` outright, and `ValidateArtifacts` returns `Pass: true` when discovery
yields nothing (`artifact_validate.go:129`). A consumer who adopts the
`.backstop/` layout the init seed will scaffold therefore gets a clean green over
an empty set. Without this fix REQ-029 would land and change nothing on the
validate path.

Everything here is core. There is no init code and no doctor code in this spec.

## Requirements

Requirements, claims, contracts and bundle pins are defined in frontmatter. In
summary — and this table states exactly what the requirements state, nothing
wider:

| Requirement | Bundle pin | What it makes true |
| --- | --- | --- |
| REQ-001 | `REQ-026@1.0.0` | The cohort id and a per-schema digest are functions of schema BYTES, not paths |
| REQ-002 | `REQ-027@1.0.0` | Validate and gate assert the cohort covers each declared `schema_version` and refuse green when they cannot prove it; an existing-but-empty root still passes |
| REQ-003 | `REQ-034@1.0.0` | The VALIDATION RECORD (not the artifact file) carries `schema_version@digest`, covering the base schema too |
| REQ-004 | `REQ-028@1.0.0` | Every gate and validate result records the producing binary's version and cohort |
| REQ-005 | `REQ-021@1.1.0` | `version` stamps build commit and build date; the anti-spoofing bare `dev` version string is preserved |
| REQ-006 | `REQ-029@1.0.0` | One layout table and one config-resolved artifact root — resolved to an ALWAYS-ABSOLUTE path — with `artifact_root` added to `backstop.yml` |
| REQ-007 | `REQ-029@1.0.0` | Gate status resolution, CLI discovery, typed-ref resolution, scaffolding, and the spec directory `buildGateSteps` feeds to the test_verification / test_substantiveness / coverage / contracts dimensions all read that one resolution |
| REQ-008 | `REQ-030@1.1.0` | Gate names the scanned root and whether it was configured (`false` included), fails loudly on a configured root that is absent, and surfaces — per kind, not by root containment — the artifact-shaped files the STATUS WALK would not reach, over its own enumerated exclusion set, without adopting them |

Bundle REQ-022 is IN THE SEED but is deliberately NOT pinned by this spec. See
Dependencies.

## Implementation

Ten steps. Steps 1–2 are the new authorities; 3–7 are the consumers; 8–10 are the
output surfaces.

1. **The layout authority — `pkg/artifact/layout.go` (REQ-006).** `pkg/artifact`
   imports only stdlib and `gopkg.in/yaml.v3` today, so every consumer —
   `pkg/gate`, `pkg/validate`, `pkg/scaffold`, `pkg/config`, `cmd/backstop` — can
   import it with no cycle. It holds one `Kind`→`{Directory, Extension}` table
   covering all seven kinds, `Kinds()` for exhaustive enumeration, and
   `ClassifyFilename` — which subsumes `cmd/backstop`'s `artifactPatterns` map
   and is consumed by BOTH discovery and the ungated-artifact scan, so the set of
   files the gate picks up and the set it reports leaving out are defined by one
   predicate.

2. **The root resolver — same file (REQ-006).** `ResolveRoot(projectRoot,
   declared)` returns a `Root{Path, Declared, Configured}`. Rejects absolute and
   escaping declarations with `*RootInvalidError`; ~~returns `*RootMissingError`
   for a configured root that is absent or is not a directory~~ returns
   `*RootMissingError` for a configured root that is ABSENT (see the 2026-08-14
   correction below); resolves an empty declaration to the project root marked
   unconfigured.

   > **CORRECTION (2026-08-14, v1.2.4) — NOT-A-DIRECTORY IS `*RootInvalidError`,
   > NOT `*RootMissingError`.** The struck clause folded not-a-directory into
   > `*RootMissingError`, contradicting `RootInvalidError`'s own contract note in
   > this same spec ("Covers absolute, escaping, and not-a-directory
   > declarations (CLM-035/036/037)"). The contract note is the correct one, and
   > it is the SHIPPED behavior: confirmed during PLAN-SPEC-068's
   > implementation, where the code and CLM-035's mandated test both pin
   > not-a-directory to `*RootInvalidError` with a populated `Reason`. The
   > authoritative mapping is: ABSENT → `*RootMissingError`; ABSOLUTE, ESCAPING,
   > NOT-A-DIRECTORY → `*RootInvalidError`. This is not cosmetic — the drift is
   > live downstream, because PLAN-SPEC-070's `backstop doctor` branches on
   > exactly these two error types, so a reader who trusted the stale clause
   > would route a not-a-directory root to the wrong diagnostic. `Root.Dir(kind)` is the
   only sanctioned way to name a type directory. Alongside it,
   `Config.ArtifactRoot` is added to `pkg/config` AND `artifact_root` is added to
   `artifacts/backstop-yml/v1/schema.json` — that schema is
   `additionalProperties: false` at the top level, so both edits are required or
   a config declaring the key fails the JSON-schema pass.

3. **Gate status resolution (REQ-007).** `ResolveArtifactStatus`'s parameter
   becomes the RESOLVED root and the five literal joins become `Root.Dir` calls.
   Both call sites (`cmd/backstop/gate.go:902`, `:983`) pass the resolved value,
   which `runGate` computes ONCE after loading config (`gate.go:65-80` already
   has both `cfg` and `projectRoot` in hand). The walkers' `os.IsNotExist`
   tolerance at lines 382/409/437 is KEPT deliberately: a missing type directory
   under an existing root must stay a non-error, because that is what preserves
   the empty-root pass. Loud failure moves up to the ROOT.

4. **CLI discovery (REQ-007).** `DiscoverArtifacts` takes a `Root` and walks
   `Root.Path`. Its skip list becomes root-relative: the non-corpus names — the
   tool-agnostic base (`.git`, `testdata`, `prototype`) plus whatever installed
   packs declare via `classification.dependency_dirs`, handed in as a parameter
   since ISSUE-122 rather than read from a core-local literal — still skip, and
   `.backstop/packs` ALWAYS skips regardless of where the root sits, but
   `.backstop` itself is no longer skipped when it IS the root. `artifactPatterns`
   is deleted in favor of `ClassifyFilename`. `runArtifactValidate` currently
   uses `os.Getwd()` as its project root (`artifact_validate.go:284`) while
   `runGate` uses the `backstop.yml` directory — reconcile on the config-derived
   root so the two commands cannot disagree about which corpus they scanned.

5. **Typed-reference resolution (REQ-007).** `resolvedByTypeDir`
   (`pkg/validate/resolved_by.go:46-52`) fuses three things in one literal: a
   typed-ref PREFIX key (`BUNDLE`, `SPEC`, `ISSUE`, `PLAN`, `DIR`), a directory,
   and an extension. Only the last two are layout, and only they move: the
   directory and extension come from `artifact.LayoutFor`, and the PREFIX→`Kind`
   mapping STAYS LOCAL to `resolved_by.go`. That is the intended end state, not a
   half-measure — `LayoutFor` is keyed by `Kind` and this spec's `pkg/artifact`
   surface deliberately declares no prefix→kind function, because the five
   prefixes are the resolved-by grammar's accepted vocabulary (they are what
   `resolvedByTypedRefRe` matches) rather than a layout fact, and `DIR` →
   `directive` would defeat any naive lowercase coercion in any case. After the
   change the local map holds prefix→`artifact.Kind` and no directory or
   extension string. Do NOT change the anchoring: `typedRefArtifactExists`
   resolves siblings relative to the CITING artifact's own `SourcePath`
   (`filepath.Join(dir, "..", td.dir)`, line 120), which already works under a
   `.backstop/` layout. Only the duplicated table is the defect here.

6. **Scaffolding (REQ-007, 2026-08-15 correction).** `TargetDir` takes a `Root`
   instead of a project root string. `IDPrefix`, `DigitCount`, `DefaultStatus`
   and `BodySections` are scaffold's own and stay put. The other two changes went
   further than this step originally described; the shipped shape is:

   - `ValidArtifactTypes` is NOT kept with its `Directory` values merely
     re-sourced. The exported package-level map is REPLACED by
     `ArtifactTypeFor(artifactType string) (ArtifactTypeConfig, bool)`, and
     `Directory` is REMOVED from `ArtifactTypeConfig` outright — there is no
     per-type directory field left for anything to read `artifact.LayoutFor`
     into, and callers needing a directory consume `artifact.LayoutFor` /
     `Root.Dir` directly. Replacing the map rather than re-sourcing one field
     also closed a `no-global-mutable-state` pack finding raised during
     implementation (an exported package-level map is writable by every
     importer), so this is a stronger change than de-duplicating a table.
   - `IDOptions.ProjectRoot` (a bare project-root string) becomes
     `IDOptions.Root` (`artifact.Root`) — an exported `pkg/scaffold` API change
     this step did not originally mention at all. It closes a real write-path vs
     ID-fallback-path root divergence found during impl-review: `ResolveID`'s
     local-scan fallback counts existing artifacts to pick the next number, so
     under a configured `.backstop` root it scanned a nonexistent
     `<project>/specs`, found nothing, and silently RESTARTED numbering at 001
     while the write below used the resolved root. Root resolution is therefore
     hoisted ABOVE the `ResolveID` call in `cmd/backstop/artifact_new.go`
     (`artifact_new.go:92`), so one resolved `Root` feeds both the ID scan and
     `TargetDir`.

7. **The gate's spec directory (REQ-007, 2026-08-14 correction).**
   `buildGateSteps` (`cmd/backstop/gate.go:647`) computes `specDir :=
   filepath.Join(projectRoot, "specs")` at line 652 and hands it to four
   dimensions — `test_verification` (`:706`), `test_substantiveness` (`:710`),
   `coverage` (`:718`), `contracts` (`:721`). It takes the resolved `Root` and
   `specDir` becomes `root.Dir(artifact.KindSpec)`. `projectRoot` STAYS a separate
   parameter: those same steps use it as the CODE directory they walk for test
   files, and under a `.backstop/` layout the artifact root and the code root are
   genuinely different. All three call sites pass the resolved root —
   `gate.go:124` from `runGate`'s single resolution, and `baseline.go:64` and
   `waiver.go:88`, which resolve it the same way; leaving either of the latter two
   on the project root would reinstate the defect on the baseline and waiver paths
   while the gate path looked fixed. This consumer is not interchangeable with the
   status walk: `gate.ExtractMandatedTests` (`pkg/gate/step_testverify.go:113-116`)
   ERRORS on a missing directory rather than tolerating it, which is why a
   `.backstop/`-rooted project hard-breaks on these four dimensions today.

8. **The cohort — `pkg/schema/cohort.go` (REQ-001, REQ-003).**
   `ComputeCohort(fsys)` folds a sorted `path:file-digest` manifest into one ID
   and records a per-`<type>/v<N>` digest. `DigestFor` returns `ok=false` for an
   uncovered `schema_version`; `SchemaIdentity` renders
   `<schema_version>@<digest>` including the base schema that `LoadArtifactSchema`
   merges in (`pkg/schema/load.go:99-144`). `computeCohortID`
   (`cmd/backstop/root.go:206`) is DELETED and the `version` command reads
   `Cohort.ID`.

9. **Assertion and refusal (REQ-002).** In `ValidateArtifacts`
   (`cmd/backstop/artifact_validate.go:113-214`), before loading a schema, look
   the artifact's declared `schema_version` up in the cohort. Uncovered ⇒ refuse
   with a diagnostic naming the artifact path, the declared `schema_version` and
   the cohort ID (today's failure is a generic wrapped `loading schema for %s`
   that names none of the three). Covered ⇒ record the identity and validate as
   before. Plans, which route by discovery type and carry no `schema_version`,
   are recorded as schema-less. Zero discovered artifacts under an existing root
   stays `Pass: true` — REQ-008 requires that outcome — but the result carries
   the asserted count and the scanned root so an empty pass reads as empty. The
   gate inherits all of it through `realArtifactValidator` (`gate.go:1568-1578`),
   which delegates to this same function.

10. **Result surfaces (REQ-003, REQ-004, REQ-008).** `GateResult` gains five
    additive fields — binary version, cohort, schema identities, artifact root,
    artifact-root-configured — placed alongside the `GitSHA`/`GeneratedAt` pair
    ISSUE-059 added the same way, keeping `gate/v1` unchanged. Four take
    `omitempty`; `ArtifactRootConfigured` does NOT, because `encoding/json` drops
    a `false` bool under `omitempty` and `false` is both the default and REQ-008's
    motivating state, so an `omitempty` there would emit no field at all in the
    one case the requirement exists for. The validate envelope
    (`cmd/backstop/output.go:47-52`) gains the equivalents plus an `artifacts`
    array of per-artifact records — `GateResult.SchemaIdentities` is a flat
    per-SCHEMA list and cannot express REQ-003's identity for EACH validated
    artifact, so the validate side carries its own record type rather than
    reusing it, and `artifacts_asserted` likewise omits `omitempty` so a zero
    count stays legible. Both
    renderings of each command read ONE resolved value, mirroring how the version
    command already shares `reportedVersion` between its JSON and text branches
    (`root.go:107`). Ungated-artifact findings are produced by
    `gate.FindUngatedArtifacts` and returned through `realArtifactValidator`
    so they land on the `artifact_validation` step, and they are marked
    `ProjectWide` — see Sharp Edges, without it a diff-scoped run filters them
    away.

Deliberately NOT in scope: any `init` or `doctor` code; the baseline artifact and
its schema (BUNDLE-007 / DD-10 own it, and this spec does not add cohort fields
to it); `.goreleaser.yml` (Sharp Edges); and bundle REQ-022's mechanism
(Dependencies).

## Verification

Verification configuration is in frontmatter: integration level, an 80% coverage
threshold, and a test command spanning the seven packages the requirements touch.
Integration is the honest level — the load-bearing claims are that four separate
consumers agree about one root, and that a `.backstop/`-rooted project and a
repo-root project reach equivalent discovery, neither of which can be shown
inside a single package.

Two claim families deserve a note on how they must be proven:

- The **in-place revision** claims (cohort ID change, identity change) must be
  proven against a FIXTURE `fs.FS` whose schema bytes change while its paths do
  not — not by editing the real embedded schemas, which would churn the whole
  corpus. This is why `ComputeCohort` takes an `fs.FS` rather than reading
  `SchemaFS` directly.
- The **ungated-artifact surfacing** claims must be proven on fixture projects,
  not by dogfood. Verified 2026-08-13: every artifact-shaped file in
  backstop-core outside the excluded trees sits directly in its expected type
  directory — 116 issues, 102 plans, 51 specs, 27 directives, 26 bundles, 18
  adrs, zero strays — so this repo produces no findings and cannot exercise the
  positive cases. That measurement is also CLM-060's evidence: the repo passes
  because its files are placed correctly, not because of a carve-out. Sixty-seven
  further artifact-shaped files DO live under `testdata/`, `prototype/` and
  `.backstop/packs/`, so the exclusion set is load-bearing, not decorative — drop
  it and this repo's own gate floods.

## Sharp Edges

1. **Discovery skips `.backstop` today, and the bundle does not mention it.** The
   bundle names three hardcodings; `cmd/backstop/artifact_discover.go:48` is a
   fourth, and it is the one that decides whether the whole feature does
   anything. `.backstop` is `SkipDir`'d unconditionally, and `ValidateArtifacts`
   returns `Pass: true` on an empty discovery — so a `.backstop/`-rooted consumer
   validating zero artifacts currently gets a green. If a planner scopes only the
   three files the bundle names, REQ-029 lands and the init seed's layout still
   cannot be validated. The fix has a trap of its own: `.backstop/packs/` holds
   installed packs, several of which are themselves backstop repos with their own
   artifacts, so the exclusion must become root-relative (always exclude
   `.backstop/packs`) rather than simply deleted.

2. **"Outside the resolved root" is the wrong predicate, and it fails on the
   bundle's own example.** This spec's 1.0.0 read REQ-030's surfacing clause as
   root containment, which is vacuous exactly where the requirement was written
   to bite: backstop-runtime places `.backstop/bundles/` in a repo that
   configures NO artifact root, so the root defaults to the project root, those
   bundles are INSIDE it, and a containment test reports nothing while none of
   them are discovered or gated. The bundle's own 0.10.0 correction says the
   surfacing clause is what closes that case — so the predicate must be per KIND:
   not directly in `Root.Dir(kind)`. Caught by SPEC-070's author while wiring
   doctor's REQ-025 against this contract, and corrected in 1.1.0; the helper was
   renamed `FindUngatedArtifacts` so the name cannot re-suggest containment. If a
   later reader is tempted back toward containment, the test is CLM-058.

3. **Diff-scoped runs filter File-less findings away.** `filterViolations`
   (`pkg/gate/scope.go:302-326`) keeps a violation only when it is `ProjectWide`
   or its `File` is in the computed scope. An ungated-artifact finding names a file
   that is by definition not in the diff, so a bare `backstop gate` would silently
   drop exactly the finding REQ-008 exists to surface. It must be marked
   `ProjectWide` — the same structural exemption build/typecheck findings use.
   This is a one-line property with a whole requirement riding on it, which is why
   it has its own claim.

4. **`resolved_by.go` does not hardcode the root — the bundle's phrasing drifts
   from the code.** The bundle lists it alongside two genuine root hardcodings,
   but `typedRefArtifactExists` (line 107) anchors on the citing artifact's OWN
   `SourcePath` and resolves siblings through `filepath.Join(dir, "..", td.dir)`.
   That already works under `.backstop/`. What IS duplicated is the
   type→directory map. "Fixing" this file by feeding it a configured root would
   break the sibling anchoring and regress a case that currently works.

5. **The zero-artifact pass is protected, not closed.** DD-15 says refuse on "I
   cannot tell", and it would be easy to read that as "finding nothing is
   suspicious". REQ-030 v1.1.0 says the opposite in as many words: an existing-
   but-empty root stays a legitimate pass, because the validated greenfield
   profile reached gate PASS with empty artifact dirs. The refusal is scoped
   strictly to "I found something I cannot prove I can validate." An implementer
   who tightens this into "empty is a failure" breaks the init seed's acceptance
   bar before it is written. SCOPE, NARROWED 2026-08-14 (v1.2.2): the protected
   outcome is ROOT RESOLUTION AND THE `artifact_validation` DIMENSION, not the
   aggregate gate verdict. Once REQ-007's fifth consumer lands, the four
   spec-directory dimensions read the resolved root's `specs/`, and the run over
   that fixture may be RED on their account while the empty-root pass this edge
   protects is fully intact. MECHANISM CORRECTED (2026-08-14, v1.2.3): the
   v1.2.2 wording said an empty `.backstop/` has no `specs/` directory and that
   `ExtractMandatedTests` (`pkg/gate/step_testverify.go:113-116`) errors loudly
   on it. That describes a shape `backstop init` cannot produce and is NOT the
   shape of the fixture behind CLM-056, whose `.backstop/specs/` EXISTS and is
   empty; `os.ReadDir` succeeds on it, so those four dimensions read it cleanly
   and any failure they report over that tree comes from the zero-artifact
   corpus itself, not from root resolution. The SCOPE is unchanged by this
   correction. An implementer who asserts `Pass: true` on the aggregate to
   satisfy CLM-056 will find that assertion in direct conflict with CLM-069.

6. **REQ-003's record choice buys cross-run detection, not in-run omniscience.**
   The bundle permits the digest to live on the artifact or on the validation
   record; this spec chooses the record, because an artifact-carried digest would
   require rewriting every artifact on every in-place schema revision — in a repo
   whose standing rule is that artifacts are never hand-edited — and would make
   the digest part of the content it digests. The honest consequence: a single
   run under a stale binary still cannot know it is stale. What the record buys
   is that its green is self-describing, so a later run, a CI job, or a human can
   see the identities differ. Quarantine is the consumer's act; this spec
   delivers the evidence, and the baseline — the one stored, projected-as-truth
   result in this system — is BUNDLE-007's and is deliberately untouched.

7. **The cohort string changes shape, and exactly two live tests die with it.**
   `11-schemas[adr/v1,…]` becomes a digest, so anything asserting the old form
   breaks. Verified at HEAD, the breakage is narrower and different from what a
   reader would guess. `TestCLI_ComputeCohortID_Empty` (`root_test.go:323`) and
   `TestCLI_ComputeCohortID_NonEmpty` (`:329`) call `computeCohortID` DIRECTLY and
   assert its literal output (`"empty"`, `1-schemas[spec/v1]`); deleting the
   function is a COMPILE BREAK for the `cmd/backstop` test package, not a
   behavioral failure, so it cannot be discovered late. Neither is mandated by any
   spec claim — verified: nothing in `specs/` names either test — so they are
   unowned tests whose premise disappears with their subject, and the plan must
   SANCTION their retirement explicitly rather than letting an implementer delete
   tests to make a build pass. What does NOT need updating is the pair a reader
   would expect to: `TestCLI_Version_HumanOutput` (`root_test.go:132`) and
   `TestCLI_Version_JSONOutput` (`:148`) are SPEC-005's CLM-022/CLM-023 and assert
   only loose content and field PRESENCE (`"version"`, `"go"`; the keys `version`,
   `schema_cohort`, `go_version`), never the cohort string's shape — they survive
   this change untouched and are the regression floor for the `version` rewrite.
   SPEC-005 (`status: draft`) still describes the path-derived cohort in its
   REQ-006 prose and needs an alignment pass through the artifact agents; that is
   a text change, not a test change. Do not work around any of this by keeping a
   second, legacy cohort string.

8. **`.goreleaser.yml` is not this spec's file.** It injects only
   `-X main.version` today (`.goreleaser.yml:49`) and is governed by the
   go-distribution pack's own rules (ISSUE-101), one of which asserts on the
   ldflags line. REQ-005 is therefore satisfied from `debug.BuildInfo` VCS
   settings, with injected commit/date honored IF a release path ever supplies
   them. A binary installed from the module cache has no VCS settings at all and
   will report `commit: unknown` — that is correct behavior, not a gap: for that
   binary the released version string already identifies the build.

9. **`artifact_root` is a hard-fail key against an older binary.** The
   `backstop.yml` schema is `additionalProperties: false`, so a consumer config
   declaring `artifact_root` fails outright on any binary predating this change.
   That is the compatibility direction these guards exist to make legible and it
   is not softened here — but the init seed must not scaffold the key into a
   config it then hands to an unknown binary without saying so.

10. **Two project roots, one corpus — and one of them is RELATIVE.** `artifact
    validate` roots at `os.Getwd()` (`artifact_validate.go:284`), which is
    absolute, while `gate` roots at the `backstop.yml` directory and falls back to
    the literal `"."` when `DiscoverConfigPath` fails (`gate.go:77`), which is
    not. Two failure modes ride on this and only one is obvious. The obvious one:
    a `validate` run from a subdirectory could resolve a different root than the
    gate and report a different corpus, both green — reconciled in step 4. The
    silent one: REQ-008's per-kind rule compares `Root.Dir(kind)` against each
    walked file's parent directory as STRINGS, so a relative root paired with an
    absolute walk (or the reverse) matches nothing or matches everything, and
    both outcomes look like a working implementation — zero findings reads as "a
    clean corpus" and one-per-artifact reads as "the scan is definitely running."
    This spec DECIDES it rather than leaving it to the implementer: `ResolveRoot`
    absolutizes its `projectRoot` argument and `Root.Path` is absolute by
    contract (REQ-006), and `FindUngatedArtifacts` absolutizes its own
    `projectRoot` argument on entry, so neither a `"."` caller nor a future one
    can reintroduce the mixed pair. CLM-066 and CLM-067 pin both halves.

11. **This repo's one capability artifact is a near-miss the per-kind rule must
    not trip over.** `capabilities/CAP-001-pack-gate-enforcement/capability.yml`
    is nested a directory deeper than the layout expects AND is named
    `capability.yml`, not `CAP-001-….capability.yml`. It escapes only because
    `ClassifyFilename`'s suffix test (`.capability.yml`) does not match a bare
    `capability.yml` — verified 2026-08-13. So the file is invisible to
    discovery today and stays invisible after this change, which is the
    behavior-preserving outcome. An implementer who "improves" the capability
    pattern to also match the bare name will turn this repo's own gate red on a
    file nobody asked them to touch. Leave the pattern alone; if the nesting is
    genuinely wrong, that is its own issue.

12. **The discovery skip list baked language/tool nouns into core. RESOLVED
    2026-08-16 by ISSUE-122 — do not re-introduce them.** As shipped by this
    spec, the skip list carried `vendor` (a Go noun) and `node_modules` (a
    JavaScript/npm noun) as literals in core code, which this spec knowingly
    propagated: it sat outside BUNDLE-003's requirement set, and REQ-007 was a
    behavior-PRESERVING change, so re-scoping it would have put an invariant fix
    on the critical path of the seed everything else waits on. It was filed as
    ISSUE-122 and that issue is now DELIVERED, so this is a closed defect rather
    than a knowingly-carried one. The names are GONE from core: `pkg/artifact`
    holds only the tool-agnostic base (`.git`, `testdata`, `prototype`), the
    ecosystem nouns arrive as data from installed packs'
    `classification.dependency_dirs`, and `artifact.NonCorpusDirs` unions the two
    into the ONE exclusion set both walks take as an explicit parameter. The
    hazard that survives is the OPPOSITE one: an editor who "restores" a familiar
    noun to a core literal re-opens ISSUE-122 while every test stays green,
    because the pack-declared half of the set already covers it in practice. Add
    a dependency directory name to a PACK, never to core.
    **The aggravating factor that is now structurally answered.** This spec gives
    the exclusion set a SECOND consumer: `FindUngatedArtifacts` excludes what
    REQ-008 determines. Note what that set is NOT — it is not "whatever discovery
    skips", and REQ-008 says so in as many words, because discovery skips
    `.backstop` wholesale except when `.backstop` is the root and inheriting that
    by reference would exclude the unconfigured motivating case before the scan
    could find it. The two rules are therefore ALMOST the same and differ in one
    entry, which is exactly the shape that invites two hand-typed copies. ONE
    shared authority holds the common names and each caller adds its own
    `.backstop` rule on top; the names are never typed twice. Two copies would
    drift, which would make the set of files the gate picks up and the set it
    reports leaving out disagree — the precise property the shared
    `ClassifyFilename` predicate exists to guarantee. The single authority is why
    ISSUE-122's fix landed in one place rather than two.

## Dependencies

**Bundle REQ-022 is in this seed and is NOT specced here.** It requires that when
a pack declares core capabilities the running binary does not provide, the
consumer-facing failure NAME that gap — the missing capability and the pack
requiring it — instead of surfacing only the downstream engine error (`declared
stdout_artifact … not produced`). Its v1.1.0 text defers the MECHANISM WHOLLY to
BUNDLE-020 (Pack Core Version Compatibility): what a pack declares and how it is
compared. BUNDLE-020's DD-4 is founder-resolved — a pack declares named
CAPABILITY CONTRACTS at the pack↔core wire seam and compatibility is a
capability-SET comparison, deliberately NOT a version ordering — but its OQ-2
(where it is enforced: add time, lock verification, or gate preflight) and OQ-3
(failure posture) are both still OPEN, and BUNDLE-020 is still `exploring`.

Nothing can be built from the outcome alone. To name a missing capability, core
must first be told what capabilities exist and which a pack requires; to know
where the diagnostic fires and whether it blocks, OQ-2 and OQ-3 must be answered.
A renderer with no producer would be dead code, and inventing the producer is
precisely what the bundle's own consumption boundary forbids: "This seed's spec
must consume whatever BUNDLE-020 lands and must not decide it." Writing a spec
requirement here would therefore mean either shipping a stub or answering another
bundle's open questions.

So it is recorded rather than resolved. **REQ-022 lands as a delta spec against
this one once BUNDLE-020 resolves OQ-2 and OQ-3**, and that spec owes: the
capability-set comparison consumed from BUNDLE-020's declaration surface, the
enforcement point OQ-2 names, the posture OQ-3 names, and the diagnostic that
names both the missing capability and the requiring pack ahead of any downstream
engine error. This spec's REQ-004 is its groundwork — a result that carries the
producing binary's identity is what makes a capability-gap diagnostic
attributable — and nothing here forecloses any option BUNDLE-020 still has open.
There is no version-ordering language anywhere in this spec, deliberately.

**This deferral needs a BUNDLE-003-side note, and that is not a fix this spec can
make.** BUNDLE-003's Spec Seeds table still reads "REQ-021, REQ-022, REQ-026 –
REQ-030, REQ-034 | 8" for the guards seed, so on the corpus REQ-022 reads as
covered by this spec while this spec pins seven of the eight. The convention this
repo already uses for exactly that situation is an explicit ownership note in the
BUNDLE — the same treatment REQ-033 got as "DANGLING: NO OWNER" and REQ-012 /
REQ-013 / REQ-018 got as CONSUMED-not-built pointing at ISSUE-055 and ISSUE-056.
REQ-022 needs the same one-line note naming BUNDLE-020 as the blocker and the
delta spec as its future owner. It is a bundle edit, not a spec edit, and it is
out of this spec's lane: bundles are evolved through the bundle agents, and a
spec author reaching into its own source bundle to reclassify a requirement is
precisely the drift that convention exists to prevent. Recorded here so the
deferral is not silently lost between the two artifacts; Review Question 11
carries the same flag to whoever reviews the implementation.

Two further external dependencies are consumed, not built: `pkg/recipe` and the
CI recipe pack (SPEC-054 / SPEC-067, both `implemented`) are the init seed's
concern, not this one; and the baseline subsystem (BUNDLE-007, ISSUE-056) owns
the stored result this spec's provenance fields will eventually let a consumer
quarantine.

## Review Questions

1. Does `DiscoverArtifacts` still exclude `.backstop/packs/` when the resolved
   artifact root IS `.backstop`? Show the test that adds an artifact-shaped file
   under an installed pack and asserts it is neither discovered nor reported as
   ungated.
2. Are the ungated-artifact findings marked `ProjectWide`? Run the gate
   diff-scoped on a fixture with a misplaced bundle and confirm the finding survives
   `filterViolations`.
2a. Is the surfacing predicate PER KIND (`not directly in Root.Dir(kind)`) rather
   than root containment? Point at the fixture where the artifact root is
   UNCONFIGURED and `.backstop/bundles/` is still surfaced — a containment
   implementation passes every other test in this family and fails only that one,
   which is why it has its own claim.
3. Does `resolveVersion` still return bare `dev` for `(devel)`, for a `+`-bearing
   version, and for a pseudo-version — with the same precedence order — after the
   build-identity fields were added around it?
4. Is `computeCohortID` actually deleted, or does a second cohort string still
   exist somewhere for backward compatibility? A surviving legacy path would make
   the guard optional.
5. Does the schema identity change when ONLY `artifacts/base/schema.json` is
   revised? If not, the digest is covering the extension schema alone and misses
   half of what `LoadArtifactSchema` merges.
6. Does a configured-but-absent artifact root fail, while a configured-and-empty
   one resolves and validates clean? Both, in the same test file, or the
   distinction will drift. And does the empty-root test assert the ROOT
   RESOLUTION / `artifact_validation` outcome rather than the aggregate
   `Pass: true`? Asserting the aggregate over an empty `.backstop/` collides with
   CLM-069, because a zero-artifact corpus can legitimately fail the four
   spec-directory dimensions — and others — for reasons that have nothing to do
   with root resolution.
7. Do `artifact validate` and `gate` resolve the same artifact root when
   `validate` is invoked from a subdirectory of the project?
8. Is `artifact_root` present in BOTH `pkg/config`'s struct and
   `artifacts/backstop-yml/v1/schema.json`? Adding only the struct field passes
   the yaml decoder and then fails the JSON-schema pass.
9. Did anything in this change introduce a capability-comparison or
   version-ordering mechanism? It must not — that is BUNDLE-020's, and its
   questions are open.
10. Is `Root.Path` absolute at every call site, and does `FindUngatedArtifacts`
    return the same findings for the relative and the absolute form of one
    project root? A mixed pair yields zero findings or one per artifact, and
    neither looks like a bug from the outside. Check the `gate.go:77` `"."`
    fallback path specifically — it is the one caller that can hand in a relative
    root.
11. Is `artifact_root_configured` present in `gate --json` for an UNCONFIGURED
    root, with the value `false`? An `omitempty` on that bool drops the field
    entirely in the default case, which is the case REQ-008 was written for.
12. Does the validate `--json` output carry an identity for EACH validated
    artifact, not just a flat list of schema identities? The gate side's
    per-schema list is a different claim (CLM-016) and cannot discharge CLM-013.
13. Has BUNDLE-003 gained its REQ-022 ownership note yet? This spec pins seven of
    the seed's eight bundle requirements and records why the eighth is deferred,
    but the bundle's own Spec Seeds table still lists REQ-022 under this seed.
    That note is a BUNDLE edit through the bundle agents — NOT a fix to make
    here — and it is flagged so the deferral is not lost between the two
    artifacts.
14. The cobra-cli-standards pack carries a stdout/stderr separation rule
    (`CLI-004`) that plainly applies to the new diagnostics on `version`,
    `validate` and `gate`, but the pack declares no `STD-CLI-NNN` identifier this
    spec could bind in `follows`. That mapping is NOT invented here (DD-13);
    confirm the diagnostics honor the rule and escalate the missing standard
    identifier rather than guessing one.

## References

- `bundles/BUNDLE-003-onboarding-experience.bundle.md` — source bundle
  (v0.10.2, `defined`); this spec is its "Trustworthy-green guards (core)" seed,
  first in the declared implementation order.
- `bundles/BUNDLE-020-pack-core-version-compatibility.bundle.md` — owner of
  REQ-022's mechanism; OQ-2 and OQ-3 open.
- `specs/SPEC-005-cli-foundation.spec.md` — original owner of the `version`
  command and the path-derived cohort this spec supersedes; needs an alignment
  pass.
- `issues/ISSUE-056-local-first-baseline-seeding.issue.md`,
  `issues/ISSUE-055-local-provenance-cache-for-local-packs.issue.md` — the
  consumed-not-built halves of the bundle, neither of which is this seed's.
- `issues/ISSUE-122-baked-ecosystem-literals-in-artifact-discover.issue.md` —
  the baked-ecosystem-noun defect Sharp Edge 12
  filed and this spec knowingly propagated; DELIVERED 2026-08-16, which is what
  the 1.3.0 reconciliation below records.

## Version History

- **1.3.0** (2026-08-16) — **Reconciliation to ISSUE-122 as delivered. No
  requirement verdict, claim id or mandated test name changes.** ISSUE-122
  removed the ecosystem nouns `vendor` and `node_modules` from core's shared
  non-corpus directory list; the exclusion BEHAVIOR is unchanged, only its SOURCE
  moved. Core now holds the TOOL-AGNOSTIC BASE (`.git`, `testdata`, `prototype`)
  and the ecosystem names arrive as data from installed packs'
  `classification.dependency_dirs`, unioned by the new `artifact.NonCorpusDirs`
  and handed to both corpus walks as an explicit parameter. **Contract drift
  (BLOCKING, since this spec is `implemented` and `contract_signature` is set to
  block).** Three signatures are corrected to the shipped code:
  `gate.FindUngatedArtifacts` gains a third `nonCorpus artifact.NonCorpusDirs`
  parameter; `DiscoverArtifacts` gains the same as a third parameter; and
  `realArtifactValidator`'s exhaustive field enumeration is EXTENDED with
  `nonCorpus artifact.NonCorpusDirs` (deliberately extended, not restructured
  into the drift-proof `/* existing fields */` form the neighbouring
  `ValidateConfig` entry uses). The call-form example inside `ValidateAll`'s note
  is updated in the same pass so it cannot contradict the signature it
  illustrates. **Prose.** The five-name enumeration is restated in all seven
  places as "the tool-agnostic base plus whatever installed packs declare" —
  including CLM-062's CLAIM TEXT, which the change made FALSE while its mandated
  test kept its name and kept passing, exactly the silent vacuous green this
  project exists to prevent. `TestFindUngatedArtifacts_ExcludesEnumeratedNonCorpusTreesButWalksDotBackstop`
  is UNCHANGED by name and still substantiates the claim: its fixture now spans
  both sources, with the `vendor`/`node_modules` half arriving through the
  injection. The `NonCorpusDirNames` SIGNATURE is unchanged — only the names it
  returns changed — so only that contract's note enumeration was stale. Sharp
  Edge 12 now records the bake as RESOLVED by ISSUE-122 rather than knowingly
  propagated, and names the surviving hazard: adding a dependency directory name
  back into a core literal re-opens the defect while every test stays green.
  The `.backstop` / `.backstop/packs` asymmetry is UNCHANGED throughout and
  remains local to each walk.
- **1.2.9** (2026-08-15) — **Scope clarification to 1.2.8's gate sentence, and
  nothing else.** No requirement, claim, contract, test or mechanism is added,
  removed or reworded. The close-out entry named `./bin/backstop gate` without
  stating its SCOPE, which could be misread as claiming the full-corpus sweep is
  clean. The sentence below is narrowed to the DIFF-SCOPED gate — the scope
  actually run and verified — and the `--all` picture is now stated explicitly
  rather than left to inference.
- **1.2.8** (2026-08-15) — **CLOSE-OUT: status `draft` -> `implemented`.** No
  requirement, claim, contract, test or mechanism is added, removed or reworded;
  this entry records the evidence the flip rests on. **Build and tests.**
  `go build ./...`, `go vet ./...` and `go test ./... -race -count=1` all clean
  across 17 packages, zero failures. **Gate (DIFF-SCOPED).** The diff-scoped
  `./bin/backstop gate` — bare, i.e. scoped to the diff vs merge-base plus
  untracked files, which is the scope actually run and verified for this
  close-out — reports `pass: true` with every real dimension clean —
  `pack_engines`, `test_verification`, `test_substantiveness`,
  `coverage_threshold`, `contract_signature`, `artifact_status_drift`,
  `requirement_traceability`, `waiver_resolution` — with only pre-existing
  advisory warnings remaining. This is deliberately NOT a claim about
  `./bin/backstop gate --all`. The full-corpus sweep DOES surface violations —
  `pack_engines`, `test_substantiveness`, `coverage_threshold` and
  `contract_signature` all report findings — and those are pre-existing corpus
  debt rather than breakage this implementation introduced: this spec's entire
  contract surface, `pkg/artifact/layout.go`, appears ZERO times in that sweep's
  output; the `contract_signature` findings are owned by SPEC-056, SPEC-057,
  SPEC-042 and SPEC-047, every one of them already `implemented` — and therefore
  already under contract enforcement — before this flip; and the `pack_engines`
  and `test_substantiveness` findings sit in unrelated pre-existing files
  carrying the established `accepted-risk` waiver pattern. That is this project's
  standing "gate dogfood mostly dark" / "self-pack RED is roadmap" position, not
  new debt from this work. One `--all` finding IS attributable to a file this
  spec touched and is being fixed under separate lineage rather than papered over
  here: `pkg/scaffold/idresolver.go` sits at 87.5% (77 of 88 statements), which
  trips the 90% PER-FILE coverage ratchet that only the full sweep checks — a
  different granularity from the 80 package-level floor this spec declares and
  that both the diff-scoped gate and the impl-reviewer read. **Corpus.** `./bin/backstop artifact validate --all` reports `pass: true`
  over 347 artifacts. **Mandated tests.** All 71 mandated test names in this
  spec's `claims` block (70 as planned, plus CLM-070's
  `TestArtifactNew_IDNumberingContinuesUnderConfiguredRoot`, added in 1.2.7)
  were confirmed PRESENT in the tree by name at close-out time, and the
  impl-reviewer ran real MUTATION tests against five of the trickiest claims —
  each correctly reddened when the fix under test was reverted, so those claims
  are falsification-verified rather than presence-verified. **Coverage.** Every
  touched package sits well above the 80 floor this spec declares:
  `pkg/artifact` 96.2, `pkg/schema` 94.7, `pkg/config` 90.4, `pkg/validate`
  96.2, `pkg/gate` 93.4, `pkg/scaffold` 93.4, `cmd/backstop` 91.6.
  **Pack releases.** Implementation required two real external pack releases,
  both executed and verified on their public remotes:
  `backstop-ai/backstop-core-architecture@0.1.2` and
  `backstop-ai/backstop-self@1.1.3` — the architecture and self packs had to
  learn the new one-layout-authority shape before core could go green against
  them, which is the packs-only path working as designed rather than a
  workaround. **Contract enforcement.** Contract-signature enforcement activates
  at `implemented` status, so the corpus-wide validate above was re-run AFTER
  this flip and is the real test of the `contracts` block, not a pre-flip
  reading.
- **1.0.0 through 1.2.7** — this spec records its own revision history INLINE
  rather than in this section: each correction is stamped in the requirement,
  claim or contract text it changed (see REQ-007's and REQ-008's
  `CORRECTION (2026-08-14, v1.2.x)` clauses, CLM-056's narrowing, and the
  `NonCorpusDirNames` and `ResolveRoot` contract notes). This section is opened
  by the close-out entry above; earlier revisions are deliberately NOT
  back-filled here, because restating them would duplicate — and risk drifting
  from — the in-place text that is their authority.
