---
title: "Traceability Fail-Loud on Undeclared / Capability-Absent Dimensions"
number: SPEC-036
created: "2026-06-22"
status: draft
schema_version: spec/v1
spec_version: 1.1.0

implementation:
  summary: >
    Close the traceability half of the vacuous-green hole on the EXISTING binary,
    with NO analyzer deletion and NO engine. Today the three traceability gate steps
    — test substantiveness (StepTestSubstantiveness), coverage threshold
    (StepCoverageThreshold), and contract signature (StepContractSignature) — are
    hardwired Go analyzers that silently no-op on non-Go projects: substantiveness
    parses with go/parser and finds nothing, coverage finds "no Go package coverage
    targets in scope", and contract verification skips non-.go files. All three return
    a clean PASS while enforcing nothing — the exact failure the code-check half
    eliminated. This spec introduces a single language-agnostic POLARITY layer in front
    of each traceability step that classifies a project's relationship to each
    traceability DIMENSION (substantiveness, coverage, contracts) into exactly three
    classes and assigns each a fixed exit/report behavior, per the bundle's OQ-1 fork
    resolution. A dimension is DECLARED iff the project's backstop.yml carries an
    `enforcement.toolchain` entry whose `gate_type` names that dimension. Class (1)
    BROKEN-DECLARED — a declared dimension whose command errors, emits unparseable
    output, or names an unknown toolchain key — fails LOUD AND BLOCKS (config error,
    exit 2). Class (2) CAPABILITY-ABSENT — the dimension is NOT declared (and no
    capability is wired for the project's stack) — emits a conspicuous, specific
    warn-with-how-to-adopt on the REPORT SURFACE and PASSES (exit 0), forever, never
    auto-promoting to blocking. Class (3) DECLARED-INTENT-UNMET — the dimension IS
    declared but the capability it needs is missing — is a BROKEN PROMISE and BLOCKS
    (config error, exit 2). Loudness lives on the report surface (a new `warning` step
    status rendered conspicuously by FormatHuman/FormatJSON), NOT on the exit code,
    because exit 0 is invisible in CI. Every class-1/2/3 message is fail-loud AND
    useful: it names the dimension, the project's stack/language, the exact pack or
    command that is missing or broken, the declare-or-waive next step, and (for broken
    declarations) expected-vs-got — never a bare exit code. An explicit per-dimension
    WAIVE in the declaration surface silences the class-2 advisory for projects that
    have decided not to adopt a dimension. The traceability analyzers themselves
    (step_testverify.go, step_contract.go, step_coverage.go) are NOT touched by this
    spec — their eradication and pack re-implementation are Seeds 3/4 of BUNDLE-009.
    This spec only adds the classification + report-surface layer that wraps them.
  package: pkg/gate

verification:
  level: unit
  test_command: go test ./pkg/gate/ ./pkg/config/ ./cmd/backstop/ -race -coverprofile=cover.out
  coverage_threshold: 90

requirements:
  - id: REQ-001
    text: >
      The gate MUST classify each project's relationship to each of the three
      traceability dimensions — substantiveness, coverage, contracts — into exactly
      ONE of three mutually-exclusive classes, and no fourth class may exist: (1)
      BROKEN-DECLARED, (2) CAPABILITY-ABSENT, (3) DECLARED-INTENT-UNMET. A dimension
      is DECLARED iff the project's backstop.yml `enforcement.toolchain` map contains
      an entry whose `gate_type` equals that dimension's name; otherwise it is
      UNDECLARED. The classifier MUST be language-agnostic: it reads the declaration
      surface and the capability availability for the project's stack, and contains no
      baked language-specific or tool-specific branch. A dimension that is both
      declared AND has its capability present and working is NOT one of the three
      fail-loud classes and proceeds to its normal traceability step unchanged.
    supports: stack-aware-traceability:REQ-001
    follows: STD-GO-001:GO-010
  - id: REQ-002
    text: >
      A BROKEN-DECLARED dimension (class 1) — a declared dimension whose declared
      command errors at invocation, whose output is unparseable by its declared
      format, or whose `enforcement.toolchain` entry names an unknown toolchain key —
      MUST fail LOUD AND BLOCK: the wrapping step result MUST set ConfigErr = true so
      the gate halts and returns exit code 2 (the vacuous-green enemy). It MUST NOT
      return exit 0 and MUST NOT be downgraded to a warning. The blocking applies
      regardless of the project's language.
    supports: stack-aware-traceability:REQ-001
    follows: STD-GO-001:GO-010
  - id: REQ-003
    text: >
      A CAPABILITY-ABSENT dimension (class 2) — a dimension the project has NOT
      declared (and for which no built-in capability is wired for the project's stack)
      — MUST emit a conspicuous, specific warn-with-how-to-adopt advisory on the REPORT
      SURFACE and PASS (exit 0). It MUST NOT set ConfigErr, MUST NOT produce a step
      status of "fail", and MUST NOT contribute to a non-zero exit code. This behavior
      is PERMANENT: a capability-absent dimension never auto-promotes to blocking and
      there is no `required:`/`block:` knob that escalates it. Blocking on absence is
      explicitly PROHIBITED because it makes adoption hostile.

      On the EXISTING binary (Seed 1, no engine, no pack) the gate MUST derive the
      `CapabilityState` for each traceability dimension WITHOUT stubbing, from the
      project's declared language (`cfg.Language`, the backstop.yml `language:` field
      already read in the gate path) joined against baked-Go-analyzer presence: the
      ONLY traceability capability that exists on the existing binary is the baked Go
      analyzers (step_testverify / step_coverage / step_contract), which are
      Go-specific. Therefore the capability is PRESENT/WORKING for a dimension ONLY
      when `cfg.Language == "go"` AND the baked Go analyzer for that dimension exists
      (and, for Working, is not broken). When `cfg.Language != "go"` the baked Go
      analyzer does not apply and no pack provides the dimension, so the capability is
      ABSENT and an UNDECLARED dimension on a non-Go project MUST classify as class 2
      (capability-absent → warn-with-guidance, exit 0) — NOT a silent pass and NOT a
      mis-applied Go analyzer. This is the runtime-unblock case: `backstop gate` on the
      TypeScript runtime (`language: typescript`) yields a class-2 advisory per
      traceability dimension ("substantiveness needs a TypeScript pack you haven't
      pulled — declare or waive"). This derivation reads `cfg.Language` + baked-analyzer
      presence ONLY; it MUST NOT introduce a pack or engine (REQ-008).
    supports: stack-aware-traceability:REQ-001
    follows: STD-GO-001:GO-010
  - id: REQ-004
    text: >
      A DECLARED-INTENT-UNMET dimension (class 3) — a dimension the project HAS
      declared but whose required capability (pack, engine, or command) is missing —
      MUST be treated as a BROKEN PROMISE: it fails LOUD AND BLOCKS by setting
      ConfigErr = true (exit 2), exactly like class 1. It MUST NOT be downgraded to a
      class-2 warn-and-pass; once a project has declared a dimension, a missing
      capability for it is a defect, not un-adopted capability.
    supports: stack-aware-traceability:REQ-001
    follows: STD-GO-001:GO-010
  - id: REQ-005
    text: >
      Loudness MUST live on the REPORT SURFACE, not the exit code. The class-2
      advisory MUST be carried by a step result that the human output formatter
      (FormatHuman) renders CONSPICUOUSLY — visually distinct from a silent pass — and
      that the JSON output formatter (FormatJSON) emits as a machine-readable
      warning-tagged entry. A class-2 advisory MUST NOT be representable only by exit
      code, because exit 0 is invisible in CI. A passing gate that contains one or more
      class-2 advisories MUST surface them in both the human and JSON output.
      The at-a-glance human SUMMARY line MUST itself reflect warnings: `GateResult`
      MUST carry a `StepsWarned` counter incremented for every `warning`-status step,
      `NewGateResultWithScope` MUST populate it, and `FormatHuman`'s summary line MUST
      render the warned count alongside passed/failed/skipped — so a class-2 advisory
      cannot vanish from the summary surface a reviewer reads on a green run (a warning
      visible only as a per-step `formatStatus` marker but absent from the summary
      counts is non-compliant).
    supports: stack-aware-traceability:REQ-001
    follows: STD-GO-001:GO-010
  - id: REQ-006
    text: >
      Every fail-loud message — for all three classes — MUST be fail-loud AND USEFUL:
      it MUST name (a) which DIMENSION (substantiveness / coverage / contracts), (b)
      the project's STACK / language, (c) the exact PACK or COMMAND that is missing or
      broken, and (d) the next-step guidance (declare-or-waive for class 2;
      declare-the-capability or fix-the-command for classes 1 and 3). For a
      BROKEN-DECLARED command/format error (class 1) the message MUST additionally
      carry EXPECTED-VS-GOT detail (the declared command/format and the observed
      failure). No traceability fail-loud path may surface a bare exit code or an
      unannotated "failed" with no cause and no fix.
    supports: stack-aware-traceability:REQ-001
    follows: STD-GO-001:GO-010
  - id: REQ-007
    text: >
      An explicit per-dimension WAIVE MUST be available to silence the class-2
      advisory. When a project's `enforcement.toolchain` declaration marks a dimension
      as waived, that dimension's class-2 capability-absent advisory MUST be suppressed
      from the report surface and the gate MUST still pass (exit 0). A waive applies
      ONLY to the class-2 advisory: it MUST NOT silence a class-1 BROKEN-DECLARED nor a
      class-3 DECLARED-INTENT-UNMET failure (those continue to block at exit 2), because
      a waive is an opt-out of un-adopted capability, never a license to hide a defect
      or a broken promise.
    supports: stack-aware-traceability:REQ-001
    follows: STD-GO-001:GO-010
  - id: REQ-008
    text: >
      This spec MUST NOT touch the traceability ANALYZERS. The Go go/parser / go/ast
      logic in step_testverify.go, step_contract.go, and step_coverage.go MUST remain
      byte-for-byte unchanged by this spec; the classification + report-surface layer
      wraps those steps and only intercepts at the declaration/capability boundary. No
      new engine (grep / ast-grep) and no pack authoring is in scope — this seed ships
      entirely on the existing binary. Their deletion and pack re-implementation are
      explicitly deferred to BUNDLE-009 Seeds 3 and 4.
    supports: stack-aware-traceability:REQ-001
    follows: STD-GO-001:GO-010

claims:
  - id: CLM-001
    requirement: REQ-001
    text: >
      A backstop.yml whose enforcement.toolchain declares a substantiveness gate_type
      classifies the substantiveness dimension as DECLARED.
    tests:
      - TestClassify_SubstantivenessDeclared_WhenToolchainHasGateType
  - id: CLM-002
    requirement: REQ-001
    text: >
      A backstop.yml with no enforcement.toolchain entry for coverage classifies the
      coverage dimension as UNDECLARED.
    tests:
      - TestClassify_CoverageUndeclared_WhenNoToolchainGateType
  - id: CLM-003
    requirement: REQ-001
    text: >
      A dimension that is declared AND has a working capability is classified as
      NEITHER of the three fail-loud classes and routes to its normal step.
    tests:
      - TestClassify_DeclaredAndWorking_IsNotAFailLoudClass
  - id: CLM-004
    requirement: REQ-001
    text: >
      Each dimension resolves to exactly one of the three classes (or the
      none/proceed outcome); the classifier never returns two classes for one
      dimension.
    tests:
      - TestClassify_MutuallyExclusive_ExactlyOneClassPerDimension
  - id: CLM-005
    requirement: REQ-001
    text: >
      The classifier reaches its verdict from the declaration surface and capability
      availability alone, with no language-specific branch — the same undeclared
      input yields class-2 for a Go project and a TypeScript project alike.
    tests:
      - TestClassify_LanguageAgnostic_SameVerdictAcrossStacks
  - id: CLM-006
    requirement: REQ-002
    text: >
      A declared dimension whose command exits non-zero is classified BROKEN-DECLARED
      and the wrapping step sets ConfigErr = true (exit 2).
    tests:
      - TestBrokenDeclared_CommandErrors_BlocksExit2
  - id: CLM-007
    requirement: REQ-002
    text: >
      A declared dimension whose command output cannot be parsed by its declared
      format is classified BROKEN-DECLARED and blocks at exit 2.
    tests:
      - TestBrokenDeclared_UnparseableOutput_BlocksExit2
  - id: CLM-008
    requirement: REQ-002
    text: >
      A declared dimension whose enforcement.toolchain entry names an unknown
      toolchain key is classified BROKEN-DECLARED and blocks at exit 2.
    tests:
      - TestBrokenDeclared_UnknownToolchainKey_BlocksExit2
  - id: CLM-009
    requirement: REQ-002
    text: >
      A BROKEN-DECLARED dimension is never downgraded to a warning and never yields
      exit 0.
    tests:
      - TestBrokenDeclared_NeverDowngradedToWarn
  - id: CLM-010
    requirement: REQ-003
    text: >
      An undeclared, capability-absent dimension emits a report-surface advisory and
      the wrapping step passes (exit 0, ConfigErr false).
    tests:
      - TestCapabilityAbsent_Undeclared_WarnsAndPassesExit0
  - id: CLM-011
    requirement: REQ-003
    text: >
      A capability-absent dimension's step status is not "fail" and contributes
      nothing to a non-zero exit code.
    tests:
      - TestCapabilityAbsent_StatusNotFail_NoExitContribution
  - id: CLM-012
    requirement: REQ-003
    text: >
      A capability-absent dimension never auto-promotes to blocking — repeated
      classification of the same undeclared dimension stays class-2 with no escalation
      knob honored.
    tests:
      - TestCapabilityAbsent_NeverAutoPromotesToBlocking
  - id: CLM-013
    requirement: REQ-004
    text: >
      A dimension that IS declared but whose required capability is missing is
      classified DECLARED-INTENT-UNMET and blocks at exit 2 (ConfigErr true).
    tests:
      - TestDeclaredIntentUnmet_MissingCapability_BlocksExit2
  - id: CLM-014
    requirement: REQ-004
    text: >
      A DECLARED-INTENT-UNMET dimension is NOT downgraded to a class-2 warn-and-pass —
      it does not yield exit 0.
    tests:
      - TestDeclaredIntentUnmet_NotDowngradedToWarnAndPass
  - id: CLM-015
    requirement: REQ-005
    text: >
      FormatHuman renders a class-2 advisory conspicuously — visually distinct from a
      silent pass (a warning marker present in the rendered output).
    tests:
      - TestReportSurface_FormatHuman_RendersWarningConspicuously
  - id: CLM-016
    requirement: REQ-005
    text: >
      FormatJSON emits a class-2 advisory as a machine-readable warning-tagged entry
      distinguishable from a pass with no advisory.
    tests:
      - TestReportSurface_FormatJSON_EmitsWarningTaggedEntry
  - id: CLM-017
    requirement: REQ-005
    text: >
      A gate that passes (exit 0) while carrying class-2 advisories still surfaces
      those advisories in both human and JSON output — the advisory is never
      representable by exit code alone.
    tests:
      - TestReportSurface_PassingGateStillSurfacesAdvisories
  - id: CLM-018
    requirement: REQ-006
    text: >
      A class-2 advisory message names the dimension, the project's stack/language,
      the exact pack/command to adopt, and the declare-or-waive next step.
    tests:
      - TestMessage_Class2_NamesDimensionStackPackAndDeclareOrWaive
  - id: CLM-019
    requirement: REQ-006
    text: >
      A class-1 broken-declared command/format error message carries
      expected-vs-got detail (the declared command/format and the observed failure).
    tests:
      - TestMessage_Class1_CarriesExpectedVsGot
  - id: CLM-020
    requirement: REQ-006
    text: >
      A class-3 declared-intent-unmet message names the declared dimension and the
      specific missing capability plus the fix.
    tests:
      - TestMessage_Class3_NamesMissingCapabilityAndFix
  - id: CLM-021
    requirement: REQ-006
    text: >
      No traceability fail-loud path produces a bare exit code or an unannotated
      "failed" with no cause and no fix.
    tests:
      - TestMessage_NoBareExitCodeOrUnannotatedFailure
  - id: CLM-022
    requirement: REQ-007
    text: >
      A dimension marked waived in the declaration surface has its class-2 advisory
      suppressed and the gate still passes (exit 0).
    tests:
      - TestWaive_SuppressesClass2Advisory_StillPasses
  - id: CLM-023
    requirement: REQ-007
    text: >
      A waive on a dimension does NOT silence a class-1 BROKEN-DECLARED failure — that
      dimension still blocks at exit 2.
    tests:
      - TestWaive_DoesNotSilenceClass1BrokenDeclared
  - id: CLM-024
    requirement: REQ-007
    text: >
      A waive on a dimension does NOT silence a class-3 DECLARED-INTENT-UNMET failure —
      that dimension still blocks at exit 2.
    tests:
      - TestWaive_DoesNotSilenceClass3DeclaredIntentUnmet
  - id: CLM-025
    requirement: REQ-008
    text: >
      The classification layer leaves the substantiveness analyzer's verdict
      unchanged when a dimension is declared and working — a substantive Go test still
      passes and a hollow Go test still fails through step_testverify.go untouched.
    tests:
      - TestNoAnalyzerChange_Substantiveness_VerdictPreserved
  - id: CLM-026
    requirement: REQ-008
    text: >
      The classification layer leaves the contract analyzer's verdict unchanged when
      the contracts dimension is declared and working — a matching Go contract still
      passes and a mismatched one still fails through step_contract.go untouched.
    tests:
      - TestNoAnalyzerChange_Contracts_VerdictPreserved
  - id: CLM-027
    requirement: REQ-008
    text: >
      The classification layer leaves the coverage analyzer's verdict unchanged when
      the coverage dimension is declared and working — an above-threshold Go package
      still passes and a below-threshold one still fails through step_coverage.go
      untouched.
    tests:
      - TestNoAnalyzerChange_Coverage_VerdictPreserved
  - id: CLM-028
    requirement: REQ-008
    text: >
      The classifier wrapper in buildGateSteps INTERCEPTS for an assigned class:
      when a dimension classifies as class 1, 2, or 3 the wrapping step returns
      PolarityStepResult and the underlying analyzer is NOT reached, and when the
      dimension is declared-and-working the wrapper FALLS THROUGH unchanged and the
      underlying analyzer IS reached — verified by a sentinel/spy on the analyzer
      delegate so an unwired classifier (analyzer always reached) fails the test.
    tests:
      - TestWiring_ClassifierInterceptsClass123_AndFallsThroughWhenWorking
  - id: CLM-029
    requirement: REQ-003
    text: >
      On the existing binary, an UNDECLARED traceability dimension on a non-Go
      project (cfg.Language == "typescript") derives an ABSENT CapabilityState from
      cfg.Language + baked-analyzer presence and classifies as class 2
      (capability-absent → warn, exit 0) — NOT a silent pass and NOT a mis-applied
      Go analyzer; the same undeclared dimension on a Go project with the baked
      analyzer present is capability-present.
    tests:
      - TestCapabilityState_NonGoProject_DerivesAbsentClass2
  - id: CLM-030
    requirement: REQ-005
    text: >
      A gate carrying one or more class-2 warning steps reflects the warned count in
      the FormatHuman SUMMARY line (StepsWarned populated by NewGateResultWithScope
      and rendered alongside passed/failed/skipped) — not only as a per-step
      formatStatus marker — so the advisory cannot vanish from the at-a-glance
      summary on a green run.
    tests:
      - TestReportSurface_FormatHuman_SummaryReflectsWarnedCount

contracts:
  - file: pkg/gate/traceability_polarity.go
    provides:
      - name: TraceabilityDimension
        kind: type
        signature: "type TraceabilityDimension string"
      - name: DimensionSubstantiveness
        kind: constant
        signature: "DimensionSubstantiveness TraceabilityDimension = \"substantiveness\""
      - name: DimensionCoverage
        kind: constant
        signature: "DimensionCoverage TraceabilityDimension = \"coverage\""
      - name: DimensionContracts
        kind: constant
        signature: "DimensionContracts TraceabilityDimension = \"contracts\""
      - name: PolarityClass
        kind: type
        signature: "type PolarityClass int"
      - name: ClassifyDimension
        kind: function
        signature: "func ClassifyDimension(cfg *config.Config, dim TraceabilityDimension, cap CapabilityState) PolarityClass"
      - name: CapabilityState
        kind: type
        signature: "type CapabilityState struct { Present bool; Working bool; PackOrCommand string; Detail string }"
      - name: PolarityStepResult
        kind: function
        signature: "func PolarityStepResult(stepName string, dim TraceabilityDimension, class PolarityClass, cfg *config.Config, cap CapabilityState) StepResult"
    consumes:
      - source: pkg/config
        name: Config
        kind: type
      - source: pkg/config
        name: ToolchainPass
        kind: type
  - file: pkg/config/config.go
    provides:
      - name: ToolchainPass
        kind: type
        signature: "type ToolchainPass struct { Command string; Format string; Extensions []string; TestDependencyCommand string; GateType string; Waived bool }"
    consumes: []
  - file: pkg/gate/result.go
    provides:
      - name: GateResult
        kind: type
        signature: "type GateResult struct { ...existing fields...; StepsWarned int `json:\"steps_warned\"` } // GateResult gains a StepsWarned counter incremented for every warning-status step by NewGateResultWithScope; warning is non-failing for Pass and rendered in the FormatHuman summary line"
    consumes: []
  - file: cmd/backstop/gate.go
    provides:
      - name: buildGateSteps
        kind: function
        signature: "func buildGateSteps(projectRoot string, scope ...*gate.GateScope) []gate.StepFunc // wraps the three traceability step builders: derives CapabilityState from cfg.Language + baked-analyzer presence, calls ClassifyDimension, returns PolarityStepResult (intercepts) for class 1/2/3 and falls through to the unchanged analyzer when declared-and-working"
    consumes:
      - source: pkg/gate
        name: ClassifyDimension
        kind: function
      - source: pkg/gate
        name: PolarityStepResult
        kind: function
      - source: pkg/config
        name: Config
        kind: type
---

# SPEC-036: Traceability Fail-Loud on Undeclared / Capability-Absent Dimensions

## Overview

BUNDLE-009 (`stack-aware-traceability`, maturity `ready`, v0.6.0) makes the three
traceability gate steps — test substantiveness, coverage threshold, and contract
signature — stop vacuously passing on projects whose declared language is not Go.
Today all three are hardwired Go analyzers: on a TypeScript project they parse
nothing, find no coverage targets, and skip non-`.go` files, returning a clean PASS
while enforcing nothing. That is silent non-enforcement — the same vacuous-green
failure the code-check half of the kill chain already eliminated.

This spec implements **Spec Seed 1 — Fail-loud on undeclared / capability-absent
traceability dimensions**, the foundational, fully-independent slice of BUNDLE-009. It
ships entirely on the EXISTING binary: no traceability analyzer is deleted, no
structural engine (grep / ast-grep) is stood up, and no pack is authored. Those are
Seeds 3 and 4. Seed 1 adds one language-agnostic **classification + report-surface
layer** that sits in front of each traceability step and assigns a project's
relationship to each dimension to exactly one of three classes, with fixed exit/report
behavior per the bundle's OQ-1 fork resolution:

| Class | Condition | Exit | Surface |
|-------|-----------|------|---------|
| (1) BROKEN-DECLARED | declared dimension; command errors / unparseable output / unknown toolchain key | **2 (blocks)** | loud error, expected-vs-got |
| (2) CAPABILITY-ABSENT | dimension NOT declared, no capability wired for the stack | **0 (passes)** | conspicuous report-surface warn-with-how-to-adopt (waivable), forever |
| (3) DECLARED-INTENT-UNMET | dimension declared, but required capability missing | **2 (blocks)** | loud error (broken promise) |

The governing invariant is the bundle's documented **"loud ≠ blocking"**: loudness
lives on the **report surface**, not the exit code, because exit 0 is invisible in CI.
Blocking is reserved for **defects and broken promises** (classes 1 and 3); un-adopted
capability (class 2) is loud-but-passing, permanently, because blocking on absence
makes adoption hostile (violates make-the-right-thing-easy). Declaring a dimension is
the opt-in to enforcement; there is no separate `required:`/`block:` flag and no
"measure-but-don't-fail" rung.

## Requirements

Requirements are defined in the frontmatter `requirements[]` array (REQ-001 through
REQ-008). Each traces to BUNDLE-009 `REQ-001` (the fail-loud polarity requirement,
resting on the OQ-1 fork resolution and the OQ-4 exit-code reconciliation).

The three-class allowlist is exhaustive and the classes are mutually exclusive — a
dimension is exactly one of {BROKEN-DECLARED, CAPABILITY-ABSENT, DECLARED-INTENT-UNMET}
or it is none of them (declared and working, proceeding to its normal step). No fourth
fail-loud class is permitted. The blocking set is exactly {class 1, class 3}; the
passing set is exactly {class 2}. A waive applies ONLY to class 2.

## Implementation

Target package: `pkg/gate` (the classification + report-surface layer) plus a small,
additive extension to `pkg/config` (the declaration surface). The work is purely the
fail-loud wrapper; the analyzers are untouched (REQ-008).

### Declaration surface extension (pkg/config)

`ToolchainPass` (pkg/config/config.go) gains two additive, optional fields:

- `gate_type` — names the traceability dimension this toolchain entry declares
  (`substantiveness` | `coverage` | `contracts`). Its presence with a matching value
  is what makes a dimension DECLARED.
- `waived` — a boolean opting the dimension out of its class-2 advisory.

Both fields are optional and default to the zero value, so existing `backstop.yml`
files (and the backstop-yml JSON schema) remain valid; the strict-unknown-fields YAML
decode is satisfied because the fields are added to the struct and schema together.

### Classification (pkg/gate/traceability_polarity.go — new file)

A single language-agnostic function `ClassifyDimension(cfg, dim, cap)` returns a
`PolarityClass` for one dimension by reading only:

1. **Declared?** — whether `cfg.Enforcement.Toolchain` has an entry whose `gate_type`
   equals `dim`. (A declared entry may also carry an unknown toolchain key, which is a
   class-1 condition.)
2. **Capability state** — the `CapabilityState{Present, Working, PackOrCommand, Detail}`
   passed in by the gate wiring, describing whether the dimension's capability exists
   for the project's stack and whether its declared command ran/parsed cleanly.

The classification decision table (the complete allowlist):

| Declared | Capability | Command/format | → Class |
|----------|-----------|----------------|---------|
| yes | present | errors / unparseable / unknown key | (1) BROKEN-DECLARED |
| yes | missing | n/a | (3) DECLARED-INTENT-UNMET |
| yes | present | clean | none — proceed to normal step |
| no | absent | n/a | (2) CAPABILITY-ABSENT |
| no | present | n/a | none — proceed to normal step |

`ClassifyDimension` contains NO language- or tool-specific branch — it joins the
declaration surface against the supplied `CapabilityState`. Stack-specific knowledge
(which pack/command a dimension needs) is computed by the gate wiring and handed in via
`CapabilityState`, not baked into the classifier.

### Report-surface emission (pkg/gate/traceability_polarity.go + pkg/gate/output.go)

`PolarityStepResult(stepName, dim, class, cfg, cap)` converts a class into a
`StepResult`:

- **Class 1 / Class 3** → `Status: "fail"`, `ConfigErr: true` (so `Gate.Run` halts and
  `ExitCode` returns 2), with a fail-loud-and-useful `Violation` message (REQ-006).
- **Class 2** → a new `Status: "warning"` step result with `ConfigErr: false` and a
  conspicuous advisory `Violation` tagged `Severity: "warning"`. A `warning` step is
  counted as a PASS for exit purposes: `NewGateResultWithScope` MUST treat `warning` as
  non-failing (it does not set `Pass = false`) AND MUST increment a new
  `GateResult.StepsWarned` counter for it. If the dimension is `waived`, the
  advisory is suppressed and the step is a plain `pass`.

`output.go` gains a `warning` branch in `formatStatus` AND in the `FormatHuman` summary
line — the summary renders the warned count alongside passed/failed/skipped (e.g.
`Steps: N passed, M failed, K skipped, W warned`) so a class-2 advisory cannot vanish
from the at-a-glance summary on a green run. `FormatJSON` emits `StepsWarned` and
already serializes `Status` and `Violations[].Severity`, so the warning-tagged entry is
machine-readable without a schema change beyond the new status string value and counter.

### Wiring (cmd/backstop/gate.go)

`buildGateSteps` wraps each of the three traceability step builders
(`buildTestSubstantivenessStep`, `buildCoverageStep`, `buildContractStep`) so that,
before delegating to the existing analyzer step, it computes the `CapabilityState` for
the dimension and the project's stack, calls `ClassifyDimension`, and — for classes 1,
2, 3 — returns `PolarityStepResult` instead of running the analyzer (the wrapper
INTERCEPTS; the analyzer is NOT reached). Only the "declared-and-working" /
"undeclared-but-present" outcomes fall through to the existing analyzer step, which is
unchanged (REQ-008). This wrapper lives in `cmd/backstop/gate.go`, OUTSIDE the
`pkg/gate`/`pkg/config` unit scope, so the verification `test_command` adds
`./cmd/backstop/` and CLM-028 mechanically verifies the intercept-vs-fall-through wiring
with a spy on the analyzer delegate — an UNWIRED classifier (analyzer always reached)
fails that test, closing the unit-green-but-unwired integration gap.

**`CapabilityState` derivation on the existing binary (no engine, no pack).** The
wiring computes `CapabilityState` from the project's declared language (`cfg.Language`,
already read in the gate path via `gateLanguage`) joined against baked-Go-analyzer
presence — NOT a stub. The only traceability capability that exists on the existing
binary is the baked Go analyzers (step_testverify / step_coverage / step_contract),
which are Go-specific. So a dimension is `Present`/`Working` only when
`cfg.Language == "go"` AND its baked Go analyzer exists (and isn't broken — coverage's
brittle path is the class-1 broken-declared case for a declared Go coverage dimension);
the capability is `Absent` when `cfg.Language != "go"`. Consequently an undeclared
dimension on a non-Go project (e.g. `language: typescript`, the runtime-unblock case)
lands in class 2 (capability-absent → warn, exit 0), not a silent pass and not a
mis-applied Go analyzer (CLM-029). The derivation reads `cfg.Language` +
baked-analyzer presence only; it introduces no pack or engine (REQ-008). The wiring is
the only place stack-specific capability knowledge lives, and it is computed, not baked
into the classifier.

### Processing steps (enumerated for the planner)

1. Extend `ToolchainPass` (+ backstop-yml schema) with `gate_type` and `waived`.
2. Add a `declaredDimension` helper that joins `enforcement.toolchain` against a
   `TraceabilityDimension`.
3. Implement `ClassifyDimension` over the decision table above.
4. Implement `PolarityStepResult` (class → StepResult, including the waive suppression).
5. Make `NewGateResultWithScope` treat `warning` as non-failing and increment a new
   `GateResult.StepsWarned` counter for it.
6. Add the `warning` rendering branch to `FormatHuman`/`formatStatus` AND render the
   warned count in the `FormatHuman` summary line; emit `StepsWarned` in `FormatJSON`.
7. Derive `CapabilityState` from `cfg.Language` + baked-analyzer presence and wrap the
   three traceability step builders in `buildGateSteps` to classify first — intercepting
   for class 1/2/3 and falling through to the unchanged analyzer when declared-and-working.

## Verification

- **Level:** unit. The classifier, the class→StepResult mapping, the warn/pass/block
  exit semantics, the report-surface rendering (including the `StepsWarned` summary
  count), and the waive suppression are all unit-testable in `pkg/gate` and `pkg/config`
  with table-driven inputs; no live tool is required (the analyzers are not exercised by
  this layer — REQ-008 asserts they are untouched, verified by preserving their existing
  passing tests). The classifier-in-front-of-the-analyzer WIRING lives in
  `cmd/backstop/gate.go` (`buildGateSteps` and the three step builders), so the test
  scope adds `./cmd/backstop/`: CLM-028 verifies the wrapper intercepts for class 1/2/3
  (analyzer not reached) and falls through unchanged when declared-and-working (analyzer
  reached) via a spy on the analyzer delegate, and CLM-029 verifies the non-Go
  `CapabilityState` derivation. This closes the unit-green-but-unwired integration gap.
- **Coverage threshold:** 90 (unit level).
- **Test command:** `go test ./pkg/gate/ ./pkg/config/ ./cmd/backstop/ -race -coverprofile=cover.out`.

Claims are defined in the frontmatter `claims[]` array. The class matrix is covered
exhaustively: each of the three classes has positive claims (it produces the right
exit/surface), each blocking class has a "not-downgraded" negative claim, the
capability-absent class has a "never-auto-promotes" claim, and the waive path has one
positive (silences class 2) plus two negatives (does NOT silence class 1, does NOT
silence class 3). REQ-008 is covered by three "verdict-preserved" claims, one per
analyzer, plus a wiring claim that the classifier intercepts for class 1/2/3 and falls
through unchanged when declared-and-working (spy on the analyzer delegate). The
non-Go `CapabilityState` derivation (undeclared dimension on a TypeScript project →
class 2) is covered by its own claim so it cannot be stubbed, and the `StepsWarned`
summary surface is covered by a claim asserting the warned count appears in the
`FormatHuman` summary line, not just as a per-step marker.

## Sharp Edges

- **A class-2 advisory that is not conspicuous enough is a back-door vacuous green.**
  The entire point of class 2 is that it passes (exit 0); if the report surface renders
  it indistinguishably from a silent pass, the project is back to silent
  non-enforcement with extra steps. The `warning` status MUST be visually distinct in
  human output AND a distinct severity-tagged entry in JSON (REQ-005). A reviewer must
  confirm the rendering is conspicuous, not merely present.

- **Distinguishing class 1 (BROKEN-DECLARED) from class 3 (DECLARED-INTENT-UNMET) from
  class 2 (CAPABILITY-ABSENT) hinges entirely on the "declared?" boolean.** All three
  involve a missing/broken capability; the ONLY discriminator that flips class 2
  (warn-pass) into class 3 (block) is whether the project declared the dimension. If the
  `gate_type` join is wrong (e.g. matches loosely, or a typo'd toolchain key silently
  reads as "undeclared"), a broken promise silently degrades to a warn-and-pass — a
  vacuous green. The join MUST be exact and a malformed/unknown declared key is class 1,
  not class 2.

- **The waive path is a deliberate hole and must be scoped tightly.** A waive silences
  class 2 only. If a waive accidentally suppressed class 1 or class 3, a project could
  hide a genuinely broken declaration or an unmet declared intent behind `waived: true`
  — turning the adoption-friendliness mechanism into a defect-concealment mechanism.
  CLM-023 and CLM-024 pin this shut.

- **`warning` is a new step status and the exit-code path must not regress.**
  `NewGateResultWithScope` currently sets `Pass = false` only on `"fail"`. A `warning`
  status MUST be counted into the new `StepsWarned` counter (so it appears in the
  summary) but MUST NOT flip `Pass`. If `warning` were accidentally treated like `fail`,
  class 2 would block — the precise outcome the bundle prohibits. If it were silently
  dropped (unknown status, no `StepsWarned` increment), the advisory would vanish from
  the summary counts a reviewer reads on a green run — the "blends into passing" drop
  this sharp edge warns of. The `FormatHuman` summary line MUST render the warned count
  alongside passed/failed/skipped (CLM-030), not only the per-step `formatStatus`
  marker (CLM-015).

- **Additive config fields must not break the strict YAML decode or the embedded
  schema.** `LoadConfigFromPath` uses `KnownFields(true)` AND validates against the
  embedded backstop-yml JSON schema. The new `gate_type`/`waived` fields must be added
  to BOTH the struct and the schema in lockstep, or every existing `backstop.yml` that
  omits them stays valid while one that sets them fails schema validation (or vice
  versa).

- **REQ-008 (analyzers untouched) is an easy line to cross.** The temptation is to
  "improve" the Go analyzer while wiring the wrapper. Any change to the verdict logic in
  step_testverify.go / step_contract.go / step_coverage.go is out of scope for this
  seed and risks conflicting with the Seed 3/4 eradication. The wrapper intercepts
  BEFORE the analyzer; it does not modify it.

## Review Questions

- Does the human output render a class-2 advisory in a way a reviewer would NOTICE on a
  green run — a distinct marker/section — rather than a line that blends into passing
  steps? (A reviewer should eyeball the actual rendered output, not just assert the
  string is present.)
- Is the `gate_type` → dimension join exact, such that an unknown or typo'd toolchain
  key is classified BROKEN-DECLARED (class 1, blocks) rather than silently falling
  through to UNDECLARED (class 2, passes)?
- Does `NewGateResultWithScope` treat `warning` as non-failing for `Pass` AND count it
  in the summary, with a test that a warning-only gate returns exit 0 while still
  surfacing the advisory?
- Does the waive suppression key off the SAME `gate_type` join as classification, so a
  waive can only ever target a real declared/declarable dimension and can never reach a
  class-1 or class-3 result?
- Are the three analyzer steps genuinely unreached when a class is assigned (the wrapper
  returns before delegating), and genuinely reached unchanged when the dimension is
  declared-and-working?
- Do all class-1/2/3 messages name dimension + stack + exact pack/command + next step,
  and do class-1 command/format errors carry expected-vs-got, with no bare-exit-code or
  unannotated-failure path?

## References

- `bundles/BUNDLE-009-stack-aware-traceability.bundle.md` — REQ-001; OQ-1 (fork
  resolution: declared == blocking, absent == warn, no separate flag; the three-class
  model); OQ-4 (exit-code reconciliation: broken-declared → exit 2, undeclared → warn
  exit 0); OQ-5 (fail-loud stays in-scope as Seed 1, not a separate prerequisite);
  Spec Seed 1.
- `pkg/gate/gate.go` — `Gate.Run` / `ExitCode` / `ConfigErr` (exit-2 halt mechanism);
  `pkg/gate/result.go` — `StepResult.Status`, `NewGateResultWithScope` (the `Pass`
  computation the `warning` status extends).
- `pkg/gate/output.go` — `formatStatus` / `FormatHuman` / `FormatJSON` (the report
  surface that must render `warning` conspicuously).
- `pkg/config/config.go` — `Enforcement.Toolchain` (`map[string]ToolchainPass`), the
  declaration surface a dimension is declared through.
- `cmd/backstop/gate.go` — `buildGateSteps` and the three traceability step builders
  (`buildTestSubstantivenessStep`, `buildCoverageStep`, `buildContractStep`) the
  classification layer wraps.
- BUNDLE-009 invariant "loud ≠ blocking" — block defects + broken promises;
  warn-with-guidance for un-adopted capability; the report surface carries loudness
  because exit 0 is invisible in CI.

## Version History

- **1.1.0** (2026-06-22) — Corrective pass closing the spec-reviewer re-review FAIL on
  three issues. (1) **Wiring-verification gap:** the classifier-in-front-of-the-analyzer
  wiring lives in `cmd/backstop/gate.go`, outside the original `pkg/gate`/`pkg/config`
  test scope, and no claim verified the classifier was wired IN FRONT OF the analyzers
  (CLM-025/026/027 only assert analyzer verdicts are preserved — an unwired classifier
  satisfies those too). Added `./cmd/backstop/` to the verification `test_command` and
  CLM-028, which verifies the wrapper intercepts for class 1/2/3 (analyzer not reached)
  and falls through unchanged when declared-and-working (analyzer reached) via a spy on
  the analyzer delegate. (2) **`CapabilityState` derivation specified concretely:**
  REQ-003 now states the no-pack-binary derivation — capability is Present/Working only
  when `cfg.Language == "go"` AND the baked Go analyzer for the dimension exists, Absent
  when `cfg.Language != "go"` — so an undeclared dimension on a non-Go (TypeScript)
  project lands in class 2 (the runtime-unblock case), not a silent pass or mis-applied
  Go analyzer. CLM-029 pins this so the planner cannot stub it. The derivation reads
  `cfg.Language` + baked-analyzer presence only; no pack/engine (REQ-008 unchanged).
  (3) **`warning` in the summary surface:** REQ-005 now mandates a `GateResult.StepsWarned`
  counter populated by `NewGateResultWithScope` and rendered in the `FormatHuman` summary
  line; CLM-030 asserts the warned count appears in the summary surface, not only as a
  per-step `formatStatus` marker (CLM-015). No analyzer change, no engine, no pack —
  REQ-008 and the three-class matrix / exit polarities / waive matrix are unchanged.
- **1.0.0** (2026-06-22) — Initial spec authored from BUNDLE-009 (stack-aware-traceability),
  Spec Seed 1 (fail-loud on undeclared / capability-absent traceability dimensions).
