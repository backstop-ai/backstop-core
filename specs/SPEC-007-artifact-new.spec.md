---
title: "SPEC-007: Artifact New — Scaffolding, Auto-ID, Template Rendering"
number: SPEC-007
created: "2026-04-04"
status: draft
schema_version: spec/v1
spec_version: 1.0.0

implementation:
  summary: >
    Implement the backstop artifact new <type> command that scaffolds artifacts
    for all seven types (spec, plan, issue, adr, directive, bundle, capability)
    with auto-assigned next-available IDs via git annotated tag reservation.
    The command renders type-specific templates with correct frontmatter,
    accepts a --slug flag for the human-readable filename suffix, writes the
    file to the correct directory (specs/, plans/, issues/, adrs/, directives/,
    bundles/, capabilities/), and supports both JSON and human output modes
    with consistent exit codes (0 success, 1 violations, 2 config error).
    ID reservation uses git annotated tags in backstop/<type>/<number> format
    with atomic push, retry on conflict, and offline fallback to local
    filesystem scan.
  package: cmd/backstop

verification:
  level: unit
  test_command: go test ./cmd/backstop/... -run TestArtifactNew -race -coverprofile=cover.out
  coverage_threshold: 90

requirements:
  - id: REQ-001
    text: >
      The backstop artifact new command must accept exactly one positional
      argument specifying the artifact type. The type must be one of seven
      valid values: spec, plan, issue, adr, directive, bundle, capability.
      An unrecognized type is a validation error (exit code 2). A missing
      type argument is a usage error (exit code 2).
    supports: cli:REQ-002

  - id: REQ-002
    text: >
      The command must accept a --slug flag providing the human-readable
      filename suffix. The slug must conform to the pattern ^[a-z][a-z0-9]*(-[a-z0-9]+)*$
      with a minimum length of 2 and maximum length of 64 characters.
      An invalid slug is a validation error (exit code 2). If --slug is
      not provided, the command must prompt for or require a slug (exit
      code 2 if missing).
    supports: cli:REQ-002

  - id: REQ-003
    text: >
      The command must auto-assign the next available ID for the given
      artifact type. ID assignment uses git annotated tag reservation:
      fetch existing backstop/<type>/* tags, compute the next sequential
      number, create a git annotated tag backstop/<type>/<number>, and
      push the tag atomically. If the push fails due to conflict (another
      developer reserved the same number), the command must retry with the
      next available number. Gaps from unused reservations are acceptable
      and must not be filled.
    supports: cli:REQ-002

  - id: REQ-004
    text: >
      When git is unavailable (not a git repository, or git binary not
      found), the command must fall back to local filesystem scan: scan
      the target directory for existing artifacts, extract their numeric
      IDs, and assign the next sequential number. The fallback must not
      attempt any git operations. The command must not fail solely because
      git is unavailable.
    supports: cli:REQ-002

  - id: REQ-005
    text: >
      Each artifact type must be written to its correct target directory
      relative to the project root. The mapping is: spec -> specs/,
      plan -> plans/, issue -> issues/, adr -> adrs/, directive -> directives/,
      bundle -> bundles/, capability -> capabilities/. The command must
      create the target directory if it does not exist. If the file already
      exists at the target path, the command must refuse to overwrite it
      and exit with code 1.
    supports: cli:REQ-002

  - id: REQ-006
    text: >
      Each artifact type must produce a file with the correct filename
      pattern. The patterns are: spec -> SPEC-<NNN>-<slug>.spec.md,
      plan -> PLAN-SPEC-<NNN>-<slug>.plan.yml, issue -> ISSUE-<NNN>-<slug>.md,
      adr -> ADR-<NNNN>-<slug>.adr.md, directive -> D-<NNN>-<slug>.directive.md,
      bundle -> <slug>.bundle.md, capability -> CAP-<NNN>-<slug>.capability.md.
      The numeric portion must use zero-padded formatting matching the
      type's ID pattern (3 digits for spec/issue/directive/capability,
      4 digits for adr). Plans and bundles have type-specific naming rules
      as described.
    supports: cli:REQ-002

  - id: REQ-007
    text: >
      Each artifact type must be scaffolded with type-specific frontmatter
      containing all required metadata fields for that type's schema. At
      minimum: spec requires title, number, created, status, schema_version,
      spec_version; plan requires plan_id, spec_id, created, status; issue
      requires title, schema_version, and the issue nested block with id,
      title, type, status, created; adr requires title, number, created,
      status, schema_version, deciders, decisions; bundle requires title,
      schema_version, and the bundle nested block; directive and capability
      require title, number, created, status, schema_version. All date
      fields must default to today's date. Status fields must default to
      the initial status for that type (draft for spec/plan/adr, open for
      issue, idea for bundle maturity).
    supports: cli:REQ-002

  - id: REQ-008
    text: >
      Each artifact type that uses markdown (spec, issue, adr, directive,
      bundle, capability) must be scaffolded with the required sections
      from its schema as empty markdown headings. Plan artifacts, being
      pure YAML, must be scaffolded with an empty phases array. Spec
      scaffolds must include Overview, Requirements, Implementation, and
      Verification sections. ADR scaffolds must include Context, Decision,
      and Consequences sections.
    supports: cli:REQ-002

  - id: REQ-009
    text: >
      The command must support --json and default human output modes
      consistent with SPEC-005 output layer conventions. In JSON mode,
      the output must include: the artifact type, the assigned ID, the
      file path written, and a schema_version field. In human mode, the
      output must display the created file path and assigned ID. Both
      modes must produce identical underlying data.
    supports: cli:REQ-007

  - id: REQ-010
    text: >
      Exit codes must follow the CLI convention: 0 on successful scaffold,
      1 when the target file already exists (conflict), 2 on configuration
      or usage error (invalid type, invalid slug, missing arguments, git
      tag push failure after all retries exhausted in online mode). Exit
      code 2 takes precedence over exit code 1.
    supports: cli:REQ-007

  - id: REQ-011
    text: >
      The git tag reservation must use annotated tags (not lightweight
      tags) in the format backstop/<type>/<number> where type is the
      artifact type name and number is the zero-padded numeric ID. The
      tag message must include the creation timestamp and the slug. The
      push must target only the specific tag, not all tags. The retry
      logic must attempt at most 3 retries on conflict before failing.
    supports: cli:REQ-002

  - id: REQ-012
    text: >
      The command must be a thin adapter consistent with SPEC-005 REQ-008.
      The command function must only parse flags, resolve the ID, render
      the template, write the file, and format the result. Template
      rendering logic and ID resolution logic must be in a pkg/ package,
      not in cmd/.
    supports: cli:REQ-002

claims:
  # REQ-001: Type argument validation — all 7 types
  - id: CLM-001
    requirement: REQ-001
    text: "backstop artifact new spec accepts spec as a valid type"
    tests:
      - TestArtifactNew_ValidType_Spec

  - id: CLM-002
    requirement: REQ-001
    text: "backstop artifact new plan accepts plan as a valid type"
    tests:
      - TestArtifactNew_ValidType_Plan

  - id: CLM-003
    requirement: REQ-001
    text: "backstop artifact new issue accepts issue as a valid type"
    tests:
      - TestArtifactNew_ValidType_Issue

  - id: CLM-004
    requirement: REQ-001
    text: "backstop artifact new adr accepts adr as a valid type"
    tests:
      - TestArtifactNew_ValidType_ADR

  - id: CLM-005
    requirement: REQ-001
    text: "backstop artifact new directive accepts directive as a valid type"
    tests:
      - TestArtifactNew_ValidType_Directive

  - id: CLM-006
    requirement: REQ-001
    text: "backstop artifact new bundle accepts bundle as a valid type"
    tests:
      - TestArtifactNew_ValidType_Bundle

  - id: CLM-007
    requirement: REQ-001
    text: "backstop artifact new capability accepts capability as a valid type"
    tests:
      - TestArtifactNew_ValidType_Capability

  - id: CLM-008
    requirement: REQ-001
    text: "backstop artifact new with unrecognized type exits with code 2"
    tests:
      - TestArtifactNew_InvalidType_Exit2

  - id: CLM-009
    requirement: REQ-001
    text: "backstop artifact new with no type argument exits with code 2"
    tests:
      - TestArtifactNew_MissingType_Exit2

  # REQ-002: Slug validation
  - id: CLM-010
    requirement: REQ-002
    text: "Valid slug conforming to pattern is accepted"
    tests:
      - TestArtifactNew_ValidSlug_Accepted

  - id: CLM-011
    requirement: REQ-002
    text: "Slug starting with a digit is rejected with exit code 2"
    tests:
      - TestArtifactNew_SlugStartsWithDigit_Exit2

  - id: CLM-012
    requirement: REQ-002
    text: "Slug containing uppercase letters is rejected with exit code 2"
    tests:
      - TestArtifactNew_SlugUppercase_Exit2

  - id: CLM-013
    requirement: REQ-002
    text: "Slug shorter than 2 characters is rejected with exit code 2"
    tests:
      - TestArtifactNew_SlugTooShort_Exit2

  - id: CLM-014
    requirement: REQ-002
    text: "Slug longer than 64 characters is rejected with exit code 2"
    tests:
      - TestArtifactNew_SlugTooLong_Exit2

  - id: CLM-015
    requirement: REQ-002
    text: "Missing --slug flag exits with code 2"
    tests:
      - TestArtifactNew_MissingSlug_Exit2

  # REQ-003: Git tag ID reservation
  - id: CLM-016
    requirement: REQ-003
    text: "ID is assigned by fetching existing tags and computing next sequential number"
    tests:
      - TestArtifactNew_GitTagReservation_NextSequential

  - id: CLM-017
    requirement: REQ-003
    text: "Git annotated tag is created in backstop/<type>/<number> format"
    tests:
      - TestArtifactNew_GitTagReservation_CreatesAnnotatedTag

  - id: CLM-018
    requirement: REQ-003
    text: "On push conflict, command retries with incremented number"
    tests:
      - TestArtifactNew_GitTagReservation_RetryOnConflict

  - id: CLM-019
    requirement: REQ-003
    text: "Gaps from unused reservations are not filled"
    tests:
      - TestArtifactNew_GitTagReservation_GapsPreserved

  # REQ-004: Offline fallback
  - id: CLM-020
    requirement: REQ-004
    text: "Falls back to local filesystem scan when not in a git repository"
    tests:
      - TestArtifactNew_OfflineFallback_NotGitRepo

  - id: CLM-021
    requirement: REQ-004
    text: "Falls back to local scan when git binary is not available"
    tests:
      - TestArtifactNew_OfflineFallback_NoGitBinary

  - id: CLM-022
    requirement: REQ-004
    text: "Local scan correctly identifies next ID from existing artifacts in directory"
    tests:
      - TestArtifactNew_OfflineFallback_ScansExistingArtifacts

  # REQ-005: Target directory — all 7 types
  - id: CLM-023
    requirement: REQ-005
    text: "Spec artifact is written to specs/ directory"
    tests:
      - TestArtifactNew_Directory_Spec

  - id: CLM-024
    requirement: REQ-005
    text: "Plan artifact is written to plans/ directory"
    tests:
      - TestArtifactNew_Directory_Plan

  - id: CLM-025
    requirement: REQ-005
    text: "Issue artifact is written to issues/ directory"
    tests:
      - TestArtifactNew_Directory_Issue

  - id: CLM-026
    requirement: REQ-005
    text: "ADR artifact is written to adrs/ directory"
    tests:
      - TestArtifactNew_Directory_ADR

  - id: CLM-027
    requirement: REQ-005
    text: "Directive artifact is written to directives/ directory"
    tests:
      - TestArtifactNew_Directory_Directive

  - id: CLM-028
    requirement: REQ-005
    text: "Bundle artifact is written to bundles/ directory"
    tests:
      - TestArtifactNew_Directory_Bundle

  - id: CLM-029
    requirement: REQ-005
    text: "Capability artifact is written to capabilities/ directory"
    tests:
      - TestArtifactNew_Directory_Capability

  - id: CLM-030
    requirement: REQ-005
    text: "Target directory is created if it does not exist"
    tests:
      - TestArtifactNew_Directory_CreatedIfMissing

  - id: CLM-031
    requirement: REQ-005
    text: "Command refuses to overwrite existing file and exits with code 1"
    tests:
      - TestArtifactNew_Directory_ExistingFileRefused

  # REQ-006: Filename patterns — all 7 types
  - id: CLM-032
    requirement: REQ-006
    text: "Spec filename follows SPEC-<NNN>-<slug>.spec.md pattern"
    tests:
      - TestArtifactNew_Filename_Spec

  - id: CLM-033
    requirement: REQ-006
    text: "Plan filename follows PLAN-SPEC-<NNN>-<slug>.plan.yml pattern"
    tests:
      - TestArtifactNew_Filename_Plan

  - id: CLM-034
    requirement: REQ-006
    text: "Issue filename follows ISSUE-<NNN>-<slug>.md pattern"
    tests:
      - TestArtifactNew_Filename_Issue

  - id: CLM-035
    requirement: REQ-006
    text: "ADR filename follows ADR-<NNNN>-<slug>.adr.md pattern with 4-digit padding"
    tests:
      - TestArtifactNew_Filename_ADR

  - id: CLM-036
    requirement: REQ-006
    text: "Directive filename follows D-<NNN>-<slug>.directive.md pattern"
    tests:
      - TestArtifactNew_Filename_Directive

  - id: CLM-037
    requirement: REQ-006
    text: "Bundle filename follows <slug>.bundle.md pattern"
    tests:
      - TestArtifactNew_Filename_Bundle

  - id: CLM-038
    requirement: REQ-006
    text: "Capability filename follows CAP-<NNN>-<slug>.capability.md pattern"
    tests:
      - TestArtifactNew_Filename_Capability

  # REQ-007: Frontmatter — all 7 types
  - id: CLM-039
    requirement: REQ-007
    text: "Spec scaffold contains required frontmatter: title, number, created, status, schema_version, spec_version"
    tests:
      - TestArtifactNew_Frontmatter_Spec

  - id: CLM-040
    requirement: REQ-007
    text: "Plan scaffold contains required frontmatter: plan_id, spec_id, created, status"
    tests:
      - TestArtifactNew_Frontmatter_Plan

  - id: CLM-041
    requirement: REQ-007
    text: "Issue scaffold contains required frontmatter: title, schema_version, issue block with id, title, type, status, created"
    tests:
      - TestArtifactNew_Frontmatter_Issue

  - id: CLM-042
    requirement: REQ-007
    text: "ADR scaffold contains required frontmatter: title, number, created, status, schema_version, deciders, decisions"
    tests:
      - TestArtifactNew_Frontmatter_ADR

  - id: CLM-043
    requirement: REQ-007
    text: "Directive scaffold contains required frontmatter: title, number, created, status, schema_version"
    tests:
      - TestArtifactNew_Frontmatter_Directive

  - id: CLM-044
    requirement: REQ-007
    text: "Bundle scaffold contains required frontmatter: title, schema_version, bundle block with name, version, created, category"
    tests:
      - TestArtifactNew_Frontmatter_Bundle

  - id: CLM-045
    requirement: REQ-007
    text: "Capability scaffold contains required frontmatter: title, number, created, status, schema_version"
    tests:
      - TestArtifactNew_Frontmatter_Capability

  - id: CLM-046
    requirement: REQ-007
    text: "Date fields default to today's date in YYYY-MM-DD format"
    tests:
      - TestArtifactNew_Frontmatter_DefaultDate

  - id: CLM-047
    requirement: REQ-007
    text: "Status defaults to draft for spec, plan, and adr"
    tests:
      - TestArtifactNew_Frontmatter_DefaultStatusDraft

  - id: CLM-048
    requirement: REQ-007
    text: "Status defaults to open for issue"
    tests:
      - TestArtifactNew_Frontmatter_DefaultStatusOpen

  - id: CLM-049
    requirement: REQ-007
    text: "Bundle maturity defaults to idea"
    tests:
      - TestArtifactNew_Frontmatter_DefaultMaturityIdea

  # REQ-008: Body sections — all 7 types
  - id: CLM-050
    requirement: REQ-008
    text: "Spec scaffold includes Overview, Requirements, Implementation, and Verification section headings"
    tests:
      - TestArtifactNew_Sections_Spec

  - id: CLM-051
    requirement: REQ-008
    text: "Plan scaffold includes empty phases array in YAML"
    tests:
      - TestArtifactNew_Sections_Plan

  - id: CLM-052
    requirement: REQ-008
    text: "Issue scaffold includes Problem section heading"
    tests:
      - TestArtifactNew_Sections_Issue

  - id: CLM-053
    requirement: REQ-008
    text: "ADR scaffold includes Context, Decision, and Consequences section headings"
    tests:
      - TestArtifactNew_Sections_ADR

  - id: CLM-054
    requirement: REQ-008
    text: "Directive scaffold includes required section headings from its schema"
    tests:
      - TestArtifactNew_Sections_Directive

  - id: CLM-055
    requirement: REQ-008
    text: "Bundle scaffold includes section headings appropriate for idea maturity"
    tests:
      - TestArtifactNew_Sections_Bundle

  - id: CLM-056
    requirement: REQ-008
    text: "Capability scaffold includes required section headings from its schema"
    tests:
      - TestArtifactNew_Sections_Capability

  # REQ-009: Output modes
  - id: CLM-057
    requirement: REQ-009
    text: "JSON output includes artifact type, assigned ID, file path, and schema_version"
    tests:
      - TestArtifactNew_Output_JSON_Fields

  - id: CLM-058
    requirement: REQ-009
    text: "Human output displays created file path and assigned ID"
    tests:
      - TestArtifactNew_Output_Human_Display

  - id: CLM-059
    requirement: REQ-009
    text: "JSON and human output contain identical underlying data"
    tests:
      - TestArtifactNew_Output_DataParity

  # REQ-010: Exit codes
  - id: CLM-060
    requirement: REQ-010
    text: "Exit code 0 on successful scaffold"
    tests:
      - TestArtifactNew_ExitCode_0_Success

  - id: CLM-061
    requirement: REQ-010
    text: "Exit code 1 when target file already exists"
    tests:
      - TestArtifactNew_ExitCode_1_FileExists

  - id: CLM-062
    requirement: REQ-010
    text: "Exit code 2 on invalid type argument"
    tests:
      - TestArtifactNew_ExitCode_2_InvalidType

  - id: CLM-063
    requirement: REQ-010
    text: "Exit code 2 on invalid slug"
    tests:
      - TestArtifactNew_ExitCode_2_InvalidSlug

  - id: CLM-064
    requirement: REQ-010
    text: "Exit code 2 when all git tag push retries are exhausted"
    tests:
      - TestArtifactNew_ExitCode_2_RetriesExhausted

  - id: CLM-065
    requirement: REQ-010
    text: "Exit code 2 takes precedence over exit code 1"
    tests:
      - TestArtifactNew_ExitCode_2_PrecedesOne

  # REQ-011: Git tag details
  - id: CLM-066
    requirement: REQ-011
    text: "Tag is annotated, not lightweight"
    tests:
      - TestArtifactNew_GitTag_IsAnnotated

  - id: CLM-067
    requirement: REQ-011
    text: "Tag format is backstop/<type>/<number>"
    tests:
      - TestArtifactNew_GitTag_Format

  - id: CLM-068
    requirement: REQ-011
    text: "Tag message includes creation timestamp and slug"
    tests:
      - TestArtifactNew_GitTag_MessageContents

  - id: CLM-069
    requirement: REQ-011
    text: "Push targets only the specific tag, not all tags"
    tests:
      - TestArtifactNew_GitTag_PushSpecificTag

  - id: CLM-070
    requirement: REQ-011
    text: "Retry logic attempts at most 3 retries before failing"
    tests:
      - TestArtifactNew_GitTag_MaxRetries

  # REQ-012: Thin adapter
  - id: CLM-071
    requirement: REQ-012
    text: "Command function delegates template rendering to pkg/ package"
    tests:
      - TestArtifactNew_ThinAdapter_DelegatesRendering

  - id: CLM-072
    requirement: REQ-012
    text: "Command function delegates ID resolution to pkg/ package"
    tests:
      - TestArtifactNew_ThinAdapter_DelegatesIDResolution

contracts:
  - file: cmd/backstop/artifact_new.go
    provides:
      - name: NewArtifactNewCommand
        kind: function
        signature: "func NewArtifactNewCommand() *cobra.Command"
        notes: "Cobra command for backstop artifact new <type> --slug <slug>"
    consumes:
      - source: github.com/spf13/cobra
        name: Command
        kind: type
      - source: pkg/scaffold
        name: Scaffold
        kind: function
      - source: pkg/scaffold
        name: ResolveID
        kind: function
      - source: cmd/backstop/output
        name: Formatter
        kind: interface

  - file: pkg/scaffold/scaffold.go
    provides:
      - name: Scaffold
        kind: function
        signature: "func Scaffold(artifactType string, id string, slug string, date string) ([]byte, error)"
        notes: "Renders artifact template with frontmatter and body sections for the given type"
      - name: ResolveID
        kind: function
        signature: "func ResolveID(artifactType string, opts IDOptions) (string, error)"
        notes: "Resolves the next available ID via git tags or local filesystem scan"
      - name: IDOptions
        kind: type
        signature: "type IDOptions struct"
        notes: "Options for ID resolution: project root, git availability, max retries"
      - name: ArtifactTypeConfig
        kind: type
        signature: "type ArtifactTypeConfig struct"
        notes: "Per-type configuration: directory, filename pattern, ID prefix, digit count, template"
      - name: ValidArtifactTypes
        kind: variable
        signature: "var ValidArtifactTypes map[string]ArtifactTypeConfig"
        notes: "Registry of all 7 artifact types and their configuration"
      - name: TargetDir
        kind: function
        signature: "func TargetDir(artifactType string, projectRoot string) string"
        notes: "Returns the target directory for the given artifact type"
      - name: Filename
        kind: function
        signature: "func Filename(artifactType string, id string, slug string) string"
        notes: "Returns the filename for the given artifact type, ID, and slug"
    consumes: []

  - file: pkg/scaffold/idresolver.go
    provides:
      - name: GitTagResolver
        kind: type
        signature: "type GitTagResolver struct"
        notes: "Resolves IDs via git annotated tags in backstop/<type>/<number> format"
      - name: LocalScanResolver
        kind: type
        signature: "type LocalScanResolver struct"
        notes: "Fallback resolver that scans the local filesystem for existing artifact IDs"
    consumes: []

  - file: pkg/scaffold/slug.go
    provides:
      - name: ValidateSlug
        kind: function
        signature: "func ValidateSlug(slug string) error"
        notes: "Validates slug against pattern, min/max length constraints"
    consumes: []
---

# SPEC-007: Artifact New — Scaffolding, Auto-ID, Template Rendering

## Overview

The `backstop artifact new <type>` command scaffolds new backstop artifacts. It is
the primary way developers and agents create specs, plans, issues, ADRs, directives,
bundles, and capabilities. The command handles the full lifecycle of artifact creation:
validating the type and slug, reserving a unique ID via git annotated tags (with
offline fallback), rendering a type-specific template with correct frontmatter and
body sections, and writing the file to the correct directory.

This spec covers bundle REQ-002: "backstop artifact new must scaffold artifacts for
all seven types (spec, plan, issue, adr, directive, bundle, capability) with
auto-assigned next-available ID via git annotated tag reservation."

The command depends on the CLI foundation (SPEC-005) for Cobra integration, output
formatting, and exit code handling.

## Requirements

Requirements are defined in frontmatter. Key design decisions from the bundle:

- **OQ-8 Resolution (Git tag reservation):** IDs are reserved atomically via git
  annotated tags in `backstop/<type>/<number>` format. Push is the atomic claim.
  Retry on conflict. Offline fallback to local filesystem scan. Gaps from unused
  reservations are acceptable.
- **DD-2 (Thin adapters):** The command is a thin Cobra adapter. Template rendering
  and ID resolution logic live in `pkg/scaffold/`.

### Supported Artifact Types

| Type | ID Pattern | Filename Pattern | Target Directory | Default Status |
|------|-----------|-----------------|-----------------|----------------|
| spec | SPEC-NNN | SPEC-NNN-slug.spec.md | specs/ | draft |
| plan | PLAN-SPEC-NNN | PLAN-SPEC-NNN-slug.plan.yml | plans/ | draft |
| issue | ISSUE-NNN | ISSUE-NNN-slug.md | issues/ | open |
| adr | ADR-NNNN | ADR-NNNN-slug.adr.md | adrs/ | draft |
| directive | D-NNN | D-NNN-slug.directive.md | directives/ | draft |
| bundle | (none) | slug.bundle.md | bundles/ | idea (maturity) |
| capability | CAP-NNN | CAP-NNN-slug.capability.md | capabilities/ | draft |

### ID Resolution Strategy

1. **Online (git available):** Fetch `backstop/<type>/*` tags from remote. Compute
   next sequential number. Create annotated tag `backstop/<type>/<number>`. Push
   the specific tag. On conflict, retry with incremented number (max 3 retries).
2. **Offline (git unavailable):** Scan target directory for existing artifacts.
   Extract numeric IDs from filenames. Assign next sequential number. No remote
   interaction.

Bundles do not have numeric IDs — their filename is `<slug>.bundle.md`. The ID
resolution step is skipped for bundles.

### Exit Codes

| Code | Condition |
|------|-----------|
| 0 | Artifact scaffolded successfully |
| 1 | Target file already exists (conflict) |
| 2 | Invalid type, invalid slug, missing arguments, git retries exhausted |

## Implementation

The implementation creates files in two packages: `cmd/backstop/` (thin command
adapter) and `pkg/scaffold/` (template rendering, ID resolution, slug validation).

### Pass 1: Artifact Type Registry (pkg/scaffold/scaffold.go)

Define `ArtifactTypeConfig` struct holding per-type configuration: directory name,
filename template, ID prefix, digit count, required frontmatter fields, required
body sections, and default status. Define `ValidArtifactTypes` map keyed by type
name with all 7 types registered.

### Pass 2: Slug Validation (pkg/scaffold/slug.go)

Implement `ValidateSlug()` that checks the slug against the pattern
`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`, minimum length 2, maximum length 64. Returns a
descriptive error on validation failure.

### Pass 3: ID Resolution — Git Tags (pkg/scaffold/idresolver.go)

Implement `GitTagResolver` that:
1. Runs `git tag -l backstop/<type>/*` to list existing tags
2. Parses numeric suffixes to find the highest reserved number
3. Creates annotated tag: `git tag -a backstop/<type>/<next> -m "<timestamp> <slug>"`
4. Pushes the tag: `git push origin backstop/<type>/<next>`
5. On push failure (exit code non-zero with "rejected" in stderr), increments the
   number and retries (up to 3 retries)
6. Fetches tags before first attempt: `git fetch --tags`

### Pass 4: ID Resolution — Local Fallback (pkg/scaffold/idresolver.go)

Implement `LocalScanResolver` that scans the target directory, matches filenames
against the type's filename pattern, extracts numeric IDs, and returns the next
sequential number.

### Pass 5: Template Rendering (pkg/scaffold/scaffold.go)

Implement `Scaffold()` that:
1. Looks up the `ArtifactTypeConfig` for the given type
2. Renders YAML frontmatter with all required fields using default values
3. Renders body sections (markdown headings for markdown types, YAML structure for
   plans)
4. Returns the complete file content as bytes

### Pass 6: Filename and Directory (pkg/scaffold/scaffold.go)

Implement `Filename()` and `TargetDir()` that compute the output filename and
directory path from the type configuration, ID, and slug.

### Pass 7: Cobra Command (cmd/backstop/artifact_new.go)

Implement `NewArtifactNewCommand()` that:
1. Defines the Cobra command with `artifact new` usage and --slug flag
2. Validates the type argument (must be in `ValidArtifactTypes`)
3. Validates the slug via `ValidateSlug()`
4. Calls `ResolveID()` to get the next available ID
5. Calls `Scaffold()` to render the template
6. Creates the target directory if needed
7. Checks for existing file (exit 1 if exists)
8. Writes the file
9. Formats output via the Formatter interface
10. Returns the appropriate exit code

## Verification

Claims are defined in frontmatter. Unit-level verification with 90% coverage
threshold on the cmd/backstop/ package tree. Tests use temporary directories and
mock git operations to avoid real git tag creation in test environments.

## Sharp Edges

- **Directive and capability schemas do not exist yet.** The artifacts/directive/
  and artifacts/capability/ directories have no schema.json files. The scaffold
  command must produce reasonable templates for these types based on the base
  artifact schema conventions, but the templates cannot be validated against a
  type-specific schema until those schemas are created. The templates may need
  updating when schemas are introduced.

- **Plan ID derivation is indirect.** Plan IDs are `PLAN-SPEC-NNN` (or
  `PLAN-ISSUE-NNN`), derived from the spec or issue they implement. The
  `backstop artifact new plan` command needs additional context (which spec/issue)
  that other types do not. The initial implementation may require a `--spec` or
  `--issue` flag for plan scaffolding, or the plan template may use a placeholder
  spec_id that the user fills in.

- **Bundle filenames have no numeric ID.** Bundles use `<slug>.bundle.md` with no
  numeric prefix. The git tag reservation and ID resolution logic must handle the
  bundle type as a special case — either skip tag reservation entirely or use the
  slug as the tag identifier. The tag format for bundles would be
  `backstop/bundle/<slug>` rather than `backstop/bundle/<number>`.

- **Git tag push requires remote access and permissions.** The atomic push will
  fail if the developer has no push access to the remote, if the remote is
  unreachable, or if the repository has tag protection rules. The fallback to
  local scan must be triggered by any git operation failure, not just conflict.

- **Race between tag creation and file write.** A developer could reserve an ID
  via git tag but crash before writing the file. This creates a gap in the ID
  sequence. The spec explicitly accepts gaps (REQ-003), but this also means a
  reserved-but-unused ID cannot be reclaimed without manually deleting the tag.

- **ADR uses 4-digit padding while other types use 3-digit.** The ID formatting
  logic must be per-type, not a shared constant. Getting this wrong would produce
  filenames that fail schema validation.

- **Offline mode produces local-only IDs with collision risk.** When two developers
  both work offline and create the same artifact type, they will get the same ID.
  This is the accepted trade-off per OQ-8 resolution — git merge conflict is the
  natural resolution.

## Review Questions

1. Should `backstop artifact new plan` require a `--spec` or `--issue` flag to
   derive the plan_id and spec_id, or should it accept the spec/issue ID as a
   second positional argument?

2. When the git fetch for existing tags fails (network issue, not "no git"), should
   the command fall back to local scan or fail with exit code 2? The current spec
   says offline fallback triggers when git is unavailable, but a fetch failure in
   an otherwise-git-enabled repository is ambiguous.

3. For bundle scaffolding, should the tag format use the slug
   (`backstop/bundle/<slug>`) to detect duplicate bundle names, or should bundles
   skip tag reservation entirely since they have no numeric ID?

4. The directive and capability types lack schemas. Should the scaffold command
   refuse to create these types until schemas exist, or should it use best-effort
   templates based on the base artifact schema?

5. If the remote has tag protection rules that reject the push, does the retry
   logic waste retries on a permanent failure? Should the command distinguish
   between a conflict rejection and a permission rejection?

6. Should the `--slug` flag have a `--title` companion that auto-derives the slug
   (e.g., "My Cool Feature" becomes "my-cool-feature"), or is manual slug entry
   sufficient for the initial implementation?

## References

- Bundle: cli (spec seed 3 — backstop artifact new)
- Bundle requirement: REQ-002
- Resolved OQ: OQ-8 (git annotated tag reservation for concurrent ID assignment)
- SPEC-005: CLI Foundation (Cobra skeleton, output layer, exit codes)
- ADR-0002: Canonical artifact primitives
- ADR-0008: CLI as agent API
- D-069: CLI as universal agent API
- D-070: Schema evolution rules
