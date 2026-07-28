---
title: "Pack Distribution Content Identity"
number: SPEC-057
created: "2026-07-27"
updated: "2026-07-27"
status: implemented
schema_version: spec/v1
spec_version: 1.0.1

implementation:
  summary: >
    BUNDLE-006's AUTHORED CONTENT IDENTITY delta spec. It pins exactly one
    bundle requirement at its current edition — `REQ-021@1.1.0`: the content
    hash covers every AUTHORED file, and root repository-control metadata is
    neither copied into an installed pack nor folded into its hash (DD-20,
    DD-24) — and it is deliberately narrow. The bundle's other 1.1.0 content
    identity pin, `REQ-020@1.1.0` (manifest-name lock key plus verbatim source
    coordinate), is ALREADY owned by SPEC-056, which is `implemented`; this spec
    does not re-pin it, because `replaced-by` accepts a LIST
    (`pkg/validate/terminal.go:37-38`), so SPEC-015 retires as
    `replaced-by: [SPEC-056, SPEC-057]` with each successor owning the pin it
    actually delivers. The legacy-hash MIGRATION mechanism (`REQ-041@1.1.0`,
    DD-28, OQ-6's typed migration diagnostic and explicit remote re-validating
    migration operation), the shared staged-filesystem transaction coordinator
    (`REQ-040@1.1.0`), `pack relock`'s argument shape (ISSUE-074 residual), and
    the hermetic lifecycle parity suite (`REQ-042@1.1.0`) all stay in their own
    later seeds. REQ-021@1.1.0 is HALF-delivered today. SPEC-055's
    `ExecGitCloner.Clone` strips the root `.git` from every clone before any
    caller sees it (CLM-101/CLM-102), so the remote path is already
    authored-content-only and its add→fresh-clone-install hash EQUALITY is
    proven. The local-path half is not built — ISSUE-088.
    `ComputeContentHash` (`hash.go:17`) walks every file it is given with no
    `.git` awareness of any kind, and `copyDirRecursive` (`add.go:222`) copies
    every file it is given the same way, so a local-path `pack add <dir>` of a
    pack checked out with its own repository carries reflog timestamps and
    object layout into both the installed tree and the recorded hash. The
    divergence is MEASURED, not theorized: the same pack hashed `639f74fb…`
    without a root `.git` and `bb86715c…` with one (bundle ~771-773), and this
    repository's own lock carries three such contaminated entries today (Sharp
    Edges). What lands is one unexported predicate — root-relative name equality
    against the package's existing `gitDirectoryName` constant — consumed by
    BOTH boundaries: `ComputeContentHash` skips the entry (and its subtree) so
    no call site can hash metadata, and `copyDirRecursive` skips it so no
    installed pack can contain it. Both, not either: the copy boundary alone
    leaves `materializeLocalPack` (`install.go:78`) hashing the operator's
    SOURCE directory as it sits on disk, and the hash boundary alone leaves a
    `.git` tree copied into `.backstop/packs/`. The predicate keys on the walk's
    own root and on EXACT name equality, so `.gitignore`, `.gitattributes`,
    `.github/`, and a nested non-root `.git` all remain content — matching,
    exactly, the one-path strip the cloner already performs, which is what makes
    local and remote identity converge rather than merely both change.
  subject: pkg/pack/distribution

verification:
  level: integration
  test_command: go test ./pkg/pack/distribution/... ./cmd/backstop/... -race -coverprofile=cover.out
  coverage_threshold: 80

requirements:
  - id: REQ-001
    text: >
      `ComputeContentHash` must exclude root repository-control metadata from
      the manifest it hashes, regardless of the shape that metadata takes: a
      `.git` DIRECTORY (and its entire subtree), a `.git` FILE (the worktree or
      submodule pointer), or a `.git` SYMLINK, which must be skipped WITHOUT
      being followed — including a dangling one, which must not fail the walk.
      The exclusion is keyed on EXACT name equality at the walked directory's
      own root: `.gitignore`, `.gitattributes`, `.github/`, `.git-hooks/`, and
      any `.git`-named entry BELOW the root remain authored content and continue
      to contribute. A directory containing only root repository metadata must
      therefore hash as an empty directory, and a directory containing no such
      metadata must hash to exactly the value the documented sorted
      `relative-path:file-hash` manifest algorithm produces — unchanged from
      today. Errors from the walk must be handled explicitly and wrapped with
      the path at fault.
    supports:
      - pack-distribution-lifecycle:REQ-021@1.1.0
    follows:
      - STD-GO-001:GO-010
      - STD-GO-001:GO-011
  - id: REQ-002
    text: >
      The pack copy step must apply the SAME root-metadata exclusion, so no
      installed pack under `.backstop/packs/` ever contains root
      repository-control metadata no matter which source it came from, and a
      nested non-root `.git` is still copied because it is authored content. As
      a consequence, a local-path `pack add` of a directory carrying its own
      root `.git` must record the content hash of the metadata-free authored
      tree — the same value the identical content hashes to with no repository
      present — so local and remote identity CONVERGE on one algorithm:
      subsequent repository churn in the operator's source directory must not
      break `pack install`'s hash verification of that local pack, and `pack
      relock` of the installed tree must reproduce the hash `pack add`
      recorded. The remote half of this convergence already holds via
      SPEC-055's post-clone strip and is claimed here as the baseline the local
      half must meet.
    supports:
      - pack-distribution-lifecycle:REQ-021@1.1.0
    follows:
      - STD-GO-001:GO-011

claims:
  # REQ-001 — the hash boundary, across every shape root metadata takes and
  # every adjacent name that must NOT be mistaken for it (ISSUE-088).
  - id: CLM-001
    requirement: REQ-001
    text: A root .git DIRECTORY and its entire subtree contribute nothing to the content hash
    tests:
      - TestComputeContentHash_RootGitDirectoryExcluded
  - id: CLM-002
    requirement: REQ-001
    text: A root .git FILE — the worktree or submodule pointer — contributes nothing to the content hash
    tests:
      - TestComputeContentHash_RootGitFileExcluded
  - id: CLM-003
    requirement: REQ-001
    text: A root .git SYMLINK is skipped without being followed, so its target's content never reaches the manifest
    tests:
      - TestComputeContentHash_RootGitSymlinkExcludedAndNotFollowed
  - id: CLM-004
    requirement: REQ-001
    text: A DANGLING root .git symlink is skipped rather than failing the walk, which it does today
    tests:
      - TestComputeContentHash_DanglingRootGitSymlinkDoesNotFailTheWalk
  - id: CLM-005
    requirement: REQ-001
    text: One authored tree hashes IDENTICALLY with and without a root .git directory present, closing the measured 639f74fb/bb86715c divergence
    tests:
      - TestComputeContentHash_IdenticalWithAndWithoutRootGitMetadata
  - id: CLM-006
    requirement: REQ-001
    text: Authored dotfiles and dot-directories at the root — .gitignore, .gitattributes, .github/ — remain content and still change the hash
    tests:
      - TestComputeContentHash_AuthoredDotfilesRemainContent
  - id: CLM-007
    requirement: REQ-001
    text: Root entries whose names merely BEGIN with .git — .gitignore, .gitattributes, .github, .git-hooks — are not treated as metadata, proving exact-name rather than prefix matching
    tests:
      - TestComputeContentHash_RootGitPrefixedNamesAreNotMetadata
  - id: CLM-008
    requirement: REQ-001
    text: A .git directory BELOW the root remains authored content and still contributes, matching the cloner's one-path strip
    tests:
      - TestComputeContentHash_NestedGitDirectoryRemainsContent
  - id: CLM-009
    requirement: REQ-001
    text: A directory holding nothing but root repository metadata hashes as an empty directory
    tests:
      - TestComputeContentHash_DirectoryHoldingOnlyRootGitHashesAsEmpty
  - id: CLM-010
    requirement: REQ-001
    text: A tree with no root metadata hashes to exactly the documented sorted relative-path:file-hash manifest digest, so metadata-free packs see no lock churn
    tests:
      - TestComputeContentHash_MetadataFreeTreeMatchesDocumentedManifest

  # REQ-002 — the copy boundary and the local/remote convergence it buys.
  - id: CLM-011
    requirement: REQ-002
    text: A local-path add of a source directory carrying a root .git installs a tree with no .git in it
    tests:
      - TestAddCommand_Run_LocalSourceRootGitNotInstalled
  - id: CLM-012
    requirement: REQ-002
    text: A local-path add of a .git-carrying source records the same content hash as an add of the identical content with no repository present
    tests:
      - TestAddCommand_Run_LocalSourceHashMatchesMetadataFreeCopy
  - id: CLM-013
    requirement: REQ-002
    text: A local-path add still installs a nested non-root .git, because the exclusion is a root property and that tree is authored content
    tests:
      - TestAddCommand_Run_LocalSourceNestedGitStillInstalled
  - id: CLM-014
    requirement: REQ-002
    text: Repository churn in a local pack's source directory after the add does not break install's hash verification of that pack
    tests:
      - TestInstallCommand_Run_LocalSourceGitChurnStillVerifies
  - id: CLM-015
    requirement: REQ-002
    text: Relock of an installed local pack reproduces the hash the add recorded rather than reporting a change
    tests:
      - TestRelock_LocalPackReproducesAddTimeHash
  - id: CLM-016
    requirement: REQ-002
    subject: cmd/backstop
    text: A local-path add and a remote add of byte-identical authored content record the SAME content hash
    tests:
      - TestE2E_PackAdd_LocalAndRemoteHashesConverge
  - id: CLM-017
    requirement: REQ-002
    subject: cmd/backstop
    text: The remote add then fresh-clone install round trip still hashes equal, the baseline the local half converges on
    tests:
      - TestE2E_PackAddThenInstall_RoundTripHashesMatch

contracts:
  - file: pkg/pack/distribution/hash.go
    provides:
      - name: ComputeContentHash
        kind: function
        signature: "func ComputeContentHash(dir string) (string, error)"
        notes: "Signature UNCHANGED — REQ-001 changes what the walk folds in, not how it is called, so all seven existing call sites (command.go:272 add's post-copy hash, command.go:538 install's source verification, command.go:992 replaceInstalledPack, install.go:78 local materialization, list.go:143, relock.go:54, verify.go:64) inherit the authored-content boundary without edits. The walk skips the root metadata entry and returns filepath.SkipDir for the directory shape so the subtree is never descended, and the symlink shapes are skipped on the lstat info the walk already carries rather than by opening the path."
      - name: ComputeFileHash
        kind: function
        signature: "func ComputeFileHash(path string) (string, error)"
        notes: "Unchanged, listed to keep this file's declared surface complete. It is never reached for a root metadata entry, which is what keeps a dangling .git symlink from failing the walk (CLM-004)."
      - name: isRootRepositoryMetadata
        kind: function
        signature: "func isRootRepositoryMetadata(rel string) bool"
        notes: "The ONE predicate both boundaries consume (REQ-001/REQ-002), so the hash and the copy cannot drift into disagreeing about what a pack is. It takes the slash-normalized path RELATIVE to the walk's own root and returns true only for exact equality with the package's EXISTING gitDirectoryName constant (gitcloner.go:29) — never a prefix, never a suffix, never a nested match, and never a second string literal. It is unexported because it is a property of this package's two walks, not a distribution API; it is declared here because it is the load-bearing anti-drift mechanism and a plan must not quietly implement two copies of it."
    consumes:
      - source: pkg/pack/distribution
        name: gitDirectoryName
        kind: constant
      - source: path/filepath
        name: Walk
        kind: function
      - source: path/filepath
        name: SkipDir
        kind: variable
  - file: pkg/pack/distribution/add.go
    provides:
      - name: copyDirRecursive
        kind: function
        signature: "func copyDirRecursive(src, dst string) error"
        notes: "Signature UNCHANGED (REQ-002). It consumes isRootRepositoryMetadata against the path relative to src, so the exclusion follows the COPY's root. There are SEVEN call sites. FIVE copy a pack root: RunValidationOnScratchCopy (command.go:48), add's install copy (command.go:267), install's git-pack copy (command.go:554), replaceInstalledPack for update and upgrade (command.go:988), and local materialization (install.go:90). Of those five, THREE change behavior in practice — :48, :267, and install.go:90 — because :554 and :988 are fed by an ExecGitCloner clone whose root .git was already stripped, so they have nothing left to exclude. ONE caveat on :554: its opts.CachePath branch (command.go:509-513) reads a pre-populated cache directory no cloner ever touched, so a cache entry carrying a root .git WOULD change behavior there, which is a reason to keep the exclusion at the copy rather than trusting the clone. The remaining TWO are not pack-root copies at all — install's rollback snapshot (command.go:466, packsDir to snapshotDir) and its restore (command.go:477, back again) — where a pack's .git sits at <name>/.git, not at the walk root, so a rollback restores exactly what it snapshotted. That scoping is deliberate; a rollback that silently dropped files would be worse than one that copies too much."
    consumes:
      - source: pkg/pack/distribution
        name: isRootRepositoryMetadata
        kind: function
---

# SPEC-057: Pack Distribution Content Identity

## Overview

BUNDLE-006 revised REQ-021 to 1.1.0 — a pack's content identity covers authored
content, not repository metadata — and instructed that the revision land in a new
delta spec rather than by rewriting SPEC-015's frozen 1.0.0 pin. This is that
spec, and it pins that one requirement.

Half of it already holds. SPEC-055's cloner strips the root `.git` from every
clone, so remote installs are reproducible today and SPEC-055's own claim notes
disclaim the rest. The rest is ISSUE-088: a local-path add hashes and installs
whatever sits in the operator's directory, repository metadata included, so the
same pack content yields different hashes on different machines. That was
measured, not predicted — `639f74fb…` without a root `.git`, `bb86715c…` with
one — and this repository's own `backstop.lock` carries three entries recorded
that way.

What lands is small and mechanical: one predicate, two walks. The point is not
that local installs stop hashing `.git`; it is that local and remote installs end
up computing the SAME function of the same authored bytes, which is what a lock
file has to mean before a fleet can trust it.

The bundle's other 1.1.0 content identity requirement, `REQ-020@1.1.0`, is NOT
pinned here. SPEC-056 delivered and pins it, and `replaced-by` accepts a list, so
SPEC-015 retires pointing at both successors rather than at one spec that
re-states the other's work.

## Requirements

Requirements, claims, and bundle pins are defined in frontmatter. In summary:

| Requirement | Bundle pin | Boundary |
| --- | --- | --- |
| REQ-001 — the hash excludes root repository metadata in every shape it takes | `REQ-021@1.1.0` | `ComputeContentHash` |
| REQ-002 — the copy excludes it too; local and remote hashes converge | `REQ-021@1.1.0` | `copyDirRecursive` |

## Implementation

The work is confined to two walks in `pkg/pack/distribution` plus their tests.

1. **The predicate.** Add `isRootRepositoryMetadata(rel string) bool` to
   `hash.go`. It takes a forward-slash-normalized path RELATIVE to the walk's
   own root and returns true only for exact equality with the package's existing
   `gitDirectoryName` constant (`gitcloner.go:29`) — it must CONSUME that
   constant, not introduce a second `".git"` literal. It has no knowledge of
   shape — directory, file, and symlink are all the same name — which is what
   makes the three shapes one rule rather than three branches.

2. **The hash boundary (`ComputeContentHash`, `hash.go:17`).** Inside the
   existing `filepath.Walk` callback, after computing `rel` and normalizing it
   to slashes, test the predicate BEFORE the `info.IsDir()` early return:
   return `filepath.SkipDir` when the entry is a directory so the subtree is
   never descended, and return `nil` otherwise so a `.git` file or symlink is
   skipped without `ComputeFileHash` ever opening it. Walk errors keep their
   explicit handling and path-naming wrap.

3. **The copy boundary (`copyDirRecursive`, `add.go:222`).** Same shape: derive
   `rel` (it already does), normalize, test the predicate, `SkipDir` for a
   directory and `nil` otherwise.

4. **Nothing else changes.** All seven `ComputeContentHash` call sites —
   `command.go:272` (add's hash of the installed copy), `command.go:538`
   (install's source verification), `command.go:992` (`replaceInstalledPack`),
   `install.go:78` (local materialization), `list.go:143`, `relock.go:54`, and
   `verify.go:64` — inherit the boundary through the function they already call,
   as does every `copyDirRecursive` call site. No signature changes, no new
   exported symbols, no call-site edits. The remote path is unaffected in
   behavior — the cloner already stripped the root `.git` before the copy or
   hash ever saw it — but it now holds that property twice, which is the intent:
   the guarantee stops depending on the cloner alone.

5. **The scratch-validation copy seam, resolved in advance.** Of the seven
   `copyDirRecursive` call sites enumerated in the `add.go` contract note, three
   change behavior in practice, and this is the one whose consequence is not
   obvious. `RunValidationOnScratchCopy` (`command.go:48`) copies `packDir` into a
   scratch tree with the PACK ROOT as the walk root, so after this change `pack
   check` and `pack test` validate a `.git`-free tree. This was checked rather
   than assumed and the conclusion is BENIGN: `pkg/packval` contains no `.git`
   reference in any non-test file, so no phase reads, requires, or writes
   repository metadata, and the scratch-copy indirection SPEC-056 introduced
   (REQ-008) means nothing packval does can reach the tree that is hashed
   anyway. The planner does not need to re-derive this; it needs only to avoid
   "fixing" it back.

6. **Out of scope, by name.** The typed migration-required diagnostic and the
   explicit remote re-validating migration operation (`REQ-041@1.1.0`, DD-28,
   OQ-6); `pack relock`'s path-versus-name argument shape (ISSUE-074 residual);
   the staged-filesystem transaction coordinator (`REQ-040@1.1.0`); and the
   hermetic lifecycle parity suite (`REQ-042@1.1.0`).

## Verification

Verification configuration is in frontmatter: integration level, an 80% coverage
threshold, and a test command covering `pkg/pack/distribution` and
`cmd/backstop`. Both packages are in scope because the convergence claim can
only be made where a real local add and a real remote add meet — the hermetic
remote harness in `cmd/backstop`.

## Sharp Edges

- **This breaks `pack install` in THIS repository, and the implementing plan
  owes the relock.** `backstop.lock` here holds three local entries whose
  recorded hash was computed over a `.git`-carrying installed tree, verified by
  recomputing each one both ways against the working tree on 2026-07-27:
  `backstop/self` (`local_path: ../backstop-self-pack`, locked
  `3eb4324c…`), `backstop/go-standards` (locked `a3fdd4a2…`), and
  `backstop/go-toolchain` (locked `be64300e…`). All three installed copies carry
  a root `.git` today; all three currently match their lock and all three stop
  matching the moment this spec lands. The other three entries —
  `backstop/contracts`, `backstop/substantiveness`, and the git-source
  `backstop-ai/cobra-cli-standards` — carry no root `.git` and are unaffected,
  hashing identically before and after. Relocking those three entries is a
  STATED OBLIGATION of the plan that implements this spec, not a follow-up:
  without it, `pack install` fails in-repo on `backstop/self` with a hash
  mismatch. Note that a review pass initially identified only `backstop/self`;
  the count is three, and the plan should re-measure rather than trust either
  number.

- **The gate will NOT warn first.** `pack_lock_verification` calls
  `distribution.VerifyLock` (`cmd/backstop/pack_gate.go:193`), which skips every
  entry with `source_type: local` (`verify.go:47`). So the contaminated-hash
  class this spec closes has never been visible at the gate at all: it surfaces
  at `pack install`'s materialize-time comparison and in `pack list`'s
  stale/locked column. Anyone reading "the gate would have caught it" into this
  spec is reading something that is not there — and anyone expecting the gate to
  go red as a reminder to relock will not get one.

- **Beyond this repository, the same landing turns legacy local locks into what
  looks like tamper.** Any lock entry anywhere whose hash was recorded from a
  `.git`-carrying local source now mismatches with the existing
  `hash mismatch for local pack …` diagnostic. The bundle's designated remedy is
  `pack relock` (DD-28), whose own argument shape is still broken (ISSUE-074
  residual: it takes a path where its siblings take a name). This spec
  deliberately does not add the typed migration diagnostic OQ-6 resolved on —
  that is `REQ-041@1.1.0` — so between this spec landing and that seed landing,
  a legacy local hash reads like tamper. Whoever sequences these should know
  that ordering cost before choosing it; Review Question 1 asks it directly.

- **Root-relative is a real constraint, not an implementation detail.**
  `copyDirRecursive` is also used for install's packs-directory rollback
  snapshot (`command.go:466`), whose walk root is `.backstop/packs/`, not a pack
  root. A pack's `.git` there sits at `<name>/.git` and is therefore NOT
  excluded, so a rollback restores exactly what it snapshotted. If someone
  "fixes" the predicate to match `.git` at any depth, the copy boundary and the
  cloner's one-path strip stop agreeing and the convergence this spec exists to
  buy is silently lost.

- **Nested `.git` stays content, which is a deliberate residual.** DD-24 scopes
  the exclusion to the pack ROOT and the cloner strips exactly one path, so a
  local source carrying a nested `.git` still hashes it while a fresh clone of
  the same repository generally cannot contain one at all (git does not track a
  nested repository as a directory). That is a narrow remaining asymmetry. It is
  left standing rather than widened, because widening it means inventing
  semantics the bundle did not decide; Review Question 3 escalates it.

- **`.git` is not the only repository-control metadata that exists.** `.hg`,
  `.svn`, and `.bzr` are the same category, and DD-24 names only `.git`. The
  predicate implements exactly what was decided. If a pack is ever authored in
  a non-git VCS, this boundary does not cover it, and the fix is a bundle
  decision rather than a quiet addition to a regex.

- **`REQ-020@1.1.0` belongs to SPEC-056 and must not be restated here.** An
  earlier draft of this spec re-pinned it and re-cited SPEC-056's own mandated
  test names, on the reasoning that SPEC-015's retirement needed both 1.1.0
  successors in one place. That reasoning was wrong: `replaced-by` accepts a
  list (`pkg/validate/terminal.go:37-38`). Anyone tempted to re-add it for
  completeness should check the mechanism first — the cost was twenty
  double-mandated test names across two live specs, bought for nothing. ONE
  shared mandate is left standing on purpose: CLM-017 re-cites SPEC-055's
  `TestE2E_PackAddThenInstall_RoundTripHashesMatch` because REQ-002's
  convergence clause is meaningless without naming the remote baseline the
  local half has to meet. That is a single test asserted from two angles, not a
  requirement owned twice.

## Review Questions

1. Does the implementation leave a local pack whose lock carries a legacy
   metadata-inclusive hash with any diagnostic beyond the generic
   `hash mismatch for local pack …`? It must not invent a migration message —
   that is `REQ-041@1.1.0` — but if it does add one, that is scope creep and
   the reviewer should say so.

2. Does `grep -n '\.git"' pkg/pack/distribution/*.go` (excluding tests) still
   return exactly TWO hits — `gitDirectoryName` at `gitcloner.go:29` and the
   benign `https://github.com/…​.git` URL suffix in `resolveGitURL`
   (`add.go:109`)? A third hit means the predicate introduced its own literal
   instead of consuming the constant, which is the drift this spec's one-rule
   design exists to prevent.

3. Does any test or implementation branch treat a `.git` BELOW the walk root as
   metadata? It must not. If the implementer believes it should, that is a
   bundle-level question about DD-24's scope, not a judgment call to make in
   code.

4. Do the REQ-001 tests construct their fixtures as real on-disk shapes — an
   actual directory, an actual pointer file, an actual symlink including a
   dangling one — rather than asserting through a stub or a table of names? The
   dangling-symlink claim in particular is only meaningful against a real
   dangling link, because the defect it guards is `os.Open` following it.

5. Does CLM-010's expected digest come from independently recomputing the
   documented manifest inside the test, rather than from pasting whatever the
   implementation printed? A digest captured from the code under test asserts
   nothing.

6. Does the local/remote convergence test (CLM-016) compare hashes of
   genuinely identical authored bytes — same file set, same contents — such
   that a failure means the algorithms diverged rather than that the two
   fixtures were never the same pack?

7. Were `backstop/self`, `backstop/go-standards`, and `backstop/go-toolchain`
   relocked as part of the implementation, and does `pack install` succeed
   in-repo afterwards? The gate stays green either way, so this cannot be
   inferred from a passing gate.

## References

- `bundles/BUNDLE-006-pack-distribution-lifecycle.bundle.md` — source bundle.
  `REQ-021@1.1.0` requirement text (~line 309-331); DD-20 and DD-24; OQ-6
  resolution (~line 1260); the measured hash pair and "REQ-021@1.1.0 and DD-24
  remain specified-and-unbuilt" (~line 740, 771-773); the authored content
  identity seed (~line 1393).
- `bundles/BUNDLE-006-pack-distribution-lifecycle.bundle.md` (~line 1435),
  "Revision Impact on Existing Artifacts" — `SPEC-015-pack-distribution.spec.md`
  remains historically pinned to `REQ-020@1.0.0`/`REQ-021@1.0.0` and that pin
  MUST NOT be rewritten; it describes the algorithm SPEC-015 evaluated. SPEC-015
  is to be retired `replaced` with `replaced-by: [SPEC-056, SPEC-057]` — the
  field accepts a list (`pkg/validate/terminal.go:37-38`) — SPEC-056 succeeding
  its `REQ-020@1.0.0` pin and SPEC-057 its `REQ-021@1.0.0` pin. That retirement
  happens ONCE THIS SPEC IS IMPLEMENTED, not before, and not by this spec's
  authoring. SPEC-015's two ancestor-removed mandated test names,
  `TestPackAdd_AlreadyInstalledExitsNonZero` and
  `TestPackAdd_LocalPathNotClonedToPacksDir`, retire with it.
- `issues/ISSUE-088-local-pack-git-residual-content-hash.issue.md` — the
  local-path residual this spec closes. Its call-site list cites pre-SPEC-056
  line numbers for `command.go`; the current three are `command.go:272`,
  `command.go:538`, and `command.go:992`.
- `specs/SPEC-055-production-remote-dependency-assembly.spec.md` — the
  post-clone root `.git` strip (CLM-101/CLM-102) that made the remote path
  authored-content-only, and its explicit disclaimer of `REQ-021@1.1.0`.
- `specs/SPEC-056-remote-identity-version-validation.spec.md` — owns and pins
  `REQ-020@1.1.0` (its REQ-003/REQ-004/REQ-005) and the scratch-copy validation
  ordering (its REQ-008). This spec neither re-pins nor contradicts it.
- `issues/ISSUE-074-pack-relock-silent-failure.issue.md` — the migration
  vehicle's residual argument-shape defect, which this spec does not touch.
- `pkg/pack/distribution/hash.go`, `pkg/pack/distribution/add.go` — the two
  walks that change.
- `pkg/pack/distribution/gitcloner.go:29,258` — the `gitDirectoryName` constant
  the predicate must consume, and the one-path strip whose scope it mirrors.
- `cmd/backstop/pack_gate.go:193`, `pkg/pack/distribution/verify.go:47` — why
  the gate stays silent about local-pack hashes.

## Version History

- **1.0.1** (2026-07-27): Status flip to `implemented`. Documentary only — no
  requirement, claim, contract, or behavior changed; the delivered contract is
  the one 1.0.0 states. PLAN-SPEC-057 executed in full, commit series
  `5f14db5, 743e160, 4ccced3`: the shared `isRootRepositoryMetadata` predicate
  consuming the existing `gitDirectoryName` constant, the exclusion applied at
  both the `ComputeContentHash` walk and the `copyDirRecursive` walk, and the
  three contaminated in-repo lock entries (`backstop/self`,
  `backstop/go-standards`, `backstop/go-toolchain`) relocked as the Sharp Edges
  section obliged. Every mandated test passes by name under `-race`.
- **1.0.0** (2026-07-27): Initial spec. Authored against BUNDLE-006's authored
  content identity seed, pinning `REQ-021@1.1.0` only. An earlier draft also
  re-pinned `REQ-020@1.1.0` and re-cited SPEC-056's mandated test names; review
  removed that half, because SPEC-056 already owns the pin and `replaced-by`
  accepts a list, so SPEC-015 retires pointing at both successors rather than
  at one spec restating the other's work.
