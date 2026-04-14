---
title: "Pack Manifest and Authoring — What an Agent Needs to Extract a Pack"
number: BUNDLE-004
created: "2026-04-11"
schema_version: bundle/v1

bundle:
  name: pack-manifest-authoring
  version: "0.15.0"
  created: "2026-04-11"
  updated: "2026-04-08"
  category: feature

status:
  maturity: ready

requirements:
  - id: REQ-001
    text: "The pack manifest file must be named pack.yml and contain required top-level fields: name, version, language, archetype, description, and content."
    traces: [DD-1, DD-7, DD-8]

  - id: REQ-002
    text: "Pack names must follow org/pack-name two-part format, accepting alphanumeric characters and hyphens, normalized to lowercase internally for namespacing and matching, with original casing preserved in display output."
    traces: [DD-50]

  - id: REQ-003
    text: "Pack directories must follow a canonical layout: pack.yml (required), go.mod (required), rules/ (required if layer 2 rules exist), fixtures/rules/ (required, one lowercase subdir per rule ID), standards/ (optional), scaffolds/ (required if archetype is code), validators/ (required if layer 3 rules exist). Deviations are rejected by pack check."
    traces: [DD-51, DD-36]

  - id: REQ-004
    text: "Every pack must declare exactly one archetype: enforcement (rules only, no code content) or code (ships SDKs/scaffolds AND rules enforcing correct usage). The archetype field is required in the manifest."
    traces: [DD-16]

  - id: REQ-005
    text: "Code packs must enforce bidirectional co-occurrence: every scaffold must have at least one enforcement rule, AND every rule must reference at least one scaffold or SDK via pairs_with. A rule without code content pairing in a code pack is a validation error."
    traces: [DD-16, DD-44]

  - id: REQ-006
    text: "The content block must declare typed content using the allowed types: rules, scaffolds, sdk, contracts, test_patterns, ast_checks, rubrics. A valid enforcement pack may contain only a ruleset (no scaffolds or SDK required). Enforcement packs must not declare scaffolds or sdk."
    traces: [DD-7, DD-9]

  - id: REQ-007
    text: "Every rule must declare a risk_class field with one of four allowed values: security, correctness, style, perf. risk_class appears on ALL rules regardless of layer."
    traces: [DD-6]

  - id: REQ-008
    text: "Rules are organized into three enforcement layers: layer 1 (built-in tool rules, highest trust), layer 2 (custom declarative semgrep rules, medium trust), and layer 3 (custom validators, lowest trust). Each rule must declare its layer."
    traces: [DD-13, DD-22]

  - id: REQ-009
    text: "Layer 2 rules must declare both a rule: field (pointing to the compiled semgrep YAML file) and a standard: field (filepath to .standard.md or inline string). The rule ID in pack.yml must be identical to the semgrep rule ID in the .yml file."
    traces: [DD-30, DD-31, DD-34]

  - id: REQ-010
    text: "Layer 3 rules must declare a category: field with one of three values: presence, structural, or other. Categories presence and structural are auto-accepted with no justification required. Category other requires a mandatory justification: field explaining why layers 1-2 cannot handle the check."
    traces: [DD-14, DD-29]

  - id: REQ-011
    text: "Layer 3 validators must declare input_scope (single-file or multi-file) and a validator: field pointing to the validator script. Validators are invoked as validator.sh <fixture-path> with exit 0 = pass, non-zero = fail."
    traces: [DD-15, DD-32]

  - id: REQ-012
    text: "Layer 3 validators must run in process isolation: separate process, no network access, no filesystem writes outside pack directory, no environment variable access."
    traces: [DD-26]

  - id: REQ-013
    text: "Every claim must declare at least one positive fixture (known-good, must not trigger the rule) and at least one negative fixture (known-bad, must trigger the rule). Fixture paths are relative to the pack root and declared inline on claims in pack.yml."
    traces: [DD-3, DD-25]

  - id: REQ-014
    text: "Security-class rules must include at least one bypass_attempt: true negative fixture in addition to standard negative fixtures. Bypass-attempt fixtures test adversarial or accidental circumvention, not just obvious violations."
    traces: [DD-40]

  - id: REQ-015
    text: "Fixtures are plain files in engine-native format. Semgrep fixtures must include engine-native annotations (// ruleid: and // ok: comments). Claim-to-fixture mapping is declared exclusively in pack.yml, not in fixture file metadata."
    traces: [DD-25]

  - id: REQ-016
    text: "Fixture directories must use lowercase naming matching rule IDs (e.g., fixtures/rules/err-001/), consistent with the rule ID convention of lowercase kebab-case."
    traces: [DD-36, DD-31]

  - id: REQ-017
    text: "Rule IDs must use lowercase kebab-case format (e.g., err-001, stripe-001). On load, rule IDs are namespaced with the pack name using slash delimiters: pack-name/rule-id."
    traces: [DD-31, DD-48]

  - id: REQ-018
    text: "Pack versioning uses three levels: pack version (semver, whole artifact), ruleset version (all rules as a cohort, defaults to pack version for enforcement packs if omitted), and item version (scaffolds and SDKs individually). Individual rules are unversioned."
    traces: [DD-27, DD-19, DD-37]

  - id: REQ-019
    text: "Scaffolds must declare a tier field with value complete or skeleton. Complete scaffolds have all functions implemented and tests passing. Skeleton scaffolds have at least one stub/TODO; tests exist structurally but are not executed."
    traces: [DD-17]

  - id: REQ-020
    text: "Every scaffold must declare test_command specifying how to run its tests. For complete tier, pack test executes this command. For skeleton tier, the command is declared but not executed during validation."
    traces: [DD-35]

  - id: REQ-021
    text: "Scaffolds must declare use_when (scenario list), assumes (preconditions), and pairs_with (related items as a single object with optional keys rules, scaffolds, sdk). sample_config values must be flat key-value strings matching environment variable naming conventions."
    traces: [DD-20, DD-17]

  - id: REQ-022
    text: "The sdk content type is a single optional entry with fields: module (canonical reference), version, and provides (list of public surface the SDK exposes). A pack ships at most one SDK."
    traces: [DD-11]

  - id: REQ-023
    text: "tool_config entries declare language-native tool configuration requirements. Each entry specifies tool, file (consumer-side target path), and settings. Entries may stand alone with their own rule ID as layer 1 enforcement (requiring risk_class and claims with fixtures) or support custom rules via required_by."
    traces: [DD-24, DD-33, DD-38]

  - id: REQ-024
    text: "Every tool_config entry must be traceable to at least one rule in the pack. No orphan tool config is allowed."
    traces: [DD-24]

  - id: REQ-025
    text: ".standard.md files are optional prose documentation, not required by pack check. The standard: field in the manifest is optional and accepts a filepath, inline string, or may be omitted. Semgrep rule YAML is the source of truth for rule patterns."
    traces: [DD-45, DD-34]

  - id: REQ-026
    text: "Pack directories must include a go.mod (or language equivalent) for fixture dependencies. The CLI manages this file: pack new creates it, pack test runs go mod tidy automatically before fixture execution if deps are stale."
    traces: [DD-46]

  - id: REQ-027
    text: "Claim IDs must be unique within the pack. Recommended format is rule-id-clm-NNN (e.g., err-001-clm-001) for composition safety."
    traces: [DD-39]

  - id: REQ-028
    text: "pack check runs instant structural verification: manifest parsing, field validation, coherence, archetype constraints, layer enforcement, risk class requirements. pack test runs fixture execution: semgrep --test, tool_config tool execution, layer 3 validators, scaffold rendering and test execution."
    traces: [DD-49]

  - id: REQ-029
    text: "pack try <project-path> runs the pack's rules against a real codebase for author exploration — no gate, no other packs, just this pack against that code."
    traces: [DD-52]

  - id: REQ-030
    text: "Each pack targets exactly one language. Cross-language capabilities are expressed as a family of single-language packs coordinated by convention (shared publisher, shared name prefix, lockstep version cadence)."
    traces: [DD-8]

  - id: REQ-031
    text: "The manifest must support versioned coordinate references from specs and plans using the format pack-name@pack-version:item-name@item-version."
    traces: [DD-21]

  - id: REQ-032
    text: "The embedded Go standards pack (SPEC-012) must be loaded through the same path as third-party packs. No special case."
    traces: [DD-4]

  - id: REQ-033
    text: "Scaffolds are copy-once templates rendered to the consumer's repo. Once rendered, the consumer owns the output and it is not updated by pack upgrades. SDKs are native-language modules tracked in the manifest but not distributed by backstop."
    traces: [DD-10]

  - id: REQ-034
    text: "A pack's rules may reference its own SDK surface via pairs_with. Rule + SDK + scaffold move as a single versioned unit. Rules can assume the SDK's provides surface exists."
    traces: [DD-12]

  - id: REQ-035
    text: "All pack archetypes require mechanical proof via fixtures at their tier's expected completeness level. No archetype is exempt from fixture requirements."
    traces: [DD-18]

problem:
  summary: >
    BUNDLE-001 defines the full pack lifecycle — distribution, composition,
    supply chain, curation, LLM review — spanning 54 design decisions and 36
    open questions. That scope is far too broad to hand to an agent whose job
    is simply to extract a pack from an existing codebase. This bundle focuses
    exclusively on what an agent needs to author a valid pack: the manifest
    schema, content type definitions, archetype rules, scaffold tiers,
    enforcement layers, fixture requirements, and item-level versioning. It
    deliberately excludes distribution, composition, supply chain, registry,
    and curation concerns.

  user_story: >
    As an agent extracting a pack from an existing codebase, I need a clear
    manifest schema, content type definitions, archetype rules, and fixture
    requirements so I can produce a valid pack without needing to understand
    distribution, composition, or registry concerns.

  success_criteria:
    - A pack.yml JSON schema is implemented and enforced by pack check
    - pack new scaffolds the canonical directory layout (pack.yml, go.mod, rules/, fixtures/rules/, standards/, scaffolds/, validators/) per DD-51
    - pack check validates manifest structure against the schema (implementation delegated to BUNDLE-005, but the schema it checks against is this bundle's output)
    - An authoring agent can produce a valid pack.yml from this bundle's requirements alone, with no dependency on BUNDLE-001
    - The embedded Go standards pack (SPEC-012) is migrated to the new manifest format and loads through the same path as third-party packs per DD-4/REQ-032

solution:
  approach: >
    A focused pack authoring contract — manifest schema, content types,
    archetypes, scaffold tiers, enforcement layers, fixture requirements —
    extracted from BUNDLE-001's design decisions and distilled into a
    buildable scope with a concrete pack.yml example. The manifest is
    agent-first per ADR-0001: optimized for machine generation and parsing,
    with structured fields over freeform prose.

  assumptions:
    - Semgrep is available as the primary static analysis engine for layers 1-2
    - Go is the first supported language; language-equivalent patterns (go.mod, go test) generalize later
    - BUNDLE-005 (pack validation) implements the check/test pipeline against this bundle's schema
    - Pack distribution, registry, and composition are out of scope (BUNDLE-006)
    - The existing SPEC-012 Go standards pack is the migration proof point
---

# Pack Manifest and Authoring

## Current Thinking

### The extraction problem

An agent looking at an existing codebase with standards, patterns, and
conventions needs to produce a pack — a portable, validated, declarative
artifact that captures those standards for reuse. Today, the only real pack
is SPEC-012 (Go standards, embedded via go:embed). Extracting a new pack
requires knowledge that is scattered across BUNDLE-001's 54 DDs and 36 OQs,
most of which are irrelevant to authoring.

This bundle distills the authoring-relevant subset into a self-contained
contract. An agent consuming this bundle should be able to produce a valid
`pack.yml` + content files without reading BUNDLE-001 at all.

### What a pack IS (from the agent's perspective)

A pack is declarative data. Not Go code, not interfaces, not plugins. It is:
- A `pack.yml` manifest declaring metadata, content types, and versioning
- Content files referenced by the manifest (rules, scaffolds, SDK references,
  fixtures)
- A validation contract: the pack is not loadable unless it mechanically
  proves it does what it claims

### The two archetypes

Every pack is one of two archetypes:

1. **Enforcement pack** — ships rules only. Semgrep patterns, linter configs,
   custom validators, and fixtures proving them. No code ships to the
   consumer's repo.

2. **Code pack** — ships SDKs, scaffolds, or both — AND always ships rules
   that enforce correct usage of that code. The co-occurrence rule: if the
   manifest declares `sdks` or `scaffolds`, it must also declare `rules`
   covering that code surface. A pack with code but no enforcement rules is
   a validation failure.

### The three-layer enforcement model

Packs layer enforcement in three tiers ordered by trust:

- **Layer 1 — Built-in tool rules (highest trust).** Semgrep core rules,
  golangci-lint, `go vet`, language-native linters. Packs reference these;
  they don't reimplement them.
- **Layer 2 — Custom declarative rules (medium trust).** Pack-authored
  semgrep patterns compiled from `.standard.md`. Declarative, pattern-
  matching, sandboxed. Subject to mandatory claim→fixture→rule validation.
  This is the bulk of what a pack ships.
- **Layer 3 — Custom validators (lowest trust, highest scrutiny).** Shell
  scripts, AST walkers, cross-file checks — things layers 1 and 2 genuinely
  cannot express. Subject to the strictest validation. The escape hatch, not
  the default path.

### Scaffold tiers

Scaffolds come in two tiers with different verification expectations:

- **Complete scaffold:** All exported functions implemented, all tests
  substantive and passing, no TODOs. Consumer changes config only. `pack
  test` runs tests via `test_command` and expects all pass.
- **Skeleton scaffold:** At least one exported function is a stub/TODO. Test
  functions exist structurally (right name, right file) but bodies are empty
  or comment-only — they do NOT prescribe behavior because the consumer
  decides the implementation. `pack check` validates structure only (files
  exist, function signatures exist, test function names exist). Tests are NOT
  executed. Skeleton tests declare WHAT to test, not HOW it should behave.

### Content types

A pack's `content:` block declares which types it provides:

| Type | Description |
|------|-------------|
| `rules` | Semgrep patterns from `.standard.md` with claim-fixture-rule mapping and risk class |
| `scaffolds` | Copy-once templates producing new files; declare tier (complete/skeleton) and `use_when` scenarios |
| `sdk` | Singular native-language module reference; backstop tracks but doesn't distribute |
| `contracts` | Reusable public API signatures for the contract-signature gate step |
| `test_patterns` | Per-language test substantiveness heuristics |
| `ast_checks` | Declarative AST-level rules (layer 3 custom validators per DD-43/DD-44) |
| `rubrics` | Versioned LLM review rubrics |

A minimal pack may ship only `rules`; a full pack ships any subset.
Fixture paths are declared inline on claims, relative to pack root — there
is no separate `fixtures` content type.

### Item-level versioning

The pack carries a semver (`version: "1.2.0"`). Individual items (scaffolds,
SDKs) carry their own versions independently. A pack at v1.2.0 might contain
`stripe-webhook-handler@v2` and `event-router@v1`. Item versions enable
specs and plans to reference specific items for reproducibility.

### The `use_when` scenario model

Each scaffold declares scenarios where it is the right choice:
- `use_when:` — plain-language scenario descriptions
- `assumes:` — preconditions expected in the consumer's codebase
- `pairs_with:` — other pack items that should be used alongside

### Author-time vs consumption-time distinction

The pack lifecycle has two completely separate execution contexts:

- **`pack check` + `pack test` = author's tools.** `pack check` runs instant
  structural verification — manifest parsing, field validation, coherence,
  archetype constraints, layer enforcement, risk class requirements. `pack test`
  runs fixture execution — semgrep --test, tool_config execution, layer 3
  validators, scaffold rendering and test execution. This is what the pack
  author runs while building the pack: `pack check` for the fast inner loop,
  `pack test` to prove fixtures pass.

- **`backstop gate` = consumer's tool.** Runs the pack's rules as part of the
  normal gate pipeline. The pack is invisible infrastructure — the consumer
  never interacts with "pack" as a concept. They run `backstop gate --all` and
  the pack's semgrep rules, tool configs, and custom validators execute
  alongside everything else. The loader resolves installed packs and merges
  their enforcement into the gate's check pipeline at runtime.

### Connected prescription graph (manifest-facing aspect)

The manifest must support being referenced by versioned coordinates from
specs and plans: `pack-name@pack-version:item-name@item-version`. This is
the mechanism that makes pack content traceable through the backstop
lifecycle.

## Draft pack.yml Examples

### Enforcement pack (rules only, no code)

```yaml
# pack.yml — enforcement pack archetype
name: acme/go-http-standards
version: "1.0.0"
language: go
archetype: enforcement
description: "HTTP handler standards for Go services"

tool_config:
  # Standalone layer 1 enforcement — tool config IS the rule (DD-33)
  - id: tool-errcheck
    tool: golangci-lint
    risk_class: correctness      # required on standalone tool_config (DD-33)
    file: .golangci.yml          # consumer-side target path (DD-38)
    settings:
      linters:
        enable:
          - errcheck
    claims:
      - id: tool-errcheck-clm-001
        text: "All error returns must be checked"
        fixtures:
          positive:
            - fixtures/rules/tool-errcheck/good-checked-err.go
          negative:
            - fixtures/rules/tool-errcheck/unchecked-err.go

  # Tool config supporting a custom layer 2 rule
  - tool: golangci-lint
    file: .golangci.yml          # consumer-side target path (DD-38)
    settings:
      linters:
        enable:
          - gocritic
      linters-settings:
        gocritic:
          enabled-checks:
            - httpNoBody
    required_by: [http-001]

content:
  ruleset:
    version: "1.0.0"             # defaults to pack version if omitted (DD-37)
    rules:
      - id: http-001             # lowercase kebab-case (DD-31)
        standard: standards/http-error-handling.standard.md
        rule: rules/http-001.yml # semgrep rule file (DD-30), ID must match
        risk_class: correctness  # on all rules (DD-6)
        layer: 2
        claims:
          - id: http-001-clm-001
            text: "HTTP handlers must wrap errors with context"
            fixtures:
              positive:          # list (DD-25)
                - fixtures/rules/http-001/good-error-wrap.go
              negative:
                - fixtures/rules/http-001/bare-error-return.go
                - fixtures/rules/http-001/nil-error-swallow.go
          - id: http-001-clm-002
            text: "Error responses must include request ID"
            fixtures:
              positive:
                - fixtures/rules/http-001/good-request-id.go
              negative:
                - fixtures/rules/http-001/missing-request-id.go

      - id: http-002
        standard: standards/http-middleware.standard.md
        rule: rules/http-002.yml
        risk_class: security
        layer: 2
        claims:
          - id: http-002-clm-001
            text: "All handlers must use auth middleware"
            fixtures:
              positive:
                - fixtures/rules/http-002/good-auth-chain.go
              negative:
                - path: fixtures/rules/http-002/no-auth-handler.go
                - path: fixtures/rules/http-002/auth-after-handler.go
                - path: fixtures/rules/http-002/noop-middleware.go
                  bypass_attempt: true    # middleware applied but always passes (DD-40)
                - path: fixtures/rules/http-002/auth-in-handler-body.go
                  bypass_attempt: true    # auth check in handler, not middleware (DD-40)

      - id: http-003
        standard: standards/http-handler-presence.standard.md
        risk_class: correctness
        layer: 3
        validator: validators/handler-presence-check.sh
        input_scope: multi-file
        category: presence        # layer 3 only (DD-29), auto-accepted
        claims:
          - id: http-003-clm-001
            text: "Every HTTP handler package must have middleware.go"
            fixtures:
              positive:
                - fixtures/rules/http-003/valid-package/
              negative:
                - fixtures/rules/http-003/missing-middleware/
                - fixtures/rules/http-003/empty-middleware/

      - id: http-004
        standard: "Binary payloads must use protobuf encoding"  # inline string (DD-34)
        risk_class: correctness
        layer: 3
        validator: validators/binary-payload-check.sh
        input_scope: single-file
        category: other           # layer 3 only (DD-29), requires justification
        justification: "Binary file format inspection — semgrep cannot parse non-text content"
        claims:
          - id: http-004-clm-001
            text: "Binary payloads must use protobuf encoding, not custom formats"
            fixtures:
              positive:
                - fixtures/rules/http-004/valid-proto.bin
              negative:
                - fixtures/rules/http-004/custom-binary.bin
```

### Code pack (rules + complete scaffold + skeleton scaffold + SDK)

```yaml
# pack.yml — code pack archetype
name: acme/go-stripe-integration
version: "2.1.0"
language: go
archetype: code
description: "Stripe integration scaffolds, SDK, and enforcement for Go"

content:
  sdk:
    module: github.com/acme/stripe-go-sdk
    version: "1.3.0"
    provides:
      - stripe.NewClient
      - stripe.Subscribe
      - stripe.HandleWebhook
      - stripe.VerifySignature

  scaffolds:
    - id: stripe-webhook-handler
      version: "2"
      tier: complete
      path: scaffolds/webhook-handler/
      test_command: "go test ./..."     # explicit test command (DD-35)
      description: "Production-ready Stripe webhook handler with signature verification"
      use_when:
        - "Service needs to receive and process Stripe webhook events"
        - "Handler must verify webhook signatures before processing"
      assumes:
        - "Go module with net/http routing already configured"
        - "Environment variable pattern for secrets (STRIPE_WEBHOOK_SECRET)"
      pairs_with:                       # single object (DD-20)
        rules: [stripe-001, stripe-002]
        sdk: stripe.HandleWebhook
      sample_config:                    # flat key-value, env var names (DD-17)
        WEBHOOK_SECRET_ENV: "STRIPE_WEBHOOK_SECRET"
        ROUTE_PREFIX: "/webhooks/stripe"

    - id: stripe-event-router
      version: "1"
      tier: skeleton                    # structural validation only (DD-17)
      path: scaffolds/event-router/
      test_command: "go test ./..."     # not executed for skeleton (DD-35)
      description: "Event routing structure for dispatching Stripe events to handlers"
      use_when:
        - "Service processes multiple Stripe event types with different handlers"
        - "Event routing needs to be testable in isolation"
      assumes:
        - "stripe-webhook-handler scaffold already rendered"
        - "At least 2 distinct event types to handle"
      pairs_with:
        scaffolds: [stripe-webhook-handler]
        sdk: stripe.Subscribe
      sample_config:
        EVENT_HANDLER_COUNT: "3"

  ruleset:
    version: "2.1.0"
    rules:
      - id: stripe-001               # lowercase kebab-case (DD-31)
        standard: standards/stripe-sdk-usage.standard.md
        rule: rules/stripe-001.yml    # semgrep rule file (DD-30)
        risk_class: correctness       # on all rules (DD-6)
        layer: 2
        claims:
          - id: stripe-001-clm-001
            text: "Must use pack SDK client, not raw stripe-go"
            fixtures:
              positive:               # list (DD-25)
                - fixtures/rules/stripe-001/good-sdk-usage.go
              negative:
                - fixtures/rules/stripe-001/raw-stripe-import.go
          - id: stripe-001-clm-002
            text: "SDK client must be injected, not constructed inline"
            fixtures:
              positive:
                - fixtures/rules/stripe-001/good-injection.go
              negative:
                - fixtures/rules/stripe-001/inline-construction.go

      - id: stripe-002
        standard: standards/stripe-webhook-security.standard.md
        rule: rules/stripe-002.yml
        risk_class: security
        layer: 2
        claims:
          - id: stripe-002-clm-001
            text: "Webhook signature must be verified before event processing"
            fixtures:
              positive:
                - fixtures/rules/stripe-002/good-sig-verify.go
              negative:
                - path: fixtures/rules/stripe-002/no-sig-verify.go
                - path: fixtures/rules/stripe-002/sig-verify-after-process.go
                - path: fixtures/rules/stripe-002/sig-verify-noop.go
                  bypass_attempt: true    # verify called but result ignored (DD-40)
          - id: stripe-002-clm-002
            text: "Webhook secret must come from environment, not hardcoded"
            fixtures:
              positive:
                - fixtures/rules/stripe-002/good-env-secret.go
              negative:
                - path: fixtures/rules/stripe-002/hardcoded-secret.go
                - path: fixtures/rules/stripe-002/obfuscated-secret.go
                  bypass_attempt: true    # base64-encoded hardcoded value (DD-40)
                - path: fixtures/rules/stripe-002/env-with-fallback.go
                  bypass_attempt: true    # env read with hardcoded fallback (DD-40)

      - id: stripe-003
        standard: standards/stripe-scaffold-structure.standard.md
        risk_class: correctness
        layer: 3
        validator: validators/scaffold-structure-check.sh
        input_scope: multi-file
        category: structural          # layer 3 only (DD-29), auto-accepted
        claims:
          - id: stripe-003-clm-001
            text: "Rendered scaffold must contain handler_test.go alongside handler.go"
            fixtures:
              positive:
                - fixtures/rules/stripe-003/valid-structure/  # directory fixture (DD-32)
              negative:
                - fixtures/rules/stripe-003/missing-test/
                - fixtures/rules/stripe-003/wrong-test-name/
```

## Draft Requirements

Requirements are formally defined in the frontmatter `requirements:` block
(REQ-001 through REQ-035). Each requirement traces to one or more design
decisions. See the frontmatter for the authoritative list.

## Draft Design Decisions

- **DD-1:** Packs are declarative data, not Go code. No interfaces for
  authors to implement. The Go-side loader adapts to pack authors; errors
  speak in pack-author vocabulary. Rationale: rejects the 14-interface model
  from the founder's day-job project. [Source: BUNDLE-001 DD-1]

- **DD-2:** Validation is a hard precondition for loading. Unvalidated packs
  cannot be used. Both `pack check` and `pack test` must pass at 100% before
  a pack is loadable. [Source: BUNDLE-001 DD-3]

- **DD-3:** Claim-fixture-rule mapping is enforced mechanically. A rule that
  claims X but whose fixture does not exercise X is a validation failure.
  Every claim must have both positive (known-good) and negative (known-bad)
  fixtures. [Source: BUNDLE-001 DD-4]

- **DD-4:** The embedded Go standards pack (SPEC-012) is loaded through the
  same path as third-party packs. No special case. Proves the loader works
  for outsiders. [Source: BUNDLE-001 DD-5]

- **DD-5:** Packs are network-isolated at load. No fetches at validate,
  compile, or run time. All inputs are in the pack or in explicitly-added
  sibling packs. [Source: BUNDLE-001 DD-13]

- **DD-6:** Every rule declares a `risk_class:` field with allowed values
  `security`, `correctness`, `style`, `perf`. This is distinct from the
  `category:` field on layer 3 rules (DD-29). `risk_class:` appears on ALL
  rules regardless of layer; `category:` appears ONLY on layer 3 rules.
  Security-class rules carry stricter requirements: mandatory bypass-attempt
  negative fixtures, mandatory author signature, stricter claim coverage
  thresholds. Enforced by `pack check` Phase 1 (field exists, valid enum)
  and Phase 6 (security-class requirements). [Source: BUNDLE-001 DD-14]

- **DD-7:** Pack content is typed and extensible via the `content:` block.
  Types: rules, scaffolds, sdk, contracts, test_patterns, ast_checks,
  rubrics. A minimal pack ships only rules; a full pack any subset.
  The archetype (enforcement vs code) is an additional manifest-level
  declaration alongside the content type list. There is no top-level
  `fixtures:` content type — fixture paths are declared inline on claims
  and are relative to the pack root. [Source: BUNDLE-001 DD-16]

- **DD-8:** A pack is a language-specific enforcement unit. Each pack targets
  exactly one language. Cross-language capabilities are expressed as a family
  of single-language packs coordinated by convention (shared publisher, shared
  name prefix, lockstep version cadence). Backstop never models cross-language
  packs. [Source: BUNDLE-001 DD-17]

- **DD-9:** Recipes are not a content type. What was previously discussed as
  "recipes" maps to scaffold tiers (complete and skeleton). There is no
  advisory tier — non-enforced advice is documentation, not a pack artifact.
  [Source: BUNDLE-001 DD-18]

- **DD-10:** Scaffolds are copy-once templates backstop renders and writes.
  SDKs are native-language modules consumers import via their native
  toolchain; backstop tracks the reference but does not distribute SDK code.
  Because the pack is language-specific, scaffolds don't need a `language:`
  field and `sdk:` is singular. [Source: BUNDLE-001 DD-19]

- **DD-11:** `sdk:` is a single optional entry. Fields: `module` (canonical
  reference), `version`, and `provides:` (public surface the pack claims the
  SDK exposes). A pack ships at most one SDK. [Source: BUNDLE-001 DD-20]

- **DD-12:** A pack's rules can reference its own SDK surface. Rules enforce
  correct usage of the SDK. Rule + SDK + scaffold move as a single versioned
  unit. This makes SDK usage enforceable, not merely available.
  [Source: BUNDLE-001 DD-21]

- **DD-13:** Enforcement is layered: built-in rules → custom declarative
  rules → custom validators. Don't use layer 3 for something layer 1 or 2
  can handle. Maximize use of battle-tested tooling, minimize custom surface
  area. [Source: BUNDLE-001 DD-43]

- **DD-14:** Layer 3 acceptance uses a presence-vs-content heuristic.
  Semgrep and linters validate content — they operate on code that exists.
  They literally cannot detect what isn't there. This gives a clean mechanical
  classification. **Auto-accepted categories (no justification needed):**
  (1) Presence/absence checks — file exists, function exists, field exists,
  pattern exists somewhere in the project. Semgrep cannot detect missing things.
  (2) Structural/filesystem checks — naming conventions, file placement,
  directory structure. Not content-level analysis. **Requires-justification
  category:** (3) Content-level checks that semgrep/linters allegedly can't
  handle — binary file inspection, encoding validation, semantic checks beyond
  pattern matching, content requiring external context. These need the
  `justification:` field explaining why existing tools can't do it.
  [Source: BUNDLE-001 DD-44, refined via presence-vs-content distinction]

- **DD-15:** Layer 3 validators must declare their input scope (single-file
  or multi-file). Multi-file validators carry stricter verification
  requirements. No layer 3 validator may write files, make network calls, or
  access environment variables. [Source: BUNDLE-001 DD-45]

- **DD-16:** Two pack archetypes — enforcement packs and code packs.
  Enforcement packs ship rules only. Code packs ship SDKs/scaffolds AND
  always ship rules enforcing correct usage. Mandatory co-occurrence rule:
  every scaffold must have at least one enforcement rule. No infrastructure
  exemption — if an agent can't think of a rule for a scaffold, the scaffold
  isn't opinionated enough to be in a pack. The rule is "if you're going to
  do X, do it this way" — every scaffold is an opinion about how to do
  something, and that opinion must be enforceable. Bidirectional: every
  scaffold must have at least one rule, AND every rule must reference at
  least one scaffold or SDK via `pairs_with`. Rules without code content
  pairing belong in an enforcement pack.
  [Source: BUNDLE-001 DD-46]

- **DD-17:** Scaffolds have two tiers — complete and skeleton — with
  different verification expectations. **Complete:** all exported functions
  implemented, all tests substantive and passing, no TODOs. Consumer changes
  config only. `pack test` runs tests and expects all pass.
  **Skeleton:** at least one exported function is a stub/TODO. Test functions
  exist structurally (right name, right file) but bodies are empty or
  comment-only — they do NOT prescribe behavior because the consumer decides
  the implementation. `pack check` validates structure only (files exist,
  function signatures exist, test function names exist). Tests are NOT
  executed. The consumer's own `backstop gate` handles behavioral validation
  after implementation. Skeleton tests should declare WHAT to test, not HOW
  it should behave — the scaffold author cannot predict consumer-specific
  behavior. `sample_config:` values should be strings, flat key-value (no
  nesting), names matching environment variable conventions. These are
  substituted during `pack test` scaffold rendering.
  [Source: BUNDLE-001 DD-47]

- **DD-18:** All pack archetypes require mechanical proof via fixtures. No
  pack is loadable without proving it does what it claims at its tier's
  expected completeness level. [Source: BUNDLE-001 DD-48]

- **DD-19:** Pack items (scaffolds, SDKs) are individually versioned within
  the pack. Both pack-level and item-level versions appear in the manifest.
  Item-level versions enable specs and plans to reference specific item
  versions for reproducibility. [Source: BUNDLE-001 DD-49]

- **DD-20:** Scaffolds declare `use_when` scenarios — a list of situations
  where the scaffold is the right choice. Includes `assumes:` (preconditions)
  and `pairs_with:` (related items). `pairs_with:` is a single object with
  optional keys `rules`, `scaffolds`, `sdk` — not a list of single-key
  objects. Example: `pairs_with: {rules: [err-001], scaffolds: [config-loader]}`.
  `pairs_with.rules` references should use actual rule IDs that exist in the
  pack's `content.ruleset.rules` or `tool_config` entries. While dangling
  references are validated in BUNDLE-005 Phase 2 (coherence), authoring agents
  should ensure references resolve to avoid downstream validation failures.
  Serves spec-writer agents and `pack check`.
  [Source: BUNDLE-001 DD-50, revised round 2 P1]

- **DD-21:** The manifest must support being referenced by versioned
  coordinates from specs/plans: `pack-name@pack-version:item-name@item-version`.
  This is the pack-manifest-facing aspect of the connected prescription graph.
  [Source: BUNDLE-001 DD-54]

- **DD-22:** Three execution paths for rule enforcement. Packs support three
  engines: (1) semgrep for layers 1-2 custom rules, (2) language-native tools
  (golangci-lint, ruff, clippy, etc.) referenced and configured by the pack
  but executed by backstop, (3) custom validators for layer 3. Each engine
  has its own fixture execution path. Packs do not reimplement what existing
  tools already catch. [Source: OQ-3 resolution]

- **DD-23:** Packs only ship net-new rules; language-native built-in rules
  are the implied floor. A pack does not enumerate or re-declare built-in
  linter rules. The language toolchain's default ruleset is assumed present.
  Packs layer custom enforcement on top. This keeps packs focused on what's
  unique to the pack author's standards. [Source: OQ-3 resolution]

- **DD-24:** Packs declare tool configuration requirements under `tool_config:`.
  If a pack needs a language-native tool rule that's off by default, it
  declares the required configuration in `pack.yml`. `pack add` reads
  `tool_config` and merges it into the consumer's existing tool config file
  (`.golangci.yml`, `pyproject.toml`, etc.) or creates one. Every
  `tool_config` entry must be traceable to at least one rule in the pack —
  `pack check` checks this. No orphan tool config: if you're turning a
  tool on, there must be a rule that requires it. [Source: OQ-3 resolution]

- **DD-25:** Fixtures are plain files in engine-native format; mapping lives
  in `pack.yml`. Fixture files are just code (or input data). They don't
  carry metadata about which claim they belong to. The claim-to-fixture
  mapping is declared exclusively in the manifest. `pack test` reads the
  manifest, resolves fixture paths, dispatches to the correct engine. No
  backstop-specific annotation format — but engine-native annotations are
  expected and required (e.g., semgrep `--test` requires `// ruleid:` and
  `// ok:` comments in fixtures; these are semgrep's convention, not
  backstop's). Both positive and negative fixtures are lists. At least one
  positive required, at least one negative required per claim. All positives
  must not trigger the rule. All negatives must trigger the rule.
  [Source: OQ-2 resolution]

- **DD-26:** Layer 3 validators run in process isolation. Separate process,
  no network access, no filesystem writes outside the pack directory, no
  environment variable access. Enforced at the OS level. Simpler than WASM,
  stronger than convention-only. [Source: OQ-1 resolution]

- **DD-27:** Three version levels — pack, ruleset, item. Pack version
  (semver) covers the whole artifact. Ruleset version covers all rules as a
  cohort — rules evolve together as a single enforcement posture. Item
  version covers scaffolds and SDKs individually. Specs pin to ruleset
  version for enforcement and item version for code. Individual rules are
  unversioned; they move as a unit under the ruleset. [Source: OQ-6 resolution]

- **DD-29:** Layer 3 validators declare a `category:` field that enables
  mechanical acceptance. This field appears ONLY on layer 3 rules and is
  distinct from `risk_class:` (DD-6) which appears on ALL rules. Three
  category values: `presence` (checking that something exists or is missing),
  `structural` (checking filesystem conventions — naming, placement, directory
  structure), and `other` (everything else). Categories `presence` and
  `structural` are auto-accepted by `pack check` — no `justification:`
  field required because the reason is mechanical: semgrep and linters operate
  on existing content, not on absence or filesystem structure. Category
  `other` requires a mandatory `justification:` field explaining why layers
  1-2 can't handle the check. These two fields never collide: `risk_class:`
  classifies severity (security/correctness/style/perf), `category:`
  classifies layer-3 validation type (presence/structural/other).

- **DD-30:** Every layer 2 rule must declare a `rule:` field pointing to
  the compiled semgrep YAML rule file (e.g., `rule: rules/err-001.yml`).
  This is the file `semgrep --test` needs. The `standard:` field points to
  the human/agent-readable `.standard.md` documentation; the `rule:` field
  points to the machine-executable semgrep pattern. Both are required for
  layer 2 rules. Layer 1 and layer 3 rules do not use the `rule:` field.
  [Source: prototype review P0]

- **DD-31:** The pack rule ID (declared in `pack.yml` as `id:`) and the
  semgrep rule ID (in the `.yml` rule file and `// ruleid:` annotations)
  must be identical. No separate mapping. `pack test` checks this by
  reading the semgrep rule file and comparing the ID field. Convention:
  lowercase kebab-case (e.g., `err-001`, `slotly-001`). This keeps things
  simple and unambiguous. [Source: prototype review P0]

- **DD-32:** Layer 3 validators are invoked as: `validator.sh <fixture-path>`.
  For single-file `input_scope`: fixture-path is a file. For multi-file
  `input_scope`: fixture-path is a directory. Exit 0 = pass (no violation
  found). Exit non-zero = fail (violation found). Stdout may contain a
  violation message. For directory fixtures, the fixture directory must
  contain the expected structure (e.g., an `internal/` subdirectory if the
  validator expects one). `pack test` creates no intermediate structure —
  the fixture is the complete input. [Source: prototype review P0]

- **DD-33:** `tool_config` entries may stand alone as layer 1 enforcement
  with their own rule ID (e.g., `id: tool-errcheck`). They do not need to
  reference a custom layer 2 rule via `required_by`. Some tool configs ARE
  the enforcement, not a dependency of custom rules. When `tool_config` has
  its own rule ID, it is a full first-class rule and MUST declare `risk_class:`
  like any other rule (DD-6). It must also have at least one claim with
  fixtures proving the tool catches the intended violation.
  [Source: prototype review P1, revised round 2 P1]

- **DD-34:** `standard:` accepts either a filepath (to a `.standard.md` file
  that must exist on disk) or an inline string (short description). Filepath
  is preferred when the pack author generates `.standard.md` documentation.
  Inline string is acceptable for rules extracted from implicit conventions
  where generating a full `.standard.md` would be overhead.
  [Source: prototype review P1]

- **DD-35:** Every scaffold declares `test_command:` specifying how to run
  its tests (e.g., `go test ./...`, `pytest`, `cargo test`). Explicit is
  better than inferring from `language:`. `pack test` uses this when
  executing scaffold tests (complete tier only).
  [Source: prototype review P1]

- **DD-36:** Fixture directories use lowercase matching rule IDs (e.g.,
  `fixtures/rules/err-001/`, not `ERR-001/`). Consistent with rule ID
  convention (DD-31: lowercase kebab-case).
  [Source: prototype review P2]

- **DD-37:** For enforcement packs, ruleset version defaults to pack version
  if omitted. Explicit override is allowed.
  [Source: prototype review P2]

- **DD-38:** The `file:` field in `tool_config` entries is a consumer-side
  target path (e.g., `.golangci.yml`), not a pack-internal path. `pack
  validate` Phase 1 does NOT check its existence on disk. It is the path
  `pack add` will write to in the consumer's repo.
  [Source: prototype review P2]

- **DD-39:** Claim IDs must be unique within the pack. Recommended format:
  `CLM-NNN` namespaced by rule (e.g., `ERR-001-CLM-001`) for composition
  safety. But within-pack uniqueness is the hard requirement.
  [Source: prototype review P2]

- **DD-40:** Security-class rules require bypass-attempt negative fixtures
  in addition to standard negative fixtures. A bypass-attempt fixture is NOT
  just another way to violate the rule — it is code that TRIES TO SATISFY
  the rule but does so incorrectly, or code that appears to follow the
  convention but actually circumvents it. This distinction matters because
  standard negatives test obvious violations while bypass-attempt negatives
  test adversarial or accidental circumvention.

  **For a rule "encryption key must come from environment":**
  - Standard negative: `key := "hardcoded-secret"` (obvious violation)
  - Bypass-attempt negative: `key := base64Decode("aGFyZGNvZGVkLXNlY3JldA==")` (obfuscated hardcoded value)
  - Bypass-attempt negative: `key := os.Getenv("KEY"); if key == "" { key = "fallback-secret" }` (env with hardcoded fallback)

  **For a rule "API endpoints must use auth middleware":**
  - Standard negative: no middleware at all
  - Bypass-attempt negative: middleware applied but with `next.ServeHTTP(w, r)` always called regardless of auth result (no-op middleware)
  - Bypass-attempt negative: auth check in the handler body instead of middleware (circumvents the architectural pattern)

  Bypass-attempt fixtures are mechanically identified in the manifest via
  `bypass_attempt: true` on negative fixture entries (see BUNDLE-005 Phase 6).
  This guidance appears here in the authoring contract so extraction agents
  know what to produce — not just in the validation pipeline.
  [Source: round 2 P1]

- **DD-28: Pack rules execute as part of `backstop gate`, not as a separate
  workflow.** At consumption time, a pack's enforcement is invisible
  infrastructure — the consumer runs `backstop gate --all` and the pack's
  rules execute alongside everything else in the existing code check pipeline.
  Semgrep rules from the pack feed into the semgrep pass. Language-native tool
  configs from the pack are merged into the tool's config and execute in the
  lint/build passes. Custom validators from the pack execute in their own pass.
  The consumer never runs `pack check` or `pack test` — those are the author's tools for
  verifying pack well-formedness. The consumer just runs the gate and the
  pack's enforcement is included. This means the loader must resolve installed
  packs, extract their rules and tool configs, and merge them into the gate's
  check pipeline at runtime. From the consumer's perspective, packs don't exist
  as a concept they interact with — enforcement just works.

- **DD-41:** backstop.yml declares ONE version per pack — the current
  enforced version. All code is validated against the current version, not
  the version it was written under. Specs record which pack version they
  were written against as audit history/provenance, not as a compliance
  shield. When a consumer bumps the pack version in backstop.yml, the gate
  immediately enforces the new version against all new code. Existing
  violations from the version bump are captured in a remediation bundle,
  not blocking.

- **DD-42:** Pack version upgrades auto-generate a remediation bundle.
  `backstop pack upgrade <pack>@<version>` scans the codebase against the
  new version, surfaces all new violations, and creates a bundle scoping
  the remediation work. The bundle contains: what changed between versions
  (new rules, tightened rules, removed rules), which files are affected and
  violation counts per rule, grouped by package/domain. The bundle starts
  at maturity `defined` (problem is fully scoped, no OQs needed) and
  follows the normal backstop workflow: specs, plans, agents implement
  fixes, gate verifies. Pack upgrades are first-class work items in the
  backstop lifecycle, not side effects tracked in spreadsheets.

- **DD-43:** Enforcement pack semver — adding rules is a major version
  bump. For enforcement packs, breaking changes are anything that causes
  previously-compliant code to become non-compliant:
  - Major (breaking, generates remediation bundle): adding a new rule,
    tightening an existing rule (catches more cases), removing a positive
    fixture scenario (a "correct" pattern no longer accepted), changing
    risk class upward.
  - Minor (non-breaking): adding new positive fixtures (another correct
    way), loosening a rule (fewer catches), improving documentation,
    adding bypass-attempt fixtures.
  - Patch: fixing false positives, fixing incorrect fixtures, typos,
    metadata.
  Key insight: "adding a rule" is breaking because the consumer's
  previously-green build may go red. This inverts normal semver intuition
  where adding is minor. `backstop pack upgrade` uses semver to decide
  behavior: patch/minor auto-upgrade silently, major generates a
  remediation bundle.

- **DD-44:** Code pack rules must relate to the pack's code content —
  bidirectional co-occurrence. Every scaffold must have at least one rule
  (already DD-16). Additionally, every rule in a code pack must reference
  at least one scaffold or SDK via `pairs_with`. A rule that doesn't pair
  with any code content in a code pack is a validation error — it belongs
  in an enforcement pack, not a code pack. This prevents code packs from
  sneaking in arbitrary enforcement unrelated to their scaffolds. On
  upgrade, consumers only get violations relevant to the code patterns the
  pack provides, not random enforcement that slipped in.

- **DD-45:** `.standard.md` is optional documentation, not a required pack
  artifact. Semgrep rule YAML (`rules/*.yml`) is the source of truth for
  rule patterns, written directly by the author or agent. `.standard.md`
  files are optional prose documentation explaining rationale, examples,
  and context for humans. The `standard:` field in the manifest is
  optional — filepath, inline string, or omitted. `pack compile` in its
  original form (`.standard.md` -> semgrep YAML) is deprecated — the
  authoring workflow is: write semgrep YAML directly, declare it in the
  manifest, write fixtures, validate. Agent context (use_when, assumes,
  pairs_with, claims) lives in the manifest, not in `.standard.md`.

- **DD-46:** Pack directories include a `go.mod` (or language equivalent)
  for fixture dependencies. Fixtures import real packages (`gorm.io/gorm`,
  `go.uber.org/zap`, etc.) and need those dependencies resolved. The pack
  ships a `go.mod` as an implementation detail — `pack new` creates it
  during scaffolding, `pack test` runs `go mod tidy` automatically
  before fixture execution if deps are stale. The pack author never runs Go
  toolchain commands directly for pack management — the CLI abstracts the
  toolchain. Authors write fixtures with whatever imports they need; `pack
  validate` handles resolution.

- **DD-47:** `pack add` merges tool_config additively and escalates on
  conflict. When installing a pack, backstop reads the consumer's existing
  tool config files (`.golangci.yml`, `pyproject.toml`, etc.) and compares
  with the pack's `tool_config:` declarations. Additive changes (enabling a
  linter that wasn't configured) are merged automatically. Conflicts (pack
  wants a setting that contradicts the consumer's existing config) cause
  `pack add` to stop and surface the specific conflict for the consumer to
  resolve. No silent overrides in either direction. This follows backstop's
  core principle: explicit decisions, nothing silent. The consumer sees
  exactly what conflicts and makes a conscious choice.

- **DD-48:** Namespaced rule IDs use `pack-name/rule-id` format. On load,
  the loader prefixes every rule ID with the pack's canonical name using
  slash delimiters: `acme/go-http-standards/err-001`. This is the ID
  consumers see in violation output and what specs reference. Version is
  NOT embedded in the ID — it's tracked in the lockfile and backstop.yml.
  IDs are stable across version bumps. This follows industry convention:
  ESLint (`@typescript-eslint/no-unused-vars`), semgrep
  (`r/go.lang.security.audit.xss`), golangci-lint (`gocritic/hugeParam`).

- **DD-49:** Pack authoring has two verification commands: `pack check` and
  `pack test`. `pack check` runs instant structural verification — manifest
  parsing, field validation, coherence (claims/fixtures exist), archetype
  constraints, layer enforcement, risk class requirements. This is the agent's
  tight inner loop — runs in seconds, gives fast feedback on manifest issues.
  `pack test` runs fixture execution — semgrep --test, tool_config tool
  execution, layer 3 validator execution, scaffold rendering and test
  execution. This is the "prove it works" step — slower, runs external tools.
  The author workflow is: write -> `pack check` (fast loop until structurally
  clean) -> `pack test` (prove fixtures pass) -> repeat. `pack check`
  corresponds to BUNDLE-005 phases 1, 2, 4, 5, 6. `pack test` corresponds to
  BUNDLE-005 phases 1-6 (all phases, including phase 3 fixture execution).
  `pack test` re-runs phases 1/2/4/5/6 as a precondition before running
  phase 3, rather than trusting that `pack check` was run recently. Both must
  pass for a pack to be loadable.

- **DD-50:** Pack naming convention. Pack names are two-part: `org/pack-name`.
  Accept alphanumeric characters and hyphens, any casing. Normalized to
  lowercase internally for rule namespacing, lockfile keys, and matching.
  Original casing preserved in display output. Must be two-part (`org/name`).
  Local path packs use the directory name as the pack name. `pack check` warns
  on mixed case but does not block. Examples: `acme/go-http-standards`,
  `42crunch/api-security`, `slotly/go-standards`. This maps naturally to git
  repo paths — `pack add acme/go-http-standards` can infer
  `github.com/acme/go-http-standards`.

- **DD-51:** Canonical pack directory layout enforced by `pack check`. The
  following layout is mandatory. `pack new` scaffolds it exactly. `pack check`
  rejects deviations.

  ```
  pack.yml          (required)
  go.mod            (required, managed by CLI)
  rules/            (required if layer 2 rules exist)
  fixtures/rules/   (required, one lowercase subdir per rule ID)
  standards/        (optional, .standard.md documentation)
  scaffolds/        (required if archetype is code)
  validators/       (required if layer 3 rules exist)
  ```

  Files must live in their designated directories. A semgrep rule YAML in
  `standards/` or a fixture in `rules/` is a layout violation. This strict
  layout means agents produce consistent output, `pack check` can validate
  without parsing every path, and consumers always know where to find things.

- **DD-52:** Three commands for testing packs at different scopes. `pack test`
  runs fixtures — the author's verification that rules catch what they claim.
  `pack try <project-path>` runs the pack's rules against a real codebase —
  the author's exploration of how their rules perform on real code, no gate,
  no other packs, just this pack against that code. `backstop gate --pack
  <name>` runs the gate scoped to a specific pack — the consumer's way to see
  what one pack catches without noise from other packs. Three distinct
  commands, three distinct intents: fixture verification, real-world
  exploration, scoped enforcement.

## Design Notes (Demoted from Requirements)

These are architectural invariants that inform the design but are not
mechanically testable against the manifest:

- **Network isolation at load (ex-REQ-028, DD-5):** Packs are
  network-isolated at load. No fetches at validate, compile, or run time.
  All inputs must be in the pack or in explicitly-added sibling packs.
  This is an architectural invariant enforced by the loader, not something
  a unit test against the manifest verifies.

- **Packs ship net-new rules only (ex-REQ-040, DD-23):** Language-native
  built-in rules are the implied floor, not enumerated by the pack. Packs
  layer custom enforcement on top of the language toolchain's default
  ruleset. "Net-new" is not mechanically verifiable at the manifest level.

## Cross-Bundle References

The following concerns were originally drafted as BUNDLE-004 requirements
but belong to other bundles. Preserved here for traceability:

- **Validation as hard precondition for loading (BUNDLE-005):** Both pack
  check and pack test must pass at 100% before a pack is loadable.
  Unvalidated packs cannot be used. [ex-REQ-029, DD-2]

- **tool_config merge on pack add (BUNDLE-006 REQ-004):** Additive merge
  of tool_config on `pack add`; conflicts cause `pack add` to stop and
  surface the specific conflict. No silent overrides. [ex-REQ-032, DD-47]

- **Enforcement pack semver (BUNDLE-006 REQ-025):** Adding a new rule or
  tightening an existing rule is a major version bump. Adding positive
  fixtures or loosening a rule is minor. Fixing false positives or typos
  is patch. [ex-REQ-033, DD-43]

- **Pack upgrade generates remediation bundle (BUNDLE-006 REQ-014):**
  `backstop pack upgrade <pack>@<version>` scans the codebase against
  the new version and auto-generates a remediation bundle. [ex-REQ-034,
  DD-42]

- **backstop.yml declares one version per pack (BUNDLE-006 REQ-023):**
  One enforced version per pack. Existing violations from a version bump
  are captured in a remediation bundle, not blocking. [ex-REQ-035, DD-41]

- **Pack rules execute as part of backstop gate (gate/loader scope):**
  At consumption time, the loader resolves installed packs and merges
  their enforcement into the gate's check pipeline at runtime. Consumers
  never run pack check or pack test. [ex-REQ-038, DD-28]

## Open Questions

- **OQ-1: Sandbox boundary — declarative only forever? RESOLVED.** DD-13/DD-14/DD-15
  define layer 3 custom validators as the escape hatch. The remaining open
  question is the exact sandbox implementation for layer 3. Options:
  (a) process isolation (separate process, restricted syscalls);
  (b) seccomp/pledge sandboxing;
  (c) WASM runtime;
  (d) convention + review only (no mechanical sandbox, rely on validator
  constraints + LLM review).
  Lean: (a) for v1 — separate process with no network/no write/no env access
  enforced at the OS level. Simpler than WASM, stronger than convention.
  [Source: BUNDLE-001 OQ-3]
  **Resolution:** Process isolation. Layer 3 validators run in a separate
  process with no network, no filesystem writes, no env var access, enforced
  at the OS level. Simpler than WASM, stronger than convention. See DD-26.

- **OQ-2: Fixture format — semgrep native vs backstop wrapper. RESOLVED.** Lean
  entirely on semgrep `--test` format (familiar tooling, lower barrier) or
  wrap it in a backstop format (stricter claim mapping, accommodates non-
  semgrep engines like layer 3 validators)? Options:
  (a) pure semgrep `--test` format for layer 1-2, separate format for layer 3;
  (b) unified backstop wrapper format for all layers;
  (c) semgrep format with backstop metadata annotations (comments/frontmatter).
  Lean: (c) — keeps semgrep tooling working, adds claim mapping via
  annotations.
  [Source: BUNDLE-001 OQ-9]
  **Resolution:** Fixtures are plain files in whatever format their engine
  expects. Claim-to-fixture mapping lives exclusively in `pack.yml`. No
  backstop-specific annotation format — but engine-native annotations are
  expected and required (e.g., semgrep `--test` requires `// ruleid:` and
  `// ok:` comments). `pack test` reads the manifest for the mapping and
  dispatches each fixture to the correct engine (semgrep `--test`,
  language-native tool, or custom validator execution). Both positive and
  negative fixtures are lists. At least one positive required, at least one
  negative required per claim. See DD-25.

- **OQ-3: Non-semgrep rules — do they live in packs? RESOLVED.** AST checks, contract
  signature checks, test substantiveness patterns, custom Go-side checkers.
  If yes, the rule format must accommodate multiple engines and the loader
  must dispatch by engine. If no, there's a parallel "checks-that-aren't-
  packs" concept that fragments the model. Options:
  (a) yes, all in packs with engine dispatch;
  (b) layer 3 covers this already (custom validators);
  (c) only semgrep in packs, everything else is CLI-internal.
  Lean: (b) — layer 3 custom validators are the answer. The content type
  `ast_checks` is specifically a layer 3 validator per DD-14.
  [Source: BUNDLE-001 OQ-11]
  **Resolution:** Three execution paths — semgrep (layers 1-2 custom rules),
  language-native tools (golangci-lint, ruff, clippy, etc. — referenced and
  configured by the pack, executed by backstop), and custom validators
  (layer 3 escape hatch). Packs only ship what's net new — custom rules.
  Language-native built-in rules are the implied floor, not enumerated by the
  pack. Packs declare tool config requirements in `pack.yml` under a
  `tool_config:` block, and `pack add` merges them into the consumer's repo
  config files. See DD-22, DD-23, DD-24.

- **OQ-4: Layer 3 acceptance heuristics. RESOLVED.** The principle "try layers 1-2
  first" is clear, but how do we mechanically accept/reject a layer 3
  submission? Options:
  (a) self-declaration only ("I declare this can't be a semgrep rule
  because: [category]");
  (b) self-declaration + LLM reviewer cross-check;
  (c) mandatory justification field in manifest + LLM review;
  (d) allow anything in layer 3 but flag for higher scrutiny.
  Lean: (c) — the `justification:` field in the manifest (shown in examples
  above) plus LLM review at catalog submission.
  [Source: BUNDLE-001 OQ-31]
  **Resolution:** Layer 3 validators declare a `category:` field with three
  values: `presence` (checking that something exists or is missing),
  `structural` (checking filesystem conventions), and `other` (everything
  else). Categories `presence` and `structural` are auto-accepted by `pack
  check` — no `justification:` field required because semgrep mechanically
  cannot detect absence or filesystem structure. Category `other` requires a
  mandatory `justification:` field explaining why layers 1-2 can't handle it.
  LLM cross-check of `other` justifications happens at catalog submission
  time, not authoring time. See DD-14, DD-29.

- **OQ-5: Scaffold rendering and validation mechanics. RESOLVED.** Complete scaffolds
  must "pass out of the box with config-only changes." Questions: Where do
  sample config values live? How does `pack test` handle scaffolds
  needing external services? Options:
  (a) sample config in the manifest's scaffold entry + test doubles baked
  into the scaffold;
  (b) separate fixture config file per scaffold + Docker-compose fixtures;
  (c) sample config in manifest, scaffolds must be self-contained (no
  external deps in tests).
  Lean: (a) — `sample_config` in manifest, scaffold tests use test doubles.
  [Source: BUNDLE-001 OQ-32]
  **Resolution:** `sample_config` in the manifest's scaffold entry. Test
  doubles baked into the scaffold. `pack test` renders to a temp
  directory, substitutes sample_config values, runs the scaffold's test
  command. Scaffolds that need external services must ship their own test
  doubles — validation cannot depend on external service availability.

- **OQ-6: Item-level versioning granularity. RESOLVED.** Is item-level versioning
  mandatory for all content types, or only for scaffolds and SDKs? Options:
  (a) mandatory for all content types;
  (b) mandatory for scaffolds and SDKs only (consumer-facing, breaking
  changes matter), optional for rules;
  (c) optional everywhere, pack-level version is sufficient.
  Lean: (b) — rules change frequently and are covered by pack-level version +
  tamper detection.
  [Source: BUNDLE-001 OQ-33]
  **Resolution:** Three version levels: pack version (the whole artifact,
  semver), ruleset version (all rules as a cohort — rules evolve together as
  a single enforcement posture), item version (scaffolds and SDKs
  individually). Specs pin to ruleset version for enforcement and item
  version for scaffolds/SDKs. Individual rules are unversioned — they move
  as a unit under the ruleset. See DD-27.

## Spec Seeds

- **Pack Manifest Schema** — The `pack.yml` format definition, field
  validation rules, content type declarations, archetype constraints,
  versioning fields, `use_when` model, and the JSON schema for machine
  validation. The artifact a pack-authoring agent reads to know what to
  produce.

- **Pack Validation (Authoring-Time)** — `pack check` + `pack test` for authors:
  claim-fixture-rule coherence, fixture coverage, layer enforcement, scaffold
  rendering + test execution, co-occurrence rule enforcement, risk class
  requirements. The gate between "files exist" and "pack is loadable."

- **Pack Fixture Framework** — Fixture format specification, claim mapping
  annotations, positive/negative fixture requirements per content type,
  scaffold fixture expectations per tier, layer 3 validator fixture format.
  The testing contract that proves a pack does what it claims.

## References

- **BUNDLE-001** — parent vision bundle for the full pack lifecycle
- **ADR-0001** — agent-first discipline framework (manifest must be
  optimized for machine generation and parsing)
- **SPEC-012** — existing Go standards pack (the first pack to be extracted
  under this model)

## Version History

- **0.15.0** (2026-04-08): Advanced to `ready` maturity. Added success
  criteria: pack.yml schema implemented and enforced, pack new scaffolds
  canonical layout, pack check validates against the schema, authoring
  agents can produce valid manifests from requirements alone, SPEC-012
  migrated to new format. All OQs resolved, 35 requirements finalized,
  52 design decisions documented. Ready for spec generation.

- **0.14.0** (2026-04-08): Fixed review findings — removed 6 out-of-scope
  REQs (REQ-029, REQ-032-035, REQ-038) to Cross-Bundle References, demoted
  2 untestable REQs (REQ-028, REQ-040) to Design Notes, added 3 REQs for
  orphan DDs (DD-10 scaffold/SDK ownership, DD-12 rule-SDK pairing, DD-18
  fixture proof requirement), tightened REQ-031 to author-only `pack try`
  scope, fixed REQ-006 specificity (enforcement pack may contain only
  ruleset), fixed REQ-025 specificity (not required by pack check).
  Requirements renumbered REQ-001 through REQ-035. Maturity held at
  `defined`.

- **0.13.0** (2026-04-08): Advanced to `defined` maturity. Drafted 40 formal
  requirements (REQ-001 through REQ-040) covering: manifest structure and
  required fields, pack naming convention, canonical directory layout, two
  archetypes with bidirectional co-occurrence, content type declarations,
  three-layer enforcement model, layer 3 categories and auto-acceptance,
  ruleset versioning (three levels), scaffold tiers and verification,
  use_when/assumes/pairs_with/sample_config/test_command, fixture declarations
  (positive/negative lists, bypass_attempt), rule: field and rule ID convention,
  risk_class field, tool_config declarations, .standard.md as optional docs,
  go.mod for fixture deps, namespaced rule IDs on load, pack check/pack test/
  pack try commands, tool_config merge strategy, enforcement pack semver model,
  pack upgrade remediation bundles, and single enforced version in backstop.yml.
  All requirements trace to design decisions.

- **0.12.0** (2026-04-08): Added DD-50 (pack naming convention: org/pack-name,
  alphanumeric + hyphens, normalize to lowercase), DD-51 (canonical directory
  layout enforced by pack check), DD-52 (pack test / pack try / gate --pack
  for three testing scopes).

- **0.11.0** (2026-04-08): Split `pack validate` into `pack check` (structural,
  phases 1/2/4/5/6) and `pack test` (fixture execution, all phases including
  phase 3). DD-49 added. References throughout updated.

- **0.10.0** (2026-04-08): Added DD-48 — namespaced rule IDs use
  pack-name/rule-id format, matching industry convention (ESLint,
  semgrep, golangci-lint). Version not embedded in ID.

- **0.9.0** (2026-04-08): Added DD-47 — tool_config merge strategy:
  additive auto-merge, escalate on conflict.

- **0.8.0** (2026-04-08): Added DD-46 — pack directories include go.mod
  for fixture deps, managed automatically by pack new and pack test.

- **0.7.0** (2026-04-08): Added pack versioning model (DD-41: single
  enforced version in backstop.yml, DD-42: auto-generated remediation
  bundles on pack upgrade, DD-43: enforcement pack semver with "adding
  rules is breaking"). Added bidirectional co-occurrence (DD-44: code pack
  rules must pair with scaffolds/SDK). Deprecated .standard.md as
  compilation source (DD-45: semgrep YAML is source of truth, .standard.md
  is optional documentation). Revised DD-16 for bidirectional co-occurrence
  check.

- **0.6.0** (2026-04-08): Round 2 P1 fixes. tool_config standalone entries
  require `risk_class:` like any other rule (DD-33 revised). Added bypass-
  attempt fixture guidance with concrete examples for security rules (DD-40).
  `pairs_with.rules` references should use actual rule IDs — authoring
  guidance added to avoid dangling references (DD-20 revised). Pack.yml
  examples updated: enforcement pack shows `risk_class: correctness` on
  tool_config entry and bypass-attempt negative fixtures on security rule
  (http-002); code pack shows bypass-attempt negative fixtures on security
  rule (stripe-002). Negative fixture entries for security rules use object
  format with `path:` and optional `bypass_attempt: true`.

- **0.5.0** (2026-04-08): Comprehensive update from prototype review
  findings. Fixed category/risk_class YAML collision — `risk_class:` on all
  rules, `category:` only on layer 3 (P0). Added semgrep rule file path
  `rule:` field on layer 2 rules (DD-30, P0). Defined rule ID = semgrep ID
  in lowercase kebab-case (DD-31, P0). Specified layer 3 invocation contract
  (DD-32, P0). Fixed annotation language — engine-native annotations expected,
  no backstop-specific format (DD-25 revised, P0). Defined tool_config as
  standalone layer 1 enforcement (DD-33, P1). Kept strict co-occurrence rule —
  no infrastructure exemption (DD-16 revised, P1). Refined skeleton tier to
  structural-only validation, tests NOT executed (DD-17 revised, P1). Defined
  `standard:` field accepts filepath or inline string (DD-34, P1). Added
  `test_command:` on scaffolds (DD-35, P1). Formalized `risk_class` enum
  (DD-6 revised, P2). Fixture directory case convention (DD-36, P2).
  `pairs_with:` as single object (DD-20 revised, P2). Positive fixtures now
  lists (DD-25 revised, P2). Default ruleset version (DD-37, P2).
  `tool_config.file` is consumer-side (DD-38, P2). Removed
  `content.fixtures.directory` (DD-7 revised, P2). `sample_config` guidance
  (DD-17, P2). Claim ID convention (DD-39, P2). 12 new/revised DDs.
  Pack.yml examples fully rewritten.

- **0.4.0** (2026-04-08): Refined layer 3 acceptance heuristic based on
  presence-vs-content distinction (DD-14 revised, DD-29 added). Presence and
  structural checks are auto-accepted; only "other" category requires
  justification. Updated pack.yml examples and OQ-4 resolution.

- **0.3.0** (2026-04-08): Added DD-28 (pack rules execute as part of
  `backstop gate`, not as a separate workflow). Clarified author-time vs
  consumption-time distinction. Critical integration requirement for the
  pack→gate wiring.

- **0.2.0** (2026-04-08): All 6 OQs resolved. Added DD-22 through DD-27
  (three execution paths, net-new-only rules, tool_config, engine-native
  fixtures, process isolation sandbox, three-level versioning). Updated
  pack.yml examples with tool_config, ruleset versioning, and negative
  fixture lists. Ready for weekend prototype.

- **0.1.0** (2026-04-11): Initial bundle at `exploring`. Extracted 21 DDs
  and 6 OQs from BUNDLE-001. Drafted concrete `pack.yml` examples for both
  archetypes. Identified 3 spec seeds.
