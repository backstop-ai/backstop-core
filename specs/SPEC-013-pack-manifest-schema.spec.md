---
title: "SPEC-013: Pack Manifest Types and Parsing"
number: SPEC-013
created: "2026-04-14"
status: draft
schema_version: spec/v1
spec_version: 1.0.0

implementation:
  summary: >
    Define and implement the pack.yml manifest types and parsing: Go structs
    for representing pack manifests, YAML parsing into typed structures, field
    enum validation, pack naming normalization, fixture entry polymorphism,
    versioning fields, coordinate reference parsing, and namespaced rule IDs.
    This spec covers the data model — what a valid pack.yml looks like as Go
    types. Constraint validation (layer-specific field requirements, archetype
    enforcement, co-occurrence, layout) is in SPEC-016.
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
    supports: pack-manifest-authoring:REQ-001@1.0.0

  - id: REQ-002
    text: >
      Pack names must follow the two-part org/pack-name format. Each part
      accepts alphanumeric characters and hyphens. Names are normalized to
      lowercase internally for namespacing and matching. Original casing is
      preserved in the Name field. A name that does not contain exactly one
      slash separator, or contains characters outside [a-zA-Z0-9-], must be
      rejected.
    supports: pack-manifest-authoring:REQ-002@1.0.0

  - id: REQ-003
    text: >
      Every pack must declare exactly one archetype field with value
      "enforcement" or "code". Any other value or a missing archetype field
      must be rejected.
    supports: pack-manifest-authoring:REQ-004@1.0.0

  - id: REQ-004
    text: >
      Every rule must declare a risk_class field with one of four allowed
      values: security, correctness, style, perf. A rule with a missing or
      invalid risk_class must be rejected. This applies to all rules
      regardless of layer, including standalone tool_config rules.
    supports: pack-manifest-authoring:REQ-007@1.0.0

  - id: REQ-005
    text: >
      Rules must be organized into three enforcement layers. Each rule must
      declare a layer field with value 1, 2, or 3. A rule with a missing or
      invalid layer value must be rejected.
    supports: pack-manifest-authoring:REQ-008@1.0.0

  - id: REQ-006
    text: >
      Every claim must declare at least one positive fixture (known-good) and
      at least one negative fixture (known-bad). Fixture paths are relative
      to the pack root and declared inline on claims. A claim with zero
      positive fixtures or zero negative fixtures must be rejected.
    supports: pack-manifest-authoring:REQ-013@1.0.0

  - id: REQ-007
    text: >
      Rule IDs must use lowercase kebab-case format matching the pattern
      ^[a-z][a-z0-9]*(-[a-z0-9]+)*$. On load, rule IDs are namespaced with
      the pack name using slash delimiters: pack-name/rule-id. A rule ID
      not matching the pattern must be rejected.
    supports: pack-manifest-authoring:REQ-017@1.0.0

  - id: REQ-008
    text: >
      Pack versioning uses three levels. Pack version (required, semver
      format) covers the whole artifact. Ruleset version (optional, defaults
      to pack version for enforcement packs if omitted) covers all rules as
      a cohort. Item version (on scaffolds and SDKs individually) enables
      spec and plan references. All version fields that are present must
      conform to semver format.
    supports: pack-manifest-authoring:REQ-018@1.0.0

  - id: REQ-009
    text: >
      Scaffolds must declare a tier field with value complete or skeleton.
      Any other value or a missing tier must be rejected.
    supports: pack-manifest-authoring:REQ-019@1.0.0

  - id: REQ-010
    text: >
      Every scaffold must declare a test_command field specifying how to run
      its tests. A scaffold missing test_command must be rejected.
    supports: pack-manifest-authoring:REQ-020@1.0.0

  - id: REQ-011
    text: >
      Scaffolds must declare use_when (non-empty list of scenario strings),
      assumes (non-empty list of precondition strings), and pairs_with (a
      single object with optional keys: rules, scaffolds, sdk). sample_config
      values must be flat key-value string pairs. A scaffold missing use_when
      or assumes must be rejected. pairs_with is required on scaffolds.
    supports: pack-manifest-authoring:REQ-021@1.0.0

  - id: REQ-012
    text: >
      The sdk content type is a single optional entry with required fields:
      module (string), version (string), and provides (non-empty list of
      strings representing the public surface). A pack may declare at most
      one SDK. An sdk entry missing module, version, or provides, or an
      sdk with an empty provides list, must be rejected.
    supports: pack-manifest-authoring:REQ-022@1.0.0

  - id: REQ-013
    text: >
      tool_config entries must declare tool (string) and file (string,
      consumer-side target path). Standalone tool_config entries (those with
      an id field) are layer 1 rules and must additionally declare risk_class
      and claims with fixtures. Supporting tool_config entries (those with
      required_by) must not declare id, risk_class, or claims. A tool_config
      entry must be either standalone (has id) or supporting (has required_by),
      not both and not neither.
    supports: pack-manifest-authoring:REQ-023@1.0.0

  - id: REQ-014
    text: >
      The standard field on rules accepts a filepath string (pointing to a
      .standard.md file) or an inline string. The field is parsed as a
      string regardless of content. Layer-specific requirements for when
      standard must be present are enforced in SPEC-016.
    supports: pack-manifest-authoring:REQ-025@1.0.0

  - id: REQ-015
    text: >
      Claim IDs must be unique within the pack. Duplicate claim IDs within
      a single pack manifest must be rejected.
    supports: pack-manifest-authoring:REQ-027@1.0.0

  - id: REQ-016
    text: >
      Each pack targets exactly one language via the required language field.
      The language field must be a non-empty string.
    supports: pack-manifest-authoring:REQ-030@1.0.0

  - id: REQ-017
    text: >
      The manifest must support versioned coordinate references from specs
      and plans using the format pack-name@pack-version:item-name@item-version.
      ParseCoordinate must parse this format and return the constituent parts.
      An invalid coordinate format must return an error.
    supports: pack-manifest-authoring:REQ-031@1.0.0

  - id: REQ-018
    text: >
      Scaffolds are copy-once templates. The manifest must represent scaffolds
      with a path field (directory within the pack), and the scaffold struct
      must not imply update semantics. SDKs are native-language module
      references tracked in the manifest; the SDK struct contains module,
      version, and provides but no distribution mechanism.
    supports: pack-manifest-authoring:REQ-033@1.0.0

  - id: REQ-019
    text: >
      A pack's rules may reference its own SDK surface via pairs_with.sdk on
      the rule or scaffold. The pairs_with field is a single object with
      optional keys rules, scaffolds, sdk. Structural parsing of pairs_with
      must accept all three keys.
    supports: pack-manifest-authoring:REQ-034@1.0.0

  - id: REQ-020
    text: >
      Semgrep fixtures must include engine-native annotations (// ruleid:
      and // ok: comments). Claim-to-fixture mapping is declared exclusively
      in pack.yml, not in fixture file metadata. Fixture entries in the
      manifest may be a plain string path or an object with path and optional
      bypass_attempt boolean.
    supports: pack-manifest-authoring:REQ-015@1.0.0


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

  # REQ-003: Archetype enum
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

  # REQ-004: risk_class enum
  - id: CLM-017
    requirement: REQ-004
    text: risk_class security is accepted
    tests:
      - TestValidateRiskClass_Security

  - id: CLM-018
    requirement: REQ-004
    text: risk_class correctness is accepted
    tests:
      - TestValidateRiskClass_Correctness

  - id: CLM-019
    requirement: REQ-004
    text: risk_class style is accepted
    tests:
      - TestValidateRiskClass_Style

  - id: CLM-020
    requirement: REQ-004
    text: risk_class perf is accepted
    tests:
      - TestValidateRiskClass_Perf

  - id: CLM-021
    requirement: REQ-004
    text: risk_class with invalid value is rejected
    tests:
      - TestValidateRiskClass_Invalid

  - id: CLM-022
    requirement: REQ-004
    text: Missing risk_class is rejected
    tests:
      - TestValidateRiskClass_Missing

  # REQ-005: Layer enum
  - id: CLM-023
    requirement: REQ-005
    text: Layer 1 is accepted
    tests:
      - TestValidateLayer_One

  - id: CLM-024
    requirement: REQ-005
    text: Layer 2 is accepted
    tests:
      - TestValidateLayer_Two

  - id: CLM-025
    requirement: REQ-005
    text: Layer 3 is accepted
    tests:
      - TestValidateLayer_Three

  - id: CLM-026
    requirement: REQ-005
    text: Layer 0 is rejected
    tests:
      - TestValidateLayer_Zero

  - id: CLM-027
    requirement: REQ-005
    text: Layer 4 is rejected
    tests:
      - TestValidateLayer_Four

  - id: CLM-028
    requirement: REQ-005
    text: Missing layer is rejected
    tests:
      - TestValidateLayer_Missing

  # REQ-006: Fixture presence
  - id: CLM-029
    requirement: REQ-006
    text: Claim with both positive and negative fixtures is accepted
    tests:
      - TestValidateFixtures_BothPresent

  - id: CLM-030
    requirement: REQ-006
    text: Claim with zero positive fixtures is rejected
    tests:
      - TestValidateFixtures_NoPositive

  - id: CLM-031
    requirement: REQ-006
    text: Claim with zero negative fixtures is rejected
    tests:
      - TestValidateFixtures_NoNegative

  # REQ-007: Rule ID format
  - id: CLM-032
    requirement: REQ-007
    text: Lowercase kebab-case rule ID is accepted
    tests:
      - TestValidateRuleID_ValidKebab

  - id: CLM-033
    requirement: REQ-007
    text: Rule ID with uppercase characters is rejected
    tests:
      - TestValidateRuleID_Uppercase

  - id: CLM-034
    requirement: REQ-007
    text: Rule ID with underscores is rejected
    tests:
      - TestValidateRuleID_Underscores

  - id: CLM-035
    requirement: REQ-007
    text: Rule ID is correctly namespaced with pack name on load
    tests:
      - TestNamespaceRuleID_Correct

  # REQ-008: Versioning
  - id: CLM-036
    requirement: REQ-008
    text: Valid semver pack version is accepted
    tests:
      - TestValidateVersion_ValidSemver

  - id: CLM-037
    requirement: REQ-008
    text: Invalid pack version format is rejected
    tests:
      - TestValidateVersion_InvalidFormat

  - id: CLM-038
    requirement: REQ-008
    text: Enforcement pack with omitted ruleset version defaults to pack version
    tests:
      - TestValidateVersion_RulesetDefaultsToPackVersion

  - id: CLM-039
    requirement: REQ-008
    text: Explicit ruleset version overrides pack version
    tests:
      - TestValidateVersion_ExplicitRulesetVersion

  # REQ-009: Scaffold tier enum
  - id: CLM-040
    requirement: REQ-009
    text: Scaffold tier complete is accepted
    tests:
      - TestValidateScaffoldTier_Complete

  - id: CLM-041
    requirement: REQ-009
    text: Scaffold tier skeleton is accepted
    tests:
      - TestValidateScaffoldTier_Skeleton

  - id: CLM-042
    requirement: REQ-009
    text: Scaffold tier with invalid value is rejected
    tests:
      - TestValidateScaffoldTier_Invalid

  - id: CLM-043
    requirement: REQ-009
    text: Scaffold missing tier is rejected
    tests:
      - TestValidateScaffoldTier_Missing

  # REQ-010: Scaffold test_command
  - id: CLM-044
    requirement: REQ-010
    text: Scaffold with test_command is accepted
    tests:
      - TestValidateScaffold_WithTestCommand

  - id: CLM-045
    requirement: REQ-010
    text: Scaffold missing test_command is rejected
    tests:
      - TestValidateScaffold_MissingTestCommand

  # REQ-011: Scaffold use_when, assumes, pairs_with, sample_config
  - id: CLM-046
    requirement: REQ-011
    text: Scaffold with use_when, assumes, and pairs_with is accepted
    tests:
      - TestValidateScaffold_AllFieldsPresent

  - id: CLM-047
    requirement: REQ-011
    text: Scaffold missing use_when is rejected
    tests:
      - TestValidateScaffold_MissingUseWhen

  - id: CLM-048
    requirement: REQ-011
    text: Scaffold missing assumes is rejected
    tests:
      - TestValidateScaffold_MissingAssumes

  - id: CLM-049
    requirement: REQ-011
    text: Scaffold missing pairs_with is rejected
    tests:
      - TestValidateScaffold_MissingPairsWith

  - id: CLM-050
    requirement: REQ-011
    text: Scaffold with empty use_when list is rejected
    tests:
      - TestValidateScaffold_EmptyUseWhen

  - id: CLM-051
    requirement: REQ-011
    text: Scaffold sample_config with non-string values is rejected
    tests:
      - TestValidateScaffold_SampleConfigNonString

  - id: CLM-052
    requirement: REQ-011
    text: Scaffold pairs_with parses rules, scaffolds, and sdk keys
    tests:
      - TestParseScaffold_PairsWithAllKeys

  # REQ-012: SDK content type
  - id: CLM-053
    requirement: REQ-012
    text: Valid SDK with module, version, and provides is accepted
    tests:
      - TestValidateSDK_Valid

  - id: CLM-054
    requirement: REQ-012
    text: SDK missing module is rejected
    tests:
      - TestValidateSDK_MissingModule

  - id: CLM-055
    requirement: REQ-012
    text: SDK missing version is rejected
    tests:
      - TestValidateSDK_MissingVersion

  - id: CLM-056
    requirement: REQ-012
    text: SDK with empty provides list is rejected
    tests:
      - TestValidateSDK_EmptyProvides

  - id: CLM-057
    requirement: REQ-012
    text: SDK missing provides is rejected
    tests:
      - TestValidateSDK_MissingProvides

  # REQ-013: tool_config structure
  - id: CLM-058
    requirement: REQ-013
    text: Standalone tool_config with id, risk_class, tool, file, and claims is accepted
    tests:
      - TestValidateToolConfig_Standalone

  - id: CLM-059
    requirement: REQ-013
    text: Supporting tool_config with required_by, tool, and file is accepted
    tests:
      - TestValidateToolConfig_Supporting

  - id: CLM-060
    requirement: REQ-013
    text: tool_config with both id and required_by is rejected
    tests:
      - TestValidateToolConfig_BothIdAndRequiredBy

  - id: CLM-061
    requirement: REQ-013
    text: tool_config with neither id nor required_by is rejected
    tests:
      - TestValidateToolConfig_NeitherIdNorRequiredBy

  - id: CLM-062
    requirement: REQ-013
    text: Standalone tool_config missing risk_class is rejected
    tests:
      - TestValidateToolConfig_StandaloneMissingRiskClass

  - id: CLM-063
    requirement: REQ-013
    text: Standalone tool_config missing claims is rejected
    tests:
      - TestValidateToolConfig_StandaloneMissingClaims

  - id: CLM-064
    requirement: REQ-013
    text: tool_config missing tool field is rejected
    tests:
      - TestValidateToolConfig_MissingTool

  - id: CLM-065
    requirement: REQ-013
    text: tool_config missing file field is rejected
    tests:
      - TestValidateToolConfig_MissingFile

  # REQ-014: Standard field optionality
  - id: CLM-066
    requirement: REQ-014
    text: Standard field with filepath string is parsed correctly
    tests:
      - TestParseStandard_Filepath

  - id: CLM-067
    requirement: REQ-014
    text: Standard field with inline string is parsed correctly
    tests:
      - TestParseStandard_InlineString

  # REQ-015: Claim ID uniqueness
  - id: CLM-070
    requirement: REQ-015
    text: Manifest with unique claim IDs is accepted
    tests:
      - TestValidateClaimIDs_Unique

  - id: CLM-071
    requirement: REQ-015
    text: Manifest with duplicate claim IDs is rejected
    tests:
      - TestValidateClaimIDs_Duplicate

  # REQ-016: Language field
  - id: CLM-072
    requirement: REQ-016
    text: Valid language string is accepted
    tests:
      - TestValidateLanguage_Valid

  - id: CLM-073
    requirement: REQ-016
    text: Empty language string is rejected
    tests:
      - TestValidateLanguage_Empty

  # REQ-017: Coordinate references
  - id: CLM-074
    requirement: REQ-017
    text: Valid coordinate pack-name@version:item-name@item-version is parsed correctly
    tests:
      - TestParseCoordinate_FullFormat

  - id: CLM-075
    requirement: REQ-017
    text: Coordinate with missing item version is parsed (item version optional)
    tests:
      - TestParseCoordinate_NoItemVersion

  - id: CLM-076
    requirement: REQ-017
    text: Invalid coordinate format returns error
    tests:
      - TestParseCoordinate_Invalid

  # REQ-018: Scaffold copy-once / SDK module ref
  - id: CLM-077
    requirement: REQ-018
    text: Scaffold struct contains path field and no update mechanism
    tests:
      - TestScaffoldType_HasPathNoUpdate

  - id: CLM-078
    requirement: REQ-018
    text: SDK struct contains module, version, provides and no distribution fields
    tests:
      - TestSDKType_ModuleVersionProvides

  # REQ-019: pairs_with parsing
  - id: CLM-079
    requirement: REQ-019
    text: pairs_with with rules, scaffolds, and sdk keys is parsed correctly
    tests:
      - TestParsePairsWith_AllKeys

  - id: CLM-080
    requirement: REQ-019
    text: pairs_with with only rules key is parsed correctly
    tests:
      - TestParsePairsWith_RulesOnly

  # REQ-020: Fixture entry format
  - id: CLM-081
    requirement: REQ-020
    text: Positive fixture as plain string path is parsed
    tests:
      - TestParseFixture_StringPath

  - id: CLM-082
    requirement: REQ-020
    text: Negative fixture as object with path and bypass_attempt is parsed
    tests:
      - TestParseFixture_ObjectWithBypass

  - id: CLM-083
    requirement: REQ-020
    text: Negative fixture as object with path only (no bypass_attempt) is parsed
    tests:
      - TestParseFixture_ObjectWithoutBypass




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
---

# SPEC-013: Pack Manifest Types and Parsing

## Overview

A pack is a portable, declarative artifact that bundles standards, rules, scaffolds, and SDK references for reuse across projects. This spec defines the Go types that represent a pack manifest and the parsing logic that turns `pack.yml` bytes into those types.

Scope:

- Go struct definitions for every manifest element (Manifest, Content, Rule, Claim, Scaffold, SDK, ToolConfigEntry, PairsWith, Coordinate, FixtureEntry)
- YAML parsing via `ParseManifest` and `ParseManifestFile`
- Field-level validation: required fields, enum values, format patterns
- Fixture entry polymorphism (string path or `{path, bypass_attempt}` object)
- Coordinate reference parsing (`ParseCoordinate`)
- Namespaced rule ID construction (`NamespacedRuleID`)

Out of scope: layer-specific field requirements, archetype content restrictions, co-occurrence rules, and directory layout validation. Those belong to SPEC-016.

## Requirements

Requirements are defined in frontmatter. Claims are defined in frontmatter.

### Enum Values

The following enums are validated at parse time:

| Field | Allowed values |
|---|---|
| `archetype` | `enforcement`, `code` |
| `risk_class` | `security`, `correctness`, `style`, `perf` |
| `layer` | `1`, `2`, `3` |
| `tier` (scaffold) | `complete`, `skeleton` |

### Format Patterns

| Field | Pattern |
|---|---|
| Pack name | Two-part `org/pack-name`, each part `[a-zA-Z0-9-]`, exactly one `/` |
| Rule ID | `^[a-z][a-z0-9]*(-[a-z0-9]+)*$` (lowercase kebab-case) |
| Version fields | Semver format |
| Coordinate | `pack-name@pack-version:item-name@item-version` |

### tool_config Classification

A `tool_config` entry is exactly one of:

| Variant | Identifying field | Required fields | Prohibited fields |
|---|---|---|---|
| Standalone | `id` | `tool`, `file`, `risk_class`, `claims` | `required_by` |
| Supporting | `required_by` | `tool`, `file` | `id`, `risk_class`, `claims` |

An entry with both `id` and `required_by`, or neither, is rejected.

## Implementation

### Package Structure

```
pkg/pack/
  manifest.go            # Type definitions and parsing
  manifest_test.go       # Tests for all parsing and validation behavior
```

### Type Hierarchy

```
Manifest
 ├── Name, Version, Language, Archetype, Description
 ├── Content
 │    ├── Ruleset { Version, Rules []Rule }
 │    │    └── Rule { ID, Standard, RiskClass, Layer, Claims []Claim, PairsWith, ... }
 │    │         └── Claim { ID, Text, Fixtures { Positive []FixtureEntry, Negative []FixtureEntry } }
 │    ├── Scaffolds []Scaffold { ID, Version, Tier, Path, TestCommand, UseWhen, Assumes, PairsWith, SampleConfig }
 │    └── SDK { Module, Version, Provides }
 └── ToolConfig []ToolConfigEntry { ID, Tool, File, Settings, RiskClass, Claims, RequiredBy }
```

### Parsing Flow

`ParseManifest` accepts raw YAML bytes and returns a validated `*Manifest`:

1. **Deserialize.** `yaml.Unmarshal` into the `Manifest` struct. FixtureEntry uses a custom `UnmarshalYAML` to handle the string/object polymorphism.
2. **Required fields.** Reject if any of `name`, `version`, `language`, `archetype`, `description`, or `content` is missing.
3. **Name validation.** Verify exactly one `/`, alphanumeric-plus-hyphen parts. Normalize to lowercase for matching; preserve original casing.
4. **Enum validation.** Check `archetype`, `risk_class`, `layer`, and scaffold `tier` against their allowed value sets.
5. **Format validation.** Validate rule IDs against the kebab-case pattern. Validate all version fields against semver.
6. **Fixture presence.** Every claim must have at least one positive and one negative fixture.
7. **Claim ID uniqueness.** Scan all claims across rules and tool_config entries; reject duplicates.
8. **tool_config classification.** Each entry must be exactly standalone or supporting. Validate required fields per variant.
9. **Scaffold fields.** Verify `test_command`, `use_when`, `assumes`, and `pairs_with` presence and non-emptiness.
10. **SDK fields.** Verify `module`, `version`, and non-empty `provides`.
11. **Ruleset version default.** For enforcement packs with omitted ruleset version, set it to pack version.
12. **Namespace rule IDs.** Prefix each rule ID with the pack name via `NamespacedRuleID`.

`ParseManifestFile` reads a file from disk and delegates to `ParseManifest`.

`ParseCoordinate` parses `pack-name@pack-version:item-name@item-version` into its constituent parts. Item version is optional.

## Verification

Verification is defined in frontmatter. Unit-level testing at 90% coverage.

```
go test ./pkg/pack/ -race -coverprofile=cover.out
```

## Sharp Edges

1. **FixtureEntry polymorphism.** Fixtures can be a plain string path or an object with `path` and optional `bypass_attempt`. A custom `UnmarshalYAML` is required. If implemented incorrectly, `bypass_attempt` flags are silently dropped and all fixtures appear as string paths.

2. **Ruleset version defaulting is conditional.** Omitted ruleset version defaults to pack version only for enforcement packs. Code packs must not inherit this default. The branching logic must key off `archetype`, not the presence of rules.

3. **Empty vs missing content.** `content: {}` and a missing `content` key both deserialize to a zero-value `Content` struct in Go. The parser must distinguish these cases or treat both as invalid uniformly.

4. **Name normalization requires two fields.** The normalized lowercase form drives matching and namespacing. The original casing must survive for display. Forgetting to preserve original casing, or normalizing in place, breaks round-trip fidelity.

5. **Fixture entry in positive vs negative context.** `bypass_attempt` is meaningful only on negative fixtures. The type system does not prevent setting it on positive fixtures. Whether to reject or ignore this is a design decision that should be explicit.

## Review Questions

1. Does `ParseManifest` return all validation errors at once, or does it fail on the first error? Batch error reporting is more useful for pack authors, but the spec does not mandate either approach.

2. If a pack name contains uppercase letters, is the normalized form used everywhere downstream (rule namespacing, coordinate matching), or could the original casing leak into comparisons?

3. Can a `FixtureEntry` have an empty string path? The spec requires fixture presence but does not explicitly address zero-length paths.

4. If `pairs_with.sdk` references a non-existent SDK, is that a parse-time error (this spec) or a constraint-time error (SPEC-016)? The boundary must be clear.

5. How does `ParseManifest` handle unknown YAML keys? Strict unmarshaling rejects typos early; lenient unmarshaling risks silent misconfiguration.

## References

- **BUNDLE-004** (pack-manifest-authoring) -- source bundle for pack manifest requirements
- **SPEC-016** -- pack manifest constraints and layout; consumes types defined here
- **ADR-0001** -- agent-first discipline framework
