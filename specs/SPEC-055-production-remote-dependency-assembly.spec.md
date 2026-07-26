---
title: "Production Remote Dependency Assembly"
number: SPEC-055
created: "2026-07-26"
status: implemented
schema_version: spec/v1
spec_version: 1.3.1

implementation:
  summary: >
    The FOUNDATION reliability seed of BUNDLE-006 (pack-distribution-lifecycle):
    the production dependencies every remote lifecycle command advertises but does
    not have, plus the constructor shape (DD-30 / OQ-8 option (c)) that makes an
    incompletely assembled command a COMPILE-TIME failure rather than a runtime
    panic. Today `cmd/backstop/pack_add.go` builds `distribution.AddOptions` with a
    nil `GitCloner` and a nil `Validator`; `distribution.Add` dereferences the
    cloner at `add.go:153` and terminates with a raw SIGSEGV (ISSUE-073), while the
    nil validator silently SKIPS `pack check` + `pack test` behind an
    `if opts.Validator != nil` guard. `install.go:165` nil-guards instead of
    dereferencing, so it does not panic — it exits 1 with NO output at all, which
    is worse to act on. Six things land here. (1) Three concrete production
    implementations that do not exist anywhere in the tree: `ExecGitCloner`
    (real `git clone` / `git ls-remote`), `PackvalValidator` (the same
    `pkg/packval` pipeline the `pack check` / `pack test` commands run), and
    `TagVersionResolver` (semver resolution over real remote tags).
    `pkg/scaffold/git_executor_real.go` RealGitExecutor is NOT reusable: its
    `ListTags(pattern string)` runs `git tag -l` against the LOCAL repository,
    which is a different operation from `GitCloner.ListTags(url string)`, and it
    has no `Clone` at all. (2) Four command TYPES with positional constructors —
    `NewAddCommand`, `NewInstallCommand`, `NewUpdateCommand`, `NewUpgradeCommand` —
    each taking exactly the dependencies its command requires; the dependency
    fields are REMOVED from the options structs and the package-level `Add` /
    `Install` / `Update` / `Upgrade` free functions are DELETED, because leaving
    either one keeps the nil hole open. (3) Cobra wiring in a single production
    assembly file so `pack add` / `install` / `update` / `upgrade` run real
    dependencies, and validation actually runs on remote adds. (4) The REDESIGN of
    the two NON-TEST helpers that currently depend on the nil-validator-skip
    contract those deletions remove — `distribution.InstallContractsLocalPack` /
    `InstallContractsLocalPackWithValidator` and package-main's
    `installSubstantivenessLocalPack` — plus the named, bounded, and LARGE set of
    test call sites the deletions break, dominated by the distribution package's
    own six external suites (139 tests, 138 free-function call sites, 193
    dependency-field lines), whose 62 SPEC-015-mandated test names must survive
    verbatim. (5) The CROSS-CUTTING error-surfacing repair the
    bundle assigns to this seed: `cmd/backstop/main.go` suppresses the message for
    every `ExitViolations` error, which is correct for `gate` / `pack check` (they
    already printed findings) and silently discards the diagnostic for the nine
    commands that use exit 1 to mean "this failed" — `pack relock`,
    `pack install`, and `recipe apply` are the three observed victims across three
    subsystems. The default inverts to LOUD, with an explicit `Explained` opt-out
    on the four sites that legitimately already printed. (6) The two test
    MECHANISMS the error-surfacing and JSON claims require in order to be
    assertable at all: an extracted `reportError` seam (because `main()` calls
    `os.Exit` and is untestable) and a stream-separated CLI runner (because the
    existing `runBackstop` helper uses `CombinedOutput`, which merges stdout and
    stderr and would let every stream assertion pass vacuously). OUT OF SCOPE
    (later BUNDLE-006 seeds): authored-content hashing as a general copy/hash
    boundary across all sources (REQ-021@1.1.0 — the Clone strip below solves the
    remote source only), the legacy-hash migration operation and `pack relock`'s
    argument shape (REQ-041 / ISSUE-074), source-coordinate-versus-manifest
    identity (REQ-039), the shared staged-filesystem transaction coordinator
    (REQ-040), and the violation scanner and remediation bundle generator
    themselves (REQ-014 / REQ-018). This spec drives ONE real `git clone` end to
    end through the built binary and lands the hermetic harness substrate the
    parity suite (REQ-042) extends. Per the Clone-strip amendment it DOES verify a
    green remote add → install round trip with matching content hashes, because
    `Clone` strips the root `.git` it created before handing the tree to
    distribution; that does NOT deliver REQ-021@1.1.0, which owns the copy/hash
    boundary for ALL sources plus the legacy-lock migration, so local-path packs
    and pre-existing contaminated locks remain the identity/migration seed's.
  subject: pkg/pack/distribution

verification:
  level: integration
  test_command: go test ./pkg/pack/distribution/... ./cmd/backstop/... -race -coverprofile=cover.out
  coverage_threshold: 80

requirements:
  - id: REQ-001
    text: >
      `pkg/pack/distribution` must provide `ExecGitCloner`, a concrete
      implementation of the existing `GitCloner` interface that executes the real
      `git` binary as a subprocess. `Clone(url, version, destDir)` must fetch the
      exact requested tag (a shallow, single-ref clone) into `destDir` and then
      STRIP the root `.git` directory before returning, so the tree it hands
      distribution is AUTHORED CONTENT ONLY. The strip is the cloner's own cleanup:
      `.git` is repository-control metadata the clone itself created, not content
      that came from the pack author, so removing it makes `Clone`'s contract
      "materialize the authored tree at this ref" rather than "leave a git working
      copy behind". A `Clone` that returns while a root `.git` remains has not
      satisfied this requirement. `ListTags(url)` must list the tags of a REMOTE
      repository and return bare tag names with any peeled `^{}` suffix entries
      excluded. Both must run git
      NON-INTERACTIVELY (a credential or host-key prompt must never be able to
      block the process), and both must INHERIT the ambient git configuration
      environment rather than clearing it, so already-configured credential
      helpers keep working and a `url.<base>.insteadOf` rewrite can redirect the
      production URL to a local repository. Before invoking git, a URL or ref
      beginning with `-` must be rejected, and the URL and ref must be passed
      after a `--` separator so neither can be interpreted as a git option.
      Constructing the cloner must NOT probe for the git binary, so an airgapped
      `pack install --cache` still assembles.
    supports: pack-distribution-lifecycle:REQ-038@1.1.0
    follows: STD-GO-001:GO-010
  - id: REQ-002
    text: >
      Every `ExecGitCloner` failure must return a typed `*GitError` carrying a
      human-readable diagnostic that names the operation, the URL, the requested
      ref where one applies, and the captured git stderr. A missing tag, an
      unreachable repository, a git binary absent from PATH, and an invocation
      that exceeds the cloner's configured timeout must each produce a distinct
      readable message. No path may panic, return a bare exit status with no
      explanation, or report an empty tag list where the underlying git command
      failed.
    supports: pack-distribution-lifecycle:REQ-001@1.0.0
    follows: STD-GO-001:GO-011
  - id: REQ-003
    text: >
      `pkg/pack/distribution` must provide `PackvalValidator`, a concrete
      implementation of the existing `Validator` interface that runs the SAME
      `pkg/packval` pipeline the `backstop pack check` and `backstop pack test`
      commands run — `RunPackCheck` in the pipeline's `check` mode and
      `RunPackTest` in its `test` mode — so distribution applies exactly the
      validation authority those commands apply, with no second implementation of
      pack validation. A failing pipeline must return a `*ValidationError` naming
      the pack directory, the failing phase, and the validation error count.
    supports: pack-distribution-lifecycle:REQ-002@1.0.0
    follows: STD-GO-001:GO-010
  - id: REQ-004
    text: >
      `pkg/pack/distribution` must provide `TagVersionResolver`, a concrete
      implementation of the existing `VersionResolver` interface built over a
      `GitCloner`'s remote tag listing. `ResolveLatestCompatible` must consider
      only tags matching strict `X.Y.Z` after normalizing a single optional
      leading `v` (prerelease, build-metadata, and arbitrary tags are ignored),
      must select the highest such version sharing the CURRENT version's MAJOR
      component, and must return the current version unchanged when no newer
      compatible tag exists. The same-major rule applies LITERALLY at major zero:
      with a current version of `0.1.0` and a `v0.2.0` tag available, `0.2.0` is
      compatible and resolves. `IsMajorBump` must report whether two versions
      differ in their major component. A tag-listing failure must propagate as an
      error and must never be reported as "already at the latest version".
    supports: pack-distribution-lifecycle:REQ-030@1.0.0
    follows: STD-GO-001:GO-010
  - id: REQ-005
    text: >
      A lifecycle command's required dependencies must be structurally
      impossible to omit. `AddOptions`, `InstallOptions`, `UpdateOptions`, and
      `UpgradeOptions` must carry NO dependency-typed field — no `GitCloner`, no
      `Validator`, no `VersionResolver`, no `Scanner`, no `RemediationGenerator`
      — retaining only their non-dependency knobs, so a caller cannot express an
      options value that omits a dependency. Distribution must provide no
      internal default for any dependency: no constructor, command path, or
      package-level helper may substitute a fallback implementation when one is
      not supplied, so a test double is never mistakable for production wiring.
    supports: pack-distribution-lifecycle:REQ-038@1.1.0
    follows: STD-GO-001:GO-010
  - id: REQ-006
    text: >
      Each lifecycle command must be a distinct type constructed by a positional
      constructor that takes exactly the dependencies that command requires, so
      that omitting one is a compile-time failure: `NewAddCommand(GitCloner,
      Validator)`; `NewInstallCommand(GitCloner)`; `NewUpdateCommand(GitCloner,
      Validator, VersionResolver)`; `NewUpgradeCommand(GitCloner, Validator,
      Scanner, RemediationGenerator)`; and `NewTagVersionResolver(GitCloner)`.
      Install takes NO validator, because `pack install` is the hash-verified
      restore path and does not re-validate. Because an explicitly written `nil`
      remains expressible, each constructor must additionally FAIL CLOSED: a nil
      argument must return a typed `*MissingDependencyError` naming the command
      and the missing dependency, never a partially assembled command, never a
      panic, and never a deferred nil dereference at execution time.
    supports: pack-distribution-lifecycle:REQ-038@1.1.0
    follows: STD-GO-001:GO-010
  - id: REQ-007
    text: >
      `pkg/pack/distribution` must expose no package-level `Add`, `Install`,
      `Update`, or `Upgrade` entry point. Those free functions must be removed and
      their pipelines must become methods on the command types from REQ-006, since
      a surviving free function taking an options value would remain callable
      without any dependency and would reinstate exactly the hole REQ-005 and
      REQ-006 close.
    supports: pack-distribution-lifecycle:REQ-038@1.1.0
    follows: STD-GO-001:GO-010
  - id: REQ-008
    text: >
      Validation must actually run where the bundle says it runs. `AddCommand` and
      `UpgradeCommand` must invoke `RunPackCheck` and then `RunPackTest` on the
      resolved pack directory UNCONDITIONALLY before any consumer state is
      mutated; the `if opts.Validator != nil` guards that let a nil validator skip
      validation silently must be gone. A failure of either step must abort the
      command with the validation diagnostic and leave no installed content,
      manifest entry, lock entry, or provenance entry behind. `InstallCommand`
      must NOT validate — it verifies content hashes only. This single
      unconditional-validation change is what delivers the validate-before-install
      obligation for BOTH pack add and pack upgrade.
    supports:
      - pack-distribution-lifecycle:REQ-002@1.0.0
      - pack-distribution-lifecycle:REQ-015@1.0.0
    follows: STD-GO-001:GO-010
  - id: REQ-009
    text: >
      `UpgradeCommand` must invoke its `Scanner` and, whenever the scan reports
      violations, its `RemediationGenerator` UNCONDITIONALLY; the
      `if opts.Scanner != nil` / `if len(violations) > 0 && opts.RemediationGenerator != nil`
      guards that currently let a missing capability produce a successful upgrade
      reporting zero baselined violations must be gone. The violation scanner and
      the remediation bundle generator THEMSELVES are NOT built by this spec —
      they are bundle REQ-014 and REQ-018 and remain undelivered, so this spec
      does not pin them — meaning production must wire explicit implementations
      returning a typed `*CapabilityUnavailableError` that names the capability
      and the requirement tracking it, and the upgrade must run its violation scan
      BEFORE it mutates any consumer state (tool config, provenance, installed
      content, backstop.yml, backstop.lock), so an unavailable capability fails
      loud without leaving the consumer partially mutated. What this requirement
      delivers is the ASSEMBLY and fail-loud propagation of those dependencies,
      which is REQ-038's obligation for upgrade, not the capabilities themselves.
    supports: pack-distribution-lifecycle:REQ-038@1.1.0
    follows: STD-GO-001:GO-010
  - id: REQ-010
    text: >
      `cmd/backstop` must assemble the production dependencies for `pack add`,
      `pack install`, `pack update`, and `pack upgrade` in ONE production assembly
      file that constructs `ExecGitCloner`, `PackvalValidator`, and
      `TagVersionResolver` and returns fully-assembled command values, replacing
      the nil-dependency `AddOptions` / `InstallOptions` / `UpdateOptions` /
      `UpgradeOptions` literals in `pack_add.go`, `pack_install.go`,
      `pack_update.go`, and `pack_upgrade.go`. An assembly failure must surface as
      a configuration error with exit code 2, not as a violation.
    supports: pack-distribution-lifecycle:REQ-008@1.0.0
    follows: STD-GO-001:GO-010
  - id: REQ-011
    text: >
      A non-zero exit must always carry a diagnostic the consumer can read. The
      CLI's error surfacing must default to LOUD: `ExitCodeError` gains an
      `Explained` field, and the CLI's error-reporting seam must print the message
      to stderr for every `ExitCodeError` with a non-empty message UNLESS
      `Explained` is set, replacing the blanket suppression of `ExitViolations`
      messages. `Explained` may be set ONLY at the four sites whose command has
      already printed structured findings before returning — `gate` (and only for
      its `ExitViolations` verdict, so its exit-2 message keeps printing),
      `pack check`, `pack test`, and `artifact validate`. Every other
      `ExitViolations` site — `pack add`, `pack install`, `pack update`,
      `pack upgrade`, `pack remove`, `pack list`, `pack relock`, `recipe apply`,
      and `artifact new` — must leave `Explained` unset and therefore print. The
      opt-out direction is deliberate: a future command that forgets to declare
      itself explained produces a duplicated diagnostic, never a silent failure.
    supports: pack-distribution-lifecycle:REQ-038@1.1.0
    follows: STD-GO-001:GO-011
  - id: REQ-012
    text: >
      When the global `--json` flag is set, a failing `pack add`, `pack install`,
      `pack update`, or `pack upgrade` must write ONE parseable structured JSON
      error object to stdout carrying the command path, an error kind, and the
      human message, and must still exit non-zero. The kind must be derived from
      distribution's typed errors — `git` for `*GitError`, `validation` for
      `*ValidationError`, `dependency` for `*MissingDependencyError`,
      `capability` for `*CapabilityUnavailableError`, and a default kind
      otherwise. Emitting the JSON object counts as explaining the failure, so the
      human stderr line must be suppressed in that mode and stdout must hold
      exactly one JSON document.
    supports: pack-distribution-lifecycle:REQ-038@1.1.0
    follows: STD-GO-001:GO-011
  - id: REQ-013
    text: >
      The two NON-TEST helpers that today call the package-level `Add` and depend
      on the nil-validator-skip contract must be redesigned, not left to the
      implementer. In `pkg/pack/distribution/contracts_local_install.go`,
      `InstallContractsLocalPack` and `InstallContractsLocalPackWithValidator`
      must collapse into a single exported helper
      `InstallContractsLocalPack(add *AddCommand, repoRoot, projectDir string)`
      that RECEIVES an assembled command rather than assembling one, because a
      library-layer helper that constructed its own `PackvalValidator` would be
      exactly the internal defaulting REQ-005 forbids; the validator-aware variant
      and its "a nil validator skips those phases" doc contract must be deleted,
      since under REQ-008 validation is unconditional and the distinction no
      longer exists. In `cmd/backstop/gate_substantiveness_e2e.go`,
      the METHOD `func (w *e2eWorkspace) installSubstantivenessLocalPack(repoRoot
      string) error` must keep its signature and obtain its command from the
      production assembly helper, which is legitimate there because `cmd/backstop`
      IS the assembly layer. Both helpers must run validation unconditionally, and
      the `packs/contracts` and `packs/substantiveness` sources must pass
      `pack check` + `pack test` through that path. Separately, the distribution
      package's OWN external test suites are the largest consumers of the deleted
      API and must be migrated wholesale to constructor-assembled commands,
      preserving every SPEC-015-mandated test name verbatim.
    supports: pack-distribution-lifecycle:REQ-038@1.1.0
    follows: STD-GO-001:GO-010
  - id: REQ-014
    text: >
      The error-surfacing and JSON-error behavior must be ASSERTABLE, which it is
      not through the existing harness. `main()` calls `os.Exit` and cannot be
      tested, so its error-reporting decision must be extracted into
      `reportError(w io.Writer, err error) int`, which writes the human
      diagnostic to the supplied writer unless the error is an `ExitCodeError`
      with `Explained` set or an empty message, and returns the exit code
      (`ExitConfigError` for an untyped error). `main()` must become a call to it.
      Separately, the existing `runBackstop` CLI test helper returns
      `CombinedOutput`, which MERGES stdout and stderr — every "prints to stderr"
      and every "stdout holds exactly one JSON document" assertion would pass
      vacuously against it — so a stream-separated runner returning stdout and
      stderr independently alongside the exit code must be added and used by the
      claims that assert on a specific stream. No claim under REQ-011 or REQ-012
      may be driven by a merged-stream helper.
    supports: pack-distribution-lifecycle:REQ-038@1.1.0
    follows: STD-GO-001:GO-011

claims:
  # REQ-001 — production Git cloner over the real git binary
  - id: CLM-001
    requirement: REQ-001
    text: Clone materializes a pack tree from a real local git repository at the requested tag, driving the real git binary
    tests:
      - TestExecGitCloner_Clone_MaterializesTaggedRepo
  - id: CLM-002
    requirement: REQ-001
    text: Clone checks out the requested tag and not HEAD when the repository has later commits past that tag
    tests:
      - TestExecGitCloner_Clone_ChecksOutRequestedTagNotHead
  - id: CLM-003
    requirement: REQ-001
    text: ListTags returns bare tag names from a real repository and excludes peeled "^{}" ref entries
    tests:
      - TestExecGitCloner_ListTags_ReturnsBareTagNamesWithoutPeeledRefs
  - id: CLM-004
    requirement: REQ-001
    text: Clone runs git non-interactively so a credential prompt cannot block the process
    tests:
      - TestExecGitCloner_Clone_RunsGitNonInteractively
  - id: CLM-005
    requirement: REQ-001
    text: Clone inherits the ambient git configuration environment, so a url.insteadOf rewrite redirects the production URL to a local repository
    tests:
      - TestExecGitCloner_Clone_HonorsAmbientGitConfigRedirect
  - id: CLM-006
    requirement: REQ-001
    text: Clone rejects a URL beginning with "-" before invoking git
    tests:
      - TestExecGitCloner_Clone_RejectsOptionLikeURL
  - id: CLM-007
    requirement: REQ-001
    text: Clone rejects a ref beginning with "-" before invoking git
    tests:
      - TestExecGitCloner_Clone_RejectsOptionLikeRef
  - id: CLM-008
    requirement: REQ-001
    text: ListTags rejects a URL beginning with "-" before invoking git
    tests:
      - TestExecGitCloner_ListTags_RejectsOptionLikeURL
  - id: CLM-009
    requirement: REQ-001
    text: Constructing the cloner does not probe for the git binary, so assembly succeeds where git is absent
    tests:
      - TestNewExecGitCloner_DoesNotProbeForGitBinary
  - id: CLM-101
    requirement: REQ-001
    text: Clone returns a tree with no root .git directory, driven against a real tagged git repository whose clone would otherwise carry one
    tests:
      - TestExecGitCloner_Clone_StripsRootGitDirectory
  - id: CLM-102
    requirement: REQ-001
    text: Two separate clones of the same tag produce byte-identical content hashes, because the .git that would differ between them is stripped
    tests:
      - TestExecGitCloner_Clone_RepeatedClonesHashIdentically

  # REQ-002 — typed, readable git diagnostics on every failure path
  - id: CLM-010
    requirement: REQ-002
    text: Cloning a tag that does not exist returns a GitError naming the URL and the requested tag
    tests:
      - TestExecGitCloner_Clone_MissingTagReturnsTypedDiagnostic
  - id: CLM-011
    requirement: REQ-002
    text: Cloning an unreachable repository returns a GitError carrying the captured git stderr
    tests:
      - TestExecGitCloner_Clone_UnreachableURLCarriesStderr
  - id: CLM-012
    requirement: REQ-002
    text: A git binary absent from PATH returns a GitError naming the executable rather than a bare exec failure
    tests:
      - TestExecGitCloner_Clone_MissingGitBinaryReturnsTypedDiagnostic
  - id: CLM-013
    requirement: REQ-002
    text: A git invocation exceeding the configured timeout returns a GitError naming the timeout instead of hanging
    tests:
      - TestExecGitCloner_Clone_TimeoutReturnsTypedDiagnostic
  - id: CLM-014
    requirement: REQ-002
    text: ListTags against an unreachable URL returns an error rather than an empty tag list
    tests:
      - TestExecGitCloner_ListTags_UnreachableURLErrorsRatherThanEmpty

  # REQ-003 — production validator running the same packval pipeline as the CLI
  - id: CLM-015
    requirement: REQ-003
    text: RunPackCheck passes for a pack that the pack check pipeline passes
    tests:
      - TestPackvalValidator_RunPackCheck_PassesValidPack
  - id: CLM-016
    requirement: REQ-003
    text: RunPackCheck returns a ValidationError naming the failing phase for a structurally invalid pack
    tests:
      - TestPackvalValidator_RunPackCheck_InvalidPackReturnsValidationError
  - id: CLM-017
    requirement: REQ-003
    text: RunPackTest passes for a pack that the pack test pipeline passes
    tests:
      - TestPackvalValidator_RunPackTest_PassesValidPack
  - id: CLM-018
    requirement: REQ-003
    text: RunPackTest returns a ValidationError for a pack whose fixture phase fails
    tests:
      - TestPackvalValidator_RunPackTest_FixtureFailureReturnsValidationError
  - id: CLM-019
    requirement: REQ-003
    text: The validator's verdict matches the pack check and pack test commands' verdict on the same pack directory
    subject: cmd/backstop
    tests:
      - TestPackvalValidator_MatchesPackCheckAndPackTestCommandVerdicts

  # REQ-004 — semver resolution over real remote tags
  - id: CLM-020
    requirement: REQ-004
    text: ResolveLatestCompatible selects the highest tag sharing the current version's major component
    tests:
      - TestTagVersionResolver_ResolvesHighestSameMajor
  - id: CLM-021
    requirement: REQ-004
    text: ResolveLatestCompatible never crosses a major boundary when a higher-major tag is present
    tests:
      - TestTagVersionResolver_DoesNotCrossMajorBoundary
  - id: CLM-022
    requirement: REQ-004
    text: The same-major rule applies literally at major zero — 0.1.0 resolves to an available 0.2.0
    tests:
      - TestTagVersionResolver_ZeroMajorMinorIsCompatible
  - id: CLM-023
    requirement: REQ-004
    text: ResolveLatestCompatible normalizes a single optional leading "v" on tags
    tests:
      - TestTagVersionResolver_NormalizesOptionalVPrefix
  - id: CLM-024
    requirement: REQ-004
    text: ResolveLatestCompatible ignores prerelease, build-metadata, and arbitrary non-strict-semver tags
    tests:
      - TestTagVersionResolver_IgnoresNonStrictSemverTags
  - id: CLM-025
    requirement: REQ-004
    text: ResolveLatestCompatible returns the current version unchanged when no newer compatible tag exists
    tests:
      - TestTagVersionResolver_NoNewerCompatibleReturnsCurrent
  - id: CLM-026
    requirement: REQ-004
    text: A tag-listing failure propagates as an error and is never reported as already-at-latest
    tests:
      - TestTagVersionResolver_ListTagsFailurePropagatesAsError
  - id: CLM-027
    requirement: REQ-004
    text: IsMajorBump reports true for versions differing in their major component
    tests:
      - TestTagVersionResolver_IsMajorBump_TrueAcrossMajors
  - id: CLM-028
    requirement: REQ-004
    text: IsMajorBump reports false for versions sharing a major component
    tests:
      - TestTagVersionResolver_IsMajorBump_FalseWithinMajor
  - id: CLM-029
    requirement: REQ-004
    text: ResolveLatestCompatible drives the real remote tag listing against a hermetic tagged repository
    tests:
      - TestTagVersionResolver_ResolvesAgainstRealTaggedRepository

  # REQ-005 — options carry no dependencies and distribution supplies no defaults
  - id: CLM-030
    requirement: REQ-005
    text: AddOptions, InstallOptions, UpdateOptions, and UpgradeOptions declare no dependency-typed field
    kind: absence
    tests:
      - TestDistribution_OptionsStructsCarryNoDependencyFields
  - id: CLM-031
    requirement: REQ-005
    text: No constructor, command path, or package-level helper substitutes a default GitCloner, Validator, VersionResolver, Scanner, or RemediationGenerator
    kind: absence
    tests:
      - TestDistribution_NoInternalDependencyDefaults

  # REQ-006 — the per-command dependency matrix, every cell
  - id: CLM-032
    requirement: REQ-006
    text: NewAddCommand with a real cloner and a real validator returns an assembled command
    tests:
      - TestNewAddCommand_CompleteAssemblySucceeds
  - id: CLM-033
    requirement: REQ-006
    text: NewAddCommand with a nil cloner returns a MissingDependencyError naming the git cloner
    tests:
      - TestNewAddCommand_NilGitClonerNamesDependency
  - id: CLM-034
    requirement: REQ-006
    text: NewAddCommand with a nil validator returns a MissingDependencyError naming the validator
    tests:
      - TestNewAddCommand_NilValidatorNamesDependency
  - id: CLM-035
    requirement: REQ-006
    text: NewAddCommand with every dependency nil returns an error and never a partially assembled command
    tests:
      - TestNewAddCommand_AllDependenciesNilReturnsError
  - id: CLM-036
    requirement: REQ-006
    text: NewInstallCommand with a real cloner returns an assembled command
    tests:
      - TestNewInstallCommand_CompleteAssemblySucceeds
  - id: CLM-037
    requirement: REQ-006
    text: NewInstallCommand with a nil cloner returns a MissingDependencyError naming the git cloner
    tests:
      - TestNewInstallCommand_NilGitClonerNamesDependency
  - id: CLM-038
    requirement: REQ-006
    text: NewUpdateCommand with a real cloner, validator, and resolver returns an assembled command
    tests:
      - TestNewUpdateCommand_CompleteAssemblySucceeds
  - id: CLM-039
    requirement: REQ-006
    text: NewUpdateCommand with a nil cloner returns a MissingDependencyError naming the git cloner
    tests:
      - TestNewUpdateCommand_NilGitClonerNamesDependency
  - id: CLM-040
    requirement: REQ-006
    text: NewUpdateCommand with a nil validator returns a MissingDependencyError naming the validator
    tests:
      - TestNewUpdateCommand_NilValidatorNamesDependency
  - id: CLM-041
    requirement: REQ-006
    text: NewUpdateCommand with a nil version resolver returns a MissingDependencyError naming the version resolver
    tests:
      - TestNewUpdateCommand_NilVersionResolverNamesDependency
  - id: CLM-042
    requirement: REQ-006
    text: NewUpgradeCommand with a real cloner, validator, scanner, and remediation generator returns an assembled command
    tests:
      - TestNewUpgradeCommand_CompleteAssemblySucceeds
  - id: CLM-043
    requirement: REQ-006
    text: NewUpgradeCommand with a nil cloner returns a MissingDependencyError naming the git cloner
    tests:
      - TestNewUpgradeCommand_NilGitClonerNamesDependency
  - id: CLM-044
    requirement: REQ-006
    text: NewUpgradeCommand with a nil validator returns a MissingDependencyError naming the validator
    tests:
      - TestNewUpgradeCommand_NilValidatorNamesDependency
  - id: CLM-045
    requirement: REQ-006
    text: NewUpgradeCommand with a nil scanner returns a MissingDependencyError naming the scanner
    tests:
      - TestNewUpgradeCommand_NilScannerNamesDependency
  - id: CLM-046
    requirement: REQ-006
    text: NewUpgradeCommand with a nil remediation generator returns a MissingDependencyError naming the remediation generator
    tests:
      - TestNewUpgradeCommand_NilRemediationGeneratorNamesDependency
  - id: CLM-047
    requirement: REQ-006
    text: NewTagVersionResolver with a nil cloner returns a MissingDependencyError naming the git cloner
    tests:
      - TestNewTagVersionResolver_NilGitClonerNamesDependency
  - id: CLM-048
    requirement: REQ-006
    text: An install command is constructible without any validator, since install does not re-validate
    tests:
      - TestNewInstallCommand_RequiresNoValidator
  - id: CLM-090
    requirement: REQ-006
    text: NewTagVersionResolver with a real cloner returns an assembled resolver
    tests:
      - TestNewTagVersionResolver_CompleteAssemblySucceeds

  # REQ-007 — no free-function bypass survives
  - id: CLM-049
    requirement: REQ-007
    text: The distribution package declares no package-level Add, Install, Update, or Upgrade entry point
    kind: absence
    tests:
      - TestDistribution_NoPackageLevelCommandEntryPoints

  # REQ-008 — validation actually runs where the bundle says it runs
  - id: CLM-050
    requirement: REQ-008
    text: AddCommand runs pack check and then pack test before copying any content into the consumer project
    tests:
      - TestAddCommand_RunsCheckThenTestBeforeInstall
  - id: CLM-051
    requirement: REQ-008
    text: A pack check failure aborts AddCommand with the validation diagnostic and leaves no installed content, manifest, lock, or provenance entry
    tests:
      - TestAddCommand_CheckFailureAbortsWithoutMutation
  - id: CLM-052
    requirement: REQ-008
    text: A pack test failure aborts AddCommand with the validation diagnostic and leaves no installed content, manifest, lock, or provenance entry
    tests:
      - TestAddCommand_TestFailureAbortsWithoutMutation
  - id: CLM-053
    requirement: REQ-008
    text: UpgradeCommand runs pack check and pack test before installing the new version
    tests:
      - TestUpgradeCommand_ValidatesBeforeInstall
  - id: CLM-054
    requirement: REQ-008
    text: InstallCommand restores a pack whose content would fail validation, proving install verifies hashes and does not validate
    tests:
      - TestInstallCommand_DoesNotValidate
  - id: CLM-055
    requirement: REQ-008
    text: A remote pack add whose pack fails validation exits non-zero through the built CLI with the validation diagnostic
    subject: cmd/backstop
    tests:
      - TestE2E_PackAdd_ValidationFailureIsLoud

  # REQ-009 — upgrade's scan and remediation capabilities are explicit, never skipped
  - id: CLM-056
    requirement: REQ-009
    text: UpgradeCommand invokes its scanner unconditionally, including when the scan reports zero violations
    tests:
      - TestUpgradeCommand_ScannerInvokedUnconditionally
  - id: CLM-057
    requirement: REQ-009
    text: UpgradeCommand invokes its remediation generator whenever the scan reports violations, and a generation failure aborts the upgrade
    tests:
      - TestUpgradeCommand_RemediationFailureAbortsUpgrade
  - id: CLM-058
    requirement: REQ-009
    text: UpgradeCommand propagates a scanner's CapabilityUnavailableError unchanged, driven by a test-declared double in the distribution package
    tests:
      - TestUpgradeCommand_PropagatesCapabilityUnavailableError
  - id: CLM-059
    requirement: REQ-009
    text: An unavailable capability fails before any consumer state is mutated — tool config, provenance, installed content, backstop.yml, and backstop.lock are unchanged
    tests:
      - TestUpgradeCommand_UnavailableCapabilityFailsBeforeMutation
  - id: CLM-091
    requirement: REQ-009
    text: The production unavailable scanner and remediation generator each return a CapabilityUnavailableError naming the capability and its tracking requirement
    subject: cmd/backstop
    tests:
      - TestProductionUpgradeCapabilities_NameCapabilityAndReference
  - id: CLM-060
    requirement: REQ-009
    text: pack upgrade through the built CLI reports the unavailable remediation capability instead of a successful upgrade with zero baselined violations
    subject: cmd/backstop
    tests:
      - TestE2E_PackUpgrade_UnavailableCapabilityIsLoud

  # REQ-010 — production Cobra wiring
  - id: CLM-061
    requirement: REQ-010
    text: The production assembly helpers return fully-assembled add, install, update, and upgrade commands with no error
    subject: cmd/backstop
    tests:
      - TestProductionPackCommands_AssembleCompletely
  - id: CLM-062
    requirement: REQ-010
    text: pack add of a hermetic tagged repository installs and locks the pack through the built CLI using a real git clone
    subject: cmd/backstop
    tests:
      - TestE2E_PackAdd_RemoteTaggedRepositoryInstallsAndLocks
  - id: CLM-063
    requirement: REQ-010
    text: pack add of a tag that does not exist exits non-zero with a diagnostic and no stack trace, closing the ISSUE-073 panic
    subject: cmd/backstop
    tests:
      - TestE2E_PackAdd_MissingTagDiagnosticNotPanic
  - id: CLM-064
    requirement: REQ-010
    text: pack install --cache restores a locked pack through the built CLI
    subject: cmd/backstop
    tests:
      - TestE2E_PackInstall_CacheRestore
  - id: CLM-065
    requirement: REQ-010
    text: A remote pack add followed by pack install from the committed lock on a FRESH clone yields a matching content hash — the round trip verifies hash equality end to end through the built CLI
    subject: cmd/backstop
    tests:
      - TestE2E_PackAddThenInstall_RoundTripHashesMatch
  - id: CLM-103
    requirement: REQ-010
    text: pack install against a git-sourced lock entry whose recorded hash does not match the cloned content fails loudly naming both hashes, so the round-trip equality claim cannot pass by verification having been removed
    subject: cmd/backstop
    tests:
      - TestE2E_PackInstall_GitSourceHashMismatchIsLoud
  - id: CLM-066
    requirement: REQ-010
    text: pack update resolves a newer compatible tag through the built CLI against a hermetic tagged repository
    subject: cmd/backstop
    tests:
      - TestE2E_PackUpdate_ResolvesNewerCompatibleTag
  - id: CLM-067
    requirement: REQ-010
    text: No pack lifecycle command constructs a distribution options value carrying a dependency field
    subject: cmd/backstop
    kind: absence
    tests:
      - TestPackCommands_ConstructNoDependencyCarryingOptions

  # REQ-011 — error surfacing: every silent exit-1 site becomes loud, every explained site stays quiet
  - id: CLM-068
    requirement: REQ-011
    text: A failing pack add prints its diagnostic to stderr and exits 1, asserted on the separated stderr stream of the built CLI
    subject: cmd/backstop
    tests:
      - TestExitSurfacing_PackAdd_PrintsDiagnostic
  - id: CLM-069
    requirement: REQ-011
    text: A failing pack install prints its diagnostic to stderr and exits 1, asserted through the reportError seam
    subject: cmd/backstop
    tests:
      - TestExitSurfacing_PackInstall_PrintsDiagnostic
  - id: CLM-070
    requirement: REQ-011
    text: A failing pack update prints its diagnostic to stderr and exits 1, asserted through the reportError seam
    subject: cmd/backstop
    tests:
      - TestExitSurfacing_PackUpdate_PrintsDiagnostic
  - id: CLM-071
    requirement: REQ-011
    text: A failing pack upgrade prints its diagnostic to stderr and exits 1, asserted through the reportError seam
    subject: cmd/backstop
    tests:
      - TestExitSurfacing_PackUpgrade_PrintsDiagnostic
  - id: CLM-072
    requirement: REQ-011
    text: A failing pack remove prints its diagnostic to stderr and exits 1, asserted through the reportError seam
    subject: cmd/backstop
    tests:
      - TestExitSurfacing_PackRemove_PrintsDiagnostic
  - id: CLM-073
    requirement: REQ-011
    text: A failing pack list prints its diagnostic to stderr and exits 1, asserted through the reportError seam
    subject: cmd/backstop
    tests:
      - TestExitSurfacing_PackList_PrintsDiagnostic
  - id: CLM-074
    requirement: REQ-011
    text: A failing pack relock prints its diagnostic to the separated stderr stream of the built CLI and exits 1, closing the ISSUE-074 silent exit
    subject: cmd/backstop
    tests:
      - TestExitSurfacing_PackRelock_PrintsDiagnostic
  - id: CLM-075
    requirement: REQ-011
    text: A failing recipe apply prints its diagnostic to the separated stderr stream of the built CLI and exits 1, closing the ISSUE-080 silent exit
    subject: cmd/backstop
    tests:
      - TestExitSurfacing_RecipeApply_PrintsDiagnostic
  - id: CLM-076
    requirement: REQ-011
    text: An artifact new that refuses an existing file prints its diagnostic to stderr and exits 1, asserted through the reportError seam
    subject: cmd/backstop
    tests:
      - TestExitSurfacing_ArtifactNew_PrintsDiagnostic
  - id: CLM-077
    requirement: REQ-011
    text: A gate run that finds violations exits 1 with its findings and no added Error line on the separated stderr stream
    subject: cmd/backstop
    tests:
      - TestExitSurfacing_GateViolations_NoDuplicateErrorLine
  - id: CLM-078
    requirement: REQ-011
    text: A gate configuration failure still prints its message and exits 2, unchanged by the explained opt-out
    subject: cmd/backstop
    tests:
      - TestExitSurfacing_GateConfigError_StillPrints
  - id: CLM-079
    requirement: REQ-011
    text: A pack check that finds validation errors exits 1 with its report and no added Error line on the separated stderr stream
    subject: cmd/backstop
    tests:
      - TestExitSurfacing_PackCheck_NoDuplicateErrorLine
  - id: CLM-080
    requirement: REQ-011
    text: A pack test that finds validation errors exits 1 with its report and no added Error line on the separated stderr stream
    subject: cmd/backstop
    tests:
      - TestExitSurfacing_PackTest_NoDuplicateErrorLine
  - id: CLM-081
    requirement: REQ-011
    text: An artifact validate that finds violations exits 1 with its report and no added Error line on the separated stderr stream
    subject: cmd/backstop
    tests:
      - TestExitSurfacing_ArtifactValidate_NoDuplicateErrorLine
  - id: CLM-082
    requirement: REQ-011
    text: An ExitCodeError carrying a message with Explained unset always prints through reportError, so the default is loud
    subject: cmd/backstop
    tests:
      - TestExitSurfacing_DefaultIsLoud

  # REQ-012 — structured JSON errors under --json
  - id: CLM-083
    requirement: REQ-012
    text: A failing pack add with --json writes exactly one parseable JSON error object to the separated stdout stream of the built CLI
    subject: cmd/backstop
    tests:
      - TestJSONError_PackAdd_EmitsSingleParseableObject
  - id: CLM-084
    requirement: REQ-012
    text: The JSON error object carries the command path, an error kind, and the human message
    subject: cmd/backstop
    tests:
      - TestJSONError_CarriesCommandKindAndMessage
  - id: CLM-085
    requirement: REQ-012
    text: A git failure is classified as kind git and a validation failure as kind validation
    subject: cmd/backstop
    tests:
      - TestJSONError_ClassifiesGitAndValidationKinds
  - id: CLM-086
    requirement: REQ-012
    text: A dependency failure is classified as kind dependency and an unavailable capability as kind capability
    subject: cmd/backstop
    tests:
      - TestJSONError_ClassifiesDependencyAndCapabilityKinds
  - id: CLM-087
    requirement: REQ-012
    text: An unclassified failure is classified with the default kind rather than an empty kind
    subject: cmd/backstop
    tests:
      - TestJSONError_UnclassifiedFailureUsesDefaultKind
  - id: CLM-088
    requirement: REQ-012
    text: --json leaves the separated stderr stream empty so stdout holds exactly one JSON document
    subject: cmd/backstop
    tests:
      - TestJSONError_SuppressesHumanDiagnostic
  - id: CLM-089
    requirement: REQ-012
    text: Failing pack install, pack update, and pack upgrade emit the same JSON error envelope as pack add
    subject: cmd/backstop
    tests:
      - TestJSONError_InstallUpdateUpgradeShareEnvelope

  # REQ-013 — the redesigned local-install helpers
  - id: CLM-092
    requirement: REQ-013
    text: InstallContractsLocalPack installs the contracts pack source through a caller-supplied assembled command with validation running
    tests:
      - TestInstallContractsLocalPack_InstallsWithSuppliedCommand
  - id: CLM-093
    requirement: REQ-013
    text: The packs/contracts source passes pack check and pack test through the unconditional validation path, so the dogfood install stays green
    tests:
      - TestInstallContractsLocalPack_ContractsPackPassesUnconditionalValidation
  - id: CLM-094
    requirement: REQ-013
    text: The distribution package declares no validator-aware install variant and no nil-validator-skip path
    kind: absence
    tests:
      - TestDistribution_NoValidatorAwareInstallVariant
  - id: CLM-100
    requirement: REQ-013
    text: Every SPEC-015-mandated test name declared in the six migrated distribution suites still exists verbatim after migration, so SPEC-015's claim-to-test lineage is unbroken
    kind: absence
    tests:
      - TestDistribution_Spec015MandatedTestNamesPreserved
  - id: CLM-095
    requirement: REQ-013
    text: The installSubstantivenessLocalPack method on e2eWorkspace keeps its receiver and signature, obtains its command from the production assembly helper, and installs the substantiveness pack with validation running
    subject: cmd/backstop
    tests:
      - TestInstallSubstantivenessLocalPack_UsesProductionAssembly

  # REQ-014 — the seams that make REQ-011 and REQ-012 assertable
  - id: CLM-096
    requirement: REQ-014
    text: reportError writes the diagnostic to the supplied writer and returns the error's exit code
    subject: cmd/backstop
    tests:
      - TestReportError_WritesDiagnosticAndReturnsCode
  - id: CLM-097
    requirement: REQ-014
    text: reportError writes nothing when Explained is set and still returns the exit code
    subject: cmd/backstop
    tests:
      - TestReportError_ExplainedWritesNothing
  - id: CLM-098
    requirement: REQ-014
    text: reportError maps an untyped error to the config exit code and still prints its message
    subject: cmd/backstop
    tests:
      - TestReportError_UntypedErrorMapsToConfigExit
  - id: CLM-099
    requirement: REQ-014
    text: The stream-separated CLI runner returns stdout and stderr independently for a command that writes to both, where the merged-output helper cannot
    subject: cmd/backstop
    tests:
      - TestRunBackstopStreams_SeparatesStdoutFromStderr

contracts:
  - file: pkg/pack/distribution/gitcloner.go
    provides:
      - name: ExecGitCloner
        kind: type
        signature: "type ExecGitCloner struct { GitBinary string; Timeout time.Duration }"
        notes: "The production GitCloner (REQ-001). GitBinary is the git executable to invoke; empty means \"git\" resolved from PATH — it exists so the missing-binary diagnostic (CLM-012) is testable without mutating the test process's PATH. Timeout bounds one git invocation so a hung remote cannot wedge CI (CLM-013). Neither field is a dependency in the DD-30 sense: this type IS the concrete production implementation, and REQ-005's no-internal-defaults rule forbids distribution synthesizing a GitCloner where one was not supplied, not a concrete implementation carrying its own tunables."
      - name: NewExecGitCloner
        kind: function
        signature: "func NewExecGitCloner() *ExecGitCloner"
        notes: "Constructs the production cloner with the package's default timeout. Deliberately does NOT probe for the git binary (CLM-009), because pack install --cache must assemble in an airgapped environment where git is absent; a missing binary surfaces at Clone/ListTags time as a typed GitError instead."
      - name: ExecGitCloner.Clone
        kind: method
        signature: "func (c *ExecGitCloner) Clone(url, version, destDir string) error"
        notes: "Satisfies the existing GitCloner interface declared at add.go:14. Runs a shallow single-ref git clone of the exact requested tag into destDir (which the Add/Install/Update/Upgrade pipelines already create via os.MkdirTemp, so it is empty), then STRIPS the root .git directory before returning (CLM-101), so every downstream copy and ComputeContentHash sees authored content only. This is what makes remote-pack content hashes REPRODUCIBLE across clones and machines within this spec — two clones of the same tag differ only in their .git (reflog timestamps, object layout), so removing it makes add and install agree by construction (CLM-065). It does NOT deliver bundle REQ-021@1.1.0, which is a requirement about the COPY/HASH boundary for ALL sources plus the legacy-lock migration; local-path packs still hash whatever is on disk, and this spec deliberately does not pin REQ-021 (Sharp Edges). Rejects an option-like url or version before invoking git and passes both after a -- separator (CLM-006/007). Sets the non-interactive git environment (CLM-004) while INHERITING the ambient environment so url.insteadOf redirection works (CLM-005). Every failure returns *GitError (REQ-002)."
      - name: ExecGitCloner.ListTags
        kind: method
        signature: "func (c *ExecGitCloner) ListTags(url string) ([]string, error)"
        notes: "Satisfies the existing GitCloner interface. Lists the REMOTE repository's tags and returns bare tag names with peeled ^{} entries excluded (CLM-003) — this is why pkg/scaffold's RealGitExecutor.ListTags(pattern string) cannot be reused: it lists the LOCAL repository's tags via git tag -l, a different operation with a different argument meaning, and it has no Clone method at all. An underlying git failure returns an error, never an empty slice (CLM-014)."
    consumes:
      - source: os/exec
        name: CommandContext
        kind: function
      - source: context
        name: WithTimeout
        kind: function
  - file: pkg/pack/distribution/validator.go
    provides:
      - name: PackvalValidator
        kind: type
        signature: "type PackvalValidator struct { }"
        notes: "The production Validator (REQ-003). Stateless: every call constructs a fresh packval pipeline over the given pack directory, exactly as cmd/backstop/pack_check.go and pack_test_cmd.go do."
      - name: NewPackvalValidator
        kind: function
        signature: "func NewPackvalValidator() *PackvalValidator"
        notes: "Constructs the production validator. This is the concrete implementation, not an internal default: nothing inside distribution may CALL it to fill a dependency a caller failed to supply (REQ-005/CLM-031), which is why the redesigned InstallContractsLocalPack receives an assembled command instead of building one. cmd/backstop's assembly file is the only production caller."
      - name: PackvalValidator.RunPackCheck
        kind: method
        signature: "func (v *PackvalValidator) RunPackCheck(packDir string) error"
        notes: "Satisfies the existing Validator interface declared at add.go:20. Runs packval.NewPipeline(packDir, packval.PipelineOptions{Mode: \"check\"}).Run() — the same pipeline and mode the pack check command runs — and returns *ValidationError when the result status is fail (CLM-015/016/019). No second implementation of pack validation exists."
      - name: PackvalValidator.RunPackTest
        kind: method
        signature: "func (v *PackvalValidator) RunPackTest(packDir string) error"
        notes: "Satisfies the existing Validator interface. Runs the pipeline in \"test\" mode, matching the pack test command, which likewise supplies no explicit FixtureExecutor — packval.RunFixtures substitutes its DefaultExecutor for a nil one (phase3.go), so the validator's fixture behavior is identical to the CLI's by construction (CLM-017/018/019)."
    consumes:
      - source: pkg/packval
        name: NewPipeline
        kind: function
      - source: pkg/packval
        name: PipelineOptions
        kind: type
  - file: pkg/pack/distribution/versionresolver.go
    provides:
      - name: TagVersionResolver
        kind: type
        signature: "type TagVersionResolver struct { git GitCloner }"
        notes: "The production VersionResolver (REQ-004). The git field is UNEXPORTED so the resolver cannot be assembled by composite literal without a cloner — the same structural rule the command constructors apply (REQ-005/REQ-006)."
      - name: NewTagVersionResolver
        kind: function
        signature: "func NewTagVersionResolver(git GitCloner) (*TagVersionResolver, error)"
        notes: "The only way to build a TagVersionResolver: a complete assembly succeeds (CLM-090) and a nil cloner returns *MissingDependencyError (CLM-047)."
      - name: TagVersionResolver.ResolveLatestCompatible
        kind: method
        signature: "func (r *TagVersionResolver) ResolveLatestCompatible(packName, currentVersion string) (string, error)"
        notes: "Satisfies the existing VersionResolver interface declared at update.go:14. Resolves the pack's git URL via the package's existing resolveGitURL, lists remote tags, keeps only strict X.Y.Z after stripping one optional leading v, and returns the highest same-major version — or currentVersion when none is newer (CLM-020/021/022/023/024/025/026). Strict X.Y.Z matches the corpus convention already used by pkg/validate/bundle.go's semverRe; no new module dependency is introduced (Masterminds/semver is present only as an indirect golangci-lint dependency)."
      - name: TagVersionResolver.IsMajorBump
        kind: method
        signature: "func (r *TagVersionResolver) IsMajorBump(current, resolved string) bool"
        notes: "Satisfies the existing VersionResolver interface. Compares major components only (CLM-027/028); the update pipeline already calls it as a belt after resolution refuses to cross a major."
  - file: pkg/pack/distribution/command.go
    provides:
      - name: MissingDependencyError
        kind: type
        signature: "type MissingDependencyError struct { Command string; Dependency string }"
        notes: "Typed fail-closed constructor error (REQ-006), in the same shape as the package's existing GitError and ValidationError. Names both the command being assembled and the dependency that was nil, so the diagnostic identifies the wiring site."
      - name: CapabilityUnavailableError
        kind: type
        signature: "type CapabilityUnavailableError struct { Capability string; Reference string }"
        notes: "Typed error for a required capability that is declared in the interface set but not yet implemented (REQ-009). Reference names the requirement tracking the gap — pack upgrade's remediation generation and violation scan are BUNDLE-006 REQ-014/REQ-018, which this spec does NOT deliver and does NOT pin — so the diagnostic points at the work rather than reading as a defect. Declared in distribution (not cmd/backstop) so UpgradeCommand.Run can classify and propagate it (CLM-058) and json_error.go can key the capability kind off it (CLM-086), while the production implementations that RETURN it live in the assembly layer, which is why CLM-058/059 are distribution-level claims driven by test-declared doubles and CLM-091/060 carry the production-wiring proof."
      - name: AddCommand
        kind: type
        signature: "type AddCommand struct { git GitCloner; validator Validator }"
        notes: "Unexported dependency fields: an AddCommand cannot be built by composite literal outside the package, so NewAddCommand is the only assembly path (REQ-006)."
      - name: NewAddCommand
        kind: function
        signature: "func NewAddCommand(git GitCloner, validator Validator) (*AddCommand, error)"
        notes: "Positional constructor — omitting an argument fails to compile (DD-30 / OQ-8 option (c)); an explicitly written nil returns *MissingDependencyError (CLM-032/033/034/035). Add requires BOTH dependencies because pack add clones and validates (REQ-038@1.1.0). The local-path branch of pack add does not clone, but the cloner is still structurally required: NewExecGitCloner is free to construct and probes nothing, so a local-only consumer pays no cost (Sharp Edges)."
      - name: AddCommand.Run
        kind: method
        signature: "func (c *AddCommand) Run(packRef string, opts AddOptions) (*AddResult, error)"
        notes: "The former package-level Add pipeline, now a method reading its dependencies from the receiver (REQ-007). Validation is unconditional and runs before any copy into the consumer project (REQ-008/CLM-050/051/052). Note it has NO backstop.yml precheck before cloning — unlike the update and upgrade pipelines, which call readPackVersion first — which is why pack add is the only command whose existing CLI tests can reach a live network clone once the cloner is real (Pass 8)."
      - name: InstallCommand
        kind: type
        signature: "type InstallCommand struct { git GitCloner }"
        notes: "Install's only dependency is the cloner — it verifies content hashes and does NOT re-validate (DD-12/REQ-008/CLM-048/054)."
      - name: NewInstallCommand
        kind: function
        signature: "func NewInstallCommand(git GitCloner) (*InstallCommand, error)"
        notes: "A nil cloner returns *MissingDependencyError (CLM-036/037). The cloner is required even for the --cache path, which never clones, because NewExecGitCloner probes nothing and an optional dependency is exactly the shape DD-30 forbids."
      - name: InstallCommand.Run
        kind: method
        signature: "func (c *InstallCommand) Run(opts InstallOptions) (*InstallResult, error)"
        notes: "The former package-level Install pipeline as a method. The install.go:165 nil-cloner guard and its diagnostic-free error are removed: the cloner is present by construction, so the remaining failure modes are real git and hash failures, each carrying a diagnostic (REQ-011)."
      - name: UpdateCommand
        kind: type
        signature: "type UpdateCommand struct { git GitCloner; validator Validator; resolver VersionResolver }"
        notes: "Update resolves a compatible version, clones, validates, and tamper-checks, so it requires all three (REQ-038@1.1.0)."
      - name: NewUpdateCommand
        kind: function
        signature: "func NewUpdateCommand(git GitCloner, validator Validator, resolver VersionResolver) (*UpdateCommand, error)"
        notes: "A nil in any position returns *MissingDependencyError naming that dependency (CLM-038/039/040/041). Replaces the runtime 'version resolver required for update' check at update.go:55, which was a fail-closed guard on an optional field."
      - name: UpdateCommand.Run
        kind: method
        signature: "func (c *UpdateCommand) Run(packName string, opts UpdateOptions) (*UpdateResult, error)"
        notes: "The former package-level Update pipeline as a method. Retains the readPackVersion precheck against backstop.yml, which runs BEFORE any clone."
      - name: UpgradeCommand
        kind: type
        signature: "type UpgradeCommand struct { git GitCloner; validator Validator; scanner Scanner; remediation RemediationGenerator }"
        notes: "Upgrade clones, validates, scans the consumer codebase, and generates remediation artifacts, so it requires all four (REQ-038@1.1.0)."
      - name: NewUpgradeCommand
        kind: function
        signature: "func NewUpgradeCommand(git GitCloner, validator Validator, scanner Scanner, remediation RemediationGenerator) (*UpgradeCommand, error)"
        notes: "A nil in any position returns *MissingDependencyError naming that dependency (CLM-042/043/044/045/046)."
      - name: UpgradeCommand.Run
        kind: method
        signature: "func (c *UpgradeCommand) Run(packRef string, opts UpgradeOptions) (*UpgradeResult, error)"
        notes: "The former package-level Upgrade pipeline as a method, with the scan moved AHEAD of tool-config merge so an unavailable capability fails before any consumer mutation (REQ-009/CLM-059), and with the nil-skip guards on scanner and remediation generator removed (CLM-056/057). Propagates *CapabilityUnavailableError unchanged (CLM-058). Retains the readPackVersion precheck against backstop.yml, which runs BEFORE any clone."
  - file: pkg/pack/distribution/add.go
    provides:
      - name: AddOptions
        kind: type
        signature: "type AddOptions struct { ProjectDir string; Version string }"
        notes: "The GitCloner and Validator fields are REMOVED (REQ-005) — those fields are exactly what let cmd/backstop/pack_add.go:31-34 build a nil-dependency options value. NOTE that this declaration does NOT itself enforce their removal: the contracts pack's signature compiler reduces any struct to `type AddOptions $$$` (compile-signature.sh's leading-token `type ` branch) and never compares field lists, so a reintroduced GitCloner field would still match this contract. The enforcement is CLM-030, the dedicated absence claim that scans the four options structs for dependency-typed fields. CLM-030 is therefore load-bearing and must not be dropped as redundant with this entry; the same caveat applies to the InstallOptions, UpdateOptions, and UpgradeOptions declarations below."
  - file: pkg/pack/distribution/install.go
    provides:
      - name: InstallOptions
        kind: type
        signature: "type InstallOptions struct { ProjectDir string; CachePath string; LocalPackDir string }"
        notes: "The GitCloner field is REMOVED (REQ-005/CLM-030). LocalPackDir is retained unchanged — it is a source override for local packs, not a dependency."
  - file: pkg/pack/distribution/update.go
    provides:
      - name: UpdateOptions
        kind: type
        signature: "type UpdateOptions struct { ProjectDir string; Acknowledge bool }"
        notes: "The GitCloner, Validator, and VersionResolver fields are REMOVED (REQ-005/CLM-030). Acknowledge is retained — it is the consumer's tamper-acknowledgment flag (bundle REQ-013), not a dependency."
  - file: pkg/pack/distribution/upgrade.go
    provides:
      - name: UpgradeOptions
        kind: type
        signature: "type UpgradeOptions struct { ProjectDir string }"
        notes: "All four dependency fields (GitCloner, Validator, RemediationGenerator, Scanner) are REMOVED (REQ-005/CLM-030), leaving only the project directory."
  - file: pkg/pack/distribution/contracts_local_install.go
    provides:
      - name: InstallContractsLocalPack
        kind: function
        signature: "func InstallContractsLocalPack(add *AddCommand, repoRoot, projectDir string) (*AddResult, error)"
        notes: "REDESIGNED (REQ-013). Was InstallContractsLocalPack(repoRoot, projectDir) delegating to InstallContractsLocalPackWithValidator(..., nil), which called the free Add at contracts_local_install.go:40 — the entry point REQ-007 deletes — and documented 'a nil Validator skips the stale pack check/test phases'. It now RECEIVES an assembled *AddCommand rather than assembling one, because a library-layer helper constructing its own PackvalValidator would be precisely the internal defaulting REQ-005 forbids (CLM-031/092). The validator-aware variant is DELETED along with its nil-skip doc contract, since under REQ-008 validation is unconditional and the distinction it existed to express no longer exists (CLM-094); packs/contracts passes those phases (CLM-093)."
      - name: ContractsPackSourceDir
        kind: function
        signature: "func ContractsPackSourceDir(repoRoot string) string"
        notes: "UNCHANGED. Declared here only to pin that the source-dir resolver is unaffected by the helper redesign."
    consumes:
      - source: path/filepath
        name: Join
        kind: function
  - file: cmd/backstop/pack_wiring.go
    provides:
      - name: newProductionAddCommand
        kind: function
        signature: "func newProductionAddCommand() (*distribution.AddCommand, error)"
        notes: "THE production assembly seam for pack add (REQ-010): constructs ExecGitCloner + PackvalValidator and returns the assembled command. Every pack lifecycle command's dependencies are built in this one file, so 'what does production actually wire?' has a single answer instead of being spread across four Cobra files that each built a partial options literal. Also the source installSubstantivenessLocalPack draws its command from (REQ-013/CLM-095) — legitimate here because cmd/backstop IS the assembly layer, unlike pkg/pack/distribution."
      - name: newProductionInstallCommand
        kind: function
        signature: "func newProductionInstallCommand() (*distribution.InstallCommand, error)"
        notes: "Assembles pack install with the production cloner only — install does not validate (DD-12)."
      - name: newProductionUpdateCommand
        kind: function
        signature: "func newProductionUpdateCommand() (*distribution.UpdateCommand, error)"
        notes: "Assembles pack update with the production cloner, validator, and a TagVersionResolver built over the same cloner."
      - name: newProductionUpgradeCommand
        kind: function
        signature: "func newProductionUpgradeCommand() (*distribution.UpgradeCommand, error)"
        notes: "Assembles pack upgrade with the production cloner and validator plus the explicit unavailable-capability implementations below, so the missing scan/remediation capability fails loud instead of being a nil that silently produced a zero-violation success."
      - name: unavailableScanner
        kind: type
        signature: "type unavailableScanner struct { }"
        notes: "An explicit Scanner implementation standing in for the violation scan that BUNDLE-006 REQ-014 owns and this spec does not build. It returns *CapabilityUnavailableError rather than an empty violation slice, converting a silent vacuous success into a readable failure (CLM-091/060)."
      - name: unavailableScanner.ScanViolations
        kind: method
        signature: "func (s unavailableScanner) ScanViolations(projectDir, packDir string) ([]string, error)"
        notes: "Satisfies the existing Scanner interface declared at upgrade.go:17."
      - name: unavailableRemediationGenerator
        kind: type
        signature: "type unavailableRemediationGenerator struct { }"
        notes: "An explicit RemediationGenerator implementation standing in for the remediation bundle generation that BUNDLE-006 REQ-018 owns and this spec does not build."
      - name: unavailableRemediationGenerator.GenerateBundle
        kind: method
        signature: "func (g unavailableRemediationGenerator) GenerateBundle(projectDir string, violations []string) (string, error)"
        notes: "Satisfies the existing RemediationGenerator interface declared at upgrade.go:12."
    consumes:
      - source: pkg/pack/distribution
        name: NewExecGitCloner
        kind: function
      - source: pkg/pack/distribution
        name: NewPackvalValidator
        kind: function
      - source: pkg/pack/distribution
        name: NewTagVersionResolver
        kind: function
      - source: pkg/pack/distribution
        name: NewAddCommand
        kind: function
      - source: pkg/pack/distribution
        name: NewInstallCommand
        kind: function
      - source: pkg/pack/distribution
        name: NewUpdateCommand
        kind: function
      - source: pkg/pack/distribution
        name: NewUpgradeCommand
        kind: function
  - file: cmd/backstop/artifact_validate.go
    provides:
      - name: ExitCodeError
        kind: type
        signature: "type ExitCodeError struct { Code int; Message string; Explained bool }"
        notes: "Explained is the NEW field (REQ-011). It is an explicit opt-OUT of printing, not an opt-in: reportError prints Message to its writer for every ExitCodeError with a non-empty message unless Explained is set. Only four sites may set it — gate (for its ExitViolations verdict only, so the exit-2 message keeps printing), pack check, pack test, and artifact validate — because only those four have already written structured findings to the consumer. The nine other ExitViolations sites leave it unset and therefore print, closing the silent exit-1 defect that ISSUE-074 (pack relock) and ISSUE-080 (recipe apply) each surfaced independently."
  - file: cmd/backstop/main.go
    provides:
      - name: reportError
        kind: function
        signature: "func reportError(w io.Writer, err error) int"
        notes: "The extracted, TESTABLE error-reporting seam (REQ-014). main() calls os.Exit and therefore cannot be exercised by a test, so its decision — which errors print, to which stream, and with which exit code — moves here and main() becomes os.Exit(reportError(os.Stderr, err)). Writes 'Error: <message>' to w unless the error is an *ExitCodeError with Explained set or an empty message; returns the ExitCodeError's Code, or ExitConfigError for an untyped error (CLM-096/097/098). This is what makes the per-site dispositions in REQ-011 assertable at all: without it the only available observable is runBackstop's merged CombinedOutput stream, against which every stream assertion passes vacuously."
    consumes:
      - source: io
        name: Writer
        kind: type
      - source: errors
        name: As
        kind: function
  - file: cmd/backstop/json_error.go
    provides:
      - name: writeJSONError
        kind: function
        signature: "func writeJSONError(w io.Writer, command string, err error) error"
        notes: "Renders the single structured JSON error document for a failing distribution command under --json (REQ-012). Classifies the kind by matching distribution's typed errors with errors.As — *GitError to git, *ValidationError to validation, *MissingDependencyError to dependency, *CapabilityUnavailableError to capability, everything else to the default kind (CLM-085/086/087). The caller writes to stdout and then returns an ExitCodeError with Explained set, so the JSON document is the only thing emitted and stderr stays empty (CLM-088)."
    consumes:
      - source: encoding/json
        name: Marshal
        kind: function
      - source: errors
        name: As
        kind: function
---

# SPEC-055: Production Remote Dependency Assembly

## Overview

BUNDLE-006's distribution posture is Homebrew's: install a pack by name from the tap
that hosts it, reproducibly, with remote as the default rather than an eventual
addition. Every install to date is a local path, so that posture is advertised
rather than proven — and the live verification recorded in bundle v0.9.0 shows why.
`backstop pack add <org>/<pack>@<version>` reaches `distribution.Add` with a nil
`GitCloner` and terminates with a raw SIGSEGV stack trace (ISSUE-073). The same
options literal supplies a nil `Validator`, and `add.go:160`'s
`if opts.Validator != nil` guard turns that into a silently skipped `pack check` +
`pack test` — a remote add that reached installation would install an unvalidated
pack. `install.go:165` guards instead of dereferencing, so it does not panic; it
exits 1 with no stderr at all, which is harder to act on than the panic because
there is nothing to read.

Underneath all three is one shape: a required dependency was modeled as an optional
struct field. That is the shape OQ-8 resolved against. DD-30 makes lifecycle command
dependencies structurally mandatory — constructors make incomplete options
unrepresentable, a missing dependency is a compile-time failure rather than a runtime
diagnostic, and distribution provides no internal defaults so a test double is never
mistakable for production wiring. The rationale the user endorsed is that this
codebase is extended by agents working at speed from artifacts, and the nil-cloner
panic was exactly a case where the wiring was optional and nobody remembered it.
Structural impossibility beats remembered diligence.

This spec is the foundation seed. It builds the three production implementations that
do not exist anywhere in the tree, restructures the four lifecycle commands into
constructor-assembled types, wires the Cobra commands to real dependencies, redesigns
the two non-test helpers that depend on the nil-validator-skip contract those changes
remove, fixes the cross-cutting error-surfacing defect the bundle assigns to this
seam, and lands the two test mechanisms without which the error-surfacing claims
cannot be asserted at all. It drives one real `git clone` end to end through the built
binary so the production path is proven rather than stubbed.

Under the Clone-strip amendment it also verifies the full remote add → install round
trip with MATCHING content hashes. `Clone` removes the root `.git` it created before
returning, so the tree distribution copies and hashes is authored content only, and two
clones of the same tag — which otherwise differ in reflog timestamps and object layout
— hash identically. That closes the reproducibility gap for the remote source at the
seam that creates it. It is not the same thing as delivering REQ-021@1.1.0: that
requirement is a copy/hash boundary for ALL sources plus an explicit legacy-lock
migration, and local-path packs, mixed-boundary projects, and pre-existing contaminated
locks remain the identity/migration seed's work. Sharp Edges states both residuals.

Later seeds depend on the seams this one lands: authored-content identity and
legacy-lock migration, source-coordinate-versus-manifest identity, the shared
staged-filesystem transaction coordinator, and the hermetic parity suite, which
EXTENDS this spec's harness substrate to the full REQ-042 command matrix rather than
building it from scratch.

## Requirements

Requirements are defined in the frontmatter `requirements` block with their bundle
`supports` pins. This section states the load-bearing shapes in prose.

### The production implementations (REQ-001 through REQ-004)

Three concrete types land in `pkg/pack/distribution`, each satisfying an interface the
package already declares and none of which has a production implementation today:

| Interface (existing) | Declared at | Production implementation (new) |
| --- | --- | --- |
| `GitCloner` | `add.go:14` | `ExecGitCloner` — real `git clone` / remote tag listing |
| `Validator` | `add.go:20` | `PackvalValidator` — the `pkg/packval` pipeline in check and test modes |
| `VersionResolver` | `update.go:14` | `TagVersionResolver` — strict-semver resolution over remote tags |

`pkg/scaffold/git_executor_real.go`'s `RealGitExecutor` is not a candidate for reuse
and is not modified by this spec. It implements `ListTags(pattern string)` by running
`git tag -l <pattern>` against the LOCAL repository — a different operation with a
different argument meaning from `GitCloner.ListTags(url string)` — and it has no
`Clone` method at all. That is why it was never wired.

`ExecGitCloner` must inherit the ambient git configuration environment rather than
clearing it. This is what makes hermetic testing possible without a URL-rewriting
seam in production code: a test sets `GIT_CONFIG_GLOBAL` to a temporary gitconfig
containing a `url.<local-repo>.insteadOf = https://github.com/<org>/<pack>.git`
rewrite, and the production code path — including the hardcoded
`resolveGitURL` construction at `add.go:315` — runs unchanged against a local
repository, network-free. It is also what keeps a consumer's already-configured
credential helper working, which the bundle's out-of-scope list requires (backstop
adds no new authentication, credential storage, or SSH-agent management).

`TagVersionResolver` treats the same-major rule literally, including at major zero:
with a current version of `0.1.0` and a `v0.2.0` tag available, `0.2.0` is compatible
and resolves. Strict semver would call a `0.x` minor bump breaking; the bundle's
enforcement semver model (DD-8) is about what a version change means for RULES —
adding rules is major, loosening is minor, false-positive fixes are patch — and says
nothing about a `0.x` caveat. Choosing the literal reading and stating it is the
honest disposition; changing it later is a bundle revision, not an implementation
detail.

### Constructor-enforced assembly (REQ-005 through REQ-007)

The dependency matrix below is the whole of REQ-006. Each constructor takes exactly
the dependencies its command requires and nothing else:

| Constructor | GitCloner | Validator | VersionResolver | Scanner | RemediationGenerator |
| --- | --- | --- | --- | --- | --- |
| `NewAddCommand` | required | required | — | — | — |
| `NewInstallCommand` | required | — | — | — | — |
| `NewUpdateCommand` | required | required | required | — | — |
| `NewUpgradeCommand` | required | required | — | required | required |
| `NewTagVersionResolver` | required | — | — | — | — |

A dash means the constructor does not take that dependency at all. Install takes no
validator because `pack install` is the hash-verified restore path and does not
re-validate (DD-12). Update takes no scanner or remediation generator because
minor/patch changes are non-breaking under the semver contract and generate no
remediation bundle (DD-13).

Three things together make an incomplete command unrepresentable rather than merely
discouraged, and all three are required — any one alone leaves the hole open:

1. The options structs lose every dependency-typed field, so no caller can express an
   options value that omits a dependency.
2. The package-level `Add`, `Install`, `Update`, and `Upgrade` free functions are
   removed and their pipelines become methods, so there is no entry point reachable
   without a constructor.
3. Each constructor is positional, so omitting an argument fails to compile.

An explicitly written `nil` argument remains expressible in Go, so each constructor
additionally fails closed with a typed `*MissingDependencyError` naming the command
and the dependency. That is DD-30's sanctioned fail-closed validation serving as a
backstop to the structural rule rather than as a substitute for it.

### Validation and capability honesty (REQ-008, REQ-009)

The `if opts.Validator != nil` guards in the add and upgrade pipelines are removed;
validation is unconditional and runs before any consumer state is mutated. That single
change is what delivers the validate-before-install obligation for BOTH commands,
which is why REQ-008 pins bundle REQ-002 (validation gate at add) and REQ-015
(validation gate at upgrade) together.

The `if opts.Scanner != nil` and
`if len(violations) > 0 && opts.RemediationGenerator != nil` guards in the upgrade
pipeline are removed too, because their current effect is a vacuous success: with both
dependencies nil, `pack upgrade` skips the scan, generates no remediation bundle,
installs the new major version, and reports success with zero baselined violations.

Neither the violation scanner nor the remediation bundle generator is built by this
spec — they are bundle REQ-014 and REQ-018, and this spec does NOT pin them as
delivered, because pinning an undelivered requirement would mark it covered for the
traceability gate. What REQ-009 delivers is the assembly and fail-loud propagation of
those dependencies, which is REQ-038's obligation for upgrade, so REQ-009 pins
REQ-038 and nothing else. Production wires explicit implementations that return
`*CapabilityUnavailableError` naming the capability and its tracking requirement, and
the upgrade pipeline runs its scan BEFORE it merges tool config, so the failure lands
before any consumer mutation. `pack upgrade` therefore becomes loudly unavailable
instead of quietly wrong. That is a deliberate trade, and it is named in Sharp Edges.

### The helper redesign the deletions force (REQ-013)

Deleting the free functions and the nil-validator-skip contract breaks two NON-TEST
files, and their replacements are designed here rather than left to the implementer:

| File | Today | Replacement |
| --- | --- | --- |
| `pkg/pack/distribution/contracts_local_install.go:40` | `InstallContractsLocalPack(repoRoot, projectDir)` delegates to `InstallContractsLocalPackWithValidator(..., nil)`, which calls free `Add` with `AddOptions{Validator: validator}` and documents "a nil Validator skips those phases" | One helper `InstallContractsLocalPack(add *AddCommand, repoRoot, projectDir string)` that RECEIVES an assembled command; the validator-aware variant and the nil-skip doc contract are deleted |
| `cmd/backstop/gate_substantiveness_e2e.go:106` | The METHOD `func (w *e2eWorkspace) installSubstantivenessLocalPack(repoRoot string) error` calls `distribution.Add(..., AddOptions{ProjectDir: w.root})`, relying on the nil-validator skip | Signature UNCHANGED (still a method on `*e2eWorkspace`); obtains its command from `newProductionAddCommand()` |

The asymmetry is deliberate and is REQ-005's boundary made concrete. A helper inside
`pkg/pack/distribution` may NOT construct a `PackvalValidator` for itself — that is
exactly the internal defaulting REQ-005 forbids, and it would make a library-layer
helper indistinguishable from production wiring. A helper inside `cmd/backstop` MAY,
because `cmd/backstop` IS the assembly layer. The substantiveness helper is a METHOD
on `*e2eWorkspace`, not a free function, and keeping that receiver and signature
stable is exactly what lets its four existing call sites need no edits at all.

Both helpers now validate unconditionally. This is safe: `packs/contracts` and
`packs/substantiveness` both pass `pack check` and `pack test`, and CLM-093 asserts it
rather than assuming it.

### Error surfacing (REQ-011, REQ-012) and the seams that make it assertable (REQ-014)

`cmd/backstop/main.go` currently suppresses the error message whenever the exit code
is `ExitViolations`, on the assumption that a command exiting 1 has already printed
structured findings. That assumption holds for `gate`, `pack check`, `pack test`, and
`artifact validate`. It is false for every command that uses exit 1 to mean "this
operation failed": the diagnostic is generated and then discarded. Three victims in
three subsystems — `pack relock` (ISSUE-074), `pack install`, and `recipe apply`
(ISSUE-080) — make it a property of the seam, not a per-command oversight.

The default inverts. `ExitCodeError` gains `Explained`, the reporting decision moves
into `reportError`, and exactly these sites are affected:

| Site | Disposition |
| --- | --- |
| `gate` (ExitViolations verdict) | `Explained` — findings already printed |
| `gate` (exit 2) | prints, unchanged |
| `pack check` | `Explained` — report already printed |
| `pack test` | `Explained` — report already printed |
| `artifact validate` | `Explained` — violations already printed |
| `pack add` | prints (was silent) |
| `pack install` | prints (was silent) |
| `pack update` | prints (was silent) |
| `pack upgrade` | prints (was silent) |
| `pack remove` | prints (was silent) |
| `pack list` | prints (was silent) |
| `pack relock` | prints (was silent — ISSUE-074) |
| `recipe apply` | prints (was silent — ISSUE-080) |
| `artifact new` | prints (was silent) |

The opt-out direction is the point: a command added later that forgets to declare
itself explained produces a duplicated diagnostic, which a reviewer notices, rather
than a silent failure, which nobody notices until someone dogfoods it.

Under `--json`, a failing `pack add`, `pack install`, `pack update`, or `pack upgrade`
writes one structured JSON error document to stdout carrying the command path, a kind
derived from distribution's typed errors, and the human message, then returns an
`ExitCodeError` with `Explained` set so stdout holds exactly one document and stderr
stays clean. The four commands currently ignore the persistent `--json` flag entirely
(their constructors take it as `_ *bool`).

None of this is assertable through the existing harness, which is why REQ-014 is a
requirement rather than an implementation note. `main()` calls `os.Exit`, so no test
can drive it — hence `reportError(w io.Writer, err error) int`, which takes the
writer and returns the code. And `runBackstop` (`pack_authoring_loop_test.go:27`)
returns `CombinedOutput`, a MERGED stream: against it, "prints its diagnostic to
stderr" passes for a command that printed to stdout, and "stdout holds exactly one
JSON document" passes for a run that also wrote a human line to stderr. Every claim
under REQ-011 and REQ-012 would pass vacuously. A stream-separated runner is therefore
mandatory, and each such claim names in its own text which of the two mechanisms
drives it.

## Implementation

The work decomposes into nine ordered passes. Each is separately compilable except
where noted; passes 4, 5, and 8 must land together because pass 4 deletes the entry
points passes 5 and 8's callers use.

**Pass 1 — `ExecGitCloner` (`pkg/pack/distribution/gitcloner.go`, new).**
Implement `Clone` as a shallow single-ref clone of the exact tag into the caller's
destination directory (which the pipelines already create via `os.MkdirTemp`, so it is
empty and git accepts it), and `ListTags` as a remote tag listing whose peeled `^{}`
entries are filtered out — including them would feed the resolver duplicate garbage.
Reject an option-like URL or ref before invoking git and pass both after a `--`
separator. Force git non-interactive so a credential or host-key prompt cannot block,
and otherwise inherit the environment. Bound each invocation by the configured timeout
via a cancellable context. Wrap every failure in `*GitError` carrying the operation,
URL, ref, and captured stderr. `NewExecGitCloner` performs no PATH probe. Before
returning a successful clone, remove the root `.git` directory, so every downstream
consumer — `copyDirRecursive` into `.backstop/packs/`, `ComputeContentHash`,
`DetectTamper` — sees authored content only.

The strip's blast radius inside distribution was swept rather than assumed, and it is
narrow. `copyDirRecursive` and `ComputeContentHash` are the intended beneficiaries: the
installed tree and its hash become authored-content-only for git sources, which is what
makes the round trip verify. `DetectTamper` is UNAFFECTED — `detectFixtureRemoval`
walks only `<oldPackDir>/testdata`, and every other check compares fields read from the
two `pack.yml` manifests, so no tamper check ever traverses a root `.git` in either the
installed tree or the fresh clone. It remains correct when an OLD installed pack still
carries `.git` (predating this seed) while the new clone does not, because neither side
of any comparison reaches that path. Provenance and `tool_config` merging read the pack
manifest and declared config fragments, never repository metadata, so they are likewise
untouched. The local-path branch of `AddCommand.Run` never calls `Clone` at all and is
therefore unchanged in every respect — that is residual (a) in Sharp Edges, not an
oversight.

**Pass 2 — `PackvalValidator` (`pkg/pack/distribution/validator.go`, new).**
`RunPackCheck` and `RunPackTest` each construct a `packval` pipeline over the given
pack directory in `check` and `test` mode respectively — the same construction
`cmd/backstop/pack_check.go` and `cmd/backstop/pack_test_cmd.go` perform — and return
`*ValidationError` when the result status is `fail`. Neither supplies an explicit
fixture executor, matching the commands; `packval.RunFixtures` substitutes its default
executor for a nil one, so behavior is identical to the CLI by construction rather
than by coincidence. `pkg/packval` imports `pkg/baseengines`, `pkg/check`, `pkg/pack`,
and `pkg/pack/engine`, none of which imports `pkg/pack/distribution`, so this
dependency edge introduces no import cycle.

**Pass 3 — `TagVersionResolver` (`pkg/pack/distribution/versionresolver.go`, new).**
Resolve the pack's git URL through the package's existing `resolveGitURL`, list remote
tags through the injected cloner, filter to strict `X.Y.Z` after stripping one optional
leading `v`, and select the highest same-major version, returning the current version
when none is newer. Parse with the corpus's existing strict-semver convention
(`pkg/validate/bundle.go`'s `semverRe` shape); no new module dependency is added.
`IsMajorBump` compares major components. A tag-listing error propagates.

**Pass 4 — command types and constructors (`pkg/pack/distribution/command.go`, new;
`add.go`, `install.go`, `update.go`, `upgrade.go` modified).**
Introduce `MissingDependencyError` and `CapabilityUnavailableError`. Define
`AddCommand`, `InstallCommand`, `UpdateCommand`, and `UpgradeCommand` with unexported
dependency fields and the positional constructors in the REQ-006 matrix, each
fail-closed on nil. Strip the dependency fields from the four options structs. Convert
the four package-level pipelines into methods on their command types and DELETE the
free functions. Inside the methods: remove the `Validator != nil` guards so validation
is unconditional (`add.go:160`, `upgrade.go:64`); remove the nil-cloner guard and its
diagnostic-free error (`install.go:165`); remove the runtime version-resolver-required
check (`update.go:55`); remove the `Scanner != nil` and `RemediationGenerator != nil`
guards and move the violation scan ahead of the tool-config merge (`upgrade.go`).

**Pass 5 — production wiring (`cmd/backstop/pack_wiring.go`, new; `pack_add.go`,
`pack_install.go`, `pack_update.go`, `pack_upgrade.go` modified).**
One file assembles every lifecycle command's production dependencies and returns
assembled commands, plus the two explicit unavailable-capability implementations for
upgrade. The four Cobra files call their assembly helper, treat an assembly failure as
an `ExitConfigError` (exit 2), and invoke the command's `Run` method. The
nil-dependency options literals at `pack_add.go:31-34` and its three siblings are gone.

**Pass 6 — error surfacing and the reporting seam (`cmd/backstop/artifact_validate.go`
and `cmd/backstop/main.go` modified, plus the fourteen exit sites).**
Add `Explained` to `ExitCodeError`. Extract `reportError(w io.Writer, err error) int`
into `main.go` carrying the print/suppress/exit-code decision, and reduce `main()` to
`os.Exit(reportError(os.Stderr, err))`. Set `Explained` at exactly four sites:
`gate.go:176` gated on `exitCode == ExitViolations` so the exit-2 message keeps
printing, `pack_check.go:44`, `pack_test_cmd.go:41`, and `artifact_validate.go:356`.
Leave it unset at `pack_add.go:38`, `pack_install.go:21`, `pack_update.go:26`,
`pack_upgrade.go:23`, `pack_remove.go:21`, `pack_list.go:21`, `pack_relock.go:20`,
`recipe_apply.go:70`, and `artifact_new.go:110`. Cobra's own error printing is already
suppressed by the root command's `SilenceErrors` (`root.go:30`), so `reportError`
remains the sole printer and no double-print arises from that direction.

**Pass 7 — JSON errors (`cmd/backstop/json_error.go`, new; the four pack Cobra files
modified).** Render one JSON error document classified by `errors.As` over
distribution's typed errors. The four pack commands stop ignoring the persistent
`--json` pointer their constructors already receive: on failure with `--json` set they
write the document to stdout and return an `ExitCodeError` with `Explained` set.

**Pass 8 — the named edit set the deletions force.** This is bounded and enumerated,
not open-ended. Two NON-TEST files are redesigned per REQ-013; the rest are mechanical
migrations or explicit retirements:

| File and site | Edit |
| --- | --- |
| `pkg/pack/distribution/contracts_local_install.go:40` | Redesign per REQ-013; delete `InstallContractsLocalPackWithValidator` and the nil-skip doc contract |
| `cmd/backstop/gate_substantiveness_e2e.go:107` | Draw the command from `newProductionAddCommand()`; signature unchanged |
| `pkg/pack/distribution/add_test.go` (58 tests, 57 free-function calls, 31 dependency-field lines) | BULK MIGRATION. Every test reaches the pipeline through free `Add` and injects its mocks through `AddOptions` fields, so all of it fails to compile after Pass 4. Migrate via a shared test helper that builds an `*AddCommand` from the existing mock cloner/validator values, so the per-test edit is mechanical: options literal in, helper call out |
| `pkg/pack/distribution/update_test.go` (25 tests, 25 calls, 61 dependency-field lines) | Same bulk migration, through a shared `*UpdateCommand` helper (cloner + validator + resolver mocks) |
| `pkg/pack/distribution/upgrade_test.go` (24 tests, 24 calls, 88 dependency-field lines) | Same, through a shared `*UpgradeCommand` helper (cloner + validator + scanner + remediation mocks). Highest dependency-field density of the six |
| `pkg/pack/distribution/install_test.go` (20 tests, 20 calls, 11 dependency-field lines) | Same, through a shared `*InstallCommand` helper (cloner mock only) |
| `pkg/pack/distribution/install_materialize_test.go` (7 tests, 7 calls, 2 dependency-field lines) | Same `*InstallCommand` helper |
| `pkg/pack/distribution/install_reconcile_test.go` (5 tests, 5 calls, 0 dependency-field lines) | Same `*InstallCommand` helper; call-site-only change |
| `pkg/pack/distribution/contracts_provisioning_test.go:95, 178, 193, 198` | Pass an assembled `*AddCommand` to the new `InstallContractsLocalPack` |
| `pkg/pack/distribution/contracts_provisioning_test.go:224` | DELETE `TestInstallContractsLocalPackWithValidator_RunsCheckPhases` — it exists solely to prove the nil-validator-skip distinction REQ-008 removes, so it cannot be migrated, only retired. Verified NOT mandated by any spec or plan, so no claim lineage is stranded |
| `pkg/pack/distribution/contracts_provisioning_test.go:215-219` | DELETE `stubValidator` with it — verified used by no other test in the package |
| `cmd/backstop/gate_substantiveness_provisioning_test.go:111` | Replace the direct `distribution.Add(...)` call with the assembled command |
| `cmd/backstop/gate_contract_wiring_test.go:235` | Pass an assembled command to `InstallContractsLocalPack` |
| `cmd/backstop/gate_contract_e2e_harness_test.go:118` | Same |
| `cmd/backstop/gate_contract_novacuous_test.go:74` | Same |
| `cmd/backstop/gate_substantiveness_e2e_test.go:57, 94, 168, 200` | NO EDIT — `installSubstantivenessLocalPack` keeps its signature |
| `cmd/backstop/coverage_cli_test.go:97-108` | `TestCLI_PackAdd_NonexistentPack` passes `nonexistent/pack@1.0.0` and today survives on `defer func() { recover() }()`. Post-spec it becomes a LIVE clone of `https://github.com/nonexistent/pack.git`, violating this seed's hermeticity. Retarget it through the hermetic `GIT_CONFIG_GLOBAL` `insteadOf` redirect so it exercises the missing-repository diagnostic offline, and drop the `recover` (there is no longer a panic to catch). If retargeting would merely duplicate CLM-063, delete the test instead — but it must not be left as-is |

The six distribution suites dominate this set: 139 tests, 138 free-function call sites,
and 193 dependency-field lines, all in external package `distribution_test`, reaching
the pipelines ONLY through the free functions and injecting mocks ONLY through the
options fields. They are the canonical consumers of the API Pass 4 deletes, and every
one of them fails to compile until migrated. The migration is mechanical rather than
semantic — the mocks themselves are unchanged; only their delivery moves from an
options literal to a constructor — which is why one shared command-building helper per
command type collapses the work.

**Every SPEC-015-mandated test name must be preserved VERBATIM.** 62 of the 100 test
names SPEC-015 mandates are declared in these six files. BUNDLE-006's revision-impact
section forbids rewriting SPEC-015's historical `REQ-021@1.0.0` pin, and renaming a
mandated test would break that spec's claim-to-test lineage just as effectively as
editing the pin would. The migration therefore changes call sites and assembly only —
never a `func TestX` identifier. CLM-100 asserts this mechanically rather than leaving
it as an instruction.

`pack add` is the ONLY one of the four commands whose existing CLI tests are exposed to
a live clone by this change. `TestCLI_PackUpdate_NonexistentPack` and
`TestCLI_PackUpgrade_NonexistentPack` reach `readPackVersion` against a `backstop.yml`
declaring no packs and error before any clone, and `TestCLI_PackInstall_NoLockfile`
errors on the absent lock. `AddCommand.Run` has no such precheck — it clones
immediately after the installed-and-current test.

**Pass 9 — hermetic end-to-end coverage.** Add the stream-separated CLI runner required
by REQ-014 alongside the existing `runBackstop` (which stays for the merged-output
callers that legitimately do not care about streams). The hermetic harness creates a
temporary git repository containing a pack that genuinely passes `pack check` and
`pack test` (verified by driving those commands against the same fixture in the same
test, so the fixture is proven rather than asserted), tags it, and points a temporary
`GIT_CONFIG_GLOBAL` `insteadOf` rewrite at it. The end-to-end claims drive the binary
built by the existing `buildBackstopBinary` helper, so the exact command wiring shipped
to consumers is what executes. This harness is the substrate the REQ-042 parity suite
extends to the full command matrix.

## Verification

`go test ./pkg/pack/distribution/... ./cmd/backstop/... -race -coverprofile=cover.out`
at integration level with an 80% coverage threshold. Integration is the right level
because the load-bearing claims cross a package boundary (`cmd/backstop` assembling
`pkg/pack/distribution`), cross a process boundary (real `git` subprocesses), and are
driven through the built binary rather than in-process.

Claims are defined in the frontmatter `claims` block. The spec-level subject is
`pkg/pack/distribution`; claims whose mandated test drives the CLI or lives in package
`main` carry a per-claim `subject: cmd/backstop`. Six claims are marked
`kind: absence` because each proves a structural fact about what is or is not present
in the sources rather than a runtime behavior: the dependency fields on the options
structs are gone, distribution's internal defaults are gone, the free-function entry
points are gone, the validator-aware install variant is gone, no CLI command constructs
a dependency-carrying options value, and no SPEC-015-mandated test name was lost in the
migration. `kind: absence` suppresses the substantiveness noTarget join for those
tests, which is independent of `subject`; the two coexist, and the CLI-side absence
claim carries both because its scan targets the `cmd/backstop` sources.

CLM-030 deserves particular protection. The `AddOptions` contract declaration LOOKS
like it enforces the dependency-field removal, but the contracts pack's signature
compiler reduces every struct to `type Name $$$` and never compares field lists, so a
reintroduced `GitCloner` field would still satisfy the contract. CLM-030's source scan
is the only thing that actually catches it.

Two claims under REQ-009 are distribution-level and driven by test-declared doubles
(the production unavailable-capability implementations live in package `main`, which
`pkg/pack/distribution` cannot import): CLM-058 proves `UpgradeCommand.Run` propagates
the typed error unchanged and CLM-059 proves it fails before mutation. The
production-wiring proof lives on the `cmd/backstop` side — CLM-091 that the real stubs
name the capability and its tracking requirement, CLM-060 that `pack upgrade` reports
it end to end.

Each REQ-011 and REQ-012 claim names its driving mechanism in the claim text, because
the choice is load-bearing rather than incidental:

- The **`reportError` seam** drives the per-site disposition claims, where what matters
  is which errors print and with which exit code, observed on a supplied writer.
- The **stream-separated runner** drives every claim asserting on a specific stream of
  a real process: the two ISSUE victims (`pack relock`, `recipe apply`), `pack add`,
  the four already-explained no-duplicate-line claims, and the JSON stdout-purity
  claims.
- Direct unit tests drive `writeJSONError`'s kind classification, which needs no
  process at all.

No claim under REQ-011 or REQ-012 may be driven by `runBackstop`'s merged
`CombinedOutput`; against a merged stream every one of them passes vacuously.

The coverage that decides whether this spec is real:

- At least one claim drives an actual `git clone` of an actual tagged repository
  through the actual production code path. A mock-only green would leave exactly the
  defect this spec exists to close.
- The round-trip equality claim (CLM-065) is paired with a genuine-mismatch claim
  (CLM-103) on purpose. Equality alone is satisfiable by deleting hash verification
  entirely; only the pair proves both that hashes agree AND that disagreement is still
  caught. Neither claim may be dropped as redundant with the other.
- The validation-failure claims are the proof that the validator is wired, not merely
  present: today a nil validator makes an invalid pack install cleanly, so a test that
  only asserts a valid pack installs would pass against the broken code.
- The error-surfacing claims assert on the DIAGNOSTIC the consumer sees, on the
  correct stream, never on the exit code alone. An exit-1-with-no-output failure passes
  an exit-code-only assertion, which is precisely how the three silent victims survived
  until someone dogfooded them.
- The dependency matrix is covered cell by cell: every constructor's complete assembly
  succeeds, and every individual dependency, nil'd alone, produces a named error.

## Sharp Edges

**Deleting the free functions breaks a bounded but LARGE set of call sites, and the
bulk of it is the distribution package's own test suites.** The headline number is not
the handful of helper sites: it is 139 tests across six files
(`add_test.go`, `update_test.go`, `upgrade_test.go`, `install_test.go`,
`install_materialize_test.go`, `install_reconcile_test.go`) with 138 free-function call
sites and 193 dependency-field lines, every one of which stops compiling at Pass 4.
That is roughly an order of magnitude more than the helper and CLI sites combined. The
set is fully enumerated in Pass 8 and the work is mechanical — mocks unchanged, only
their delivery moves — but anyone estimating this spec from the helper rows alone will
be badly wrong. Two of the breaking sites are additionally NOT tests
(`contracts_local_install.go`, `gate_substantiveness_e2e.go`); they are
production-shaped helpers that depend on the nil-validator skip and need designed
replacements (REQ-013), not migration. One test can only be retired. The consequence is
that pass 4, pass 5, and pass 8 cannot be split across separate merges, and the diff
will be very large in a way that is easy to mistake for scope creep.

**A careless migration can silently break SPEC-015's lineage.** 62 of the 100 test
names SPEC-015 mandates live in the six migrated suites. A migration that "tidies" a
test name while rewriting its call site severs that spec's claim-to-test join exactly
as a rewritten `REQ-021@1.0.0` pin would — and BUNDLE-006 explicitly forbids touching
that pin. The failure is quiet: the suite still passes, and only SPEC-015's traceability
goes red later. CLM-100 exists because an instruction not to rename would not survive a
139-test mechanical edit.

**One test can only be deleted, not migrated.**
`TestInstallContractsLocalPackWithValidator_RunsCheckPhases` asserts that supplying a
validator makes the check/test phases run — a distinction that exists only because a
nil validator skipped them. REQ-008 removes the skip, so the test's subject evaporates;
there is no migrated form of it that asserts anything. It is unmandated by any spec or
plan, so nothing is stranded, but a reader diffing the change will see test coverage
go down and should understand why.

**One existing test becomes a live network call.** `TestCLI_PackAdd_NonexistentPack`
currently survives because `distribution.Add` panics on the nil cloner and the test
catches it with `recover`. Once the cloner is real, that test attempts to clone
`https://github.com/nonexistent/pack.git` from CI. Hermeticity is a stated property of
this seed, so the test must be retargeted through the `insteadOf` redirect or deleted;
leaving it is a silent regression from "hermetic" to "depends on GitHub being
reachable and that repository staying absent." The add command is uniquely exposed
because, unlike update and upgrade, its pipeline has no `backstop.yml` precheck before
the clone.

**The Clone strip makes REMOTE hashes reproducible here, but leaves two residuals that
belong to the identity/migration seed.** Without the strip, `pack add` would copy the
clone directory — root `.git` included — into `.backstop/packs/` and hash it; the live
verification measured the same pack at `639f74fb…` without `.git` and `bb86715c…` with
it, and two clones of one tag differ in reflog timestamps alone, so a fresh-clone
install was guaranteed to mismatch. Stripping `.git` in `Clone` removes that at the
source, and CLM-065 now verifies round-trip hash EQUALITY end to end. What it does not
do is deliver bundle REQ-021@1.1.0, and this spec deliberately does not pin it. Two
residuals are willed forward to the authored-content-identity and legacy-lock-migration
seed, which must inherit them explicitly:

- **Local-path packs are untouched.** They are never cloned, so nothing strips anything:
  `ComputeContentHash` still hashes whatever is on disk, including any `.git` a local
  pack source happens to carry because it is itself a repository checkout. REQ-021@1.1.0
  owns the copy/hash boundary for ALL sources; this seed only fixed the one source it
  creates.
- **A transitional asymmetry now exists.** After this seed, remote installs are
  `.git`-free while local installs are unchanged, so two packs in the same project can
  have hashes computed under different effective content boundaries. Worse, any lock
  entry written BEFORE this seed still carries a `.git`-contaminated hash and will
  mismatch against a stripped clone — which is exactly the failure OQ-6's migration
  policy (DD-28) exists to distinguish from authored-content tampering. Nothing here
  migrates those entries; a consumer with a pre-existing git-sourced lock needs the
  migration operation, not this spec.

Anyone reading a green round trip here as "REQ-021 is done" will be wrong on both
counts.

**Stripping `.git` destroys information a later seed may want.** Once `Clone` returns,
the resolved commit SHA, the tag's peeled object, and the remote URL as git recorded it
are gone — they lived in the `.git` this spec deletes. If provenance, attestation, or
the migration operation later needs the commit a tag pointed at, the cloner must
CAPTURE it during the clone and return it as data, not read it back off the tree. A
future contributor who tries to recover it by reading `.git` will find nothing and may
"fix" it by removing the strip, silently reintroducing the non-reproducible hash.

**Hermetic testing depends on the cloner NOT sanitizing its environment.** URL
redirection works because `GIT_CONFIG_GLOBAL` reaches the git subprocess. A future
hygiene change that clears the environment for determinism would silently convert
every hermetic test into either a network call or a failure, and the "we test the
production path" property would evaporate without any test turning red for the right
reason. The inheritance is load-bearing behavior, not an oversight.

**`pack upgrade` gets worse before it gets better.** Today it silently reports a
successful major upgrade with zero baselined violations. After this spec it fails with
an unavailable-capability diagnostic. That is correct — a zero-violation major upgrade
that never ran a scan is a vacuous green — but it will read as a regression to anyone
who was relying on the command appearing to work, and any test asserting `pack
upgrade` succeeds must be re-examined rather than patched to expect success.

**Default-loud error surfacing can double-print.** A command that already writes its
own error to stderr AND returns a populated `Message` will now emit both. The site
audit in this spec is exhaustive as of authoring, but a command added later inherits
the loud default. The trade is deliberate and asymmetric: a duplicate diagnostic is
noticed and fixed, a silent failure is not.

**The merged-stream harness will hide a regression in exactly this area.**
`runBackstop` remains in the tree for its existing callers. Any future error-surfacing
test written against it rather than the stream-separated runner will pass whether or
not the diagnostic went to the right stream — which is how the current silent-exit-1
defect went unnoticed through several specs' worth of CLI tests.

**Newly visible `pack relock` errors will look like a fix that did not happen.**
Pass 6 makes `pack relock <name>` print its real error instead of nothing. The
underlying ISSUE-074 defect — that relock accepts a filesystem PATH while every
sibling command takes a pack NAME — is owned by the content-identity and migration
seed and is NOT fixed here. A consumer who sees a message where there was silence may
reasonably conclude relock now works.

**Requiring a `GitCloner` to construct `InstallCommand` is questionable for
`--cache`.** The airgapped restore path never clones. The dependency is required
anyway, because an "optional when a flag is set" dependency is exactly the shape DD-30
forbids. This is only safe because `NewExecGitCloner` performs no PATH probe; if a
later change makes it validate git's presence at construction time, every airgapped
`pack install --cache` breaks with a misleading error.

**The `0.x` compatibility reading is a real decision, not a default.** Treating
same-major literally means `pack update` auto-applies `0.1.0 → 0.9.0` without a
remediation bundle. For a pre-1.0 enforcement pack that is a large behavioral change
arriving through the frictionless path. The alternative — treating `0.x` minors as
major-equivalent — would strand every pre-1.0 pack on `pack upgrade`. The literal
reading is chosen and claimed; disagreement is a bundle revision.

**Peeled tag refs will corrupt version resolution if the filter is dropped.** A remote
tag listing that includes `refs/tags/v1.0.0^{}` alongside `refs/tags/v1.0.0` yields
duplicate and malformed entries. The resolver's strict-semver filter would discard the
`^{}` form today, so the bug would be invisible — until a future relaxation of the
version pattern lets it through. The exclusion belongs in the cloner, where the ref
shape is known, not in the resolver.

**Local-path `pack add` structurally requires a cloner it never uses.** A consumer who
only ever adds local packs still assembles an `ExecGitCloner`. This costs nothing
(construction probes nothing) but it does mean the dependency graph overstates what a
local-only workflow actually needs, which will look wrong to a reader tracing why a
local add depends on git.

## Review Questions

1. Does any code path in `pkg/pack/distribution` — including
   `contracts_local_install.go` — construct a `GitCloner`, `Validator`,
   `VersionResolver`, `Scanner`, or `RemediationGenerator` that was not passed in by a
   caller? A single fallback anywhere reinstates the defect DD-30 resolved against and
   makes a test double indistinguishable from production wiring.

2. Is there any remaining way to execute an add, install, update, or upgrade pipeline
   without going through a constructor — a surviving free function, an exported helper
   that takes an options value, or a method on a zero-value command? Grep for
   top-level `func Add(`, `func Install(`, `func Update(`, and `func Upgrade(` in the
   package, not just for the absence test passing.

3. Does at least one test invoke the real `git` binary against a real repository
   through the production `ExecGitCloner`, or is every clone in the suite a mock? If
   the answer is "a mock", this spec delivered nothing it exists to deliver.

4. Does any test in the suite reach the network? Specifically, was
   `TestCLI_PackAdd_NonexistentPack` retargeted or deleted — and does `go test` still
   pass with networking disabled?

5. Does the hermetic fixture pack actually pass `pack check` and `pack test`, verified
   by driving those commands against it — or was it hand-written to satisfy whatever
   the validator happened to accept? A fixture fabricated to fit already-written code
   cannot falsify anything.

6. Does the validation-failure test use a pack that genuinely fails validation, and
   does it fail for the reason the test claims? A pack that fails for an unrelated
   structural reason would pass the assertion while proving nothing about whether the
   validator is wired.

7. Is any REQ-011 or REQ-012 claim driven by `runBackstop`'s merged `CombinedOutput`
   rather than the stream-separated runner or the `reportError` seam? A merged-buffer
   assertion passes regardless of which stream was written, so such a claim proves
   nothing — and this is the exact blind spot that let the silent-exit-1 defect survive
   several specs' worth of CLI tests.

8. After pass 6, is there any `ExitCodeError` construction in `cmd/backstop` that sets
   `Explained` outside the four sanctioned sites — and does each sanctioned site
   actually print structured output on the path that returns it, or only on some
   paths? A gate path that returns `ExitViolations` without having printed findings
   would be newly silent.

9. Does `UpgradeCommand.Run` run its violation scan before the tool-config merge, and
   is that ordering asserted by a test that would fail if the two were swapped back?
   Ordering that is correct by inspection but unasserted regresses on the next edit.

10. Do the constructor nil tests assert on the NAMED dependency in the error, or only
    that an error occurred? "An error occurred" passes even if the constructor reports
    the wrong dependency, which is the diagnostic quality REQ-006 is about.

11. Did the enumerated edit set stay closed? Specifically: did any call site outside
    the Pass 8 table need changing, and did deleting
    `TestInstallContractsLocalPackWithValidator_RunsCheckPhases` strand any claim?
    Verify against the current corpus rather than against this spec's assertion, since
    the corpus moves.

12. Did the six-suite migration rename, merge, or drop ANY test? Diff the set of
    `func Test…` identifiers in the six files before and after; it should differ by
    exactly the one retired test. In particular, are all 62 SPEC-015-mandated names
    still present verbatim — and does CLM-100's test actually read those names from
    SPEC-015 rather than from a hardcoded list that would drift?

13. Does `Clone` strip the root `.git` on EVERY successful return path, including
    whatever fast path a `--cache`-adjacent or retry branch might add later — and does
    the round-trip test actually clone twice (add from one clone, install from a fresh
    one) rather than reusing a single cloned tree? A round trip that reuses one clone
    proves nothing about reproducibility.

14. Does anything in the new code carry a language, tool, or platform literal beyond
    the `git` invocation itself? `git` is distribution's transport, declared by DD-1
    as the distribution channel, not a language toolchain — but a rule name, linter
    name, or file-extension branch appearing here would be a baked-knowledge defect
    and `backstop/self` should say so.

## References

- BUNDLE-006 — pack distribution lifecycle; this spec implements the "Production
  remote dependency assembly" reliability seed
- BUNDLE-006 DD-25 — production command wiring is part of the tested contract;
  hermetic tests redirect git URLs to temporary local repositories
- BUNDLE-006 DD-30 (OQ-8 option (c)) — constructors make incomplete options
  unrepresentable; fail-closed nil validation only as an incremental step
- BUNDLE-006 DD-1 — git-native distribution; `org/pack-name` resolves to a git URL
- BUNDLE-006 DD-8, DD-12, DD-13 — enforcement semver model; install is the
  hash-verified fast path that does not re-validate; update versus upgrade split
- BUNDLE-006 REQ-042 — the hermetic parity suite, which EXTENDS this spec's harness
  substrate to the full remote command matrix rather than building it from scratch
- BUNDLE-006 REQ-014, REQ-018 — the violation scanner and remediation bundle
  generator, which this spec deliberately does NOT deliver and does not pin
- BUNDLE-006 REQ-021@1.1.0, DD-24, DD-28 — the authored-content copy/hash boundary and
  the legacy-lock migration. The Clone strip here solves the REMOTE source only and is
  deliberately NOT pinned to REQ-021; the identity/migration seed must inherit the two
  residuals named in Sharp Edges (local-path packs, and the transitional asymmetry
  including pre-existing contaminated locks)
- ISSUE-073 — `pack add` nil GitCloner panic and silently skipped validation
- ISSUE-074 — `pack relock` silent exit 1; this spec fixes the silence, not the
  argument shape
- ISSUE-080 — `recipe apply` silent exit 1; the third victim of the same seam
- ADR-0001 — machine-first output; the `--json` structured error envelope
- SPEC-037, SPEC-038 — the substantiveness and contracts provisioning harnesses whose
  local-install helpers REQ-013 redesigns
- SPEC-054 — recipe apply and manifest; precedent for per-claim `subject` and for
  declaring only contract forms the contracts pack can verify (ISSUE-078)

## Version History

- **1.3.1** (2026-07-26): Status flip to `implemented`. Documentary only — no
  requirement, claim, contract, or behavior changed; the delivered contract is the one
  1.3.0 states. PLAN-SPEC-055 executed in full (53 tasks, all twelve phases) through the
  commit series `cbe0869..40a92cd`: the hermetic remote substrate (phase 1), the
  `reportError` seam and separated-stream runner (phase 2), per-site error surfacing —
  four `Explained`, nine loud (phase 3), `ExecGitCloner` with the Clone strip (phase 4),
  `PackvalValidator` and `TagVersionResolver` (phases 5–6), the fail-closed positional
  constructors that make an unassembled command a compile error (phase 7, DD-30 /
  OQ-8(c)), production cobra wiring that kills the ISSUE-073 nil-cloner panic (phase 8),
  the single `--json` error document on stdout (phase 9), the 134-test migration of the
  distribution suites onto the constructor-assembled commands (phase 10), the deletion of
  the four free functions and the dependency fields so nil is unrepresentable (phase 11),
  and the kill chain (phase 12). All 103 claims' mandated tests are present and green with
  ONE accountable exception: the `pack upgrade` SUCCESS path is unreachable from the
  production wiring until the scanner/remediation capability seed lands (BUNDLE-006
  REQ-014/REQ-018), so its coverage carries an expiring waiver at
  `cmd/backstop/pack_upgrade.go:1` — `@waiver:coverage_threshold:deferred:2026-10-24` —
  which that seed must remove rather than renew. TASK-053 reconciled the two
  `docs/CODEBASE-MAP.md` known-gap sections against the as-built tree. PLAN-SPEC-055 moves
  to `completed`; its `spec_version` pin stays at `1.3.0`, the version it was executed
  against.
- **1.3.0** (2026-07-26): Clone-strip amendment, user-adopted following an
  out-of-session mediation. `ExecGitCloner.Clone` now STRIPS the root `.git` directory
  it created before returning, so the tree handed to distribution is authored content
  only; REQ-001 and the `Clone` contract note state the obligation and the reasoning
  (the cloner is removing an artifact it made, which makes its contract "materialize
  the authored tree at this ref"). Added CLM-101 (clone returns a `.git`-free tree,
  driven against a real tagged repository) and CLM-102 (two clones of one tag hash
  identically). REPLACED CLM-065: the old claim asserted that a git-sourced install
  reports a hash MISMATCH loudly — an honesty claim about a known gap — with the
  stronger successor that a remote add followed by a fresh-clone install verifies hash
  EQUALITY end to end. Added CLM-103 to retain the genuine-mismatch-is-loud behavior as
  a vacuity guard, since equality alone would also pass if hash verification were
  deleted outright; Verification states the pair must not be collapsed. Rewrote the
  non-reproducible-hash Sharp Edge: remote hashes are reproducible IN THIS SEED, with
  two residuals explicitly willed to the identity/migration seed — (a) local-path packs
  are never cloned so they still hash whatever is on disk including any `.git` their
  source carries, and (b) a transitional asymmetry now exists in which remote installs
  are `.git`-free while local installs are unchanged and pre-existing git-sourced lock
  entries still carry contaminated hashes that only the DD-28 migration can repair.
  Added a Sharp Edge that the strip destroys the resolved commit SHA and other git
  metadata, which a later provenance or migration need must CAPTURE during the clone
  rather than read back off the tree. Swept the blast radius rather than spot-fixing:
  `DetectTamper` is unaffected (`detectFixtureRemoval` walks only `<oldPackDir>/testdata`
  and every other check compares `pack.yml` fields, so no comparison traverses a root
  `.git` on either side, including the mixed case of an old contaminated install against
  a stripped clone), and provenance/`tool_config` read manifests and declared fragments
  only; both findings are recorded in Pass 1. Updated the summary and Overview, which
  previously disclaimed the round trip, and added Review Question 13 on strip coverage
  and on the round-trip test genuinely cloning twice. This spec still does NOT pin
  REQ-021@1.1.0.
- **1.2.0** (2026-07-26): Second review rework closing three findings. (N1) Added the
  distribution package's OWN six external test suites to the Pass 8 edit set with a
  bulk-migration disposition — `add_test.go`, `update_test.go`, `upgrade_test.go`,
  `install_test.go`, `install_materialize_test.go`, `install_reconcile_test.go`, 139
  tests / 138 free-function call sites / 193 dependency-field lines, all in external
  package `distribution_test` and all failing to compile at Pass 4. They are the
  canonical consumers of the deleted API and were omitted entirely. Specified shared
  per-command test helpers as the migration vehicle, recorded that 62 of SPEC-015's 100
  mandated test names live in these files and must survive VERBATIM (BUNDLE-006 forbids
  rewriting SPEC-015's `REQ-021@1.0.0` pin, and a rename severs the same lineage),
  added CLM-100 to assert that mechanically, and rescaled the "bounded and enumerated"
  Sharp Edge — the set is bounded but roughly an order of magnitude larger than the
  previous table implied. (N2) Corrected the false claim in the `AddOptions` contract
  note that the contracts gate fails if a dependency field returns: the signature
  compiler's leading-token `type ` branch reduces any struct to `type AddOptions $$$`
  and never compares field lists, so CLM-030's source scan is the actual enforcement.
  Stated the caveat on all four options declarations and in Verification, explicitly so
  CLM-030 is not later dropped as redundant. (N3) Corrected
  `installSubstantivenessLocalPack` from a free function to its real form, the method
  `func (w *e2eWorkspace) installSubstantivenessLocalPack(repoRoot string) error`, in
  REQ-013, both body tables, and CLM-095 — the receiver is precisely what makes the
  "four call sites need no edits" claim true. Corrected that site's line reference to
  `:106`.
- **1.1.0** (2026-07-26): Review rework closing eight blockers. Added REQ-013 designing
  the replacements for the two NON-TEST helpers the deletions break
  (`distribution.InstallContractsLocalPack` / `InstallContractsLocalPackWithValidator`
  and package-main's `installSubstantivenessLocalPack`) and enumerating the full,
  bounded edit set in Pass 8, including the retirement of
  `TestInstallContractsLocalPackWithValidator_RunsCheckPhases` and its `stubValidator`
  (verified unmandated by any spec or plan, and unused elsewhere). Added REQ-014
  mandating the extracted `reportError` seam and a stream-separated CLI runner, because
  `main()` calls `os.Exit` and `runBackstop` returns merged `CombinedOutput`, against
  which every REQ-011/REQ-012 stream assertion would pass vacuously; each such claim
  now names its driving mechanism. Corrected the bundle pins: REQ-008 now pins REQ-002
  and REQ-015 together, since one unconditional-validation change delivers
  validate-before-install for both add and upgrade, and REQ-009 pins REQ-038 alone
  because bundle REQ-014 and REQ-018 are NOT delivered here and pinning them would mark
  them covered. Gave `cmd/backstop/coverage_cli_test.go:97` an explicit disposition,
  since a real cloner turns it into a live GitHub clone, and recorded that `add` is the
  only one of the four commands so exposed (update and upgrade precheck `backstop.yml`,
  install prechecks the lock). Split CLM-058/059 (distribution-level, test-declared
  doubles) from the production-wiring proof, adding CLM-091 for the production stubs'
  typed error. Corrected the absence-claim prose to five and removed the false
  statement that absence claims cannot carry a subject. Normalized every `install.go`
  reference to `:165`. Reframed the REQ-042 relationship: this spec LANDS the hermetic
  harness substrate that the parity seed extends, rather than deferring it wholesale.
- **1.0.0** (2026-07-26): Initial spec. Covers BUNDLE-006 REQ-038@1.1.0 (production
  dependency assembly and diagnostics), REQ-001@1.0.0 (git resolution, clone, and
  diagnostic exit), REQ-002@1.0.0 (validation gate at add), REQ-008@1.0.0 (install
  clones lock pins), REQ-015@1.0.0 (validation gate at upgrade), and REQ-030@1.0.0
  (enforcement semver resolution), plus the cross-cutting error-surfacing repair the
  bundle assigns to this seed. Explicitly defers REQ-021@1.1.0, REQ-039, REQ-040,
  REQ-041, and the REQ-014/REQ-018 capabilities to later seeds.
