---
name: spec035-phase4-gap
description: SPEC-035 impl review found Phase 4 (cmd/backstop dispatch) tasks largely skipped despite green tests + 91% coverage — false green
metadata:
  type: project
---

SPEC-035 (pack-declared engines + trusted-tool allowlist), branch
impl/spec-035-pack-declared-engines, FAILED backstop review on 2026-06-22.

The pattern: Phases 1,2,3,5,6 (engine leaf-package, pkg/check rename, manifest
parsing, validateEngine gate, OQ-1 migration) were implemented well and
substantively tested. Phase 4 (cmd/backstop dispatch — TASK-023..030) was the
hole: the DISPATCH-time allowlist gate landed (runFindingsEngine ->
checkEngineToolAllowed, genuinely before RunStdout, proven by recording runner),
but everything ELSE in Phase 4 was skipped:
  - REQ-004 pattern-arg: enum/parse/Rule.Pattern exist, but the gatherEngineInputs
    InputModePatternArg CASE does not exist (falls to default -> wrong error).
    3 mandated tests missing; fixture orphaned.
  - REQ-003 CLM-030: provisionEngines has NO allowlist gate (skips Provision!=nil).
  - REQ-006 a/b: isNativeSarifLintEngine/isNativeGoTestEngine STILL sniff
    "golangci-lint"/"go test" prefixes; StrictSarif/PackageScoped flags exist on
    the struct but are unused; CLM-026 grep FAILS.
  - REQ-008 CLM-033/034/035: no convert/validator-posture or platform-surfacing
    impl or tests.
9 of 37 mandated tests missing; all cluster in these skipped areas.

Why: 91.4% coverage + all-green masked it — absent tests/impl don't fail or drag
coverage. Classic [[feedback_integration_gap]] shape but worse: whole sub-phase
skipped. REQ-006c (ExpectedLayout derivation) and REQ-003 field-contract folding
WERE done correctly in pkg/pack, so the gap is cmd/backstop-specific.

Note: contract_signature gate failures for GateType/FieldContract/EngineBinding
are KNOWN-brittle go/parser file-scoping (BUNDLE-009 target) — symbols verified to
genuinely exist and be correct; NOT real defects.
