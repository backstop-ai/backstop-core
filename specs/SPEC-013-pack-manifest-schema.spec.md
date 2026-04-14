---
title: "SPEC-013: Pack Manifest Schema"
number: SPEC-013
created: "2026-04-08"
status: draft
schema_version: spec/v1
spec_version: 1.0.0

implementation:
  summary: >
    Define and implement the pack.yml manifest schema: types for parsing and
    representing pack manifests, field validation rules, content type
    declarations, archetype constraints (enforcement vs code), three-layer
    enforcement model, scaffold tiers, fixture declarations, versioning fields,
    rule ID conventions, tool_config declarations, and the coordinate reference
    format. The package provides ParseManifest for loading pack.yml into typed
    Go structs and ValidateManifest for structural field-level validation
    (required fields, enums, format constraints). Coherence checks (reference
    resolution, co-occurrence enforcement) are deferred to BUNDLE-005. This
    spec covers what a valid pack.yml looks like; BUNDLE-005 covers whether
    its internal references are consistent.
  package: pkg/pack

verification:
  level: unit
  test_command: go test ./pkg/pack/ -race -coverprofile=cover.out
  coverage_threshold: 90

requirements:
  - id: REQ-001
    text: >
      The pack manifest file must be named pack.yml. ParseManifest must parse
      it and extract the required top-level fields: name, version, language,
      archetype, description, and content. A manifest missing any required
      field must return a validation error identifying the missing field.
    supports: pack-manifest-authoring:REQ-001

  - id: REQ-002
    text: >
      Pack names must follow the two-part org/pack-name format. Each part
      accepts alphanumeric characters and hyphens. Names are normalized to
      lowercase internally for namespacing and matching. Original casing is
      preserved in the Name field. A name that does not contain exactly one
      slash separator, or contains characters outside [a-zA-Z0-9-], must be
      rejected.
    supports: pack-manifest-authoring:REQ-002

  - id: REQ-003
    text: >
      Every pack must declare exactly one archetype field with value
      "enforcement" or "code". Any other value or a missing archetype field
      must be rejected.
    supports: pack-manifest-authoring:REQ-004

  - id: REQ-004
    text: >
      The content block must declare typed content using only the allowed
      types: ruleset, scaffolds, sdk, contracts, test_patterns, ast_checks,
      rubrics. An enforcement pack must not declare scaffolds or sdk in its
      content block. An enforcement pack with only a ruleset (no other content
      types) is valid.
    supports: pack-manifest-authoring:REQ-006

  - id: REQ-005
    text: >
      Every rule must declare a risk_class field with one of four allowed
      values: security, correctness, style, perf. A rule with a missing or
      invalid risk_class must be rejected. This applies to all rules
      regardless of layer, including standalone tool_config rules.
    supports: pack-manifest-authoring:REQ-007

  - id: REQ-006
    text: >
      Rules must be organized into three enforcement layers. Each rule must
      declare a layer field with value 1, 2, or 3. Layer 1 is built-in tool
      rules (highest trust). Layer 2 is custom declarative semgrep rules
      (medium trust). Layer 3 is custom validators (lowest trust). A rule
      with a missing or invalid layer value must be rejected.
    supports: pack-manifest-authoring:REQ-008

  - id: REQ-007
    text: >
      Layer 2 rules must declare both a rule field (pointing to the compiled
      semgrep YAML file path) and a standard field (filepath to .standard.md
      or inline string). A layer 2 rule missing either field must be rejected.
      Layer 1 and layer 3 rules must not declare the rule field.
    supports: pack-manifest-authoring:REQ-009

  - id: REQ-008
    text: >
      Layer 3 rules must declare a category field with one of three values:
      presence, structural, or other. Categories presence and structural
      require no justification field. Category other requires a mandatory
      justification field; omitting justification when category is other must
      be rejected. A layer 3 rule missing category must be rejected. Layer 1
      and layer 2 rules must not declare category.
    supports: pack-manifest-authoring:REQ-010

  - id: REQ-009
    text: >
      Layer 3 rules must declare an input_scope field with value single-file
      or multi-file, and a validator field pointing to the validator script
      path. A layer 3 rule missing input_scope or validator must be rejected.
      Layer 1 and layer 2 rules must not declare input_scope or validator.
    supports: pack-manifest-authoring:REQ-011

  - id: REQ-010
    text: >
      Every claim must declare at least one positive fixture (known-good) and
      at least one negative fixture (known-bad). Fixture paths are relative
      to the pack root and declared inline on claims. A claim with zero
      positive fixtures or zero negative fixtures must be rejected.
    supports: pack-manifest-authoring:REQ-013

  - id: REQ-011
    text: >
      Security-class rules (risk_class: security) must include at least one
      negative fixture with bypass_attempt: true, in addition to standard
      negative fixtures. A security-class rule where no negative fixture
      has bypass_attempt: true must be rejected.
    supports: pack-manifest-authoring:REQ-014

  - id: REQ-012
    text: >
      Rule IDs must use lowercase kebab-case format matching the pattern
      ^[a-z][a-z0-9]*(-[a-z0-9]+)*$. On load, rule IDs are namespaced with
      the pack name using slash delimiters: pack-name/rule-id. A rule ID
      not matching the pattern must be rejected.
    supports: pack-manifest-authoring:REQ-017

  - id: REQ-013
    text: >
      Pack versioning uses three levels. Pack version (required, semver
      format) covers the whole artifact. Ruleset version (optional, defaults
      to pack version for enforcement packs if omitted) covers all rules as
      a cohort. Item version (on scaffolds and SDKs individually) enables
      spec and plan references. All version fields that are present must
      conform to semver format. A pack version that does not match semver
      must be rejected.
    supports: pack-manifest-authoring:REQ-018

  - id: REQ-014
    text: >
      Scaffolds must declare a tier field with value complete or skeleton.
      Any other value or a missing tier must be rejected.
    supports: pack-manifest-authoring:REQ-019

  - id: REQ-015
    text: >
      Every scaffold must declare a test_command field specifying how to run
      its tests. A scaffold missing test_command must be rejected.
    supports: pack-manifest-authoring:REQ-020

  - id: REQ-016
    text: >
      Scaffolds must declare use_when (non-empty list of scenario strings),
      assumes (non-empty list of precondition strings), and pairs_with (a
      single object with optional keys: rules, scaffolds, sdk). sample_config
      values must be flat key-value string pairs matching environment variable
      naming conventions (uppercase, underscores). A scaffold missing use_when
      or assumes must be rejected. pairs_with is required on scaffolds.
    supports: pack-manifest-authoring:REQ-021

  - id: REQ-017
    text: >
      The sdk content type is a single optional entry with required fields:
      module (string), version (string), and provides (non-empty list of
      strings representing the public surface). A pack may declare at most
      one SDK. An sdk entry missing module, version, or provides, or an
      sdk with an empty provides list, must be rejected.
    supports: pack-manifest-authoring:REQ-022

  - id: REQ-018
    text: >
      tool_config entries must declare tool (string) and file (string,
      consumer-side target path). Standalone tool_config entries (those with
      an id field) are layer 1 rules and must additionally declare risk_class
      and claims with fixtures. Supporting tool_config entries (those with
      required_by) must not declare id, risk_class, or claims. A tool_config
      entry must be either standalone (has id) or supporting (has required_by),
      not both and not neither.
    supports: pack-manifest-authoring:REQ-023

  - id: REQ-019
    text: >
      Every tool_config entry must be traceable to at least one rule. A
      standalone tool_config is its own rule (via id). A supporting
      tool_config must reference at least one rule via required_by. A
      tool_config entry that is neither standalone nor supporting must be
      rejected.
    supports: pack-manifest-authoring:REQ-024

  - id: REQ-020
    text: >
      The standard field on rules is optional and accepts a filepath string
      (pointing to a .standard.md file) or an inline string. For layer 2
      rules, the standard field is required (per REQ-007). For other layers,
      standard is optional and may be omitted.
    supports: pack-manifest-authoring:REQ-025

  - id: REQ-021
    text: >
      Claim IDs must be unique within the pack. Recommended format is
      rule-id-clm-NNN (e.g., err-001-clm-001). Duplicate claim IDs within
      a single pack manifest must be rejected.
    supports: pack-manifest-authoring:REQ-027

  - id: REQ-022
    text: >
      Each pack targets exactly one language via the required language field.
      The language field must be a non-empty string. Cross-language packs are
      not supported.
    supports: pack-manifest-authoring:REQ-030

  - id: REQ-023
    text: >
      The manifest must support versioned coordinate references from specs
      and plans using the format pack-name@pack-version:item-name@item-version.
      The ParseCoordinate function must parse this format and return the
      constituent parts. An invalid coordinate format must return an error.
    supports: pack-manifest-authoring:REQ-031

  - id: REQ-024
    text: >
      The embedded Go standards pack must be loadable through the same
      ParseManifest path as third-party packs. No special-case code paths
      for embedded packs.
    supports: pack-manifest-authoring:REQ-032

  - id: REQ-025
    text: >
      Scaffolds are copy-once templates. The manifest must represent scaffolds
      with a path field (directory within the pack), and the scaffold struct
      must not imply update semantics. SDKs are native-language module
      references tracked in the manifest; the SDK struct contains module,
      version, and provides but no distribution mechanism.
    supports: pack-manifest-authoring:REQ-033

  - id: REQ-026
    text: >
      A pack's rules may reference its own SDK surface via pairs_with.sdk on
      the rule or scaffold. The pairs_with field is a single object with
      optional keys rules, scaffolds, sdk. Structural parsing of pairs_with
      must accept all three keys.
    supports: pack-manifest-authoring:REQ-034

  - id: REQ-027
    text: >
      Fixture directories must use lowercase naming matching rule IDs (e.g.,
      fixtures/rules/err-001/). This is consistent with the rule ID convention
      of lowercase kebab-case. Fixture directory names that do not match their
      rule ID in lowercase must be flagged.
    supports: pack-manifest-authoring:REQ-016

  - id: REQ-028
    text: >
      Semgrep fixtures must include engine-native annotations (// ruleid:
      and // ok: comments). Claim-to-fixture mapping is declared exclusively
      in pack.yml, not in fixture file metadata. Fixture entries in the
      manifest may be a plain string path or an object with path and optional
      bypass_attempt boolean.
    supports: pack-manifest-authoring:REQ-015

  - id: REQ-029
    text: >
      The canonical pack directory layout is: pack.yml (required), go.mod
      (required), rules/ (required if layer 2 rules exist), fixtures/rules/
      (required, one lowercase subdir per rule ID), standards/ (optional),
      scaffolds/ (required if archetype is code), validators/ (required if
      layer 3 rules exist). ValidateManifest must check that the manifest
      content is consistent with these directory expectations based on
      archetype and declared layers.
    supports: pack-manifest-authoring:REQ-003

  - id: REQ-030
    text: >
      Layer 3 validators must run in process isolation: separate process, no
      network access, no filesystem writes outside pack directory, no
      environment variable access. The manifest types must represent
      input_scope and validator path so that the runtime can enforce isolation.
    supports: pack-manifest-authoring:REQ-012

  - id: REQ-031
    text: >
      pack check runs structural verification and pack test runs fixture
      execution. The manifest schema types must support both workflows: pack
      check validates manifest structure, field presence, enum values,
      archetype constraints, and layer-specific field requirements. pack test
      additionally executes fixtures. The types in this spec support the
      schema that both commands validate against.
    supports: pack-manifest-authoring:REQ-028

  - id: REQ-032
    text: >
      pack try <project-path> runs the pack's rules against a real codebase
      for author exploration. The manifest types must include sufficient
      metadata (rule paths, tool configs, validator paths) to enable this
      command.
    supports: pack-manifest-authoring:REQ-029

  - id: REQ-033
    text: >
      Code packs must enforce bidirectional co-occurrence: every scaffold
      must have at least one enforcement rule via pairs_with.rules, AND every
      rule must reference at least one scaffold or SDK via pairs_with. A code
      pack scaffold without a paired rule, or a code pack rule without a
      paired scaffold or SDK, is a validation error. Enforcement packs must
      not declare scaffolds or sdk.
    supports: pack-manifest-authoring:REQ-005

  - id: REQ-034
    text: >
      All pack archetypes require mechanical proof via fixtures at their
      tier's expected completeness level. No archetype is exempt from fixture
      requirements. Every rule (in both enforcement and code packs) must have
      at least one claim, and every claim must have fixtures.
    supports: pack-manifest-authoring:REQ-035

  - id: REQ-035
    text: >
      Pack directories must include a go.mod (or language equivalent) for
      fixture dependencies. The language field determines which dependency
      file is expected.
    supports: pack-manifest-authoring:REQ-026

claims:
  # REQ-001: Required top-level fields
  - id: CLM-001
    requirement: REQ-001
    text: ParseManifest parses a valid pack.yml and populates all top-level fields
    tests:
      - TestParseManifest_ValidEnforcementPack
      - TestParseManifest_ValidCodePack

  - id: CLM-002
    requirement: REQ-001
    text: ParseManifest rejects a manifest missing the name field
    tests:
      - TestParseManifest_MissingName

  - id: CLM-003
    requirement: REQ-001
    text: ParseManifest rejects a manifest missing the version field
    tests:
      - TestParseManifest_MissingVersion

  - id: CLM-004
    requirement: REQ-001
    text: ParseManifest rejects a manifest missing the language field
    tests:
      - TestParseManifest_MissingLanguage

  - id: CLM-005
    requirement: REQ-001
    text: ParseManifest rejects a manifest missing the archetype field
    tests:
      - TestParseManifest_MissingArchetype

  - id: CLM-006
    requirement: REQ-001
    text: ParseManifest rejects a manifest missing the description field
    tests:
      - TestParseManifest_MissingDescription

  - id: CLM-007
    requirement: REQ-001
    text: ParseManifest rejects a manifest missing the content block
    tests:
      - TestParseManifest_MissingContent

  # REQ-002: Pack naming
  - id: CLM-008
    requirement: REQ-002
    text: Valid two-part org/pack-name is accepted
    tests:
      - TestValidateName_ValidTwoPart

  - id: CLM-009
    requirement: REQ-002
    text: Name without slash separator is rejected
    tests:
      - TestValidateName_NoSlash

  - id: CLM-010
    requirement: REQ-002
    text: Name with more than one slash is rejected
    tests:
      - TestValidateName_MultipleSlashes

  - id: CLM-011
    requirement: REQ-002
    text: Name with invalid characters is rejected
    tests:
      - TestValidateName_InvalidChars

  - id: CLM-012
    requirement: REQ-002
    text: Name is normalized to lowercase internally
    tests:
      - TestValidateName_NormalizedToLowercase

  # REQ-003: Archetype enum — enforcement passes, code passes, other fails
  - id: CLM-013
    requirement: REQ-003
    text: Archetype enforcement is accepted
    tests:
      - TestValidateArchetype_Enforcement

  - id: CLM-014
    requirement: REQ-003
    text: Archetype code is accepted
    tests:
      - TestValidateArchetype_Code

  - id: CLM-015
    requirement: REQ-003
    text: Archetype with invalid value is rejected
    tests:
      - TestValidateArchetype_Invalid

  - id: CLM-016
    requirement: REQ-003
    text: Missing archetype field is rejected
    tests:
      - TestValidateArchetype_Missing

  # REQ-004: Content types and enforcement pack constraints
  - id: CLM-017
    requirement: REQ-004
    text: Enforcement pack with only ruleset is valid
    tests:
      - TestValidateContent_EnforcementRulesetOnly

  - id: CLM-018
    requirement: REQ-004
    text: Enforcement pack declaring scaffolds is rejected
    tests:
      - TestValidateContent_EnforcementWithScaffolds

  - id: CLM-019
    requirement: REQ-004
    text: Enforcement pack declaring sdk is rejected
    tests:
      - TestValidateContent_EnforcementWithSDK

  - id: CLM-020
    requirement: REQ-004
    text: Code pack with ruleset and scaffolds is valid
    tests:
      - TestValidateContent_CodeWithRulesetAndScaffolds

  - id: CLM-021
    requirement: REQ-004
    text: Content block with unknown content type key is rejected
    tests:
      - TestValidateContent_UnknownContentType

  # REQ-005: risk_class enum — all four values pass, invalid fails, missing fails
  - id: CLM-022
    requirement: REQ-005
    text: risk_class security is accepted
    tests:
      - TestValidateRiskClass_Security

  - id: CLM-023
    requirement: REQ-005
    text: risk_class correctness is accepted
    tests:
      - TestValidateRiskClass_Correctness

  - id: CLM-024
    requirement: REQ-005
    text: risk_class style is accepted
    tests:
      - TestValidateRiskClass_Style

  - id: CLM-025
    requirement: REQ-005
    text: risk_class perf is accepted
    tests:
      - TestValidateRiskClass_Perf

  - id: CLM-026
    requirement: REQ-005
    text: risk_class with invalid value is rejected
    tests:
      - TestValidateRiskClass_Invalid

  - id: CLM-027
    requirement: REQ-005
    text: Missing risk_class is rejected
    tests:
      - TestValidateRiskClass_Missing

  # REQ-006: Layer enum — 1 passes, 2 passes, 3 passes, 0 fails, 4 fails, missing fails
  - id: CLM-028
    requirement: REQ-006
    text: Layer 1 is accepted
    tests:
      - TestValidateLayer_One

  - id: CLM-029
    requirement: REQ-006
    text: Layer 2 is accepted
    tests:
      - TestValidateLayer_Two

  - id: CLM-030
    requirement: REQ-006
    text: Layer 3 is accepted
    tests:
      - TestValidateLayer_Three

  - id: CLM-031
    requirement: REQ-006
    text: Layer 0 is rejected
    tests:
      - TestValidateLayer_Zero

  - id: CLM-032
    requirement: REQ-006
    text: Layer 4 is rejected
    tests:
      - TestValidateLayer_Four

  - id: CLM-033
    requirement: REQ-006
    text: Missing layer is rejected
    tests:
      - TestValidateLayer_Missing

  # REQ-007: Layer 2 requires rule + standard; layers 1 and 3 must not have rule field
  - id: CLM-034
    requirement: REQ-007
    text: Layer 2 rule with both rule and standard fields is accepted
    tests:
      - TestValidateLayer2_BothFields

  - id: CLM-035
    requirement: REQ-007
    text: Layer 2 rule missing rule field is rejected
    tests:
      - TestValidateLayer2_MissingRuleField

  - id: CLM-036
    requirement: REQ-007
    text: Layer 2 rule missing standard field is rejected
    tests:
      - TestValidateLayer2_MissingStandard

  - id: CLM-037
    requirement: REQ-007
    text: Layer 1 rule declaring rule field is rejected
    tests:
      - TestValidateLayer1_WithRuleField

  - id: CLM-038
    requirement: REQ-007
    text: Layer 3 rule declaring rule field is rejected
    tests:
      - TestValidateLayer3_WithRuleField

  # REQ-008: Layer 3 category — presence passes, structural passes, other passes with justification, other fails without justification
  - id: CLM-039
    requirement: REQ-008
    text: Layer 3 rule with category presence is accepted without justification
    tests:
      - TestValidateLayer3Category_Presence

  - id: CLM-040
    requirement: REQ-008
    text: Layer 3 rule with category structural is accepted without justification
    tests:
      - TestValidateLayer3Category_Structural

  - id: CLM-041
    requirement: REQ-008
    text: Layer 3 rule with category other and justification is accepted
    tests:
      - TestValidateLayer3Category_OtherWithJustification

  - id: CLM-042
    requirement: REQ-008
    text: Layer 3 rule with category other missing justification is rejected
    tests:
      - TestValidateLayer3Category_OtherMissingJustification

  - id: CLM-043
    requirement: REQ-008
    text: Layer 3 rule missing category is rejected
    tests:
      - TestValidateLayer3Category_Missing

  - id: CLM-044
    requirement: REQ-008
    text: Layer 3 rule with invalid category is rejected
    tests:
      - TestValidateLayer3Category_Invalid

  - id: CLM-045
    requirement: REQ-008
    text: Layer 1 rule declaring category is rejected
    tests:
      - TestValidateLayer1_WithCategory

  - id: CLM-046
    requirement: REQ-008
    text: Layer 2 rule declaring category is rejected
    tests:
      - TestValidateLayer2_WithCategory

  # REQ-009: Layer 3 requires input_scope + validator; layers 1 and 2 must not have them
  - id: CLM-047
    requirement: REQ-009
    text: Layer 3 rule with input_scope single-file and validator is accepted
    tests:
      - TestValidateLayer3_SingleFileWithValidator

  - id: CLM-048
    requirement: REQ-009
    text: Layer 3 rule with input_scope multi-file and validator is accepted
    tests:
      - TestValidateLayer3_MultiFileWithValidator

  - id: CLM-049
    requirement: REQ-009
    text: Layer 3 rule missing input_scope is rejected
    tests:
      - TestValidateLayer3_MissingInputScope

  - id: CLM-050
    requirement: REQ-009
    text: Layer 3 rule missing validator is rejected
    tests:
      - TestValidateLayer3_MissingValidator

  - id: CLM-051
    requirement: REQ-009
    text: Layer 3 rule with invalid input_scope is rejected
    tests:
      - TestValidateLayer3_InvalidInputScope

  - id: CLM-052
    requirement: REQ-009
    text: Layer 1 rule declaring input_scope is rejected
    tests:
      - TestValidateLayer1_WithInputScope

  - id: CLM-053
    requirement: REQ-009
    text: Layer 2 rule declaring input_scope is rejected
    tests:
      - TestValidateLayer2_WithInputScope

  - id: CLM-054
    requirement: REQ-009
    text: Layer 1 rule declaring validator is rejected
    tests:
      - TestValidateLayer1_WithValidator

  - id: CLM-055
    requirement: REQ-009
    text: Layer 2 rule declaring validator is rejected
    tests:
      - TestValidateLayer2_WithValidator

  # REQ-010: Fixture presence
  - id: CLM-056
    requirement: REQ-010
    text: Claim with both positive and negative fixtures is accepted
    tests:
      - TestValidateFixtures_BothPresent

  - id: CLM-057
    requirement: REQ-010
    text: Claim with zero positive fixtures is rejected
    tests:
      - TestValidateFixtures_NoPositive

  - id: CLM-058
    requirement: REQ-010
    text: Claim with zero negative fixtures is rejected
    tests:
      - TestValidateFixtures_NoNegative

  # REQ-011: Security bypass_attempt
  - id: CLM-059
    requirement: REQ-011
    text: Security-class rule with at least one bypass_attempt fixture is accepted
    tests:
      - TestValidateSecurityFixtures_WithBypass

  - id: CLM-060
    requirement: REQ-011
    text: Security-class rule with no bypass_attempt fixture is rejected
    tests:
      - TestValidateSecurityFixtures_NoBypass

  - id: CLM-061
    requirement: REQ-011
    text: Non-security rule without bypass_attempt fixture is accepted
    tests:
      - TestValidateSecurityFixtures_NonSecurityNoBypass

  # REQ-012: Rule ID format
  - id: CLM-062
    requirement: REQ-012
    text: Lowercase kebab-case rule ID is accepted
    tests:
      - TestValidateRuleID_ValidKebab

  - id: CLM-063
    requirement: REQ-012
    text: Rule ID with uppercase characters is rejected
    tests:
      - TestValidateRuleID_Uppercase

  - id: CLM-064
    requirement: REQ-012
    text: Rule ID with underscores is rejected
    tests:
      - TestValidateRuleID_Underscores

  - id: CLM-065
    requirement: REQ-012
    text: Rule ID is correctly namespaced with pack name on load
    tests:
      - TestNamespaceRuleID_Correct

  # REQ-013: Versioning
  - id: CLM-066
    requirement: REQ-013
    text: Valid semver pack version is accepted
    tests:
      - TestValidateVersion_ValidSemver

  - id: CLM-067
    requirement: REQ-013
    text: Invalid pack version format is rejected
    tests:
      - TestValidateVersion_InvalidFormat

  - id: CLM-068
    requirement: REQ-013
    text: Enforcement pack with omitted ruleset version defaults to pack version
    tests:
      - TestValidateVersion_RulesetDefaultsToPackVersion

  - id: CLM-069
    requirement: REQ-013
    text: Explicit ruleset version overrides pack version
    tests:
      - TestValidateVersion_ExplicitRulesetVersion

  # REQ-014: Scaffold tier — complete passes, skeleton passes, other fails, missing fails
  - id: CLM-070
    requirement: REQ-014
    text: Scaffold tier complete is accepted
    tests:
      - TestValidateScaffoldTier_Complete

  - id: CLM-071
    requirement: REQ-014
    text: Scaffold tier skeleton is accepted
    tests:
      - TestValidateScaffoldTier_Skeleton

  - id: CLM-072
    requirement: REQ-014
    text: Scaffold tier with invalid value is rejected
    tests:
      - TestValidateScaffoldTier_Invalid

  - id: CLM-073
    requirement: REQ-014
    text: Scaffold missing tier is rejected
    tests:
      - TestValidateScaffoldTier_Missing

  # REQ-015: Scaffold test_command
  - id: CLM-074
    requirement: REQ-015
    text: Scaffold with test_command is accepted
    tests:
      - TestValidateScaffold_WithTestCommand

  - id: CLM-075
    requirement: REQ-015
    text: Scaffold missing test_command is rejected
    tests:
      - TestValidateScaffold_MissingTestCommand

  # REQ-016: Scaffold use_when, assumes, pairs_with, sample_config
  - id: CLM-076
    requirement: REQ-016
    text: Scaffold with use_when, assumes, and pairs_with is accepted
    tests:
      - TestValidateScaffold_AllFieldsPresent

  - id: CLM-077
    requirement: REQ-016
    text: Scaffold missing use_when is rejected
    tests:
      - TestValidateScaffold_MissingUseWhen

  - id: CLM-078
    requirement: REQ-016
    text: Scaffold missing assumes is rejected
    tests:
      - TestValidateScaffold_MissingAssumes

  - id: CLM-079
    requirement: REQ-016
    text: Scaffold missing pairs_with is rejected
    tests:
      - TestValidateScaffold_MissingPairsWith

  - id: CLM-080
    requirement: REQ-016
    text: Scaffold with empty use_when list is rejected
    tests:
      - TestValidateScaffold_EmptyUseWhen

  - id: CLM-081
    requirement: REQ-016
    text: Scaffold sample_config with non-string values is rejected
    tests:
      - TestValidateScaffold_SampleConfigNonString

  - id: CLM-082
    requirement: REQ-016
    text: Scaffold pairs_with parses rules, scaffolds, and sdk keys
    tests:
      - TestParseScaffold_PairsWithAllKeys

  # REQ-017: SDK content type
  - id: CLM-083
    requirement: REQ-017
    text: Valid SDK with module, version, and provides is accepted
    tests:
      - TestValidateSDK_Valid

  - id: CLM-084
    requirement: REQ-017
    text: SDK missing module is rejected
    tests:
      - TestValidateSDK_MissingModule

  - id: CLM-085
    requirement: REQ-017
    text: SDK missing version is rejected
    tests:
      - TestValidateSDK_MissingVersion

  - id: CLM-086
    requirement: REQ-017
    text: SDK with empty provides list is rejected
    tests:
      - TestValidateSDK_EmptyProvides

  - id: CLM-087
    requirement: REQ-017
    text: SDK missing provides is rejected
    tests:
      - TestValidateSDK_MissingProvides

  # REQ-018: tool_config — standalone vs supporting
  - id: CLM-088
    requirement: REQ-018
    text: Standalone tool_config with id, risk_class, tool, file, and claims is accepted
    tests:
      - TestValidateToolConfig_Standalone

  - id: CLM-089
    requirement: REQ-018
    text: Supporting tool_config with required_by, tool, and file is accepted
    tests:
      - TestValidateToolConfig_Supporting

  - id: CLM-090
    requirement: REQ-018
    text: tool_config with both id and required_by is rejected
    tests:
      - TestValidateToolConfig_BothIdAndRequiredBy

  - id: CLM-091
    requirement: REQ-018
    text: tool_config with neither id nor required_by is rejected
    tests:
      - TestValidateToolConfig_NeitherIdNorRequiredBy

  - id: CLM-092
    requirement: REQ-018
    text: Standalone tool_config missing risk_class is rejected
    tests:
      - TestValidateToolConfig_StandaloneMissingRiskClass

  - id: CLM-093
    requirement: REQ-018
    text: Standalone tool_config missing claims is rejected
    tests:
      - TestValidateToolConfig_StandaloneMissingClaims

  - id: CLM-094
    requirement: REQ-018
    text: tool_config missing tool field is rejected
    tests:
      - TestValidateToolConfig_MissingTool

  - id: CLM-095
    requirement: REQ-018
    text: tool_config missing file field is rejected
    tests:
      - TestValidateToolConfig_MissingFile

  # REQ-019: tool_config traceability
  - id: CLM-096
    requirement: REQ-019
    text: Standalone tool_config is traceable via its own id
    tests:
      - TestValidateToolConfigTrace_Standalone

  - id: CLM-097
    requirement: REQ-019
    text: Supporting tool_config is traceable via required_by
    tests:
      - TestValidateToolConfigTrace_Supporting

  # REQ-020: Standard field optionality
  - id: CLM-098
    requirement: REQ-020
    text: Layer 2 rule with filepath standard is accepted
    tests:
      - TestValidateStandard_Layer2Filepath

  - id: CLM-099
    requirement: REQ-020
    text: Layer 2 rule with inline string standard is accepted
    tests:
      - TestValidateStandard_Layer2InlineString

  - id: CLM-100
    requirement: REQ-020
    text: Layer 3 rule without standard is accepted
    tests:
      - TestValidateStandard_Layer3NoStandard

  - id: CLM-101
    requirement: REQ-020
    text: Layer 1 rule without standard is accepted
    tests:
      - TestValidateStandard_Layer1NoStandard

  # REQ-021: Claim ID uniqueness
  - id: CLM-102
    requirement: REQ-021
    text: Manifest with unique claim IDs is accepted
    tests:
      - TestValidateClaimIDs_Unique

  - id: CLM-103
    requirement: REQ-021
    text: Manifest with duplicate claim IDs is rejected
    tests:
      - TestValidateClaimIDs_Duplicate

  # REQ-022: Language field
  - id: CLM-104
    requirement: REQ-022
    text: Valid language string is accepted
    tests:
      - TestValidateLanguage_Valid

  - id: CLM-105
    requirement: REQ-022
    text: Empty language string is rejected
    tests:
      - TestValidateLanguage_Empty

  # REQ-023: Coordinate references
  - id: CLM-106
    requirement: REQ-023
    text: Valid coordinate pack-name@version:item-name@item-version is parsed correctly
    tests:
      - TestParseCoordinate_FullFormat

  - id: CLM-107
    requirement: REQ-023
    text: Coordinate with missing item version is parsed (item version optional)
    tests:
      - TestParseCoordinate_NoItemVersion

  - id: CLM-108
    requirement: REQ-023
    text: Invalid coordinate format returns error
    tests:
      - TestParseCoordinate_Invalid

  # REQ-024: Embedded pack same path
  - id: CLM-109
    requirement: REQ-024
    text: Embedded Go standards pack is parsed via ParseManifest without special-case code
    tests:
      - TestParseManifest_EmbeddedPackSamePath

  # REQ-025: Scaffold copy-once / SDK module ref
  - id: CLM-110
    requirement: REQ-025
    text: Scaffold struct contains path field and no update mechanism
    tests:
      - TestScaffoldType_HasPathNoUpdate

  - id: CLM-111
    requirement: REQ-025
    text: SDK struct contains module, version, provides and no distribution fields
    tests:
      - TestSDKType_ModuleVersionProvides

  # REQ-026: pairs_with parsing
  - id: CLM-112
    requirement: REQ-026
    text: pairs_with with rules, scaffolds, and sdk keys is parsed correctly
    tests:
      - TestParsePairsWith_AllKeys

  - id: CLM-113
    requirement: REQ-026
    text: pairs_with with only rules key is parsed correctly
    tests:
      - TestParsePairsWith_RulesOnly

  # REQ-027: Fixture directory naming
  - id: CLM-114
    requirement: REQ-027
    text: Fixture directory matching rule ID in lowercase is accepted
    tests:
      - TestValidateFixtureDir_MatchesRuleID

  - id: CLM-115
    requirement: REQ-027
    text: Fixture directory not matching rule ID is flagged
    tests:
      - TestValidateFixtureDir_Mismatch

  # REQ-028: Fixture entry format (string or object)
  - id: CLM-116
    requirement: REQ-028
    text: Positive fixture as plain string path is parsed
    tests:
      - TestParseFixture_StringPath

  - id: CLM-117
    requirement: REQ-028
    text: Negative fixture as object with path and bypass_attempt is parsed
    tests:
      - TestParseFixture_ObjectWithBypass

  - id: CLM-118
    requirement: REQ-028
    text: Negative fixture as object with path only (no bypass_attempt) is parsed
    tests:
      - TestParseFixture_ObjectWithoutBypass

  # REQ-029: Directory layout expectations from manifest
  - id: CLM-119
    requirement: REQ-029
    text: Enforcement pack with layer 2 rules expects rules/ directory
    tests:
      - TestExpectedLayout_EnforcementWithLayer2

  - id: CLM-120
    requirement: REQ-029
    text: Pack with layer 3 rules expects validators/ directory
    tests:
      - TestExpectedLayout_WithLayer3

  - id: CLM-121
    requirement: REQ-029
    text: Code pack expects scaffolds/ directory
    tests:
      - TestExpectedLayout_CodePack

  - id: CLM-122
    requirement: REQ-029
    text: Pack always expects fixtures/rules/ directory
    tests:
      - TestExpectedLayout_FixturesAlways

  # REQ-030: Layer 3 isolation fields
  - id: CLM-123
    requirement: REQ-030
    text: Layer 3 rule validator and input_scope are represented in manifest types
    tests:
      - TestLayer3Type_ValidatorAndInputScope

  # REQ-031: Types support both pack check and pack test
  - id: CLM-124
    requirement: REQ-031
    text: Manifest types include all fields needed for structural validation
    tests:
      - TestManifestTypes_StructuralFields

  - id: CLM-125
    requirement: REQ-031
    text: Manifest types include all fields needed for fixture execution
    tests:
      - TestManifestTypes_FixtureExecutionFields

  # REQ-032: pack try metadata
  - id: CLM-126
    requirement: REQ-032
    text: Manifest types include rule paths and tool configs needed for pack try
    tests:
      - TestManifestTypes_PackTryMetadata

  # REQ-033: Co-occurrence — archetype x content type matrix
  - id: CLM-127
    requirement: REQ-033
    text: Code pack with scaffolds and rules (bidirectional pairing) is accepted
    tests:
      - TestValidateCoOccurrence_CodePackValid

  - id: CLM-128
    requirement: REQ-033
    text: Code pack scaffold without any paired rule is rejected
    tests:
      - TestValidateCoOccurrence_ScaffoldNoPairedRule

  - id: CLM-129
    requirement: REQ-033
    text: Code pack rule without pairs_with scaffold or SDK is rejected
    tests:
      - TestValidateCoOccurrence_RuleNoPairedContent

  - id: CLM-130
    requirement: REQ-033
    text: Enforcement pack declaring scaffolds is rejected
    tests:
      - TestValidateCoOccurrence_EnforcementWithScaffolds

  - id: CLM-131
    requirement: REQ-033
    text: Enforcement pack declaring sdk is rejected
    tests:
      - TestValidateCoOccurrence_EnforcementWithSDK

  # REQ-034: Fixture proof for all archetypes
  - id: CLM-132
    requirement: REQ-034
    text: Rule with claims and fixtures is accepted for enforcement pack
    tests:
      - TestValidateFixtureProof_EnforcementWithFixtures

  - id: CLM-133
    requirement: REQ-034
    text: Rule with claims and fixtures is accepted for code pack
    tests:
      - TestValidateFixtureProof_CodeWithFixtures

  - id: CLM-134
    requirement: REQ-034
    text: Rule without any claims is rejected
    tests:
      - TestValidateFixtureProof_RuleNoClaims

  # REQ-035: go.mod / language dependency file
  - id: CLM-135
    requirement: REQ-035
    text: Go-language pack expects go.mod in directory layout
    tests:
      - TestExpectedLayout_GoModRequired

contracts:
  - file: pkg/pack/manifest.go
    provides:
      - name: Manifest
        kind: type
        signature: "type Manifest struct"
        notes: "Top-level pack manifest: Name, Version, Language, Archetype, Description, Content, ToolConfig"
      - name: Content
        kind: type
        signature: "type Content struct"
        notes: "Content block: Ruleset, Scaffolds, SDK, Contracts, TestPatterns, ASTChecks, Rubrics"
      - name: Ruleset
        kind: type
        signature: "type Ruleset struct"
        notes: "Ruleset with Version and Rules slice"
      - name: Rule
        kind: type
        signature: "type Rule struct"
        notes: "ID, Standard, RulePath, RiskClass, Layer, Claims, Category, Justification, Validator, InputScope, PairsWith"
      - name: Claim
        kind: type
        signature: "type Claim struct"
        notes: "ID, Text, Fixtures"
      - name: Fixtures
        kind: type
        signature: "type Fixtures struct"
        notes: "Positive []FixtureEntry, Negative []FixtureEntry"
      - name: FixtureEntry
        kind: type
        signature: "type FixtureEntry struct"
        notes: "Path string, BypassAttempt bool"
      - name: Scaffold
        kind: type
        signature: "type Scaffold struct"
        notes: "ID, Version, Tier, Path, TestCommand, Description, UseWhen, Assumes, PairsWith, SampleConfig"
      - name: SDK
        kind: type
        signature: "type SDK struct"
        notes: "Module, Version, Provides"
      - name: ToolConfigEntry
        kind: type
        signature: "type ToolConfigEntry struct"
        notes: "ID, Tool, File, Settings, RiskClass, Claims, RequiredBy"
      - name: PairsWith
        kind: type
        signature: "type PairsWith struct"
        notes: "Rules []string, Scaffolds []string, SDK string"
      - name: Coordinate
        kind: type
        signature: "type Coordinate struct"
        notes: "PackName, PackVersion, ItemName, ItemVersion"
      - name: ParseManifest
        kind: function
        signature: "func ParseManifest(data []byte) (*Manifest, error)"
      - name: ParseManifestFile
        kind: function
        signature: "func ParseManifestFile(path string) (*Manifest, error)"
      - name: ParseCoordinate
        kind: function
        signature: "func ParseCoordinate(ref string) (*Coordinate, error)"
      - name: NamespacedRuleID
        kind: function
        signature: "func NamespacedRuleID(packName, ruleID string) string"
    consumes: []

  - file: pkg/pack/validate_manifest.go
    provides:
      - name: ValidateManifest
        kind: function
        signature: "func ValidateManifest(m *Manifest) []ValidationError"
        notes: "Structural validation: required fields, enums, format constraints, layer-specific rules, archetype constraints"
      - name: ValidationError
        kind: type
        signature: "type ValidationError struct"
        notes: "Field, Message, Rule string"
      - name: ExpectedLayout
        kind: function
        signature: "func ExpectedLayout(m *Manifest) []string"
        notes: "Returns expected directory entries based on archetype and declared layers"
    consumes: []
---

# SPEC-013: Pack Manifest Schema

## Overview

This spec defines the pack.yml manifest schema: the types, parsing, and structural validation rules that determine what constitutes a valid pack manifest. A pack is a declarative, portable artifact capturing standards, rules, scaffolds, and SDK references for reuse across projects. The manifest is the pack's root declaration, optimized for machine generation and parsing per ADR-0001 (agent-first).

The scope covers:

- **Type definitions** for all manifest structures (Manifest, Content, Rule, Claim, Scaffold, SDK, ToolConfigEntry, etc.)
- **Parsing** of pack.yml into typed Go structs (ParseManifest, ParseManifestFile)
- **Structural validation** of field presence, enum values, format constraints, layer-specific field requirements, and archetype constraints (ValidateManifest)
- **Coordinate references** for versioned pack item references (ParseCoordinate)
- **Namespaced rule IDs** for pack-scoped rule identification (NamespacedRuleID)

This spec does NOT cover coherence validation (reference resolution, co-occurrence enforcement beyond structural constraints), fixture execution, or distribution lifecycle. Those are scoped to BUNDLE-005 and BUNDLE-006 respectively.

## Requirements

Requirements are defined in frontmatter. Claims are defined in frontmatter.

### Key Design Decisions

- **Declarative data, not Go code.** Pack manifests are YAML data parsed into structs. No interfaces for pack authors to implement. The Go-side loader adapts to pack authors; errors speak in pack-author vocabulary.
- **Two archetypes.** Every pack is either enforcement (rules only) or code (ships scaffolds/SDKs AND rules enforcing correct usage). Enforcement packs must not declare scaffolds or sdk.
- **Three enforcement layers.** Layer 1 (built-in tool rules), layer 2 (custom semgrep rules), layer 3 (custom validators). Each layer has distinct required and prohibited fields.
- **risk_class vs category.** risk_class (security/correctness/style/perf) appears on ALL rules. category (presence/structural/other) appears ONLY on layer 3 rules. They never collide.
- **Three version levels.** Pack version (whole artifact), ruleset version (all rules as cohort), item version (scaffolds and SDKs individually).
- **Fixture mapping in manifest.** Claim-to-fixture mapping is declared exclusively in pack.yml. Fixture files are plain code with engine-native annotations.

### Layer Field Requirements Matrix

| Field | Layer 1 | Layer 2 | Layer 3 |
|-------|---------|---------|---------|
| risk_class | Required | Required | Required |
| layer | Required | Required | Required |
| rule (semgrep path) | Prohibited | Required | Prohibited |
| standard | Optional | Required | Optional |
| category | Prohibited | Prohibited | Required |
| justification | N/A | N/A | Required if category=other |
| input_scope | Prohibited | Prohibited | Required |
| validator | Prohibited | Prohibited | Required |

### Archetype Content Constraints

| Content Type | enforcement | code |
|-------------|-------------|------|
| ruleset | Allowed | Allowed |
| scaffolds | Prohibited | Allowed |
| sdk | Prohibited | Allowed |
| contracts | Allowed | Allowed |
| test_patterns | Allowed | Allowed |
| ast_checks | Allowed | Allowed |
| rubrics | Allowed | Allowed |

### risk_class Enum

Allowed values: security, correctness, style, perf. All other values are rejected.

### Scaffold Tier Enum

Allowed values: complete, skeleton. All other values are rejected.

### Layer 3 Category Enum

Allowed values: presence, structural, other. Categories presence and structural are auto-accepted (no justification required). Category other requires a mandatory justification field.

## Implementation

### Package Structure

```
pkg/pack/
  manifest.go            # Type definitions and parsing (ParseManifest, ParseManifestFile, ParseCoordinate, NamespacedRuleID)
  validate_manifest.go   # Structural validation (ValidateManifest, ExpectedLayout)
  manifest_test.go       # Tests for parsing and type behavior
  validate_manifest_test.go  # Tests for structural validation
```

### Parsing Flow

```
pack.yml bytes
  -> yaml.Unmarshal into Manifest struct
  -> normalize Name to lowercase (preserve original in DisplayName)
  -> parse FixtureEntry (string path or {path, bypass_attempt} object)
  -> resolve ruleset version default (pack version if omitted for enforcement packs)
  -> return *Manifest, error
```

### Validation Passes (ValidateManifest)

1. **Required top-level fields** — name, version, language, archetype, description, content must be present and non-empty.
2. **Name format** — must be two-part org/pack-name with valid characters.
3. **Version format** — pack version must be valid semver. Ruleset and item versions, if present, must be valid semver.
4. **Archetype enum** — must be enforcement or code.
5. **Content type allowlist** — only known content types; enforcement packs must not have scaffolds or sdk.
6. **Rule validation** — for each rule: risk_class enum, layer enum, rule ID format, layer-specific field enforcement (layer 2 needs rule+standard, layer 3 needs category+validator+input_scope, layers 1/3 must not have rule field, layers 1/2 must not have category/validator/input_scope).
7. **Claim validation** — claim ID uniqueness, fixture presence (positive and negative required), bypass_attempt on security-class rules.
8. **Scaffold validation** — tier enum, test_command required, use_when/assumes/pairs_with required, sample_config format.
9. **SDK validation** — module, version, provides required and non-empty.
10. **tool_config validation** — each entry is standalone (id) xor supporting (required_by), standalone entries need risk_class and claims, all entries need tool and file.
11. **Co-occurrence** — code packs require bidirectional scaffold-rule pairing; enforcement packs prohibit scaffolds and sdk.
12. **Expected layout** — derive expected directories from archetype and layers.

### Coordinate Parsing

ParseCoordinate parses `pack-name@pack-version:item-name@item-version` into constituent parts. The item-name and item-version portions are optional (for pack-level references).

### Namespaced Rule IDs

NamespacedRuleID concatenates pack name and rule ID with a slash: `org/pack-name/rule-id`.

## Verification

Verification is defined in frontmatter. Unit-level verification with 90% coverage threshold.

Test command: `go test ./pkg/pack/ -race -coverprofile=cover.out`

Claims are defined in frontmatter. Each requirement maps to specific test functions covering both positive (accepted) and negative (rejected) scenarios. The dependency matrix rule is applied to all enum fields (archetype, risk_class, layer, tier, category) and to the layer field requirements matrix.

## Sharp Edges

- **Structural vs coherence validation boundary.** This spec validates field presence, enums, and format constraints. It does NOT validate that a rule field path actually points to an existing file, or that pairs_with references resolve to real rules. Those are coherence checks belonging to BUNDLE-005. Implementers must resist the temptation to add file-existence checks to ValidateManifest — that crosses the boundary.

- **FixtureEntry polymorphism.** Negative fixtures can be plain string paths or objects with path and bypass_attempt. The YAML unmarshaler must handle both forms. A custom UnmarshalYAML on FixtureEntry is needed. Getting this wrong causes silent data loss (bypass_attempt flags silently dropped).

- **tool_config dual identity.** A standalone tool_config (with id) is simultaneously a tool configuration AND a layer 1 rule. It must satisfy both tool_config requirements (tool, file) and rule requirements (risk_class, claims with fixtures). Missing this duality means standalone tool_configs pass validation without fixture proof.

- **Ruleset version defaulting.** For enforcement packs, omitted ruleset version defaults to pack version. For code packs, this default does NOT apply (code packs may have ruleset and item versions that diverge from pack version). Getting the conditional wrong means code packs silently inherit pack version as ruleset version.

- **Co-occurrence is bidirectional.** It is not enough that every scaffold has a paired rule. Every rule in a code pack must also pair with a scaffold or SDK. A code pack with 10 scaffolds and 11 rules where the 11th rule has no pairs_with is invalid. The bidirectionality is easy to forget when implementing only one direction.

- **Category vs risk_class confusion.** Both are enums on rules. category appears ONLY on layer 3; risk_class appears on ALL rules. Validators that check category on layer 1/2 rules, or that skip risk_class on layer 3 rules, have a bug.

- **Empty vs missing distinction.** An empty content block (`content: {}`) is different from a missing content block. Both should be rejected (content must have at least a ruleset), but they may deserialize differently in Go. The YAML unmarshaler must distinguish these cases.

## Review Questions

1. Does ValidateManifest reject a layer 2 rule that declares both a rule field AND a validator field? The spec prohibits validator on layers 1-2, but an explicit test for the combination is warranted.

2. When a standalone tool_config has risk_class: security, does the bypass_attempt fixture requirement from REQ-011 apply to its claims? The spec says risk_class applies to ALL rules including standalone tool_config — confirm the implementation follows through.

3. Does ParseManifest preserve the original casing of the pack name for display while normalizing to lowercase for matching? A regression here would break display output while keeping matching correct (or vice versa).

4. If a code pack has rules that pair with the SDK (via pairs_with.sdk) but no scaffolds at all, is that valid? The bundle says "ships SDKs/scaffolds AND rules" — a code pack with only SDK and rules (no scaffolds) should still be valid per co-occurrence if every rule pairs with the SDK.

5. Does ExpectedLayout correctly handle the conditional directories? A pack with zero layer 2 rules should NOT expect rules/. A pack with zero layer 3 rules should NOT expect validators/. Only the presence of rules at those layers triggers the directory expectation.

6. Are claim IDs validated for uniqueness across BOTH ruleset claims and tool_config claims? A duplicate between a ruleset claim and a tool_config claim should be caught.

## References

- **BUNDLE-004** (pack-manifest-authoring) — source bundle, REQ-001 through REQ-035
- **ADR-0001** — agent-first discipline framework (manifest optimized for machine generation)
- **BUNDLE-005** — pack validation pipeline (coherence checks, fixture execution)
- **BUNDLE-006** — pack distribution lifecycle
- **SPEC-012** — existing Go standards pack (first pack under this model)
