---
title: "Pack Validation — How an Agent Verifies Its Pack Output Is Correct"
number: BUNDLE-005
created: "2026-04-08"
schema_version: bundle/v2

bundle:
  name: pack-validation
  version: "1.1.0"
  created: "2026-04-08"
  updated: "2026-04-08"
  category: feature

status:
  maturity: ready

requirements:
  - id: REQ-001
    version: "1.0.0"
    text: >
      pack check must run phases 1 (structural), 2 (coherence), 4 (archetype),
      5 (layer), and 6 (risk_class) — manifest-only checks that do not require
      external tool execution. [DD-1, DD-12]
  - id: REQ-002
    version: "1.0.0"
    text: >
      pack test must run all six phases (1-6) including phase 3 (fixture
      execution), which invokes external tools such as semgrep and language
      toolchains. [DD-1, DD-3, DD-12]
  - id: REQ-003
    version: "1.0.0"
    text: >
      Phase 1 structural validation must verify: manifest parses as valid YAML,
      all required fields present (name, version, language, archetype, content),
      field values are valid enums (archetype, version semver), language must be
      a value from the supported language set (initial supported languages: go;
      additional languages are added as language packs are developed; pack check
      must reject unrecognized language values), layer 2 rules declare a rule
      field pointing to an existing file, all content type declarations are
      valid, and all referenced file paths exist on disk. [DD-1]
  - id: REQ-004
    version: "1.0.0"
    text: >
      Phase 1 must exclude tool_config.file from file-existence checks because
      it is a consumer-side target path, not a pack-internal file. [DD-1]
  - id: REQ-005
    version: "1.0.0"
    text: >
      Phase 1 must verify that risk_class is present and a valid enum (security,
      correctness, style, perf) on every rule, including standalone tool_config
      entries that have their own id. A missing risk_class is a structural
      error. [DD-4, DD-10]
  - id: REQ-006
    version: "1.0.0"
    text: >
      Phase 2 coherence must verify: every rule has at least one claim, every
      claim has both positive and negative fixtures (lists, at least one of each),
      every fixture file exists and is non-empty, and claim IDs are unique within
      the pack. [DD-2, DD-16]
  - id: REQ-007
    version: "1.0.0"
    text: >
      Phase 2 must enforce rule ID uniqueness spanning both
      content.ruleset.rules and tool_config entries that have their own id.
      A duplicate ID across these two sources is a hard error. [DD-14]
  - id: REQ-008
    version: "1.0.0"
    text: >
      Phase 2 must verify that tool_config entries with their own rule ID also
      have claims and fixtures proving the tool catches the intended
      violation. [DD-14]
  - id: REQ-009
    version: "1.0.0"
    text: >
      Phase 2 must check that pairs_with.rules entries in scaffold declarations
      resolve to actual rule IDs in the pack. Dangling references are a coherence
      warning, not a hard error. [DD-23]
  - id: REQ-010
    version: "1.0.0"
    text: >
      Phase 2 must emit a warning (not hard error) for orphan fixture files in
      the fixture directory that are not referenced by any claim. [DD-2]
  - id: REQ-011
    version: "1.0.0"
    text: >
      Phase 3 fixture execution for layer 1-2 semgrep rules must run semgrep
      --test requiring 100% pass rate: every positive fixture must NOT trigger
      the rule, every negative fixture MUST trigger the rule. [DD-3, DD-15,
      DD-16]
  - id: REQ-012
    version: "1.0.0"
    text: >
      Phase 3 must verify that the pack rule ID exactly matches the semgrep rule
      ID in the referenced rule file. Mismatch is a hard error. [DD-18]
  - id: REQ-013
    version: "1.0.0"
    text: >
      Phase 3 fixture execution for tool_config-dependent rules must create a
      temporary module environment (copying the pack's go.mod), copy the fixture
      file in, run the configured tool, and check results. Positive fixtures must
      pass clean, every negative fixture must trigger the expected
      diagnostic. [DD-15, DD-21]
  - id: REQ-014
    version: "1.0.0"
    text: >
      Phase 3 must run go mod tidy for Go packs in the pack directory before
      any fixture execution to ensure dependencies are resolved. Language-specific
      dependency resolution commands for other languages are defined when those
      languages are added to the supported set. Failure to resolve deps is a
      Phase 3 pre-check error. [DD-21]
  - id: REQ-015
    version: "1.0.0"
    text: >
      Phase 3 fixture execution for layer 3 validators must invoke the validator
      as validator.sh <fixture-path>. For single-file input_scope the path is a
      file; for multi-file input_scope the path is a directory. Exit 0 = pass, exit
      non-zero = fail. Positive fixtures must exit 0, every negative fixture must
      exit non-zero. [DD-6, DD-15, DD-19]
  - id: REQ-016
    version: "1.0.0"
    text: >
      Phase 3 must validate complete scaffolds by rendering with sample_config
      from the manifest, then running the scaffold's test_command. Tests must
      pass. [DD-8, DD-9]
  - id: REQ-017
    version: "1.0.0"
    text: >
      Phase 3 must validate skeleton scaffolds with structural checks only:
      scaffold directory exists, expected files present, test function names
      present. Tests must NOT be run. test_command is only used for complete
      scaffolds. [DD-8, DD-20]
  - id: REQ-018
    version: "1.0.0"
    text: >
      Phase 3 must verify SDK references by checking that the provides surface
      is declared at the manifest level. SDK test suite execution is not in
      scope. [DD-9]
  - id: REQ-019
    version: "1.0.0"
    text: >
      Phase 4 archetype enforcement must verify that code packs (declaring sdk
      or scaffolds) also declare rules, and every scaffold has at least one
      enforcement rule. A code pack without enforcement rules is a hard
      error. [DD-7]
  - id: REQ-020
    version: "1.0.0"
    text: >
      Phase 4 must enforce bidirectional co-occurrence: in a code pack, every
      rule must reference at least one scaffold or SDK via pairs_with. A rule
      without code content pairing is a hard error. [DD-7]
  - id: REQ-021
    version: "1.0.0"
    text: >
      Phase 4 must verify that enforcement packs do not declare sdk or scaffolds.
      An enforcement pack with code content is a hard error. [DD-7]
  - id: REQ-022
    version: "1.0.0"
    text: >
      Phase 5 layer enforcement must verify every rule declares its layer (1, 2,
      or 3) and that risk_class is a valid enum on all rules. [DD-4, DD-5]
  - id: REQ-023
    version: "1.0.0"
    text: >
      Phase 5 must check that category (presence, structural, other) is present
      ONLY on layer 3 rules. category must NOT appear on layer 1 or layer 2
      rules. [DD-5, DD-17]
  - id: REQ-024
    version: "1.0.0"
    text: >
      Phase 5 must auto-accept layer 3 rules with category presence or
      structural (no justification required). Category other requires a mandatory
      non-empty justification field. [DD-5, DD-17]
  - id: REQ-025
    version: "1.0.0"
    text: >
      Phase 5 must verify layer 3 rules declare input_scope (single-file or
      multi-file) and a validator field pointing to an executable file. [DD-6]
  - id: REQ-026
    version: "1.0.0"
    text: >
      Phase 6 risk class enforcement must verify every security-class rule has
      at least one negative fixture with bypass_attempt true per claim. The
      negative fixture list accepts both plain path strings and objects with
      path and optional bypass_attempt boolean; pack test normalizes before
      checking. [DD-10, DD-22]
  - id: REQ-027
    version: "1.0.0"
    text: >
      Phase 6 must enforce independent fixture coverage per claim for
      security-class rules — no shared fixtures across security claims. [DD-10]
  - id: REQ-028
    version: "1.0.0"
    text: >
      The validation pipeline must run phases in strict dependency order
      (structural > coherence > fixtures > archetype > layer > risk_class).
      If phase N fails, phases N+1 through 6 must be skipped. [DD-12]
  - id: REQ-029
    version: "1.0.0"
    text: >
      Validation must be idempotent and side-effect-free. Running pack check or
      pack test twice on the same pack must produce the same pass/fail result,
      the same errors, and the same warnings. Timing fields (duration_ms) are
      excluded from the idempotency guarantee. No files modified, no state
      persisted, no network calls. The pack directory is read-only
      input. [DD-13]
  - id: REQ-030
    version: "1.0.0"
    text: >
      Validation output must be JSON by default per ADR-0001. Every error must
      include: the failing phase, the specific check name, the offending item
      (rule ID, claim ID, file path), a human-readable message, a fix hint the
      agent can act on, and the manifest_path where the problem
      originates. [DD-11]
  - id: REQ-031
    version: "1.0.0"
    text: >
      Validation must support a --format=text flag for human-readable output as
      a secondary format. The primary consumer is an agent. [DD-11]
  - id: REQ-032
    version: "1.0.0"
    text: >
      When a negative fixture fails to trigger its rule in Phase 3, the error
      must include a fix hint with engine-limitation guidance: explaining that
      the fixture may represent a pattern the rule engine cannot detect and
      should be removed and documented rather than shipped as an untestable
      fixture. [DD-16]
  - id: REQ-033
    version: "1.0.0"
    text: >
      Errors must be reported in pipeline phase order, then by manifest order
      within a phase, to enable the agent to work through them
      top-down. [DD-11, DD-12]
  - id: REQ-034
    version: "1.0.0"
    text: >
      Phase 3 must execute layer 3 validators in a restricted process
      environment that prevents filesystem writes outside the pack directory,
      network access, and environment variable access. Sandbox violations are
      hard errors. [DD-6]

problem:
  summary: >
    BUNDLE-004 tells an agent what to produce when extracting a pack — the
    manifest schema, content types, archetype rules, scaffold tiers, fixture
    requirements. But it does not tell the agent how to verify that its output
    is correct. Without a validation contract, the agent is guessing whether
    its pack is loadable. Validation is a hard precondition for loading
    (BUNDLE-001 DD-3): an unvalidated pack cannot be used. This bundle defines
    the validation pipeline — what checks run, in what order, what passes,
    what fails, and what the errors look like — so an authoring agent can
    produce a pack, validate it, fix problems, and iterate until the pack is
    mechanically proven correct. Together, BUNDLE-004 (what to produce) and
    this bundle (how to verify it) are the complete contract for the pack
    extraction prototype.

  user_story: >
    As an agent that just extracted a pack from an existing codebase, I need
    to validate that my output is structurally correct, that every rule has
    claims backed by fixtures, that fixture coverage is 100%, that scaffolds
    meet their tier's completeness requirements, that any custom validators
    are justified and sandboxed, and that archetype constraints are satisfied
    — so I can fix problems and iterate until the pack is valid.

  success_criteria:
    - pack check command is implemented and runs phases 1 (structural), 2 (coherence), 4 (archetype), 5 (layer), and 6 (risk_class)
    - pack test command is implemented and runs all six phases including phase 3 (fixture execution)
    - Three fixture execution paths work correctly — semgrep --test for layer 1-2 rules, tool_config in a temporary module environment, and layer 3 validators in a sandboxed process
    - JSON output format matches the specified schema with phase statuses, error objects (check, item, message, fix_hint, manifest_path), and warnings
    - Early termination works — phase N failure skips phases N+1 through 6, preventing cascading errors
    - The prototype enforcement pack and code packs from Slotly validate successfully through the full pipeline

solution:
  approach: >
    A validation pipeline (pack check + pack test) that checks structural
    completeness, claim-fixture-rule coherence, fixture execution (semgrep
    --test), archetype constraint enforcement, layer-model
    compliance, and risk-class requirements. `pack check` runs phases
    1/2/4/5/6 (manifest-only checks, fast). `pack test` runs all phases
    1-6 including phase 3 fixture execution (slower, external tools).
    Output is machine-parseable (JSON, per ADR-0001) and actionable —
    every error includes the failing check, the offending item, and a fix
    hint — so the authoring agent can fix-and-retry in a tight loop. The
    pipeline runs checks in dependency order: structural checks first
    (malformed manifest blocks everything else), then content coherence,
    then fixture execution, then archetype/layer/risk enforcement. Early
    termination on structural failure avoids noisy cascading errors.

  assumptions:
    - BUNDLE-004 pack manifest schema is finalized — pack check and pack test validate against that schema
    - semgrep is available on PATH for phase 3 fixture execution of layer 1-2 rules
    - Go toolchain is available on PATH for go mod tidy and tool_config fixture execution in Go packs
    - The pack directory structure follows BUNDLE-004 DD-51 canonical layout
    - ADR-0001 agent-first discipline applies — JSON is the primary output format, human-readable is secondary
    - Layer 3 validator sandboxing uses OS-level process isolation (no container runtime required)
    - The embedded Go standards pack (SPEC-012) and Slotly extraction packs exist as validation targets
---

# Pack Validation

## Current Thinking

### The validation problem

An agent extracting a pack from a real codebase produces files: a `pack.yml`
manifest, semgrep rules, fixtures, possibly scaffolds and validators. But
"files exist" is not "pack is valid." BUNDLE-001 DD-3 makes validation a
hard precondition for loading — no pack enters the system without mechanical
proof that it does what it claims. BUNDLE-004 defines the authoring contract
(what to produce); this bundle defines the verification contract (how to
prove it's correct).

The validation pipeline must be fast enough for an agent to run it in a
tight loop: produce pack, validate, read errors, fix, re-validate. If
validation takes minutes, the agent can't iterate effectively. If errors
are vague ("validation failed"), the agent can't fix them. The pipeline
must be fast and the output must be specific.

### The validation pipeline: `pack check + pack test`

`pack check + pack test` runs a sequence of checks in dependency order. Each phase
depends on the previous phase passing. If a phase fails, subsequent phases
are skipped to avoid cascading noise.

**Phase 1 — Structural validation**
- Manifest parses as valid YAML
- All required fields present (`name`, `version`, `language`, `archetype`,
  `content`)
- Field values are valid (archetype is `enforcement` or `code`, version is
  semver, language is from the supported set — initially `go`)
- `risk_class:` field exists on every rule and is a valid enum (`security`,
  `correctness`, `style`, `perf`) — this includes `tool_config` entries with
  their own `id:`. A standalone tool_config rule missing `risk_class:` is a
  structural validation error.
- Layer 2 rules declare a `rule:` field pointing to an existing semgrep
  YAML file on disk
- All content type declarations are valid (known type, required subfields
  present)
- All file paths referenced in the manifest exist on disk — EXCEPT
  `tool_config.file` which is a consumer-side target path (not checked)
- No `content.fixtures.directory` field (removed — fixture paths are inline
  on claims)

**Phase 2 — Claim-fixture-rule coherence**
- Every rule has at least one claim
- Every claim has both positive and negative fixtures (both are lists, at
  least one of each required)
- Every fixture file referenced in a claim exists and is non-empty
- No orphan fixtures (files in the fixture directory not referenced by any
  claim) — warning, not hard error
- Claim IDs are unique within the pack
- Rule IDs are unique within the pack — uniqueness spans BOTH
  `content.ruleset.rules` AND `tool_config` entries that have their own
  `id:`. A tool_config entry with `id: slotly-004` and a ruleset rule
  with `id: slotly-004` is a uniqueness violation.
- `tool_config` entries with their own rule ID must also have claims and
  fixtures proving the tool catches the intended violation
- `pairs_with.rules` entries in scaffold declarations must resolve to
  actual rule IDs in the pack's `content.ruleset.rules` or `tool_config`
  entries. Dangling references are a coherence warning (not hard error —
  the referenced rule might be in a different pack the consumer also loads).

**Phase 3 — Fixture execution (three paths)**
- For layer 1-2 semgrep rules: `semgrep --test` at 100%
  pass rate. Every positive fixture must NOT trigger the rule. Every negative
  fixture (all items in the list) MUST trigger the rule. Additionally, `pack
  test` reads the semgrep rule file declared in `rule:` and verifies
  the rule ID matches the pack rule ID (DD-31). Mismatch is a hard error.
  When a negative fixture fails to trigger its rule, the error includes a
  fix hint: "Negative fixture <path> did not trigger rule <id>. If this
  fixture represents a pattern the rule engine genuinely cannot detect (e.g.,
  obfuscated values, indirect references, cross-file dataflow), remove it
  from the fixture list and document the limitation in the rule's .standard.md
  file. Not every bypass can be caught by static analysis — documenting known
  gaps is more valuable than shipping untestable fixtures."
- For tool_config-dependent rules (including standalone tool_config with own
  rule ID): execute the language-native tool (e.g., golangci-lint) against
  each fixture. Fixture files cannot be validated in isolation — tools like
  golangci-lint require a buildable project (Go module with go.mod, proper
  package declaration). `pack check + pack test` Phase 3 creates a temporary module
  environment around tool_config fixtures: (1) creates a temp directory with
  `go mod init pack-fixture-test` for Go packs, (2) copies the
  fixture file into the temp module, (3) runs the configured tool against the
  temp module, (4) checks results. This is the same pattern as scaffold
  rendering — `pack check + pack test` creates a controlled environment, runs the
  check, tears it down. Fixture files must be valid Go source files with
  proper `package` declarations but do NOT need their own `go.mod`.
  Positive fixtures must pass clean. Every negative fixture in
  the list must trigger the expected diagnostic.
- For layer 3 validators: invoke as `validator.sh <fixture-path>` (DD-32).
  For single-file `input_scope`: fixture-path is a file. For multi-file
  `input_scope`: fixture-path is a directory containing the complete expected
  structure. Exit 0 = pass (no violation found). Exit non-zero = fail
  (violation found). Positive fixtures must exit 0. Every negative fixture
  must exit non-zero. `pack check + pack test` creates no intermediate structure —
  the fixture is the complete input.
- For complete scaffolds: render with `sample_config` from the manifest,
  then run the scaffold's `test_command`. Tests must pass.
- For skeleton scaffolds: structural validation only — `pack check + pack test`
  checks scaffold directory exists, expected files present, test function
  names present. Tests are NOT run. `test_command` is only used for complete
  scaffolds. The consumer's own `backstop gate` handles behavioral
  validation after implementation.
- For SDK references: verify the `provides:` surface is declared (manifest-
  level check only — SDK test suite execution is the SDK's own CI
  responsibility, not `pack check + pack test`).

**Phase 4 — Archetype constraint enforcement**
- If archetype is `code` (manifest declares `sdk` or `scaffolds`): verify
  `rules` are also declared. Every scaffold must have at least one
  enforcement rule — no infrastructure exemption. If an agent can't think
  of a rule for a scaffold, the scaffold isn't opinionated enough to be in
  a pack. A code pack without enforcement rules is a hard error.
- If archetype is `code`: verify bidirectional co-occurrence — every rule
  must reference at least one scaffold or SDK via `pairs_with`. A rule
  without code content pairing in a code pack is a Phase 4 hard error.
  Error message: "Rule <id> does not reference any scaffold or SDK via
  pairs_with. General enforcement rules belong in an enforcement pack, not
  a code pack."
- If archetype is `enforcement`: verify no `sdk` or `scaffolds` are
  declared. An enforcement pack with code content is a hard error (wrong
  archetype declaration).

**Phase 5 — Layer enforcement**
- Every rule declares its layer (1, 2, or 3)
- `risk_class:` is checked on ALL rules (valid enum: security, correctness,
  style, perf) — this is a distinct field from `category:`
- `category:` is checked ONLY on layer 3 rules (`presence` | `structural` |
  `other`). It must NOT appear on layer 1 or layer 2 rules.
- If category is `presence` or `structural`: no `justification:` required,
  auto-accepted — semgrep cannot detect absence or filesystem structure
- If category is `other`: mandatory `justification:` field, validated as
  non-empty
- Layer 3 rules must have an `input_scope` declaration (`single-file` or
  `multi-file`)
- Layer 3 rules must have a `validator` field pointing to an executable file
- Layer 3 validators must not write files, make network calls, or access
  environment variables (verified by sandbox execution in phase 3)

**Phase 6 — Risk class enforcement**
- Every rule declares a `risk_class` (security, correctness, style, perf)
- Security-class rules carry stricter requirements:
  - Mandatory bypass-attempt negative fixtures, mechanically identified via
    `bypass_attempt: true` on negative fixture entries in the manifest.
    Security-class rules must have at least one negative fixture with
    `bypass_attempt: true` per claim. This changes negative fixture entries
    for security rules from a flat list of path strings to a list of objects
    with `path:` and optional `bypass_attempt:` boolean. Non-security rules
    can still use plain path strings for negative fixtures. The manifest
    schema accepts both forms (string or object with `path:` key) in
    negative fixture lists; `pack check + pack test` normalizes before checking.
    Example:
    ```yaml
    negative:
      - path: fixtures/rules/cfg-001/hardcoded-secret.go
      - path: fixtures/rules/cfg-001/obfuscated-secret.go
        bypass_attempt: true
      - path: fixtures/rules/cfg-001/fallback-secret.go
        bypass_attempt: true
    ```
  - Stricter claim coverage (every security claim must be independently
    fixtureed — no shared fixtures across security claims)

### The validation feedback loop

The pipeline is designed for an agent-driven fix-and-retry loop:

```
agent produces pack
  -> runs `pack check + pack test`
  -> reads JSON output
  -> fixes errors (one or more)
  -> runs `pack check + pack test` again
  -> repeats until all phases pass
```

The loop must be tight. This means:
1. **Fast execution.** Structural and coherence checks are instant. Fixture
   execution (semgrep --test) is the bottleneck — keep fixture sets small and
   focused during authoring.
2. **Specific errors.** Every error identifies the failing check, the
   offending manifest path or file, and what the agent should do to fix it.
3. **Stable ordering.** Errors are reported in pipeline phase order, then by
   manifest order within a phase. The agent can work through them top-down.
4. **No cascading noise.** Phase failure stops the pipeline. The agent fixes
   phase N before seeing phase N+1 errors.
5. **Idempotent.** Running `pack check + pack test` twice on the same pack produces
   the same pass/fail result, errors, and warnings. Timing fields
   (`duration_ms`) are excluded. No side effects, no state.

### Validation output format (agent-first per ADR-0001)

Output is JSON. The primary consumer is an agent, not a human reading a
terminal. Humans can pipe through `jq` or use `--format=text` for a
human-readable summary.

**Passing validation:**
```json
{
  "status": "pass",
  "pack": "acme/go-http-standards",
  "version": "1.0.0",
  "phases": [
    {"phase": "structural", "status": "pass", "checks": 12, "duration_ms": 8},
    {"phase": "coherence", "status": "pass", "checks": 6, "duration_ms": 3},
    {"phase": "fixtures", "status": "pass", "checks": 8, "duration_ms": 1204},
    {"phase": "archetype", "status": "pass", "checks": 2, "duration_ms": 1},
    {"phase": "layer", "status": "pass", "checks": 4, "duration_ms": 1},
    {"phase": "risk_class", "status": "pass", "checks": 3, "duration_ms": 1}
  ],
  "errors": [],
  "warnings": []
}
```

**Failing validation:**
```json
{
  "status": "fail",
  "pack": "acme/go-http-standards",
  "version": "1.0.0",
  "phases": [
    {"phase": "structural", "status": "pass", "checks": 12, "duration_ms": 8},
    {"phase": "coherence", "status": "fail", "checks": 6, "duration_ms": 4},
    {"phase": "fixtures", "status": "skipped", "reason": "coherence failed"},
    {"phase": "archetype", "status": "skipped", "reason": "coherence failed"},
    {"phase": "layer", "status": "skipped", "reason": "coherence failed"},
    {"phase": "risk_class", "status": "skipped", "reason": "coherence failed"}
  ],
  "errors": [
    {
      "phase": "coherence",
      "check": "claim_has_fixtures",
      "rule": "http-001",
      "claim": "http-001-clm-002",
      "message": "Claim http-001-clm-002 on rule http-001 has no negative fixture",
      "fix_hint": "Add a negative fixture file at fixtures/rules/http-001/ showing code that violates this claim, and reference it in the 'negative' list under http-001-clm-002 in pack.yml",
      "manifest_path": "content.ruleset.rules[0].claims[1].fixtures.negative"
    },
    {
      "phase": "coherence",
      "check": "fixture_file_exists",
      "rule": "http-002",
      "claim": "http-002-clm-001",
      "message": "Fixture file fixtures/rules/http-002/good-auth-chain.go does not exist",
      "fix_hint": "Create the file at fixtures/rules/http-002/good-auth-chain.go containing code that satisfies claim http-002-clm-001",
      "manifest_path": "content.ruleset.rules[1].claims[0].fixtures.positive"
    },
  ],
  "warnings": [
    {
      "phase": "coherence",
      "check": "orphan_fixtures",
      "message": "2 fixture files in fixtures/ are not referenced by any claim",
      "files": ["fixtures/rules/old-test.go", "fixtures/rules/scratch.go"],
      "fix_hint": "Remove these files or reference them in a claim"
    }
  ]
}
```

**Failing validation (fixture execution — rule ID mismatch):**
```json
{
  "status": "fail",
  "pack": "acme/go-http-standards",
  "version": "1.0.0",
  "phases": [
    {"phase": "structural", "status": "pass", "checks": 14, "duration_ms": 9},
    {"phase": "coherence", "status": "pass", "checks": 8, "duration_ms": 4},
    {"phase": "fixtures", "status": "fail", "checks": 10, "duration_ms": 1102},
    {"phase": "archetype", "status": "skipped", "reason": "fixtures failed"},
    {"phase": "layer", "status": "skipped", "reason": "fixtures failed"},
    {"phase": "risk_class", "status": "skipped", "reason": "fixtures failed"}
  ],
  "errors": [
    {
      "phase": "fixtures",
      "check": "rule_id_matches_semgrep",
      "rule": "http-001",
      "message": "Pack rule ID 'http-001' does not match semgrep rule ID 'HTTP_001' in rules/http-001.yml",
      "fix_hint": "The id field in rules/http-001.yml must exactly match the id in pack.yml. Update the semgrep rule file to use 'http-001' (lowercase kebab-case)",
      "manifest_path": "content.ruleset.rules[0].rule"
    }
  ],
  "warnings": []
}
```

**Failing validation (fixture execution — negative fixture list):**
```json
{
  "status": "fail",
  "pack": "acme/go-http-standards",
  "version": "1.0.0",
  "phases": [
    {"phase": "structural", "status": "pass", "checks": 14, "duration_ms": 8},
    {"phase": "coherence", "status": "pass", "checks": 8, "duration_ms": 3},
    {"phase": "fixtures", "status": "fail", "checks": 10, "duration_ms": 1102},
    {"phase": "archetype", "status": "skipped", "reason": "fixtures failed"},
    {"phase": "layer", "status": "skipped", "reason": "fixtures failed"},
    {"phase": "risk_class", "status": "skipped", "reason": "fixtures failed"}
  ],
  "errors": [
    {
      "phase": "fixtures",
      "check": "negative_fixture_triggers_rule",
      "rule": "http-001",
      "claim": "http-001-clm-001",
      "message": "Negative fixture 2 of 3 did not trigger rule http-001: fixtures/rules/http-001/nil-error-swallow.go passed when it should have failed",
      "fix_hint": "The negative fixture must contain code that violates claim http-001-clm-001. Update the fixture so it actually demonstrates the violation, or remove it from the negative list if it is not a valid violation mode",
      "manifest_path": "content.ruleset.rules[0].claims[0].fixtures.negative[1]"
    }
  ],
  "warnings": []
}
```

### What validation does NOT cover

This bundle deliberately excludes:

- **Distribution validation** — lockfile integrity, git-ref pinning, cache
  consistency. That's BUNDLE-001's distribution scope.
- **Composition validation** — rule collisions, fixture composition across
  packs, severity resolution. That's multi-pack, not single-pack.
- **Supply chain scanning** — OSV-Scanner, sigstore, Trivy/Grype. Those are
  supply chain subchecks for consumption-time, not authoring-time validation.
- **LLM review** — the curated catalog's LLM reviewer is a separate artifact.
  `pack check + pack test` is mechanical checks only.
- **Publishing attestations** — deferred to BUNDLE-002.
- **Consumer-side verification** — `pack verify` (consumer checking an
  installed pack) is different from `pack check + pack test` (author checking their
  own pack).

## Draft Requirements

34 formal requirements (REQ-001 through REQ-034) are defined in the frontmatter
`requirements:` block. They decompose as follows:

| Phase | Requirements | Coverage |
|-------|-------------|----------|
| Commands | REQ-001, REQ-002 | pack check (phases 1/2/4/5/6), pack test (all phases) |
| Phase 1 — Structural | REQ-003, REQ-004, REQ-005 | YAML parsing, required fields, enums, file existence, tool_config.file exclusion, risk_class on all rules |
| Phase 2 — Coherence | REQ-006, REQ-007, REQ-008, REQ-009, REQ-010 | Claims, fixtures, unique IDs, tool_config traceability, pairs_with references, orphan warnings |
| Phase 3 — Fixtures | REQ-011, REQ-012, REQ-013, REQ-014, REQ-015, REQ-016, REQ-017, REQ-018, REQ-034 | Semgrep execution, rule ID matching, tool_config temp module, go.mod tidy, layer 3 validators, complete scaffolds, skeleton scaffolds, SDK references, layer 3 sandbox enforcement |
| Phase 4 — Archetype | REQ-019, REQ-020, REQ-021 | Code pack rules required, bidirectional co-occurrence, enforcement pack no code content |
| Phase 5 — Layer | REQ-022, REQ-023, REQ-024, REQ-025 | Layer declaration, risk_class on all, category on layer 3 only, auto-acceptance, input_scope/validator |
| Phase 6 — Risk class | REQ-026, REQ-027 | Bypass-attempt fixtures for security rules, independent fixture coverage |
| Cross-cutting | REQ-028, REQ-029, REQ-030, REQ-031, REQ-032, REQ-033 | Early termination, idempotency, JSON output, --format=text, negative fixture guidance, error ordering |

## Draft Design Decisions

- **DD-1:** Validation is a hard precondition for loading. Unvalidated packs
  cannot be used. `pack check + pack test` must pass at 100% before a pack is loadable.
  There is no "load anyway" flag, no override, no escape hatch. An invalid
  pack is not a pack. Rationale: if validation is optional, it doesn't happen
  — agents skip optional steps. [Source: BUNDLE-001 DD-3]

- **DD-2:** Claim-fixture-rule mapping is enforced mechanically. Every rule
  must have claims. Every claim must have both positive and negative fixtures.
  Every fixture must exercise its claim. A rule that claims X but whose
  fixture does not exercise X is a validation failure. This is the core
  coherence guarantee that makes packs trustworthy — you can't ship a rule
  without proving it catches what it says it catches.
  [Source: BUNDLE-001 DD-4]

- **DD-3:** Integrate existing tools as authoring-time subchecks. For layer
  1-2 rules, `semgrep --test` is the fixture execution engine. For layer 3
  validators, direct execution in a sandbox. For scaffolds, rendering + test
  execution via the pack's language toolchain. `pack check + pack test` orchestrates
  these tools; it does not reimplement them. This DD covers only the
  authoring-time subset — supply chain scanners (OSV, sigstore, Trivy) are
  consumption-time subchecks outside this bundle's scope.
  [Source: BUNDLE-001 DD-7, authoring-time subset only]

- **DD-4:** Enforcement is layered and validation enforces the layering.
  Layer 1 (built-in tool rules) > Layer 2 (custom declarative rules) >
  Layer 3 (custom validators). `pack check + pack test` checks that every rule
  declares its layer, and that layer 3 rules include the required
  justification, input scope, and validator executable. The pipeline does
  not mechanically verify that a layer 3 check couldn't have been a layer 2
  rule (that's undecidable), but it does verify the justification field
  exists and the validator meets sandbox constraints.
  [Source: BUNDLE-001 DD-43]

- **DD-5:** Layer 3 acceptance uses a presence-vs-content heuristic via
  the `category:` field. This field appears ONLY on layer 3 rules and is
  distinct from `risk_class:` (which appears on ALL rules). Semgrep and
  linters validate content — they operate on code that exists. They cannot
  detect what isn't there. This gives a clean mechanical classification.
  **Auto-accepted categories (no justification needed):**
  (1) Presence/absence checks — file exists, function exists, field exists,
  pattern exists somewhere in the project. Semgrep cannot detect missing things.
  (2) Structural/filesystem checks — naming conventions, file placement,
  directory structure. Not content-level analysis. **Requires-justification
  category:** (3) Content-level checks that semgrep/linters allegedly can't
  handle — binary file inspection, encoding validation, semantic checks beyond
  pattern matching, content requiring external context. These need the
  `justification:` field explaining why existing tools can't do it. `pack
  check` checks `category:` field value and only requires `justification:`
  for category `other`.
  [Source: BUNDLE-001 DD-44, refined via presence-vs-content distinction]

- **DD-6:** Layer 3 validators must declare input scope and respect sandbox
  constraints. `input_scope: single-file` or `input_scope: multi-file` is
  mandatory. Multi-file validators carry stricter verification requirements
  (must declare which files they read and why). No layer 3 validator may
  write files, make network calls, or access environment variables — `pack
  test` enforces these constraints by executing validators in a
  restricted environment. Violations are hard errors.
  [Source: BUNDLE-001 DD-45]

- **DD-7:** Two archetypes with a strict co-occurrence rule. Code packs
  (declaring `sdk` or `scaffolds`) must also declare `rules` covering their
  code surface. Every scaffold must have at least one enforcement rule — no
  infrastructure exemption. If an agent can't think of a rule for a scaffold,
  the scaffold isn't opinionated enough to be in a pack. `pack check + pack test`
  enforces this: if the manifest declares code content without enforcement
  rules, validation fails. Enforcement packs must not declare code content.
  The archetype field in the manifest must match the actual content
  declarations. [Source: BUNDLE-001 DD-46]

- **DD-8:** Scaffold tiers determine validation expectations. Complete
  scaffolds: render with sample_config, run `test_command`, tests must pass.
  Skeleton scaffolds: structural validation only — `pack check + pack test` checks
  scaffold directory exists, expected files present, test function names
  present. Tests are NOT run. `test_command` is only used for complete
  scaffolds. Skeleton test bodies are empty or comment-only — they declare
  WHAT to test, not HOW it should behave. The consumer's own `backstop gate`
  handles behavioral validation after implementation. Each scaffold's `tier`
  field tells `pack check + pack test` which verification level to apply.
  [Source: BUNDLE-001 DD-47]

- **DD-9:** All archetypes require mechanical proof via fixtures. No pack
  is loadable without proving it does what it claims at its tier's expected
  completeness level. Enforcement packs: 100% fixture coverage for all
  rules. Code packs (SDK): `provides` surface declared, rules for SDK
  usage have fixtures. Code packs (complete scaffold): rendered tests pass.
  Code packs (skeleton scaffold): structural validity proven. "Trust me,
  it works" is not a valid validation state.
  [Source: BUNDLE-001 DD-48]

- **DD-10:** Security-class rules carry stricter validation requirements.
  Every rule declares a `risk_class`. Security-class rules require: (a)
  mandatory bypass-attempt negative fixtures demonstrating the rule catches
  circumvention attempts, (b) independent fixture coverage per claim — no
  shared fixtures across security claims, (c) in future: mandatory author
  signature (deferred until signing story is resolved). `pack check + pack test`
  enforces (a) and (b); (c) is a future DD.
  [Source: BUNDLE-001 DD-14]

- **DD-11:** Validation output is machine-parseable JSON by default, per
  ADR-0001. Every error includes: the failing phase, the specific check,
  the offending item (rule ID, claim ID, file path), a human-readable
  message, a fix hint the agent can act on, and the manifest path where
  the problem originates. The agent is the primary consumer; human-readable
  output is a secondary format flag (`--format=text`). Rationale: agent-
  first discipline means the hot path (produce-validate-fix loop) optimizes
  for machine consumption.
  [Source: ADR-0001]

- **DD-12:** Validation pipeline runs phases in dependency order with early
  termination. Phase N+1 does not run if phase N fails. This prevents
  cascading errors (a missing fixture file would cause structural, coherence,
  AND fixture execution errors if all phases ran). The agent fixes the
  earliest failing phase first, then re-runs to discover later-phase issues.
  Phase order: structural > coherence > fixtures > archetype > layer >
  risk_class.
  [Source: New — pipeline design for agent feedback loop]

- **DD-13:** Validation is idempotent and side-effect-free. Running `pack
  check` or `pack test` twice on the same pack produces identical output. No files are
  modified, no state is persisted, no network calls are made. The pack
  directory is read-only input. This enables the tight agent loop: validate,
  fix, re-validate, with confidence that validation itself didn't change
  anything.
  [Source: New — agent feedback loop requirements]

- **DD-14:** `pack check + pack test` enforces tool_config traceability. Every entry
  in `tool_config:` must be traceable to at least one rule in the pack that
  depends on that tool/setting. Orphan tool config (turning on a tool with
  no corresponding rule) is a validation failure. This is checked in the
  coherence phase (phase 2). [Source: OQ-2 resolution, BUNDLE-004 DD-24]

- **DD-15:** Three fixture execution paths in phase 3. Phase 3 of the
  validation pipeline dispatches fixtures to three engines: (1) `semgrep
  --test` for layer 1-2 semgrep rules, (2) language-native tool execution
  against fixtures for tool_config-dependent rules, (3) custom validator
  execution in process-isolated sandbox for layer 3. Each path checks:
  positive fixtures must pass, all negative fixtures must fail.
  [Source: OQ-1 and OQ-2 resolutions]

- **DD-16:** Both positive and negative fixtures are lists; all must behave
  correctly. Each claim has one or more positive fixtures and one or more
  negative fixtures (at least one of each required). `pack check + pack test` phase
  3 runs every positive fixture and requires none to trigger the rule, and
  runs every negative fixture and requires all of them to trigger the rule.
  A negative fixture that passes is a hard error — the rule doesn't catch
  the violation mode it claims to catch. A positive fixture that fails is
  also a hard error — the rule has a false positive.
  [Source: OQ-1 resolution, BUNDLE-004 DD-25]

- **DD-17:** `pack check + pack test` mechanically classifies layer 3 validators by
  category. For `category: presence` and `category: structural`, validation
  auto-accepts — it checks that the category field is present and valid but
  does not require a `justification:` field. For `category: other`, validation
  requires a non-empty `justification:` field. This check runs in phase 5
  (layer enforcement). The mechanical classification eliminates the "is a
  justification good enough?" question for the two most common layer 3 use
  cases, keeping the validation loop tight for authors. LLM review of `other`
  justifications happens at catalog submission time, not during `pack check + pack test`.
  [Source: DD-5 refined, BUNDLE-004 DD-29]

- **DD-18:** `pack check + pack test` verifies pack rule ID matches semgrep rule ID
  by reading the `rule:` file and comparing the `id:` field. Mismatch is a
  Phase 3 hard error. This eliminates a class of bugs where the manifest
  references one rule but the semgrep file contains a different one. The
  check is simple: parse the semgrep YAML, extract `rules[0].id`, compare
  to the `id:` in `pack.yml`. [Source: BUNDLE-004 DD-31]

- **DD-19:** Layer 3 validators are invoked as `validator.sh <fixture-path>`.
  For single-file `input_scope`: fixture-path is a file. For multi-file
  `input_scope`: fixture-path is a directory. Exit 0 = pass (no violation
  found). Exit non-zero = fail (violation found). Stdout may contain a
  violation message. For directory fixtures, the fixture directory must
  contain the complete expected structure. `pack check + pack test` creates no
  intermediate structure — the fixture is the complete input.
  [Source: BUNDLE-004 DD-32]

- **DD-20:** Skeleton scaffolds are structurally validated only. `pack
  check` checks: scaffold directory exists, expected files present, test
  function names present. Tests are NOT run. `test_command` is only used for
  complete scaffolds. This distinction matters because skeleton test bodies
  are intentionally empty — they declare WHAT to test, not HOW it should
  behave. Running empty tests would always pass trivially and prove nothing.
  [Source: BUNDLE-004 DD-17]

- **DD-21:** For tool_config rules (language-native tools like golangci-lint,
  ruff), fixture files cannot be validated in isolation — tools like
  golangci-lint require a buildable project (Go module with go.mod, proper
  package declaration). `pack check + pack test` no longer creates a `go.mod` from
  scratch for tool_config fixtures — the pack's own `go.mod` is used
  (see BUNDLE-004 DD-46). Phase 3 copies the pack's `go.mod` into the temp
  module environment (or runs fixtures directly in the pack directory).
  Before running any fixture execution (semgrep, tool_config, or layer 3),
  `pack check + pack test` runs `go mod tidy` for Go packs in the pack
  directory to ensure deps are resolved. This is automatic and silent unless
  it fails, in which case the error is surfaced as a Phase 3 pre-check
  failure. Fixture files must be valid Go source files with proper `package`
  declarations but do NOT need their own `go.mod`.
  [Source: round 2 P1, revised per BUNDLE-004 DD-46]

- **DD-22:** Bypass-attempt negative fixtures for security-class rules are
  mechanically identified via `bypass_attempt: true` on negative fixture
  entries in the manifest. `pack check + pack test` Phase 6 checks that every security-
  class rule has at least one negative fixture with `bypass_attempt: true` per
  claim. The negative fixture list schema accepts both plain path strings and
  objects with `path:` and optional `bypass_attempt:` boolean — `pack check + pack test`
  normalizes before checking. Non-security rules can use either form but
  `bypass_attempt:` is only validated for security-class rules.
  [Source: round 2 P1, BUNDLE-004 DD-40]

- **DD-23:** `pairs_with.rules` entries in scaffold declarations must resolve
  to actual rule IDs in the pack's `content.ruleset.rules` or `tool_config`
  entries. `pack check + pack test` Phase 2 checks this and emits a coherence warning
  (not hard error) for dangling references — the referenced rule might be in
  a different pack the consumer also loads. This balances correctness with
  cross-pack composition flexibility.
  [Source: round 2 P1, BUNDLE-004 DD-20]

## Open Questions

- **OQ-1: Fixture format — semgrep native vs backstop wrapper. RESOLVED.** Lean
  entirely on semgrep `--test` format (familiar tooling, lower barrier) or
  wrap it in a backstop format (stricter claim mapping, accommodates non-
  semgrep engines like layer 3 validators)? Directly affects what `pack
  test` phase 3 checks and how it runs fixture execution. Options:
  (a) pure semgrep `--test` format for layer 1-2, separate format for
  layer 3;
  (b) unified backstop wrapper format for all layers;
  (c) semgrep format with backstop metadata annotations
  (comments/frontmatter).
  Lean: (c) — keeps semgrep tooling working, adds claim mapping via
  annotations. Validation checks that annotations are present and consistent
  with manifest claims.
  [Source: BUNDLE-001 OQ-9]
  **Resolution:** Same as BUNDLE-004 OQ-2. Fixtures are plain files in
  engine-native format. Mapping in `pack.yml`. Both positive and negative
  fixtures are lists (at least one of each required per claim). `pack
  test` dispatches to correct engine per fixture. No backstop-specific
  annotation format — engine-native annotations expected (e.g., semgrep
  `// ruleid:` and `// ok:` comments). See DD-15, DD-16.

- **OQ-2: Non-semgrep rules — what engines are valid? RESOLVED.** AST checks,
  contract signature checks, test substantiveness patterns, custom Go-side
  checkers. If packs support multiple engines, `pack check + pack test` must know
  how to execute fixtures for each engine. If layer 3 custom validators are
  the answer for everything non-semgrep, validation only needs semgrep
  --test + validator execution. Options:
  (a) yes, multiple engines in packs with per-engine fixture execution;
  (b) layer 3 covers everything non-semgrep (custom validators);
  (c) only semgrep in packs, everything else is CLI-internal.
  Lean: (b) — layer 3 custom validators are the answer. `pack check + pack test`
  needs exactly two fixture execution paths: semgrep --test for layers 1-2,
  and direct validator execution for layer 3.
  [Source: BUNDLE-001 OQ-11]
  **Resolution:** Same as BUNDLE-004 OQ-3. Three execution paths: semgrep
  `--test` for layers 1-2, language-native tool execution for tool_config
  rules, custom validator execution in sandbox for layer 3. `pack check + pack test`
  needs three fixture execution paths, not two. See DD-15.

- **OQ-3: Layer 3 acceptance heuristics — how does validation decide? RESOLVED.**
  The principle "try layers 1-2 first" is clear, but how does `pack
  check` mechanically accept/reject a layer 3 submission? It cannot
  determine whether a check is expressible in semgrep (undecidable in
  general). Options:
  (a) self-declaration only (justification field, no verification);
  (b) self-declaration + LLM reviewer cross-check;
  (c) mandatory justification field + category validation (must reference
  a recognized layer-3 category from DD-5) + LLM review at catalog
  submission;
  (d) allow anything in layer 3 but flag for higher scrutiny.
  Lean: (c) for `pack check + pack test`: require justification + category. The LLM
  cross-check happens at catalog submission, not at authoring-time
  validation.
  [Source: BUNDLE-001 OQ-31]
  **Resolution:** Layer 3 validators declare a `category:` field with three
  values: `presence` (checking that something exists or is missing),
  `structural` (checking filesystem conventions), and `other` (everything
  else). Categories `presence` and `structural` are auto-accepted by `pack
  check` — no `justification:` field required because semgrep mechanically
  cannot detect absence or filesystem structure. Category `other` requires a
  mandatory `justification:` field explaining why layers 1-2 can't handle it.
  LLM review of `other` justifications happens at catalog submission time,
  not during `pack check + pack test`. See DD-5, DD-17.

- **OQ-4: Scaffold rendering and validation mechanics. RESOLVED.** Complete scaffolds
  must "pass out of the box with config-only changes." How does `pack
  test` render and test them? Questions: Where do sample config values
  live? How does validation handle scaffolds needing external services?
  How heavy is this step? Options:
  (a) sample config in manifest's scaffold entry + test doubles baked into
  scaffold;
  (b) separate fixture config file per scaffold + Docker-compose fixtures;
  (c) sample config in manifest, scaffolds must be self-contained (no
  external deps in tests).
  Lean: (a) — `sample_config` in manifest, scaffold tests use test doubles.
  `pack check + pack test` renders the scaffold into a temp directory, substitutes
  sample_config values, and runs the scaffold's test command. Scaffolds
  that require external services must ship their own test doubles.
  [Source: BUNDLE-001 OQ-32]
  **Resolution:** Same as BUNDLE-004 OQ-5. `sample_config` in manifest,
  test doubles in scaffold, render to temp dir, run tests. Validation
  cannot depend on external service availability.

## Spec Seeds

- **Pack Validate Command** — The `pack check + pack test` CLI command: argument
  parsing, phase orchestration, early termination logic, exit codes,
  integration with the existing CLI command tree. Covers phases 1
  (structural), 2 (coherence), 4 (archetype), 5 (layer), and 6
  (risk_class) — the checks that are manifest-only and don't require
  external tool execution. The artifact that makes `backstop pack check + pack test
  <path>` work.

- **Fixture Execution Framework** — Phase 3 of the pipeline: running
  semgrep --test for layer 1-2 rules, executing layer 3 validators in a
  sandbox, rendering and testing scaffolds, verifying SDK surface
  declarations. The artifact that connects pack validation to external tools
  (semgrep, language toolchains, sandbox runtime). Depends on OQ-1 (fixture
  format) and OQ-2 (valid engines) being resolved.

- **Validation Output Format** — The JSON output schema for `pack check + pack test`
  results: phase statuses, error objects (check, item, message, fix_hint,
  manifest_path), warning objects, summary statistics. The `--format=text`
  human-readable renderer. The contract that makes validation output
  agent-first per ADR-0001. Includes the error catalog — every possible
  validation error with its code, message template, and fix hint template.

## References

- **BUNDLE-001** — parent vision bundle for the full pack lifecycle (54 DDs,
  36 OQs). This bundle extracts and focuses the validation-relevant subset.
- **BUNDLE-004** — sibling bundle for pack manifest and authoring (46 DDs,
  6 OQs resolved). Defines what a valid pack looks like; this bundle defines
  how to verify it.
- **ADR-0001** — agent-first discipline framework. Validation output must be
  machine-parseable and optimized for agent consumption.
- **SPEC-012** — existing Go standards pack. The first pack to validate
  under this model — if the validation pipeline can't validate SPEC-012,
  it's broken.

## Version History

- **1.1.0** (2026-04-08): Advanced maturity from `defined` to `ready`. Added
  success criteria (6 items covering both commands, three fixture execution
  paths, JSON output schema, early termination, and end-to-end validation of
  prototype packs). Added solution assumptions (7 items covering manifest
  schema dependency, tool availability, directory layout, agent-first output,
  sandboxing, and validation targets). Fixed date metadata inconsistency
  (created date corrected from 2026-04-11 to 2026-04-08). Replaced remaining
  "pack validate" references in prose with "pack check" or "pack test" as
  appropriate per DD-1/DD-12.

- **1.0.0** (2026-04-08): Fixed review findings. Added REQ-034 (layer 3
  sandbox enforcement, traced to DD-6). Fixed REQ-029 idempotency claim to
  exclude timing fields (duration_ms) from the guarantee. Specified language
  enum in REQ-003 (initial set: go, reject unrecognized values). Committed
  REQ-011 to semgrep --test (removed "or equivalent"). Committed REQ-014 to
  go mod tidy for Go packs (removed "or language equivalent", deferred other
  languages to when they are added). 34 requirements total.

- **0.9.0** (2026-04-08): Advanced maturity from `exploring` to `defined`.
  Added 33 formal requirements (REQ-001 through REQ-033) in frontmatter
  requirements block. Requirements cover all six pipeline phases, both
  commands (pack check, pack test), early termination, idempotency, JSON
  output format, fix hints, negative fixture engine-limitation guidance,
  and error ordering. Each requirement traces to one or more DDs. Added
  bundle.updated field required by defined maturity gate.

- **0.7.0** (2026-04-08): Revised DD-21 — pack check + pack test uses pack's own
  go.mod for fixture deps, runs go mod tidy automatically before fixture
  execution. [Source: BUNDLE-004 DD-46]

- **0.6.0** (2026-04-08): Added bidirectional co-occurrence check to
  Phase 4 — code pack rules must pair with scaffolds/SDK via `pairs_with`.
  A rule without code content pairing in a code pack is a Phase 4 hard
  error. [Source: BUNDLE-004 DD-44]

- **0.5.0** (2026-04-08): Round 2 P1 fixes. Rule ID uniqueness now spans
  both `content.ruleset.rules` and `tool_config` entries (Phase 2 revised).
  tool_config fixture execution creates temp module environment for buildable
  context (DD-21). `pairs_with.rules` reference validation added to Phase 2
  as coherence warning (DD-23). Standalone `tool_config` entries with own
  `id:` must declare `risk_class:` — missing is a structural error (Phase 1
  revised). Bypass-attempt negative fixtures mechanically identified via
  `bypass_attempt: true` field on negative fixture entries (DD-22, Phase 6
  revised). Negative fixture schema accepts both plain strings and objects
  with `path:` key. Three new DDs (DD-21, DD-22, DD-23).

- **0.4.0** (2026-04-08): Updated validation pipeline to match BUNDLE-004
  v0.5.0 changes. Distinct `risk_class:`/`category:` fields — `risk_class:`
  checked on all rules (Phase 1/5), `category:` only on layer 3 (Phase 5).
  Semgrep rule ID matching added to Phase 3 (DD-18). Layer 3 invocation
  contract defined (DD-19). Strict co-occurrence — every scaffold needs a
  rule, no infrastructure exemption (Phase 4, DD-7 revised). Skeleton
  scaffolds structural-only validation — tests NOT run (DD-20, DD-8 revised).
  Both positive and negative fixtures as lists (DD-16 revised). `tool_config`
  with own rule ID must have claims+fixtures (Phase 2). `tool_config.file`
  excluded from file-existence checks (Phase 1). `content.fixtures.directory`
  removed (Phase 1). Phase 3 updated with `test_command` for complete
  scaffolds. Three new DDs (DD-18, DD-19, DD-20). Pipeline phases updated.
  Added rule-ID-mismatch error example. Updated all JSON examples to use
  lowercase kebab-case IDs.

- **0.3.0** (2026-04-08): Refined layer 3 acceptance in validation pipeline.
  DD-5 revised, DD-17 added. `pack check + pack test` auto-accepts `presence` and
  `structural` categories; only `other` requires justification. Updated
  Phase 5 description and OQ-3 resolution.

- **0.2.0** (2026-04-08): All 4 OQs resolved. Added DD-14 (tool_config
  traceability), DD-15 (three fixture execution paths), DD-16 (negative
  fixture list). Updated validation pipeline phase 3 and failure examples.
  Aligned with BUNDLE-004 0.2.0 resolutions.

- **0.1.0** (2026-04-08): Initial bundle at `exploring`. Extracted 10 DDs
  from BUNDLE-001 (DD-3, DD-4, DD-7, DD-14, DD-43, DD-44, DD-45, DD-46,
  DD-47, DD-48), renumbered as DD-1 through DD-10. Added 3 new DDs
  (DD-11 validation output format, DD-12 pipeline phase ordering, DD-13
  idempotency). Extracted 4 OQs from BUNDLE-001 (OQ-9, OQ-11, OQ-31,
  OQ-32), renumbered as OQ-1 through OQ-4. Defined the six-phase validation
  pipeline (structural > coherence > fixtures > archetype > layer >
  risk_class). Drafted concrete validation output examples for both passing
  and failing packs. Identified 3 spec seeds: pack check + pack test command, fixture
  execution framework, validation output format.
