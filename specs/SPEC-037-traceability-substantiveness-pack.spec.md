---
title: "Traceability Substantiveness Pack"
number: SPEC-037
created: "2026-06-22"
updated: "2026-08-16"
status: implemented
schema_version: spec/v1
spec_version: 1.2.9

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
    SET-MEMBERSHIP test: is the spec's declared implementation.subject (reduced to an
    opaque last-segment token) among the extracted referenced symbols? The noTarget SEMANTICS stay in the gate as a set
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
  subject: pkg/gate

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
    supports: stack-aware-traceability:REQ-002@1.0.0
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
    supports: stack-aware-traceability:REQ-002@1.0.0
    follows: STD-GO-001:GO-010
  - id: REQ-003
    text: >
      The Q2 noTarget check ("does this test exercise the unit under test?") MUST be
      split across the pack/gate boundary per OQ-6 option (a). The PACK does
      language-specific EXTRACTION: a positive ast-grep query emitting the set of
      packages/symbols a test references (the pack is spec-UNAWARE — it only reports
      what the test references). The GATE does a trivial, language-agnostic
      SET-MEMBERSHIP test: the spec's declared implementation.subject (reduced to its
      last path segment, an OPAQUE token — the gate bakes in NO cmd//pkg/ layout
      knowledge) MUST be a member of the pack-extracted referenced-symbol set, else the
      gate raises a test_substantiveness noTarget violation ("test
      X does not call package P"). The noTarget SEMANTICS MUST live in the gate as a SET
      TEST consuming pack data — NOT as a baked language analyzer; the gate MUST NOT
      re-introduce go/parser or any per-language reference resolution. The gate set-join
      MUST preserve the existing acceptable-coarseness behavior: (a) when the spec/claim
      declares an EMPTY subject (no callable subject to qualify against, so
      TargetPackageName returns ""), the noTarget check is SKIPPED (no violation — a test
      with no qualifiable target cannot fail the join); the empty-target skip is keyed on
      an EMPTY subject, NOT on any cmd//pkg/ layout classification (TargetPackageName
      reduces a non-empty subject to its last path segment with no layout special-case
      per ISSUE-047); (b) when
      the test resides in the target package itself (same-package), the join is
      satisfied without requiring a package-qualified reference, mirroring the deleted
      analyzer's same-package short-circuit. The disambiguation between (a)/(b) and a
      genuine noTarget MUST be a set/string test over pack-extracted data, never a
      re-baked AST analysis. The Q2 extraction query MUST run REAL ast-grep through the
      production dispatch path (no stub) — multi-rule ast-grep dispatch (ISSUE-028) and
      sandboxed convert (ISSUE-029) make the real extraction genuinely runnable.
    supports: stack-aware-traceability:REQ-003@1.0.0
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
      ast-grep over the real fixtures and assert concrete findings. This is now
      unconditionally runnable (no stub justified): multi-rule ast-grep packs dispatch for
      real (ISSUE-028) and convert scripts run under the real macOS sandbox (ISSUE-029), so
      the TS hollow-test rule MUST run real ast-grep through the production dispatch path.
    supports: stack-aware-traceability:REQ-007@1.0.0
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
    supports: stack-aware-traceability:REQ-002@1.0.0
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
    supports: stack-aware-traceability:REQ-008@1.0.0
    follows: STD-GO-001:GO-010
  - id: REQ-007
    text: >
      The extraction→set-join CONSUMPTION SEAM MUST be specified, because the live
      dispatch (dispatchPackEngines → runFindingsEngine → ParsePackFindings,
      cmd/backstop/pack_gate.go) flattens SARIF into a FLAT []gate.Violation that
      carries NEITHER a discriminator FINE-GRAINED enough to separate the Q1 hollow
      findings from the Q2 extraction findings NOR per-test finding identity.
      (gate.Violation carries a GateType field as of ISSUE-118; check.Violation still
      does not. GateType is stamped from the PRODUCING ENGINE BINDING, and the
      substantiveness pack declares Q1 and Q2 under a SINGLE binding, so every
      substantiveness finding carries the same gate_type — it can never partition the
      two roles.) Two consumption
      problems MUST be solved GATE-SIDE and language-agnostically, consuming only pack
      SARIF (NO re-baked spec-aware gate-side AST walk — Sharp Edge 2):
      (1) ROUTING — substantiveness findings MUST be isolated out of the flat
      pack_engines stream by NAMESPACED RULE-ID convention, NOT by gate_type filtering
      (gate_type is per-BINDING and so cannot separate the two roles): the
      substantiveness pack declares stable
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
    supports: stack-aware-traceability:REQ-003@1.0.0
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

      DELETE-OR-MIGRATE EXTENDS TO cmd/backstop (existing-test-coupling). The capability
      re-key (REQ-009) invalidates a SHIPPED SPEC-036 test that lives in this spec's own
      test scope: TestCapabilityState_NonGoProject_DerivesAbsentClass2 at
      cmd/backstop/gate_capability_test.go:17 asserts the OLD keying for the
      SUBSTANTIVENESS dimension (it iterates DimensionSubstantiveness alongside coverage and
      contracts and asserts a Go project yields Present via the baked analyzer). Once
      deriveCapabilityState re-keys substantiveness onto the installed pack, that assertion
      goes RED on `./cmd/backstop/` (this spec's test_command). That test MUST therefore be
      MIGRATED as part of the re-key: its substantiveness arm updated to assert the
      INSTALLED-pack keying, with its coverage and contracts arms LEFT UNCHANGED (those
      dimensions do not re-key — REQ-009 / CLM-036). After this spec, NO test in pkg/gate OR
      cmd/backstop asserts the deleted baked-analyzer keying for the substantiveness
      dimension, and no claim is left orphaned by a silently-broken shipped test.
    supports: stack-aware-traceability:REQ-002@1.0.0
    follows: STD-GO-001:GO-010
  - id: REQ-009
    text: >
      PROVISIONING MODEL (first principle — anything that runs in a gate is INSTALLED
      from a pack; the binary ships only a way to install packs and execute them).
      The substantiveness pack (its Q1 hollow-test ast-grep rule + Q2 referenced-symbol
      extraction rule) is an ORDINARY INSTALLED PACK, NOT a privileged tier. It MUST NOT
      be built into the binary, MUST NOT be embedded via `//go:embed` or any compiled-in
      asset, MUST NOT be reached through a baked code path or analyzer bridge, and MUST
      NOT be a production reliance on a `testdata` fixture (testdata may be used by tests
      ONLY, never as the path a real gate run resolves the pack from). Those prohibitions
      are absolute and are the whole of what this requirement forbids. For dogfooding,
      backstop-core MUST install the substantiveness pack into ITSELF through the STANDARD
      PACK DISTRIBUTION PATH (`pack add`), such that the pack is DECLARED in backstop.yml,
      LOCKED in backstop.lock with a resolvable source, and lock verification PASSES.
      EITHER source type SATISFIES this requirement: a LOCAL source (`pack add
      <local-source>` → the `local` declaration value + a `local` source-type lockfile
      entry; `VerifyLock` SKIPS local packs, so no remote artifact is required) OR a REMOTE
      source (`pack add <org>/<pack>@<version>` → a `git` source-type lockfile entry
      carrying the source coordinate and tag). The requirement is INSTALLED-AND-RESOLVABLE
      via the distribution path, NOT `local` specifically; the local-pack support shipped in
      pkg/pack/distribution/{add,install,verify}.go and the remote path shipped in
      SPEC-055/SPEC-056 are both acceptable provisioning routes. backstop-core thereby GATES
      ITSELF on substantiveness through this installed pack — the dogfood (backstop's own
      gate going RED on a genuinely hollow backstop test via the installed pack) IS the
      proof the path is real, not a stub. PUBLISHING these rules as a backstop-ai reference
      pack consumed REMOTELY is therefore an ACCEPTED provisioning state, not a deviation,
      and changes nothing about the invariant: the binary still ships no substantiveness
      rules. This pins BUNDLE-009 REQ-010's "the backstop binary holds no language/tool
      specifics for traceability" — the rules live in a pack, installed.

      CAPABILITY RE-KEY (live locus). SPEC-036 shipped deriveCapabilityState at
      cmd/backstop/gate.go:272, which derives every traceability dimension's
      CapabilityState from cfg.Language + baked-Go-analyzer presence. Because this spec
      DELETES the baked Go substantiveness analyzer (REQ-001), the SUBSTANTIVENESS arm of
      deriveCapabilityState MUST be re-keyed onto the INSTALLED substantiveness pack
      (Present/Working iff the pack is installed/resolvable) — the implementer changes
      that function, not an abstract behavior. The re-key MUST be
      SUBSTANTIVENESS-DIMENSION-ONLY: the coverage and contracts arms of
      deriveCapabilityState MUST stay on their existing keying (coverage was descoped per
      BUNDLE-009 REQ-009; contracts keeps its baked analyzer until SPEC-038/Seed 4 ships
      its pack), because re-keying contracts now would break it before its pack exists.
      This aligns SPEC-036's derivation via implementation (per align-predating-artifacts);
      SPEC-036 itself is NOT revised by this spec.
    supports: stack-aware-traceability:REQ-010@1.0.0
    follows: STD-GO-001:GO-010
  - id: REQ-010
    text: >
      REAL-OVER-INSTALLED-PACK END-TO-END PROOF (closes the recurring pack-provisioning
      integration gap). Beyond REQ-005's spy on the dispatch SEAM (which proves the step
      CALLS the dispatcher but NOT that the whole pipeline runs over a real INSTALLED
      pack), there MUST be a test that INSTALLS the substantiveness pack through the
      standard distribution path REQ-009 mandates — a LOCAL source is the appropriate choice
      for a hermetic test workspace (`pack add` a local source → declared + locked), and
      REQ-009 permits either source type — and then runs the REAL
      gate substantiveness path over it END TO END through the PRODUCTION pipeline: real
      pack resolution → real dispatchPackEngines → real ast-grep over real fixtures → real
      convert (ast-grep→SARIF via the convert script under the real macOS sandbox) →
      SARIF → gate route + set-join. The test MUST assert that a genuinely HOLLOW backstop
      test (a real `*_test.go` source that asserts nothing) yields a REAL
      test_substantiveness violation produced by the WHOLE pipeline. It MUST NOT be
      satisfiable by a stub, by pointing production at a `testdata` pack directory, or by a
      seam spy alone: it MUST FAIL if the substantiveness pack is NOT actually installed
      (absent local declaration/lock) or NOT actually run (no real ast-grep dispatch).
      This is now genuinely runnable on the live substrate: multi-rule ast-grep packs
      dispatch for real (ISSUE-028 — the substantiveness pack has 2 ast-grep rules: Q1
      hollow + Q2 extraction) and convert scripts run under the real macOS sandbox
      (ISSUE-029), so the end-to-end path is no longer blocked. REQ-005's seam-spy claims
      are RETAINED (they prove wiring); this REQ ADDS the over-installed-pack proof on top.
    supports: stack-aware-traceability:REQ-010@1.0.0
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
      When the spec/claim declares an EMPTY subject (TargetPackageName reduces it to the
      empty-target "" token), the gate set-join SKIPS the noTarget check and raises no
      violation regardless of the extracted set — the skip is keyed on an EMPTY subject,
      not on any cmd//pkg/ layout classification.
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
    subject: cmd/backstop
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
    subject: cmd/backstop
    text: >
      An UNWIRED substantiveness step (one that never reaches dispatchPackEnginesFn)
      FAILS the wiring test — the spy on the real dispatchPackEnginesFn seam records zero
      invocations and the test asserts non-zero, so a regression to an unwired or
      baked-analyzer substantiveness step is caught.
    tests:
      - TestWiring_UnwiredSubstantivenessStep_FailsDispatchSpy
  - id: CLM-017
    requirement: REQ-005
    subject: cmd/backstop
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
      rest — no gate_type field is consulted (it is per-BINDING and cannot separate the
      two roles).
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
      the relocated pkg/gate.TargetPackageName, which reduces an OPAQUE subject to its
      last path segment with NO cmd//pkg/ layout knowledge (e.g. cmd/backstop→backstop,
      pkg/gate→gate, internal/foo→foo, a bare token passes through unchanged); ONLY an
      EMPTY subject yields the empty-target "" case (ISSUE-047 removed the cmd//pkg/
      layout special-casing).
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
  - id: CLM-037
    requirement: REQ-008
    subject: cmd/backstop
    text: >
      The shipped SPEC-036 test TestCapabilityState_NonGoProject_DerivesAbsentClass2
      (cmd/backstop/gate_capability_test.go:17), which asserts the OLD baked-analyzer
      keying for the substantiveness dimension and would go RED on ./cmd/backstop/ once
      substantiveness re-keys, is MIGRATED as part of the re-key: its SUBSTANTIVENESS arm
      asserts the INSTALLED-pack keying, while its COVERAGE and CONTRACTS arms are left
      UNCHANGED — so the shipped test is not silently broken and no claim is orphaned, and
      ./cmd/backstop/ stays green.
    tests:
      - TestCapability_ShippedSpec036Test_MigratedForSubstantivenessRekey
  - id: CLM-030
    requirement: REQ-009
    subject: cmd/backstop
    text: >
      The substantiveness pack is provisioned as an ORDINARY INSTALLED pack, not a baked
      tier: no `//go:embed` (or other compiled-in asset) carries the substantiveness rule
      YAML, and no production gate code path resolves the pack from a `testdata`
      directory — the substantiveness rules are absent from the binary and present only in
      an installed pack.
    tests:
      - TestProvisioning_SubstantivenessPackNotEmbeddedNorTestdata
  - id: CLM-031
    requirement: REQ-009
    subject: cmd/backstop
    text: >
      backstop-core dogfood-installs the substantiveness pack into itself through the
      STANDARD PACK DISTRIBUTION PATH: after `pack add`, backstop.yml DECLARES the pack and
      backstop.lock carries a RESOLVABLE entry for it, and lock verification PASSES. EITHER
      source type satisfies this: a LOCAL install (the `local` declaration value + a `local`
      source-type lock entry, which VerifyLock skips so no remote artifact is required) OR a
      REMOTE install (a `git` source-type lock entry carrying the source coordinate and
      tag). What is pinned is that the pack is INSTALLED AND RESOLVABLE via the distribution
      path — NOT that its source is local, and never a baked, `//go:embed`ed, or
      testdata-resolved provisioning.
    tests:
      - TestProvisioning_SubstantivenessInstalledViaDistributionPath_LocalOrGit_DeclaredAndLocked
  - id: CLM-032
    requirement: REQ-010
    subject: cmd/backstop
    text: >
      REAL over-installed-pack end-to-end: with the substantiveness pack INSTALLED as a
      local pack, a genuinely hollow backstop `*_test.go` source run through the WHOLE
      production pipeline (real pack resolution → real dispatchPackEngines → real ast-grep
      → real convert-under-sandbox → SARIF → gate route + set-join) yields a REAL
      test_substantiveness violation — proven without a stub, without pointing production
      at testdata, and not merely via the dispatch-seam spy.
    tests:
      - TestE2E_SubstantivenessInstalledLocalPack_RealGate_HollowRed
  - id: CLM-033
    requirement: REQ-010
    subject: cmd/backstop
    text: >
      The end-to-end proof FAILS if the substantiveness pack is not actually installed or
      not actually run: with the local pack declaration/lock ABSENT (or the real ast-grep
      dispatch not reached), the same hollow fixture produces NO substantiveness violation
      through the production path, so the test cannot pass vacuously — it pins that the
      verdict came from the real installed pack, not from a residual baked path.
    tests:
      - TestE2E_SubstantivenessUninstalled_NoVacuousGreen
  - id: CLM-034
    requirement: REQ-010
    subject: cmd/backstop
    text: >
      The end-to-end pipeline exercises BOTH the multi-rule ast-grep dispatch (the
      substantiveness pack's Q1 hollow + Q2 extraction rules both dispatch, per ISSUE-028)
      and the convert script under the real macOS sandbox (ast-grep→SARIF via the convert
      script, per ISSUE-029) — so the proof is over the real engine + real convert, not a
      single-rule or sandbox-bypassed shortcut.
    tests:
      - TestE2E_SubstantivenessMultiRuleDispatch_AndSandboxedConvert
  - id: CLM-035
    requirement: REQ-009
    subject: cmd/backstop
    text: >
      The substantiveness CAPABILITY re-keys at the LIVE LOCUS deriveCapabilityState
      (cmd/backstop/gate.go:272, the function SPEC-036 shipped that currently derives
      from cfg.Language + baked-Go-analyzer presence): for the SUBSTANTIVENESS dimension
      ONLY, the source becomes "the substantiveness pack is INSTALLED / resolvable" —
      NOT the deleted go/parser analyzer and NOT a built-in tier. With the pack installed,
      deriveCapabilityState returns Present/Working for substantiveness and the gate RUNS
      it; without the pack AND undeclared, the dimension classifies class-2 (capability-
      absent → warn-with-guidance, exit 0, per SPEC-036); without the pack AND declared,
      it classifies class-3 (declared-intent-unmet → block). The re-key is
      substantiveness-dimension-only: the coverage and contracts arms of
      deriveCapabilityState stay on their existing keying unchanged (CLM-036).
    tests:
      - TestCapability_SubstantivenessKeyedOnInstalledPack_NotBakedAnalyzer
  - id: CLM-036
    requirement: REQ-009
    subject: cmd/backstop
    text: >
      The re-key is DIMENSION-ASYMMETRIC: deriveCapabilityState's COVERAGE and CONTRACTS
      arms MUST NOT re-key to an installed pack — coverage was descoped (its analyzer is
      deleted with no in-bundle replacement) and contracts keeps its baked analyzer until
      SPEC-038/Seed 4 ships its pack. For a Go project, deriveCapabilityState(cfg,
      DimensionContracts) and (cfg, DimensionCoverage) MUST still return their existing
      keying (unchanged), while (cfg, DimensionSubstantiveness) keys on the installed pack
      — so the substantiveness re-key does not break contracts/coverage before their packs
      exist.
    tests:
      - TestCapability_RekeyIsSubstantivenessOnly_CoverageContractsUnchanged

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
        signature: "func TargetPackageName(subject string) string"
      - name: RouteSubstantivenessFindings
        kind: function
        signature: "func RouteSubstantivenessFindings(violations []Violation) (hollow, extraction []Violation)"
      - name: ReferencedSetForTest
        kind: function
        signature: "func ReferencedSetForTest(extraction []Violation, test MandatedTest) ReferencedSymbolSet"
    consumes: []
  - file: cmd/backstop/gate.go
    provides:
      - name: buildTestSubstantivenessStep
        kind: function
        signature: "func buildTestSubstantivenessStep(specDir, codeDir, projectRoot string, scope *gate.GateScope, classifier gate.SourceClassifier, matcher gate.TestNameMatcher) gate.StepFunc"
      - name: deriveCapabilityState
        kind: function
        signature: "func deriveCapabilityState(packs []*pack.Manifest, dim gate.TraceabilityDimension, stack string) gate.CapabilityState"
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
        signature: "func checkSubstantiveness"
        absent: true
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
the flat `pack_engines` stream by namespaced rule ID, since gate_type is per-BINDING and
cannot separate the two roles), and local REQ-008 pins delete-or-migrate of the existing analyzer-coupled tests
(preserving the changed-file scope behavior through the pack path).

The Q2 noTarget set-join is an exhaustive, language-agnostic allowlist over the
pack-extracted referenced-symbol set and the spec's declared target subject: the join is
SATISFIED (no violation) iff the target token is a member of the extracted set, OR the
target token is empty (the subject was empty — skipped), OR the test is same-package;
otherwise it is a noTarget violation. No fourth disposition exists, and the verdict is a
set/string test, never a re-baked AST analysis. TargetPackageName reduces the declared
subject to its last path segment as an OPAQUE token (no cmd//pkg/ layout knowledge —
ISSUE-047); only an empty subject yields the empty-target skip.

## Implementation

Target package: `pkg/gate` (the language-agnostic set-join + SARIF consumption) plus the
substantiveness step re-wiring in `cmd/backstop/gate.go`, plus the pack rule YAML and
fixtures (Go substantiveness pack rules + the shared TypeScript proof pack
substantiveness rules). The go/parser analyzer in `pkg/gate/step_testverify.go` is
deleted.

### Provisioning — the substantiveness pack is an ORDINARY INSTALLED pack (REQ-009)

First principle: anything that runs in a gate is INSTALLED from a pack; the binary ships
ONLY a way to install packs and execute them. The substantiveness pack is therefore an
ordinary installed pack — NOT built-in, NOT `//go:embed`-bundled, NOT a baked bridge, NOT
a production reliance on `testdata` (testdata is a test-only convenience, never the path a
real gate resolves the pack from). Those prohibitions are absolute. For dogfooding,
backstop-core installs the pack into ITSELF through the STANDARD distribution path
(`pack add`), and EITHER source type satisfies REQ-009:

- **LOCAL** — `pack add <local-source>` records the pack in `backstop.yml` with the `local`
  source value and writes a `local`-source lockfile entry (`distribution.Add` →
  `updateBackstopYml` → `yml.Packs[packName] = "local"`; lockfile `SourceType: "local"`).
  `VerifyLock` SKIPS `local` packs (they are not under `packsDir`), so a local install needs
  no remote artifact — the local source IS the pack.
- **REMOTE** — `pack add <org>/<pack>@<version>` clones the published pack at its tag through
  the identity gate and writes a `git`-source lockfile entry carrying `source_coordinate`,
  `git_ref`, and `content_hash`, verified by `VerifyLock` against the installed tree.

What REQ-009 pins is INSTALLED-AND-RESOLVABLE via the distribution path — declared in
`backstop.yml`, locked in `backstop.lock`, lock verification passing — not `local`
specifically. backstop-core thus gates ITSELF on substantiveness via the installed pack: the
dogfood (backstop's own gate going RED on a genuinely hollow backstop test, through the
installed pack) is the proof the path is real. Publishing these rules as a backstop-ai
reference pack consumed REMOTELY is an accepted provisioning state, not a deviation — the
binary still ships no substantiveness rules either way.

**CapabilityState keying after deletion (REQ-009 / CLM-035 / CLM-036).** SPEC-036 shipped
`deriveCapabilityState` at `cmd/backstop/gate.go:272`, which keys every traceability
dimension's `CapabilityState` on `cfg.Language` + baked-Go-analyzer presence (the only
capability that existed on the no-pack binary). This spec DELETES the baked substantiveness
analyzer (REQ-001), so the SUBSTANTIVENESS arm of that function MUST be re-keyed onto the
INSTALLED pack: the substantiveness capability is Present/Working iff the substantiveness
pack is installed / resolvable. With the pack installed → the dimension RUNS; without it AND
undeclared → class-2 capability-absent (warn, exit 0); without it AND declared → class-3
declared-intent-unmet (block). The re-key is **substantiveness-dimension-only**: the
coverage and contracts arms of `deriveCapabilityState` stay on their existing keying
unchanged (coverage was descoped — BUNDLE-009 REQ-009; contracts keeps its baked analyzer
until SPEC-038/Seed 4 ships its pack, so re-keying it now would mark it absent before its
pack exists). The shipped SPEC-036 test
`TestCapabilityState_NonGoProject_DerivesAbsentClass2`
(`cmd/backstop/gate_capability_test.go:17`) asserts the OLD substantiveness keying and goes
RED on `./cmd/backstop/` once the re-key lands; it is MIGRATED as part of the re-key (its
substantiveness arm updated, its coverage/contracts arms unchanged — REQ-008 / CLM-037). This
is an open, intended alignment of SPEC-036's derivation via implementation, not a silent
drift (per align-predating-artifacts; SPEC-036 itself is not revised — see Version History
and Sharp Edges).

### Real over-installed-pack end-to-end proof (REQ-010)

REQ-005's spy proves the substantiveness STEP calls the dispatch seam; it does NOT prove
the whole pipeline runs over a real INSTALLED pack. REQ-010 adds that proof: a test
INSTALLS the substantiveness pack through the distribution path REQ-009 mandates — a LOCAL
source being the appropriate choice for a hermetic test workspace — and runs the REAL gate path
end to end — real pack resolution → real `dispatchPackEngines` → real ast-grep over real
fixtures → real convert (ast-grep→SARIF via the convert script under the real macOS
sandbox) → SARIF → gate route + set-join — asserting a genuinely hollow backstop test
yields a REAL `test_substantiveness` violation. It MUST FAIL if the pack is not actually
installed (no local declaration/lock) or not actually run (no real dispatch), so it is
unsatisfiable by a stub, by testdata-pointed-at-by-production, or by the seam spy alone.
The substrate now makes this runnable: multi-rule ast-grep packs dispatch for real
(ISSUE-028 — the substantiveness pack carries 2 ast-grep rules, Q1 hollow + Q2 extraction)
and convert scripts run under the real macOS sandbox (ISSUE-029). The REQ-005 seam-spy
claims are retained alongside this proof.

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

- `TargetPackageName(subject)` reduces the declared subject to an OPAQUE target token —
  its last `/`-segment (`filepath.Base`), with NO cmd//pkg/ layout knowledge (ISSUE-047
  removed the layout special-casing): `cmd/backstop`→`backstop`, `pkg/gate`→`gate`,
  `internal/foo`→`foo`, and a bare token passes through unchanged. The ONLY special case
  is the EMPTY subject, which returns `""` (the empty-target token the set-join SKIPS).
  Carried forward from the deleted analyzer's `targetPackageName`, now as a pure
  language-agnostic last-segment reduction.
- `NoTargetViolation(funcName, targetPkg, referenced, samePackage)` returns a violation
  iff `targetPkg` is non-empty AND `!samePackage` AND `targetPkg` is NOT a member of the
  `referenced` set. The complete decision table:

| targetPkg | samePackage | target ∈ referenced set | → noTarget? |
|-----------|-------------|-------------------------|-------------|
| "" (empty subject) | n/a | n/a | NO — skipped |
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
`[]gate.Violation` that carries NEITHER a discriminator fine-grained enough to separate the
Q1 hollow findings from the Q2 extraction findings NOR per-test finding identity.
`gate.Violation` carries a `GateType` field as of ISSUE-118 (`check.Violation` still does
not), but it is stamped from the PRODUCING ENGINE BINDING and the substantiveness pack
declares both rules under a SINGLE binding — so every substantiveness finding carries the
same gate_type and the gate still cannot route or key substantiveness findings by it. Two
gate-side, language-agnostic
helpers in `pkg/gate/substantiveness_join.go` consume the flat pack SARIF:

- **Routing by namespaced rule ID** — `RouteSubstantivenessFindings(violations,
  hollowRuleID, extractionRuleID)` partitions the flat `pack_engines` stream into
  hollow-findings and extraction-findings by matching the substantiveness pack's stable,
  namespaced (`pack.NamespacedRuleID`) rule IDs on each violation's `Rule`. All other pack
  rules are ignored. No `gate_type` field is consulted (it is per-BINDING and cannot
  separate the two roles).
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
   (hollow / substantive / no-target / calls-target / same-package). The pack is an
   ORDINARY pack (no `//go:embed`, no baked tier); dogfood-install it into backstop-core
   via the standard distribution path `pack add` — EITHER a local source (declared `local`
   in backstop.yml + `local` lockfile entry; `VerifyLock` skips local packs) OR a published
   remote coordinate (`git` lockfile entry with source coordinate + tag) — so backstop-core
   gates itself on substantiveness through the installed, declared, locked pack (REQ-009).
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
   `./cmd/backstop/`. Re-key the SUBSTANTIVENESS arm of `deriveCapabilityState`
   (`cmd/backstop/gate.go:272`, the SPEC-036 function) onto the INSTALLED pack
   (Present/Working iff the pack is installed/resolvable), NOT the deleted baked-Go-analyzer
   presence — leaving the coverage and contracts arms UNCHANGED (REQ-009 / CLM-035 /
   CLM-036). In the same change, MIGRATE the shipped SPEC-036 test
   `TestCapabilityState_NonGoProject_DerivesAbsentClass2`
   (`cmd/backstop/gate_capability_test.go:17`): update its substantiveness arm to the
   installed-pack keying, leave its coverage/contracts arms untouched, so `./cmd/backstop/`
   stays green (REQ-008 / CLM-037).
6a. Add the REAL over-installed-pack end-to-end test (REQ-010): install the substantiveness
   pack via the distribution path (a LOCAL source, for a hermetic test workspace — REQ-009
   accepts either source type), run the REAL gate path end to end (real pack resolution → real
   `dispatchPackEngines` → real ast-grep → real convert-under-sandbox → SARIF → route +
   set-join) over a genuinely hollow backstop test, and assert a REAL `test_substantiveness`
   violation — failing if the pack is not actually installed or not actually run (no stub,
   no testdata-pointed-at-by-production, not the seam spy alone).
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
the-pack-path preservation claim, and a migrated-shipped-SPEC-036-test claim
(`TestCapabilityState_NonGoProject_DerivesAbsentClass2` re-keyed for substantiveness, its
coverage/contracts arms untouched, so `./cmd/backstop/` stays green). The wiring is covered by a spy on the REAL
`dispatchPackEnginesFn` seam proving the substantiveness step reaches the dispatcher, a
negative spy proving an unwired/baked path fails, and a claim that no baked analyzer
delegate is invoked. The strangler-equivalence is covered one claim per verdict-matrix cell
(hollow, substantive, no-target, calls-target, same-package) plus an unsatisfiable-by-stub
claim. The TS proof claims are unsatisfiable by a stub (real ast-grep over real fixtures,
riding the shared dispatch path).

The PROVISIONING model is covered by a not-embedded-nor-testdata claim and an
installed-via-the-distribution-path claim (declared in backstop.yml + locked with a
resolvable source and lock verification passing — satisfied by EITHER a `local` entry, which
VerifyLock skips so no remote artifact is needed, or a `git` entry carrying source coordinate
+ tag); the substantiveness CAPABILITY-keying is covered by a claim pinning the
capability source to the INSTALLED pack at the live locus `deriveCapabilityState`
(`cmd/backstop/gate.go:272`) (Present/Working iff installed; undeclared+absent → class-2,
declared+absent → class-3 per SPEC-036), NOT the deleted baked analyzer, plus a
dimension-asymmetry claim pinning that ONLY the substantiveness arm re-keys (coverage and
contracts arms unchanged). The REAL
over-installed-pack END-TO-END proof is covered by a hollow-yields-real-violation-through-
the-whole-pipeline claim, a no-vacuous-green claim (uninstalled/unrun → no violation, so it
cannot pass without the real installed pack), and a claim that the proof exercises both the
multi-rule ast-grep dispatch (ISSUE-028) and the sandboxed convert (ISSUE-029). These E2E
claims are, by construction, unsatisfiable by a stub, by testdata-pointed-at-by-production,
or by the seam spy alone.

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

- **The dispatch returns the WRONG shape — a role-blind, identity-less stream.**
  The live dispatch (`dispatchPackEngines` → `runFindingsEngine` → `ParsePackFindings`)
  flattens SARIF into a flat `[]gate.Violation` that carries no ROLE discriminator and
  discards which test a symbol belongs to. `gate.Violation` gained a `GateType` field in
  ISSUE-118 (`check.Violation` still has none), but it is stamped from the PRODUCING
  ENGINE BINDING, and the substantiveness pack declares Q1 and Q2 under a SINGLE binding —
  so it is uniform across every substantiveness finding. Two traps follow: (1) a
  reviewer/implementer who tries to ISOLATE the Q1 and Q2 partitions by `gate_type` will
  find the field uniform and useless for that cut — routing MUST be by
  namespaced rule-ID convention (`RouteSubstantivenessFindings`); (2) an implementer who
  tries to recover per-test identity by re-walking the test AST gate-side would re-bake the
  exact spec-aware gate-side analyzer Sharp Edge #2 forbids — keying MUST come from the
  extraction rule's own SARIF (file + enclosing function in each result) joined by
  `(FilePath, FuncName)` in `ReferencedSetForTest`. The "dispatch later changed to carry
  gate_type natively" enhancement anticipated here HAS since landed — ISSUE-118 added
  `gate.Violation.GateType`, stamped per-violation from its producing binding. It does NOT
  displace this routing: being per-BINDING, it is uniform across Q1 and Q2 and so cannot
  make the role cut this seam needs.

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

- **`TargetPackageName` is a pure opaque-token reduction — the cmd//pkg/ layout
  derivation was DELIBERATELY removed (ISSUE-047), not a regression.** The deleted
  analyzer's `targetPackageName` baked this-repo layout knowledge (empty `""` for `cmd/`
  and non-`pkg/` paths, last component only for `pkg/...`). That layout special-casing
  was a baked-repo-layout assumption and was REMOVED: `TargetPackageName` now reduces an
  OPAQUE subject to its last `/`-segment (`filepath.Base`) with no layout branching, so a
  `cmd/...` subject now yields a REAL leaf token (`cmd/backstop`→`backstop`), NOT `""`.
  This is an INTENDED behavior change — a `cmd/` path producing a non-empty target is now
  CORRECT, not a regression. The empty-target SKIP is preserved but keyed on an EMPTY
  SUBJECT (a spec/claim that declares no callable subject), never on a cmd//pkg/ layout
  classification. CLM-009 pins the empty-subject skip; CLM-028 pins the opaque last-segment
  reduction. A reviewer must NOT "restore" the cmd//pkg/ empty-target derivation — that
  would re-bake the repo-layout assumption ISSUE-047 eradicated.

- **The testdata-as-production trap (the recurring pack-provisioning integration gap).**
  Every prior pack-migration impl in this codebase passed its tests by pointing at a
  `testdata` pack or stubbing the dispatcher, leaving the REAL installed-pack path
  unproven — and that gap bit earlier seeds. A green Q1/Q2 unit suite over a `testdata`
  pack does NOT prove backstop gates itself on substantiveness via an INSTALLED pack. The
  guard is REQ-010's over-installed-pack E2E (CLM-032..034): `pack add` the pack (a LOCAL
  source, for a hermetic test workspace) and run the WHOLE production pipeline, with a
  no-vacuous-green negative (uninstalled/unrun → no violation). If the only "real" path in
  the suite resolves the pack from `testdata`, the gap is re-opened — production must resolve
  the pack from the INSTALLED declaration (local or remote), never from `testdata`.

- **`//go:embed` / baked-tier temptation.** It is tempting to bundle the substantiveness
  rule YAML into the binary (embed, or a privileged "backstop ships built-in rules" tier)
  "so the dogfood always works." That is exactly the baked tier the zero-baked-checks rule
  eradicates and REQ-009 prohibits: the binary ships ONLY a way to install + execute packs.
  The dogfood works because backstop-core INSTALLS the pack through the distribution path
  (local OR remote source — REQ-009 accepts either), not because the rules are compiled in.
  CLM-030 pins no-embed / no-testdata-in-production; CLM-031 pins installed-and-resolvable.

- **The source-type loosening is NOT a loosening of the anti-baking invariant (v1.2.4).**
  REQ-009/CLM-031 originally hard-required a LOCAL install because that was the only
  distribution path that existed when this spec was written; backstop-core's own
  `go-substantiveness` pack now installs REMOTELY (`source_type: git`, coordinate
  `backstop-ai/go-substantiveness`, tag `v1.2.0`) after the fleet-wide migration to published
  coordinates. The requirement was amended to accept EITHER source type. The trap: reading
  that amendment as general permission to relax provisioning. It is not — `//go:embed`, a
  compiled-in asset, a baked analyzer bridge, and any production path resolving the pack from
  `testdata` remain ABSOLUTELY prohibited, exactly as strictly as before. The ONLY thing that
  moved is `local` → `local OR remote`; "installed, declared, locked, resolvable" is still
  the bar, and a pack that is none of those still fails REQ-009.

- **A green local-install test does NOT prove backstop-core's LIVE provisioning (v1.2.4).**
  The mandated provisioning tests build scratch/temp projects (`t.TempDir()`, a separate E2E
  workspace helper) and never read backstop-core's own `backstop.yml` / `backstop.lock` —
  which is exactly how CLM-031 stayed green for months while backstop-core's actual
  substantiveness pack silently migrated from `local` to `git` in commit `905120f`. Their
  green proves the MECHANISM works, not that backstop-core uses it that way. Any future
  claim intended to pin backstop-core's OWN dogfood state must read the repo's live
  declaration/lock, not a fixture — otherwise the same drift recurs undetected.

- **Capability-keying drift against SPEC-036 (open alignment, not silent).** SPEC-036 keys
  the substantiveness `CapabilityState` on `cfg.Language` + baked-Go-analyzer presence — the
  only capability that existed on the no-pack binary — at the shipped function
  `deriveCapabilityState` (`cmd/backstop/gate.go:272`). This spec DELETES that analyzer, so
  leaving the capability keyed on baked-analyzer presence would make substantiveness read as
  permanently capability-ABSENT even when the pack is installed and working (a false class-2,
  silently un-enforced). The substantiveness arm MUST move to the INSTALLED pack (Present/
  Working iff resolvable), changing `deriveCapabilityState` at that exact locus. This is an
  intended, openly-recorded alignment of SPEC-036's derivation via implementation (per
  align-predating-artifacts; SPEC-036 itself is NOT revised here) — the impl/planner must
  update the derivation, not preserve the stale baked-analyzer keying. CLM-035 pins it.

- **Dimension-asymmetry — re-key SUBSTANTIVENESS ONLY, never coverage/contracts.**
  `deriveCapabilityState` serves all three traceability dimensions, and the shipped test
  iterates all three. The temptation is to re-key the whole function onto installed packs.
  That is WRONG and would break the gate: coverage was descoped (its analyzer is deleted
  with no in-bundle replacement — BUNDLE-009 REQ-009), and contracts keeps its baked analyzer
  until SPEC-038/Seed 4 ships its pack, so re-keying contracts now would mark it
  capability-absent before its pack exists. ONLY the substantiveness arm re-keys to the
  installed substantiveness pack; the coverage and contracts arms stay on their existing
  keying unchanged. CLM-036 pins the asymmetry; the migrated shipped test (CLM-037) leaves
  its coverage/contracts arms untouched.

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
- Is the empty-target case (keyed on an EMPTY subject, NOT a cmd//pkg/ layout
  classification) SKIPPED (no violation) and the same-package case SATISFIED, matching the
  deleted analyzer's short-circuits — and does `TargetPackageName` reduce a non-empty
  subject to its last path segment as an opaque token (no cmd//pkg/ layout special-case
  per ISSUE-047), so the eradication de-bakes the repo-layout assumption without dropping
  the empty-subject skip?
- Are the TS hollow-test claims satisfied by REAL ast-grep over REAL `.test.ts` fixtures
  riding the shared dispatch path, unsatisfiable by a stubbed pack output, and authored
  into the SAME TS proof pack SPEC-038 shares?
- Does `buildTestSubstantivenessStep` reach the substantiveness pack through the EXISTING
  `resolveDispatchPackEngines()` / `dispatchPackEnginesFn` seam (the same one code check
  and the `pack_engines` step use) rather than re-implementing dispatch, and does the spy
  sit on that REAL seam (not a parallel stub)?
- Are substantiveness findings isolated out of the flat `pack_engines` `[]gate.Violation`
  stream by NAMESPACED rule-ID convention (NOT by a gate_type field, which is per-BINDING
  and therefore uniform across the two roles), and are extraction findings keyed back to a `MandatedTest` by
  `(FilePath, FuncName)` from the pack's own SARIF — with NO gate-side test-AST re-walk?
- Is the substantiveness pack an ORDINARY INSTALLED pack — with NO `//go:embed` / baked
  tier and NO production code path resolving it from `testdata` — and does backstop-core
  dogfood-install it into itself through the standard `pack add` distribution path, declared
  in backstop.yml and locked with a RESOLVABLE source that lock verification passes (either
  a `local` entry, which VerifyLock skips, or a `git` entry with source coordinate + tag)?
- Does any check of the provisioning model assert on SOURCE TYPE where it should be
  asserting on INSTALLED-AND-RESOLVABLE — i.e. would a correctly-installed REMOTE pack fail
  a test that should pass, or would a `//go:embed`ed / testdata-resolved pack sneak past a
  test that only checks "some lock entry exists"?
- Is there a REAL over-installed-pack END-TO-END test that installs the pack as a local
  pack and runs the WHOLE production pipeline (real pack resolution → real
  `dispatchPackEngines` → real ast-grep → real convert-under-sandbox → SARIF → route +
  set-join) over a genuinely hollow test, asserting a REAL violation — and does it FAIL when
  the pack is NOT installed or NOT run (not a stub, not testdata-in-production, not the seam
  spy alone)? Does it exercise both multi-rule dispatch (ISSUE-028) and sandboxed convert
  (ISSUE-029)?
- Is the substantiveness `CapabilityState` keyed on the INSTALLED pack (Present/Working iff
  resolvable; undeclared+absent → class-2, declared+absent → class-3 per SPEC-036), and NOT
  on the deleted baked-Go-analyzer presence — so an installed-and-working pack is not
  mis-classified as capability-absent?
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
  locus the pack path replaces the analyzer at), and `deriveCapabilityState` (line 272, the
  SPEC-036 function whose SUBSTANTIVENESS arm re-keys onto the installed pack — REQ-009).
- SPEC-035 — pack-declared engines + trusted-tool allowlist + `pattern-arg` input mode +
  gate-TYPE binding (`GateTypeSubstantiveness`); the substrate this pack binds ast-grep
  through.
- BUNDLE-010 / SPEC-033 — the shipped ast-grep engine + reusable ast-grep→SARIF converter
  + engine-organized pack layout this spec's rules ride.
- SPEC-034 — the strangler licensing pattern (prove equivalence on real fixtures, then
  delete) this spec repeats for substantiveness.
- SPEC-038 (Seed 4, contracts) — shares the single TypeScript proof pack this spec adds
  substantiveness rules to.
- `pkg/pack/distribution/{add,install,verify}.go` — the pack provisioning path (REQ-009).
  LOCAL: `Add` records the `local` source in backstop.yml + lockfile and `VerifyLock` SKIPS
  `local` packs (verify.go line ~46–49), so a local source needs no remote artifact. REMOTE:
  `Add` resolves the version, clones at the tag through the identity gate, and records a
  `git` lock entry with `source_coordinate` / `git_ref` / `content_hash`. REQ-009 accepts
  either — see SPEC-055 (remote pack assembly) and SPEC-056 (manifest name as install
  identity) for the remote path.
- `backstop.yml` / `backstop.lock` (live, as of 2026-08-10) — backstop-core's own
  `backstop-ai/go-substantiveness` is installed REMOTELY (`source_type: git`, `git_ref:
  v1.2.0`, `source_coordinate: backstop-ai/go-substantiveness`), migrated from the original
  local install by commit `905120f` (fleet-wide move to published coordinates). This is the
  reality v1.2.4 amended REQ-009/CLM-031 to describe.
- SPEC-036 — shipped `deriveCapabilityState` (`cmd/backstop/gate.go:272`) deriving the
  `CapabilityState` from `cfg.Language` + baked-Go-analyzer presence, and the shipped test
  `TestCapabilityState_NonGoProject_DerivesAbsentClass2`
  (`cmd/backstop/gate_capability_test.go:17`). This spec deletes that analyzer and re-keys the
  SUBSTANTIVENESS arm onto the INSTALLED pack (REQ-009 / CLM-035 / CLM-036), migrating that
  shipped test's substantiveness arm (REQ-008 / CLM-037); coverage/contracts arms unchanged.
  An openly-recorded alignment via implementation — SPEC-036 itself is NOT revised.
- ISSUE-028 — multi-rule ast-grep packs now dispatch for real (the substantiveness pack's
  Q1 hollow + Q2 extraction rules both dispatch), unblocking the real end-to-end path.
- ISSUE-029 — convert scripts (ast-grep→SARIF via jq) now run under the real macOS sandbox,
  unblocking the real convert step in the end-to-end pipeline.

## Version History

- **1.2.9** (2026-08-16) — **PROSE-ONLY correction: the `GateType` non-existence assertions
  are now false.** PLAN-ISSUE-118 added a `GateType` field to `gate.Violation`
  (`pkg/gate/result.go`), stamped per-violation from its PRODUCING ENGINE BINDING. Eight
  places in this spec asserted, as standing fact or as the rationale for rule-ID routing,
  that no such field exists. All eight are corrected here to the rationale that SURVIVES the
  change: `GateType` is per-BINDING, and the substantiveness pack declares the Q1 hollow rule
  and the Q2 extraction rule under a SINGLE `gate_type: substantiveness` ast-grep binding, so
  the field is UNIFORM across both roles and can never make the partition REQ-007 needs.
  Corrected: REQ-007's text (2 spots), CLM-024's rationale aside, the Requirements-section
  summary, the REQ-007 Implementation subsection (2 spots), Sharp Edge 5, and the REQ-007
  Review Question. Sharp Edge 5's closing hypothetical ("if the dispatch is later changed to
  carry gate_type natively, that is a separate enhancement") is now SATISFIED, not falsified,
  and is rewritten to record that ISSUE-118 IS that change and why it does not displace the
  routing. The 1.1.0 history entry keeps its original claim, re-framed as true AT THE TIME
  with a pointer here — history is not rewritten.
  **Explicitly NOT changed.** No requirement's testable assertion, no claim verdict, and no
  mandated test name — CLM-024 in particular keeps
  `TestRoute_PartitionsSubstantivenessByRuleID_FromFlatStream` verbatim and is neither
  weakened, renumbered, nor retired. All 37 claims and 37 mandated test names stand exactly
  as 1.2.8 left them. Also deliberately left alone: the ISSUE-064 residue in which several
  spots (CLM-024 included) still describe routing as "by NAMESPACED RULE-ID convention" while
  the shipped `RouteSubstantivenessFindings` switches on the `substantiveness_role` PROPERTY.
  That drift PREDATES this correction, is a separate issue, and was neither fixed nor
  re-asserted as accurate while editing the text around it.
  **Not covered by this spec edit.** Five sibling docstrings in `pkg/gate` carry the same
  falsified assertion (`substantiveness_join.go` x3, `substantiveness_join_test.go`,
  `substantiveness_hollow_violation_test.go`); they are source comments, outside a spec
  author's write scope, and are handed off separately.

- **1.2.8** (2026-08-15) — **CLOSE-OUT: status `draft` -> `implemented`**, plus the one
  accuracy fix the flip forced. No requirement, claim, mandated test, or mechanism is added,
  removed, or reworded; all 37 claims and 37 mandated test names stand exactly as 1.2.7 left
  them, and NO source file changed. The evidence below was RE-VERIFIED in this session rather
  than copied forward from the implementation report.
  **Lineage.** This spec's implementation is complete across all phases, ending with phase-6a
  (TASK-550/551), which landed CLM-031's mandated-test rename — the OPEN FOLLOW-ON recorded in
  1.2.4 and the last outstanding item of the eleven-claim subject-join fix chain that 1.2.5,
  1.2.6 and 1.2.7 worked through. Phase-6a took TWO independent impl-review passes. Round 1
  found one real defect, and a diagnostic one rather than a mechanical slip: a `*string`
  formatted with a bare `%v` on its mismatch branch, so the failure message printed a POINTER
  ADDRESS instead of the value that mismatched — a test that fails uselessly. It was fixed
  (commit 4906704), and the same shape was found at three OTHER pre-existing sites elsewhere
  in the codebase and filed as ISSUE-132 rather than absorbed here; those three are explicitly
  NOT this spec's to fix. Round 2 confirmed clean.
  **Build and tests.** `go build ./...` and `go vet ./pkg/gate/ ./cmd/backstop/` are clean.
  This spec's own declared `test_command` — `go test ./pkg/gate/ ./cmd/backstop/ -race` — was
  RUN AT CLOSE-OUT and exits 0 with zero failures: `pkg/gate` ok in 47.9s, `cmd/backstop` ok in
  209.5s.
  **Coverage.** Measured at close-out against this spec's declared 80 floor, from that same
  run: `pkg/gate` 93.4%, `cmd/backstop` 91.9%. Both clear it.
  **Mandated tests.** All 37 test names in this spec's `claims` block were confirmed PRESENT in
  the tree BY NAME at close-out, by scanning for each `func <name>(` rather than trusting the
  implementation report — 26 in `pkg/gate`, 11 in `cmd/backstop`, zero missing. That scan is
  what confirms CLM-031 now resolves to the RENAMED function
  `TestProvisioning_SubstantivenessInstalledViaDistributionPath_LocalOrGit_DeclaredAndLocked`
  and that the stale `...AsLocalPack` name 1.2.7 removed survives nowhere.
  **Claim subjects — the defect class this spec's own close-out surfaced, now discharged.**
  Every one of the 37 claims was re-audited against the subject join at close-out. Eleven carry
  an explicit `subject: cmd/backstop` (CLM-015/016/017, CLM-030/031, CLM-032/033/034,
  CLM-035/036/037) and all eleven of their mandated tests do live in `cmd/backstop`; the other
  26 carry no `subject:` and correctly inherit the spec-level `implementation.subject:
  pkg/gate`, where all 26 of their tests live. No claim in this spec carries `kind: absence`,
  so not one of the 37 is exempt from the join — every one is really enforced from this flip
  onward. This is the SECOND spec in which the class was found and the one that made it
  generalizable: a claim with no explicit `subject:` silently inherits the spec-level package,
  and `test_substantiveness` filters `draft` specs out through `ContractsAreDue` before the
  join ever runs, so a wrong inherited subject is STRUCTURALLY INVISIBLE until the
  `implemented` flip. Here it hid ten real `noTarget` violations (1.2.5) behind an eleventh
  that passed only INCIDENTALLY, on a symbol-reference branch that a refactor could have
  removed (CLM-035, 1.2.6). Two further traps make it worse than it looks: only
  package-qualified CALL/selector references are extracted, so composite literals, constants
  and type positions do NOT rescue a mis-subject; and `kind: absence` claims skip the join
  entirely, so they "pass" without being checked. The durable lesson — detect this by forcing
  `implemented` on an ISOLATED SCRATCH COPY and running the real join, never by flipping the
  live tree to find out — is recorded at
  `.claude/agent-memory/spec-author/feedback_omitted_subject_inherits_wrong_package.md`.
  **The accuracy fix the flip forced: four contract signatures had drifted from the shipped
  source.** Contract declarations are collected only for `implemented` specs, so while this
  spec sat at `draft` its `contracts` block was never enforced at all. Re-verifying it BY HAND
  against the real source — which is the only way, because a `kind: function` contract entry
  compiles to an EXISTENCE-ONLY query that confirms the symbol exists and never compares
  parameter or return lists — found four declared signatures stale. `TargetPackageName`'s
  parameter was still named `implementationPackage` (the source says `subject`, renamed when
  ISSUE-047 de-baked the repo-layout assumption); `RouteSubstantivenessFindings` still declared
  two `hollowRuleID, extractionRuleID string` parameters it no longer takes;
  `buildTestSubstantivenessStep` was missing the `classifier gate.SourceClassifier, matcher
  gate.TestNameMatcher` parameters SPEC-045 added; and `deriveCapabilityState` still declared
  `cfg *config.Config, dim` where the source takes `packs []*pack.Manifest, dim, stack string`
  after SPEC-046. All four are corrected here, SPEC-TEXT ONLY — no source moved to meet the
  spec. Because the check is existence-only, none of these would EVER have gone red; they are
  fixed because they were wrong, not because anything failed.
  **Gate (DIFF-SCOPED), run AFTER the flip.** The bare `./bin/backstop gate` — diff vs
  merge-base plus untracked, the scope actually verified for this close-out — exits 0 with
  every blocking dimension green: `pack_lock_verification`, `artifact_validation`,
  `pack_engines`, `test_verification`, `test_substantiveness`, `coverage_threshold`,
  `contract_signature`, `artifact_status_drift`, `requirement_traceability` and
  `waiver_resolution`, all with ZERO violations. `test_substantiveness` and
  `contract_signature` are the two that only activate at `implemented`, so this post-flip run
  is the first real reading either has ever given on this spec — a pre-flip green would have
  proven nothing about either, which is exactly why the flip was applied first and the gate run
  second. `./bin/backstop gate --file cmd/backstop/gate_substantiveness_provisioning_test.go`
  likewise exits 0. Every residual finding is a non-blocking warning and each was ATTRIBUTED
  rather than waved past: 173 `requirement_traceability_advisory` and 2
  `artifact_status_drift_advisory`, this project's standing repo-wide advisories, and neither
  drift advisory names this spec. Following SPEC-068 1.2.9, SPEC-069 1.3.4 and SPEC-070 1.1.5,
  this is deliberately NOT a claim about `./bin/backstop gate --all`, which carries standing
  pre-existing debt unrelated to this spec. `./bin/backstop artifact validate --spec SPEC-037`
  passes at `implemented`.
  **A verification-methodology note worth carrying forward.** The first post-flip gate run
  exited 2 at `pack_engines` — `go-arch-lint` "not found on PATH" — and the gate ABORTED there,
  never reaching `test_substantiveness` or `contract_signature`. The tool was in fact installed
  the whole time; `$(go env GOPATH)/bin` simply was not on the invoking shell's PATH. A
  Layer-0 assume-present tool missing from PATH therefore masks every downstream dimension
  behind a failure that looks nothing like a spec defect, and a close-out that stopped at that
  exit-2 would have recorded either a false defect or, worse, no reading at all for the two
  dimensions the flip exists to activate.
  **What the flip closed.** BUNDLE-009 `stack-aware-traceability` REQ-002, REQ-003, REQ-007,
  REQ-008 and REQ-010 — this spec's five `supports` targets — now have implemented-spec
  coverage and have dropped OUT of the traceability advisory set, confirmed against the
  post-flip run. Five of the bundle's ten requirements remain uncovered and none are this
  spec's to close: REQ-001 (SPEC-036, still `draft`), REQ-004/005/006 (SPEC-038, still
  `draft`), and REQ-009, the COVERAGE arm, which BUNDLE-009 descoped with no in-bundle
  replacement and which no spec owns. REQ-007 and REQ-010 are shared with SPEC-038; this flip
  closes them, and SPEC-038's own remaining seam is unaffected.
  **Known open items, expected and NOT resolved here.** (1) `PLAN-SPEC-037` remains at `draft`
  and its close-out is a separate, still-pending action — this entry asserts nothing about the
  plan's status, and the plan is one of the two `artifact_status_drift_advisory` entries above.
  (2) ISSUE-132 is OPEN for the three sibling `%v`-on-pointer diagnostic sites outside this
  spec's scope. (3) SPEC-046 (`implemented`) and SPEC-038 (`draft`) BOTH declare
  `deriveCapabilityState` with the same stale `cfg *config.Config` signature this revision just
  corrected here. Two specs declaring one symbol is an accepted pattern in this corpus, but it
  means a shape change must edit every declaring spec or the untouched ones rot silently with
  no red anywhere — which is precisely what happened. Editing another spec's contracts block is
  outside this close-out's scope, so the divergence is SURFACED here rather than absorbed, and
  is owed a follow-on.
- **1.2.7** (2026-08-15) — **CLOSES THE CLM-031 RESIDUE carried forward since 1.2.4/1.2.5/1.2.6 —
  both halves, together, in one edit.** CLM-031 was the last of the eleven `cmd/backstop`-resident
  claims still inheriting the spec-level `implementation.subject: pkg/gate`, and the last carrier
  of the stale `AsLocalPack` mandated-test name that 1.2.4's amendment left contradicting its own
  claim text. Two changes, both to CLM-031's metadata only:
  (1) **`subject: cmd/backstop` added.** Same defect and same fix as the nine siblings amended in
  1.2.5 and the tenth (CLM-035) in 1.2.6. CLM-031's mandated test has NO join branch to the
  inherited `pkg/gate` subject in either direction: its file's directory leaf is `backstop`, not
  `gate`, so colocation fails, and its PACK-EXTRACTED referenced-symbol set is
  `{add, distribution, filepath, os, strings, t}` — no `gate` reference at all, so the symbol
  branch fails too. Unlike CLM-035's incidental pass, this one had no surviving branch: it was one
  of the 10 blocking `test_substantiveness` noTarget violations the 1.2.5 scratch-copy flip
  simulation reproduced. What CLM-031 actually pins — the `pack add` distribution path,
  backstop.yml declaration, lockfile entry and `VerifyLock` — is exercised from `cmd/backstop` by
  construction, so `cmd/backstop` is the correct subject, not a workaround.
  (2) **Mandated test renamed** `TestProvisioning_SubstantivenessInstalledAsLocalPack_DeclaredAndLocked`
  → `TestProvisioning_SubstantivenessInstalledViaDistributionPath_LocalOrGit_DeclaredAndLocked`,
  discharging the OPEN FOLLOW-ON recorded in 1.2.4. The new name maps 1:1 onto the claim text as
  amended in 1.2.4 — distribution-path install, EITHER source type, declared + locked +
  VerifyLock passing — with no over- or under-claiming; the old name asserted `local`
  specifically, which the amended claim no longer requires. The rename alone would NOT have
  cleared the noTarget violation (the extracted symbol set is a property of the test BODY, not of
  the function name), which is exactly why 1.2.5 recorded that whichever revision closes the
  rename must land the `subject:` in the same edit. It does, here.
  **What this revision does NOT do: the test body is untouched.** This is a spec-text-only edit,
  identical in kind to 1.2.5 and 1.2.6 — it corrects the spec's own claim METADATA to match the
  test name the implementation will land under. Rewriting the test body (and renaming the function
  in `cmd/backstop`) is the plan's work, owned by PLAN-SPEC-037 phase-6a TASK-550/551, and has not
  yet been executed. Until that lands, the spec names a test the tree does not yet declare — the
  expected and intended intermediate state, invisible to the gate because `ContractsAreDue`
  filters a `draft` spec's mandated tests out of the join entirely.
  **No requirement, contract, mechanism, claim text or claim count changes.** No waiver was taken,
  no test was weakened, `status` unchanged (`draft`). With this revision no affected claim carries
  an unresolved `subject:` or a stale mandated-test name, so nothing in this family now blocks the
  eventual promotion to `implemented`.
- **1.2.6** (2026-08-15) — **HARDENING (PREVENTIVE, NOT CORRECTIVE): CLM-035's passing join was
  incidental, and is now structural.** Nothing was broken before this revision and nothing is
  fixed by it. CLM-035's mandated test
  `TestCapability_SubstantivenessKeyedOnInstalledPack_NotBakedAnalyzer` joined successfully under
  the inherited spec-level `implementation.subject: pkg/gate` and would have continued to; the
  only edit is a `subject: cmd/backstop` line on CLM-035, matching the nine siblings amended in
  1.2.5. No requirement, contract, mechanism or mandated-test NAME changes, and no test file is
  touched.
  **The fragility this closes.** 1.2.5 deliberately left CLM-035 alone because its join passed on
  the OR-branch the other nine failed: the subject join is satisfied by colocation with the
  subject package OR by the test's own PACK-EXTRACTED referenced-symbol set naming it, and this
  test's body happens to make real `gate.`-qualified calls (`gate.ClassifyDimension`,
  `gate.DimensionSubstantiveness`) that the extraction rule records. That is an artifact of the
  test body, not of what the claim pins. The claim's actual subject is `deriveCapabilityState`
  (`cmd/backstop/gate.go`), and the test file is `cmd/backstop/gate_capability_rekey_test.go` —
  both `cmd/backstop`, neither `pkg/gate`. A refactor that routed those assertions through a
  local helper instead of calling `gate.` directly would drop the only surviving branch and
  silently reintroduce the exact `noTarget` failure 1.2.5 corrected for the siblings — invisible
  until the spec flips to `implemented`, since `ContractsAreDue` filters a `draft` spec's tests
  out of the join entirely.
  **What changes.** The join basis moves from an incidental symbol reference to structural
  colocation with the package the claim actually targets, so it no longer depends on any
  particular call surviving inside the test body. The claim text, its mandated test, that test's
  assertions and its rigor are untouched; no waiver was taken and no test was weakened.
  **Residue unchanged.** CLM-031 remains the one affected claim still carrying no `subject:`, for
  the reason recorded in 1.2.5 — whichever plan closes its mandated-test rename must also add
  `subject: cmd/backstop` to it, or this spec still cannot promote to `implemented`.
- **1.2.5** (2026-08-15) — **ACCURACY FIX: nine claims declared a subject their mandated tests
  do not sit in.** No requirement, contract, mechanism or mandated-test NAME is added, removed
  or reworded, and no test file changed — the only edit is a `subject:` line on nine claims.
  **The defect.** No claim in this spec carried an explicit `subject:`, so all of them inherited
  the spec-level `implementation.subject: pkg/gate` → target token `gate`. That is correct for
  the 26 mandated tests that live in `pkg/gate` (satisfied by colocation), but 11 of this spec's
  37 mandated tests live in `cmd/backstop` instead, and 10 of those 11 do not reference `gate`
  in the PACK-EXTRACTED referenced-symbol set. The subject join
  (`gate.NoTargetViolationForTest` → `gate.NoTargetViolation`,
  `pkg/gate/substantiveness_join.go`, over `testFileColocatedWithTarget`,
  `cmd/backstop/gate.go`) is satisfied by colocation with the subject package OR by the test's
  own extracted symbol set naming it, and neither held: those files' directory leaf is
  `backstop`, not `gate`, and the extraction rule (`referenced-symbol-go.yml` in
  `backstop-ai/go-substantiveness`) records only package-qualified CALL/selector references — a
  TYPE-position mention such as the closure parameter `_ *gate.GateScope` in
  `TestWiring_SubstantivenessStepRoutesThroughDispatchSeam` is not recorded, even though that
  test genuinely exercises the gate. The eleventh, CLM-035's
  `TestCapability_SubstantivenessKeyedOnInstalledPack_NotBakedAnalyzer`, does make a real
  `gate.`-qualified call and so joins successfully; it is left as-is.
  **Why this was invisible.** `test_substantiveness` filters mandated tests through
  `ContractsAreDue` before the join (`buildTestSubstantivenessStep`, `cmd/backstop/gate.go`), so
  a `draft` spec's tests never enter it. Confirmed by simulating the flip on an ISOLATED SCRATCH
  COPY of the repo (never the live tree): forcing `status: implemented` with no other change
  produced exactly 10 new blocking `test_substantiveness` noTarget violations, one per affected
  test, none of them present in `.backstop/baseline.json` — and the policy for this dimension is
  `applies-to: new-code, level: block`, which grandfathers against the BASELINE, so absence from
  the baseline means they block.
  **Why `cmd/backstop` is the correct subject, not a workaround.** These are the WIRING
  (CLM-015/016/017), PROVISIONING (CLM-030), CAPABILITY-REKEY (CLM-036, CLM-037) and E2E
  (CLM-032/033/034) claims, and every unit they pin — `buildTestSubstantivenessStep`,
  `deriveCapabilityState`, the dispatch seam, the installed-pack provisioning and the real-gate
  end-to-end path — lives in `cmd/backstop` by construction. The join basis moves from a subject
  the tests never had to structural colocation with the subject they actually target; the tests,
  their assertions and their rigor are untouched. No waiver was taken and no test was weakened.
  **Known residue, deliberately NOT fixed here.** CLM-031 is the tenth affected claim and is
  left untouched in this revision because its mandated test name
  `TestProvisioning_SubstantivenessInstalledAsLocalPack_DeclaredAndLocked` is already recorded
  (1.2.4, above) as needing its own issue → plan for the rename. That rename alone will NOT
  clear CLM-031's noTarget violation — the test's extracted symbol set is
  `{add, distribution, filepath, os, strings, t}` and contains no `gate` regardless of what the
  function is called — so whichever plan closes the rename MUST also add `subject: cmd/backstop`
  to CLM-031, or this spec still cannot promote to `implemented`.
- **1.2.4** (2026-08-10) — FOUNDER-RULED amendment reconciling the PROVISIONING model to
  shipped reality (same class of text-only correction as 1.2.3, per align-predating-artifacts:
  a live `draft` spec's text must not contradict shipped state). REQ-009 and CLM-031 were
  written when a LOCAL install was the only distribution path that existed, and hard-required
  `local` specifically (backstop.yml `local` declaration + `local` source-type lock entry +
  VerifyLock passing without a remote artifact). Reality moved: backstop-core's own
  `backstop-ai/go-substantiveness` pack now installs REMOTELY (`source_type: git`,
  `source_coordinate: backstop-ai/go-substantiveness`, `git_ref: v1.2.0`), changed by commit
  `905120f` ("feat(ISSUE-020): phase 1 — pack fleet to published coordinates") — a bulk
  migration of ALL backstop-core packs to published coordinates, not a substantiveness-specific
  decision. REQ-009's own text already anticipated remote publication as invariant-preserving,
  so the no-baked-pack concern was never violated; only the `local`-specific wording was stale.
  AMENDED: REQ-009 and CLM-031 now require the pack to be INSTALLED via the standard `pack add`
  distribution path — DECLARED in backstop.yml, LOCKED with a resolvable source, lock
  verification PASSING — with EITHER source type acceptable (`local`, which VerifyLock skips,
  or `git` with coordinate + tag); remote publication is restated as an ACCEPTED provisioning
  state rather than out-of-scope. UNCHANGED IN STRICTNESS: the prohibitions are verbatim as
  strict as before — no baked-into-the-binary, no `//go:embed`/compiled-in asset, no baked
  code path or analyzer bridge, no production reliance on `testdata`. Also reworded for
  consistency: REQ-010's `per REQ-009` cross-reference and the E2E prose (a LOCAL source
  remains the appropriate choice for a hermetic test workspace), the Implementation
  provisioning subsection and processing step 1, the Verification provisioning summary, the
  testdata-as-production and `//go:embed` Sharp Edges, one Review Question (plus one added),
  and the distribution References. Added two Sharp Edges: the source-type loosening is NOT a
  loosening of the anti-baking invariant, and a green local-install FIXTURE test does not
  prove backstop-core's LIVE provisioning (the exact reason this drift went undetected — the
  mandated tests run against `t.TempDir()` projects and never read the repo's own
  backstop.yml/backstop.lock). NO requirement or claim added/removed, NO mandated-test name
  renamed, `status` unchanged (`draft`). OPEN FOLLOW-ON: CLM-031's mandated test name
  `TestProvisioning_SubstantivenessInstalledAsLocalPack_DeclaredAndLocked` still says
  "AsLocalPack" and now contradicts the amended claim text; renaming it is real test-code work
  that needs its own issue → plan before this spec can close as `implemented`.
- **1.2.3** (2026-07-07) — Prose reconciliation to the shipped ISSUE-047 de-baking of
  `pkg/gate.TargetPackageName` (per the align-predating-artifacts rule this spec's REQ-008
  itself cites — a LIVE `draft` spec's text must not contradict shipped behavior). ISSUE-047
  removed the baked this-repo layout knowledge from `TargetPackageName`: it previously
  returned the last path component only for `pkg/...` and `""` for `cmd/...` and non-`pkg/`
  paths. It NOW reduces an OPAQUE subject token to its last `/`-segment (`filepath.Base`)
  with NO cmd//pkg/ layout branching (`cmd/backstop`→`backstop`, `pkg/gate`→`gate`,
  `internal/foo`→`foo`, bare token unchanged), and the empty-target SKIP is triggered ONLY
  by an EMPTY subject — never by a cmd//pkg/ path classification. Reworded to match: REQ-003
  (empty-target skip keyed on an empty subject, cmd//pkg/ framing removed), CLM-009, CLM-028
  (opaque last-segment reduction; only empty subject yields `""`), the `TargetPackageName`
  Sharp Edge (now flags the layout-derivation removal as an INTENDED change, not a
  regression), and the design-body Q2 description / decision table / Requirements summary /
  Review Questions. Migrated the spec-level target key `implementation.package` →
  `implementation.subject` (canonical schema key; value unchanged, `pkg/gate`). NO
  requirement, claim, or mandated-test NAME renamed or removed — text-only reconciliation of
  the behavior ISSUE-047 shipped in code (test bodies updated there; names preserved for
  lineage).
- **1.2.1** (2026-06-23) — Spec-review corrective pass on the SPEC-036 capability re-keying
  coupling (3 blocking fixes; SPEC-036 NOT revised — aligned via implementation per
  align-predating-artifacts). (1) NAMED THE LIVE LOCUS: REQ-009, CLM-035, the Implementation
  capability subsection, processing step 6, and a new `deriveCapabilityState` contract entry
  now name `deriveCapabilityState` (`cmd/backstop/gate.go:272`) as the function whose
  substantiveness arm re-keys onto the installed pack — so the implementer changes the right
  place, not an abstract behavior. (2) EXTENDED DELETE-OR-MIGRATE to `cmd/backstop`: REQ-008
  + new CLM-037 name the shipped SPEC-036 test
  `TestCapabilityState_NonGoProject_DerivesAbsentClass2`
  (`cmd/backstop/gate_capability_test.go:17`) — which asserts the OLD substantiveness keying
  and goes RED on `./cmd/backstop/` once the re-key lands — for migration of its
  substantiveness arm (coverage/contracts arms untouched), so no shipped test breaks silently
  and no claim is orphaned. (3) PINNED DIMENSION-ASYMMETRY: new REQ-009 constraint + new
  CLM-036 + a new Sharp Edge state the re-key is SUBSTANTIVENESS-DIMENSION-ONLY — the coverage
  and contracts arms of `deriveCapabilityState` stay unchanged (coverage descoped; contracts
  keeps its analyzer until SPEC-038/Seed 4), so the implementer does not over-apply the re-key
  and break contracts before its pack exists. REQ-009/REQ-010 provisioning + real-E2E core and
  the sound spec core are unchanged.
- **1.2.0** (2026-06-23) — Targeted alignment with the restated first principle (anything
  that runs in a gate is INSTALLED from a pack; the binary ships only install + execute) and
  the now-fixed substrate (ISSUE-028 multi-rule ast-grep dispatch, ISSUE-029 sandboxed
  convert). (1) New REQ-009 + CLM-030/CLM-031/CLM-035 pin the PROVISIONING model: the
  substantiveness pack is an ORDINARY INSTALLED pack — NOT built-in, NOT `//go:embed`, NOT a
  baked bridge, NOT a production reliance on testdata — dogfood-installed into backstop-core
  as a LOCAL pack (declared + locked; VerifyLock skips local packs); backstop-core gates
  itself on substantiveness through it. (2) New REQ-010 + CLM-032/CLM-033/CLM-034 add the
  REAL over-installed-pack END-TO-END proof (install local pack → real dispatch → real
  ast-grep → real sandboxed convert → SARIF → set-join → real violation on a hollow test),
  failing if the pack isn't actually installed/run — closing the recurring pack-provisioning
  integration gap that REQ-005's seam-spy alone did not (REQ-005 + its CLM-015..017 are
  RETAINED). (3) CLM-035 re-keys the substantiveness `CapabilityState` (SPEC-036) on the
  INSTALLED pack rather than the deleted baked-Go-analyzer presence — an OPEN, intended
  alignment of SPEC-036's derivation for the substantiveness dimension (flagged here and in
  Sharp Edges; no silent drift). (4) REQ-003/REQ-004 reaffirm they MUST run REAL ast-grep
  (no stub) now that ISSUE-028/029 make it unconditionally runnable. Added two Sharp Edges
  (testdata-as-production trap; `//go:embed`/baked-tier temptation), one capability-keying
  Sharp Edge, and three Review Questions. The sound core is unchanged: three-class polarity
  (SPEC-036), language-agnostic set-join (REQ-002/NoTargetViolation), strangler-before-
  deletion (REQ-006), Q1 hollow semantics, analyzer deletion (REQ-001), and the TS proof rule.

- **1.1.0** (2026-06-22) — Spec-review corrective pass reconciling with the live dispatch
  shape (3 blockers): (1) REQ-005 + CLM-015..017 + the `buildTestSubstantivenessStep`
  contract now target the REAL `resolveDispatchPackEngines()` / `dispatchPackEnginesFn`
  seam (the substantiveness step CONSUMES the pre-existing `dispatchPackEngines` dispatcher
  instead of re-implementing it; spy sits on the real seam). (2) New local REQ-007 +
  CLM-024..026 + `RouteSubstantivenessFindings` / `ReferencedSetForTest` contracts specify
  the extraction→set-join consumption seam — routing substantiveness findings out of the
  flat `pack_engines` `[]gate.Violation` stream by namespaced rule ID (justified AT THE TIME
  by the violation carrying no gate_type at all; ISSUE-118 has since added the field — see
  1.2.9) and keying extraction findings to a `MandatedTest` by
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
