---
title: "Bash Toolchain Pack for Shell Test Discovery and Verdicts"
schema_version: issue/v1

issue:
  id: ISSUE-187
  title: "Bash Toolchain Pack for Shell Test Discovery and Verdicts"
  type: enhancement
  status: ready
  created: "2026-08-25"

complexity:
  scope: contained
  uncertainty: known
  risk: moderate

verification:
  level: build
  test_command: "backstop pack test && ./scripts/release-acceptance.sh"

implementation:
  summary: >
    Author and release the external `backstop-ai/bash-toolchain` mechanism pack. The
    pack supplies existing Core with shell-test file classification, Bash function-name
    extraction, and one `gate_type: test` engine whose producer/convert path executes
    the repository's canonical shell verifier exactly once and normalizes its overall or
    suite diagnostics to SARIF. Prove the pack against backstop-core's SPEC-072
    terminal-promotion corpus without changing Core,
    SPEC-072, PLAN-SPEC-072, or any substantiveness behavior.
  package: github.com/backstop-ai/bash-toolchain

requirements:
  - id: REQ-001
    text: >
      Publish an external enforcement/mechanism pack named
      `backstop-ai/bash-toolchain`. Its manifest must use Core's existing declaration
      surfaces only: `classification.test`, `test_name_patterns`, and an engine with
      `gate_type: test`. The implementation must require no Backstop Core feature,
      schema, matcher, dispatcher, or policy change.
      The work includes bootstrapping the previously absent external repository,
      establishing its pack validation/fixture harness, and publishing an immutable
      version consumed by the acceptance fixture; an unversioned local-only directory
      is not delivery.
  - id: REQ-002
    text: >
      The pack must classify `scripts/tests/**/*.sh` as test files and must also
      classify the exact top-level verifier path `scripts/verify-public-product-model.sh`
      needed by the SPEC-072 corpus. Classification must remain narrow: arbitrary shell
      scripts outside those verifier/test paths are not claimed as tests merely because
      they end in `.sh`.
  - id: REQ-003
    text: >
      `test_name_patterns` must extract a valid Bash identifier from both supported
      declaration forms: `name() {` (allowing ordinary horizontal whitespace) and the
      justified Bash form `function name` with an optional `()` before `{`. Patterns
      must capture only group 1 as the function name, must not treat calls, comments,
      strings, or malformed declarations as tests, and must compile under Core's
      existing Go regular-expression matcher.
  - id: REQ-004
    text: >
      The pack must provide a project-wide Bash test engine and pack-owned
      producer/converter. Dynamic execution is repository-command-level, separate from
      static mandated-test discovery: the producer must execute the repository's
      canonical top-level verifier exactly once, or an equivalently explicit
      pack-configured project test command exactly once, and preserve its combined
      diagnostics and exit status. It must not source self-executing suite scripts or
      individually reinvoke their functions. The converter must emit valid SARIF for an
      overall or identifiable suite failure, while a passing command emits valid empty
      SARIF. Execution and normalization knowledge remain in the pack; per-function
      dynamic verdict association is not required.
  - id: REQ-005
    text: >
      On the current backstop-core SPEC-072 corpus, the installed pack must resolve all
      47 mandated-test references to exactly 44 distinct Bash function declarations.
      Duplicate references are legitimate references to the same function and must not
      be misreported as absent. Static function cardinality must not drive dynamic
      invocation cardinality: the exact SPEC-072 canonical verifier is executed once by
      the engine and dispatches all eight shell suites through the repository's own
      contract. The verifier and all suites must still pass independently of the gate.
  - id: REQ-006
    text: >
      With SPEC-072 changed to `implemented` and PLAN-SPEC-072 changed to `completed` in
      an isolated acceptance fixture, and with the released pack declared, locked, and
      installed, `test_verification` must report zero absent mandated tests and
      `artifact_status_drift` must report zero broken-promise violations for SPEC-072.
      This terminal-transition proof must use the real assembled gate and the real 47-to-44
      corpus, not hand-built discovery maps or suppressed findings. The consumer corpus
      must be checked out at immutable Backstop Core commit
      `2855ccd1438c455fc2a6842978c15e5cf582ff5b` (tree
      `97a9480b579b8aac1f4fec8d8294c70aee56a232`); only the two isolated status edits may
      differ, so later moving-branch content cannot silently change acceptance.
  - id: REQ-007
    text: >
      The Bash pack must compose with the installed Go toolchain pack. Adding it must not
      change Go test-file discovery, Go `func Test...` name extraction, or Go test-engine
      verdict routing; one mixed fixture must statically discover both a Bash function
      name and a Go test name, execute the Bash canonical verifier once, and execute the
      Go test engine through their existing pack declarations.
  - id: REQ-008
    text: >
      Failure modes must remain loud under existing Core semantics. An absent Bash pack
      must not make the SPEC-072 terminal fixture green; a missing or malformed
      classification/name declaration must leave the affected Bash mandated tests
      observably unresolved or fail manifest/matcher construction; a missing Bash
      executable, canonical verifier failure, producer crash, or malformed converter
      output must produce the existing blocking/config-error or test-engine verdict
      behavior.
      None may be normalized to empty successful SARIF.
  - id: REQ-009
    text: >
      Scope is limited to the Bash mechanism pack and its pack-owned fixtures, release,
      and consumer acceptance harness. Bash substantiveness rules or verdicts,
      multi-provider fan-in, per-stack/per-file applicability changes, new Core features,
      and edits to BUNDLE-009, BUNDLE-012, SPEC-072, or PLAN-SPEC-072 are prohibited.

claims:
  - id: CLM-001
    requirement: REQ-001
    text: The external bash-toolchain manifest uses only existing classification, name-pattern, and test-engine declaration surfaces.
    tests:
      - TestBashToolchainManifestUsesExistingCoreSurfaces
  - id: CLM-002
    requirement: REQ-002
    text: The pack classifies the SPEC-072 shell test tree and exact top-level verifier without broadly claiming unrelated shell scripts.
    tests:
      - TestBashToolchainClassificationIsNarrowAndComplete
  - id: CLM-003
    requirement: REQ-003
    text: Declared patterns extract supported Bash function declaration forms and reject non-declarations.
    tests:
      - TestBashToolchainNamePatternsDeclarationMatrix
  - id: CLM-004
    requirement: REQ-004
    text: The Bash engine executes the canonical project verifier exactly once without sourcing its self-executing suites and emits overall or suite SARIF only on failure.
    tests:
      - TestBashToolchainCanonicalVerifierSingleExecutionAndSARIF
  - id: CLM-005
    requirement: REQ-005
    text: The real SPEC-072 corpus maps 47 mandated references to 44 distinct Bash function declarations while the canonical verifier remains the single dynamic execution unit.
    tests:
      - TestBashToolchainSPEC072Resolves47ReferencesTo44Functions
  - id: CLM-006
    requirement: REQ-006
    text: The immutable pinned real terminal-transition fixture clears test-verification absence and artifact-status broken-promise drift without suppression.
    tests:
      - TestBashToolchainSPEC072TerminalPromotionClearsDiscoveryAndDrift
  - id: CLM-007
    requirement: REQ-007
    text: A mixed Go and Bash consumer preserves Go discovery and verdicts while adding Bash static discovery and one canonical-verifier verdict.
    tests:
      - TestBashToolchainMixedGoBashDiscoveryDoesNotRegressGo
  - id: CLM-008
    requirement: REQ-008
    text: An absent or incomplete Bash pack leaves the terminal fixture visibly non-green.
    tests:
      - TestBashToolchainAbsentOrIncompletePackIsLoud
  - id: CLM-009
    requirement: REQ-008
    text: Tool absence, canonical verifier failure, producer failure, and malformed conversion are never converted into successful empty SARIF.
    tests:
      - TestBashToolchainExecutionFailureMatrixIsLoud
  - id: CLM-010
    requirement: REQ-009
    text: The delivered change contains no Core, substantiveness, bundle, SPEC-072, or PLAN-SPEC-072 edits.
    tests:
      - TestBashToolchainChangeFence
  - id: CLM-011
    requirement: REQ-001
    text: The accepted external bash-toolchain candidate is published immutably and a fresh consumer installs the remote release at the same accepted identity.
    tests:
      - TestBashToolchainImmutableReleaseInstallAcceptance

contracts:
  - file: pack.yml
    provides:
      - name: bash_test_classification
        kind: variable
        signature: "classification.test: [scripts/tests/**/*.sh, scripts/verify-public-product-model.sh]"
      - name: bash_test_name_patterns
        kind: variable
        signature: "test_name_patterns: Bash name() and function name declaration regexes with capture group 1"
      - name: bash_test_engine
        kind: variable
        signature: "engines.bash-test { gate_type: test, scope_kind: project-wide, project command, producer, convert, crash_guard: true }"
  - file: scripts/test-produce.sh
    provides:
      - name: bash_test_producer
        kind: function
        signature: "bash_test_producer(project_root, canonical_test_command) -> combined diagnostics and process exit status"
  - file: scripts/test-to-sarif.sh
    provides:
      - name: bash_test_to_sarif
        kind: function
        signature: "bash_test_to_sarif(stdin canonical-verifier diagnostics) -> SARIF 2.1.0"
---

# Bash Toolchain Pack for Shell Test Discovery and Verdicts

## Problem

SPEC-072's implementation is verified by shell functions rather than Go tests. Its
claims contain 47 mandated-test references resolving to 44 distinct Bash functions in
`scripts/tests/public-product-model/**/*.sh` and the top-level
`scripts/verify-public-product-model.sh`. The exact verifier and every suite pass, and
the named functions physically exist, but Backstop cannot see them with the packs
currently installed.

Core already has the necessary language-neutral consumer surfaces. It unions installed
packs' `classification.test` globs into `SourceClassifier`, compiles their
`test_name_patterns` into `TestNameMatcher`, and routes a declared `gate_type: test`
engine into the mandated-test verdict join. The installed Go toolchain supplies only Go
test globs, a `func Test...` pattern, and `go test` execution. No installed pack supplies
the corresponding Bash declarations or execution mechanism.

The observable result appears only at the honest terminal transition. With SPEC-072
draft and PLAN-SPEC-072 draft, its promises are not due. When an isolated transition
sets SPEC-072 to `implemented` and PLAN-SPEC-072 to `completed`, the scoped gate reports
47 absent mandated tests in `test_verification` and 91 blocking broken-promise findings
in `artifact_status_drift`, while requirement traceability itself is green. The artifact
cannot truthfully reach terminal status even though its verifier is green.

The missing product is one external mechanism pack: `backstop-ai/bash-toolchain`. This
is not a request to alter Core's discovery semantics or to add shell knowledge to the
binary. It is the pack-side declaration and execution counterpart to the existing Go
toolchain pack.

## Solution

Author, validate, release, declare, lock, and install `backstop-ai/bash-toolchain`.
Keep all Bash knowledge in the pack:

1. Declare only the website verifier's shell test paths through
   `classification.test`.
2. Declare Go-regexp-compatible extraction patterns for the supported Bash function
   declaration forms, with the function identifier in capture group 1.
3. Provide one project-wide `gate_type: test` engine. Static discovery continues to read
   the classified files and declaration patterns, but dynamic execution invokes the
   repository's canonical top-level verifier exactly once. Its converter emits valid
   SARIF for overall or suite diagnostics. It never sources the eight self-executing
   suite files and never reinvokes their 44 functions individually.
4. Bootstrap the external repository and its fixture harness, validate and immutably
   release the pack, then run that release against a disposable checkout of Backstop
   Core commit `2855ccd1438c455fc2a6842978c15e5cf582ff5b` (tree
   `97a9480b579b8aac1f4fec8d8294c70aee56a232`) containing only the isolated
   SPEC-072/PLAN-SPEC-072 terminal status transition.

The acceptance is deliberately corpus-level. Merely proving one sample `foo() {}` line
matches a regex is insufficient: the real 47 references must resolve to 44 real functions,
the canonical verifier must run once, terminal drift must clear, and a deliberately
failing canonical verifier must still block.

## Verification

Verification is executable but deliberately non-numeric: this issue defines no Bash
coverage producer or coverage domain, so a percentage threshold would not measure its
completeness. Completeness is the exact requirements-to-claims-to-mandated-tests matrix,
artifact validation, and the following pack/build acceptance surfaces.

The pack's own phase-3 fixtures prove declaration extraction, single-command execution, SARIF
polarity, and loud failure handling. The consumer harness then installs the released pack
into valid disposable bundle→spec→plan consumers and runs the assembled Backstop gate there;
the pack repository itself intentionally carries no production spec corpus and is not gated as
if it did. The harness covers:

- the unmodified Go discovery corpus, to pin non-regression;
- a mixed Go+Bash repository, to prove composition without changing Go discovery; and
- immutable Backstop Core commit `2855ccd1438c455fc2a6842978c15e5cf582ff5b`, tree
  `97a9480b579b8aac1f4fec8d8294c70aee56a232`, with
  status changes applied only in the disposable SPEC-072/PLAN-SPEC-072 fixture,
  proving 47 references resolve to 44 functions and both terminal blocker classes clear.

The harness must also remove the Bash pack, remove each required declaration in turn,
make the producer and converter fail independently, and make the canonical verifier
fail. Each mutation must produce the existing loud outcome rather than a false green.
No acceptance step sources or rewrites the self-executing suite scripts.

## References

- `specs/SPEC-072-public-product-model.spec.md` — the 47 mandated-test references.
- `plans/PLAN-SPEC-072-public-product-model.plan.yml` — terminal plan transition used
  only by the disposable acceptance fixture.
- `scripts/verify-public-product-model.sh` and
  `scripts/tests/public-product-model/**/*.sh` — the passing 44-function corpus.
- `specs/SPEC-045-de-go-test-verification-discovery.spec.md` — existing pack-declared
  classification and test-name extraction consumer contract.
- `.backstop/packs/backstop-ai/go-toolchain/pack.yml` — reference mechanism-pack shape
  and the Go non-regression surface.
- ISSUE-118 — existing `gate_type: test` verdict routing and loud capability behavior.

## Existence-in-world Check

Searched all current `issues/` and `bundles/` for `bash-toolchain`, shell/Bash test
engines, Bash function declaration extraction, shell verifier classification, and
dispatcher normalization before filing. No open issue or bundle seed owns this focused
external mechanism pack. BUNDLE-012 owns Core's already-delivered language-neutral
consumer and BUNDLE-009 owns substantiveness packs; neither owns this pack, and both
Core feature work and substantiveness are explicitly outside ISSUE-187.
