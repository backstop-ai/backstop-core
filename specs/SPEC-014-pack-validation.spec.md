---
title: "SPEC-014: Pack Validation Pipeline"
number: SPEC-014
created: "2026-04-14"
status: draft
schema_version: spec/v1
spec_version: 1.0.0

implementation:
  summary: >
    Implement the pack validation pipeline as two CLI commands (pack check and
    pack test) backed by a pkg/packval library. The pipeline runs six phases in
    strict dependency order: (1) structural, (2) coherence, (3) fixture
    execution, (4) archetype, (5) layer, (6) risk_class. pack check runs phases
    1/2/4/5/6 (manifest-only, no external tools). pack test runs all six phases
    including phase 3 (semgrep, tool_config temp modules, layer 3 sandbox).
    Output is JSON by default (ADR-0001) with every error including phase, check
    name, offending item, message, fix hint, and manifest path. Early
    termination skips remaining phases when any phase fails. The pipeline is
    idempotent and side-effect-free.
  package: pkg/packval

verification:
  level: integration
  test_command: go test ./pkg/packval/... ./cmd/backstop/... -run TestPackVal -race -coverprofile=cover.out
  coverage_threshold: 80

requirements:
  - id: REQ-001
    text: >
      pack check must run phases 1 (structural), 2 (coherence), 4 (archetype),
      5 (layer), and 6 (risk_class). Phase 3 (fixture execution) must NOT run
      during pack check.
    supports: pack-validation:REQ-001

  - id: REQ-002
    text: >
      pack test must run all six phases (1 through 6) including phase 3 (fixture
      execution), which invokes external tools such as semgrep and language
      toolchains.
    supports: pack-validation:REQ-002

  - id: REQ-003
    text: >
      Phase 1 structural validation must verify: manifest parses as valid YAML,
      all required fields present (name, version, language, archetype, content),
      field values are valid enums (archetype must be enforcement or code,
      version must be valid semver), language must be a value from the supported
      language set (initially: go; unrecognized language values must be
      rejected), layer 2 rules declare a rule field pointing to an existing
      file, all content type declarations are valid, and all referenced file
      paths exist on disk.
    supports: pack-validation:REQ-003

  - id: REQ-004
    text: >
      Phase 1 must exclude tool_config.file from file-existence checks because
      it is a consumer-side target path, not a pack-internal file.
    supports: pack-validation:REQ-004

  - id: REQ-005
    text: >
      Phase 1 must verify that risk_class is present and a valid enum (security,
      correctness, style, perf) on every rule, including standalone tool_config
      entries that have their own id. A missing risk_class is a structural
      error. An invalid risk_class value is a structural error.
    supports: pack-validation:REQ-005

  - id: REQ-006
    text: >
      Phase 2 coherence must verify: every rule has at least one claim, every
      claim has both positive and negative fixtures (lists, at least one of
      each), every fixture file exists and is non-empty, and claim IDs are
      unique within the pack.
    supports: pack-validation:REQ-006

  - id: REQ-007
    text: >
      Phase 2 must enforce rule ID uniqueness spanning both
      content.ruleset.rules and tool_config entries that have their own id. A
      duplicate ID across these two sources is a hard error.
    supports: pack-validation:REQ-007

  - id: REQ-008
    text: >
      Phase 2 must verify that tool_config entries with their own rule ID also
      have claims and fixtures proving the tool catches the intended violation.
    supports: pack-validation:REQ-008

  - id: REQ-009
    text: >
      Phase 2 must check that pairs_with.rules entries in scaffold declarations
      resolve to actual rule IDs in the pack. Dangling references are a
      coherence warning, not a hard error.
    supports: pack-validation:REQ-009

  - id: REQ-010
    text: >
      Phase 2 must emit a warning (not hard error) for orphan fixture files in
      the fixture directory that are not referenced by any claim.
    supports: pack-validation:REQ-010

  - id: REQ-011
    text: >
      Phase 3 fixture execution for layer 1-2 semgrep rules must run semgrep
      --test requiring 100% pass rate: every positive fixture must NOT trigger
      the rule, every negative fixture MUST trigger the rule.
    supports: pack-validation:REQ-011

  - id: REQ-012
    text: >
      Phase 3 must verify that the pack rule ID exactly matches the semgrep rule
      ID in the referenced rule file. A mismatch is a hard error.
    supports: pack-validation:REQ-012

  - id: REQ-013
    text: >
      Phase 3 fixture execution for tool_config-dependent rules must create a
      temporary module environment (copying the pack's go.mod), copy the fixture
      file in, run the configured tool, and check results. Positive fixtures
      must pass clean. Every negative fixture must trigger the expected
      diagnostic.
    supports: pack-validation:REQ-013

  - id: REQ-014
    text: >
      Phase 3 must run go mod tidy for Go packs in a temporary copy of the
      pack directory before any fixture execution to ensure dependencies are
      resolved. The original pack directory is not modified — go mod tidy's
      side effects (go.sum changes) are confined to the temp copy. Failure to
      resolve deps is a Phase 3 pre-check error. Language-specific dependency
      resolution commands for other languages are defined when those languages
      are added to the supported set.
    supports: pack-validation:REQ-014

  - id: REQ-015
    text: >
      Phase 3 fixture execution for layer 3 validators must invoke the
      validator as validator.sh <fixture-path>. For single-file input_scope
      the path is a file; for multi-file input_scope the path is a directory.
      Exit 0 = pass, exit non-zero = fail. Positive fixtures must exit 0.
      Every negative fixture must exit non-zero.
    supports: pack-validation:REQ-015

  - id: REQ-016
    text: >
      Phase 3 must validate complete scaffolds by rendering with sample_config
      from the manifest, then running the scaffold's test_command. Tests must
      pass.
    supports: pack-validation:REQ-016

  - id: REQ-017
    text: >
      Phase 3 must validate skeleton scaffolds with structural checks only:
      scaffold directory exists, expected files present, test function names
      present. Tests must NOT be run. test_command is only used for complete
      scaffolds.
    supports: pack-validation:REQ-017

  - id: REQ-018
    text: >
      Phase 3 must verify SDK references by checking that the provides surface
      is declared at the manifest level. SDK test suite execution is not in
      scope.
    supports: pack-validation:REQ-018

  - id: REQ-019
    text: >
      Phase 4 archetype enforcement must verify that code packs (declaring sdk
      or scaffolds) also declare rules, and every scaffold has at least one
      enforcement rule. A code pack without enforcement rules is a hard error.
    supports: pack-validation:REQ-019

  - id: REQ-020
    text: >
      Phase 4 must enforce bidirectional co-occurrence: in a code pack, every
      rule must reference at least one scaffold or SDK via pairs_with. A rule
      without code content pairing in a code pack is a hard error.
    supports: pack-validation:REQ-020

  - id: REQ-021
    text: >
      Phase 4 must verify that enforcement packs do not declare sdk or
      scaffolds. An enforcement pack with code content (sdk or scaffolds) is a
      hard error.
    supports: pack-validation:REQ-021

  - id: REQ-022
    text: >
      Phase 5 layer enforcement must verify every rule declares its layer (1,
      2, or 3) and that risk_class is a valid enum on all rules.
    supports: pack-validation:REQ-022

  - id: REQ-023
    text: >
      Phase 5 must check that category (presence, structural, other) is present
      ONLY on layer 3 rules. Layer 3 rules MUST declare a category field.
      category must NOT appear on layer 1 or layer 2 rules.
    supports: pack-validation:REQ-023

  - id: REQ-024
    text: >
      Phase 5 must auto-accept layer 3 rules with category presence or
      structural (no justification required). Category other requires a
      mandatory non-empty justification field.
    supports: pack-validation:REQ-024

  - id: REQ-025
    text: >
      Phase 5 must verify layer 3 rules declare input_scope (single-file or
      multi-file) and a validator field pointing to an executable file.
    supports: pack-validation:REQ-025

  - id: REQ-026
    text: >
      Phase 6 risk class enforcement must verify every security-class rule has
      at least one negative fixture with bypass_attempt true per claim. The
      negative fixture list accepts both plain path strings and objects with
      path and optional bypass_attempt boolean; pack test normalizes before
      checking.
    supports: pack-validation:REQ-026

  - id: REQ-027
    text: >
      Phase 6 must enforce independent fixture coverage per claim for
      security-class rules. No shared fixtures across security claims within
      the same rule.
    supports: pack-validation:REQ-027

  - id: REQ-028
    text: >
      The validation pipeline must run phases in strict dependency order
      (structural > coherence > fixtures > archetype > layer > risk_class). If
      phase N fails, phases N+1 through 6 must be skipped.
    supports: pack-validation:REQ-028

  - id: REQ-029
    text: >
      Validation must be idempotent and side-effect-free. Running pack check or
      pack test twice on the same pack must produce the same pass/fail result,
      the same errors, and the same warnings. Timing fields (duration_ms) are
      excluded from the idempotency guarantee. No files modified, no state
      persisted, no network calls.
    supports: pack-validation:REQ-029

  - id: REQ-030
    text: >
      Validation output must be JSON by default per ADR-0001. Every error must
      include: the failing phase, the specific check name, the offending item
      (rule ID, claim ID, file path), a human-readable message, a fix hint the
      agent can act on, and the manifest_path where the problem originates.
    supports: pack-validation:REQ-030

  - id: REQ-031
    text: >
      Validation must support a --format=text flag for human-readable output as
      a secondary format. The primary consumer is an agent.
    supports: pack-validation:REQ-031

  - id: REQ-032
    text: >
      When a negative fixture fails to trigger its rule in Phase 3, the error
      must include a fix hint with engine-limitation guidance: explaining that
      the fixture may represent a pattern the rule engine cannot detect and
      should be removed and documented rather than shipped as an untestable
      fixture.
    supports: pack-validation:REQ-032

  - id: REQ-033
    text: >
      Errors must be reported in pipeline phase order, then by manifest order
      within a phase, to enable the agent to work through them top-down.
    supports: pack-validation:REQ-033

  - id: REQ-034
    text: >
      Phase 3 must execute layer 3 validators in a restricted process
      environment that prevents filesystem writes outside the pack directory,
      network access, and environment variable access. Sandbox violations are
      hard errors.
    supports: pack-validation:REQ-034

claims:
  # --- REQ-001: pack check phases ---
  - id: CLM-001
    requirement: REQ-001
    text: pack check runs phases 1, 2, 4, 5, 6 and reports results
    tests:
      - TestPackVal_CheckRunsManifestOnlyPhases
  - id: CLM-002
    requirement: REQ-001
    text: pack check does NOT run phase 3 (fixture execution)
    tests:
      - TestPackVal_CheckSkipsPhase3

  # --- REQ-002: pack test phases ---
  - id: CLM-003
    requirement: REQ-002
    text: pack test runs all six phases including phase 3
    tests:
      - TestPackVal_TestRunsAllSixPhases

  # --- REQ-003: Phase 1 structural ---
  - id: CLM-004
    requirement: REQ-003
    text: Phase 1 rejects manifest that does not parse as valid YAML
    tests:
      - TestPackVal_P1_InvalidYAML
  - id: CLM-005
    requirement: REQ-003
    text: Phase 1 rejects manifest missing required field name
    tests:
      - TestPackVal_P1_MissingName
  - id: CLM-006
    requirement: REQ-003
    text: Phase 1 rejects manifest missing required field version
    tests:
      - TestPackVal_P1_MissingVersion
  - id: CLM-007
    requirement: REQ-003
    text: Phase 1 rejects manifest missing required field language
    tests:
      - TestPackVal_P1_MissingLanguage
  - id: CLM-008
    requirement: REQ-003
    text: Phase 1 rejects manifest missing required field archetype
    tests:
      - TestPackVal_P1_MissingArchetype
  - id: CLM-009
    requirement: REQ-003
    text: Phase 1 rejects manifest missing required field content
    tests:
      - TestPackVal_P1_MissingContent
  - id: CLM-010
    requirement: REQ-003
    text: Phase 1 rejects invalid archetype value (not enforcement or code)
    tests:
      - TestPackVal_P1_InvalidArchetype
  - id: CLM-011
    requirement: REQ-003
    text: Phase 1 rejects invalid version (non-semver)
    tests:
      - TestPackVal_P1_InvalidVersion
  - id: CLM-012
    requirement: REQ-003
    text: Phase 1 rejects unrecognized language value
    tests:
      - TestPackVal_P1_UnsupportedLanguage
  - id: CLM-013
    requirement: REQ-003
    text: Phase 1 accepts language go as supported
    tests:
      - TestPackVal_P1_LanguageGoAccepted
  - id: CLM-014
    requirement: REQ-003
    text: Phase 1 rejects layer 2 rule where rule field points to nonexistent file
    tests:
      - TestPackVal_P1_RuleFileMissing
  - id: CLM-015
    requirement: REQ-003
    text: Phase 1 rejects referenced file path that does not exist on disk
    tests:
      - TestPackVal_P1_ReferencedFileNotFound
  - id: CLM-016
    requirement: REQ-003
    text: Phase 1 passes a fully valid manifest with all required fields
    tests:
      - TestPackVal_P1_ValidManifest

  # --- REQ-004: tool_config.file exclusion ---
  - id: CLM-017
    requirement: REQ-004
    text: Phase 1 does not check tool_config.file for file existence
    tests:
      - TestPackVal_P1_ToolConfigFileExcluded

  # --- REQ-005: risk_class on all rules ---
  - id: CLM-018
    requirement: REQ-005
    text: Phase 1 rejects rule missing risk_class
    tests:
      - TestPackVal_P1_MissingRiskClass
  - id: CLM-019
    requirement: REQ-005
    text: Phase 1 rejects rule with invalid risk_class value
    tests:
      - TestPackVal_P1_InvalidRiskClass
  - id: CLM-020
    requirement: REQ-005
    text: Phase 1 rejects standalone tool_config entry missing risk_class
    tests:
      - TestPackVal_P1_ToolConfigMissingRiskClass
  - id: CLM-021
    requirement: REQ-005
    text: Phase 1 accepts all four valid risk_class values (security, correctness, style, perf)
    tests:
      - TestPackVal_P1_ValidRiskClassSecurity
      - TestPackVal_P1_ValidRiskClassCorrectness
      - TestPackVal_P1_ValidRiskClassStyle
      - TestPackVal_P1_ValidRiskClassPerf

  # --- REQ-006: Phase 2 coherence ---
  - id: CLM-022
    requirement: REQ-006
    text: Phase 2 rejects rule with no claims
    tests:
      - TestPackVal_P2_RuleNoClaims
  - id: CLM-023
    requirement: REQ-006
    text: Phase 2 rejects claim with no positive fixtures
    tests:
      - TestPackVal_P2_ClaimNoPositiveFixtures
  - id: CLM-024
    requirement: REQ-006
    text: Phase 2 rejects claim with no negative fixtures
    tests:
      - TestPackVal_P2_ClaimNoNegativeFixtures
  - id: CLM-025
    requirement: REQ-006
    text: Phase 2 rejects fixture file that does not exist
    tests:
      - TestPackVal_P2_FixtureFileNotFound
  - id: CLM-026
    requirement: REQ-006
    text: Phase 2 rejects fixture file that is empty (zero bytes)
    tests:
      - TestPackVal_P2_FixtureFileEmpty
  - id: CLM-027
    requirement: REQ-006
    text: Phase 2 rejects duplicate claim IDs within the pack
    tests:
      - TestPackVal_P2_DuplicateClaimIDs

  # --- REQ-007: Rule ID uniqueness ---
  - id: CLM-028
    requirement: REQ-007
    text: Phase 2 rejects duplicate rule ID between ruleset and tool_config
    tests:
      - TestPackVal_P2_DuplicateRuleIDRulesetAndToolConfig
  - id: CLM-029
    requirement: REQ-007
    text: Phase 2 rejects duplicate rule ID within ruleset rules
    tests:
      - TestPackVal_P2_DuplicateRuleIDWithinRuleset
  - id: CLM-030
    requirement: REQ-007
    text: Phase 2 accepts unique rule IDs spanning ruleset and tool_config
    tests:
      - TestPackVal_P2_UniqueRuleIDsAcrossSources

  # --- REQ-008: tool_config traceability ---
  - id: CLM-031
    requirement: REQ-008
    text: Phase 2 rejects tool_config with own rule ID but no claims
    tests:
      - TestPackVal_P2_ToolConfigOwnIDNoClaims
  - id: CLM-032
    requirement: REQ-008
    text: Phase 2 rejects tool_config with own rule ID but no fixtures
    tests:
      - TestPackVal_P2_ToolConfigOwnIDNoFixtures
  - id: CLM-033
    requirement: REQ-008
    text: Phase 2 accepts tool_config with own rule ID and claims with fixtures
    tests:
      - TestPackVal_P2_ToolConfigOwnIDWithClaimsAndFixtures

  # --- REQ-009: pairs_with resolution ---
  - id: CLM-034
    requirement: REQ-009
    text: Phase 2 emits warning for dangling pairs_with.rules reference
    tests:
      - TestPackVal_P2_DanglingPairsWithWarning
  - id: CLM-035
    requirement: REQ-009
    text: Phase 2 does NOT emit hard error for dangling pairs_with.rules reference
    tests:
      - TestPackVal_P2_DanglingPairsWithNotError
  - id: CLM-036
    requirement: REQ-009
    text: Phase 2 accepts pairs_with.rules that resolve to existing rule IDs
    tests:
      - TestPackVal_P2_ValidPairsWithReference

  # --- REQ-010: orphan fixtures ---
  - id: CLM-037
    requirement: REQ-010
    text: Phase 2 emits warning for orphan fixture files not referenced by any claim
    tests:
      - TestPackVal_P2_OrphanFixtureWarning
  - id: CLM-038
    requirement: REQ-010
    text: Phase 2 does NOT emit hard error for orphan fixture files
    tests:
      - TestPackVal_P2_OrphanFixtureNotError

  # --- REQ-011: semgrep fixture execution ---
  - id: CLM-039
    requirement: REQ-011
    text: Phase 3 passes when all positive fixtures do not trigger the semgrep rule
    tests:
      - TestPackVal_P3_SemgrepPositivePass
  - id: CLM-040
    requirement: REQ-011
    text: Phase 3 fails when a positive fixture triggers the semgrep rule (false positive)
    tests:
      - TestPackVal_P3_SemgrepPositiveFalsePositive
  - id: CLM-041
    requirement: REQ-011
    text: Phase 3 passes when all negative fixtures trigger the semgrep rule
    tests:
      - TestPackVal_P3_SemgrepNegativeAllTrigger
  - id: CLM-042
    requirement: REQ-011
    text: Phase 3 fails when a negative fixture does not trigger the semgrep rule
    tests:
      - TestPackVal_P3_SemgrepNegativeNotTriggered

  # --- REQ-012: semgrep rule ID match ---
  - id: CLM-043
    requirement: REQ-012
    text: Phase 3 fails when pack rule ID does not match semgrep rule ID
    tests:
      - TestPackVal_P3_SemgrepRuleIDMismatch
  - id: CLM-044
    requirement: REQ-012
    text: Phase 3 passes when pack rule ID matches semgrep rule ID
    tests:
      - TestPackVal_P3_SemgrepRuleIDMatch

  # --- REQ-013: tool_config fixture execution ---
  - id: CLM-045
    requirement: REQ-013
    text: Phase 3 creates temp module environment with pack go.mod for tool_config fixtures
    tests:
      - TestPackVal_P3_ToolConfigTempModule
  - id: CLM-046
    requirement: REQ-013
    text: Phase 3 passes when tool_config positive fixture produces no diagnostic
    tests:
      - TestPackVal_P3_ToolConfigPositiveClean
  - id: CLM-047
    requirement: REQ-013
    text: Phase 3 fails when tool_config negative fixture does not trigger expected diagnostic
    tests:
      - TestPackVal_P3_ToolConfigNegativeNotTriggered
  - id: CLM-048
    requirement: REQ-013
    text: Phase 3 passes when tool_config negative fixture triggers expected diagnostic
    tests:
      - TestPackVal_P3_ToolConfigNegativeTriggered

  # --- REQ-014: go mod tidy pre-check ---
  - id: CLM-049
    requirement: REQ-014
    text: Phase 3 runs go mod tidy before fixture execution for Go packs
    tests:
      - TestPackVal_P3_GoModTidyRuns
  - id: CLM-050
    requirement: REQ-014
    text: Phase 3 reports pre-check error when go mod tidy fails
    tests:
      - TestPackVal_P3_GoModTidyFails

  # --- REQ-015: layer 3 validator execution ---
  - id: CLM-051
    requirement: REQ-015
    text: Phase 3 invokes layer 3 validator with file path for single-file input_scope
    tests:
      - TestPackVal_P3_Layer3SingleFileInvocation
  - id: CLM-052
    requirement: REQ-015
    text: Phase 3 invokes layer 3 validator with directory path for multi-file input_scope
    tests:
      - TestPackVal_P3_Layer3MultiFileInvocation
  - id: CLM-053
    requirement: REQ-015
    text: Phase 3 passes when layer 3 positive fixture exits 0
    tests:
      - TestPackVal_P3_Layer3PositiveExitZero
  - id: CLM-054
    requirement: REQ-015
    text: Phase 3 fails when layer 3 positive fixture exits non-zero
    tests:
      - TestPackVal_P3_Layer3PositiveExitNonZero
  - id: CLM-055
    requirement: REQ-015
    text: Phase 3 passes when layer 3 negative fixture exits non-zero
    tests:
      - TestPackVal_P3_Layer3NegativeExitNonZero
  - id: CLM-056
    requirement: REQ-015
    text: Phase 3 fails when layer 3 negative fixture exits 0
    tests:
      - TestPackVal_P3_Layer3NegativeExitZero

  # --- REQ-016: complete scaffold validation ---
  - id: CLM-057
    requirement: REQ-016
    text: Phase 3 renders complete scaffold with sample_config and runs test_command
    tests:
      - TestPackVal_P3_CompleteScaffoldRenderAndTest
  - id: CLM-058
    requirement: REQ-016
    text: Phase 3 fails when complete scaffold test_command fails
    tests:
      - TestPackVal_P3_CompleteScaffoldTestFails

  # --- REQ-017: skeleton scaffold validation ---
  - id: CLM-059
    requirement: REQ-017
    text: Phase 3 checks skeleton scaffold directory exists and expected files present
    tests:
      - TestPackVal_P3_SkeletonScaffoldStructure
  - id: CLM-060
    requirement: REQ-017
    text: Phase 3 checks skeleton scaffold test function names are present
    tests:
      - TestPackVal_P3_SkeletonScaffoldTestNames
  - id: CLM-061
    requirement: REQ-017
    text: Phase 3 does NOT run test_command for skeleton scaffolds
    tests:
      - TestPackVal_P3_SkeletonScaffoldNoTestExecution

  # --- REQ-018: SDK reference validation ---
  - id: CLM-062
    requirement: REQ-018
    text: Phase 3 passes when SDK provides surface is declared in manifest
    tests:
      - TestPackVal_P3_SDKProvidesDeclared
  - id: CLM-063
    requirement: REQ-018
    text: Phase 3 fails when SDK provides surface is missing from manifest
    tests:
      - TestPackVal_P3_SDKProvidesMissing

  # --- REQ-019: code pack requires rules ---
  - id: CLM-064
    requirement: REQ-019
    text: Phase 4 fails for code pack with scaffolds but no rules
    tests:
      - TestPackVal_P4_CodePackNoRules
  - id: CLM-065
    requirement: REQ-019
    text: Phase 4 fails for code pack scaffold with no enforcement rule
    tests:
      - TestPackVal_P4_ScaffoldNoEnforcementRule
  - id: CLM-066
    requirement: REQ-019
    text: Phase 4 passes for code pack with scaffolds and enforcement rules
    tests:
      - TestPackVal_P4_CodePackWithRulesPass

  # --- REQ-020: bidirectional co-occurrence ---
  - id: CLM-067
    requirement: REQ-020
    text: Phase 4 fails when code pack rule has no pairs_with reference to scaffold or SDK
    tests:
      - TestPackVal_P4_RuleNoPairsWithInCodePack
  - id: CLM-068
    requirement: REQ-020
    text: Phase 4 passes when every code pack rule has pairs_with reference
    tests:
      - TestPackVal_P4_AllRulesHavePairsWith

  # --- REQ-021: enforcement pack no code content ---
  - id: CLM-069
    requirement: REQ-021
    text: Phase 4 fails when enforcement pack declares sdk
    tests:
      - TestPackVal_P4_EnforcementPackWithSDK
  - id: CLM-070
    requirement: REQ-021
    text: Phase 4 fails when enforcement pack declares scaffolds
    tests:
      - TestPackVal_P4_EnforcementPackWithScaffolds
  - id: CLM-071
    requirement: REQ-021
    text: Phase 4 passes for enforcement pack with rules only (no sdk, no scaffolds)
    tests:
      - TestPackVal_P4_EnforcementPackRulesOnlyPass

  # --- REQ-022: layer declaration and risk_class ---
  - id: CLM-072
    requirement: REQ-022
    text: Phase 5 fails when a rule does not declare its layer
    tests:
      - TestPackVal_P5_MissingLayer
  - id: CLM-073
    requirement: REQ-022
    text: Phase 5 fails when a rule declares an invalid layer value (not 1, 2, or 3)
    tests:
      - TestPackVal_P5_InvalidLayer
  - id: CLM-074
    requirement: REQ-022
    text: Phase 5 passes for rules with valid layers (1, 2, 3) and valid risk_class
    tests:
      - TestPackVal_P5_ValidLayerAndRiskClass

  # --- REQ-023: category on layer 3 only ---
  - id: CLM-075
    requirement: REQ-023
    text: Phase 5 fails when layer 1 rule has category field
    tests:
      - TestPackVal_P5_Layer1WithCategoryFails
  - id: CLM-076
    requirement: REQ-023
    text: Phase 5 fails when layer 2 rule has category field
    tests:
      - TestPackVal_P5_Layer2WithCategoryFails
  - id: CLM-077
    requirement: REQ-023
    text: Phase 5 passes when layer 3 rule has valid category field
    tests:
      - TestPackVal_P5_Layer3WithCategoryPass
  - id: CLM-078
    requirement: REQ-023
    text: Phase 5 fails when layer 3 rule is missing category field
    tests:
      - TestPackVal_P5_Layer3MissingCategory

  # --- REQ-024: category auto-acceptance ---
  - id: CLM-079
    requirement: REQ-024
    text: Phase 5 auto-accepts layer 3 rule with category presence (no justification)
    tests:
      - TestPackVal_P5_CategoryPresenceAutoAccepted
  - id: CLM-080
    requirement: REQ-024
    text: Phase 5 auto-accepts layer 3 rule with category structural (no justification)
    tests:
      - TestPackVal_P5_CategoryStructuralAutoAccepted
  - id: CLM-081
    requirement: REQ-024
    text: Phase 5 fails when layer 3 rule has category other with empty justification
    tests:
      - TestPackVal_P5_CategoryOtherEmptyJustificationFails
  - id: CLM-082
    requirement: REQ-024
    text: Phase 5 fails when layer 3 rule has category other with missing justification
    tests:
      - TestPackVal_P5_CategoryOtherMissingJustificationFails
  - id: CLM-083
    requirement: REQ-024
    text: Phase 5 passes when layer 3 rule has category other with non-empty justification
    tests:
      - TestPackVal_P5_CategoryOtherWithJustificationPass

  # --- REQ-025: layer 3 input_scope and validator ---
  - id: CLM-084
    requirement: REQ-025
    text: Phase 5 fails when layer 3 rule is missing input_scope
    tests:
      - TestPackVal_P5_Layer3MissingInputScope
  - id: CLM-085
    requirement: REQ-025
    text: Phase 5 fails when layer 3 rule has invalid input_scope value
    tests:
      - TestPackVal_P5_Layer3InvalidInputScope
  - id: CLM-086
    requirement: REQ-025
    text: Phase 5 fails when layer 3 rule is missing validator field
    tests:
      - TestPackVal_P5_Layer3MissingValidator
  - id: CLM-087
    requirement: REQ-025
    text: Phase 5 fails when layer 3 validator file does not exist
    tests:
      - TestPackVal_P5_Layer3ValidatorFileNotFound
  - id: CLM-088
    requirement: REQ-025
    text: Phase 5 passes when layer 3 rule has valid input_scope and existing validator
    tests:
      - TestPackVal_P5_Layer3ValidInputScopeAndValidator

  # --- REQ-026: security bypass_attempt fixtures ---
  - id: CLM-089
    requirement: REQ-026
    text: Phase 6 fails when security rule claim has no bypass_attempt fixture
    tests:
      - TestPackVal_P6_SecurityClaimNoBypassAttempt
  - id: CLM-090
    requirement: REQ-026
    text: Phase 6 passes when security rule claim has at least one bypass_attempt fixture
    tests:
      - TestPackVal_P6_SecurityClaimWithBypassAttempt
  - id: CLM-091
    requirement: REQ-026
    text: Phase 6 normalizes plain string fixtures and object fixtures before checking
    tests:
      - TestPackVal_P6_NormalizeMixedFixtureFormats
  - id: CLM-092
    requirement: REQ-026
    text: Phase 6 does not require bypass_attempt on non-security rules
    tests:
      - TestPackVal_P6_NonSecurityNoBypassRequired

  # --- REQ-027: independent security claim fixtures ---
  - id: CLM-093
    requirement: REQ-027
    text: Phase 6 fails when security claims share a fixture file within the same rule
    tests:
      - TestPackVal_P6_SecurityClaimsSharedFixtureFails
  - id: CLM-094
    requirement: REQ-027
    text: Phase 6 passes when security claims have independent fixtures
    tests:
      - TestPackVal_P6_SecurityClaimsIndependentFixturesPass

  # --- REQ-028: early termination ---
  - id: CLM-095
    requirement: REQ-028
    text: Phase 2 is skipped when phase 1 fails
    tests:
      - TestPackVal_EarlyTermination_P1FailSkipsP2
  - id: CLM-096
    requirement: REQ-028
    text: Phase 3 is skipped when phase 2 fails
    tests:
      - TestPackVal_EarlyTermination_P2FailSkipsP3
  - id: CLM-097
    requirement: REQ-028
    text: Phases 4-6 are skipped when phase 3 fails
    tests:
      - TestPackVal_EarlyTermination_P3FailSkipsP4P5P6
  - id: CLM-098
    requirement: REQ-028
    text: Phase 5 is skipped when phase 4 fails
    tests:
      - TestPackVal_EarlyTermination_P4FailSkipsP5
  - id: CLM-099
    requirement: REQ-028
    text: Phase 6 is skipped when phase 5 fails
    tests:
      - TestPackVal_EarlyTermination_P5FailSkipsP6
  - id: CLM-100
    requirement: REQ-028
    text: All six phases run when all pass
    tests:
      - TestPackVal_AllPhasesRunOnSuccess

  # --- REQ-029: idempotency ---
  - id: CLM-101
    requirement: REQ-029
    text: Running validation twice produces identical pass/fail, errors, and warnings (excluding duration_ms)
    tests:
      - TestPackVal_Idempotent
  - id: CLM-102
    requirement: REQ-029
    text: Validation does not modify any files in the pack directory
    tests:
      - TestPackVal_NoSideEffects

  # --- REQ-030: JSON output format ---
  - id: CLM-103
    requirement: REQ-030
    text: Output is JSON by default containing status, pack, version, phases, errors, warnings
    tests:
      - TestPackVal_JSONOutputStructure
  - id: CLM-104
    requirement: REQ-030
    text: Every error includes phase, check, item, message, fix_hint, manifest_path
    tests:
      - TestPackVal_ErrorFieldsComplete
  - id: CLM-105
    requirement: REQ-030
    text: Phase status objects include phase name, status, checks count, and duration_ms
    tests:
      - TestPackVal_PhaseStatusFields
  - id: CLM-106
    requirement: REQ-030
    text: Skipped phases include reason field explaining which phase failed
    tests:
      - TestPackVal_SkippedPhaseReason

  # --- REQ-031: --format=text ---
  - id: CLM-107
    requirement: REQ-031
    text: Validation supports --format=text producing human-readable output
    tests:
      - TestPackVal_TextFormat
  - id: CLM-108
    requirement: REQ-031
    text: Default format is JSON when --format is not specified
    tests:
      - TestPackVal_DefaultFormatJSON

  # --- REQ-032: negative fixture engine-limitation hint ---
  - id: CLM-109
    requirement: REQ-032
    text: >
      Error for negative fixture that does not trigger its rule includes
      engine-limitation fix hint
    tests:
      - TestPackVal_P3_NegativeFixtureEngineLimitationHint

  # --- REQ-033: error ordering ---
  - id: CLM-110
    requirement: REQ-033
    text: Errors are ordered by phase first, then by manifest order within phase
    tests:
      - TestPackVal_ErrorOrdering

  # --- REQ-034: layer 3 sandbox ---
  - id: CLM-111
    requirement: REQ-034
    text: Phase 3 sandbox prevents filesystem writes outside pack directory
    tests:
      - TestPackVal_P3_SandboxBlocksFilesystemWrite
  - id: CLM-112
    requirement: REQ-034
    text: Phase 3 sandbox prevents network access
    tests:
      - TestPackVal_P3_SandboxBlocksNetwork
  - id: CLM-113
    requirement: REQ-034
    text: Phase 3 sandbox prevents environment variable access
    tests:
      - TestPackVal_P3_SandboxBlocksEnvVars
  - id: CLM-114
    requirement: REQ-034
    text: Phase 3 reports sandbox violation as hard error
    tests:
      - TestPackVal_P3_SandboxViolationIsHardError

contracts:
  - file: pkg/packval/pipeline.go
    provides:
      - name: Pipeline
        kind: type
        signature: "type Pipeline struct"
        notes: "Holds phase configuration, pack directory path, and mode (check vs test)"
      - name: NewPipeline
        kind: function
        signature: "func NewPipeline(packDir string, opts PipelineOptions) *Pipeline"
      - name: Run
        kind: method
        signature: "func (p *Pipeline) Run() *Result"
      - name: PipelineOptions
        kind: type
        signature: "type PipelineOptions struct"
        notes: "Mode (check/test), Format (json/text)"
    consumes: []

  - file: pkg/packval/result.go
    provides:
      - name: Result
        kind: type
        signature: "type Result struct"
        notes: "Status, Pack, Version, Phases []PhaseResult, Errors []ValidationError, Warnings []ValidationWarning"
      - name: PhaseResult
        kind: type
        signature: "type PhaseResult struct"
        notes: "Phase, Status, Checks, DurationMs, Reason (for skipped)"
      - name: ValidationError
        kind: type
        signature: "type ValidationError struct"
        notes: "Phase, Check, Rule, Claim, Message, FixHint, ManifestPath"
      - name: ValidationWarning
        kind: type
        signature: "type ValidationWarning struct"
        notes: "Phase, Check, Message, Files, FixHint"
    consumes: []

  - file: pkg/packval/phase1.go
    provides:
      - name: RunStructural
        kind: function
        signature: "func RunStructural(pack *PackManifest, packDir string) *PhaseResult"
    consumes: []

  - file: pkg/packval/phase2.go
    provides:
      - name: RunCoherence
        kind: function
        signature: "func RunCoherence(pack *PackManifest, packDir string) *PhaseResult"
    consumes: []

  - file: pkg/packval/phase3.go
    provides:
      - name: RunFixtures
        kind: function
        signature: "func RunFixtures(pack *PackManifest, packDir string, executor FixtureExecutor) *PhaseResult"
    consumes: []

  - file: pkg/packval/phase4.go
    provides:
      - name: RunArchetype
        kind: function
        signature: "func RunArchetype(pack *PackManifest) *PhaseResult"
    consumes: []

  - file: pkg/packval/phase5.go
    provides:
      - name: RunLayer
        kind: function
        signature: "func RunLayer(pack *PackManifest, packDir string) *PhaseResult"
    consumes: []

  - file: pkg/packval/phase6.go
    provides:
      - name: RunRiskClass
        kind: function
        signature: "func RunRiskClass(pack *PackManifest) *PhaseResult"
    consumes: []

  - file: pkg/packval/manifest.go
    provides:
      - name: PackManifest
        kind: type
        signature: "type PackManifest struct"
        notes: "Parsed pack.yml representation with Name, Version, Language, Archetype, Content, ToolConfig"
      - name: ParseManifest
        kind: function
        signature: "func ParseManifest(path string) (*PackManifest, error)"
    consumes: []

  - file: pkg/packval/executor.go
    provides:
      - name: FixtureExecutor
        kind: interface
        signature: "type FixtureExecutor interface"
        notes: "RunSemgrep, RunToolConfig, RunValidator, RunScaffoldTest methods"
      - name: DefaultExecutor
        kind: type
        signature: "type DefaultExecutor struct"
        notes: "Implements FixtureExecutor using real OS commands"
    consumes: []

  - file: pkg/packval/sandbox.go
    provides:
      - name: SandboxedRun
        kind: function
        signature: "func SandboxedRun(cmd string, args []string, packDir string) ([]byte, error)"
        notes: "Executes command in restricted environment (no network, no env vars, fs writes constrained)"
    consumes: []

  - file: cmd/backstop/pack_check.go
    provides:
      - name: packCheckCmd
        kind: variable
        signature: "var packCheckCmd *cobra.Command"
    consumes:
      - source: pkg/packval
        name: Pipeline
        kind: type
      - source: pkg/packval
        name: NewPipeline
        kind: function

  - file: cmd/backstop/pack_test_cmd.go
    provides:
      - name: packTestCmd
        kind: variable
        signature: "var packTestCmd *cobra.Command"
    consumes:
      - source: pkg/packval
        name: Pipeline
        kind: type
      - source: pkg/packval
        name: NewPipeline
        kind: function
---

# SPEC-014: Pack Validation Pipeline

## Overview

The pack validation pipeline verifies that a pack produced by an authoring agent is structurally correct, internally coherent, fixture-proven, archetype-compliant, layer-model-compliant, and risk-class-compliant. It is exposed as two CLI commands: `pack check` (manifest-only validation, fast) and `pack test` (full validation including external tool execution). Together they enable a tight agent-driven feedback loop: produce pack, validate, read errors, fix, re-validate.

The pipeline runs six phases in strict dependency order. Early termination ensures that if phase N fails, phases N+1 through 6 are skipped to avoid cascading noise. Output is machine-parseable JSON by default (ADR-0001) with every error including the phase, check name, offending item, message, fix hint, and manifest path. A `--format=text` flag provides human-readable output as a secondary format.

The three fixture execution paths in phase 3 cover the full rule taxonomy: semgrep `--test` for layer 1-2 declarative rules, temporary module environments for tool_config-dependent rules, and sandboxed process execution for layer 3 custom validators.

## Requirements

Requirements are defined in frontmatter. Claims are defined in frontmatter.

### Phase Summary

| Phase | Name | pack check | pack test | Key Checks |
|-------|------|-----------|----------|------------|
| 1 | structural | Yes | Yes | YAML parsing, required fields, enums, file existence, risk_class |
| 2 | coherence | Yes | Yes | Claims, fixtures, unique IDs, tool_config traceability, pairs_with |
| 3 | fixtures | No | Yes | semgrep --test, tool_config temp module, layer 3 sandbox, scaffolds, SDK |
| 4 | archetype | Yes | Yes | Code pack rules, bidirectional co-occurrence, enforcement pack no code |
| 5 | layer | Yes | Yes | Layer declaration, category on layer 3 only, justification, input_scope |
| 6 | risk_class | Yes | Yes | Bypass-attempt fixtures, independent security claim fixtures |

### Archetype Constraint Matrix

| Archetype | May declare rules | May declare scaffolds | May declare sdk | Bidirectional pairs_with required |
|-----------|------------------|-----------------------|-----------------|----------------------------------|
| enforcement | Yes (required) | No (hard error) | No (hard error) | No |
| code | Yes (required) | Yes | Yes | Yes (every rule must pair) |

### Layer-Category Matrix

| Layer | category field | justification field |
|-------|---------------|-------------------|
| 1 | Must NOT appear (hard error if present) | N/A |
| 2 | Must NOT appear (hard error if present) | N/A |
| 3 | Required (presence, structural, or other) | Required only when category is other |

### Risk Class Rules

| risk_class | bypass_attempt required | Independent claim fixtures required |
|------------|------------------------|-------------------------------------|
| security | Yes (at least one per claim) | Yes (no shared fixtures across claims) |
| correctness | No | No |
| style | No | No |
| perf | No | No |

## Implementation

### Package Structure

```
pkg/packval/
  pipeline.go    — Pipeline orchestrator: mode selection, phase ordering, early termination
  result.go      — Result, PhaseResult, ValidationError, ValidationWarning types
  manifest.go    — PackManifest type and YAML parser
  phase1.go      — Phase 1: structural validation
  phase2.go      — Phase 2: coherence validation
  phase3.go      — Phase 3: fixture execution (dispatches to executor)
  phase4.go      — Phase 4: archetype enforcement
  phase5.go      — Phase 5: layer enforcement
  phase6.go      — Phase 6: risk class enforcement
  executor.go    — FixtureExecutor interface and DefaultExecutor
  sandbox.go     — SandboxedRun for layer 3 validators
  format.go      — JSON (default) and text output formatters

cmd/backstop/
  pack_check.go     — pack check CLI command (thin adapter)
  pack_test_cmd.go  — pack test CLI command (thin adapter)
```

### Pipeline Orchestration

The Pipeline.Run() method executes phases in order. For `pack check` mode, it runs phases 1, 2, 4, 5, 6 (skipping phase 3). For `pack test` mode, it runs all phases 1 through 6. After each phase, if the phase status is "fail", subsequent phases are marked "skipped" with a reason field indicating which phase failed, and Run returns immediately.

### Phase 1 — Structural Validation

1. Parse pack.yml as YAML; reject on parse error
2. Check required fields: name, version, language, archetype, content
3. Validate field values: archetype in {enforcement, code}, version matches semver, language in supported set {go}
4. Check risk_class on every rule (ruleset and standalone tool_config): must be in {security, correctness, style, perf}
5. For layer 2 rules: verify rule field points to existing file
6. For all referenced file paths: verify existence on disk, EXCEPT tool_config.file (consumer-side)
7. Validate all content type declarations

### Phase 2 — Coherence Validation

1. Verify every rule has at least one claim
2. Verify every claim has both positive and negative fixture lists (at least one each)
3. Verify every fixture file exists and is non-empty
4. Verify claim ID uniqueness within the pack
5. Verify rule ID uniqueness spanning ruleset rules and standalone tool_config entries
6. Verify tool_config entries with own rule ID have claims and fixtures
7. Check pairs_with.rules entries resolve to existing rule IDs (warning on dangling)
8. Scan for orphan fixture files not referenced by any claim (warning)

### Phase 3 — Fixture Execution

Three execution paths dispatched by the FixtureExecutor interface:

**Semgrep path (layer 1-2 rules):**
1. Read semgrep rule file, verify rule ID matches pack rule ID
2. Run semgrep --test: positive fixtures must not trigger, all negative fixtures must trigger
3. On negative fixture failure, include engine-limitation fix hint

**Tool_config path:**
1. Create temporary module directory, copy pack's go.mod
2. Copy fixture file into temp module
3. Run configured tool against temp module
4. Positive must pass clean, negative must trigger diagnostic
5. Clean up temp directory

**Layer 3 validator path:**
1. Invoke validator.sh with fixture path (file for single-file, directory for multi-file)
2. Check exit code: 0 = pass, non-zero = fail
3. Execute in sandbox (SandboxedRun)

**Scaffold validation:**
- Complete scaffolds: render with sample_config, run test_command
- Skeleton scaffolds: structural checks only (directory, files, test names), no test execution

**SDK validation:**
- Verify provides surface is declared at manifest level

**Pre-check:**
- Run go mod tidy for Go packs before any fixture execution

### Phase 4 — Archetype Enforcement

1. If archetype is code: verify rules are declared, every scaffold has enforcement rule, every rule has pairs_with
2. If archetype is enforcement: verify no sdk or scaffolds declared

### Phase 5 — Layer Enforcement

1. Verify every rule declares layer (1, 2, or 3)
2. Verify risk_class is valid on all rules
3. Verify category is present ONLY on layer 3 rules (hard error on layer 1 or 2)
4. Layer 3 with category presence or structural: auto-accept, no justification required
5. Layer 3 with category other: require non-empty justification
6. Layer 3: require input_scope (single-file or multi-file) and validator pointing to executable file

### Phase 6 — Risk Class Enforcement

1. For security-class rules: verify at least one bypass_attempt negative fixture per claim
2. Normalize mixed fixture formats (plain strings and objects with path/bypass_attempt)
3. For security-class rules: verify independent fixtures per claim (no sharing)

### Output Format

JSON output structure:
```json
{
  "status": "pass|fail",
  "pack": "<pack-name>",
  "version": "<pack-version>",
  "phases": [
    {"phase": "<name>", "status": "pass|fail|skipped", "checks": N, "duration_ms": N, "reason": "..."}
  ],
  "errors": [
    {"phase": "...", "check": "...", "rule": "...", "claim": "...", "message": "...", "fix_hint": "...", "manifest_path": "..."}
  ],
  "warnings": [
    {"phase": "...", "check": "...", "message": "...", "files": [...], "fix_hint": "..."}
  ]
}
```

## Verification

Verification config is defined in frontmatter. Claims are defined in frontmatter. Test command targets both pkg/packval and cmd/backstop for integration coverage.

### Test Strategy

- **Unit tests** in pkg/packval: each phase function tested in isolation with crafted manifests and fixture directories built in t.TempDir()
- **Integration tests** in cmd/backstop: end-to-end pack check and pack test commands against fixture packs
- **Fixture executor tests**: use a mock FixtureExecutor to test phase 3 dispatch logic without requiring semgrep/go toolchain
- **Sandbox tests**: verify SandboxedRun blocks filesystem writes, network, and env var access

## Sharp Edges

1. **Semgrep annotation format dependency.** Phase 3 relies on semgrep `--test` mode, which requires engine-native annotations (`// ruleid:` and `// ok:` comments). If semgrep changes its test format in a future version, fixture validation breaks. The spec does not pin a semgrep version — pack authors must ensure annotations match their semgrep version.

2. **tool_config temp module may diverge from real consumer environment.** The temp module copies the pack's go.mod but not the full vendor tree or replace directives. A fixture that passes in the temp module could fail in a real project with different dependency versions or replace directives. This is a known fidelity gap.

3. **Layer 3 sandbox is OS-dependent.** SandboxedRun uses OS-level process isolation (not containers). The exact mechanism differs between macOS (sandbox-exec) and Linux (seccomp/namespaces). Sandbox enforcement on unsupported platforms may need to degrade to a warning or skip, which weakens the guarantee.

4. **Idempotency excludes go mod tidy side effects.** REQ-029 says "no files modified" but REQ-014 says "run go mod tidy in the pack directory." If go mod tidy modifies go.sum, the pack directory is technically mutated. The implementation must run go mod tidy in a copy or accept that this is an exception to the idempotency rule. This tension is unresolved in the bundle.

5. **category:other justification quality is not validated.** Phase 5 checks that justification is non-empty but cannot assess whether the justification is genuine. A single-character justification passes mechanical validation. LLM review at catalog submission time is the intended quality gate, but that is out of scope for this spec.

6. **Orphan fixture detection depends on fixture directory convention.** Phase 2 orphan detection scans "the fixture directory" but the bundle does not specify an explicit fixtures root. If fixtures are scattered across arbitrary paths, orphan detection may miss files or produce false positives.

7. **pack test file naming collision with Go test framework.** The CLI command is named `pack test` but the Go file cannot be named `pack_test.go` (Go treats `_test.go` files as test files). The contract uses `pack_test_cmd.go` to avoid this collision.

## Review Questions

1. Should go mod tidy run in the original pack directory (mutating it, violating REQ-029 idempotency) or in a temp copy (adding complexity and possibly diverging from the real environment)?

2. How should sandbox enforcement degrade on platforms where OS-level process isolation is not available? Should it fail open (warning) or fail closed (hard error)?

3. The FixtureExecutor interface makes phase 3 testable via mocks, but are there integration-level risks where the mock diverges from real semgrep/tool behavior that unit tests would miss?

4. REQ-007 enforces rule ID uniqueness across ruleset and tool_config, but what about rule IDs that collide with claim IDs? Is the ID namespace shared or separate?

5. Skeleton scaffold "expected files present" check (REQ-017) — who defines the expected file list? Is it part of the manifest schema or is it convention-based?

6. When pack check runs phases 4/5/6 after skipping phase 3, could there be false passes for checks that depend on fixture execution results? For example, phase 6 checks bypass_attempt fixtures exist, but does not verify they actually work.

## References

- **BUNDLE-005** — Pack Validation bundle (source of all 34 requirements)
- **BUNDLE-004** — Pack Manifest schema (defines what a valid pack looks like)
- **ADR-0001** — Agent-first discipline (JSON output, machine-parseable errors)
- **SPEC-012** — Go standards pack (first validation target)
