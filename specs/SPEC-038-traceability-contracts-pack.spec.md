---
title: "Traceability Contracts Pack"
number: SPEC-038
created: "2026-06-22"
updated: "2026-06-23"
status: draft
schema_version: spec/v1
spec_version: 1.2.1

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
  - id: REQ-013
    text: >
      PROVISIONING MODEL (first principle — anything that runs in a gate is INSTALLED
      from a pack; the binary ships ONLY a way to install packs and execute them). The
      Go CONTRACTS pack — the contract→ast-grep-pattern compiler (REQ-002) + the grep
      absence probe binding (REQ-003) authored as an ORDINARY pack — is an ordinary
      INSTALLED pack, NOT a privileged tier. It MUST NOT be built into the binary, MUST
      NOT be embedded via `//go:embed` or any compiled-in asset, MUST NOT be reached
      through a baked code path or analyzer bridge, and MUST NOT be a production reliance
      on a `testdata` fixture (testdata may be used by tests ONLY, never as the path a
      real gate run resolves the pack from). For dogfooding, backstop-core MUST install
      the Go contracts pack into ITSELF as a LOCAL pack via the existing distribution
      path (`pkg/pack/distribution/{add,install,verify}.go`): `pack add <local-source>`
      records it in backstop.yml with the `local` source value and writes a `local`
      lockfile entry; `VerifyLock` SKIPS local packs (verify.go ~line 46–49), so a local
      source needs no remote artifact. backstop-core thereby GATES ITSELF on contracts
      through this installed local pack. PACK-OWNERSHIP PRECISION: the TypeScript contract
      RULES are CO-OWNED with the shared TS proof pack (REQ-007, the same pack SPEC-037
      Seed 3 installs); the GO contract rules ship in the NEW Go contracts pack this spec
      authors — that Go pack is the installable artifact whose installation REQ-013/REQ-014
      assert (the TS pack's installation is already covered by SPEC-037). This pins
      BUNDLE-009 REQ-010's "the backstop binary holds no language/tool specifics for
      traceability" for contracts — the rules live in a pack, installed.
    supports: stack-aware-traceability:REQ-010
    follows: STD-GO-001:GO-010
  - id: REQ-014
    text: >
      REAL-OVER-INSTALLED-PACK END-TO-END PROOF (closes the recurring pack-provisioning
      integration gap; mirrors SPEC-037 REQ-010). Beyond REQ-006's wiring spy (which
      proves the contract step CONSUMES the pack SARIF path but NOT that the whole
      pipeline runs over a real INSTALLED pack), there MUST be a test that INSTALLS the Go
      contracts pack as a LOCAL pack (per REQ-013: `pack add` a local source → declared +
      locked) and then runs the REAL gate contract path END TO END through the PRODUCTION
      pipeline: real pack resolution → real `dispatchPackEngines` → real ast-grep
      (signature presence) + real grep (symbol absence) over real fixtures → real convert
      (engine-output→SARIF via the convert script under the real macOS sandbox) → SARIF →
      gate verdict (match-verdict + absence polarity + file-scanned guard). The test MUST
      assert TWO real violations produced by the WHOLE pipeline: (a) a MISSING/mismatched
      signature (no ast-grep match) yields a real contract VIOLATION (exit 2), and (b) a
      PRESENT forbidden symbol (real grep match under an absence contract) yields a real
      absence VIOLATION (exit 2). It MUST NOT be satisfiable by a stub, by pointing
      production at a `testdata` pack directory, or by the wiring spy alone: it MUST FAIL
      if the contracts pack is NOT actually installed (absent local declaration/lock) or
      NOT actually run (no real ast-grep / grep dispatch). The substrate is now shipped:
      multi-rule ast-grep dispatch (ISSUE-028 — the contracts pack carries an ast-grep
      signature rule) and sandboxed convert (ISSUE-029) make the real path runnable, and
      the grep engine is pack-declared via SPEC-035's `pattern-arg` + the trusted-tool
      allowlist (REQ-005). REQ-006's wiring-spy claims are RETAINED (they prove wiring);
      this REQ ADDS the over-installed-pack proof on top.
    supports: stack-aware-traceability:REQ-010
    follows: STD-GO-001:GO-010
  - id: REQ-015
    text: >
      CONTRACTS-ARM CAPABILITY RE-KEY (live locus; mirrors SPEC-037 REQ-009 / CLM-035 /
      CLM-036). SPEC-036 shipped `deriveCapabilityState` at `cmd/backstop/gate.go:273`,
      which derives each traceability dimension's `CapabilityState`. SPEC-037 (Seed 3)
      ALREADY re-keyed the SUBSTANTIVENESS arm onto its installed pack and LEFT the
      coverage + contracts arms on the baked-Go keying. Because THIS spec DELETES the baked
      Go contract analyzer (REQ-001), the CONTRACTS arm of `deriveCapabilityState` MUST now
      be re-keyed onto the INSTALLED contracts pack (Present/Working iff the contracts pack
      is installed / resolvable — read from `cfg.Packs`, NOT the deleted go/parser analyzer
      and NOT a built-in tier; mirroring the existing `substantivenessPackInstalled` /
      `substantivenessPackName` helpers with contracts equivalents). CRITICAL ASYMMETRY —
      the 3-arm END STATE after this spec MUST be stated explicitly: (1) the SUBSTANTIVENESS
      arm stays re-keyed on the installed substantiveness pack (LEFT AS-IS from Seed 3 — this
      spec MUST NOT touch it); (2) the CONTRACTS arm re-keys onto the installed contracts
      pack (this spec's change); (3) the COVERAGE arm STAYS baked-Go-keyed (coverage was
      descoped per BUNDLE-009 REQ-009 — no pack exists, so re-keying it would mark it
      capability-absent with nothing to install). With the contracts pack installed → the
      contracts dimension RUNS; without it AND undeclared → class-2 capability-absent (warn,
      exit 0, per SPEC-036); without it AND declared → class-3 declared-intent-unmet (block).
      This aligns SPEC-036's derivation via implementation (per align-predating-artifacts);
      SPEC-036 itself is NOT revised. The re-key MUST MIGRATE the shipped test that asserts
      the OLD contracts keying and would go RED on `./cmd/backstop/` after the re-key:
      `TestCapabilityState_NonGoProject_DerivesAbsentClass2`
      (`cmd/backstop/gate_capability_test.go:17`) currently iterates
      `{DimensionCoverage, DimensionContracts}` TOGETHER in the baked-Go loop (asserting a Go
      project yields `Present` via the baked analyzer). Once the contracts arm re-keys,
      `DimensionContracts` MUST be SPLIT OUT of that loop — leaving ONLY `DimensionCoverage`
      on the baked-Go assertion — and the contracts dimension's installed-pack keying asserted
      separately (mirroring how Seed 3 split out `DimensionSubstantiveness`). After this spec,
      NO test in pkg/gate OR cmd/backstop asserts the deleted baked-analyzer keying for the
      contracts dimension, `./cmd/backstop/` stays green, and no claim is orphaned by a
      silently-broken shipped test.
    supports: stack-aware-traceability:REQ-010
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

  # REQ-013 — Go contracts pack is an ordinary INSTALLED local pack (not baked/embedded/testdata)
  - id: CLM-043
    requirement: REQ-013
    text: The Go contracts pack is provisioned as an ORDINARY INSTALLED pack, not a baked tier — no `//go:embed` (or other compiled-in asset) carries the contract→ast-grep-pattern compiler or grep-absence rule YAML, and no production gate code path resolves the contracts pack from a `testdata` directory; the contract rules are absent from the binary and present only in an installed pack
    tests:
      - TestProvisioning_ContractsPackNotEmbeddedNorTestdata
  - id: CLM-044
    requirement: REQ-013
    text: backstop-core dogfood-installs the Go contracts pack into itself as a LOCAL pack — after `pack add <local-source>`, backstop.yml declares the pack with the `local` source value and the lockfile carries a `local` source-type entry, and VerifyLock passes WITHOUT a remote artifact (local packs are skipped) — proving the local provisioning path is the dogfood mechanism, not a remote fetch
    tests:
      - TestProvisioning_ContractsInstalledAsLocalPack_DeclaredAndLocked
  - id: CLM-045
    requirement: REQ-013
    text: The installable artifact this spec authors is the GO contracts pack (its contract→ast-grep-pattern compiler + grep-absence binding) — the TypeScript contract rules are CO-OWNED with the shared TS proof pack (REQ-007, SPEC-037's pack), so the Go pack is the new pack whose declared+locked installation this spec asserts, distinct from the already-installed TS proof pack
    tests:
      - TestProvisioning_GoContractsPackIsTheNewInstallable_TSRulesShareProofPack

  # REQ-014 — REAL over-installed-pack end-to-end (signature + absence), unstubbable
  - id: CLM-046
    requirement: REQ-014
    text: REAL over-installed-pack end-to-end (signature) — with the Go contracts pack INSTALLED as a local pack, a contract whose declared signature is MISSING/mismatched run through the WHOLE production pipeline (real pack resolution → real dispatchPackEngines → real ast-grep → real convert-under-sandbox → SARIF → gate verdict) yields a REAL blocking contract violation (exit 2), proven without a stub, without pointing production at testdata, and not merely via the wiring spy
    tests:
      - TestE2E_ContractsInstalledLocalPack_RealGate_MissingSignatureRed
  - id: CLM-047
    requirement: REQ-014
    text: REAL over-installed-pack end-to-end (absence) — with the Go contracts pack INSTALLED as a local pack, an absence contract whose forbidden symbol is PRESENT run through the WHOLE production pipeline (real pack resolution → real dispatchPackEngines → real grep → real convert-under-sandbox → SARIF → absence polarity verdict) yields a REAL blocking absence violation (exit 2), proven with real grep over a real fixture, no stub
    tests:
      - TestE2E_ContractsInstalledLocalPack_RealGate_PresentForbiddenSymbolRed
  - id: CLM-048
    requirement: REQ-014
    text: The end-to-end proof FAILS if the contracts pack is not actually installed or not actually run — with the local pack declaration/lock ABSENT (or the real ast-grep/grep dispatch not reached), the same missing-signature and present-forbidden-symbol fixtures produce NO contract/absence violation through the production path, so the test cannot pass vacuously — it pins that the verdict came from the real installed pack, not a residual baked path
    tests:
      - TestE2E_ContractsUninstalled_NoVacuousGreen
  - id: CLM-049
    requirement: REQ-014
    text: The end-to-end pipeline exercises BOTH the real ast-grep signature dispatch (multi-rule, ISSUE-028) and the real grep absence dispatch through the convert script under the real macOS sandbox (ISSUE-029) — so the proof is over the real engines + real convert, not a single-engine or sandbox-bypassed shortcut
    tests:
      - TestE2E_ContractsRealAstGrepAndGrep_AndSandboxedConvert

  # REQ-015 — contracts-arm capability re-key (3-arm asymmetry; migrate shipped test)
  - id: CLM-050
    requirement: REQ-015
    text: "The contracts CAPABILITY re-keys at the LIVE LOCUS deriveCapabilityState (cmd/backstop/gate.go:273): for the CONTRACTS dimension ONLY, the source becomes the installed-contracts-pack signal (read from cfg.Packs via a contracts-pack-installed helper) — NOT the deleted go/parser analyzer and NOT a built-in tier. With the pack installed, deriveCapabilityState returns Present/Working for contracts and the gate RUNS it; without the pack AND undeclared, the dimension classifies class-2 (capability-absent → warn, exit 0, per SPEC-036); without the pack AND declared, it classifies class-3 (declared-intent-unmet → block)"
    tests:
      - TestCapability_ContractsKeyedOnInstalledPack_NotBakedAnalyzer
  - id: CLM-051
    requirement: REQ-015
    text: "The 3-arm END STATE is asymmetric and explicit — after this spec deriveCapabilityState keys SUBSTANTIVENESS on its installed pack (left as-is from Seed 3), CONTRACTS on its installed contracts pack (this spec), and COVERAGE on the baked-Go analyzer (UNCHANGED — coverage descoped, no pack). For a Go project with NEITHER pack installed, (cfg, DimensionCoverage) returns Present via the baked analyzer while (cfg, DimensionContracts) and (cfg, DimensionSubstantiveness) return absent (pack-not-installed) — proving only contracts re-keyed here and coverage stayed baked"
    tests:
      - TestCapability_RekeyIsContractsOnly_CoverageStaysBaked_SubstantivenessUntouched
  - id: CLM-052
    requirement: REQ-015
    text: The shipped test TestCapabilityState_NonGoProject_DerivesAbsentClass2 (cmd/backstop/gate_capability_test.go:17), which after Seed 3 iterates {DimensionCoverage, DimensionContracts} together in the baked-Go loop and would go RED on ./cmd/backstop/ once contracts re-keys, is MIGRATED — DimensionContracts is SPLIT OUT of the baked-Go loop (leaving ONLY DimensionCoverage there) and the contracts dimension's installed-pack keying is asserted separately, so the shipped test is not silently broken, no claim is orphaned, and ./cmd/backstop/ stays green
    tests:
      - TestCapability_ShippedCapabilityTest_MigratedForContractsRekey

contracts:
  - file: pkg/gate/step_contract.go
    provides:
      - name: ContractEntry
        kind: type
        signature: "type ContractEntry struct"
        notes: "RETAINED as the declared-contract record (File/Name/Kind/Signature/Absent unchanged in meaning) and EXTENDED with Scope (the absence file-OR-path parameter, REQ-003/CLM-010). The struct is now a pure data record fed to the pack engine via pattern-arg; it no longer drives a go/parser probe. Signature is passed through to the pack's contract→ast-grep-pattern compiler unmodified (REQ-002/CLM-006)."
      - name: StepContractSignatureScopedFunc
        kind: function
        signature: "func StepContractSignatureScopedFunc(results []ContractEngineResult, scope *GateScope) StepFunc"
        notes: "REWRITTEN and the SOLE retained contract entrypoint: consumes pack-produced SARIF (ContractEngineResult) instead of running go/parser. Applies the language-agnostic verdict — ast-grep match = SATISFIED / no-match = VIOLATION (REQ-002), grep present-match = absence VIOLATION (REQ-003) — plus the file-scanned guard (REQ-004). It NO LONGER imports go/parser/go/ast/go/printer (REQ-001/CLM-001). The deleted probeSymbol/findFunction/findMethod/findType/findVariable/formatFuncSignature/formatMethodSignature/underlyingTypeString/printFieldList/signaturesMatch/normalizeSignature symbols are removed (CLM-001/CLM-002/CLM-003). The non-scoped convenience wrapper StepContractSignatureFunc is DELETED (REQ-011/CLM-038/CLM-039) — not retained as a shim."
      - name: StepContractSignatureFunc
        kind: function
        signature: "func StepContractSignatureFunc(contracts []ContractEntry) StepFunc"
        absent: true
        notes: "DELETED (REQ-011/CLM-038): the non-scoped go/parser-era wrapper that only called StepContractSignatureScopedFunc(contracts, nil). It has no live non-test caller (buildContractStep uses the scoped pack-SARIF consumer); declared absent so its reappearance is a deletion regression."
    consumes:
      - source: pkg/pack/engine
        name: GateTypeContracts
        kind: constant
      - source: pkg/pack/engine
        name: InputModePatternArg
        kind: constant
  - file: pkg/gate/contract_verdict.go
    provides:
      - name: ContractEngineResult
        kind: type
        signature: "type ContractEngineResult struct"
        notes: "NEW: the gate-side, language-agnostic carrier of one pack engine probe result for one contract entry — Entry (the declared ContractEntry), Matched (did the ast-grep/grep query match), Scanned (the file-scanned guard signal, REQ-004/CLM-013), Locations (for the violation message). The gate verdicts purely off these fields; it never re-parses source."
      - name: VerifyContractVerdict
        kind: function
        signature: "func VerifyContractVerdict(r ContractEngineResult) (Violation, bool)"
        notes: "NEW: the pure, language-agnostic verdict function — present-contract no-match → violation; absence-contract match → violation; unscanned scope → loud config-error violation; otherwise no violation. The single source of the polarity/match-verdict logic that stays gate-side (REQ-002/REQ-003/REQ-004). Has zero language imports (CLM-014)."
    consumes: []
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
      - name: deriveCapabilityState
        kind: function
        signature: "func deriveCapabilityState(cfg *config.Config, dim gate.TraceabilityDimension) gate.CapabilityState"
        notes: "RE-KEYED (REQ-015/CLM-050/CLM-051): the CONTRACTS arm now keys on the INSTALLED contracts pack (Present/Working iff installed/resolvable via a contractsPackInstalled/contractsPackName helper reading cfg.Packs), NOT the deleted go/parser analyzer. Seed 3 already re-keyed the SUBSTANTIVENESS arm (LEFT AS-IS here); the COVERAGE arm STAYS baked-Go-keyed (coverage descoped, no pack). SPEC-036 is NOT revised — aligned via implementation. The shipped TestCapabilityState_NonGoProject_DerivesAbsentClass2 is migrated to split DimensionContracts out of the baked-Go loop (CLM-052)."
      - name: contractsPackInstalled
        kind: function
        signature: "func contractsPackInstalled(cfg *config.Config) bool"
        notes: "NEW (REQ-015/CLM-050): reports whether the contracts pack is INSTALLED (recorded in cfg.Packs — a local pack records the value \"local\"). The installed-pack-resolvable signal the contracts capability keys on after the baked analyzer's deletion. Reads ONLY the declaration surface (cfg.Packs), never the binary — mirrors the shipped substantivenessPackInstalled."
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

Requirements are defined in the frontmatter `requirements[]` array, each tracing to
BUNDLE-009 via `supports`. Summary of the rule surface:

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
| REQ-013 | Go contracts pack is an ordinary INSTALLED local pack | dogfood-installed into backstop-core as a `local` pack (declared + locked; VerifyLock skips local) | `//go:embed`/built-in/baked-bridge; production resolving the pack from `testdata` |
| REQ-014 | REAL over-installed-pack end-to-end (signature + absence) | install local pack → real dispatch → real ast-grep + real grep → real sandboxed convert → SARIF → real violation (missing sig + present forbidden symbol) | stub / testdata-pointed-at-by-production / wiring-spy-alone; passing when the pack isn't installed or run |
| REQ-015 | Contracts-arm capability re-key (3-arm asymmetry) | CONTRACTS keys on the installed contracts pack; SUBSTANTIVENESS left as-is (Seed 3); COVERAGE stays baked-Go | re-keying coverage; touching the substantiveness arm; leaving the shipped capability test asserting the old contracts keying (RED on `./cmd/backstop/`) |

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
3. **Author + dogfood-install the Go contracts pack (REQ-002/REQ-003/REQ-013).**
   - Author the Go contract signature rule: a pack-relative contract→ast-grep-pattern
     compiler turns the declared human-readable `Signature` into a Go ast-grep pattern,
     fed via `pattern-arg` to the ast-grep engine (`gate_type: contracts`).
   - Author the Go absence rule: the grep engine probes the declared `Scope` (file OR
     path) for the forbidden symbol.
   - PROVISIONING (REQ-013): the Go contracts pack is an ORDINARY pack — NO `//go:embed`,
     no baked tier, no production reliance on `testdata`. Dogfood-install it into
     backstop-core as a LOCAL pack via `pack add <local-source>`
     (`pkg/pack/distribution/{add,install,verify}.go`): declared `local` in backstop.yml +
     a `local` lockfile entry; `VerifyLock` SKIPS local packs, so no remote artifact is
     needed. backstop-core thereby gates ITSELF on contracts through the installed pack
     (CLM-043/CLM-044). The Go contracts pack is the NEW installable artifact this spec
     authors; the TypeScript contract rules are CO-OWNED with the shared TS proof pack
     (REQ-007 / SPEC-037's already-installed pack), so only the Go pack's
     declared+locked installation is asserted here (CLM-045).
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
6. **Wire the pack path into the gate + re-key the contracts capability (REQ-006/REQ-015).**
   Rewire `buildContractStep` / `buildGateSteps` (cmd/backstop/gate.go) to route
   ContractEntry records to the pack-produced contracts-gate-type SARIF path, then to the
   rewritten consumer. Inject the contract source via a seam so a spy verifies the pack
   path is consumed and the old analyzer is unreachable.
   - CONTRACTS-ARM RE-KEY (REQ-015): re-key the CONTRACTS arm of `deriveCapabilityState`
     (`cmd/backstop/gate.go:273`, the SPEC-036 function) onto the INSTALLED contracts pack
     (Present/Working iff installed/resolvable — read from `cfg.Packs` via a
     `contractsPackInstalled` / `contractsPackName` helper mirroring the shipped
     `substantivenessPackInstalled` / `substantivenessPackName`), NOT the deleted go/parser
     analyzer. The 3-arm END STATE is asymmetric and explicit: the SUBSTANTIVENESS arm is
     LEFT AS-IS (re-keyed by Seed 3 — do NOT touch it); the CONTRACTS arm re-keys here; the
     COVERAGE arm STAYS baked-Go-keyed (coverage descoped, no pack) (CLM-050/CLM-051).
   - MIGRATE THE SHIPPED TEST (REQ-015): `TestCapabilityState_NonGoProject_DerivesAbsentClass2`
     (`cmd/backstop/gate_capability_test.go:17`) currently iterates
     `{DimensionCoverage, DimensionContracts}` together in the baked-Go loop and would go RED
     on `./cmd/backstop/` once contracts re-keys. SPLIT `DimensionContracts` OUT of that loop
     (leaving ONLY `DimensionCoverage`) and assert the contracts dimension's installed-pack
     keying separately — mirroring how Seed 3 split out `DimensionSubstantiveness` (CLM-052).
7. **TypeScript contract rules (REQ-007).** Add the TS signature-presence (ast-grep) and
   absence (grep) contract rules to the shared TS proof pack (the one SPEC-037 / Seed 3
   owns), with real `.ts` fixtures.
8. **No-vacuous-green + dogfood (REQ-009).** Assert each enforcement case produces a real
   blocking violation, and that backstop's own previously-red contract_signature cases
   turn green under the pack path.
9. **REAL over-installed-pack end-to-end (REQ-014).** With the Go contracts pack INSTALLED
   as a local pack (pass 3), run the REAL gate contract path end to end through the
   PRODUCTION pipeline (real pack resolution → real `dispatchPackEngines` → real ast-grep
   [signature] + real grep [absence] → real convert-under-sandbox → SARIF → gate verdict)
   and assert TWO real violations: a missing/mismatched signature → contract violation
   (exit 2), and a present forbidden symbol → absence violation (exit 2). The test MUST FAIL
   if the pack is NOT installed or NOT run (no stub, no testdata-pointed-at-by-production,
   not the wiring spy alone), exercising both real ast-grep (ISSUE-028) and the real
   sandboxed convert (ISSUE-029) (CLM-046/CLM-047/CLM-048/CLM-049).

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
12. **The testdata-as-production trap (the recurring pack-provisioning integration gap).**
    Every prior pack-migration impl in this codebase passed its tests by pointing at a
    `testdata` pack or stubbing the dispatcher, leaving the REAL installed-pack path
    unproven — and that gap bit earlier seeds (SPEC-035 P4, SPEC-037 Seed 3). A green
    signature/absence unit suite over a `testdata` pack does NOT prove backstop gates itself
    on contracts via an INSTALLED pack. The guard is REQ-014's over-installed-pack E2E
    (CLM-046..049): `pack add` a LOCAL Go contracts pack and run the WHOLE production
    pipeline (real ast-grep signature + real grep absence), with a no-vacuous-green negative
    (uninstalled/unrun → no violation, CLM-048). Production must resolve the pack from the
    installed (local) declaration, never from `testdata`; CLM-043 pins no-embed /
    no-testdata-in-production.
13. **`//go:embed` / baked-tier temptation (REQ-013).** It is tempting to bundle the
    contract→ast-grep-pattern compiler or the grep-absence rule YAML into the binary
    "so the dogfood always works." That is exactly the baked tier the zero-baked-checks rule
    eradicates and REQ-013 prohibits: the binary ships ONLY a way to install + execute packs.
    The dogfood works because backstop-core INSTALLS the Go contracts pack as a LOCAL pack,
    not because the rules are compiled in (CLM-044).
14. **Capability re-key asymmetry — re-key CONTRACTS ONLY; do NOT touch substantiveness or
    coverage (REQ-015).** `deriveCapabilityState` (`cmd/backstop/gate.go:273`) serves all
    three traceability dimensions and Seed 3 ALREADY re-keyed the substantiveness arm. The
    temptation is to re-key the whole function or to "re-fix" substantiveness. Both are WRONG.
    After this spec the 3-arm end state is: SUBSTANTIVENESS keyed on its installed pack (left
    as-is — touching it would churn Seed 3's working code), CONTRACTS re-keyed onto its
    installed contracts pack (this spec), COVERAGE STILL baked-Go-keyed (coverage was
    descoped — BUNDLE-009 REQ-009 — so re-keying it would mark it capability-absent with no
    pack to install). CLM-051 pins the asymmetry; CLM-050 pins the contracts re-key at the
    live locus.
15. **The shipped capability test goes RED on the contracts re-key — migrate it in the SAME
    change (REQ-015).** `TestCapabilityState_NonGoProject_DerivesAbsentClass2`
    (`cmd/backstop/gate_capability_test.go:17`) was migrated by Seed 3 to iterate
    `{DimensionCoverage, DimensionContracts}` together in the baked-Go loop (asserting a Go
    project yields `Present` via the baked analyzer). Once the contracts arm re-keys to the
    installed pack, that loop's contracts assertion flips RED on `./cmd/backstop/` (this
    spec's test scope) — a compile-clean but FAILING test, easy to miss if the implementer
    only adds the re-key. `DimensionContracts` MUST be split out of the baked-Go loop
    (leaving ONLY `DimensionCoverage`) with the installed-pack keying asserted separately,
    exactly as Seed 3 split out `DimensionSubstantiveness`. CLM-052 pins the migration; per
    "align predating artifacts," SPEC-036 is aligned via implementation, not revised.

## Version History

- **1.2.0** (2026-06-23) — Targeted alignment with the restated first principle (anything
  that runs in a gate is INSTALLED from a pack; the binary ships only install + execute) and
  the now-fixed substrate (ISSUE-028 multi-rule ast-grep dispatch, ISSUE-029 sandboxed
  convert) — adapting the SPEC-037 v1.2.2 / Seed 3 provisioning + E2E + capability-rekey
  pattern to CONTRACTS. The contracts-eradication CORE is unchanged (grep-as-pack-declared
  engine, the pack-side contract→ast-grep compiler, absence-via-grep, strangler-equivalence
  REQ-008, the wiring spy REQ-006, the TS contract rules REQ-007, delete-or-migrate
  REQ-010/011, the dual-substrate dogfood REQ-009). The delta:
  - New REQ-013 + CLM-043/044/045 pin the PROVISIONING model: the GO contracts pack is an
    ORDINARY INSTALLED pack — NOT built-in, NOT `//go:embed`, NOT a baked bridge, NOT a
    production reliance on testdata — dogfood-installed into backstop-core as a LOCAL pack
    (declared + locked; VerifyLock skips local). PACK-OWNERSHIP precision: the TS contract
    rules are co-owned with SPEC-037's shared TS proof pack (REQ-007); the Go contracts pack
    is the NEW installable artifact this spec authors.
  - New REQ-014 + CLM-046/047/048/049 add the REAL over-installed-pack END-TO-END proof
    (install local pack → real dispatch → real ast-grep [signature] + real grep [absence] →
    real sandboxed convert → SARIF → gate verdict), asserting a real missing-signature
    violation AND a real present-forbidden-symbol absence violation, failing if the pack
    isn't actually installed/run — closing the recurring pack-provisioning integration gap
    that REQ-006's wiring-spy alone did not (REQ-006 + CLM-020..022 RETAINED).
  - New REQ-015 + CLM-050/051/052 re-key the CONTRACTS arm of `deriveCapabilityState`
    (`cmd/backstop/gate.go:273`) onto the installed contracts pack (NOT the deleted go/parser
    analyzer), with the explicit 3-arm asymmetry: SUBSTANTIVENESS left as-is (Seed 3),
    CONTRACTS re-keyed here, COVERAGE stays baked-Go-keyed. Added the `deriveCapabilityState`
    + `contractsPackInstalled` contract entries. MIGRATES the shipped test
    `TestCapabilityState_NonGoProject_DerivesAbsentClass2`
    (`cmd/backstop/gate_capability_test.go:17`): `DimensionContracts` is split out of the
    baked-Go loop (leaving ONLY `DimensionCoverage`), its installed-pack keying asserted
    separately — so `./cmd/backstop/` stays green and no claim is orphaned. SPEC-036 aligned
    via implementation, not revised (align-predating-artifacts).
  - Added four Sharp Edges (testdata-as-production trap; `//go:embed`/baked-tier temptation;
    capability re-key asymmetry; shipped-test-goes-RED migration) and five Review Questions.
    Applied the SPEC-037 v1.2.2 `absent: true` contract lesson: the deleted symbols are not
    declared as live `provides[]` signatures (only the `StepContractSignatureFunc`
    deletion-regression guard uses `absent: true`), and NO `signature:` field carries a
    trailing prose comment (bare Go signatures only) — so the spec's own contracts are clean
    for the new pack-based check.
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
12. Is the Go contracts pack an ORDINARY INSTALLED pack — with NO `//go:embed` / baked tier
    and NO production code path resolving it from `testdata` — and does backstop-core
    dogfood-install it into itself as a LOCAL pack (declared `local` in backstop.yml +
    `local` lockfile entry, VerifyLock passing without a remote artifact)?
13. Is there a REAL over-installed-pack END-TO-END test that installs the Go contracts pack
    as a local pack and runs the WHOLE production pipeline (real pack resolution → real
    `dispatchPackEngines` → real ast-grep [signature] + real grep [absence] → real
    convert-under-sandbox → SARIF → gate verdict) over real fixtures, asserting BOTH a real
    missing-signature contract violation AND a real present-forbidden-symbol absence
    violation — and does it FAIL when the pack is NOT installed or NOT run (not a stub, not
    testdata-in-production, not the wiring spy alone)?
14. Is the CONTRACTS arm of `deriveCapabilityState` (`cmd/backstop/gate.go:273`) keyed on the
    INSTALLED contracts pack (Present/Working iff resolvable; undeclared+absent → class-2,
    declared+absent → class-3 per SPEC-036) and NOT on the deleted baked-Go-analyzer presence
    — and is the 3-arm end state correct: SUBSTANTIVENESS left as-is (Seed 3), CONTRACTS
    re-keyed, COVERAGE still baked-Go-keyed (not re-keyed)?
15. Was the shipped `TestCapabilityState_NonGoProject_DerivesAbsentClass2`
    (`cmd/backstop/gate_capability_test.go:17`) migrated so `DimensionContracts` is split out
    of the baked-Go loop (leaving ONLY `DimensionCoverage`) with the contracts installed-pack
    keying asserted separately — so `./cmd/backstop/` stays green and the substantiveness arm
    is untouched?

## References

- BUNDLE-009 (`bundles/BUNDLE-009-stack-aware-traceability.bundle.md`) — Spec Seed 4;
  REQ-004/REQ-005/REQ-006/REQ-007/REQ-008; OQ-3/OQ-7/OQ-8; SD-1/SD-3; DD-1/DD-2/DD-4/DD-9.
- SPEC-035 (`specs/SPEC-035-pack-declared-engines-trusted-allowlist.spec.md`) — the
  substrate: pack-declared `engines:` block, `pattern-arg` input mode, the
  trusted-tool allowlist + `CheckToolAllowed`, and the `GateTypeContracts` gate-type.
- SPEC-037 (`specs/SPEC-037-traceability-substantiveness-pack.spec.md`) — Seed 3; co-owns
  the shared TypeScript proof pack this spec adds contract rules to, and is the TEMPLATE
  this spec mirrors for provisioning (REQ-013 ← SPEC-037 REQ-009), real over-installed-pack
  E2E (REQ-014 ← SPEC-037 REQ-010), and the capability re-key (REQ-015 ← SPEC-037 REQ-009 /
  CLM-035/036/037). SPEC-037 ALREADY re-keyed the substantiveness arm of
  `deriveCapabilityState` and shipped `substantivenessPackInstalled` / `substantivenessPackName`,
  which this spec mirrors with contracts equivalents.
- SPEC-034 (`specs/SPEC-034-native-toolchain-engine-cutover.spec.md`) — the
  strangler-equivalence-before-deletion licensing pattern reused here (REQ-008).
- SPEC-036 — shipped `deriveCapabilityState` (`cmd/backstop/gate.go:273`) and the shipped
  test `TestCapabilityState_NonGoProject_DerivesAbsentClass2`
  (`cmd/backstop/gate_capability_test.go:17`). This spec deletes the baked contract analyzer
  and re-keys the CONTRACTS arm onto the INSTALLED contracts pack (REQ-015 / CLM-050/051/052),
  migrating that shipped test's contracts arm; coverage/substantiveness arms untouched. An
  openly-recorded alignment via implementation — SPEC-036 itself is NOT revised.
- `pkg/pack/distribution/{add,install,verify}.go` — the local-pack provisioning path
  (REQ-013): `Add` records `local` source in backstop.yml + lockfile, `VerifyLock` SKIPS
  `local` packs (verify.go ~line 46–49), so a local source needs no remote artifact.
- ISSUE-013 — the contract assert-absence anti-vacuous-green policy this spec preserves
  via the grep probe + file-scanned guard.
- ISSUE-028 — multi-rule ast-grep packs now dispatch for real (the contracts pack's
  signature ast-grep rule), unblocking the real over-installed-pack end-to-end path (REQ-014).
- ISSUE-029 — convert scripts (engine-output→SARIF) now run under the real macOS sandbox,
  unblocking the real convert step in the end-to-end pipeline (REQ-014).
- pkg/gate/step_contract.go, pkg/gate/step_testverify.go (`ExtractContractEntries` +
  `ContractEntry.Scope`), pkg/pack/engine/allowlist.go, pkg/pack/engine/binding.go,
  cmd/backstop/gate.go (`buildContractStep`, `deriveCapabilityState`,
  `contractsPackInstalled`) — the touched surfaces. The analyzer-coupled tests
  (pkg/gate/step_contract_test.go, step_contract_absence_test.go,
  step_contract_absence_config_test.go, step_contract_noregress_test.go,
  step_contract_parser_absence_test.go) are deleted-or-migrated (REQ-010), and the shipped
  cmd/backstop/gate_capability_test.go is migrated for the contracts re-key (REQ-015).
