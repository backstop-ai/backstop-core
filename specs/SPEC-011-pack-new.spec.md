---
title: "SPEC-011: Pack New — Rule Pack and Code Pack Scaffolding"
number: SPEC-011
created: "2026-04-04"
status: draft
schema_version: spec/v1
spec_version: 1.0.0

implementation:
  summary: >
    Implement the backstop pack new command that scaffolds rule pack (--type rule)
    and code pack (--type code) directory structures with language-specific
    templates. Rule pack scaffolding creates a standards/<language>/ directory
    with a STD-<LANG>-NNN-<slug>.standard.md file containing detection block
    templates. Code pack scaffolding creates a recipes/<language>/<slug>/ directory
    with template recipe files. Both types auto-assign the next available pack
    number by scanning existing packs, accept a --slug flag for naming, support
    JSON and human output modes, and use exit codes 0 (success) and 2 (config
    error). The command is a thin Cobra adapter that delegates scaffolding logic
    to a pkg/ package.
  package: cmd/backstop

verification:
  level: unit
  test_command: go test ./cmd/backstop/... ./pkg/pack/... -run TestPackNew -race -coverprofile=cover.out
  coverage_threshold: 90

requirements:
  - id: REQ-001
    text: >
      The backstop pack new command must accept a --type flag specifying the
      pack type. The type must be one of two valid values: rule, code. A missing
      --type flag is a usage error (exit code 2). An unrecognized type value is
      a validation error (exit code 2).
    supports: cli:REQ-005

  - id: REQ-002
    text: >
      The command must accept a --language flag specifying the target language
      for the pack. The language must be a non-empty lowercase string matching
      the pattern ^[a-z]+$. A missing --language flag is a usage error (exit
      code 2). An unrecognized or empty language value is a validation error
      (exit code 2).
    supports: cli:REQ-005

  - id: REQ-003
    text: >
      The command must accept a --slug flag providing the human-readable name
      for the pack. The slug must conform to the pattern
      ^[a-z][a-z0-9]*(-[a-z0-9]+)*$ with a minimum length of 2 and maximum
      length of 64 characters. A missing --slug flag is a usage error (exit
      code 2). An invalid slug is a validation error (exit code 2).
    supports: cli:REQ-005

  - id: REQ-004
    text: >
      For rule pack scaffolding (--type rule), the command must create a
      standard file at standards/<language>/STD-<LANG>-<NNN>-<slug>.standard.md
      where <LANG> is the uppercased language code and <NNN> is the zero-padded
      3-digit next available number. The file must contain valid frontmatter
      with title, number, created (today's date), status (active), schema_version
      (standard/v1), language, pack, and scope fields. The frontmatter must
      include a rules array with a single template rule entry including id,
      name, category, severity, description, compliance_tier, and detection
      block placeholder fields. The standards/<language>/ directory must be created if it does not
      exist.
    supports: cli:REQ-005

  - id: REQ-005
    text: >
      For code pack scaffolding (--type code), the command must create a recipe
      directory at recipes/<language>/<slug>/ containing a README.md with the
      pack name and description placeholder, and a template recipe file named
      <slug>.recipe.md. The recipes/<language>/ and recipes/<language>/<slug>/
      directories must be created if they do not exist.
    supports: cli:REQ-005

  - id: REQ-006
    text: >
      For rule packs, the command must auto-assign the next available pack
      number for the given language by scanning existing standard files in
      standards/<language>/ matching the pattern STD-<LANG>-<NNN>-*.standard.md,
      extracting the numeric suffixes, and computing the next sequential number.
      Gaps in the number sequence must not be filled. If no existing standards
      exist for the language, the number starts at 001.
    supports: cli:REQ-005

  - id: REQ-007
    text: >
      The command must support --json and default human output modes consistent
      with SPEC-005 output layer conventions. In JSON mode, the output must
      include: the pack type, the language, the slug, the created path(s), and
      a schema_version field. In human mode, the output must display the created
      path(s) and pack identifier. Both modes must produce identical underlying
      data.
    supports: cli:REQ-007

  - id: REQ-008
    text: >
      Exit codes must follow the CLI convention: 0 on successful scaffold, 2
      on configuration or usage error (invalid type, invalid language, invalid
      slug, missing required flags). There is no exit code 1 for this command
      because pack scaffolding does not produce violations.
    supports: cli:REQ-007

  - id: REQ-009
    text: >
      If the target file (for rule packs) or target directory (for code packs)
      already exists, the command must refuse to overwrite and exit with code 2
      with an error message identifying the conflicting path. For rule packs,
      this means the exact standard file path already exists. For code packs,
      this means the recipe directory already exists.
    supports: cli:REQ-005

  - id: REQ-010
    text: >
      The command must be a thin adapter consistent with SPEC-005 REQ-008. The
      command function must only parse flags, resolve the pack number, delegate
      scaffolding, and format the result. Pack scaffolding logic (template
      rendering, number resolution, directory creation) must be in a pkg/
      package, not in cmd/.
    supports: cli:REQ-005

claims:
  # REQ-001: Type flag validation — dependency matrix for pack types
  - id: CLM-001
    requirement: REQ-001
    text: "backstop pack new --type rule accepts rule as a valid pack type"
    tests:
      - TestPackNew_ValidType_Rule

  - id: CLM-002
    requirement: REQ-001
    text: "backstop pack new --type code accepts code as a valid pack type"
    tests:
      - TestPackNew_ValidType_Code

  - id: CLM-003
    requirement: REQ-001
    text: "backstop pack new with unrecognized --type value exits with code 2"
    tests:
      - TestPackNew_InvalidType_Exit2

  - id: CLM-004
    requirement: REQ-001
    text: "backstop pack new with missing --type flag exits with code 2"
    tests:
      - TestPackNew_MissingType_Exit2

  # REQ-002: Language flag validation
  - id: CLM-005
    requirement: REQ-002
    text: "Valid lowercase language string is accepted"
    tests:
      - TestPackNew_ValidLanguage_Accepted

  - id: CLM-006
    requirement: REQ-002
    text: "Missing --language flag exits with code 2"
    tests:
      - TestPackNew_MissingLanguage_Exit2

  - id: CLM-007
    requirement: REQ-002
    text: "Empty --language value exits with code 2"
    tests:
      - TestPackNew_EmptyLanguage_Exit2

  - id: CLM-008
    requirement: REQ-002
    text: "Language containing uppercase letters exits with code 2"
    tests:
      - TestPackNew_LanguageUppercase_Exit2

  - id: CLM-009
    requirement: REQ-002
    text: "Language containing digits or hyphens exits with code 2"
    tests:
      - TestPackNew_LanguageInvalidChars_Exit2

  # REQ-003: Slug validation
  - id: CLM-010
    requirement: REQ-003
    text: "Valid slug conforming to pattern is accepted"
    tests:
      - TestPackNew_ValidSlug_Accepted

  - id: CLM-011
    requirement: REQ-003
    text: "Slug starting with a digit is rejected with exit code 2"
    tests:
      - TestPackNew_SlugStartsWithDigit_Exit2

  - id: CLM-012
    requirement: REQ-003
    text: "Slug shorter than 2 characters is rejected with exit code 2"
    tests:
      - TestPackNew_SlugTooShort_Exit2

  - id: CLM-013
    requirement: REQ-003
    text: "Slug longer than 64 characters is rejected with exit code 2"
    tests:
      - TestPackNew_SlugTooLong_Exit2

  - id: CLM-014
    requirement: REQ-003
    text: "Missing --slug flag exits with code 2"
    tests:
      - TestPackNew_MissingSlug_Exit2

  # REQ-004: Rule pack scaffolding
  - id: CLM-015
    requirement: REQ-004
    text: "Rule pack creates standard file at standards/<language>/STD-<LANG>-<NNN>-<slug>.standard.md"
    tests:
      - TestPackNew_RulePack_CreatesStandardFile

  - id: CLM-016
    requirement: REQ-004
    text: "Rule pack standard file uses uppercased language code in the STD prefix"
    tests:
      - TestPackNew_RulePack_UppercasedLangPrefix

  - id: CLM-017
    requirement: REQ-004
    text: "Rule pack standard file contains valid frontmatter with all required fields"
    tests:
      - TestPackNew_RulePack_FrontmatterFields

  - id: CLM-018
    requirement: REQ-004
    text: "Rule pack standard file frontmatter uses today's date and active status"
    tests:
      - TestPackNew_RulePack_FrontmatterDefaults

  - id: CLM-019
    requirement: REQ-004
    text: "Rule pack standard file frontmatter contains a rules array with template rule and detection block placeholder"
    tests:
      - TestPackNew_RulePack_TemplateRuleBody

  - id: CLM-020
    requirement: REQ-004
    text: "Rule pack creates standards/<language>/ directory if it does not exist"
    tests:
      - TestPackNew_RulePack_CreatesDirectoryIfMissing

  # REQ-005: Code pack scaffolding
  - id: CLM-021
    requirement: REQ-005
    text: "Code pack creates recipe directory at recipes/<language>/<slug>/"
    tests:
      - TestPackNew_CodePack_CreatesRecipeDirectory

  - id: CLM-022
    requirement: REQ-005
    text: "Code pack recipe directory contains a README.md with name and description placeholder"
    tests:
      - TestPackNew_CodePack_ReadmeContents

  - id: CLM-023
    requirement: REQ-005
    text: "Code pack recipe directory contains a template recipe file named <slug>.recipe.md"
    tests:
      - TestPackNew_CodePack_TemplateRecipeFile

  - id: CLM-024
    requirement: REQ-005
    text: "Code pack creates recipes/<language>/ and recipes/<language>/<slug>/ directories if they do not exist"
    tests:
      - TestPackNew_CodePack_CreatesDirectoriesIfMissing

  # REQ-006: Auto-assign pack number
  - id: CLM-025
    requirement: REQ-006
    text: "Next available number is computed by scanning existing standards for the language"
    tests:
      - TestPackNew_NumberAssign_ScansExisting

  - id: CLM-026
    requirement: REQ-006
    text: "Gaps in the number sequence are not filled"
    tests:
      - TestPackNew_NumberAssign_GapsPreserved

  - id: CLM-027
    requirement: REQ-006
    text: "When no existing standards exist for the language, number starts at 001"
    tests:
      - TestPackNew_NumberAssign_StartsAt001

  - id: CLM-028
    requirement: REQ-006
    text: "Number is zero-padded to 3 digits"
    tests:
      - TestPackNew_NumberAssign_ZeroPadded

  # REQ-007: Output modes
  - id: CLM-029
    requirement: REQ-007
    text: "JSON output includes pack type, language, slug, created paths, and schema_version"
    tests:
      - TestPackNew_Output_JSON_Fields

  - id: CLM-030
    requirement: REQ-007
    text: "Human output displays created paths and pack identifier"
    tests:
      - TestPackNew_Output_Human_Display

  - id: CLM-031
    requirement: REQ-007
    text: "JSON and human output contain identical underlying data"
    tests:
      - TestPackNew_Output_DataParity

  # REQ-008: Exit codes — dependency matrix for all exit conditions
  - id: CLM-032
    requirement: REQ-008
    text: "Exit code 0 on successful rule pack scaffold"
    tests:
      - TestPackNew_ExitCode_0_RulePackSuccess

  - id: CLM-033
    requirement: REQ-008
    text: "Exit code 0 on successful code pack scaffold"
    tests:
      - TestPackNew_ExitCode_0_CodePackSuccess

  - id: CLM-034
    requirement: REQ-008
    text: "Exit code 2 on invalid --type value"
    tests:
      - TestPackNew_ExitCode_2_InvalidType

  - id: CLM-035
    requirement: REQ-008
    text: "Exit code 2 on invalid --language value"
    tests:
      - TestPackNew_ExitCode_2_InvalidLanguage

  - id: CLM-036
    requirement: REQ-008
    text: "Exit code 2 on invalid --slug value"
    tests:
      - TestPackNew_ExitCode_2_InvalidSlug

  - id: CLM-037
    requirement: REQ-008
    text: "Exit code 2 on missing required flags"
    tests:
      - TestPackNew_ExitCode_2_MissingFlags

  # REQ-009: Conflict detection — per pack type
  - id: CLM-038
    requirement: REQ-009
    text: "Rule pack refuses to overwrite when standard file already exists, exits with code 2"
    tests:
      - TestPackNew_Conflict_RulePackFileExists

  - id: CLM-039
    requirement: REQ-009
    text: "Code pack refuses to create when recipe directory already exists, exits with code 2"
    tests:
      - TestPackNew_Conflict_CodePackDirExists

  - id: CLM-040
    requirement: REQ-009
    text: "Conflict error message identifies the conflicting path"
    tests:
      - TestPackNew_Conflict_ErrorIdentifiesPath

  # REQ-010: Thin adapter
  - id: CLM-041
    requirement: REQ-010
    text: "Command function delegates pack scaffolding to pkg/ package"
    tests:
      - TestPackNew_ThinAdapter_DelegatesScaffolding

  - id: CLM-042
    requirement: REQ-010
    text: "Command function delegates number resolution to pkg/ package"
    tests:
      - TestPackNew_ThinAdapter_DelegatesNumberResolution

contracts:
  - file: cmd/backstop/pack_new.go
    provides:
      - name: NewPackNewCommand
        kind: function
        signature: "func NewPackNewCommand() *cobra.Command"
        notes: "Cobra command for backstop pack new --type <type> --language <lang> --slug <slug>"
    consumes:
      - source: github.com/spf13/cobra
        name: Command
        kind: type
      - source: pkg/pack
        name: ScaffoldPack
        kind: function
      - source: pkg/pack
        name: ResolvePackNumber
        kind: function
      - source: cmd/backstop/output
        name: Formatter
        kind: interface

  - file: pkg/pack/scaffold.go
    provides:
      - name: ScaffoldPack
        kind: function
        signature: "func ScaffoldPack(opts ScaffoldOptions) (*ScaffoldResult, error)"
        notes: "Creates pack directory structure and template files for the given pack type and language"
      - name: ScaffoldOptions
        kind: type
        signature: "type ScaffoldOptions struct"
        notes: "Options for pack scaffolding: pack type, language, slug, number, project root"
      - name: ScaffoldResult
        kind: type
        signature: "type ScaffoldResult struct"
        notes: "Result of scaffolding: created paths, pack identifier, pack type, language"
      - name: ValidPackTypes
        kind: variable
        signature: "var ValidPackTypes map[string]bool"
        notes: "Registry of valid pack types: rule, code"
    consumes: []

  - file: pkg/pack/number.go
    provides:
      - name: ResolvePackNumber
        kind: function
        signature: "func ResolvePackNumber(language string, projectRoot string) (int, error)"
        notes: "Scans standards/<language>/ for existing standards and returns the next available number"
      - name: ValidateLanguage
        kind: function
        signature: "func ValidateLanguage(language string) error"
        notes: "Validates language string against ^[a-z]+$ pattern"
    consumes: []
---

# SPEC-011: Pack New — Rule Pack and Code Pack Scaffolding

## Overview

The `backstop pack new` command scaffolds new rule packs and code packs with
language-specific directory structures and template files. It is the entry point
for developers and agents creating new enforcement content (rule packs that
define standards) or implementation guidance content (code packs that provide
recipes).

This spec covers bundle REQ-005: "backstop pack new must scaffold rule pack
(--type rule) and code pack (--type code) directory structures with
language-specific templates."

The command depends on the CLI foundation (SPEC-005) for Cobra integration,
output formatting, and exit code handling.

## Requirements

Requirements are defined in frontmatter.

### Pack Types

Two pack types are supported, each producing different directory structures:

| Pack Type | Output Path | Contents |
|-----------|-------------|----------|
| `rule` | `standards/<language>/STD-<LANG>-<NNN>-<slug>.standard.md` | Standard file with frontmatter and template rule entry |
| `code` | `recipes/<language>/<slug>/` | Directory with README.md and template recipe file |

Only `rule` and `code` are valid pack types. Any other value is rejected with
exit code 2.

### Flags

| Flag | Required | Description | Validation |
|------|----------|-------------|------------|
| `--type` | Yes | Pack type: `rule` or `code` | Must be one of two valid values |
| `--language` | Yes | Target language (e.g., `go`, `typescript`) | Must match `^[a-z]+$` |
| `--slug` | Yes | Human-readable pack name | Must match `^[a-z][a-z0-9]*(-[a-z0-9]+)*$`, length 2-64 |

### Rule Pack Scaffolding

Rule packs create a single `.standard.md` file in the language's standards
directory. The filename follows the pattern `STD-<LANG>-<NNN>-<slug>.standard.md`
where `<LANG>` is the uppercased language code. The file includes:

- Frontmatter: title, number, created, status, schema_version, language, pack, scope,
  and a rules array with a template rule entry containing id, name, category,
  severity, description, compliance_tier, and detection block placeholder

The next available number is determined by scanning existing standard files in
`standards/<language>/` matching `STD-<LANG>-<NNN>-*.standard.md`. Gaps are
not filled. First pack for a language starts at 001.

### Code Pack Scaffolding

Code packs create a recipe directory in `recipes/<language>/<slug>/` containing:

- `README.md` with pack name and description placeholder
- `<slug>.recipe.md` template recipe file

### Exit Codes

| Code | Condition |
|------|-----------|
| 0 | Pack scaffolded successfully |
| 2 | Invalid type, invalid language, invalid slug, missing flags, conflict |

There is no exit code 1 for this command because pack scaffolding does not
produce violations.

## Implementation

The implementation creates files in two packages: `cmd/backstop/` (thin command
adapter) and `pkg/pack/` (scaffolding logic, number resolution, language
validation).

### Pass 1: Pack Type Registry and Language Validation (pkg/pack/number.go)

Define `ValidPackTypes` map with `rule` and `code` entries. Implement
`ValidateLanguage()` that checks the language string against `^[a-z]+$`.

### Pass 2: Number Resolution (pkg/pack/number.go)

Implement `ResolvePackNumber()` that:
1. Lists files in `standards/<language>/` matching `STD-<LANG>-<NNN>-*.standard.md`
2. Extracts 3-digit numeric suffixes from filenames
3. Computes the next sequential number (max existing + 1)
4. Returns 1 if no existing standards exist for the language
5. Preserves gaps — does not fill unused numbers

### Pass 3: Rule Pack Template (pkg/pack/scaffold.go)

Implement rule pack scaffolding within `ScaffoldPack()`:
1. Resolve the next available number via `ResolvePackNumber()`
2. Compute filename: `STD-<LANG>-<NNN>-<slug>.standard.md` with uppercased
   language code and zero-padded 3-digit number
3. Render frontmatter: title (from slug), number (`STD-<LANG>-<NNN>`), created
   (today), status (`active`), schema_version (`standard/v1`), language, pack
   (language name), scope (`language`)
4. Render frontmatter rules array with one template rule including all required
   fields and a detection block placeholder
5. Create `standards/<language>/` if it does not exist
6. Write the file; fail if it already exists

### Pass 4: Code Pack Template (pkg/pack/scaffold.go)

Implement code pack scaffolding within `ScaffoldPack()`:
1. Compute directory path: `recipes/<language>/<slug>/`
2. Create `README.md` with pack name header and description placeholder
3. Create `<slug>.recipe.md` template recipe file with placeholder content
4. Create all parent directories if they do not exist
5. Fail if `recipes/<language>/<slug>/` already exists

### Pass 5: Cobra Command (cmd/backstop/pack_new.go)

Implement `NewPackNewCommand()` that:
1. Defines the Cobra command with `pack new` usage and --type, --language,
   --slug flags
2. Validates --type (must be in `ValidPackTypes`)
3. Validates --language via `ValidateLanguage()`
4. Validates --slug via slug validation (reuses `pkg/scaffold/slug.go` or
   equivalent logic in `pkg/pack/`)
5. Calls `ScaffoldPack()` to create the pack structure
6. Formats output via the Formatter interface
7. Returns exit code 0 on success, 2 on any error

## Verification

Claims are defined in frontmatter. Unit-level verification with 90% coverage
threshold on both the cmd/backstop/ and pkg/pack/ package trees. Tests use
temporary directories for filesystem operations.

## Sharp Edges

- **Language validation is permissive by design.** The `^[a-z]+$` pattern accepts
  any lowercase alphabetic string as a language. This means `backstop pack new
  --type rule --language xyzzy` will happily create a `standards/xyzzy/` directory.
  There is no registry of "known languages" — the command trusts the user. A future
  enhancement could warn on unrecognized languages, but the current design
  prioritizes extensibility over validation.

- **Code pack template content is underspecified.** The bundle says "template files
  for the code pack recipe" but does not define what a recipe file looks like. The
  initial implementation must produce some reasonable placeholder, but the template
  will likely need revision once the recipe format stabilizes. The recipe format
  depends on decisions not yet captured in any ADR or bundle.

- **Rule pack number assignment has a race condition.** Two developers running
  `backstop pack new --type rule --language go` simultaneously will both scan the
  same directory and may compute the same next number. Unlike artifact IDs (which
  use git tag reservation per OQ-8), pack numbers use local filesystem scan only.
  The git merge conflict is the resolution, same as the offline fallback for
  artifact IDs.

- **Existing standards directories have non-uniform structure.** The current
  repository has `standards/go/`, `standards/typescript/`, and `standards/core/`.
  The `core` directory does not correspond to a language — it is a cross-language
  standard. The scanner must only count standards matching the requested language's
  prefix pattern, not all files in the directory. The `core` standards use a
  different prefix (`STD-CORE-NNN`) and will not collide with language-specific
  numbering.

- **No code pack number assignment.** Code packs use the slug as their directory
  name, not a numeric ID. This means two code packs for the same language cannot
  share a slug, but there is no global numbering scheme. This is intentional —
  code packs are named by what they do, not by sequence.

## Review Questions

1. Should the command validate that the language has an existing standards directory
   structure (e.g., `standards/go/rules/` subdirectories), or should it only create
   the minimal structure and leave further organization to the developer?

2. The rule pack template creates a single rule entry. Should the template include
   multiple detection strategy examples (semgrep, metric, native) to guide the
   developer, or should it use the simplest possible template and rely on existing
   standards as examples?

3. For code pack scaffolding, should the command accept a --name flag that is
   distinct from --slug (human-readable title vs filesystem name), or is the slug
   sufficient as both identifier and display name?

4. Should `backstop pack new --type rule --language core` be supported for creating
   cross-language standards, or should `core` be a reserved/special-cased language
   value with its own scaffolding behavior?

5. If `pkg/scaffold/slug.go` already exists from SPEC-007, should `pkg/pack/`
   import and reuse that slug validation, or should it have its own copy? Sharing
   creates a dependency between the two packages; duplicating creates drift risk.

## References

- Bundle: cli (REQ-005 — backstop pack new scaffolding)
- SPEC-005: CLI Foundation (Cobra skeleton, output layer, exit codes)
- SPEC-007: Artifact New (parallel command for artifact scaffolding)
- SPEC-009: Pack Compile (standards compilation, related pack infrastructure)
- STD-GO-001: Example standard file structure (template reference)
- ADR-0008: CLI as agent API
- D-069: CLI as universal agent API
