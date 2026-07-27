---
title: "Remote Identity Version Validation"
number: SPEC-056
created: "2026-07-26"
status: draft
schema_version: spec/v1
spec_version: 1.1.0

implementation:
  summary: >
    BUNDLE-006's IDENTITY seed (REQ-039@1.1.0 / DD-26 / DD-31), authored against
    the tree SPEC-055 left behind. SPEC-055 made remote lifecycle commands
    ASSEMBLABLE — real `ExecGitCloner`, real `PackvalValidator`, real
    `TagVersionResolver`, fail-closed positional constructors — and named
    source-coordinate-versus-manifest-identity as explicitly OUT of its scope.
    That is what lands here. Today `AddCommand.Run` (`command.go:120`) takes the
    git branch's pack name straight from `parsePackRef`, i.e. from the requested
    repository COORDINATE, and then uses that one string for the install path
    (`command.go:162`), the backstop.yml key (`add.go:158`), and the lock key
    (`command.go:251`) — while gate dispatch resolves a pack's rules, producers,
    converters, and validators under its MANIFEST name. When the two differ the
    add reports success and the gate later fails looking for assets that were
    installed somewhere else; that is the exact `missing convert script` failure
    the harness consumer hit with a pack legitimately named
    `backstop/harness-toolchain` living at
    `backstop-ai/backstop-harness-toolchain-pack`. Nothing checks that the
    manifest's declared semantic version has anything to do with the tag that was
    cloned either, and a live counterexample is already published: the harness
    toolchain pack's manifest declares `0.1.3` while its tags stop at `v0.1.1`
    (DIR-027 item 2). Six things land. (1) An identity GATE — a new
    `pkg/pack/distribution/identity.go` — that resolves exactly one effective
    version, clones at it, reads the cloned tree's `pack.yml`, and refuses before
    any consumer state is touched when the manifest version does not equal the
    effective tag version after normalizing at most one leading `v`, when no
    version was resolvable at all, or when the manifest carries no usable name.
    Name validity comes from `pkg/pack`'s own rule, newly EXPORTED as
    `pack.ValidatePackName`, so there is one authority rather than a second copy;
    version strictness deliberately does NOT come from `pkg/pack`, whose
    `semverPattern` (`manifest.go:460`) accepts prerelease and build metadata that
    no strict release tag can equal. (2) The DD-31 split made real: the manifest
    name becomes the install path, the backstop.yml key, the lock key, and the
    engine asset root, while the requested `org/repository` is recorded VERBATIM
    in a new `source_coordinate` lock field — no case folding, no suffix
    stripping, because case-insensitivity is a GitHub property and packs may be
    hosted anywhere. (3) The consumer-side half that makes (2) survivable:
    install, update, upgrade, and version resolution stop deriving the repository
    URL from the pack name and read the recorded coordinate through ONE accessor
    instead, with a loud named fallback for lock entries written before the field
    existed. Without this a divergent-name pack becomes uninstallable from its own
    lock the moment (2) lands; without the single accessor the fallback either
    double-emits on `pack update` (which resolves the coordinate twice — once for
    `ls-remote`, once for the clone) or gets reimplemented inline. (4) Divergence
    itself is a LOUD diagnostic and never a refusal — OQ-9 resolved to option (b),
    and option (a) (require coordinate == manifest name) was rejected by name; the
    ten packs published under `backstop-ai` on 2026-07-26 hold
    `name == coordinate` as a fleet CONVENTION, and a convention is something you
    notice breaking, not something the tool enforces. (5) The carriage those
    diagnostics need, which does not exist today: `UpdateResult`
    (`update.go:28-34`) and `UpgradeResult` (`upgrade.go:22-29`) have no warning
    field at all and their CLI commands render nothing of the kind, so a warning
    computed inside update or upgrade would be silently dropped. All four result
    types gain `Warnings []string` and all four CLI commands render it to stderr.
    (6) The validate/hash ordering PIN: `pack check`/`pack test` mutate the
    directory they validate — `pkg/packval/phase3.go` renders every
    `tier: complete` scaffold's `sample_config` entries into
    `<packDir>/<scaffold.path>/` before running the scaffold test — and all three
    of add (`command.go:154`), update (`command.go:539`), and upgrade
    (`command.go:654`) validate a tree in place and then copy and hash THAT tree,
    while `InstallCommand` never validates at all. For any pack with such a
    scaffold, every one of those three commands records a hash a fresh clone
    cannot reproduce. Validation therefore moves onto a SCRATCH COPY in all three;
    the tree that is copied to the install path and handed to `ComputeContentHash`
    is the pristine materialized tree. OUT OF SCOPE, and left to the later
    BUNDLE-006 seeds: the shared staged-filesystem transaction coordinator and
    rollback for failures AFTER the identity gate (REQ-040@1.1.0); the legacy
    Git-metadata-inclusive hash migration and `pack relock`'s argument shape
    (REQ-041@1.1.0 / ISSUE-074, demoted because DIR-027's remove+re-add fleet
    migration writes fresh lock entries); the general authored-content copy/hash
    boundary for ALL sources including local-path `.git` exclusion
    (REQ-021@1.1.0 — this spec closes only the validator-contamination half and
    deliberately does not claim the requirement); the hermetic lifecycle parity
    suite (REQ-042@1.1.0); and `resolveGitURL`'s hardcoded GitHub host (ISSUE-083,
    post-launch, unhomed pending a founder decision on the resolution model).
  subject: pkg/pack/distribution

verification:
  level: integration
  test_command: go test ./pkg/pack/distribution/... ./cmd/backstop/... ./pkg/packval/... ./pkg/pack/... -race -coverprofile=cover.out
  coverage_threshold: 80

requirements:
  - id: REQ-001
    text: >
      A remote lifecycle command must resolve exactly ONE effective version before
      it invokes git. The effective version is the `--version` flag when supplied,
      otherwise the `@version` suffix on the pack reference; when both are supplied
      and disagree the flag wins, matching its documented "overrides version in
      pack reference" contract. The effective version must be a strict
      `MAJOR.MINOR.PATCH` release version after normalizing at most ONE leading
      `v`; the normalized form is what is recorded, and the ref cloned is that
      form re-prefixed with a single `v`. A reference carrying no version and no
      `--version` flag, a version that is not strict `X.Y.Z` (including a
      prerelease or build-metadata suffix), and a doubled prefix such as
      `vv1.0.0` must each refuse with a typed `*VersionUnresolvedError` naming the
      reference and both ways to supply a version, BEFORE any git subprocess runs.
      Today no such check exists: `parsePackRef` returns an empty version for a
      bare `org/name` and the pipeline clones the ref `"v"`, so the operator's
      diagnostic is a git error about a nonexistent branch.
    supports:
      - pack-distribution-lifecycle:REQ-039@1.1.0
      - pack-distribution-lifecycle:REQ-028@1.0.0
    follows: STD-GO-001:GO-010
  - id: REQ-002
    text: >
      After the clone and BEFORE any consumer state is mutated, a remote lifecycle
      command that clones a specific tag must read the cloned tree's `pack.yml`
      and require its declared `version` to EQUAL the effective version, comparing
      the two after normalizing at most one leading `v` from EACH side. A
      mismatch must return a typed `*VersionMismatchError` naming the requested
      coordinate, the tag that was cloned, the manifest version found, and the
      effective version expected. A manifest that is absent or unparseable, that
      declares no `version`, or whose `version` is not strict `X.Y.Z` after that
      single-prefix normalization must return a typed `*IdentityError` naming the
      coordinate, the tag, and the offending field. This strictness is
      DELIBERATELY narrower than `pkg/pack`'s manifest validation, whose
      `semverPattern` accepts prerelease and build-metadata suffixes: a manifest
      declaring `1.0.0-rc1` is a valid manifest but is not installable by tag,
      because no strict release tag can equal it, so identity must reject it
      rather than inherit `pack.validateSemver`. This check applies to `pack add`,
      `pack update`, and `pack upgrade` — every command that clones a tag it
      intends to install. It must NOT apply to `pack install`, which is the
      hash-verified restore path and re-validates nothing (DD-12): an install
      whose content hash matches the lock must succeed even when the remote
      manifest's declared version disagrees with the locked version.
    supports: pack-distribution-lifecycle:REQ-039@1.1.0
    follows: STD-GO-001:GO-011
  - id: REQ-003
    text: >
      The manifest `name` — never the requested repository coordinate — must be
      the pack's install/runtime identity. It must determine the install path
      under `.backstop/packs/`, the key written into backstop.yml's `packs:` map,
      the backstop.lock entry key and its `name` field, and therefore the engine
      asset root under which the gate resolves the pack's rules, producers,
      converters, and validators. A git-source add must read that name from the
      cloned tree rather than deriving it from `parsePackRef`. Name validity must
      be decided by `pkg/pack`'s existing rule rather than by a second copy of it:
      the unexported `validateName` (`manifest.go:462`) must be exported as
      `pack.ValidatePackName(name string) error` and called from distribution,
      which is cycle-safe because `pkg/pack` imports nothing from
      `pkg/pack/distribution`. A manifest whose `name` is empty, absent, or
      rejected by that rule — no slash, an empty part, or a part containing
      characters outside `[A-Za-z0-9-]` — must refuse with a typed
      `*IdentityError` before mutation, because a pack that cannot be addressed
      cannot be installed. A local-path add already takes its identity from the
      manifest and must keep doing so, through this same shared resolution.
    supports:
      - pack-distribution-lifecycle:REQ-039@1.1.0
      - pack-distribution-lifecycle:REQ-020@1.1.0
    follows: STD-GO-001:GO-010
  - id: REQ-004
    text: >
      `LockEntry` must carry a `source_coordinate` field recording the requested
      `org/repository` reference EXACTLY as the operator wrote it, with the
      `@version` suffix removed and nothing else changed: no case folding, no
      suffix stripping, no host-specific normalization of any kind. It must be
      written for git-source packs and left empty for local-source packs, whose
      source is already recorded by `local_path`. It must be emitted in the
      lockfile's alphabetical key order, between `name` and `source_type`, and
      omitted entirely when empty so a lock entry written before this field
      existed parses, round-trips, and re-serializes without gaining a blank key.
      `pack update` and `pack upgrade` must PRESERVE the recorded coordinate when
      they rewrite an entry rather than re-deriving it from the pack name, and
      `pack relock` must leave it, like every other field it does not refresh,
      exactly as it found it.
    supports:
      - pack-distribution-lifecycle:REQ-039@1.1.0
      - pack-distribution-lifecycle:REQ-020@1.1.0
    follows: STD-GO-001:GO-010
  - id: REQ-005
    text: >
      Every remote operation performed on an ALREADY-LOCKED pack must resolve its
      repository coordinate from that pack's recorded `source_coordinate`, not
      from its pack name or lock key. There must be exactly ONE accessor —
      `CoordinateForEntry(packName, entry)` returning the coordinate and, when it
      had to fall back, a warning — and the URL builder `RemoteURLForEntry` must
      be layered on it rather than duplicating the fallback. This matters because
      the two consumers need different things from the same decision:
      `TagVersionResolver` builds its own URL from a coordinate
      (`versionresolver.go:70`) while the clone paths need a URL, so `pack update`
      resolves the coordinate TWICE in one invocation. The fallback warning must
      therefore be emitted exactly ONCE per command invocation per pack. When an
      entry carries no `source_coordinate` — the shape every lock entry written
      before this spec has — the command must fall back to the pack name AND warn,
      naming the pack and telling the operator to re-add or relock it; the
      fallback is a compatibility path, not the primary one, and must never be
      silent. A LOCAL-source entry has no remote at all, so a command must
      establish that an entry is a git source BEFORE resolving a coordinate:
      `pack update` already no-ops on a local pack, and `pack upgrade` — which
      today discards `readPackVersion`'s `isLocal` result (`command.go:636`) and
      clones unconditionally — must gain the same guard and refuse a local-source
      pack with a diagnostic naming `pack relock` as the local path forward.
      Without that guard the fallback warning fires on a source that has no
      repository. Without this requirement as a whole, recording the coordinate is
      decorative: the moment REQ-003 keys the lock by manifest name, any pack
      whose name differs from its repository becomes uninstallable from its lock.
    supports: pack-distribution-lifecycle:REQ-039@1.1.0
    follows: STD-GO-001:GO-011
  - id: REQ-006
    text: >
      When the manifest name and the requested coordinate DIFFER, the command must
      emit a LOUD diagnostic naming all three of the requested coordinate, the
      manifest name, and the install path the manifest name resolves to — and must
      then PROCEED. Divergence must never be a refusal and must never change the
      exit code of an otherwise-successful command: OQ-9 resolved to option (b)
      and rejected option (a) (require coordinate == manifest name) by name,
      because forcing equality would outlaw a naming convention packs legitimately
      use. Comparison is byte-exact on the full `org/name` strings with no case
      folding, for the same reason the coordinate is recorded verbatim. When the
      two are EQUAL no diagnostic may be emitted, so silence is the signal that
      the `name == coordinate` fleet convention ratified 2026-07-26 still holds.
      The diagnostic must be produced by every command that runs the identity
      gate — add, update, and upgrade — and carried out of each of them by the
      mechanism REQ-011 defines, because a warning a result type cannot hold is a
      warning nobody sees.
    supports: pack-distribution-lifecycle:REQ-039@1.1.0
    follows: STD-GO-001:GO-011
  - id: REQ-007
    text: >
      Every check in REQ-001 through REQ-003, and REQ-006's diagnostic, must
      complete before the FIRST byte of consumer state is written. For `pack add`
      the consumer-state surfaces are exactly: the installed tree under
      `.backstop/packs/`, `backstop.yml`, `backstop.lock`,
      `.backstop/pack-config-provenance.json`, any tool-config file the pack's
      `tool_config` merges into, and `.gitignore`. For `pack update` they are the
      installed tree, `backstop.yml`, and `backstop.lock`; for `pack upgrade` they
      are those three plus provenance, tool config, and any remediation artifact.
      On any identity or version refusal none of them may be created, modified, or
      removed, the command must exit non-zero, and the temporary clone and scratch
      directories must be cleaned up. The identity gate must also run BEFORE `pack
      check` and `pack test`, so a pack that is both identity-invalid and
      validation-failing reports the identity diagnostic — the cheaper, more
      specific failure. This requirement covers the ordering only; exact rollback
      for failures occurring AFTER the gate is bundle REQ-040@1.1.0's staged
      transaction coordinator and is explicitly not delivered here.
    supports: pack-distribution-lifecycle:REQ-039@1.1.0
    follows: STD-GO-001:GO-011
  - id: REQ-008
    text: >
      Pack validation must not be able to contaminate the content that is
      installed or hashed, in ANY command that validates. `pkg/packval` MUTATES
      the directory it validates: phase 3 renders each `tier: complete` scaffold's
      declared `sample_config` entries into `<packDir>/<scaffold.path>/<relPath>`
      before running that scaffold's test command. All three of `AddCommand.Run`,
      `UpdateCommand.Run`, and `UpgradeCommand.Run` currently validate a tree in
      place and then copy and hash that same tree, so the defect shape is
      identical in each. Therefore `RunPackCheck` and `RunPackTest` must be
      invoked against a SCRATCH COPY of the materialized pack tree in all three,
      and the tree that is copied to the install path and passed to
      `ComputeContentHash` must be the pristine materialized tree that no
      validator has written to. The scratch copy must be removed on both the
      success and the failure path. This applies to local-path sources as well,
      where the validated directory is the operator's own working tree and
      in-place validation writes files into it. Because `runPackvalPipeline`
      quotes the directory it was handed (`validator.go:66-71`), a validation
      failure must be reported against the ORIGINAL source — the coordinate and
      tag for a remote pack, the local path for a local one — and must never quote
      the scratch temporary directory at an operator. The observable contract is
      that the hash `pack add` records for a pack equals the hash `pack install`
      computes from a fresh clone of the same tag, for a pack that declares such a
      scaffold. This closes the validator-contamination half of the
      authored-content boundary only. It is pinned to REQ-039@1.1.0 rather than to
      REQ-021@1.1.0 DELIBERATELY: REQ-021 additionally owns `.git` exclusion for
      every source and the legacy-lock migration, and claiming support for it here
      would mark it covered at its current version in the bundle traceability
      gate while two thirds of it remain unbuilt.
    supports: pack-distribution-lifecycle:REQ-039@1.1.0
    follows: STD-GO-001:GO-010
  - id: REQ-009
    text: >
      Every refusal introduced by this spec must be a typed error carrying named
      fields rather than a formatted string, must render a human-readable
      diagnostic on stderr, and must be classified under `--json` into a stable
      wire kind: `*VersionUnresolvedError` and `*VersionMismatchError` as
      `version`, `*IdentityError` as `identity`, added alongside the existing
      `git`, `validation`, `dependency`, `capability`, and `unknown` kinds. No
      such refusal may exit non-zero with an empty stderr, and none may panic or
      emit a Go stack trace. This extends the error-surfacing and JSON-error
      mechanisms SPEC-055 built (`reportError`, `classifyJSONErrorKind`) rather
      than adding a second one.
    supports:
      - pack-distribution-lifecycle:REQ-039@1.1.0
      - pack-distribution-lifecycle:REQ-001@1.0.0
    follows: STD-GO-001:GO-011
  - id: REQ-010
    text: >
      The hermetic remote harness must be able to express the shapes this spec's
      failure and divergence claims need, without network access and without a
      second harness. It must gain a way to create tags WITHOUT rewriting the
      manifest's declared version — today `newHermeticRemote` rewrites
      `pack.yml`'s version to match each tag, which makes a tag-versus-manifest
      divergence inexpressible — and the fixture corpus must gain three
      tagged-repository pack fixtures: one whose manifest version disagrees with
      every tag (the published harness pack's shape: manifest `0.1.3`, tags
      `v0.1.0` and `v0.1.1`), one whose manifest name differs from its repository
      directory name, and one declaring a `tier: complete` scaffold with
      `sample_config`. The hermetic property required of all three is: no network
      access and no toolchain or provisioned binary. It is NOT "no external
      process" — `DefaultExecutor.RunScaffoldTest` runs `exec.Command("sh", "-c",
      testCommand)` unconditionally for a `tier: complete` scaffold
      (`executor.go:107-112`, reached from `phase3.go:173`) and `PackvalValidator`
      supplies no executor, so a shell subprocess is unavoidable on that path. The
      scaffold fixture's `test_command` must therefore be restricted to a POSIX
      shell builtin that reaches nothing outside the process. No `pack.yml` in the
      repository declares a `tier: complete` scaffold today — every existing
      scaffold declaration is `tier: skeleton` — so this fixture is the corpus's
      first, and it must pass `pack check` and `pack test` on its own.
    supports:
      - pack-distribution-lifecycle:REQ-039@1.1.0
      - pack-distribution-lifecycle:REQ-032@1.0.0
    follows: STD-GO-001:GO-010
  - id: REQ-011
    text: >
      Every warning this spec produces must have a carrier on the result and a
      renderer on the command, because neither exists for two of the four
      commands today: `UpdateResult` and `UpgradeResult` declare no warning field
      at all, and `pack update` and `pack upgrade` render nothing of the kind —
      only `pack install` does (`pack_install.go:32`). `AddResult`,
      `UpdateResult`, and `UpgradeResult` must therefore each gain
      `Warnings []string`, matching `InstallResult`'s existing field, and all four
      CLI commands must render every entry to STDERR, not stdout, so a warning is
      a diagnostic rather than part of a command's output and a stream-separated
      assertion can tell them apart. Rendering a warning must never change a
      command's exit code. Warnings ride the result rather than an injected writer
      because distribution owns no output stream and must not acquire a dependency
      to carry a string. These fields are the ONLY mechanism the divergence
      diagnostic (REQ-006) and the coordinate fallback (REQ-005) have; a field
      addition is invisible to the contracts gate, which checks symbol existence
      rather than struct shape, so each carrier and each renderer is claimed
      individually and the claims are the enforcement.
    supports:
      - pack-distribution-lifecycle:REQ-039@1.1.0
      - pack-distribution-lifecycle:REQ-001@1.0.0
    follows: STD-GO-001:GO-011

claims:
  # REQ-001 — one effective version, resolved before git runs
  - id: CLM-001
    requirement: REQ-001
    text: An @version suffix on the pack reference supplies the effective version and the cloned ref is that version with a single v prefix
    tests:
      - TestResolveEffectiveVersion_RefSuffixSuppliesVersion
  - id: CLM-002
    requirement: REQ-001
    text: The --version flag supplies the effective version when the pack reference carries none
    tests:
      - TestResolveEffectiveVersion_FlagSuppliesVersionWhenRefHasNone
  - id: CLM-003
    requirement: REQ-001
    text: The --version flag wins when it disagrees with the reference's @version suffix
    tests:
      - TestResolveEffectiveVersion_FlagOverridesDisagreeingRefSuffix
  - id: CLM-004
    requirement: REQ-001
    text: A reference suffix and a --version flag that agree resolve to that one version
    tests:
      - TestResolveEffectiveVersion_AgreeingRefAndFlagResolveOnce
  - id: CLM-005
    requirement: REQ-001
    text: A reference carrying no version and no --version flag returns a VersionUnresolvedError naming both ways to supply one
    tests:
      - TestResolveEffectiveVersion_NoVersionAnywhereIsTypedRefusal
  - id: CLM-006
    requirement: REQ-001
    text: A version that is not strict MAJOR.MINOR.PATCH returns a VersionUnresolvedError
    tests:
      - TestResolveEffectiveVersion_NonStrictVersionIsTypedRefusal
  - id: CLM-007
    requirement: REQ-001
    text: A prerelease version suffix returns a VersionUnresolvedError rather than resolving to an unreleased tag
    tests:
      - TestResolveEffectiveVersion_PrereleaseVersionIsTypedRefusal
  - id: CLM-008
    requirement: REQ-001
    text: A doubled v prefix returns a VersionUnresolvedError, because only one prefix character is normalization
    tests:
      - TestResolveEffectiveVersion_DoubledPrefixIsTypedRefusal
  - id: CLM-009
    requirement: REQ-001
    text: A single leading v normalizes away, so the recorded version carries no prefix and the cloned ref carries exactly one
    tests:
      - TestResolveEffectiveVersion_SingleLeadingPrefixNormalizes
  - id: CLM-010
    requirement: REQ-001
    text: Add refuses an unresolvable version before invoking the cloner at all, proven by a cloner double that records every invocation
    tests:
      - TestAddCommand_Run_UnresolvableVersionRefusesBeforeCloning
  - id: CLM-011
    requirement: REQ-001
    subject: cmd/backstop
    text: pack add of a bare org/name with no version exits non-zero with the diagnostic on stderr and no git error about a nonexistent branch
    tests:
      - TestE2E_PackAdd_VersionlessReferenceRefusesWithGuidance

  # REQ-002 — manifest version must equal the effective tag version
  - id: CLM-012
    requirement: REQ-002
    text: A v-prefixed tag whose manifest declares the unprefixed same version passes the equality check
    tests:
      - TestValidateRemoteIdentity_PrefixedTagUnprefixedManifestPasses
  - id: CLM-013
    requirement: REQ-002
    text: An unprefixed effective version whose manifest declares the same unprefixed version passes the equality check
    tests:
      - TestValidateRemoteIdentity_UnprefixedBothSidesPasses
  - id: CLM-014
    requirement: REQ-002
    text: A manifest that declares its version with a leading v passes against the same tag, because one prefix normalizes on each side
    tests:
      - TestValidateRemoteIdentity_PrefixedManifestPasses
  - id: CLM-015
    requirement: REQ-002
    text: A manifest version differing from the effective version returns a VersionMismatchError naming coordinate, tag, manifest version, and expected version
    tests:
      - TestValidateRemoteIdentity_ManifestVersionMismatchIsTypedRefusal
  - id: CLM-016
    requirement: REQ-002
    text: The published harness pack's shape — manifest 0.1.3 against tags v0.1.0 and v0.1.1 — refuses when v0.1.1 is requested
    tests:
      - TestValidateRemoteIdentity_HarnessPackVersionDriftRefuses
  - id: CLM-017
    requirement: REQ-002
    text: A manifest declaring no version returns an IdentityError naming the missing field
    tests:
      - TestValidateRemoteIdentity_MissingManifestVersionIsIdentityError
  - id: CLM-018
    requirement: REQ-002
    text: A manifest version that is not strict MAJOR.MINOR.PATCH returns an IdentityError
    tests:
      - TestValidateRemoteIdentity_NonStrictManifestVersionIsIdentityError
  - id: CLM-019
    requirement: REQ-002
    text: A prerelease manifest version returns an IdentityError even though pkg/pack's manifest validation accepts it, because no strict release tag can equal it
    tests:
      - TestValidateRemoteIdentity_PrereleaseManifestVersionIsIdentityError
  - id: CLM-020
    requirement: REQ-002
    text: A manifest version carrying a doubled v prefix returns an IdentityError
    tests:
      - TestValidateRemoteIdentity_DoubledPrefixManifestVersionIsIdentityError
  - id: CLM-021
    requirement: REQ-002
    text: A cloned tree with an absent or unparseable pack.yml returns an IdentityError naming the coordinate and the tag
    tests:
      - TestValidateRemoteIdentity_UnreadableManifestIsIdentityError
  - id: CLM-022
    requirement: REQ-002
    subject: cmd/backstop
    text: pack add against a hermetic repository whose manifest version disagrees with the requested tag exits non-zero with the mismatch on stderr and no success line on stdout
    tests:
      - TestE2E_PackAdd_ManifestVersionDriftRefusesLoudly
  - id: CLM-023
    requirement: REQ-002
    text: Update refuses when the resolved tag's manifest version disagrees with the resolved version
    tests:
      - TestUpdateCommand_Run_ResolvedTagVersionDriftRefuses
  - id: CLM-024
    requirement: REQ-002
    text: Upgrade refuses when the target major tag's manifest version disagrees with the requested target version
    tests:
      - TestUpgradeCommand_Run_TargetTagVersionDriftRefuses
  - id: CLM-025
    requirement: REQ-002
    text: Install succeeds on a hash-matching entry even when the remote manifest's declared version disagrees with the locked version, because install does not re-validate identity
    tests:
      - TestInstallCommand_Run_DoesNotApplyManifestVersionEquality
  - id: CLM-026
    requirement: REQ-002
    text: Update of a local-source pack remains a no-op and runs no identity gate, because a local source has no tag to disagree with
    tests:
      - TestUpdateCommand_Run_LocalSourceRemainsNoOpWithoutIdentityGate
  - id: CLM-027
    requirement: REQ-002
    text: Upgrade of a local-source pack is refused by the new local guard before the identity gate or any clone is reached
    tests:
      - TestUpgradeCommand_Run_LocalSourceRefusedBeforeIdentityGate

  # REQ-003 — the manifest name is the install/runtime identity
  - id: CLM-028
    requirement: REQ-003
    text: A git add whose manifest name differs from its coordinate installs under the manifest name, not under the coordinate
    tests:
      - TestAddCommand_Run_InstallPathComesFromManifestName
  - id: CLM-029
    requirement: REQ-003
    text: The backstop.yml packs key written by a divergent git add is the manifest name
    tests:
      - TestAddCommand_Run_ManifestKeyComesFromManifestName
  - id: CLM-030
    requirement: REQ-003
    text: The backstop.lock entry key and its name field for a divergent git add are both the manifest name
    tests:
      - TestAddCommand_Run_LockKeyComesFromManifestName
  - id: CLM-031
    requirement: REQ-003
    subject: cmd/backstop
    text: After a divergent add the gate resolves the pack's declared rule file under the manifest-name asset root and reports no missing asset
    tests:
      - TestE2E_GateResolvesPackAssetsUnderManifestName
  - id: CLM-032
    requirement: REQ-003
    text: A git add whose manifest name equals its coordinate installs under that one name, keeping the fleet convention's behavior unchanged
    tests:
      - TestAddCommand_Run_EqualNameAndCoordinateInstallsUnchanged
  - id: CLM-033
    requirement: REQ-003
    text: A manifest with an empty name returns an IdentityError before any consumer state is written
    tests:
      - TestValidateRemoteIdentity_EmptyManifestNameIsIdentityError
  - id: CLM-034
    requirement: REQ-003
    text: A manifest name carrying no slash returns an IdentityError
    tests:
      - TestValidateRemoteIdentity_UnqualifiedManifestNameIsIdentityError
  - id: CLM-035
    requirement: REQ-003
    text: A manifest name with an empty org or pack part returns an IdentityError
    tests:
      - TestValidateRemoteIdentity_EmptyNamePartIsIdentityError
  - id: CLM-036
    requirement: REQ-003
    text: A manifest name containing characters outside the pack identifier set returns an IdentityError
    tests:
      - TestValidateRemoteIdentity_InvalidNameCharactersIsIdentityError
  - id: CLM-037
    requirement: REQ-003
    subject: pkg/pack
    text: pack.ValidatePackName is exported and applies exactly the rule the unexported validateName applied, so distribution has one authority rather than a copy
    tests:
      - TestValidatePackName_ExportsTheExistingNameRule
  - id: CLM-038
    requirement: REQ-003
    text: A local-path add takes its identity from the manifest name through the same shared resolution, with no second implementation
    tests:
      - TestAddCommand_Run_LocalPathIdentityComesFromSharedResolution
  - id: CLM-039
    requirement: REQ-003
    text: Remove of a divergent-name pack by its manifest name removes the installed tree, the backstop.yml entry, and the lock entry, because all three are keyed by that one identity
    tests:
      - TestRemove_DivergentNamePackRemovesEverySurfaceByManifestName

  # REQ-004 — the source coordinate, recorded verbatim
  - id: CLM-040
    requirement: REQ-004
    text: A git add records the requested org/repository in the lock entry's source_coordinate field
    tests:
      - TestAddCommand_Run_RecordsSourceCoordinate
  - id: CLM-041
    requirement: REQ-004
    text: A mixed-case coordinate is recorded byte-for-byte as the operator typed it, with no case folding
    tests:
      - TestAddCommand_Run_RecordsMixedCaseCoordinateVerbatim
  - id: CLM-042
    requirement: REQ-004
    text: A coordinate whose repository name carries a -pack suffix is recorded with the suffix intact
    tests:
      - TestAddCommand_Run_RecordsSuffixedCoordinateVerbatim
  - id: CLM-043
    requirement: REQ-004
    text: The @version suffix is stripped from the recorded coordinate and nothing else is
    tests:
      - TestAddCommand_Run_RecordedCoordinateExcludesVersionSuffix
  - id: CLM-044
    requirement: REQ-004
    text: A local-path add records no source_coordinate, because local_path already records its source
    tests:
      - TestAddCommand_Run_LocalSourceRecordsNoCoordinate
  - id: CLM-045
    requirement: REQ-004
    text: A lock entry carrying source_coordinate round-trips through read and write unchanged
    tests:
      - TestLockfile_SourceCoordinateRoundTrips
  - id: CLM-046
    requirement: REQ-004
    text: A legacy lock entry with no source_coordinate parses and re-serializes without gaining a blank key
    tests:
      - TestLockfile_LegacyEntryWithoutCoordinateRoundTripsUnchanged
  - id: CLM-047
    requirement: REQ-004
    text: source_coordinate is emitted between name and source_type in the lockfile's sorted key order
    tests:
      - TestLockfile_SourceCoordinateSortsBetweenNameAndSourceType
  - id: CLM-048
    requirement: REQ-004
    text: Update preserves the recorded coordinate when it rewrites a lock entry
    tests:
      - TestUpdateCommand_Run_PreservesRecordedCoordinate
  - id: CLM-049
    requirement: REQ-004
    text: Upgrade preserves the recorded coordinate when it rewrites a lock entry
    tests:
      - TestUpgradeCommand_Run_PreservesRecordedCoordinate
  - id: CLM-050
    requirement: REQ-004
    text: Relock refreshes a local entry's content hash and install date and leaves every other field, source_coordinate included, exactly as it found them
    tests:
      - TestRelock_PreservesEveryFieldItDoesNotRefresh

  # REQ-005 — the recorded coordinate is what later remote operations resolve
  - id: CLM-051
    requirement: REQ-005
    text: CoordinateForEntry returns the recorded coordinate and no warning when the entry carries one
    tests:
      - TestCoordinateForEntry_RecordedCoordinateReturnsNoWarning
  - id: CLM-052
    requirement: REQ-005
    text: CoordinateForEntry falls back to the pack name and returns a warning naming the pack and the remedy when no coordinate is recorded
    tests:
      - TestCoordinateForEntry_MissingCoordinateFallsBackWithWarning
  - id: CLM-053
    requirement: REQ-005
    text: RemoteURLForEntry builds its URL from the coordinate CoordinateForEntry returns, so the resolver and the cloner cannot disagree about which repository a pack came from
    tests:
      - TestRemoteURLForEntry_BuildsURLFromTheSharedCoordinate
  - id: CLM-054
    requirement: REQ-005
    text: Install clones a divergent-name pack from its recorded coordinate rather than from its lock key
    tests:
      - TestInstallCommand_Run_ClonesFromRecordedCoordinate
  - id: CLM-055
    requirement: REQ-005
    text: Update lists tags at the recorded coordinate rather than at the pack name
    tests:
      - TestUpdateCommand_Run_ResolvesTagsAtRecordedCoordinate
  - id: CLM-056
    requirement: REQ-005
    text: Update clones at the recorded coordinate rather than at the pack name
    tests:
      - TestUpdateCommand_Run_ClonesFromRecordedCoordinate
  - id: CLM-057
    requirement: REQ-005
    text: Upgrade clones at the recorded coordinate rather than at the pack name
    tests:
      - TestUpgradeCommand_Run_ClonesFromRecordedCoordinate
  - id: CLM-058
    requirement: REQ-005
    text: ResolveLatestCompatible builds its repository URL from the coordinate it is passed
    tests:
      - TestTagVersionResolver_ResolveLatestCompatible_UsesSuppliedCoordinate
  - id: CLM-059
    requirement: REQ-005
    text: Update resolves the coordinate twice in one invocation — once to list tags, once to clone — and emits the fallback warning exactly once
    tests:
      - TestUpdateCommand_Run_FallbackWarningEmittedOncePerInvocation
  - id: CLM-060
    requirement: REQ-005
    text: Install of a local-source entry materializes from local_path, resolves no remote coordinate, and emits no fallback warning despite carrying none
    tests:
      - TestInstallCommand_Run_LocalSourceNeedsNoCoordinate
  - id: CLM-061
    requirement: REQ-005
    text: Upgrade of a local-source pack resolves no coordinate and emits no fallback warning, because the local guard refuses first
    tests:
      - TestUpgradeCommand_Run_LocalSourceResolvesNoCoordinate
  - id: CLM-062
    requirement: REQ-005
    subject: cmd/backstop
    text: A divergent-name pack added from a hermetic repository installs green into a fresh consumer carrying only the manifest and lock, with matching content hashes
    tests:
      - TestE2E_DivergentNamePack_AddThenInstallRoundTrip

  # REQ-006 — divergence is loud and never fatal
  - id: CLM-063
    requirement: REQ-006
    text: A divergent add succeeds and reports the divergence rather than refusing
    tests:
      - TestAddCommand_Run_DivergentIdentitySucceedsWithWarning
  - id: CLM-064
    requirement: REQ-006
    text: The divergence diagnostic names the requested coordinate, the manifest name, and the resolved install path
    tests:
      - TestAddCommand_Run_DivergenceWarningNamesAllThreeIdentities
  - id: CLM-065
    requirement: REQ-006
    text: An add whose manifest name equals its coordinate emits no divergence diagnostic
    tests:
      - TestAddCommand_Run_EqualIdentitiesEmitNoWarning
  - id: CLM-066
    requirement: REQ-006
    text: Comparison is byte-exact, so coordinate and manifest name differing only in letter case are reported as divergent
    tests:
      - TestAddCommand_Run_CaseOnlyDifferenceCountsAsDivergent
  - id: CLM-067
    requirement: REQ-006
    text: Divergence alone never refuses — the pack installs and the lock records the manifest name and the coordinate independently
    tests:
      - TestAddCommand_Run_DivergenceRecordsBothIdentities
  - id: CLM-068
    requirement: REQ-006
    text: Update of a divergent-name pack carries the divergence diagnostic out on its result
    tests:
      - TestUpdateCommand_Run_CarriesDivergenceWarning
  - id: CLM-069
    requirement: REQ-006
    text: Upgrade of a divergent-name pack carries the divergence diagnostic out on its result
    tests:
      - TestUpgradeCommand_Run_CarriesDivergenceWarning
  - id: CLM-070
    requirement: REQ-006
    subject: cmd/backstop
    text: pack add of a divergent-name pack exits zero, writes the divergence diagnostic to stderr, and writes the success line to stdout
    tests:
      - TestE2E_PackAdd_DivergentIdentityWarnsOnStderrAndSucceeds

  # REQ-007 — nothing is written before the gate passes
  - id: CLM-071
    requirement: REQ-007
    text: A version-mismatch refusal leaves no directory under .backstop/packs for either the manifest name or the coordinate
    tests:
      - TestAddCommand_Run_IdentityRefusalWritesNoInstalledContent
  - id: CLM-072
    requirement: REQ-007
    text: A version-mismatch refusal leaves backstop.yml byte-identical to its pre-command content
    tests:
      - TestAddCommand_Run_IdentityRefusalLeavesManifestUntouched
  - id: CLM-073
    requirement: REQ-007
    text: A version-mismatch refusal leaves backstop.lock byte-identical, and leaves it absent when it did not exist
    tests:
      - TestAddCommand_Run_IdentityRefusalLeavesLockUntouched
  - id: CLM-074
    requirement: REQ-007
    text: A version-mismatch refusal writes no pack-config-provenance.json and modifies no tool-config file
    tests:
      - TestAddCommand_Run_IdentityRefusalWritesNoProvenanceOrToolConfig
  - id: CLM-075
    requirement: REQ-007
    text: A version-mismatch refusal leaves .gitignore byte-identical, and leaves it absent when it did not exist
    tests:
      - TestAddCommand_Run_IdentityRefusalLeavesGitignoreUntouched
  - id: CLM-076
    requirement: REQ-007
    text: A pack that is both identity-invalid and validation-failing reports the identity diagnostic, proving the gate runs before pack check and pack test
    tests:
      - TestAddCommand_Run_IdentityGatePrecedesValidation
  - id: CLM-077
    requirement: REQ-007
    text: An identity refusal removes its temporary clone and scratch directories
    tests:
      - TestAddCommand_Run_IdentityRefusalLeavesNoTemporaryDirectories
  - id: CLM-078
    requirement: REQ-007
    text: An update refused by the identity gate leaves the previously installed tree, backstop.yml, and backstop.lock exactly as they were
    tests:
      - TestUpdateCommand_Run_IdentityRefusalLeavesEverySurfaceUntouched
  - id: CLM-079
    requirement: REQ-007
    text: An upgrade refused by the identity gate leaves the installed tree, backstop.yml, backstop.lock, provenance, tool config, and any remediation artifact untouched
    tests:
      - TestUpgradeCommand_Run_IdentityRefusalLeavesEverySurfaceUntouched

  # REQ-008 — validation may not contaminate installed or hashed content
  - id: CLM-080
    requirement: REQ-008
    subject: pkg/packval
    text: Phase 3 renders a tier complete scaffold's sample_config into the validated pack directory, characterizing the mutation this requirement exists to contain
    tests:
      - TestRunFixtures_CompleteScaffoldWritesSampleConfigIntoPackDir
  - id: CLM-081
    requirement: REQ-008
    text: On add, a validator that writes a file into the directory it validates leaves that file out of the installed tree
    tests:
      - TestAddCommand_Run_ValidatorWritesDoNotReachInstalledContent
  - id: CLM-082
    requirement: REQ-008
    text: On add, the recorded content hash is computed over the pristine materialized tree and is unchanged by a validator that writes into what it validates
    tests:
      - TestAddCommand_Run_ContentHashExcludesValidatorWrites
  - id: CLM-083
    requirement: REQ-008
    text: On update, validator writes reach neither the replaced installed tree nor the recorded content hash
    tests:
      - TestUpdateCommand_Run_ValidatorWritesReachNeitherContentNorHash
  - id: CLM-084
    requirement: REQ-008
    text: On upgrade, validator writes reach neither the replaced installed tree nor the recorded content hash
    tests:
      - TestUpgradeCommand_Run_ValidatorWritesReachNeitherContentNorHash
  - id: CLM-085
    requirement: REQ-008
    text: The scratch validation copy is removed on the success path
    tests:
      - TestRunValidationOnScratchCopy_RemovesScratchOnSuccess
  - id: CLM-086
    requirement: REQ-008
    text: The scratch validation copy is removed when validation fails, and the failure still leaves no installed content and no lock entry
    tests:
      - TestAddCommand_Run_ScratchCopyRemovedOnValidationFailure
  - id: CLM-087
    requirement: REQ-008
    text: A local-path add leaves the operator's source directory free of validator-authored files
    tests:
      - TestAddCommand_Run_LocalSourceDirectoryNotMutatedByValidation
      - TestPackAdd_LocalPathValidatesInPlace
  - id: CLM-088
    requirement: REQ-008
    text: A remote validation failure names the coordinate and the tag and never quotes the scratch temporary directory
    tests:
      - TestRunValidationOnScratchCopy_RemoteFailureNamesCoordinateNotScratchPath
  - id: CLM-089
    requirement: REQ-008
    text: A local validation failure names the operator's source path and never quotes the scratch temporary directory
    tests:
      - TestRunValidationOnScratchCopy_LocalFailureNamesSourcePathNotScratchPath
  - id: CLM-090
    requirement: REQ-008
    subject: cmd/backstop
    text: A pack declaring a tier complete scaffold with sample_config adds and then installs into a fresh consumer with matching content hashes
    tests:
      - TestE2E_ScaffoldConfigPack_AddThenInstallHashesMatch

  # REQ-009 — typed, readable, machine-classifiable refusals
  - id: CLM-091
    requirement: REQ-009
    text: VersionUnresolvedError renders a diagnostic naming the reference and both ways to supply a version
    tests:
      - TestVersionUnresolvedError_ErrorNamesReferenceAndRemedy
  - id: CLM-092
    requirement: REQ-009
    text: VersionMismatchError renders a diagnostic naming coordinate, tag, manifest version, and expected version
    tests:
      - TestVersionMismatchError_ErrorNamesBothVersions
  - id: CLM-093
    requirement: REQ-009
    text: IdentityError renders a diagnostic naming the coordinate and the offending field
    tests:
      - TestIdentityError_ErrorNamesCoordinateAndField
  - id: CLM-094
    requirement: REQ-009
    subject: cmd/backstop
    text: A version-unresolved failure is classified under --json as kind version
    tests:
      - TestClassifyJSONErrorKind_VersionUnresolvedIsVersionKind
  - id: CLM-095
    requirement: REQ-009
    subject: cmd/backstop
    text: A version-mismatch failure is classified under --json as kind version
    tests:
      - TestClassifyJSONErrorKind_VersionMismatchIsVersionKind
  - id: CLM-096
    requirement: REQ-009
    subject: cmd/backstop
    text: An identity failure is classified under --json as kind identity
    tests:
      - TestClassifyJSONErrorKind_IdentityErrorIsIdentityKind
  - id: CLM-097
    requirement: REQ-009
    subject: cmd/backstop
    text: An identity or version refusal never exits non-zero with an empty stderr
    tests:
      - TestE2E_IdentityRefusalNeverExitsSilently
  - id: CLM-098
    requirement: REQ-009
    subject: cmd/backstop
    text: An identity or version refusal emits no Go stack trace on either stream
    tests:
      - TestE2E_IdentityRefusalEmitsNoStackTrace

  # REQ-010 — the hermetic fixtures and harness the claims above require
  - id: CLM-099
    requirement: REQ-010
    subject: cmd/backstop
    text: The harness can create tags without rewriting the manifest version, so a tag-versus-manifest divergence is expressible
    tests:
      - TestHermeticRemote_TagsWithoutRewritingManifestVersion
  - id: CLM-100
    requirement: REQ-010
    subject: cmd/backstop
    text: The version-drift fixture repository publishes v0.1.0 and v0.1.1 while its manifest declares 0.1.3
    tests:
      - TestHermeticFixture_VersionDriftPackHasDriftingManifest
  - id: CLM-101
    requirement: REQ-010
    subject: cmd/backstop
    text: The divergent-name fixture's manifest name differs from its repository directory name
    tests:
      - TestHermeticFixture_DivergentNamePackNameDiffersFromRepository
  - id: CLM-102
    requirement: REQ-010
    subject: cmd/backstop
    text: The scaffold-config fixture declares a tier complete scaffold whose sample_config target does not exist in the authored tree, so a rendered file is detectable
    tests:
      - TestHermeticFixture_ScaffoldConfigPackDeclaresUnauthoredSampleConfig
  - id: CLM-103
    requirement: REQ-010
    subject: cmd/backstop
    text: The scaffold fixture's test_command is a POSIX shell builtin, so the unavoidable sh subprocess reaches no toolchain binary and no network
    tests:
      - TestHermeticFixture_ScaffoldTestCommandIsAShellBuiltin
  - id: CLM-104
    requirement: REQ-010
    subject: cmd/backstop
    text: Every new fixture passes pack check and pack test on its own, including the corpus's first tier complete scaffold declaration
    tests:
      - TestHermeticFixture_NewFixturesPassPackCheckAndPackTest

  # REQ-011 — every warning has a carrier and a renderer
  - id: CLM-105
    requirement: REQ-011
    text: AddResult carries warnings out of the command, so a divergence diagnostic computed before mutation reaches the caller
    tests:
      - TestAddResult_CarriesWarnings
  - id: CLM-106
    requirement: REQ-011
    text: UpdateResult carries warnings out of the command, which it could not do before this spec
    tests:
      - TestUpdateResult_CarriesWarnings
  - id: CLM-107
    requirement: REQ-011
    text: UpgradeResult carries warnings out of the command, which it could not do before this spec
    tests:
      - TestUpgradeResult_CarriesWarnings
  - id: CLM-108
    requirement: REQ-011
    text: InstallResult continues to carry its existing reconciliation warnings alongside the new coordinate fallback warning
    tests:
      - TestInstallResult_CarriesReconciliationAndFallbackWarningsTogether
  - id: CLM-109
    requirement: REQ-011
    subject: cmd/backstop
    text: pack add renders every result warning to stderr
    tests:
      - TestPackAddCommand_RendersWarningsToStderr
  - id: CLM-110
    requirement: REQ-011
    subject: cmd/backstop
    text: pack install renders every result warning to stderr rather than to stdout
    tests:
      - TestPackInstallCommand_RendersWarningsToStderr
  - id: CLM-111
    requirement: REQ-011
    subject: cmd/backstop
    text: pack update renders every result warning to stderr, a path that renders nothing today
    tests:
      - TestPackUpdateCommand_RendersWarningsToStderr
  - id: CLM-112
    requirement: REQ-011
    subject: cmd/backstop
    text: pack upgrade renders every result warning to stderr, a path that renders nothing today
    tests:
      - TestPackUpgradeCommand_RendersWarningsToStderr
  - id: CLM-113
    requirement: REQ-011
    subject: cmd/backstop
    text: Rendering a warning leaves the command's exit code unchanged
    tests:
      - TestE2E_WarningDoesNotChangeExitCode
  - id: CLM-114
    requirement: REQ-011
    subject: cmd/backstop
    text: With streams read separately, warnings appear only on stderr and command output only on stdout
    tests:
      - TestE2E_WarningsAndOutputOccupySeparateStreams

contracts:
  - file: pkg/pack/distribution/identity.go
    provides:
      - name: RemoteIdentity
        kind: type
        signature: "type RemoteIdentity struct { Coordinate string; EffectiveVersion string; Tag string; ManifestName string; InstallName string; Diverged bool }"
        notes: "The validated identity/version tuple a remote lifecycle command must hold before it may mutate consumer state (REQ-002/REQ-003/REQ-006). Coordinate is the requested org/repository VERBATIM (REQ-004). EffectiveVersion is the normalized X.Y.Z with no prefix; Tag is that value re-prefixed with exactly one v, which is what Clone is called with. ManifestName is the pack.yml name and InstallName aliases it, so the field that builds the install path, the backstop.yml key, the lock key, and the engine asset root READS as an identity rather than as a name that happens to be reused. Diverged records ManifestName != Coordinate and drives the loud non-fatal diagnostic (REQ-006); it is a field rather than a recomputation so the comparison rule lives in exactly one place."
      - name: ResolveEffectiveVersion
        kind: function
        signature: "func ResolveEffectiveVersion(reference, overrideVersion string) (coordinate string, version string, err error)"
        notes: "REQ-001. Splits the reference on @, applies overrideVersion when non-empty (the flag wins, CLM-003), and normalizes at most one leading v via the package's existing parseVersionComponents so a tag and a manifest version mean the same thing everywhere. Returns the coordinate with the @version suffix removed and NOTHING else changed (CLM-043) plus the normalized version. Refuses with *VersionUnresolvedError when no version is available or the version is not strict X.Y.Z (CLM-005/006/007/008). It runs BEFORE any git subprocess, which is why it is a free function rather than a method: no dependency is needed to reject a malformed reference."
      - name: ValidateRemoteIdentity
        kind: function
        signature: "func ValidateRemoteIdentity(coordinate, effectiveVersion, packDir string) (*RemoteIdentity, error)"
        notes: "REQ-002/REQ-003/REQ-006, and the single gate every cloning command calls immediately after Clone and before any mutation (REQ-007). Reads packDir's pack.yml through ReadManifestIdentity, requires the manifest version to equal effectiveVersion after single-prefix normalization on each side (*VersionMismatchError otherwise, CLM-015), requires a name pack.ValidatePackName accepts (*IdentityError otherwise, CLM-033/034/035/036), and sets Diverged. It never mutates packDir and never touches the consumer project, so calling it is always safe."
      - name: ReadManifestIdentity
        kind: function
        signature: "func ReadManifestIdentity(packDir string) (name string, version string, err error)"
        notes: "Reads only the two identity fields from a pack directory's pack.yml, returning *IdentityError for an absent, unparseable, or nameless manifest (CLM-017/021/033). It is separate from the package's existing readPackManifest, which models only tool_config, and it replaces the inline yaml.Unmarshal into map[string]interface{} that resolveLocalPackSource uses today (command.go:772) so local and remote sources read identity through ONE implementation (CLM-038). Version STRICTNESS is decided here by parseVersionComponents, deliberately NOT by pack.validateSemver, whose pattern (manifest.go:460) accepts prerelease and build metadata that no strict release tag can equal (REQ-002/CLM-019)."
      - name: CoordinateForEntry
        kind: function
        signature: "func CoordinateForEntry(packName string, entry LockEntry) (coordinate string, warning string)"
        notes: "REQ-005. THE single place a lock entry becomes a repository coordinate. Returns entry.SourceCoordinate when present with an empty warning (CLM-051); falls back to packName and returns a warning naming the pack and the remedy when it is absent (CLM-052), which is every entry written before this spec. It exists as its own function rather than living inside RemoteURLForEntry because TagVersionResolver builds its own URL from a coordinate (versionresolver.go:70) while the clone paths need a URL — pack update needs BOTH in one invocation, which is why the warning must be de-duplicated by its caller and is claimed as emitted once (CLM-059)."
      - name: RemoteURLForEntry
        kind: function
        signature: "func RemoteURLForEntry(packName string, entry LockEntry) (url string, warning string)"
        notes: "REQ-005. Layered on CoordinateForEntry and passing its result to the package's existing resolveGitURL, so the URL a command clones and the coordinate a resolver lists tags at can never disagree (CLM-053). The warning is RETURNED rather than logged so the caller can route it to its own result's Warnings and the CLI can put it on stderr — distribution owns no output stream (REQ-011)."
      - name: VersionUnresolvedError
        kind: type
        signature: "type VersionUnresolvedError struct { Reference string; Problem string }"
        notes: "REQ-001/REQ-009. Pre-clone refusal: no version was supplied, or the supplied version is not a strict release version. Reference is what the operator typed, so the diagnostic quotes them back to themselves; Problem states which of the two it is. Classified under --json as kind \"version\" (CLM-094)."
      - name: VersionMismatchError
        kind: type
        signature: "type VersionMismatchError struct { Coordinate string; Tag string; ManifestVersion string; ExpectedVersion string }"
        notes: "REQ-002/REQ-009. Post-clone refusal: the cloned tag and the manifest disagree about what version this is. All four fields are named in the message because an operator debugging the published harness pack needs to see 0.1.3 and v0.1.1 side by side to know the fix is a tag, not a reinstall (CLM-016). Classified under --json as kind \"version\" (CLM-095)."
      - name: IdentityError
        kind: type
        signature: "type IdentityError struct { Coordinate string; Tag string; Field string; Problem string }"
        notes: "REQ-002/REQ-003/REQ-009. Every identity-gate refusal that is not a clean version disagreement: unreadable manifest, missing or non-strict manifest version, missing or invalid manifest name. Field names the pack.yml key at fault so the diagnostic points at a line the pack author can fix. Classified under --json as kind \"identity\" (CLM-096). It is a THIRD type rather than a Problem string on VersionMismatchError because the two carry different remedies — retag the repository versus fix the manifest — and a consumer switching on kind must be able to tell them apart."
    consumes:
      - source: pkg/pack
        name: ValidatePackName
        kind: function
      - source: pkg/pack/distribution
        name: parseVersionComponents
        kind: function
      - source: gopkg.in/yaml.v3
        name: Unmarshal
        kind: function
  - file: pkg/pack/manifest.go
    provides:
      - name: ValidatePackName
        kind: function
        signature: "func ValidatePackName(name string) error"
        notes: "REQ-003/CLM-037. The existing unexported validateName (manifest.go:462), EXPORTED so distribution can apply the same rule instead of copying it: exactly one slash, non-empty parts, each part matching namePartPattern. validateName becomes a call-through so pkg/pack's own callers are unchanged and there is one implementation. The export is cycle-safe — pkg/pack imports nothing from pkg/pack/distribution — and it is deliberately narrow: only the NAME rule is exported, not validateSemver, because identity's version strictness must be narrower than the manifest model's (REQ-002)."
  - file: pkg/pack/distribution/lockfile.go
    provides:
      - name: LockEntry
        kind: type
        signature: "type LockEntry struct { Name string; Version string; GitRef *string; ContentHash string; SourceType string; InstallDate string; LocalPath string; SourceCoordinate string }"
        notes: "REQ-004: SourceCoordinate carries `yaml:\"source_coordinate,omitempty\"` and records the requested org/repository verbatim for git sources, empty for local ones (CLM-040/044). omitempty is load-bearing, not cosmetic: without it every legacy entry gains a blank key on the first rewrite (CLM-046). It is provenance and resolution input — like LocalPath it is NOT part of ComputeContentHash, because where a pack came from is not pack content. A struct-field addition is invisible to the contracts gate, so CLM-040 through CLM-050 are the enforcement."
      - name: buildSortedLockEntryNode
        kind: function
        signature: "func buildSortedLockEntryNode(entry LockEntry) *yaml.Node"
        notes: "Emits source_coordinate between name and source_type, preserving the file's alphabetical key invariant (CLM-047). Emitted only when non-empty, matching the local_path precedent directly above it."
  - file: pkg/pack/distribution/command.go
    provides:
      - name: AddCommand.Run
        kind: method
        signature: "func (c *AddCommand) Run(packRef string, opts AddOptions) (*AddResult, error)"
        notes: "Reordered, not rewritten. The git branch becomes: ResolveEffectiveVersion (REQ-001) → Clone at the resolved tag → ValidateRemoteIdentity (REQ-002/003/006) → already-installed-and-current check keyed on the MANIFEST name → runValidationOnScratchCopy (REQ-008) → mutate. The already-current check moves AFTER identity resolution because it is keyed on the install name, which for a git pack is not knowable until the manifest is read; the local branch keeps its current position since a local pack's manifest is readable without cloning. Every install-path, backstop.yml-key, and lock-key use of the parsePackRef-derived name is replaced by identity.InstallName (REQ-003)."
      - name: InstallCommand.Run
        kind: method
        signature: "func (c *InstallCommand) Run(opts InstallOptions) (*InstallResult, error)"
        notes: "Its git branch resolves the clone URL through RemoteURLForEntry instead of resolveGitURL(name) (REQ-005/CLM-054), and appends the fallback warning to result.Warnings when an entry predates source_coordinate (CLM-052). Its local branch resolves no coordinate at all (CLM-060). It performs NO identity or manifest-version check (CLM-025): install is the hash-verified restore path and DD-12 keeps it that way."
      - name: UpdateCommand.Run
        kind: method
        signature: "func (c *UpdateCommand) Run(packName string, opts UpdateOptions) (*UpdateResult, error)"
        notes: "Five changes. It reads the pack's lock entry and resolves ONE coordinate through CoordinateForEntry, passing it to both ResolveLatestCompatible and the clone so the fallback warning is emitted once (REQ-005/CLM-055/056/059). It applies ValidateRemoteIdentity to the cloned tag before tamper detection or any mutation (REQ-002/CLM-023). It validates through runValidationOnScratchCopy rather than in place at command.go:539, which is the same contamination defect add had (REQ-008/CLM-083). It preserves the recorded coordinate when it rewrites the entry (REQ-004/CLM-048). And it carries divergence and fallback warnings out on UpdateResult.Warnings (REQ-011/CLM-068)."
      - name: UpgradeCommand.Run
        kind: method
        signature: "func (c *UpgradeCommand) Run(packRef string, opts UpgradeOptions) (*UpgradeResult, error)"
        notes: "The same five changes as update against the explicit major target, plus one behavior change: it must STOP DISCARDING readPackVersion's isLocal result (command.go:636 reads `currentVersion, _, err`) and refuse a local-source pack with a diagnostic naming pack relock (REQ-005/CLM-027/061). Today it clones unconditionally, which after REQ-005 would fire the coordinate fallback warning on a pack that has no repository at all. Scratch-copy validation replaces the in-place validation at command.go:654 (REQ-008/CLM-084), and the identity gate precedes the violation scan, which already precedes the tool-config merge that is upgrade's first consumer write."
      - name: recordGitPackInLock
        kind: function
        signature: "func recordGitPackInLock(projectDir, packName, version, contentHash, sourceCoordinate string) error"
        notes: "Gains the coordinate parameter so update and upgrade cannot rewrite an entry that silently loses it (REQ-004/CLM-048/049). A parameter rather than a read-modify-write inside the helper, so a caller that has no coordinate to preserve has to say so at the call site."
      - name: runValidationOnScratchCopy
        kind: function
        signature: "func runValidationOnScratchCopy(validator Validator, packDir, sourceLabel string) error"
        notes: "REQ-008, shared by add, update, and upgrade. Copies packDir into a temporary directory, runs RunPackCheck then RunPackTest against the COPY, and removes it on both paths (CLM-085/086). This is what keeps packval's sample_config rendering (phase3.go:143-168) out of the tree that is hashed and installed, and out of an operator's own working tree for a local-path add (CLM-081/082/087). sourceLabel is what a failure is REPORTED against — the coordinate and tag for a remote pack, the local path for a local one — because runPackvalPipeline quotes the directory it was handed (validator.go:66-71) and would otherwise show an operator a /var/folders scratch path (CLM-088/089). It takes the validator as a parameter rather than reading a receiver so all three commands share one implementation."
    consumes:
      - source: pkg/pack/distribution
        name: ValidateRemoteIdentity
        kind: function
      - source: pkg/pack/distribution
        name: CoordinateForEntry
        kind: function
      - source: pkg/pack/distribution
        name: RemoteURLForEntry
        kind: function
  - file: pkg/pack/distribution/add.go
    provides:
      - name: AddResult
        kind: type
        signature: "type AddResult struct { PackName string; Version string; ContentHash string; InstalledPath string; AlreadyCurrent bool; SourceCoordinate string; Warnings []string }"
        notes: "PackName is now the MANIFEST name (REQ-003), which is what the CLI's success line should have been printing all along. SourceCoordinate is what was requested (REQ-004). Warnings matches InstallResult's existing field and carries the divergence diagnostic (REQ-006/REQ-011/CLM-105); it is empty when coordinate and manifest name agree. The warning travels on the result rather than through an output stream because distribution owns no writer; the CHECK is pre-mutation, and the rendering happens where every other CLI message is rendered."
      - name: parsePackRef
        kind: function
        signature: "func parsePackRef(ref string) (string, string)"
        notes: "Unchanged in shape but demoted in meaning: its first return is now understood as the source COORDINATE, never as the pack name. ResolveEffectiveVersion is the only caller that should read a version from it."
  - file: pkg/pack/distribution/update.go
    provides:
      - name: UpdateResult
        kind: type
        signature: "type UpdateResult struct { OldVersion string; NewVersion string; ContentHash string; NoOp bool; Message string; Warnings []string }"
        notes: "REQ-011/CLM-106. Update has NO warning carrier today (update.go:28-34), so a divergence diagnostic or coordinate fallback computed inside UpdateCommand.Run would be dropped on the floor. The field is invisible to the contracts gate — which checks that the symbol UpdateResult exists, not what shape it has — so CLM-106 and CLM-111 are what actually hold it."
      - name: VersionResolver
        kind: interface
        signature: "type VersionResolver interface { ResolveLatestCompatible(coordinate, currentVersion string) (string, error); IsMajorBump(current, resolved string) bool }"
        notes: "REQ-005/CLM-058. The first parameter is renamed from packName to coordinate and its MEANING changes with it. The signature shape is unchanged, so no existing test double breaks; what changes is what callers pass and what the diagnostics quote."
  - file: pkg/pack/distribution/upgrade.go
    provides:
      - name: UpgradeResult
        kind: type
        signature: "type UpgradeResult struct { OldVersion string; NewVersion string; ContentHash string; RemediationBundle string; BaselinedViolations int; Warnings []string }"
        notes: "REQ-011/CLM-107. Same gap as UpdateResult (upgrade.go:22-29): no warning carrier exists today. Same reasoning about the contracts gate applies; CLM-107 and CLM-112 are the enforcement."
  - file: pkg/pack/distribution/versionresolver.go
    provides:
      - name: TagVersionResolver.ResolveLatestCompatible
        kind: method
        signature: "func (r *TagVersionResolver) ResolveLatestCompatible(coordinate, currentVersion string) (string, error)"
        notes: "REQ-005/CLM-058. The URL is built from the SOURCE coordinate at versionresolver.go:70, so a pack whose name differs from its repository resolves versions against the right remote. Its caller reads that coordinate from the lock through CoordinateForEntry and passes the same value to the clone, which is what makes CLM-059's single-emission claim meaningful."
  - file: cmd/backstop/pack_add.go
    provides:
      - name: newPackAddCommand
        kind: function
        signature: "func newPackAddCommand(jsonFlag *bool) *cobra.Command"
        notes: "REQ-011/CLM-109. Writes every result.Warnings entry to cmd.ErrOrStderr() before the success line goes to stdout, so a stream-separated assertion can prove the divergence notice is a diagnostic and not part of the command's output (CLM-070). The exit code is unchanged by a warning (CLM-113)."
  - file: cmd/backstop/pack_install.go
    provides:
      - name: newPackInstallCommand
        kind: function
        signature: "func newPackInstallCommand(jsonFlag *bool) *cobra.Command"
        notes: "REQ-011/CLM-110. The existing warning loop at pack_install.go:32 moves from cmd.Printf (stdout) to cmd.ErrOrStderr(), so reconciliation warnings and the new coordinate fallback share one stream. The existing TestPackInstallCommand_PrintsStaleLockWarning stays green because executeCommand (root_test.go:17-24) points SetOut and SetErr at the SAME buffer — which is also why the new stream claims must use the stream-separated binary runner instead."
  - file: cmd/backstop/pack_update.go
    provides:
      - name: newPackUpdateCommand
        kind: function
        signature: "func newPackUpdateCommand(jsonFlag *bool) *cobra.Command"
        notes: "REQ-011/CLM-111. Gains a warning-rendering loop it does not have today — the command currently renders only a no-op message or an updated-version line — writing every result.Warnings entry to cmd.ErrOrStderr() before either."
  - file: cmd/backstop/pack_upgrade.go
    provides:
      - name: newPackUpgradeCommand
        kind: function
        signature: "func newPackUpgradeCommand(jsonFlag *bool) *cobra.Command"
        notes: "REQ-011/CLM-112. Gains the same warning-rendering loop. The file's line-1 coverage waiver stays exactly where it is: it is anchored at the first line because coverage_threshold is a locationless dimension, and moving it silently dangles it."
  - file: cmd/backstop/json_error.go
    provides:
      - name: classifyJSONErrorKind
        kind: function
        signature: "func classifyJSONErrorKind(err error) string"
        notes: "REQ-009. Gains errors.As arms for *distribution.VersionUnresolvedError and *distribution.VersionMismatchError → \"version\", and *distribution.IdentityError → \"identity\" (CLM-094/095/096). The arms go BEFORE the existing unknown default and beside the existing git/validation/dependency/capability arms; no existing classification changes."
    consumes:
      - source: pkg/pack/distribution
        name: VersionUnresolvedError
        kind: type
      - source: pkg/pack/distribution
        name: VersionMismatchError
        kind: type
      - source: pkg/pack/distribution
        name: IdentityError
        kind: type
  - file: cmd/backstop/hermetic_remote_harness_test.go
    provides:
      - name: newHermeticRemoteKeepingManifestVersion
        kind: function
        signature: "func newHermeticRemoteKeepingManifestVersion(t *testing.T, packSourceDir string, tags ...string) *hermeticRemote"
        notes: "REQ-010/CLM-099. Creates the tagged repository WITHOUT the per-tag setPackVersion rewrite newHermeticRemote applies, which is the only way a tag-versus-manifest version divergence can exist in a hermetic fixture. Both constructors share one body with the rewrite behind a flag, so the redirect, identity, and assertion machinery is not duplicated. Existing callers of newHermeticRemote are untouched."
    consumes:
      - source: cmd/backstop
        name: newHermeticRemote
        kind: function
---

# SPEC-056: Remote Identity Version Validation

## Overview

BUNDLE-006 split the remote-distribution repair into seeds. SPEC-055 delivered the
foundation — real cloner, real validator, real resolver, constructors that make an
incompletely wired command a compile error — and explicitly deferred
source-coordinate-versus-manifest-identity. This spec is that deferred seed: bundle
REQ-039@1.1.0, resolved by OQ-9 into DD-31.

The defect is one line of misplaced trust. `AddCommand.Run` derives a git pack's
name from `parsePackRef(packRef)` — the string the operator typed — and then uses
that string as the install path, the `backstop.yml` key, and the lock key. But the
gate resolves a pack's rules, producers, converters, and validators under the name
the pack DECLARES in its manifest. When those differ, `pack add` reports success and
`backstop gate` later fails looking for assets in a directory that was never
created. That is not hypothetical: the harness consumer published a pack named
`backstop/harness-toolchain` at `backstop-ai/backstop-harness-toolchain-pack`, and
the gate failed with `missing convert script`.

OQ-9 asked which of the two names is authoritative and resolved to neither-and-both:
the repository coordinate is a SOURCE LOCATION, the manifest name is an IDENTITY, and
the lock records both so a consumer can answer "where did this come from?" and "what
is it called here?" independently. That is the shape of a Homebrew tap, which is the
distribution model the bundle names as load-bearing: a formula is not identified by
the repository that hosts it.

Nothing checks version coherence either. A pack can declare any semantic version in
its manifest and be tagged with any other, and today both `pack add` and the gate
will believe whichever one they happen to read. A live counterexample is already
published — the harness toolchain pack's manifest says `0.1.3` while its tags stop
at `v0.1.1` (DIR-027 item 2) — so this is a check with a waiting first customer, not
a speculative one.

Three things travel with the identity work because they cannot be separated from it.
The first is consumer-side resolution: once the lock is keyed by manifest name, the
repository URL can no longer be derived from that key, so install, update, upgrade,
and version resolution must read the recorded coordinate instead. Without that, this
spec would make divergent-name packs uninstallable. The second is warning carriage:
two of the four commands have no way to return a warning at all, so a diagnostic
computed inside them would be computed and then dropped. The third is a validate/hash
ordering hazard described under Sharp Edges and pinned by REQ-008: `pkg/packval`
writes into the directory it validates, and add, update, and upgrade each validate
the tree they are about to hash, while `pack install` never validates at all — so
for any pack declaring a `tier: complete` scaffold with `sample_config`, all three
record a hash a fresh clone cannot reproduce. Leaving that unpinned would mean
shipping identity validation on top of a hash the identity work is supposed to make
trustworthy.

## Requirements

Requirements are defined in the frontmatter. This section explains the decisions
behind them and states the matrices they define, in the same terms.

### The identity gate and where it sits

Every command that clones a tag it intends to install runs the same three-step gate
between the clone and the first consumer write: resolve exactly one effective
version (REQ-001), require the cloned manifest to agree with that version
(REQ-002), and take the install/runtime identity from the manifest name
(REQ-003). REQ-007 fixes the gate's position — before validation, before any
mutation — and enumerates the consumer-state surfaces that must be untouched when
it refuses, per command.

`pack install` is the deliberate exception. It restores what the lock already
records, verifying content hashes and re-validating nothing (DD-12). Applying the
manifest-version equality check there would make install fail on a repository whose
manifest drifted after a successful add, even though the bytes it restored are
exactly the bytes that were locked.

### Which command performs which check

| Command | Source | Effective version (REQ-001) | Manifest version equality (REQ-002) | Identity from manifest name (REQ-003) | Coordinate written (REQ-004) | Coordinate read (REQ-005) | Scratch validation (REQ-008) |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `pack add` | git | yes | yes | yes | yes | n/a — the operator supplies it | yes |
| `pack add` | local | no — a local source has no tag | no | yes | no — `local_path` records the source | n/a | yes |
| `pack install` | git | no — the lock pins the version | no | n/a — the lock key is already the identity | no | yes | n/a — install never validates |
| `pack install` | local | no | no | n/a | no | no — it materializes from `local_path` | n/a |
| `pack update` | git | no — resolution supplies it | yes | yes | preserved on rewrite | yes | yes |
| `pack update` | local | no — update is a no-op for local | no | no | no | no | no |
| `pack upgrade` | git | yes — the explicit major target | yes | yes | preserved on rewrite | yes | yes |
| `pack upgrade` | local | no — refused by the new local guard | no | no | no | no — the guard refuses first | no |
| `pack remove` | git or local | no | no | n/a — it looks the pack up by the key REQ-003 sets | no | no | n/a |
| `pack relock` | local | no | no | no | preserved, not written | no | n/a |

Every row is claimed, including the ones that state what does NOT change. Three of
them deserve a note. `pack upgrade` on a local source is the one row this spec
changes behaviorally beyond identity: today `UpgradeCommand.Run` discards
`readPackVersion`'s `isLocal` result and clones unconditionally, and after REQ-005
that path would resolve a coordinate — and fire the fallback warning — for a pack
that has no repository, so REQ-005 requires the guard. `pack remove` looks a pack up
by its `backstop.yml` and lock key, which REQ-003 makes the manifest name, so remove
keeps working on a divergent-name pack without any change of its own. `pack relock`
refreshes a local entry's hash, touches no remote, and its argument shape is
ISSUE-074's, which belongs to the migration seed.

### Version equality, exactly

REQ-001 and REQ-002 both normalize at most ONE leading `v`, using the same
`parseVersionComponents` helper `TagVersionResolver` already applies to tags, so a
tag, a manifest version, and a resolved version all mean the same thing. This is
deliberately stricter than `pkg/pack`'s own `semverPattern`, which accepts
prerelease and build-metadata suffixes — a manifest may legitimately declare
`1.0.0-rc1` and still be a valid manifest, but no strict release tag can equal it,
so it cannot be installed by tag. The consequences, stated so the claim matrix and
the code agree:

| Effective version | Manifest `version` | Outcome |
| --- | --- | --- |
| `v1.0.0` | `1.0.0` | passes |
| `1.0.0` | `1.0.0` | passes |
| `v1.0.0` | `v1.0.0` | passes — one prefix normalizes on each side |
| `v1.0.0` | `1.0.1` | `*VersionMismatchError` |
| `v0.1.1` | `0.1.3` | `*VersionMismatchError` — the published harness pack |
| any | absent | `*IdentityError` |
| any | `1.0` | `*IdentityError` |
| any | `1.0.0-rc1` | `*IdentityError` — accepted by `pack.validateSemver`, rejected here |
| any | `vv1.0.0` | `*IdentityError` |
| any | no readable `pack.yml` at all | `*IdentityError` |
| none supplied | any | `*VersionUnresolvedError`, before git runs |
| `1.0` | any | `*VersionUnresolvedError`, before git runs |
| `1.0.0-rc1` | any | `*VersionUnresolvedError`, before git runs |
| `vv1.0.0` | any | `*VersionUnresolvedError`, before git runs |

### Name validity, exactly

REQ-003 takes name validity from `pkg/pack`'s existing rule rather than restating
it, which requires exporting `validateName` as `pack.ValidatePackName`. The rule has
three failure modes and each is claimed, alongside the empty-name case the identity
reader catches first:

| Manifest `name` | Outcome |
| --- | --- |
| `hermetic/valid-pack` | accepted |
| absent or empty | `*IdentityError` |
| `validpack` — no slash | `*IdentityError` |
| `hermetic/` or `/valid-pack` — empty part | `*IdentityError` |
| `hermetic/valid pack` — character outside `[A-Za-z0-9-]` | `*IdentityError` |

Only the NAME rule is exported. Exporting `validateSemver` alongside it would invite
exactly the reuse REQ-002 forbids.

### Divergence warns; it does not refuse

REQ-006 takes the bundle-grounded position. OQ-9 considered requiring the
coordinate to equal the manifest name — option (a) — and rejected it, because the
harness pack's naming was reasonable and the actual defect was that the install path
was built from the wrong one of the two. The ten packs published under
`backstop-ai` on 2026-07-26 all hold `name == coordinate`, and DIR-027 records that
as a ratified fleet CONVENTION. A convention is exactly the thing a tool should make
visible rather than enforce: divergence produces a diagnostic naming all three of
coordinate, manifest name, and install path, and the command proceeds. Equality
produces silence, so a warning appearing in a fleet install is itself the signal.

### Why REQ-005 and REQ-011 are in this spec and not later ones

REQ-003 changes the lock key from the coordinate to the manifest name. Every remote
operation on a locked pack currently builds its URL from that key via
`resolveGitURL`. Landing REQ-003 without REQ-005 would leave any pack whose name
differs from its repository installable exactly once and never restorable — the
lock would name a repository that does not exist. REQ-005 adds no new capability; it
redirects four existing call sites at the field REQ-004 introduces, through one
accessor so the two consumers of a coordinate cannot drift apart.

REQ-011 is the same kind of necessity one layer up. `UpdateResult` and
`UpgradeResult` have no warning field, and `pack update` and `pack upgrade` render
no warnings at all. Both of the diagnostics this spec introduces — divergence
(REQ-006) and coordinate fallback (REQ-005) — arise inside those commands. Without
carriers and renderers they would be computed correctly and then discarded, which
is a worse outcome than not computing them, because the code would look right.

### Why the validate/hash pin is here

REQ-008 is the one requirement not derived from REQ-039's text. It is here because
this spec is where the ordering constraints around identity resolution, validation,
copying, and hashing are being written down, and the existing ordering is wrong in
all three validating commands in a way that silently defeats the hash. Fixing it
costs one scratch copy per command; inheriting it would mean specifying identity
checks whose downstream artifact — a reproducible content hash — cannot be trusted
for a whole class of packs.

REQ-008 is pinned to REQ-039@1.1.0 rather than REQ-021@1.1.0, and that is a
deliberate traceability choice rather than an oversight. In spirit it serves
REQ-021@1.1.0's authored-content boundary. But REQ-021@1.1.0 also requires `.git`
exclusion for every source — SPEC-055's `Clone` strip covers the remote source only,
and local-path packs still hash whatever is on disk — and travels with the
legacy-lock migration REQ-041 owns. Declaring support for REQ-021@1.1.0 here would
mark that requirement covered at its current version in the bundle traceability
gate while two thirds of it remain unbuilt, which is exactly the vacuous green the
gate exists to prevent. The migration seed keeps REQ-021.

## Implementation

Nine passes. Each is independently compilable and testable; the ordering is chosen
so no pass leaves the tree in a state where a divergent pack is half-supported, and
so no pass computes a warning before there is somewhere to put it.

**Pass 1 — the name rule, exported.** `pkg/pack/manifest.go` gains
`ValidatePackName`, with the existing `validateName` reduced to a call-through so
there is one implementation. Nothing outside `pkg/pack` changes yet.

**Pass 2 — the identity module.** New `pkg/pack/distribution/identity.go` carrying
`RemoteIdentity`, `ResolveEffectiveVersion`, `ValidateRemoteIdentity`,
`ReadManifestIdentity`, `CoordinateForEntry`, `RemoteURLForEntry`, and the three
typed errors. Nothing calls it yet. Normalization reuses `parseVersionComponents`
from `versionresolver.go`; name validity calls `pack.ValidatePackName`.

**Pass 3 — the lock field.** `LockEntry.SourceCoordinate` with
`yaml:"source_coordinate,omitempty"`, emitted by `buildSortedLockEntryNode` between
`name` and `source_type`. Round-trip tests for both a coordinate-carrying entry and
a legacy entry that lacks the field.

**Pass 4 — warning carriers and renderers.** `Warnings []string` on `AddResult`,
`UpdateResult`, and `UpgradeResult`; the stderr rendering loop added to `pack add`,
`pack update`, and `pack upgrade`, and moved from stdout onto stderr in
`pack install`. This lands before anything produces a warning so no later pass can
compute one with nowhere to put it.

**Pass 5 — the scratch-copy validation seam.** `runValidationOnScratchCopy` in
`command.go`, and the validation calls in `AddCommand.Run` (`command.go:154`),
`UpdateCommand.Run` (`command.go:539`), and `UpgradeCommand.Run` (`command.go:654`)
all rewired through it, each passing the source label its failures should quote.
This lands before the identity rewiring so the hash the later passes assert on is
already the correct one. A characterization test in `pkg/packval` pins the mutation
this pass contains, so the seam does not become vacuous if phase 3's behavior ever
changes.

**Pass 6 — `AddCommand.Run` reordering.** The git branch becomes: resolve effective
version → clone → validate identity → already-installed-and-current check keyed on
the install name → scratch-copy validation → mutate. The
already-installed-and-current check MOVES: it is keyed on the install name, which
for a git source is unknowable before the manifest is read. The local branch keeps
its current shape but routes its manifest read through `ReadManifestIdentity`,
replacing the inline `yaml.Unmarshal` into `map[string]interface{}` at
`command.go:772`. Every remaining use of the `parsePackRef`-derived name for an
install path, a `backstop.yml` key, or a lock key becomes `identity.InstallName`.

**Pass 7 — the coordinate on the write side.** `AddCommand.Run` records
`SourceCoordinate` for git sources and leaves it empty for local ones;
`recordGitPackInLock` gains the coordinate parameter; update and upgrade pass the
coordinate they read from the existing entry so a rewrite cannot drop it.

**Pass 8 — the coordinate on the read side, and the upgrade local guard.**
`RemoteURLForEntry` replaces `resolveGitURL(name)` in `InstallCommand.Run`,
`UpdateCommand.Run`, and `UpgradeCommand.Run`;
`TagVersionResolver.ResolveLatestCompatible`'s first parameter is renamed to
`coordinate` and its caller passes the coordinate from the lock, resolved once per
invocation. `UpgradeCommand.Run` stops discarding `isLocal` and refuses a
local-source pack before any coordinate resolution. Fallback warnings are appended
to each command's result warnings.

**Pass 9 — identity checks on update and upgrade, CLI surfacing, and fixtures.**
Both cloning commands call `ValidateRemoteIdentity` immediately after their clone;
`json_error.go` gains the `version` and `identity` kinds;
`hermetic_remote_harness_test.go` gains the manifest-version-preserving
constructor; and `cmd/backstop/testdata/hermetic-remote/` gains the three fixtures
REQ-010 names. The end-to-end claims run last because they need all of the above.

### Validation passes performed by the identity gate

For the planner's task mapping, the gate is exactly five checks in this order, and a
refusal at any one of them means none of the later ones ran:

1. A version is available from the reference or the flag.
2. That version is a strict release version after one optional `v`.
3. The cloned tree has a readable `pack.yml` declaring both a name and a version.
4. The manifest version equals the effective version after one optional `v` on each
   side.
5. The manifest name is accepted by `pack.ValidatePackName`.

Divergence between the manifest name and the coordinate is computed after check 5
and is a diagnostic, never a sixth refusal.

### Existing tests this spec invalidates

Two, both named here with their disposition so the implementer does not discover
them as surprises.

`TestPackAdd_MergeToolConfigErrorRollsBack` (`add_test.go:1229`) builds a git-source
pack whose `pack.yml` is binary garbage and asserts the failure arrives from
`MergeToolConfig` → `readPackManifest`. After Pass 6 that pack is rejected far
earlier, by `ReadManifestIdentity`, so the test's premise no longer holds. It is
NOT a mandated name — no spec references it — so the disposition is a rewrite in
place: keep the name and the rollback assertion, and reach the merge failure the way
the test intends, with a manifest that parses and identity-validates but carries a
`tool_config` entry the merge cannot apply.

`TestPackAdd_LocalPathValidatesInPlace` (`add_test.go:710`) IS mandated — SPEC-015
CLM-071 lists it — and the behavior its name describes is precisely what REQ-008
retires. The disposition is: the NAME SURVIVES VERBATIM, because a mandated test
name may not disappear, and the ASSERTION UPDATES. It is re-listed here under
CLM-087 as the local-source half of that claim: a local-path add still validates the
pack it was pointed at and still succeeds, and the operator's source directory is
unchanged afterward. The name remains accurate in the sense SPEC-015 cared about —
local paths are validated rather than skipped — and stops asserting the in-place
mechanism that was never the point.

## Verification

`go test ./pkg/pack/distribution/... ./cmd/backstop/... ./pkg/packval/...
./pkg/pack/... -race -coverprofile=cover.out`, at integration level. All four
packages are in scope: the identity gate lives in distribution, its CLI surfacing
and every hermetic end-to-end claim live in `cmd/backstop`, REQ-008's
characterization claim pins behavior in `pkg/packval`, and REQ-003's exported name
rule lives in `pkg/pack`.

The coverage threshold is 80, matching SPEC-055's threshold for the same surface.
`cmd/backstop` is dominated by cobra wiring that the package's own suites exercise
indirectly, and this spec adds proportionally more test code than production code to
it. The threshold is a floor on the combined profile, not a per-package target;
`pkg/pack/distribution/identity.go` is new production code with a claim on every
branch and should not benefit from that floor.

Remote claims run against hermetic tagged repositories through the built binary via
the SPEC-055 harness, never against the network. Stream assertions MUST use the
stream-separated runner and not `executeCommand`, which points cobra's out and err
at the same buffer (`root_test.go:17-24`) and would let every stream claim pass
vacuously.

## Sharp Edges

**The hermetic harness cannot currently express the failure this spec exists to
catch.** `newHermeticRemote` calls `setPackVersion` for every tag it creates,
rewriting `pack.yml`'s version to match the tag. A version-drift fixture built with
it would be silently repaired into a passing fixture, and REQ-002's central claim
would test nothing. REQ-010 and CLM-099 exist for exactly this reason. Any
implementation that adds the drift fixture without the harness variant produces a
green suite that proves nothing.

**Validation writes into what it validates, and three commands hash what validation
wrote.** `pkg/packval/phase3.go` renders each `tier: complete` scaffold's
`sample_config` into `<packDir>/<scaffold.path>/`. `AddCommand.Run`,
`UpdateCommand.Run`, and `UpgradeCommand.Run` each validate a tree in place and then
copy and hash that same tree; `InstallCommand.Run` clones fresh and never validates.
For any pack with such a scaffold, the recorded hash is unreproducible and install is
guaranteed to fail with a mismatch that looks like tampering. No pack in the
repository declares a `tier: complete` scaffold today — every existing declaration is
`tier: skeleton` — which is why this has not fired yet. A latent defect with no live
symptom is the kind most likely to be "simplified" back in, and fixing it in add
alone would leave two thirds of it standing.

**The scaffold fixture cannot be process-free, only network-free and toolchain-free.**
`DefaultExecutor.RunScaffoldTest` runs `exec.Command("sh", "-c", testCommand)`
unconditionally for a `tier: complete` scaffold, and `PackvalValidator` supplies no
executor, so validating that fixture WILL spawn a shell. The hermetic property to
hold it to is therefore "reaches no network and no toolchain or provisioned binary",
with the `test_command` restricted to a shell builtin. Writing the claim as "invokes
no external tool" would be false by construction and would surface as a failing test
rather than as a design decision.

**For a local-path add, the directory validation mutates is the operator's own
working tree.** The same phase-3 rendering writes into the source directory the
operator pointed at, not into a temporary clone. The scratch-copy fix covers this,
but only if it is applied to the local branch too and not just to the remote one.

**A scratch copy moves the path a failure quotes.** `runPackvalPipeline` renders
`pack validation (%s) of %s failed in %s` with the directory it was handed
(`validator.go:66-71`). Once that directory is a scratch temp dir, an unmodified
implementation shows operators a `/var/folders/...` path that will not exist by the
time they look at it. REQ-008 requires the failure to be reported against the
original source instead — which is why `runValidationOnScratchCopy` takes a source
label rather than only a directory.

**Moving the already-installed-and-current check changes when a no-op is detected.**
Today `pack add` short-circuits before cloning. After Pass 6 a git add always clones
before it can know its install name, so re-adding a current pack costs one clone. The
alternative — keying the short-circuit on the coordinate — would reintroduce exactly
the coordinate-as-identity assumption this spec removes. The cost is one shallow
single-ref clone on an operation that is already a no-op, and it is deliberate.

**A coordinate resolved twice can warn twice.** `pack update` needs a coordinate for
`ls-remote` and a URL for the clone. If each call site resolves independently, a
pre-coordinate lock entry produces two identical fallback warnings for one command
invocation — noise that trains operators to ignore the signal. `CoordinateForEntry`
exists so the decision is made once per invocation, and CLM-059 is what holds it.

**Every existing lock entry lacks the coordinate, including this repository's own.**
`backstop-core`'s six entries are `source_type: local` and unaffected, but any
consumer holding git entries written before this spec — including the ten-pack
consumer DIR-027 describes — hits REQ-005's fallback path on its next install. The
fallback works for those packs precisely because they hold `name == coordinate`, and
the warning tells the operator to re-add or relock. Making the fallback silent would
turn a fleet-wide "re-add these" signal into nothing.

**Case-only divergence is real divergence.** REQ-006 compares byte-exactly, so
`Backstop-AI/go-standards` against `backstop-ai/go-standards` warns. That is
intentional and follows directly from DD-31's refusal to apply host-specific case
normalization: GitHub would treat them as the same repository, and a different host
would not. An implementation that "helpfully" case-folds the comparison silently
reintroduces the host assumption the requirement exists to remove.

**Moving install's warnings to stderr is invisible to the test that covers them.**
`TestPackInstallCommand_PrintsStaleLockWarning` reads `executeCommand`'s single
merged buffer, so the move from stdout to stderr cannot turn it red. That is
convenient and also a trap: it means the existing suite CANNOT tell the two streams
apart, and every claim in this spec about which stream a message lands on has to be
driven through the built binary with separated streams or it proves nothing.

**The existing hermetic fixtures all hold `name == coordinate` by accident.**
`hermetic/valid-pack`, `hermetic/invalid-pack`, and `hermetic/fixture-fail-pack`
each declare a manifest name equal to `hermetic/<directory>`, which is exactly what
`remoteE2ESetup` builds the coordinate from. SPEC-055's suite therefore stays green
through REQ-003 — but by coincidence, not by design. Renaming a fixture directory
without renaming its manifest would turn every SPEC-055 remote claim into a
divergence case.

**The identity gate changes which diagnostic a bad pack produces.** Because the gate
precedes validation (REQ-007), a pack that is both identity-invalid and
validation-failing now reports the identity error. SPEC-055's
`TestE2E_PackAdd_ValidationFailureIsLoud` uses `hermetic/invalid-pack`, whose
manifest declares a valid name and a version the harness rewrites to the tag, so it
still reaches validation — again, by coincidence. CLM-076 pins the ordering
deliberately so the coincidence is not the only thing holding it.

**Nothing here delivers rollback.** REQ-007 guarantees that nothing is written
before the gate; it guarantees nothing about a failure after it. `AddCommand.Run`'s
existing best-effort `rollback` closure stays exactly as it is, with all its existing
gaps, until REQ-040's shared transaction coordinator lands. A reader should not
mistake "pre-mutation ordering" for "transactional".

## Review Questions

1. Does every use of a pack name to build a filesystem path, a `backstop.yml` key,
   a lock key, or an engine asset root read the MANIFEST name, with no surviving
   path from `parsePackRef`'s first return to any of those four?
2. Does `ValidateRemoteIdentity` run before the first write to each consumer-state
   surface REQ-007 enumerates, in `add`, `update`, and `upgrade` alike — and is that
   ordering asserted, not merely arranged?
3. Is scratch-copy validation applied in all THREE validating commands, or did it
   land in `add` only while `update` (`command.go:539`) and `upgrade`
   (`command.go:654`) kept validating the tree they then hash?
4. Is validation run against a copy in the LOCAL branch as well as the remote one,
   or does the local path still hand the operator's own directory to packval?
5. Does a validation failure quote the original coordinate or local path, or does it
   quote a `/var/folders` scratch directory the operator cannot inspect?
6. Is the content hash computed over a tree that no validator has written to, and
   is that proven by a fixture that actually declares a `tier: complete` scaffold
   with `sample_config` rather than by a test double alone?
7. Does the version-drift fixture use the manifest-version-preserving harness
   constructor? If it uses `newHermeticRemote`, the drift was rewritten away and the
   claim is vacuous.
8. Is there exactly one place a lock entry becomes a coordinate, and does `pack
   update` — which needs it twice — emit the fallback warning once rather than twice?
9. Do `UpdateResult` and `UpgradeResult` actually carry a `Warnings` field, and do
   `pack update` and `pack upgrade` actually render it? The contracts gate checks
   that the symbols exist, not their shape, so only the claims catch a warning that
   is computed and dropped.
10. Do all four commands render warnings to stderr, and are the stream claims driven
    through the built binary rather than through `executeCommand`'s merged buffer?
11. Does `UpgradeCommand.Run` still discard `readPackVersion`'s `isLocal`? If so, a
    local-source pack reaches coordinate resolution and warns about a repository it
    does not have.
12. Does any comparison in the identity path apply `strings.ToLower`, `EqualFold`, or
    any other normalization to a coordinate or a manifest name beyond the single
    optional `v` on a version?
13. Does `pack install` still succeed on a hash-matching entry whose remote manifest
    version disagrees with the lock, or did the identity gate leak into the restore
    path?
14. Does distribution call `pack.ValidatePackName`, or did a second copy of the name
    rule appear in `identity.go`? And did `validateSemver` get exported alongside it,
    which would reintroduce the prerelease acceptance REQ-002 forbids?
15. Does `TestPackAdd_LocalPathValidatesInPlace` still exist by that exact name, with
    an updated assertion rather than a deletion?

## References

- BUNDLE-006 — Pack Distribution Lifecycle, REQ-039@1.1.0, REQ-020@1.1.0, DD-26,
  DD-31, OQ-9 resolution
- SPEC-055 — Production Remote Dependency Assembly (the foundation seed; declares
  this spec's surface out of its own scope)
- SPEC-015 — Pack Distribution (CLM-071 mandates
  `TestPackAdd_LocalPathValidatesInPlace`, whose assertion this spec updates)
- DIR-026 — Remote Pack Consumption (Notes: REQ-039 named the highest-value
  remaining seed)
- DIR-027 — Pack Fleet Publication and Migration (item 2: the published harness
  pack's manifest/tag drift; the `name == coordinate` convention ratified
  2026-07-26)
- ISSUE-083 — `resolveGitURL` hardcodes the GitHub host; post-launch, explicitly not
  addressed here
- `pkg/pack/distribution/command.go`, `add.go`, `update.go`, `upgrade.go`,
  `gitcloner.go`, `versionresolver.go`, `lockfile.go`, `validator.go` — the as-built
  surface this spec modifies
- `pkg/pack/manifest.go` — `validateName` and `semverPattern`, the rule this spec
  exports and the one it deliberately does not reuse
- `pkg/packval/phase3.go`, `pkg/packval/executor.go` — the `sample_config` rendering
  REQ-008 contains and the shell subprocess REQ-010 accounts for
- `cmd/backstop/hermetic_remote_harness_test.go` — the harness REQ-010 extends

## Version History

- **1.1.0** (2026-07-26): Review revision, six blockers and seven should-fixes.
  Extended REQ-008's scratch-copy pin to `pack update` and `pack upgrade`, which
  carry the identical contamination defect, and pinned the failure diagnostic to the
  original source rather than the scratch path. Added REQ-011 (warning carriers and
  stderr rendering) after finding `UpdateResult` and `UpgradeResult` have no warning
  field and their CLI commands render none. Split coordinate resolution into
  `CoordinateForEntry` with `RemoteURLForEntry` layered on it, with a
  single-emission claim for `pack update`. Named `pack.ValidatePackName` as the
  export that makes REQ-003's identifier rules reachable, and recorded that
  identity's version strictness deliberately does not reuse `pack.validateSemver`.
  Corrected the `pack upgrade` local-source row: the command discards `isLocal`
  today, so REQ-005 now requires the guard. Restated REQ-010's hermetic property as
  network-free and toolchain-free rather than process-free. Added the two
  invalidated tests with dispositions, the REQ-008 traceability-pin rationale, and
  REQ-007 refusal claims for update and upgrade.
- **1.0.0** (2026-07-26): Initial spec. BUNDLE-006's identity seed: effective
  version resolution, manifest-version equality, manifest-name-as-identity,
  verbatim source coordinate recorded and read back, loud non-fatal divergence,
  pre-mutation ordering, and the validate/hash ordering pin.
