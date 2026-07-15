---
title: "Pack Distribution Lifecycle — From Validated Pack to Running Gate"
number: BUNDLE-006
created: "2026-04-14"
schema_version: bundle/v2

bundle:
  name: pack-distribution-lifecycle
  version: "0.6.0"
  created: "2026-04-14"
  updated: "2026-04-08"
  category: feature

status:
  maturity: ready

problem:
  summary: >
    BUNDLE-004 tells an agent how to author a pack. BUNDLE-005 tells it how
    to validate one. Neither covers the lifecycle between "author has a
    validated pack in a git repo" and "consumer's gate enforces that pack's
    rules." There is no story for how packs are added, removed, updated,
    upgraded, locked, or verified at gate time. There is no lockfile format.
    There is no tool_config provenance model (so pack removal cannot revert
    config changes). There is no CI-optimized install path. Without this
    bundle, every consumer must manually clone packs, hand-edit backstop.yml,
    hand-merge tool configs, and hope their CI reproduces what they tested
    locally.

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

solution:
  approach: >
    Git-native distribution with backstop.lock for reproducibility. `pack
    add` is the consumer entry point — it clones, validates, installs,
    merges config, and updates the lockfile in one command. `pack install`
    is the CI fast path — it restores from the lockfile with hash
    verification, skipping validation (already proven at add time). `pack
    remove` uses tool_config provenance tracking to cleanly revert config
    changes. `pack update` handles minor/patch bumps with tamper detection.
    `pack upgrade` handles major bumps with remediation bundle generation.
    Lock verification at gate time prevents drift between what was validated
    and what's running. Local path packs use the same loader and validation
    but skip git resolution.

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
    version: "1.0.0"
    text: >
      backstop.lock must be YAML with sorted keys, containing per-pack entries
      with name, version, git ref (null for local packs), content hash, source
      type (git or local), and install date (DD-19)
  - id: REQ-021
    version: "1.0.0"
    text: >
      Content hash must be SHA-256 computed from a sorted manifest of
      relative-path:SHA-256-file-hash pairs covering every file in the pack
      directory (DD-20)
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
---

# Pack Distribution Lifecycle

## Current Thinking

### The distribution gap

BUNDLE-004 (authoring) and BUNDLE-005 (validation) give you a pack that is
structurally correct, fixture-proven, and mechanically verified. But a
validated pack sitting in a git repo is useless until a consumer can install
it. Today, there is no installation path. No lockfile. No way to reproduce
pack state across machines. No way to remove a pack without leaving config
debris. This bundle fills the gap between "pack exists and is valid" and
"gate enforces it."

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
for in-repo packs that evolve with the codebase.

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

## Draft Requirements

Requirements REQ-001 through REQ-037 are defined in the frontmatter
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
- **backstop.lock** (REQ-020, REQ-021): YAML sorted keys, SHA-256 sorted
  manifest hash.
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

## References

- BUNDLE-001 — parent vision bundle (pack distribution, verification, review)
- BUNDLE-004 — pack authoring (manifest schema, content types, archetypes)
- BUNDLE-005 — pack validation (pack check + pack test pipeline)
- BUNDLE-003 — onboarding (`backstop init` wires default packs via pack add)
- ADR-0001 — machine-first output (JSON for agent consumption)

## Version History

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
