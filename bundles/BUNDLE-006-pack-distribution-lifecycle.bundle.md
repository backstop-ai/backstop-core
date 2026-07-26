---
title: "Pack Distribution Lifecycle — From Validated Pack to Running Gate"
number: BUNDLE-006
created: "2026-04-14"
schema_version: bundle/v2

bundle:
  name: pack-distribution-lifecycle
  version: "0.8.0"
  created: "2026-04-14"
  updated: "2026-07-25"
  category: feature

status:
  maturity: exploring

problem:
  summary: >
    BUNDLE-006 defined and partially delivered the lifecycle between a
    validated pack repository and a consumer's running gate: add, remove,
    install, update, upgrade, lock verification, and tool-config provenance.
    Dogfooding a standalone published pack proved that production remote
    command wiring, source/manifest identity, authored-content hashing,
    legacy-lock migration, and transactional rollback do not yet reliably
    satisfy that contract. Remote commands can panic or fail before using
    their advertised dependencies, Git metadata can contaminate content
    identity, incoherent pack identity can fail only at gate time, and failed
    mutations can strand partial consumer state. The intended posture is
    Homebrew's — install a pack by name from the tap that hosts it, reproducibly,
    with remote as the default rather than an eventual addition — but every
    install to date is a local path. This reliability revision is what makes the
    remote, Homebrew-style posture real rather than advertised.

  user_story: >
    As a consumer, I want to add a pack by name, have it validated and
    installed automatically, have my tool configs updated, and have the gate
    enforce it — with a lockfile ensuring reproducibility across my team and
    CI. When I remove a pack, I want its config contributions reverted
    cleanly. When I deploy, I want `pack install` to restore everything from
    the lockfile without re-validating — fast, deterministic, reproducible.

  success_criteria:
    - >
      pack add works for both git URLs (org/pack-name@version) and local
      filesystem paths, resolving, cloning/loading, validating, installing,
      merging tool_config, and updating backstop.yml + backstop.lock in a
      single command
    - >
      pack remove reverts tool_config settings contributed by the target pack
      using provenance tracking, warns on manually-modified settings, and
      cleans up .backstop/packs/, backstop.yml, backstop.lock, and
      pack-config-provenance.json
    - >
      pack install restores all packs from backstop.lock with SHA-256 content
      hash verification, without re-running validation or merging tool_config
      (CI fast path)
    - >
      pack update resolves the latest compatible minor/patch version, validates
      the new version before updating, writes the new exact pin to
      backstop.yml, updates backstop.lock, and runs tamper detection with
      --acknowledge escalation
    - >
      pack upgrade handles major version bumps with an explicit version target,
      generates a remediation bundle scoping all new violations, and rolls back
      on remediation bundle generation failure
    - >
      pack list shows installed packs with version, lock status, archetype,
      rule count, and scaffold count in human-readable table format, with
      structured JSON output via --json
    - >
      backstop.lock is YAML with sorted keys, containing per-pack entries with
      name, version, git ref (null for local), SHA-256 content hash, source
      type, and install date
    - >
      backstop gate catches hash mismatches, missing packs, extra unlocked
      packs, and absent lockfile when packs are declared — each producing a
      specific diagnostic message and gate failure
    - >
      .backstop/pack-config-provenance.json tracks every tool_config setting
      merged by pack add, mapping config file path and setting key to source
      pack name and install-time value hash
    - >
      Error paths produce non-zero exit codes and diagnostic messages for:
      already-installed pack, not-installed pack, missing lockfile, partial
      install (atomic rollback), network/clone failure, validation failure,
      and tool_config conflict
    - >
      Every advertised remote command executes through its production-wired
      required dependencies: add clones and validates; install clones exact lock
      pins without validation; update resolves, clones, validates, and checks
      tampering; upgrade clones, validates, and generates remediation artifacts.
      Incomplete assembly and subprocess failures return diagnostics rather than
      panics
    - >
      Installed pack content and lock hashes exclude only root repository-control
      metadata, while existing metadata-inclusive locks have a deterministic,
      tamper-safe migration path: a legacy hash always fails with a typed
      migration diagnostic and is repaired only by pack relock (local) or an
      explicit re-validating remote migration operation, never accepted in place
    - >
      No lifecycle command can be assembled without its production Git,
      version-resolution, and validator dependencies — incomplete assembly is a
      compile-time failure, not a runtime diagnostic
    - >
      backstop.lock keys every entry by manifest pack name and records the
      requested source repository coordinate verbatim alongside it; install path,
      config key, and engine asset root all derive from the manifest name, so a
      pack whose name differs from its repository basename installs and dispatches
      coherently
    - >
      Add, install, update, upgrade, and remove restore the exact prior consumer
      state after any failed mutation across the command-specific surfaces in
      DD-27
    - >
      A hermetic parity suite exercises remote add/install/update/upgrade,
      gate-time lock verification, explicit remote legacy-lock migration, local
      relock migration, and remove cleanup
      through the production CLI against temporary tagged repositories without
      network access

solution:
  approach: >
    Git-native distribution modeled explicitly on Homebrew: the repository (tap)
    is where a pack lives, the pack's own name is the identity you install by,
    installs are reproducible from pinned references, and remote is the default
    posture rather than a later addition. backstop.lock supplies the
    reproducibility Homebrew gets from formula pins. `pack
    add` is the consumer entry point — it clones, validates, installs,
    merges config, and updates the lockfile in one command. `pack install`
    is the CI fast path — it restores from the lockfile with hash
    verification, skipping validation (already proven at add time). `pack
    remove` uses tool_config provenance tracking to cleanly revert config
    changes. `pack update` handles minor/patch bumps with tamper detection.
    `pack upgrade` handles major bumps with remediation bundle generation.
    Lock verification at gate time prevents drift between what was validated
    and what's running. Local path packs use the same loader and validation
    but skip git resolution. Reliability work is proved through production CLI
    assembly, canonical remote identity checks before mutation, authored-content
    hashing with an explicit legacy-lock migration, and transactional staging
    across every command that changes consumer state.

  assumptions:
    - >
      BUNDLE-004 (pack authoring) and BUNDLE-005 (pack validation) are
      implemented — pack check and pack test are available commands
    - >
      Git is available in all environments where pack add and pack update run;
      CI/airgapped environments use pack install --cache instead
    - >
      The enforcement semver model from BUNDLE-004 (adding rules = major,
      loosening = minor, false positive fixes = patch) is adopted and packs
      follow it
    - >
      Consumer config files (e.g., .golangci.yml, .eslintrc) are structured
      formats where settings can be programmatically merged and reverted

requirements:
  - id: REQ-001
    version: "1.0.0"
    text: >
      pack add must resolve org/pack-name to a git URL (github.com/org/pack-name
      by default), clone at the specified version tag, and must exit with a
      non-zero exit code and a diagnostic message identifying the failure
      (missing tag, clone failure, validation failure) if the tag does not
      exist or the clone fails (DD-1, DD-10)
  - id: REQ-002
    version: "1.0.0"
    text: >
      pack add must run pack check and pack test on the cloned pack before
      installation and must exit with a non-zero exit code and a diagnostic
      message identifying the failure (missing tag, clone failure, validation
      failure) if either validation step fails (DD-10)
  - id: REQ-003
    version: "1.0.0"
    text: >
      pack add must copy the validated pack to .backstop/packs/org/pack-name/,
      compute the content hash, and update both backstop.yml (exact version pin)
      and backstop.lock (version + hash + git ref) atomically (DD-10, DD-23)
  - id: REQ-004
    version: "1.0.0"
    text: >
      When tool_config merge encounters a conflict, pack add must exit with a
      non-zero exit code and output a diagnostic listing each conflicting
      setting, the pack's desired value, the consumer's current value, and the
      config file path. The consumer resolves conflicts manually and re-runs
      pack add. In non-interactive environments (CI), the same diagnostic and
      non-zero exit apply — no prompts (DD-9)
  - id: REQ-005
    version: "1.0.0"
    text: >
      pack add must record every tool_config setting it merges in
      .backstop/pack-config-provenance.json, mapping config file path and
      setting key to source pack name and install-time value hash (DD-16, DD-21)
  - id: REQ-006
    version: "1.0.0"
    text: >
      pack remove must read .backstop/pack-config-provenance.json, revert
      settings sourced from the target pack, and warn (rather than revert)
      any setting whose current value differs from its install-time hash (DD-11,
      DD-21)
  - id: REQ-007
    version: "1.0.0"
    text: >
      pack remove must delete the pack from .backstop/packs/, remove the pack
      entry from backstop.yml, remove the pack entry from backstop.lock, and
      remove the pack's entries from pack-config-provenance.json (DD-11)
  - id: REQ-008
    version: "1.0.0"
    text: >
      pack install must restore all packs from backstop.lock by cloning each
      at its pinned version, verifying the content hash matches the locked hash,
      and copying to .backstop/packs/ — without running pack check or pack test
      (DD-12)
  - id: REQ-009
    version: "1.0.0"
    text: >
      pack install must fail hard (non-zero exit, no partial install) when any
      pack's computed content hash does not match the hash recorded in
      backstop.lock (DD-12)
  - id: REQ-010
    version: "1.0.0"
    text: >
      pack install --cache <path> must read packs from a local directory instead
      of cloning from git, while still verifying content hashes against the
      lockfile (DD-22)
  - id: REQ-011
    version: "1.0.0"
    text: >
      pack install must not merge tool_config — config was already merged at
      pack add time and committed to the repo; install is a content-only
      restore (DD-12)
  - id: REQ-012
    version: "1.0.0"
    text: >
      pack update must resolve the latest version within the semver minor/patch
      range for the specified pack, run pack check + pack test on the new
      version before updating the version pin (if validation fails, the update
      is aborted and the previous version is retained), write the new exact
      version pin to backstop.yml, and update backstop.lock with the new
      version, hash, and git ref (DD-13, DD-23)
  - id: REQ-013
    version: "1.0.0"
    text: >
      When tamper detection finds fixture removal, severity downgrade,
      risk_class change, or rule removal, pack update must exit with a
      non-zero exit code and output a diagnostic listing each change. The
      consumer must re-run with --acknowledge flag to proceed. This applies
      in both interactive and CI environments. Tamper detection checks for:
      fixture removal, severity downgrade, risk_class change, and rule
      removal. Other content changes in minor/patch versions are accepted
      without acknowledgment (DD-4, DD-13)
  - id: REQ-014
    version: "1.0.0"
    text: >
      pack upgrade must accept an explicit major version target
      (org/pack-name@version), scan the consumer's codebase against the new
      version, generate a remediation bundle scoping all new violations, and
      baseline those violations (DD-7, DD-13)
  - id: REQ-015
    version: "1.0.0"
    text: >
      pack upgrade must run pack check and pack test on the new version
      before installing (DD-10)
  - id: REQ-016
    version: "1.0.0"
    text: >
      pack upgrade must update tool_config for the new version (the new
      version may have new or changed config requirements), with the same
      conflict escalation as pack add (REQ-004) (DD-9)
  - id: REQ-017
    version: "1.0.0"
    text: >
      pack upgrade must update backstop.yml with the new exact version pin
      and backstop.lock with the new content hash (DD-23)
  - id: REQ-018
    version: "1.0.0"
    text: >
      pack upgrade must generate a remediation bundle scoping all new
      violations. If remediation bundle generation fails, pack upgrade must
      roll back by restoring the previous version (DD-7)
  - id: REQ-019
    version: "1.0.0"
    text: >
      pack list must display installed pack name, version, lock status
      (locked/stale/missing), archetype, rule count, and scaffold count in
      a human-readable table by default, and as structured JSON with
      --json (DD-14)
  - id: REQ-020
    version: "1.1.0"
    text: >
      backstop.lock must be YAML with sorted keys, containing per-pack entries
      keyed by the manifest pack name, with version, git ref (null for local
      packs), content hash, source type (git or local), install date, and — for
      git packs — the requested source repository coordinate recorded verbatim
      as a field distinct from the pack name (DD-19, DD-31)
    versions:
      - version: "1.0.0"
        text: >
          backstop.lock must be YAML with sorted keys, containing per-pack entries
          with name, version, git ref (null for local packs), content hash, source
          type (git or local), and install date (DD-19)
      - version: "1.1.0"
        text: >
          backstop.lock must be YAML with sorted keys, containing per-pack entries
          keyed by the manifest pack name, with version, git ref (null for local
          packs), content hash, source type (git or local), install date, and — for
          git packs — the requested source repository coordinate recorded verbatim
          as a field distinct from the pack name (DD-19, DD-31)
  - id: REQ-021
    version: "1.1.0"
    text: >
      Content hash must be SHA-256 computed from a sorted manifest of
      relative-path:SHA-256-file-hash pairs covering every authored file in the
      pack directory. Repository-control metadata at the pack root, including
      a .git file, directory, symlink, or subtree, must not be copied into an
      installed pack or contribute to its content hash. Other authored dotfiles
      and dot-directories remain content (DD-20, DD-24)
    versions:
      - version: "1.0.0"
        text: >
          Content hash must be SHA-256 computed from a sorted manifest of
          relative-path:SHA-256-file-hash pairs covering every file in the pack
          directory (DD-20)
      - version: "1.1.0"
        text: >
          Content hash must be SHA-256 computed from a sorted manifest of
          relative-path:SHA-256-file-hash pairs covering every authored file in the
          pack directory. Repository-control metadata at the pack root, including
          a .git file, directory, symlink, or subtree, must not be copied into an
          installed pack or contribute to its content hash. Other authored dotfiles
          and dot-directories remain content (DD-20, DD-24)
  - id: REQ-022
    version: "1.0.0"
    text: >
      backstop gate must verify that every pack in backstop.lock is present
      in .backstop/packs/ with a matching content hash; hash mismatch, missing
      pack, or extra unlocked pack must each produce a gate failure with a
      specific diagnostic message (DD-15)
  - id: REQ-023
    version: "1.0.0"
    text: >
      backstop gate must fail with a diagnostic error if backstop.lock is
      absent when packs are declared in backstop.yml (DD-3)
  - id: REQ-024
    version: "1.0.0"
    text: >
      pack add must ensure .backstop/packs/ is listed in the project's
      .gitignore. If .gitignore does not exist, pack add creates it. If
      .backstop/packs/ is not in .gitignore, pack add appends it. Pack
      contents must never be committed to the consumer's repo (DD-18)
  - id: REQ-025
    version: "1.0.0"
    text: >
      Local path packs (referenced via path: in backstop.yml) must use the
      same pack check and pack test validation as git packs, appear in
      backstop.lock with a content hash but no git ref, and be verified at
      gate time by content hash (DD-17)
  - id: REQ-026
    version: "1.0.0"
    text: >
      pack update must be a no-op for local path packs since they update when
      their source files change; pack install must verify local pack content
      hash against the lockfile (DD-17)
  - id: REQ-027
    version: "1.0.0"
    text: >
      Packs cannot declare dependencies on other packs. There is no
      pack-to-pack dependency mechanism. Every pack in a project must be
      explicitly added by the consumer via pack add. This prevents transitive
      trust chains (DD-2)
  - id: REQ-028
    version: "1.0.0"
    text: >
      backstop.yml must store exact version pins for every pack with no range
      syntax; pack update resolves the latest compatible version internally and
      writes the resolved exact pin (DD-6, DD-23)
  - id: REQ-029
    version: "1.0.0"
    text: >
      .backstop/pack-config-provenance.json must be committed to the repo
      alongside backstop.yml and backstop.lock, tracking config file path,
      setting key/path, source pack name, and install-time value hash for
      every pack-contributed setting (DD-16, DD-21)
  - id: REQ-030
    version: "1.0.0"
    text: >
      pack update and pack upgrade must use the enforcement semver model
      (defined in BUNDLE-004) to determine behavior: pack update auto-applies
      patch and minor versions, pack upgrade handles major versions (DD-8)
  - id: REQ-031
    version: "1.0.0"
    text: >
      pack add and pack install must not install SDK dependencies declared in
      packs. SDK installation is the consumer's responsibility via native
      package managers. backstop tracks SDK references in the manifest and
      lockfile but does not distribute SDK code (DD-5)
  - id: REQ-032
    version: "1.0.0"
    text: >
      pack add for a pack already in backstop.yml must exit with a diagnostic
      suggesting pack update or pack upgrade instead
  - id: REQ-033
    version: "1.0.0"
    text: >
      pack remove for a pack not in backstop.yml must exit with a non-zero
      exit code and diagnostic
  - id: REQ-034
    version: "1.0.0"
    text: >
      pack install must fail with a diagnostic if backstop.lock does not
      exist
  - id: REQ-035
    version: "1.0.0"
    text: >
      pack install must not leave a partial installation in .backstop/packs/.
      If any pack clone fails, the entire install fails and .backstop/packs/
      is restored to its previous state (or left empty if it was a fresh
      install)
  - id: REQ-036
    version: "1.0.0"
    text: >
      pack update when the installed version is already the latest compatible
      version must be a no-op with an informational message
  - id: REQ-037
    version: "1.0.0"
    text: >
      pack add must accept a local filesystem path as an alternative to a git
      org/pack-name reference. When given a local path, pack add must skip git
      resolution and cloning, validate the pack in place (run pack check +
      pack test), register it in backstop.yml with a path: entry instead of a
      version: entry, and compute a content hash for backstop.lock. Local path
      packs are not cloned to .backstop/packs/ — they are loaded directly from
      their source path (DD-17)
  - id: REQ-038
    version: "1.1.0"
    text: >
      Every advertised remote lifecycle command must be constructible only with
      concrete production implementations of the dependencies it requires: add
      clones and validates an exact version; install clones exact lock pins without
      validation; update resolves compatible versions, clones, validates, and
      performs tamper checks; upgrade clones and validates an explicit major
      version and generates remediation artifacts. An incompletely assembled
      command must be a compile-time failure, not a runtime condition, so no
      command can reach execution with a missing dependency. Subprocess failures,
      missing tags, and malformed refs must return human diagnostics and structured
      JSON errors with non-zero exits; no command may panic by dereferencing a nil
      dependency (DD-25, DD-30).
    versions:
      - version: "1.0.0"
        text: >
          Every advertised remote lifecycle command must wire concrete production
          implementations for the dependencies it actually requires: add clones and
          validates an exact version; install clones exact lock pins without validation;
          update resolves compatible versions, clones, validates, and performs tamper
          checks; upgrade clones and validates an explicit major version and generates
          remediation artifacts. Missing dependencies, subprocess failures, missing
          tags, and malformed refs must return human diagnostics and structured JSON
          errors with non-zero exits; no command may panic by dereferencing a nil
          dependency (DD-25).
      - version: "1.1.0"
        text: >
          Every advertised remote lifecycle command must be constructible only with
          concrete production implementations of the dependencies it requires: add
          clones and validates an exact version; install clones exact lock pins without
          validation; update resolves compatible versions, clones, validates, and
          performs tamper checks; upgrade clones and validates an explicit major
          version and generates remediation artifacts. An incompletely assembled
          command must be a compile-time failure, not a runtime condition, so no
          command can reach execution with a missing dependency. Subprocess failures,
          missing tags, and malformed refs must return human diagnostics and structured
          JSON errors with non-zero exits; no command may panic by dereferencing a nil
          dependency (DD-25, DD-30).
  - id: REQ-039
    version: "1.1.0"
    text: >
      Before mutation, a remote lifecycle command must validate one exact effective
      version and record both identities: the requested source repository coordinate
      verbatim, without host-specific case normalization, and the manifest pack name
      as the install/runtime identity that determines install path, config key, lock
      key, and engine asset root. The manifest semantic version must equal the
      effective tag version after normalizing only an optional leading `v`. Any
      violation of that mapping must fail at the command boundary rather than create
      a pack that fails later during rule, producer, converter, or validator
      resolution (DD-26, DD-31).
    versions:
      - version: "1.0.0"
        text: >
          Before mutation, a remote lifecycle command must validate one exact effective
          version and resolve an unambiguous mapping among source repository coordinate,
          manifest pack name, install path, config key, lock identity, and manifest
          version. Any mismatch not authorized by the eventual canonical identity policy
          must fail at the command boundary rather than create a pack that fails later
          during rule, producer, converter, or validator resolution (DD-26).
      - version: "1.1.0"
        text: >
          Before mutation, a remote lifecycle command must validate one exact effective
          version and record both identities: the requested source repository coordinate
          verbatim, without host-specific case normalization, and the manifest pack name
          as the install/runtime identity that determines install path, config key, lock
          key, and engine asset root. The manifest semantic version must equal the
          effective tag version after normalizing only an optional leading `v`. Any
          violation of that mapping must fail at the command boundary rather than create
          a pack that fails later during rule, producer, converter, or validator
          resolution (DD-26, DD-31).
  - id: REQ-040
    version: "1.1.0"
    text: >
      Pack add, install, update, upgrade, remove, and relock must each define one
      transactional mutation boundary covering every surface that command may
      mutate, registered as participants with the single shared staged-filesystem
      transaction coordinator rather than implemented per command, according to
      DD-27. A failure must restore the exact previous state, including prior
      installed content and pre-existing consumer configuration; fresh operations
      must leave no partial state (DD-27, DD-29).
    versions:
      - version: "1.0.0"
        text: >
          Pack add, install, update, upgrade, remove, and relock must each define one
          transactional mutation boundary covering every surface that command may
          mutate according to DD-27. A failure must restore the exact previous state,
          including prior installed content and pre-existing consumer configuration;
          fresh operations must leave no partial state (DD-27).
      - version: "1.1.0"
        text: >
          Pack add, install, update, upgrade, remove, and relock must each define one
          transactional mutation boundary covering every surface that command may
          mutate, registered as participants with the single shared staged-filesystem
          transaction coordinator rather than implemented per command, according to
          DD-27. A failure must restore the exact previous state, including prior
          installed content and pre-existing consumer configuration; fresh operations
          must leave no partial state (DD-27, DD-29).
  - id: REQ-041
    version: "1.1.0"
    text: >
      A lock entry carrying a REQ-021@1.0.0 Git-metadata-inclusive content hash
      must fail with a typed migration-required diagnostic distinguishable from
      authored-content tampering; no command may accept a legacy hash in place.
      Local-path sources must migrate via pack relock. Git sources must migrate via
      an explicit remote migration operation that clones the exact pinned tag,
      reruns pack check and pack test, presents the algorithm transition, requires
      explicit consumer acknowledgment, and atomically writes sanitized installed
      content plus the updated lock entry. The migration must never prompt for
      acknowledgment in CI, which consumes an already-migrated committed lock
      (DD-24, DD-28).
    versions:
      - version: "1.0.0"
        text: >
          The transition from content-hash requirement REQ-021@1.0.0 to 1.1.0 must
          distinguish a recognized legacy Git-metadata-inclusive hash from actual
          authored-content tampering. The migration policy must be deterministic,
          non-interactive in CI, and must converge the lock and installed content on
          the 1.1.0 algorithm without silently accepting unrelated changes.
          Remote Git locks and local-path locks may require different explicit
          migration commands because remote legacy clone metadata is not reproducible
          on a fresh machine (DD-24; migration policy pending OQ-6).
      - version: "1.1.0"
        text: >
          A lock entry carrying a REQ-021@1.0.0 Git-metadata-inclusive content hash
          must fail with a typed migration-required diagnostic distinguishable from
          authored-content tampering; no command may accept a legacy hash in place.
          Local-path sources must migrate via pack relock. Git sources must migrate via
          an explicit remote migration operation that clones the exact pinned tag,
          reruns pack check and pack test, presents the algorithm transition, requires
          explicit consumer acknowledgment, and atomically writes sanitized installed
          content plus the updated lock entry. The migration must never prompt for
          acknowledgment in CI, which consumes an already-migrated committed lock
          (DD-24, DD-28).
  - id: REQ-042
    version: "1.1.0"
    text: >
      A hermetic end-to-end parity suite must build the production CLI, create
      temporary tagged pack repositories, and exercise remote add, install,
      update, and upgrade; gate-time lock verification; the explicit remote
      lock-migration operation (DD-28); local relock migration; and remove cleanup
      without GitHub or network access. The suite must cover success, subprocess
      failure, identity mismatch, legacy-hash migration, rollback via fault
      injection at the shared transaction coordinator, and repository-metadata
      exclusion using the same command wiring shipped to consumers
      (DD-25, DD-27, DD-29).
    versions:
      - version: "1.0.0"
        text: >
          A hermetic end-to-end parity suite must build the production CLI, create
          temporary tagged pack repositories, and exercise remote add, install,
          update, and upgrade; gate-time lock verification; the remote migration path
          selected by OQ-6; local relock migration; and remove cleanup without GitHub
          or network access. The suite must cover
          success, dependency failure, identity mismatch, legacy-hash migration,
          rollback, and repository-metadata exclusion using the same command wiring
          shipped to consumers (DD-25, DD-27).
      - version: "1.1.0"
        text: >
          A hermetic end-to-end parity suite must build the production CLI, create
          temporary tagged pack repositories, and exercise remote add, install,
          update, and upgrade; gate-time lock verification; the explicit remote
          lock-migration operation (DD-28); local relock migration; and remove cleanup
          without GitHub or network access. The suite must cover success, subprocess
          failure, identity mismatch, legacy-hash migration, rollback via fault
          injection at the shared transaction coordinator, and repository-metadata
          exclusion using the same command wiring shipped to consumers
          (DD-25, DD-27, DD-29).
---

# Pack Distribution Lifecycle

## Current Thinking

### The distribution gap

BUNDLE-004 (authoring) and BUNDLE-005 (validation) give you a pack that is
structurally correct, fixture-proven, and mechanically verified. BUNDLE-006
defined and partially delivered the lifecycle from a validated Git repository
to a consumer's running gate. Dogfooding a standalone harness pack proved that
the authored contract is not yet reliably delivered through production command
wiring: local-path flows work, while remote flows can panic, accept incoherent
identity, include repository metadata in content hashes, or leave partial state.
Version 0.7.0 reopens the bundle to close that contract-to-implementation gap.

### The distribution mental model is Homebrew

Packs are meant to be used and distributed the way brew formulas and taps work,
and that analogy is load-bearing rather than decorative:

- **The tap is where a pack lives; the pack name is what you install by.** A brew
  formula is not identified by the repository that hosts it, and neither is a
  pack. This is exactly the separation DD-31 makes formal — the repository
  coordinate is a source location, the manifest name is the identity.
- **Installs are reproducible from pinned references.** `brew install foo` on two
  machines yields the same bottle; `pack install` from a committed lock must
  yield byte-identical content, which is why the content hash must cover authored
  content only (DD-24) and why legacy hashes must migrate explicitly rather than
  be accepted in place (DD-28).
- **Remote is the default posture, not a later addition.** Local-path packs are
  the equivalent of a local formula file — supported and useful, but not the
  shape the ecosystem is designed around.

Today every backstop install is a local path, so the remote posture is
advertised rather than proven. The 0.7.0 reliability revision exists to close
that gap: without production dependency assembly, coherent identity, reproducible
hashes, and transactional mutation, "install a pack by name from its tap" is a
claim the CLI cannot keep.

### The command surface

Six commands cover the distribution lifecycle:

1. **`pack add <org/pack-name>@<version>`** — the primary entry point.
   Resolves the pack name to a git URL, clones at the specified version tag,
   runs `pack check` + `pack test` on the clone (refuses to install on
   failure), copies to `.backstop/packs/`, computes content hash, updates
   backstop.yml, updates backstop.lock, and merges tool_config into the
   consumer's repo. One command, fully validated, fully locked.

2. **`pack remove <org/pack-name>`** — reads tool_config provenance to
   identify which consumer config settings came from this pack, reverts
   those settings (or warns if manually modified), removes from
   `.backstop/packs/`, backstop.yml, and backstop.lock.

3. **`pack install`** — the CI/fresh-clone command. Reads backstop.lock,
   clones each pack at its pinned version+hash, verifies content hash
   matches, copies to `.backstop/packs/`. Does NOT run `pack check` or
   `pack test` (already validated at `pack add` time — hash verification is
   sufficient). Does NOT merge tool_config (already merged at `pack add`
   time and committed). Fast, deterministic, reproducible. Fails hard on
   hash mismatch.

4. **`pack update`** — pulls latest within semver minor/patch range.
   Auto-applies. No remediation bundle. Tamper detection runs (warn on
   fixture removal / severity downgrade per DD-5).

5. **`pack upgrade <org/pack-name>@<version>`** — major version bump.
   Generates a remediation bundle scoping all new violations. New violations
   baselined. The upgrade is a first-class work item in the backstop
   lifecycle.

6. **`pack list`** — shows installed packs with version, lock status
   (locked/stale/missing), archetype, rule count, scaffold count. JSON
   output for agent consumption.

### The lockfile

`backstop.lock` is mandatory from day one. Every installed pack is pinned
by version + content hash. The lockfile is committed to the repo. `pack
add` writes it. `pack update` / `pack upgrade` update it. `backstop gate`
verifies installed packs match it. Hash mismatch = gate failure. Missing
pack = gate failure. The lockfile is the single source of truth for "what
packs should be present and at what versions."

### Local path packs

backstop.yml can reference packs by local path
(`packs: [{path: ./packs/internal-standards}]`). These are loaded directly
from the filesystem, not cloned. `pack check` + `pack test` still run at
add time. They appear in backstop.lock with a content hash but no git ref.
Updates are immediate — the pack is local, no `pack update` needed. Useful
for in-repo packs that evolve with the codebase. REQ-037 and DD-17 remain the
authoritative direct-source contract. The committed implementation currently
materializes local packs into `.backstop/packs/`; that is implementation drift,
not a silent contract revision, and the content-identity/migration repair seed
must either restore direct loading or return an explicit bundle revision for
human approval.

### tool_config provenance

When `pack add` merges settings into consumer config files, it records the
mapping so `pack remove` can revert them. Each entry maps: config file path
to setting key to source pack. If a setting has been manually modified since
install, `pack remove` warns instead of silently reverting. Without
provenance tracking, `pack remove` can only delete the pack directory —
it cannot undo the config changes, leaving ghost settings from removed
packs.

### Lock verification at gate time

`backstop gate` reads backstop.lock, computes content hash of each
installed pack in `.backstop/packs/`, and compares. Mismatch = gate failure
with a specific error ("pack acme/go-http-standards installed content does
not match lock — run `pack install` to restore"). Missing pack = gate
failure. This closes the loop: `pack add` validates and locks, `pack
install` restores from lock, `backstop gate` verifies the lock holds.

### Reliability evidence from harness dogfooding (0.7.0)

The `backstop-harness` consumer published
`backstop-ai/backstop-harness-toolchain-pack` and exercised the production CLI.
The following evidence is the discovery source for the reliability revision:

- `backstop pack add backstop-ai/backstop-harness-toolchain-pack` reached
  `distribution.Add` with no production Git cloner and panicked on a nil
  dependency. This maps to REQ-038 and the production dependency assembly seed.
- Installing the first tagged release under the repository coordinate while its
  manifest declared `backstop/harness-toolchain` reported add success, but gate
  dispatch later resolved converter scripts beneath the manifest-name path and
  failed with `missing convert script`. This maps to REQ-039 and the remote
  reference/manifest identity seed. The pack corrected its identity in v0.1.1;
  core still needs fail-closed mapping semantics.
- Adding the Git-backed pack by local path under committed core changed the
  content hash from the authored v0.1.1 tree hash and copied root `.git` metadata
  into materialized content. A clean non-Git export produced the authored hash.
  This maps to REQ-021@1.1.0, REQ-041, and the content identity/migration seed.
- Review of committed add/install/update/upgrade/remove code found mutation
  surfaces without one complete rollback boundary and additional production
  commands with incomplete dependency assembly. This maps to REQ-040, REQ-042,
  the transaction seed, and the lifecycle parity seed. These are reviewed code
  findings, not consumer-owned implementation changes.

No production core fix was retained from the consumer session. The harness is
temporarily locked to a clean local non-Git export so it can use the pack without
claiming remote distribution is delivered.

## Draft Requirements

Requirements REQ-001 through REQ-042 are defined in the frontmatter
`requirements` block. They cover:

- **Pack add** (REQ-001 through REQ-005, REQ-024, REQ-031, REQ-032, REQ-037):
  git resolution with diagnostic exit on failure, validation gate, install +
  lock + manifest update, tool_config merge with conflict escalation (non-zero
  exit, diagnostic listing), provenance recording, gitignore management, SDK
  non-distribution, already-installed guard, local path pack support.
- **Pack remove** (REQ-006, REQ-007, REQ-033): provenance-based config
  revert with manual-modification detection, full cleanup across packs dir /
  yml / lock / provenance, not-installed guard.
- **Pack install** (REQ-008 through REQ-011, REQ-034, REQ-035): lockfile
  restore, hard failure on hash mismatch, --cache flag for offline/airgapped,
  no re-validation or config merge, missing lockfile guard, atomic install
  (no partial state).
- **Pack update** (REQ-012, REQ-013, REQ-036): semver minor/patch
  resolution, exact pin write, tamper detection with --acknowledge flag
  escalation, no-op when already latest.
- **Pack upgrade** (REQ-014 through REQ-018): major version target, validate
  before install, tool_config update with conflict escalation, version pin
  and lockfile update, remediation bundle generation with rollback on
  failure.
- **Pack list** (REQ-019): human table + JSON output.
- **backstop.lock** (REQ-020@1.1.0, REQ-021@1.1.0): YAML sorted keys, SHA-256
  sorted manifest hash over authored content, entries keyed by manifest name with
  the source coordinate recorded verbatim.
- **Gate verification** (REQ-022, REQ-023): content hash comparison,
  diagnostic messages for mismatch/missing/extra, mandatory lockfile when
  packs declared.
- **.backstop/packs/ lifecycle** (REQ-024): gitignore ownership by pack add,
  ephemeral, regenerated from lock.
- **Local path packs** (REQ-025, REQ-026): same validation, no git ref,
  update is no-op.
- **Trust model** (REQ-027): no pack-to-pack dependencies, no transitive
  trust chains.
- **Version semantics** (REQ-028, REQ-030): exact pins, enforcement semver
  model (references BUNDLE-004).
- **Production remote execution** (REQ-038@1.1.0, REQ-039@1.1.0): command
  dependencies that cannot be omitted at compile time, and source coordinate
  versus manifest identity with both recorded in the lock.
- **Transactional lifecycle** (REQ-040@1.1.0): one shared transaction
  coordinator with per-command participants giving exact rollback across
  installed content, config, provenance, manifests, lock, and owned gitignore
  entries.
- **Hash migration** (REQ-021@1.1.0, REQ-041@1.1.0): authored-content boundary,
  root repository-metadata exclusion, and explicit re-validating legacy-lock
  migration that never accepts a legacy hash in place.
- **Remote lifecycle parity** (REQ-042): hermetic production-CLI coverage of
  the exact remote, gate, migration, and cleanup command matrix.
- **Provenance file** (REQ-029): committed, structured tracking.
- **SDK non-distribution** (REQ-031): packs track SDK references but do not
  distribute SDK code.

## Draft Design Decisions

### Extracted from BUNDLE-001 (renumbered for this bundle)

- **DD-1:** Distribution model is git-native ("homebrew taps"), not
  npm-style. `backstop pack add <org/pack-name>@<version>` resolves to a
  git URL (github.com/org/pack-name by default), clones, validates, caches,
  and pins. No central registry required. A lightweight curated catalog
  exists for discovery but is metadata-only.
  [Source: BUNDLE-001 DD-2]

- **DD-2:** No transitive trust. If pack A references pack B, the consumer
  must add B explicitly. The dependency graph is finite, visible, and
  auditable. Designs out the multi-layer dependency attack at the model
  level.
  [Source: BUNDLE-001 DD-8]

- **DD-3:** `backstop.lock` is mandatory from v1. Content-hash pinning of
  every pack. `pack add` writes it; `pack update` diffs it; `gate` refuses
  to run on mismatch.
  [Source: BUNDLE-001 DD-9]

- **DD-4:** `pack update` performs tamper detection. Fixture removal and
  severity downgrade are red-flag events requiring explicit consumer
  acknowledgment. These cannot be auto-applied silently.
  [Source: BUNDLE-001 DD-10]

- **DD-5:** Consumers never install from backstop. Install paths for SDKs
  are always native — `go get`, `npm install`, `pip install`. Backstop is
  not a distribution endpoint for SDK code. For packs (rules, config,
  fixtures), git is the distribution channel.
  [Source: BUNDLE-001 DD-23]

- **DD-6:** backstop.yml declares ONE version per pack — the current
  enforced version. All code is validated against the current version.
  Spec-pinned versions are provenance/audit, not runtime enforcement.
  [Source: BUNDLE-004 DD-41]

- **DD-7:** Pack version upgrades auto-generate a remediation bundle.
  `backstop pack upgrade <pack>@<version>` scans the codebase against the
  new version, surfaces all new violations, and creates a bundle scoping
  the remediation work.
  [Source: BUNDLE-004 DD-42]

- **DD-8:** Enforcement pack semver model. Adding rules is a major version
  bump. Loosening rules is minor. Fixing false positives is patch.
  patch/minor auto-upgrade via `pack update`; major generates a remediation
  bundle via `pack upgrade`.
  [Source: BUNDLE-004 DD-43]

- **DD-9:** `pack add` merges tool_config additively and escalates on
  conflict. Additive changes (enabling a linter, adding a rule to
  `.golangci.yml`) are auto-merged. Conflicting changes (pack wants
  `line-length: 120`, consumer has `line-length: 100`) are escalated to
  the consumer with a diff and explanation.
  [Source: BUNDLE-004 DD-47]

### New DDs (not in any existing bundle)

- **DD-10:** `pack add` mechanics — step-by-step installation pipeline.
  (1) Resolve org/pack-name to git URL (github.com/org/pack-name by
  default, configurable via tap registry). (2) Clone at specified version
  tag. (3) Run `pack check` + `pack test` on the cloned pack — refuse to
  install if either fails. (4) Copy to `.backstop/packs/org/pack-name/`.
  (5) Compute content hash of the installed pack directory. (6) Update
  backstop.yml packs list with name, version, and source. (7) Update
  backstop.lock with version + hash + git ref. (8) Merge tool_config into
  consumer's repo (escalate on conflict per DD-9). Rationale: a single
  command that leaves no manual steps. Every pack in the system has been
  validated before it can affect the consumer.

- **DD-11:** `pack remove` mechanics — provenance-based config revert.
  (1) Read tool_config provenance to identify which consumer config settings
  came from this pack. (2) Revert those settings — or warn if they've been
  manually modified since install (hash of the value at install time vs
  current value). (3) Remove from `.backstop/packs/`. (4) Remove from
  backstop.yml packs list. (5) Remove from backstop.lock. Rationale:
  without provenance-based revert, removing a pack leaves ghost config
  settings. The manual-modification check prevents data loss when the
  consumer has intentionally tuned a pack-provided setting.

- **DD-12:** `pack install` is the CI fast path — lockfile restore without
  re-validation. Reads backstop.lock, clones each pack at its pinned
  version+hash, verifies content hash matches, copies to `.backstop/packs/`.
  Does NOT run `pack check` or `pack test` (already validated at `pack add`
  time; hash verification proves the content is identical). Does NOT merge
  tool_config (already merged at `pack add` time and committed to the repo).
  Fails hard on hash mismatch — no fallback, no retry, no "install anyway."
  Rationale: CI needs speed and determinism. Re-validating on every clone
  wastes minutes and adds failure modes. The hash is the proof.

- **DD-13:** `pack update` vs `pack upgrade` — semver-driven behavior
  split. `pack update` resolves the latest version within the semver
  minor/patch range, auto-applies, runs tamper detection (DD-4), and
  updates the lockfile. No remediation bundle — minor/patch changes are
  non-breaking by the semver contract (DD-8). `pack upgrade
  <pack>@<version>` handles major version bumps: scans the codebase against
  the new version, generates a remediation bundle (DD-7) scoping all new
  violations, baselines them, and updates backstop.yml + backstop.lock.
  Rationale: minor updates should be frictionless. Major upgrades are
  deliberate work items that flow through the backstop lifecycle.

- **DD-14:** `pack list` output — human and machine formats. Shows installed
  pack name, version, lock status (locked/stale/missing), archetype
  (enforcement/code), rule count, scaffold count. Default output is a
  human-readable table. `--json` flag produces structured JSON for agent
  consumption per ADR-0001. Rationale: `pack list` is the diagnostic
  command — both humans and agents need to understand the current pack
  state.

- **DD-15:** Lock verification at gate time — content hash comparison.
  `backstop gate` reads backstop.lock, computes content hash of each
  installed pack in `.backstop/packs/`, and compares against the locked
  hash. Mismatch = gate failure with specific error identifying the pack
  and the action to fix it ("run `pack install` to restore"). Missing pack
  = gate failure. Extra packs not in the lockfile = gate failure. This
  prevents: (a) drift from hand-editing `.backstop/packs/`, (b) stale
  installs after lockfile updates, (c) packs added outside backstop's
  command surface. Rationale: the lockfile is meaningless if nothing
  enforces it.

- **DD-16:** tool_config provenance tracking. When `pack add` merges
  settings into consumer config files, it records the mapping in
  `.backstop/pack-config-provenance.json`. Each entry maps: config file
  path, setting key/path, source pack name, value at install time (hashed).
  `pack remove` reads this to know what to revert. If a setting's current
  value differs from its install-time hash, `pack remove` warns: "setting
  X in .golangci.yml was modified since pack install — skipping revert.
  Review manually." The provenance file is committed to the repo alongside
  backstop.yml. Rationale: a separate structured file is more reliable
  than inline comments (which are fragile across tool config formats) and
  enables programmatic revert without parsing comments.

- **DD-17:** Local path packs — same validation, different resolution.
  backstop.yml can reference packs by local path
  (`packs: [{path: ./packs/internal-standards}]`). These skip git
  resolution — loaded directly from the filesystem. `pack check` + `pack
  test` still run at add time. They appear in backstop.lock with a content
  hash but no git ref. `pack update` is a no-op for local packs (they
  update when the files change). `pack install` verifies the content hash
  of local packs against the lockfile. Rationale: in-repo packs that evolve
  with the codebase are a first-class use case. They should get the same
  validation and lock verification without the git overhead.

- **DD-18:** `.backstop/packs/` is gitignored and regenerated from the
  lockfile. The directory is ephemeral — `pack install` recreates it from
  scratch. Committing pack contents to the consumer's repo would bloat the
  repo and create merge conflicts on update. The lockfile + git source is
  the durable record; `.backstop/packs/` is a local cache. Rationale:
  follows the node_modules / vendor convention — the lockfile is committed,
  the installed artifacts are not.

### DDs from resolved OQs

- **DD-19:** backstop.lock is YAML with sorted keys for deterministic diffs.
  Consistent with backstop.yml and pack.yml — the whole backstop ecosystem
  uses one format. Sorted keys minimize merge conflicts and make lockfile
  diffs reviewable. Rationale: ecosystem consistency outweighs the
  machine-first argument for JSON; sorted output addresses the merge
  conflict concern that is YAML's main weakness for lockfiles.
  [Resolved from OQ-1]

- **DD-20:** Content hash is SHA-256 of a sorted manifest of path:hash
  pairs. Each file in the pack directory is individually hashed (SHA-256),
  paths are sorted lexicographically, the path:hash pairs are combined into
  a single SHA-256 hash. The intermediate manifest is useful for debugging
  ("which file changed?") and can be emitted by `pack list --verbose`.
  Rationale: same approach as git's tree hashing. Fast (parallel file
  hashing), deterministic (sorted paths), comprehensive (every file
  contributes). Tar-based hashing is fragile across platforms; pack.yml-only
  hashing misses rule and fixture changes.
  **Revised by DD-24 / REQ-021@1.1.0:** "every file" means every authored
  content file; root repository-control metadata is not pack content.
  [Resolved from OQ-2]

- **DD-21:** tool_config provenance tracked in
  `.backstop/pack-config-provenance.json`. Maps config file path to setting
  key to source pack name. Machine-parseable, committed to the repo
  alongside backstop.yml. `pack remove` reads this file to identify which
  settings to revert. Inline comments were rejected — they are fragile
  (deleted by formatters, hand-edits) and not supported by all config
  formats (JSON has no comments). Rationale: a separate structured file
  enables programmatic revert without parsing comments and works uniformly
  across all config file formats.
  [Resolved from OQ-3]

- **DD-22:** `pack install --cache <path>` for offline/airgapped
  environments. Reads packs from a local directory instead of cloning from
  git. Hash verification still runs against the lockfile — same trust model
  regardless of source. CI environments pre-populate the cache however they
  want (artifact store, shared volume, pre-built image). Default behavior
  without the flag is clone from git. Local caching is also built into
  `pack install` by default for performance — repeated CI runs don't
  re-clone every time. Rationale: a cache path flag is the simplest
  mechanism that decouples backstop from cache population strategy.
  [Resolved from OQ-4]

- **DD-23:** Exact version pins in backstop.yml. No range syntax. `pack
  update` resolves the latest compatible version (within semver minor/patch
  per DD-8) and writes the new exact pin to backstop.yml, then updates
  backstop.lock. The consumer always sees exactly what version they're on.
  backstop.yml and the lockfile always agree on the version — no resolution
  step at install time. Rationale: ranges add complexity for minimal gain
  when `pack update` exists as the explicit upgrade path. Exact pins
  eliminate "what version am I running?" ambiguity.
  [Resolved from OQ-5]

### Reliability revision decisions (0.7.0)

- **DD-24:** Pack content identity covers authored pack content, not the
  source repository's control metadata. The root `.git` entry and anything it
  owns are excluded whether `.git` is a directory, worktree/submodule pointer
  file, or symlink. This exclusion is deliberately narrow: `.github`, other
  dotfiles, and nested authored fixtures remain content. Rationale: lock hashes
  must be reproducible across clones and must not copy repository objects,
  refs, logs, or local configuration into consumer caches.

- **DD-25:** Production command wiring is part of the tested contract. Remote
  lifecycle tests must execute the same concrete dependencies assembled by
  Cobra commands, not only distribution functions with injected mocks.
  Hermetic tests redirect Git URLs to temporary local repositories so the
  production path remains network-free and deterministic.

- **DD-26:** Remote identity is checked before mutation. The effective version
  may come from the reference or an explicit `--version` override, but it must
  resolve to one exact tag. Source coordinate, manifest name, install path,
  config key, lock identity, and manifest version must resolve coherently under
  the policy in DD-31 before the pack can affect consumer state.

- **DD-27:** Transaction participants are command-specific. Add owns installed
  content when the selected source model materializes it, tool config,
  provenance, backstop.yml, backstop.lock, and its
  gitignore contribution. Install owns installed content and, only when an
  approved hash migration occurs, the corresponding lock entry. Update owns
  installed content, affected tool config/provenance, backstop.yml, and lock.
  Upgrade owns those surfaces plus remediation artifacts and baseline changes.
  Remove owns installed content, reverted tool config, provenance, backstop.yml,
  and lock. Relock owns lock entries only. Each command atomically restores all
  participants it actually owns; no command is required to snapshot surfaces it
  does not mutate. Participants are registered with the shared coordinator in
  DD-29 rather than implemented per command.

### DDs from resolved OQs (0.8.0)

- **DD-28:** Legacy hash migration is explicit, re-validating, and never
  in-place. A lock entry carrying a REQ-021@1.0.0 (Git-metadata-inclusive) hash
  fails with a typed migration-required diagnostic that is distinguishable from
  authored-content tampering. Local-path sources migrate via `pack relock`. Git
  sources migrate via a distinct explicit remote migration operation that clones
  the exact pinned tag, reruns `pack check` + `pack test`, presents the
  old-algorithm/new-algorithm transition, requires explicit consumer
  acknowledgment, and atomically writes sanitized installed content plus the
  updated lock entry in one transaction (DD-29). No command silently accepts a
  legacy hash, and CI never acknowledges interactively — it consumes an
  already-migrated committed lock. Rationale: a fresh clone cannot reproduce
  clone-local Git state, so a dual-read that "proves" the old hash would only
  ever succeed on a machine that happens to hold the original tree; trust must
  never be inferred from unreproducible repository metadata. Revalidation is
  required precisely because the legacy hash cannot authenticate the fresh
  clone. A later in-place convenience path may be added only with evidence that
  it cannot mask authored changes.
  [Resolved from OQ-6]

- **DD-29:** One shared staged-filesystem transaction coordinator, with
  command-specific participants. Distribution exposes a single coordinator plus
  staging/snapshot primitives; add, install, update, upgrade, remove, and relock
  register the participants they own per DD-27 rather than implementing private
  rollback. A command's mutation surface becomes an explicit registration, not an
  implicit consequence of which branch executed. Rationale: rollback is the
  property most likely to silently regress when a new mutation surface is added;
  five private implementations means five places to forget it. A single
  coordinator also gives the hermetic parity suite (REQ-042) one seam for
  fault injection, so rollback is proven by construction rather than per command.
  [Resolved from OQ-7]

- **DD-30:** Lifecycle command dependencies are structurally mandatory —
  constructors make incomplete options unrepresentable. A command cannot be
  assembled without its real Git, version-resolution, and validator
  dependencies; omitting one is a compile-time failure rather than a runtime
  diagnostic or a silently-skipped step. Distribution provides no internal
  defaults, so a test double is never mistakable for production wiring.
  Fail-closed nil validation (OQ-8 option (a)) is permitted only as an
  incremental step while the constructors land, never as the end state.
  Rationale: this codebase is extended by agents working at speed from
  artifacts, and the discovery-source defect (nil GitCloner panic, REQ-038) was
  exactly a case where the wiring was optional and nobody remembered it.
  Structural impossibility beats remembered diligence.
  [Resolved from OQ-8]

- **DD-31:** Source coordinate and pack identity are separate concepts, both
  recorded in the lock. The repository coordinate (`org/repository`, resolved
  URL, tag) is the source coordinate and is recorded verbatim — no host-specific
  case normalization, because case-insensitivity is a GitHub property and packs
  may be hosted anywhere. The manifest `name` is the install/runtime identity: it
  determines the install path under `.backstop/packs/`, the config key in
  backstop.yml, the lock key, and the engine asset root used to resolve rules,
  producers, converters, and validators. Manifest semantic version must equal the
  effective tag version after normalizing only an optional leading `v`, checked
  fail-closed before any mutation (DD-26). Rationale: the harness pack
  legitimately named itself `backstop/harness-toolchain` while living at
  `backstop-ai/backstop-harness-toolchain-pack`; the defect was not the
  divergence but that the install path was built from the coordinate while gate
  dispatch resolved assets under the manifest name. Requiring equality would
  outlaw a reasonable convention; an alias field would add a third name to keep
  coherent.
  [Resolved from OQ-9]

## Open Questions

- **OQ-1: Lock file format.** [RESOLVED] Exact schema for backstop.lock. YAML (matches
  backstop.yml)? JSON (machine-first per ADR-0001)? TOML? What fields per
  entry: name, version, git ref, content hash, install date, source type
  (git/local)? How does it handle local path packs differently from git
  packs (no git ref, but still a content hash)? **Options:** (a) YAML — consistent
  with backstop.yml, human-readable, but merge conflicts are painful;
  (b) JSON — machine-first per ADR-0001, but less human-friendly;
  (c) YAML with sorted keys and deterministic output to minimize merge
  conflicts. **Lean:** (c) — YAML for consistency with backstop.yml,
  sorted keys for deterministic diffs, but open to JSON if the merge
  conflict argument is strong enough.
  **Resolution:** YAML with sorted keys for deterministic diffs. Consistent
  with backstop.yml and pack.yml — the whole backstop ecosystem uses one
  format. See DD-19.

- **OQ-2: Content hash algorithm.** [RESOLVED] SHA-256 of what exactly? The entire
  pack directory tree? Just `pack.yml`? A manifest of file paths + hashes?
  Needs to be: fast (CI can't spend seconds hashing), deterministic (same
  content = same hash regardless of filesystem metadata), and comprehensive
  (catch any file change). **Options:** (a) SHA-256 of tar archive of the
  pack directory (deterministic ordering); (b) SHA-256 of a sorted manifest
  of relative-path + file-hash pairs (like git tree hashes);
  (c) SHA-256 of pack.yml only (fast but misses rule/fixture changes).
  **Lean:** (b) — sorted manifest of path+hash pairs. Fast (hash each file
  independently, combine), deterministic (sorted paths), comprehensive
  (every file contributes). Similar to how git computes tree hashes.
  **Resolution:** SHA-256 of a sorted manifest of path:hash pairs. Each
  file in the pack directory is individually hashed, paths are sorted, the
  pairs are combined into a single hash. Catches any file change. The
  intermediate manifest is useful for debugging ("which file changed?").
  Same approach as git's tree hashing. See DD-20.

- **OQ-3: tool_config provenance storage.** [RESOLVED] DD-16 proposes
  `.backstop/pack-config-provenance.json` but alternatives exist.
  **Options:** (a) Separate JSON file — clean, programmatic, but another
  artifact to manage; (b) Inline comments in config files (`# added by
  acme/go-http-standards`) — human-readable but fragile across formats,
  some config files don't support comments (JSON); (c) Section in
  backstop.lock itself — keeps provenance with the lock, but conflates
  two concerns. **Lean:** (a) — separate file. Config formats are too
  varied for inline comments, and mixing provenance into the lockfile
  makes the lockfile harder to reason about independently.
  **Resolution:** Separate file at `.backstop/pack-config-provenance.json`.
  Maps config file path to setting key to source pack name.
  Machine-parseable, doesn't pollute consumer's config files with
  backstop-specific comments. Inline comments are fragile (deleted by
  formatters, hand-edits). `pack remove` reads this file to know what to
  revert. See DD-21.

- **OQ-4: Offline / airgapped environments.** [RESOLVED] Packs are git clones. CI
  environments without external git access cannot `pack install` from
  remote repos. **Options:** (a) Vendor packs into the repo (defeats
  gitignore, bloats repo); (b) Pre-populate `.backstop/packs/` in the CI
  image (works but couples CI image to pack versions); (c) `pack install
  --from-cache <path>` reads from a local mirror/cache directory instead
  of cloning; (d) `pack vendor` command that copies pack sources into a
  committed directory for airgapped use. **Lean:** (c) — a cache path
  flag is the simplest mechanism. The consumer or CI setup populates the
  cache however they want (pre-clone, artifact download, shared volume).
  Backstop doesn't need to know how the cache was populated.
  **Resolution:** `pack install --cache <path>` flag reads packs from a
  local directory instead of cloning from git. Hash verification still runs
  against the lockfile — same trust model regardless of source. CI
  environments pre-populate the cache however they want (artifact store,
  shared volume, pre-built image). Default behavior without the flag is
  clone from git. Local caching is also built into `pack install` by
  default for performance — repeated CI runs don't re-clone every time.
  See DD-22.

- **OQ-5: Pack pinning granularity.** [RESOLVED] backstop.yml says `version: "1.2.0"`
  — is this an exact pin or a range? If exact, `pack update` has nothing to
  resolve. If range (`^1.2.0`), `pack update` resolves to latest matching.
  **Options:** (a) Exact pin — backstop.yml always shows the precise
  installed version, `pack update` bumps the pin to latest patch/minor;
  (b) Range (`^1.2.0`) — backstop.yml declares the range, backstop.lock
  records the resolved version; (c) Exact pin in backstop.yml, range
  semantics live in the `pack update` command (it knows minor/patch are
  safe to auto-bump). **Lean:** (a) — exact pin. The consumer always
  sees exactly what version they're on. `pack update` changes the pin.
  No ambiguity about "what version am I running?" Ranges add complexity
  for minimal gain when `pack update` exists.
  **Resolution:** Exact pin in backstop.yml — always. No range syntax, no
  resolution ambiguity. `pack update` does the semver math internally
  (finds latest compatible patch/minor version), then writes the new exact
  version pin to backstop.yml and updates backstop.lock. The consumer
  always sees exactly what version they're on. backstop.yml and the
  lockfile always agree. See DD-23.

- **OQ-6: Legacy hash migration.** [RESOLVED] REQ-021@1.0.0 included root `.git`
  metadata whenever the source or installed pack was itself a repository.
  REQ-021@1.1.0 excludes that metadata. How should existing locks migrate
  without conflating a recognized legacy hash with authored-content tampering?
  An in-place consumer may still possess the exact legacy installed tree, but a
  fresh clone generally cannot reproduce clone-local refs, logs, config, object
  layout, or worktree pointers. **Options:** (a) in-place dual-read/current-write
  when the exact legacy tree proves the old hash, with a typed migration-required
  result on fresh clones; (b) always fail with a typed migration diagnostic,
  using local `pack relock` for local sources and a new explicit remote migration
  operation that clones the exact pinned tag, reruns pack check/test because the
  old hash cannot authenticate a fresh clone, presents the old/new algorithm
  transition, requires explicit acknowledgment, and atomically writes sanitized
  content plus lock; (c) treat every old hash as a normal mismatch. **Lean:** (b)
  for the first safe release; it is deterministic across fresh clones and never
  asks CI to infer trust from unreproducible repository metadata. CI must receive
  an already-migrated committed lock rather than acknowledge interactively. A
  later in-place convenience path may be added only with evidence that it cannot
  mask authored changes.
  **Resolution:** (b) — always fail on a legacy hash with a typed migration
  diagnostic; never accept a legacy hash in place. Local-path sources migrate
  via `pack relock`. Git sources migrate via a new explicit remote migration
  operation that clones the exact pinned tag, reruns `pack check` + `pack test`
  (the legacy hash cannot authenticate a fresh clone), presents the algorithm
  transition, requires explicit acknowledgment, and atomically writes sanitized
  content plus the updated lock entry. CI always receives an already-migrated
  committed lock and is never asked to acknowledge interactively. Rationale:
  trust is never inferred from unreproducible repository metadata — a fresh
  clone cannot reproduce clone-local refs, logs, config, object layout, or
  worktree pointers, so any in-place dual-read would authenticate content on a
  machine-specific accident. Deterministic failure plus an explicit,
  re-validating migration keeps "the hash is the proof" (DD-12) true through the
  algorithm transition. See DD-28.

- **OQ-7: Transaction coordinator boundary.** [RESOLVED] Current add/install/update/
  upgrade paths mutate different combinations of installed content, tool
  config, provenance, manifests, lock, and gitignore. Should each command own
  a private snapshot/rollback implementation, or should distribution expose
  one reusable local transaction coordinator? **Options:** (a) shared staged
  filesystem transaction with command-specific participants; (b) command-local
  snapshots using shared primitives; (c) narrow rollback only around newly
  introduced validation failures. **Lean:** (a) or (b); (c) does not satisfy
  existing atomicity promises and preserves known partial-state failures.
  **Resolution:** (a) — one shared staged-filesystem transaction coordinator in
  distribution, with command-specific participants registered per DD-27, exposed
  as shared primitives that add, install, update, upgrade, remove, and relock all
  consume. Rationale: rollback correctness is the hardest property to get right
  and the easiest to regress; five private implementations means five places for
  a newly added mutation surface to be forgotten. One coordinator makes "which
  surfaces does this command own?" an explicit registration rather than an
  implicit consequence of the code path taken, and it gives the parity suite
  (REQ-042) a single seam to fault-inject against. See DD-29.

- **OQ-8: Production dependency assembly.** [RESOLVED] Should Git/version dependencies be
  mandatory explicit fields supplied by every command, or should distribution
  provide safe production defaults when options omit them? **Options:** (a)
  explicit command wiring plus fail-closed nil validation in distribution;
  (b) defaults inside distribution; (c) constructors that make incomplete
  options unrepresentable. **Lean:** (c), with (a) as the incremental path.
  **Resolution:** (c) — constructors make incomplete options unrepresentable. A
  lifecycle command cannot be assembled without its real Git, version-resolution,
  and validator dependencies; a missing dependency is a compile-time failure, not
  a runtime diagnostic. Option (a)'s fail-closed nil validation is acceptable
  only as an incremental path while the restructuring lands, not as the end
  state. Rationale (user-endorsed): this codebase is extended by agents working
  at speed from artifacts, so structural impossibility beats remembered
  diligence — the nil-cloner panic (REQ-038's discovery source) was exactly a
  case where the wiring was optional and nobody remembered it. Defaults inside
  distribution (b) were rejected because they hide which dependency is actually
  in play and make a test double indistinguishable from production wiring. See
  DD-30.

- **OQ-9: Canonical repository identity.** [RESOLVED] GitHub repository names are
  case-insensitive while pack names are validated identifiers, and the pack
  ecosystem may intentionally use a conceptual manifest name that differs from
  its repository basename. Which tuple is authoritative across source URL,
  requested `org/repository`, manifest name, install path, config key, lock key,
  and engine asset root? **Options:** (a) require canonical full repository
  identity to equal manifest name; (b) treat repository identity as source
  coordinate and manifest name as install/runtime identity, recording both in
  the lock; (c) allow an explicit manifest alias/source field that maps them.
  Host-specific case normalization must not be applied to arbitrary Git hosts.
  Manifest semantic version must still equal the effective tag version after
  normalizing only an optional `v` prefix. **Lean:** (b), because source location
  and pack identity are separate concepts and existing ecosystem names need not
  mirror repository basenames.
  **Resolution:** (b) — repository identity is the SOURCE COORDINATE only,
  recorded verbatim in the lock. The manifest name is the install/runtime
  IDENTITY: it determines the install path, the config key, the lock key, and the
  engine asset root. Both are recorded in the lock so a consumer can answer
  "where did this come from?" and "what is it called here?" independently.
  Host-specific case normalization is never applied — the requested coordinate is
  preserved as written, because case folding is a GitHub property and packs may
  be hosted anywhere. Manifest semver must equal the effective tag version after
  normalizing only an optional `v` prefix; that check stays fail-closed before
  mutation. Rationale: the harness failure was not an identity-equality problem —
  the pack legitimately named itself `backstop/harness-toolchain` while living at
  `backstop-ai/backstop-harness-toolchain-pack`. Forcing equality (a) would
  outlaw a reasonable naming convention; an alias field (c) adds a third name to
  keep coherent. Separating coordinate from identity fixes the real defect, which
  was that the manifest name was not the value used to build the install path.
  See DD-31.

## Spec Seeds

- **`pack add` command** — git resolution, clone, validate, install to
  `.backstop/packs/`, config merge, backstop.yml update, backstop.lock
  update. Includes local path pack variant.

- **`pack remove` command** — provenance-based config revert, uninstall
  from `.backstop/packs/`, backstop.yml cleanup, backstop.lock cleanup.

- **`pack install` command** — CI-optimized restore from lockfile, hash
  verification, no re-validation. Includes offline/cache support.

- **`pack update` / `pack upgrade` commands** — semver-aware version
  resolution, tamper detection, remediation bundle generation for major
  upgrades.

- **`backstop.lock` format and verification** — schema, hash algorithm,
  gate-time verification, local path pack handling.

- **tool_config provenance** — tracking which settings came from which
  pack, install-time value hashing, safe revert on pack remove.

### Reliability repair seeds (0.7.0)

- **Production remote dependency assembly** — real Git/tag implementations and
  constructors that make an incompletely assembled command unrepresentable
  (DD-30), with Cobra wiring for add, install, update, and upgrade. Foundation
  seed; identity and parity depend on its production seams. Fail-closed nil
  validation is permitted only as an interim step inside this seed.

- **Authored content identity and legacy lock migration** — root repository
  metadata exclusion for copy/hash, the typed migration-required diagnostic, the
  explicit remote re-validating migration operation and local `pack relock` path
  (DD-28), gate/relock/install convergence, and compatibility tests. Independent
  of remote dependency assembly except for the clone seam the remote migration
  operation needs and for final parity.

- **Remote reference and manifest identity** — effective version handling,
  source-coordinate-versus-manifest-identity separation with both recorded in the
  lock, verbatim coordinate preservation, manifest-name-derived install path /
  config key / engine asset root, `v`-prefix-normalized version equality, and
  pre-mutation diagnostics (DD-31). Depends on production clone seams.

- **Transactional distribution mutations** — the single shared staged-filesystem
  transaction coordinator plus command-specific participant registration (DD-29)
  giving exact rollback for installed packs, consumer tool config, provenance,
  manifests, lock, owned gitignore entries, and remove cleanup according to
  DD-27. Exposes the shared transaction primitives every command repair consumes,
  and the fault-injection seam the parity suite drives.

- **Hermetic remote lifecycle parity suite** — production CLI tests using
  temporary tagged repositories and URL rewriting, covering all commands and
  failure boundaries without network access. Final integration seed; depends on
  all preceding reliability seeds and tests the exact REQ-042 command matrix.

## Revision Impact on Existing Artifacts

`SPEC-015-pack-distribution.spec.md` remains historically pinned to
`pack-distribution-lifecycle:REQ-021@1.0.0`. That pin must not be rewritten: it
describes the algorithm the spec evaluated. Its existing ready plan must not be
replayed as the repair vehicle because it implements the legacy all-files
contract and does not cover REQ-038 through REQ-042. Reliability work requires
new delta specs that pin REQ-021@1.1.0, REQ-020@1.1.0, and the 1.1.0 revisions of
REQ-038 through REQ-042. OQ-6 through OQ-9 are resolved as of 0.8.0, but the
bundle remains at exploring maturity — promotion is a separate step, and
implementation stays unauthorized until the bundle is promoted and those delta
specs are written and reviewed.

## Out of Scope for the 0.7.0 Reliability Revision

- A hosted pack registry, catalog redesign, publishing proxy, attestations, or
  native package-registry distribution (owned by BUNDLE-001/BUNDLE-002).
- New Git authentication, credential storage, SSH-agent management, or
  host-specific transport beyond executing already-configured non-interactive
  Git commands.
- General pack validation-rule redesign; this revision wires existing pack
  check/test authority through production commands.
- New local-pack provenance semantics unrelated to repository metadata and hash
  migration.
- Unrelated gate dimensions or Go test-engine output handling.
- Changing pack authoring identity rules. DD-31 fixes core's mapping between
  source coordinate and runtime pack identity; it does not constrain what
  authors may name their packs or repositories.

## References

- BUNDLE-001 — parent vision bundle (pack distribution, verification, review)
- BUNDLE-004 — pack authoring (manifest schema, content types, archetypes)
- BUNDLE-005 — pack validation (pack check + pack test pipeline)
- BUNDLE-003 — onboarding (`backstop init` wires default packs via pack add)
- ADR-0001 — machine-first output (JSON for agent consumption)

## Version History

- **0.8.0** (2026-07-25): Resolved OQ-6 through OQ-9, closing every open
  question from the reliability revision. OQ-6 → always fail a legacy
  Git-metadata-inclusive hash with a typed migration diagnostic; local sources
  migrate via `pack relock`, git sources via an explicit re-validating remote
  migration operation with consumer acknowledgment; no in-place acceptance, and
  CI never acknowledges interactively (DD-28). OQ-7 → one shared
  staged-filesystem transaction coordinator with command-specific participants
  (DD-29). OQ-8 → constructors make incomplete options unrepresentable so a
  missing production dependency is a compile-time failure, with fail-closed
  validation permitted only as an interim path (DD-30). OQ-9 → repository
  identity is the source coordinate recorded verbatim, manifest name is the
  install/runtime identity driving install path, config key, lock key, and engine
  asset root, with both recorded in the lock and no host-specific case
  normalization (DD-31). Versioned the affected requirements to 1.1.0: REQ-020
  (lock records both identities), REQ-038 (compile-time dependency
  completeness), REQ-039 (coordinate/identity separation), REQ-040 (shared
  coordinator participants), REQ-041 (explicit migration policy), REQ-042
  (parity suite covers the named migration operation and coordinator fault
  injection). Added three success criteria and made the Homebrew distribution
  mental model explicit in problem framing, solution approach, and Current
  Thinking — tap as host, pack name as identity, reproducible pinned installs,
  remote as the default posture that all-local installs have not yet proven.
  Maturity deliberately held at exploring; promotion is a separate user-driven
  step.

- **0.7.0** (2026-07-22): Reopened to exploring after dogfooding a published
  harness toolchain pack exposed production remote lifecycle gaps. Versioned
  REQ-021 to define authored-content hashing and root repository-metadata
  exclusion; added REQ-038 through REQ-042 for production dependency wiring,
  remote identity validation, transactional mutation boundaries, legacy hash
  migration, and hermetic lifecycle parity. Added DD-24 through DD-27, OQ-6
  through OQ-9, and five reliability repair spec seeds. No implementation is
  authorized until the new open questions are resolved and the bundle returns
  to ready maturity.

- **0.6.0** (2026-04-08): Advanced to ready maturity. Added success criteria
  (10 criteria covering all six commands, lockfile format, gate verification,
  provenance tracking, and error paths). Added solution assumptions (pack
  check/test availability, git availability, enforcement semver adoption,
  structured config formats). All sections complete, all OQs resolved,
  requirements finalized (REQ-001 through REQ-037). Bundle is ready for spec
  generation.

- **0.5.0** (2026-04-08): Fixed duplicate REQs (removed REQ-037/038 which
  duplicated REQ-010/011), added REQ-037 for local path pack add via
  filesystem path, clarified REQ-012 that pack update validates the new
  version before updating the pin, bounded tamper detection scope in REQ-013
  (fixture removal, severity downgrade, risk_class change, rule removal),
  cleaned up stale REQ references in version history. Requirements renumbered
  REQ-001 through REQ-037. Maturity held at defined.

- **0.4.0** (2026-04-08): Fixed review findings — added REQs for orphan DDs
  (DD-5 SDK non-distribution, DD-3 lockfile existence), specified escalation
  mechanisms (REQ-004 conflict diagnostic with non-zero exit, REQ-013
  --acknowledge flag), fleshed out pack upgrade (REQ-015 through REQ-018:
  validate before install, tool_config update, version pin update, rollback
  on failure), added error/edge case REQs (already-installed, not-installed,
  missing lockfile, partial install atomicity, no-op update), tightened
  specificity throughout (REQ-001/002 exit behavior, REQ-019/024 gitignore
  ownership, REQ-022/027 no pack-to-pack dependencies), rewrote REQ-025/030
  to reference BUNDLE-004 enforcement semver model instead of re-defining it.
  Requirements renumbered REQ-001 through REQ-036. Maturity held at defined.

- **0.3.0** (2026-04-08): Advanced to defined maturity. Drafted 25 formal
  requirements (REQ-001 through REQ-025) covering pack add/remove/install/
  update/upgrade/list commands, backstop.lock format and verification,
  content hash algorithm, tool_config provenance, local path packs, no
  transitive trust, exact version pins, enforcement semver model, and
  .backstop/packs/ lifecycle. All requirements trace to design decisions.

- **0.2.0** (2026-04-08): All 5 OQs resolved. Added DD-19 through DD-23
  (YAML lockfile, SHA-256 sorted manifest hash, provenance JSON, cache flag
  for airgapped, exact version pins). Distribution bundle is now fully
  resolved with no open questions. Maturity held at exploring pending
  requirements drafting and spec seed refinement.

- **0.1.0** (2026-04-14): Initial bundle at exploring maturity. Extracted
  9 DDs from BUNDLE-001 and BUNDLE-004 (DD-1 through DD-9). Wrote 9 new
  DDs (DD-10 through DD-18) covering pack add/remove/install/update/upgrade
  mechanics, lock verification, tool_config provenance, local path packs,
  and .backstop/packs/ lifecycle. 5 open questions. 6 spec seeds.
