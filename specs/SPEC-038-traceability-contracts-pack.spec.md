---
title: "Traceability Contracts Pack"
number: SPEC-038
created: "2026-06-22"
status: draft
schema_version: spec/v1
spec_version: 1.1.0

implementation:
  summary: >
    BUNDLE-009 Spec Seed 4 — eradicate the baked-in Go contract analyzer and
    re-implement contract verification as PACKS on the structural engines, with the
    GATE keeping only language-agnostic semantics. Today `pkg/gate/step_contract.go`
    extracts a Go API surface with `go/parser`, renders it with
    `formatFuncSignature`/`underlyingTypeString`, and compares declared-vs-actual via
    `signaturesMatch` whitespace-normalized STRING-EQUALITY — a brittle Go-source
    round-trip that is LITERALLY why backstop's own `contract_signature` gate step is
    red today. This spec DELETES that `go/parser` extraction, the rendering helpers,
    and the string-equality comparison outright, and splits contract verification
    three ways (DD-1 PACK/GATE/BINARY): (1) SIGNATURE PRESENCE becomes a pack-compiled
    ast-grep REQUIRED-pattern query — the contract stores a HUMAN-READABLE signature,
    a per-LANGUAGE PACK supplies a contract→ast-grep-pattern COMPILER (analogous to
    the pack's existing SARIF convert scripts), a match = SATISFIED / no-match =
    VIOLATION; (2) SYMBOL ABSENCE (ISSUE-013) becomes a pack-declared, allowlisted
    grep/ripgrep FORBIDDEN-pattern probe (`pattern-arg`, scope = file OR path), a
    match = PRESENT and the GATE inverts polarity (present-match IS the violation);
    (3) the GATE retains ONLY the language-agnostic match-verdict, the absence
    polarity inversion, and a shared "was-the-file-actually-scanned?" guard that
    preserves ISSUE-013's loud missing/UNSCANNED-scope error (REPLACING the old
    non-`.go`/missing config-error: a non-`.go` file is no longer an error, just another
    stack — REQ-004). The BACKSTOP BINARY NEVER
    compiles, renders, or understands a signature — doing so would be a P0
    zero-baked-language violation (explicitly flagged and rejected by the user). To
    feed the absence probe, this spec STANDS UP the grep engine the way SPEC-035
    mandates: (a) the traceability pack DECLARES grep in its `engines:` block
    (`pattern-arg` input mode + a grep-output→SARIF convert script) — NO baked
    `DefaultRegistry` entry, no ISSUE-027-style eradication debt; (b) the `grep`/`rg`
    tool is added to the backstop-owned trusted-tool allowlist
    (pkg/pack/engine/allowlist.go) so the engine clears the trust floor SPEC-035
    installed. To substantiate "beyond Go" (SD-3), this spec authors the TypeScript
    CONTRACT rules (signature presence via ast-grep on `.ts`, absence via grep) into
    the SHARED TS proof pack co-owned with SPEC-037 (Seed 3). The Go cutover is
    guarded by a strangler-equivalence pass (DD-9): the pack-produced signal is proven
    equivalent to the `go/parser` analyzer on REAL Go fixtures BEFORE `step_contract.go`
    is deleted. The new pack-fed contract step is WIRED into `buildContractStep` /
    `buildGateSteps` (cmd/backstop/gate.go) REPLACING the analyzer, verified with a
    spy so an unwired or still-using-old-analyzer path FAILS. End state: zero baked-in
    contract analyzer; only the gate-side polarity + file-scanned guard remain.
  package: pkg/gate

verification:
  level: integration
  test_command: go test ./pkg/gate/ ./cmd/backstop/ ./pkg/pack/ ./pkg/pack/engine/ -race -coverprofile=cover.out
  coverage_threshold: 80

requirements:
  - id: REQ-001
    text: >
      DELETE the baked Go contract analyzer in pkg/gate/step_contract.go: the
      `go/parser` symbol extraction (`probeSymbol`, `findFunction`, `findMethod`,
      `findType`, `findVariable`), the Go-source signature rendering
      (`formatFuncSignature`, `formatMethodSignature`, `underlyingTypeString`,
      `printFieldList`), and the `signaturesMatch`/`normalizeSignature`
      whitespace-normalized string-equality comparison MUST be removed. After this
      spec the backstop binary holds NO Go (or any-language) AST parsing, signature
      rendering, or string-equality signature comparison for contracts. The only
      contract logic that remains gate-side is the language-agnostic match-verdict,
      the absence polarity inversion (REQ-005), and the shared file-scanned guard
      (REQ-006). It is PROHIBITED for the binary to import `go/parser`, `go/ast`, or
      `go/printer` for contract verification, or to compile/render/understand a
      signature in any way.
    supports: stack-aware-traceability:REQ-004
    follows: STD-GO-001:GO-010
  - id: REQ-002
    text: >
      Re-implement contract SIGNATURE PRESENCE as a pack-compiled ast-grep
      REQUIRED-pattern query (OQ-8). The contract stores a HUMAN-READABLE signature
      (e.g. `func RouteFile(path string) []CheckType`); the LANGUAGE PACK supplies a
      contract→ast-grep-pattern COMPILER (a pack-relative script analogous to the
      pack's existing SARIF convert scripts) that transforms the declared signature
      into its language's ast-grep pattern, fed to the ast-grep engine via SPEC-035's
      `pattern-arg` input mode. The engine emits SARIF; the GATE consumes match/no-match
      with the verdict: a MATCH = contract SATISFIED, NO-MATCH = VIOLATION (the named
      symbol/signature is missing or mismatched). The BACKSTOP BINARY MUST NOT compile,
      render, or understand the signature — it passes the declared signature through to
      the pack-declared engine and reads back SARIF. Compiling/rendering a signature in
      the binary is a P0 zero-baked-language violation and is PROHIBITED.
    supports: stack-aware-traceability:REQ-004
    follows: STD-GO-001:GO-010
  - id: REQ-003
    text: >
      Re-implement contract SYMBOL ABSENCE (the ISSUE-013 assert-absence policy) as a
      pack-declared, allowlisted grep/ripgrep FORBIDDEN-pattern probe (OQ-7), fed via
      SPEC-035's `pattern-arg` input mode, with SCOPE taken from the declared contract
      as a PARAMETER — a file OR a path (scope-as-parameter, not a fork). The engine
      emits SARIF natively: a MATCH = symbol PRESENT, an EMPTY result = symbol ABSENT.
      The GATE INVERTS polarity: a present-match IS the violation ("symbol X must be
      absent, found at L:C"); an empty result with the file confirmed scanned PASSES.
      Absence MUST use GREP (text-presence), NOT ast-grep (structural): grep is
      language-agnostic and catches lingering references in comments / strings /
      non-parsed files, the conservative failure direction absence requires. The grep
      probe MUST be a PACK-DECLARED engine, NOT a grep baked into the binary
      (PROHIBITED — that would be baked tool knowledge).
    supports: stack-aware-traceability:REQ-005
    follows: STD-GO-001:GO-010
  - id: REQ-004
    text: >
      Preserve ISSUE-013's loud-on-empty guarantee with a thin, language-agnostic
      gate-side "was-the-file-actually-scanned?" guard, REPLACING the old
      `step_contract.go` non-`.go`/missing-file config-error behavior. Today the deleted
      analyzer loudly errors on an absence contract whose target is MISSING **or
      non-`.go`** (ISSUE-013's "absence is an AST/source probe and only .go files are
      checkable"). Because contracts are now language-agnostic (signature presence via
      ast-grep, absence via grep, across stacks — REQ-002/REQ-003/REQ-007), the
      "non-`.go` is an error" clause is DISSOLVED: a non-`.go` file is NOT an error, it
      is simply another stack that the pack-declared engine scans. The REPLACEMENT guard
      keys ONLY on whether the declared scope was actually scanned. An absence contract
      whose declared file/path is MISSING, or whose declared scope produced NO engine
      scan record (could not be probed), MUST be a LOUD config error (severity error,
      blocking — exit 2), NEVER a silent pass — because an empty grep result is otherwise
      indistinguishable from empty-because-not-scanned, and treating
      empty-because-not-scanned as ABSENT would be vacuous green for an absence assertion.
      It is PROHIBITED for the guard to treat a non-`.go` (non-Go) target as a config
      error on the basis of its extension/language, or to parse, AST-walk, or know any
      language (it asserts only that the engine produced a scan record for the declared
      scope). The file-extension/language is NOT a config-error axis; scanned-vs-unscanned
      is the only axis.
    supports: stack-aware-traceability:REQ-005
    follows: STD-GO-001:GO-010
  - id: REQ-005
    text: >
      Stand up the grep/ripgrep engine WITHOUT baking tool knowledge into the binary
      (SD-1 / bundle REQ-006), in the SPEC-035-mandated two-part split: (a) the
      traceability pack DECLARES grep in its `engines:` block — `pattern-arg` input
      mode, a grep-output→SARIF convert script (pack-relative, analogous to ast-grep's
      `to-sarif.sh`), and `gate_type: contracts` — so there is NO baked
      engine.DefaultRegistry entry for grep and NO ISSUE-027-style eradication debt;
      (b) the `grep` and `rg` tools are added to the backstop-owned trusted-tool
      allowlist (engine.TrustedToolAllowlist in pkg/pack/engine/allowlist.go) at a
      pinned version, so a pack-declared grep command clears SPEC-035's trust floor
      (CheckToolAllowed). A grep command MUST NOT run if `grep`/`rg` is absent from the
      allowlist — the existing un-allowlisted-tool fail-loud (exit 2) applies
      unchanged. Backstop learns no grep flags or output format; it runs the
      pack-declared command and consumes the converted SARIF.
    supports: stack-aware-traceability:REQ-006
    follows: STD-GO-001:GO-010
  - id: REQ-006
    text: >
      WIRE the pack-fed contract check into the gate IN FRONT OF / REPLACING the
      analyzer. `buildContractStep` / `buildGateSteps` (cmd/backstop/gate.go) MUST
      construct the contract step from the pack-produced SARIF path (the contracts
      gate-type engine results) plus the gate-side polarity + file-scanned guard, and
      MUST NOT call the deleted `go/parser` analyzer (`StepContractSignatureScopedFunc`
      in its current `probeSymbol` form). The wiring MUST be verifiable: a spy/sentinel
      contract source asserts the contract step consumes pack-produced SARIF and that
      the deleted analyzer entrypoint is no longer reachable, so an UNWIRED path, or one
      still routed to the old analyzer, FAILS the test. cmd/backstop/ is in the
      verification test_command so the wiring is exercised end-to-end, not assumed.
    supports: stack-aware-traceability:REQ-004
    follows: STD-GO-001:GO-010
  - id: REQ-007
    text: >
      Author the TypeScript CONTRACT rules into the SHARED TypeScript proof pack
      (co-owned with SPEC-037 / Seed 3; this spec ADDS the contract rules, it does not
      create a second TS pack): signature PRESENCE via ast-grep on `.ts` (the
      contract→pattern compiler in its TS form) and symbol ABSENCE via the new grep
      engine on `.ts`. The TS rules MUST ride the STRUCTURAL engines only (ast-grep /
      grep), NOT the TS language toolchain (eslint / tsc). Verification MUST run REAL
      ast-grep (signature) and REAL grep (absence) over REAL `.ts` fixtures asserting
      concrete present/absent/mismatch verdicts — a stub MUST NOT satisfy it. This
      substantiates the "beyond Go" claim (a pack is stack-locked, so a second
      stack-locked pack is the proof) and begins unblocking the TS runtime's self-gating.
    supports: stack-aware-traceability:REQ-007
    follows: STD-GO-001:GO-010
  - id: REQ-008
    text: >
      Guard the Go contract cutover with a strangler-equivalence pass (DD-9 / bundle
      REQ-008): on REAL Go fixtures, the pack-produced contract signal (ast-grep
      signature presence + grep absence probe) MUST be proven to reproduce the
      `go/parser` analyzer's verdicts — including ISSUE-013's absence cases (present →
      violation, absent → pass, missing/non-target file → loud config error) — BEFORE
      `step_contract.go`'s analyzer is deleted. The equivalence harness MUST run REAL
      ast-grep and REAL grep (no stub) and assert matching match/no-match/loud-error
      verdicts across the present, mismatched, and absent cases. The language-agnostic
      COMPARISON logic (match-verdict + absence polarity + file-scanned guard) stays
      gate-side and is the shared consumer in both the legacy-equivalence and the
      pack-only paths.
    supports: stack-aware-traceability:REQ-004
    follows: STD-GO-001:GO-010
  - id: REQ-009
    text: >
      No vacuous green on deletion. The pack-fed contract path MUST produce a REAL,
      blocking violation in each enforcement case so eradication does not trade
      enforcement for silence: (a) a missing/mismatched signature (no ast-grep match)
      yields a contract-violation (exit 2); (b) a present forbidden symbol (grep match
      under an absence contract) yields an absence-violation (exit 2); (c) the
      file-scanned guard yields a loud config error (exit 2) when the declared scope was
      not scanned. Additionally, running the new pack path against backstop's OWN
      previously-red `contract_signature` cases MUST resolve them to green (the
      dual-substrate dogfood payoff: the brittle `signaturesMatch`/`formatFuncSignature`
      round-trip that reds the gate today is dissolved).
    supports: stack-aware-traceability:REQ-004
    follows: STD-GO-001:GO-010
  - id: REQ-010
    text: >
      DELETE-OR-MIGRATE every existing test coupled to the deleted analyzer symbols, so
      pkg/gate still compiles and its still-valid behavior is preserved through the pack
      path (per "align predating artifacts"). The analyzer-coupled tests in pkg/gate that
      reference the deleted symbols (`StepContractSignatureFunc`,
      `StepContractSignatureScopedFunc` in its `probeSymbol` form, `probeSymbol`,
      `findFunction`/`findMethod`/`findType`/`findVariable`, `formatFuncSignature`/
      `formatMethodSignature`/`underlyingTypeString`/`printFieldList`, `signaturesMatch`/
      `normalizeSignature`) — at minimum `step_contract_test.go`,
      `step_contract_absence_test.go`, `step_contract_absence_config_test.go`,
      `step_contract_noregress_test.go`, and `step_contract_parser_absence_test.go` — MUST
      be deleted or rewritten against the new pack-SARIF consumer
      (`VerifyContractVerdict` / `ContractEngineResult`). Tests asserting behavior that
      SURVIVES (absence present→violation / absent→pass, signature present→satisfied /
      mismatch→violation, missing-scope→loud config error) MUST be MIGRATED to the pack
      path so the guarantee is still covered; tests asserting ONLY deleted internals
      (Go-source rendering, string-equality normalization, non-`.go`-is-an-error per the
      DISSOLVED REQ-004 clause) MUST be deleted. After this spec NO test in pkg/gate
      references any deleted symbol, and `go test ./pkg/gate/` compiles and passes.
    supports: stack-aware-traceability:REQ-004
    follows: STD-GO-001:GO-010
  - id: REQ-011
    text: >
      DELETE the non-scoped `StepContractSignatureFunc(contracts []ContractEntry)`
      entrypoint. It is a thin go/parser-era convenience wrapper that exists only to call
      `StepContractSignatureScopedFunc(contracts, nil)` (an unscoped probe of the deleted
      analyzer); with the analyzer deleted and `buildContractStep` routing through the
      scoped pack-SARIF consumer, it has no live caller outside tests (which REQ-010
      removes). It MUST be deleted, not retained as a wrapper, so the deletion surface is
      complete and no dead unscoped analyzer entrypoint remains. It is PROHIBITED to keep
      `StepContractSignatureFunc` as a compatibility shim. The scoped
      `StepContractSignatureScopedFunc` is the sole retained contract entrypoint (rewritten
      per REQ-001 to consume pack SARIF).
    supports: stack-aware-traceability:REQ-004
    follows: STD-GO-001:GO-010
  - id: REQ-012
    text: >
      EXTEND contract-entry extraction so the declared absence SCOPE reaches the grep
      probe. `ExtractContractEntries(specDir, projectRoot)` (pkg/gate/step_testverify.go)
      MUST populate the new `ContractEntry.Scope` field from the declared contract's
      provides entry (the file-OR-path absence parameter, REQ-003/CLM-010), in addition to
      the existing File/Name/Kind/Signature/Absent fields. The spec frontmatter
      `contracts[].provides[]` parsing (`parseSpecFrontmatter`) MUST carry the declared
      scope through to the `ContractEntry.Scope` field so a path-scoped absence probe
      receives its scope. Without this, a path-scoped absence probe (CLM-010) has no scope
      to pass through `pattern-arg` and is unreachable. The extraction MUST remain a pure
      data-record builder — it MUST NOT parse, AST-walk, or compile a signature (that
      stays in the pack); it only reads declared fields and fills the record fed to the
      pack engine.
    supports: stack-aware-traceability:REQ-005
    follows: STD-GO-001:GO-010

claims:
  # REQ-001 — delete the baked Go contract analyzer
  - id: CLM-001
    requirement: REQ-001
    text: After this spec, pkg/gate/step_contract.go contains no go/parser symbol extraction (probeSymbol/findFunction/findMethod/findType/findVariable are removed) and the package does not import go/parser, go/ast, or go/printer for contract verification
    tests:
      - TestContract_NoGoParserExtractionRemains
  - id: CLM-002
    requirement: REQ-001
    text: The Go-source signature rendering helpers (formatFuncSignature, formatMethodSignature, underlyingTypeString, printFieldList) are deleted and no longer referenced anywhere in pkg/gate
    tests:
      - TestContract_SignatureRenderingHelpersDeleted
  - id: CLM-003
    requirement: REQ-001
    text: The whitespace-normalized string-equality comparison (signaturesMatch/normalizeSignature) is deleted; contract satisfaction is no longer decided by string equality on rendered signatures
    tests:
      - TestContract_StringEqualityComparisonDeleted

  # REQ-002 — signature presence as pack-compiled ast-grep required-pattern query
  - id: CLM-004
    requirement: REQ-002
    text: A contract whose declared signature is PRESENT in the source produces an ast-grep MATCH, which the gate verdicts as SATISFIED (no violation) — proven with REAL ast-grep over a real Go fixture
    tests:
      - TestContract_SignaturePresentAstGrepMatchSatisfied
  - id: CLM-005
    requirement: REQ-002
    text: A contract whose declared signature is ABSENT or MISMATCHED produces NO ast-grep match, which the gate verdicts as a blocking contract VIOLATION — proven with REAL ast-grep over a real Go fixture
    tests:
      - TestContract_SignatureMissingAstGrepNoMatchViolation
  - id: CLM-006
    requirement: REQ-002
    text: The contract→ast-grep-pattern COMPILER lives in the pack (a pack-relative script), not in the backstop binary; the binary passes the human-readable signature through pattern-arg and never compiles or renders it
    tests:
      - TestContract_PatternCompilerLivesInPackNotBinary
  - id: CLM-007
    requirement: REQ-002
    text: A parameter-name-only / whitespace difference in the source (same param types and return) still MATCHES — the ast-grep structural pattern is name/whitespace-insensitive where the deleted string-equality was not — proven with REAL ast-grep
    tests:
      - TestContract_SignatureStructuralMatchIgnoresParamNames

  # REQ-003 — absence as pack-declared grep forbidden-pattern probe
  - id: CLM-008
    requirement: REQ-003
    text: An absence contract whose forbidden symbol is PRESENT in the scoped file produces a grep MATCH, which the gate inverts to a blocking absence VIOLATION naming the file and location — proven with REAL grep over a real fixture
    tests:
      - TestContract_AbsencePresentSymbolGrepMatchViolation
  - id: CLM-009
    requirement: REQ-003
    text: An absence contract whose forbidden symbol is genuinely ABSENT from the scoped (and confirmed-scanned) file produces an EMPTY grep result, which the gate verdicts as PASS — proven with REAL grep over a real fixture
    tests:
      - TestContract_AbsenceAbsentSymbolEmptyResultPasses
  - id: CLM-010
    requirement: REQ-003
    text: Absence scope is a PARAMETER — a single file OR a path — taken from the declared contract and passed through pattern-arg; both file-scoped and path-scoped absence probes run via the same engine binding without a code fork
    tests:
      - TestContract_AbsenceScopeFileOrPathParameterized
  - id: CLM-011
    requirement: REQ-003
    text: Absence uses the GREP engine (text-presence), not ast-grep — the absence rule binds the pack-declared grep engine, and a grep match on a token appearing in a comment/string is flagged (the conservative text-presence direction)
    tests:
      - TestContract_AbsenceUsesGrepTextPresenceNotAstGrep

  # REQ-004 — file-scanned guard preserves ISSUE-013 loud-on-empty
  - id: CLM-012
    requirement: REQ-004
    text: An absence contract whose declared file is MISSING is a LOUD config error (severity error, blocking exit 2), never a silent pass — empty-because-not-there is not treated as absent
    tests:
      - TestContract_AbsenceMissingFileLoudConfigError
  - id: CLM-013
    requirement: REQ-004
    text: An absence contract whose declared scope was NOT scanned by the engine (no scan record) is a LOUD config error, never a silent pass — empty-because-not-scanned is distinguished from absent
    tests:
      - TestContract_AbsenceUnscannedScopeLoudConfigError
  - id: CLM-014
    requirement: REQ-004
    text: The file-scanned guard is language-agnostic — it asserts a scan record exists for the declared scope and does NOT itself parse, AST-walk, or import any language package
    tests:
      - TestContract_FileScannedGuardIsLanguageAgnostic
  - id: CLM-034
    requirement: REQ-004
    text: An absence contract targeting a NON-`.go` (non-Go) file that WAS scanned by the engine is NOT a config error — the dissolved "non-Go is an error" clause no longer fires; a non-Go scanned scope produces a normal absence verdict (present→violation, absent→pass), proving extension/language is not a config-error axis
    tests:
      - TestContract_AbsenceNonGoScannedScopeIsNotConfigError

  # REQ-005 — grep engine stood up pack-declared + allowlisted
  - id: CLM-015
    requirement: REQ-005
    text: The traceability pack declares grep in its engines block with pattern-arg input mode, a grep-output→SARIF convert script, and gate_type contracts; there is NO baked engine.DefaultRegistry entry for grep
    tests:
      - TestEngine_GrepPackDeclaredNotInDefaultRegistry
  - id: CLM-016
    requirement: REQ-005
    text: grep and rg are present on the backstop-owned trusted-tool allowlist (engine.TrustedToolAllowlist) at a pinned version
    tests:
      - TestAllowlist_GrepAndRgPresentPinned
  - id: CLM-017
    requirement: REQ-005
    text: A pack-declared grep command runs through dispatch when grep is on the allowlist and lock-pinned (the trust gate passes) — proven by exercising CheckToolAllowed via the engine's tool
    tests:
      - TestAllowlist_GrepAllowlistedPinnedRuns
  - id: CLM-018
    requirement: REQ-005
    text: A pack-declared grep command is NOT run if grep/rg is removed from the allowlist — it produces the existing loud un-allowlisted-tool ConfigError (exit 2) naming the tool and pack, so the engine cannot run before it is allowlisted
    tests:
      - TestAllowlist_GrepUnallowlistedFailsLoud
  - id: CLM-019
    requirement: REQ-005
    text: The grep-output→SARIF convert script transforms a real grep match into valid SARIF (a result with a physicalLocation) consumed by the gate — proven by converting real grep output, not a hand-written SARIF stub
    tests:
      - TestEngine_GrepConvertScriptEmitsValidSarif

  # REQ-006 — wiring verified, not assumed
  - id: CLM-020
    requirement: REQ-006
    text: buildContractStep / buildGateSteps construct the contract step from the pack-produced contracts-gate-type SARIF path plus the gate-side polarity + file-scanned guard, and a spy contract source confirms that path is the one consumed
    tests:
      - TestGate_ContractStepConsumesPackSarifPath
  - id: CLM-021
    requirement: REQ-006
    text: The gate no longer routes the contract step to the deleted go/parser analyzer entrypoint — a sentinel asserts the old probeSymbol-based path is unreachable from buildContractStep, so a still-using-old-analyzer wiring FAILS
    tests:
      - TestGate_ContractStepDoesNotCallDeletedAnalyzer
  - id: CLM-022
    requirement: REQ-006
    text: An UNWIRED contract step (pack SARIF present but gate not consuming it) FAILS the wiring test — the spy detects the pack-produced contract violation is dropped rather than surfaced
    tests:
      - TestGate_UnwiredContractStepFails

  # REQ-007 — TypeScript proof pack contract rules (real fixtures)
  - id: CLM-023
    requirement: REQ-007
    text: The TS contract signature-presence rule MATCHES a present .ts signature via REAL ast-grep over a real .ts fixture (verdict SATISFIED) and yields a VIOLATION when the .ts signature is absent/mismatched
    tests:
      - TestTSPack_ContractSignaturePresenceAstGrep
  - id: CLM-024
    requirement: REQ-007
    text: The TS contract absence rule flags a PRESENT forbidden symbol in a .ts file via REAL grep (violation) and PASSES when the symbol is genuinely absent — real .ts fixtures, no stub
    tests:
      - TestTSPack_ContractAbsenceGrep
  - id: CLM-025
    requirement: REQ-007
    text: The TS contract rules ride the structural engines only — the TS pack declares ast-grep and grep for contracts and binds NO TS toolchain engine (eslint/tsc) for the contract rules
    tests:
      - TestTSPack_ContractRulesUseStructuralEnginesOnly
  - id: CLM-026
    requirement: REQ-007
    text: The TS contract rules are added to the SAME shared TS proof pack that holds Seed 3's substantiveness rules (one stack-locked TS pack), not a second TS pack
    tests:
      - TestTSPack_ContractRulesShareSubstantivenessPack

  # REQ-008 — strangler-equivalence before deletion
  - id: CLM-027
    requirement: REQ-008
    text: On a real Go fixture with a PRESENT matching signature, the pack-produced ast-grep verdict (SATISFIED) equals the go/parser analyzer's verdict (match) — proven with REAL ast-grep, not a stub
    tests:
      - TestEquivalence_GoSignaturePresentMatchesLegacy
  - id: CLM-028
    requirement: REQ-008
    text: On a real Go fixture with a MISMATCHED/missing signature, the pack-produced ast-grep verdict (VIOLATION) equals the go/parser analyzer's verdict (mismatch/not-found) — proven with REAL ast-grep
    tests:
      - TestEquivalence_GoSignatureMismatchMatchesLegacy
  - id: CLM-029
    requirement: REQ-008
    text: On a real Go fixture, the pack grep absence probe reproduces ISSUE-013's absence verdicts — present→violation, absent→pass — equal to the go/parser probeSymbol verdicts, proven with REAL grep
    tests:
      - TestEquivalence_GoAbsencePresentAndAbsentMatchLegacy
  - id: CLM-030
    requirement: REQ-008
    text: On a missing/non-target Go file, the pack path + file-scanned guard reproduces the analyzer's loud config error (not a silent pass), matching ISSUE-013's missing-file behavior
    tests:
      - TestEquivalence_GoAbsenceMissingFileMatchesLegacyLoudError

  # REQ-009 — no vacuous green on deletion (incl. dogfood)
  - id: CLM-031
    requirement: REQ-009
    text: A missing/mismatched signature through the pack path yields a real blocking contract violation (exit 2) — deletion does not produce a silent pass
    tests:
      - TestNoVacuousGreen_MissingSignatureBlocks
  - id: CLM-032
    requirement: REQ-009
    text: A present forbidden symbol through the pack grep absence path yields a real blocking absence violation (exit 2) — deletion does not produce a silent pass
    tests:
      - TestNoVacuousGreen_PresentForbiddenSymbolBlocks
  - id: CLM-033
    requirement: REQ-009
    text: Running the new pack contract path against backstop's OWN previously-red contract_signature cases resolves them to green (the dual-substrate dogfood payoff — the signaturesMatch/formatFuncSignature round-trip is dissolved)
    tests:
      - TestDogfood_BackstopOwnContractSignatureTurnsGreen

  # REQ-010 — delete-or-migrate analyzer-coupled tests
  - id: CLM-035
    requirement: REQ-010
    text: After this spec, NO test file in pkg/gate references any deleted analyzer symbol (StepContractSignatureFunc, probeSymbol, findFunction/findMethod/findType/findVariable, formatFuncSignature/formatMethodSignature/underlyingTypeString/printFieldList, signaturesMatch/normalizeSignature) — a grep over pkg/gate/*_test.go for those symbols returns nothing, and go test ./pkg/gate/ compiles
    tests:
      - TestContract_NoDeletedSymbolReferencedInGateTests
  - id: CLM-036
    requirement: REQ-010
    text: The SURVIVING absence behavior previously covered by step_contract_absence_test.go / step_contract_absence_config_test.go (present→violation, absent→pass, missing/unscanned scope→loud config error) is MIGRATED to and still covered against the pack-SARIF consumer (VerifyContractVerdict / ContractEngineResult), not dropped
    tests:
      - TestContract_MigratedAbsenceBehaviorStillCovered
  - id: CLM-037
    requirement: REQ-010
    text: Tests asserting ONLY deleted internals (Go-source signature rendering, string-equality normalization, and the dissolved non-`.go`-is-an-error clause) are DELETED — they do not survive as dead or skipped tests referencing removed symbols
    tests:
      - TestContract_DeletedInternalOnlyTestsRemoved

  # REQ-011 — delete the non-scoped StepContractSignatureFunc entrypoint
  - id: CLM-038
    requirement: REQ-011
    text: The non-scoped StepContractSignatureFunc(contracts []ContractEntry) entrypoint is DELETED from pkg/gate (no longer defined or exported) — a build/symbol check confirms it is absent and StepContractSignatureScopedFunc is the sole retained contract entrypoint
    tests:
      - TestContract_NonScopedEntrypointDeleted
  - id: CLM-039
    requirement: REQ-011
    text: No non-test caller of StepContractSignatureFunc remains (buildContractStep routes through the scoped pack-SARIF consumer), so deleting the unscoped wrapper leaves no dangling reference and is not retained as a compatibility shim
    tests:
      - TestContract_NonScopedEntrypointHasNoCaller

  # REQ-012 — ExtractContractEntries carries Scope to the grep probe
  - id: CLM-040
    requirement: REQ-012
    text: ExtractContractEntries populates ContractEntry.Scope from the declared contract's provides entry (file OR path), in addition to File/Name/Kind/Signature/Absent — a spec fixture declaring a path-scoped absence yields a ContractEntry whose Scope equals the declared path
    tests:
      - TestExtract_ContractEntryScopePopulatedFromDeclaration
  - id: CLM-041
    requirement: REQ-012
    text: A path-scoped absence ContractEntry produced by ExtractContractEntries reaches the grep probe with its declared scope intact — the Scope value flows through pattern-arg to the engine, making the path-scoped absence probe (CLM-010) buildable end to end
    tests:
      - TestExtract_PathScopedAbsenceScopeReachesGrepProbe
  - id: CLM-042
    requirement: REQ-012
    text: ExtractContractEntries remains a pure data-record builder — it does NOT parse, AST-walk, or compile a signature; it only reads declared frontmatter fields (a contract with a signature is carried through unmodified, not rendered)
    tests:
      - TestExtract_ContractEntryExtractionDoesNotParseOrCompile

contracts:
  - file: pkg/gate/step_contract.go
    provides:
      - name: ContractEntry
        kind: type
        signature: "type ContractEntry struct { File string; Name string; Kind string; Signature string; Scope string; Absent bool }"
        notes: "RETAINED as the declared-contract record (File/Name/Kind/Signature/Absent unchanged in meaning) and EXTENDED with Scope (the absence file-OR-path parameter, REQ-003/CLM-010). The struct is now a pure data record fed to the pack engine via pattern-arg; it no longer drives a go/parser probe. Signature is passed through to the pack's contract→ast-grep-pattern compiler unmodified (REQ-002/CLM-006)."
      - name: StepContractSignatureScopedFunc
        kind: function
        signature: "func StepContractSignatureScopedFunc(results []ContractEngineResult, contracts []ContractEntry, scope *GateScope) StepFunc"
        notes: "REWRITTEN and the SOLE retained contract entrypoint: consumes pack-produced SARIF (ContractEngineResult) instead of running go/parser. Applies the language-agnostic verdict — ast-grep match = SATISFIED / no-match = VIOLATION (REQ-002), grep present-match = absence VIOLATION (REQ-003) — plus the file-scanned guard (REQ-004). It NO LONGER imports go/parser/go/ast/go/printer (REQ-001/CLM-001). The deleted probeSymbol/findFunction/findMethod/findType/findVariable/formatFuncSignature/formatMethodSignature/underlyingTypeString/printFieldList/signaturesMatch/normalizeSignature symbols are removed (CLM-001/CLM-002/CLM-003). The non-scoped convenience wrapper StepContractSignatureFunc is DELETED (REQ-011/CLM-038/CLM-039) — not retained as a shim."
      - name: StepContractSignatureFunc
        kind: function
        signature: "func StepContractSignatureFunc(contracts []ContractEntry) StepFunc"
        absent: true
        notes: "DELETED (REQ-011/CLM-038): the non-scoped go/parser-era wrapper that only called StepContractSignatureScopedFunc(contracts, nil). It has no live non-test caller (buildContractStep uses the scoped pack-SARIF consumer); declared absent so its reappearance is a deletion regression."
      - name: ContractEngineResult
        kind: type
        signature: "type ContractEngineResult struct { Entry ContractEntry; Matched bool; Scanned bool; Locations []SarifLocation }"
        notes: "NEW: the gate-side, language-agnostic carrier of one pack engine probe result for one contract entry — Matched (did the ast-grep/grep query match), Scanned (the file-scanned guard signal, REQ-004/CLM-013), Locations (for the violation message). The gate verdicts purely off these fields; it never re-parses source."
      - name: VerifyContractVerdict
        kind: function
        signature: "func VerifyContractVerdict(r ContractEngineResult) (Violation, bool)"
        notes: "NEW: the pure, language-agnostic verdict function — present-contract no-match → violation; absence-contract match → violation; unscanned scope → loud config-error violation; otherwise no violation. The single source of the polarity/match-verdict logic that stays gate-side (REQ-002/REQ-003/REQ-004). Has zero language imports (CLM-014)."
    consumes:
      - source: pkg/pack/engine
        name: GateTypeContracts
        kind: constant
      - source: pkg/pack/engine
        name: InputModePatternArg
        kind: constant
  - file: pkg/gate/step_testverify.go
    provides:
      - name: ExtractContractEntries
        kind: function
        signature: "func ExtractContractEntries(specDir, projectRoot string) ([]ContractEntry, error)"
        notes: "EXTENDED (REQ-012/CLM-040/CLM-041/CLM-042): now populates the new ContractEntry.Scope field from the declared contract's provides entry (the file-OR-path absence parameter) in addition to File/Name/Kind/Signature/Absent, so a path-scoped absence probe (CLM-010) receives its scope through pattern-arg. Remains a pure data-record builder — it reads declared frontmatter fields only; it does NOT parse, AST-walk, or compile a signature (that work moves to the pack)."
    consumes: []
  - file: pkg/pack/engine/allowlist.go
    provides:
      - name: TrustedToolAllowlist
        kind: function
        signature: "func TrustedToolAllowlist() map[string]string"
        notes: "EXTENDED (not rewritten): adds `grep` and `rg` at a pinned version to the existing {tool → pinned version} map alongside semgrep/ast-grep (REQ-005/CLM-016). No other change; CheckToolAllowed continues to gate every pack-declared command, so a pack grep command clears the trust floor only because grep/rg are now on the list (CLM-017/CLM-018)."
    consumes: []
  - file: cmd/backstop/gate.go
    provides:
      - name: buildContractStep
        kind: function
        signature: "func buildContractStep(specDir, projectRoot string, scope *gate.GateScope) gate.StepFunc"
        notes: "REWIRED: extracts ContractEntry records from specs and routes them to the pack-produced contracts-gate-type SARIF path (the grep absence + ast-grep signature engines), then to gate.StepContractSignatureScopedFunc's rewritten pack-SARIF consumer. It MUST NOT route to the deleted go/parser analyzer (REQ-006/CLM-020/CLM-021). The contract source is injected via a seam so a spy can assert the pack path is consumed and the old analyzer is unreachable (CLM-020/CLM-021/CLM-022)."
    consumes:
      - source: pkg/gate
        name: StepContractSignatureScopedFunc
        kind: function
      - source: pkg/gate
        name: ContractEngineResult
        kind: type
---

# SPEC-038: Traceability Contracts Pack

## Overview

The gate's contract step (step 6, `contract_signature`) is the last and highest-risk
of BUNDLE-009's three baked-in Go traceability analyzers. Today
`pkg/gate/step_contract.go` parses the target file with `go/parser`, walks the AST to
extract the declared symbol, RENDERS it back to a Go-source string with
`formatFuncSignature`/`underlyingTypeString`, and decides satisfaction by
whitespace-normalized STRING-EQUALITY (`signaturesMatch`). That round-trip — Go source
→ AST → re-rendered Go source → string compare — is brittle (`[]byte` vs `[]uint8`,
named vs unnamed results, receiver formatting) and is **literally why backstop's own
`contract_signature` gate step is red today**. ISSUE-013 layered an absence assertion
on top, also via a gate-side `parser.ParseFile` + `probeSymbol`.

This spec (BUNDLE-009 Spec Seed 4) **deletes that analyzer outright** and re-implements
contract verification on the structural engines, following the bundle's one
architecture (DD-1, PACK / GATE / BINARY):

- **PACK** does the language-specific work and emits SARIF: a per-language
  contract→ast-grep-pattern **compiler** (signature presence, a REQUIRED pattern) and a
  grep **forbidden-pattern probe** (symbol absence). Both ride SPEC-035's `pattern-arg`
  input mode.
- **GATE** keeps only language-agnostic semantics: the **match-verdict** (ast-grep match
  = satisfied, no-match = violation), the **absence polarity inversion** (grep
  present-match = violation), and a shared **file-scanned guard** (preserving ISSUE-013's
  loud missing/UNSCANNED-scope error — REPLACING the old non-`.go`/missing config-error;
  a non-`.go` file is no longer an error, just another stack — REQ-004).
- **BINARY** knows zero language/tool specifics — it NEVER compiles, renders, or
  understands a signature (a P0 zero-baked-language violation, explicitly rejected).

To feed the absence probe, this spec also **stands up the grep engine** (which does not
exist yet — SPEC-035's registry holds semgrep/ast-grep/sandbox/config-file/golangci/
go-build/go-test, and the allowlist holds only semgrep+ast-grep): the engine is
**pack-declared** in the traceability pack's `engines:` block (no baked
`DefaultRegistry` entry), and `grep`/`rg` are added to the backstop-owned trusted-tool
allowlist. To substantiate "beyond Go" (SD-3), the spec authors the **TypeScript contract
rules** into the shared TS proof pack. The Go cutover is guarded by a
**strangler-equivalence pass** on real Go fixtures before deletion. Implementing this
spec turns backstop's own gate green on itself — the dual-substrate payoff.

## Requirements

Requirements are defined in the frontmatter `requirements[]` array (REQ-001 … REQ-009),
each tracing to BUNDLE-009 via `supports`. Summary of the rule surface:

| REQ | Rule | Allowed (pass) | Prohibited (fail / loud) |
|-----|------|----------------|--------------------------|
| REQ-001 | Delete the Go analyzer | gate keeps only polarity + match-verdict + file-scanned guard | binary importing `go/parser`/`go/ast`/`go/printer` for contracts; string-equality signature compare; rendering a signature |
| REQ-002 | Signature presence = pack-compiled ast-grep REQUIRED pattern | ast-grep match → SATISFIED | no-match → VIOLATION; binary compiling/rendering a signature (P0) |
| REQ-003 | Absence = pack-declared grep FORBIDDEN-pattern probe (`pattern-arg`, file OR path) | empty result + scanned → PASS | grep match → absence VIOLATION; using ast-grep for absence; grep baked in the binary |
| REQ-004 | File-scanned guard (REPLACES non-`.go`/missing config-error) | scope confirmed scanned, empty → PASS; a SCANNED non-`.go` scope → normal verdict (NOT an error) | missing file / unscanned scope → LOUD config error (exit 2); guard erroring on a file's extension/language; the dissolved "non-`.go` is an error" clause firing |
| REQ-005 | Stand up grep engine | pack `engines:` block (`pattern-arg` + grep→SARIF) AND `grep`/`rg` allowlisted | baked `DefaultRegistry` grep entry; running grep before it is allowlisted |
| REQ-006 | Wire pack path into the gate | `buildContractStep` consumes pack SARIF | routing to the deleted analyzer; unwired path dropping the pack violation |
| REQ-007 | TS contract rules in shared TS proof pack | ast-grep (presence) + grep (absence) on `.ts` | binding a TS toolchain engine (eslint/tsc) for contract rules; a second TS pack; stub fixtures |
| REQ-008 | Strangler-equivalence before deletion | pack verdict == `go/parser` verdict on real fixtures (real ast-grep + real grep) | deleting `step_contract.go` before equivalence is proven; stubbed equivalence |
| REQ-009 | No vacuous green on deletion | real blocking violation in each enforcement case; own contract_signature turns green | any enforcement case silently passing post-deletion |
| REQ-010 | Delete-or-migrate analyzer-coupled tests | surviving behavior migrated to the pack-SARIF consumer | a test referencing any deleted symbol surviving; pkg/gate failing to compile; dropping a surviving guarantee |
| REQ-011 | Delete non-scoped `StepContractSignatureFunc` | scoped `StepContractSignatureScopedFunc` is the sole entrypoint | retaining the unscoped wrapper as a shim; a dead unscoped analyzer entrypoint remaining |
| REQ-012 | `ExtractContractEntries` carries `Scope` | declared file-OR-path scope reaches the grep probe via `pattern-arg` | path-scoped absence (CLM-010) unreachable for lack of scope; extraction parsing/compiling a signature |

## Implementation

The work proceeds in passes (the planner maps tasks to these). The strangler order is
load-bearing: **the analyzer is not deleted until equivalence is proven** (REQ-008).

1. **Stand up the grep engine (REQ-005).**
   - Extend `engine.TrustedToolAllowlist` (pkg/pack/engine/allowlist.go) to add `grep`
     and `rg` at a pinned version. No `DefaultRegistry` entry is added.
   - In the traceability pack, declare grep in the `engines:` block: `command` (the grep
     invocation prefix), `input_mode: pattern-arg`, `input_flag`, a pack-relative
     grep-output→SARIF `convert` script (analogous to `ast-grep/to-sarif.sh`), and
     `gate_type: contracts`. The grep convert script transforms grep's match lines into
     SARIF results with `physicalLocation`s.
2. **Gate-side language-agnostic verdict + scope extraction (REQ-002/REQ-003/REQ-004/REQ-012).**
   - Add `ContractEngineResult` (the per-entry pack-probe carrier: `Matched`, `Scanned`,
     `Locations`) and the pure `VerifyContractVerdict` function in pkg/gate. This holds
     the ONLY contract logic that stays gate-side: ast-grep no-match → violation;
     absence grep match → violation; unscanned scope → loud config error; else pass. The
     unscanned-scope branch REPLACES the old non-`.go`/missing config-error path
     (REQ-004): the verdict keys on scanned-vs-unscanned only — a SCANNED non-`.go` scope
     gets a normal verdict, never an extension-based error (CLM-034).
   - It imports no language package (CLM-014).
   - Extend `ContractEntry` with the `Scope` field and populate it in
     `ExtractContractEntries` (pkg/gate/step_testverify.go) from the declared contract's
     provides entry, so the declared file-OR-path absence scope reaches the grep probe
     via `pattern-arg` (REQ-012/CLM-040/CLM-041). The extraction stays a pure data-record
     builder — no parse, no AST-walk, no signature compilation (CLM-042).
3. **Pack signature/absence rules (REQ-002/REQ-003).**
   - Author the Go contract signature rule: a pack-relative contract→ast-grep-pattern
     compiler turns the declared human-readable `Signature` into a Go ast-grep pattern,
     fed via `pattern-arg` to the ast-grep engine (`gate_type: contracts`).
   - Author the Go absence rule: the grep engine probes the declared `Scope` (file OR
     path) for the forbidden symbol.
4. **Strangler-equivalence harness (REQ-008).** On real Go fixtures, run the pack path
   (real ast-grep + real grep) and the legacy `go/parser` analyzer side by side and
   assert identical verdicts across present, mismatched, and absent cases, plus the
   missing/non-target loud-error case.
5. **Delete the baked Go analyzer + couple the deletion surface (REQ-001/REQ-010/REQ-011).**
   Remove `probeSymbol`, `findFunction`, `findMethod`, `findType`, `findVariable`,
   `formatFuncSignature`, `formatMethodSignature`, `underlyingTypeString`,
   `printFieldList`, `signaturesMatch`, `normalizeSignature`, and the
   `go/parser`/`go/ast`/`go/printer` imports. Rewrite `StepContractSignatureScopedFunc`
   to consume `[]ContractEngineResult` + the verdict function, and DELETE the non-scoped
   `StepContractSignatureFunc` wrapper (REQ-011) — the scoped form is the sole entrypoint.
   Delete-or-migrate every analyzer-coupled test (REQ-010): the tests in
   `step_contract_test.go`, `step_contract_absence_test.go`,
   `step_contract_absence_config_test.go`, `step_contract_noregress_test.go`, and
   `step_contract_parser_absence_test.go` reference the deleted symbols and will not
   compile. Surviving behavior (absence present→violation / absent→pass, signature
   present→satisfied / mismatch→violation, missing/unscanned scope→loud config error) is
   MIGRATED to the pack-SARIF consumer; tests asserting only deleted internals (Go-source
   rendering, string-equality normalization, the dissolved non-`.go`-is-an-error clause)
   are DELETED. After this pass no pkg/gate test references a deleted symbol and
   `go test ./pkg/gate/` compiles.
6. **Wire the pack path into the gate (REQ-006).** Rewire `buildContractStep` /
   `buildGateSteps` (cmd/backstop/gate.go) to route ContractEntry records to the
   pack-produced contracts-gate-type SARIF path, then to the rewritten consumer. Inject
   the contract source via a seam so a spy verifies the pack path is consumed and the old
   analyzer is unreachable.
7. **TypeScript contract rules (REQ-007).** Add the TS signature-presence (ast-grep) and
   absence (grep) contract rules to the shared TS proof pack (the one SPEC-037 / Seed 3
   owns), with real `.ts` fixtures.
8. **No-vacuous-green + dogfood (REQ-009).** Assert each enforcement case produces a real
   blocking violation, and that backstop's own previously-red contract_signature cases
   turn green under the pack path.

## Verification

- **Level:** integration — the work spans pkg/gate (the verdict + deletion), pkg/pack +
  pkg/pack/engine (the grep engine + allowlist + pattern compilation), and cmd/backstop
  (the gate wiring). Coverage threshold 80 (integration).
- **test_command** targets all four packages, with `cmd/backstop/` included so the wiring
  (REQ-006) is exercised end-to-end, not assumed.
- **Real fixtures, no stubs:** the signature (ast-grep), absence (grep), equivalence, and
  TS-proof claims MUST run the real engines over real Go/`.ts` fixtures asserting concrete
  match/no-match/loud-error verdicts. A stub MUST NOT satisfy these claims (Sharp Edge 2).
- **Allowlist gating:** CLM-017/CLM-018 verify the grep engine clears (and, when
  un-allowlisted, fails) the SPEC-035 trust floor.

## Sharp Edges

1. **P0: never compile/render/understand a signature in the binary.** The single
   hardest line. The contract→ast-grep-pattern compiler MUST live in the pack (a
   pack-relative script), exactly like the pack's SARIF convert scripts. If the binary
   gains ANY code that turns a human-readable signature into a pattern, or AST-parses to
   render one, it is a P0 zero-baked-language violation — the exact thing the user
   flagged and rejected. CLM-006 guards this; reviewers should treat any `go/parser`/
   pattern-building in pkg/gate as a hard fail.
2. **Grep coarseness is an accepted false-FAIL trade (OQ-7).** grep matches text, so a
   forbidden symbol's name appearing in a comment, string literal, or as a substring can
   produce a false-FAIL. This is deliberate: absence wants the conservative direction
   (loudly flag any textual appearance). Mitigate with word-boundary / anchored patterns
   (`func Foo\b`) and waivability — do NOT switch absence to ast-grep to "fix" it (that
   reintroduces a per-language grammar and misses comment/string references). CLM-011
   pins the text-presence behavior.
3. **The file-scanned guard is load-bearing against vacuous green.** An empty grep result
   means "absent" ONLY if the scope was actually scanned. Empty-because-missing-file and
   empty-because-not-scanned MUST be loud config errors, never PASS (CLM-012/CLM-013) —
   otherwise deleting the gate-side `parser.ParseFile` probe silently weakens ISSUE-013's
   anti-vacuous-green guarantee. The guard must stay language-agnostic (CLM-014): it
   checks for a scan record, it does not parse.
4. **ast-grep pattern precision is the new fidelity bar (OQ-8).** Fidelity doesn't vanish
   when string-equality dies — it MOVES from "exact Go-source rendering" to "ast-grep
   pattern precision." The per-language pack pattern must handle `[]byte`-vs-`[]uint8`,
   named-vs-unnamed results, and param-name insensitivity (CLM-007). A too-loose pattern
   passes a wrong signature (false SATISFIED); a too-tight pattern reds a correct one
   (false VIOLATION). The strangler-equivalence pass (REQ-008) is the bar that catches
   both before deletion.
5. **Integration-wiring trap (the recurring Seed-1 failure).** A green unit test on the
   verdict function says NOTHING about whether `buildContractStep` actually routes to the
   pack path. The wiring MUST be tested with a spy/sentinel (CLM-020/CLM-021/CLM-022) so an
   unwired path, or one still calling the deleted analyzer, FAILS. cmd/backstop/ is in the
   test_command for this reason.
6. **Grep-engine allowlist gating (atomic with SPEC-035).** The grep engine cannot run
   until `grep`/`rg` are on the trusted-tool allowlist (CheckToolAllowed gates every
   pack-declared command). If the allowlist entry is omitted, every grep absence probe
   fails loud (exit 2) rather than silently running or silently skipping — verify both the
   allowlisted-runs (CLM-017) and un-allowlisted-fails (CLM-018) cells.
7. **Strangler ordering: do not delete before equivalence.** Deleting `step_contract.go`'s
   analyzer (REQ-001) before the equivalence harness (REQ-008) passes would churn working
   tested code blind and risk a silent enforcement regression. The plan MUST sequence
   equivalence-proven → delete, not the reverse.
8. **Shared TS pack ownership.** Seed 3 (SPEC-037) and Seed 4 (this spec) both write to the
   SAME stack-locked TS proof pack. The contract rules are ADDED to it; this spec must not
   create a second TS pack (a pack is stack-locked, so two would split the proof) and must
   not clobber Seed 3's substantiveness rules (CLM-026).
9. **Deletion breaks compilation, not just behavior (REQ-010).** ~24 existing pkg/gate
   tests call the deleted analyzer symbols; once the symbols are gone the PACKAGE WON'T
   COMPILE, so the deletion task is incomplete until those tests are deleted-or-migrated in
   the SAME change (CLM-035). The trap is migrating the behavior but leaving a stale test
   referencing `formatFuncSignature`/`probeSymbol` — a compile break, not a red test. Per
   "align predating artifacts," surviving guarantees (absence polarity, missing-scope loud
   error) must be RE-COVERED against the pack path, not silently dropped (CLM-036).
10. **Dissolving the non-`.go`-error clause must not weaken the scanned guard (REQ-004).**
    The old analyzer used non-`.go` as a proxy for "can't probe." Removing that proxy is
    correct (contracts are cross-stack now), but the replacement MUST still be loud when a
    scope genuinely wasn't scanned — the axis simply moves from "is it `.go`?" to "did the
    engine produce a scan record?". A naive deletion that drops the non-`.go` branch
    WITHOUT the scan-record check would turn empty-because-unscanned into a silent PASS —
    vacuous green (CLM-013/CLM-034 guard both sides).
11. **Scope must be threaded end-to-end, not just on the struct (REQ-012).** Adding
    `ContractEntry.Scope` to the type is inert unless `ExtractContractEntries` actually
    populates it from the frontmatter AND `buildContractStep` passes it through to the grep
    `pattern-arg`. A path-scoped absence (CLM-010) is unreachable if any link in
    extraction → entry → engine drops the scope; CLM-041 asserts the full path, not just
    the field's existence.

## Version History

- **1.1.0** — Corrective pass on spec-review FAIL (4 blockers, all on the
  deletion/reconciliation surface):
  - Added REQ-010 + CLM-035/036/037: delete-or-migrate the ~24 analyzer-coupled pkg/gate
    tests so the package compiles and surviving behavior is preserved through the pack path
    (Blocker 1).
  - Reconciled REQ-004 + added CLM-034: the live "non-`.go` file → config error" behavior
    is explicitly REPLACED by the language-agnostic file-scanned guard (non-`.go` is no
    longer an error, just another stack; scanned-vs-unscanned is the only axis) (Blocker 2).
  - Added REQ-011 + CLM-038/039 and an `absent: true` contract entry: the non-scoped
    `StepContractSignatureFunc` entrypoint is DELETED (Blocker 3).
  - Added REQ-012 + CLM-040/041/042 and the `ExtractContractEntries` contract: the
    `ContractEntry.Scope` (file|path) extraction is brought into scope so path-scoped
    absence (CLM-010) is reachable (Blocker 4).
  - Preserved unchanged: the P0 compiler-in-pack-not-binary guard, the grep-engine
    pack-declared + allowlist stand-up, the strangler-before-deletion ordering, and the
    dogfood claim.
- **1.0.0** — Initial spec (BUNDLE-009 Spec Seed 4).

## Review Questions

These probe risks not fully captured by the claims; the impl-reviewer should check them
against the implementation.

1. Does any code in pkg/gate or cmd/backstop import `go/parser`, `go/ast`, or `go/printer`
   for contract verification after this spec, or build an ast-grep pattern from a
   signature inside the binary? (Either is a P0 violation — Sharp Edge 1.)
2. Is the contract→ast-grep-pattern compiler a pack-relative script invoked via the engine,
   and is the human-readable `Signature` passed through `pattern-arg` unmodified by the
   binary?
3. Does the file-scanned guard distinguish all THREE empty cases — empty-because-absent
   (pass), empty-because-missing-file (loud), empty-because-not-scanned (loud) — and is it
   free of any language-specific parsing?
4. Are the equivalence, signature, absence, and TS-proof tests running the REAL ast-grep
   and REAL grep binaries over real fixtures, such that replacing the engine with a stub
   would make them fail?
5. Is the grep engine declared ONLY in the pack's `engines:` block (no `DefaultRegistry`
   entry), and are `grep`/`rg` added to `TrustedToolAllowlist` such that removing them
   makes the absence probe fail loud?
6. Does the wiring test fail if `buildContractStep` is reverted to the old analyzer or
   left unwired — i.e. is the spy actually asserting consumption of the pack SARIF path,
   not just that some contract step exists?
7. Is the analyzer deletion sequenced strictly after the equivalence pass, and does the
   end state leave ONLY the gate-side polarity + match-verdict + file-scanned guard
   (no baked analyzer)?
8. After the deletion, does `go test ./pkg/gate/` COMPILE — i.e. is there no remaining
   test referencing `formatFuncSignature`/`probeSymbol`/`signaturesMatch`/etc. — and were
   the surviving absence/signature/missing-scope guarantees re-covered against the
   pack-SARIF consumer rather than dropped?
9. Does the file-scanned guard treat a SCANNED non-`.go` scope as a normal verdict (not a
   config error), proving the old "non-`.go` is an error" clause is dissolved, while still
   firing a loud error when the declared scope produced NO scan record?
10. Is the non-scoped `StepContractSignatureFunc` actually GONE (not retained as a
    wrapper), and is `StepContractSignatureScopedFunc` the sole contract entrypoint?
11. Does `ExtractContractEntries` populate `ContractEntry.Scope` from the declared
    contract, and does a path-scoped absence scope flow all the way through to the grep
    `pattern-arg` (so CLM-010 is reachable), without the extraction parsing or compiling a
    signature?

## References

- BUNDLE-009 (`bundles/BUNDLE-009-stack-aware-traceability.bundle.md`) — Spec Seed 4;
  REQ-004/REQ-005/REQ-006/REQ-007/REQ-008; OQ-3/OQ-7/OQ-8; SD-1/SD-3; DD-1/DD-2/DD-4/DD-9.
- SPEC-035 (`specs/SPEC-035-pack-declared-engines-trusted-allowlist.spec.md`) — the
  substrate: pack-declared `engines:` block, `pattern-arg` input mode, the
  trusted-tool allowlist + `CheckToolAllowed`, and the `GateTypeContracts` gate-type.
- SPEC-037 (`specs/SPEC-037-traceability-substantiveness-pack.spec.md`) — Seed 3; co-owns
  the shared TypeScript proof pack this spec adds contract rules to.
- SPEC-034 (`specs/SPEC-034-native-toolchain-engine-cutover.spec.md`) — the
  strangler-equivalence-before-deletion licensing pattern reused here (REQ-008).
- ISSUE-013 — the contract assert-absence anti-vacuous-green policy this spec preserves
  via the grep probe + file-scanned guard.
- pkg/gate/step_contract.go, pkg/gate/step_testverify.go (`ExtractContractEntries` +
  `ContractEntry.Scope`), pkg/pack/engine/allowlist.go, pkg/pack/engine/binding.go,
  cmd/backstop/gate.go — the touched surfaces. The analyzer-coupled tests
  (pkg/gate/step_contract_test.go, step_contract_absence_test.go,
  step_contract_absence_config_test.go, step_contract_noregress_test.go,
  step_contract_parser_absence_test.go) are deleted-or-migrated (REQ-010).
