---
title: "Coverage Production Engine"
number: SPEC-042
created: "2026-06-24"
status: draft
schema_version: spec/v1
spec_version: 1.0.0

implementation:
  summary: >
    BUNDLE-011 Spec Seed 4 (DD-7) — the PRODUCER side of coverage, the reason
    Seed 3's (SPEC-041) consumer contract is satisfiable. SPEC-041 is a DRAFT sibling
    spec: it DECLARES (on paper) a re-implemented coverage gate step that consumes a
    language-agnostic per-file `[]CoverageRecord` — it is NOT landed on `main`. The
    LIVE coverage on `main` is the OLD baked Go analyzer `StepCoverageThresholdFunc`
    (`pkg/gate/step_coverage.go`), which runs `go test -coverprofile` and parses Go
    coverage text in-binary — exactly what BUNDLE-011 REQ-011 / Seed 3 eradicates. So
    NOTHING on `main` produces a normalized per-file coverage record: the go-toolchain
    test engine emits only test-failure SARIF (`scripts/test-to-sarif.sh`), and the
    engine model (`dispatchPackEngines`, cmd/backstop/pack_gate.go) normalizes EVERY
    engine's output to SARIF (`runFindingsEngine` → `check.ParsePackFindings`) or runs
    the sandbox engine — there is no coverage-records output channel. This spec adds it.
    THREE deliverables across three bundle requirements.
    (REQ-014 — coverage-records channel) Give the engine model a SECOND normalized
    output type, **coverage-records: a per-file measurement INPUT, distinct from
    SARIF findings**. The architectural line is load-bearing: SARIF stays the
    findings/OUTPUT lingua franca; coverage is a MEASUREMENT INPUT the gate consumes
    and turns into ordinary gate violations (via SPEC-041's coverage step) — it is
    NOT a competing report format. A declared engine whose binding fills gate-type
    `coverage` (engine.GateTypeCoverage, already in the enum) routes its convert
    output to a NEW coverage-records parser (`check.ParsePackCoverage`) instead of
    the SARIF findings parser. Coverage CANNOT be SARIF: SARIF represents failures —
    sparse, located — but coverage is a COMPLETE per-file census that MUST include
    PASSING files, and non-vacuousness requires distinguishing measured-and-passed
    from not-measured (SPEC-041 REQ-003), a distinction SARIF-as-findings structurally
    cannot carry. The records flow from the sandboxed convert (resolveSandboxedRunStdout)
    through the new parser to the gate's coverage step; they NEVER enter
    `ParsePackFindings`/SARIF. The channel is a DISTINCT typed output, never coverage
    tunneled through SARIF result `properties`. (REQ-015 — the canonical CoverageRecord)
    Define the producer-side, per-FILE, path-keyed record
    `{path, covered, total, measured, excluded, metric}` with RAW COUNTS (covered/total),
    not a pre-computed percent — the GATE computes `covered/total ≥ threshold` so it
    stays metric-BLIND and the pack bakes no percentage. Two conventions are MANDATORY:
    (a) `total == 0` (no-executable-lines: pure declarations/interfaces) ⇒ N/A, NEVER a
    0%-fail; (b) `metric` is a PACK-DECLARED label (statement/line/branch/…) that MUST be
    surfaced on the report, so a polyglot repo cannot silently compare one language's
    statement-% against another's branch-% under a single number; the gate never
    interprets `metric`. This is the CANONICAL definition; SPEC-041's DRAFT consumer
    `CoverageRecord` (as drafted, `{Path, Pct, Measured, Excluded}` — a paper declaration,
    NOT landed on `main`) is RECONCILED to this (Pct → Covered/Total + Metric) — two draft
    specs must agree; the producer↔consumer contract is stated explicitly and flagged for
    coherence review, not re-authored into SPEC-041 here. (REQ-016 —
    the go-toolchain coverage engine, first concrete producer) The `go-toolchain` pack
    gains a coverage engine: a binding declaring `gate_type: coverage` whose command runs
    `go test -coverprofile=…` and whose convert script (`scripts/coverage-to-records.sh`)
    turns the Go coverage profile into per-file coverage records with `metric: "statement"`
    (Go's `-coverprofile` granularity). The record SHAPE is language-agnostic, so a future
    `typescript-toolchain` coverage engine (istanbul/nyc → the SAME records) emits the same
    shape — backstop stays format- and language-blind. A REAL end-to-end test is MANDATED:
    an INSTALLED go-toolchain pack's coverage engine running through the UN-STUBBED
    sandboxed dispatch, producing real per-file records the gate consumes — NOT
    testdata+stubbed records, NOT a parallel raw-exec path (the pack-provisioning
    integration gap that bit SPEC-035 P4 / SPEC-037 Seed 3 / SPEC-035 Seed 4).
  package: cmd/backstop

verification:
  level: integration
  test_command: go test ./cmd/backstop/ ./pkg/check/ ./pkg/pack/engine/ -race -coverprofile=cover.out
  coverage_threshold: 80

requirements:
  - id: REQ-001
    text: >
      The engine model (`dispatchPackEngines`, cmd/backstop/pack_gate.go) MUST gain a
      SECOND normalized output type — coverage-records — DISTINCT from SARIF findings.
      An engine whose declared binding fills gate-type `coverage`
      (`engine.GateTypeCoverage`) MUST have its convert output routed to a NEW
      coverage-records parser (`check.ParsePackCoverage`), NOT to the SARIF findings
      parser (`check.ParsePackFindings`). The routing key is the binding's DECLARED
      `GateType` — never a tool name, command prefix, or pack name. It is PROHIBITED for
      a coverage engine's output to be parsed as SARIF, and PROHIBITED for any of the
      four findings/output gate-types (lint, build, test, findings) to be parsed as
      coverage-records: lint/build/test/findings → SARIF (`ParsePackFindings`);
      coverage → coverage-records (`ParsePackCoverage`); substantiveness/contracts have
      their own dedicated steps and are not in this channel. The coverage-records channel
      MUST be a DISTINCT typed output (`[]check.CoverageRecord`), NEVER coverage tunneled
      through SARIF result `properties` or any other SARIF field.
    supports: collapse-legacy-codecheck-into-packs:REQ-014
    follows: STD-GO-001:GO-010
  - id: REQ-002
    text: >
      The architectural decision MUST be recorded and honored: SARIF stays the
      findings/OUTPUT lingua franca; coverage is a per-file measurement INPUT the gate
      consumes and turns into ordinary gate violations (via SPEC-041's coverage step) —
      it is NOT a competing report format. Coverage CANNOT be expressed as SARIF because
      SARIF-as-findings represents sparse, located FAILURES, whereas coverage is a
      COMPLETE per-file census that MUST include PASSING (measured-and-above-threshold)
      files; non-vacuousness (SPEC-041 REQ-003) requires distinguishing
      `measured == true` (a file the engine measured) from a file with NO record
      (`measured == false` / not-measured), a distinction the findings stream
      structurally cannot carry. Therefore the coverage-records channel MUST carry a
      record for measured-and-passing files (not only shortfalls): a coverage engine that
      emitted records ONLY for below-threshold files would be a SARIF-findings stream in
      disguise and is PROHIBITED.
    supports: collapse-legacy-codecheck-into-packs:REQ-014
    follows: STD-GO-001:GO-010
  - id: REQ-003
    text: >
      Define the CANONICAL producer-side coverage record `check.CoverageRecord` as a
      per-FILE, path-keyed value with fields
      `{Path string, Covered int, Total int, Measured bool, Excluded bool, Metric string}`.
      The record MUST carry RAW COUNTS (`Covered`/`Total`), NOT a pre-computed percent:
      the GATE computes `Covered/Total ≥ threshold` so it stays metric-BLIND and the pack
      bakes no percentage. `Path` is the toolchain-DECLARED file path (FILE granularity);
      there MUST be NO "package" noun in the record (package is a Go-native concept that
      would re-bake language knowledge). The record is the CANONICAL definition the
      producer emits and to which SPEC-041's DRAFT consumer `CoverageRecord` (a paper
      declaration, not landed on `main`) is reconciled (`Pct` → `Covered`/`Total`, plus
      `Metric`).
    supports: collapse-legacy-codecheck-into-packs:REQ-015
    follows: STD-GO-001:GO-010
  - id: REQ-004
    text: >
      The parser `check.ParsePackCoverage` MUST honor the `total == 0` convention: a
      record with `Total == 0` (no executable lines — pure declarations, interfaces,
      generated stubs) is N/A and MUST NOT be treated as a 0%-fail. The parser MUST
      preserve `Total == 0` faithfully (it parses, it does not synthesize a percentage)
      so the consumer's threshold check can skip it as N/A rather than red it. It is
      PROHIBITED for the producer channel to coerce a `Total == 0` file into a 0%
      coverage value.
    supports: collapse-legacy-codecheck-into-packs:REQ-015
    follows: STD-GO-001:GO-010
  - id: REQ-005
    text: >
      The `Metric` field MUST be a PACK-DECLARED label (e.g. statement, line, branch)
      that the producer channel carries through unchanged and that MUST be SURFACED on the
      report surface, so a polyglot repo cannot silently compare one language's
      statement-% against another's branch-% under a single number. The gate MUST stay
      metric-BLIND: it NEVER interprets, compares, or branches on the `Metric` value — it
      only surfaces it. A record with an EMPTY `Metric` from an engine that produced
      measured records MUST be a fail-loud parse/produce error (an unlabeled measurement is
      a silent-comparison hazard), never a silently-accepted blank.
    supports: collapse-legacy-codecheck-into-packs:REQ-015
    follows: STD-GO-001:GO-010
  - id: REQ-006
    text: >
      The producer↔consumer contract between this spec (Seed 4 producer) and SPEC-041
      (Seed 3 consumer) MUST be explicit and the two DRAFT record definitions MUST agree
      (both are draft sibling specs, NOT landed on `main`; neither record exists in code
      yet): SPEC-041's drafted consumer `CoverageRecord` (`{Path, Pct, Measured, Excluded}`)
      is RECONCILED to the canonical producer record (`Pct` replaced by `Covered`/`Total`,
      with `Metric` added) so a single `check.CoverageRecord` type is the shared carrier
      across the producer (dispatch) and consumer (gate coverage step). It is PROHIBITED
      to ship TWO divergent `CoverageRecord` shapes (a producer one and a consumer one)
      that must be lossily translated between dispatch and the gate step. The reconciliation
      is flagged for producer↔consumer coherence review, not silently forked.
    supports: collapse-legacy-codecheck-into-packs:REQ-015
    follows: STD-GO-001:GO-010
  - id: REQ-007
    text: >
      The `go-toolchain` pack MUST gain a coverage engine — the FIRST concrete producer —
      declared as an EngineBinding with `gate_type: coverage` whose command runs
      `go test -coverprofile=…` and whose pack-relative convert script
      (`scripts/coverage-to-records.sh`) transforms the Go coverage profile into per-file
      `CoverageRecord`s with `Metric: "statement"` (Go's `-coverprofile` granularity). The
      convert emits coverage-records JSON, NOT SARIF. The pack DATA (the binding + the
      convert script) carries all the Go-toolchain knowledge; backstop's dispatch stays
      tool- and language-blind (it runs the declared command and routes the declared
      gate-type, baking no `go`/`-coverprofile`/Go-profile knowledge into the binary). The
      record SHAPE is language-agnostic so a future `typescript-toolchain` coverage engine
      (istanbul/nyc) emits the SAME records.
    supports: collapse-legacy-codecheck-into-packs:REQ-016
    follows: STD-GO-001:GO-010
  - id: REQ-008
    text: >
      The go-toolchain coverage engine MUST be proven by a REAL end-to-end test: an
      INSTALLED go-toolchain pack's coverage engine running through the UN-STUBBED
      sandboxed dispatch (the real `dispatchPackEngines` coverage path executing the pack's
      REAL `scripts/coverage-to-records.sh` convert via `resolveSandboxedRunStdout`),
      producing real per-file `CoverageRecord`s that the gate's coverage step consumes —
      with at least one MEASURED-AND-PASSING file, one MEASURED-AND-FAILING file, and one
      `Total == 0` (N/A) file in the records. It is PROHIBITED to satisfy this requirement
      with testdata + a stubbed convert returning canned records, or with a parallel
      raw-exec path that bypasses the real convert script — the producer must be exercised
      over the real installed-pack convert, per the pack-provisioning integration gap.
    supports: collapse-legacy-codecheck-into-packs:REQ-016
    follows: STD-GO-001:GO-010

claims:
  # REQ-001 — coverage-records is a SECOND normalized output type, routed by declared gate-type
  - id: CLM-001
    requirement: REQ-001
    text: A coverage engine (binding GateType == engine.GateTypeCoverage) has its convert output routed through the NEW coverage-records parser check.ParsePackCoverage and produces []check.CoverageRecord — proven by dispatching a coverage-gate-type rule and asserting records, not SARIF violations, come back
    tests:
      - TestDispatch_CoverageGateTypeRoutesToRecordsParser
  - id: CLM-002
    requirement: REQ-001
    text: A coverage engine's convert output is NEVER parsed as SARIF — check.ParsePackFindings is not invoked for a GateTypeCoverage engine; a coverage-records payload fed to ParsePackFindings would not silently become findings
    tests:
      - TestDispatch_CoverageEngineOutputNotParsedAsSarif
  - id: CLM-003
    requirement: REQ-001
    text: A LINT engine (golangci, GateTypeLint) output routes to SARIF (ParsePackFindings), NOT to the coverage-records parser — the records channel does not absorb a lint engine
    tests:
      - TestDispatch_LintEngineRoutesToSarifNotCoverage
  - id: CLM-004
    requirement: REQ-001
    text: A BUILD engine (go-build, GateTypeBuild) output routes to SARIF (ParsePackFindings), NOT to the coverage-records parser
    tests:
      - TestDispatch_BuildEngineRoutesToSarifNotCoverage
  - id: CLM-005
    requirement: REQ-001
    text: A TEST engine (go-test, GateTypeTest) output routes to SARIF (ParsePackFindings), NOT to the coverage-records parser — a test-failure engine and a coverage engine over the same `go test` family stay distinct channels
    tests:
      - TestDispatch_TestEngineRoutesToSarifNotCoverage
  - id: CLM-006
    requirement: REQ-001
    text: A FINDINGS engine (semgrep/ast-grep, GateTypeFindings) output routes to SARIF (ParsePackFindings), NOT to the coverage-records parser
    tests:
      - TestDispatch_FindingsEngineRoutesToSarifNotCoverage
  - id: CLM-007
    requirement: REQ-001
    text: The coverage-records channel is a DISTINCT typed output ([]check.CoverageRecord), never coverage tunneled through a SARIF result's properties — a SARIF document carrying coverage-shaped data in result.properties is NOT accepted by the coverage path and a coverage record is NOT emitted as a SARIF result
    tests:
      - TestDispatch_CoverageNotTunneledThroughSarifProperties
  # REQ-002 — SARIF is OUTPUT, coverage is INPUT; the census must include passing files
  - id: CLM-008
    requirement: REQ-002
    text: The coverage channel carries a record for a MEASURED-AND-PASSING file (Measured true, Covered/Total above any threshold) — the census includes passing files, not only shortfalls; an engine emitting records only for below-threshold files is rejected as a SARIF-findings stream in disguise
    tests:
      - TestCoverage_ChannelCarriesMeasuredPassingFiles
  - id: CLM-009
    requirement: REQ-002
    text: A file the coverage engine should have measured but did NOT (absent from the records) is distinguishable as not-measured (Measured false / no record), and the channel preserves that distinction from measured-and-passed — the load-bearing reason coverage cannot be SARIF, asserted by feeding a record-set that omits an expected file and observing the not-measured signal survives to the consumer boundary
    tests:
      - TestCoverage_MeasuredPassedVsNotMeasuredDistinguished
  # REQ-003 — canonical CoverageRecord: raw counts, per-file, no package noun
  - id: CLM-010
    requirement: REQ-003
    text: check.CoverageRecord is defined with fields Path/Covered/Total/Measured/Excluded/Metric — raw integer counts (Covered, Total), NOT a float percent; a structural test asserts the field set and that no Pct/percent float field exists on the producer record
    tests:
      - TestCoverageRecord_RawCountsNoPrecomputedPercent
  - id: CLM-011
    requirement: REQ-003
    text: The gate (consumer) computes the threshold verdict from Covered/Total, staying metric-blind — proven by feeding raw counts and asserting the pass/fail verdict matches Covered/Total >= threshold, with the producer never emitting a percentage
    tests:
      - TestCoverageRecord_GateComputesPercentFromRawCounts
  - id: CLM-012
    requirement: REQ-003
    text: CoverageRecord is per-FILE and path-keyed with NO "package" noun — a structural/field test asserts Path is a file path and there is no Package field or package-granular modeling on the record
    tests:
      - TestCoverageRecord_PerFilePathKeyedNoPackageNoun
  # REQ-004 — total == 0 is N/A, never 0%-fail
  - id: CLM-013
    requirement: REQ-004
    text: ParsePackCoverage preserves Total == 0 faithfully for a no-executable-lines file (pure declarations/interfaces) — it does NOT synthesize a 0% value; the parsed record carries Total == 0 so the consumer can treat it as N/A
    tests:
      - TestParseCoverage_TotalZeroPreservedAsNA
  - id: CLM-014
    requirement: REQ-004
    text: A Total == 0 record is treated as N/A by the verdict (skipped), never a 0%-fail — feeding a Total == 0 record yields no coverage shortfall violation, where a naive 0/0 → 0% would have failed
    tests:
      - TestCoverage_TotalZeroIsNAnotFail
  # REQ-005 — metric is a pack-declared label, surfaced, gate stays metric-blind
  - id: CLM-015
    requirement: REQ-005
    text: The pack-declared Metric label (e.g. "statement") is carried through the channel unchanged and SURFACED on the report surface for measured records, so a statement-% is never silently compared against a branch-% under one number
    tests:
      - TestCoverage_MetricLabelCarriedAndSurfacedOnReport
  - id: CLM-016
    requirement: REQ-005
    text: The gate stays metric-BLIND — it never interprets, compares, or branches on the Metric value; two records with different Metric labels but identical Covered/Total produce identical verdicts (the gate only surfaces the label, never decides on it)
    tests:
      - TestCoverage_GateIsMetricBlind
  - id: CLM-017
    requirement: REQ-005
    text: A MEASURED record with an EMPTY Metric is a fail-loud produce/parse error (an unlabeled measurement is a silent-comparison hazard), never silently accepted as blank
    tests:
      - TestParseCoverage_EmptyMetricOnMeasuredRecordFailsLoud
  # REQ-006 — producer↔consumer contract: one reconciled CoverageRecord type
  - id: CLM-018
    requirement: REQ-006
    text: A SINGLE check.CoverageRecord type is the shared carrier across producer (dispatch) and consumer (gate coverage step) — SPEC-041's {Path, Pct, Measured, Excluded} is reconciled to {Path, Covered, Total, Measured, Excluded, Metric}; a test asserts the consumer step consumes the same canonical type the producer emits, with no second divergent shape
    tests:
      - TestCoverageRecord_SingleReconciledTypeAcrossProducerAndConsumer
  - id: CLM-019
    requirement: REQ-006
    text: The producer-emitted []check.CoverageRecord is consumed directly by the gate coverage step with no lossy producer→consumer translation layer — the records dispatch produces flow into the coverage step unchanged
    tests:
      - TestCoverageRecord_ProducerRecordsFlowToCoverageStepUntranslated
  # REQ-007 — go-toolchain coverage engine: declared binding + convert, language-agnostic shape
  - id: CLM-020
    requirement: REQ-007
    text: The go-toolchain pack declares a coverage engine whose binding fills gate_type coverage, whose command runs `go test -coverprofile`, and whose pack-relative convert (scripts/coverage-to-records.sh) is declared — asserted against the pack.yml + binding records
    tests:
      - TestGoToolchain_CoverageEngineDeclaredWithCoverageGateType
  - id: CLM-021
    requirement: REQ-007
    text: The go-toolchain convert script turns a Go coverage profile into per-file CoverageRecords stamped Metric "statement" (Go's -coverprofile granularity) — proven by feeding a real Go coverage profile to the script and asserting per-file records with Metric "statement"
    tests:
      - TestGoToolchain_ConvertProfileToPerFileStatementRecords
  - id: CLM-022
    requirement: REQ-007
    text: backstop dispatch bakes NO go/-coverprofile/Go-profile knowledge — the Go-toolchain coverage knowledge lives entirely in the pack (binding command + convert script); a guard asserts cmd/backstop/pack_gate.go constructs no `go test -coverprofile` command and parses no Go coverage profile text
    tests:
      - TestDispatch_NoBakedGoCoverageKnowledge
  - id: CLM-023
    requirement: REQ-007
    text: The record shape is language-agnostic — a record set as would be produced by a future typescript-toolchain coverage engine (istanbul/nyc, Metric "line"/"branch") parses through the SAME check.ParsePackCoverage into the SAME check.CoverageRecord shape, proving no Go-specific assumption is baked into the record or parser
    tests:
      - TestParseCoverage_LanguageAgnosticRecordShape
  # REQ-008 — REAL installed-pack end-to-end (un-stubbed convert), not testdata+stubbed
  - id: CLM-024
    requirement: REQ-008
    text: An INSTALLED go-toolchain pack's coverage engine runs through the UN-STUBBED sandboxed dispatch — the real dispatchPackEngines coverage path executes the pack's REAL scripts/coverage-to-records.sh via resolveSandboxedRunStdout (convert NOT stubbed) and produces real per-file CoverageRecords; a stubbed convert returning canned records would NOT satisfy this test
    tests:
      - TestGoToolchain_CoverageEngineRealEndToEndOverInstalledPack
  - id: CLM-025
    requirement: REQ-008
    text: The real end-to-end records the gate consumes include a MEASURED-AND-PASSING file, a MEASURED-AND-FAILING (below-threshold) file, and a Total == 0 (N/A) file — and the gate's verdict over them is correct (the failing file REDs, the passing and N/A files do not), proving the producer feeds a non-vacuous consumer over the real convert
    tests:
      - TestGoToolchain_RealEndToEndRecordsDriveCorrectGateVerdict

contracts:
  - file: pkg/check/coverage.go
    provides:
      - name: CoverageRecord
        kind: type
        signature: "type CoverageRecord struct"
        notes: "NEW canonical producer-side record — one FILE's coverage as normalized by a coverage engine's convert: Path (toolchain-declared file path, FILE granularity, NO package noun), Covered int, Total int (RAW counts — the gate computes Covered/Total >= threshold and stays metric-blind; NO pre-computed percent), Measured bool (whether the engine measured this file; the not-measured / measured-and-passed distinction SARIF-as-findings cannot carry — REQ-002), Excluded bool (pack-DECLARED exclusion: generated/vendored/no-executable-line), Metric string (pack-declared label statement/line/branch — surfaced on report, never interpreted by the gate). This is the canonical type SPEC-041's DRAFT consumer CoverageRecord (drafted {Path, Pct, Measured, Excluded} — a paper declaration, NOT landed on main) is reconciled to (Pct → Covered/Total + Metric); two draft specs that must agree, not a match against landed code; a SINGLE shared carrier across producer (dispatch) and consumer (gate step), no second divergent shape (REQ-006)."
      - name: ParsePackCoverage
        kind: function
        signature: "func ParsePackCoverage(out []byte) ([]CoverageRecord, error)"
        notes: "NEW (REQ-001/REQ-004/REQ-005): parses a coverage engine's normalized coverage-records JSON (the SECOND output type, DISTINCT from SARIF findings) into []CoverageRecord — the coverage analogue of ParsePackFindings. Preserves Total == 0 faithfully (no synthesized 0% — REQ-004/CLM-013); fail-louds on a MEASURED record with an empty Metric (REQ-005/CLM-017). It is NOT SARIF and must not accept SARIF; coverage data tunneled through SARIF result.properties is rejected (CLM-007)."
    consumes: []
  - file: cmd/backstop/pack_gate.go
    provides:
      - name: dispatchPackCoverage
        kind: function
        signature: "func dispatchPackCoverage(packs []*pack.Manifest, packDir, projectRoot string, scope *gate.GateScope, runner check.CommandRunner) ([]check.CoverageRecord, error)"
        notes: "NEW (REQ-001/REQ-007): the coverage-records dispatch path. Runs each installed-pack engine whose binding declares gate_type coverage (engine.GateTypeCoverage), pipes the engine's stdout through the pack's declared convert via resolveSandboxedRunStdout, and parses the normalized output via check.ParsePackCoverage (NOT ParsePackFindings). Mirrors runFindingsEngine's run→convert step but terminates in the coverage-records parser — the SECOND normalized output type. Routes purely on the binding's DECLARED GateType, baking no go/-coverprofile/Go-profile knowledge (CLM-022). The records flow to the gate's coverage step (SPEC-041's StepCoverageThreshold consumer)."
    consumes:
      - source: pkg/pack/engine
        name: GateTypeCoverage
        kind: constant
      - source: pkg/check
        name: CoverageRecord
        kind: type
      - source: pkg/check
        name: ParsePackCoverage
        kind: function
  - file: cmd/backstop/testdata/go-toolchain/.backstop/packs/backstop/go-toolchain/pack.yml
    provides:
      - name: go-coverage-rule
        kind: variable
        signature: "rule id: go-coverage, engine: go-coverage, gate_type: coverage"
        notes: "NEW (REQ-007/CLM-020): the go-toolchain pack's coverage rule/engine. The engine binding declares gate_type coverage, command `go test -coverprofile=...`, and convert scripts/coverage-to-records.sh. Pack DATA carries all Go-toolchain knowledge; the engine model bakes none."
    consumes: []
  - file: cmd/backstop/testdata/go-toolchain/.backstop/packs/backstop/go-toolchain/scripts/coverage-to-records.sh
    provides:
      - name: coverage-to-records
        kind: variable
        signature: "stdin: go coverage profile; stdout: coverage-records JSON"
        notes: "NEW (REQ-007/CLM-021): the go-toolchain coverage convert script. Reads a Go `-coverprofile` profile on stdin and emits per-file coverage-records JSON ({path, covered, total, measured, excluded, metric:\"statement\"}) on stdout — NOT SARIF. Re-expresses Go coverage-profile knowledge OUTSIDE the binary, the coverage analogue of test-to-sarif.sh. Emits a record for measured-and-PASSING files too (REQ-002/CLM-008), not only shortfalls."
    consumes: []
---

# SPEC-042: Coverage Production Engine

## Overview

This spec is **Seed 4 of BUNDLE-011** (collapse the legacy `pkg/check` engine into
pack-declared toolchain packs), owning **REQ-014, REQ-015, REQ-016** (DD-7). It is the
**PRODUCER side of coverage** — the counterpart to **SPEC-041 (Seed 3)**, which is the
**CONSUMER**. SPEC-041 is a **DRAFT** sibling spec — it *declares* (on paper) a
re-implemented coverage gate step that consumes a language-agnostic per-file
`[]CoverageRecord`; **it is not landed on `main`**, and neither is that record type. The
**live** coverage on `main` is the OLD baked Go analyzer `StepCoverageThresholdFunc`
(`pkg/gate/step_coverage.go`), which runs `go test -coverprofile` and parses Go coverage
text in-binary — the very analyzer BUNDLE-011 REQ-011 / Seed 3 eradicates. So **nothing on
`main` produces a normalized per-file coverage record**: the go-toolchain test engine emits
only test-failure SARIF (`scripts/test-to-sarif.sh`), and `dispatchPackEngines`
(cmd/backstop/pack_gate.go) normalizes **every** engine's output to SARIF
(`runFindingsEngine` → `check.ParsePackFindings`) or runs the sandbox engine — **there is no
coverage-records output channel**. Without this spec, SPEC-041's drafted coverage consumer
is unsatisfiable. This spec adds the producer.

Three deliverables, all from DD-7:

1. **The coverage-records channel — the architectural core (bundle REQ-014).** Give the
   engine model a **SECOND normalized output type: coverage-records, a per-file
   measurement INPUT distinct from SARIF findings.** The load-bearing architectural line:
   **SARIF stays the findings/OUTPUT lingua franca; coverage is a MEASUREMENT INPUT** the
   gate consumes and turns into ordinary gate violations (via SPEC-041's coverage step) —
   it is **NOT a competing report format**. A declared engine whose binding fills
   gate-type `coverage` (`engine.GateTypeCoverage`, already in the enum — pkg/pack/engine/
   gatetype.go:25) routes its convert output to a **new coverage-records parser
   (`check.ParsePackCoverage`)** instead of the SARIF findings parser. The records flow
   from the sandboxed convert (`resolveSandboxedRunStdout`) through the new parser to the
   gate's coverage step.

   **Why coverage CANNOT be SARIF (the load-bearing reason it needs its own channel):**
   SARIF-as-findings represents **failures** — sparse, located. Coverage is a **complete
   per-file census** that must include **passing** files, because non-vacuousness
   (SPEC-041 REQ-003) requires distinguishing **measured-and-passed** from
   **not-measured** — and SARIF-as-findings structurally cannot express
   "this file was measured and is fine." Squeezing coverage into the findings stream loses
   exactly the distinction the anti-vacuous-green guardrail depends on.

2. **The canonical `CoverageRecord` (bundle REQ-015).** Define the per-FILE, path-keyed
   `{Path, Covered, Total, Measured, Excluded, Metric}` with **raw counts** (`Covered`/
   `Total`), **not** a pre-computed percent — the **gate** computes
   `Covered/Total ≥ threshold` so it stays **metric-blind** and the pack bakes no
   percentage. Two conventions are mandatory:
   (a) **`Total == 0` (no executable lines: pure declarations/interfaces) ⇒ N/A**, never a
   0%-fail; and (b) **`Metric` is a pack-declared label** (statement/line/branch/…) that
   **must be surfaced on the report**, so a polyglot repo cannot silently compare one
   language's statement-% against another's branch-% under a single number; the gate never
   interprets `Metric`. **This is the canonical definition; SPEC-041's DRAFT consumer
   `CoverageRecord` (as drafted, `{Path, Pct, Measured, Excluded}` — a paper declaration,
   not landed on `main`) is reconciled to it** (`Pct` → `Covered`/`Total`, `Metric` added)
   so a single shared type crosses producer and consumer. These are two draft specs that
   must agree, not a match against a landed type.

3. **The go-toolchain coverage engine — the first concrete producer (bundle REQ-016).**
   The `go-toolchain` pack gains a coverage engine: a binding declaring `gate_type:
   coverage` whose command runs `go test -coverprofile=…` and whose convert script
   (`scripts/coverage-to-records.sh`) turns the Go coverage profile into per-file records
   with `Metric: "statement"` (Go's `-coverprofile` granularity). The record **shape is
   language-agnostic** so a future `typescript-toolchain` coverage engine (istanbul/nyc)
   emits the **same** records. A **real end-to-end test is mandated** — an installed
   go-toolchain pack's coverage engine through the un-stubbed sandboxed dispatch producing
   real per-file records the gate consumes.

**In scope:** the coverage-records output channel + its parser, the canonical
`CoverageRecord` type + its conventions, the go-toolchain coverage engine (binding +
convert), and the real installed-pack end-to-end producer test.

**Out of scope (fenced to sibling seeds):** the coverage gate STEP / threshold logic /
non-vacuousness verdict, the build-pass exemption, and the CheckType-consumer catalog →
**Seed 3 / SPEC-041** (the consumer — referenced, not re-authored here; this spec only
*reconciles* its `CoverageRecord` shape). The Step-2 cutover, the `go-toolchain` toolchain
substrate/dispatch wiring, and the no-pack loud-warn report state → **Seed 2 / SPEC-040**
(the prerequisite this spec depends on). The dead-code deletions → **Seed 1 / SPEC-039**.

## Requirements

Requirements are enumerated in the `requirements:` frontmatter (REQ-001 … REQ-008), each
tracing to a BUNDLE-011 requirement via `supports`. Mapping:

| Spec REQ | Bundle REQ (DD-7) | What it commits to |
| --- | --- | --- |
| REQ-001 | REQ-014 | Coverage-records is a SECOND normalized output type, DISTINCT from SARIF; routed by the binding's DECLARED `GateType` (coverage → `ParsePackCoverage`; lint/build/test/findings → SARIF `ParsePackFindings`); never tunneled through SARIF properties. |
| REQ-002 | REQ-014 | SARIF is the OUTPUT lingua franca, coverage is a measurement INPUT (not a competing report format); the census must carry measured-and-passing files so measured-and-passed is distinguishable from not-measured — the reason coverage can't be SARIF. |
| REQ-003 | REQ-015 | Canonical `check.CoverageRecord{Path, Covered, Total, Measured, Excluded, Metric}` — raw counts (no percent), per-FILE/path-keyed, NO "package" noun; the gate computes the percentage. |
| REQ-004 | REQ-015 | `Total == 0` ⇒ N/A, preserved faithfully by the parser, never coerced to a 0%-fail. |
| REQ-005 | REQ-015 | `Metric` is a pack-declared label, surfaced on the report, gate stays metric-blind; an empty `Metric` on a measured record fails loud. |
| REQ-006 | REQ-015 | Producer↔consumer contract: a SINGLE reconciled `CoverageRecord` type (SPEC-041's `Pct` → `Covered`/`Total` + `Metric`); no second divergent shape, no lossy translation. |
| REQ-007 | REQ-016 | go-toolchain coverage engine: binding `gate_type: coverage`, command `go test -coverprofile`, convert `scripts/coverage-to-records.sh` → `Metric "statement"`; language-agnostic shape; backstop bakes no Go-coverage knowledge. |
| REQ-008 | REQ-016 | REAL end-to-end: an INSTALLED go-toolchain pack's coverage engine through the UN-STUBBED sandboxed dispatch produces real per-file records the gate consumes — not testdata+stubbed, not a parallel raw-exec path. |

### The two normalized output types (REQ-001)

The engine model has exactly two normalized output types, routed by the binding's
**declared `GateType`** — never a tool name, command prefix, or pack name. This table is
the routing contract; the implementation and CLM-001…CLM-007 enforce it.

| Gate type | Normalized output | Parser | Channel |
| --- | --- | --- | --- |
| `lint` | SARIF findings | `check.ParsePackFindings` | findings/OUTPUT |
| `build` | SARIF findings | `check.ParsePackFindings` | findings/OUTPUT |
| `test` | SARIF findings | `check.ParsePackFindings` | findings/OUTPUT |
| `findings` | SARIF findings | `check.ParsePackFindings` | findings/OUTPUT |
| `coverage` | coverage-records | `check.ParsePackCoverage` | **measurement/INPUT (NEW)** |
| `substantiveness` | (dedicated step) | — | dedicated traceability step |
| `contracts` | (dedicated step) | — | dedicated traceability step |

`substantiveness`/`contracts` already route to their own dedicated gate steps
(`gateTypeHasDedicatedStep`, pack_gate.go:102) and are not part of either normalized
channel here. Coverage is the **third** dedicated-step gate-type — but unlike the
traceability dimensions, its dedicated step (SPEC-041's `StepCoverageThreshold`) needs a
**produced input** (the records), which is exactly the channel this spec adds.

## Implementation

Target package: `cmd/backstop` (the coverage-records dispatch path) with the canonical
record + parser in `pkg/check` and the engine declaration in the `go-toolchain` pack data.
The processing steps the planner must map tasks to:

1. **Define the canonical `check.CoverageRecord` + `check.ParsePackCoverage` (REQ-003,
   REQ-004, REQ-005, REQ-006).** Add `pkg/check/coverage.go`: the per-FILE, path-keyed
   `CoverageRecord{Path, Covered, Total, Measured, Excluded, Metric}` (raw counts, no
   percent, no "package" noun) and `ParsePackCoverage([]byte) ([]CoverageRecord, error)`,
   the coverage analogue of `ParsePackFindings`. The parser preserves `Total == 0`
   faithfully (no synthesized 0% — REQ-004) and fail-louds on a measured record with an
   empty `Metric` (REQ-005). This is the **single shared carrier** to which SPEC-041's
   consumer `CoverageRecord` is reconciled (REQ-006) — coordinate the reconciliation with
   SPEC-041 rather than forking a second shape.

2. **Add the coverage-records dispatch path `dispatchPackCoverage` (REQ-001, REQ-002).**
   In `cmd/backstop/pack_gate.go`, add the coverage analogue of the findings dispatch:
   for each installed-pack engine whose binding declares `gate_type: coverage`
   (`engine.GateTypeCoverage`), run the engine command via the runner, pipe its stdout
   through the pack's declared convert via `resolveSandboxedRunStdout`, and parse the
   normalized output via `check.ParsePackCoverage` — **terminating in the coverage-records
   parser, NOT `ParsePackFindings`**. Route purely on the binding's **declared `GateType`**
   (CLM-001), so a coverage engine's output is **never** parsed as SARIF (CLM-002) and a
   lint/build/test/findings engine's output is **never** parsed as coverage-records
   (CLM-003…CLM-006). The records are a **distinct typed output** (`[]check.CoverageRecord`),
   never coverage tunneled through SARIF `properties` (CLM-007). The channel carries records
   for measured-and-passing files (REQ-002/CLM-008), preserving the measured-vs-not-measured
   distinction (CLM-009). Bake no `go`/`-coverprofile`/Go-profile knowledge into this path
   (CLM-022).

3. **Wire the produced records to the gate's coverage step (REQ-006).** The
   `[]check.CoverageRecord` `dispatchPackCoverage` produces flow into SPEC-041's
   `StepCoverageThreshold` consumer with no lossy producer→consumer translation layer
   (CLM-019) — the same canonical type crosses the producer/consumer boundary (CLM-018).
   (The threshold verdict logic itself is SPEC-041's; this spec only supplies the input.)

4. **Declare the go-toolchain coverage engine (REQ-007).** Add a coverage rule/engine to
   the `go-toolchain` pack (`pack.yml`): a binding with `gate_type: coverage`, command
   `go test -coverprofile=…`, and convert `scripts/coverage-to-records.sh`. All Go-toolchain
   knowledge lives in the **pack data** (the binding command + the convert script); the
   dispatch path stays tool- and language-blind. The engine binding is added via the
   **pack's `engines:` block** (the declared-engine substrate from SPEC-035) — the same way
   a pack declares its own ast-grep/semgrep engines — **NOT** a baked `DefaultRegistry`
   edit. (The existing `DefaultRegistry` golangci/go-build/go-test bindings, pkg/pack/engine/
   binding.go:270-308, are precisely the baked bindings BUNDLE-011 is eradicating, so they
   are the wrong template — do not mirror them; declare the coverage engine in the pack.)

5. **Write the go-toolchain coverage convert script (REQ-007).** Add
   `scripts/coverage-to-records.sh`: reads a Go `-coverprofile` profile on stdin and emits
   per-file coverage-records JSON (`{path, covered, total, measured, excluded,
   metric:"statement"}`) on stdout — **NOT SARIF**. It re-expresses Go coverage-profile
   knowledge OUTSIDE the binary (the coverage analogue of `test-to-sarif.sh`) and emits a
   record for measured-and-**passing** files too (REQ-002/CLM-008), not only shortfalls.
   The `Metric` is `"statement"` (Go's `-coverprofile` granularity).

6. **Real installed-pack end-to-end producer test (REQ-008).** Mandate an end-to-end test
   driving the INSTALLED go-toolchain pack's coverage engine through the **un-stubbed**
   sandboxed dispatch — the real `dispatchPackCoverage` executing the pack's **real**
   `scripts/coverage-to-records.sh` via `resolveSandboxedRunStdout` (convert NOT stubbed),
   producing real per-file `CoverageRecord`s the gate consumes (CLM-024), with at least one
   measured-and-passing file, one measured-and-failing file, and one `Total == 0` (N/A)
   file, and asserting the gate's verdict over them is correct (CLM-025). This is the
   pack-provisioning integration-gap guard: **no testdata+stubbed-convert and no parallel
   raw-exec path** may stand in for the real installed-pack convert.

## Verification

- **Level:** `integration` — the producer spans `pkg/check` (the canonical record +
  parser), `cmd/backstop` (the coverage-records dispatch path + the real installed-pack
  end-to-end), and the `go-toolchain` pack data (binding + convert script); the
  channel-distinctness and real-e2e claims are only meaningful end-to-end through the
  un-stubbed sandboxed dispatch.
- **Test command:** `go test ./cmd/backstop/ ./pkg/check/ ./pkg/pack/engine/ -race -coverprofile=cover.out`
- **Coverage threshold:** 80 (integration level).

Claims (CLM-NNN) are enumerated in the `claims:` frontmatter, each mapping a REQ to
mandated test names. The **gate-type routing matrix (REQ-001)** is fully covered: coverage
routes to the records parser (CLM-001) and is never parsed as SARIF (CLM-002), while each
of the four findings/output gate-types — lint (CLM-003), build (CLM-004), test (CLM-005),
findings (CLM-006) — routes to SARIF and never to the records channel; the tunneling
escape hatch is closed (CLM-007). Every requirement carries at least one positive and one
negative/edge claim. The real installed-pack end-to-end (REQ-008) is mandated by CLM-024
(un-stubbed convert) and CLM-025 (non-vacuous verdict over measured-passing /
measured-failing / N/A records).

## Sharp Edges

1. **The coverage channel must not degenerate into "coverage tunneled through SARIF
   properties."** The whole point of a SECOND output type is that coverage is a distinct
   typed measurement, not a findings document with coverage smuggled into result
   `properties`. The cheap-but-wrong implementation reuses `ParsePackFindings` and stuffs
   `covered/total` into a SARIF result's `properties` bag — which silently re-collapses the
   measured-vs-not-measured distinction the channel exists to preserve. CLM-007 explicitly
   guards this: a SARIF document carrying coverage-shaped `properties` is NOT accepted by
   the coverage path, and a coverage record is NOT emitted as a SARIF result. Route on the
   **declared `GateType`**, parse with `ParsePackCoverage`, carry `[]check.CoverageRecord`.

2. **`Total == 0` is N/A, not 0%.** A file with no executable lines (pure declarations,
   interfaces, generated stubs) has `Covered == 0, Total == 0`. A naive `Covered/Total`
   computes `0/0` — which either panics or, if guarded to `0.0`, reds the gate as a
   0%-coverage failure. That is a false positive on a file that legitimately has nothing to
   cover. The parser must preserve `Total == 0` faithfully (REQ-004/CLM-013) and the
   consumer treats it as N/A/skip (CLM-014). The producer must NOT coerce it to a percentage.

3. **The `Metric` label must be surfaced, never silently dropped.** Different languages
   (and even different configs) measure different things: Go reports **statement** coverage,
   istanbul reports **line/branch/function**, llvm-cov reports **region**. If the report
   shows a bare "84%" with no metric, a polyglot repo silently compares Go-statement-% against
   Rust-branch-% under one number — a meaningless aggregate that looks authoritative. The
   `Metric` is pack-declared, carried through unchanged, surfaced on the report (CLM-015),
   and the gate stays metric-blind (CLM-016). A **measured record with an empty `Metric`**
   is the silent-comparison hazard in seed form, so it fails loud (CLM-017) rather than
   surfacing an unlabeled number.

4. **Non-vacuousness depends on the census including PASSING files — coverage is not a
   sparse findings stream.** The strongest temptation is to emit records only for files
   below threshold (mirroring how findings engines emit only violations). That is precisely
   the SARIF-findings-in-disguise failure mode REQ-002 forbids: with only shortfall records,
   a file the engine **should** have measured but didn't (a build/instrumentation gap) is
   indistinguishable from a file that's fine — and SPEC-041's "no record for an in-scope
   changed non-excluded path ⇒ loud error" guardrail cannot fire, producing a vacuous green.
   The channel MUST carry measured-and-passing records (CLM-008) so measured-and-passed is
   distinguishable from not-measured (CLM-009).

5. **The pack-provisioning integration gap: a stubbed convert proves nothing.** This is a
   RECURRING failure mode (it bit SPEC-035 P4, SPEC-037 Seed 3, SPEC-035 Seed 4): the
   producer is "tested" with testdata + a stubbed dispatcher returning canned records,
   leaving the real installed-pack convert path unexercised — so a broken
   `coverage-to-records.sh`, a mis-declared binding, or a convert/parser shape mismatch ships
   green. REQ-008/CLM-024 mandates the un-stubbed path (the real convert via
   `resolveSandboxedRunStdout`, like the existing ast-grep e2e harness), and CLM-025 makes it
   non-vacuous (the records must drive a correct gate verdict over passing/failing/N/A files).

6. **Producer↔consumer record drift (the cross-spec hazard).** SPEC-041 (Seed 3) — a DRAFT
   sibling spec, NOT landed on `main` — drafted a consumer `CoverageRecord{Path, Pct,
   Measured, Excluded}` — a **pre-computed percent**, no metric. This spec's canonical
   producer record is `{Path, Covered, Total, Measured, Excluded, Metric}` — **raw counts**,
   with metric. Neither record exists in code yet; both are draft declarations that must
   agree. If both shapes ship, dispatch produces
   one and the gate step consumes the other, forcing a lossy translation (counts → percent,
   metric dropped) at the boundary — exactly the silent-comparison and 0/0 hazards above,
   reintroduced. REQ-006/CLM-018 mandates a SINGLE reconciled type; the reconciliation must
   be coordinated with SPEC-041 (per align-predating-artifacts: update the predating consumer
   shape openly, don't fork). Flag this for producer↔consumer coherence review.

7. **Route on declared `GateType`, never on tool/command identity.** It is tempting to
   detect "the coverage engine" by sniffing the command (`go test -coverprofile`) or the
   pack/engine name — exactly the baked tool-knowledge the thin-executor principle forbids.
   The routing key is the binding's **declared `gate_type: coverage`** (`engine.GateTypeCoverage`),
   identical in spirit to how `StrictSarif`/`PackageScoped` replaced the golangci/`go test`
   command-prefix sniffs. A future `typescript-toolchain` coverage engine declaring
   `gate_type: coverage` must route through the SAME path with zero new dispatch code
   (CLM-023).

## Review Questions

These probe risks not fully pinned by the claims; the impl-reviewer should check each
against the diff.

1. Does the coverage dispatch route SOLELY on `binding.GateType == engine.GateTypeCoverage`,
   with no command-string sniff (`-coverprofile`), no pack-name check, and no engine-name
   check anywhere in the coverage path? (Sharp Edge 7 / CLM-022.)
2. Is `Total == 0` carried through unmodified from the convert output into the parsed
   `CoverageRecord` (i.e. the parser does not compute, normalize, or default a percentage),
   so the N/A determination is the consumer's, not silently pre-decided by the producer?
   (REQ-004/CLM-013.)
3. Does the go-toolchain `coverage-to-records.sh` emit a record for at least one
   measured-and-**passing** file in the real e2e (not only below-threshold files), and would
   the test FAIL if the script were changed to emit only shortfalls? (REQ-002/CLM-008,
   Sharp Edge 4.)
4. In the real end-to-end test, is the convert script executed via the un-stubbed
   `resolveSandboxedRunStdout` (the pack's real script on disk), and would substituting a
   stub that returns canned records make the test pass trivially? If a stub would pass it,
   the test does not satisfy REQ-008. (CLM-024, Sharp Edge 5.)
5. Is there exactly ONE `CoverageRecord` type shared by `dispatchPackCoverage` (producer) and
   SPEC-041's coverage step (consumer), or are there two shapes with a translation layer
   between them? If SPEC-041 still defines its own `{Path, Pct, …}`, was its reconciliation
   coordinated (not silently left divergent)? (REQ-006/CLM-018, Sharp Edge 6.)
6. Does a measured record with an empty `Metric` actually fail loud (a real error path with a
   test), or is the empty-metric guard only asserted in a comment? (REQ-005/CLM-017.)
7. Is the go-toolchain coverage engine declared via the pack's `engines:` block (declared
   substrate), not added to the baked `DefaultRegistry`? (REQ-007, thin-executor.)

## References

- BUNDLE-011 (collapse-legacy-codecheck-into-packs) — **Seed 4**, REQ-014/REQ-015/REQ-016,
  DD-7. This spec is the coverage PRODUCER; Seed 3 (SPEC-041) is the CONSUMER.
- SPEC-041 (coverage-reimpl-checktype-catalog) — **Seed 3, the CONSUMER — a DRAFT sibling
  spec, NOT landed on `main`**. *Drafts* a gate coverage step over `[]CoverageRecord`; its
  *drafted* consumer record `{Path, Pct, Measured, Excluded}` is reconciled to this spec's
  canonical `{Path, Covered, Total, Measured, Excluded, Metric}` (REQ-006) — two draft specs
  that must agree, neither record in code yet. The LIVE coverage on `main` is instead the OLD
  baked `StepCoverageThresholdFunc` (`pkg/gate/step_coverage.go`) that Seed 3 eradicates.
  Producer↔consumer contract.
- SPEC-040 (toolchain-pack-cutover) — **Seed 2, the PREREQUISITE**. Establishes the
  toolchain-pack dispatch substrate (the `go-toolchain` cutover) this spec's coverage engine
  rides on. Out of scope here; depended on.
- Code (verified on `main` 2026-06-24): `cmd/backstop/pack_gate.go` —
  `dispatchPackEngines` normalizes EVERY engine to SARIF via `runFindingsEngine` →
  `check.ParsePackFindings` (the gap this spec fills with a second channel);
  `gateTypeHasDedicatedStep` (:102) already treats `GateTypeCoverage` as a dedicated step
  but nothing produces its input; `resolveSandboxedRunStdout` (:62) is the convert seam the
  coverage path reuses. `pkg/pack/engine/gatetype.go:25` — `GateTypeCoverage` already in the
  enum. `pkg/check/parsers.go:45` — `ParsePackFindings`, the SARIF analogue the new
  `ParsePackCoverage` parallels. `pkg/gate/step_coverage.go` — `StepCoverageThresholdFunc`,
  the OLD baked Go coverage analyzer (`go test -coverprofile` + Go-text parsing) that is the
  LIVE coverage on `main` and that Seed 3 (SPEC-041) eradicates; no normalized per-file
  coverage record exists on `main`. `cmd/backstop/testdata/go-toolchain/.../pack.yml` +
  `scripts/test-to-sarif.sh` — today the go-toolchain test engine emits only test-failure
  SARIF, no coverage. `cmd/backstop/dispatch_astgrep_e2e_test.go` — the real un-stubbed
  installed-pack convert e2e pattern REQ-008 must follow.
- [[project_thin_executor_engine_packs]] — backstop knows no engine, runs declared commands,
  speaks normalized output; the coverage channel adds a second normalized type without baking
  Go-coverage knowledge.
- [[project_pack_provisioning_integration_gap]] — the recurring testdata+stubbed gap REQ-008
  guards against.
- [[feedback_loud_not_blocking]] — the anti-vacuous-green philosophy the
  measured-vs-not-measured distinction (REQ-002) serves.
