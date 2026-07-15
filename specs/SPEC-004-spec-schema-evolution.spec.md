---
title: "SPEC-004: Spec Schema Evolution — Standards Binding, Review Questions, Context Injection"
number: SPEC-004
created: "2026-04-03"
status: deprecated
reason: >
  Legacy scaffolding-era cluster retired with the agent-definitions bundle. Per
  terminal-state conventions, deprecated needs no successor pointer.
schema_version: spec/v1
spec_version: 1.0.0

implementation:
  summary: >
    Extend the spec artifact schema and validator to support optional standards
    binding on requirements (follows field) and an optional Review Questions
    section for adversarial impl-reviewer prompts. Update the spec-author and
    impl-reviewer agent definitions with standards-binding and review-question
    instructions. Add a SessionStart hook that reads compiled manifests from
    .backstop/rules/ and injects available standards as agent context.
  package: pkg/validate

verification:
  level: integration
  test_command: go test ./... -run "TestSpec_|TestSpecAuthorAgent_|TestImplReviewerAgent_|TestStandardsHook_|TestSettingsJson_" -race -coverprofile=cover.out
  coverage_threshold: 80

requirements:
  - id: REQ-001
    text: >
      The spec schema must support an optional follows field on each
      requirement. When present, follows must be a non-empty string or
      array of strings. Each value must match the format STD-LANG-NNN:RULE-ID
      (e.g., STD-GO-001:GO-010) or be a plain recipe name matching the
      pattern ^[a-z][a-z0-9]*(-[a-z0-9]+)*$. An empty follows field or a
      value matching neither format is a validation error.
    supports: agent-definitions:REQ-014

  - id: REQ-002
    text: >
      When a requirement does not include a follows field, validation must
      pass without error. The follows field is strictly optional.
    supports: agent-definitions:REQ-014

  - id: REQ-003
    text: >
      The spec schema must support an optional Review Questions section in the
      markdown body. The optional_sections list in the schema JSON must include
      "Review Questions". The validator checks for section heading presence
      only — content quality validation is not possible because the parser
      tracks section headings but not section content. Content quality is
      the spec-reviewer's responsibility.
    supports: agent-definitions:REQ-015

  - id: REQ-004
    text: >
      When a spec does not include a Review Questions section, validation must
      pass without error. The section is strictly optional.
    supports: agent-definitions:REQ-015

  - id: REQ-005
    text: >
      The spec-author agent definition (.claude/agents/spec-author.md) must
      include instructions to bind applicable standards to requirements using
      the follows field, to generate a Review Questions section with
      adversarial questions for the impl-reviewer, and to escalate to the
      human when standards do not cover the exact case rather than guessing
      (DD-13 escalation-over-guessing pattern).
    supports: agent-definitions:REQ-016

  - id: REQ-006
    text: >
      The impl-reviewer agent definition (.claude/agents/impl-reviewer.md)
      must include instructions to check review questions from the spec
      during code review and to verify that implementations follow the
      specific standard rules referenced in requirement follows fields,
      in addition to claim verification.
    supports: agent-definitions:REQ-017

  - id: REQ-007
    text: >
      A standards context hook script must exist that reads compiled manifests
      from .backstop/rules/*.manifest.json and outputs a summary of
      available standards (standard ID, language, rule count) as context
      for the agent. The hook must gracefully handle the case where the
      .backstop/rules/ directory does not exist or contains no manifests.
    supports: agent-definitions:REQ-018

  - id: REQ-008
    text: >
      The standards context hook must be registered in .claude/settings.json
      under both hooks.SessionStart and hooks.SubagentStart using the
      required Claude Code hook format: each entry must be an object with
      a "matcher" string (empty string to match all) and a "hooks" array
      containing the command objects. Bare command objects without the
      matcher+hooks wrapper are rejected by Claude Code at startup.
    supports: agent-definitions:REQ-018

  - id: REQ-009
    text: >
      The follows field values with the standard rule format must use the
      pattern ^STD-[A-Z]+-[0-9]{3}:[A-Z]+-[0-9]+$ to match the standard
      numbering convention (e.g., STD-GO-001:GO-010, STD-JAVA-001:J-042).
      Values not matching the standard rule format must match the recipe
      name pattern ^[a-z][a-z0-9]*(-[a-z0-9]+)*$ instead. A value matching
      neither pattern is a validation error.
    supports: agent-definitions:REQ-014

  - id: REQ-010
    text: >
      All existing spec validation rules must continue to function: filename
      pattern, slug validation, number/filename consistency, title/number
      consistency, status enum, extension metadata, schema_version cross-check,
      verification block, implementation block, requirements array, claims
      array, contracts, and capabilities.

claims:
  - id: CLM-001
    requirement: REQ-001
    text: Requirement with a single valid standard rule follows value passes validation
    tests:
      - TestSpec_Follows_SingleStandardRule

  - id: CLM-002
    requirement: REQ-001
    text: Requirement with a single valid recipe name follows value passes validation
    tests:
      - TestSpec_Follows_SingleRecipeName

  - id: CLM-003
    requirement: REQ-001
    text: Requirement with an array of valid follows values passes validation
    tests:
      - TestSpec_Follows_ArrayValid

  - id: CLM-004
    requirement: REQ-001
    text: Requirement with an empty follows string fails validation
    tests:
      - TestSpec_Follows_EmptyStringFails

  - id: CLM-005
    requirement: REQ-001
    text: Requirement with an empty follows array fails validation
    tests:
      - TestSpec_Follows_EmptyArrayFails

  - id: CLM-006
    requirement: REQ-001
    text: Requirement with an invalid follows value fails validation
    tests:
      - TestSpec_Follows_InvalidFormatFails

  - id: CLM-007
    requirement: REQ-002
    text: Requirement without follows field passes validation
    tests:
      - TestSpec_Follows_OmittedPasses

  - id: CLM-008
    requirement: REQ-003
    text: Spec with non-empty Review Questions section passes validation
    tests:
      - TestSpec_ReviewQuestions_PresentPasses

  - id: CLM-009
    requirement: REQ-003
    text: >
      Spec with empty Review Questions section passes validation (parser
      limitation — content checking not possible, presence-only validation)
    tests:
      - TestSpec_ReviewQuestions_EmptyPassesDueToParserLimitation

  - id: CLM-010
    requirement: REQ-004
    text: Spec without Review Questions section passes validation
    tests:
      - TestSpec_ReviewQuestions_OmittedPasses

  - id: CLM-011
    requirement: REQ-005
    text: >
      The spec-author agent definition includes instructions for standards
      binding via follows field, review question generation, and escalation
      when standards are ambiguous
    tests:
      - TestSpecAuthorAgent_ContainsFollowsInstructions
      - TestSpecAuthorAgent_ContainsReviewQuestionsInstructions
      - TestSpecAuthorAgent_ContainsEscalationInstructions

  - id: CLM-012
    requirement: REQ-006
    text: >
      The impl-reviewer agent definition includes instructions to check
      review questions and follows-field standard rules during code review
    tests:
      - TestImplReviewerAgent_ContainsReviewQuestionsInstructions
      - TestImplReviewerAgent_ContainsFollowsCheckInstructions

  - id: CLM-013
    requirement: REQ-007
    text: >
      The standards context hook reads manifests and outputs a summary of
      available standards
    tests:
      - TestStandardsHook_OutputsManifestSummary

  - id: CLM-014
    requirement: REQ-007
    text: >
      The standards context hook handles missing .backstop/rules/ directory
      gracefully (exits 0, no error output)
    tests:
      - TestStandardsHook_MissingDirectoryGraceful

  - id: CLM-015
    requirement: REQ-007
    text: >
      The standards context hook handles empty .backstop/rules/ directory
      gracefully (exits 0, no error output)
    tests:
      - TestStandardsHook_EmptyDirectoryGraceful

  - id: CLM-016
    requirement: REQ-008
    text: >
      The settings.json file registers the standards context hook under
      both hooks.SessionStart and hooks.SubagentStart using the correct
      matcher+hooks array format
    tests:
      - TestSettingsJson_SessionStartHookRegistered
      - TestSettingsJson_SubagentStartHookRegistered
      - TestSettingsJson_SessionStartHookFormat
      - TestSettingsJson_SubagentStartHookFormat

  - id: CLM-017
    requirement: REQ-009
    text: >
      Follows value matching STD-LANG-NNN:RULE-ID pattern passes validation
      (e.g., STD-GO-001:GO-010)
    tests:
      - TestSpec_Follows_StandardRuleFormatPasses

  - id: CLM-018
    requirement: REQ-009
    text: >
      Follows value matching recipe name pattern passes validation
      (e.g., error-handling-recipe)
    tests:
      - TestSpec_Follows_RecipeNameFormatPasses

  - id: CLM-019
    requirement: REQ-009
    text: >
      Follows value matching neither standard rule nor recipe name pattern
      fails validation (e.g., INVALID, 123-bad, STD-go-001:go-010)
    tests:
      - TestSpec_Follows_NeitherFormatFails

  - id: CLM-020
    requirement: REQ-010
    text: >
      Representative existing spec validation rules still function after
      follows and review questions additions: filename pattern, requirements
      array, claims array, verification block
    tests:
      - TestSpec_ExistingRules_FilenamePatternsStillWork
      - TestSpec_ExistingRules_RequirementsArrayStillValidated
      - TestSpec_ExistingRules_ClaimsArrayStillValidated
      - TestSpec_ExistingRules_VerificationBlockStillValidated

  - id: CLM-021
    requirement: REQ-001
    text: >
      Requirement with follows containing a mix of valid and invalid values
      fails validation — all values must be valid
    tests:
      - TestSpec_Follows_MixedValidInvalidFails

  - id: CLM-022
    requirement: REQ-009
    text: >
      Follows value with lowercase standard prefix fails validation
      (e.g., std-go-001:GO-010 — standard prefix must be uppercase)
    tests:
      - TestSpec_Follows_LowercaseStdPrefixFails

  - id: CLM-023
    requirement: REQ-009
    text: >
      Follows value with uppercase recipe name fails validation
      (e.g., Error-Handling — recipe names must be lowercase kebab)
    tests:
      - TestSpec_Follows_UppercaseRecipeNameFails

contracts:
  - file: pkg/validate/spec.go
    provides:
      - name: Spec
        kind: function
        signature: "func Spec(art *artifact.ParsedArtifact, sch *schema.Schema) ValidationResult"
        notes: "Extended with follows field validation on requirements and Review Questions section validation"
    consumes:
      - source: pkg/artifact
        name: ParsedArtifact
        kind: type
      - source: pkg/schema
        name: Schema
        kind: type

  - file: artifacts/spec/v1/schema.json
    provides:
      - name: spec-schema
        kind: constant
        signature: "JSON schema definition"
        notes: "Updated: requirements item_optional_keys adds 'follows', optional_sections adds 'Review Questions'"
    consumes: []

  - file: .claude/agents/spec-author.md
    provides:
      - name: spec-author-agent
        kind: constant
        signature: "Agent definition markdown"
        notes: "Updated with standards binding and review question generation instructions"
    consumes: []

  - file: .claude/agents/impl-reviewer.md
    provides:
      - name: impl-reviewer-agent
        kind: constant
        signature: "Agent definition markdown"
        notes: "Updated with review questions checking instructions"
    consumes: []

  - file: .claude/hooks/backstop-standards-context.sh
    provides:
      - name: standards-context-hook
        kind: constant
        signature: "Shell script"
        notes: "SessionStart hook that reads .backstop/rules/*.manifest.json and outputs standards summary"
    consumes: []

  - file: .claude/settings.json
    provides:
      - name: settings-config
        kind: constant
        signature: "JSON configuration"
        notes: "Updated with SessionStart and SubagentStart hook registration"
    consumes: []
---

# SPEC-004: Spec Schema Evolution — Standards Binding, Review Questions, Context Injection

## Overview

The spec validator (pkg/validate/spec.go) validates spec artifacts against the spec
schema. Currently it handles requirements with optional `supports` fields for bundle
traceability, but has no mechanism for binding requirements to specific standard rules
or for validating adversarial review questions.

This spec adds three capabilities:

1. **Standards binding via `follows` field** — requirements can reference specific
   standard rules (STD-GO-001:GO-010) or recipe names (error-handling-recipe) that
   constrain their implementation. This gives implementers precise references and
   gives reviewers specific rules to check against.

2. **Review Questions section** — an optional markdown body section where the spec
   author writes adversarial questions for the impl-reviewer. These capture risks
   the author already sees that claims alone do not express.

3. **Standards context injection** — a SessionStart hook that reads compiled
   manifests from .backstop/rules/ and injects available standards as context.
   Agents do not discover standards manually; the hook tells them what is available.

Additionally, the spec-author and impl-reviewer agent definitions are updated with
instructions for using these features.

## Requirements

Requirements are defined in frontmatter. Key design decisions from the bundle:

- **DD-10: Standards bound at requirement level.** The `follows` field traces from
  a specific standard rule down to the requirement that uses it, not just the spec
  level. This gives implementers and reviewers precise references.

- **DD-11: Spec-level review questions.** Adversarial questions authored during
  spec creation that the impl-reviewer must check. Captures risk the spec author
  sees that claims alone do not express.

- **DD-12: Hook-injected standards context.** SessionStart hooks read compiled
  manifests and inject available standards as agent context. Agents do not discover
  standards manually.

- **DD-13: Escalation over guessing.** When standards do not cover the exact case,
  agents escalate to the human rather than improvising.

### Follows Field Format

The `follows` field accepts two formats:

| Format | Pattern | Example |
|--------|---------|---------|
| Standard rule reference | `^STD-[A-Z]+-[0-9]{3}:[A-Z]+-[0-9]+$` | `STD-GO-001:GO-010` |
| Recipe name | `^[a-z][a-z0-9]*(-[a-z0-9]+)*$` | `error-handling-recipe` |

A value must match one of these two patterns. Values matching neither are rejected.
The field accepts a single string or an array of strings. When present, it must be
non-empty.

### Review Questions Section

The Review Questions section is an optional markdown body section. When present, it
must contain at least one question (non-empty content after the heading). When absent,
validation passes without error.

## Implementation

### Changes to spec schema (artifacts/spec/v1/schema.json)

1. **Add `follows` to requirement optional keys** — update the `requirements`
   nested block's `item_optional_keys` to include `"follows"` alongside the
   existing `"supports"`.

2. **Add `Review Questions` to optional sections** — add `"Review Questions"` to
   the `optional_sections` array.

### Changes to spec validator (pkg/validate/spec.go)

3. **Follows field validation** — in `validateRequirements`, after the existing
   `supports` validation, add a block that checks for the `follows` field on
   each requirement. When present:
   - Accept a single string or array of strings (same pattern as `supports`)
   - Reject empty string or empty array
   - Validate each value against the standard rule regex
     (`^STD-[A-Z]+-[0-9]{3}:[A-Z]+-[0-9]+$`) or the recipe name regex
     (`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`). Values matching neither are rejected.

4. **Review Questions section validation** — add a new validation pass after
   capabilities validation that checks: if a "Review Questions" section exists
   in the markdown body, it must have non-empty content. This uses the existing
   `art.Sections` map.

### Changes to agent definitions

5. **spec-author.md** — add a section instructing the spec author to:
   - Check available standards (injected by the SessionStart hook)
   - Bind applicable standard rules to requirements using the `follows` field
   - Generate a Review Questions section with adversarial questions targeting
     risks the claims alone do not cover
   - Escalate to the human when standards do not cover the exact case (DD-13)

6. **impl-reviewer.md** — add a review step instructing the impl-reviewer to:
   - Check the spec's Review Questions section during code review
   - Evaluate each question against the implementation
   - Report which review questions are satisfied and which are not
   - Check that implementations follow the specific standard rules referenced
     in requirement `follows` fields

### Changes to hooks

7. **backstop-standards-context.sh** — new SessionStart hook that:
   - Reads all `.manifest.json` files from `.backstop/rules/`
   - Extracts standard ID, language, and rule count from each manifest
   - Outputs a formatted summary (e.g., "Available standards: STD-GO-001 (go,
     14 rules)")
   - Exits 0 with no output if the directory does not exist or is empty

8. **settings.json** — register the new hook under `hooks.SessionStart`.

### New regex patterns (pkg/validate/spec.go)

```go
var (
    followsStdRuleRe = regexp.MustCompile(`^STD-[A-Z]+-[0-9]{3}:[A-Z]+-[0-9]+$`)
    followsRecipeRe  = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)
)
```

## Verification

Verification is defined in frontmatter. Integration-level verification with 80%
coverage threshold across the spec validator, agent definitions, and hook scripts.

Claims are defined in frontmatter. Each claim maps a requirement to specific test
functions.

Tests for validator changes (CLM-001 through CLM-010, CLM-017 through CLM-023) run
in `pkg/validate/spec_test.go`. Tests for agent definitions (CLM-011, CLM-012) and
hook/settings changes (CLM-013 through CLM-016) run as file-content verification
tests that read the files and assert expected content is present.

## Sharp Edges

- **Follows format ambiguity with recipe names.** The recipe name pattern
  `^[a-z][a-z0-9]*(-[a-z0-9]+)*$` is intentionally broad. A value like `go-010`
  matches the recipe pattern but could be confused for a rule ID missing its
  standard prefix. The validator does not distinguish intent — if it matches
  either pattern, it passes. Spec authors must use the full `STD-LANG-NNN:RULE-ID`
  format when referencing standard rules.

- **Review Questions section is presence-validated only.** The parser tracks section
  headings but not section content, so the validator can only check that the heading
  exists. An empty Review Questions section passes validation. Content quality —
  including whether actual questions exist — is the spec-reviewer's responsibility.
  This is a known parser limitation, not a design choice.

- **SessionStart hook depends on compiled manifests existing.** If the standards
  compiler has not been run, `.backstop/rules/` will be empty or missing. The hook
  handles this gracefully (exits 0), but the agent will have no standards context.
  This is a silent degradation — the agent will not know standards are available.

- **Hook output format is not structured.** The SessionStart hook outputs
  human-readable text, not JSON. If a future consumer needs structured standards
  metadata, the hook output format will need to change. For now, the audience is
  agent context injection where human-readable is appropriate.

- **Follows field is not cross-validated against compiled manifests.** The validator
  checks format only. A follows value of `STD-GO-001:GO-999` passes validation even
  if rule GO-999 does not exist in the compiled manifest. Cross-validation against
  actual manifests is a future enhancement — it requires the validator to have
  access to the compiled rules directory, which is not currently part of its API.

- **Claude Code settings.json hook format is strict.** Every hook event entry
  requires the `{"matcher": "...", "hooks": [...]}` wrapper structure. Bare
  command objects placed directly in the event array cause Claude Code to reject
  the entire settings file at startup with no fallback. This was a real failure
  during SPEC-004 implementation — the SessionStart and SubagentStart hooks were
  written as bare objects and Claude Code refused to load the file. Tests must
  validate the structural format, not just the presence of the command string.

- **Agent instruction changes are not mechanically enforced.** REQ-005 and REQ-006
  require specific instructions in agent definition files. The claims verify the
  content is present via grep-style tests, but there is no runtime guarantee that
  agents follow these instructions. Instructions are soft constraints.

- **Backward compatibility for specs without follows.** Specs written before this
  change have no follows fields. Unlike the plan schema evolution (SPEC-002) where
  plans without type fields are rejected, specs without follows pass validation
  because the field is optional. No migration needed.

## Review Questions

- Does the follows field validation correctly handle the case where a value starts
  with `STD-` but does not match the full standard rule pattern? It should fail,
  not fall through to the recipe pattern.
- Does the SessionStart hook correctly parse manifests with unexpected JSON structure
  (missing fields, extra fields)?
- Are the spec-author instructions specific enough that an agent knows when to use
  `follows` versus when a standard is not applicable?
- Does the impl-reviewer actually check follows references against the implementation,
  or just report them?

## References

- Bundle: agent-definitions v0.3.0 (spec schema evolution seed, REQ-014 through REQ-018)
- DD-10: Standards bound at requirement level via follows field
- DD-11: Spec-level review questions for impl-reviewer
- DD-12: Hook-injected standards context from compiled manifests
- DD-13: Escalation over guessing when standards are ambiguous
- SPEC-001: Standards Compiler (manifest format reference)
- SPEC-002: Plan Schema Evolution (spec format reference)
- ADR-0018: Workflow State Machine
