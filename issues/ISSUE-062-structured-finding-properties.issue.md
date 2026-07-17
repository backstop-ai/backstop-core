---
title: "Structured Finding Properties — Pack Machine-Data in SARIF Properties, Not Parsed Prose"
schema_version: issue/v1

issue:
  id: ISSUE-062
  title: "Structured Finding Properties — Pack Machine-Data in SARIF Properties, Not Parsed Prose"
  type: technical-debt
  status: closed
  created: "2026-07-17"
  closed: "2026-07-17"

delivered_by: PLAN-ISSUE-062

complexity:
  scope: contained
  uncertainty: known
  risk: moderate

verification:
  level: unit
  coverage_threshold: 90
  test_command: "go test ./pkg/check/... ./pkg/gate/... -run 'Properties|Substantiveness'"

implementation:
  summary: >
    Give pack-emitted findings a structured channel to the gate: preserve SARIF
    result `properties` as a typed map through the check SARIF parser onto
    check.Violation and gate.Violation (following the ISSUE-059 `Trace` precedent —
    additive, omitempty, excluded from baseline identity). Rewrite the
    substantiveness join to read the enclosing test name and referenced symbol from
    those structured properties instead of parsing them out of the free-text message,
    and delete the whitespace-delimited message parsers (tokenValue /
    parseExtractionMessage / funcNameFromMessage) that silently truncate any name
    containing a space — the latent Go-func-name assumption that makes substantiveness
    incorrect for string-named (it()/test()) tests. Add a backstop/self rule that
    flags structural name-extraction on the neutral spine so the class stays caught.
  package: pkg/check, pkg/gate, packs/substantiveness, backstop-self-pack

requirements:
  - id: REQ-001
    text: >
      The check SARIF parser (`pkg/check/parsers.go`) must preserve result-level
      `properties` from a SARIF finding as a structured `Properties map[string]string`
      on `check.Violation`, populated from the SARIF result's `properties` object
      (string values only for v1). A result with no `properties` yields a nil/empty map,
      never an error. The change is additive: every existing field
      (rule/file/line/message/severity/fingerprint) is unchanged, and findings that
      carry no properties behave exactly as today.
  - id: REQ-002
    text: >
      `gate.Violation` must carry the structured `Properties` across from
      `check.Violation`, following the ISSUE-059 `Trace` precedent EXACTLY: an additive
      `omitempty` field that is DELIBERATELY EXCLUDED from baseline identity and
      RegionHash — `EnrichViolationIdentity` must continue to fold only
      `Rule|File|RegionHash(Message|Severity|SourcePack)`, so a violation gaining or
      losing `Properties` never destabilizes baseline grandfathering. Consumers reading
      only rule/file/message/severity are unaffected (additive under gate/v1).
  - id: REQ-003
    text: >
      The substantiveness join (`pkg/gate/substantiveness_join.go`) must read the
      enclosing test name and the referenced symbol from a finding's structured
      `Properties` (keys `func` and `symbol`), NOT by parsing the free-text `message`.
      The whitespace-delimited message parsers — `tokenValue`, `parseExtractionMessage`,
      `funcNameFromMessage` — must be DELETED. `ReferencedSetForTest` and `IsTestHollow`
      must join `MandatedTest.FuncName` against the property value verbatim, so the join
      is correct for a name containing spaces, quotes, or any character — i.e. a
      string-named vitest/jest test (`it('a b c', …)`), not only a single-token Go
      `TestXxx` func name.
  - id: REQ-004
    text: >
      The substantiveness pack's ast-grep convert script (`packs/substantiveness/
      ast-grep/to-sarif.sh`) must emit the matched metavariables — the enclosing test
      name and the referenced symbol — as SARIF result `properties` (`func`, `symbol`),
      read from ast-grep's structured `metaVariables.single.<NAME>.text`, not
      interpolated into the message. The human-readable `message` is retained for the
      report surface ("test X has no assertions (hollow)") but carries NO machine-parsed
      contract; the machine contract moves entirely to `properties`.
  - id: REQ-005
    text: >
      `backstop/self` must gain a rule that flags structural name/identifier extraction
      on the neutral gate spine — a whitespace-class string literal (`" "`, `" \t\n"`)
      or `strings.Fields` / `strings.Split(x, " ")` used to slice a value out of a
      message/string in the gate-spine files B3 already scopes (`pkg/gate/*.go`,
      dispatch/routing). This is the baked-language-assumption that produced `tokenValue`
      (Go identifiers are whitespace-free; other languages' test names are not) and which
      the existing literal-only rules (`.go`/`_test.go`/`./...`, tool names, extensions)
      cannot see. The rule closes the CLASS, not just the `tokenValue` instance; ships
      with a positive fixture (a whitespace-split name extractor) and a negative fixture
      (a structured-property read).
  - id: REQ-006
    text: >
      Core must carry an adversarial round-trip test proving the structured-properties
      path survives an enclosing test name containing spaces AND quotes (e.g. a name like
      a real vitest description) intact from the SARIF property through the join to the
      per-test verdict — so the whitespace assumption can never silently regress. A
      status-quo test that only exercises single-token Go names is insufficient and does
      not satisfy this requirement.

claims:
  - id: CLM-001
    requirement: REQ-001
    text: The check SARIF parser copies a result's string properties onto check.Violation.Properties.
    tests:
      - TestParseSarif_CarriesResultProperties
  - id: CLM-002
    requirement: REQ-001
    text: A SARIF result with no properties yields an empty or nil Properties map and no error.
    tests:
      - TestParseSarif_NoPropertiesIsEmptyNotError
  - id: CLM-003
    requirement: REQ-002
    text: gate.Violation carries Properties across from check.Violation additively.
    tests:
      - TestGateViolation_CarriesProperties
  - id: CLM-004
    requirement: REQ-002
    text: A violation's Properties are excluded from baseline identity and RegionHash.
    tests:
      - TestProperties_ExcludedFromBaselineIdentity
  - id: CLM-005
    requirement: REQ-003
    text: The substantiveness join reads func and symbol from Properties, not the message.
    tests:
      - TestSubstantivenessJoin_ReadsPropertiesNotMessage
  - id: CLM-006
    requirement: REQ-003
    text: The join is correct for a test name containing spaces.
    tests:
      - TestSubstantivenessJoin_SpacedTestNameJoins
  - id: CLM-007
    requirement: REQ-003
    kind: absence
    text: The whitespace-delimited message parsers are removed from the codebase.
    tests:
      - TestTokenValueParsersRemoved
  - id: CLM-008
    requirement: REQ-004
    text: The substantiveness convert emits func and symbol as SARIF properties from ast-grep metaVariables.
    tests:
      - TestSubstantivenessConvertEmitsPropertiesFromMetavars
  - id: CLM-009
    requirement: REQ-005
    text: The self rule flags a whitespace-split name extractor on the neutral spine.
    tests:
      - TestSelfRuleFlagsStructuralNameSplitOnSpine
  - id: CLM-010
    requirement: REQ-006
    text: A test name with spaces and quotes round-trips through properties to the per-test verdict.
    tests:
      - TestSubstantiveness_SpacedQuotedNameRoundTrips

contracts:
  - file: pkg/check/parsers.go
    provides:
      - name: Violation
        kind: type
        signature: "type Violation struct"
  - file: pkg/gate/substantiveness_join.go
    provides:
      - name: tokenValue
        kind: function
        signature: "func tokenValue(s, key string) string"
        absent: true
      - name: parseExtractionMessage
        kind: function
        signature: "func parseExtractionMessage(message string) (funcName, symbol string)"
        absent: true
---

# Structured Finding Properties — Pack Machine-Data in SARIF Properties, Not Parsed Prose

## Problem

A pack-emitted finding can reach the gate with only one machine-usable payload: the
free-text `message`. The check SARIF parser (`pkg/check/parsers.go` `sarifLog`) reads
`ruleId / level / message.text / locations / partialFingerprints / suppressions` and
**drops `properties`**; `check.Violation` and `gate.Violation` have no structured slot
for pack-sourced data (`gate.Violation.Trace` exists but is computed gate-side by the
traceability step, ISSUE-059, never sourced from a pack's SARIF). Coverage even
*explicitly rejects* tunneling structured data through SARIF properties
(`pkg/check/coverage.go`). So any pack that needs to hand the gate a structured value
is forced to stuff it into the message and have the gate parse it back out.

The substantiveness join is exactly this anti-pattern and shows its cost. The
substantiveness pack emits `"referenced-symbol func=$FN symbol=$PKG"`; the gate parses
it with `tokenValue` (`pkg/gate/substantiveness_join.go`), which reads `func=<name>` and
**stops at the first whitespace**. That is correct for a Go `TestXxx` func name (a single
whitespace-free token) and is exercised green by core's own Go tests and the pack's Go
fixtures — so nothing in a Go codebase ever fails. It is silently WRONG for any consumer
whose tests are string-named — vitest/jest `it('surfaces a plan spec_id …', …)` — where
the "func name" is a natural-language description with spaces: `func=surfaces a plan …`
truncates to `surfaces`, and the join breaks. This is the same class as the mandated-test
name-extractor apostrophe truncation. It surfaced building a TypeScript consumer
(bclabs-portal), which is the first non-Go exercise of the pack-to-gate finding protocol.

## Root cause

There is no structured channel for pack-to-gate finding data — the message string is the
only path, so machine-data must be parsed out of prose, and that parsing bakes a
language assumption (whitespace-free identifiers). The fix is the same move ISSUE-059
made gate-side, generalized to the pack-sourced case: carry machine-data in typed fields,
never parse it out of a human-readable message.

## Fix

1. **Generic properties pass-through (pkg/check).** Teach the SARIF parser to preserve a
   result's `properties` as `Properties map[string]string` on `check.Violation` (REQ-001).
   This is a reusable channel, not substantiveness-specific — any pack can now hand the
   gate typed data.
2. **Carry it on gate.Violation (pkg/gate).** Additive `omitempty`, excluded from baseline
   identity exactly like `Trace` (REQ-002).
3. **Rewrite the substantiveness join to read properties, delete the prose parsers**
   (REQ-003). `func`/`symbol` come from `Properties`; `tokenValue` /
   `parseExtractionMessage` / `funcNameFromMessage` are removed. The join then works for
   any name shape.
4. **Pack convert emits properties from ast-grep metaVariables** (REQ-004). ast-grep's
   `--json` already exposes `metaVariables.single.<NAME>.text`; the convert lifts them into
   SARIF `properties`. The prose message stays for humans.
5. **Self rule closes the class** (REQ-005). Flag whitespace/structural name-extraction on
   the neutral spine, so the next `tokenValue` is caught in core's own suite rather than by
   a downstream non-Go consumer.
6. **Adversarial round-trip test** (REQ-006). A spaced+quoted name must survive to the
   per-test verdict; Go-only single-token tests do not prove this.

## Out of scope

- The TypeScript substantiveness pack itself (`backstop/substantiveness`, `language:
  typescript` — the ast-grep hollow-test + referenced-symbol rules for `it()`/`test()`).
  It is ENABLED by this issue (it needs the properties channel + the whitespace-free join
  fix to work) but is separate pack authoring, tracked on its own.
- The `backstop/self` language-vs-project pack-mismatch check (a related but distinct
  self gap that would have flagged the Go contracts/substantiveness packs installed into a
  TS project at onboarding). Separate issue.

## Notes / references

- Precedent: ISSUE-059 (gate trace structured fields) — same principle (typed fields, not
  parsed prose), applied gate-side; this generalizes it to pack-sourced findings.
- The `Trace` field comment in `pkg/gate/result.go` documents the exact additive /
  omitempty / excluded-from-baseline-identity pattern REQ-002 must follow.
- ast-grep metaVariables verified present in `ast-grep scan --json` output
  (`metaVariables.single.FN.text`, `.PKG.text`), so REQ-004 is a convert-script change.
- Discovered dogfooding backstop on a TypeScript consumer (bclabs-portal); the portal is
  the first cross-language exercise of these packs, which is why a Go-shaped assumption
  latent since SPEC-037 surfaced now.

## Resolution

Delivered by PLAN-ISSUE-062.

- **REQ-001/002 (structured channel).** `check.Violation` and `gate.Violation` gained a
  `Properties map[string]string`. `parseSarif` (`pkg/check/parsers.go`) copies a SARIF
  result's string `properties` onto `check.Violation`; a property-less result yields a
  nil map, never an error. `gate.Violation.Properties` follows the ISSUE-059 `Trace`
  precedent — additive, `omitempty`, and DELIBERATELY excluded from baseline identity
  (`EnrichViolationIdentity` still folds only `Rule|File|RegionHash(Message|Severity|
  SourcePack)`). Properties are carried across at BOTH check→gate mapping sites: the
  production `runFindingsEngine` (`cmd/backstop/pack_gate.go`) and the pkg/gate dispatch
  glue (`substantiveness_q1_dispatch.go`).
- **REQ-003 (join rewrite).** `ReferencedSetForTest` / `IsTestHollow`
  (`pkg/gate/substantiveness_join.go`) now read `func`/`symbol` from `Properties` and
  compare to `MandatedTest.FuncName` verbatim, so a name with spaces or quotes joins
  correctly. The whitespace-delimited message parsers `tokenValue`,
  `parseExtractionMessage`, and `funcNameFromMessage` are DELETED.
- **REQ-004 (convert).** `packs/substantiveness/ast-grep/to-sarif.sh` lifts the matched
  `metaVariables.single.FN/PKG` into SARIF result `properties` (func/symbol); the rule
  messages are simplified to human-only.
- **REQ-005 (self rule).** `backstop/self` gained Family B5
  (`no-structural-name-split-on-spine`) flagging `strings.Fields` / `strings.Split(x,
  " ")` / `IndexAny` on a whitespace-class literal on the neutral spine, with a positive
  (structured read) and negative (whitespace-split) fixture. NOTE: activating B5 in
  backstop-core's own gate surfaces a pre-existing cascade (`splitCommand` /
  `engineToolName` command tokenization in `cmd/backstop/pack_gate*.go`) held for
  separate triage per the plan's cascade guard; the rule is authored and fixture-tested
  but not yet installed into `.backstop/packs`.
- **REQ-006 (adversarial round-trip).** `TestSubstantiveness_SpacedQuotedNameRoundTrips`
  drives a spaced+quoted vitest-style name through convert → parse → join to the correct
  per-test verdict, with a truncated-name negative control.
