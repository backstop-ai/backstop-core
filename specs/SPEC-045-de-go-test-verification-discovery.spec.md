---
title: "De-Go'd Test-Verification Discovery + Language-Neutral Package/Spec-Relevance Matchers"
number: SPEC-045
created: "2026-06-28"
status: implemented
schema_version: spec/v1
spec_version: 1.1.0

implementation:
  summary: >
    BUNDLE-012 Spec Seed 2 — de-Go's the gate CONSUMER's test-verification
    discovery and the Go-package / `./...` matchers, building on the now-fixed
    foundation contracts SPEC-043 (the pack-declared `classification: {source,
    test}` globs → `pack.Manifest.Classification`, consumed via the `pkg/gate`
    `SourceClassifier`) and SPEC-044 (the `(path, metric)` coverage index). It does
    FOUR coupled de-Go's. (1) TEST-FILE DISCOVERY (BUNDLE-012 REQ-002):
    `pkg/gate/step_testverify.go` `collectTestFuncNamesScoped` (~L247) today walks
    `_test.go` files; it is rewired to discover test files via SPEC-043's
    pack-declared TEST globs (a new `SourceClassifier.IsTestFile` predicate over the
    test-glob set the classifier already holds), so a TS `.test.ts`/`.spec.ts` file
    is discovered exactly as a Go `_test.go` file is — both from pack DATA, zero
    baked extension. (2) TEST-NAME EXTRACTION (BUNDLE-012 REQ-002): the baked
    Go-shaped `funcPattern` (`^func\s+(Test\w+)\s*\(`) is DELETED; the
    test-name/indicator pattern comes from a NEW pack-declared DATA field
    `Manifest.TestNamePatterns` (a list of regexes, capture group 1 = the test name)
    merged across declared toolchain packs and compiled into a `TestNameMatcher`. The
    go-toolchain pack declares the `func Test...` regex AS DATA; a bun pack declares
    the `test(...)`/`describe(...)`/`it(...)` regexes. (3) THE GO-PACKAGE MATCHER
    (BUNDLE-012 REQ-003): `cmd/backstop/gate.go` `goFilePackageMatchesTarget` (~L934)
    today opens the test file and reads its Go `package <name>` clause for the
    substantiveness same-package short-circuit; it is replaced with a language-neutral
    `testFileColocatedWithTarget` that compares the test file's directory leaf to the
    target unit — no `package` clause, no file read. (4) THE COVERAGE SPEC-RELEVANCE
    MATCHERS (BUNDLE-012 REQ-003): `pkg/gate/step_coverage.go` `coverageSpecRelevantToFile`
    (~L343) bakes a `.go`/`_testdata.go` suffix and a `./...` / `./<dir>` Go-package-glob
    convention into the threshold-selection relevance; those literals are removed,
    leaving language-neutral directory matching (the already-neutral `packagePathMatches`
    is retained and confirmed). When NO toolchain pack declares test globs/patterns, the
    test-verification step surfaces a DISTINCT, VISIBLE non-blocking `warning` state
    rather than reporting every mandated test as falsely "not found" or silently
    passing. Per the integration-gap lesson, the merged classifier + merged
    `TestNameMatcher` MUST be threaded into the LIVE gate test-verification AND
    substantiveness steps and proven end-to-end, not only unit-tested.
  package: pkg/gate

verification:
  level: integration
  test_command: go test ./pkg/gate/ ./pkg/pack/ ./cmd/backstop/ -race -coverprofile=cover.out
  coverage_threshold: 80

requirements:
  - id: REQ-001
    text: >
      Test-verification must discover test FILES via the pack-declared TEST globs
      (SPEC-043's `classification.test`), NOT a baked `_test.go` walk.
      `collectTestFuncNamesScoped` (and its unscoped wrapper) MUST consume SPEC-043's
      `gate.SourceClassifier` and walk only files for which the classifier reports a
      TEST-glob match (a new `SourceClassifier.IsTestFile(path) bool` predicate over
      the union of declared test globs). It is PROHIBITED for the walk to retain any
      baked `_test.go` (or any other extension) string literal: with a classifier that
      declares only bun test globs (`**/*.test.ts`, `**/*.spec.ts`), a `.test.ts` file
      MUST be discovered AND a `_test.go` file MUST NOT — proving the baked Go walk is
      gone. With the go-toolchain test globs declared, a `_test.go` file MUST still be
      discovered. Files matching no declared test glob are NOT discovered. The existing
      scope filtering (an out-of-scope test file is skipped) is retained.
    supports: language-neutral-consumer-ts-toolchain:REQ-002
  - id: REQ-002
    text: >
      Test-NAME extraction must come from pack-declared DATA, NOT a baked `func Test`
      grep. The Go-shaped `funcPattern` regex MUST be DELETED from
      `pkg/gate/step_testverify.go`. A NEW pack manifest field `TestNamePatterns
      []string` (yaml `test_name_patterns`) declares one or more regexes whose capture
      group 1 is the test name; the merged union across declared toolchain packs is
      compiled into a `TestNameMatcher` the discovery walk applies per line. The
      go-toolchain pack declares the `func Test...` regex AS DATA; a bun pack declares
      `test(...)`/`describe(...)`/`it(...)` regexes. With only the bun patterns
      declared, a Go `func TestFoo(` line MUST extract NO name (the baked Go literal is
      gone); with the go pattern declared it extracts `TestFoo`. An INVALID declared
      regex MUST be a LOUD error (a returned construction error), NEVER a silent skip.
    supports: language-neutral-consumer-ts-toolchain:REQ-002
    follows: STD-GO-001:GO-010
  - id: REQ-003
    text: >
      The substantiveness same-package short-circuit must use a language-neutral
      mechanism, NOT a Go `package` clause. `cmd/backstop/gate.go`
      `goFilePackageMatchesTarget` (which opens the file and reads `package <name>`)
      MUST be DELETED and replaced by `testFileColocatedWithTarget(filePath, targetPkg
      string) bool`, which reports same-unit by comparing the test file's directory
      leaf (`filepath.Base(filepath.Dir(filePath))`) to the target package name — no
      file read, no `package` clause, carrying no Go assumption. An empty `targetPkg`
      yields false (preserved). A TS `.test.ts` file co-located with its target MUST be
      reported same-unit WITHOUT any `package` clause existing.
    supports: language-neutral-consumer-ts-toolchain:REQ-003
  - id: REQ-004
    text: >
      The coverage spec-relevance derivation must use language-neutral directory
      matching, NOT a Go-package identity model or a `./...` glob.
      `pkg/gate/step_coverage.go` `coverageSpecRelevantToFile` MUST drop its baked
      `.go` / `_testdata.go` suffix gating AND its `./...` / `./<dir>` test-command
      substring matching; relevance is decided by directory matching against the spec's
      implementation package (the already-neutral `packagePathMatches`, retained) plus
      a root fallback for a spec with no specific implementation package. A changed
      `.ts` file in a directory matching a spec's implementation package MUST be
      relevant (proving the `.go` literal is gone); a changed file in a non-matching
      directory MUST NOT be. It is PROHIBITED for `coverageSpecRelevantToFile` /
      `packagePathMatches` to retain any baked `.go`, `_testdata.go`, or `./...` string
      literal.
    supports: language-neutral-consumer-ts-toolchain:REQ-003
  - id: REQ-005
    text: >
      When NO declared toolchain pack contributes test globs OR test-name patterns
      (test-discovery capability absent) and mandated tests exist, the
      test-verification step MUST report a DISTINCT, VISIBLE non-blocking state
      (`warning` status with a Reason naming the absent capability) rather than (a)
      reporting every mandated test as falsely "not found" or (b) silently passing.
      Consistent with loud-not-blocking, the capability-absent state MUST NOT flip the
      gate to fail. CRUCIALLY, this MUST NOT mask real misses: when test globs/patterns
      ARE declared (capability present) and a specific mandated test is genuinely
      absent from the codebase, that MUST remain a LOUD blocking failure.
    supports: language-neutral-consumer-ts-toolchain:REQ-002
    follows: STD-GO-001:GO-010
  - id: REQ-006
    text: >
      The merged `SourceClassifier` (test globs) and merged `TestNameMatcher` MUST be
      threaded into the LIVE gate test-verification step
      (`StepTestVerificationScopedFunc`) and the substantiveness step
      (`ResolveMandatedTestPaths` + `testFileColocatedWithTarget`), built from the
      UNION of the declared toolchain pack manifests in `cmd/backstop`, and proven
      END-TO-END (an executed gate over a declared toolchain pack whose test globs +
      name patterns cover a non-Go test file discovers and verifies that test), not
      only by unit tests over hand-constructed inputs — closing the integration gap
      where a correct unit is never wired into the live gate. The merge MUST be a UNION
      across all declared toolchain packs (a go pack and a bun pack together discover
      both `_test.go` and `.test.ts` mandated tests).
    supports: language-neutral-consumer-ts-toolchain:REQ-002

claims:
  # REQ-001 — test-FILE discovery via pack-declared test globs (the discovery matrix)
  - id: CLM-001
    requirement: REQ-001
    text: MANDATED — a TS test file `app/foo.test.ts` IS discovered as a test file via the declared bun test glob `**/*.test.ts`
    tests:
      - TestTestVerify_TSTestFileDiscoveredViaDeclaredGlobs
  - id: CLM-002
    requirement: REQ-001
    text: A TS `.spec.ts` test file (`app/foo.spec.ts`) IS discovered via the declared bun test glob `**/*.spec.ts`
    tests:
      - TestTestVerify_TSSpecFileDiscoveredViaDeclaredGlobs
  - id: CLM-003
    requirement: REQ-001
    text: MANDATED — a Go `pkg/x/foo_test.go` file IS STILL discovered as a test file via the declared go-toolchain test glob `**/*_test.go`
    tests:
      - TestTestVerify_GoTestFileStillDiscoveredViaGoGlobs
  - id: CLM-004
    requirement: REQ-001
    text: A non-test TS source file (`app/foo.ts`) matching no declared test glob is NOT discovered as a test file
    tests:
      - TestTestVerify_NonTestTSSourceNotDiscovered
  - id: CLM-005
    requirement: REQ-001
    text: A non-test Go source file (`pkg/x/foo.go`) matching no declared test glob is NOT discovered as a test file
    tests:
      - TestTestVerify_NonTestGoSourceNotDiscovered
  - id: CLM-006
    requirement: REQ-001
    text: A file matching no declared glob at all (`README.md`) is NOT discovered as a test file
    tests:
      - TestTestVerify_UnmatchedFileNotDiscovered
  - id: CLM-007
    requirement: REQ-001
    text: DE-GO PROOF — with ONLY bun test globs declared (`**/*.test.ts`), a `_test.go` file is NOT discovered, proving the baked `_test.go` walk is gone and discovery keys only on the declared globs
    tests:
      - TestTestVerify_NoBakedGoWalk_GoTestNotDiscoveredWithoutGoGlobs
  - id: CLM-008
    requirement: REQ-001
    text: Scope filtering is retained — an out-of-scope test file (not in the active GateScope) is skipped by the scoped discovery walk
    tests:
      - TestTestVerify_OutOfScopeTestFileSkipped
  - id: CLM-009
    requirement: REQ-001
    text: UNION across toolchain packs — with both go (`**/*_test.go`) and bun (`**/*.test.ts`) test globs declared, BOTH a `_test.go` and a `.test.ts` file are discovered from the one merged classifier (polyglot)
    tests:
      - TestTestVerify_DiscoversAcrossUnionedTestGlobs
  # REQ-002 — test-NAME extraction from pack-declared regex DATA
  - id: CLM-010
    requirement: REQ-002
    text: With the go-declared `func Test...` pattern, the matcher extracts `TestFoo` from a `func TestFoo(t *testing.T) {` line (capture group 1)
    tests:
      - TestTestNameMatcher_ExtractsGoFuncNameFromDeclaredPattern
  - id: CLM-011
    requirement: REQ-002
    text: With the bun-declared `test(...)` pattern, the matcher extracts the test name from `test('renders the widget', () => {` (capture group 1 = `renders the widget`)
    tests:
      - TestTestNameMatcher_ExtractsTSTestNameFromDeclaredPattern
  - id: CLM-012
    requirement: REQ-002
    text: With the bun-declared `describe(...)` pattern, the matcher extracts the name from `describe('widget suite', () => {`
    tests:
      - TestTestNameMatcher_ExtractsTSDescribeNameFromDeclaredPattern
  - id: CLM-013
    requirement: REQ-002
    text: With the bun-declared `it(...)` pattern, the matcher extracts the name from `it('does the thing', async () => {`
    tests:
      - TestTestNameMatcher_ExtractsTSItNameFromDeclaredPattern
  - id: CLM-014
    requirement: REQ-002
    text: A line matching no declared pattern (e.g. `const x = 1`) yields no extracted name (FindName returns ok=false)
    tests:
      - TestTestNameMatcher_NonMatchingLineYieldsNoName
  - id: CLM-015
    requirement: REQ-002
    text: DE-GO PROOF — with ONLY the bun patterns declared, a Go `func TestFoo(` line extracts NO name, proving the baked `func Test` literal is gone and extraction keys only on declared patterns
    tests:
      - TestTestNameMatcher_NoBakedFuncTest_GoLineNotExtractedWithoutGoPattern
  - id: CLM-016
    requirement: REQ-002
    text: LOUD-NOT-SILENT — an invalid declared regex (e.g. `([`) makes NewTestNameMatcher return a construction error, never a silently-dropped pattern
    tests:
      - TestTestNameMatcher_InvalidRegexIsLoudError
  - id: CLM-017
    requirement: REQ-002
    text: The baked Go `funcPattern` regex literal is DELETED from step_testverify.go — a source guard over pkg/gate asserts no `func\s+(Test\w+)` literal remains so a reintroduced baked pattern fails
    tests:
      - TestTestVerify_NoBakedFuncTestRegexLiteral
  - id: CLM-018
    requirement: REQ-002
    text: UNION across packs — with both the go and bun name patterns declared, the merged matcher extracts a Go func name from a Go line AND a TS test name from a TS line
    tests:
      - TestTestNameMatcher_MergesPatternsAcrossPacks
  # REQ-003 — language-neutral substantiveness same-unit matcher
  - id: CLM-019
    requirement: REQ-003
    text: A test file co-located with its target (`pkg/gate/foo_test.go`, targetPkg `gate`) is reported same-unit (testFileColocatedWithTarget returns true) by directory-leaf comparison
    tests:
      - TestColocated_SameDirectoryLeafIsSameUnit
  - id: CLM-020
    requirement: REQ-003
    text: A test file in a different directory than its target (`pkg/other/foo_test.go`, targetPkg `gate`) is NOT same-unit (returns false)
    tests:
      - TestColocated_DifferentDirectoryNotSameUnit
  - id: CLM-021
    requirement: REQ-003
    text: An empty targetPkg yields false (preserved guard) regardless of the file path
    tests:
      - TestColocated_EmptyTargetIsFalse
  - id: CLM-022
    requirement: REQ-003
    text: DE-GO PROOF — a TS `app/widget/foo.test.ts` with targetPkg `widget` is reported same-unit WITHOUT any Go `package` clause existing in the file, proving no clause is read
    tests:
      - TestColocated_TSFileSameUnitWithoutPackageClause
  - id: CLM-023
    requirement: REQ-003
    text: The Go `package`-clause reader `goFilePackageMatchesTarget` is DELETED from cmd/backstop/gate.go — declared absent so reintroducing a `package <name>`-reading matcher is caught
    tests:
      - TestSubstantiveness_NoBakedGoPackageClauseReader
  # REQ-004 — language-neutral coverage spec-relevance
  - id: CLM-024
    requirement: REQ-004
    text: DE-GO PROOF — a changed TS file (`app/widget/foo.ts`) in a directory matching a spec's implementation package (`app/widget`) IS spec-relevant, proving the baked `.go` suffix gate is gone
    tests:
      - TestCoverageRelevance_TSFileInMatchingPackageIsRelevant
  - id: CLM-025
    requirement: REQ-004
    text: NO REGRESSION — a changed Go file (`pkg/gate/foo.go`) in a directory matching a spec's implementation package (`pkg/gate`) is STILL spec-relevant
    tests:
      - TestCoverageRelevance_GoFileInMatchingPackageStillRelevant
  - id: CLM-026
    requirement: REQ-004
    text: A changed file in a directory NOT matching the spec's implementation package is NOT spec-relevant
    tests:
      - TestCoverageRelevance_NonMatchingDirectoryNotRelevant
  - id: CLM-027
    requirement: REQ-004
    text: DE-GO PROOF — the root fallback applies a spec with NO specific implementation package project-wide when includeRootCommand is true, WITHOUT any `./...` test-command literal
    tests:
      - TestCoverageRelevance_RootFallbackWithoutDotDotDotGlob
  - id: CLM-028
    requirement: REQ-004
    text: A spec WITH a specific implementation package is NOT made relevant to a non-matching directory by the root fallback (includeRootCommand does not over-broaden a package-scoped spec)
    tests:
      - TestCoverageRelevance_SpecificSpecNotBroadenedByRootFallback
  - id: CLM-029
    requirement: REQ-004
    text: packagePathMatches matches a `.ts` directory identically to a `.go` directory — it is pure directory matching carrying no extension/language assumption
    tests:
      - TestPackagePathMatches_LanguageNeutralDirectoryMatching
  - id: CLM-030
    requirement: REQ-004
    text: No baked `.go`/`_testdata.go`/`./...` string literal remains in coverageSpecRelevantToFile or packagePathMatches — a source guard asserts the relevance path keys only on directory matching
    tests:
      - TestCoverageRelevance_NoBakedGoLiteralsInSpecRelevance
  # REQ-005 — discovery capability-absent is visible, not misleading or silent
  - id: CLM-031
    requirement: REQ-005
    text: BOTH-ABSENT — when NO toolchain pack contributes test globs AND none contributes test-name patterns and mandated tests exist, the step returns a DISTINCT visible `warning` status with a Reason naming the absent test-discovery capability — never an unqualified pass and never a mass false "not found" fail
    tests:
      - TestTestVerify_DiscoveryCapabilityAbsentIsVisibleWarningNotSilentOrFalseFail
  - id: CLM-032
    requirement: REQ-005
    text: The discovery-capability-absent state is NON-blocking — status is not fail solely because discovery capability is absent, so the gate Pass is not flipped
    tests:
      - TestTestVerify_DiscoveryCapabilityAbsentDoesNotBlock
  - id: CLM-033
    requirement: REQ-005
    text: NO MASKING — when BOTH test globs AND test-name patterns ARE declared (capability fully present) and a specific mandated test is genuinely absent from the codebase, the step still raises a LOUD blocking failure for that test
    tests:
      - TestTestVerify_CapabilityPresentGenuineMissStillFails
  - id: CLM-037
    requirement: REQ-005
    text: PARTIAL-CAPABILITY — when a pack declares test globs BUT no `test_name_patterns` (so the walk finds the files but FindName returns false for every line), the step returns the DISTINCT visible `warning` naming the missing name patterns — it MUST NOT report every mandated test as falsely "not found" (the mass false-blocking-fail the OR guard exists to prevent)
    tests:
      - TestTestVerify_TestGlobsDeclaredButNoNamePatterns_IsVisibleWarningNotMassFail
  # REQ-006 — merged classifier + matcher wired into the live gate (integration gap)
  - id: CLM-034
    requirement: REQ-006
    text: END-TO-END — the live gate threads the merged classifier + matcher into the test-verification step; with a declared toolchain pack whose test globs + name patterns cover `.test.ts`, a `.test.ts` file whose test name matches a mandated claim is discovered and verified by the REAL gate
    tests:
      - TestGate_TestVerificationConsumesMergedDiscoveryEndToEnd
  - id: CLM-035
    requirement: REQ-006
    text: END-TO-END — the live substantiveness step consumes testFileColocatedWithTarget; a TS test file co-located with its target short-circuits the same-unit join in the real gate (no Go package clause involved)
    tests:
      - TestGate_SubstantivenessUsesColocationEndToEnd
  - id: CLM-036
    requirement: REQ-006
    text: UNION in the live gate — with two toolchain packs declared (go + bun), the live test-verification step discovers BOTH a `_test.go` and a `.test.ts` mandated test from the merged glob + pattern set
    tests:
      - TestGate_DiscoveryMergesAcrossDeclaredToolchainPacks

contracts:
  - file: pkg/pack/manifest.go
    provides:
      - name: Manifest.TestNamePatterns
        kind: variable
        signature: "TestNamePatterns []string `yaml:\"test_name_patterns\"`"
        notes: "NEW field on pack.Manifest (REQ-002/CLM-010..CLM-018): the pack-declared test-name/indicator regexes as DATA. Each pattern's capture group 1 is the test name. Optional; zero-value (nil) when the `test_name_patterns:` block is absent. The go-toolchain reference declares the `func Test...` regex; a bun pack declares the `test(...)`/`describe(...)`/`it(...)` regexes. DISJOINT from SPEC-043's Manifest.Classification field — both add fields to this file; flagged for the cross-consistency pass."
    consumes: []
  - file: pkg/gate/step_testverify.go
    provides:
      - name: TestNameMatcher
        kind: type
        signature: "type TestNameMatcher struct"
        notes: "NEW (REQ-002): holds the compiled union of declared test-name regexes. Carries no language knowledge — data + match logic only. Replaces the baked funcPattern."
      - name: NewTestNameMatcher
        kind: function
        signature: "func NewTestNameMatcher(patterns []string) (TestNameMatcher, error)"
        notes: "NEW (REQ-002/CLM-016): compiles the merged regex list; an INVALID regex returns a construction error (loud, never a silent drop). Each pattern must expose capture group 1 as the test name."
      - name: TestNameMatcher.FindName
        kind: method
        signature: "func (m TestNameMatcher) FindName(line string) (string, bool)"
        notes: "NEW (REQ-002/CLM-010..CLM-015): returns capture group 1 of the first matching declared pattern, or ok=false. With only bun patterns, a Go `func TestFoo(` line returns ok=false (no baked Go literal)."
      - name: TestNameMatcher.HasPatterns
        kind: method
        signature: "func (m TestNameMatcher) HasPatterns() bool"
        notes: "NEW (REQ-005): reports whether any name patterns are declared, so the step can surface the DISTINCT discovery-capability-absent state instead of a misleading mass not-found fail."
      - name: SourceClassifier.IsTestFile
        kind: method
        signature: "func (c SourceClassifier) IsTestFile(path string) bool"
        notes: "NEW METHOD on SPEC-043's gate.SourceClassifier (REQ-001/CLM-001..CLM-009): true iff path matches some declared TEST glob. Added in this file (same package pkg/gate) reading the test-glob matchers SPEC-043 already stores on the classifier. SEAM: depends on SPEC-043 keeping SourceClassifier in pkg/gate with its test-glob set populated — flagged for the cross-consistency pass."
      - name: SourceClassifier.HasTestGlobs
        kind: method
        signature: "func (c SourceClassifier) HasTestGlobs() bool"
        notes: "NEW METHOD on SPEC-043's gate.SourceClassifier (REQ-005): reports whether any test globs are declared, paired with TestNameMatcher.HasPatterns to detect discovery-capability-absent. Same-package addition; flagged with IsTestFile."
      - name: collectTestFuncNamesScoped
        kind: function
        signature: "func collectTestFuncNamesScoped(codeDir string, scope *GateScope, classifier SourceClassifier, matcher TestNameMatcher) map[string]string"
        notes: "MODIFIED (REQ-001/REQ-002): gains classifier + matcher. Walks only classifier.IsTestFile paths (no `_test.go` literal) and extracts names via matcher.FindName (no funcPattern). Scope filtering retained (CLM-008)."
      - name: collectTestFuncNames
        kind: function
        signature: "func collectTestFuncNames(codeDir string, classifier SourceClassifier, matcher TestNameMatcher) map[string]string"
        notes: "MODIFIED convenience wrapper: delegates to collectTestFuncNamesScoped with nil scope, now carrying the classifier + matcher."
      - name: ResolveMandatedTestPaths
        kind: function
        signature: "func ResolveMandatedTestPaths(mandated []MandatedTest, codeDir string, classifier SourceClassifier, matcher TestNameMatcher) []MandatedTest"
        notes: "MODIFIED (REQ-006): gains classifier + matcher so the substantiveness path resolves test file paths via the same pack-declared discovery the verification step uses."
      - name: StepTestVerificationScopedFunc
        kind: function
        signature: "func StepTestVerificationScopedFunc(specDir, codeDir string, scope *GateScope, classifier SourceClassifier, matcher TestNameMatcher) StepFunc"
        notes: "MODIFIED (REQ-001/REQ-002/REQ-005/REQ-006): gains classifier + matcher. Discovery needs BOTH globs (to find files) AND patterns (to extract names), so capability is PRESENT only when BOTH are declared: when `!classifier.HasTestGlobs() || !matcher.HasPatterns()` (EITHER missing) and mandated tests exist, returns a DISTINCT non-blocking `warning` status with a Reason naming the missing input (REQ-005/CLM-031/CLM-032/CLM-037), never an unqualified pass nor a mass not-found fail. The partial case (globs declared, patterns absent) is specifically intercepted here so FindName returning false for every line cannot become a mass false not-found fail. When capability is FULLY present (both declared), a genuinely-missing mandated test stays a loud fail (CLM-033)."
      - name: StepTestVerificationFunc
        kind: function
        signature: "func StepTestVerificationFunc(specDir, codeDir string, classifier SourceClassifier, matcher TestNameMatcher) StepFunc"
        notes: "MODIFIED convenience wrapper: delegates to StepTestVerificationScopedFunc with nil scope, now carrying the classifier + matcher."
      - name: funcPattern
        kind: variable
        signature: "var funcPattern = regexp.MustCompile(`^func\\s+(Test\\w+)\\s*\\(`)"
        absent: true
        scope: pkg/gate
        notes: "DELETED (REQ-002/CLM-015/CLM-017): the baked Go-shaped test-name regex. Replaced by the pack-declared TestNameMatcher. Declared absent so reintroducing a `func\\s+(Test\\w+)` literal is caught as a regression."
    consumes:
      - source: pkg/gate
        name: SourceClassifier
        kind: type
      - source: pkg/gate
        name: GateScope
        kind: type
      - source: pkg/gate
        name: MandatedTest
        kind: type
  - file: pkg/gate/step_coverage.go
    provides:
      - name: coverageSpecRelevantToFile
        kind: function
        signature: "func coverageSpecRelevantToFile(spec SpecVerification, file string, includeRootCommand bool) bool"
        notes: "MODIFIED (REQ-004/CLM-024..CLM-028): the baked `.go`/`_testdata.go` suffix gate and the `./...` / `./<dir>` test-command substring matching are REMOVED. Relevance is directory matching against spec.ImplementationPackage via packagePathMatches, plus a root fallback (includeRootCommand && spec has no specific implementation package). DISJOINT function from SPEC-043's measurable-path edit and SPEC-044's index/threshold-loop edit in this same file — triple-overlap flagged for the cross-consistency pass."
      - name: packagePathMatches
        kind: function
        signature: "func packagePathMatches(changedDir string, specPackage string) bool"
        notes: "RETAINED + CONFIRMED language-neutral (REQ-004/CLM-029): pure directory string matching with no extension/language assumption. Its only Go-coupling (being fed `.go`-filtered input) is removed in coverageSpecRelevantToFile. Listed to lock its language-neutral behavior."
    consumes:
      - source: pkg/gate
        name: SpecVerification
        kind: type
  - file: cmd/backstop/gate.go
    provides:
      - name: testFileColocatedWithTarget
        kind: function
        signature: "func testFileColocatedWithTarget(filePath, targetPkg string) bool"
        notes: "NEW (REQ-003/CLM-019..CLM-022): language-neutral same-unit predicate replacing goFilePackageMatchesTarget. Compares filepath.Base(filepath.Dir(filePath)) to targetPkg; no file read, no `package` clause. Empty targetPkg => false."
      - name: goFilePackageMatchesTarget
        kind: function
        signature: "func goFilePackageMatchesTarget(filePath, targetPkg string) bool"
        absent: true
        scope: cmd/backstop/gate.go
        notes: "DELETED (REQ-003/CLM-023): the Go `package <name>`-clause reader. Replaced by testFileColocatedWithTarget. Declared absent so reintroducing a clause-reading same-package matcher is caught. DISJOINT from SPEC-046's bridge deletion in this same file — flagged for the cross-consistency pass."
      - name: mergeTestNameMatcher
        kind: function
        signature: "func mergeTestNameMatcher(packSets ...[]*pack.Manifest) (gate.TestNameMatcher, error)"
        notes: "NEW (REQ-006/CLM-018/CLM-036): unions Manifest.TestNamePatterns across the declared toolchain packs and compiles one gate.TestNameMatcher. Built where the manifests are visible (cmd/backstop) so pkg/gate takes no pkg/pack dependency. Invalid regex surfaces as a loud config-error step."
      - name: buildTestSubstantivenessStep
        kind: function
        signature: "func buildTestSubstantivenessStep(specDir, codeDir, projectRoot string, scope *gate.GateScope, classifier gate.SourceClassifier, matcher gate.TestNameMatcher) gate.StepFunc"
        notes: "MODIFIED (REQ-003/REQ-006): gains classifier + matcher (threaded into ResolveMandatedTestPaths) and uses testFileColocatedWithTarget for the same-unit short-circuit instead of goFilePackageMatchesTarget."
    consumes:
      - source: pkg/gate
        name: SourceClassifier
        kind: type
      - source: pkg/gate
        name: TestNameMatcher
        kind: type
      - source: pkg/pack
        name: Manifest
        kind: type
      - source: cmd/backstop
        name: mergeSourceClassifier
        kind: function
---

# SPEC-045: De-Go'd Test-Verification Discovery + Language-Neutral Package/Spec-Relevance Matchers

## Overview

This spec is **Seed 2 of BUNDLE-012** (language-neutral gate consumer + TypeScript
toolchain pack). It **builds on the fixed foundation contracts**:

- **SPEC-043** — the pack-declared `classification: {source, test}` glob block parsed
  onto `pack.Manifest.Classification`, consumed via the `pkg/gate` `SourceClassifier`.
  This spec **consumes SPEC-043's TEST globs** for test-file discovery; it does **not**
  redefine the classifier or its source-side behavior.
- **SPEC-044** — the `(path, metric)` coverage index + per-metric thresholds. This
  spec does **not** touch `indexCoverageByPath` or the threshold loop.

It de-Go's **four** consumer sites the `backstop/self`
`no-language-literal-on-neutral-spine` rule flags:

1. **Test-FILE discovery (BUNDLE-012 REQ-002).** `pkg/gate/step_testverify.go`
   `collectTestFuncNamesScoped` walks `_test.go`; rewired to discover via SPEC-043's
   pack-declared TEST globs (a new `SourceClassifier.IsTestFile` predicate).
2. **Test-NAME extraction (BUNDLE-012 REQ-002).** The baked Go-shaped `funcPattern`
   regex is deleted; the test-name pattern comes from a **new pack-declared DATA
   field** `Manifest.TestNamePatterns`, compiled into a `TestNameMatcher`.
3. **The substantiveness same-package matcher (BUNDLE-012 REQ-003).**
   `cmd/backstop/gate.go` `goFilePackageMatchesTarget` reads the Go `package` clause;
   replaced by a language-neutral `testFileColocatedWithTarget` (directory-leaf
   comparison, no file read).
4. **The coverage spec-relevance matchers (BUNDLE-012 REQ-003).**
   `pkg/gate/step_coverage.go` `coverageSpecRelevantToFile` bakes a `.go`/`_testdata.go`
   suffix and a `./...`/`./<dir>` Go-package-glob convention; those literals are
   removed, leaving language-neutral directory matching.

The **mandated proof** is symmetric: a TS `.test.ts` file IS discovered as a test via
the declared test globs, **and** a Go `_test.go` file still is via the go pack's globs
(CLM-001 / CLM-003).

**In scope:** test-file discovery from declared globs; test-name extraction from
declared regex DATA; the language-neutral co-location + spec-relevance matchers; the
discovery-capability-absent visible state; and wiring the merged classifier + matcher
into the live gate end-to-end.

**Out of scope (fenced to sibling seeds):** the classification glob CONTRACT + coverage
measurable-path → **SPEC-043** (Seed 1, consumed here). The coverage RECORD model /
`(path, metric)` index / per-metric thresholds → **SPEC-044**. Deleting the `language:`
bridge + retiring the `language:` field + the traceability-classifier rehome →
**SPEC-046** (Seed 3). The `backstop/bun-toolchain` pack + ratchet→block flip + external
proof → **SPEC-047** (Seed 5). This spec **adds** the `test_name_patterns` DATA to the
go-toolchain pack so the dogfood keeps discovering Go tests after `funcPattern` is gone,
but does not author the bun pack.

## Requirements

Requirements are enumerated in the `requirements:` frontmatter (REQ-001 … REQ-006),
each tracing to BUNDLE-012 REQ-002 or REQ-003 via `supports`. Summary:

| Spec REQ | Bundle REQ | Commits to |
| --- | --- | --- |
| REQ-001 | REQ-002 | Test-FILE discovery via SPEC-043's pack-declared TEST globs (`IsTestFile`); no baked `_test.go` walk. |
| REQ-002 | REQ-002 | Test-NAME extraction from pack-declared `test_name_patterns` DATA; `funcPattern` deleted; invalid regex is loud. |
| REQ-003 | REQ-003 | `goFilePackageMatchesTarget` (Go `package`-clause reader) replaced by language-neutral `testFileColocatedWithTarget`. |
| REQ-004 | REQ-003 | `coverageSpecRelevantToFile` drops `.go`/`_testdata.go`/`./...` literals; directory matching only (`packagePathMatches` retained). |
| REQ-005 | REQ-002 | EITHER test globs OR test-name patterns missing (capability present only when BOTH declared) => DISTINCT visible non-blocking `warning`, not a silent pass nor a false mass not-found fail (incl. the partial globs-but-no-patterns case); full capability never masks real misses. |
| REQ-006 | REQ-002, REQ-003 | The merged classifier + matcher are threaded into the LIVE test-verification + substantiveness steps and proven end-to-end. |

### Test-FILE discovery matrix (REQ-001)

`SourceClassifier.IsTestFile(path)` over the union of declared TEST globs decides
discovery:

| file | declared test globs | discovered as test? | Claim |
| --- | --- | --- | --- |
| `app/foo.test.ts` | bun `**/*.test.ts` | **YES** | CLM-001 |
| `app/foo.spec.ts` | bun `**/*.spec.ts` | **YES** | CLM-002 |
| `pkg/x/foo_test.go` | go `**/*_test.go` | **YES** | CLM-003 |
| `app/foo.ts` | (no matching glob) | **NO** | CLM-004 |
| `pkg/x/foo.go` | (no matching glob) | **NO** | CLM-005 |
| `README.md` | (no matching glob) | **NO** | CLM-006 |
| `pkg/x/foo_test.go` | **only** bun `**/*.test.ts` | **NO** (de-Go proof) | CLM-007 |

Plus: scope filtering is retained (CLM-008); the classifier is the **union** across
packs so a polyglot repo discovers both `_test.go` and `.test.ts` (CLM-009).

### Test-NAME extraction matrix (REQ-002)

`TestNameMatcher.FindName(line)` returns capture group 1 of the first matching declared
pattern:

| line | declared patterns | extracted name | Claim |
| --- | --- | --- | --- |
| `func TestFoo(t *testing.T) {` | go `func Test...` | `TestFoo` | CLM-010 |
| `test('renders the widget', () => {` | bun `test(...)` | `renders the widget` | CLM-011 |
| `describe('widget suite', () => {` | bun `describe(...)` | `widget suite` | CLM-012 |
| `it('does the thing', async () => {` | bun `it(...)` | `does the thing` | CLM-013 |
| `const x = 1` | any | (none, ok=false) | CLM-014 |
| `func TestFoo(` | **only** bun patterns | (none — de-Go proof) | CLM-015 |

Plus: an invalid declared regex is a loud construction error (CLM-016); the baked
`funcPattern` literal is deleted (CLM-017); the merged matcher unions patterns across
packs (CLM-018).

### Language-neutral matchers (REQ-003, REQ-004)

- **`testFileColocatedWithTarget`** replaces the Go `package`-clause reader: same-unit
  ⟺ `filepath.Base(filepath.Dir(filePath)) == targetPkg`. No file read, no clause. A TS
  `.test.ts` co-located with its target is same-unit *without any `package` clause
  existing* (CLM-022).
- **`coverageSpecRelevantToFile`** drops the baked `.go`/`_testdata.go` suffix gate and
  the `./...`/`./<dir>` test-command convention. Relevance = `packagePathMatches`
  (directory matching against `spec.ImplementationPackage`) plus a root fallback for a
  spec with no specific implementation package. `packagePathMatches` is already pure
  directory matching and is retained + confirmed language-neutral (CLM-029).

### Seams with the parallel siblings

This spec shares files with three siblings. Contract entries are declared on **disjoint
functions**; the shared-file overlaps are flagged here for the cross-consistency pass:

| Shared file | This spec edits | Sibling edits | Flag |
| --- | --- | --- | --- |
| `pkg/gate/step_coverage.go` | `coverageSpecRelevantToFile`, `packagePathMatches` | SPEC-043 measurable-path; SPEC-044 `(path, metric)` index + threshold loop | **Triple overlap** — disjoint functions; this spec does not touch the classifier param or the record index. |
| `cmd/backstop/gate.go` | `goFilePackageMatchesTarget`→`testFileColocatedWithTarget`, `mergeTestNameMatcher`, `buildTestSubstantivenessStep` | SPEC-046 deletes the `language:` toolchain bridge (`loadBridgedToolchainPacks` et al.) | Disjoint functions; this spec **reuses** SPEC-043's `mergeSourceClassifier` and builds the matcher from the same declared-toolchain-pack manifests SPEC-046 reworks the source of. |
| `pkg/pack/manifest.go` | adds `Manifest.TestNamePatterns` | SPEC-043 adds `Manifest.Classification` | Disjoint fields. |
| `pkg/gate` `SourceClassifier` (SPEC-043's type) | adds `IsTestFile`, `HasTestGlobs` methods | SPEC-043 defines the type + source-side methods | Same-package method additions reading SPEC-043's test-glob set — **confirm SPEC-043 keeps the test globs populated on the classifier** (it does: `NewSourceClassifier(source, test []string)`). |

## Implementation

Target package: **`pkg/gate`** (the discovery rewrite, the matcher, the classifier
method), with the new DATA field in **`pkg/pack`** and the merge/wiring + co-location
matcher in **`cmd/backstop`**. Processing steps the planner maps tasks to:

1. **Add the `test_name_patterns` DATA field to the manifest (REQ-002).** Add
   `TestNamePatterns []string \`yaml:"test_name_patterns"\`` to `pack.Manifest` in
   `pkg/pack/manifest.go` (optional; nil when absent). Add the Go regex AS DATA to the
   `backstop/go-toolchain` pack's `pack.yml` (its own repo **and** the gitignored
   dogfood copy) so backstop-core's own gate keeps discovering `func Test...` after the
   baked `funcPattern` is deleted:
   ```yaml
   test_name_patterns:
     - "^\\s*func\\s+(Test\\w+)\\s*\\("
   ```
   The bun pack (SPEC-047) will declare its own patterns, e.g.
   `"(?:\\bit|\\btest|\\bdescribe)\\s*\\(\\s*['\\\"`]([^'\\\"`]+)"`.

2. **Add the `TestNameMatcher` (REQ-002).** In `pkg/gate/step_testverify.go` add
   `TestNameMatcher`, `NewTestNameMatcher(patterns []string) (TestNameMatcher, error)`
   (compiles each regex; an invalid regex returns a loud error), `FindName(line)
   (string, bool)` (capture group 1 of the first match), and `HasPatterns()`. **Delete
   `funcPattern`.**

3. **Add `IsTestFile` + `HasTestGlobs` to SPEC-043's `SourceClassifier` (REQ-001).** In
   the same package (`pkg/gate`), add methods reading the test-glob matchers SPEC-043
   stores: `IsTestFile(path) bool` (matches some declared test glob) and
   `HasTestGlobs() bool`.

4. **Rewire discovery (REQ-001/REQ-002).** Change `collectTestFuncNamesScoped` to take
   `(classifier SourceClassifier, matcher TestNameMatcher)`: walk only
   `classifier.IsTestFile(path)` files (no `_test.go` literal), extract names via
   `matcher.FindName` (no `funcPattern`). Thread the same params through
   `collectTestFuncNames` and `ResolveMandatedTestPaths`.

5. **Capability-absent state (REQ-005).** Discovery needs BOTH inputs — test globs to
   FIND the test file AND name patterns to EXTRACT the test name — so capability is
   PRESENT only when BOTH are declared. In `StepTestVerificationScopedFunc`: when
   `!classifier.HasTestGlobs() || !matcher.HasPatterns()` (EITHER missing) and mandated
   tests exist, return a DISTINCT non-blocking `warning` status with a Reason naming the
   absent discovery capability (and, in the partial case, naming WHICH input is missing —
   e.g. test globs declared but no `test_name_patterns`), never an unqualified `pass` and
   never a mass "not found" fail. The partial case is the dangerous one: globs find the
   files but with no patterns `FindName` returns false for every line → empty `found` map
   → every mandated test would falsely report "not found"; the OR guard intercepts that
   before it becomes a mass false-blocking fail. When capability is FULLY present (both
   globs AND patterns declared), a genuinely-missing mandated test stays a loud blocking
   failure (unchanged behavior).

6. **De-Go the substantiveness same-unit matcher (REQ-003).** In `cmd/backstop/gate.go`
   **delete `goFilePackageMatchesTarget`** and add `testFileColocatedWithTarget(filePath,
   targetPkg) bool` = `targetPkg != "" && filepath.Base(filepath.Dir(filePath)) ==
   targetPkg`. Update `buildTestSubstantivenessStep` to call it.

7. **De-Go the coverage spec-relevance matchers (REQ-004).** In
   `pkg/gate/step_coverage.go` rewrite `coverageSpecRelevantToFile`: remove the
   `.go`/`_testdata.go` suffix gate and the `./...`/`./<dir>` test-command matching;
   relevance = `packagePathMatches(dir, spec.ImplementationPackage)` plus a root
   fallback (`includeRootCommand && spec.ImplementationPackage` is empty/root). Leave
   `packagePathMatches` unchanged (already directory-only).

8. **Wire the merged classifier + matcher into the live gate (REQ-006).** In
   `cmd/backstop/gate.go` add `mergeTestNameMatcher(packSets ...[]*pack.Manifest)
   (gate.TestNameMatcher, error)` (union of `TestNamePatterns`), **reuse** SPEC-043's
   `mergeSourceClassifier` for the classifier, and thread both into
   `StepTestVerificationScopedFunc` (the L629 call site) and
   `buildTestSubstantivenessStep`. A merge regex-compile error surfaces as a loud
   config-error step. Prove end-to-end that the live gate discovers + verifies a `.test.ts`
   test (CLM-034) and unions across declared toolchain packs (CLM-036).

## Verification

- **Level:** `integration` (threshold 80). The matcher, classifier method, and matchers
  are unit-testable in `pkg/gate`, but REQ-006 is a cross-package wiring guarantee
  (`cmd/backstop` → `pkg/gate`, reading `pkg/pack` manifests), so the spec is verified at
  integration level with mandated end-to-end gate tests that the merged discovery is
  actually consumed ([[feedback_integration_gap]]).
- **Command:** `go test ./pkg/gate/ ./pkg/pack/ ./cmd/backstop/ -race
  -coverprofile=cover.out`.
- **Mandated tests:** every test named in the `claims[]` `tests:` fields. The
  load-bearing symmetric pair is `TestTestVerify_TSTestFileDiscoveredViaDeclaredGlobs`
  (CLM-001) and `TestTestVerify_GoTestFileStillDiscoveredViaGoGlobs` (CLM-003) — a TS
  `.test.ts` file IS discovered via the declared globs and a Go `_test.go` file still is.

## Sharp Edges

- **The classifier must expose the TEST globs, not only `IsMeasurableSource`.** SPEC-043
  exposes `IsMeasurableSource` (source ∧ ¬test) and `HasSourceGlobs`. Test discovery
  needs a **pure test-glob match** (`IsTestFile`), which `IsMeasurableSource` does NOT
  provide (a test file is by definition *not* measurable source). This spec adds
  `IsTestFile`/`HasTestGlobs` as same-package methods. If SPEC-043's final classifier
  drops or renames its stored test-glob set, this breaks — reconcile in the
  cross-consistency pass.
- **Mandated test names for TS are the `test(...)` STRING, not a function name.** For Go
  the mandated name is the `func TestX` identifier; for TS it is the string passed to
  `test('...')`. The `TestNameMatcher` returns capture group 1 verbatim, so a TS spec's
  claim `tests:` must list the exact string used in `test(...)`. Authors of TS specs must
  not assume a Go-style identifier.
- **`testFileColocatedWithTarget` is a behavior change, not a transliteration.** The old
  matcher read the actual Go `package` clause (which can differ from the directory name —
  e.g. `package foo` in directory `bar/`). The new matcher uses the directory leaf. For
  Go's normal case (package name == directory leaf, plus the `_test` external variant in
  the same directory) this is equivalent, but a deliberately mis-named package would now
  be judged by its directory. This is the intended language-neutral semantics — flagged so
  it is not mistaken for a regression.
- **Dropping the `.go` gate in `coverageSpecRelevantToFile` widens relevance.** A changed
  non-source file (e.g. `README.md`) in a matching directory now selects that spec's
  threshold. This is safe: SPEC-043's measurable-source guard decides which files actually
  need records, and a non-source change contributes no measurable file — so no coverage
  requirement fires. But a reviewer expecting the old `.go` filter must understand the
  guard moved upstream.
- **Dropping `./...` NARROWS coverage relevance for impl-package specs.** Symmetric to the
  `.go`-gate widening above, removing the `./...`-in-test_command project-wide relevance
  fallback intentionally narrows relevance: the rewritten `coverageSpecRelevantToFile`
  applies the root/project-wide fallback ONLY when `spec.ImplementationPackage == ""`; a
  spec WITH a specific implementation package is matched by `packagePathMatches` against
  that package and is NO LONGER pulled in project-wide just because its test command
  contained `./...` (CLM-028 locks this). This is within BUNDLE-012 REQ-003 (the `./...`
  literal is one of the baked Go-package-glob conventions being removed) and is the
  intended language-neutral semantics — flagged here so a reviewer does not read the
  dropped `./...` fallback as an accidental coverage-relevance regression.
- **Capability-absent must not become a blanket excuse.** REQ-005 makes "EITHER test
  globs OR test-name patterns not declared" a non-blocking warning — but ONLY that exact
  condition (capability is present only when BOTH are declared). When capability is FULLY
  present (both declared), a genuinely-missing mandated test must still RED (CLM-033).
  Conflating the two (e.g. warning whenever *any* test is not found) would silently gut
  test-verification — the inverse of the bundle's anti-vacuous-green mission.
- **The PARTIAL case (globs but no patterns) is the trap REQ-005 exists to catch.** If the
  guard were both-absent (AND) instead of either-absent (OR), a pack that declares test
  globs but omits `test_name_patterns` would slip through as "capability present": the walk
  finds every test file, but `FindName` returns false for every line → empty `found` map →
  EVERY mandated test reports "not found" → mass false blocking fail misattributing the
  config gap to the codebase. The OR guard (`!HasTestGlobs() || !HasPatterns()`) intercepts
  this and surfaces it as the DISTINCT warning naming the missing patterns (CLM-037). A
  reviewer must not "simplify" the guard back to AND.
- **Merge-time regex compile errors must be loud.** `mergeTestNameMatcher` compiles
  pack-declared regexes; an invalid pattern in a pack must surface as a loud config-error
  step (GO-010 — no silent failure), not a silently-empty matcher that makes discovery
  find nothing and then mass-fail every mandated test.
- **Ordering with SPEC-046's bridge deletion.** This spec reads the declared-toolchain-pack
  manifests in `cmd/backstop` to build the matcher; SPEC-046 reworks where those manifests
  come from (deleting the `language:` bridge). The matcher merge must read from whatever
  manifest set survives SPEC-046 — co-owned plumbing flagged for the cross-consistency pass.

## Review Questions

- Is the `_test.go` walk actually replaced by `classifier.IsTestFile` (pack-declared test
  globs), so a `.test.ts` file is discovered and — with only bun globs declared — a
  `_test.go` file is NOT? (REQ-001/CLM-007.)
- Is `funcPattern` actually DELETED from `step_testverify.go`, and does extraction key only
  on the pack-declared `TestNamePatterns`, so a Go `func TestFoo(` line extracts nothing
  when only bun patterns are declared? (REQ-002/CLM-015/CLM-017.)
- Does `testFileColocatedWithTarget` reach its verdict WITHOUT opening the file or reading
  a `package` clause, and is `goFilePackageMatchesTarget` deleted? (REQ-003/CLM-022/CLM-023.)
- Do `coverageSpecRelevantToFile` and `packagePathMatches` retain NO `.go`/`_testdata.go`/`./...`
  literal, with relevance decided by directory matching plus a root fallback? (REQ-004/CLM-030.)
- Is the capability guard either-absent (`!HasTestGlobs() || !HasPatterns()`) — NOT
  both-absent — so a pack declaring test globs but no `test_name_patterns` surfaces the
  DISTINCT visible `warning` (naming the missing patterns) rather than a mass false "not
  found" fail? And when BOTH are declared, does a genuine miss still RED?
  (REQ-005/CLM-031/CLM-033/CLM-037.)
- Does the LIVE gate (not just a unit) build the merged classifier + matcher from the
  declared toolchain packs and consume them in BOTH the test-verification and
  substantiveness steps, proven by an end-to-end `.test.ts` discovery test? (REQ-006/CLM-034.)
- Does an invalid pack-declared regex surface as a loud config-error step rather than a
  silently-empty matcher? (REQ-002/CLM-016, sharp edges.)

## References

- BUNDLE-012 (`language-neutral-consumer-ts-toolchain`) — this is Seed 2; REQ-002 + REQ-003
  are the bundle requirements this spec implements.
- SPEC-043 (`pack-declared-globs-coverage-consumer`) — Seed 1, the foundation contract:
  the `classification: {source, test}` globs, `pack.Manifest.Classification`, the
  `SourceClassifier`, and `mergeSourceClassifier`. This spec consumes its TEST globs and
  reuses `mergeSourceClassifier`; it adds `IsTestFile`/`HasTestGlobs`.
- SPEC-044 (`multi-metric-coverage-records`) — the `(path, metric)` coverage index +
  per-metric thresholds; shares `pkg/gate/step_coverage.go` (disjoint functions).
- SPEC-046 (`retire-language-toolchain-bridge`) — deletes the `language:` bridge in
  `cmd/backstop/gate.go` (disjoint functions; co-owned manifest plumbing).
- SPEC-047 — the `backstop/bun-toolchain` pack that declares the bun `classification.test`
  globs and `test_name_patterns` this spec's consumer reads.
- [[feedback_loud_not_blocking]] — governs REQ-002/REQ-005: an invalid regex and a missing
  discovery capability must be LOUD; capability-absent is loud but non-blocking.
- [[feedback_zero_baked_checks]] / DD-1 — the thin-executor first principle: discovery reads
  pack-declared globs + patterns as DATA; zero baked language knowledge.
- [[feedback_integration_gap]] — REQ-006's end-to-end wiring guard against a correct unit
  never reaching the live gate (the recurring stubbed-dispatcher gap).
- [[packs_always_external]] — the go-toolchain/bun-toolchain packs live in their own repos;
  the `test_name_patterns` block must land there (and in the gitignored dogfood copy).
- Code (this branch): `pkg/gate/step_testverify.go` `collectTestFuncNamesScoped` (~L247),
  `funcPattern` (~L173), `ResolveMandatedTestPaths` (~L385); `cmd/backstop/gate.go`
  `goFilePackageMatchesTarget` (~L934), `buildTestSubstantivenessStep`, the
  `StepTestVerificationScopedFunc` call site (~L629); `pkg/gate/step_coverage.go`
  `coverageSpecRelevantToFile` (~L343), `packagePathMatches` (~L360); `pkg/pack/manifest.go`
  `Manifest`; reference `.backstop/packs/backstop/go-toolchain/pack.yml`.

## Version History

- **1.1.0** (2026-06-30) — Status → `implemented`. The BUNDLE-012 Seed 1 (Pillar A) code
  shipped and passed impl-review PASS; the de-Go'd test-verification discovery and the
  language-neutral package/spec-relevance matchers are live. No requirement, claim, or
  contract text changed — lifecycle transition only.
- **1.0.0** (2026-06-28) — Initial spec authored from BUNDLE-012 Seed 1 (Pillar A).
