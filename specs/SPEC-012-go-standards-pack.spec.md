---
title: "SPEC-012: Go Standards Pack — Bootstrap Rule Content"
number: SPEC-012
created: "2026-04-04"
status: draft
schema_version: spec/v1
spec_version: 1.0.0

implementation:
  summary: >
    Populate the Go standards pack (standards/go/) with enforceable rule content
    across three bootstrap categories: core, test, and security. The existing
    STD-GO-001 standard artifact defines the rule metadata (IDs, categories,
    detection strategies, severity, compliance tiers). This spec covers
    authoring the semgrep YAML rule files in standards/go/rules/{core,test,security}/,
    the testdata fixtures in standards/go/testdata/{valid,invalid}/ that prove
    each rule works, and updating STD-GO-001 with any new rules ported from
    the mechsuit go-conventions.md source. After implementation, `backstop pack
    compile` produces working enforcement manifests and `backstop code check`
    can enforce Go standards against real code. The remaining seven categories
    (performance, observability, integration, contracts, concurrency,
    resilience, accessibility) are scaffolded but empty — content is deferred
    to future specs.
  package: standards/go

verification:
  level: integration
  test_command: go test ./pkg/compile/ -run TestGoStandard -race -coverprofile=cover.out && semgrep --config standards/go/rules/ --test standards/go/testdata/
  coverage_threshold: 80

requirements:
  - id: REQ-001
    text: >
      The core category (standards/go/rules/core/) must contain semgrep YAML
      rules enforcing the following Go conventions: no global mutable state
      (GO-003), no init() functions (new rule GO-004), dependency injection
      via constructors (new rule GO-005), and structured logging required
      (new rule GO-006). Each rule must have a unique rule ID prefixed with
      go.core., a message explaining the violation, severity matching
      STD-GO-001, and backstop metadata (category, rule_id, compliance_tier).
    follows:
      - STD-GO-001:GO-003

  - id: REQ-002
    text: >
      The core category must also contain semgrep YAML rules for: no ignored
      errors (GO-010), error wrapping required (GO-011), no naked returns in
      functions longer than 5 lines (GO-012), no panic in library code
      (GO-013), and error type suffix (GO-021). GO-010 has strategy: pattern
      in STD-GO-001. GO-012 has strategy: pattern in STD-GO-001 but its
      function-length constraint cannot be expressed in semgrep; the semgrep
      rule must include a note field explaining this limitation. GO-021 has
      strategy: regex in STD-GO-001 and produces a semgrep YAML entry using
      pattern-regex. GO-020 (no-stuttering-exports) has strategy: delegated
      in STD-GO-001 and is enforced by golangci-lint; it must NOT appear in
      any semgrep YAML rule file. GO-020 appears only in the compiled
      manifest via the compiler. GO-040 (import-ordering) has strategy:
      delegated in STD-GO-001 and is enforced by golangci-lint (goimports);
      it must NOT appear in any semgrep YAML rule file. GO-040 appears only
      in the compiled manifest via the compiler.
    follows:
      - STD-GO-001:GO-010
      - STD-GO-001:GO-011
      - STD-GO-001:GO-012
      - STD-GO-001:GO-013
      - STD-GO-001:GO-020
      - STD-GO-001:GO-021
      - STD-GO-001:GO-040

  - id: REQ-003
    text: >
      The test category (standards/go/rules/test/) must address three rules:
      every source file has a corresponding test file (GO-030, metric strategy),
      table-driven tests with t.Run() for multiple cases (GO-031, pattern
      strategy with note), and no time.Sleep in tests (GO-032, pattern strategy
      semgrep rule). GO-032 must be scoped to _test.go files only. GO-030 uses
      metric detection (not semgrep) and must include a note documenting the
      limitation. GO-031 uses pattern detection with a note acknowledging that
      multi-assertion detection cannot be fully expressed as a single semgrep
      pattern. Only GO-032 produces a semgrep YAML rule; GO-030 and GO-031
      are enforced through their respective strategies in the manifest.
    follows:
      - STD-GO-001:GO-030
      - STD-GO-001:GO-031
      - STD-GO-001:GO-032

  - id: REQ-004
    text: >
      The security category (standards/go/rules/security/) must contain
      semgrep YAML rules enforcing at minimum: no hardcoded credentials
      (new rule GO-060), no use of crypto/md5 or crypto/sha1 for password
      hashing (new rule GO-061), SQL injection prevention via parameterized
      queries (new rule GO-062), and no sensitive data in log messages
      (new rule GO-063). Each rule must include backstop metadata with
      tier set to baseline and references to the applicable CWE or OWASP
      identifier in the metadata.

  - id: REQ-005
    text: >
      Every semgrep rule file must follow the standard semgrep YAML format
      with a top-level rules array. Each rule entry must contain: id
      (format: go.<category>.<kebab-name>), patterns or pattern field,
      message, severity (ERROR, WARNING, or INFO), languages set to [go],
      and a metadata block containing backstop-specific fields (category,
      rule_id, compliance_tier, rationale). Two ID fields are required:
      (1) the semgrep `id` field uses the format go.<category>.<kebab-name>
      (e.g., go.core.no-global-mutable-state), and (2) the
      `metadata.backstop.rule_id` field maps to the STD-GO-001 rule ID
      (e.g., GO-003). Both fields must be present in every rule entry.

  - id: REQ-006
    text: >
      The testdata/invalid/ directory must contain at least one Go source
      file per semgrep rule. Each invalid fixture must trigger exactly the
      rule it tests. The filename must follow the pattern
      <rule-id>-<short-description>.go (e.g., go-003-global-mutable-state.go).
      Invalid fixtures must be syntactically valid Go that compiles but
      violates the rule.

  - id: REQ-007
    text: >
      The testdata/valid/ directory must contain at least one Go source
      file demonstrating the correct pattern for each rule. Valid fixtures
      must pass all rules without violations. The filename must follow the
      pattern <rule-id>-<short-description>.go (e.g.,
      go-003-dependency-injection.go).

  - id: REQ-008
    text: >
      STD-GO-001 must be updated to include new rules ported from mechsuit
      go-conventions.md that are not already present: GO-004
      (no-init-functions, core, error, baseline), GO-005
      (constructor-injection-required, core, warning, standard), GO-006
      (structured-logging-required, core, warning, standard), GO-060
      (no-hardcoded-credentials, security, error, baseline), GO-061
      (no-weak-password-hashing, security, error, baseline), GO-062
      (no-sql-concatenation, security, error, baseline), GO-063
      (no-sensitive-data-in-logs, security, warning, standard). Each new
      rule must have all required fields per the standard/v1 schema: id,
      name, category, severity, description, detection.

  - id: REQ-009
    text: >
      Running `backstop pack compile` against the updated STD-GO-001 must
      produce a valid enforcement manifest, semgrep YAML config, and native
      checks JSON in .backstop/rules/. The compiled semgrep config must
      include all pattern-strategy rules from core, test, and security
      categories. Metric-strategy rules must appear in native checks JSON.
      Delegated-strategy rules must appear in the manifest with their
      delegated_to target but not in semgrep YAML.

  - id: REQ-010
    text: >
      Each detection strategy used in STD-GO-001 rules must map correctly
      to compiler output: pattern-strategy rules produce semgrep YAML entries,
      metric-strategy rules produce native check JSON entries,
      regex-strategy rules produce semgrep YAML entries with pattern-regex,
      and delegated-strategy rules produce manifest entries with no semgrep
      or native output. Advisory rules (note-only detection with no
      enforceable fields) must produce manifest entries with enforcement
      set to advisory.

  - id: REQ-011
    text: >
      The seven deferred categories (performance, observability, integration,
      contracts, concurrency, resilience, accessibility) must exist as
      empty directories under standards/go/rules/. The accessibility/
      directory does not currently exist and must be created by this spec.
      No rule files may be added to any deferred category directory in this
      spec. The existing concurrency rules in STD-GO-001 (GO-050, GO-051)
      are the only rules in deferred categories and must continue to compile
      correctly.

  - id: REQ-012
    text: >
      All semgrep rules must be runnable with semgrep OSS (no Pro features
      required). Rules must not use cross-file analysis, interprocedural
      taint tracking, or any feature requiring semgrep Pro or semgrep CI.
      Each rule must be single-file and intraprocedural only.

claims:
  # REQ-001: Core rules — required conventions
  - id: CLM-001
    requirement: REQ-001
    text: Core rules directory contains a semgrep YAML file for GO-003 no-global-mutable-state
    tests:
      - TestGoStandard_CoreRuleExists_GO003

  - id: CLM-002
    requirement: REQ-001
    text: Core rules directory contains a semgrep YAML file for GO-004 no-init-functions
    tests:
      - TestGoStandard_CoreRuleExists_GO004

  - id: CLM-003
    requirement: REQ-001
    text: Core rules directory contains a semgrep YAML file for GO-005 constructor-injection
    tests:
      - TestGoStandard_CoreRuleExists_GO005

  - id: CLM-004
    requirement: REQ-001
    text: Core rules directory contains a semgrep YAML file for GO-006 structured-logging
    tests:
      - TestGoStandard_CoreRuleExists_GO006

  - id: CLM-005
    requirement: REQ-001
    text: Each core rule file has correct backstop metadata (category, rule_id, compliance_tier)
    tests:
      - TestGoStandard_CoreRuleMetadata

  # REQ-002: Core rules — error handling and naming
  - id: CLM-006
    requirement: REQ-002
    text: Core rules directory contains a semgrep YAML file for GO-011 error-wrapping-required
    tests:
      - TestGoStandard_CoreRuleExists_GO011

  - id: CLM-007
    requirement: REQ-002
    text: Core rules directory contains a semgrep YAML file for GO-013 no-panic-in-library-code
    tests:
      - TestGoStandard_CoreRuleExists_GO013

  - id: CLM-052
    requirement: REQ-002
    text: Core rules directory contains a semgrep YAML file for GO-012 no-naked-returns
    tests:
      - TestGoStandard_CoreRuleExists_GO012

  - id: CLM-008
    requirement: REQ-002
    text: GO-012 no-naked-returns rule includes a note explaining the function-length constraint limitation
    tests:
      - TestGoStandard_CoreRule_GO012_HasNote

  - id: CLM-009
    requirement: REQ-002
    text: GO-020 no-stuttering-exports has a delegation note in STD-GO-001 explaining enforcement by golangci-lint (not in any semgrep rule file)
    tests:
      - TestGoStandard_STD_GO001_GO020_HasDelegationNote

  - id: CLM-047
    requirement: REQ-002
    text: GO-012 exists in STD-GO-001 rules array as a pattern-strategy rule with function-length constraint
    tests:
      - TestGoStandard_STD_GO001_HasRule_GO012

  - id: CLM-048
    requirement: REQ-002
    text: GO-020 exists in STD-GO-001 rules array as a delegated-strategy rule (not in semgrep YAML)
    tests:
      - TestGoStandard_STD_GO001_HasRule_GO020

  - id: CLM-053
    requirement: REQ-002
    text: Core rules directory contains a semgrep YAML rule for GO-010 no-ignored-errors with pattern strategy
    tests:
      - TestGoStandard_CoreRuleExists_GO010

  - id: CLM-054
    requirement: REQ-002
    text: GO-010 exists in STD-GO-001 rules array as a pattern-strategy rule
    tests:
      - TestGoStandard_STD_GO001_HasRule_GO010

  - id: CLM-055
    requirement: REQ-002
    text: Core rules directory contains a semgrep YAML rule for GO-021 error-type-suffix with pattern-regex strategy
    tests:
      - TestGoStandard_CoreRuleExists_GO021

  - id: CLM-056
    requirement: REQ-002
    text: GO-021 exists in STD-GO-001 rules array as a regex-strategy rule
    tests:
      - TestGoStandard_STD_GO001_HasRule_GO021

  - id: CLM-057
    requirement: REQ-002
    text: GO-040 exists in STD-GO-001 rules array as a delegated-strategy rule
    tests:
      - TestGoStandard_STD_GO001_HasRule_GO040

  - id: CLM-058
    requirement: REQ-002
    text: GO-040 import-ordering does NOT appear in any semgrep YAML rule file (delegated to golangci-lint)
    tests:
      - TestGoStandard_GO040_NotInSemgrepYAML

  # REQ-003: Test rules
  - id: CLM-010
    requirement: REQ-003
    text: Test rules directory contains a semgrep YAML file for GO-032 no-time-sleep-in-tests
    tests:
      - TestGoStandard_TestRuleExists_GO032

  - id: CLM-011
    requirement: REQ-003
    text: GO-032 semgrep rule is scoped to _test.go files only
    tests:
      - TestGoStandard_TestRule_GO032_ScopedToTestFiles

  - id: CLM-012
    requirement: REQ-003
    text: GO-030 test-file-required includes a note explaining metric-based detection limitation
    tests:
      - TestGoStandard_TestRule_GO030_HasNote

  - id: CLM-013
    requirement: REQ-003
    text: GO-031 table-driven-tests includes a note explaining custom analysis requirement
    tests:
      - TestGoStandard_TestRule_GO031_HasNote

  - id: CLM-049
    requirement: REQ-003
    text: GO-030 exists in STD-GO-001 rules array as a metric-strategy rule
    tests:
      - TestGoStandard_STD_GO001_HasRule_GO030

  - id: CLM-050
    requirement: REQ-003
    text: GO-031 exists in STD-GO-001 rules array as a pattern-strategy rule with note
    tests:
      - TestGoStandard_STD_GO001_HasRule_GO031

  # REQ-004: Security rules
  - id: CLM-014
    requirement: REQ-004
    text: Security rules directory contains a semgrep YAML file for GO-060 no-hardcoded-credentials
    tests:
      - TestGoStandard_SecurityRuleExists_GO060

  - id: CLM-015
    requirement: REQ-004
    text: Security rules directory contains a semgrep YAML file for GO-061 no-weak-password-hashing
    tests:
      - TestGoStandard_SecurityRuleExists_GO061

  - id: CLM-016
    requirement: REQ-004
    text: Security rules directory contains a semgrep YAML file for GO-062 no-sql-concatenation
    tests:
      - TestGoStandard_SecurityRuleExists_GO062

  - id: CLM-017
    requirement: REQ-004
    text: Security rules directory contains a semgrep YAML file for GO-063 no-sensitive-data-in-logs
    tests:
      - TestGoStandard_SecurityRuleExists_GO063

  - id: CLM-018
    requirement: REQ-004
    text: Each security rule includes CWE or OWASP identifier in backstop metadata
    tests:
      - TestGoStandard_SecurityRuleMetadata_ComplianceRefs

  # REQ-005: Semgrep rule format
  - id: CLM-019
    requirement: REQ-005
    text: All semgrep rule files parse as valid YAML with a top-level rules array
    tests:
      - TestGoStandard_AllRulesValidYAML

  - id: CLM-020
    requirement: REQ-005
    text: Every rule entry has required fields (id, pattern/patterns, message, severity, languages, metadata)
    tests:
      - TestGoStandard_AllRulesHaveRequiredFields

  - id: CLM-021
    requirement: REQ-005
    text: Every rule ID follows the go.<category>.<kebab-name> format
    tests:
      - TestGoStandard_AllRuleIDsFollowFormat

  - id: CLM-022
    requirement: REQ-005
    text: Every rule has languages set to [go]
    tests:
      - TestGoStandard_AllRulesTargetGo

  - id: CLM-051
    requirement: REQ-005
    text: Every rule has a metadata.backstop.rule_id field that maps to a valid STD-GO-001 rule ID
    tests:
      - TestGoStandard_AllRulesHaveBackstopRuleID

  # REQ-006: Invalid testdata fixtures
  - id: CLM-023
    requirement: REQ-006
    text: Each semgrep rule has a corresponding invalid fixture file in testdata/invalid/
    tests:
      - TestGoStandard_InvalidFixtureExistsPerRule

  - id: CLM-024
    requirement: REQ-006
    text: Invalid fixture filenames follow the <rule-id>-<description>.go pattern
    tests:
      - TestGoStandard_InvalidFixtureNamingConvention

  - id: CLM-025
    requirement: REQ-006
    text: Running semgrep against invalid fixtures produces violations for the expected rule
    tests:
      - TestGoStandard_InvalidFixtureTriggersRule

  # REQ-007: Valid testdata fixtures
  - id: CLM-026
    requirement: REQ-007
    text: Each semgrep rule has a corresponding valid fixture file in testdata/valid/
    tests:
      - TestGoStandard_ValidFixtureExistsPerRule

  - id: CLM-027
    requirement: REQ-007
    text: Valid fixture filenames follow the <rule-id>-<description>.go pattern
    tests:
      - TestGoStandard_ValidFixtureNamingConvention

  - id: CLM-028
    requirement: REQ-007
    text: Running semgrep against valid fixtures produces zero violations
    tests:
      - TestGoStandard_ValidFixturePassesAllRules

  # REQ-008: STD-GO-001 updates
  - id: CLM-029
    requirement: REQ-008
    text: STD-GO-001 contains rule GO-004 no-init-functions with all required schema fields
    tests:
      - TestGoStandard_STD_GO001_HasRule_GO004

  - id: CLM-030
    requirement: REQ-008
    text: STD-GO-001 contains rule GO-005 constructor-injection-required with all required schema fields
    tests:
      - TestGoStandard_STD_GO001_HasRule_GO005

  - id: CLM-031
    requirement: REQ-008
    text: STD-GO-001 contains rule GO-006 structured-logging-required with all required schema fields
    tests:
      - TestGoStandard_STD_GO001_HasRule_GO006

  - id: CLM-032
    requirement: REQ-008
    text: STD-GO-001 contains security rules GO-060 through GO-063 with all required schema fields
    tests:
      - TestGoStandard_STD_GO001_HasSecurityRules

  - id: CLM-033
    requirement: REQ-008
    text: Updated STD-GO-001 validates against the standard/v1 schema without errors
    tests:
      - TestGoStandard_STD_GO001_ValidatesAgainstSchema

  # REQ-009: Pack compile integration
  - id: CLM-034
    requirement: REQ-009
    text: backstop pack compile produces an enforcement manifest for STD-GO-001
    tests:
      - TestGoStandard_PackCompile_ProducesManifest

  - id: CLM-035
    requirement: REQ-009
    text: Compiled semgrep config contains all pattern-strategy rules from core, test, and security
    tests:
      - TestGoStandard_PackCompile_SemgrepIncludesPatternRules

  - id: CLM-036
    requirement: REQ-009
    text: Compiled native checks JSON contains all metric-strategy rules
    tests:
      - TestGoStandard_PackCompile_NativeIncludesMetricRules

  - id: CLM-037
    requirement: REQ-009
    text: Delegated-strategy rules appear in manifest but not in semgrep YAML
    tests:
      - TestGoStandard_PackCompile_DelegatedNotInSemgrep

  # REQ-010: Detection strategy mapping
  - id: CLM-038
    requirement: REQ-010
    text: Pattern-strategy rules produce semgrep YAML entries with pattern field
    tests:
      - TestGoStandard_StrategyMapping_PatternToSemgrep

  - id: CLM-039
    requirement: REQ-010
    text: Metric-strategy rules produce native check JSON entries with metric and threshold fields
    tests:
      - TestGoStandard_StrategyMapping_MetricToNative

  - id: CLM-040
    requirement: REQ-010
    text: Regex-strategy rules produce semgrep YAML entries with pattern-regex field
    tests:
      - TestGoStandard_StrategyMapping_RegexToSemgrep

  - id: CLM-041
    requirement: REQ-010
    text: Delegated-strategy rules produce manifest entries with delegated_to and no semgrep or native output
    tests:
      - TestGoStandard_StrategyMapping_DelegatedToManifest

  - id: CLM-042
    requirement: REQ-010
    text: >
      Advisory rules (note-only) produce manifest entries with enforcement
      set to advisory. Tested using a synthetic advisory fixture rule since
      no current STD-GO-001 rule uses advisory strategy.
    tests:
      - TestGoStandard_StrategyMapping_AdvisoryToManifest

  # REQ-011: Deferred categories
  - id: CLM-043
    requirement: REQ-011
    text: Deferred category directories exist and are empty (no rule files)
    tests:
      - TestGoStandard_DeferredCategoriesEmpty

  - id: CLM-044
    requirement: REQ-011
    text: Existing concurrency rules GO-050 and GO-051 compile without errors
    tests:
      - TestGoStandard_ConcurrencyRulesCompile

  # REQ-012: Semgrep OSS compatibility
  - id: CLM-045
    requirement: REQ-012
    text: All semgrep rules run successfully with semgrep OSS (no Pro features)
    tests:
      - TestGoStandard_AllRulesRunWithSemgrepOSS

  - id: CLM-046
    requirement: REQ-012
    text: No rule uses cross-file analysis, taint mode, or join mode
    tests:
      - TestGoStandard_NoProFeaturesUsed

contracts:
  - file: standards/go/STD-GO-001-go-code-standards.standard.md
    provides:
      - name: STD-GO-001
        kind: type
        signature: "standard/v1 artifact with rules array"
        notes: "Updated with new rules GO-004 through GO-006 (core) and GO-060 through GO-063 (security)"
    consumes:
      - source: artifacts/standard/v1/schema.json
        name: standard/v1
        kind: type

  - file: standards/go/rules/core/go-core.yml
    provides:
      - name: go.core rules
        kind: variable
        signature: "semgrep YAML rules file"
        notes: "Core category rules: GO-003, GO-004, GO-005, GO-006, GO-010, GO-011, GO-012, GO-013 (pattern strategy), GO-021 (regex strategy, pattern-regex). GO-012 has a function-length constraint that semgrep cannot enforce; the rule includes a note. GO-020 and GO-040 are delegated-strategy and do NOT appear in this file."
    consumes: []

  - file: standards/go/rules/test/go-test.yml
    provides:
      - name: go.test rules
        kind: variable
        signature: "semgrep YAML rules file"
        notes: "Test category rules: GO-032"
    consumes: []

  - file: standards/go/rules/security/go-security.yml
    provides:
      - name: go.security rules
        kind: variable
        signature: "semgrep YAML rules file"
        notes: "Security category rules: GO-060, GO-061, GO-062, GO-063"
    consumes: []
---

# SPEC-012: Go Standards Pack — Bootstrap Rule Content

## Overview

The Go standards pack (standards/go/) has the skeleton in place: STD-GO-001
defines rule metadata, the rules/ directory has subdirectories for nine of
the ten categories (ADR-0006) — the accessibility/ directory is missing and
must be created by this spec. The testdata/ directory has valid/ and invalid/
subdirectories. What it lacks is content — the actual semgrep YAML rules
that `backstop pack compile` turns into enforcement manifests and `backstop
code check` runs against real code.

This spec fills three of the ten categories with enforceable rules:

1. **Core** — Go idioms, structure, error handling, naming (ported from
   mechsuit go-conventions.md and STD-GO-001 existing rules)
2. **Test** — Test quality enforcement (table-driven tests, no time.Sleep)
3. **Security** — Baseline security rules (hardcoded creds, weak crypto,
   SQL injection, log leakage)

These three categories represent the bootstrap minimum. The remaining seven
categories (performance, observability, integration, contracts, concurrency,
resilience, accessibility) remain scaffolded but empty. Content for those
categories is deferred to future specs as the pack matures.

The mechsuit go-conventions.md is the source material for the core rules.
Not every convention maps to a semgrep rule — some require metric analysis
(file/function length), some require cross-file analysis (test file existence),
and some are delegated to existing tools (goimports, golangci-lint). Rules
that cannot be fully expressed as semgrep patterns include a note field
documenting the limitation and what supplementary tooling is needed.

## Requirements

Requirements are defined in frontmatter.

### Core Rules (REQ-001, REQ-002)

The core category covers the highest-value Go conventions from mechsuit
go-conventions.md and STD-GO-001:

| Rule ID | Name | Detection | In Semgrep YAML? | Tier |
|---------|------|-----------|------------------|------|
| GO-003 | no-global-mutable-state | pattern (semgrep) | Yes | baseline |
| GO-004 | no-init-functions | pattern (semgrep) | Yes | baseline |
| GO-005 | constructor-injection-required | pattern (semgrep) | Yes | standard |
| GO-006 | structured-logging-required | pattern (semgrep) | Yes | standard |
| GO-010 | no-ignored-errors | pattern (semgrep) | Yes | baseline |
| GO-011 | error-wrapping-required | pattern (semgrep) | Yes | standard |
| GO-012 | no-naked-returns | pattern + note (function length constraint) | Yes (with note) | standard |
| GO-013 | no-panic-in-library-code | pattern (semgrep) | Yes | baseline |
| GO-020 | no-stuttering-exports | delegated (golangci-lint revive) | No — manifest only | standard |
| GO-021 | error-type-suffix | regex (pattern-regex) | Yes | standard |
| GO-040 | import-ordering | delegated (golangci-lint goimports) | No — manifest only | standard |

GO-010 has strategy: pattern and appears in go-core.yml. GO-012 has strategy:
pattern in STD-GO-001 and appears in go-core.yml, but its function-length
constraint cannot be expressed in semgrep. The semgrep rule includes a note
documenting this limitation. GO-021 has strategy: regex and appears in
go-core.yml using pattern-regex. GO-020 and GO-040 have strategy: delegated
and do NOT appear in any semgrep YAML file. They are enforced by golangci-lint
and appear only in the compiled manifest via the compiler.

### Test Rules (REQ-003)

| Rule ID | Name | Detection | Tier |
|---------|------|-----------|------|
| GO-030 | test-file-required | metric + note | baseline |
| GO-031 | table-driven-tests | pattern + note (multi-assertion detection) | standard |
| GO-032 | no-time-sleep-in-tests | pattern (semgrep, scoped to _test.go) | baseline |

GO-030 and GO-031 require analysis beyond semgrep's capabilities and include
notes. GO-032 is fully expressible as a semgrep pattern scoped to test files.

### Security Rules (REQ-004)

| Rule ID | Name | Detection | Tier | Compliance Ref |
|---------|------|-----------|------|----------------|
| GO-060 | no-hardcoded-credentials | pattern (semgrep) | baseline | CWE-798, OWASP A07 |
| GO-061 | no-weak-password-hashing | pattern (semgrep) | baseline | CWE-327, OWASP A02 |
| GO-062 | no-sql-concatenation | pattern (semgrep) | baseline | CWE-89, OWASP A03 |
| GO-063 | no-sensitive-data-in-logs | pattern (semgrep) | standard | CWE-532, OWASP A09 |

All security rules are in the baseline or standard tier and include CWE/OWASP
references in their backstop metadata. These align with the security tier
model in ADR-0007.

### Detection Strategy Mapping (REQ-010)

Each detection strategy in STD-GO-001 maps to a specific compiler output:

| Strategy | Compiler Output | Example Rule |
|----------|----------------|--------------|
| pattern | semgrep YAML entry with pattern field | GO-003, GO-013 |
| metric | native check JSON entry with metric/threshold | GO-001, GO-002 |
| regex | semgrep YAML entry with pattern-regex | GO-021 |
| delegated | manifest entry with delegated_to, no semgrep/native output | GO-020, GO-040 |
| advisory (note-only) | manifest entry with enforcement: advisory | (no current rule — hypothetical future rules with note-only detection and no enforceable pattern) |

## Implementation

### Category Mapping: STD-GO-001 to Pack Directories

STD-GO-001 uses fine-grained categories (structure, error-handling, naming,
testing, etc.). The pack directories use coarser groupings. This is the
explicit mapping:

| STD-GO-001 Category | Pack Directory | Rationale |
|---------------------|----------------|-----------|
| structure | core | Core Go idioms |
| error-handling | core | Core Go idioms |
| naming | core | Core Go idioms |
| testing | test | Test-specific rules |
| security | security | Security-specific rules |
| imports | core | Core Go idioms (delegated, manifest only) |
| concurrency | concurrency | Deferred category |

This mapping means a rule's STD-GO-001 category (e.g., "error-handling")
differs from its pack directory (e.g., "core") and its semgrep rule ID
prefix (e.g., go.core.error-wrapping-required). The `metadata.backstop.rule_id`
field links back to the STD-GO-001 rule ID regardless of directory placement.

### Step 0: Create accessibility/ directory

The accessibility/ directory does not currently exist under standards/go/rules/.
Create it as an empty directory to ensure all ten category directories from
ADR-0006 are present:

```
mkdir -p standards/go/rules/accessibility/
```

### Step 1: Update STD-GO-001 with new rules

Add seven new rules to the STD-GO-001 frontmatter rules array:

**Core additions (GO-004 through GO-006):**
- GO-004: no-init-functions — `func init() { ... }` pattern, error severity, baseline tier
- GO-005: constructor-injection-required — detect `New` constructors accepting dependencies, warning severity, standard tier
- GO-006: structured-logging-required — detect `log.Print`/`log.Fatal` usage (should use zap), warning severity, standard tier

**Security additions (GO-060 through GO-063):**
- GO-060: no-hardcoded-credentials — detect string assignments to variables named password/secret/key/token, error severity, baseline tier
- GO-061: no-weak-password-hashing — detect `crypto/md5` and `crypto/sha1` imports for password contexts, error severity, baseline tier
- GO-062: no-sql-concatenation — detect string concatenation in SQL query strings, error severity, baseline tier
- GO-063: no-sensitive-data-in-logs — detect logging of variables named password/token/secret, warning severity, standard tier

### Step 2: Author semgrep YAML rule files

Create three rule files:

**standards/go/rules/core/go-core.yml** — Rules GO-003, GO-004, GO-005, GO-006, GO-010, GO-011, GO-012, GO-013 (pattern strategy), and GO-021 (regex strategy, using pattern-regex). Each rule follows the semgrep YAML format with backstop metadata block. GO-012 is included with a note about its function-length constraint limitation. GO-020 and GO-040 are NOT included — they are delegated-strategy rules enforced by golangci-lint and appear only in the compiled manifest.

**standards/go/rules/test/go-test.yml** — Rule GO-032 (no-time-sleep-in-tests). Scoped to `_test.go` files using semgrep's paths include/exclude mechanism.

**standards/go/rules/security/go-security.yml** — Rules GO-060, GO-061, GO-062, GO-063. Each includes CWE/OWASP identifiers in metadata.

### Step 3: Author testdata fixtures

**testdata/invalid/** — One Go file per semgrep-enforceable rule. Each file contains a minimal Go program that violates exactly that rule. Files are syntactically valid Go.

**testdata/valid/** — One Go file per rule demonstrating the correct pattern. Files must pass all rules without violations.

### Step 4: Verify compilation

Run `backstop pack compile` against the updated STD-GO-001 and verify:
- Enforcement manifest includes all rules with correct enforcement types
- Semgrep YAML config includes all pattern/regex rules
- Native checks JSON includes all metric rules
- Delegated rules appear in manifest only
- Advisory rules appear with enforcement: advisory

### Step 5: Verify semgrep execution

Run semgrep with the compiled rules against testdata:
- All invalid fixtures trigger their expected rule
- All valid fixtures produce zero violations
- No semgrep Pro features are required

## Verification

Verification is defined in frontmatter. Integration-level verification at 80%
coverage threshold. The test suite validates both the standard artifact content
and the round-trip through the compiler and semgrep engine.

Claims are defined in frontmatter.

## Sharp Edges

- **Semgrep pattern limitations for Go.** Several rules from go-conventions.md
  cannot be fully expressed as semgrep patterns. Function length (GO-001,
  GO-002) requires metric analysis. Test file existence (GO-030) requires
  filesystem inspection. Named return variable tracking (GO-012) requires
  function-scope context that semgrep handles imprecisely. These rules have
  notes documenting the limitation, but implementers must resist the urge to
  write over-broad patterns that produce false positives to "fill the gap."

- **GO-005 constructor injection is an aspirational pattern, not a mechanical
  rule.** Detecting "should use constructor injection" requires understanding
  intent. The semgrep rule can flag direct struct literal creation with
  exported fields that look like dependencies, but false positives are likely
  for simple data types. The compliance tier is standard (not baseline)
  specifically because this rule needs human judgment on edge cases.

- **Security rule scope creep.** The four security rules here are baseline
  coverage — they catch obvious mistakes, not sophisticated attacks. GO-060
  catches `password = "hunter2"` but not credentials loaded from a config
  file and assigned to a non-obvious variable name. ADR-0007 defines the
  full security tier model. These rules are floor, not ceiling.

- **Testdata fixtures are the behavioral contract.** If a fixture is wrong
  (an invalid fixture that does not actually violate the rule, or a valid
  fixture that does), the rule appears to work but is not actually enforcing
  anything. Fixture quality is as important as rule quality. Fixtures must be
  reviewed with the same rigor as the rules themselves.

- **Category boundary ambiguity and the mapping table.** STD-GO-001 uses
  fine-grained categories (structure, error-handling, naming) while the pack
  directories use coarser groupings (core, test, security). The explicit
  mapping (structure/error-handling/naming -> core, testing -> test, security
  -> security) means a rule's STD-GO-001 category differs from its semgrep
  rule ID prefix. If a rule moves categories in a future version, the semgrep
  rule ID (go.<category>.<name>) changes, which breaks baseline comparisons.
  Category assignment is a one-way door.

- **GORM-specific rules from go-conventions.md are excluded.** The mechsuit
  conventions include GORM-specific patterns (error checking on GORM calls,
  explicit preloads). These are framework-specific, not language-level. They
  belong in a future go-gorm pack or the integration category, not in the
  bootstrap core. This is a deliberate scope decision, not an oversight.

- **Semgrep version sensitivity.** Semgrep's pattern matching engine evolves
  between versions. A pattern that matches in semgrep 1.x might not match in
  2.x, or might match differently. The testdata fixtures are the regression
  safety net — if semgrep changes behavior, the fixtures catch it. But this
  means the pack has an implicit version coupling with semgrep that is not
  captured in the standard artifact.

## Review Questions

1. Do any of the new semgrep patterns produce false positives on the backstop-core codebase itself? Run the compiled rules against pkg/ to verify.

2. Does the GO-003 no-global-mutable-state pattern correctly exempt sync primitives and regexp.MustCompile, or does it need explicit exceptions in the semgrep rule?

3. Can GO-062 (no-sql-concatenation) distinguish between string concatenation in a SQL context vs general string concatenation? What is the false positive rate on codebases that don't use raw SQL?

4. Are the testdata fixtures sufficient to catch semgrep version regressions? Each fixture should test the minimal pattern that triggers the rule, not a complex example that might pass for unrelated reasons.

5. Does the updated STD-GO-001 with new rules still compile correctly with the existing pkg/compile code, or do the new rules use detection fields that the compiler does not yet handle?

6. Does the category mapping table (structure/error-handling/naming -> core) correctly place every rule? Verify that no rule's semgrep ID prefix contradicts its pack directory placement.

7. Are the CWE/OWASP references on security rules correct? Cross-reference GO-060 through GO-063 against the actual CWE and OWASP entry texts to verify the mapping is precise, not approximate.

## References

- Bundle: cli (spec seed context — pack compile and code check depend on rule content)
- ADR-0006: Standards Packs — semgrep engine, pack structure, ten categories
- ADR-0007: Security Standards — tiered compliance, ASVS mapping, CWE/OWASP
- STD-GO-001: Go Code Standards — existing rule definitions
- mechsuit go-conventions.md: Source material for Go convention rules
- SPEC-009: Pack Compile — CLI adapter that consumes compiled standards
- SPEC-008: Code Check — CLI command that runs enforcement
