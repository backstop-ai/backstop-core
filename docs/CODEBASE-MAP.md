# backstop-core — codebase map

Navigational context, auto-loaded via CLAUDE.md. Generated 2026-07-25 by a
full-repo sweep; regenerate (ask a session to re-explore and rewrite this
file) after structural changes. Invariants/behavior live in CLAUDE.md — this
file is only WHERE things are.

> README correction: library code is NOT in `internal/` (empty dir) — it all
> lives under `pkg/`.

## Entry points

- `cmd/backstop/main.go:8` — main; maps `*ExitCodeError` → exit codes
  (0 pass / 1 violations / 2 config).
- `cmd/backstop/root.go:20` `NewRootCommand()`; all subcommands wired at
  `root.go:144`. gate → `gate.go:33`; pack subcommands in `pack_*.go`;
  artifact in `artifact_*.go`; baseline `baseline.go:24`; waiver
  `waiver.go:26`; `commands` (agent-discovery JSON) inline `root.go:126`.

## Package map

- `cmd/backstop/` — CLI layer + gate step wiring (`gate.go`, `pack_gate.go`,
  `pack_gate_provision.go`). Test-heavy (~90 files, mostly tests).
- `pkg/gate/` — CORE: gate kill-chain. `Gate` orchestrator (`gate.go`),
  `StepFunc`/`StepResult`/`GateResult` (`result.go:135,169`), step impls
  (`step_*.go`), scope, baseline compare, policy, traceability/drift.
- `pkg/pack/` — CORE: manifest model + `engines:` parsing (`manifest.go`),
  coordinates.
  - `pkg/pack/distribution/` — pack lifecycle: `add/install/update/upgrade/
    remove/relock.go`, `identity.go` (SPEC-056: the identity gate, the three
    typed refusals, and `CoordinateForEntry`), `lockfile.go`, `hash.go`,
    `provenance.go`, `config_merge.go`, `tamper.go`.
  - `pkg/pack/engine/` — engine bindings (`binding.go`), `GateType`
    (`gatetype.go`), pinned tool versions (`allowlist.go` — semgrep 1.96.0).
- `pkg/check/` — findings execution + SARIF (`runner.go`, `parsers.go:42`
  `parseSarif`), coverage channel (`coverage.go`).
- `pkg/waiver/` — `@waiver:` grammar (`waiver.go`), adjudication
  (`adjudicate.go:86` — byte-scans finding line + line above; zero-baked, no
  comment lexer), non-waivable policy (`policy.go`).
- `pkg/packval/` — `pack check`/`pack test` pipeline (phases 1–6: structural,
  coherence, fixtures, archetype, layer, risk-class).
- `pkg/validate/` — per-artifact-type semantic validators (spec/plan/issue/
  adr/bundle/directive/contracts + traceability refs).
- Support: `pkg/config/` (backstop.yml loader), `pkg/schema/` (artifact
  JSON-schema load/resolve), `pkg/artifact/` (frontmatter parse),
  `pkg/scaffold/` (ID/slug alloc, git tag ops), `pkg/baseengines/`
  (embedded base-engines pack).

## Gate flow (`backstop gate --all`)

`cmd/backstop/gate.go:57 runGate` → config + scope (`--all` →
`GateScopeModeAll`, `gate.go:85`) → `buildGateSteps` (`gate.go:617`; loads
installed packs, derives SourceClassifier/TestNameMatcher) → ordered steps
(`gate.go:729-747`, reorder `gate.go:846`):
`pack_lock_verification → artifact_validation → pack_engines →
test_verification → substantiveness → coverage → contracts → status_drift →
requirement_traceability → waiver_resolution → baseline_comparison →
ledger_integrity` (+ toolchain warn step when none declared, `gate.go:605`).
`Gate.Run` (`pkg/gate/gate.go:132`) executes, swaps waiver result
(`gate.go:149`) and baseline (`gate.go:155`), applies policy (`gate.go:168`).
pack_engines: `packValidatorStep` (`gate.go:788`) → `provisionEngines` →
`dispatchPackEngines` (`pack_gate.go:267`) → `runFindingsEngine` → SARIF.

## Pack lifecycle (`pack add <ref>`)

`cmd/backstop/pack_add.go` → `buildAddCommand` (`pack_wiring.go` — assembles
`ExecGitCloner`/`PackvalValidator` via fail-closed positional constructors,
SPEC-055) → `AddCommand.Run` (`add.go`). Free `distribution.Add/Install/
Update/Upgrade` are DELETED; commands exist only via `New*Command`
constructors that error naming any nil dependency (`command.go`).
THE MANIFEST NAME IS THE INSTALL IDENTITY (SPEC-056). The requested
`org/repository` is a SOURCE COORDINATE and nothing else: the install path,
the backstop.yml key, the lock key and therefore the engine asset root all
read the name the cloned `pack.yml` declares. When the two differ the add
SUCCEEDS and reports it loudly — divergence is a diagnostic, never a refusal.

LOCAL path (`isLocalPath`): `resolveLocalPackSource` reads identity through
`ReadManifestIdentity` (`identity.go`) — the SAME reader the remote path uses,
so there is one implementation of "what is this pack called" → `source_type:
"local"`, project-relative `local_path` in lock, and NO `source_coordinate`.

REMOTE `org/pack@version`: `ResolveEffectiveVersion` (`identity.go`) resolves
EXACTLY ONE version BEFORE any git subprocess runs (`--version` beats the `@`
suffix; strict `X.Y.Z` after at most one leading `v`) → `resolveGitURL` →
`ExecGitCloner.Clone` (`gitcloner.go`) at that tag — real shallow tag clone,
root `.git` STRIPPED so remote hashes are reproducible → THE IDENTITY GATE,
`ValidateRemoteIdentity` (`identity.go`), which reads the cloned `pack.yml`
and refuses BEFORE any consumer state is touched when the manifest version
does not equal the tag, when the manifest is absent or unparseable, or when
its name is unusable (decided by `pack.ValidatePackName` — one authority, not
a copy). Typed refusals `*VersionUnresolvedError`, `*VersionMismatchError`
and `*IdentityError` classify under `--json` as kinds `version` and `identity`
(`cmd/backstop/json_error.go`). Validation then runs UNCONDITIONALLY through
the same `pkg/packval` pipeline `pack check` uses (`validator.go`).

Common tail: validate a SCRATCH COPY (`RunValidationOnScratchCopy`,
`command.go`) → copy the PRISTINE materialized tree to
`.backstop/packs/<manifest-name>/` → `ComputeContentHash` (`hash.go:17`) →
config merge/provenance → `updateBackstopYml` → lockfile (recording
`source_coordinate` VERBATIM for git sources) → ensureGitignore.

THE SCRATCH COPY IS LOAD-BEARING, NOT HYGIENE. `pkg/packval` phase 3 renders
every `tier: complete` scaffold's `sample_config` into the directory it
validates, so validating in place and then hashing that same tree recorded a
hash no fresh clone could reproduce — an install failure that looks exactly
like tampering. Validating a copy is what makes the hash `pack add` records
equal the hash `pack install` computes.

Update/upgrade resolve versions via `TagVersionResolver` (`versionresolver.go`,
strict semver over remote tags) at the RECORDED coordinate, run the same
identity gate immediately after their clone, and PRESERVE `source_coordinate`
across the rewrite; upgrade refuses a local-source pack outright (pointing at
`pack relock`) and scans BEFORE any consumer mutation.

### Known gap — `pack relock` arg-shape asymmetry (ISSUE-074, residual)

The SILENT half of ISSUE-074 is FIXED (SPEC-055 REQ-011: relock failures
print their diagnostic to stderr). The residual: `relock` still takes a
filesystem PATH arg (`Use: "relock [path]"`, `pack_relock.go:13`) while
remove/update/upgrade take a pack NAME — guessing wrong now errors loudly
instead of silently, but the asymmetry itself is unresolved (identity/
migration seed territory, where relock gains its migration role).

Relock is NOT on SPEC-056's edit list for a reason worth recording: it does a
READ-MODIFY-WRITE on the lock entry (`relock.go:59-61` — set ContentHash and
InstallDate, write the same struct back), so it preserves `source_coordinate`
and every other field it does not refresh BY CONSTRUCTION. That is CLM-050,
and it contrasts with `recordGitPackInLock`, which REPLACES the whole entry
and therefore had to be given the coordinate as an explicit parameter.

### Resolved 2026-07-26 — remote pack resolution (was ISSUE-073)

SPEC-055 delivered the production remote path: `ExecGitCloner`
(`gitcloner.go`, real Clone + remote ListTags, root `.git` stripped for
reproducible hashes), unconditional validation (`validator.go`), strict-
semver resolution (`versionresolver.go`), and fail-closed constructors
(`command.go`) that make missing dependencies unconstructable — the old
nil-deref panic class is unrepresentable. Wiring in `pack_wiring.go`;
hermetic remote E2E via `GIT_CONFIG_GLOBAL` insteadOf redirect
(`hermetic_remote_harness_test.go` + `testdata/hermetic-remote/`), incl.
an add→fresh-clone-install round trip asserting hash EQUALITY. Remote
installs are no longer test-mock-only.

## Engines

pack.yml `engines:` → `Manifest.Engines` (`manifest.go:29,68`) →
`EngineBinding` (`manifest.go:296`). Capability presence keyed on declared
`gate_type` (`gate.go:369`), never pack name. Registry = pack bindings over
embedded base-engines defaults (`pkg/baseengines`, root `embed.go`).
InputModes: rule-flags (semgrep `--config` per rule file), config-file,
pattern-arg, none (sandbox validator). Provisioning: backstop-introduced
tools (semgrep/ast-grep) auto-provision at pinned versions
(`allowlist.go:22`); Layer-0 tools (go, golangci-lint) must be on PATH else
exit-2. Coverage is a separate channel: `dispatchPackCoverage`
(`pack_gate.go:339`) → `ParsePackCoverage`, routed on `GateType==coverage`.

## Config & schemas

backstop.yml: schema `artifacts/backstop-yml/v1/schema.json`, loader
`pkg/config/config.go`. backstop.lock: `distribution/lockfile.go`
(`LockEntry{name,version,git_ref,content_hash,source_type,install_date,
source_coordinate,
local_path}`). Artifact schemas embedded via root `embed.go` (`SchemaFS`,
`//go:embed all:artifacts`); on disk `artifacts/<type>/v<N>/schema.json`.

## Tests

Colocated `*_test.go`; e2e fixture projects under `cmd/backstop/testdata/`
(each with own `.backstop/packs/`), `pkg/gate/testdata/` (waiver-e2e,
substantiveness, traceability, contracts), `pkg/pack/distribution/testdata/`
(tamper fixtures), `pkg/packval/testdata/`. Run: `make test` (race),
`make lint`, `make coverage` (90% threshold), `make ci`.

## Repo-root oddities

- `embed.go` (package backstopcore): go:embed can't reach up, so schemas +
  base-engines pack embed from root (`SchemaFS`, `BaseEnginesFS`).
- `capabilities/`, `directives/` (DIR-001..025), `recipes/`, `standards/`,
  plus `specs/plans/issues/bundles/adrs/` — the repo governing itself with
  its own artifact vocabulary; backstop-core is its own test corpus.
- `packs/` at root: source of the three locally-installed packs the lock
  references; `./backstop` prebuilt binary + `bin/`.
