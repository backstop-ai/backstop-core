---
title: "Traceability Substantiveness Pack"
number: SPEC-037
created: "2026-06-22"
status: draft
schema_version: spec/v1
spec_version: 1.1.0

implementation:
  summary: >
    Eradicate the baked-in Go substantiveness analyzer in pkg/gate/step_testverify.go
    and re-implement test substantiveness as the BUNDLE-009 PACK/GATE/BINARY split
    (Spec Seed 3). Today step_testverify.go's checkSubstantiveness parses every
    mandated test with go/parser and answers two questions in baked Go: Q1 hollow-test
    ("does this test assert anything?" — hasAssertions over a go/ast walk against a
    hardcoded assertionSelectors vocabulary) and Q2 noTarget ("does this test exercise
    the unit under test?" — callsTargetPackage / samePackage joined against the spec's
    implementation.package). That is a baked language analyzer — the exact anomaly the
    zero-baked-checks standing rule eradicates. This spec deletes it and splits its two
    questions across the pack/gate boundary, riding BUNDLE-010's already-shipped
    ast-grep engine and SPEC-035's pack-declared-engines + pattern-arg + gate-TYPE
    substrate. Q1 becomes an ast-grep FINDINGS rule with a per-language assertion
    vocabulary in YAML, emitting SARIF — a hollow test produces a finding (RED), a
    substantive test produces none (GREEN); Go is INCLUDED (ast-grep speaks Go, so the
    Go analyzer is REPLACED by a pack, not preserved as a native tier). Q2 becomes a
    pack EXTRACTION (a positive ast-grep query emitting which packages/symbols a test
    references — no spec awareness in the pack) PLUS a thin, language-agnostic GATE
    SET-MEMBERSHIP test: is the spec's declared implementation.package among the
    extracted referenced symbols? The noTarget SEMANTICS stay in the gate as a set
    test (gate logic consuming pack data, NOT a baked analyzer); only the EXTRACTION
    moves to the pack. The backstop BINARY learns zero language/tool specifics: the
    pack declares ast-grep and the assertion/extraction rules, and the gate consumes
    SARIF findings + the extracted-symbol set. The "beyond Go" proof (SD-3) adds REAL
    ast-grep hollow-test substantiveness rules over REAL .test.ts fixtures to the
    shared TypeScript proof pack (the same pack SPEC-038/Seed 4 adds its TS contract
    rules to — this spec contributes the substantiveness rules to it). Per BUNDLE-009
    REQ-008 / DD-9 the pack-produced substantiveness signal is proven EQUIVALENT to the
    deleted go/parser analyzer on REAL Go fixtures (a strangler-equivalence pass)
    BEFORE step_testverify.go's analyzer is deleted — no vacuous deletion. The gate
    wiring lives in cmd/backstop/gate.go (buildTestSubstantivenessStep wired into
    buildGateSteps), so a spy/sentinel mechanically proves the pack path is in front of
    / replacing the analyzer — an UNWIRED path, or one still calling the old analyzer,
    FAILS the test. End state for substantiveness: zero baked-in analyzer; the only
    gate-side logic that remains is the noTarget set-join + SARIF consumption, both
    language-agnostic.
  package: pkg/gate

verification:
  level: integration
  test_command: go test ./pkg/gate/ ./cmd/backstop/ -race -coverprofile=cover.out
  coverage_threshold: 80

requirements:
  - id: REQ-001
    text: >
      The baked-in Go substantiveness analyzer in pkg/gate/step_testverify.go MUST be
      DELETED: the go/parser-based checkSubstantiveness function, its hasAssertions
      go/ast walk, the hardcoded assertionSelectors vocabulary map, the
      callsTargetPackage / samePackage / targetPackageName helpers, and the
      StepTestSubstantivenessFunc / StepTestSubstantivenessScopedFunc analyzer
      constructors MUST NOT survive in the gate. After this spec, no go/parser- or
      go/ast-based test-substantiveness analysis may exist anywhere in pkg/gate or
      cmd/backstop. The "test must be substantive" INVARIANT is preserved (it is
      re-implemented per REQ-002/REQ-003); only the baked Go ANALYZER is eradicated.
      The deletion MUST NOT be vacuous: it is gated on the REQ-006 strangler-equivalence
      pass passing first.
    supports: stack-aware-traceability:REQ-002
    follows: STD-GO-001:GO-010
  - id: REQ-002
    text: >
      The Q1 hollow-test check ("does this test assert anything?") MUST be
      re-implemented as an ast-grep FINDINGS rule carried by a stack-locked pack, with
      a per-language assertion vocabulary expressed in the rule YAML, emitting SARIF
      through BUNDLE-010's ast-grep→SARIF path. A test whose body contains NO
      assertion-shaped call MUST produce a SARIF finding (the test is hollow → RED); a
      test whose body contains at least one assertion-shaped call MUST produce NO
      finding (the test is substantive → GREEN). Go MUST be INCLUDED as an ast-grep
      pack rule (ast-grep speaks Go) — the Go substantiveness check is REPLACED by a
      pack rule, NOT preserved as a baked native tier. The gate MUST consume the
      ast-grep findings and raise a test_substantiveness violation for each hollow-test
      finding, preserving the existing "test X has no assertions (hollow)" semantics on
      the report surface. The assertion vocabulary lives ONLY in the pack rule YAML;
      the backstop binary MUST carry no hardcoded assertion-selector list.
    supports: stack-aware-traceability:REQ-002
    follows: STD-GO-001:GO-010
  - id: REQ-003
    text: >
      The Q2 noTarget check ("does this test exercise the unit under test?") MUST be
      split across the pack/gate boundary per OQ-6 option (a). The PACK does
      language-specific EXTRACTION: a positive ast-grep query emitting the set of
      packages/symbols a test references (the pack is spec-UNAWARE — it only reports
      what the test references). The GATE does a trivial, language-agnostic
      SET-MEMBERSHIP test: the spec's declared implementation.package (its last path
      component, the target package) MUST be a member of the pack-extracted referenced-
      symbol set, else the gate raises a test_substantiveness noTarget violation ("test
      X does not call package P"). The noTarget SEMANTICS MUST live in the gate as a SET
      TEST consuming pack data — NOT as a baked language analyzer; the gate MUST NOT
      re-introduce go/parser or any per-language reference resolution. The gate set-join
      MUST preserve the existing acceptable-coarseness behavior: (a) when the spec's
      implementation.package yields an EMPTY target package (e.g. a cmd/ package or a
      non-pkg/ path, where targetPackageName returns ""), the noTarget check is SKIPPED
      (no violation — a test with no qualifiable target cannot fail the join); (b) when
      the test resides in the target package itself (same-package), the join is
      satisfied without requiring a package-qualified reference, mirroring the deleted
      analyzer's same-package short-circuit. The disambiguation between (a)/(b) and a
      genuine noTarget MUST be a set/string test over pack-extracted data, never a
      re-baked AST analysis.
    supports: stack-aware-traceability:REQ-003
    follows: STD-GO-001:GO-010
  - id: REQ-004
    text: >
      The "beyond Go" proof (BUNDLE-009 REQ-007 / SD-3) MUST be REAL, not mocked: the
      shared TypeScript proof pack MUST carry a REAL ast-grep hollow-test
      substantiveness rule that fires over REAL `.test.ts` fixtures — a hollow
      `.test.ts` test (calls the subject, asserts nothing) MUST produce a SARIF finding
      and a substantive `.test.ts` test (uses `expect`/an assertion-verb call) MUST
      produce none, exercised through the same ast-grep dispatch path as the Go rule.
      This is the single TS proof pack SHARED with SPEC-038 (Seed 4) — this spec ADDS
      the substantiveness rules to that pack; SPEC-038 adds the contract rules. No claim
      for REQ-004 may be satisfiable by a stubbed pack output: the rule MUST run real
      ast-grep over the real fixtures and assert concrete findings.
    supports: stack-aware-traceability:REQ-007
    follows: STD-GO-001:GO-010
  - id: REQ-005
    text: >
      The pack-based substantiveness path MUST be WIRED into the gate IN FRONT OF /
      REPLACING the deleted analyzer at the substantiveness step builder in
      cmd/backstop/gate.go (buildTestSubstantivenessStep, wired into buildGateSteps) —
      not merely shipped as an unreferenced pack. GROUND TRUTH (live code): the
      ast-grep dispatch is NOT a method on the substantiveness step — it is the
      pre-existing group-by-engine dispatcher dispatchPackEngines (cmd/backstop/pack_gate.go),
      reached through the resolveDispatchPackEngines() / dispatchPackEnginesFn test seam
      (cmd/backstop/code_check.go), already wired as the pack_engines StepFunc in
      buildGateSteps and used by `code check`. buildTestSubstantivenessStep therefore MUST
      NOT re-implement dispatch; it MUST consume the substantiveness pack's findings +
      extraction through that SAME dispatch seam (the substantiveness step calls
      resolveDispatchPackEngines() over the substantiveness pack set, exactly as code
      check and the pack_engines step do) and then run the gate set-join over the result.
      It MUST NOT call the deleted go/parser analyzer (which no longer exists). This MUST
      be proven MECHANICALLY by a spy on the REAL dispatchPackEnginesFn seam such that an
      UNWIRED substantiveness step (one that never reaches dispatchPackEnginesFn) or one
      that still routes to a baked analyzer FAILS the test (the spy records zero
      dispatch invocations and the test asserts non-zero). The verification test_command
      MUST include ./cmd/backstop/ so the wiring locus is covered; this closes the known
      unit-green-but-unwired integration gap.
    supports: stack-aware-traceability:REQ-002
    follows: STD-GO-001:GO-010
  - id: REQ-006
    text: >
      Per BUNDLE-009 REQ-008 / DD-9 (the SPEC-034 strangler pattern), the deletion in
      REQ-001 MUST be guarded by a strangler-equivalence pass: BEFORE the go/parser
      analyzer is deleted, the pack-produced substantiveness signal (Q1 hollow-test
      findings + Q2 extraction + gate set-join) MUST be proven to reproduce the deleted
      analyzer's verdicts on REAL Go test fixtures across the full verdict matrix —
      hollow (Q1 RED), substantive (Q1 GREEN), no-target (Q2 noTarget RED), calls-target
      (Q2 GREEN), and same-package (Q2 GREEN via short-circuit). The equivalence pass
      MUST run REAL ast-grep over REAL Go fixtures and assert concrete findings — it
      MUST NOT be satisfiable by a stub. No vacuous green: the equivalence pass MUST
      demonstrate the pack path produces the hollow-test finding (RED on a genuinely
      hollow test, GREEN on a substantive one) so the eradication does not trade
      enforcement for silence.
    supports: stack-aware-traceability:REQ-008
    follows: STD-GO-001:GO-010
  - id: REQ-007
    text: >
      The extraction→set-join CONSUMPTION SEAM MUST be specified, because the live
      dispatch (dispatchPackEngines → runFindingsEngine → ParsePackFindings,
      cmd/backstop/pack_gate.go) flattens SARIF into a FLAT []gate.Violation that
      carries NEITHER a gate_type discriminator (gate.Violation / check.Violation have
      no GateType field — confirmed) NOR per-test finding identity. Two consumption
      problems MUST be solved GATE-SIDE and language-agnostically, consuming only pack
      SARIF (NO re-baked spec-aware gate-side AST walk — Sharp Edge 2):
      (1) ROUTING — substantiveness findings MUST be isolated out of the flat
      pack_engines stream by NAMESPACED RULE-ID convention, NOT by gate_type filtering
      (the violation carries no gate_type): the substantiveness pack declares stable
      rule IDs (a Q1 hollow-test rule ID and a Q2 referenced-symbol extraction rule ID),
      and the gate selects substantiveness violations by matching the pack's namespaced
      rule IDs (pack.NamespacedRuleID form) on the flat []gate.Violation. A
      RouteSubstantivenessFindings(violations, hollowRuleID, extractionRuleID) helper
      partitions the flat stream into hollow-findings and extraction-findings, ignoring
      all other pack rules.
      (2) KEYING — the Q2 extraction findings MUST be keyed back to a specific
      MandatedTest: the extraction rule emits one SARIF result per (test, referenced-
      symbol) carrying the test's file + enclosing test function name in the finding's
      File/Message/region, and the gate joins each extraction finding to its MandatedTest
      by (FilePath, FuncName) to assemble that test's ReferencedSymbolSet, which is then
      fed to NoTargetViolation(funcName, targetPkg, referenced, samePackage). The keying
      and routing are string/set operations over pack SARIF; the gate re-introduces NO
      go/parser, NO go/ast, and NO per-(test,target) parameterized pack query (the pack
      stays spec-unaware). If a mandated test produces NO extraction finding at all, its
      ReferencedSymbolSet is empty and the set-join's existing dispositions (empty-target
      skip / same-package satisfied / otherwise noTarget) apply unchanged.
    supports: stack-aware-traceability:REQ-003
    follows: STD-GO-001:GO-010
  - id: REQ-008
    text: >
      The existing analyzer-coupled tests in pkg/gate/step_testverify_test.go that call
      the symbols REQ-001 deletes MUST be DELETED-OR-MIGRATED so the package still
      compiles after the analyzer is removed (per the align-predating-artifacts rule):
      TestGate_TargetPackageName, TestGate_TestBodyHelpers,
      TestGate_HasAssertions_HelperPatterns, TestGate_TestSubstantiveness_SubstantiveTestPasses,
      TestGate_TestSubstantiveness_HollowTestFails, TestGate_TestSubstantiveness_NoTargetCallFails,
      and TestGateSteps_FilterToChangedFiles_TestSubstantiveness all reference
      hasAssertions / checkSubstantiveness / StepTestSubstantiveness*Func / targetPackageName
      and would fail to compile once those are gone. Tests whose subject is the deleted
      analyzer (the hasAssertions / checkSubstantiveness / TestBodyHelpers cases) MUST be
      DELETED. TargetPackageName coverage MUST be MIGRATED to the relocated
      pkg/gate.TargetPackageName (the behavior is preserved per REQ-003). CRITICALLY, the
      changed-file SCOPE behavior currently asserted by
      TestGateSteps_FilterToChangedFiles_TestSubstantiveness (a substantiveness verdict is
      raised only for mandated tests whose file is IN the gate's changed-file scope, and
      suppressed for out-of-scope files) MUST be PRESERVED through the new pack path: a
      replacement test MUST assert the same scope-aware behavior over the pack-dispatch +
      set-join path, so scope-aware substantiveness coverage is not silently dropped. After
      this spec, NO test in pkg/gate references any deleted analyzer symbol.
    supports: stack-aware-traceability:REQ-002
    follows: STD-GO-001:GO-010

claims:
  - id: CLM-001
    requirement: REQ-001
    text: >
      After this spec, pkg/gate carries no go/parser- or go/ast-based test-
      substantiveness analyzer: checkSubstantiveness, hasAssertions, assertionSelectors,
      callsTargetPackage, samePackage, and the StepTestSubstantiveness* analyzer
      constructors are absent from the gate source.
    tests:
      - TestSubstantiveness_BakedGoAnalyzerDeleted
  - id: CLM-002
    requirement: REQ-001
    text: >
      The "test must be substantive" invariant survives the deletion: a hollow mandated
      test still fails the substantiveness step (via the pack path), proving the
      deletion removed the analyzer, not the enforcement.
    tests:
      - TestSubstantiveness_InvariantSurvivesDeletion_HollowStillFails
  - id: CLM-003
    requirement: REQ-002
    text: >
      A Go test function whose body contains no assertion-shaped call produces an
      ast-grep hollow-test SARIF finding (RED) through the pack dispatch path.
    tests:
      - TestQ1_Go_HollowTest_ProducesFinding
  - id: CLM-004
    requirement: REQ-002
    text: >
      A Go test function whose body contains at least one assertion-shaped call
      produces NO hollow-test finding (GREEN) through the pack dispatch path.
    tests:
      - TestQ1_Go_SubstantiveTest_ProducesNoFinding
  - id: CLM-005
    requirement: REQ-002
    text: >
      The gate consumes the ast-grep hollow-test findings and raises a
      test_substantiveness violation per hollow finding, preserving the "test X has no
      assertions (hollow)" report-surface semantics.
    tests:
      - TestQ1_GateConsumesHollowFindings_RaisesViolation
  - id: CLM-006
    requirement: REQ-002
    text: >
      The assertion vocabulary lives only in the pack rule YAML and not in the binary:
      no hardcoded assertion-selector list exists in pkg/gate or cmd/backstop after the
      analyzer deletion.
    tests:
      - TestQ1_AssertionVocabularyOnlyInPack_NotBaked
  - id: CLM-007
    requirement: REQ-003
    text: >
      When the pack-extracted referenced-symbol set for a test CONTAINS the spec's
      declared target package, the gate set-join is satisfied and raises NO noTarget
      violation.
    tests:
      - TestQ2_SetJoin_ReferencesTarget_NoViolation
  - id: CLM-008
    requirement: REQ-003
    text: >
      When the pack-extracted referenced-symbol set for a test does NOT contain the
      spec's declared target package, the gate set-join fails and raises a noTarget
      violation ("test X does not call package P").
    tests:
      - TestQ2_SetJoin_DoesNotReferenceTarget_RaisesNoTarget
  - id: CLM-009
    requirement: REQ-003
    text: >
      When the spec's implementation.package yields an EMPTY target package (cmd/ or
      non-pkg/ path), the gate set-join SKIPS the noTarget check and raises no violation
      regardless of the extracted set.
    tests:
      - TestQ2_SetJoin_EmptyTargetPackage_Skipped
  - id: CLM-010
    requirement: REQ-003
    text: >
      When the test resides in the target package itself (same-package), the gate
      set-join treats the join as satisfied without requiring a package-qualified
      reference, mirroring the deleted analyzer's same-package short-circuit.
    tests:
      - TestQ2_SetJoin_SamePackage_Satisfied
  - id: CLM-011
    requirement: REQ-003
    text: >
      The noTarget verdict is computed by the gate as a set/string membership test over
      pack-extracted data, with no go/parser or per-language reference resolution
      re-introduced in the gate.
    tests:
      - TestQ2_NoTargetIsGateSetTest_NotBakedAnalyzer
  - id: CLM-012
    requirement: REQ-004
    text: >
      A hollow `.test.ts` fixture (calls the subject, asserts nothing) produces an
      ast-grep hollow-test SARIF finding when the TypeScript proof pack's
      substantiveness rule runs REAL ast-grep over the REAL fixture.
    tests:
      - TestTS_HollowTestTs_ProducesFinding_RealAstGrep
  - id: CLM-013
    requirement: REQ-004
    text: >
      A substantive `.test.ts` fixture (uses expect/an assertion-verb call) produces NO
      hollow-test finding when the TypeScript proof pack's substantiveness rule runs
      REAL ast-grep over the REAL fixture.
    tests:
      - TestTS_SubstantiveTestTs_ProducesNoFinding_RealAstGrep
  - id: CLM-014
    requirement: REQ-004
    text: >
      The TypeScript substantiveness rule rides the SAME ast-grep dispatch path as the
      Go rule (not a parallel mock): the TS hollow-test finding is produced through the
      shared pack-engine dispatch, and the claim is unsatisfiable by a stubbed pack
      output.
    tests:
      - TestTS_SubstantivenessRidesSharedDispatch_NotStub
  - id: CLM-015
    requirement: REQ-005
    text: >
      buildTestSubstantivenessStep reaches the substantiveness pack's findings +
      extraction through the REAL resolveDispatchPackEngines() / dispatchPackEnginesFn
      seam (the same dispatcher code check and the pack_engines step use) and then runs
      the gate set-join — a spy installed on dispatchPackEnginesFn records that the
      substantiveness step invoked the dispatch seam with the substantiveness pack set.
    tests:
      - TestWiring_SubstantivenessStepRoutesThroughDispatchSeam
  - id: CLM-016
    requirement: REQ-005
    text: >
      An UNWIRED substantiveness step (one that never reaches dispatchPackEnginesFn)
      FAILS the wiring test — the spy on the real dispatchPackEnginesFn seam records zero
      invocations and the test asserts non-zero, so a regression to an unwired or
      baked-analyzer substantiveness step is caught.
    tests:
      - TestWiring_UnwiredSubstantivenessStep_FailsDispatchSpy
  - id: CLM-017
    requirement: REQ-005
    text: >
      The substantiveness step does not re-implement dispatch or call the deleted
      go/parser analyzer: the wiring test asserts the verdict originates from the
      dispatchPackEnginesFn seam (spy-observed) and that no analyzer delegate is invoked
      (it no longer exists), so a hollow/noTarget verdict can only have come from the
      pack dispatch path.
    tests:
      - TestWiring_NoBakedAnalyzerDelegateInvoked
  - id: CLM-018
    requirement: REQ-006
    text: >
      Strangler-equivalence on a hollow Go fixture: the pack path's Q1 verdict (finding
      → RED) equals the pre-deletion go/parser analyzer's hollow verdict, proven over a
      real Go fixture with real ast-grep.
    tests:
      - TestStrangler_Go_Hollow_PackEqualsAnalyzer
  - id: CLM-019
    requirement: REQ-006
    text: >
      Strangler-equivalence on a substantive Go fixture: the pack path's Q1 verdict (no
      finding → GREEN) equals the analyzer's substantive verdict over a real Go fixture.
    tests:
      - TestStrangler_Go_Substantive_PackEqualsAnalyzer
  - id: CLM-020
    requirement: REQ-006
    text: >
      Strangler-equivalence on a no-target Go fixture: the pack extraction + gate
      set-join Q2 verdict (noTarget → RED) equals the analyzer's noTarget verdict over a
      real Go fixture.
    tests:
      - TestStrangler_Go_NoTarget_PackEqualsAnalyzer
  - id: CLM-021
    requirement: REQ-006
    text: >
      Strangler-equivalence on a calls-target Go fixture: the pack extraction + gate
      set-join Q2 verdict (GREEN) equals the analyzer's calls-target verdict over a real
      Go fixture.
    tests:
      - TestStrangler_Go_CallsTarget_PackEqualsAnalyzer
  - id: CLM-022
    requirement: REQ-006
    text: >
      Strangler-equivalence on a same-package Go fixture: the pack extraction + gate
      set-join Q2 verdict (GREEN via same-package satisfaction) equals the analyzer's
      same-package short-circuit verdict over a real Go fixture.
    tests:
      - TestStrangler_Go_SamePackage_PackEqualsAnalyzer
  - id: CLM-023
    requirement: REQ-006
    text: >
      The strangler-equivalence pass is unsatisfiable by a stub: it runs real ast-grep
      over real Go fixtures and asserts concrete findings, so a pack that emits nothing
      (a silent gap) FAILS the hollow-fixture equivalence claim rather than passing
      vacuously.
    tests:
      - TestStrangler_NotSatisfiableByStub_RealFindingsRequired
  - id: CLM-024
    requirement: REQ-007
    text: >
      RouteSubstantivenessFindings isolates substantiveness findings out of a FLAT
      []gate.Violation pack_engines stream by NAMESPACED RULE-ID convention (matching the
      pack's hollow-test and extraction rule IDs in pack.NamespacedRuleID form): given a
      flat stream containing substantiveness findings interleaved with unrelated pack-rule
      findings, the helper returns ONLY the hollow + extraction findings and ignores the
      rest — no gate_type field is consulted (the violation carries none).
    tests:
      - TestRoute_PartitionsSubstantivenessByRuleID_FromFlatStream
  - id: CLM-025
    requirement: REQ-007
    text: >
      Each Q2 extraction finding is keyed back to its MandatedTest by (FilePath,
      FuncName): the gate joins extraction findings to mandated tests and assembles the
      per-test ReferencedSymbolSet that is fed to NoTargetViolation — an extraction
      finding for test X contributes only to X's symbol set, never to another test's.
    tests:
      - TestRoute_KeysExtractionFindingsToMandatedTest
  - id: CLM-026
    requirement: REQ-007
    text: >
      A mandated test that produces NO extraction finding yields an EMPTY
      ReferencedSymbolSet, and the set-join's existing dispositions apply unchanged
      (empty-target → skip, same-package → satisfied, otherwise → noTarget) — the
      keying/routing add no fourth disposition and re-introduce no go/parser.
    tests:
      - TestRoute_NoExtractionFinding_EmptySetFlowsToSetJoin
  - id: CLM-027
    requirement: REQ-008
    text: >
      After this spec, no test in pkg/gate references a deleted analyzer symbol
      (hasAssertions, checkSubstantiveness, assertionSelectors, callsTargetPackage,
      samePackage, targetPackageName, StepTestSubstantivenessFunc,
      StepTestSubstantivenessScopedFunc): the analyzer-subject tests
      (TestGate_TestBodyHelpers, TestGate_HasAssertions_HelperPatterns,
      TestGate_TestSubstantiveness_SubstantiveTestPasses,
      TestGate_TestSubstantiveness_HollowTestFails,
      TestGate_TestSubstantiveness_NoTargetCallFails) are deleted and the package compiles.
    tests:
      - TestSubstantiveness_AnalyzerCoupledTestsDeleted_PackageCompiles
  - id: CLM-028
    requirement: REQ-008
    text: >
      TargetPackageName coverage previously in TestGate_TargetPackageName is MIGRATED to
      the relocated pkg/gate.TargetPackageName with behavior preserved: pkg/... yields the
      last path component, cmd/... and non-pkg/ paths yield "" (the empty-target case).
    tests:
      - TestTargetPackageName_MigratedBehaviorPreserved
  - id: CLM-029
    requirement: REQ-008
    text: >
      The changed-file scope behavior formerly asserted by
      TestGateSteps_FilterToChangedFiles_TestSubstantiveness is PRESERVED through the pack
      path: a mandated test whose file is IN the gate's changed-file scope yields a
      substantiveness verdict via the pack-dispatch + set-join path, while an out-of-scope
      mandated test's file is suppressed — scope-aware substantiveness coverage is not
      silently dropped.
    tests:
      - TestSubstantiveness_ScopeAwareThroughPackPath_Preserved

contracts:
  - file: pkg/gate/substantiveness_join.go
    provides:
      - name: ReferencedSymbolSet
        kind: type
        signature: "type ReferencedSymbolSet map[string]bool"
      - name: NoTargetViolation
        kind: function
        signature: "func NoTargetViolation(funcName, targetPkg string, referenced ReferencedSymbolSet, samePackage bool) (Violation, bool)"
      - name: TargetPackageName
        kind: function
        signature: "func TargetPackageName(implementationPackage string) string"
      - name: RouteSubstantivenessFindings
        kind: function
        signature: "func RouteSubstantivenessFindings(violations []Violation, hollowRuleID, extractionRuleID string) (hollow []Violation, extraction []Violation) // partition the FLAT pack_engines stream by NAMESPACED rule ID (NOT gate_type — the violation carries none), isolating substantiveness findings from unrelated pack rules (REQ-007)"
      - name: ReferencedSetForTest
        kind: function
        signature: "func ReferencedSetForTest(extraction []Violation, test MandatedTest) ReferencedSymbolSet // key extraction findings back to a MandatedTest by (FilePath, FuncName) and assemble its referenced-symbol set (REQ-007/CLM-025)"
    consumes: []
  - file: cmd/backstop/gate.go
    provides:
      - name: buildTestSubstantivenessStep
        kind: function
        signature: "func buildTestSubstantivenessStep(specDir, codeDir, projectRoot string, scope *gate.GateScope) gate.StepFunc // re-implemented to reach the substantiveness pack's Q1 findings + Q2 extraction through the EXISTING resolveDispatchPackEngines()/dispatchPackEnginesFn seam (NOT a re-implemented dispatcher), then run the gate set-join (NoTargetViolation) keyed by MandatedTest; replaces the deleted go/parser analyzer; spy-verified on the real dispatch seam (CLM-015..017)"
    consumes:
      - source: cmd/backstop
        name: resolveDispatchPackEngines
        kind: function
      - source: pkg/gate
        name: ExtractMandatedTests
        kind: function
      - source: pkg/gate
        name: RouteSubstantivenessFindings
        kind: function
      - source: pkg/gate
        name: NoTargetViolation
        kind: function
      - source: pkg/gate
        name: ReferencedSymbolSet
        kind: type
  - file: pkg/gate/step_testverify.go
    provides:
      - name: checkSubstantiveness
        kind: function
        signature: "DELETED — the go/parser substantiveness analyzer (checkSubstantiveness, hasAssertions, assertionSelectors, callsTargetPackage, samePackage, StepTestSubstantivenessFunc, StepTestSubstantivenessScopedFunc) is removed; substantiveness is re-implemented via the pack path + gate set-join (REQ-001)"
    consumes: []
---

# SPEC-037: Traceability Substantiveness Pack

## Overview

BUNDLE-009 (`stack-aware-traceability`, maturity `ready`, v0.6.0) completes a
strangler repeat of SPEC-034 over the gate's traceability subsystem: it eradicates
the baked-in Go traceability analyzers and re-implements them as stack-locked PACKS on
the structural engines, leaving the gate holding only language-agnostic semantics. This
spec implements **Spec Seed 3 — Substantiveness as a pack**.

Today `pkg/gate/step_testverify.go` carries a baked Go analyzer. `checkSubstantiveness`
parses each mandated test with `go/parser` and answers two questions in hardcoded Go:

- **Q1 — hollow-test:** "does this test assert anything?" — `hasAssertions` walks the
  `go/ast` against a hardcoded `assertionSelectors` map.
- **Q2 — noTarget:** "does this test exercise the unit under test?" —
  `callsTargetPackage` / `samePackage` join the test's AST against the spec's declared
  `implementation.package` (the target package).

That baked language analyzer is the exact anomaly the zero-baked-checks standing rule
eradicates (ast-grep speaks Go, so the Go path is REPLACED by a pack, not preserved as
a native tier). This spec deletes it and splits its two questions across the
**PACK / GATE / BINARY** boundary, riding BUNDLE-010's shipped ast-grep engine and
SPEC-035's pack-declared-engines + `pattern-arg` + gate-TYPE substrate:

| Question | Was (deleted) | Becomes |
|----------|---------------|---------|
| Q1 hollow-test | `hasAssertions` go/ast walk + baked `assertionSelectors` | ast-grep FINDINGS rule, per-language assertion vocabulary in YAML → SARIF; hollow = finding (RED), substantive = no finding (GREEN); **Go included** |
| Q2 noTarget | `callsTargetPackage` / `samePackage` go/ast join against `implementation.package` | pack EXTRACTION (positive ast-grep query → referenced symbols) + thin language-agnostic GATE SET-JOIN against the spec's target package |

The **PACK** does the language-specific work (Q1 vocabulary rule + Q2 extraction query),
emitting SARIF. The **GATE** keeps only the language-agnostic noTarget SET-MEMBERSHIP
test (is the declared target package among the extracted symbols?) and SARIF
consumption. The backstop **BINARY** learns zero language/tool specifics. The "beyond
Go" proof adds REAL ast-grep hollow-test rules over REAL `.test.ts` fixtures to the
shared TypeScript proof pack (the same pack SPEC-038/Seed 4 extends with its contract
rules). Per BUNDLE-009 REQ-008 / DD-9, a strangler-equivalence pass proves the pack
signal reproduces the deleted analyzer's verdicts on REAL Go fixtures BEFORE the
analyzer is deleted.

## Requirements

Requirements are defined in the frontmatter `requirements[]` array. They trace to
BUNDLE-009 `REQ-002` (hollow-test as a pack; OQ-2/OQ-6), `REQ-003` (noTarget as
pack-extraction + gate set-join; OQ-6 option a), `REQ-007` (the TypeScript proof pack;
SD-3), and `REQ-008` (strangler-equivalence before deletion; DD-9). (Bundle requirement
numbers differ from this spec's local requirement IDs.) This spec's local REQ-007 pins the
extraction→set-join CONSUMPTION SEAM (how per-test referenced symbols travel from the flat
dispatch SARIF into the gate set-join and how substantiveness findings are routed out of
the flat `pack_engines` stream by namespaced rule ID, since the violation carries no
gate_type), and local REQ-008 pins delete-or-migrate of the existing analyzer-coupled tests
(preserving the changed-file scope behavior through the pack path).

The Q2 noTarget set-join is an exhaustive, language-agnostic allowlist over the
pack-extracted referenced-symbol set and the spec's declared target package: the join is
SATISFIED (no violation) iff the target package is a member of the extracted set, OR the
target package is empty (cmd/ or non-pkg/ path — skipped), OR the test is same-package;
otherwise it is a noTarget violation. No fourth disposition exists, and the verdict is a
set/string test, never a re-baked AST analysis.

## Implementation

Target package: `pkg/gate` (the language-agnostic set-join + SARIF consumption) plus the
substantiveness step re-wiring in `cmd/backstop/gate.go`, plus the pack rule YAML and
fixtures (Go substantiveness pack rules + the shared TypeScript proof pack
substantiveness rules). The go/parser analyzer in `pkg/gate/step_testverify.go` is
deleted.

### Q1 — hollow-test as an ast-grep findings pack (REQ-002)

The hollow-test check moves entirely into an ast-grep rule with a per-language assertion
vocabulary in YAML (the spike's `hollow-test-ts` / Go-equivalent shape: a
test-declaration node that does NOT `has` a descendant assertion-shaped call, vocabulary
matched by an assertion-verb regex — `require|assert|check|verify|expect|must`). The rule
emits findings through BUNDLE-010's ast-grep→SARIF converter. A hollow test (no
assertion-shaped call) produces a finding (RED); a substantive test produces none
(GREEN). Go is authored as an ast-grep rule (ast-grep speaks Go) — the Go check is a
pack rule, not a baked tier. The gate consumes the SARIF findings and raises one
`test_substantiveness` violation per hollow finding, preserving the existing
"test X has no assertions (hollow)" message. The assertion vocabulary lives ONLY in the
rule YAML; the binary carries no `assertionSelectors`.

### Q2 — noTarget as pack-extraction + gate set-join (REQ-003)

The pack does a positive ast-grep EXTRACTION query: emit the set of packages/symbols a
test references (the pack is spec-UNAWARE — it reports what the test references, nothing
about the spec). The gate then performs the language-agnostic SET-MEMBERSHIP test in a
new `pkg/gate/substantiveness_join.go`:

- `TargetPackageName(implementationPackage)` derives the target package (last path
  component for `pkg/...`; empty for `cmd/...` and non-`pkg/` paths) — carried forward
  from the deleted analyzer's `targetPackageName`, now as language-agnostic string logic.
- `NoTargetViolation(funcName, targetPkg, referenced, samePackage)` returns a violation
  iff `targetPkg` is non-empty AND `!samePackage` AND `targetPkg` is NOT a member of the
  `referenced` set. The complete decision table:

| targetPkg | samePackage | target ∈ referenced set | → noTarget? |
|-----------|-------------|-------------------------|-------------|
| "" (cmd/ or non-pkg/) | n/a | n/a | NO — skipped |
| non-empty | yes | n/a | NO — same-package satisfied |
| non-empty | no | yes | NO — references target |
| non-empty | no | no | YES — noTarget violation |

The `samePackage` boolean is derived language-agnostically from the extracted data /
test file package vs the target package (a string comparison), not a re-baked AST walk.
The noTarget SEMANTICS live in the gate as this set test; the gate re-introduces no
go/parser.

### The extraction→set-join consumption seam (REQ-007)

GROUND TRUTH: the live ast-grep dispatch (`dispatchPackEngines` → `runFindingsEngine` →
`check.ParsePackFindings`, `cmd/backstop/pack_gate.go`) flattens SARIF into a FLAT
`[]gate.Violation` that carries NEITHER a `gate_type` discriminator (`gate.Violation` and
`check.Violation` have NO `GateType` field) NOR per-test finding identity. So the gate
cannot route or key substantiveness findings by gate_type. Two gate-side, language-agnostic
helpers in `pkg/gate/substantiveness_join.go` consume the flat pack SARIF:

- **Routing by namespaced rule ID** — `RouteSubstantivenessFindings(violations,
  hollowRuleID, extractionRuleID)` partitions the flat `pack_engines` stream into
  hollow-findings and extraction-findings by matching the substantiveness pack's stable,
  namespaced (`pack.NamespacedRuleID`) rule IDs on each violation's `Rule`. All other pack
  rules are ignored. No `gate_type` field is consulted (the violation carries none).
- **Keying extraction findings back to a test** — the Q2 extraction rule emits one SARIF
  result per (test, referenced-symbol) carrying the test's file + enclosing test-function
  name; `ReferencedSetForTest(extraction, test)` joins those findings to a `MandatedTest`
  by `(FilePath, FuncName)` and assembles that test's `ReferencedSymbolSet`, which is fed
  to `NoTargetViolation(...)`. A test with NO extraction finding gets an empty set, and the
  decision table above applies unchanged.

This keeps the set-join GATE-SIDE and language-agnostic (consuming pack SARIF only); it
re-bakes NO spec-aware gate-side AST walk and adds NO per-(test,target) parameterized pack
query (the pack stays spec-unaware).

### Deletion (REQ-001)

After the strangler-equivalence pass (REQ-006) passes, `pkg/gate/step_testverify.go`'s
`checkSubstantiveness`, `hasAssertions`, `assertionSelectors`, `callsTargetPackage`,
`samePackage`, `targetPackageName` (relocated as `TargetPackageName` in the set-join
file), and the `StepTestSubstantivenessFunc` / `StepTestSubstantivenessScopedFunc`
analyzer constructors are deleted. The mandated-test extraction (`ExtractMandatedTests`,
`MandatedTest`, `ResolveMandatedTestPaths`) and the test-existence verification step are
NOT substantiveness analysis and are retained.

### The TypeScript proof pack (REQ-004)

The shared TypeScript proof pack gains a REAL ast-grep hollow-test substantiveness rule
over REAL `.test.ts` fixtures (one hollow, one substantive), exercised through the same
ast-grep dispatch path as the Go rule. This is the SAME pack SPEC-038 (Seed 4) extends
with its TS contract rules — this spec ADDS the substantiveness rules to it. The rule and
fixtures are real (no stub): the claims assert concrete findings produced by real
ast-grep.

### Wiring (cmd/backstop/gate.go) (REQ-005)

GROUND TRUTH: the ast-grep dispatch is NOT a method the substantiveness step owns — it is
the pre-existing group-by-engine dispatcher `dispatchPackEngines`
(`cmd/backstop/pack_gate.go`), reached through the `resolveDispatchPackEngines()` /
`dispatchPackEnginesFn` test seam (`cmd/backstop/code_check.go`), already wired as the
`pack_engines` StepFunc in `buildGateSteps` and used by `code check`.
`buildTestSubstantivenessStep` is therefore re-implemented NOT to re-implement dispatch but
to CONSUME the substantiveness pack's findings + extraction through that SAME seam: it
extracts the mandated tests (`ExtractMandatedTests` + `ResolveMandatedTestPaths`), calls
`resolveDispatchPackEngines()` over the substantiveness pack set (exactly as code check and
the `pack_engines` step do), routes the flat `[]gate.Violation` result with
`RouteSubstantivenessFindings`, keys the extraction findings per test with
`ReferencedSetForTest`, and runs `NoTargetViolation`. It is wired into `buildGateSteps`,
replacing the deleted analyzer. Because the wiring locus lives in `cmd/backstop/gate.go`
(OUTSIDE `pkg/gate`), the verification `test_command` adds `./cmd/backstop/`, and a spy
installed on the REAL `dispatchPackEnginesFn` seam mechanically proves the substantiveness
step reaches the dispatcher with the substantiveness pack set (CLM-015) and that an
unwired step that never reaches `dispatchPackEnginesFn`, or a baked-analyzer path, fails
(CLM-016/CLM-017) — closing the unit-green-but-unwired gap.

### Strangler-equivalence (REQ-006)

Before the deletion, an equivalence pass runs REAL ast-grep over REAL Go fixtures across
the full verdict matrix — hollow (Q1 RED), substantive (Q1 GREEN), no-target (Q2 RED),
calls-target (Q2 GREEN), same-package (Q2 GREEN) — and asserts the pack path's verdict
equals the (still-present, pre-deletion) analyzer's verdict on each. It is unsatisfiable
by a stub (a pack emitting nothing fails the hollow-fixture claim).

### Processing steps (enumerated for the planner)

1. Author the Go substantiveness pack: a Q1 hollow-test ast-grep rule (assertion-verb
   vocabulary in YAML) + a Q2 referenced-symbol extraction ast-grep query + Go fixtures
   (hollow / substantive / no-target / calls-target / same-package).
2. Add the substantiveness rule + `.test.ts` fixtures (hollow / substantive) to the
   shared TypeScript proof pack.
3. Implement `pkg/gate/substantiveness_join.go`: `ReferencedSymbolSet`,
   `TargetPackageName`, `NoTargetViolation` (the language-agnostic set-join decision
   table), plus the consumption-seam helpers `RouteSubstantivenessFindings` (partition the
   flat `pack_engines` stream by namespaced rule ID) and `ReferencedSetForTest` (key
   extraction findings to a `MandatedTest` by `(FilePath, FuncName)`).
4. Implement gate consumption of the ast-grep hollow-test SARIF findings → one
   `test_substantiveness` violation per routed hollow finding.
5. Run the strangler-equivalence pass over real Go fixtures (full verdict matrix);
   confirm pack verdict == analyzer verdict.
6. Re-implement `buildTestSubstantivenessStep` to CONSUME the substantiveness pack through
   the existing `resolveDispatchPackEngines()` / `dispatchPackEnginesFn` seam (not a
   re-implemented dispatcher) + route/key + set-join, wire into `buildGateSteps`, and add
   the spy-verified wiring test (spy on the REAL `dispatchPackEnginesFn`) in
   `./cmd/backstop/`.
7. DELETE `step_testverify.go`'s go/parser substantiveness analyzer (checkSubstantiveness,
   hasAssertions, assertionSelectors, callsTargetPackage, samePackage,
   StepTestSubstantiveness* constructors) only after step 5 passes; and in the SAME change
   delete-or-migrate the analyzer-coupled tests in `step_testverify_test.go` (REQ-008) —
   delete the analyzer-subject tests, migrate `TargetPackageName` coverage, and add the
   scope-aware replacement test through the pack path — so `pkg/gate` still compiles and
   scope-aware substantiveness coverage is preserved.

## Verification

- **Level:** integration. The Q1 hollow-test rule and the TS proof, the Q2 extraction,
  and the strangler-equivalence pass all run REAL ast-grep over REAL fixtures through the
  pack-engine dispatch path — that crosses the pack/engine/gate seam, so the verification
  is integration, not pure unit. The wiring locus (`buildTestSubstantivenessStep` /
  `buildGateSteps`) lives in `cmd/backstop/gate.go`, so the test scope includes
  `./cmd/backstop/`: the spy-verified wiring claims (CLM-015..017) prove the pack path
  is in front of / replacing the analyzer.
- **Coverage threshold:** 80 (integration level).
- **Test command:** `go test ./pkg/gate/ ./cmd/backstop/ -race -coverprofile=cover.out`.

Claims are defined in the frontmatter `claims[]` array. The Q1 hollow-test matrix is
covered both-polarity per language (Go hollow → finding, Go substantive → no finding; TS
hollow → finding, TS substantive → no finding). The Q2 set-join allowlist is covered
exhaustively across its four dispositions (references-target → no violation,
no-reference → noTarget violation, empty-target → skipped, same-package → satisfied) plus
a "verdict is a gate set test, not a baked analyzer" claim. The extraction→set-join
consumption seam is covered by a routing claim (substantiveness findings partitioned out of
the FLAT `pack_engines` stream by namespaced rule ID, NOT gate_type), a keying claim
(extraction findings keyed back to a `MandatedTest` by `(FilePath, FuncName)`), and a
no-finding→empty-set claim. The deletion is covered by an "analyzer absent" claim and an
"invariant survives" claim. The existing analyzer-coupled tests are covered by a
deleted-and-compiles claim, a `TargetPackageName`-migrated claim, and a scope-aware-through-
the-pack-path preservation claim. The wiring is covered by a spy on the REAL
`dispatchPackEnginesFn` seam proving the substantiveness step reaches the dispatcher, a
negative spy proving an unwired/baked path fails, and a claim that no baked analyzer
delegate is invoked. The strangler-equivalence is covered one claim per verdict-matrix cell
(hollow, substantive, no-target, calls-target, same-package) plus an unsatisfiable-by-stub
claim. The TS proof claims are unsatisfiable by a stub (real ast-grep over real fixtures,
riding the shared dispatch path).

## Sharp Edges

- **Q2 is deliberately coarse — acceptable coarseness, not a defect (OQ-2).** The pack
  extraction + set-join answers "does this test reference the target package?" coarsely
  (syntactic referenced-symbol membership), exactly as the deleted Go analyzer did
  (syntactic package-qualifier match + same-package short-circuit). It WILL false-pass on
  aliased imports / harness indirection and is NOT chased to deep import resolution —
  that is a recorded design choice (diminishing returns, false-positive risk), not debt.
  A reviewer must not "fix" Q2 into a per-language resolver; the coarse syntactic slice
  is the contract.

- **The noTarget join is a spec × test-file join the PACK cannot see alone.** A
  single-file ast-grep query has no knowledge of the spec's declared target package; the
  pack only EXTRACTS referenced symbols. If the join logic leaks back into the pack (a
  per-(test,target) parameterized query), the pack becomes spec-aware and the
  "pack-extraction + thin gate set-join" architecture collapses into a re-baked analyzer.
  The target-package knowledge MUST stay in the gate; the pack stays spec-unaware.

- **Strangler-equivalence MUST run before deletion, on REAL fixtures.** The deletion
  (REQ-001) is gated on the equivalence pass (REQ-006). If the analyzer is deleted before
  equivalence is demonstrated, the equivalence claims have nothing to compare against and
  the deletion is vacuous. The equivalence MUST run real ast-grep over real Go fixtures
  (a stubbed pack output passing the equivalence claim is the precise vacuous-green this
  guards against). Order matters: prove parity, THEN delete.

- **The wiring trap (unit-green-but-unwired).** Authoring a correct pack rule and a
  correct gate set-join, both unit-green, does NOT prove the substantiveness STEP routes
  through them. `buildTestSubstantivenessStep` could still silently call a leftover path
  (or no path), producing a green gate that enforces nothing. The spy on the REAL
  `dispatchPackEnginesFn` seam (CLM-015..017) and the `./cmd/backstop/` test scope are the
  mechanical proof; without them the integration gap re-opens.

- **The dispatch returns the WRONG shape — a flat, gate_type-less, identity-less stream.**
  The live dispatch (`dispatchPackEngines` → `runFindingsEngine` → `ParsePackFindings`)
  flattens SARIF into a flat `[]gate.Violation` that carries NO `gate_type` (neither
  `gate.Violation` nor `check.Violation` has a `GateType` field) and discards which test a
  symbol belongs to. Two traps follow: (1) a reviewer/implementer who tries to ISOLATE
  substantiveness findings by `gate_type` will find no such field — routing MUST be by
  namespaced rule-ID convention (`RouteSubstantivenessFindings`); (2) an implementer who
  tries to recover per-test identity by re-walking the test AST gate-side would re-bake the
  exact spec-aware gate-side analyzer Sharp Edge #2 forbids — keying MUST come from the
  extraction rule's own SARIF (file + enclosing function in each result) joined by
  `(FilePath, FuncName)` in `ReferencedSetForTest`. If the dispatch is later changed to
  carry gate_type natively, that is a separate enhancement; this spec consumes the flat
  stream as it exists today.

- **Vacuous green on deletion — silence masquerading as enforcement.** Deleting the
  go/parser analyzer must not leave a silent gap. If the pack rule is mis-authored (wrong
  node kind, vocabulary that matches everything), a genuinely hollow test could produce no
  finding and the gate would pass — trading enforcement for silence. CLM-002 (invariant
  survives) and the RED-on-hollow / GREEN-on-substantive claims (CLM-003/CLM-004,
  CLM-018/CLM-023) pin that the pack path actually produces the hollow-test finding.

- **The TS proof pack is SHARED with SPEC-038 — additive, not a fork.** This spec ADDS
  substantiveness rules to the single TypeScript proof pack; SPEC-038 adds contract
  rules. If this spec creates a SEPARATE TS pack, the "beyond Go" proof fragments and the
  two seeds collide on pack identity. The substantiveness rules are added to the same
  stack-locked pack; the planner must coordinate with SPEC-038's pack on a shared
  manifest.

- **`TargetPackageName` relocation must not change behavior.** The deleted analyzer's
  `targetPackageName` (empty for cmd/ and non-pkg/, last component for pkg/...) becomes
  `TargetPackageName` in the set-join file. A subtle change here (e.g. returning a
  non-empty target for a cmd/ path) flips the empty-target SKIP into a spurious noTarget
  violation. The relocation is behavior-preserving; CLM-009 pins the empty-target skip.

## Review Questions

- Does the substantiveness STEP in `buildGateSteps` genuinely route through the pack
  dispatch path (Q1 findings + Q2 extraction) such that a spy records the pack path was
  reached, and does an unwired/baked path fail that spy — not merely assert the pack rule
  exists in isolation?
- Is the assertion vocabulary entirely in the pack rule YAML, with NO `assertionSelectors`
  (or any hardcoded assertion-verb list) surviving in `pkg/gate` or `cmd/backstop`?
- Does the Q2 noTarget verdict come from a gate-side set/string membership test over
  pack-extracted referenced symbols, with NO `go/parser` / `go/ast` reference resolution
  re-introduced in the gate?
- Does the strangler-equivalence pass run BEFORE the analyzer deletion, over REAL Go
  fixtures with REAL ast-grep, asserting concrete pack findings equal the analyzer's
  verdict on each of hollow / substantive / no-target / calls-target / same-package?
- Is the empty-target (cmd/ or non-pkg/) case SKIPPED (no violation) and the same-package
  case SATISFIED, matching the deleted analyzer's short-circuits — so the eradication
  preserves behavior, not just removes code?
- Are the TS hollow-test claims satisfied by REAL ast-grep over REAL `.test.ts` fixtures
  riding the shared dispatch path, unsatisfiable by a stubbed pack output, and authored
  into the SAME TS proof pack SPEC-038 shares?
- Does `buildTestSubstantivenessStep` reach the substantiveness pack through the EXISTING
  `resolveDispatchPackEngines()` / `dispatchPackEnginesFn` seam (the same one code check
  and the `pack_engines` step use) rather than re-implementing dispatch, and does the spy
  sit on that REAL seam (not a parallel stub)?
- Are substantiveness findings isolated out of the flat `pack_engines` `[]gate.Violation`
  stream by NAMESPACED rule-ID convention (NOT by a gate_type field, which the violation
  does not carry), and are extraction findings keyed back to a `MandatedTest` by
  `(FilePath, FuncName)` from the pack's own SARIF — with NO gate-side test-AST re-walk?
- Do the existing analyzer-coupled tests in `step_testverify_test.go` get deleted-or-
  migrated so `pkg/gate` still compiles, and is the changed-file scope behavior of
  `TestGateSteps_FilterToChangedFiles_TestSubstantiveness` preserved through the pack path
  (not silently dropped)?

## References

- `bundles/BUNDLE-009-stack-aware-traceability.bundle.md` — REQ-002 (hollow-test as a
  pack; OQ-2/OQ-6), REQ-003 (noTarget as pack-extraction + gate set-join; OQ-6 option a),
  REQ-007 (TypeScript proof pack; SD-3), REQ-008 (strangler-equivalence before deletion;
  DD-9), REQ-010 (zero-baked-analyzer end state); Spec Seed 3; DD-1 (PACK/GATE/BINARY
  split), DD-3 (noTarget = pack-extraction + thin gate set-join), the ast-grep
  feasibility spike (hollow-test rule shape + assertion vocabulary).
- `pkg/gate/step_testverify.go` — the deleted analyzer: `checkSubstantiveness`,
  `hasAssertions`, `assertionSelectors`, `callsTargetPackage`, `samePackage`,
  `targetPackageName`, `StepTestSubstantivenessFunc` / `StepTestSubstantivenessScopedFunc`.
  Retained: `ExtractMandatedTests`, `MandatedTest`, `ResolveMandatedTestPaths`, the
  test-existence verification step.
- `cmd/backstop/gate.go` — `buildTestSubstantivenessStep`, `buildGateSteps` (the wiring
  locus the pack path replaces the analyzer at).
- SPEC-035 — pack-declared engines + trusted-tool allowlist + `pattern-arg` input mode +
  gate-TYPE binding (`GateTypeSubstantiveness`); the substrate this pack binds ast-grep
  through.
- BUNDLE-010 / SPEC-033 — the shipped ast-grep engine + reusable ast-grep→SARIF converter
  + engine-organized pack layout this spec's rules ride.
- SPEC-034 — the strangler licensing pattern (prove equivalence on real fixtures, then
  delete) this spec repeats for substantiveness.
- SPEC-038 (Seed 4, contracts) — shares the single TypeScript proof pack this spec adds
  substantiveness rules to.

## Version History

- **1.1.0** (2026-06-22) — Spec-review corrective pass reconciling with the live dispatch
  shape (3 blockers): (1) REQ-005 + CLM-015..017 + the `buildTestSubstantivenessStep`
  contract now target the REAL `resolveDispatchPackEngines()` / `dispatchPackEnginesFn`
  seam (the substantiveness step CONSUMES the pre-existing `dispatchPackEngines` dispatcher
  instead of re-implementing it; spy sits on the real seam). (2) New local REQ-007 +
  CLM-024..026 + `RouteSubstantivenessFindings` / `ReferencedSetForTest` contracts specify
  the extraction→set-join consumption seam — routing substantiveness findings out of the
  flat `pack_engines` `[]gate.Violation` stream by namespaced rule ID (the violation
  carries no gate_type) and keying extraction findings to a `MandatedTest` by
  `(FilePath, FuncName)` from pack SARIF, staying gate-side and language-agnostic; added the
  dispatch-seam-mismatch Sharp Edge. (3) New local REQ-008 + CLM-027..029 name the existing
  analyzer-coupled tests in `step_testverify_test.go` for delete-or-migrate and preserve the
  changed-file scope behavior through the pack path. The pack path, Q1/Q2 design, strangler
  ordering, zero-baked alignment, and the 3 pre-loaded guards are unchanged.
- **1.0.0** (2026-06-22) — Initial spec authored from BUNDLE-009 (stack-aware-traceability),
  Spec Seed 3 (substantiveness as a pack): delete the baked Go substantiveness analyzer;
  Q1 hollow-test → ast-grep findings pack (Go + TS); Q2 noTarget → pack extraction + thin
  gate set-join; TypeScript proof pack (shared with SPEC-038); strangler-equivalence on
  real Go fixtures before deletion; spy-verified wiring in cmd/backstop/gate.go.
