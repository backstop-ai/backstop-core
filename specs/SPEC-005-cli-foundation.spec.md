---
title: "SPEC-005: CLI Foundation — Command Skeleton, Embed Cohort, Config Loading, Output Layer"
number: SPEC-005
created: "2026-04-04"
status: draft
schema_version: spec/v1
spec_version: 1.0.0

implementation:
  summary: >
    Build the foundational CLI infrastructure for the backstop binary: Cobra
    command skeleton with three namespaces (artifact, code, pack) plus the
    top-level gate command, go:embed schema cohort embedding all artifact
    schemas into the binary, backstop.yml project manifest loader with
    walk-up discovery and BACKSTOP_CONFIG env var override, JSON/human output
    formatting layer controlled by a --json flag, consistent exit code
    handling (0 pass, 1 violations, 2 config error), version command
    reporting schema cohort identity, commands --json for agent discovery,
    and help generation. Commands are thin adapters with no business logic
    in cmd/ — all enforcement logic stays in pkg/.
  package: cmd/backstop, pkg/config

verification:
  level: integration
  test_command: go test ./cmd/backstop/... ./pkg/config/... -run "TestCLI|TestIntegration|TestConfig" -race -coverprofile=cover.out
  coverage_threshold: 80

requirements:
  - id: REQ-001
    text: >
      The CLI must use Cobra as the command framework with a root command
      (backstop) and three namespace commands (artifact, code, pack) plus
      a top-level gate command. Each namespace command is a Cobra command
      group. The root command displays help when invoked without subcommands.
    supports: cli:REQ-007

  - id: REQ-002
    text: >
      All artifact schemas (artifacts/*/v*/schema.json and artifacts/base/schema.json)
      must be embedded in the binary via go:embed. The embedded filesystem must
      be accessible to pkg/schema for schema loading without any runtime
      filesystem dependency. Each CLI binary version constitutes a schema
      cohort — a locked set of schemas that the binary validates against.
    supports: cli:REQ-008

  - id: REQ-003
    text: >
      The CLI must discover and load backstop.yml before any enforcement
      command (artifact validate, code check, pack compile, gate) executes.
      Discovery uses walk-up from cwd (searching parent directories until
      backstop.yml is found, similar to go.mod discovery). The BACKSTOP_CONFIG
      environment variable overrides walk-up discovery when set. If
      backstop.yml is not found, the CLI must exit with code 2 and an error
      message. If backstop.yml is found but fails schema validation, the CLI
      must exit with code 2 and a validation error message.
    supports: cli:REQ-009

  - id: REQ-004
    text: >
      All commands that produce results must support a --json flag. When
      --json is set, output is structured JSON to stdout. When --json is
      not set, output is formatted human-readable text to stdout. Both
      modes must produce identical underlying violation/result data — the
      formatter is presentation only, with zero logic. JSON output must
      include a schema_version field in every response for independent
      contract evolution.
    supports: cli:REQ-007

  - id: REQ-005
    text: >
      Exit codes must be consistent across all commands: 0 means all checks
      pass (or the command completed successfully), 1 means violations were
      found, 2 means a configuration error occurred (invalid backstop.yml,
      missing schemas, bad flags). No other exit codes are used. Exit code
      2 takes precedence over exit code 1 — if config is invalid, the CLI
      must not attempt enforcement.
    supports: cli:REQ-007

  - id: REQ-006
    text: >
      The backstop version command must print the CLI binary version, the
      embedded schema cohort identifier (derived from the set of embedded
      schema versions), and the Go version used to build the binary. When
      --json is set, this information must be a JSON object with fields
      version, schema_cohort, and go_version.
    supports: cli:REQ-011

  - id: REQ-007
    text: >
      The backstop commands --json command must return a JSON array
      describing the full command tree. Each entry must include the
      command name, full path (e.g. "artifact validate"), description,
      and available flags. This is the agent discovery endpoint — agents
      use it to learn what commands are available without parsing help text.
    supports: cli:REQ-011

  - id: REQ-008
    text: >
      Help generation must be automatic via Cobra for all commands and
      subcommands. Each command must have a Short description (one line)
      and a Long description (paragraph). The root command help must list
      all namespace groups and top-level commands. Namespace command help
      must list subcommands within that namespace.

  - id: REQ-009
    text: >
      The human output formatter must respect the NO_COLOR environment
      variable. When NO_COLOR is set (to any value), the formatter must
      not emit ANSI color codes. When NO_COLOR is not set, the formatter
      may use color codes for terminal output.

  - id: REQ-010
    text: >
      The backstop.yml loader must return a typed Go struct representing
      the project manifest. The struct must include: project name, language,
      runtimes (array of strings), enforcement configuration (security tier,
      waiver_warning_days), pack declarations (rules and code packs with
      versions), and registry configuration (scope-based resolution). The
      loader must reject unknown top-level keys to prevent typos from silently
      being ignored. A JSON schema for backstop.yml must be defined at
      artifacts/backstop-yml/v1/schema.json and the loader must validate
      against it.
    supports: cli:REQ-009

  - id: REQ-011
    text: >
      The backstop.yml schema must define required fields (project, language),
      optional fields (runtimes, enforcement, packs, registries), and their
      types and constraints. The enforcement block must support security.tier
      (enum: baseline, standard, compliance) and waiver_warning_days (integer,
      default 30). The packs block must support rules and code sub-blocks
      with pack name to version mappings. The registries block must support
      scope-based registry resolution (scope string to URL mapping).

  - id: REQ-012
    text: >
      The go:embed schema cohort must embed all schema files matching
      the pattern artifacts/*/v*/schema.json plus artifacts/base/schema.json.
      The embed directive must use an embed.FS type. A function must be
      provided that returns the list of embedded schema paths for
      introspection (used by the version command to compute the cohort
      identifier).

  - id: REQ-013
    text: >
      The BACKSTOP_CONFIG environment variable, when set to a file path,
      must bypass walk-up discovery entirely and load backstop.yml from
      the specified path. If the path does not exist or the file is
      unreadable, the CLI must exit with code 2. If BACKSTOP_CONFIG is
      set to an empty string, it must be treated as unset (walk-up
      discovery proceeds normally).

  - id: REQ-014
    text: >
      The assembled CLI binary must pass end-to-end integration tests that
      prove each command pipeline works when built and invoked as a real
      binary: backstop artifact validate produces real violations from a
      spec file, backstop code check runs semgrep against real Go code,
      backstop pack compile produces real manifests from a standard file,
      backstop gate combines artifact and code validation into a pass/fail
      result, and backstop artifact new scaffolds a real artifact with an
      auto-assigned ID. These tests build the binary and invoke it as a
      subprocess — they are not unit tests of individual packages.

claims:
  # REQ-001: Cobra command skeleton
  - id: CLM-001
    requirement: REQ-001
    text: Root command exists and displays help when invoked without subcommands
    tests:
      - TestCLI_RootCommand_ShowsHelp

  - id: CLM-002
    requirement: REQ-001
    text: Artifact namespace command exists as a command group
    tests:
      - TestCLI_ArtifactNamespace_Exists

  - id: CLM-003
    requirement: REQ-001
    text: Code namespace command exists as a command group
    tests:
      - TestCLI_CodeNamespace_Exists

  - id: CLM-004
    requirement: REQ-001
    text: Pack namespace command exists as a command group
    tests:
      - TestCLI_PackNamespace_Exists

  - id: CLM-005
    requirement: REQ-001
    text: Gate command exists as a top-level command (not under a namespace)
    tests:
      - TestCLI_GateCommand_Exists

  # REQ-002: go:embed schemas
  - id: CLM-006
    requirement: REQ-002
    text: All artifact schemas are accessible from the embedded filesystem
    tests:
      - TestCLI_EmbedCohort_AllSchemasPresent

  - id: CLM-007
    requirement: REQ-002
    text: Embedded schemas can be read and parsed as valid JSON
    tests:
      - TestCLI_EmbedCohort_SchemasParseAsJSON

  - id: CLM-008
    requirement: REQ-002
    text: Base schema is included in the embedded filesystem
    tests:
      - TestCLI_EmbedCohort_BaseSchemaPresent

  # REQ-003: backstop.yml loading — enforcement commands
  - id: CLM-009
    requirement: REQ-003
    text: Config is loaded via walk-up discovery from cwd when backstop.yml exists in a parent directory
    tests:
      - TestCLI_ConfigLoader_WalkUpDiscovery

  - id: CLM-010
    requirement: REQ-003
    text: CLI exits with code 2 when backstop.yml is not found in any parent directory
    tests:
      - TestCLI_ConfigLoader_NotFound_Exit2

  - id: CLM-011
    requirement: REQ-003
    text: CLI exits with code 2 when backstop.yml fails schema validation
    tests:
      - TestCLI_ConfigLoader_InvalidSchema_Exit2

  - id: CLM-012
    requirement: REQ-003
    text: Config is loaded before enforcement commands execute (artifact validate requires config)
    tests:
      - TestCLI_ConfigLoader_LoadedBeforeEnforcement

  # REQ-004: JSON/human output parity
  - id: CLM-013
    requirement: REQ-004
    text: "--json flag produces structured JSON to stdout"
    tests:
      - TestCLI_Output_JSONFlag_ProducesJSON

  - id: CLM-014
    requirement: REQ-004
    text: Default output (no --json) produces human-readable text
    tests:
      - TestCLI_Output_Default_ProducesHumanText

  - id: CLM-015
    requirement: REQ-004
    text: JSON and human output contain identical underlying violation data
    tests:
      - TestCLI_Output_JSONAndHuman_IdenticalData

  - id: CLM-016
    requirement: REQ-004
    text: JSON output includes schema_version field in every response
    tests:
      - TestCLI_Output_JSON_HasSchemaVersion

  # REQ-005: Exit codes
  - id: CLM-017
    requirement: REQ-005
    text: Exit code 0 when all checks pass
    tests:
      - TestCLI_ExitCode_0_OnPass

  - id: CLM-018
    requirement: REQ-005
    text: Exit code 1 when violations are found
    tests:
      - TestCLI_ExitCode_1_OnViolations

  - id: CLM-019
    requirement: REQ-005
    text: Exit code 2 on configuration error
    tests:
      - TestCLI_ExitCode_2_OnConfigError

  - id: CLM-020
    requirement: REQ-005
    text: Exit code 2 takes precedence over exit code 1 when config is invalid
    tests:
      - TestCLI_ExitCode_2_PrecedesViolations

  - id: CLM-021
    requirement: REQ-005
    text: No exit codes other than 0, 1, 2 are returned by any command
    tests:
      - TestCLI_ExitCode_OnlyValidCodes

  # REQ-006: version command
  - id: CLM-022
    requirement: REQ-006
    text: Version command prints CLI version, schema cohort, and Go version in human mode
    tests:
      - TestCLI_Version_HumanOutput

  - id: CLM-023
    requirement: REQ-006
    text: Version command with --json outputs JSON object with version, schema_cohort, and go_version fields
    tests:
      - TestCLI_Version_JSONOutput

  # REQ-007: commands --json
  - id: CLM-024
    requirement: REQ-007
    text: Commands --json returns a JSON array of command descriptors
    tests:
      - TestCLI_Commands_JSON_ReturnsArray

  - id: CLM-025
    requirement: REQ-007
    text: Each command descriptor includes name, path, description, and flags fields
    tests:
      - TestCLI_Commands_JSON_DescriptorFields

  - id: CLM-026
    requirement: REQ-007
    text: Command tree includes all namespace commands and their subcommands
    tests:
      - TestCLI_Commands_JSON_IncludesAllNamespaces

  # REQ-008: Help generation
  - id: CLM-027
    requirement: REQ-008
    text: Root command help lists all namespace groups and top-level commands
    tests:
      - TestCLI_Help_RootListsNamespaces

  - id: CLM-028
    requirement: REQ-008
    text: Namespace command help lists subcommands within that namespace
    tests:
      - TestCLI_Help_NamespaceListsSubcommands

  - id: CLM-029
    requirement: REQ-008
    text: Every command has a Short and Long description set
    tests:
      - TestCLI_Help_AllCommandsHaveDescriptions

  # REQ-009: NO_COLOR support
  - id: CLM-030
    requirement: REQ-009
    text: Human output omits ANSI color codes when NO_COLOR is set
    tests:
      - TestCLI_NoColor_OmitsANSI

  - id: CLM-031
    requirement: REQ-009
    text: Human output may include ANSI color codes when NO_COLOR is not set
    tests:
      - TestCLI_NoColor_AllowsANSIWhenUnset

  # REQ-010: backstop.yml typed struct
  - id: CLM-032
    requirement: REQ-010
    text: >
      Loader returns a typed Config struct with project name, language,
      runtimes, enforcement configuration (security tier, waiver_warning_days),
      pack declarations (rules and code packs with versions), and registry
      configuration (scope-based resolution)
    tests:
      - TestConfig_Struct_AllFields

  - id: CLM-033
    requirement: REQ-010
    text: Loader rejects backstop.yml with unknown top-level keys
    tests:
      - TestConfig_Struct_RejectsUnknownKeys

  - id: CLM-034
    requirement: REQ-010
    text: backstop.yml loader validates against the embedded schema
    tests:
      - TestConfig_LoaderValidatesAgainstSchema

  # REQ-011: backstop.yml schema fields
  - id: CLM-035
    requirement: REQ-011
    text: backstop.yml schema requires project and language fields
    tests:
      - TestConfig_RequiredFields_ProjectAndLanguage

  - id: CLM-036
    requirement: REQ-011
    text: backstop.yml schema accepts valid enforcement block with tier enum
    tests:
      - TestConfig_Enforcement_ValidTier

  - id: CLM-037
    requirement: REQ-011
    text: backstop.yml schema rejects invalid enforcement tier value
    tests:
      - TestConfig_Enforcement_InvalidTier

  - id: CLM-038
    requirement: REQ-011
    text: backstop.yml schema accepts valid packs block with version strings
    tests:
      - TestConfig_Packs_ValidVersions

  - id: CLM-039
    requirement: REQ-011
    text: backstop.yml schema accepts valid registries block with scope resolution
    tests:
      - TestConfig_Registries_ScopeResolution

  - id: CLM-040
    requirement: REQ-011
    text: backstop.yml schema defaults waiver_warning_days to 30 when omitted
    tests:
      - TestConfig_WaiverWarningDays_Default

  # REQ-012: Embed introspection
  - id: CLM-041
    requirement: REQ-012
    text: Embed directive uses embed.FS type for the schema filesystem
    tests:
      - TestCLI_Embed_UsesEmbedFS

  - id: CLM-042
    requirement: REQ-012
    text: Schema listing function returns all embedded schema file paths
    tests:
      - TestCLI_Embed_ListSchemaPaths

  - id: CLM-043
    requirement: REQ-012
    text: Embedded schemas match the set of schema files on disk at build time
    tests:
      - TestCLI_Embed_MatchesDiskSchemas

  # REQ-013: BACKSTOP_CONFIG env var
  - id: CLM-044
    requirement: REQ-013
    text: BACKSTOP_CONFIG overrides walk-up discovery and loads from specified path
    tests:
      - TestConfig_BackstopConfig_OverridesWalkUp

  - id: CLM-045
    requirement: REQ-013
    text: BACKSTOP_CONFIG pointing to a nonexistent file exits with code 2
    tests:
      - TestConfig_BackstopConfig_NonexistentFile_Exit2

  - id: CLM-046
    requirement: REQ-013
    text: BACKSTOP_CONFIG set to empty string is treated as unset (walk-up proceeds)
    tests:
      - TestConfig_BackstopConfig_EmptyString_FallsBackToWalkUp

  # REQ-014: Integration tests
  - id: CLM-047
    requirement: REQ-014
    text: >
      Built binary runs backstop artifact validate against a real spec file
      and returns structured JSON with violations
    tests:
      - TestIntegration_ArtifactValidate_RealSpec

  - id: CLM-048
    requirement: REQ-014
    text: >
      Built binary runs backstop code check --file against a Go file and
      returns structured JSON with semgrep results
    tests:
      - TestIntegration_CodeCheck_RealGoFile

  - id: CLM-049
    requirement: REQ-014
    text: >
      Built binary runs backstop pack compile against a real standard and
      produces enforcement manifests in .backstop/rules/
    tests:
      - TestIntegration_PackCompile_RealStandard

  - id: CLM-050
    requirement: REQ-014
    text: >
      Built binary runs backstop gate and produces a structured pass/fail
      result combining artifact validation and code check
    tests:
      - TestIntegration_Gate_EndToEnd

  - id: CLM-051
    requirement: REQ-014
    text: >
      Built binary runs backstop artifact new spec and produces a valid
      scaffolded spec file with auto-assigned ID
    tests:
      - TestIntegration_ArtifactNew_ScaffoldsSpec

contracts:
  - file: cmd/backstop/main.go
    provides:
      - name: main
        kind: function
        signature: "func main()"
        notes: "Entry point — creates root command and executes"
    consumes:
      - source: cmd/backstop/root
        name: NewRootCommand
        kind: function

  - file: cmd/backstop/root.go
    provides:
      - name: NewRootCommand
        kind: function
        signature: "func NewRootCommand() *cobra.Command"
        notes: "Builds the Cobra command tree with all namespaces and top-level commands"
    consumes:
      - source: github.com/spf13/cobra
        name: Command
        kind: type
      - source: pkg/config
        name: LoadConfig
        kind: function

  - file: cmd/backstop/embed.go
    provides:
      - name: SchemaFS
        kind: variable
        signature: "var SchemaFS embed.FS"
        notes: "Embedded filesystem containing all artifact schemas"
      - name: ListSchemas
        kind: function
        signature: "func ListSchemas() ([]string, error)"
        notes: "Returns paths of all embedded schema files for cohort introspection"
    consumes: []

  - file: pkg/config/config.go
    provides:
      - name: Config
        kind: type
        signature: "type Config struct"
        notes: "Typed representation of backstop.yml project manifest"
      - name: LoadConfig
        kind: function
        signature: "func LoadConfig() (*Config, error)"
        notes: "Discovers and loads backstop.yml via walk-up or BACKSTOP_CONFIG env var"
      - name: DiscoverConfigPath
        kind: function
        signature: "func DiscoverConfigPath() (string, error)"
        notes: "Walks up from cwd to find backstop.yml, checks BACKSTOP_CONFIG first"
    consumes: []

  - file: cmd/backstop/output.go
    provides:
      - name: Formatter
        kind: interface
        signature: "type Formatter interface { FormatResult(result interface{}) (string, error) }"
        notes: "Output formatting contract for JSON and human modes"
      - name: JSONFormatter
        kind: type
        signature: "type JSONFormatter struct"
        notes: "Implements Formatter for structured JSON output with schema_version"
      - name: HumanFormatter
        kind: type
        signature: "type HumanFormatter struct"
        notes: "Implements Formatter for human-readable terminal output, respects NO_COLOR"
    consumes:
      - source: pkg/validate
        name: ValidationResult
        kind: type

  - file: cmd/backstop/exitcode.go
    provides:
      - name: ExitPass
        kind: constant
        signature: "const ExitPass = 0"
      - name: ExitViolations
        kind: constant
        signature: "const ExitViolations = 1"
      - name: ExitConfigError
        kind: constant
        signature: "const ExitConfigError = 2"
      - name: ExitWithResult
        kind: function
        signature: "func ExitWithResult(result ValidationResult, configErr error) int"
        notes: "Determines exit code — config error (2) takes precedence over violations (1)"
    consumes:
      - source: pkg/validate
        name: ValidationResult
        kind: type
---

# SPEC-005: CLI Foundation — Command Skeleton, Embed Cohort, Config Loading, Output Layer

## Overview

The backstop CLI binary (cmd/backstop/) is the universal agent API — every agent,
runtime, and workflow interacts with backstop by shelling out to CLI commands. The
validation engine, schema infrastructure, artifact parsers, and standards compiler
exist as Go libraries in pkg/, but there is no CLI binary to expose them.

This spec covers the foundational infrastructure that every CLI command depends on:

1. **Cobra command skeleton** — root command with three namespace groups (artifact,
   code, pack) plus a top-level gate command
2. **go:embed schema cohort** — all artifact schemas embedded in the binary via
   embed.FS, eliminating runtime filesystem dependencies for schema loading
3. **backstop.yml loader** — project manifest discovery (walk-up from cwd) and
   validation, with BACKSTOP_CONFIG env var override
4. **JSON/human output layer** — --json flag for structured machine output,
   default human-readable output, identical underlying data in both modes
5. **Exit code handling** — 0 (pass), 1 (violations), 2 (config error)
6. **version command** — schema cohort identification
7. **commands --json** — agent discovery endpoint for the full command tree
8. **Help generation** — automatic via Cobra with Short/Long descriptions

Commands are thin adapters (DD-2): parse flags, call pkg/, format output, set
exit code. No business logic lives in cmd/.

## Requirements

Requirements are defined in frontmatter. Key design decisions from the bundle:

- **DD-1 (Cobra):** The command framework is Cobra, providing native subcommand
  nesting, flag parsing, help generation, and shell completion.
- **DD-2 (Thin adapters):** Commands contain no enforcement logic. All validation,
  compilation, and analysis stays in pkg/.
- **DD-3 (go:embed):** Schemas are embedded via go:embed so each CLI version is a
  self-contained schema cohort with no filesystem dependency.
- **DD-4 (Config before enforcement):** backstop.yml is loaded and validated before
  any enforcement command runs. Invalid config produces exit code 2.
- **DD-5 (Versioned JSON output):** JSON output includes a schema_version field for
  independent contract evolution.

### Command Structure

| Command | Type | Description |
|---------|------|-------------|
| `backstop` | Root | Displays help when invoked without subcommands |
| `backstop artifact` | Namespace | Artifact lifecycle commands |
| `backstop code` | Namespace | Implementation validation commands |
| `backstop pack` | Namespace | Enforcement content commands |
| `backstop gate` | Top-level | Full reconciliation kill chain |
| `backstop version` | Top-level | Schema cohort identification |
| `backstop commands` | Top-level | Agent discovery endpoint |

### Exit Codes

| Code | Meaning | Precedence |
|------|---------|------------|
| 0 | All checks pass / command succeeded | Lowest |
| 1 | Violations found | Middle |
| 2 | Configuration error | Highest — if config is invalid, no enforcement runs |

### Config Discovery Order

1. Check BACKSTOP_CONFIG env var (if set and non-empty, use that path directly)
2. Walk up from cwd checking each directory for backstop.yml
3. If neither finds a file, exit with code 2

## Implementation

The implementation creates the following files in cmd/backstop/:

### Pass 1: Embed Cohort (embed.go)

Embed all artifact schemas using go:embed with the pattern
`artifacts/*/v*/schema.json` plus `artifacts/base/schema.json`. Expose the
embedded filesystem as a package-level `embed.FS` variable. Provide a
`ListSchemas()` function that walks the embedded FS and returns all schema
file paths for cohort introspection.

### Pass 2: Config Loader (pkg/config/config.go)

Implement `DiscoverConfigPath()` which checks BACKSTOP_CONFIG env var first
(empty string treated as unset), then walks up from cwd. Implement `LoadConfig()`
which reads and validates the discovered file against its schema, returning a
typed `Config` struct. Unknown top-level keys are rejected to catch typos.
The Config type, LoadConfig, and DiscoverConfigPath live in pkg/config/ so they
are importable by other packages.

### Pass 3: Output Formatter (output.go)

Define the `Formatter` interface with `FormatResult()`. Implement `JSONFormatter`
(serializes to JSON with schema_version field) and `HumanFormatter` (formatted
text, respects NO_COLOR env var). Both formatters operate on the same underlying
data — the formatter is presentation only with zero logic.

### Pass 4: Exit Code Handler (exitcode.go)

Define constants `ExitPass` (0), `ExitViolations` (1), `ExitConfigError` (2).
Implement `ExitWithResult()` which determines the exit code with config error
taking precedence over violations.

### Pass 5: Command Skeleton (root.go, main.go)

Build the Cobra command tree: root command, three namespace commands (artifact,
code, pack), gate command, version command, commands command. Each command gets
Short and Long descriptions. The root command wires the --json persistent flag.

### Pass 6: Version Command

Implement version command that reports CLI version, schema cohort identifier
(computed from ListSchemas()), and Go runtime version. Supports --json output.

### Pass 7: Commands Discovery

Implement `commands --json` that walks the Cobra command tree and outputs a
JSON array of command descriptors with name, path, description, and flags.

## Verification

Claims are defined in frontmatter. Integration-level verification with 80%
coverage threshold across cmd/backstop/ and pkg/config/ packages.

## Sharp Edges

- **Package boundary for embed.FS.** The go:embed directive must be in a package
  at or above the artifacts/ directory relative to the module root. If cmd/backstop/
  is the package with the embed directive, the relative path to artifacts/ must be
  correct. Getting this wrong results in a compile error, not a runtime error, but
  it can be confusing. An alternative is a dedicated internal/embed package at the
  module root level.

- **Config loader vs non-enforcement commands.** Version, commands, and help do NOT
  require backstop.yml. The config loading must be conditional — only enforcement
  commands (artifact validate, code check, pack compile, gate) trigger config
  discovery. If the loader runs unconditionally, users cannot even check the CLI
  version without a backstop.yml present.

- **BACKSTOP_CONFIG path validation.** The env var accepts a file path. If set to a
  directory rather than a file, the loader must detect this and return a config error,
  not silently fail or panic. Similarly, relative paths in BACKSTOP_CONFIG should be
  resolved relative to cwd, not relative to some other anchor.

- **JSON output contract stability.** The schema_version field in JSON output
  creates a versioning contract. Once agents depend on a specific schema_version,
  changes to the JSON structure require incrementing the version and potentially
  supporting multiple versions simultaneously. The first schema_version value
  sets the baseline — choose it carefully.

- **Cobra's os.Exit behavior.** Cobra's default error handling calls os.Exit(1).
  This must be overridden to use our exit code semantics (1 = violations, 2 = config
  error). The root command should use `SilenceErrors: true` and `SilenceUsage: true`
  with manual exit code handling.

- **Unknown key rejection in backstop.yml.** Rejecting unknown keys is strict and
  could break forward compatibility. If a newer backstop.yml feature is used with
  an older CLI, the older CLI will reject the file. This is intentional (fail fast
  on version mismatch) but must be documented clearly in the error message.

- **NO_COLOR interaction with --json.** When both --json and NO_COLOR are set, the
  --json flag takes precedence — JSON output never contains ANSI codes regardless
  of NO_COLOR. The NO_COLOR check only matters for the human formatter.

- **Thin adapter enforcement is a code review concern.** cmd/ files must be thin
  adapters — flag parsing, pkg/ delegation, output formatting. Whether enforcement
  logic has leaked into cmd/ cannot be meaningfully asserted by a unit test ("no
  business logic in cmd/" is not a mechanically testable property). This is enforced
  through code review, not automated claims.

## Review Questions

1. Does the embed directive path correctly resolve from cmd/backstop/ to
   ../../artifacts/ at build time, or should the embed live in a higher-level package?

2. When backstop.yml walk-up discovery reaches the filesystem root without finding
   the file, is the error message actionable enough for a user to understand what
   happened and how to fix it?

3. If a command requires config (enforcement command) and config loading fails,
   does the CLI print both the config error AND usage/help, or only the error?
   Printing help on config errors could bury the actual problem.

4. For commands --json, does the command descriptor schema need its own versioning,
   or does the top-level schema_version in the JSON response cover it?

5. Can the schema cohort identifier be computed cheaply enough to include in every
   JSON response, or should it only appear in version output?

6. If BACKSTOP_CONFIG points to a valid YAML file that is not a backstop.yml
   (e.g. a docker-compose.yml), does the loader produce a clear error distinguishing
   "wrong file type" from "invalid backstop.yml"?

## References

- Bundle: cli (CLI Foundation spec seed 1)
- Bundle requirements: REQ-007 (output parity), REQ-008 (go:embed), REQ-009 (backstop.yml), REQ-011 (commands/version)
- Bundle design decisions: DD-1 (Cobra), DD-2 (thin adapters), DD-3 (go:embed), DD-4 (config before enforcement), DD-5 (versioned JSON)
- Resolved OQs: OQ-1 (Cobra), OQ-2 (schema_version field), OQ-12 (walk-up + BACKSTOP_CONFIG)
- ADR-0004: Structured output contract
- ADR-0005: Project manifest (backstop.yml)
- ADR-0008: CLI as agent API
- D-028: go:embed schema cohort
- D-069: CLI as universal agent API
- D-070: Schema evolution rules
