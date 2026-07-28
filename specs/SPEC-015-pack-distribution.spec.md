---
title: "SPEC-015: Pack Distribution Lifecycle"
number: SPEC-015
created: "2026-04-14"
updated: "2026-07-27"
status: replaced
replaced-by:
  - SPEC-056
  - SPEC-057
schema_version: spec/v1
spec_version: 2.0.0

implementation:
  summary: >
    Implement the pack distribution lifecycle: six commands (pack add, pack
    remove, pack install, pack update, pack upgrade, pack list) plus the
    backstop.lock format, SHA-256 content hashing, gate-time lock verification,
    tool_config provenance tracking, and .backstop/packs/ lifecycle management.
    pack add is the primary entry point that resolves, clones, validates,
    installs, merges config, and locks in a single command. pack install is the
    CI fast path that restores from the lockfile with hash verification and no
    re-validation. pack remove uses provenance tracking to revert config
    changes. pack update handles minor/patch bumps with tamper detection. pack
    upgrade handles major bumps with remediation bundle generation. pack list
    provides diagnostic output in human and JSON formats.
  package: pkg/pack/distribution

verification:
  level: integration
  test_command: go test ./pkg/pack/distribution/... -race -coverprofile=cover.out
  coverage_threshold: 80

requirements:
  - id: REQ-001
    text: >
      pack add must resolve org/pack-name to a git URL
      (github.com/org/pack-name by default), clone at the specified version
      tag, and exit with a non-zero exit code and a diagnostic message
      identifying the failure (missing tag, clone failure) if the tag does
      not exist or the clone fails.
    supports: pack-distribution-lifecycle:REQ-001@1.0.0

  - id: REQ-002
    text: >
      pack add must run pack check and pack test on the cloned pack before
      installation. If either validation step fails, pack add must exit with
      a non-zero exit code and a diagnostic message identifying the
      validation failure.
    supports: pack-distribution-lifecycle:REQ-002@1.0.0

  - id: REQ-003
    text: >
      pack add must copy the validated pack to .backstop/packs/org/pack-name/,
      compute the SHA-256 content hash, and update both backstop.yml (exact
      version pin) and backstop.lock (version, hash, git ref) atomically. If
      any step in the install pipeline fails after cloning, pack add must
      roll back all changes (no partial install state).
    supports: pack-distribution-lifecycle:REQ-003@1.0.0

  - id: REQ-004
    text: >
      When tool_config merge encounters a conflict (pack's desired value
      differs from consumer's current value for the same setting), pack add
      must exit with a non-zero exit code and output a diagnostic listing
      each conflicting setting, the pack's desired value, the consumer's
      current value, and the config file path. No interactive prompts in any
      environment. The consumer resolves conflicts manually and re-runs
      pack add.
    supports: pack-distribution-lifecycle:REQ-004@1.0.0

  - id: REQ-005
    text: >
      pack add must record every tool_config setting it merges in
      .backstop/pack-config-provenance.json, mapping config file path and
      setting key to source pack name and install-time value hash (SHA-256
      of the serialized setting value).
    supports: pack-distribution-lifecycle:REQ-005@1.0.0

  - id: REQ-006
    text: >
      pack remove must read .backstop/pack-config-provenance.json, revert
      settings sourced from the target pack, and warn (rather than revert)
      any setting whose current value hash differs from its install-time
      hash. The warning must identify the config file, setting key, and
      that the value was modified since install.
    supports: pack-distribution-lifecycle:REQ-006@1.0.0

  - id: REQ-007
    text: >
      pack remove must delete the pack from .backstop/packs/, remove the
      pack entry from backstop.yml, remove the pack entry from
      backstop.lock, and remove the pack's entries from
      pack-config-provenance.json.
    supports: pack-distribution-lifecycle:REQ-007@1.0.0

  - id: REQ-008
    text: >
      pack install must restore all packs from backstop.lock by cloning
      each at its pinned version, verifying the content hash matches the
      locked hash, and copying to .backstop/packs/. pack install must not
      run pack check or pack test.
    supports: pack-distribution-lifecycle:REQ-008@1.0.0

  - id: REQ-009
    text: >
      pack install must fail with a non-zero exit code and no partial
      install when any pack's computed content hash does not match the hash
      recorded in backstop.lock. The diagnostic must identify which pack
      had the hash mismatch.
    supports: pack-distribution-lifecycle:REQ-009@1.0.0

  - id: REQ-010
    text: >
      pack install --cache <path> must read packs from a local directory
      instead of cloning from git, while still verifying content hashes
      against the lockfile.
    supports: pack-distribution-lifecycle:REQ-010@1.0.0

  - id: REQ-011
    text: >
      pack install must not merge tool_config. Config was already merged
      at pack add time and committed to the repo. Install is a
      content-only restore.
    supports: pack-distribution-lifecycle:REQ-011@1.0.0

  - id: REQ-012
    text: >
      pack update must resolve the latest version within the semver
      minor/patch range for the specified pack, run pack check and pack
      test on the new version before updating. If validation fails, the
      update is aborted and the previous version is retained. On success,
      pack update writes the new exact version pin to backstop.yml and
      updates backstop.lock with the new version, hash, and git ref.
    supports: pack-distribution-lifecycle:REQ-012@1.0.0

  - id: REQ-013
    text: >
      When tamper detection finds fixture removal, severity downgrade,
      risk_class change, or rule removal during pack update, it must exit
      with a non-zero exit code and output a diagnostic listing each
      change. The consumer must re-run with --acknowledge flag to proceed.
      This applies in both interactive and CI environments. Other content
      changes in minor/patch versions are accepted without acknowledgment.
    supports: pack-distribution-lifecycle:REQ-013@1.0.0

  - id: REQ-014
    text: >
      pack upgrade must accept an explicit major version target
      (org/pack-name@version), scan the consumer's codebase against the
      new version, generate a remediation bundle scoping all new
      violations, and baseline those violations.
    supports: pack-distribution-lifecycle:REQ-014@1.0.0

  - id: REQ-015
    text: >
      pack upgrade must run pack check and pack test on the new version
      before installing. If validation fails, the upgrade is aborted.
    supports: pack-distribution-lifecycle:REQ-015@1.0.0

  - id: REQ-016
    text: >
      pack upgrade must update tool_config for the new version (the new
      version may have new or changed config requirements), with the same
      conflict escalation as pack add (REQ-004): non-zero exit and
      diagnostic listing on conflict, no interactive prompts.
    supports: pack-distribution-lifecycle:REQ-016@1.0.0

  - id: REQ-017
    text: >
      pack upgrade must update backstop.yml with the new exact version pin
      and backstop.lock with the new content hash.
    supports: pack-distribution-lifecycle:REQ-017@1.0.0

  - id: REQ-018
    text: >
      pack upgrade must generate a remediation bundle scoping all new
      violations. If remediation bundle generation fails, pack upgrade
      must roll back by restoring the previous version, backstop.yml,
      backstop.lock, and tool_config to their pre-upgrade state.
    supports: pack-distribution-lifecycle:REQ-018@1.0.0

  - id: REQ-019
    text: >
      pack list must display installed pack name, version, lock status
      (locked, stale, or missing), archetype, rule count, and scaffold
      count in a human-readable table by default. With --json, pack list
      must output the same data as structured JSON.
    supports: pack-distribution-lifecycle:REQ-019@1.0.0

  - id: REQ-020
    text: >
      backstop.lock must be YAML with sorted keys, containing per-pack
      entries with name, version, git ref (null for local packs), content
      hash (SHA-256), source type (git or local), and install date
      (RFC 3339).
    supports: pack-distribution-lifecycle:REQ-020@1.0.0

  - id: REQ-021
    text: >
      Content hash must be SHA-256 computed from a sorted manifest of
      relative-path:SHA-256-file-hash pairs covering every file in the
      pack directory. Paths are sorted lexicographically. Each file is
      individually hashed, then the sorted path:hash manifest is hashed
      to produce the final content hash.
    supports: pack-distribution-lifecycle:REQ-021@1.0.0

  - id: REQ-022
    text: >
      backstop gate must verify that every pack in backstop.lock is
      present in .backstop/packs/ with a matching content hash. Hash
      mismatch must produce a gate failure with a diagnostic identifying
      the pack and suggesting "run pack install to restore". Missing pack
      must produce a gate failure identifying which pack is missing. Extra
      unlocked pack (present in .backstop/packs/ but not in backstop.lock)
      must produce a gate failure identifying the extra pack.
    supports: pack-distribution-lifecycle:REQ-022@1.0.0

  - id: REQ-023
    text: >
      backstop gate must fail with a diagnostic error if backstop.lock is
      absent when packs are declared in backstop.yml.
    supports: pack-distribution-lifecycle:REQ-023@1.0.0

  - id: REQ-024
    text: >
      pack add must ensure .backstop/packs/ is listed in the project's
      .gitignore. If .gitignore does not exist, pack add creates it. If
      .backstop/packs/ is not already in .gitignore, pack add appends it.
      Pack contents must never be committed to the consumer's repo.
    supports: pack-distribution-lifecycle:REQ-024@1.0.0

  - id: REQ-025
    text: >
      Local path packs (referenced via path: in backstop.yml) must use
      the same pack check and pack test validation as git packs, appear
      in backstop.lock with a content hash but null git ref, and be
      verified at gate time by content hash.
    supports: pack-distribution-lifecycle:REQ-025@1.0.0

  - id: REQ-026
    text: >
      pack update must be a no-op with an informational message for local
      path packs since they update when their source files change. pack
      install must verify local pack content hash against the lockfile.
    supports: pack-distribution-lifecycle:REQ-026@1.0.0

  - id: REQ-027
    text: >
      Packs cannot declare dependencies on other packs. There is no
      pack-to-pack dependency mechanism. Every pack in a project must be
      explicitly added by the consumer via pack add. If a pack manifest
      references another pack, the system must not resolve or install it
      transitively.
    supports: pack-distribution-lifecycle:REQ-027@1.0.0

  - id: REQ-028
    text: >
      backstop.yml must store exact version pins for every pack with no
      range syntax. pack update resolves the latest compatible version
      internally and writes the resolved exact pin.
    supports: pack-distribution-lifecycle:REQ-028@1.0.0

  - id: REQ-029
    text: >
      .backstop/pack-config-provenance.json must be committed to the repo
      alongside backstop.yml and backstop.lock. It must track config file
      path, setting key/path, source pack name, and install-time value
      hash for every pack-contributed setting.
    supports: pack-distribution-lifecycle:REQ-029@1.0.0

  - id: REQ-030
    text: >
      pack update and pack upgrade must use the enforcement semver model:
      pack update auto-applies patch and minor versions, pack upgrade
      handles major versions. pack update must refuse to apply a major
      version bump and must direct the consumer to use pack upgrade
      instead.
    supports: pack-distribution-lifecycle:REQ-030@1.0.0

  - id: REQ-031
    text: >
      pack add and pack install must not install SDK dependencies declared
      in packs. SDK installation is the consumer's responsibility via
      native package managers. backstop tracks SDK references in the
      manifest and lockfile but does not distribute SDK code.
    supports: pack-distribution-lifecycle:REQ-031@1.0.0

  - id: REQ-032
    text: >
      pack add for a pack already listed in backstop.yml must exit with a
      non-zero exit code and a diagnostic suggesting pack update or pack
      upgrade instead.
    supports: pack-distribution-lifecycle:REQ-032@1.0.0

  - id: REQ-033
    text: >
      pack remove for a pack not listed in backstop.yml must exit with a
      non-zero exit code and a diagnostic identifying that the pack is
      not installed.
    supports: pack-distribution-lifecycle:REQ-033@1.0.0

  - id: REQ-034
    text: >
      pack install must fail with a non-zero exit code and a diagnostic
      if backstop.lock does not exist.
    supports: pack-distribution-lifecycle:REQ-034@1.0.0

  - id: REQ-035
    text: >
      pack install must not leave a partial installation in
      .backstop/packs/. If any pack clone or hash verification fails, the
      entire install fails and .backstop/packs/ is restored to its
      previous state (or left empty if it was a fresh install).
    supports: pack-distribution-lifecycle:REQ-035@1.0.0

  - id: REQ-036
    text: >
      pack update when the installed version is already the latest
      compatible version must be a no-op with an informational message.
    supports: pack-distribution-lifecycle:REQ-036@1.0.0

  - id: REQ-037
    text: >
      pack add must accept a local filesystem path as an alternative to a
      git org/pack-name reference. When given a local path, pack add must
      skip git resolution and cloning, validate the pack in place (run
      pack check and pack test), register it in backstop.yml with a path:
      entry instead of a version: entry, and compute a content hash for
      backstop.lock. Local path packs are not cloned to .backstop/packs/
      — they are loaded directly from their source path.
    supports: pack-distribution-lifecycle:REQ-037@1.0.0

claims:
  # --- pack add: git resolution and cloning ---
  - id: CLM-001
    requirement: REQ-001
    text: pack add resolves org/pack-name to github.com/org/pack-name git URL and clones at specified version tag
    tests:
      - TestPackAdd_ResolvesGitURL
      - TestPackAdd_ClonesAtVersionTag

  - id: CLM-002
    requirement: REQ-001
    text: pack add exits non-zero with diagnostic when version tag does not exist
    tests:
      - TestPackAdd_MissingTagExitsNonZero

  - id: CLM-003
    requirement: REQ-001
    text: pack add exits non-zero with diagnostic when clone fails (network error, invalid URL)
    tests:
      - TestPackAdd_CloneFailureExitsNonZero

  # --- pack add: validation gate ---
  - id: CLM-004
    requirement: REQ-002
    text: pack add runs pack check and pack test on cloned pack before installation
    tests:
      - TestPackAdd_RunsPackCheckBeforeInstall
      - TestPackAdd_RunsPackTestBeforeInstall

  - id: CLM-005
    requirement: REQ-002
    text: pack add exits non-zero with diagnostic when pack check fails
    tests:
      - TestPackAdd_PackCheckFailureAbortsInstall

  - id: CLM-006
    requirement: REQ-002
    text: pack add exits non-zero with diagnostic when pack test fails
    tests:
      - TestPackAdd_PackTestFailureAbortsInstall

  # --- pack add: install + lock + manifest update ---
  - id: CLM-007
    requirement: REQ-003
    text: pack add copies validated pack to .backstop/packs/org/pack-name/, computes content hash, updates backstop.yml and backstop.lock
    tests:
      - TestPackAdd_CopiesToPacksDir
      - TestPackAdd_ComputesContentHash
      - TestPackAdd_UpdatesBackstopYml
      - TestPackAdd_UpdatesBackstopLock

  - id: CLM-008
    requirement: REQ-003
    text: pack add rolls back all changes when a post-clone step fails (atomic install)
    tests:
      - TestPackAdd_RollbackOnPostCloneFailure

  # --- pack add: tool_config conflict ---
  - id: CLM-009
    requirement: REQ-004
    text: pack add exits non-zero with diagnostic listing conflicting settings, desired value, current value, and config file path on tool_config conflict
    tests:
      - TestPackAdd_ToolConfigConflictExitsNonZero
      - TestPackAdd_ToolConfigConflictDiagnosticFormat

  - id: CLM-010
    requirement: REQ-004
    text: pack add merges tool_config additively when there are no conflicts
    tests:
      - TestPackAdd_ToolConfigAdditiveMerge

  # --- pack add: provenance recording ---
  - id: CLM-011
    requirement: REQ-005
    text: pack add records every merged tool_config setting in pack-config-provenance.json with config file path, setting key, source pack name, and install-time value hash
    tests:
      - TestPackAdd_RecordsProvenance
      - TestPackAdd_ProvenanceContainsAllFields

  # --- pack remove: provenance-based revert ---
  - id: CLM-012
    requirement: REQ-006
    text: pack remove reverts settings sourced from the target pack using provenance data
    tests:
      - TestPackRemove_RevertsProvenanceSettings

  - id: CLM-013
    requirement: REQ-006
    text: pack remove warns instead of reverting when a setting's current value hash differs from install-time hash
    tests:
      - TestPackRemove_WarnsOnModifiedSetting

  # --- pack remove: cleanup ---
  - id: CLM-014
    requirement: REQ-007
    text: pack remove deletes pack from .backstop/packs/, removes from backstop.yml, backstop.lock, and pack-config-provenance.json
    tests:
      - TestPackRemove_DeletesFromPacksDir
      - TestPackRemove_RemovesFromBackstopYml
      - TestPackRemove_RemovesFromBackstopLock
      - TestPackRemove_RemovesFromProvenance

  # --- pack install: lockfile restore ---
  - id: CLM-015
    requirement: REQ-008
    text: pack install restores all packs from backstop.lock by cloning at pinned version and verifying content hash
    tests:
      - TestPackInstall_RestoresFromLockfile
      - TestPackInstall_VerifiesContentHash

  - id: CLM-016
    requirement: REQ-008
    text: pack install does not run pack check or pack test
    tests:
      - TestPackInstall_SkipsValidation

  # --- pack install: hash mismatch ---
  - id: CLM-017
    requirement: REQ-009
    text: pack install fails hard with non-zero exit and diagnostic when content hash does not match lockfile
    tests:
      - TestPackInstall_HashMismatchFailsHard

  # --- pack install: cache flag ---
  - id: CLM-018
    requirement: REQ-010
    text: pack install --cache reads packs from local directory instead of cloning, still verifies content hashes
    tests:
      - TestPackInstall_CacheReadsFromLocalDir
      - TestPackInstall_CacheStillVerifiesHash

  # --- pack install: no config merge ---
  - id: CLM-019
    requirement: REQ-011
    text: pack install does not merge tool_config
    tests:
      - TestPackInstall_SkipsToolConfigMerge

  # --- pack update: semver resolution + validation ---
  - id: CLM-020
    requirement: REQ-012
    text: pack update resolves latest minor/patch version, validates, and writes new exact pin to backstop.yml and backstop.lock
    tests:
      - TestPackUpdate_ResolvesLatestMinorPatch
      - TestPackUpdate_ValidatesBeforeUpdate
      - TestPackUpdate_WritesExactPin
      - TestPackUpdate_UpdatesLockfile

  - id: CLM-021
    requirement: REQ-012
    text: pack update aborts and retains previous version when validation fails on the new version
    tests:
      - TestPackUpdate_AbortsOnValidationFailure

  # --- pack update: tamper detection ---
  - id: CLM-022
    requirement: REQ-013
    text: pack update exits non-zero with diagnostic when tamper detection finds fixture removal
    tests:
      - TestPackUpdate_TamperDetectsFixtureRemoval

  - id: CLM-023
    requirement: REQ-013
    text: pack update exits non-zero with diagnostic when tamper detection finds severity downgrade
    tests:
      - TestPackUpdate_TamperDetectsSeverityDowngrade

  - id: CLM-024
    requirement: REQ-013
    text: pack update exits non-zero with diagnostic when tamper detection finds risk_class change
    tests:
      - TestPackUpdate_TamperDetectsRiskClassChange

  - id: CLM-025
    requirement: REQ-013
    text: pack update exits non-zero with diagnostic when tamper detection finds rule removal
    tests:
      - TestPackUpdate_TamperDetectsRuleRemoval

  - id: CLM-026
    requirement: REQ-013
    text: pack update proceeds when --acknowledge flag is provided after tamper detection
    tests:
      - TestPackUpdate_AcknowledgeBypassesTamperBlock

  - id: CLM-027
    requirement: REQ-013
    text: pack update accepts non-tamper content changes in minor/patch versions without acknowledgment
    tests:
      - TestPackUpdate_NonTamperChangesAccepted

  # --- pack upgrade: major version ---
  - id: CLM-028
    requirement: REQ-014
    text: pack upgrade accepts explicit major version target and generates a remediation bundle scoping new violations
    tests:
      - TestPackUpgrade_AcceptsMajorVersionTarget
      - TestPackUpgrade_GeneratesRemediationBundle

  - id: CLM-029
    requirement: REQ-014
    text: pack upgrade baselines new violations from the major version
    tests:
      - TestPackUpgrade_BaselinesNewViolations

  # --- pack upgrade: validation ---
  - id: CLM-030
    requirement: REQ-015
    text: pack upgrade runs pack check and pack test on the new version before installing
    tests:
      - TestPackUpgrade_ValidatesBeforeInstall

  - id: CLM-031
    requirement: REQ-015
    text: pack upgrade aborts when validation fails on the new version
    tests:
      - TestPackUpgrade_AbortsOnValidationFailure

  # --- pack upgrade: tool_config ---
  - id: CLM-032
    requirement: REQ-016
    text: pack upgrade updates tool_config for the new version with same conflict escalation as pack add
    tests:
      - TestPackUpgrade_UpdatesToolConfig
      - TestPackUpgrade_ToolConfigConflictExitsNonZero

  # --- pack upgrade: version pin ---
  - id: CLM-033
    requirement: REQ-017
    text: pack upgrade updates backstop.yml with new exact version pin and backstop.lock with new content hash
    tests:
      - TestPackUpgrade_UpdatesBackstopYml
      - TestPackUpgrade_UpdatesBackstopLock

  # --- pack upgrade: rollback ---
  - id: CLM-034
    requirement: REQ-018
    text: pack upgrade generates remediation bundle scoping all new violations
    tests:
      - TestPackUpgrade_RemediationBundleCoversAllViolations

  - id: CLM-035
    requirement: REQ-018
    text: pack upgrade rolls back to previous version when remediation bundle generation fails
    tests:
      - TestPackUpgrade_RollbackOnRemediationFailure

  # --- pack list ---
  - id: CLM-036
    requirement: REQ-019
    text: pack list displays name, version, lock status, archetype, rule count, and scaffold count in human-readable table
    tests:
      - TestPackList_HumanTableOutput

  - id: CLM-037
    requirement: REQ-019
    text: pack list --json outputs the same data as structured JSON
    tests:
      - TestPackList_JsonOutput

  - id: CLM-038
    requirement: REQ-019
    text: >
      pack list lock status is "locked" when hash matches lockfile, "stale"
      when hash differs, and "missing" when pack is in lockfile but not
      installed
    tests:
      - TestPackList_LockStatusLocked
      - TestPackList_LockStatusStale
      - TestPackList_LockStatusMissing

  # --- backstop.lock format ---
  - id: CLM-039
    requirement: REQ-020
    text: backstop.lock is YAML with sorted keys containing name, version, git ref, content hash, source type, and install date per pack
    tests:
      - TestLockfile_YamlSortedKeys
      - TestLockfile_ContainsAllRequiredFields

  - id: CLM-040
    requirement: REQ-020
    text: backstop.lock stores null git ref for local packs and a valid git ref for git packs
    tests:
      - TestLockfile_NullGitRefForLocalPack
      - TestLockfile_ValidGitRefForGitPack

  # --- content hash ---
  - id: CLM-041
    requirement: REQ-021
    text: content hash is SHA-256 of sorted manifest of relative-path:SHA-256-file-hash pairs covering every file in pack directory
    tests:
      - TestContentHash_SortedManifest
      - TestContentHash_CoversAllFiles
      - TestContentHash_DeterministicOutput

  # --- gate verification ---
  - id: CLM-042
    requirement: REQ-022
    text: backstop gate fails with diagnostic when installed pack content hash does not match lockfile hash
    tests:
      - TestGate_HashMismatchFails

  - id: CLM-043
    requirement: REQ-022
    text: backstop gate fails with diagnostic when a locked pack is missing from .backstop/packs/
    tests:
      - TestGate_MissingPackFails

  - id: CLM-044
    requirement: REQ-022
    text: backstop gate fails with diagnostic when an extra unlocked pack is present in .backstop/packs/ but not in backstop.lock
    tests:
      - TestGate_ExtraUnlockedPackFails

  - id: CLM-045
    requirement: REQ-022
    text: backstop gate passes when all packs match their locked hashes with no extras
    tests:
      - TestGate_AllPacksMatchPasses

  - id: CLM-046
    requirement: REQ-023
    text: backstop gate fails with diagnostic when backstop.lock is absent but packs are declared in backstop.yml
    tests:
      - TestGate_MissingLockfileWithPacksFails

  - id: CLM-047
    requirement: REQ-023
    text: backstop gate passes when no packs are declared in backstop.yml and no lockfile exists
    tests:
      - TestGate_NoPacksNoLockfilePasses

  # --- gitignore management ---
  - id: CLM-048
    requirement: REQ-024
    text: pack add creates .gitignore with .backstop/packs/ entry when .gitignore does not exist
    tests:
      - TestPackAdd_CreatesGitignore

  - id: CLM-049
    requirement: REQ-024
    text: pack add appends .backstop/packs/ to existing .gitignore when not already present
    tests:
      - TestPackAdd_AppendsToGitignore

  - id: CLM-050
    requirement: REQ-024
    text: pack add does not modify .gitignore when .backstop/packs/ is already listed
    tests:
      - TestPackAdd_GitignoreAlreadyPresent

  # --- local path packs ---
  - id: CLM-051
    requirement: REQ-025
    text: local path packs are validated with pack check and pack test same as git packs
    tests:
      - TestLocalPack_ValidatedSameAsGit

  - id: CLM-052
    requirement: REQ-025
    text: local path packs appear in backstop.lock with content hash but null git ref
    tests:
      - TestLocalPack_LockEntryHasHashNoGitRef

  - id: CLM-053
    requirement: REQ-025
    text: backstop gate verifies local path packs by content hash
    tests:
      - TestGate_LocalPackVerifiedByHash

  # --- local pack update behavior ---
  - id: CLM-054
    requirement: REQ-026
    text: pack update is a no-op with informational message for local path packs
    tests:
      - TestPackUpdate_LocalPackNoOp

  - id: CLM-055
    requirement: REQ-026
    text: pack install verifies local pack content hash against lockfile
    tests:
      - TestPackInstall_LocalPackHashVerification

  # --- no transitive dependencies ---
  - id: CLM-056
    requirement: REQ-027
    text: packs cannot declare dependencies on other packs and no transitive resolution occurs
    tests:
      - TestPackAdd_NoTransitiveDependencies

  # --- exact version pins ---
  - id: CLM-057
    requirement: REQ-028
    text: backstop.yml stores exact version pins with no range syntax
    tests:
      - TestBackstopYml_ExactVersionPins

  - id: CLM-058
    requirement: REQ-028
    text: pack update resolves latest compatible version and writes resolved exact pin
    tests:
      - TestPackUpdate_WritesResolvedExactPin

  # --- provenance file format ---
  - id: CLM-059
    requirement: REQ-029
    text: pack-config-provenance.json tracks config file path, setting key, source pack name, and install-time value hash
    tests:
      - TestProvenance_TracksAllFields

  - id: CLM-060
    requirement: REQ-029
    text: pack-config-provenance.json is a committed file alongside backstop.yml and backstop.lock
    tests:
      - TestProvenance_CommittedToRepo

  # --- enforcement semver model ---
  - id: CLM-061
    requirement: REQ-030
    text: pack update auto-applies patch and minor versions
    tests:
      - TestPackUpdate_AppliesPatchVersion
      - TestPackUpdate_AppliesMinorVersion

  - id: CLM-062
    requirement: REQ-030
    text: pack update refuses major version bump and directs consumer to use pack upgrade
    tests:
      - TestPackUpdate_RefusesMajorVersion

  # --- SDK non-distribution ---
  - id: CLM-063
    requirement: REQ-031
    text: pack add does not install SDK dependencies declared in packs
    tests:
      - TestPackAdd_SkipsSDKDependencies

  - id: CLM-064
    requirement: REQ-031
    text: pack install does not install SDK dependencies declared in packs
    tests:
      - TestPackInstall_SkipsSDKDependencies

  # --- already-installed guard ---
  - id: CLM-065
    requirement: REQ-032
    text: pack add for an already-installed pack exits non-zero with diagnostic suggesting pack update or pack upgrade
    tests:
      - TestPackAdd_AlreadyInstalledExitsNonZero

  # --- not-installed guard ---
  - id: CLM-066
    requirement: REQ-033
    text: pack remove for a pack not in backstop.yml exits non-zero with diagnostic
    tests:
      - TestPackRemove_NotInstalledExitsNonZero

  # --- missing lockfile guard ---
  - id: CLM-067
    requirement: REQ-034
    text: pack install fails with non-zero exit and diagnostic when backstop.lock does not exist
    tests:
      - TestPackInstall_MissingLockfileExitsNonZero

  # --- atomic install ---
  - id: CLM-068
    requirement: REQ-035
    text: pack install restores .backstop/packs/ to previous state when any pack clone fails (no partial install)
    tests:
      - TestPackInstall_AtomicRollbackOnCloneFailure

  - id: CLM-069
    requirement: REQ-035
    text: pack install restores .backstop/packs/ to previous state when any hash verification fails (no partial install)
    tests:
      - TestPackInstall_AtomicRollbackOnHashFailure

  # --- already-latest guard ---
  - id: CLM-070
    requirement: REQ-036
    text: pack update is a no-op with informational message when installed version is already the latest compatible version
    tests:
      - TestPackUpdate_AlreadyLatestNoOp

  # --- local path pack add ---
  - id: CLM-071
    requirement: REQ-037
    text: "pack add accepts a local filesystem path, skips git resolution, validates in place, registers with path: entry in backstop.yml"
    tests:
      - TestPackAdd_LocalPathSkipsGit
      - TestPackAdd_LocalPathValidatesInPlace
      - TestPackAdd_LocalPathRegistersWithPathEntry

  - id: CLM-072
    requirement: REQ-037
    text: pack add for local path computes content hash for backstop.lock and does not clone to .backstop/packs/
    tests:
      - TestPackAdd_LocalPathComputesHash
      - TestPackAdd_LocalPathNotClonedToPacksDir

contracts:
  - file: pkg/pack/distribution/add.go
    provides:
      - name: Add
        kind: function
        signature: "func Add(packRef string, opts AddOptions) (*AddResult, error)"
      - name: AddOptions
        kind: type
        signature: "type AddOptions struct"
      - name: AddResult
        kind: type
        signature: "type AddResult struct"
    consumes:
      - source: pkg/pack/distribution
        name: ComputeContentHash
        kind: function
      - source: pkg/pack/distribution
        name: WriteLockfile
        kind: function
      - source: pkg/pack/distribution
        name: MergeToolConfig
        kind: function

  - file: pkg/pack/distribution/remove.go
    provides:
      - name: Remove
        kind: function
        signature: "func Remove(packName string, opts RemoveOptions) (*RemoveResult, error)"
      - name: RemoveOptions
        kind: type
        signature: "type RemoveOptions struct"
      - name: RemoveResult
        kind: type
        signature: "type RemoveResult struct"
    consumes:
      - source: pkg/pack/distribution
        name: ReadProvenance
        kind: function
      - source: pkg/pack/distribution
        name: WriteLockfile
        kind: function

  - file: pkg/pack/distribution/install.go
    provides:
      - name: Install
        kind: function
        signature: "func Install(opts InstallOptions) (*InstallResult, error)"
      - name: InstallOptions
        kind: type
        signature: "type InstallOptions struct"
        notes: "Includes CachePath string for --cache flag"
      - name: InstallResult
        kind: type
        signature: "type InstallResult struct"
    consumes:
      - source: pkg/pack/distribution
        name: ReadLockfile
        kind: function
      - source: pkg/pack/distribution
        name: ComputeContentHash
        kind: function

  - file: pkg/pack/distribution/update.go
    provides:
      - name: Update
        kind: function
        signature: "func Update(packName string, opts UpdateOptions) (*UpdateResult, error)"
      - name: UpdateOptions
        kind: type
        signature: "type UpdateOptions struct"
        notes: "Includes Acknowledge bool for --acknowledge flag"
      - name: UpdateResult
        kind: type
        signature: "type UpdateResult struct"
    consumes:
      - source: pkg/pack/distribution
        name: ComputeContentHash
        kind: function
      - source: pkg/pack/distribution
        name: WriteLockfile
        kind: function
      - source: pkg/pack/distribution
        name: DetectTamper
        kind: function

  - file: pkg/pack/distribution/upgrade.go
    provides:
      - name: Upgrade
        kind: function
        signature: "func Upgrade(packRef string, opts UpgradeOptions) (*UpgradeResult, error)"
      - name: UpgradeOptions
        kind: type
        signature: "type UpgradeOptions struct"
      - name: UpgradeResult
        kind: type
        signature: "type UpgradeResult struct"
    consumes:
      - source: pkg/pack/distribution
        name: ComputeContentHash
        kind: function
      - source: pkg/pack/distribution
        name: WriteLockfile
        kind: function
      - source: pkg/pack/distribution
        name: MergeToolConfig
        kind: function

  - file: pkg/pack/distribution/list.go
    provides:
      - name: List
        kind: function
        signature: "func List(opts ListOptions) (*ListResult, error)"
      - name: ListOptions
        kind: type
        signature: "type ListOptions struct"
        notes: "Includes JSON bool for --json flag"
      - name: ListResult
        kind: type
        signature: "type ListResult struct"
      - name: PackInfo
        kind: type
        signature: "type PackInfo struct"
        notes: "Name, Version, LockStatus, Archetype, RuleCount, ScaffoldCount"
    consumes:
      - source: pkg/pack/distribution
        name: ReadLockfile
        kind: function
      - source: pkg/pack/distribution
        name: ComputeContentHash
        kind: function

  - file: pkg/pack/distribution/lockfile.go
    provides:
      - name: Lockfile
        kind: type
        signature: "type Lockfile struct"
      - name: LockEntry
        kind: type
        signature: "type LockEntry struct"
        notes: "Name, Version, GitRef *string, ContentHash, SourceType, InstallDate"
      - name: ReadLockfile
        kind: function
        signature: "func ReadLockfile(path string) (*Lockfile, error)"
      - name: WriteLockfile
        kind: function
        signature: "func WriteLockfile(path string, lockfile *Lockfile) error"
    consumes: []

  - file: pkg/pack/distribution/hash.go
    provides:
      - name: ComputeContentHash
        kind: function
        signature: "func ComputeContentHash(dir string) (string, error)"
      - name: ComputeFileHash
        kind: function
        signature: "func ComputeFileHash(path string) (string, error)"
    consumes: []

  - file: pkg/pack/distribution/provenance.go
    provides:
      - name: Provenance
        kind: type
        signature: "type Provenance struct"
      - name: ProvenanceEntry
        kind: type
        signature: "type ProvenanceEntry struct"
        notes: "ConfigFile, SettingKey, SourcePack, ValueHash"
      - name: ReadProvenance
        kind: function
        signature: "func ReadProvenance(path string) (*Provenance, error)"
      - name: WriteProvenance
        kind: function
        signature: "func WriteProvenance(path string, prov *Provenance) error"
    consumes: []

  - file: pkg/pack/distribution/config_merge.go
    provides:
      - name: MergeToolConfig
        kind: function
        signature: "func MergeToolConfig(packDir string, projectDir string, prov *Provenance) (*MergeResult, error)"
      - name: MergeResult
        kind: type
        signature: "type MergeResult struct"
        notes: "Merged []ProvenanceEntry, Conflicts []ConfigConflict"
      - name: ConfigConflict
        kind: type
        signature: "type ConfigConflict struct"
        notes: "ConfigFile, SettingKey, PackValue, CurrentValue"
    consumes: []

  - file: pkg/pack/distribution/tamper.go
    provides:
      - name: DetectTamper
        kind: function
        signature: "func DetectTamper(oldPackDir string, newPackDir string) (*TamperResult, error)"
      - name: TamperResult
        kind: type
        signature: "type TamperResult struct"
        notes: "Changes []TamperChange, HasTamper bool"
      - name: TamperChange
        kind: type
        signature: "type TamperChange struct"
        notes: "Kind (fixture_removal, severity_downgrade, risk_class_change, rule_removal), Description"
    consumes: []

  - file: pkg/pack/distribution/verify.go
    provides:
      - name: VerifyLock
        kind: function
        signature: "func VerifyLock(lockfile *Lockfile, packsDir string, ymlPacks []string) (*VerifyResult, error)"
      - name: VerifyResult
        kind: type
        signature: "type VerifyResult struct"
        notes: "Pass bool, Failures []LockFailure"
      - name: LockFailure
        kind: type
        signature: "type LockFailure struct"
        notes: "Pack, Kind (hash_mismatch, missing_pack, extra_unlocked, missing_lockfile), Message"
    consumes:
      - source: pkg/pack/distribution
        name: ComputeContentHash
        kind: function
      - source: pkg/pack/distribution
        name: ReadLockfile
        kind: function
---

# SPEC-015: Pack Distribution Lifecycle

## Overview

This spec covers the distribution lifecycle for backstop enforcement packs: the pipeline between "pack exists in a git repo and passes validation" and "consumer's gate enforces that pack's rules with cryptographic reproducibility." Six commands form the distribution surface: `pack add`, `pack remove`, `pack install`, `pack update`, `pack upgrade`, and `pack list`. Supporting infrastructure includes the `backstop.lock` lockfile, SHA-256 content hashing, gate-time lock verification, tool_config provenance tracking, and `.backstop/packs/` lifecycle management.

The distribution model is git-native (similar to Homebrew taps). There is no central registry. `pack add` is the primary entry point: it resolves, clones, validates, installs, merges config, and locks in a single atomic command. `pack install` is the CI fast path: it restores from the lockfile with hash verification and no re-validation. `pack remove` uses provenance tracking to cleanly revert config changes. `pack update` handles minor/patch bumps with tamper detection. `pack upgrade` handles major bumps with remediation bundle generation. `pack list` provides diagnostic output for humans and agents.

Key design principles:
- **Validate before install**: Every pack is proven via `pack check` and `pack test` before it can affect the consumer. `pack install` skips validation because the content hash proves identity with what was validated at add time.
- **Lockfile as single source of truth**: `backstop.lock` pins every pack by version and SHA-256 content hash. The gate verifies installed packs match the lock. Hash mismatch, missing pack, or extra unlocked pack are gate failures.
- **Provenance-based config revert**: `pack add` records every tool_config setting it merges. `pack remove` reads this provenance to revert settings, warning on manual modifications.
- **No transitive trust**: Packs cannot depend on other packs. Every pack is explicitly added by the consumer.
- **Exact version pins**: backstop.yml stores exact versions, never ranges. `pack update` resolves internally and writes the resolved pin.
- **Local path packs**: First-class support for in-repo packs loaded directly from the filesystem with the same validation and lock verification as git packs.

## Requirements

Requirements are defined in frontmatter. REQ-001 through REQ-037 cover:

- **Pack add** (REQ-001 through REQ-005, REQ-024, REQ-031, REQ-032, REQ-037): git resolution with diagnostic exit on failure, validation gate, install + lock + manifest update with atomic rollback, tool_config merge with conflict escalation, provenance recording, gitignore management, SDK non-distribution, already-installed guard, local path pack support.
- **Pack remove** (REQ-006, REQ-007, REQ-033): provenance-based config revert with manual-modification detection, full cleanup across packs dir / yml / lock / provenance, not-installed guard.
- **Pack install** (REQ-008 through REQ-011, REQ-034, REQ-035): lockfile restore with hash verification, hard failure on hash mismatch, --cache flag for offline/airgapped, no re-validation or config merge, missing lockfile guard, atomic install with rollback.
- **Pack update** (REQ-012, REQ-013, REQ-030, REQ-036): semver minor/patch resolution, exact pin write, tamper detection with --acknowledge escalation, no-op when already latest, refuses major bumps.
- **Pack upgrade** (REQ-014 through REQ-018, REQ-030): major version target, validate before install, tool_config update with conflict escalation, version pin and lockfile update, remediation bundle generation with rollback on failure.
- **Pack list** (REQ-019): human table and JSON output with lock status (locked/stale/missing).
- **backstop.lock** (REQ-020, REQ-021): YAML sorted keys, SHA-256 sorted manifest hash.
- **Gate verification** (REQ-022, REQ-023): content hash comparison, diagnostic messages for mismatch/missing/extra, mandatory lockfile when packs declared.
- **Gitignore management** (REQ-024): pack add owns .gitignore entry for .backstop/packs/.
- **Local path packs** (REQ-025, REQ-026, REQ-037): same validation, null git ref, update is no-op, loaded from source path.
- **Trust model** (REQ-027): no pack-to-pack dependencies.
- **Version semantics** (REQ-028, REQ-030): exact pins, enforcement semver model.
- **Provenance file** (REQ-029): committed structured tracking.
- **SDK non-distribution** (REQ-031): packs track SDK references but do not distribute SDK code.

## Implementation

### Package Structure

```
pkg/pack/distribution/
├── add.go            # pack add: resolve, clone, validate, install, merge, lock
├── remove.go         # pack remove: provenance revert, cleanup
├── install.go        # pack install: lockfile restore, hash verify, --cache
├── update.go         # pack update: semver resolve, validate, tamper detect
├── upgrade.go        # pack upgrade: major version, remediation bundle, rollback
├── list.go           # pack list: table and JSON output
├── lockfile.go       # backstop.lock read/write, YAML sorted keys
├── hash.go           # SHA-256 content hashing (sorted manifest approach)
├── provenance.go     # pack-config-provenance.json read/write
├── config_merge.go   # tool_config additive merge with conflict detection
├── tamper.go         # tamper detection (fixture/severity/risk_class/rule changes)
├── verify.go         # gate-time lock verification
└── *_test.go         # tests for each file
```

### Command Pipeline: pack add

1. Parse pack reference (org/pack-name@version or local path)
2. If git reference: resolve to git URL, clone at version tag
3. If local path: validate path exists and contains pack.yml
4. Run `pack check` on the pack directory
5. Run `pack test` on the pack directory
6. If git: copy validated pack to `.backstop/packs/org/pack-name/`
7. Compute SHA-256 content hash of installed pack directory
8. Merge tool_config into consumer's project (detect conflicts, abort on conflict)
9. Record merged settings in `pack-config-provenance.json`
10. Update `backstop.yml` with exact version pin (or path: entry for local)
11. Update `backstop.lock` with version, hash, git ref, source type, install date
12. Ensure `.backstop/packs/` is in `.gitignore`
13. On failure at any step after clone: roll back all changes

### Command Pipeline: pack remove

1. Verify pack is listed in backstop.yml (exit non-zero if not)
2. Read `pack-config-provenance.json`
3. For each setting sourced from the target pack: compare current value hash to install-time hash
4. Revert settings where hash matches; warn where hash differs
5. Delete pack from `.backstop/packs/`
6. Remove pack entry from `backstop.yml`
7. Remove pack entry from `backstop.lock`
8. Remove pack's entries from `pack-config-provenance.json`

### Command Pipeline: pack install

1. Verify `backstop.lock` exists (exit non-zero if not)
2. Snapshot current `.backstop/packs/` state for atomic rollback
3. For each pack in lockfile:
   a. If git source: clone at pinned version (or read from --cache path)
   b. If local source: read from local path
   c. Compute content hash, compare to locked hash
   d. On hash mismatch: abort entire install, restore snapshot
   e. Copy to `.backstop/packs/`
4. Do NOT run pack check, pack test, or tool_config merge

### Command Pipeline: pack update

1. Read current version from backstop.yml
2. Resolve latest compatible minor/patch version from git tags
3. If local pack: no-op with informational message
4. If already at latest: no-op with informational message
5. If resolved version is a major bump: refuse, direct to pack upgrade
6. Clone new version, run pack check + pack test
7. Run tamper detection against current version
8. If tamper detected and --acknowledge not set: exit non-zero with diagnostic
9. Install new version, update backstop.yml and backstop.lock

### Command Pipeline: pack upgrade

1. Parse explicit major version target from pack reference
2. Clone new version, run pack check + pack test
3. Scan consumer codebase against new version for violations
4. Generate remediation bundle scoping new violations
5. If remediation bundle generation fails: roll back everything
6. Update tool_config (conflict escalation same as pack add)
7. Update backstop.yml and backstop.lock
8. Baseline new violations

### Command Pipeline: pack list

1. Read backstop.yml for declared packs
2. Read backstop.lock for locked versions and hashes
3. For each pack: compute current content hash, compare to lock
4. Determine lock status: locked (match), stale (mismatch), missing (not installed)
5. Read pack manifest for archetype, rule count, scaffold count
6. Output as human table (default) or JSON (--json)

### Lock Verification at Gate Time

1. Read backstop.yml for declared packs
2. If packs declared but backstop.lock absent: fail with diagnostic
3. Read backstop.lock
4. For each locked pack: verify present in .backstop/packs/ with matching hash
5. For each pack in .backstop/packs/: verify present in backstop.lock
6. Any mismatch, missing pack, or extra unlocked pack: gate failure with specific diagnostic

### Content Hash Algorithm

SHA-256 of a sorted manifest of relative-path:SHA-256-file-hash pairs:
1. Walk all files in the pack directory
2. Compute SHA-256 hash of each file's content
3. Build manifest: `relative/path:hexdigest` per file
4. Sort manifest entries lexicographically by path
5. Join sorted entries with newlines
6. Compute SHA-256 of the joined manifest string

### backstop.lock Format

```yaml
packs:
  acme/go-http-standards:
    content_hash: "sha256:a1b2c3..."
    git_ref: "v1.2.0"
    install_date: "2026-04-14T10:30:00Z"
    name: "acme/go-http-standards"
    source_type: "git"
    version: "1.2.0"
  internal/local-rules:
    content_hash: "sha256:d4e5f6..."
    git_ref: null
    install_date: "2026-04-14T11:00:00Z"
    name: "internal/local-rules"
    source_type: "local"
    version: null
```

### pack-config-provenance.json Format

```json
{
  "entries": [
    {
      "config_file": ".golangci.yml",
      "setting_key": "linters.enable.revive",
      "source_pack": "acme/go-http-standards",
      "value_hash": "sha256:abc123..."
    }
  ]
}
```

## Verification

Verification is defined in frontmatter. Integration-level verification with 80% coverage threshold.

Test command: `go test ./pkg/pack/distribution/... -race -coverprofile=cover.out`

Claims are defined in frontmatter. They cover every requirement with both positive (passes) and negative (fails) scenarios, including:
- Git resolution success and failure paths
- Validation gate pass and abort paths
- Atomic rollback on post-clone failures
- tool_config merge success and conflict paths
- Provenance recording and revert paths (including modified-setting warnings)
- Lockfile restore with hash verification pass and mismatch paths
- Cache-based install
- Tamper detection for all four tamper types (fixture removal, severity downgrade, risk_class change, rule removal) plus --acknowledge bypass
- Major version upgrade with remediation bundle generation and rollback
- Lock status computation (locked, stale, missing)
- Gate verification for all four failure modes (hash mismatch, missing pack, extra unlocked, missing lockfile)
- Gitignore creation, append, and idempotency
- Local path pack add, install, update, and gate verification
- Already-installed and not-installed error guards
- Missing lockfile guard
- Atomic install rollback on clone failure and hash failure
- Already-latest no-op
- SDK non-distribution
- Exact version pin enforcement
- Major version refusal by pack update

## Sharp Edges

- **Network failures during pack add**: A clone failure after tool_config has been partially merged requires full rollback of config changes, backstop.yml, backstop.lock, and .backstop/packs/. The rollback itself can fail (e.g., filesystem permission error during revert), leaving the project in an inconsistent state. The implementation must log the rollback failure clearly so the consumer can manually recover.

- **Partial installs during pack install**: `pack install` must be atomic across all packs. If pack 3 of 5 fails hash verification, packs 1 and 2 must be rolled back. The snapshot-and-restore approach (copy .backstop/packs/ before starting) has a cost: large pack directories mean slow snapshots. An alternative is a staging directory with atomic rename, but this fails on cross-device moves.

- **Config format diversity**: tool_config merge must handle YAML (.golangci.yml), JSON (.eslintrc.json), TOML, and potentially other formats. Each format has different merge semantics (YAML anchors, JSON schema references, TOML inline tables). A setting key like `linters.enable.revive` may have different structural representations across formats. The merge layer must be extensible per format without coupling to specific tools.

- **Hash instability across platforms**: Content hashing depends on file content being byte-identical across platforms. Git's autocrlf setting can change line endings between Windows and Unix, producing different hashes for the same logical content. The hash algorithm must either normalize line endings or document that packs must use consistent line endings (enforced by .gitattributes).

- **Provenance drift after manual config edits**: If a consumer edits a pack-provided setting and then runs `pack remove`, the warning is correct but the setting is orphaned — it came from a pack that no longer exists, but the consumer intended to keep it. There is no mechanism to "adopt" a pack-provided setting as consumer-owned. Over time, provenance.json can accumulate stale entries from removed packs if warnings are ignored.

- **Tamper detection false positives**: The four tamper categories (fixture removal, severity downgrade, risk_class change, rule removal) are heuristics. A pack author might legitimately remove a fixture that tested a rule they replaced with a better one. The --acknowledge escape hatch exists, but frequent false positives erode trust in the mechanism. The tamper detection logic must be precise about what constitutes each category.

- **Remediation bundle generation timing**: `pack upgrade` generates a remediation bundle by scanning the consumer's codebase. If the codebase has uncommitted changes, the scan results may not reflect the committed state. The spec does not require a clean-working-tree check, which could lead to remediation bundles that are stale by the time they are acted on.

- **Local pack hash drift**: Local path packs are not cloned to .backstop/packs/ — they are loaded from their source path. If the consumer modifies local pack files after `pack add`, the content hash in backstop.lock becomes stale. The gate will catch this, but the fix is to re-run `pack add` (which re-validates), not `pack install`. This differs from git packs where `pack install` is always the fix.

- **Concurrent pack operations**: Nothing prevents two terminal sessions from running `pack add` simultaneously. Both would read backstop.yml, compute changes, and write — the second write wins and the first pack's changes are lost. File-level locking or a .backstop/lock file could prevent this, but is not specified.

## Review Questions

1. Does the atomic rollback in `pack add` restore .gitignore changes if the gitignore was created (not just appended to) by the failing add operation?
2. When `pack upgrade` rolls back due to remediation bundle failure, does it also revert tool_config changes made for the new version, or only the version pin and lockfile?
3. How does `pack install --cache` handle the case where the cache directory contains a pack at a different version than what the lockfile specifies? Does it fail with a hash mismatch or a more specific "version mismatch in cache" diagnostic?
4. If two packs contribute the same setting to the same config file, what happens when the first pack is removed? Does provenance tracking handle multi-pack ownership of the same setting key?
5. Does `pack update` with --acknowledge persist the acknowledgment in backstop.lock or backstop.yml so that subsequent `pack install` does not re-trigger tamper detection (which it should not, since install skips validation)?
6. When a local path pack's source directory is deleted or moved, `pack install` cannot verify its hash. Does this produce a "missing pack" or "hash mismatch" diagnostic, and is the distinction clear to the consumer?
7. Does the content hash algorithm handle empty directories within a pack? Empty directories have no files to hash, so two packs differing only by empty directory presence would have the same hash.

## References

- BUNDLE-006: Pack Distribution Lifecycle (pack-distribution-lifecycle v0.6.0)
- BUNDLE-004: Pack Authoring (enforcement semver model)
- BUNDLE-005: Pack Validation (pack check, pack test)
- ADR-0001: Agent-first design (structured JSON output for agent consumption)

## Version History

- **2.0.0** (2026-07-27) — Status → `replaced` (terminal), with
  `replaced-by: [SPEC-056, SPEC-057]`. The field accepts a LIST
  (`pkg/validate/terminal.go:37-38`), and two successors is the honest shape
  here: BUNDLE-006's "Revision Impact on Existing Artifacts" (~line 1435)
  revised the two requirements this spec pins to 1.1.0 and prescribed new delta
  specs rather than a rewrite of these pins, and the revision landed split.
  **SPEC-056** succeeds the `pack-distribution-lifecycle:REQ-020@1.0.0` pin,
  delivering `REQ-020@1.1.0` — the lock keyed by manifest name with the
  requested source coordinate recorded verbatim as a distinct field.
  **SPEC-057** succeeds the `REQ-021@1.0.0` pin, delivering `REQ-021@1.1.0` —
  the content hash and the installed tree covering authored content only, with
  root repository metadata neither copied nor hashed.

  This spec's own `REQ-020@1.0.0` and `REQ-021@1.0.0` pins are the HISTORICAL
  RECORD and are deliberately left as written: they describe the algorithm this
  spec actually evaluated, which included root `.git` in the content hash. The
  bundle is explicit that the pin must not be rewritten in place, so it was not.
  Retirement is the mechanism that keeps the record intact while removing it
  from enforcement — terminal specs are excluded from gate enforcement, which
  clears the `requirement_traceability` violations these two superseded pins
  produce.

  Its two ancestor-removed mandated test names,
  `TestPackAdd_AlreadyInstalledExitsNonZero` and
  `TestPackAdd_LocalPathNotClonedToPacksDir`, retire with it: both name behavior
  whose call sites no longer exist after the SPEC-055/056 constructor and
  identity work, and neither has a migrated form to point at. Recorded openly
  per align-predating-artifacts.
- **1.0.0** (2026-04-14) — Initial spec: the six pack lifecycle commands, the
  `backstop.lock` format, SHA-256 content hashing over every file, gate-time
  lock verification, tool_config provenance, and `.backstop/packs/` lifecycle
  management.
