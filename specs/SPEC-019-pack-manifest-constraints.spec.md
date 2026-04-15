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
      ValidateManifest must verify that layer 3 rules declare both
      input_scope and validator fields, which are required for the runtime
      to enforce process isolation. This is a constraint check, not a type
      definition — the types are defined in SPEC-013.
    supports: pack-manifest-authoring:REQ-012

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

  - id: CLM-033
    requirement: REQ-006
    text: tool_config with neither id nor required_by is rejected by traceability check
    tests:
      - TestValidateToolConfigTrace_NeitherIdNorRequiredBy

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
  - id: CLM-052
    requirement: REQ-009
    text: ExpectedLayout always includes pack.yml
    tests:
      - TestExpectedLayout_PackYmlAlways

  - id: CLM-053
    requirement: REQ-009
    text: ExpectedLayout always includes go.mod for Go packs
    tests:
      - TestExpectedLayout_GoModAlways

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
    text: ValidateManifest rejects layer 3 rule missing input_scope or validator fields
    tests:
      - TestValidateConstraint_Layer3MissingIsolationFields

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
    text: Co-occurrence pass independently rejects enforcement pack with scaffolds (distinct from content type check in REQ-001)
    tests:
      - TestValidateCoOccurrence_EnforcementWithScaffolds

  - id: CLM-048
    requirement: REQ-013
    text: Co-occurrence pass independently rejects enforcement pack with sdk (distinct from content type check in REQ-001)
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

SPEC-013 defines what a pack manifest **is** — the Go types, YAML parsing, and enum validation. This spec defines what a pack manifest **must obey** — the cross-field constraints, layer-specific field rules, archetype restrictions, co-occurrence invariants, and directory layout expectations that `ValidateManifest` enforces.

The boundary is deliberate: types and parsing are stable; constraint rules evolve as the pack ecosystem grows. Separating the two lets contributors change validation logic without touching the data model.

This spec covers:

- **Layer field constraints** — which fields are required, optional, or prohibited per enforcement layer (1, 2, 3)
- **Archetype content restrictions** — what enforcement packs vs code packs may declare
- **Layer 3 category validation** — auto-acceptance for `presence`/`structural`, justification required for `other`
- **Security fixture requirements** — `bypass_attempt` on negative fixtures for security-class rules
- **Bidirectional co-occurrence** — every scaffold must pair with a rule, and every rule must pair with a scaffold or SDK
- **tool_config traceability** — every entry must be traceable to at least one rule
- **Fixture proof** — all archetypes require mechanical proof via fixtures
- **Directory layout** — canonical structure derived from archetype and declared layers

All validation logic lives in `pkg/pack/validate_manifest.go`. It consumes the types defined in SPEC-013's `pkg/pack/manifest.go`.

## Requirements

Requirements and claims are defined in frontmatter.

### Layer Field Requirements Matrix

Each enforcement layer permits a different set of fields. "Required" means the validator rejects a rule missing the field. "Prohibited" means the validator rejects a rule declaring it. This matrix drives REQ-002, REQ-003, and REQ-004.

| Field              | Layer 1    | Layer 2    | Layer 3                    |
|--------------------|------------|------------|----------------------------|
| risk_class         | Required   | Required   | Required                   |
| layer              | Required   | Required   | Required                   |
| rule (semgrep path)| Prohibited | Required   | Prohibited                 |
| standard           | Optional   | Required   | Optional                   |
| category           | Prohibited | Prohibited | Required                   |
| justification      | N/A        | N/A        | Required if category=other |
| input_scope        | Prohibited | Prohibited | Required                   |
| validator          | Prohibited | Prohibited | Required                   |

Every cell in this matrix has a corresponding claim. Fields marked "Prohibited" on a given layer are tested with a negative claim that rejects the field's presence. Fields marked "Required" are tested with both a positive claim (field present, accepted) and a negative claim (field missing, rejected).

### Archetype Content Constraints

The `content` block declares what a pack ships. Enforcement packs exist to enforce rules against existing code; they must not bundle scaffolds or SDK. Code packs pair scaffolds with enforcement rules.

| Content Type   | enforcement | code    |
|----------------|-------------|---------|
| ruleset        | Allowed     | Allowed |
| scaffolds      | Prohibited  | Allowed |
| sdk            | Prohibited  | Allowed |
| contracts      | Allowed     | Allowed |
| test_patterns  | Allowed     | Allowed |
| ast_checks     | Allowed     | Allowed |
| rubrics        | Allowed     | Allowed |

An unknown content type key (not in this table) is always rejected.

## Implementation

`ValidateManifest` runs a series of passes over a parsed `*Manifest`. Each pass produces zero or more `ValidationError` values. The passes are ordered so that earlier passes catch structural problems before later passes assume field presence.

### Pass 1: Content Type Allowlist

Verify every key in the `content` block is one of the seven known types. Then check archetype restrictions: enforcement packs must not declare `scaffolds` or `sdk`. Reject unknown keys immediately.

### Pass 2: Layer-Specific Field Enforcement

Iterate all rules and enforce the Layer Field Requirements Matrix above. Layer 2 rules must have both `rule` (semgrep YAML path) and `standard`. Layer 3 rules must have `category`, `input_scope`, and `validator`. Layer 1 and layer 3 rules must not declare `rule`. Layer 1 and layer 2 rules must not declare `category`, `input_scope`, or `validator`.

### Pass 3: Layer 3 Category Validation

For layer 3 rules, validate the `category` enum (`presence`, `structural`, `other`). Categories `presence` and `structural` require no justification. Category `other` requires a non-empty `justification` field. A missing or unrecognized category is rejected.

### Pass 4: Security Bypass Attempt

For rules with `risk_class: security`, verify at least one negative fixture has `bypass_attempt: true`. Non-security rules are not subject to this check. This ensures security rules are tested against adversarial inputs.

### Pass 5: tool_config Traceability

Every `tool_config` entry must be traceable to a rule. An entry is traceable if it is standalone (has an `id`, making it its own layer 1 rule) or supporting (has `required_by`, referencing another rule). An entry that is both or neither is rejected.

### Pass 6: Bidirectional Co-occurrence

Code packs require that every scaffold references at least one rule via `pairs_with.rules`, and every rule references at least one scaffold or SDK via `pairs_with`. A code pack scaffold without a paired rule, or a code pack rule without a paired scaffold or SDK, is a validation error. Enforcement packs must not declare `scaffolds` or `sdk` (redundant with Pass 1, but co-occurrence logic must also reject this to avoid silently skipping the check).

### Pass 7: Fixture Proof

Every rule must have at least one claim, and every claim must have fixtures. No archetype is exempt. This pass does not check fixture content — only that the manifest declares the required structure.

### Pass 8: Expected Layout

`ExpectedLayout` derives the expected directory entries from the manifest's archetype and declared layers:

- `pack.yml` — always required
- `go.mod` — always required (language-dependent in the future)
- `rules/` — required if any layer 2 rules exist
- `fixtures/rules/` — always required, with one lowercase subdirectory per rule ID
- `standards/` — optional
- `scaffolds/` — required if archetype is `code`
- `validators/` — required if any layer 3 rules exist

### Fixture Directory Naming

Fixture directories must use lowercase names matching rule IDs (e.g., `fixtures/rules/err-001/`). A mismatch between the directory name and the rule ID is flagged.

### Embedded Pack Support

The embedded Go standards pack must be loadable through the same `ParseManifest` path as third-party packs. No special-case code paths for embedded packs.

### Command Metadata Support

The manifest types must carry sufficient metadata for three CLI workflows:
- **`pack check`** — structural validation (field presence, enum values, archetype constraints, layer rules)
- **`pack test`** — fixture execution (fixture paths, validator paths, test commands)
- **`pack try <project-path>`** — rule execution against a real codebase (rule paths, tool configs, validator paths)

### Layer 3 Process Isolation

Layer 3 validators run in process isolation: separate process, no network access, no filesystem writes outside the pack directory, no environment variable access. The manifest types represent `input_scope` and `validator` path so the runtime can enforce these constraints.

## Verification

Verification is defined in frontmatter. Unit-level verification targeting `pkg/pack/` with 90% coverage threshold.

Test command: `go test ./pkg/pack/ -race -coverprofile=cover.out`

## Sharp Edges

- **Layer field matrix completeness.** The matrix has 8 fields across 3 layers — 24 cells. It is easy to test that layer 3 requires `category` but forget to test that layer 2 prohibits it. Every cell must have a corresponding claim. This is the dependency matrix rule in action.

- **Bidirectional co-occurrence is two checks, not one.** Scaffold-to-rule and rule-to-scaffold are independent validations. Implementing only one direction (e.g., checking that scaffolds reference rules but not that rules reference scaffolds) leaves half the invariant unenforced. Both directions must be tested.

- **`category` vs `risk_class` confusion.** Both are string enums on rules, but they serve different purposes and have different scope. `risk_class` applies to all rules on all layers. `category` applies only to layer 3. A validator that checks `category` on layer 1 rules, or forgets to check `risk_class` on layer 3 rules, has a bug.

- **`tool_config` dual identity.** A standalone `tool_config` entry is simultaneously a tool configuration and a layer 1 rule. It must satisfy tool_config structural requirements (tool, file) AND rule requirements (risk_class, claims with fixtures). The validation must apply both sets of checks.

- **Enforcement pack prohibition is checked in two places.** Pass 1 (content type allowlist) and Pass 6 (co-occurrence) both reject `scaffolds`/`sdk` on enforcement packs. If only one pass runs (e.g., early return), the other prohibition is skipped. Both passes must execute independently.

## Review Questions

1. Does `ValidateManifest` accumulate all errors or stop at the first? If it stops early, later passes never run, and some constraint violations go unreported. Verify the implementation collects errors across all passes.

2. When a standalone `tool_config` has `id` but is missing `risk_class`, does the error message identify it as a tool_config validation failure or a rule validation failure? Contributors need clear diagnostics.

3. If a layer 3 rule declares `category: other` with an empty string `justification: ""`, does the validator reject it? Empty string is not the same as absent in Go. The check must test for non-empty, not just present.

4. For fixture directory naming (REQ-008), what happens when a rule ID contains uppercase in the manifest but the directory is correctly lowercase? The rule ID validation in SPEC-013 rejects uppercase IDs, so this case may be unreachable — but the interaction between the two specs should be explicit.

5. Does `ExpectedLayout` return paths that match the OS path separator, or does it always use forward slashes? Pack portability depends on consistent path representation.

6. In bidirectional co-occurrence, if a scaffold's `pairs_with.rules` references a rule ID that does not exist in the manifest, is that caught by this spec or deferred to a later resolution pass? The boundary between structural validation and semantic resolution should be clear.

## References

- **SPEC-013** — Pack manifest types and parsing. Defines the Go types that this spec's validation logic consumes. The two specs share `pkg/pack/` but have distinct responsibilities: SPEC-013 owns the data model, SPEC-019 owns the constraint rules.
- **BUNDLE-004** (pack-manifest-authoring) — Source bundle. Requirements REQ-003 through REQ-035 are the origin of the constraints in this spec.
- **BUNDLE-005** (pack-validation-pipeline) — Consumer of these constraints. The validation pipeline invokes `ValidateManifest` as part of `pack check`.
- **ADR-0001** — Agent-first discipline framework. Pack manifests are optimized for machine generation and parsing.
