---
title: "Test Name Discovery Drops Later Regex Matches on the Same Line"
schema_version: issue/v1

issue:
  id: ISSUE-188
  title: "Test Name Discovery Drops Later Regex Matches on the Same Line"
  type: bug
  status: ready
  created: "2026-08-25"

complexity:
  scope: contained
  uncertainty: known
  risk: moderate

verification:
  level: build
  test_command: "./bin/backstop gate --all"

implementation:
  summary: >
    Correct the language-neutral TestNameMatcher and test-file collection path to
    enumerate every capture-group-1 match on each physical line across all declared
    patterns, rather than returning only the first match. Preserve loud regex validation,
    existing single-name behavior, language neutrality, and deterministic deduplication;
    prove the fix on a generic same-line fixture and ISSUE-187's pinned SPEC-072 corpus.
  package: pkg/gate, cmd/backstop

requirements:
  - id: REQ-001
    text: >
      `TestNameMatcher` must expose an API that returns every nonempty capture-group-1
      test name matched on one physical line by every pack-declared
      `test_name_patterns` regex. It must use Go's existing regexp engine and declared
      pattern data only. A line containing two non-overlapping declarations matched by
      one pattern must return both names; returning only the first
      `FindStringSubmatch` result is prohibited.
  - id: REQ-002
    text: >
      Enumeration must be deterministic and duplicate-free. Results are ordered by
      match start byte in the source line and then by declared-pattern order for matches
      at the same position; repeated discovery of the same name at the same position,
      including overlap between two declared patterns, contributes one name. Empty or
      missing capture group 1 is not a discovered test name.
  - id: REQ-003
    text: >
      `collectTestFuncNamesScoped` and its unscoped/resolution consumers must add every
      name enumerated from every scanned line to the existing name-to-file discovery
      index. Existing file classification, scope filtering, name/path deduplication, and
      `ResolveMandatedTestPaths` attribution semantics remain unchanged. A later valid
      declaration on a line must resolve a mandated test to that physical test file.
  - id: REQ-004
    text: >
      Existing one-name behavior must remain source-compatible or be migrated cleanly.
      If `FindName(line) (string, bool)` remains, it returns the first name from the new
      deterministic enumeration and preserves its current no-match result. Existing Go
      `func Test...` and TypeScript/Bun `test`/`describe`/`it` single-declaration lines
      must produce exactly the same names and presence results as before.
  - id: REQ-005
    text: >
      Invalid declared regexes must remain loud construction errors, an empty matcher
      must still report no discovery capability, and the correction must introduce no
      baked Bash, Go, TypeScript, function-declaration, separator, extension, or
      test-framework syntax into Core. All language-specific recognition remains in
      installed packs' declared patterns and classification globs.
  - id: REQ-006
    text: >
      Acceptance must reproduce the real failure against ISSUE-187's immutable pinned
      Backstop Core/SPEC-072 corpus: the pre-fix consumer discovers 36 of 44 distinct
      Bash function names because eight later semicolon-separated declarations are on
      lines whose first declaration wins; the corrected consumer discovers all 44 from
      the same unchanged files and unchanged Bash pack declarations. The 47 mandated
      references must resolve to those 44 names without suppression, allowing the
      existing terminal Bash-pack acceptance to proceed with zero absent names and zero
      resulting SPEC-072 broken-promise drift.
  - id: REQ-007
    text: >
      Scope is limited to generic TestNameMatcher enumeration, its Core collection/API
      migration, and regression acceptance. No Bash regex or pack change, no Core
      classification/capability/verdict policy change, no substantiveness work, and no
      edit to ISSUE-187, PLAN-ISSUE-187, SPEC-045, SPEC-072, or PLAN-SPEC-072 is allowed.
      A self-contained checked-in `testdata` reference-pack fixture containing only the
      classification/name-pattern declarations and minimal deterministic test-engine
      declaration plus producer/converter needed to run assembled acceptance is
      permitted. That plumbing must be byte-pinned, inert outside the fixture, never
      production- or default-registered, and must neither duplicate nor consume the
      external Bash pack release implementation or unfinished PLAN-ISSUE-187 state.

claims:
  - id: CLM-001
    requirement: REQ-001
    text: One declared pattern returns both non-overlapping names from a physical line containing two declarations.
    tests:
      - TestTestNameMatcher_FindNamesReturnsAllMatchesOnLine
  - id: CLM-002
    requirement: REQ-002
    text: Multiple declared patterns produce stable source-order results and overlapping patterns do not duplicate a name.
    tests:
      - TestTestNameMatcher_FindNamesDeterministicAcrossPatternOverlap
  - id: CLM-003
    requirement: REQ-002
    text: Empty or absent capture group 1 is not returned as a discovered name.
    tests:
      - TestTestNameMatcher_FindNamesRejectsEmptyOrMissingCapture
  - id: CLM-004
    requirement: REQ-003
    text: Test-file collection indexes every same-line name to the physical file so a later name resolves its mandated-test path.
    tests:
      - TestTestVerify_CollectsAndResolvesLaterSameLineName
  - id: CLM-005
    requirement: REQ-004
    text: The compatibility API returns the first deterministic name and preserves the existing no-match result.
    tests:
      - TestTestNameMatcher_FindNameCompatibilityUsesFirstEnumeratedName
  - id: CLM-006
    requirement: REQ-004
    text: Existing Go and TypeScript single-declaration extraction results are unchanged.
    tests:
      - TestTestNameMatcher_GoAndTSSingleMatchBehaviorUnchanged
  - id: CLM-007
    requirement: REQ-005
    text: Invalid regex construction remains loud and an empty matcher remains capability-absent.
    tests:
      - TestTestNameMatcher_AllMatchEnumerationPreservesLoudValidation
  - id: CLM-008
    requirement: REQ-005
    text: Core's matcher and collector contain no baked Bash declaration or separator syntax.
    tests:
      - TestTestNameMatcher_AllMatchEnumerationRemainsLanguageNeutral
  - id: CLM-009
    requirement: REQ-006
    text: The unchanged pinned SPEC-072 corpus and Bash declarations resolve all 47 references to 44 distinct functions, including the eight formerly hidden later same-line names.
    tests:
      - TestGate_SPEC072DiscoversAll44BashFunctionsAcrossSameLineDeclarations
  - id: CLM-010
    requirement: REQ-006
    text: With complete name discovery, the existing terminal Bash-pack acceptance has no absent SPEC-072 mandated test and no consequent artifact-status broken promise.
    tests:
      - TestGate_SPEC072TerminalDiscoveryAndDriftClearAfterAllMatchEnumeration
  - id: CLM-011
    requirement: REQ-007
    text: The new all-match API exists, while Core production source vendors or registers no Bash mechanism pack and the acceptance fixture is isolated under testdata.
    tests:
      - TestTestNameMatcher_AllMatchAPIAndNoBashMechanismInCore

contracts:
  - file: pkg/gate/step_testverify.go
    provides:
      - name: TestNameMatcher.FindNames
        kind: method
        signature: "func (m TestNameMatcher) FindNames(line string) []string"
      - name: TestNameMatcher.FindName
        kind: method
        signature: "func (m TestNameMatcher) FindName(line string) (string, bool)"
        notes: "Compatibility surface: returns the first deterministic FindNames result, or empty/false."
      - name: collectTestFuncNamesScoped
        kind: function
        signature: "func collectTestFuncNamesScoped(codeDir string, scope *GateScope, classifier SourceClassifier, matcher TestNameMatcher) map[string]string"
    consumes:
      - source: pkg/gate
        name: SourceClassifier
        kind: type
      - source: pkg/gate
        name: GateScope
        kind: type
---

# Test Name Discovery Drops Later Regex Matches on the Same Line

## Problem

Core's pack-declared test-name discovery has the right language-neutral inputs but
silently truncates a valid line to one discovered name. `TestNameMatcher.FindName` loops
over the compiled declared patterns and calls `regexp.FindStringSubmatch(line)`. It
returns immediately on the first match of the first matching pattern. The collection
loop calls that API once per physical line, so neither a second match of the same pattern
nor any later pattern match can enter the `name -> filePath` index.

ISSUE-187's immutable SPEC-072 acceptance exposed the defect without requiring new
syntax in Core. The external Bash pack correctly classifies the verifier files and
declares a generic Bash-function regex. Of 44 distinct function declarations, Core finds
36. The missing eight are exactly the later declarations on semicolon-separated physical
lines. Their first declarations are found; their later declarations are invisible. The
pack regex is not the limiting factor: applying it as an all-match regex sees both.

This is a generic Core consumer bug in the implemented SPEC-045 surface. Any pack whose
language or test style permits multiple declarations on one physical line can lose names
the same way. Encoding a Bash-specific separator or changing the Bash pack to compensate
would hide the consumer defect and violate the thin, language-neutral executor boundary.

## Solution

Add an all-match API on `TestNameMatcher` and make collection consume it. For each
physical line, gather every capture-group-1 match from every declared pattern, order
matches by source position with declared-pattern order as the stable tie-break, and
deduplicate overlap before returning names. Keep `FindName` as a compatibility wrapper
over the first result unless the implementation can cleanly migrate every caller in the
same bounded change.

Do not alter file classification, capability polarity, mandated-test due status, path
attribution, or verdict routing. The collection loop simply inserts every enumerated name
into the same existing map it already uses. Invalid regexes still fail in
`NewTestNameMatcher`, and the matcher learns no language syntax.

## Verification

Verification is executable and non-numeric because the issue is a bounded enumeration
correctness defect, not a new coverage domain. The focused tests must falsify the old
implementation with two matches on one physical line, pattern overlap, capture-group
edge cases, path resolution of the later name, and unchanged Go/TypeScript behavior.
The authoritative command is `./bin/backstop gate --all`: focused Go tests are
implementation evidence executed through the declared Go toolchain pack and assembled
gate, not a competing raw-test acceptance surface.

The end-to-end acceptance uses the immutable Core/SPEC-072 corpus at reachable commit
`2855ccd1438c455fc2a6842978c15e5cf582ff5b` and tree
`97a9480b579b8aac1f4fec8d8294c70aee56a232`, plus the released Bash pack declarations
already pinned by ISSUE-187. It must demonstrate the measured 36/44
pre-fix result and 44/44 post-fix result without changing the corpus or pack, then run the
existing terminal fixture so all 47 references resolve and the discovery-caused drift
clears. The test may stage the pinned corpus in a disposable directory; it must not edit
ISSUE-187, its plan, the pack workspace, or SPEC-072. It may use a self-contained
checked-in `testdata` reference-pack fixture reproducing only the required
classification/name-pattern declarations and the minimal deterministic engine
declaration, producer, and converter required for the assembled gate. Those fixture
bytes are pinned and inert outside `testdata`; they are never production/default
registered, do not stand in for an external Bash pack release, and must not import,
install, duplicate, or depend on unfinished PLAN-ISSUE-187 output.

## References

- `pkg/gate/step_testverify.go` — `TestNameMatcher.FindName`,
  `collectTestFuncNamesScoped`, and `ResolveMandatedTestPaths`.
- `specs/SPEC-045-de-go-test-verification-discovery.spec.md` — implemented
  pack-declared discovery contract whose single-match implementation is incomplete.
- ISSUE-187 — external Bash toolchain pack and immutable SPEC-072 acceptance that exposed
  the 36/44 result.
- `specs/SPEC-072-public-product-model.spec.md` — 47 mandated references resolving to
  44 distinct Bash function declarations.

## Existence-in-world Check

Searched every current `issues/` and `bundles/` artifact for same-line test declarations,
all-match enumeration, `FindStringSubmatch`, `FindName`, multiple regex matches, pattern
overlap, and semicolon-separated declarations. No open issue or bundle seed owns this
generic consumer defect. BUNDLE-012/SPEC-045 delivered the declaration-driven discovery
surface but do not own this newly observed truncation bug.
