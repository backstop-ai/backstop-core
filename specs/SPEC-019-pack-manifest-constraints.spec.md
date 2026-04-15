---
title: "SPEC-019: Pack Manifest Constraints and Layout"
number: SPEC-019
created: "2026-04-14"
status: draft
schema_version: spec/v1
spec_version: 1.0.0

implementation:
  summary: >
    Implement the constraint and layout validation rules that govern pack
    manifests beyond basic type parsing. This covers layer-specific field
    requirements and prohibitions, archetype content constraints, bidirectional
    co-occurrence enforcement, layer 3 category auto-acceptance, scaffold tier
    verification expectations, tool_config traceability, canonical directory
    layout validation, namespaced rule IDs, fixture proof requirements, and
    command metadata support. This spec consumes the types defined in SPEC-013
    and adds the validation rules that enforce correct manifest structure.
  package: pkg/pack

verification:
  level: unit
  test_command: go test ./pkg/pack/ -race -coverprofile=cover.out
  coverage_threshold: 90

requirements:
  - id: REQ-001
    text: >
      The content block must declare typed content using only the allowed
      types: ruleset, scaffolds, sdk, contracts, test_patterns, ast_checks,
      rubrics. An enforcement pack must not declare scaffolds or sdk in its
      content block. An enforcement pack with only a ruleset (no other content
      types) is valid.
    supports: pack-manifest-authoring:REQ-006

  - id: REQ-002
    text: >
      Layer 2 rules must declare both a rule field (pointing to the compiled
      semgrep YAML file path) and a standard field (filepath to .standard.md
      or inline string). A layer 2 rule missing either field must be rejected.
      Layer 1 and layer 3 rules must not declare the rule field.
    supports: pack-manifest-authoring:REQ-009

  - id: REQ-003
    text: >
      Layer 3 rules must declare a category field with one of three values:
      presence, structural, or other. Categories presence and structural
      require no justification field. Category other requires a mandatory
      justification field; omitting justification when category is other must
      be rejected. A layer 3 rule missing category must be rejected. Layer 1
      and layer 2 rules must not declare category.
    supports: pack-manifest-authoring:REQ-010

  - id: REQ-004
    text: >
      Layer 3 rules must declare an input_scope field with value single-file
      or multi-file, and a validator field pointing to the validator script
      path. A layer 3 rule missing input_scope or validator must be rejected.
      Layer 1 and layer 2 rules must not declare input_scope or validator.
    supports: pack-manifest-authoring:REQ-011

  - id: REQ-005
    text: >
      Security-class rules (risk_class: security) must include at least one
      negative fixture with bypass_attempt: true, in addition to standard
      negative fixtures. A security-class rule where no negative fixture
      has bypass_attempt: true must be rejected.
    supports: pack-manifest-authoring:REQ-014

  - id: REQ-006
    text: >
      Every tool_config entry must be traceable to at least one rule. A
      standalone tool_config is its own rule (via id). A supporting
      tool_config must reference at least one rule via required_by. A
      tool_config entry that is neither standalone nor supporting must be
      rejected.
    supports: pack-manifest-authoring:REQ-024

  - id: REQ-007
    text: >
      The embedded Go standards pack must be loadable through the same
      ParseManifest path as third-party packs. No special-case code paths
      for embedded packs.
    supports: pack-manifest-authoring:REQ-032

  - id: REQ-008
    text: >
      Fixture directories must use lowercase naming matching rule IDs (e.g.,
      fixtures/rules/err-001/). Fixture directory names that do not match their
      rule ID in lowercase must be flagged.
    supports: pack-manifest-authoring:REQ-016

  - id: REQ-009
    text: >
      The canonical pack directory layout is: pack.yml (required), go.mod
      (required), rules/ (required if layer 2 rules exist), fixtures/rules/
      (required, one lowercase subdir per rule ID), standards/ (optional),
      scaffolds/ (required if archetype is code), validators/ (required if
      layer 3 rules exist). ValidateManifest must check that the manifest
      content is consistent with these directory expectations based on
      archetype and declared layers.
    supports: pack-manifest-authoring:REQ-003

  - id: REQ-010
    text: >
      Layer 3 validators must run in process isolation: separate process, no
      network access, no filesystem writes outside pack directory, no
      environment variable access. The manifest types must represent
      input_scope and validator path so that the runtime can enforce isolation.
    supports: pack-manifest-authoring:REQ-012

  - id: REQ-011
    text: >
      pack check runs structural verification and pack test runs fixture
      execution. The manifest schema types must support both workflows: pack
      check validates manifest structure, field presence, enum values,
      archetype constraints, and layer-specific field requirements. pack test
      additionally executes fixtures.
    supports: pack-manifest-authoring:REQ-028

  - id: REQ-012
    text: >
      pack try <project-path> runs the pack's rules against a real codebase
      for author exploration. The manifest types must include sufficient
      metadata (rule paths, tool configs, validator paths) to enable this
      command.
    supports: pack-manifest-authoring:REQ-029

  - id: REQ-013
    text: >
      Code packs must enforce bidirectional co-occurrence: every scaffold
      must have at least one enforcement rule via pairs_with.rules, AND every
      rule must reference at least one scaffold or SDK via pairs_with. A code
      pack scaffold without a paired rule, or a code pack rule without a
      paired scaffold or SDK, is a validation error. Enforcement packs must
      not declare scaffolds or sdk.
    supports: pack-manifest-authoring:REQ-005

  - id: REQ-014
    text: >
      All pack archetypes require mechanical proof via fixtures at their
      tier's expected completeness level. No archetype is exempt from fixture
      requirements. Every rule (in both enforcement and code packs) must have
      at least one claim, and every claim must have fixtures.
    supports: pack-manifest-authoring:REQ-035

claims:
  # REQ-001: Content types and enforcement pack constraints
  - id: CLM-001
    requirement: REQ-001
    text: Enforcement pack with only ruleset is valid
    tests:
      - TestValidateContent_EnforcementRulesetOnly

  - id: CLM-002
    requirement: REQ-001
    text: Enforcement pack declaring scaffolds is rejected
    tests:
      - TestValidateContent_EnforcementWithScaffolds

  - id: CLM-003
    requirement: REQ-001
    text: Enforcement pack declaring sdk is rejected
    tests:
      - TestValidateContent_EnforcementWithSDK

  - id: CLM-004
    requirement: REQ-001
    text: Code pack with ruleset and scaffolds is valid
    tests:
      - TestValidateContent_CodeWithRulesetAndScaffolds

  - id: CLM-005
    requirement: REQ-001
    text: Content block with unknown content type key is rejected
    tests:
      - TestValidateContent_UnknownContentType

  # REQ-002: Layer 2 required fields
  - id: CLM-006
    requirement: REQ-002
    text: Layer 2 rule with both rule and standard fields is accepted
    tests:
      - TestValidateLayer2_BothFields

  - id: CLM-007
    requirement: REQ-002
    text: Layer 2 rule missing rule field is rejected
    tests:
      - TestValidateLayer2_MissingRuleField

  - id: CLM-008
    requirement: REQ-002
    text: Layer 2 rule missing standard field is rejected
    tests:
      - TestValidateLayer2_MissingStandard

  - id: CLM-009
    requirement: REQ-002
    text: Layer 1 rule declaring rule field is rejected
    tests:
      - TestValidateLayer1_WithRuleField

  - id: CLM-010
    requirement: REQ-002
    text: Layer 3 rule declaring rule field is rejected
    tests:
      - TestValidateLayer3_WithRuleField

  # REQ-003: Layer 3 category
  - id: CLM-011
    requirement: REQ-003
    text: Layer 3 rule with category presence is accepted without justification
    tests:
      - TestValidateLayer3Category_Presence

  - id: CLM-012
    requirement: REQ-003
    text: Layer 3 rule with category structural is accepted without justification
    tests:
      - TestValidateLayer3Category_Structural

  - id: CLM-013
    requirement: REQ-003
    text: Layer 3 rule with category other and justification is accepted
    tests:
      - TestValidateLayer3Category_OtherWithJustification

  - id: CLM-014
    requirement: REQ-003
    text: Layer 3 rule with category other missing justification is rejected
    tests:
      - TestValidateLayer3Category_OtherMissingJustification

  - id: CLM-015
    requirement: REQ-003
    text: Layer 3 rule missing category is rejected
    tests:
      - TestValidateLayer3Category_Missing

  - id: CLM-016
    requirement: REQ-003
    text: Layer 3 rule with invalid category is rejected
    tests:
      - TestValidateLayer3Category_Invalid

  - id: CLM-017
    requirement: REQ-003
    text: Layer 1 rule declaring category is rejected
    tests:
      - TestValidateLayer1_WithCategory

  - id: CLM-018
    requirement: REQ-003
    text: Layer 2 rule declaring category is rejected
    tests:
      - TestValidateLayer2_WithCategory

  # REQ-004: Layer 3 input_scope and validator
  - id: CLM-019
    requirement: REQ-004
    text: Layer 3 rule with input_scope single-file and validator is accepted
    tests:
      - TestValidateLayer3_SingleFileWithValidator

  - id: CLM-020
    requirement: REQ-004
    text: Layer 3 rule with input_scope multi-file and validator is accepted
    tests:
      - TestValidateLayer3_MultiFileWithValidator

  - id: CLM-021
    requirement: REQ-004
    text: Layer 3 rule missing input_scope is rejected
    tests:
      - TestValidateLayer3_MissingInputScope

  - id: CLM-022
    requirement: REQ-004
    text: Layer 3 rule missing validator is rejected
    tests:
      - TestValidateLayer3_MissingValidator

  - id: CLM-023
    requirement: REQ-004
    text: Layer 3 rule with invalid input_scope is rejected
    tests:
      - TestValidateLayer3_InvalidInputScope

  - id: CLM-024
    requirement: REQ-004
    text: Layer 1 rule declaring input_scope is rejected
    tests:
      - TestValidateLayer1_WithInputScope

  - id: CLM-025
    requirement: REQ-004
    text: Layer 2 rule declaring input_scope is rejected
    tests:
      - TestValidateLayer2_WithInputScope

  - id: CLM-026
    requirement: REQ-004
    text: Layer 1 rule declaring validator is rejected
    tests:
      - TestValidateLayer1_WithValidator

  - id: CLM-027
    requirement: REQ-004
    text: Layer 2 rule declaring validator is rejected
    tests:
      - TestValidateLayer2_WithValidator

  # REQ-005: Security bypass_attempt
  - id: CLM-028
    requirement: REQ-005
    text: Security-class rule with at least one bypass_attempt fixture is accepted
    tests:
      - TestValidateSecurityFixtures_WithBypass

  - id: CLM-029
    requirement: REQ-005
    text: Security-class rule with no bypass_attempt fixture is rejected
    tests:
      - TestValidateSecurityFixtures_NoBypass

  - id: CLM-030
    requirement: REQ-005
    text: Non-security rule without bypass_attempt fixture is accepted
    tests:
      - TestValidateSecurityFixtures_NonSecurityNoBypass

  # REQ-006: tool_config traceability
  - id: CLM-031
    requirement: REQ-006
    text: Standalone tool_config is traceable via its own id
    tests:
      - TestValidateToolConfigTrace_Standalone

  - id: CLM-032
    requirement: REQ-006
    text: Supporting tool_config is traceable via required_by
    tests:
      - TestValidateToolConfigTrace_Supporting

  # REQ-007: Embedded pack same path
  - id: CLM-033
    requirement: REQ-007
    text: Embedded Go standards pack is parsed via ParseManifest without special-case code
    tests:
      - TestParseManifest_EmbeddedPackSamePath

  # REQ-008: Fixture directory naming
  - id: CLM-034
    requirement: REQ-008
    text: Fixture directory matching rule ID in lowercase is accepted
    tests:
      - TestValidateFixtureDir_MatchesRuleID

  - id: CLM-035
    requirement: REQ-008
    text: Fixture directory not matching rule ID is flagged
    tests:
      - TestValidateFixtureDir_Mismatch

  # REQ-009: Directory layout
  - id: CLM-036
    requirement: REQ-009
    text: Enforcement pack with layer 2 rules expects rules/ directory
    tests:
      - TestExpectedLayout_EnforcementWithLayer2

  - id: CLM-037
    requirement: REQ-009
    text: Pack with layer 3 rules expects validators/ directory
    tests:
      - TestExpectedLayout_WithLayer3

  - id: CLM-038
    requirement: REQ-009
    text: Code pack expects scaffolds/ directory
    tests:
      - TestExpectedLayout_CodePack

  - id: CLM-039
    requirement: REQ-009
    text: Pack always expects fixtures/rules/ directory
    tests:
      - TestExpectedLayout_FixturesAlways

  # REQ-010: Layer 3 isolation fields
  - id: CLM-040
    requirement: REQ-010
    text: Layer 3 rule validator and input_scope are represented in manifest types
    tests:
      - TestLayer3Type_ValidatorAndInputScope

  # REQ-011: Types support pack check and pack test
  - id: CLM-041
    requirement: REQ-011
    text: Manifest types include all fields needed for structural validation
    tests:
      - TestManifestTypes_StructuralFields

  - id: CLM-042
    requirement: REQ-011
    text: Manifest types include all fields needed for fixture execution
    tests:
      - TestManifestTypes_FixtureExecutionFields

  # REQ-012: pack try metadata
  - id: CLM-043
    requirement: REQ-012
    text: Manifest types include rule paths and tool configs needed for pack try
    tests:
      - TestManifestTypes_PackTryMetadata

  # REQ-013: Bidirectional co-occurrence
  - id: CLM-044
    requirement: REQ-013
    text: Code pack with scaffolds and rules (bidirectional pairing) is accepted
    tests:
      - TestValidateCoOccurrence_CodePackValid

  - id: CLM-045
    requirement: REQ-013
    text: Code pack scaffold without any paired rule is rejected
    tests:
      - TestValidateCoOccurrence_ScaffoldNoPairedRule

  - id: CLM-046
    requirement: REQ-013
    text: Code pack rule without pairs_with scaffold or SDK is rejected
    tests:
      - TestValidateCoOccurrence_RuleNoPairedContent

  - id: CLM-047
    requirement: REQ-013
    text: Enforcement pack declaring scaffolds is rejected
    tests:
      - TestValidateCoOccurrence_EnforcementWithScaffolds

  - id: CLM-048
    requirement: REQ-013
    text: Enforcement pack declaring sdk is rejected
    tests:
      - TestValidateCoOccurrence_EnforcementWithSDK

  # REQ-014: Fixture proof for all archetypes
  - id: CLM-049
    requirement: REQ-014
    text: Rule with claims and fixtures is accepted for enforcement pack
    tests:
      - TestValidateFixtureProof_EnforcementWithFixtures

  - id: CLM-050
    requirement: REQ-014
    text: Rule with claims and fixtures is accepted for code pack
    tests:
      - TestValidateFixtureProof_CodeWithFixtures

  - id: CLM-051
    requirement: REQ-014
    text: Rule without any claims is rejected
    tests:
      - TestValidateFixtureProof_RuleNoClaims

contracts:
  - file: pkg/pack/validate_manifest.go
    provides:
      - name: ValidateManifest
        kind: function
        signature: "func ValidateManifest(m *Manifest) []ValidationError"
        notes: "Structural validation: required fields, enums, format constraints, layer-specific rules, archetype constraints, co-occurrence, fixture proof"
      - name: ValidationError
        kind: type
        signature: "type ValidationError struct"
        notes: "Field, Message, Rule string"
      - name: ExpectedLayout
        kind: function
        signature: "func ExpectedLayout(m *Manifest) []string"
        notes: "Returns expected directory entries based on archetype and declared layers"
    consumes:
      - source: pkg/pack/manifest.go
        name: Manifest
        kind: type
      - source: pkg/pack/manifest.go
        name: Rule
        kind: type
      - source: pkg/pack/manifest.go
        name: Scaffold
        kind: type
      - source: pkg/pack/manifest.go
        name: ToolConfigEntry
        kind: type
---

# SPEC-019: Pack Manifest Constraints and Layout

## Overview

This spec defines the constraint and layout validation rules that govern pack manifests. While SPEC-013 covers the types, parsing, and basic enum validation, this spec covers the cross-field constraints, layer-specific field requirements, archetype enforcement, co-occurrence rules, and directory layout expectations.

The scope covers:

- **Layer field constraints** — which fields are required and prohibited per layer (1, 2, 3)
- **Content type restrictions** — what an enforcement pack vs code pack may declare
- **Layer 3 category auto-acceptance** — presence/structural need no justification
- **Security rule fixture requirements** — bypass_attempt on negatives
- **Bidirectional co-occurrence** — scaffold↔rule pairing in code packs
- **Directory layout** — canonical structure enforced by ValidateManifest
- **tool_config traceability** — every entry must trace to a rule
- **Fixture proof** — all archetypes require mechanical proof via fixtures

This spec consumes types from SPEC-013 (pkg/pack/manifest.go) and adds validation logic in pkg/pack/validate_manifest.go.

## Requirements

Requirements are defined in frontmatter. Claims are defined in frontmatter.

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

## Implementation

### Validation Passes (added to ValidateManifest)

1. **Content type allowlist** — only known content types; enforcement packs must not have scaffolds or sdk.
2. **Layer-specific field enforcement** — layer 2 needs rule+standard, layer 3 needs category+validator+input_scope, layers 1/3 must not have rule field, layers 1/2 must not have category/validator/input_scope.
3. **Layer 3 category validation** — presence/structural auto-accepted, other requires justification.
4. **Security bypass_attempt** — security-class rules must have at least one bypass_attempt negative fixture.
5. **tool_config traceability** — every entry is standalone (id) xor supporting (required_by).
6. **Co-occurrence** — code packs require bidirectional scaffold-rule pairing; enforcement packs prohibit scaffolds and sdk.
7. **Fixture proof** — every rule has claims, every claim has fixtures.
8. **Expected layout** — derive expected directories from archetype and layers.

## Verification

Verification is defined in frontmatter. Unit-level verification with 90% coverage threshold.

Test command: `go test ./pkg/pack/ -race -coverprofile=cover.out`

## Sharp Edges

- **Layer field matrix completeness.** Every cell in the layer×field matrix needs testing. It's easy to test that layer 3 requires category but forget to test that layer 2 prohibits it. The dependency matrix rule applies.

- **Co-occurrence is bidirectional.** Every scaffold must have a paired rule AND every rule must pair with a scaffold/SDK. Easy to implement only one direction.

- **Category vs risk_class confusion.** Both are enums on rules. category is layer 3 only; risk_class is all rules. Validators checking category on layer 1/2 rules, or skipping risk_class on layer 3, have a bug.

- **tool_config dual identity.** Standalone tool_config is both a tool configuration AND a layer 1 rule. It must satisfy both sets of requirements.

## References

- **SPEC-013** — pack manifest types and parsing (consumed by this spec)
- **BUNDLE-004** — source bundle, REQ-003 through REQ-035
- **ADR-0001** — agent-first discipline framework
- **BUNDLE-005** — pack validation pipeline (consumes these constraints)
