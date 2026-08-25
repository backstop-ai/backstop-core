---
title: "go-toolchain drops justified exclusions for files absent from the Linux coverage inventory"
schema_version: issue/v1
delivered_by: PLAN-ISSUE-186

issue:
  id: ISSUE-186
  title: "go-toolchain drops justified exclusions for files absent from the Linux coverage inventory"
  type: bug
  status: closed
  created: "2026-08-24"
  closed: "2026-08-25"

complexity:
  scope: isolated
  uncertainty: known
  risk: moderate

verification:
  level: integration
  coverage_threshold: 80
  test_command: "go test ./cmd/backstop/... -run 'TestGoToolchainCoverageExclusions' -count=1"

implementation:
  summary: >
    Change the external backstop-ai/go-toolchain pack's producer to validate declarations and carry
    them through a lossless tab-framed directive, and change its parse-only coverage converter to
    emit one synthetic excluded statement record for each justified exclusion that has no profile
    or producer-listed GoFiles record. Preserve single-record annotation, deterministic output,
    fail-closed parsing, and the language-blind core boundary. Release the pack and adopt the
    released version in backstop-core's lock file.
  package: backstop-ai/go-toolchain/scripts

requirements:
  - id: REQ-001
    text: >
      For every accepted `#backstop-coverage-exclude` declaration produced from the consumer's
      tracked exclusion file, the converter MUST emit exactly one coverage record. If no profile
      block or `#backstop-gofile` output already represents the repo-relative path, it MUST append a
      synthetic record with that exact repo-relative path, `covered: 0`, `total: 0`,
      `measured: false`, `excluded: true`, metric `statement`, and the exact declared justification.
      This rule applies specifically because the declaration exists; the converter MUST NOT add a
      generic record pass for other files absent from profile and GoFiles inventories.
  - id: REQ-002
    text: >
      If an exclusion path is already represented by an aggregated profile record or a zero-statement
      GoFiles-derived record, the converter MUST annotate that existing record with `excluded: true`
      and the justification rather than emit a duplicate or alter its covered, total, measured,
      metric, path, or relative output position. Duplicate declarations for one path MUST still
      produce exactly one record; the last justified declaration in producer order MUST supply the
      record's justification, while the path's first real, GoFiles, or exclusion-only representation
      MUST determine its output position.
  - id: REQ-003
    text: >
      Output MUST remain deterministic for identical input and valid JSON. Profile records MUST retain
      first-profile order, eligible GoFiles-only records MUST follow in first GoFiles order, and
      exclusion-only records MUST follow in first exclusion-declaration order. Paths and
      justifications MUST preserve their accepted UTF-8 text through conversion without trimming,
      field collapsing, or Unicode normalization. JSON output MUST escape double quote and backslash
      as `\"` and `\\`, and accepted justification horizontal tab (HT), carriage return (CR), form
      feed (FF), and vertical tab (VT) characters as `\t`, `\r`, `\f`, and `\u000b`, so parsing the JSON
      recovers the original sequence of Unicode characters. Those four controls are accepted only as
      surrounding or internal whitespace in a justification that also contains substantive content
      under REQ-004. Within the supported text domain, every other C0 control, including backspace,
      MUST be rejected fail-closed before a directive is honored; LF is the physical-record delimiter
      and cannot occur inside a value. Path values MUST reject every C0 control. The converter MUST
      defensively drop any malformed directive that bypasses producer validation rather than emit
      invalid UTF-8, invalid JSON, or an exclusion record.
  - id: REQ-004
    text: >
      The declaration file's supported input domain MUST be valid UTF-8 text containing no NUL, with
      LF as the sole physical-record delimiter and every record LF-terminated. Invalid UTF-8, NUL, and
      an unterminated EOF tail are outside that supported text domain; this issue does not require
      POSIX sh/awk to detect, diagnose, or recover from those byte sequences, and MUST NOT claim that
      NUL is detectably rejected. Within the supported domain, the first literal TAB character is the
      sole field delimiter. An empty physical record is blank and ignored. A record whose first
      character is `#` is a comment and ignored; leading whitespace before `#` does not make a
      comment. Every other record is accepted only when text before the first TAB forms a nonempty
      canonical repo-relative path and the complete text after it contains at least one byte/code
      point outside the portable within-record ASCII whitespace set: space (SP, U+0020), horizontal
      tab (HT, U+0009), carriage return (CR, U+000D), form feed (FF, U+000C), and vertical tab (VT,
      U+000B). The path is preserved exactly, MUST NOT begin with `/`, end with `/`, contain an empty
      `/`-separated segment, or contain a segment equal to `.` or `..`, and MUST contain no C0 control,
      including HT or CR. Spaces, `#`, double quotes,
      backslashes, and non-ASCII UTF-8 characters in a path are ordinary preserved text, except that
      `#` as the first character makes the entire record a comment. The complete remainder after the
      first TAB is the justification: leading/trailing SP and every additional HT, CR, FF, or VT are
      data, not trimming or delimiters, but an empty remainder or one containing only characters from
      that five-character whitespace set is malformed. Double quotes, backslashes, non-ASCII UTF-8
      text, and HT/CR/FF/VT around or within substantive content are accepted subject to REQ-003
      escaping and preserved exactly. CR is never a record delimiter and is never silently stripped
      as a CRLF convention: a CR immediately before LF is preserved as the final justification
      character when the remainder also contains substantive content, invalidates a path if present
      there, and does not by itself make an otherwise whitespace-only remainder valid. Missing TAB,
      empty or SP/HT/CR/FF/VT-only justification, empty path, absolute/noncanonical/traversing path,
      forbidden in-domain control, and a whitespace-prefixed comment without a valid TAB-separated remainder
      MUST be dropped and create no exclusion metadata or synthetic record. With no accepted
      declaration, converter output MUST be unchanged; unrelated absent files MUST receive no generic
      synthetic record.
  - id: REQ-005
    text: >
      For every accepted declaration, `scripts/coverage-produce.sh` MUST emit exactly one physical
      directive using `printf` with the framing
      `#backstop-coverage-exclude<TAB><path><TAB><complete-justification-remainder><LF>`; it MUST NOT
      use `echo` or whitespace-field reconstruction. `scripts/coverage-to-records.sh` MUST recognize
      that literal marker-plus-TAB prefix and split the raw remainder at its first TAB delimiter,
      without `$1`/`$2`, `NF`, default awk field splitting, trimming, or rebuilding from fields. Thus
      path and the complete accepted justification remainder arrive byte-for-byte at JSON escaping.
      Producer validation and converter defensive validation MUST use the same grammar. The
      producer/convert split and all Go-specific parsing and record synthesis MUST remain owned by
      the external go-toolchain pack; backstop-core MUST gain no Go, build-tag, platform-file, or
      exclusion-synthesis branch.
  - id: REQ-006
    text: >
      Delivery MUST include a go-toolchain manifest version bump, a matching published git tag whose
      version text equals `pack.yml` exactly, a successful `tag-integrity` workflow run for that tag,
      and backstop-core adoption of that exact released version and git tag through its tracked
      `backstop.lock`. Delivery evidence MUST record the pack commit, manifest version, tag, successful
      tag-integrity run ID or URL and conclusion, and the adopted lock version/git_ref. An edit only to
      a local installed pack, an untagged manifest bump, a failed or absent integrity run, or a core
      fixture without the external release and lock adoption is not delivery.

claims:
  - id: CLM-001
    requirement: REQ-001
    text: A justified exclusion for a build-tagged source absent from both the Linux profile and GoFiles inventory emits one synthetic excluded, unmeasured 0/0 statement record with exact path and justification.
    tests:
      - TestGoToolchainCoverageExclusions_AbsentBuildTaggedFileEmitsExcludedRecord
  - id: CLM-002
    requirement: REQ-002
    text: Existing profile and GoFiles records are annotated in place, and duplicate declarations never duplicate records and use the last justified declaration.
    tests:
      - TestGoToolchainCoverageExclusions_ExistingRecordsAnnotatedWithoutDuplicates
      - TestGoToolchainCoverageExclusions_DuplicateDeclarationsLastWinsFirstPosition
  - id: CLM-003
    requirement: REQ-003
    text: Mixed profile, GoFiles-only, and exclusion-only supported UTF-8 text has stable valid JSON whose decoded paths and substantive justifications preserve accepted SP/HT/CR/FF/VT whitespace without normalization, while other in-domain controls are dropped fail-closed.
    tests:
      - TestGoToolchainCoverageExclusions_DeterministicOrderAndJSONEscaping
  - id: CLM-004
    requirement: REQ-004
    text: The LF-delimited UTF-8 record grammar preserves accepted justification whitespace exactly, requires at least one byte/code point outside SP/HT/CR/FF/VT, handles CR as specified, and drops every enumerated in-domain malformed or unsafe declaration; no accepted declaration and unrelated absent files synthesize nothing.
    tests:
      - TestGoToolchainCoverageExclusions_DeclarationGrammarAndPortableWhitespaceCases
      - TestGoToolchainCoverageExclusions_NoDeclarationDoesNotSynthesize
  - id: CLM-005
    requirement: REQ-005
    text: The producer uses printf to emit a tab-framed directive and the converter parses raw delimiters, preserving the complete justification remainder without whitespace-field reconstruction; the behavior remains pack-owned with no language-specific branch in core.
    tests:
      - TestGoToolchainCoverageExclusions_TabFramedProtocolIsLosslessAndPackOwned
  - id: CLM-006
    requirement: REQ-006
    text: The released git tag equals the bumped pack manifest version exactly and its tag-integrity workflow completes successfully, with recorded commit and run evidence.
    tests:
      - TestGoToolchainCoverageExclusions_ReleasedTagMatchesManifest
  - id: CLM-007
    requirement: REQ-006
    text: Backstop-core's tracked lock adopts the exact successfully released go-toolchain version and git tag rather than a local or fixture-only copy.
    tests:
      - TestGoToolchainCoverageExclusions_ReleasedPackIsAdopted

contracts:
  - file: scripts/coverage-produce.sh
    provides:
      - name: coverage-exclusion-directive-production
        kind: function
        signature: "accepted '<path><TAB><justification>' physical line -> '#backstop-coverage-exclude<TAB><path><TAB><justification><LF>' via printf"
    consumes:
      - source: .backstop/coverage-exclusions
        name: coverage-exclusion-declaration-line
        kind: function
  - file: scripts/coverage-to-records.sh
    provides:
      - name: coverage-exclusion-record-conversion
        kind: function
        signature: "enriched cover.out on stdin -> deterministic coverage-records JSON on stdout"
    consumes:
      - source: scripts/coverage-produce.sh
        name: tab-framed-coverage-exclusion-directive
        kind: function
---

# go-toolchain drops justified exclusions for files absent from the Linux coverage inventory

## Resolution

Delivered by `PLAN-ISSUE-186`. Commit `142999e` added executable regression coverage for
exclusion-only records, collision handling, deterministic ordering, lossless tab framing,
declaration grammar, and release/adoption identity. Commit `aefde06` adopted the published
`backstop-ai/go-toolchain` `1.9.0` / `v1.9.0` release in `backstop.yml` and `backstop.lock` and
retired the stale space-framed core fixture. The tracked lock records the released git source and
content hash. After installing the locked pack fleet, closeout reran the plan's focused
`TestGoToolchainCoverageExclusions|TestCoverageExclusion` command with Go 1.25.3; both the
`cmd/backstop` and root-package suites passed.

## Problem

The external `backstop-ai/go-toolchain` pack correctly folds justified consumer declarations from
`.backstop/coverage-exclusions` into `cover.out` as `#backstop-coverage-exclude` lines. Its sandboxed
`scripts/coverage-to-records.sh` converter stores those declarations, but emits JSON only while
iterating records discovered from profile blocks and eligible producer-listed `#backstop-gofile`
lines. An exclusion path absent from both inventories is never visited, so its valid declaration is
silently lost.

PR #8 (`fix/external-pack-sandbox`) exposed the defect on Linux CI run `32773697312`, job
`97579734097`. The uploaded `cover.out` contains both declarations:

- `pkg/packval/sandbox_linux_helper.go`, which has Linux profile blocks and a `#backstop-gofile`, is
  emitted as excluded and produces the expected warning;
- `pkg/packval/sandbox_nonlinux.go`, whose `//go:build !linux` file is absent from both the Ubuntu
  profile and Linux `go list .GoFiles`, has its full justified `#backstop-coverage-exclude` line at
  `cover.out:7724` but receives no record.

The resulting `gate-report.json` contains one blocking violation:
`coverage_unmeasured` for `pkg/packval/sandbox_nonlinux.go`, saying it is not pack-declared excluded,
despite the declaration being present in the converter input. The coverage step reports one
blocking violation plus one exclusion warning; the overall gate reports 9 passed, 1 failed, 2
skipped, 2 warned, and 1 blocking violation plus 193 warnings. This is a false positive, not an
unmeasured-code waiver: the consumer supplied the exact reviewed declaration mechanism the pack
defines, including a concrete platform/build-tag justification, and the converter dropped it solely
because the host platform omitted the file from the inventories it happens to iterate.

This is isolated to the external pack's producer/convert contract. Core already consumes generic
coverage records and must remain language-blind. Teaching core about Go build tags or synthesizing
records for arbitrary absent source would violate the thin-executor boundary and risk vacuous green.

## Solution

Define the declaration and enrichment boundary as a lossless protocol, not a whitespace convention.
The producer validates physical declaration lines against the exact grammar in REQ-004 and uses
`printf` to carry accepted values as marker, path, and complete justification separated by literal
TAB bytes. The converter parses the raw line prefix and first delimiter in the remainder; it does not
tokenize with awk's default whitespace fields or reconstruct a justification from tokens.

After aggregating profile and eligible GoFiles records, make the pack converter iterate the
justified exclusion declarations once and append a synthetic excluded record only for each path not
already represented. Preserve the existing record when one exists and annotate it; never emit two
records for one path. Duplicate declarations use explicit last-justification-wins semantics while
the first representation fixes deterministic output position. JSON escaping preserves every
supported byte, and both producer and converter reject forbidden controls fail-closed. The converter
does not infer exclusions for undeclared absent files.

The fix ships from `backstop-ai/go-toolchain`, followed by a versioned/tagged release and normal
`backstop.lock` adoption in core. No converter logic moves into core.

## Verification

Fixtures must execute the real pack scripts and prove all boundary cases in one focused corpus:

- an absent `//go:build` platform file represented only by a justified exclusion declaration;
- exclusions colliding with a real profile record and a GoFiles-derived record;
- repeated declarations for one path, proving exactly one record, last justification, and first
  representation position;
- path and justification values containing non-ASCII UTF-8 text, spaces, `#`, quotes, and backslashes;
  substantive justification with leading/trailing/internal HT, CR, FF, and VT; no Unicode
  normalization; and decoded JSON equality;
- CR immediately before LF as preserved justification data, CR in a path as invalid, a CR-only
  non-comment record as malformed, every SP/HT/CR/FF/VT-only combination as malformed, and each of
  those whitespace characters preserved when substantive content is present;
- every malformed grammar case, including missing/empty fields, absolute/traversing/noncanonical
  paths, backspace/other forbidden in-domain controls, portable-ASCII-whitespace-only justification,
  and whitespace-prefixed
  pseudo-comments;
- valid UTF-8/LF-terminated supported fixtures, with invalid UTF-8, NUL-containing input, and an
  unterminated EOF tail explicitly treated as outside the promised POSIX sh/awk detection domain;
- no declaration at all and an unrelated absent file, proving no generic synthesis;
- a protocol fixture that would change under default awk whitespace splitting, proving `printf`
  framing and raw-delimiter parsing preserve the complete justification remainder;
- byte-identical output across repeated conversion of identical mixed-order input; and
- manifest/tag exact equality, the successful tag-integrity run ID/URL and conclusion, and core
  lock version/git_ref equality, so tests and close evidence cannot pass against only a local
  installed copy.

## References

- PR #8 CI run `32773697312`, job `97579734097`, artifact `9537228210`: direct failing Linux report
  and enriched `cover.out` evidence.
- `backstop-go-toolchain-pack/scripts/coverage-produce.sh:95-136`: producer folding and fail-closed
  declaration validation.
- `backstop-go-toolchain-pack/scripts/coverage-to-records.sh:68-77,95-123`: exclusions are stored but
  output is limited to profile/GoFiles-derived `order[]` entries.
- ISSUE-045 / PLAN-ISSUE-045: precedent establishing un-sandboxed producer enrichment, parse-only
  pack converter, repo-relative coverage records, and a language-blind core.
- ISSUE-179 / PLAN-ISSUE-179: precedent requiring an external pack version bump, matching release
  tag, tag-integrity evidence, and tracked core lock adoption.
- Existence-in-world check: no other open issue or bundle seed found for exclusion-only records of
  build-tagged/platform files; ISSUE-186 is the preallocated owner of this defect.
