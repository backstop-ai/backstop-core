---
title: "SPEC-017: Pack Gate Integration"
number: SPEC-017
created: "2026-04-20"
status: draft
schema_version: spec/v1
spec_version: 1.0.0

implementation:
  summary: >
    Wire the pack system into the backstop gate. This is the integration
    spec that connects pack infrastructure (SPEC-013/014/015/016) to the
    gate pipeline (SPEC-010). After this spec, backstop gate loads installed
    packs, merges their rules into the code check pipeline, verifies lock
    integrity, and reports violations with namespaced pack rule IDs.
  package: cmd/backstop

verification:
  level: integration
  test_command: go test ./cmd/backstop/ ./pkg/gate/ -race -coverprofile=cover.out -v
  coverage_threshold: 80

requirements:
  - id: REQ-001
    text: >
      backstop gate must load the packs list from backstop.yml, resolve
      each pack from .backstop/packs/, and parse its manifest via
      ParseManifestFile. If backstop.yml declares no packs, the gate
      proceeds without pack enforcement. If backstop.yml declares packs
      but .backstop/packs/ is missing or empty, the gate must fail with
      a diagnostic identifying the missing packs.

  - id: REQ-002
    text: >
      backstop gate must run VerifyLock as a pre-check before any other
      gate step. VerifyLock compares content hashes of installed packs
      against backstop.lock. Hash mismatch, missing pack, or extra
      unlocked pack must cause the gate to fail with a specific
      diagnostic before any code checks run. Missing backstop.lock
      when packs are declared must also fail.

  - id: REQ-003
    text: >
      For each installed pack with layer 2 semgrep rules, the gate must
      merge those rules into the code check semgrep pass. The pack's
      rule: field points to the semgrep YAML file within the pack
      directory. The gate passes these as additional --config arguments
      to the semgrep invocation. Violations from pack rules must appear
      in gate output with namespaced rule IDs (pack-name/rule-id).

  - id: REQ-004
    text: >
      For each installed pack with tool_config entries, the gate must
      ensure the consumer's tool config files reflect the pack's
      requirements before running the lint pass. If tool_config was
      merged during pack add, the lint pass picks it up automatically
      from the consumer's config files. The gate does not re-merge
      at runtime — it verifies the config files exist and runs the
      tools.

  - id: REQ-005
    text: >
      For each installed pack with layer 3 custom validators, the gate
      must execute each validator against the project's source files
      using the invocation contract (validator.sh <path>, exit 0=pass,
      non-zero=fail) within process isolation (SandboxedRun). Validator
      violations must appear in gate output with namespaced rule IDs.

  - id: REQ-006
    text: >
      When multiple packs are installed, the gate must load and enforce
      all packs. Rule IDs are namespaced by pack name (pack-name/rule-id)
      to prevent collisions. Violations from different packs are
      independently attributed. The gate does not resolve inter-pack
      conflicts — it reports all violations from all packs.

  - id: REQ-007
    text: >
      The gate must report pack-sourced violations in the same format
      as native violations, with the addition of the pack name in the
      rule ID. JSON output includes a source_pack field on violations
      from packs. Human output prefixes the rule ID with the pack name.

  - id: REQ-008
    text: >
      If a pack's rules reference files that don't exist in the installed
      pack directory (e.g., a rule: path pointing to a missing .yml file),
      the gate must fail with a diagnostic identifying the broken pack
      and the missing file, rather than silently skipping the rule.

  - id: REQ-009
    text: >
      The pack loading and rule merging must not modify any files on disk.
      The gate reads pack manifests, semgrep rules, and validator scripts
      but never writes to .backstop/packs/, backstop.yml, backstop.lock,
      or consumer config files. Gate execution is read-only.

  - id: REQ-010
    text: >
      backstop code check must also load and enforce pack rules when
      packs are installed. The same pack loading logic used by the gate
      applies to code check. Diff-scoped code check (default mode)
      runs pack rules only against changed files.

claims:
  # REQ-001: Pack loading
  - id: CLM-001
    requirement: REQ-001
    text: Gate loads packs from backstop.yml and parses manifests from .backstop/packs/
    tests:
      - TestGateIntegration_LoadsPacksFromConfig

  - id: CLM-002
    requirement: REQ-001
    text: Gate proceeds without pack enforcement when no packs declared
    tests:
      - TestGateIntegration_NoPacks

  - id: CLM-003
    requirement: REQ-001
    text: Gate fails with diagnostic when declared packs are missing from .backstop/packs/
    tests:
      - TestGateIntegration_MissingPackDir

  # REQ-002: Lock verification
  - id: CLM-004
    requirement: REQ-002
    text: Gate runs VerifyLock before other steps and fails on hash mismatch
    tests:
      - TestGateIntegration_LockHashMismatch

  - id: CLM-005
    requirement: REQ-002
    text: Gate fails when backstop.lock is missing but packs are declared
    tests:
      - TestGateIntegration_MissingLockfile

  - id: CLM-006
    requirement: REQ-002
    text: Gate fails when installed pack is not in backstop.lock
    tests:
      - TestGateIntegration_ExtraUnlockedPack

  # REQ-003: Semgrep rule merging
  - id: CLM-007
    requirement: REQ-003
    text: Gate merges pack semgrep rules into code check and violations appear with namespaced IDs
    tests:
      - TestGateIntegration_SemgrepRulesMerged

  - id: CLM-008
    requirement: REQ-003
    text: Pack semgrep rule violations include pack-name/rule-id format in output
    tests:
      - TestGateIntegration_NamespacedRuleIDs

  # REQ-004: tool_config
  - id: CLM-009
    requirement: REQ-004
    text: Gate lint pass uses tool config that was merged during pack add
    tests:
      - TestGateIntegration_ToolConfigApplied

  # REQ-005: Layer 3 validators
  - id: CLM-010
    requirement: REQ-005
    text: Gate executes pack layer 3 validators in sandbox and reports violations
    tests:
      - TestGateIntegration_Layer3ValidatorExecuted

  - id: CLM-011
    requirement: REQ-005
    text: Layer 3 validator violations appear with namespaced rule IDs
    tests:
      - TestGateIntegration_Layer3NamespacedIDs

  # REQ-006: Multi-pack
  - id: CLM-012
    requirement: REQ-006
    text: Gate loads and enforces rules from multiple installed packs
    tests:
      - TestGateIntegration_MultiplePacksEnforced

  - id: CLM-013
    requirement: REQ-006
    text: Violations from different packs are independently attributed
    tests:
      - TestGateIntegration_MultiPackAttribution

  # REQ-007: Output format
  - id: CLM-014
    requirement: REQ-007
    text: JSON gate output includes source_pack field on pack violations
    tests:
      - TestGateIntegration_JSONSourcePackField

  - id: CLM-015
    requirement: REQ-007
    text: Human gate output shows pack-name/rule-id prefix
    tests:
      - TestGateIntegration_HumanOutputPackPrefix

  # REQ-008: Broken pack detection
  - id: CLM-016
    requirement: REQ-008
    text: Gate fails with diagnostic when pack rule file is missing
    tests:
      - TestGateIntegration_BrokenPackRuleFile

  # REQ-009: Read-only execution
  - id: CLM-017
    requirement: REQ-009
    text: Gate does not modify any files in .backstop/packs/ or consumer config
    tests:
      - TestGateIntegration_ReadOnlyExecution

  # REQ-010: Code check integration
  - id: CLM-018
    requirement: REQ-010
    text: backstop code check loads and enforces pack rules on changed files
    tests:
      - TestGateIntegration_CodeCheckLoadsPacks

  - id: CLM-019
    requirement: REQ-010
    text: backstop code check --all enforces pack rules across entire codebase
    tests:
      - TestGateIntegration_CodeCheckAllLoadsPacks

contracts:
  - file: cmd/backstop/gate.go
    provides:
      - name: loadInstalledPacks
        kind: function
        signature: "func loadInstalledPacks(projectRoot string) ([]*pack.Manifest, error)"
        notes: "Reads backstop.yml packs list, resolves from .backstop/packs/, returns parsed manifests"
      - name: mergePackRules
        kind: function
        signature: "func mergePackRules(packs []*pack.Manifest, packDir string) ([]string, error)"
        notes: "Returns additional semgrep --config paths from all pack layer 2 rules"
      - name: runPackValidators
        kind: function
        signature: "func runPackValidators(packs []*pack.Manifest, packDir, projectRoot string) ([]gate.Violation, error)"
        notes: "Executes all layer 3 validators from installed packs in sandbox"
    consumes:
      - source: pkg/pack/manifest.go
        name: ParseManifestFile
        kind: function
      - source: pkg/pack/manifest.go
        name: NamespacedRuleID
        kind: function
      - source: pkg/pack/distribution/verify.go
        name: VerifyLock
        kind: function
      - source: pkg/pack/distribution/lockfile.go
        name: ReadLockfile
        kind: function
      - source: pkg/packval/sandbox.go
        name: SandboxedRun
        kind: function
---

# SPEC-017: Pack Gate Integration

## Overview

This is the integration spec that bridges the pack system (SPEC-013/014/015/016) with the gate pipeline (SPEC-010). The pack infrastructure specs built the pieces — manifest parsing, validation pipeline, distribution lifecycle, constraint validation. This spec wires them into the gate so that installed packs actually enforce their rules on user code.

**Without this spec:** packs install, validate, and distribute correctly, but the gate ignores them. `backstop gate` runs without loading pack rules. The pack system exists but doesn't enforce.

**With this spec:** `backstop gate` loads installed packs, merges their semgrep rules into the code check pipeline, executes layer 3 validators, verifies lock integrity, and reports violations with namespaced pack rule IDs. The user installs a pack and the gate just works.

**Verification Level:** Integration (80% coverage)
**Dependencies:** SPEC-010 (gate), SPEC-013 (manifest), SPEC-014 (validation), SPEC-015 (distribution), SPEC-016 (constraints)
**Capability:** CAP-001 (Pack Gate Enforcement)

## Requirements

Requirements and claims are defined in frontmatter.

### Integration Points

| Gate Component | Pack Component | Integration |
|---|---|---|
| Gate pre-step | VerifyLock | Lock integrity check before any steps |
| Gate step 2 (code check / semgrep) | Pack layer 2 rules | Additional --config paths |
| Gate step 2 (code check / lint) | Pack tool_config | Consumer config files already merged |
| New: pack validator step | Pack layer 3 validators | SandboxedRun execution |
| Gate output | NamespacedRuleID | pack-name/rule-id format |

### Loading Flow

```
backstop gate
  → read backstop.yml packs list
  → if no packs declared: proceed without pack enforcement
  → read backstop.lock
  → VerifyLock (hash mismatch = fail before any checks)
  → for each pack:
      → ParseManifestFile from .backstop/packs/<pack>/pack.yml
      → collect layer 2 semgrep rule paths
      → collect layer 3 validator paths
  → merge semgrep rules into code check --config
  → run code check with merged rules
  → run layer 3 validators in sandbox
  → namespace all violations with pack-name/rule-id
  → report
```

## Implementation

### Pack Loading

`loadInstalledPacks` reads `backstop.yml`, extracts the packs list, and for each declared pack resolves the pack directory at `.backstop/packs/<pack-name>/`, calls `pack.ParseManifestFile` on the pack's `pack.yml`, and returns the parsed manifests. If a declared pack's directory doesn't exist, returns an error identifying the missing pack.

### Rule Merging

`mergePackRules` iterates loaded manifests and collects all layer 2 rule file paths (the `rule:` field on each rule), resolved relative to the pack's install directory. These paths are passed as additional `--config` arguments to the semgrep invocation in the code check step.

### Validator Execution

`runPackValidators` iterates loaded manifests and for each layer 3 rule, executes the validator script via `SandboxedRun` against the project's source files. Input scope (single-file vs multi-file) determines how the validator is invoked.

### Lock Verification

`VerifyLock` from `pkg/pack/distribution/verify.go` is called as a pre-step before any gate checks. It compares content hashes of installed packs against `backstop.lock`. Any mismatch, missing pack, or extra unlocked pack causes the gate to fail immediately with a specific diagnostic.

### Violation Namespacing

All violations from pack rules are namespaced using `pack.NamespacedRuleID(packName, ruleID)` before being added to the gate result. JSON output includes a `source_pack` field. Human output prefixes the rule ID.

## Verification

Verification is defined in frontmatter. Integration-level testing at 80% coverage.

Tests use temporary project directories with pre-installed packs (testdata fixtures), mocked semgrep execution where needed, and real gate pipeline calls to verify end-to-end wiring.

## Sharp Edges

1. **Pack loading performance.** Loading and parsing every installed pack on every gate run adds startup latency. For v1 this is acceptable. Future optimization: cache parsed manifests keyed by content hash.

2. **Semgrep invocation with many --config paths.** Each pack rule adds a --config argument. A project with 5 packs and 20 rules each = 100 --config paths. Verify semgrep handles this gracefully.

3. **Layer 3 sandbox on different platforms.** SandboxedRun is macOS-only (sandbox-exec). On Linux the gate must either fail closed (reject layer 3 rules) or execute without sandbox (less secure). The integration must handle both paths.

4. **tool_config was already merged at `pack add` time.** The gate does NOT re-merge tool_config at runtime — it trusts that `pack add` already wrote the config files. If a user manually reverts their `.golangci.yml`, the pack's config is lost and the gate won't catch what the pack intended to catch. This is by design but worth documenting.

5. **Interaction with diff-scoped code check.** `backstop code check` (diff-scoped) must also load pack rules. But layer 3 validators may need to see files outside the diff scope (e.g., presence checks across the whole project). The integration must decide: run layer 3 validators against the full project or only the diff.

## Review Questions

1. Should VerifyLock run as a gate pre-step (before step 1) or as part of step 1 (artifact validation)? Pre-step is cleaner but adds a new concept to the gate pipeline.

2. When a pack rule fires on a file, should the violation's `file` field show the absolute path or the path relative to the project root? Consistency with existing gate output matters.

3. If semgrep fails to load a pack's rule file (invalid YAML, incompatible semgrep version), should that fail the entire gate or just skip that pack's rules with a warning?

4. Should `backstop code check --file <path>` (single-file hook mode) also load pack rules? The 2-second execution budget may not allow loading and parsing all pack manifests.

## References

- **CAP-001** — Pack Gate Enforcement capability (acceptance criteria)
- **SPEC-010** — Gate (kill chain pipeline)
- **SPEC-013** — Pack manifest types (ParseManifestFile, NamespacedRuleID)
- **SPEC-015** — Pack distribution (VerifyLock, ReadLockfile)
- **BUNDLE-004** — DD-28 (pack rules execute as part of backstop gate)
